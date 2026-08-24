package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"pocketbase/pluginsystem"
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

const (
	routingMaxManeuverInputPoints      = 200000
	routingMaxManeuvers                = 1000
	routingMaxProviderInstructionChars = 500
	routingMaxManeuverStreetNames      = 4
	routingMaxManeuverStreetNameChars  = 120
	routingMaxManeuverHostRequests     = 32
)

var (
	routingManeuverRateLimiter  = util.NewRateLimiter(30, time.Minute)
	routingManeuverPluginCaller = callRoutingManeuverPlugin
	routingManeuverGPXReader    = util.ReadTrailGPX
)

type pluginRoutingManeuverHTTPInput struct {
	TrailID  string `json:"trailId"`
	Language string `json:"language,omitempty"`
	Share    string `json:"share,omitempty"`
}

type pluginRoutingManeuverPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type pluginRoutingManeuverTrackPart struct {
	Points []pluginRoutingManeuverPoint `json:"points"`
}

type pluginRoutingManeuverLimits struct {
	MaxGeometryPoints                int   `json:"maxGeometryPoints"`
	MaxManeuvers                     int   `json:"maxManeuvers"`
	MaxProviderInstructionCharacters int   `json:"maxProviderInstructionCharacters"`
	MaxStreetNames                   int   `json:"maxStreetNames"`
	MaxStreetNameCharacters          int   `json:"maxStreetNameCharacters"`
	MaxResponseBytes                 int64 `json:"maxResponseBytes"`
}

type pluginRoutingManeuverRequest struct {
	TrackParts          []pluginRoutingManeuverTrackPart `json:"trackParts"`
	Mode                string                           `json:"mode,omitempty"`
	Category            string                           `json:"category,omitempty"`
	Subcategory         string                           `json:"subcategory,omitempty"`
	Profile             pluginRoutingProfile             `json:"profile"`
	Preferences         map[string]any                   `json:"preferences,omitempty"`
	RequiredPreferences []string                         `json:"requiredPreferences,omitempty"`
	Language            string                           `json:"language,omitempty"`
	Limits              pluginRoutingManeuverLimits      `json:"limits"`
}

type pluginRoutingManeuver struct {
	Type                string   `json:"type"`
	ProviderInstruction string   `json:"providerInstruction,omitempty"`
	DistanceMeters      float64  `json:"distanceMeters"`
	DurationSeconds     *float64 `json:"durationSeconds,omitempty"`
	BeginShapeIndex     int      `json:"beginShapeIndex"`
	EndShapeIndex       int      `json:"endShapeIndex"`
	BearingBefore       *float64 `json:"bearingBefore,omitempty"`
	BearingAfter        *float64 `json:"bearingAfter,omitempty"`
	RoundaboutExit      *int     `json:"roundaboutExit,omitempty"`
	StreetNames         []string `json:"streetNames,omitempty"`
	Warnings            []string `json:"warnings,omitempty"`
}

type pluginRoutingManeuverOutput struct {
	Geometry  pluginRoutingGeometry     `json:"geometry"`
	Maneuvers []pluginRoutingManeuver   `json:"maneuvers,omitempty"`
	Warnings  []string                  `json:"warnings,omitempty"`
	Error     *pluginsystem.PluginError `json:"error,omitempty"`
}

func maneuverLimits() pluginRoutingManeuverLimits {
	return pluginRoutingManeuverLimits{
		MaxGeometryPoints:                routingMaxPolylinePoints,
		MaxManeuvers:                     routingMaxManeuvers,
		MaxProviderInstructionCharacters: routingMaxProviderInstructionChars,
		MaxStreetNames:                   routingMaxManeuverStreetNames,
		MaxStreetNameCharacters:          routingMaxManeuverStreetNameChars,
		MaxResponseBytes:                 routingMaxResponseBodyBytes,
	}
}

// PluginSystemRoutingManeuvers generates provider-neutral turn-by-turn data
// for one persisted trail. The handler deliberately remains stateless.
func PluginSystemRoutingManeuvers(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}
	var data pluginRoutingManeuverHTTPInput
	if err := e.BindBody(&data); err != nil {
		return apis.NewBadRequestError("Failed to read request data", err)
	}
	data.TrailID = strings.TrimSpace(data.TrailID)
	data.Language = strings.TrimSpace(data.Language)
	if data.TrailID == "" {
		return routingJSONError(e, routingErrorFromCode("invalid_request", "trailId is required"), nil)
	}
	if len(data.Language) > 64 {
		return routingJSONError(e, routingErrorFromCode("invalid_request", "language is too long"), nil)
	}
	settings, err := ResolveRoutingSettings(e.App, e.Auth.Id)
	if err != nil {
		return err
	}
	if !routingFeatureEnabled(settings, "navigation", true) {
		return routingJSONError(e, routingErrorFromCode("navigation_disabled", "navigation is not enabled"), nil)
	}
	if err := routingManeuverRateLimiter.CheckRateLimit(e.Auth.Id, "maneuvers"); err != nil {
		return e.TooManyRequestsError("too many maneuver requests", err)
	}

	trail, err := e.App.FindRecordById("trails", data.TrailID)
	if err != nil {
		return apis.NewNotFoundError("trail not found", nil)
	}
	if !util.TrailViewableByUser(e.App, trail, e.Auth.Id, data.Share) {
		return apis.NewForbiddenError("not allowed to navigate this trail", nil)
	}
	gpxContent, err := routingManeuverGPXReader(e.App, trail)
	if err != nil {
		return err
	}
	trackParts, err := normalizedManeuverTrackParts(gpxContent)
	if err != nil {
		return routingJSONError(e, err, nil)
	}

	runtime, err := maneuverRuntime(e, settings.ManeuverPluginID)
	if err != nil {
		var routingErr *routingError
		if !errors.As(err, &routingErr) {
			e.App.Logger().Error("Failed to resolve maneuver engine", "pluginId", settings.ManeuverPluginID, "error", err)
		}
		return routingJSONError(e, err, nil)
	}
	category, subcategory := routingTrailCategoryNames(e.App, trail)
	resolved := pluginRoutingRouteRequest{
		Category:    category,
		Subcategory: subcategory,
		Profile:     pluginRoutingProfile{PluginID: runtime.Plugin.Manifest.ID},
	}
	if err := applyRoutingManeuverProfile(e.Request.Context(), e.App, e.Auth.Id, runtime, &resolved, &settings); err != nil {
		return routingJSONError(e, err, nil)
	}
	runtime.Request = resolved
	runtime.ClientRequest = cloneRoutingRouteRequest(resolved)
	runtime.PreparedProfileRefresh = &routingPreparedProfileRefresh{}
	routingCtx, closeRoutingSessions := withRoutingPluginSessions(e.Request.Context())
	defer closeRoutingSessions()
	runtime, err = prepareRoutingRuntimeProfile(routingCtx, runtime)
	if err != nil {
		return routingJSONError(e, err, nil)
	}

	request := pluginRoutingManeuverRequest{
		TrackParts:          trackParts,
		Mode:                runtime.Request.Mode,
		Category:            runtime.Request.Category,
		Subcategory:         runtime.Request.Subcategory,
		Profile:             runtime.Request.Profile,
		Preferences:         cloneRoutingMap(runtime.Request.Preferences),
		RequiredPreferences: append([]string(nil), runtime.Request.RequiredPreferences...),
		Language:            data.Language,
		Limits:              maneuverLimits(),
	}
	output, err := callPreparedRoutingManeuverPlugin(routingCtx, runtime, request)
	if err != nil {
		return routingJSONError(e, routingErrorFromCall(err), nil)
	}
	if output.Error != nil {
		return routingJSONError(e, routingErrorFromPluginError(*output.Error), nil)
	}
	if err := normalizeRoutingManeuverOutput(&output); err != nil {
		return routingJSONError(e, err, nil)
	}
	return e.JSON(http.StatusOK, output)
}

func maneuverRuntime(e *core.RequestEvent, pluginID string) (routingEngineRuntime, error) {
	preferred := strings.TrimSpace(pluginID)
	engines, err := routingEngines(e.App, e.Auth.Id)
	if err != nil {
		return routingEngineRuntime{}, err
	}
	pluginIDs := make([]string, 0, len(engines)+1)
	if preferred != "" {
		pluginIDs = append(pluginIDs, preferred)
	}
	for _, engine := range engines {
		if engine.Enabled && containsRoutingString(engine.Roles, "maneuvers") && engine.PluginID != preferred {
			pluginIDs = append(pluginIDs, engine.PluginID)
		}
	}
	for _, candidate := range pluginIDs {
		runtime, runtimeErr := maneuverRuntimeForPlugin(e, candidate)
		if runtimeErr == nil {
			return runtime, nil
		}
		var routingErr *routingError
		if !errors.As(runtimeErr, &routingErr) || routingErr.Code != "maneuver_engine_unavailable" {
			return routingEngineRuntime{}, runtimeErr
		}
	}
	return routingEngineRuntime{}, routingErrorFromCode("maneuver_engine_unavailable", "no enabled maneuver engine is available")
}

func maneuverRuntimeForPlugin(e *core.RequestEvent, pluginID string) (routingEngineRuntime, error) {
	plugin, err := pluginsystem.LoadInstalledPlugin(e.App, "", pluginID)
	if errors.Is(err, pluginsystem.ErrPluginNotFound) {
		return routingEngineRuntime{}, routingErrorFromCode("maneuver_engine_unavailable", "the selected maneuver engine is unavailable")
	}
	if err != nil {
		return routingEngineRuntime{}, err
	}
	if plugin.Manifest.Type != pluginsystem.PluginTypeRouting || !routingManifestSupportsManeuvers(plugin.Manifest) {
		return routingEngineRuntime{}, routingErrorFromCode("maneuver_engine_unavailable", "the selected maneuver engine is unavailable")
	}
	capability, err := pluginCapability(plugin, "maneuvers", "v1")
	if err != nil {
		return routingEngineRuntime{}, err
	}
	instance, err := routingEnabledPluginInstance(e.App, e.Auth.Id, pluginID, "")
	if err != nil {
		return routingEngineRuntime{}, err
	}
	if instance == nil {
		return routingEngineRuntime{}, routingErrorFromCode("maneuver_engine_unavailable", "the selected maneuver engine is unavailable")
	}
	auth, err := decryptedInstanceAuth(instance)
	if err != nil {
		return routingEngineRuntime{}, err
	}
	return routingEngineRuntime{
		Selection:  routingEngineSelection{PluginID: pluginID, InstanceID: instance.Id},
		Plugin:     plugin,
		Capability: capability,
		Instance:   instance,
		Auth:       auth,
		Config:     effectiveRoutingPluginConfig(e.App, plugin, instance),
	}, nil
}

func applyRoutingManeuverProfile(ctx context.Context, app core.App, userID string, runtime routingEngineRuntime, request *pluginRoutingRouteRequest, settings *routingSettings) error {
	err := applyRoutingCategoryMappingContext(ctx, app, userID, runtime.Plugin.Manifest.ID, instanceID(runtime.Instance), request, settings)
	if err == nil && request.Profile.Key != "" {
		restrictRoutingPluginPreferences(runtime.Plugin, request)
		return nil
	}
	if err != nil {
		var routingErr *routingError
		if !errors.As(err, &routingErr) || routingErr.Code != "mapping_missing" {
			return err
		}
		category, subcategory := request.Category, request.Subcategory
		fallback := cloneRoutingRouteRequest(*request)
		fallback.Category = ""
		fallback.Subcategory = ""
		fallback.Profile = pluginRoutingProfile{PluginID: runtime.Plugin.Manifest.ID}
		if err := applyRoutingCategoryMappingContext(ctx, app, userID, runtime.Plugin.Manifest.ID, instanceID(runtime.Instance), &fallback, settings); err != nil {
			return err
		}
		fallback.Category = category
		fallback.Subcategory = subcategory
		*request = fallback
	}
	if err := applyRoutingManifestDefaultProfile(runtime.Plugin.Manifest, request); err != nil {
		return err
	}
	restrictRoutingPluginPreferences(runtime.Plugin, request)
	return nil
}

func applyRoutingManifestDefaultProfile(manifest pluginsystem.Manifest, request *pluginRoutingRouteRequest) error {
	profileKeys := stringSlice(mapValue(manifest.HostConfig["routing"])["defaultProfiles"])
	for _, raw := range anySlice(mapValue(manifest.Metadata["routing"])["nativeProfiles"]) {
		key := stringValue(mapValue(raw)["key"])
		if key != "" && !containsRoutingString(profileKeys, key) {
			profileKeys = append(profileKeys, key)
		}
	}
	for _, key := range profileKeys {
		profile := routingManifestNativeProfile(manifest, key)
		if len(profile) == 0 {
			continue
		}
		mode := stringValue(profile["mode"])
		request.Mode = mode
		request.Profile.PluginID = manifest.ID
		request.Profile.Key = key
		request.Profile.Kind = "builtin"
		request.Profile.Mode = mode
		request.Profile.Metadata = profile
		request.Profile.NativeConfig = mergeRoutingConfigLayers(mapValue(profile["nativeConfig"]), request.Profile.NativeConfig)
		return nil
	}
	return routingErrorFromCode("unsupported_profile", "maneuver engine has no default routing profile")
}

func routingTrailCategoryNames(app core.App, trail *core.Record) (string, string) {
	category := strings.TrimSpace(trail.GetString("federated_category_name"))
	subcategory := strings.TrimSpace(trail.GetString("federated_subcategory_name"))
	if id := strings.TrimSpace(trail.GetString("category")); id != "" {
		if record, err := app.FindRecordById("categories", id); err == nil {
			if name := strings.TrimSpace(record.GetString("name")); name != "" {
				category = name
			}
		}
	}
	if id := strings.TrimSpace(trail.GetString("subcategory")); id != "" {
		if record, err := app.FindRecordById("subcategories", id); err == nil {
			if name := strings.TrimSpace(record.GetString("name")); name != "" {
				subcategory = name
			}
		}
	}
	return category, subcategory
}

func normalizedManeuverTrackParts(content []byte) ([]pluginRoutingManeuverTrackPart, error) {
	if len(content) == 0 {
		return nil, routingErrorFromCode("track_shape_unavailable", "trail has no usable GPX track or route")
	}
	decoder := xml.NewDecoder(bytes.NewReader(content))
	trackParts := make([]pluginRoutingManeuverTrackPart, 0)
	var currentTrack []pluginRoutingManeuverPoint
	var firstRoute []pluginRoutingManeuverPoint
	insideTrackSegment := false
	insideRoute := false
	routeHasPointElements := false
	firstNonEmptyRouteChosen := false
	trackPointCount := 0
	firstRoutePointCount := 0

	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, routingErrorFromCode("track_shape_unavailable", "trail GPX cannot be parsed")
		}
		switch token := token.(type) {
		case xml.StartElement:
			switch token.Name.Local {
			case "trkseg":
				insideTrackSegment = true
				currentTrack = nil
			case "trkpt":
				if insideTrackSegment {
					if point, ok := maneuverPointFromXML(token); ok {
						currentTrack = append(currentTrack, point)
						trackPointCount++
						if trackPointCount > routingMaxManeuverInputPoints {
							return nil, routingErrorFromCode("track_limit_exceeded", "trail contains too many normalized track points")
						}
					}
				}
			case "rte":
				insideRoute = !firstNonEmptyRouteChosen
				if insideRoute {
					routeHasPointElements = false
					firstRoute = nil
					firstRoutePointCount = 0
				}
			case "rtept":
				if insideRoute {
					routeHasPointElements = true
					if point, ok := maneuverPointFromXML(token); ok {
						firstRoutePointCount++
						if firstRoutePointCount <= routingMaxManeuverInputPoints+1 {
							firstRoute = append(firstRoute, point)
						}
					}
				}
			}
		case xml.EndElement:
			switch token.Name.Local {
			case "trkseg":
				if insideTrackSegment && len(currentTrack) > 0 {
					trackParts = append(trackParts, pluginRoutingManeuverTrackPart{Points: currentTrack})
				}
				insideTrackSegment = false
				currentTrack = nil
			case "rte":
				if insideRoute && routeHasPointElements {
					firstNonEmptyRouteChosen = true
				}
				insideRoute = false
			}
		}
	}

	if trackPointCount >= 2 {
		return trackParts, nil
	}
	if !firstNonEmptyRouteChosen || firstRoutePointCount < 2 {
		return nil, routingErrorFromCode("track_shape_unavailable", "trail has no usable GPX track or route")
	}
	if firstRoutePointCount > routingMaxManeuverInputPoints {
		return nil, routingErrorFromCode("track_limit_exceeded", "trail contains too many normalized route points")
	}
	return []pluginRoutingManeuverTrackPart{{Points: firstRoute}}, nil
}

func maneuverPointFromXML(element xml.StartElement) (pluginRoutingManeuverPoint, bool) {
	var latText, lonText string
	for _, attribute := range element.Attr {
		switch attribute.Name.Local {
		case "lat":
			latText = attribute.Value
		case "lon":
			lonText = attribute.Value
		}
	}
	lat, latErr := strconv.ParseFloat(strings.TrimSpace(latText), 64)
	lon, lonErr := strconv.ParseFloat(strings.TrimSpace(lonText), 64)
	if latErr != nil || lonErr != nil || math.IsNaN(lat) || math.IsInf(lat, 0) || math.IsNaN(lon) || math.IsInf(lon, 0) || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return pluginRoutingManeuverPoint{}, false
	}
	return pluginRoutingManeuverPoint{Lat: lat, Lon: lon}, true
}

func callRoutingManeuverPlugin(ctx context.Context, plugin pluginsystem.LocalPlugin, capability pluginsystem.CapabilityManifest, instance *core.Record, auth map[string]any, config map[string]any, request pluginRoutingManeuverRequest) (pluginRoutingManeuverOutput, error) {
	var output pluginRoutingManeuverOutput
	err := callRoutingPluginWithLimits(ctx, plugin, capability, instance, auth, config, request, &output, routingMaxManeuverHostRequests, routingMaxResponseBodyBytes)
	return output, err
}

func callPreparedRoutingManeuverPlugin(ctx context.Context, runtime routingEngineRuntime, request pluginRoutingManeuverRequest) (pluginRoutingManeuverOutput, error) {
	output, err := routingManeuverPluginCaller(ctx, runtime.Plugin, runtime.Capability, runtime.Instance, runtime.Auth, runtime.Config, request)
	if request.Profile.PreparedKey == "" || !routingManeuverPreparedProfileRetryable(output, err) {
		return output, err
	}
	refresh := refreshPreparedRoutingProfile(ctx, runtime, request.Profile.PreparedKey, runtime.Request)
	request.Profile.PreparedKey = refresh.key
	return routingManeuverPluginCaller(ctx, runtime.Plugin, runtime.Capability, runtime.Instance, runtime.Auth, runtime.Config, request)
}

func routingManeuverPreparedProfileRetryable(output pluginRoutingManeuverOutput, err error) bool {
	if output.Error != nil {
		return output.Error.Code == "unsupported_profile"
	}
	if err == nil {
		return false
	}
	var callErr pluginsystem.PluginCallError
	if errors.As(err, &callErr) {
		return callErr.PluginError.Code == "unsupported_profile"
	}
	var routingErr *routingError
	return errors.As(err, &routingErr) && routingErr.Code == "unsupported_profile"
}

var maneuverTypes = map[string]struct{}{
	"start": {}, "destination": {}, "continue": {},
	"turn_left": {}, "turn_right": {}, "turn_slight_left": {}, "turn_slight_right": {},
	"turn_sharp_left": {}, "turn_sharp_right": {}, "keep_left": {}, "keep_right": {},
	"uturn_left": {}, "uturn_right": {}, "uturn": {}, "roundabout_enter": {},
	"roundabout_exit": {}, "exit_left": {}, "exit_right": {}, "ramp_straight": {},
	"ramp_left": {}, "ramp_right": {}, "merge": {}, "merge_left": {}, "merge_right": {},
	"ferry": {}, "unknown": {},
}

var maneuverResponseWarnings = map[string]struct{}{
	"input_simplified": {}, "geometry_simplified": {}, "segment_gap_bridged": {},
}

var maneuverItemWarnings = map[string]struct{}{
	"maneuver_type_unknown": {},
}

func normalizeRoutingManeuverOutput(output *pluginRoutingManeuverOutput) error {
	points, geometryErr := decodeRoutingGeometry(output.Geometry)
	if geometryErr != nil {
		return routingErrorFromCode("invalid_plugin_response", geometryErr.Message)
	}
	if len(output.Maneuvers) < 2 || len(output.Maneuvers) > routingMaxManeuvers {
		return routingErrorFromCode("invalid_plugin_response", "maneuver plugin must return between 2 and 1000 maneuvers")
	}
	warnings, err := normalizedManeuverWarnings(output.Warnings, maneuverResponseWarnings)
	if err != nil {
		return err
	}
	output.Warnings = warnings
	lastGeometryIndex := len(points) - 1
	for index := range output.Maneuvers {
		maneuver := &output.Maneuvers[index]
		if _, ok := maneuverTypes[maneuver.Type]; !ok {
			return routingErrorFromCode("invalid_plugin_response", "maneuver plugin returned an unknown maneuver type")
		}
		if maneuver.Type == "unknown" && !containsRoutingString(maneuver.Warnings, "maneuver_type_unknown") {
			return routingErrorFromCode("invalid_plugin_response", "unknown maneuvers must include maneuver_type_unknown")
		}
		if !finiteNonnegative(maneuver.DistanceMeters) {
			return routingErrorFromCode("invalid_plugin_response", "maneuver distanceMeters must be finite and non-negative")
		}
		if maneuver.DurationSeconds != nil && !finiteNonnegative(*maneuver.DurationSeconds) {
			return routingErrorFromCode("invalid_plugin_response", "maneuver durationSeconds must be finite and non-negative")
		}
		if maneuver.BeginShapeIndex < 0 || maneuver.BeginShapeIndex > maneuver.EndShapeIndex || maneuver.EndShapeIndex > lastGeometryIndex {
			return routingErrorFromCode("invalid_plugin_response", "maneuver geometry interval is invalid")
		}
		if maneuver.BeginShapeIndex == maneuver.EndShapeIndex && maneuver.DistanceMeters != 0 {
			return routingErrorFromCode("invalid_plugin_response", "degenerate maneuver intervals must have zero distance")
		}
		if index > 0 && maneuver.BeginShapeIndex != output.Maneuvers[index-1].EndShapeIndex {
			return routingErrorFromCode("invalid_plugin_response", "maneuver intervals must form a complete partition")
		}
		if err := validateOptionalBearing(maneuver.BearingBefore); err != nil {
			return err
		}
		if err := validateOptionalBearing(maneuver.BearingAfter); err != nil {
			return err
		}
		if maneuver.RoundaboutExit != nil && *maneuver.RoundaboutExit < 1 {
			return routingErrorFromCode("invalid_plugin_response", "roundaboutExit must be positive")
		}
		if maneuver.RoundaboutExit != nil && maneuver.Type != "roundabout_enter" && maneuver.Type != "roundabout_exit" {
			return routingErrorFromCode("invalid_plugin_response", "roundaboutExit is only valid for roundabout maneuvers")
		}
		if utf8.RuneCountInString(maneuver.ProviderInstruction) > routingMaxProviderInstructionChars {
			return routingErrorFromCode("invalid_plugin_response", "providerInstruction exceeds the maneuver limit")
		}
		if len(maneuver.StreetNames) > routingMaxManeuverStreetNames {
			return routingErrorFromCode("invalid_plugin_response", "maneuver has too many street names")
		}
		for _, name := range maneuver.StreetNames {
			if utf8.RuneCountInString(name) > routingMaxManeuverStreetNameChars {
				return routingErrorFromCode("invalid_plugin_response", "maneuver street name exceeds the character limit")
			}
		}
		itemWarnings, err := normalizedManeuverWarnings(maneuver.Warnings, maneuverItemWarnings)
		if err != nil {
			return err
		}
		maneuver.Warnings = itemWarnings
	}
	first := output.Maneuvers[0]
	last := output.Maneuvers[len(output.Maneuvers)-1]
	if first.Type != "start" || first.BeginShapeIndex != 0 {
		return routingErrorFromCode("invalid_plugin_response", "first maneuver must start at geometry index zero")
	}
	if last.Type != "destination" || last.BeginShapeIndex != last.EndShapeIndex || last.EndShapeIndex != lastGeometryIndex {
		return routingErrorFromCode("invalid_plugin_response", "final maneuver must be a degenerate destination at the final geometry point")
	}
	return nil
}

func normalizedManeuverWarnings(values []string, allowed map[string]struct{}) ([]string, error) {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := allowed[value]; !ok {
			return nil, routingErrorFromCode("invalid_plugin_response", "maneuver plugin returned an unknown warning code")
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result, nil
}

func validateOptionalBearing(value *float64) error {
	if value != nil && (math.IsNaN(*value) || math.IsInf(*value, 0) || *value < 0 || *value >= 360) {
		return routingErrorFromCode("invalid_plugin_response", "maneuver bearings must be finite degrees from zero up to 360")
	}
	return nil
}

func finiteNonnegative(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
}

// Marshal once in tests and adapters to make the absence of trail/share/raw
// GPX fields at the plugin boundary straightforward to assert.
func marshalRoutingManeuverRequest(request pluginRoutingManeuverRequest) ([]byte, error) {
	return json.Marshal(request)
}
