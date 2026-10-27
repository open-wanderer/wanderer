package routes

import (
	"github.com/pocketbase/pocketbase/core"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRemoteCommentSyncRejectsNoteAttributionToAFreshCachedForeignHost(t *testing.T) {
	f := newRemoteCommentFixture(t)
	origin := f.save("activitypub_actors", map[string]any{"iri": "https://origin.example/users/alice", "domain": "origin.example", "public_key": "test-key", "last_fetched": time.Now()})
	author := f.save("activitypub_actors", map[string]any{"iri": "https://notes.example/users/carol", "domain": "notes.example", "public_key": "test-key", "last_fetched": time.Now()})
	foreign := f.save("activitypub_actors", map[string]any{"iri": "https://elsewhere.example/users/carol", "domain": "elsewhere.example", "public_key": "test-key", "last_fetched": time.Now()})
	trail := f.save("trails", map[string]any{"name": "Remote", "author": origin.Id, "public": true, "iri": "https://origin.example/api/v1/trail/original"})
	network := &canonicalCommentNetwork{replies: map[string]canonicalCommentReply{}}
	for _, entry := range []struct {
		id     string
		author *core.Record
	}{{"own-host", author}, {"cached-foreign-host", foreign}} {
		iri := "https://notes.example/notes/" + entry.id
		network.list = append(network.list, map[string]any{"iri": iri})
		network.replies["notes.example/notes/"+entry.id] = canonicalCommentReply{status: http.StatusOK, body: map[string]any{
			"id": iri, "type": "Note", "attributedTo": entry.author.GetString("iri"), "inReplyTo": trail.GetString("iri"),
			"to": "https://www.w3.org/ns/activitystreams#Public", "content": "Confirmed " + entry.id,
		}}
	}
	useRemoteCommentClient(t, &http.Client{Transport: network})
	runConfirmedCommentPolicySync(t, f, trail)
	accepted, err := f.app.FindFirstRecordByData("comments", "iri", "https://notes.example/notes/own-host")
	if err != nil || accepted.GetString("author") != author.Id || accepted.GetString("text") != "Confirmed own-host" {
		t.Fatalf("same-host reply was not saved under its confirmed author: record=%v error=%v", accepted, err)
	}
	if record, _ := f.app.FindFirstRecordByData("comments", "iri", "https://notes.example/notes/cached-foreign-host"); record != nil {
		t.Fatalf("Note was attributed to a cached actor of another host: %v", record)
	}
	if network.count("notes.example/notes/") != 2 || network.count("elsewhere.example") != 0 {
		t.Fatalf("unexpected canonical author requests: %v", network.requests)
	}
}

func TestRemoteCommentSyncFollowsOnlyTheExactTLSUpgradeForCommentsAndNotes(t *testing.T) {
	for _, kind := range []string{"comment", "note"} {
		for _, scenario := range []string{"upgrade", "host", "port", "path", "query", "downgrade", "second-hop"} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				f := newRemoteCommentFixture(t)
				origin := f.save("activitypub_actors", map[string]any{"iri": "https://origin.example/users/alice", "domain": "origin.example", "public_key": "test-key", "last_fetched": time.Now()})
				author := f.save("activitypub_actors", map[string]any{"iri": "https://third.example/users/carol", "domain": "third.example", "public_key": "test-key", "last_fetched": time.Now()})
				trail := f.save("trails", map[string]any{"name": "Remote", "author": origin.Id, "public": true, "iri": "https://origin.example/api/v1/trail/original"})
				path := "/api/v1/comment/reply"
				if kind == "note" {
					path = "/notes/reply"
				}
				iri := "http://third.example" + path
				if scenario == "downgrade" {
					iri = "https://third.example" + path
				}
				var requests atomic.Int32
				client := &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
					if request.URL.Host == "origin.example" {
						return richTextSyncTransport{map[string]any{"items": []any{map[string]any{"iri": iri}}}}.RoundTrip(request)
					}
					count := requests.Add(1)
					if count == 1 || scenario == "second-hop" && count == 2 {
						target := *request.URL
						target.Scheme = "https"
						switch scenario {
						case "host":
							target.Host = "other.example"
						case "port":
							target.Host = "third.example:8443"
						case "path":
							target.Path = "/changed"
						case "query":
							target.RawQuery = "other=query"
						case "downgrade":
							target.Scheme = "http"
						}
						return &http.Response{StatusCode: http.StatusMovedPermanently, Header: http.Header{"Location": {target.String()}}, Body: http.NoBody, Request: request}, nil
					}
					// A changed endpoint cannot prove the original identity merely
					// by echoing it; its redirect must already have been refused.
					objectIRI := "https://third.example" + path
					payload := map[string]any{"id": "reply", "iri": objectIRI, "text": "Confirmed over TLS", "expand": map[string]any{
						"author": map[string]any{"iri": author.GetString("iri")}, "trail": map[string]any{"iri": trail.GetString("iri")},
					}}
					if kind == "note" {
						payload = map[string]any{"id": objectIRI, "type": "Note", "attributedTo": author.GetString("iri"), "inReplyTo": trail.GetString("iri"),
							"to": "https://www.w3.org/ns/activitystreams#Public", "content": "Confirmed over TLS"}
					}
					return richTextSyncTransport{payload}.RoundTrip(request)
				}}}
				useRemoteCommentClient(t, client)
				runConfirmedCommentPolicySync(t, f, trail)
				comments, err := f.app.FindAllRecords("comments")
				if err != nil {
					t.Fatal(err)
				}
				if scenario == "upgrade" {
					if len(comments) != 1 || requests.Load() != 2 || comments[0].GetString("iri") != iri || comments[0].GetString("author") != author.Id || !strings.Contains(comments[0].GetString("text"), "Confirmed over TLS") {
						t.Fatalf("same-host TLS reply was not imported: requests=%d comments=%v", requests.Load(), comments)
					}
				} else {
					want := int32(1)
					if scenario == "second-hop" {
						want = 2
					}
					if len(comments) != 0 || requests.Load() != want {
						t.Fatalf("canonical reply followed a changed endpoint: requests=%d want=%d comments=%v", requests.Load(), want, comments)
					}
				}
			})
		}
	}
}
