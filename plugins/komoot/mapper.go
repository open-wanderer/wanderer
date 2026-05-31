//go:build tinygo

package main

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func tourImport(tour *detailedTour) (trailImport, error) {
	gpxData, err := tourGPX(tour)
	if err != nil {
		return trailImport{}, err
	}

	privacy := privacyFromStatus(tour.Status)
	return trailImport{
		Source: trailImportSource{
			Provider:   "komoot",
			ExternalID: strconv.FormatInt(tour.ID, 10),
		},
		Kind:         kindFromType(tour.Type),
		Name:         tour.Name,
		StartedAt:    tour.Date,
		ActivityType: activityType(tour.Sport),
		Privacy:      &privacy,
		Track: track{
			Format:        "gpx",
			ContentBase64: base64.StdEncoding.EncodeToString(gpxData),
		},
		Waypoints: waypoints(tour),
		Photos:    photos(tour),
		Metadata: map[string]any{
			"sourceSport": tour.Sport,
			"difficulty":  tour.Difficulty.Grade,
		},
	}, nil
}

func tourGPX(tour *detailedTour) ([]byte, error) {
	points := tour.Embedded.Coordinates.Items
	if len(points) == 0 {
		return nil, fmt.Errorf("track has no points")
	}

	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	builder.WriteString(`<gpx version="1.1" creator="wanderer Komoot plugin" xmlns="http://www.topografix.com/GPX/1/1">` + "\n")
	builder.WriteString("<trk><name>")
	writeEscaped(&builder, tour.Name)
	builder.WriteString("</name><trkseg>\n")
	startedAt, _ := time.Parse(time.RFC3339, tour.Date)
	for _, point := range points {
		builder.WriteString(fmt.Sprintf(`<trkpt lat="%.8f" lon="%.8f"><ele>%.2f</ele>`, point.Lat, point.Lng, point.Alt))
		if !startedAt.IsZero() {
			builder.WriteString("<time>")
			builder.WriteString(startedAt.Add(time.Duration(point.T) * time.Millisecond).UTC().Format(time.RFC3339))
			builder.WriteString("</time>")
		}
		builder.WriteString("</trkpt>\n")
	}
	builder.WriteString("</trkseg></trk></gpx>\n")
	return []byte(builder.String()), nil
}

func writeEscaped(builder *strings.Builder, value string) {
	var escaped strings.Builder
	_ = xml.EscapeText(&escaped, []byte(value))
	builder.WriteString(escaped.String())
}

func waypoints(tour *detailedTour) []waypoint {
	result := make([]waypoint, 0, len(tour.Embedded.Timeline.Embedded.Items))
	for _, item := range tour.Embedded.Timeline.Embedded.Items {
		ref := item.Embedded.Reference
		if ref.Name == "" || ref.StartPoint.Lat == 0 && ref.StartPoint.Lng == 0 {
			continue
		}

		description := ""
		if len(ref.Embedded.Tips.Embedded.Items) > 0 {
			description = ref.Embedded.Tips.Embedded.Items[0].Text
		}
		ele := ref.StartPoint.Alt
		result = append(result, waypoint{
			ExternalID:  strconv.FormatInt(ref.ID, 10),
			Name:        ref.Name,
			Description: description,
			Lat:         ref.StartPoint.Lat,
			Lon:         ref.StartPoint.Lng,
			Ele:         &ele,
			Icon:        "circle",
		})
	}
	return result
}

func photos(tour *detailedTour) []photo {
	images := tour.Embedded.CoverImages.Embedded.Items
	if len(images) == 0 && tour.MapImage.Src != "" {
		images = []imageItem{{Src: tour.MapImage.Src, Type: "image/jpeg"}}
	}

	result := make([]photo, 0, len(images))
	for _, image := range images {
		source := expandImageURL(image.Src)
		if source == "" || strings.HasSuffix(strings.ToLower(source), ".gif") {
			continue
		}
		result = append(result, photo{
			ExternalID:  strconv.FormatInt(image.ID, 10),
			Filename:    filenameForImage(image.ID),
			ContentType: contentType(image.Type),
			Lat:         optionalCoordinate(image.Location.Lat),
			Lon:         optionalCoordinate(image.Location.Lng),
			Source: mediaSource{
				Type: "url",
				URL:  source,
			},
		})
	}
	return result
}

func expandImageURL(source string) string {
	source = strings.ReplaceAll(source, "{crop}", "false")
	source = strings.ReplaceAll(source, "{width}", "")
	source = strings.ReplaceAll(source, "{height}", "")
	return source
}

func filenameForImage(id int64) string {
	if id <= 0 {
		return "komoot-photo.jpg"
	}
	return fmt.Sprintf("komoot-%d.jpg", id)
}

func contentType(value string) string {
	if strings.HasPrefix(value, "image/") {
		return value
	}
	return "image/jpeg"
}

func optionalCoordinate(value float64) *float64 {
	if value == 0 {
		return nil
	}
	return &value
}

func kindFromType(value string) string {
	if value == "tour_recorded" {
		return "completed"
	}
	return "planned"
}

func privacyFromStatus(value string) string {
	if value == "public" {
		return "public"
	}
	return "private"
}

func activityType(sport string) string {
	switch sport {
	case "hike", "mountaineering":
		return "hiking"
	case "jogging":
		return "running"
	case "touringbicycle", "mtb", "racebike", "mtb_easy", "mtb_advanced":
		return "biking"
	default:
		return sport
	}
}
