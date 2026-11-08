package gpx

import (
	"fmt"
	"io"
	"math"
	"runtime"
	"runtime/debug"
	"sync"
	"testing"
	"time"
)

// startSampler records the peak live heap while it runs. A goroutine calls
// runtime.ReadMemStats every interval and keeps the largest HeapAlloc it saw;
// the returned stop function ends it and returns that peak. Peak live heap,
// not cumulative allocation, is what a memory budget is about (Pitfall 10), so
// nothing here or in the tests that use it reads a cumulative counter.
func startSampler(interval time.Duration) (stop func() uint64) {
	var (
		mu   sync.Mutex
		peak uint64
		done = make(chan struct{})
		quit = make(chan struct{})
	)
	sample := func() {
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		mu.Lock()
		if ms.HeapAlloc > peak {
			peak = ms.HeapAlloc
		}
		mu.Unlock()
	}
	go func() {
		defer close(done)
		tick := time.NewTicker(interval)
		defer tick.Stop()
		for {
			sample()
			select {
			case <-quit:
				return
			case <-tick.C:
			}
		}
	}()
	return func() uint64 {
		close(quit)
		<-done
		sample()
		mu.Lock()
		defer mu.Unlock()
		return peak
	}
}

// komootStream generates a Komoot-shaped GPX document on demand: a GPX 1.1
// header, one track and segment, then n single-line <trkpt> elements with a
// position, an elevation and a time, then the closing tags. The document is
// never held in memory; produced counts the bytes handed out so far.
type komootStream struct {
	n, i     int
	stage    int // 0 header, 1 points, 2 tail, 3 done
	pending  []byte
	produced int64
}

const (
	komootHeader = `<?xml version="1.0" encoding="UTF-8"?><gpx version="1.1" creator="komoot.de" ` +
		`xmlns="http://www.topografix.com/GPX/1/1"><metadata><name>Komoot Stream</name></metadata>` +
		`<trk><name>Komoot Stream</name><trkseg>`
	komootTail = `</trkseg></trk></gpx>`
)

// newKomootStream returns a stream of n points. Coordinates walk
// deterministically near 47 N, 11 E and every time is a valid 20-character
// RFC 3339 timestamp, one second after the previous point.
func newKomootStream(n int) *komootStream { return &komootStream{n: n} }

func (k *komootStream) Read(p []byte) (int, error) {
	total := 0
	for total < len(p) {
		if len(k.pending) == 0 {
			switch k.stage {
			case 0:
				k.pending = append(k.pending[:0], komootHeader...)
				k.stage = 1
			case 1:
				if k.i >= k.n {
					k.stage = 2
					continue
				}
				f := float64(k.i)
				lat := 47 + 0.05*math.Sin(f*0.001) + 0.0001*float64(k.i%7)
				lon := 11 + 0.05*math.Cos(f*0.001) + 0.0001*float64(k.i%5)
				ele := 1500 + 400*math.Sin(f/300)
				ts := time.Date(2024, 5, 1, 10, 0, 0, 0, time.UTC).Add(time.Duration(k.i) * time.Second)
				k.pending = fmt.Appendf(k.pending[:0],
					`<trkpt lat="%.6f" lon="%.6f"><ele>%.1f</ele><time>%s</time></trkpt>`,
					lat, lon, ele, ts.Format(time.RFC3339))
				k.i++
			case 2:
				k.pending = append(k.pending[:0], komootTail...)
				k.stage = 3
			default:
				if total == 0 {
					return 0, io.EOF
				}
				return total, nil
			}
		}
		c := copy(p[total:], k.pending)
		total += c
		k.pending = k.pending[c:]
		k.produced += int64(c)
	}
	return total, nil
}

// heapDelta is peak minus baseline, clamped at zero.
func heapDelta(peak, base uint64) uint64 {
	if peak < base {
		return 0
	}
	return peak - base
}

// Parsing 40,000 points (a 3.9 MB file) must stay in single-digit MB of live
// heap above the baseline at GOGC=25, and the retained Result must be small.
// The test is serial: a parallel test would distort the heap reading.
func TestParse40kPointsHeap(t *testing.T) {
	const (
		points      = 40000
		peakBudget  = 9 << 20
		retainedMax = 3 << 20
	)
	old := debug.SetGCPercent(25)
	t.Cleanup(func() { debug.SetGCPercent(old) })

	src := newKomootStream(points)
	runtime.GC()
	var base runtime.MemStats
	runtime.ReadMemStats(&base)

	stop := startSampler(200 * time.Microsecond)
	res, err := Parse(src, 25<<20, "komoot.gpx")
	peak := stop()
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if res.PointCount != points {
		t.Fatalf("PointCount = %d, want %d", res.PointCount, points)
	}
	t.Logf("generated %d bytes for %d points", src.produced, points)
	if src.produced < 3_850_000 || src.produced > 3_950_000 {
		t.Errorf("the stream is %d bytes, want about 3.9 MB (3,850,000 to 3,950,000)", src.produced)
	}

	if d := heapDelta(peak, base.HeapAlloc); d >= peakBudget {
		t.Errorf("peak live heap %d bytes above baseline, want under %d (9 MiB)", d, peakBudget)
	} else {
		t.Logf("peak live heap %d bytes above baseline", d)
	}

	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	if d := heapDelta(after.HeapAlloc, base.HeapAlloc); d >= retainedMax {
		t.Errorf("retained %d bytes after GC, want under %d (3 MiB)", d, retainedMax)
	} else {
		t.Logf("retained %d bytes after GC", d)
	}
	runtime.KeepAlive(res)
}

// countingWriter discards its input and counts it.
type countingWriter struct{ n int64 }

func (c *countingWriter) Write(p []byte) (int, error) { c.n += int64(len(p)); return len(p), nil }

// Encoding the same Result streams: the encoder never buffers the payload, so
// the live heap above the baseline stays small.
func TestEncodeGeoJSON40kHeap(t *testing.T) {
	const peakBudget = 4 << 20
	old := debug.SetGCPercent(25)
	t.Cleanup(func() { debug.SetGCPercent(old) })

	res, err := Parse(newKomootStream(40000), 25<<20, "komoot.gpx")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	runtime.GC()
	var base runtime.MemStats
	runtime.ReadMemStats(&base)

	cw := &countingWriter{}
	stop := startSampler(200 * time.Microsecond)
	err = res.EncodeGeoJSON(cw)
	peak := stop()
	if err != nil {
		t.Fatalf("EncodeGeoJSON: %v", err)
	}
	if d := heapDelta(peak, base.HeapAlloc); d >= peakBudget {
		t.Errorf("peak live heap %d bytes above baseline while encoding, want under %d (4 MiB)", d, peakBudget)
	} else {
		t.Logf("peak live heap %d bytes above baseline while encoding", d)
	}
	t.Logf("compressed GeoJSON: %d bytes (for reference, not asserted)", cw.n)
	runtime.KeepAlive(res)
}
