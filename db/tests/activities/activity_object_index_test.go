package activities_test

import (
	"slices"
	"sort"
	"strings"
	"testing"

	_ "pocketbase/migrations"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const objectIndexName = "idx_activitypub_activities_object_id"

// newMigratedTestApp applies every migration to a temporary database, skipping
// only the migrations that configure the external Meilisearch indexes.
func newMigratedTestApp(t *testing.T) *core.BaseApp {
	t.Helper()
	t.Setenv("ORIGIN", "https://example.com")
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	migrations := slices.Clone(core.AppMigrations.Items())
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].File < migrations[j].File
	})
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

// TestHandedOutLookupUsesObjectIndex checks that the lookup a Delete of a
// private trail or list runs (federation.wasHandedOut) is answered from the
// object id index instead of scanning every recorded activity.
func TestHandedOutLookupUsesObjectIndex(t *testing.T) {
	app := newMigratedTestApp(t)

	collection, err := app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		t.Fatal(err)
	}
	if collection.GetIndex(objectIndexName) == "" {
		t.Fatalf("%s missing from activitypub_activities", objectIndexName)
	}

	type planRow struct {
		Detail string `db:"detail"`
	}
	var rows []planRow
	err = app.DB().NewQuery(`
		EXPLAIN QUERY PLAN
		SELECT EXISTS (
			SELECT 1
			FROM activitypub_activities
			WHERE json_extract(object, '$.id') = {:iri} AND type IN ('Create', 'Update')
		)
	`).Bind(dbx.Params{"iri": "https://example.com/api/v1/trail/t1"}).All(&rows)
	if err != nil {
		t.Fatal(err)
	}
	details := make([]string, 0, len(rows))
	for _, r := range rows {
		details = append(details, r.Detail)
	}
	plan := strings.Join(details, "\n")
	if !strings.Contains(plan, objectIndexName) {
		t.Fatalf("query plan does not use %s:\n%s", objectIndexName, plan)
	}
}
