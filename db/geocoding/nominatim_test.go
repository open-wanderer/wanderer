package geocoding

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

func TestClientWireFormat(t *testing.T) {
	const (
		query   = "Höhe &format=other? # +"
		payload = " {\n\"type\":\"FeatureCollection\", \"features\":[]}\n"
	)
	type request struct {
		path   string
		params url.Values
	}
	requests := make(chan request, 9)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", r.Method)
		}
		if r.UserAgent() != "wanderer/0.21.2+test (+https://github.com/open-wanderer/wanderer)" {
			t.Errorf("User-Agent = %q", r.UserAgent())
		}
		if user, pass, ok := r.BasicAuth(); !ok || user != "service@local" || pass != "test:pass" {
			t.Error("missing or incorrect Basic Auth")
		}
		if r.URL.User != nil {
			t.Error("unexpected userinfo in request URL")
		}
		requests <- request{r.URL.EscapedPath(), r.URL.Query()}
		w.Header().Set("Content-Type", "application/geo+json")
		w.WriteHeader(http.StatusAccepted)
		io.WriteString(w, payload)
	}))
	defer upstream.Close()

	base := "http://service%40local:test%3Apass@" + strings.TrimPrefix(upstream.URL, "http://") + "/mount%20point/geo%2F"
	c := newClientForTest(t, base, "0.21.2+test")
	for _, tt := range []struct {
		name, path string
		params     url.Values
		call       func() (Response, error)
	}{
		{
			"search with limit", "search", url.Values{"q": {query}, "limit": {"2e0"}},
			func() (Response, error) { return c.Search(t.Context(), query, "2e0") },
		},
		{
			"search without limit", "search", url.Values{"q": {query}},
			func() (Response, error) { return c.Search(t.Context(), query, "") },
		},
		{
			"reverse", "reverse", url.Values{"lat": {"+01.20"}, "lon": {"-1e-4"}},
			func() (Response, error) { return c.Reverse(t.Context(), "+01.20", "-1e-4") },
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := tt.call()
			if err != nil || resp.StatusCode != http.StatusAccepted || string(resp.Body) != payload {
				t.Fatalf("response = %+v, %v; want 202 and unchanged GeoJSON", resp, err)
			}
			got := <-requests
			if want := "/mount%20point/geo%2F/" + tt.path; got.path != want {
				t.Errorf("path = %q, want %q", got.path, want)
			}
			tt.params.Set("format", "geojson")
			tt.params.Set("addressdetails", "1")
			if !reflect.DeepEqual(got.params, tt.params) {
				t.Errorf("params = %v, want %v", got.params, tt.params)
			}
		})
	}
}

func TestNewClientBaseURL(t *testing.T) {
	for _, tt := range []struct {
		name, base, scheme, host string
		public                   bool
	}{
		{"default", " ", "https", publicHost, true},
		{"schemeless", " nominatim.openstreetmap.org/nominatim ", "https", publicHost, true},
		{"normalized public host", "HTTP://NOMINATIM.OpenStreetMap.ORG.:8080/api", "http", "NOMINATIM.OpenStreetMap.ORG.:8080", true},
		{"private name", "nominatim:8080", "https", "nominatim:8080", false},
		{"private IPv6", "http://[fd00::1]:8080", "http", "[fd00::1]:8080", false},
		{"lookalike host and path", "http://nominatim.openstreetmap.org.example/nominatim.openstreetmap.org", "http", "nominatim.openstreetmap.org.example", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c := newClientForTest(t, tt.base, "")
			if c.baseURL.Scheme != tt.scheme || c.baseURL.Host != tt.host || (c.queue != nil) != tt.public {
				t.Errorf("parsed URL = %s, public = %t", c.baseURL, c.queue != nil)
			}
			if !strings.HasPrefix(c.userAgent, "wanderer/dev ") {
				t.Errorf("default User-Agent = %q", c.userAgent)
			}
		})
	}

	for _, bad := range []string{
		"ftp://user:test-pass@nominatim", "http:///no-host", "http://nominatim:65536",
		"https:/nominatim",
		"https://user:test-pass@nominatim/?token=test-secret", "https://nominatim/#",
		"https://nominatim。openstreetmap.org",
	} {
		if _, err := NewClient(bad, "test"); err != ErrInvalidBaseURL {
			t.Errorf("invalid base returned %v, want sanitized configuration error", err)
		}
	}
	if _, err := NewClient("", "test\r\nX-Extra: test"); err != ErrInvalidVersion {
		t.Errorf("invalid version returned %v, want sanitized configuration error", err)
	}
}

func TestClientDoesNotFollowRedirects(t *testing.T) {
	var received atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
	}))
	defer other.Close()

	for _, tt := range []struct {
		name, location string
		tls            bool
	}{
		{"same origin", "/another-path", false},
		{"different origin", other.URL + "/another-path", false},
		{"HTTPS downgrade", other.URL + "/another-path", true},
		{"invalid Location", "http://[invalid", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var requests atomic.Int32
			upstream := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if _, _, ok := r.BasicAuth(); !ok {
					t.Error("missing configured auth on initial request")
				}
				w.Header().Set("Location", tt.location)
				w.WriteHeader(http.StatusTemporaryRedirect)
				io.WriteString(w, `{"message":"diagnostic containing credentials"}`)
			}))
			if tt.tls {
				upstream.StartTLS()
			} else {
				upstream.Start()
			}
			defer upstream.Close()

			c := newClientForTest(t, strings.Replace(upstream.URL, "://", "://test:pass@", 1), "test")
			if tt.tls {
				// Trust only the loopback server's test certificate.
				c.transport.(*http.Transport).TLSClientConfig = upstream.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
			}
			resp, err := c.Reverse(t.Context(), "1", "2")
			if err != nil || resp.StatusCode != http.StatusTemporaryRedirect || string(resp.Body) != upstreamFailurePayload {
				t.Fatalf("response = %+v, %v; want sanitized redirect status and body", resp, err)
			}
			if n := requests.Load(); n != 1 {
				t.Errorf("initial origin got %d requests, want 1", n)
			}
		})
	}
	if n := received.Load(); n != 0 {
		t.Errorf("redirect target got %d requests, want 0", n)
	}
}

func TestClientJSONContract(t *testing.T) {
	for _, tt := range []struct {
		name, encoding string
		status         int
		body, want     string
	}{
		{name: "valid JSON", status: 200, body: "[null, true]", want: "[null, true]"},
		{name: "empty", status: 204, want: `{}`},
		{name: "empty with encoding header", status: 204, encoding: "gzip", want: `{}`},
		{name: "UTF-8 BOM", status: 200, body: "\xef\xbb\xbf{}", want: `{}`},
		{name: "non-JSON", status: 200, body: "test-secret in gateway diagnostic", want: invalidResponsePayload},
		{name: "non-UTF-8", status: 200, body: "\"\xff\"", want: invalidResponsePayload},
		{name: "unsupported encoding", status: 200, encoding: "br", body: "encoded", want: invalidResponsePayload},
		{name: "error JSON", status: 404, body: `{"error": "test-secret"}`, want: upstreamFailurePayload},
		{name: "error text", status: 503, body: "http://test:test-secret@private/service", want: upstreamFailurePayload},
	} {
		t.Run(tt.name, func(t *testing.T) {
			attempts := 0
			c := stubClient(t, "http://private", func(_ *http.Request) (*http.Response, error) {
				attempts++
				reader := strings.NewReader(tt.body)
				body := io.NopCloser(reader)
				if tt.status == http.StatusNoContent {
					body = http.NoBody
				}
				if tt.status >= 400 {
					body = &testBody{Reader: readerFunc(func(p []byte) (int, error) {
						t.Error("unexpected read of upstream error body")
						return reader.Read(p)
					})}
				}
				return &http.Response{StatusCode: tt.status, Header: http.Header{"Content-Encoding": {tt.encoding}}, Body: body}, nil
			})
			resp, err := c.Search(t.Context(), "a place", "")
			if err != nil || resp.StatusCode != tt.status || string(resp.Body) != tt.want {
				t.Errorf("response = %+v, %v; want %d %s", resp, err, tt.status, tt.want)
			}
			if attempts != 1 {
				t.Errorf("attempts = %d, want 1", attempts)
			}
		})
	}
}

func TestClientResponseLimits(t *testing.T) {
	maxBody := []byte(`"` + strings.Repeat("x", maxResponseBytes-2) + `"`)
	oversized := []byte(`"` + strings.Repeat("x", maxResponseBytes-1) + `"`)
	largeMember := gzipForTest(t, nil, make([]byte, 65535))
	for _, tt := range []struct {
		name          string
		wire          []byte
		gzip, chunked bool
		tooLarge      bool
	}{
		{name: "exact limit", wire: maxBody},
		{name: "content length over limit", wire: oversized, tooLarge: true},
		{name: "chunked over limit", wire: oversized, chunked: true, tooLarge: true},
		{name: "gzip decoded exact limit", wire: gzipForTest(t, maxBody, nil), gzip: true},
		{name: "gzip decoded over limit", wire: gzipForTest(t, oversized, nil), gzip: true, chunked: true, tooLarge: true},
		{name: "gzip wire over limit", wire: bytes.Repeat(largeMember, maxGzipResponseBytes/len(largeMember)+1), gzip: true, chunked: true, tooLarge: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var requests atomic.Int32
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if tt.gzip {
					if r.Header.Get("Accept-Encoding") != "gzip" {
						t.Error("client did not negotiate gzip")
					}
					w.Header().Set("Content-Encoding", "gzip")
				}
				if tt.chunked {
					w.(http.Flusher).Flush()
				} else {
					w.Header().Set("Content-Length", strconv.Itoa(len(tt.wire)))
				}
				w.Write(tt.wire)
			}))
			defer upstream.Close()

			resp, err := newClientForTest(t, upstream.URL, "test").Search(t.Context(), "test", "")
			if tt.tooLarge {
				if err != ErrResponseTooLarge || len(resp.Body) != 0 {
					t.Errorf("oversized body: len = %d, error = %v", len(resp.Body), err)
				}
			} else if err != nil || resp.StatusCode != http.StatusOK || !bytes.Equal(resp.Body, maxBody) {
				t.Errorf("bounded body: status = %d, len = %d, error = %v", resp.StatusCode, len(resp.Body), err)
			}
			if n := requests.Load(); n != 1 {
				t.Errorf("requests = %d, want 1", n)
			}
		})
	}
}

func gzipForTest(t *testing.T, data, extra []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	writer.Extra = extra
	if _, err := writer.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func newClientForTest(t *testing.T, base, version string) *Client {
	t.Helper()
	c, err := NewClient(base, version)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func stubClient(t *testing.T, base string, f roundTripFunc) *Client {
	t.Helper()
	c := newClientForTest(t, base, "test")
	c.transport = f
	return c
}

type readerFunc func([]byte) (int, error)

func (f readerFunc) Read(p []byte) (int, error) { return f(p) }

type testBody struct {
	io.Reader
	onClose func()
}

func (b *testBody) Close() error {
	if b.onClose != nil {
		b.onClose()
	}
	return nil
}
