package routes

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func TestRemoteChildAuthorsPersistBeforeTransactionAndDoNotBlockOtherContributions(t *testing.T) {
	for _, kind := range []string{"trail", "list"} {
		for _, scenario := range []string{"new", "wiped", "unavailable", "store-failed"} {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				t.Setenv("ORIGIN", "https://local.example")
				f := newRemoteSyncFixture(t)
				count := 11
				if scenario == "wiped" {
					count = 29
				}
				if scenario == "store-failed" {
					count = 2
				}
				collection, childCollection := f.trails, f.logs
				relation := "summit_logs_via_trail"
				if kind == "list" {
					collection, childCollection, relation = f.lists, f.trails, "trails"
					collection.Fields.Add(&core.TextField{Name: "iri"}, &core.TextField{Name: "name"}, &core.BoolField{Name: "needs_full_sync"}, &core.BoolField{Name: "full_sync_completed"})
					if err := f.app.Save(collection); err != nil {
						t.Fatal(err)
					}
				}
				local := f.save(collection, map[string]any{"iri": "https://remote.example/api/v1/" + kind + "/parent", "author": f.remoteActor.Id, "name": "Original", "needs_full_sync": true})
				children := []any{}
				for i := range count {
					authorIRI := fmt.Sprintf("https://remote.example/users/contributor%d", i)
					if scenario == "wiped" {
						f.save(f.actors, map[string]any{"iri": authorIRI, "public_key": "", "last_fetched": time.Time{}})
					}
					children = append(children, map[string]any{"id": fmt.Sprintf("item%d", i), "name": "Child", "text": "Child", "public": true, "expand": map[string]any{"author": map[string]any{"iri": authorIRI}}})
				}
				if scenario == "store-failed" {
					f.app.OnRecordCreate(childCollection.Name).BindFunc(func(*core.RecordEvent) error { return errors.New("child storage unavailable") })
				}
				payload := map[string]any{"id": "parent", "name": "Synced", "public": true, "expand": map[string]any{relation: children}}
				originalClient, originalActor := newRemoteSyncHTTPClient, remoteSyncGetActorByIRI
				t.Cleanup(func() { newRemoteSyncHTTPClient, remoteSyncGetActorByIRI = originalClient, originalActor })
				newRemoteSyncHTTPClient = func() *http.Client { return &http.Client{Transport: richTextSyncTransport{payload}} }
				var lookups, inTransaction atomic.Int32
				remoteSyncGetActorByIRI = func(app core.App, ctx context.Context, iri string, includeFollows bool) (*core.Record, error) {
					lookups.Add(1)
					if app.TxInfo() != nil {
						inTransaction.Add(1)
						return nil, errors.New("actor lookup inside record transaction")
					}
					if includeFollows {
						t.Error("child identity lookup requested follower collections")
					}
					if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > sameOriginAuthorTimeout {
						t.Error("child author request has no bounded deadline")
					}
					if scenario == "unavailable" && strings.HasSuffix(iri, "contributor10") {
						return nil, errors.New("actor origin unavailable")
					}
					actor, err := app.FindFirstRecordByData("activitypub_actors", "iri", iri)
					if err != nil {
						actor = core.NewRecord(f.actors)
					}
					actor.Load(map[string]any{"iri": iri, "domain": "remote.example", "public_key": "confirmed-key", "last_fetched": time.Now()})
					if err := app.Save(actor); err != nil {
						return nil, err
					}
					return actor, nil
				}
				var err error
				if kind == "trail" {
					_, err = performFullSync(f.app, t.Context(), nil, local)
				} else {
					_, err = performFullListSync(f.app, t.Context(), nil, local)
				}
				if lookups.Load() != int32(count) || inTransaction.Load() != 0 {
					t.Fatalf("child identity lookup count=%d inTX=%d, want%d/0", lookups.Load(), inTransaction.Load(), count)
				}
				want := count
				if scenario == "unavailable" {
					want--
				}
				actors, lookupErr := f.app.FindRecordsByFilter("activitypub_actors", "public_key='confirmed-key'", "", -1, 0)
				if lookupErr != nil || len(actors) != want {
					t.Fatalf("confirmed keys lost: count%d want%d error%v", len(actors), want, lookupErr)
				}
				if scenario == "store-failed" {
					if err == nil {
						t.Fatal("fixture did not roll back record import")
					}
					if _, err := f.app.FindFirstRecordByData(childCollection, "text", "Child"); err == nil {
						t.Fatal("failed record transaction left imported children")
					}
					return
				}
				if err != nil {
					t.Fatalf("one unavailable contributor aborted remaining sync: %v", err)
				}
				stored, err := f.app.FindRecordById(collection, local.Id)
				if err != nil || !stored.GetBool("full_sync_completed") {
					t.Fatalf("sync did not complete: %v %v", stored, err)
				}
				if kind == "list" {
					if len(stored.GetStringSlice("trails")) != want {
						t.Fatalf("synced relations=%d want%d", len(stored.GetStringSlice("trails")), want)
					}
				} else {
					got, err := f.app.CountRecords(childCollection.Name)
					if err != nil || got != int64(want) {
						t.Fatalf("synced contribution count=%d want%d error=%v", got, want, err)
					}
				}
			})
		}
	}
}
