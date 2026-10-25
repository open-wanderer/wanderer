package routes

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"pocketbase/federation"
	"pocketbase/util"

	pub "github.com/go-ap/activitypub"
	"github.com/go-fed/httpsig"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

type knownKeyOrigin struct {
	mu          sync.Mutex
	documents   map[string]string
	requests    map[string]int
	unavailable map[string]bool
	statuses    map[string]int
	held        map[string]<-chan struct{}
	started     map[string]chan struct{}
}

func (o *knownKeyOrigin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	o.mu.Lock()
	o.requests[r.URL.Path]++
	if o.requests[r.URL.Path] == 1 && o.started[r.URL.Path] != nil {
		close(o.started[r.URL.Path])
	}
	held, unavailable, document, status := o.held[r.URL.Path], o.unavailable[r.URL.Path], o.documents[r.URL.Path], o.statuses[r.URL.Path]
	o.mu.Unlock()
	if held != nil {
		select {
		case <-held:
		case <-r.Context().Done():
			return
		}
	}
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	if unavailable {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	if document != "" {
		w.Header().Set("Content-Type", "application/activity+json")
		w.Write([]byte(document))
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

func (o *knownKeyOrigin) count(path string) int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.requests[path]
}

type knownKeyWorld struct {
	app                          *core.BaseApp
	origin                       *knownKeyOrigin
	client                       *http.Client
	recipient                    *core.Record
	key, previousKey, otherKey   *rsa.PrivateKey
	publicKey, previousPublicKey string
	requests                     atomic.Int64
	requestContext               context.Context
}

// Use the application schema, with the retained-key field available both
// before and after the cache-refresh migration enters the commit sequence.
func newKnownKeyWorld(t *testing.T, actorTimeout ...time.Duration) *knownKeyWorld {
	t.Helper()
	t.Setenv("ORIGIN", "https://example.com")
	t.Setenv("POCKETBASE_PROXY_SECRET", "known-key-test-secret")
	// No outbound signing is needed to observe persisted incoming follows.
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "")
	app := newRemoteRealSchemaApp(t)
	t.Setenv("ORIGIN", "https://example.com")
	actors, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	if actors.Fields.GetByName("legacy_public_key") == nil {
		actors.Fields.Add(&core.TextField{Name: "legacy_public_key", Hidden: true})
		if err := app.Save(actors); err != nil {
			t.Fatal(err)
		}
	}
	keyPair := func() (*rsa.PrivateKey, string) {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
		if err != nil {
			t.Fatal(err)
		}
		return key, string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: encoded}))
	}
	key, publicKey := keyPair()
	previousKey, previousPublicKey := keyPair()
	otherKey, _ := keyPair()
	w := &knownKeyWorld{app: app, key: key, previousKey: previousKey, otherKey: otherKey, publicKey: publicKey, previousPublicKey: previousPublicKey,
		origin: &knownKeyOrigin{documents: map[string]string{}, requests: map[string]int{}, unavailable: map[string]bool{}, held: map[string]<-chan struct{}{}, started: map[string]chan struct{}{}}}
	user := w.save(t, "users", map[string]any{"email": "reader@example.com", "username": "reader"})
	w.recipient = w.save(t, "activitypub_actors", map[string]any{"username": "Reader", "preferred_username": "reader", "user": user.Id, "is_local": true, "domain": "example.com", "iri": "https://example.com/api/v1/activitypub/user/reader", "inbox": "https://example.com/api/v1/activitypub/user/reader/inbox", "public_key": publicKey})
	server := httptest.NewTLSServer(w.origin)
	t.Cleanup(server.Close)
	timeout := time.Second
	if len(actorTimeout) > 0 {
		timeout = actorTimeout[0]
	}
	client := &http.Client{Timeout: timeout, Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, DisableKeepAlives: true,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			if err := util.CheckDialRateLimit(ctx, host); err != nil {
				return nil, err
			}
			return (&net.Dialer{Timeout: time.Second}).DialContext(ctx, network, server.Listener.Addr().String())
		}}}
	w.client = client
	t.Cleanup(client.CloseIdleConnections)
	previousLookup := getInboxActorByIRI
	getInboxActorByIRI = func(gotApp core.App, ctx context.Context, iri string, includeFollows bool) (*core.Record, error) {
		contextActor, _ := ctx.Value("actor").(string)
		signingActor, lookupErr := app.FindRecordById("activitypub_actors", strings.TrimPrefix(contextActor, "actor:"))
		if gotApp != app || includeFollows || !strings.HasPrefix(contextActor, "actor:") || lookupErr != nil || !signingActor.GetBool("is_local") {
			return nil, fmt.Errorf("incorrect actor refresh context")
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, iri, nil)
		if err != nil {
			return nil, err
		}
		response, err := client.Do(request)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			if response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusGone {
				return nil, federation.ErrActorNotFound
			}
			return nil, fmt.Errorf("actor response: %d", response.StatusCode)
		}
		var document struct {
			ID        string `json:"id"`
			PublicKey string `json:"publicKey"`
		}
		if err := json.NewDecoder(response.Body).Decode(&document); err != nil {
			return nil, err
		}
		if document.ID != iri {
			return nil, fmt.Errorf("different actor document")
		}
		actor, err := app.FindFirstRecordByData("activitypub_actors", "iri", iri)
		if err != nil {
			return nil, err
		}
		actor.Set("public_key", document.PublicKey)
		actor.Set("legacy_public_key", "")
		parsed, _ := url.Parse(iri)
		actor.Set("domain", parsed.Host)
		actor.Set("last_fetched", time.Now())
		return actor, app.SaveNoValidate(actor)
	}
	t.Cleanup(func() { getInboxActorByIRI = previousLookup })
	previousBudget := util.ActivityPubRateLimiter
	util.ActivityPubRateLimiter = util.NewRateLimiter(30, time.Minute)
	t.Cleanup(func() { util.ActivityPubRateLimiter = previousBudget })
	return w
}

func (w *knownKeyWorld) save(t *testing.T, collection string, values map[string]any) *core.Record {
	t.Helper()
	schema, err := w.app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(schema)
	record.Load(values)
	if err := w.app.SaveNoValidate(record); err != nil {
		t.Fatal(err)
	}
	return record
}

func (w *knownKeyWorld) sender(t *testing.T, name string, domain string, retained bool) *core.Record {
	t.Helper()
	iri := "https://remote.example/api/v1/activitypub/user/" + name
	values := map[string]any{"iri": iri, "inbox": iri + "/inbox", "domain": domain, "username": name, "preferred_username": name}
	if retained {
		values["legacy_public_key"] = w.previousPublicKey
	}
	actor := w.save(t, "activitypub_actors", values)
	document, err := json.Marshal(map[string]string{"id": iri, "publicKey": w.publicKey})
	if err != nil {
		t.Fatal(err)
	}
	w.origin.documents["/api/v1/activitypub/user/"+name] = string(document)
	return actor
}

func (w *knownKeyWorld) deliver(t *testing.T, iri string, key *rsa.PrivateKey) int {
	t.Helper()
	return w.deliverActivity(t, iri, key, w.recipient, pub.FollowType, w.recipient.GetString("iri"), w.requestContext)
}

func (w *knownKeyWorld) deliverActivity(t *testing.T, iri string, key *rsa.PrivateKey, recipient *core.Record, kind pub.ActivityVocabularyType, object string, ctx context.Context) int {
	t.Helper()
	body, err := json.Marshal(pub.Activity{ID: pub.IRI(fmt.Sprintf("https://remote.example/activities/%d", w.requests.Add(1))), Type: kind, Actor: pub.IRI(iri), Object: pub.IRI(object)})
	if err != nil {
		t.Fatal(err)
	}
	path := strings.TrimPrefix(recipient.GetString("inbox"), "https://example.com")
	request := httptest.NewRequest(http.MethodPost, "https://example.com"+path, bytes.NewReader(body))
	if ctx != nil {
		request = request.WithContext(ctx)
	}
	request.Header.Set("X-Internal-Secret", "known-key-test-secret")
	request.Header.Set("X-Forwarded-Path", path)
	request.Header.Set("Content-Type", "application/activity+json")
	request.Header.Set("Date", strings.ReplaceAll(time.Now().UTC().Format(time.RFC1123), "UTC", "GMT"))
	request.Header.Set("Host", "example.com")
	if key != nil {
		signer, _, err := httpsig.NewSigner([]httpsig.Algorithm{httpsig.RSA_SHA256}, httpsig.DigestSha256, []string{"(request-target)", "Date", "Digest", "Content-Type", "Host"}, httpsig.Signature, 60)
		if err != nil {
			t.Fatal(err)
		}
		if err := signer.SignRequest(key, iri+"#main-key", request, body); err != nil {
			t.Fatal(err)
		}
	}
	event := &core.RequestEvent{App: w.app}
	event.Request = request
	event.Response = httptest.NewRecorder()
	err = ActivitypubActivityProcess(event)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := federation.WaitForDeliveries(ctx); err != nil {
		t.Fatal(err)
	}
	var apiErr *router.ApiError
	if errors.As(err, &apiErr) {
		return apiErr.Status
	}
	if err != nil {
		t.Fatal(err)
	}
	return event.Response.(*httptest.ResponseRecorder).Code
}

func TestInboxKnownKeyReloadsWithRecipientBudgetInUse(t *testing.T) {
	for _, budgetUse := range []string{"lookups", "deliveries"} {
		t.Run(budgetUse, func(t *testing.T) {
			w := newKnownKeyWorld(t)
			if budgetUse == "lookups" {
				ctx, err := util.GetSafeActorContext(nil, w.recipient)
				if err != nil {
					t.Fatal(err)
				}
				for i := 0; i < 30; i++ {
					if err := util.CheckDialRateLimit(ctx, "remote.example"); err != nil {
						t.Fatal(err)
					}
				}
				if err := util.CheckDialRateLimit(ctx, "remote.example"); !errors.Is(err, util.ErrRateLimited) {
					t.Fatalf("lookup budget: %v", err)
				}
			} else {
				for i := 0; i < 31; i++ {
					iri := fmt.Sprintf("https://remote.example/api/v1/activitypub/user/unregistered%02d", i)
					if status := w.deliver(t, iri, nil); status != http.StatusUnauthorized {
						t.Fatalf("delivery %d: %d", i, status)
					}
				}
				if got := w.origin.count("/api/v1/activitypub/user/unregistered30"); got != 0 {
					t.Fatalf("last unregistered actor fetched %d times", got)
				}
			}
			for i := 0; i < 40; i++ {
				name := fmt.Sprintf("sender%02d", i)
				actor := w.sender(t, name, "remote.example", true)
				if status := w.deliver(t, actor.GetString("iri"), w.key); status != http.StatusOK {
					t.Fatalf("known sender %d: %d", i, status)
				}
				if got := w.origin.count("/api/v1/activitypub/user/" + name); got != 1 {
					t.Fatalf("sender %d fetched %d times", i, got)
				}
			}
			follows, err := w.app.FindAllRecords("follows")
			if err != nil {
				t.Fatal(err)
			}
			if len(follows) != 40 {
				t.Fatalf("accepted follows: %d", len(follows))
			}
		})
	}
}

func TestInboxKnownKeyReloadsOnceWhenOriginUnavailable(t *testing.T) {
	w := newKnownKeyWorld(t)
	actor := w.sender(t, "sender", "remote.example", true)
	path := "/api/v1/activitypub/user/sender"
	w.origin.unavailable[path] = true
	// The recipient has its entire budget available. A refused actor claim
	// must not fall back to that budget.
	for i := 0; i < 20; i++ {
		if status := w.deliver(t, actor.GetString("iri"), w.previousKey); status != http.StatusUnauthorized {
			t.Fatalf("delivery %d: %d", i, status)
		}
		if i == 0 {
			// Domain casing may change independently of the actor's identity.
			actor.Set("domain", "REMOTE.EXAMPLE")
			if err := w.app.SaveNoValidate(actor); err != nil {
				t.Fatal(err)
			}
		}
	}
	if got := w.origin.count(path); got != 1 {
		t.Fatalf("actor reloaded %d times", got)
	}
	stored, err := w.app.FindRecordById("activitypub_actors", actor.Id)
	if err != nil {
		t.Fatal(err)
	}
	if stored.GetString("public_key") != "" || stored.GetString("legacy_public_key") != w.previousPublicKey {
		t.Fatal("unavailable origin changed retained key")
	}
}

func TestInboxKnownKeyReloadUsesConfirmedKey(t *testing.T) {
	for _, keyUse := range []string{"earlier", "different", "confirmed"} {
		t.Run(keyUse, func(t *testing.T) {
			w := newKnownKeyWorld(t)
			actor := w.sender(t, "sender", "remote.example", true)
			key := w.key
			want := http.StatusOK
			if keyUse == "earlier" {
				key = w.previousKey
				want = http.StatusUnauthorized
			}
			if keyUse == "different" {
				key = w.otherKey
				want = http.StatusUnauthorized
			}
			if status := w.deliver(t, actor.GetString("iri"), key); status != want {
				t.Fatalf("signature response: %d", status)
			}
			if got := w.origin.count("/api/v1/activitypub/user/sender"); got != 1 {
				t.Fatalf("actor reloaded %d times", got)
			}
		})
	}
}

func TestInboxOtherActorsKeepRecipientBudget(t *testing.T) {
	for _, scenario := range []string{"no-retained-key", "different-host", "local-record"} {
		t.Run(scenario, func(t *testing.T) {
			w := newKnownKeyWorld(t)
			domain := "remote.example"
			if scenario == "different-host" {
				domain = "other.example"
			}
			actor := w.sender(t, "sender", domain, scenario != "no-retained-key")
			if scenario == "local-record" {
				actor.Set("is_local", true)
				if err := w.app.SaveNoValidate(actor); err != nil {
					t.Fatal(err)
				}
			}
			ctx, err := util.GetSafeActorContext(nil, w.recipient)
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 30; i++ {
				if err := util.CheckDialRateLimit(ctx, "remote.example"); err != nil {
					t.Fatal(err)
				}
			}
			if status := w.deliver(t, actor.GetString("iri"), w.key); status != http.StatusUnauthorized {
				t.Fatalf("delivery: %d", status)
			}
			if got := w.origin.count("/api/v1/activitypub/user/sender"); got != 0 {
				t.Fatalf("actor outside recipient budget fetched %d times", got)
			}
		})
	}
}

func TestKnownKeyReloadRequiresStoredRemoteIdentity(t *testing.T) {
	schema := core.NewBaseCollection("activitypub_actors")
	schema.Fields.Add(&core.TextField{Name: "iri"}, &core.TextField{Name: "domain"}, &core.TextField{Name: "public_key"}, &core.TextField{Name: "legacy_public_key"}, &core.BoolField{Name: "is_local"})
	base := core.NewRecord(schema)
	base.Load(map[string]any{"iri": "https://remote.example/users/sender", "domain": "remote.example", "legacy_public_key": "earlier key"})
	if !actorNeedsKnownKeyReload(base) {
		t.Fatal("stored remote identity not eligible")
	}
	for _, change := range []struct {
		field string
		value any
	}{{"is_local", true}, {"public_key", "confirmed key"}, {"legacy_public_key", ""}, {"domain", "other.example"}, {"domain", ""}, {"iri", ":invalid"}} {
		t.Run(change.field+fmt.Sprint(change.value), func(t *testing.T) {
			record := base.Clone()
			record.Set(change.field, change.value)
			if actorNeedsKnownKeyReload(record) {
				t.Fatalf("record with %s=%v eligible", change.field, change.value)
			}
		})
	}
	if actorNeedsKnownKeyReload(nil) {
		t.Fatal("absent actor eligible")
	}
}

func TestInboxKnownKeyReloadDoesNotClaimCancelledRequest(t *testing.T) {
	w := newKnownKeyWorld(t)
	actor := w.sender(t, "sender", "remote.example", true)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w.requestContext = ctx
	if status := w.deliver(t, actor.GetString("iri"), w.key); status != http.StatusUnauthorized {
		t.Fatalf("cancelled request: %d", status)
	}
	if got := w.origin.count("/api/v1/activitypub/user/sender"); got != 0 {
		t.Fatalf("cancelled request reached origin %d times", got)
	}
	w.requestContext = nil
	if status := w.deliver(t, actor.GetString("iri"), w.key); status != http.StatusOK {
		t.Fatalf("subsequent request: %d", status)
	}
	if got := w.origin.count("/api/v1/activitypub/user/sender"); got != 1 {
		t.Fatalf("origin requests: %d", got)
	}
}

func TestInboxKnownKeyReloadSharesConcurrentRecipients(t *testing.T) {
	for _, offset := range []time.Duration{0, 5 * time.Millisecond, 15 * time.Millisecond} {
		t.Run(offset.String(), func(t *testing.T) {
			w := newKnownKeyWorld(t)
			actor := w.sender(t, "sender", "remote.example", true)
			recipients := []*core.Record{w.recipient}
			for i := 1; i < 5; i++ {
				name := fmt.Sprintf("reader%d", i)
				user := w.save(t, "users", map[string]any{"email": name + "@example.com", "username": name})
				iri := "https://example.com/api/v1/activitypub/user/" + name
				recipients = append(recipients, w.save(t, "activitypub_actors", map[string]any{"username": name, "preferred_username": name, "user": user.Id, "is_local": true, "domain": "example.com", "iri": iri, "inbox": iri + "/inbox", "public_key": w.publicKey}))
			}
			path := "/api/v1/activitypub/user/sender"
			release, started := make(chan struct{}), make(chan struct{})
			w.origin.held[path], w.origin.started[path] = release, started
			var releaseOnce sync.Once
			defer releaseOnce.Do(func() { close(release) })
			responses := make(chan int, len(recipients))
			for i, recipient := range recipients {
				go func(delay time.Duration, recipient *core.Record) {
					time.Sleep(delay)
					responses <- w.deliverActivity(t, actor.GetString("iri"), w.key, recipient, pub.FollowType, recipient.GetString("iri"), nil)
				}(time.Duration(i)*offset, recipient)
			}
			select {
			case <-started:
			case <-time.After(2 * time.Second):
				t.Fatal("actor refresh did not start")
			}
			select {
			case status := <-responses:
				t.Errorf("delivery completed before the actor refresh: %d", status)
				responses <- status
			case <-time.After(100 * time.Millisecond):
			}
			releaseOnce.Do(func() { close(release) })
			for range recipients {
				select {
				case status := <-responses:
					if status != http.StatusOK {
						t.Errorf("recipient delivery: %d", status)
					}
				case <-time.After(3 * time.Second):
					t.Fatal("recipient delivery did not complete")
				}
			}
			if got := w.origin.count(path); got != 1 {
				t.Fatalf("actor refresh requests: %d", got)
			}
			follows, err := w.app.FindAllRecords("follows")
			if err != nil || len(follows) != len(recipients) {
				t.Fatalf("stored follows: %d, error: %v", len(follows), err)
			}
		})
	}
}

func TestInboxKnownKeyReloadWaitRespectsRequestDeadline(t *testing.T) {
	w := newKnownKeyWorld(t)
	actor := w.sender(t, "sender", "remote.example", true)
	path := "/api/v1/activitypub/user/sender"
	release, started := make(chan struct{}), make(chan struct{})
	w.origin.held[path], w.origin.started[path] = release, started
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	first := make(chan int, 1)
	go func() { first <- w.deliver(t, actor.GetString("iri"), w.key) }()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("actor refresh did not start")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	begin := time.Now()
	status := w.deliverActivity(t, actor.GetString("iri"), w.key, w.recipient, pub.FollowType, w.recipient.GetString("iri"), ctx)
	if status != http.StatusUnauthorized || ctx.Err() == nil || time.Since(begin) > 500*time.Millisecond {
		t.Fatalf("waiting delivery: status=%d, context=%v, elapsed=%v", status, ctx.Err(), time.Since(begin))
	}
	select {
	case status := <-first:
		t.Fatalf("shared refresh ended with waiting request: %d", status)
	default:
	}
	releaseOnce.Do(func() { close(release) })
	if status := <-first; status != http.StatusOK {
		t.Fatalf("first delivery: %d", status)
	}
	if status := w.deliver(t, actor.GetString("iri"), w.key); status != http.StatusOK {
		t.Fatalf("later delivery: %d", status)
	}
	if got := w.origin.count(path); got != 1 {
		t.Fatalf("actor refresh requests: %d", got)
	}
}

func TestKnownKeyReloadReadsCurrentActorBeforeClaim(t *testing.T) {
	w := newKnownKeyWorld(t)
	actor := w.sender(t, "sender", "remote.example", true)
	ctx, err := util.GetSafeActorContext(nil, w.recipient)
	if err != nil {
		t.Fatal(err)
	}
	bucket := "known-actor:" + w.app.DataDir() + ":" + actor.Id
	ctx = util.WithRateLimitBucket(ctx, bucket)
	for i := 0; i < 2; i++ {
		stored, err := reloadKnownInboxActor(w.app, ctx, actor)
		if err != nil || stored == nil || stored.GetString("public_key") != w.publicKey || stored.GetString("legacy_public_key") != "" {
			t.Fatalf("refresh %d: record=%v error=%v", i, stored, err)
		}
	}
	if got := w.origin.count("/api/v1/activitypub/user/sender"); got != 1 {
		t.Fatalf("actor refresh requests: %d", got)
	}
}
