package routes

import (
	"context"
	"net/http"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/logger"
)

func TestRemoteSyncUsesRemoteAuthors(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for _, target := range []string{"summit_log", "trail"} {
		for _, expandedAuthor := range []bool{true, false} {
			name := target + "/expanded_author"
			if !expandedAuthor {
				name = target + "/inherited_author"
			}
			t.Run(name, func(t *testing.T) {
				f := newRemoteSyncFixture(t)
				parentAuthor := f.remoteActor.Id
				raw := map[string]any{"id": "remote-record", "name": "Remote", "text": "Remote", "public": true, "photos": []any{"remote.jpg"}}
				if expandedAuthor {
					raw["expand"] = map[string]any{"author": map[string]any{"iri": f.localActor.GetString("iri")}}
				} else {
					parentAuthor = f.localActor.Id
				}
				if target == "summit_log" {
					trail := f.save(f.trails, map[string]any{"author": parentAuthor, "public": true})
					if err := syncSummitLogs(f.app, t.Context(), trail, "https://remote.example", []any{raw}); err != nil {
						t.Fatal(err)
					}
					records, err := f.app.FindAllRecords("summit_logs")
					if err != nil || len(records) != 0 {
						t.Fatalf("remote summit log persisted: count=%d error=%v", len(records), err)
					}
				} else {
					list := f.save(f.lists, map[string]any{"author": parentAuthor})
					if err := syncTrails(f.app, t.Context(), list, "https://remote.example", []any{raw}); err != nil {
						t.Fatal(err)
					}
					records, err := f.app.FindAllRecords("trails")
					if err != nil || len(records) != 0 || len(list.GetStringSlice("trails")) != 0 {
						t.Fatalf("remote trail persisted: count=%d error=%v", len(records), err)
					}
				}
			})
		}
	}
}

func TestRemoteSyncResolvesAuthorReferences(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteSyncFixture(t)
	for _, tc := range []struct {
		actor *core.Record
		iri   string
	}{
		{f.localActor, "https://local.example/api/v1/summit-log/own"},
		{f.remoteActor, "https://remote.example/api/v1/summit-log/remote"},
	} {
		raw := map[string]any{"expand": map[string]any{"author": map[string]any{"iri": tc.actor.GetString("iri")}}}
		author, allowed, err := resolveRemoteSyncAuthor(f.app, t.Context(), f.remoteActor.Id, tc.iri, raw)
		if err != nil || !allowed || author != tc.actor.Id {
			t.Fatalf("legitimate author rejected: author=%s allowed=%t error=%v", author, allowed, err)
		}
	}
}

type remoteSyncFixture struct {
	t                                      *testing.T
	app                                    *pbtests.TestApp
	actors, trails, waypoints, logs, lists *core.Collection
	localActor, remoteActor                *core.Record
}

func drainTestAppStartupLogs(t *testing.T, app core.App) {
	t.Helper()
	// Startup logs predate TestApp disabling storage; drain them while the app is live.
	if err := app.Logger().Handler().(*logger.BatchHandler).WriteAll(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func newRemoteSyncFixture(t *testing.T) *remoteSyncFixture {
	t.Helper()
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	drainTestAppStartupLogs(t, app)
	app.Settings().Logs.MaxDays = 0
	f := &remoteSyncFixture{t: t, app: app}
	collection := func(name string, fields ...core.Field) *core.Collection {
		t.Helper()
		c := core.NewBaseCollection(name)
		c.Fields.Add(fields...)
		if err := app.Save(c); err != nil {
			t.Fatal(err)
		}
		return c
	}
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	user := core.NewRecord(users)
	user.SetEmail("local@example.com")
	user.SetPassword("local-test-password")
	if err := app.Save(user); err != nil {
		t.Fatal(err)
	}
	settings := collection("settings", &core.TextField{Name: "user"}, &core.TextField{Name: "privacy"})
	f.save(settings, map[string]any{"user": user.Id})
	collection("follows", &core.TextField{Name: "follower"}, &core.TextField{Name: "followee"}, &core.TextField{Name: "status"})
	f.actors = collection("activitypub_actors", &core.BoolField{Name: "is_local"}, &core.TextField{Name: "iri"}, &core.TextField{Name: "domain"}, &core.TextField{Name: "public_key"}, &core.TextField{Name: "user"}, &core.DateField{Name: "last_fetched"})
	f.localActor = f.save(f.actors, map[string]any{"is_local": true, "iri": "https://local.example/api/v1/actor/alice", "user": user.Id})
	f.remoteActor = f.save(f.actors, map[string]any{"iri": "https://remote.example/api/v1/actor/riley", "last_fetched": time.Now()})
	f.trails = collection("trails",
		&core.TextField{Name: "iri"}, &core.TextField{Name: "name"}, &core.TextField{Name: "description"}, &core.BoolField{Name: "public"},
		&core.BoolField{Name: "needs_full_sync"}, &core.BoolField{Name: "full_sync_completed"}, &core.RelationField{Name: "author", CollectionId: f.actors.Id, MaxSelect: 1},
	)
	f.waypoints = collection("waypoints",
		&core.TextField{Name: "iri"}, &core.TextField{Name: "name"}, &core.TextField{Name: "description"},
		&core.RelationField{Name: "author", CollectionId: f.actors.Id, MaxSelect: 1}, &core.RelationField{Name: "trail", CollectionId: f.trails.Id, MaxSelect: 1},
	)
	f.logs = collection("summit_logs",
		&core.TextField{Name: "iri"}, &core.TextField{Name: "text"},
		&core.RelationField{Name: "author", CollectionId: f.actors.Id, MaxSelect: 1}, &core.RelationField{Name: "trail", CollectionId: f.trails.Id, MaxSelect: 1},
	)
	f.lists = collection("lists", &core.TextField{Name: "iri"}, &core.TextField{Name: "name"}, &core.BoolField{Name: "public"}, &core.BoolField{Name: "needs_full_sync"}, &core.BoolField{Name: "full_sync_completed"}, &core.RelationField{Name: "author", CollectionId: f.actors.Id, MaxSelect: 1}, &core.RelationField{Name: "trails", CollectionId: f.trails.Id, MaxSelect: 100})
	return f
}

func (f *remoteSyncFixture) save(collection *core.Collection, data map[string]any) *core.Record {
	f.t.Helper()
	if collection.Name == "activitypub_actors" {
		if _, exists := data["domain"]; !exists {
			iri, _ := data["iri"].(string)
			if parsed, err := url.Parse(iri); err == nil {
				data["domain"] = parsed.Host
			}
		}
		if _, exists := data["public_key"]; !exists {
			data["public_key"] = "test-key"
		}
	}
	record := core.NewRecord(collection)
	record.Load(data)
	if err := f.app.Save(record); err != nil {
		f.t.Fatal(err)
	}
	return record
}

func (f *remoteSyncFixture) assertUnchanged(record *core.Record, fields ...string) {
	f.t.Helper()
	stored, err := f.app.FindRecordById(record.Collection(), record.Id)
	if err != nil {
		f.t.Fatal(err)
	}
	for _, field := range fields {
		if got, want := stored.GetString(field), record.GetString(field); got != want {
			f.t.Errorf("%s.%s changed from %q to %q", record.Collection().Name, field, want, got)
		}
	}
}

func TestRemoteListSyncOnlyReferencesLocalTrails(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteSyncFixture(t)
	list := f.save(f.lists, map[string]any{"author": f.remoteActor.Id})
	for _, public := range []bool{true, false} {
		local := f.save(f.trails, map[string]any{
			"iri":  "https://local.example/api/v1/trail/" + map[bool]string{true: "public", false: "private"}[public],
			"name": "Local name", "description": "Local description", "author": f.localActor.Id, "public": public,
		})
		payload := map[string]any{
			"id": "remote-copy", "iri": local.GetString("iri"), "name": "Remote copy", "description": "Stale copy", "public": true,
			"expand": map[string]any{"author": map[string]any{"iri": f.remoteActor.GetString("iri")}},
		}
		if err := syncTrails(f.app, t.Context(), list, "https://remote.example", []any{payload}); err != nil {
			t.Fatal(err)
		}
		f.assertUnchanged(local, "iri", "name", "description", "author", "public", "needs_full_sync")
		want := []string{}
		if public {
			want = append(want, local.Id)
		}
		if got := list.GetStringSlice("trails"); !slices.Equal(got, want) {
			t.Fatalf("local trail reference = %v, want [%s]", got, local.Id)
		}
	}
}

func TestRemoteTrailSyncDoesNotMoveOrOverwriteLocalChildren(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteSyncFixture(t)
	privateTrail := f.save(f.trails, map[string]any{"iri": "https://local.example/api/v1/trail/private", "author": f.localActor.Id})
	remoteTrail := f.save(f.trails, map[string]any{"iri": "https://remote.example/api/v1/trail/remote", "author": f.remoteActor.Id, "public": true})
	for _, originalTrail := range []*core.Record{privateTrail, remoteTrail} {
		waypoint := f.save(f.waypoints, map[string]any{"iri": "https://local.example/api/v1/waypoint/" + originalTrail.Id, "author": f.localActor.Id, "trail": originalTrail.Id, "name": "Local waypoint"})
		log := f.save(f.logs, map[string]any{"iri": "https://local.example/api/v1/summit-log/" + originalTrail.Id, "author": f.localActor.Id, "trail": originalTrail.Id, "text": "Local entry"})
		for _, child := range []*core.Record{waypoint, log} {
			raw := map[string]any{"id": "copy-id", "iri": child.GetString("iri"), "name": "Remote copy", "text": "Remote copy", "photos": []any{"remote.jpg"}}
			var err error
			if child.Collection().Name == "waypoints" {
				err = syncWaypoints(f.app, t.Context(), remoteTrail, "https://remote.example", []any{raw})
			} else {
				err = syncSummitLogs(f.app, t.Context(), remoteTrail, "https://remote.example", []any{raw})
			}
			if err != nil {
				t.Fatal(err)
			}
			f.assertUnchanged(child, "iri", "author", "trail", "name", "text")
		}
	}
}

func TestRemoteSyncOnlyMutatesRecordsFromThePayloadOrigin(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteSyncFixture(t)
	thirdActor := f.save(f.actors, map[string]any{"iri": "https://third.example/api/v1/actor/carol", "last_fetched": time.Now()})
	thirdTrail := f.save(f.trails, map[string]any{"iri": "https://third.example/api/v1/trail/original", "author": thirdActor.Id, "name": "Original trail", "public": true})
	remoteTrail := f.save(f.trails, map[string]any{"iri": "https://remote.example/api/v1/trail/remote", "author": f.remoteActor.Id})
	waypoint := f.save(f.waypoints, map[string]any{"iri": "https://third.example/api/v1/waypoint/original", "author": thirdActor.Id, "trail": thirdTrail.Id, "name": "Original waypoint"})
	log := f.save(f.logs, map[string]any{"iri": "https://third.example/api/v1/summit-log/original", "author": thirdActor.Id, "trail": thirdTrail.Id, "text": "Original entry"})
	list := f.save(f.lists, map[string]any{"author": f.remoteActor.Id})
	for _, record := range []*core.Record{thirdTrail, waypoint, log} {
		raw := map[string]any{"id": "copy-id", "iri": record.GetString("iri"), "name": "Remote copy", "text": "Remote copy", "public": true}
		var err error
		switch record.Collection().Name {
		case "trails":
			err = syncTrails(f.app, t.Context(), list, "https://remote.example", []any{raw})
		case "waypoints":
			err = syncWaypoints(f.app, t.Context(), remoteTrail, "https://remote.example", []any{raw})
		case "summit_logs":
			err = syncSummitLogs(f.app, t.Context(), remoteTrail, "https://remote.example", []any{raw})
		}
		if err != nil {
			t.Fatal(err)
		}
		f.assertUnchanged(record, "iri", "author", "trail", "name", "text", "public", "needs_full_sync")
	}
	if got := list.GetStringSlice("trails"); !slices.Equal(got, []string{thirdTrail.Id}) {
		t.Fatalf("foreign trail reference = %v, want [%s]", got, thirdTrail.Id)
	}
}

func TestRemoteSyncCannotReassignExistingRemoteRecords(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteSyncFixture(t)
	otherActor := f.save(f.actors, map[string]any{"iri": "https://remote.example/api/v1/actor/other", "last_fetched": time.Now()})
	otherTrail := f.save(f.trails, map[string]any{"iri": "https://remote.example/api/v1/trail/other", "author": otherActor.Id, "name": "Original trail"})
	remoteTrail := f.save(f.trails, map[string]any{"iri": "https://remote.example/api/v1/trail/remote", "author": f.remoteActor.Id})
	list := f.save(f.lists, map[string]any{"author": f.remoteActor.Id})
	if err := syncTrails(f.app, t.Context(), list, "https://remote.example", []any{map[string]any{"id": "copy", "iri": otherTrail.GetString("iri"), "name": "Remote copy"}}); err != nil {
		t.Fatal(err)
	}
	f.assertUnchanged(otherTrail, "author", "name", "iri")
	for _, author := range []string{f.remoteActor.Id, otherActor.Id} {
		waypoint := f.save(f.waypoints, map[string]any{"iri": "https://remote.example/api/v1/waypoint/" + author, "author": author, "trail": otherTrail.Id, "name": "Original waypoint"})
		log := f.save(f.logs, map[string]any{"iri": "https://remote.example/api/v1/summit-log/" + author, "author": author, "trail": otherTrail.Id, "text": "Original entry"})
		if err := syncWaypoints(f.app, t.Context(), remoteTrail, "https://remote.example", []any{map[string]any{"id": "copy", "iri": waypoint.GetString("iri"), "name": "Remote copy"}}); err != nil {
			t.Fatal(err)
		}
		if err := syncSummitLogs(f.app, t.Context(), remoteTrail, "https://remote.example", []any{map[string]any{"id": "copy", "iri": log.GetString("iri"), "text": "Remote copy"}}); err != nil {
			t.Fatal(err)
		}
		f.assertUnchanged(waypoint, "iri", "author", "trail", "name")
		f.assertUnchanged(log, "iri", "author", "trail", "text")
	}
}

func TestRemoteListSkipsUnknownForeignTrailsWithRequiredNames(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteSyncFixture(t)
	f.trails.Fields.GetByName("name").(*core.TextField).Required = true
	if err := f.app.Save(f.trails); err != nil {
		t.Fatal(err)
	}
	thirdActor := f.save(f.actors, map[string]any{"iri": "https://third.example/api/v1/actor/carol", "last_fetched": time.Now()})
	list := f.save(f.lists, map[string]any{"author": f.remoteActor.Id})
	const canonicalIRI = "https://third.example/api/v1/trail/original-id"
	if err := f.app.RunInTransaction(func(app core.App) error {
		if err := syncTrails(app, t.Context(), list, "https://remote.example", []any{
			map[string]any{"id": "list-copy-id", "iri": canonicalIRI, "name": "Untrusted copy", "public": true,
				"expand": map[string]any{"author": map[string]any{"iri": thirdActor.GetString("iri")}}},
			map[string]any{"id": "owned-trail", "name": "Authoritative trail", "public": true},
		}); err != nil {
			return err
		}
		return app.Save(list)
	}); err != nil {
		t.Fatalf("unknown foreign reference rolled back the list: %v", err)
	}
	if trail, _ := f.app.FindFirstRecordByData("trails", "iri", canonicalIRI); trail != nil {
		t.Fatal("untrusted foreign shell persisted")
	}
	trails, err := f.app.FindAllRecords("trails")
	if err != nil || len(trails) != 1 || trails[0].GetString("name") != "Authoritative trail" || !slices.Equal(list.GetStringSlice("trails"), []string{trails[0].Id}) {
		t.Fatalf("valid list trail lost with foreign reference: trails=%v error=%v", trails, err)
	}
}

func TestRemoteListSkipsUnknownLocalAndUnverifiableForeignTrails(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteSyncFixture(t)
	list := f.save(f.lists, map[string]any{"author": f.remoteActor.Id})
	if err := syncTrails(f.app, t.Context(), list, "https://remote.example", []any{
		map[string]any{"iri": "https://local.example/api/v1/trail/unknown"},
		map[string]any{"iri": "https://third.example/api/v1/trail/unknown"},
		map[string]any{"iri": "https://third.example/api/v1/trail/unknown", "expand": map[string]any{"author": map[string]any{"iri": f.remoteActor.GetString("iri")}}},
		map[string]any{"iri": "/api/v1/trail/relative"},
	}); err != nil {
		t.Fatal(err)
	}
	records, err := f.app.FindAllRecords("trails")
	if err != nil || len(records) != 0 || len(list.GetStringSlice("trails")) != 0 {
		t.Fatalf("unverifiable trails imported: count=%d error=%v", len(records), err)
	}
}

func TestRemoteSyncIRIMatchesOrigin(t *testing.T) {
	for _, tc := range []struct {
		iri  string
		want bool
	}{
		{"https://remote.example/api/v1/trail/owned", true},
		{"https://REMOTE.EXAMPLE/api/v1/trail/owned", true},
		{"https://another.example/api/v1/trail/foreign", false},
		{"https://remote.example:8443/api/v1/trail/foreign", false},
		{"http://remote.example/api/v1/trail/foreign", true},
		{"https://user@remote.example/api/v1/trail/foreign", false},
		{"/api/v1/trail/relative", false},
		{"https://%/invalid", false},
	} {
		if got := remoteSyncIRIMatchesOrigin(tc.iri, "https://remote.example"); got != tc.want {
			t.Errorf("origin check for %q = %t, want %t", tc.iri, got, tc.want)
		}
	}
}

type remoteSyncTransport struct {
	roundTrip func(*http.Request) (*http.Response, error)
}

func (transport remoteSyncTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport.roundTrip(request)
}
