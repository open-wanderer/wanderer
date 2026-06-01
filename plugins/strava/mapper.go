//go:build tinygo

package main

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	sdkgpx "github.com/open-wanderer/wanderer/plugins/sdk/gpx"
)

func routeImport(route route, gpxData []byte) (trailImport, error) {
	if len(gpxData) == 0 {
		return trailImport{}, fmt.Errorf("route GPX is empty")
	}
	privacy := privacyFromPrivate(route.Private)
	startedAt := time.Unix(route.Timestamp, 0).UTC().Format(time.RFC3339)
	return trailImport{
		Source: trailImportSource{
			Provider:   "strava",
			ExternalID: route.IDStr,
		},
		Kind:         "planned",
		Name:         route.Name,
		Description:  route.Description,
		StartedAt:    startedAt,
		ActivityType: activityTypeForRoute(route.Type),
		Privacy:      &privacy,
		Track: track{
			Format:        "gpx",
			ContentBase64: base64.StdEncoding.EncodeToString(gpxData),
		},
		Waypoints: routeWaypoints(route),
		Metadata: map[string]any{
			"distance":            route.Distance,
			"elevationGain":       route.ElevationGain,
			"duration":            route.EstimatedMovingTime,
			"providerCategory":    routeCategory(route.Type),
			"estimatedMovingTime": route.EstimatedMovingTime,
		},
	}, nil
}

func activityImport(activity *detailedActivity, streams *activityStreamResponse, photos []activityPhoto) (trailImport, error) {
	if len(activity.StartLatlng) < 2 {
		return trailImport{}, fmt.Errorf("activity has no start coordinate")
	}
	gpxData, err := activityGPX(activity, streams)
	if err != nil {
		return trailImport{}, err
	}
	privacy := privacyFromPrivate(activity.Private)
	return trailImport{
		Source: trailImportSource{
			Provider:   "strava",
			ExternalID: strconv.FormatInt(activity.ID, 10),
		},
		Kind:         "completed",
		Name:         activity.Name,
		Description:  activity.Description,
		StartedAt:    activity.StartDate,
		ActivityType: activityType(activity),
		Privacy:      &privacy,
		Track: track{
			Format:        "gpx",
			ContentBase64: base64.StdEncoding.EncodeToString(gpxData),
		},
		Photos: activityPhotos(activity, photos),
		Metadata: map[string]any{
			"distance":         activity.Distance,
			"elevationGain":    activity.TotalElevationGain,
			"duration":         activity.ElapsedTime,
			"providerCategory": providerActivityType(activity),
		},
	}, nil
}

func routeWaypoints(route route) []waypoint {
	points := make([]waypoint, 0, len(route.Waypoints))
	for i, wp := range route.Waypoints {
		if len(wp.Latlng) < 2 {
			continue
		}
		name := wp.Title
		if name == "" {
			name = strconv.Itoa(i)
		}
		points = append(points, waypoint{
			Name:        name,
			Description: wp.Description,
			Lat:         wp.Latlng[0],
			Lon:         wp.Latlng[1],
			Icon:        "circle",
		})
	}
	return points
}

func activityPhotos(activity *detailedActivity, apiPhotos []activityPhoto) []photo {
	photos := make([]photo, 0, len(apiPhotos))
	seen := make(map[string]bool, len(apiPhotos))
	for _, apiPhoto := range apiPhotos {
		url := apiPhoto.Urls.Num600
		if url == "" {
			url = apiPhoto.Urls.Num100
		}
		if url == "" {
			continue
		}
		externalID := apiPhoto.UniqueID
		if externalID == "" {
			externalID = url
		}
		if seen[externalID] {
			continue
		}
		seen[externalID] = true
		photos = append(photos, photo{
			ExternalID: externalID,
			Filename:   fmt.Sprintf("strava-%s.jpg", safePhotoID(externalID)),
			Source: mediaSource{
				Type: "url",
				URL:  url,
			},
		})
	}
	if len(photos) > 0 {
		return photos
	}
	if activity.Photos.Primary.Urls.Num600 == "" {
		return nil
	}
	externalID := strconv.FormatInt(activity.Photos.Primary.ID, 10)
	return []photo{{
		ExternalID: externalID,
		Filename:   fmt.Sprintf("strava-%s.jpg", externalID),
		Source: mediaSource{
			Type: "url",
			URL:  activity.Photos.Primary.Urls.Num600,
		},
	}}
}

func safePhotoID(value string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "?", "-", "&", "-", "=", "-")
	return replacer.Replace(value)
}

func activityGPX(activity *detailedActivity, streams *activityStreamResponse) ([]byte, error) {
	if streams == nil || len(streams.LatLng.Data) == 0 {
		return nil, fmt.Errorf("activity has no latlng stream")
	}
	startedAt, _ := time.Parse(time.RFC3339, activity.StartDate)

	points := make([]sdkgpx.Point, 0, len(streams.LatLng.Data))
	for i, latlng := range streams.LatLng.Data {
		if len(latlng) < 2 || i >= len(streams.Time.Data) {
			continue
		}
		elevation := 0.0
		if i < len(streams.Altitude.Data) {
			elevation = streams.Altitude.Data[i]
		}
		point := sdkgpx.Point{
			Lat:       latlng[0],
			Lon:       latlng[1],
			Elevation: &elevation,
		}
		if !startedAt.IsZero() {
			pointTime := startedAt.Add(time.Duration(streams.Time.Data[i]) * time.Second).UTC()
			point.Time = &pointTime
		}
		points = append(points, point)
	}
	return sdkgpx.Track("wanderer Strava plugin", activity.Name, points)
}

func privacyFromPrivate(private bool) string {
	if private {
		return "private"
	}
	return "public"
}

func activityTypeForRoute(routeType int) string {
	switch routeType {
	case 1:
		return "biking"
	case 2:
		return "walking"
	default:
		return ""
	}
}

func routeCategory(routeType int) string {
	return fmt.Sprintf("route:%d", routeType)
}

func providerActivityType(activity *detailedActivity) string {
	if activity.SportType != "" {
		return activity.SportType
	}
	return activity.Type
}

func activityType(activity *detailedActivity) string {
	value := providerActivityType(activity)
	switch value {
	case "AlpineSki", "BackcountrySki", "IceSkate", "NordicSki", "RollerSki", "Snowboard":
		return "skiing"
	case "Canoeing", "Kayaking", "Kitesurf", "Rowing", "Sail", "StandUpPaddling", "Surfing", "Windsurf":
		return "canoeing"
	case "Hike", "Snowshoe":
		return "hiking"
	case "Run", "VirtualRun", "Walk", "Golf", "Skateboard", "Wheelchair":
		return "walking"
	case "Ride", "EBikeRide", "Handcycle", "InlineSkate", "Velomobile", "VirtualRide":
		return "biking"
	case "RockClimbing":
		return "climbing"
	default:
		return value
	}
}
