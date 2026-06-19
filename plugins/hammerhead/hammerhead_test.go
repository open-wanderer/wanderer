package main

import (
	"strings"
	"testing"

	sdkgpx "github.com/open-wanderer/wanderer/plugins/sdk/gpx"
	"github.com/open-wanderer/wanderer/plugins/sdk/polyline"
)

func TestUserIDFromJWT(t *testing.T) {
	token := "header.eyJzdWIiOiJ1c2VyLTEyMyJ9.signature"
	got, err := userIDFromJWT(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "user-123" {
		t.Fatalf("got %q", got)
	}
}

func TestUserIDFromJWTRejectsInvalidToken(t *testing.T) {
	if _, err := userIDFromJWT("not-a-jwt"); err == nil {
		t.Fatal("expected error")
	}
}

func TestDecodePolyline(t *testing.T) {
	points, err := polyline.Decode("_p~iF~ps|U_ulLnnqC_mqNvxq`@", 1e5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(points) != 3 {
		t.Fatalf("expected 3 points, got %d", len(points))
	}
	if points[0][0] != 38.5 || points[0][1] != -120.2 {
		t.Fatalf("unexpected first point: %#v", points[0])
	}
}

func TestDecodePolylineNormalizesOutOfRangeScale(t *testing.T) {
	points, err := polyline.Decode("_p~iF~ps|U", 1e5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	points[0][0] *= 10
	points[0][1] *= 10
	polyline.NormalizeCoordinateScale(points)
	if points[0][0] != 38.5 || points[0][1] != -120.2 {
		t.Fatalf("expected normalized point, got %#v", points[0])
	}
}

func TestShouldSwapCoordinates(t *testing.T) {
	coords := [][2]float64{{120.2, 38.5}, {121.0, 39.0}}
	if !polyline.ShouldSwapCoordinates(coords) {
		t.Fatal("expected coordinates to be detected as swapped")
	}
}

func TestProportionalIndex(t *testing.T) {
	if got := polyline.ProportionalIndex(2, 5, 3); got != 1 {
		t.Fatalf("got %d, want 1", got)
	}
	if got := polyline.ProportionalIndex(4, 5, 3); got != 2 {
		t.Fatalf("got %d, want 2", got)
	}
}

func TestGPXBytesEscapesTrackName(t *testing.T) {
	data, err := sdkgpx.Track("wanderer Hammerhead plugin", "A & B", []sdkgpx.Point{{Lat: 46.1, Lon: 8.2}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gpx := string(data)
	if !strings.Contains(gpx, "<name>A &amp; B</name>") {
		t.Fatalf("expected escaped name, got %s", gpx)
	}
	if !strings.Contains(gpx, `lat="46.10000000" lon="8.20000000"`) {
		t.Fatalf("expected track point, got %s", gpx)
	}
}

func TestTrailGPXFilename(t *testing.T) {
	tests := map[string]string{
		"":                    "trail.gpx",
		"My Route":            "My Route.gpx",
		"My Route.gpx":        "My Route.gpx",
		"../Bad/Route\\Name ": "Bad-Route-Name.gpx",
	}

	for input, want := range tests {
		if got := trailGPXFilename(input); got != want {
			t.Fatalf("trailGPXFilename(%q) = %q, want %q", input, got, want)
		}
	}
}
