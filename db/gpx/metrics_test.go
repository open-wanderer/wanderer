package gpx

import (
	"math"
	"math/rand/v2"
	"testing"
)

func TestWrapLon(t *testing.T) {
	tests := []struct {
		in, want float64
	}{
		{190, -170},
		{-190, 170},
		{540, -180},
		{360, 0},
		{720, 0},
		{181, -179},
		{-181, 179},
		{0, 0},
		{180, 180},
		{-180, -180},
		{11.001, 11.001},
	}
	for _, tc := range tests {
		if got := wrapLon(tc.in); got != tc.want {
			t.Errorf("wrapLon(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestClipLat(t *testing.T) {
	tests := []struct {
		in, want float64
	}{
		{95.5, 90},
		{-95.5, -90},
		{90, 90},
		{-90, -90},
		{47.001, 47.001},
	}
	for _, tc := range tests {
		if got := clipLat(tc.in); got != tc.want {
			t.Errorf("clipLat(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestHaversine(t *testing.T) {
	t.Run("fixture 01 hop", func(t *testing.T) {
		got := haversine(47.0, 11.0, 47.001, 11.001)
		if want := 134.59240148587796; math.Abs(got-want) > metreTolerance {
			t.Errorf("haversine = %v, want %v", got, want)
		}
	})
	t.Run("latitude is clipped inside the distance", func(t *testing.T) {
		if got, want := haversine(89, 0, 95.5, 0), haversine(89, 0, 90, 0); got != want {
			t.Errorf("clipped hop = %v, want %v", got, want)
		}
	})
	t.Run("longitude is wrapped inside the distance", func(t *testing.T) {
		if got, want := haversine(10, 179, 10, 190), haversine(10, 179, 10, -170); got != want {
			t.Errorf("wrapped hop = %v, want %v", got, want)
		}
	})
	t.Run("fixture 16 total", func(t *testing.T) {
		// DERIVATION.md of 16-out-of-range-coordinates: the sum of the three hops.
		var total float64
		total += haversine(89, 0, 95.5, 0)
		total += haversine(95.5, 0, 10, 179)
		total += haversine(10, 179, 10, 190)
		if want := 10211294.96071456; math.Abs(total-want) > metreTolerance {
			t.Errorf("fixture 16 distance = %v, want %v", total, want)
		}
	})
}

// naiveLegsDuration is a literal transcription of legsDurationMs in
// web/src/lib/models/gpx/track-segment.ts, over a list that keeps every point.
func naiveLegsDuration(times []int64, anchors []bool) int64 {
	var total int64
	legStart := 0
	addLeg := func(legEnd int) {
		start, end := times[legStart], times[legEnd]
		if legEnd > legStart && start != noTime && end != noTime {
			total += end - start
		}
	}
	for i := 1; i < len(times)-1; i++ {
		if anchors[i] {
			addLeg(i - 1)
			legStart = i
		}
	}
	if len(times) > 0 {
		addLeg(len(times) - 1)
	}
	return total
}

func TestLegsDurationDifferential(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	for n := 0; n < 10000; n++ {
		length := rng.IntN(13)
		times := make([]int64, length)
		anchors := make([]bool, length)
		var acc legsAccumulator
		for i := 0; i < length; i++ {
			times[i] = noTime
			if rng.IntN(5) != 0 { // a point lacks <time> one time in five
				times[i] = int64(rng.IntN(2001)-500) * 1000 // out-of-order times give negative legs
			}
			anchors[i] = rng.IntN(10) < 3
			acc.add(times[i], anchors[i])
			// total() is a pure read: it must agree with the oracle after every prefix.
			got, ok := acc.total()
			if want := naiveLegsDuration(times[:i+1], anchors[:i+1]); !ok || got != want {
				t.Fatalf("sequence %d prefix %d: incremental %d (ok %v), naive %d (times %v anchors %v)", n, i+1, got, ok, want, times[:i+1], anchors[:i+1])
			}
		}
	}
}

// TestDurationMinInt64 pins the one sum that cannot be negated: finish makes a
// negative duration positive, which math.MinInt64 has no representation for.
func TestDurationMinInt64(t *testing.T) {
	b := newBuilder()
	b.sawTrk = true
	b.tracks = []measuredTrack{{segments: []measuredSegment{{
		lat:    []float64{47},
		lon:    []float64{11},
		ele:    []float64{math.NaN()},
		timeMs: []int64{noTime},
	}}}}
	b.trkDuration = math.MinInt64
	res, err := b.finish("")
	wantErr(t, res, err, ErrMalformed)
}

func TestAddDuration(t *testing.T) {
	tests := []struct {
		a, b   int64
		want   int64
		wantOK bool
	}{
		{5, -3, 2, true},
		{0, 0, 0, true},
		{math.MaxInt64, 0, math.MaxInt64, true},
		{math.MinInt64, 0, math.MinInt64, true},
		{math.MaxInt64, math.MinInt64, -1, true},
		{math.MaxInt64, 1, 0, false},
		{math.MinInt64, -1, 0, false},
		{math.MinInt64, math.MinInt64, 0, false},
		{math.MaxInt64, math.MaxInt64, 0, false},
	}
	for _, tc := range tests {
		got, ok := addDuration(tc.a, tc.b)
		if ok != tc.wantOK || (ok && got != tc.want) {
			t.Errorf("addDuration(%d, %d) = (%d, %v), want (%d, %v)", tc.a, tc.b, got, ok, tc.want, tc.wantOK)
		}
	}
}

// TestLegsAccumulatorOverflow feeds extreme anchored legs, one 17,280,000,000,000,000 ms
// leg per pair. 533 pairs sum to 9210240000000000000 and fit, a 534th does not.
func TestLegsAccumulatorOverflow(t *testing.T) {
	const (
		earliest = int64(-8_640_000_000_000_000)
		latest   = int64(8_640_000_000_000_000)
		perLeg   = latest - earliest
	)
	var acc legsAccumulator
	for pair := 1; pair <= 534; pair++ {
		acc.add(earliest, pair > 1)
		acc.add(latest, false)
		got, ok := acc.total()
		if pair <= 533 {
			if !ok || got != int64(pair)*perLeg {
				t.Fatalf("after %d pairs total = (%d, %v), want (%d, true)", pair, got, ok, int64(pair)*perLeg)
			}
			continue
		}
		if ok {
			t.Fatalf("after %d pairs total = (%d, true), want overflow", pair, got)
		}
	}
	// The 534th leg is still open, so total() reports it without latching
	// anything. Closing it with an anchor sets the flag for good, and later
	// points cannot clear it.
	acc.add(earliest, true)
	acc.add(earliest, false)
	for i := 0; i < 3; i++ {
		if _, ok := acc.total(); ok {
			t.Fatalf("overflow flag was cleared after %d further points", i)
		}
		acc.add(earliest, false)
	}
}
