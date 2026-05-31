package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"math"
	"strconv"
	"time"
)

type gpxPoint struct {
	Lat       float64
	Lon       float64
	Elevation *float64
	Time      *time.Time
}

func activityGPX(activity *activity) ([]byte, error) {
	points := make([]gpxPoint, 0, len(activity.RecordData.Timestamp))
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
		points = append(points, gpxPoint{
			Lat:       lat,
			Lon:       lng,
			Elevation: &elevation,
			Time:      &pointTime,
		})
	}
	return gpxBytes(activity.ActivityData.Name, points)
}

func tourGPX(tour *tour) ([]byte, error) {
	coords, err := decodePolyline(tour.RoutePolyline)
	if err != nil {
		return nil, err
	}
	elevations, _ := decodeElevations(tour.Elevation.Polyline, 100000)
	points := make([]gpxPoint, 0, len(coords))
	for i, coord := range coords {
		var elevation *float64
		if len(elevations) == len(coords) {
			elevation = &elevations[i]
		}
		points = append(points, gpxPoint{
			Lat:       coord[0],
			Lon:       coord[1],
			Elevation: elevation,
		})
	}
	return gpxBytes(tour.Name, points)
}

func gpxBytes(name string, points []gpxPoint) ([]byte, error) {
	if len(points) == 0 {
		return nil, fmt.Errorf("track has no points")
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	buf.WriteString(`<gpx version="1.1" creator="wanderer Hammerhead plugin" xmlns="http://www.topografix.com/GPX/1/1">`)
	buf.WriteString("<trk>")
	buf.WriteString("<name>")
	xml.EscapeText(&buf, []byte(name))
	buf.WriteString("</name>")
	buf.WriteString("<trkseg>")
	for _, point := range points {
		buf.WriteString(`<trkpt lat="`)
		buf.WriteString(strconv.FormatFloat(point.Lat, 'f', 7, 64))
		buf.WriteString(`" lon="`)
		buf.WriteString(strconv.FormatFloat(point.Lon, 'f', 7, 64))
		buf.WriteString(`">`)
		if point.Elevation != nil {
			buf.WriteString("<ele>")
			buf.WriteString(strconv.FormatFloat(*point.Elevation, 'f', 2, 64))
			buf.WriteString("</ele>")
		}
		if point.Time != nil {
			buf.WriteString("<time>")
			buf.WriteString(point.Time.Format(time.RFC3339))
			buf.WriteString("</time>")
		}
		buf.WriteString("</trkpt>")
	}
	buf.WriteString("</trkseg>")
	buf.WriteString("</trk>")
	buf.WriteString("</gpx>")
	return buf.Bytes(), nil
}

func decodePolyline(s string) ([][2]float64, error) {
	var coords [][2]float64
	index := 0
	lat := 0
	lng := 0
	for index < len(s) {
		dlat, next, err := decodePolylineValue(s, index)
		if err != nil {
			return nil, err
		}
		index = next
		dlng, next, err := decodePolylineValue(s, index)
		if err != nil {
			return nil, err
		}
		index = next
		lat += dlat
		lng += dlng
		coords = append(coords, [2]float64{float64(lat) / 1e5, float64(lng) / 1e5})
	}
	return coords, nil
}

func decodeElevations(s string, precision float64) ([]float64, error) {
	var values []float64
	index := 0
	value := 0
	for index < len(s) {
		delta, next, err := decodePolylineValue(s, index)
		if err != nil {
			return nil, err
		}
		index = next
		value += delta
		values = append(values, float64(value)/precision)
	}
	return values, nil
}

func decodePolylineValue(s string, index int) (int, int, error) {
	result := 0
	shift := uint(0)
	for {
		if index >= len(s) {
			return 0, index, fmt.Errorf("invalid polyline encoding")
		}
		b := int(s[index]) - 63
		index++
		result |= (b & 0x1F) << shift
		shift += 5
		if b < 0x20 {
			break
		}
	}
	return (result >> 1) ^ (-(result & 1)), index, nil
}
