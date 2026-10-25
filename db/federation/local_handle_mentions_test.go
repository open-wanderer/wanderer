package federation

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestActorsFromMentionsResolvesLocalHandleAliasesWithoutRequests(t *testing.T) {
	app := setupActorLookupTestApp(t)
	t.Setenv("ORIGIN", "https://www.local.example")
	actors, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"user", "preferred_username", "inbox"} {
		actors.Fields.Add(&core.TextField{Name: name})
	}
	if err := app.Save(actors); err != nil {
		t.Fatal(err)
	}
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	user := core.NewRecord(users)
	if err := app.SaveNoValidate(user); err != nil {
		t.Fatal(err)
	}
	settings := core.NewBaseCollection("settings")
	settings.Fields.Add(&core.TextField{Name: "user"}, &core.TextField{Name: "bio"})
	follows := core.NewBaseCollection("follows")
	follows.Fields.Add(&core.TextField{Name: "follower"}, &core.TextField{Name: "followee"}, &core.TextField{Name: "status"})
	for _, collection := range []*core.Collection{settings, follows} {
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}
	preferences := core.NewRecord(settings)
	preferences.Set("user", user.Id)
	if err := app.Save(preferences); err != nil {
		t.Fatal(err)
	}
	local := core.NewRecord(actors)
	local.Set("user", user.Id)
	local.Set("is_local", true)
	local.Set("preferred_username", "alice")
	local.Set("domain", "local.example")
	local.Set("iri", "https://www.local.example/api/v1/activitypub/user/alice")
	local.Set("inbox", local.GetString("iri")+"/inbox")
	if err := app.Save(local); err != nil {
		t.Fatal(err)
	}

	previousActor, previousWebfinger, previousCollection := newActorFetchHTTPClient, newWebfingerHTTPClient, newHTTPClient
	t.Cleanup(func() {
		newActorFetchHTTPClient, newWebfingerHTTPClient, newHTTPClient = previousActor, previousWebfinger, previousCollection
	})
	var actorRequests, webfingerRequests, collectionRequests atomic.Int32
	refuseRequests := func(counter *atomic.Int32) func() *http.Client {
		return func() *http.Client {
			return &http.Client{Transport: identityLookupTransport(func(request *http.Request) (*http.Response, error) {
				counter.Add(1)
				return nil, fmt.Errorf("unexpected request while resolving local mention: %s", request.URL)
			})}
		}
	}
	newActorFetchHTTPClient = refuseRequests(&actorRequests)
	newWebfingerHTTPClient = refuseRequests(&webfingerRequests)
	newHTTPClient = refuseRequests(&collectionRequests)

	for _, handle := range []string{"@ALICE", "@alice@www.local.example", "@ALICE@WWW.LOCAL.EXAMPLE", "@Alice@LOCAL.EXAMPLE"} {
		t.Run(handle, func(t *testing.T) {
			content := `<p>See <a class="mention" href="/profile">` + handle + `</a> for this trail.</p>`
			mentioned, err := ActorsFromMentions(app, context.Background(), content)
			if err != nil || len(mentioned) != 1 {
				t.Fatalf("local mention returned %d actors, error=%v", len(mentioned), err)
			}
			if mentioned[0].Id != local.Id || !mentioned[0].GetBool("is_local") || mentioned[0].GetString("inbox") != local.GetString("inbox") {
				t.Fatalf("mention did not select the stored local recipient: %v", mentioned[0])
			}
		})
	}
	if actorRequests.Load() != 0 || webfingerRequests.Load() != 0 || collectionRequests.Load() != 0 {
		t.Fatalf("local mentions made actor=%d WebFinger=%d collection=%d requests", actorRequests.Load(), webfingerRequests.Load(), collectionRequests.Load())
	}
}
