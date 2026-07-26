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
	// parallel-path constant per D-01 rather than a shared import.
	regionVectorMaxZoom = 14

	// regionDemMaxZoom mirrors the per-cell tile generator's DEM maxzoom
	// (12) — must stay in lockstep with that constant and the Dart
	// offline_style_rewriter.dart `_offlineDemMaxZoom` constant.
	regionDemMaxZoom = 12

	// mapterhornSource is Mapterhorn's downloadable global DEM pmtiles
	// archive. Mirrors the per-cell tile generator's own constant — a local,
	// parallel-path copy (D-01/D-02), not a shared import.
	mapterhornSource = "https://download.mapterhorn.com/planet.pmtiles"
)

var (
	// inFlightMu / inFlight dedupe concurrent builds of the same region,
	// keyed by region id (regions.path) — mirrors the per-cell generator's
	// in-flight guard (Pitfall 5), re-keyed for region-scale builds.
	inFlightMu sync.Mutex
	inFlight   = map[string]*sync.WaitGroup{}
)

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

// BuildAll is the cron entrypoint: it queries the seeded `regions` table for
// every enabled leaf (EXTRACT-02) and pre-builds each region's vector + DEM
// archives, one region fully completing before the next starts (bounds
// subprocess concurrency to 1). A query error or a single region's build
// failure is logged and never aborts the whole pass — this must never panic
// or block the cron.
func BuildAll(app core.App) {
	leafRecords, err := app.FindAllRecords("regions",
		dbx.NewExp("kind = {:kind} && enabled = {:enabled}",
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
// in-flight guard (Pitfall 5). regionID is the leaf's materialized `path`
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

	var bboxSlice []float64
	if err := record.UnmarshalJSONField("bbox", &bboxSlice); err != nil || len(bboxSlice) != 4 {
		log.Printf("[regions] region %s has an invalid or missing bbox (len=%d, err=%v)", regionID, len(bboxSlice), err)
		return
	}
	bbox := [4]float64{bboxSlice[0], bboxSlice[1], bboxSlice[2], bboxSlice[3]}

	var polygon map[string]any
	if err := record.UnmarshalJSONField("polygon", &polygon); err != nil || len(polygon) == 0 {
		log.Printf("[regions] region %s has an invalid or missing polygon (err=%v)", regionID, err)
		return
	}

	archive, err := findOrCreateRegionRecord(app, regionID, name, bbox)
	if err != nil {
		log.Printf("[regions] failed to find/create region_archives record for %s: %v", regionID, err)
		return
	}

	// A seed-catalog-refresh bbox change invalidates both the cached vector
	// staleness comparison and the DEM's "build once" guarantee (D-11) —
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
	// changed (D-11) — Mapterhorn's source has no date-stamped URL to
	// compare against.
	needsDem := archive.GetString("dem_status") != "ready" || configChanged

	if !needsVector && !needsDem {
		return
	}

	// Lazy: only write the polygon temp file when a build actually needs it,
	// avoiding an unnecessary temp write when nothing needs building.
	polyPath, err := writePolygonTempFile(polygon)
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

// findOrCreateRegionRecord finds the region_archives record for regionID,
// creating one (status/dem_status "building") if none exists yet. Mirrors
// the per-cell generator's findOrCreateRecord shape. region_id now stores
// regions.path (A2), not an admin-typed slug.
func findOrCreateRegionRecord(app core.App, regionID, name string, bbox [4]float64) (*core.Record, error) {
	collection, err := app.FindCollectionByNameOrId("region_archives")
	if err != nil {
		return nil, fmt.Errorf("region_archives collection not found: %w", err)
	}

	records, err := app.FindAllRecords("region_archives",
		dbx.NewExp("region_id = {:id}", dbx.Params{"id": regionID}),
	)
	if err != nil {
		return nil, err
	}

	if len(records) > 0 {
		return records[0], nil
	}

	record := core.NewRecord(collection)
	record.Set("region_id", regionID)
	record.Set("name", name)
	record.Set("min_lon", bbox[0])
	record.Set("min_lat", bbox[1])
	record.Set("max_lon", bbox[2])
	record.Set("max_lat", bbox[3])
	record.Set("status", "building")
	record.Set("dem_status", "building")

	if err := app.Save(record); err != nil {
		return nil, err
	}

	return record, nil
}

// buildVector extracts region regionID's vector archive at
// regionVectorMaxZoom from url (a Protomaps daily-build URL) to a temp path,
// then atomically renames it into place on success (D-08: an already-ready
// region keeps serving its current file for the entire rebuild — status
// only ever flips to "building" here when there is no existing ready file to
// protect). Clips to the region's canonical polygon (polyPath, EXTRACT-01)
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
// buildVector (D-08). DEM is best-effort: it never returns an error and
// never blocks or affects the vector build's result — mirrors the per-cell
// generator's generateDemCell. Clips to the region's canonical polygon
// (polyPath, EXTRACT-01) rather than its bounding box.
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
