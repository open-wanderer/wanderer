package permissions_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"pocketbase/hooks"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type ownershipFixture struct {
	app                                           *core.BaseApp
	handler                                       http.Handler
	owner, editor, stranger                       *core.Record
	ownerActor, editorActor, strangerActor        *core.Record
	trail, otherTrail, list, otherList            *core.Record
	comment, waypoint, log, instance              *core.Record
	trailShare, listShare, link, follow, settings *core.Record
	categoryPreference, subcategoryPreference     *core.Record
	otherCategory, otherSubcategory               *core.Record
}

func newOwnershipFixture(t *testing.T) ownershipFixture {
	t.Helper()
	app := newRulesTestApp(t)
	newUser := func(name string) *core.Record {
		return saveRulesTestRecord(t, app, "users", map[string]any{
			"username": name, "password": "test-password", "email": name + "@example.com",
		})
	}
	owner, editor, stranger := newUser("owner"), newUser("editor"), newUser("stranger")
	newActor := func(name, user string) *core.Record {
		return saveRulesTestRecord(t, app, "activitypub_actors", map[string]any{
			"username": name, "preferred_username": name, "domain": "example.com",
			"user": user, "is_local": true, "public_key": "test-key",
			"iri": "https://example.com/" + name, "inbox": "https://example.com/" + name + "/inbox",
		})
	}
	ownerActor, editorActor, strangerActor := newActor("owner", owner.Id), newActor("editor", editor.Id), newActor("stranger", stranger.Id)
	trail := saveRulesTestRecord(t, app, "trails", map[string]any{
		"name": "Owner trail", "author": ownerActor.Id, "public": false,
	})
	otherTrail := saveRulesTestRecord(t, app, "trails", map[string]any{
		"name": "Other private trail", "author": strangerActor.Id, "public": false,
	})
	list := saveRulesTestRecord(t, app, "lists", map[string]any{
		"name": "Owner list", "author": ownerActor.Id, "public": false, "trails": []string{trail.Id},
	})
	otherList := saveRulesTestRecord(t, app, "lists", map[string]any{
		"name": "Other private list", "author": strangerActor.Id, "public": false, "trails": []string{otherTrail.Id},
	})
	comment := saveRulesTestRecord(t, app, "comments", map[string]any{
		"author": ownerActor.Id, "trail": trail.Id, "text": "Original comment",
	})
	waypoint := saveRulesTestRecord(t, app, "waypoints", map[string]any{
		"name": "Original waypoint", "author": ownerActor.Id, "trail": trail.Id,
	})
	log := saveRulesTestRecord(t, app, "summit_logs", map[string]any{
		"author": ownerActor.Id, "trail": trail.Id, "text": "Original summit log",
	})
	instance := saveRulesTestRecord(t, app, "plugin_instances", map[string]any{
		"user": owner.Id, "plugin_id": "hammerhead", "status": "configured",
	})
	trailShare := saveRulesTestRecord(t, app, "trail_share", map[string]any{
		"trail": trail.Id, "actor": editorActor.Id, "permission": "edit",
	})
	listShare := saveRulesTestRecord(t, app, "list_share", map[string]any{
		"list": list.Id, "actor": editorActor.Id, "permission": "edit",
	})
	link := saveRulesTestRecord(t, app, "trail_link_share", map[string]any{
		"trail": trail.Id, "permission": "view", "token": strings.Repeat("a", 32),
	})
	follow := saveRulesTestRecord(t, app, "follows", map[string]any{
		"follower": ownerActor.Id, "followee": editorActor.Id, "status": "accepted",
	})
	settings := saveRulesTestRecord(t, app, "settings", map[string]any{
		"user": owner.Id, "language": "en", "unit": "metric", "mapFocus": "trails",
	})
	category := saveRulesTestRecord(t, app, "categories", map[string]any{"name": "Ownership category"})
	otherCategory := saveRulesTestRecord(t, app, "categories", map[string]any{"name": "Other ownership category"})
	subcategory := saveRulesTestRecord(t, app, "subcategories", map[string]any{"name": "Ownership subcategory", "category": category.Id})
	otherSubcategory := saveRulesTestRecord(t, app, "subcategories", map[string]any{"name": "Other ownership subcategory", "category": otherCategory.Id})
	categoryPreference := saveRulesTestRecord(t, app, "user_category_preferences", map[string]any{
		"user": owner.Id, "category": category.Id, "visible": true, "priority": 1,
	})
	subcategoryPreference := saveRulesTestRecord(t, app, "user_subcategory_preferences", map[string]any{
		"user": owner.Id, "subcategory": subcategory.Id, "visible": true, "priority": 1,
	})
	// Share request hooks stay unregistered to prove that the migrated rules
	// protect targets independently. Preference validators preserve the real
	// API contract for permitted updates.
	app.OnRecordUpdateRequest("user_category_preferences").BindFunc(hooks.ValidateUserCategoryPreferenceHandler())
	app.OnRecordUpdateRequest("user_subcategory_preferences").BindFunc(hooks.ValidateUserSubcategoryPreferenceHandler())
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	return ownershipFixture{
		app: app, handler: mux, owner: owner, editor: editor, stranger: stranger,
		ownerActor: ownerActor, editorActor: editorActor, strangerActor: strangerActor,
		trail: trail, otherTrail: otherTrail, list: list, otherList: otherList,
		comment: comment, waypoint: waypoint, log: log, instance: instance,
		trailShare: trailShare, listShare: listShare, link: link, follow: follow, settings: settings,
		categoryPreference: categoryPreference, subcategoryPreference: subcategoryPreference,
		otherCategory: otherCategory, otherSubcategory: otherSubcategory,
	}
}

func ownershipRequest(t *testing.T, handler http.Handler, record, auth *core.Record, method, format string, body map[string]any, shareToken ...string) *httptest.ResponseRecorder {
	t.Helper()
	var data bytes.Buffer
	contentType := "application/json"
	if format == "json" {
		if err := json.NewEncoder(&data).Encode(body); err != nil {
			t.Fatal(err)
		}
	} else {
		writer := multipart.NewWriter(&data)
		if format == "multipart-json" {
			payload, err := json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
			if err := writer.WriteField("@jsonPayload", string(payload)); err != nil {
				t.Fatal(err)
			}
		} else {
			for key, value := range body {
				if err := writer.WriteField(key, fmt.Sprint(value)); err != nil {
					t.Fatal(err)
				}
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		contentType = writer.FormDataContentType()
	}
	path := "/api/collections/" + record.Collection().Name + "/records/" + record.Id
	if len(shareToken) > 0 {
		path += "?share=" + url.QueryEscape(shareToken[0])
	}
	request := httptest.NewRequest(method, path, &data)
	request.Header.Set("Content-Type", contentType)
	if auth != nil {
		token, err := auth.NewAuthToken()
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestImmutableOwnershipAPI(t *testing.T) {
	f := newOwnershipFixture(t)
	for _, target := range []struct {
		record      *core.Record
		field       string
		replacement string
	}{
		{f.trail, "author", f.editorActor.Id},
		{f.list, "author", f.editorActor.Id},
		{f.comment, "author", f.editorActor.Id},
		{f.comment, "trail", f.otherTrail.Id},
		{f.waypoint, "author", f.editorActor.Id},
		{f.waypoint, "trail", f.otherTrail.Id},
		{f.log, "author", f.editorActor.Id},
		{f.log, "trail", f.otherTrail.Id},
		{f.instance, "user", f.editor.Id},
		{f.instance, "plugin_id", "komoot"},
		{f.trailShare, "trail", f.otherTrail.Id},
		{f.listShare, "list", f.otherList.Id},
		{f.link, "trail", f.otherTrail.Id},
		{f.follow, "follower", f.editorActor.Id},
		{f.settings, "user", f.editor.Id},
		{f.categoryPreference, "user", f.editor.Id},
		{f.subcategoryPreference, "user", f.editor.Id},
	} {
		for _, format := range []string{"json", "multipart", "multipart-json"} {
			original := target.record.GetString(target.field)
			mutations := []struct {
				name string
				body map[string]any
			}{
				{"replace", map[string]any{target.field: target.replacement}},
				{"append", map[string]any{target.field + "+": target.replacement}},
				{"remove", map[string]any{target.field + "-": original}},
				{"prefix-append", map[string]any{"+" + target.field: target.replacement}},
				{"prefix-remove", map[string]any{"-" + target.field: original}},
				{"echo-and-remove", map[string]any{target.field: original, target.field + "-": original}},
			}
			content := map[string]map[string]any{
				"trail_share":                  {"actor": f.strangerActor.Id, "permission": "view"},
				"list_share":                   {"actor": f.strangerActor.Id, "permission": "view"},
				"trail_link_share":             {"token": strings.Repeat("b", 32), "permission": "edit"},
				"follows":                      {"followee": f.strangerActor.Id, "status": "pending"},
				"settings":                     {"language": "de", "unit": "imperial"},
				"user_category_preferences":    {"category": f.otherCategory.Id, "visible": false},
				"user_subcategory_preferences": {"subcategory": f.otherSubcategory.Id, "visible": false},
			}[target.record.Collection().Name]
			if content != nil {
				body := map[string]any{target.field: target.replacement}
				for key, value := range content {
					body[key] = value
				}
				mutations = append(mutations, struct {
					name string
					body map[string]any
				}{"replace-with-content", body})
			}
			for _, mutation := range mutations {
				t.Run(target.record.Collection().Name+"/"+target.field+"/"+format+"/"+mutation.name, func(t *testing.T) {
					response := ownershipRequest(t, f.handler, target.record, f.owner, http.MethodPatch, format, mutation.body)
					if strings.HasPrefix(mutation.name, "replace") && response.Code != http.StatusNotFound {
						t.Errorf("status = %d; want 404: %s", response.Code, response.Body.String())
					}
					fresh, err := f.app.FindRecordById(target.record.Collection().Name, target.record.Id)
					if err != nil {
						t.Fatal(err)
					}
					// PocketBase may ignore an unsupported modifier. A 200 no-op is
					// harmless, but no representation may change the protected field.
					if got := fresh.GetString(target.field); got != original {
						t.Fatalf("protected field changed from %q to %q (HTTP %d)", original, got, response.Code)
					}
					for key := range content {
						if got, want := fresh.Get(key), target.record.Get(key); !reflect.DeepEqual(got, want) {
							t.Errorf("rejected binding change altered %s: got %v; want %v", key, got, want)
						}
					}
					if target.record.Collection().Name == "trail_share" || target.record.Collection().Name == "list_share" || target.record.Collection().Name == "trail_link_share" {
						assertOwnershipShareAccess(t, f, target.record)
					}
				})
			}
		}
	}
}

func assertOwnershipShareAccess(t *testing.T, f ownershipFixture, share *core.Record) {
	t.Helper()
	original, foreign, reader := f.trail, f.otherTrail, f.editor
	var token []string
	if share.Collection().Name == "list_share" {
		original, foreign = f.list, f.otherList
	} else if share.Collection().Name == "trail_link_share" {
		reader = nil
		token = []string{share.GetString("token")}
	}
	for _, check := range []struct {
		record *core.Record
		status int
	}{
		{original, http.StatusOK},
		{foreign, http.StatusNotFound},
	} {
		response := ownershipRequest(t, f.handler, check.record, reader, http.MethodGet, "json", nil, token...)
		if response.Code != check.status {
			t.Errorf("share recipient GET %s: status = %d; want %d: %s", check.record.GetString("name"), response.Code, check.status, response.Body.String())
		}
	}
	for _, auth := range []*core.Record{f.owner, f.editor} {
		response := ownershipRequest(t, f.handler, foreign, auth, http.MethodPatch, "json", map[string]any{"name": "Unauthorized foreign edit"})
		if response.Code != http.StatusNotFound {
			t.Errorf("foreign object update by %s: status = %d; want 404: %s", auth.GetString("username"), response.Code, response.Body.String())
		}
	}
}

func TestImmutableOwnershipKeepsAuthorizedUpdates(t *testing.T) {
	f := newOwnershipFixture(t)
	for _, record := range []*core.Record{f.trail, f.list} {
		for _, format := range []string{"json", "multipart", "multipart-json"} {
			t.Run("edit-share/"+record.Collection().Name+"/"+format, func(t *testing.T) {
				body := map[string]any{"name": "Shared edit", "author": f.ownerActor.Id}
				response := ownershipRequest(t, f.handler, record, f.editor, http.MethodPatch, format, body)
				if response.Code != http.StatusOK {
					t.Fatalf("edit share status = %d; want 200: %s", response.Code, response.Body.String())
				}
				body["author"] = f.editorActor.Id
				response = ownershipRequest(t, f.handler, record, f.editor, http.MethodPatch, format, body)
				if response.Code != http.StatusNotFound {
					t.Fatalf("takeover status = %d; want 404: %s", response.Code, response.Body.String())
				}
				response = ownershipRequest(t, f.handler, record, f.editor, http.MethodDelete, "json", nil)
				if response.Code != http.StatusNotFound {
					t.Fatalf("delete after takeover status = %d; want 404", response.Code)
				}
				fresh, err := f.app.FindRecordById(record.Collection().Name, record.Id)
				if err != nil || fresh.GetString("author") != f.ownerActor.Id || fresh.GetString("name") != "Shared edit" {
					t.Fatalf("shared content did not retain its owner and authorized edit: %v", err)
				}
				response = ownershipRequest(t, f.handler, record, f.stranger, http.MethodPatch, format, map[string]any{"name": "Unauthorized"})
				if response.Code != http.StatusNotFound {
					t.Fatalf("stranger status = %d; want 404", response.Code)
				}
			})
		}
	}
	for _, target := range []struct {
		record   *core.Record
		body     map[string]any
		bindings map[string]any
	}{
		{f.trail, map[string]any{"name": "Updated trail"}, map[string]any{"author": f.ownerActor.Id}},
		{f.list, map[string]any{"name": "Updated list"}, map[string]any{"author": f.ownerActor.Id}},
		{f.comment, map[string]any{"text": "Updated comment"}, map[string]any{"author": f.ownerActor.Id, "trail": f.trail.Id}},
		{f.waypoint, map[string]any{"name": "Updated waypoint"}, map[string]any{"author": f.ownerActor.Id, "trail": f.trail.Id}},
		{f.log, map[string]any{"text": "Updated log"}, map[string]any{"author": f.ownerActor.Id, "trail": f.trail.Id}},
		{f.instance, map[string]any{"status": "disabled"}, map[string]any{"user": f.owner.Id, "plugin_id": "hammerhead"}},
		{f.trailShare, map[string]any{"actor": f.strangerActor.Id, "permission": "view"}, map[string]any{"trail": f.trail.Id}},
		{f.listShare, map[string]any{"actor": f.strangerActor.Id, "permission": "view"}, map[string]any{"list": f.list.Id}},
		{f.link, map[string]any{"token": strings.Repeat("b", 32), "permission": "edit"}, map[string]any{"trail": f.trail.Id}},
		{f.follow, map[string]any{"followee": f.strangerActor.Id, "status": "pending"}, map[string]any{"follower": f.ownerActor.Id}},
		{f.settings, map[string]any{"language": "de", "unit": "imperial", "mapFocus": "location"}, map[string]any{"user": f.owner.Id}},
		{f.categoryPreference, map[string]any{"category": f.otherCategory.Id, "visible": false}, map[string]any{"user": f.owner.Id}},
		{f.subcategoryPreference, map[string]any{"subcategory": f.otherSubcategory.Id, "visible": false}, map[string]any{"user": f.owner.Id}},
	} {
		for _, format := range []string{"json", "multipart", "multipart-json"} {
			for _, bindingMode := range []string{"omitted", "unchanged"} {
				t.Run(target.record.Collection().Name+"/"+format+"/"+bindingMode, func(t *testing.T) {
					body := make(map[string]any, len(target.body)+len(target.bindings))
					for key, value := range target.body {
						body[key] = value
					}
					if bindingMode == "unchanged" {
						for key, value := range target.bindings {
							body[key] = value
						}
					}
					response := ownershipRequest(t, f.handler, target.record, f.owner, http.MethodPatch, format, body)
					if response.Code != http.StatusOK {
						t.Fatalf("status = %d; want 200: %s", response.Code, response.Body.String())
					}
					fresh, err := f.app.FindRecordById(target.record.Collection().Name, target.record.Id)
					if err != nil {
						t.Fatal(err)
					}
					for _, values := range []map[string]any{target.body, target.bindings} {
						for key, value := range values {
							if got := fresh.Get(key); !reflect.DeepEqual(got, value) {
								t.Errorf("%s = %v; want %v", key, got, value)
							}
						}
					}
					if (fresh.Collection().Name == "user_category_preferences" || fresh.Collection().Name == "user_subcategory_preferences") && fresh.GetInt("priority") != 1 {
						t.Errorf("preference priority = %d; want 1", fresh.GetInt("priority"))
					}
				})
			}
		}
	}
}

func TestImmutableOwnershipRulesKeepTrustedSavesAndSuperusers(t *testing.T) {
	f := newOwnershipFixture(t)
	// Merge and federation paths use model saves rather than the records API.
	f.log.Set("author", f.editorActor.Id)
	f.log.Set("trail", f.otherTrail.Id)
	if err := f.app.Save(f.log); err != nil {
		t.Fatalf("trusted model save: %v", err)
	}
	fresh, err := f.app.FindRecordById("summit_logs", f.log.Id)
	if err != nil || fresh.GetString("author") != f.editorActor.Id || fresh.GetString("trail") != f.otherTrail.Id {
		t.Fatalf("trusted model save did not move the record: %v", err)
	}
	superuser := saveRulesTestRecord(t, f.app, "_superusers", map[string]any{
		"email": "admin@example.com", "password": "test-password",
	})
	// This checks PocketBase's rule bypass only. Production share request hooks
	// still reject target changes, including requests made by superusers.
	for _, target := range []struct {
		record *core.Record
		body   map[string]any
	}{
		{f.trail, map[string]any{"author": f.editorActor.Id}},
		{f.instance, map[string]any{"user": f.editor.Id, "plugin_id": "komoot"}},
		{f.trailShare, map[string]any{"trail": f.otherTrail.Id}},
		{f.listShare, map[string]any{"list": f.otherList.Id}},
		{f.link, map[string]any{"trail": f.otherTrail.Id}},
		{f.follow, map[string]any{"follower": f.editorActor.Id}},
		{f.settings, map[string]any{"user": f.editor.Id}},
		{f.categoryPreference, map[string]any{"user": f.editor.Id}},
		{f.subcategoryPreference, map[string]any{"user": f.editor.Id}},
	} {
		response := ownershipRequest(t, f.handler, target.record, superuser, http.MethodPatch, "json", target.body)
		if response.Code != http.StatusOK {
			t.Fatalf("superuser %s update status = %d; want 200: %s", target.record.Collection().Name, response.Code, response.Body.String())
		}
		fresh, err := f.app.FindRecordById(target.record.Collection().Name, target.record.Id)
		if err != nil {
			t.Fatal(err)
		}
		for key, value := range target.body {
			if got := fresh.GetString(key); got != value {
				t.Errorf("superuser %s = %q; want %q", key, got, value)
			}
		}
	}
}

func TestImmutableOwnershipMigrationIsIdempotentAndKeepsLockedRules(t *testing.T) {
	app := newRulesTestApp(t)
	targets := []struct {
		collection string
		fields     []string
	}{
		{"trails", []string{"author"}},
		{"lists", []string{"author"}},
		{"comments", []string{"author", "trail"}},
		{"waypoints", []string{"author", "trail"}},
		{"summit_logs", []string{"author", "trail"}},
		{"plugin_instances", []string{"user", "plugin_id"}},
		{"trail_share", []string{"trail"}},
		{"list_share", []string{"list"}},
		{"trail_link_share", []string{"trail"}},
		{"follows", []string{"follower"}},
		{"settings", []string{"user"}},
		{"user_category_preferences", []string{"user"}},
		{"user_subcategory_preferences", []string{"user"}},
	}
	before := make(map[string]string, len(targets))
	for _, target := range targets {
		collection, err := app.FindCollectionByNameOrId(target.collection)
		if err != nil {
			t.Fatal(err)
		}
		if collection.UpdateRule == nil {
			t.Fatalf("%s update rule unexpectedly locked", target.collection)
		}
		before[target.collection] = *collection.UpdateRule
	}
	var apply func(core.App) error
	for _, migration := range core.AppMigrations.Items() {
		if migration.File == "1791110001_immutable_record_ownership.go" {
			apply = migration.Up
			break
		}
	}
	if apply == nil {
		t.Fatal("ownership migration not registered")
	}
	if err := apply(app); err != nil {
		t.Fatal(err)
	}
	for _, target := range targets {
		t.Run("idempotent/"+target.collection, func(t *testing.T) {
			collection, err := app.FindCollectionByNameOrId(target.collection)
			if err != nil {
				t.Fatal(err)
			}
			if collection.UpdateRule == nil || *collection.UpdateRule != before[target.collection] {
				t.Fatal("migration changed the already guarded rule")
			}
			for _, field := range target.fields {
				if strings.Count(*collection.UpdateRule, "@request.body."+field+":changed") != 1 {
					t.Errorf("%s ownership guard must occur once", field)
				}
			}
			collection.UpdateRule = nil
			if err := app.Save(collection); err != nil {
				t.Fatal(err)
			}
		})
	}
	if err := apply(app); err != nil {
		t.Fatal(err)
	}
	for _, target := range targets {
		t.Run("locked/"+target.collection, func(t *testing.T) {
			collection, err := app.FindCollectionByNameOrId(target.collection)
			if err != nil {
				t.Fatal(err)
			}
			if collection.UpdateRule != nil {
				t.Fatal("migration unlocked a locked API")
			}
		})
	}
}
