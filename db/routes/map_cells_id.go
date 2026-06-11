package routes

import (
	"net/http"
	"os"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/services/tiles"
)

func MapCellsGet(e *core.RequestEvent) error {
	cell, err := parseCellKey(e.Request.PathValue("cellKey"))
	if err != nil {
		return e.BadRequestError("invalid cell key", err)
	}

	records, _ := e.App.FindAllRecords("tile_cells",
		dbx.NewExp("cell_key = {:key}", dbx.Params{"key": cell.CacheKey()}),
	)

	if len(records) > 0 && records[0].GetString("status") == "ready" {
		if _, err := os.Stat(tiles.CellPath(cell)); err == nil {
			return e.JSON(http.StatusOK, map[string]any{
				"status":       "ready",
				"download_url": "/map/cells/" + cell.CacheKey() + "/download",
			})
		}
	}

	if len(records) > 0 && records[0].GetString("status") == "pending" {
		return e.JSON(http.StatusAccepted, map[string]any{
			"status":     "pending",
			"status_url": "/map/cells/" + cell.CacheKey() + "/status",
		})
	}

	go func() {
		if err := tiles.EnsureCell(e.App, cell); err != nil {
			// Error is persisted to the DB record by EnsureCell itself.
		}
	}()

	return e.JSON(http.StatusAccepted, map[string]any{
		"status":     "pending",
		"status_url": "/map/cells/" + cell.CacheKey() + "/status",
	})
}

func MapCellsStatus(e *core.RequestEvent) error {
	cell, err := parseCellKey(e.Request.PathValue("cellKey"))
	if err != nil {
		return e.BadRequestError("Invalid cell key", err)
	}

	records, _ := e.App.FindAllRecords("tile_cells",
		dbx.NewExp("cell_key = {:key}", dbx.Params{"key": cell.CacheKey()}),
	)

	if len(records) == 0 {
		return e.JSON(http.StatusOK, map[string]any{"status": "pending"})
	}

	r := records[0]
	status := r.GetString("status")

	resp := map[string]any{"key": cell.CacheKey(), "status": status}
	if status == "ready" {
		resp["size_bytes"] = int64(r.GetFloat("size_bytes"))
		resp["download_url"] = "/map/cells/" + cell.CacheKey() + "/download"
	}
	if status == "error" {
		resp["error"] = r.GetString("error_message")
	}

	return e.JSON(http.StatusOK, resp)
}

func MapCellsDownload(e *core.RequestEvent) error {
	cell, err := parseCellKey(e.Request.PathValue("cellKey"))
	if err != nil {
		return e.BadRequestError("Invalid cell key", err)
	}

	if _, err := os.Stat(tiles.CellPath(cell)); err != nil {
		return e.NotFoundError("Cell not ready yet", nil)
	}

	go tiles.IncrementDownloadCount(e.App, cell)
	return e.FileFS(os.DirFS("./pb_data/pmtiles_cache"), cell.CacheKey()+".pmtiles")
}
