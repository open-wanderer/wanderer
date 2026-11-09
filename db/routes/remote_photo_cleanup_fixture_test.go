package routes

import (
	"fmt"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"pocketbase/hooks"
	"testing"
	"time"
)

// Shared schema fixtures are independent of the child-cleanup regression
// file. Photo identity tests can select them without importing another
// feature's tests; asset-specific assertions retain the current schema here.

type remoteChildCleanupFixture struct {
	*remoteSyncFixture
	target                              string
	children, links, assets, trailLinks *core.Collection
	trail                               *core.Record
}

func newRemoteChildCleanupFixture(t *testing.T, target string) *remoteChildCleanupFixture {
	t.Helper()
	base := newRemoteSyncFixture(t)
	base.remoteActor.Set("iri", "https://a.example/api/v1/actor/riley")
	base.remoteActor.Set("domain", "a.example")
	if err := base.app.Save(base.remoteActor); err != nil {
		t.Fatal(err)
	}
	f := &remoteChildCleanupFixture{remoteSyncFixture: base, target: target}
	f.children = map[string]*core.Collection{"waypoint": f.waypoints, "summit_log": f.logs}[target]
	f.trail = f.save(f.trails, map[string]any{"iri": "https://a.example/api/v1/trail/trail", "author": f.remoteActor.Id, "public": true})
	f.assets, _ = f.app.FindCollectionByNameOrId("assets")
	f.assets.Fields.Add(&core.TextField{Name: "type"}, &core.TextField{Name: "storage_mode"}, &core.TextField{Name: "external_provider"}, &core.TextField{Name: "external_id"}, &core.TextField{Name: "author"}, &core.FileField{Name: "file", MaxSelect: 1, MaxSize: 1024})
	if err := f.app.Save(f.assets); err != nil {
		t.Fatal(err)
	}
	for name, field := range map[string]string{"trail_assets": "trail", "waypoint_assets": "waypoint", "summit_log_assets": "summit_log"} {
		links, err := f.app.FindCollectionByNameOrId(name)
		if err != nil {
			t.Fatal(err)
		}
		collection := map[string]*core.Collection{"trail": f.trails, "waypoint": f.waypoints, "summit_log": f.logs}[field]
		links.Fields.RemoveByName(field)
		if err := f.app.Save(links); err != nil {
			t.Fatal(err)
		}
		links.Fields.Add(&core.RelationField{Name: field, CollectionId: collection.Id, MaxSelect: 1, CascadeDelete: true})
		if err := f.app.Save(links); err != nil {
			t.Fatal(err)
		}
		if field == target {
			f.links = links
		}
		if field == "trail" {
			f.trailLinks = links
		}
	}
	f.app.OnRecordDelete("waypoints", "summit_logs").BindFunc(hooks.CollectDeletedRecordAssetsHandler())
	f.app.OnRecordAfterDeleteSuccess("waypoints", "summit_logs").BindFunc(hooks.DeleteOrphanedAssetsHandler())
	f.app.OnRecordAfterDeleteError("waypoints", "summit_logs").BindFunc(hooks.DeleteOrphanedAssetsAfterErrorHandler())
	return f
}

func (f *remoteChildCleanupFixture) child(iri, author, trail string) *core.Record {
	return f.save(f.children, map[string]any{"iri": iri, "author": author, "trail": trail, "name": "Original", "text": "Original"})
}

func (f *remoteChildCleanupFixture) photo(child *core.Record, name string) *core.Record {
	f.t.Helper()
	file, err := filesystem.NewFileFromBytes([]byte("photo content"), name)
	if err != nil {
		f.t.Fatal(err)
	}
	asset := f.save(f.assets, map[string]any{"type": "photo", "storage_mode": "copy", "external_provider": "federation", "external_id": "https://a.example/api/v1/assets/" + name, "author": f.remoteActor.Id, "file": file})
	f.save(f.links, map[string]any{f.target: child.Id, "asset": asset.Id})
	return asset
}

func (f *remoteChildCleanupFixture) sync(app core.App, raw []any) error {
	if f.target == "waypoint" {
		return syncWaypoints(app, f.t.Context(), f.trail, "https://a.example", raw)
	}
	return syncSummitLogs(app, f.t.Context(), f.trail, "https://a.example", raw)
}

func (f *remoteChildCleanupFixture) assertFile(asset *core.Record, want bool) {
	f.t.Helper()
	storage, err := f.app.NewFilesystem()
	if err != nil {
		f.t.Fatal(err)
	}
	defer storage.Close()
	key := fmt.Sprintf("%s/%s", asset.BaseFilesPath(), asset.GetString("file"))
	got, err := storage.Exists(key)
	// PocketBase removes a deleted record's file prefix asynchronously.
	for deadline := time.Now().Add(time.Second); err == nil && got != want && time.Now().Before(deadline); {
		time.Sleep(5 * time.Millisecond)
		got, err = storage.Exists(key)
	}
	if err != nil || got != want {
		f.t.Fatalf("photo bytes at %s: exists=%t want=%t error=%v", key, got, want, err)
	}
}
