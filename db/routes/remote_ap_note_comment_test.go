package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type activityPubCommentNetwork struct {
	*canonicalCommentNetwork
	t *testing.T
}

func (network activityPubCommentNetwork) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.URL.Host == "notes.example" {
		if request.Method != http.MethodGet || request.URL.RawQuery != "" || !strings.Contains(request.Header.Get("Accept"), "application/activity+json") {
			network.t.Errorf("Note was not fetched through content negotiation at its exact URI: %s %s", request.Method, request.URL)
		}
	}
	return network.canonicalCommentNetwork.RoundTrip(request)
}

func TestRemoteCommentSyncImportsOnlyConfirmedPublicActivityPubReplies(t *testing.T) {
	f := newRemoteCommentFixture(t)
	f.app.Settings().Logs.MaxDays = 0
	network := &canonicalCommentNetwork{replies: map[string]canonicalCommentReply{}}
	useRemoteCommentClient(t, &http.Client{Transport: activityPubCommentNetwork{network, t}})
	author := f.save("activitypub_actors", map[string]any{
		"preferred_username": "carol", "is_local": false, "domain": "notes.example", "public_key": "test-key",
		"iri": "https://notes.example/users/carol", "last_fetched": time.Now(),
	})
	origin := f.save("activitypub_actors", map[string]any{
		"preferred_username": "alice", "is_local": false, "domain": "origin.example", "public_key": "test-key",
		"iri": "https://origin.example/api/v1/activitypub/user/alice", "last_fetched": time.Now(),
	})
	trail := f.save("trails", map[string]any{"name": "Remote", "author": origin.Id, "public": true, "iri": "https://origin.example/api/v1/trail/original"})
	local := f.localActor("local")
	uri := func(id string) string { return "https://notes.example/users/carol/statuses/" + id }
	for _, id := range []string{"accepted", "wrong-id", "wrong-parent", "foreign-author", "local-author", "ambiguous-author", "private", "missing-address", "wrong-type", "redirect", "gone"} {
		object := map[string]any{
			"@context": "https://www.w3.org/ns/activitystreams", "type": "Note", "id": uri(id),
			"attributedTo": author.GetString("iri"), "inReplyTo": trail.GetString("iri"),
			"to": []any{"https://www.w3.org/ns/activitystreams#Public"}, "content": `<p onclick="alert(1)"><strong>Canonical text</strong><script>discard</script></p>`,
		}
		reply := canonicalCommentReply{status: http.StatusOK, body: object}
		switch id {
		case "wrong-id":
			object["id"] = "https://elsewhere.example/users/carol/statuses/wrong-id"
		case "wrong-parent":
			object["inReplyTo"] = "https://origin.example/api/v1/trail/other"
		case "foreign-author":
			object["attributedTo"] = "https://elsewhere.example/users/carol"
		case "local-author":
			object["attributedTo"] = local.GetString("iri")
		case "ambiguous-author":
			object["attributedTo"] = []any{author.GetString("iri"), "https://notes.example/users/other"}
		case "private":
			object["to"] = []any{origin.GetString("iri")}
		case "missing-address":
			delete(object, "to")
		case "wrong-type":
			object["type"] = "Person"
		case "redirect":
			reply.status, reply.location = http.StatusFound, "https://elsewhere.example/users/carol/statuses/redirect"
		case "gone":
			reply.status = http.StatusGone
		}
		network.replies["notes.example/users/carol/statuses/"+id] = reply
		network.list = append(network.list, map[string]any{"iri": uri(id), "text": "Origin text must not be used", "expand": map[string]any{"author": map[string]any{"iri": local.GetString("iri")}}})
	}
	// A permanent missing reply is removed only after its own host confirms it.
	gone := f.save("comments", map[string]any{"iri": uri("gone"), "text": "Existing reply", "trail": trail.Id, "author": author.Id})
	network.list = append(network.list, network.list[0])
	request := httptest.NewRequest(http.MethodGet, "/remote/trail/"+trail.Id+"/comments", nil)
	request.SetPathValue("id", trail.Id)
	event := &core.RequestEvent{App: f.app}
	event.Request, event.Response = request, httptest.NewRecorder()
	if err := RemoteTrailCommentsList(event); err != nil {
		t.Fatal(err)
	}
	accepted, err := f.app.FindFirstRecordByData("comments", "iri", uri("accepted"))
	if err != nil || accepted.GetString("author") != author.Id || accepted.GetString("trail") != trail.Id || !strings.Contains(accepted.GetString("text"), "Canonical text") || strings.Contains(accepted.GetString("text"), "onclick") || strings.Contains(accepted.GetString("text"), "discard") {
		t.Fatalf("public Note did not retain its verified fields and safe HTML: %v %v", accepted, err)
	}
	comments, err := f.app.FindAllRecords("comments")
	if err != nil || len(comments) != 1 {
		t.Fatalf("unconfirmed or private Note was imported: count=%d error=%v", len(comments), err)
	}
	if retained, err := f.app.FindRecordById("comments", gone.Id); err == nil {
		t.Fatalf("confirmed missing reply was retained: %v", retained)
	}
	if network.count("elsewhere.example") != 0 || network.count("notes.example/users/carol/statuses/accepted") != 1 {
		t.Fatalf("unexpected redirect or duplicate fetch: %v", network.requests)
	}
}
