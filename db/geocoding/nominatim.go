// Package geocoding provides a client for a trusted, operator-configured
// Nominatim service.
package geocoding

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// DefaultBaseURL is the public Nominatim service.
	DefaultBaseURL = "https://" + publicHost
	publicHost     = "nominatim.openstreetmap.org"
	publicInterval = time.Second
	attemptTimeout = 10 * time.Second
	maxRetries     = 2
	maxRetryAfter  = 30 * time.Second

	maxResponseBytes       = 1 << 20
	maxGzipResponseBytes   = 2 * maxResponseBytes
	invalidResponsePayload = `{"message":"Nominatim returned a non-JSON response"}`
	upstreamFailurePayload = `{"message":"Nominatim request failed"}`
)

var (
	// ErrInvalidBaseURL indicates an unsupported or malformed operator URL.
	ErrInvalidBaseURL = errors.New("invalid Nominatim base URL")
	// ErrInvalidVersion indicates a version that cannot be used as a product token.
	ErrInvalidVersion = errors.New("invalid Wanderer version")

	// ErrRequestFailed indicates a non-timeout request or response-read failure.
	ErrRequestFailed = errors.New("Nominatim request failed")
	// ErrRequestTimeout indicates that the last upstream attempt timed out.
	// Caller deadlines and cancellations return the caller's context error instead.
	ErrRequestTimeout = errors.New("Nominatim request timed out")
	// ErrResponseTooLarge indicates that the successful response exceeded a body limit.
	ErrResponseTooLarge = errors.New("Nominatim response exceeds size limit")
)

// Response is an upstream HTTP result. StatusCode preserves the upstream
// status, including on non-2xx responses. Body is always JSON: the original
// JSON/GeoJSON on success, {} for an empty success, or a generic message for
// non-2xx statuses and non-JSON or unsupported-encoding success bodies.
// No headers or upstream error body content are exposed. HTTP results have a
// nil error; failures to obtain a usable result return a zero Response.
type Response struct {
	StatusCode int
	Body       json.RawMessage
}

// Client is safe for concurrent use. Construct it with NewClient and reuse
// a single client for search and reverse so public Nominatim calls share the
// same rate limit.
type Client struct {
	baseURL   *url.URL
	basicAuth *url.Userinfo
	userAgent string
	transport http.RoundTripper
	queue     *publicQueue
}

type publicQueue struct {
	slot      chan struct{}
	lastStart time.Time // protected by slot
}

// NewClient creates a client for a fixed HTTP(S) URL. An empty baseURL uses
// DefaultBaseURL, and a schemeless baseURL defaults to HTTPS. An explicit port
// and path prefix are supported, but query strings and fragments are not.
// Use ASCII (or punycode) for the host. URL userinfo is sent as Basic Auth, so
// prefer HTTPS to avoid sending credentials in cleartext. An empty version
// identifies the application as a dev build.
func NewClient(baseURL, version string) (*Client, error) {
	base, err := parseBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	version = strings.TrimSpace(version)
	if version == "" {
		version = "dev"
	}
	for _, r := range version {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' ||
			strings.ContainsRune(".+-_", r)) {
			return nil, ErrInvalidVersion
		}
	}

	// Keep retry ownership here: Go can silently replay GET requests on reused
	// HTTP/1 connections and retry HTTP/2 streams. With fresh HTTP/1 connections
	// every attempt goes through our retry budget and public pacing.
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	client := &Client{
		baseURL:   base,
		basicAuth: base.User,
		userAgent: "wanderer/" + version + " (+https://github.com/open-wanderer/wanderer)",
		transport: &http.Transport{
			Proxy:                  http.ProxyFromEnvironment,
			Protocols:              protocols,
			DisableKeepAlives:      true,
			DisableCompression:     true, // bound both gzip input and decoded output ourselves
			MaxResponseHeaderBytes: 64 << 10,
		},
	}
	base.User = nil
	if strings.EqualFold(strings.TrimSuffix(base.Hostname(), "."), publicHost) {
		client.queue = &publicQueue{slot: make(chan struct{}, 1)}
	}
	return client, nil
}

// UsesPublicService reports whether the configured host is public Nominatim,
// using the same decision as the shared rate limiter.
func (c *Client) UsesPublicService() bool {
	return c.queue != nil
}

func parseBaseURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = DefaultBaseURL
	} else if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	base, err := url.Parse(raw)
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") ||
		base.Hostname() == "" || strings.HasSuffix(base.Host, ":") ||
		base.ForceQuery || base.RawQuery != "" || strings.Contains(raw, "#") {
		return nil, ErrInvalidBaseURL
	}

	host := base.Hostname()
	for _, r := range host {
		// net/http can implicitly convert a Unicode host to the public host
		// after the rate-limit decision. Require the actual wire spelling.
		if r >= utf8.RuneSelf {
			return nil, ErrInvalidBaseURL
		}
	}
	if strings.HasPrefix(base.Host, "[") {
		addr, err := netip.ParseAddr(host)
		if err != nil || !addr.Is6() {
			return nil, ErrInvalidBaseURL
		}
	} else if strings.Contains(host, ":") {
		return nil, ErrInvalidBaseURL // IPv6 literals must be bracketed
	}
	if port := base.Port(); port != "" {
		p, err := strconv.ParseUint(port, 10, 16)
		if err != nil || p == 0 {
			return nil, ErrInvalidBaseURL
		}
	}

	// Make relative endpoints resolve under the prefix, preserving path escapes
	// (including a literal escaped slash at the end of the prefix).
	if !strings.HasSuffix(base.EscapedPath(), "/") {
		base.Path += "/"
		if base.RawPath != "" {
			base.RawPath += "/"
		}
	}
	return base, nil
}

// Search requests GeoJSON with address details. Pass an empty limit to omit
// it. Query values are encoded without reinterpretation; callers can validate
// them according to their entry point's input contract.
func (c *Client) Search(ctx context.Context, q, limit string) (Response, error) {
	params := url.Values{"q": {q}}
	if limit != "" {
		params.Set("limit", limit)
	}
	return c.get(ctx, "search", params)
}

// Reverse requests GeoJSON with address details for lat and lon. The
// coordinate values are encoded without reinterpretation.
func (c *Client) Reverse(ctx context.Context, lat, lon string) (Response, error) {
	return c.get(ctx, "reverse", url.Values{"lat": {lat}, "lon": {lon}})
}

func (c *Client) get(ctx context.Context, endpoint string, params url.Values) (Response, error) {
	params.Set("format", "geojson")
	params.Set("addressdetails", "1")
	target := c.baseURL.ResolveReference(&url.URL{Path: endpoint, RawQuery: params.Encode()})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return Response{}, ErrRequestFailed
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/geo+json, application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	if c.basicAuth != nil {
		password, _ := c.basicAuth.Password()
		req.SetBasicAuth(c.basicAuth.Username(), password)
	}

	if c.queue != nil {
		select {
		case c.queue.slot <- struct{}{}:
			defer func() { <-c.queue.slot }()
		case <-ctx.Done():
			return Response{}, ctx.Err()
		}
	}

	for attempt := 0; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return Response{}, err
		}
		if c.queue != nil {
			if err := wait(ctx, time.Until(c.queue.lastStart.Add(publicInterval))); err != nil {
				return Response{}, err
			}
		}

		response, retryAfter, err := c.attempt(req)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return Response{}, ctxErr
		}
		if attempt == maxRetries {
			return response, err
		}
		if err == ErrRequestFailed || err == ErrRequestTimeout {
			continue
		}
		if err != nil || response.StatusCode != http.StatusTooManyRequests {
			return response, err
		}
		delay := retryDelay(retryAfter, attempt, time.Now())
		if delay > maxRetryAfter {
			return response, nil
		}
		if err := wait(ctx, delay); err != nil {
			return Response{}, err
		}
	}
}

func (c *Client) attempt(req *http.Request) (Response, string, error) {
	ctx, cancel := context.WithTimeout(req.Context(), attemptTimeout)
	defer cancel()
	if ctx.Err() != nil {
		return Response{}, "", requestFailure(ctx, nil)
	}
	req = req.WithContext(ctx)
	if c.queue != nil {
		c.queue.lastStart = time.Now()
	}

	// A RoundTrip never follows redirects, including same-origin ones. Both
	// endpoints have fixed direct paths. A self-hosted instance behind a
	// redirect should be configured with its final base URL. This also keeps
	// credentials and pacing confined to the configured origin.
	resp, err := c.transport.RoundTrip(req)
	if err != nil {
		return Response{}, "", requestFailure(ctx, err)
	}
	defer resp.Body.Close()

	result := Response{StatusCode: resp.StatusCode}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// The body may contain reflected credentials, URLs or diagnostic data.
		// Do not read it or allow it to delay a status-based retry decision.
		result.Body = json.RawMessage(upstreamFailurePayload)
		return result, resp.Header.Get("Retry-After"), nil
	}
	result.Body, err = readSuccessBody(ctx, resp)
	if err != nil {
		return Response{}, "", err
	}
	if ctx.Err() != nil {
		return Response{}, "", requestFailure(ctx, nil)
	}
	return result, "", nil
}

// Classify without returning or wrapping the potentially sensitive cause.
// The caller's context error takes precedence in get.
func requestFailure(ctx context.Context, cause error) error {
	var timeout interface{ Timeout() bool }
	if ctx.Err() == context.DeadlineExceeded || (errors.As(cause, &timeout) && timeout.Timeout()) {
		return ErrRequestTimeout
	}
	return ErrRequestFailed
}

func readSuccessBody(ctx context.Context, resp *http.Response) (json.RawMessage, error) {
	if resp.Body == http.NoBody {
		return json.RawMessage(`{}`), nil
	}
	var reader io.Reader = resp.Body
	var compressed *io.LimitedReader
	switch strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Encoding"))) {
	case "", "identity":
		if resp.ContentLength > maxResponseBytes {
			return nil, ErrResponseTooLarge
		}
	case "gzip":
		if resp.ContentLength > maxGzipResponseBytes {
			return nil, ErrResponseTooLarge
		}
		compressed = &io.LimitedReader{R: reader, N: maxGzipResponseBytes + 1}
		decoded, err := gzip.NewReader(compressed)
		if compressed.N == 0 {
			return nil, ErrResponseTooLarge
		}
		if err != nil {
			return nil, requestFailure(ctx, err)
		}
		defer decoded.Close()
		reader = decoded
	default:
		return json.RawMessage(invalidResponsePayload), nil
	}

	body, err := io.ReadAll(io.LimitReader(reader, maxResponseBytes+1))
	if len(body) > maxResponseBytes || compressed != nil && compressed.N == 0 {
		return nil, ErrResponseTooLarge
	}
	if err != nil {
		return nil, requestFailure(ctx, err)
	}
	body = bytes.TrimPrefix(body, []byte("\xef\xbb\xbf"))
	if len(body) == 0 {
		return json.RawMessage(`{}`), nil
	}
	if !utf8.Valid(body) || !json.Valid(body) {
		return json.RawMessage(invalidResponsePayload), nil
	}
	return body, nil
}

func retryDelay(value string, attempt int, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	seconds, err := strconv.ParseFloat(value, 64)
	if seconds >= 0 && (err == nil && !math.IsInf(seconds, 0) || errors.Is(err, strconv.ErrRange)) {
		// Compare before converting to time.Duration so large values can't
		// overflow and turn into short/negative waits.
		if seconds > maxRetryAfter.Seconds() {
			return maxRetryAfter + time.Nanosecond
		}
		return time.Duration(seconds * float64(time.Second))
	}
	if at, err := http.ParseTime(value); err == nil {
		return max(0, at.Sub(now))
	}
	return time.Second << attempt
}

func wait(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
	return ctx.Err()
}
