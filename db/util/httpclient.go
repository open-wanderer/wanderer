package util

import (
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// IntegrationHTTPClient is the shared HTTP client for outbound requests to the
// external fitness providers (Strava, Komoot, Hammerhead). Unlike a zero-value
// http.Client it enforces an overall timeout, so a stalled provider connection
// can never hang the nightly sync indefinitely.
var IntegrationHTTPClient = &http.Client{
	Timeout: 60 * time.Second,
}

const (
	// MaxIntegrationDownloadBytes caps the size of a single downloaded payload
	// (GPX export, photo, JSON response) so an unexpectedly huge or malicious
	// response cannot exhaust memory during a sync.
	MaxIntegrationDownloadBytes int64 = 100 << 20 // 100 MiB

	integrationMaxAttempts   = 4
	integrationBaseBackoff   = 500 * time.Millisecond
	integrationMaxBackoff    = 30 * time.Second
	integrationMaxRetryAfter = 60 * time.Second
)

// DoWithRetry executes an idempotent HTTP request, retrying on transient
// failures: network errors, HTTP 429 (Too Many Requests) and 5xx responses. It
// honors a Retry-After header when present and otherwise backs off
// exponentially with jitter.
//
// newRequest is a factory rather than a *http.Request because a request body
// can only be consumed once; each attempt needs a freshly built request. Use
// this only for idempotent operations (GET, token refresh) - never for uploads
// that would duplicate data when retried.
func DoWithRetry(client *http.Client, newRequest func() (*http.Request, error)) (*http.Response, error) {
	if client == nil {
		client = IntegrationHTTPClient
	}

	var lastErr error
	var wait time.Duration

	for attempt := 1; attempt <= integrationMaxAttempts; attempt++ {
		if wait > 0 {
			time.Sleep(wait)
		}

		req, err := newRequest()
		if err != nil {
			return nil, err
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			wait = backoffDuration(attempt, 0)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("received status %d", resp.StatusCode)
			retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
			resp.Body.Close()
			wait = backoffDuration(attempt, retryAfter)
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", integrationMaxAttempts, lastErr)
}

// backoffDuration returns how long to wait before the next attempt. A positive
// retryAfter (from a Retry-After header) takes precedence but is capped, so a
// provider cannot stall a sync for an arbitrarily long time. Otherwise it grows
// exponentially with the attempt number and adds jitter to avoid thundering herds.
func backoffDuration(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		if retryAfter > integrationMaxRetryAfter {
			return integrationMaxRetryAfter
		}
		return retryAfter
	}

	backoff := float64(integrationBaseBackoff) * math.Pow(2, float64(attempt-1))
	if backoff > float64(integrationMaxBackoff) {
		backoff = float64(integrationMaxBackoff)
	}
	jitter := time.Duration(rand.Int63n(int64(integrationBaseBackoff)))
	return time.Duration(backoff) + jitter
}

// parseRetryAfter interprets a Retry-After header value, which is either a
// number of seconds or an HTTP date. Returns 0 when absent or unparseable.
func parseRetryAfter(v string) time.Duration {
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return 0
}

// ReadAllLimited reads from r up to max bytes, returning an error if the source
// exceeds the limit. Use it instead of io.ReadAll for response bodies whose size
// is controlled by a remote server.
func ReadAllLimited(r io.Reader, max int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, fmt.Errorf("response exceeds maximum allowed size of %d bytes", max)
	}
	return data, nil
}
