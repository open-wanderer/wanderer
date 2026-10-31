package routes

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"pocketbase/federation"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

// seedPendingInstanceFollow creates the remote instance actor on srvURL and a
// pending follows row remote -> local standing for Follow F1. F2 is a later
// Follow of the same pair that is not stored yet.
func seedPendingInstanceFollow(t *testing.T, app core.App, local *core.Record, srvURL string) (remote, row *core.Record, f1, f2 string) {
	t.Helper()
	remote = createFedAdminTestActor(t, app, srvURL+"/actor", "instance", false)
	f1 = srvURL + "/activity/f1"
	f2 = srvURL + "/activity/f2"
	seedFollowActivity(t, app, f1, remote.GetString("iri"), local.GetString("iri"))
	row = createFedAdminTestFollow(t, app, remote.Id, local.Id, "pending")
	row.Set("activity_iri", f1)
	if err := app.Save(row); err != nil {
		t.Fatalf("set activity_iri: %v", err)
	}
	return remote, row, f1, f2
}

func followActivity(remote, local *core.Record, followIRI string) pub.Activity {
	a := pub.FollowNew(pub.IRI(followIRI), pub.IRI(local.GetString("iri")))
	a.Actor = pub.IRI(remote.GetString("iri"))
	return *a
}

// retryFollow delivers a Follow with the given IRI from remote to local, as a
// peer retrying its request does.
func retryFollow(t *testing.T, app core.App, remote, local *core.Record, followIRI string) {
	t.Helper()
	if err := federation.ProcessFollowActivity(app, remote, followActivity(remote, local, followIRI)); err != nil {
		t.Fatalf("ProcessFollowActivity(%s): %v", followIRI, err)
	}
}

// sendUndoFollow delivers an Undo of the given Follow from remote to local.
func sendUndoFollow(app core.App, remote, local *core.Record, followIRI, undoIRI string) error {
	follow := followActivity(remote, local, followIRI)
	undo := pub.UndoNew(pub.IRI(undoIRI), &follow)
	undo.Actor = pub.IRI(remote.GetString("iri"))
	return federation.ProcessUndoActivity(app, remote, *undo)
}

func reloadFollow(t *testing.T, app core.App, id string) *core.Record {
	t.Helper()
	rec, err := app.FindRecordById("follows", id)
	if err != nil {
		t.Fatalf("find follow %s: %v", id, err)
	}
	return rec
}

// assertSingleDelivery waits for one delivery, lets a stray second one show up
// and checks the type and the wrapped Follow of the single delivery.
func assertSingleDelivery(t *testing.T, inbox *recordingInbox, wantType, wantWrapped string) {
	t.Helper()
	inbox.waitForDeliveries(1, 5*time.Second)
	time.Sleep(500 * time.Millisecond)
	got := inbox.waitForDeliveries(1, time.Second)
	if want := []string{wantType}; !slices.Equal(got, want) {
		t.Fatalf("delivered types = %v, want %v", got, want)
	}
	if got := inbox.wrappedIDs(); !slices.Equal(got, []string{wantWrapped}) {
		t.Fatalf("wrapped ids = %v, want [%s]", got, wantWrapped)
	}
}

// TestFederationApproveAnswersCurrentFollow replays this interleaving: the
// approve request loads the pending row, the peer retries its
// Follow (F2) before the status is saved, and the Accept must answer F2.
func TestFederationApproveAnswersCurrentFollow(t *testing.T) {
	app, local := newDisconnectDeliveryTestApp(t)
	inbox := newRecordingInbox(t)
	remote, row, f1, f2 := seedPendingInstanceFollow(t, app, local, inbox.server.URL)

	stale := reloadFollow(t, app, row.Id)
	retryFollow(t, app, remote, local, f2)

	if err := setFollowStatus(app, stale, "accepted", local.Id); err != nil {
		t.Fatalf("setFollowStatus: %v", err)
	}

	current := reloadFollow(t, app, row.Id)
	if current.GetString("status") != "accepted" || current.GetString("activity_iri") != f2 {
		t.Fatalf("row = status %q activity_iri %q, want accepted and %s",
			current.GetString("status"), current.GetString("activity_iri"), f2)
	}
	assertSingleDelivery(t, inbox, "Accept", f2)

	// The peer's Undo of the superseded Follow must not end the relationship.
	if err := sendUndoFollow(app, remote, local, f1, inbox.server.URL+"/activity/u1"); err != nil {
		t.Fatalf("Undo(F1): %v", err)
	}
	after := reloadFollow(t, app, row.Id)
	if after.GetString("status") != "accepted" || after.GetString("activity_iri") != f2 {
		t.Fatalf("after Undo(F1) row = status %q activity_iri %q, want accepted and %s",
			after.GetString("status"), after.GetString("activity_iri"), f2)
	}
}

// TestFederationRejectAnswersCurrentFollow is the reject twin of the test above.
func TestFederationRejectAnswersCurrentFollow(t *testing.T) {
	app, local := newDisconnectDeliveryTestApp(t)
	inbox := newRecordingInbox(t)
	remote, row, _, f2 := seedPendingInstanceFollow(t, app, local, inbox.server.URL)

	stale := reloadFollow(t, app, row.Id)
	retryFollow(t, app, remote, local, f2)

	if err := setFollowStatus(app, stale, "rejected", local.Id); err != nil {
		t.Fatalf("setFollowStatus: %v", err)
	}

	current := reloadFollow(t, app, row.Id)
	if current.GetString("status") != "rejected" || current.GetString("activity_iri") != f2 {
		t.Fatalf("row = status %q activity_iri %q, want rejected and %s",
			current.GetString("status"), current.GetString("activity_iri"), f2)
	}
	assertSingleDelivery(t, inbox, "Reject", f2)
}

// vanishingFollowApp deletes a follows row the first time the handler looks up
// the local instance actor, which happens after the route loaded the row. It
// stands in for the peer's Undo arriving mid-request.
type vanishingFollowApp struct {
	core.App
	followID string
	once     sync.Once
}

func (a *vanishingFollowApp) FindFirstRecordByFilter(collection any, filter string, params ...dbx.Params) (*core.Record, error) {
	if name, ok := collection.(string); ok && name == "activitypub_actors" {
		a.once.Do(func() {
			if row, err := a.App.FindRecordById("follows", a.followID); err == nil {
				_ = a.App.Delete(row)
			}
		})
	}
	return a.App.FindFirstRecordByFilter(collection, filter, params...)
}

func (a *vanishingFollowApp) FindRecordsByFilter(collection any, filter string, sort string, limit int, offset int, params ...dbx.Params) ([]*core.Record, error) {
	if name, ok := collection.(string); ok && name == "activitypub_actors" {
		a.once.Do(func() {
			if row, err := a.App.FindRecordById("follows", a.followID); err == nil {
				_ = a.App.Delete(row)
			}
		})
	}
	return a.App.FindRecordsByFilter(collection, filter, sort, limit, offset, params...)
}

// superuserFollowEvent builds the request event of an admin POST naming the
// follows row id in the path.
func superuserFollowEvent(t *testing.T, app core.App, id string) *core.RequestEvent {
	t.Helper()
	superusers, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
	if err != nil {
		t.Fatalf("find %s: %v", core.CollectionNameSuperusers, err)
	}
	req := httptest.NewRequest(http.MethodPost, "/federation/follow/"+id, nil)
	req.SetPathValue("id", id)
	e := &core.RequestEvent{App: app}
	e.Request = req
	e.Response = httptest.NewRecorder()
	e.Auth = core.NewRecord(superusers)
	return e
}

// TestFederationApproveVanishedFollowIs404 checks that a row withdrawn between
// the route's load and the transaction is a 404 and delivers nothing.
func TestFederationApproveVanishedFollowIs404(t *testing.T) {
	handlers := map[string]func(*core.RequestEvent) error{
		"approve": FederationApprove,
		"reject":  FederationReject,
	}
	for name, handler := range handlers {
		t.Run(name, func(t *testing.T) {
			app, local := newDisconnectDeliveryTestApp(t)
			inbox := newRecordingInbox(t)
			_, row, _, _ := seedPendingInstanceFollow(t, app, local, inbox.server.URL)

			e := superuserFollowEvent(t, &vanishingFollowApp{App: app, followID: row.Id}, row.Id)
			err := handler(e)

			var apiErr *router.ApiError
			if !errors.As(err, &apiErr) || apiErr.Status != http.StatusNotFound {
				t.Fatalf("handler error = %v, want a 404 ApiError", err)
			}
			if got := inbox.waitForDeliveries(1, 500*time.Millisecond); len(got) != 0 {
				t.Fatalf("delivered %v for a withdrawn request, want nothing", got)
			}
		})
	}
}

// TestFederationApproveTwiceSendsOneAccept checks that approving the same
// pending row from two stale copies delivers a single Accept.
func TestFederationApproveTwiceSendsOneAccept(t *testing.T) {
	app, local := newDisconnectDeliveryTestApp(t)
	inbox := newRecordingInbox(t)
	_, row, f1, _ := seedPendingInstanceFollow(t, app, local, inbox.server.URL)

	first := reloadFollow(t, app, row.Id)
	second := reloadFollow(t, app, row.Id)
	for i, copy := range []*core.Record{first, second} {
		if err := setFollowStatus(app, copy, "accepted", local.Id); err != nil {
			t.Fatalf("approval %d: %v", i+1, err)
		}
	}

	assertSingleDelivery(t, inbox, "Accept", f1)
}

// TestFederationApproveSerializesWithRetriedFollow starts a retried Follow from
// inside the approval transaction. It must wait for the commit and then leave
// one accepted row for F2, with an Accept for F2 delivered.
func TestFederationApproveSerializesWithRetriedFollow(t *testing.T) {
	app, local := newDisconnectDeliveryTestApp(t)
	inbox := newRecordingInbox(t)
	remote, row, f1, f2 := seedPendingInstanceFollow(t, app, local, inbox.server.URL)
	stale := reloadFollow(t, app, row.Id)

	var fired atomic.Bool
	retryDone := make(chan struct{})
	var retryErr error
	var finishedInsideTx bool
	app.OnRecordUpdate("follows").BindFunc(func(e *core.RecordEvent) error {
		if fired.CompareAndSwap(false, true) {
			go func() {
				defer close(retryDone)
				retryErr = federation.ProcessFollowActivity(app, remote, followActivity(remote, local, f2))
			}()
			// With the approval's transaction open, the retry blocks and the wait expires.
			select {
			case <-retryDone:
				finishedInsideTx = true
			case <-time.After(300 * time.Millisecond):
			}
		}
		return e.Next()
	})

	if err := setFollowStatus(app, stale, "accepted", local.Id); err != nil {
		t.Fatalf("setFollowStatus: %v", err)
	}
	if !fired.Load() {
		t.Fatalf("update hook never fired; the interleaving was not injected")
	}
	select {
	case <-retryDone:
	case <-time.After(5 * time.Second):
		t.Fatalf("injected Follow(F2) did not finish within 5s")
	}
	if finishedInsideTx {
		t.Fatalf("Follow(F2) finished while the approval transaction was open")
	}
	if retryErr != nil {
		t.Fatalf("injected Follow(F2): %v", retryErr)
	}

	rows, err := app.FindAllRecords("follows")
	if err != nil {
		t.Fatalf("list follows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("want exactly 1 follows row, got %d", len(rows))
	}
	got := rows[0]
	if got.GetString("status") != "accepted" || got.GetString("activity_iri") != f2 {
		t.Fatalf("row = status %q activity_iri %q, want accepted and %s",
			got.GetString("status"), got.GetString("activity_iri"), f2)
	}

	if types := inbox.waitForDeliveries(2, 5*time.Second); !slices.Contains(types, "Accept") || len(types) < 2 {
		t.Fatalf("delivered types = %v, want two Accepts", types)
	}
	if !slices.Contains(inbox.wrappedIDs(), f2) {
		t.Fatalf("wrapped ids = %v, want one wrapping %s", inbox.wrappedIDs(), f2)
	}

	if err := sendUndoFollow(app, remote, local, f1, inbox.server.URL+"/activity/u1"); err != nil {
		t.Fatalf("Undo(F1): %v", err)
	}
	if after := reloadFollow(t, app, row.Id); after.GetString("status") != "accepted" {
		t.Fatalf("after Undo(F1) status = %q, want accepted", after.GetString("status"))
	}
}

// TestFederationApproveOutboundIsNotInbound checks the direction guard on an
// outbound row.
func TestFederationApproveOutboundIsNotInbound(t *testing.T) {
	app := newFederationAdminTestApp(t)
	local := createFedAdminTestActor(t, app, "https://local.example.com/api/v1/activitypub/instance", "instance", true)
	remote := createFedAdminTestActor(t, app, "https://remote.example.com/api/v1/activitypub/instance", "instance", false)
	out := createFedAdminTestFollow(t, app, local.Id, remote.Id, "pending")

	err := setFollowStatus(app, reloadFollow(t, app, out.Id), "accepted", local.Id)
	if !errors.Is(err, errNotInboundFollow) {
		t.Fatalf("err = %v, want errNotInboundFollow", err)
	}
	assertFollowStatus(t, app, "outbound", out.Id, "pending")
}

// TestDisconnectPeerRejectsCurrentFollow checks that disconnecting from a
// stale copy of a pending inbound row rejects the Follow the row now stands for.
func TestDisconnectPeerRejectsCurrentFollow(t *testing.T) {
	app, local := newDisconnectDeliveryTestApp(t)
	inbox := newRecordingInbox(t)
	remote, row, _, f2 := seedPendingInstanceFollow(t, app, local, inbox.server.URL)

	stale := reloadFollow(t, app, row.Id)
	retryFollow(t, app, remote, local, f2)

	if err := disconnectPeer(app, local.Id, stale); err != nil {
		t.Fatalf("disconnectPeer: %v", err)
	}

	current := reloadFollow(t, app, row.Id)
	if current.GetString("status") != "rejected" || current.GetString("activity_iri") != f2 {
		t.Fatalf("row = status %q activity_iri %q, want rejected and %s",
			current.GetString("status"), current.GetString("activity_iri"), f2)
	}
	assertSingleDelivery(t, inbox, "Reject", f2)
}

// TestDisconnectPeerVanishedInboundIsDone checks that a request the peer
// withdrew since the row was loaded counts as already ended.
func TestDisconnectPeerVanishedInboundIsDone(t *testing.T) {
	app, local := newDisconnectDeliveryTestApp(t)
	inbox := newRecordingInbox(t)
	remote, row, f1, _ := seedPendingInstanceFollow(t, app, local, inbox.server.URL)

	stale := reloadFollow(t, app, row.Id)
	if err := sendUndoFollow(app, remote, local, f1, inbox.server.URL+"/activity/u1"); err != nil {
		t.Fatalf("Undo(F1): %v", err)
	}
	assertFollowStatus(t, app, "inbound", row.Id, "deleted")

	if err := disconnectPeer(app, local.Id, stale); err != nil {
		t.Fatalf("disconnectPeer: %v", err)
	}
	if got := inbox.waitForDeliveries(1, 500*time.Millisecond); len(got) != 0 {
		t.Fatalf("delivered %v for a withdrawn request, want nothing", got)
	}
}
