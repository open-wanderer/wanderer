package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/go-fed/httpsig"
	"github.com/pocketbase/pocketbase/core"
)

// The first deliveries of a new trail or list reach their recipients at the
// same time after the shared key refresh. Every recipient gets its feed entry.
func TestInboxFirstRemoteObjectReachesEveryRecipient(t *testing.T) {
	for _, collection := range []string{"trails", "lists"} {
		for _, count := range []int{5, 20} {
			t.Run(fmt.Sprintf("%s-%d", collection, count), func(t *testing.T) {
				// The held actor fetch must cover the three-second join barrier.
				w := newKnownKeyWorld(t, 4*time.Second)
				handler := knownKeyInboxRouter(t, w)
				actor := w.sender(t, "sender", "remote.example", true)
				recipients := []*core.Record{w.recipient}
				for i := 1; i < count; i++ {
					name := fmt.Sprintf("reader%02d", i)
					user := w.save(t, "users", map[string]any{"email": name + "@example.com", "username": name})
					iri := "https://example.com/api/v1/activitypub/user/" + name
					recipients = append(recipients, w.save(t, "activitypub_actors", map[string]any{"username": name, "preferred_username": name, "user": user.Id, "is_local": true, "domain": "example.com", "iri": iri, "inbox": iri + "/inbox", "public_key": w.publicKey}))
				}
				// The application schema must retain its real uniqueness constraint.
				schema, err := w.app.FindCollectionByNameOrId(collection)
				if err != nil {
					t.Fatal(err)
				}
				uniqueIRI := false
				for _, index := range schema.Indexes {
					uniqueIRI = uniqueIRI || (strings.Contains(index, "UNIQUE INDEX") && strings.Contains(index, "(`iri`)"))
				}
				if !uniqueIRI {
					t.Fatalf("%s schema has no unique IRI index", collection)
				}
				path := "/api/v1/activitypub/user/sender"
				refreshRelease, refreshStarted := make(chan struct{}), make(chan struct{})
				w.origin.held[path], w.origin.started[path] = refreshRelease, refreshStarted
				var finishRefresh sync.Once
				defer finishRefresh.Do(func() { close(refreshRelease) })
				// Hold every insert until all recipients try to store the object.
				writeRelease := make(chan struct{})
				var finishWrite sync.Once
				defer finishWrite.Do(func() { close(writeRelease) })
				var writes sync.WaitGroup
				writes.Add(count)
				w.app.OnRecordCreate(collection).BindFunc(func(event *core.RecordEvent) error {
					writes.Done()
					<-writeRelease
					return event.Next()
				})
				type result struct {
					status int
					err    error
				}
				results := make(chan result, count)
				deliver := func(recipient *core.Record) {
					status, err := w.deliverFirstObject(handler, actor, recipient, collection)
					results <- result{status, err}
				}
				go deliver(recipients[0])
				select {
				case <-refreshStarted:
				case <-time.After(2 * time.Second):
					t.Fatal("actor refresh did not start")
				}
				for _, recipient := range recipients[1:] {
					go deliver(recipient)
				}
				waitForKnownKeyReloadCallers(t, count)
				finishRefresh.Do(func() { close(refreshRelease) })
				inserting := make(chan struct{})
				go func() { writes.Wait(); close(inserting) }()
				select {
				case <-inserting:
				case <-time.After(3 * time.Second):
					t.Fatal("recipients did not all try to store the object")
				}
				finishWrite.Do(func() { close(writeRelease) })
				for range recipients {
					select {
					case r := <-results:
						if r.err != nil || r.status != http.StatusOK {
							t.Errorf("recipient response: %d, %v", r.status, r.err)
						}
					case <-time.After(5 * time.Second):
						t.Fatal("recipient delivery did not finish")
					}
				}
				stored, err := w.app.FindAllRecords(collection)
				if err != nil || len(stored) != 1 {
					t.Fatalf("stored %s: %d, %v", collection, len(stored), err)
				}
				feed, err := w.app.FindAllRecords("feed")
				if err != nil || len(feed) != count {
					t.Fatalf("first feed entries: %d, want %d; error: %v", len(feed), count, err)
				}
				if got := w.origin.count(path); got != 1 {
					t.Fatalf("actor refresh requests: %d", got)
				}
			})
		}
	}
}

func (w *knownKeyWorld) deliverFirstObject(handler http.Handler, actor, recipient *core.Record, collection string, kinds ...pub.ActivityVocabularyType) (int, error) {
	iri := actor.GetString("iri")
	object := &pub.Object{ID: "https://remote.example/api/v1/trail/firsttrail00001", Type: pub.DocumentType,
		AttributedTo: pub.IRI(iri), Name: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Morning walk")),
		Location: &pub.Place{Type: pub.PlaceType}}
	if collection == "lists" {
		object = &pub.Object{ID: "https://remote.example/api/v1/list/firstlist000001", Type: pub.CollectionType,
			AttributedTo: pub.IRI(iri), Name: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Favourites"))}
	}
	kind := pub.CreateType
	activityID := "https://remote.example/activities/first-" + collection
	if len(kinds) > 0 {
		kind = kinds[0]
		activityID += "-" + string(kind)
	}
	body, err := json.Marshal(pub.Activity{ID: pub.IRI(activityID), Type: kind, Actor: pub.IRI(iri), Object: object})
	if err != nil {
		return 0, err
	}
	path := strings.TrimPrefix(recipient.GetString("inbox"), "https://example.com")
	request := httptest.NewRequest(http.MethodPost, "https://example.com"+path, bytes.NewReader(body))
	ctx, cancel := context.WithTimeout(request.Context(), 5*time.Second)
	defer cancel()
	request = request.WithContext(ctx)
	request.Header.Set("X-Internal-Secret", "known-key-test-secret")
	request.Header.Set("X-Forwarded-Path", path)
	request.Header.Set("Content-Type", "application/activity+json")
	request.Header.Set("Date", strings.ReplaceAll(time.Now().UTC().Format(time.RFC1123), "UTC", "GMT"))
	request.Header.Set("Host", "example.com")
	signer, _, err := httpsig.NewSigner([]httpsig.Algorithm{httpsig.RSA_SHA256}, httpsig.DigestSha256, []string{"(request-target)", "Date", "Digest", "Content-Type", "Host"}, httpsig.Signature, 60)
	if err != nil {
		return 0, err
	}
	if err := signer.SignRequest(w.key, iri+"#main-key", request, body); err != nil {
		return 0, err
	}
	request.URL.Path = "/inbox"
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if body := strings.TrimSpace(recorder.Body.String()); body != "null" {
		return recorder.Code, fmt.Errorf("processing error: %s", body)
	}
	return recorder.Code, nil
}

// Requests that share one refresh wait in reloadKnownInboxActor itself.
func waitForKnownKeyReloadCallers(t *testing.T, count int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		callers := countInboxActorReloadWaiters()
		if callers >= count {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("requests waiting for the refresh: %d, want %d", callers, count)
		}
		time.Sleep(5 * time.Millisecond)
	}
}
