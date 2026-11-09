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
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Golden comparisons here always run on the DECOMPRESSED bytes: the deflate
// output may change between Go releases, the JSON may not.

type gjFeature struct {
	Type     string `json:"type"`
	Geometry struct {
		Type        string      `json:"type"`
		Coordinates [][]float64 `json:"coordinates"`
	} `json:"geometry"`
	Properties map[string]json.RawMessage `json:"properties"`
}

type gjDoc struct {
	Type     string                     `json:"type"`
	Features []gjFeature                `json:"features"`
	Bbox     []float64                  `json:"bbox"`
	Wanderer map[string]json.RawMessage `json:"wanderer"`
}

// gunzipGeoJSON encodes res, checks the gzip magic and the validity of the
// JSON, and returns the decompressed bytes.
func gunzipGeoJSON(t *testing.T, res *Result) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := res.EncodeGeoJSON(&buf); err != nil {
		t.Fatalf("EncodeGeoJSON: %v", err)
	}
	if b := buf.Bytes(); len(b) < 2 || b[0] != 0x1f || b[1] != 0x8b {
		t.Fatalf("output does not start with the gzip magic")
	}
	zr, err := gzip.NewReader(&buf)
	if err != nil {
		t.Fatalf("gzip.NewReader: %v", err)
	}
	raw, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("gunzip: %v", err)
	}
	if !json.Valid(raw) {
		t.Fatalf("decompressed output is not valid JSON")
	}
	return raw
}

func decodeGeoJSON(t *testing.T, res *Result) (gjDoc, []byte) {
	t.Helper()
	raw := gunzipGeoJSON(t, res)
	var doc gjDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("decoding GeoJSON: %v", err)
	}
	return doc, raw
}

func parseCorpusFixture(t *testing.T, dir string) *Result {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(corpusRoot, dir, "input.gpx"))
	if err != nil {
		t.Fatalf("fixture %s: %v", dir, err)
	}
	res, err := Parse(bytes.NewReader(b), parseLimit, "")
	if err != nil {
		t.Fatalf("fixture %s: Parse: %v", dir, err)
	}
	return res
}

// parseDoc is mustParse with no fallback name.
func parseDoc(t *testing.T, doc string) *Result {
	t.Helper()
	return mustParse(t, doc, "")
}

func (f gjFeature) str(t *testing.T, key string) string {
	t.Helper()
	raw, ok := f.Properties[key]
	if !ok {
		t.Fatalf("property %q is missing", key)
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("property %q: %v", key, err)
	}
	return s
}

func (f gjFeature) intProp(t *testing.T, key string) int {
	t.Helper()
	raw, ok := f.Properties[key]
	if !ok {
		t.Fatalf("property %q is missing", key)
	}
	var n int
	if err := json.Unmarshal(raw, &n); err != nil {
		t.Fatalf("property %q: %v", key, err)
	}
	return n
}

func (f gjFeature) times(t *testing.T) []*string {
	t.Helper()
	var cp struct {
		Times []*string `json:"times"`
	}
	raw, ok := f.Properties["coordinateProperties"]
	if !ok {
		t.Fatalf("property coordinateProperties is missing")
	}
	if err := json.Unmarshal(raw, &cp); err != nil {
		t.Fatalf("coordinateProperties: %v", err)
	}
	return cp.Times
}

func (f gjFeature) anchors(t *testing.T) []int {
	t.Helper()
	raw, ok := f.Properties["anchors"]
	if !ok {
		t.Fatalf("property anchors is missing")
	}
	if string(raw) == "null" {
		t.Fatalf("anchors is null, want an array")
	}
	var a []int
	if err := json.Unmarshal(raw, &a); err != nil {
		t.Fatalf("anchors: %v", err)
	}
	return a
}

func (d gjDoc) wandererMarker(t *testing.T) (v float64, elevation string) {
	t.Helper()
	if len(d.Wanderer) != 2 {
		t.Fatalf("wanderer has %d members, want exactly v and elevation", len(d.Wanderer))
	}
	if err := json.Unmarshal(d.Wanderer["v"], &v); err != nil {
		t.Fatalf("wanderer.v: %v", err)
	}
	if err := json.Unmarshal(d.Wanderer["elevation"], &elevation); err != nil {
		t.Fatalf("wanderer.elevation: %v", err)
	}
	return v, elevation
}

// trkptsDoc builds a one-segment track from raw <trkpt> elements.
func trkptsDoc(points ...string) string {
	return `<gpx><trk><trkseg>` + strings.Join(points, "") + `</trkseg></trk></gpx>`
}

func TestGeoJSONShapeAndDeterminism(t *testing.T) {
	res := parseCorpusFixture(t, "10-realistic-track")
	doc, raw := decodeGeoJSON(t, res)

	if doc.Type != "FeatureCollection" {
		t.Errorf("type = %q, want FeatureCollection", doc.Type)
	}
	if len(doc.Features) == 0 {
		t.Fatalf("no features")
	}
	want := []float64{res.Bounds.MinLon, res.Bounds.MinLat, res.Bounds.MaxLon, res.Bounds.MaxLat}
	if !reflect.DeepEqual(doc.Bbox, want) {
		t.Errorf("bbox = %v, want [minLon minLat maxLon maxLat] = %v", doc.Bbox, want)
	}
	if v, elevation := doc.wandererMarker(t); v != 1 || elevation != "file" {
		t.Errorf("wanderer = {v:%v, elevation:%q}, want {v:1, elevation:file}", v, elevation)
	}
	if !bytes.Contains(raw, []byte(`"wanderer":{"v":1,"elevation":"file"}`)) {
		t.Errorf("the marker is not spelled as the exact literal member")
	}
	for i, f := range doc.Features {
		if f.Type != "Feature" || f.Geometry.Type != "LineString" {
			t.Errorf("feature %d: type %q geometry %q", i, f.Type, f.Geometry.Type)
		}
	}
	// Two encodes of the same Result give identical decompressed bytes. The
	// compressed bytes are never compared.
	if again := gunzipGeoJSON(t, res); !bytes.Equal(raw, again) {
		t.Errorf("two encodes of one Result decompress to different bytes")
	}
}

func TestGeoJSONPropertySet(t *testing.T) {
	const gpxDoc = `<gpx><trk><name>Full</name><desc>Everything</desc><type>hiking</type><number>7</number>` +
		`<trkseg><trkpt lat="47" lon="11"><ele>1</ele><time>2024-05-01T10:00:00Z</time></trkpt>` +
		`<trkpt lat="47.001" lon="11"><ele>2</ele><time>2024-05-01T10:00:10.250Z</time></trkpt></trkseg></trk></gpx>`
	doc, _ := decodeGeoJSON(t, parseDoc(t, gpxDoc))
	if len(doc.Features) != 1 {
		t.Fatalf("%d features, want 1", len(doc.Features))
	}
	f := doc.Features[0]
	keys := make([]string, 0, len(f.Properties))
	for k := range f.Properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	wantKeys := []string{"anchors", "coordinateProperties", "desc", "featureId", "name", "number", "segmentId", "type"}
	if !reflect.DeepEqual(keys, wantKeys) {
		t.Errorf("property keys = %v, want %v", keys, wantKeys)
	}
	if got := f.str(t, "name"); got != "Full" {
		t.Errorf("name = %q", got)
	}
	if got := f.str(t, "desc"); got != "Everything" {
		t.Errorf("desc = %q", got)
	}
	if got := f.str(t, "type"); got != "hiking" {
		t.Errorf("type = %q", got)
	}
	if got := f.intProp(t, "number"); got != 7 {
		t.Errorf("number = %d, want 7", got)
	}
	times := f.times(t)
	if len(times) != 2 || times[0] == nil || times[1] == nil {
		t.Fatalf("times = %v", times)
	}
	if *times[0] != "2024-05-01T10:00:00.000Z" || *times[1] != "2024-05-01T10:00:10.250Z" {
		t.Errorf("times = %q, %q, want ISO strings with milliseconds and Z", *times[0], *times[1])
	}
}

func TestGeoJSONOptionalPropertiesAndNullTimes(t *testing.T) {
	// No name, desc, type or number, and a point without a time.
	doc, _ := decodeGeoJSON(t, parseDoc(t, trkptsDoc(
		`<trkpt lat="47" lon="11"><time>2024-05-01T10:00:00Z</time></trkpt>`,
		`<trkpt lat="47.001" lon="11"></trkpt>`,
	)))
	f := doc.Features[0]
	for _, key := range []string{"name", "desc", "type", "number"} {
		if _, ok := f.Properties[key]; ok {
			t.Errorf("property %q present, want it omitted", key)
		}
	}
	for _, key := range []string{"featureId", "segmentId", "coordinateProperties", "anchors"} {
		if _, ok := f.Properties[key]; !ok {
			t.Errorf("property %q missing", key)
		}
	}
	times := f.times(t)
	if len(times) != 2 || times[0] == nil || times[1] != nil {
		t.Errorf("times = %v, want [time, null]", times)
	}

	// An unusable <number> is omitted, not written as 0.
	doc, _ = decodeGeoJSON(t, parseDoc(t,
		`<gpx><trk><number>abc</number><trkseg><trkpt lat="47" lon="11"/></trkseg></trk></gpx>`))
	if _, ok := doc.Features[0].Properties["number"]; ok {
		t.Errorf("number present for an unusable <number>")
	}
}

func TestGeoJSONEscapesHostileStrings(t *testing.T) {
	nasty := "q\"uote\\back</script>&amp;\ttab" + string(rune(0x2028)) + "end"
	// The XML escapes make the decoder deliver the characters above.
	gpxDoc := `<gpx><trk><name>q"uote\back&lt;/script&gt;&amp;amp;` + "\t" + `tab` + string(rune(0x2028)) + `end</name>` +
		`<desc>{"injected":true}</desc><trkseg><trkpt lat="47" lon="11"/></trkseg></trk></gpx>`
	doc, raw := decodeGeoJSON(t, parseDoc(t, gpxDoc))
	f := doc.Features[0]
	if got := f.str(t, "name"); got != nasty {
		t.Errorf("name round trip: got %q, want %q", got, nasty)
	}
	if got := f.str(t, "desc"); got != `{"injected":true}` {
		t.Errorf("desc = %q", got)
	}
	if bytes.Contains(raw, []byte(`"injected":true`)) {
		t.Errorf("hostile text escaped its string")
	}
	if len(doc.Features) != 1 {
		t.Errorf("%d features, want 1", len(doc.Features))
	}
}

func TestGeoJSONPositionsWithoutElevation(t *testing.T) {
	// Fixture 03: the third point has an empty <ele>.
	doc, _ := decodeGeoJSON(t, parseCorpusFixture(t, "03-missing-vs-empty-ele"))
	coords := doc.Features[0].Geometry.Coordinates
	if len(coords) != 5 {
		t.Fatalf("%d coordinates, want 5", len(coords))
	}
	wantLen := []int{3, 3, 2, 3, 3}
	for i, c := range coords {
		if len(c) != wantLen[i] {
			t.Errorf("position %d = %v, want %d elements (D-18: no fabricated elevation)", i, c, wantLen[i])
		}
	}
	if c := coords[0]; c[0] != 11 || c[1] != 47 || c[2] != 1000 {
		t.Errorf("position 0 = %v, want [lon lat ele] = [11 47 1000]", c)
	}
	if c := coords[2]; c[0] != 11 || c[1] != 47.002 {
		t.Errorf("position 2 = %v, want [11 47.002]", c)
	}
	if _, elevation := doc.wandererMarker(t); elevation != "file" {
		t.Errorf("elevation = %q, want file (4 of 5 points carry one and they vary)", elevation)
	}
}

func TestGeoJSONNoElevationMarker(t *testing.T) {
	t.Run("flat track keeps its values", func(t *testing.T) {
		doc, _ := decodeGeoJSON(t, parseDoc(t, trkptsDoc(
			`<trkpt lat="47" lon="11"><ele>1000</ele></trkpt>`,
			`<trkpt lat="47.001" lon="11"><ele>1000</ele></trkpt>`,
			`<trkpt lat="47.002" lon="11"><ele>1000</ele></trkpt>`,
		)))
		if v, elevation := doc.wandererMarker(t); v != 1 || elevation != "none" {
			t.Errorf("wanderer = {v:%v, elevation:%q}, want {v:1, elevation:none}", v, elevation)
		}
		for i, c := range doc.Features[0].Geometry.Coordinates {
			if len(c) != 3 || c[2] != 1000 {
				t.Errorf("position %d = %v, want the real ele 1000 kept", i, c)
			}
		}
	})
	t.Run("under half the points", func(t *testing.T) {
		doc, _ := decodeGeoJSON(t, parseDoc(t, trkptsDoc(
			`<trkpt lat="47" lon="11"><ele>1000</ele></trkpt>`,
			`<trkpt lat="47.001" lon="11"/>`,
			`<trkpt lat="47.002" lon="11"><ele>1100</ele></trkpt>`,
			`<trkpt lat="47.003" lon="11"/>`,
			`<trkpt lat="47.004" lon="11"/>`,
		)))
		if _, elevation := doc.wandererMarker(t); elevation != "none" {
			t.Errorf("elevation = %q, want none", elevation)
		}
		wantLen := []int{3, 2, 3, 2, 2}
		for i, c := range doc.Features[0].Geometry.Coordinates {
			if len(c) != wantLen[i] {
				t.Errorf("position %d = %v, want %d elements", i, c, wantLen[i])
			}
		}
		if c := doc.Features[0].Geometry.Coordinates[2]; c[2] != 1100 {
			t.Errorf("position 2 = %v, want the real ele 1100 kept", c)
		}
	})
	t.Run("marker follows HasElevation", func(t *testing.T) {
		res := parseDoc(t, trkptsDoc(`<trkpt lat="47" lon="11"><ele>5</ele></trkpt>`))
		doc, _ := decodeGeoJSON(t, res)
		_, elevation := doc.wandererMarker(t)
		if res.HasElevation || elevation != "none" {
			t.Errorf("HasElevation=%v elevation=%q, want false and none for a single usable value", res.HasElevation, elevation)
		}
	})
}

func TestGeoJSONMarkerValuesAllFixtures(t *testing.T) {
	for _, f := range loadCorpus(t) {
		t.Run(f.dir, func(t *testing.T) {
			res := parseCorpusFixture(t, f.dir)
			doc, raw := decodeGeoJSON(t, res)
			v, elevation := doc.wandererMarker(t)
			if v != 1 {
				t.Errorf("wanderer.v = %v, want 1", v)
			}
			if elevation != "file" && elevation != "none" {
				t.Errorf("wanderer.elevation = %q, want file or none", elevation)
			}
			if (elevation == "none") != !res.HasElevation {
				t.Errorf("elevation %q with HasElevation %v", elevation, res.HasElevation)
			}
			if bytes.Contains(raw, []byte(`"dem"`)) {
				t.Errorf("the reserved dem value was emitted")
			}
			for i, ft := range doc.Features {
				for j, c := range ft.Geometry.Coordinates {
					if len(c) != 2 && len(c) != 3 {
						t.Errorf("feature %d position %d has %d elements", i, j, len(c))
					}
					for _, x := range c {
						if math.IsNaN(x) || math.IsInf(x, 0) {
							t.Errorf("feature %d position %d is not finite: %v", i, j, c)
						}
					}
				}
			}
		})
	}
}

func TestGeoJSONSegments(t *testing.T) {
	doc, _ := decodeGeoJSON(t, parseCorpusFixture(t, "09-multi-segment-planner-route"))
	if len(doc.Features) != 2 {
		t.Fatalf("%d features, want one per segment", len(doc.Features))
	}
	for i, f := range doc.Features {
		if got := f.intProp(t, "segmentId"); got != i {
			t.Errorf("feature %d: segmentId = %d, want %d", i, got, i)
		}
		if got := f.intProp(t, "featureId"); got != i {
			t.Errorf("feature %d: featureId = %d, want it equal to segmentId", i, got)
		}
		if n := len(f.Geometry.Coordinates); n != 3 {
			t.Errorf("feature %d: %d coordinates, want 3", i, n)
		}
	}

	// A skipped empty segment does not renumber the later ones (A10).
	doc, _ = decodeGeoJSON(t, parseDoc(t, `<gpx><trk><name>Gap</name>`+
		`<trkseg><trkpt lat="47" lon="11"/></trkseg>`+
		`<trkseg></trkseg>`+
		`<trkseg><trkpt lat="48" lon="11"/></trkseg></trk></gpx>`))
	if len(doc.Features) != 2 {
		t.Fatalf("%d features, want 2", len(doc.Features))
	}
	if a, b := doc.Features[0].intProp(t, "segmentId"), doc.Features[1].intProp(t, "segmentId"); a != 0 || b != 2 {
		t.Errorf("segmentIds = %d, %d, want 0 and 2 (the file position)", a, b)
	}
	if a, b := doc.Features[0].intProp(t, "featureId"), doc.Features[1].intProp(t, "featureId"); a != 0 || b != 2 {
		t.Errorf("featureIds = %d, %d, want 0 and 2", a, b)
	}

	// Two tracks: the segment index restarts for each track.
	doc, _ = decodeGeoJSON(t, parseDoc(t, `<gpx><trk><name>A</name><trkseg><trkpt lat="47" lon="11"/></trkseg></trk>`+
		`<trk><name>B</name><trkseg><trkpt lat="48" lon="11"/></trkseg></trk></gpx>`))
	if len(doc.Features) != 2 || doc.Features[1].str(t, "name") != "B" || doc.Features[1].intProp(t, "segmentId") != 0 {
		t.Errorf("second track: %d features, name/segmentId not B/0", len(doc.Features))
	}
}

func TestGeoJSONAnchors(t *testing.T) {
	// Fixture 15: the anchor is the third of four points.
	doc, _ := decodeGeoJSON(t, parseCorpusFixture(t, "15-route-anchor"))
	if got := doc.Features[0].anchors(t); !reflect.DeepEqual(got, []int{2}) {
		t.Errorf("fixture 15 anchors = %v, want [2]", got)
	}

	// Anchors on the first and last point are not interior and are left out.
	const ext = `<extensions><wanderer:anchor xmlns:wanderer="` + wandererNamespace + `"/></extensions>`
	doc, _ = decodeGeoJSON(t, parseDoc(t, trkptsDoc(
		`<trkpt lat="47" lon="11">`+ext+`</trkpt>`,
		`<trkpt lat="47.001" lon="11"/>`,
		`<trkpt lat="47.002" lon="11">`+ext+`</trkpt>`,
		`<trkpt lat="47.003" lon="11"/>`,
		`<trkpt lat="47.004" lon="11">`+ext+`</trkpt>`,
	)))
	if got := doc.Features[0].anchors(t); !reflect.DeepEqual(got, []int{2}) {
		t.Errorf("anchors = %v, want only the interior index [2]", got)
	}

	// The index is into the feature's own coordinates: a positionless point
	// before the anchor does not shift it (Pitfall 13).
	doc, _ = decodeGeoJSON(t, parseDoc(t, trkptsDoc(
		`<trkpt/>`,
		`<trkpt lat="47" lon="11"/>`,
		`<trkpt lat="47.001" lon="11"/>`,
		`<trkpt lat="47.002" lon="11">`+ext+`</trkpt>`,
		`<trkpt lat="47.003" lon="11"/>`,
	)))
	if got := doc.Features[0].anchors(t); !reflect.DeepEqual(got, []int{2}) {
		t.Errorf("anchors with a dropped positionless point = %v, want [2]", got)
	}

	// No anchors: the key is present as an empty array.
	doc, raw := decodeGeoJSON(t, parseCorpusFixture(t, "09-multi-segment-planner-route"))
	if got := doc.Features[0].anchors(t); len(got) != 0 {
		t.Errorf("anchors = %v, want empty", got)
	}
	if !bytes.Contains(raw, []byte(`"anchors":[]`)) {
		t.Errorf("an empty anchors array is not written as []")
	}
}

func TestGeoJSONPositionlessPoints(t *testing.T) {
	doc, _ := decodeGeoJSON(t, parseCorpusFixture(t, "14-positionless-ends"))
	f := doc.Features[0]
	if n := len(f.Geometry.Coordinates); n != 2 {
		t.Fatalf("%d coordinates, want only the two positioned points", n)
	}
	times := f.times(t)
	want := []string{"2024-05-01T10:00:10.000Z", "2024-05-01T10:00:20.000Z"}
	if len(times) != 2 || times[0] == nil || times[1] == nil || *times[0] != want[0] || *times[1] != want[1] {
		t.Errorf("times = %v, want %v", times, want)
	}
}

func TestGeoJSONPromotedRoutes(t *testing.T) {
	// Fixture 17: two routes become two track features, waypoints are not emitted.
	doc, raw := decodeGeoJSON(t, parseCorpusFixture(t, "17-route-only-promotion"))
	if len(doc.Features) != 2 {
		t.Fatalf("%d features, want 2", len(doc.Features))
	}
	for i, name := range []string{"Route Only North", "Route Only Return"} {
		f := doc.Features[i]
		if got := f.str(t, "name"); got != name {
			t.Errorf("feature %d name = %q, want %q", i, got, name)
		}
		if got := f.intProp(t, "segmentId"); got != 0 {
			t.Errorf("feature %d segmentId = %d, want 0", i, got)
		}
		if f.Geometry.Type != "LineString" {
			t.Errorf("feature %d geometry = %q", i, f.Geometry.Type)
		}
	}
	if bytes.Contains(raw, []byte("Route Camp")) || bytes.Contains(raw, []byte(`"Point"`)) {
		t.Errorf("a waypoint was emitted")
	}

	// Fixture 18: the routes of a mixed file are not emitted (D-04).
	doc, _ = decodeGeoJSON(t, parseCorpusFixture(t, "18-mixed-rte-trk"))
	if len(doc.Features) != 1 {
		t.Fatalf("%d features, want 1", len(doc.Features))
	}
	if got := doc.Features[0].str(t, "name"); got != "Mixed Track Wins" {
		t.Errorf("name = %q, want Mixed Track Wins", got)
	}
	for _, c := range doc.Features[0].Geometry.Coordinates {
		if c[1] < 0 {
			t.Errorf("position %v comes from the route in the southern hemisphere", c)
		}
	}
}

// closeRecorder records Close calls and keeps what it was given.
type closeRecorder struct {
	bytes.Buffer
	closes int
}

func (c *closeRecorder) Close() error { c.closes++; return nil }

// failingWriter fails every write after limit bytes and counts the calls that
// came after the failure.
type failingWriter struct {
	limit, written, afterFailure int
	failed                       bool
}

var errSink = errors.New("sink failed")

func (w *failingWriter) Write(p []byte) (int, error) {
	if w.failed {
		w.afterFailure++
		return 0, errSink
	}
	if w.written+len(p) > w.limit {
		w.failed = true
		return 0, errSink
	}
	w.written += len(p)
	return len(p), nil
}

func TestGeoJSONGzipAndWriterOwnership(t *testing.T) {
	res := parseCorpusFixture(t, "10-realistic-track")

	rec := &closeRecorder{}
	if err := res.EncodeGeoJSON(rec); err != nil {
		t.Fatalf("EncodeGeoJSON: %v", err)
	}
	if rec.closes != 0 {
		t.Errorf("the encoder closed the caller's writer %d times, want 0", rec.closes)
	}
	// The stream is complete without any Close of ours: the gzip trailer was written.
	zr, err := gzip.NewReader(bytes.NewReader(rec.Bytes()))
	if err != nil {
		t.Fatalf("gzip.NewReader: %v", err)
	}
	if _, err := io.ReadAll(zr); err != nil {
		t.Errorf("the gzip stream is not complete: %v", err)
	}
	if zr.Name != "" || !zr.ModTime.IsZero() {
		t.Errorf("gzip header carries Name %q ModTime %v, want neither", zr.Name, zr.ModTime)
	}

	// A writer that fails at once, and one that fails mid-stream.
	if err := res.EncodeGeoJSON(&failingWriter{limit: 0}); err == nil {
		t.Errorf("a writer failing at the first write gave a nil error")
	}
	var b strings.Builder
	b.WriteString(`<gpx><trk><trkseg>`)
	for i := 0; i < 30000; i++ {
		b.WriteString(`<trkpt lat="`)
		b.WriteString(strconv.FormatFloat(47+float64(i)*1.37e-6*float64(i%97+1), 'f', -1, 64))
		b.WriteString(`" lon="`)
		b.WriteString(strconv.FormatFloat(11+float64(i)*2.11e-6*float64(i%89+1), 'f', -1, 64))
		b.WriteString(`"><ele>`)
		b.WriteString(strconv.Itoa(1000 + i%311))
		b.WriteString(`</ele></trkpt>`)
	}
	b.WriteString(`</trkseg></trk></gpx>`)
	big := parseDoc(t, b.String())
	fw := &failingWriter{limit: 200}
	if err := big.EncodeGeoJSON(fw); !errors.Is(err, errSink) {
		t.Errorf("a writer failing mid-stream gave %v, want the sink error", err)
	}
	if fw.afterFailure > 1 {
		t.Errorf("the encoder kept writing after the failure (%d more writes)", fw.afterFailure)
	}
}

// WR-01: years outside 0000-9999 are written in the six-digit signed form of
// Date.prototype.toISOString, so JavaScript's Date reads every time back.
func TestGeoJSONExtendedYearTimes(t *testing.T) {
	res := parseDoc(t, trkptsDoc(
		`<trkpt lat="1" lon="1"><time>+012345-01-01T00:00:00Z</time></trkpt>`,
		`<trkpt lat="1.1" lon="1.1"><time>-000001-06-01T12:00:00Z</time></trkpt>`,
		`<trkpt lat="1.2" lon="1.2"><time>2024-05-01T10:00:00Z</time></trkpt>`,
	))
	doc, _ := decodeGeoJSON(t, res)
	times := doc.Features[0].times(t)
	want := []string{"+012345-01-01T00:00:00.000Z", "-000001-06-01T12:00:00.000Z", "2024-05-01T10:00:00.000Z"}
	if len(times) != len(want) {
		t.Fatalf("times = %v, want %v", times, want)
	}
	for i, w := range want {
		if times[i] == nil || *times[i] != w {
			t.Errorf("times[%d] = %v, want %q", i, times[i], w)
		}
	}
}
