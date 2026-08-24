package routes

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"testing"
	"time"

	"pocketbase/pluginsystem"

	"github.com/pocketbase/pocketbase/core"
)

func TestPreparedRoundTripCallRefreshesRejectedProfile(t *testing.T) {
	previousCache := routingPreparedProfiles
	previousPrepareCaller := routingProfilePreparePluginCaller
	previousRoundTripCaller := routingRoundTripPluginCaller
	routingPreparedProfiles = newRoutingPreparedProfileCache(time.Minute, 8)
	t.Cleanup(func() {
		routingPreparedProfiles = previousCache
		routingProfilePreparePluginCaller = previousPrepareCaller
		routingRoundTripPluginCaller = previousRoundTripCaller
	})

	runtime := routingEngineRuntime{
		Plugin: pluginsystem.LocalPlugin{Manifest: pluginsystem.Manifest{
			ID: "round-trip-test", Version: "1",
			Capabilities: []pluginsystem.CapabilityManifest{{Name: "profile_prepare", Version: "v1", Export: "profile_prepare_v1"}},
		}},
		Capability: pluginsystem.CapabilityManifest{Name: "round_trip", Version: "v1", Export: "round_trip_v1"},
		Request: pluginRoutingRouteRequest{
			Mode: "foot", Profile: pluginRoutingProfile{Key: "pedestrian", PreparedKey: "stale-profile"},
		},
	}
	fingerprint, err := routingPreparedProfileFingerprint(runtime)
	if err != nil {
		t.Fatalf("prepared profile fingerprint: %v", err)
	}
	runtime.PreparedProfileFingerprint = fingerprint
	routingPreparedProfiles.entries[fingerprint] = routingPreparedProfileEntry{
		PreparedKey: "stale-profile", ExpiresAt: time.Now().Add(time.Minute),
	}

	type observedPreparation struct {
		capability string
		request    pluginRoutingProfilePrepareRequest
	}
	prepareCalls := make(chan observedPreparation, 2)
	routingProfilePreparePluginCaller = func(_ context.Context, _ pluginsystem.LocalPlugin, capability pluginsystem.CapabilityManifest, _ *core.Record, _ map[string]any, _ map[string]any, request pluginRoutingProfilePrepareRequest) (pluginRoutingProfilePrepareOutput, error) {
		prepareCalls <- observedPreparation{capability: capability.Name, request: request}
		return pluginRoutingProfilePrepareOutput{PreparedKey: "fresh-profile"}, nil
	}
	calledKeys := []string{}
	routingRoundTripPluginCaller = func(_ context.Context, _ pluginsystem.LocalPlugin, _ pluginsystem.CapabilityManifest, _ *core.Record, _ map[string]any, _ map[string]any, request pluginRoutingRoundTripRequest) (pluginRoutingRouteOutput, error) {
		calledKeys = append(calledKeys, request.Profile.PreparedKey)
		if request.Profile.PreparedKey == "stale-profile" {
			return pluginRoutingRouteOutput{Error: &pluginsystem.PluginError{Code: "unsupported_profile", Message: "expired"}}, nil
		}
		return pluginRoutingRouteOutput{}, nil
	}

	request := pluginRoutingRoundTripRequest{Profile: runtime.Request.Profile}
	output, err := callPreparedRoutingRoundTripPlugin(context.Background(), runtime, request)
	if err != nil || output.Error != nil {
		t.Fatalf("round-trip call after refresh = %#v, %v", output, err)
	}
	if len(prepareCalls) != 1 || len(calledKeys) != 2 || calledKeys[0] != "stale-profile" || calledKeys[1] != "fresh-profile" {
		t.Fatalf("round-trip refresh used preparations=%d keys=%v", len(prepareCalls), calledKeys)
	}
	preparation := <-prepareCalls
	if preparation.capability != "profile_prepare" || preparation.request.Profile.PreparedKey != "" {
		t.Fatalf("profile refresh request = %s %#v", preparation.capability, preparation.request)
	}
}

func TestRoundTripRequestBounds(t *testing.T) {
	valid := pluginRoutingRoundTripRequest{
		Start: pluginRoutingAnchor{Lat: 47, Lon: 8}, TargetDistance: 1000,
	}
	if err := validateRoutingRoundTripRequest(valid); err != nil {
		t.Fatalf("expected lower distance boundary to be valid: %v", err)
	}
	valid.TargetDistance = 300000
	if err := validateRoutingRoundTripRequest(valid); err != nil {
		t.Fatalf("expected upper distance boundary to be valid: %v", err)
	}
	for _, distance := range []float64{999, 300001, math.NaN()} {
		invalid := valid
		invalid.TargetDistance = distance
		if err := validateRoutingRoundTripRequest(invalid); err == nil {
			t.Fatalf("expected distance %v to be rejected", distance)
		}
	}
}

func TestMaterializeRoundTripUsesUniqueAnchorsCyclicSegmentsAndProvenance(t *testing.T) {
	points := [][]float64{
		{47.0, 8.0},
		{47.01, 8.02},
		{47.02, 8.04},
		{47.01, 8.06},
		{47.0, 8.04},
		{47.0, 8.0},
	}
	geometry := testGeometry(points)
	candidate := pluginRoutingCandidate{
		Geometry: &geometry,
		Summary:  pluginRoutingSummary{Distance: 12000, Duration: 3600},
		SuggestedAnchors: []pluginRoutingAnchor{
			{Lat: 47.02, Lon: 8.04},
			{Lat: 47.0, Lon: 8.04},
		},
	}
	roundTripRequest := pluginRoutingRoundTripRequest{
		Start: pluginRoutingAnchor{Lat: 47, Lon: 8}, TargetDistance: 10000, Seed: "test",
	}
	direction := 270.0
	roundTripRequest.Direction = &direction
	resolved := pluginRoutingRouteRequest{
		RoutingMode: "round_trip", Category: "Hiking", Mode: "foot",
		Profile:     pluginRoutingProfile{Key: "hiking-mountain", Kind: "builtin"},
		Preferences: map[string]any{"hillPreference": 0.5},
	}
	client := cloneRoutingRouteRequest(resolved)

	materialized, err := materializeRoutingRoundTripCandidate(candidate, roundTripRequest, resolved, client, testRoutingPlugin, nil, "rt_test", 0)
	if err != nil {
		t.Fatalf("materialize round trip: %v", err)
	}
	if len(materialized.SnappedAnchors) != 3 {
		t.Fatalf("expected start and two unique synthetic anchors, got %#v", materialized.SnappedAnchors)
	}
	if materialized.SnappedAnchors[0] != roundTripRequest.Start {
		t.Fatalf("expected original start anchor first, got %#v", materialized.SnappedAnchors[0])
	}
	if len(materialized.Segments) != len(materialized.SnappedAnchors) {
		t.Fatalf("expected one cyclic segment per unique anchor, got %d", len(materialized.Segments))
	}
	segmentDistance := 0.0
	segmentDuration := 0.0
	for _, segment := range materialized.Segments {
		segmentDistance += segment.Distance
		segmentDuration += segment.Duration
	}
	if math.Abs(segmentDistance-materialized.Summary.Distance) > 0.001 || math.Abs(segmentDuration-materialized.Summary.Duration) > 0.001 {
		t.Fatalf("segment totals differ from summary: distance=%v duration=%v summary=%#v", segmentDistance, segmentDuration, materialized.Summary)
	}
	last := materialized.Segments[len(materialized.Segments)-1]
	if last.ToAnchor != 0 {
		t.Fatalf("expected final segment to close to the first anchor, got %#v", last)
	}
	for _, segment := range materialized.Segments {
		if segment.Provenance == nil || segment.Provenance.Source != "round_trip" || segment.Provenance.RouteTopology != "closed_loop" || segment.Provenance.RoundTripRequestID == "" {
			t.Fatalf("expected round-trip provenance on every segment, got %#v", segment.Provenance)
		}
		if segment.Provenance.RoundTripSeed != "test" || segment.Provenance.RoundTripDirection == nil || *segment.Provenance.RoundTripDirection != 270 {
			t.Fatalf("expected direction and seed in persisted provenance, got %#v", segment.Provenance)
		}
		if segment.Provenance.RoutingMode != "" {
			t.Fatalf("dedicated round-trip generation must not claim segment/via mode: %#v", segment.Provenance)
		}
	}
	if materialized.RoundTrip == nil || materialized.RoundTrip.Seed != "test" || materialized.RoundTrip.RequestID == "" {
		t.Fatalf("expected round-trip metadata, got %#v", materialized.RoundTrip)
	}
}

func TestRoundTripSyntheticAnchorLimitAndClosureValidation(t *testing.T) {
	points := make([][]float64, 12)
	suggestions := make([]pluginRoutingAnchor, 0, 10)
	for index := range points {
		points[index] = []float64{47 + float64(index)*0.001, 8 + float64(index)*0.001}
		if index > 0 && index < len(points)-1 {
			suggestions = append(suggestions, pluginRoutingAnchor{Lat: points[index][0], Lon: points[index][1]})
		}
	}
	points[len(points)-1] = []float64{47, 8}
	geometry := testGeometry(points)
	request := pluginRoutingRoundTripRequest{Start: pluginRoutingAnchor{Lat: 47, Lon: 8}, TargetDistance: 10000}
	resolved := pluginRoutingRouteRequest{RoutingMode: "round_trip", Profile: pluginRoutingProfile{Key: "test"}}
	materialized, err := materializeRoutingRoundTripCandidate(
		pluginRoutingCandidate{Geometry: &geometry, Summary: pluginRoutingSummary{Distance: 10000}, SuggestedAnchors: suggestions},
		request, resolved, resolved, testRoutingPlugin, nil, "rt_test", 0,
	)
	if err != nil {
		t.Fatalf("materialize round trip: %v", err)
	}
	if synthetic := len(materialized.SnappedAnchors) - 1; synthetic != routingRoundTripMaxSynthetic {
		t.Fatalf("expected synthetic anchor cap %d, got %d", routingRoundTripMaxSynthetic, synthetic)
	}

	openPoints := append([][]float64(nil), points...)
	openPoints[len(openPoints)-1] = []float64{47.2, 8.2}
	openGeometry := testGeometry(openPoints)
	_, err = materializeRoutingRoundTripCandidate(
		pluginRoutingCandidate{Geometry: &openGeometry, Summary: pluginRoutingSummary{Distance: 10000}},
		request, resolved, resolved, testRoutingPlugin, nil, "rt_test", 0,
	)
	if err == nil {
		t.Fatal("expected open geometry to be rejected")
	}
}

func TestRoundTripUsesSnappedProviderStartWithoutChangingGeometry(t *testing.T) {
	points := [][]float64{{47, 8}, {47.01, 8.01}, {47.01, 7.99}, {47, 8}}
	geometry := testGeometry(points)
	request := pluginRoutingRoundTripRequest{
		Start: pluginRoutingAnchor{Lat: 47.0005, Lon: 8}, TargetDistance: 10000,
	}
	resolved := pluginRoutingRouteRequest{RoutingMode: "round_trip", Profile: pluginRoutingProfile{Key: "test"}}
	materialized, err := materializeRoutingRoundTripCandidate(
		pluginRoutingCandidate{Geometry: &geometry, Summary: pluginRoutingSummary{Distance: 10000}},
		request, resolved, resolved, testRoutingPlugin, nil, "rt_test", 0,
	)
	if err != nil {
		t.Fatalf("materialize snapped start: %v", err)
	}
	if materialized.Geometry.Coordinates != geometry.Coordinates {
		t.Fatal("host changed provider round-trip geometry")
	}
	if got := materialized.SnappedAnchors[0]; got.Lat != 47 || got.Lon != 8 {
		t.Fatalf("expected provider-snapped first anchor, got %#v", got)
	}

	request.Start.Lat = 47.002
	_, err = materializeRoutingRoundTripCandidate(
		pluginRoutingCandidate{Geometry: &geometry, Summary: pluginRoutingSummary{Distance: 10000}},
		request, resolved, resolved, testRoutingPlugin, nil, "rt_test", 0,
	)
	if err == nil {
		t.Fatal("expected round trip starting more than 100 m away to be rejected")
	}

	request.Start = pluginRoutingAnchor{Lat: 47, Lon: 8}
	openWithinOldTolerance := [][]float64{{47, 8}, {47.01, 8.01}, {47.01, 7.99}, {47.0002, 8}}
	openGeometry := testGeometry(openWithinOldTolerance)
	_, err = materializeRoutingRoundTripCandidate(
		pluginRoutingCandidate{Geometry: &openGeometry, Summary: pluginRoutingSummary{Distance: 10000}},
		request, resolved, resolved, testRoutingPlugin, nil, "rt_test", 0,
	)
	if err == nil {
		t.Fatal("expected a round trip with a roughly 22 m closure gap to be rejected")
	}
}

func TestRoundTripMetadataValidation(t *testing.T) {
	direction359 := 359
	if err := validateRoutingRoundTripMetadata(&pluginRoutingRoundTripMetadata{
		Direction: &direction359, Attempts: 1, Tolerance: 0.1,
	}); err != nil {
		t.Fatalf("expected valid metadata: %v", err)
	}
	if err := validateRoutingRoundTripMetadata(nil); err != nil {
		t.Fatalf("expected omitted metadata to remain optional: %v", err)
	}
	directionNegative := -1
	direction360 := 360
	cases := []*pluginRoutingRoundTripMetadata{
		{Direction: &directionNegative, Attempts: 1, Tolerance: 0.1},
		{Direction: &direction360, Attempts: 1, Tolerance: 0.1},
		{Attempts: 0, Tolerance: 0.1},
		{Attempts: routingRoundTripMaxAttempts + 1, Tolerance: 0.1},
		{Attempts: 1, Tolerance: -0.1},
		{Attempts: 1, Tolerance: 1.1},
		{Attempts: 1, Tolerance: math.NaN()},
	}
	for _, metadata := range cases {
		if err := validateRoutingRoundTripMetadata(metadata); err == nil {
			t.Fatalf("expected invalid metadata to be rejected: %#v", metadata)
		}
	}
}

func TestRoundTripFallbackSplitsAreDistanceDistributed(t *testing.T) {
	points := [][]float64{
		{47, 8}, {47, 8.01}, {47, 8.02}, {47, 8.03}, {47, 8.04},
		{47, 8.05}, {47, 8.04}, {47, 8.03}, {47, 8.02}, {47, 8.01}, {47, 8},
	}
	splits := routingRoundTripDistanceSplits(points, routingRoundTripFallbackAnchors)
	if len(splits) != routingRoundTripFallbackAnchors {
		t.Fatalf("expected %d fallback splits, got %#v", routingRoundTripFallbackAnchors, splits)
	}
	for index := 1; index < len(splits); index++ {
		if splits[index] <= splits[index-1] {
			t.Fatalf("fallback splits are not ordered: %#v", splits)
		}
	}
}

func TestRoundTripRequestIDsAreUnique(t *testing.T) {
	first := newRoutingRoundTripRequestID()
	second := newRoutingRoundTripRequestID()
	if first == second || len(first) != len("rt_")+16 || len(second) != len("rt_")+16 {
		t.Fatalf("expected unique opaque request ids, got %q and %q", first, second)
	}
}

func TestRoundTripHandlerInvokesDedicatedCapability(t *testing.T) {
	app, auth := newRoutingHandlerTestApp(t, map[string]bool{"route-a": true}, "segment")
	addRoutingTestCapability(t, app, "route-a", pluginsystem.CapabilityManifest{
		Name: "round_trip", Version: "v1", Export: "round_trip_v1",
	})
	previousCaller := routingRoundTripPluginCaller
	routingRoundTripPluginCaller = func(_ context.Context, plugin pluginsystem.LocalPlugin, capability pluginsystem.CapabilityManifest, _ *core.Record, _ map[string]any, _ map[string]any, request pluginRoutingRoundTripRequest) (pluginRoutingRouteOutput, error) {
		if plugin.Manifest.ID != "route-a" || capability.Export != "round_trip_v1" || request.TargetDistance != 10000 {
			t.Fatalf("unexpected round-trip invocation: plugin=%s capability=%#v request=%#v", plugin.Manifest.ID, capability, request)
		}
		geometry := testGeometry([][]float64{{47, 8}, {47.01, 8.02}, {47.02, 8}, {47, 8}})
		return pluginRoutingRouteOutput{Candidates: []pluginRoutingCandidate{{
			Geometry: &geometry, Summary: pluginRoutingSummary{Distance: 9800, Duration: 3000},
		}}}, nil
	}
	t.Cleanup(func() { routingRoundTripPluginCaller = previousCaller })

	payload, _ := json.Marshal(map[string]any{
		"pluginId": "route-a", "start": map[string]float64{"lat": 47, "lon": 8},
		"targetDistance": 10000, "mode": "foot", "profile": map[string]any{"key": "pedestrian"},
	})
	status, body := callRoutingHandler(t, app, auth, PluginSystemRoutingRoundTrip, string(payload))
	if status != http.StatusOK {
		t.Fatalf("expected round-trip success, got %d: %s", status, body)
	}
	var output pluginRoutingRouteOutput
	if err := json.Unmarshal([]byte(body), &output); err != nil {
		t.Fatalf("decode round-trip response: %v", err)
	}
	if len(output.Candidates) != 1 || len(output.Candidates[0].Segments) != len(output.Candidates[0].SnappedAnchors) {
		t.Fatalf("expected materialized cyclic candidate, got %#v", output)
	}
}

func TestRoundTripHandlerFiltersPreferencesByManifestAndMode(t *testing.T) {
	app, auth := newRoutingHandlerTestApp(t, map[string]bool{"route-a": true}, "segment")
	addRoutingTestCapability(t, app, "route-a", pluginsystem.CapabilityManifest{
		Name: "round_trip", Version: "v1", Export: "round_trip_v1",
	})
	previousCaller := routingRoundTripPluginCaller
	routingRoundTripPluginCaller = func(_ context.Context, _ pluginsystem.LocalPlugin, _ pluginsystem.CapabilityManifest, _ *core.Record, _ map[string]any, _ map[string]any, request pluginRoutingRoundTripRequest) (pluginRoutingRouteOutput, error) {
		if request.Preferences["hillPreference"] != 0.7 || request.Preferences["providerKnob"] != true {
			t.Fatalf("supported/provider round-trip preferences were removed: %#v", request.Preferences)
		}
		if _, exists := request.Preferences["speedPreference"]; exists {
			t.Fatalf("undeclared round-trip preference reached plugin: %#v", request.Preferences)
		}
		geometry := testGeometry([][]float64{{47, 8}, {47.01, 8.02}, {47.02, 8}, {47, 8}})
		return pluginRoutingRouteOutput{Candidates: []pluginRoutingCandidate{{
			Geometry: &geometry, Summary: pluginRoutingSummary{Distance: 9800, Duration: 3000},
		}}}, nil
	}
	t.Cleanup(func() { routingRoundTripPluginCaller = previousCaller })

	payload, _ := json.Marshal(map[string]any{
		"pluginId": "route-a", "start": map[string]float64{"lat": 47, "lon": 8},
		"targetDistance": 10000, "mode": "foot", "profile": map[string]any{"key": "pedestrian"},
		"preferences": map[string]any{"hillPreference": 0.7, "speedPreference": 0.5, "providerKnob": true},
	})
	status, body := callRoutingHandler(t, app, auth, PluginSystemRoutingRoundTrip, string(payload))
	if status != http.StatusOK {
		t.Fatalf("round-trip preference response = %d %s", status, body)
	}
}
