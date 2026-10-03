package federation

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const (
	lifecycleOrigin         = "https://trails.example.com"
	lifecycleLocalInstance  = "https://trails.example.com/api/v1/activitypub/instance"
	lifecycleRemoteInstance = "https://peer.example.com/api/v1/activitypub/instance"
	lifecycleLocalPerson    = "https://trails.example.com/api/v1/activitypub/user/lena"
	lifecycleRemotePerson   = "https://peer.example.com/api/v1/activitypub/user/bob"
	lifecycleF1             = "https://peer.example.com/api/v1/activitypub/activity/f1"
	lifecycleF2             = "https://peer.example.com/api/v1/activitypub/activity/f2"
)

// newLifecycleTestApp builds on newInboxTestApp and brings the follows
// collection closer to production: "rejected" status, the activity_iri column
// and the unique (follower, followee) index.
func newLifecycleTestApp(t *testing.T) core.App {
	t.Helper()
	t.Setenv("ORIGIN", lifecycleOrigin)
	t.Setenv("POCKETBASE_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	app := newInboxTestApp(t)
	collection, err := app.FindCollectionByNameOrId("follows")
	if err != nil {
		t.Fatalf("find follows: %v", err)
	}
	status, ok := collection.Fields.GetByName("status").(*core.SelectField)
	if !ok {
		t.Fatalf("status is not a select field")
	}
	status.Values = append(status.Values, "rejected")
	collection.Fields.Add(&core.TextField{Name: "activity_iri"})
	collection.AddIndex("idx_lifecycle_follow_pair", true, "follower, followee", "")
	if err := app.Save(collection); err != nil {
		t.Fatalf("save follows: %v", err)
	}
	return app
}

func lifecycleFollow(id, actorIRI, objectIRI string) pub.Activity {
	a := pub.FollowNew(pub.IRI(id), pub.IRI(objectIRI))
	a.Actor = pub.IRI(actorIRI)
	return *a
}

func lifecycleUndo(id string, follow pub.Activity) pub.Activity {
	u := pub.UndoNew(pub.IRI(id), &follow)
	u.Actor = follow.Actor
	return *u
}

func lifecycleRow(t *testing.T, app core.App, followerID, followeeID string) *core.Record {
	t.Helper()
	row, err := app.FindFirstRecordByFilter("follows",
		"follower={:follower} && followee={:followee}",
		dbx.Params{"follower": followerID, "followee": followeeID})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		t.Fatalf("find follow row: %v", err)
	}
	return row
}

func lifecycleCountRows(t *testing.T, app core.App) int {
	t.Helper()
	rows, err := app.FindAllRecords("follows")
	if err != nil {
		t.Fatalf("list follows: %v", err)
	}
	return len(rows)
}

// lifecycleInboundFixture returns the local instance actor and a remote peer
// instance actor, the typical inbound Follow pair.
func lifecycleInboundFixture(t *testing.T, app core.App) (local, remote *core.Record) {
	t.Helper()
	local = createTestActor(t, app, lifecycleLocalInstance, "instance", true)
	remote = createTestActor(t, app, lifecycleRemoteInstance, "instance", false)
	return local, remote
}

func lifecycleSetStatus(t *testing.T, app core.App, row *core.Record, status string) {
	t.Helper()
	row.Set("status", status)
	if err := app.Save(row); err != nil {
		t.Fatalf("set status %s: %v", status, err)
	}
}

func TestLifecycleRetryUndoBeforeNewFollow(t *testing.T) {
	app := newLifecycleTestApp(t)
	local, remote := lifecycleInboundFixture(t, app)

	f1 := lifecycleFollow(lifecycleF1, remote.GetString("iri"), local.GetString("iri"))
	if err := ProcessFollowActivity(app, remote, f1); err != nil {
		t.Fatalf("F1: %v", err)
	}
	row := lifecycleRow(t, app, remote.Id, local.Id)
	if row == nil || row.GetString("activity_iri") != lifecycleF1 {
		t.Fatalf("want pending row with activity_iri F1, got %v", row)
	}
	lifecycleSetStatus(t, app, row, "rejected")

	if err := ProcessUndoActivity(app, remote, lifecycleUndo("https://peer.example.com/api/v1/activitypub/activity/u1", f1)); err != nil {
		t.Fatalf("Undo F1: %v", err)
	}
	if lifecycleRow(t, app, remote.Id, local.Id) != nil {
		t.Fatalf("row should be deleted by Undo{F1}")
	}

	f2 := lifecycleFollow(lifecycleF2, remote.GetString("iri"), local.GetString("iri"))
	if err := ProcessFollowActivity(app, remote, f2); err != nil {
		t.Fatalf("F2: %v", err)
	}
	row = lifecycleRow(t, app, remote.Id, local.Id)
	if row == nil || row.GetString("status") != "pending" || row.GetString("activity_iri") != lifecycleF2 {
		t.Fatalf("want one pending row with activity_iri F2, got %v", row)
	}
	if n := lifecycleCountRows(t, app); n != 1 {
		t.Fatalf("want 1 row, got %d", n)
	}
}

func TestLifecycleRetryNewFollowBeforeUndo(t *testing.T) {
	app := newLifecycleTestApp(t)
	local, remote := lifecycleInboundFixture(t, app)

	f1 := lifecycleFollow(lifecycleF1, remote.GetString("iri"), local.GetString("iri"))
	if err := ProcessFollowActivity(app, remote, f1); err != nil {
		t.Fatalf("F1: %v", err)
	}
	row := lifecycleRow(t, app, remote.Id, local.Id)
	lifecycleSetStatus(t, app, row, "rejected")

	f2 := lifecycleFollow(lifecycleF2, remote.GetString("iri"), local.GetString("iri"))
	if err := ProcessFollowActivity(app, remote, f2); err != nil {
		t.Fatalf("F2: %v", err)
	}
	row = lifecycleRow(t, app, remote.Id, local.Id)
	if row == nil || row.GetString("status") != "pending" || row.GetString("activity_iri") != lifecycleF2 {
		t.Fatalf("want pending row with activity_iri F2 after F2, got %v", row)
	}

	if err := ProcessUndoActivity(app, remote, lifecycleUndo("https://peer.example.com/api/v1/activitypub/activity/u1", f1)); err != nil {
		t.Fatalf("late Undo F1 should be ignored, got error: %v", err)
	}
	row = lifecycleRow(t, app, remote.Id, local.Id)
	if row == nil {
		t.Fatalf("stale Undo{F1} deleted the row of F2")
	}
	if row.GetString("status") != "pending" || row.GetString("activity_iri") != lifecycleF2 {
		t.Fatalf("row changed by stale Undo: status=%s iri=%s", row.GetString("status"), row.GetString("activity_iri"))
	}
}

func TestLifecycleReplayOfUndoneFollowIgnored(t *testing.T) {
	app := newLifecycleTestApp(t)
	local, remote := lifecycleInboundFixture(t, app)

	f1 := lifecycleFollow(lifecycleF1, remote.GetString("iri"), local.GetString("iri"))
	if err := ProcessFollowActivity(app, remote, f1); err != nil {
		t.Fatalf("F1: %v", err)
	}
	if err := ProcessUndoActivity(app, remote, lifecycleUndo("https://peer.example.com/api/v1/activitypub/activity/u1", f1)); err != nil {
		t.Fatalf("Undo F1: %v", err)
	}
	if lifecycleRow(t, app, remote.Id, local.Id) != nil {
		t.Fatalf("row should be deleted")
	}
	if err := ProcessFollowActivity(app, remote, f1); err != nil {
		t.Fatalf("replay F1: %v", err)
	}
	if lifecycleRow(t, app, remote.Id, local.Id) != nil {
		t.Fatalf("replay of an undone Follow recreated the row")
	}
}

func TestLifecycleDuplicateFollowIsNoop(t *testing.T) {
	app := newLifecycleTestApp(t)
	local, remote := lifecycleInboundFixture(t, app)

	f2 := lifecycleFollow(lifecycleF2, remote.GetString("iri"), local.GetString("iri"))
	for i := 0; i < 2; i++ {
		if err := ProcessFollowActivity(app, remote, f2); err != nil {
			t.Fatalf("F2 delivery %d: %v", i, err)
		}
	}
	if n := lifecycleCountRows(t, app); n != 1 {
		t.Fatalf("want 1 follow row, got %d", n)
	}
	acts, err := app.FindAllRecords("activitypub_activities", dbx.HashExp{"iri": lifecycleF2})
	if err != nil {
		t.Fatalf("list activities: %v", err)
	}
	if len(acts) != 1 {
		t.Fatalf("want exactly 1 activity row with iri F2, got %d", len(acts))
	}
}

func TestLifecycleLegacyRowDeletedByUndo(t *testing.T) {
	app := newLifecycleTestApp(t)
	local, remote := lifecycleInboundFixture(t, app)

	col, err := app.FindCollectionByNameOrId("follows")
	if err != nil {
		t.Fatal(err)
	}
	row := core.NewRecord(col)
	row.Set("follower", remote.Id)
	row.Set("followee", local.Id)
	row.Set("status", "pending")
	if err := app.Save(row); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	f1 := lifecycleFollow(lifecycleF1, remote.GetString("iri"), local.GetString("iri"))
	if err := ProcessUndoActivity(app, remote, lifecycleUndo("https://peer.example.com/api/v1/activitypub/activity/u1", f1)); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if lifecycleRow(t, app, remote.Id, local.Id) != nil {
		t.Fatalf("legacy row (empty activity_iri) should be deleted by Undo")
	}
}

func TestLifecycleUndoNilObject(t *testing.T) {
	app := newLifecycleTestApp(t)
	_, remote := lifecycleInboundFixture(t, app)

	undo := pub.UndoNew(pub.IRI("https://peer.example.com/api/v1/activitypub/activity/u1"), nil)
	undo.Actor = pub.IRI(remote.GetString("iri"))
	if err := ProcessUndoActivity(app, remote, *undo); err == nil {
		t.Fatalf("Undo with nil object should return an error")
	}
}

func TestLifecycleUserLevelFollowUnchanged(t *testing.T) {
	app := newLifecycleTestApp(t)
	person := createTestActor(t, app, lifecycleLocalPerson, "person", true)
	remote := createTestActor(t, app, lifecycleRemotePerson, "", false)

	fa := lifecycleFollow("https://peer.example.com/api/v1/activitypub/activity/fa", remote.GetString("iri"), person.GetString("iri"))
	// Accept delivery fails to sign in tests; the follows row state is what matters.
	_ = ProcessFollowActivity(app, remote, fa)
	row := lifecycleRow(t, app, remote.Id, person.Id)
	if row == nil || row.GetString("status") != "accepted" {
		t.Fatalf("want one accepted user row, got %v", row)
	}
	if row.GetString("activity_iri") != "" {
		t.Fatalf("user-level follow must keep empty activity_iri, got %q", row.GetString("activity_iri"))
	}

	fb := lifecycleFollow("https://peer.example.com/api/v1/activitypub/activity/fb", remote.GetString("iri"), person.GetString("iri"))
	_ = ProcessFollowActivity(app, remote, fb)
	if n := lifecycleCountRows(t, app); n != 1 {
		t.Fatalf("want 1 user row after second Follow, got %d", n)
	}

	other := lifecycleFollow("https://peer.example.com/api/v1/activitypub/activity/fc", remote.GetString("iri"), person.GetString("iri"))
	if err := ProcessUndoActivity(app, remote, lifecycleUndo("https://peer.example.com/api/v1/activitypub/activity/u9", other)); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if lifecycleRow(t, app, remote.Id, person.Id) != nil {
		t.Fatalf("user-level row should be deleted by Undo-by-pair")
	}
}

func lifecycleSeedFollowRow(t *testing.T, app core.App, followerID, followeeID, status, activityIRI string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("follows")
	if err != nil {
		t.Fatal(err)
	}
	row := core.NewRecord(col)
	row.Set("follower", followerID)
	row.Set("followee", followeeID)
	row.Set("status", status)
	row.Set("activity_iri", activityIRI)
	if err := app.Save(row); err != nil {
		t.Fatalf("seed follow row: %v", err)
	}
	return row
}

func lifecycleSeedFollowActivity(t *testing.T, app core.App, iri, actorIRI, objectIRI string) {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		t.Fatal(err)
	}
	rec := core.NewRecord(col)
	rec.Set("iri", iri)
	rec.Set("type", string(pub.FollowType))
	rec.Set("actor", actorIRI)
	rec.Set("object", objectIRI)
	rec.Set("published", time.Now())
	if err := app.Save(rec); err != nil {
		t.Fatalf("seed follow activity: %v", err)
	}
	time.Sleep(5 * time.Millisecond)
}

// lifecycleLatestActivity returns the newest activity of the given type.
func lifecycleLatestActivity(t *testing.T, app core.App, typ pub.ActivityVocabularyType) *core.Record {
	t.Helper()
	recs, err := app.FindRecordsByFilter("activitypub_activities", "type={:type}", "-created", 1, 0, dbx.Params{"type": string(typ)})
	if err != nil {
		t.Fatalf("find %s activity: %v", typ, err)
	}
	if len(recs) == 0 {
		t.Fatalf("no %s activity stored", typ)
	}
	return recs[0]
}

func lifecycleAssertWraps(t *testing.T, rec *core.Record, want, notWant string) {
	t.Helper()
	obj := rec.GetString("object")
	if !strings.Contains(obj, want) {
		t.Errorf("%s object should reference %s, got %s", rec.GetString("type"), want, obj)
	}
	if notWant != "" && strings.Contains(obj, notWant) {
		t.Errorf("%s object should not reference %s, got %s", rec.GetString("type"), notWant, obj)
	}
}

func TestLifecycleCreateFollowPersistsActivityIRI(t *testing.T) {
	app := newLifecycleTestApp(t)
	local, remote := lifecycleInboundFixture(t, app)

	row := lifecycleSeedFollowRow(t, app, local.Id, remote.Id, "pending", "")
	if err := CreateFollowActivity(app, row); err != nil {
		t.Fatalf("CreateFollowActivity: %v", err)
	}
	reloaded, err := app.FindRecordById("follows", row.Id)
	if err != nil {
		t.Fatal(err)
	}
	iri := reloaded.GetString("activity_iri")
	if iri == "" {
		t.Fatalf("instance follow should persist activity_iri")
	}
	if got := lifecycleLatestActivity(t, app, pub.FollowType).GetString("iri"); got != iri {
		t.Fatalf("stored Follow iri %q != activity_iri %q", got, iri)
	}

	// user-level follow: activity_iri stays empty
	person := createTestActor(t, app, lifecycleLocalPerson, "person", true)
	remotePerson := createTestActor(t, app, lifecycleRemotePerson, "person", false)
	userRow := lifecycleSeedFollowRow(t, app, person.Id, remotePerson.Id, "pending", "")
	if err := CreateFollowActivity(app, userRow); err != nil {
		t.Fatalf("CreateFollowActivity user: %v", err)
	}
	reloaded, err = app.FindRecordById("follows", userRow.Id)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.GetString("activity_iri") != "" {
		t.Fatalf("user-level follow must keep empty activity_iri")
	}
}

func TestLifecycleUnfollowUsesStoredFollowIRI(t *testing.T) {
	app := newLifecycleTestApp(t)
	local, remote := lifecycleInboundFixture(t, app)
	lifecycleSeedFollowActivity(t, app, lifecycleF1, local.GetString("iri"), remote.GetString("iri"))
	lifecycleSeedFollowActivity(t, app, lifecycleF2, local.GetString("iri"), remote.GetString("iri"))

	row := lifecycleSeedFollowRow(t, app, local.Id, remote.Id, "pending", lifecycleF1)
	if err := CreateUnfollowActivity(app, row); err != nil {
		t.Fatalf("CreateUnfollowActivity: %v", err)
	}
	lifecycleAssertWraps(t, lifecycleLatestActivity(t, app, pub.UndoType), lifecycleF1, lifecycleF2)

	// empty activity_iri: newest Follow is used
	row.Set("activity_iri", "")
	if err := CreateUnfollowActivity(app, row); err != nil {
		t.Fatalf("CreateUnfollowActivity fallback: %v", err)
	}
	time.Sleep(5 * time.Millisecond)
	lifecycleAssertWraps(t, lifecycleLatestActivity(t, app, pub.UndoType), lifecycleF2, lifecycleF1)
}

func TestLifecycleAcceptRejectUseStoredFollowIRI(t *testing.T) {
	app := newLifecycleTestApp(t)
	local, remote := lifecycleInboundFixture(t, app)
	lifecycleSeedFollowActivity(t, app, lifecycleF1, remote.GetString("iri"), local.GetString("iri"))
	lifecycleSeedFollowActivity(t, app, lifecycleF2, remote.GetString("iri"), local.GetString("iri"))
	row := lifecycleSeedFollowRow(t, app, remote.Id, local.Id, "pending", lifecycleF1)
	row.Set("activity_iri", lifecycleF1)

	if err := CreateAcceptFollowActivity(app, row); err != nil {
		t.Fatalf("CreateAcceptFollowActivity: %v", err)
	}
	lifecycleAssertWraps(t, lifecycleLatestActivity(t, app, pub.AcceptType), lifecycleF1, lifecycleF2)

	if err := CreateRejectFollowActivity(app, row); err != nil {
		t.Fatalf("CreateRejectFollowActivity: %v", err)
	}
	lifecycleAssertWraps(t, lifecycleLatestActivity(t, app, pub.RejectType), lifecycleF1, lifecycleF2)

	// the stored IRI wins over "newest": point the row at F2
	row.Set("activity_iri", lifecycleF2)
	time.Sleep(5 * time.Millisecond)
	if err := CreateAcceptFollowActivity(app, row); err != nil {
		t.Fatalf("CreateAcceptFollowActivity F2: %v", err)
	}
	lifecycleAssertWraps(t, lifecycleLatestActivity(t, app, pub.AcceptType), lifecycleF2, lifecycleF1)
}

func TestLifecycleStaleAcceptRejectIgnored(t *testing.T) {
	app := newLifecycleTestApp(t)
	local, remote := lifecycleInboundFixture(t, app)
	row := lifecycleSeedFollowRow(t, app, local.Id, remote.Id, "pending", lifecycleF2)

	status := func() string {
		r, err := app.FindRecordById("follows", row.Id)
		if err != nil {
			t.Fatal(err)
		}
		return r.GetString("status")
	}
	followOf := func(id string) *pub.Activity {
		f := lifecycleFollow(id, local.GetString("iri"), remote.GetString("iri"))
		return &f
	}
	accept := func(id string) pub.Activity {
		a := pub.AcceptNew(pub.IRI("https://peer.example.com/api/v1/activitypub/activity/a-"+id[len(id)-2:]), followOf(id))
		a.Actor = pub.IRI(remote.GetString("iri"))
		return *a
	}
	reject := func(id string) pub.Activity {
		a := pub.RejectNew(pub.IRI("https://peer.example.com/api/v1/activitypub/activity/r-"+id[len(id)-2:]), followOf(id))
		a.Actor = pub.IRI(remote.GetString("iri"))
		return *a
	}

	if err := ProcessAcceptActivity(app, remote, accept(lifecycleF1)); err != nil {
		t.Fatalf("stale Accept: %v", err)
	}
	if got := status(); got != "pending" {
		t.Fatalf("stale Accept changed status to %s", got)
	}
	if err := ProcessAcceptActivity(app, remote, accept(lifecycleF2)); err != nil {
		t.Fatalf("Accept F2: %v", err)
	}
	if got := status(); got != "accepted" {
		t.Fatalf("Accept F2 should accept, got %s", got)
	}
	if err := ProcessRejectActivity(app, remote, reject(lifecycleF1)); err != nil {
		t.Fatalf("stale Reject: %v", err)
	}
	if got := status(); got != "accepted" {
		t.Fatalf("stale Reject changed status to %s", got)
	}
	if err := ProcessRejectActivity(app, remote, reject(lifecycleF2)); err != nil {
		t.Fatalf("Reject F2: %v", err)
	}
	if got := status(); got != "rejected" {
		t.Fatalf("Reject F2 should reject, got %s", got)
	}
}

func TestLifecycleAcceptedRowNewFollowResendsAccept(t *testing.T) {
	app := newLifecycleTestApp(t)
	local, remote := lifecycleInboundFixture(t, app)
	lifecycleSeedFollowActivity(t, app, lifecycleF1, remote.GetString("iri"), local.GetString("iri"))
	row := lifecycleSeedFollowRow(t, app, remote.Id, local.Id, "accepted", lifecycleF1)

	f2 := lifecycleFollow(lifecycleF2, remote.GetString("iri"), local.GetString("iri"))
	if err := ProcessFollowActivity(app, remote, f2); err != nil {
		t.Fatalf("re-Follow: %v", err)
	}
	reloaded, err := app.FindRecordById("follows", row.Id)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.GetString("status") != "accepted" || reloaded.GetString("activity_iri") != lifecycleF2 {
		t.Fatalf("want accepted with activity_iri F2, got %s / %s", reloaded.GetString("status"), reloaded.GetString("activity_iri"))
	}
	if _, err := app.FindFirstRecordByData("activitypub_activities", "iri", lifecycleF2); err != nil {
		t.Fatalf("Follow F2 activity not stored: %v", err)
	}
	lifecycleAssertWraps(t, lifecycleLatestActivity(t, app, pub.AcceptType), lifecycleF2, lifecycleF1)
}
