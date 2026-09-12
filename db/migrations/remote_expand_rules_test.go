package migrations

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func TestTagsViewMigrationKeepsCatalogRestricted(t *testing.T) {
	app := newRulesTestApp(t, "1789200001_updated_tags_view.go")
	tag := saveRulesTestRecord(t, app, "tags", map[string]any{"name": "alpine"})
	owner := saveRulesTestRecord(t, app, "users", map[string]any{
		"username": "owner", "password": "test-password", "email": "owner@example.com",
	})
	actor := saveRulesTestRecord(t, app, "activitypub_actors", map[string]any{
		"username": "owner", "preferred_username": "owner", "domain": "example.com", "user": owner.Id, "public_key": "test-key",
		"iri": "https://example.com/owner", "inbox": "https://example.com/owner/inbox",
	})
	trail := saveRulesTestRecord(t, app, "trails", map[string]any{
		"name": "public", "public": true, "author": actor.Id, "tags": []string{tag.Id},
	})
	migration := ruleMigration(t, "1789200001_updated_tags_view.go")
	previousListRule := *tag.Collection().ListRule
	previousCreateRule := *tag.Collection().CreateRule

	assertAnonymousAccess := func(wantView bool) {
		t.Helper()
		record, err := app.FindRecordById("tags", tag.Id)
		if err != nil {
			t.Fatal(err)
		}
		col := record.Collection()
		if col.ListRule == nil || *col.ListRule != previousListRule ||
			col.CreateRule == nil || *col.CreateRule != previousCreateRule {
			t.Fatal("migration changed the tag list or create rule")
		}
		for _, test := range []struct {
			name string
			rule *string
			want bool
		}{
			{"view", record.Collection().ViewRule, wantView},
			{"list", record.Collection().ListRule, false},
			{"create", record.Collection().CreateRule, false},
		} {
			got, err := app.CanAccessRecord(record, &core.RequestInfo{}, test.rule)
			if err != nil || got != test.want {
				t.Fatalf("anonymous %s access = %v, %v; want %v", test.name, got, err, test.want)
			}
		}
		freshTrail, err := app.FindRecordById("trails", trail.Id)
		if err != nil {
			t.Fatal(err)
		}
		e := &core.RequestEvent{}
		e.App = app
		e.Request = httptest.NewRequest("GET", "/trail?expand=tags", nil)
		if err := apis.EnrichRecord(e, freshTrail); err != nil {
			t.Fatal(err)
		}
		expanded := freshTrail.ExpandedAll("tags")
		if (len(expanded) == 1) != wantView || len(expanded) > 1 {
			t.Fatalf("expanded %d tags; want view access %v", len(expanded), wantView)
		}
		if wantView && expanded[0].GetString("name") != "alpine" {
			t.Fatalf("expanded tag name = %q; want alpine", expanded[0].GetString("name"))
		}
	}

	assertAnonymousAccess(false)
	if err := app.RunInTransaction(migration.Up); err != nil {
		t.Fatal(err)
	}
	assertAnonymousAccess(true)
	if err := app.RunInTransaction(migration.Down); err != nil {
		t.Fatal(err)
	}
	assertAnonymousAccess(false)
}

func TestSummitLogShareMigration(t *testing.T) {
	app := newRulesTestApp(t, "1789200003_updated_summit_logs_link_share.go")
	newActor := func(name string) (*core.Record, *core.Record) {
		t.Helper()
		user := saveRulesTestRecord(t, app, "users", map[string]any{
			"username": name, "password": "test-password", "email": name + "@example.com",
		})
		actor := saveRulesTestRecord(t, app, "activitypub_actors", map[string]any{
			"username": name, "preferred_username": name, "domain": "example.com", "user": user.Id, "public_key": "test-key",
			"iri": "https://example.com/" + name, "inbox": "https://example.com/" + name + "/inbox",
		})
		return user, actor
	}
	owner, actor := newActor("owner")
	recipient, recipientActor := newActor("recipient")
	newTrail := func(name string, public bool) (*core.Record, *core.Record) {
		t.Helper()
		trail := saveRulesTestRecord(t, app, "trails", map[string]any{
			"name": name, "author": actor.Id, "public": public,
		})
		log := saveRulesTestRecord(t, app, "summit_logs", map[string]any{
			"trail": trail.Id, "author": actor.Id,
		})
		return trail, log
	}
	privateTrail, privateLog := newTrail("private", false)
	unrelatedTrail, unrelatedLog := newTrail("unrelated", false)
	publicTrail, publicLog := newTrail("public", true)
	link := saveRulesTestRecord(t, app, "trail_link_share", map[string]any{
		"trail": privateTrail.Id, "permission": "view",
	})
	saveRulesTestRecord(t, app, "trail_share", map[string]any{
		"trail": unrelatedTrail.Id, "actor": recipientActor.Id, "permission": "view",
	})
	token := link.GetString("token")
	previousListRule := *privateLog.Collection().ListRule
	previousViewRule := *privateLog.Collection().ViewRule

	assertAccess := func(t *testing.T, trail, log, auth *core.Record, share string, want bool) {
		t.Helper()
		record, err := app.FindRecordById("summit_logs", log.Id)
		if err != nil {
			t.Fatal(err)
		}
		request := &core.RequestInfo{Auth: auth, Query: map[string]string{}}
		if share != "" {
			request.Query["share"] = share
		}
		for _, rule := range []*string{record.Collection().ListRule, record.Collection().ViewRule} {
			got, err := app.CanAccessRecord(record, request, rule)
			if err != nil || got != want {
				t.Fatalf("summit log access = %v, %v; want %v", got, err, want)
			}
		}

		freshTrail, err := app.FindRecordById("trails", trail.Id)
		if err != nil {
			t.Fatal(err)
		}
		e := &core.RequestEvent{}
		e.App, e.Auth = app, auth
		query := url.Values{"expand": {"summit_logs_via_trail"}}
		if share != "" {
			query.Set("share", share)
		}
		e.Request = httptest.NewRequest("GET", "/trail?"+query.Encode(), nil)
		if err := apis.EnrichRecord(e, freshTrail); err != nil {
			t.Fatal(err)
		}
		expanded := freshTrail.ExpandedAll("summit_logs_via_trail")
		if (len(expanded) == 1) != want || len(expanded) > 1 {
			t.Fatalf("expanded %d summit logs; want access %v", len(expanded), want)
		}
		if want && expanded[0].Id != log.Id {
			t.Fatalf("expanded log %s; want %s", expanded[0].Id, log.Id)
		}
	}

	// Link access is added only by this migration; ordinary anonymous access
	// stays denied before and after it.
	assertAccess(t, privateTrail, privateLog, nil, "", false)
	assertAccess(t, privateTrail, privateLog, nil, token, false)
	migration := ruleMigration(t, "1789200003_updated_summit_logs_link_share.go")
	if err := app.RunInTransaction(migration.Up); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name       string
		trail, log *core.Record
		auth       *core.Record
		share      string
		want       bool
	}{
		{"valid share", privateTrail, privateLog, nil, token, true},
		{"no share", privateTrail, privateLog, nil, "", false},
		{"invalid share", privateTrail, privateLog, nil, "wrong", false},
		{"other trail's share", unrelatedTrail, unrelatedLog, nil, token, false},
		{"owner without share", privateTrail, privateLog, owner, "", true},
		{"actor share recipient", unrelatedTrail, unrelatedLog, recipient, "", true},
		{"authenticated without access", privateTrail, privateLog, recipient, "", false},
		{"public without share", publicTrail, publicLog, nil, "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			assertAccess(t, test.trail, test.log, test.auth, test.share, test.want)
		})
	}
	if err := app.RunInTransaction(migration.Down); err != nil {
		t.Fatal(err)
	}
	col, err := app.FindCollectionByNameOrId("summit_logs")
	if err != nil {
		t.Fatal(err)
	}
	if *col.ListRule != previousListRule || *col.ViewRule != previousViewRule {
		t.Fatal("rollback did not restore the previous summit log rules")
	}
	assertAccess(t, privateTrail, privateLog, nil, "", false)
	assertAccess(t, privateTrail, privateLog, nil, token, false)
}
