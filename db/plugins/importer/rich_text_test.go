package importer

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"pocketbase/pluginsystem"
	"pocketbase/util"
)

func TestImportTrailSanitizesRichTextBeforeFieldValidation(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	app.OnRecordValidate().BindFunc(util.SanitizeHTML())

	collection := func(name string, fields ...core.Field) *core.Collection {
		t.Helper()
		c := core.NewBaseCollection(name)
		c.Fields.Add(fields...)
		if err := app.Save(c); err != nil {
			t.Fatal(err)
		}
		return c
	}
	actors := collection("activitypub_actors", &core.TextField{Name: "user"}, &core.TextField{Name: "summary"})
	actor := core.NewRecord(actors)
	actor.Set("user", "local-user")
	if err := app.Save(actor); err != nil {
		t.Fatal(err)
	}
	trails := collection("trails",
		&core.TextField{Name: "name"}, &core.TextField{Name: "description", Max: 10000},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
		&core.FileField{Name: "gpx", MaxSelect: 1, MaxSize: 1 << 20},
		&core.BoolField{Name: "completed"}, &core.DateField{Name: "date"},
	)
	collection("waypoints",
		&core.TextField{Name: "name"}, &core.TextField{Name: "description"},
		&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
	)
	collection("summit_logs",
		&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
		&core.DateField{Name: "date"}, &core.TextField{Name: "text"},
	)
	collection("trail_external_reference",
		&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1},
		&core.TextField{Name: "user"}, &core.TextField{Name: "provider"},
		&core.TextField{Name: "external_id"}, &core.TextField{Name: "plugin_id"},
		&core.TextField{Name: "provider_category"}, &core.DateField{Name: "provider_category_checked_at"},
		&core.AutodateField{Name: "created", OnCreate: true},
	)

	// Quotes and ampersands expand during sanitization. The stored serialization,
	// including closing tags, must fit both the trail and waypoint field limits.
	longText := `<p onclick="blocked()"><strong>Safe ` + strings.Repeat(`'"&山🚲`, 3000) + `</strong></p>`
	item := pluginsystem.TrailImport{
		Source: pluginsystem.TrailImportSource{Provider: "test", ExternalID: "rich-text-activity"},
		Kind:   "completed", Name: "Rich-text import", Description: longText, Track: gpxTrack(),
		Waypoints: []pluginsystem.Waypoint{
			{Name: "Long description", Description: longText, Lat: 46, Lon: 8},
			{Name: "Next waypoint", Description: "<p>Still imported</p>", Lat: 46.001, Lon: 8.001},
		},
	}
	opts := Options{UserID: "local-user", ActorID: actor.Id, CreateSummitLogForCompleted: true,
		Manifest: pluginsystem.Manifest{ID: "test"}}
	result, err := ImportTrail(context.Background(), app, item, opts)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Created {
		t.Fatal("expected a completed imported trail")
	}
	trail, err := app.FindRecordById(trails, result.TrailID)
	if err != nil {
		t.Fatal(err)
	}
	if trail.GetString("gpx") == "" || !trail.GetBool("completed") || trail.GetDateTime("date").IsZero() {
		t.Fatal("import lost its track or completed-activity metadata")
	}
	assertRichText := func(value string, limit int) {
		t.Helper()
		if utf8.RuneCountInString(value) > limit || !utf8.ValidString(value) || strings.Contains(value, "blocked()") {
			t.Fatal("imported rich text is unsafe or exceeds the stored field limit")
		}
		if !strings.Contains(value, "<strong>Safe ") || !strings.HasSuffix(value, "</strong></p>") {
			t.Fatal("import lost safe formatting or cut closing tags")
		}
	}
	assertRichText(trail.GetString("description"), 10000)
	waypoints, err := app.FindRecordsByFilter("waypoints", "", "", 0, 0)
	if err != nil || len(waypoints) != 2 {
		t.Fatalf("import stopped before all waypoints: count=%d error=%v", len(waypoints), err)
	}
	for _, waypoint := range waypoints {
		if waypoint.GetString("name") == "Long description" {
			assertRichText(waypoint.GetString("description"), 5000)
		}
	}
	for _, name := range []string{"summit_logs", "trail_external_reference"} {
		records, err := app.FindRecordsByFilter(name, "", "", 0, 0)
		if err != nil || len(records) != 1 {
			t.Fatalf("import did not finish %s: count=%d error=%v", name, len(records), err)
		}
	}
	replay, err := ImportTrail(context.Background(), app, item, opts)
	if err != nil || !replay.Skipped {
		t.Fatalf("completed import was not deduplicated: result=%v error=%v", replay, err)
	}
}
