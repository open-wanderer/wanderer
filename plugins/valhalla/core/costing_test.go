package core

import (
	"reflect"
	"testing"
)

func TestResolveCosting(t *testing.T) {
	tests := []struct {
		name       string
		profileKey string
		mode       string
		want       string
	}{
		{name: "explicit profile", profileKey: " bicycle ", mode: "foot", want: "bicycle"},
		{name: "foot default", mode: "foot", want: "pedestrian"},
		{name: "bike default", profileKey: " ", mode: "bike", want: "bicycle"},
		{name: "motor default", mode: "motor", want: "auto"},
		{name: "unsupported mode", mode: "unknown", want: ""},
		{name: "empty", mode: "", want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ResolveCosting(test.profileKey, test.mode); got != test.want {
				t.Fatalf("ResolveCosting(%q, %q) = %q, want %q", test.profileKey, test.mode, got, test.want)
			}
		})
	}
}

func TestBuildCostingOptionsTranslatesPreferences(t *testing.T) {
	tests := []struct {
		name        string
		costing     string
		preferences map[string]any
		want        map[string]any
	}{
		{
			name:    "pedestrian",
			costing: "pedestrian",
			preferences: map[string]any{
				"speedPreference":     5.5,
				"hillPreference":      float32(0.4),
				"maxHikingDifficulty": 3,
				"roadPreference":      0.9,
			},
			want: map[string]any{
				"walking_speed":         5.5,
				"use_hills":             float32(0.4),
				"max_hiking_difficulty": 3,
			},
		},
		{
			name:    "bicycle",
			costing: "bicycle",
			preferences: map[string]any{
				"speedPreference":  int64(24),
				"hillPreference":   0.2,
				"roadPreference":   0.7,
				"avoidBadSurfaces": 0.8,
			},
			want: map[string]any{
				"cycling_speed":      int64(24),
				"use_hills":          0.2,
				"use_roads":          0.7,
				"avoid_bad_surfaces": 0.8,
			},
		},
		{
			name:    "auto",
			costing: "auto",
			preferences: map[string]any{
				"speedPreference": 130,
				"vehicleWidth":    2.1,
				"vehicleHeight":   2.8,
			},
			want: map[string]any{
				"top_speed": 130,
				"width":     2.1,
				"height":    2.8,
			},
		},
		{
			name:        "non-numeric values are ignored",
			costing:     "pedestrian",
			preferences: map[string]any{"speedPreference": "fast", "hillPreference": true},
			want:        map[string]any{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			want := map[string]any{test.costing: test.want}
			if got := BuildCostingOptions(test.costing, test.preferences, nil); !reflect.DeepEqual(got, want) {
				t.Fatalf("BuildCostingOptions() = %#v, want %#v", got, want)
			}
		})
	}
}

func TestBuildCostingOptionsMergesOnlySelectedNativeNamespace(t *testing.T) {
	got := BuildCostingOptions(
		"bicycle",
		map[string]any{"speedPreference": 20.0, "hillPreference": 0.5},
		map[string]any{
			"bicycle":      map[string]any{"cycling_speed": 25.0, "shortest": true},
			"pedestrian":   map[string]any{"walking_speed": 6.0},
			"bicycle_type": "Road",
		},
	)
	want := map[string]any{"bicycle": map[string]any{
		"cycling_speed": 25.0,
		"use_hills":     0.5,
		"shortest":      true,
		"bicycle_type":  "Road",
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildCostingOptions() = %#v, want %#v", got, want)
	}
	if _, found := got["pedestrian"]; found {
		t.Fatalf("unselected costing namespace leaked: %#v", got)
	}
}
