package federation

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

type identityLookupTransport func(*http.Request) (*http.Response, error)

func (transport identityLookupTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestIdentityOnlyActorLookupDoesNotFetchFollowerCollections(t *testing.T) {
	app := setupActorLookupTestApp(t)
	originalActor, originalCollection := newActorFetchHTTPClient, newHTTPClient
	t.Cleanup(func() { newActorFetchHTTPClient, newHTTPClient = originalActor, originalCollection })
	actorRequests, collectionRequests := 0, 0
	client := &http.Client{Transport: identityLookupTransport(func(request *http.Request) (*http.Response, error) {
		actorRequests++
		iri := request.URL.String()
		body := fmt.Sprintf(`{"id":%q,"type":"Person","preferredUsername":"contributor","inbox":%q,"outbox":%q,"followers":%q,"following":%q,"publicKey":{"id":%q,"owner":%q,"publicKeyPem":"confirmed-key"}}`, iri, iri+"/inbox", iri+"/outbox", iri+"/followers", iri+"/following", iri+"#main-key", iri)
		return &http.Response{StatusCode: 200, Header: make(http.Header), Request: request, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	newActorFetchHTTPClient = func() *http.Client { return client }
	newHTTPClient = func() *http.Client {
		collectionRequests++
		return &http.Client{Transport: identityLookupTransport(func(request *http.Request) (*http.Response, error) {
			count := 5
			if strings.HasSuffix(request.URL.Path, "/following") {
				count = 8
			}
			body := fmt.Sprintf(`{"id":%q,"type":"OrderedCollectionPage","totalItems":%d,"orderedItems":[]}`, request.URL.String(), count)
			return &http.Response{StatusCode: 200, Header: make(http.Header), Request: request, Body: io.NopCloser(strings.NewReader(body))}, nil
		})}
	}
	for i := range 11 {
		iri := fmt.Sprintf("https://origin.example/users/contributor%d", i)
		actor, err := GetActorByIRI(app, WithActorIdentityLookup(context.Background()), iri, false)
		if err != nil || actor == nil || actor.GetString("public_key") != "confirmed-key" {
			t.Fatalf("identity-only actor lookup failed: %v %v", actor, err)
		}
	}
	if actorRequests != 11 || collectionRequests != 0 {
		t.Fatalf("identity lookup made %d actor/%d follower requests", actorRequests, collectionRequests)
	}
	count, err := app.CountRecords("activitypub_actors")
	if err != nil || count != 11 {
		t.Fatalf("confirmed actor identities not persisted: %d %v", count, err)
	}
	iri := "https://origin.example/users/contributor0"
	if _, err := GetActorByIRI(app, context.Background(), iri, false); err != nil {
		t.Fatal(err)
	}
	if actorRequests != 11 || collectionRequests != 0 {
		t.Fatal("plain identity reread fetched profile counts")
	}
	for i := range 3 {
		profile, err := GetActorByIRI(app, context.Background(), iri, true)
		if err != nil || profile == nil || profile.GetInt("follower_count") != 5 || profile.GetInt("following_count") != 8 || actorRequests != 11 || collectionRequests != 2 {
			t.Fatalf("profile lookup %d after identity sync: actor=%v err=%v actor requests=%d collection requests=%d", i+1, profile, err, actorRequests, collectionRequests)
		}
	}

	for _, tc := range []struct {
		name           string
		ctx            context.Context
		includeFollows bool
	}{
		{"profile", context.Background(), false},
		{"explicit", WithActorIdentityLookup(context.Background()), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			profile, err := GetActorByIRI(app, tc.ctx, "https://origin.example/users/"+tc.name, tc.includeFollows)
			if err != nil || profile == nil || profile.GetInt("follower_count") != 5 || profile.GetInt("following_count") != 8 {
				t.Fatalf("new profile lookup lost counts: %v %v", profile, err)
			}
		})
	}
}

func setupActorLookupTestApp(t *testing.T) *pbtests.TestApp {
	t.Helper()
	t.Setenv("ORIGIN", "https://local.example")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "test-key")
	app := setupActorDeleteTestApp(t)
	actors, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	actors.Fields.Add(
		&core.TextField{Name: "domain"}, &core.TextField{Name: "public_key"}, &core.DateField{Name: "last_fetched"},
		&core.TextField{Name: "followers"}, &core.TextField{Name: "following"},
		&core.NumberField{Name: "follower_count", OnlyInt: true}, &core.NumberField{Name: "following_count", OnlyInt: true},
		&core.BoolField{Name: "follower_count_pending", Hidden: true}, &core.BoolField{Name: "following_count_pending", Hidden: true},
	)
	if err := app.Save(actors); err != nil {
		t.Fatal(err)
	}
	return app
}
