package routes

import (
	"net/http"
	"slices"
	"testing"
)

func TestRemoteListTrailExpandsRequireRecordIDs(t *testing.T) {
	for _, id := range []any{nil, "", " \t "} {
		t.Run(map[bool]string{true: "missing", false: "blank"}[id == nil], func(t *testing.T) {
			t.Setenv("ORIGIN", "https://a.example")
			f := newRemoteSyncFixture(t)
			f.remoteActor.Set("iri", "https://b.example/api/v1/actor/alice")
			if err := f.app.Save(f.remoteActor); err != nil {
				t.Fatal(err)
			}
			child := f.save(f.trails, map[string]any{"iri": "https://b.example/api/v1/trail/retained", "name": "Stored trail", "author": f.remoteActor.Id})
			list := f.save(f.lists, map[string]any{"iri": "https://b.example/api/v1/list/copy", "author": f.remoteActor.Id, "trails": []string{child.Id}})
			payload := map[string]any{"id": "copy", "iri": list.GetString("iri"), "expand": map[string]any{
				"trails": []any{
					map[string]any{"id": id, "iri": child.GetString("iri"), "name": "Changed"},
					map[string]any{"id": id, "iri": "https://b.example/api/v1/trail/new", "name": "New trail"},
				},
			}}
			previous := newRemoteSyncHTTPClient
			t.Cleanup(func() { newRemoteSyncHTTPClient = previous })
			newRemoteSyncHTTPClient = func() *http.Client { return &http.Client{Transport: richTextSyncTransport{payload}} }
			if _, err := performFullListSync(f.app, t.Context(), nil, list); err != nil {
				t.Fatal(err)
			}
			stored, err := f.app.FindRecordById(f.lists, list.Id)
			if err != nil || !slices.Equal(stored.GetStringSlice("trails"), []string{child.Id}) {
				t.Fatalf("incomplete trail expands changed the list relations: %v %v", stored, err)
			}
			f.assertUnchanged(child, "name", "author", "iri")
			if record, _ := f.app.FindFirstRecordByData(f.trails, "iri", "https://b.example/api/v1/trail/new"); record != nil {
				t.Fatal("trail expand without a record id created a copy")
			}
		})
	}
}
