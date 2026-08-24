//go:build tinygo

package main

import "github.com/open-wanderer/wanderer/plugins/sdk"

type pluginError = sdk.PluginError

type routingRouteInput struct {
	Instance sdk.InstanceRef `json:"instance"`
	Auth     map[string]any  `json:"auth,omitempty"`
	Config   map[string]any  `json:"config,omitempty"`
	Request  routeRequest    `json:"request"`
}

type routingElevationInput struct {
	Instance sdk.InstanceRef  `json:"instance"`
	Auth     map[string]any   `json:"auth,omitempty"`
	Config   map[string]any   `json:"config,omitempty"`
	Request  elevationRequest `json:"request"`
}

type routingManeuverInput struct {
	Instance sdk.InstanceRef     `json:"instance"`
	Auth     map[string]any      `json:"auth,omitempty"`
	Config   map[string]any      `json:"config,omitempty"`
	Request  sdk.ManeuverRequest `json:"request"`
}

type routeRequest struct {
	RoutingMode         string         `json:"routingMode"`
	Anchors             []anchor       `json:"anchors"`
	Mode                string         `json:"mode,omitempty"`
	Profile             routingProfile `json:"profile"`
	Preferences         map[string]any `json:"preferences,omitempty"`
	RequiredPreferences []string       `json:"requiredPreferences,omitempty"`
	Options             routeOptions   `json:"options,omitempty"`
}

type elevationRequest struct {
	EncodedPolyline string   `json:"encodedPolyline,omitempty"`
	Coordinates     []anchor `json:"coordinates,omitempty"`
}

type anchor struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type routingProfile struct {
	ID           string         `json:"id,omitempty"`
	PluginID     string         `json:"pluginId,omitempty"`
	Key          string         `json:"key"`
	Kind         string         `json:"kind,omitempty"`
	NativeConfig map[string]any `json:"nativeConfig,omitempty"`
}

type routeOptions struct {
	Alternatives     int  `json:"alternatives,omitempty"`
	IncludeElevation bool `json:"includeElevation,omitempty"`
}

type routeOutput struct {
	Candidates []routeCandidate `json:"candidates,omitempty"`
	Error      *pluginError     `json:"error,omitempty"`
}

type elevationOutput struct {
	Heights []float64    `json:"heights,omitempty"`
	Error   *pluginError `json:"error,omitempty"`
}

type routeCandidate struct {
	ID             string    `json:"id"`
	ProfileKey     string    `json:"profileKey,omitempty"`
	Geometry       *geometry `json:"geometry,omitempty"`
	Summary        summary   `json:"summary"`
	Segments       []segment `json:"segments"`
	SnappedAnchors []anchor  `json:"snappedAnchors,omitempty"`
	Warnings       []string  `json:"warnings,omitempty"`
}

type geometry struct {
	Format      string `json:"format"`
	Precision   int    `json:"precision"`
	Coordinates string `json:"coordinates"`
}

type summary struct {
	Distance      float64 `json:"distance"`
	Duration      float64 `json:"duration"`
	ElevationGain float64 `json:"elevationGain,omitempty"`
	ElevationLoss float64 `json:"elevationLoss,omitempty"`
}

type segment struct {
	FromAnchor int      `json:"fromAnchor"`
	ToAnchor   int      `json:"toAnchor"`
	Geometry   geometry `json:"geometry"`
	Distance   float64  `json:"distance"`
	Duration   float64  `json:"duration"`
}

type valhallaRouteRequest struct {
	DirectionsType string         `json:"directions_type"`
	Locations      []anchor       `json:"locations"`
	Costing        string         `json:"costing"`
	CostingOptions map[string]any `json:"costing_options,omitempty"`
	Alternates     int            `json:"alternates,omitempty"`
}

type valhallaRouteResponse struct {
	Trip       valhallaTrip             `json:"trip"`
	Alternates []valhallaAlternateRoute `json:"alternates,omitempty"`
}

type valhallaAlternateRoute struct {
	Trip valhallaTrip `json:"trip"`
}

type valhallaTrip struct {
	Locations []valhallaLocation `json:"locations"`
	Legs      []valhallaLeg      `json:"legs"`
	Summary   valhallaSummary    `json:"summary"`
}

type valhallaLocation struct {
	Lat           float64 `json:"lat"`
	Lon           float64 `json:"lon"`
	OriginalIndex int     `json:"original_index"`
}

type valhallaLeg struct {
	Summary valhallaSummary `json:"summary"`
	Shape   string          `json:"shape"`
}

type valhallaSummary struct {
	Time   float64 `json:"time"`
	Length float64 `json:"length"`
}

type valhallaHeightRequest struct {
	EncodedPolyline string   `json:"encoded_polyline,omitempty"`
	Shape           []anchor `json:"shape,omitempty"`
}

type valhallaHeightResponse struct {
	Height []float64 `json:"height"`
}
