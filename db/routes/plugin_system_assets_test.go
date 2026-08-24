package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pocketbase/pluginsystem"
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/core"
)

func TestAssetPluginProviderEnabledDefaultsToEnabled(t *testing.T) {
	cases := []struct {
		name       string
		hostConfig map[string]any
		provider   string
		want       bool
	}{
		{"missing autoAttach enables upload", map[string]any{}, "upload", true},
		{"missing autoAttach enables trail plugins", map[string]any{}, "hammerhead", true},
		{"missing key enables provider", map[string]any{"autoAttach": map[string]any{}}, "upload", true},
		{"explicit upload false disables upload", map[string]any{"autoAttach": map[string]any{"upload": false}}, "upload", false},
		{"explicit trail false disables trail plugins", map[string]any{"autoAttach": map[string]any{"trailPlugins": false}}, "komoot", false},
		{"explicit trail false keeps manual maintenance enabled", map[string]any{"autoAttach": map[string]any{"trailPlugins": false}}, pluginAssetMaintenanceProvider, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := assetPluginProviderEnabled(tc.hostConfig, tc.provider); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAssetPluginAutoAttachAllowedForTrail(t *testing.T) {
	cases := []struct {
		name      string
		provider  string
		completed bool
		want      bool
	}{
		{"completed trail plugin import", "hammerhead", true, true},
		{"planned trail plugin import", "hammerhead", false, false},
		{"completed upload", "upload", true, true},
		{"not completed upload", "upload", false, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := assetPluginAutoAttachAllowedForTrail(tc.provider, tc.completed); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAssetMaintenanceIsGeneratedRoutePreviewAsset(t *testing.T) {
	cases := []struct {
		name   string
		values map[string]any
		want   bool
	}{
		{
			name: "metadata kind",
			values: map[string]any{
				"file": "photo.jpg",
				"metadata": map[string]any{
					"generated": map[string]any{"kind": "route-preview"},
				},
			},
			want: true,
		},
		{
			name:   "legacy wanderer route preview filename",
			values: map[string]any{"file": "wanderer-route-preview.webp"},
			want:   true,
		},
		{
			name:   "legacy route hash filename",
			values: map[string]any{"file": "route_abcdef123.webp"},
			want:   true,
		},
		{
			name: "external provider with matching filename is a regular photo",
			values: map[string]any{
				"file":              "route_abcdef123.webp",
				"external_provider": "immich",
			},
			want: false,
		},
		{
			name: "taken at with matching filename is a regular photo",
			values: map[string]any{
				"file":     "route_abcdef123.webp",
				"taken_at": time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
			},
			want: false,
		},
		{
			name:   "regular photo",
			values: map[string]any{"file": "photo.jpg"},
			want:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			record := newAssetMaintenanceRecord(tc.values)
			if got := util.IsGeneratedRoutePreviewAsset(record); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestAssetPluginAutoAttachHasTimeWindow(t *testing.T) {
	cases := []struct {
		name    string
		request pluginAssetLibraryActionInput
		want    bool
	}{
		{"started and ended", pluginAssetLibraryActionInput{StartedAt: "2026-01-01T10:00:00Z", EndedAt: "2026-01-01T11:00:00Z"}, true},
		{"missing started", pluginAssetLibraryActionInput{EndedAt: "2026-01-01T11:00:00Z"}, false},
		{"missing ended", pluginAssetLibraryActionInput{StartedAt: "2026-01-01T10:00:00Z"}, false},
		{"blank values", pluginAssetLibraryActionInput{StartedAt: " ", EndedAt: "\t"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := assetPluginAutoAttachHasTimeWindow(tc.request); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTrailTrackPointsFromBytesUsesTimestamps(t *testing.T) {
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="test" xmlns="http://www.topografix.com/GPX/1/1">
  <trk>
    <trkseg>
      <trkpt lat="46.0000" lon="8.0000"><time>2026-01-01T10:00:00Z</time></trkpt>
      <trkpt lat="46.0100" lon="8.0100"><time>2026-01-01T10:15:00Z</time></trkpt>
    </trkseg>
  </trk>
</gpx>`)

	points, start, end, err := trailTrackPointsFromBytes(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(points) != 2 {
		t.Fatalf("got %d points, want 2", len(points))
	}
	if points[0].Timestamp != "2026-01-01T10:00:00Z" || points[1].Timestamp != "2026-01-01T10:15:00Z" {
		t.Fatalf("unexpected point timestamps: %#v", points)
	}
	if start.Format(time.RFC3339) != "2026-01-01T10:00:00Z" {
		t.Fatalf("got start %s", start.Format(time.RFC3339))
	}
	if end.Format(time.RFC3339) != "2026-01-01T10:15:00Z" {
		t.Fatalf("got end %s", end.Format(time.RFC3339))
	}
}

func TestApplyAssetLibraryTrackPointsDoesNotSetTimeWindow(t *testing.T) {
	request := pluginAssetLibraryActionInput{}
	start := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)

	applyAssetLibraryTrackPoints(&request, nil, start, end)

	if request.StartedAt != "" || request.EndedAt != "" {
		t.Fatalf("got %s - %s, want no time window", request.StartedAt, request.EndedAt)
	}
}

func TestApplyAssetLibraryTrailTimeWindowSetsStartedAndEndedAt(t *testing.T) {
	request := pluginAssetLibraryActionInput{}
	start := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)

	applyAssetLibraryTrailTimeWindow(&request, start, end)

	if request.StartedAt != "2026-01-10T10:00:00Z" || request.EndedAt != "2026-01-10T12:00:00Z" {
		t.Fatalf("got %s - %s, want trail time window", request.StartedAt, request.EndedAt)
	}
	if request.TakenAfter != "" || request.TakenBefore != "" {
		t.Fatalf("got explicit window %s - %s, want none", request.TakenAfter, request.TakenBefore)
	}
}

func TestApplyAssetLibraryTrackPointsKeepsExplicitTimeWindow(t *testing.T) {
	request := pluginAssetLibraryActionInput{
		TakenAfter:  "2026-01-01T00:00:00Z",
		TakenBefore: "2026-01-31T23:59:59Z",
	}
	if err := applyAssetLibraryExplicitTimeWindow(&request); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	start := time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)

	applyAssetLibraryTrackPoints(&request, nil, start, end)

	if request.StartedAt != "2026-01-01T00:00:00Z" || request.EndedAt != "2026-01-31T23:59:59Z" {
		t.Fatalf("got %s - %s, want explicit window", request.StartedAt, request.EndedAt)
	}
}

func TestPluginAssetLibraryRequestTracksZeroLocationPresence(t *testing.T) {
	var request pluginAssetLibraryRequest
	if err := json.Unmarshal([]byte(`{"lat":0,"lon":0}`), &request); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !request.latSet || !request.lonSet {
		t.Fatalf("got latSet=%v lonSet=%v, want both present", request.latSet, request.lonSet)
	}
}

func TestPluginAssetLibraryRequestIgnoresNullLocationPresence(t *testing.T) {
	var request pluginAssetLibraryRequest
	if err := json.Unmarshal([]byte(`{"lat":null,"lon":null}`), &request); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if request.latSet || request.lonSet {
		t.Fatalf("got latSet=%v lonSet=%v, want both absent", request.latSet, request.lonSet)
	}
}

func TestGeocodingUsesTrustedServiceClientForCustomServiceBase(t *testing.T) {
	cases := []struct {
		baseURL string
		want    bool
	}{
		{waypointNameOverpassDefaultURL, false},
		{waypointNameNominatimDefaultURL + "/", false},
		{"http://nominatim:8080", true},
		{"https://maps.example.test/nominatim", true},
	}
	for _, tc := range cases {
		if got := geocodingUsesTrustedServiceClient(tc.baseURL); got != tc.want {
			t.Fatalf("geocodingUsesTrustedServiceClient(%q) = %v, want %v", tc.baseURL, got, tc.want)
		}
	}
}

func TestNormalizeAssetLibraryDateFilterUsesPocketBaseDateLayout(t *testing.T) {
	got, err := normalizeAssetLibraryDateFilter("2026-07-05T09:00:00Z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "2026-07-05 09:00:00.000Z" {
		t.Fatalf("got %q, want PocketBase date layout", got)
	}
}

func TestAssetLibraryPaginationForRequestUsesDefaultsAndClamps(t *testing.T) {
	pagination := assetLibraryPaginationForRequest(pluginAssetLibraryRequest{
		Page:    0,
		PerPage: 999,
	}, pluginAssetLibraryActionInput{}, false)

	if !pagination.Enabled {
		t.Fatal("pagination should be enabled")
	}
	if pagination.Page != 1 || pagination.PerPage != 250 {
		t.Fatalf("pagination = %#v, want page 1 perPage 250", pagination)
	}
}

func TestAssetLibraryPaginationForRequestSkipsSpatialMatching(t *testing.T) {
	withLocation := assetLibraryPaginationForRequest(pluginAssetLibraryRequest{
		Page:    2,
		PerPage: 100,
	}, pluginAssetLibraryActionInput{}, true)
	if withLocation.Enabled {
		t.Fatal("location matching should not be paginated before distance ranking")
	}

	withTrack := assetLibraryPaginationForRequest(pluginAssetLibraryRequest{
		Page:    2,
		PerPage: 100,
	}, pluginAssetLibraryActionInput{Points: []pluginAssetTrackPoint{{Lat: 46, Lon: 8}}}, false)
	if withTrack.Enabled {
		t.Fatal("track matching should not be paginated before route ranking")
	}
}

func TestAssetLibraryFilenameKeepsStoredFileBeforeRemoteName(t *testing.T) {
	record := newAssetMaintenanceRecord(map[string]any{
		"file":        "stored.jpg",
		"external_id": "external-id",
	})
	metadata := map[string]any{
		"remote": map[string]any{"filename": "remote.jpg"},
	}

	if got := assetLibraryFilename(record, metadata); got != "stored.jpg" {
		t.Fatalf("got %q, want stored file name", got)
	}
}

func TestSortAssetLibraryCandidatesUsesRouteOrderBeforeDistance(t *testing.T) {
	candidates := []pluginAssetCandidate{
		{AssetID: "later-nearer", DistanceFromStart: 300, Distance: 1, TakenAt: "2026-01-02T00:00:00Z"},
		{AssetID: "earlier-farther", DistanceFromStart: 100, Distance: 50, TakenAt: "2026-01-01T00:00:00Z"},
	}

	sortAssetLibraryCandidates(candidates, true)

	if candidates[0].AssetID != "earlier-farther" {
		t.Fatalf("got first candidate %q, want route-order candidate", candidates[0].AssetID)
	}
}

func TestSortAssetPluginAutoAttachCandidatesUsesStableGlobalOrder(t *testing.T) {
	candidates := []pluginAssetCandidate{
		{AssetID: "invalid-time", DistanceFromStart: 10, Distance: 5, TakenAt: "invalid"},
		{AssetID: "later", DistanceFromStart: 10, Distance: 5, TakenAt: "2026-01-02T00:00:00Z"},
		{AssetID: "earlier", DistanceFromStart: 10, Distance: 5, TakenAt: "2026-01-01T00:00:00Z"},
		{AssetID: "near-start", DistanceFromStart: 5, Distance: 100, TakenAt: "invalid"},
	}

	sortAssetPluginAutoAttachCandidates(candidates)

	want := []string{"near-start", "earlier", "later", "invalid-time"}
	for index, assetID := range want {
		if candidates[index].AssetID != assetID {
			t.Fatalf("candidate %d = %q, want %q", index, candidates[index].AssetID, assetID)
		}
	}
}

func TestStableUniqueAssetIDsTrimsAndPreservesFirstOccurrence(t *testing.T) {
	got := stableUniqueAssetIDs([]string{" a ", "b", "a", "", " b "})
	want := []string{"a", "b"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestNormalizeAssetPluginImportIDsChecksLimitAfterDeduplication(t *testing.T) {
	duplicateValues := make([]string, pluginAssetImportMaxIDs+1)
	for index := range duplicateValues {
		duplicateValues[index] = "same"
	}
	got, err := normalizeAssetPluginImportIDs(duplicateValues)
	if err != nil || len(got) != 1 {
		t.Fatalf("deduplicated request got %#v, err=%v", got, err)
	}

	uniqueValues := make([]string, pluginAssetImportMaxIDs+1)
	for index := range uniqueValues {
		uniqueValues[index] = fmt.Sprintf("asset-%d", index)
	}
	if _, err := normalizeAssetPluginImportIDs(uniqueValues); err == nil {
		t.Fatalf("request with %d unique IDs was accepted", len(uniqueValues))
	}
}

func TestAssetImportAtMaximumSizeGetsSufficientRequestBudget(t *testing.T) {
	request := pluginAssetLibraryActionInput{Action: "import", AssetIDs: make([]string, pluginAssetImportMaxIDs)}
	options := assetPluginRuntimeCallOptions(request)
	want := pluginAssetImportMaxIDs + 8
	if options.MaxHostRequests != want {
		t.Fatalf("budget = %d, want %d", options.MaxHostRequests, want)
	}
	if pluginsystem.EffectiveMaxHostRequests(options) != want {
		t.Fatalf("effective budget does not permit maximum import: %#v", options)
	}
}

func TestValidateAssetPluginImportPartition(t *testing.T) {
	valid := pluginAssetLibraryOutput{
		Photos:        []pluginsystem.Photo{{ExternalID: "a"}},
		OmittedAssets: []pluginAssetOmission{{AssetID: "b", Reason: "unsupported"}},
	}
	if err := validateAssetPluginImportPartition([]string{"a", "b"}, valid); err != nil {
		t.Fatalf("valid partition rejected: %v", err)
	}

	cases := []struct {
		name   string
		output pluginAssetLibraryOutput
	}{
		{"missing", pluginAssetLibraryOutput{Photos: []pluginsystem.Photo{{ExternalID: "a"}}}},
		{"unrequested", pluginAssetLibraryOutput{Photos: []pluginsystem.Photo{{ExternalID: "a"}, {ExternalID: "c"}}, OmittedAssets: []pluginAssetOmission{{AssetID: "b", Reason: "unsupported"}}}},
		{"duplicate photo", pluginAssetLibraryOutput{Photos: []pluginsystem.Photo{{ExternalID: "a"}, {ExternalID: "a"}}, OmittedAssets: []pluginAssetOmission{{AssetID: "b", Reason: "unsupported"}}}},
		{"overlap", pluginAssetLibraryOutput{Photos: []pluginsystem.Photo{{ExternalID: "a"}}, OmittedAssets: []pluginAssetOmission{{AssetID: "a", Reason: "unsupported"}, {AssetID: "b", Reason: "unsupported"}}}},
		{"unrequested omission", pluginAssetLibraryOutput{Photos: []pluginsystem.Photo{{ExternalID: "a"}, {ExternalID: "b"}}, OmittedAssets: []pluginAssetOmission{{AssetID: "c", Reason: "unsupported"}}}},
		{"duplicate omission", pluginAssetLibraryOutput{Photos: []pluginsystem.Photo{{ExternalID: "a"}}, OmittedAssets: []pluginAssetOmission{{AssetID: "b", Reason: "unsupported"}, {AssetID: "b", Reason: "unsupported"}}}},
		{"omission without reason", pluginAssetLibraryOutput{Photos: []pluginsystem.Photo{{ExternalID: "a"}}, OmittedAssets: []pluginAssetOmission{{AssetID: "b"}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateAssetPluginImportPartition([]string{"a", "b"}, tc.output); err == nil {
				t.Fatal("invalid partition accepted")
			}
		})
	}
}

func TestValidatePluginAssetContinuationRejectsProtocolViolations(t *testing.T) {
	if _, _, err := validatePluginAssetContinuation(nil, nil, true, nil); err == nil {
		t.Fatal("hasMore without state was accepted")
	}
	state := map[string]any{"page": 2.0, "offset": 10.0}
	if _, _, err := validatePluginAssetContinuation(state, state, true, nil); err == nil {
		t.Fatal("unchanged state was accepted")
	}
	hash, _, err := pluginAssetStateHash(map[string]any{"page": 3})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := validatePluginAssetContinuation(state, map[string]any{"page": 3}, true, map[string]bool{hash: true}); err == nil {
		t.Fatal("cyclic state was accepted")
	}
}

func TestPluginAssetCandidatesResponseDoesNotExposeRawStateOrStats(t *testing.T) {
	encoded, err := json.Marshal(pluginAssetCandidatesResponse{Candidates: []pluginAssetCandidate{}, CursorID: "opaque"})
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	if strings.Contains(text, `"state"`) || strings.Contains(text, `"stats"`) {
		t.Fatalf("browser response exposed plugin internals: %s", text)
	}
}

func TestPluginAssetCursorIsOpaqueBoundAndProgressive(t *testing.T) {
	store := pluginAssetCursorStore{items: map[string]pluginAssetCursorEntry{}}
	binding := pluginAssetCursorBinding{UserID: "user", ActorID: "actor", PluginID: "immich", InstanceID: "instance", QueryFingerprint: "query", ConfigFingerprint: "config"}
	state := map[string]any{"page": 2, "offset": 10}
	stateHash, _, err := pluginAssetStateHash(state)
	if err != nil {
		t.Fatal(err)
	}
	cursorID, err := store.create(binding, state, stateHash)
	if err != nil {
		t.Fatal(err)
	}
	if len(cursorID) != 32 || strings.Contains(cursorID, "page") {
		t.Fatalf("cursor ID is not an opaque 128-bit token: %q", cursorID)
	}
	entry, restartRequired, err := store.resume(cursorID, binding)
	if err != nil || restartRequired || entry.StateHash != stateHash {
		t.Fatalf("resume = %#v, restart=%v, err=%v", entry, restartRequired, err)
	}
	wrongBinding := binding
	wrongBinding.UserID = "other"
	if _, restartRequired, err := store.resume(cursorID, wrongBinding); err != nil || !restartRequired {
		t.Fatalf("mismatched binding restart=%v err=%v", restartRequired, err)
	}

	nextState := map[string]any{"page": 3, "offset": 0}
	nextHash, _, err := pluginAssetStateHash(nextState)
	if err != nil {
		t.Fatal(err)
	}
	restartRequired, err = store.advance(cursorID, binding, stateHash, nextState, nextHash, map[string]bool{stateHash: true, nextHash: true}, true)
	if err != nil || restartRequired {
		t.Fatalf("advance restart=%v err=%v", restartRequired, err)
	}
	resumed, restartRequired, err := store.resume(cursorID, binding)
	if err != nil || restartRequired {
		t.Fatalf("second resume restart=%v err=%v", restartRequired, err)
	}
	if _, _, err := validatePluginAssetContinuation(resumed.State, state, true, resumed.Seen); err == nil {
		t.Fatal("A -> B -> A cycle across cursor requests was accepted")
	}
	if restartRequired, err = store.advance(cursorID, binding, stateHash, nextState, nextHash, nil, true); err != nil || !restartRequired {
		t.Fatalf("stale advance restart=%v err=%v", restartRequired, err)
	}
}

func TestPluginAssetCursorDiscardsSameUserBindingMismatchAndCompletion(t *testing.T) {
	store := pluginAssetCursorStore{items: map[string]pluginAssetCursorEntry{}}
	binding := pluginAssetCursorBinding{UserID: "user", ActorID: "actor", PluginID: "immich", InstanceID: "instance", QueryFingerprint: "query", ConfigFingerprint: "config"}
	state := map[string]any{"page": 2}
	hash, _, err := pluginAssetStateHash(state)
	if err != nil {
		t.Fatal(err)
	}
	cursorID, err := store.create(binding, state, hash)
	if err != nil {
		t.Fatal(err)
	}
	mismatch := binding
	mismatch.QueryFingerprint = "changed-query"
	if _, restartRequired, err := store.resume(cursorID, mismatch); err != nil || !restartRequired {
		t.Fatalf("mismatched query restart=%v err=%v", restartRequired, err)
	}
	if _, restartRequired, err := store.resume(cursorID, binding); err != nil || !restartRequired {
		t.Fatalf("mismatched cursor was not discarded: restart=%v err=%v", restartRequired, err)
	}

	cursorID, err = store.create(binding, state, hash)
	if err != nil {
		t.Fatal(err)
	}
	if restartRequired, err := store.advance(cursorID, binding, hash, nil, "", nil, false); err != nil || restartRequired {
		t.Fatalf("completion restart=%v err=%v", restartRequired, err)
	}
	if _, restartRequired, err := store.resume(cursorID, binding); err != nil || !restartRequired {
		t.Fatalf("completed cursor remained available: restart=%v err=%v", restartRequired, err)
	}
}

func TestPluginAssetCursorBindingIncludesAuthentication(t *testing.T) {
	request := pluginAssetLibraryActionInput{Action: "candidates", TakenAfter: "2026-01-01T00:00:00Z"}
	first, err := newPluginAssetCursorBinding("user", "actor", "immich", "instance", request, map[string]any{"apiKey": "first"}, map[string]any{"url": "https://example.test"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := newPluginAssetCursorBinding("user", "actor", "immich", "instance", request, map[string]any{"apiKey": "second"}, map[string]any{"url": "https://example.test"})
	if err != nil {
		t.Fatal(err)
	}
	if first.ConfigFingerprint == second.ConfigFingerprint {
		t.Fatal("changing only authentication did not invalidate cursor binding")
	}
}

func TestPluginAssetCursorStoreExpiresAndLimitsEntriesPerUser(t *testing.T) {
	store := pluginAssetCursorStore{items: map[string]pluginAssetCursorEntry{}}
	binding := pluginAssetCursorBinding{UserID: "user", ActorID: "actor", PluginID: "immich", InstanceID: "instance", QueryFingerprint: "query", ConfigFingerprint: "config"}
	state := map[string]any{"page": 2}
	hash, _, err := pluginAssetStateHash(state)
	if err != nil {
		t.Fatal(err)
	}
	firstID, err := store.create(binding, state, hash)
	if err != nil {
		t.Fatal(err)
	}
	store.Lock()
	entry := store.items[firstID]
	entry.ExpiresAt = time.Now().Add(-time.Second)
	store.items[firstID] = entry
	store.Unlock()
	if _, restartRequired, err := store.resume(firstID, binding); err != nil || !restartRequired {
		t.Fatalf("expired cursor restart=%v err=%v", restartRequired, err)
	}

	for index := 0; index < pluginAssetCursorMaxEntriesPerUser+2; index++ {
		binding.QueryFingerprint = fmt.Sprintf("query-%d", index)
		if _, err := store.create(binding, state, hash); err != nil {
			t.Fatal(err)
		}
	}
	store.Lock()
	count := store.userEntryCountLocked("user")
	store.Unlock()
	if count != pluginAssetCursorMaxEntriesPerUser {
		t.Fatalf("user cursor count = %d, want %d", count, pluginAssetCursorMaxEntriesPerUser)
	}
}

func TestPluginAssetStateHashUsesCanonicalMapOrderingAndSizeLimit(t *testing.T) {
	left, _, err := pluginAssetStateHash(map[string]any{"page": 2, "offset": 4})
	if err != nil {
		t.Fatal(err)
	}
	right, _, err := pluginAssetStateHash(map[string]any{"offset": 4, "page": 2})
	if err != nil {
		t.Fatal(err)
	}
	if left != right {
		t.Fatalf("equivalent states hashed differently: %s != %s", left, right)
	}
	if _, _, err := pluginAssetStateHash(map[string]any{"value": strings.Repeat("x", pluginAssetCursorMaxStateBytes)}); err == nil {
		t.Fatal("oversized state was accepted")
	}
}

func TestAssetLibraryCoordinateBoundsExpandsRouteByRadius(t *testing.T) {
	bounds, ok := assetLibraryCoordinateBounds(pluginAssetLibraryActionInput{
		Points: []pluginAssetTrackPoint{
			{Lat: 46.00, Lon: 8.00},
			{Lat: 46.01, Lon: 8.02},
		},
	}, false)

	if !ok {
		t.Fatal("expected coordinate bounds")
	}
	if bounds.minLat >= 46.00 || bounds.maxLat <= 46.01 || bounds.minLon >= 8.00 || bounds.maxLon <= 8.02 {
		t.Fatalf("bounds were not expanded around route: %#v", bounds)
	}
}

func TestAssetPluginDistanceFromStart(t *testing.T) {
	points := []pluginAssetTrackPoint{
		{Lat: 46.000, Lon: 8.000, Distance: 0},
		{Lat: 46.010, Lon: 8.010, Distance: 1500},
		{Lat: 46.020, Lon: 8.020, Distance: 3000},
	}

	if got := assetPluginDistanceFromStart(points, 46.011, 8.011); got != 1500 {
		t.Fatalf("got %v, want 1500", got)
	}
}

func TestAssetPluginDistanceFromStartDefaultsToZeroWithoutPoints(t *testing.T) {
	if got := assetPluginDistanceFromStart(nil, 46.011, 8.011); got != 0 {
		t.Fatalf("got %v, want 0", got)
	}
}

func TestWaypointPhotoClusterGroupsNearbyPhotos(t *testing.T) {
	clusters := clusterWaypointPhotos(
		[]waypointClusterPhoto{
			{ID: "a", Lat: 46.00000, Lon: 8.00000},
			{ID: "b", Lat: 46.00005, Lon: 8.00005},
		},
		nil,
		waypointMergeSettings{Enabled: true, Radius: 50},
	)

	if len(clusters) != 1 {
		t.Fatalf("got %d clusters, want 1", len(clusters))
	}
	if len(clusters[0].Photos) != 2 {
		t.Fatalf("got %d photos, want 2", len(clusters[0].Photos))
	}
}

func TestWaypointPhotoClusterUsesExistingWaypoint(t *testing.T) {
	clusters := clusterWaypointPhotos(
		[]waypointClusterPhoto{{ID: "asset1", Lat: 46.00005, Lon: 8.00005}},
		[]waypointClusterWaypoint{{ID: "wp1", Lat: 46.00000, Lon: 8.00000}},
		waypointMergeSettings{Enabled: true, Radius: 50},
	)

	if len(clusters) != 1 {
		t.Fatalf("got %d clusters, want 1", len(clusters))
	}
	if clusters[0].Waypoint != "wp1" {
		t.Fatalf("got waypoint %q, want wp1", clusters[0].Waypoint)
	}
	if len(clusters[0].Photos) != 1 || clusters[0].Photos[0] != "asset1" {
		t.Fatalf("unexpected cluster photos: %#v", clusters[0].Photos)
	}
}

func TestAssetPluginPhotosForClusterPreservesClusterOrder(t *testing.T) {
	photos := map[string]pluginsystem.Photo{
		"a": {ExternalID: "a"},
		"b": {ExternalID: "b"},
	}

	got := assetPluginPhotosForCluster([]string{"b", "a"}, photos)
	if len(got) != 2 {
		t.Fatalf("got %d photos, want 2", len(got))
	}
	if got[0].ExternalID != "b" || got[1].ExternalID != "a" {
		t.Fatalf("unexpected order: %#v", got)
	}
}

func TestAssetPluginMaxWaypointsReadsPluginConfig(t *testing.T) {
	cases := []struct {
		name   string
		config map[string]any
		want   int
	}{
		{"default", map[string]any{}, defaultAssetPluginMaxWaypoints},
		{"plugin section", map[string]any{"plugin": map[string]any{"maxWaypoints": "7"}}, 7},
		{"flat config fallback", map[string]any{"maxWaypoints": 3}, 3},
		{"zero disables limit", map[string]any{"plugin": map[string]any{"maxWaypoints": 0}}, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := assetPluginMaxWaypoints(tc.config); got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestAssetPluginPhotoImportLimitsOnlyWhenEnforced(t *testing.T) {
	hostConfig := map[string]any{
		"maxPhotosPerTrail":     9,
		"maxPhotosPerWaypoint":  4,
		"maxPhotosPerSummitLog": 7,
	}

	if got := assetPluginPhotoImportLimits(hostConfig, false); got != nil {
		t.Fatalf("manual import got limits %#v, want nil", got)
	}

	got := assetPluginPhotoImportLimits(hostConfig, true)
	if got == nil {
		t.Fatal("auto attach got nil limits")
	}
	if got.MaxPhotosPerTrail != 9 || got.MaxPhotosPerWaypoint != 4 || got.MaxPhotosPerSummitLog != 7 {
		t.Fatalf("unexpected limits: %#v", got)
	}
}

func TestLimitAssetPluginWaypointClustersLimitsNewWaypointsOnly(t *testing.T) {
	clusters := []waypointPhotoCluster{
		{Waypoint: "wp1", Photos: []string{"existing-a"}},
		{Photos: []string{"standalone"}},
		{Photos: []string{"new-a"}, Count: 1, Lat: 46.0, Lon: 8.0},
		{Photos: []string{"new-b"}, Count: 1, Lat: 46.1, Lon: 8.1},
		{Waypoint: "wp2", Photos: []string{"existing-b"}},
		{Photos: []string{"new-c"}, Count: 1, Lat: 46.2, Lon: 8.2},
	}

	limited := limitAssetPluginWaypointClusters(clusters, 2)

	if len(limited) != 5 {
		t.Fatalf("got %d clusters, want 5: %#v", len(limited), limited)
	}
	if limited[0].Waypoint != "wp1" || limited[1].Photos[0] != "standalone" || limited[2].Photos[0] != "new-a" || limited[3].Photos[0] != "new-b" || limited[4].Waypoint != "wp2" {
		t.Fatalf("unexpected limited clusters: %#v", limited)
	}
}

func TestLimitAssetPluginWaypointClustersZeroIsUnlimited(t *testing.T) {
	clusters := []waypointPhotoCluster{
		{Photos: []string{"new-a"}, Count: 1, Lat: 46.0, Lon: 8.0},
		{Photos: []string{"new-b"}, Count: 1, Lat: 46.1, Lon: 8.1},
	}

	limited := limitAssetPluginWaypointClusters(clusters, 0)

	if len(limited) != len(clusters) {
		t.Fatalf("got %d clusters, want %d", len(limited), len(clusters))
	}
}

func TestAssetPluginClusterCreatesWaypointRequiresCoordinates(t *testing.T) {
	if assetPluginClusterCreatesWaypoint(waypointPhotoCluster{Photos: []string{"standalone"}}) {
		t.Fatal("standalone photo cluster without coordinates should not create a waypoint")
	}
	if !assetPluginClusterCreatesWaypoint(waypointPhotoCluster{Photos: []string{"geo"}, Count: 1, Lat: 46.0, Lon: 8.0}) {
		t.Fatal("geotagged photo cluster should create a waypoint")
	}
}

func TestAssetPluginClusterHasWaypointTargetRejectsStandaloneCluster(t *testing.T) {
	if assetPluginClusterHasWaypointTarget(waypointPhotoCluster{Photos: []string{"standalone"}}) {
		t.Fatal("standalone photo cluster without coordinates should not have a waypoint target")
	}
	if !assetPluginClusterHasWaypointTarget(waypointPhotoCluster{Waypoint: "wp1", Photos: []string{"existing"}}) {
		t.Fatal("existing waypoint cluster should have a waypoint target")
	}
	if !assetPluginClusterHasWaypointTarget(waypointPhotoCluster{Photos: []string{"geo"}, Count: 1, Lat: 46.0, Lon: 8.0}) {
		t.Fatal("geotagged photo cluster should have a waypoint target")
	}
}

func TestAssetPluginWaypointForClusterRejectsStandaloneCluster(t *testing.T) {
	waypoint, created, err := assetPluginWaypointForCluster(
		context.Background(),
		nil,
		"actor",
		"trail",
		waypointPhotoCluster{Photos: []string{"standalone"}},
		nil,
		waypointMergeSettings{},
	)

	if err == nil {
		t.Fatal("expected error for standalone cluster without coordinates")
	}
	if created {
		t.Fatal("standalone cluster should not create a waypoint")
	}
	if waypoint != nil {
		t.Fatalf("got waypoint %#v, want nil", waypoint)
	}
}

func TestResolveWaypointNameUsesOverpassPOI(t *testing.T) {
	reverseCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/interpreter":
			if !strings.Contains(r.URL.Query().Get("data"), "around:50") {
				t.Fatalf("expected overpass radius in query, got %q", r.URL.Query().Get("data"))
			}
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"elements":[{"type":"node","id":1,"lat":46,"lon":8,"tags":{"name":"Aussichtspunkt","tourism":"viewpoint"}}]}`))
		case "/reverse":
			reverseCalled = true
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"name":"Fallback"}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("OVERPASS_API_URL", server.URL)
	t.Setenv("NOMINATIM_URL", server.URL)
	withWaypointNameHTTPClient(t, server.Client())

	name, err := resolveWaypointName(context.Background(), nil, 46, 8, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "Aussichtspunkt" {
		t.Fatalf("got %q, want Aussichtspunkt", name)
	}
	if reverseCalled {
		t.Fatal("reverse geocoding should not be called when overpass finds a POI")
	}
}

func TestResolveWaypointNameFallsBackToNominatim(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/interpreter":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"elements":[]}`))
		case "/reverse":
			if r.Header.Get("User-Agent") == "" {
				t.Fatal("expected user agent on nominatim request")
			}
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"name":"","address":{"road":"Ridge Trail"},"display_name":"12, Ridge Trail, Exampletown"}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("OVERPASS_API_URL", server.URL)
	t.Setenv("NOMINATIM_URL", server.URL)
	withWaypointNameHTTPClient(t, server.Client())

	name, err := resolveWaypointName(context.Background(), nil, 46, 8, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "Ridge Trail" {
		t.Fatalf("got %q, want Ridge Trail", name)
	}
}

func TestResolveWaypointNameFallsBackToLowerNominatimZoom(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/interpreter":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"elements":[]}`))
		case "/reverse":
			w.Header().Set("content-type", "application/json")
			if r.URL.Query().Get("zoom") == "18" {
				_, _ = w.Write([]byte(`{"error":"Unable to geocode"}`))
				return
			}
			_, _ = w.Write([]byte(`{"name":"","address":{"village":"Example Village"},"display_name":"Example Village"}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("OVERPASS_API_URL", server.URL)
	t.Setenv("NOMINATIM_URL", server.URL)
	withWaypointNameHTTPClient(t, server.Client())

	name, err := resolveWaypointName(context.Background(), nil, 46, 8, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "Example Village" {
		t.Fatalf("got %q, want Example Village", name)
	}
}

func TestResolveWaypointNameFallsBackToCoordinates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/interpreter":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"elements":[]}`))
		case "/reverse":
			w.Header().Set("content-type", "application/json")
			_, _ = w.Write([]byte(`{"error":"Unable to geocode"}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	t.Setenv("OVERPASS_API_URL", server.URL)
	t.Setenv("NOMINATIM_URL", server.URL)
	withWaypointNameHTTPClient(t, server.Client())

	name, err := resolveWaypointName(context.Background(), nil, 46.123456, 8.654321, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "46.12346, 8.65432" {
		t.Fatalf("got %q, want coordinate fallback", name)
	}
}

func withWaypointNameHTTPClient(t *testing.T, client *http.Client) {
	t.Helper()
	oldClient := waypointNameHTTPClient
	oldLastNominatimCall := waypointNameLastNominatimCall
	waypointNameHTTPClient = client
	waypointNameLastNominatimCall = time.Time{}
	t.Cleanup(func() {
		waypointNameHTTPClient = oldClient
		waypointNameLastNominatimCall = oldLastNominatimCall
	})
}

func newAssetMaintenanceRecord(values map[string]any) *core.Record {
	collection := core.NewBaseCollection("assets")
	collection.Fields.Add(
		&core.TextField{Name: "file"},
		&core.TextField{Name: "external_provider"},
		&core.TextField{Name: "external_id"},
		&core.DateField{Name: "taken_at"},
		&core.JSONField{Name: "metadata"},
	)

	record := core.NewRecord(collection)
	for key, value := range values {
		record.Set(key, value)
	}
	return record
}
