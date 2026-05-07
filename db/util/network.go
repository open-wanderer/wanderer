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
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

var ErrRateLimited = fmt.Errorf("rate limit exceeded for origin")

type RateLimiter struct {
	mu       sync.RWMutex
	requests map[string][]time.Time
	maxReqs  int
	window   time.Duration
	key      []byte
}

func NewRateLimiter(maxReqs int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		maxReqs:  maxReqs,
		window:   window,
		key:      make([]byte, 32),
	}
	rand.Read(rl.key)

	// Background worker: Cleans up memory and rotates keys
	go rl.maintenanceWorker()
	return rl
}

func (rl *RateLimiter) maintenanceWorker() {
	ticker := time.NewTicker(rl.window * 2)
	for range ticker.C {
		rl.mu.Lock()

		newKey := make([]byte, 32)
		rand.Read(newKey)
		rl.key = newKey

		rl.requests = make(map[string][]time.Time)

		rl.mu.Unlock()
	}
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
