package regions

import (
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func TestDisplayNameMigrationKeepsAdminNames(t *testing.T) {
	app := newRegionsTestApp(t)

	setName := func(path, name string) {
		t.Helper()
		if _, err := app.DB().NewQuery("UPDATE regions SET name = {:name} WHERE path = {:path}").
			Bind(dbx.Params{"name": name, "path": path}).Execute(); err != nil {
			t.Fatal(err)
		}
	}
	nameOf := func(path string) string {
		t.Helper()
		region, err := app.FindFirstRecordByFilter("regions", "path = {:path}", dbx.Params{"path": path})
		if err != nil {
			t.Fatal(err)
		}
		return region.GetString("name")
	}

	graz := "austria.austria_styria_graz"
	altai := "russian_federation.russia_altai_krai"
	if nameOf(graz) != "Styria — Graz" {
		t.Fatalf("fresh seed name for %s = %q", graz, nameOf(graz))
	}
	// An instance seeded before the display names, and an admin rename.
	setName(graz, "Styria_Graz")
	setName(altai, "Altai (renamed by admin)")

	migration := findMigration(t, "1791200100_update_region_display_names.go")
	if err := app.RunInTransaction(migration.Up); err != nil {
		t.Fatal(err)
	}

	if got := nameOf(graz); got != "Styria — Graz" {
		t.Errorf("%s name = %q, want the display name", graz, got)
	}
	if got := nameOf(altai); got != "Altai (renamed by admin)" {
		t.Errorf("%s name = %q, want the admin's name kept", altai, got)
	}
}

func findMigration(t *testing.T, file string) *core.Migration {
	t.Helper()
	for _, migration := range core.AppMigrations.Items() {
		if migration.File == file {
			return migration
		}
	}
	t.Fatalf("migration %s not registered", file)
	return nil
}
