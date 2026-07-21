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
	// keyed by region id — mirrors the per-cell generator's in-flight guard
	// (Pitfall 5), re-keyed for region-scale builds.
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

// BuildAll is the cron entrypoint: it loads the configured region catalog
// and pre-builds each region's vector + DEM archives, one region fully
// completing before the next starts (bounds subprocess concurrency to 1).
// A config-load error or a single region's build failure is logged and
// never aborts the whole pass — this must never panic or block the cron.
func BuildAll(app core.App) {
	regionsList, err := LoadRegionCatalog()
	if err != nil {
		log.Printf("[regions] failed to load region catalog: %v", err)
		return
	}

	for _, r := range regionsList {
		buildRegionSafely(app, r)
	}
}

// buildRegionSafely wraps buildRegion with a recover so a single region's
// unexpected panic can never abort BuildAll's loop over the remaining
// regions.
func buildRegionSafely(app core.App, r Region) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("[regions] recovered from panic building region %s: %v", r.ID, rec)
		}
	}()
	buildRegion(app, r)
}

// buildRegion pre-builds one region's vector and, best-effort, DEM archive.
// Concurrent builds of the same region id are deduped via the package-level
// in-flight guard (Pitfall 5).
func buildRegion(app core.App, r Region) {
	inFlightMu.Lock()
	if wg, ok := inFlight[r.ID]; ok {
		inFlightMu.Unlock()
		wg.Wait()
		return
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	inFlight[r.ID] = wg
	inFlightMu.Unlock()

	defer func() {
		wg.Done()
		inFlightMu.Lock()
		delete(inFlight, r.ID)
		inFlightMu.Unlock()
	}()

	if err := os.MkdirAll(filepath.Join(RegionCacheDir, r.ID), 0755); err != nil {
		log.Printf("[regions] failed to create cache dir for region %s: %v", r.ID, err)
		return
	}

	record, err := findOrCreateRegionRecord(app, r)
	if err != nil {
		log.Printf("[regions] failed to find/create region_archives record for %s: %v", r.ID, err)
		return
	}

	// A config-bbox edit invalidates both the cached vector staleness
	// comparison and the DEM's "build once" guarantee (D-11) — detect it
	// once up front and persist the new bbox before building.
	configChanged := bboxChanged(r.Bbox,
		record.GetFloat("min_lon"), record.GetFloat("min_lat"),
		record.GetFloat("max_lon"), record.GetFloat("max_lat"),
	)
	if configChanged {
		record.Set("min_lon", r.Bbox[0])
		record.Set("min_lat", r.Bbox[1])
		record.Set("max_lon", r.Bbox[2])
		record.Set("max_lat", r.Bbox[3])
		record.Set("name", r.Name)
		if err := app.Save(record); err != nil {
			log.Printf("[regions] failed to persist updated bbox for region %s: %v", r.ID, err)
		}
	}

	date, url := ValidProtomapsDateAndURL()
	if configChanged || needsVectorRebuild(record.GetString("vector_built_date"), date) {
		buildVector(app, record, r, url, date)
	}

	// DEM builds once and never auto-rebuilds unless the config bbox changed
	// (D-11) — Mapterhorn's source has no date-stamped URL to compare
	// against.
	if record.GetString("dem_status") != "ready" || configChanged {
		buildDem(app, record, r)
	}
}

// findOrCreateRegionRecord finds the region_archives record for r.ID,
// creating one (status/dem_status "building") if none exists yet. Mirrors
// the per-cell generator's findOrCreateRecord shape.
func findOrCreateRegionRecord(app core.App, r Region) (*core.Record, error) {
	collection, err := app.FindCollectionByNameOrId("region_archives")
	if err != nil {
		return nil, fmt.Errorf("region_archives collection not found: %w", err)
	}

	records, err := app.FindAllRecords("region_archives",
		dbx.NewExp("region_id = {:id}", dbx.Params{"id": r.ID}),
	)
	if err != nil {
		return nil, err
	}

	if len(records) > 0 {
		return records[0], nil
	}

	record := core.NewRecord(collection)
	record.Set("region_id", r.ID)
	record.Set("name", r.Name)
	record.Set("min_lon", r.Bbox[0])
	record.Set("min_lat", r.Bbox[1])
	record.Set("max_lon", r.Bbox[2])
	record.Set("max_lat", r.Bbox[3])
	record.Set("status", "building")
	record.Set("dem_status", "building")

	if err := app.Save(record); err != nil {
		return nil, err
	}

	return record, nil
}

// buildVector extracts region r's vector archive at regionVectorMaxZoom from
// url (a Protomaps daily-build URL) to a temp path, then atomically renames
// it into place on success (D-08: an already-ready region keeps serving its
// current file for the entire rebuild — status only ever flips to "building"
// here when there is no existing ready file to protect).
func buildVector(app core.App, record *core.Record, r Region, url, date string) {
	final := RegionArchivePath(r.ID)
	tmp := final + ".building"

	// bbox is built from the region's float64 coordinates only — never from
	// raw config text — before it is interpolated into the exec argument
	// (T-21.5-02).
	bbox := fmt.Sprintf("%f,%f,%f,%f", r.Bbox[0], r.Bbox[1], r.Bbox[2], r.Bbox[3])

	if _, err := os.Stat(final); err != nil {
		record.Set("status", "building")
		if err := app.Save(record); err != nil {
			log.Printf("[regions] failed to mark region %s vector building: %v", r.ID, err)
		}
	}

	log.Printf("[regions] building vector archive for region %s (bbox: %s)", r.ID, bbox)

	ctx, cancel := context.WithTimeout(context.Background(), regionExtractTimeout())
	defer cancel()

	cmd := exec.CommandContext(ctx, "pmtiles", "extract",
		url,
		tmp,
		"--bbox="+bbox,
		fmt.Sprintf("--maxzoom=%d", regionVectorMaxZoom),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		os.Remove(tmp)
		_ = setError(app, record, fmt.Errorf("pmtiles extract (vector) failed for region %s: %w", r.ID, err))
		return
	}

	fi, err := os.Stat(tmp)
	if err != nil {
		_ = setError(app, record, fmt.Errorf("could not stat vector output for region %s: %w", r.ID, err))
		return
	}

	if err := os.Rename(tmp, final); err != nil {
		_ = setError(app, record, fmt.Errorf("could not promote vector archive for region %s: %w", r.ID, err))
		return
	}

	record.Set("status", "ready")
	record.Set("vector_size_bytes", fi.Size())
	record.Set("vector_built_date", date)
	record.Set("error_message", "")
	if err := app.Save(record); err != nil {
		log.Printf("[regions] failed to save vector-ready record for region %s: %v", r.ID, err)
		return
	}

	log.Printf("[regions] vector archive for region %s ready (%d bytes)", r.ID, fi.Size())
}

// buildDem extracts region r's DEM archive at regionDemMaxZoom from
// mapterhornSource, same temp-path + atomic-rename discipline as buildVector
// (D-08). DEM is best-effort: it never returns an error and never blocks or
// affects the vector build's result — mirrors the per-cell generator's
// generateDemCell.
func buildDem(app core.App, record *core.Record, r Region) {
	final := RegionDemPath(r.ID)
	tmp := final + ".building"

	bbox := fmt.Sprintf("%f,%f,%f,%f", r.Bbox[0], r.Bbox[1], r.Bbox[2], r.Bbox[3])

	if _, err := os.Stat(final); err != nil {
		record.Set("dem_status", "building")
		if err := app.Save(record); err != nil {
			log.Printf("[regions] failed to mark region %s DEM building: %v", r.ID, err)
		}
	}

	log.Printf("[regions] building DEM archive for region %s (bbox: %s)", r.ID, bbox)

	ctx, cancel := context.WithTimeout(context.Background(), regionExtractTimeout())
	defer cancel()

	cmd := exec.CommandContext(ctx, "pmtiles", "extract",
		mapterhornSource,
		tmp,
		"--bbox="+bbox,
		fmt.Sprintf("--maxzoom=%d", regionDemMaxZoom),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		os.Remove(tmp)
		log.Printf("[regions] DEM extract failed for region %s: %v", r.ID, err)
		record.Set("dem_status", "error")
		record.Set("dem_error_message", err.Error())
		if err := app.Save(record); err != nil {
			log.Printf("[regions] failed to save DEM-error record for region %s: %v", r.ID, err)
		}
		return
	}

	fi, err := os.Stat(tmp)
	if err != nil {
		log.Printf("[regions] could not stat DEM output for region %s: %v", r.ID, err)
		record.Set("dem_status", "error")
		record.Set("dem_error_message", err.Error())
		_ = app.Save(record)
		return
	}

	if err := os.Rename(tmp, final); err != nil {
		log.Printf("[regions] could not promote DEM archive for region %s: %v", r.ID, err)
		record.Set("dem_status", "error")
		record.Set("dem_error_message", err.Error())
		_ = app.Save(record)
		return
	}

	record.Set("dem_status", "ready")
	record.Set("dem_size_bytes", fi.Size())
	record.Set("dem_error_message", "")
	if err := app.Save(record); err != nil {
		log.Printf("[regions] failed to save DEM-ready record for region %s: %v", r.ID, err)
		return
	}

	log.Printf("[regions] DEM archive for region %s ready (%d bytes)", r.ID, fi.Size())
}

// setError marks record's vector build as failed and persists the error
// message. Mirrors the per-cell generator's setError.
func setError(app core.App, record *core.Record, err error) error {
	record.Set("status", "error")
	record.Set("error_message", err.Error())
	_ = app.Save(record)
	return err
}
