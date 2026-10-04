package permissions_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestTagNameSchemaKeepsUnicodeAndPlainMarkup(t *testing.T) {
	app := newRulesTestApp(t)
	collection, err := app.FindCollectionByNameOrId("tags")
	if err != nil {
		t.Fatal(err)
	}
	field := collection.Fields.GetByName("name").(*core.TextField)
	if field.Max != 5000 || field.Pattern != `^[^\x00-\x1f\x7f]*$` || field.Required || field.Min != 0 || len(collection.Indexes) != 0 {
		t.Fatalf("unexpected tag length/empty-name contract: %#v", field)
	}
	for _, name := range []string{"Grüezi", "日本語", "مرحبا", "e\u0301", "👩‍👩‍👧‍👦", "[route].* (test)", "O'Brien", "<img src=x>", "&lt;img&gt;", "", strings.Repeat("🌍", 5000)} {
		record := saveRulesTestRecord(t, app, "tags", map[string]any{"name": name})
		fresh, err := app.FindRecordById("tags", record.Id)
		if err != nil || fresh.GetString("name") != name {
			t.Fatalf("valid tag text changed: %v", err)
		}
	}
	for _, name := range []string{"null\x00byte", "line\nfeed", "tab\tname", "del\x7f", "trailing\n", "trailing\r", "\n", strings.Repeat("🌍", 5001)} {
		record := core.NewRecord(collection)
		record.Set("name", name)
		if err := app.Save(record); err == nil {
			t.Error("invalid tag name was accepted")
		}
	}
}

func TestTagNameMigrationPreservesIDsRelationsAndLegitimateNames(t *testing.T) {
	app := newRulesTestApp(t)
	collection, err := app.FindCollectionByNameOrId("tags")
	if err != nil {
		t.Fatal(err)
	}
	// Reproduce the previous schema, then apply the new migration to old data.
	field := collection.Fields.GetByName("name").(*core.TextField)
	field.Max, field.Pattern = 0, ""
	if err := app.Save(collection); err != nil {
		t.Fatal(err)
	}
	legacy := []struct{ before, after string }{
		{"Grüezi 🌍", "Grüezi 🌍"}, {"e\u0301 👩‍👩‍👧‍👦 مرحبا", "e\u0301 👩‍👩‍👧‍👦 مرحبا"},
		{"<img src=x>", "<img src=x>"}, {"&lt;img&gt;", "&lt;img&gt;"},
		{"", ""}, {"\n\t\x7f", ""}, {"same", "same"}, {"\x00same", "same"},
		{"Grü\nnezi", "Grünezi"},
		{" leading and trailing spaces ", " leading and trailing spaces "},
		{strings.Repeat("🌍", 5000), strings.Repeat("🌍", 5000)},
	}
	ids := make([]string, 0, len(legacy))
	expected := map[string]string{}
	updated := map[string]string{}
	for _, item := range legacy {
		record := saveRulesTestRecord(t, app, "tags", map[string]any{"name": item.before})
		ids = append(ids, record.Id)
		expected[record.Id] = item.after
		updated[record.Id] = record.GetString("updated")
	}
	// Cross the backfill's batch boundary with more old names, without changing
	// or merging IDs when cleanup results in duplicate or empty names.
	for i := range 205 {
		name := fmt.Sprintf("tag\t%03d", i)
		record := saveRulesTestRecord(t, app, "tags", map[string]any{"name": name})
		expected[record.Id] = strings.ReplaceAll(name, "\t", "")
		updated[record.Id] = record.GetString("updated")
	}
	owner := saveRulesTestRecord(t, app, "users", map[string]any{
		"username": "owner", "password": "test-password", "email": "owner@example.com",
	})
	actor := saveRulesTestRecord(t, app, "activitypub_actors", map[string]any{
		"username": "owner", "preferred_username": "owner", "domain": "example.com", "user": owner.Id,
		"public_key": "test-key", "is_local": true, "iri": "https://example.com/owner", "inbox": "https://example.com/owner/inbox",
	})
	trail := saveRulesTestRecord(t, app, "trails", map[string]any{
		"name": "Tagged trail", "author": actor.Id, "public": true, "tags": ids,
	})
	var apply, rollback func(core.App) error
	for _, migration := range core.AppMigrations.Items() {
		if migration.File == "1791110002_tag_name_text.go" {
			apply = migration.Up
			rollback = migration.Down
			break
		}
	}
	if apply == nil || rollback == nil {
		t.Fatal("tag migration is not registered")
	}
	assertStoredData := func(stage string) {
		t.Helper()
		records, err := app.FindAllRecords("tags")
		if err != nil || len(records) != len(expected) {
			t.Fatalf("%s: tag record count changed: %d, %v", stage, len(records), err)
		}
		for _, record := range records {
			want, exists := expected[record.Id]
			if !exists {
				t.Errorf("%s: migration introduced an unexpected tag ID %s", stage, record.Id)
				continue
			}
			if record.GetString("name") != want {
				t.Errorf("%s: migration changed name bytes or failed control cleanup for %s", stage, record.Id)
			}
			if record.GetString("updated") != updated[record.Id] {
				t.Errorf("%s: migration changed the historical updated date for %s", stage, record.Id)
			}
		}
		fresh, err := app.FindRecordById("trails", trail.Id)
		if err != nil || !slices.Equal(fresh.GetStringSlice("tags"), ids) {
			t.Fatalf("%s: migration changed trail tag links: %v", stage, err)
		}
	}
	for range 2 {
		if err := apply(app); err != nil {
			t.Fatal(err)
		}
		assertStoredData("up")
	}
	if err := rollback(app); err != nil {
		t.Fatal(err)
	}
	collection, err = app.FindCollectionByNameOrId("tags")
	if err != nil {
		t.Fatal(err)
	}
	field = collection.Fields.GetByName("name").(*core.TextField)
	if field.Max != 0 || field.Pattern != "" || field.Required || field.Min != 0 || len(collection.Indexes) != 0 {
		t.Fatalf("down did not restore the optional/non-unique original tag schema: %#v", field)
	}
	// Down restores validation, never control bytes that Up already removed.
	assertStoredData("down")
	postDownName := "post\tdown\x7f"
	postDown := saveRulesTestRecord(t, app, "tags", map[string]any{"name": postDownName})
	expected[postDown.Id] = postDownName
	updated[postDown.Id] = postDown.GetString("updated")
	assertStoredData("post-down write")
	if err := apply(app); err != nil {
		t.Fatal(err)
	}
	expected[postDown.Id] = "postdown"
	assertStoredData("up after down")
}
