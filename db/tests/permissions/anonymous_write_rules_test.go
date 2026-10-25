package permissions_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func TestCurrentAnonymousWriteRules(t *testing.T) {
	app := newRulesTestApp(t)
	newUser := func(name string) *core.Record {
		t.Helper()
		return saveRulesTestRecord(t, app, "users", map[string]any{
			"username": name, "password": "test-password", "email": name + "@example.com",
		})
	}
	owner, editor, stranger := newUser("owner"), newUser("editor"), newUser("stranger")
	newActor := func(name, user string) *core.Record {
		t.Helper()
		return saveRulesTestRecord(t, app, "activitypub_actors", map[string]any{
			"username": name, "preferred_username": name, "domain": "example.com",
			"user": user, "is_local": user != "", "public_key": "test-key",
			"iri": "https://example.com/" + name, "inbox": "https://example.com/" + name + "/inbox",
		})
	}
	ownerActor := newActor("owner", owner.Id)
	editorActor := newActor("editor", editor.Id)
	remoteActor := newActor("remote", "")

	type content struct {
		trail, list, comment, trailShare, listShare, link, like, follow *core.Record
	}
	newContent := func(name string, actor *core.Record) content {
		t.Helper()
		trail := saveRulesTestRecord(t, app, "trails", map[string]any{
			"name": name, "author": actor.Id, "public": true,
		})
		list := saveRulesTestRecord(t, app, "lists", map[string]any{
			"name": name, "author": actor.Id, "public": true, "trails": []string{trail.Id},
		})
		comment := saveRulesTestRecord(t, app, "comments", map[string]any{
			"author": actor.Id, "trail": trail.Id, "text": name,
		})
		trailShare := saveRulesTestRecord(t, app, "trail_share", map[string]any{
			"trail": trail.Id, "actor": editorActor.Id, "permission": "edit",
		})
		listShare := saveRulesTestRecord(t, app, "list_share", map[string]any{
			"list": list.Id, "actor": editorActor.Id, "permission": "edit",
		})
		link := saveRulesTestRecord(t, app, "trail_link_share", map[string]any{
			"trail": trail.Id, "permission": "view",
		})
		like := saveRulesTestRecord(t, app, "trail_like", map[string]any{
			"trail": trail.Id, "actor": actor.Id,
		})
		follow := saveRulesTestRecord(t, app, "follows", map[string]any{
			"follower": actor.Id, "followee": editorActor.Id, "status": "accepted",
		})
		return content{trail, list, comment, trailShare, listShare, link, like, follow}
	}
	local := newContent("local", ownerActor)
	// Federated copies: the author has no local user account.
	remote := newContent("remote", remoteActor)
	settings := saveRulesTestRecord(t, app, "settings", map[string]any{
		"user": owner.Id, "language": "en", "unit": "metric", "mapFocus": "trails",
	})
	apiToken := saveRulesTestRecord(t, app, "api_tokens", map[string]any{
		"user": owner.Id, "name": "test", "token": strings.Repeat("a", 64),
	})

	assertWrite := func(t *testing.T, record, auth *core.Record, wantUpdate, wantDelete bool) {
		t.Helper()
		fresh, err := app.FindRecordById(record.Collection().Name, record.Id)
		if err != nil {
			t.Fatal(err)
		}
		for _, check := range []struct {
			name string
			rule *string
			want bool
		}{
			{"update", fresh.Collection().UpdateRule, wantUpdate},
			{"delete", fresh.Collection().DeleteRule, wantDelete},
		} {
			// Update rules that also constrain @request.body get an unchanged body.
			body := map[string]any{"seen": true}
			got, err := app.CanAccessRecord(fresh, &core.RequestInfo{Auth: auth, Body: body}, check.rule)
			if err != nil || got != check.want {
				t.Errorf("%s %s access = %v, %v; want %v", fresh.Collection().Name, check.name, got, err, check.want)
			}
		}
	}

	t.Run("anonymous cannot write federated content", func(t *testing.T) {
		for _, record := range []*core.Record{
			remote.trail, remote.list, remote.comment, remote.trailShare, remote.listShare,
			remote.link, remote.like, remote.follow,
		} {
			assertWrite(t, record, nil, false, false)
		}
	})
	t.Run("anonymous cannot write local content", func(t *testing.T) {
		for _, record := range []*core.Record{
			local.trail, local.list, local.comment, local.trailShare, local.listShare,
			local.link, local.like, local.follow, settings, apiToken,
		} {
			assertWrite(t, record, nil, false, false)
		}
	})
	t.Run("stranger cannot write", func(t *testing.T) {
		for _, record := range []*core.Record{
			local.trail, local.list, local.comment, local.trailShare, local.listShare,
			local.link, local.like, local.follow, settings, apiToken,
			remote.trail, remote.list, remote.comment,
		} {
			assertWrite(t, record, stranger, false, false)
		}
	})
	t.Run("owner keeps write access", func(t *testing.T) {
		for _, test := range []struct {
			record                 *core.Record
			wantUpdate, wantDelete bool
		}{
			{local.trail, true, true},
			{local.list, true, true},
			{local.comment, true, true},
			{local.trailShare, true, true},
			{local.listShare, true, true},
			{local.link, true, true},
			{local.like, false, true},
			{local.follow, true, true},
			{settings, true, false},
			{apiToken, false, true},
		} {
			assertWrite(t, test.record, owner, test.wantUpdate, test.wantDelete)
		}
	})
	t.Run("edit share recipient can update but not delete", func(t *testing.T) {
		assertWrite(t, local.trail, editor, true, false)
		assertWrite(t, local.list, editor, true, false)
	})
	t.Run("create rules require authentication", func(t *testing.T) {
		// Create rules are evaluated against the incoming record, so exercise
		// them through the records API instead of CanAccessRecord.
		router, err := apis.NewRouter(app)
		if err != nil {
			t.Fatal(err)
		}
		mux, err := router.BuildMux()
		if err != nil {
			t.Fatal(err)
		}
		for collection, body := range map[string]map[string]any{
			"trail_share": {"trail": remote.trail.Id, "actor": remoteActor.Id, "permission": "view"},
			"list_share":  {"list": remote.list.Id, "actor": remoteActor.Id, "permission": "view"},
			"follows":     {"follower": remoteActor.Id, "followee": ownerActor.Id, "status": "pending"},
		} {
			data, err := json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodPost, "/api/collections/"+collection+"/records", bytes.NewReader(data))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest {
				t.Errorf("anonymous POST %s: status %d; want %d: %s", collection, response.Code, http.StatusBadRequest, response.Body.String())
			}
		}
	})
}
