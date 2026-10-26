package migrations

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func TestHidePrivateActivitySnapshots(t *testing.T) {
	app := core.NewBaseApp(core.BaseAppConfig{DataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.ResetBootstrapState() })

	subjects := map[string]*core.Collection{}
	for _, name := range []string{"trails", "lists", "comments", "summit_logs"} {
		c := core.NewBaseCollection(name)
		c.Fields.Add(&core.TextField{Name: "iri"}, &core.BoolField{Name: "public"})
		if err := app.Save(c); err != nil {
			t.Fatal(err)
		}
		subjects[name] = c
	}
	activities := core.NewBaseCollection("activitypub_activities")
	activities.ListRule = types.Pointer("")
	activities.ViewRule = types.Pointer("")
	activities.Fields.Add(&core.TextField{Name: "type"}, &core.JSONField{Name: "object"})
	if err := app.Save(activities); err != nil {
		t.Fatal(err)
	}

	subject := func(collection, iri string, public bool) {
		r := core.NewRecord(subjects[collection])
		r.Set("iri", iri)
		r.Set("public", public)
		if err := app.Save(r); err != nil {
			t.Fatal(err)
		}
	}
	const (
		privateTrail = "https://local.example/api/v1/trail/private"
		publicTrail  = "https://local.example/api/v1/trail/public"
		privateList  = "https://local.example/api/v1/list/private"
		comment      = "https://local.example/api/v1/comment/live"
	)
	subject("trails", privateTrail, false)
	subject("trails", publicTrail, true)
	subject("lists", privateList, false)
	subject("comments", comment, true)

	activity := func(typ string, object any) string {
		r := core.NewRecord(activities)
		r.Set("type", typ)
		r.Set("object", object)
		if err := app.Save(r); err != nil {
			t.Fatal(err)
		}
		return r.Id
	}
	hidden := []string{
		activity("Create", map[string]any{"id": privateTrail}),
		activity("Announce", map[string]any{"id": privateTrail}),
		activity("Update", map[string]any{"id": privateList}),
		// A reply to the private trail, whether or not the reply still exists.
		activity("Create", map[string]any{"id": comment, "inReplyTo": privateTrail}),
	}
	visible := []string{
		activity("Create", map[string]any{"id": publicTrail}),
		activity("Create", map[string]any{"id": comment, "inReplyTo": publicTrail}),
		activity("Delete", privateTrail),
		activity("Like", publicTrail),
		// A summit log imported without an IRI was recorded with an empty id.
		activity("Create", map[string]any{"id": "", "inReplyTo": publicTrail}),
	}
	purged := []string{
		activity("Create", map[string]any{"id": "https://local.example/api/v1/comment/deleted", "inReplyTo": publicTrail}),
		activity("Update", map[string]any{"id": "https://local.example/api/v1/trail/deleted"}),
		activity("Announce", map[string]any{"id": "https://local.example/api/v1/list/deleted"}),
	}

	if err := hidePrivateActivitySnapshotsUp1791110004(app); err != nil {
		t.Fatal(err)
	}

	activities, err := app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		t.Fatal(err)
	}
	anonymous := &core.RequestInfo{}
	check := func(ids []string, wantHidden bool) {
		t.Helper()
		for _, id := range ids {
			r, err := app.FindRecordById(activities, id)
			if err != nil {
				t.Fatalf("activity %s should have been kept: %v", id, err)
			}
			if got := r.GetBool("hidden"); got != wantHidden {
				t.Errorf("%s %s: hidden = %v, want %v", r.GetString("type"), r.GetString("object"), got, wantHidden)
			}
			canView, err := app.CanAccessRecord(r, anonymous, activities.ViewRule)
			if err != nil {
				t.Fatal(err)
			}
			if canView == wantHidden {
				t.Errorf("%s %s: publicly viewable = %v", r.GetString("type"), r.GetString("object"), canView)
			}
		}
	}
	check(hidden, true)
	check(visible, false)
	for _, id := range purged {
		if _, err := app.FindRecordById(activities, id); err == nil {
			t.Errorf("activity %s of a deleted object survived", id)
		}
	}
	if activities.ListRule == nil || *activities.ListRule != "hidden = false" {
		t.Errorf("listRule = %v, want hidden = false", activities.ListRule)
	}
}
