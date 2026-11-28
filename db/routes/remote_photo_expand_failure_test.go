package routes

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/dbx"
)

func TestRemoteFullSyncKeepsPhotosUntilAssetRelationsProveTheirSet(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	for _, target := range []string{"trail", "waypoint", "summit_log"} {
		for _, state := range []string{"missing", "unexpanded", "empty"} {
			t.Run(target+"/"+state, func(t *testing.T) {
				fixtureTarget := target
				if target == "trail" {
					fixtureTarget = "waypoint"
				}
				f := newRemoteChildCleanupFixture(t, fixtureTarget)
				child := f.child("https://a.example/api/v1/"+fixtureTarget+"/retained", f.remoteActor.Id, f.trail.Id)
				photo := f.photo(child, "retained.jpg")
				targetID, linkCollection := child.Id, f.links
				if target == "trail" {
					// Make this file exclusively a trail photo.
					links, err := f.app.FindAllRecords(f.links)
					if err != nil {
						t.Fatal(err)
					}
					for _, link := range links {
						if err := f.app.Delete(link); err != nil {
							t.Fatal(err)
						}
					}
					f.save(f.trailLinks, map[string]any{"trail": f.trail.Id, "asset": photo.Id})
					targetID, linkCollection = f.trail.Id, f.trailLinks
				}
				payload := map[string]any{"id": "trail", "name": "Updated", "public": true, "photos": []any{}}
				raw := payload
				if target != "trail" {
					raw = map[string]any{"id": "retained", "iri": child.GetString("iri"), "name": "Updated", "text": "Updated", "photos": []any{}}
					payload["expand"] = map[string]any{map[string]string{"waypoint": "waypoints_via_trail", "summit_log": "summit_logs_via_trail"}[target]: []any{raw}}
				}
				if state != "missing" {
					expand, _ := raw["expand"].(map[string]any)
					if expand == nil {
						expand = map[string]any{}
						raw["expand"] = expand
					}
					links := []any{}
					if state == "unexpanded" {
						links = []any{map[string]any{"id": "link", "asset": photo.Id}}
					}
					expand[remotePhotoAssetLinkKey(target)] = links
				}
				original := newRemoteSyncHTTPClient
				t.Cleanup(func() { newRemoteSyncHTTPClient = original })
				newRemoteSyncHTTPClient = func() *http.Client { return &http.Client{Transport: richTextSyncTransport{payload}} }
				if _, err := performFullSync(f.app, t.Context(), nil, f.trail); err != nil {
					t.Fatal(err)
				}
				count, err := f.app.CountRecords(linkCollection.Id, dbx.HashExp{target: targetID, "asset": photo.Id})
				want := 1
				if state == "empty" {
					want = 0
				}
				if err != nil || count != int64(want) {
					t.Fatalf("photo set after %s expand: links=%d want=%d error=%v", state, count, want, err)
				}
				f.assertFile(photo, want == 1)
			})
		}
	}
}

func TestLegacyTrailRetryCanConfirmRemovalOfAllLegacyPhotos(t *testing.T) {
	t.Setenv("ORIGIN", "https://b.example")
	f := newRemoteChildCleanupFixture(t, "waypoint")
	child := f.child("https://a.example/api/v1/waypoint/point", f.remoteActor.Id, f.trail.Id)
	photo := f.photo(child, "legacy.jpg")
	links, err := f.app.FindAllRecords(f.links)
	if err != nil {
		t.Fatal(err)
	}
	for _, link := range links {
		if err := f.app.Delete(link); err != nil {
			t.Fatal(err)
		}
	}
	f.save(f.trailLinks, map[string]any{"trail": f.trail.Id, "asset": photo.Id})
	original := newRemoteSyncHTTPClient
	t.Cleanup(func() { newRemoteSyncHTTPClient = original })
	requests := 0
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
			requests++
			if strings.Contains(request.URL.Query().Get("expand"), "_assets_via_") {
				return &http.Response{StatusCode: http.StatusRequestHeaderFieldsTooLarge, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("unsupported asset expands")), Request: request}, nil
			}
			return richTextSyncTransport{map[string]any{"id": "trail", "name": "Legacy", "public": true, "photos": []any{}}}.RoundTrip(request)
		}}}
	}
	if _, err := performFullSync(f.app, t.Context(), nil, f.trail); err != nil {
		t.Fatal(err)
	}
	count, err := f.app.CountRecords(f.trailLinks, dbx.HashExp{"trail": f.trail.Id, "asset": photo.Id})
	if err != nil || count != 0 || requests != 2 {
		t.Fatalf("confirmed legacy emptiness did not remove photos: links=%d requests=%d error=%v", count, requests, err)
	}
	f.assertFile(photo, false)
}
