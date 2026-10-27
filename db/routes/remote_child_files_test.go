package routes

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"io"
	"net"
	"net/http"
	"slices"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func saveRemoteChildRecord(t *testing.T, app core.App, collection string, values map[string]any) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(col)
	record.Load(values)
	if err := app.Save(record); err != nil {
		t.Fatal(err)
	}
	return record
}

func TestRemoteChildMetadataSurvivesFileFailures(t *testing.T) {
	var photo bytes.Buffer
	if err := png.Encode(&photo, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	photoBytes := photo.Bytes()
	gpxBytes := []byte(`<?xml version="1.0"?><gpx version="1.1"><trk><name>Saved route</name></trk></gpx>`)
	for _, kind := range []string{"waypoint", "summit-log"} {
		t.Run(kind, func(t *testing.T) {
			app := newRemoteRealSchemaApp(t)
			origin := "https://remote.example"
			actor := saveRemoteChildRecord(t, app, "activitypub_actors", map[string]any{
				"username": "riley@remote.example", "preferred_username": "riley", "domain": "remote.example",
				"iri": origin + "/api/v1/activitypub/user/riley", "inbox": origin + "/inbox/riley", "public_key": "test-key",
			})
			collection, relation, textField := "waypoints", "waypoints_via_trail", "description"
			if kind == "summit-log" {
				collection, relation, textField = "summit_logs", "summit_logs_via_trail", "text"
			}
			sequence := 0
			for _, failure := range []string{"server", "rate-limit", "forbidden", "network", "field-limit"} {
				for _, state := range []string{"new", "existing-empty", "existing-files"} {
					t.Run(failure+"/"+state, func(t *testing.T) {
						sequence++
						remoteID := fmt.Sprintf("child%010d", sequence)
						iri := origin + "/api/v1/" + kind + "/" + remoteID
						parentID := fmt.Sprintf("copy%011d", sequence)
						parent := saveRemoteChildRecord(t, app, "trails", map[string]any{"name": "Saved trail", "iri": origin + "/api/v1/trail/" + parentID, "author": actor.Id, "public": true, "needs_full_sync": true})
						var previous *core.Record
						var savedPhotos []string
						var savedGPX string
						newFile := func(data []byte, name string) *filesystem.File {
							t.Helper()
							file, err := filesystem.NewFileFromBytes(data, name)
							if err != nil {
								t.Fatal(err)
							}
							return file
						}
						if state != "new" {
							fields := map[string]any{"iri": iri, "name": "Child", textField: "Saved text", "author": actor.Id, "trail": parent.Id}
							if state == "existing-files" {
								fields["photos"] = []*filesystem.File{newFile(photoBytes, "saved.png")}
								if kind == "summit-log" {
									fields["gpx"] = newFile(gpxBytes, "saved.gpx")
								}
							}
							previous = saveRemoteChildRecord(t, app, collection, fields)
							savedPhotos, savedGPX = previous.GetStringSlice("photos"), previous.GetString("gpx")
						}
						text := "Updated text"
						raw := map[string]any{"id": remoteID, "iri": iri, "name": "Child", textField: text, "photos": []any{"next.png"}}
						if kind == "summit-log" {
							raw["gpx"] = "next.gpx"
						}
						refreshTransport(t, func(*http.Request) map[string]any {
							return map[string]any{"id": parentID, "iri": parent.GetString("iri"), "name": "Refreshed trail", "public": true, "expand": map[string]any{relation: []any{raw}}}
						})
						previousFetch := fetchRemoteSyncMedia
						t.Cleanup(func() { fetchRemoteSyncMedia = previousFetch })
						repaired, downloads := false, 0
						fetchRemoteSyncMedia = func(ctx context.Context, url, source, name string, limit int64) (*filesystem.File, func(), error) {
							downloads++
							field := "photos"
							if name == "next.gpx" {
								field = "gpx"
								if savedGPX != "" {
									t.Fatal("existing GPX requested again")
								}
							}
							col, err := app.FindCollectionByNameOrId(collection)
							if err != nil || limit != col.Fields.GetByName(field).(*core.FileField).MaxSize {
								t.Fatalf("local field limit was not used: %d, %v", limit, err)
							}
							if !repaired {
								err := fmt.Errorf("unexpected status code: %d", map[string]int{"server": 503, "rate-limit": 429, "forbidden": 403}[failure])
								switch failure {
								case "network":
									err = &net.OpError{Op: "read", Net: "tcp", Err: io.ErrUnexpectedEOF}
								case "field-limit":
									err = fmt.Errorf("file exceeds size limit %d", limit)
								}
								return nil, func() {}, err
							}
							data := photoBytes
							if name == "next.gpx" {
								data = gpxBytes
							}
							return newFile(data, name), func() {}, nil
						}
						var child *core.Record
						for attempt := 0; attempt < 3; attempt++ {
							if attempt == 1 {
								text = ""
								raw[textField] = text
							}
							if attempt == 2 {
								repaired = true
							}
							before := downloads
							var err error
							parent, err = performFullSync(app, t.Context(), nil, parent)
							if err != nil || parent.GetBool("needs_full_sync") || !parent.GetBool("full_sync_completed") {
								t.Fatalf("child file failure prevented metadata sync: %v", err)
							}
							if downloads == before {
								t.Fatal("missing child file was not requested")
							}
							child, err = app.FindFirstRecordByData(collection, "iri", iri)
							if err != nil {
								t.Fatalf("child metadata missing after file failure: %v", err)
							}
							if child.GetString(textField) != text || child.GetString("author") != actor.Id || child.GetString("trail") != parent.Id || (kind == "waypoint" && child.GetString("name") != "Child") {
								t.Fatalf("child metadata was not updated: %v", child.PublicExport())
							}
							if previous != nil && child.Id != previous.Id {
								t.Fatal("existing child was replaced")
							}
							if attempt < 2 && (!slices.Equal(child.GetStringSlice("photos"), savedPhotos) || child.GetString("gpx") != savedGPX) {
								t.Fatal("failed download changed existing files")
							}
						}
						photos := child.GetStringSlice("photos")
						if len(photos) != len(savedPhotos)+1 || !remotePhotoAlreadyStored(photos, "next.png") || (kind == "summit-log" && child.GetString("gpx") == "") {
							t.Fatal("later sync did not fill missing child files")
						}
						storage, err := app.NewFilesystem()
						if err != nil {
							t.Fatal(err)
						}
						defer storage.Close()
						for _, filename := range photos {
							reader, err := storage.GetReader(child.BaseFilesPath() + "/" + filename)
							if err != nil {
								t.Fatal(err)
							}
							data, err := io.ReadAll(reader)
							reader.Close()
							if err != nil || !bytes.Equal(data, photoBytes) {
								t.Fatal("stored photo bytes changed")
							}
						}
						if savedGPX != "" && child.GetString("gpx") != savedGPX {
							t.Fatal("stored GPX was replaced")
						}
						if kind == "summit-log" {
							reader, err := storage.GetReader(child.BaseFilesPath() + "/" + child.GetString("gpx"))
							if err != nil {
								t.Fatal(err)
							}
							data, err := io.ReadAll(reader)
							reader.Close()
							if err != nil || !bytes.Equal(data, gpxBytes) {
								t.Fatal("stored GPX bytes changed")
							}
						}
					})
				}
			}
		})
	}
}
