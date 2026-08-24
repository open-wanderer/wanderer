package util

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestRateLimiterKeepsRollingWindowAcrossMaintenance(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	limiter := NewRateLimiter(2, time.Minute)
	limiter.now = func() time.Time { return now }

	if err := limiter.CheckRateLimit("user", "maneuvers"); err != nil {
		t.Fatalf("first request: %v", err)
	}
	now = now.Add(30 * time.Second)
	if err := limiter.CheckRateLimit("user", "maneuvers"); err != nil {
		t.Fatalf("second request: %v", err)
	}

	limiter.mu.Lock()
	limiter.pruneExpiredLocked(now)
	limiter.mu.Unlock()
	if err := limiter.CheckRateLimit("user", "maneuvers"); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("maintenance reset the active rolling window: %v", err)
	}

	now = now.Add(31 * time.Second)
	if err := limiter.CheckRateLimit("user", "maneuvers"); err != nil {
		t.Fatalf("expired requests were not released: %v", err)
	}
}

func TestFetchPublicURLRejectsUnsafeInputs(t *testing.T) {
	tests := []string{
		"ftp://example.com/file.jpg",
		"http://user:pass@example.com/file.jpg",
		"http://127.0.0.1/file.jpg",
		"http://localhost/file.jpg",
		"http://10.0.0.1/file.jpg",
		"http://169.254.169.254/latest/meta-data",
		"http://[::1]/file.jpg",
		"http://[fc00::1]/file.jpg",
		"http://example.com:8080/file.jpg",
	}
	for _, rawURL := range tests {
		t.Run(rawURL, func(t *testing.T) {
			if _, err := FetchPublicURL(context.Background(), rawURL, 1024); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestReadBoundedForPlugin(t *testing.T) {
	if _, err := ReadBoundedForPlugin(bytes.NewReader([]byte("1234")), 4); err != nil {
		t.Fatalf("unexpected exact-limit error: %v", err)
	}
	if _, err := ReadBoundedForPlugin(bytes.NewReader([]byte("12345")), 4); err == nil {
		t.Fatal("expected oversized response error")
	}
}

func TestSafeFetchedFileName(t *testing.T) {
	tests := []struct {
		name        string
		fallback    string
		finalURL    string
		contentType string
		want        string
	}{
		{
			name:        "uses URL filename with extension",
			fallback:    "activitypub-photo",
			finalURL:    "https://example.com/photos/camera.jpg?token=ignored",
			contentType: "image/png",
			want:        "camera.jpg",
		},
		{
			name:        "uses fallback extension when URL has none",
			fallback:    "activitypub-trail.gpx",
			finalURL:    "https://example.com/download",
			contentType: "application/xml+gpx",
			want:        "activitypub-trail.gpx",
		},
		{
			name:        "adds content type extension",
			fallback:    "activitypub-photo",
			finalURL:    "https://example.com/download",
			contentType: "image/png; charset=binary",
			want:        "download.png",
		},
		{
			name:        "sanitizes path fallback",
			fallback:    "../../bad",
			finalURL:    "https://example.com/",
			contentType: "",
			want:        "bad.bin",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := safeFetchedFileName(test.fallback, test.finalURL, test.contentType); got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestValidatePluginMediaStatus(t *testing.T) {
	for _, statusCode := range []int{200, 204, 299} {
		if err := ValidatePluginMediaStatus(statusCode); err != nil {
			t.Fatalf("status %d should be accepted: %v", statusCode, err)
		}
	}
	for _, statusCode := range []int{199, 300, 404, 429, 500} {
		if err := ValidatePluginMediaStatus(statusCode); err == nil {
			t.Fatalf("status %d should be rejected", statusCode)
		}
	}
}

func TestIsRetryablePluginMediaError(t *testing.T) {
	for _, statusCode := range []int{408, 425, 429, 500, 502, 503, 504} {
		if !IsRetryablePluginMediaError(ValidatePluginMediaStatus(statusCode)) {
			t.Fatalf("status %d should be retryable", statusCode)
		}
	}
	for _, statusCode := range []int{400, 401, 403, 404, 501} {
		if IsRetryablePluginMediaError(ValidatePluginMediaStatus(statusCode)) {
			t.Fatalf("status %d should not be retryable", statusCode)
		}
	}
	if IsRetryablePluginMediaError(fmt.Errorf("network failure")) {
		t.Fatal("untyped errors should not be retryable")
	}

	t.Run("client timeout", func(t *testing.T) {
		err := &url.Error{Op: "Get", URL: "https://example.com/photo.jpg", Err: context.DeadlineExceeded}
		if !IsRetryablePluginMediaError(err) {
			t.Fatal("client timeout should be retryable")
		}
	})

	t.Run("connection error", func(t *testing.T) {
		err := &url.Error{
			Op:  "Get",
			URL: "https://example.com/photo.jpg",
			Err: &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connection reset")},
		}
		if !IsRetryablePluginMediaError(err) {
			t.Fatal("connection error should be retryable")
		}
	})

	t.Run("truncated response", func(t *testing.T) {
		if !IsRetryablePluginMediaError(fmt.Errorf("read response: %w", io.ErrUnexpectedEOF)) {
			t.Fatal("truncated response should be retryable")
		}
	})

	t.Run("cancelled request", func(t *testing.T) {
		err := &url.Error{Op: "Get", URL: "https://example.com/photo.jpg", Err: context.Canceled}
		if IsRetryablePluginMediaError(err) {
			t.Fatal("cancelled request should not be retryable")
		}
	})

	t.Run("missing DNS name", func(t *testing.T) {
		err := &url.Error{
			Op:  "Get",
			URL: "https://missing.example/photo.jpg",
			Err: &net.DNSError{Name: "missing.example", IsNotFound: true},
		}
		if IsRetryablePluginMediaError(err) {
			t.Fatal("missing DNS name should not be retryable")
		}
	})
}

func TestConnectorTLSConfigRejectsInsecureMode(t *testing.T) {
	if _, err := connectorTLSConfig("insecure", nil); err == nil {
		t.Fatal("expected insecure TLS mode to be rejected")
	}
}

func TestConnectorIPAllowed(t *testing.T) {
	tests := []struct {
		ip            string
		allowPrivate  bool
		allowLoopback bool
		want          bool
	}{
		{ip: "8.8.8.8", want: true},
		{ip: "10.0.0.1", want: false},
		{ip: "10.0.0.1", allowPrivate: true, want: true},
		{ip: "fc00::1", allowPrivate: true, want: true},
		{ip: "127.0.0.1", allowPrivate: true, want: false},
		{ip: "127.0.0.1", allowPrivate: true, allowLoopback: true, want: true},
		{ip: "::1", allowPrivate: true, allowLoopback: true, want: true},
		{ip: "169.254.1.1", allowPrivate: true, want: false},
		{ip: "100.64.0.1", allowPrivate: true, want: false},
		{ip: "192.0.2.1", allowPrivate: true, want: false},
	}
	for _, test := range tests {
		t.Run(test.ip, func(t *testing.T) {
			if got := connectorIPAllowed(net.ParseIP(test.ip), test.allowPrivate, test.allowLoopback); got != test.want {
				t.Fatalf("got %v, want %v", got, test.want)
			}
		})
	}
}

func TestConnectorHTTPClientAllowsOnlyExplicitlyConfiguredLoopback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := ConnectorHTTPClient(ConnectorHTTPPolicy{
		BaseURL:      server.URL,
		AllowPrivate: true,
	}, nil)
	if err != nil {
		t.Fatalf("create loopback connector client: %v", err)
	}
	response, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("request explicit loopback connector: %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusNoContent)
	}
	localhostURL := strings.Replace(server.URL, "127.0.0.1", "localhost", 1)
	localhostClient, err := ConnectorHTTPClient(ConnectorHTTPPolicy{
		BaseURL:      localhostURL,
		AllowPrivate: true,
	}, nil)
	if err != nil {
		t.Fatalf("create localhost connector client: %v", err)
	}
	localhostResponse, err := localhostClient.Get(localhostURL)
	if err != nil {
		t.Fatalf("request localhost connector: %v", err)
	}
	localhostResponse.Body.Close()

	blocked, err := ConnectorHTTPClient(ConnectorHTTPPolicy{BaseURL: server.URL}, nil)
	if err != nil {
		t.Fatalf("create public-only connector client: %v", err)
	}
	if _, err := blocked.Get(server.URL); err == nil {
		t.Fatal("expected loopback connector without allowPrivate to be blocked")
	}
}

type staticConnectorIPResolver struct {
	ips []net.IP
}

func (r staticConnectorIPResolver) LookupIP(context.Context, string, string) ([]net.IP, error) {
	return append([]net.IP(nil), r.ips...), nil
}

func TestConnectorHTTPClientBlocksPublicNameResolvingToLoopback(t *testing.T) {
	previousResolver := connectorHostResolver
	connectorHostResolver = staticConnectorIPResolver{ips: []net.IP{net.ParseIP("127.0.0.1")}}
	t.Cleanup(func() { connectorHostResolver = previousResolver })

	client, err := ConnectorHTTPClient(ConnectorHTTPPolicy{
		BaseURL:      "http://public.example:17777",
		AllowPrivate: true,
	}, nil)
	if err != nil {
		t.Fatalf("create connector client: %v", err)
	}
	if _, err := client.Get("http://public.example:17777/brouter"); err == nil || !strings.Contains(err.Error(), "outside allowed IP policy") {
		t.Fatalf("public hostname resolving to loopback error = %v", err)
	}
}

func TestConnectorHostIsLoopback(t *testing.T) {
	tests := map[string]bool{
		"localhost":   true,
		"LOCALHOST":   true,
		"127.0.0.1":   true,
		"127.0.0.2":   true,
		"::1":         true,
		"example.com": false,
		"10.0.0.1":    false,
	}
	for host, want := range tests {
		if got := connectorHostIsLoopback(host); got != want {
			t.Errorf("connectorHostIsLoopback(%q) = %v, want %v", host, got, want)
		}
	}
}
