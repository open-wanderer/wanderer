package migrations

import (
	"database/sql"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestAdditionalAnonymousReadRulesMigration(t *testing.T) {
	testAdditionalAnonymousReadRules(t, true)
}

func TestCurrentAdditionalAnonymousReadRules(t *testing.T) {
	testAdditionalAnonymousReadRules(t, false)
}

func testAdditionalAnonymousReadRules(t *testing.T, exerciseMigration bool) {
	t.Helper()
	const migrationFile = "1789200002_guard_anonymous_read_rules.go"
	beforeMigration := ""
	if exerciseMigration {
		beforeMigration = migrationFile
	}
	app := newRulesTestApp(t, beforeMigration)
	newUser := func(name string) *core.Record {
		t.Helper()
		return saveRulesTestRecord(t, app, "users", map[string]any{
			"username": name, "password": "test-password", "email": name + "@example.com",
		})
	}
	owner, stranger := newUser("owner"), newUser("stranger")
	newActor := func(name, user string) *core.Record {
		t.Helper()
		return saveRulesTestRecord(t, app, "activitypub_actors", map[string]any{
			"username": name, "preferred_username": name, "domain": "example.com",
			"user": user, "is_local": user != "", "public_key": "test-key",
			"iri": "https://example.com/" + name, "inbox": "https://example.com/" + name + "/inbox",
		})
	}
	ownerActor := newActor("owner", owner.Id)
	remoteActor := newActor("remote", "")
	newTrail := func(actor *core.Record) *core.Record {
		t.Helper()
		return saveRulesTestRecord(t, app, "trails", map[string]any{
			"name": "private", "author": actor.Id, "public": false,
			"distance": 12, "min_lat": 46, "max_lat": 47, "min_lon": 8, "max_lon": 9,
		})
	}
	localTrail, remoteTrail := newTrail(ownerActor), newTrail(remoteActor)
	newLink := func(trail *core.Record) *core.Record {
		t.Helper()
		return saveRulesTestRecord(t, app, "trail_link_share", map[string]any{
			"trail": trail.Id, "permission": "view",
		})
	}
	localLink, remoteLink := newLink(localTrail), newLink(remoteTrail)
	newFeedEntry := func(actor, trail *core.Record) *core.Record {
		t.Helper()
		return saveRulesTestRecord(t, app, "feed", map[string]any{
			"actor": actor.Id, "item": trail.Id, "type": "trail",
		})
	}
	localFeed, remoteFeed := newFeedEntry(ownerActor, localTrail), newFeedEntry(remoteActor, remoteTrail)
	settings := saveRulesTestRecord(t, app, "settings", map[string]any{
		"user": owner.Id, "language": "en", "unit": "metric", "mapFocus": "trails",
	})
	apiToken := saveRulesTestRecord(t, app, "api_tokens", map[string]any{
		"user": owner.Id, "name": "test", "token": strings.Repeat("a", 64),
	})

	assertAccess := func(t *testing.T, record, auth *core.Record, wantList, wantView bool) {
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
			{"list", fresh.Collection().ListRule, wantList},
			{"view", fresh.Collection().ViewRule, wantView},
		} {
			got, err := app.CanAccessRecord(fresh, &core.RequestInfo{Auth: auth}, check.rule)
			if err != nil || got != check.want {
				t.Errorf("%s %s access = %v, %v; want %v", fresh.Collection().Name, check.name, got, err, check.want)
			}
		}
	}
	// The remote owner's empty user relation previously exposed link tokens
	// and remote feed entries to anonymous callers.
	if exerciseMigration {
		assertAccess(t, remoteLink, nil, true, true)
		assertAccess(t, remoteFeed, nil, true, false)
	}

	type ownerAccess struct {
		record               *core.Record
		ownerList, ownerView bool
	}
	cases := []ownerAccess{
		{localLink, true, true},
		{settings, true, true},
		{localFeed, true, false},
		{apiToken, true, true},
	}
	for _, name := range []string{"trails_bounding_box", "trails_filter"} {
		// These current view queries already exclude remote actors. The new
		// auth check is additional protection, not a fix for leaked aggregates.
		if _, err := app.FindRecordById(name, remoteActor.Id); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("%s remote actor row: got %v; want no rows", name, err)
		}
		local, err := app.FindRecordById(name, ownerActor.Id)
		if err != nil {
			t.Fatal(err)
		}
		assertAccess(t, local, nil, false, false)
		cases = append(cases, ownerAccess{local, false, true})
	}

	previousRules := make(map[string][5]*string, len(cases))
	for _, test := range cases {
		col := test.record.Collection()
		previousRules[col.Name] = [5]*string{col.ListRule, col.ViewRule, col.CreateRule, col.UpdateRule, col.DeleteRule}
	}
	migration := ruleMigration(t, migrationFile)
	if exerciseMigration {
		if err := app.RunInTransaction(migration.Up); err != nil {
			t.Fatal(err)
		}
	}
	for _, test := range cases {
		t.Run(test.record.Collection().Name, func(t *testing.T) {
			assertAccess(t, test.record, nil, false, false)
			assertAccess(t, test.record, stranger, false, false)
			assertAccess(t, test.record, owner, test.ownerList, test.ownerView)
			col, err := app.FindCollectionByNameOrId(test.record.Collection().Name)
			if err != nil {
				t.Fatal(err)
			}
			previous := previousRules[col.Name]
			if (previous[0] == nil) != (col.ListRule == nil) || (previous[1] == nil) != (col.ViewRule == nil) {
				t.Fatal("migration changed a locked read rule")
			}
			if !reflect.DeepEqual(previous[2:], []*string{col.CreateRule, col.UpdateRule, col.DeleteRule}) {
				t.Fatal("migration changed write rules")
			}
		})
	}
	assertAccess(t, remoteLink, nil, false, false)
	assertAccess(t, remoteFeed, nil, false, false)
	if !exerciseMigration {
		return
	}
	if err := app.RunInTransaction(migration.Down); err != nil {
		t.Fatal(err)
	}
	for name, previous := range previousRules {
		col, err := app.FindCollectionByNameOrId(name)
		if err != nil {
			t.Fatal(err)
		}
		current := [5]*string{col.ListRule, col.ViewRule, col.CreateRule, col.UpdateRule, col.DeleteRule}
		if !reflect.DeepEqual(current, previous) {
			t.Errorf("%s rollback did not restore the exact prior rules", name)
		}
	}
	assertAccess(t, remoteLink, nil, true, true)
	assertAccess(t, remoteFeed, nil, true, false)
}
