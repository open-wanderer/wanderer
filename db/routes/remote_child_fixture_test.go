package routes

import (
	"github.com/pocketbase/pocketbase/core"
	"testing"
)

type remoteChildFixture struct {
	*remoteSyncFixture
	target   string
	children *core.Collection
	trail    *core.Record
}

func newRemoteChildFixture(t *testing.T, target string) *remoteChildFixture {
	base := newRemoteSyncFixture(t)
	f := &remoteChildFixture{remoteSyncFixture: base, target: target}
	f.children = map[string]*core.Collection{"waypoint": f.waypoints, "summit_log": f.logs}[target]
	f.trail = f.save(f.trails, map[string]any{"iri": "https://remote.example/api/v1/trail/trail", "author": f.remoteActor.Id, "public": true})
	return f
}
func (f *remoteChildFixture) child(iri, author, trail string) *core.Record {
	return f.save(f.children, map[string]any{"iri": iri, "author": author, "trail": trail, "name": "Original", "text": "Original"})
}
func (f *remoteChildFixture) sync(app core.App, raw []any) error {
	if f.target == "waypoint" {
		return syncWaypoints(app, f.t.Context(), f.trail, "https://remote.example", raw)
	}
	return syncSummitLogs(app, f.t.Context(), f.trail, "https://remote.example", raw)
}
