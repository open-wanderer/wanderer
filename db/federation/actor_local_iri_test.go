package federation

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

type localIRILookupFixture struct {
	app                                                  *pbtests.TestApp
	before                                               map[string]map[string]any
	actorRequests, collectionRequests, webfingerRequests int
}

func newLocalIRILookupFixture(t *testing.T, origin, iri, state string) *localIRILookupFixture {
	t.Helper()
	f := &localIRILookupFixture{app: setupActorLookupTestApp(t), before: make(map[string]map[string]any)}
	t.Setenv("ORIGIN", origin)
	actors, err := f.app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"type", "user", "preferred_username", "username", "inbox", "outbox", "summary", "icon", "private_key"} {
		actors.Fields.Add(&core.TextField{Name: field})
	}
	if err := f.app.Save(actors); err != nil {
		t.Fatal(err)
	}
	save := func(recordIRI string, local, cached bool) {
		t.Helper()
		parsed, err := url.Parse(recordIRI)
		if err != nil {
			t.Fatal(err)
		}
		actor := core.NewRecord(actors)
		for field, value := range map[string]any{
			"iri": recordIRI, "is_local": local, "domain": parsed.Host,
			"preferred_username": "alice", "username": "Stored Alice", "user": "local-user",
			"inbox": recordIRI + "/inbox", "outbox": recordIRI + "/outbox",
			"followers": recordIRI + "/followers", "following": recordIRI + "/following",
			"summary": "Stored profile", "icon": "https://local.example/stored.png",
			"public_key": "stored-public-key", "private_key": "stored-private-key",
			"follower_count": 23, "following_count": 41,
			"follower_count_pending": true, "following_count_pending": true,
			"last_fetched": time.Now().Add(-3 * time.Hour),
		} {
			actor.Set(field, value)
		}
		if local {
			actor.Set("domain", strings.TrimPrefix(parsed.Hostname(), "www."))
		}
		if cached {
			actor.Set("last_fetched", time.Now())
		}
		if err := f.app.Save(actor); err != nil {
			t.Fatal(err)
		}
		stored, err := f.app.FindRecordById(actors, actor.Id)
		if err != nil {
			t.Fatal(err)
		}
		f.before[actor.Id] = stored.FieldsData()
	}
	save(origin+"/api/v1/activitypub/user/alice", true, true)
	if state != "new" {
		save(iri, false, state == "cached")
	}
	previousActor, previousCollection, previousWebfinger := newActorFetchHTTPClient, newHTTPClient, newWebfingerHTTPClient
	t.Cleanup(func() {
		newActorFetchHTTPClient, newHTTPClient, newWebfingerHTTPClient = previousActor, previousCollection, previousWebfinger
	})
	response := func(request *http.Request, body string) *http.Response {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Request: request, Body: io.NopCloser(strings.NewReader(body))}
	}
	newActorFetchHTTPClient = func() *http.Client {
		return &http.Client{Transport: identityLookupTransport(func(request *http.Request) (*http.Response, error) {
			f.actorRequests++
			document := *request.URL
			if document.Scheme == "http" {
				document.Scheme = "https"
			}
			id := document.String()
			body := fmt.Sprintf(`{"id":%q,"type":"Person","preferredUsername":"remote-alice","inbox":%q,"outbox":%q,"followers":%q,"following":%q,"publicKey":{"id":%q,"owner":%q,"publicKeyPem":"fetched-key"}}`, id, id+"/inbox", id+"/outbox", id+"/followers", id+"/following", id+"#main-key", id)
			return response(request, body), nil
		})}
	}
	newHTTPClient = func() *http.Client {
		return &http.Client{Transport: identityLookupTransport(func(request *http.Request) (*http.Response, error) {
			f.collectionRequests++
			return response(request, `{"type":"OrderedCollectionPage","totalItems":73,"orderedItems":[]}`), nil
		})}
	}
	newWebfingerHTTPClient = func() *http.Client {
		return &http.Client{Transport: identityLookupTransport(func(request *http.Request) (*http.Response, error) {
			f.webfingerRequests++
			return response(request, fmt.Sprintf(`{"links":[{"rel":"self","href":%q}]}`, iri)), nil
		})}
	}
	return f
}

func (f *localIRILookupFixture) assertStoredActorsUnchanged(t *testing.T, extraRows int) {
	t.Helper()
	for id, before := range f.before {
		stored, err := f.app.FindRecordById("activitypub_actors", id)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(stored.FieldsData(), before) {
			t.Errorf("stored actor %s changed during lookup", id)
		}
	}
	if count, err := f.app.CountRecords("activitypub_actors"); err != nil || count != int64(len(f.before)+extraRows) {
		t.Errorf("actor records after lookup: count=%d err=%v, want %d", count, err, len(f.before)+extraRows)
	}
}

func TestLocalActorIRIsWithoutExactLocalRecordAreNotFetchedOrCached(t *testing.T) {
	for _, scenario := range []struct {
		name, origin string
		iris         []string
	}{
		{"plain", "https://local.example", []string{
			"https://local.example/api/v1/activitypub/user/ALICE",
			"http://local.example/api/v1/activitypub/user/alice",
			"http://local.example:80/api/v1/activitypub/user/alice",
			"https://local.example:443/api/v1/activitypub/user/ALICE",
			"https://local.example/api/v1/activitypub/user/missing",
		}},
		{"www", "https://www.local.example", []string{
			"https://www.local.example/api/v1/activitypub/user/ALICE",
			"https://WWW.LOCAL.EXAMPLE/api/v1/activitypub/user/ALICE",
			"https://local.example/api/v1/activitypub/user/alice",
			"http://www.local.example/api/v1/activitypub/user/alice",
			"http://local.example:80/api/v1/activitypub/user/alice",
			"https://www.local.example/api/v1/activitypub/user/missing",
		}},
		{"www-port", "https://www.local.example:8443", []string{
			"https://www.local.example:8443/api/v1/activitypub/user/ALICE",
			"https://WWW.LOCAL.EXAMPLE:8443/api/v1/activitypub/user/ALICE",
			"https://local.example:8443/api/v1/activitypub/user/alice",
			"http://www.local.example:8443/api/v1/activitypub/user/alice",
			"http://local.example:8443/api/v1/activitypub/user/alice",
			"https://www.local.example:8443/api/v1/activitypub/user/missing",
		}},
		{"http", "http://local.example", []string{
			"http://local.example/api/v1/activitypub/user/ALICE",
			"https://local.example:80/api/v1/activitypub/user/alice",
			"http://local.example/api/v1/activitypub/user/missing",
		}},
	} {
		for index, iri := range scenario.iris {
			for _, state := range []string{"new", "cached", "expired"} {
				for _, follows := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/%d/%s/follows=%v", scenario.name, index, state, follows), func(t *testing.T) {
						f := newLocalIRILookupFixture(t, scenario.origin, iri, state)
						actor, err := GetActorByIRI(f.app, t.Context(), iri, follows)
						if actor != nil || !errors.Is(err, ErrActorNotFound) {
							t.Errorf("local IRI lookup returned actor=%v err=%v, want no actor and not found", actor, err)
						}
						if f.actorRequests != 0 || f.collectionRequests != 0 || f.webfingerRequests != 0 {
							t.Errorf("local IRI lookup made requests: actors=%d collections=%d WebFinger=%d", f.actorRequests, f.collectionRequests, f.webfingerRequests)
						}
						f.assertStoredActorsUnchanged(t, 0)
					})
				}
			}
		}
	}
}

func TestLocalActorIRIFromRemoteWebfingerIsNotFetchedOrCached(t *testing.T) {
	for _, scenario := range []struct{ origin, iri string }{
		{"https://local.example", "https://local.example/api/v1/activitypub/user/ALICE"},
		{"https://www.local.example", "http://local.example:80/api/v1/activitypub/user/alice"},
		{"https://www.local.example:8443", "http://local.example:8443/api/v1/activitypub/user/alice"},
	} {
		t.Run(scenario.origin, func(t *testing.T) {
			f := newLocalIRILookupFixture(t, scenario.origin, scenario.iri, "new")
			actor, err := GetActorByHandle(f.app, t.Context(), "remote-alice@directory.example", true)
			if actor != nil || !errors.Is(err, ErrActorNotFound) {
				t.Errorf("local WebFinger target returned actor=%v err=%v", actor, err)
			}
			if f.webfingerRequests != 1 || f.actorRequests != 0 || f.collectionRequests != 0 {
				t.Errorf("local WebFinger target requests: WebFinger=%d actor=%d collections=%d", f.webfingerRequests, f.actorRequests, f.collectionRequests)
			}
			f.assertStoredActorsUnchanged(t, 0)
		})
	}
}

func TestRemoteActorIRIsRemainFetchableOnOtherHostsAndPorts(t *testing.T) {
	for _, scenario := range []struct{ origin, iri string }{
		{"https://local.example", "https://remote.example/api/v1/activitypub/user/alice"},
		{"https://local.example", "https://local.example:8443/api/v1/activitypub/user/alice"},
		{"https://www.local.example", "https://local.example:8443/api/v1/activitypub/user/alice"},
		{"https://www.local.example:8443", "https://www.local.example:9443/api/v1/activitypub/user/alice"},
		{"https://www.local.example:8443", "https://www.local.example/api/v1/activitypub/user/alice"},
		{"https://www.local.example:8443", "http://www.local.example/api/v1/activitypub/user/alice"},
		{"https://www.local.example:8443", "http://www.local.example:80/api/v1/activitypub/user/alice"},
		{"https://local.example", "https://www.local.example/api/v1/activitypub/user/alice"},
		{"http://local.example", "https://local.example/api/v1/activitypub/user/alice"},
	} {
		t.Run(scenario.iri, func(t *testing.T) {
			f := newLocalIRILookupFixture(t, scenario.origin, scenario.iri, "new")
			actor, err := GetActorByIRI(f.app, t.Context(), scenario.iri, true)
			if err != nil || actor == nil || actor.GetBool("is_local") || actor.GetString("iri") != scenario.iri || actor.GetString("public_key") != "fetched-key" {
				t.Errorf("remote IRI lookup returned actor=%v err=%v", actor, err)
			}
			if f.actorRequests != 1 || f.collectionRequests != 2 || f.webfingerRequests != 0 {
				t.Errorf("remote IRI requests: actors=%d collections=%d WebFinger=%d", f.actorRequests, f.collectionRequests, f.webfingerRequests)
			}
			f.assertStoredActorsUnchanged(t, 1)
		})
	}
}
