package main

import (
	"errors"
	"net/http/httptest"
	"pocketbase/internal/srch0"
	"pocketbase/util"
	"strings"
	"testing"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func TestSRCH0Mutation(t *testing.T) {
	t.Setenv("ORIGIN", "https://wanderer.invalid")
	// Federation delivery is outside these search observations. An absent key
	// makes its asynchronous transport exit before attempting any HTTP request.
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "")
	d := srch0.Data(t)
	for _, c := range srch0.Cases(t, "mutation", "go-mutation") {
		t.Run(c.ID, func(t *testing.T) {
			app := srch0.App(t, d)
			m := srch0.NewMeili(t)
			if extra, ok := c.Input["extra_records"].([]any); ok {
				for _, v := range extra {
					x := v.(map[string]any)
					collection, _ := app.FindCollectionByNameOrId(x["collection"].(string))
					r := core.NewRecord(collection)
					srch0.Assign(t, r, x["values"].(map[string]any))
					srch0.Save(t, app, r)
				}
			}
			for _, v := range c.Input["seed"].([]any) {
				x := v.(map[string]any)
				collection := x["collection"].(string)
				r := srch0.Record(t, app, collection, x["id"].(string))
				var err error
				switch collection {
				case "trails":
					err = util.IndexTrails(app, []*core.Record{r}, m.Client)
				case "lists":
					err = util.IndexLists(app, []*core.Record{r}, m.Client)
				case "activitypub_actors":
					err = util.IndexActors([]*core.Record{r}, m.Client)
				default:
					t.Fatalf("unsupported index collection %s", collection)
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			before := m.Snapshot()
			firstMutation := len(m.Calls())
			// Exercise the application's actual registry and saved relations.
			setupEventHandlers(&pocketbase.PocketBase{App: app}, m.Client)
			for _, v := range c.Input["operations"].([]any) {
				x := v.(map[string]any)
				collection := x["collection"].(string)
				action := x["action"].(string)
				var r *core.Record
				if action == "create" || action == "share-create" {
					coll, _ := app.FindCollectionByNameOrId(collection)
					r = core.NewRecord(coll)
					if source, ok := x["source_id"].(string); ok {
						original := srch0.Record(t, app, collection, source)
						values := original.FieldsData()
						delete(values, "id")
						srch0.Assign(t, r, values)
					}
					r.Id = srch0.ID(x["id"].(string))
				} else {
					r = srch0.Record(t, app, collection, x["id"].(string))
				}
				if values, ok := x["set"].(map[string]any); ok {
					srch0.Assign(t, r, values)
				}
				var err error
				switch action {
				case "create", "save":
					err = app.Save(r)
				case "delete":
					err = app.Delete(r)
				case "share-create", "share-delete":
					e := &core.RecordRequestEvent{RequestEvent: &core.RequestEvent{App: app}, Record: r}
					e.Collection = r.Collection()
					e.Request = httptest.NewRequest("POST", "/api/collections/"+collection+"/records", nil)
					e.Response = httptest.NewRecorder()
					if action == "share-create" {
						err = app.OnRecordCreateRequest(collection).Trigger(e, func(e *core.RecordRequestEvent) error { return e.App.Save(e.Record) })
					} else {
						err = app.OnRecordDeleteRequest(collection).Trigger(e, func(e *core.RecordRequestEvent) error { return e.App.Delete(e.Record) })
					}
				default:
					t.Fatalf("unknown mutation action %s", action)
				}
				if c.Input["record_boundary"] == "defensive" {
					status := "accepted"
					if err != nil {
						status = "error"
						var fields validation.Errors
						if errors.As(err, &fields) && fields["difficulty"] != nil {
							status = "rejected"
						}
					}
					persisted := srch0.Record(t, app, collection, r.Id).GetString("name") == r.GetString("name")
					srch0.Assert(t, c, map[string]any{"validation": status, "engine_requests": len(m.Calls()) - firstMutation, "persisted": persisted}, c.Observed["diagnostics"])
					m.AssertRealMaterialization(t, c)
					return
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			calls := m.Calls()[firstMutation:]
			for i := range calls {
				if strings.HasPrefix(calls[i].Path, "/tasks/") {
					calls[i].Path = "/tasks/{task_uid}"
				}
			}
			got := map[string]any{"before": before, "mutation": c.Input["operations"], "engine_requests": calls, "after": m.Snapshot()}
			if probes, ok := c.Input["database_probes"].([]any); ok {
				values := map[string]any{}
				for _, v := range probes {
					p := v.(map[string]any)
					r := srch0.Record(t, app, p["collection"].(string), p["id"].(string))
					fields := map[string]any{}
					for _, f := range p["fields"].([]any) {
						fields[f.(string)] = r.Get(f.(string))
					}
					values[p["collection"].(string)+"/"+p["id"].(string)] = fields
				}
				got["database"] = values
			}
			assertSRCH0CurrentIndex(t, app, m, before)
			for _, operation := range c.Input["operations"].([]any) {
				x := operation.(map[string]any)
				if x["action"] == "save" && (x["collection"] == "activitypub_actors" || x["collection"] == "tags" || x["collection"] == "categories") {
					assertSRCH0NewMetadataDocumentsComplete(t, m, before)
					assertSRCH0CurrentIndex(t, app, m, m.Snapshot())
				}
			}
			srch0.Assert(t, c, got, c.Observed["mutation_state"])
			m.AssertRealMaterialization(t, c)
		})
	}
}
