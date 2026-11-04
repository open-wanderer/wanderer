// Package routeimage renders the placeholder image for trails and summit logs
// without photos: the route on the Polar Night background with the contour
// lines of the docs hero. It needs no tiles, so it works offline and for every
// import path.
//
// The image is an SVG. The browser draws it sharp at every size, strokes keep
// their width however small the image is shown, and it follows the page's
// color scheme. Presentation attributes carry the light style for renderers
// that ignore <style>; the dark style overrides them in a media query.
package routeimage

import (
	"fmt"
	"math"
	"strings"
)

const (
	Width  = 1200
	Height = 800

	// The route is fitted into the centred square, so square crops of the
	// image (the trail list) keep all of it. padding is the share of the
	// square kept free on each side.
	padding = 0.14

	// Stroke widths in screen pixels: non-scaling-stroke keeps them the same
	// at every size the image is shown at.
	routeWidth  = 2.5
	casingWidth = 6.0
	dotSize     = 8.0
	ringSize    = 13.0

	// tolerance is how far, in image units, the drawn route may stray from
	// the recorded one.
	tolerance = 0.5
)

type Point struct {
	Lat float64
	Lon float64
}

const style = `@media (prefers-color-scheme: dark) {
.bg0 { stop-color: #0b0d16 }
.bg1 { stop-color: #12141c }
.bg2 { stop-color: #1b1e2c }
.bg3 { stop-color: #242734 }
.contours { stroke: #fff; opacity: 0.625 }
.casing { stroke: #0b0d16; stroke-opacity: 0.6 }
.route, .end { stroke: #fff }
.ring { stroke: #0b0d16 }
}`

// Render draws the route. Each segment is drawn as its own line, so gaps
// between GPX segments stay gaps. seed picks the contour pattern; the same
// seed always gives the same background.
func Render(segments [][]Point, seed float64) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`, Width, Height, Width, Height)
	b.WriteString("<style>" + style + "</style>")
	// The light gradient of the app's welcome screen; the dark one is the
	// docs hero.
	b.WriteString(`<defs><linearGradient id="bg" x1="0" y1="0" x2="0" y2="1">` +
		`<stop offset="0" class="bg0" stop-color="#f8f9fc"/>` +
		`<stop offset="0.3" class="bg1" stop-color="#eff1f6"/>` +
		`<stop offset="0.62" class="bg2" stop-color="#e2e5ee"/>` +
		`<stop offset="1" class="bg3" stop-color="#d5d9e4"/>` +
		`</linearGradient></defs>`)
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="url(#bg)"/>`, Width, Height)
	writeContours(&b, seed)

	lines := project(segments)
	if len(lines) > 0 {
		d := pathData(lines)
		writeStroke(&b, "casing", d, "#ffffff", casingWidth, ` stroke-opacity="0.7"`)
		writeStroke(&b, "route", d, "#12141c", routeWidth, "")

		first := lines[0][0]
		lastLine := lines[len(lines)-1]
		last := lastLine[len(lastLine)-1]
		if first.dist(last) > 0.03*Height {
			writeMarker(&b, last, "end", "#12141c")
		}
		writeMarker(&b, first, "start", "#4ade80")
	}

	b.WriteString("</svg>")
	return []byte(b.String())
}

func writeStroke(b *strings.Builder, class, d, color string, width float64, extra string) {
	fmt.Fprintf(b, `<path class="%s" d="%s" fill="none" stroke="%s" stroke-width="%s" stroke-linecap="round" stroke-linejoin="round" vector-effect="non-scaling-stroke"%s/>`,
		class, d, color, num(width), extra)
}

// writeMarker draws a dot with a ring as zero-length lines with round caps,
// so non-scaling-stroke keeps them the same size as the route.
func writeMarker(b *strings.Builder, p vec, class, color string) {
	d := fmt.Sprintf("M%s %sh0", num(p.x), num(p.y))
	writeStroke(b, "ring", d, "#ffffff", ringSize, "")
	writeStroke(b, class, d, color, dotSize, "")
}

func pathData(lines [][]vec) string {
	var b strings.Builder
	for _, line := range lines {
		for i, p := range line {
			switch i {
			case 0:
				b.WriteString("M")
			case 1:
				b.WriteString("L")
			default:
				b.WriteString(" ")
			}
			b.WriteString(num(p.x) + " " + num(p.y))
		}
		if len(line) == 1 {
			b.WriteString("h0")
		}
	}
	return b.String()
}

// num prints a coordinate with one decimal, which is finer than any screen
// the image is shown on.
func num(v float64) string {
	s := fmt.Sprintf("%.1f", v)
	s = strings.TrimSuffix(s, ".0")
	if s == "-0" {
		return "0"
	}
	return s
}

// project maps the segments into image space with Web Mercator, fits them
// into the padded centre square and simplifies them to what can be seen.
func project(segments [][]Point) [][]vec {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	merc := make([][]vec, 0, len(segments))
	for _, seg := range segments {
		line := make([]vec, 0, len(seg))
		for _, p := range seg {
			if math.IsNaN(p.Lat) || math.IsNaN(p.Lon) || math.Abs(p.Lat) > 85 || math.Abs(p.Lon) > 180 {
				continue
			}
			v := mercator(p)
			minX, maxX = math.Min(minX, v.x), math.Max(maxX, v.x)
			minY, maxY = math.Min(minY, v.y), math.Max(maxY, v.y)
			line = append(line, v)
		}
		if len(line) > 0 {
			merc = append(merc, line)
		}
	}
	if len(merc) == 0 {
		return nil
	}

	spanX, spanY := maxX-minX, maxY-minY
	avail := Height * (1 - 2*padding)
	scale := 0.0
	if span := math.Max(spanX, spanY); span > 0 {
		scale = avail / span
	}
	offX := (Width - spanX*scale) / 2
	offY := (Height - spanY*scale) / 2

	lines := make([][]vec, 0, len(merc))
	for _, seg := range merc {
		line := make([]vec, len(seg))
		for i, v := range seg {
			line[i] = vec{offX + (v.x-minX)*scale, offY + (v.y-minY)*scale}
		}
		lines = append(lines, simplify(line, tolerance))
	}
	return lines
}

func mercator(p Point) vec {
	lat := p.Lat * math.Pi / 180
	return vec{
		x: (p.Lon + 180) / 360,
		y: (1 - math.Log(math.Tan(lat)+1/math.Cos(lat))/math.Pi) / 2,
	}
}
