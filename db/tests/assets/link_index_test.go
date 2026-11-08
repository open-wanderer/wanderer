package assets_test

import (
	"strings"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func TestAssetLinkCollectionsEnforceUniqueTargets(t *testing.T) {
	app := newAssetFilterTestApp(t)
	targets, asset := assetLinkIndexFixture(t, app)
	for kind, ids := range targets {
		t.Run(kind, func(t *testing.T) {
			collection := kind + "_assets"
			saveAssetFilterRecord(t, app, collection, map[string]any{"asset": asset.Id, kind: ids[0]})
			table, err := app.FindCollectionByNameOrId(collection)
			if err != nil {
				t.Fatal(err)
			}
			duplicate := core.NewRecord(table)
			duplicate.Set("asset", asset.Id)
			duplicate.Set(kind, ids[0])
			if err := app.SaveNoValidate(duplicate); err == nil {
				t.Fatal("database accepted a duplicate asset/target link")
			}
			saveAssetFilterRecord(t, app, collection, map[string]any{"asset": asset.Id, kind: ids[1]})
		})
	}
}

func TestAssetLinkCollectionsUseTargetIndexes(t *testing.T) {
	app := newAssetFilterTestApp(t)
	targets, _ := assetLinkIndexFixture(t, app)
	for kind, ids := range targets {
		t.Run(kind, func(t *testing.T) {
			collection := kind + "_assets"
			index := "idx_" + collection + "_" + kind
			rows := []struct{ Detail string }{}
			if err := app.DB().NewQuery("EXPLAIN QUERY PLAN SELECT id FROM " + collection + " WHERE " + kind + " IN ({:first},{:second})").Bind(dbx.Params{"first": ids[0], "second": ids[1]}).All(&rows); err != nil {
				t.Fatal(err)
			}
			used := false
			for _, row := range rows {
				if strings.Contains(row.Detail, "INDEX "+index+" (") {
					used = true
				}
				if strings.Contains(row.Detail, "SCAN "+collection) || strings.Contains(row.Detail, "TEMP B-TREE") {
					t.Fatalf("target query was not indexed: %s", row.Detail)
				}
			}
			if !used {
				t.Fatalf("exact target index %s not used: %v", index, rows)
			}
		})
	}
}

func assetLinkIndexFixture(t *testing.T, app core.App) (map[string][]string, *core.Record) {
	t.Helper()
	actor := saveAssetFilterRecord(t, app, "activitypub_actors", map[string]any{"username": "photos", "preferred_username": "photos", "is_local": true, "public_key": "key", "iri": "https://a.example/photos", "inbox": "https://a.example/inbox"})
	targets := map[string][]string{}
	for i := 0; i < 2; i++ {
		trail := saveAssetFilterRecord(t, app, "trails", map[string]any{"name": "Trail", "author": actor.Id})
		point := saveAssetFilterRecord(t, app, "waypoints", map[string]any{"name": "Point", "author": actor.Id, "trail": trail.Id})
		entry := saveAssetFilterRecord(t, app, "summit_logs", map[string]any{"author": actor.Id, "trail": trail.Id, "date": "2026-01-01"})
		targets["trail"] = append(targets["trail"], trail.Id)
		targets["waypoint"] = append(targets["waypoint"], point.Id)
		targets["summit_log"] = append(targets["summit_log"], entry.Id)
	}
	asset := saveAssetFilterRecord(t, app, "assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": actor.Id})
	return targets, asset
}

func TestAssetLinkCollectionsUseUniqueIndexesForAssetQueries(t *testing.T) {
	app := newAssetFilterTestApp(t)
	_, asset := assetLinkIndexFixture(t, app)
	for _, kind := range []string{"trail", "waypoint", "summit_log"} {
		t.Run(kind, func(t *testing.T) {
			collection := kind + "_assets"
			queries := []string{
				"SELECT id FROM " + collection + " WHERE asset={:asset} LIMIT 1",
				"SELECT * FROM " + collection + " WHERE asset={:asset}",
				"SELECT * FROM " + collection + " WHERE asset IN ({:asset},'other') AND id>'' ORDER BY id LIMIT 150",
				"SELECT id FROM " + collection + " WHERE asset={:asset} AND " + kind + "='target' LIMIT 1",
			}
			for _, query := range queries {
				rows := []struct{ Detail string }{}
				if err := app.DB().NewQuery("EXPLAIN QUERY PLAN " + query).Bind(dbx.Params{"asset": asset.Id}).All(&rows); err != nil {
					t.Fatal(err)
				}
				used := false
				for _, row := range rows {
					if strings.Contains(row.Detail, "INDEX idx_"+collection+"_unique (") {
						used = true
					}
					if strings.Contains(row.Detail, "SCAN "+collection) {
						t.Fatalf("asset query scans the table: %s", row.Detail)
					}
				}
				if !used {
					t.Fatalf("asset query did not use unique index: %s: %v", query, rows)
				}
			}
			redundant := []struct{ Name string }{}
			if err := app.DB().NewQuery("SELECT name FROM sqlite_master WHERE type='index' AND name={:name}").Bind(dbx.Params{"name": "idx_" + collection + "_asset"}).All(&redundant); err != nil {
				t.Fatal(err)
			}
			if len(redundant) != 0 {
				t.Fatal("redundant asset index remains")
			}
		})
	}
}
