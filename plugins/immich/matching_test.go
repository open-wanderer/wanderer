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
