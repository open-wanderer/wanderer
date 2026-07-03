package main

import (
	"encoding/base64"
	"fmt"
	"time"
)

func activityImport(activity *intervalsActivity, gpxData []byte) (trailImport, error) {
	if len(gpxData) == 0 {
		return trailImport{}, fmt.Errorf("activity GPX is empty")
	}

	privacy := "public"
	startedAt := activity.StartDate
	if startedAt == "" {
		startedAt = activity.StartDateLocal
	}

	if t, err := time.Parse("2006-01-02T15:04:05", startedAt); err == nil {
		startedAt = t.Format(time.RFC3339)
	} else if t, err := time.Parse("2006-01-02T15:04:05Z", startedAt); err == nil {
		startedAt = t.Format(time.RFC3339)
	}

	return trailImport{
		Source: trailImportSource{
			Provider:   "intervals",
			ExternalID: activity.ID,
		},
		Kind:         "completed",
		Name:         activity.Name,
		Description:  activity.Description,
		StartedAt:    startedAt,
		ActivityType: mapActivityType(activity.Type),
		Privacy:      &privacy,
		Track: track{
			Format:        "gpx",
			ContentBase64: base64.StdEncoding.EncodeToString(gpxData),
		},
		Metadata: map[string]any{
			"distance":         activity.Distance,
			"elevationGain":    activity.TotalElevationGain,
			"duration":         activity.ElapsedTime,
			"providerCategory": activity.Type,
		},
	}, nil
}

func mapActivityType(intervalsType string) string {
	switch intervalsType {
	case "Ride", "VirtualRide", "EBikeRide", "MountainBikeRide", "GravelRide":
		return "biking"
	case "Run", "TrailRun", "VirtualRun":
		return "running"
	case "Hike":
		return "hiking"
	case "Walk":
		return "walking"
	case "Swim":
		return "swimming"
	default:
		return "other"
	}
}
