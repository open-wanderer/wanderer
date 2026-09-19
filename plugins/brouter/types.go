//go:build tinygo

package main

import (
	"github.com/open-wanderer/wanderer/plugins/brouter/core"
	"github.com/open-wanderer/wanderer/plugins/sdk"
)

type pluginError = sdk.PluginError

type routingRouteInput struct {
	Instance sdk.InstanceRef `json:"instance"`
	Auth     map[string]any  `json:"auth,omitempty"`
	Config   map[string]any  `json:"config,omitempty"`
	Request  routeRequest    `json:"request"`
}

type routingRoundTripInput struct {
	Instance sdk.InstanceRef  `json:"instance"`
	Auth     map[string]any   `json:"auth,omitempty"`
	Config   map[string]any   `json:"config,omitempty"`
	Request  roundTripRequest `json:"request"`
}

type routingProfileIntrospectInput struct {
	Instance sdk.InstanceRef          `json:"instance"`
	Auth     map[string]any           `json:"auth,omitempty"`
	Config   map[string]any           `json:"config,omitempty"`
	Request  profileIntrospectRequest `json:"request"`
}

type routingProfilePrepareInput struct {
	Instance sdk.InstanceRef       `json:"instance"`
	Auth     map[string]any        `json:"auth,omitempty"`
	Config   map[string]any        `json:"config,omitempty"`
	Request  profilePrepareRequest `json:"request"`
}

type profilePrepareRequest struct {
	Mode                string         `json:"mode,omitempty"`
	Profile             routingProfile `json:"profile"`
	Preferences         map[string]any `json:"preferences,omitempty"`
	RequiredPreferences []string       `json:"requiredPreferences,omitempty"`
}

type profilePrepareOutput struct {
	PreparedKey string       `json:"preparedKey,omitempty"`
	Error       *pluginError `json:"error,omitempty"`
}

type profileIntrospectRequest struct {
	Profile routingProfile `json:"profile"`
}

type profileIntrospectOutput struct {
	NativeControlGroups  []nativeControlGroup `json:"nativeControlGroups,omitempty"`
	Mode                 string               `json:"mode,omitempty"`
	SupportedPreferences []string             `json:"supportedPreferences,omitempty"`
	Metadata             map[string]any       `json:"metadata,omitempty"`
	Error                *pluginError         `json:"error,omitempty"`
}

type nativeControlGroup struct {
	Key      string            `json:"key"`
	Label    string            `json:"label"`
	Labels   map[string]string `json:"labels,omitempty"`
	Controls []nativeControl   `json:"controls"`
}

type nativeControl struct {
	Key       string              `json:"key"`
	Label     string              `json:"label,omitempty"`
	Type      string              `json:"type"`
	UI        string              `json:"ui,omitempty"`
	ValueType string              `json:"valueType,omitempty"`
	Min       *float64            `json:"min,omitempty"`
	Max       *float64            `json:"max,omitempty"`
	Step      *float64            `json:"step,omitempty"`
	Default   any                 `json:"default,omitempty"`
	Current   any                 `json:"current,omitempty"`
	Target    string              `json:"target,omitempty"`
	Path      []string            `json:"path,omitempty"`
	Options   []map[string]string `json:"options,omitempty"`
}

type routeRequest = core.RouteRequest
type roundTripRequest = core.RoundTripRequest
type anchor = core.Anchor
type routingProfile = core.RoutingProfile
type routeOptions = core.RouteOptions

type routeOutput struct {
	Candidates []routeCandidate `json:"candidates,omitempty"`
	Error      *pluginError     `json:"error,omitempty"`
}

type roundTripOutput = routeOutput

type routeCandidate = core.RouteCandidate
type elevation = core.Elevation
type geometry = core.Geometry
type summary = core.Summary
type segment = core.Segment
type brouterFeatureCollection = core.FeatureCollection

type brouterRoundTripFeatureCollection = core.RawFeatureCollection

type brouterProfileUploadResponse struct {
	Error     any    `json:"error,omitempty"`
	ProfileID string `json:"profileid,omitempty"`
}

type brouterFeature = core.Feature
type brouterProperties = core.Properties
type brouterGeometry = core.LineString
