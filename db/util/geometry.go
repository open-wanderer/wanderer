package util

import (
	"bytes"
	"fmt"
	"io"

	"github.com/pocketbase/pocketbase/core"
	"github.com/tkrajina/gpxgo/gpx"
)

type TrailGeometryMetrics struct {
	MeanDistanceMeters  float64
	MaxDistanceMeters   float64
	StartDistanceMeters float64
	EndDistanceMeters   float64
}

type trailPoint struct {
	Lat float64
	Lon float64
}

func TrailCoordinates(app core.App, r *core.Record) ([][2]float64, error) {
	gpxPath := r.GetString("gpx")
	if gpxPath == "" {
		return nil, nil
	}

	fsys, err := app.NewFilesystem()
	if err != nil {
		return nil, err
	}
	defer fsys.Close()

	reader, err := fsys.GetReader(r.BaseFilesPath() + "/" + gpxPath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	content := new(bytes.Buffer)
	if _, err := io.Copy(content, reader); err != nil {
		return nil, err
	}

	gpxData, err := gpx.Parse(content)
	if err != nil {
		return nil, err
	}

	points := make([][2]float64, 0)
	for _, trk := range gpxData.Tracks {
		for _, seg := range trk.Segments {
			for _, pt := range seg.Points {
				points = append(points, [2]float64{pt.Latitude, pt.Longitude})
			}
		}
	}

	return points, nil
}

func TrailGeometrySimilarity(app core.App, a *core.Record, b *core.Record) (*TrailGeometryMetrics, error) {
	aCoords, err := TrailCoordinates(app, a)
	if err != nil {
		return nil, fmt.Errorf("load source geometry: %w", err)
	}
	bCoords, err := TrailCoordinates(app, b)
	if err != nil {
		return nil, fmt.Errorf("load target geometry: %w", err)
	}

	return CompareTrailCoordinates(aCoords, bCoords)
}

func CompareTrailCoordinates(aCoords [][2]float64, bCoords [][2]float64) (*TrailGeometryMetrics, error) {
	if len(aCoords) < 2 || len(bCoords) < 2 {
		return nil, fmt.Errorf("missing geometry")
	}

	a := resampleTrailCoordinates(aCoords, 64)
	b := resampleTrailCoordinates(bCoords, 64)
	if len(a) < 2 || len(b) < 2 {
		return nil, fmt.Errorf("missing geometry")
	}

	return compareSampledTrails(a, b), nil
}

func compareSampledTrails(a []trailPoint, b []trailPoint) *TrailGeometryMetrics {
	count := min(len(a), len(b))
	if count == 0 {
		return &TrailGeometryMetrics{}
	}

	sum := 0.0
	maxDistance := 0.0
	for i := 0; i < count; i++ {
		distance := HaversineDistanceMeters(a[i].Lat, a[i].Lon, b[i].Lat, b[i].Lon)
		sum += distance
		if distance > maxDistance {
			maxDistance = distance
		}
	}

	return &TrailGeometryMetrics{
		MeanDistanceMeters:  sum / float64(count),
		MaxDistanceMeters:   maxDistance,
		StartDistanceMeters: HaversineDistanceMeters(a[0].Lat, a[0].Lon, b[0].Lat, b[0].Lon),
		EndDistanceMeters:   HaversineDistanceMeters(a[count-1].Lat, a[count-1].Lon, b[count-1].Lat, b[count-1].Lon),
	}
}

func resampleTrailCoordinates(coords [][2]float64, targetPoints int) []trailPoint {
	points := make([]trailPoint, 0, len(coords))
	for _, coord := range coords {
		points = append(points, trailPoint{Lat: coord[0], Lon: coord[1]})
	}

	if len(points) <= 2 || targetPoints <= 2 {
		return points
	}

	cumulative := make([]float64, len(points))
	total := 0.0
	for i := 1; i < len(points); i++ {
		total += HaversineDistanceMeters(points[i-1].Lat, points[i-1].Lon, points[i].Lat, points[i].Lon)
		cumulative[i] = total
	}

	if total == 0 {
		return []trailPoint{points[0], points[len(points)-1]}
	}

	resampled := make([]trailPoint, 0, targetPoints)
	for i := 0; i < targetPoints; i++ {
		targetDistance := (float64(i) / float64(targetPoints-1)) * total
		resampled = append(resampled, interpolateTrailPoint(points, cumulative, targetDistance))
	}

	return resampled
}

func interpolateTrailPoint(points []trailPoint, cumulative []float64, targetDistance float64) trailPoint {
	if targetDistance <= 0 {
		return points[0]
	}
	lastIndex := len(points) - 1
	if targetDistance >= cumulative[lastIndex] {
		return points[lastIndex]
	}

	for i := 1; i < len(points); i++ {
		if cumulative[i] < targetDistance {
			continue
		}

		prevDistance := cumulative[i-1]
		nextDistance := cumulative[i]
		if nextDistance == prevDistance {
			return points[i]
		}

		ratio := (targetDistance - prevDistance) / (nextDistance - prevDistance)
		return trailPoint{
			Lat: points[i-1].Lat + (points[i].Lat-points[i-1].Lat)*ratio,
			Lon: points[i-1].Lon + (points[i].Lon-points[i-1].Lon)*ratio,
		}
	}

	return points[lastIndex]
}
