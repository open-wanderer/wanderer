package migrations

import (
	"fmt"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestPurgeUnreferencedRemoteActorsKeepsEveryReferencedActor(t *testing.T) {
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.ResetBootstrapState() })

	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(
		&core.BoolField{Name: "is_local"},
		&core.URLField{Name: "iri"},
		&core.URLField{Name: "inbox"},
		&core.SelectField{Name: "actor_type", Values: []string{"person", "instance"}, MaxSelect: 1},
	)
	if err := app.Save(actors); err != nil {
		t.Fatal(err)
	}
	referencing := map[string]*core.RelationField{
		"trails":  {Name: "author", CollectionId: actors.Id, MaxSelect: 1, CascadeDelete: true},
		"follows": {Name: "followee", CollectionId: actors.Id, MaxSelect: 1, CascadeDelete: true},
		"feed":    {Name: "actor", CollectionId: actors.Id, MaxSelect: 1, CascadeDelete: true},
		"groups":  {Name: "members", CollectionId: actors.Id, MaxSelect: 5},
	}
	activities := core.NewBaseCollection("activitypub_activities")
	activities.Fields.Add(
		&core.TextField{Name: "type"},
		&core.URLField{Name: "actor"},
		&core.JSONField{Name: "to"},
		&core.JSONField{Name: "cc"},
	)
	if err := app.Save(activities); err != nil {
		t.Fatal(err)
	}
	for name, field := range referencing {
		collection := core.NewBaseCollection(name)
		collection.Fields.Add(field)
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}

	newActor := func(id string, local bool, actorType string) {
		t.Helper()
		record := core.NewRecord(actors)
		record.Id = id
		record.Set("is_local", local)
		record.Set("actor_type", actorType)
		record.Set("iri", "https://remote.example/users/"+id)
		record.Set("inbox", "https://remote.example/users/"+id+"/inbox")
		if err := app.Save(record); err != nil {
			t.Fatal(err)
		}
	}
	reference := func(collection, id string, value any) {
		t.Helper()
		c, _ := app.FindCollectionByNameOrId(collection)
		record := core.NewRecord(c)
		record.Id = id
		record.Set(referencing[collection].Name, value)
		if err := app.Save(record); err != nil {
			t.Fatal(err)
		}
	}

	kept := []string{"localperson0001", "localinstance01", "remoteinstance1", "trailauthor0001", "followee0000001", "feedactor000001", "member000000001", "member000000002", "mentioned000001", "announced000001", "sender000000001", "barecc000000001", "bareto000000001", "untyped00000001"}
	newActor("localperson0001", true, "person")
	newActor("localinstance01", true, "instance")
	newActor("remoteinstance1", false, "instance")
	for _, id := range kept[3:] {
		newActor(id, false, "person")
	}
	reference("trails", "trail0000000001", "trailauthor0001")
	reference("follows", "follow000000001", "followee0000001")
	reference("feed", "feed00000000001", "feedactor000001")
	reference("groups", "group0000000001", []string{"member000000001", "member000000002"})
	// Rows from before actor_type default to '' and are still person actors.
	if _, err := app.DB().NewQuery("UPDATE activitypub_actors SET actor_type = '' WHERE id = 'untyped00000001'").Execute(); err != nil {
		t.Fatal(err)
	}
	reference("feed", "feed00000000002", "untyped00000001")

	activity := func(activityType, actor string, to, cc []string) {
		t.Helper()
		record := core.NewRecord(activities)
		record.Set("type", activityType)
		record.Set("actor", actor)
		record.Set("to", to)
		record.Set("cc", cc)
		if err := app.Save(record); err != nil {
			t.Fatal(err)
		}
	}
	// A mention is addressed by inbox in cc; a share whose trail_share row was
	// revoked survives only in its Announce's to.
	activity("Create", "https://wanderer.example/users/local", []string{"https://www.w3.org/ns/activitystreams#Public"}, []string{"https://remote.example/users/mentioned000001/inbox"})
	activity("Announce", "https://wanderer.example/users/local", []string{"https://remote.example/users/announced000001"}, nil)
	activity("Like", "https://remote.example/users/sender000000001", nil, nil)
	// About half of the stored rows hold to and cc as a bare JSON string, and a
	// malformed value must not abort the migration.
	for _, raw := range []string{
		`INSERT INTO activitypub_activities (id, type, actor, [to], cc) VALUES ('barestring00001', 'Create', '', '"https://www.w3.org/ns/activitystreams#Public"', '"https://remote.example/users/barecc000000001/inbox"')`,
		`INSERT INTO activitypub_activities (id, type, actor, [to], cc) VALUES ('barestring00002', 'Announce', '', '"https://remote.example/users/bareto000000001"', '[]')`,
		`INSERT INTO activitypub_activities (id, type, actor, [to], cc) VALUES ('malformed000001', 'Create', '', 'not json', '["unterminated')`,
	} {
		if _, err := app.DB().NewQuery(raw).Execute(); err != nil {
			t.Fatal(err)
		}
	}

	const orphans = 5001 // Cross the batch boundary.
	for i := 0; i < orphans; i++ {
		newActor(fmt.Sprintf("orphan%09d", i), false, "person")
	}
	if _, err := app.DB().NewQuery("UPDATE activitypub_actors SET actor_type = '' WHERE id = 'orphan000000000'").Execute(); err != nil {
		t.Fatal(err)
	}

	// The migration runner calls it inside its own transaction.
	purge := func() error {
		return app.RunInTransaction(func(txApp core.App) error {
			return upPurgeUnreferencedRemoteActors1791713197(txApp)
		})
	}
	if err := purge(); err != nil {
		t.Fatal(err)
	}

	total, err := app.CountRecords("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	if total != int64(len(kept)) {
		t.Fatalf("got %d actors after purge, want %d", total, len(kept))
	}
	for _, id := range kept {
		if _, err := app.FindRecordById("activitypub_actors", id); err != nil {
			t.Fatalf("referenced, local or instance actor %s was deleted", id)
		}
	}
	// The raw DELETE does not cascade: a deleted referenced actor would show
	// up as a relation value without its row.
	for collection, field := range referencing {
		var dangling []string
		err := app.DB().NewQuery(fmt.Sprintf(
			"SELECT j.value FROM {{%s}} c, json_each(CASE WHEN json_valid(c.[[%s]]) THEN c.[[%s]] ELSE json_array(c.[[%s]]) END) j WHERE j.value NOT IN (SELECT id FROM activitypub_actors)",
			collection, field.Name, field.Name, field.Name,
		)).Column(&dangling)
		if err != nil {
			t.Fatal(err)
		}
		if len(dangling) > 0 {
			t.Fatalf("%s.%s points at deleted actors %v", collection, field.Name, dangling)
		}
	}

	if err := purge(); err != nil {
		t.Fatalf("rerun failed: %v", err)
	}
	if rerun, _ := app.CountRecords("activitypub_actors"); rerun != total {
		t.Fatalf("rerun deleted %d more actors", total-rerun)
	}
}
