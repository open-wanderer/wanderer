package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"

	"pocketbase/hooks"
	"pocketbase/routes"
	"pocketbase/services/trailmerge"
)

// A trail merge copies the summit logs of its source to the target and links
// their photos to the copies, in one transaction. The links run their hooks
// after the commit and queue an Update of each copy, so the merge announces
// the copies through the same queue: followers get one Create per log, with
// its photos, and whoever it mentions is notified once.
func TestTrailMergeAnnouncesSummitLogsWithOneCreate(t *testing.T) {
	f := newRecordHooksFixture(t)
	alice := f.localActor("alice")
	user, err := f.app.FindRecordById("users", alice.GetString("user"))
	if err != nil {
		t.Fatal(err)
	}
	source := f.save("trails", map[string]any{"name": "Source", "author": alice.Id, "date": "2026-01-01 00:00:00.000Z"})
	target := f.save("trails", map[string]any{"name": "Target", "author": alice.Id, "public": true,
		"iri": "https://a.example/api/v1/trail/target"})
	log := f.save("summit_logs", map[string]any{"trail": source.Id, "author": alice.Id, "date": "2026-02-01 00:00:00.000Z",
		"text": "Summit"})
	photo := f.save("assets", map[string]any{"type": "photo", "storage_mode": "copy", "author": alice.Id, "file": "summit.jpg"})
	f.save("summit_log_assets", map[string]any{"asset": photo.Id, "summit_log": log.Id})

	// The queue and its link hooks, as setupEventHandlers binds them.
	updates := hooks.NewFederationUpdates(f.app)
	links := hooks.FederateOnAssetLinkChange(updates)
	f.app.OnRecordAfterCreateSuccess("trail_assets", "waypoint_assets", "summit_log_assets").BindFunc(links)
	f.app.OnRecordAfterUpdateSuccess("trail_assets").BindFunc(links)
	f.app.OnRecordAfterDeleteSuccess("trail_assets", "waypoint_assets", "summit_log_assets").BindFunc(links)
	search := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"taskUid":1,"status":"enqueued"}`))
	}))
	t.Cleanup(search.Close)

	body := fmt.Sprintf(`{"sourceTrailId":%q,"targetTrailId":%q,"settings":{"summitLog":true,"photos":true}}`, source.Id, target.Id)
	e := &core.RequestEvent{App: f.app, Auth: user}
	e.Request = httptest.NewRequest(http.MethodPost, "/trail-merge", strings.NewReader(body))
	e.Request.Header.Set("Content-Type", "application/json")
	e.Response = httptest.NewRecorder()
	if err := routes.TrailMerge(meilisearch.New(search.URL), updates.EnqueueCreate)(e); err != nil {
		t.Fatal(err)
	}
	if sent := federatedActivities(t, f.app); len(sent) != 0 {
		t.Fatalf("the merge sent %v before the photos had settled", sent)
	}

	// A shutdown sends what is pending without waiting for it to settle.
	if err := f.app.OnTerminate().Trigger(&core.TerminateEvent{App: f.app}); err != nil {
		t.Fatal(err)
	}
	// The log made from the source trail, and the copy of its log.
	want := []string{"Create with 0 photos", "Create with 1 photos"}
	if sent := federatedActivities(t, f.app); !slices.Equal(sent, want) {
		t.Fatalf("sent %v, want %v", sent, want)
	}
}

// A plugin sync merges an imported trail into the one trail of its author
// that it matches, when auto-merge is on, and hands the summit logs that merge
// creates to the same queue as the route does.
func TestTrailAutoMergeHandsSummitLogsToAnnouncer(t *testing.T) {
	f := newRecordHooksFixture(t)
	alice := f.localActor("alice")
	gpx := func() *filesystem.File {
		track := `<?xml version="1.0" encoding="UTF-8"?><gpx version="1.1" creator="test" xmlns="http://www.topografix.com/GPX/1/1">` +
			`<trk><trkseg><trkpt lat="46.5" lon="7.5"></trkpt><trkpt lat="46.51" lon="7.52"></trkpt></trkseg></trk></gpx>`
		file, err := filesystem.NewFileFromBytes([]byte(track), "track.gpx")
		if err != nil {
			t.Fatal(err)
		}
		return file
	}
	target := f.save("trails", map[string]any{"name": "Ridge", "author": alice.Id, "public": true, "gpx": gpx(),
		"iri": "https://a.example/api/v1/trail/ridge"})
	imported := f.save("trails", map[string]any{"name": "Ridge", "author": alice.Id, "date": "2026-01-01 00:00:00.000Z",
		"gpx": gpx()})
	search := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"taskUid":1,"status":"enqueued"}`))
	}))
	t.Cleanup(search.Close)

	announced := []string{}
	err := trailmerge.TryAutoMergeImportedTrail(f.app, meilisearch.New(search.URL), context.Background(), alice, imported.Id,
		trailmerge.PluginAutoMergeSettings{Enabled: true}, func(log *core.Record) { announced = append(announced, log.Id) })
	if err != nil {
		t.Fatal(err)
	}

	logs, err := f.app.FindAllRecords("summit_logs", dbx.HashExp{"trail": target.Id})
	if err != nil {
		t.Fatal(err)
	}
	// The log made from the imported trail.
	if len(logs) != 1 || !slices.Equal(announced, []string{logs[0].Id}) {
		t.Fatalf("announced %v of %d summit logs on the target", announced, len(logs))
	}
	if sent := federatedActivities(t, f.app); len(sent) != 0 {
		t.Fatalf("the auto-merge sent %v itself", sent)
	}
}

// federatedActivities lists the activities that have gone out, each by its
// type and the number of photos its object carries.
func federatedActivities(t *testing.T, app core.App) []string {
	t.Helper()
	records, err := app.FindAllRecords("activitypub_activities")
	if err != nil {
		t.Fatal(err)
	}
	activities := []string{}
	for _, record := range records {
		var object map[string]any
		if err := record.UnmarshalJSONField("object", &object); err != nil {
			t.Fatal(err)
		}
		// A single attachment is not wrapped in a list.
		attachments, ok := object["attachment"].([]any)
		if !ok && object["attachment"] != nil {
			attachments = []any{object["attachment"]}
		}
		photos := 0
		for _, attachment := range attachments {
			if document, ok := attachment.(map[string]any); ok && document["type"] == "Image" {
				photos++
			}
		}
		activities = append(activities, fmt.Sprintf("%s with %d photos", record.GetString("type"), photos))
	}
	slices.Sort(activities)
	return activities
}
