package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"pocketbase/util"
)

func TestFullRemoteSyncAcceptsLargeTrailAndListDocuments(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for _, kind := range []string{"trail", "list"} {
		t.Run(kind, func(t *testing.T) {
			f := newRemoteSyncFixture(t)
			payload := map[string]any{"id": kind, "iri": "https://remote.example/api/v1/" + kind + "/" + kind, "name": "Large synced object", "public": true}
			expand := map[string]any{}
			count := 1000
			if kind == "trail" {
				logs := make([]any, count)
				for i := range logs {
					logs[i] = map[string]any{"id": fmt.Sprintf("entry-%d", i), "text": strings.Repeat("a", 2200)}
				}
				expand["summit_logs_via_trail"] = logs
			} else {
				count = 1000
				field := f.lists.Fields.GetByName("trails").(*core.RelationField)
				field.MaxSelect = count
				f.lists.Fields.Add(&core.TextField{Name: "iri"}, &core.TextField{Name: "name"}, &core.BoolField{Name: "public"}, &core.BoolField{Name: "needs_full_sync"}, &core.BoolField{Name: "full_sync_completed"})
				if err := f.app.Save(f.lists); err != nil {
					t.Fatal(err)
				}
				trails := make([]any, count)
				for i := range trails {
					trails[i] = map[string]any{"id": fmt.Sprintf("trail-%d", i), "name": "Remote trail", "public": true, "description": strings.Repeat("a", 2200)}
				}
				expand["trails"] = trails
			}
			payload["expand"] = expand
			wire, err := json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			if int64(len(wire)) <= util.FederationObjectBodyLimit || int64(len(wire)) > util.FederationSyncBodyLimit {
				t.Fatalf("fixture size=%d outside intended realistic response range", len(wire))
			}
			original := newRemoteSyncHTTPClient
			t.Cleanup(func() { newRemoteSyncHTTPClient = original })
			newRemoteSyncHTTPClient = func() *http.Client { return &http.Client{Transport: richTextSyncTransport{payload}} }
			collection := f.trails
			if kind == "list" {
				collection = f.lists
			}
			local := f.save(collection, map[string]any{"iri": payload["iri"], "author": f.remoteActor.Id, "name": "Original", "needs_full_sync": true})
			if kind == "trail" {
				_, err = performFullSync(f.app, t.Context(), nil, local)
			} else {
				_, err = performFullListSync(f.app, t.Context(), nil, local)
			}
			if err != nil {
				t.Fatalf("realistic %d-byte %s failed full sync: %v", len(wire), kind, err)
			}
			stored, err := f.app.FindRecordById(collection, local.Id)
			if err != nil || !stored.GetBool("full_sync_completed") || stored.GetBool("needs_full_sync") {
				t.Fatalf("sync flags: %v error=%v", stored, err)
			}
			if kind == "trail" {
				got, err := f.app.CountRecords("summit_logs")
				if err != nil || got != int64(count) {
					t.Fatalf("synced entry count=%d error=%v", got, err)
				}
			} else if len(stored.GetStringSlice("trails")) != count {
				t.Fatalf("synced trail references=%d", len(stored.GetStringSlice("trails")))
			}
		})
	}
}

func TestOversizeRemoteSyncServesOnlyCompletedCurrentlyReadableCopies(t *testing.T) {
	for _, collection := range []string{"trails", "lists"} {
		for _, completed := range []bool{true, false} {
			for _, public := range []bool{true, false} {
				t.Run(fmt.Sprintf("%s/completed%t/public%t", collection, completed, public), func(t *testing.T) {
					f := setupHandleLookupTest(t, "https://local.example", http.StatusOK)
					actor := f.remoteActor(t, "alice", "remote.example")
					copy := f.object(t, collection, actor.Id, "https://remote.example", "remoteid0000001", true, false)
					copy.Set("full_sync_completed", completed)
					copy.Set("public", public)
					if err := f.app.Save(copy); err != nil {
						t.Fatal(err)
					}
					f.transport.body = `{"id":"remoteid0000001","name":"Unconfirmed"}` + strings.Repeat(" ", int(util.FederationSyncBodyLimit))
					status, body := f.get(t, collection, copy.Id, "@alice@remote.example")
					want := http.StatusUnprocessableEntity
					if completed {
						want = http.StatusOK
						if !public {
							want = http.StatusForbidden
						}
					}
					if status != want || strings.Contains(body, "Unconfirmed") {
						t.Fatalf("oversize sync status=%d want=%d body=%s", status, want, body)
					}
					stored, err := f.app.FindRecordById(collection, copy.Id)
					if err != nil || stored.GetString("name") != "object" || !stored.GetBool("needs_full_sync") {
						t.Fatalf("failed sync rewrote cache: %v %v", stored, err)
					}
				})
			}
		}
	}
}
