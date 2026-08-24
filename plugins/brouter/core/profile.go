package core

import (
	"reflect"
	"strconv"
	"strings"
)

type profileModeAssignment struct {
	mode     string
	variable string
}

var profileModeAssignments = []profileModeAssignment{
	{
		mode:     "bike",
		variable: "validForBikes",
	},
	{
		mode:     "foot",
		variable: "validForFoot",
	},
	{
		mode:     "motor",
		variable: "validForCars",
	},
}

// DetectProfileMode returns a mode only when the BRouter profile enables
// exactly one of its standard validFor flags.
func DetectProfileMode(content string) string {
	detected := ""
	for _, line := range strings.Split(content, "\n") {
		if comment := strings.IndexByte(line, '#'); comment >= 0 {
			line = line[:comment]
		}
		fields := strings.Fields(strings.ReplaceAll(line, "=", " = "))
		if len(fields) < 3 || !strings.EqualFold(fields[0], "assign") {
			continue
		}
		valueIndex := 2
		if fields[valueIndex] == "=" {
			valueIndex++
		}
		if valueIndex >= len(fields) || !profileModeEnabledValue(fields[valueIndex]) {
			continue
		}
		mode := ""
		for _, assignment := range profileModeAssignments {
			if strings.EqualFold(fields[1], assignment.variable) {
				mode = assignment.mode
				break
			}
		}
		if mode == "" {
			continue
		}
		if detected != "" && detected != mode {
			return ""
		}
		detected = mode
	}
	return detected
}

func profileModeEnabledValue(value string) bool {
	if strings.EqualFold(value, "true") {
		return true
	}
	number, err := strconv.ParseFloat(value, 64)
	return err == nil && number == 1
}

// TemplateKeyForParameters infers which upstream profile an uploaded file is
// built on, so uploads of the well-known BRouter profiles get the same
// preference translation as the curated ones. It takes the declared value types
// because `consider_traffic` is what separates the trekking and fastbike
// families, and it is a flag in one and a scale in the other. A family is only
// selected from an unambiguous signature; partial or contradictory signatures
// return no base and fall back to the generic mapping, which writes the same
// values as the trekking adapter minus the family-specific knobs.
func TemplateKeyForParameters(parameters map[string]string) string {
	_, hasSACLimit := parameters["SAC_scale_limit"]
	_, hasSACPreferred := parameters["SAC_scale_preferred"]
	_, hasPreferUnpaved := parameters["prefer_unpaved_paths"]
	_, hasSteepInclines := parameters["avoid_steep_inclines"]
	_, hasMTBFactor := parameters["MTB_factor"]
	_, hasHills := parameters["hills"]
	_, hasPathPreference := parameters["path_preference"]
	_, hasVmax := parameters["vmax"]
	_, hasAvoidUnpaved := parameters["avoid_unpaved"]
	_, hasAvoidToll := parameters["avoid_toll"]
	_, hasAvoidUnsafe := parameters["avoid_unsafe"]
	_, hasUnpavedPenalty := parameters["unpavedPenalty"]
	_, hasBadSurfaceCost := parameters["bad_surface_cost"]
	_, hasBikerPower := parameters["bikerPower"]
	_, hasAllowMotorways := parameters["allow_motorways"]
	_, hasStickToCycleRoutes := parameters["stick_to_cycleroutes"]
	hasSurfaceKnob := hasUnpavedPenalty || hasBadSurfaceCost
	hasTrekkingOnlyKnob := hasAvoidUnsafe || hasStickToCycleRoutes

	candidates := map[string]bool{}
	if hasSACLimit || hasSACPreferred {
		candidates[TemplateHike] = true
	}
	if hasPreferUnpaved && hasSteepInclines {
		candidates[TemplateGravel] = true
	}
	if hasMTBFactor || (hasHills && hasPathPreference) {
		candidates[TemplateMTB] = true
	}
	if hasVmax && (hasAvoidUnpaved || hasAvoidToll) {
		candidates[TemplateCar] = true
	}

	// The traffic parameter and the kinematic model are shared by many custom
	// profiles. The family-specific access knob is what makes the combination a
	// sufficiently strong fingerprint. A mixed signature deliberately matches
	// neither family and therefore keeps the profile's generic calibration.
	switch parameters["consider_traffic"] {
	case "boolean":
		if (hasTrekkingOnlyKnob || hasSurfaceKnob) && !hasAllowMotorways {
			candidates[TemplateTrekking] = true
		}
	case "number":
		if hasBikerPower && hasAllowMotorways && !hasTrekkingOnlyKnob {
			candidates[TemplateFastbike] = true
		}
	case "":
		// Profiles shipped by older Wanderer versions predate the traffic
		// annotation but expose this otherwise characteristic pair.
		if hasAvoidUnsafe && hasSurfaceKnob && !hasAllowMotorways {
			candidates[TemplateTrekking] = true
		}
	}
	if len(candidates) != 1 {
		return ""
	}
	for candidate := range candidates {
		return candidate
	}
	return ""
}

// SupportedStandardPreferences returns the Wanderer preferences that can be
// translated to parameters explicitly exposed by a BRouter profile. It probes
// the preference adapter at both ends of each preference range and only
// advertises a control when at least one exposed target actually changes. That
// avoids no-op controls when a recognized family writes a constant helper value
// in addition to its real, but unexposed, cost parameters.
func SupportedStandardPreferences(parameters map[string]string, mode string) []string {
	templateKey := TemplateKeyForParameters(parameters)

	candidates := []string{}
	switch mode {
	case "foot":
		candidates = []string{"hillPreference", "maxHikingDifficulty"}
	case "bike":
		candidates = []string{"speedPreference", "hillPreference", "roadPreference", "avoidBadSurfaces"}
	case "motor":
		candidates = []string{"speedPreference", "avoidBadSurfaces"}
	}

	supported := []string{}
	for _, preference := range candidates {
		low, high := standardPreferenceProbeValues(preference)
		lowConfig := NativeConfigWithPreferences(templateKey, nil, map[string]any{preference: low}, mode)
		highConfig := NativeConfigWithPreferences(templateKey, nil, map[string]any{preference: high}, mode)
		if exposedParameterChanges(parameters, mapValue(lowConfig["parameters"]), mapValue(highConfig["parameters"])) {
			supported = append(supported, preference)
		}
	}
	return supported
}

func standardPreferenceProbeValues(preference string) (float64, float64) {
	switch preference {
	case "speedPreference":
		return 0, 300
	case "maxHikingDifficulty":
		return 0, 6
	default:
		return 0, 1
	}
}

func exposedParameterChanges(exposed map[string]string, low map[string]any, high map[string]any) bool {
	for key := range exposed {
		lowValue, lowOK := low[key]
		highValue, highOK := high[key]
		if lowOK != highOK || (lowOK && !reflect.DeepEqual(lowValue, highValue)) {
			return true
		}
	}
	return false
}
