package gpx

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// noTime marks a point whose <time> is absent or unparseable.
const noTime int64 = math.MinInt64

// maxNumberText bounds the text collected for one <ele> or <time> element. A
// longer value is treated as unusable rather than buffered.
const maxNumberText = 1024

// measuredSegment is one run of usable points in struct-of-arrays form.
type measuredSegment struct {
	fileIndex int       // 0-based position among its <trk>'s <trkseg> elements; 0 for a promoted <rte>
	lat, lon  []float64 // usable points only, file values
	ele       []float64 // math.NaN() where the point has no usable <ele>
	timeMs    []int64   // noTime where <time> is absent or unparseable
	anchors   []int     // indices into lat/lon of stored points that carry a wanderer:anchor mark
}

// measuredTrack is one <trk>, or one promoted <rte>.
type measuredTrack struct {
	name, desc, typ string
	number          float64
	hasNumber       bool
	segments        []measuredSegment // only segments with at least one usable point
}

// segmentTimes is the first and last point time of a segment, noTime when absent.
type segmentTimes struct{ start, end int64 }

// decimalNumber is the TypeScript decimal grammar (web/src/lib/models/gpx/number.ts),
// ASCII digits only.
var decimalNumber = regexp.MustCompile(`^[+-]?(?:[0-9]+\.?[0-9]*|\.[0-9]+)(?:[eE][+-]?[0-9]+)?$`)

// parseNumber reads a coordinate or elevation. strconv.ParseFloat alone also
// accepts NaN, Inf, hex floats and underscores, so the decimal grammar gates it
// first, and a value out of float range is rejected rather than read as Inf.
// strings.TrimSpace trims U+00A0 like JavaScript's trim.
func parseNumber(raw string) (float64, bool) {
	s := strings.TrimSpace(raw)
	if !decimalNumber.MatchString(s) {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || !isFinite(f) {
		return 0, false
	}
	return f, true
}

// classifyDecodeError maps a decoder or reader error onto a sentinel. The
// messages are static plus a line number: they never carry decoder text or
// input bytes.
func classifyDecodeError(err error) error {
	var se *xml.SyntaxError
	switch {
	case errors.Is(err, ErrTooLarge):
		return ErrTooLarge
	case errors.As(err, &se) && se.Msg == "unexpected EOF":
		return ErrTruncated
	case errors.Is(err, io.EOF):
		return fmt.Errorf("%w: empty document", ErrMalformed)
	case errors.As(err, &se):
		return fmt.Errorf("%w: syntax error at line %d", ErrMalformed, se.Line)
	default:
		return fmt.Errorf("%w: unreadable document", ErrMalformed)
	}
}

// decode streams the document through the builder.
//
// The decoder is tolerant (Strict=false) and resolves the HTML entity set, so
// "&nbsp;" and friends decode. Its HTML auto-close list stays unset: that list
// marks <link> as void, which would split GPX <link><text> (Pitfall 4). The
// charset reader is the identity because the pipeline has already decoded.
func decode(in io.Reader, fallbackName string) (*Result, error) {
	d := xml.NewDecoder(in)
	d.Strict = false
	d.Entity = xml.HTMLEntity
	d.CharsetReader = func(_ string, r io.Reader) (io.Reader, error) { return r, nil }

	b := newBuilder()
	seenRoot := false
	for {
		tok, err := d.Token()
		if err != nil {
			if err == io.EOF && seenRoot {
				break
			}
			return nil, classifyDecodeError(err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			seenRoot = true
			b.start(t)
		case xml.EndElement:
			b.end(t.Name.Local)
		case xml.CharData:
			b.text(t)
		}
	}
	return b.finish(fallbackName)
}

// wandererNamespace is the namespace of the wanderer:anchor leg marker, as
// route_anchor_marks.dart and gpx_segment_util.ts write it. An undeclared
// prefix leaves the decoder's Name.Space as the literal prefix, which TS and
// Dart also match.
const (
	wandererNamespace = "https://wanderer.to/xmlns/gpx/1"
	wandererPrefix    = "wanderer"
)

// point is the <trkpt> or <rtept> being read.
type point struct {
	lat, lon       float64
	hasLat, hasLon bool
	ele            float64
	hasEle         bool
	seenEle        bool
	timeMs         int64
	seenTime       bool
	anchor         bool
}

// segBuilder collects one <trkseg>, or the usable points of one <rte>.
type segBuilder struct {
	seg      measuredSegment
	legs     legsAccumulator // duration, fed per the rules of the point kind
	first    int64           // time of the first point fed to legs
	last     int64           // time of the last point fed to legs
	timeSeen int             // points fed to legs
}

func newSegBuilder(fileIndex int) *segBuilder {
	return &segBuilder{seg: measuredSegment{fileIndex: fileIndex}, first: noTime, last: noTime}
}

// add feeds a closed point. A point with a usable position is stored. Its time
// and anchor also feed the duration, as does that of a positionless point when
// timeAll is set: the app still reads the time of a <trkpt> without a position
// (fixture 14), whereas a <rtept> is filtered before timing (fixture 17).
func (sb *segBuilder) add(p *point, timeAll bool) {
	usable := p.hasLat && p.hasLon
	if usable || timeAll {
		sb.legs.add(p.timeMs, p.anchor)
		if sb.timeSeen == 0 {
			sb.first = p.timeMs
		}
		sb.last = p.timeMs
		sb.timeSeen++
	}
	if !usable {
		return
	}
	ele := math.NaN()
	if p.hasEle {
		ele = p.ele
	}
	if p.anchor {
		sb.seg.anchors = append(sb.seg.anchors, len(sb.seg.lat))
	}
	sb.seg.lat = append(sb.seg.lat, p.lat)
	sb.seg.lon = append(sb.seg.lon, p.lon)
	sb.seg.ele = append(sb.seg.ele, ele)
	sb.seg.timeMs = append(sb.seg.timeMs, p.timeMs)
}

// times is the first and last time of the points fed to the duration.
func (sb *segBuilder) times() segmentTimes { return segmentTimes{start: sb.first, end: sb.last} }

// builder turns the token stream into measured tracks. GPX structure is
// matched on Name.Local only, so prefixed GPX elements work. Everything inside
// an <extensions> subtree is skipped, except a wanderer:anchor directly below
// the <extensions> of a point.
//
// Tracks and routes go to separate stores because a <rte> may precede a <trk>
// in the file: which one is measured is decided in finish().
type builder struct {
	stack []string // open element names, outermost first
	skip  int      // depth inside an <extensions> subtree, 0 outside
	// pointExt is set while inside the <extensions> of a point.
	pointExt bool

	sawTrk     bool
	tracks     []measuredTrack
	routes     []measuredTrack
	segOrdinal int // <trkseg> elements seen so far in the current <trk>

	tsb *segBuilder // open <trkseg>
	rsb *segBuilder // open <rte>
	pt  *point

	// Signed duration sums of each store: a segment counts even when all of
	// its points are positionless.
	trkDuration, rteDuration int64
	// Times of the first <trkseg> of the first <trk>, and of the first <rte>.
	trkFirst, rteFirst segmentTimes

	collect  string // "ele" or "time" while that element's text is gathered
	buf      []byte
	overflow bool
}

func newBuilder() *builder {
	none := segmentTimes{start: noTime, end: noTime}
	return &builder{trkFirst: none, rteFirst: none}
}

func (b *builder) parent() string {
	if len(b.stack) == 0 {
		return ""
	}
	return b.stack[len(b.stack)-1]
}

func isAnchor(name xml.Name) bool {
	return name.Local == "anchor" && (name.Space == wandererNamespace || name.Space == wandererPrefix)
}

func (b *builder) start(t xml.StartElement) {
	if b.skip > 0 {
		if b.skip == 1 && b.pointExt && b.pt != nil && isAnchor(t.Name) {
			b.pt.anchor = true
		}
		b.skip++
		return
	}
	name := t.Name.Local
	parent := b.parent()
	b.stack = append(b.stack, name)
	switch name {
	case "extensions":
		b.skip = 1
		b.pointExt = b.pt != nil && (parent == "trkpt" || parent == "rtept")
	case "trk":
		// Any <trk> element, even an empty one, blocks route promotion.
		b.sawTrk = true
		b.tracks = append(b.tracks, measuredTrack{})
		b.segOrdinal = 0
	case "trkseg":
		if parent == "trk" {
			b.tsb = newSegBuilder(b.segOrdinal)
			b.segOrdinal++
		}
	case "rte":
		b.routes = append(b.routes, measuredTrack{})
		b.rsb = newSegBuilder(0)
	case "trkpt":
		if parent == "trkseg" && b.tsb != nil {
			b.pt = newPoint(t)
		}
	case "rtept":
		if parent == "rte" && b.rsb != nil {
			b.pt = newPoint(t)
		}
	case "ele":
		if b.pt != nil && (parent == "trkpt" || parent == "rtept") && !b.pt.seenEle {
			b.pt.seenEle = true
			b.beginText(name)
		}
	case "time":
		if b.pt != nil && (parent == "trkpt" || parent == "rtept") && !b.pt.seenTime {
			b.pt.seenTime = true
			b.beginText(name)
		}
	}
}

func (b *builder) beginText(name string) {
	b.collect = name
	b.buf = b.buf[:0]
	b.overflow = false
}

func newPoint(t xml.StartElement) *point {
	p := &point{timeMs: noTime}
	for _, a := range t.Attr {
		if a.Name.Space != "" {
			continue
		}
		switch a.Name.Local {
		case "lat":
			if !p.hasLat {
				p.lat, p.hasLat = parseNumber(a.Value)
			}
		case "lon":
			if !p.hasLon {
				p.lon, p.hasLon = parseNumber(a.Value)
			}
		}
	}
	return p
}

func (b *builder) text(data xml.CharData) {
	if b.collect == "" || b.skip > 0 {
		return
	}
	if len(b.buf)+len(data) > maxNumberText {
		b.overflow = true
		return
	}
	b.buf = append(b.buf, data...)
}

// end closes the innermost open element called name, auto-closing any
// elements opened inside it. An end tag with no matching open element is
// ignored, as the tolerant decoder can emit those. Inside an <extensions>
// subtree only the depth is tracked: nothing in there is on the stack.
func (b *builder) end(name string) {
	if b.skip > 1 {
		b.skip--
		return
	}
	if b.skip == 1 {
		b.skip = 0
		b.pointExt = false
		b.stack = b.stack[:len(b.stack)-1]
		return
	}
	idx := -1
	for i := len(b.stack) - 1; i >= 0; i-- {
		if b.stack[i] == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}
	for len(b.stack) > idx {
		top := b.stack[len(b.stack)-1]
		b.stack = b.stack[:len(b.stack)-1]
		b.closeElement(top)
	}
}

func (b *builder) closeElement(name string) {
	switch name {
	case "ele", "time":
		if b.collect != name {
			return
		}
		b.collect = ""
		if b.pt == nil || b.overflow {
			return
		}
		if name == "ele" {
			if v, ok := parseNumber(string(b.buf)); ok {
				b.pt.ele, b.pt.hasEle = v, true
			}
		} else if ms, ok := parseTime(string(b.buf)); ok {
			b.pt.timeMs = ms
		}
	case "trkpt":
		if b.pt != nil && b.tsb != nil {
			b.tsb.add(b.pt, true)
		}
		b.pt = nil
	case "rtept":
		if b.pt != nil && b.rsb != nil {
			b.rsb.add(b.pt, false)
		}
		b.pt = nil
	case "trkseg":
		if b.tsb == nil || len(b.tracks) == 0 {
			return
		}
		sb := b.tsb
		b.tsb = nil
		b.trkDuration += sb.legs.total()
		if len(b.tracks) == 1 && sb.seg.fileIndex == 0 {
			b.trkFirst = sb.times()
		}
		if len(sb.seg.lat) > 0 {
			tr := &b.tracks[len(b.tracks)-1]
			tr.segments = append(tr.segments, sb.seg)
		}
	case "rte":
		if b.rsb == nil || len(b.routes) == 0 {
			return
		}
		sb := b.rsb
		b.rsb = nil
		b.rteDuration += sb.legs.total()
		if len(b.routes) == 1 {
			b.rteFirst = sb.times()
		}
		if len(sb.seg.lat) > 0 {
			rt := &b.routes[len(b.routes)-1]
			rt.segments = append(rt.segments, sb.seg)
		}
	}
}

// finish chooses the measured source and measures it. The source is the
// tracks when the file has any <trk> element, otherwise its routes promoted to
// one segment each (D-04). A <rte> beside a <trk> contributes nothing.
func (b *builder) finish(fallbackName string) (*Result, error) {
	source, duration, first := b.routes, b.rteDuration, b.rteFirst
	if b.sawTrk {
		source, duration, first = b.tracks, b.trkDuration, b.trkFirst
	}
	var start *measuredSegment
	for ti := range source {
		if len(source[ti].segments) > 0 {
			start = &source[ti].segments[0]
			break
		}
	}
	if start == nil {
		return nil, ErrNoTrack
	}
	m := measure(source)
	if duration < 0 {
		duration = -duration
	}
	return &Result{
		Start:         LatLon{Lat: start.lat[0], Lon: start.lon[0]},
		Distance:      m.distance,
		ElevationGain: m.gain,
		ElevationLoss: m.loss,
		DurationMs:    duration,
		PointCount:    m.count,
		Bounds:        m.bounds,
		Centroid:      m.centroid,
		HasElevation:  m.hasElevation,
		tracks:        source,
		firstTimes:    first,
	}, nil
}
