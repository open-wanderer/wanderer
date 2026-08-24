package core

import (
	"errors"
	"reflect"
	"testing"

	"github.com/open-wanderer/wanderer/plugins/sdk"
	"github.com/open-wanderer/wanderer/plugins/sdk/polyline"
)

func TestGenerateManeuversBatchesByPointLimitAndStitchesIntervals(t *testing.T) {
	request := testManeuverRequest(8)
	request.Language = "de"
	calls := 0
	result, err := GenerateManeuvers(request, ManeuverOptions{MaxShape: 4, MaxDistanceMeters: 1000000, Overlap: 2}, func(request TraceRequest) (TraceResponse, error) {
		calls++
		if request.Language != "de" || request.Units != "kilometers" {
			t.Fatalf("trace language/units = %q/%q", request.Language, request.Units)
		}
		return traceResponseFor(request.Shape, 26), nil
	})
	if err != nil {
		t.Fatalf("generate maneuvers: %v", err)
	}
	if calls < 2 {
		t.Fatalf("trace calls = %d, want batching", calls)
	}
	assertCompleteManeuverIntervals(t, result)
	for index, maneuver := range result.Maneuvers[1 : len(result.Maneuvers)-1] {
		if maneuver.Type == "start" || maneuver.Type == "destination" {
			t.Fatalf("artificial boundary maneuver at %d: %#v", index+1, maneuver)
		}
	}
}

func TestGenerateManeuversUsesSharedCostingOptions(t *testing.T) {
	request := testManeuverRequest(3)
	request.Mode = "bike"
	request.Profile = sdk.ManeuverProfile{
		Key: " ",
		NativeConfig: map[string]any{
			"bicycle":      map[string]any{"use_hills": 0.8},
			"pedestrian":   map[string]any{"use_hills": 0.1},
			"bicycle_type": "Road",
		},
	}
	request.Preferences = map[string]any{
		"speedPreference": 22.0,
		"hillPreference":  0.2,
	}
	wantOptions := map[string]any{
		"cycling_speed": 22.0,
		"use_hills":     0.8,
		"bicycle_type":  "Road",
	}
	result, err := GenerateManeuvers(request, ManeuverOptions{}, func(traceRequest TraceRequest) (TraceResponse, error) {
		if traceRequest.Costing != "bicycle" {
			t.Fatalf("trace costing = %q, want bicycle", traceRequest.Costing)
		}
		if len(traceRequest.CostingOptions) != 1 {
			t.Fatalf("trace costing namespaces = %#v", traceRequest.CostingOptions)
		}
		if got := traceRequest.CostingOptions["bicycle"]; !reflect.DeepEqual(got, wantOptions) {
			t.Fatalf("trace costing options = %#v, want %#v", got, wantOptions)
		}
		return traceResponseFor(traceRequest.Shape, 8), nil
	})
	if err != nil {
		t.Fatalf("generate maneuvers: %v", err)
	}
	assertCompleteManeuverIntervals(t, result)
}

func TestGenerateManeuversStitchesPastDiscardedStartInterval(t *testing.T) {
	request := testManeuverRequest(10)
	calls := 0
	result, err := GenerateManeuvers(request, ManeuverOptions{MaxShape: 6, MaxDistanceMeters: 1000000, Overlap: 2}, func(request TraceRequest) (TraceResponse, error) {
		calls++
		response := traceResponseFor(request.Shape, 8)
		if response.Trip.Legs[0].Maneuvers[0].EndShapeIndex != 3 {
			t.Fatalf("start maneuver end = %d, want 3", response.Trip.Legs[0].Maneuvers[0].EndShapeIndex)
		}
		return response, nil
	})
	if err != nil {
		t.Fatalf("generate maneuvers with overlapping start interval: %v", err)
	}
	if calls != 2 {
		t.Fatalf("trace calls = %d, want 2", calls)
	}
	assertCompleteManeuverIntervals(t, result)
	if got := result.Maneuvers[2].BeginShapeIndex; got != result.Maneuvers[1].EndShapeIndex {
		t.Fatalf("first maneuver from second chunk begins at %d, want %d", got, result.Maneuvers[1].EndShapeIndex)
	}
}

func TestGenerateManeuversStitchesMapMatchedGeometryWithoutSharedVertices(t *testing.T) {
	request := testManeuverRequest(6)
	calls := 0
	result, err := GenerateManeuvers(request, ManeuverOptions{MaxShape: 4, MaxDistanceMeters: 1000000, Overlap: 2}, func(request TraceRequest) (TraceResponse, error) {
		calls++
		points := request.Shape
		if calls == 1 {
			return traceResponseForCoordinates([][2]float64{
				{points[0].Lat, points[0].Lon},
				{points[1].Lat, points[1].Lon},
				{points[2].Lat, points[2].Lon},
				{points[3].Lat, points[3].Lon},
			}, 8), nil
		}
		return traceResponseForCoordinates([][2]float64{
			{points[0].Lat, points[0].Lon},
			{points[0].Lat + 0.0005, points[0].Lon},
			{points[1].Lat + 0.0005, points[1].Lon},
			{points[2].Lat + 0.0005, points[2].Lon},
			{points[len(points)-1].Lat, points[len(points)-1].Lon},
		}, 8), nil
	})
	if err != nil {
		t.Fatalf("generate map-matched batches: %v", err)
	}
	if calls != 2 {
		t.Fatalf("trace calls = %d, want 2", calls)
	}
	assertCompleteManeuverIntervals(t, result)
	geometry, err := polyline.Decode(result.Geometry.Coordinates, 1e6)
	if err != nil {
		t.Fatalf("decode stitched geometry: %v", err)
	}
	if len(geometry) != 7 {
		t.Fatalf("stitched geometry = %#v", geometry)
	}
}

func TestGenerateManeuversBatchesByDistanceAndMapsRoundabout(t *testing.T) {
	request := testManeuverRequest(6)
	calls := 0
	result, err := GenerateManeuvers(request, ManeuverOptions{MaxShape: 100, MaxDistanceMeters: 150, Overlap: 1}, func(request TraceRequest) (TraceResponse, error) {
		calls++
		return traceResponseFor(request.Shape, 27), nil
	})
	if err != nil {
		t.Fatalf("generate maneuvers: %v", err)
	}
	if calls < 2 {
		t.Fatalf("trace calls = %d, want distance batching", calls)
	}
	foundExit := false
	for _, maneuver := range result.Maneuvers {
		foundExit = foundExit || maneuver.Type == "roundabout_exit"
	}
	if !foundExit {
		t.Fatalf("roundabout exit mapping missing: %#v", result.Maneuvers)
	}
	assertCompleteManeuverIntervals(t, result)
}

func TestGenerateManeuversReportsBridgedSourcePartGap(t *testing.T) {
	request := testManeuverRequest(2)
	request.TrackParts = append(request.TrackParts, sdk.ManeuverTrackPart{Points: []sdk.ManeuverPoint{
		{Lat: 47.01, Lon: 8.01}, {Lat: 47.011, Lon: 8.011},
	}})
	result, err := GenerateManeuvers(request, ManeuverOptions{MaxShape: 100}, func(request TraceRequest) (TraceResponse, error) {
		return traceResponseFor(request.Shape, 26), nil
	})
	if err != nil {
		t.Fatalf("generate maneuvers: %v", err)
	}
	if len(result.Warnings) != 1 || result.Warnings[0] != "segment_gap_bridged" {
		t.Fatalf("warnings = %#v, want segment_gap_bridged", result.Warnings)
	}
}

func TestGenerateManeuversRejectsUnbridgeableSourcePartGap(t *testing.T) {
	request := testManeuverRequest(2)
	request.TrackParts = append(request.TrackParts, sdk.ManeuverTrackPart{Points: []sdk.ManeuverPoint{
		{Lat: 0, Lon: 0}, {Lat: 0.001, Lon: 0.001},
	}})
	_, err := GenerateManeuvers(request, ManeuverOptions{}, func(request TraceRequest) (TraceResponse, error) {
		return traceResponseFor(request.Shape, 8), nil
	})
	var adapterErr *AdapterError
	if !errors.As(err, &adapterErr) || adapterErr.Code != "discontinuous_track" {
		t.Fatalf("gap error = %#v", err)
	}
}

func TestGenerateManeuversRejectsInconsistentProviderIntervals(t *testing.T) {
	request := testManeuverRequest(3)
	_, err := GenerateManeuvers(request, ManeuverOptions{}, func(request TraceRequest) (TraceResponse, error) {
		response := traceResponseFor(request.Shape, 8)
		response.Trip.Legs[0].Maneuvers[0].Type = 8
		return response, nil
	})
	var adapterErr *AdapterError
	if !errors.As(err, &adapterErr) || adapterErr.Code != "provider_error" {
		t.Fatalf("provider interval error = %#v", err)
	}
}

func TestValidateCompleteIntervalsDistinguishesProviderAndStitchingErrors(t *testing.T) {
	maneuvers := []sdk.Maneuver{
		{Type: "start", BeginShapeIndex: 0, EndShapeIndex: 1},
		{Type: "continue", BeginShapeIndex: 2, EndShapeIndex: 3},
		{Type: "destination", BeginShapeIndex: 3, EndShapeIndex: 3},
	}
	providerErr := validateCompleteIntervals(maneuvers, 4, "Valhalla response")
	if providerErr == nil || providerErr.Error() != "Valhalla response contains maneuver intervals with gaps" {
		t.Fatalf("provider validation error = %v", providerErr)
	}
	stitchingErr := validateCompleteIntervals(maneuvers, 4, "Valhalla trace stitching")
	if stitchingErr == nil || stitchingErr.Error() != "Valhalla trace stitching contains maneuver intervals with gaps" {
		t.Fatalf("stitching validation error = %v", stitchingErr)
	}
}

func testManeuverRequest(pointCount int) sdk.ManeuverRequest {
	points := make([]sdk.ManeuverPoint, pointCount)
	for index := range points {
		points[index] = sdk.ManeuverPoint{Lat: 47 + float64(index)*0.001, Lon: 8}
	}
	return sdk.ManeuverRequest{
		TrackParts: []sdk.ManeuverTrackPart{{Points: points}},
		Mode:       "foot",
		Profile:    sdk.ManeuverProfile{Key: "pedestrian"},
		Limits: sdk.ManeuverLimits{
			MaxGeometryPoints:                20000,
			MaxManeuvers:                     1000,
			MaxProviderInstructionCharacters: 500,
			MaxStreetNames:                   4,
			MaxStreetNameCharacters:          120,
			MaxResponseBytes:                 4194304,
		},
	}
}

func traceResponseFor(points []sdk.ManeuverPoint, middleType int) TraceResponse {
	coordinates := make([][2]float64, len(points))
	for index, point := range points {
		coordinates[index] = [2]float64{point.Lat, point.Lon}
	}
	return traceResponseForCoordinates(coordinates, middleType)
}

func traceResponseForCoordinates(coordinates [][2]float64, middleType int) TraceResponse {
	last := len(coordinates) - 1
	firstTurn := min(3, last)
	exit := 2
	encoded, err := polyline.Encode(coordinates, 1e6)
	if err != nil {
		panic(err)
	}
	return TraceResponse{Trip: TraceTrip{Legs: []TraceLeg{{
		Shape: encoded,
		Maneuvers: []TraceManeuver{
			{Type: 1, BeginShapeIndex: 0, EndShapeIndex: firstTurn},
			{Type: middleType, Length: 0.1, BeginShapeIndex: firstTurn, EndShapeIndex: last, RoundaboutExitCount: &exit},
			{Type: 4, BeginShapeIndex: last, EndShapeIndex: last},
		},
	}}}}
}

func assertCompleteManeuverIntervals(t *testing.T, result sdk.ManeuverResult) {
	t.Helper()
	if len(result.Maneuvers) < 2 {
		t.Fatalf("maneuvers = %d", len(result.Maneuvers))
	}
	if result.Maneuvers[0].Type != "start" || result.Maneuvers[0].BeginShapeIndex != 0 {
		t.Fatalf("invalid first maneuver: %#v", result.Maneuvers[0])
	}
	for index := 1; index < len(result.Maneuvers); index++ {
		if result.Maneuvers[index].BeginShapeIndex != result.Maneuvers[index-1].EndShapeIndex {
			t.Fatalf("interval gap at %d: %#v", index, result.Maneuvers)
		}
	}
	last := result.Maneuvers[len(result.Maneuvers)-1]
	if last.Type != "destination" || last.BeginShapeIndex != last.EndShapeIndex || last.DistanceMeters != 0 {
		t.Fatalf("invalid destination: %#v", last)
	}
}
