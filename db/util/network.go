package util

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

var ErrRateLimited = fmt.Errorf("rate limit exceeded for origin")

type RateLimiter struct {
	mu           sync.RWMutex
	requests     map[string][]time.Time
	maxReqs      int
	window       time.Duration
	key          []byte
	retainWindow bool
}

func NewRateLimiter(maxReqs int, window time.Duration) *RateLimiter {
	return newRateLimiter(maxReqs, window, false)
}

func newRateLimiter(maxReqs int, window time.Duration, retainWindow bool) *RateLimiter {
	rl := &RateLimiter{
		requests:     make(map[string][]time.Time),
		maxReqs:      maxReqs,
		window:       window,
		key:          make([]byte, 32),
		retainWindow: retainWindow,
	}
	rand.Read(rl.key)

	// Background worker: Cleans up memory and rotates keys
	go rl.maintenanceWorker()
	return rl
}

func (rl *RateLimiter) maintenanceWorker() {
	ticker := time.NewTicker(rl.window * 2)
	for now := range ticker.C {
		rl.maintain(now)
	}
}

func (rl *RateLimiter) maintain(now time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.retainWindow {
		threshold := now.Add(-rl.window)
		for key, timestamps := range rl.requests {
			remaining := timestamps[:0]
			for _, timestamp := range timestamps {
				if timestamp.After(threshold) {
					remaining = append(remaining, timestamp)
				}
			}
			if len(remaining) == 0 {
				delete(rl.requests, key)
			} else {
				rl.requests[key] = remaining
			}
		}
		return
	}

	newKey := make([]byte, 32)
	rand.Read(newKey)
	rl.key = newKey
	rl.requests = make(map[string][]time.Time)
}

func (rl *RateLimiter) CheckRateLimit(identifier string, host string) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	h := hmac.New(sha256.New, rl.key)
	h.Write([]byte(identifier + ":" + host))
	key := hex.EncodeToString(h.Sum(nil))

	now := time.Now()
	threshold := now.Add(-rl.window)

	timestamps := rl.requests[key]
	w := 0
	for _, t := range timestamps {
		if t.After(threshold) {
			timestamps[w] = t
			w++
		}
	}
	timestamps = timestamps[:w]

	if len(timestamps) >= rl.maxReqs {
		rl.requests[key] = timestamps
		return ErrRateLimited
	}

	rl.requests[key] = append(timestamps, now)
	return nil
}

var ActivityPubRateLimiter = NewRateLimiter(30, time.Minute)

// KnownActorKeyReloads admits one refresh per actor and minute. Maintenance
// removes expired claims while preserving the full window of recent claims.
var KnownActorKeyReloads = newRateLimiter(1, time.Minute, true)

type rateLimitBucketKey struct{}

// WithRateLimitBucket changes the dial budget without changing the actor
// whose key signs the request.
func WithRateLimitBucket(ctx context.Context, bucket string) context.Context {
	return context.WithValue(ctx, rateLimitBucketKey{}, bucket)
}

// RateLimitBucket returns the dial budget selected by ctx: the
// WithRateLimitBucket override, else the "actor" value, else "system".
func RateLimitBucket(ctx context.Context) string {
	identifier, _ := ctx.Value(rateLimitBucketKey{}).(string)
	if identifier == "" {
		identifier, _ = ctx.Value("actor").(string)
	}
	if identifier == "" {
		identifier = "system"
	}
	return identifier
}

// CheckDialRateLimit applies the dial budget selected by ctx.
func CheckDialRateLimit(ctx context.Context, host string) error {
	return ActivityPubRateLimiter.CheckRateLimit(RateLimitBucket(ctx), host)
}

type safeTransport struct {
	transport http.RoundTripper
}

func (t *safeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	host := req.URL.Hostname()
	if host == "" {
		return nil, fmt.Errorf("invalid host in request")
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve host: %w", err)
	}

	for _, ip := range ips {
		if isPrivateOrReservedIP(ip) {
			return nil, fmt.Errorf("request to private/reserved IP address blocked: %s", ip)
		}
	}

	return t.transport.RoundTrip(req)
}

func isPrivateOrReservedIP(ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		// Fail closed: an address we cannot parse is one we cannot vet.
		return true
	}

	// Both the literal address and any IPv4 it embeds via an IPv6 transition
	// mechanism (NAT64, 6to4, Teredo, IPv4-compatible) must clear the policy.
	for _, candidate := range policyCandidates(addr) {
		if isReservedAddr(candidate) || candidate.IsPrivate() {
			return true
		}
	}

	return false
}

// publicDialContext resolves the host once, refuses private or reserved addresses,
// and dials only the vetted addresses, alternating families without another lookup.
func publicDialContext(dialer *net.Dialer, rateLimit bool) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, _ := net.SplitHostPort(addr)

		if rateLimit {
			if err := CheckDialRateLimit(ctx, host); err != nil {
				return nil, err
			}
		}

		// The timeout covers resolution and all address attempts together.
		dialCtx := ctx
		if dialer.Timeout > 0 {
			var cancel context.CancelFunc
			dialCtx, cancel = context.WithTimeout(ctx, dialer.Timeout)
			defer cancel()
		}
		ips, err := net.DefaultResolver.LookupIP(dialCtx, "ip", host)
		if err != nil || len(ips) == 0 {
			return nil, fmt.Errorf("failed to resolve: %w", err)
		}

		for _, ip := range ips {
			if isPrivateOrReservedIP(ip) {
				return nil, fmt.Errorf("SSRF blocked: %s", ip)
			}
		}

		ips = interleaveAddressFamilies(ips)
		// Try only the vetted addresses, alternating families. A connection attempt
		// gets its share of the remaining time, with a two-second minimum when
		// the overall deadline allows it.
		deadline, bounded := dialCtx.Deadline()
		var firstErr error
		for i, ip := range ips {
			if err := dialCtx.Err(); err != nil {
				if firstErr == nil {
					firstErr = err
				}
				break
			}
			attemptCtx := dialCtx
			cancel := context.CancelFunc(func() {})
			if bounded {
				remaining := time.Until(deadline)
				if remaining <= 0 {
					if firstErr == nil {
						firstErr = context.DeadlineExceeded
					}
					break
				}
				share := max(remaining/time.Duration(len(ips)-i), min(remaining, 2*time.Second))
				attemptCtx, cancel = context.WithTimeout(dialCtx, share)
			}
			conn, err := dialer.DialContext(attemptCtx, network, net.JoinHostPort(ip.String(), port))
			cancel()
			if err == nil {
				return conn, nil
			}
			if firstErr == nil {
				firstErr = err
			}
		}
		return nil, firstErr
	}
}

func SafeHTTPClient() *http.Client {
	return safeHTTPClientWithTimeouts(30*time.Second, 10*time.Second, 10*time.Second)
}

func safeHTTPClientWithTimeouts(total, headers, handshake time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: 30 * time.Second}

	return &http.Client{
		Timeout: total,
		Transport: &http.Transport{
			ResponseHeaderTimeout: headers,
			TLSHandshakeTimeout:   handshake,
			DialContext:           publicDialContext(dialer, true),
		},
	}
}

// SafeDeliveryClient posts activities to remote inboxes. Inbox URLs come from
// remote actor documents, so only public addresses are dialed and redirects
// are not followed. It is not rate limited: fanout to one host is legitimate.
func SafeDeliveryClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: timeout}

	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: publicDialContext(dialer, false),
			// A custom DialContext requires HTTP/2 to be enabled explicitly.
			// Deliveries to one host share HTTP/2 connections; every new
			// connection still passes through publicDialContext.
			ForceAttemptHTTP2: true,
			MaxIdleConns:      100,
			// Keep one idle connection per parallel delivery to a host.
			MaxIdleConnsPerHost: 5,
			IdleConnTimeout:     90 * time.Second,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func GetSafeActorContext(r *http.Request, userActor *core.Record) (context.Context, error) {
	var identifier string

	if userActor != nil {
		identifier = "actor:" + userActor.Id
	} else if r != nil {
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		identifier = "anon:" + ip
	} else {
		return nil, errors.New("request or actor must be defined")
	}

	parentCtx := context.Background()
	if r != nil {
		parentCtx = r.Context()
	}
	return context.WithValue(parentCtx, "actor", identifier), nil
}

// interleaveAddressFamilies preserves order within each family and starts with
// the first family in the resolver answer.
func interleaveAddressFamilies(ips []net.IP) []net.IP {
	if len(ips) < 2 {
		return ips
	}
	var first, other []net.IP
	ipv4First := ips[0].To4() != nil
	for _, ip := range ips {
		if (ip.To4() != nil) == ipv4First {
			first = append(first, ip)
		} else {
			other = append(other, ip)
		}
	}
	ordered := make([]net.IP, 0, len(ips))
	for i := 0; i < len(first) || i < len(other); i++ {
		if i < len(first) {
			ordered = append(ordered, first[i])
		}
		if i < len(other) {
			ordered = append(ordered, other[i])
		}
	}
	return ordered
}
