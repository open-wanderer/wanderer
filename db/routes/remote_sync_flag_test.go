package routes

import (
	"context"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

// A list sync must preserve local sync state: placeholders need a full sync
// before they can be served, while completed copies remain usable as a cache.
func TestSyncTrailsKeepsLocalSyncFlags(t *testing.T) {
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(
		&core.TextField{Name: "iri"},
		&core.TextField{Name: "name"},
		&core.TextField{Name: "author"},
		&core.BoolField{Name: "public"},
		&core.BoolField{Name: "needs_full_sync"},
		&core.BoolField{Name: "full_sync_completed"},
		&core.NumberField{Name: "distance"},
	)
	if err := app.Save(trails); err != nil {
		t.Fatal(err)
	}

	lists := core.NewBaseCollection("lists")
	lists.Fields.Add(
		&core.TextField{Name: "author"},
		&core.RelationField{Name: "trails", CollectionId: trails.Id, MaxSelect: 100},
	)
	if err := app.Save(lists); err != nil {
		t.Fatal(err)
	}
	list := core.NewRecord(lists)
	list.Set("author", "actor0000000001")

	const origin = "https://remote.example"
	// Metadata import mutates the payload, so each sync needs a fresh copy.
	remoteTrails := func(name string, needsSync, completed bool) []any {
		return []any{map[string]any{
			"id":                  "remote00000001",
			"name":                name,
			"public":              true,
			"distance":            1234.5,
			"needs_full_sync":     needsSync,
			"full_sync_completed": completed,
		}}
	}

	if err := syncTrails(app, context.Background(), list, origin, remoteTrails("Remote trail", false, true)); err != nil {
		t.Fatalf("syncTrails: %v", err)
	}

	placeholder, err := app.FindFirstRecordByData("trails", "iri", origin+"/api/v1/trail/remote00000001")
	if err != nil {
		t.Fatalf("placeholder not created: %v", err)
	}
	if !placeholder.GetBool("needs_full_sync") {
		t.Fatal("placeholder from list sync must stay needs_full_sync = true")
	}
	if placeholder.GetBool("full_sync_completed") {
		t.Fatal("placeholder from list sync must stay full_sync_completed = false")
	}
	if cached := cachedRecordFallback(app, "trails", placeholder.Id, errRemoteUnavailable); cached != nil {
		t.Fatal("placeholder must not be served as a completed cache")
	}
	if placeholder.GetString("name") != "Remote trail" || placeholder.GetFloat("distance") != 1234.5 || !placeholder.GetBool("public") {
		t.Fatalf("metadata not loaded: %v", placeholder.PublicExport())
	}
	if got := list.GetStringSlice("trails"); len(got) != 1 || got[0] != placeholder.Id {
		t.Fatalf("list.trails = %v, want [%s]", got, placeholder.Id)
	}

	// A copy that was fully synced before keeps that state when the list is
	// synced again, even if the remote payload claims otherwise.
	placeholder.Set("needs_full_sync", false)
	placeholder.Set("full_sync_completed", true)
	if err := app.Save(placeholder); err != nil {
		t.Fatal(err)
	}
	if err := syncTrails(app, context.Background(), list, origin, remoteTrails("Updated trail", true, false)); err != nil {
		t.Fatalf("syncTrails: %v", err)
	}
	synced, err := app.FindRecordById("trails", placeholder.Id)
	if err != nil {
		t.Fatal(err)
	}
	if synced.GetBool("needs_full_sync") {
		t.Fatal("fully synced copy must not be flipped back to pending by the remote payload")
	}
	if !synced.GetBool("full_sync_completed") {
		t.Fatal("fully synced copy must retain its local completion flag")
	}
	if synced.GetString("name") != "Updated trail" {
		t.Fatalf("name = %q, want %q", synced.GetString("name"), "Updated trail")
	}

	// A later refresh request must still allow the completed copy as fallback.
	synced.Set("needs_full_sync", true)
	if err := app.Save(synced); err != nil {
		t.Fatal(err)
	}
	if err := syncTrails(app, context.Background(), list, origin, remoteTrails("Updated trail", false, false)); err != nil {
		t.Fatalf("syncTrails: %v", err)
	}
	cached := cachedRecordFallback(app, "trails", synced.Id, errRemoteUnavailable)
	if cached == nil || cached.Id != synced.Id {
		t.Fatal("completed copy must remain available as fallback while a refresh is pending")
	}
	if !cached.GetBool("needs_full_sync") {
		t.Fatal("pending refresh must not be cleared by the remote payload")
	}
}

func TestSyncListMetadataKeepsLocalSyncFlags(t *testing.T) {
	lists := core.NewBaseCollection("lists")
	lists.Fields.Add(
		&core.TextField{Name: "name"},
		&core.BoolField{Name: "needs_full_sync"},
		&core.BoolField{Name: "full_sync_completed"},
	)

	for _, tt := range []struct {
		name      string
		needsSync bool
		completed bool
	}{
		{"placeholder", true, false},
		{"completed", false, true},
		{"pending refresh", true, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			record := core.NewRecord(lists)
			record.Set("needs_full_sync", tt.needsSync)
			record.Set("full_sync_completed", tt.completed)

			syncListMetadata(record, map[string]any{
				"id":                  "remote00000001",
				"name":                "Remote list",
				"needs_full_sync":     !tt.needsSync,
				"full_sync_completed": !tt.completed,
			})

			if got := record.GetBool("needs_full_sync"); got != tt.needsSync {
				t.Fatalf("needs_full_sync = %v, want local value %v", got, tt.needsSync)
			}
			if got := record.GetBool("full_sync_completed"); got != tt.completed {
				t.Fatalf("full_sync_completed = %v, want local value %v", got, tt.completed)
			}
			if record.GetString("name") != "Remote list" {
				t.Fatalf("name = %q, want %q", record.GetString("name"), "Remote list")
			}
		})
	}
}
