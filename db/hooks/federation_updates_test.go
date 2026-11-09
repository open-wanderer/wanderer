package hooks

import (
	"cmp"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"pocketbase/federation"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/security"
)

const (
	testFederationUpdateDelay = 50 * time.Millisecond
	testFederationCreateDelay = time.Second
)

// The editor saves a trail or summit log first and its photo links one request
// at a time afterwards, and a photo can also be linked, unlinked or marked as
// thumbnail on its own. Followers get one Update once the changes have
// settled, built from the links as they are then, and only for content of a
// local author that they can see and that still exists.
func TestFederationUpdatesSendOneUpdateOnceLinksSettle(t *testing.T) {
	f := newFederationUpdatesFixture(t)
	synctest.Test(t, func(t *testing.T) {
		f.t = t
		local := f.save(f.actors, map[string]any{"is_local": true})
		remote := f.save(f.actors, map[string]any{"is_local": false})
		trail := f.save(f.trails, map[string]any{"public": true, "author": local.Id})
		waypoint := f.save(f.waypoints, map[string]any{"trail": trail.Id})
		log := f.save(f.logs, map[string]any{"trail": trail.Id, "author": local.Id})
		otherTrail := f.save(f.trails, map[string]any{"public": true, "author": local.Id})
		otherWaypoint := f.save(f.waypoints, map[string]any{"trail": otherTrail.Id})
		otherLog := f.save(f.logs, map[string]any{"trail": otherTrail.Id, "author": local.Id})
		remoteTrail := f.save(f.trails, map[string]any{"public": true, "author": remote.Id})
		privateTrail := f.save(f.trails, map[string]any{"public": false, "author": local.Id})
		logOnPrivateTrail := f.save(f.logs, map[string]any{"trail": privateTrail.Id, "author": local.Id})
		deletedTrail := f.save(f.trails, map[string]any{"public": true, "author": local.Id})

		// An editor save of the trail, which reaches UpdateTrailHandler, and of
		// the summit log, as UpdateSummitLogHandler enqueues it.
		f.editTrail(trail)
		first := f.link("trail_assets", "trail", trail)
		f.link("trail_assets", "trail", trail)
		f.link("waypoint_assets", "waypoint", waypoint)
		first.Set("is_thumbnail", true)
		if err := f.app.Save(first); err != nil {
			t.Fatal(err)
		}
		if err := f.app.Delete(first); err != nil {
			t.Fatal(err)
		}
		f.updates.Enqueue(log)
		f.link("summit_log_assets", "summit_log", log)
		f.link("summit_log_assets", "summit_log", log)

		// Photos linked on their own, a waypoint's as an Update of its trail.
		f.link("waypoint_assets", "waypoint", otherWaypoint)
		f.link("summit_log_assets", "summit_log", otherLog)

		// Nothing for a remote author, for what followers cannot see, or for a
		// trail that is gone by the time the Update would go out.
		f.link("trail_assets", "trail", remoteTrail)
		f.link("trail_assets", "trail", privateTrail)
		f.link("summit_log_assets", "summit_log", logOnPrivateTrail)
		f.link("trail_assets", "trail", deletedTrail)
		if err := f.app.Delete(deletedTrail); err != nil {
			t.Fatal(err)
		}

		f.expect(
			federatedActivity{typ: pub.UpdateType, target: "summit_logs/" + log.Id, photos: 2},
			federatedActivity{typ: pub.UpdateType, target: "summit_logs/" + otherLog.Id, photos: 1},
			federatedActivity{typ: pub.UpdateType, target: "trails/" + otherTrail.Id, photos: 0},
			federatedActivity{typ: pub.UpdateType, target: "trails/" + trail.Id, photos: 1},
		)
	})
}

// Followers see a new trail or summit log, or a trail its save makes public,
// with one Create that carries its photos: the editor links them after the
// save, and receivers would notify mentioned users once more for an Update
// following with them. The Create waits longer than an Update would, and
// changes until it goes out do not turn it into one. Later edits are Updates.
func TestFederationUpdatesAnnounceNewContentWithOneCreate(t *testing.T) {
	f := newFederationUpdatesFixture(t)
	synctest.Test(t, func(t *testing.T) {
		f.t = t
		local := f.save(f.actors, map[string]any{"is_local": true})
		trail := f.save(f.trails, map[string]any{"public": true, "author": local.Id})
		draft := f.save(f.trails, map[string]any{"public": false, "author": local.Id})
		f.link("trail_assets", "trail", draft)
		privateTrail := f.save(f.trails, map[string]any{"public": false, "author": local.Id})
		deletedTrail := f.save(f.trails, map[string]any{"public": true, "author": local.Id})

		// A trail created public, as CreateTrailHandler enqueues it, and a summit
		// log on it, as CreateSummitLogHandler does.
		f.updates.EnqueueCreate(trail)
		log := f.save(f.logs, map[string]any{"trail": trail.Id, "author": local.Id})
		f.updates.EnqueueCreate(log)
		// A trail published after its photos, the way ImportTrail and a
		// publication from the editor do it.
		f.publish(draft)
		// Nothing for what followers cannot see, or for what is gone before it
		// would be announced.
		f.updates.EnqueueCreate(privateTrail)
		f.updates.EnqueueCreate(f.save(f.logs, map[string]any{"trail": privateTrail.Id, "author": local.Id}))
		f.updates.EnqueueCreate(deletedTrail)
		if err := f.app.Delete(deletedTrail); err != nil {
			t.Fatal(err)
		}

		// Photos and edits after an Update would have gone out, in two rounds
		// further apart than an Update would wait.
		time.Sleep(4 * testFederationUpdateDelay)
		waypoint := f.save(f.waypoints, map[string]any{"trail": trail.Id})
		f.link("trail_assets", "trail", trail)
		f.link("waypoint_assets", "waypoint", waypoint)
		f.link("summit_log_assets", "summit_log", log)
		time.Sleep(4 * testFederationUpdateDelay)
		f.link("trail_assets", "trail", trail)
		f.editTrail(trail)
		f.link("summit_log_assets", "summit_log", log)
		f.updates.Enqueue(log)
		f.link("trail_assets", "trail", draft)

		f.expect(
			federatedActivity{typ: pub.CreateType, target: "summit_logs/" + log.Id, photos: 2},
			federatedActivity{typ: pub.CreateType, target: "trails/" + draft.Id, photos: 2},
			federatedActivity{typ: pub.CreateType, target: "trails/" + trail.Id, photos: 2},
		)

		f.editTrail(trail)
		f.updates.Enqueue(log)
		f.link("trail_assets", "trail", draft)

		f.expect(
			federatedActivity{typ: pub.UpdateType, target: "summit_logs/" + log.Id, photos: 2},
			federatedActivity{typ: pub.UpdateType, target: "trails/" + draft.Id, photos: 3},
			federatedActivity{typ: pub.UpdateType, target: "trails/" + trail.Id, photos: 2},
		)
	})
}

func TestFederationUpdatesDoNotAnnounceRemoteIRIsForLocalAuthors(t *testing.T) {
	t.Setenv("ORIGIN", "https://a.example")
	f := newFederationUpdatesFixture(t)
	synctest.Test(t, func(t *testing.T) {
		f.t = t
		local := f.save(f.actors, map[string]any{"is_local": true})
		trail := f.save(f.trails, map[string]any{"public": true, "author": local.Id, "iri": "https://b.example/api/v1/trail/record"})
		log := f.save(f.logs, map[string]any{"trail": trail.Id, "author": local.Id, "iri": "https://b.example/api/v1/summit-log/record"})
		f.updates.EnqueueCreate(trail)
		f.updates.Enqueue(log)
		f.link("trail_assets", "trail", trail)
		f.link("summit_log_assets", "summit_log", log)
		// Recheck the origin when sending, including records that were queued
		// before their canonical IRI changed.
		f.updates.federate(federationUpdateTarget{collection: "trails", id: trail.Id}, pub.UpdateType)
		f.updates.federate(federationUpdateTarget{collection: "summit_logs", id: log.Id}, pub.UpdateType)
		f.terminate()
		f.check()

		localTrail := f.save(f.trails, map[string]any{"public": true, "author": local.Id, "iri": "https://a.example/api/v1/trail/own"})
		localLog := f.save(f.logs, map[string]any{"trail": localTrail.Id, "author": local.Id, "iri": "https://a.example/api/v1/summit-log/own"})
		f.updates.EnqueueCreate(localTrail)
		f.updates.Enqueue(localLog)
		f.terminate()
		f.check(
			federatedActivity{typ: pub.CreateType, target: "trails/" + localTrail.Id},
			federatedActivity{typ: pub.UpdateType, target: "summit_logs/" + localLog.Id},
		)
	})
}

// What is pending lives in memory only, so it goes out when the app shuts
// down instead of being lost with the process, and not again afterwards. A
// send that hangs holds the shutdown up no longer than its timeout.
func TestFederationUpdatesSendPendingActivitiesOnShutdown(t *testing.T) {
	f := newFederationUpdatesFixture(t)
	local := f.save(f.actors, map[string]any{"is_local": true})
	trail := f.save(f.trails, map[string]any{"public": true, "author": local.Id})
	log := f.save(f.logs, map[string]any{"trail": trail.Id, "author": local.Id})
	edited := f.save(f.trails, map[string]any{"public": true, "author": local.Id})

	f.updates.EnqueueCreate(trail)
	f.updates.EnqueueCreate(log)
	f.link("summit_log_assets", "summit_log", log)
	f.editTrail(edited)
	f.terminate()

	f.check(
		federatedActivity{typ: pub.CreateType, target: "summit_logs/" + log.Id, photos: 1},
		federatedActivity{typ: pub.CreateType, target: "trails/" + trail.Id, photos: 0},
		federatedActivity{typ: pub.UpdateType, target: "trails/" + edited.Id, photos: 0},
	)
	time.Sleep(testFederationCreateDelay)
	f.expect()

	hanging := newFederationUpdates(f.app, testFederationUpdateDelay, testFederationCreateDelay, func(core.App, *core.Record, *core.Record, pub.ActivityVocabularyType) error {
		time.Sleep(time.Second)
		return nil
	})
	hanging.shutdownTimeout = 2 * testFederationUpdateDelay
	hanging.EnqueueCreate(trail)
	start := time.Now()
	f.terminate()
	if waited := time.Since(start); waited >= time.Second {
		t.Fatalf("shutdown waited %v for a send that takes a second", waited)
	}
}

// The inboxes receive the activities in the background, which ends with the
// process. So the shutdown waits for the deliveries of what it sends, and of
// what was sent shortly before, within the same timeout: an inbox that does
// not answer holds it up no longer than that.
func TestFederationUpdatesWaitForDeliveriesOnShutdown(t *testing.T) {
	t.Setenv("ORIGIN", "https://a.example")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", testEncryptionKey)
	f := newFederationUpdatesFixture(t)
	f.app.OnRecordAfterUpdateSuccess("trails").Unbind(f.trailSaveHandler)
	t.Cleanup(func() {
		f.updates.flushAll()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := federation.WaitForDeliveries(ctx); err != nil {
			t.Error(err)
		}
	})
	local := f.signingActor()
	trail := f.save(f.trails, map[string]any{"public": true, "author": local.Id})
	stalled := f.save(f.trails, map[string]any{"public": true, "author": local.Id})

	var delivered atomic.Int32
	stall := make(chan struct{})
	release := sync.OnceFunc(func() { close(stall) })
	t.Cleanup(release)
	slow := f.localInbox(local, func() { time.Sleep(200 * time.Millisecond); delivered.Add(1) })
	hanging := f.localInbox(local, func() { <-stall })
	inboxes := map[string]string{trail.Id: slow, stalled.Id: hanging}
	f.updates.send = func(app core.App, author *core.Record, record *core.Record, typ pub.ActivityVocabularyType) error {
		return f.postLocalTrail(author, record, typ, inboxes[record.Id])
	}

	f.updates.EnqueueCreate(trail)
	f.terminate()
	if got := delivered.Load(); got != 1 {
		t.Fatalf("shutdown returned with %d of 1 pending activities delivered", got)
	}
	// Such as a Like, which goes out right away.
	f.postLocalTrail(local, f.save(f.trails, map[string]any{"public": true, "author": local.Id}), pub.CreateType, slow)
	f.terminate()
	if got := delivered.Load(); got != 2 {
		t.Fatal("shutdown returned before an activity sent shortly before it was delivered")
	}

	f.updates.shutdownTimeout = 2 * testFederationUpdateDelay
	f.updates.EnqueueCreate(stalled)
	start := time.Now()
	f.terminate()
	if waited := time.Since(start); waited >= time.Second {
		t.Fatalf("shutdown waited %v for an inbox that does not answer", waited)
	}
	// The deliveries are counted across tests, and the next shutdown would
	// wait for this one.
	release()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := federation.WaitForDeliveries(ctx); err != nil {
		t.Fatal(err)
	}
}

// A timer that fires as the app shuts down takes its activity from the
// pending ones before the activity is built and handed over for delivery. So
// the shutdown waits for it, and then for its delivery, within the same
// timeout: one that is not built in time holds it up no longer than that.
func TestFederationUpdatesWaitForFiredTimersOnShutdown(t *testing.T) {
	t.Setenv("ORIGIN", "https://a.example")
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", testEncryptionKey)
	f := newFederationUpdatesFixture(t)
	f.app.OnRecordAfterUpdateSuccess("trails").Unbind(f.trailSaveHandler)
	t.Cleanup(func() {
		f.updates.flushAll()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := federation.WaitForDeliveries(ctx); err != nil {
			t.Error(err)
		}
	})
	local := f.signingActor()
	trail := f.save(f.trails, map[string]any{"public": true, "author": local.Id})
	stuck := f.save(f.trails, map[string]any{"public": true, "author": local.Id})

	var delivered atomic.Int32
	inbox := f.localInbox(local, func() { time.Sleep(200 * time.Millisecond); delivered.Add(1) })
	// The sender is held up while it builds an activity, as by a busy
	// database: that of trail until the test lets it go, that of stuck for a
	// second.
	building := make(chan struct{}, 2)
	built := make(chan struct{})
	release := sync.OnceFunc(func() { close(built) })
	t.Cleanup(release)
	f.updates.send = func(app core.App, author *core.Record, record *core.Record, typ pub.ActivityVocabularyType) error {
		building <- struct{}{}
		if record.Id == stuck.Id {
			time.Sleep(time.Second)
			return nil
		}
		<-built
		return f.postLocalTrail(author, record, typ, inbox)
	}
	waitUntilBuilding := func() {
		t.Helper()
		select {
		case <-building:
		case <-time.After(5 * time.Second):
			t.Fatal("the timer did not fire")
		}
	}

	f.updates.Enqueue(trail)
	waitUntilBuilding()
	shutdown := make(chan error, 1)
	go func() {
		shutdown <- f.app.OnTerminate().Trigger(&core.TerminateEvent{App: f.app})
	}()
	select {
	case <-shutdown:
		t.Fatal("shutdown returned while an activity whose timer had fired was being built")
	case <-time.After(4 * testFederationUpdateDelay):
	}
	release()
	if err := <-shutdown; err != nil {
		t.Fatal(err)
	}
	if got := delivered.Load(); got != 1 {
		t.Fatal("shutdown returned before an activity whose timer had fired was delivered")
	}

	f.updates.shutdownTimeout = 2 * testFederationUpdateDelay
	f.updates.Enqueue(stuck)
	waitUntilBuilding()
	start := time.Now()
	f.terminate()
	if waited := time.Since(start); waited >= time.Second {
		t.Fatalf("shutdown waited %v for an activity that takes a second to build", waited)
	}
}

// federationUpdatesFixture binds the photo link hooks, and the trail saves as
// UpdateTrailHandler does, to updates on a minimal schema. Its sender records
// each activity with the photos its target links when it is built.
// Timer-only tests create the database fixture before their synctest bubble:
// PocketBase's filesystem watcher needs real I/O; scheduled sends use fake time.
type federationUpdatesFixture struct {
	t                               *testing.T
	app                             *pbtests.TestApp
	updates                         *FederationUpdates
	actors, trails, waypoints, logs *core.Collection
	mu                              sync.Mutex
	sent                            []federatedActivity
	trailSaveHandler                string
}

type federatedActivity struct {
	typ    pub.ActivityVocabularyType
	target string
	photos int64
}

func newFederationUpdatesFixture(t *testing.T) *federationUpdatesFixture {
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	f := &federationUpdatesFixture{t: t, app: app}

	collection := func(name string, fields ...core.Field) *core.Collection {
		t.Helper()
		c := core.NewBaseCollection(name)
		c.Fields.Add(fields...)
		if err := app.Save(c); err != nil {
			t.Fatal(err)
		}
		return c
	}
	f.actors = collection("activitypub_actors", &core.BoolField{Name: "is_local"}, &core.TextField{Name: "private_key"})
	f.trails = collection("trails",
		&core.TextField{Name: "iri"},
		&core.BoolField{Name: "public"},
		&core.RelationField{Name: "author", CollectionId: f.actors.Id, MaxSelect: 1},
	)
	f.waypoints = collection("waypoints",
		&core.RelationField{Name: "trail", CollectionId: f.trails.Id, MaxSelect: 1, CascadeDelete: true},
	)
	f.logs = collection("summit_logs",
		&core.TextField{Name: "iri"},
		&core.RelationField{Name: "trail", CollectionId: f.trails.Id, MaxSelect: 1, CascadeDelete: true},
		&core.RelationField{Name: "author", CollectionId: f.actors.Id, MaxSelect: 1},
	)
	collection("trail_assets",
		&core.TextField{Name: "asset"},
		&core.RelationField{Name: "trail", CollectionId: f.trails.Id, MaxSelect: 1, CascadeDelete: true},
		&core.BoolField{Name: "is_thumbnail"},
	)
	collection("waypoint_assets",
		&core.TextField{Name: "asset"},
		&core.RelationField{Name: "waypoint", CollectionId: f.waypoints.Id, MaxSelect: 1, CascadeDelete: true},
	)
	collection("summit_log_assets",
		&core.TextField{Name: "asset"},
		&core.RelationField{Name: "summit_log", CollectionId: f.logs.Id, MaxSelect: 1, CascadeDelete: true},
	)

	f.updates = newFederationUpdates(app, testFederationUpdateDelay, testFederationCreateDelay, func(app core.App, _ *core.Record, record *core.Record, typ pub.ActivityVocabularyType) error {
		links, field := "trail_assets", "trail"
		if record.Collection().Name == "summit_logs" {
			links, field = "summit_log_assets", "summit_log"
		}
		photos, err := app.CountRecords(links, dbx.HashExp{field: record.Id})
		if err != nil {
			return err
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		f.sent = append(f.sent, federatedActivity{typ: typ, target: record.Collection().Name + "/" + record.Id, photos: photos})
		return nil
	})
	handler := FederateOnAssetLinkChange(f.updates)
	app.OnRecordAfterCreateSuccess("trail_assets", "waypoint_assets", "summit_log_assets").BindFunc(handler)
	app.OnRecordAfterUpdateSuccess("trail_assets").BindFunc(handler)
	app.OnRecordAfterDeleteSuccess("trail_assets", "waypoint_assets", "summit_log_assets").BindFunc(handler)
	f.trailSaveHandler = app.OnRecordAfterUpdateSuccess("trails").BindFunc(func(e *core.RecordEvent) error {
		f.updates.EnqueueTrailSave(e.Record)
		return e.Next()
	})
	return f
}

func (f *federationUpdatesFixture) save(collection *core.Collection, values map[string]any) *core.Record {
	f.t.Helper()
	record := core.NewRecord(collection)
	record.Load(values)
	if err := f.app.Save(record); err != nil {
		f.t.Fatal(err)
	}
	return record
}

// localInbox exercises actual in-process delivery. A record hook can delay
// applying its feed entry, like a busy database, while shutdown waits for it.
func (f *federationUpdatesFixture) localInbox(sender *core.Record, apply func()) string {
	f.t.Helper()
	for _, name := range []string{"iri", "inbox"} {
		if f.actors.Fields.GetByName(name) == nil {
			f.actors.Fields.Add(&core.TextField{Name: name})
		}
	}
	if err := f.app.Save(f.actors); err != nil {
		f.t.Fatal(err)
	}
	sender.Set("iri", "https://a.example/api/v1/activitypub/user/"+sender.Id)
	sender.Set("inbox", "")
	if err := f.app.Save(sender); err != nil {
		f.t.Fatal(err)
	}
	if _, err := f.app.FindCollectionByNameOrId("feed"); err != nil {
		feed := core.NewBaseCollection("feed")
		feed.Fields.Add(&core.TextField{Name: "actor"}, &core.TextField{Name: "author"}, &core.TextField{Name: "item"}, &core.TextField{Name: "type"})
		if err := f.app.Save(feed); err != nil {
			f.t.Fatal(err)
		}
	}
	recipient := f.save(f.actors, map[string]any{"is_local": true})
	recipient.Set("iri", "https://a.example/api/v1/activitypub/user/"+recipient.Id)
	recipient.Set("inbox", recipient.GetString("iri")+"/inbox")
	if err := f.app.Save(recipient); err != nil {
		f.t.Fatal(err)
	}
	f.app.OnRecordCreate("feed").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetString("actor") == recipient.Id {
			apply()
		}
		return e.Next()
	})
	return recipient.GetString("inbox")
}

func (f *federationUpdatesFixture) postLocalTrail(author, record *core.Record, typ pub.ActivityVocabularyType, inbox string) error {
	iri := "https://a.example/api/v1/trail/" + record.Id
	record.Set("iri", iri)
	if err := f.app.Save(record); err != nil {
		return err
	}
	activity := pub.ActivityNew(pub.IRI(iri+"/activity"), typ, &pub.Object{ID: pub.IRI(iri), Type: pub.NoteType})
	activity.Actor = pub.IRI(author.GetString("iri"))
	return federation.PostActivity(f.app, author, activity, []string{inbox})
}

// signingActor saves a local actor with the private key that
// federation.PostActivity signs its deliveries with.
func (f *federationUpdatesFixture) signingActor() *core.Record {
	f.t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		f.t.Fatal(err)
	}
	privateKey, err := security.Encrypt(x509.MarshalPKCS1PrivateKey(key), testEncryptionKey)
	if err != nil {
		f.t.Fatal(err)
	}
	return f.save(f.actors, map[string]any{"is_local": true, "private_key": privateKey})
}

func (f *federationUpdatesFixture) link(name string, field string, target *core.Record) *core.Record {
	f.t.Helper()
	c, err := f.app.FindCollectionByNameOrId(name)
	if err != nil {
		f.t.Fatal(err)
	}
	return f.save(c, map[string]any{"asset": "photo", field: target.Id})
}

// editTrail saves a trail as it is stored, the way an edit reaches the hooks.
func (f *federationUpdatesFixture) editTrail(trail *core.Record) {
	f.t.Helper()
	stored, err := f.app.FindRecordById("trails", trail.Id)
	if err != nil {
		f.t.Fatal(err)
	}
	if err := f.app.Save(stored); err != nil {
		f.t.Fatal(err)
	}
}

// publish makes a private trail public in a transaction of its own.
func (f *federationUpdatesFixture) publish(trail *core.Record) {
	f.t.Helper()
	err := f.app.RunInTransaction(func(txApp core.App) error {
		stored, err := txApp.FindRecordById("trails", trail.Id)
		if err != nil {
			return err
		}
		stored.Set("public", true)
		return txApp.Save(stored)
	})
	if err != nil {
		f.t.Fatal(err)
	}
}

// terminate runs what is bound to the app's shutdown, which leaves the app
// itself usable.
func (f *federationUpdatesFixture) terminate() {
	f.t.Helper()
	if err := f.app.OnTerminate().Trigger(&core.TerminateEvent{App: f.app}); err != nil {
		f.t.Fatal(err)
	}
}

// expect waits for as many activities as it wants, and long enough after
// them for one sent on top to show up, and checks what has been sent.
func (f *federationUpdatesFixture) expect(want ...federatedActivity) {
	f.t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		f.mu.Lock()
		sent := len(f.sent)
		f.mu.Unlock()
		if sent >= len(want) {
			break
		}
	}
	time.Sleep(6 * testFederationUpdateDelay)
	f.check(want...)
}

// check takes what has been sent so far, which has to be what it wants.
func (f *federationUpdatesFixture) check(want ...federatedActivity) {
	f.t.Helper()
	f.mu.Lock()
	sent := f.sent
	f.sent = nil
	f.mu.Unlock()
	order := func(a, b federatedActivity) int {
		return cmp.Or(cmp.Compare(a.target, b.target), cmp.Compare(a.typ, b.typ), cmp.Compare(a.photos, b.photos))
	}
	slices.SortFunc(sent, order)
	slices.SortFunc(want, order)
	if !slices.Equal(sent, want) {
		f.t.Fatalf("sent %+v, want %+v", sent, want)
	}
}
