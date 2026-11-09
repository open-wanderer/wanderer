package routes

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func TestRemoteSyncPrunesMissingOriginChildrenAndTheirOrphanedPhotos(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	for _, target := range []string{"waypoint", "summit_log"} {
		t.Run(target, func(t *testing.T) {
			f := newRemoteChildCleanupFixture(t, target)
			kind := map[string]string{"waypoint": "waypoint", "summit_log": "summit-log"}[target]
			removed := f.child("https://a.example/api/v1/"+kind+"/removed", f.remoteActor.Id, f.trail.Id)
			kept := f.child("https://a.example/api/v1/"+kind+"/kept", f.remoteActor.Id, f.trail.Id)
			local := f.child("https://b.example/api/v1/"+kind+"/local", f.localActor.Id, f.trail.Id)
			foreign := f.child("https://a.example:8443/api/v1/"+kind+"/foreign", f.remoteActor.Id, f.trail.Id)
			legacyLocal := f.child("https://a.example/api/v1/"+kind+"/legacy-local", f.localActor.Id, f.trail.Id)
			otherTrail := f.save(f.trails, map[string]any{"author": f.remoteActor.Id})
			other := f.child("https://a.example/api/v1/"+kind+"/other", f.remoteActor.Id, otherTrail.Id)
			orphan := f.photo(removed, "orphan.jpg")
			shared := f.photo(removed, "shared.jpg")
			keptPhoto := f.photo(kept, "kept.jpg")
			f.save(f.trailLinks, map[string]any{"trail": f.trail.Id, "asset": shared.Id})
			f.assertFile(orphan, true)
			raw := map[string]any{"id": "kept", "iri": kept.GetString("iri"), "name": "Updated", "text": "Updated"}
			if err := f.app.RunInTransaction(func(app core.App) error { return f.syncPhotoSnapshot(app, []any{raw}) }); err != nil {
				t.Fatal(err)
			}
			if _, err := f.app.FindRecordById(f.children, removed.Id); !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("missing remote child survived: %v", err)
			}
			for _, record := range []*core.Record{kept, local, foreign, legacyLocal, other} {
				if _, err := f.app.FindRecordById(f.children, record.Id); err != nil {
					t.Fatalf("independent or listed child deleted: %s: %v", record.Id, err)
				}
			}
			if _, err := f.app.FindRecordById(f.assets, orphan.Id); !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("orphaned photo survived: %v", err)
			}
			f.assertFile(orphan, false)
			for _, asset := range []*core.Record{shared, keptPhoto} {
				if _, err := f.app.FindRecordById(f.assets, asset.Id); err != nil {
					t.Fatalf("linked photo deleted: %s: %v", asset.Id, err)
				}
				f.assertFile(asset, true)
			}
			links, err := f.app.CountRecords(f.links.Id, dbx.HashExp{target: removed.Id})
			if err != nil || links != 0 {
				t.Fatalf("removed child's links survived: links=%d error=%v", links, err)
			}
			// A present, empty expand also retracts the last origin-owned
			// child. Local and third-instance entries remain independent.
			if err := f.app.RunInTransaction(func(app core.App) error { return f.syncPhotoSnapshot(app, []any{}) }); err != nil {
				t.Fatal(err)
			}
			if _, err := f.app.FindRecordById(f.children, kept.Id); !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("empty expand did not retract last origin child: %v", err)
			}
			if _, err := f.app.FindRecordById(f.assets, keptPhoto.Id); !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("last child's orphan photo survived: %v", err)
			}
			f.assertFile(keptPhoto, false)
		})
	}
}

func TestRemoteChildCleanupRollsBackWithTheSync(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	f := newRemoteChildCleanupFixture(t, "waypoint")
	child := f.child("https://a.example/api/v1/waypoint/removed", f.remoteActor.Id, f.trail.Id)
	photo := f.photo(child, "retained.jpg")
	abort := errors.New("later sync step failed")
	err := f.app.RunInTransaction(func(app core.App) error {
		if err := f.syncPhotoSnapshot(app, []any{}); err != nil {
			return err
		}
		if _, err := app.FindRecordById(f.children, child.Id); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("sync did not delete the child inside its transaction: %v", err)
		}
		return abort
	})
	if !errors.Is(err, abort) {
		t.Fatal(err)
	}
	if _, err := f.app.FindRecordById(f.children, child.Id); err != nil {
		t.Fatalf("rolled-back child deletion persisted: %v", err)
	}
	if _, err := f.app.FindRecordById(f.assets, photo.Id); err != nil {
		t.Fatalf("rolled-back orphan cleanup persisted: %v", err)
	}
	f.assertFile(photo, true)
}

func TestRemoteSaturatedChildExpandKeepsCopiesBeyondPocketBaseLimit(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	for _, target := range []string{"waypoint", "summit_log"} {
		t.Run(target, func(t *testing.T) {
			f := newRemoteChildCleanupFixture(t, target)
			kind := map[string]string{"waypoint": "waypoint", "summit_log": "summit-log"}[target]
			unlisted := f.child("https://a.example/api/v1/"+kind+"/last", f.remoteActor.Id, f.trail.Id)
			photo := f.photo(unlisted, "last.jpg")
			// This is the first capped page of a trail with at least 1001
			// children; the omitted child's photo must remain available.
			raw := make([]any, remoteChildExpandLimit)
			for i := range raw {
				raw[i] = map[string]any{"id": fmt.Sprintf("child-%d", i), "name": "Present", "text": "Present"}
			}
			if err := f.app.RunInTransaction(func(app core.App) error { return f.syncPhotoSnapshot(app, raw) }); err != nil {
				t.Fatal(err)
			}
			count, err := f.app.CountRecords(f.children.Id, dbx.HashExp{"trail": f.trail.Id})
			if err != nil || count != remoteChildExpandLimit+1 {
				t.Fatalf("saturated child expand lost cached copies: count=%d error=%v", count, err)
			}
			if _, err := f.app.FindRecordById(f.assets, photo.Id); err != nil {
				t.Fatalf("saturated child expand deleted its unlisted child's photo: %v", err)
			}
			f.assertFile(photo, true)
		})
	}
}

func TestRemoteSyncMissingOrMalformedChildExpandDoesNotPrune(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	for _, target := range []string{"waypoint", "summit_log"} {
		t.Run(target, func(t *testing.T) {
			f := newRemoteChildCleanupFixture(t, target)
			kind := map[string]string{"waypoint": "waypoint", "summit_log": "summit-log"}[target]
			child := f.child("https://a.example/api/v1/"+kind+"/retained", f.remoteActor.Id, f.trail.Id)
			photo := f.photo(child, "retained.jpg")
			if err := f.syncPhotoSnapshot(f.app, []any{map[string]any{"name": "no identity"}}); err != nil {
				t.Fatal(err)
			}
			previousClient := newRemoteSyncHTTPClient
			t.Cleanup(func() { newRemoteSyncHTTPClient = previousClient })
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: richTextSyncTransport{map[string]any{"id": "trail", "public": true}}}
			}
			if _, err := performFullSync(f.app, t.Context(), nil, f.trail); err != nil {
				t.Fatal(err)
			}
			if _, err := f.app.FindRecordById(f.children, child.Id); err != nil {
				t.Fatalf("non-authoritative child response deleted data: %v", err)
			}
			if _, err := f.app.FindRecordById(f.assets, photo.Id); err != nil {
				t.Fatal(err)
			}
			f.assertFile(photo, true)
		})
	}
}

// The current sync coordinator batches child writes and prunes only once the
// origin document has supplied a complete list.
func (f *remoteChildCleanupFixture) syncPhotoSnapshot(app core.App, raw []any) error {
	previous := newRemoteSyncHTTPClient
	defer func() { newRemoteSyncHTTPClient = previous }()
	relation := map[string]string{"waypoint": "waypoints_via_trail", "summit_log": "summit_logs_via_trail"}[f.target]
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: richTextSyncTransport{map[string]any{"id": "trail", "public": true, "expand": map[string]any{relation: raw}}}}
	}
	_, err := performFullSync(app, f.t.Context(), nil, f.trail)
	return err
}
