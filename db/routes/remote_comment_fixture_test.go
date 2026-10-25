package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/types"
)

type remoteCommentFixture struct {
	t   *testing.T
	app *pbtests.TestApp
}

func newRemoteCommentFixture(t *testing.T) *remoteCommentFixture {
	t.Helper()
	t.Setenv("ORIGIN", "https://example.com")
	base := newRemoteSyncFixture(t)
	base.actors.Fields.Add(&core.TextField{Name: "preferred_username"}, &core.TextField{Name: "username"}, &core.TextField{Name: "inbox"})
	if err := base.app.Save(base.actors); err != nil {
		t.Fatal(err)
	}
	comments := core.NewBaseCollection("comments")
	comments.Fields.Add(&core.URLField{Name: "iri"}, &core.TextField{Name: "text"}, &core.RelationField{Name: "author", CollectionId: base.actors.Id, MaxSelect: 1}, &core.RelationField{Name: "trail", CollectionId: base.trails.Id, MaxSelect: 1}, &core.AutodateField{Name: "created", OnCreate: true}, &core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
	comments.ListRule, comments.ViewRule = types.Pointer("trail.public=true"), types.Pointer("trail.public=true")
	if err := base.app.Save(comments); err != nil {
		t.Fatal(err)
	}
	return &remoteCommentFixture{t: t, app: base.app}
}

func (f *remoteCommentFixture) save(name string, values map[string]any) *core.Record {
	f.t.Helper()
	collection, err := f.app.FindCollectionByNameOrId(name)
	if err != nil {
		f.t.Fatal(err)
	}
	record := core.NewRecord(collection)
	record.Load(values)
	if name == "users" && values["password"] == nil {
		record.SetPassword("comment-test-password")
	}
	if err := f.app.Save(record); err != nil {
		f.t.Fatal(err)
	}
	return record
}

func (f *remoteCommentFixture) localActor(name string) *core.Record {
	f.t.Helper()
	user := f.save("users", map[string]any{"email": "comment-" + name + "@example.com", "username": name})
	f.save("settings", map[string]any{"user": user.Id})
	return f.save("activitypub_actors", map[string]any{"user": user.Id, "preferred_username": name, "is_local": true, "iri": "https://example.com/api/v1/activitypub/user/" + name, "domain": "example.com"})
}

func useRemoteCommentClient(t *testing.T, client *http.Client) {
	t.Helper()
	previous := newRemoteCommentSyncHTTPClient
	newRemoteCommentSyncHTTPClient = func() *http.Client { return client }
	t.Cleanup(func() { newRemoteCommentSyncHTTPClient = previous })
}

func runConfirmedCommentPolicySync(t *testing.T, f *remoteCommentFixture, trail *core.Record) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/remote/trail/"+trail.Id+"/comments", nil)
	request.SetPathValue("id", trail.Id)
	response := httptest.NewRecorder()
	event := &core.RequestEvent{App: f.app}
	event.Request, event.Response = request, response
	if err := RemoteTrailCommentsList(event); err != nil || response.Code != http.StatusOK {
		t.Fatalf("remote comment sync failed: status=%d error=%v", response.Code, err)
	}
}
