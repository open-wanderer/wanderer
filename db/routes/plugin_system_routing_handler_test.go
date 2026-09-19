package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"pocketbase/pluginsystem"
	"pocketbase/util"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func TestRoutingCandidateHandlerIsFeatureGated(t *testing.T) {
	app, auth := newRoutingHandlerTestApp(t, map[string]bool{"route-a": true}, "segment")
	status, body := callRoutingHandler(t, app, auth, PluginSystemRoutingRouteCandidates, `{}`)
	if status != http.StatusForbidden || !strings.Contains(body, "not enabled") {
		t.Fatalf("candidate handler response = %d %s", status, body)
	}
}

func TestRoutingHandlerIsDisabledByUserSetting(t *testing.T) {
	app, auth := newRoutingHandlerTestApp(t, map[string]bool{"route-a": true}, "segment")
	if err := saveRoutingSettingsConfig(app, routingScopeUser, auth.Id, map[string]any{
		"exposedFeatures": map[string]bool{"routing": false},
	}); err != nil {
		t.Fatalf("disable routing: %v", err)
	}
	var calls atomic.Int32
	withRoutingCaller(t, func(_ context.Context, _ pluginsystem.LocalPlugin, _ pluginsystem.CapabilityManifest, _ *core.Record, _ map[string]any, _ map[string]any, _ pluginRoutingRouteRequest) (pluginRoutingRouteOutput, error) {
		calls.Add(1)
		return pluginRoutingRouteOutput{}, nil
	})

	for _, tc := range []struct {
		name    string
		handler func(*core.RequestEvent) error
		body    string
	}{
		{name: "route", handler: PluginSystemRoutingRoute, body: routingHandlerRequest("segment", false, 1, "route-a")},
		{name: "route candidates", handler: PluginSystemRoutingRouteCandidates, body: routingHandlerRequest("segment", true, 1, "route-a")},
		{name: "profile preparation", handler: PluginSystemRoutingProfilePrepare, body: `{}`},
		{name: "elevation", handler: PluginSystemRoutingElevation, body: `{}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body := callRoutingHandler(t, app, auth, tc.handler, tc.body)
			if status != http.StatusForbidden || !strings.Contains(body, `"code":"routing_disabled"`) {
				t.Fatalf("disabled routing handler response = %d %s", status, body)
			}
		})
	}
	if calls.Load() != 0 {
		t.Fatalf("disabled routing called a plugin %d time(s)", calls.Load())
	}
}

func TestRoutingSettingsCannotEnableWithoutEnabledRoutePlugin(t *testing.T) {
	app, auth := newRoutingHandlerTestApp(t, map[string]bool{"route-a": true}, "segment")
	instances, err := app.FindRecordsByFilter("plugin_instances", "user='"+auth.Id+"'", "", -1, 0)
	if err != nil {
		t.Fatalf("find routing plugin instances: %v", err)
	}
	for _, instance := range instances {
		instance.Set("enabled", false)
		if err := app.Save(instance); err != nil {
			t.Fatalf("disable routing plugin instance: %v", err)
		}
	}

	request := `{"exposedFeatures":{"routing":true}}`
	status, body := callRoutingHandler(t, app, auth, PluginSystemRoutingSettingsPatch, request)
	if status != http.StatusBadRequest || !strings.Contains(body, `"code":"routing_plugin_required"`) {
		t.Fatalf("routing enable without plugin response = %d %s", status, body)
	}

	instances[0].Set("enabled", true)
	if err := app.Save(instances[0]); err != nil {
		t.Fatalf("enable routing plugin instance: %v", err)
	}
	status, body = callRoutingHandler(t, app, auth, PluginSystemRoutingSettingsPatch, request)
	if status != http.StatusOK {
		t.Fatalf("routing enable with plugin response = %d %s", status, body)
	}
}

func TestRoutingHandlerRejectsExplicitUnavailableViaMode(t *testing.T) {
	app, auth := newRoutingHandlerTestApp(t, map[string]bool{"route-a": false}, "segment")
	status, body := callRoutingHandler(t, app, auth, PluginSystemRoutingRoute, routingHandlerRequest("via", false, 1, "route-a"))
	if status != http.StatusUnprocessableEntity || !strings.Contains(body, `"code":"routing_mode_unavailable"`) {
		t.Fatalf("via handler response = %d %s", status, body)
	}
	if !strings.Contains(body, `"pluginId":"route-a"`) {
		t.Fatalf("via rejection omitted the skipped engine: %s", body)
	}
}

func TestRoutingHandlerFallsBackAndUsesSingleCallFastPath(t *testing.T) {
	app, auth := newRoutingHandlerTestApp(t, map[string]bool{"route-a": false}, "via")
	callCount := 0
	withRoutingCaller(t, func(_ context.Context, plugin pluginsystem.LocalPlugin, _ pluginsystem.CapabilityManifest, _ *core.Record, _ map[string]any, _ map[string]any, request pluginRoutingRouteRequest) (pluginRoutingRouteOutput, error) {
		callCount++
		if len(request.Anchors) != 3 {
			return pluginRoutingRouteOutput{}, fmt.Errorf("fast path received %d anchors, want 3", len(request.Anchors))
		}
		return pluginRoutingRouteOutput{Candidates: []pluginRoutingCandidate{routingHandlerCandidate(request, plugin.Manifest.ID)}}, nil
	})

	status, body := callRoutingHandler(t, app, auth, PluginSystemRoutingRoute, routingHandlerRequest("", false, 1, "route-a"))
	if status != http.StatusOK || !strings.Contains(body, "routing_mode_fallback") {
		t.Fatalf("fallback handler response = %d %s", status, body)
	}
	if callCount != 1 {
		t.Fatalf("normal segment route used %d plugin calls, want 1", callCount)
	}
}

func TestRoutingHandlerOverfetchesProviderPrimaryForReferencedVariants(t *testing.T) {
	app, auth := newRoutingHandlerTestApp(t, map[string]bool{"route-a": true}, "segment")
	setRoutingHandlerTestAlternatives(t, app, "route-a", 4)
	var calls atomic.Int32
	withRoutingCaller(t, func(_ context.Context, plugin pluginsystem.LocalPlugin, _ pluginsystem.CapabilityManifest, _ *core.Record, _ map[string]any, _ map[string]any, request pluginRoutingRouteRequest) (pluginRoutingRouteOutput, error) {
		if request.Options.Alternatives != 4 {
			return pluginRoutingRouteOutput{}, fmt.Errorf("provider candidate count = %d, want 4", request.Options.Alternatives)
		}
		calls.Add(1)
		return pluginRoutingRouteOutput{Candidates: []pluginRoutingCandidate{routingHandlerCandidate(request, plugin.Manifest.ID)}}, nil
	})

	var request map[string]any
	if err := json.Unmarshal([]byte(routingHandlerRequest("segment", true, 3, "route-a")), &request); err != nil {
		t.Fatalf("decode routing request: %v", err)
	}
	request["referenceGeometry"] = testGeometry([][]float64{{47, 8.1}, {47.04, 8.1}})
	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("encode routing request: %v", err)
	}
	status, responseBody := callRoutingHandler(t, app, auth, PluginSystemRoutingRoute, string(payload))
	if status != http.StatusOK || !strings.Contains(responseBody, "routing_variants_fewer_than_requested") {
		t.Fatalf("referenced variant response = %d %s", status, responseBody)
	}
	if calls.Load() != 2 {
		t.Fatalf("provider route calls = %d, want 2", calls.Load())
	}
}

func TestRoutingHandlerIgnoresClientPreparedProfileKey(t *testing.T) {
	app, auth := newRoutingHandlerTestApp(t, map[string]bool{"route-a": true}, "segment")
	withRoutingCaller(t, func(_ context.Context, plugin pluginsystem.LocalPlugin, _ pluginsystem.CapabilityManifest, _ *core.Record, _ map[string]any, _ map[string]any, request pluginRoutingRouteRequest) (pluginRoutingRouteOutput, error) {
		if request.Profile.PreparedKey != "" {
			return pluginRoutingRouteOutput{}, fmt.Errorf("route accepted client prepared key %q", request.Profile.PreparedKey)
		}
		return pluginRoutingRouteOutput{Candidates: []pluginRoutingCandidate{routingHandlerCandidate(request, plugin.Manifest.ID)}}, nil
	})

	var request map[string]any
	if err := json.Unmarshal([]byte(routingHandlerRequest("segment", false, 1, "route-a")), &request); err != nil {
		t.Fatalf("decode routing request: %v", err)
	}
	request["profile"].(map[string]any)["preparedKey"] = "client-controlled"
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("encode routing request: %v", err)
	}
	if status, responseBody := callRoutingHandler(t, app, auth, PluginSystemRoutingRoute, string(body)); status != http.StatusOK {
		t.Fatalf("route response = %d %s", status, responseBody)
	}
}

func TestRoutingHandlerDiscoversAndComposesAcrossEngines(t *testing.T) {
	app, auth := newRoutingHandlerTestApp(t, map[string]bool{"route-a": true, "route-b": true}, "segment")
	withRoutingCaller(t, func(_ context.Context, plugin pluginsystem.LocalPlugin, _ pluginsystem.CapabilityManifest, _ *core.Record, _ map[string]any, _ map[string]any, request pluginRoutingRouteRequest) (pluginRoutingRouteOutput, error) {
		return pluginRoutingRouteOutput{Candidates: []pluginRoutingCandidate{routingHandlerCandidate(request, plugin.Manifest.ID)}}, nil
	})

	body := `{
		"pluginId":"route-a",
		"engineMode":"parallel",
		"desiredVariants":3,
		"requestVariants":true,
		"routingMode":"segment",
		"anchors":[{"lat":47.0,"lon":8.0},{"lat":47.02,"lon":8.0},{"lat":47.04,"lon":8.0}],
		"mode":"foot",
		"profile":{"key":"pedestrian"}
	}`
	status, responseBody := callRoutingHandler(t, app, auth, PluginSystemRoutingRoute, body)
	if status != http.StatusOK {
		t.Fatalf("parallel handler response = %d %s", status, responseBody)
	}
	var response pluginRoutingRouteOutput
	if err := json.Unmarshal([]byte(responseBody), &response); err != nil {
		t.Fatalf("decode handler response: %v", err)
	}
	foundMixed := false
	for _, candidate := range response.Candidates {
		if candidate.CompositionMode != "segment_composed" || len(candidate.Segments) != 2 {
			continue
		}
		if candidate.Segments[0].Provenance != nil && candidate.Segments[1].Provenance != nil &&
			candidate.Segments[0].Provenance.PluginID != candidate.Segments[1].Provenance.PluginID {
			foundMixed = true
			break
		}
	}
	if !foundMixed {
		t.Fatalf("handler returned no cross-engine composition: %s", responseBody)
	}
}

func TestRoutingHandlerAppliesVariantRateLimit(t *testing.T) {
	app, auth := newRoutingHandlerTestApp(t, map[string]bool{"route-a": true}, "segment")
	withRoutingCaller(t, func(_ context.Context, plugin pluginsystem.LocalPlugin, _ pluginsystem.CapabilityManifest, _ *core.Record, _ map[string]any, _ map[string]any, request pluginRoutingRouteRequest) (pluginRoutingRouteOutput, error) {
		return pluginRoutingRouteOutput{Candidates: []pluginRoutingCandidate{routingHandlerCandidate(request, plugin.Manifest.ID)}}, nil
	})
	previousLimiter := routingVariantRateLimiter
	routingVariantRateLimiter = util.NewRateLimiter(1, time.Minute)
	t.Cleanup(func() { routingVariantRateLimiter = previousLimiter })
	body := routingHandlerRequest("segment", true, 2, "route-a")
	if status, responseBody := callRoutingHandler(t, app, auth, PluginSystemRoutingRoute, body); status != http.StatusOK {
		t.Fatalf("first variant request = %d %s", status, responseBody)
	}
	if status, responseBody := callRoutingHandler(t, app, auth, PluginSystemRoutingRoute, body); status != http.StatusTooManyRequests {
		t.Fatalf("second variant request = %d %s", status, responseBody)
	}
}

func TestRoutingHandlerReusesPreparedProfileUntilEffectiveInputChanges(t *testing.T) {
	app, auth := newRoutingHandlerTestApp(t, map[string]bool{"route-a": true}, "segment")
	addRoutingTestCapability(t, app, "route-a", pluginsystem.CapabilityManifest{
		Name: "profile_prepare", Version: "v1", Export: "profile_prepare_v1",
	})

	previousCache := routingPreparedProfiles
	routingPreparedProfiles = newRoutingPreparedProfileCache(time.Minute, 32)
	t.Cleanup(func() { routingPreparedProfiles = previousCache })
	var prepareCalls atomic.Int32
	withRoutingPrepareCaller(t, func(_ context.Context, _ pluginsystem.LocalPlugin, _ pluginsystem.CapabilityManifest, _ *core.Record, _ map[string]any, _ map[string]any, request pluginRoutingProfilePrepareRequest) (pluginRoutingProfilePrepareOutput, error) {
		prepareCalls.Add(1)
		return pluginRoutingProfilePrepareOutput{PreparedKey: "prepared-" + request.Mode}, nil
	})
	withRoutingCaller(t, func(_ context.Context, plugin pluginsystem.LocalPlugin, _ pluginsystem.CapabilityManifest, _ *core.Record, _ map[string]any, _ map[string]any, request pluginRoutingRouteRequest) (pluginRoutingRouteOutput, error) {
		if request.Profile.PreparedKey == "" {
			return pluginRoutingRouteOutput{}, fmt.Errorf("route call did not receive a prepared profile key")
		}
		return pluginRoutingRouteOutput{Candidates: []pluginRoutingCandidate{routingHandlerCandidate(request, plugin.Manifest.ID)}}, nil
	})

	body := routingHandlerRequest("segment", false, 1, "route-a")
	for attempt := 0; attempt < 2; attempt++ {
		if status, responseBody := callRoutingHandler(t, app, auth, PluginSystemRoutingRoute, body); status != http.StatusOK {
			t.Fatalf("prepared route %d = %d %s", attempt+1, status, responseBody)
		}
	}
	if prepareCalls.Load() != 1 {
		t.Fatalf("profile preparation calls = %d, want 1", prepareCalls.Load())
	}

	var changed map[string]any
	if err := json.Unmarshal([]byte(body), &changed); err != nil {
		t.Fatalf("decode routing request: %v", err)
	}
	changed["preferences"] = map[string]any{"hillPreference": 0.75}
	changedBody, err := json.Marshal(changed)
	if err != nil {
		t.Fatalf("encode changed routing request: %v", err)
	}
	if status, responseBody := callRoutingHandler(t, app, auth, PluginSystemRoutingRoute, string(changedBody)); status != http.StatusOK {
		t.Fatalf("changed prepared route = %d %s", status, responseBody)
	}
	if prepareCalls.Load() != 2 {
		t.Fatalf("profile preparation calls after preference change = %d, want 2", prepareCalls.Load())
	}
}

func TestRoutingProfilePrepareHandlerWarmsProfileWithoutAnchors(t *testing.T) {
	app, auth := newRoutingHandlerTestApp(t, map[string]bool{"route-a": true}, "segment")
	addRoutingTestCapability(t, app, "route-a", pluginsystem.CapabilityManifest{
		Name: "profile_prepare", Version: "v1", Export: "profile_prepare_v1",
	})
	previousCache := routingPreparedProfiles
	routingPreparedProfiles = newRoutingPreparedProfileCache(time.Minute, 32)
	t.Cleanup(func() { routingPreparedProfiles = previousCache })
	var prepareCalls atomic.Int32
	withRoutingPrepareCaller(t, func(_ context.Context, _ pluginsystem.LocalPlugin, _ pluginsystem.CapabilityManifest, _ *core.Record, _ map[string]any, _ map[string]any, request pluginRoutingProfilePrepareRequest) (pluginRoutingProfilePrepareOutput, error) {
		prepareCalls.Add(1)
		if request.Mode != "foot" || request.Profile.Key != "pedestrian" {
			return pluginRoutingProfilePrepareOutput{}, fmt.Errorf("unexpected preparation request: %#v", request)
		}
		return pluginRoutingProfilePrepareOutput{PreparedKey: "prepared-profile"}, nil
	})
	body := `{
		"pluginId":"route-a",
		"engines":[{"pluginId":"route-a"}],
		"engineMode":"single",
		"mode":"foot",
		"profile":{"key":"pedestrian"}
	}`
	for attempt := 0; attempt < 2; attempt++ {
		status, responseBody := callRoutingHandler(t, app, auth, PluginSystemRoutingProfilePrepare, body)
		if status != http.StatusOK || !strings.Contains(responseBody, `"prepared":1`) {
			t.Fatalf("profile warm-up %d = %d %s", attempt+1, status, responseBody)
		}
	}
	withRoutingCaller(t, func(_ context.Context, plugin pluginsystem.LocalPlugin, _ pluginsystem.CapabilityManifest, _ *core.Record, _ map[string]any, _ map[string]any, request pluginRoutingRouteRequest) (pluginRoutingRouteOutput, error) {
		return pluginRoutingRouteOutput{Candidates: []pluginRoutingCandidate{routingHandlerCandidate(request, plugin.Manifest.ID)}}, nil
	})
	if status, responseBody := callRoutingHandler(t, app, auth, PluginSystemRoutingRoute, routingHandlerRequest("segment", false, 1, "route-a")); status != http.StatusOK {
		t.Fatalf("route after profile warm-up = %d %s", status, responseBody)
	}
	if prepareCalls.Load() != 1 {
		t.Fatalf("profile warm-up and equivalent route used %d preparations, want 1", prepareCalls.Load())
	}
}

func TestRoutingProfilePrepareHandlerNoOpsWithoutCapability(t *testing.T) {
	app, auth := newRoutingHandlerTestApp(t, map[string]bool{"route-a": true}, "segment")
	var prepareCalls atomic.Int32
	withRoutingPrepareCaller(t, func(_ context.Context, _ pluginsystem.LocalPlugin, _ pluginsystem.CapabilityManifest, _ *core.Record, _ map[string]any, _ map[string]any, _ pluginRoutingProfilePrepareRequest) (pluginRoutingProfilePrepareOutput, error) {
		prepareCalls.Add(1)
		return pluginRoutingProfilePrepareOutput{PreparedKey: "unexpected"}, nil
	})
	body := `{
		"pluginId":"route-a",
		"engines":[{"pluginId":"route-a"}],
		"engineMode":"single",
		"mode":"foot",
		"profile":{"key":"pedestrian"}
	}`
	status, responseBody := callRoutingHandler(t, app, auth, PluginSystemRoutingProfilePrepare, body)
	if status != http.StatusOK || !strings.Contains(responseBody, `"prepared":0`) {
		t.Fatalf("profile warm-up without capability = %d %s", status, responseBody)
	}
	withRoutingCaller(t, func(_ context.Context, plugin pluginsystem.LocalPlugin, _ pluginsystem.CapabilityManifest, _ *core.Record, _ map[string]any, _ map[string]any, request pluginRoutingRouteRequest) (pluginRoutingRouteOutput, error) {
		return pluginRoutingRouteOutput{Candidates: []pluginRoutingCandidate{routingHandlerCandidate(request, plugin.Manifest.ID)}}, nil
	})
	if status, responseBody := callRoutingHandler(t, app, auth, PluginSystemRoutingRoute, routingHandlerRequest("segment", false, 1, "route-a")); status != http.StatusOK {
		t.Fatalf("route without profile preparation capability = %d %s", status, responseBody)
	}
	if prepareCalls.Load() != 0 {
		t.Fatalf("profile preparation caller invoked %d times without capability", prepareCalls.Load())
	}
}

func TestRoutingProfilePreparationFailureFallsBackToNormalRoute(t *testing.T) {
	app, auth := newRoutingHandlerTestApp(t, map[string]bool{"route-a": true}, "segment")
	addRoutingTestCapability(t, app, "route-a", pluginsystem.CapabilityManifest{
		Name: "profile_prepare", Version: "v1", Export: "profile_prepare_v1",
	})
	previousCache := routingPreparedProfiles
	routingPreparedProfiles = newRoutingPreparedProfileCache(time.Minute, 32)
	t.Cleanup(func() { routingPreparedProfiles = previousCache })
	withRoutingPrepareCaller(t, func(context.Context, pluginsystem.LocalPlugin, pluginsystem.CapabilityManifest, *core.Record, map[string]any, map[string]any, pluginRoutingProfilePrepareRequest) (pluginRoutingProfilePrepareOutput, error) {
		return pluginRoutingProfilePrepareOutput{}, context.DeadlineExceeded
	})
	withRoutingCaller(t, func(_ context.Context, plugin pluginsystem.LocalPlugin, _ pluginsystem.CapabilityManifest, _ *core.Record, _ map[string]any, _ map[string]any, request pluginRoutingRouteRequest) (pluginRoutingRouteOutput, error) {
		if request.Profile.PreparedKey != "" {
			return pluginRoutingRouteOutput{}, fmt.Errorf("fallback route received prepared key %q", request.Profile.PreparedKey)
		}
		return pluginRoutingRouteOutput{Candidates: []pluginRoutingCandidate{routingHandlerCandidate(request, plugin.Manifest.ID)}}, nil
	})
	status, responseBody := callRoutingHandler(t, app, auth, PluginSystemRoutingRoute, routingHandlerRequest("segment", false, 1, "route-a"))
	if status != http.StatusOK || !strings.Contains(responseBody, "routing_profile_preparation_failed") {
		t.Fatalf("route after preparation failure = %d %s", status, responseBody)
	}
	if strings.Contains(responseBody, `"engineErrors"`) {
		t.Fatalf("successful fallback exposed preparation as engine failure: %s", responseBody)
	}
}

func TestRoutingRouteRefreshesRejectedPreparedProfile(t *testing.T) {
	app, auth := newRoutingHandlerTestApp(t, map[string]bool{"route-a": true}, "segment")
	addRoutingTestCapability(t, app, "route-a", pluginsystem.CapabilityManifest{
		Name: "profile_prepare", Version: "v1", Export: "profile_prepare_v1",
	})
	previousCache := routingPreparedProfiles
	routingPreparedProfiles = newRoutingPreparedProfileCache(time.Minute, 32)
	t.Cleanup(func() { routingPreparedProfiles = previousCache })
	var prepareCalls atomic.Int32
	withRoutingPrepareCaller(t, func(context.Context, pluginsystem.LocalPlugin, pluginsystem.CapabilityManifest, *core.Record, map[string]any, map[string]any, pluginRoutingProfilePrepareRequest) (pluginRoutingProfilePrepareOutput, error) {
		if prepareCalls.Add(1) == 1 {
			return pluginRoutingProfilePrepareOutput{PreparedKey: "stale-profile"}, nil
		}
		return pluginRoutingProfilePrepareOutput{PreparedKey: "fresh-profile"}, nil
	})
	var routeCalls atomic.Int32
	withRoutingCaller(t, func(_ context.Context, plugin pluginsystem.LocalPlugin, _ pluginsystem.CapabilityManifest, _ *core.Record, _ map[string]any, _ map[string]any, request pluginRoutingRouteRequest) (pluginRoutingRouteOutput, error) {
		routeCalls.Add(1)
		switch request.Profile.PreparedKey {
		case "stale-profile":
			return pluginRoutingRouteOutput{Error: &pluginsystem.PluginError{Code: "unsupported_profile", Message: "prepared profile expired"}}, nil
		case "fresh-profile":
			return pluginRoutingRouteOutput{Candidates: []pluginRoutingCandidate{routingHandlerCandidate(request, plugin.Manifest.ID)}}, nil
		default:
			return pluginRoutingRouteOutput{}, fmt.Errorf("route received unexpected prepared key %q", request.Profile.PreparedKey)
		}
	})
	status, responseBody := callRoutingHandler(t, app, auth, PluginSystemRoutingRoute, routingHandlerRequest("segment", false, 1, "route-a"))
	if status != http.StatusOK {
		t.Fatalf("route with refreshed profile = %d %s", status, responseBody)
	}
	if prepareCalls.Load() != 2 || routeCalls.Load() != 2 {
		t.Fatalf("self-healing route used %d preparations and %d route calls", prepareCalls.Load(), routeCalls.Load())
	}
}

func TestRoutingRouteCoordinatesSegmentRetriesAfterPreparedProfileRefresh(t *testing.T) {
	app, auth := newRoutingHandlerTestApp(t, map[string]bool{"route-a": true}, "segment")
	addRoutingTestCapability(t, app, "route-a", pluginsystem.CapabilityManifest{
		Name: "profile_prepare", Version: "v1", Export: "profile_prepare_v1",
	})
	previousCache := routingPreparedProfiles
	routingPreparedProfiles = newRoutingPreparedProfileCache(time.Minute, 32)
	t.Cleanup(func() { routingPreparedProfiles = previousCache })
	var prepareCalls atomic.Int32
	withRoutingPrepareCaller(t, func(context.Context, pluginsystem.LocalPlugin, pluginsystem.CapabilityManifest, *core.Record, map[string]any, map[string]any, pluginRoutingProfilePrepareRequest) (pluginRoutingProfilePrepareOutput, error) {
		if prepareCalls.Add(1) == 1 {
			return pluginRoutingProfilePrepareOutput{PreparedKey: "stale-profile"}, nil
		}
		return pluginRoutingProfilePrepareOutput{PreparedKey: "fresh-profile"}, nil
	})
	var staleCalls atomic.Int32
	var freshCalls atomic.Int32
	var inlineCalls atomic.Int32
	withRoutingCaller(t, func(_ context.Context, plugin pluginsystem.LocalPlugin, _ pluginsystem.CapabilityManifest, _ *core.Record, _ map[string]any, _ map[string]any, request pluginRoutingRouteRequest) (pluginRoutingRouteOutput, error) {
		switch request.Profile.PreparedKey {
		case "stale-profile":
			staleCalls.Add(1)
			return pluginRoutingRouteOutput{Error: &pluginsystem.PluginError{Code: "unsupported_profile", Message: "prepared profile expired"}}, nil
		case "fresh-profile":
			freshCalls.Add(1)
			return pluginRoutingRouteOutput{Candidates: []pluginRoutingCandidate{routingHandlerCandidate(request, plugin.Manifest.ID)}}, nil
		default:
			inlineCalls.Add(1)
			return pluginRoutingRouteOutput{Candidates: []pluginRoutingCandidate{routingHandlerCandidate(request, plugin.Manifest.ID)}}, nil
		}
	})

	body := routingHandlerRequestWithAnchorCount(t, 5, true, 2, "route-a")
	if status, responseBody := callRoutingHandler(t, app, auth, PluginSystemRoutingRoute, body); status != http.StatusOK {
		t.Fatalf("coordinated profile refresh = %d %s", status, responseBody)
	}
	if prepareCalls.Load() != 2 || staleCalls.Load() != 4 || freshCalls.Load() != 4 || inlineCalls.Load() != 0 {
		t.Fatalf("first request used preparations=%d stale=%d fresh=%d inline=%d", prepareCalls.Load(), staleCalls.Load(), freshCalls.Load(), inlineCalls.Load())
	}
	if status, responseBody := callRoutingHandler(t, app, auth, PluginSystemRoutingRoute, body); status != http.StatusOK {
		t.Fatalf("request after cache healing = %d %s", status, responseBody)
	}
	if prepareCalls.Load() != 2 || staleCalls.Load() != 4 || freshCalls.Load() != 8 || inlineCalls.Load() != 0 {
		t.Fatalf("two requests used preparations=%d stale=%d fresh=%d inline=%d", prepareCalls.Load(), staleCalls.Load(), freshCalls.Load(), inlineCalls.Load())
	}
}

func TestRoutingRouteFailedRefreshRetriesAllSegmentsWithInlineProfile(t *testing.T) {
	app, auth := newRoutingHandlerTestApp(t, map[string]bool{"route-a": true}, "segment")
	addRoutingTestCapability(t, app, "route-a", pluginsystem.CapabilityManifest{
		Name: "profile_prepare", Version: "v1", Export: "profile_prepare_v1",
	})
	previousCache := routingPreparedProfiles
	routingPreparedProfiles = newRoutingPreparedProfileCache(time.Minute, 32)
	t.Cleanup(func() { routingPreparedProfiles = previousCache })
	var prepareCalls atomic.Int32
	withRoutingPrepareCaller(t, func(context.Context, pluginsystem.LocalPlugin, pluginsystem.CapabilityManifest, *core.Record, map[string]any, map[string]any, pluginRoutingProfilePrepareRequest) (pluginRoutingProfilePrepareOutput, error) {
		if prepareCalls.Add(1) == 1 {
			return pluginRoutingProfilePrepareOutput{PreparedKey: "stale-profile"}, nil
		}
		return pluginRoutingProfilePrepareOutput{}, context.DeadlineExceeded
	})
	var staleCalls atomic.Int32
	var inlineCalls atomic.Int32
	withRoutingCaller(t, func(_ context.Context, plugin pluginsystem.LocalPlugin, _ pluginsystem.CapabilityManifest, _ *core.Record, _ map[string]any, _ map[string]any, request pluginRoutingRouteRequest) (pluginRoutingRouteOutput, error) {
		if request.Profile.PreparedKey == "stale-profile" {
			staleCalls.Add(1)
			return pluginRoutingRouteOutput{Error: &pluginsystem.PluginError{Code: "unsupported_profile", Message: "prepared profile expired"}}, nil
		}
		inlineCalls.Add(1)
		return pluginRoutingRouteOutput{Candidates: []pluginRoutingCandidate{routingHandlerCandidate(request, plugin.Manifest.ID)}}, nil
	})

	body := routingHandlerRequestWithAnchorCount(t, 5, true, 2, "route-a")
	if status, responseBody := callRoutingHandler(t, app, auth, PluginSystemRoutingRoute, body); status != http.StatusOK {
		t.Fatalf("inline fallback after failed refresh = %d %s", status, responseBody)
	}
	if prepareCalls.Load() != 2 || staleCalls.Load() != 4 || inlineCalls.Load() != 4 {
		t.Fatalf("failed refresh used preparations=%d stale=%d inline=%d", prepareCalls.Load(), staleCalls.Load(), inlineCalls.Load())
	}
}

func TestRoutingRouteDoesNotRetryInvalidRequestWithPreparedProfile(t *testing.T) {
	app, auth := newRoutingHandlerTestApp(t, map[string]bool{"route-a": true}, "segment")
	addRoutingTestCapability(t, app, "route-a", pluginsystem.CapabilityManifest{
		Name: "profile_prepare", Version: "v1", Export: "profile_prepare_v1",
	})
	previousCache := routingPreparedProfiles
	routingPreparedProfiles = newRoutingPreparedProfileCache(time.Minute, 32)
	t.Cleanup(func() { routingPreparedProfiles = previousCache })
	var prepareCalls atomic.Int32
	withRoutingPrepareCaller(t, func(context.Context, pluginsystem.LocalPlugin, pluginsystem.CapabilityManifest, *core.Record, map[string]any, map[string]any, pluginRoutingProfilePrepareRequest) (pluginRoutingProfilePrepareOutput, error) {
		prepareCalls.Add(1)
		return pluginRoutingProfilePrepareOutput{PreparedKey: "prepared-profile"}, nil
	})
	var routeCalls atomic.Int32
	withRoutingCaller(t, func(context.Context, pluginsystem.LocalPlugin, pluginsystem.CapabilityManifest, *core.Record, map[string]any, map[string]any, pluginRoutingRouteRequest) (pluginRoutingRouteOutput, error) {
		routeCalls.Add(1)
		return pluginRoutingRouteOutput{Error: &pluginsystem.PluginError{Code: "invalid_request", Message: "invalid route option"}}, nil
	})
	if status, _ := callRoutingHandler(t, app, auth, PluginSystemRoutingRoute, routingHandlerRequest("segment", false, 1, "route-a")); status == http.StatusOK {
		t.Fatal("invalid routing request unexpectedly succeeded")
	}
	if prepareCalls.Load() != 1 || routeCalls.Load() != 1 {
		t.Fatalf("invalid request used %d preparations and %d route calls", prepareCalls.Load(), routeCalls.Load())
	}
}

func TestRoutingPreparationFallbackBudgetErrorIncludesPreparationFailure(t *testing.T) {
	app, auth := newRoutingHandlerTestApp(t, map[string]bool{
		"route-a": true, "route-b": true, "route-c": true, "route-d": true,
	}, "segment")
	addRoutingTestCapability(t, app, "route-a", pluginsystem.CapabilityManifest{
		Name: "profile_prepare", Version: "v1", Export: "profile_prepare_v1",
	})
	previousCache := routingPreparedProfiles
	routingPreparedProfiles = newRoutingPreparedProfileCache(time.Minute, 32)
	t.Cleanup(func() { routingPreparedProfiles = previousCache })
	withRoutingPrepareCaller(t, func(context.Context, pluginsystem.LocalPlugin, pluginsystem.CapabilityManifest, *core.Record, map[string]any, map[string]any, pluginRoutingProfilePrepareRequest) (pluginRoutingProfilePrepareOutput, error) {
		return pluginRoutingProfilePrepareOutput{}, context.DeadlineExceeded
	})

	var request map[string]any
	if err := json.Unmarshal([]byte(routingHandlerRequestWithAnchorCount(t, 16, true, 3, "route-a")), &request); err != nil {
		t.Fatalf("decode routing request: %v", err)
	}
	request["engineMode"] = "parallel"
	request["engines"] = []map[string]any{{"pluginId": "route-a"}, {"pluginId": "route-b"}, {"pluginId": "route-c"}, {"pluginId": "route-d"}}
	request["profile"] = map[string]any{"pluginId": "route-a", "key": "custom", "contentBase64": "cHJvZmlsZQ=="}
	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("encode routing request: %v", err)
	}
	status, responseBody := callRoutingHandler(t, app, auth, PluginSystemRoutingRoute, string(payload))
	if status != http.StatusUnprocessableEntity || !strings.Contains(responseBody, "profile_preparation_fanout_limit_exceeded") || !strings.Contains(responseBody, "preparationErrors") || !strings.Contains(responseBody, "provider_timeout") {
		t.Fatalf("preparation fallback budget response = %d %s", status, responseBody)
	}
}

func TestAutomaticRoutingFanoutWarningIsReturnedOnceAfterPreparationReplanning(t *testing.T) {
	app, auth := newRoutingHandlerTestApp(t, map[string]bool{
		"route-a": true, "route-b": true, "route-c": true, "route-d": true,
	}, "segment")
	for _, pluginID := range []string{"route-a", "route-b", "route-c", "route-d"} {
		setRoutingHandlerTestAlternatives(t, app, pluginID, 3)
	}
	addRoutingTestCapability(t, app, "route-a", pluginsystem.CapabilityManifest{
		Name: "profile_prepare", Version: "v1", Export: "profile_prepare_v1",
	})
	previousCache := routingPreparedProfiles
	routingPreparedProfiles = newRoutingPreparedProfileCache(time.Minute, 32)
	t.Cleanup(func() { routingPreparedProfiles = previousCache })
	withRoutingPrepareCaller(t, func(context.Context, pluginsystem.LocalPlugin, pluginsystem.CapabilityManifest, *core.Record, map[string]any, map[string]any, pluginRoutingProfilePrepareRequest) (pluginRoutingProfilePrepareOutput, error) {
		return pluginRoutingProfilePrepareOutput{}, context.DeadlineExceeded
	})
	withRoutingCaller(t, func(_ context.Context, _ pluginsystem.LocalPlugin, _ pluginsystem.CapabilityManifest, _ *core.Record, _ map[string]any, _ map[string]any, request pluginRoutingRouteRequest) (pluginRoutingRouteOutput, error) {
		candidate := testSingleSegmentCandidate(request.Anchors[0], request.Anchors[1], "")
		return pluginRoutingRouteOutput{Candidates: []pluginRoutingCandidate{candidate}}, nil
	})

	var request map[string]any
	if err := json.Unmarshal([]byte(routingHandlerRequestWithAnchorCount(t, 16, true, 3, "route-a")), &request); err != nil {
		t.Fatalf("decode routing request: %v", err)
	}
	request["engineMode"] = "parallel"
	delete(request, "engines")
	request["profile"] = map[string]any{"pluginId": "route-a", "key": "custom", "contentBase64": "cHJvZmlsZQ=="}
	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("encode routing request: %v", err)
	}
	status, responseBody := callRoutingHandler(t, app, auth, PluginSystemRoutingRoute, string(payload))
	if status != http.StatusOK {
		t.Fatalf("automatic fanout response = %d %s", status, responseBody)
	}
	var output pluginRoutingRouteOutput
	if err := json.Unmarshal([]byte(responseBody), &output); err != nil {
		t.Fatalf("decode automatic fanout response: %v", err)
	}
	count := 0
	for _, warning := range output.Warnings {
		if warning == "routing_parallel_engines_reduced_for_fanout" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("fanout reduction warning count = %d, warnings=%v", count, output.Warnings)
	}
}

func TestRoutingProfilePrepareHandlerWarmsAllParallelEngines(t *testing.T) {
	app, auth := newRoutingHandlerTestApp(t, map[string]bool{"route-a": true, "route-b": true}, "segment")
	for _, pluginID := range []string{"route-a", "route-b"} {
		addRoutingTestCapability(t, app, pluginID, pluginsystem.CapabilityManifest{
			Name: "profile_prepare", Version: "v1", Export: "profile_prepare_v1",
		})
	}
	controlsInput := routingEffectiveControlsInput{}
	controlsInput.Routing.Engines = []routingEngineSelection{{PluginID: "route-a"}, {PluginID: "route-b"}}
	controls, err := ResolveRoutingEffectiveControls(app, auth.Id, controlsInput)
	if err != nil {
		t.Fatalf("resolve parallel preparation support: %v", err)
	}
	if !controls.ProfilePreparationSupported["route-a"] || !controls.ProfilePreparationSupported["route-b"] {
		t.Fatalf("effective controls omitted preparation support: %#v", controls.ProfilePreparationSupported)
	}
	previousCache := routingPreparedProfiles
	routingPreparedProfiles = newRoutingPreparedProfileCache(time.Minute, 32)
	t.Cleanup(func() { routingPreparedProfiles = previousCache })
	var prepared sync.Map
	withRoutingPrepareCaller(t, func(_ context.Context, plugin pluginsystem.LocalPlugin, _ pluginsystem.CapabilityManifest, _ *core.Record, _ map[string]any, _ map[string]any, _ pluginRoutingProfilePrepareRequest) (pluginRoutingProfilePrepareOutput, error) {
		prepared.Store(plugin.Manifest.ID, true)
		return pluginRoutingProfilePrepareOutput{PreparedKey: "prepared-" + plugin.Manifest.ID}, nil
	})
	body := `{
		"pluginId":"route-a",
		"engines":[{"pluginId":"route-a"},{"pluginId":"route-b"}],
		"engineMode":"parallel",
		"requestVariants":true,
		"mode":"foot",
		"profile":{"key":"pedestrian"}
	}`
	status, responseBody := callRoutingHandler(t, app, auth, PluginSystemRoutingProfilePrepare, body)
	if status != http.StatusOK || !strings.Contains(responseBody, `"prepared":2`) {
		t.Fatalf("parallel profile warm-up = %d %s", status, responseBody)
	}
	for _, pluginID := range []string{"route-a", "route-b"} {
		if _, ok := prepared.Load(pluginID); !ok {
			t.Fatalf("parallel warm-up omitted %s", pluginID)
		}
	}
}

func TestRoutingProfilePrepareHandlerRejectsOversizedInlineProfile(t *testing.T) {
	app, auth := newRoutingHandlerTestApp(t, map[string]bool{"route-a": true}, "segment")
	body := fmt.Sprintf(`{
		"pluginId":"route-a",
		"mode":"foot",
		"profile":{"key":"custom","contentBase64":%q}
	}`, strings.Repeat("A", routingProfileContentMaxBytes*2))
	status, responseBody := callRoutingHandler(t, app, auth, PluginSystemRoutingProfilePrepare, body)
	if status != http.StatusBadRequest || !strings.Contains(responseBody, "exceeds host limit") {
		t.Fatalf("oversized profile warm-up = %d %s", status, responseBody)
	}
}

func newRoutingHandlerTestApp(t *testing.T, viaSupport map[string]bool, defaultMode string) (*pocketbase.PocketBase, *core.Record) {
	t.Helper()
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap routing handler app: %v", err)
	}
	t.Cleanup(func() { _ = app.ResetBootstrapState() })

	routingSettingsCollection := core.NewBaseCollection("routing_settings")
	routingSettingsCollection.Fields.Add(
		&core.TextField{Name: "scope"},
		&core.TextField{Name: "user"},
		&core.JSONField{Name: "config"},
	)
	installedCollection := core.NewBaseCollection("installed_plugins")
	installedCollection.Fields.Add(
		&core.TextField{Name: "plugin_id"}, &core.TextField{Name: "name"},
		&core.TextField{Name: "type"}, &core.TextField{Name: "version"},
		&core.TextField{Name: "runtime"}, &core.TextField{Name: "path"},
		&core.JSONField{Name: "manifest"}, &core.JSONField{Name: "config"},
		&core.TextField{Name: "status"},
	)
	instanceCollection := core.NewBaseCollection("plugin_instances")
	instanceCollection.Fields.Add(
		&core.TextField{Name: "user"}, &core.TextField{Name: "plugin_id"},
		&core.BoolField{Name: "enabled"}, &core.JSONField{Name: "auth"},
		&core.JSONField{Name: "config"}, &core.TextField{Name: "status"},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	routingMappingsCollection := core.NewBaseCollection("routing_profile_mappings")
	routingMappingsCollection.Fields.Add(
		&core.TextField{Name: "scope"}, &core.TextField{Name: "user"},
		&core.TextField{Name: "category"}, &core.TextField{Name: "subcategory"},
		&core.TextField{Name: "plugin_id"}, &core.TextField{Name: "instance_id"},
		&core.TextField{Name: "native_profile_key"}, &core.TextField{Name: "profile"},
		&core.JSONField{Name: "preferences"}, &core.JSONField{Name: "native_config"},
	)
	for _, collection := range []*core.Collection{routingSettingsCollection, installedCollection, instanceCollection, routingMappingsCollection} {
		if err := app.Save(collection); err != nil {
			t.Fatalf("save %s collection: %v", collection.Name, err)
		}
	}

	auth := core.NewRecord(core.NewBaseCollection("users"))
	auth.Id = "user12345678901"
	builtin := core.NewRecord(routingSettingsCollection)
	builtin.Set("scope", routingScopeBuiltin)
	builtin.Set("config", map[string]any{
		"primaryRoutePluginId": firstRoutingPluginID(viaSupport),
		"defaultVariantCount":  3,
		"defaultRoutingMode":   defaultMode,
		"exposedFeatures": map[string]any{
			"parallelRouting": true, "variants": true, "routeCandidates": false,
		},
	})
	if err := app.Save(builtin); err != nil {
		t.Fatalf("save builtin routing settings: %v", err)
	}

	pluginDir := t.TempDir()
	if err := writeRoutingTestWASM(pluginDir); err != nil {
		t.Fatalf("write test wasm: %v", err)
	}
	for _, pluginID := range sortedRoutingPluginIDs(viaSupport) {
		supportsVia := viaSupport[pluginID]
		manifest := pluginsystem.Manifest{
			ManifestVersion: pluginsystem.ManifestVersion,
			ID:              pluginID, Type: pluginsystem.PluginTypeRouting, Name: pluginID, Version: "1.0.0",
			Runtime:      pluginsystem.RuntimeManifest{Type: pluginsystem.RuntimeWASM, Entrypoint: "plugin.wasm"},
			Capabilities: []pluginsystem.CapabilityManifest{{Name: "route", Version: "v1", Export: "route_v1"}},
			Metadata: map[string]any{"routing": map[string]any{
				"supportsViaRouting": supportsVia, "supportsAlternatives": false, "maxAlternatives": 1,
				"standardPreferences": []any{map[string]any{"key": "hillPreference"}},
				"providerPreferences": []any{"providerKnob"},
			}},
		}
		installed := core.NewRecord(installedCollection)
		installed.Set("plugin_id", pluginID)
		installed.Set("name", pluginID)
		installed.Set("type", pluginsystem.PluginTypeRouting)
		installed.Set("version", "1.0.0")
		installed.Set("runtime", pluginsystem.RuntimeWASM)
		installed.Set("path", pluginDir)
		installed.Set("manifest", manifest)
		installed.Set("config", map[string]any{})
		installed.Set("status", "available")
		if err := app.Save(installed); err != nil {
			t.Fatalf("save installed plugin %s: %v", pluginID, err)
		}
		instance := core.NewRecord(instanceCollection)
		instance.Set("user", auth.Id)
		instance.Set("plugin_id", pluginID)
		instance.Set("enabled", true)
		instance.Set("auth", map[string]any{})
		instance.Set("config", map[string]any{})
		instance.Set("status", "configured")
		if err := app.Save(instance); err != nil {
			t.Fatalf("save plugin instance %s: %v", pluginID, err)
		}
	}
	return app, auth
}

func setRoutingHandlerTestAlternatives(t *testing.T, app core.App, pluginID string, maximum int) {
	t.Helper()
	records, err := app.FindRecordsByFilter("installed_plugins", "", "", -1, 0)
	if err != nil {
		t.Fatalf("find installed routing plugin: %v", err)
	}
	for _, record := range records {
		if record.GetString("plugin_id") != pluginID {
			continue
		}
		var manifest pluginsystem.Manifest
		if err := record.UnmarshalJSONField("manifest", &manifest); err != nil {
			t.Fatalf("decode installed routing manifest: %v", err)
		}
		manifest.Metadata["routing"] = map[string]any{
			"supportsViaRouting":   true,
			"supportsAlternatives": true,
			"maxAlternatives":      float64(maximum),
		}
		record.Set("manifest", manifest)
		if err := app.Save(record); err != nil {
			t.Fatalf("save installed routing manifest: %v", err)
		}
		return
	}
	t.Fatalf("installed routing plugin %q not found", pluginID)
}

func callRoutingHandler(t *testing.T, app core.App, auth *core.Record, handler func(*core.RequestEvent) error, body string) (int, string) {
	t.Helper()
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("create API router: %v", err)
	}
	router.POST("/routing-test", func(e *core.RequestEvent) error {
		e.Auth = auth
		return handler(e)
	})
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatalf("build API router: %v", err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/routing-test", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(recorder, request)
	return recorder.Code, recorder.Body.String()
}

func routingHandlerRequest(routingMode string, variants bool, desired int, pluginID string) string {
	request := map[string]any{
		"pluginId": pluginID, "engines": []map[string]any{{"pluginId": pluginID}},
		"engineMode": "single", "desiredVariants": desired, "requestVariants": variants,
		"anchors": []map[string]float64{{"lat": 47.0, "lon": 8.0}, {"lat": 47.02, "lon": 8.0}, {"lat": 47.04, "lon": 8.0}},
		"mode":    "foot", "profile": map[string]any{"key": "pedestrian"},
	}
	if routingMode != "" {
		request["routingMode"] = routingMode
	}
	payload, _ := json.Marshal(request)
	return string(payload)
}

func routingHandlerRequestWithAnchorCount(t *testing.T, anchorCount int, variants bool, desired int, pluginID string) string {
	t.Helper()
	var request map[string]any
	if err := json.Unmarshal([]byte(routingHandlerRequest("segment", variants, desired, pluginID)), &request); err != nil {
		t.Fatalf("decode routing request: %v", err)
	}
	anchors := make([]map[string]float64, anchorCount)
	for index := range anchors {
		anchors[index] = map[string]float64{"lat": 47 + float64(index)*0.02, "lon": 8}
	}
	request["anchors"] = anchors
	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("encode routing request: %v", err)
	}
	return string(payload)
}

func routingHandlerCandidate(request pluginRoutingRouteRequest, pluginID string) pluginRoutingCandidate {
	offset := 0.0
	if pluginID == "route-b" {
		offset = 0.012
	}
	segments := make([]pluginRoutingSegment, 0, len(request.Anchors)-1)
	allPoints := make([][]float64, 0, len(request.Anchors)*2)
	for index := 0; index+1 < len(request.Anchors); index++ {
		start, end := request.Anchors[index], request.Anchors[index+1]
		points := [][]float64{
			{start.Lat, start.Lon},
			{(start.Lat + end.Lat) / 2, (start.Lon+end.Lon)/2 + offset},
			{end.Lat, end.Lon},
		}
		geometry := testGeometry(points)
		segments = append(segments, pluginRoutingSegment{
			FromAnchor: index, ToAnchor: index + 1, Geometry: geometry,
			Distance: 2000 + offset*1000, Duration: 1000 + offset*1000,
		})
		if index > 0 {
			points = points[1:]
		}
		allPoints = append(allPoints, points...)
	}
	geometry := testGeometry(allPoints)
	return pluginRoutingCandidate{
		ID: "provider-id", ProfileKey: request.Profile.Key, Geometry: &geometry,
		Summary:  pluginRoutingSummary{Distance: float64(len(segments)) * (2000 + offset*1000), Duration: float64(len(segments)) * (1000 + offset*1000)},
		Segments: segments, SnappedAnchors: append([]pluginRoutingAnchor(nil), request.Anchors...),
	}
}

func withRoutingCaller(t *testing.T, caller func(context.Context, pluginsystem.LocalPlugin, pluginsystem.CapabilityManifest, *core.Record, map[string]any, map[string]any, pluginRoutingRouteRequest) (pluginRoutingRouteOutput, error)) {
	t.Helper()
	previous := routingRoutePluginCaller
	routingRoutePluginCaller = caller
	t.Cleanup(func() { routingRoutePluginCaller = previous })
}

func withRoutingPrepareCaller(t *testing.T, caller func(context.Context, pluginsystem.LocalPlugin, pluginsystem.CapabilityManifest, *core.Record, map[string]any, map[string]any, pluginRoutingProfilePrepareRequest) (pluginRoutingProfilePrepareOutput, error)) {
	t.Helper()
	previous := routingProfilePreparePluginCaller
	routingProfilePreparePluginCaller = caller
	t.Cleanup(func() { routingProfilePreparePluginCaller = previous })
}

func addRoutingTestCapability(t *testing.T, app core.App, pluginID string, capability pluginsystem.CapabilityManifest) {
	t.Helper()
	installed, err := app.FindFirstRecordByFilter("installed_plugins", "plugin_id='"+pluginID+"'", nil)
	if err != nil {
		t.Fatalf("find installed routing plugin: %v", err)
	}
	manifest := pluginsystem.Manifest{}
	if err := installed.UnmarshalJSONField("manifest", &manifest); err != nil {
		t.Fatalf("decode installed routing manifest: %v", err)
	}
	manifest.Capabilities = append(manifest.Capabilities, capability)
	if capability.Name == "round_trip" {
		routingMetadata := mapValue(manifest.Metadata["routing"])
		routingMetadata["supportsRoundTrip"] = true
		manifest.Metadata["routing"] = routingMetadata
	}
	installed.Set("manifest", manifest)
	if err := app.Save(installed); err != nil {
		t.Fatalf("save installed routing manifest: %v", err)
	}
}

func firstRoutingPluginID(values map[string]bool) string {
	pluginIDs := sortedRoutingPluginIDs(values)
	if len(pluginIDs) > 0 {
		return pluginIDs[0]
	}
	return ""
}

func sortedRoutingPluginIDs(values map[string]bool) []string {
	pluginIDs := make([]string, 0, len(values))
	for pluginID := range values {
		pluginIDs = append(pluginIDs, pluginID)
	}
	sort.Strings(pluginIDs)
	return pluginIDs
}

func writeRoutingTestWASM(dir string) error {
	return os.WriteFile(filepath.Join(dir, "plugin.wasm"), []byte{0}, 0o600)
}
