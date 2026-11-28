package migrations

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"sort"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func TestAssetLinkWritesWithMixedTrailShares(t *testing.T) {
	t.Setenv("ORIGIN", "https://a.example")
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.ResetBootstrapState() })
	// Apply the real schema, omitting only migrations that configure an
	// external Meilisearch instance rather than changing the database.
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
	save := func(collection string, data map[string]any) *core.Record {
		t.Helper()
		col, err := app.FindCollectionByNameOrId(collection)
		if err != nil {
			t.Fatal(err)
		}
		record := core.NewRecord(col)
		record.Load(data)
		if err := app.Save(record); err != nil {
			t.Fatalf("save %s: %v", collection, err)
		}
		return record
	}
	newUser := func(name string) (*core.Record, *core.Record) {
		user := save("users", map[string]any{
			"username": name, "email": name + "@a.example", "password": "test-password",
		})
		actor := save("activitypub_actors", map[string]any{
			"username": name, "preferred_username": name, "domain": "a.example",
			"user": user.Id, "is_local": true, "public_key": "test-key",
			"iri": "https://a.example/" + name, "inbox": "https://a.example/" + name + "/inbox",
		})
		return user, actor
	}
	owner, ownerActor := newUser("owner")
	editor, editorActor := newUser("editor")
	reader, readerActor := newUser("reader")
	trail := save("trails", map[string]any{"name": "Mixed shares", "author": ownerActor.Id})
	var editorShare *core.Record
	for _, share := range []struct {
		actor      *core.Record
		permission string
	}{{editorActor, "edit"}, {readerActor, "view"}} {
		record := save("trail_share", map[string]any{
			"trail": trail.Id, "actor": share.actor.Id, "permission": share.permission,
		})
		if share.actor == editorActor {
			editorShare = record
		}
	}
	waypoint := save("waypoints", map[string]any{"name": "Waypoint", "author": ownerActor.Id, "trail": trail.Id})
	summitLog := save("summit_logs", map[string]any{"author": ownerActor.Id, "trail": trail.Id})
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	request := func(t *testing.T, auth *core.Record, method, path string, body map[string]any, want int) {
		t.Helper()
		var payload bytes.Buffer
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(method, path, &payload)
		req.Header.Set("Content-Type", "application/json")
		token, err := auth.NewAuthToken()
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		if response.Code != want {
			t.Fatalf("%s %s as %s: status = %d; want %d: %s", method, path, auth.GetString("username"), response.Code, want, response.Body.String())
		}
	}
	for _, target := range []struct {
		collection, field string
		record            *core.Record
	}{
		{"trail_assets", "trail", trail},
		{"waypoint_assets", "waypoint", waypoint},
		{"summit_log_assets", "summit_log", summitLog},
	} {
		t.Run(target.collection, func(t *testing.T) {
			ownerAsset := save("assets", map[string]any{"author": ownerActor.Id, "type": "photo", "storage_mode": "copy"})
			link := save(target.collection, map[string]any{"asset": ownerAsset.Id, target.field: target.record.Id})
			path := "/api/collections/" + target.collection + "/records"
			for _, user := range []struct {
				auth, actor              *core.Record
				create, update, deletion int
			}{
				{reader, readerActor, http.StatusBadRequest, http.StatusNotFound, http.StatusNotFound},
				{editor, editorActor, http.StatusOK, http.StatusOK, http.StatusNoContent},
			} {
				if target.field == "summit_log" && user.auth == editor {
					user.create, user.update, user.deletion = http.StatusBadRequest, http.StatusNotFound, http.StatusNotFound
				}
				// Creation uses the caller's own asset, so only the target's
				// edit permission determines whether the new link is allowed.
				asset := save("assets", map[string]any{"author": user.actor.Id, "type": "photo", "storage_mode": "copy"})
				request(t, user.auth, http.MethodPost, path, map[string]any{"asset": asset.Id, target.field: target.record.Id}, user.create)
				// Both users can read the owner's photo. Edit shares permit
				// changes to trail and waypoint links; summit-log links
				// require the entry author or trail author.
				request(t, user.auth, http.MethodGet, path+"/"+link.Id, nil, http.StatusOK)
				body := map[string]any{}
				if target.field == "trail" {
					body["is_thumbnail"] = true
				}
				request(t, user.auth, http.MethodPatch, path+"/"+link.Id, body, user.update)
				request(t, user.auth, http.MethodDelete, path+"/"+link.Id, nil, user.deletion)
			}

			body := map[string]any{}
			if target.field == "trail" {
				body["is_thumbnail"] = true
			}
			for _, user := range []struct {
				auth, actor *core.Record
			}{{reader, readerActor}, {editor, editorActor}} {
				contributedAsset := save("assets", map[string]any{
					"author": user.actor.Id, "type": "photo", "storage_mode": "copy",
				})
				contributedLink := save(target.collection, map[string]any{
					"asset": contributedAsset.Id, target.field: target.record.Id,
				})
				if user.auth == editor {
					// While the edit share remains, the editor can update
					// their own trail or waypoint link, but not a summit-log
					// link on another author's entry.
					status := http.StatusOK
					if target.field == "summit_log" {
						status = http.StatusNotFound
					}
					request(t, editor, http.MethodPatch, path+"/"+contributedLink.Id, body, status)
					if err := app.Delete(editorShare); err != nil {
						t.Fatal(err)
					}
				}
				// Asset ownership alone cannot change the target's photo
				// presentation, including after an edit share is revoked.
				request(t, user.auth, http.MethodPatch, path+"/"+contributedLink.Id, body, http.StatusNotFound)
				request(t, owner, http.MethodPatch, path+"/"+contributedLink.Id, body, http.StatusOK)
				// Contributors may still withdraw their own photo links.
				request(t, user.auth, http.MethodDelete, path+"/"+contributedLink.Id, nil, http.StatusNoContent)
				if user.auth == editor {
					editorShare = save("trail_share", map[string]any{
						"trail": trail.Id, "actor": editorActor.Id, "permission": "edit",
					})
				}
			}
		})
	}
}
