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

// setText copies the collected text children onto the track or route. number
// goes through parseNumber, like a coordinate.
func (t *measuredTrack) setText(ts *textSet) {
	t.name, t.desc, t.typ = ts.name, ts.desc, ts.typ
	t.number, t.hasNumber = parseNumber(ts.number)
}

// segmentTimes is the first and last point time of a segment, noTime when absent.
type segmentTimes struct{ start, end int64 }

// textSet holds the text children of one owner element: <metadata>, a <trk>,
// a <rte> or a <wpt>. The first value of each child wins, and every value is
// whitespace-trimmed (A7). An owner only ever receives the children that are
// meaningful for it, so the rest stay empty.
type textSet struct {
	name, desc, sym, typ, number string
	seen                         uint8 // one bit per child kind
}

// textKinds are the element names whose text is collected, in seen-bit order.
var textKinds = [...]string{"name", "desc", "sym", "type", "number"}

// put stores the trimmed text of a child element unless that child was seen.
func (ts *textSet) put(kind, value string) {
	for i, k := range textKinds {
		if k != kind {
			continue
		}
		if ts.seen&(1<<i) != 0 {
			return
		}
		ts.seen |= 1 << i
		value = strings.TrimSpace(value)
		switch kind {
		case "name":
			ts.name = value
		case "desc":
			ts.desc = value
		case "sym":
			ts.sym = value
		case "type":
			ts.typ = value
		case "number":
			ts.number = value
		}
		return
	}
}

// rawWaypoint is a <wpt> with a usable position, before its icon is resolved.
type rawWaypoint struct {
	lat, lon       float64
	hasLat, hasLon bool
	text           textSet
}

// decimalNumber is the TypeScript decimal grammar (web/src/lib/models/gpx/number.ts),
// ASCII digits only.
var decimalNumber = regexp.MustCompile(`^[+-]?(?:[0-9]+\.?[0-9]*|\.[0-9]+)(?:[eE][+-]?[0-9]+)?$`)

// parseNumber reads a coordinate or elevation. strconv.ParseFloat alone also
// accepts NaN, Inf, hex floats and underscores, so the decimal grammar gates it
// first, and a value out of float range is rejected rather than read as Inf.
// Text longer than maxNumberText is rejected, which keeps an oversized
// attribute value from being matched and parsed at all. strings.TrimSpace
// trims U+00A0 like JavaScript's trim.
func parseNumber(raw string) (float64, bool) {
	s := strings.TrimSpace(raw)
	if len(s) > maxNumberText || !decimalNumber.MatchString(s) {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || !isFinite(f) {
		return 0, false
	}
	return f, true
}

// parseBounded is parseNumber plus a magnitude bound: a value whose absolute
// value exceeds limit is unusable, like text that is not a number (D-01). The
// bound is inclusive, so exactly +-limit is kept.
func parseBounded(raw string, limit float64) (float64, bool) {
	v, ok := parseNumber(raw)
	if !ok || math.Abs(v) > limit {
		return 0, false
	}
	return v, true
}

// checkMeasurement refuses a measurement with a non-finite total. It is a
// backstop: the value bounds keep every sum finite, so it only fires if a later
// change loosens them. The reason is static and never echoes input.
func checkMeasurement(m measurement) error {
	if !isFinite(m.distance) || !isFinite(m.gain) || !isFinite(m.loss) ||
		!isFinite(m.centroid.Lat) || !isFinite(m.centroid.Lon) {
		return fmt.Errorf("%w: measurement out of range", ErrMalformed)
	}
	return nil
}

// classifyDecodeError maps a decoder or reader error onto a sentinel. The
// messages are static plus a line number: they never carry decoder text or
// input bytes.
//
// A truncated document is a SyntaxError whose message starts with "unexpected
// EOF". encoding/xml appends " in CDATA section" when the cut falls inside
// CDATA, so the prefix is the contract D-08 relies on, and TestTruncated pins
// the stdlib wording.
func classifyDecodeError(err error) error {
	var se *xml.SyntaxError
	switch {
	case errors.Is(err, ErrTooLarge):
		return ErrTooLarge
	case errors.Is(err, ErrMalformed):
		return err // raised by the reader pipeline with a static reason
	case errors.As(err, &se) && strings.HasPrefix(se.Msg, "unexpected EOF"):
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
	depth := 0
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
			// The decoder allocates per open element, even inside an
			// <extensions> subtree the builder skips, so the depth is
			// bounded here on every element.
			if depth++; depth > maxDepth {
				return nil, fmt.Errorf("%w: nesting too deep", ErrMalformed)
			}
			b.start(t)
		case xml.EndElement:
			depth--
			b.end(t.Name.Local)
		case xml.CharData:
			b.text(t)
		}
		if b.err != nil {
			return nil, b.err
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
// (fixture 14), whereas a <rtept> is filtered before timing (fixture 17). It
// reports whether the point was stored.
func (sb *segBuilder) add(p *point, timeAll bool) (stored bool) {
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
		return false
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
	return true
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
//
// Position rule: an element acts only at its place in the GPX schema. The root
// is depth 1, and depth means len(stack) after the element is pushed. A <trk>,
// <rte> or <wpt> is a direct child of the root (depth 2), a <trkseg> a direct
// child of such a <trk> (3), a <trkpt> a direct child of that segment (4), a
// <rtept> a direct child of such a <rte> (3), and <ele>, <time> and
// <extensions> are read only as direct children of the open point. Everything
// else is ignored, and a closer acts only on the element that opened its owner,
// so malformed nesting never moves data between owners or ends one early. This
// matches the TypeScript reader (xml2js, direct children only) and Dart.
type builder struct {
	stack []string // open element names, outermost first
	skip  int      // depth inside an <extensions> subtree, 0 outside
	// pointExt is set while inside the <extensions> of a point.
	pointExt bool

	sawTrk     bool
	tracks     []measuredTrack
	routes     []measuredTrack
	segOrdinal int // <trkseg> elements seen so far in the current <trk>

	tsb     *segBuilder // open <trkseg>
	rsb     *segBuilder // open <rte>
	pt      *point
	ptDepth int // depth of the open point

	// Signed duration sums of each store: a segment counts even when all of
	// its points are positionless.
	trkDuration, rteDuration int64
	// Set when a sum into the matching duration overflowed an int64. Only the
	// measured store's flag is read (D-04), so an overflowing route store beside
	// a track fails nothing.
	trkOverflow, rteOverflow bool
	// Times of the first <trkseg> of the first <trk>, and of the first <rte>.
	trkFirst, rteFirst segmentTimes

	// Text of the elements PARSE-04 reads. Only direct children of <metadata>,
	// <trk>, <rte> and <wpt> are collected, never <author><name> or
	// <copyright>.
	metaText, trkText, rteText textSet
	wpt                        *rawWaypoint // open <wpt>
	waypoints                  []rawWaypoint
	// Names of the first <trk> and the first <rte> element, empty or not.
	firstTrkName, firstRteName string

	// Running totals against the structural limits, and the first limit error,
	// which the token loop returns.
	points, segments, nWaypoints, textUsed int
	trackElements                          int // root-level <trk> and <rte> elements seen
	err                                    error

	collect      string   // element whose text is gathered: ele, time, or one of textKinds
	collectOwner *textSet // receiver of a textKinds element, nil for ele and time
	collectDepth int      // len(stack) while the collected element is open
	budgeted     bool     // the text counts towards maxTextBytes
	limit        int      // collected bytes beyond which a numeric value is unusable
	buf          []byte
	overflow     bool
}

// fail records the first structural-limit error. Its reason is static.
func (b *builder) fail(kind error, reason string) {
	if b.err == nil {
		b.err = fmt.Errorf("%w: %s", kind, reason)
	}
}

// addSegment counts a stored, non-empty segment.
func (b *builder) addSegment() {
	if b.segments++; b.segments > maxSegments {
		b.fail(ErrTooLarge, "too many segments")
	}
}

// addTrackElement counts a root-level <trk> or <rte> and reports whether it may
// be stored. Every one costs a measuredTrack, so empty ones count too.
func (b *builder) addTrackElement() bool {
	if b.trackElements++; b.trackElements > maxTracks {
		b.fail(ErrTooLarge, "too many tracks")
		return false
	}
	return true
}

// addPoint counts a stored point.
func (b *builder) addPoint() {
	if b.points++; b.points > maxPoints {
		b.fail(ErrTooLarge, "too many points")
	}
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
		b.pointExt = b.isPointChild(parent)
	case "trk":
		if len(b.stack) == 2 {
			// Any root-level <trk> element, even an empty one, blocks route
			// promotion. One nested elsewhere is not a track, as in TS.
			b.sawTrk = true
			if !b.addTrackElement() {
				return
			}
			b.tracks = append(b.tracks, measuredTrack{})
			b.trkText = textSet{}
			b.segOrdinal = 0
		}
	case "trkseg":
		if len(b.stack) == 3 && parent == "trk" {
			b.tsb = newSegBuilder(b.segOrdinal)
			b.segOrdinal++
		}
	case "rte":
		if len(b.stack) == 2 {
			if !b.addTrackElement() {
				return
			}
			b.routes = append(b.routes, measuredTrack{})
			b.rteText = textSet{}
			b.rsb = newSegBuilder(0)
		}
	case "wpt":
		if len(b.stack) == 2 {
			b.wpt = newWaypoint(t)
		}
	case "trkpt":
		if len(b.stack) == 4 && parent == "trkseg" && b.tsb != nil {
			b.pt, b.ptDepth = newPoint(t), len(b.stack)
		}
	case "rtept":
		if len(b.stack) == 3 && parent == "rte" && b.rsb != nil {
			b.pt, b.ptDepth = newPoint(t), len(b.stack)
		}
	case "ele":
		if b.isPointChild(parent) && !b.pt.seenEle {
			b.pt.seenEle = true
			b.beginText(name, nil, maxNumberText)
		}
	case "time":
		if b.isPointChild(parent) && !b.pt.seenTime {
			b.pt.seenTime = true
			b.beginText(name, nil, maxNumberText)
		}
	case "name", "desc", "sym", "type", "number":
		// Retained text is charged against maxTextBytes as it arrives. <number>
		// is numeric and is capped per element like <ele>.
		if owner := b.owner(parent); owner != nil && b.collect == "" {
			limit := math.MaxInt
			if name == "number" {
				limit = maxNumberText
			}
			b.beginText(name, owner, limit)
		}
	}
}

// isPointChild reports whether the element just pushed is a direct child of the
// open point, whose name is parent.
func (b *builder) isPointChild(parent string) bool {
	return b.pt != nil && len(b.stack) == b.ptDepth+1 && (parent == "trkpt" || parent == "rtept")
}

// owner is the textSet that receives a text child of parent, or nil when that
// child is not one the parser reads. The depth check keeps it to direct
// children of the root's own <metadata>, <trk>, <rte> and <wpt>.
func (b *builder) owner(parent string) *textSet {
	if len(b.stack) != 3 {
		return nil
	}
	switch parent {
	case "metadata":
		return &b.metaText
	case "trk":
		return &b.trkText
	case "rte":
		return &b.rteText
	case "wpt":
		if b.wpt != nil {
			return &b.wpt.text
		}
	}
	return nil
}

func (b *builder) beginText(name string, owner *textSet, limit int) {
	b.collect = name
	b.collectOwner = owner
	b.collectDepth = len(b.stack)
	b.budgeted = owner != nil && name != "number"
	b.limit = limit
	b.buf = b.buf[:0]
	b.overflow = false
}

// newWaypoint reads the position of a <wpt>. A coordinate that is not a number,
// or is beyond maxAbsCoordinate, leaves the waypoint unusable, so it is dropped and never placed at 0,0.
func newWaypoint(t xml.StartElement) *rawWaypoint {
	w := &rawWaypoint{}
	for _, a := range t.Attr {
		if a.Name.Space != "" {
			continue
		}
		switch a.Name.Local {
		case "lat":
			if !w.hasLat {
				w.lat, w.hasLat = parseBounded(a.Value, maxAbsCoordinate)
			}
		case "lon":
			if !w.hasLon {
				w.lon, w.hasLon = parseBounded(a.Value, maxAbsCoordinate)
			}
		}
	}
	return w
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
				p.lat, p.hasLat = parseBounded(a.Value, maxAbsCoordinate)
			}
		case "lon":
			if !p.hasLon {
				p.lon, p.hasLon = parseBounded(a.Value, maxAbsCoordinate)
			}
		}
	}
	return p
}

func (b *builder) text(data xml.CharData) {
	if b.collect == "" || b.skip > 0 {
		return
	}
	if b.overflow {
		return
	}
	if b.budgeted {
		if b.textUsed+len(b.buf)+len(data) > maxTextBytes {
			b.overflow = true
			b.fail(ErrTooLarge, "too much text")
			return
		}
	} else if len(data) > b.limit-len(b.buf) {
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
	case "name", "desc", "sym", "type", "number":
		// len(stack) is already popped, so the collected element sat one deeper.
		if b.collect != name || b.collectDepth != len(b.stack)+1 {
			return
		}
		b.collect = ""
		if b.budgeted {
			b.textUsed += len(b.buf)
		}
		if b.collectOwner == nil || b.overflow {
			return
		}
		b.collectOwner.put(name, string(b.buf))
	case "ele", "time":
		if b.collect != name || b.collectDepth != len(b.stack)+1 {
			return
		}
		b.collect = ""
		if b.pt == nil || b.overflow {
			return
		}
		if name == "ele" {
			if v, ok := parseBounded(string(b.buf), maxAbsElevation); ok {
				b.pt.ele, b.pt.hasEle = v, true
			}
		} else if ms, ok := parseTime(string(b.buf)); ok {
			b.pt.timeMs = ms
		}
	case "trkpt":
		if b.pt == nil || len(b.stack)+1 != b.ptDepth {
			return
		}
		if b.tsb != nil {
			if b.tsb.add(b.pt, true) {
				b.addPoint()
			}
		}
		b.pt = nil
	case "rtept":
		if b.pt == nil || len(b.stack)+1 != b.ptDepth {
			return
		}
		if b.rsb != nil {
			if b.rsb.add(b.pt, false) {
				b.addPoint()
			}
		}
		b.pt = nil
	case "trk":
		if len(b.stack) != 1 || len(b.tracks) == 0 {
			return
		}
		b.tracks[len(b.tracks)-1].setText(&b.trkText)
		if len(b.tracks) == 1 {
			b.firstTrkName = b.trkText.name
		}
	case "wpt":
		if len(b.stack) != 1 {
			return
		}
		if w := b.wpt; w != nil && w.hasLat && w.hasLon {
			if b.nWaypoints++; b.nWaypoints > maxWaypoints {
				b.fail(ErrTooLarge, "too many waypoints")
			} else {
				b.waypoints = append(b.waypoints, *w)
			}
		}
		b.wpt = nil
	case "trkseg":
		if len(b.stack) != 2 || b.tsb == nil || len(b.tracks) == 0 {
			return
		}
		sb := b.tsb
		b.tsb = nil
		legs, legsOK := sb.legs.total()
		var sumOK bool
		if b.trkDuration, sumOK = addDuration(b.trkDuration, legs); !legsOK || !sumOK {
			b.trkOverflow = true
		}
		if len(b.tracks) == 1 && sb.seg.fileIndex == 0 {
			b.trkFirst = sb.times()
		}
		if len(sb.seg.lat) > 0 {
			tr := &b.tracks[len(b.tracks)-1]
			tr.segments = append(tr.segments, sb.seg)
			b.addSegment()
		}
	case "rte":
		if len(b.stack) != 1 || b.rsb == nil || len(b.routes) == 0 {
			return
		}
		b.routes[len(b.routes)-1].setText(&b.rteText)
		if len(b.routes) == 1 {
			b.firstRteName = b.rteText.name
		}
		sb := b.rsb
		b.rsb = nil
		legs, legsOK := sb.legs.total()
		var sumOK bool
		if b.rteDuration, sumOK = addDuration(b.rteDuration, legs); !legsOK || !sumOK {
			b.rteOverflow = true
		}
		if len(b.routes) == 1 {
			b.rteFirst = sb.times()
		}
		if len(sb.seg.lat) > 0 {
			rt := &b.routes[len(b.routes)-1]
			rt.segments = append(rt.segments, sb.seg)
			b.addSegment()
		}
	}
}

// finish chooses the measured source and measures it. The source is the
// tracks when the file has any <trk> element, otherwise its routes promoted to
// one segment each (D-04). A <rte> beside a <trk> contributes nothing.
func (b *builder) finish(fallbackName string) (*Result, error) {
	source, duration, first, overflow := b.routes, b.rteDuration, b.rteFirst, b.rteOverflow
	if b.sawTrk {
		source, duration, first, overflow = b.tracks, b.trkDuration, b.trkFirst, b.trkOverflow
	}
	if _, ok := firstPoint(source); !ok {
		return nil, ErrNoTrack
	}
	m := measure(source)
	if err := checkMeasurement(m); err != nil {
		return nil, err
	}
	// An overflowed sum is a wrapped number, and math.MinInt64 cannot be
	// negated: neither is a duration, so both are an unusable measurement.
	if overflow || duration == math.MinInt64 {
		return nil, fmt.Errorf("%w: measurement out of range", ErrMalformed)
	}
	if duration < 0 {
		duration = -duration
	}
	r := &Result{
		Distance:      m.distance,
		ElevationGain: m.gain,
		ElevationLoss: m.loss,
		DurationMs:    duration,
		PointCount:    m.count,
		Bounds:        m.bounds,
		Centroid:      m.centroid,
		tracks:        source,
		firstTimes:    first,
	}
	b.deriveTrail(r, fallbackName)
	return r, nil
}
