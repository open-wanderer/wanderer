package geocoding

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

func TestPublicQueueSerializesBodyAndSpacesStarts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		release := make(chan struct{})
		var starts []time.Duration
		var paths []string
		c := stubClient(t, "https://NOMINATIM.OpenStreetMap.ORG.:443", func(req *http.Request) (*http.Response, error) {
			starts = append(starts, time.Since(start))
			paths = append(paths, req.URL.Path)
			resp := stubResponse(http.StatusOK, "")
			if len(starts) == 1 {
				resp.Body = &testBody{Reader: readerFunc(func(p []byte) (int, error) {
					select {
					case <-release:
						return 0, io.EOF
					case <-req.Context().Done():
						return 0, req.Context().Err()
					}
				})}
			}
			return resp, nil
		})

		first := runSearch(c, ctx)
		synctest.Wait()
		second := runReverse(c, ctx)
		queuedCtx, cancelQueued := context.WithCancelCause(ctx)
		defer cancelQueued(nil)
		queued := runSearch(c, queuedCtx)
		synctest.Wait()
		cancelQueued(errors.New("private cancellation detail"))
		if err := <-queued; err != context.Canceled {
			t.Fatalf("queued error = %v, want context.Canceled", err)
		}

		time.Sleep(2 * time.Second)
		synctest.Wait()
		if len(starts) != 1 {
			t.Errorf("%d attempts started while the first body was still open, want 1", len(starts))
		}
		close(release)
		for _, done := range []<-chan error{first, second} {
			if err := <-done; err != nil {
				t.Fatal(err)
			}
		}
		if _, err := c.Search(ctx, "another search", ""); err != nil {
			t.Fatal(err)
		}
		if want := []time.Duration{0, 2 * time.Second, 3 * time.Second}; !reflect.DeepEqual(starts, want) {
			t.Errorf("starts = %v, want %v", starts, want)
		}
		if want := []string{"/search", "/reverse", "/search"}; !reflect.DeepEqual(paths, want) {
			t.Errorf("paths = %v, want %v", paths, want)
		}
	})
}

func TestPublicQueueIncludesBothKindsOfRetry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		var starts []time.Duration
		var paths []string
		c := stubClient(t, "", func(req *http.Request) (*http.Response, error) {
			starts = append(starts, time.Since(start))
			paths = append(paths, req.URL.Path)
			switch len(starts) {
			case 1:
				return nil, errors.New("test transport failure")
			case 2:
				return stubResponse(http.StatusTooManyRequests, "2"), nil
			default:
				return stubResponse(http.StatusOK, ""), nil
			}
		})

		search := runSearch(c, t.Context())
		synctest.Wait()
		reverse := runReverse(c, t.Context())
		for _, done := range []<-chan error{search, reverse} {
			if err := <-done; err != nil {
				t.Fatal(err)
			}
		}
		if want := []time.Duration{0, time.Second, 3 * time.Second, 4 * time.Second}; !reflect.DeepEqual(starts, want) {
			t.Errorf("starts = %v, want %v", starts, want)
		}
		if want := []string{"/search", "/search", "/search", "/reverse"}; !reflect.DeepEqual(paths, want) {
			t.Errorf("paths = %v, want %v", paths, want)
		}
	})
}

func TestClientRetryAfter(t *testing.T) {
	for _, tt := range []struct {
		name, value string
		date        time.Duration
		starts      []time.Duration
	}{
		{name: "numeric seconds", value: "2.5", starts: []time.Duration{0, 2500 * time.Millisecond, 5 * time.Second}},
		{name: "HTTP date", date: 5 * time.Second, starts: []time.Duration{0, 5 * time.Second, 5 * time.Second}},
		{name: "past date", date: -time.Second, starts: []time.Duration{0, 0, 0}},
		{name: "missing", starts: []time.Duration{0, time.Second, 3 * time.Second}},
		{name: "invalid", value: "not a date", starts: []time.Duration{0, time.Second, 3 * time.Second}},
		{name: "non-finite", value: "Infinity", starts: []time.Duration{0, time.Second, 3 * time.Second}},
		{name: "exact cap", value: "30", starts: []time.Duration{0, 30 * time.Second, 60 * time.Second}},
		{name: "seconds over cap", value: "3600", starts: []time.Duration{0}},
		{name: "date over cap", date: 31 * time.Second, starts: []time.Duration{0}},
		{name: "overflowing seconds", value: "1e400", starts: []time.Duration{0}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				start := time.Now()
				value := tt.value
				if tt.date != 0 {
					value = start.Add(tt.date).UTC().Format(http.TimeFormat)
				}
				var starts []time.Duration
				c := stubClient(t, "http://private/nominatim.openstreetmap.org", func(_ *http.Request) (*http.Response, error) {
					starts = append(starts, time.Since(start))
					return stubResponse(http.StatusTooManyRequests, value), nil
				})
				resp, err := c.Search(t.Context(), "busy", "")
				if err != nil || resp.StatusCode != http.StatusTooManyRequests || string(resp.Body) != upstreamFailurePayload {
					t.Errorf("response = %+v, %v; want sanitized 429", resp, err)
				}
				if !reflect.DeepEqual(starts, tt.starts) {
					t.Errorf("starts = %v, want %v", starts, tt.starts)
				}
			})
		})
	}
}

func TestClientUpstreamFailuresAreSanitized(t *testing.T) {
	detail := errors.New("http://test:pass@private:8080/search?q=private")
	timeout := fmt.Errorf("network timeout: %w", testTimeoutError{detail})
	for _, tt := range []struct {
		name, readEncoding string
		cause, want        error
	}{
		{name: "transport failure", cause: detail, want: ErrRequestFailed},
		{name: "transport timeout", cause: timeout, want: ErrRequestTimeout},
		{name: "body read failure", readEncoding: "identity", cause: detail, want: ErrRequestFailed},
		{name: "body read timeout", readEncoding: "identity", cause: timeout, want: ErrRequestTimeout},
		{name: "gzip header read timeout", readEncoding: "gzip", cause: timeout, want: ErrRequestTimeout},
	} {
		t.Run(tt.name, func(t *testing.T) {
			attempts := 0
			c := stubClient(t, "http://test:pass@private:8080", func(_ *http.Request) (*http.Response, error) {
				attempts++
				if tt.readEncoding == "" {
					return nil, tt.cause
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Encoding": {tt.readEncoding}},
					Body:       io.NopCloser(readerFunc(func([]byte) (int, error) { return 0, tt.cause })),
				}, nil
			})
			if _, err := c.Search(t.Context(), "private", ""); err != tt.want {
				t.Errorf("error = %v, want %v without original cause", err, tt.want)
			}
			if attempts != 3 {
				t.Errorf("attempts = %d, want 3", attempts)
			}
		})
	}
}

type testTimeoutError struct{ error }

func (testTimeoutError) Timeout() bool { return true }

func TestAttemptTimeoutCoversHeadersAndBodyAndReleasesQueue(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		var starts []time.Duration
		closedBodies := 0
		c := stubClient(t, "", func(req *http.Request) (*http.Response, error) {
			starts = append(starts, time.Since(start))
			if deadline, ok := req.Context().Deadline(); !ok || deadline.Sub(time.Now()) != 10*time.Second {
				t.Error("request does not have a 10-second attempt deadline")
			}
			if req.URL.Path == "/reverse" {
				return stubResponse(http.StatusOK, ""), nil
			}
			if len(starts) == 1 {
				// No headers arrive on the first attempt.
				<-req.Context().Done()
				return nil, req.Context().Err()
			}
			// Later attempts receive headers, then stall in the body. The same
			// deadline must cover both phases.
			time.Sleep(4 * time.Second)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       &testBody{Reader: readUntilCanceled(req.Context()), onClose: func() { closedBodies++ }},
			}, nil
		})
		search := runSearch(c, t.Context())
		synctest.Wait()
		reverse := runReverse(c, t.Context())

		if err := <-search; err != ErrRequestTimeout {
			t.Errorf("search error = %v, want sanitized timeout failure", err)
		}
		if err := <-reverse; err != nil {
			t.Errorf("queued reverse failed: %v", err)
		}
		if want := []time.Duration{0, 10 * time.Second, 20 * time.Second, 30 * time.Second}; !reflect.DeepEqual(starts, want) {
			t.Errorf("starts = %v, want %v", starts, want)
		}
		if closedBodies != 2 {
			t.Errorf("closed bodies = %d, want 2", closedBodies)
		}
	})
}

func TestCallerDeadlineInterruptsWaitsAndIO(t *testing.T) {
	for _, phase := range []string{"spacing", "retry backoff", "body read"} {
		t.Run(phase, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				start := time.Now()
				attempts := 0
				c := stubClient(t, "", func(req *http.Request) (*http.Response, error) {
					attempts++
					if req.URL.Path == "/search" {
						switch phase {
						case "retry backoff":
							return stubResponse(http.StatusTooManyRequests, "10"), nil
						case "body read":
							return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(readUntilCanceled(req.Context()))}, nil
						}
					}
					return stubResponse(http.StatusOK, ""), nil
				})
				if phase == "spacing" {
					if _, err := c.Search(t.Context(), "seed", ""); err != nil {
						t.Fatal(err)
					}
				}

				ctx, cancel := context.WithTimeout(t.Context(), 250*time.Millisecond)
				defer cancel()
				if _, err := c.Search(ctx, "interrupt me", ""); err != context.DeadlineExceeded {
					t.Errorf("error = %v, want caller deadline", err)
				}
				if attempts != 1 || time.Since(start) != 250*time.Millisecond {
					t.Errorf("%d attempts after %s; want 1 at caller deadline", attempts, time.Since(start))
				}
				if _, err := c.Reverse(t.Context(), "1", "2"); err != nil {
					t.Fatal(err)
				}
				if attempts != 2 || time.Since(start) != time.Second {
					t.Errorf("%d attempts after %s; want queue released at next slot", attempts, time.Since(start))
				}
			})
		})
	}
}

func stubResponse(status int, retryAfter string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Retry-After": {retryAfter}},
		Body:       io.NopCloser(strings.NewReader(`{}`)),
	}
}

func readUntilCanceled(ctx context.Context) io.Reader {
	return readerFunc(func([]byte) (int, error) {
		<-ctx.Done()
		return 0, ctx.Err()
	})
}

func runSearch(c *Client, ctx context.Context) <-chan error {
	done := make(chan error, 1)
	go func() {
		_, err := c.Search(ctx, "test search", "")
		done <- err
	}()
	return done
}

func runReverse(c *Client, ctx context.Context) <-chan error {
	done := make(chan error, 1)
	go func() {
		_, err := c.Reverse(ctx, "1", "2")
		done <- err
	}()
	return done
}
