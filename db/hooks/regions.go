package hooks

import (
	"fmt"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// ValidateRegionPathReferenceHandler enforces the path-based "foreign key"
// that connects region_archives and region_geometry back to the regions
// table. Neither collection uses a PocketBase relation field — PocketBase
// relations reference a record's opaque id, but these tables deliberately
// join on the regions.path materialized key (provably unique and stable
// across re-seeds, and the same string every on-disk archive dir and download
// URL is named after; see db/migrations/1785000000_create_regions_collection.go).
// A relation would trade that stable natural key for an id that changes on
// every re-seed, orphaning the expensive pre-built archives on disk.
//
// This runs as an OnRecordCreate/OnRecordUpdate MODEL hook (not the *Request
// variant) so it also covers the internal writes that actually create these
// rows — services/regions/builder.go's app.Save for archives and the
// on-demand geometry fetch path's persist-on-enabled write for geometry —
// not just API traffic (of which these internal-only collections see none).
// A row whose path has no matching regions row is rejected, giving the
// referential guarantee a relation would otherwise provide.
func ValidateRegionPathReferenceHandler() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		path := e.Record.GetString("path")
		if path == "" {
			return fmt.Errorf("%s.path is required and must reference a regions.path", e.Record.Collection().Name)
		}

		region, err := e.App.FindFirstRecordByFilter(
			"regions",
			"path = {:path}",
			dbx.Params{"path": path},
		)
		if err != nil || region == nil {
			return fmt.Errorf("%s.path %q does not reference an existing regions.path", e.Record.Collection().Name, path)
		}

		return e.Next()
	}
}
