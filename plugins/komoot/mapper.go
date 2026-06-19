package main

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	sdkgpx "github.com/open-wanderer/wanderer/plugins/sdk/gpx"
)

func tourImport(tour *detailedTour, routeImages []imageItem) (trailImport, error) {
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
		Description:  tour.Description,
		StartedAt:    tour.Date,
		ActivityType: activityType(tour.Sport),
		Privacy:      &privacy,
		Track: track{
			Format:        "gpx",
			ContentBase64: base64.StdEncoding.EncodeToString(gpxData),
		},
		Waypoints: waypoints(tour),
		Photos:    photos(tour, routeImages),
		Metadata: map[string]any{
			"distance":         tour.Distance,
			"elevationGain":    tour.ElevationUp,
			"elevationLoss":    tour.ElevationDown,
			"duration":         tour.Duration,
			"providerCategory": tour.Sport,
			"sourceSport":      tour.Sport,
			"difficulty":       tour.Difficulty.Grade,
		},
	}, nil
}

func tourGPX(tour *detailedTour) ([]byte, error) {
	items := tour.Embedded.Coordinates.Items
	points := make([]sdkgpx.Point, 0, len(items))
	startedAt, _ := time.Parse(time.RFC3339, tour.Date)
	for _, item := range items {
		elevation := item.Alt
		point := sdkgpx.Point{
			Lat:       item.Lat,
			Lon:       item.Lng,
			Elevation: &elevation,
		}
		if !startedAt.IsZero() {
			pointTime := startedAt.Add(time.Duration(item.T) * time.Millisecond).UTC()
			point.Time = &pointTime
		}
		points = append(points, point)
	}
	return sdkgpx.Track("wanderer Komoot plugin", tour.Name, points)
}

func waypoints(tour *detailedTour) []waypoint {
	result := make([]waypoint, 0, len(tour.Embedded.WayPoints.Embedded.Items)+len(tour.Embedded.Timeline.Embedded.Items))
	seen := map[string]bool{}
	result = appendWaypoints(result, seen, tour.Embedded.WayPoints.Embedded.Items)
	result = appendWaypoints(result, seen, tour.Embedded.Timeline.Embedded.Items)
	return result
}

func appendWaypoints(result []waypoint, seen map[string]bool, items []timelineItem) []waypoint {
	for _, item := range items {
		ref := item.Embedded.Reference
		point, ok := waypointPoint(ref)
		if ref.Name == "" || !ok {
			continue
		}
		key := ref.ID.String()
		if key == "" {
			key = fmt.Sprintf("%s:%0.7f:%0.7f", ref.Name, point.Lat, point.Lng)
		}
		if seen[key] {
			continue
		}
		seen[key] = true

		description := ""
		if len(ref.Embedded.Tips.Embedded.Items) > 0 {
			description = ref.Embedded.Tips.Embedded.Items[0].Text
		}
		ele := point.Alt
		result = append(result, waypoint{
			ExternalID:  ref.ID.String(),
			Name:        ref.Name,
			Description: description,
			Lat:         point.Lat,
			Lon:         point.Lng,
			Ele:         &ele,
			Icon:        "circle",
			Photos:      waypointPhotos(item),
		})
	}
	return result
}

func waypointPoint(ref waypointReference) (point, bool) {
	if ref.StartPoint.Lat != 0 || ref.StartPoint.Lng != 0 {
		return ref.StartPoint, true
	}
	if ref.Location.Lat != 0 || ref.Location.Lng != 0 {
		return ref.Location, true
	}
	return point{}, false
}

func photos(tour *detailedTour, routeImages []imageItem) []photo {
	images := routeImages
	if len(images) == 0 {
		images = tour.Embedded.CoverImages.Embedded.Items
	}
	if len(images) == 0 && tour.MapImage.Src != "" {
		images = []imageItem{{Src: tour.MapImage.Src, Type: "image/jpeg"}}
	}
	return photosFromImages(images, "komoot-photo.jpg")
}

func waypointPhotos(item timelineItem) []photo {
	ref := item.Embedded.Reference
	images := ref.Embedded.Images.Embedded.Items
	if ref.Embedded.FrontImage.Src != "" {
		images = append([]imageItem{ref.Embedded.FrontImage}, images...)
	}
	return photosFromImages(images, "komoot-waypoint-photo.jpg")
}

func photosFromImages(images []imageItem, fallbackFilename string) []photo {
	result := make([]photo, 0, len(images))
	seen := map[string]bool{}
	for _, image := range images {
		source := expandImageURL(image.Src)
		if source == "" || strings.HasSuffix(strings.ToLower(source), ".gif") {
			continue
		}
		key := image.ID.String()
		if key == "" {
			key = source
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, photo{
			ExternalID:  image.ID.String(),
			Filename:    filenameForImage(image.ID, fallbackFilename),
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

func filenameForImage(id flexibleID, fallback string) string {
	if id.String() == "" {
		return fallback
	}
	return fmt.Sprintf("komoot-%s.jpg", id.String())
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
