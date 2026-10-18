package regions

import (
	"log"
	"os"

	"github.com/pocketbase/pocketbase/core"
)

// ReconcileArchives garbage-collects region_archives rows whose on-disk data
// has vanished. Because the archive files (RegionCacheDir/{path}/vector.pmtiles
// and dem.pmtiles) live outside the database, an admin deleting them by hand —
// or a wiped/rotated cache volume — leaves rows that advertise a download the
// filesystem can no longer serve (a 404, the exact id-vs-path failure class
// this table has already burned us on). Run once at startup, it deletes any
// row for which NEITHER the vector nor the DEM archive file exists on disk.
//
// A deleted row is self-healing: the next build pass recreates it via
// findOrCreateRegionRecord, and RegionsList already reports a leaf with no
// archive row as "building". Rows that still back at least one on-disk file
// are left untouched. Individual delete failures are logged and skipped rather
// than aborting startup; only a failure to enumerate the table is returned.
func ReconcileArchives(app core.App) error {
	records, err := app.FindAllRecords("region_archives")
	if err != nil {
		return err
	}

	deleted := 0
	for _, record := range records {
		path := record.GetString("path")

		if archiveFileExists(RegionArchivePath(path)) || archiveFileExists(RegionDemPath(path)) {
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

// archiveFileExists reports whether a regular file exists at path.
func archiveFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
