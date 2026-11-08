package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"pocketbase/routes"
)

func photoTargetDeleteRequest(t *testing.T, f *recordHooksFixture) func(*core.Record, string) *httptest.ResponseRecorder {
	t.Helper()
	api, err := apis.NewRouter(f.app)
	if err != nil {
		t.Fatal(err)
	}
	api.DELETE("/assets/{id}", routes.AssetDelete)
	mux, err := api.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	return func(actor *core.Record, path string) *httptest.ResponseRecorder {
		t.Helper()
		user, err := f.app.FindRecordById("users", actor.GetString("user"))
		if err != nil {
			t.Fatal(err)
		}
		token, err := user.NewAuthToken()
		if err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(http.MethodDelete, path, nil)
		request.Header.Set("Authorization", token)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		return response
	}
}

func TestPhotoTargetPermissionsFollowTrailAndEntryOwnership(t *testing.T) {
	f := newRecordHooksFixture(t)
	owner, editor, entryAuthor := f.localActor("target-owner"), f.localActor("target-editor"), f.localActor("entry-author")
	trail := f.save("trails", map[string]any{"name": "Shared", "author": owner.Id})
	waypoint := f.save("waypoints", map[string]any{"author": editor.Id, "trail": trail.Id})
	entry := f.save("summit_logs", map[string]any{"author": entryAuthor.Id, "trail": trail.Id})
	share := f.save("trail_share", map[string]any{"trail": trail.Id, "actor": editor.Id, "permission": "edit"})
	request := photoTargetDeleteRequest(t, f)
	ownerPhoto := f.photo(owner)
	waypointLink := f.save("waypoint_assets", map[string]any{"asset": ownerPhoto.Id, "waypoint": waypoint.Id})
	if err := f.app.Delete(share); err != nil {
		t.Fatal(err)
	}
	if response := request(editor, "/assets/"+ownerPhoto.Id+"?waypoint="+waypoint.Id); response.Code != 403 || !f.exists(waypointLink) {
		t.Fatalf("waypoint author retained revoked edit permission: %d %s", response.Code, response.Body.String())
	}
	ownPhoto := f.photo(editor)
	ownWaypointLink := f.save("waypoint_assets", map[string]any{"asset": ownPhoto.Id, "waypoint": waypoint.Id})
	otherTrail := f.save("trails", map[string]any{"name": "Other", "author": owner.Id})
	if response := request(editor, "/assets/"+ownPhoto.Id+"?waypoint="+waypoint.Id+"&trail="+otherTrail.Id); response.Code != 403 || !f.exists(ownWaypointLink) || !f.exists(ownPhoto) {
		t.Fatalf("photo withdrawal ignored a mismatched child/trail context: %d %s", response.Code, response.Body.String())
	}
	if response := request(editor, "/assets/"+ownPhoto.Id+"?waypoint="+waypoint.Id); response.Code != 200 || f.exists(ownWaypointLink) || f.exists(ownPhoto) {
		t.Fatalf("photo author could not withdraw after an edit share ended: %d %s", response.Code, response.Body.String())
	}
	f.save("trail_share", map[string]any{"trail": trail.Id, "actor": editor.Id, "permission": "edit"})
	thirdPhoto := f.photo(entryAuthor)
	thirdLink := f.save("summit_log_assets", map[string]any{"asset": thirdPhoto.Id, "summit_log": entry.Id})
	if response := request(editor, "/assets/"+thirdPhoto.Id+"?summit_log="+entry.Id); response.Code != 403 || !f.exists(thirdLink) {
		t.Fatalf("trail editor changed another author's entry: %d %s", response.Code, response.Body.String())
	}
	if response := request(owner, "/assets/"+thirdPhoto.Id+"?summit_log="+entry.Id); response.Code != 200 || f.exists(thirdLink) || !f.exists(thirdPhoto) {
		t.Fatalf("trail author could not remove an entry photo: %d %s", response.Code, response.Body.String())
	}
	thirdLink = f.save("summit_log_assets", map[string]any{"asset": thirdPhoto.Id, "summit_log": entry.Id})
	if response := request(entryAuthor, "/assets/"+thirdPhoto.Id+"?summit_log="+entry.Id); response.Code != 200 || f.exists(thirdLink) || f.exists(thirdPhoto) {
		t.Fatalf("entry author could not remove their photo: %d %s", response.Code, response.Body.String())
	}
	contributed := f.photo(editor)
	contributedLink := f.save("summit_log_assets", map[string]any{"asset": contributed.Id, "summit_log": entry.Id})
	if response := request(entryAuthor, "/assets/"+contributed.Id+"?summit_log="+entry.Id); response.Code != 200 || f.exists(contributedLink) || !f.exists(contributed) {
		t.Fatalf("entry author could not remove another author's photo: %d %s", response.Code, response.Body.String())
	}
	contributedLink = f.save("summit_log_assets", map[string]any{"asset": contributed.Id, "summit_log": entry.Id})
	if response := request(editor, "/assets/"+contributed.Id+"?summit_log="+entry.Id); response.Code != 200 || f.exists(contributedLink) {
		t.Fatalf("photo author could not withdraw from another entry: %d %s", response.Code, response.Body.String())
	}
}

func TestTargetedPhotoRemovalIsIdempotentForUnlinkedAssets(t *testing.T) {
	f := newRecordHooksFixture(t)
	owner := f.localActor("remove-owner")
	trail := f.save("trails", map[string]any{"name": "Target", "author": owner.Id})
	orphan := f.photo(owner)
	linked := f.photo(owner)
	f.save("trail_assets", map[string]any{"asset": linked.Id, "trail": trail.Id})
	request := photoTargetDeleteRequest(t, f)
	check := func(id string, deleted bool) {
		t.Helper()
		response := request(owner, "/assets/"+id+"?trail="+trail.Id)
		var body struct {
			Acknowledged bool `json:"acknowledged"`
			AssetDeleted bool `json:"assetDeleted"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if response.Code != 200 || !body.Acknowledged || body.AssetDeleted != deleted {
			t.Fatalf("unexpected targeted removal: %d %s", response.Code, response.Body.String())
		}
	}
	check("unknown00000001", false)
	check(orphan.Id, false)
	if !f.exists(orphan) {
		t.Fatal("targeted retry deleted an unrelated owned orphan")
	}
	check(linked.Id, true)
	check(linked.Id, false)
	if response := request(owner, "/assets/"+orphan.Id+"?trail=missing00000001"); response.Code != 404 {
		t.Fatalf("missing target did not return 404: %d %s", response.Code, response.Body.String())
	}
	if response := request(owner, "/assets/unknown00000001"); response.Code != 404 {
		t.Fatalf("missing explicit asset did not return 404: %d %s", response.Code, response.Body.String())
	}
	if response := request(owner, "/assets/"+orphan.Id); response.Code != 200 || f.exists(orphan) {
		t.Fatalf("explicit orphan deletion failed: %d %s", response.Code, response.Body.String())
	}
}

func TestTargetedPhotoRemovalMissingNoopAfterShareEnds(t *testing.T) {
	for _, kind := range []string{"trail", "waypoint", "summit_log"} {
		t.Run(kind, func(t *testing.T) {
			f := newRecordHooksFixture(t)
			owner, contributor := f.localActor("noop-owner"), f.localActor("noop-contributor")
			trail := f.save("trails", map[string]any{"name": "Private", "author": owner.Id})
			target := trail
			if kind == "waypoint" {
				target = f.save("waypoints", map[string]any{"author": contributor.Id, "trail": trail.Id})
			}
			if kind == "summit_log" {
				target = f.save("summit_logs", map[string]any{"author": owner.Id, "trail": trail.Id})
			}
			collection := kind + "_assets"
			share := f.save("trail_share", map[string]any{"trail": trail.Id, "actor": contributor.Id, "permission": "edit"})
			photo, otherPhoto := f.photo(contributor), f.photo(owner)
			f.save(collection, map[string]any{"asset": photo.Id, kind: target.Id})
			otherLink := f.save(collection, map[string]any{"asset": otherPhoto.Id, kind: target.Id})
			if err := f.app.Delete(share); err != nil {
				t.Fatal(err)
			}
			request := photoTargetDeleteRequest(t, f)
			path := "/assets/" + photo.Id + "?" + kind + "=" + target.Id
			if response := request(contributor, "/assets/"+otherPhoto.Id+"?"+kind+"="+target.Id); response.Code != 403 || !f.exists(otherLink) {
				t.Fatalf("unexpected foreign photo removal: %d %s", response.Code, response.Body.String())
			}
			if response := request(contributor, path); response.Code != 200 || f.exists(photo) {
				t.Fatalf("author withdrawal failed: %d %s", response.Code, response.Body.String())
			}
			var mu sync.Mutex
			queries := []string{}
			for _, database := range []*dbx.DB{f.app.ConcurrentDB().(*dbx.DB), f.app.NonconcurrentDB().(*dbx.DB)} {
				database := database
				previous := database.QueryLogFunc
				database.QueryLogFunc = func(_ context.Context, _ time.Duration, query string, _ *sql.Rows, _ error) {
					if strings.Contains(query, "FROM `"+collection+"`") {
						mu.Lock()
						queries = append(queries, query)
						mu.Unlock()
					}
				}
				defer func() { database.QueryLogFunc = previous }()
			}
			response := request(contributor, path)
			var body struct {
				Acknowledged bool `json:"acknowledged"`
				AssetDeleted bool `json:"assetDeleted"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if response.Code != 200 || !body.Acknowledged || body.AssetDeleted {
				t.Fatalf("missing-asset retry required revoked edit permission: %d %s", response.Code, response.Body.String())
			}
			mu.Lock()
			linkQueries := len(queries)
			mu.Unlock()
			if linkQueries != 0 {
				t.Fatalf("missing-asset no-op entered the link deletion path: %v", queries)
			}
			if response := request(contributor, "/assets/"+photo.Id+"?"+kind+"=missing00000001"); response.Code != 404 {
				t.Fatalf("missing-asset retry skipped target existence: %d %s", response.Code, response.Body.String())
			}
			if kind != "trail" {
				otherTrail := f.save("trails", map[string]any{"name": "Other", "author": owner.Id})
				if response := request(contributor, path+"&trail="+otherTrail.Id); response.Code != 403 {
					t.Fatalf("missing-asset retry skipped child/trail context: %d %s", response.Code, response.Body.String())
				}
			}
			if !f.exists(otherPhoto) || !f.exists(otherLink) {
				t.Fatal("missing-asset no-op changed an existing photo or link")
			}
		})
	}
}
