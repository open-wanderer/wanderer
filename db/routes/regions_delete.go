package routes

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/services/regions"
)

// RegionArchiveDelete removes a leaf region's on-disk vector/DEM pmtiles
// archives (if present) and its region_archives bookkeeping record. Nothing
// else in this codebase ever deletes a built archive (there is no staleness
// cleanup pass), so an admin disabling a region from the catalog had no way
// to reclaim that disk space short of touching the filesystem directly.
//
// Registered superuser-only (RequireSuperuserAuth) at a distinct path from
// the /regions group in main.go, which is intentionally scoped to any
// logged-in app user for the read-only catalog/download flow —
// deleting archives is an admin-only, destructive action and must not share
// that weaker auth gate.
//
// Files are removed idempotently: a file that's already gone is not an
// error, matching the existing download handlers' NotFound-on-missing
// posture rather than surfacing a 500 for a state that isn't actually
// broken (e.g. the admin retries after a partial failure).
func RegionArchiveDelete(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	if !regions.IsValidRegionID(id) {
		return e.BadRequestError("invalid region id", nil)
	}

	if err := removeArchiveFileIfExists(regions.RegionArchivePath(id)); err != nil {
		return e.InternalServerError("failed to delete vector archive", err)
	}
	if err := removeArchiveFileIfExists(regions.RegionDemPath(id)); err != nil {
		return e.InternalServerError("failed to delete dem archive", err)
	}
	// Best-effort: only succeeds if the region's directory is now empty; a
	// non-empty-dir error is expected (e.g. a rebuild raced in and dropped a
	// new file between the two removals above) and deliberately ignored.
	_ = os.Remove(filepath.Join(regions.RegionCacheDir, id))

	records, err := e.App.FindAllRecords("region_archives",
		dbx.NewExp("path = {:path}", dbx.Params{"path": id}),
	)
	if err != nil {
		return e.InternalServerError("failed to load region_archives record", err)
	}
	for _, r := range records {
		if err := e.App.Delete(r); err != nil {
			return e.InternalServerError("failed to delete region_archives record", err)
		}
	}

	return e.JSON(http.StatusOK, map[string]any{"deleted": true})
}

func removeArchiveFileIfExists(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
