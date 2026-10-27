package federation

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

type trailMentionFixture struct{ *addressingFixture }

func newTrailMentionFixture(t *testing.T) *trailMentionFixture {
	t.Helper()
	f := &trailMentionFixture{setupAddressingTestApp(t)}
	actors, err := f.app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	actors.Fields.Add(&core.TextField{Name: "user"})
	if err := f.app.Save(actors); err != nil {
		t.Fatal(err)
	}
	settings := core.NewBaseCollection("settings")
	settings.Fields.Add(&core.TextField{Name: "user"}, &core.JSONField{Name: "notifications"})
	notifications := core.NewBaseCollection("notifications")
	notifications.Fields.Add(&core.TextField{Name: "type"}, &core.TextField{Name: "author"}, &core.TextField{Name: "recipient"}, &core.JSONField{Name: "metadata"}, &core.BoolField{Name: "seen"})

	for _, collection := range []*core.Collection{settings, notifications} {
		if err := f.app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}
	return f
}

func (f *trailMentionFixture) localActor(t *testing.T, name string) *core.Record {
	t.Helper()
	actor := f.actor(t, name, true)
	user := f.record(t, "users", map[string]any{"email": name + "@example.test", "password": "local-mention-test-password", "passwordConfirm": "local-mention-test-password"})
	actor.Set("user", user.Id)
	actor.Set("iri", os.Getenv("ORIGIN")+"/api/v1/activitypub/user/"+name)
	actor.Set("inbox", actor.GetString("iri")+"/inbox")
	actor.Set("followers", actor.GetString("iri")+"/followers")
	if err := f.app.Save(actor); err != nil {
		t.Fatal(err)
	}
	f.record(t, "settings", map[string]any{"user": user.Id, "notifications": map[string]any{"trail_mention": map[string]any{"web": true, "email": false}}})
	return actor
}

func (f *trailMentionFixture) assertMentions(t *testing.T, author, trail *core.Record, recipients ...*core.Record) {
	t.Helper()
	records, err := f.app.FindAllRecords("notifications")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != len(recipients) {
		t.Fatalf("mention notifications=%d, want %d", len(records), len(recipients))
	}
	counts := map[string]int{}
	for _, record := range records {
		var metadata map[string]string
		if err := record.UnmarshalJSONField("metadata", &metadata); err != nil {
			t.Fatal(err)
		}
		if record.GetString("type") != "trail_mention" || record.GetString("author") != author.Id || metadata["id"] != trail.Id {
			t.Fatalf("unexpected mention notification: %v", record)
		}
		counts[record.GetString("recipient")]++
	}
	for _, recipient := range recipients {
		if counts[recipient.Id] != 1 {
			t.Fatalf("mention notifications for %s=%d, want 1", recipient.GetString("preferred_username"), counts[recipient.Id])
		}
	}
}

func TestLocalTrailMentionIsDeliveredOnceAmongThreeInboxes(t *testing.T) {
	f := newTrailMentionFixture(t)
	var requests atomic.Int32
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(origin.Close)
	t.Setenv("ORIGIN", origin.URL)
	author := f.localActor(t, "author")
	first := f.localActor(t, "first")
	second := f.localActor(t, "second")
	older := f.actor(t, "older", false)
	older.Set("inbox", origin.URL+"/api/v1/activitypub/user/mentioned/inbox")
	if err := f.app.Save(older); err != nil {
		t.Fatal(err)
	}
	mentioned := f.localActor(t, "mentioned")
	f.follow(t, first, author)
	f.follow(t, second, author)
	trail := f.record(t, "trails", map[string]any{"author": author.Id, "name": "Local trail", "public": true,
		"description": `<p><a class="mention">@first</a> <a class="mention">@mentioned</a></p>`})
	trail.Set("iri", origin.URL+"/api/v1/trail/"+trail.Id)
	if err := f.app.Save(trail); err != nil {
		t.Fatal(err)
	}
	if err := CreateTrailActivity(f.app, context.Background(), trail, pub.CreateType); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	if err := WaitForDeliveries(ctx); err != nil {
		t.Fatal(err)
	}
	f.assertMentions(t, author, trail, first, mentioned)
	if count, err := f.app.CountRecords("feed"); err != nil || count != 3 {
		t.Fatalf("local feed deliveries=%d, want 3: %v", count, err)
	}
	if requests.Load() != 0 {
		t.Fatalf("local delivery made %d ORIGIN requests", requests.Load())
	}
}
