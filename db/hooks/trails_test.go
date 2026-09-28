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

	t.Run("clears timestamp when trail becomes incomplete", func(t *testing.T) {
		record := persistedTrail(t, true, now)
		record.Set("completed", false)

		setTrailCompletedAt(record, now)

		if !record.GetDateTime("completed_at").IsZero() {
			t.Fatalf("completed_at = %v, want zero value", record.Get("completed_at"))
		}
	})

	t.Run("clears timestamp when incomplete state is unchanged", func(t *testing.T) {
		record := persistedTrail(t, false, now)

		setTrailCompletedAt(record, now.Add(time.Hour))

		if !record.GetDateTime("completed_at").IsZero() {
			t.Fatalf("completed_at = %v, want zero value", record.Get("completed_at"))
		}
	})
}

func persistedTrail(t *testing.T, completed bool, completedAt time.Time) *core.Record {
	t.Helper()

	collection := core.NewBaseCollection("trails")
	collection.Fields.Add(
		&core.BoolField{Name: "completed"},
		&core.DateField{Name: "completed_at"},
	)
	record := core.NewRecord(collection)
	record.Id = "trail000000001"
	record.Set("completed", completed)
	record.Set("completed_at", completedAt)
	if err := record.PostScan(); err != nil {
		t.Fatalf("failed to persist test trail state: %v", err)
	}

	return record
}
