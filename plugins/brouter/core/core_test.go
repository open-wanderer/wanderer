package core

import (
	"encoding/json"
	"errors"
	"math"
	"testing"
)

func TestCandidateFromFeatureSegmentsAndPolylineCompliance(t *testing.T) {
	request := RouteRequest{
		RoutingMode: "segment",
		Anchors: []Anchor{
			{Lat: 47.0, Lon: 8.0},
			{Lat: 47.1, Lon: 8.1},
			{Lat: 47.2, Lon: 8.2},
		},
	}
	feature := Feature{
		Properties: Properties{
			TrackLength: "300",
			TotalTime:   "120",
			Ascend:      "12",
			Times:       []float64{0, 20, 60, 90, 120},
		},
		Geometry: LineString{
			Type: "LineString",
			Coordinates: [][]float64{
				{8.0, 47.0, 400},
				{8.05, 47.05, 405},
				{8.1, 47.1, 410},
				{8.15, 47.15, 415},
				{8.2, 47.2, 420},
			},
		},
	}

	candidate, err := CandidateFromFeature(request, "trekking", feature, testParseFloat)
	if err != nil {
		t.Fatalf("candidate from BRouter feature: %v", err)
	}
	if candidate.Geometry == nil || candidate.Geometry.Format != EncodedPolylineFormat || candidate.Geometry.Precision != EncodedPolylinePrecision || candidate.Geometry.Coordinates == "" {
		t.Fatalf("expected canonical route geometry, got %#v", candidate.Geometry)
	}
	if len(candidate.Segments) != 2 {
		t.Fatalf("expected one segment per adjacent anchor pair, got %d", len(candidate.Segments))
	}
	for i, segment := range candidate.Segments {
		if segment.FromAnchor != i || segment.ToAnchor != i+1 {
			t.Fatalf("unexpected segment anchor indexes at %d: %#v", i, segment)
		}
		if segment.Geometry.Format != EncodedPolylineFormat || segment.Geometry.Precision != EncodedPolylinePrecision || segment.Geometry.Coordinates == "" {
			t.Fatalf("expected canonical segment geometry at %d: %#v", i, segment.Geometry)
		}
		if segment.Distance <= 0 || segment.Duration <= 0 {
			t.Fatalf("expected positive segment metrics at %d: %#v", i, segment)
		}
	}
	if candidate.Elevation == nil || len(candidate.Elevation.Heights) != 5 || candidate.Elevation.Heights[2] != 410 {
		t.Fatalf("expected BRouter Z coordinates to become candidate heights, got %#v", candidate.Elevation)
	}
	if len(candidate.SnappedAnchors) != len(request.Anchors) {
		t.Fatalf("expected snapped anchors for every request anchor, got %#v", candidate.SnappedAnchors)
	}
}

func TestRoundTripCalibrationAndSeededDirection(t *testing.T) {
	if radius := InitialRoundTripRadius(20000); math.Abs(radius-(20000/(2*math.Pi))) > 0.001 {
		t.Fatalf("expected circular initial radius, got %v", radius)
	}
	if radius := NextRoundTripRadius(5000, 20000, 25000); math.Abs(radius-4000) > 0.001 {
		t.Fatalf("expected proportional radius correction, got %v", radius)
	}
	if !RoundTripWithinTolerance(20000, 18000) {
		t.Fatal("expected lower tolerance boundary to be accepted")
	}
	if RoundTripWithinTolerance(20000, 17999) {
		t.Fatal("expected distance outside tolerance to be rejected")
	}
	first := RoundTripDirection(nil, "stable-seed")
	if first < 0 || first > 359 || first != RoundTripDirection(nil, "stable-seed") {
		t.Fatalf("expected stable seeded direction, got %d", first)
	}
	direction := 359.6
	if got := RoundTripDirection(&direction, ""); got != 0 {
		t.Fatalf("expected rounded normalized direction, got %d", got)
	}
	if got := RoundTripDirection(nil, ""); got != -1 {
		t.Fatalf("expected provider-random direction, got %d", got)
	}
}

func TestCalibrateRoundTripCorrectsRadiusAndStopsWithinTolerance(t *testing.T) {
	distances := []float64{30000, 20500}
	radii := []float64{}
	candidate, err := CalibrateRoundTrip(20000, func(radius float64, attempt int) (RouteCandidate, error) {
		radii = append(radii, radius)
		return RouteCandidate{
			Summary:   Summary{Distance: distances[attempt-1]},
			RoundTrip: &RoundTripMetadata{},
		}, nil
	})
	if err != nil {
		t.Fatalf("calibrate round trip: %v", err)
	}
	if len(radii) != 2 || math.Abs(radii[1]-NextRoundTripRadius(radii[0], 20000, 30000)) > 0.001 {
		t.Fatalf("unexpected calibration radii: %#v", radii)
	}
	if candidate.Summary.Distance != 20500 || candidate.RoundTrip == nil || candidate.RoundTrip.Attempts != 2 {
		t.Fatalf("unexpected calibrated candidate: %#v", candidate)
	}
	if len(candidate.Warnings) != 0 {
		t.Fatalf("unexpected calibration warnings: %#v", candidate.Warnings)
	}
}

func TestCalibrateRoundTripReturnsBestCandidateAfterPartialFailure(t *testing.T) {
	candidate, err := CalibrateRoundTrip(20000, func(_ float64, attempt int) (RouteCandidate, error) {
		if attempt == 2 {
			return RouteCandidate{}, errors.New("provider failed")
		}
		return RouteCandidate{
			Summary:   Summary{Distance: 25000},
			RoundTrip: &RoundTripMetadata{},
		}, nil
	})
	if err != nil {
		t.Fatalf("expected partial calibration result, got %v", err)
	}
	if candidate.RoundTrip == nil || candidate.RoundTrip.Attempts != 1 {
		t.Fatalf("failed attempt was counted as successful: %#v", candidate.RoundTrip)
	}
	expected := map[string]bool{
		"round_trip_distance_adjustment_incomplete": true,
		"round_trip_target_tolerance_not_met":       true,
	}
	for _, warning := range candidate.Warnings {
		delete(expected, warning)
	}
	if len(expected) != 0 {
		t.Fatalf("missing partial calibration warnings: %#v", candidate.Warnings)
	}
}

func TestCalibrateRoundTripReturnsInitialFailure(t *testing.T) {
	expected := errors.New("provider failed")
	_, err := CalibrateRoundTrip(20000, func(_ float64, _ int) (RouteCandidate, error) {
		return RouteCandidate{}, expected
	})
	if !errors.Is(err, expected) {
		t.Fatalf("expected initial provider error, got %v", err)
	}
}

func TestRoundTripCandidateFromFeature(t *testing.T) {
	feature := Feature{
		Properties: Properties{TrackLength: "300", TotalTime: "120", Ascend: "12"},
		Geometry: LineString{Type: "LineString", Coordinates: [][]float64{
			{8.0, 47.0, 400}, {8.1, 47.1, 410}, {8.0, 47.0, 400},
		}},
	}
	candidate, err := RoundTripCandidateFromFeature(
		"trekking",
		feature,
		[]Anchor{{Lat: 47.05, Lon: 8.05}},
		300,
		90,
		2,
		testParseFloat,
	)
	if err != nil {
		t.Fatalf("candidate from round-trip feature: %v", err)
	}
	if candidate.Geometry == nil || len(candidate.Segments) != 0 || len(candidate.SuggestedAnchors) != 1 {
		t.Fatalf("unexpected round-trip candidate: %#v", candidate)
	}
	if candidate.RoundTrip == nil || candidate.RoundTrip.Attempts != 2 || candidate.RoundTrip.Direction == nil || *candidate.RoundTrip.Direction != 90 {
		t.Fatalf("missing round-trip metadata: %#v", candidate.RoundTrip)
	}
}

func TestRoundTripCandidateOmitsUnknownRandomDirection(t *testing.T) {
	feature := Feature{
		Properties: Properties{TrackLength: "300"},
		Geometry: LineString{Type: "LineString", Coordinates: [][]float64{
			{8.0, 47.0}, {8.1, 47.1}, {8.0, 47.0},
		}},
	}
	candidate, err := RoundTripCandidateFromFeature("trekking", feature, nil, 300, -1, 1, testParseFloat)
	if err != nil {
		t.Fatalf("candidate from random round trip: %v", err)
	}
	if candidate.RoundTrip == nil || candidate.RoundTrip.Direction != nil {
		t.Fatalf("expected unknown random direction to be omitted, got %#v", candidate.RoundTrip)
	}
}

func TestParseRoundTripFeaturesFiltersProviderEndpoints(t *testing.T) {
	collection := RawFeatureCollection{Features: []json.RawMessage{
		json.RawMessage(`{"type":"Feature","properties":{"track-length":"1000"},"geometry":{"type":"LineString","coordinates":[[8,47],[8.01,47.01],[8,47]]}}`),
		json.RawMessage(`{"type":"Feature","properties":{"name":"from"},"geometry":{"type":"Point","coordinates":[8,47]}}`),
		json.RawMessage(`{"type":"Feature","properties":{"name":"rt1"},"geometry":{"type":"Point","coordinates":[8.01,47.01]}}`),
		json.RawMessage(`{"type":"Feature","properties":{"name":"to_rt"},"geometry":{"type":"Point","coordinates":[8,47]}}`),
	}}
	feature, suggestions, err := ParseRoundTripFeatures(collection.Features, Anchor{Lat: 47, Lon: 8})
	if err != nil {
		t.Fatalf("parse round-trip features: %v", err)
	}
	if len(feature.Geometry.Coordinates) != 3 || len(suggestions) != 1 {
		t.Fatalf("unexpected parsed round-trip features: feature=%#v suggestions=%#v", feature, suggestions)
	}
	if suggestions[0].Lat != 47.01 || suggestions[0].Lon != 8.01 {
		t.Fatalf("unexpected synthetic waypoint: %#v", suggestions[0])
	}
}

func TestParseRoundTripFeaturesRequiresLineString(t *testing.T) {
	_, _, err := ParseRoundTripFeatures([]json.RawMessage{
		json.RawMessage(`{"type":"Feature","properties":{"name":"rt1"},"geometry":{"type":"Point","coordinates":[8.01,47.01]}}`),
	}, Anchor{Lat: 47, Lon: 8})
	if err == nil {
		t.Fatal("expected missing line feature to fail")
	}
}

func testParseFloat(value string) float64 {
	switch value {
	case "300":
		return 300
	case "120":
		return 120
	case "12":
		return 12
	default:
		return 0
	}
}

func TestBRouterPreferenceMappingKeepsUpstreamDefaultsAtNeutralValues(t *testing.T) {
	tests := []struct {
		name        string
		templateKey string
		mode        string
		preferences map[string]any
		numbers     map[string]float64
		booleans    map[string]bool
	}{
		{
			name:        "hike",
			templateKey: TemplateHike,
			mode:        "foot",
			preferences: map[string]any{"hillPreference": NeutralHillPreference, "maxHikingDifficulty": 3.0},
			numbers: map[string]float64{
				"uphillcostvalue":     7,
				"downhillcostvalue":   7,
				"SAC_scale_limit":     3,
				"SAC_scale_preferred": 1,
			},
		},
		{
			name:        "trekking",
			templateKey: TemplateTrekking,
			mode:        "bike",
			preferences: map[string]any{
				"speedPreference":  NeutralTrekkingSpeed,
				"hillPreference":   NeutralHillPreference,
				"roadPreference":   NeutralRoadPreference,
				"avoidBadSurfaces": NeutralTrekkingBadSurfaces,
			},
			numbers:  map[string]float64{"bikerPower": 100, "uphillcost": 0, "unpavedPenalty": 1},
			booleans: map[string]bool{"avoid_unsafe": false, "consider_traffic": false},
		},
		{
			name:        "fastbike",
			templateKey: TemplateFastbike,
			mode:        "bike",
			preferences: map[string]any{
				"speedPreference":  NeutralFastbikeSpeed,
				"hillPreference":   NeutralHillPreference,
				"roadPreference":   0.9,
				"avoidBadSurfaces": NeutralFastbikeBadSurfaces,
			},
			numbers: map[string]float64{"bikerPower": 100, "uphillcost": 0, "unpavedPenalty": 1, "consider_traffic": 0.1},
		},
		{
			name:        "gravel",
			templateKey: TemplateGravel,
			mode:        "bike",
			preferences: map[string]any{
				"speedPreference":  NeutralGravelSpeed,
				"hillPreference":   0.7,
				"roadPreference":   NeutralRoadPreference,
				"avoidBadSurfaces": 0.5,
			},
			numbers: map[string]float64{"bikerPower": 150},
			booleans: map[string]bool{
				"consider_elevation":        false,
				"avoid_steep_inclines":      false,
				"consider_traffic_estimate": false,
				"prefer_unpaved_paths":      false,
				"assume_wet_conditions":     false,
			},
		},
		{
			name:        "mtb",
			templateKey: TemplateMTB,
			mode:        "bike",
			preferences: map[string]any{
				"hillPreference":   0.7,
				"roadPreference":   NeutralRoadPreference,
				"avoidBadSurfaces": NeutralMTBBadSurfaces,
			},
			numbers:  map[string]float64{"hills": 0, "MTB_factor": 0},
			booleans: map[string]bool{"avoid_unsafe": false},
		},
		{
			name:        "car",
			templateKey: TemplateCar,
			mode:        "motor",
			preferences: map[string]any{"speedPreference": 160.0, "avoidBadSurfaces": 0.5},
			numbers:     map[string]float64{"vmax": 160},
			booleans:    map[string]bool{"avoid_unpaved": false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nativeConfig := NativeConfigWithPreferences(tt.templateKey, nil, tt.preferences, tt.mode)
			parameters := nativeConfig["parameters"].(map[string]any)
			for key, expected := range tt.numbers {
				assertFloat(t, parameters[key], expected)
			}
			for key, expected := range tt.booleans {
				assertBool(t, parameters, key, expected)
			}
		})
	}
}

func TestBRouterPreferenceMappingStepsWhereUpstreamHasNoScale(t *testing.T) {
	moderate := NativeConfigWithPreferences(TemplateGravel, nil, map[string]any{
		"hillPreference":   0.5,
		"avoidBadSurfaces": 0.2,
	}, "bike")["parameters"].(map[string]any)
	assertBool(t, moderate, "consider_elevation", true)
	assertBool(t, moderate, "avoid_steep_inclines", false)
	assertBool(t, moderate, "prefer_unpaved_paths", true)

	strict := NativeConfigWithPreferences(TemplateGravel, nil, map[string]any{
		"hillPreference":   0.2,
		"avoidBadSurfaces": 0.9,
	}, "bike")["parameters"].(map[string]any)
	assertBool(t, strict, "consider_elevation", false)
	assertBool(t, strict, "avoid_steep_inclines", true)
	assertBool(t, strict, "assume_wet_conditions", true)

	mtb := NativeConfigWithPreferences(TemplateMTB, nil, map[string]any{
		"hillPreference":   0.2,
		"avoidBadSurfaces": 0,
	}, "bike")["parameters"].(map[string]any)
	assertFloat(t, mtb["hills"], 2)
	assertFloat(t, mtb["MTB_factor"], 1.5)
}

func TestBRouterPreferenceMappingSkipsPreferencesTheBaseCannotHonor(t *testing.T) {
	for _, templateKey := range []string{TemplateHike, TemplateMTB} {
		t.Run(templateKey, func(t *testing.T) {
			nativeConfig := NativeConfigWithPreferences(templateKey, nil, map[string]any{
				"speedPreference": 12,
			}, "")
			if parameters, ok := nativeConfig["parameters"]; ok {
				t.Fatalf("expected no parameters for a base without a kinematic model, got %#v", parameters)
			}
		})
	}
}

func TestBRouterPreferenceMappingInterpolatesAndClampsBikeControls(t *testing.T) {
	nativeConfig := NativeConfigWithPreferences(TemplateTrekking, map[string]any{
		"parameters": map[string]any{"custom": true},
	}, map[string]any{
		"speedPreference":  60,
		"hillPreference":   0,
		"roadPreference":   0.2,
		"avoidBadSurfaces": 1,
	}, "bike")

	parameters := nativeConfig["parameters"].(map[string]any)
	if parameters["custom"] != true {
		t.Fatalf("expected existing native config parameters to survive merge, got %#v", parameters)
	}
	assertFloat(t, parameters["bikerPower"], 300)
	assertFloat(t, parameters["uphillcost"], 120)
	assertFloat(t, parameters["unpavedPenalty"], 5)
	assertBool(t, parameters, "avoid_unsafe", true)
	assertBool(t, parameters, "consider_traffic", true)

	tolerant := NativeConfigWithPreferences(TemplateTrekking, nil, map[string]any{
		"avoidBadSurfaces": 0,
	}, "bike")["parameters"].(map[string]any)
	assertFloat(t, tolerant["unpavedPenalty"], 0)
}

func TestBRouterPreferenceMappingFootControls(t *testing.T) {
	nativeConfig := NativeConfigWithPreferences(TemplateHike, nil, map[string]any{
		"hillPreference":      1,
		"maxHikingDifficulty": 8,
	}, "foot")

	parameters := nativeConfig["parameters"].(map[string]any)
	// The switch stays on: in this profile it also governs the cost of steps.
	assertBool(t, parameters, "consider_elevation", true)
	assertFloat(t, parameters["uphillcostvalue"], 0)
	assertFloat(t, parameters["downhillcostvalue"], 0)
	assertFloat(t, parameters["SAC_scale_limit"], 6)
	// A tolerant limit must not make the profile prefer alpine terrain.
	assertFloat(t, parameters["SAC_scale_preferred"], 3)
}

func TestBRouterPreferenceMappingCustomFootAndMotorControls(t *testing.T) {
	footConfig := NativeConfigWithPreferences("", nil, map[string]any{
		"hillPreference":      0.8,
		"maxHikingDifficulty": 4,
	}, "foot")
	footParameters := footConfig["parameters"].(map[string]any)
	assertBool(t, footParameters, "consider_elevation", true)
	assertFloat(t, footParameters["SAC_scale_limit"], 4)
	assertFloat(t, footParameters["SAC_scale_preferred"], 2)

	motorConfig := NativeConfigWithPreferences("", nil, map[string]any{
		"speedPreference": 320,
	}, "motor")
	motorParameters := motorConfig["parameters"].(map[string]any)
	assertFloat(t, motorParameters["vmax"], 300)

	slowMotorConfig := NativeConfigWithPreferences("", nil, map[string]any{
		"speedPreference": 5,
	}, "motor")
	slowMotorParameters := slowMotorConfig["parameters"].(map[string]any)
	assertFloat(t, slowMotorParameters["vmax"], MinCarSpeed)
}

func TestBRouterPreferenceMappingDoesNotMutateInputNativeConfig(t *testing.T) {
	input := map[string]any{"parameters": map[string]any{"bikerPower": 120.0}}
	_ = NativeConfigWithPreferences("bike-balanced", input, map[string]any{"speedPreference": 50}, "bike")

	parameters := input["parameters"].(map[string]any)
	assertFloat(t, parameters["bikerPower"], 120)
}

func assertBool(t *testing.T, parameters map[string]any, key string, expected bool) {
	t.Helper()
	if parameters[key] != expected {
		t.Fatalf("expected %s to be %v, got %#v", key, expected, parameters[key])
	}
}

func assertFloat(t *testing.T, actual any, expected float64) {
	t.Helper()
	value, ok := NumericValue(actual)
	if !ok {
		t.Fatalf("expected numeric value %v, got %#v", expected, actual)
	}
	if math.Abs(value-expected) > 0.000001 {
		t.Fatalf("expected %v, got %v", expected, value)
	}
}

// The hiking base uses consider_elevation for a second, unrelated job: it also
// decides whether steps cost 1.0 or 3.0. Moving it with the slider would make
// the top of the range quietly start avoiding stairs, so the preference has to
// come out of the cost values alone.
func TestHikeHillPreferenceNeverTogglesTheStepsSwitch(t *testing.T) {
	previous := -1.0
	for _, hills := range []float64{0, 0.25, 0.5, 0.75, 0.94, 0.95, 1} {
		parameters := NativeConfigWithPreferences(TemplateHike, nil, map[string]any{
			"hillPreference": hills,
		}, "foot")["parameters"].(map[string]any)
		assertBool(t, parameters, "consider_elevation", true)

		cost, ok := NumericValue(parameters["uphillcostvalue"])
		if !ok {
			t.Fatalf("hill preference %v produced no uphill cost", hills)
		}
		if previous >= 0 && cost >= previous {
			t.Fatalf("uphill cost did not fall at hill preference %v: %v after %v", hills, cost, previous)
		}
		previous = cost
	}
	if previous != 0 {
		t.Fatalf("the top of the slider left an uphill cost of %v", previous)
	}
}
