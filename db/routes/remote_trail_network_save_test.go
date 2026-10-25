package routes

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/types"
)

func TestRemoteTrailRefreshNetworkSaveWins(t *testing.T) {
	for _, change := range []string{"update", "delete"} {
		t.Run(change, func(t *testing.T) {
			t.Setenv("ORIGIN", "https://local.example")
			f := childRefreshFixture(t)
			f.trails.Fields.Add(&core.TextField{Name: "federated_category_name"}, &core.FileField{Name: "photos", MaxSelect: 10, MaxSize: 1024})
			if err := f.app.Save(f.trails); err != nil {
				t.Fatal(err)
			}
			parent := f.save(f.trails, map[string]any{"iri": "https://remote.example/api/v1/trail/copy", "name": "Stored trail", "public": false, "author": f.remoteActor.Id, "needs_full_sync": true})
			contributor := f.save(f.actors, map[string]any{"iri": "https://contributor.example/actor/riley", "last_fetched": time.Now()})
			logIRI := "https://contributor.example/api/v1/summit-log/entry"
			var changed atomic.Bool
			var confirmations atomic.Int32
			previousClient, previousMedia := newRemoteSyncHTTPClient, fetchRemoteSyncMedia
			t.Cleanup(func() { newRemoteSyncHTTPClient, fetchRemoteSyncMedia = previousClient, previousMedia })
			var prepared, released atomic.Int32
			fetchRemoteSyncMedia = func(ctx context.Context, raw, origin, name string, limit int64) (*filesystem.File, func(), error) {
				prepared.Add(1)
				file, err := filesystem.NewFileFromBytes([]byte("prepared image"), name)
				return file, func() { released.Add(1) }, err
			}
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: remoteSyncTransport{roundTrip: func(req *http.Request) (*http.Response, error) {
					if req.URL.Host == "contributor.example" {
						confirmations.Add(1)
						if changed.CompareAndSwap(false, true) {
							current, err := f.app.FindRecordById("trails", parent.Id)
							if err != nil {
								return nil, err
							}
							if change == "delete" {
								err = f.app.Delete(current)
							} else {
								current.Set("name", "Incoming update")
								current.Set("public", true)
								current.Set("needs_full_sync", true)
								current.Set("federated_category_name", "Incoming category")
								// Ensure distinct timestamps even within the same millisecond.
								updated, parseErr := types.ParseDateTime(current.GetDateTime("updated").Time().Add(time.Second))
								if parseErr != nil {
									return nil, parseErr
								}
								current.SetRaw("updated", updated)
								err = f.app.Save(current)
							}
							if err != nil {
								return nil, err
							}
						}
						return richTextSyncTransport{map[string]any{"id": "entry", "iri": logIRI, "text": "Confirmed entry", "expand": map[string]any{"author": map[string]any{"iri": contributor.GetString("iri")}, "trail": map[string]any{"iri": parent.GetString("iri")}}}}.RoundTrip(req)
					}
					return richTextSyncTransport{map[string]any{"id": "copy", "iri": parent.GetString("iri"), "name": "Document trail", "public": false, "photos": []any{"photo.png"}, "expand": map[string]any{
						"author":                map[string]any{"iri": f.remoteActor.GetString("iri")},
						"category":              map[string]any{"name": "Document category"},
						"waypoints_via_trail":   []any{map[string]any{"id": "point", "name": "Document waypoint"}},
						"summit_logs_via_trail": []any{map[string]any{"id": "entry", "iri": logIRI, "updated": "first"}},
					}}}.RoundTrip(req)
				}}}
			}
			got, err := performFullSync(f.app, t.Context(), nil, parent)
			if !changed.Load() || confirmations.Load() != 1 {
				t.Fatal("confirmation did not perform the concurrent change")
			}
			if prepared.Load() != 1 || released.Load() != 1 {
				t.Fatalf("prepared files not released: prepared=%d released=%d", prepared.Load(), released.Load())
			}
			for _, collection := range []string{"waypoints", "summit_logs"} {
				if count, findErr := f.app.CountRecords(collection); findErr != nil || count != 0 {
					t.Fatalf("older refresh wrote %s: count=%d error=%v", collection, count, findErr)
				}
			}
			if change == "delete" {
				if !errors.Is(err, errRemoteGone) {
					t.Fatalf("deleted trail returned %v", err)
				}
				if _, findErr := f.app.FindRecordById("trails", parent.Id); !errors.Is(findErr, sql.ErrNoRows) {
					t.Fatalf("deleted trail was written again: %v", findErr)
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
				if record.GetString("name") != "Incoming update" || !record.GetBool("public") || !record.GetBool("needs_full_sync") || record.GetString("federated_category_name") != "Incoming category" || record.GetBool("full_sync_completed") || len(record.GetStringSlice("photos")) != 0 {
					t.Fatalf("network-phase save was overwritten: %v", record)
				}
			}
			got, err = performFullSync(f.app, t.Context(), nil, stored)
			if err != nil {
				t.Fatal(err)
			}
			stored, err = f.app.FindRecordById("trails", parent.Id)
			if err != nil {
				t.Fatal(err)
			}
			for _, record := range []*core.Record{got, stored} {
				if record.GetString("name") != "Document trail" || record.GetBool("needs_full_sync") || !record.GetBool("full_sync_completed") {
					t.Fatalf("later refresh did not complete: %v", record)
				}
			}
			for _, collection := range []string{"waypoints", "summit_logs"} {
				if count, findErr := f.app.CountRecords(collection); findErr != nil || count != 1 {
					t.Fatalf("later refresh did not write %s: count=%d error=%v", collection, count, findErr)
				}
			}
			if prepared.Load() != 2 || released.Load() != 2 {
				t.Fatalf("later refresh files not released: prepared=%d released=%d", prepared.Load(), released.Load())
			}
		})
	}
}

func TestRemoteTrailRefreshBeforeRequestSaveDoesNotStopSync(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := childRefreshFixture(t)
	parent := f.save(f.trails, map[string]any{"iri": "https://remote.example/api/v1/trail/copy", "name": "Stored trail", "author": f.remoteActor.Id, "needs_full_sync": true})
	stale, err := f.app.FindRecordById("trails", parent.Id)
	if err != nil {
		t.Fatal(err)
	}
	parent.Set("name", "Before request")
	parent.Set("public", true)
	updated, err := types.ParseDateTime(parent.GetDateTime("updated").Time().Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	parent.SetRaw("updated", updated)
	if err := f.app.Save(parent); err != nil {
		t.Fatal(err)
	}
	refreshTransport(t, func(*http.Request) map[string]any {
		return map[string]any{"id": "copy", "iri": parent.GetString("iri"), "name": "Document trail", "public": true, "expand": map[string]any{"author": map[string]any{"iri": f.remoteActor.GetString("iri")}}}
	})
	got, err := performFullSync(f.app, t.Context(), nil, stale)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := f.app.FindRecordById("trails", parent.Id)
	if err != nil {
		t.Fatal(err)
	}
	for _, record := range []*core.Record{got, stored} {
		if record.GetString("name") != "Document trail" || record.GetBool("needs_full_sync") || !record.GetBool("full_sync_completed") {
			t.Fatalf("save before request stopped refresh: %v", record)
		}
	}
}
