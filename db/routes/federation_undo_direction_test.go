package routes

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"pocketbase/federation"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/dbx"
)

// TestInboundUndoSendsNoWrongDirectionUndo checks that deleting an inbound
// follows row stores no Undo, while deleting an outbound row delivers one.
func TestInboundUndoSendsNoWrongDirectionUndo(t *testing.T) {
	app, local := newDisconnectDeliveryTestApp(t)
	inbox := newRecordingInbox(t)
	remote, out, in := seedDisconnectMutualPeer(t, app, local, inbox.server.URL)
	remoteIRI := remote.GetString("iri")

	follow := pub.FollowNew(pub.IRI(inbox.server.URL+"/activity/in1"), pub.IRI(local.GetString("iri")))
	follow.Actor = pub.IRI(remoteIRI)
	undo := pub.UndoNew(pub.IRI(inbox.server.URL+"/activity/undo1"), follow)
	undo.Actor = pub.IRI(remoteIRI)

	if err := federation.ProcessUndoActivity(app, remote, *undo); err != nil {
		t.Fatalf("ProcessUndoActivity: %v", err)
	}

	if _, err := app.FindRecordById("follows", in.Id); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("inbound row must be gone, find error = %v", err)
	}

	var wrong int
	if err := app.DB().
		NewQuery("SELECT COUNT(*) FROM activitypub_activities WHERE type = 'Undo' AND actor = {:actor}").
		Bind(dbx.Params{"actor": remoteIRI}).
		Row(&wrong); err != nil {
		t.Fatalf("count undo activities: %v", err)
	}
	if wrong != 0 {
		t.Fatalf("%d Undo activities attributed to the peer actor, want 0", wrong)
	}

	if err := app.Delete(out); err != nil {
		t.Fatalf("delete outbound row: %v", err)
	}
	got := inbox.waitForDeliveries(1, 5*time.Second)
	time.Sleep(200 * time.Millisecond)
	got = inbox.waitForDeliveries(1, time.Second)
	if len(got) != 1 || got[0] != "Undo" {
		t.Fatalf("delivered types = %v, want exactly one Undo", got)
	}
}
