package util

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/pocketbase/pocketbase/core"
	"github.com/tkrajina/gpxgo/gpx"
	"github.com/twpayne/go-polyline"
)

const PolylineMaxLength = 5 * 1024 * 1024

// ErrTrailGPXChanged means the geometry was computed for a GPX file that is no
// longer the trail's current file. Callers must not store that geometry.
var ErrTrailGPXChanged = errors.New("trail GPX changed while computing geometry")

type TrailGeometry struct {
	Polyline            string
	MinLat              float64
	MaxLat              float64
	MinLon              float64
	MaxLon              float64
	BoundingBoxDiagonal float64
}

func ComputeTrailGeometry(app core.App, r *core.Record) (*TrailGeometry, error) {
	gpxPath := r.GetString("gpx")
	if len(gpxPath) == 0 {
		return NewTrailGeometry(nil, r.GetFloat("lat"), r.GetFloat("lon"))
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
	return NewTrailGeometry(gpxData, r.GetFloat("lat"), r.GetFloat("lon"))
}

// NewTrailGeometry computes the persisted geometry fields from an already
// parsed GPX. This lets imports store GPX, metrics and geometry in the same
// record write instead of relying on a post-commit follow-up.
func NewTrailGeometry(gpxData *gpx.GPX, fallbackLat float64, fallbackLon float64) (*TrailGeometry, error) {
	geometry := &TrailGeometry{
		MinLat: fallbackLat,
		MaxLat: fallbackLat,
		MinLon: fallbackLon,
		MaxLon: fallbackLon,
	}
	if gpxData == nil {
		return geometry, nil
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
	// Encoded polylines are ASCII-only, so byte length matches character length.
	if len(geometry.Polyline) > PolylineMaxLength {
		return nil, fmt.Errorf("polyline exceeds maximum length of %d characters", PolylineMaxLength)
	}

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
	if r == nil || r.Id == "" {
		return fmt.Errorf("persisted trail is required")
	}

	expectedGPX := r.GetString("gpx")
	// The event record may have gone stale while an after-update hook was
	// running. Compute from a fresh copy and refuse work for an older GPX.
	fresh, err := app.FindRecordById(r.Collection().Id, r.Id)
	if err != nil {
		return fmt.Errorf("reload trail for geometry: %w", err)
	}
	if fresh.GetString("gpx") != expectedGPX {
		return fmt.Errorf("%w: expected %q, found %q", ErrTrailGPXChanged, expectedGPX, fresh.GetString("gpx"))
	}
	geometry, err := ComputeTrailGeometry(app, fresh)
	if err != nil {
		return err
	}

	err = app.RunInTransaction(func(txApp core.App) error {
		stored, err := txApp.FindRecordById(r.Collection().Id, r.Id)
		if err != nil {
			return fmt.Errorf("reload trail before saving geometry: %w", err)
		}
		if stored.GetString("gpx") != expectedGPX {
			return fmt.Errorf("%w: expected %q, found %q", ErrTrailGPXChanged, expectedGPX, stored.GetString("gpx"))
		}
		ApplyTrailGeometry(stored, geometry)
		stored.IgnoreUnchangedFields(true)
		if err := txApp.UnsafeWithoutHooks().Save(stored); err != nil {
			return fmt.Errorf("save trail geometry: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	// Follow-up hooks and callers should see the geometry computed for this
	// event, without copying unrelated fields from the freshly loaded record.
	ApplyTrailGeometry(r, geometry)
	return nil
}

// ApplyTrailGeometry adds only geometry-derived fields to a record. Callers
// decide whether those fields are part of a primary write or a guarded
// follow-up.
func ApplyTrailGeometry(r *core.Record, geometry *TrailGeometry) {
	if r == nil || geometry == nil {
		return
	}
	r.Set("polyline", geometry.Polyline)
	r.Set("min_lat", geometry.MinLat)
	r.Set("max_lat", geometry.MaxLat)
	r.Set("min_lon", geometry.MinLon)
	r.Set("max_lon", geometry.MaxLon)
	r.Set("bounding_box_diagonal", geometry.BoundingBoxDiagonal)
}
