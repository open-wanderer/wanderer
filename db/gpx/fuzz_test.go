package gpx

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/twpayne/go-polyline"
)

// fuzzLimit is the byte cap of every fuzz iteration, small so each one is fast.
const fuzzLimit = 1 << 20

// fuzzSeeds gathers the seed inputs: every corpus input, every exporter
// sample, and hostile inputs built in code. A missing directory fails the
// fuzz test instead of silently shrinking the seed set.
func fuzzSeeds(f *testing.F) [][]byte {
	var seeds [][]byte

	corpus, err := os.ReadDir(corpusRoot)
	if err != nil {
		f.Fatalf("corpus %q: %v", corpusRoot, err)
	}
	for _, e := range corpus {
		if !e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(corpusRoot, e.Name(), "input.gpx"))
		if err != nil {
			f.Fatalf("corpus fixture %s: %v", e.Name(), err)
		}
		seeds = append(seeds, b)
	}
	if len(seeds) < corpusMinFixtures {
		f.Fatalf("corpus gave %d seeds, want at least %d", len(seeds), corpusMinFixtures)
	}

	exporters, err := filepath.Glob(filepath.Join("testdata", "exporters", "*.gpx"))
	if err != nil || len(exporters) == 0 {
		f.Fatalf("testdata/exporters has no samples (%v)", err)
	}
	for _, p := range exporters {
		b, err := os.ReadFile(p)
		if err != nil {
			f.Fatalf("%s: %v", p, err)
		}
		seeds = append(seeds, b)
	}

	// Fixture 10 cut at five offsets.
	full, err := os.ReadFile(filepath.Join(corpusRoot, "10-realistic-track", "input.gpx"))
	if err != nil {
		f.Fatalf("fixture 10: %v", err)
	}
	for i := 1; i <= 5; i++ {
		seeds = append(seeds, full[:len(full)*i/6])
	}

	const track = `<trk><trkseg><trkpt lat="47" lon="11"><ele>5</ele></trkpt></trkseg></trk>`
	seeds = append(seeds,
		[]byte(entityBomb()),
		[]byte("<gpx><trk><name>a\x00b</name><trkseg><trkpt lat=\"47\" lon=\"11\"/></trkseg></trk></gpx>"),
		[]byte("<gpx><trk><name>a\xff\xfeb</name><trkseg><trkpt lat=\"47\" lon=\"11\"/></trkseg></trk></gpx>"),
		[]byte(`<gpx><trk><trkseg><trkpt lat="1e999" lon="11"/><trkpt lat="47" lon="11"/></trkseg></trk></gpx>`),
		[]byte("<gpx>"+strings.Repeat("<a>", 200)+track+"</gpx>"),
		[]byte(`<?xml version="1.0" encoding="x-bogus"?><gpx>`+track+`</gpx>`),
		append(utf16le(`<gpx><trk><name>a`, 0xD800), utf16le(`b</name><trkseg><trkpt lat="47" lon="11"/></trkseg></trk></gpx>`)[2:]...),
		// CR-01: finite numbers far beyond any real value. They once summed to
		// Inf and left a nil error beside a garbage polyline.
		[]byte(`<gpx><trk><trkseg><trkpt lat="47" lon="11"><ele>1e308</ele></trkpt><trkpt lat="47.001" lon="11"><ele>-1e308</ele></trkpt></trkseg></trk></gpx>`),
		[]byte(`<gpx><trk><trkseg><trkpt lat="1e308" lon="11"/><trkpt lat="1e308" lon="11"/><trkpt lat="47" lon="11"/></trkseg></trk></gpx>`),
		// CR-02: a flood of empty track and route elements, then one real track.
		[]byte("<gpx>"+strings.Repeat("<trk/><rte/>", 40)+track+"</gpx>"),
	)
	return seeds
}

func finite(vs ...float64) bool {
	for _, v := range vs {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}

// checkResult asserts the invariants of a successful parse.
func checkResult(t *testing.T, res *Result) {
	t.Helper()
	if res == nil {
		t.Fatalf("nil Result without an error")
	}
	if res.PointCount < 1 {
		t.Fatalf("PointCount = %d, want at least 1", res.PointCount)
	}
	b := res.Bounds
	if !finite(b.MinLat, b.MaxLat, b.MinLon, b.MaxLon) {
		t.Fatalf("Bounds is not finite: %+v", b)
	}
	stored := 0
	for ti := range res.tracks {
		for si := range res.tracks[ti].segments {
			s := &res.tracks[ti].segments[si]
			if len(s.lat) != len(s.lon) || len(s.lat) != len(s.ele) || len(s.lat) != len(s.timeMs) {
				t.Fatalf("segment arrays differ in length: %d %d %d %d", len(s.lat), len(s.lon), len(s.ele), len(s.timeMs))
			}
			for i := range s.lat {
				stored++
				if !finite(s.lat[i], s.lon[i]) {
					t.Fatalf("stored position %d is not finite", i)
				}
				if s.lat[i] < b.MinLat || s.lat[i] > b.MaxLat || s.lon[i] < b.MinLon || s.lon[i] > b.MaxLon {
					t.Fatalf("stored position %d lies outside Bounds %+v", i, b)
				}
				if e := s.ele[i]; e == e && math.IsInf(e, 0) {
					t.Fatalf("stored elevation %d is infinite", i)
				}
			}
		}
	}
	if stored != res.PointCount {
		t.Fatalf("stored %d points, PointCount = %d", stored, res.PointCount)
	}
	if !finite(res.Distance, res.ElevationGain, res.ElevationLoss) ||
		res.Distance < 0 || res.ElevationGain < 0 || res.ElevationLoss < 0 {
		t.Fatalf("metrics must be finite and non-negative: distance %v gain %v loss %v", res.Distance, res.ElevationGain, res.ElevationLoss)
	}
	if res.DurationMs < 0 {
		t.Fatalf("DurationMs = %d, want non-negative", res.DurationMs)
	}
	if !finite(res.Centroid.Lat, res.Centroid.Lon, res.Start.Lat, res.Start.Lon) {
		t.Fatalf("Centroid or Start is not finite")
	}
	enc, err := res.Polyline()
	if err != nil && !errors.Is(err, ErrTooComplex) {
		t.Fatalf("Polyline: %v, want nil or ErrTooComplex", err)
	}
	if err == nil {
		// The encoded track must decode back to coordinates inside Bounds. A
		// garbage encoding of out-of-range values would not (CR-01).
		coords, rest, derr := polyline.DecodeCoords([]byte(enc))
		if derr != nil || len(rest) != 0 {
			t.Fatalf("Polyline does not decode: %v with %d bytes left", derr, len(rest))
		}
		if len(coords) > res.PointCount {
			t.Fatalf("Polyline decodes to %d coordinates, PointCount = %d", len(coords), res.PointCount)
		}
		const slack = 1e-5
		for i, c := range coords {
			if len(c) != 2 || c[0] < b.MinLat-slack || c[0] > b.MaxLat+slack || c[1] < b.MinLon-slack || c[1] > b.MaxLon+slack {
				t.Fatalf("decoded polyline coordinate %d lies outside Bounds %+v", i, b)
			}
		}
	}

	var buf bytes.Buffer
	if err := res.EncodeGeoJSON(&buf); err != nil {
		t.Fatalf("EncodeGeoJSON: %v", err)
	}
	zr, err := gzip.NewReader(&buf)
	if err != nil {
		t.Fatalf("GeoJSON is not gzip: %v", err)
	}
	raw, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("GeoJSON gunzip: %v", err)
	}
	if !json.Valid(raw) {
		t.Fatalf("GeoJSON is not valid JSON")
	}
	var doc struct {
		Bbox []float64 `json:"bbox"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("GeoJSON decode: %v", err)
	}
	if len(doc.Bbox) != 4 || !finite(doc.Bbox...) {
		t.Fatalf("GeoJSON bbox = %v, want four finite numbers", doc.Bbox)
	}
}

// FuzzParse feeds arbitrary bytes to Parse. Plain `go test` runs the seed
// corpus only, which is cheap enough for every CI run. The engine run is
// manual:
//
//	go test ./gpx/ -run '^$' -fuzz '^FuzzParse$' -fuzztime 10m
//
// A crasher the engine writes under testdata/fuzz/FuzzParse/ is committed
// together with its fix.
func FuzzParse(f *testing.F) {
	for _, seed := range fuzzSeeds(f) {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		// A panic is reported with the input length only, never the input.
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("panic on an input of %d bytes: %v", len(data), r)
			}
		}()
		res, err := Parse(bytes.NewReader(data), fuzzLimit, "fuzz.gpx")
		if err != nil {
			if res != nil {
				t.Fatalf("non-nil Result with error %q", err)
			}
			if n := sentinelCount(err); n != 1 {
				t.Fatalf("error %q matches %d sentinels, want exactly 1", err, n)
			}
			return
		}
		checkResult(t, res)
	})
}
