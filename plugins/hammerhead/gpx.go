package main

import (
	"math"
	"time"

	sdkgpx "github.com/open-wanderer/wanderer/plugins/sdk/gpx"
	"github.com/open-wanderer/wanderer/plugins/sdk/polyline"
)

func activityGPX(activity *activity) ([]byte, error) {
	points := make([]sdkgpx.Point, 0, len(activity.RecordData.Timestamp))
	const zeroEps = 1e-4
	for i, timestamp := range activity.RecordData.Timestamp {
		if i >= len(activity.RecordData.Lat) || i >= len(activity.RecordData.Lng) {
			continue
		}
		lat := activity.RecordData.Lat[i]
		lng := activity.RecordData.Lng[i]
		if math.Abs(lat) < zeroEps && math.Abs(lng) < zeroEps {
			continue
		}
		elevation := 0.0
		if i < len(activity.RecordData.Elevation) {
			elevation = activity.RecordData.Elevation[i] / 1000.0
		}
		pointTime := time.Unix(int64(timestamp), 0).UTC()
		points = append(points, sdkgpx.Point{
			Lat:       lat,
			Lon:       lng,
			Elevation: &elevation,
			Time:      &pointTime,
		})
	}
	return sdkgpx.Track("wanderer Hammerhead plugin", activity.ActivityData.Name, points)
}

func tourGPX(tour *tour) ([]byte, error) {
	coords, err := polyline.Decode(tour.RoutePolyline, 1e5)
	if err != nil {
		return nil, err
	}
	polyline.NormalizeCoordinateScale(coords)
	elevations, _ := polyline.DecodeValues(tour.Elevation.Polyline, 100000)
	points := make([]sdkgpx.Point, 0, len(coords))
	swap := polyline.ShouldSwapCoordinates(coords)
	for i, coord := range coords {
		lat := coord[0]
		lon := coord[1]
		if swap {
			lat, lon = coord[1], coord[0]
		}
		var elevation *float64
		if len(elevations) == len(coords) {
			elevation = &elevations[i]
		} else if len(elevations) > 0 {
			elevation = &elevations[polyline.ProportionalIndex(i, len(coords), len(elevations))]
		}
		points = append(points, sdkgpx.Point{
			Lat:       lat,
			Lon:       lon,
			Elevation: elevation,
		})
	}
	return sdkgpx.Track("wanderer Hammerhead plugin", tour.Name, points)
}
