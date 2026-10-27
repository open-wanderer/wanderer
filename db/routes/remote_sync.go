package routes

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/pocketbase/pocketbase/apis"
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/core"
)

var newRemoteSyncHTTPClient = util.SafeHTTPClient

// SetRemoteSyncClientForTesting replaces the client remote objects are
// synchronized and confirmed with and returns a restore function. Tests only.
func SetRemoteSyncClientForTesting(f func() *http.Client) (restore func()) {
	previous := newRemoteSyncHTTPClient
	newRemoteSyncHTTPClient = f
	return func() { newRemoteSyncHTTPClient = previous }
}

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
	if iri == "" || util.IsLocalIRI(iri) || remoteRecordHasLocalAuthor(app, record) {
		return
	}
	if err := app.Delete(record); err != nil {
		app.Logger().Error("could not drop copy of a remote object reported gone", "collection", collection, "iri", iri, "error", err)
		return
	}
	app.Logger().Info("dropped copy of a remote object reported gone", "collection", collection, "iri", iri)
}

func remoteRecordHasLocalAuthor(app core.App, record *core.Record) bool {
	author, err := app.FindRecordById("activitypub_actors", record.GetString("author"))
	return err == nil && author.GetBool("is_local")
}

// A local handle selects records as a lookup without a handle. Remote copies
// refresh from their stored IRI; remote handles select their own origin.
// The route still checks the selected record's current view rule.
func remoteHandleSelectsRecord(app core.App, record *core.Record, actor *core.Record, origin string) bool {
	if actor.GetBool("is_local") {
		return true
	}
	return !remoteRecordHasLocalAuthor(app, record) && (remoteSyncIRIMatchesOrigin(record.GetString("iri"), origin) || (actor.Id != "" && record.GetString("author") == actor.Id))
}

// stripLocalSyncFields removes local sync state from a remote payload.
// Remote values must not mark a placeholder as complete or discard a
// previously completed full sync.
func stripLocalSyncFields(data map[string]any) {
	delete(data, "needs_full_sync")
	delete(data, "full_sync_completed")
}

// A rejected full document is distinct from an unreachable origin. Build the
// link from stored identity fields, never from response content.
func remoteDocumentLimitError(app core.App, record *core.Record, collection string) error {
	origin, err := url.Parse(record.GetString("iri"))
	if err != nil || (origin.Scheme != "http" && origin.Scheme != "https") || origin.Host == "" || origin.User != nil {
		return apis.NewApiError(http.StatusUnprocessableEntity, "This content is too large to load here.", nil)
	}
	kind := strings.TrimSuffix(collection, "s")
	prefix := "/api/v1/" + kind + "/"
	id := strings.TrimPrefix(origin.Path, prefix)
	if !strings.HasPrefix(origin.Path, prefix) || id == "" || strings.Contains(id, "/") {
		return apis.NewApiError(http.StatusUnprocessableEntity, "This content is too large to load here.", nil)
	}
	actor, err := app.FindRecordById("activitypub_actors", record.GetString("author"))
	if err != nil || actor.GetString("preferred_username") == "" {
		return apis.NewApiError(http.StatusUnprocessableEntity, "This content is too large to load here.", nil)
	}
	path := "/trail/view/"
	if collection == "lists" {
		path = "/lists/"
	}
	origin.Path = path + "@" + actor.GetString("preferred_username") + "/" + id
	origin.RawPath, origin.RawQuery, origin.Fragment = "", "", ""
	response := apis.NewApiError(http.StatusUnprocessableEntity, "This content is too large to load here. You can view it on its original instance.", nil)
	response.Data = map[string]any{"code": "remote_document_limit", "origin_url": origin.String()}
	return response
}

func isRemoteDocumentLimitError(err error) bool {
	return errors.Is(err, util.ErrFederationBodyTooLarge) || errors.Is(err, errRemoteSyncJSONComplexity)
}
