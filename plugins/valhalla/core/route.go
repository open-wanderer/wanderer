package core

import "fmt"

// MaxRouteCandidates is the maximum number of routes returned for one request.
const MaxRouteCandidates = 4

// Anchor identifies a route location in latitude/longitude coordinates.
type Anchor struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// RoutingProfile identifies the selected Wanderer routing profile and its
// provider-specific configuration.
type RoutingProfile struct {
	ID           string         `json:"id,omitempty"`
	PluginID     string         `json:"pluginId,omitempty"`
	Key          string         `json:"key"`
	Kind         string         `json:"kind,omitempty"`
	NativeConfig map[string]any `json:"nativeConfig,omitempty"`
}

// RouteOptions controls optional route variants and elevation output.
type RouteOptions struct {
	Alternatives     int  `json:"alternatives,omitempty"`
	IncludeElevation bool `json:"includeElevation,omitempty"`
}

// RoutingRequest is the Wanderer route_v1 request consumed by the adapter.
type RoutingRequest struct {
	RoutingMode         string         `json:"routingMode"`
	Anchors             []Anchor       `json:"anchors"`
	Mode                string         `json:"mode,omitempty"`
	Profile             RoutingProfile `json:"profile"`
	Preferences         map[string]any `json:"preferences,omitempty"`
	RequiredPreferences []string       `json:"requiredPreferences,omitempty"`
	Options             RouteOptions   `json:"options,omitempty"`
}

// RouteRequest is the request body sent to Valhalla's /route endpoint.
type RouteRequest struct {
	DirectionsType string         `json:"directions_type"`
	Locations      []Anchor       `json:"locations"`
	Costing        string         `json:"costing"`
	CostingOptions map[string]any `json:"costing_options,omitempty"`
	Alternates     int            `json:"alternates,omitempty"`
}

// BuildRouteRequest validates a Wanderer routing request and translates it to
// Valhalla's /route request contract.
func BuildRouteRequest(req RoutingRequest) (RouteRequest, string, int, error) {
	if len(req.Anchors) < 2 {
		return RouteRequest{}, "", 0, fmt.Errorf("at least two anchors are required")
	}
	costing := ResolveCosting(req.Profile.Key, req.Mode)
	if costing == "" {
		return RouteRequest{}, "", 0, fmt.Errorf("routing profile key is required")
	}
	request := RouteRequest{
		DirectionsType: "none",
		Locations:      req.Anchors,
		Costing:        costing,
		CostingOptions: BuildCostingOptions(costing, req.Preferences, req.Profile.NativeConfig),
	}
	candidateCount := req.Options.Alternatives
	if candidateCount < 1 {
		candidateCount = 1
	}
	if candidateCount > MaxRouteCandidates {
		candidateCount = MaxRouteCandidates
	}
	// Valhalla does not provide alternatives for multipoint routes.
	if len(req.Anchors) == 2 {
		request.Alternates = candidateCount - 1
	}
	return request, costing, candidateCount, nil
}
