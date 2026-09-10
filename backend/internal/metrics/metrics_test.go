package metrics

import (
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestNewRegistryIsEmpty(t *testing.T) {
	r := NewRegistry()
	out := r.Render()
	if !strings.Contains(out, "http_requests_in_flight 0") {
		t.Error("empty registry should expose the in-flight gauge at zero")
	}
	for _, metric := range []string{
		"http_requests_total{",
		"http_request_duration_seconds_count{",
		"sync_runs_total{",
	} {
		if strings.Contains(out, metric) {
			t.Errorf("empty registry emitted unrecorded series %q", metric)
		}
	}
}

func TestRecordRequestAggregatesAndRenders(t *testing.T) {
	r := NewRegistry()
	r.RecordRequest("/api/v1/metrics/overview", "GET", 200, 0.250)
	r.RecordRequest("/api/v1/metrics/overview", "GET", 200, 0.125)
	r.RecordRequest("/api/v1/metrics/overview", "GET", 500, 0.500)

	r.RequestStarted()
	r.RequestStarted()
	r.RequestFinished()
	if got := r.InFlightRequests(); got != 1 {
		t.Fatalf("InFlightRequests: want 1, got %d", got)
	}

	out := r.Render()
	wantLines := []string{
		`http_requests_in_flight 1`,
		`http_requests_total{code="200",method="GET",route="/api/v1/metrics/overview"} 2`,
		`http_requests_total{code="500",method="GET",route="/api/v1/metrics/overview"} 1`,
		`http_request_duration_seconds_count{route="/api/v1/metrics/overview"} 3`,
		`http_request_duration_seconds_sum{route="/api/v1/metrics/overview"} 0.875000`,
	}
	for _, want := range wantLines {
		if !strings.Contains(out, want) {
			t.Errorf("Render() missing %q in:\n%s", want, out)
		}
	}
}

func TestRecordSyncRunEmitsSortedSeries(t *testing.T) {
	r := NewRegistry()
	r.RecordSyncRun("tiktok", "FAILED")
	r.RecordSyncRun("meta", "SUCCESS")
	r.RecordSyncRun("meta", "SUCCESS")
	r.RecordSyncRun("tiktok", "SUCCESS")

	out := r.Render()
	// Sorted by label: meta before tiktok, status alphabetical within.
	positions := map[string]int{
		`sync_runs_total{channel="meta",status="SUCCESS"} 2`:   strings.Index(out, `sync_runs_total{channel="meta",status="SUCCESS"}`),
		`sync_runs_total{channel="tiktok",status="FAILED"} 1`:  strings.Index(out, `sync_runs_total{channel="tiktok",status="FAILED"}`),
		`sync_runs_total{channel="tiktok",status="SUCCESS"} 1`: strings.Index(out, `sync_runs_total{channel="tiktok",status="SUCCESS"}`),
	}
	for line, pos := range positions {
		if pos < 0 {
			t.Errorf("Render() missing %q in:\n%s", line, out)
		}
	}
	if positions[`sync_runs_total{channel="meta",status="SUCCESS"} 2`] > positions[`sync_runs_total{channel="tiktok",status="FAILED"} 1`] {
		t.Errorf("sync_runs_total series not sorted alphabetically:\n%s", out)
	}
}

func TestLabelEscapingMatchesPrometheusSpec(t *testing.T) {
	r := NewRegistry()
	r.RecordRequest("/route\"with\\bad\nchars", "GET", 200, 0.001)
	out := r.Render()
	if !strings.Contains(out, `route="/route\"with\\bad\nchars"`) {
		t.Errorf("Render() did not escape label properly:\n%s", out)
	}
}

func TestHandlerServesPrometheusContentType(t *testing.T) {
	r := NewRegistry()
	r.RecordRequest("/api/v1/health", "GET", 200, 0.001)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	Handler(r)(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if got := resp.StatusCode; got != http.StatusOK {
		t.Fatalf("status: want 200, got %d", got)
	}
	if got := resp.Header.Get("Content-Type"); got != "text/plain; version=0.0.4; charset=utf-8" {
		t.Errorf("Content-Type: want Prometheus text, got %q", got)
	}
	if !strings.Contains(string(body), "http_requests_total") {
		t.Errorf("body missing http_requests_total: %q", body)
	}
}

func TestHandlerDefaultsToProcessRegistry(t *testing.T) {
	Default.RecordRequest("/api/v1/health", "GET", 200, 0.001)
	defer func() { Default.httpRequestsTotal = &labeledCounter{values: map[string]*int64{}} }() // isolate

	rec := httptest.NewRecorder()
	Handler(nil)(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()
	if !strings.Contains(body, `route="/api/v1/health"`) {
		t.Errorf("default registry not used: %s", body)
	}
}

func TestConcurrentRecordAndRenderIsSafe(t *testing.T) {
	r := NewRegistry()
	var wg sync.WaitGroup
	const goroutines = 8
	const perGoroutine = 250
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				r.RecordRequest("/api/v1/metrics/overview", "GET", 200, 0.001)
				r.RecordSyncRun("meta", "SUCCESS")
				if j%50 == 0 {
					r.RequestStarted()
					r.RequestFinished()
				}
				_ = r.InFlightRequests()
				_ = r.Render()
			}
		}()
	}
	wg.Wait()
	runtime.GC()

	out := r.Render()
	want := `http_requests_total{code="200",method="GET",route="/api/v1/metrics/overview"} 2000`
	if !strings.Contains(out, want) {
		t.Errorf("expected %q in render output, got:\n%s", want, out)
	}
	wantSum := `http_request_duration_seconds_count{route="/api/v1/metrics/overview"} 2000`
	if !strings.Contains(out, wantSum) {
		t.Errorf("expected %q in render output, got:\n%s", wantSum, out)
	}
}
