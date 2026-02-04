package util

import (
	"bytes"
	"io"
	"math"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/tkrajina/gpxgo/gpx"
)

type TrailBoundingBox struct {
	MinLat float64
	MinLon float64
	MaxLat float64
	MaxLon float64
}

func UpdateTrailBoundingBox(app core.App, r *core.Record) (bool, error) {
	bbox, err := computeTrailBoundingBox(app, r)
	if err != nil || bbox == nil {
		return false, err
	}
	*bbox = normalizeTrailBoundingBox(*bbox)

	if r.GetFloat("min_lat") == bbox.MinLat &&
		r.GetFloat("min_lon") == bbox.MinLon &&
		r.GetFloat("max_lat") == bbox.MaxLat &&
		r.GetFloat("max_lon") == bbox.MaxLon {
		return false, nil
	}

	_, err = app.DB().NewQuery(
		`UPDATE trails
		SET min_lat = {:min_lat},
			min_lon = {:min_lon},
			max_lat = {:max_lat},
			max_lon = {:max_lon}
		WHERE id = {:id}`,
	).Bind(dbx.Params{
		"id":      r.Id,
		"min_lat": bbox.MinLat,
		"min_lon": bbox.MinLon,
		"max_lat": bbox.MaxLat,
		"max_lon": bbox.MaxLon,
	}).Execute()
	if err != nil {
		return false, err
	}

	// Keep in-memory record in sync for indexing in the same request.
	r.Set("min_lat", bbox.MinLat)
	r.Set("min_lon", bbox.MinLon)
	r.Set("max_lat", bbox.MaxLat)
	r.Set("max_lon", bbox.MaxLon)

	return true, nil
}

func computeTrailBoundingBox(app core.App, r *core.Record) (*TrailBoundingBox, error) {
	gpxPath := r.GetString("gpx")
	if len(gpxPath) == 0 {
		lat := r.GetFloat("lat")
		lon := r.GetFloat("lon")
		if lat == 0 && lon == 0 {
			return nil, nil
		}
		return &TrailBoundingBox{
			MinLat: lat,
			MinLon: lon,
			MaxLat: lat,
			MaxLon: lon,
		}, nil
	}

	gpxKey := r.BaseFilesPath() + "/" + gpxPath
	fsys, err := app.NewFilesystem()
	if err != nil {
		return nil, err
	}
	defer fsys.Close()

	gpxFile, err := fsys.GetReader(gpxKey)
	if err != nil {
		return nil, err
	}
	defer gpxFile.Close()

	content := new(bytes.Buffer)
	if _, err := io.Copy(content, gpxFile); err != nil {
		return nil, err
	}

	gpxData, err := gpx.Parse(content)
	if err != nil {
		return nil, err
	}

	minLat := math.Inf(1)
	minLon := math.Inf(1)
	maxLat := math.Inf(-1)
	maxLon := math.Inf(-1)

	updateBounds := func(lat, lon float64) {
		minLat = math.Min(minLat, lat)
		minLon = math.Min(minLon, lon)
		maxLat = math.Max(maxLat, lat)
		maxLon = math.Max(maxLon, lon)
	}

	for _, trk := range gpxData.Tracks {
		for _, seg := range trk.Segments {
			for _, pt := range seg.Points {
				updateBounds(pt.Latitude, pt.Longitude)
			}
		}
	}

	for _, rte := range gpxData.Routes {
		for _, pt := range rte.Points {
			updateBounds(pt.Latitude, pt.Longitude)
		}
	}

	for _, wp := range gpxData.Waypoints {
		updateBounds(wp.Latitude, wp.Longitude)
	}

	if math.IsInf(minLat, 1) || math.IsInf(minLon, 1) {
		return nil, nil
	}

	return &TrailBoundingBox{
		MinLat: minLat,
		MinLon: minLon,
		MaxLat: maxLat,
		MaxLon: maxLon,
	}, nil
}
