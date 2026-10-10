package importer

import (
	"context"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"pocketbase/pluginsystem"
)

func TestImportCompletedTrailSetsSummitLogIRI(t *testing.T) {
	t.Setenv("ORIGIN", "https://wanderer.example.test")
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	collection := func(name string, fields ...core.Field) *core.Collection {
		t.Helper()
		c := core.NewBaseCollection(name)
		c.Fields.Add(fields...)
		if err := app.Save(c); err != nil {
			t.Fatal(err)
		}
		return c
	}
	actors := collection("activitypub_actors", &core.TextField{Name: "user"})
	actor := core.NewRecord(actors)
	actor.Set("user", "import-test-user")
	if err := app.Save(actor); err != nil {
		t.Fatal(err)
	}
	trails := collection("trails",
		&core.TextField{Name: "name"}, &core.TextField{Name: "description"},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
		&core.FileField{Name: "gpx", MaxSelect: 1, MaxSize: 1 << 20},
		&core.BoolField{Name: "completed"}, &core.DateField{Name: "date"},
	)
	collection("summit_logs",
		&core.URLField{Name: "iri"},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
		&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1},
		&core.DateField{Name: "date"},
		&core.NumberField{Name: "distance"}, &core.NumberField{Name: "duration"},
	)
	collection("trail_external_reference",
		&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1},
		&core.TextField{Name: "user"}, &core.TextField{Name: "provider"},
		&core.TextField{Name: "external_id"}, &core.TextField{Name: "plugin_id"},
		&core.TextField{Name: "provider_category"}, &core.DateField{Name: "provider_category_checked_at"},
		&core.AutodateField{Name: "created", OnCreate: true},
	)

	item := pluginsystem.TrailImport{
		Source: pluginsystem.TrailImportSource{Provider: "test", ExternalID: "completed-activity"},
		Kind:   "completed", Name: "Imported activity", Track: gpxTrack(),
	}
	opts := Options{UserID: "import-test-user", ActorID: actor.Id, CreateSummitLogForCompleted: true,
		Manifest: pluginsystem.Manifest{ID: "test"}}
	result, err := ImportTrail(context.Background(), app, item, opts)
	if err != nil {
		t.Fatal(err)
	}
	logs, err := app.FindRecordsByFilter("summit_logs", "", "", 0, 0)
	if err != nil || len(logs) != 1 {
		t.Fatalf("expected one imported summit log, got %d: %v", len(logs), err)
	}
	log := logs[0]
	if got, want := log.GetString("iri"), "https://wanderer.example.test/api/v1/summit-log/"+log.Id; got != want {
		t.Errorf("summit log iri = %q, want %q", got, want)
	}
	if log.GetString("trail") != result.TrailID || log.GetString("author") != actor.Id {
		t.Error("summit log lost its imported trail or author")
	}
	if log.GetDateTime("date").IsZero() || log.GetFloat("distance") <= 0 || log.GetFloat("duration") != 600 {
		t.Error("summit log lost its imported activity metrics")
	}
	replay, err := ImportTrail(context.Background(), app, item, opts)
	if err != nil || !replay.Skipped {
		t.Fatalf("repeat import must remain deduplicated: result=%v error=%v", replay, err)
	}
	logs, err = app.FindRecordsByFilter("summit_logs", "", "", 0, 0)
	if err != nil || len(logs) != 1 || logs[0].GetString("iri") != log.GetString("iri") {
		t.Fatalf("repeat import changed the summit log identity: %v", err)
	}

	t.Run("missing origin does not leave a partial import", func(t *testing.T) {
		t.Setenv("ORIGIN", "")
		item.Source.ExternalID = "another-completed-activity"
		if result, err := ImportTrail(context.Background(), app, item, opts); err == nil || result != nil {
			t.Fatalf("expected import to fail without ORIGIN, got result=%v error=%v", result, err)
		}
		for _, name := range []string{"trails", "summit_logs", "trail_external_reference"} {
			records, err := app.FindRecordsByFilter(name, "", "", 0, 0)
			if err != nil || len(records) != 1 {
				t.Fatalf("failed import changed %s: count=%d error=%v", name, len(records), err)
			}
		}
	})
}
