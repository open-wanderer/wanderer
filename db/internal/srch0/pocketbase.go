package srch0

import (
	"fmt"
	"slices"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	_ "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

var externalSettingsMigrations = [...]string{
	"1742167033_init_meilisearch.go",
	"1744651602_add_polyline.go",
	"1749831369_update_sortable_attributes.go",
}

// Verify the complete exclusion list before any migration executes. A renamed
// external migration must fail here instead of reaching a developer's engine.
func databaseMigrations(registered []*core.Migration) (core.MigrationsList, error) {
	var missing []string
	for _, name := range externalSettingsMigrations {
		if !slices.ContainsFunc(registered, func(migration *core.Migration) bool { return migration.File == name }) {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return core.MigrationsList{}, fmt.Errorf("SRCH0: ausgeschlossene externe Migration nicht registriert: %s; Migrationsnamen und Ausschlussliste vor der Ausführung prüfen", strings.Join(missing, ", "))
	}
	migrations := core.MigrationsList{}
	for _, migration := range registered {
		if !slices.Contains(externalSettingsMigrations[:], migration.File) {
			migrations.Add(migration)
		}
	}
	if len(migrations.Items()) == 0 {
		return core.MigrationsList{}, fmt.Errorf("SRCH0: keine Repository-Datenbankmigrationen registriert")
	}
	return migrations, nil
}

// App runs the repository's database migrations in a fresh temporary database.
// Callers import pocketbase/migrations. Three old migrations touch only external
// Meilisearch settings and capture its URL at package initialization; those are
// excluded here. Current engine settings are exercised by TestSRCH0Startup and
// the real-engine matrix instead. Every collection and validation rule comes
// from the actual database migrations, including the users auth collection.
func App(t TestingT, d Dataset) *core.BaseApp {
	t.Helper()
	migrations, err := databaseMigrations(core.AppMigrations.Items())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("ORIGIN", "https://wanderer.invalid")
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})
	if _, err := core.NewMigrationsRunner(app, migrations).Up(); err != nil {
		t.Fatal(err)
	}
	// Replace seeded taxonomy records, retaining their real schema and constraints.
	for _, name := range []string{"subcategories", "categories"} {
		records, err := app.FindAllRecords(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, record := range records {
			if err := app.Delete(record); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, name := range []string{"users", "activitypub_actors", "categories", "subcategories", "tags", "trails", "lists", "trail_share", "trail_like", "list_share"} {
		collection, err := app.FindCollectionByNameOrId(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, values := range d.Records[name] {
			// An unknown enum is a raw defensive probe, not a valid stored trail.
			if name == "trails" && values["id"] == "difficulty-unknown" {
				if field, ok := collection.Fields.GetByName("difficulty").(*core.SelectField); ok && !slices.Contains(field.Values, "unknown") {
					continue
				}
			}
			r := core.NewRecord(collection)
			Assign(t, r, values)
			if name == "users" {
				r.SetPassword("srch0-synthetic-password")
			}
			if name == "activitypub_actors" {
				r.Set("public_key", "synthetic-public-key-not-used-for-signatures")
			}
			if err := app.Save(r); err != nil {
				t.Fatalf("seed %s/%v: %v", name, values["id"], err)
			}
		}
	}
	t.Logf("PocketBase schema: %d repository migrations; validated records; %d external settings migrations excluded", len(migrations.Items()), len(externalSettingsMigrations))
	return app
}

func Assign(t TestingT, record *core.Record, values Object) {
	t.Helper()
	for key, value := range values {
		if key == "id" {
			record.Id = ID(value.(string))
			continue
		}
		if _, ok := record.Collection().Fields.GetByName(key).(*core.AutodateField); ok {
			// PocketBase explicitly supports SetRaw for historical/autodate
			// imports. Freeze fixture time without replacing the real field.
			date, err := types.ParseDateTime(value)
			if err != nil {
				t.Fatal(err)
			}
			record.SetRaw(key, date)
			continue
		}
		if _, ok := record.Collection().Fields.GetByName(key).(*core.RelationField); ok {
			switch ids := value.(type) {
			case string:
				if ids != "" {
					value = ID(ids)
				}
			case []any:
				mapped := make([]string, len(ids))
				for i, id := range ids {
					mapped[i] = ID(id.(string))
				}
				value = mapped
			case []string:
				mapped := make([]string, len(ids))
				for i, id := range ids {
					mapped[i] = ID(id)
				}
				value = mapped
			}
		}
		if record.Collection().Name == "feed" && key == "item" {
			value = ID(fmt.Sprint(value))
		}
		record.Set(key, value)
	}
}
