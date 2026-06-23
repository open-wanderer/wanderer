package gpx

import (
	"strings"
	"testing"
	"time"
)

func TestTrackEscapesFields(t *testing.T) {
	elevation := 123.456
	timestamp := time.Date(2026, 6, 1, 10, 30, 0, 0, time.UTC)
	data, err := Track("creator & test", "A & B", []Point{{
		Lat:       46.1,
		Lon:       8.2,
		Elevation: &elevation,
		Time:      &timestamp,
	}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gpx := string(data)
	for _, want := range []string{
		`creator="creator &amp; test"`,
		"<name>A &amp; B</name>",
		`lat="46.10000000" lon="8.20000000"`,
		"<ele>123.46</ele>",
		"<time>2026-06-01T10:30:00Z</time>",
	} {
		if !strings.Contains(gpx, want) {
			t.Fatalf("expected %q in %s", want, gpx)
		}
	}
}

func TestTrackRejectsEmptyPoints(t *testing.T) {
	if _, err := Track("", "empty", nil); err == nil {
		t.Fatal("expected error")
	}
}
