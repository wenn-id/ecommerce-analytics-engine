package connector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	maxResponseBodyBytes = 10 << 20 // 10 MiB
	maxRetries           = 3
)

// apiClient is a small shared HTTP helper for the real platform connectors:
// it adds timeouts, a response size cap, JSON (de)serialization and retry
// with exponential backoff for transient failures (429/5xx and transport
// errors), honoring Retry-After when the platform sends it (#62).
type apiClient struct {
	http    *http.Client
	baseURL string
}

func newAPIClient(baseURL string) *apiClient {
	return &apiClient{
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
	}
}

// getJSON performs a GET request against baseURL+path with the given query
// parameters and headers, decoding the response into out.
func (c *apiClient) getJSON(ctx context.Context, path string, query url.Values, header http.Header, out interface{}) error {
	return c.doJSON(ctx, http.MethodGet, path, query, header, nil, out)
}

func (c *apiClient) doJSON(ctx context.Context, method, path string, query url.Values, header http.Header, body []byte, out interface{}) error {
	attempts := maxRetries + 1
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			backoff := c.backoffDuration(attempt, lastErr)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		req, err := c.buildRequest(ctx, method, path, query, header, body)
		if err != nil {
			return err
		}

		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("%s %s%s: %w", method, c.baseURL, path, err)
			continue // transport error: retry
		}
		err = c.decodeResponse(resp, out)
		if isRetryableError(err) {
			lastErr = err
			continue
		}
		return err
	}
	return fmt.Errorf("giving up after %d attempts: %w", attempts, lastErr)
}

func (c *apiClient) buildRequest(ctx context.Context, method, path string, query url.Values, header http.Header, body []byte) (*http.Request, error) {
	fullURL := c.baseURL + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, reader)
	if err != nil {
		return nil, err
	}
	for k, vals := range header {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	return req, nil
}

func (c *apiClient) decodeResponse(resp *http.Response, out interface{}) error {
	defer resp.Body.Close()
	limited := io.LimitReader(resp.Body, maxResponseBodyBytes)
	payload, err := io.ReadAll(limited)
	if err != nil {
		return &retryableError{err: fmt.Errorf("read response: %w", err)}
	}
	if resp.StatusCode >= 400 {
		msg := string(payload)
		if len(msg) > 300 {
			msg = msg[:300]
		}
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			re := &retryableError{err: fmt.Errorf("HTTP %d from %s: %s", resp.StatusCode, resp.Request.URL.Redacted(), msg)}
			setRetryAfter(re, resp.Header)
			return re
		}
		return fmt.Errorf("HTTP %d from %s: %s", resp.StatusCode, resp.Request.URL.Redacted(), msg)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(payload, out); err != nil {
		return fmt.Errorf("decode response from %s: %w", resp.Request.URL.Redacted(), err)
	}
	return nil
}

// backoffDuration computes exponential backoff, extended to the platform's
// Retry-After hint when present.
func (c *apiClient) backoffDuration(attempt int, err error) time.Duration {
	base := time.Duration(1<<uint(attempt-1)) * 500 * time.Millisecond
	if base > 30*time.Second {
		base = 30 * time.Second
	}
	if re, ok := err.(*retryableError); ok && re.retryAfter > base {
		return re.retryAfter
	}
	return base
}

type retryableError struct {
	err        error
	retryAfter time.Duration
}

func (e *retryableError) Error() string { return e.err.Error() }
func (e *retryableError) Unwrap() error { return e.err }

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*retryableError)
	return ok
}

// setRetryAfter attaches a Retry-After hint to a retryable error.
func setRetryAfter(err *retryableError, header http.Header) {
	if header == nil {
		return
	}
	if v := header.Get("Retry-After"); v != "" {
		if secs, perr := strconv.Atoi(v); perr == nil && secs > 0 && secs <= 300 {
			err.retryAfter = time.Duration(secs) * time.Second
			return
		}
		if at, perr := http.ParseTime(v); perr == nil {
			if d := time.Until(at); d > 0 && d <= 5*time.Minute {
				err.retryAfter = d
			}
		}
	}
}

func logConnectorMode(channel, mode string, detail string) {
	slog.Info("connector initialized", "channel", channel, "mode", mode, "detail", detail)
}
