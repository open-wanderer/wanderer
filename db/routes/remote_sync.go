package routes

import (
	"errors"
	"net/http"
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/core"
)

var newRemoteSyncHTTPClient = util.SafeHTTPClient

// errRemoteGone marks a sync the origin answered with 404 or 410: the object
// was deleted there, and a copy here must not outlive it. A Delete that never
// arrived, because the origin did not know this instance held a copy, is
// caught up with this way the next time the copy is refreshed.
//
// A 403 is deliberately not gone: the sync carries no credentials, so 403
// means the object was taken private, and dropping the copy would cascade
// away comments and summit logs local users left on it. Copies of content
// taken private are left to a later tombstone mechanism.
var errRemoteGone = errors.New("remote object gone")

func isGoneStatus(code int) bool {
	return code == http.StatusNotFound || code == http.StatusGone
}

// backgroundSyncFailed records a refresh that failed without anybody to
// report it to, so a copy that cannot be brought up to date is at least
// visible in the logs.
func backgroundSyncFailed(app core.App, collection, iri string, err error) {
	app.Logger().Warn("background sync of a remote object failed", "collection", collection, "iri", iri, "error", err)
}

// dropGoneRemoteRecord removes the local copy of a remote object whose origin
// reports it gone. Local content is never touched: a local IRI is not synced
// in the first place, and this repeats that guarantee where the deletion
// happens. Deleting through the app runs the usual hooks, so the search index
// and feeds are cleaned up as for an inbound Delete.
func dropGoneRemoteRecord(app core.App, collection, id string) {
	if id == "" {
		return
	}
	record, err := app.FindRecordById(collection, id)
	if err != nil {
		return
	}
	iri := record.GetString("iri")
	if iri == "" || util.IsLocalIRI(iri) {
		return
	}
	if err := app.Delete(record); err != nil {
		app.Logger().Error("could not drop copy of a remote object reported gone", "collection", collection, "iri", iri, "error", err)
		return
	}
	app.Logger().Info("dropped copy of a remote object reported gone", "collection", collection, "iri", iri)
}

// stripLocalSyncFields removes local sync state from a remote payload.
// Remote values must not mark a placeholder as complete or discard a
// previously completed full sync.
func stripLocalSyncFields(data map[string]any) {
	delete(data, "needs_full_sync")
	delete(data, "full_sync_completed")
}
