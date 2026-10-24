package routes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func TestRemoteTrailCompletionPreservesConcurrentSaves(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for _, change := range []string{"update", "likes", "none", "delete"} {
		t.Run(change, func(t *testing.T) {
			f := childRefreshFixture(t)
			f.trails.Fields.Add(&core.NumberField{Name: "like_count"})
			if err := f.app.Save(f.trails); err != nil {
				t.Fatal(err)
			}
			parent := f.save(f.trails, map[string]any{"iri": "https://remote.example/api/v1/trail/copy", "name": "Stored trail", "public": false, "author": f.remoteActor.Id, "needs_full_sync": true})
			children := make([]any, remoteSyncWriteBatch+1)
			for i := range children {
				children[i] = map[string]any{"id": fmt.Sprintf("child%d", i), "name": "Refreshed child"}
			}
			refreshTransport(t, func(*http.Request) map[string]any {
				return map[string]any{"id": "copy", "iri": parent.GetString("iri"), "name": "Refreshed trail", "public": false, "expand": map[string]any{"author": map[string]any{"iri": f.remoteActor.GetString("iri")}, "waypoints_via_trail": children}}
			})
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			batchCommitted := make(chan struct{})
			writerDone := make(chan error, 1)
			var entered atomic.Bool
			f.app.OnRecordAfterCreateSuccess("waypoints").BindFunc(func(e *core.RecordEvent) error {
				if entered.CompareAndSwap(false, true) {
					close(batchCommitted)
					select {
					case err := <-writerDone:
						if err != nil {
							return err
						}
					case <-ctx.Done():
						return ctx.Err()
					}
				}
				return e.Next()
			})
			go func() {
				select {
				case <-batchCommitted:
				case <-ctx.Done():
					writerDone <- ctx.Err()
					return
				}
				current, err := f.app.FindRecordById("trails", parent.Id)
				if err == nil {
					switch change {
					case "update":
						current.Set("public", true)
						current.Set("needs_full_sync", true)
						// Make the incoming update's timestamp distinct even if
						// both commits fall in the same millisecond.
						updated, parseErr := types.ParseDateTime(current.GetDateTime("updated").Time().Add(time.Second))
						if parseErr != nil {
							err = parseErr
							break
						}
						current.SetRaw("updated", updated)
						err = f.app.Save(current)
					case "likes":
						current.Set("like_count", 7)
						err = f.app.Save(current)
					case "delete":
						err = f.app.Delete(current)
					}
				}
				writerDone <- err
			}()
			got, err := performFullSync(f.app, ctx, nil, parent)
			if !entered.Load() {
				t.Fatal("child batch was not committed")
			}
			if change == "delete" {
				if !errors.Is(err, errRemoteGone) {
					t.Fatalf("deleted trail returned %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			stored, err := f.app.FindRecordById("trails", parent.Id)
			if err != nil {
				t.Fatal(err)
			}
			for _, record := range []*core.Record{got, stored} {
				if !record.GetBool("full_sync_completed") || record.GetString("name") != "Refreshed trail" {
					t.Fatalf("refresh did not complete: %v", record)
				}
				switch change {
				case "update":
					if !record.GetBool("public") || !record.GetBool("needs_full_sync") {
						t.Fatalf("incoming update lost: public=%t needs_full_sync=%t", record.GetBool("public"), record.GetBool("needs_full_sync"))
					}
				case "likes":
					if record.GetInt("like_count") != 7 {
						t.Fatalf("like recount lost: %d", record.GetInt("like_count"))
					}
				case "none":
					if record.GetBool("needs_full_sync") {
						t.Fatal("unchanged trail still needs a full sync")
					}
				}
			}
			if count, err := f.app.CountRecords("waypoints"); err != nil || count != int64(len(children)) {
				t.Fatalf("child batches did not finish: count=%d error=%v", count, err)
			}
		})
	}
}

func TestRemoteTrailCompletionKeepsSaveErrors(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := childRefreshFixture(t)
	parent := f.save(f.trails, map[string]any{"iri": "https://remote.example/api/v1/trail/copy", "author": f.remoteActor.Id, "needs_full_sync": true})
	refreshTransport(t, func(*http.Request) map[string]any {
		return map[string]any{"id": "copy", "iri": parent.GetString("iri"), "expand": map[string]any{"author": map[string]any{"iri": f.remoteActor.GetString("iri")}}}
	})
	f.app.OnRecordUpdateExecute("trails").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetBool("full_sync_completed") && !e.Record.GetBool("needs_full_sync") {
			return sql.ErrNoRows
		}
		return e.Next()
	})
	_, err := performFullSync(f.app, t.Context(), nil, parent)
	if !errors.Is(err, sql.ErrNoRows) || errors.Is(err, errRemoteGone) {
		t.Fatalf("save error changed: %v", err)
	}
	if _, err := f.app.FindRecordById("trails", parent.Id); err != nil {
		t.Fatalf("stored trail disappeared: %v", err)
	}
}
