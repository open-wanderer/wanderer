package regions

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pocketbase/pocketbase/core"
)

// ReconcileArchives garbage-collects region_archives rows whose on-disk data
// has vanished. Because the archive files (CacheDir/{path}/vector.pmtiles
// and dem.pmtiles) live outside the database, an admin deleting them by hand —
// or a wiped/rotated cache volume — leaves rows that advertise a download the
// filesystem can no longer serve (a 404, the exact id-vs-path failure class
// this table has already burned us on). Run once at startup, it deletes any
// row for which NEITHER the vector nor the DEM archive file exists on disk,
// unless the row records a failed build: those carry the error the admin page
// and clients show, and the next build pass retries them anyway.
//
// A row marked "error" whose file is on disk goes back to "ready", keeping its
// error message: v0.21.0-app set "error" on a failed rebuild even though the
// previous archive was still there, and nothing else would clear it until the
// next rebuild. This needs no upstream, so it cannot be held up by the failure
// that caused the error.
//
// It also removes `.building` files left behind by a build that was
// interrupted (crash, kill, power loss). No build can be running at startup,
// so any such file is debris.
//
// A deleted row is self-healing: the next build pass recreates it via
// findOrCreateRegionRecord, and RegionsList already reports a leaf with no
// archive row as "building". Rows that still back at least one on-disk file
// are left untouched. Individual delete failures are logged and skipped rather
// than aborting startup; only a failure to enumerate the table is returned.
func ReconcileArchives(app core.App) error {
	removeInterruptedBuilds(app)

	records, err := app.FindAllRecords("region_archives")
	if err != nil {
		return err
	}

	deleted := 0
	for _, record := range records {
		path := record.GetString("path")

		hasVector := archiveFileExists(RegionArchivePath(app, path))
		hasDem := archiveFileExists(RegionDemPath(app, path))
		if hasVector || hasDem {
			markBuiltArchivesReady(app, record, hasVector, hasDem)
			continue
		}
		if record.GetString("status") == "error" || record.GetString("dem_status") == "error" {
			continue
		}

		if err := app.Delete(record); err != nil {
			log.Printf("[regions] reconcile: failed to delete orphaned archive row for %q: %v", path, err)
			continue
		}
		deleted++
		log.Printf("[regions] reconcile: deleted orphaned archive row for %q (no vector/dem file on disk)", path)
	}

	if deleted > 0 {
		log.Printf("[regions] reconcile: removed %d orphaned region_archives row(s)", deleted)
	}

	return nil
}

// markBuiltArchivesReady clears an "error" status on an archive that exists.
func markBuiltArchivesReady(app core.App, record *core.Record, hasVector, hasDem bool) {
	recovered := false
	if hasVector && record.GetString("status") == "error" {
		record.Set("status", "ready")
		recovered = true
	}
	if hasDem && record.GetString("dem_status") == "error" {
		record.Set("dem_status", "ready")
		recovered = true
	}
	if !recovered {
		return
	}

	path := record.GetString("path")
	if err := app.Save(record); err != nil {
		log.Printf("[regions] reconcile: failed to mark built archives of %q ready: %v", path, err)
		return
	}
	log.Printf("[regions] reconcile: marked %q ready, its archive is on disk", path)
}

// removeInterruptedBuilds deletes the temporary `.building` outputs of builds
// that never finished.
func removeInterruptedBuilds(app core.App) {
	leftovers, err := filepath.Glob(filepath.Join(CacheDir(app), "*", "*.pmtiles.building"))
	if err != nil {
		return
	}
	for _, leftover := range leftovers {
		if err := os.Remove(leftover); err != nil {
			log.Printf("[regions] reconcile: failed to remove interrupted build %q: %v", leftover, err)
			continue
		}
		log.Printf("[regions] reconcile: removed interrupted build %q", leftover)
	}
}

// WarnIfExtractorMissing logs a warning at startup when the pmtiles CLI that
// builds region archives is not on PATH. The Docker image ships it; native
// installs have to provide it themselves, and without it every build fails.
func WarnIfExtractorMissing(app core.App) {
	if _, err := exec.LookPath("pmtiles"); err != nil {
		app.Logger().Warn("pmtiles is not on PATH; offline region archives cannot be built until it is installed", "error", err)
	}
}

// archiveFileExists reports whether a regular file exists at path.
func archiveFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
