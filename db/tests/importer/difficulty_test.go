package importer_test

import (
	"context"
	"encoding/base64"
	"slices"
	"sort"
	"testing"

	_ "pocketbase/migrations"
	"pocketbase/plugins/importer"
	"pocketbase/pluginsystem"

	"github.com/pocketbase/pocketbase/core"
)

func TestImportTrailPreservesUnknownDifficulty(t *testing.T) {
	app := newImporterTestApp(t)
	user := saveImporterTestRecord(t, app, "users", map[string]any{
		"username": "importer", "password": "test-password", "email": "importer@example.com",
	})
	author := saveImporterTestRecord(t, app, "activitypub_actors", map[string]any{
		"username": "importer", "preferred_username": "importer", "domain": "example.com",
		"user": user.Id, "is_local": true, "public_key": "test-key",
		"iri": "https://example.com/importer", "inbox": "https://example.com/importer/inbox",
	})
	opts := importer.Options{UserID: user.Id, ActorID: author.Id}

	for _, kind := range []string{"planned", "completed"} {
		t.Run(kind, func(t *testing.T) {
			item := pluginsystem.TrailImport{
				Name: "Unrated " + kind, Kind: kind,
				Source: pluginsystem.TrailImportSource{Provider: "test-provider", ExternalID: kind},
				Track: pluginsystem.Track{
					Format: "gpx",
					ContentBase64: base64.StdEncoding.EncodeToString([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="test"><trk><trkseg>
<trkpt lat="46" lon="8"><ele>100</ele><time>2026-01-01T10:00:00Z</time></trkpt>
<trkpt lat="46.001" lon="8.001"><ele>120</ele><time>2026-01-01T10:10:00Z</time></trkpt>
</trkseg></trk></gpx>`)),
				},
				// Provider-specific metadata is not a canonical wanderer rating.
				Metadata: map[string]any{"difficulty": "easy"},
			}
			result, err := importer.ImportTrail(context.Background(), app, item, opts)
			if err != nil {
				t.Fatal(err)
			}
			if !result.Created || result.Skipped {
				t.Fatalf("first import = %#v, want a new trail", result)
			}
			trail := findImporterTestTrail(t, app, result.TrailID)
			if got := trail.GetString("difficulty"); got != "" {
				t.Fatalf("new import difficulty = %q, want unknown", got)
			}
			if trail.GetString("gpx") == "" {
				t.Fatal("new import did not persist the GPX file")
			}

			// Deduplication must preserve both the absence of a rating and an
			// explicit rating assigned after import, without creating a new trail.
			for _, difficulty := range []string{"", "easy", "moderate", "difficult"} {
				trail.Set("difficulty", difficulty)
				if err := app.Save(trail); err != nil {
					t.Fatal(err)
				}
				repeated, err := importer.ImportTrail(context.Background(), app, item, opts)
				if err != nil {
					t.Fatal(err)
				}
				if repeated.Created || !repeated.Skipped || repeated.TrailID != trail.Id {
					t.Fatalf("repeat import = %#v, want existing trail %s", repeated, trail.Id)
				}
				if got := findImporterTestTrail(t, app, trail.Id).GetString("difficulty"); got != difficulty {
					t.Fatalf("repeat import difficulty = %q, want unchanged %q", got, difficulty)
				}
			}
		})
	}
}

func findImporterTestTrail(t *testing.T, app core.App, id string) *core.Record {
	t.Helper()
	trail, err := app.FindRecordById("trails", id)
	if err != nil {
		t.Fatal(err)
	}
	return trail
}

// Use the production schema, including its optional difficulty field and
// provider-scoped references. Only external search-index migrations are skipped.
func newImporterTestApp(t *testing.T) *core.BaseApp {
	t.Helper()
	t.Setenv("ORIGIN", "https://example.com")
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	migrations := slices.Clone(core.AppMigrations.Items())
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].File < migrations[j].File })
	for _, migration := range migrations {
		switch migration.File {
		case "1742167033_init_meilisearch.go", "1744651602_add_polyline.go", "1749831369_update_sortable_attributes.go":
			continue
		}
		if err := app.RunInTransaction(migration.Up); err != nil {
			t.Fatalf("apply %s: %v", migration.File, err)
		}
	}
	return app
}

func saveImporterTestRecord(t *testing.T, app core.App, collection string, values map[string]any) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(col)
	record.Load(values)
	if err := app.Save(record); err != nil {
		t.Fatalf("save %s: %v", collection, err)
	}
	return record
}
