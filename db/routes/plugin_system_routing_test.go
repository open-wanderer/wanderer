package routes

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"pocketbase/pluginsystem"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/twpayne/go-polyline"
)

var testRoutingPlugin = pluginsystem.LocalPlugin{Manifest: pluginsystem.Manifest{
	ID:   "valhalla",
	Type: pluginsystem.PluginTypeRouting,
	Name: "Valhalla",
}}

type testRoutingRuntimeSession struct {
	closed int
}

func (s *testRoutingRuntimeSession) Call(context.Context, string, []byte, pluginsystem.RuntimeCallOptions) ([]byte, error) {
	return []byte(`{}`), nil
}

func (s *testRoutingRuntimeSession) Close(context.Context) error {
	s.closed++
	return nil
}

func TestRoutingSessionManagerReusesPerInstanceAndSeparatesLanes(t *testing.T) {
	previousOpener := routingRuntimeSessionOpener
	opened := []*testRoutingRuntimeSession{}
	routingRuntimeSessionOpener = func(context.Context, pluginsystem.LocalPlugin, pluginsystem.RequestPolicyContext) (pluginsystem.RuntimeSession, error) {
		session := &testRoutingRuntimeSession{}
		opened = append(opened, session)
		return session, nil
	}
	t.Cleanup(func() { routingRuntimeSessionOpener = previousOpener })

	ctx, closeSessions := withRoutingPluginSessions(context.Background())
	first, managed, err := routingSessionForContext(ctx, testRoutingPlugin, nil, pluginsystem.RequestPolicyContext{})
	if err != nil || !managed {
		t.Fatalf("open managed routing session: managed=%v err=%v", managed, err)
	}
	second, _, err := routingSessionForContext(ctx, testRoutingPlugin, nil, pluginsystem.RequestPolicyContext{})
	if err != nil || first != second || len(opened) != 1 {
		t.Fatalf("same instance did not reuse its worker: opened=%d err=%v", len(opened), err)
	}
	lane, _, err := routingSessionForContext(withRoutingSessionLane(ctx, 1), testRoutingPlugin, nil, pluginsystem.RequestPolicyContext{})
	if err != nil || lane == first || len(opened) != 2 {
		t.Fatalf("parallel lane did not receive an independent worker: opened=%d err=%v", len(opened), err)
	}
	closeSessions()
	for _, session := range opened {
		if session.closed != 1 {
			t.Fatalf("worker closed %d times, want once", session.closed)
		}
	}
}

func TestRoutingSessionManagerInvalidationConcurrentWithReplacementOpen(t *testing.T) {
	previousOpener := routingRuntimeSessionOpener
	first := &testRoutingRuntimeSession{}
	replacement := &testRoutingRuntimeSession{}
	replacementStarted := make(chan struct{})
	releaseReplacement := make(chan struct{})
	var openCount atomic.Int32
	routingRuntimeSessionOpener = func(context.Context, pluginsystem.LocalPlugin, pluginsystem.RequestPolicyContext) (pluginsystem.RuntimeSession, error) {
		switch openCount.Add(1) {
		case 1:
			return first, nil
		case 2:
			close(replacementStarted)
			<-releaseReplacement
			return replacement, nil
		default:
			return nil, fmt.Errorf("unexpected routing session open")
		}
	}
	t.Cleanup(func() { routingRuntimeSessionOpener = previousOpener })

	manager := &routingSessionManager{sessions: map[string]*routingManagedSession{}}
	defer manager.close()
	ctx := context.Background()
	session, err := manager.session(ctx, testRoutingPlugin, "default", pluginsystem.RequestPolicyContext{})
	if err != nil || session != first {
		t.Fatalf("open first routing session: session=%T err=%v", session, err)
	}
	manager.invalidate(ctx, testRoutingPlugin.Manifest.ID, "default", first)

	type sessionResult struct {
		session pluginsystem.RuntimeSession
		err     error
	}
	result := make(chan sessionResult, 1)
	go func() {
		session, err := manager.session(ctx, testRoutingPlugin, "default", pluginsystem.RequestPolicyContext{})
		result <- sessionResult{session: session, err: err}
	}()
	<-replacementStarted

	const invalidatorCount = 16
	startInvalidation := make(chan struct{})
	stopInvalidation := make(chan struct{})
	var invalidatorsReady sync.WaitGroup
	var invalidatorsDone sync.WaitGroup
	invalidatorsReady.Add(invalidatorCount)
	invalidatorsDone.Add(invalidatorCount)
	for range invalidatorCount {
		go func() {
			defer invalidatorsDone.Done()
			<-startInvalidation
			invalidatorsReady.Done()
			for {
				select {
				case <-stopInvalidation:
					return
				default:
					manager.invalidate(ctx, testRoutingPlugin.Manifest.ID, "default", first)
				}
			}
		}()
	}
	close(startInvalidation)
	invalidatorsReady.Wait()
	close(releaseReplacement)
	opened := <-result
	close(stopInvalidation)
	invalidatorsDone.Wait()

	if opened.err != nil || opened.session != replacement {
		t.Fatalf("open replacement routing session: session=%T err=%v", opened.session, opened.err)
	}
	reused, err := manager.session(ctx, testRoutingPlugin, "default", pluginsystem.RequestPolicyContext{})
	if err != nil || reused != replacement {
		t.Fatalf("old invalidation removed the replacement: session=%T err=%v", reused, err)
	}
	if first.closed != 1 || replacement.closed != 0 {
		t.Fatalf("unexpected close counts before manager shutdown: first=%d replacement=%d", first.closed, replacement.closed)
	}

	manager.close()
	if replacement.closed != 1 {
		t.Fatalf("replacement session closed %d times, want once", replacement.closed)
	}
}

func TestRoutingWorkerCapacityIsNotReportedAsProviderTimeout(t *testing.T) {
	err := routingErrorFromCall(pluginsystem.WorkerCapacityError{Err: context.DeadlineExceeded})
	if err.Code != "worker_capacity_exhausted" || err.HTTPStatus != http.StatusServiceUnavailable {
		t.Fatalf("worker capacity error was misclassified: %#v", err)
	}
}

func TestNormalizeRoutingRouteOutputPreservesPartialSuccessErrorsAndSnappedAnchors(t *testing.T) {
	request := testRoutingRequest()
	output := pluginRoutingRouteOutput{
		Candidates: []pluginRoutingCandidate{testRoutingCandidate(request)},
		EngineErrors: []pluginRoutingEngineError{{
			Code:     "provider_error",
			Message:  "secondary engine failed",
			PluginID: "brouter",
		}},
	}

	normalized, err := normalizeRoutingRouteOutput(request, output, testRoutingPlugin, nil)
	if err != nil {
		t.Fatalf("expected partial success, got error: %v", err)
	}
	if len(normalized.Candidates) != 1 {
		t.Fatalf("expected one candidate, got %d", len(normalized.Candidates))
	}
	candidate := normalized.Candidates[0]
	if candidate.ID != "valhalla:default:0" {
		t.Fatalf("expected host-owned candidate id, got %q", candidate.ID)
	}
	if candidate.PluginID != "valhalla" || candidate.InstanceID != "default" || candidate.Provider != "Valhalla" {
		t.Fatalf("missing provenance: %#v", candidate)
	}
	if len(candidate.SnappedAnchors) != len(request.Anchors) {
		t.Fatalf("expected snapped anchors to be preserved: %#v", candidate.SnappedAnchors)
	}
	if len(normalized.EngineErrors) != 1 || normalized.EngineErrors[0].PluginID != "brouter" {
		t.Fatalf("expected engine error to be preserved: %#v", normalized.EngineErrors)
	}
}

func TestRoutingHTTPStatusMapping(t *testing.T) {
	cases := []struct {
		name   string
		err    *routingError
		status int
	}{
		{"invalid request", &routingError{Code: "invalid_request", Message: "bad", HTTPStatus: http.StatusBadRequest}, http.StatusBadRequest},
		{"preparation fallback exceeds work budget", routingErrorFromCode("profile_preparation_fanout_limit_exceeded", "fallback too large"), http.StatusUnprocessableEntity},
		{"unroutable", routingNoCandidateError(nil), http.StatusUnprocessableEntity},
		{"provider failure", routingNoCandidateError([]pluginRoutingEngineError{{Code: "provider_error"}}), http.StatusBadGateway},
		{"provider unavailable", routingNoCandidateError([]pluginRoutingEngineError{{Code: "provider_unavailable"}}), http.StatusBadGateway},
		{"provider timeout", routingNoCandidateError([]pluginRoutingEngineError{{Code: "provider_timeout"}}), http.StatusGatewayTimeout},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err.HTTPStatus != tc.status {
				t.Fatalf("expected %d, got %d (%#v)", tc.status, tc.err.HTTPStatus, tc.err)
			}
		})
	}
}

func TestRoutingStructuredErrorCodes(t *testing.T) {
	pluginErr := routingErrorFromPluginError(pluginsystem.PluginError{
		Code:    "no_route",
		Message: "no path",
	})
	if pluginErr.Code != "no_route" || pluginErr.HTTPStatus != http.StatusUnprocessableEntity {
		t.Fatalf("unexpected plugin error mapping: %#v", pluginErr)
	}

	timeoutErr := routingErrorFromCall(context.DeadlineExceeded)
	if timeoutErr.Code != "provider_timeout" || timeoutErr.HTTPStatus != http.StatusGatewayTimeout {
		t.Fatalf("unexpected timeout mapping: %#v", timeoutErr)
	}

	unavailable := routingErrorFromPluginError(pluginsystem.PluginError{
		Code:    "provider_unavailable",
		Message: "Valhalla is down",
	})
	if unavailable.Code != "provider_unavailable" || unavailable.HTTPStatus != http.StatusBadGateway {
		t.Fatalf("unexpected provider_unavailable mapping: %#v", unavailable)
	}

	tooLarge := routingNoCandidateError([]pluginRoutingEngineError{{Code: "response_too_large"}})
	if tooLarge.Code != "provider_error" || tooLarge.HTTPStatus != http.StatusBadGateway {
		t.Fatalf("unexpected response size mapping: %#v", tooLarge)
	}
}

func TestRoutingPluginCheckAcceptsCoverageErrorsOnly(t *testing.T) {
	cases := []struct {
		code   string
		accept bool
	}{
		{"no_route", true},
		{"unsupported_profile", true},
		{"invalid_candidate", true},
		{"provider_unavailable", false},
		{"provider_error", false},
		{"connector_error", false},
		{"provider_timeout", false},
	}
	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			err := routingErrorFromCode(tc.code, tc.code)
			if got := routingPluginCheckAcceptsError(err); got != tc.accept {
				t.Fatalf("expected accept=%v, got %v for %#v", tc.accept, got, err)
			}
		})
	}
}

func TestRoutingTotalFailureKeepsEngineErrorsInDetail(t *testing.T) {
	engineErrors := []pluginRoutingEngineError{{
		Code:       "provider_unavailable",
		Message:    "Valhalla is down",
		PluginID:   "valhalla",
		HTTPStatus: http.StatusBadGateway,
	}}

	status, body := routingErrorResponse(routingNoCandidateError(engineErrors), map[string]any{
		"engineErrors": engineErrors,
	})
	if status != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, status)
	}
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data map, got %#v", body["data"])
	}
	if data["code"] != "provider_error" {
		t.Fatalf("expected provider_error response code, got %#v", data["code"])
	}
	detail, ok := body["detail"].(map[string]any)
	if !ok {
		t.Fatalf("expected detail map, got %#v", body["detail"])
	}
	if got, ok := detail["engineErrors"].([]pluginRoutingEngineError); !ok || len(got) != 1 || got[0].Code != "provider_unavailable" {
		t.Fatalf("expected engine errors in detail, got %#v", detail["engineErrors"])
	}
}

func TestRoutingErrorResponseOmitsEmptyDetail(t *testing.T) {
	status, body := routingErrorResponse(routingErrorFromCode("invalid_request", "bad request"), nil)
	if status != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, status)
	}
	if _, ok := body["detail"]; ok {
		t.Fatalf("expected no detail field, got %#v", body)
	}
	data, ok := body["data"].(map[string]any)
	if !ok || data["code"] != "invalid_request" {
		t.Fatalf("expected routing error data, got %#v", body["data"])
	}
}

func TestRoutingSegmentValidation(t *testing.T) {
	request := testRoutingRequest()
	candidate := testRoutingCandidate(request)
	candidate.Segments = candidate.Segments[:1]

	_, err := normalizeRoutingRouteOutput(request, pluginRoutingRouteOutput{
		Candidates: []pluginRoutingCandidate{candidate},
	}, testRoutingPlugin, nil)
	if err == nil {
		t.Fatal("expected invalid segment count to reject candidate")
	}
	routingErr := err.(*routingError)
	if routingErr.Code != "no_route" || routingErr.HTTPStatus != http.StatusUnprocessableEntity {
		t.Fatalf("unexpected error: %#v", routingErr)
	}
}

func TestRoutingCanonicalPolylineValidation(t *testing.T) {
	request := testRoutingRequest()
	candidate := testRoutingCandidate(request)
	candidate.Segments[0].Geometry.Precision = 5

	_, err := normalizeRoutingRouteOutput(request, pluginRoutingRouteOutput{
		Candidates: []pluginRoutingCandidate{candidate},
	}, testRoutingPlugin, nil)
	if err == nil {
		t.Fatal("expected invalid precision to reject candidate")
	}

	candidate = testRoutingCandidate(request)
	candidate.Segments[0].Geometry.Coordinates = "not a polyline"
	_, err = normalizeRoutingRouteOutput(request, pluginRoutingRouteOutput{
		Candidates: []pluginRoutingCandidate{candidate},
	}, testRoutingPlugin, nil)
	if err == nil {
		t.Fatal("expected invalid polyline to reject candidate")
	}
}

func TestRoutingElevationStatusNormalization(t *testing.T) {
	output := pluginRoutingElevationOutput{}
	normalizeRoutingElevationOutput(&output)

	if output.Status != "empty" {
		t.Fatalf("expected empty status, got %q", output.Status)
	}
	if output.Heights == nil || len(output.Heights) != 0 {
		t.Fatalf("expected empty heights slice, got %#v", output.Heights)
	}

	output = pluginRoutingElevationOutput{Heights: []float64{400, 401}}
	normalizeRoutingElevationOutput(&output)
	if output.Status != "included" {
		t.Fatalf("expected included status, got %q", output.Status)
	}

	output = pluginRoutingElevationOutput{Heights: []float64{400, math.NaN()}}
	normalizeRoutingElevationOutput(&output)
	if output.Status != "partial" {
		t.Fatalf("expected partial status, got %q", output.Status)
	}
}

func TestRoutingCandidateElevationNormalization(t *testing.T) {
	request := testRoutingRequest()
	candidate := testRoutingCandidate(request)
	candidate.Elevation = &pluginRoutingElevation{
		Heights: []float64{400, 401, 402},
	}

	normalized, err := normalizeRoutingRouteOutput(request, pluginRoutingRouteOutput{
		Candidates: []pluginRoutingCandidate{candidate},
	}, testRoutingPlugin, nil)
	if err != nil {
		t.Fatalf("expected candidate elevation to normalize: %v", err)
	}
	elevation := normalized.Candidates[0].Elevation
	if elevation == nil || elevation.Status != "included" || elevation.Source != "route" {
		t.Fatalf("unexpected candidate elevation: %#v", elevation)
	}

	candidate = testRoutingCandidate(request)
	candidate.Elevation = &pluginRoutingElevation{Heights: []float64{400}}
	_, err = normalizeRoutingRouteOutput(request, pluginRoutingRouteOutput{
		Candidates: []pluginRoutingCandidate{candidate},
	}, testRoutingPlugin, nil)
	if err == nil {
		t.Fatal("expected mismatched elevation length to reject candidate")
	}
}

func TestRoutingHostLimitEnforcement(t *testing.T) {
	request := testRoutingRequest()
	request.Anchors = make([]pluginRoutingAnchor, routingMaxAnchors+1)
	for i := range request.Anchors {
		request.Anchors[i] = pluginRoutingAnchor{Lat: 47, Lon: 8}
	}
	if err := validateRoutingRouteRequest(request); err == nil || !strings.Contains(err.Error(), "too many routing anchors") {
		t.Fatalf("expected anchor limit error, got %v", err)
	}

	request = testRoutingRequest()
	request.Options.Alternatives = routingMaxProviderCandidates + 1
	if err := validateRoutingRouteRequest(request); err == nil || !strings.Contains(err.Error(), "too many routing variants") {
		t.Fatalf("expected variant limit error, got %v", err)
	}
	if routingMaxHostRequests <= routingMaxProviderCandidates+1 {
		t.Fatalf("host request budget %d leaves no reserve for profile upload plus %d route calls", routingMaxHostRequests, routingMaxProviderCandidates)
	}

	coords := make([][]float64, routingMaxElevationPoints+1)
	for i := range coords {
		coords[i] = []float64{47, 8}
	}
	elevationRequest := pluginRoutingElevationRequest{
		EncodedPolyline: string(routingPolylineCodec.EncodeCoords(nil, coords)),
	}
	err := validateRoutingElevationRequest(elevationRequest)
	if err == nil {
		t.Fatalf("expected elevation point limit error, got %v", err)
	}
	if routingErr := err.(*routingError); routingErr.Code != "point_limit_exceeded" {
		t.Fatalf("expected point limit error, got %#v", routingErr)
	}

	err = validateRoutingElevationRequest(pluginRoutingElevationRequest{EncodedPolyline: "not a polyline"})
	if err == nil {
		t.Fatal("expected invalid request polyline error")
	}
	if routingErr := err.(*routingError); routingErr.HTTPStatus != http.StatusBadRequest {
		t.Fatalf("expected bad request for invalid request polyline, got %#v", routingErr)
	}
}

func TestRoutingNativeCandidateCountUsesManifestBound(t *testing.T) {
	plugin := pluginsystem.LocalPlugin{Manifest: pluginsystem.Manifest{Metadata: map[string]any{
		"routing": map[string]any{"supportsAlternatives": true, "maxAlternatives": float64(3)},
	}}}
	if got := routingNativeCandidateCount(plugin, 4, true); got != 3 {
		t.Fatalf("native candidate count = %d, want 3", got)
	}
	if got := routingNativeCandidateCount(plugin, 4, false); got != 1 {
		t.Fatalf("implicit request candidate count = %d, want 1", got)
	}
}

func TestRoutingProviderCandidateTargetOverfetchesReferencePrimary(t *testing.T) {
	tests := []struct {
		name         string
		desired      int
		hasReference bool
		want         int
	}{
		{name: "ordinary request", desired: 3, want: 3},
		{name: "single variant excludes reference", desired: 1, hasReference: true, want: 2},
		{name: "maximum variants retain provider headroom", desired: routingMaxVariants, hasReference: true, want: routingMaxProviderCandidates},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := routingProviderCandidateTarget(test.desired, test.hasReference); got != test.want {
				t.Fatalf("provider candidate target = %d, want %d", got, test.want)
			}
		})
	}
}

func TestComposeRoutingCandidatePreservesSegmentProvenance(t *testing.T) {
	first := testSingleSegmentCandidate(
		pluginRoutingAnchor{Lat: 47, Lon: 8},
		pluginRoutingAnchor{Lat: 47.01, Lon: 8.01},
		"valhalla",
	)
	second := testSingleSegmentCandidate(
		pluginRoutingAnchor{Lat: 47.01, Lon: 8.01},
		pluginRoutingAnchor{Lat: 47.02, Lon: 8.03},
		"brouter",
	)
	composed, err := composeRoutingCandidate([]pluginRoutingCandidate{first, second})
	if err != nil {
		t.Fatalf("compose route: %v", err)
	}
	if composed.CompositionMode != "segment_composed" {
		t.Fatalf("composition mode = %q", composed.CompositionMode)
	}
	if len(composed.Segments) != 2 || composed.Segments[0].Provenance.PluginID != "valhalla" || composed.Segments[1].Provenance.PluginID != "brouter" {
		t.Fatalf("segment provenance was not preserved: %#v", composed.Segments)
	}
	if composed.Segments[1].FromAnchor != 1 || composed.Segments[1].ToAnchor != 2 {
		t.Fatalf("segment indexes were not rebased: %#v", composed.Segments[1])
	}
}

func TestCurateRoutingCandidatesRemovesGeometricDuplicates(t *testing.T) {
	start := pluginRoutingAnchor{Lat: 47, Lon: 8}
	end := pluginRoutingAnchor{Lat: 47.02, Lon: 8.02}
	primary := testSingleSegmentCandidate(start, end, "valhalla")
	duplicate := primary
	duplicate.PluginID = "brouter"
	different := testSingleSegmentCandidate(start, end, "brouter")
	different.Geometry = geometryPointer(testGeometry([][]float64{{47, 8}, {47.015, 8.0}, {47.02, 8.02}}))
	different.Segments[0].Geometry = *different.Geometry
	selected := curateRoutingCandidates([]pluginRoutingCandidate{primary, duplicate, different}, 3)
	if len(selected) != 2 {
		t.Fatalf("curated candidates = %d, want 2", len(selected))
	}
}

func TestCurateRoutingCandidatesExcludesReferenceRoute(t *testing.T) {
	start := pluginRoutingAnchor{Lat: 47, Lon: 8}
	end := pluginRoutingAnchor{Lat: 47.02, Lon: 8.02}
	reference := testSingleSegmentCandidate(start, end, "existing-route")
	matching := reference
	matching.PluginID = "primary"
	different := testSingleSegmentCandidate(start, end, "alternative")
	different.Geometry = geometryPointer(testGeometry([][]float64{{47, 8}, {47.015, 8.0}, {47.02, 8.02}}))
	different.Segments[0].Geometry = *different.Geometry

	selected := curateRoutingCandidates(
		[]pluginRoutingCandidate{matching, different},
		3,
		reference,
	)
	if len(selected) != 1 || selected[0].PluginID != "alternative" {
		t.Fatalf("reference-like candidate was not removed: %#v", selected)
	}
}

func TestRoutingElevationShortlistKeepsLowerRankedDiverseCandidate(t *testing.T) {
	start := pluginRoutingAnchor{Lat: 47, Lon: 8}
	end := pluginRoutingAnchor{Lat: 47.02, Lon: 8.02}
	primary := testSingleSegmentCandidate(start, end, "primary")
	primary.Summary.Duration = 100
	candidates := make([]pluginRoutingCandidate, 0, 7)
	for index := 0; index < 6; index++ {
		duplicate := primary
		duplicate.PluginID = fmt.Sprintf("duplicate-%d", index)
		duplicate.Summary.Duration += float64(index)
		candidates = append(candidates, duplicate)
	}
	diverse := testSingleSegmentCandidate(start, end, "diverse")
	diverse.Summary.Duration = 1_000
	diverse.Geometry = geometryPointer(testGeometry([][]float64{{47, 8}, {47.015, 8.0}, {47.02, 8.02}}))
	diverse.Segments[0].Geometry = *diverse.Geometry
	candidates = append(candidates, diverse)

	shortlist := routingElevationCandidateShortlist(candidates, 3)
	foundDiverse := false
	for _, candidate := range shortlist {
		foundDiverse = foundDiverse || candidate.PluginID == "diverse"
	}
	if !foundDiverse {
		t.Fatalf("diverse candidate below the score-only cutoff was lost: %#v", shortlist)
	}
}

func TestRoutingShortRouteReductionThreshold(t *testing.T) {
	short := []pluginRoutingAnchor{{Lat: 47, Lon: 8}, {Lat: 47.001, Lon: 8}}
	long := []pluginRoutingAnchor{{Lat: 47, Lon: 8}, {Lat: 47.01, Lon: 8}}
	if routingRouteLengthMeters(short) > routingShortRouteMeters {
		t.Fatalf("expected short route below threshold")
	}
	if routingRouteLengthMeters(long) <= routingShortRouteMeters {
		t.Fatalf("expected long route above threshold")
	}
	request := testRoutingRequest()
	request.Anchors = short
	request.Options.Alternatives = 4
	if got := routingSegmentNativeCandidateCount(request, 0); got != 1 {
		t.Fatalf("short segment native candidate count = %d, want 1", got)
	}
	request.Anchors = long
	if got := routingSegmentNativeCandidateCount(request, 0); got != 4 {
		t.Fatalf("long segment native candidate count = %d, want 4", got)
	}
}

func TestRoutingSegmentFanoutWorkIsBounded(t *testing.T) {
	runtimes := make([]routingEngineRuntime, routingMaxEngines)
	for index := range runtimes {
		request := testRoutingRequest()
		request.Options.Alternatives = routingMaxProviderCandidates
		request.Anchors = []pluginRoutingAnchor{
			{Lat: 47.00, Lon: 8}, {Lat: 47.01, Lon: 8}, {Lat: 47.02, Lon: 8},
			{Lat: 47.03, Lon: 8}, {Lat: 47.04, Lon: 8},
		}
		runtimes[index].Request = request
	}
	if work := routingSegmentFanoutWork(runtimes); work != routingMaxFanoutWork {
		t.Fatalf("fanout work = %d, want budget boundary %d", work, routingMaxFanoutWork)
	}
	runtimes[0].Request.Anchors = append(runtimes[0].Request.Anchors, pluginRoutingAnchor{Lat: 47.05, Lon: 8})
	if work := routingSegmentFanoutWork(runtimes); work <= routingMaxFanoutWork {
		t.Fatalf("fanout work = %d, expected it to exceed %d", work, routingMaxFanoutWork)
	}
}

func TestDiverseRoutingBeamsPreserveDifferentSegmentChoices(t *testing.T) {
	beams := []routingCandidateComposition{
		{ChoiceKeys: []string{"a", "a", "a"}, Score: 1},
		{ChoiceKeys: []string{"a", "a", "b"}, Score: 2},
		{ChoiceKeys: []string{"a", "b", "a"}, Score: 3},
		{ChoiceKeys: []string{"b", "b", "b"}, Score: 4},
	}
	selected := diverseRoutingBeams(beams, 2)
	if len(selected) != 2 {
		t.Fatalf("selected %d beams, want 2", len(selected))
	}
	if routingBeamChoiceDistance(selected[0].ChoiceKeys, selected[1].ChoiceKeys) != 3 {
		t.Fatalf("quality-only beam survived instead of the diverse beam: %#v", selected)
	}
}

func TestRoutingSegmentFanoutWorkCountsPreparedProfileOncePerEngine(t *testing.T) {
	request := testRoutingRequest()
	request.Options.Alternatives = routingMaxProviderCandidates
	request.Profile.ContentBase64 = "cHJvZmlsZQ=="
	request.Anchors = make([]pluginRoutingAnchor, routingMaxVariantAnchors)
	for index := range request.Anchors {
		request.Anchors[index] = pluginRoutingAnchor{Lat: 47 + float64(index)*0.01, Lon: 8}
	}
	runtime := routingEngineRuntime{
		Request: request,
		Plugin: pluginsystem.LocalPlugin{Manifest: pluginsystem.Manifest{Capabilities: []pluginsystem.CapabilityManifest{{
			Name: "profile_prepare", Version: "v1", Export: "profile_prepare_v1",
		}}}},
	}
	work := routingSegmentFanoutWork([]routingEngineRuntime{runtime})
	want := (routingMaxVariantAnchors-1)*routingMaxProviderCandidates + 1
	if work != want {
		t.Fatalf("prepared-profile fanout work = %d, want %d", work, want)
	}
}

func TestRoutingSegmentFanoutWorkCountsProfilePerSegmentWithoutPreparation(t *testing.T) {
	request := testRoutingRequest()
	request.Options.Alternatives = routingMaxProviderCandidates
	request.Profile.ContentBase64 = "cHJvZmlsZQ=="
	request.Anchors = make([]pluginRoutingAnchor, routingMaxVariantAnchors)
	for index := range request.Anchors {
		request.Anchors[index] = pluginRoutingAnchor{Lat: 47 + float64(index)*0.01, Lon: 8}
	}
	work := routingSegmentFanoutWork([]routingEngineRuntime{{Request: request}})
	want := (routingMaxVariantAnchors - 1) * (routingMaxProviderCandidates + 1)
	if work != want {
		t.Fatalf("unprepared-profile fanout work = %d, want %d", work, want)
	}
}

func TestRoutingSegmentExecutionWorkCountsFailedPreparationPerSegment(t *testing.T) {
	request := testRoutingRequest()
	request.Options.Alternatives = routingMaxProviderCandidates
	request.Profile.ContentBase64 = "cHJvZmlsZQ=="
	request.Anchors = make([]pluginRoutingAnchor, routingMaxVariantAnchors)
	for index := range request.Anchors {
		request.Anchors[index] = pluginRoutingAnchor{Lat: 47 + float64(index)*0.01, Lon: 8}
	}
	runtime := routingEngineRuntime{
		Request: request,
		Plugin: pluginsystem.LocalPlugin{Manifest: pluginsystem.Manifest{Capabilities: []pluginsystem.CapabilityManifest{{
			Name: "profile_prepare", Version: "v1", Export: "profile_prepare_v1",
		}}}},
	}
	if estimated := routingSegmentFanoutWork([]routingEngineRuntime{runtime}); estimated != 61 {
		t.Fatalf("preparation-capable estimate = %d, want 61", estimated)
	}
	if actual := routingSegmentExecutionWork([]routingEngineRuntime{runtime}); actual != 75 {
		t.Fatalf("failed-preparation execution work = %d, want 75", actual)
	}
}

func TestRoutingSelectionsKeepExplicitEngineListAuthoritative(t *testing.T) {
	settings := routingSettings{
		PrimaryRoutePluginID: "primary",
	}
	explicit, automatic, err := routingSelectionsForRequest(nil, "", pluginRoutingRouteHTTPInput{
		PluginID: "primary",
		Engines:  []routingEngineSelection{{PluginID: "primary", InstanceID: "primary-instance"}},
	}, settings, true)
	if err != nil {
		t.Fatalf("resolve explicit engine selections: %v", err)
	}
	if len(explicit) != 1 || explicit[0].PluginID != "primary" || explicit[0].InstanceID != "primary-instance" {
		t.Fatalf("explicit engine list was expanded with configured comparisons: %#v", explicit)
	}
	if automatic {
		t.Fatal("explicit engine list was marked as automatic discovery")
	}

	defaults, automatic, err := routingSelectionsForRequest(nil, "", pluginRoutingRouteHTTPInput{
		PluginID: "primary",
	}, settings, false)
	if err != nil {
		t.Fatalf("resolve single-engine selection: %v", err)
	}
	if len(defaults) != 1 || defaults[0].PluginID != "primary" || automatic {
		t.Fatalf("single-engine request selection = %#v automatic=%t", defaults, automatic)
	}
}

func TestRoutingSelectionsDiscoverEnabledRouteEnginesDeterministically(t *testing.T) {
	app, auth := newRoutingHandlerTestApp(t, map[string]bool{
		"primary": true,
		"alpha":   true,
		"bravo":   true,
		"charlie": true,
		"delta":   true,
		"echo":    true,
	}, "segment")
	disabled, err := app.FindFirstRecordByData("plugin_instances", "plugin_id", "bravo")
	if err != nil {
		t.Fatalf("find engine to disable: %v", err)
	}
	disabled.Set("enabled", false)
	if err := app.Save(disabled); err != nil {
		t.Fatalf("disable comparison engine: %v", err)
	}
	selections, automatic, err := routingSelectionsForRequest(
		app,
		auth.Id,
		pluginRoutingRouteHTTPInput{PluginID: "primary", EngineMode: "parallel"},
		routingSettings{PrimaryRoutePluginID: "primary"},
		true,
	)
	if err != nil {
		t.Fatalf("discover route engines: %v", err)
	}
	if !automatic {
		t.Fatal("host-discovered selection was not marked automatic")
	}
	got := make([]string, 0, len(selections))
	for _, selection := range selections {
		got = append(got, selection.PluginID)
	}
	orderedInstances, err := app.FindRecordsByFilter("plugin_instances", "user={:user} && enabled=true", "+created,+id", -1, 0, dbx.Params{"user": auth.Id})
	if err != nil {
		t.Fatalf("resolve engine setup order: %v", err)
	}
	want := []string{"primary"}
	for _, instance := range orderedInstances {
		pluginID := instance.GetString("plugin_id")
		if pluginID == "primary" {
			continue
		}
		want = append(want, pluginID)
		if len(want) == routingMaxEngines {
			break
		}
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("discovered selections = %v, want setup order %v", got, want)
	}
}

func TestRoutingEngineDiscoveryChoosesOldestEnabledInstanceDeterministically(t *testing.T) {
	app, auth := newRoutingHandlerTestApp(t, map[string]bool{"route-a": true}, "segment")
	instances, err := app.FindRecordsByFilter("plugin_instances", "user={:user} && plugin_id='route-a'", "+created,+id", -1, 0, dbx.Params{"user": auth.Id})
	if err != nil || len(instances) != 1 {
		t.Fatalf("find original route instance: count=%d err=%v", len(instances), err)
	}
	duplicate := core.NewRecord(instances[0].Collection())
	duplicate.Set("user", auth.Id)
	duplicate.Set("plugin_id", "route-a")
	duplicate.Set("enabled", true)
	duplicate.Set("auth", map[string]any{})
	duplicate.Set("config", map[string]any{})
	duplicate.Set("status", "configured")
	if err := app.Save(duplicate); err != nil {
		t.Fatalf("save duplicate route instance: %v", err)
	}
	instances, err = app.FindRecordsByFilter("plugin_instances", "user={:user} && plugin_id='route-a' && enabled=true", "+created,+id", -1, 0, dbx.Params{"user": auth.Id})
	if err != nil || len(instances) != 2 {
		t.Fatalf("find enabled route instances: count=%d err=%v", len(instances), err)
	}
	wantInstanceID := instances[0].Id

	engines, err := routingEngines(app, auth.Id)
	if err != nil {
		t.Fatalf("discover routing engines: %v", err)
	}
	if len(engines) != 1 || engines[0].InstanceID != wantInstanceID {
		t.Fatalf("engine instance = %#v, want %s", engines, wantInstanceID)
	}
	selected, err := routingEnabledPluginInstance(app, auth.Id, "route-a", "")
	if err != nil || selected == nil || selected.Id != wantInstanceID {
		t.Fatalf("runtime instance = %#v err=%v, want %s", selected, err, wantInstanceID)
	}
}

func TestAutomaticRoutingFanoutKeepsAllEnginesBeforeNativeAlternatives(t *testing.T) {
	request := pluginRoutingRouteRequest{Options: pluginRoutingOptions{Alternatives: 4}}
	for index := 0; index < 10; index++ {
		request.Anchors = append(request.Anchors, pluginRoutingAnchor{Lat: 47 + float64(index)*0.01, Lon: 8})
	}
	runtimes := []routingEngineRuntime{{Request: request}, {Request: request}, {Request: request}, {Request: request}}
	planned, reduced := routingRuntimesWithinFanout(runtimes, routingSegmentFanoutWork)
	if reduced || len(planned) != 4 {
		t.Fatalf("planned runtimes = %d reduced=%t, want all four engines without an engine-reduction warning", len(planned), reduced)
	}
	gotAlternatives := make([]int, len(planned))
	for index := range planned {
		gotAlternatives[index] = planned[index].Request.Options.Alternatives
	}
	if fmt.Sprint(gotAlternatives) != "[2 2 2 1]" {
		t.Fatalf("planned alternatives = %v, want balanced primary-first allocation", gotAlternatives)
	}
	if work := routingSegmentFanoutWork(planned); work != 63 {
		t.Fatalf("planned work = %d, want 63", work)
	}
}

func TestAutomaticRoutingFanoutMaximizesBaselineEngineCount(t *testing.T) {
	runtime := func(pluginID string, anchors int, profile bool) routingEngineRuntime {
		request := pluginRoutingRouteRequest{Options: pluginRoutingOptions{Alternatives: 3}}
		for index := 0; index < anchors; index++ {
			request.Anchors = append(request.Anchors, pluginRoutingAnchor{Lat: 47 + float64(index)*0.01, Lon: 8})
		}
		if profile {
			request.Profile.ContentBase64 = "cHJvZmlsZQ=="
		}
		return routingEngineRuntime{
			Plugin:  pluginsystem.LocalPlugin{Manifest: pluginsystem.Manifest{ID: pluginID}},
			Request: request,
		}
	}
	runtimes := []routingEngineRuntime{
		runtime("primary", 11, false),
		runtime("expensive", 21, true),
		runtime("cheap-a", 11, false),
		runtime("cheap-b", 11, false),
	}
	planned, reduced := routingRuntimesWithinFanout(runtimes, routingSegmentFanoutWork)
	if !reduced || len(planned) != 3 {
		t.Fatalf("planned runtimes = %d reduced=%t, want primary plus both cheap engines", len(planned), reduced)
	}
	got := []string{planned[0].Plugin.Manifest.ID, planned[1].Plugin.Manifest.ID, planned[2].Plugin.Manifest.ID}
	if strings.Join(got, ",") != "primary,cheap-a,cheap-b" {
		t.Fatalf("baseline engine selection = %v", got)
	}
}

func TestAppendRoutingWarningOnceDoesNotDuplicateFanoutWarning(t *testing.T) {
	warnings := appendRoutingWarningOnce(nil, "routing_parallel_engines_reduced_for_fanout")
	warnings = appendRoutingWarningOnce(warnings, "routing_parallel_engines_reduced_for_fanout")
	if len(warnings) != 1 {
		t.Fatalf("fanout warning was appended %d times: %v", len(warnings), warnings)
	}
}

func TestRoutingDiversityFiltersMostlyOverlappingLocalizedDetour(t *testing.T) {
	straight := make([][]float64, 31)
	detour := make([][]float64, 31)
	for index := range straight {
		lat := 47.0 + float64(index)*0.01
		straight[index] = []float64{lat, 8.0}
		detour[index] = []float64{lat, 8.0}
	}
	detour[14][1] = 8.02
	detour[15][1] = 8.025
	detour[16][1] = 8.02
	left := pluginRoutingCandidate{
		Geometry: geometryPointer(testGeometry(straight)),
		Summary:  pluginRoutingSummary{Distance: 30_000, Duration: 18_000},
	}
	right := pluginRoutingCandidate{
		Geometry: geometryPointer(testGeometry(detour)),
		Summary:  pluginRoutingSummary{Distance: 31_000, Duration: 18_500},
	}
	if !routingCandidatesTooSimilar(left, right) {
		t.Fatal("a route with only a short localized detour must be filtered as mostly overlapping")
	}
}

func TestRoutingDiversityKeepsSustainedAlternative(t *testing.T) {
	straight := make([][]float64, 31)
	alternative := make([][]float64, 31)
	for index := range straight {
		lat := 47.0 + float64(index)*0.01
		straight[index] = []float64{lat, 8.0}
		alternative[index] = []float64{lat, 8.0}
		if index >= 8 && index <= 22 {
			alternative[index][1] = 8.02
		}
	}
	left := pluginRoutingCandidate{
		Geometry: geometryPointer(testGeometry(straight)),
		Summary:  pluginRoutingSummary{Distance: 30_000, Duration: 18_000},
	}
	right := pluginRoutingCandidate{
		Geometry: geometryPointer(testGeometry(alternative)),
		Summary:  pluginRoutingSummary{Distance: 34_000, Duration: 20_000},
	}
	if routingCandidatesTooSimilar(left, right) {
		t.Fatal("a sustained alternative corridor must remain a distinct route variant")
	}
}

func TestRoutingRankingDoesNotRewardMissingElevation(t *testing.T) {
	start := pluginRoutingAnchor{Lat: 47, Lon: 8}
	end := pluginRoutingAnchor{Lat: 47.02, Lon: 8.02}
	withElevation := testSingleSegmentCandidate(start, end, "with-elevation")
	withElevation.Summary.Duration = 100
	withElevation.Summary.ElevationGain = 1000
	withElevation.Elevation = &pluginRoutingElevation{Heights: []float64{0, 1000}}
	withoutElevation := testSingleSegmentCandidate(start, end, "without-elevation")
	withoutElevation.Summary.Duration = 200

	selected := curateRoutingCandidates([]pluginRoutingCandidate{withoutElevation, withElevation}, 1)
	if len(selected) != 1 || selected[0].PluginID != "with-elevation" {
		t.Fatalf("missing elevation distorted ranking: %#v", selected)
	}
}

func TestRoutingProvenanceKeepsClientRequestBeforeMapping(t *testing.T) {
	request := testRoutingRequest()
	request.Preferences = map[string]any{"speedPreference": 0.8}
	request.Profile.NativeConfig = map[string]any{"mapped": true}
	clientRequest := cloneRoutingRouteRequest(request)
	clientRequest.Preferences = map[string]any{"speedPreference": 0.8, "roadPreference": 0.6}
	clientRequest.Profile.NativeConfig = map[string]any{"client": true}
	candidate := testRoutingCandidate(request)
	addRoutingCandidateProvenance(&candidate, request, clientRequest, testRoutingPlugin, nil)
	provenance := candidate.Segments[0].Provenance
	if provenance == nil || provenance.RequestedPreferences["roadPreference"] != 0.6 {
		t.Fatalf("client preferences were not preserved: %#v", provenance)
	}
	if provenance.RequestedNativeConfig["client"] != true || provenance.NativeConfig["mapped"] != true {
		t.Fatalf("client and effective native config were not separated: %#v", provenance)
	}
}

func TestRoutingCandidateIDIncludesSegmentProvenance(t *testing.T) {
	start := pluginRoutingAnchor{Lat: 47, Lon: 8}
	end := pluginRoutingAnchor{Lat: 47.02, Lon: 8.02}
	first := testSingleSegmentCandidate(start, end, "valhalla")
	second := testSingleSegmentCandidate(start, end, "brouter")
	if routingCandidateID(first, "segment") == routingCandidateID(second, "segment") {
		t.Fatal("candidate IDs must change when segment provenance changes")
	}
	if routingCandidateID(first, "segment") != routingCandidateID(first, "segment") {
		t.Fatal("candidate IDs must be stable for the same provenance and geometry")
	}
}

func testRoutingRequest() pluginRoutingRouteRequest {
	return pluginRoutingRouteRequest{
		RoutingMode: "segment",
		Mode:        "foot",
		Profile:     pluginRoutingProfile{Key: "pedestrian"},
		Anchors: []pluginRoutingAnchor{
			{Lat: 47.0, Lon: 8.0},
			{Lat: 47.1, Lon: 8.1},
			{Lat: 47.2, Lon: 8.2},
		},
	}
}

func testRoutingCandidate(request pluginRoutingRouteRequest) pluginRoutingCandidate {
	segments := make([]pluginRoutingSegment, 0, len(request.Anchors)-1)
	for i := 0; i < len(request.Anchors)-1; i++ {
		geometry := testGeometry([][]float64{
			{request.Anchors[i].Lat, request.Anchors[i].Lon},
			{request.Anchors[i+1].Lat, request.Anchors[i+1].Lon},
		})
		segments = append(segments, pluginRoutingSegment{
			FromAnchor: i,
			ToAnchor:   i + 1,
			Geometry:   geometry,
			Distance:   100,
			Duration:   60,
		})
	}
	geometry := testGeometry([][]float64{
		{request.Anchors[0].Lat, request.Anchors[0].Lon},
		{request.Anchors[1].Lat, request.Anchors[1].Lon},
		{request.Anchors[2].Lat, request.Anchors[2].Lon},
	})
	return pluginRoutingCandidate{
		ID:             "plugin-owned-id",
		ProfileKey:     request.Profile.Key,
		Geometry:       &geometry,
		Summary:        pluginRoutingSummary{Distance: 200, Duration: 120},
		Segments:       segments,
		SnappedAnchors: request.Anchors,
	}
}

func testGeometry(coords [][]float64) pluginRoutingGeometry {
	return pluginRoutingGeometry{
		Format:      routingPolylineFormat,
		Precision:   routingPolylinePrecision,
		Coordinates: string(polyline.Codec{Dim: 2, Scale: routingPolylineScale}.EncodeCoords(nil, coords)),
	}
}

func testSingleSegmentCandidate(start, end pluginRoutingAnchor, pluginID string) pluginRoutingCandidate {
	geometry := testGeometry([][]float64{{start.Lat, start.Lon}, {end.Lat, end.Lon}})
	return pluginRoutingCandidate{
		PluginID:   pluginID,
		InstanceID: "instance-" + pluginID,
		Provider:   pluginID,
		Geometry:   &geometry,
		Summary:    pluginRoutingSummary{Distance: 2000, Duration: 1200},
		Segments: []pluginRoutingSegment{{
			FromAnchor: 0, ToAnchor: 1, Geometry: geometry, Distance: 2000, Duration: 1200,
			Provenance: &pluginRoutingSegmentProvenance{PluginID: pluginID, RoutingMode: "segment"},
		}},
		SnappedAnchors: []pluginRoutingAnchor{start, end},
	}
}

func geometryPointer(value pluginRoutingGeometry) *pluginRoutingGeometry {
	return &value
}
