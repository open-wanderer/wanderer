package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func TestShareTargetUpdatesThroughRecordsAPI(t *testing.T) {
	for _, config := range []struct{ collection, target, recipient string }{
		{"trail_share", "trail", "actor"},
		{"list_share", "list", "actor"},
		{"trail_link_share", "trail", "token"},
	} {
		t.Run(config.collection, func(t *testing.T) {
			// Bootstrap only system collections; app migrations require Meilisearch.
			app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
			t.Cleanup(func() {
				if err := app.ResetBootstrapState(); err != nil {
					t.Error(err)
				}
			})
			if err := app.Bootstrap(); err != nil {
				t.Fatal(err)
			}
			app.Settings().Logs.MaxDays = 0
			users, err := app.FindCollectionByNameOrId("users")
			if err != nil {
				t.Fatal(err)
			}
			actors := core.NewBaseCollection("activitypub_actors")
			actors.Fields.Add(&core.RelationField{Name: "user", CollectionId: users.Id, MaxSelect: 1})
			objects := core.NewBaseCollection(config.target + "s")
			objects.Fields.Add(&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1, Required: true})
			shares := core.NewBaseCollection(config.collection)
			shares.Fields.Add(
				&core.RelationField{Name: config.target, CollectionId: objects.Id, MaxSelect: 1, Required: true},
				&core.SelectField{Name: "permission", Values: []string{"view", "edit"}, MaxSelect: 1, Required: true},
			)
			if config.recipient == "actor" {
				shares.Fields.Add(&core.RelationField{Name: "actor", CollectionId: actors.Id, MaxSelect: 1, Required: true})
			} else {
				shares.Fields.Add(&core.TextField{Name: "token", Min: 32, Max: 32, Required: true})
				shares.Indexes = []string{"CREATE UNIQUE INDEX idx_test_link_share_trail ON trail_link_share (trail)"}
			}
			// Keep the ownership and share-access rules relevant to these private objects.
			shares.UpdateRule = types.Pointer(config.target + ".author.user = @request.auth.id")
			for _, collection := range []*core.Collection{actors, objects, shares} {
				if err := app.Save(collection); err != nil {
					t.Fatal(err)
				}
			}
			accessRule := "(@request.auth.id != '' && " + config.collection + "_via_" + config.target + ".actor.user ?= @request.auth.id)"
			if config.recipient == "token" {
				accessRule = "(trail_link_share_via_trail.token != '' && trail_link_share_via_trail.token = @request.query.share)"
			}
			objects.ViewRule = types.Pointer("author.user = @request.auth.id || " + accessRule)
			if err := app.Save(objects); err != nil {
				t.Fatal(err)
			}
			save := func(t *testing.T, collection *core.Collection, data map[string]string) *core.Record {
				t.Helper()
				record := core.NewRecord(collection)
				for field, value := range data {
					record.Set(field, value)
				}
				if err := app.Save(record); err != nil {
					t.Fatal(err)
				}
				return record
			}
			owner := save(t, users, map[string]string{"email": "owner@example.com", "password": "test-password"})
			recipient := save(t, users, map[string]string{"email": "recipient@example.com", "password": "test-password"})
			foreignOwner := save(t, users, map[string]string{"email": "foreign@example.com", "password": "test-password"})
			ownerActor := save(t, actors, map[string]string{"user": owner.Id})
			recipientActor := save(t, actors, map[string]string{"user": recipient.Id})
			foreignActor := save(t, actors, map[string]string{"user": foreignOwner.Id})
			ownObject := save(t, objects, map[string]string{"author": ownerActor.Id})
			foreignObject := save(t, objects, map[string]string{"author": foreignActor.Id})
			ownerToken, err := owner.NewAuthToken()
			if err != nil {
				t.Fatal(err)
			}
			readerToken, err := recipient.NewAuthToken()
			if err != nil {
				t.Fatal(err)
			}
			initialRecipient, changedRecipient := recipientActor.Id, ownerActor.Id
			if config.recipient == "token" {
				initialRecipient, changedRecipient = strings.Repeat("a", 32), strings.Repeat("b", 32)
				readerToken = "" // Link shares must work for anonymous readers.
			}

			// Use production registration, so removing any share hook breaks this test.
			// Seed users and objects first to avoid unrelated indexing/federation hooks.
			setupEventHandlers(&pocketbase.PocketBase{App: app}, nil)
			router, err := apis.NewRouter(app)
			if err != nil {
				t.Fatal(err)
			}
			mux, err := router.BuildMux()
			if err != nil {
				t.Fatal(err)
			}
			readObject := func(t *testing.T, object *core.Record, token string, status int) {
				t.Helper()
				path := "/api/collections/" + objects.Name + "/records/" + object.Id
				if config.recipient == "token" {
					path += "?share=" + token
				}
				shareTargetRequest(t, mux, http.MethodGet, path, nil, readerToken, status)
			}
			for _, test := range []struct {
				name   string
				patch  map[string]string
				status int
			}{
				{"reject target change atomically", map[string]string{config.target: foreignObject.Id, "permission": "edit", config.recipient: changedRecipient}, http.StatusBadRequest},
				{"permission with omitted target", map[string]string{"permission": "edit"}, http.StatusOK},
				{"permission with unchanged target", map[string]string{config.target: ownObject.Id, "permission": "edit"}, http.StatusOK},
				{"change " + config.recipient, map[string]string{config.recipient: changedRecipient}, http.StatusOK},
			} {
				t.Run(test.name, func(t *testing.T) {
					want := map[string]string{config.target: ownObject.Id, "permission": "view", config.recipient: initialRecipient}
					share := save(t, shares, want)
					t.Cleanup(func() {
						if err := app.Delete(share); err != nil {
							t.Error(err)
						}
					})
					readObject(t, ownObject, initialRecipient, http.StatusOK)
					readObject(t, foreignObject, initialRecipient, http.StatusNotFound)
					shareTargetRequest(t, mux, http.MethodPatch, "/api/collections/"+shares.Name+"/records/"+share.Id, test.patch, ownerToken, test.status)
					if test.status == http.StatusOK {
						for field, value := range test.patch {
							want[field] = value
						}
					}
					stored, err := app.FindRecordById(shares.Name, share.Id)
					if err != nil {
						t.Fatal(err)
					}
					for field, value := range want {
						if got := stored.GetString(field); got != value {
							t.Errorf("stored %s = %q; want %q", field, got, value)
						}
					}
					readObject(t, foreignObject, initialRecipient, http.StatusNotFound)
					status := http.StatusOK
					if want[config.recipient] != initialRecipient {
						status = http.StatusNotFound
					}
					readObject(t, ownObject, initialRecipient, status)
					if config.recipient == "token" {
						readObject(t, ownObject, want["token"], http.StatusOK)
						readObject(t, foreignObject, changedRecipient, http.StatusNotFound)
					}
				})
			}
		})
	}
}

func shareTargetRequest(t *testing.T, mux http.Handler, method, path string, body map[string]string, auth string, status int) {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(data))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", auth)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != status {
		t.Fatalf("%s %s: status %d; want %d: %s", method, path, response.Code, status, response.Body.String())
	}
}
