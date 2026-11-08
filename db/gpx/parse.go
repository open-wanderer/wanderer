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

	b := &builder{}
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

// point is the <trkpt> or <rtept> being read.
type point struct {
	lat, lon       float64
	hasLat, hasLon bool
	ele            float64
	hasEle         bool
	seenEle        bool
}

// builder turns the token stream into measured tracks. GPX structure is
// matched on Name.Local only, so prefixed GPX elements work. Everything inside
// an <extensions> subtree is skipped.
type builder struct {
	stack []string // open element names, outermost first
	skip  int      // depth inside an <extensions> subtree, 0 outside

	tracks []measuredTrack
	seg    *measuredSegment
	pt     *point

	segOrdinal int // <trkseg> elements seen so far in the current <trk>

	collecting bool // text of the current <ele> is being gathered
	buf        []byte
	overflow   bool
}

func (b *builder) parent() string {
	if len(b.stack) == 0 {
		return ""
	}
	return b.stack[len(b.stack)-1]
}

func (b *builder) start(t xml.StartElement) {
	if b.skip > 0 {
		b.skip++
		return
	}
	name := t.Name.Local
	parent := b.parent()
	b.stack = append(b.stack, name)
	switch name {
	case "extensions":
		b.skip = 1
	case "trk":
		b.tracks = append(b.tracks, measuredTrack{})
		b.segOrdinal = 0
	case "trkseg":
		if parent == "trk" && len(b.tracks) > 0 {
			b.seg = &measuredSegment{fileIndex: b.segOrdinal}
			b.segOrdinal++
		}
	case "trkpt":
		if parent == "trkseg" && b.seg != nil {
			b.pt = newPoint(t)
		}
	case "ele":
		if parent == "trkpt" && b.pt != nil && !b.pt.seenEle {
			b.pt.seenEle = true
			b.collecting = true
			b.buf = b.buf[:0]
			b.overflow = false
		}
	}
}

func newPoint(t xml.StartElement) *point {
	p := &point{}
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
	if !b.collecting || b.skip > 0 {
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
	case "ele":
		if b.collecting {
			b.collecting = false
			if b.pt != nil && !b.overflow {
				if v, ok := parseNumber(string(b.buf)); ok {
					b.pt.ele, b.pt.hasEle = v, true
				}
			}
		}
	case "trkpt":
		if b.pt != nil && b.seg != nil {
			if b.pt.hasLat && b.pt.hasLon {
				ele := math.NaN()
				if b.pt.hasEle {
					ele = b.pt.ele
				}
				b.seg.lat = append(b.seg.lat, b.pt.lat)
				b.seg.lon = append(b.seg.lon, b.pt.lon)
				b.seg.ele = append(b.seg.ele, ele)
				b.seg.timeMs = append(b.seg.timeMs, noTime)
			}
		}
		b.pt = nil
	case "trkseg":
		if b.seg != nil && len(b.tracks) > 0 {
			if len(b.seg.lat) > 0 {
				tr := &b.tracks[len(b.tracks)-1]
				tr.segments = append(tr.segments, *b.seg)
			}
		}
		b.seg = nil
	}
}

// finish chooses the measured source and measures it.
func (b *builder) finish(fallbackName string) (*Result, error) {
	tracks := b.tracks
	points := 0
	for ti := range tracks {
		for si := range tracks[ti].segments {
			points += len(tracks[ti].segments[si].lat)
		}
	}
	if points == 0 {
		return nil, ErrNoTrack
	}
	m := measure(tracks)
	first := &tracks[0]
	for ti := range tracks {
		if len(tracks[ti].segments) > 0 {
			first = &tracks[ti]
			break
		}
	}
	seg := first.segments[0]
	return &Result{
		Start:         LatLon{Lat: seg.lat[0], Lon: seg.lon[0]},
		Distance:      m.distance,
		ElevationGain: m.gain,
		ElevationLoss: m.loss,
		PointCount:    m.count,
		Bounds:        m.bounds,
		Centroid:      m.centroid,
		HasElevation:  m.hasElevation,
		tracks:        tracks,
	}, nil
}
