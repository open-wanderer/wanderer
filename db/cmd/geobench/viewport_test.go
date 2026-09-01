package main

import (
	"math"
	"strings"
	"testing"
)

func TestGeoBoundingBoxFilter(t *testing.T) {
	tests := []struct {
		name     string
		viewport GeoViewport
		want     string
	}{
		{
			name:     "ordinary viewport",
			viewport: GeoViewport{North: 47.4, West: 8.5, South: 47.3, East: 8.6},
			want:     "_geoBoundingBox([47.4000000, 8.6000000], [47.3000000, 8.5000000])",
		},
		{
			name:     "antimeridian viewport",
			viewport: GeoViewport{North: 1, West: 179.8, South: -1, East: -179.8},
			want:     "(_geoBoundingBox([1.0000000, 180.0000000], [-1.0000000, 179.8000000]) OR _geoBoundingBox([1.0000000, -179.8000000], [-1.0000000, -180.0000000]))",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := geoBoundingBoxFilter(test.viewport)
			if err != nil {
				t.Fatalf("build filter: %v", err)
			}
			if got != test.want {
				t.Fatalf("filter = %q, want %q", got, test.want)
			}
		})
	}
}

func TestGeoBoundingBoxFilterRejectsInvalidViewport(t *testing.T) {
	tests := []struct {
		name     string
		viewport GeoViewport
		want     string
	}{
		{name: "latitude order", viewport: GeoViewport{North: 46, West: 8, South: 47, East: 9}, want: "north"},
		{name: "zero width", viewport: GeoViewport{North: 47, West: 8, South: 46, East: 8}, want: "must differ"},
		{name: "longitude range", viewport: GeoViewport{North: 47, West: -181, South: 46, East: 8}, want: "west"},
		{name: "not finite", viewport: GeoViewport{North: math.NaN(), West: 8, South: 46, East: 9}, want: "north"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := geoBoundingBoxFilter(test.viewport)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want text %q", err, test.want)
			}
		})
	}
}
