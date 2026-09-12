package hooks

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/types"
)

func TestUpdateShareTargetHandlerThroughRecordsAPI(t *testing.T) {
	for _, objectField := range []string{"trail", "list"} {
		t.Run(objectField, func(t *testing.T) {
			api := newShareUpdateAPITest(t, objectField)
			t.Run("unguarded API allows retargeting another owner's private object", func(t *testing.T) {
				record := api.newShare(t, api.privateObject, api.localActor)
				api.assertRecipientAccess(t, http.StatusNotFound)
				response := api.patch(t, record, map[string]string{objectField: api.foreignPrivateObject.Id}, api.ownerToken)
				if response.Code != http.StatusOK {
					t.Fatalf("unguarded PATCH status %d; want 200: %s", response.Code, response.Body.String())
				}
				persisted, err := api.app.FindRecordById(api.shares.Name, record.Id)
				if err != nil {
					t.Fatal(err)
				}
				if persisted.GetString(objectField) != api.foreignPrivateObject.Id {
					t.Fatal("unguarded PATCH did not persist the unauthorized target")
				}
				api.assertRecipientAccess(t, http.StatusOK)
			})
			api.assertRecipientAccess(t, http.StatusNotFound)

			api.app.OnRecordUpdateRequest(api.shares.Name).BindFunc(UpdateShareTargetHandler(objectField))
			api.runCases(t, []shareUpdateAPICase{
				{"retarget another owner's private object", api.privateObject, api.localActor, map[string]string{objectField: api.foreignPrivateObject.Id}, api.ownerToken, http.StatusBadRequest},
				{"change target and recipient together", api.privateObject, api.localActor, map[string]string{objectField: api.foreignPrivateObject.Id, "actor": api.ownerActor.Id}, api.ownerToken, http.StatusBadRequest},
				{"different owned target", api.privateObject, api.localActor, map[string]string{objectField: api.publicObject.Id}, api.ownerToken, http.StatusBadRequest},
				{"empty target", api.privateObject, api.localActor, map[string]string{objectField: ""}, api.ownerToken, http.StatusBadRequest},
				{"unknown target", api.privateObject, api.localActor, map[string]string{objectField: "missing00000000"}, api.ownerToken, http.StatusBadRequest},
				{"permission-only change", api.privateObject, api.localActor, map[string]string{"permission": "edit"}, api.ownerToken, http.StatusOK},
				{"explicit unchanged target with permission change", api.privateObject, api.localActor, map[string]string{objectField: api.privateObject.Id, "permission": "edit"}, api.ownerToken, http.StatusOK},
				{"local recipient-only change", api.privateObject, api.localActor, map[string]string{"actor": api.ownerActor.Id}, api.ownerToken, http.StatusOK},
				{"remote recipient-only change leaves target unchanged", api.privateObject, api.localActor, map[string]string{"actor": api.remoteActor.Id}, api.ownerToken, http.StatusOK},
				{"existing remote share permission-only change", api.privateObject, api.remoteActor, map[string]string{"permission": "edit"}, api.ownerToken, http.StatusOK},
				{"invalid recipient uses record validation", api.privateObject, api.localActor, map[string]string{"actor": "missing00000000"}, api.ownerToken, http.StatusBadRequest},
				{"anonymous cannot update permission", api.privateObject, api.localActor, map[string]string{"permission": "edit"}, "", http.StatusNotFound},
				{"recipient cannot update owner's share", api.privateObject, api.localActor, map[string]string{"permission": "edit"}, api.strangerToken, http.StatusNotFound},
			})
		})
	}
}

// shareUpdateAPITest exercises share hooks through authenticated Records API
// requests, including PocketBase's authorization against the original record.
type shareUpdateAPITest struct {
	app                  *pbtests.TestApp
	mux                  http.Handler
	objectField          string
	objects, shares      *core.Collection
	ownerActor           *core.Record
	localActor           *core.Record
	remoteActor          *core.Record
	privateObject        *core.Record
	publicObject         *core.Record
	foreignPrivateObject *core.Record
	ownerToken           string
	strangerToken        string
}

type shareUpdateAPICase struct {
	name          string
	object, actor *core.Record
	patch         map[string]string
	token         string
	status        int
}

func newShareUpdateAPITest(t *testing.T, objectField string) *shareUpdateAPITest {
	t.Helper()
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	actors := core.NewBaseCollection("activitypub_actors")
	objects := core.NewBaseCollection(objectField + "s")
	shares := core.NewBaseCollection(objectField + "_share")
	actors.Fields.Add(
		&core.BoolField{Name: "is_local"},
		&core.RelationField{Name: "user", CollectionId: users.Id, MaxSelect: 1},
	)
	objects.Fields.Add(
		&core.BoolField{Name: "public"},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1, Required: true},
	)
	shares.Fields.Add(
		&core.RelationField{Name: objectField, CollectionId: objects.Id, MaxSelect: 1, Required: true},
		&core.RelationField{Name: "actor", CollectionId: actors.Id, MaxSelect: 1, Required: true},
		&core.SelectField{Name: "permission", Values: []string{"view", "edit"}, MaxSelect: 1, Required: true},
	)
	// Match the production owner path, so request authentication and
	// PocketBase's update-rule query participate in every PATCH.
	shares.UpdateRule = types.Pointer(objectField + ".author.user = @request.auth.id")
	shares.ViewRule = types.Pointer(objectField + ".author.user = @request.auth.id || actor.user = @request.auth.id")
	for _, collection := range []*core.Collection{actors, objects, shares} {
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}
	objects.ViewRule = types.Pointer("author.user = @request.auth.id || public = true || (@request.auth.id != \"\" && " + shares.Name + "_via_" + objectField + ".actor.user ?= @request.auth.id)")
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
	newUser := func(email string) (*core.Record, string) {
		t.Helper()
		user := save(users, map[string]any{"email": email, "password": "test-password"})
		token, err := user.NewAuthToken()
		if err != nil {
			t.Fatal(err)
		}
		return user, token
	}
	owner, ownerToken := newUser("owner@example.com")
	stranger, strangerToken := newUser("stranger@example.com")
	foreignOwner, _ := newUser("foreign-owner@example.com")
	ownerActor := save(actors, map[string]any{"is_local": true, "user": owner.Id})
	localActor := save(actors, map[string]any{"is_local": true, "user": stranger.Id})
	foreignActor := save(actors, map[string]any{"is_local": true, "user": foreignOwner.Id})
	remoteActor := save(actors, map[string]any{"is_local": false})
	privateObject := save(objects, map[string]any{"author": ownerActor.Id, "public": false})
	publicObject := save(objects, map[string]any{"author": ownerActor.Id, "public": true})
	foreignPrivateObject := save(objects, map[string]any{"author": foreignActor.Id, "public": false})

	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	return &shareUpdateAPITest{
		app: app, mux: mux, objectField: objectField, objects: objects, shares: shares,
		ownerActor: ownerActor, localActor: localActor, remoteActor: remoteActor,
		privateObject: privateObject, publicObject: publicObject, foreignPrivateObject: foreignPrivateObject,
		ownerToken: ownerToken, strangerToken: strangerToken,
	}
}

func (api *shareUpdateAPITest) newShare(t *testing.T, object, actor *core.Record) *core.Record {
	t.Helper()
	record := core.NewRecord(api.shares)
	record.Load(map[string]any{api.objectField: object.Id, "actor": actor.Id, "permission": "view"})
	if err := api.app.Save(record); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := api.app.Delete(record); err != nil {
			t.Error(err)
		}
	})
	return record
}

func (api *shareUpdateAPITest) assertRecipientAccess(t *testing.T, wantStatus int) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/api/collections/"+api.objects.Name+"/records/"+api.foreignPrivateObject.Id, nil)
	request.Header.Set("Authorization", api.strangerToken)
	response := httptest.NewRecorder()
	api.mux.ServeHTTP(response, request)
	if response.Code != wantStatus {
		t.Fatalf("recipient GET foreign private object: status %d; want %d: %s", response.Code, wantStatus, response.Body.String())
	}
}

func (api *shareUpdateAPITest) patch(t *testing.T, record *core.Record, patch map[string]string, token string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(patch)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPatch, "/api/collections/"+api.shares.Name+"/records/"+record.Id, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", token)
	}
	response := httptest.NewRecorder()
	api.app.ResetEventCalls()
	api.mux.ServeHTTP(response, request)
	return response
}

func (api *shareUpdateAPITest) runCases(t *testing.T, cases []shareUpdateAPICase) {
	t.Helper()
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			record := api.newShare(t, test.object, test.actor)
			api.assertRecipientAccess(t, http.StatusNotFound)
			response := api.patch(t, record, test.patch, test.token)
			if response.Code != test.status {
				t.Fatalf("PATCH status %d; want %d: %s", response.Code, test.status, response.Body.String())
			}
			want := map[string]string{api.objectField: test.object.Id, "actor": test.actor.Id, "permission": "view"}
			wantUpdates := 0
			if test.status == http.StatusOK {
				wantUpdates = 1
				for field, value := range test.patch {
					want[field] = value
				}
			}
			persisted, err := api.app.FindRecordById(api.shares.Name, record.Id)
			if err != nil {
				t.Fatal(err)
			}
			got := map[string]string{api.objectField: persisted.GetString(api.objectField), "actor": persisted.GetString("actor"), "permission": persisted.GetString("permission")}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("persisted share = %v; want %v", got, want)
			}
			if got := api.app.EventCalls["OnRecordUpdateExecute"]; got != wantUpdates {
				t.Errorf("record update events = %d; want %d", got, wantUpdates)
			}
			api.assertRecipientAccess(t, http.StatusNotFound)
		})
	}
}
