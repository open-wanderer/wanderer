package main

import (
	"fmt"
	"testing"
)

func TestMatchAssetCandidatesReturnsAllMatchingCandidates(t *testing.T) {
	assets := make([]immichAsset, 0, 30)
	for i := 0; i < 30; i++ {
		lat := 46.0 + float64(i)*0.000001
		lon := 8.0
		assets = append(assets, immichAsset{
			ID:               fmt.Sprintf("asset-%02d", i),
			FileCreatedAt:    fmt.Sprintf("2026-01-01T10:%02d:00Z", i),
			OriginalFileName: fmt.Sprintf("asset-%02d.jpg", i),
			ExifInfo: immichExifInfo{
				Latitude:  &lat,
				Longitude: &lon,
			},
		})
	}

	candidates := sortMatches(matchAssetCandidates(assets, assetLibraryRequest{Lat: 46.0, Lon: 8.0}, immichConfig{MaxDistanceMeters: 1000}))

	if len(candidates) != len(assets) {
		t.Fatalf("got %d candidates, want %d", len(candidates), len(assets))
	}
	if candidates[0].AssetID != "asset-00" {
		t.Fatalf("got first candidate %q, want asset-00", candidates[0].AssetID)
	}
}

func TestMatchAssetCandidatesUsesMapBoundsWithoutAReferencePoint(t *testing.T) {
	tests := []struct {
		name   string
		bounds assetBounds
		lat    float64
		lon    float64
		want   bool
	}{
		{"inside", assetBounds{West: 7, South: 45, East: 9, North: 47}, 46, 8, true},
		{"southwest edge", assetBounds{West: 7, South: 45, East: 9, North: 47}, 45, 7, true},
		{"northeast edge", assetBounds{West: 7, South: 45, East: 9, North: 47}, 47, 9, true},
		{"west of bounds", assetBounds{West: 7, South: 45, East: 9, North: 47}, 46, 6, false},
		{"east of bounds", assetBounds{West: 7, South: 45, East: 9, North: 47}, 46, 10, false},
		{"south of bounds", assetBounds{West: 7, South: 45, East: 9, North: 47}, 44, 8, false},
		{"north of bounds", assetBounds{West: 7, South: 45, East: 9, North: 47}, 48, 8, false},
		{"origin", assetBounds{West: -1, South: -1, East: 1, North: 1}, 0, 0, true},
		{"antimeridian east", assetBounds{West: 170, South: -10, East: -170, North: 10}, 0, 175, true},
		{"antimeridian west", assetBounds{West: 170, South: -10, East: -170, North: 10}, 0, -175, true},
		{"antimeridian excludes Greenwich", assetBounds{West: 170, South: -10, East: -170, North: 10}, 0, 0, false},
		{"antimeridian still restricts latitude", assetBounds{West: 170, South: -10, East: -170, North: 10}, 20, 175, false},
		{"full world west", assetBounds{West: -180, South: -90, East: 180, North: 90}, -90, -180, true},
		{"full world east", assetBounds{West: -180, South: -90, East: 180, North: 90}, 90, 180, true},
		{"full world middle", assetBounds{West: -180, South: -90, East: 180, North: 90}, 0, 0, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			asset := geotaggedImmichAsset("photo", test.lat, test.lon)
			candidates := matchAssetCandidates([]immichAsset{asset}, assetLibraryRequest{Bounds: &test.bounds}, immichConfig{MaxDistanceMeters: 1})
			if got := len(candidates) == 1; got != test.want {
				t.Fatalf("matched = %v, want %v", got, test.want)
			}
			if test.want && (candidates[0].PointLat != test.lat || candidates[0].PointLon != test.lon || candidates[0].Distance != 0) {
				t.Fatalf("candidate should use photo coordinates without a route: %#v", candidates[0])
			}
		})
	}
}

func TestMatchAssetCandidatesBoundsOverrideRadiusAndRetainNearestRoutePoint(t *testing.T) {
	assets := []immichAsset{
		geotaggedImmichAsset("inside-map", 46.5, 8.5),
		geotaggedImmichAsset("outside-map-on-route", 46, 8),
	}
	request := assetLibraryRequest{
		Bounds: &assetBounds{West: 8.4, South: 46.4, East: 8.6, North: 46.6},
		Points: []trackPoint{
			{Lat: 46, Lon: 8, Distance: 0},
			{Lat: 46.1, Lon: 8.1, Distance: 12345},
		},
		DoubleRadius: true,
	}
	candidates := matchAssetCandidates(assets, request, immichConfig{MaxDistanceMeters: 10})
	if len(candidates) != 1 || candidates[0].AssetID != "inside-map" {
		t.Fatalf("candidates = %#v, want only photo inside map bounds", candidates)
	}
	candidate := candidates[0]
	if candidate.PointLat != 46.1 || candidate.PointLon != 8.1 || candidate.DistanceFromStart != 12345 || candidate.Distance < 10000 {
		t.Fatalf("nearest route point metadata not retained: %#v", candidate)
	}
}

func TestMatchAssetCandidatesBoundsRetainOwnershipAndGeotagFilters(t *testing.T) {
	owned := geotaggedImmichAsset("owned", 46, 8)
	owned.OwnerID = "user"
	other := geotaggedImmichAsset("other", 46, 8)
	other.OwnerID = "other-user"
	noCoordinates := immichAsset{ID: "no-coordinates", OwnerID: "user"}
	noLongitude := geotaggedImmichAsset("no-longitude", 46, 8)
	noLongitude.OwnerID = "user"
	noLongitude.ExifInfo.Longitude = nil
	candidates := matchAssetCandidates([]immichAsset{owned, other, noCoordinates, noLongitude}, assetLibraryRequest{
		Bounds: &assetBounds{West: 7, South: 45, East: 9, North: 47},
	}, immichConfig{OwnedOnly: true, UserID: "user"})
	if len(candidates) != 1 || candidates[0].AssetID != "owned" {
		t.Fatalf("candidates = %#v, want only owned geotagged photo", candidates)
	}
}

func TestMatchAssetCandidatesWithoutBoundsRetainsRadius(t *testing.T) {
	assets := []immichAsset{geotaggedImmichAsset("photo", 46.001, 8)}
	config := immichConfig{MaxDistanceMeters: 100}
	for _, request := range []assetLibraryRequest{
		{Lat: 46, Lon: 8},
		{Points: []trackPoint{{Lat: 46, Lon: 8}}},
	} {
		if candidates := matchAssetCandidates(assets, request, config); len(candidates) != 0 {
			t.Fatalf("photo outside radius matched: %#v", candidates)
		}
		request.DoubleRadius = true
		if candidates := matchAssetCandidates(assets, request, config); len(candidates) != 1 {
			t.Fatalf("photo within doubled radius not matched: %#v", candidates)
		}
	}
	if candidates := matchAssetCandidates(assets, assetLibraryRequest{}, config); len(candidates) != 0 {
		t.Fatalf("request without location or bounds unexpectedly matched: %#v", candidates)
	}
}

func geotaggedImmichAsset(id string, lat, lon float64) immichAsset {
	return immichAsset{ID: id, ExifInfo: immichExifInfo{Latitude: &lat, Longitude: &lon}}
}
