package routes

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"pocketbase/federation"
	"pocketbase/util"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/go-fed/httpsig"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func (w *knownKeyWorld) deliverOrderedActivity(handler http.Handler, actor, recipient *core.Record, activity pub.Activity) (int, error) {
	activity.Actor = pub.IRI(actor.GetString("iri"))
	body, err := json.Marshal(activity)
	if err != nil {
		return 0, err
	}
	path := strings.TrimPrefix(recipient.GetString("inbox"), "https://example.com")
	request := httptest.NewRequest(http.MethodPost, "https://example.com"+path, bytes.NewReader(body))
	ctx, cancel := context.WithTimeout(request.Context(), 10*time.Second)
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
	if err := signer.SignRequest(w.key, actor.GetString("iri")+"#main-key", request, body); err != nil {
		return 0, err
	}
	request.URL.Path = "/inbox"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		return response.Code, fmt.Errorf("response: %s", response.Body.String())
	}
	return response.Code, nil
}

type orderedInboxResult struct {
	status int
	err    error
}

func orderedInboxResultOK(t *testing.T, result <-chan orderedInboxResult) {
	t.Helper()
	select {
	case got := <-result:
		if got.err != nil || got.status != http.StatusOK {
			t.Fatalf("inbox response: %d, %v", got.status, got.err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("inbox request did not finish")
	}
}

func orderTrail(iri, actor string) *pub.Object {
	return &pub.Object{ID: pub.IRI(iri), Type: pub.DocumentType, AttributedTo: pub.IRI(actor),
		Name: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Morning walk")), Location: &pub.Place{Type: pub.PlaceType}}
}

func TestInboxOriginalAndRetractionShareReload(t *testing.T) {
	for _, kind := range []pub.ActivityVocabularyType{pub.LikeType, pub.FollowType, pub.CreateType} {
		t.Run(string(kind), func(t *testing.T) {
			w := newKnownKeyWorld(t, 5*time.Second)
			handler := knownKeyInboxRouter(t, w)
			for round := 0; round < 20; round++ {
				name := fmt.Sprintf("sender%02d", round)
				actor := w.sender(t, name, "remote.example", true)
				iri := fmt.Sprintf("https://remote.example/api/v1/trail/walk%011d", round)
				var object pub.Item = pub.IRI(iri)
				collection := "trail_like"
				if kind == pub.FollowType {
					object, collection = pub.IRI(w.recipient.GetString("iri")), "follows"
				}
				if kind == pub.LikeType {
					w.save(t, "trails", map[string]any{"iri": iri, "name": "Morning walk", "author": w.recipient.Id})
				}
				if kind == pub.CreateType {
					object, collection = orderTrail(iri, actor.GetString("iri")), "trails"
				}
				original := pub.Activity{ID: pub.IRI(fmt.Sprintf("https://remote.example/activities/first-%s-%d", kind, round)), Type: kind, Actor: pub.IRI(actor.GetString("iri")), Object: object}
				retraction := pub.Activity{ID: pub.IRI(fmt.Sprintf("https://remote.example/activities/second-%s-%d", kind, round)), Type: pub.UndoType, Object: &original}
				if kind == pub.CreateType {
					retraction.Type, retraction.Object = pub.DeleteType, pub.IRI(iri)
				}
				// The embedded original ID is deliberately independent of the
				// handler's actor/target identity when undoing a like or follow.
				if embedded, ok := retraction.Object.(*pub.Activity); ok {
					copy := *embedded
					copy.ID = "https://remote.example/activities/recorded-original"
					retraction.Object = &copy
				}
				writes := make(chan struct{})
				var releaseWrites sync.Once
				defer releaseWrites.Do(func() { close(writes) })
				hook := w.app.OnRecordCreate(collection).BindFunc(func(event *core.RecordEvent) error {
					<-writes
					return event.Next()
				})
				release, started := make(chan struct{}), make(chan struct{})
				path := "/api/v1/activitypub/user/" + name
				w.origin.held[path], w.origin.started[path] = release, started
				originalResult, undoResult := make(chan orderedInboxResult, 1), make(chan orderedInboxResult, 1)
				go func() {
					status, err := w.deliverOrderedActivity(handler, actor, w.recipient, original)
					originalResult <- orderedInboxResult{status, err}
				}()
				select {
				case <-started:
				case <-time.After(2 * time.Second):
					t.Fatal("refresh did not start")
				}
				go func() {
					status, err := w.deliverOrderedActivity(handler, actor, w.recipient, retraction)
					undoResult <- orderedInboxResult{status, err}
				}()
				waitForKnownKeyReloadCallers(t, 2)
				close(release)
				orderedInboxResultOK(t, undoResult)
				releaseWrites.Do(func() { close(writes) })
				orderedInboxResultOK(t, originalResult)
				w.app.OnRecordCreate(collection).Unbind(hook)
				records, err := w.app.FindAllRecords(collection)
				if err != nil || len(records) != 0 {
					t.Fatalf("round %d: %s remaining: %d, %v", round, collection, len(records), err)
				}
				if kind == pub.CreateType {
					feed, err := w.app.FindAllRecords("feed")
					if err != nil || len(feed) != 0 {
						t.Fatalf("round %d: feed remaining: %d, %v", round, len(feed), err)
					}
				}
			}
		})
	}
}

func TestInboxDeleteCoversParallelRecipientsAndUpdate(t *testing.T) {
	w := newKnownKeyWorld(t, 5*time.Second)
	handler := knownKeyInboxRouter(t, w)
	actor := w.sender(t, "sender", "remote.example", true)
	recipients := []*core.Record{w.recipient}
	for i := 1; i < 20; i++ {
		name := fmt.Sprintf("reader%02d", i)
		user := w.save(t, "users", map[string]any{"email": name + "@example.com", "username": name})
		iri := "https://example.com/api/v1/activitypub/user/" + name
		recipients = append(recipients, w.save(t, "activitypub_actors", map[string]any{"username": name, "preferred_username": name, "user": user.Id, "is_local": true, "domain": "example.com", "iri": iri, "inbox": iri + "/inbox", "public_key": w.publicKey}))
	}
	iri := "https://remote.example/api/v1/trail/morningwalk0001"
	original := pub.Activity{ID: "https://remote.example/activities/original", Type: pub.CreateType, Object: orderTrail(iri, actor.GetString("iri"))}
	deletion := pub.Activity{ID: "https://remote.example/activities/deletion", Type: pub.DeleteType, Object: pub.IRI(iri)}
	release, started := make(chan struct{}), make(chan struct{})
	w.origin.held["/api/v1/activitypub/user/sender"], w.origin.started["/api/v1/activitypub/user/sender"] = release, started
	results := make(chan orderedInboxResult, 21)
	go func() {
		status, err := w.deliverOrderedActivity(handler, actor, recipients[0], original)
		results <- orderedInboxResult{status, err}
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("refresh did not start")
	}
	for _, recipient := range recipients[1:] {
		go func() {
			status, err := w.deliverOrderedActivity(handler, actor, recipient, original)
			results <- orderedInboxResult{status, err}
		}()
	}
	go func() {
		status, err := w.deliverOrderedActivity(handler, actor, w.recipient, deletion)
		results <- orderedInboxResult{status, err}
	}()
	waitForKnownKeyReloadCallers(t, 21)
	close(release)
	for i := 0; i < 21; i++ {
		orderedInboxResultOK(t, results)
	}
	original.Type = pub.UpdateType
	status, err := w.deliverOrderedActivity(handler, actor, w.recipient, original)
	if status != http.StatusOK || err != nil {
		t.Fatalf("update response: %d, %v", status, err)
	}
	for _, collection := range []string{"trails", "feed"} {
		records, err := w.app.FindAllRecords(collection)
		if err != nil || len(records) != 0 {
			t.Fatalf("%s remaining: %d, %v", collection, len(records), err)
		}
	}
}

func TestInboxUndoLeavesLaterActionsAndOtherActors(t *testing.T) {
	w := newKnownKeyWorld(t)
	handler := knownKeyInboxRouter(t, w)
	actor := w.sender(t, "sender", "remote.example", true)
	other := w.sender(t, "another", "remote.example", true)
	iri := "https://example.com/api/v1/trail/morningwalk0001"
	w.save(t, "trails", map[string]any{"iri": iri, "name": "Morning walk", "author": w.recipient.Id})
	like := pub.Activity{ID: "https://remote.example/activities/like", Type: pub.LikeType, Object: pub.IRI(iri)}
	undo := pub.Activity{ID: "https://remote.example/activities/undo", Type: pub.UndoType, Object: &like}
	for index, step := range []struct {
		actor      *core.Record
		activity   pub.Activity
		own, other int
	}{{actor, like, 1, 0}, {actor, undo, 0, 0}, {actor, like, 1, 0}, {actor, undo, 0, 0}, {actor, undo, 0, 0}, {other, like, 0, 1}, {actor, like, 1, 1}, {actor, like, 1, 1}} {
		status, err := w.deliverOrderedActivity(handler, step.actor, w.recipient, step.activity)
		if status != http.StatusOK || err != nil {
			t.Fatalf("step %d activity %s: %d, %v", index, step.activity.Type, status, err)
		}
		records, err := w.app.FindAllRecords("trail_like")
		if err != nil {
			t.Fatal(err)
		}
		counts := map[string]int{}
		for _, record := range records {
			counts[record.GetString("actor")]++
		}
		if counts[actor.Id] != step.own || counts[other.Id] != step.other {
			t.Fatalf("step %d: likes per actor: %v, want %d and %d", index, counts, step.own, step.other)
		}
	}
}

func TestInboxDeleteRemovesFeedInsertedAfterDeletion(t *testing.T) {
	w := newKnownKeyWorld(t)
	handler := knownKeyInboxRouter(t, w)
	actor := w.sender(t, "sender", "remote.example", true)
	iri := "https://remote.example/api/v1/trail/morningwalk0001"
	original := pub.Activity{ID: "https://remote.example/activities/original", Type: pub.CreateType, Object: orderTrail(iri, actor.GetString("iri"))}
	deletion := pub.Activity{ID: "https://remote.example/activities/deletion", Type: pub.DeleteType, Object: pub.IRI(iri)}
	started, release := make(chan struct{}), make(chan struct{})
	var finish sync.Once
	defer finish.Do(func() { close(release) })
	w.app.OnRecordCreate("feed").BindFunc(func(event *core.RecordEvent) error {
		if event.Record.GetString("type") != "trail" {
			return event.Next()
		}
		close(started)
		<-release
		return event.Next()
	})
	results := make(chan orderedInboxResult, 1)
	go func() {
		status, err := w.deliverOrderedActivity(handler, actor, w.recipient, original)
		results <- orderedInboxResult{status, err}
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("feed insert did not start")
	}
	status, err := w.deliverOrderedActivity(handler, actor, w.recipient, deletion)
	if status != http.StatusOK || err != nil {
		t.Fatalf("delete response: %d, %v", status, err)
	}
	finish.Do(func() { close(release) })
	orderedInboxResultOK(t, results)
	for _, collection := range []string{"trails", "feed"} {
		records, err := w.app.FindAllRecords(collection)
		if err != nil || len(records) != 0 {
			t.Fatalf("%s remaining: %d, %v", collection, len(records), err)
		}
	}
}

func TestInboxDeleteRemembersObjectsCreatedAfterItsFirstLookup(t *testing.T) {
	for _, collection := range []string{"trails", "lists"} {
		t.Run(collection, func(t *testing.T) {
			w := newKnownKeyWorld(t)
			handler := knownKeyInboxRouter(t, w)
			actor := w.sender(t, "sender", "remote.example", true)
			kind, feedType := "trail", "trail"
			if collection == "lists" {
				kind, feedType = "list", "list"
			}
			iri := "https://remote.example/api/v1/" + kind + "/morningwalk0001"
			object := orderTrail(iri, actor.GetString("iri"))
			if collection == "lists" {
				object.Type, object.Location = pub.CollectionType, nil
			}
			original := pub.Activity{ID: "https://remote.example/activities/original", Type: pub.CreateType, Object: object}
			deletion := pub.Activity{ID: "https://remote.example/activities/deletion", Type: pub.DeleteType, Object: pub.IRI(iri)}
			lookupStarted, resumeLookup := make(chan struct{}), make(chan struct{})
			feedStarted, resumeFeed := make(chan struct{}), make(chan struct{})
			var finishLookup, finishFeed sync.Once
			defer finishLookup.Do(func() { close(resumeLookup) })
			defer finishFeed.Do(func() { close(resumeFeed) })
			var intercepted atomic.Bool
			db := w.app.ConcurrentDB().(*dbx.DB)
			previous := db.QueryLogFunc
			db.QueryLogFunc = func(ctx context.Context, elapsed time.Duration, query string, rows *sql.Rows, err error) {
				if previous != nil {
					previous(ctx, elapsed, query, rows, err)
				}
				if !strings.Contains(query, "`"+collection+"`") || !intercepted.CompareAndSwap(false, true) {
					return
				}
				// Consume the initial empty result before allowing the Create to
				// save. Delete's real handler will find the object afterwards.
				if err != nil || rows == nil || rows.Next() {
					t.Error("first delete lookup was not empty")
				}
				close(lookupStarted)
				<-resumeLookup
			}
			defer func() { db.QueryLogFunc = previous }()
			w.app.OnRecordCreate("feed").BindFunc(func(event *core.RecordEvent) error {
				if event.Record.GetString("type") != feedType {
					return event.Next()
				}
				close(feedStarted)
				<-resumeFeed
				return event.Next()
			})
			deleted, created := make(chan orderedInboxResult, 1), make(chan orderedInboxResult, 1)
			go func() {
				status, err := w.deliverOrderedActivity(handler, actor, w.recipient, deletion)
				deleted <- orderedInboxResult{status, err}
			}()
			select {
			case <-lookupStarted:
			case <-time.After(3 * time.Second):
				t.Fatal("first delete lookup did not start")
			}
			go func() {
				status, err := w.deliverOrderedActivity(handler, actor, w.recipient, original)
				created <- orderedInboxResult{status, err}
			}()
			select {
			case <-feedStarted:
			case <-time.After(3 * time.Second):
				t.Fatal("create did not reach its feed insert")
			}
			finishLookup.Do(func() { close(resumeLookup) })
			orderedInboxResultOK(t, deleted)
			objects, err := w.app.FindAllRecords(collection)
			if err != nil || len(objects) != 0 {
				t.Fatalf("objects before late feed: %d, %v", len(objects), err)
			}
			finishFeed.Do(func() { close(resumeFeed) })
			orderedInboxResultOK(t, created)
			feed, err := w.app.FindAllRecords("feed")
			if err != nil || len(feed) != 0 {
				t.Fatalf("feed remaining after late object deletion: %d, %v", len(feed), err)
			}
		})
	}
}

func TestInboxDeleteKeepsFeedForReusedItemIdentity(t *testing.T) {
	for _, replacement := range []string{"other-object", "other-author", "other-collection"} {
		t.Run(replacement, func(t *testing.T) {
			w := newKnownKeyWorld(t)
			handler := knownKeyInboxRouter(t, w)
			actor := w.sender(t, "sender", "remote.example", true)
			other := w.sender(t, "another", "remote.example", true)
			iri := "https://remote.example/api/v1/trail/morningwalk0001"
			original := pub.Activity{ID: "https://remote.example/activities/original", Type: pub.CreateType, Object: orderTrail(iri, actor.GetString("iri"))}
			deletion := pub.Activity{ID: "https://remote.example/activities/deletion", Type: pub.DeleteType, Object: pub.IRI(iri)}
			started, release := make(chan string, 1), make(chan struct{})
			var finish sync.Once
			defer finish.Do(func() { close(release) })
			feedHook := w.app.OnRecordCreate("feed").BindFunc(func(event *core.RecordEvent) error {
				if event.Record.GetString("type") != "trail" || event.Record.GetString("item") == "" {
					return event.Next()
				}
				select {
				case started <- event.Record.GetString("item"):
				default:
				}
				<-release
				return event.Next()
			})
			results := make(chan orderedInboxResult, 1)
			go func() {
				status, err := w.deliverOrderedActivity(handler, actor, w.recipient, original)
				results <- orderedInboxResult{status, err}
			}()
			var itemID string
			select {
			case itemID = <-started:
			case <-time.After(3 * time.Second):
				t.Fatal("feed insert did not start")
			}
			status, err := w.deliverOrderedActivity(handler, actor, w.recipient, deletion)
			if status != http.StatusOK || err != nil {
				t.Fatalf("delete response: %d, %v", status, err)
			}
			collection, feedType, newIRI, author := "trails", "trail", iri, actor.Id
			switch replacement {
			case "other-object":
				newIRI = "https://remote.example/api/v1/trail/afternoonwalk01"
			case "other-author":
				author = other.Id
			case "other-collection":
				collection, feedType, newIRI = "lists", "list", "https://remote.example/api/v1/list/morningwalk0001"
			}
			replacementRecord := w.save(t, collection, map[string]any{"id": itemID, "iri": newIRI, "author": author, "name": "Another walk"})
			w.app.OnRecordCreate("feed").Unbind(feedHook)
			// Saving directly deliberately avoids the held incoming insert.
			schema, err := w.app.FindCollectionByNameOrId("feed")
			if err != nil {
				t.Fatal(err)
			}
			feed := core.NewRecord(schema)
			feed.Load(map[string]any{"actor": w.recipient.Id, "item": itemID, "type": feedType})
			if err := w.app.SaveNoValidate(feed); err != nil {
				t.Fatal(err)
			}
			finish.Do(func() { close(release) })
			orderedInboxResultOK(t, results)
			if _, err := w.app.FindRecordById(collection, replacementRecord.Id); err != nil {
				t.Fatalf("replacement record removed: %v", err)
			}
			if _, err := w.app.FindRecordById("feed", feed.Id); err != nil {
				t.Fatalf("replacement feed removed: %v", err)
			}
		})
	}
}

// Keep the real Object layout so the library converts this test value to its
// usual Object. Only the second ID read pauses, after the original's marker
// check and before its normal handler. No production callback is replaced.
type orderedInboxObject pub.Object

type orderedInboxObjectPause struct {
	object          *orderedInboxObject
	started, resume chan struct{}
	calls           atomic.Int32
}

var orderedInboxObjectPaused atomic.Pointer[orderedInboxObjectPause]

func (object *orderedInboxObject) GetID() pub.ID {
	if pause := orderedInboxObjectPaused.Load(); pause != nil && pause.object == object && pause.calls.Add(1) == 2 {
		close(pause.started)
		<-pause.resume
	}
	return (*pub.Object)(object).GetID()
}

func (object *orderedInboxObject) GetLink() pub.IRI { return (*pub.Object)(object).GetLink() }
func (object *orderedInboxObject) GetType() pub.ActivityVocabularyType {
	return (*pub.Object)(object).GetType()
}
func (object *orderedInboxObject) IsLink() bool       { return (*pub.Object)(object).IsLink() }
func (object *orderedInboxObject) IsObject() bool     { return (*pub.Object)(object).IsObject() }
func (object *orderedInboxObject) IsCollection() bool { return (*pub.Object)(object).IsCollection() }

func TestInboxRetractionKeepsFeedForStoredLocalIdentity(t *testing.T) {
	for _, collection := range []string{"trails", "lists"} {
		for _, activityType := range []pub.ActivityVocabularyType{pub.CreateType, pub.UpdateType} {
			t.Run(collection+"/"+string(activityType), func(t *testing.T) {
				w := newKnownKeyWorld(t)
				handler := knownKeyInboxRouter(t, w)
				actor := w.sender(t, "sender", "remote.example", true)
				kind, feedType := "trail", "trail"
				if collection == "lists" {
					kind, feedType = "list", "list"
				}
				iri := "https://example.com/api/v1/" + kind + "/morningwalk0001"
				record := w.save(t, collection, map[string]any{"id": "morningwalk0001", "iri": iri, "author": actor.Id, "name": "Morning walk"})
				feed := w.save(t, "feed", map[string]any{"actor": w.recipient.Id, "item": record.Id, "type": feedType})
				if !util.RecordHasLocalIdentity(w.app, record) {
					t.Fatal("fixture has no stored local identity")
				}
				deletion := pub.Activity{ID: "https://remote.example/activities/deletion", Type: pub.DeleteType, Actor: pub.IRI(actor.GetString("iri")), Object: pub.IRI(iri)}
				if err := federation.ProcessDeleteActivity(w.app, actor, deletion); err == nil {
					t.Fatal("delete handler accepted the stored local identity")
				}
				payload := orderTrail(iri, actor.GetString("iri"))
				if collection == "lists" {
					payload.Type, payload.Location = pub.CollectionType, nil
				}
				object := (*orderedInboxObject)(payload)
				converted, err := pub.ToObject(object)
				if err != nil || converted != payload {
					t.Fatalf("test value did not convert to the real object: %v", err)
				}
				pause := &orderedInboxObjectPause{object: object, started: make(chan struct{}), resume: make(chan struct{})}
				var finish sync.Once
				defer finish.Do(func() { close(pause.resume) })
				orderedInboxObjectPaused.Store(pause)
				defer orderedInboxObjectPaused.Store(nil)
				result := make(chan error, 1)
				original := pub.Activity{Type: activityType, Actor: pub.IRI(actor.GetString("iri")), Object: object}
				go func() { result <- federation.ProcessInboxActivity(w.app, actor, w.recipient, original) }()
				select {
				case <-pause.started:
				case <-time.After(3 * time.Second):
					t.Fatal("original did not reach its handler")
				}
				status, err := w.deliverOrderedActivity(handler, actor, w.recipient, deletion)
				if status != http.StatusOK || err != nil {
					t.Fatalf("signed delete response: %d, %v", status, err)
				}
				if _, err := w.app.FindRecordById(collection, record.Id); err != nil {
					t.Fatalf("rejected signed delete changed own object: %v", err)
				}
				if _, err := w.app.FindRecordById("feed", feed.Id); err != nil {
					t.Fatalf("rejected direct deletion changed feed: %v", err)
				}
				finish.Do(func() { close(pause.resume) })
				select {
				case err := <-result:
					if err == nil || !strings.Contains(err.Error(), "origin does not match") {
						t.Fatalf("original did not reach its normal origin check: %v", err)
					}
				case <-time.After(3 * time.Second):
					t.Fatal("original did not finish")
				}
				if _, err := w.app.FindRecordById(collection, record.Id); err != nil {
					t.Fatalf("stored own object removed: %v", err)
				}
				if _, err := w.app.FindRecordById("feed", feed.Id); err != nil {
					t.Fatalf("rejected original and delete changed feed: %v", err)
				}
			})
		}
	}
}

func TestInboxMarkedOriginalSkipsHandlers(t *testing.T) {
	w := newKnownKeyWorld(t)
	handler := knownKeyInboxRouter(t, w)
	actor := w.sender(t, "sender", "remote.example", true)
	iri := "https://remote.example/api/v1/trail/morningwalk0001"
	deletion := pub.Activity{ID: "https://remote.example/activities/deletion", Type: pub.DeleteType, Object: pub.IRI(iri)}
	status, err := w.deliverOrderedActivity(handler, actor, w.recipient, deletion)
	if status != http.StatusOK || err != nil {
		t.Fatalf("delete response: %d, %v", status, err)
	}
	inserts := 0
	w.app.OnRecordCreate("trails").BindFunc(func(event *core.RecordEvent) error { inserts++; return event.Next() })
	for _, kind := range []pub.ActivityVocabularyType{pub.CreateType, pub.UpdateType} {
		activity := pub.Activity{ID: "https://remote.example/activities/original", Type: kind, Object: orderTrail(iri, actor.GetString("iri"))}
		status, err := w.deliverOrderedActivity(handler, actor, w.recipient, activity)
		if status != http.StatusOK || err != nil {
			t.Fatalf("original response: %d, %v", status, err)
		}
	}
	if inserts != 0 {
		t.Fatalf("marked originals reached the create hook %d times", inserts)
	}
}

func TestInboxUndoSecondPassRemovesEarlierLike(t *testing.T) {
	w := newKnownKeyWorld(t)
	handler := knownKeyInboxRouter(t, w)
	actor := w.sender(t, "sender", "remote.example", true)
	iri := "https://example.com/api/v1/trail/morningwalk0001"
	w.save(t, "trails", map[string]any{"iri": iri, "name": "Morning walk", "author": w.recipient.Id})
	like := pub.Activity{ID: "https://remote.example/activities/like", Type: pub.LikeType, Object: pub.IRI(iri)}
	undo := pub.Activity{ID: "https://remote.example/activities/undo", Type: pub.UndoType, Object: &like}
	warmup := pub.Activity{ID: "https://remote.example/activities/follow", Type: pub.FollowType, Object: pub.IRI(w.recipient.GetString("iri"))}
	if status, err := w.deliverOrderedActivity(handler, actor, w.recipient, warmup); status != http.StatusOK || err != nil {
		t.Fatalf("warm-up: %d, %v", status, err)
	}
	entered, write := make(chan struct{}), make(chan struct{})
	var enterOnce, writeOnce sync.Once
	defer writeOnce.Do(func() { close(write) })
	hook := w.app.OnRecordCreate("trail_like").BindFunc(func(event *core.RecordEvent) error {
		enterOnce.Do(func() { close(entered) })
		<-write
		return event.Next()
	})
	likeResult := make(chan orderedInboxResult, 1)
	go func() {
		status, err := w.deliverOrderedActivity(handler, actor, w.recipient, like)
		likeResult <- orderedInboxResult{status, err}
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("like did not reach its insert")
	}
	missing, resume := make(chan struct{}), make(chan struct{})
	var finish sync.Once
	defer finish.Do(func() { close(resume) })
	var intercepted atomic.Bool
	db := w.app.ConcurrentDB().(*dbx.DB)
	previous := db.QueryLogFunc
	db.QueryLogFunc = func(ctx context.Context, elapsed time.Duration, query string, rows *sql.Rows, err error) {
		if previous != nil {
			previous(ctx, elapsed, query, rows, err)
		}
		if !strings.Contains(query, "`trail_like`") || !strings.Contains(query, "SELECT") || !intercepted.CompareAndSwap(false, true) {
			return
		}
		// The original reached the inbox first but has not committed yet.
		if err != nil || rows == nil || rows.Next() {
			t.Error("first undo lookup was not empty")
		}
		close(missing)
		<-resume
	}
	defer func() { db.QueryLogFunc = previous }()
	undoResult := make(chan orderedInboxResult, 1)
	go func() {
		status, err := w.deliverOrderedActivity(handler, actor, w.recipient, undo)
		undoResult <- orderedInboxResult{status, err}
	}()
	select {
	case <-missing:
	case <-time.After(3 * time.Second):
		t.Fatal("first undo lookup did not start")
	}
	writeOnce.Do(func() { close(write) })
	orderedInboxResultOK(t, likeResult)
	w.app.OnRecordCreate("trail_like").Unbind(hook)
	finish.Do(func() { close(resume) })
	orderedInboxResultOK(t, undoResult)
	likes, err := w.app.FindAllRecords("trail_like")
	if err != nil || len(likes) != 0 {
		t.Fatalf("likes after second undo pass: %d, %v", len(likes), err)
	}
	later := pub.Activity{ID: "https://remote.example/activities/later", Type: pub.LikeType, Object: pub.IRI(iri)}
	if status, err := w.deliverOrderedActivity(handler, actor, w.recipient, later); status != http.StatusOK || err != nil {
		t.Fatalf("later like response: %d, %v", status, err)
	}
	likes, err = w.app.FindAllRecords("trail_like")
	if err != nil || len(likes) != 1 {
		t.Fatalf("later like was not stored: %d, %v", len(likes), err)
	}
}

func TestInboxUndoSecondPassKeepsLaterLike(t *testing.T) {
	w := newKnownKeyWorld(t)
	handler := knownKeyInboxRouter(t, w)
	actor := w.sender(t, "sender", "remote.example", true)
	iri := "https://example.com/api/v1/trail/morningwalk0001"
	w.save(t, "trails", map[string]any{"iri": iri, "name": "Morning walk", "author": w.recipient.Id})
	like := pub.Activity{ID: "https://remote.example/activities/like", Type: pub.LikeType, Object: pub.IRI(iri)}
	undo := pub.Activity{ID: "https://remote.example/activities/undo", Type: pub.UndoType, Object: &like}
	missing, resume := make(chan struct{}), make(chan struct{})
	var finish sync.Once
	defer finish.Do(func() { close(resume) })
	var intercepted atomic.Bool
	db := w.app.ConcurrentDB().(*dbx.DB)
	previous := db.QueryLogFunc
	db.QueryLogFunc = func(ctx context.Context, elapsed time.Duration, query string, rows *sql.Rows, err error) {
		if previous != nil {
			previous(ctx, elapsed, query, rows, err)
		}
		if !strings.Contains(query, "`trail_like`") || !strings.Contains(query, "SELECT") || !intercepted.CompareAndSwap(false, true) {
			return
		}
		if err != nil || rows == nil || rows.Next() {
			t.Error("first undo lookup was not empty")
		}
		close(missing)
		<-resume
	}
	defer func() { db.QueryLogFunc = previous }()
	undoResult := make(chan orderedInboxResult, 1)
	go func() {
		status, err := w.deliverOrderedActivity(handler, actor, w.recipient, undo)
		undoResult <- orderedInboxResult{status, err}
	}()
	select {
	case <-missing:
	case <-time.After(3 * time.Second):
		t.Fatal("first undo lookup did not start")
	}
	if status, err := w.deliverOrderedActivity(handler, actor, w.recipient, like); status != http.StatusOK || err != nil {
		t.Fatalf("later like response: %d, %v", status, err)
	}
	finish.Do(func() { close(resume) })
	orderedInboxResultOK(t, undoResult)
	likes, err := w.app.FindAllRecords("trail_like")
	if err != nil || len(likes) != 1 {
		t.Fatalf("later like was removed: %d, %v", len(likes), err)
	}
}

func TestInboxActionsAfterUndoApplyInArrivalOrder(t *testing.T) {
	for _, kind := range []pub.ActivityVocabularyType{pub.LikeType, pub.FollowType} {
		for _, repeated := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/repeated=%t", kind, repeated), func(t *testing.T) {
				w := newKnownKeyWorld(t)
				handler := knownKeyInboxRouter(t, w)
				actor := w.sender(t, "sender", "remote.example", true)
				iri := "https://example.com/api/v1/trail/morningwalk0001"
				var object pub.Item = pub.IRI(iri)
				collection, field := "trail_like", "actor"
				if kind == pub.FollowType {
					object, collection, field = pub.IRI(w.recipient.GetString("iri")), "follows", "follower"
				} else {
					w.save(t, "trails", map[string]any{"iri": iri, "name": "Morning walk", "author": w.recipient.Id})
				}
				first := pub.Activity{ID: "https://remote.example/activities/first", Type: kind, Object: object}
				undo := pub.Activity{ID: "https://remote.example/activities/undo", Type: pub.UndoType, Object: &first}
				next := pub.Activity{ID: "https://remote.example/activities/next", Type: kind, Object: object}
				steps := []struct {
					activity       pub.Activity
					count, accepts int
				}{}
				accepts := 0
				if repeated {
					if kind == pub.FollowType {
						accepts = 1
					}
					steps = append(steps, struct {
						activity       pub.Activity
						count, accepts int
					}{first, 1, accepts})
				}
				steps = append(steps, struct {
					activity       pub.Activity
					count, accepts int
				}{undo, 0, accepts})
				if repeated {
					steps = append(steps, struct {
						activity       pub.Activity
						count, accepts int
					}{undo, 0, accepts})
				}
				if kind == pub.FollowType {
					accepts++
				}
				steps = append(steps, struct {
					activity       pub.Activity
					count, accepts int
				}{next, 1, accepts})
				for index, step := range steps {
					if status, err := w.deliverOrderedActivity(handler, actor, w.recipient, step.activity); status != http.StatusOK || err != nil {
						t.Fatalf("step %d: %d, %v", index, status, err)
					}
					records, err := w.app.FindAllRecords(collection, dbx.HashExp{field: actor.Id})
					if err != nil || len(records) != step.count {
						t.Fatalf("step %d: %s=%d, want %d: %v", index, collection, len(records), step.count, err)
					}
					accepted, err := w.app.FindAllRecords("activitypub_activities", dbx.HashExp{"type": "Accept"})
					if err != nil || len(accepted) != step.accepts {
						t.Fatalf("step %d: accepts=%d, want %d: %v", index, len(accepted), step.accepts, err)
					}
				}
			})
		}
	}
}

func TestInboxLaterActionsShareUndoReload(t *testing.T) {
	for _, kind := range []pub.ActivityVocabularyType{pub.LikeType, pub.FollowType} {
		t.Run(string(kind), func(t *testing.T) {
			w := newKnownKeyWorld(t, 5*time.Second)
			handler := knownKeyInboxRouter(t, w)
			kept := 0
			for round := 0; round < 20; round++ {
				name := fmt.Sprintf("sender%02d", round)
				actor := w.sender(t, name, "remote.example", true)
				iri := fmt.Sprintf("https://example.com/api/v1/trail/walk%011d", round)
				var object pub.Item = pub.IRI(iri)
				collection, field := "trail_like", "actor"
				if kind == pub.FollowType {
					object, collection, field = pub.IRI(w.recipient.GetString("iri")), "follows", "follower"
				} else {
					w.save(t, "trails", map[string]any{"iri": iri, "name": "Morning walk", "author": w.recipient.Id})
				}
				original := pub.Activity{ID: pub.IRI(fmt.Sprintf("https://remote.example/activities/first-%d", round)), Type: kind, Actor: pub.IRI(actor.GetString("iri")), Object: object}
				undo := pub.Activity{ID: pub.IRI(fmt.Sprintf("https://remote.example/activities/undo-%d", round)), Type: pub.UndoType, Object: &original}
				next := pub.Activity{ID: pub.IRI(fmt.Sprintf("https://remote.example/activities/next-%d", round)), Type: kind, Object: object}
				release, started := make(chan struct{}), make(chan struct{})
				path := "/api/v1/activitypub/user/" + name
				w.origin.held[path], w.origin.started[path] = release, started
				undone, applied := make(chan orderedInboxResult, 1), make(chan orderedInboxResult, 1)
				go func() {
					status, err := w.deliverOrderedActivity(handler, actor, w.recipient, undo)
					undone <- orderedInboxResult{status, err}
				}()
				select {
				case <-started:
				case <-time.After(2 * time.Second):
					t.Fatal("refresh did not start")
				}
				go func() {
					status, err := w.deliverOrderedActivity(handler, actor, w.recipient, next)
					applied <- orderedInboxResult{status, err}
				}()
				waitForKnownKeyReloadCallers(t, 2)
				close(release)
				orderedInboxResultOK(t, undone)
				orderedInboxResultOK(t, applied)
				records, err := w.app.FindAllRecords(collection, dbx.HashExp{field: actor.Id})
				if err != nil {
					t.Fatal(err)
				}
				if len(records) == 1 {
					kept++
				} else if len(records) != 0 {
					t.Fatalf("round %d: %d records", round, len(records))
				}
			}
			t.Logf("later %s actions applied: %d/20", kind, kept)
			if kept != 20 {
				t.Fatalf("later actions applied: %d/20", kept)
			}
			if kind == pub.FollowType {
				accepted, err := w.app.FindAllRecords("activitypub_activities", dbx.HashExp{"type": "Accept"})
				if err != nil || len(accepted) != 20 {
					t.Fatalf("accepts=%d, want 20: %v", len(accepted), err)
				}
			}
		})
	}
}

func TestInboxOverlappingUndosKeepLatestArrival(t *testing.T) {
	w := newKnownKeyWorld(t)
	handler := knownKeyInboxRouter(t, w)
	actor := w.sender(t, "sender", "remote.example", true)
	iri := "https://example.com/api/v1/trail/morningwalk0001"
	w.save(t, "trails", map[string]any{"iri": iri, "name": "Morning walk", "author": w.recipient.Id})
	like := pub.Activity{ID: "https://remote.example/activities/like", Type: pub.LikeType, Object: pub.IRI(iri)}
	first := pub.Activity{ID: "https://remote.example/activities/first-undo", Type: pub.UndoType, Object: &like}
	second := pub.Activity{ID: "https://remote.example/activities/second-undo", Type: pub.UndoType, Object: &like}
	missing, resume := make(chan struct{}), make(chan struct{})
	var finishLookup sync.Once
	defer finishLookup.Do(func() { close(resume) })
	var intercepted atomic.Bool
	db := w.app.ConcurrentDB().(*dbx.DB)
	previous := db.QueryLogFunc
	db.QueryLogFunc = func(ctx context.Context, elapsed time.Duration, query string, rows *sql.Rows, err error) {
		if previous != nil {
			previous(ctx, elapsed, query, rows, err)
		}
		if !strings.Contains(query, "`trail_like`") || !strings.Contains(query, "SELECT") || !intercepted.CompareAndSwap(false, true) {
			return
		}
		if err != nil || rows == nil || rows.Next() {
			t.Error("first undo lookup was not empty")
		}
		close(missing)
		<-resume
	}
	defer func() { db.QueryLogFunc = previous }()
	firstResult := make(chan orderedInboxResult, 1)
	go func() {
		status, err := w.deliverOrderedActivity(handler, actor, w.recipient, first)
		firstResult <- orderedInboxResult{status, err}
	}()
	select {
	case <-missing:
	case <-time.After(3 * time.Second):
		t.Fatal("first undo did not reach its lookup")
	}
	entered, write := make(chan struct{}), make(chan struct{})
	var finishWrite, enterOnce sync.Once
	defer finishWrite.Do(func() { close(write) })
	hook := w.app.OnRecordCreate("trail_like").BindFunc(func(event *core.RecordEvent) error {
		enterOnce.Do(func() { close(entered) })
		<-write
		return event.Next()
	})
	likeResult := make(chan orderedInboxResult, 1)
	go func() {
		status, err := w.deliverOrderedActivity(handler, actor, w.recipient, like)
		likeResult <- orderedInboxResult{status, err}
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("like did not reach its insert")
	}
	if status, err := w.deliverOrderedActivity(handler, actor, w.recipient, second); status != http.StatusOK || err != nil {
		t.Fatalf("second undo: %d, %v", status, err)
	}
	// The newer Undo has completed both empty lookups before the older Undo
	// resumes and tries to remember its own, earlier arrival.
	finishLookup.Do(func() { close(resume) })
	orderedInboxResultOK(t, firstResult)
	finishWrite.Do(func() { close(write) })
	orderedInboxResultOK(t, likeResult)
	w.app.OnRecordCreate("trail_like").Unbind(hook)
	likes, err := w.app.FindAllRecords("trail_like")
	if err != nil || len(likes) != 0 {
		t.Fatalf("like between the undos remained: %d, %v", len(likes), err)
	}
	later := pub.Activity{ID: "https://remote.example/activities/later", Type: pub.LikeType, Object: pub.IRI(iri)}
	if status, err := w.deliverOrderedActivity(handler, actor, w.recipient, later); status != http.StatusOK || err != nil {
		t.Fatalf("later like: %d, %v", status, err)
	}
	likes, err = w.app.FindAllRecords("trail_like")
	if err != nil || len(likes) != 1 {
		t.Fatalf("later like not stored: %d, %v", len(likes), err)
	}
}
