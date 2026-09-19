package importer_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"
	"testing"

	_ "pocketbase/migrations"
	"pocketbase/plugins/importer"
	"pocketbase/pluginsystem"

	"github.com/pocketbase/pocketbase/core"
)

func TestImportProviderDifficultyContract(t *testing.T) {
	app, opts := newProviderDifficultyApp(t)
	for _, test := range []struct {
		name  string
		field string
		want  string
	}{
		{"legacy", "", ""},
		{"empty", `,"difficulty":""`, ""},
		{"easy", `,"difficulty":"easy"`, "easy"},
		{"moderate", `,"difficulty":"moderate"`, "moderate"},
		{"difficult", `,"difficulty":"difficult"`, "difficult"},
	} {
		for _, kind := range []string{"planned", "completed"} {
			t.Run(test.name+"/"+kind, func(t *testing.T) {
				// Decode plugin JSON so omitted fields exercise older-plugin behavior.
				payload := fmt.Sprintf(`{"name":"Provider rating","kind":%q,"source":{"provider":"test-provider","externalId":%q},"metadata":{"difficulty":"difficult"}%s}`, kind, test.name+"-"+kind, test.field)
				var item pluginsystem.TrailImport
				if err := json.Unmarshal([]byte(payload), &item); err != nil {
					t.Fatal(err)
				}
				item.Track = providerDifficultyGPX()
				result, err := importer.ImportTrail(context.Background(), app, item, opts)
				if err != nil {
					t.Fatal(err)
				}
				if !result.Created || result.Skipped {
					t.Fatalf("first import = %#v, want new trail", result)
				}
				trail := providerDifficultyTrail(t, app, result.TrailID)
				if got := trail.GetString("difficulty"); got != test.want {
					t.Fatalf("difficulty = %q, want %q", got, test.want)
				}
				if trail.GetString("gpx") == "" {
					t.Fatal("import did not persist its GPX")
				}

				// Re-imports retain user-edited ratings, including unknown. They
				// are skipped even if the plugin now emits invalid fields or GPX.
				for _, savedDifficulty := range []string{"", "easy", "moderate", "difficult"} {
					trail.Set("difficulty", savedDifficulty)
					if err := app.Save(trail); err != nil {
						t.Fatal(err)
					}
					for _, incomingDifficulty := range []string{"", "easy", "difficult", "invalid"} {
						item.Difficulty = incomingDifficulty
						item.Track = pluginsystem.Track{Format: "invalid"}
						repeated, err := importer.ImportTrail(context.Background(), app, item, opts)
						if err != nil {
							t.Fatal(err)
						}
						if !repeated.Skipped || repeated.Created || repeated.TrailID != trail.Id {
							t.Fatalf("repeat import = %#v, want skipped existing trail", repeated)
						}
						if got := providerDifficultyTrail(t, app, trail.Id).GetString("difficulty"); got != savedDifficulty {
							t.Fatalf("repeat import changed difficulty from %q to %q", savedDifficulty, got)
						}
					}
				}
			})
		}
	}
	if got := providerDifficultyRecordCount(t, app, "trails"); got != 10 {
		t.Fatalf("trail count = %d, want 10 (duplicates must not create trails)", got)
	}
}

func TestImportProviderDifficultyRejectsInvalidBeforeGPXAndRecordCreation(t *testing.T) {
	app, opts := newProviderDifficultyApp(t)
	for _, difficulty := range []string{"unknown", "hard", "Easy", " easy", "easy ", "0"} {
		t.Run(difficulty, func(t *testing.T) {
			for _, track := range []pluginsystem.Track{providerDifficultyGPX(), {Format: "invalid"}} {
				item := pluginsystem.TrailImport{
					Name: "Invalid rating", Difficulty: difficulty,
					Source: pluginsystem.TrailImportSource{Provider: "test-provider", ExternalID: difficulty},
					Track:  track,
				}
				result, err := importer.ImportTrail(context.Background(), app, item, opts)
				if err == nil || !strings.Contains(err.Error(), "unsupported trail difficulty") || !strings.Contains(err.Error(), fmt.Sprintf("%q", difficulty)) {
					t.Fatalf("import error = %v, want difficulty validation before GPX processing", err)
				}
				if result != nil {
					t.Fatalf("invalid import returned result %#v", result)
				}
			}
		})
	}
	for _, collection := range []string{"trails", "trail_external_reference", "waypoints", "summit_logs"} {
		if got := providerDifficultyRecordCount(t, app, collection); got != 0 {
			t.Errorf("invalid import created %d records in %s", got, collection)
		}
	}
}

func providerDifficultyGPX() pluginsystem.Track {
	return pluginsystem.Track{
		Format: "gpx",
		ContentBase64: base64.StdEncoding.EncodeToString([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="test"><trk><trkseg>
<trkpt lat="46" lon="8"><ele>100</ele><time>2026-01-01T10:00:00Z</time></trkpt>
<trkpt lat="46.001" lon="8.001"><ele>120</ele><time>2026-01-01T10:10:00Z</time></trkpt>
</trkseg></trk></gpx>`)),
	}
}

func providerDifficultyTrail(t *testing.T, app core.App, id string) *core.Record {
	t.Helper()
	record, err := app.FindRecordById("trails", id)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func providerDifficultyRecordCount(t *testing.T, app core.App, collection string) int64 {
	t.Helper()
	count, err := app.CountRecords(collection)
	if err != nil {
		t.Fatal(err)
	}
	return count
}

// Apply production migrations, except those requiring an external search engine.
// Keep this fixture independent of the unknown-difficulty search-fix tests.
func newProviderDifficultyApp(t *testing.T) (*core.BaseApp, importer.Options) {
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
	user := saveProviderDifficultyRecord(t, app, "users", map[string]any{
		"username": "importer", "password": "test-password", "email": "importer@example.com",
	})
	author := saveProviderDifficultyRecord(t, app, "activitypub_actors", map[string]any{
		"username": "importer", "preferred_username": "importer", "domain": "example.com",
		"user": user.Id, "is_local": true, "public_key": "test-key",
		"iri": "https://example.com/importer", "inbox": "https://example.com/importer/inbox",
	})
	return app, importer.Options{UserID: user.Id, ActorID: author.Id}
}

func saveProviderDifficultyRecord(t *testing.T, app core.App, collection string, values map[string]any) *core.Record {
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
