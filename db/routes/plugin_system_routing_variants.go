package routes

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"pocketbase/pluginsystem"
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

const (
	routingShortRouteMeters      = 300.0
	routingCompositionToleranceM = 250.0
	routingMaxFanoutWork         = 64
	routingMaxParallelElevation  = 4
)

var (
	routingVariantRateLimiter      = util.NewRateLimiter(60, time.Minute)
	routingCandidateSetRateLimiter = util.NewRateLimiter(10, time.Minute)
	routingRoutePluginCaller       = callRoutingRoutePlugin
)

type routingEngineRuntime struct {
	Selection                  routingEngineSelection
	Plugin                     pluginsystem.LocalPlugin
	Capability                 pluginsystem.CapabilityManifest
	Instance                   *core.Record
	Auth                       map[string]any
	Config                     map[string]any
	Request                    pluginRoutingRouteRequest
	ClientRequest              pluginRoutingRouteRequest
	PreparedProfileFingerprint string
	PreparedProfileRefresh     *routingPreparedProfileRefresh
}

type routingSegmentCandidate struct {
	Candidate pluginRoutingCandidate
	Score     float64
	ChoiceKey string
}

type routingCandidateComposition struct {
	Candidates []pluginRoutingCandidate
	ChoiceKeys []string
	Score      float64
}

func PluginSystemRoutingRouteCandidates(e *core.RequestEvent) error {
	return pluginSystemRoutingRoute(e, true)
}

func pluginSystemRoutingRoute(e *core.RequestEvent, broadCandidateSet bool) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}
	var data pluginRoutingRouteHTTPInput
	if err := e.BindBody(&data); err != nil {
		return apis.NewBadRequestError("Failed to read request data", err)
	}
	if data.EngineMode != "" && data.EngineMode != "single" && data.EngineMode != "parallel" {
		return routingJSONError(e, routingErrorFromCode("invalid_request", "engineMode must be single or parallel"), nil)
	}
	if data.DesiredVariants < 0 {
		return routingJSONError(e, routingErrorFromCode("invalid_request", "desiredVariants must not be negative"), nil)
	}
	settings, err := ResolveRoutingSettings(e.App, e.Auth.Id)
	if err != nil {
		return err
	}
	if !routingFeatureEnabled(settings, "routing", true) {
		return routingJSONError(e, routingErrorFromCode("routing_disabled", "routing is not enabled"), nil)
	}
	if broadCandidateSet && !e.HasSuperuserAuth() && !routingCandidateSetExposed(e.App) {
		return apis.NewForbiddenError("routing candidate sets are not enabled", nil)
	}

	explicitRoutingMode := strings.TrimSpace(data.RoutingMode) != ""
	routingMode := strings.TrimSpace(data.RoutingMode)
	if routingMode == "" {
		routingMode = settings.DefaultRoutingMode
	}
	if routingMode == "" {
		routingMode = "segment"
	}
	if routingMode != "segment" && routingMode != "via" {
		return routingJSONError(e, routingErrorFromCode("invalid_request", "routingMode must be segment or via"), nil)
	}

	requestVariants := data.RequestVariants || broadCandidateSet
	desiredVariants := 1
	if requestVariants {
		if !routingFeatureEnabled(settings, "variants", true) {
			return apis.NewForbiddenError("routing variants are not enabled", nil)
		}
		desiredVariants = data.DesiredVariants
		if desiredVariants == 0 {
			desiredVariants = settings.DefaultVariantCount
		}
		if desiredVariants < 1 {
			desiredVariants = 1
		}
		if desiredVariants > routingMaxVariants {
			return routingJSONError(e, routingErrorFromCode("variant_limit_exceeded", "too many routing variants requested"), nil)
		}
		if len(data.Anchors) > routingMaxVariantAnchors {
			return routingJSONError(e, routingErrorFromCode("anchor_limit_exceeded", "too many anchors for routing variants"), map[string]any{"maxAnchors": routingMaxVariantAnchors})
		}
		limiter := routingVariantRateLimiter
		limitKey := "variants"
		if broadCandidateSet {
			limiter = routingCandidateSetRateLimiter
			limitKey = "candidate-set"
		}
		if err := limiter.CheckRateLimit(e.Auth.Id, limitKey); err != nil {
			return e.TooManyRequestsError("too many routing variant requests", err)
		}
	}

	parallel := routingUsesParallelEngines(data, requestVariants)
	if parallel && !routingFeatureEnabled(settings, "parallelRouting", true) {
		parallel = false
		data.EngineMode = "single"
		data.Engines = nil
	}
	selections, automaticEngineDiscovery, err := routingSelectionsForRequest(e.App, e.Auth.Id, data, settings, parallel)
	if err != nil {
		return routingJSONError(e, err, nil)
	}

	baseRequest := data.pluginRoutingRouteRequest
	baseRequest.RoutingMode = routingMode
	baseRequest.Options.Alternatives = 1
	// Provider preparation references are host-owned and must never be accepted
	// from an HTTP client.
	baseRequest.Profile.PreparedKey = ""
	if err := validateRoutingRouteRequest(baseRequest); err != nil {
		return routingJSONError(e, err, nil)
	}
	referenceCandidates := []pluginRoutingCandidate{}
	if requestVariants && data.ReferenceGeometry != nil {
		referenceCandidate, referenceErr := routingReferenceCandidate(data.ReferenceGeometry)
		if referenceErr != nil {
			return routingJSONError(e, referenceErr, nil)
		}
		referenceCandidates = append(referenceCandidates, referenceCandidate)
	}
	providerCandidateTarget := routingProviderCandidateTarget(
		desiredVariants,
		len(referenceCandidates) > 0,
	)

	runtimes, engineErrors := routingRuntimesForMode(e, selections, baseRequest, routingMode, providerCandidateTarget, requestVariants, &settings)

	warnings := []string{}
	if routingMode == "via" && len(runtimes) == 0 {
		if explicitRoutingMode {
			return routingJSONError(e, routingErrorFromCode("routing_mode_unavailable", "via routing is not available for the selected engines"), map[string]any{"engineErrors": engineErrors})
		}
		routingMode = "segment"
		baseRequest.RoutingMode = routingMode
		warnings = append(warnings, "routing_mode_fallback")
		var fallbackErrors []pluginRoutingEngineError
		runtimes, fallbackErrors = routingRuntimesForMode(e, selections, baseRequest, routingMode, providerCandidateTarget, requestVariants, &settings)
		engineErrors = append(engineErrors, fallbackErrors...)
	}
	if len(runtimes) == 0 {
		return routingJSONError(e, routingNoCandidateError(engineErrors), map[string]any{"engineErrors": engineErrors})
	}
	routingCtx, closeRoutingSessions := withRoutingPluginSessions(e.Request.Context())
	defer closeRoutingSessions()
	if routingMode == "segment" && requestVariants {
		if automaticEngineDiscovery {
			var reduced bool
			runtimes, reduced = routingRuntimesWithinFanout(runtimes, routingSegmentFanoutWork)
			if reduced {
				warnings = appendRoutingWarningOnce(warnings, "routing_parallel_engines_reduced_for_fanout")
			}
		}
		if work := routingSegmentFanoutWork(runtimes); work > routingMaxFanoutWork {
			return routingJSONError(e, routingErrorFromCode("fanout_limit_exceeded", "routing variant request exceeds the provider-call budget"), map[string]any{"work": work, "maxWork": routingMaxFanoutWork})
		}
	}
	runtimes, preparationErrors := prepareRoutingRuntimeProfiles(routingCtx, runtimes)
	if len(preparationErrors) > 0 {
		warnings = append(warnings, "routing_profile_preparation_failed")
		if routingMode == "segment" && requestVariants {
			if automaticEngineDiscovery {
				var reduced bool
				runtimes, reduced = routingRuntimesWithinFanout(runtimes, routingSegmentExecutionWork)
				if reduced {
					warnings = appendRoutingWarningOnce(warnings, "routing_parallel_engines_reduced_for_fanout")
				}
			}
			if work := routingSegmentExecutionWork(runtimes); work > routingMaxFanoutWork {
				return routingJSONError(e, routingErrorFromCode("profile_preparation_fanout_limit_exceeded", "routing profile preparation failed and the fallback exceeds the provider-call budget"), map[string]any{
					"work":              work,
					"maxWork":           routingMaxFanoutWork,
					"preparationErrors": compactRoutingEngineErrors(preparationErrors),
				})
			}
		}
	}

	var candidates []pluginRoutingCandidate
	var pipelineErrors []pluginRoutingEngineError
	if routingMode == "via" {
		candidates, pipelineErrors = routeViaCandidates(routingCtx, runtimes)
	} else if len(runtimes) == 1 && !requestVariants {
		candidates, pipelineErrors = routeWholeCandidates(routingCtx, runtimes[0], "segment_single_engine")
	} else {
		candidates, pipelineErrors = routeSegmentCandidates(routingCtx, runtimes, len(baseRequest.Anchors), providerCandidateTarget, broadCandidateSet)
	}
	engineErrors = append(engineErrors, pipelineErrors...)
	if len(candidates) == 0 {
		return routingJSONError(e, routingNoCandidateError(engineErrors), map[string]any{
			"engineErrors": engineErrors,
			"request": map[string]any{
				"category": baseRequest.Category, "subcategory": baseRequest.Subcategory,
				"mode": baseRequest.Mode, "routingMode": routingMode,
			},
		})
	}
	// Route calls are complete. Release their per-engine workers before the
	// independent elevation phase so a distinct elevation provider cannot wait
	// behind idle route sessions.
	closeRoutingSessions()

	effectiveDesired := desiredVariants
	if !broadCandidateSet && requestVariants && routingRouteLengthMeters(baseRequest.Anchors) <= routingShortRouteMeters {
		effectiveDesired = 1
		warnings = append(warnings, "routing_variants_reduced_for_short_route")
	}
	limit := effectiveDesired
	if broadCandidateSet {
		limit = routingMaxCandidateSet
	}
	similarToReference := routingCandidatesMatchReferences(candidates, referenceCandidates)

	if (requestVariants || baseRequest.Options.IncludeElevation) && settings.ElevationPluginID != "" {
		var elevationWarnings []string
		elevationCtx, closeElevationSessions := withRoutingPluginSessions(e.Request.Context())
		candidates, elevationWarnings = enrichRoutingCandidateElevation(elevationCtx, e, settings.ElevationPluginID, candidates, limit, referenceCandidates...)
		closeElevationSessions()
		warnings = append(warnings, elevationWarnings...)
	}
	if similarToReference {
		warnings = append(warnings, "routing_variants_similar_to_original_removed")
	}
	candidates = curateRoutingCandidates(candidates, limit, referenceCandidates...)
	if !broadCandidateSet && requestVariants && len(candidates) < effectiveDesired {
		warnings = append(warnings, "routing_variants_fewer_than_requested")
	}
	for index := range candidates {
		candidates[index].ID = routingCandidateID(candidates[index], routingMode)
	}
	return e.JSON(http.StatusOK, pluginRoutingRouteOutput{
		Candidates:   candidates,
		EngineErrors: compactRoutingEngineErrors(engineErrors),
		Warnings:     uniqueRoutingStrings(warnings),
	})
}

func routingCandidateSetExposed(app core.App) bool {
	builtin, builtinErr := routingSettingsForScope(app, routingScopeBuiltin, "")
	admin, adminErr := routingSettingsForScope(app, routingScopeAdmin, "")
	if adminErr == nil {
		if enabled, configured := admin.ExposedFeatures["routeCandidates"]; configured {
			return enabled
		}
	}
	return builtinErr == nil && builtin.ExposedFeatures["routeCandidates"]
}

func routingSelectionsForRequest(app core.App, userID string, data pluginRoutingRouteHTTPInput, settings routingSettings, parallel bool) ([]routingEngineSelection, bool, error) {
	primary := strings.TrimSpace(data.PluginID)
	if primary == "" {
		primary = settings.PrimaryRoutePluginID
	}
	selections := append([]routingEngineSelection(nil), data.Engines...)
	if len(selections) == 0 && primary != "" {
		selections = append(selections, routingEngineSelection{PluginID: primary, InstanceID: strings.TrimSpace(data.InstanceID)})
	}
	automatic := parallel && len(data.Engines) == 0
	// Explicit engine lists remain authoritative for API clients. When callers
	// request parallel variants without a list, the host owns discovery so every
	// client observes the same enabled-engine policy.
	if parallel && len(data.Engines) == 0 {
		engines, err := routingEngines(app, userID)
		if err != nil {
			return nil, false, err
		}
		// Discovery follows the user's stable engine setup order. The primary
		// selection remains first regardless of when it was configured.
		sort.SliceStable(engines, func(i, j int) bool {
			if engines[i].discoveryRank == engines[j].discoveryRank {
				return engines[i].PluginID < engines[j].PluginID
			}
			return engines[i].discoveryRank < engines[j].discoveryRank
		})
		for _, engine := range engines {
			if len(selections) == routingMaxEngines {
				break
			}
			if !engine.Enabled || engine.PluginID == primary || !containsRoutingString(engine.Roles, "route") {
				continue
			}
			selections = append(selections, routingEngineSelection{PluginID: engine.PluginID, InstanceID: engine.InstanceID})
		}
	}
	if !parallel && len(selections) > 1 {
		selections = selections[:1]
	}
	deduped := make([]routingEngineSelection, 0, len(selections))
	seen := map[string]bool{}
	for _, selection := range selections {
		selection.PluginID = strings.TrimSpace(selection.PluginID)
		if selection.PluginID == "" {
			continue
		}
		key := selection.PluginID + ":" + selection.InstanceID
		if seen[key] {
			continue
		}
		seen[key] = true
		deduped = append(deduped, selection)
	}
	if len(deduped) == 0 {
		return nil, false, routingErrorFromCode("invalid_request", "routing plugin is required")
	}
	if len(deduped) > routingMaxEngines {
		return nil, false, routingErrorFromCode("engine_limit_exceeded", "too many routing engines requested")
	}
	return deduped, automatic, nil
}

func routingUsesParallelEngines(data pluginRoutingRouteHTTPInput, requestVariants bool) bool {
	return requestVariants && (data.EngineMode == "parallel" || len(data.Engines) > 1)
}

func routingRuntimesForMode(
	e *core.RequestEvent,
	selections []routingEngineSelection,
	baseRequest pluginRoutingRouteRequest,
	routingMode string,
	desiredVariants int,
	requestVariants bool,
	settings *routingSettings,
) ([]routingEngineRuntime, []pluginRoutingEngineError) {
	runtimes := make([]routingEngineRuntime, 0, len(selections))
	engineErrors := []pluginRoutingEngineError{}
	for index, selection := range selections {
		runtime, loadErr := routingRuntimeForSelection(e, selection, "route")
		if loadErr != nil {
			engineErrors = append(engineErrors, routingEngineErrorForSelection(loadErr, selection))
			continue
		}
		if routingMode == "via" && !routingManifestBool(runtime.Plugin, "supportsViaRouting") {
			engineErrors = append(engineErrors, routingEngineErrorFromRoutingError(
				routingErrorFromCode("routing_mode_unavailable", "selected routing engine does not support via routing"),
				runtime.Plugin,
				runtime.Instance,
			))
			continue
		}
		request := cloneRoutingRouteRequest(baseRequest)
		request.RoutingMode = routingMode
		if index != 0 || (request.Profile.PluginID != "" && request.Profile.PluginID != runtime.Plugin.Manifest.ID) {
			request.Profile = pluginRoutingProfile{PluginID: runtime.Plugin.Manifest.ID}
		} else {
			request.Profile.PluginID = runtime.Plugin.Manifest.ID
		}
		request.Options.Alternatives = routingNativeCandidateCount(runtime.Plugin, desiredVariants, requestVariants)
		runtime.ClientRequest = cloneRoutingRouteRequest(request)
		if mapErr := applyRoutingCategoryMappingContext(e.Request.Context(), e.App, e.Auth.Id, runtime.Plugin.Manifest.ID, instanceID(runtime.Instance), &request, settings); mapErr != nil {
			engineErrors = append(engineErrors, routingEngineErrorFromError(mapErr, runtime.Plugin, runtime.Instance))
			continue
		}
		restrictRoutingPluginPreferences(runtime.Plugin, &request)
		runtime.Request = request
		runtime.PreparedProfileRefresh = &routingPreparedProfileRefresh{}
		runtimes = append(runtimes, runtime)
	}
	return runtimes, engineErrors
}

func routingSegmentFanoutWork(runtimes []routingEngineRuntime) int {
	work := 0
	for _, runtime := range runtimes {
		_, canPrepareProfile := routingProfilePrepareCapability(runtime.Plugin)
		for segmentIndex := 0; segmentIndex+1 < len(runtime.Request.Anchors); segmentIndex++ {
			work += routingSegmentNativeCandidateCount(runtime.Request, segmentIndex)
			if runtime.Request.Profile.ContentBase64 != "" && !canPrepareProfile {
				work++
			}
		}
		if runtime.Request.Profile.ContentBase64 != "" && canPrepareProfile {
			work++
		}
	}
	return work
}

func routingSegmentExecutionWork(runtimes []routingEngineRuntime) int {
	work := 0
	for _, runtime := range runtimes {
		preparedProfile := runtime.Request.Profile.PreparedKey != ""
		for segmentIndex := 0; segmentIndex+1 < len(runtime.Request.Anchors); segmentIndex++ {
			work += routingSegmentNativeCandidateCount(runtime.Request, segmentIndex)
			if runtime.Request.Profile.ContentBase64 != "" && !preparedProfile {
				work++
			}
		}
		if runtime.Request.Profile.ContentBase64 != "" && preparedProfile {
			work++
		}
	}
	return work
}

func routingRuntimesWithinFanout(
	runtimes []routingEngineRuntime,
	workFor func([]routingEngineRuntime) int,
) ([]routingEngineRuntime, bool) {
	if len(runtimes) == 0 {
		return runtimes, false
	}

	// First reserve one candidate per segment for as many engines as fit. The
	// primary (the first usable runtime) is always retained. Additional engines
	// are chosen by cheapest baseline work, with discovery order as the stable
	// tie-breaker, which maximizes comparison breadth under the fixed budget.
	baseline := make([]routingEngineRuntime, len(runtimes))
	desiredAlternatives := make([]int, len(runtimes))
	for index, runtime := range runtimes {
		desiredAlternatives[index] = runtime.Request.Options.Alternatives
		if desiredAlternatives[index] < 1 {
			desiredAlternatives[index] = 1
		}
		runtime.Request.Options.Alternatives = 1
		baseline[index] = runtime
	}
	type baselineCandidate struct {
		index int
		work  int
	}
	additional := make([]baselineCandidate, 0, len(baseline)-1)
	for index := 1; index < len(baseline); index++ {
		additional = append(additional, baselineCandidate{index: index, work: workFor([]routingEngineRuntime{baseline[index]})})
	}
	sort.SliceStable(additional, func(i, j int) bool {
		if additional[i].work == additional[j].work {
			return additional[i].index < additional[j].index
		}
		return additional[i].work < additional[j].work
	})
	selected := []int{0}
	planned := []routingEngineRuntime{baseline[0]}
	for _, candidate := range additional {
		withEngine := append(append([]routingEngineRuntime(nil), planned...), baseline[candidate.index])
		if workFor(withEngine) > routingMaxFanoutWork {
			continue
		}
		selected = append(selected, candidate.index)
		planned = withEngine
	}
	// Restore discovery order after the cheapest-baseline admission decision.
	sort.Ints(selected)
	planned = planned[:0]
	for _, index := range selected {
		planned = append(planned, baseline[index])
	}

	// Spend the remaining budget on native alternatives in rounds. This keeps
	// candidate depth balanced across retained engines while resolving equal
	// opportunities primary-first.
	for {
		allocated := false
		for plannedIndex, originalIndex := range selected {
			if planned[plannedIndex].Request.Options.Alternatives >= desiredAlternatives[originalIndex] {
				continue
			}
			candidate := append([]routingEngineRuntime(nil), planned...)
			candidate[plannedIndex].Request.Options.Alternatives++
			if workFor(candidate) > routingMaxFanoutWork {
				continue
			}
			planned = candidate
			allocated = true
		}
		if !allocated {
			break
		}
	}

	// The warning associated with this result specifically says that engines
	// were reduced. Native-alternative clamping alone therefore remains silent;
	// the final fewer-than-requested warning covers an actual result shortfall.
	return planned, len(planned) != len(runtimes)
}

func routingRuntimeForSelection(e *core.RequestEvent, selection routingEngineSelection, capabilityName string) (routingEngineRuntime, error) {
	plugin, capability, err := localPluginCapability(e.App, selection.PluginID, capabilityName, "v1")
	if err != nil {
		return routingEngineRuntime{}, err
	}
	if plugin.Manifest.Type != pluginsystem.PluginTypeRouting {
		return routingEngineRuntime{}, apis.NewBadRequestError("plugin is not a routing plugin", nil)
	}
	instance, err := routingEnabledPluginInstance(e.App, e.Auth.Id, selection.PluginID, selection.InstanceID)
	if err != nil {
		return routingEngineRuntime{}, err
	}
	if instance == nil {
		return routingEngineRuntime{}, apis.NewBadRequestError("routing plugin is not enabled", map[string]string{"pluginId": selection.PluginID})
	}
	auth, err := decryptedInstanceAuth(instance)
	if err != nil {
		return routingEngineRuntime{}, err
	}
	config := effectiveRoutingPluginConfig(e.App, plugin, instance)
	return routingEngineRuntime{
		Selection: routingEngineSelection{PluginID: selection.PluginID, InstanceID: instance.Id},
		Plugin:    plugin, Capability: capability, Instance: instance, Auth: auth, Config: config,
	}, nil
}

func routingRuntimeForElevation(e *core.RequestEvent, pluginID string) (routingEngineRuntime, error) {
	return routingRuntimeForSelection(e, routingEngineSelection{PluginID: pluginID}, "elevation")
}

func routeViaCandidates(ctx context.Context, runtimes []routingEngineRuntime) ([]pluginRoutingCandidate, []pluginRoutingEngineError) {
	var mu sync.Mutex
	candidates := []pluginRoutingCandidate{}
	engineErrors := []pluginRoutingEngineError{}
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, routingMaxParallelCalls)
	for _, runtime := range runtimes {
		runtime := runtime
		wg.Add(1)
		go func() {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			runtimeCandidates, runtimeErrors := routeWholeCandidates(ctx, runtime, "via_route")
			mu.Lock()
			defer mu.Unlock()
			engineErrors = append(engineErrors, runtimeErrors...)
			candidates = append(candidates, runtimeCandidates...)
		}()
	}
	wg.Wait()
	return candidates, engineErrors
}

func routeWholeCandidates(ctx context.Context, runtime routingEngineRuntime, compositionMode string) ([]pluginRoutingCandidate, []pluginRoutingEngineError) {
	output, callErr := callPreparedRoutingRoutePlugin(ctx, runtime, runtime.Request)
	if callErr != nil {
		return nil, []pluginRoutingEngineError{routingEngineErrorFromError(callErr, runtime.Plugin, runtime.Instance)}
	}
	if output.Error != nil {
		return nil, []pluginRoutingEngineError{routingEngineErrorFromPluginError(*output.Error, runtime.Plugin, runtime.Instance)}
	}
	normalized, normalizeErr := normalizeRoutingRouteOutput(runtime.Request, output, runtime.Plugin, runtime.Instance)
	engineErrors := append([]pluginRoutingEngineError(nil), normalized.EngineErrors...)
	if normalizeErr != nil {
		engineErrors = append(engineErrors, routingEngineErrorFromError(normalizeErr, runtime.Plugin, runtime.Instance))
		return nil, engineErrors
	}
	candidates := make([]pluginRoutingCandidate, 0, len(normalized.Candidates))
	for index := range normalized.Candidates {
		candidate := normalized.Candidates[index]
		candidate.CompositionMode = compositionMode
		addRoutingCandidateProvenance(&candidate, runtime.Request, runtime.ClientRequest, runtime.Plugin, runtime.Instance)
		candidates = append(candidates, candidate)
	}
	return candidates, engineErrors
}

func routeSegmentCandidates(ctx context.Context, runtimes []routingEngineRuntime, anchorCount int, desiredVariants int, broad bool) ([]pluginRoutingCandidate, []pluginRoutingEngineError) {
	segmentCount := anchorCount - 1
	segmentSets := make([][]routingSegmentCandidate, segmentCount)
	laneCounts := routingSegmentSessionLaneCounts(len(runtimes), segmentCount)
	engineErrors := []pluginRoutingEngineError{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for runtimeIndex, runtime := range runtimes {
		laneCount := laneCounts[runtimeIndex]
		for lane := 0; lane < laneCount; lane++ {
			runtime := runtime
			lane := lane
			wg.Add(1)
			go func() {
				defer wg.Done()
				callCtx := ctx
				// Lane zero deliberately uses the unqualified per-engine session. If
				// profile preparation opened it already, its worker is reused instead
				// of consuming an additional global worker slot.
				if lane > 0 {
					callCtx = withRoutingSessionLane(ctx, lane)
				}
				for segmentIndex := lane; segmentIndex < segmentCount; segmentIndex += laneCount {
					request := cloneRoutingRouteRequest(runtime.Request)
					request.Anchors = append([]pluginRoutingAnchor(nil), runtime.Request.Anchors[segmentIndex:segmentIndex+2]...)
					request.Options.Alternatives = routingSegmentNativeCandidateCount(runtime.Request, segmentIndex)
					clientRequest := cloneRoutingRouteRequest(runtime.ClientRequest)
					clientRequest.Anchors = append([]pluginRoutingAnchor(nil), runtime.ClientRequest.Anchors[segmentIndex:segmentIndex+2]...)
					output, callErr := callPreparedRoutingRoutePlugin(callCtx, runtime, request)
					if callErr != nil {
						mu.Lock()
						engineErrors = append(engineErrors, routingEngineErrorFromError(callErr, runtime.Plugin, runtime.Instance))
						mu.Unlock()
						continue
					}
					if output.Error != nil {
						mu.Lock()
						engineErrors = append(engineErrors, routingEngineErrorFromPluginError(*output.Error, runtime.Plugin, runtime.Instance))
						mu.Unlock()
						continue
					}
					normalized, normalizeErr := normalizeRoutingRouteOutput(request, output, runtime.Plugin, runtime.Instance)
					mu.Lock()
					engineErrors = append(engineErrors, normalized.EngineErrors...)
					if normalizeErr == nil {
						for index := range normalized.Candidates {
							candidate := normalized.Candidates[index]
							candidate.CompositionMode = "segment_single_engine"
							addRoutingCandidateProvenance(&candidate, request, clientRequest, runtime.Plugin, runtime.Instance)
							segmentSets[segmentIndex] = append(segmentSets[segmentIndex], routingSegmentCandidate{
								Candidate: candidate,
								Score:     routingCandidateScore(candidate, candidate.Elevation != nil),
								ChoiceKey: fmt.Sprintf("%s:%s:%d", runtime.Plugin.Manifest.ID, instanceID(runtime.Instance), index),
							})
						}
					}
					mu.Unlock()
				}
			}()
		}
	}
	wg.Wait()
	for _, candidates := range segmentSets {
		if len(candidates) == 0 {
			return nil, engineErrors
		}
	}
	perSegmentLimit := desiredVariants
	if broad {
		perSegmentLimit = routingMaxProviderCandidates
	}
	for index := range segmentSets {
		sort.SliceStable(segmentSets[index], func(i, j int) bool { return segmentSets[index][i].Score < segmentSets[index][j].Score })
		if len(segmentSets[index]) > perSegmentLimit {
			segmentSets[index] = segmentSets[index][:perSegmentLimit]
		}
	}

	beams := []routingCandidateComposition{{}}
	beamLimit := routingMaxCandidateSet
	for _, segmentSet := range segmentSets {
		next := make([]routingCandidateComposition, 0, len(beams)*len(segmentSet))
		for _, beam := range beams {
			for _, option := range segmentSet {
				choices := append(append([]pluginRoutingCandidate(nil), beam.Candidates...), option.Candidate)
				choiceKeys := append(append([]string(nil), beam.ChoiceKeys...), option.ChoiceKey)
				next = append(next, routingCandidateComposition{Candidates: choices, ChoiceKeys: choiceKeys, Score: beam.Score + option.Score})
			}
		}
		sort.SliceStable(next, func(i, j int) bool { return next[i].Score < next[j].Score })
		if len(next) > beamLimit {
			next = diverseRoutingBeams(next, beamLimit)
		}
		beams = next
	}
	composed := make([]pluginRoutingCandidate, 0, len(beams))
	for _, beam := range beams {
		candidate, err := composeRoutingCandidate(beam.Candidates)
		if err == nil {
			composed = append(composed, candidate)
		}
	}
	return composed, engineErrors
}

// routingSegmentSessionLaneCounts shares the global parallel-call budget among
// engines in stable request order. Every selected engine receives one lane
// before a second lane is assigned, so the active/primary engine wins only the
// indivisible remainder. Since lane zero is the engine's base session, profile
// preparation and segment execution together retain no more route workers than
// this budget permits.
func routingSegmentSessionLaneCounts(engineCount int, segmentCount int) []int {
	if engineCount <= 0 || segmentCount <= 0 {
		return nil
	}
	counts := make([]int, engineCount)
	budget := routingMaxParallelCalls
	if budget < engineCount {
		// Route validation currently limits engines below the parallel-call
		// budget. Preserve correctness if that invariant changes later.
		budget = engineCount
	}
	assigned := 0
	for assigned < budget {
		allocated := false
		for index := range counts {
			if assigned >= budget {
				break
			}
			if counts[index] >= segmentCount {
				continue
			}
			counts[index]++
			assigned++
			allocated = true
		}
		if !allocated {
			break
		}
	}
	return counts
}

func diverseRoutingBeams(sorted []routingCandidateComposition, limit int) []routingCandidateComposition {
	if limit <= 0 || len(sorted) == 0 {
		return nil
	}
	if len(sorted) <= limit {
		return sorted
	}
	selected := make([]routingCandidateComposition, 0, limit)
	selected = append(selected, sorted[0])
	used := map[int]bool{0: true}
	for len(selected) < limit {
		bestIndex := -1
		bestDistance := -1
		for index := 1; index < len(sorted); index++ {
			if used[index] {
				continue
			}
			minimumDistance := len(sorted[index].ChoiceKeys)
			for _, existing := range selected {
				distance := routingBeamChoiceDistance(sorted[index].ChoiceKeys, existing.ChoiceKeys)
				if distance < minimumDistance {
					minimumDistance = distance
				}
			}
			if minimumDistance > bestDistance {
				bestIndex = index
				bestDistance = minimumDistance
			}
		}
		if bestIndex < 0 {
			break
		}
		used[bestIndex] = true
		selected = append(selected, sorted[bestIndex])
	}
	sort.SliceStable(selected, func(i, j int) bool { return selected[i].Score < selected[j].Score })
	return selected
}

func routingBeamChoiceDistance(left []string, right []string) int {
	length := len(left)
	if len(right) < length {
		length = len(right)
	}
	distance := 0
	for index := 0; index < length; index++ {
		if left[index] != right[index] {
			distance++
		}
	}
	if len(left) > length {
		distance += len(left) - length
	}
	if len(right) > length {
		distance += len(right) - length
	}
	return distance
}

func routingSegmentNativeCandidateCount(request pluginRoutingRouteRequest, segmentIndex int) int {
	if segmentIndex < 0 || segmentIndex+1 >= len(request.Anchors) {
		return 1
	}
	start, end := request.Anchors[segmentIndex], request.Anchors[segmentIndex+1]
	if routingCoordinateDistanceMeters(start.Lat, start.Lon, end.Lat, end.Lon) <= routingShortRouteMeters {
		return 1
	}
	if request.Options.Alternatives < 1 {
		return 1
	}
	return request.Options.Alternatives
}

func composeRoutingCandidate(parts []pluginRoutingCandidate) (pluginRoutingCandidate, error) {
	if len(parts) == 0 {
		return pluginRoutingCandidate{}, fmt.Errorf("cannot compose an empty route")
	}
	result := pluginRoutingCandidate{
		Segments:       make([]pluginRoutingSegment, 0, len(parts)),
		SnappedAnchors: make([]pluginRoutingAnchor, 0, len(parts)+1),
		Warnings:       []string{},
	}
	allPoints := [][]float64{}
	allHeights := []float64{}
	completeElevation := true
	pluginID := parts[0].PluginID
	instanceID := parts[0].InstanceID
	provider := parts[0].Provider
	profileKey := parts[0].ProfileKey
	for index, part := range parts {
		if len(part.Segments) != 1 {
			return pluginRoutingCandidate{}, fmt.Errorf("segment composition requires one segment per part")
		}
		segment := part.Segments[0]
		points, geometryErr := decodeRoutingGeometry(segment.Geometry)
		if geometryErr != nil {
			return pluginRoutingCandidate{}, geometryErr
		}
		if len(allPoints) > 0 {
			previous := allPoints[len(allPoints)-1]
			if routingCoordinateDistanceMeters(previous[0], previous[1], points[0][0], points[0][1]) > routingCompositionToleranceM {
				return pluginRoutingCandidate{}, fmt.Errorf("route segment endpoints are not continuous")
			}
			points = points[1:]
		}
		allPoints = append(allPoints, points...)
		segment.FromAnchor = index
		segment.ToAnchor = index + 1
		result.Segments = append(result.Segments, segment)
		result.Summary.Distance += segment.Distance
		result.Summary.Duration += segment.Duration
		result.Summary.ElevationGain += part.Summary.ElevationGain
		result.Summary.ElevationLoss += part.Summary.ElevationLoss
		result.Warnings = append(result.Warnings, part.Warnings...)
		if len(part.SnappedAnchors) == 2 {
			if index == 0 {
				result.SnappedAnchors = append(result.SnappedAnchors, part.SnappedAnchors[0])
			}
			result.SnappedAnchors = append(result.SnappedAnchors, part.SnappedAnchors[1])
		}
		if part.Elevation == nil || len(part.Elevation.Heights) == 0 {
			completeElevation = false
		} else if completeElevation {
			heights := part.Elevation.Heights
			if index > 0 && len(heights) > 0 {
				heights = heights[1:]
			}
			allHeights = append(allHeights, heights...)
		}
		if part.PluginID != pluginID || part.InstanceID != instanceID {
			pluginID, instanceID, provider, profileKey = "", "", "", ""
		}
	}
	result.Geometry = &pluginRoutingGeometry{Format: routingPolylineFormat, Precision: routingPolylinePrecision, Coordinates: string(routingPolylineCodec.EncodeCoords(nil, allPoints))}
	if completeElevation && len(allHeights) == len(allPoints) {
		result.Elevation = &pluginRoutingElevation{Heights: allHeights, Status: "included", Source: "route"}
	}
	result.PluginID = pluginID
	result.InstanceID = instanceID
	result.Provider = provider
	result.ProfileKey = profileKey
	if pluginID == "" {
		result.CompositionMode = "segment_composed"
	} else {
		result.CompositionMode = "segment_single_engine"
	}
	result.Warnings = uniqueRoutingStrings(result.Warnings)
	return result, nil
}

func addRoutingCandidateProvenance(
	candidate *pluginRoutingCandidate,
	request pluginRoutingRouteRequest,
	clientRequest pluginRoutingRouteRequest,
	plugin pluginsystem.LocalPlugin,
	instance *core.Record,
) {
	for index := range candidate.Segments {
		candidate.Segments[index].Provenance = &pluginRoutingSegmentProvenance{
			Category: request.Category, Subcategory: request.Subcategory, RoutingMode: request.RoutingMode,
			Preferences: cloneRoutingMap(request.Preferences), PluginID: plugin.Manifest.ID,
			RequestedPreferences: cloneRoutingMap(clientRequest.Preferences),
			InstanceID:           instanceID(instance), Provider: plugin.Manifest.Name, ProfileID: request.Profile.ID,
			ProfileKey: request.Profile.Key, ProfileKind: request.Profile.Kind,
			NativeConfig:          cloneRoutingMap(request.Profile.NativeConfig),
			RequestedNativeConfig: cloneRoutingMap(clientRequest.Profile.NativeConfig),
			ProfileRevision:       routingResolvedProfileRevision(request.Profile),
		}
	}
}

func enrichRoutingCandidateElevation(ctx context.Context, e *core.RequestEvent, pluginID string, candidates []pluginRoutingCandidate, finalLimit int, references ...pluginRoutingCandidate) ([]pluginRoutingCandidate, []string) {
	runtime, err := routingRuntimeForElevation(e, pluginID)
	if err != nil {
		return candidates, []string{"routing_elevation_unavailable"}
	}
	candidates = routingElevationCandidateShortlist(candidates, finalLimit, references...)
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, routingMaxParallelElevation)
	for index := range candidates {
		candidate := &candidates[index]
		if candidate.Geometry == nil || (candidate.Elevation != nil && len(candidate.Elevation.Heights) > 0) {
			continue
		}
		index := index
		wg.Add(1)
		go func() {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			candidate := &candidates[index]
			callCtx := withRoutingSessionLane(ctx, index%routingMaxParallelElevation)
			output, callErr := callRoutingElevationPlugin(callCtx, runtime.Plugin, runtime.Capability, runtime.Instance, runtime.Auth, runtime.Config, pluginRoutingElevationRequest{EncodedPolyline: candidate.Geometry.Coordinates})
			if callErr != nil || output.Error != nil {
				return
			}
			normalizeRoutingElevationOutput(&output)
			points, geometryErr := decodeRoutingGeometry(*candidate.Geometry)
			if geometryErr != nil || len(output.Heights) != len(points) {
				return
			}
			candidate.Elevation = &pluginRoutingElevation{Heights: output.Heights, Status: "included", Source: runtime.Plugin.Manifest.ID}
			candidate.Summary.ElevationGain, candidate.Summary.ElevationLoss = routingElevationTotals(output.Heights)
		}()
	}
	wg.Wait()
	return candidates, nil
}

func routingElevationCandidateShortlist(candidates []pluginRoutingCandidate, finalLimit int, references ...pluginRoutingCandidate) []pluginRoutingCandidate {
	shortlistLimit := finalLimit * 2
	if shortlistLimit < 1 {
		shortlistLimit = 1
	}
	if shortlistLimit > len(candidates) {
		shortlistLimit = len(candidates)
	}
	if shortlistLimit > routingMaxCandidateSet {
		shortlistLimit = routingMaxCandidateSet
	}
	return curateRoutingCandidates(candidates, shortlistLimit, references...)
}

func curateRoutingCandidates(candidates []pluginRoutingCandidate, limit int, references ...pluginRoutingCandidate) []pluginRoutingCandidate {
	if limit < 1 || len(candidates) == 0 {
		return nil
	}
	includeElevation := routingCandidatesHaveComparableElevation(candidates)
	sort.SliceStable(candidates, func(i, j int) bool {
		return routingCandidateScore(candidates[i], includeElevation) < routingCandidateScore(candidates[j], includeElevation)
	})
	selected := make([]pluginRoutingCandidate, 0, limit)
	for _, candidate := range candidates {
		duplicate := routingCandidatesMatchReferences([]pluginRoutingCandidate{candidate}, references)
		for _, existing := range selected {
			if routingCandidatesTooSimilar(existing, candidate) {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}
		selected = append(selected, candidate)
		if len(selected) == limit {
			break
		}
	}
	return selected
}

func routingReferenceCandidate(geometry *pluginRoutingGeometry) (pluginRoutingCandidate, *routingError) {
	if geometry == nil {
		return pluginRoutingCandidate{}, routingErrorFromCode("invalid_request", "reference route geometry is required")
	}
	points, err := decodeRoutingGeometry(*geometry)
	if err != nil {
		return pluginRoutingCandidate{}, routingErrorFromCode("invalid_request", "reference route geometry is invalid")
	}
	distance := 0.0
	for index := 1; index < len(points); index++ {
		distance += routingCoordinateDistanceMeters(
			points[index-1][0],
			points[index-1][1],
			points[index][0],
			points[index][1],
		)
	}
	return pluginRoutingCandidate{
		Geometry: geometry,
		Summary:  pluginRoutingSummary{Distance: distance},
	}, nil
}

func routingCandidatesMatchReferences(candidates, references []pluginRoutingCandidate) bool {
	for _, candidate := range candidates {
		for _, reference := range references {
			if routingCandidatesTooSimilar(reference, candidate) {
				return true
			}
		}
	}
	return false
}

func routingCandidatesTooSimilar(left, right pluginRoutingCandidate) bool {
	if left.Geometry == nil || right.Geometry == nil {
		return false
	}
	if left.Geometry.Coordinates == right.Geometry.Coordinates {
		return true
	}
	leftPoints, leftErr := decodeRoutingGeometry(*left.Geometry)
	rightPoints, rightErr := decodeRoutingGeometry(*right.Geometry)
	if leftErr != nil || rightErr != nil {
		return false
	}
	if len(leftPoints) < 2 || len(rightPoints) < 2 {
		return true
	}
	const samples = 200
	leftSamples := routingSampledPoints(leftPoints, samples)
	rightSamples := routingSampledPoints(rightPoints, samples)
	corridorMeters := math.Max(30, math.Min(math.Max(left.Summary.Distance, right.Summary.Distance)*0.001, 200))
	const minimumOverlap = 0.8
	return routingSampleOverlap(leftSamples, rightSamples, corridorMeters) >= minimumOverlap &&
		routingSampleOverlap(rightSamples, leftSamples, corridorMeters) >= minimumOverlap
}

func routingSampledPoints(points [][]float64, count int) [][]float64 {
	if len(points) == 0 || count < 1 {
		return nil
	}
	if len(points) == 1 || count == 1 {
		return [][]float64{append([]float64(nil), points[0]...)}
	}
	segmentLengths := make([]float64, len(points)-1)
	totalLength := 0.0
	for index := range segmentLengths {
		segmentLengths[index] = routingCoordinateDistanceMeters(points[index][0], points[index][1], points[index+1][0], points[index+1][1])
		totalLength += segmentLengths[index]
	}
	if totalLength == 0 {
		return [][]float64{append([]float64(nil), points[0]...)}
	}
	sampled := make([][]float64, 0, count)
	segmentIndex := 0
	covered := 0.0
	for sampleIndex := 0; sampleIndex < count; sampleIndex++ {
		target := float64(sampleIndex) / float64(count-1) * totalLength
		for segmentIndex+1 < len(segmentLengths) && covered+segmentLengths[segmentIndex] < target {
			covered += segmentLengths[segmentIndex]
			segmentIndex++
		}
		ratio := 0.0
		if segmentLengths[segmentIndex] > 0 {
			ratio = (target - covered) / segmentLengths[segmentIndex]
		}
		sampled = append(sampled, []float64{
			points[segmentIndex][0] + (points[segmentIndex+1][0]-points[segmentIndex][0])*ratio,
			points[segmentIndex][1] + (points[segmentIndex+1][1]-points[segmentIndex][1])*ratio,
		})
	}
	return sampled
}

func routingSampleOverlap(source, target [][]float64, corridorMeters float64) float64 {
	if len(source) == 0 || len(target) < 2 {
		return 0
	}
	withinCorridor := 0
	for _, point := range source {
		for index := 1; index < len(target); index++ {
			if routingPointToSegmentDistanceMeters(point, target[index-1], target[index]) <= corridorMeters {
				withinCorridor++
				break
			}
		}
	}
	return float64(withinCorridor) / float64(len(source))
}

func routingPointToSegmentDistanceMeters(point, start, end []float64) float64 {
	const earthRadius = 6371000.0
	latScale := math.Pi / 180 * earthRadius
	lonScale := latScale * math.Cos(point[0]*math.Pi/180)
	startX := (start[1] - point[1]) * lonScale
	startY := (start[0] - point[0]) * latScale
	endX := (end[1] - point[1]) * lonScale
	endY := (end[0] - point[0]) * latScale
	deltaX := endX - startX
	deltaY := endY - startY
	lengthSquared := deltaX*deltaX + deltaY*deltaY
	if lengthSquared == 0 {
		return math.Hypot(startX, startY)
	}
	ratio := -(startX*deltaX + startY*deltaY) / lengthSquared
	ratio = math.Max(0, math.Min(1, ratio))
	return math.Hypot(startX+ratio*deltaX, startY+ratio*deltaY)
}

func routingCandidateScore(candidate pluginRoutingCandidate, includeElevation bool) float64 {
	duration := candidate.Summary.Duration
	if duration == 0 {
		duration = candidate.Summary.Distance / 1.4
	}
	elevationScore := 0.0
	if includeElevation {
		elevationScore = candidate.Summary.ElevationGain * 2
	}
	return duration + candidate.Summary.Distance/20 + elevationScore + float64(len(candidate.Warnings))*300
}

func routingCandidatesHaveComparableElevation(candidates []pluginRoutingCandidate) bool {
	if len(candidates) == 0 {
		return false
	}
	for _, candidate := range candidates {
		if candidate.Elevation == nil || len(candidate.Elevation.Heights) == 0 {
			return false
		}
	}
	return true
}

func routingElevationTotals(heights []float64) (float64, float64) {
	gain, loss := 0.0, 0.0
	for index := 1; index < len(heights); index++ {
		if math.IsNaN(heights[index]) || math.IsNaN(heights[index-1]) || math.IsInf(heights[index], 0) || math.IsInf(heights[index-1], 0) {
			continue
		}
		delta := heights[index] - heights[index-1]
		if delta > 0 {
			gain += delta
		} else {
			loss -= delta
		}
	}
	return gain, loss
}

func routingRouteLengthMeters(anchors []pluginRoutingAnchor) float64 {
	total := 0.0
	for index := 1; index < len(anchors); index++ {
		total += routingCoordinateDistanceMeters(anchors[index-1].Lat, anchors[index-1].Lon, anchors[index].Lat, anchors[index].Lon)
	}
	return total
}

func routingCoordinateDistanceMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371000.0
	lat1Rad, lat2Rad := lat1*math.Pi/180, lat2*math.Pi/180
	dLat, dLon := (lat2-lat1)*math.Pi/180, (lon2-lon1)*math.Pi/180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return earthRadius * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func routingNativeCandidateCount(plugin pluginsystem.LocalPlugin, desired int, requested bool) int {
	if !requested || !routingManifestBool(plugin, "supportsAlternatives") {
		return 1
	}
	maximum := intValue(mapValue(plugin.Manifest.Metadata["routing"])["maxAlternatives"])
	if maximum < 1 {
		maximum = 1
	}
	if desired > maximum {
		return maximum
	}
	return desired
}

func routingProviderCandidateTarget(desired int, hasReference bool) int {
	if hasReference && desired < routingMaxProviderCandidates {
		return desired + 1
	}
	return desired
}

func routingManifestBool(plugin pluginsystem.LocalPlugin, key string) bool {
	value, _ := mapValue(plugin.Manifest.Metadata["routing"])[key].(bool)
	return value
}

func routingFeatureEnabled(settings routingSettings, key string, fallback bool) bool {
	value, ok := settings.ExposedFeatures[key]
	if !ok {
		return fallback
	}
	return value
}

func cloneRoutingRouteRequest(request pluginRoutingRouteRequest) pluginRoutingRouteRequest {
	clone := request
	clone.Anchors = append([]pluginRoutingAnchor(nil), request.Anchors...)
	clone.Preferences = cloneRoutingMap(request.Preferences)
	clone.RequiredPreferences = append([]string(nil), request.RequiredPreferences...)
	clone.Profile.Metadata = cloneRoutingMap(request.Profile.Metadata)
	clone.Profile.NativeConfig = cloneRoutingMap(request.Profile.NativeConfig)
	return clone
}

func routingEngineErrorForSelection(err error, selection routingEngineSelection) pluginRoutingEngineError {
	routingErr := routingErrorFromCall(err)
	return pluginRoutingEngineError{Code: routingErr.Code, Message: routingErr.Message, PluginID: selection.PluginID, InstanceID: selection.InstanceID, HTTPStatus: routingErr.HTTPStatus}
}

func routingCandidateID(candidate pluginRoutingCandidate, mode string) string {
	provenanceParts := make([]string, 0, len(candidate.Segments))
	for _, segment := range candidate.Segments {
		if segment.Provenance == nil {
			continue
		}
		if encoded, err := json.Marshal(segment.Provenance); err == nil {
			provenanceParts = append(provenanceParts, string(encoded))
		}
	}
	digest := sha256.Sum256([]byte(strings.Join(provenanceParts, "|") + "|" + routingCandidateGeometry(candidate)))
	return fmt.Sprintf("routing:%s:%s:%x", mode, candidate.CompositionMode, digest[:6])
}

func routingCandidateGeometry(candidate pluginRoutingCandidate) string {
	if candidate.Geometry != nil {
		return candidate.Geometry.Coordinates
	}
	parts := make([]string, 0, len(candidate.Segments))
	for _, segment := range candidate.Segments {
		parts = append(parts, segment.Geometry.Coordinates)
	}
	return strings.Join(parts, "|")
}

func compactRoutingEngineErrors(values []pluginRoutingEngineError) []pluginRoutingEngineError {
	result := make([]pluginRoutingEngineError, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		key := value.PluginID + ":" + value.InstanceID + ":" + value.Code + ":" + value.Message
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, value)
	}
	return result
}

func uniqueRoutingStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func appendRoutingWarningOnce(warnings []string, warning string) []string {
	for _, existing := range warnings {
		if existing == warning {
			return warnings
		}
	}
	return append(warnings, warning)
}
