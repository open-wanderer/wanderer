package hooks

import (
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func TestSetTrailCompletedAt(t *testing.T) {
	now := time.Date(2026, time.August, 10, 12, 30, 0, 0, time.FixedZone("CEST", 2*60*60))

	t.Run("sets timestamp when trail becomes completed", func(t *testing.T) {
		record := core.NewRecord(core.NewBaseCollection("trails"))
		record.Set("completed", true)

		setTrailCompletedAt(record, now)

		if got := record.GetDateTime("completed_at").Time(); !got.Equal(now.UTC()) {
			t.Fatalf("completed_at = %v, want %v", got, now.UTC())
		}
	})

	t.Run("preserves an existing completion timestamp", func(t *testing.T) {
		existing := time.Date(2025, time.June, 14, 0, 0, 0, 0, time.UTC)
		record := core.NewRecord(core.NewBaseCollection("trails"))
		record.Set("completed", true)
		record.Set("completed_at", existing)

		setTrailCompletedAt(record, now)

		if got := record.GetDateTime("completed_at").Time(); !got.Equal(existing) {
			t.Fatalf("completed_at = %v, want %v", got, existing)
		}
	})

	t.Run("clears timestamp when trail is no longer completed", func(t *testing.T) {
		record := core.NewRecord(core.NewBaseCollection("trails"))
		record.Set("completed", false)
		record.Set("completed_at", now)

		setTrailCompletedAt(record, now)

		if !record.GetDateTime("completed_at").IsZero() {
			t.Fatalf("completed_at = %v, want zero value", record.Get("completed_at"))
		}
	})
}
