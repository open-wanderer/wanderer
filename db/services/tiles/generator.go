package tiles

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	dbx "github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const (
	maxZoom = 14

	// demMaxZoom caps the offline DEM (hillshade) extraction lower than the
	// vector basemap's maxZoom — hillshading doesn't need z14 detail, and
	// MapLibre overzooms past it. MUST match the Dart
	// `_offlineDemMaxZoom` constant in offline_style_rewriter.dart (lockstep).
	demMaxZoom = 12

	// mapterhornSource is Mapterhorn's downloadable global DEM pmtiles
	// archive, used as the bbox-extraction source for offline hillshade
	// tiles. Static (unlike getValidProtomapsURL, no date-rotation).
	mapterhornSource = "https://download.mapterhorn.com/planet.pmtiles"

	cacheDir = "./pb_data/pmtiles_cache"

	// extractTimeout bounds each pmtiles extract subprocess so a hung or
	// unresponsive remote source (build.protomaps.com / download.mapterhorn.com)
	// can't wedge a cell's inFlight entry — and therefore every other request
	// for the same cell — indefinitely.
	extractTimeout = 5 * time.Minute
)

var (
	inFlightMu sync.Mutex
	inFlight   = map[string]*sync.WaitGroup{}
)

func CellPath(cell GridCell) string {
	return filepath.Join(cacheDir, cell.CacheKey()+".pmtiles")
}

// DemCellPath returns the path to the companion per-cell DEM (hillshade)
// pmtiles archive, extracted from mapterhornSource alongside the vector cell.
func DemCellPath(cell GridCell) string {
	return filepath.Join(cacheDir, cell.CacheKey()+"_dem.pmtiles")
}

func EnsureCell(app core.App, cell GridCell) error {
	inFlightMu.Lock()
	if wg, ok := inFlight[cell.CacheKey()]; ok {
		inFlightMu.Unlock()
		wg.Wait()
		return nil
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	inFlight[cell.CacheKey()] = wg
	inFlightMu.Unlock()

	defer func() {
		wg.Done()
		inFlightMu.Lock()
		delete(inFlight, cell.CacheKey())
		inFlightMu.Unlock()
	}()

	record, err := findOrCreateRecord(app, cell)
	if err != nil {
		return fmt.Errorf("failed to find/create tile_cells record: %w", err)
	}

	if record.GetString("status") == "ready" && record.GetString("dem_status") == "ready" {
		if _, err := os.Stat(CellPath(cell)); err == nil {
			if _, err := os.Stat(DemCellPath(cell)); err == nil {
				return nil
			}
		}
		log.Printf("[tiles] cell %s marked ready but file(s) missing, regenerating", cell.CacheKey())
	}

	return generateCell(app, record, cell)
}

func findOrCreateRecord(app core.App, cell GridCell) (*core.Record, error) {
	collection, err := app.FindCollectionByNameOrId("tile_cells")
	if err != nil {
		return nil, fmt.Errorf("tile_cells collection not found: %w", err)
	}

	records, err := app.FindAllRecords("tile_cells",
		dbx.NewExp("cell_key = {:key}", dbx.Params{"key": cell.CacheKey()}),
	)
	if err != nil {
		return nil, err
	}

	if len(records) > 0 {
		return records[0], nil
	}

	record := core.NewRecord(collection)
	record.Set("cell_key", cell.CacheKey())
	record.Set("status", "pending")
	record.Set("dem_status", "pending")
	record.Set("min_lon", cell.MinLon)
	record.Set("min_lat", cell.MinLat)
	record.Set("max_lon", cell.MaxLon)
	record.Set("max_lat", cell.MaxLat)
	record.Set("download_count", 0)

	if err := app.Save(record); err != nil {
		return nil, err
	}

	return record, nil
}

func getValidProtomapsURL() string {
	now := time.Now().UTC()

	todayURL := fmt.Sprintf("https://build.protomaps.com/%s.pmtiles", now.Format("20060102"))
	if urlExists(todayURL) {
		return todayURL
	}

	yesterdayURL := fmt.Sprintf("https://build.protomaps.com/%s.pmtiles", now.AddDate(0, 0, -1).Format("20060102"))
	return yesterdayURL
}

func urlExists(url string) bool {
	resp, err := http.Head(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func generateCell(app core.App, record *core.Record, cell GridCell) error {
	outputPath := CellPath(cell)

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return setError(app, record, fmt.Errorf("failed to create cache dir: %w", err))
	}

	bbox := fmt.Sprintf("%f,%f,%f,%f",
		cell.MinLon, cell.MinLat, cell.MaxLon, cell.MaxLat,
	)

	// Vector cell extraction — preserve an already-cached vector cell (only
	// re-extract when the file is actually missing), so a DEM-only refresh
	// (e.g. pre-existing vector-ready cells backfilling a DEM archive) does
	// not needlessly re-download the vector archive.
	if _, err := os.Stat(outputPath); err != nil {
		record.Set("status", "pending")
		record.Set("error_message", "")
		_ = app.Save(record)

		log.Printf("[tiles] generating cell %s (bbox: %s)", cell.CacheKey(), bbox)

		pmtilesSource := getValidProtomapsURL()

		ctx, cancel := context.WithTimeout(context.Background(), extractTimeout)
		defer cancel()

		cmd := exec.CommandContext(ctx, "pmtiles", "extract",
			pmtilesSource,
			outputPath,
			"--bbox="+bbox,
			fmt.Sprintf("--maxzoom=%d", maxZoom),
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			os.Remove(outputPath)
			return setError(app, record, fmt.Errorf("pmtiles extract failed: %w", err))
		}

		fi, err := os.Stat(outputPath)
		if err != nil {
			return setError(app, record, fmt.Errorf("could not stat output file: %w", err))
		}

		record.Set("status", "ready")
		record.Set("size_bytes", fi.Size())
		record.Set("error_message", "")
		if err := app.Save(record); err != nil {
			return err
		}

		log.Printf("[tiles] cell %s ready (%d bytes)", cell.CacheKey(), fi.Size())
	}

	// DEM cell extraction — independent lifecycle. Hillshade is cosmetic, so
	// a DEM failure is logged but MUST NOT return an error or block the
	// vector basemap from serving.
	generateDemCell(app, record, cell, bbox)

	return nil
}

func generateDemCell(app core.App, record *core.Record, cell GridCell, bbox string) {
	demOutputPath := DemCellPath(cell)

	record.Set("dem_status", "pending")
	record.Set("dem_error_message", "")
	_ = app.Save(record)

	log.Printf("[tiles] generating DEM cell %s (bbox: %s)", cell.CacheKey(), bbox)

	ctx, cancel := context.WithTimeout(context.Background(), extractTimeout)
	defer cancel()

	demCmd := exec.CommandContext(ctx, "pmtiles", "extract",
		mapterhornSource,
		demOutputPath,
		"--bbox="+bbox,
		fmt.Sprintf("--maxzoom=%d", demMaxZoom),
	)
	demCmd.Stdout = os.Stdout
	demCmd.Stderr = os.Stderr

	if err := demCmd.Run(); err != nil {
		os.Remove(demOutputPath)
		log.Printf("[tiles] DEM extract failed for cell %s: %v", cell.CacheKey(), err)
		record.Set("dem_status", "error")
		record.Set("dem_error_message", err.Error())
		if err := app.Save(record); err != nil {
			log.Printf("[tiles] failed to save DEM-error record for cell %s: %v", cell.CacheKey(), err)
		}
		return
	}

	fi, err := os.Stat(demOutputPath)
	if err != nil {
		log.Printf("[tiles] could not stat DEM output for cell %s: %v", cell.CacheKey(), err)
		record.Set("dem_status", "error")
		record.Set("dem_error_message", err.Error())
		_ = app.Save(record)
		return
	}

	record.Set("dem_status", "ready")
	record.Set("dem_size_bytes", fi.Size())
	record.Set("dem_error_message", "")
	if err := app.Save(record); err != nil {
		log.Printf("[tiles] failed to save DEM-ready record for cell %s: %v", cell.CacheKey(), err)
		return
	}

	log.Printf("[tiles] DEM cell %s ready (%d bytes)", cell.CacheKey(), fi.Size())
}

func setError(app core.App, record *core.Record, err error) error {
	record.Set("status", "error")
	record.Set("error_message", err.Error())
	_ = app.Save(record)
	return err
}

func IncrementDownloadCount(app core.App, cell GridCell) {
	records, err := app.FindAllRecords("tile_cells",
		dbx.NewExp("cell_key = {:key}", dbx.Params{"key": cell.CacheKey()}),
	)
	if err != nil || len(records) == 0 {
		return
	}
	r := records[0]
	r.Set("download_count", r.GetInt("download_count")+1)
	_ = app.Save(r)
}
