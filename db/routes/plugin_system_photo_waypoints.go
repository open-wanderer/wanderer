package routes

import (
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"pocketbase/plugins/importer"
	"pocketbase/pluginsystem"
	"pocketbase/util"
)

// maxPhotoWaypointsPerTrail caps the waypoints made from photos, since with
// merging switched off every photo would become a waypoint of its own.
const maxPhotoWaypointsPerTrail = 25

// movePhotosToWaypoints moves the trail photos that have a position onto
// waypoints before the import: photos near one of the item's own waypoints
// join it, the rest are grouped into new, unnamed waypoints. The first photo is
// the cover and stays a trail photo, and so do photos without a valid position and
// photos over the waypoint or per-waypoint limits, so none are lost.
func movePhotosToWaypoints(app core.App, item *pluginsystem.TrailImport, mapping map[string]importer.CategoryMappingValue) {
	if len(item.Photos) < 2 {
		return
	}
	settings, err := getWaypointMergeSettings(app, importer.CategoryTargetForImport(app, *item, mapping).CategoryID)
	if err != nil {
		app.Logger().Warn("keeping plugin photos on the trail, waypoint merge settings unavailable", "provider", item.Source.Provider, "external_id", item.Source.ExternalID, "error", err)
		return
	}
	groupPhotosIntoWaypoints(item, settings)
}

func groupPhotosIntoWaypoints(item *pluginsystem.TrailImport, settings waypointMergeSettings) {
	trailPhotos := []pluginsystem.Photo{item.Photos[0]}
	clusterPhotos := []waypointClusterPhoto{}
	// Clustering compares every photo with every cluster, and no more photos
	// than this can be kept on waypoints, so the rest stay on the trail.
	maxClusterPhotos := (maxPhotoWaypointsPerTrail + len(item.Waypoints)) * util.DefaultPluginMaxMediaItemsPerEntity
	for i, photo := range item.Photos[1:] {
		if photo.Lat == nil || photo.Lon == nil || *photo.Lat < -90 || *photo.Lat > 90 || *photo.Lon < -180 || *photo.Lon > 180 || len(clusterPhotos) >= maxClusterPhotos {
			trailPhotos = append(trailPhotos, photo)
			continue
		}
		clusterPhotos = append(clusterPhotos, waypointClusterPhoto{ID: strconv.Itoa(i + 1), Lat: *photo.Lat, Lon: *photo.Lon, Time: unixMillis(photo.TakenAt)})
	}
	if len(clusterPhotos) == 0 {
		return
	}

	waypoints := make([]waypointClusterWaypoint, len(item.Waypoints))
	for i, waypoint := range item.Waypoints {
		waypoints[i] = waypointClusterWaypoint{ID: strconv.Itoa(i), Lat: waypoint.Lat, Lon: waypoint.Lon, Time: unixMillis(waypoint.Time)}
	}

	newWaypoints := 0
	for _, cluster := range clusterWaypointPhotos(clusterPhotos, waypoints, settings) {
		photos := make([]pluginsystem.Photo, 0, len(cluster.Photos))
		for _, id := range cluster.Photos {
			index, _ := strconv.Atoi(id)
			photos = append(photos, item.Photos[index])
		}
		if cluster.Waypoint != "" {
			index, _ := strconv.Atoi(cluster.Waypoint)
			waypoint := &item.Waypoints[index]
			taken := min(len(photos), max(0, util.DefaultPluginMaxMediaItemsPerEntity-len(waypoint.Photos)))
			waypoint.Photos = append(waypoint.Photos, photos[:taken]...)
			trailPhotos = append(trailPhotos, photos[taken:]...)
			continue
		}
		if newWaypoints >= maxPhotoWaypointsPerTrail {
			trailPhotos = append(trailPhotos, photos...)
			continue
		}
		newWaypoints++
		taken := min(len(photos), util.DefaultPluginMaxMediaItemsPerEntity)
		item.Waypoints = append(item.Waypoints, pluginsystem.Waypoint{Lat: cluster.Lat, Lon: cluster.Lon, Photos: photos[:taken]})
		trailPhotos = append(trailPhotos, photos[taken:]...)
	}
	item.Photos = trailPhotos
}

func unixMillis(t *time.Time) *int64 {
	if t == nil {
		return nil
	}
	millis := t.UnixMilli()
	return &millis
}
