// Package metrics provides a dependency-free Prometheus-compatible registry
// for the small set of operational counters and duration summaries this
// service needs (#39). It speaks the standard text exposition format, so any
// Prometheus scraper can consume /metrics as-is.
package metrics

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

type labeledCounter struct {
	mu     sync.Mutex
	values map[string]*int64
}

func (c *labeledCounter) inc(labelValue string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	v, ok := c.values[labelValue]
	if !ok {
		v = new(int64)
		c.values[labelValue] = v
	}
	atomic.AddInt64(v, 1)
}

func (c *labeledCounter) snapshot() map[string]int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(map[string]int64, len(c.values))
	for k, v := range c.values {
		out[k] = atomic.LoadInt64(v)
	}
	return out
}

// labeledSummary accumulates a count and a total for duration-like series,
// exported as <name>_count and <name>_sum (Prometheus summary convention).
type labeledSummary struct {
	mu     sync.Mutex
	counts map[string]*int64
	totals map[string]*int64
}

func (s *labeledSummary) observe(labelValue string, seconds float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cnt, ok := s.counts[labelValue]
	if !ok {
		cnt = new(int64)
		s.counts[labelValue] = cnt
		s.totals[labelValue] = new(int64)
	}
	// Store milliseconds in the integer total to avoid float atomics.
	atomic.AddInt64(cnt, 1)
	atomic.AddInt64(s.totals[labelValue], int64(seconds*1000))
}

type Registry struct {
	httpRequestsTotal      *labeledCounter // label: code|method|route
	httpRequestDuration    *labeledSummary // label: route
	syncRunsTotal          *labeledCounter // label: channel|status
	httpRequestsInFlight   int64
	activeSyncGaugeChannel string
}

// Default is the process-wide registry used by the HTTP layer and the sync
// service. A package-level default keeps instrumentation wiring trivial in a
// modular monolith of this size.
var Default = NewRegistry()

func NewRegistry() *Registry {
	return &Registry{
		httpRequestsTotal:   &labeledCounter{values: map[string]*int64{}},
		httpRequestDuration: &labeledSummary{counts: map[string]*int64{}, totals: map[string]*int64{}},
		syncRunsTotal:       &labeledCounter{values: map[string]*int64{}},
	}
}

// RecordRequest tracks one completed HTTP request.
func (r *Registry) RecordRequest(route, method string, statusCode int, durationSeconds float64) {
	r.httpRequestsTotal.inc(fmt.Sprintf("%d|%s|%s", statusCode, method, route))
	r.httpRequestDuration.observe(route, durationSeconds)
}

// RequestStarted/RequestFinished maintain the in-flight request gauge.
func (r *Registry) RequestStarted()  { atomic.AddInt64(&r.httpRequestsInFlight, 1) }
func (r *Registry) RequestFinished() { atomic.AddInt64(&r.httpRequestsInFlight, -1) }

// InFlightRequests returns the current number of in-flight HTTP requests.
func (r *Registry) InFlightRequests() int64 { return atomic.LoadInt64(&r.httpRequestsInFlight) }

// RecordSyncRun tracks one channel sync attempt with its outcome
// (SUCCESS / PARTIAL_FAILURE / FAILED).
func (r *Registry) RecordSyncRun(channel, status string) {
	r.syncRunsTotal.inc(channel + "|" + status)
}

// Render produces the Prometheus text exposition format.
func (r *Registry) Render() string {
	var b strings.Builder

	b.WriteString("# HELP http_requests_total Total HTTP requests processed.\n")
	b.WriteString("# TYPE http_requests_total counter\n")
	reqs := r.httpRequestsTotal.snapshot()
	keys := make([]string, 0, len(reqs))
	for k := range reqs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		parts := strings.SplitN(k, "|", 3)
		fmt.Fprintf(&b, "http_requests_total{code=%q,method=%q,route=%q} %d\n",
			parts[0], parts[1], parts[2], reqs[k])
	}

	r.httpRequestDuration.mu.Lock()
	routes := make([]string, 0, len(r.httpRequestDuration.counts))
	for route := range r.httpRequestDuration.counts {
		routes = append(routes, route)
	}
	sort.Strings(routes)
	b.WriteString("# HELP http_request_duration_seconds HTTP request durations.\n")
	b.WriteString("# TYPE http_request_duration_seconds summary\n")
	for _, route := range routes {
		cnt := atomic.LoadInt64(r.httpRequestDuration.counts[route])
		totalMs := atomic.LoadInt64(r.httpRequestDuration.totals[route])
		fmt.Fprintf(&b, "http_request_duration_seconds_count{route=%q} %d\n", route, cnt)
		fmt.Fprintf(&b, "http_request_duration_seconds_sum{route=%q} %f\n", route, float64(totalMs)/1000)
	}
	r.httpRequestDuration.mu.Unlock()

	b.WriteString("# HELP http_requests_in_flight Current in-flight HTTP requests.\n")
	b.WriteString("# TYPE http_requests_in_flight gauge\n")
	fmt.Fprintf(&b, "http_requests_in_flight %d\n", r.InFlightRequests())

	b.WriteString("# HELP sync_runs_total Channel sync runs by outcome.\n")
	b.WriteString("# TYPE sync_runs_total counter\n")
	syncs := r.syncRunsTotal.snapshot()
	keys = keys[:0]
	for k := range syncs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		parts := strings.SplitN(k, "|", 2)
		fmt.Fprintf(&b, "sync_runs_total{channel=%q,status=%q} %d\n",
			parts[0], parts[1], syncs[k])
	}

	return b.String()
}

// Handler serves the registry in Prometheus text format.
func Handler(reg *Registry) http.HandlerFunc {
	if reg == nil {
		reg = Default
	}
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		_, _ = w.Write([]byte(reg.Render()))
	}
}
