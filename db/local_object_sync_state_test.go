package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"pocketbase/util"
)

func TestRecordsAPIKeepsObjectIdentityAndSyncStateServerOwned(t *testing.T) {
	f := newRecordHooksFixture(t)
	owner := f.localActor("owner")
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
	for _, name := range []string{"trails", "lists"} {
		t.Run(name, func(t *testing.T) {
			mutate := func(method, suffix string, fields map[string]any) map[string]any {
				body, err := json.Marshal(fields)
				if err != nil {
					t.Fatal(err)
				}
				request := httptest.NewRequest(method, "/api/collections/"+name+"/records"+suffix, strings.NewReader(string(body)))
				request.Header.Set("Content-Type", "application/json")
				request.Header.Set("Authorization", token)
				response := httptest.NewRecorder()
				mux.ServeHTTP(response, request)
				if response.Code != http.StatusOK {
					t.Fatalf("records mutation: %d %s", response.Code, response.Body.String())
				}
				var result map[string]any
				if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				return result
			}
			created := mutate(http.MethodPost, "", map[string]any{"name": "Local", "author": owner.Id, "iri": "https://foreign.example/object", "needs_full_sync": true, "full_sync_completed": true})
			id := created["id"].(string)
			if !util.IsLocalIRI(created["iri"].(string)) || created["needs_full_sync"] != false || created["full_sync_completed"] != false {
				t.Fatalf("create accepted client bookkeeping: %#v", created)
			}
			stored, err := f.app.FindRecordById(name, id)
			if err != nil {
				t.Fatal(err)
			}
			stored.Set("iri", "https://historical.example/api/v1/object/"+id)
			stored.Set("full_sync_completed", true)
			if err := f.app.UnsafeWithoutHooks().Save(stored); err != nil {
				t.Fatal(err)
			}
			updated := mutate(http.MethodPatch, "/"+id, map[string]any{"name": "Renamed", "iri": "https://replacement.example/object", "needs_full_sync": true, "full_sync_completed": false})
			if updated["name"] != "Renamed" || updated["iri"] != stored.GetString("iri") || updated["needs_full_sync"] != false || updated["full_sync_completed"] != true {
				t.Fatalf("update changed server bookkeeping: %#v", updated)
			}
		})
	}
}

func TestRecordsAPISuperuserCanRepairObjectIdentityAndSyncState(t *testing.T) {
	f := newRecordHooksFixture(t)
	owner := f.localActor("owner")
	admin := f.save("_superusers", map[string]any{"email": "administrator@example.com", "password": "long-test-password"})
	f.bindHooks()
	token, err := admin.NewAuthToken()
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
	for _, name := range []string{"trails", "lists"} {
		t.Run(name, func(t *testing.T) {
			record := f.save(name, map[string]any{"name": "Repair", "author": owner.Id})
			iri := "https://historical.example/api/v1/object/" + record.Id
			body, err := json.Marshal(map[string]any{"iri": iri, "needs_full_sync": true, "full_sync_completed": true})
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodPatch, "/api/collections/"+name+"/records/"+record.Id, strings.NewReader(string(body)))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", token)
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("superuser repair: %d %s", response.Code, response.Body.String())
			}
			repaired, err := f.app.FindRecordById(name, record.Id)
			if err != nil {
				t.Fatal(err)
			}
			if repaired.GetString("iri") != iri || !repaired.GetBool("needs_full_sync") || !repaired.GetBool("full_sync_completed") {
				t.Fatalf("superuser repair was discarded: %#v", repaired.PublicExport())
			}
		})
	}
}
