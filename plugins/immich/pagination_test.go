package main

import (
	"fmt"
	"testing"
)

func TestSearchAssetCandidatePagesContinuesDenseProviderPageWithoutLoss(t *testing.T) {
	page := assetCandidateProviderPage{Items: testImmichAssets(250, func(int) bool { return true })}
	pageCalls := []int{}
	fetchPage := func(pageNumber int) (assetCandidateProviderPage, error) {
		pageCalls = append(pageCalls, pageNumber)
		return page, nil
	}
	limits := assetCandidateSearchLimits{MaxItems: 100, MaxScannedItems: 2500, MaxProviderRequests: 10}
	request := assetLibraryRequest{Lat: 46, Lon: 8}
	config := immichConfig{MaxDistanceMeters: 100}
	seen := map[string]bool{}
	var state map[string]any
	wantOffsets := []int{100, 200}
	wantScanned := []int{100, 100, 50}

	for batch := 0; batch < 3; batch++ {
		result, err := searchAssetCandidatePages(request, config, state, limits, fetchPage)
		if err != nil {
			t.Fatal(err)
		}
		if result.ScannedItems != wantScanned[batch] {
			t.Fatalf("batch %d scanned %d items, want %d", batch, result.ScannedItems, wantScanned[batch])
		}
		for _, candidate := range result.Candidates {
			if seen[candidate.AssetID] {
				t.Fatalf("candidate %q returned more than once", candidate.AssetID)
			}
			seen[candidate.AssetID] = true
		}
		if batch < 2 {
			if !result.HasMore || assetCandidateStateInt(result.State, "page", 0) != 1 || assetCandidateStateInt(result.State, "offset", -1) != wantOffsets[batch] {
				t.Fatalf("batch %d continuation = %#v, hasMore=%v", batch, result.State, result.HasMore)
			}
			state = result.State
		} else if result.HasMore || len(result.State) != 0 {
			t.Fatalf("final continuation = %#v, hasMore=%v", result.State, result.HasMore)
		}
	}

	if len(seen) != 250 {
		t.Fatalf("got %d unique candidates, want 250", len(seen))
	}
	if len(pageCalls) != 3 || pageCalls[0] != 1 || pageCalls[1] != 1 || pageCalls[2] != 1 {
		t.Fatalf("provider page calls = %#v, want page 1 reloaded three times", pageCalls)
	}
}

func TestSearchAssetCandidatePagesStopsScanBudgetMidPage(t *testing.T) {
	page := assetCandidateProviderPage{Items: testImmichAssets(100, func(index int) bool { return index%20 == 0 })}
	fetchPage := func(int) (assetCandidateProviderPage, error) { return page, nil }
	limits := assetCandidateSearchLimits{MaxItems: 100, MaxScannedItems: 55, MaxProviderRequests: 10}
	request := assetLibraryRequest{Lat: 46, Lon: 8}
	config := immichConfig{MaxDistanceMeters: 100}

	first, err := searchAssetCandidatePages(request, config, nil, limits, fetchPage)
	if err != nil {
		t.Fatal(err)
	}
	if !first.HasMore || first.ScannedItems != 55 || len(first.Candidates) != 3 {
		t.Fatalf("first batch = %#v", first)
	}
	if offset := assetCandidateStateInt(first.State, "offset", -1); offset != 55 {
		t.Fatalf("first offset = %d, want raw asset offset 55", offset)
	}

	second, err := searchAssetCandidatePages(request, config, first.State, limits, fetchPage)
	if err != nil {
		t.Fatal(err)
	}
	if second.HasMore || second.ScannedItems != 45 || len(second.Candidates) != 2 {
		t.Fatalf("second batch = %#v", second)
	}
	got := map[string]bool{}
	for _, result := range []assetCandidateSearchResult{first, second} {
		for _, candidate := range result.Candidates {
			got[candidate.AssetID] = true
		}
	}
	for _, assetID := range []string{"asset-0", "asset-20", "asset-40", "asset-60", "asset-80"} {
		if !got[assetID] {
			t.Fatalf("candidate %q was skipped across scan continuation", assetID)
		}
	}
}

func TestSearchAssetCandidatePagesOffsetCountsRawAssetsNotCandidates(t *testing.T) {
	page := assetCandidateProviderPage{Items: testImmichAssets(10, func(index int) bool { return index == 4 || index == 8 })}
	fetchPage := func(int) (assetCandidateProviderPage, error) { return page, nil }
	limits := assetCandidateSearchLimits{MaxItems: 1, MaxScannedItems: 100, MaxProviderRequests: 10}
	request := assetLibraryRequest{Lat: 46, Lon: 8}
	config := immichConfig{MaxDistanceMeters: 100}

	first, err := searchAssetCandidatePages(request, config, nil, limits, fetchPage)
	if err != nil {
		t.Fatal(err)
	}
	if first.ScannedItems != 5 || len(first.Candidates) != 1 || first.Candidates[0].AssetID != "asset-4" {
		t.Fatalf("first batch = %#v", first)
	}
	if offset := assetCandidateStateInt(first.State, "offset", -1); offset != 5 {
		t.Fatalf("first offset = %d, want 5 raw assets", offset)
	}

	second, err := searchAssetCandidatePages(request, config, first.State, limits, fetchPage)
	if err != nil {
		t.Fatal(err)
	}
	if second.ScannedItems != 4 || len(second.Candidates) != 1 || second.Candidates[0].AssetID != "asset-8" {
		t.Fatalf("second batch = %#v", second)
	}
	if offset := assetCandidateStateInt(second.State, "offset", -1); offset != 9 {
		t.Fatalf("second offset = %d, want 9 raw assets", offset)
	}

	final, err := searchAssetCandidatePages(request, config, second.State, limits, fetchPage)
	if err != nil {
		t.Fatal(err)
	}
	if final.HasMore || final.ScannedItems != 1 || len(final.Candidates) != 0 {
		t.Fatalf("final batch = %#v", final)
	}
}

func TestSearchAssetCandidatePagesStopsAtProviderRequestBudget(t *testing.T) {
	pages := map[int]assetCandidateProviderPage{
		1: {Items: testImmichAssets(1, func(int) bool { return false }), NextPage: stringPointer("2")},
		2: {Items: testImmichAssets(1, func(int) bool { return false }), NextPage: stringPointer("3")},
		3: {Items: testImmichAssets(1, func(int) bool { return false })},
	}
	result, err := searchAssetCandidatePages(assetLibraryRequest{Lat: 46, Lon: 8}, immichConfig{MaxDistanceMeters: 100}, nil, assetCandidateSearchLimits{
		MaxItems:            100,
		MaxScannedItems:     100,
		MaxProviderRequests: 2,
	}, func(page int) (assetCandidateProviderPage, error) {
		return pages[page], nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasMore || result.ProviderRequests != 2 || result.ScannedItems != 2 {
		t.Fatalf("result = %#v", result)
	}
	if page := assetCandidateStateInt(result.State, "page", 0); page != 3 {
		t.Fatalf("continuation page = %d, want 3", page)
	}
}

func TestNextCandidatePageRejectsNonProgressingValues(t *testing.T) {
	for _, value := range []string{"invalid", "1", "2"} {
		if _, _, err := nextCandidatePage(2, stringPointer(value)); err == nil {
			t.Fatalf("nextPage %q was accepted after page 2", value)
		}
	}
	next, hasMore, err := nextCandidatePage(2, stringPointer("3"))
	if err != nil || !hasMore || next != 3 {
		t.Fatalf("valid next page = %d, hasMore=%v, err=%v", next, hasMore, err)
	}
}

func testImmichAssets(count int, matches func(index int) bool) []immichAsset {
	assets := make([]immichAsset, 0, count)
	for index := 0; index < count; index++ {
		asset := immichAsset{ID: fmt.Sprintf("asset-%d", index), FileCreatedAt: fmt.Sprintf("2026-01-01T10:%02d:00Z", index%60)}
		if matches(index) {
			lat := 46.0
			lon := 8.0
			asset.ExifInfo.Latitude = &lat
			asset.ExifInfo.Longitude = &lon
		}
		assets = append(assets, asset)
	}
	return assets
}

func stringPointer(value string) *string {
	return &value
}
