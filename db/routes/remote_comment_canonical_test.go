package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type canonicalCommentReply struct {
	status   int
	body     map[string]any
	location string
}

// canonicalCommentNetwork serves the origin's comment list and the comment
// endpoints of other hosts, and records every request.
type canonicalCommentNetwork struct {
	mu       sync.Mutex
	list     []map[string]any
	replies  map[string]canonicalCommentReply
	requests []string
}

func (network *canonicalCommentNetwork) RoundTrip(request *http.Request) (*http.Response, error) {
	network.mu.Lock()
	network.requests = append(network.requests, request.URL.Host+request.URL.Path)
	reply, found := network.replies[request.URL.Host+request.URL.Path]
	network.mu.Unlock()
	if request.URL.Host == "origin.example" && request.URL.Path == "/api/v1/comment" {
		reply, found = canonicalCommentReply{status: http.StatusOK, body: map[string]any{"items": network.list}}, true
	}
	if !found {
		reply = canonicalCommentReply{status: http.StatusNotFound, body: map[string]any{}}
	}
	body, err := json.Marshal(reply.body)
	if err != nil {
		return nil, err
	}
	header := http.Header{"Content-Type": []string{"application/json"}}
	if reply.location != "" {
		header.Set("Location", reply.location)
	}
	return &http.Response{StatusCode: reply.status, Header: header, Request: request, Body: io.NopCloser(strings.NewReader(string(body)))}, nil
}

func (network *canonicalCommentNetwork) count(prefix string) int {
	network.mu.Lock()
	defer network.mu.Unlock()
	count := 0
	for _, request := range network.requests {
		if strings.HasPrefix(request, prefix) {
			count++
		}
	}
	return count
}

func TestRemoteCommentSyncReadsOtherHostsCommentsFromTheirOwnHost(t *testing.T) {
	f := newRemoteCommentFixture(t)
	f.app.Settings().Logs.MaxDays = 0
	network := &canonicalCommentNetwork{replies: map[string]canonicalCommentReply{}}
	useRemoteCommentClient(t, &http.Client{Transport: network})
	remote := func(name, host string) *core.Record {
		iri := "https://" + host + "/api/v1/activitypub/user/" + name
		return f.save("activitypub_actors", map[string]any{"preferred_username": name, "is_local": false, "domain": host, "public_key": "test-key", "iri": iri, "last_fetched": time.Now()})
	}
	alice, carol, erin, riley := remote("alice", "origin.example"), remote("carol", "third.example"), remote("erin", "third.example"), remote("riley", "another.example")
	local := f.localActor("local")
	trailIRI := "https://origin.example/api/v1/trail/originaltrail1"
	trail := f.save("trails", map[string]any{"name": "Remote trail", "author": alice.Id, "public": true, "iri": trailIRI})
	other := f.save("trails", map[string]any{"name": "Other trail", "author": alice.Id, "public": true, "iri": "https://origin.example/api/v1/trail/originaltrail2"})
	private := f.save("trails", map[string]any{"name": "Private trail", "author": local.Id})

	canonical := func(id, text string, author *core.Record, parent string) {
		network.replies["third.example/api/v1/comment/"+id] = canonicalCommentReply{status: http.StatusOK, body: map[string]any{
			"id": id, "iri": "https://third.example/api/v1/comment/" + id, "text": text, "author": "x", "trail": "y",
			"expand": map[string]any{"author": map[string]any{"iri": author.GetString("iri")}, "trail": map[string]any{"iri": parent}}}}
	}
	listed := func(iri string) map[string]any {
		return map[string]any{"iri": iri, "text": "Origin text", "author": local.Id,
			"expand": map[string]any{"author": map[string]any{"iri": riley.GetString("iri")}}}
	}
	third := func(id string) string { return "https://third.example/api/v1/comment/" + id }

	// Accepted only with the comment host's content and author.
	canonical("accepted", "Canonical text", carol, trailIRI)
	canonical("refreshed", "Edited text", carol, trailIRI)
	refreshed := f.save("comments", map[string]any{"iri": third("refreshed"), "trail": trail.Id, "author": carol.Id, "text": "Stale text"})
	// Refused: local or other-host authors, another trail, mismatching identity.
	canonical("local-author", "As local", local, trailIRI)
	canonical("other-host-author", "As riley", riley, trailIRI)
	canonical("other-trail", "Elsewhere", carol, other.GetString("iri"))
	canonical("no-trail", "Without parent", carol, "")
	canonical("id-mismatch", "Other id", carol, trailIRI)
	network.replies["third.example/api/v1/comment/id-mismatch"].body["id"] = "another"
	canonical("iri-mismatch", "Other iri", carol, trailIRI)
	network.replies["third.example/api/v1/comment/iri-mismatch"].body["iri"] = "https://fourth.example/api/v1/comment/iri-mismatch"
	// Refused: an author that is local here, registered under a former host
	// of this instance whose domain now serves the comment.
	former := f.save("users", map[string]any{"email": "former@example.com", "username": "former"})
	f.save("settings", map[string]any{"user": former.Id})
	formerActor := f.save("activitypub_actors", map[string]any{"user": former.Id, "preferred_username": "former", "username": "former", "is_local": true,
		"domain": "former.example", "iri": "https://former.example/api/v1/activitypub/user/former", "inbox": "https://former.example/api/v1/activitypub/user/former/inbox",
		"public_key": "-----BEGIN PUBLIC KEY-----\nx\n-----END PUBLIC KEY-----", "last_fetched": time.Now()})
	network.replies["former.example/api/v1/comment/former-local"] = canonicalCommentReply{status: http.StatusOK, body: map[string]any{
		"id": "former-local", "iri": "https://former.example/api/v1/comment/former-local", "text": "As a local user",
		"expand": map[string]any{"author": map[string]any{"iri": formerActor.GetString("iri")}, "trail": map[string]any{"iri": trailIRI}}}}
	// Refused: the stored copy belongs to another author or trail.
	canonical("owned-by-erin", "Now carol", carol, trailIRI)
	ownedByErin := f.save("comments", map[string]any{"iri": third("owned-by-erin"), "trail": trail.Id, "author": erin.Id, "text": "Erin text"})
	canonical("on-other-trail", "Moved", carol, trailIRI)
	onOtherTrail := f.save("comments", map[string]any{"iri": third("on-other-trail"), "trail": other.Id, "author": carol.Id, "text": "Other trail text"})
	// Never taken over: local and historical local comments.
	localComment := f.save("comments", map[string]any{"trail": private.Id, "author": local.Id, "text": "Local text"})
	localComment.Set("iri", "https://example.com/api/v1/comment/"+localComment.Id)
	historical := f.save("comments", map[string]any{"trail": trail.Id, "author": local.Id, "text": "Historical text"})
	historical.Set("iri", "https://old.example/api/v1/comment/"+historical.Id)
	for _, record := range []*core.Record{localComment, historical} {
		if err := f.app.SaveNoValidate(record); err != nil {
			t.Fatal(err)
		}
	}
	network.replies["old.example/api/v1/comment/"+historical.Id] = canonicalCommentReply{status: http.StatusOK, body: map[string]any{
		"id": historical.Id, "iri": historical.GetString("iri"), "text": "Taken over",
		"expand": map[string]any{"author": map[string]any{"iri": "https://old.example/api/v1/activitypub/user/old"}, "trail": map[string]any{"iri": trailIRI}}}}
	// Gone and redirected comments are skipped, and the stored copy is kept.
	gone := f.save("comments", map[string]any{"iri": third("gone"), "trail": trail.Id, "author": carol.Id, "text": "Kept text"})
	network.replies["third.example/api/v1/comment/redirect"] = canonicalCommentReply{status: http.StatusFound, location: "https://another.example/api/v1/comment/redirect", body: map[string]any{}}

	network.list = []map[string]any{}
	for _, id := range []string{"accepted", "refreshed", "local-author", "other-host-author", "other-trail", "no-trail", "id-mismatch", "iri-mismatch", "owned-by-erin", "on-other-trail", "gone", "redirect", "accepted"} {
		network.list = append(network.list, listed(third(id)))
	}
	network.list = append(network.list, listed(localComment.GetString("iri")), listed(historical.GetString("iri")), listed("https://former.example/api/v1/comment/former-local"),
		listed("https://mastodon.example/users/someone/statuses/1"), listed("https://third.example/api/v1/comment/a/b"))
	// One invalid origin entry does not discard the others.
	network.list = append(network.list, map[string]any{"iri": "https://origin.example/api/v1/comment/bad id", "text": "Invalid", "expand": map[string]any{"author": map[string]any{"iri": alice.GetString("iri")}}},
		map[string]any{"iri": "https://origin.example/api/v1/comment/valid", "text": "Origin comment", "expand": map[string]any{"author": map[string]any{"iri": alice.GetString("iri")}}})
	for i := 0; i < 40; i++ {
		network.list = append(network.list, listed(third(fmt.Sprintf("bulk%d", i))))
	}

	request := httptest.NewRequest(http.MethodGet, "/remote/trail/"+trail.Id+"/comments", nil)
	request.SetPathValue("id", trail.Id)
	response := httptest.NewRecorder()
	event := &core.RequestEvent{App: f.app}
	event.Request, event.Response = request, response
	if err := RemoteTrailCommentsList(event); err != nil || response.Code != http.StatusOK {
		t.Fatalf("comment list: %v %d", err, response.Code)
	}

	expect := func(iri, author, trailID, text string) {
		t.Helper()
		record, err := f.app.FindFirstRecordByData("comments", "iri", iri)
		if err != nil || record.GetString("author") != author || record.GetString("trail") != trailID || record.GetString("text") != text {
			t.Fatalf("%s: got %v %v", iri, record, err)
		}
	}
	expect(third("accepted"), carol.Id, trail.Id, "Canonical text")
	expect(third("refreshed"), carol.Id, trail.Id, "Edited text")
	expect(third("owned-by-erin"), erin.Id, trail.Id, ownedByErin.GetString("text"))
	expect(third("on-other-trail"), carol.Id, other.Id, onOtherTrail.GetString("text"))
	expect(third("gone"), carol.Id, trail.Id, gone.GetString("text"))
	expect(localComment.GetString("iri"), local.Id, private.Id, "Local text")
	expect(historical.GetString("iri"), local.Id, trail.Id, "Historical text")
	expect("https://origin.example/api/v1/comment/valid", alice.Id, trail.Id, "Origin comment")
	for _, id := range []string{"local-author", "other-host-author", "other-trail", "no-trail", "id-mismatch", "iri-mismatch", "redirect"} {
		if record, _ := f.app.FindFirstRecordByData("comments", "iri", third(id)); record != nil {
			t.Fatalf("%s accepted: %v", id, record)
		}
	}
	if record, _ := f.app.FindFirstRecordByData("comments", "iri", "https://former.example/api/v1/comment/former-local"); record != nil {
		t.Fatalf("comment attributed to a local actor: %v", record)
	}
	if refreshed.GetString("text") == "Edited text" {
		t.Fatal("fixture changed unexpectedly")
	}
	if network.count("another.example") != 0 || network.count("old.example") != 0 || network.count("example.com") != 0 || network.count("mastodon.example") > 1 {
		t.Fatalf("unexpected requests: %v", network.requests)
	}
	if fetched := network.count("third.example/api/v1/comment/"); fetched > 30 || network.count("third.example/api/v1/comment/accepted") != 1 {
		t.Fatalf("canonical fetches not bounded or deduplicated: %d", fetched)
	}
}
