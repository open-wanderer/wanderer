package routes

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"pocketbase/federation"
	"pocketbase/hooks"

	"github.com/pocketbase/pocketbase/core"
)

// newDisconnectDeliveryTestApp extends the admin fixture with the
// activitypub_activities collection, the instance follow hooks and a local
// instance actor with a real key pair.
func newDisconnectDeliveryTestApp(t *testing.T) (core.App, *core.Record) {
	t.Helper()
	app := newFederationAdminTestApp(t)

	activitiesJSON := `[{
		"id": "pbc_3752774184",
		"name": "activitypub_activities",
		"type": "base",
		"system": false,
		"listRule": "",
		"viewRule": "",
		"createRule": null,
		"updateRule": null,
		"deleteRule": null,
		"fields": [
			{"autogeneratePattern":"[a-z0-9]{15}","hidden":false,"id":"text3208210256","max":15,"min":15,"name":"id","pattern":"^[a-z0-9]+$","presentable":false,"primaryKey":true,"required":true,"system":true,"type":"text"},
			{"exceptDomains":[],"hidden":false,"id":"url2434853685","name":"iri","onlyDomains":[],"presentable":false,"required":true,"system":false,"type":"url"},
			{"autogeneratePattern":"","hidden":false,"id":"text2363381545","max":0,"min":0,"name":"type","pattern":"","presentable":false,"primaryKey":false,"required":true,"system":false,"type":"text"},
			{"hidden":false,"id":"json3616002756","maxSize":0,"name":"to","presentable":false,"required":false,"system":false,"type":"json"},
			{"hidden":false,"id":"json3685882489","maxSize":0,"name":"cc","presentable":false,"required":false,"system":false,"type":"json"},
			{"hidden":false,"id":"json2893285722","maxSize":0,"name":"object","presentable":false,"required":true,"system":false,"type":"json"},
			{"exceptDomains":[],"hidden":false,"id":"url1148540665","name":"actor","onlyDomains":[],"presentable":false,"required":true,"system":false,"type":"url"},
			{"hidden":false,"id":"date1748787223","max":"","min":"","name":"published","presentable":false,"required":true,"system":false,"type":"date"},
			{"autogeneratePattern":"","hidden":false,"id":"text1653163849","max":15,"min":15,"name":"relation","pattern":"^[a-z0-9]+$","presentable":false,"primaryKey":false,"required":false,"system":false,"type":"text"},
			{"hidden":false,"id":"autodate2990389176","name":"created","onCreate":true,"onUpdate":false,"presentable":false,"system":false,"type":"autodate"},
			{"hidden":false,"id":"autodate3332085495","name":"updated","onCreate":true,"onUpdate":true,"presentable":false,"system":false,"type":"autodate"}
		],
		"indexes": [],
		"system": false
	}]`
	if err := app.ImportCollectionsByMarshaledJSON([]byte(activitiesJSON), false); err != nil {
		t.Fatalf("create activitypub_activities collection: %v", err)
	}

	app.OnRecordAfterUpdateSuccess("follows").BindFunc(hooks.InstanceFollowUpdateHandler())
	app.OnRecordAfterDeleteSuccess("follows").BindFunc(hooks.InstanceFollowDeleteHandler())

	if err := federation.InitInstanceActor(app); err != nil {
		t.Fatalf("InitInstanceActor: %v", err)
	}
	local, err := findLocalInstanceActor(app)
	if err != nil {
		t.Fatalf("findLocalInstanceActor: %v", err)
	}
	return app, local
}

// recordingInbox is an httptest peer inbox that records the "type" of every
// POSTed activity.
type recordingInbox struct {
	server *httptest.Server
	mu     sync.Mutex
	types  []string
}

func newRecordingInbox(t *testing.T) *recordingInbox {
	t.Helper()
	r := &recordingInbox{}
	r.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost {
			body, _ := io.ReadAll(req.Body)
			var doc struct {
				Type string `json:"type"`
			}
			_ = json.Unmarshal(body, &doc)
			r.mu.Lock()
			r.types = append(r.types, doc.Type)
			r.mu.Unlock()
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(r.server.Close)
	return r
}

// waitForDeliveries polls until at least want activities arrived or the
// deadline passes, then returns the sorted types received so far.
func (r *recordingInbox) waitForDeliveries(want int, deadline time.Duration) []string {
	stop := time.Now().Add(deadline)
	for time.Now().Before(stop) {
		r.mu.Lock()
		n := len(r.types)
		r.mu.Unlock()
		if n >= want {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := slices.Clone(r.types)
	slices.Sort(out)
	return out
}

func seedFollowActivity(t *testing.T, app core.App, iri, actorIRI, objectIRI string) {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("activitypub_activities")
	if err != nil {
		t.Fatalf("find activitypub_activities: %v", err)
	}
	r := core.NewRecord(col)
	r.Set("iri", iri)
	r.Set("type", "Follow")
	r.Set("actor", actorIRI)
	r.Set("object", objectIRI)
	r.Set("published", time.Now())
	if err := app.Save(r); err != nil {
		t.Fatalf("save follow activity %s: %v", iri, err)
	}
}

// seedDisconnectMutualPeer creates the remote instance actor on the recording
// server, the Follow activities of both directions and two accepted follows
// rows. It returns the remote actor and the outbound and inbound rows.
func seedDisconnectMutualPeer(t *testing.T, app core.App, local *core.Record, srvURL string) (remote, out, in *core.Record) {
	t.Helper()
	remote = createFedAdminTestActor(t, app, srvURL+"/actor", "instance", false)
	localIRI := local.GetString("iri")
	remoteIRI := remote.GetString("iri")

	seedFollowActivity(t, app, "https://local.example.com/api/v1/activitypub/activity/out1", localIRI, remoteIRI)
	seedFollowActivity(t, app, srvURL+"/activity/in1", remoteIRI, localIRI)

	out = createFedAdminTestFollow(t, app, local.Id, remote.Id, "accepted")
	in = createFedAdminTestFollow(t, app, remote.Id, local.Id, "accepted")
	return remote, out, in
}

// TestDisconnectPeerDeliversUndoAndReject checks that a mutual disconnect
// delivers exactly one Undo and one Reject to the peer inbox.
func TestDisconnectPeerDeliversUndoAndReject(t *testing.T) {
	app, local := newDisconnectDeliveryTestApp(t)
	inbox := newRecordingInbox(t)
	_, out, in := seedDisconnectMutualPeer(t, app, local, inbox.server.URL)

	if err := disconnectPeer(app, local.Id, out); err != nil {
		t.Fatalf("disconnectPeer: %v", err)
	}

	inbox.waitForDeliveries(2, 5*time.Second)
	time.Sleep(200 * time.Millisecond) // let a stray extra delivery show up
	got := inbox.waitForDeliveries(2, time.Second)
	want := []string{"Reject", "Undo"}
	if !slices.Equal(got, want) {
		t.Fatalf("delivered types = %v, want %v", got, want)
	}

	assertFollowStatus(t, app, "outbound", out.Id, "deleted")
	assertFollowStatus(t, app, "inbound", in.Id, "rejected")
}
