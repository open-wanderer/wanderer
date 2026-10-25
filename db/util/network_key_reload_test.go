package util

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestKnownKeyReloadWindowSurvivesMaintenance(t *testing.T) {
	limiter := newRateLimiter(1, time.Minute, true)
	if err := limiter.CheckRateLimit("actor", "remote.example"); err != nil {
		t.Fatal(err)
	}
	limiter.maintain(time.Now())
	if err := limiter.CheckRateLimit("actor", "remote.example"); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("recent claim removed: %v", err)
	}
	limiter.maintain(time.Now().Add(time.Minute))
	if err := limiter.CheckRateLimit("actor", "remote.example"); err != nil {
		t.Fatalf("expired claim retained: %v", err)
	}
}

func TestDialBudgetKeepsSigningActor(t *testing.T) {
	previous := ActivityPubRateLimiter
	ActivityPubRateLimiter = NewRateLimiter(1, time.Minute)
	t.Cleanup(func() { ActivityPubRateLimiter = previous })
	ctx := context.WithValue(context.Background(), "actor", "actor:reader")
	if err := CheckDialRateLimit(ctx, "remote.example"); err != nil {
		t.Fatal(err)
	}
	own := WithRateLimitBucket(ctx, "known-actor:sender")
	if own.Value("actor") != "actor:reader" {
		t.Fatal("dial budget changed signing actor")
	}
	if err := CheckDialRateLimit(own, "remote.example"); err != nil {
		t.Fatalf("separate budget: %v", err)
	}
	if err := CheckDialRateLimit(ctx, "remote.example"); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("recipient budget reset: %v", err)
	}
}

func TestHTTPClientUsesSelectedActorBudget(t *testing.T) {
	for _, recipientRequests := range []int{1, 3} {
		t.Run(fmt.Sprintf("recipient-requests-%d", recipientRequests), func(t *testing.T) {
			previous := ActivityPubRateLimiter
			ActivityPubRateLimiter = NewRateLimiter(3, time.Minute)
			t.Cleanup(func() { ActivityPubRateLimiter = previous })
			ctx := context.WithValue(context.Background(), "actor", "actor:reader")
			ctx, cancel := context.WithTimeout(ctx, time.Second)
			t.Cleanup(cancel)
			const host = "192.0.2.1"
			client := SafeHTTPClient()
			t.Cleanup(client.CloseIdleConnections)
			request := func(ctx context.Context) error {
				t.Helper()
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+host+"/actor", nil)
				if err != nil {
					t.Fatal(err)
				}
				response, err := client.Do(req)
				if response != nil {
					response.Body.Close()
				}
				return err
			}
			assertAddressChecked := func(ctx context.Context) {
				t.Helper()
				err := request(ctx)
				if err == nil || errors.Is(err, ErrRateLimited) || !strings.Contains(err.Error(), "blocked") {
					t.Fatalf("request did not reach the address check: %v", err)
				}
			}
			for i := 0; i < recipientRequests; i++ {
				assertAddressChecked(ctx)
			}
			selected := WithRateLimitBucket(ctx, "known-actor:sender")
			for i := 0; i < 3; i++ {
				assertAddressChecked(selected)
			}
			if err := request(selected); !errors.Is(err, ErrRateLimited) {
				t.Fatalf("selected budget did not retain its limit: %v", err)
			}
			for i := recipientRequests; i < 3; i++ {
				assertAddressChecked(ctx)
			}
			if err := request(ctx); !errors.Is(err, ErrRateLimited) {
				t.Fatalf("recipient budget changed: %v", err)
			}
		})
	}
}
