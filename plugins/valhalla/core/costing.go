package core

import "strings"

// ResolveCosting returns an explicitly selected Valhalla costing or the
// default costing for the Wanderer routing mode. An empty result means that
// neither is available.
func ResolveCosting(profileKey string, mode string) string {
	if costing := strings.TrimSpace(profileKey); costing != "" {
		return costing
	}
	return costingForMode(mode)
}

func costingForMode(mode string) string {
	switch mode {
	case "foot":
		return "pedestrian"
	case "bike":
		return "bicycle"
	case "motor":
		return "auto"
	default:
		return ""
	}
}

// BuildCostingOptions builds Valhalla's costing_options object. Generic routing
// preferences are translated first, then the selected native costing namespace
// and profile-wide native options override them.
func BuildCostingOptions(costing string, preferences map[string]any, nativeConfig map[string]any) map[string]any {
	options := map[string]any{}
	switch costing {
	case "pedestrian":
		setNumberOption(options, "walking_speed", preferences, "speedPreference")
		setNumberOption(options, "use_hills", preferences, "hillPreference")
		setNumberOption(options, "max_hiking_difficulty", preferences, "maxHikingDifficulty")
	case "bicycle":
		setNumberOption(options, "cycling_speed", preferences, "speedPreference")
		setNumberOption(options, "use_hills", preferences, "hillPreference")
		setNumberOption(options, "use_roads", preferences, "roadPreference")
		setNumberOption(options, "avoid_bad_surfaces", preferences, "avoidBadSurfaces")
	case "auto":
		setNumberOption(options, "top_speed", preferences, "speedPreference")
		setNumberOption(options, "width", preferences, "vehicleWidth")
		setNumberOption(options, "height", preferences, "vehicleHeight")
	}
	mergeNativeCostingOptions(options, costing, nativeConfig)
	return map[string]any{costing: options}
}

func mergeNativeCostingOptions(options map[string]any, costing string, nativeConfig map[string]any) {
	for key, value := range nativeConfig {
		if key == costing {
			if nested, ok := value.(map[string]any); ok {
				for nestedKey, nestedValue := range nested {
					options[nestedKey] = nestedValue
				}
			}
			continue
		}
		// Flat values are retained for declared profile-wide Valhalla options
		// such as bicycle_type. Other costing namespaces are never forwarded.
		if _, nested := value.(map[string]any); !nested {
			options[key] = value
		}
	}
}

func setNumberOption(options map[string]any, optionKey string, preferences map[string]any, preferenceKey string) {
	value, ok := preferences[preferenceKey]
	if !ok {
		return
	}
	switch value := value.(type) {
	case float64:
		options[optionKey] = value
	case float32:
		options[optionKey] = value
	case int:
		options[optionKey] = value
	case int64:
		options[optionKey] = value
	}
}
