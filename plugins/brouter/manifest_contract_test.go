package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/open-wanderer/wanderer/plugins/brouter/core"
)

func TestManifestAlternativeLimitMatchesAdapter(t *testing.T) {
	payload, err := os.ReadFile("plugin.json")
	if err != nil {
		t.Fatalf("read plugin manifest: %v", err)
	}
	var manifest struct {
		Metadata struct {
			Routing struct {
				MaxAlternatives int `json:"maxAlternatives"`
			} `json:"routing"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(payload, &manifest); err != nil {
		t.Fatalf("decode plugin manifest: %v", err)
	}
	if manifest.Metadata.Routing.MaxAlternatives != core.MaxRouteCandidates {
		t.Fatalf("manifest maxAlternatives = %d, adapter limit = %d", manifest.Metadata.Routing.MaxAlternatives, core.MaxRouteCandidates)
	}
}

func TestManifestDeclaresProfilePreparationCapability(t *testing.T) {
	payload, err := os.ReadFile("plugin.json")
	if err != nil {
		t.Fatalf("read plugin manifest: %v", err)
	}
	var manifest struct {
		Capabilities []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
			Export  string `json:"export"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(payload, &manifest); err != nil {
		t.Fatalf("decode plugin manifest: %v", err)
	}
	for _, capability := range manifest.Capabilities {
		if capability.Name == "profile_prepare" && capability.Version == "v1" && capability.Export == "profile_prepare_v1" {
			return
		}
	}
	t.Fatal("manifest does not declare profile_prepare.v1")
}

func TestManifestDeclaresRoundTripCapabilityAndDiscoveryFlag(t *testing.T) {
	payload, err := os.ReadFile("plugin.json")
	if err != nil {
		t.Fatalf("read plugin manifest: %v", err)
	}
	var manifest struct {
		Capabilities []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
			Export  string `json:"export"`
		} `json:"capabilities"`
		Metadata struct {
			Routing struct {
				SupportsRoundTrip bool `json:"supportsRoundTrip"`
			} `json:"routing"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(payload, &manifest); err != nil {
		t.Fatalf("decode plugin manifest: %v", err)
	}
	found := false
	for _, capability := range manifest.Capabilities {
		if capability.Name == "round_trip" && capability.Version == "v1" && capability.Export == "round_trip_v1" {
			found = true
		}
	}
	if !found || !manifest.Metadata.Routing.SupportsRoundTrip {
		t.Fatalf("round-trip manifest contract incomplete: capability=%v discovery=%v", found, manifest.Metadata.Routing.SupportsRoundTrip)
	}
}

type manifestRoutingProfile struct {
	Key                  string   `json:"key"`
	Mode                 string   `json:"mode"`
	TemplateKey          string   `json:"templateKey"`
	SupportedPreferences []string `json:"supportedPreferences"`
	NativeControlGroups  []struct {
		Controls []struct {
			Key     string `json:"key"`
			Default any    `json:"default"`
		} `json:"controls"`
	} `json:"nativeControlGroups"`
}

func manifestRoutingProfiles(t *testing.T) []manifestRoutingProfile {
	t.Helper()
	payload, err := os.ReadFile("plugin.json")
	if err != nil {
		t.Fatalf("read plugin manifest: %v", err)
	}
	var manifest struct {
		Metadata struct {
			Routing struct {
				NativeProfiles []manifestRoutingProfile `json:"nativeProfiles"`
			} `json:"routing"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(payload, &manifest); err != nil {
		t.Fatalf("decode plugin manifest: %v", err)
	}
	profiles := manifest.Metadata.Routing.NativeProfiles
	if len(profiles) == 0 {
		t.Fatal("manifest declares no native profiles")
	}
	return profiles
}

// Rendering rewrites annotated assignments in place and fails when a parameter
// is not annotated, so every declared template parameter must exist in its base.
func TestEveryTemplateParameterIsAnnotatedInItsBaseProfile(t *testing.T) {
	for templateKey, template := range brouterTemplateDefinitions() {
		t.Run(templateKey, func(t *testing.T) {
			parameters := map[string]any{}
			for _, parameter := range template.Parameters {
				parameters[parameter.Key] = parameter.Default
			}
			content, err := renderBRouterGeneratedProfile(
				map[string]any{"templateKey": templateKey},
				map[string]any{"parameters": parameters},
			)
			if err != nil {
				t.Fatalf("render %s from %s: %v", templateKey, template.Base, err)
			}
			if content == "" {
				t.Fatalf("render %s produced no content", templateKey)
			}
		})
	}
}

func TestManifestProfilesResolveToATemplateOrANativeKey(t *testing.T) {
	templates := brouterTemplateDefinitions()
	for _, profile := range manifestRoutingProfiles(t) {
		if profile.TemplateKey == "" {
			if brouterTemplateKeyForProfile(profile.Key) != "" {
				t.Fatalf("profile %s has an adapter template but does not declare it", profile.Key)
			}
			continue
		}
		if _, ok := templates[profile.TemplateKey]; !ok {
			t.Fatalf("profile %s declares unknown template %s", profile.Key, profile.TemplateKey)
		}
		if adapter := brouterTemplateKeyForProfile(profile.Key); adapter != profile.TemplateKey {
			t.Fatalf("profile %s maps to template %q in the adapter but declares %q", profile.Key, adapter, profile.TemplateKey)
		}
	}
}

// A profile may only advertise the standard preferences its base can honor:
// hiking-mountain and mtb carry no kinematic model, shortest exposes nothing.
func TestManifestSupportedPreferencesMatchTheAdapterMapping(t *testing.T) {
	byMode := map[string][]string{
		"foot":  {"speedPreference", "hillPreference", "maxHikingDifficulty"},
		"bike":  {"speedPreference", "hillPreference", "roadPreference", "avoidBadSurfaces"},
		"motor": {"speedPreference", "avoidBadSurfaces"},
	}
	for _, profile := range manifestRoutingProfiles(t) {
		t.Run(profile.Key, func(t *testing.T) {
			declared := map[string]bool{}
			for _, key := range profile.SupportedPreferences {
				declared[key] = true
			}
			if profile.TemplateKey == "" {
				// Without a template the profile routes by its provider key and
				// nothing is ever rendered into it.
				if len(declared) > 0 {
					t.Fatalf("profile without a template declares preferences %v", profile.SupportedPreferences)
				}
				return
			}
			for _, preference := range byMode[profile.Mode] {
				nativeConfig := core.NativeConfigWithPreferences(
					profile.TemplateKey, nil, map[string]any{preference: 0.5}, profile.Mode,
				)
				_, mapped := nativeConfig["parameters"]
				if mapped != declared[preference] {
					t.Fatalf("preference %s: declared=%v, adapter maps it=%v", preference, declared[preference], mapped)
				}
			}
		})
	}
}

// A template default that differs from its base file would silently rewrite an
// unmodified profile, so the declared defaults must mirror the upstream files.
func TestTemplateDefaultsMirrorTheBaseProfiles(t *testing.T) {
	for templateKey, template := range brouterTemplateDefinitions() {
		t.Run(templateKey, func(t *testing.T) {
			content, err := brouterProfilePresets.ReadFile("profiles/" + template.Base)
			if err != nil {
				t.Fatalf("read %s: %v", template.Base, err)
			}
			base := map[string]any{}
			for _, parameter := range parseBRouterProfileParameters(string(content)) {
				base[parameter.Key] = parameter.Default
			}
			for _, parameter := range template.Parameters {
				expected, ok := base[parameter.Key]
				if !ok {
					t.Fatalf("%s does not annotate %s", template.Base, parameter.Key)
				}
				if !brouterTemplateValuesEqual(parameter.Default, expected) {
					t.Fatalf("%s default for %s is %#v, %s says %#v",
						templateKey, parameter.Key, parameter.Default, template.Base, expected)
				}
			}
		})
	}
}

// Rendering without any values must hand the base file through untouched.
func TestRenderingWithoutValuesKeepsTheBaseVerbatim(t *testing.T) {
	for templateKey, template := range brouterTemplateDefinitions() {
		t.Run(templateKey, func(t *testing.T) {
			content, err := brouterProfilePresets.ReadFile("profiles/" + template.Base)
			if err != nil {
				t.Fatalf("read %s: %v", template.Base, err)
			}
			rendered, err := renderBRouterGeneratedProfile(
				map[string]any{"templateKey": templateKey},
				map[string]any{},
			)
			if err != nil {
				t.Fatalf("render %s: %v", templateKey, err)
			}
			if rendered != string(content) {
				t.Fatalf("rendering %s without values changed %s", templateKey, template.Base)
			}
		})
	}
}

// A native control shows its default before the user touches it, so a default
// that disagrees with the base file would advertise the wrong starting point.
func TestManifestControlDefaultsMirrorTheBaseProfiles(t *testing.T) {
	templates := brouterTemplateDefinitions()
	for _, profile := range manifestRoutingProfiles(t) {
		template, ok := templates[profile.TemplateKey]
		if !ok {
			continue
		}
		content, err := brouterProfilePresets.ReadFile("profiles/" + template.Base)
		if err != nil {
			t.Fatalf("read %s: %v", template.Base, err)
		}
		base := map[string]any{}
		for _, parameter := range parseBRouterProfileParameters(string(content)) {
			base[parameter.Key] = parameter.Default
		}
		for _, group := range profile.NativeControlGroups {
			for _, control := range group.Controls {
				expected, ok := base[control.Key]
				if !ok {
					t.Fatalf("%s does not annotate %s", template.Base, control.Key)
				}
				if !brouterTemplateValuesEqual(control.Default, expected) {
					t.Fatalf("%s control %s defaults to %#v, %s says %#v",
						profile.Key, control.Key, control.Default, template.Base, expected)
				}
			}
		}
	}
}

// An uploaded profile carries no declared ranges, so values the profile itself
// uses must not be rejected: Poutnik's factors go negative and a car's mass is
// in the thousands.
func TestUploadedProfilesAcceptValuesOutsideAnyInventedRange(t *testing.T) {
	content, err := brouterProfilePresets.ReadFile("profiles/mtb.brf")
	if err != nil {
		t.Fatalf("read mtb.brf: %v", err)
	}
	templateKey := core.TemplateKeyForParameters(brouterProfileParameterTypes(string(content)))
	if templateKey != core.TemplateMTB {
		t.Fatalf("inferred template = %q, want %q", templateKey, core.TemplateMTB)
	}

	nativeConfig := core.NativeConfigWithPreferences(templateKey, map[string]any{
		"parameters": map[string]any{"smallpaved_factor": -1.5},
	}, map[string]any{"avoidBadSurfaces": 0.9}, "bike")
	rendered, err := renderBRouterCustomProfile(string(content), nativeConfig)
	if err != nil {
		t.Fatalf("render uploaded mtb profile: %v", err)
	}
	if !strings.Contains(rendered, "MTB_factor             -1.2") {
		t.Fatal("expected the negative MTB factor to be rendered")
	}
	if !strings.Contains(rendered, "smallpaved_factor      -1.5") {
		t.Fatal("expected the negative small-paved factor to be rendered")
	}

	car, err := brouterProfilePresets.ReadFile("profiles/car-vario.brf")
	if err != nil {
		t.Fatalf("read car-vario.brf: %v", err)
	}
	rendered, err = renderBRouterCustomProfile(string(car), map[string]any{
		"parameters": map[string]any{"totalweight": 2400.0},
	})
	if err != nil {
		t.Fatalf("render uploaded car profile: %v", err)
	}
	if !strings.Contains(rendered, "2400") {
		t.Fatal("expected a mass above the previously invented maximum to be rendered")
	}
}

// The rendered artefact is what the provider sees: the hiking base must keep
// its elevation switch on across the whole preference range, because it also
// governs the cost of steps there.
func TestRenderedHikingProfileKeepsTheStepsCostStable(t *testing.T) {
	for _, hills := range []float64{0, 0.5, 1} {
		rendered, err := renderBRouterGeneratedProfile(
			map[string]any{"templateKey": core.TemplateHike},
			core.NativeConfigWithPreferences(core.TemplateHike, nil, map[string]any{
				"hillPreference": hills,
			}, "foot"),
		)
		if err != nil {
			t.Fatalf("render hiking profile at %v: %v", hills, err)
		}
		if !strings.Contains(rendered, "consider_elevation     = true") {
			t.Fatalf("hill preference %v switched off elevation and tripled the cost of steps", hills)
		}
	}
}
