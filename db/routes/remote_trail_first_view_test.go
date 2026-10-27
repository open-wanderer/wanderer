package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func TestRemoteTrailFirstViewCompletesAfterInboundUpdate(t *testing.T) {
	w := newKnownKeyWorld(t)
	inbox := knownKeyInboxRouter(t, w)
	actor := w.sender(t, "riley", "remote.example", false)
	iri := "https://remote.example/api/v1/trail/morningwalk0001"
	object := orderTrail(iri, actor.GetString("iri"))
	create := pub.Activity{ID: "https://remote.example/activities/create-walk", Type: pub.CreateType, Object: object}
	if status, err := w.deliverOrderedActivity(inbox, actor, w.recipient, create); err != nil || status != http.StatusOK {
		t.Fatalf("create failed: status=%d error=%v", status, err)
	}
	placeholder, err := w.app.FindFirstRecordByData("trails", "iri", iri)
	if err != nil {
		t.Fatal(err)
	}
	if hasCompletedFullSync(placeholder) || !placeholder.GetBool("needs_full_sync") {
		t.Fatal("create did not leave an incomplete trail")
	}
	// The stored time precedes the inbound save even on a fast local database.
	old, err := types.ParseDateTime(time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	placeholder.SetRaw("updated", old)
	if err := w.app.SaveNoValidate(placeholder); err != nil {
		t.Fatal(err)
	}
	previousHandle, previousClient := remoteHandleGetActor, newRemoteSyncHTTPClient
	t.Cleanup(func() { remoteHandleGetActor, newRemoteSyncHTTPClient = previousHandle, previousClient })
	remoteHandleGetActor = func(app core.App, ctx context.Context, handle string, follows bool) (*core.Record, error) {
		return app.FindRecordById("activitypub_actors", actor.Id)
	}
	var documents atomic.Int32
	newRemoteSyncHTTPClient = func() *http.Client {
		return &http.Client{Transport: remoteSyncTransport{roundTrip: func(request *http.Request) (*http.Response, error) {
			if documents.Add(1) == 1 {
				update := pub.Activity{ID: "https://remote.example/activities/update-walk", Type: pub.UpdateType, Object: object}
				if _, err := w.deliverOrderedActivity(inbox, actor, w.recipient, update); err != nil {
					return nil, err
				}
			}
			return richTextSyncTransport{map[string]any{
				"id": "morningwalk0001", "iri": iri, "name": "Morning walk", "public": true,
				"expand": map[string]any{
					"author":                map[string]any{"iri": actor.GetString("iri")},
					"waypoints_via_trail":   []any{map[string]any{"id": "restpoint000001", "name": "Rest point", "lat": 47, "lon": 8}},
					"summit_logs_via_trail": []any{map[string]any{"id": "logentry0000001", "text": "A clear morning"}},
				},
			}}.RoundTrip(request)
		}}}
	}
	router, err := apis.NewRouter(w.app)
	if err != nil {
		t.Fatal(err)
	}
	router.GET("/remote/trail/{id}", RemoteTrailGet)
	handler, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/remote/trail/morningwalk0001?handle=@riley@remote.example&expand=waypoints_via_trail,summit_logs_via_trail", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("view failed: status=%d body=%s", response.Code, response.Body.String())
	}
	var body struct {
		ID                string                       `json:"id"`
		NeedsFullSync     bool                         `json:"needs_full_sync"`
		FullSyncCompleted bool                         `json:"full_sync_completed"`
		Expand            map[string][]json.RawMessage `json:"expand"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	stored, err := w.app.FindRecordById("trails", placeholder.Id)
	if err != nil {
		t.Fatal(err)
	}
	if !body.FullSyncCompleted || body.NeedsFullSync || !hasCompletedFullSync(stored) || stored.GetBool("needs_full_sync") {
		t.Fatalf("first view returned an incomplete trail: body=%s stored=%v", response.Body.String(), stored)
	}
	if body.ID != placeholder.Id || documents.Load() != 2 {
		t.Fatalf("view did not refresh once from the stored trail: id=%s documents=%d", body.ID, documents.Load())
	}
	for _, collection := range []string{"waypoints", "summit_logs"} {
		records, err := w.app.FindAllRecords(collection)
		if err != nil || len(records) != 1 || records[0].GetString("trail") != placeholder.Id || len(body.Expand[collection+"_via_trail"]) != 1 {
			t.Fatalf("first view omitted %s: records=%v error=%v body=%s", collection, records, err, response.Body.String())
		}
	}
}
