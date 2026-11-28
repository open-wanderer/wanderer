package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"image"
	"image/png"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestRemoteFullSyncCarriesStoredAndForwardedShareTokens(t *testing.T) {
	t.Setenv("ORIGIN", "https://a.example")
	for _, source := range []string{"stored", "forwarded"} {
		t.Run(source, func(t *testing.T) {
			f := newRemoteSyncFixture(t)
			f.remoteActor.Set("iri", "https://b.example/api/v1/actor/riley")
			f.remoteActor.Set("domain", "b.example")
			if err := f.app.Save(f.remoteActor); err != nil {
				t.Fatal(err)
			}
			origin := "https://b.example"
			token := "abcdefghijklmnopqrstuvwxyz012345"
			f.trails.Fields.Add(&core.FileField{Name: "gpx", MaxSelect: 1, MaxSize: 1 << 20})
			if err := f.app.Save(f.trails); err != nil {
				t.Fatal(err)
			}
			trail := f.save(f.trails, map[string]any{"iri": origin + "/api/v1/trail/remote", "author": f.remoteActor.Id})
			var requestURL *url.URL
			if source == "stored" {
				shares := core.NewBaseCollection("trail_share")
				shares.Fields.Add(&core.TextField{Name: "trail"}, &core.TextField{Name: "remote_token", Hidden: true})
				if err := f.app.Save(shares); err != nil {
					t.Fatal(err)
				}
				f.save(shares, map[string]any{"trail": trail.Id, "remote_token": token})
			} else {
				requestURL = &url.URL{RawQuery: "share=" + token}
			}
			assetLink := func(id string) map[string]any {
				return map[string]any{"expand": map[string]any{"asset": map[string]any{"id": id, "type": "photo", "file": id + ".png", "collectionId": "assets"}}}
			}
			waypoint := map[string]any{"id": "point", "iri": origin + "/api/v1/waypoint/point", "name": "Hut", "expand": map[string]any{"waypoint_assets_via_waypoint": []any{assetLink("pointphoto")}}}
			entry := map[string]any{"id": "entry", "iri": origin + "/api/v1/summit-log/entry", "text": "Entry", "author": f.remoteActor.Id, "expand": map[string]any{"author": map[string]any{"iri": f.remoteActor.GetString("iri")}, "trail": map[string]any{"iri": trail.GetString("iri")}, "summit_log_assets_via_summit_log": []any{assetLink("entryphoto")}}}
			documents := 0
			refreshTransport(t, func(request *http.Request) map[string]any {
				documents++
				if request.URL.Query().Get("share") != token {
					t.Fatal("document/child fetch lost token")
				}
				if strings.Contains(request.URL.Path, "/summit-log/") {
					return entry
				}
				return map[string]any{"id": "remote", "iri": trail.GetString("iri"), "name": "Shared", "gpx": "route.gpx", "public": false, "expand": map[string]any{"trail_assets_via_trail": []any{assetLink("trailphoto")}, "waypoints_via_trail": []any{waypoint}, "summit_logs_via_trail": []any{entry}}}
			})
			var imageBytes bytes.Buffer
			if err := png.Encode(&imageBytes, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
				t.Fatal(err)
			}
			previous := fetchRemoteSyncMedia
			t.Cleanup(func() { fetchRemoteSyncMedia = previous })
			downloads := map[string]int{}
			fetchRemoteSyncMedia = func(ctx context.Context, raw, origin, name string, limit int64) (*filesystem.File, func(), error) {
				parsed, err := url.Parse(raw)
				if err != nil || parsed.Query().Get("share") != token {
					t.Fatal("media fetch lost share token")
				}
				downloads[parsed.Path]++
				body := imageBytes.Bytes()
				if name == "route.gpx" {
					body = []byte("<gpx/>")
				}
				file, err := filesystem.NewFileFromBytes(body, name)
				return file, func() {}, err
			}
			result, err := performFullSync(f.app, t.Context(), requestURL, trail)
			if err != nil {
				t.Fatal(err)
			}
			if result.GetString("gpx") == "" || documents < 1 || len(downloads) != 4 {
				t.Fatalf("incomplete shared sync: documents=%d media=%v", documents, downloads)
			}
			for _, target := range []string{"trail", "waypoint", "summit_log"} {
				record := result
				if target != "trail" {
					collection := map[string]string{"waypoint": "waypoints", "summit_log": "summit_logs"}[target]
					iri := origin + "/api/v1/" + strings.ReplaceAll(target, "_", "-") + "/" + map[string]string{"waypoint": "point", "summit_log": "entry"}[target]
					record, err = f.app.FindFirstRecordByData(collection, "iri", iri)
					if err != nil {
						t.Fatal(err)
					}
				}
				photos := testPhotoAssets(t, f.app, record)
				if len(photos) != 1 || strings.Contains(photos[0].GetString("external_id"), "share=") {
					t.Fatalf("shared %s photo missing or identity contains credential: %v", target, photos)
				}
			}
		})
	}
}

func TestRemoteSyncUsesAnotherRecipientsTokenAfterRevocation(t *testing.T) {
	t.Setenv("ORIGIN", "https://a.example")
	f := newRemoteSyncFixture(t)
	f.remoteActor.Set("iri", "https://b.example/api/v1/actor/riley")
	f.remoteActor.Set("domain", "b.example")
	if err := f.app.Save(f.remoteActor); err != nil {
		t.Fatal(err)
	}
	origin := "https://b.example"
	trail := f.save(f.trails, map[string]any{"iri": origin + "/api/v1/trail/remote", "author": f.remoteActor.Id})
	shares := core.NewBaseCollection("trail_share")
	shares.Fields.Add(&core.TextField{Name: "trail"}, &core.TextField{Name: "remote_token", Hidden: true})
	if err := f.app.Save(shares); err != nil {
		t.Fatal(err)
	}
	revoked := strings.Repeat("a", 32)
	active := strings.Repeat("b", 32)
	f.save(shares, map[string]any{"id": "aaaaaaaaaaaaaaa", "trail": trail.Id, "remote_token": revoked})
	f.save(shares, map[string]any{"id": "bbbbbbbbbbbbbbb", "trail": trail.Id, "remote_token": active})
	previous := newRemoteSyncHTTPClient
	t.Cleanup(func() { newRemoteSyncHTTPClient = previous })
	attempts := []string{}
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
			token := request.URL.Query().Get("share")
			attempts = append(attempts, token)
			if token == revoked {
				return &http.Response{StatusCode: http.StatusNotFound, Header: make(http.Header), Body: http.NoBody, Request: request}, nil
			}
			if token != active {
				t.Fatal("sync selected an unknown recipient token")
			}
			return richTextSyncTransport{map[string]any{"id": "remote", "iri": trail.GetString("iri"), "name": "Still shared", "public": false, "expand": map[string]any{"trail_assets_via_trail": []any{}, "waypoints_via_trail": []any{}, "summit_logs_via_trail": []any{}}}}.RoundTrip(request)
		}}}
	}
	result, err := performFullSync(f.app, t.Context(), nil, trail)
	if err != nil || result.GetString("name") != "Still shared" || len(attempts) != 2 || attempts[0] != revoked || attempts[1] != active {
		t.Fatalf("another recipient's active share was ignored: attempts=%d error=%v", len(attempts), err)
	}
}

func TestRemoteDocumentFetchErrorsDoNotExposeShareTokens(t *testing.T) {
	token := "abcdefghijklmnopqrstuvwxyz012345"
	for _, kind := range []string{"child", "trail"} {
		for _, response := range []string{"status", "network"} {
			t.Run(kind+"/"+response, func(t *testing.T) {
				source, _ := url.Parse("https://b.example/api/v1/trail/remote?share=" + token)
				client := &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
					if request.URL.Query().Get("share") != token {
						t.Fatal("request lost token")
					}
					if response == "network" {
						return nil, fmt.Errorf("fetch failed for %s", request.URL)
					}
					return &http.Response{StatusCode: http.StatusNotFound, Header: make(http.Header), Body: http.NoBody, Request: request}, nil
				}}}
				var err error
				if kind == "child" {
					_, err = fetchRemoteDocument(t.Context(), client, source, "summit log")
				} else {
					_, err = fetchRemoteSyncDocument(t.Context(), client, source, "trail")
				}
				if err == nil || strings.Contains(err.Error(), token) {
					t.Fatalf("document fetch error exposes share credential: %v", err)
				}
				var status *remoteFetchStatusError
				if errors.As(err, &status) && strings.Contains(status.URL, token) {
					t.Fatal("status error retains share credential")
				}
				encoded, encodeErr := json.Marshal(err)
				if encodeErr != nil || bytes.Contains(encoded, []byte(token)) {
					t.Fatal("serialized document error exposes share credential")
				}
			})
		}
	}
}
