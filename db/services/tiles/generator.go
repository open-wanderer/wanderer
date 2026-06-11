package tiles

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	dbx "github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const (
	pmtilesSource = "https://build.protomaps.com/20260610.pmtiles"

	maxZoom = 14

	cacheDir = "./pb_data/pmtiles_cache"
)

var (
	inFlightMu sync.Mutex
	inFlight   = map[string]*sync.WaitGroup{}
)

func CellPath(cell GridCell) string {
	return filepath.Join(cacheDir, cell.CacheKey()+".pmtiles")
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

	if record.GetString("status") == "ready" {
		if _, err := os.Stat(CellPath(cell)); err == nil {
			return nil
		}
		log.Printf("[tiles] cell %s marked ready but file missing, regenerating", cell.CacheKey())
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

func generateCell(app core.App, record *core.Record, cell GridCell) error {
	outputPath := CellPath(cell)

	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return setError(app, record, fmt.Errorf("failed to create cache dir: %w", err))
	}

	record.Set("status", "pending")
	record.Set("error_message", "")
	_ = app.Save(record)

	bbox := fmt.Sprintf("%f,%f,%f,%f",
		cell.MinLon, cell.MinLat, cell.MaxLon, cell.MaxLat,
	)

	log.Printf("[tiles] generating cell %s (bbox: %s)", cell.CacheKey(), bbox)

	cmd := exec.Command("pmtiles", "extract",
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
	return nil
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
