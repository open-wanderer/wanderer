package routes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"net/http"
	"net/http/httptest"
	"pocketbase/pluginsystem"
	"testing"
	"time"
)

func TestSyncPluginCapabilityAnnouncesAutoMergedEntry(t *testing.T) {
	app := newRemoteRealSchemaApp(t)
	t.Setenv("ORIGIN", "https://a.example")
	save := func(collection string, values map[string]any) *core.Record {
		t.Helper()
		return saveRemoteChildRecord(t, app, collection, values)
	}
	user := save("users", map[string]any{"username": "walker", "email": "walker@a.example", "password": "test-password"})
	actor := save("activitypub_actors", map[string]any{"user": user.Id, "preferred_username": "walker", "username": "walker", "is_local": true, "domain": "a.example", "iri": "https://a.example/api/v1/activitypub/user/walker", "public_key": "key", "inbox": "https://a.example/inbox"})
	track := `<gpx version="1.1" creator="test" xmlns="http://www.topografix.com/GPX/1/1"><trk><trkseg><trkpt lat="46.5" lon="7.5"/><trkpt lat="46.51" lon="7.52"/></trkseg></trk></gpx>`
	file, err := filesystem.NewFileFromBytes([]byte(track), "route.gpx")
	if err != nil {
		t.Fatal(err)
	}
	target := save("trails", map[string]any{"name": "Ridge", "author": actor.Id, "public": true, "gpx": file, "iri": "https://a.example/api/v1/trail/ridge"})
	instance := save("plugin_instances", map[string]any{"user": user.Id, "plugin_id": "komoot", "enabled": true, "status": "configured"})
	source := pluginsystem.TrailImportSource{Provider: "komoot", ExternalID: "walk-1"}
	started := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	session := &pluginSyncRuntimeSession{session: syncLimitTestSession{call: func(_ context.Context, export string, _ []byte) ([]byte, error) {
		if export == "list" {
			return json.Marshal(pluginSystemListOutput{Items: []pluginsystem.TrailSummary{{Source: source}}})
		}
		return json.Marshal(pluginSystemDetailOutput{Item: pluginsystem.TrailImport{Source: source, Name: "Ridge", StartedAt: &started, Track: pluginsystem.Track{Format: "gpx", ContentBase64: base64.StdEncoding.EncodeToString([]byte(track))}}})
	}}}
	search := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"taskUid":1,"status":"enqueued"}`))
	}))
	t.Cleanup(search.Close)
	var announced []string
	result, err := syncPluginCapability(t.Context(), app, meilisearch.New(search.URL), session, pluginsystem.LocalPlugin{Manifest: pluginsystem.Manifest{ID: "komoot"}}, pluginsystem.CapabilityManifest{Export: "list"}, pluginsystem.CapabilityManifest{Export: "detail"}, instance, actor, nil, nil, map[string]any{"merge": map[string]any{"enabled": true}}, false, false, func(entry *core.Record) { announced = append(announced, entry.Id) })
	if err != nil || result.Imported != 1 {
		t.Fatalf("sync result=%+v error=%v", result, err)
	}
	entries, err := app.FindAllRecords("summit_logs", dbx.HashExp{"trail": target.Id})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || len(announced) != 1 || announced[0] != entries[0].Id {
		t.Fatalf("merged entries=%d announced=%v", len(entries), announced)
	}
	activities, err := app.FindAllRecords("activitypub_activities")
	if err != nil {
		t.Fatal(err)
	}
	if len(activities) != 0 {
		t.Fatalf("sync wrote %d activities directly", len(activities))
	}
}
