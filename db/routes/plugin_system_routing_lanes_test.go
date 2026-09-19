package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"pocketbase/pluginsystem"
)

func TestRoutingSegmentSessionLaneCounts(t *testing.T) {
	tests := []struct {
		name         string
		engineCount  int
		segmentCount int
		want         []int
	}{
		{name: "no engines", engineCount: 0, segmentCount: 15, want: nil},
		{name: "no segments", engineCount: 2, segmentCount: 0, want: nil},
		{name: "one engine", engineCount: 1, segmentCount: 15, want: []int{8}},
		{name: "two engines", engineCount: 2, segmentCount: 15, want: []int{4, 4}},
		{name: "remainder is primary first", engineCount: 3, segmentCount: 15, want: []int{3, 3, 2}},
		{name: "four engines", engineCount: 4, segmentCount: 15, want: []int{2, 2, 2, 2}},
		{name: "segments cap lanes", engineCount: 3, segmentCount: 2, want: []int{2, 2, 2}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := routingSegmentSessionLaneCounts(test.engineCount, test.segmentCount)
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("lane counts = %v, want %v", got, test.want)
			}
		})
	}
}

func TestRouteSegmentCandidatesBoundsAndReusesWorkerSessionLanes(t *testing.T) {
	tracker := newRoutingLaneTestTracker()
	previousOpener := routingRuntimeSessionOpener
	previousCaller := routingRoutePluginCaller
	routingRuntimeSessionOpener = tracker.openSession
	routingRoutePluginCaller = callRoutingRoutePlugin
	t.Cleanup(func() {
		routingRuntimeSessionOpener = previousOpener
		routingRoutePluginCaller = previousCaller
	})

	request := pluginRoutingRouteRequest{
		RoutingMode: "segment",
		Mode:        "foot",
		Profile:     pluginRoutingProfile{Key: "pedestrian"},
		Options:     pluginRoutingOptions{Alternatives: 1},
	}
	for index := 0; index <= 15; index++ {
		request.Anchors = append(request.Anchors, pluginRoutingAnchor{Lat: float64(index), Lon: 8 + float64(index)/100})
	}
	runtimes := []routingEngineRuntime{
		routingLaneTestRuntime("route-a", request),
		routingLaneTestRuntime("route-b", request),
	}

	ctx, closeSessions := withRoutingPluginSessions(context.Background())
	defer closeSessions()
	// Simulate profile preparation. Segment lane zero must reuse these two base
	// sessions instead of opening a separate lane-zero worker for each engine.
	for _, runtime := range runtimes {
		if _, _, err := routingSessionForContext(ctx, runtime.Plugin, runtime.Instance, pluginsystem.RequestPolicyContext{}); err != nil {
			t.Fatalf("open preparation session for %s: %v", runtime.Plugin.Manifest.ID, err)
		}
	}

	type routeResult struct {
		candidates []pluginRoutingCandidate
		errors     []pluginRoutingEngineError
	}
	done := make(chan routeResult, 1)
	go func() {
		candidates, errors := routeSegmentCandidates(ctx, runtimes, len(request.Anchors), 3, false)
		done <- routeResult{candidates: candidates, errors: errors}
	}()

	select {
	case <-tracker.bothEnginesActive:
		close(tracker.releaseCalls)
	case <-time.After(2 * time.Second):
		close(tracker.releaseCalls)
		<-done
		closeSessions()
		t.Fatal("all worker lanes across both engines did not become active before release")
	}
	result := <-done
	if len(result.errors) != 0 || len(result.candidates) == 0 {
		closeSessions()
		t.Fatalf("segment routing failed: candidates=%d errors=%#v", len(result.candidates), result.errors)
	}
	closeSessions()

	sessions, maxActive := tracker.snapshot()
	if len(sessions) != routingMaxParallelCalls {
		t.Fatalf("opened %d route sessions, want global bound %d", len(sessions), routingMaxParallelCalls)
	}
	if maxActive != routingMaxParallelCalls {
		t.Fatalf("parallel calls peaked at %d, want %d active lanes", maxActive, routingMaxParallelCalls)
	}
	for _, pluginID := range []string{"route-a", "route-b"} {
		for lane := 0; lane < 4; lane++ {
			key := routingLaneTestSessionKey(pluginID, lane)
			session, ok := sessions[key]
			if !ok {
				t.Fatalf("missing deterministic session lane %s", key)
			}
			wantSegments := []int{}
			for segmentIndex := 0; segmentIndex < 15; segmentIndex++ {
				if segmentIndex%4 == lane {
					wantSegments = append(wantSegments, segmentIndex)
				}
			}
			sort.Ints(session.segments)
			if !reflect.DeepEqual(session.segments, wantSegments) {
				t.Errorf("%s handled segments %v, want %v", key, session.segments, wantSegments)
			}
			if len(session.segments) < 3 {
				t.Errorf("%s did not amortize startup: calls=%d", key, len(session.segments))
			}
			if session.closed != 1 {
				t.Errorf("%s closed %d times, want once", key, session.closed)
			}
		}
	}
}

func routingLaneTestRuntime(pluginID string, request pluginRoutingRouteRequest) routingEngineRuntime {
	plugin := pluginsystem.LocalPlugin{Manifest: pluginsystem.Manifest{
		ID: pluginID, Name: pluginID, Type: pluginsystem.PluginTypeRouting,
	}}
	return routingEngineRuntime{
		Plugin: plugin,
		Capability: pluginsystem.CapabilityManifest{
			Name: "route", Version: "v1", Export: "route_v1",
		},
		Request:       cloneRoutingRouteRequest(request),
		ClientRequest: cloneRoutingRouteRequest(request),
	}
}

type routingLaneTestTracker struct {
	mu                sync.Mutex
	sessions          map[string]*routingLaneTestSession
	activeCalls       int
	maxActiveCalls    int
	activeByPlugin    map[string]int
	bothEnginesOnce   sync.Once
	bothEnginesActive chan struct{}
	releaseCalls      chan struct{}
}

type routingLaneTestSession struct {
	callMu   sync.Mutex
	tracker  *routingLaneTestTracker
	pluginID string
	segments []int
	closed   int
}

type routingLaneTestSessionSnapshot struct {
	segments []int
	closed   int
}

func newRoutingLaneTestTracker() *routingLaneTestTracker {
	return &routingLaneTestTracker{
		sessions:          map[string]*routingLaneTestSession{},
		activeByPlugin:    map[string]int{},
		bothEnginesActive: make(chan struct{}),
		releaseCalls:      make(chan struct{}),
	}
}

func (t *routingLaneTestTracker) openSession(ctx context.Context, plugin pluginsystem.LocalPlugin, _ pluginsystem.RequestPolicyContext) (pluginsystem.RuntimeSession, error) {
	lane := 0
	if value, ok := ctx.Value(routingSessionLaneContextKey{}).(int); ok {
		lane = value
	}
	session := &routingLaneTestSession{tracker: t, pluginID: plugin.Manifest.ID}
	key := routingLaneTestSessionKey(plugin.Manifest.ID, lane)
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, exists := t.sessions[key]; exists {
		return nil, fmt.Errorf("duplicate session open for %s", key)
	}
	t.sessions[key] = session
	return session, nil
}

func (s *routingLaneTestSession) Call(ctx context.Context, _ string, input []byte, _ pluginsystem.RuntimeCallOptions) ([]byte, error) {
	s.callMu.Lock()
	defer s.callMu.Unlock()

	var payload pluginRoutingRouteInput
	if err := json.Unmarshal(input, &payload); err != nil {
		return nil, err
	}
	segmentIndex := int(payload.Request.Anchors[0].Lat)
	s.tracker.mu.Lock()
	s.segments = append(s.segments, segmentIndex)
	s.tracker.activeCalls++
	s.tracker.activeByPlugin[s.pluginID]++
	if s.tracker.activeCalls > s.tracker.maxActiveCalls {
		s.tracker.maxActiveCalls = s.tracker.activeCalls
	}
	activeEngines := 0
	for _, count := range s.tracker.activeByPlugin {
		if count > 0 {
			activeEngines++
		}
	}
	if activeEngines >= 2 && s.tracker.activeCalls >= routingMaxParallelCalls {
		s.tracker.bothEnginesOnce.Do(func() { close(s.tracker.bothEnginesActive) })
	}
	s.tracker.mu.Unlock()
	defer func() {
		s.tracker.mu.Lock()
		s.tracker.activeCalls--
		s.tracker.activeByPlugin[s.pluginID]--
		s.tracker.mu.Unlock()
	}()

	select {
	case <-s.tracker.releaseCalls:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	request := payload.Request
	geometry := testGeometry([][]float64{
		{request.Anchors[0].Lat, request.Anchors[0].Lon},
		{request.Anchors[1].Lat, request.Anchors[1].Lon},
	})
	output := pluginRoutingRouteOutput{Candidates: []pluginRoutingCandidate{{
		ProfileKey: request.Profile.Key,
		Geometry:   &geometry,
		Summary:    pluginRoutingSummary{Distance: 100, Duration: 60},
		Segments: []pluginRoutingSegment{{
			FromAnchor: 0, ToAnchor: 1, Geometry: geometry, Distance: 100, Duration: 60,
		}},
		SnappedAnchors: append([]pluginRoutingAnchor(nil), request.Anchors...),
	}}}
	return json.Marshal(output)
}

func (s *routingLaneTestSession) Close(context.Context) error {
	s.tracker.mu.Lock()
	s.closed++
	s.tracker.mu.Unlock()
	return nil
}

func (t *routingLaneTestTracker) snapshot() (map[string]routingLaneTestSessionSnapshot, int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	sessions := make(map[string]routingLaneTestSessionSnapshot, len(t.sessions))
	for key, session := range t.sessions {
		sessions[key] = routingLaneTestSessionSnapshot{
			segments: append([]int(nil), session.segments...),
			closed:   session.closed,
		}
	}
	return sessions, t.maxActiveCalls
}

func routingLaneTestSessionKey(pluginID string, lane int) string {
	return fmt.Sprintf("%s:lane-%d", pluginID, lane)
}
