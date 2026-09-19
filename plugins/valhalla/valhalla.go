//go:build tinygo

package main

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/open-wanderer/wanderer/plugins/sdk"
	"github.com/open-wanderer/wanderer/plugins/sdk/polyline"
	valhallacore "github.com/open-wanderer/wanderer/plugins/valhalla/core"
)

const (
	valhallaConnector        = "api"
	valhallaJSONBytes        = 4 * 1024 * 1024
	encodedPolylineFormat    = "encoded_polyline"
	encodedPolylinePrecision = 6
)

var valhallaJSONTypes = []string{"application/json"}

func handleRoute(input routingRouteInput) (routeOutput, error) {
	req := input.Request
	valhallaReq, costing, candidateCount, err := valhallacore.BuildRouteRequest(req)
	if err != nil {
		return routeOutput{}, err
	}
	var valhallaResp valhallaRouteResponse
	if err := postJSON("/route", valhallaReq, &valhallaResp); err != nil {
		return routeOutput{}, err
	}
	candidates := make([]routeCandidate, 0, 1+len(valhallaResp.Alternates))
	primary, err := candidateFromValhallaTrip(req, costing, "primary", valhallaResp.Trip)
	if err != nil {
		return routeOutput{}, err
	}
	candidates = append(candidates, primary)
	for index, alternate := range valhallaResp.Alternates {
		if len(candidates) >= candidateCount {
			break
		}
		candidate, err := candidateFromValhallaTrip(req, costing, fmt.Sprintf("alternative-%d", index+1), alternate.Trip)
		if err != nil {
			continue
		}
		candidates = append(candidates, candidate)
	}
	return routeOutput{Candidates: candidates}, nil
}

func candidateFromValhallaTrip(req routeRequest, costing string, id string, trip valhallaTrip) (routeCandidate, error) {
	if len(trip.Legs) == 0 {
		return routeCandidate{}, fmt.Errorf("valhalla returned no route legs")
	}
	candidate := routeCandidate{
		ID:             id,
		ProfileKey:     costing,
		Summary:        summaryFromValhalla(trip.Summary),
		Segments:       segmentsFromValhalla(req, trip.Legs),
		SnappedAnchors: snappedAnchors(trip.Locations),
	}
	if len(trip.Legs) == 1 {
		candidate.Geometry = &candidate.Segments[0].Geometry
	} else {
		coordinates := make([][2]float64, 0)
		for _, leg := range trip.Legs {
			decoded, err := polyline.Decode(leg.Shape, 1e6)
			if err != nil {
				return routeCandidate{}, err
			}
			if len(coordinates) > 0 && len(decoded) > 0 {
				decoded = decoded[1:]
			}
			coordinates = append(coordinates, decoded...)
		}
		candidate.Geometry = &geometry{
			Format:      encodedPolylineFormat,
			Precision:   encodedPolylinePrecision,
			Coordinates: mustEncodePolyline(coordinates, 1e6),
		}
	}
	return candidate, nil
}

func handleElevation(input routingElevationInput) (elevationOutput, error) {
	req := input.Request
	valhallaReq := valhallaHeightRequest{
		EncodedPolyline: req.EncodedPolyline,
		Shape:           req.Coordinates,
	}
	var valhallaResp valhallaHeightResponse
	if err := postJSON("/height", valhallaReq, &valhallaResp); err != nil {
		return elevationOutput{}, err
	}
	return elevationOutput{Heights: valhallaResp.Height}, nil
}

func handleManeuvers(input routingManeuverInput) (sdk.ManeuverResult, error) {
	options := valhallacore.ManeuverOptions{
		MaxShape:          configInt(input.Config, "traceMaxShape", valhallacore.DefaultTraceMaxShape),
		MaxDistanceMeters: configNumber(input.Config, "traceMaxDistanceMeters", valhallacore.DefaultTraceMaxDistanceMeters),
		Overlap:           configInt(input.Config, "traceOverlap", valhallacore.DefaultTraceOverlap),
	}
	if trace, ok := input.Config["trace"].(map[string]any); ok {
		options.MaxShape = configInt(trace, "maxShape", options.MaxShape)
		options.MaxDistanceMeters = configNumber(trace, "maxDistance", options.MaxDistanceMeters)
		options.Overlap = configInt(trace, "overlap", options.Overlap)
	}
	return valhallacore.GenerateManeuvers(input.Request, options, func(request valhallacore.TraceRequest) (valhallacore.TraceResponse, error) {
		var response valhallacore.TraceResponse
		if err := postJSON("/trace_route", request, &response); err != nil {
			return valhallacore.TraceResponse{}, err
		}
		return response, nil
	})
}

func configInt(config map[string]any, key string, fallback int) int {
	value := configNumber(config, key, float64(fallback))
	if value < 1 || value > 2000000000 {
		return fallback
	}
	return int(value)
}

func configNumber(config map[string]any, key string, fallback float64) float64 {
	value, ok := config[key]
	if !ok {
		return fallback
	}
	var number float64
	switch value := value.(type) {
	case float64:
		number = value
	case float32:
		number = float64(value)
	case int:
		number = float64(value)
	case int64:
		number = float64(value)
	default:
		return fallback
	}
	if number <= 0 || math.IsNaN(number) || math.IsInf(number, 0) {
		return fallback
	}
	return number
}

func postJSON(path string, body any, out any) error {
	response, responseBody, err := sdk.PostJSON(valhallaConnector, path, nil, map[string]string{
		"Accept": "application/json",
	}, body, sdk.ResponseExpect{ContentTypes: valhallaJSONTypes, MaxBytes: valhallaJSONBytes})
	if err != nil {
		return err
	}
	if response.Status < 200 || response.Status >= 300 {
		return fmt.Errorf("valhalla request failed (%d): %s", response.Status, string(responseBody))
	}
	return json.Unmarshal(responseBody, out)
}

func summaryFromValhalla(s valhallaSummary) summary {
	return summary{
		Distance: s.Length * 1000,
		Duration: s.Time,
	}
}

func segmentsFromValhalla(req routeRequest, legs []valhallaLeg) []segment {
	segments := make([]segment, 0, len(legs))
	for i, leg := range legs {
		segments = append(segments, segment{
			FromAnchor: i,
			ToAnchor:   i + 1,
			Geometry: geometry{
				Format:      encodedPolylineFormat,
				Precision:   encodedPolylinePrecision,
				Coordinates: leg.Shape,
			},
			Distance: leg.Summary.Length * 1000,
			Duration: leg.Summary.Time,
		})
	}
	return segments
}

func snappedAnchors(locations []valhallaLocation) []anchor {
	if len(locations) == 0 {
		return nil
	}
	anchors := make([]anchor, len(locations))
	for i, location := range locations {
		index := location.OriginalIndex
		if index < 0 || index >= len(anchors) {
			index = i
		}
		anchors[index] = anchor{Lat: location.Lat, Lon: location.Lon}
	}
	return anchors
}

func mustEncodePolyline(coords [][2]float64, precision float64) string {
	encoded, _ := polyline.Encode(coords, precision)
	return encoded
}
