// Package regions: this file writes a leaf region's GeoJSON polygon field
// out to a unique temp .geojson file suitable for `pmtiles extract
// --region=<file>` — the CLI requires a real local file path, not inline
// JSON or stdin.
package regions

import (
	"encoding/json"
	"fmt"
	"os"
)

// writePolygonTempFile marshals polygon (a GeoJSON Polygon or MultiPolygon
// object, the shape produced by core.Record.UnmarshalJSONField("polygon",
// ...)) to a uniquely-named temp file and returns its path. The caller owns
// cleanup — always `defer os.Remove(path)` immediately after a successful
// call.
//
// A nil or empty polygon is rejected with an error and no file is created —
// a leaf region must always carry geometry. The polygon's JSON content is
// never interpolated into a shell string; this helper only ever produces a
// file path, matching the safe --bbox= argument construction it replaces.
func writePolygonTempFile(polygon map[string]any) (string, error) {
	if len(polygon) == 0 {
		return "", fmt.Errorf("region polygon is empty")
	}

	data, err := json.Marshal(polygon)
	if err != nil {
		return "", fmt.Errorf("marshal polygon: %w", err)
	}

	f, err := os.CreateTemp("", "region-*.geojson")
	if err != nil {
		return "", fmt.Errorf("create temp polygon file: %w", err)
	}

	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", fmt.Errorf("write temp polygon file: %w", err)
	}

	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", fmt.Errorf("close temp polygon file: %w", err)
	}

	return f.Name(), nil
}
