package routes

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type invalidRemoteCommentTransport struct{}

func (invalidRemoteCommentTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	items := []map[string]any{}
	for _, id := range []string{"first", "bad id", "last"} {
		items = append(items, map[string]any{
			"id": id, "iri": "https://remote.example/api/v1/comment/" + id, "text": id,
			"expand": map[string]any{"author": map[string]any{"iri": "https://remote.example/api/v1/activitypub/user/alice"}},
		})
	}
	body, err := json.Marshal(map[string]any{"items": items})
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Request: request, Body: io.NopCloser(strings.NewReader(string(body)))}, err
}

func TestRemoteCommentsKeepValidEntriesAroundAnInvalidRecord(t *testing.T) {
	f := newRemoteCommentFixture(t)
	f.app.Settings().Logs.MaxDays = 0
	author := f.save("activitypub_actors", map[string]any{
		"preferred_username": "alice", "is_local": false, "domain": "remote.example", "public_key": "test-key",
		"iri": "https://remote.example/api/v1/activitypub/user/alice", "last_fetched": time.Now(),
	})
	trail := f.save("trails", map[string]any{"name": "Remote", "author": author.Id, "public": true, "iri": "https://remote.example/api/v1/trail/original"})
	useRemoteCommentClient(t, &http.Client{Transport: invalidRemoteCommentTransport{}})
	request := httptest.NewRequest(http.MethodGet, "/remote/trail/"+trail.Id+"/comments", nil)
	request.SetPathValue("id", trail.Id)
	response := httptest.NewRecorder()
	event := &core.RequestEvent{App: f.app}
	event.Request, event.Response = request, response
	if err := RemoteTrailCommentsList(event); err != nil || response.Code != http.StatusOK {
		t.Fatalf("comment page failed: status=%d error=%v body=%s", response.Code, err, response.Body)
	}
	comments, err := f.app.FindAllRecords("comments")
	if err != nil || len(comments) != 2 {
		t.Fatalf("an invalid record discarded valid comments: count=%d error=%v body=%s", len(comments), err, response.Body)
	}
	for _, id := range []string{"first", "last"} {
		comment, err := f.app.FindFirstRecordByData("comments", "iri", "https://remote.example/api/v1/comment/"+id)
		if err != nil || comment.GetString("author") != author.Id || comment.GetString("trail") != trail.Id || comment.GetString("text") != id {
			t.Fatalf("valid comment %s lost its fields: %v %v", id, comment, err)
		}
	}
}
