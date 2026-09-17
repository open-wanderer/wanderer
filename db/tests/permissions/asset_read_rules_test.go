package permissions_test

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestAssetReadRulesRevokeAnonymousShareAccess(t *testing.T) {
	app := newRulesTestApp(t)
	newUser := func(name string) *core.Record {
		return saveRulesTestRecord(t, app, "users", map[string]any{
			"username": name, "email": name + "@example.com", "password": "test-password",
		})
	}
	owner, recipient := newUser("owner"), newUser("recipient")
	newActor := func(name, user string) *core.Record {
		return saveRulesTestRecord(t, app, "activitypub_actors", map[string]any{
			"username": name, "preferred_username": name, "domain": "example.com", "user": user,
			"public_key": "test-key", "is_local": user != "", "iri": "https://example.com/" + name,
			"inbox": "https://example.com/" + name + "/inbox",
		})
	}
	ownerActor := newActor("owner", owner.Id)
	recipientActor := newActor("recipient", recipient.Id)
	remoteActor := newActor("remote", "")

	assertAccess := func(t *testing.T, record, auth *core.Record, token string, want bool) {
		t.Helper()
		info := &core.RequestInfo{Auth: auth, Query: map[string]string{"share": token}}
		for _, check := range []struct {
			name string
			rule *string
		}{
			{"list", record.Collection().ListRule},
			{"view", record.Collection().ViewRule},
		} {
			got, err := app.CanAccessRecord(record, info, check.rule)
			if err != nil || got != want {
				t.Errorf("%s %s access = %v, %v; want %v", record.Collection().Name, check.name, got, err, want)
			}
		}
	}

	for _, actor := range []*core.Record{ownerActor, remoteActor} {
		for _, target := range []string{"trail", "waypoint", "summit_log"} {
			t.Run(actor.GetString("username")+"/"+target, func(t *testing.T) {
				trail := saveRulesTestRecord(t, app, "trails", map[string]any{"name": "private", "author": actor.Id})
				targetRecord := trail
				if target != "trail" {
					targetRecord = saveRulesTestRecord(t, app, target+"s", map[string]any{
						"name": "attachment", "author": actor.Id, "trail": trail.Id,
					})
				}
				asset := saveRulesTestRecord(t, app, "assets", map[string]any{
					"author": actor.Id, "type": "photo", "storage_mode": "link_private",
					"external_provider": "immich", "external_id": trail.Id,
				})
				link := saveRulesTestRecord(t, app, target+"_assets", map[string]any{
					target: targetRecord.Id, "asset": asset.Id,
				})
				records := []*core.Record{asset, link}
				for _, record := range records {
					assertAccess(t, record, nil, "", false)
					assertAccess(t, record, recipient, "", false)
					assertAccess(t, record, owner, "", actor.Id == ownerActor.Id)
				}

				share := saveRulesTestRecord(t, app, "trail_link_share", map[string]any{
					"trail": trail.Id, "permission": "view",
				})
				token := share.GetString("token")
				for _, record := range records {
					assertAccess(t, record, nil, token, true)
					assertAccess(t, record, nil, "", false)
					assertAccess(t, record, nil, "wrong", false)
				}
				if err := app.Delete(share); err != nil {
					t.Fatal(err)
				}
				for _, record := range records {
					assertAccess(t, record, nil, token, false)
				}

				personalShare := saveRulesTestRecord(t, app, "trail_share", map[string]any{
					"trail": trail.Id, "actor": recipientActor.Id, "permission": "view",
				})
				for _, record := range records {
					assertAccess(t, record, recipient, "", true)
					assertAccess(t, record, nil, token, false)
				}
				if err := app.Delete(personalShare); err != nil {
					t.Fatal(err)
				}

				trail.Set("public", true)
				if err := app.Save(trail); err != nil {
					t.Fatal(err)
				}
				for _, record := range records {
					assertAccess(t, record, nil, "", true)
				}
				trail.Set("public", false)
				if err := app.Save(trail); err != nil {
					t.Fatal(err)
				}
				for _, record := range records {
					assertAccess(t, record, nil, "", false)
				}
			})
		}
	}
}
