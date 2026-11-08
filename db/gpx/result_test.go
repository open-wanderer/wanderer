package gpx

import (
	"fmt"
	"strings"
	"testing"
)

// gpxDoc builds a GPX document around the given body. These tests feed their
// documents through the real Parse, so they are end-to-end cases, not struct
// fakes.
func gpxDoc(metadata, body string) string {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<gpx version="1.1" creator="test" xmlns="http://www.topografix.com/GPX/1/1">` + "\n")
	if metadata != "" {
		sb.WriteString("<metadata>" + metadata + "</metadata>\n")
	}
	sb.WriteString(body)
	sb.WriteString("\n</gpx>\n")
	return sb.String()
}

// trkpts renders n track points 0.001 degrees (about 111 m) apart. ele returns
// the <ele> text of point i, or "" for a point without one; tm does the same
// for <time>.
func trkpts(n int, ele, tm func(i int) string) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&sb, `<trkpt lat="%.6f" lon="11.0">`, 47+float64(i)*0.001)
		if ele != nil {
			if e := ele(i); e != "" {
				sb.WriteString("<ele>" + e + "</ele>")
			}
		}
		if tm != nil {
			if v := tm(i); v != "" {
				sb.WriteString("<time>" + v + "</time>")
			}
		}
		sb.WriteString("</trkpt>\n")
	}
	return sb.String()
}

func trk(name, points string) string {
	n := ""
	if name != "" {
		n = "<name>" + name + "</name>"
	}
	return "<trk>" + n + "<trkseg>\n" + points + "</trkseg></trk>"
}

func mustParse(t *testing.T, doc, fallback string) *Result {
	t.Helper()
	res, err := Parse(strings.NewReader(doc), parseLimit, fallback)
	if err != nil {
		t.Fatalf("Parse(%.80q): %v", doc, err)
	}
	return res
}

func TestHasElevation(t *testing.T) {
	ramp := func(i int) string { return fmt.Sprint(100 + 10*i) }
	partial := func(withEle int) func(int) string {
		return func(i int) string {
			if i < withEle {
				return ramp(i)
			}
			return ""
		}
	}
	rteOnlyEle := `<rte><rtept lat="-33.0" lon="151.0"><ele>10</ele></rtept>` +
		`<rtept lat="-33.001" lon="151.0"><ele>40</ele></rtept></rte>`

	for _, tt := range []struct {
		name string
		doc  string
		want bool
	}{
		{"no ele at all", gpxDoc("", trk("t", trkpts(10, nil, nil))), false},
		{"499 of 1000", gpxDoc("", trk("t", trkpts(1000, partial(499), nil))), false},
		{"500 of 1000 is exactly half", gpxDoc("", trk("t", trkpts(1000, partial(500), nil))), true},
		{"all zero is identical", gpxDoc("", trk("t", trkpts(10, func(int) string { return "0" }, nil))), false},
		{"all 1000 is identical", gpxDoc("", trk("t", trkpts(10, func(int) string { return "1000" }, nil))), false},
		{"single point with ele", gpxDoc("", trk("t", trkpts(1, func(int) string { return "500" }, nil))), false},
		{"a literal 0 is a usable elevation", gpxDoc("", trk("t", trkpts(4, func(i int) string { return []string{"0", "5", "0", "5"}[i] }, nil))), true},
		{"real profile", gpxDoc("", trk("t", trkpts(20, ramp, nil))), true},
		{"unparseable ele is not usable", gpxDoc("", trk("t", trkpts(4, func(int) string { return "high" }, nil))), false},
		{"mixed file counts trk points only", gpxDoc("", rteOnlyEle+trk("t", trkpts(5, nil, nil))), false},
		{"mixed file with trk eles", gpxDoc("", trk("t", trkpts(5, ramp, nil))+rteOnlyEle), true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res := mustParse(t, tt.doc, "")
			if res.HasElevation != tt.want {
				t.Fatalf("HasElevation = %v, want %v (input %.80q)", res.HasElevation, tt.want, tt.doc)
			}
		})
	}

	t.Run("flag never zeroes gain and loss", func(t *testing.T) {
		// 400 of 1000 points carry a climbing profile: the flag is false, but the
		// measured gain is whatever the points say.
		doc := gpxDoc("", trk("t", trkpts(1000, partial(400), nil)))
		res := mustParse(t, doc, "")
		if res.HasElevation {
			t.Fatalf("HasElevation = true, want false for 400 of 1000")
		}
		if !(res.ElevationGain > 0) {
			t.Fatalf("ElevationGain = %v, want it left unchanged and above 0", res.ElevationGain)
		}
	})
}

func TestNameFallback(t *testing.T) {
	meta := func(n string) string { return "<name>" + n + "</name>" }
	rte := func(n string) string {
		return `<rte><name>` + n + `</name><rtept lat="47.0" lon="11.0"/></rte>`
	}
	pts := trkpts(2, nil, nil)
	for _, tt := range []struct {
		name, doc, fallback, want string
	}{
		{"metadata wins", gpxDoc(meta("M"), trk("T", pts)+rte("R")), "F", "M"},
		{"empty metadata falls to trk", gpxDoc(meta(""), trk("T", pts)+rte("R")), "F", "T"},
		{"blank metadata falls to trk", gpxDoc(meta("   "), trk("T", pts)), "F", "T"},
		{"no metadata falls to trk", gpxDoc("", trk("T", pts)), "F", "T"},
		{"empty trk falls to rte", gpxDoc("", trk("", pts)+rte("R")), "F", "R"},
		{"all empty falls to fallback", gpxDoc("", trk("", pts)), "F", "F"},
		{"all empty and no fallback is empty", gpxDoc("", trk("", pts)), "", ""},
		{"name is trimmed", gpxDoc(meta("  Alpine \n"), trk("", pts)), "", "Alpine"},
		{"cdata name", gpxDoc("<name><![CDATA[A & B]]></name>", trk("", pts)), "", "A & B"},
		{"author name is never read", gpxDoc(`<author><name>Someone</name></author>`, trk("", pts)), "", ""},
		{"only the first trk names the file", gpxDoc("", trk("", pts)+trk("Second", pts)), "F", "F"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res := mustParse(t, tt.doc, tt.fallback)
			if res.Name != tt.want {
				t.Fatalf("Name = %.80q, want %.80q (input %.80q)", res.Name, tt.want, tt.doc)
			}
		})
	}

	t.Run("description is metadata desc", func(t *testing.T) {
		res := mustParse(t, gpxDoc("<desc> Nice walk </desc>", trk("T", pts)), "")
		if res.Description != "Nice walk" {
			t.Fatalf("Description = %.80q, want %q", res.Description, "Nice walk")
		}
	})
}

func TestStartDate(t *testing.T) {
	at := func(first, last string) string {
		return gpxDoc("", trk("t", trkpts(3, nil, func(i int) string {
			switch i {
			case 0:
				return first
			case 2:
				return last
			}
			return ""
		})))
	}
	for _, tt := range []struct {
		name, doc, want string
	}{
		{"first and last time", at("2024-05-01T10:00:00Z", "2024-05-01T11:00:00Z"), "2024-05-01"},
		{"last point has no time", at("2024-05-01T10:00:00Z", ""), ""},
		{"first point has no time", at("", "2024-05-01T11:00:00Z"), ""},
		{"no times", at("", ""), ""},
		{"date is UTC", at("2024-05-01T23:30:00-02:00", "2024-05-02T01:00:00Z"), "2024-05-02"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res := mustParse(t, tt.doc, "")
			if res.StartDate != tt.want {
				t.Fatalf("StartDate = %q, want %q (input %.80q)", res.StartDate, tt.want, tt.doc)
			}
		})
	}
}

func TestWaypoints(t *testing.T) {
	body := trk("t", trkpts(2, nil, nil))
	wpt := func(attrs, children string) string { return "<wpt " + attrs + ">" + children + "</wpt>\n" }
	for _, tt := range []struct {
		name, wpts string
		want       []Waypoint
	}{
		{"known sym", wpt(`lat="47.5" lon="11.5"`, `<name>Camp</name><sym>campground</sym>`),
			[]Waypoint{{Lat: 47.5, Lon: 11.5, Name: "Camp", Icon: "campground"}}},
		{"sym is trimmed", wpt(`lat="47.5" lon="11.5"`, `<sym> campground </sym>`),
			[]Waypoint{{Lat: 47.5, Lon: 11.5, Icon: "campground"}}},
		{"unknown sym gives no icon", wpt(`lat="47.5" lon="11.5"`, `<sym>airbnb</sym>`),
			[]Waypoint{{Lat: 47.5, Lon: 11.5}}},
		{"hostile sym gives no icon", wpt(`lat="47.5" lon="11.5"`, `<sym>"><script>x</script></sym>`),
			[]Waypoint{{Lat: 47.5, Lon: 11.5}}},
		{"description kept", wpt(`lat="47.5" lon="11.5"`, `<desc>Water here</desc>`),
			[]Waypoint{{Lat: 47.5, Lon: 11.5, Description: "Water here"}}},
		{"bad lat is dropped, never 0,0", wpt(`lat="bad" lon="11.0"`, `<name>x</name>`), nil},
		{"missing lon is dropped", wpt(`lat="47.0"`, `<name>x</name>`), nil},
		{"file order kept", wpt(`lat="47.1" lon="11.1"`, `<name>a</name>`) + wpt(`lat="47.2" lon="11.2"`, `<name>b</name>`),
			[]Waypoint{{Lat: 47.1, Lon: 11.1, Name: "a"}, {Lat: 47.2, Lon: 11.2, Name: "b"}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res := mustParse(t, gpxDoc("", tt.wpts+body), "")
			if len(res.Waypoints) != len(tt.want) {
				t.Fatalf("got %d waypoints %+v, want %d", len(res.Waypoints), res.Waypoints, len(tt.want))
			}
			for i, w := range tt.want {
				if res.Waypoints[i] != w {
					t.Fatalf("waypoint %d = %+v, want %+v", i, res.Waypoints[i], w)
				}
			}
		})
	}
}
