package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	pub "github.com/go-ap/activitypub"
	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"

	"pocketbase/federation"
	"pocketbase/services/trailmerge"
)

func TestMergedEntriesKeepTheirDeliveryState(t *testing.T) {
	f := newRecordHooksFixture(t)
	owner, contributor := f.localActor("owner"), f.localActor("contributor")
	source := f.save("trails", map[string]any{"name": "Source", "author": owner.Id})
	target := f.save("trails", map[string]any{"name": "Target", "author": owner.Id, "public": true})
	f.save("summit_logs", map[string]any{"trail": source.Id, "author": contributor.Id, "text": "Contribution", "date": "2026-02-01 00:00:00.000Z"})
	f.save("summit_logs", map[string]any{"trail": source.Id, "author": owner.Id, "text": "Earlier copy", "date": "2026-02-03 00:00:00.000Z", "federation_suppressed": true})
	f.bindHooks()
	search := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"taskUid":1,"status":"enqueued"}`))
	}))
	t.Cleanup(search.Close)
	if err := trailmerge.Merge(f.app, meilisearch.New(search.URL), context.Background(), owner, source.Id, target.Id, trailmerge.MergeSettings{SummitLog: true}); err != nil {
		t.Fatal(err)
	}
	logs, err := f.app.FindRecordsByFilter("summit_logs", "trail={:trail} && author={:author}", "", -1, 0, dbx.Params{"trail": target.Id, "author": contributor.Id})
	if err != nil || len(logs) != 1 || !logs[0].GetBool("federation_suppressed") {
		t.Fatalf("merged contribution did not retain its delivery marker: records=%v error=%v", logs, err)
	}
	copy := logs[0]
	retained, err := f.app.FindFirstRecordByFilter("summit_logs", "trail={:trail} && text='Earlier copy'", dbx.Params{"trail": target.Id})
	if err != nil || !retained.GetBool("federation_suppressed") {
		t.Fatalf("repeated merge changed the existing delivery state: record=%v error=%v", retained, err)
	}
	count := func(want int64) {
		t.Helper()
		actual, err := f.app.CountRecords("activitypub_activities", dbx.HashExp{"actor": contributor.GetString("iri")})
		if err != nil || actual != want {
			t.Fatalf("contributor activities=%d want=%d error=%v", actual, want, err)
		}
	}
	count(0)
	for _, typ := range []pub.ActivityVocabularyType{pub.CreateType, pub.UpdateType} {
		if err := federation.CreateSummitLogActivity(f.app, context.Background(), copy, typ); err != nil {
			t.Fatal(err)
		}
		count(0)
	}
	if err := federation.CreateSummitLogDeleteActivity(f.app, copy); err != nil {
		t.Fatal(err)
	}
	count(0)
	user, err := f.app.FindRecordById("users", contributor.GetString("user"))
	if err != nil {
		t.Fatal(err)
	}
	token, err := user.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	router, err := apis.NewRouter(f.app)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	request := func(method, suffix string, payload map[string]any) map[string]any {
		t.Helper()
		var body bytes.Buffer
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(method, "/api/collections/summit_logs/records"+suffix, &body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", token)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		if response.Code != http.StatusOK {
			t.Fatalf("entry request status=%d: %s", response.Code, response.Body.String())
		}
		if strings.Contains(response.Body.String(), "federation_suppressed") {
			t.Fatal("internal delivery state appeared in the response")
		}
		var result map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	request(http.MethodPatch, "/"+copy.Id, map[string]any{"text": "Edited contribution", "federation_suppressed": false})
	copy, err = f.app.FindRecordById("summit_logs", copy.Id)
	if err != nil || !copy.GetBool("federation_suppressed") || copy.GetString("text") != "Edited contribution" {
		t.Fatalf("entry update changed delivery state: record=%v error=%v", copy, err)
	}
	count(0)
	control := request(http.MethodPost, "", map[string]any{"text": "New contribution", "date": "2026-02-02 00:00:00.000Z", "author": contributor.Id, "trail": target.Id, "federation_suppressed": true})
	stored, err := f.app.FindRecordById("summit_logs", control["id"].(string))
	if err != nil || stored.GetBool("federation_suppressed") {
		t.Fatalf("entry create accepted request delivery state: record=%v error=%v", stored, err)
	}
	count(1)
	if err := federation.CreateSummitLogDeleteActivity(f.app, stored); err != nil {
		t.Fatal(err)
	}
	count(2)
}

func TestEntryDeliveryStateUsesRegisteredRequestHooks(t *testing.T) {
	f := newRecordHooksFixture(t)
	owner := f.localActor("owner")
	trail := f.save("trails", map[string]any{"name": "Target", "author": owner.Id})
	entry := f.save("summit_logs", map[string]any{"trail": trail.Id, "author": owner.Id, "text": "Stored entry", "federation_suppressed": true})
	admin := f.save(core.CollectionNameSuperusers, map[string]any{"email": "administrator@example.com"})
	token, err := admin.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	f.bindHooks()
	checked := 0
	stop := errors.New("request delivery state checked")
	inspect := func(e *core.RecordRequestEvent) error {
		checked++
		if !e.HasSuperuserAuth() {
			t.Fatal("request did not authenticate the administrator")
		}
		if e.Record.GetBool("federation_suppressed") != e.Record.Original().GetBool("federation_suppressed") {
			t.Error("registered request hooks accepted a change to internal delivery state")
		}
		// Observe the real request chain before its persistence callback. The
		// downstream summit-log hooks require an ordinary user-linked actor.
		return stop
	}
	f.app.OnRecordCreateRequest("summit_logs").Bind(&hook.Handler[*core.RecordRequestEvent]{Priority: 100, Func: inspect})
	f.app.OnRecordUpdateRequest("summit_logs").Bind(&hook.Handler[*core.RecordRequestEvent]{Priority: 100, Func: inspect})
	router, err := apis.NewRouter(f.app)
	if err != nil {
		t.Fatal(err)
	}
	handler, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []struct {
		method, suffix string
		marker         bool
	}{{http.MethodPost, "", true}, {http.MethodPatch, "/" + entry.Id, false}} {
		var body bytes.Buffer
		if err := json.NewEncoder(&body).Encode(map[string]any{"text": "Requested entry", "author": owner.Id, "trail": trail.Id, "date": "2026-02-01 00:00:00.000Z", "federation_suppressed": change.marker}); err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(change.method, "/api/collections/summit_logs/records"+change.suffix, &body)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", token)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("request did not stop at the delivery-state observer: %d %s", response.Code, response.Body.String())
		}
	}
	if checked != 2 {
		t.Fatalf("observed %d request hooks, want create and update", checked)
	}
}
