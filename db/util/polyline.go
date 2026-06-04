package util

import (
	"bytes"
	"fmt"
	"io"

	"github.com/pocketbase/pocketbase/core"
	"github.com/tkrajina/gpxgo/gpx"
	"github.com/twpayne/go-polyline"
)

const PolylineMaxLength = 5 * 1024 * 1024

type TrailGeometry struct {
	Polyline            string
	MinLat              float64
	MaxLat              float64
	MinLon              float64
	MaxLon              float64
	BoundingBoxDiagonal float64
}

func ComputeTrailGeometry(app core.App, r *core.Record) (*TrailGeometry, error) {
	geometry := &TrailGeometry{
		MinLat: r.GetFloat("lat"),
		MaxLat: r.GetFloat("lat"),
		MinLon: r.GetFloat("lon"),
		MaxLon: r.GetFloat("lon"),
	}

	gpxPath := r.GetString("gpx")
	if len(gpxPath) == 0 {
		return geometry, nil
	}

	fsys, err := app.NewFilesystem()
	if err != nil {
		return nil, fmt.Errorf("open filesystem: %w", err)
	}
	defer fsys.Close()

	gpxFilePath := r.BaseFilesPath() + "/" + gpxPath
	gpxFile, err := fsys.GetReader(gpxFilePath)
	if err != nil {
		return nil, fmt.Errorf("open gpx file %q: %w", gpxFilePath, err)
	}
	defer gpxFile.Close()

	content := new(bytes.Buffer)
	if _, err = io.Copy(content, gpxFile); err != nil {
		return nil, fmt.Errorf("read gpx file %q: %w", gpxFilePath, err)
	}

	gpxData, err := gpx.Parse(content)
	if err != nil {
		return nil, fmt.Errorf("parse gpx file %q: %w", gpxFilePath, err)
	}

	minLat, maxLat, minLon, maxLon := 90.0, -90.0, 180.0, -180.0
	hasPoints := false

	addPoint := func(lat, lon float64) {
		if lat < minLat {
			minLat = lat
		}
		if lat > maxLat {
			maxLat = lat
		}
		if lon < minLon {
			minLon = lon
		}
		if lon > maxLon {
			maxLon = lon
		}
		hasPoints = true
	}

	for _, trk := range gpxData.Tracks {
		for _, seg := range trk.Segments {
			for _, pt := range seg.Points {
				addPoint(pt.Latitude, pt.Longitude)
			}
		}
	}

	for _, rte := range gpxData.Routes {
		for _, pt := range rte.Points {
			addPoint(pt.Latitude, pt.Longitude)
		}
	}

	gpxData.SimplifyTracks(50)
	coordinates := make([][]float64, 0)
	for _, trk := range gpxData.Tracks {
		for _, seg := range trk.Segments {
			for _, pt := range seg.Points {
				coordinates = append(coordinates, []float64{pt.Latitude, pt.Longitude})
			}
		}
	}
	geometry.Polyline = string(polyline.EncodeCoords(coordinates))

	if hasPoints {
		geometry.MinLat = minLat
		geometry.MaxLat = maxLat
		geometry.MinLon = minLon
		geometry.MaxLon = maxLon
		geometry.BoundingBoxDiagonal = HaversineDistance(minLat, minLon, maxLat, maxLon)
	}

	return geometry, nil
}

func ComputePolyline(app core.App, r *core.Record) (string, error) {
	geometry, err := ComputeTrailGeometry(app, r)
	if err != nil {
		return "", err
	}
	return geometry.Polyline, nil
}

func SavePolyline(app core.App, r *core.Record) error {
	geometry, err := ComputeTrailGeometry(app, r)
	if err != nil {
		return err
	}
	// Encoded polylines are ASCII-only, so byte length matches character length.
	if len(geometry.Polyline) > PolylineMaxLength {
		return fmt.Errorf("polyline exceeds maximum length of %d characters", PolylineMaxLength)
	}
	r.Set("polyline", geometry.Polyline)
	r.Set("min_lat", geometry.MinLat)
	r.Set("max_lat", geometry.MaxLat)
	r.Set("min_lon", geometry.MinLon)
	r.Set("max_lon", geometry.MaxLon)
	r.Set("bounding_box_diagonal", geometry.BoundingBoxDiagonal)
	if err := app.UnsafeWithoutHooks().Save(r); err != nil {
		return fmt.Errorf("save trail geometry: %w", err)
	}
	return nil
}
