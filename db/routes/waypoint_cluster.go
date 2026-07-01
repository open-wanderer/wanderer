package routes

import (
	"net/http"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/util"
)

const defaultWaypointMergeRadius = 50

type waypointMergeSettings struct {
	Enabled bool
	Radius  float64
}

type waypointClusterRequest struct {
	Category  string                    `json:"category"`
	Photos    []waypointClusterPhoto    `json:"photos"`
	Waypoints []waypointClusterWaypoint `json:"waypoints"`
}

type waypointClusterPhoto struct {
	ID  string  `json:"id"`
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type waypointClusterWaypoint struct {
	ID  string  `json:"id"`
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type waypointPhotoCluster struct {
	Waypoint string   `json:"waypoint,omitempty"`
	Photos   []string `json:"photos"`
	SumLat   float64  `json:"-"`
	SumLon   float64  `json:"-"`
	Count    int      `json:"-"`
	Lat      float64  `json:"lat"`
	Lon      float64  `json:"lon"`
}

type categorySettings struct {
	WaypointMergeEnabled *bool    `json:"wp_merge_enabled"`
	WaypointMergeRadius  *float64 `json:"wp_merge_radius"`
}

func WaypointCluster(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}

	var data waypointClusterRequest
	if err := e.BindBody(&data); err != nil {
		return apis.NewBadRequestError("Failed to read request data", err)
	}

	if data.Category != "" && len(data.Category) != 15 {
		return apis.NewBadRequestError("Invalid category", nil)
	}

	for _, photo := range data.Photos {
		if photo.ID == "" {
			return apis.NewBadRequestError("Invalid photo id", nil)
		}
		if photo.Lat < -90 || photo.Lat > 90 {
			return apis.NewBadRequestError("Invalid photo latitude", nil)
		}
		if photo.Lon < -180 || photo.Lon > 180 {
			return apis.NewBadRequestError("Invalid photo longitude", nil)
		}
	}

	for _, waypoint := range data.Waypoints {
		if waypoint.ID == "" {
			return apis.NewBadRequestError("Invalid waypoint id", nil)
		}
		if waypoint.Lat < -90 || waypoint.Lat > 90 {
			return apis.NewBadRequestError("Invalid waypoint latitude", nil)
		}
		if waypoint.Lon < -180 || waypoint.Lon > 180 {
			return apis.NewBadRequestError("Invalid waypoint longitude", nil)
		}
	}

	mergeSettings, err := getWaypointMergeSettings(e.App, data.Category)
	if err != nil {
		return err
	}

	return e.JSON(http.StatusOK, map[string]any{
		"mergeEnabled": mergeSettings.Enabled,
		"mergeRadius":  mergeSettings.Radius,
		"clusters":     clusterWaypointPhotos(data.Photos, data.Waypoints, mergeSettings),
	})
}

func getWaypointMergeSettings(app core.App, categoryId string) (waypointMergeSettings, error) {
	defaultSettings := waypointMergeSettings{
		Enabled: true,
		Radius:  defaultWaypointMergeRadius,
	}

	if categoryId == "" {
		return defaultSettings, nil
	}

	category, err := app.FindRecordById("categories", categoryId)
	if err != nil {
		return waypointMergeSettings{}, err
	}

	var settings categorySettings
	if err := category.UnmarshalJSONField("settings", &settings); err != nil {
		return defaultSettings, nil
	}

	if settings.WaypointMergeEnabled != nil {
		defaultSettings.Enabled = *settings.WaypointMergeEnabled
	}

	if settings.WaypointMergeRadius != nil && *settings.WaypointMergeRadius >= 0 {
		defaultSettings.Radius = *settings.WaypointMergeRadius
	}

	return defaultSettings, nil
}

func clusterWaypointPhotos(photos []waypointClusterPhoto, waypoints []waypointClusterWaypoint, mergeSettings waypointMergeSettings) []waypointPhotoCluster {
	clusters := []waypointPhotoCluster{}

	if mergeSettings.Enabled {
		for _, waypoint := range waypoints {
			clusters = append(clusters, newWaypointCluster(waypoint))
		}
	}

	for _, photo := range photos {
		if !mergeSettings.Enabled {
			clusters = append(clusters, newWaypointPhotoCluster(photo))
			continue
		}

		matchingClusterIndex := -1
		for i, cluster := range clusters {
			distanceToCenter := util.HaversineDistance(cluster.Lat, cluster.Lon, photo.Lat, photo.Lon)
			if distanceToCenter <= mergeSettings.Radius {
				matchingClusterIndex = i
				break
			}
		}

		if matchingClusterIndex >= 0 {
			addPhotoToWaypointCluster(&clusters[matchingClusterIndex], photo)
		} else {
			clusters = append(clusters, newWaypointPhotoCluster(photo))
		}
	}

	return clusters
}

func newWaypointPhotoCluster(photo waypointClusterPhoto) waypointPhotoCluster {
	return waypointPhotoCluster{
		Photos: []string{photo.ID},
		SumLat: photo.Lat,
		SumLon: photo.Lon,
		Count:  1,
		Lat:    photo.Lat,
		Lon:    photo.Lon,
	}
}

func newWaypointCluster(waypoint waypointClusterWaypoint) waypointPhotoCluster {
	return waypointPhotoCluster{
		Waypoint: waypoint.ID,
		Photos:   []string{},
		SumLat:   waypoint.Lat,
		SumLon:   waypoint.Lon,
		Count:    1,
		Lat:      waypoint.Lat,
		Lon:      waypoint.Lon,
	}
}

func addPhotoToWaypointCluster(cluster *waypointPhotoCluster, photo waypointClusterPhoto) {
	cluster.Photos = append(cluster.Photos, photo.ID)
	cluster.SumLat += photo.Lat
	cluster.SumLon += photo.Lon
	cluster.Count++
	cluster.Lat = cluster.SumLat / float64(cluster.Count)
	cluster.Lon = cluster.SumLon / float64(cluster.Count)
}
