package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"
	"pocketbase/routes"
	"pocketbase/services/trailmerge"
)

func TestTrailMergeRequiresOwnershipOfEverySource(t *testing.T) {
	f := newRecordHooksFixture(t)
	search := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"taskUid":1,"status":"enqueued"}`))
	}))
	t.Cleanup(search.Close)
	client := meilisearch.New(search.URL)
	owner, other := f.localActor("owner"), f.localActor("other")
	source := f.save("trails", map[string]any{"name": "Private source", "author": owner.Id})
	target := f.save("trails", map[string]any{"name": "Target", "author": other.Id, "public": true})
	entry := f.save("summit_logs", map[string]any{"trail": source.Id, "author": owner.Id, "text": "Source entry", "date": "2026-01-01 00:00:00.000Z"})
	user, err := f.app.FindRecordById("users", other.GetString("user"))
	if err != nil {
		t.Fatal(err)
	}
	for _, deleteSource := range []bool{false, true} {
		if trailmerge.CanMerge(f.app, other.Id, source, target, deleteSource) {
			t.Fatal("source owned by another actor accepted")
		}
		if err := trailmerge.Merge(f.app, client, context.Background(), other, source.Id, target.Id, trailmerge.MergeSettings{SummitLog: true, Delete: deleteSource}); !errors.Is(err, trailmerge.ErrMergeNotAllowed) {
			t.Fatalf("service error = %v", err)
		}
		body := fmt.Sprintf(`{"sourceTrailId":%q,"targetTrailId":%q,"settings":{"summitLog":true,"delete":%t}}`, source.Id, target.Id, deleteSource)
		e := &core.RequestEvent{App: f.app, Auth: user}
		e.Request = httptest.NewRequest(http.MethodPost, "/trail-merge", strings.NewReader(body))
		e.Request.Header.Set("Content-Type", "application/json")
		e.Response = httptest.NewRecorder()
		if err := routes.TrailMerge(nil)(e); err == nil {
			t.Fatal("route accepted source owned by another actor")
		}
	}
	if _, err := trailmerge.Suggest(f.app, other.Id, trailmerge.SuggestRequest{Mode: trailmerge.SuggestModeManualSelection, TrailIDs: []string{source.Id, target.Id}}); !errors.Is(err, trailmerge.ErrSourceActorMismatch) {
		t.Fatalf("suggestion error = %v", err)
	}
	for _, record := range []*core.Record{source, target, entry} {
		stored, err := f.app.FindRecordById(record.Collection(), record.Id)
		if err != nil || stored.GetString("author") != record.GetString("author") || stored.GetString("trail") != record.GetString("trail") {
			t.Fatalf("refused merge changed %s: record=%v error=%v", record.Collection().Name, stored, err)
		}
	}
}
