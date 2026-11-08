package gpx

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// sentinels are the five typed errors a caller can test with errors.Is.
var sentinels = []error{ErrTruncated, ErrNoTrack, ErrTooLarge, ErrMalformed, ErrTooComplex}

// sentinelCount counts how many sentinels err matches. Every error Parse
// returns must match exactly one.
func sentinelCount(err error) int {
	n := 0
	for _, s := range sentinels {
		if errors.Is(err, s) {
			n++
		}
	}
	return n
}

// wantErr asserts Parse failed with exactly the sentinel want and no Result.
func wantErr(t *testing.T, res *Result, err error, want error) {
	t.Helper()
	if res != nil {
		t.Errorf("Result is non-nil on error %v", err)
	}
	if !errors.Is(err, want) {
		t.Errorf("err = %v, want %v", err, want)
		return
	}
	if n := sentinelCount(err); n != 1 {
		t.Errorf("err %q matches %d sentinels, want exactly 1", err, n)
	}
}

func parseString(s string) (*Result, error) {
	return Parse(strings.NewReader(s), 25<<20, "")
}

// repeatReader streams head, n copies of item and tail without ever holding
// the whole document, so the cap tests never build a multi-megabyte slice.
type repeatReader struct {
	head, item, tail string
	n                int
	stage, idx, off  int
}

func (r *repeatReader) Read(p []byte) (int, error) {
	total := 0
	for total < len(p) {
		var src string
		switch r.stage {
		case 0:
			src = r.head
		case 1:
			if r.idx >= r.n {
				r.stage, r.off = 2, 0
				continue
			}
			src = r.item
		case 2:
			src = r.tail
		default:
			if total == 0 {
				return 0, io.EOF
			}
			return total, nil
		}
		k := copy(p[total:], src[r.off:])
		total += k
		r.off += k
		if r.off == len(src) {
			r.off = 0
			if r.stage == 1 {
				r.idx++
			} else {
				r.stage++
			}
		}
	}
	return total, nil
}

const onePoint = `<trk><trkseg><trkpt lat="47" lon="11"/></trkseg></trk>`

func readFixture10(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(corpusRoot, "10-realistic-track", "input.gpx"))
	if err != nil {
		t.Fatalf("fixture 10: %v", err)
	}
	return string(b)
}

func TestTruncated(t *testing.T) {
	full := readFixture10(t)
	if _, err := parseString(full); err != nil {
		t.Fatalf("the intact fixture must parse: %v", err)
	}
	// Three places to cut: in the middle of text, in the middle of a tag, and
	// right after a complete child.
	cuts := map[string]int{
		"mid-text":                   strings.Index(full, "Alpine Ridge Loop") + 6,
		"mid-tag":                    strings.Index(full, "<trkpt") + 8,
		"after a complete </trkpt>":  strings.Index(full, "</trkpt>") + len("</trkpt>"),
		"inside the xml declaration": 20,
		"inside an end tag":          strings.Index(full, "</gpx>") + 4,
	}
	for name, at := range cuts {
		t.Run(name, func(t *testing.T) {
			res, err := parseString(full[:at])
			wantErr(t, res, err, ErrTruncated)
		})
	}

	// The classification rests on the stdlib reporting a truncated document as
	// a SyntaxError with exactly this message. If a Go release changes it,
	// this test fails instead of truncated files turning into ErrMalformed.
	t.Run("stdlib still reports unexpected EOF", func(t *testing.T) {
		d := xml.NewDecoder(strings.NewReader(full[:strings.Index(full, "</trkpt>")+len("</trkpt>")]))
		d.Strict = false
		d.Entity = xml.HTMLEntity
		var err error
		for err == nil {
			_, err = d.Token()
		}
		var se *xml.SyntaxError
		if !errors.As(err, &se) || se.Msg != "unexpected EOF" {
			t.Fatalf("decoder error = %#v, want *xml.SyntaxError with Msg %q", err, "unexpected EOF")
		}
	})
}

func TestNoTrack(t *testing.T) {
	cases := map[string]string{
		"only waypoints": `<gpx><wpt lat="47" lon="11"><name>x</name></wpt><wpt lat="47.1" lon="11"/></gpx>`,
		"trk with only positionless points": `<gpx><trk><trkseg>` +
			`<trkpt><time>2024-05-01T10:00:00Z</time></trkpt><trkpt lat="N/A" lon="11"/></trkseg></trk></gpx>`,
		"empty trk beside a populated rte": `<gpx><trk/><rte><rtept lat="47" lon="11"/><rtept lat="47.1" lon="11"/></rte></gpx>`,
		"trk without points":               `<gpx><trk><name>x</name><trkseg/></trk></gpx>`,
		"bare gpx":                         `<gpx/>`,
	}
	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			res, err := parseString(doc)
			wantErr(t, res, err, ErrNoTrack)
		})
	}
}

func TestMalformed(t *testing.T) {
	cases := map[string]string{
		"empty input":           "",
		"whitespace only":       " \n\t \r\n",
		"text without a root":   "this is not a gpx file",
		"only a declaration":    `<?xml version="1.0"?>`,
		"only a comment":        `<!-- nothing here -->`,
		"attribute without end": `<gpx a=">`,
	}
	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			res, err := parseString(doc)
			if errors.Is(err, ErrTruncated) {
				wantErr(t, res, err, ErrTruncated)
				return
			}
			wantErr(t, res, err, ErrMalformed)
		})
	}
	t.Run("empty and whitespace-only are malformed, not truncated", func(t *testing.T) {
		for _, doc := range []string{"", " \n\t "} {
			res, err := parseString(doc)
			wantErr(t, res, err, ErrMalformed)
		}
	})
}

func TestLimits(t *testing.T) {
	// nesting hides inside <extensions>, which the builder skips, so the depth
	// check has to sit in the token loop.
	nested := func(depth int) string {
		k := depth - 2 // <gpx> and <extensions> take two levels
		return `<gpx>` + onePoint + `<extensions>` + strings.Repeat("<a>", k) + strings.Repeat("</a>", k) + `</extensions></gpx>`
	}
	t.Run("nesting to exactly maxDepth parses", func(t *testing.T) {
		if _, err := parseString(nested(maxDepth)); err != nil {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("nesting past maxDepth is malformed", func(t *testing.T) {
		res, err := parseString(nested(maxDepth + 1))
		wantErr(t, res, err, ErrMalformed)
	})
	t.Run("a deep unclosed run is malformed, not unbounded", func(t *testing.T) {
		res, err := Parse(&repeatReader{head: "<gpx>", item: "<a>", n: 100_000, tail: ""}, 25<<20, "")
		wantErr(t, res, err, ErrMalformed)
	})

	points := func(n int) io.Reader {
		return &repeatReader{head: `<gpx><trk><trkseg>`, item: `<trkpt lat="47.0" lon="11.0"/>`, n: n,
			tail: `</trkseg></trk></gpx>`}
	}
	t.Run("exactly maxPoints parses", func(t *testing.T) {
		res, err := Parse(points(maxPoints), 64<<20, "")
		if err != nil {
			t.Fatalf("got %v", err)
		}
		if res.PointCount != maxPoints {
			t.Errorf("PointCount = %d, want %d", res.PointCount, maxPoints)
		}
	})
	t.Run("points past maxPoints are too large", func(t *testing.T) {
		res, err := Parse(points(maxPoints+1), 64<<20, "")
		wantErr(t, res, err, ErrTooLarge)
	})
	t.Run("route points count towards maxPoints", func(t *testing.T) {
		res, err := Parse(&repeatReader{head: `<gpx><rte>`, item: `<rtept lat="47.0" lon="11.0"/>`, n: maxPoints + 1,
			tail: `</rte></gpx>`}, 64<<20, "")
		wantErr(t, res, err, ErrTooLarge)
	})
	t.Run("positionless points are not stored and not counted", func(t *testing.T) {
		res, err := Parse(&repeatReader{head: `<gpx><trk><trkseg><trkpt lat="47" lon="11"/>`, item: `<trkpt lat="N/A" lon="11"/>`,
			n: maxPoints + 10, tail: `</trkseg></trk></gpx>`}, 64<<20, "")
		if err != nil {
			t.Fatalf("got %v", err)
		}
		if res.PointCount != 1 {
			t.Errorf("PointCount = %d, want 1", res.PointCount)
		}
	})

	segments := func(n int, item string) io.Reader {
		return &repeatReader{head: `<gpx><trk>`, item: item, n: n, tail: `</trk></gpx>`}
	}
	seg := `<trkseg><trkpt lat="47" lon="11"/></trkseg>`
	t.Run("exactly maxSegments parses", func(t *testing.T) {
		res, err := Parse(segments(maxSegments, seg), 64<<20, "")
		if err != nil {
			t.Fatalf("got %v", err)
		}
		if res.PointCount != maxSegments {
			t.Errorf("PointCount = %d, want %d", res.PointCount, maxSegments)
		}
	})
	t.Run("segments past maxSegments are too large", func(t *testing.T) {
		res, err := Parse(segments(maxSegments+1, seg), 64<<20, "")
		wantErr(t, res, err, ErrTooLarge)
	})
	t.Run("empty segments are not stored", func(t *testing.T) {
		res, err := Parse(&repeatReader{head: `<gpx><trk><trkseg><trkpt lat="47" lon="11"/></trkseg>`, item: `<trkseg/>`,
			n: 3 * maxSegments, tail: `</trk></gpx>`}, 64<<20, "")
		if err != nil {
			t.Fatalf("got %v", err)
		}
		if res.PointCount != 1 {
			t.Errorf("PointCount = %d, want 1", res.PointCount)
		}
	})

	waypoints := func(n int) io.Reader {
		return &repeatReader{head: `<gpx>`, item: `<wpt lat="47" lon="11"/>`, n: n, tail: onePoint + `</gpx>`}
	}
	t.Run("exactly maxWaypoints parses", func(t *testing.T) {
		res, err := Parse(waypoints(maxWaypoints), 64<<20, "")
		if err != nil {
			t.Fatalf("got %v", err)
		}
		if len(res.Waypoints) != maxWaypoints {
			t.Errorf("%d waypoints, want %d", len(res.Waypoints), maxWaypoints)
		}
	})
	t.Run("waypoints past maxWaypoints are too large", func(t *testing.T) {
		res, err := Parse(waypoints(maxWaypoints+1), 64<<20, "")
		wantErr(t, res, err, ErrTooLarge)
	})
	t.Run("unusable waypoints are not stored and not counted", func(t *testing.T) {
		res, err := Parse(&repeatReader{head: `<gpx>`, item: `<wpt lat="N/A" lon="11"/>`, n: maxWaypoints + 10,
			tail: onePoint + `</gpx>`}, 64<<20, "")
		if err != nil {
			t.Fatalf("got %v", err)
		}
		if len(res.Waypoints) != 0 {
			t.Errorf("%d waypoints, want 0", len(res.Waypoints))
		}
	})

	// 8 MiB of retained text is exactly 8192 chunks of 1 KiB. CDATA chunks are
	// separate tokens, so the budget has to be charged as they arrive.
	chunk := "<![CDATA[" + strings.Repeat("x", 1024-len("<![CDATA[]]>")) + "]]>"
	if len(chunk) != 1024 {
		t.Fatalf("test chunk is %d bytes", len(chunk))
	}
	descText := func(chunks int, extra string) io.Reader {
		return &repeatReader{head: `<gpx><metadata><desc>`, item: chunk, n: chunks,
			tail: extra + `</desc></metadata>` + onePoint + `</gpx>`}
	}
	chunkText := 1024 - len("<![CDATA[]]>")
	t.Run("retained text of exactly maxTextBytes parses", func(t *testing.T) {
		n := maxTextBytes / chunkText
		res, err := Parse(descText(n, ""), 64<<20, "")
		if err != nil {
			t.Fatalf("got %v", err)
		}
		if len(res.Description) != n*chunkText {
			t.Errorf("Description length %d, want %d", len(res.Description), n*chunkText)
		}
	})
	t.Run("retained text past maxTextBytes is too large", func(t *testing.T) {
		res, err := Parse(descText(maxTextBytes/chunkText+2, ""), 64<<20, "")
		wantErr(t, res, err, ErrTooLarge)
	})
	t.Run("the text budget is shared between elements", func(t *testing.T) {
		half := maxTextBytes / chunkText / 2
		doc := &repeatReader{head: `<gpx><metadata><name>`, item: chunk, n: half + 1,
			tail: `</name><desc>` + strings.Repeat(chunk, half+1) + `</desc></metadata>` + onePoint + `</gpx>`}
		res, err := Parse(doc, 64<<20, "")
		wantErr(t, res, err, ErrTooLarge)
	})

	t.Run("an ele of 10000 CDATA chunks is unusable without an error", func(t *testing.T) {
		huge := strings.Repeat("<![CDATA[1]]>", 10000)
		doc := `<gpx><trk><trkseg>` +
			`<trkpt lat="47.0" lon="11"><ele>100</ele></trkpt>` +
			`<trkpt lat="47.001" lon="11"><ele>` + huge + `</ele></trkpt>` +
			`<trkpt lat="47.002" lon="11"><ele>120</ele></trkpt>` +
			`</trkseg></trk></gpx>`
		d := xml.NewDecoder(strings.NewReader(doc))
		d.Strict = false
		b := newBuilder()
		longest := 0
		for {
			tok, err := d.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			switch tk := tok.(type) {
			case xml.StartElement:
				b.start(tk)
			case xml.EndElement:
				b.end(tk.Name.Local)
			case xml.CharData:
				b.text(tk)
			}
			if len(b.buf) > longest {
				longest = len(b.buf)
			}
			if b.err != nil {
				t.Fatalf("builder error %v", b.err)
			}
		}
		if longest > maxNumberText {
			t.Errorf("numeric text buffer reached %d bytes, want at most %d", longest, maxNumberText)
		}
		res, err := b.finish("")
		if err != nil {
			t.Fatal(err)
		}
		ele := res.tracks[0].segments[0].ele
		if len(ele) != 3 || ele[0] != 100 || !math.IsNaN(ele[1]) || ele[2] != 120 {
			t.Errorf("ele = %v, want [100 NaN 120]", ele)
		}
	})
	t.Run("the same through Parse", func(t *testing.T) {
		huge := strings.Repeat("<![CDATA[1]]>", 10000)
		doc := `<gpx><trk><trkseg><trkpt lat="47" lon="11"><ele>` + huge + `</ele><time>` + huge + `</time></trkpt></trkseg></trk></gpx>`
		res, err := parseString(doc)
		if err != nil {
			t.Fatalf("got %v", err)
		}
		if res.PointCount != 1 || res.HasElevation {
			t.Errorf("PointCount %d HasElevation %v", res.PointCount, res.HasElevation)
		}
	})
}

func TestNoInputEcho(t *testing.T) {
	const marker = "ZZ-SECRET-MARKER"
	cases := map[string]string{
		"marker as an unterminated attribute": `<gpx ` + marker + `="`,
		"marker in an unclosed tag":           `<gpx><trk><trkseg><trkpt lat="` + marker,
		"marker as the only text":             marker,
		"marker in a bad entity":              `<gpx><name>&` + marker + `;</name><` + marker + ` ="></gpx>`,
		"marker in a comment before nothing":  `<!-- ` + marker + ` -->`,
		"marker in deep element names":        `<gpx>` + strings.Repeat("<"+marker+">", maxDepth+1),
		"marker in a waypoint-only file":      `<gpx><wpt lat="47" lon="11"><name>` + marker + `</name></wpt></gpx>`,
		"marker in a processing instruction":  `<?` + marker + ` ?>`,
		"marker in an over-long text run":     `<gpx><desc>` + marker + strings.Repeat("x", maxRun) + `</desc></gpx>`,
		"marker in the declared charset":      `<?xml version="1.0" encoding="` + marker + `"?><gpx>`,
		"marker in an end tag":                `<gpx></` + marker + `>`,
	}
	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			res, err := parseString(doc)
			if err == nil {
				t.Fatalf("expected an error, got a Result %+v", res)
			}
			if res != nil {
				t.Error("Result is non-nil on error")
			}
			if strings.Contains(err.Error(), marker) {
				t.Errorf("error echoes input: %q", err.Error())
			}
			if n := sentinelCount(err); n != 1 {
				t.Errorf("error %q matches %d sentinels, want exactly 1", err, n)
			}
		})
	}
}

func TestParseNumber(t *testing.T) {
	rejected := []string{
		"NaN", "nan", "Inf", "+Inf", "-inf", "infinity", "0x1p4", "0X1.8p1", "1_000", "1e400", "-1e400",
		"1,5", "\u0661\u0662\u0663", "1.5f", "", "   ", "&nbsp;", "--1", "1e", "e5", ".", "+", "1.2.3",
		"1 2", strings.Repeat("1", 300), "0." + strings.Repeat("0", 300) + "1",
	}
	for _, in := range rejected {
		if v, ok := parseNumber(in); ok {
			t.Errorf("parseNumber(%.30q) = (%v, true), want rejected", in, v)
		}
	}
	accepted := []struct {
		in   string
		want float64
	}{
		{"4.7E1", 47}, {" 1.5 ", 1.5}, {"\u00a01.5\u00a0", 1.5}, {"\t-2.25\n", -2.25}, {"+3", 3}, {".5", 0.5}, {"5.", 5},
		{"1e-400", 0}, {"0", 0}, {"1E+2", 100}, {strings.Repeat("1", 200), 1.1111111111111111e199},
	}
	for _, tc := range accepted {
		v, ok := parseNumber(tc.in)
		if !ok || v != tc.want {
			t.Errorf("parseNumber(%.30q) = (%v, %v), want (%v, true)", tc.in, v, ok, tc.want)
		}
	}
	if v, ok := parseNumber("-0"); !ok || v != 0 || !math.Signbit(v) {
		t.Errorf("parseNumber(\"-0\") = (%v, %v), want negative zero accepted", v, ok)
	}

	// Differential test against the reference rule: the decimal grammar, then
	// ParseFloat, then a finite result. The grammar is hand-rolled here so the
	// check does not share the production regexp.
	refGrammar := func(s string) bool {
		i, n := 0, len(s)
		digits := func() int {
			start := i
			for i < n && s[i] >= '0' && s[i] <= '9' {
				i++
			}
			return i - start
		}
		if i < n && (s[i] == '+' || s[i] == '-') {
			i++
		}
		switch {
		case i < n && s[i] == '.':
			i++
			if digits() == 0 {
				return false
			}
		default:
			if digits() == 0 {
				return false
			}
			if i < n && s[i] == '.' {
				i++
				digits()
			}
		}
		if i < n && (s[i] == 'e' || s[i] == 'E') {
			i++
			if i < n && (s[i] == '+' || s[i] == '-') {
				i++
			}
			if digits() == 0 {
				return false
			}
		}
		return i == n
	}
	reference := func(raw string) (float64, bool) {
		s := strings.TrimSpace(raw)
		if len(s) > maxNumberText || !refGrammar(s) {
			return 0, false
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
			return 0, false
		}
		return f, true
	}
	alphabet := []string{"0", "1", "2", "5", "9", "+", "-", ".", "e", "E", "x", "_", ",", " ", "\u00a0", "N", "a", "\u0663"}
	rng := rand.New(rand.NewPCG(3, 4))
	for i := 0; i < 10000; i++ {
		var sb strings.Builder
		for j, n := 0, 1+rng.IntN(8); j < n; j++ {
			sb.WriteString(alphabet[rng.IntN(len(alphabet))])
		}
		in := sb.String()
		gv, gok := parseNumber(in)
		wv, wok := reference(in)
		if gok != wok || (gok && math.Float64bits(gv) != math.Float64bits(wv)) {
			t.Fatalf("parseNumber(%q) = (%v, %v), reference (%v, %v)", in, gv, gok, wv, wok)
		}
	}
}

func TestDropsBadPositions(t *testing.T) {
	good := func(i int) string {
		return fmt.Sprintf(`<trkpt lat="%.3f" lon="%.3f"/>`, 47+float64(i)*0.001, 11+float64(i)*0.001)
	}
	bad := []string{
		`<trkpt lat="" lon="11"/>`,
		`<trkpt lat="N/A" lon="11"/>`,
		`<trkpt lat="1e999" lon="11"/>`,
		`<trkpt lat="47" lon="NaN"/>`,
		`<trkpt lat="47"/>`,
		`<trkpt lon="11"/>`,
		`<trkpt/>`,
		`<trkpt lat="  " lon="  "/>`,
		`<trkpt lat="47" lon="1e999"/>`,
		`<trkpt lat="Infinity" lon="11"/>`,
	}
	var sb strings.Builder
	sb.WriteString(`<gpx><trk><trkseg>`)
	valid := 0
	for i, b := range bad {
		sb.WriteString(good(i))
		valid++
		sb.WriteString(b)
	}
	sb.WriteString(good(len(bad)))
	valid++
	sb.WriteString(`</trkseg></trk>`)
	// A route-only twin and a waypoint, to cover the other two position paths.
	sb.WriteString(`<wpt lat="N/A" lon="11"><name>bad</name></wpt><wpt lat="47.5" lon="11.5"><name>good</name></wpt></gpx>`)

	res, err := parseString(sb.String())
	if err != nil {
		t.Fatalf("got %v", err)
	}
	if res.PointCount != valid {
		t.Errorf("PointCount = %d, want %d", res.PointCount, valid)
	}
	if res.Bounds.MinLat < 47 || res.Bounds.MinLon < 11 || res.Bounds.MaxLat > 47.02 {
		t.Errorf("Bounds %+v include a point that is not one of the valid ones (a 0,0 point?)", res.Bounds)
	}
	if res.Start == (LatLon{}) || res.Start.Lat < 47 {
		t.Errorf("Start = %+v", res.Start)
	}
	if len(res.Waypoints) != 1 || res.Waypoints[0].Name != "good" {
		t.Errorf("Waypoints = %+v, want just the good one", res.Waypoints)
	}
	for _, tr := range res.tracks {
		for _, s := range tr.segments {
			for i := range s.lat {
				if s.lat[i] == 0 && s.lon[i] == 0 {
					t.Fatalf("a point sits at 0,0")
				}
			}
		}
	}
}
