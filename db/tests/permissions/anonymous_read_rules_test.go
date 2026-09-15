package permissions_test

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestCurrentAnonymousReadRules(t *testing.T) {
	app := newRulesTestApp(t)
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
		assertAccess(t, localPrivate.log, nil, "wrong", false, false)
		assertAccess(t, remotePrivate.trail, nil, link.GetString("token"), false, false)
		assertAccess(t, localPrivate.comment, nil, link.GetString("token"), false, false)
	})
}
