package util

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type RateLimiter struct {
	mu       sync.RWMutex
	requests map[string][]time.Time
	maxReqs  int
	window   time.Duration
	salt     string
}

var ErrRateLimited = fmt.Errorf("rate limit exceeded for origin")

func NewRateLimiter(maxReqs int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		maxReqs:  maxReqs,
		window:   window,
		salt:     fmt.Sprintf("%d", time.Now().UnixNano()),
	}

	go rl.cleanupWorker()
	return rl
}

func (rl *RateLimiter) CheckRateLimit(identifier string, host string) error {
	key := rl.hashIdentifier(identifier + ":" + host)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	threshold := now.Add(-rl.window)

	var active []time.Time
	for _, t := range rl.requests[key] {
		if t.After(threshold) {
			active = append(active, t)
		}
	}

	if len(active) >= rl.maxReqs {
		rl.requests[key] = active
		return ErrRateLimited
	}

	rl.requests[key] = append(active, now)
	return nil
}

func (rl *RateLimiter) cleanupWorker() {
	ticker := time.NewTicker(rl.window * 2)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, timestamps := range rl.requests {
			// If the newest timestamp is older than the window, delete the whole key
			if len(timestamps) == 0 || now.Sub(timestamps[len(timestamps)-1]) > rl.window {
				delete(rl.requests, key)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) hashIdentifier(id string) string {
	hash := sha256.Sum256([]byte(id + rl.salt))
	return fmt.Sprintf("%x", hash)
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
				host, port, _ := net.SplitHostPort(addr)

				identifier, _ := ctx.Value("actor").(string)
				if identifier == "" {
					identifier = "system"
				}

				if err := ActivityPubRateLimiter.CheckRateLimit(identifier, host); err != nil {
					return nil, err
				}

				ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
				if err != nil || len(ips) == 0 {
					return nil, fmt.Errorf("failed to resolve: %w", err)
				}

				for _, ip := range ips {
					if isPrivateOrReservedIP(ip) {
						return nil, fmt.Errorf("SSRF blocked: %s", ip)
					}
				}

				// Standard practice: Dial the first resolved IP to prevent TOCTOU/Rebinding
				return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
			},
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
