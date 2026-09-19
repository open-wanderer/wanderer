package routes

import (
	"encoding/json"
	"fmt"
)

// Bounds select photos in the visible map area instead of applying the trail or
// waypoint radius. West > East represents a viewport crossing the antimeridian.
type pluginAssetSearchBounds struct {
	West  float64 `json:"west"`
	South float64 `json:"south"`
	East  float64 `json:"east"`
	North float64 `json:"north"`
}

func (b *pluginAssetSearchBounds) UnmarshalJSON(data []byte) error {
	var values struct {
		West  *float64 `json:"west"`
		South *float64 `json:"south"`
		East  *float64 `json:"east"`
		North *float64 `json:"north"`
	}
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	if values.West == nil || values.South == nil || values.East == nil || values.North == nil {
		return fmt.Errorf("bounds requires west, south, east, and north")
	}
	*b = pluginAssetSearchBounds{
		West: *values.West, South: *values.South,
		East: *values.East, North: *values.North,
	}
	return b.validate()
}

func (b *pluginAssetSearchBounds) validate() error {
	if b == nil {
		return nil
	}
	if !isFiniteCoordinate(b.South, b.West) || !isFiniteCoordinate(b.North, b.East) || b.South > b.North {
		return fmt.Errorf("invalid bounds: coordinates must be finite and in range, with south <= north")
	}
	return nil
}

func (b *pluginAssetSearchBounds) contains(lat, lon float64) bool {
	if b == nil {
		return true
	}
	if !isFiniteCoordinate(lat, lon) || lat < b.South || lat > b.North {
		return false
	}
	if b.West > b.East {
		return lon >= b.West || lon <= b.East
	}
	return lon >= b.West && lon <= b.East
}
