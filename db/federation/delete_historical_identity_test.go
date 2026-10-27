package federation

import (
	"testing"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

func TestDeleteGuardsPreserveHistoricalLocalIdentityAndRejectImportedIdentity(t *testing.T) {
	f := setupAddressingTestApp(t)
	local := f.actor(t, "alice", true)
	trail := f.record(t, "trails", map[string]any{"author": local.Id, "public": true}, "trail")
	log := f.record(t, "summit_logs", map[string]any{"author": local.Id, "trail": trail.Id}, "summit-log")
	list := f.record(t, "lists", map[string]any{"author": local.Id, "public": true}, "list")
	for _, record := range []*core.Record{trail, log, list} {
		record.Set("iri", "https://intermediate.example:444/api/v1/"+map[string]string{"trails": "trail", "summit_logs": "summit-log", "lists": "list"}[record.Collection().Name]+"/"+record.Id)
		if err := f.app.Save(record); err != nil {
			t.Fatal(err)
		}
	}
	for _, operation := range []func() error{
		func() error { return CreateTrailDeleteActivity(f.app, trail, DeleteAudience{}) },
		func() error { return CreateSummitLogDeleteActivity(f.app, log) },
		func() error { return CreateListDeleteActivity(f.app, list, DeleteAudience{}) },
	} {
		if err := operation(); err != nil {
			t.Fatal(err)
		}
	}
	count, err := f.app.CountRecords("activitypub_activities")
	if err != nil || count != 3 {
		t.Fatalf("historical local deletes missing: count=%d err=%v", count, err)
	}
	remote := f.actor(t, "remote", false)
	remoteTrail := f.record(t, "trails", map[string]any{"author": remote.Id, "public": true, "iri": "https://remote.example/api/v1/trail/remote"})
	comment := f.record(t, "comments", map[string]any{"author": local.Id, "trail": remoteTrail.Id, "iri": "https://remote.example/api/v1/comment/foreign"})
	list.Set("iri", "https://remote.example/api/v1/list/foreign")
	for _, operation := range []func() error{
		func() error { return CreateCommentDeleteActivity(f.app, nil, comment) },
		func() error { return CreateListDeleteActivity(f.app, list, DeleteAudience{}) },
	} {
		if err := operation(); err != nil {
			t.Fatal(err)
		}
	}
	count, err = f.app.CountRecords("activitypub_activities")
	if err != nil || count != 3 {
		t.Fatalf("remote identity deletion signed: count=%d err=%v", count, err)
	}
	comment.Set("iri", "https://previous.example/api/v1/comment/"+comment.Id)
	if err := f.app.Save(comment); err != nil {
		t.Fatal(err)
	}
	if err := CreateCommentDeleteActivity(f.app, nil, comment); err != nil {
		t.Fatal(err)
	}
	activity := f.delivered(t, "remote")
	if activity["type"] != "Delete" || activity["object"] != comment.GetString("iri") {
		t.Fatalf("historical local comment did not reach the remote audience: %v", activity)
	}
}

func TestInboundDeleteCannotRemoveLocalOrForeignOriginRecords(t *testing.T) {
	f := setupAddressingTestApp(t)
	remote := f.actor(t, "remote", false)
	for _, collection := range []string{"trails", "lists", "summit_logs", "comments"} {
		for _, origin := range []string{"https://local.example", "https://foreign.example"} {
			record := f.record(t, collection, map[string]any{"author": remote.Id, "iri": origin + "/api/v1/" + map[string]string{"trails": "trail", "lists": "list", "summit_logs": "summit-log", "comments": "comment"}[collection] + "/private"})
			activity := pub.Activity{Type: pub.DeleteType, Actor: pub.IRI(remote.GetString("iri")), Object: pub.IRI(record.GetString("iri"))}
			var err error
			switch collection {
			case "trails":
				err = processDeleteTrailActivity(f.app, remote, activity)
			case "lists":
				err = processDeleteListActivity(f.app, remote, activity)
			case "summit_logs":
				err = processDeleteSummitLogActivity(f.app, remote, activity)
			case "comments":
				err = processDeleteCommentActivity(f.app, remote, activity)
			}
			if err == nil {
				t.Errorf("%s delete accepted for %s", collection, origin)
			}
			if _, lookupErr := f.app.FindRecordById(collection, record.Id); lookupErr != nil {
				t.Errorf("protected %s deleted: %v", collection, lookupErr)
			}
		}
	}
}
