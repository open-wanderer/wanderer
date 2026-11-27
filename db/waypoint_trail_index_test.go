package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func TestWaypointTrailIndexMigration(t *testing.T) {
	app := newRealSchemaApp(t)
	var migration *core.Migration
	for _, candidate := range core.AppMigrations.Items() {
		if candidate.File == "1791800002_waypoints_trail_index.go" {
			migration = candidate
		}
	}
	if migration == nil {
		t.Fatal("waypoint trail index migration not registered")
	}
	assertSearch := func() {
		t.Helper()
		var plan []struct{ Detail string }
		if err := app.DB().NewQuery("EXPLAIN QUERY PLAN SELECT id, iri FROM waypoints WHERE trail = {:trail}").Bind(dbx.Params{"trail": "parent"}).All(&plan); err != nil {
			t.Fatal(err)
		}
		if len(plan) != 1 || !strings.Contains(plan[0].Detail, "SEARCH waypoints USING INDEX idx_waypoints_trail") {
			t.Fatalf("trail query should use the additive index: %+v", plan)
		}
	}
	assertSearch() // Fresh install, including all historical schema migrations.
	if err := app.RunInTransaction(migration.Down); err != nil {
		t.Fatal(err)
	}
	collection, err := app.FindCollectionByNameOrId("waypoints")
	if err != nil {
		t.Fatal(err)
	}
	collection.Indexes = append(collection.Indexes, "CREATE INDEX idx_waypoints_test_name ON waypoints (name)")
	if err := app.Save(collection); err != nil {
		t.Fatal(err)
	}
	before := append([]string(nil), collection.Indexes...)
	waypoint := core.NewRecord(collection)
	waypoint.Set("name", "Preserved")
	waypoint.Set("description", "Historical data")
	waypoint.Set("iri", "https://remote.example/api/v1/waypoint/preserved")
	if err := app.SaveNoValidate(waypoint); err != nil {
		t.Fatal(err)
	}
	// A one-row table may legitimately be cheaper to scan. Give SQLite a
	// representative unrelated cache before checking the upgrade query plan.
	if _, err := app.DB().NewQuery(`WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i+1 FROM n WHERE i<1000)
		INSERT INTO waypoints (id, trail) SELECT printf('w%014d', i), 'unrelated' FROM n`).Execute(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ { // Upgrade and idempotent reapplication.
		if err := app.RunInTransaction(migration.Up); err != nil {
			t.Fatal(err)
		}
		assertSearch()
	}
	if err := app.RunInTransaction(migration.Down); err != nil {
		t.Fatal(err)
	}
	collection, err = app.FindCollectionByNameOrId("waypoints")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(collection.Indexes, before) {
		t.Fatalf("down must retain every other index: got %v want %v, error %v", collection.Indexes, before, err)
	}
	var count int
	if err := app.DB().NewQuery("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_waypoints_trail'").Row(&count); err != nil || count != 0 {
		t.Fatalf("down left the index behind: count=%d error=%v", count, err)
	}
	stored, err := app.FindRecordById("waypoints", waypoint.Id)
	if err != nil || stored.GetString("name") != "Preserved" || stored.GetString("description") != "Historical data" || stored.GetString("iri") != waypoint.GetString("iri") {
		t.Fatalf("migration modified waypoint data: %v, %v", stored, err)
	}
}
