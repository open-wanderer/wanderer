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
			if got, want := acc.total(), naiveLegsDuration(times[:i+1], anchors[:i+1]); got != want {
				t.Fatalf("sequence %d prefix %d: incremental %d, naive %d (times %v anchors %v)", n, i+1, got, want, times[:i+1], anchors[:i+1])
			}
		}
	}
}
