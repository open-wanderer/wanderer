package gpx

import (
	"bytes"
	"testing"
)

// deref maps the corpus's JSON null onto the empty string, which is how Go
// spells "no description", "no icon" and "no date". Go has no "today" default
// for the date, so a null expected date requires an empty StartDate.
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// TestCorpusTrail checks every fixture's trail block: the fields a trail
// record needs beyond the metrics, derived from the same measured source.
func TestCorpusTrail(t *testing.T) {
	for _, f := range loadCorpus(t) {
		t.Run(f.dir, func(t *testing.T) {
			res, err := Parse(bytes.NewReader(f.input), parseLimit, "")
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			want := f.expected.Trail
			if res.Name != want.Name {
				t.Errorf("name: got %.80q, want %.80q", res.Name, want.Name)
			}
			if got, w := res.Description, deref(want.Description); got != w {
				t.Errorf("description: got %.80q, want %.80q", got, w)
			}
			assertCorpusMatch(t, "lat", res.Start.Lat, want.Lat, 0)
			assertCorpusMatch(t, "lon", res.Start.Lon, want.Lon, 0)
			if got, w := res.StartDate, deref(want.Date); got != w {
				t.Errorf("date: got %.80q, want %.80q", got, w)
			}
			assertCorpusMatch(t, "distance", res.Distance, want.Distance, metreTolerance)
			assertCorpusMatch(t, "elevationGain", res.ElevationGain, want.ElevationGain, metreTolerance)
			assertCorpusMatch(t, "elevationLoss", res.ElevationLoss, want.ElevationLoss, metreTolerance)
			assertCorpusMatch(t, "duration", float64(res.DurationMs)/1000, want.Duration, 0)
			assertCorpusMatch(t, "minLat", res.Bounds.MinLat, want.MinLat, 0)
			assertCorpusMatch(t, "maxLat", res.Bounds.MaxLat, want.MaxLat, 0)
			assertCorpusMatch(t, "minLon", res.Bounds.MinLon, want.MinLon, 0)
			assertCorpusMatch(t, "maxLon", res.Bounds.MaxLon, want.MaxLon, 0)

			if len(res.Waypoints) != len(want.Waypoints) {
				t.Fatalf("waypoints: got %d, want %d", len(res.Waypoints), len(want.Waypoints))
			}
			for i, w := range want.Waypoints {
				got := res.Waypoints[i]
				assertCorpusMatch(t, "waypoint lat", got.Lat, w.Lat, 0)
				assertCorpusMatch(t, "waypoint lon", got.Lon, w.Lon, 0)
				if got.Name != w.Name {
					t.Errorf("waypoint %d name: got %.80q, want %.80q", i, got.Name, w.Name)
				}
				if g, ww := got.Description, deref(w.Description); g != ww {
					t.Errorf("waypoint %d description: got %.80q, want %.80q", i, g, ww)
				}
				if g, ww := got.Icon, deref(w.Icon); g != ww {
					t.Errorf("waypoint %d icon: got %.80q, want %.80q", i, g, ww)
				}
			}
		})
	}
}
