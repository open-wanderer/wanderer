package federation

import (
	"context"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"pocketbase/util"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
)

// These fixtures deliberately have no sanitizing model hook. Each external
// writer must prepare its content before PocketBase validates the record.
func TestIncomingActivitiesExplicitlyBoundRichText(t *testing.T) {
	for _, kind := range []string{"trail", "list", "comment", "summit-log"} {
		t.Run(kind, func(t *testing.T) {
			f := setupAddressingTestApp(t)
			author := f.actor(t, "author", false)
			object := pub.ObjectNew(pub.NoteType)
			object.ID = pub.IRI("https://remote.example/api/v1/" + kind + "/rich-text")
			object.Name = pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Remote content"))
			object.Content = pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef,
				`<p onclick="blocked()"><strong>Safe `+strings.Repeat(`'"&山🚲`, 3000)+`</strong></p>`))
			object.Location = &pub.Place{Name: pub.NaturalLanguageValuesNew(pub.LangRefValueNew(pub.NilLangRef, "Somewhere"))}
			object.StartTime = time.Now()
			if kind == "comment" || kind == "summit-log" {
				trail := f.record(t, "trails", map[string]any{"author": author.Id,
					"iri": "https://remote.example/api/v1/trail/parent", "public": true})
				object.InReplyTo = pub.IRI(trail.GetString("iri"))
			}
			activity := pub.ActivityNew(pub.IRI("https://remote.example/activity/rich-text"), pub.CreateType, object)
			var err error
			collection, field := "", "description"
			switch kind {
			case "trail":
				collection = "trails"
				_, err = util.TrailFromActivity(context.Background(), *activity, f.app, author)
			case "list":
				collection = "lists"
				_, err = util.ListFromActivity(context.Background(), *activity, f.app, author)
			case "comment":
				collection, field = "comments", "text"
				err = processCreateOrUpdateCommentActivity(context.Background(), *activity, f.app, author)
			case "summit-log":
				collection, field = "summit_logs", "text"
				err = processCreateOrUpdateSummitLogActivity(context.Background(), *activity, f.app, author)
			}
			if err != nil {
				t.Fatal(err)
			}
			record, err := f.app.FindFirstRecordByData(collection, "iri", object.ID.String())
			if err != nil {
				t.Fatal(err)
			}
			assertBoundedFederatedRichText(t, record.GetString(field), "Safe ")
		})
	}
}

func TestLocalBioProjectionExplicitlyBoundsActorSummary(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", summitLogTestKey)
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	add := func(name string, fields ...core.Field) *core.Collection {
		t.Helper()
		collection := core.NewBaseCollection(name)
		collection.Fields.Add(fields...)
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
		return collection
	}
	settingsCollection := add("settings", &core.TextField{Name: "user"}, &core.TextField{Name: "bio", Max: 10000}, &core.TextField{Name: "privacy"})
	add("activitypub_actors", &core.TextField{Name: "user"}, &core.TextField{Name: "summary"},
		&core.TextField{Name: "iri"}, &core.TextField{Name: "username"},
		&core.TextField{Name: "preferred_username"}, &core.TextField{Name: "domain"},
		&core.TextField{Name: "inbox"}, &core.TextField{Name: "outbox"},
		&core.TextField{Name: "followers"}, &core.TextField{Name: "following"},
		&core.TextField{Name: "public_key"}, &core.TextField{Name: "private_key"},
		&core.BoolField{Name: "is_local"}, &core.DateField{Name: "last_fetched"},
		&core.NumberField{Name: "follower_count"}, &core.NumberField{Name: "following_count"})
	add("follows", &core.TextField{Name: "follower"}, &core.TextField{Name: "followee"}, &core.TextField{Name: "status"})
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	users.Fields.Add(&core.TextField{Name: "username"})
	if err := app.Save(users); err != nil {
		t.Fatal(err)
	}
	user := core.NewRecord(users)
	user.SetEmail("local@example.test")
	user.SetPassword("test-password-1234")
	user.Set("username", "local-user")
	if err := app.Save(user); err != nil {
		t.Fatal(err)
	}
	settings := core.NewRecord(settingsCollection)
	settings.Set("user", user.Id)
	bio := `<p onclick="blocked()"><strong>Safe ` + strings.Repeat(`'"&山🚲`, 1500) + `</strong></p>`
	settings.Set("bio", bio)
	if err := app.Save(settings); err != nil {
		t.Fatal(err)
	}
	actor, err := util.ActorFromUser(app, user)
	if err != nil {
		t.Fatal(err)
	}
	assertBoundedFederatedRichText(t, actor.GetString("summary"), "Safe ")

	bio = strings.Replace(bio, "Safe ", "Updated ", 1)
	settings.Set("bio", bio)
	if err := app.Save(settings); err != nil {
		t.Fatal(err)
	}
	actor, err = assembleActor(app, context.Background(), actor, false)
	if err != nil {
		t.Fatal(err)
	}
	assertBoundedFederatedRichText(t, actor.GetString("summary"), "Updated ")
	storedSettings, err := app.FindRecordById(settingsCollection, settings.Id)
	if err != nil {
		t.Fatal(err)
	}
	if storedSettings.GetString("bio") != bio {
		t.Fatal("projecting the actor summary changed the original bio")
	}
}

func assertBoundedFederatedRichText(t *testing.T, value, prefix string) {
	t.Helper()
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > 5000 || strings.Contains(value, "blocked()") {
		t.Fatal("federated rich text is unsafe or exceeds the destination limit")
	}
	if !strings.HasPrefix(value, "<p><strong>"+prefix) || !strings.HasSuffix(value, "</strong></p>") {
		t.Fatal("federated rich text lost safe formatting or closing tags")
	}
	if !strings.Contains(value, `'"&amp;山🚲`) {
		t.Fatal("federated rich text unnecessarily escaped quotes or lost Unicode")
	}
}
