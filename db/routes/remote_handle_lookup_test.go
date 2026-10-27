package routes

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/pocketbase/pocketbase/tools/types"
)

// The web links a trail or list through /trail/view/<handle>/<record id>:
// search hits use @user, the profile feed @user@<actor domain>, and pages
// that derive the handle from the object IRI use @user@<IRI host>. A user
// keeps the actor IRI registered under the ORIGIN of the time, so after an
// ORIGIN change (host, port or scheme) the handle and the IRIs of newer
// objects name different origins.

type handleLookupTransport struct {
	mu       sync.Mutex
	status   int
	body     string
	requests []string
}

func (t *handleLookupTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	t.mu.Lock()
	t.requests = append(t.requests, r.URL.Scheme+"://"+r.URL.Host+r.URL.Path)
	t.mu.Unlock()
	body := t.body
	if body == "" {
		body = `{"message":"nope"}`
	}
	return &http.Response{StatusCode: t.status, Header: make(http.Header), Request: r,
		Body: io.NopCloser(strings.NewReader(body))}, nil
}

func (t *handleLookupTransport) seen() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]string(nil), t.requests...)
}

type handleLookupFixture struct {
	app       *pbtests.TestApp
	transport *handleLookupTransport
	actors    *core.Collection
}

func setupHandleLookupTest(t *testing.T, origin string, status int) *handleLookupFixture {
	t.Helper()
	t.Setenv("ORIGIN", origin)
	transport := &handleLookupTransport{status: status}
	originalClient := newRemoteSyncHTTPClient
	newRemoteSyncHTTPClient = func() *http.Client { return &http.Client{Transport: transport} }
	t.Cleanup(func() { newRemoteSyncHTTPClient = originalClient })

	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	drainTestAppStartupLogs(t, app)
	app.Settings().Logs.MaxDays = 0
	save := func(c *core.Collection) {
		if err := app.Save(c); err != nil {
			t.Fatal(err)
		}
	}
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	settings := core.NewBaseCollection("settings")
	settings.Fields.Add(&core.RelationField{Name: "user", CollectionId: users.Id, MaxSelect: 1}, &core.TextField{Name: "privacy"}, &core.TextField{Name: "bio"})
	save(settings)
	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(&core.TextField{Name: "preferred_username"}, &core.TextField{Name: "domain"}, &core.TextField{Name: "iri"}, &core.TextField{Name: "public_key"},
		&core.BoolField{Name: "is_local"}, &core.DateField{Name: "last_fetched"}, &core.RelationField{Name: "user", CollectionId: users.Id, MaxSelect: 1})
	save(actors)
	follows := core.NewBaseCollection("follows")
	follows.Fields.Add(&core.RelationField{Name: "follower", CollectionId: actors.Id, MaxSelect: 1},
		&core.RelationField{Name: "followee", CollectionId: actors.Id, MaxSelect: 1}, &core.TextField{Name: "status"})
	save(follows)
	for _, name := range []string{"trails", "lists"} {
		c := core.NewBaseCollection(name)
		c.ViewRule = types.Pointer("public = true")
		c.Fields.Add(&core.TextField{Name: "iri"}, &core.TextField{Name: "name"}, &core.BoolField{Name: "public"},
			&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
			&core.BoolField{Name: "needs_full_sync"}, &core.BoolField{Name: "full_sync_completed"}, &core.DateField{Name: "updated"})
		save(c)
	}
	return &handleLookupFixture{app: app, transport: transport, actors: actors}
}

func (f *handleLookupFixture) record(t *testing.T, collection string, values map[string]any) *core.Record {
	t.Helper()
	c, err := f.app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(c)
	r.Load(values)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

// localActor registers a user under an earlier ORIGIN: the actor keeps that
// IRI and domain.
func (f *handleLookupFixture) localActor(t *testing.T, name, registeredOrigin, domain string) *core.Record {
	t.Helper()
	users, _ := f.app.FindCollectionByNameOrId("users")
	user := core.NewRecord(users)
	user.SetEmail(name + "@example.com")
	user.SetPassword("Passw0rd!Passw0rd")
	if err := f.app.Save(user); err != nil {
		t.Fatal(err)
	}
	f.record(t, "settings", map[string]any{"user": user.Id})
	return f.record(t, "activitypub_actors", map[string]any{"preferred_username": name, "domain": domain, "is_local": true,
		"iri": registeredOrigin + "/api/v1/activitypub/user/" + name, "user": user.Id, "last_fetched": time.Now()})
}

func (f *handleLookupFixture) remoteActor(t *testing.T, name, host string) *core.Record {
	t.Helper()
	return f.record(t, "activitypub_actors", map[string]any{"preferred_username": name, "domain": host, "is_local": false,
		"iri": "https://" + host + "/api/v1/activitypub/user/" + name, "public_key": "test-key", "last_fetched": time.Now()})
}

// object stores a trail or list whose IRI is <iriOrigin>/api/v1/<kind>/<id>
// with its own record ID unless remoteID is given.
func (f *handleLookupFixture) object(t *testing.T, collection, author, iriOrigin, remoteID string, needsFullSync, stale bool) *core.Record {
	t.Helper()
	updated := time.Now()
	if stale {
		updated = updated.Add(-remoteSyncThreshold - time.Minute)
	}
	r := f.record(t, collection, map[string]any{"name": "object", "public": true, "author": author,
		"full_sync_completed": true, "needs_full_sync": needsFullSync, "updated": updated})
	id := remoteID
	if id == "" {
		id = r.Id
	}
	r.Set("iri", iriOrigin+"/api/v1/"+strings.TrimSuffix(collection, "s")+"/"+id)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
	return r
}

func (f *handleLookupFixture) get(t *testing.T, collection, id, handle string) (int, string) {
	t.Helper()
	// Keep the fixture alive and make stored-state assertions after the refresh.
	defer func() {
		if record, err := f.app.FindRecordById(collection, id); err == nil {
			if err := waitForBackgroundSync(record, 10*time.Second); err != nil {
				t.Fatal(err)
			}
		}
	}()

	handler := RemoteTrailGet
	if collection == "lists" {
		handler = RemoteListGet
	}
	e := &core.RequestEvent{App: f.app}
	e.Request = httptest.NewRequest(http.MethodGet, "/remote/"+strings.TrimSuffix(collection, "s")+"/"+id+"?handle="+strings.ReplaceAll(handle, "@", "%40"), nil)
	e.Request.SetPathValue("id", id)
	response := httptest.NewRecorder()
	e.Response = response
	if err := handler(e); err != nil {
		var apiErr *router.ApiError
		if !errors.As(err, &apiErr) {
			t.Fatal(err)
		}
		return apiErr.Status, ""
	}
	return response.Code, response.Body.String()
}

func TestRemoteHandleLookupServesLocalObjectsCreatedAfterAnOriginChange(t *testing.T) {
	for _, change := range []struct{ name, registered, domain string }{
		{"host", "https://old.example", "old.example"},
		{"scheme", "http://local.example", "local.example"},
		{"port", "https://local.example:8443", "local.example"},
	} {
		for _, collection := range []string{"trails", "lists"} {
			t.Run(change.name+"/"+collection, func(t *testing.T) {
				f := setupHandleLookupTest(t, "https://local.example", http.StatusNotFound)
				alice := f.localActor(t, "alice", change.registered, change.domain)
				before := f.object(t, collection, alice.Id, change.registered, "", false, false)
				after := f.object(t, collection, alice.Id, "https://local.example", "", true, false)
				for _, record := range []*core.Record{before, after} {
					for _, handle := range []string{"@alice", "@alice@" + change.domain} {
						if status, body := f.get(t, collection, record.Id, handle); status != http.StatusOK || !strings.Contains(body, record.Id) {
							t.Fatalf("%s via %s: status %d, want the local object", record.GetString("iri"), handle, status)
						}
					}
				}
				if fresh, _ := f.app.FindRecordById(collection, after.Id); fresh.GetBool("needs_full_sync") {
					t.Fatal("local object stays flagged for a remote sync")
				}
				if requests := f.transport.seen(); len(requests) > 0 {
					t.Fatalf("local objects were fetched remotely: %v", requests)
				}
			})
		}
	}
}

// A remote handle never selects a local object through its ID, whatever its
// IRI; a handle that cannot be resolved at all falls back to the plain local
// record, which the view rule then guards like a request without a handle.
func TestRemoteHandleLookupKeepsLocalRecordBindings(t *testing.T) {
	for _, collection := range []string{"trails", "lists"} {
		t.Run(collection, func(t *testing.T) {
			f := setupHandleLookupTest(t, "https://local.example", http.StatusNotFound)
			alice := f.localActor(t, "alice", "https://old.example", "old.example")
			f.remoteActor(t, "riley", "remote.example")
			for _, iriOrigin := range []string{"https://old.example", "https://local.example"} {
				for _, public := range []bool{true, false} {
					local := f.object(t, collection, alice.Id, iriOrigin, "", true, true)
					local.Set("public", public)
					if err := f.app.Save(local); err != nil {
						t.Fatal(err)
					}
					status, body := f.get(t, collection, local.Id, "@riley@remote.example")
					if status != http.StatusNotFound || strings.Contains(body, local.Id) {
						t.Fatalf("remote handle answered %d with the local object", status)
					}
					// 127.0.0.1 is refused by the address policy, so the actor lookup fails
					// and the request is answered like one without a handle.
					want := http.StatusOK
					if !public {
						want = http.StatusForbidden
					}
					if status, _ := f.get(t, collection, local.Id, "@nobody@127.0.0.1"); status != want {
						t.Fatalf("unresolvable handle answered %d, want %d", status, want)
					}
					fresh, err := f.app.FindRecordById(collection, local.Id)
					if err != nil || fresh.GetString("author") != alice.Id || fresh.GetString("iri") != local.GetString("iri") || fresh.GetBool("public") != public {
						t.Fatalf("local object changed or dropped: %v", err)
					}
				}
			}
			for _, request := range f.transport.seen() {
				if !strings.HasPrefix(request, "https://remote.example/") {
					t.Fatalf("local object was fetched from %s", request)
				}
			}
		})
	}
}

// Pages of local users link the trails of their activities through their own
// handle (profile statistics); a summit log on a remote trail then pairs a
// local handle with the local ID of a remote copy. The copy is served and
// refreshed from its own IRI only.
func TestRemoteHandleLookupServesRemoteCopiesThroughLocalHandles(t *testing.T) {
	for _, collection := range []string{"trails", "lists"} {
		t.Run(collection, func(t *testing.T) {
			f := setupHandleLookupTest(t, "https://local.example", http.StatusForbidden)
			f.localActor(t, "alice", "https://old.example", "old.example")
			bob := f.remoteActor(t, "bob", "remote.example")
			copyRecord := f.object(t, collection, bob.Id, "https://remote.example", "remoteid0000001", false, true)
			iri := copyRecord.GetString("iri")
			waitSyncIdle := func() {
				deadline := time.Now().Add(10 * time.Second)
				for time.Now().Before(deadline) {
					_, trailBusy := trailSyncing.Load(iri)
					_, listBusy := listSyncing.Load(iri)
					if !trailBusy && !listBusy {
						return
					}
					time.Sleep(time.Millisecond)
				}
				t.Errorf("background refresh for %s did not finish", iri)
			}
			t.Cleanup(waitSyncIdle)
			for _, handle := range []string{"@alice", "@alice@old.example"} {
				if status, body := f.get(t, collection, copyRecord.Id, handle); status != http.StatusOK || !strings.Contains(body, copyRecord.Id) {
					t.Fatalf("remote copy via local handle %s: status %d", handle, status)
				}
			}
			waitSyncIdle()
			if len(f.transport.seen()) == 0 {
				t.Fatal("remote copy was not refreshed")
			}
			for _, request := range f.transport.seen() {
				if !strings.HasPrefix(request, "https://remote.example/") {
					t.Fatalf("copy refreshed from %s instead of its own IRI", request)
				}
			}
			if fresh, err := f.app.FindRecordById(collection, copyRecord.Id); err != nil || fresh.GetString("author") != bob.Id {
				t.Fatal("copy changed or dropped")
			}
		})
	}
}
