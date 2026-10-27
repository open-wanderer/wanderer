package routes

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"pocketbase/federation"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestInboxSelfDeletionRequiresPermanentGoneAtTheActorOrigin(t *testing.T) {
	for _, state := range []string{"gone", "not-found", "unavailable", "redirect-host", "another-object", "local-actor"} {
		t.Run(state, func(t *testing.T) {
			f := setupHandleLookupTest(t, "https://local.example", http.StatusOK)
			t.Setenv("POCKETBASE_PROXY_SECRET", "gone-test-secret")
			f.actors.Fields.Add(&core.TextField{Name: "inbox"})
			if err := f.app.Save(f.actors); err != nil {
				t.Fatal(err)
			}
			recipient := f.localActor(t, "bob", "https://local.example", "local.example")
			recipient.Set("inbox", recipient.GetString("iri")+"/inbox")
			if err := f.app.Save(recipient); err != nil {
				t.Fatal(err)
			}
			actor := f.remoteActor(t, "alice", "remote.example")
			actor.Set("public_key", "")
			if state == "local-actor" {
				actor.Set("is_local", true)
			}
			if err := f.app.Save(actor); err != nil {
				t.Fatal(err)
			}
			trails, _ := f.app.FindCollectionByNameOrId("trails")
			trails.Fields.GetByName("author").(*core.RelationField).CascadeDelete = true
			if err := f.app.Save(trails); err != nil {
				t.Fatal(err)
			}
			copy := f.object(t, "trails", actor.Id, "https://remote.example", "copy00000000001", false, false)
			comments := core.NewBaseCollection("comments")
			comments.Fields.Add(&core.TextField{Name: "text"}, &core.RelationField{Name: "author", CollectionId: f.actors.Id, MaxSelect: 1, CascadeDelete: true})
			if err := f.app.Save(comments); err != nil {
				t.Fatal(err)
			}
			summitLogs := core.NewBaseCollection("summit_logs")
			summitLogs.Fields.Add(&core.TextField{Name: "iri"}, &core.RelationField{Name: "author", CollectionId: f.actors.Id, MaxSelect: 1, CascadeDelete: true})
			if err := f.app.Save(summitLogs); err != nil {
				t.Fatal(err)
			}
			waypoints := core.NewBaseCollection("waypoints")
			waypoints.Fields.Add(&core.TextField{Name: "iri"}, &core.RelationField{Name: "author", CollectionId: f.actors.Id, MaxSelect: 1, CascadeDelete: true})
			if err := f.app.Save(waypoints); err != nil {
				t.Fatal(err)
			}
			comment := f.record(t, "comments", map[string]any{"text": "Remote contribution", "author": actor.Id})
			previous := getInboxActorByIRI
			getInboxActorByIRI = func(core.App, context.Context, string, bool) (*core.Record, error) {
				return nil, federation.ErrActorNotFound
			}
			t.Cleanup(func() { getInboxActorByIRI = previous })
			var probes, foreignRequests atomic.Int32
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
					probes.Add(1)
					if request.URL.Host != "remote.example" {
						foreignRequests.Add(1)
					}
					response, err := richTextSyncTransport{map[string]any{}}.RoundTrip(request)
					response.StatusCode = http.StatusGone
					switch state {
					case "not-found":
						response.StatusCode = http.StatusNotFound
					case "unavailable":
						response.StatusCode = http.StatusServiceUnavailable
					case "redirect-host":
						response.StatusCode = http.StatusFound
						response.Header.Set("Location", "https://another.example/actor")
					}
					return response, err
				}}}
			}
			object := actor.GetString("iri")
			if state == "another-object" {
				object = recipient.GetString("iri")
			}
			body := fmt.Sprintf(`{"type":"Delete","actor":%q,"object":%q}`, actor.GetString("iri"), object)
			request := httptest.NewRequest(http.MethodPost, recipient.GetString("inbox"), strings.NewReader(body))
			request.Header.Set("X-Internal-Secret", "gone-test-secret")
			request.Header.Set("X-Forwarded-Path", "/api/v1/activitypub/user/bob/inbox")
			event := &core.RequestEvent{App: f.app}
			event.Request, event.Response = request, httptest.NewRecorder()
			err := ActivitypubActivityProcess(event)
			if state == "gone" {
				if err != nil {
					t.Fatalf("confirmed self-deletion failed: %v", err)
				}
			} else {
				var apiError *router.ApiError
				if !errors.As(err, &apiError) || apiError.Status != http.StatusUnauthorized {
					t.Fatalf("unconfirmed self-deletion accepted: %v", err)
				}
			}
			for _, record := range []*core.Record{actor, copy, comment} {
				_, err := f.app.FindRecordById(record.Collection(), record.Id)
				if (err != nil) != (state == "gone") {
					t.Fatalf("%s cleanup=%v for %s", record.Collection().Name, err != nil, state)
				}
			}
			want := int32(1)
			if state == "another-object" || state == "local-actor" {
				want = 0
			}
			if probes.Load() != want || foreignRequests.Load() != 0 {
				t.Fatalf("origin probes=%d foreign=%d", probes.Load(), foreignRequests.Load())
			}
		})
	}
}
