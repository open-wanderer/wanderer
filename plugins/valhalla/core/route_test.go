package core

import (
	"reflect"
	"testing"
)

func TestBuildRouteRequestUsesSharedCostingOptions(t *testing.T) {
	request, costing, candidateCount, err := BuildRouteRequest(RoutingRequest{
		Anchors: []Anchor{{Lat: 47, Lon: 8}, {Lat: 47.1, Lon: 8.1}},
		Mode:    "bike",
		Profile: RoutingProfile{NativeConfig: map[string]any{
			"bicycle":      map[string]any{"use_hills": 0.8},
			"pedestrian":   map[string]any{"use_hills": 0.1},
			"bicycle_type": "Road",
		}},
		Preferences: map[string]any{
			"speedPreference": 22.0,
			"hillPreference":  0.2,
		},
		Options: RouteOptions{Alternatives: 20},
	})
	if err != nil {
		t.Fatalf("build route request: %v", err)
	}
	if costing != "bicycle" || request.Costing != "bicycle" {
		t.Fatalf("costing = %q/request %q, want bicycle", costing, request.Costing)
	}
	if candidateCount != MaxRouteCandidates || request.Alternates != MaxRouteCandidates-1 {
		t.Fatalf("candidate count/alternates = %d/%d", candidateCount, request.Alternates)
	}
	wantOptions := map[string]any{
		"cycling_speed": 22.0,
		"use_hills":     0.8,
		"bicycle_type":  "Road",
	}
	if got := request.CostingOptions["bicycle"]; !reflect.DeepEqual(got, wantOptions) {
		t.Fatalf("route costing options = %#v, want %#v", got, wantOptions)
	}
	if len(request.CostingOptions) != 1 {
		t.Fatalf("route costing namespaces = %#v", request.CostingOptions)
	}
}

func TestBuildRouteRequestOmitsAlternatesForMultipointRoute(t *testing.T) {
	request, _, candidateCount, err := BuildRouteRequest(RoutingRequest{
		Anchors: []Anchor{{Lat: 47, Lon: 8}, {Lat: 47.1, Lon: 8.1}, {Lat: 47.2, Lon: 8.2}},
		Profile: RoutingProfile{Key: "pedestrian"},
		Options: RouteOptions{Alternatives: 3},
	})
	if err != nil {
		t.Fatalf("build route request: %v", err)
	}
	if candidateCount != 3 {
		t.Fatalf("candidate count = %d, want 3", candidateCount)
	}
	if request.Alternates != 0 {
		t.Fatalf("alternates = %d, want 0", request.Alternates)
	}
}

func TestBuildRouteRequestValidatesInput(t *testing.T) {
	tests := []struct {
		name    string
		request RoutingRequest
	}{
		{name: "anchors", request: RoutingRequest{Profile: RoutingProfile{Key: "pedestrian"}}},
		{name: "costing", request: RoutingRequest{Anchors: []Anchor{{}, {}}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, _, err := BuildRouteRequest(test.request); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
