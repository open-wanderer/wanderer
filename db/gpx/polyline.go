package gpx

import (
	"fmt"
	"math"

	"github.com/twpayne/go-polyline"
)

// This file is an exact port of the simplification that gpxgo v1.4.0 does for
// SimplifyTracks(50), so the polyline recomputed here is byte-identical to the
// one stored today. gpxgo itself is imported only by the golden test.
//
// Unlike metrics.go, the RDP expressions below are deliberately textually
// identical to gpxgo and carry no explicit rounding barriers: the barriers in
// metrics.go exist to match JavaScript bit for bit, whereas here the target is
// gpxgo compiled by the same Go compiler, and the same expressions compile to
// the same instructions (fused or not) on every architecture.

const oneDegree = 1000.0 * 10000.8 / 90.0
const earthRadius = 6371 * 1000

// polylineTolerance is the simplification distance in metres that the stored
// polyline has always used.
const polylineTolerance = 50

// rdpWorkBudget is the most points the simplification may scan before
// Polyline gives up with ErrTooComplex (A2). The recursion is O(n^2) in the
// worst case, which an adversarial zigzag can reach; the budget is about two
// seconds of work. It is a variable only so a test can lower it.
var rdpWorkBudget int64 = 2_000_000_000

func toRad(x float64) float64 {
	return x / 180. * math.Pi
}

// rdpHaversine is gpxgo's HaversineDistance.
func rdpHaversine(lat1, lon1, lat2, lon2 float64) float64 {
	dLat := toRad(lat1 - lat2)
	dLon := toRad(lon1 - lon2)
	thisLat1 := toRad(lat1)
	thisLat2 := toRad(lat2)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(thisLat1)*math.Cos(thisLat2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	d := earthRadius * c

	return d
}

// rdpDistance2D is gpxgo's distance with threeD and haversine both false.
func rdpDistance2D(lat1, lon1, lat2, lon2 float64) float64 {
	absLat := math.Abs(lat1 - lat2)
	absLon := math.Abs(lon1 - lon2)
	if absLat > 0.2 || absLon > 0.2 {
		return rdpHaversine(lat1, lon1, lat2, lon2)
	}

	coef := math.Cos(toRad(lat1))
	x := lat1 - lat2
	y := (lon1 - lon2) * coef

	distance2d := math.Sqrt(x*x+y*y) * oneDegree

	return distance2d
}

// distanceFromLine is gpxgo's: Heron's formula for the height over the base
// p1-p2, with the zero-length-base shortcut. The argument order of the three
// distances (p1 to p2, p1 to point, p2 to point) matters because the distance
// takes its cosine from its first latitude.
func distanceFromLine(lat, lon, lat1, lon1, lat2, lon2 float64) float64 {
	a := rdpDistance2D(lat1, lon1, lat2, lon2)

	if a == 0 {
		return rdpDistance2D(lat1, lon1, lat, lon)
	}

	b := rdpDistance2D(lat1, lon1, lat, lon)
	c := rdpDistance2D(lat2, lon2, lat, lon)

	s := (a + b + c) / 2.

	return 2.0 * math.Sqrt(math.Abs(s*(s-a)*(s-b)*(s-c))) / a
}

// lineEquationCoefficients is gpxgo's getLineEquationCoefficients: a x + b y + c = 0
// through two points in (lat, lon) as (x, y).
func lineEquationCoefficients(lat1, lon1, lat2, lon2 float64) (float64, float64, float64) {
	if lon1 == lon2 {
		// Vertical line:
		return 0.0, 1.0, -lon1
	}
	a := (lat1 - lat2) / (lon1 - lon2)
	b := lat1 - lon1*a
	return 1.0, -a, -b
}

// simplify runs gpxgo's simplifyPoints over one segment and returns the kept
// point indices in output order. It is the explicit-stack form of the
// recursion, which emits points[:pos] simplified, then points[pos], then
// points[pos+1:] simplified, so a long shrinking zigzag cannot exhaust the
// goroutine stack. Every scanned span is charged against budget.
func simplify(lat, lon []float64, maxDistance float64, budget *int64) ([]int, error) {
	type span struct {
		lo, hi int
		emit   bool // emit index lo instead of scanning
	}
	var out []int
	stack := []span{{lo: 0, hi: len(lat)}}
	for len(stack) > 0 {
		s := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if s.emit {
			out = append(out, s.lo)
			continue
		}
		n := s.hi - s.lo
		if n < 3 {
			for i := s.lo; i < s.hi; i++ {
				out = append(out, i)
			}
			continue
		}
		*budget -= int64(n)
		if *budget < 0 {
			return nil, fmt.Errorf("%w: work budget exceeded", ErrTooComplex)
		}

		b, e := s.lo, s.hi-1
		a, bb, c := lineEquationCoefficients(lat[b], lon[b], lat[e], lon[e])

		tmpMaxDistance := -1000000000.0
		tmpMaxDistancePosition := 0
		for pointNo := 0; pointNo < n; pointNo++ {
			d := math.Abs(a*lat[s.lo+pointNo] + bb*lon[s.lo+pointNo] + c)
			if d > tmpMaxDistance {
				tmpMaxDistance = d
				tmpMaxDistancePosition = pointNo
			}
		}
		pos := s.lo + tmpMaxDistancePosition

		realMaxDistance := distanceFromLine(lat[pos], lon[pos], lat[b], lon[b], lat[e], lon[e])
		if realMaxDistance < maxDistance {
			out = append(out, b, e)
			continue
		}
		// Pushed last-to-first so the left span is processed first.
		stack = append(stack, span{lo: pos + 1, hi: s.hi}, span{lo: pos, emit: true}, span{lo: s.lo, hi: pos})
	}
	return out, nil
}

// Polyline is the encoded, simplified track: every segment of the measured
// source is simplified on its own at 50 m (D-07) and the kept points are
// concatenated, in order, as one encoded string of [lat, lon] pairs at 1e5
// precision. The measured source already holds a route-only file's promoted
// routes and leaves a mixed file's routes out (D-04, D-06). The caller decides
// what to do with a polyline over its own length limit.
//
// It returns ErrTooComplex when the geometry would need more work than the
// budget allows. Parse and the metrics are unaffected by that.
func (r *Result) Polyline() (string, error) {
	budget := rdpWorkBudget
	coords := make([][]float64, 0)
	for ti := range r.tracks {
		for si := range r.tracks[ti].segments {
			seg := &r.tracks[ti].segments[si]
			kept, err := simplify(seg.lat, seg.lon, polylineTolerance, &budget)
			if err != nil {
				return "", err
			}
			for _, i := range kept {
				coords = append(coords, []float64{seg.lat[i], seg.lon[i]})
			}
		}
	}
	return string(polyline.EncodeCoords(coords)), nil
}
