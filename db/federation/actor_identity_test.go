package federation

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

type actorIdentityTransport struct{ document string }

func (t actorIdentityTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(t.document))}, nil
}

func TestCachedActorIdentityCannotUseKeyFromAnotherOrigin(t *testing.T) {
	collection := core.NewBaseCollection("activitypub_actors")
	collection.Fields.Add(&core.TextField{Name: "iri"}, &core.TextField{Name: "domain"}, &core.TextField{Name: "public_key"})
	actor := core.NewRecord(collection)
	actor.Set("iri", "https://origin.example/api/v1/activitypub/user/carol")
	actor.Set("domain", "peer.example")
	actor.Set("public_key", "cached-key")
	if CachedActorIdentityMatchesOrigin(actor) {
		t.Fatal("historical cache with a different host can authenticate")
	}
	actor.Set("domain", "origin.example")
	if !CachedActorIdentityMatchesOrigin(actor) {
		t.Fatal("valid cached actor rejected")
	}
}

// An inconsistent cached record is fetched again from the host of its IRI.
func TestInconsistentCachedActorIsRefetchedFromItsOwnHost(t *testing.T) {
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "test-key")
	app := setupActorDeleteTestApp(t)
	collection, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	collection.Fields.Add(&core.TextField{Name: "domain"}, &core.TextField{Name: "public_key"}, &core.DateField{Name: "last_fetched"})
	if err := app.Save(collection); err != nil {
		t.Fatal(err)
	}
	origin := "https://origin.example/api/v1/activitypub/user/carol"
	cached := core.NewRecord(collection)
	cached.Set("iri", origin)
	cached.Set("domain", "peer.example")
	cached.Set("public_key", "cached-key")
	cached.Set("last_fetched", time.Now().UTC())
	if err := app.Save(cached); err != nil {
		t.Fatal(err)
	}
	original := newActorFetchHTTPClient
	newActorFetchHTTPClient = func() *http.Client {
		return &http.Client{Transport: actorIdentityTransport{fmt.Sprintf(`{"id":%q,"type":"Person","preferredUsername":"carol","inbox":%q,"outbox":%q,"publicKey":{"id":%q,"owner":%q,"publicKeyPem":"origin-key"}}`, origin, origin+"/inbox", origin+"/outbox", origin+"#main-key", origin)}}
	}
	defer func() { newActorFetchHTTPClient = original }()
	actor, err := GetActorByIRI(app, context.Background(), origin, false)
	if err != nil || actor == nil || actor.GetString("public_key") != "origin-key" || actor.GetString("domain") != "origin.example" {
		t.Fatalf("inconsistent cache served: actor=%v err=%v", actor, err)
	}
}

func TestActorFetchCannotCacheAnotherActorsIdentityOrKeyOwner(t *testing.T) {
	requested := "https://peer.example/api/v1/activitypub/user/peer"
	origin := "https://origin.example/api/v1/activitypub/user/carol"
	for _, scenario := range []struct {
		name, id, owner, key string
		valid                bool
	}{
		// Vary the document ID independently from the owner and key host.
		{name: "foreign-id", id: origin, owner: requested, key: requested + "#main-key"},
		{name: "foreign-identity", id: origin, owner: origin, key: origin + "#main-key"},
		{name: "foreign-owner", id: requested, owner: origin, key: requested + "#main-key"},
		{name: "foreign-key-host", id: requested, owner: requested, key: origin + "#main-key"},
		{name: "mastodon-key", id: requested, owner: requested, key: requested + "#main-key", valid: true},
		{name: "gotosocial-key", id: requested, owner: requested, key: requested + "/main-key", valid: true},
		{name: "bookwyrm-key", id: requested, owner: requested, key: requested + "/#main-key", valid: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Setenv("POCKETBASE_ENCRYPTION_KEY", "test-key")
			app := setupActorDeleteTestApp(t)
			original := newActorFetchHTTPClient
			newActorFetchHTTPClient = func() *http.Client {
				return &http.Client{Transport: actorIdentityTransport{fmt.Sprintf(`{"id":%q,"type":"Person","preferredUsername":"peer","inbox":%q,"outbox":%q,"publicKey":{"id":%q,"owner":%q,"publicKeyPem":"cached-key"}}`, scenario.id, requested+"/inbox", requested+"/outbox", scenario.key, scenario.owner)}}
			}
			defer func() { newActorFetchHTTPClient = original }()
			actor, err := GetActorByIRI(app, context.Background(), requested, false)
			count, countErr := app.CountRecords("activitypub_actors")
			if countErr != nil {
				t.Fatal(countErr)
			}
			if scenario.valid {
				if err != nil || actor == nil || actor.GetString("iri") != requested || count != 1 {
					t.Fatalf("valid actor rejected: actor=%v err=%v count=%d", actor, err, count)
				}
			} else if err == nil || count != 0 {
				t.Fatalf("invalid actor was stored in cache: err=%v count=%d", err, count)
			}
		})
	}
}
