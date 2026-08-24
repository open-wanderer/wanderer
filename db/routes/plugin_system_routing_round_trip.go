package routes

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"pocketbase/pluginsystem"
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

const (
	routingRoundTripMinTargetMeters  = 1000.0
	routingRoundTripMaxTargetMeters  = 300000.0
	routingRoundTripMaxSynthetic     = 8
	routingRoundTripFallbackAnchors  = 4
	routingRoundTripStartMeters      = 100.0
	routingRoundTripClosureMeters    = 10.0
	routingRoundTripSuggestionMeters = 250.0
	routingRoundTripMaxAttempts      = 16
)

var (
	routingRoundTripRateLimiter  = util.NewRateLimiter(10, time.Minute)
	routingRoundTripPluginCaller = callRoutingRoundTripPlugin
)

func PluginSystemRoutingRoundTrip(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}
	var data pluginRoutingRoundTripHTTPInput
	if err := e.BindBody(&data); err != nil {
		return apis.NewBadRequestError("Failed to read request data", err)
	}
	if strings.TrimSpace(data.PluginID) == "" {
		return routingJSONError(e, routingErrorFromCode("routing_plugin_required", "round-trip routing plugin is required"), nil)
	}
	if err := validateRoutingRoundTripRequest(data.pluginRoutingRoundTripRequest); err != nil {
		return routingJSONError(e, err, nil)
	}
	settings, err := ResolveRoutingSettings(e.App, e.Auth.Id)
	if err != nil {
		return err
	}
	if !routingFeatureEnabled(settings, "routing", true) {
		return routingJSONError(e, routingErrorFromCode("routing_disabled", "routing is not enabled"), nil)
	}
	if err := routingRoundTripRateLimiter.CheckRateLimit(e.Auth.Id, "round-trip"); err != nil {
		return e.TooManyRequestsError("too many round-trip requests", err)
	}

	runtime, err := routingRuntimeForSelection(e, routingEngineSelection{
		PluginID: data.PluginID, InstanceID: data.InstanceID,
	}, "round_trip")
	if err != nil {
		return routingJSONError(e, err, nil)
	}
	if !routingManifestBool(runtime.Plugin, "supportsRoundTrip") {
		return routingJSONError(e, routingErrorFromCode("routing_mode_unavailable", "selected routing engine does not advertise round-trip support"), nil)
	}
	clientRequest := pluginRoutingRouteRequest{
		RoutingMode:         "round_trip",
		Anchors:             []pluginRoutingAnchor{data.Start},
		Mode:                data.Mode,
		Category:            data.Category,
		Subcategory:         data.Subcategory,
		Profile:             data.Profile,
		Preferences:         cloneRoutingMap(data.Preferences),
		RequiredPreferences: append([]string(nil), data.RequiredPreferences...),
		Options:             data.Options,
	}
	clientRequest.Profile.PreparedKey = ""
	resolvedRequest := cloneRoutingRouteRequest(clientRequest)
	if err := applyRoutingCategoryMappingContext(e.Request.Context(), e.App, e.Auth.Id, runtime.Plugin.Manifest.ID, instanceID(runtime.Instance), &resolvedRequest, &settings); err != nil {
		return routingJSONError(e, err, nil)
	}
	restrictRoutingPluginPreferences(runtime.Plugin, &resolvedRequest)
	runtime.Request = resolvedRequest
	runtime.ClientRequest = clientRequest
	routingCtx, closeRoutingSessions := withRoutingPluginSessions(e.Request.Context())
	defer closeRoutingSessions()
	runtime, err = prepareRoutingRuntimeProfile(routingCtx, runtime)
	if err != nil {
		return routingJSONError(e, err, nil)
	}

	request := pluginRoutingRoundTripRequest{
		Start:               data.Start,
		TargetDistance:      data.TargetDistance,
		Direction:           data.Direction,
		Seed:                data.Seed,
		Mode:                runtime.Request.Mode,
		Category:            runtime.Request.Category,
		Subcategory:         runtime.Request.Subcategory,
		Profile:             runtime.Request.Profile,
		Preferences:         cloneRoutingMap(runtime.Request.Preferences),
		RequiredPreferences: append([]string(nil), runtime.Request.RequiredPreferences...),
		Options:             runtime.Request.Options,
	}
	output, err := callPreparedRoutingRoundTripPlugin(routingCtx, runtime, request)
	if err != nil {
		return routingJSONError(e, routingErrorFromCall(err), nil)
	}
	if output.Error != nil {
		return routingJSONError(e, routingErrorFromPluginError(*output.Error), nil)
	}
	if len(output.Candidates) == 0 {
		return routingJSONError(e, routingErrorFromCode("no_route", "round-trip engine returned no route"), nil)
	}

	pluginCandidates := output.Candidates
	if len(pluginCandidates) > 1 {
		pluginCandidates = pluginCandidates[:1]
		output.Warnings = append(output.Warnings, "round_trip_candidates_truncated")
	}
	materialized := make([]pluginRoutingCandidate, 0, len(pluginCandidates))
	engineErrors := append([]pluginRoutingEngineError(nil), output.EngineErrors...)
	requestID := newRoutingRoundTripRequestID()
	for index := range pluginCandidates {
		candidate, materializeErr := materializeRoutingRoundTripCandidate(
			pluginCandidates[index], request, runtime.Request, runtime.ClientRequest, runtime.Plugin, runtime.Instance, requestID, index,
		)
		if materializeErr != nil {
			engineErrors = append(engineErrors, routingEngineErrorFromError(materializeErr, runtime.Plugin, runtime.Instance))
			continue
		}
		materialized = append(materialized, candidate)
	}
	if len(materialized) == 0 {
		return routingJSONError(e, routingNoCandidateError(engineErrors), map[string]any{"engineErrors": engineErrors})
	}
	return e.JSON(http.StatusOK, pluginRoutingRouteOutput{
		Candidates: materialized, EngineErrors: engineErrors, Warnings: uniqueRoutingStrings(output.Warnings),
	})
}

func validateRoutingRoundTripRequest(request pluginRoutingRoundTripRequest) error {
	if !validRoutingCoordinate(request.Start) {
		return routingErrorFromCode("invalid_coordinate", "round-trip start must be a valid WGS84 coordinate")
	}
	if request.TargetDistance < routingRoundTripMinTargetMeters || request.TargetDistance > routingRoundTripMaxTargetMeters || math.IsNaN(request.TargetDistance) || math.IsInf(request.TargetDistance, 0) {
		return routingErrorFromCode("invalid_request", "round-trip targetDistance must be between 1000 and 300000 meters")
	}
	if request.Direction != nil && (math.IsNaN(*request.Direction) || math.IsInf(*request.Direction, 0) || *request.Direction < 0 || *request.Direction >= 360) {
		return routingErrorFromCode("invalid_request", "round-trip direction must be between 0 and 360 degrees")
	}
	if len(request.Seed) > 128 {
		return routingErrorFromCode("invalid_request", "round-trip seed exceeds 128 characters")
	}
	if err := validateRoutingInlineProfileContent(request.Profile.ContentBase64); err != nil {
		return err
	}
	if err := validateRoutingPreferences(request.Preferences, false); err != nil {
		return routingErrorFromCode("invalid_request", "invalid routing preferences: "+err.Error())
	}
	if err := validateRoutingConfig(request.Profile.NativeConfig, "profile.nativeConfig"); err != nil {
		return routingErrorFromCode("invalid_request", err.Error())
	}
	if err := validateRoutingRequiredPreferences(request.RequiredPreferences); err != nil {
		return routingErrorFromCode("invalid_request", err.Error())
	}
	return nil
}

func callRoutingRoundTripPlugin(ctx context.Context, plugin pluginsystem.LocalPlugin, capability pluginsystem.CapabilityManifest, instance *core.Record, auth map[string]any, config map[string]any, request pluginRoutingRoundTripRequest) (pluginRoutingRouteOutput, error) {
	var output pluginRoutingRouteOutput
	err := callRoutingPlugin(ctx, plugin, capability, instance, auth, config, request, &output)
	return output, err
}

func callPreparedRoutingRoundTripPlugin(ctx context.Context, runtime routingEngineRuntime, request pluginRoutingRoundTripRequest) (pluginRoutingRouteOutput, error) {
	output, err := routingRoundTripPluginCaller(ctx, runtime.Plugin, runtime.Capability, runtime.Instance, runtime.Auth, runtime.Config, request)
	if request.Profile.PreparedKey == "" || !routingPreparedProfileRetryable(output, err) {
		return output, err
	}
	refresh := refreshPreparedRoutingProfile(ctx, runtime, request.Profile.PreparedKey, runtime.Request)
	request.Profile.PreparedKey = refresh.key
	return routingRoundTripPluginCaller(ctx, runtime.Plugin, runtime.Capability, runtime.Instance, runtime.Auth, runtime.Config, request)
}

func materializeRoutingRoundTripCandidate(candidate pluginRoutingCandidate, request pluginRoutingRoundTripRequest, resolved pluginRoutingRouteRequest, client pluginRoutingRouteRequest, plugin pluginsystem.LocalPlugin, instance *core.Record, requestID string, index int) (pluginRoutingCandidate, error) {
	if candidate.Geometry == nil {
		return pluginRoutingCandidate{}, routingErrorFromCode("invalid_candidate", "round-trip candidate must include full geometry")
	}
	points, geometryErr := decodeRoutingGeometry(*candidate.Geometry)
	if geometryErr != nil {
		return pluginRoutingCandidate{}, geometryErr
	}
	if len(points) < 3 {
		return pluginRoutingCandidate{}, routingErrorFromCode("invalid_candidate", "round-trip geometry must contain at least three points")
	}
	if candidate.Summary.Distance < 0 || candidate.Summary.Duration < 0 || candidate.Summary.ElevationGain < 0 || candidate.Summary.ElevationLoss < 0 {
		return pluginRoutingCandidate{}, routingErrorFromCode("invalid_candidate", "round-trip summary values must not be negative")
	}
	startDistance := routingCoordinateDistanceMeters(points[0][0], points[0][1], request.Start.Lat, request.Start.Lon)
	closureDistance := routingCoordinateDistanceMeters(points[0][0], points[0][1], points[len(points)-1][0], points[len(points)-1][1])
	if startDistance > routingRoundTripStartMeters || closureDistance > routingRoundTripClosureMeters {
		return pluginRoutingCandidate{}, routingErrorFromCode("invalid_candidate", "round-trip geometry is not closed at the requested start")
	}
	if err := validateRoutingRoundTripMetadata(candidate.RoundTrip); err != nil {
		return pluginRoutingCandidate{}, err
	}
	if candidate.Elevation != nil {
		normalizeRoutingCandidateElevation(candidate.Elevation)
		if len(candidate.Elevation.Heights) > 0 && len(candidate.Elevation.Heights) != len(points) {
			return pluginRoutingCandidate{}, routingErrorFromCode("invalid_candidate", "round-trip elevation heights must match geometry point count")
		}
	}

	splits := routingRoundTripSuggestedSplits(points, candidate.SuggestedAnchors)
	if len(splits) == 0 {
		splits = routingRoundTripDistanceSplits(points, routingRoundTripFallbackAnchors)
	}
	if len(splits) == 0 {
		return pluginRoutingCandidate{}, routingErrorFromCode("invalid_candidate", "round-trip geometry cannot be split into editable segments")
	}
	anchors := make([]pluginRoutingAnchor, 0, len(splits)+1)
	anchors = append(anchors, pluginRoutingAnchor{Lat: points[0][0], Lon: points[0][1]})
	for _, split := range splits {
		anchors = append(anchors, pluginRoutingAnchor{Lat: points[split][0], Lon: points[split][1]})
	}
	actualDistance := candidate.Summary.Distance
	geometryDistance := routingPointDistance(points)
	if actualDistance == 0 {
		actualDistance = geometryDistance
		candidate.Summary.Distance = actualDistance
	}
	metadata := candidate.RoundTrip
	if metadata == nil {
		metadata = &pluginRoutingRoundTripMetadata{}
	}
	if metadata.Direction == nil && request.Direction != nil {
		direction := int(math.Floor(*request.Direction+0.5)) % 360
		metadata.Direction = &direction
	}
	metadata.RequestID = requestID
	metadata.TargetDistance = request.TargetDistance
	metadata.ActualDistance = actualDistance
	metadata.Seed = request.Seed
	candidate.RoundTrip = metadata
	candidate.SnappedAnchors = anchors
	candidate.SuggestedAnchors = nil
	candidate.Segments = make([]pluginRoutingSegment, 0, len(anchors))
	boundaries := append([]int{0}, splits...)
	boundaries = append(boundaries, len(points)-1)
	for segmentIndex := 0; segmentIndex < len(anchors); segmentIndex++ {
		start, end := boundaries[segmentIndex], boundaries[segmentIndex+1]
		part := points[start : end+1]
		partDistance := routingPointDistance(part)
		distance := partDistance
		duration := 0.0
		if geometryDistance > 0 {
			distance = actualDistance * partDistance / geometryDistance
			duration = candidate.Summary.Duration * partDistance / geometryDistance
		}
		candidate.Segments = append(candidate.Segments, pluginRoutingSegment{
			FromAnchor: segmentIndex,
			ToAnchor:   (segmentIndex + 1) % len(anchors),
			Geometry: pluginRoutingGeometry{
				Format: routingPolylineFormat, Precision: routingPolylinePrecision,
				Coordinates: string(routingPolylineCodec.EncodeCoords(nil, part)),
			},
			Distance: distance,
			Duration: duration,
			Provenance: &pluginRoutingSegmentProvenance{
				Source: "round_trip", RouteTopology: "closed_loop", RoundTripRequestID: requestID,
				RoundTripTargetMeters: request.TargetDistance, RoundTripActualMeters: actualDistance,
				RoundTripDirection: metadata.Direction, RoundTripSeed: request.Seed,
				SyntheticFromAnchor: segmentIndex != 0, SyntheticToAnchor: (segmentIndex+1)%len(anchors) != 0,
				Category: resolved.Category, Subcategory: resolved.Subcategory,
				Preferences: cloneRoutingMap(resolved.Preferences), RequestedPreferences: cloneRoutingMap(client.Preferences),
				PluginID: plugin.Manifest.ID, InstanceID: instanceID(instance), Provider: plugin.Manifest.Name,
				ProfileID: resolved.Profile.ID, ProfileKey: resolved.Profile.Key, ProfileKind: resolved.Profile.Kind,
				NativeConfig: cloneRoutingMap(resolved.Profile.NativeConfig), RequestedNativeConfig: cloneRoutingMap(client.Profile.NativeConfig),
				ProfileRevision: routingResolvedProfileRevision(resolved.Profile),
			},
		})
	}
	candidate.ID = fmt.Sprintf("%s:%s:round-trip:%d", plugin.Manifest.ID, instanceID(instance), index)
	candidate.Provider = plugin.Manifest.Name
	candidate.PluginID = plugin.Manifest.ID
	candidate.InstanceID = instanceID(instance)
	candidate.CompositionMode = "round_trip"
	candidate.Warnings = uniqueRoutingStrings(candidate.Warnings)
	return candidate, nil
}

func validateRoutingRoundTripMetadata(metadata *pluginRoutingRoundTripMetadata) error {
	if metadata == nil {
		return nil
	}
	if metadata.Direction != nil && (*metadata.Direction < 0 || *metadata.Direction >= 360) {
		return routingErrorFromCode("invalid_candidate", "round-trip direction metadata must be between 0 and 359 degrees")
	}
	if metadata.Attempts < 1 || metadata.Attempts > routingRoundTripMaxAttempts {
		return routingErrorFromCode("invalid_candidate", "round-trip attempts metadata is outside the supported range")
	}
	if math.IsNaN(metadata.Tolerance) || math.IsInf(metadata.Tolerance, 0) || metadata.Tolerance < 0 || metadata.Tolerance > 1 {
		return routingErrorFromCode("invalid_candidate", "round-trip tolerance metadata must be between 0 and 1")
	}
	return nil
}

func routingRoundTripSuggestedSplits(points [][]float64, suggestions []pluginRoutingAnchor) []int {
	if len(suggestions) == 0 || len(points) < 4 {
		return nil
	}
	result := make([]int, 0, min(len(suggestions), routingRoundTripMaxSynthetic))
	last := 0
	for _, suggestion := range suggestions {
		if len(result) >= routingRoundTripMaxSynthetic || !validRoutingCoordinate(suggestion) {
			break
		}
		best, bestDistance := -1, math.Inf(1)
		for pointIndex := last + 1; pointIndex < len(points)-1; pointIndex++ {
			distance := routingCoordinateDistanceMeters(points[pointIndex][0], points[pointIndex][1], suggestion.Lat, suggestion.Lon)
			if distance < bestDistance {
				best, bestDistance = pointIndex, distance
			}
		}
		if best > last && bestDistance <= routingRoundTripSuggestionMeters {
			result = append(result, best)
			last = best
		}
	}
	return result
}

func routingRoundTripDistanceSplits(points [][]float64, desired int) []int {
	if len(points) < 4 || desired < 1 {
		return nil
	}
	desired = min(desired, routingRoundTripMaxSynthetic)
	total := routingPointDistance(points)
	if total <= 0 {
		return nil
	}
	result := make([]int, 0, desired)
	cumulative := 0.0
	nextTarget := total / float64(desired+1)
	for pointIndex := 1; pointIndex < len(points)-1 && len(result) < desired; pointIndex++ {
		cumulative += routingCoordinateDistanceMeters(points[pointIndex-1][0], points[pointIndex-1][1], points[pointIndex][0], points[pointIndex][1])
		if cumulative >= nextTarget {
			result = append(result, pointIndex)
			nextTarget = total * float64(len(result)+1) / float64(desired+1)
		}
	}
	return result
}

func routingPointDistance(points [][]float64) float64 {
	distance := 0.0
	for index := 1; index < len(points); index++ {
		distance += routingCoordinateDistanceMeters(points[index-1][0], points[index-1][1], points[index][0], points[index][1])
	}
	return distance
}

func newRoutingRoundTripRequestID() string {
	var value [8]byte
	if _, err := rand.Read(value[:]); err == nil {
		return "rt_" + hex.EncodeToString(value[:])
	}
	return fmt.Sprintf("rt_%x", time.Now().UnixNano())
}
