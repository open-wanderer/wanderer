package federation

import (
	"bytes"
	"context"
	"encoding/json"
	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"pocketbase/util"
	"strings"
	"testing"
	"time"
)

type announceLogApp struct {
	core.App
	logger *slog.Logger
}

func (a announceLogApp) Logger() *slog.Logger { return a.logger }

func TestAnnouncePhotoTokensStayWithTheirShareRecipient(t *testing.T) {
	f := setupAddressingTestApp(t)
	actors, _ := f.app.FindCollectionByNameOrId("activitypub_actors")
	trails, _ := f.app.FindCollectionByNameOrId("trails")
	shares := core.NewBaseCollection("trail_share")
	shares.Fields.Add(&core.RelationField{Name: "actor", CollectionId: actors.Id, MaxSelect: 1}, &core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1}, &core.TextField{Name: "permission"}, &core.TextField{Name: "federation_token", Hidden: true}, &core.TextField{Name: "remote_token", Hidden: true})
	if err := f.app.Save(shares); err != nil {
		t.Fatal(err)
	}
	activities, _ := f.app.FindCollectionByNameOrId("activitypub_activities")
	activities.Fields.Add(&core.BoolField{Name: "hidden"})
	rule := "hidden=false"
	activities.ListRule = &rule
	activities.ViewRule = &rule
	if err := f.app.Save(activities); err != nil {
		t.Fatal(err)
	}
	owner := f.actor(t, "owner", true)
	recipient := f.actor(t, "recipient", false)
	follower := f.actor(t, "follower", false)
	f.follow(t, follower, owner)
	trail := f.record(t, "trails", map[string]any{"name": "Shared", "author": owner.Id, "public": true}, "trail")
	file, err := filesystem.NewFileFromBytes([]byte("GIF89a123456789"), "camera.gif")
	if err != nil {
		t.Fatal(err)
	}
	photo := f.record(t, "assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": owner.Id, "file": file})
	f.record(t, "trail_assets", map[string]any{"trail": trail.Id, "asset": photo.Id})
	share := f.record(t, "trail_share", map[string]any{"trail": trail.Id, "actor": recipient.Id, "permission": "view"})
	var logged bytes.Buffer
	app := announceLogApp{f.app, slog.New(slog.NewJSONHandler(&logged, nil))}
	if err := CreateAnnounceActivity(app, share, TrailAnnounceType); err != nil {
		t.Fatal(err)
	}
	delivered := f.delivered(t, "recipient")
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if err := WaitForDeliveries(ctx); err != nil {
		t.Fatal(err)
	}
	token := share.GetString("federation_token")
	if !util.ValidFederatedShareToken(token) {
		t.Fatal("missing share token")
	}
	body, err := json.Marshal(delivered)
	if err != nil {
		t.Fatal(err)
	}
	var activity pub.Activity
	if err := json.Unmarshal(body, &activity); err != nil {
		t.Fatal(err)
	}
	object, err := pub.ToObject(activity.Object)
	if err != nil {
		t.Fatal(err)
	}
	attachments, err := util.ActivityPubAttachmentItems(object.Attachment)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, item := range attachments.Collection() {
		attachment, err := pub.ToObject(item)
		if err != nil {
			t.Fatal(err)
		}
		if attachment.Type != pub.ImageType {
			continue
		}
		count++
		parsed, err := url.Parse(attachment.URL.GetLink().String())
		if err != nil || parsed.Query().Get("share") != token {
			t.Fatal("recipient photo URL has no directed token")
		}
		if strings.Contains(attachment.ID.String(), token) || strings.Contains(attachment.ID.String(), "share=") {
			t.Fatal("photo identity contains token")
		}
	}
	if count != 1 {
		t.Fatalf("photos=%d", count)
	}
	if f.in.wait("follower", 1, 100*time.Millisecond) != 0 {
		t.Fatal("private Announce reached a follower")
	}
	if err := CreateTrailActivity(app, t.Context(), trail, pub.CreateType); err != nil {
		t.Fatal(err)
	}
	publicActivity := f.delivered(t, "follower")
	publicBody, err := json.Marshal(publicActivity)
	if err != nil || bytes.Contains(publicBody, []byte(token)) {
		t.Fatal("public activity exposes share token")
	}
	if err := WaitForDeliveries(ctx); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(logged.Bytes(), []byte(token)) {
		t.Fatal("delivery logs expose share token")
	}
	public, err := util.ObjectFromTrail(f.app, trail, nil)
	if err != nil {
		t.Fatal(err)
	}
	publicJSON, err := json.Marshal(public)
	if err != nil || bytes.Contains(publicJSON, []byte(token)) {
		t.Fatal("public object exposes share token")
	}
	recorded, err := f.app.FindFirstRecordByData("activitypub_activities", "type", "Announce")
	if err != nil || !recorded.GetBool("hidden") {
		t.Fatal("Announce is public")
	}
	event := &core.RequestEvent{App: f.app}
	event.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	info, _ := event.RequestInfo()
	allowed, err := f.app.CanAccessRecord(recorded, info, activities.ViewRule)
	if err != nil || allowed {
		t.Fatalf("private activity readable: %t %v", allowed, err)
	}
}
