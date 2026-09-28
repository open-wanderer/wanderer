package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"sort"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/types"
)

func TestRemoteTrailCommentsListAppliesNestedViewRules(t *testing.T) {
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

	users := core.NewAuthCollection("xt_comment_users")
	save(users)

	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(
		&core.RelationField{Name: "user", CollectionId: users.Id, MaxSelect: 1},
		&core.BoolField{Name: "is_local"},
	)
	actors.ViewRule = types.Pointer("")
	actors.ListRule = actors.ViewRule
	save(actors)

	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
		&core.BoolField{Name: "public"},
		&core.TextField{Name: "name"},
		&core.TextField{Name: "iri"},
	)
	trails.ViewRule = types.Pointer("public = true || author.user = @request.auth.id")
	trails.ListRule = trails.ViewRule
	save(trails)

	shares := core.NewBaseCollection("trail_share")
	shares.Fields.Add(
		&core.RelationField{Name: "actor", CollectionId: actors.Id, MaxSelect: 1},
		&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1},
	)
	save(shares)

	comments := core.NewBaseCollection("comments")
	comments.Fields.Add(
		&core.RelationField{Name: "author", CollectionId: actors.Id, MaxSelect: 1},
		&core.RelationField{Name: "trail", CollectionId: trails.Id, MaxSelect: 1},
	)
	// Keep the actual comments rule: a public comment is accessible even when
	// other trails reachable through its author's back-relation are private.
	comments.ListRule = types.Pointer("((@request.auth.id != \"\" && trail.author.user = @request.auth.id) || trail.public = true) || author.user = @request.auth.id || trail.trail_share_via_trail.actor.user ?= @request.auth.id")
	comments.ViewRule = comments.ListRule
	save(comments)

	owner := core.NewRecord(users)
	owner.SetEmail("comment-owner@example.com")
	owner.SetPassword("secret-password")
	save(owner)

	actor := core.NewRecord(actors)
	actor.Set("user", owner.Id)
	actor.Set("is_local", true)
	save(actor)

	newTrail := func(name string, public bool) *core.Record {
		t.Helper()
		record := core.NewRecord(trails)
		record.Set("author", actor.Id)
		record.Set("public", public)
		record.Set("name", name)
		save(record)
		return record
	}
	publicTrail := newTrail("public", true)
	newTrail("private", false)

	comment := core.NewRecord(comments)
	comment.Set("author", actor.Id)
	comment.Set("trail", publicTrail.Id)
	save(comment)

	for _, tt := range []struct {
		name       string
		auth       *core.Record
		expand     string
		wantTrails []string
	}{
		{
			name:       "anonymous nested expansion excludes private trails",
			expand:     "author.trails_via_author",
			wantTrails: []string{"public"},
		},
		{
			name:       "owner nested expansion includes private trails",
			auth:       owner,
			expand:     "author.trails_via_author",
			wantTrails: []string{"private", "public"},
		},
		{
			name:   "anonymous author expansion remains available",
			expand: "author",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/remote/trail/"+publicTrail.Id+"/comments?sort=id&expand="+url.QueryEscape(tt.expand), nil)
			request.SetPathValue("id", publicTrail.Id)
			response := httptest.NewRecorder()
			e := &core.RequestEvent{}
			e.App = app
			e.Auth = tt.auth
			e.Request = request
			e.Response = response

			if err := RemoteTrailCommentsList(e); err != nil {
				t.Fatal(err)
			}
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
			}

			var body struct {
				Items []struct {
					ID     string `json:"id"`
					Expand struct {
						Author struct {
							ID     string `json:"id"`
							Expand struct {
								Trails []struct {
									Name string `json:"name"`
								} `json:"trails_via_author"`
							} `json:"expand"`
						} `json:"author"`
					} `json:"expand"`
				} `json:"items"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if len(body.Items) != 1 || body.Items[0].ID != comment.Id {
				t.Fatalf("expected the public comment, got %s", response.Body.String())
			}
			author := body.Items[0].Expand.Author
			if author.ID != actor.Id {
				t.Fatalf("expanded author = %q, want %q", author.ID, actor.Id)
			}
			var names []string
			for _, trail := range author.Expand.Trails {
				names = append(names, trail.Name)
			}
			sort.Strings(names)
			if !slices.Equal(names, tt.wantTrails) {
				t.Fatalf("expanded trails = %v, want %v", names, tt.wantTrails)
			}
		})
	}
}
