package gpx

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/twpayne/go-polyline"
)

// fixtureResult parses one corpus fixture by directory name.
func fixtureResult(t *testing.T, dir string) (*Result, []byte) {
	t.Helper()
	input, err := os.ReadFile(filepath.Join(corpusRoot, dir, "input.gpx"))
	if err != nil {
		t.Fatalf("fixture %s: %v", dir, err)
	}
	res, err := Parse(bytes.NewReader(input), parseLimit, "")
	if err != nil {
		t.Fatalf("Parse(%s): %v", dir, err)
	}
	return res, input
}

func decodePolyline(t *testing.T, s string) [][]float64 {
	t.Helper()
	coords, rest, err := polyline.DecodeCoords([]byte(s))
	if err != nil || len(rest) != 0 {
		t.Fatalf("DecodeCoords(%.80q): rest %d, err %v", s, len(rest), err)
	}
	return coords
}

// round5 is the polyline's own precision.
func round5(v float64) float64 { return math.Round(v*1e5) / 1e5 }

func TestPolylineRouteOnly(t *testing.T) {
	res, _ := fixtureResult(t, "17-route-only-promotion")
	got, err := res.Polyline()
	if err != nil {
		t.Fatalf("Polyline: %v", err)
	}
	if got == "" {
		t.Fatalf("route-only file got an empty polyline, want its promoted routes (D-06)")
	}
	// Derived by hand from the fixture: the positionless first rtept is dropped,
	// each of the two routes is collinear so it keeps its two ends, and the
	// routes are simplified one at a time and concatenated (D-07).
	want := [][]float64{{47.1, 11.2}, {47.102, 11.2}, {47.102, 11.2015}, {47.1, 11.2015}}
	coords := decodePolyline(t, got)
	if len(coords) != len(want) {
		t.Fatalf("decoded %d points %v, want %d %v", len(coords), coords, len(want), want)
	}
	for i := range want {
		if round5(coords[i][0]) != want[i][0] || round5(coords[i][1]) != want[i][1] {
			t.Fatalf("point %d = %v, want %v", i, coords[i], want[i])
		}
	}
}

func TestPolylineMixed(t *testing.T) {
	res, _ := fixtureResult(t, "18-mixed-rte-trk")
	got, err := res.Polyline()
	if err != nil {
		t.Fatalf("Polyline: %v", err)
	}
	coords := decodePolyline(t, got)
	if len(coords) == 0 {
		t.Fatalf("mixed file got an empty polyline")
	}
	for i, c := range coords {
		// The route is in Sydney (lat -33.85), the track in Tyrol (lat 47.2).
		if c[0] < 47 {
			t.Fatalf("point %d = %v comes from the <rte>, want <trk> points only (D-04)", i, c)
		}
	}
}

func TestPolylineMultiSegment(t *testing.T) {
	res, _ := fixtureResult(t, "09-multi-segment-planner-route")
	var segs []measuredSegment
	for _, tr := range res.tracks {
		segs = append(segs, tr.segments...)
	}
	if len(segs) < 2 {
		t.Fatalf("fixture 09 has %d segments, want at least 2", len(segs))
	}
	whole, err := res.Polyline()
	if err != nil {
		t.Fatalf("Polyline: %v", err)
	}
	var want [][]float64
	for i, seg := range segs {
		one := &Result{tracks: []measuredTrack{{segments: []measuredSegment{seg}}}}
		s, err := one.Polyline()
		if err != nil {
			t.Fatalf("segment %d: %v", i, err)
		}
		want = append(want, decodePolyline(t, s)...)
	}
	got := decodePolyline(t, whole)
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("whole polyline is not the per-segment concatenation:\n got %v\nwant %v", got, want)
	}
}

// zigzag is a track whose amplitude shrinks along it, so every level of the
// simplification peels off about one point: the worst case for the recursion.
func zigzag(n int) string {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0"?><gpx version="1.1"><trk><trkseg>`)
	for i := 0; i < n; i++ {
		side := 1.0
		if i%2 == 1 {
			side = -1
		}
		amp := 1.5e-3 * (1 - 0.5*float64(i)/float64(n))
		sb.WriteString(`<trkpt lat="` + strconv.FormatFloat(47+float64(i)*1e-4, 'f', -1, 64) +
			`" lon="` + strconv.FormatFloat(11+side*amp, 'f', -1, 64) + `"/>`)
	}
	sb.WriteString(`</trkseg></trk></gpx>`)
	return sb.String()
}

func TestPolylineTooComplex(t *testing.T) {
	doc := zigzag(2000)
	res := mustParse(t, doc, "")
	if _, err := res.Polyline(); err != nil {
		t.Fatalf("Polyline with the default budget: %v", err)
	}

	old := rdpWorkBudget
	t.Cleanup(func() { rdpWorkBudget = old })
	rdpWorkBudget = 10_000

	got, err := res.Polyline()
	if !errors.Is(err, ErrTooComplex) {
		t.Fatalf("Polyline with a lowered budget: err = %v, want ErrTooComplex", err)
	}
	if got != "" {
		t.Fatalf("Polyline returned %.40q alongside ErrTooComplex, want nothing", got)
	}
	// Parse and the metrics are unaffected by the budget.
	again := mustParse(t, doc, "")
	if !(again.Distance > 0) || again.PointCount != 2000 {
		t.Fatalf("Parse under the lowered budget: distance %v, points %d", again.Distance, again.PointCount)
	}
}
