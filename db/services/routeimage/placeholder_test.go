package routeimage

import (
	"bytes"
	"encoding/xml"
	"github.com/tkrajina/gpxgo/gpx"
	"testing"
)

const routeGPX = `<gpx version="1.1"><rte><rtept lat="46.6" lon="7.2"/><rtept lat="46.7" lon="7.3"/></rte></gpx>`

func TestSegmentsFallsBackToRoutes(t *testing.T) {
	data, err := gpx.ParseBytes([]byte(routeGPX))
	if err != nil {
		t.Fatal(err)
	}
	segments := Segments(data)
	if len(segments) != 1 || len(segments[0]) != 2 {
		t.Fatalf("segments = %v, want the route's two points", segments)
	}
}

func TestRenderEdgeCases(t *testing.T) {
	cases := map[string]struct {
		segments [][]Point
		route    bool
	}{
		"no segments":      {nil, false},
		"single point":     {[][]Point{{{46.6, 7.2}}}, true},
		"north-south only": {[][]Point{{{46.6, 7.2}, {46.7, 7.2}}}, true},
		"invalid points":   {[][]Point{{{91, 7.2}, {46.6, 181}}}, false},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			data := Render(c.segments, 1)
			checkSVG(t, data)
			if got := bytes.Contains(data, []byte(`class="route"`)); got != c.route {
				t.Fatalf("has route = %v, want %v", got, c.route)
			}
			if bytes.Contains(data, []byte("NaN")) || bytes.Contains(data, []byte("Inf")) {
				t.Fatal("the SVG holds an invalid number")
			}
		})
	}
}

// The route is fitted into the centred square, so the square crop of the
// trail list keeps all of it.
func TestRouteFitsCentreSquare(t *testing.T) {
	wide := [][]Point{{{46.6, 6.0}, {46.6, 9.0}}}
	for _, line := range project(wide) {
		for _, p := range line {
			left := float64(Width-Height) / 2
			if p.x < left || p.x > left+Height || p.y < 0 || p.y > Height {
				t.Fatalf("point %v lies outside the centre square", p)
			}
		}
	}
}

func TestSimplify(t *testing.T) {
	line := []vec{{0, 0}, {1, 0.1}, {2, 0}, {3, 5}, {4, 0}}
	got := simplify(line, 0.5)
	want := []vec{{0, 0}, {2, 0}, {3, 5}, {4, 0}}
	if len(got) != len(want) {
		t.Fatalf("simplify = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("simplify = %v, want %v", got, want)
		}
	}
}

func checkSVG(t *testing.T, data []byte) {
	t.Helper()
	var root struct {
		XMLName xml.Name
		Width   int `xml:"width,attr"`
		Height  int `xml:"height,attr"`
	}
	if err := xml.Unmarshal(data, &root); err != nil {
		t.Fatalf("not well-formed XML: %v", err)
	}
	if root.XMLName.Local != "svg" || root.Width != Width || root.Height != Height {
		t.Fatalf("root = %s %dx%d, want svg %dx%d", root.XMLName.Local, root.Width, root.Height, Width, Height)
	}
}
