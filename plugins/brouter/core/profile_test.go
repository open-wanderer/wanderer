package core

import "testing"

func TestProfileNumericValueKeepsPreferenceNumbersStrict(t *testing.T) {
	if value, ok := ProfileNumericValue("12.5"); !ok || value != 12.5 {
		t.Fatalf("profile numeric string = %v, %v", value, ok)
	}
	if _, ok := ProfileNumericValue("not-a-number"); ok {
		t.Fatal("invalid profile numeric string was accepted")
	}
	if value, ok := ProfileNumericValue(12.5); !ok || value != 12.5 {
		t.Fatalf("profile JSON number = %v, %v", value, ok)
	}
	if _, ok := NumericValue("12.5"); ok {
		t.Fatal("standard preference numeric conversion accepted a string")
	}
}

func TestDetectProfileMode(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{name: "bike", content: "assign validForBikes = true", want: "bike"},
		{name: "foot", content: "assign validForFoot        1", want: "foot"},
		{name: "foot compact assignment", content: "assign validForFoot=1 # walking", want: "foot"},
		{name: "foot case insensitive", content: "ASSIGN VALIDFORFOOT = TRUE", want: "foot"},
		{name: "motor", content: "assign validForCars = 1.0", want: "motor"},
		{name: "duplicate mode remains unambiguous", content: "assign validForFoot 1\nassign validForFoot = true", want: "foot"},
		{
			name: "disabled and commented",
			content: `
# assign validForBikes = true
assign validForFoot = false
`,
		},
		{
			name: "ambiguous",
			content: `
assign validForBikes = true
assign validForFoot = true
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := DetectProfileMode(test.content); got != test.want {
				t.Fatalf("DetectProfileMode() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestSupportedStandardPreferences(t *testing.T) {
	tests := []struct {
		name       string
		mode       string
		parameters map[string]string
		want       []string
	}{
		{
			name: "trekking profile",
			mode: "bike",
			parameters: map[string]string{
				"bikerPower": "number", "consider_elevation": "boolean",
				"avoid_unsafe": "boolean", "consider_traffic": "boolean", "unpavedPenalty": "number",
			},
			want: []string{"speedPreference", "hillPreference", "roadPreference", "avoidBadSurfaces"},
		},
		{
			name: "fastbike profile",
			mode: "bike",
			parameters: map[string]string{
				"bikerPower": "number", "consider_elevation": "boolean",
				"consider_traffic": "number", "unpavedPenalty": "number", "allow_motorways": "boolean",
			},
			want: []string{"speedPreference", "hillPreference", "roadPreference", "avoidBadSurfaces"},
		},
		{
			name: "gravel profile",
			mode: "bike",
			parameters: map[string]string{
				"bikerPower": "number", "consider_elevation": "boolean", "avoid_steep_inclines": "boolean",
				"consider_traffic_estimate": "boolean", "prefer_unpaved_paths": "boolean",
			},
			want: []string{"speedPreference", "hillPreference", "roadPreference", "avoidBadSurfaces"},
		},
		{
			name: "mountain bike profile has no kinematic model",
			mode: "bike",
			parameters: map[string]string{
				"hills": "number", "avoid_unsafe": "boolean", "MTB_factor": "number",
			},
			want: []string{"hillPreference", "roadPreference", "avoidBadSurfaces"},
		},
		{
			name:       "recognized hike does not advertise a constant elevation switch",
			mode:       "foot",
			parameters: map[string]string{"SAC_scale_limit": "number", "consider_elevation": "boolean"},
			want:       []string{"maxHikingDifficulty"},
		},
		{
			name: "recognized hike advertises hills when a variable cost is exposed",
			mode: "foot",
			parameters: map[string]string{
				"SAC_scale_limit": "number", "consider_elevation": "boolean", "uphillcostvalue": "number",
			},
			want: []string{"hillPreference", "maxHikingDifficulty"},
		},
		{
			name:       "motor profile",
			mode:       "motor",
			parameters: map[string]string{"vmax": "number", "avoid_toll": "boolean", "totalweight": "number"},
			want:       []string{"speedPreference"},
		},
		{
			name:       "parameters for another mode are ignored",
			mode:       "foot",
			parameters: map[string]string{"bikerPower": "number", "avoid_unsafe": "boolean"},
			want:       []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := SupportedStandardPreferences(test.parameters, test.mode)
			if len(got) != len(test.want) {
				t.Fatalf("SupportedStandardPreferences() = %#v, want %#v", got, test.want)
			}
			for index := range got {
				if got[index] != test.want[index] {
					t.Fatalf("SupportedStandardPreferences() = %#v, want %#v", got, test.want)
				}
			}
		})
	}
}

func TestTemplateKeyForParameters(t *testing.T) {
	tests := []struct {
		name       string
		parameters map[string]string
		want       string
	}{
		{
			name:       "trekking and fastbike differ in the type of consider_traffic",
			parameters: map[string]string{"avoid_unsafe": "boolean", "consider_traffic": "boolean", "bikerPower": "number"},
			want:       TemplateTrekking,
		},
		{
			name: "the complete fastbike signature is recognized",
			parameters: map[string]string{
				"consider_traffic": "number", "bikerPower": "number", "allow_motorways": "boolean",
			},
			want: TemplateFastbike,
		},
		{
			name:       "a single weak signal is not enough",
			parameters: map[string]string{"avoid_unsafe": "boolean", "allow_steps": "boolean"},
			want:       "",
		},
		{
			name:       "gravel needs both of its own switches",
			parameters: map[string]string{"prefer_unpaved_paths": "boolean", "avoid_steep_inclines": "boolean"},
			want:       TemplateGravel,
		},
		{
			name:       "a lone vmax is not a car profile",
			parameters: map[string]string{"vmax": "number"},
			want:       "",
		},
		{
			name:       "profiles Wanderer shipped before the rename still resolve",
			parameters: map[string]string{"avoid_unsafe": "boolean", "bad_surface_cost": "number"},
			want:       TemplateTrekking,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := TemplateKeyForParameters(test.parameters); got != test.want {
				t.Fatalf("TemplateKeyForParameters() = %q, want %q", got, test.want)
			}
		})
	}
}

// Profiles uploaded from the presets Wanderer shipped before adopting the
// upstream parameter name must keep responding to the surface preference.
func TestSurfacePreferenceAlsoWritesTheLegacyParameter(t *testing.T) {
	parameters := NativeConfigWithPreferences(TemplateTrekking, nil, map[string]any{
		"avoidBadSurfaces": 1,
	}, "bike")["parameters"].(map[string]any)
	assertFloat(t, parameters["unpavedPenalty"], 5)
	assertFloat(t, parameters["bad_surface_cost"], 5)

	tolerant := NativeConfigWithPreferences(TemplateTrekking, nil, map[string]any{
		"avoidBadSurfaces": 0,
	}, "bike")["parameters"].(map[string]any)
	assertFloat(t, tolerant["unpavedPenalty"], 0)
	// The legacy knob is a multiplier: below 1 it would make rough ways cheap.
	assertFloat(t, tolerant["bad_surface_cost"], 1)
}

func TestTemplateKeyForParametersNeedsMoreThanOneSignal(t *testing.T) {
	tests := []struct {
		name       string
		parameters map[string]string
		want       string
	}{
		{
			name:       "a lone traffic flag is not trekking",
			parameters: map[string]string{"consider_traffic": "boolean"},
			want:       "",
		},
		{
			name:       "a lone traffic scale is not fastbike",
			parameters: map[string]string{"consider_traffic": "number"},
			want:       "",
		},
		{
			name:       "a traffic scale next to rider power remains generic",
			parameters: map[string]string{"consider_traffic": "number", "bikerPower": "number"},
			want:       "",
		},
		{
			name: "a complete fastbike signature is accepted",
			parameters: map[string]string{
				"consider_traffic": "number", "bikerPower": "number", "allow_motorways": "boolean",
			},
			want: TemplateFastbike,
		},
		{
			name: "a mixed fastbike and trekking signature remains generic",
			parameters: map[string]string{
				"consider_traffic": "number", "bikerPower": "number", "allow_motorways": "boolean",
				"avoid_unsafe": "boolean",
			},
			want: "",
		},
		{
			name: "two otherwise complete family signatures remain generic",
			parameters: map[string]string{
				"consider_traffic": "number", "bikerPower": "number", "allow_motorways": "boolean",
				"SAC_scale_limit": "number",
			},
			want: "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := TemplateKeyForParameters(test.parameters); got != test.want {
				t.Fatalf("TemplateKeyForParameters() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestPartialFastbikeSignatureKeepsTheGenericSpeedCalibration(t *testing.T) {
	templateKey := TemplateKeyForParameters(map[string]string{
		"consider_traffic": "number",
		"bikerPower":       "number",
	})
	if templateKey != "" {
		t.Fatalf("partial template signature = %q, want generic adapter", templateKey)
	}

	nativeConfig := NativeConfigWithPreferences(templateKey, nil, map[string]any{
		"speedPreference": NeutralTrekkingSpeed,
	}, "bike")
	parameters := nativeConfig["parameters"].(map[string]any)
	assertFloat(t, parameters["bikerPower"], 100)
}
