package migrations

import (
	"reflect"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestAnonymousReadRulesMigration(t *testing.T) {
	testAnonymousReadRules(t, false)
}

func TestCurrentAnonymousReadRules(t *testing.T) {
	testAnonymousReadRules(t, true)
}

func testAnonymousReadRules(t *testing.T, currentSchema bool) {
	t.Helper()
	beforeMigration := ""
	if !currentSchema {
		beforeMigration = "1789200002_guard_anonymous_read_rules.go"
	}
	app := newRulesTestApp(t, beforeMigration)
	newUser := func(name string) *core.Record {
		t.Helper()
		return saveRulesTestRecord(t, app, "users", map[string]any{
			"username": name, "password": "test-password", "email": name + "@example.com",
		})
	}
	owner, recipient := newUser("owner"), newUser("recipient")
	newActor := func(name, user string) *core.Record {
		t.Helper()
		return saveRulesTestRecord(t, app, "activitypub_actors", map[string]any{
			"username": name, "preferred_username": name, "domain": "example.com", "user": user, "public_key": "test-key", "is_local": user != "",
			"iri": "https://example.com/" + name, "inbox": "https://example.com/" + name + "/inbox",
		})
	}
	ownerActor := newActor("owner", owner.Id)
	recipientActor := newActor("recipient", recipient.Id)
	remoteActor := newActor("remote", "")

	type content struct {
		trail, list, waypoint, comment, log *core.Record
	}
	newContent := func(name string, actor *core.Record, public bool) content {
		t.Helper()
		trail := saveRulesTestRecord(t, app, "trails", map[string]any{
			"name": name, "author": actor.Id, "public": public,
		})
		list := saveRulesTestRecord(t, app, "lists", map[string]any{
			"name": name, "author": actor.Id, "public": public, "trails": []string{trail.Id},
		})
		waypoint := saveRulesTestRecord(t, app, "waypoints", map[string]any{
			"name": name, "author": actor.Id, "trail": trail.Id,
		})
		comment := saveRulesTestRecord(t, app, "comments", map[string]any{
			"author": actor.Id, "trail": trail.Id, "text": name,
		})
		log := saveRulesTestRecord(t, app, "summit_logs", map[string]any{
			"author": actor.Id, "trail": trail.Id,
		})
		return content{trail, list, waypoint, comment, log}
	}
	localPrivate := newContent("local-private", ownerActor, false)
	remotePrivate := newContent("remote-private", remoteActor, false)
	localPublic := newContent("local-public", ownerActor, true)
	remotePublic := newContent("remote-public", remoteActor, true)
	shared := newContent("shared", ownerActor, false)
	newShare := func(collection, field string, record, actor *core.Record) *core.Record {
		t.Helper()
		return saveRulesTestRecord(t, app, collection, map[string]any{
			field: record.Id, "actor": actor.Id, "permission": "view",
		})
	}
	trailShare := newShare("trail_share", "trail", shared.trail, recipientActor)
	listShare := newShare("list_share", "list", shared.list, recipientActor)
	remoteTrailShare := newShare("trail_share", "trail", shared.trail, remoteActor)
	remoteListShare := newShare("list_share", "list", shared.list, remoteActor)
	newLike := func(trail, actor *core.Record) *core.Record {
		t.Helper()
		return saveRulesTestRecord(t, app, "trail_like", map[string]any{
			"trail": trail.Id, "actor": actor.Id,
		})
	}
	ownerLike := newLike(localPrivate.trail, ownerActor)
	remotePrivateLike := newLike(remotePrivate.trail, remoteActor)
	remotePublicLike := newLike(localPublic.trail, remoteActor)
	link := saveRulesTestRecord(t, app, "trail_link_share", map[string]any{
		"trail": localPrivate.trail.Id, "permission": "view",
	})

	assertAccess := func(t *testing.T, record, auth *core.Record, token string, wantList, wantView bool) {
		t.Helper()
		fresh, err := app.FindRecordById(record.Collection().Name, record.Id)
		if err != nil {
			t.Fatal(err)
		}
		info := &core.RequestInfo{Auth: auth, Query: map[string]string{"share": token}}
		for _, check := range []struct {
			name string
			rule *string
			want bool
		}{
			{"list", fresh.Collection().ListRule, wantList},
			{"view", fresh.Collection().ViewRule, wantView},
		} {
			got, err := app.CanAccessRecord(fresh, info, check.rule)
			if err != nil || got != check.want {
				t.Errorf("%s %s access = %v, %v; want %v", fresh.Collection().Name, check.name, got, err, check.want)
			}
		}
	}

	// Exercise the real previous rules: both missing shares and remote actors'
	// empty user relations can match an anonymous caller's empty auth id.
	if !currentSchema {
		assertAccess(t, remotePrivate.trail, nil, "", true, true)
		assertAccess(t, remotePrivate.list, nil, "", true, true)
		assertAccess(t, localPrivate.waypoint, nil, "", true, true)
		assertAccess(t, localPrivate.comment, nil, "", true, true)
		assertAccess(t, localPrivate.log, nil, "", true, true)
		assertAccess(t, remotePrivateLike, nil, "", true, true)
	}

	collections := []string{"trails", "lists", "waypoints", "comments", "summit_logs", "trail_share", "list_share", "trail_like"}
	previousRules := make(map[string][5]*string, len(collections))
	for _, name := range collections {
		col, err := app.FindCollectionByNameOrId(name)
		if err != nil {
			t.Fatal(err)
		}
		previousRules[name] = [5]*string{col.ListRule, col.ViewRule, col.CreateRule, col.UpdateRule, col.DeleteRule}
	}
	migration := ruleMigration(t, "1789200002_guard_anonymous_read_rules.go")
	if !currentSchema {
		if err := app.RunInTransaction(migration.Up); err != nil {
			t.Fatal(err)
		}
	}

	for _, test := range []struct {
		name string
		data content
		auth *core.Record
		want bool
	}{
		{"anonymous local private", localPrivate, nil, false},
		{"anonymous remote private", remotePrivate, nil, false},
		{"anonymous shared private", shared, nil, false},
		{"anonymous local public", localPublic, nil, true},
		{"anonymous remote public", remotePublic, nil, true},
		{"local owner", localPrivate, owner, true},
		{"unrelated authenticated user", localPrivate, recipient, false},
		{"share recipient", shared, recipient, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, record := range []*core.Record{test.data.trail, test.data.list, test.data.waypoint, test.data.comment, test.data.log} {
				assertAccess(t, record, test.auth, "", test.want, test.want)
			}
		})
	}
	for _, share := range []*core.Record{trailShare, listShare, remoteTrailShare, remoteListShare} {
		t.Run(share.Collection().Name+"/"+share.Id, func(t *testing.T) {
			assertAccess(t, share, nil, "", false, false)
			assertAccess(t, share, owner, "", true, true)
			isRecipient := share.GetString("actor") == recipientActor.Id
			assertAccess(t, share, recipient, "", isRecipient, isRecipient)
		})
	}
	t.Run("likes retain distinct list and view visibility", func(t *testing.T) {
		assertAccess(t, ownerLike, nil, "", false, false)
		assertAccess(t, ownerLike, owner, "", true, true)
		assertAccess(t, remotePrivateLike, nil, "", false, false)
		assertAccess(t, remotePublicLike, nil, "", true, false)
		assertAccess(t, remotePublicLike, owner, "", true, false)
	})
	t.Run("share links remain scoped to their trail", func(t *testing.T) {
		for _, record := range []*core.Record{localPrivate.trail, localPrivate.waypoint} {
			assertAccess(t, record, nil, link.GetString("token"), true, true)
			assertAccess(t, record, nil, "wrong", false, false)
		}
		// Summit-log share-link support is added by the later expansion fix.
		if !currentSchema {
			assertAccess(t, localPrivate.log, nil, link.GetString("token"), false, false)
		}
		assertAccess(t, localPrivate.log, nil, "wrong", false, false)
		assertAccess(t, remotePrivate.trail, nil, link.GetString("token"), false, false)
		assertAccess(t, localPrivate.comment, nil, link.GetString("token"), false, false)
	})
	for _, name := range collections {
		col, err := app.FindCollectionByNameOrId(name)
		if err != nil {
			t.Fatal(err)
		}
		previous := previousRules[name]
		if !reflect.DeepEqual([3]*string{col.CreateRule, col.UpdateRule, col.DeleteRule}, [3]*string{previous[2], previous[3], previous[4]}) {
			t.Errorf("%s write rules changed", name)
		}
	}
	if currentSchema {
		return
	}
	if err := app.RunInTransaction(migration.Down); err != nil {
		t.Fatal(err)
	}
	for _, name := range collections {
		col, err := app.FindCollectionByNameOrId(name)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual([5]*string{col.ListRule, col.ViewRule, col.CreateRule, col.UpdateRule, col.DeleteRule}, previousRules[name]) {
			t.Errorf("%s rollback did not restore the previous rules", name)
		}
	}
}
