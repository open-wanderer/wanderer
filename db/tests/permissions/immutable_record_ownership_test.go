package permissions_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type ownershipFixture struct {
	app                              *core.BaseApp
	handler                          http.Handler
	owner, editor, stranger          *core.Record
	ownerActor, editorActor          *core.Record
	trail, otherTrail, list          *core.Record
	comment, waypoint, log, instance *core.Record
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
	ownerActor, editorActor := newActor("owner", owner.Id), newActor("editor", editor.Id)
	trail := saveRulesTestRecord(t, app, "trails", map[string]any{
		"name": "Owner trail", "author": ownerActor.Id, "public": true,
	})
	otherTrail := saveRulesTestRecord(t, app, "trails", map[string]any{
		"name": "Other trail", "author": editorActor.Id, "public": true,
	})
	list := saveRulesTestRecord(t, app, "lists", map[string]any{
		"name": "Owner list", "author": ownerActor.Id, "public": true, "trails": []string{trail.Id},
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
	for collection, data := range map[string]map[string]any{
		"trail_share": {"trail": trail.Id, "actor": editorActor.Id, "permission": "edit"},
		"list_share":  {"list": list.Id, "actor": editorActor.Id, "permission": "edit"},
	} {
		saveRulesTestRecord(t, app, collection, data)
	}
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	return ownershipFixture{app, mux, owner, editor, stranger, ownerActor, editorActor,
		trail, otherTrail, list, comment, waypoint, log, instance}
}

func ownershipRequest(t *testing.T, handler http.Handler, record, auth *core.Record, method, format string, body map[string]any) *httptest.ResponseRecorder {
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
	} {
		for _, format := range []string{"json", "multipart", "multipart-json"} {
			original := target.record.GetString(target.field)
			for _, mutation := range []struct {
				name string
				body map[string]any
			}{
				{"replace", map[string]any{target.field: target.replacement}},
				{"append", map[string]any{target.field + "+": target.replacement}},
				{"remove", map[string]any{target.field + "-": original}},
				{"prefix-append", map[string]any{"+" + target.field: target.replacement}},
				{"prefix-remove", map[string]any{"-" + target.field: original}},
				{"echo-and-remove", map[string]any{target.field: original, target.field + "-": original}},
			} {
				t.Run(target.record.Collection().Name+"/"+target.field+"/"+format+"/"+mutation.name, func(t *testing.T) {
					response := ownershipRequest(t, f.handler, target.record, f.owner, http.MethodPatch, format, mutation.body)
					if mutation.name == "replace" && response.Code != http.StatusNotFound {
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
				})
			}
		}
	}
}

func TestImmutableOwnershipKeepsAuthorizedUpdates(t *testing.T) {
	f := newOwnershipFixture(t)
	for _, target := range []struct {
		record *core.Record
		body   map[string]any
	}{
		{f.trail, map[string]any{"name": "Updated trail", "author": f.ownerActor.Id}},
		{f.list, map[string]any{"name": "Updated list", "author": f.ownerActor.Id}},
		{f.comment, map[string]any{"text": "Updated comment", "author": f.ownerActor.Id, "trail": f.trail.Id}},
		{f.waypoint, map[string]any{"name": "Updated waypoint", "author": f.ownerActor.Id, "trail": f.trail.Id}},
		{f.log, map[string]any{"text": "Updated log", "author": f.ownerActor.Id, "trail": f.trail.Id}},
		{f.instance, map[string]any{"status": "disabled", "user": f.owner.Id, "plugin_id": "hammerhead"}},
	} {
		for _, format := range []string{"json", "multipart", "multipart-json"} {
			t.Run(target.record.Collection().Name+"/"+format, func(t *testing.T) {
				response := ownershipRequest(t, f.handler, target.record, f.owner, http.MethodPatch, format, target.body)
				if response.Code != http.StatusOK {
					t.Fatalf("status = %d; want 200: %s", response.Code, response.Body.String())
				}
				fresh, err := f.app.FindRecordById(target.record.Collection().Name, target.record.Id)
				if err != nil {
					t.Fatal(err)
				}
				for key, value := range target.body {
					if got := fresh.GetString(key); got != value {
						t.Errorf("%s = %q; want %q", key, got, value)
					}
				}
			})
		}
	}
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
}

func TestImmutableOwnershipKeepsTrustedSavesAndSuperusers(t *testing.T) {
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
	for _, target := range []struct {
		record *core.Record
		body   map[string]any
	}{
		{f.trail, map[string]any{"author": f.editorActor.Id}},
		{f.instance, map[string]any{"user": f.editor.Id, "plugin_id": "komoot"}},
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

func TestImmutableOwnershipMigrationIsQueueIndependentAndIdempotent(t *testing.T) {
	app := newRulesTestApp(t)
	if _, err := app.FindCollectionByNameOrId("plugin_webhook_jobs"); err == nil {
		t.Fatal("this ownership regression must run without the planned webhook queue")
	}
	collection, err := app.FindCollectionByNameOrId("comments")
	if err != nil {
		t.Fatal(err)
	}
	before := *collection.UpdateRule
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
	collection, err = app.FindCollectionByNameOrId("comments")
	if err != nil || *collection.UpdateRule != before {
		t.Fatalf("migration changed the already guarded rule: %v", err)
	}
	if strings.Count(before, "@request.body.author:changed") != 1 {
		t.Fatal("ownership guard must occur once")
	}
	collection.UpdateRule = nil
	if err := app.Save(collection); err != nil {
		t.Fatal(err)
	}
	if err := apply(app); err != nil {
		t.Fatal(err)
	}
	collection, err = app.FindCollectionByNameOrId("comments")
	if err != nil || collection.UpdateRule != nil {
		t.Fatalf("migration unlocked a locked API: %v", err)
	}
}
