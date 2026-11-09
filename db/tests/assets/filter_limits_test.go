package assets_test

import (
	"fmt"
	"slices"
	"sort"
	"testing"
	"time"

	"pocketbase/hooks"
	_ "pocketbase/migrations"
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/core"
)

func newAssetFilterTestApp(t *testing.T) *core.BaseApp {
	t.Helper()
	t.Setenv("ORIGIN", "https://a.example")
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	app.Settings().Logs.MaxDays = 0
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	migrations := slices.Clone(core.AppMigrations.Items())
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].File < migrations[j].File })
	for _, migration := range migrations {
		switch migration.File {
		case "1742167033_init_meilisearch.go", "1744651602_add_polyline.go", "1749831369_update_sortable_attributes.go":
			// Only these external search-index migrations are unrelated to the schema.
			continue
		}
		if err := app.RunInTransaction(migration.Up); err != nil {
			t.Fatalf("apply %s: %v", migration.File, err)
		}
	}
	app.OnRecordCreate("assets").BindFunc(hooks.StoreAssetSourceMetadata())
	app.OnRecordUpdate("assets").BindFunc(hooks.StoreAssetSourceMetadata())
	return app
}

func saveAssetFilterRecord(t *testing.T, app core.App, collection string, data map[string]any) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(col)
	record.Load(data)
	if err := app.Save(record); err != nil {
		t.Fatalf("save %s: %v", collection, err)
	}
	return record
}

func TestLargeTrailAssetQueriesWithCurrentSchema(t *testing.T) {
	app := newAssetFilterTestApp(t)
	actor := saveAssetFilterRecord(t, app, "activitypub_actors", map[string]any{
		"username": "photos", "preferred_username": "photos", "domain": "a.example",
		"public_key": "test-key", "is_local": true, "iri": "https://a.example/photos",
		"inbox": "https://a.example/photos/inbox",
	})
	trail := saveAssetFilterRecord(t, app, "trails", map[string]any{"name": "many photos", "author": actor.Id})
	otherTrail := saveAssetFilterRecord(t, app, "trails", map[string]any{"name": "other trail", "author": actor.Id})

	const count = 225
	allIDs := []string{}
	directIDs := []string{}
	var firstWaypoint, firstSummitLog *core.Record
	if err := app.RunInTransaction(func(tx core.App) error {
		for _, target := range []string{"trail", "waypoint", "summit_log"} {
			for i := range count {
				targetID := trail.Id
				if target != "trail" {
					record := saveAssetFilterRecord(t, tx, target+"s", map[string]any{
						"name": fmt.Sprintf("%s %d", target, i), "trail": trail.Id, "author": actor.Id,
					})
					targetID = record.Id
					if i == 0 && target == "waypoint" {
						firstWaypoint = record
					}
					if i == 0 && target == "summit_log" {
						firstSummitLog = record
					}
				}
				asset := saveAssetFilterRecord(t, tx, "assets", map[string]any{
					"type": "photo", "storage_mode": "copy", "author": actor.Id,
				})
				allIDs = append(allIDs, asset.Id)
				if target == "trail" {
					directIDs = append(directIDs, asset.Id)
				}
				saveAssetFilterRecord(t, tx, target+"_assets", map[string]any{
					target: targetID, "asset": asset.Id,
					"created": time.Date(2026, 1, 1, 0, 0, i, 0, time.UTC),
				})
			}
		}
		// The same photo on several targets must still appear only once in a
		// trail's publication inventory.
		saveAssetFilterRecord(t, tx, "waypoint_assets", map[string]any{"waypoint": firstWaypoint.Id, "asset": directIDs[0]})
		saveAssetFilterRecord(t, tx, "summit_log_assets", map[string]any{"summit_log": firstSummitLog.Id, "asset": directIDs[0]})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	otherAsset := saveAssetFilterRecord(t, app, "assets", map[string]any{
		"type": "photo", "storage_mode": "copy", "author": actor.Id,
	})
	saveAssetFilterRecord(t, app, "trail_assets", map[string]any{"trail": otherTrail.Id, "asset": otherAsset.Id})

	t.Run("photos retain link order and global limit", func(t *testing.T) {
		for _, limit := range []int{-1, 3, 175} {
			photos, err := util.PhotoAssetsForTarget(app, "trail", trail.Id, limit)
			if err != nil {
				t.Fatal(err)
			}
			want := directIDs
			if limit > 0 {
				want = want[:limit]
			}
			got := make([]string, 0, len(photos))
			for _, photo := range photos {
				got = append(got, photo.Id)
			}
			if !slices.Equal(got, want) {
				t.Fatalf("limit %d returned %d photos with unexpected order, want %d", limit, len(got), len(want))
			}
		}
		if _, err := util.ObjectFromTrail(app, trail, nil); err != nil {
			t.Fatalf("federation object: %v", err)
		}
	})
}
