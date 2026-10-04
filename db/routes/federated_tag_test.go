package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"pocketbase/tagname"
	"pocketbase/util"
	"slices"
	"strings"
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

const (
	federatedTagFirstID = "aaaaaaaaaaaaaaa"
	federatedTagOtherID = "zzzzzzzzzzzzzzz"
	federatedTagLiteral = "🥾 👩‍👩‍👧‍👦 <b>[.*]</b> &amp; ' || id != ''"
)

func setupFederatedTagImport(t *testing.T) (*pbtests.TestApp, *core.Collection, *core.Record) {
	t.Helper()
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	save := func(model core.Model) {
		t.Helper()
		if err := app.Save(model); err != nil {
			t.Fatal(err)
		}
	}
	tags := core.NewBaseCollection("tags")
	tags.Fields.Add(&core.TextField{Name: "name", Max: tagname.MaxLength, Pattern: tagname.Pattern})
	save(tags)
	for _, id := range []string{federatedTagOtherID, federatedTagFirstID} {
		record := core.NewRecord(tags)
		record.Id = id
		record.Set("name", "ab")
		save(record)
	}
	empty := core.NewRecord(tags)
	save(empty)
	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(&core.BoolField{Name: "is_local"})
	save(actors)
	actor := core.NewRecord(actors)
	save(actor)
	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(
		&core.TextField{Name: "name"},
		&core.TextField{Name: "iri"},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
		&core.RelationField{Name: "tags", CollectionId: tags.Id, MaxSelect: 100},
		&core.BoolField{Name: "needs_full_sync"},
		&core.BoolField{Name: "full_sync_completed"},
	)
	save(trails)
	// Relations to duplicate and empty historical IDs must remain untouched.
	historical := core.NewRecord(trails)
	historical.Set("tags", []string{federatedTagOtherID, empty.Id})
	save(historical)
	t.Cleanup(func() {
		reloaded, err := app.FindRecordById("trails", historical.Id)
		if err != nil || !slices.Equal(reloaded.GetStringSlice("tags"), []string{federatedTagOtherID, empty.Id}) {
			t.Errorf("historical relations changed: %v, %v", reloaded, err)
		}
		for _, id := range []string{federatedTagFirstID, federatedTagOtherID, empty.Id} {
			if _, err := app.FindRecordById("tags", id); err != nil {
				t.Errorf("historical tag %s removed: %v", id, err)
			}
		}
	})
	return app, trails, actor
}

func assertFederatedTagNames(t *testing.T, app core.App, trailID string, want []string) {
	t.Helper()
	trail, err := app.FindRecordById("trails", trailID)
	if err != nil {
		t.Fatal(err)
	}
	ids := trail.GetStringSlice("tags")
	if len(ids) != len(want) {
		t.Fatalf("tags = %v, want names %v", ids, want)
	}
	for i, id := range ids {
		tag, err := app.FindRecordById("tags", id)
		if err != nil || tag.GetString("name") != want[i] {
			t.Fatalf("tag %d = %v, %v; want %q", i, tag, err, want[i])
		}
		if want[i] == "ab" && id != federatedTagFirstID {
			t.Fatalf("duplicate-name selection = %s, want smallest ID", id)
		}
	}
}

func TestActivityPubTrailImportNormalizesTags(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for i, tt := range []struct {
		name string
		tags []string
		want []string
	}{
		{"controls collisions and literal text", []string{"a\tb", "ab", "\n\t\x7f", "\n" + federatedTagLiteral + "\x00"}, []string{"ab", federatedTagLiteral}},
		{"control-only is omitted", []string{"\n\t\x7f", ""}, nil},
		{"failed tag does not abort import", []string{strings.Repeat("a", tagname.MaxLength+1), "a\tb"}, []string{"ab"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app, _, actor := setupFederatedTagImport(t)
			object := pub.ObjectNew(pub.NoteType)
			object.ID = pub.IRI(fmt.Sprintf("https://remote.example/api/v1/trail/tag%d", i))
			object.Name = pub.DefaultNaturalLanguageValue("Remote trail")
			object.Location = &pub.Place{Name: pub.DefaultNaturalLanguageValue("Somewhere")}
			object.StartTime = time.Now()
			for _, name := range tt.tags {
				tag := pub.ObjectNew(pub.ObjectType)
				tag.Name = pub.DefaultNaturalLanguageValue("tag")
				tag.Content = pub.DefaultNaturalLanguageValue(name)
				object.Tag = append(object.Tag, tag)
			}
			trail, err := util.TrailFromActivity(pub.Activity{Type: pub.CreateType, Object: object}, app, actor)
			if err != nil {
				t.Fatal(err)
			}
			assertFederatedTagNames(t, app, trail.Id, tt.want)
		})
	}
}

type federatedTagTransport []byte

func (body federatedTagTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(body))}, nil
}

func TestRemoteTrailFullSyncNormalizesTags(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	overlong := strings.Repeat("a", tagname.MaxLength+1)
	for _, tt := range []struct {
		name string
		tags any
		want []string
		keep bool
	}{
		{"controls collisions and literal text", []any{map[string]any{"name": "a\tb"}, map[string]any{"name": "ab"}, map[string]any{"name": "\n\t\x7f"}, map[string]any{"name": "\n" + federatedTagLiteral}}, []string{"ab", federatedTagLiteral}, false},
		{"control-only clears previous relations", []any{map[string]any{"name": "\n\t\x7f"}}, nil, false},
		{"explicit empty list clears previous relations", []any{}, nil, false},
		{"missing tags retain previous relations", nil, []string{"ab"}, true},
		{"malformed list retains previous relations", "not-a-list", []string{"ab"}, true},
		{"malformed items retain previous relations", []any{nil, map[string]any{"name": 42}}, []string{"ab"}, true},
		{"all failed saves retain previous relations", []any{map[string]any{"name": overlong}}, []string{"ab"}, true},
		{"empty name and failed save retain previous relations", []any{map[string]any{"name": "\n\t\x7f"}, map[string]any{"name": overlong}}, []string{"ab"}, true},
		{"empty name and malformed item retain previous relations", []any{map[string]any{"name": "\n\t\x7f"}, map[string]any{"name": 42}}, []string{"ab"}, true},
		{"partial success remains tolerant", []any{map[string]any{"name": overlong}, map[string]any{"name": "a\tb"}}, []string{"ab"}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app, collection, actor := setupFederatedTagImport(t)
			trail := core.NewRecord(collection)
			trail.Set("iri", "https://remote.example/api/v1/trail/tags")
			trail.Set("author", actor.Id)
			trail.Set("tags", []string{federatedTagOtherID})
			if err := app.Save(trail); err != nil {
				t.Fatal(err)
			}
			remote := map[string]any{"id": "remotetrail00001", "name": "Updated trail"}
			if tt.tags != nil {
				remote["expand"] = map[string]any{"tags": tt.tags}
			}
			body, err := json.Marshal(remote)
			if err != nil {
				t.Fatal(err)
			}
			previousClient := newRemoteSyncHTTPClient
			newRemoteSyncHTTPClient = func() *http.Client { return &http.Client{Transport: federatedTagTransport(body)} }
			t.Cleanup(func() { newRemoteSyncHTTPClient = previousClient })
			requestURL, _ := url.Parse("https://local.example/api/v1/trail/tags?expand=tags")
			if _, err := performFullSync(app, context.Background(), requestURL, trail); err != nil {
				t.Fatal(err)
			}
			if tt.keep {
				reloaded, err := app.FindRecordById("trails", trail.Id)
				if err != nil || !slices.Equal(reloaded.GetStringSlice("tags"), []string{federatedTagOtherID}) {
					t.Fatalf("previous relation was not retained: %v, %v", reloaded, err)
				}
			} else {
				assertFederatedTagNames(t, app, trail.Id, tt.want)
			}
		})
	}
}
