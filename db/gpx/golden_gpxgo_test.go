package gpx

import (
	"bytes"
	"fmt"
	"math"
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"

	gpxgo "github.com/tkrajina/gpxgo/gpx"
	"github.com/twpayne/go-polyline"
)

// This file pins Result.Polyline to the polyline db/util stores today, which
// gpxgo v1.4.0 produces. It is the only file that may import gpxgo, and it is
// deleted together with PARSE-08 (the removal of gpxgo). The alias is
// required because both packages are named gpx.

// encodeOld is the body of db/util/polyline.go that builds the stored polyline:
// parse with gpxgo, simplify every segment at 50 m, collect [lat, lon] over
// tracks and segments, encode.
func encodeOld(t *testing.T, data []byte) (string, error) {
	t.Helper()
	gpxData, err := gpxgo.ParseBytes(data)
	if err != nil {
		return "", err
	}
	gpxData.SimplifyTracks(50)
	coordinates := make([][]float64, 0)
	for _, trk := range gpxData.Tracks {
		for _, seg := range trk.Segments {
			for _, pt := range seg.Points {
				coordinates = append(coordinates, []float64{pt.Latitude, pt.Longitude})
			}
		}
	}
	return string(polyline.EncodeCoords(coordinates)), nil
}

// goldenSkips are the corpus fixtures where gpxgo and this package do not read
// the same point set, so byte identity is not claimed. Each entry is filled
// from an observed difference. A skipped fixture must really differ: an entry
// whose fixture matches fails the test, so a stale skip cannot hide a pass.
var goldenSkips = []struct{ dir, reason string }{}

func goldenSkip(dir string) (string, bool) {
	for _, s := range goldenSkips {
		if s.dir == dir {
			return s.reason, true
		}
	}
	return "", false
}

func TestPolylineGoldenCorpus(t *testing.T) {
	for _, f := range loadCorpus(t) {
		t.Run(f.dir, func(t *testing.T) {
			res, err := Parse(bytes.NewReader(f.input), parseLimit, "")
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			got, err := res.Polyline()
			if err != nil {
				t.Fatalf("Polyline: %v", err)
			}
			want, oldErr := encodeOld(t, f.input)
			if reason, skip := goldenSkip(f.dir); skip {
				if oldErr == nil && want == got {
					t.Fatalf("stale skip (%s): gpxgo and Polyline agree (%.60q), remove the entry", reason, got)
				}
				return
			}
			if oldErr != nil {
				t.Fatalf("gpxgo rejects a fixture that is not in the skip table: %v", oldErr)
			}
			if got != want {
				t.Fatalf("polyline differs from gpxgo:\n got %q\nwant %q", got, want)
			}
		})
	}
}

// goldenCases is the number of generated documents; the plan requires at least
// 1000.
const goldenCases = 1200

// genDoc writes one generated GPX document. The coordinates are formatted with
// the shortest round-tripping decimal, so gpxgo and Parse read the same values.
func genDoc(rng *rand.Rand, shape int) string {
	scales := []float64{1e-6, 1e-5, 1e-4, 1e-3, 1e-2, 1e-1}
	scale := scales[rng.IntN(len(scales))]
	baseLat := rng.Float64()*120 - 60
	baseLon := rng.Float64()*340 - 170

	walk := func(n int, step float64) [][2]float64 {
		pts := make([][2]float64, 0, n)
		lat, lon := baseLat, baseLon
		for i := 0; i < n; i++ {
			pts = append(pts, [2]float64{lat, lon})
			lat += (rng.Float64()*2 - 1) * step
			lon += (rng.Float64()*2 - 1) * step
		}
		return pts
	}

	var segs [][][2]float64
	switch shape {
	case 0: // random walk at a random step scale
		segs = [][][2]float64{walk(1+rng.IntN(400), scale)}
	case 1: // several segments
		for k := 2 + rng.IntN(3); k > 0; k-- {
			segs = append(segs, walk(1+rng.IntN(150), scale))
		}
	case 2: // vertical: constant longitude, with the odd off-line point
		n := 3 + rng.IntN(200)
		pts := make([][2]float64, 0, n)
		for i := 0; i < n; i++ {
			lon := baseLon
			if rng.IntN(8) == 0 && i != 0 && i != n-1 {
				lon += (rng.Float64()*2 - 1) * scale
			}
			pts = append(pts, [2]float64{baseLat + float64(i)*scale, lon})
		}
		segs = [][][2]float64{pts}
	case 3: // closed loop: first point equals last, so the base has length zero
		n := 4 + rng.IntN(200)
		radius := scale * float64(1+rng.IntN(50))
		pts := make([][2]float64, 0, n)
		for i := 0; i < n-1; i++ {
			a := 2 * math.Pi * float64(i) / float64(n-1)
			pts = append(pts, [2]float64{baseLat + radius*math.Sin(a), baseLon + radius*math.Cos(a)})
		}
		pts = append(pts, pts[0])
		segs = [][][2]float64{pts}
	case 4: // duplicate points
		raw := walk(2+rng.IntN(100), scale)
		var pts [][2]float64
		for _, p := range raw {
			for k := 1 + rng.IntN(3); k > 0; k-- {
				pts = append(pts, p)
			}
		}
		segs = [][][2]float64{pts}
	case 5: // spans above 0.2 degrees, gpxgo's haversine branch
		segs = [][][2]float64{walk(3+rng.IntN(60), 0.2+rng.Float64()*0.4)}
	default: // dense switchback, like fixture 12
		n := 20 + rng.IntN(300)
		pts := make([][2]float64, 0, n)
		for i := 0; i < n; i++ {
			side := 1.0
			if i%2 == 1 {
				side = -1
			}
			pts = append(pts, [2]float64{baseLat + float64(i)*0.0001, baseLon + side*0.0005*(1+rng.Float64()*0.2)})
		}
		segs = [][][2]float64{pts}
	}

	num := func(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?><gpx version="1.1" creator="golden"><trk>`)
	for _, seg := range segs {
		sb.WriteString("<trkseg>")
		for _, p := range seg {
			fmt.Fprintf(&sb, `<trkpt lat="%s" lon="%s"></trkpt>`, num(p[0]), num(p[1]))
		}
		sb.WriteString("</trkseg>")
	}
	sb.WriteString("</trk></gpx>")
	return sb.String()
}

func TestPolylineGoldenGenerated(t *testing.T) {
	const shapes = 7
	for c := 0; c < goldenCases; c++ {
		seed := uint64(c)*2654435761 + 12345
		rng := rand.New(rand.NewPCG(seed, seed))
		shape := c % shapes
		doc := genDoc(rng, shape)

		res, err := Parse(strings.NewReader(doc), parseLimit, "")
		if err != nil {
			t.Fatalf("case %d (seed %d, shape %d): Parse: %v", c, seed, shape, err)
		}
		got, err := res.Polyline()
		if err != nil {
			t.Fatalf("case %d (seed %d, shape %d): Polyline: %v", c, seed, shape, err)
		}
		want, err := encodeOld(t, []byte(doc))
		if err != nil {
			t.Fatalf("case %d (seed %d, shape %d): gpxgo: %v", c, seed, shape, err)
		}
		if got != want {
			t.Fatalf("case %d (seed %d, shape %d): polyline differs from gpxgo:\n got %.200q\nwant %.200q", c, seed, shape, got, want)
		}
	}
}
