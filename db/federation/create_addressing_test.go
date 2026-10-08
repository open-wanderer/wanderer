package federation

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"pocketbase/util"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/security"
)

// The object carries the activity's to/cc (GoToSocial reads visibility from
// it); the activity itself is addressed as before.

const publicIRI = "https://www.w3.org/ns/activitystreams#Public"

type addressingFixture struct {
	app *pbtests.TestApp
	in  *inboxes
}

func setupAddressingTestApp(t *testing.T) *addressingFixture {
	t.Helper()

	t.Setenv("ORIGIN", "https://local.example")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", summitLogTestKey)

	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)

	add := func(name string, fields ...core.Field) *core.Collection {
		c := core.NewBaseCollection(name)
		c.Fields.Add(fields...)
		if err := app.Save(c); err != nil {
			t.Fatal(err)
		}
		return c
	}
	text := func(names ...string) []core.Field {
		fields := []core.Field{}
		for _, n := range names {
			fields = append(fields, &core.TextField{Name: n})
		}
		return fields
	}

	actors := add("activitypub_actors", append(text("iri", "inbox", "followers", "preferred_username", "domain", "private_key"),
		&core.BoolField{Name: "is_local"})...)
	rel := func(name, collectionId string) core.Field {
		return &core.RelationField{Name: name, CollectionId: collectionId, MaxSelect: 1}
	}
	add("follows", rel("follower", actors.Id), rel("followee", actors.Id), &core.TextField{Name: "status"})
	categories := add("categories", text("name")...)
	tags := add("tags", text("name")...)
	trails := add("trails", append(text("iri", "name", "description"), &core.BoolField{Name: "public"},
		rel("author", actors.Id), rel("category", categories.Id), rel("subcategory", categories.Id),
		&core.RelationField{Name: "tags", CollectionId: tags.Id, MaxSelect: 99})...)
	add("lists", append(text("iri", "name", "description"), &core.BoolField{Name: "public"}, rel("author", actors.Id))...)
	add("comments", append(text("iri", "text"), rel("author", actors.Id), rel("trail", trails.Id))...)
	add("summit_logs", append(text("iri", "text"), rel("author", actors.Id), rel("trail", trails.Id))...)
	add("activitypub_activities", append(text("iri", "type", "actor", "published"),
		&core.JSONField{Name: "to"}, &core.JSONField{Name: "cc"}, &core.JSONField{Name: "object"})...)

	return &addressingFixture{app: app, in: newInboxes(t)}
}

// actor is local (can sign) or remote (inbox on the test server).
func (f *addressingFixture) actor(t *testing.T, name string, local bool) *core.Record {
	t.Helper()

	iri := "https://remote.example/users/" + name
	inbox := f.in.url(name)
	if local {
		iri = "https://local.example/api/v1/activitypub/user/" + name
		inbox = iri + "/inbox"
	}
	r := f.record(t, "activitypub_actors", map[string]any{
		"iri": iri, "inbox": inbox, "followers": iri + "/followers", "preferred_username": name, "is_local": local,
	})
	if local {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatal(err)
		}
		encrypted, err := security.Encrypt(x509.MarshalPKCS1PrivateKey(key), summitLogTestKey)
		if err != nil {
			t.Fatal(err)
		}
		r.Set("private_key", encrypted)
		if err := f.app.Save(r); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

func (f *addressingFixture) follow(t *testing.T, follower, followee *core.Record) {
	f.record(t, "follows", map[string]any{"follower": follower.Id, "followee": followee.Id, "status": "accepted"})
}

// record saves a record; content (path != "") gets its IRI like the hooks do.
func (f *addressingFixture) record(t *testing.T, collection string, fields map[string]any, path ...string) *core.Record {
	t.Helper()

	c, err := f.app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatal(err)
	}
	r := core.NewRecord(c)
	r.Load(fields)
	if err := f.app.Save(r); err != nil {
		t.Fatal(err)
	}
	if len(path) > 0 {
		r.Set("iri", "https://local.example/api/v1/"+path[0]+"/"+r.Id)
		if err := f.app.Save(r); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

// delivered returns the activity delivered to name's inbox.
func (f *addressingFixture) delivered(t *testing.T, name string) map[string]any {
	t.Helper()

	if f.in.wait(name, 1, 5*time.Second) < 1 {
		t.Fatalf("nothing delivered to %s", name)
	}
	var activity map[string]any
	if err := json.Unmarshal(f.in.body(name), &activity); err != nil {
		t.Fatal(err)
	}
	return activity
}

// iris reads to/cc, serialized as a single IRI or an array.
func iris(v any) []string {
	switch x := v.(type) {
	case string:
		return []string{x}
	case []any:
		out := []string{}
		for _, e := range x {
			s, _ := e.(string)
			out = append(out, s)
		}
		return out
	}
	return nil
}

func assertAddressed(t *testing.T, activity map[string]any, to, cc []string) {
	t.Helper()

	object, _ := activity["object"].(map[string]any)
	for name, v := range map[string]map[string]any{"activity": activity, "object": object} {
		if got := iris(v["to"]); !reflect.DeepEqual(got, to) {
			t.Errorf("%s to = %v, want %v", name, got, to)
		}
		if got := iris(v["cc"]); !reflect.DeepEqual(got, cc) {
			t.Errorf("%s cc = %v, want %v", name, got, cc)
		}
	}
}

func TestCreateActivitiesAddressObject(t *testing.T) {
	ctx := context.Background()

	t.Run("trail", func(t *testing.T) {
		f := setupAddressingTestApp(t)
		author := f.actor(t, "author", true)
		f.follow(t, f.actor(t, "follower", false), author)
		trail := f.record(t, "trails", map[string]any{"author": author.Id, "public": true, "name": "Tour"}, "trail")

		if err := CreateTrailActivity(f.app, ctx, trail, pub.CreateType); err != nil {
			t.Fatal(err)
		}
		assertAddressed(t, f.delivered(t, "follower"), []string{publicIRI}, []string{author.GetString("followers")})
	})

	t.Run("comment", func(t *testing.T) {
		f := setupAddressingTestApp(t)
		author := f.actor(t, "author", true)
		trailAuthor := f.actor(t, "trailauthor", false)
		trail := f.record(t, "trails", map[string]any{"author": trailAuthor.Id, "public": true}, "trail")
		comment := f.record(t, "comments", map[string]any{"author": author.Id, "trail": trail.Id, "text": "toll"}, "comment")

		if err := CreateCommentActivity(f.app, ctx, comment, pub.CreateType); err != nil {
			t.Fatal(err)
		}
		assertAddressed(t, f.delivered(t, "trailauthor"), []string{publicIRI}, []string{trailAuthor.GetString("inbox")})
	})

	t.Run("summit log", func(t *testing.T) {
		f := setupAddressingTestApp(t)
		author := f.actor(t, "author", true)
		trailAuthor := f.actor(t, "trailauthor", false)
		f.follow(t, f.actor(t, "follower", false), author)
		trail := f.record(t, "trails", map[string]any{"author": trailAuthor.Id, "public": true}, "trail")
		log := f.record(t, "summit_logs", map[string]any{"author": author.Id, "trail": trail.Id, "text": "oben"}, "summit-log")

		if err := CreateSummitLogActivity(f.app, ctx, log, pub.CreateType); err != nil {
			t.Fatal(err)
		}
		assertAddressed(t, f.delivered(t, "follower"),
			[]string{publicIRI, trailAuthor.GetString("iri")}, []string{author.GetString("followers")})
	})

	t.Run("list", func(t *testing.T) {
		f := setupAddressingTestApp(t)
		author := f.actor(t, "author", true)
		f.follow(t, f.actor(t, "follower", false), author)
		list := f.record(t, "lists", map[string]any{"author": author.Id, "public": true, "name": "Touren"}, "list")

		if err := CreateListActivity(f.app, list, pub.CreateType); err != nil {
			t.Fatal(err)
		}
		assertAddressed(t, f.delivered(t, "follower"), []string{publicIRI}, []string{author.GetString("followers")})
	})
}

func TestFetchedObjectsArePublic(t *testing.T) {
	f := setupAddressingTestApp(t)
	author := f.actor(t, "author", true)
	trail := f.record(t, "trails", map[string]any{"author": author.Id, "public": true}, "trail")
	comment := f.record(t, "comments", map[string]any{"author": author.Id, "trail": trail.Id}, "comment")

	trailObject, err := util.ObjectFromTrail(f.app, trail, nil)
	if err != nil {
		t.Fatal(err)
	}
	commentObject, err := util.ObjectFromComment(f.app, comment, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, object := range []*pub.Object{trailObject, commentObject} {
		body, _ := json.Marshal(object)
		var m map[string]any
		_ = json.Unmarshal(body, &m)
		if !reflect.DeepEqual(iris(m["to"]), []string{publicIRI}) || !reflect.DeepEqual(iris(m["cc"]), []string{author.GetString("followers")}) {
			t.Errorf("%s: to = %v, cc = %v", object.ID, m["to"], m["cc"])
		}
	}
}
