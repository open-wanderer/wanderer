package main

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// Capture the production collector and PocketBase cascade queries so changes
// to either query cannot leave a separately copied EXPLAIN fixture passing.
func TestAssetCascadeDeletesUseRegisteredTargetIndexes(t *testing.T) {
	for _, root := range []string{"trails", "activitypub_actors"} {
		t.Run(root, func(t *testing.T) {
			f := newRecordHooksFixture(t)
			actor := f.localActor("cascade-index")
			trail := f.save("trails", map[string]any{"name": "Trail", "author": actor.Id})
			f.save("waypoints", map[string]any{"name": "Point", "author": actor.Id, "trail": trail.Id})
			f.save("summit_logs", map[string]any{"author": actor.Id, "trail": trail.Id, "date": "2026-01-01"})
			f.bindHooks()
			queries := map[string]string{}
			db := f.app.NonconcurrentDB().(*dbx.DB)
			previous := db.QueryLogFunc
			db.QueryLogFunc = func(_ context.Context, _ time.Duration, query string, _ *sql.Rows, _ error) {
				if strings.Contains(query, "UNION SELECT asset FROM waypoint_assets") {
					queries["root"] = query
				}
				for _, name := range []string{"waypoints", "summit_logs"} {
					if strings.Contains(query, "SELECT `"+name+"`.* FROM `"+name+"`") && strings.Contains(query, "`"+name+"`.`trail`=") {
						queries[name] = query
					}
				}
			}
			target := trail
			if root == "activitypub_actors" {
				target = actor
			}
			err := f.app.Delete(target)
			db.QueryLogFunc = previous
			if err != nil {
				t.Fatal(err)
			}
			for _, kind := range []string{"root", "waypoints", "summit_logs"} {
				query := queries[kind]
				if query == "" {
					t.Fatalf("production %s lookup was not captured: %v", kind, queries)
				}
				indexes := []string{}
				for _, name := range []string{"waypoints", "summit_logs"} {
					if kind != "root" && kind != name {
						continue
					}
					trailIndex := "idx_" + name + "_trail"
					if name == "summit_logs" {
						trailIndex += "_author"
					}
					indexes = append(indexes, trailIndex)
					if kind == "root" && root == "activitypub_actors" {
						authorIndex := "idx_" + name + "_author"
						if name == "summit_logs" {
							authorIndex += "_date"
						}
						indexes = append(indexes, authorIndex)
					}
				}
				assertCascadeQueryIndexes(t, f.app, query, indexes)
			}
		})
	}
}

func assertCascadeQueryIndexes(t *testing.T, app core.App, query string, indexes []string) {
	t.Helper()
	plan := []struct{ Detail string }{}
	if err := app.DB().NewQuery("EXPLAIN QUERY PLAN " + query).All(&plan); err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, row := range plan {
		t.Log(row.Detail)
		for _, name := range []string{"waypoints", "summit_logs"} {
			if strings.Contains(row.Detail, "SCAN "+name) || strings.Contains(row.Detail, "SCAN `"+name+"`") {
				t.Fatalf("cascade scans %s: %s", name, query)
			}
		}
		for _, index := range indexes {
			if strings.Contains(row.Detail, "INDEX "+index+" (") {
				found[index] = true
			}
		}
	}
	for _, index := range indexes {
		if !found[index] {
			t.Fatalf("cascade query does not use %s: %s: %v", index, query, plan)
		}
	}
}
