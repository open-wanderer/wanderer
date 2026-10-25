package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
)

func TestRecordsAPIKeepsChildObjectIdentityServerOwned(t *testing.T) {
	f := newRecordHooksFixture(t)
	owner := f.localActor("owner")
	trail := f.save("trails", map[string]any{"name": "Parent", "author": owner.Id})
	f.bindHooks()
	user, err := f.app.FindRecordById("users", owner.GetString("user"))
	if err != nil {
		t.Fatal(err)
	}
	token, err := user.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	router, err := apis.NewRouter(f.app)
	if err != nil {
		t.Fatal(err)
	}
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	for collection, kind := range map[string]string{"comments": "comment", "summit_logs": "summit-log", "waypoints": "waypoint"} {
		t.Run(collection, func(t *testing.T) {
			mutate := func(method, suffix string, fields map[string]any) map[string]any {
				body, err := json.Marshal(fields)
				if err != nil {
					t.Fatal(err)
				}
				request := httptest.NewRequest(method, "/api/collections/"+collection+"/records"+suffix, strings.NewReader(string(body)))
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set("Authorization", token)
				response := httptest.NewRecorder()
				mux.ServeHTTP(response, request)
				if response.Code != http.StatusOK {
					t.Fatalf("child mutation: %d %s", response.Code, response.Body.String())
				}
				var result map[string]any
				if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				return result
			}
			created := mutate(http.MethodPost, "", map[string]any{
				"name": "Point", "text": "Entry", "date": "2026-01-01 00:00:00.000Z", "trail": trail.Id, "author": owner.Id,
				"iri": "https://foreign.example/api/v1/" + kind + "/future",
			})
			id := created["id"].(string)
			stored, err := f.app.FindRecordById(collection, id)
			if err != nil {
				t.Fatal(err)
			}
			if stored.GetString("iri") != fmt.Sprintf("https://example.com/api/v1/%s/%s", kind, id) || created["iri"] == "https://foreign.example/api/v1/"+kind+"/future" {
				t.Fatalf("client supplied child identity was accepted: stored=%#v response=%#v", stored.PublicExport(), created)
			}
			stored.Set("iri", "https://historical.example/api/v1/"+kind+"/"+id)
			if err := f.app.UnsafeWithoutHooks().Save(stored); err != nil {
				t.Fatal(err)
			}
			updated := mutate(http.MethodPatch, "/"+id, map[string]any{"name": "Renamed", "text": "Edited", "iri": "https://replacement.example/object"})
			if updated["iri"] != stored.GetString("iri") || updated["author"] != owner.Id || updated["trail"] != trail.Id {
				t.Fatalf("child update changed identity: %#v", updated)
			}
			if collection == "waypoints" && updated["name"] != "Renamed" || collection != "waypoints" && updated["text"] != "Edited" {
				t.Fatalf("ordinary child edit was discarded: %#v", updated)
			}
		})
	}
}
