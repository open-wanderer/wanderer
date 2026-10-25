package util

import (
	"context"
	"testing"
)

func TestRateLimitBucket(t *testing.T) {
	t.Run("defaults to system", func(t *testing.T) {
		if got := RateLimitBucket(context.Background()); got != "system" {
			t.Fatalf("got %q, want system", got)
		}
	})

	t.Run("uses the actor value", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "actor", "actor:abc")
		if got := RateLimitBucket(ctx); got != "actor:abc" {
			t.Fatalf("got %q, want actor:abc", got)
		}
	})

	t.Run("override wins over the actor value", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "actor", "actor:abc")
		ctx = WithRateLimitBucket(ctx, "inbox-actor:https://x/y")
		if got := RateLimitBucket(ctx); got != "inbox-actor:https://x/y" {
			t.Fatalf("got %q, want the override", got)
		}
	})
}

func TestGetSafeActorContextActorValueUntouchedByOverride(t *testing.T) {
	ctx := context.WithValue(context.Background(), "actor", "actor:abc")
	ctx = WithRateLimitBucket(ctx, "inbox-lifecycle:abc")
	if got, _ := ctx.Value("actor").(string); got != "actor:abc" {
		t.Fatalf("actor value = %q, want actor:abc (signing key selection must not change)", got)
	}
}
