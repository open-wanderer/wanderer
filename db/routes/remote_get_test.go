package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/types"
)

func TestRemoteGetHandlersApplyExpansionViewRules(t *testing.T) {
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	save := func(model core.Model) {
		t.Helper()
		if err := app.Save(model); err != nil {
			t.Fatal(err)
		}
	}

	users := core.NewAuthCollection("xt_remote_users")
	save(users)
	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(&core.RelationField{Name: "user", CollectionId: users.Id, MaxSelect: 1})
	actors.ViewRule = types.Pointer("")
	save(actors)
	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(
		&core.BoolField{Name: "public"},
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
	)
	save(trails)
	shares := core.NewBaseCollection("trail_link_share")
	shares.Fields.Add(
		&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1},
		&core.TextField{Name: "token"},
	)
	save(shares)
	trails.ViewRule = types.Pointer("public = true || (@request.auth.id != '' && author.user = @request.auth.id) || (@request.query.share != '' && trail_link_share_via_trail.token ?= @request.query.share)")
	save(trails)
	lists := core.NewBaseCollection("lists")
	lists.Fields.Add(&core.RelationField{Name: "trails", CollectionId: trails.Id, MaxSelect: 10})
	lists.ViewRule = types.Pointer("")
	save(lists)

	owner := core.NewRecord(users)
	owner.SetEmail("owner@example.com")
	owner.SetPassword("secret-password")
	save(owner)
	actor := core.NewRecord(actors)
	actor.Set("user", owner.Id)
	save(actor)
	newTrail := func(public bool) *core.Record {
		t.Helper()
		record := core.NewRecord(trails)
		record.Set("author", actor.Id)
		record.Set("public", public)
		save(record)
		return record
	}
	publicTrail, privateTrail := newTrail(true), newTrail(false)
	share := core.NewRecord(shares)
	share.Set("trail", privateTrail.Id)
	share.Set("token", "private-trail-token")
	save(share)
	list := core.NewRecord(lists)
	list.Set("trails", []string{publicTrail.Id, privateTrail.Id})
	save(list)

	type recordResponse struct {
		ID     string                     `json:"id"`
		Expand map[string]json.RawMessage `json:"expand"`
	}
	for _, route := range []struct {
		name, id, expand string
		handler          func(*core.RequestEvent) error
	}{
		{"list", list.Id, "trails", RemoteListGet},
		{"trail", publicTrail.Id, "author.trails_via_author", RemoteTrailGet},
	} {
		for _, tc := range []struct {
			name, token string
			auth        *core.Record
			wantPrivate bool
		}{
			{name: "anonymous"},
			{name: "owner", auth: owner, wantPrivate: true},
			{name: "valid share", token: share.GetString("token"), wantPrivate: true},
			{name: "wrong share", token: "wrong"},
		} {
			t.Run(route.name+"/"+tc.name, func(t *testing.T) {
				query := url.Values{"expand": {route.expand}}
				if tc.token != "" {
					query.Set("share", tc.token)
				}
				request := httptest.NewRequest(http.MethodGet, "/remote/"+route.name+"/"+route.id+"?"+query.Encode(), nil)
				request.SetPathValue("id", route.id)
				response := httptest.NewRecorder()
				e := &core.RequestEvent{}
				e.App, e.Auth, e.Request, e.Response = app, tc.auth, request, response
				if err := route.handler(e); err != nil {
					t.Fatal(err)
				}
				if response.Code != http.StatusOK {
					t.Fatalf("status = %d; body: %s", response.Code, response.Body.String())
				}
				var body recordResponse
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body.ID != route.id {
					t.Fatalf("record = %q, want %q", body.ID, route.id)
				}
				raw := body.Expand["trails"]
				if route.name == "trail" {
					var author recordResponse
					if err := json.Unmarshal(body.Expand["author"], &author); err != nil {
						t.Fatal(err)
					}
					if author.ID != actor.Id {
						t.Fatalf("author = %q, want %q", author.ID, actor.Id)
					}
					raw = author.Expand["trails_via_author"]
				}
				var expanded []recordResponse
				if err := json.Unmarshal(raw, &expanded); err != nil {
					t.Fatalf("missing or invalid expansion: %v; body: %s", err, response.Body.String())
				}
				got := make([]string, 0, len(expanded))
				for _, trail := range expanded {
					got = append(got, trail.ID)
				}
				want := []string{publicTrail.Id}
				if tc.wantPrivate {
					want = append(want, privateTrail.Id)
				}
				slices.Sort(got)
				slices.Sort(want)
				if !slices.Equal(got, want) {
					t.Fatalf("expanded trails = %v, want %v; body: %s", got, want, response.Body.String())
				}
			})
		}
	}
}
