package main

import (
	"fmt"
	"math"
	"strconv"
)

// GeoViewport describes the visible latitude/longitude bounds of a map. West
// may be greater than East to represent a viewport crossing the antimeridian.
type GeoViewport struct {
	North float64
	West  float64
	South float64
	East  float64
}

// geoBoundingBoxFilter builds the Meilisearch filter for a viewport. The
// documented _geoBoundingBox form requires west < east, so an antimeridian
// viewport is conservatively split into the two conventional longitude ranges
// and joined with OR.
func geoBoundingBoxFilter(viewport GeoViewport) (string, error) {
	if err := validateGeoViewport(viewport); err != nil {
		return "", err
	}
	if viewport.West < viewport.East {
		return geoBoundingBoxExpression(viewport.North, viewport.West, viewport.South, viewport.East), nil
	}

	west := geoBoundingBoxExpression(viewport.North, viewport.West, viewport.South, 180)
	east := geoBoundingBoxExpression(viewport.North, -180, viewport.South, viewport.East)
	return fmt.Sprintf("(%s OR %s)", west, east), nil
}

func validateGeoViewport(viewport GeoViewport) error {
	values := []struct {
		name    string
		value   float64
		minimum float64
		maximum float64
	}{
		{name: "north", value: viewport.North, minimum: -90, maximum: 90},
		{name: "west", value: viewport.West, minimum: -180, maximum: 180},
		{name: "south", value: viewport.South, minimum: -90, maximum: 90},
		{name: "east", value: viewport.East, minimum: -180, maximum: 180},
	}
	for _, value := range values {
		if math.IsNaN(value.value) || math.IsInf(value.value, 0) || value.value < value.minimum || value.value > value.maximum {
			return fmt.Errorf("viewport %s must be finite and between %g and %g, got %g", value.name, value.minimum, value.maximum, value.value)
		}
	}
	if viewport.North <= viewport.South {
		return fmt.Errorf("viewport north must be greater than south, got %g <= %g", viewport.North, viewport.South)
	}
	if viewport.West == viewport.East {
		return fmt.Errorf("viewport west and east must differ, got %g", viewport.West)
	}
	return nil
}

func geoBoundingBoxExpression(north, west, south, east float64) string {
	// Meilisearch expects the top-right [north, east] corner first and the
	// bottom-left [south, west] corner second.
	return fmt.Sprintf(
		"_geoBoundingBox([%s, %s], [%s, %s])",
		formatGeoFilterNumber(north),
		formatGeoFilterNumber(east),
		formatGeoFilterNumber(south),
		formatGeoFilterNumber(west),
	)
}

func formatGeoFilterNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', 7, 64)
}
