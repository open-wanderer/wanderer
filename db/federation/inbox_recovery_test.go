package federation

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

func TestInboxDeliveryRecoveryDrainsAndPreservesSiblingDelivery(t *testing.T) {
	f := setupAddressingTestApp(t)
	sender := f.actor(t, "sender", true)
	local := f.actor(t, "local", true)
	remote := f.actor(t, "remote", false)

	trail := f.record(t, "trails", map[string]any{"name": "Local trail", "author": sender.Id, "public": true}, "trail")
	var attempts atomic.Int32
	f.app.OnRecordCreate("feed").BindFunc(func(e *core.RecordEvent) error {
		attempts.Add(1)
		panic("fixture stopped local feed creation")
	})
	activity := pub.CreateNew(pub.IRI("https://local.example/activity/create"), &pub.Object{ID: pub.IRI(trail.GetString("iri")), Type: pub.NoteType})
	activity.Actor = pub.IRI(sender.GetString("iri"))
	if err := PostActivity(f.app, sender, activity, []string{local.GetString("inbox"), remote.GetString("inbox")}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	if err := WaitForDeliveries(ctx); err != nil {
		t.Fatalf("deliveries did not drain: %v", err)
	}
	if attempts.Load() != 1 {
		t.Fatalf("local feed attempts = %d, want 1", attempts.Load())
	}
	if count := f.in.count("remote"); count != 1 {
		t.Fatalf("sibling inbox deliveries = %d, want 1", count)
	}
}
