package util

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
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
	DefaultPluginMediaMaxBytes int64 = 50 << 20
	// DefaultPluginMaxMediaItemsPerEntity caps photos per trail or per waypoint
	// individually so trail photos cannot starve waypoint photos out of the
	// shared import byte budget.
	DefaultPluginMaxMediaItemsPerEntity       = 20
	DefaultPluginMaxImportMediaBytes    int64 = 500 << 20
)

type SafeFetchResult struct {
	Body        []byte
	ContentType string
	FinalURL    string
}

type PluginMediaHTTPStatusError struct {
	StatusCode int
}

func (e PluginMediaHTTPStatusError) Error() string {
	return fmt.Sprintf("plugin media request returned HTTP status %d", e.StatusCode)
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
		// safeurl's built-in blocklist covers NAT64, 6to4 and Teredo, but not the
		// deprecated IPv4-compatible form (RFC 4291), which embeds an IPv4 the
		// same way: ::7f00:1 is 127.0.0.1.
		SetBlockedIPsCIDR("::/96").
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

func ValidatePluginMediaStatus(statusCode int) error {
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return PluginMediaHTTPStatusError{StatusCode: statusCode}
	}
	return nil
}

func IsRetryablePluginMediaError(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}

	var statusErr PluginMediaHTTPStatusError
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
	// The literal address and the IPv4 it may embed are both subject to the policy.
	for _, candidate := range policyCandidates(addr) {
		if isReservedAddr(candidate) {
			return false
		}
		if candidate.IsPrivate() && !allowPrivate {
			return false
		}
	}
	return true
}

// policyCandidates returns every address that must clear the IP policy before a
// dial is allowed: the literal address, plus any IPv4 it embeds via an IPv6
// transition mechanism. Both are unmapped, since netip's IsPrivate and friends
// do not see through the 4-in-6 form.
func policyCandidates(addr netip.Addr) []netip.Addr {
	return []netip.Addr{addr.Unmap(), unwrapTransitionIP(addr)}
}

// isReservedAddr reports whether addr belongs to a range that must never be
// dialled, regardless of any allow-private escape hatch.
func isReservedAddr(addr netip.Addr) bool {
	if addr.Is4In6() {
		addr = addr.Unmap()
	}
	if addr.IsLoopback() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() ||
		addr.IsMulticast() || addr.IsUnspecified() {
		return true
	}
	return isSpecialPurposeIP(addr)
}

func isSpecialPurposeIP(addr netip.Addr) bool {
	for _, prefix := range specialPurposePrefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

// unwrapTransitionIP returns the IPv4 address embedded in an IPv6 transition
// address, or addr unchanged when nothing is embedded. The transition families
// themselves are also listed in specialPurposePrefixes; unwrapping additionally
// covers the network-specific NAT64 prefixes (RFC 6052 permits /32 through /64),
// which cannot be enumerated.
func unwrapTransitionIP(addr netip.Addr) netip.Addr {
	addr = addr.Unmap()
	if !addr.Is6() {
		return addr
	}
	b := addr.As16()

	switch {
	// NAT64 well-known prefix 64:ff9b::/96 (RFC 6052)
	case nat64WellKnown.Contains(addr):
		return netip.AddrFrom4([4]byte{b[12], b[13], b[14], b[15]})
	// 6to4 2002::/16 (RFC 3056) — IPv4 sits in bytes 2..5
	case sixToFour.Contains(addr):
		return netip.AddrFrom4([4]byte{b[2], b[3], b[4], b[5]})
	// Teredo 2001::/32 (RFC 4380) — client IPv4 is the last 4 bytes, XOR 0xff
	case teredo.Contains(addr):
		return netip.AddrFrom4([4]byte{b[12] ^ 0xff, b[13] ^ 0xff, b[14] ^ 0xff, b[15] ^ 0xff})
	// IPv4-compatible ::a.b.c.d (deprecated by RFC 4291)
	case ipv4Compatible.Contains(addr) && !addr.IsUnspecified() && !addr.IsLoopback():
		return netip.AddrFrom4([4]byte{b[12], b[13], b[14], b[15]})
	}
	return addr
}

var (
	nat64WellKnown = netip.MustParsePrefix("64:ff9b::/96")
	sixToFour      = netip.MustParsePrefix("2002::/16")
	teredo         = netip.MustParsePrefix("2001::/32")
	ipv4Compatible = netip.MustParsePrefix("::/96")
)

var specialPurposePrefixes = mustPrefixes(
	"0.0.0.0/8",
	"100.64.0.0/10",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"192.0.0.0/24",
	"192.0.2.0/24",
	"192.88.99.0/24",
	"198.18.0.0/15",
	"198.51.100.0/24",
	"203.0.113.0/24",
	"224.0.0.0/4",
	"240.0.0.0/4",
	"::/128",
	"::1/128",
	"::/96",
	"64:ff9b::/96",
	"64:ff9b:1::/48",
	"100::/64",
	"2001::/32",
	"2001:db8::/32",
	"2002::/16",
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
