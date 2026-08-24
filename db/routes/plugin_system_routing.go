package routes

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"pocketbase/pluginsystem"
	"pocketbase/services/pluginhost"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/twpayne/go-polyline"
)

const (
	routingPolylineFormat    = "encoded_polyline"
	routingPolylinePrecision = 6
	routingPolylineScale     = 1e6
	routingMaxAnchors        = 50
	routingMaxVariantAnchors = 16
	routingMaxVariants       = 3
	// Providers count their primary route as one candidate. Variant requests
	// may therefore fetch one extra candidate so the public limit still means
	// three alternatives to the current route.
	routingMaxProviderCandidates = routingMaxVariants + 1
	routingMaxCandidateSet       = 12
	routingMaxEngines            = 4
	routingMaxParallelCalls      = 8
	routingMaxPolylinePoints     = 20000
	routingMaxElevationPoints    = 20000
	routingMaxResponseBodyBytes  = 4 * 1024 * 1024
	// A routing call may prepare provider-owned state before fetching bounded
	// candidates. Keep a small reserve while enforcing one uniform connector
	// request budget for every routing plugin.
	routingMaxHostRequests = routingMaxProviderCandidates + 2
	routingPluginTimeout   = 30 * time.Second
)

var routingPolylineCodec = polyline.Codec{Dim: 2, Scale: routingPolylineScale}
var routingRuntimeSessionOpener = func(ctx context.Context, plugin pluginsystem.LocalPlugin, policy pluginsystem.RequestPolicyContext) (pluginsystem.RuntimeSession, error) {
	runtime, err := pluginsystem.NewRuntimeRegistry().RuntimeFor(plugin)
	if err != nil {
		return nil, err
	}
	return runtime.OpenSession(ctx, plugin, policy)
}

type pluginRoutingRouteInput struct {
	Instance pluginsystem.InstanceRef  `json:"instance"`
	Auth     map[string]any            `json:"auth,omitempty"`
	Config   map[string]any            `json:"config,omitempty"`
	Request  pluginRoutingRouteRequest `json:"request"`
}

type pluginRoutingElevationInput struct {
	Instance pluginsystem.InstanceRef      `json:"instance"`
	Auth     map[string]any                `json:"auth,omitempty"`
	Config   map[string]any                `json:"config,omitempty"`
	Request  pluginRoutingElevationRequest `json:"request"`
}

type pluginRoutingProfileIntrospectInput struct {
	Instance pluginsystem.InstanceRef              `json:"instance"`
	Auth     map[string]any                        `json:"auth,omitempty"`
	Config   map[string]any                        `json:"config,omitempty"`
	Request  pluginRoutingProfileIntrospectRequest `json:"request"`
}

type pluginRoutingProfilePrepareRequest struct {
	Mode                string               `json:"mode,omitempty"`
	Profile             pluginRoutingProfile `json:"profile"`
	Preferences         map[string]any       `json:"preferences,omitempty"`
	RequiredPreferences []string             `json:"requiredPreferences,omitempty"`
}

type pluginRoutingRouteRequest struct {
	RoutingMode         string                `json:"routingMode"`
	Anchors             []pluginRoutingAnchor `json:"anchors"`
	Mode                string                `json:"mode,omitempty"`
	Category            string                `json:"category,omitempty"`
	Subcategory         string                `json:"subcategory,omitempty"`
	Profile             pluginRoutingProfile  `json:"profile"`
	Preferences         map[string]any        `json:"preferences,omitempty"`
	RequiredPreferences []string              `json:"requiredPreferences,omitempty"`
	Options             pluginRoutingOptions  `json:"options,omitempty"`
}

type pluginRoutingRoundTripRequest struct {
	Start               pluginRoutingAnchor  `json:"start"`
	TargetDistance      float64              `json:"targetDistance"`
	Direction           *float64             `json:"direction,omitempty"`
	Seed                string               `json:"seed,omitempty"`
	Mode                string               `json:"mode,omitempty"`
	Category            string               `json:"category,omitempty"`
	Subcategory         string               `json:"subcategory,omitempty"`
	Profile             pluginRoutingProfile `json:"profile"`
	Preferences         map[string]any       `json:"preferences,omitempty"`
	RequiredPreferences []string             `json:"requiredPreferences,omitempty"`
	Options             pluginRoutingOptions `json:"options,omitempty"`
}

type pluginRoutingElevationRequest struct {
	EncodedPolyline string                `json:"encodedPolyline,omitempty"`
	Coordinates     []pluginRoutingAnchor `json:"coordinates,omitempty"`
}

type pluginRoutingProfileIntrospectRequest struct {
	Profile pluginRoutingProfile `json:"profile"`
}

type pluginRoutingAnchor struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type pluginRoutingProfile struct {
	ID            string         `json:"id,omitempty"`
	PluginID      string         `json:"pluginId,omitempty"`
	Key           string         `json:"key"`
	Kind          string         `json:"kind,omitempty"`
	Mode          string         `json:"mode,omitempty"`
	ContentBase64 string         `json:"contentBase64,omitempty"`
	ContentType   string         `json:"contentType,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	NativeConfig  map[string]any `json:"nativeConfig,omitempty"`
	PreparedKey   string         `json:"preparedKey,omitempty"`
}

type pluginRoutingOptions struct {
	Alternatives     int  `json:"alternatives,omitempty"`
	IncludeElevation bool `json:"includeElevation,omitempty"`
}

type pluginRoutingRouteOutput struct {
	Candidates   []pluginRoutingCandidate   `json:"candidates,omitempty"`
	EngineErrors []pluginRoutingEngineError `json:"engineErrors,omitempty"`
	Warnings     []string                   `json:"warnings,omitempty"`
	Error        *pluginsystem.PluginError  `json:"error,omitempty"`
}

type pluginRoutingElevationOutput struct {
	Heights []float64                 `json:"heights,omitempty"`
	Status  string                    `json:"status,omitempty"`
	Error   *pluginsystem.PluginError `json:"error,omitempty"`
}

type pluginRoutingProfileIntrospectOutput struct {
	NativeControlGroups  []routingNativeControlGroup `json:"nativeControlGroups,omitempty"`
	Mode                 string                      `json:"mode,omitempty"`
	SupportedPreferences []string                    `json:"supportedPreferences,omitempty"`
	Metadata             map[string]any              `json:"metadata,omitempty"`
	Error                *pluginsystem.PluginError   `json:"error,omitempty"`
}

type pluginRoutingProfilePrepareOutput struct {
	PreparedKey string                    `json:"preparedKey,omitempty"`
	Error       *pluginsystem.PluginError `json:"error,omitempty"`
}

type pluginRoutingEngineError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	PluginID   string `json:"pluginId,omitempty"`
	InstanceID string `json:"instanceId,omitempty"`
	Provider   string `json:"provider,omitempty"`
	HTTPStatus int    `json:"-"`
}

type pluginRoutingCandidate struct {
	ID               string                          `json:"id"`
	ProfileKey       string                          `json:"profileKey,omitempty"`
	Geometry         *pluginRoutingGeometry          `json:"geometry,omitempty"`
	Elevation        *pluginRoutingElevation         `json:"elevation,omitempty"`
	Summary          pluginRoutingSummary            `json:"summary"`
	Segments         []pluginRoutingSegment          `json:"segments"`
	SnappedAnchors   []pluginRoutingAnchor           `json:"snappedAnchors,omitempty"`
	SuggestedAnchors []pluginRoutingAnchor           `json:"suggestedAnchors,omitempty"`
	RoundTrip        *pluginRoutingRoundTripMetadata `json:"roundTrip,omitempty"`
	Warnings         []string                        `json:"warnings,omitempty"`
	Provider         string                          `json:"provider,omitempty"`
	PluginID         string                          `json:"pluginId,omitempty"`
	InstanceID       string                          `json:"instanceId,omitempty"`
	CompositionMode  string                          `json:"compositionMode,omitempty"`
}

type pluginRoutingRoundTripMetadata struct {
	RequestID      string  `json:"requestId,omitempty"`
	TargetDistance float64 `json:"targetDistance"`
	ActualDistance float64 `json:"actualDistance"`
	Direction      *int    `json:"direction,omitempty"`
	Seed           string  `json:"seed,omitempty"`
	Attempts       int     `json:"attempts"`
	Tolerance      float64 `json:"tolerance"`
}

type pluginRoutingElevation struct {
	Heights []float64 `json:"heights,omitempty"`
	Status  string    `json:"status,omitempty"`
	Source  string    `json:"source,omitempty"`
}

type pluginRoutingGeometry struct {
	Format      string `json:"format"`
	Precision   int    `json:"precision"`
	Coordinates string `json:"coordinates"`
}

type pluginRoutingSummary struct {
	Distance      float64 `json:"distance"`
	Duration      float64 `json:"duration"`
	ElevationGain float64 `json:"elevationGain,omitempty"`
	ElevationLoss float64 `json:"elevationLoss,omitempty"`
}

type pluginRoutingSegment struct {
	FromAnchor int                             `json:"fromAnchor"`
	ToAnchor   int                             `json:"toAnchor"`
	Geometry   pluginRoutingGeometry           `json:"geometry"`
	Distance   float64                         `json:"distance"`
	Duration   float64                         `json:"duration"`
	Provenance *pluginRoutingSegmentProvenance `json:"provenance,omitempty"`
}

type pluginRoutingSegmentProvenance struct {
	Source                string         `json:"source,omitempty"`
	RouteTopology         string         `json:"routeTopology,omitempty"`
	RoundTripRequestID    string         `json:"roundTripRequestId,omitempty"`
	RoundTripTargetMeters float64        `json:"roundTripTargetMeters,omitempty"`
	RoundTripActualMeters float64        `json:"roundTripActualMeters,omitempty"`
	RoundTripDirection    *int           `json:"roundTripDirection,omitempty"`
	RoundTripSeed         string         `json:"roundTripSeed,omitempty"`
	SyntheticFromAnchor   bool           `json:"syntheticFromAnchor,omitempty"`
	SyntheticToAnchor     bool           `json:"syntheticToAnchor,omitempty"`
	Category              string         `json:"category,omitempty"`
	Subcategory           string         `json:"subcategory,omitempty"`
	RoutingMode           string         `json:"routingMode,omitempty"`
	Preferences           map[string]any `json:"preferences,omitempty"`
	RequestedPreferences  map[string]any `json:"requestedPreferences,omitempty"`
	PluginID              string         `json:"pluginId,omitempty"`
	InstanceID            string         `json:"instanceId,omitempty"`
	Provider              string         `json:"provider,omitempty"`
	ProfileID             string         `json:"profileId,omitempty"`
	ProfileKey            string         `json:"profileKey,omitempty"`
	ProfileKind           string         `json:"profileKind,omitempty"`
	NativeConfig          map[string]any `json:"nativeConfig,omitempty"`
	RequestedNativeConfig map[string]any `json:"requestedNativeConfig,omitempty"`
	ProfileRevision       string         `json:"profileRevision,omitempty"`
}

type pluginRoutingRouteHTTPInput struct {
	PluginID          string                   `json:"pluginId,omitempty"`
	InstanceID        string                   `json:"instanceId,omitempty"`
	Engines           []routingEngineSelection `json:"engines,omitempty"`
	EngineMode        string                   `json:"engineMode,omitempty"`
	DesiredVariants   int                      `json:"desiredVariants,omitempty"`
	RequestVariants   bool                     `json:"requestVariants,omitempty"`
	ReferenceGeometry *pluginRoutingGeometry   `json:"referenceGeometry,omitempty"`
	pluginRoutingRouteRequest
}

type pluginRoutingRoundTripHTTPInput struct {
	PluginID   string `json:"pluginId"`
	InstanceID string `json:"instanceId,omitempty"`
	pluginRoutingRoundTripRequest
}

type pluginRoutingCheckHTTPInput struct {
	PluginID string         `json:"pluginId"`
	Config   map[string]any `json:"config,omitempty"`
}

type pluginRoutingElevationHTTPInput struct {
	PluginID   string `json:"pluginId,omitempty"`
	InstanceID string `json:"instanceId,omitempty"`
	pluginRoutingElevationRequest
}

func PluginSystemRoutingRoute(e *core.RequestEvent) error {
	return pluginSystemRoutingRoute(e, false)
}

func PluginSystemRoutingCheck(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}
	var data pluginRoutingCheckHTTPInput
	if err := e.BindBody(&data); err != nil {
		return apis.NewBadRequestError("Failed to read request data", err)
	}
	if strings.TrimSpace(data.PluginID) == "" {
		return apis.NewBadRequestError("routing plugin is required", nil)
	}
	plugin, capability, err := localPluginCapability(e.App, data.PluginID, "route", "v1")
	if err != nil {
		return err
	}
	if plugin.Manifest.Type != pluginsystem.PluginTypeRouting {
		return apis.NewBadRequestError("plugin is not a routing plugin", nil)
	}
	config := routingPluginCheckConfig(e.App, plugin, data.Config)
	request := routingPluginCheckRequest(plugin)
	output, err := callRoutingRoutePlugin(e.Request.Context(), plugin, capability, nil, map[string]any{}, config, request)
	if err != nil {
		routingErr := routingErrorFromCall(err)
		if routingPluginCheckAcceptsError(routingErr) {
			return e.JSON(http.StatusOK, map[string]any{"ok": true})
		}
		return routingJSONError(e, routingErr, nil)
	}
	if output.Error != nil {
		routingErr := routingErrorFromPluginError(*output.Error)
		if routingPluginCheckAcceptsError(routingErr) {
			return e.JSON(http.StatusOK, map[string]any{"ok": true})
		}
		return routingJSONError(e, routingErr, nil)
	}
	normalized, normalizeErr := normalizeRoutingRouteOutput(request, output, plugin, nil)
	if normalizeErr != nil {
		var routingErr *routingError
		if errors.As(normalizeErr, &routingErr) && routingPluginCheckAcceptsError(routingErr) {
			return e.JSON(http.StatusOK, map[string]any{"ok": true})
		}
		return routingJSONError(e, normalizeErr, nil)
	}
	return e.JSON(http.StatusOK, map[string]any{"ok": len(normalized.Candidates) > 0})
}

func routingPluginCheckAcceptsError(err *routingError) bool {
	if err == nil {
		return true
	}
	switch err.Code {
	case "provider_error", "provider_unavailable", "connector_error", "provider_timeout", "timeout":
		return false
	default:
		return true
	}
}

func PluginSystemRoutingElevation(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}
	settings, err := ResolveRoutingSettings(e.App, e.Auth.Id)
	if err != nil {
		return err
	}
	if !routingFeatureEnabled(settings, "routing", true) {
		return routingJSONError(e, routingErrorFromCode("routing_disabled", "routing is not enabled"), nil)
	}
	var data pluginRoutingElevationHTTPInput
	if err := e.BindBody(&data); err != nil {
		return apis.NewBadRequestError("Failed to read request data", err)
	}
	if data.PluginID == "" {
		data.PluginID = settings.ElevationPluginID
	}
	if data.PluginID == "" {
		return apis.NewBadRequestError("routing elevation plugin is required", nil)
	}
	if err := validateRoutingElevationRequest(data.pluginRoutingElevationRequest); err != nil {
		return routingJSONError(e, err, nil)
	}
	runtime, err := routingRuntimeForSelection(e, routingEngineSelection{
		PluginID: data.PluginID, InstanceID: data.InstanceID,
	}, "elevation")
	if err != nil {
		return err
	}
	output, err := callRoutingElevationPlugin(e.Request.Context(), runtime.Plugin, runtime.Capability, runtime.Instance, runtime.Auth, runtime.Config, data.pluginRoutingElevationRequest)
	if err != nil {
		return routingJSONError(e, routingErrorFromCall(err), nil)
	}
	if output.Error != nil {
		return routingJSONError(e, routingErrorFromPluginError(*output.Error), nil)
	}
	normalizeRoutingElevationOutput(&output)
	return e.JSON(http.StatusOK, output)
}

func routingPluginCheckConfig(app core.App, plugin pluginsystem.LocalPlugin, submitted map[string]any) map[string]any {
	config := effectiveRoutingPluginConfig(app, plugin, nil)
	if len(submitted) == 0 {
		return config
	}
	submittedConfig := pluginsystem.CloneJSONMap(submitted)
	delete(submittedConfig, "host")
	pluginsystem.DeepMergeConfig(config, submittedConfig)
	return config
}

func routingPluginCheckRequest(plugin pluginsystem.LocalPlugin) pluginRoutingRouteRequest {
	mode := "foot"
	metadataRouting := mapValue(plugin.Manifest.Metadata["routing"])
	if modes := stringSlice(metadataRouting["modes"]); len(modes) > 0 {
		mode = modes[0]
	}
	return pluginRoutingRouteRequest{
		RoutingMode: "segment",
		Anchors: []pluginRoutingAnchor{
			{Lat: 47.3769, Lon: 8.5417},
			{Lat: 47.3775, Lon: 8.5450},
		},
		Mode: mode,
		Profile: pluginRoutingProfile{
			PluginID: plugin.Manifest.ID,
			Kind:     "builtin",
		},
		Options: pluginRoutingOptions{
			Alternatives:     1,
			IncludeElevation: false,
		},
	}
}

func effectiveRoutingPluginConfig(app core.App, plugin pluginsystem.LocalPlugin, instance *core.Record) map[string]any {
	config := pluginhost.EffectiveConfig(app, plugin.Manifest.ID, instance)
	if len(config) == 0 {
		hostConfig, _ := pluginsystem.CloneJSONValue(plugin.Manifest.HostConfig).(map[string]any)
		if hostConfig == nil {
			hostConfig = map[string]any{}
		}
		config = map[string]any{"host": hostConfig, "plugin": map[string]any{}}
	}
	return config
}

func callRoutingRoutePlugin(ctx context.Context, plugin pluginsystem.LocalPlugin, capability pluginsystem.CapabilityManifest, instance *core.Record, auth map[string]any, config map[string]any, request pluginRoutingRouteRequest) (pluginRoutingRouteOutput, error) {
	var output pluginRoutingRouteOutput
	err := callRoutingPlugin(ctx, plugin, capability, instance, auth, config, request, &output)
	return output, err
}

func callRoutingElevationPlugin(ctx context.Context, plugin pluginsystem.LocalPlugin, capability pluginsystem.CapabilityManifest, instance *core.Record, auth map[string]any, config map[string]any, request pluginRoutingElevationRequest) (pluginRoutingElevationOutput, error) {
	var output pluginRoutingElevationOutput
	err := callRoutingPlugin(ctx, plugin, capability, instance, auth, config, request, &output)
	return output, err
}

func callRoutingProfileIntrospectPlugin(ctx context.Context, plugin pluginsystem.LocalPlugin, capability pluginsystem.CapabilityManifest, instance *core.Record, auth map[string]any, config map[string]any, request pluginRoutingProfileIntrospectRequest) (pluginRoutingProfileIntrospectOutput, error) {
	var output pluginRoutingProfileIntrospectOutput
	err := callRoutingPlugin(ctx, plugin, capability, instance, auth, config, request, &output)
	return output, err
}

func callRoutingProfilePreparePlugin(ctx context.Context, plugin pluginsystem.LocalPlugin, capability pluginsystem.CapabilityManifest, instance *core.Record, auth map[string]any, config map[string]any, request pluginRoutingProfilePrepareRequest) (pluginRoutingProfilePrepareOutput, error) {
	var output pluginRoutingProfilePrepareOutput
	err := callRoutingPlugin(ctx, plugin, capability, instance, auth, config, request, &output)
	return output, err
}

func callRoutingPlugin(ctx context.Context, plugin pluginsystem.LocalPlugin, capability pluginsystem.CapabilityManifest, instance *core.Record, auth map[string]any, config map[string]any, request any, output any) error {
	return callRoutingPluginWithLimits(ctx, plugin, capability, instance, auth, config, request, output, routingMaxHostRequests, routingMaxResponseBodyBytes)
}

func callRoutingPluginWithLimits(ctx context.Context, plugin pluginsystem.LocalPlugin, capability pluginsystem.CapabilityManifest, instance *core.Record, auth map[string]any, config map[string]any, request any, output any, maxHostRequests int, maxResponseBytes int) error {
	ctx, cancel := context.WithTimeout(ctx, routingPluginTimeout)
	defer cancel()
	policy := pluginhost.InstancePolicy(plugin, config).WithHostAuth(auth)
	session, managed, err := routingSessionForContext(ctx, plugin, instance, policy)
	if err != nil {
		return err
	}
	if !managed {
		defer func() { _ = session.Close(context.Background()) }()
	}
	input := map[string]any{
		"instance": pluginsystem.InstanceRef{ID: instanceID(instance), PluginID: plugin.Manifest.ID},
		"auth":     pluginsystem.PluginInputAuth(plugin, auth),
		"config":   pluginhost.RuntimeConfig(config),
		"request":  request,
	}
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return err
	}
	outputBytes, err := session.Call(ctx, capability.Export, inputBytes, pluginsystem.RuntimeCallOptions{MaxHostRequests: maxHostRequests})
	if err != nil {
		if managed && pluginsystem.IsRuntimeSessionFatalError(err) {
			routingSessionManagerFromContext(ctx).invalidate(ctx, plugin.Manifest.ID, instanceID(instance), session)
		}
		return err
	}
	if len(outputBytes) > maxResponseBytes {
		return &routingError{Code: "response_too_large", Message: "routing plugin response exceeds host limit", HTTPStatus: http.StatusBadGateway}
	}
	if err := json.Unmarshal(outputBytes, output); err != nil {
		return fmt.Errorf("plugin returned invalid %s output: %w", capability.Export, err)
	}
	return nil
}

type routingSessionContextKey struct{}
type routingSessionLaneContextKey struct{}

type routingManagedSession struct {
	ready   chan struct{}
	session pluginsystem.RuntimeSession
	err     error
}

type routingSessionManager struct {
	mu       sync.Mutex
	sessions map[string]*routingManagedSession
	closed   bool
}

func withRoutingPluginSessions(ctx context.Context) (context.Context, func()) {
	manager := &routingSessionManager{sessions: map[string]*routingManagedSession{}}
	return context.WithValue(ctx, routingSessionContextKey{}, manager), manager.close
}

func routingSessionManagerFromContext(ctx context.Context) *routingSessionManager {
	manager, _ := ctx.Value(routingSessionContextKey{}).(*routingSessionManager)
	return manager
}

func withRoutingSessionLane(ctx context.Context, lane int) context.Context {
	return context.WithValue(ctx, routingSessionLaneContextKey{}, lane)
}

func routingSessionForContext(ctx context.Context, plugin pluginsystem.LocalPlugin, instance *core.Record, policy pluginsystem.RequestPolicyContext) (pluginsystem.RuntimeSession, bool, error) {
	manager := routingSessionManagerFromContext(ctx)
	if manager == nil {
		session, err := routingRuntimeSessionOpener(ctx, plugin, policy)
		return session, false, err
	}
	session, err := manager.session(ctx, plugin, instanceID(instance), policy)
	return session, true, err
}

func (m *routingSessionManager) session(ctx context.Context, plugin pluginsystem.LocalPlugin, instanceID string, policy pluginsystem.RequestPolicyContext) (pluginsystem.RuntimeSession, error) {
	key := routingSessionKey(ctx, plugin.Manifest.ID, instanceID)
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil, fmt.Errorf("routing worker sessions are closed")
	}
	if existing := m.sessions[key]; existing != nil {
		m.mu.Unlock()
		select {
		case <-existing.ready:
			return existing.session, existing.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	entry := &routingManagedSession{ready: make(chan struct{})}
	m.sessions[key] = entry
	m.mu.Unlock()

	session, openErr := routingRuntimeSessionOpener(ctx, plugin, policy)
	m.mu.Lock()
	entry.session, entry.err = session, openErr
	if openErr != nil && m.sessions[key] == entry {
		delete(m.sessions, key)
	}
	m.mu.Unlock()
	close(entry.ready)
	return session, openErr
}

func routingSessionKey(ctx context.Context, pluginID string, instanceID string) string {
	key := pluginID + ":" + instanceID
	if lane, ok := ctx.Value(routingSessionLaneContextKey{}).(int); ok {
		key += fmt.Sprintf(":lane-%d", lane)
	}
	return key
}

func (m *routingSessionManager) invalidate(ctx context.Context, pluginID string, instanceID string, session pluginsystem.RuntimeSession) {
	key := routingSessionKey(ctx, pluginID, instanceID)
	m.mu.Lock()
	entry := m.sessions[key]
	matched := entry != nil && entry.session == session
	if matched {
		delete(m.sessions, key)
	}
	m.mu.Unlock()
	if matched {
		_ = session.Close(context.Background())
	}
}

func (m *routingSessionManager) close() {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	m.closed = true
	entries := make([]*routingManagedSession, 0, len(m.sessions))
	for _, entry := range m.sessions {
		entries = append(entries, entry)
	}
	m.sessions = nil
	m.mu.Unlock()
	for _, entry := range entries {
		<-entry.ready
		if entry.session != nil {
			_ = entry.session.Close(context.Background())
		}
	}
}

func instanceID(instance *core.Record) string {
	if instance == nil {
		return "default"
	}
	return instance.Id
}

// routingEnabledPluginInstance deterministically chooses the first configured
// enabled instance when a caller did not select one explicitly. Creation time
// represents setup order and the record ID is the stable tie-breaker for
// legacy rows with the same timestamp.
func routingEnabledPluginInstance(app core.App, userID string, pluginID string, requestedInstanceID string) (*core.Record, error) {
	filter := "user={:user} && plugin_id={:plugin_id} && enabled=true"
	params := dbx.Params{"user": userID, "plugin_id": pluginID}
	if requestedInstanceID != "" && requestedInstanceID != "default" {
		filter += " && id={:instance_id}"
		params["instance_id"] = requestedInstanceID
	}
	records, err := app.FindRecordsByFilter("plugin_instances", filter, "+created,+id", 1, 0, params)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	return records[0], nil
}

type routingError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
}

func (e *routingError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

func routingErrorResponse(err error, detail any) (int, map[string]any) {
	var routingErr *routingError
	if !errors.As(err, &routingErr) {
		routingErr = routingErrorFromCall(err)
	}
	body := map[string]any{
		"data": map[string]any{
			"code":    routingErr.Code,
			"message": routingErr.Message,
		},
		"message": routingErr.Message,
		"status":  routingErr.HTTPStatus,
	}
	if detail != nil {
		body["detail"] = detail
	}
	return routingErr.HTTPStatus, body
}

func routingJSONError(e *core.RequestEvent, err error, detail any) error {
	status, body := routingErrorResponse(err, detail)
	return e.JSON(status, body)
}

func routingErrorFromCall(err error) *routingError {
	if err == nil {
		return nil
	}
	var capacityErr pluginsystem.WorkerCapacityError
	if errors.As(err, &capacityErr) {
		return &routingError{Code: "worker_capacity_exhausted", Message: capacityErr.Error(), HTTPStatus: http.StatusServiceUnavailable}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &routingError{Code: "provider_timeout", Message: "routing provider timed out", HTTPStatus: http.StatusGatewayTimeout}
	}
	var budgetErr pluginsystem.HostRequestBudgetError
	if errors.As(err, &budgetErr) {
		return &routingError{Code: "provider_request_limit_exceeded", Message: budgetErr.Error(), HTTPStatus: http.StatusBadGateway}
	}
	var routingErr *routingError
	if errors.As(err, &routingErr) {
		return routingErr
	}
	return routingErrorFromCode("provider_error", err.Error())
}

func routingErrorFromPluginError(pluginErr pluginsystem.PluginError) *routingError {
	code := pluginErr.Code
	if code == "" {
		code = "provider_error"
	}
	return routingErrorFromCode(code, pluginErr.Message)
}

func routingErrorFromCode(code string, message string) *routingError {
	if code == "" {
		code = "provider_error"
	}
	if message == "" {
		message = code
	}
	status := http.StatusBadGateway
	switch code {
	case "routing_disabled", "navigation_disabled":
		status = http.StatusForbidden
	case "invalid_request", "invalid_coordinate", "anchor_limit_exceeded", "variant_limit_exceeded", "engine_limit_exceeded", "point_limit_exceeded", "fanout_limit_exceeded", "routing_plugin_required":
		status = http.StatusBadRequest
	case "no_route", "unsupported_profile", "mapping_missing", "missing_preference", "invalid_candidate", "invalid_geometry", "routing_mode_unavailable", "profile_preparation_fanout_limit_exceeded", "track_shape_unavailable", "track_limit_exceeded", "discontinuous_track":
		status = http.StatusUnprocessableEntity
	case "maneuver_engine_unavailable", "worker_capacity_exhausted":
		status = http.StatusServiceUnavailable
	case "provider_timeout", "timeout":
		status = http.StatusGatewayTimeout
	case "provider_error", "provider_unavailable", "connector_error", "plugin_error", "response_too_large", "internal_error", "invalid_plugin_response", "provider_request_limit_exceeded":
		status = http.StatusBadGateway
	}
	return &routingError{Code: code, Message: message, HTTPStatus: status}
}

func routingEngineErrorFromPluginError(pluginErr pluginsystem.PluginError, plugin pluginsystem.LocalPlugin, instance *core.Record) pluginRoutingEngineError {
	routingErr := routingErrorFromPluginError(pluginErr)
	return routingEngineErrorFromRoutingError(routingErr, plugin, instance)
}

func routingEngineErrorFromError(err error, plugin pluginsystem.LocalPlugin, instance *core.Record) pluginRoutingEngineError {
	return routingEngineErrorFromRoutingError(routingErrorFromCall(err), plugin, instance)
}

func routingEngineErrorFromRoutingError(err *routingError, plugin pluginsystem.LocalPlugin, instance *core.Record) pluginRoutingEngineError {
	return pluginRoutingEngineError{
		Code:       err.Code,
		Message:    err.Message,
		PluginID:   plugin.Manifest.ID,
		InstanceID: instanceID(instance),
		Provider:   plugin.Manifest.Name,
		HTTPStatus: err.HTTPStatus,
	}
}

func validateRoutingRouteRequest(request pluginRoutingRouteRequest) error {
	if routingMaxEngines < 1 {
		return &routingError{Code: "engine_limit_exceeded", Message: "routing host has no available engines", HTTPStatus: http.StatusUnprocessableEntity}
	}
	if len(request.Anchors) < 2 {
		return &routingError{Code: "invalid_request", Message: "at least two routing anchors are required", HTTPStatus: http.StatusBadRequest}
	}
	if len(request.Anchors) > routingMaxAnchors {
		return &routingError{Code: "anchor_limit_exceeded", Message: "too many routing anchors", HTTPStatus: http.StatusBadRequest}
	}
	if err := validateRoutingInlineProfileContent(request.Profile.ContentBase64); err != nil {
		return err
	}
	if err := validateRoutingPreferences(request.Preferences, false); err != nil {
		return &routingError{Code: "invalid_request", Message: "invalid routing preferences: " + err.Error(), HTTPStatus: http.StatusBadRequest}
	}
	if err := validateRoutingConfig(request.Profile.NativeConfig, "profile.nativeConfig"); err != nil {
		return &routingError{Code: "invalid_request", Message: err.Error(), HTTPStatus: http.StatusBadRequest}
	}
	if err := validateRoutingRequiredPreferences(request.RequiredPreferences); err != nil {
		return &routingError{Code: "invalid_request", Message: err.Error(), HTTPStatus: http.StatusBadRequest}
	}
	for _, anchor := range request.Anchors {
		if !validRoutingCoordinate(anchor) {
			return &routingError{Code: "invalid_coordinate", Message: "routing anchors must be valid WGS84 coordinates", HTTPStatus: http.StatusBadRequest}
		}
	}
	if request.Options.Alternatives < 0 {
		return &routingError{Code: "invalid_request", Message: "routing variants must not be negative", HTTPStatus: http.StatusBadRequest}
	}
	if request.Options.Alternatives > routingMaxProviderCandidates {
		return &routingError{Code: "variant_limit_exceeded", Message: "too many routing variants requested", HTTPStatus: http.StatusBadRequest}
	}
	return nil
}

func validateRoutingRequiredPreferences(required []string) error {
	if len(required) > routingPreferenceMaxEntries {
		return fmt.Errorf("too many required preferences (maximum %d)", routingPreferenceMaxEntries)
	}
	seen := map[string]bool{}
	for _, key := range required {
		key = strings.TrimSpace(key)
		if key == "" || len(key) > routingConfigMaxKeyBytes {
			return fmt.Errorf("required preference key is empty or too long")
		}
		if seen[key] {
			return fmt.Errorf("required preference %q is duplicated", key)
		}
		seen[key] = true
	}
	return nil
}

func validateRoutingInlineProfileContent(contentBase64 string) error {
	if contentBase64 == "" {
		return nil
	}
	if len(contentBase64) > base64.StdEncoding.EncodedLen(routingProfileContentMaxBytes) {
		return routingErrorFromCode("invalid_request", "routing profile content exceeds host limit")
	}
	content, err := base64.StdEncoding.DecodeString(contentBase64)
	if err != nil {
		return routingErrorFromCode("invalid_request", "routing profile content is not valid base64")
	}
	if len(content) == 0 || len(content) > routingProfileContentMaxBytes {
		return routingErrorFromCode("invalid_request", "routing profile content exceeds host limit")
	}
	return nil
}

func validateRoutingElevationRequest(request pluginRoutingElevationRequest) error {
	if request.EncodedPolyline == "" && len(request.Coordinates) == 0 {
		return &routingError{Code: "invalid_request", Message: "encodedPolyline or coordinates are required", HTTPStatus: http.StatusBadRequest}
	}
	if request.EncodedPolyline != "" {
		points, err := decodeRoutingGeometry(pluginRoutingGeometry{
			Format:      routingPolylineFormat,
			Precision:   routingPolylinePrecision,
			Coordinates: request.EncodedPolyline,
		})
		if err != nil {
			return &routingError{Code: err.Code, Message: err.Message, HTTPStatus: http.StatusBadRequest}
		}
		if len(points) > routingMaxElevationPoints {
			return &routingError{Code: "point_limit_exceeded", Message: "too many elevation points", HTTPStatus: http.StatusBadRequest}
		}
	}
	if len(request.Coordinates) > routingMaxElevationPoints {
		return &routingError{Code: "point_limit_exceeded", Message: "too many elevation points", HTTPStatus: http.StatusBadRequest}
	}
	for _, coordinate := range request.Coordinates {
		if !validRoutingCoordinate(coordinate) {
			return &routingError{Code: "invalid_coordinate", Message: "elevation coordinates must be valid WGS84 coordinates", HTTPStatus: http.StatusBadRequest}
		}
	}
	return nil
}

// Routing contracts are defined in openspec/design/routing-plugin.md and
// hardened by openspec/changes/routing-phase-2-host-contracts.
func normalizeRoutingRouteOutput(request pluginRoutingRouteRequest, output pluginRoutingRouteOutput, plugin pluginsystem.LocalPlugin, instance *core.Record) (pluginRoutingRouteOutput, error) {
	normalized := pluginRoutingRouteOutput{
		Candidates:   make([]pluginRoutingCandidate, 0, len(output.Candidates)),
		EngineErrors: append([]pluginRoutingEngineError(nil), output.EngineErrors...),
	}
	for i, candidate := range output.Candidates {
		if err := normalizeRoutingCandidate(request, &candidate, plugin, instance, i); err != nil {
			normalized.EngineErrors = append(normalized.EngineErrors, routingEngineErrorFromRoutingError(err, plugin, instance))
			continue
		}
		normalized.Candidates = append(normalized.Candidates, candidate)
	}
	if len(normalized.Candidates) == 0 {
		return normalized, routingNoCandidateError(normalized.EngineErrors)
	}
	return normalized, nil
}

func normalizeRoutingCandidate(request pluginRoutingRouteRequest, candidate *pluginRoutingCandidate, plugin pluginsystem.LocalPlugin, instance *core.Record, index int) *routingError {
	candidate.ID = fmt.Sprintf("%s:%s:%d", plugin.Manifest.ID, instanceID(instance), index)
	candidate.Provider = plugin.Manifest.Name
	candidate.PluginID = plugin.Manifest.ID
	candidate.InstanceID = instanceID(instance)
	expectedSegments := len(request.Anchors) - 1
	if len(candidate.Segments) != expectedSegments {
		return &routingError{Code: "invalid_candidate", Message: "route candidate must include one segment per adjacent anchor pair", HTTPStatus: http.StatusUnprocessableEntity}
	}
	for i, segment := range candidate.Segments {
		if segment.FromAnchor != i || segment.ToAnchor != i+1 {
			return &routingError{Code: "invalid_candidate", Message: "route segment anchor indexes do not match the request", HTTPStatus: http.StatusUnprocessableEntity}
		}
		if _, err := decodeRoutingGeometry(segment.Geometry); err != nil {
			return err
		}
		if segment.Distance < 0 || segment.Duration < 0 {
			return &routingError{Code: "invalid_candidate", Message: "route segment summary values must not be negative", HTTPStatus: http.StatusUnprocessableEntity}
		}
	}
	if candidate.Geometry != nil {
		points, err := decodeRoutingGeometry(*candidate.Geometry)
		if err != nil {
			return err
		}
		if candidate.Elevation != nil {
			normalizeRoutingCandidateElevation(candidate.Elevation)
			if len(candidate.Elevation.Heights) > 0 && len(candidate.Elevation.Heights) != len(points) {
				return &routingError{Code: "invalid_candidate", Message: "route candidate elevation heights must match geometry point count", HTTPStatus: http.StatusUnprocessableEntity}
			}
		}
	}
	if len(candidate.SnappedAnchors) > 0 {
		if len(candidate.SnappedAnchors) != len(request.Anchors) {
			return &routingError{Code: "invalid_candidate", Message: "snappedAnchors must match request anchor count", HTTPStatus: http.StatusUnprocessableEntity}
		}
		for _, anchor := range candidate.SnappedAnchors {
			if !validRoutingCoordinate(anchor) {
				return &routingError{Code: "invalid_coordinate", Message: "snappedAnchors must be valid WGS84 coordinates", HTTPStatus: http.StatusUnprocessableEntity}
			}
		}
	}
	if candidate.Summary.Distance < 0 || candidate.Summary.Duration < 0 || candidate.Summary.ElevationGain < 0 || candidate.Summary.ElevationLoss < 0 {
		return &routingError{Code: "invalid_candidate", Message: "route summary values must not be negative", HTTPStatus: http.StatusUnprocessableEntity}
	}
	return nil
}

func decodeRoutingGeometry(geometry pluginRoutingGeometry) ([][]float64, *routingError) {
	if geometry.Format != routingPolylineFormat || geometry.Precision != routingPolylinePrecision || geometry.Coordinates == "" {
		return nil, &routingError{Code: "invalid_geometry", Message: "route geometry must be encoded_polyline with precision 6", HTTPStatus: http.StatusUnprocessableEntity}
	}
	points, rest, err := routingPolylineCodec.DecodeCoords([]byte(geometry.Coordinates))
	if err != nil || len(rest) > 0 || len(points) == 0 {
		return nil, &routingError{Code: "invalid_geometry", Message: "route geometry is not a valid encoded polyline", HTTPStatus: http.StatusUnprocessableEntity}
	}
	if len(points) > routingMaxPolylinePoints {
		return nil, &routingError{Code: "point_limit_exceeded", Message: "route geometry contains too many points", HTTPStatus: http.StatusUnprocessableEntity}
	}
	for _, point := range points {
		if len(point) != 2 || !validRoutingCoordinate(pluginRoutingAnchor{Lat: point[0], Lon: point[1]}) {
			return nil, &routingError{Code: "invalid_geometry", Message: "route geometry coordinates must be ordered as [lat, lon]", HTTPStatus: http.StatusUnprocessableEntity}
		}
	}
	return points, nil
}

func routingNoCandidateError(engineErrors []pluginRoutingEngineError) *routingError {
	if len(engineErrors) == 0 {
		return routingErrorFromCode("no_route", "no usable route candidate was returned")
	}
	timeoutCount := 0
	providerCount := 0
	unroutableCount := 0
	for _, engineErr := range engineErrors {
		status := engineErr.HTTPStatus
		if status == 0 {
			status = routingErrorFromCode(engineErr.Code, engineErr.Message).HTTPStatus
		}
		switch status {
		case http.StatusGatewayTimeout:
			timeoutCount++
		case http.StatusBadGateway:
			providerCount++
		case http.StatusUnprocessableEntity:
			unroutableCount++
		}
	}
	if timeoutCount > 0 && timeoutCount >= providerCount {
		return routingErrorFromCode("provider_timeout", "all routing providers timed out")
	}
	if providerCount > 0 || timeoutCount > 0 {
		return routingErrorFromCode("provider_error", "all routing providers failed")
	}
	if unroutableCount > 0 {
		return routingErrorFromCode("no_route", "no usable route candidate remains")
	}
	return routingErrorFromCode("no_route", "no usable route candidate remains")
}

func normalizeRoutingElevationOutput(output *pluginRoutingElevationOutput) {
	if output.Heights == nil {
		output.Heights = []float64{}
	}
	if len(output.Heights) == 0 {
		output.Status = "empty"
		return
	}
	output.Status = "included"
	for _, height := range output.Heights {
		if math.IsNaN(height) || math.IsInf(height, 0) {
			output.Status = "partial"
			return
		}
	}
}

func normalizeRoutingCandidateElevation(elevation *pluginRoutingElevation) {
	if elevation.Heights == nil {
		elevation.Heights = []float64{}
	}
	if elevation.Source == "" {
		elevation.Source = "route"
	}
	if len(elevation.Heights) == 0 {
		elevation.Status = "empty"
		return
	}
	elevation.Status = "included"
	for _, height := range elevation.Heights {
		if math.IsNaN(height) || math.IsInf(height, 0) {
			elevation.Status = "partial"
			return
		}
	}
}

func validRoutingCoordinate(anchor pluginRoutingAnchor) bool {
	return anchor.Lat >= -90 && anchor.Lat <= 90 && anchor.Lon >= -180 && anchor.Lon <= 180
}
