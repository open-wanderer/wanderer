package routes

import (
	"testing"
	"time"

	"pocketbase/pluginsystem"
)

func positionedPhoto(id string, lat float64) pluginsystem.Photo {
	lon := 11.0
	return pluginsystem.Photo{ExternalID: id, Lat: &lat, Lon: &lon}
}

func photoIDs(photos []pluginsystem.Photo) []string {
	ids := make([]string, len(photos))
	for i, photo := range photos {
		ids[i] = photo.ExternalID
	}
	return ids
}

func TestMovePhotosToWaypoints(t *testing.T) {
	item := pluginsystem.TrailImport{
		Waypoints: []pluginsystem.Waypoint{{Name: "Hut", Lat: 47.0, Lon: 11}},
		Photos: []pluginsystem.Photo{
			positionedPhoto("cover", 47.0),
			{ExternalID: "no-position"},
			positionedPhoto("bad-position", 95),
			positionedPhoto("at-hut", 47.0002),
			positionedPhoto("lake-1", 47.1),
			positionedPhoto("lake-2", 47.1002),
		},
	}

	movePhotosToWaypoints(nil, &item, nil)

	if got := photoIDs(item.Photos); len(got) != 3 || got[0] != "cover" || got[1] != "no-position" || got[2] != "bad-position" {
		t.Fatalf("trail photos = %v, want [cover no-position bad-position]", got)
	}
	if len(item.Waypoints) != 2 {
		t.Fatalf("got %d waypoints, want 2", len(item.Waypoints))
	}
	if got := photoIDs(item.Waypoints[0].Photos); len(got) != 1 || got[0] != "at-hut" {
		t.Fatalf("hut photos = %v, want [at-hut]", got)
	}
	lake := item.Waypoints[1]
	if got := photoIDs(lake.Photos); lake.Name != "" || len(got) != 2 || got[0] != "lake-1" || got[1] != "lake-2" {
		t.Fatalf("new waypoint = %+v, want an unnamed waypoint with both lake photos", lake)
	}
}

func TestGroupPhotosIntoWaypointsCapsNewWaypoints(t *testing.T) {
	item := pluginsystem.TrailImport{Photos: []pluginsystem.Photo{{ExternalID: "cover"}}}
	for i := range 30 {
		item.Photos = append(item.Photos, positionedPhoto("p", 47.0+float64(i)*0.01))
	}

	groupPhotosIntoWaypoints(&item, waypointMergeSettings{Enabled: false})

	if len(item.Waypoints) != maxPhotoWaypointsPerTrail {
		t.Fatalf("got %d waypoints, want %d", len(item.Waypoints), maxPhotoWaypointsPerTrail)
	}
	if len(item.Photos) != 6 {
		t.Fatalf("got %d trail photos, want the cover and the 5 over the cap", len(item.Photos))
	}
}

func TestGroupPhotosIntoWaypointsKeepsPhotosOverTheWaypointLimit(t *testing.T) {
	item := pluginsystem.TrailImport{Photos: []pluginsystem.Photo{{ExternalID: "cover"}}}
	for range 25 {
		item.Photos = append(item.Photos, positionedPhoto("p", 47.0))
	}

	groupPhotosIntoWaypoints(&item, waypointMergeSettings{Enabled: true, Radius: defaultWaypointMergeRadius})

	if len(item.Waypoints) != 1 {
		t.Fatalf("got %d waypoints, want 1", len(item.Waypoints))
	}
	if len(item.Waypoints[0].Photos) != 20 {
		t.Fatalf("got %d waypoint photos, want 20", len(item.Waypoints[0].Photos))
	}
	if len(item.Photos) != 6 {
		t.Fatalf("got %d trail photos, want the cover and the 5 over the limit", len(item.Photos))
	}
}

func TestGroupPhotosIntoWaypointsBoundsClusteringWork(t *testing.T) {
	const total = 100_000
	item := pluginsystem.TrailImport{Photos: make([]pluginsystem.Photo, 0, total+1)}
	item.Photos = append(item.Photos, pluginsystem.Photo{ExternalID: "cover"})
	for i := range total {
		lat, lon := -80+float64(i%1000)*0.16, -170+float64(i/1000)*3.4
		item.Photos = append(item.Photos, pluginsystem.Photo{Lat: &lat, Lon: &lon})
	}

	start := time.Now()
	groupPhotosIntoWaypoints(&item, waypointMergeSettings{Enabled: true, Radius: defaultWaypointMergeRadius})

	if elapsed := time.Since(start); elapsed > 20*time.Second {
		t.Fatalf("grouping took %v, want it bounded by the photo cap", elapsed)
	}
	if len(item.Waypoints) != maxPhotoWaypointsPerTrail {
		t.Fatalf("got %d waypoints, want %d", len(item.Waypoints), maxPhotoWaypointsPerTrail)
	}
	if want := total + 1 - maxPhotoWaypointsPerTrail; len(item.Photos) != want {
		t.Fatalf("got %d trail photos, want %d", len(item.Photos), want)
	}
}

func TestGroupPhotosIntoWaypointsEdgeCases(t *testing.T) {
	at := func(id string, lat, lon float64) pluginsystem.Photo {
		return pluginsystem.Photo{ExternalID: id, Lat: &lat, Lon: &lon}
	}
	taken := func(id string, minutes int) pluginsystem.Photo {
		photo := at(id, 47, 11)
		ts := time.Date(2026, 1, 1, 10, minutes, 0, 0, time.UTC)
		photo.TakenAt = &ts
		return photo
	}
	merging := waypointMergeSettings{Enabled: true, Radius: defaultWaypointMergeRadius}

	t.Run("existing waypoint keeps its photo limit", func(t *testing.T) {
		item := pluginsystem.TrailImport{
			Waypoints: []pluginsystem.Waypoint{{Lat: 47, Lon: 11, Photos: make([]pluginsystem.Photo, 19)}},
			Photos:    []pluginsystem.Photo{{ExternalID: "cover"}},
		}
		for range 5 {
			item.Photos = append(item.Photos, at("p", 47, 11))
		}
		groupPhotosIntoWaypoints(&item, merging)
		if len(item.Waypoints) != 1 || len(item.Waypoints[0].Photos) != 20 {
			t.Fatalf("waypoints = %d, photos on the first = %d, want 1 and 20", len(item.Waypoints), len(item.Waypoints[0].Photos))
		}
		if len(item.Photos) != 5 {
			t.Fatalf("got %d trail photos, want the cover and 4 over the limit", len(item.Photos))
		}
	})

	t.Run("out of range positions stay on the trail", func(t *testing.T) {
		item := pluginsystem.TrailImport{Photos: []pluginsystem.Photo{
			{ExternalID: "cover"}, at("lat", -91, 11), at("lon-high", 47, 181), at("lon-low", 47, -181),
		}}
		groupPhotosIntoWaypoints(&item, merging)
		if len(item.Waypoints) != 0 || len(item.Photos) != 4 {
			t.Fatalf("waypoints = %d, trail photos = %d, want 0 and 4", len(item.Waypoints), len(item.Photos))
		}
	})

	t.Run("zero or one photo is left alone", func(t *testing.T) {
		for _, photos := range [][]pluginsystem.Photo{nil, {at("cover", 47, 11)}} {
			item := pluginsystem.TrailImport{Photos: photos}
			movePhotosToWaypoints(nil, &item, nil)
			if len(item.Photos) != len(photos) || len(item.Waypoints) != 0 {
				t.Fatalf("item changed: %d photos, %d waypoints", len(item.Photos), len(item.Waypoints))
			}
		}
	})

	t.Run("photos far apart in time make separate waypoints", func(t *testing.T) {
		item := pluginsystem.TrailImport{Photos: []pluginsystem.Photo{
			{ExternalID: "cover"}, taken("early", 0), taken("late", 50),
		}}
		groupPhotosIntoWaypoints(&item, merging)
		if len(item.Waypoints) != 2 {
			t.Fatalf("got %d waypoints, want 2", len(item.Waypoints))
		}
	})
}
