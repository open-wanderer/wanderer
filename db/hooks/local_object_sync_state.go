package hooks

import "github.com/pocketbase/pocketbase/core"

// PreserveLocalObjectSyncState keeps identity and remote bookkeeping owned by
// the server. Internal imports and syncs write through model hooks instead.
func PreserveLocalObjectSyncState(fields ...string) func(*core.RecordRequestEvent) error {
	if len(fields) == 0 {
		fields = []string{"iri", "needs_full_sync", "full_sync_completed"}
	}
	return func(e *core.RecordRequestEvent) error {
		if e.HasSuperuserAuth() {
			return e.Next()
		}
		for _, field := range fields {
			e.Record.Set(field, e.Record.Original().GetRaw(field))
		}
		return e.Next()
	}
}
