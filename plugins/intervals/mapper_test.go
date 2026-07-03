package main

import "testing"

func TestMapActivityType(t *testing.T) {
	cases := map[string]string{
		"Ride":        "biking",
		"VirtualRide": "biking",
		"Run":         "running",
		"TrailRun":    "running",
		"Hike":        "hiking",
		"Walk":        "walking",
		"Swim":        "swimming",
		"Unknown":     "other",
	}

	for providerType, want := range cases {
		if got := mapActivityType(providerType); got != want {
			t.Fatalf("mapActivityType(%q) = %q, want %q", providerType, got, want)
		}
	}
}
