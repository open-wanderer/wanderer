package util

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"time"

	"github.com/doyensec/safeurl"
)

const (
	DefaultPluginMediaMaxBytes       int64 = 50 << 20
	DefaultPluginMaxImportMediaItems       = 20
	DefaultPluginMaxImportMediaBytes int64 = 200 << 20
)

type SafeFetchResult struct {
	Body        []byte
	ContentType string
	FinalURL    string
}

type ConnectorHTTPPolicy struct {
	BaseURL      string
	AllowPrivate bool
	TLSMode      string
	TLSCABundle  []byte
}

func FetchPublicURL(ctx context.Context, rawURL string, maxBytes int64) (*SafeFetchResult, error) {
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
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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
