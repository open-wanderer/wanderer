package tiles

import (
	"fmt"
	"math"
)

const GridSize = 0.5

type GridCell struct {
	MinLon float64
	MinLat float64
	MaxLon float64
	MaxLat float64
}

func (c GridCell) CacheKey() string {
	return fmt.Sprintf("%.2f_%.2f_%.2f_%.2f", c.MinLon, c.MinLat, c.MaxLon, c.MaxLat)
}

func BboxToGridCells(minLon, minLat, maxLon, maxLat float64) []GridCell {
	startLon := math.Floor(minLon/GridSize) * GridSize
	startLat := math.Floor(minLat/GridSize) * GridSize

	var cells []GridCell

	for lon := startLon; lon < maxLon; lon += GridSize {
		for lat := startLat; lat < maxLat; lat += GridSize {
			cells = append(cells, GridCell{
				MinLon: math.Round(lon*100) / 100,
				MinLat: math.Round(lat*100) / 100,
				MaxLon: math.Round((lon+GridSize)*100) / 100,
				MaxLat: math.Round((lat+GridSize)*100) / 100,
			})
		}
	}

	return cells
}
