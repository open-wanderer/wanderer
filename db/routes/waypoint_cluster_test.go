package routes

import "testing"

func clusterTestPhoto(id string, lat float64, minutes *int64) waypointClusterPhoto {
	photo := waypointClusterPhoto{ID: id, Lat: lat, Lon: 11}
	if minutes != nil {
		millis := *minutes * 60 * 1000
		photo.Time = &millis
	}
	return photo
}

func minutes(m int64) *int64 {
	return &m
}

func TestClusterWaypointPhotosSplitsPassesByTime(t *testing.T) {
	settings := waypointMergeSettings{Enabled: true, Radius: defaultWaypointMergeRadius}

	// Same spot about 20 m apart, once on the way out and once on the way back.
	photos := []waypointClusterPhoto{
		clusterTestPhoto("out", 47.0, minutes(8)),
		clusterTestPhoto("back", 47.0002, minutes(300)),
	}
	if clusters := clusterWaypointPhotos(photos, nil, settings); len(clusters) != 2 {
		t.Fatalf("got %d clusters, want 2", len(clusters))
	}
}

func TestClusterWaypointPhotosMergesNearbyPhotosCloseInTime(t *testing.T) {
	settings := waypointMergeSettings{Enabled: true, Radius: defaultWaypointMergeRadius}

	// A chain of photos each within the gap of the previous one stays together.
	photos := []waypointClusterPhoto{
		clusterTestPhoto("a", 47.0, minutes(0)),
		clusterTestPhoto("b", 47.0001, minutes(10)),
		clusterTestPhoto("c", 47.0, minutes(20)),
	}
	clusters := clusterWaypointPhotos(photos, nil, settings)
	if len(clusters) != 1 || len(clusters[0].Photos) != 3 {
		t.Fatalf("got %+v, want one cluster with 3 photos", clusters)
	}
}

func TestClusterWaypointPhotosMergesWithoutTimes(t *testing.T) {
	settings := waypointMergeSettings{Enabled: true, Radius: defaultWaypointMergeRadius}

	photos := []waypointClusterPhoto{
		clusterTestPhoto("timed", 47.0, minutes(8)),
		clusterTestPhoto("untimed", 47.0002, nil),
	}
	waypoints := []waypointClusterWaypoint{{ID: "wp", Lat: 47.0001, Lon: 11}}
	clusters := clusterWaypointPhotos(photos, waypoints, settings)
	if len(clusters) != 1 || len(clusters[0].Photos) != 2 {
		t.Fatalf("got %+v, want one cluster with 2 photos", clusters)
	}
}
