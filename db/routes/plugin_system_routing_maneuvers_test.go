package routes

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"pocketbase/pluginsystem"
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/core"
)

func TestPreparedManeuverCallRefreshesRejectedProfile(t *testing.T) {
	previousCache := routingPreparedProfiles
	previousPrepareCaller := routingProfilePreparePluginCaller
	previousManeuverCaller := routingManeuverPluginCaller
	routingPreparedProfiles = newRoutingPreparedProfileCache(time.Minute, 8)
	t.Cleanup(func() {
		routingPreparedProfiles = previousCache
		routingProfilePreparePluginCaller = previousPrepareCaller
		routingManeuverPluginCaller = previousManeuverCaller
	})

	runtime := routingEngineRuntime{
		Plugin: pluginsystem.LocalPlugin{Manifest: pluginsystem.Manifest{
			ID: "maneuver-test", Version: "1",
			Capabilities: []pluginsystem.CapabilityManifest{{Name: "profile_prepare", Version: "v1", Export: "profile_prepare_v1"}},
		}},
		Capability: pluginsystem.CapabilityManifest{Name: "maneuvers", Version: "v1", Export: "maneuvers_v1"},
		Request: pluginRoutingRouteRequest{
			Mode: "foot", Profile: pluginRoutingProfile{Key: "pedestrian", PreparedKey: "stale-profile"},
		},
	}
	fingerprint, err := routingPreparedProfileFingerprint(runtime)
	if err != nil {
		t.Fatalf("prepared profile fingerprint: %v", err)
	}
	runtime.PreparedProfileFingerprint = fingerprint
	routingPreparedProfiles.entries[fingerprint] = routingPreparedProfileEntry{
		PreparedKey: "stale-profile", ExpiresAt: time.Now().Add(time.Minute),
	}

	type observedPreparation struct {
		capability string
		request    pluginRoutingProfilePrepareRequest
	}
	prepareCalls := make(chan observedPreparation, 2)
	routingProfilePreparePluginCaller = func(_ context.Context, _ pluginsystem.LocalPlugin, capability pluginsystem.CapabilityManifest, _ *core.Record, _ map[string]any, _ map[string]any, request pluginRoutingProfilePrepareRequest) (pluginRoutingProfilePrepareOutput, error) {
		prepareCalls <- observedPreparation{capability: capability.Name, request: request}
		return pluginRoutingProfilePrepareOutput{PreparedKey: "fresh-profile"}, nil
	}
	calledKeys := []string{}
	routingManeuverPluginCaller = func(_ context.Context, _ pluginsystem.LocalPlugin, _ pluginsystem.CapabilityManifest, _ *core.Record, _ map[string]any, _ map[string]any, request pluginRoutingManeuverRequest) (pluginRoutingManeuverOutput, error) {
		calledKeys = append(calledKeys, request.Profile.PreparedKey)
		if request.Profile.PreparedKey == "stale-profile" {
			return pluginRoutingManeuverOutput{Error: &pluginsystem.PluginError{Code: "unsupported_profile", Message: "expired"}}, nil
		}
		return pluginRoutingManeuverOutput{}, nil
	}

	request := pluginRoutingManeuverRequest{Profile: runtime.Request.Profile}
	output, err := callPreparedRoutingManeuverPlugin(context.Background(), runtime, request)
	if err != nil || output.Error != nil {
		t.Fatalf("maneuver call after refresh = %#v, %v", output, err)
	}
	if len(prepareCalls) != 1 || len(calledKeys) != 2 || calledKeys[0] != "stale-profile" || calledKeys[1] != "fresh-profile" {
		t.Fatalf("maneuver refresh used preparations=%d keys=%v", len(prepareCalls), calledKeys)
	}
	preparation := <-prepareCalls
	if preparation.capability != "profile_prepare" || preparation.request.Profile.PreparedKey != "" {
		t.Fatalf("profile refresh request = %s %#v", preparation.capability, preparation.request)
	}
}

func TestRoutingManeuverHandlerUsesIndependentNavigationFeatureAndPluginContract(t *testing.T) {
	app, auth := newRoutingHandlerTestApp(t, map[string]bool{"navigation-a": true}, "segment")
	installed, err := app.FindFirstRecordByData("installed_plugins", "plugin_id", "navigation-a")
	if err != nil {
		t.Fatalf("find maneuver plugin: %v", err)
	}
	var manifest pluginsystem.Manifest
	if err := installed.UnmarshalJSONField("manifest", &manifest); err != nil {
		t.Fatalf("decode maneuver plugin manifest: %v", err)
	}
	manifest.Capabilities = append(manifest.Capabilities, pluginsystem.CapabilityManifest{Name: "maneuvers", Version: "v1", Export: "maneuvers_v1"})
	manifest.Capabilities = append(manifest.Capabilities, pluginsystem.CapabilityManifest{Name: "profile_prepare", Version: "v1", Export: "profile_prepare_v1"})
	manifest.Metadata["routing"] = map[string]any{
		"roles":               []string{"route", "maneuvers"},
		"nativeProfiles":      []any{map[string]any{"key": "pedestrian", "mode": "foot"}},
		"categoryMappings":    []any{map[string]any{"category": "Hiking", "profile": "pedestrian"}},
		"standardPreferences": []any{map[string]any{"key": "hillPreference", "modes": []any{"foot"}}},
		"providerPreferences": []any{"providerKnob"},
	}
	installed.Set("manifest", manifest)
	if err := app.Save(installed); err != nil {
		t.Fatalf("save maneuver plugin manifest: %v", err)
	}
	if err := saveRoutingSettingsConfig(app, routingScopeBuiltin, "", map[string]any{
		"primaryRoutePluginId": "navigation-a",
		"maneuverPluginId":     "navigation-a",
		"exposedFeatures": map[string]any{
			"routing": false, "navigation": true,
		},
		"defaultPreferences": map[string]any{
			"hillPreference": 0.7, "speedPreference": 0.5, "providerKnob": true,
		},
	}); err != nil {
		t.Fatalf("save maneuver settings: %v", err)
	}
	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(
		&core.BoolField{Name: "public"},
		&core.TextField{Name: "author"},
		&core.TextField{Name: "category"},
		&core.TextField{Name: "subcategory"},
		&core.TextField{Name: "gpx"},
	)
	if err := app.Save(trails); err != nil {
		t.Fatalf("save trails collection: %v", err)
	}
	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(&core.TextField{Name: "user"})
	linkShares := core.NewBaseCollection("trail_link_share")
	linkShares.Fields.Add(&core.TextField{Name: "trail"}, &core.TextField{Name: "token"})
	for _, collection := range []*core.Collection{actors, linkShares} {
		if err := app.Save(collection); err != nil {
			t.Fatalf("save %s collection: %v", collection.Name, err)
		}
	}
	actor := core.NewRecord(actors)
	actor.Set("user", auth.Id)
	if err := app.Save(actor); err != nil {
		t.Fatalf("save authenticated actor: %v", err)
	}
	trail := core.NewRecord(trails)
	trail.Set("public", false)
	trail.Set("author", "another-actor")
	trail.Set("category", "Hiking")
	trail.Set("gpx", "track.gpx")
	if err := app.Save(trail); err != nil {
		t.Fatalf("save trail: %v", err)
	}
	linkShare := core.NewRecord(linkShares)
	linkShare.Set("trail", trail.Id)
	linkShare.Set("token", "shared-trail-token")
	if err := app.Save(linkShare); err != nil {
		t.Fatalf("save trail link share: %v", err)
	}

	previousReader := routingManeuverGPXReader
	routingManeuverGPXReader = func(core.App, *core.Record) ([]byte, error) {
		return []byte(`<gpx><trk><trkseg><trkpt lat="47" lon="8"/><trkpt lat="47.1" lon="8.1"/></trkseg></trk></gpx>`), nil
	}
	t.Cleanup(func() { routingManeuverGPXReader = previousReader })
	previousLimiter := routingManeuverRateLimiter
	routingManeuverRateLimiter = util.NewRateLimiter(2, time.Minute)
	t.Cleanup(func() { routingManeuverRateLimiter = previousLimiter })
	previousCaller := routingManeuverPluginCaller
	previousPrepareCaller := routingProfilePreparePluginCaller
	previousPreparedProfiles := routingPreparedProfiles
	routingPreparedProfiles = newRoutingPreparedProfileCache(time.Minute, 16)
	routingProfilePreparePluginCaller = func(_ context.Context, _ pluginsystem.LocalPlugin, capability pluginsystem.CapabilityManifest, _ *core.Record, _ map[string]any, _ map[string]any, request pluginRoutingProfilePrepareRequest) (pluginRoutingProfilePrepareOutput, error) {
		if capability.Name != "profile_prepare" || request.Profile.Key != "pedestrian" {
			t.Fatalf("profile preparation request = %s %#v", capability.Name, request)
		}
		return pluginRoutingProfilePrepareOutput{PreparedKey: "prepared-pedestrian"}, nil
	}
	t.Cleanup(func() {
		routingProfilePreparePluginCaller = previousPrepareCaller
		routingPreparedProfiles = previousPreparedProfiles
	})
	routingManeuverPluginCaller = func(_ context.Context, plugin pluginsystem.LocalPlugin, capability pluginsystem.CapabilityManifest, _ *core.Record, _ map[string]any, _ map[string]any, request pluginRoutingManeuverRequest) (pluginRoutingManeuverOutput, error) {
		if plugin.Manifest.ID != "navigation-a" || capability.Name != "maneuvers" {
			t.Fatalf("selected runtime = %s/%s", plugin.Manifest.ID, capability.Name)
		}
		if len(request.TrackParts) != 1 || len(request.TrackParts[0].Points) != 2 || request.Profile.Key != "pedestrian" || request.Profile.PreparedKey != "prepared-pedestrian" || request.Mode != "foot" {
			t.Fatalf("resolved maneuver request = %#v", request)
		}
		if request.Language != "de" || request.Limits.MaxManeuvers != 1000 {
			t.Fatalf("maneuver language/limits = %#v", request)
		}
		if request.Preferences["hillPreference"] != 0.7 || request.Preferences["providerKnob"] != true {
			t.Fatalf("supported/provider maneuver preferences were removed: %#v", request.Preferences)
		}
		if _, exists := request.Preferences["speedPreference"]; exists {
			t.Fatalf("undeclared maneuver preference reached plugin: %#v", request.Preferences)
		}
		output := validManeuverOutput()
		output.Maneuvers[0].ProviderInstruction = "Start on Example"
		return output, nil
	}
	t.Cleanup(func() { routingManeuverPluginCaller = previousCaller })

	if err := saveRoutingSettingsConfig(app, routingScopeUser, auth.Id, map[string]any{
		"exposedFeatures": map[string]any{"navigation": false},
	}); err != nil {
		t.Fatalf("save stale user navigation setting: %v", err)
	}
	if err := saveRoutingSettingsConfig(app, routingScopeAdmin, "", map[string]any{
		"exposedFeatures": map[string]any{"navigation": false},
	}); err != nil {
		t.Fatalf("disable navigation as admin: %v", err)
	}
	status, body := callRoutingHandler(t, app, auth, PluginSystemRoutingManeuvers, `{"trailId":"`+trail.Id+`","language":"de"}`)
	if status != 403 || !strings.Contains(body, `"code":"navigation_disabled"`) {
		t.Fatalf("disabled navigation response = %d %s", status, body)
	}
	if err := saveRoutingSettingsConfig(app, routingScopeAdmin, "", map[string]any{
		"exposedFeatures": map[string]any{"navigation": true},
	}); err != nil {
		t.Fatalf("enable navigation as admin: %v", err)
	}

	status, body = callRoutingHandler(t, app, auth, PluginSystemRoutingManeuvers, `{"trailId":"`+trail.Id+`","language":"de"}`)
	if status != 403 {
		t.Fatalf("private maneuver response without share = %d %s", status, body)
	}
	status, body = callRoutingHandler(t, app, auth, PluginSystemRoutingManeuvers, `{"trailId":"`+trail.Id+`","language":"de","share":"shared-trail-token"}`)
	if status != 200 || !strings.Contains(body, `"providerInstruction":"Start on Example"`) || !strings.Contains(body, `"maneuvers"`) {
		t.Fatalf("maneuver handler response = %d %s", status, body)
	}
	status, body = callRoutingHandler(t, app, auth, PluginSystemRoutingManeuvers, `{"trailId":"`+trail.Id+`","language":"de","share":"shared-trail-token"}`)
	if status != 429 {
		t.Fatalf("rate-limited maneuver response = %d %s", status, body)
	}
}

func TestRoutingManeuverDiscoveryUsesExecutableCapabilityAsRoleAuthority(t *testing.T) {
	manifest := pluginsystem.Manifest{Metadata: map[string]any{"routing": map[string]any{"roles": []string{"maneuvers"}}}}
	if routingManifestSupportsManeuvers(manifest) {
		t.Fatal("metadata-only maneuver declaration was eligible")
	}
	manifest.Metadata = map[string]any{"routing": map[string]any{"roles": []string{"route"}}}
	manifest.Capabilities = []pluginsystem.CapabilityManifest{{Name: "maneuvers", Version: "v1", Export: "maneuvers_v1"}}
	if !routingManifestSupportsManeuvers(manifest) {
		t.Fatal("executable maneuver capability was not eligible")
	}
	manifest.Metadata = map[string]any{"routing": map[string]any{"roles": []string{"route", "maneuvers"}}}
	if !routingManifestSupportsManeuvers(manifest) {
		t.Fatal("matching maneuver metadata and capability were not eligible")
	}
}

func TestRoutingManeuverRuntimeFallsBackFromUnavailablePreference(t *testing.T) {
	app, auth := newRoutingHandlerTestApp(t, map[string]bool{
		"preferred-route-only": true,
		"fallback-maneuvers":   true,
	}, "segment")
	installed, err := app.FindFirstRecordByData("installed_plugins", "plugin_id", "fallback-maneuvers")
	if err != nil {
		t.Fatalf("find fallback maneuver plugin: %v", err)
	}
	var manifest pluginsystem.Manifest
	if err := installed.UnmarshalJSONField("manifest", &manifest); err != nil {
		t.Fatalf("decode fallback maneuver manifest: %v", err)
	}
	manifest.Capabilities = append(manifest.Capabilities, pluginsystem.CapabilityManifest{
		Name: "maneuvers", Version: "v1", Export: "maneuvers_v1",
	})
	installed.Set("manifest", manifest)
	if err := app.Save(installed); err != nil {
		t.Fatalf("save fallback maneuver manifest: %v", err)
	}

	status, body := callRoutingHandler(t, app, auth, func(e *core.RequestEvent) error {
		runtime, runtimeErr := maneuverRuntime(e, "preferred-route-only")
		if runtimeErr != nil {
			return runtimeErr
		}
		return e.JSON(200, map[string]string{"pluginId": runtime.Plugin.Manifest.ID})
	}, `{}`)
	if status != 200 || !strings.Contains(body, `"pluginId":"fallback-maneuvers"`) {
		t.Fatalf("maneuver fallback response = %d %s", status, body)
	}
}

func TestRoutingManeuverInvocationFailuresHaveStableErrors(t *testing.T) {
	if got := routingErrorFromCall(context.DeadlineExceeded); got.Code != "provider_timeout" || got.HTTPStatus != 504 {
		t.Fatalf("deadline error = %#v", got)
	}
	if got := routingErrorFromCall(pluginsystem.HostRequestBudgetError{Limit: routingMaxManeuverHostRequests}); got.Code != "provider_request_limit_exceeded" || got.HTTPStatus != 502 {
		t.Fatalf("request-budget error = %#v", got)
	}
	if got := routingErrorFromCode("maneuver_engine_unavailable", "unavailable"); got.HTTPStatus != 503 {
		t.Fatalf("unavailable maneuver engine error = %#v", got)
	}
}

func TestRoutingManeuverDefaultProfileUsesManifestDefault(t *testing.T) {
	manifest := pluginsystem.Manifest{
		ID: "navigation-a",
		HostConfig: map[string]any{"routing": map[string]any{
			"defaultProfiles": []any{"bicycle", "pedestrian"},
		}},
		Metadata: map[string]any{"routing": map[string]any{
			"nativeProfiles": []any{
				map[string]any{"key": "pedestrian", "mode": "foot"},
				map[string]any{"key": "bicycle", "mode": "bike"},
			},
		}},
	}
	request := pluginRoutingRouteRequest{Profile: pluginRoutingProfile{PluginID: manifest.ID}}
	if err := applyRoutingManifestDefaultProfile(manifest, &request); err != nil {
		t.Fatalf("apply default maneuver profile: %v", err)
	}
	if request.Profile.Key != "bicycle" || request.Mode != "bike" {
		t.Fatalf("default maneuver profile = %#v", request)
	}
}

func TestRoutingTrailCategoryNamesDoNotLeakBrokenRelationIDs(t *testing.T) {
	app, _ := newRoutingHandlerTestApp(t, map[string]bool{"navigation-a": true}, "segment")
	trails := core.NewBaseCollection("category_test_trails")
	trails.Fields.Add(
		&core.TextField{Name: "category"},
		&core.TextField{Name: "subcategory"},
		&core.TextField{Name: "federated_category_name"},
		&core.TextField{Name: "federated_subcategory_name"},
	)
	trail := core.NewRecord(trails)
	trail.Set("category", "broken-category-id")
	trail.Set("subcategory", "broken-subcategory-id")
	trail.Set("federated_category_name", "Federated Hiking")
	trail.Set("federated_subcategory_name", "Federated Alpine")
	category, subcategory := routingTrailCategoryNames(app, trail)
	if category != "Federated Hiking" || subcategory != "Federated Alpine" {
		t.Fatalf("resolved category fallback = %q/%q", category, subcategory)
	}
}

func TestNormalizedManeuverTrackPartsPrefersAllTrackSegments(t *testing.T) {
	gpx := `<gpx><rte><rtept lat="1" lon="2"/><rtept lat="3" lon="4"/></rte>` +
		`<trk><trkseg><trkpt lat="47" lon="8"/><trkpt lat="bad" lon="8"/></trkseg>` +
		`<trkseg><trkpt lat="47.1" lon="8.1"/></trkseg></trk></gpx>`
	parts, err := normalizedManeuverTrackParts([]byte(gpx))
	if err != nil {
		t.Fatalf("normalize GPX: %v", err)
	}
	if len(parts) != 2 || len(parts[0].Points) != 1 || len(parts[1].Points) != 1 {
		t.Fatalf("normalized track parts = %#v", parts)
	}
	if parts[0].Points[0].Lat != 47 || parts[1].Points[0].Lat != 47.1 {
		t.Fatalf("track order was not preserved: %#v", parts)
	}
}

func TestNormalizedManeuverTrackPartsUsesOnlyFirstNonEmptyRoute(t *testing.T) {
	tests := []struct {
		name    string
		gpx     string
		wantErr string
		wantLat float64
	}{
		{
			name:    "first empty route is skipped",
			gpx:     `<gpx><rte></rte><rte><rtept lat="47" lon="8"/><rtept lat="48" lon="9"/></rte></gpx>`,
			wantLat: 47,
		},
		{
			name:    "later usable route does not replace unusable first route",
			gpx:     `<gpx><rte><rtept lat="47" lon="8"/></rte><rte><rtept lat="48" lon="9"/><rtept lat="49" lon="10"/></rte></gpx>`,
			wantErr: "track_shape_unavailable",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parts, err := normalizedManeuverTrackParts([]byte(test.gpx))
			if test.wantErr != "" {
				var routingErr *routingError
				if err == nil || !errors.As(err, &routingErr) || routingErr.Code != test.wantErr {
					t.Fatalf("error = %#v, want %s", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalize route fallback: %v", err)
			}
			if len(parts) != 1 || len(parts[0].Points) != 2 || parts[0].Points[0].Lat != test.wantLat {
				t.Fatalf("route fallback = %#v", parts)
			}
		})
	}
}

func TestNormalizedManeuverTrackPartsEnforcesPointAdmissionLimit(t *testing.T) {
	for _, test := range []struct {
		name      string
		count     int
		wantError bool
	}{
		{name: "at limit", count: routingMaxManeuverInputPoints},
		{name: "over limit", count: routingMaxManeuverInputPoints + 1, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var gpx strings.Builder
			gpx.WriteString(`<gpx><trk><trkseg>`)
			for index := 0; index < test.count; index++ {
				gpx.WriteString(`<trkpt lat="47" lon="8"/>`)
			}
			gpx.WriteString(`</trkseg></trk></gpx>`)
			parts, err := normalizedManeuverTrackParts([]byte(gpx.String()))
			if !test.wantError {
				if err != nil || len(parts) != 1 || len(parts[0].Points) != routingMaxManeuverInputPoints {
					t.Fatalf("point limit result parts=%d err=%v", len(parts), err)
				}
				return
			}
			var routingErr *routingError
			if err == nil || !errors.As(err, &routingErr) || routingErr.Code != "track_limit_exceeded" {
				t.Fatalf("point limit error = %#v", err)
			}
		})
	}
}

func TestNormalizedManeuverTrackPartsHasNoManeuverSpecificByteLimit(t *testing.T) {
	gpx := `<gpx><!--` + strings.Repeat("x", routingMaxResponseBodyBytes+1) + `--><trk><trkseg>` +
		`<trkpt lat="47" lon="8"/><trkpt lat="48" lon="9"/>` +
		`</trkseg></trk></gpx>`
	parts, err := normalizedManeuverTrackParts([]byte(gpx))
	if err != nil || len(parts) != 1 || len(parts[0].Points) != 2 {
		t.Fatalf("large persisted GPX was rejected: parts=%d err=%v", len(parts), err)
	}
}

func TestManeuverPluginRequestContainsOnlyNormalizedHostData(t *testing.T) {
	request := pluginRoutingManeuverRequest{
		TrackParts: []pluginRoutingManeuverTrackPart{{Points: []pluginRoutingManeuverPoint{{Lat: 47, Lon: 8}, {Lat: 48, Lon: 9}}}},
		Language:   "de",
		Limits:     maneuverLimits(),
	}
	payload, err := marshalRoutingManeuverRequest(request)
	if err != nil {
		t.Fatalf("marshal maneuver request: %v", err)
	}
	for _, forbidden := range []string{"trailId", "share", "<gpx", "maxTrackPoints", "anchors"} {
		if strings.Contains(string(payload), forbidden) {
			t.Fatalf("plugin request leaked %q: %s", forbidden, payload)
		}
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode maneuver request: %v", err)
	}
	limits := decoded["limits"].(map[string]any)
	if limits["maxGeometryPoints"] != float64(20000) || limits["maxResponseBytes"] != float64(4194304) {
		t.Fatalf("maneuver limits = %#v", limits)
	}
}

func TestNormalizeRoutingManeuverOutputValidatesCompleteContract(t *testing.T) {
	valid := validManeuverOutput()
	if err := normalizeRoutingManeuverOutput(&valid); err != nil {
		t.Fatalf("valid maneuver output: %v", err)
	}

	tests := []struct {
		name   string
		change func(*pluginRoutingManeuverOutput)
	}{
		{name: "zero maneuvers", change: func(output *pluginRoutingManeuverOutput) { output.Maneuvers = nil }},
		{name: "one maneuver", change: func(output *pluginRoutingManeuverOutput) { output.Maneuvers = output.Maneuvers[:1] }},
		{name: "interval gap", change: func(output *pluginRoutingManeuverOutput) {
			output.Maneuvers[0].EndShapeIndex = 0
			output.Maneuvers[0].DistanceMeters = 0
		}},
		{name: "nondegenerate destination", change: func(output *pluginRoutingManeuverOutput) { output.Maneuvers[1].BeginShapeIndex = 0 }},
		{name: "degenerate distance", change: func(output *pluginRoutingManeuverOutput) { output.Maneuvers[1].DistanceMeters = 1 }},
		{name: "unknown vocabulary", change: func(output *pluginRoutingManeuverOutput) { output.Maneuvers[0].Type = "provider_turn_42" }},
		{name: "unknown without warning", change: func(output *pluginRoutingManeuverOutput) { output.Maneuvers[0].Type = "unknown" }},
		{name: "too many street names", change: func(output *pluginRoutingManeuverOutput) {
			output.Maneuvers[0].StreetNames = []string{"1", "2", "3", "4", "5"}
		}},
		{name: "long provider instruction", change: func(output *pluginRoutingManeuverOutput) {
			output.Maneuvers[0].ProviderInstruction = strings.Repeat("🧭", 501)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := validManeuverOutput()
			test.change(&output)
			var routingErr *routingError
			if err := normalizeRoutingManeuverOutput(&output); err == nil || !errors.As(err, &routingErr) || routingErr.Code != "invalid_plugin_response" {
				t.Fatalf("validation error = %#v", err)
			}
		})
	}
}

func validManeuverOutput() pluginRoutingManeuverOutput {
	return pluginRoutingManeuverOutput{
		Geometry: testGeometry([][]float64{{47, 8}, {47.1, 8.1}}),
		Maneuvers: []pluginRoutingManeuver{
			{Type: "start", DistanceMeters: 100, BeginShapeIndex: 0, EndShapeIndex: 1, StreetNames: []string{"Example"}},
			{Type: "destination", DistanceMeters: 0, BeginShapeIndex: 1, EndShapeIndex: 1},
		},
	}
}
