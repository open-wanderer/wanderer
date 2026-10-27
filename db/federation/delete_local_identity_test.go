package federation

import (
	"testing"
)

func TestLocalActorCannotAnnounceDeletionOfRemoteIdentity(t *testing.T) {
	f := setupAddressingTestApp(t)
	local := f.record(t, "activitypub_actors", map[string]any{"is_local": true, "iri": "https://local.example/users/alice"})
	trail := f.record(t, "trails", map[string]any{"author": local.Id, "iri": "https://remote.example/api/v1/trail/imported", "public": true})
	log := f.record(t, "summit_logs", map[string]any{"author": local.Id, "trail": trail.Id, "iri": "https://remote.example/api/v1/summit-log/imported"})
	for _, operation := range []func() error{
		func() error { return CreateTrailDeleteActivity(f.app, trail, DeleteAudience{}) },
		func() error { return CreateSummitLogDeleteActivity(f.app, log) },
	} {
		if err := operation(); err != nil {
			t.Fatal(err)
		}
	}
	count, err := f.app.CountRecords("activitypub_activities")
	if err != nil || count != 0 {
		t.Fatalf("signed remote identity deletion: count=%d error=%v", count, err)
	}
}
