package federation

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

func TestLocalDeliveryMaintainsFeedFollowLikeAndShareWithoutOriginHTTP(t *testing.T) {
	f := setupAddressingTestApp(t)
	var requests atomic.Int32
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(origin.Close)
	t.Setenv("ORIGIN", origin.URL)
	actors, _ := f.app.FindCollectionByNameOrId("activitypub_actors")
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
	sender := f.actor(t, "alice", true)
	recipient := f.actor(t, "bob", true)
	for _, actor := range []*core.Record{sender, recipient} {
		actor.Set("iri", origin.URL+"/api/v1/activitypub/user/"+actor.GetString("preferred_username"))
		actor.Set("inbox", actor.GetString("iri")+"/inbox")
		actor.Set("user", actor.Id)
		if err := f.app.Save(actor); err != nil {
			t.Fatal(err)
		}
		f.record(t, "settings", map[string]any{"user": actor.Id, "notifications": map[string]any{
			"new_follower": map[string]any{"web": true, "email": false}, "trail_like": map[string]any{"web": true, "email": false}, "trail_share": map[string]any{"web": true, "email": false}}})
	}
	wait := func() {
		t.Helper()
		ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
		defer cancel()
		if err := WaitForDeliveries(ctx); err != nil {
			t.Fatal(err)
		}
	}
	follow := f.record(t, "follows", map[string]any{"follower": sender.Id, "followee": recipient.Id, "status": "pending"})
	if err := CreateFollowActivity(f.app, follow); err != nil {
		t.Fatal(err)
	}
	wait()
	stored, _ := f.app.FindRecordById("follows", follow.Id)
	if stored.GetString("status") != "accepted" {
		t.Fatal("local follow was not accepted")
	}
	trail := f.record(t, "trails", map[string]any{"name": "Local trail", "author": sender.Id, "public": true}, "trail")
	trail.Set("iri", origin.URL+"/api/v1/trail/"+trail.Id)
	if err := f.app.Save(trail); err != nil {
		t.Fatal(err)
	}
	create := pub.CreateNew(pub.IRI(origin.URL+"/activity/create"), &pub.Object{ID: pub.IRI(trail.GetString("iri")), Type: pub.NoteType})
	create.Actor = pub.IRI(sender.GetString("iri"))
	if err := PostActivity(f.app, sender, create, []string{recipient.GetString("inbox")}); err != nil {
		t.Fatal(err)
	}
	wait()
	feeds, _ := f.app.FindAllRecords("feed")
	if len(feeds) != 1 || feeds[0].GetString("actor") != recipient.Id || feeds[0].GetString("item") != trail.Id {
		t.Fatalf("local feed missing: %v", feeds)
	}
	like := pub.LikeNew(pub.IRI(origin.URL+"/activity/like"), pub.IRI(trail.GetString("iri")))
	like.Actor = pub.IRI(recipient.GetString("iri"))
	if err := PostActivity(f.app, recipient, like, []string{sender.GetString("inbox")}); err != nil {
		t.Fatal(err)
	}
	wait()
	share := pub.AnnounceNew(pub.IRI(origin.URL+"/activity/share"), pub.IRI(trail.GetString("iri")))
	share.Actor, share.To = pub.IRI(sender.GetString("iri")), pub.ItemCollection{pub.IRI(recipient.GetString("iri"))}
	if err := PostActivity(f.app, sender, share, []string{recipient.GetString("inbox")}); err != nil {
		t.Fatal(err)
	}
	wait()
	seen := map[string]int{}
	records, _ := f.app.FindAllRecords("notifications")
	for _, record := range records {
		seen[record.GetString("type")]++
	}
	for _, kind := range []string{"new_follower", "trail_like", "trail_share"} {
		if seen[kind] != 1 {
			t.Fatalf("local %s notification count=%d, want1", kind, seen[kind])
		}
	}
	if requests.Load() != 0 {
		t.Fatalf("local activities attempted %d external requests", requests.Load())
	}
	for _, state := range []string{"remote-sender", "wrong-wire-actor", "wrong-key"} {
		candidate := sender.Fresh()
		if state == "remote-sender" {
			candidate.Set("is_local", false)
			candidate.Set("iri", "https://remote.example/actor")
		}
		if state == "wrong-key" {
			candidate.Set("private_key", "another key")
		}
		wireActor := sender.GetString("iri")
		if state == "wrong-wire-actor" {
			wireActor = recipient.GetString("iri")
		}
		body := []byte(`{"type":"Like","actor":"` + wireActor + `","object":"` + trail.GetString("iri") + `"}`)
		if handled, err := deliverLocalActivity(f.app, candidate, recipient.GetString("inbox"), body); !handled || err == nil {
			t.Fatalf("unbound %s activated local delivery", state)
		}
	}
}
