package federation

import (
	"sort"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

// TestInstanceFollowerInboxes_ReturnsAcceptedFollowers verifies that
// instanceFollowerInboxes returns inbox URLs for all accepted followers
// of the local instance actor.
func TestInstanceFollowerInboxes_ReturnsAcceptedFollowers(t *testing.T) {
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	app := newInboxTestApp(t)

	// Create local instance actor with the IRI that instanceFollowerInboxes resolves
	instanceActor := createTestActor(t, app,
		"https://trails.example.com/api/v1/activitypub/instance",
		"instance", true)

	// Create two remote follower actors
	follower1 := createTestActor(t, app,
		"https://remote1.example.com/api/v1/activitypub/instance",
		"instance", false)
	follower2 := createTestActor(t, app,
		"https://remote2.example.com/api/v1/activitypub/instance",
		"instance", false)

	// Seed accepted follows: both remote actors follow the instance actor
	followsCol, err := app.FindCollectionByNameOrId("follows")
	if err != nil {
		t.Fatalf("find follows collection: %v", err)
	}

	for _, followerID := range []string{follower1.Id, follower2.Id} {
		r := core.NewRecord(followsCol)
		r.Set("follower", followerID)
		r.Set("followee", instanceActor.Id)
		r.Set("status", "accepted")
		if err := app.Save(r); err != nil {
			t.Fatalf("save follow record for %s: %v", followerID, err)
		}
	}

	inboxes, err := instanceFollowerInboxes(app)
	if err != nil {
		t.Fatalf("instanceFollowerInboxes returned error: %v", err)
	}

	want := []string{
		follower1.GetString("iri") + "/inbox",
		follower2.GetString("iri") + "/inbox",
	}

	// Sort both slices for deterministic comparison (JOIN order is non-deterministic)
	sort.Strings(inboxes)
	sort.Strings(want)

	if len(inboxes) != len(want) {
		t.Fatalf("got %d inboxes, want %d; got: %v", len(inboxes), len(want), inboxes)
	}
	for i := range want {
		if inboxes[i] != want[i] {
			t.Errorf("inbox[%d] = %q, want %q", i, inboxes[i], want[i])
		}
	}
}

// TestInstanceFollowerInboxes_NoInstanceActor verifies that instanceFollowerInboxes
// returns (nil, nil) when no instance actor record exists yet (startup-safe, D-02).
func TestInstanceFollowerInboxes_NoInstanceActor(t *testing.T) {
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	app := newInboxTestApp(t)
	// Intentionally do NOT create the instance actor

	inboxes, err := instanceFollowerInboxes(app)
	if err != nil {
		t.Fatalf("instanceFollowerInboxes returned unexpected error: %v", err)
	}
	if len(inboxes) != 0 {
		t.Errorf("expected empty slice, got %v", inboxes)
	}
}

// TestInstanceFollowerInboxes_OnlyAcceptedFollowers verifies that only accepted
// follows are included; pending follows are excluded.
func TestInstanceFollowerInboxes_OnlyAcceptedFollowers(t *testing.T) {
	t.Setenv("ORIGIN", "https://trails.example.com")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	app := newInboxTestApp(t)

	instanceActor := createTestActor(t, app,
		"https://trails.example.com/api/v1/activitypub/instance",
		"instance", true)

	acceptedFollower := createTestActor(t, app,
		"https://accepted.example.com/api/v1/activitypub/instance",
		"instance", false)
	pendingFollower := createTestActor(t, app,
		"https://pending.example.com/api/v1/activitypub/instance",
		"instance", false)

	followsCol, err := app.FindCollectionByNameOrId("follows")
	if err != nil {
		t.Fatalf("find follows collection: %v", err)
	}

	// One accepted follow
	accepted := core.NewRecord(followsCol)
	accepted.Set("follower", acceptedFollower.Id)
	accepted.Set("followee", instanceActor.Id)
	accepted.Set("status", "accepted")
	if err := app.Save(accepted); err != nil {
		t.Fatalf("save accepted follow: %v", err)
	}

	// One pending follow
	pending := core.NewRecord(followsCol)
	pending.Set("follower", pendingFollower.Id)
	pending.Set("followee", instanceActor.Id)
	pending.Set("status", "pending")
	if err := app.Save(pending); err != nil {
		t.Fatalf("save pending follow: %v", err)
	}

	inboxes, err := instanceFollowerInboxes(app)
	if err != nil {
		t.Fatalf("instanceFollowerInboxes returned error: %v", err)
	}

	if len(inboxes) != 1 {
		t.Fatalf("expected 1 inbox (accepted follower only), got %d: %v", len(inboxes), inboxes)
	}
	wantInbox := acceptedFollower.GetString("iri") + "/inbox"
	if inboxes[0] != wantInbox {
		t.Errorf("inbox = %q, want %q", inboxes[0], wantInbox)
	}
}
