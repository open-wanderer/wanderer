package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"pocketbase/routes"

	pub "github.com/go-ap/activitypub"
	"github.com/go-fed/httpsig"
	"github.com/pocketbase/pocketbase/core"
)

type trailMentionInboxFixture struct {
	app *core.BaseApp
	key *rsa.PrivateKey
}

func newTrailMentionInboxFixture(t *testing.T) *trailMentionInboxFixture {
	t.Helper()
	app := newRealSchemaApp(t)
	t.Setenv("POCKETBASE_PROXY_SECRET", "mention-inbox-secret")
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return &trailMentionInboxFixture{app: app, key: key}
}

func (f *trailMentionInboxFixture) actor(t *testing.T, name string, local bool) *core.Record {
	t.Helper()
	collection, err := f.app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	actor := core.NewRecord(collection)
	origin := "https://remote.example"
	if local {
		origin = "https://example.com"
		users, err := f.app.FindCollectionByNameOrId("users")
		if err != nil {
			t.Fatal(err)
		}
		user := core.NewRecord(users)
		user.Set("username", name)
		user.Set("email", name+"@example.test")
		user.SetPassword("mention-inbox-test-password")
		if err := f.app.Save(user); err != nil {
			t.Fatal(err)
		}
		settingsCollection, err := f.app.FindCollectionByNameOrId("settings")
		if err != nil {
			t.Fatal(err)
		}
		settings := core.NewRecord(settingsCollection)
		settings.Set("user", user.Id)
		settings.Set("notifications", map[string]any{"trail_mention": map[string]any{"web": true, "email": false}})
		if err := f.app.Save(settings); err != nil {
			t.Fatal(err)
		}
		actor.Set("user", user.Id)
	}
	der, err := x509.MarshalPKIXPublicKey(&f.key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	actor.Set("username", name)
	actor.Set("preferred_username", name)
	actor.Set("iri", origin+"/api/v1/activitypub/user/"+name)
	actor.Set("inbox", actor.GetString("iri")+"/inbox")
	actor.Set("domain", strings.TrimPrefix(origin, "https://"))
	actor.Set("is_local", local)
	actor.Set("public_key", string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})))
	if err := f.app.Save(actor); err != nil {
		t.Fatal(err)
	}
	return actor
}

func (f *trailMentionInboxFixture) body(t *testing.T, author, mentioned *core.Record, iri string) []byte {
	t.Helper()
	mention := pub.MentionNew(pub.IRI(mentioned.GetString("iri")))
	mention.Href = pub.IRI(mentioned.GetString("iri"))
	object := &pub.Object{ID: pub.IRI(iri), Type: pub.NoteType,
		Name:         pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Mentioned trail")),
		Content:      pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Trail description")),
		AttributedTo: pub.IRI(author.GetString("iri")), Location: &pub.Place{Type: pub.PlaceType}, Tag: pub.ItemCollection{mention}}
	activity := pub.CreateNew(pub.IRI(author.GetString("iri")+"/create"), object)
	activity.Actor = pub.IRI(author.GetString("iri"))
	body, err := json.Marshal(activity)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

type trailMentionInboxResult struct {
	err    error
	status int
}

func (f *trailMentionInboxFixture) process(author, recipient *core.Record, body []byte) trailMentionInboxResult {
	path := "/api/v1/activitypub/user/" + recipient.GetString("preferred_username") + "/inbox"
	request := httptest.NewRequest(http.MethodPost, "https://example.com"+path, strings.NewReader(string(body)))
	request.Header.Set("X-Internal-Secret", "mention-inbox-secret")
	request.Header.Set("X-Forwarded-Path", path)
	request.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	signer, _, err := httpsig.NewSigner([]httpsig.Algorithm{httpsig.RSA_SHA256}, httpsig.DigestSha256, []string{"(request-target)", "date", "digest"}, httpsig.Signature, 60)
	if err != nil {
		return trailMentionInboxResult{err: err}
	}
	if err := signer.SignRequest(f.key, author.GetString("iri")+"#main-key", request, body); err != nil {
		return trailMentionInboxResult{err: err}
	}
	response := httptest.NewRecorder()
	event := &core.RequestEvent{App: f.app}
	event.Request, event.Response = request, response
	err = routes.ActivitypubActivityProcess(event)
	return trailMentionInboxResult{err: err, status: response.Code}
}

func (f *trailMentionInboxFixture) assertMention(t *testing.T, author, mentioned *core.Record, iri string) {
	t.Helper()
	trail, err := f.app.FindFirstRecordByData("trails", "iri", iri)
	if err != nil {
		t.Fatal(err)
	}
	records, err := f.app.FindAllRecords("notifications")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("mention notifications=%d, want 1", len(records))
	}
	var metadata map[string]string
	if err := records[0].UnmarshalJSONField("metadata", &metadata); err != nil {
		t.Fatal(err)
	}
	if records[0].GetString("type") != "trail_mention" || records[0].GetString("recipient") != mentioned.Id || records[0].GetString("author") != author.Id || metadata["id"] != trail.Id {
		t.Fatalf("unexpected mention notification: %v", records[0])
	}
}

func TestRemoteTrailMentionSurvivesConcurrentInboxCreates(t *testing.T) {
	f := newTrailMentionInboxFixture(t)
	author := f.actor(t, "remoteauthor", false)
	mentioned := f.actor(t, "mentioned", true)
	follower := f.actor(t, "follower", true)
	follows, err := f.app.FindCollectionByNameOrId("follows")
	if err != nil {
		t.Fatal(err)
	}
	for _, recipient := range []*core.Record{mentioned, follower} {
		follow := core.NewRecord(follows)
		follow.Set("follower", recipient.Id)
		follow.Set("followee", author.Id)
		follow.Set("status", "accepted")
		if err := f.app.Save(follow); err != nil {
			t.Fatal(err)
		}
	}
	iri := "https://remote.example/api/v1/trail/present"
	body := f.body(t, author, mentioned, iri)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	firstReady, winnerSaved := make(chan struct{}), make(chan struct{})
	var attempts atomic.Int32
	type createResult struct {
		attempt int32
		err     error
	}
	creates := make(chan createResult, 2)
	f.app.OnRecordCreate("trails").BindFunc(func(e *core.RecordEvent) error {
		attempt := attempts.Add(1)
		if attempt == 1 {
			close(firstReady)
			select {
			case <-winnerSaved:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		err := e.Next()
		creates <- createResult{attempt: attempt, err: err}
		if attempt == 2 {
			close(winnerSaved)
		}
		return err
	})
	results := make(chan trailMentionInboxResult, 2)
	go func() { results <- f.process(author, mentioned, body) }()
	select {
	case <-firstReady:
	case <-ctx.Done():
		t.Fatal("mentioned inbox did not reach the create barrier")
	}
	go func() { results <- f.process(author, follower, body) }()
	for range 2 {
		select {
		case result := <-results:
			if result.err != nil || result.status != http.StatusOK {
				t.Fatalf("signed inbox was not acknowledged: status=%d err=%v", result.status, result.err)
			}
		case <-ctx.Done():
			t.Fatal("concurrent inbox processing did not finish")
		}
	}
	for range 2 {
		result := <-creates
		if result.attempt == 1 && (result.err == nil || !strings.Contains(strings.ToLower(result.err.Error()), "unique")) {
			t.Fatalf("mentioned inbox must lose the unique IRI insert, got %v", result.err)
		}
		if result.attempt == 2 && result.err != nil {
			t.Fatalf("other inbox did not store the trail: %v", result.err)
		}
	}
	if count, err := f.app.CountRecords("trails"); err != nil || count != 1 {
		t.Fatalf("stored trails=%d, want 1: %v", count, err)
	}
	f.assertMention(t, author, mentioned, iri)
}

func TestLocalTrailMentionMatchesInboxWithAnEarlierRecipientRecord(t *testing.T) {
	f := newTrailMentionInboxFixture(t)
	author := f.actor(t, "author", true)
	older := f.actor(t, "older", false)
	older.Set("inbox", "https://example.com/api/v1/activitypub/user/mentioned/inbox")
	if err := f.app.Save(older); err != nil {
		t.Fatal(err)
	}
	mentioned := f.actor(t, "mentioned", true)
	recipient, err := f.app.FindFirstRecordByData("activitypub_actors", "inbox", mentioned.GetString("inbox"))
	if err != nil || recipient.Id != older.Id {
		t.Fatalf("fixture did not resolve the earlier recipient: %v %v", recipient, err)
	}
	trails, err := f.app.FindCollectionByNameOrId("trails")
	if err != nil {
		t.Fatal(err)
	}
	trail := core.NewRecord(trails)
	trail.Set("iri", "https://example.com/api/v1/trail/present")
	trail.Set("name", "Local mentioned trail")
	trail.Set("author", author.Id)
	trail.Set("public", true)
	if err := f.app.Save(trail); err != nil {
		t.Fatal(err)
	}
	result := f.process(author, mentioned, f.body(t, author, mentioned, trail.GetString("iri")))
	if result.err != nil || result.status != http.StatusOK {
		t.Fatalf("local signed inbox was not acknowledged: status=%d err=%v", result.status, result.err)
	}
	f.assertMention(t, author, mentioned, trail.GetString("iri"))
}
