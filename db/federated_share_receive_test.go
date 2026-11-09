package main

import (
	"bytes"
	"encoding/json"
	pub "github.com/go-ap/activitypub"
	"image"
	"image/png"
	"io"
	"net/http"
	"pocketbase/federation"
	"pocketbase/util"
	"strings"
	"testing"
)

type federatedSharePhotoTransport func(*http.Request) (*http.Response, error)

func (f federatedSharePhotoTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestIncomingAnnounceStoresItsShareTokenHidden(t *testing.T) {
	f := newTrailMentionInboxFixture(t)
	recipient := f.actor(t, "recipient", true)
	sender := f.actor(t, "sender", false)
	sender.Set("iri", "https://b.example/api/v1/activitypub/user/sender")
	sender.Set("inbox", "https://b.example/inbox/sender")
	sender.Set("domain", "b.example")
	if err := f.app.Save(sender); err != nil {
		t.Fatal(err)
	}
	settings, err := f.app.FindFirstRecordByData("settings", "user", recipient.GetString("user"))
	if err != nil {
		t.Fatal(err)
	}
	settings.Set("notifications", map[string]any{"trail_share": map[string]any{"web": true, "email": false}})
	if err := f.app.Save(settings); err != nil {
		t.Fatal(err)
	}
	token := "abcdefghijklmnopqrstuvwxyz012345"
	var content bytes.Buffer
	if err := png.Encode(&content, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	downloads := 0
	restore := util.SetRemoteAttachmentClientForTesting(&http.Client{Transport: federatedSharePhotoTransport(func(request *http.Request) (*http.Response, error) {
		downloads++
		if request.URL.Query().Get("share") != token {
			t.Fatal("Announce photo download lost share token")
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(content.Bytes())), Request: request}, nil
	})})
	defer restore()
	iri := "https://b.example/api/v1/trail/shared"
	object := &pub.Object{ID: pub.IRI(iri), Type: pub.NoteType, AttributedTo: pub.IRI(sender.GetString("iri")), Name: pub.DefaultNaturalLanguageValue("Shared"), Location: &pub.Place{Type: pub.PlaceType}, Tag: pub.ItemCollection{}, Attachment: util.ActivityPubAttachmentArray(pub.ItemCollection{pub.Image{Type: pub.ImageType, ID: pub.IRI("https://b.example/api/v1/assets/photo"), URL: pub.IRI("https://b.example/api/v1/files/assets/photo/camera.png?share=" + token)}})}
	activity := pub.AnnounceNew(pub.IRI("https://b.example/api/v1/activitypub/activity/shared"), object)
	activity.Actor = pub.IRI(sender.GetString("iri"))
	activity.To = pub.ItemCollection{pub.IRI(recipient.GetString("iri"))}
	activity.Tag = pub.ItemCollection{pub.Object{Type: pub.NoteType, Name: pub.DefaultNaturalLanguageValue(util.FederatedShareTokenTag), Content: pub.DefaultNaturalLanguageValue(token)}}
	if err := federation.ProcessAnnounceActivity(f.app, sender, *activity); err != nil {
		t.Fatal(err)
	}
	trail, err := f.app.FindFirstRecordByData("trails", "iri", iri)
	if err != nil {
		t.Fatal(err)
	}
	if trail.GetBool("public") {
		t.Fatal("directed trail became public")
	}
	if got := util.RemoteTrailShareTokens(f.app, trail.Id); len(got) != 1 || got[0] != token {
		t.Fatal("incoming share token was not stored")
	}
	share, err := f.app.FindFirstRecordByData("trail_share", "trail", trail.Id)
	if err != nil {
		t.Fatal(err)
	}
	if share.GetString("federation_token") != "" {
		t.Fatal("remote credential grants local bearer reads")
	}
	wire, err := json.Marshal(share)
	if err != nil || strings.Contains(string(wire), token) || strings.Contains(string(wire), "remote_token") {
		t.Fatal("share response exposes token")
	}
	wire, err = json.Marshal(trail)
	if err != nil || strings.Contains(string(wire), token) {
		t.Fatal("copy response exposes token")
	}
	photos, err := util.PhotoAssetsForTarget(f.app, "trail", trail.Id, -1)
	if err != nil || len(photos) != 1 || downloads != 1 {
		t.Fatalf("shared photo not stored: count=%d downloads=%d error=%v", len(photos), downloads, err)
	}
	if photos[0].GetString("external_id") != "https://b.example/api/v1/assets/photo" {
		t.Fatal("photo identity changed")
	}
}
