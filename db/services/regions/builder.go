package regions

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	dbx "github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const (
	// regionVectorMaxZoom mirrors the per-cell tile generator's vector
	// maxzoom (14) — kept in lockstep intentionally, but declared as a local,
	// parallel-path constant rather than a shared import.
	regionVectorMaxZoom = 14

	// regionDemMaxZoom mirrors the per-cell tile generator's DEM maxzoom
	// (12) — must stay in lockstep with that constant and the Dart
	// offline_style_rewriter.dart `_offlineDemMaxZoom` constant.
	regionDemMaxZoom = 12

	// mapterhornSource is Mapterhorn's downloadable global DEM pmtiles
	// archive. Mirrors the per-cell tile generator's own constant — a local,
	// parallel-path copy, not a shared import.
	mapterhornSource = "https://download.mapterhorn.com/planet.pmtiles"
)

var (
	// inFlightMu / inFlight dedupe concurrent builds of the same region,
	// keyed by region id (regions.path) — mirrors the per-cell generator's
	// in-flight guard, re-keyed for region-scale builds.
	inFlightMu sync.Mutex
	inFlight   = map[string]*sync.WaitGroup{}

	// syncMu / syncRunning guard against two full BuildAll passes running
	// at once — whether cron-vs-cron (a slow prior run still going when the
	// next scheduled tick fires), cron-vs-manual, or manual-vs-manual (an
	// admin double-clicking "Sync now"). This is a coarser guard than
	// inFlight above: inFlight already dedupes concurrent work on any single
	// region, but two overlapping BuildAll loops would otherwise still both
	// iterate the full leaf list and contend/wait on every in-flight region
	// for no benefit, and there'd be no single "is a sync running" signal
	// for the admin UI to poll.
	syncMu      sync.Mutex
	syncRunning bool
)

// TryStartSync atomically claims the single global "a BuildAll pass is
// running" slot, returning false (claiming nothing) if one is already
// running. Callers that get true MUST call FinishSync when done (typically
// via defer), including on the goroutine driving a BuildAllLocked call.
func TryStartSync() bool {
	syncMu.Lock()
	defer syncMu.Unlock()
	if syncRunning {
		return false
	}
	syncRunning = true
	return true
}

// FinishSync releases the slot claimed by a prior successful TryStartSync.
func FinishSync() {
	syncMu.Lock()
	syncRunning = false
	syncMu.Unlock()
}

// SyncRunning reports whether a BuildAll pass (cron-triggered or manually
// triggered via the admin "Sync now" button) is currently in progress.
func SyncRunning() bool {
	syncMu.Lock()
	defer syncMu.Unlock()
	return syncRunning
}

// regionExtractTimeout bounds each region-scale pmtiles extract subprocess.
// It is intentionally distinct from (and longer than) the per-cell
// generator's extractTimeout (5m, tuned for 0.5° grid cells) — a region bbox
// can cover a metro area or an entire state, which needs much more time.
// Overridable via REGION_ARCHIVE_EXTRACT_TIMEOUT (a Go duration string,
// e.g. "45m"); defaults to 30 minutes when unset or unparseable.
func regionExtractTimeout() time.Duration {
	if v := os.Getenv("REGION_ARCHIVE_EXTRACT_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return 30 * time.Minute
}

// BuildAll is the cron entrypoint: it claims the sync slot (skipping this
// invocation entirely, logged, if one is already running — see syncRunning)
// and runs BuildAllLocked. This must never panic or block the cron.
func BuildAll(app core.App) {
	if !TryStartSync() {
		log.Printf("[regions] BuildAll already in progress, skipping this invocation")
		return
	}
	defer FinishSync()
	BuildAllLocked(app)
}

// BuildAllLocked queries the seeded `regions` table for every enabled leaf
// and pre-builds each region's vector + DEM archives, one
// region fully completing before the next starts (bounds subprocess
// concurrency to 1). A query error or a single region's build failure is
// logged and never aborts the whole pass.
//
// Callers MUST already hold the sync slot (via a successful TryStartSync,
// released with FinishSync) — this function does not claim it itself, so
// that a caller driving it from a goroutine (the manual "Sync now" HTTP
// route) can claim the slot synchronously and answer its request
// immediately with whether a sync actually started, before the goroutine
// (and thus this function) has even begun running.
func BuildAllLocked(app core.App) {
	leafRecords, err := app.FindAllRecords("regions",
		dbx.NewExp("kind = {:kind} AND enabled = {:enabled}",
			dbx.Params{"kind": "leaf", "enabled": true}),
	)
	if err != nil {
		log.Printf("[regions] failed to query enabled leaf regions: %v", err)
		return
	}

	for _, record := range leafRecords {
		buildRegionSafely(app, record)
	}
}

// buildRegionSafely wraps buildRegion with a recover so a single region's
// unexpected panic can never abort BuildAll's loop over the remaining
// regions.
func buildRegionSafely(app core.App, record *core.Record) {
	regionID := record.GetString("path")
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("[regions] recovered from panic building region %s: %v", regionID, rec)
		}
	}()
	buildRegion(app, record)
}

// buildRegion pre-builds one region's vector and, best-effort, DEM archive.
// Concurrent builds of the same region id are deduped via the package-level
// in-flight guard. regionID is the leaf's materialized `path`
// (A2 — the provably-unique key across the seeded catalog, unlike
// comaps_id).
func buildRegion(app core.App, record *core.Record) {
	regionID := record.GetString("path")
	name := record.GetString("name")

	inFlightMu.Lock()
	if wg, ok := inFlight[regionID]; ok {
		inFlightMu.Unlock()
		wg.Wait()
		return
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	inFlight[regionID] = wg
	inFlightMu.Unlock()

	defer func() {
		wg.Done()
		inFlightMu.Lock()
		delete(inFlight, regionID)
		inFlightMu.Unlock()
	}()

	if err := os.MkdirAll(filepath.Join(RegionCacheDir, regionID), 0755); err != nil {
		log.Printf("[regions] failed to create cache dir for region %s: %v", regionID, err)
		return
	}

	// Geometry MUST resolve before the archive record is created. Since bbox
	// lives in region_geometry rather than on the regions record,
	// ResolveGeometry is the only source of the four bounds, and
	// region_archives requires all of them — creating the record first fails
	// validation with "min_lat: cannot be blank" for any region that has no
	// archive row yet (i.e. every newly enabled region).
	//
	// Resolving up front also runs ahead of the !needsVector && !needsDem
	// early return below, which is what makes a deleted or corrupted
	// region_geometry row heal on the very next cron run even when nothing
	// else needed rebuilding. The accepted cost: a region whose
	// geometry row was destroyed costs one upstream request on the next run.
	// The preserved property: a well-formed cached row still costs zero
	// network (ResolveGeometry returns it with no fetch).
	//
	// Only a failed refetch reaches setError — this is the last
	// resort, after ResolveGeometry has already attempted the self-heal.
	geometry, bbox, err := ResolveGeometry(app, record)
	if err != nil {
		// A region that has never built has no archive row to carry the
		// error, and one cannot be created without the bbox that just failed
		// to resolve — so there is nothing to mark. Log instead. Either way
		// buildRegionSafely's per-region isolation keeps the run going.
		if archive, findErr := findRegionArchiveRecord(app, regionID); findErr == nil && archive != nil {
			_ = setError(app, archive, fmt.Errorf("resolve geometry for region %s: %w", regionID, err))
		} else {
			log.Printf("[regions] failed to resolve geometry for region %s (no archive record to mark): %v", regionID, err)
		}
		return
	}

	archive, err := findOrCreateRegionRecord(app, regionID, name, bbox)
	if err != nil {
		log.Printf("[regions] failed to find/create region_archives record for %s: %v", regionID, err)
		return
	}

	// A seed-catalog-refresh bbox change invalidates both the cached vector
	// staleness comparison and the DEM's "build once" guarantee —
	// detect it once up front and persist the new bbox before building.
	// Re-keyed to the seeded catalog's bbox per A4 (keep the check, do not
	// delete it).
	configChanged := bboxChanged(bbox,
		archive.GetFloat("min_lon"), archive.GetFloat("min_lat"),
		archive.GetFloat("max_lon"), archive.GetFloat("max_lat"),
	)
	if configChanged {
		archive.Set("min_lon", bbox[0])
		archive.Set("min_lat", bbox[1])
		archive.Set("max_lon", bbox[2])
		archive.Set("max_lat", bbox[3])
		archive.Set("name", name)
		if err := app.Save(archive); err != nil {
			log.Printf("[regions] failed to persist updated bbox for region %s: %v", regionID, err)
		}
	}

	date, url := ValidProtomapsDateAndURL()
	needsVector := configChanged || needsVectorRebuild(archive.GetString("vector_built_date"), date)

	// DEM builds once and never auto-rebuilds unless the catalog bbox
	// changed — Mapterhorn's source has no date-stamped URL to
	// compare against.
	needsDem := archive.GetString("dem_status") != "ready" || configChanged

	if !needsVector && !needsDem {
		return
	}

	polyPath, err := writePolygonTempFile(geometry)
	if err != nil {
		log.Printf("[regions] failed to write polygon temp file for region %s: %v", regionID, err)
		return
	}
	defer os.Remove(polyPath)

	if needsVector {
		buildVector(app, archive, regionID, polyPath, url, date)
	}
	if needsDem {
		buildDem(app, archive, regionID, polyPath)
	}
}

// findRegionArchiveRecord returns the existing region_archives record for
// regionID, or (nil, nil) when the region has never been built. Unlike
// findOrCreateRegionRecord it never creates one, which is what the geometry
// failure path needs: a create is impossible there precisely because the
// bbox it would require is the thing that failed to resolve.
func findRegionArchiveRecord(app core.App, regionID string) (*core.Record, error) {
	records, err := app.FindAllRecords("region_archives",
		dbx.NewExp("path = {:path}", dbx.Params{"path": regionID}),
	)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	return records[0], nil
}

// findOrCreateRegionRecord finds the region_archives record for regionID,
// creating one (status/dem_status "building") if none exists yet. Mirrors
// the per-cell generator's findOrCreateRecord shape. regionID is a
// regions.path value, stored in the region_archives.path field.
//
// bbox must be supplied by the caller, resolved from region_geometry via
// ResolveGeometry (it does not live on the regions record). It is NOT
// optional: region_archives declares all four bounds as required number
// fields, and PocketBase treats a required number as "non-zero", so a
// record saved with zeroed bounds is rejected outright with
// "min_lat: cannot be blank". An earlier revision of this function omitted
// bbox on the theory that the caller's configChanged check would backfill
// it — that never runs, because the Save below fails first, breaking every
// newly enabled region at its first build.
func findOrCreateRegionRecord(app core.App, regionID, name string, bbox [4]float64) (*core.Record, error) {
	collection, err := app.FindCollectionByNameOrId("region_archives")
	if err != nil {
		return nil, fmt.Errorf("region_archives collection not found: %w", err)
	}

	existing, err := findRegionArchiveRecord(app, regionID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	record := core.NewRecord(collection)
	record.Set("path", regionID)
	record.Set("name", name)
	record.Set("status", "building")
	record.Set("dem_status", "building")
	record.Set("min_lon", bbox[0])
	record.Set("min_lat", bbox[1])
	record.Set("max_lon", bbox[2])
	record.Set("max_lat", bbox[3])

	if err := app.Save(record); err != nil {
		return nil, err
	}

	return record, nil
}

// buildVector extracts region regionID's vector archive at
// regionVectorMaxZoom from url (a Protomaps daily-build URL) to a temp path,
// then atomically renames it into place on success (an already-ready
// region keeps serving its current file for the entire rebuild — status
// only ever flips to "building" here when there is no existing ready file to
// protect). Clips to the region's canonical polygon (polyPath)
// rather than its bounding box.
func buildVector(app core.App, archive *core.Record, regionID, polyPath, url, date string) {
	final := RegionArchivePath(regionID)
	tmp := final + ".building"

	if _, err := os.Stat(final); err != nil {
		archive.Set("status", "building")
		if err := app.Save(archive); err != nil {
			log.Printf("[regions] failed to mark region %s vector building: %v", regionID, err)
		}
	}

	log.Printf("[regions] building vector archive for region %s (region=%s)", regionID, polyPath)

	ctx, cancel := context.WithTimeout(context.Background(), regionExtractTimeout())
	defer cancel()

	cmd := exec.CommandContext(ctx, "pmtiles", "extract",
		url,
		tmp,
		"--region="+polyPath,
		fmt.Sprintf("--maxzoom=%d", regionVectorMaxZoom),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		os.Remove(tmp)
		_ = setError(app, archive, fmt.Errorf("pmtiles extract (vector) failed for region %s: %w", regionID, err))
		return
	}

	fi, err := os.Stat(tmp)
	if err != nil {
		_ = setError(app, archive, fmt.Errorf("could not stat vector output for region %s: %w", regionID, err))
		return
	}

	if err := os.Rename(tmp, final); err != nil {
		_ = setError(app, archive, fmt.Errorf("could not promote vector archive for region %s: %w", regionID, err))
		return
	}

	archive.Set("status", "ready")
	archive.Set("vector_size_bytes", fi.Size())
	archive.Set("vector_built_date", date)
	archive.Set("error_message", "")
	if err := app.Save(archive); err != nil {
		log.Printf("[regions] failed to save vector-ready record for region %s: %v", regionID, err)
		return
	}

	log.Printf("[regions] vector archive for region %s ready (%d bytes)", regionID, fi.Size())
}

// buildDem extracts region regionID's DEM archive at regionDemMaxZoom from
// mapterhornSource, same temp-path + atomic-rename discipline as
// buildVector. DEM is best-effort: it never returns an error and
// never blocks or affects the vector build's result — mirrors the per-cell
// generator's generateDemCell. Clips to the region's canonical polygon
// (polyPath) rather than its bounding box.
func buildDem(app core.App, archive *core.Record, regionID, polyPath string) {
	final := RegionDemPath(regionID)
	tmp := final + ".building"

	if _, err := os.Stat(final); err != nil {
		archive.Set("dem_status", "building")
		if err := app.Save(archive); err != nil {
			log.Printf("[regions] failed to mark region %s DEM building: %v", regionID, err)
		}
	}

	log.Printf("[regions] building DEM archive for region %s (region=%s)", regionID, polyPath)

	ctx, cancel := context.WithTimeout(context.Background(), regionExtractTimeout())
	defer cancel()

	cmd := exec.CommandContext(ctx, "pmtiles", "extract",
		mapterhornSource,
		tmp,
		"--region="+polyPath,
		fmt.Sprintf("--maxzoom=%d", regionDemMaxZoom),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		os.Remove(tmp)
		log.Printf("[regions] DEM extract failed for region %s: %v", regionID, err)
		archive.Set("dem_status", "error")
		archive.Set("dem_error_message", err.Error())
		if err := app.Save(archive); err != nil {
			log.Printf("[regions] failed to save DEM-error record for region %s: %v", regionID, err)
		}
		return
	}

	fi, err := os.Stat(tmp)
	if err != nil {
		log.Printf("[regions] could not stat DEM output for region %s: %v", regionID, err)
		archive.Set("dem_status", "error")
		archive.Set("dem_error_message", err.Error())
		_ = app.Save(archive)
		return
	}

	if err := os.Rename(tmp, final); err != nil {
		log.Printf("[regions] could not promote DEM archive for region %s: %v", regionID, err)
		archive.Set("dem_status", "error")
		archive.Set("dem_error_message", err.Error())
		_ = app.Save(archive)
		return
	}

	archive.Set("dem_status", "ready")
	archive.Set("dem_size_bytes", fi.Size())
	archive.Set("dem_error_message", "")
	if err := app.Save(archive); err != nil {
		log.Printf("[regions] failed to save DEM-ready record for region %s: %v", regionID, err)
		return
	}

	log.Printf("[regions] DEM archive for region %s ready (%d bytes)", regionID, fi.Size())
}

// setError marks record's vector build as failed and persists the error
// message. Mirrors the per-cell generator's setError.
func setError(app core.App, record *core.Record, err error) error {
	record.Set("status", "error")
	record.Set("error_message", err.Error())
	_ = app.Save(record)
	return err
}
