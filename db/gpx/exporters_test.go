package gpx

// The files under testdata/exporters are hand-written samples shaped like what
// real exporters emit (Garmin Connect, GaiaGPS, a legacy windows-1252 desktop
// tool, and a grab bag of defects seen from Locus-style apps). They are never
// real user files and hold no real recordings. The go tool ignores testdata/
// when it builds, so nothing here is compiled or shipped; db/gpx/testdata is
// the first testdata directory in db/.
//
// windows-1252-latin.gpx contains single raw bytes (0x93, 0xFC, 0xE9, 0x94,
// 0xB0), not UTF-8 sequences. Editors that "fix" the encoding on save break it,
// which is why the tests assert its exact decoded text.

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readSample(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "exporters", name))
	if err != nil {
		t.Fatalf("sample %s: %v", name, err)
	}
	return b
}

func parseSample(t *testing.T, name string, data []byte) *Result {
	t.Helper()
	res, err := Parse(bytes.NewReader(data), 25<<20, name)
	if err != nil {
		t.Fatalf("Parse(%s): %v", name, err)
	}
	return res
}

func TestExporterSamples(t *testing.T) {
	t.Run("garmin-extensions.gpx", func(t *testing.T) {
		raw := readSample(t, "garmin-extensions.gpx")
		res := parseSample(t, "garmin-extensions.gpx", raw)

		// <link> comes first in <metadata> and has no HTML void-element
		// treatment, so the track name after it survives.
		if res.Name != "Morning Run" {
			t.Errorf("Name = %q, want %q", res.Name, "Morning Run")
		}
		if res.PointCount != 6 {
			t.Errorf("PointCount = %d, want 6", res.PointCount)
		}
		// 06:00:00 to 06:02:30. The vendor <time> of 2099 inside <extensions>
		// must not be read.
		if res.DurationMs != 150_000 {
			t.Errorf("DurationMs = %d, want 150000", res.DurationMs)
		}
		if !res.HasElevation {
			t.Errorf("HasElevation = false, want true (5 of 6 points carry an elevation)")
		}
		if res.ElevationGain <= 0 || res.ElevationGain > 20 {
			t.Errorf("ElevationGain = %v, want a small positive value (the vendor 9999 must not count)", res.ElevationGain)
		}

		// The same file without the vendor elements parses to the same result:
		// extensions are skipped, not read.
		stripped := bytes.Replace(raw, []byte("<ele>9999</ele>"), nil, 1)
		stripped = bytes.Replace(stripped, []byte("<time>2099-01-01T00:00:00Z</time>"), nil, 1)
		if len(stripped) == len(raw) {
			t.Fatal("the vendor elements are missing from the sample")
		}
		twin := parseSample(t, "garmin-without-vendor", stripped)
		if res.ElevationGain != twin.ElevationGain || res.ElevationLoss != twin.ElevationLoss ||
			res.DurationMs != twin.DurationMs || res.Distance != twin.Distance {
			t.Errorf("vendor elements changed the result: %+v vs %+v", res, twin)
		}
	})

	t.Run("gaiagps-route-only.gpx", func(t *testing.T) {
		res := parseSample(t, "gaiagps-route-only.gpx", readSample(t, "gaiagps-route-only.gpx"))

		// Route promotion: a route-only file is measured like a track.
		if res.PointCount != 8 {
			t.Errorf("PointCount = %d, want 8", res.PointCount)
		}
		if res.Distance <= 0 {
			t.Errorf("Distance = %v, want greater than 0", res.Distance)
		}
		poly, err := res.Polyline()
		if err != nil || poly == "" {
			t.Errorf("Polyline() = (%q, %v), want a non-empty polyline", poly, err)
		}
		if res.Name != "Hochplateau Traverse" {
			t.Errorf("Name = %q, want the route's name", res.Name)
		}
		if res.DurationMs != 0 {
			t.Errorf("DurationMs = %d, want 0 for a route without times", res.DurationMs)
		}
		if len(res.Waypoints) != 2 {
			t.Fatalf("%d waypoints, want 2", len(res.Waypoints))
		}
		if got := res.Waypoints[0].Icon; got != "campground" {
			t.Errorf("Waypoints[0].Icon = %q, want %q", got, "campground")
		}
		if got := res.Waypoints[1].Icon; got != "" {
			t.Errorf("Waypoints[1].Icon = %q, want empty for an unknown sym", got)
		}
	})

	t.Run("windows-1252-latin.gpx", func(t *testing.T) {
		raw := readSample(t, "windows-1252-latin.gpx")
		if bytes.Contains(raw, []byte("\xc3")) {
			t.Fatal("the sample holds UTF-8 sequences, so it no longer tests windows-1252 decoding")
		}
		res := parseSample(t, "windows-1252-latin.gpx", raw)

		if want := "\u201cGr\u00fcn\u00e9\u201d"; res.Name != want {
			t.Errorf("Name = %q, want %q", res.Name, want)
		}
		if want := "Steep 35\u00b0 incline"; res.Description != want {
			t.Errorf("Description = %q, want %q", res.Description, want)
		}
		if res.PointCount != 3 {
			t.Errorf("PointCount = %d, want 3", res.PointCount)
		}
	})

	t.Run("tolerant-fields.gpx", func(t *testing.T) {
		res := parseSample(t, "tolerant-fields.gpx", readSample(t, "tolerant-fields.gpx"))

		// Valid points: the first four, then the last two. The five points with
		// a bad or missing lat or lon are dropped.
		if res.PointCount != 6 {
			t.Errorf("PointCount = %d, want 6", res.PointCount)
		}
		// No point was read as 0,0 or as a half-read position.
		b := res.Bounds
		if b.MinLat <= 46 || b.MinLon <= 10 || b.MaxLat >= 48 || b.MaxLon >= 12 {
			t.Errorf("Bounds = %+v, want every point near 47 N, 11 E", b)
		}
		// The elevations "", "&nbsp;" and "N/A" count as missing.
		ele := res.tracks[0].segments[0].ele
		missing := 0
		for _, e := range ele {
			if math.IsNaN(e) {
				missing++
			}
		}
		if len(ele) != 6 || missing != 3 {
			t.Errorf("ele = %v, want 6 points with 3 missing", ele)
		}
		// 06:00:00 to 06:02:30: the "N/A" time is no time and the points
		// around it still count.
		if res.DurationMs != 150_000 {
			t.Errorf("DurationMs = %d, want 150000", res.DurationMs)
		}
		if !strings.Contains(res.Name, "\u00a0") || !strings.Contains(res.Name, "\u00e9") {
			t.Errorf("Name = %q, want it to hold U+00A0 and the e acute", res.Name)
		}
		if want := "Line one & <b>two</b>"; res.Description != want {
			t.Errorf("Description = %q, want the CDATA text %q verbatim", res.Description, want)
		}
	})
}
