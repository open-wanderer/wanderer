package main

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
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"pocketbase/federation"
	"pocketbase/routes"

	pub "github.com/go-ap/activitypub"
	"github.com/go-fed/httpsig"
	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/pocketbase/pocketbase/tools/security"
)

const legacyDeleteEncryptionKey = "0123456789abcdef0123456789abcdef"

// legacyOrigin answers the signed actor fetch and the unsigned confirmation
// request of the actor's origin. Remote self-deletions deliver nothing.
type legacyOrigin struct {
	mu                     sync.Mutex
	fetch, confirm         string
	document               []byte
	onConfirm              func()
	fetches, confirmations int
}

func (o *legacyOrigin) set(fetch, confirm string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.fetch, o.confirm = fetch, confirm
}

func (o *legacyOrigin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	o.mu.Lock()
	mode, document := o.fetch, o.document
	if r.Header.Get("Accept") == "application/activity+json" {
		mode = o.confirm
		o.confirmations++
		if o.onConfirm != nil {
			o.onConfirm()
		}
	} else {
		o.fetches++
	}
	o.mu.Unlock()
	switch mode {
	case "200":
		w.Header().Set("Content-Type", "application/activity+json")
		w.Write(document)
	case "404":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"code":404,"message":"The requested resource wasn't found.","data":{}}`))
	case "410":
		w.WriteHeader(http.StatusGone)
	case "500":
		w.WriteHeader(http.StatusInternalServerError)
	case "503":
		w.WriteHeader(http.StatusServiceUnavailable)
	case "timeout":
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
		w.WriteHeader(http.StatusNotFound)
	case "redirect-host":
		w.Header().Set("Location", "https://other.example"+r.URL.Path)
		w.WriteHeader(http.StatusFound)
	case "redirect-path":
		if r.URL.Path == "/moved" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Location", "https://"+r.Host+"/moved")
		w.WriteHeader(http.StatusFound)
	case "network":
		if conn, _, err := w.(http.Hijacker).Hijack(); err == nil {
			conn.Close()
		}
	default:
		w.WriteHeader(http.StatusTeapot)
	}
}

type legacySearch struct {
	mu      sync.Mutex
	deletes []string
}

func (s *legacySearch) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodGet {
		w.Write([]byte(`{"uid":1,"status":"succeeded"}`))
		return
	}
	if r.Method == http.MethodDelete {
		s.mu.Lock()
		s.deletes = append(s.deletes, r.URL.Path)
		s.mu.Unlock()
	}
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"taskUid":1,"status":"enqueued"}`))
}

type legacyWorld struct {
	app            *core.BaseApp
	origin         *legacyOrigin
	search         *legacySearch
	aliceKey       *rsa.PrivateKey
	alice, bob     *core.Record
	labels         []string
	records        map[string]*core.Record
	otherKey       *rsa.PrivateKey
	carol          *core.Record
	carolKey       *rsa.PrivateKey
	aliceIRI       string
	aliceCopyTrail string
}

type legacyWorldOptions struct {
	iri          string // alice's actor IRI
	current      bool   // alice still has a current key bound to her origin
	domain       string // stored domain, defaults to remote.example
	alternateKey bool   // the legacy key belongs to another account
	ownRecord    bool   // a record on this instance's own path names alice as author
	newKey       bool   // the origin's document carries a different key
}

func legacyKeyPair(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return key, string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}

func legacyClient(server *httptest.Server, timeout time.Duration) *http.Client {
	address := server.Listener.Addr().String()
	dialer := &net.Dialer{Timeout: time.Second}
	return &http.Client{Timeout: timeout, Transport: &http.Transport{
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, address)
		},
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}
}

func newLegacyWorld(t *testing.T, options legacyWorldOptions) *legacyWorld {
	t.Helper()
	app := newRealSchemaApp(t)
	t.Setenv("POCKETBASE_PROXY_SECRET", "legacy-delete-secret")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", legacyDeleteEncryptionKey)
	w := &legacyWorld{app: app, origin: &legacyOrigin{fetch: "404", confirm: "404"}, search: &legacySearch{}, records: map[string]*core.Record{}}
	server := httptest.NewTLSServer(w.origin)
	t.Cleanup(server.Close)
	t.Cleanup(federation.SetActorFetchClientForTesting(func() *http.Client { return legacyClient(server, 2*time.Second) }))
	t.Cleanup(routes.SetRemoteSyncClientForTesting(func() *http.Client { return legacyClient(server, 10*time.Second) }))

	save := func(label, collection string, values map[string]any) *core.Record {
		t.Helper()
		record := core.NewRecord(mustCollection(t, app, collection))
		record.Load(values)
		if err := app.SaveNoValidate(record); err != nil {
			t.Fatalf("save %s: %v", label, err)
		}
		if label != "" {
			w.labels = append(w.labels, label)
			w.records[label] = record
		}
		return record
	}

	bobKey, bobPEM := legacyKeyPair(t)
	encrypted, err := security.Encrypt(x509.MarshalPKCS1PrivateKey(bobKey), legacyDeleteEncryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	user := save("", "users", map[string]any{"email": "bob@example.com", "username": "bob"})
	w.bob = save("bob", "activitypub_actors", map[string]any{"user": user.Id, "username": "Bob", "preferred_username": "bob", "is_local": true,
		"domain": "example.com", "iri": "https://example.com/api/v1/activitypub/user/bob", "inbox": "https://example.com/api/v1/activitypub/user/bob/inbox",
		"public_key": bobPEM, "private_key": encrypted})

	var alicePEM, otherPEM string
	w.aliceKey, alicePEM = legacyKeyPair(t)
	w.otherKey, otherPEM = legacyKeyPair(t)
	w.aliceIRI = options.iri
	if w.aliceIRI == "" {
		w.aliceIRI = "https://remote.example/api/v1/activitypub/user/alice"
	}
	domain := options.domain
	if domain == "" {
		domain = "remote.example"
	}
	alice := map[string]any{"username": "Alice", "preferred_username": "alice", "domain": domain, "iri": w.aliceIRI,
		"inbox": w.aliceIRI + "/inbox", "followers": w.aliceIRI + "/followers", "legacy_public_key": alicePEM}
	if options.alternateKey {
		alice["legacy_public_key"] = otherPEM
	}
	if options.current {
		alice["public_key"], alice["last_fetched"] = alicePEM, time.Now()
		if domain == "remote.example" {
			delete(alice, "legacy_public_key")
		}
	}
	w.alice = save("alice", "activitypub_actors", alice)
	documentKey := alicePEM
	if options.newKey {
		_, documentKey = legacyKeyPair(t)
	}
	w.origin.document, err = json.Marshal(map[string]any{
		"@context": []string{"https://www.w3.org/ns/activitystreams", "https://w3id.org/security/v1"},
		"id":       w.aliceIRI, "type": "Person", "preferredUsername": "alice", "name": "Alice",
		"inbox": w.aliceIRI + "/inbox", "outbox": w.aliceIRI + "/outbox", "followers": w.aliceIRI + "/followers", "following": w.aliceIRI + "/following",
		"publicKey": map[string]any{"id": w.aliceIRI + "#main-key", "owner": w.aliceIRI, "publicKeyPem": documentKey},
	})
	if err != nil {
		t.Fatal(err)
	}
	var carolPEM string
	w.carolKey, carolPEM = legacyKeyPair(t)
	w.carol = save("carol", "activitypub_actors", map[string]any{"username": "Carol", "preferred_username": "carol", "domain": "remote.example",
		"iri": "https://remote.example/api/v1/activitypub/user/carol", "inbox": "https://remote.example/api/v1/activitypub/user/carol/inbox", "legacy_public_key": carolPEM})

	bobTrail := save("bobTrail", "trails", map[string]any{"author": w.bob.Id, "name": "Local trail", "public": true})
	bobTrail.Set("iri", "https://example.com/api/v1/trail/"+bobTrail.Id)
	if err := app.SaveNoValidate(bobTrail); err != nil {
		t.Fatal(err)
	}
	w.aliceCopyTrail = "https://remote.example/api/v1/trail/remotetrail0001"
	aliceTrail := save("aliceTrail", "trails", map[string]any{"author": w.alice.Id, "name": "Remote copy", "public": true, "iri": w.aliceCopyTrail})
	save("aliceList", "lists", map[string]any{"author": w.alice.Id, "name": "Remote list", "public": true, "iri": "https://remote.example/api/v1/list/remotelist00001"})
	save("aliceComment", "comments", map[string]any{"author": w.alice.Id, "trail": bobTrail.Id, "text": "Remote comment", "iri": "https://remote.example/api/v1/comment/remotecomment01"})
	save("aliceLike", "trail_like", map[string]any{"actor": w.alice.Id, "trail": bobTrail.Id})
	save("bobFollowsAlice", "follows", map[string]any{"follower": w.bob.Id, "followee": w.alice.Id, "status": "accepted"})
	for _, entry := range []struct{ label, collection, kind string }{{"bobComment", "comments", "comment"}, {"bobSummitLog", "summit_logs", "summit-log"}} {
		record := save(entry.label, entry.collection, map[string]any{"author": w.bob.Id, "trail": aliceTrail.Id, "text": "Local entry", "date": "2026-10-01 10:00:00.000Z"})
		record.Set("iri", "https://example.com/api/v1/"+entry.kind+"/"+record.Id)
		if err := app.SaveNoValidate(record); err != nil {
			t.Fatal(err)
		}
	}
	save("bobLike", "trail_like", map[string]any{"actor": w.bob.Id, "trail": aliceTrail.Id})
	save("bobList", "lists", map[string]any{"author": w.bob.Id, "name": "Local list", "trails": []string{aliceTrail.Id}})
	if options.ownRecord {
		own := save("ownTrail", "trails", map[string]any{"author": w.alice.Id, "name": "Own trail", "public": true})
		own.Set("iri", "https://example.com/api/v1/trail/"+own.Id)
		if err := app.SaveNoValidate(own); err != nil {
			t.Fatal(err)
		}
	}

	searchServer := httptest.NewServer(w.search)
	t.Cleanup(searchServer.Close)
	setupEventHandlers(&pocketbase.PocketBase{App: app}, meilisearch.New(searchServer.URL))
	return w
}

func mustCollection(t *testing.T, app core.App, name string) *core.Collection {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId(name)
	if err != nil {
		t.Fatal(err)
	}
	return collection
}

// deliver posts an activity to bob's inbox the way PostActivity signs it.
func (w *legacyWorld) deliver(t *testing.T, activity pub.Activity, key *rsa.PrivateKey, keyID string) (int, string, error) {
	t.Helper()
	body, err := json.Marshal(activity)
	if err != nil {
		t.Fatal(err)
	}
	path := "/api/v1/activitypub/user/bob/inbox"
	request := httptest.NewRequest(http.MethodPost, "https://example.com"+path, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/activity+json")
	request.Header.Set("Date", strings.ReplaceAll(time.Now().UTC().Format(time.RFC1123), "UTC", "GMT"))
	request.Header.Set("Host", "example.com")
	if key != nil {
		signer, _, err := httpsig.NewSigner([]httpsig.Algorithm{httpsig.RSA_SHA256}, httpsig.DigestSha256, []string{"(request-target)", "Date", "Digest", "Content-Type", "Host"}, httpsig.Signature, 60)
		if err != nil {
			t.Fatal(err)
		}
		if err := signer.SignRequest(key, keyID, request, body); err != nil {
			t.Fatal(err)
		}
	}
	request.Header.Set("X-Internal-Secret", "legacy-delete-secret")
	request.Header.Set("X-Forwarded-Path", path)
	response := httptest.NewRecorder()
	event := &core.RequestEvent{App: w.app}
	event.Request, event.Response = request, response
	err = routes.ActivitypubActivityProcess(event)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if waitErr := federation.WaitForDeliveries(ctx); waitErr != nil {
		t.Fatal(waitErr)
	}
	var apiErr *router.ApiError
	if errors.As(err, &apiErr) {
		return apiErr.Status, "", err
	}
	if err != nil {
		return -1, "", err
	}
	return response.Code, strings.TrimSpace(response.Body.String()), nil
}

func (w *legacyWorld) selfDelete() pub.Activity {
	activity := pub.Activity{Type: pub.DeleteType, ID: pub.IRI("https://remote.example/api/v1/activitypub/activity/delete0000001"),
		Actor: pub.IRI(w.aliceIRI), Object: pub.IRI(w.aliceIRI), To: pub.ItemCollection{pub.PublicNS}}
	return activity
}

// gone lists the labelled records that no longer exist.
func (w *legacyWorld) gone() []string {
	var gone []string
	for _, label := range w.labels {
		record := w.records[label]
		if _, err := w.app.FindRecordById(record.Collection(), record.Id); err != nil {
			gone = append(gone, label)
		}
	}
	return gone
}

func (w *legacyWorld) searchDeletes() []string {
	w.search.mu.Lock()
	defer w.search.mu.Unlock()
	deletes := slices.Clone(w.search.deletes)
	for i, path := range deletes {
		for label, record := range w.records {
			path = strings.ReplaceAll(path, record.Id, "<"+label+">")
		}
		deletes[i] = path
	}
	slices.Sort(deletes)
	return deletes
}

func (w *legacyWorld) aliceKeys(t *testing.T) (current, legacy string) {
	t.Helper()
	stored, err := w.app.FindRecordById("activitypub_actors", w.alice.Id)
	if err != nil {
		return "<deleted>", "<deleted>"
	}
	return stored.GetString("public_key"), stored.GetString("legacy_public_key")
}

var legacyAccountGone = []string{"alice", "aliceTrail", "aliceList", "aliceComment", "aliceLike", "bobFollowsAlice", "bobComment", "bobSummitLog", "bobLike"}

func TestLegacyKeyAuthenticatesConfirmedSelfDeletion(t *testing.T) {
	// The signed self-deletion of an actor with a current key is the
	// reference (dev applies the same cascade); the legacy key must remove
	// exactly the same.
	reference := newLegacyWorld(t, legacyWorldOptions{current: true})
	status, referenceBody, err := reference.deliver(t, reference.selfDelete(), reference.aliceKey, reference.aliceIRI+"#main-key")
	if status != http.StatusOK || err != nil || !slices.Equal(reference.gone(), legacyAccountGone) {
		t.Fatalf("reference self-deletion: status=%d err=%v gone=%v", status, err, reference.gone())
	}
	referenceSearch := reference.searchDeletes()
	if !slices.Contains(referenceSearch, "/indexes/actors/documents/<alice>") {
		t.Fatalf("reference self-deletion kept the actor in search: %v", referenceSearch)
	}
	for _, origin := range []string{"404", "410"} {
		t.Run(origin, func(t *testing.T) {
			w := newLegacyWorld(t, legacyWorldOptions{})
			w.origin.set(origin, origin)
			status, body, err := w.deliver(t, w.selfDelete(), w.aliceKey, w.aliceIRI+"#main-key")
			if status != http.StatusOK || err != nil || body != referenceBody {
				t.Fatalf("status=%d err=%v body=%s, want %s", status, err, body, referenceBody)
			}
			if gone := w.gone(); !slices.Equal(gone, legacyAccountGone) {
				t.Fatalf("cascade differs from the signed self-deletion: %v", gone)
			}
			if search := w.searchDeletes(); !slices.Equal(search, referenceSearch) {
				t.Fatalf("search cleanup %v, want %v", search, referenceSearch)
			}
		})
	}
}

func TestLegacyKeyRefusesUnconfirmedSelfDeletion(t *testing.T) {
	type sign struct {
		key   func(*legacyWorld) *rsa.PrivateKey
		keyID func(*legacyWorld) string
	}
	alice := sign{func(w *legacyWorld) *rsa.PrivateKey { return w.aliceKey }, func(w *legacyWorld) string { return w.aliceIRI + "#main-key" }}
	for _, tc := range []struct {
		name           string
		options        legacyWorldOptions
		fetch, confirm string
		sign           sign
		activity       func(*legacyWorld) pub.Activity
		legacyKept     bool
	}{
		{name: "domain mismatch", options: legacyWorldOptions{domain: "other.example"}, fetch: "404", confirm: "404", sign: alice, legacyKept: true},
		{name: "unsigned", fetch: "404", confirm: "404", sign: sign{func(*legacyWorld) *rsa.PrivateKey { return nil }, alice.keyID}, legacyKept: true},
		{name: "other key", fetch: "404", confirm: "404", sign: sign{func(w *legacyWorld) *rsa.PrivateKey { return w.otherKey }, alice.keyID}, legacyKept: true},
		{name: "another actor of the origin", fetch: "404", confirm: "404", sign: sign{func(w *legacyWorld) *rsa.PrivateKey { return w.carolKey }, func(w *legacyWorld) string { return w.carol.GetString("iri") + "#main-key" }}, legacyKept: true},
		{name: "legacy key under another key id", fetch: "404", confirm: "404", sign: sign{alice.key, func(w *legacyWorld) string { return w.carol.GetString("iri") + "#main-key" }}, legacyKept: true},
		{name: "key id on another host", fetch: "404", confirm: "404", sign: sign{alice.key, func(*legacyWorld) string { return "https://other.example/api/v1/activitypub/user/alice#main-key" }}, legacyKept: true},
		{name: "account exists with a new key", options: legacyWorldOptions{newKey: true}, fetch: "200", confirm: "200", sign: alice},
		{name: "previous key from another account, account exists", options: legacyWorldOptions{alternateKey: true}, fetch: "200", confirm: "200", sign: sign{func(w *legacyWorld) *rsa.PrivateKey { return w.otherKey }, alice.keyID}},
		{name: "origin 500", fetch: "500", confirm: "500", sign: alice, legacyKept: true},
		{name: "origin 503", fetch: "503", confirm: "503", sign: alice, legacyKept: true},
		{name: "origin timeout", fetch: "timeout", confirm: "timeout", sign: alice, legacyKept: true},
		{name: "redirect to another host", fetch: "redirect-host", confirm: "redirect-host", sign: alice, legacyKept: true},
		{name: "redirect within the host", fetch: "redirect-path", confirm: "redirect-path", sign: alice, legacyKept: true},
		{name: "network error", fetch: "network", confirm: "network", sign: alice, legacyKept: true},
		{name: "fetch 404, confirmation 503", fetch: "404", confirm: "503", sign: alice, legacyKept: true},
		{name: "fetch 503, confirmation 404", fetch: "503", confirm: "404", sign: alice, legacyKept: true},
		{name: "delete of a copy", fetch: "404", confirm: "404", sign: alice, legacyKept: true, activity: func(w *legacyWorld) pub.Activity {
			a := w.selfDelete()
			a.Object = pub.IRI(w.aliceCopyTrail)
			return a
		}},
		{name: "delete naming another actor", fetch: "404", confirm: "404", sign: alice, legacyKept: true, activity: func(w *legacyWorld) pub.Activity {
			a := w.selfDelete()
			a.Object = pub.IRI(w.carol.GetString("iri"))
			return a
		}},
		{name: "like", fetch: "404", confirm: "404", sign: alice, legacyKept: true, activity: func(w *legacyWorld) pub.Activity {
			return pub.Activity{Type: pub.LikeType, ID: pub.IRI(w.aliceIRI + "/like/1"), Actor: pub.IRI(w.aliceIRI), Object: pub.IRI(w.records["bobTrail"].GetString("iri"))}
		}},
		{name: "update of the actor itself", fetch: "404", confirm: "404", sign: alice, legacyKept: true, activity: func(w *legacyWorld) pub.Activity {
			a := w.selfDelete()
			a.Type = pub.UpdateType
			return a
		}},
		{name: "unconfirmed current key", options: legacyWorldOptions{current: true, domain: "other.example"}, fetch: "404", confirm: "404", sign: alice, legacyKept: true},
		{name: "other software", options: legacyWorldOptions{iri: "https://remote.example/users/alice"}, fetch: "404", confirm: "404", sign: alice, legacyKept: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newLegacyWorld(t, tc.options)
			w.origin.set(tc.fetch, tc.confirm)
			_, legacyBefore := w.aliceKeys(t)
			activity := w.selfDelete()
			if tc.activity != nil {
				activity = tc.activity(w)
			}
			status, _, err := w.deliver(t, activity, tc.sign.key(w), tc.sign.keyID(w))
			if status != http.StatusUnauthorized {
				t.Fatalf("status=%d err=%v", status, err)
			}
			if gone := w.gone(); len(gone) != 0 {
				t.Fatalf("refused self-deletion removed %v", gone)
			}
			current, legacy := w.aliceKeys(t)
			if tc.legacyKept != (legacy == legacyBefore) || (!tc.legacyKept && (legacy != "" || current == "")) {
				t.Fatalf("keys after refusal: current=%t legacy=%t", current != "", legacy != "")
			}
		})
	}
}

func TestLegacyKeyDeletionKeepsOwnRecords(t *testing.T) {
	w := newLegacyWorld(t, legacyWorldOptions{ownRecord: true})
	status, _, err := w.deliver(t, w.selfDelete(), w.aliceKey, w.aliceIRI+"#main-key")
	if status != http.StatusOK || err != nil {
		t.Fatalf("status=%d err=%v", status, err)
	}
	if gone := w.gone(); len(gone) != 0 {
		t.Fatalf("cleanup removed %v despite an own record", gone)
	}
}

func TestLegacyKeyRecheckedBeforeDeletion(t *testing.T) {
	w := newLegacyWorld(t, legacyWorldOptions{})
	// Another request confirms the actor at its origin while this one
	// waits for the confirmation.
	w.origin.onConfirm = func() {
		if _, err := w.app.DB().NewQuery("UPDATE activitypub_actors SET public_key='confirmed', legacy_public_key='' WHERE id={:id}").Bind(map[string]any{"id": w.alice.Id}).Execute(); err != nil {
			t.Error(err)
		}
	}
	status, _, err := w.deliver(t, w.selfDelete(), w.aliceKey, w.aliceIRI+"#main-key")
	if status != http.StatusUnauthorized || len(w.gone()) != 0 {
		t.Fatalf("status=%d err=%v gone=%v", status, err, w.gone())
	}
}

func TestLegacyKeyDomainRecheckedBeforeDeletion(t *testing.T) {
	w := newLegacyWorld(t, legacyWorldOptions{})
	w.origin.onConfirm = func() {
		if _, err := w.app.DB().NewQuery("UPDATE activitypub_actors SET domain='other.example' WHERE id={:id}").Bind(map[string]any{"id": w.alice.Id}).Execute(); err != nil {
			t.Error(err)
		}
	}
	status, _, err := w.deliver(t, w.selfDelete(), w.aliceKey, w.aliceIRI+"#main-key")
	if status != http.StatusUnauthorized || len(w.gone()) != 0 {
		t.Fatalf("status=%d err=%v gone=%v", status, err, w.gone())
	}
}

func TestOtherSoftwareSelfDeletionStillNeedsGone(t *testing.T) {
	w := newLegacyWorld(t, legacyWorldOptions{iri: "https://remote.example/users/alice"})
	w.origin.set("410", "410")
	status, _, err := w.deliver(t, w.selfDelete(), nil, "")
	if status != http.StatusOK || !slices.Equal(w.gone(), legacyAccountGone) {
		t.Fatalf("status=%d err=%v gone=%v", status, err, w.gone())
	}
}

func TestConfirmedActorRefreshDropsLegacyKey(t *testing.T) {
	for _, path := range []string{"inbox", "lookup"} {
		t.Run(path, func(t *testing.T) {
			w := newLegacyWorld(t, legacyWorldOptions{})
			w.origin.set("200", "200")
			if path == "inbox" {
				like := pub.Activity{Type: pub.LikeType, ID: pub.IRI(w.aliceIRI + "/like/1"), Actor: pub.IRI(w.aliceIRI), Object: pub.IRI(w.records["bobTrail"].GetString("iri"))}
				if status, _, err := w.deliver(t, like, w.aliceKey, w.aliceIRI+"#main-key"); status != http.StatusOK {
					t.Fatalf("status=%d err=%v", status, err)
				}
			} else if _, err := federation.GetActorByIRI(w.app, context.WithValue(context.Background(), "actor", "actor:"+w.bob.Id), w.aliceIRI, false); err != nil {
				t.Fatal(err)
			}
			current, legacy := w.aliceKeys(t)
			if current == "" || legacy != "" {
				t.Fatalf("after a confirmed refresh: current=%t legacy=%t", current != "", legacy != "")
			}
			// From now on the legacy key no longer counts.
			w.origin.set("404", "404")
			stored, err := w.app.FindRecordById("activitypub_actors", w.alice.Id)
			if err != nil {
				t.Fatal(err)
			}
			stored.Set("public_key", "")
			if err := w.app.SaveNoValidate(stored); err != nil {
				t.Fatal(err)
			}
			if status, _, err := w.deliver(t, w.selfDelete(), w.aliceKey, w.aliceIRI+"#main-key"); status != http.StatusUnauthorized || len(w.gone()) != 0 {
				t.Fatalf("status=%d err=%v gone=%v", status, err, w.gone())
			}
		})
	}
}
