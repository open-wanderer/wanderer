package federation

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/doyensec/safeurl"
	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/util"
)

// A peer whose ORIGIN has a port serves attachments from that port. These tests
// use the production client: the port is allowed, but loopback is refused by
// the IP policy, so the server receives nothing.

type portOrigin struct {
	base  string
	hits  *attachmentHits
	actor *core.Record
}

// portOriginSetup starts a server, builds http://127.0.0.1:<P> and creates a
// remote person actor on that host:port.
func portOriginSetup(t *testing.T, f *attachmentFixture) portOrigin {
	t.Helper()
	server, hits := attachmentServer(t, attachmentSmallBody)
	port := server.Listener.Addr().(*net.TCPAddr).Port
	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	actor := createTestActor(t, f.app, base+"/api/v1/activitypub/user/porty", "person", false)
	return portOrigin{base: base, hits: hits, actor: actor}
}

func portRequireIPRefusal(t *testing.T, err error, hits *attachmentHits) {
	t.Helper()
	if err == nil {
		t.Fatal("activity with a loopback attachment was accepted")
	}
	var ipErr *safeurl.AllowedIPError
	if !errors.As(err, &ipErr) {
		t.Errorf("error = %v, want the IP policy (*safeurl.AllowedIPError)", err)
	}
	var portErr *safeurl.AllowedPortError
	if errors.As(err, &portErr) {
		t.Errorf("error = %v, refused by the port policy, want the IP policy", err)
	}
	if n := hits.total(); n != 0 {
		t.Errorf("server received %d requests, want 0", n)
	}
}

func TestTrailCreateFromNonDefaultPortOriginPassesPortPolicy(t *testing.T) {
	f := attachmentApp(t)
	t.Cleanup(util.SetRemoteAttachmentClientForTesting(nil))
	o := portOriginSetup(t, f)

	act := feedDedupTrailCreate(o.actor.GetString("iri"), o.base+"/api/v1/trail/t1")
	act.Object.(*pub.Object).Attachment = pub.ItemCollection{attachmentGPX(o.base + "/api/v1/trail/t1/file.gpx")}
	err := ProcessCreateOrUpdateActivity(f.app, context.Background(), o.actor, f.owner.alice, act)

	portRequireIPRefusal(t, err, o.hits)
}

func TestSummitLogCreateFromNonDefaultPortOriginPassesPortPolicy(t *testing.T) {
	f := attachmentApp(t)
	t.Cleanup(util.SetRemoteAttachmentClientForTesting(nil))
	o := portOriginSetup(t, f)

	trailIRI := o.base + "/api/v1/trail/t1"
	seedTrailRecord(t, f.app, trailIRI, true, o.actor.Id)
	act := ownerSummitLogActivity(pub.CreateType, o.actor.GetString("iri"), o.base+"/api/v1/summit-log/s2", trailIRI, "summit")
	act.Object.(*pub.Object).Attachment = pub.ItemCollection{attachmentGPX(o.base + "/api/v1/summit-log/s2/file.gpx")}
	err := ProcessCreateOrUpdateActivity(f.app, context.Background(), o.actor, f.owner.alice, act)

	portRequireIPRefusal(t, err, o.hits)
}

func TestListCreateFromNonDefaultPortOriginPassesPortPolicy(t *testing.T) {
	f := attachmentApp(t)
	t.Cleanup(util.SetRemoteAttachmentClientForTesting(nil))
	o := portOriginSetup(t, f)

	act := feedDedupListCreate(o.actor.GetString("iri"), o.base+"/api/v1/list/l1")
	act.Object.(*pub.Object).Attachment = pub.ItemCollection{attachmentImage(o.base + "/api/v1/list/l1/avatar.png")}
	err := ProcessCreateOrUpdateActivity(f.app, context.Background(), o.actor, f.owner.alice, act)

	portRequireIPRefusal(t, err, o.hits)
}
