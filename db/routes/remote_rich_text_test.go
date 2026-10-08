package routes

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"pocketbase/util"
)

type richTextSyncTransport struct{ payload map[string]any }

func (transport richTextSyncTransport) RoundTrip(*http.Request) (*http.Response, error) {
	data, err := json.Marshal(transport.payload)
	if err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(string(data))),
	}, nil
}

func TestRemoteSyncExplicitlyBoundsRichText(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	app.OnRecordValidate().BindFunc(util.SanitizeHTML())
	save := func(model core.Model) {
		t.Helper()
		if err := app.Save(model); err != nil {
			t.Fatal(err)
		}
	}
	collection := func(name, textField string, limit int) *core.Collection {
		t.Helper()
		c := core.NewBaseCollection(name)
		c.Fields.Add(
			&core.TextField{Name: textField, Max: limit},
			&core.TextField{Name: "name"},
			&core.TextField{Name: "iri"},
			&core.TextField{Name: "author"},
			&core.TextField{Name: "trail"},
			&core.BoolField{Name: "needs_full_sync"},
			&core.BoolField{Name: "full_sync_completed"},
		)
		save(c)
		return c
	}
	trails := collection("trails", "description", 10000)
	lists := collection("lists", "description", 0)
	lists.Fields.Add(&core.RelationField{Name: "trails", CollectionId: trails.Id, MaxSelect: 10})
	save(lists)
	collection("waypoints", "description", 0)
	collection("summit_logs", "text", 0)

	longText := `<p onclick="blocked()"><strong>Safe ` + strings.Repeat(`'"&山🚲`, 4000) + `</strong></p>`
	assertStored := func(collection, id, field string, limit int) {
		t.Helper()
		stored, err := app.FindRecordById(collection, id)
		if err != nil {
			t.Fatal(err)
		}
		value := stored.GetString(field)
		if utf8.RuneCountInString(value) > limit || strings.Contains(value, "blocked()") || !strings.HasPrefix(value, "<p><strong>Safe ") || !strings.HasSuffix(value, "</strong></p>") {
			t.Fatalf("%s.%s was not safely bounded: length=%d, prefix=%.40s", collection, field, utf8.RuneCountInString(value), value)
		}
		if err := app.Validate(stored); err != nil {
			t.Fatalf("%s failed field validation after import: %v", collection, err)
		}
	}

	oldClient := newRemoteSyncHTTPClient
	t.Cleanup(func() { newRemoteSyncHTTPClient = oldClient })
	setPayload := func(payload map[string]any) {
		newRemoteSyncHTTPClient = func() *http.Client {
			return &http.Client{Transport: richTextSyncTransport{payload}}
		}
	}
	requestURL, err := url.Parse("https://local.example/api/v1/remote?expand=waypoints_via_trail,summit_logs_via_trail")
	if err != nil {
		t.Fatal(err)
	}

	trail := core.NewRecord(trails)
	trail.Set("author", "remote-actor")
	trail.Set("iri", "https://remote.example/api/v1/trail/remote-trail")
	save(trail)
	setPayload(map[string]any{
		"id": "remote-trail", "name": "Remote trail", "description": longText,
		"expand": map[string]any{
			"waypoints_via_trail":   []any{map[string]any{"id": "remote-waypoint", "description": longText}},
			"summit_logs_via_trail": []any{map[string]any{"id": "remote-log", "text": longText}},
		},
	})
	if _, err := performFullSync(app, context.Background(), requestURL, trail); err != nil {
		t.Fatalf("full trail sync rejected imported rich text: %v", err)
	}
	assertStored("trails", trail.Id, "description", 10000)
	for _, entry := range []struct{ collection, field string }{{"waypoints", "description"}, {"summit_logs", "text"}} {
		records, err := app.FindAllRecords(entry.collection)
		if err != nil || len(records) != 1 {
			t.Fatalf("%s not fully imported: count=%d error=%v", entry.collection, len(records), err)
		}
		assertStored(entry.collection, records[0].Id, entry.field, 5000)
	}

	list := core.NewRecord(lists)
	list.Set("author", "remote-actor")
	list.Set("iri", "https://remote.example/api/v1/list/remote-list")
	save(list)
	setPayload(map[string]any{
		"id": "remote-list", "description": longText,
		"expand": map[string]any{
			"trails": []any{map[string]any{"id": "list-trail", "description": longText}},
		},
	})
	if _, err := performFullListSync(app, context.Background(), requestURL, list); err != nil {
		t.Fatalf("full list sync rejected imported rich text: %v", err)
	}
	assertStored("lists", list.Id, "description", 5000)
	storedList, err := app.FindRecordById(lists, list.Id)
	if err != nil {
		t.Fatal(err)
	}
	ids := storedList.GetStringSlice("trails")
	if len(ids) != 1 {
		t.Fatalf("list trail missing after import: %v", ids)
	}
	assertStored("trails", ids[0], "description", 10000)
}
