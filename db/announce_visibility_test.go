package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"pocketbase/federation"
	"pocketbase/util"

	pub "github.com/go-ap/activitypub"
	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func TestDirectedAnnounceGrantsRecipientAccessWithoutPublishingNewCopies(t *testing.T) {
	f := newLocalHandleFixture(t, "https://example.com", false)
	save := func(collection string, values map[string]any) *core.Record {
		return saveInboundRecord(t, f.app, collection, values)
	}
	sender := save("activitypub_actors", map[string]any{
		"username": "Remote", "preferred_username": "remote", "iri": "https://peer.example/api/v1/activitypub/user/remote",
		"inbox": "https://peer.example/inbox", "public_key": "remote-key",
	})
	recipient := f.actor
	f.settings.Set("notifications", map[string]any{
		"trail_share": map[string]bool{"web": false, "email": false}, "list_share": map[string]bool{"web": false, "email": false},
	})
	if err := f.app.Save(f.settings); err != nil {
		t.Fatal(err)
	}
	token := f.token
	router, err := apis.NewRouter(f.app)
	if err != nil {
		t.Fatal(err)
	}
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	client := newInboundSearchClient(t)
	setupEventHandlers(&pocketbase.PocketBase{App: f.app}, client)
	for _, kind := range []string{"trail", "list"} {
		collection := map[string]string{"trail": "trails", "list": "lists"}[kind]
		object := &pub.Object{ID: pub.IRI("https://peer.example/api/v1/" + kind + "/shared"), Type: pub.NoteType,
			AttributedTo: pub.IRI(sender.GetString("iri")), Name: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Shared")),
			Location: &pub.Place{Type: pub.PlaceType}, Tag: pub.ItemCollection{}, To: pub.ItemCollection{pub.PublicNS}}
		activity := pub.Activity{Type: pub.AnnounceType, Actor: pub.IRI(sender.GetString("iri")), Object: object,
			To: pub.ItemCollection{pub.IRI(recipient.GetString("iri"))}}
		if err := federation.ProcessAnnounceActivity(f.app, sender, activity); err != nil {
			t.Fatal(err)
		}
		record, err := f.app.FindFirstRecordByData(collection, "iri", object.ID.String())
		if err != nil || record.GetBool("public") {
			t.Fatalf("directed %s became public: %v %v", kind, record, err)
		}
		for _, auth := range []string{"", token} {
			request := httptest.NewRequest(http.MethodGet, "/api/collections/"+collection+"/records/"+record.Id, nil)
			request.Header.Set("Authorization", auth)
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			want := http.StatusNotFound
			if auth != "" {
				want = http.StatusOK
			}
			if response.Code != want {
				t.Fatalf("%s access status=%d want=%d: %s", kind, response.Code, want, response.Body.String())
			}
		}
		// The owner's delayed publication arrives after the directed share.
		// It must promote the same copy without waiting for a full sync.
		activity.Type = pub.CreateType
		activity.To = pub.ItemCollection{pub.PublicNS}
		if kind == "trail" {
			record, err = util.TrailFromActivity(activity, f.app, sender)
		} else {
			record, err = util.ListFromActivity(activity, f.app, sender)
		}
		if err != nil || !record.GetBool("public") {
			t.Fatalf("delayed owner %s publication left a private share copy: %v %v", kind, record, err)
		}
		request := httptest.NewRequest(http.MethodGet, "/api/collections/"+collection+"/records/"+record.Id, nil)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("confirmed %s publication not readable anonymously: %d %s", kind, response.Code, response.Body.String())
		}
		activity.Type = pub.AnnounceType
		activity.To = pub.ItemCollection{pub.IRI(recipient.GetString("iri"))}
		if err := federation.ProcessAnnounceActivity(f.app, sender, activity); err != nil {
			t.Fatal(err)
		}
		published, err := f.app.FindRecordById(collection, record.Id)
		if err != nil || !published.GetBool("public") {
			t.Fatalf("later directed share hid confirmed %s: %v %v", kind, published, err)
		}
		// Ordinary publication still produces a public copy.
		object.ID = pub.IRI("https://peer.example/api/v1/" + kind + "/published")
		activity.Type = pub.CreateType
		activity.To = pub.ItemCollection{pub.PublicNS}
		if kind == "trail" {
			record, err = util.TrailFromActivity(activity, f.app, sender)
		} else {
			record, err = util.ListFromActivity(activity, f.app, sender)
		}
		if err != nil || !record.GetBool("public") {
			t.Fatalf("ordinary %s publication lost visibility: %v %v", kind, record, err)
		}
	}
}

func saveInboundRecord(t *testing.T, app core.App, collection string, values map[string]any) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(col)
	record.Load(values)
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}
	return record
}

func newInboundSearchClient(t *testing.T) meilisearch.ServiceManager {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"uid":1,"indexUid":"trails","status":"succeeded","type":"documentAdditionOrUpdate"}`))
			return
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"taskUid":1,"status":"enqueued","type":"documentAdditionOrUpdate","enqueuedAt":"2026-10-17T00:00:00Z"}`))
	}))
	t.Cleanup(server.Close)
	return meilisearch.New(server.URL)
}
