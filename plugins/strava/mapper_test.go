package main

import "testing"

func TestActivityTypeMapsRunsToRunning(t *testing.T) {
	cases := map[string]string{
		"Run":        "running",
		"TrailRun":   "running",
		"VirtualRun": "running",
		"Walk":       "walking",
	}

	for providerType, want := range cases {
		if got := activityTypeFromProvider(providerType); got != want {
			t.Fatalf("activityTypeFromProvider(%q) = %q, want %q", providerType, got, want)
		}
	}
}
