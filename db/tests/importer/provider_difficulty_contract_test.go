package importer_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"testing"

	_ "pocketbase/migrations"
	"pocketbase/plugins/importer"
	"pocketbase/pluginsystem"

	"github.com/pocketbase/pocketbase/core"
)

func TestImportProviderDifficultyContract(t *testing.T) {
	app, opts := newProviderDifficultyApp(t)
	tests := []struct {
		name  string
		field string
		want  string
	}{
		{"legacy", "", ""},
		{"empty", `,"difficulty":""`, ""},
		{"null", `,"difficulty":null`, ""},
		{"easy", `,"difficulty":"easy"`, "easy"},
		{"moderate", `,"difficulty":"moderate"`, "moderate"},
		{"difficult", `,"difficulty":"difficult"`, "difficult"},
		{"invalid string", `,"difficulty":"hard"`, ""},
		{"wrong case", `,"difficulty":"Easy"`, ""},
	}
	for _, test := range tests {
		for _, kind := range []string{"planned", "completed"} {
			t.Run(test.name+"/"+kind, func(t *testing.T) {
				// Legacy metadata is deliberately different from the canonical field:
				// it must neither supply a missing rating nor override a valid one.
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
			})
		}
	}
	if got, want := providerDifficultyRecordCount(t, app, "trails"), int64(2*len(tests)); got != want {
		t.Fatalf("trail count = %d, want %d", got, want)
	}
}

func TestImportProviderDifficultyIgnoresInvalidStrings(t *testing.T) {
	app, opts := newProviderDifficultyApp(t)
	for _, difficulty := range []string{"unknown", "hard", "Easy", " easy", "easy ", "0"} {
		t.Run(difficulty, func(t *testing.T) {
			// Construct the item directly: normalization is an importer guarantee,
			// rather than only a side effect of JSON decoding.
			item := pluginsystem.TrailImport{
				Name: "Invalid rating", Difficulty: difficulty,
				Source:   pluginsystem.TrailImportSource{Provider: "test-provider", ExternalID: difficulty},
				Track:    providerDifficultyGPX(),
				Metadata: map[string]any{"difficulty": "easy"},
			}
			result, err := importer.ImportTrail(context.Background(), app, item, opts)
			if err != nil {
				t.Fatalf("invalid optional rating must not block the import: %v", err)
			}
			if !result.Created || result.Skipped {
				t.Fatalf("import = %#v, want new trail", result)
			}
			if got := providerDifficultyTrail(t, app, result.TrailID).GetString("difficulty"); got != "" {
				t.Fatalf("invalid rating stored as %q, want unknown", got)
			}
		})
	}
}

func TestImportProviderDifficultyPreservesExistingTrails(t *testing.T) {
	app, opts := newProviderDifficultyApp(t)
	for _, savedDifficulty := range []string{"", "easy", "moderate", "difficult"} {
		t.Run(savedDifficulty, func(t *testing.T) {
			item := pluginsystem.TrailImport{
				Name: "Original name", Difficulty: savedDifficulty,
				Source: pluginsystem.TrailImportSource{Provider: "test-provider", ExternalID: "existing-" + savedDifficulty},
				Track:  providerDifficultyGPX(),
			}
			original, err := importer.ImportTrail(context.Background(), app, item, opts)
			if err != nil {
				t.Fatal(err)
			}
			// Preserve all existing ratings, including unknown. Duplicates still
			// skip GPX processing even if the new rating or track is invalid.
			item.Track = pluginsystem.Track{Format: "invalid"}
			for _, incomingDifficulty := range []string{"", "easy", "moderate", "difficult", "hard", "Easy"} {
				item.Difficulty = incomingDifficulty
				result, err := importer.ImportTrail(context.Background(), app, item, opts)
				if err != nil {
					t.Fatal(err)
				}
				if result.Created || !result.Skipped || result.TrailID != original.TrailID {
					t.Fatalf("duplicate result = %#v, want skipped original trail", result)
				}
				if got := providerDifficultyTrail(t, app, original.TrailID).GetString("difficulty"); got != savedDifficulty {
					t.Fatalf("incoming %q changed existing difficulty from %q to %q", incomingDifficulty, savedDifficulty, got)
				}
			}
		})
	}
	if got := providerDifficultyRecordCount(t, app, "trails"); got != 4 {
		t.Fatalf("trail count = %d, want 4 (duplicates must not create trails)", got)
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
