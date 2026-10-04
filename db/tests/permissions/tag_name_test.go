package permissions_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func TestTagSearchQuotedValuesKeepFilterLiteral(t *testing.T) {
	app := newRulesTestApp(t)
	owner := saveRulesTestRecord(t, app, "users", map[string]any{
		"username": "tagsearch", "password": "test-password", "email": "tagsearch@example.com",
	})
	token, err := owner.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	names := []string{"Grüezi 🌍", "O'Brien", "x' || name != '' || name~'", `x" || name != "" || name~"`, `tag\name`}
	ids := map[string]string{}
	for _, name := range names {
		ids[name] = saveRulesTestRecord(t, app, "tags", map[string]any{"name": name}).Id
	}
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range names {
		// PocketBase's JS parameter builder produces a JSON-quoted string;
		// exercise that representation against the real backend filter parser.
		query := url.Values{"filter": {"name ~ " + strconv.Quote(name)}}
		request := httptest.NewRequest(http.MethodGet, "/api/collections/tags/records?"+query.Encode(), nil)
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("quoted tag query failed: HTTP %d", response.Code)
		}
		var result struct{ Items []struct{ ID string } }
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if len(result.Items) != 1 || result.Items[0].ID != ids[name] {
			t.Error("quoted tag query matched other names or lost its literal value")
		}
	}
}

func TestTagNameSchemaKeepsUnicodeAndPlainMarkup(t *testing.T) {
	app := newRulesTestApp(t)
	collection, err := app.FindCollectionByNameOrId("tags")
	if err != nil {
		t.Fatal(err)
	}
	field := collection.Fields.GetByName("name").(*core.TextField)
	if field.Max != 5000 || field.Required || field.Min != 0 {
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
	}
	ids := make([]string, 0, len(legacy))
	expected := map[string]string{}
	for _, item := range legacy {
		record := saveRulesTestRecord(t, app, "tags", map[string]any{"name": item.before})
		ids = append(ids, record.Id)
		expected[record.Id] = item.after
	}
	// Cross the backfill's batch boundary with more old names, without changing
	// or merging IDs when cleanup results in duplicate or empty names.
	for i := range 205 {
		name := fmt.Sprintf("tag\t%03d", i)
		record := saveRulesTestRecord(t, app, "tags", map[string]any{"name": name})
		expected[record.Id] = strings.ReplaceAll(name, "\t", "")
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
	var apply func(core.App) error
	for _, migration := range core.AppMigrations.Items() {
		if migration.File == "1791110002_tag_name_text.go" {
			apply = migration.Up
			break
		}
	}
	if apply == nil {
		t.Fatal("tag migration is not registered")
	}
	for range 2 {
		if err := apply(app); err != nil {
			t.Fatal(err)
		}
		records, err := app.FindAllRecords("tags")
		if err != nil || len(records) != len(expected) {
			t.Fatalf("tag record count changed: %d, %v", len(records), err)
		}
		for _, record := range records {
			if record.GetString("name") != expected[record.Id] {
				t.Errorf("migration changed a legitimate name or failed control cleanup for %s", record.Id)
			}
		}
		fresh, err := app.FindRecordById("trails", trail.Id)
		if err != nil || !slices.Equal(fresh.GetStringSlice("tags"), ids) {
			t.Fatalf("migration changed trail tag links: %v", err)
		}
	}
}
