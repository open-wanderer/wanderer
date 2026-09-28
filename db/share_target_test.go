package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/meilisearch/meilisearch-go"
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

func TestShareIndexTracksRemainingRecipients(t *testing.T) {
	for _, target := range []string{"trail", "list"} {
		t.Run(target, func(t *testing.T) {
			api := newShareTestAPI(t, target+"_share", target, "actor")
			t.Run("share lifecycle", func(t *testing.T) {
				api.newShare(t, api.publicObject, api.ownerActor.Id)
				api.index.assertShares(t, api.publicObject.Id, api.ownerActor.Id)
				t.Run("additional recipient", func(t *testing.T) {
					share := api.newShare(t, api.publicObject, api.localActor.Id)
					api.index.assertShares(t, api.publicObject.Id, api.ownerActor.Id, api.localActor.Id)
					api.patch(t, share, map[string]string{"actor": api.remoteActor.Id}, http.StatusOK)
					api.index.assertShares(t, api.publicObject.Id, api.ownerActor.Id, api.remoteActor.Id)
					share.Set("actor", api.localActor.Id)
					if err := api.app.Save(share); err != nil {
						t.Fatal(err)
					}
					api.index.assertShares(t, api.publicObject.Id, api.ownerActor.Id, api.localActor.Id)
				})
				// Cleanup deletes the second share through the internal record API.
				api.index.assertShares(t, api.publicObject.Id, api.ownerActor.Id)
			})
			api.index.assertShares(t, api.publicObject.Id)

		})
	}
}

func TestShareIndexPreservesRecipientsOnAPIDelete(t *testing.T) {
	for _, target := range []string{"trail", "list"} {
		t.Run(target, func(t *testing.T) {
			api := newShareTestAPI(t, target+"_share", target, "actor")
			api.newShare(t, api.publicObject, api.ownerActor.Id)
			share := api.newShare(t, api.publicObject, api.localActor.Id)
			path := "/api/collections/" + api.shares.Name + "/records/" + share.Id
			updatesBefore := api.index.updateCount()
			shareRequest(t, api.mux, http.MethodDelete, path, nil, api.readerToken, http.StatusNotFound)
			if _, err := api.app.FindRecordById(api.shares.Name, share.Id); err != nil {
				t.Fatalf("rejected delete removed share: %v", err)
			}
			if updates := api.index.updateCount(); updates != updatesBefore {
				t.Fatalf("rejected delete changed index: updates = %d; want %d", updates, updatesBefore)
			}
			shareRequest(t, api.mux, http.MethodDelete, path, nil, api.ownerToken, http.StatusNoContent)
			if _, err := api.app.FindRecordById(api.shares.Name, share.Id); !errors.Is(err, sql.ErrNoRows) {
				t.Fatalf("deleted share lookup = %v; want no rows", err)
			}
			api.index.assertShares(t, api.publicObject.Id, api.ownerActor.Id)
		})
	}
}

func TestShareIndexTracksInternalTargetChange(t *testing.T) {
	for _, target := range []string{"trail", "list"} {
		t.Run(target, func(t *testing.T) {
			api := newShareTestAPI(t, target+"_share", target, "actor")
			api.newShare(t, api.publicObject, api.ownerActor.Id)
			share := api.newShare(t, api.publicObject, api.localActor.Id)
			api.index.assertShares(t, api.publicObject.Id, api.ownerActor.Id, api.localActor.Id)
			share, err := api.app.FindRecordById(api.shares.Name, share.Id)
			if err != nil {
				t.Fatal(err)
			}
			share.Set(target, api.privateObject.Id)
			if err := api.app.Save(share); err != nil {
				t.Fatal(err)
			}
			api.index.assertShares(t, api.publicObject.Id, api.ownerActor.Id)
			api.index.assertShares(t, api.privateObject.Id, api.localActor.Id)
		})
	}
}

func TestShareIndexIgnoresRolledBackMutations(t *testing.T) {
	for _, target := range []string{"trail", "list"} {
		t.Run(target, func(t *testing.T) {
			api := newShareTestAPI(t, target+"_share", target, "actor")
			share := api.newShare(t, api.publicObject, api.localActor.Id)
			for _, operation := range []string{"create", "update", "delete"} {
				t.Run(operation, func(t *testing.T) {
					updatesBefore := api.index.updateCount()
					rollback := errors.New("test transaction rollback")
					err := api.app.RunInTransaction(func(tx core.App) error {
						if operation == "create" {
							created := core.NewRecord(api.shares)
							created.Load(map[string]any{target: api.publicObject.Id, "actor": api.ownerActor.Id, "permission": "view"})
							if err := tx.Save(created); err != nil {
								return err
							}
						} else {
							stored, err := tx.FindRecordById(api.shares.Name, share.Id)
							if err != nil {
								return err
							}
							if operation == "update" {
								stored.Set("actor", api.remoteActor.Id)
								err = tx.Save(stored)
							} else {
								err = tx.Delete(stored)
							}
							if err != nil {
								return err
							}
						}
						return rollback
					})
					if !errors.Is(err, rollback) {
						t.Fatalf("transaction result = %v; want rollback", err)
					}
					if updates := api.index.updateCount(); updates != updatesBefore {
						t.Errorf("rolled back %s changed index: updates = %d; want %d", operation, updates, updatesBefore)
					}
					api.index.assertShares(t, api.publicObject.Id, api.localActor.Id)
					stored, err := api.app.FindAllRecords(api.shares.Name)
					if err != nil || len(stored) != 1 {
						t.Fatalf("shares after rollback: count = %d, error = %v; want one share", len(stored), err)
					}
					if stored[0].Id != share.Id || stored[0].GetString("actor") != api.localActor.Id {
						t.Errorf("rollback failed to preserve original share: %v", stored[0])
					}
				})
			}
		})
	}
}

type shareTestAPI struct {
	app                                        *core.BaseApp
	mux                                        http.Handler
	index                                      *shareTestIndex
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
	shares.DeleteRule = types.Pointer(target + ".author.user = @request.auth.id")
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
	// Use production registration and record outgoing index updates, so missing
	// authorization or successful-mutation hooks break the tests.
	index, client := newShareTestIndex(t, objects.Name)
	setupEventHandlers(&pocketbase.PocketBase{App: app}, client)
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	return &shareTestAPI{
		app: app, mux: mux, index: index, target: target, recipient: recipient, objects: objects, shares: shares,
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
		stored, err := api.app.FindRecordById(api.shares.Name, share.Id)
		if errors.Is(err, sql.ErrNoRows) {
			return // Some lifecycle tests delete the share through the records API.
		}
		if err != nil {
			t.Error(err)
			return
		}
		if err := api.app.Delete(stored); err != nil {
			t.Error(err)
		}
	})
	return share
}

func (api *shareTestAPI) patch(t *testing.T, share *core.Record, patch map[string]string, status int) {
	t.Helper()
	want := map[string]string{api.target: share.GetString(api.target), api.recipient: share.GetString(api.recipient), "permission": share.GetString("permission")}
	updatesBefore := api.index.updateCount()
	shareRequest(t, api.mux, http.MethodPatch, "/api/collections/"+api.shares.Name+"/records/"+share.Id, patch, api.ownerToken, status)
	if status == http.StatusOK {
		for field, value := range patch {
			want[field] = value
		}
	} else if updates := api.index.updateCount(); updates != updatesBefore {
		t.Errorf("rejected share update changed the index: updates = %d; want %d", updates, updatesBefore)
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

type shareTestIndexUpdate struct {
	ID     string   `json:"id"`
	Shares []string `json:"shares"`
}

type shareTestIndex struct {
	mu      sync.Mutex
	updates []shareTestIndexUpdate
}

func newShareTestIndex(t *testing.T, collection string) (*shareTestIndex, meilisearch.ServiceManager) {
	t.Helper()
	index := &shareTestIndex{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/indexes/"+collection+"/documents" {
			t.Errorf("unexpected index request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		var updates []shareTestIndexUpdate
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&updates); err != nil {
			t.Errorf("decode index update: %v", err)
			http.Error(w, "invalid document update", http.StatusBadRequest)
			return
		}
		for _, update := range updates {
			if update.ID == "" || update.Shares == nil {
				t.Errorf("incomplete share index update: %+v", update)
				http.Error(w, "incomplete document update", http.StatusBadRequest)
				return
			}
		}
		index.mu.Lock()
		index.updates = append(index.updates, updates...)
		index.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"taskUid":1,"status":"enqueued","type":"documentAdditionOrUpdate"}`))
	}))
	t.Cleanup(server.Close)
	return index, meilisearch.New(server.URL)
}

func (index *shareTestIndex) updateCount() int {
	index.mu.Lock()
	defer index.mu.Unlock()
	return len(index.updates)
}

func (index *shareTestIndex) assertShares(t *testing.T, id string, want ...string) {
	t.Helper()
	index.mu.Lock()
	defer index.mu.Unlock()
	slices.Sort(want)
	for i := len(index.updates) - 1; i >= 0; i-- {
		if update := index.updates[i]; update.ID == id {
			got := slices.Clone(update.Shares)
			slices.Sort(got)
			if !slices.Equal(got, want) {
				t.Errorf("indexed shares for %s = %v; want %v", id, got, want)
			}
			return
		}
	}
	t.Fatalf("no share index update for %s", id)
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
