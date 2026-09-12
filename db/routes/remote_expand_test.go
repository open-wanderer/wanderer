package routes

import (
	"encoding/json"
	"net/http/httptest"
	"slices"
	"sort"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/types"
)

// expandAndReturn must only expand related records the caller is allowed to
// view, exactly like the records API does. Otherwise a public list would hand
// out the private trails it references to anyone who requests ?expand=trails.
func TestExpandAndReturnAppliesViewRulesOfRelatedCollection(t *testing.T) {
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	users := core.NewAuthCollection("xt_viewers")
	users.ViewRule = types.Pointer("")
	if err := app.Save(users); err != nil {
		t.Fatal(err)
	}

	items := core.NewBaseCollection("xt_items")
	items.Fields.Add(
		&core.TextField{Name: "name"},
		&core.BoolField{Name: "public"},
		&core.RelationField{Name: "author", CollectionId: users.Id, MaxSelect: 1},
	)
	items.ViewRule = types.Pointer("public = true || author = @request.auth.id")
	items.ListRule = items.ViewRule
	if err := app.Save(items); err != nil {
		t.Fatal(err)
	}

	bundles := core.NewBaseCollection("xt_bundles")
	bundles.Fields.Add(
		&core.RelationField{Name: "items", CollectionId: items.Id, MaxSelect: 10},
		&core.RelationField{Name: "author", CollectionId: users.Id, MaxSelect: 1},
	)
	bundles.ViewRule = types.Pointer("")
	bundles.ListRule = bundles.ViewRule
	if err := app.Save(bundles); err != nil {
		t.Fatal(err)
	}

	newUser := func(email string) *core.Record {
		t.Helper()
		u := core.NewRecord(users)
		u.SetEmail(email)
		u.SetPassword("secret-password")
		if err := app.Save(u); err != nil {
			t.Fatal(err)
		}
		return u
	}
	owner := newUser("owner@example.com")
	stranger := newUser("stranger@example.com")

	newItem := func(name string, public bool) *core.Record {
		t.Helper()
		r := core.NewRecord(items)
		r.Set("name", name)
		r.Set("public", public)
		r.Set("author", owner.Id)
		if err := app.Save(r); err != nil {
			t.Fatal(err)
		}
		return r
	}
	publicItem := newItem("public", true)
	privateItem := newItem("private", false)

	bundle := core.NewRecord(bundles)
	bundle.Set("items", []string{publicItem.Id, privateItem.Id})
	bundle.Set("author", owner.Id)
	if err := app.Save(bundle); err != nil {
		t.Fatal(err)
	}

	superusers, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		t.Fatal(err)
	}
	superuser := core.NewRecord(superusers)

	for _, tt := range []struct {
		name        string
		auth        *core.Record
		expand      string
		privateOnly bool
		preload     string
		want        []string
	}{
		{name: "anonymous", expand: "items", want: []string{"public"}},
		{name: "other user", auth: stranger, expand: "items", want: []string{"public"}},
		{name: "owner", auth: owner, expand: "items", want: []string{"private", "public"}},
		{name: "superuser", auth: superuser, expand: "items", want: []string{"private", "public"}},
		{name: "remote expand with no visible items", expand: "items", privateOnly: true, preload: "remote"},
		{name: "indexing expand with no visible items", expand: "items", privateOnly: true, preload: "index"},
		{name: "indexing expand with visible subset", expand: "items", preload: "index", want: []string{"public"}},
		{name: "no expand requested", preload: "remote"},
		{name: "different expand requested", expand: "author", preload: "remote"},
		{name: "invalid expand requested", expand: "missing", preload: "remote"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			record, err := app.FindRecordById(bundles, bundle.Id)
			if err != nil {
				t.Fatal(err)
			}
			if tt.privateOnly {
				record.Set("items", []string{privateItem.Id})
			}
			switch tt.preload {
			case "remote":
				// Decoded federation responses are loaded into the returned record.
				record.Load(map[string]any{"expand": map[string]any{
					"items": []any{map[string]any{"id": privateItem.Id, "name": "private"}},
				}})
			case "index":
				// Save hooks expand records without API rules for search indexing.
				if errs := app.ExpandRecord(record, []string{"items.author"}, nil); len(errs) > 0 {
					t.Fatal(errs)
				}
			}

			e := &core.RequestEvent{}
			e.App = app
			e.Auth = tt.auth
			e.Request = httptest.NewRequest("GET", "/remote/list/"+bundle.Id+"?expand="+tt.expand, nil)
			recorder := httptest.NewRecorder()
			e.Response = recorder
			if err := expandAndReturn(e, record); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != 200 {
				t.Fatalf("status = %d", recorder.Code)
			}

			var body struct {
				Expand map[string]json.RawMessage `json:"expand"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			var expandedItems []struct {
				Name   string         `json:"name"`
				Expand map[string]any `json:"expand"`
			}
			if raw, ok := body.Expand["items"]; ok {
				if err := json.Unmarshal(raw, &expandedItems); err != nil {
					t.Fatal(err)
				}
			}
			names := make([]string, 0, len(expandedItems))
			for _, item := range expandedItems {
				names = append(names, item.Name)
				if len(item.Expand) != 0 {
					t.Fatalf("unrequested nested expand retained: %s", recorder.Body.String())
				}
			}
			sort.Strings(names)
			if !slices.Equal(names, tt.want) {
				t.Fatalf("expanded items = %v, want %v; body: %s", names, tt.want, recorder.Body.String())
			}
			wantKeys := 0
			if len(tt.want) > 0 {
				wantKeys++
			}
			if tt.expand == "author" {
				wantKeys++
				var author struct {
					ID string `json:"id"`
				}
				if err := json.Unmarshal(body.Expand["author"], &author); err != nil {
					t.Fatal(err)
				}
				if author.ID != owner.Id {
					t.Fatalf("author = %q, want %q", author.ID, owner.Id)
				}
			}
			if len(body.Expand) != wantKeys {
				t.Fatalf("unexpected expansions: %s", recorder.Body.String())
			}
		})
	}
}
