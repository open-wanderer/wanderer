package gpx

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/twpayne/go-polyline"
)

// These tests feed Parse generated streams that never exist as a []byte and
// check three things for each: the sentinel it ends with, how many bytes the
// parser consumed before it stopped, and how long it took. They run serially:
// a parallel test would distort the heap and time bounds.

// hostileChunk is the most a hostile source returns per Read. It models a
// network or file source that hands over data in small pieces, so the bytes
// consumed measure how soon the parser stops, not how much the readers above
// it chose to ask for at once.
const hostileChunk = 512

// hostileTimeLimit is the longest any hostile case may take.
const hostileTimeLimit = 10 * time.Second

// chunkReader limits each Read to max bytes.
type chunkReader struct {
	r   io.Reader
	max int
}

func (c chunkReader) Read(p []byte) (int, error) {
	if len(p) > c.max {
		p = p[:c.max]
	}
	return c.r.Read(p)
}

// forever is the repeat count of an endless stream.
const forever = math.MaxInt

// hostileSource (counting with the countingReader of reader_test.go) streams head, then n copies of item, then tail, in chunks of
// hostileChunk, counting what is consumed. n == forever never ends.
func hostileSource(head, item string, n int, tail string) *countingReader {
	return &countingReader{r: chunkReader{r: &repeatReader{head: head, item: item, n: n, tail: tail}, max: hostileChunk}}
}

// utf16le encodes ASCII text as UTF-16LE with a BOM, then appends raw code
// units, so a test can place a lone surrogate.
func utf16le(ascii string, units ...uint16) []byte {
	out := []byte{0xFF, 0xFE}
	for i := 0; i < len(ascii); i++ {
		out = append(out, ascii[i], 0)
	}
	for _, u := range units {
		out = append(out, byte(u), byte(u>>8))
	}
	return out
}

// entityBomb is the classic billion-laughs document followed by a valid track.
func entityBomb() string {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0"?><!DOCTYPE lolz [<!ENTITY lol "lol">`)
	for i := 1; i <= 9; i++ {
		prev := "lol"
		if i > 1 {
			prev = fmt.Sprintf("lol%d", i-1)
		}
		fmt.Fprintf(&sb, `<!ENTITY lol%d "`, i)
		for j := 0; j < 10; j++ {
			fmt.Fprintf(&sb, "&%s;", prev)
		}
		sb.WriteString(`">`)
	}
	sb.WriteString(`]><gpx><metadata><name>&lol9;</name></metadata>` +
		`<trk><trkseg><trkpt lat="47" lon="11"/></trkseg></trk></gpx>`)
	return sb.String()
}

// The earliest and latest instants the time grammar accepts. A positionless
// segment running from one to the other adds 17,280,000,000,000,000 ms to the
// duration, so 533 of them fit in an int64 and 534 do not.
const (
	earliestTime   = `<time>-271821-04-20T00:00:00Z</time>`
	latestTime     = `<time>+275760-09-13T00:00:00Z</time>`
	extremeSegment = `<trkseg><trkpt>` + earliestTime + `</trkpt><trkpt>` + latestTime + `</trkpt></trkseg>`
	// anchoredPair closes one extreme leg per pair, inside a single segment.
	anchoredPair = `<trkpt>` + earliestTime + `<extensions><wanderer:anchor/></extensions></trkpt><trkpt>` + latestTime + `</trkpt>`
	extremeRoute = `<rte><rtept lat="47" lon="11">` + earliestTime + `</rtept><rtept lat="47.001" lon="11">` + latestTime + `</rtept></rte>`
)

type hostileCase struct {
	name string
	// src builds the input; the counting reader is returned so the case can
	// read its consumption afterwards.
	src      func() *countingReader
	maxBytes int64
	// want is the set of acceptable sentinels; nil means Parse must succeed.
	want []error
	// maxRead bounds the bytes consumed from the source.
	maxRead int64
	// maxPeak and maxRetained, when positive, bound the live heap above the
	// baseline while parsing and after a collection.
	maxPeak, maxRetained uint64
	// check runs on a successful result.
	check func(t *testing.T, res *Result)
}

func TestHostileInputs(t *testing.T) {
	const (
		pointItem     = `<trkpt lat="47" lon="11"/>`
		waypointItem  = `<wpt lat="47" lon="11"/>`
		readAhead     = 64 << 10
		oneMiBPlusEnd = maxRun + readAhead
	)
	cases := []hostileCase{
		{
			name:     "endless text run",
			src:      func() *countingReader { return hostileSource(`<gpx><trk><name>`, "aaaaaaaaaaaaaaaa", forever, "") },
			maxBytes: 1 << 30, want: []error{ErrMalformed}, maxRead: oneMiBPlusEnd,
		},
		{
			name:     "endless attribute value",
			src:      func() *countingReader { return hostileSource(`<gpx a="`, "aaaaaaaaaaaaaaaa", forever, "") },
			maxBytes: 1 << 30, want: []error{ErrMalformed}, maxRead: oneMiBPlusEnd,
		},
		{
			name:     "endless tag name",
			src:      func() *countingReader { return hostileSource(`<gpx><trk><`, "aaaaaaaaaaaaaaaa", forever, "") },
			maxBytes: 1 << 30, want: []error{ErrMalformed}, maxRead: oneMiBPlusEnd,
		},
		{
			name:     "endless whitespace",
			src:      func() *countingReader { return hostileSource(`<gpx>`, "                ", forever, "") },
			maxBytes: 1 << 30, want: []error{ErrMalformed}, maxRead: oneMiBPlusEnd,
		},
		{
			name:     "endless nesting",
			src:      func() *countingReader { return hostileSource(`<gpx>`, "<a>", forever, "") },
			maxBytes: 1 << 30, want: []error{ErrMalformed}, maxRead: 4 << 10,
		},
		{
			name:     "nesting one past maxDepth",
			src:      func() *countingReader { return hostileSource(`<gpx>`, "<a>", maxDepth, "") },
			maxBytes: 1 << 30, want: []error{ErrMalformed}, maxRead: 4 << 10,
		},
		{
			name:     "empty segment flood under a 4 MiB cap",
			src:      func() *countingReader { return hostileSource(`<gpx><trk>`, `<trkseg/>`, forever, "") },
			maxBytes: 4 << 20, want: []error{ErrTooLarge}, maxRead: 4<<20 + 1,
			maxRetained: 1 << 20,
		},
		{
			name:     "waypoint flood past maxWaypoints",
			src:      func() *countingReader { return hostileSource(`<gpx>`, waypointItem, forever, "") },
			maxBytes: 1 << 30, want: []error{ErrTooLarge}, maxRead: int64(maxWaypoints+1)*int64(len(waypointItem)) + 4<<10,
		},
		{
			name:     "positionless waypoint flood under a 4 MiB cap",
			src:      func() *countingReader { return hostileSource(`<gpx>`, `<wpt/>`, forever, "") },
			maxBytes: 4 << 20, want: []error{ErrTooLarge, ErrNoTrack}, maxRead: 4<<20 + 1,
		},
		{
			name:     "point flood under a 1 GiB cap",
			src:      func() *countingReader { return hostileSource(`<gpx><trk><trkseg>`, pointItem, forever, "") },
			maxBytes: 1 << 30, want: []error{ErrTooLarge}, maxRead: int64(maxPoints+1)*int64(len(pointItem)) + 4<<10,
		},
		{
			name: "latitude of 2 MiB of digits",
			src: func() *countingReader {
				return hostileSource(`<gpx><trk><trkseg><trkpt lat="`, strings.Repeat("1", 1024), 2048,
					`" lon="11"/></trkseg></trk></gpx>`)
			},
			maxBytes: 25 << 20, want: []error{ErrMalformed}, maxRead: oneMiBPlusEnd,
		},
		{
			name: "100,000 xmlns attributes in one tag",
			src: func() *countingReader {
				return hostileSource(`<gpx `, `xmlns:a="u" `, 100_000, `><trk><trkseg><trkpt lat="47" lon="11"/></trkseg></trk></gpx>`)
			},
			maxBytes: 25 << 20, want: []error{ErrMalformed}, maxRead: oneMiBPlusEnd,
		},
		{
			name: "entity bomb keeps its literal text",
			src: func() *countingReader {
				return &countingReader{r: strings.NewReader(entityBomb())}
			},
			maxBytes: 25 << 20, want: nil, maxRead: 1 << 20, maxPeak: 16 << 20,
			check: func(t *testing.T, res *Result) {
				if res.Name != "&lol9;" {
					t.Errorf("Name = %q, want the literal entity text %q", res.Name, "&lol9;")
				}
				if res.PointCount != 1 {
					t.Errorf("PointCount = %d, want 1", res.PointCount)
				}
			},
		},
		{
			name: "finite-but-huge elevation swing",
			src: func() *countingReader {
				return &countingReader{r: strings.NewReader(`<gpx><trk><trkseg>` +
					`<trkpt lat="47" lon="11"><ele>1e308</ele></trkpt>` +
					`<trkpt lat="47.001" lon="11"><ele>-1e308</ele></trkpt>` +
					`</trkseg></trk></gpx>`)}
			},
			maxBytes: 25 << 20, want: nil, maxRead: 1 << 20,
			check: func(t *testing.T, res *Result) {
				if res.ElevationGain != 0 || res.ElevationLoss != 0 || res.HasElevation {
					t.Errorf("gain %v loss %v HasElevation %v, want 0, 0, false", res.ElevationGain, res.ElevationLoss, res.HasElevation)
				}
				if res.PointCount != 2 {
					t.Errorf("PointCount = %d, want 2", res.PointCount)
				}
			},
		},
		{
			name: "finite-but-huge latitudes",
			src: func() *countingReader {
				return &countingReader{r: strings.NewReader(`<gpx><trk><trkseg>` +
					`<trkpt lat="1e308" lon="11"/><trkpt lat="1e308" lon="11"/><trkpt lat="47" lon="11"/>` +
					`</trkseg></trk></gpx>`)}
			},
			maxBytes: 25 << 20, want: nil, maxRead: 1 << 20,
			check: func(t *testing.T, res *Result) {
				want := LatLon{Lat: 47, Lon: 11}
				if res.PointCount != 1 || res.Centroid != want || res.Start != want {
					t.Errorf("PointCount %d Centroid %+v Start %+v, want 1 and %+v for both", res.PointCount, res.Centroid, res.Start, want)
				}
				enc, err := res.Polyline()
				if err != nil {
					t.Fatalf("Polyline: %v", err)
				}
				coords, rest, err := polyline.DecodeCoords([]byte(enc))
				if err != nil || len(rest) != 0 || len(coords) != 1 {
					t.Fatalf("decoded %v with %d bytes left, error %v, want one coordinate", coords, len(rest), err)
				}
				b := res.Bounds
				if c := coords[0]; c[0] < b.MinLat-1e-5 || c[0] > b.MaxLat+1e-5 || c[1] < b.MinLon-1e-5 || c[1] > b.MaxLon+1e-5 {
					t.Errorf("decoded coordinate %v lies outside Bounds %+v", c, b)
				}
			},
		},
		{
			name: "only out-of-bound positions",
			src: func() *countingReader {
				return &countingReader{r: strings.NewReader(`<gpx><trk><trkseg>` +
					`<trkpt lat="1e308" lon="11"/><trkpt lat="47" lon="-1e308"/><trkpt lat="1e7" lon="1e7"/>` +
					`</trkseg></trk></gpx>`)}
			},
			maxBytes: 25 << 20, want: []error{ErrNoTrack}, maxRead: 1 << 20,
		},
		{
			name: "duration at the int64 edge parses",
			src: func() *countingReader {
				return hostileSource(`<gpx><trk><trkseg><trkpt lat="47" lon="11"/></trkseg>`, extremeSegment, 533, `</trk></gpx>`)
			},
			maxBytes: 25 << 20, want: nil, maxRead: 1 << 20,
			check: func(t *testing.T, res *Result) {
				const want = 9210240000000000000 // 533 extreme segments, the most an int64 holds
				if res.DurationMs != want || res.PointCount != 1 {
					t.Errorf("DurationMs %d PointCount %d, want %d and 1", res.DurationMs, res.PointCount, want)
				}
			},
		},
		{
			name: "duration past int64 is malformed",
			src: func() *countingReader {
				return hostileSource(`<gpx><trk><trkseg><trkpt lat="47" lon="11"/></trkseg>`, extremeSegment, 534, `</trk></gpx>`)
			},
			maxBytes: 25 << 20, want: []error{ErrMalformed}, maxRead: 1 << 20,
		},
		{
			name: "anchored legs past int64 in one segment are malformed",
			src: func() *countingReader {
				return hostileSource(`<gpx><trk><trkseg><trkpt lat="47" lon="11"/></trkseg><trkseg>`, anchoredPair, 600, `</trkseg></trk></gpx>`)
			},
			maxBytes: 25 << 20, want: []error{ErrMalformed}, maxRead: 1 << 20,
		},
		{
			name: "an overflowing route store beside a track is not measured",
			src: func() *countingReader {
				return hostileSource(`<gpx>`, extremeRoute, 534, `<trk><trkseg><trkpt lat="47" lon="11"/></trkseg></trk></gpx>`)
			},
			maxBytes: 25 << 20, want: nil, maxRead: 1 << 20,
			check: func(t *testing.T, res *Result) {
				if res.DurationMs != 0 || res.PointCount != 1 {
					t.Errorf("DurationMs %d PointCount %d, want 0 and 1 (routes beside a track are not measured)", res.DurationMs, res.PointCount)
				}
			},
		},
		{
			name: "broken UTF-16 surrogate",
			src: func() *countingReader {
				data := utf16le(`<gpx><trk><name>a`, 0xD800)
				data = append(data, utf16le(`b</name><trkseg><trkpt lat="47" lon="11"/></trkseg></trk></gpx>`)[2:]...)
				return &countingReader{r: bytes.NewReader(data)}
			},
			maxBytes: 25 << 20, want: nil, maxRead: 1 << 20,
			check: func(t *testing.T, res *Result) {
				if !strings.Contains(res.Name, string(rune(0xFFFD))) {
					t.Errorf("Name = %q, want the lone surrogate repaired to U+FFFD", res.Name)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stopPeak func() uint64
			var baseline runtime.MemStats
			if tc.maxPeak > 0 || tc.maxRetained > 0 {
				runtime.GC()
				runtime.ReadMemStats(&baseline)
			}
			if tc.maxPeak > 0 {
				stopPeak = startSampler(200 * time.Microsecond)
			}
			src := tc.src()
			start := time.Now()
			res, err := Parse(src, tc.maxBytes, "hostile.gpx")
			elapsed := time.Since(start)
			var peak uint64
			if stopPeak != nil {
				peak = stopPeak()
			}

			if elapsed > hostileTimeLimit {
				t.Errorf("took %v, want under %v", elapsed, hostileTimeLimit)
			}
			if src.n > tc.maxRead {
				t.Errorf("consumed %d bytes, want at most %d", src.n, tc.maxRead)
			}
			if tc.want == nil {
				if err != nil {
					t.Fatalf("Parse: %v", err)
				}
				if tc.check != nil {
					tc.check(t, res)
				}
			} else {
				matched := false
				for _, w := range tc.want {
					if errors.Is(err, w) {
						matched = true
					}
				}
				if !matched {
					t.Errorf("err = %v, want one of %v", err, tc.want)
				} else if n := sentinelCount(err); n != 1 {
					t.Errorf("err %q matches %d sentinels, want exactly 1", err, n)
				}
				if res != nil {
					t.Errorf("Result is non-nil on error")
				}
			}

			if tc.maxPeak > 0 && peak > baseline.HeapAlloc && peak-baseline.HeapAlloc > tc.maxPeak {
				t.Errorf("peak heap %d bytes above baseline, want under %d", peak-baseline.HeapAlloc, tc.maxPeak)
			}
			if tc.maxRetained > 0 {
				res = nil
				runtime.GC()
				var after runtime.MemStats
				runtime.ReadMemStats(&after)
				if after.HeapAlloc > baseline.HeapAlloc && after.HeapAlloc-baseline.HeapAlloc > tc.maxRetained {
					t.Errorf("retained %d bytes after GC, want under %d", after.HeapAlloc-baseline.HeapAlloc, tc.maxRetained)
				}
			}
			t.Logf("consumed %d bytes in %v", src.n, elapsed.Round(time.Millisecond))
		})
	}
}

// TestSizeCapStreams pins the byte cap against an endless valid stream: the
// source is read for exactly the limit plus the one probe byte, never further.
func TestSizeCapStreams(t *testing.T) {
	const limit = 2 << 20
	point := `<trkpt lat="47.123456" lon="11.123456"><ele>1000.5</ele><time>2024-05-01T10:00:00Z</time></trkpt>`
	src := hostileSource(`<gpx><trk><trkseg>`, point, forever, "")
	start := time.Now()
	res, err := Parse(src, limit, "cap.gpx")
	elapsed := time.Since(start)

	wantErr(t, res, err, ErrTooLarge)
	if src.n < limit || src.n > limit+1 {
		t.Errorf("consumed %d bytes, want exactly %d plus at most 1 probe byte", src.n, limit)
	}
	if elapsed > hostileTimeLimit {
		t.Errorf("took %v, want under %v", elapsed, hostileTimeLimit)
	}

	// The same stream under a cap that is not hit by a structural limit first,
	// but with the document closed in time, parses.
	src = hostileSource(`<gpx><trk><trkseg>`, point, 1000, `</trkseg></trk></gpx>`)
	res, err = Parse(src, limit, "cap.gpx")
	if err != nil {
		t.Fatalf("a 1000-point document under the cap: %v", err)
	}
	if res.PointCount != 1000 {
		t.Errorf("PointCount = %d, want 1000", res.PointCount)
	}
}
