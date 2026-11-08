package gpx

import "math"

// The functions in this file are a line-for-line port of
// web/src/lib/models/gpx/utils.ts and gpx-metrics-computation.ts. Their
// results must agree with TypeScript and Dart bit for bit, so the evaluation
// order is deliberate and must not be "improved".

const (
	// thresholdXY and thresholdZ are GpxMetricsComputation(5, 5) in gpx.ts.
	thresholdXY = 5.0
	thresholdZ  = 5.0
)

func clipLat(v float64) float64 { return math.Min(math.Max(v, -90), 90) }

// wrapLon maps a longitude into [-180, 180]. math.Mod takes the sign of the
// dividend, like JavaScript's %.
func wrapLon(v float64) float64 {
	if v >= -180 && v <= 180 {
		return v
	}
	r := math.Mod(v+180, 360)
	if r < 0 {
		r += 360
	}
	return r - 180
}

// haversine is the distance in metres between two positions. Latitude is
// clipped and longitude wrapped here and nowhere else: bounds, centroid and
// start keep the file's values (corpus fixture 16).
//
// Pitfall 1: Go may fuse x*y + z into one fused multiply-add on arm64, which
// rounds once where JavaScript rounds twice. Every product that feeds a sum is
// wrapped in float64(), which the spec defines as a rounding barrier, so the
// result matches JS on amd64 and on the arm64 production hosts.
//
// Pitfall 2: the scale is R*c rounded first and then times 1000, with R in
// kilometres, not 6371000*c, which differs in the last bit.
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	lat1, lat2 = clipLat(lat1), clipLat(lat2)
	lon1, lon2 = wrapLon(lon1), wrapLon(lon2)
	const R = 6371.0
	dLat := (lat2 - lat1) * (math.Pi / 180)
	dLon := (lon2 - lon1) * (math.Pi / 180)
	a := float64(math.Sin(dLat/2)*math.Sin(dLat/2)) +
		float64(float64(float64(math.Cos(lat1*(math.Pi/180))*math.Cos(lat2*(math.Pi/180)))*math.Sin(dLon/2))*math.Sin(dLon/2))
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return float64(R*c) * 1000
}

func isFinite(v float64) bool { return !math.IsInf(v, 0) && !math.IsNaN(v) }

// meter accumulates distance and the filtered elevation totals over the
// points of the measured source, with no reset between segments or tracks.
//
// Elevation uses the defer-then-publish noise filter: an above-threshold move
// is held in a single pending slot and discarded if the track returns to the
// pre-excursion elevation without moving horizontally. The raw and smoothed
// distance anchors of the TypeScript class are not reported and are dropped.
type meter struct {
	started        bool
	lastLat        float64
	lastLon        float64
	distance       float64
	lastFilteredZ  float64
	hasFilteredZ   bool
	zLat, zLon     float64 // position where lastFilteredZ was last set
	hasZPoint      bool
	pendingDelta   float64
	pendingAnchorZ float64
	hasPending     bool
	gainSm, lossSm float64
}

// add feeds one usable point. ele is NaN when the point has no usable elevation.
func (m *meter) add(lat, lon, ele float64) {
	hasEle := !math.IsNaN(ele)
	if !m.started {
		// The first point initialises the anchors and adds no distance. When it
		// has no elevation both anchors stay unset, so the first point that
		// does carry one becomes the anchor instead of diffing against 0.
		m.started = true
		m.lastLat, m.lastLon = lat, lon
		if hasEle {
			m.lastFilteredZ, m.hasFilteredZ = ele, true
			m.zLat, m.zLon, m.hasZPoint = lat, lon, true
		}
		return
	}

	if d := haversine(m.lastLat, m.lastLon, lat, lon); isFinite(d) {
		m.distance += d
	}
	m.lastLat, m.lastLon = lat, lon

	if !hasEle {
		return
	}
	if !m.hasFilteredZ {
		m.lastFilteredZ, m.hasFilteredZ = ele, true
		m.zLat, m.zLon, m.hasZPoint = lat, lon, true
		m.pendingDelta, m.hasPending = 0, false
		return
	}

	diff := ele - m.lastFilteredZ
	if math.Abs(diff) < thresholdZ {
		return
	}
	returnDistance := math.Inf(1)
	if m.hasZPoint {
		returnDistance = haversine(m.zLat, m.zLon, lat, lon)
	}
	// diff is never zero here (|diff| >= 5), so comparing signs is comparing
	// whether exactly one of the two is positive.
	cancels := m.pendingDelta != 0 && m.hasPending &&
		(diff > 0) != (m.pendingDelta > 0) &&
		math.Abs(ele-m.pendingAnchorZ) < thresholdZ &&
		isFinite(returnDistance) && returnDistance < thresholdXY
	if cancels {
		m.lastFilteredZ = m.pendingAnchorZ
		m.zLat, m.zLon, m.hasZPoint = lat, lon, true
		m.pendingDelta, m.hasPending = 0, false
		return
	}
	// This move confirms whatever was pending, then becomes the pending move.
	if m.pendingDelta > 0 {
		m.gainSm += m.pendingDelta
	} else if m.pendingDelta < 0 {
		m.lossSm -= m.pendingDelta
	}
	m.pendingAnchorZ, m.hasPending = m.lastFilteredZ, true
	m.pendingDelta = diff
	m.lastFilteredZ = ele
	m.zLat, m.zLon, m.hasZPoint = lat, lon, true
}

// final returns the reported pair: the published totals plus a still-pending
// excursion, which a track that ends mid-climb has not yet confirmed. It is
// never the monotonic smoothed pair alone (corpus fixture 04: 88, not 80).
func (m *meter) final() (gain, loss float64) {
	return m.gainSm + math.Max(m.pendingDelta, 0), m.lossSm + math.Max(-m.pendingDelta, 0)
}

// measurement is the single-pass result over a measured source.
type measurement struct {
	distance, gain, loss float64
	count                int
	bounds               Bounds
	centroid             LatLon
	hasElevation         bool
}

// measure walks every stored point of tracks once, in file order, through one
// shared meter. Bounds and centroid run over the stored file values. The
// caller guarantees at least one point.
func measure(tracks []measuredTrack) measurement {
	var m meter
	var sumLat, sumLon float64
	b := Bounds{MinLat: math.Inf(1), MaxLat: math.Inf(-1), MinLon: math.Inf(1), MaxLon: math.Inf(-1)}
	count := 0
	hasEle := false
	for ti := range tracks {
		for si := range tracks[ti].segments {
			seg := &tracks[ti].segments[si]
			for i := range seg.lat {
				lat, lon, ele := seg.lat[i], seg.lon[i], seg.ele[i]
				m.add(lat, lon, ele)
				if !math.IsNaN(ele) {
					hasEle = true
				}
				sumLat += lat
				sumLon += lon
				count++
				b.MinLat = math.Min(b.MinLat, lat)
				b.MaxLat = math.Max(b.MaxLat, lat)
				b.MinLon = math.Min(b.MinLon, lon)
				b.MaxLon = math.Max(b.MaxLon, lon)
			}
		}
	}
	gain, loss := m.final()
	return measurement{
		distance:     m.distance,
		gain:         gain,
		loss:         loss,
		count:        count,
		bounds:       b,
		centroid:     LatLon{Lat: sumLat / float64(count), Lon: sumLon / float64(count)},
		hasElevation: hasEle,
	}
}

// legsAccumulator is the incremental form of legsDurationMs in
// web/src/lib/models/gpx/track-segment.ts.
//
// Stub: the real accumulator follows in the implementation commit.
type legsAccumulator struct{}

func (l *legsAccumulator) add(timeMs int64, anchor bool) {}

func (l *legsAccumulator) total() int64 { return 0 }
