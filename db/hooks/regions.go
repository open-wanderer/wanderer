package hooks

import (
	"fmt"
	"log"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/services/regions"
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

// CacheGeometryOnEnableHandler caches a leaf region's boundary geometry the
// moment the region is enabled, by calling regions.ResolveGeometry — whose
// persist branch fires precisely because the record is now enabled.
//
// Why this exists as a server-side hook rather than a client call: the
// original design assumed the admin picker's toggle-on redraw would populate
// region_geometry as a side effect. It did not, for two compounding reasons.
// The redraw read the collection directly instead of the read-through route,
// so it never triggered a fetch at all; and it ran *before* the enabling PATCH
// resolved, so even a route call would have seen enabled=false and taken the
// disabled pass-through with no write. The result was that nothing ever wrote
// region_geometry from the admin flow — rows only appeared when the archive
// cron eventually ran buildRegion.
//
// Keying off the record transition instead of a UI call closes that hole for
// every path that can enable a region: the picker, a direct REST PATCH, or the
// PocketBase collection editor. The admin map therefore has coverage geometry
// to draw as soon as a region is enabled, rather than only after the next cron
// build.
//
// Bound to OnRecordAfterUpdateSuccess (not OnRecordUpdate) deliberately: the
// row must be committed before ResolveGeometry reads its enabled flag back,
// and the geometry write must not join the enabling update's transaction.
//
// The fetch runs in a goroutine so a ~hundreds-of-ms upstream CoMaps request
// never blocks the PATCH response — the picker's bulk enable PATCHes leaves in
// chunks, and serialising an upstream fetch into each one would make enabling
// a country crawl. Callers that need geometry synchronously use the
// /regions/{path}/geometry route, which is read-through and therefore correct
// regardless of whether this background write has landed yet.
func CacheGeometryOnEnableHandler(app core.App) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		if err := e.Next(); err != nil {
			return err
		}

		if e.Record.GetString("kind") != "leaf" || !e.Record.GetBool("enabled") {
			return nil
		}

		// Only act on a false -> true transition. Without this, every unrelated
		// update to an already-enabled region (a re-save, an admin edit) would
		// re-enter ResolveGeometry.
		if original := e.Record.Original(); original != nil && original.GetBool("enabled") {
			return nil
		}

		path := e.Record.GetString("path")

		go func() {
			// Re-read the record inside the goroutine rather than closing over
			// e.Record: the event's record is mutable and may be reused after
			// this hook returns, and re-reading also confirms the enable
			// actually committed.
			region, err := app.FindFirstRecordByFilter("regions", "path = {:path}", dbx.Params{"path": path})
			if err != nil || region == nil {
				log.Printf("[regions] geometry cache-on-enable: region %s not found after update: %v", path, err)
				return
			}
			if _, _, err := regions.ResolveGeometry(app, region); err != nil {
				// Non-fatal by design: the region is enabled either way, and
				// buildRegion's self-heal will refetch at build time.
				// Losing the cache write only costs the admin map its outline
				// until then.
				log.Printf("[regions] geometry cache-on-enable failed for %s: %v", path, err)
			}
		}()

		return nil
	}
}
