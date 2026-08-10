package util

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/doyensec/safeurl"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

const (
	DefaultPluginMediaMaxBytes          int64 = 50 << 20
	DefaultPluginMaxMediaItemsPerEntity       = 20
	DefaultPluginMaxImportMediaBytes    int64 = 500 << 20
	DefaultPluginMaxPhotosPerTrail            = 20
	DefaultPluginMaxPhotosPerWaypoint         = 5
	DefaultPluginMaxPhotosPerSummitLog        = 20
)

type SafeFetchResult struct {
	Body        []byte
	ContentType string
	FinalURL    string
}

type HTTPStatusError struct {
	StatusCode int
	Message    string
}

func (e HTTPStatusError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("unexpected HTTP status: %d", e.StatusCode)
}

type ConnectorHTTPPolicy struct {
	BaseURL      string
	AllowPrivate bool
	TLSMode      string
	TLSCABundle  []byte
}

func FetchPublicURL(ctx context.Context, rawURL string, maxBytes int64) (*SafeFetchResult, error) {
	return FetchPublicURLWithHeaders(ctx, rawURL, maxBytes, nil)
}

func FetchPublicURLWithHeaders(ctx context.Context, rawURL string, maxBytes int64, headers map[string]string) (*SafeFetchResult, error) {
	if maxBytes <= 0 {
		maxBytes = DefaultPluginMediaMaxBytes
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid public URL")
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("public URL must not include credentials")
	}
	config := safeurl.GetConfigBuilder().
		SetTimeout(60*time.Second).
		SetAllowedSchemes("http", "https").
		SetAllowedPorts(80, 443).
		EnableIPv6(true).
		AllowSendingCredentials(false).
		SetCheckRedirect(publicMediaRedirectPolicy).
		Build()
	client := safeurl.Client(config)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	for key, value := range headers {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := ValidatePluginMediaStatus(resp.StatusCode); err != nil {
		return nil, err
	}

	body, err := ReadBoundedForPlugin(resp.Body, maxBytes)
	if err != nil {
		return nil, err
	}
	return &SafeFetchResult{
		Body:        body,
		ContentType: resp.Header.Get("Content-Type"),
		FinalURL:    resp.Request.URL.String(),
	}, nil
}

func FetchPublicFile(ctx context.Context, rawURL string, fallbackName string, maxBytes int64) (*filesystem.File, error) {
	fetched, err := FetchPublicURL(ctx, rawURL, maxBytes)
	if err != nil {
		return nil, err
	}
	return filesystem.NewFileFromBytes(
		fetched.Body,
		safeFetchedFileName(fallbackName, fetched.FinalURL, fetched.ContentType),
	)
}

func ValidatePluginMediaStatus(statusCode int) error {
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return HTTPStatusError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("plugin media request returned HTTP status %d", statusCode),
		}
	}
	return nil
}

func IsRetryablePluginMediaError(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}

	var statusErr HTTPStatusError
	if errors.As(err, &statusErr) {
		switch statusErr.StatusCode {
		case http.StatusRequestTimeout,
			http.StatusTooEarly,
			http.StatusTooManyRequests,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			return true
		default:
			return false
		}
	}

	// A missing DNS name and explicit request cancellation are permanent for
	// this import. Other transport-level failures are worth a bounded retry.
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) && dnsErr.IsNotFound {
		return false
	}
	var networkErr net.Error
	if errors.As(err, &networkErr) && (networkErr.Timeout() || networkErr.Temporary()) {
		return true
	}
	var operationErr *net.OpError
	return errors.As(err, &operationErr) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF)
}

func publicMediaRedirectPolicy(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return fmt.Errorf("too many redirects")
	}
	if req.URL.User != nil {
		return fmt.Errorf("redirect URL must not include credentials")
	}
	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		return fmt.Errorf("redirect scheme must be http or https")
	}
	if len(via) > 0 && via[len(via)-1].URL.Scheme == "https" && req.URL.Scheme == "http" {
		return fmt.Errorf("redirect downgrades https to http")
	}
	return nil
}

func ConnectorHTTPClient(policy ConnectorHTTPPolicy, checkRedirect func(req *http.Request, via []*http.Request) error) (*http.Client, error) {
	base, err := url.Parse(policy.BaseURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return nil, fmt.Errorf("invalid connector baseURL")
	}
	tlsConfig, err := connectorTLSConfig(policy.TLSMode, policy.TLSCABundle)
	if err != nil {
		return nil, err
	}
	dialer := &net.Dialer{Timeout: 30 * time.Second}
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
		DialContext: func(ctx context.Context, network string, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
			if err != nil || len(ips) == 0 {
				return nil, fmt.Errorf("failed to resolve connector host: %w", err)
			}
			var selected net.IP
			for _, ip := range ips {
				if connectorIPAllowed(ip, policy.AllowPrivate) {
					selected = ip
					break
				}
			}
			if selected == nil {
				return nil, fmt.Errorf("connector host resolved outside allowed IP policy")
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(selected.String(), port))
		},
	}
	return &http.Client{
		Timeout:       60 * time.Second,
		Transport:     transport,
		CheckRedirect: checkRedirect,
	}, nil
}

func connectorTLSConfig(mode string, caBundle []byte) (*tls.Config, error) {
	switch mode {
	case "", "system":
		return nil, nil
	case "customCA":
		roots, err := x509.SystemCertPool()
		if err != nil || roots == nil {
			roots = x509.NewCertPool()
		}
		if len(caBundle) == 0 || !roots.AppendCertsFromPEM(caBundle) {
			return nil, fmt.Errorf("connector customCA bundle is invalid")
		}
		return &tls.Config{RootCAs: roots}, nil
	default:
		return nil, fmt.Errorf("unsupported connector TLS mode %q", mode)
	}
}

func connectorIPAllowed(ip net.IP, allowPrivate bool) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	if addr.Is4In6() {
		addr = addr.Unmap()
	}
	if addr.IsLoopback() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() ||
		addr.IsMulticast() || addr.IsUnspecified() {
		return false
	}
	if isSpecialPurposeIP(addr) {
		return false
	}
	if addr.IsPrivate() {
		return allowPrivate
	}
	return true
}

func isSpecialPurposeIP(addr netip.Addr) bool {
	for _, prefix := range specialPurposePrefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

var specialPurposePrefixes = mustPrefixes(
	"0.0.0.0/8",
	"100.64.0.0/10",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"192.0.0.0/24",
	"192.0.2.0/24",
	"198.18.0.0/15",
	"198.51.100.0/24",
	"203.0.113.0/24",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"::/128",
	"::1/128",
	"64:ff9b::/96",
	"100::/64",
	"2001:db8::/32",
	"fe80::/10",
	"ff00::/8",
)

func mustPrefixes(values ...string) []netip.Prefix {
	prefixes := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			panic(err)
		}
		prefixes = append(prefixes, prefix)
	}
	return prefixes
}

func ReadBoundedForPlugin(reader io.Reader, maxBytes int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("response exceeds maximum size")
	}
	return body, nil
}

func safeFetchedFileName(fallbackName string, finalURL string, contentType string) string {
	candidates := []string{publicURLPathBase(finalURL), fallbackName}
	firstSafe := ""
	filename := ""
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || strings.Contains(candidate, "/") || strings.Contains(candidate, "\\") {
			continue
		}
		base := filepath.Base(candidate)
		if base == "." || base == ".." {
			continue
		}
		if firstSafe == "" {
			firstSafe = base
		}
		if ext := filepath.Ext(base); ext != "" && ext != "." {
			filename = base
			break
		}
	}
	if filename == "" {
		filename = firstSafe
	}
	if filename == "" {
		filename = filepath.Base(strings.TrimSpace(fallbackName))
	}
	if filename == "" || filename == "." || filename == ".." {
		filename = "download"
	}
	filename = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '-'
		default:
			return r
		}
	}, filename)
	if ext := filepath.Ext(filename); ext == "" || ext == "." {
		filename += extensionFromContentType(contentType)
	}
	return filename
}

func publicURLPathBase(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return filepath.Base(parsed.Path)
}

func extensionFromContentType(contentType string) string {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err != nil {
		mediaType = strings.TrimSpace(contentType)
	}
	if extensions, err := mime.ExtensionsByType(mediaType); err == nil && len(extensions) > 0 {
		return extensions[0]
	}
	return ".bin"
}
