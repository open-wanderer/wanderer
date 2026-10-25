package federation

import (
	"fmt"
	"strings"
	"testing"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/util"
)

func TestLocalActorActivitiesResolveStoredObjects(t *testing.T) {
	for _, collection := range []string{"trails", "lists"} {
		kind := map[string]string{"trails": "trail", "lists": "list"}[collection]
		for _, test := range []struct {
			name        string
			pattern     string
			otherAuthor bool
			missing     bool
			wantError   bool
		}{
			{name: "alternate-host-other-id", pattern: "https://b.example/api/v1/%s/aaaaaaaaaaaaaaa"},
			{name: "alternate-host-own-id", pattern: "https://b.example/api/v1/%s/{id}"},
			{name: "encoded-path-own-id", pattern: "https://b.example/wander%%20er/api/v1/%s/{id}"},
			{name: "query-own-id", pattern: "https://b.example/api/v1/%s/{id}?v=1"},
			{name: "missing-record", pattern: "https://b.example/api/v1/%s/unknown", missing: true, wantError: true},
			{name: "other-author", pattern: "https://b.example/api/v1/%s/{id}", otherAuthor: true, wantError: true},
			{name: "instance-path", pattern: "https://a.example/api/v1/%s/{id}"},
			{name: "instance-path-other-author", pattern: "https://a.example/api/v1/%s/{id}", otherAuthor: true},
		} {
			t.Run(collection+"/"+test.name, func(t *testing.T) {
				f := setupAddressingTestApp(t)
				t.Setenv("ORIGIN", "https://a.example")
				owner := f.record(t, "activitypub_actors", map[string]any{
					"iri": "https://a.example/api/v1/activitypub/user/owner", "is_local": true,
				})
				author := owner
				if test.otherAuthor {
					author = f.record(t, "activitypub_actors", map[string]any{
						"iri": "https://a.example/api/v1/activitypub/user/other", "is_local": true,
					})
				}
				record := f.record(t, collection, map[string]any{"author": author.Id, "name": "Original", "public": false})
				iri := strings.ReplaceAll(fmt.Sprintf(test.pattern, kind), "{id}", record.Id)
				if !test.missing {
					record.Set("iri", iri)
					if err := f.app.Save(record); err != nil {
						t.Fatal(err)
					}
				}
				writes := 0
				countWrite := func(e *core.RecordEvent) error {
					writes++
					return e.Next()
				}
				f.app.OnRecordCreate(collection).BindFunc(countWrite)
				f.app.OnRecordUpdate(collection).BindFunc(countWrite)
				object := pub.Object{
					ID: pub.IRI(iri), Type: pub.NoteType, Tag: pub.ItemCollection{}, Location: &pub.Place{Type: pub.PlaceType},
					Name: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Changed")),
				}
				activity := pub.Activity{Type: pub.UpdateType, Object: &object, Actor: pub.IRI(owner.GetString("iri"))}
				var got *core.Record
				var err error
				if collection == "trails" {
					got, err = util.TrailFromActivity(activity, f.app, owner)
				} else {
					got, err = util.ListFromActivity(activity, f.app, owner)
				}
				if test.wantError {
					if err == nil || got != nil {
						t.Errorf("expected refusal, got record=%v err=%v", got, err)
					}
				} else if err != nil || got == nil || got.Id != record.Id {
					t.Errorf("stored record was not resolved: record=%v err=%v", got, err)
				}
				count, err := f.app.CountRecords(collection)
				stored, findErr := f.app.FindRecordById(collection, record.Id)
				if err != nil || findErr != nil {
					t.Fatalf("read stored records: count=%v record=%v", err, findErr)
				}
				if writes != 0 || count != 1 || stored.GetBool("public") || stored.GetString("name") != "Original" || stored.GetString("iri") != record.GetString("iri") {
					t.Fatalf("stored values changed: writes=%d count=%d public=%t name=%q iri=%q", writes, count, stored.GetBool("public"), stored.GetString("name"), stored.GetString("iri"))
				}
			})
		}
	}
}
