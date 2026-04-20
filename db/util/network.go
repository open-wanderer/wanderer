package util

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

type RateLimiter struct {
	mu       sync.RWMutex
	requests map[string][]time.Time
	maxReqs  int
	window   time.Duration
}

var ErrRateLimited = fmt.Errorf("rate limit exceeded for origin")

func NewRateLimiter(maxReqs int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		maxReqs:  maxReqs,
		window:   window,
	}
}

func (rl *RateLimiter) CheckRateLimit(origin string) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	threshold := now.Add(-rl.window)

	timestamps := rl.requests[origin]

	w := 0
	for _, t := range timestamps {
		if t.After(threshold) {
			timestamps[w] = t
			w++
		}
	}
	rl.requests[origin] = timestamps[:w]

	if len(rl.requests[origin]) >= rl.maxReqs {
		return ErrRateLimited
	}

	rl.requests[origin] = append(rl.requests[origin], now)
	return nil
}

var ActivityPubRateLimiter = NewRateLimiter(30, time.Minute)

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
	if ip.IsLoopback() {
		return true
	}

	if ip.IsPrivate() {
		return true
	}

	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	if ip.IsMulticast() {
		return true
	}

	if ip.IsUnspecified() {
		return true
	}

	return false
}

func SafeHTTPClient() *http.Client {
	dialer := &net.Dialer{Timeout: 30 * time.Second}

	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, err
				}

				actorID, _ := ctx.Value("actorID").(string)
				if actorID == "" {
					actorID = "system"
				}

				limitKey := fmt.Sprintf("%s:%s", actorID, host)

				if err := ActivityPubRateLimiter.CheckRateLimit(limitKey); err != nil {
					return nil, err
				}

				ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
				if err != nil {
					return nil, err
				}
				for _, ip := range ips {
					if isPrivateOrReservedIP(ip) {
						return nil, fmt.Errorf("SSRF blocked: %s", ip)
					}
				}

				// Use the first validated IP to prevent DNS Rebinding
				return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
			},
		},
	}
}
