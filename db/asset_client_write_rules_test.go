package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func TestAssetClientWriteRules(t *testing.T) {
	f := newRecordHooksFixture(t)
	t.Setenv("ORIGIN", "https://a.example")
	user := f.save("users", map[string]any{"username": "owner", "email": "owner@a.example"})
	owner := f.save("activitypub_actors", map[string]any{
		"user": user.Id, "username": "owner", "preferred_username": "owner", "domain": "a.example",
		"iri": "https://a.example/owner", "is_local": true,
	})
	remote := f.save("activitypub_actors", map[string]any{
		"username": "remote", "preferred_username": "remote", "domain": "b.example",
		"iri": "https://b.example/remote", "is_local": false,
	})
	trail := f.save("trails", map[string]any{"name": "Public copy", "author": remote.Id, "public": true})
	waypoint := f.save("waypoints", map[string]any{"name": "Waypoint", "author": remote.Id, "trail": trail.Id})
	entry := f.save("summit_logs", map[string]any{"author": remote.Id, "trail": trail.Id})
	remoteAsset := f.photo(remote)
	ownedAsset := f.save("assets", map[string]any{
		"type": "photo", "author": owner.Id, "storage_mode": "copy", "remote_status": "available",
		"metadata": map[string]any{"content_hash": "stored-hash", "label": "old"},
	})
	type target struct {
		collection, field string
		record            *core.Record
	}
	targets := []target{{"trail_assets", "trail", trail}, {"waypoint_assets", "waypoint", waypoint}, {"summit_log_assets", "summit_log", entry}}
	links := make([]*core.Record, len(targets))
	for i, target := range targets {
		links[i] = f.save(target.collection, map[string]any{"asset": remoteAsset.Id, target.field: target.record.Id})
	}
	f.bindHooks()
	router, err := apis.NewRouter(f.app)
	if err != nil {
		t.Fatal(err)
	}
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	request := func(t *testing.T, auth *core.Record, method, path string, body map[string]any, want int) *httptest.ResponseRecorder {
		t.Helper()
		var payload bytes.Buffer
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(method, path, &payload)
		req.Header.Set("Content-Type", "application/json")
		if auth != nil {
			token, err := auth.NewAuthToken()
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Authorization", token)
		}
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		if response.Code != want {
			t.Fatalf("%s %s: status = %d; want %d: %s", method, path, response.Code, want, response.Body.String())
		}
		return response
	}
	snapshot := func(t *testing.T, record *core.Record) string {
		t.Helper()
		stored, err := f.app.FindRecordById(record.Collection(), record.Id)
		if err != nil {
			t.Fatalf("record disappeared: %v", err)
		}
		data, err := json.Marshal(stored.FieldsData())
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	count := func(t *testing.T, collection string) int64 {
		t.Helper()
		n, err := f.app.CountRecords(collection)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	for _, record := range append([]*core.Record{remoteAsset}, links...) {
		t.Run("anonymous_write/"+record.Collection().Name, func(t *testing.T) {
			for _, method := range []string{http.MethodPatch, http.MethodDelete} {
				t.Run(method, func(t *testing.T) {
					before := snapshot(t, record)
					request(t, nil, method, "/api/collections/"+record.Collection().Name+"/records/"+record.Id, map[string]any{}, http.StatusNotFound)
					if after := snapshot(t, record); after != before {
						t.Fatalf("anonymous request changed record: before=%s after=%s", before, after)
					}
				})
			}
		})
	}
	t.Run("anonymous_create/assets", func(t *testing.T) {
		before := count(t, "assets")
		request(t, nil, http.MethodPost, "/api/collections/assets/records", map[string]any{"type": "photo", "author": remote.Id, "storage_mode": "copy"}, http.StatusBadRequest)
		if count(t, "assets") != before {
			t.Fatal("anonymous request created an asset")
		}
	})
	for _, target := range targets {
		t.Run("anonymous_create/"+target.collection, func(t *testing.T) {
			// A separate asset avoids an existing-link unique constraint masking
			// the authentication requirement.
			asset := f.photo(remote)
			before := count(t, target.collection)
			request(t, nil, http.MethodPost, "/api/collections/"+target.collection+"/records", map[string]any{"asset": asset.Id, target.field: target.record.Id}, http.StatusBadRequest)
			if count(t, target.collection) != before {
				t.Fatal("anonymous request created a link")
			}
		})
	}
	t.Run("foreign_author", func(t *testing.T) {
		before := count(t, "assets")
		request(t, user, http.MethodPost, "/api/collections/assets/records", map[string]any{"type": "photo", "author": remote.Id, "storage_mode": "copy"}, http.StatusBadRequest)
		if count(t, "assets") != before {
			t.Fatal("request created an asset for a different author")
		}
	})
	for _, field := range []struct {
		name  string
		value any
	}{
		{"external_provider", "provider"}, {"external_id", "provider-photo"},
		{"remote_error", "client-error"}, {"remote_checked_at", "2026-01-01 00:00:00.000Z"},
		{"remote_missing_since", "2026-01-01 00:00:00.000Z"}, {"remote_status", "missing"},
		{"storage_mode", "link_private"},
	} {
		t.Run("managed_fields/"+field.name, func(t *testing.T) {
			for _, method := range []string{http.MethodPost, http.MethodPatch} {
				t.Run(method, func(t *testing.T) {
					path := "/api/collections/assets/records"
					body := map[string]any{"type": "photo", "author": owner.Id, "storage_mode": "copy"}
					want := http.StatusBadRequest
					if method == http.MethodPatch {
						path += "/" + ownedAsset.Id
						body = map[string]any{}
						want = http.StatusNotFound
					}
					body[field.name] = field.value
					before, total := snapshot(t, ownedAsset), count(t, "assets")
					request(t, user, method, path, body, want)
					if snapshot(t, ownedAsset) != before || count(t, "assets") != total {
						t.Fatal("rejected request changed the stored assets")
					}
				})
			}
		})
	}
	for _, status := range []string{"", "available"} {
		t.Run("copy_create/"+status, func(t *testing.T) {
			request(t, user, http.MethodPost, "/api/collections/assets/records", map[string]any{
				"type": "photo", "author": owner.Id, "storage_mode": "copy", "remote_status": status,
			}, http.StatusOK)
		})
	}
	t.Run("client_content_hash/create", func(t *testing.T) {
		response := request(t, user, http.MethodPost, "/api/collections/assets/records", map[string]any{
			"type": "photo", "author": owner.Id, "storage_mode": "copy",
			"metadata": map[string]any{"content_hash": "client-hash", "label": "new"},
		}, http.StatusOK)
		var result struct{ Id string }
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		stored, err := f.app.FindRecordById("assets", result.Id)
		if err != nil {
			t.Fatal(err)
		}
		metadata := map[string]any{}
		if err := stored.UnmarshalJSONField("metadata", &metadata); err != nil {
			t.Fatal(err)
		}
		if _, exists := metadata["content_hash"]; exists || metadata["label"] != "new" {
			t.Fatalf("client hash persisted or other metadata lost: %v", metadata)
		}
	})
	t.Run("client_content_hash/update", func(t *testing.T) {
		request(t, user, http.MethodPatch, "/api/collections/assets/records/"+ownedAsset.Id, map[string]any{
			"metadata": map[string]any{"content_hash": "client-hash", "label": "new"},
		}, http.StatusOK)
		stored, err := f.app.FindRecordById("assets", ownedAsset.Id)
		if err != nil {
			t.Fatal(err)
		}
		metadata := map[string]any{}
		if err := stored.UnmarshalJSONField("metadata", &metadata); err != nil {
			t.Fatal(err)
		}
		if metadata["content_hash"] != "stored-hash" || metadata["label"] != "new" {
			t.Fatalf("trusted hash changed or client metadata lost: %v", metadata)
		}
	})
}
