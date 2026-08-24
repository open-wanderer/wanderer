package core

// Template keys identify the upstream BRouter profile a curated or generated
// profile is rendered from. Wanderer's standard preferences translate
// differently per base because the upstream profiles expose different knobs:
// the BRouter core profiles carry a kinematic model and numeric costs, while
// the Poutnik templates (hiking-mountain, mtb) and quaelnix' gravel profile
// expose stepped switches instead.
const (
	TemplateHike     = "hike"          // hiking-mountain.brf
	TemplateTrekking = "bike-balanced" // trekking.brf
	TemplateFastbike = "fastbike"      // fastbike.brf
	TemplateGravel   = "gravel"        // gravel.brf
	TemplateMTB      = "mtb"           // mtb.brf
	TemplateCar      = "car"           // car-vario.brf
)

// Neutral preference values keep the base profile's own cost values, so a
// category preset only has to name what it wants to deviate in. One deliberate
// exception: the hiking base ships with elevation costs switched off entirely,
// while Wanderer keeps them switched on and expresses the preference through
// the cost values instead — a hiking route that ignores climbs is not what the
// slider promises, and that switch has a second job in this profile.
const (
	NeutralHillPreference        = 0.5
	NeutralTrekkingSpeed         = 20.0
	NeutralFastbikeSpeed         = 28.0
	NeutralGravelSpeed           = 18.0
	NeutralTrekkingBadSurfaces   = 0.5
	NeutralFastbikeBadSurfaces   = 0.8
	NeutralMTBBadSurfaces        = 0.5
	NeutralRoadPreference        = 0.5
	NeutralCarBadSurfaces        = 0.5
	hillPreferenceAvoidStrongly  = 0.35
	hillPreferenceIgnore         = 0.7
	gravelBadSurfacesPreferUnpav = 0.35
	gravelBadSurfacesAssumeWet   = 0.8

	// MinCarSpeed is the lowest target speed the car kinematic model handles
	// without the routing search becoming pathological.
	MinCarSpeed = 20.0
)

func NativeConfigWithPreferences(templateKey string, nativeConfig map[string]any, preferences map[string]any, mode string) map[string]any {
	output := cloneMap(nativeConfig)
	parameters := cloneMap(MapValue(output["parameters"]))

	switch templateKey {
	case TemplateHike:
		applyHikePreferences(parameters, preferences)
	case TemplateTrekking:
		applyTrekkingPreferences(parameters, preferences)
	case TemplateFastbike:
		applyFastbikePreferences(parameters, preferences)
	case TemplateGravel:
		applyGravelPreferences(parameters, preferences)
	case TemplateMTB:
		applyMTBPreferences(parameters, preferences)
	case TemplateCar:
		applyCarPreferences(parameters, preferences)
	default:
		applyUploadedProfilePreferences(parameters, preferences, mode)
	}

	if len(parameters) > 0 {
		output["parameters"] = parameters
	}
	return output
}

// hiking-mountain exposes no kinematic model, so speedPreference has no target
// and is deliberately dropped instead of silently rendered somewhere else.
func applyHikePreferences(parameters map[string]any, preferences map[string]any) {
	if hills, ok := NumericValue(preferences["hillPreference"]); ok {
		// The hiking base overloads consider_elevation: besides gating the climb
		// costs it also triples the cost of steps. Toggling it would make the top
		// of the slider quietly start avoiding stairs, so it stays on and the
		// cost values carry the whole range — they reach zero at the top, which
		// is what "elevation does not matter" means.
		parameters["consider_elevation"] = true
		cost := CostForPreference(hills, NeutralHillPreference, 7, 30)
		parameters["uphillcostvalue"] = cost
		parameters["downhillcostvalue"] = cost
	}
	if difficulty, ok := NumericValue(preferences["maxHikingDifficulty"]); ok {
		limit := ClampFloat(difficulty, 0, 6)
		parameters["SAC_scale_limit"] = limit
		parameters["SAC_scale_preferred"] = PreferredHikingDifficulty(limit)
	}
}

func applyTrekkingPreferences(parameters map[string]any, preferences map[string]any) {
	if speed, ok := NumericValue(preferences["speedPreference"]); ok {
		parameters["bikerPower"] = BikerPower(speed, NeutralTrekkingSpeed, 100)
	}
	if hills, ok := NumericValue(preferences["hillPreference"]); ok {
		parameters["consider_elevation"] = hills < 0.95
		parameters["uphillcost"] = CostForPreference(hills, NeutralHillPreference, 0, 120)
	}
	if roads, ok := NumericValue(preferences["roadPreference"]); ok {
		// Both knobs are boolean in the trekking family, so the slider acts as
		// two thresholds rather than a scale.
		parameters["avoid_unsafe"] = roads < NeutralRoadPreference
		parameters["consider_traffic"] = roads < 0.35
	}
	if surfaces, ok := NumericValue(preferences["avoidBadSurfaces"]); ok {
		setSurfacePenalty(parameters, PenaltyForPreference(surfaces, NeutralTrekkingBadSurfaces, 1, 5))
	}
}

func applyFastbikePreferences(parameters map[string]any, preferences map[string]any) {
	if speed, ok := NumericValue(preferences["speedPreference"]); ok {
		parameters["bikerPower"] = BikerPower(speed, NeutralFastbikeSpeed, 100)
	}
	if hills, ok := NumericValue(preferences["hillPreference"]); ok {
		parameters["consider_elevation"] = hills < 0.95
		parameters["uphillcost"] = CostForPreference(hills, NeutralHillPreference, 0, 120)
	}
	if roads, ok := NumericValue(preferences["roadPreference"]); ok {
		// fastbike is the only base with a real 0..1 traffic scale.
		parameters["consider_traffic"] = ClampFloat(1-roads, 0, 1)
	}
	if surfaces, ok := NumericValue(preferences["avoidBadSurfaces"]); ok {
		setSurfacePenalty(parameters, PenaltyForPreference(surfaces, NeutralFastbikeBadSurfaces, 1, 5))
	}
}

// quaelnix' gravel profile derives its elevation and surface costs from
// booleans, so both preferences collapse into three steps each.
func applyGravelPreferences(parameters map[string]any, preferences map[string]any) {
	if speed, ok := NumericValue(preferences["speedPreference"]); ok {
		parameters["bikerPower"] = BikerPower(speed, NeutralGravelSpeed, 150)
	}
	if hills, ok := NumericValue(preferences["hillPreference"]); ok {
		// uphillcost is `consider_elevation ? 80 : avoid_steep_inclines ? 160 : 0`.
		parameters["consider_elevation"] = hills >= hillPreferenceAvoidStrongly && hills < hillPreferenceIgnore
		parameters["avoid_steep_inclines"] = hills < hillPreferenceAvoidStrongly
	}
	if roads, ok := NumericValue(preferences["roadPreference"]); ok {
		parameters["consider_traffic_estimate"] = roads < NeutralRoadPreference
	}
	if surfaces, ok := NumericValue(preferences["avoidBadSurfaces"]); ok {
		parameters["prefer_unpaved_paths"] = surfaces < gravelBadSurfacesPreferUnpav
		parameters["assume_wet_conditions"] = surfaces > gravelBadSurfacesAssumeWet
	}
}

// The Poutnik MTB template has no kinematic model either; its elevation
// handling is a discrete mode rather than a cost.
func applyMTBPreferences(parameters map[string]any, preferences map[string]any) {
	if hills, ok := NumericValue(preferences["hillPreference"]); ok {
		parameters["hills"] = HillsMode(hills)
	}
	if roads, ok := NumericValue(preferences["roadPreference"]); ok {
		parameters["avoid_unsafe"] = roads < NeutralRoadPreference
	}
	if surfaces, ok := NumericValue(preferences["avoidBadSurfaces"]); ok {
		// MTB_factor is a -3..3 shift between rough terrain and small paved roads.
		parameters["MTB_factor"] = ClampFloat((NeutralMTBBadSurfaces-surfaces)*3, -1.5, 1.5)
	}
}

func applyCarPreferences(parameters map[string]any, preferences map[string]any) {
	if speed, ok := NumericValue(preferences["speedPreference"]); ok {
		// Target speeds below ~20 km/h make the car kinematic model explore so
		// much of the graph that providers cut the search short.
		parameters["vmax"] = ClampFloat(speed, MinCarSpeed, 300)
	}
	if surfaces, ok := NumericValue(preferences["avoidBadSurfaces"]); ok {
		parameters["avoid_unpaved"] = surfaces > NeutralCarBadSurfaces
	}
}

// applyUploadedProfilePreferences targets user-uploaded profiles whose base
// could not be identified. It emits every upstream parameter name a preference
// can map to; rendering drops the ones the profile does not expose.
func applyUploadedProfilePreferences(parameters map[string]any, preferences map[string]any, mode string) {
	switch mode {
	case "foot":
		if hills, ok := NumericValue(preferences["hillPreference"]); ok {
			parameters["consider_elevation"] = hills < 0.95
			cost := CostForPreference(hills, NeutralHillPreference, 7, 30)
			parameters["uphillcostvalue"] = cost
			parameters["downhillcostvalue"] = cost
		}
		if difficulty, ok := NumericValue(preferences["maxHikingDifficulty"]); ok {
			limit := ClampFloat(difficulty, 0, 6)
			parameters["SAC_scale_limit"] = limit
			parameters["SAC_scale_preferred"] = PreferredHikingDifficulty(limit)
		}
	case "bike":
		if speed, ok := NumericValue(preferences["speedPreference"]); ok {
			parameters["bikerPower"] = BikerPower(speed, NeutralTrekkingSpeed, 100)
		}
		if hills, ok := NumericValue(preferences["hillPreference"]); ok {
			parameters["consider_elevation"] = hills < 0.95
			parameters["uphillcost"] = CostForPreference(hills, NeutralHillPreference, 0, 120)
			parameters["hills"] = HillsMode(hills)
		}
		if roads, ok := NumericValue(preferences["roadPreference"]); ok {
			parameters["avoid_unsafe"] = roads < NeutralRoadPreference
			parameters["consider_traffic"] = ClampFloat(1-roads, 0, 1)
		}
		if surfaces, ok := NumericValue(preferences["avoidBadSurfaces"]); ok {
			setSurfacePenalty(parameters, PenaltyForPreference(surfaces, NeutralTrekkingBadSurfaces, 1, 5))
		}
	case "motor":
		if speed, ok := NumericValue(preferences["speedPreference"]); ok {
			parameters["vmax"] = ClampFloat(speed, MinCarSpeed, 300)
		}
		if surfaces, ok := NumericValue(preferences["avoidBadSurfaces"]); ok {
			parameters["avoid_unpaved"] = surfaces > NeutralCarBadSurfaces
		}
	}
}

// setSurfacePenalty writes the upstream surface knob together with the legacy
// name Wanderer used before it adopted `unpavedPenalty`. Profiles uploaded from
// those older presets keep responding to the preference; rendering drops
// whichever of the two names a profile does not expose.
func setSurfacePenalty(parameters map[string]any, penalty float64) {
	parameters["unpavedPenalty"] = penalty
	// The legacy knob multiplies the unpaved cost instead of adding to it, so
	// dropping below 1 there would make rough ways cheaper than paved ones.
	parameters["bad_surface_cost"] = ClampFloat(penalty, 1, 5)
}

// BikerPower converts a target speed into the kinematic model's rider power.
func BikerPower(speed float64, neutralSpeed float64, neutralPower float64) float64 {
	return ClampFloat(neutralPower+((speed-neutralSpeed)*6), 50, 300)
}

// CostForPreference maps a preference where a lower value means "avoid this
// more" onto a BRouter cost: the neutral value yields defaultCost, 0 yields
// maxCost, and 1 removes the cost entirely.
func CostForPreference(preference float64, neutral float64, defaultCost float64, maxCost float64) float64 {
	if neutral <= 0 {
		return ClampFloat(defaultCost, 0, maxCost)
	}
	if preference < neutral {
		return ClampFloat(defaultCost+((neutral-preference)/neutral)*(maxCost-defaultCost), 0, maxCost)
	}
	if neutral >= 1 {
		return ClampFloat(defaultCost, 0, maxCost)
	}
	return ClampFloat(defaultCost*(1-((preference-neutral)/(1-neutral))), 0, maxCost)
}

// PenaltyForPreference maps a preference where a higher value means "avoid this
// more" onto an upstream penalty multiplier: the neutral value keeps the
// profile's own default, 0 drops the surcharge completely, and 1 yields
// maxPenalty.
func PenaltyForPreference(preference float64, neutral float64, defaultPenalty float64, maxPenalty float64) float64 {
	if neutral <= 0 {
		return ClampFloat(maxPenalty, 0, maxPenalty)
	}
	if preference <= neutral {
		return ClampFloat(defaultPenalty*(preference/neutral), 0, maxPenalty)
	}
	if neutral >= 1 {
		return ClampFloat(defaultPenalty, 0, maxPenalty)
	}
	return ClampFloat(defaultPenalty+((preference-neutral)/(1-neutral))*(maxPenalty-defaultPenalty), 0, maxPenalty)
}

// PreferredHikingDifficulty keeps the preferred SAC level clearly below the
// allowed limit. Deriving it as limit-2 would make a tolerant limit actively
// prefer difficult alpine terrain and penalize easy paths.
func PreferredHikingDifficulty(limit float64) float64 {
	preferred := float64(int(limit / 2))
	if preferred > limit {
		return limit
	}
	return ClampFloat(preferred, 0, 6)
}

// HillsMode maps a hill preference onto the Poutnik templates' discrete
// elevation modes: 0 leaves elevation alone, 1 penalizes climbs above 3%, and
// 2 avoids slopes altogether.
func HillsMode(hillPreference float64) float64 {
	switch {
	case hillPreference >= hillPreferenceIgnore:
		return 0
	case hillPreference >= hillPreferenceAvoidStrongly:
		return 1
	default:
		return 2
	}
}

func ClampFloat(value float64, min float64, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func NumericValue(value any) (float64, bool) {
	switch value := value.(type) {
	case float64:
		return value, true
	case float32:
		return float64(value), true
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	case int32:
		return float64(value), true
	case jsonNumber:
		parsed, err := value.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

type jsonNumber interface {
	Float64() (float64, error)
}

// MapValue returns a JSON object or an empty object for any other value.
// Keeping this conversion in core lets the preference translator and profile
// renderer interpret decoded configuration identically.
func MapValue(value any) map[string]any {
	if value, ok := value.(map[string]any); ok && value != nil {
		return value
	}
	return map[string]any{}
}

func cloneMap(input map[string]any) map[string]any {
	output := map[string]any{}
	for key, value := range input {
		if nested, ok := value.(map[string]any); ok {
			output[key] = cloneMap(nested)
			continue
		}
		output[key] = value
	}
	return output
}
