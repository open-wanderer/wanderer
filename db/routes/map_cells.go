package routes

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"pocketbase/services/tiles"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func MapCellsList(e *core.RequestEvent) error {
	minLon, minLat, maxLon, maxLat, err := parseBbox(e.Request.URL.Query().Get("bbox"))
	if err != nil {
		return e.BadRequestError("Invalid bbox. Expected: minLon,minLat,maxLon,maxLat", err)
	}

	cells := tiles.BboxToGridCells(minLon, minLat, maxLon, maxLat)

	type cellInfo struct {
		Key       string `json:"key"`
		Status    string `json:"status"`
		URL       string `json:"url"`
		SizeBytes int64  `json:"size_bytes,omitempty"`
	}

	result := make([]cellInfo, 0, len(cells))
	for _, cell := range cells {
		info := cellInfo{
			Key: cell.CacheKey(),
			URL: "/map/cells/" + cell.CacheKey(),
		}

		records, _ := e.App.FindAllRecords("tile_cells",
			dbx.NewExp("cell_key = {:key}", dbx.Params{"key": cell.CacheKey()}),
		)

		if len(records) > 0 {
			r := records[0]
			info.Status = r.GetString("status")
			if info.Status == "ready" {
				info.SizeBytes = int64(r.GetFloat("size_bytes"))
			}
		} else {
			info.Status = "new"
		}

		result = append(result, info)
	}

	return e.JSON(http.StatusOK, map[string]any{
		"cells": result,
	})
}

func parseBbox(raw string) (minLon, minLat, maxLon, maxLat float64, err error) {
	parts := strings.Split(raw, ",")
	if len(parts) != 4 {
		return 0, 0, 0, 0, fmt.Errorf("expected 4 comma-separated values")
	}
	vals := make([]float64, 4)
	for i, p := range parts {
		v, e := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if e != nil {
			return 0, 0, 0, 0, fmt.Errorf("invalid value %q: %w", p, e)
		}
		vals[i] = v
	}
	return vals[0], vals[1], vals[2], vals[3], nil
}

func parseCellKey(key string) (tiles.GridCell, error) {
	parts := strings.Split(key, "_")
	if len(parts) != 4 {
		return tiles.GridCell{}, fmt.Errorf("expected 4 underscore-separated values")
	}
	vals := make([]float64, 4)
	for i, p := range parts {
		v, err := strconv.ParseFloat(p, 64)
		if err != nil {
			return tiles.GridCell{}, fmt.Errorf("invalid value %q: %w", p, err)
		}
		vals[i] = v
	}
	return tiles.GridCell{
		MinLon: vals[0],
		MinLat: vals[1],
		MaxLon: vals[2],
		MaxLat: vals[3],
	}, nil
}
