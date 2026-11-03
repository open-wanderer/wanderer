package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/federation"
	"pocketbase/util"
)

const instanceActorPathForTest = "/api/v1/activitypub/instance"

// seedLocalInstanceActor stores a local instance actor the way an older build
// would have, with fixed keys so tests can prove they are never rewritten.
func seedLocalInstanceActor(t *testing.T, app core.App, iri, domain, name string) *core.Record {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(collection)
	record.Set("actor_type", "instance")
	record.Set("preferred_username", name)
	record.Set("username", "Wanderer at "+domain)
	record.Set("domain", domain)
	record.Set("iri", iri)
	record.Set("inbox", iri+"/inbox")
	record.Set("outbox", iri+"/outbox")
	record.Set("is_local", true)
	record.Set("public_key", "PUB-KEY-1")
	record.Set("private_key", "PRIV-1")
	record.Set("last_fetched", time.Now())
	if err := app.Save(record); err != nil {
		t.Fatalf("seed local instance actor %s: %v", iri, err)
	}
	return record
}

// seedRemoteInstancePeer stores a remote instance actor on host and an accepted
// follows row from that peer to the local instance actor.
func seedRemoteInstancePeer(t *testing.T, app core.App, host string, local *core.Record) *core.Record {
	t.Helper()
	actors, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		t.Fatal(err)
	}
	iri := "https://" + host + instanceActorPathForTest
	peer := core.NewRecord(actors)
	peer.Set("actor_type", "instance")
	peer.Set("preferred_username", "-instance")
	peer.Set("username", "Wanderer at "+host)
	peer.Set("domain", host)
	peer.Set("iri", iri)
	peer.Set("inbox", iri+"/inbox")
	peer.Set("outbox", iri+"/outbox")
	peer.Set("is_local", false)
	peer.Set("public_key", "PEER-PUB")
	peer.Set("last_fetched", time.Now())
	if err := app.Save(peer); err != nil {
		t.Fatalf("seed remote instance peer %s: %v", host, err)
	}
	follows, err := app.FindCollectionByNameOrId("follows")
	if err != nil {
		t.Fatal(err)
	}
	follow := core.NewRecord(follows)
	follow.Set("follower", peer.Id)
	follow.Set("followee", local.Id)
	follow.Set("status", "accepted")
	if err := app.Save(follow); err != nil {
		t.Fatalf("seed accepted follow from %s: %v", host, err)
	}
	return peer
}

// localInstanceActors returns every local instance actor, oldest first.
func localInstanceActors(t *testing.T, app core.App) []*core.Record {
	t.Helper()
	records, err := app.FindRecordsByFilter("activitypub_actors", "actor_type = 'instance' && is_local = true", "@rowid", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	return records
}

func TestInstanceActorFollowsOriginInPlace(t *testing.T) {
	t.Run("BranchInstallMovesToNewHost", func(t *testing.T) {
		app := newRealSchemaApp(t)
		t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
		local := seedLocalInstanceActor(t, app, "http://localhost:3000"+instanceActorPathForTest, "localhost", "instance")
		peer := seedRemoteInstancePeer(t, app, "peer.example", local)
		follows, err := app.FindRecordsByFilter("follows", "followee = {:id}", "@rowid", 0, 0, map[string]any{"id": local.Id})
		if err != nil || len(follows) != 1 {
			t.Fatalf("seeded follows = %d, err %v", len(follows), err)
		}
		followBefore := follows[0]

		t.Setenv("ORIGIN", "https://trails.example.com")
		if err := federation.InitInstanceActor(app); err != nil {
			t.Fatalf("InitInstanceActor: %v", err)
		}

		actors := localInstanceActors(t, app)
		if len(actors) != 1 {
			t.Fatalf("local instance actors = %d, want 1", len(actors))
		}
		got := actors[0]
		wantIRI := "https://trails.example.com" + instanceActorPathForTest
		for field, want := range map[string]string{
			"iri":                wantIRI,
			"inbox":              wantIRI + "/inbox",
			"outbox":             wantIRI + "/outbox",
			"domain":             "trails.example.com",
			"username":           "Wanderer at trails.example.com",
			"preferred_username": "-instance",
			"public_key":         "PUB-KEY-1",
			"private_key":        "PRIV-1",
		} {
			if g := got.GetString(field); g != want {
				t.Errorf("%s = %q, want %q", field, g, want)
			}
		}
		if got.Id != local.Id {
			t.Errorf("record id changed from %s to %s", local.Id, got.Id)
		}

		ok, err := federation.IsAcceptedPeerHost(app, "https://"+peer.GetString("domain")+"/api/v1/activitypub/user/bob")
		if err != nil || !ok {
			t.Errorf("IsAcceptedPeerHost = %v, %v; want true", ok, err)
		}

		followsAfter, err := app.FindRecordsByFilter("follows", "followee = {:id}", "@rowid", 0, 0, map[string]any{"id": got.Id})
		if err != nil || len(followsAfter) != 1 {
			t.Fatalf("follows after = %d, err %v", len(followsAfter), err)
		}
		if followsAfter[0].Id != followBefore.Id || followsAfter[0].GetString("status") != "accepted" ||
			followsAfter[0].GetString("follower") != peer.Id {
			t.Errorf("follows row changed: %v", followsAfter[0].FieldsData())
		}
	})

	t.Run("HttpToHttpsSameHost", func(t *testing.T) {
		app := newRealSchemaApp(t)
		t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
		local := seedLocalInstanceActor(t, app, "http://localhost:3000"+instanceActorPathForTest, "localhost", "instance")

		for _, origin := range []string{"https://trails.example.com", "http://trails.example.com", "https://trails.example.com"} {
			t.Setenv("ORIGIN", origin)
			if err := federation.InitInstanceActor(app); err != nil {
				t.Fatalf("InitInstanceActor with ORIGIN %s: %v", origin, err)
			}
			actors := localInstanceActors(t, app)
			if len(actors) != 1 || actors[0].Id != local.Id {
				t.Fatalf("ORIGIN %s: local instance actors = %d (id %v), want the original one", origin, len(actors), actors)
			}
			got := actors[0]
			if want := origin + instanceActorPathForTest; got.GetString("iri") != want {
				t.Errorf("ORIGIN %s: iri = %q, want %q", origin, got.GetString("iri"), want)
			}
			if got.GetString("public_key") != "PUB-KEY-1" || got.GetString("private_key") != "PRIV-1" {
				t.Errorf("ORIGIN %s: key pair changed", origin)
			}
		}
	})
}

func TestInstanceActorFreshInstallCreatesOne(t *testing.T) {
	app := newRealSchemaApp(t)
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	if err := federation.InitInstanceActor(app); err != nil {
		t.Fatalf("InitInstanceActor: %v", err)
	}
	actors := localInstanceActors(t, app)
	if len(actors) != 1 {
		t.Fatalf("local instance actors = %d, want 1", len(actors))
	}
	first := actors[0]
	if got := first.GetString("preferred_username"); got != "-instance" {
		t.Errorf("preferred_username = %q, want -instance", got)
	}
	if !strings.HasPrefix(first.GetString("public_key"), "-----BEGIN PUBLIC KEY-----") {
		t.Errorf("public_key is not PEM: %q", first.GetString("public_key"))
	}

	time.Sleep(10 * time.Millisecond)
	if err := federation.InitInstanceActor(app); err != nil {
		t.Fatalf("second InitInstanceActor: %v", err)
	}
	actors = localInstanceActors(t, app)
	if len(actors) != 1 {
		t.Fatalf("local instance actors after second run = %d, want 1", len(actors))
	}
	if before, after := first.GetString("updated"), actors[0].GetString("updated"); before != after {
		t.Errorf("updated changed from %q to %q: an unchanged run must not save", before, after)
	}
}

func TestPeerLookupUsesStoredInstanceActor(t *testing.T) {
	app := newRealSchemaApp(t)
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	local := seedLocalInstanceActor(t, app, "http://localhost:3000"+instanceActorPathForTest, "localhost", "instance")
	seedRemoteInstancePeer(t, app, "peer.example", local)

	// ORIGIN moved but InitInstanceActor has not run (or was stopped by a
	// conflict): the stored row is still on its old IRI.
	t.Setenv("ORIGIN", "https://trails.example.com")

	ok, err := federation.IsAcceptedPeerHost(app, "https://peer.example/api/v1/activitypub/user/bob")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("IsAcceptedPeerHost = false, want true: the lookup must use the stored instance actor, not an IRI built from ORIGIN")
	}
}

// capturingLogApp wraps app so InitInstanceActor and the finder log into buf.
func capturingLogApp(app core.App, buf *bytes.Buffer) core.App {
	logger := slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	return &searchInitLogApp{App: app, logger: logger}
}

func TestDuplicateLocalInstanceActorsUseOldest(t *testing.T) {
	app := newRealSchemaApp(t) // ORIGIN https://example.com
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	a := seedLocalInstanceActor(t, app, "http://localhost:3000"+instanceActorPathForTest, "localhost", "instance")
	b := seedLocalInstanceActor(t, app, "http://127.0.0.1:8090"+instanceActorPathForTest, "127.0.0.1", "instance")

	var buf bytes.Buffer
	if err := federation.InitInstanceActor(capturingLogApp(app, &buf)); err != nil {
		t.Fatalf("InitInstanceActor: %v", err)
	}

	actors := localInstanceActors(t, app)
	if len(actors) != 2 {
		t.Fatalf("local instance actors = %d, want 2 (no third, no merge)", len(actors))
	}
	if actors[0].Id != a.Id || actors[1].Id != b.Id {
		t.Fatalf("order changed: got %s, %s; want %s, %s", actors[0].Id, actors[1].Id, a.Id, b.Id)
	}
	if got, want := actors[0].GetString("iri"), "https://example.com"+instanceActorPathForTest; got != want {
		t.Errorf("older actor iri = %q, want %q", got, want)
	}
	if got := actors[0].GetString("preferred_username"); got != "-instance" {
		t.Errorf("older actor preferred_username = %q, want -instance", got)
	}
	if got, want := actors[1].GetString("iri"), "http://127.0.0.1:8090"+instanceActorPathForTest; got != want {
		t.Errorf("newer actor iri = %q, want unchanged %q", got, want)
	}
	if got := actors[1].GetString("preferred_username"); got != "instance" {
		t.Errorf("newer actor preferred_username = %q, want unchanged instance", got)
	}

	logs := buf.String()
	for _, want := range []string{"level=ERROR", "more than one local instance actor", a.Id} {
		if !strings.Contains(logs, want) {
			t.Errorf("log lacks %q:\n%s", want, logs)
		}
	}

	for i := 0; i < 2; i++ {
		got, err := federation.FindLocalInstanceActor(app)
		if err != nil {
			t.Fatal(err)
		}
		if got.Id != a.Id {
			t.Errorf("call %d: FindLocalInstanceActor = %s, want the oldest %s", i, got.Id, a.Id)
		}
	}
}

func TestDuplicateInstanceActorOwningOriginIRIFailsLoudly(t *testing.T) {
	app := newRealSchemaApp(t) // ORIGIN https://example.com
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	a := seedLocalInstanceActor(t, app, "http://localhost:3000"+instanceActorPathForTest, "localhost", "instance")
	b := seedLocalInstanceActor(t, app, "https://example.com"+instanceActorPathForTest, "example.com", "instance")

	var buf bytes.Buffer
	err := federation.InitInstanceActor(capturingLogApp(app, &buf))
	if err == nil {
		t.Fatal("InitInstanceActor returned nil, want an error: the newer sibling owns the target IRI")
	}
	if !strings.Contains(err.Error(), a.Id) {
		t.Errorf("error %q does not name the updated actor %s", err, a.Id)
	}

	var conflictLine string
	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.Contains(line, "another actor owns its IRI") {
			conflictLine = line
		}
	}
	if conflictLine == "" {
		t.Fatalf("no conflict log line:\n%s", buf.String())
	}
	for _, want := range []string{"level=ERROR", a.Id, b.Id} {
		if !strings.Contains(conflictLine, want) {
			t.Errorf("conflict log line lacks %q: %s", want, conflictLine)
		}
	}

	actors := localInstanceActors(t, app)
	if len(actors) != 2 {
		t.Fatalf("local instance actors = %d, want 2", len(actors))
	}
	for i, want := range []struct{ iri, domain, name string }{
		{"http://localhost:3000" + instanceActorPathForTest, "localhost", "instance"},
		{"https://example.com" + instanceActorPathForTest, "example.com", "instance"},
	} {
		got := actors[i]
		if got.GetString("iri") != want.iri || got.GetString("domain") != want.domain || got.GetString("preferred_username") != want.name {
			t.Errorf("row %d changed: iri %q, domain %q, name %q", i, got.GetString("iri"), got.GetString("domain"), got.GetString("preferred_username"))
		}
	}
}

// newInstanceUser creates a user, its settings row and its person actor, the
// way signup does.
func newInstanceUser(t *testing.T, app core.App, username, email string) (user, actor *core.Record) {
	t.Helper()
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	user = core.NewRecord(users)
	user.Set("username", username)
	user.Set("email", email)
	user.SetPassword("instance-user-test-password")
	if err := app.Save(user); err != nil {
		t.Fatalf("save user %q: %v", username, err)
	}
	settingsCollection, err := app.FindCollectionByNameOrId("settings")
	if err != nil {
		t.Fatal(err)
	}
	settings := core.NewRecord(settingsCollection)
	settings.Set("user", user.Id)
	if err := app.Save(settings); err != nil {
		t.Fatalf("save settings for %q: %v", username, err)
	}
	actor, err = util.ActorFromUser(app, user)
	if err != nil {
		t.Fatalf("ActorFromUser %q: %v", username, err)
	}
	return user, actor
}

func TestUserNamedInstanceDoesNotBlockInstanceActor(t *testing.T) {
	t.Run("UserFirst", func(t *testing.T) {
		app := newRealSchemaApp(t)
		t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
		user, actor := newInstanceUser(t, app, "Instance", "instance@example.test")

		if err := federation.InitInstanceActor(app); err != nil {
			t.Fatalf("InitInstanceActor with a user named Instance present: %v", err)
		}

		actors := localInstanceActors(t, app)
		if len(actors) != 1 {
			t.Fatalf("local instance actors = %d, want 1", len(actors))
		}
		if got := actors[0].GetString("preferred_username"); got != federation.InstanceActorUsername {
			t.Errorf("instance actor preferred_username = %q, want %q", got, federation.InstanceActorUsername)
		}
		gotUser, err := app.FindRecordById("users", user.Id)
		if err != nil {
			t.Fatal(err)
		}
		if got := gotUser.GetString("username"); got != "Instance" {
			t.Errorf("user username = %q, want Instance (unchanged)", got)
		}
		gotActor, err := app.FindRecordById("activitypub_actors", actor.Id)
		if err != nil {
			t.Fatalf("person actor gone: %v", err)
		}
		if gotActor.GetString("preferred_username") != "instance" || gotActor.GetString("domain") != "example.com" ||
			gotActor.GetString("actor_type") != "person" {
			t.Errorf("person actor changed: %q on %q (%s)", gotActor.GetString("preferred_username"),
				gotActor.GetString("domain"), gotActor.GetString("actor_type"))
		}
	})

	t.Run("InstanceActorFirst", func(t *testing.T) {
		app := newRealSchemaApp(t)
		t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
		if err := federation.InitInstanceActor(app); err != nil {
			t.Fatalf("InitInstanceActor: %v", err)
		}
		_, actor := newInstanceUser(t, app, "instance", "instance@example.test")
		if actor == nil || actor.Id == "" {
			t.Fatal("person actor was not saved")
		}
		if got := len(localInstanceActors(t, app)); got != 1 {
			t.Errorf("local instance actors = %d, want 1", got)
		}
	})
}

func TestInstanceActorUsernameIsNotRegistrable(t *testing.T) {
	app := newRealSchemaApp(t)
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	save := func(username, email string) error {
		user := core.NewRecord(users)
		user.Set("username", username)
		user.Set("email", email)
		user.SetPassword("instance-user-test-password")
		return app.Save(user)
	}
	for i, name := range []string{"-instance", "-Instance"} {
		err := save(name, "reserved"+string(rune('a'+i))+"@example.test")
		var verrs validation.Errors
		if !errors.As(err, &verrs) {
			t.Fatalf("saving user %q: err = %v, want a validation error", name, err)
		}
		if _, ok := verrs["username"]; !ok {
			t.Errorf("saving user %q: validation errors %v lack the username field", name, verrs)
		}
	}
	for i, name := range []string{"instance-x", "instance"} {
		if err := save(name, "free"+string(rune('a'+i))+"@example.test"); err != nil {
			t.Errorf("saving user %q: %v", name, err)
		}
	}
}

func TestHandleLookupsNeverResolveInstanceActor(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()

	check := func(t *testing.T, app core.App, person *core.Record) {
		t.Helper()
		for _, handle := range []string{"instance", "Instance@example.com", "@INSTANCE@example.com"} {
			got, err := federation.GetActorByHandle(app, cancelled, handle, false)
			if err != nil {
				t.Errorf("GetActorByHandle(%q): %v", handle, err)
				continue
			}
			if got.Id != person.Id {
				t.Errorf("GetActorByHandle(%q) = %s (%s), want the person actor %s", handle, got.Id,
					got.GetString("actor_type"), person.Id)
			}
		}
		for _, handle := range []string{"-instance", "-instance@example.com", "@-INSTANCE@example.com", ""} {
			got, err := federation.GetActorByHandle(app, cancelled, handle, false)
			if !errors.Is(err, federation.ErrActorNotFound) {
				id := ""
				if got != nil {
					id = got.Id
				}
				t.Errorf("GetActorByHandle(%q) = %q, %v; want ErrActorNotFound", handle, id, err)
			}
		}

		actors, err := app.FindCollectionByNameOrId("activitypub_actors")
		if err != nil {
			t.Fatal(err)
		}
		remote := core.NewRecord(actors)
		remote.Set("actor_type", "instance")
		remote.Set("username", "Wanderer at peer.example")
		remote.Set("preferred_username", "instance")
		remote.Set("domain", "peer.example")
		remote.Set("iri", "https://peer.example"+instanceActorPathForTest)
		remote.Set("inbox", "https://peer.example"+instanceActorPathForTest+"/inbox")
		remote.Set("is_local", false)
		remote.Set("public_key", "PEER-PUB")
		remote.Set("last_fetched", time.Now())
		if err := app.Save(remote); err != nil {
			t.Fatalf("store remote instance actor: %v", err)
		}
		if got, err := federation.GetActorByHandle(app, cancelled, "instance@peer.example", false); err == nil && got.Id == remote.Id {
			t.Errorf("GetActorByHandle(instance@peer.example) returned the stored remote instance actor %s", remote.Id)
		}

		mentions, err := federation.ActorsFromMentions(app, cancelled,
			`<p><a class="mention" href="https://example.com/-instance">@-instance</a> `+
				`<a class="mention" href="https://example.com/instance">@instance</a></p>`)
		if err != nil {
			t.Fatal(err)
		}
		if len(mentions) != 1 || mentions[0].Id != person.Id {
			t.Errorf("ActorsFromMentions returned %d actors, want exactly the person actor", len(mentions))
		}
		if err := app.Delete(remote); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("UserFirst", func(t *testing.T) {
		app := newRealSchemaApp(t)
		t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
		_, person := newInstanceUser(t, app, "Instance", "instance@example.test")
		if err := federation.InitInstanceActor(app); err != nil {
			t.Fatalf("InitInstanceActor: %v", err)
		}
		check(t, app, person)
	})
	t.Run("InstanceActorFirst", func(t *testing.T) {
		app := newRealSchemaApp(t)
		t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
		if err := federation.InitInstanceActor(app); err != nil {
			t.Fatalf("InitInstanceActor: %v", err)
		}
		_, person := newInstanceUser(t, app, "instance", "instance@example.test")
		check(t, app, person)
	})
}
