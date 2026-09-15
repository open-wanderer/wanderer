package routes

import "pocketbase/util"

var newRemoteSyncHTTPClient = util.SafeHTTPClient

// stripLocalSyncFields removes local sync state from a remote payload.
// Remote values must not mark a placeholder as complete or discard a
// previously completed full sync.
func stripLocalSyncFields(data map[string]any) {
	delete(data, "needs_full_sync")
	delete(data, "full_sync_completed")
}
