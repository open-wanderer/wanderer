package routes

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func remoteTagFixture(t *testing.T, maxSelect int) (*remoteSyncFixture, *core.Collection) {
	t.Helper()
	f := newRemoteSyncFixture(t)
	tags := core.NewBaseCollection("tags")
	tags.Fields.Add(&core.TextField{Name: "name", Max: 5000})
	if err := f.app.Save(tags); err != nil {
		t.Fatal(err)
	}
	f.trails.Fields.Add(&core.RelationField{Name: "tags", CollectionId: tags.Id, MaxSelect: maxSelect})
	if err := f.app.Save(f.trails); err != nil {
		t.Fatal(err)
	}
	f.lists.Fields.GetByName("trails").(*core.RelationField).MaxSelect = 2000
	if err := f.app.Save(f.lists); err != nil {
		t.Fatal(err)
	}
	return f, tags
}

func remoteTagValues(prefix string, n int) []any {
	values := make([]any, n)
	for i := range values {
		values[i] = map[string]any{"name": fmt.Sprintf("%s_%04d", prefix, i)}
	}
	return values
}

func syncRemoteTagPayload(t *testing.T, f *remoteSyncFixture, kind string, record *core.Record, expand map[string]any) *core.Record {
	t.Helper()
	expand["author"] = map[string]any{"iri": f.remoteActor.GetString("iri")}
	payload := map[string]any{"id": "remote", "iri": record.GetString("iri"), "name": "Refreshed", "public": true, "expand": expand}
	previous := newRemoteSyncHTTPClient
	t.Cleanup(func() { newRemoteSyncHTTPClient = previous })
	newRemoteSyncHTTPClient = func() *http.Client { return &http.Client{Transport: richTextSyncTransport{payload}} }
	var got *core.Record
	var err error
	if kind == "trail" {
		got, err = performFullSync(f.app, t.Context(), nil, record)
	} else {
		got, err = performFullListSync(f.app, t.Context(), nil, record)
	}
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestRemoteSyncTagsShareCollectionBudgetAndRespectFieldLimit(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for _, kind := range []string{"trail", "list"} {
		t.Run(kind, func(t *testing.T) {
			maxSelect := 999
			if kind == "trail" {
				maxSelect = 3
			}
			f, _ := remoteTagFixture(t, maxSelect)
			col := f.trails
			if kind == "list" {
				col = f.lists
			}
			parent := f.save(col, map[string]any{"iri": "https://remote.example/api/v1/" + kind + "/remote", "author": f.remoteActor.Id})
			expand := map[string]any{"tags": remoteTagValues("top", 20)}
			if kind == "list" {
				children := make([]any, 10)
				for i := range children {
					children[i] = map[string]any{"id": fmt.Sprintf("child%d", i), "name": "Child", "public": true, "expand": map[string]any{"tags": remoteTagValues(fmt.Sprintf("child%d", i), 150)}}
				}
				expand = map[string]any{"trails": children}
			}
			writes := 0
			perTx := map[*core.TxAppInfo]int{}
			f.app.OnRecordCreateExecute("tags").BindFunc(func(e *core.RecordEvent) error {
				err := e.Next()
				if err == nil {
					writes++
					if tx := e.App.TxInfo(); tx != nil {
						perTx[tx]++
					}
				}
				return err
			})
			got := syncRemoteTagPayload(t, f, kind, parent, expand)
			want := maxSelect
			if kind == "list" {
				want = remoteSyncChildLimit
			}
			if n, err := f.app.CountRecords("tags"); err != nil || n != int64(want) || writes != want {
				t.Fatalf("tags=%d writes=%d want%d error%v", n, writes, want, err)
			}
			for _, n := range perTx {
				if n > remoteSyncWriteBatch {
					t.Fatalf("%d tag writes held one transaction", n)
				}
			}
			if kind == "trail" && len(got.GetStringSlice("tags")) != maxSelect {
				t.Fatal("field limit prevented valid metadata save")
			}
			if kind == "list" && len(got.GetStringSlice("trails")) != 10 {
				t.Fatal("tag budget truncated list entries")
			}
		})
	}
}

func TestRemoteSyncTagPreparationKeepsRepeatedNamesAndExistingRecords(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f, tags := remoteTagFixture(t, 999)
	first := f.save(tags, map[string]any{"id": "aaaaaaaaaaaaaaa", "name": "shared name"})
	duplicate := f.save(tags, map[string]any{"id": "zzzzzzzzzzzzzzz", "name": "shared name"})
	list := f.save(f.lists, map[string]any{"iri": "https://remote.example/api/v1/list/remote", "author": f.remoteActor.Id})
	children := make([]any, 200)
	for i := range children {
		values := remoteTagValues("common", 4)
		values = append(values, map[string]any{"name": "shared\tname"}, map[string]any{"name": "shared name"})
		children[i] = map[string]any{"id": fmt.Sprintf("child%d", i), "name": "Child", "public": true, "expand": map[string]any{"tags": values}}
	}
	got := syncRemoteTagPayload(t, f, "list", list, map[string]any{"trails": children})
	if len(got.GetStringSlice("trails")) != 200 {
		t.Fatal("ordinary list did not finish")
	}
	if n, err := f.app.CountRecords("tags"); err != nil || n != 6 {
		t.Fatalf("repeated names consumed collection budget: count=%d err=%v", n, err)
	}
	for _, id := range got.GetStringSlice("trails") {
		trail, err := f.app.FindRecordById("trails", id)
		if err != nil || len(trail.GetStringSlice("tags")) != 5 || !slices.Contains(trail.GetStringSlice("tags"), first.Id) {
			t.Fatalf("shared tags missing on %s: %v", id, err)
		}
	}
	f.assertUnchanged(first, "name")
	f.assertUnchanged(duplicate, "name")
}

func TestRemoteSyncTagBudgetRetainsExistingRelationsOnLaterTrails(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f, tags := remoteTagFixture(t, 999)
	tag := f.save(tags, map[string]any{"name": "Retained"})
	later := f.save(f.trails, map[string]any{"iri": "https://remote.example/api/v1/trail/later", "name": "Cached", "author": f.remoteActor.Id, "tags": []string{tag.Id}})
	list := f.save(f.lists, map[string]any{"iri": "https://remote.example/api/v1/list/remote", "author": f.remoteActor.Id, "trails": []string{later.Id}})
	children := []any{}
	for i := 0; i < 5; i++ {
		children = append(children, map[string]any{"id": fmt.Sprintf("child%d", i), "name": "Child", "expand": map[string]any{"tags": remoteTagValues(fmt.Sprintf("child%d", i), 200)}})
	}
	children = append(children, map[string]any{"id": "later", "name": "Refreshed later", "expand": map[string]any{"tags": []any{map[string]any{"name": "Retained"}, map[string]any{"name": "Omitted new label"}}}})
	syncRemoteTagPayload(t, f, "list", list, map[string]any{"trails": children})
	stored, err := f.app.FindRecordById("trails", later.Id)
	if err != nil || !slices.Equal(stored.GetStringSlice("tags"), []string{tag.Id}) || stored.GetString("name") != "Refreshed later" {
		t.Fatalf("later metadata lost saved tags: %v %v", stored, err)
	}
	if n, err := f.app.CountRecords("tags"); err != nil || n != remoteSyncChildLimit+1 {
		t.Fatalf("unexpected new tags beyond collection budget: %d %v", n, err)
	}
}

func TestRemoteSyncDoesNotPrepareTagsForExcludedTrails(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f, tags := remoteTagFixture(t, 999)
	existing := f.save(tags, map[string]any{"name": "Saved"})
	local := f.save(f.trails, map[string]any{"iri": "https://local.example/api/v1/trail/local", "name": "Local", "author": f.localActor.Id, "public": true, "tags": []string{existing.Id}})
	historical := f.save(f.trails, map[string]any{"iri": "https://remote.example/api/v1/trail/historical", "name": "Historical", "author": f.localActor.Id, "public": true, "tags": []string{existing.Id}})
	other := f.save(f.actors, map[string]any{"iri": "https://remote.example/api/v1/actor/other"})
	unchanged := f.save(f.trails, map[string]any{"iri": "https://remote.example/api/v1/trail/unchanged", "name": "Unchanged", "author": other.Id, "public": true, "tags": []string{existing.Id}})
	list := f.save(f.lists, map[string]any{"iri": "https://remote.example/api/v1/list/remote", "author": f.remoteActor.Id})
	children := []any{}
	for _, id := range []string{"local", "historical", "unchanged", "foreign", "unconfirmed", "allowed"} {
		iri := "https://remote.example/api/v1/trail/" + id
		if id == "local" {
			iri = local.GetString("iri")
		}
		if id == "foreign" {
			iri = "https://other.example/api/v1/trail/foreign"
		}
		expand := map[string]any{"tags": []any{map[string]any{"name": id}}}
		if id == "unconfirmed" {
			expand["author"] = map[string]any{"iri": "https://remote.example/api/v1/actor/unavailable"}
		}
		children = append(children, map[string]any{"id": id, "iri": iri, "name": "Remote name", "public": true, "expand": expand})
	}
	previous := remoteSyncGetActorByIRI
	t.Cleanup(func() { remoteSyncGetActorByIRI = previous })
	remoteSyncGetActorByIRI = func(core.App, context.Context, string, bool) (*core.Record, error) {
		return nil, fmt.Errorf("origin unavailable")
	}
	syncRemoteTagPayload(t, f, "list", list, map[string]any{"trails": children})
	if n, err := f.app.CountRecords("tags"); err != nil || n != 2 {
		t.Fatalf("excluded metadata created tags: %d %v", n, err)
	}
	for _, record := range []*core.Record{local, historical, unchanged} {
		f.assertUnchanged(record, "name", "tags", "author", "iri")
	}
}

func TestRemoteSyncTagPreparationUsesTheConfirmedListAuthor(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f, _ := remoteTagFixture(t, 999)
	previousAuthor := f.save(f.actors, map[string]any{"iri": "https://previous.example/api/v1/actor/riley"})
	list := f.save(f.lists, map[string]any{"iri": "https://remote.example/api/v1/list/remote", "author": previousAuthor.Id})
	previous := remoteSyncGetActorByIRI
	t.Cleanup(func() { remoteSyncGetActorByIRI = previous })
	remoteSyncGetActorByIRI = func(core.App, context.Context, string, bool) (*core.Record, error) { return f.remoteActor, nil }
	got := syncRemoteTagPayload(t, f, "list", list, map[string]any{"trails": []any{map[string]any{"id": "child", "name": "Child", "expand": map[string]any{"tags": remoteTagValues("confirmed", 1)}}}})
	ids := got.GetStringSlice("trails")
	if len(ids) != 1 {
		t.Fatalf("confirmed author lost child: %v", ids)
	}
	trail, err := f.app.FindRecordById("trails", ids[0])
	if err != nil || len(trail.GetStringSlice("tags")) != 1 || trail.GetString("author") != f.remoteActor.Id {
		t.Fatalf("tag preparation used previous author: %v %v", trail, err)
	}
}
