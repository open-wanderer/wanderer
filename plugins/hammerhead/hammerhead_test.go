package main

import (
	"strings"
	"testing"
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
	points, err := decodePolyline("_p~iF~ps|U_ulLnnqC_mqNvxq`@")
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

func TestGPXBytesEscapesTrackName(t *testing.T) {
	data, err := gpxBytes("A & B", []gpxPoint{{Lat: 46.1, Lon: 8.2}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gpx := string(data)
	if !strings.Contains(gpx, "<name>A &amp; B</name>") {
		t.Fatalf("expected escaped name, got %s", gpx)
	}
	if !strings.Contains(gpx, `lat="46.1000000" lon="8.2000000"`) {
		t.Fatalf("expected track point, got %s", gpx)
	}
}
