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
			api := newShareTestAPI(t, config.collection, config.target, config.recipient)
			initialRecipient, changedRecipient := api.localActor.Id, api.ownerActor.Id
			readerToken := api.readerToken
			if config.recipient == "token" {
				initialRecipient, changedRecipient = strings.Repeat("a", 32), strings.Repeat("b", 32)
				readerToken = "" // Link shares must work for anonymous readers.
			}
			readObject := func(t *testing.T, object *core.Record, token string, status int) {
				t.Helper()
				path := "/api/collections/" + api.objects.Name + "/records/" + object.Id
				if config.recipient == "token" {
					path += "?share=" + token
				}
				shareRequest(t, api.mux, http.MethodGet, path, nil, readerToken, status)
			}
			for _, test := range []struct {
				name   string
				patch  map[string]string
				status int
			}{
				{"reject target change atomically", map[string]string{config.target: api.foreignObject.Id, "permission": "edit", config.recipient: changedRecipient}, http.StatusBadRequest},
				{"permission with omitted target", map[string]string{"permission": "edit"}, http.StatusOK},
				{"permission with unchanged target", map[string]string{config.target: api.privateObject.Id, "permission": "edit"}, http.StatusOK},
				{"change " + config.recipient, map[string]string{config.recipient: changedRecipient}, http.StatusOK},
			} {
				t.Run(test.name, func(t *testing.T) {
					share := api.newShare(t, api.privateObject, initialRecipient)
					readObject(t, api.privateObject, initialRecipient, http.StatusOK)
					readObject(t, api.foreignObject, initialRecipient, http.StatusNotFound)
					api.patch(t, share, test.patch, test.status)
					readObject(t, api.foreignObject, initialRecipient, http.StatusNotFound)
					status := http.StatusOK
					if test.status == http.StatusOK && test.patch[config.recipient] == changedRecipient {
						status = http.StatusNotFound
					}
					readObject(t, api.privateObject, initialRecipient, status)
					if config.recipient == "token" {
						token := initialRecipient
						if status == http.StatusNotFound {
							token = changedRecipient
						}
						readObject(t, api.privateObject, token, http.StatusOK)
						readObject(t, api.foreignObject, changedRecipient, http.StatusNotFound)
					}
				})
			}
		})
	}
}

type shareTestAPI struct {
	app                                        *core.BaseApp
	mux                                        http.Handler
	target, recipient                          string
	objects, shares                            *core.Collection
	ownerActor, localActor, remoteActor        *core.Record
	privateObject, publicObject, foreignObject *core.Record
	ownerToken, readerToken                    string
}

func newShareTestAPI(t *testing.T, collection, target, recipient string) *shareTestAPI {
	t.Helper()
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
	actors.Fields.Add(
		&core.RelationField{Name: "user", CollectionId: users.Id, MaxSelect: 1},
		&core.BoolField{Name: "is_local"},
	)
	objects := core.NewBaseCollection(target + "s")
	objects.Fields.Add(
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1, Required: true},
		&core.BoolField{Name: "public"},
	)
	shares := core.NewBaseCollection(collection)
	shares.Fields.Add(
		&core.RelationField{Name: target, CollectionId: objects.Id, MaxSelect: 1, Required: true},
		&core.SelectField{Name: "permission", Values: []string{"view", "edit"}, MaxSelect: 1, Required: true},
	)
	if recipient == "actor" {
		shares.Fields.Add(&core.RelationField{Name: "actor", CollectionId: actors.Id, MaxSelect: 1, Required: true})
	} else {
		shares.Fields.Add(&core.TextField{Name: "token", Min: 32, Max: 32, Required: true})
		shares.Indexes = []string{"CREATE UNIQUE INDEX idx_test_link_share_trail ON trail_link_share (trail)"}
	}
	// Keep the production ownership and share-access paths.
	shares.CreateRule = types.Pointer(target + ".author.user = @request.auth.id")
	shares.UpdateRule = types.Pointer(target + ".author.user = @request.auth.id")
	for _, collection := range []*core.Collection{actors, objects, shares} {
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}
	accessRule := "(@request.auth.id != '' && " + collection + "_via_" + target + ".actor.user ?= @request.auth.id)"
	if recipient == "token" {
		accessRule = "(trail_link_share_via_trail.token != '' && trail_link_share_via_trail.token = @request.query.share)"
	}
	objects.ViewRule = types.Pointer("author.user = @request.auth.id || public = true || " + accessRule)
	if err := app.Save(objects); err != nil {
		t.Fatal(err)
	}
	save := func(collection *core.Collection, data map[string]any) *core.Record {
		t.Helper()
		record := core.NewRecord(collection)
		record.Load(data)
		if err := app.Save(record); err != nil {
			t.Fatal(err)
		}
		return record
	}
	owner := save(users, map[string]any{"email": "owner@example.com", "password": "test-password"})
	reader := save(users, map[string]any{"email": "reader@example.com", "password": "test-password"})
	foreignOwner := save(users, map[string]any{"email": "foreign@example.com", "password": "test-password"})
	ownerActor := save(actors, map[string]any{"user": owner.Id, "is_local": true})
	localActor := save(actors, map[string]any{"user": reader.Id, "is_local": true})
	remoteActor := save(actors, map[string]any{"is_local": false})
	foreignActor := save(actors, map[string]any{"user": foreignOwner.Id, "is_local": true})
	privateObject := save(objects, map[string]any{"author": ownerActor.Id})
	publicObject := save(objects, map[string]any{"author": ownerActor.Id, "public": true})
	foreignObject := save(objects, map[string]any{"author": foreignActor.Id})
	ownerToken, err := owner.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	readerToken, err := reader.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	// Seed fixtures first to avoid unrelated indexing/federation hooks.
	// Use production registration, so missing share hooks break the tests.
	setupEventHandlers(&pocketbase.PocketBase{App: app}, nil)
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	return &shareTestAPI{
		app: app, mux: mux, target: target, recipient: recipient, objects: objects, shares: shares,
		ownerActor: ownerActor, localActor: localActor, remoteActor: remoteActor,
		privateObject: privateObject, publicObject: publicObject, foreignObject: foreignObject,
		ownerToken: ownerToken, readerToken: readerToken,
	}
}

func (api *shareTestAPI) newShare(t *testing.T, object *core.Record, recipient string) *core.Record {
	t.Helper()
	share := core.NewRecord(api.shares)
	share.Load(map[string]any{api.target: object.Id, api.recipient: recipient, "permission": "view"})
	if err := api.app.Save(share); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := api.app.Delete(share); err != nil {
			t.Error(err)
		}
	})
	return share
}

func (api *shareTestAPI) patch(t *testing.T, share *core.Record, patch map[string]string, status int) {
	t.Helper()
	want := map[string]string{api.target: share.GetString(api.target), api.recipient: share.GetString(api.recipient), "permission": share.GetString("permission")}
	shareRequest(t, api.mux, http.MethodPatch, "/api/collections/"+api.shares.Name+"/records/"+share.Id, patch, api.ownerToken, status)
	if status == http.StatusOK {
		for field, value := range patch {
			want[field] = value
		}
	}
	stored, err := api.app.FindRecordById(api.shares.Name, share.Id)
	if err != nil {
		t.Fatal(err)
	}
	for field, value := range want {
		if got := stored.GetString(field); got != value {
			t.Errorf("stored %s = %q; want %q", field, got, value)
		}
	}
}

func shareRequest(t *testing.T, mux http.Handler, method, path string, body map[string]string, auth string, status int) {
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
