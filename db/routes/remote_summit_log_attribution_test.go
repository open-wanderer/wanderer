package routes

import (
	"testing"
	"time"
)

func TestSummitLogAttributionRepairIsLimitedToRemoteTrailAuthorFallback(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteSyncFixture(t)
	trail := f.save(f.trails, map[string]any{"author": f.remoteActor.Id, "iri": "https://remote.example/api/v1/trail/parent"})
	corrected := f.save(f.actors, map[string]any{"iri": "https://remote.example/users/carol", "last_fetched": time.Now()})
	other := f.save(f.actors, map[string]any{"iri": "https://remote.example/users/dave", "last_fetched": time.Now()})
	third := f.save(f.actors, map[string]any{"iri": "https://third.example/users/carol", "last_fetched": time.Now()})
	for _, tc := range []struct {
		name, stored, proposed, parent string
		allowed                        bool
	}{
		{"legacy-fallback", f.remoteActor.Id, corrected.Id, trail.Id, true},
		{"another-owner", other.Id, corrected.Id, trail.Id, false},
		{"local-owner", f.localActor.Id, corrected.Id, trail.Id, false},
		{"wrong-author-origin", f.remoteActor.Id, third.Id, trail.Id, false},
		{"wrong-parent", f.remoteActor.Id, corrected.Id, "different-parent", false},
	} {
		log := f.save(f.logs, map[string]any{"iri": "https://remote.example/api/v1/summit-log/" + tc.name, "author": tc.stored, "trail": trail.Id, "text": "Original"})
		if got := remoteSyncRecordCanUpdate(f.app, log, tc.proposed, tc.parent); got != tc.allowed {
			t.Fatalf("%s: attribution repair allowed=%t want=%t", tc.name, got, tc.allowed)
		}
	}
}
