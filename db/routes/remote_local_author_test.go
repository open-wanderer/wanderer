package routes

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestRemoteSyncRetainsLocallyAuthoredRecords(t *testing.T) {
	for _, kind := range []string{"trail", "list"} {
		t.Run(kind, func(t *testing.T) {
			collection := "trails"
			handler := RemoteTrailGet
			if kind == "list" {
				collection = "lists"
				handler = RemoteListGet
			}
			f := setupGoneTest(t, http.StatusNotFound, collection, kind, handler)
			actors, err := f.app.FindCollectionByNameOrId("activitypub_actors")
			if err != nil {
				t.Fatal(err)
			}
			local := core.NewRecord(actors)
			local.Set("is_local", true)
			local.Set("iri", "https://older.example/actor/bob")
			if err := f.app.Save(local); err != nil {
				t.Fatal(err)
			}
			record := f.cached(t, true)
			record.Set("author", local.Id)
			record.Set("iri", "https://intermediate.example/api/v1/"+kind+"/"+record.Id)
			if err := f.app.Save(record); err != nil {
				t.Fatal(err)
			}
			var synced *core.Record
			if kind == "trail" {
				synced, err = performFullSync(f.app, context.Background(), &url.URL{}, record)
			} else {
				synced, err = performFullListSync(f.app, context.Background(), &url.URL{}, record)
			}
			if err != nil || synced.Id != record.Id {
				t.Fatalf("local historical object was fetched remotely: %v", err)
			}
			dropGoneRemoteRecord(f.app, collection, record.Id)
			if _, err = f.app.FindRecordById(collection, record.Id); err != nil {
				t.Fatalf("local historical object was deleted: %v", err)
			}
			e := &core.RequestEvent{}
			e.App = f.app
			var shell *core.Record
			if kind == "trail" {
				shell, err = findLocalTrailByRemoteInfo(e, context.Background(), "alice@remote.example", record.Id)
			} else {
				shell, err = findLocalListByRemoteInfo(e, context.Background(), "alice@remote.example", record.Id)
			}
			if err != nil {
				t.Fatal(err)
			}
			if shell.Id == record.Id {
				t.Fatal("remote handle selected a locally authored object through its ID")
			}
		})
	}
}
