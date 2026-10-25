package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type commentSyncTransport struct{ items []map[string]any }

func (transport commentSyncTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	body, err := json.Marshal(map[string]any{"items": transport.items})
	if err != nil {
		return nil, err
	}
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Request: request, Body: io.NopCloser(strings.NewReader(string(body)))}, nil
}

func TestRemoteCommentSyncKeepsStoredIdentityOwnerAndParent(t *testing.T) {
	f := newRemoteCommentFixture(t)
	f.app.Settings().Logs.MaxDays = 0
	local := f.localActor("local")
	remote := f.save("activitypub_actors", map[string]any{"preferred_username": "remote", "is_local": false,
		"iri": "https://remote.example/api/v1/activitypub/user/remote", "domain": "remote.example", "public_key": "test-key", "last_fetched": time.Now()})
	other := f.save("activitypub_actors", map[string]any{"preferred_username": "other", "is_local": false,
		"iri": "https://remote.example/api/v1/activitypub/user/other", "domain": "remote.example", "public_key": "test-key", "last_fetched": time.Now()})
	trail := f.save("trails", map[string]any{"name": "Remote trail", "author": remote.Id, "public": true, "iri": "https://remote.example/api/v1/trail/original"})
	private := f.save("trails", map[string]any{"name": "Private trail", "author": local.Id})
	protected := []*core.Record{}
	items := []map[string]any{}
	for index, scenario := range []string{"local-iri", "historical-local", "other-origin", "other-owner", "other-parent"} {
		author, parent, origin := remote, trail, "https://remote.example"
		if scenario == "local-iri" {
			origin = "https://example.com"
			author, parent = local, private
		}
		if scenario == "historical-local" {
			author, parent = local, private
		}
		if scenario == "other-origin" {
			origin = "https://third.example"
		}
		if scenario == "other-owner" {
			author = other
		}
		if scenario == "other-parent" {
			parent = private
		}
		iri := fmt.Sprintf("%s/api/v1/comment/protected%d", origin, index)
		comment := f.save("comments", map[string]any{"author": author.Id, "trail": parent.Id, "iri": iri, "text": "Protected " + scenario})
		protected = append(protected, comment)
		items = append(items, map[string]any{"iri": iri, "text": "Replacement", "author": local.Id,
			"expand": map[string]any{"author": map[string]any{"iri": remote.GetString("iri")}}})
	}
	items = append(items, map[string]any{"iri": "https://remote.example/api/v1/comment/local-attribution", "text": "Local attribution",
		"expand": map[string]any{"author": map[string]any{"iri": local.GetString("iri")}}})
	valid := f.save("comments", map[string]any{"author": remote.Id, "trail": trail.Id, "iri": "https://remote.example/api/v1/comment/valid", "text": "Old remote text"})
	for _, id := range []string{"valid", "new"} {
		items = append(items, map[string]any{"iri": "https://remote.example/api/v1/comment/" + id, "text": "Accepted remote " + id,
			"author": local.Id, "trail": private.Id, "id": protected[0].Id,
			"expand": map[string]any{"author": map[string]any{"iri": remote.GetString("iri")}}})
	}
	useRemoteCommentClient(t, &http.Client{Transport: commentSyncTransport{items}})
	request := httptest.NewRequest(http.MethodGet, "/remote/trail/"+trail.Id+"/comments", nil)
	request.SetPathValue("id", trail.Id)
	response := httptest.NewRecorder()
	event := &core.RequestEvent{App: f.app}
	event.Request, event.Response = request, response
	if err := RemoteTrailCommentsList(event); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK {
		t.Fatalf("comment list status: %d %s", response.Code, response.Body.String())
	}
	for _, before := range protected {
		after, err := f.app.FindRecordById("comments", before.Id)
		if err != nil || after.GetString("text") != before.GetString("text") || after.GetString("author") != before.GetString("author") || after.GetString("trail") != before.GetString("trail") {
			t.Fatalf("sync changed protected comment %s: %v %v", before.GetString("text"), after, err)
		}
	}
	count, err := f.app.CountRecords("comments")
	if err != nil || count != int64(len(protected)+2) {
		t.Fatalf("unexpected comments created: %d %v", count, err)
	}
	for _, iri := range []string{valid.GetString("iri"), "https://remote.example/api/v1/comment/new"} {
		comment, err := f.app.FindFirstRecordByData("comments", "iri", iri)
		if err != nil || comment.GetString("author") != remote.Id || comment.GetString("trail") != trail.Id || !strings.HasPrefix(comment.GetString("text"), "Accepted remote") {
			t.Fatalf("valid origin comment rejected or raw fields accepted: %v %v", comment, err)
		}
	}
}
