// Package routeimage renders the placeholder image for trails without photos:
// the route on the Polar Night background with the contour lines of the docs
// hero. It needs no tiles, so it works offline and for every import path.
package routeimage

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"golang.org/x/image/vector"
)

const (
	Width  = 1200
	Height = 800

	// padding is the share of each side kept free around the route.
	padding = 0.14
)

type Point struct {
	Lat float64
	Lon float64
}

type gradientStop struct {
	at    float64
	color color.NRGBA
}

var (
	// The hero gradient of docs/src/pages/index.astro.
	background = []gradientStop{
		{0, color.NRGBA{0x0b, 0x0d, 0x16, 0xff}},
		{0.30, color.NRGBA{0x12, 0x14, 0x1c, 0xff}},
		{0.62, color.NRGBA{0x1b, 0x1e, 0x2c, 0xff}},
		{1, color.NRGBA{0x24, 0x27, 0x34, 0xff}},
	}

	routeColor  = color.NRGBA{0xff, 0xff, 0xff, 0xff}
	casingColor = color.NRGBA{0x0b, 0x0d, 0x16, 0x99}
	startColor  = color.NRGBA{0x4a, 0xde, 0x80, 0xff}
	endColor    = color.NRGBA{0xff, 0xff, 0xff, 0xff}
	markerRing  = color.NRGBA{0x0b, 0x0d, 0x16, 0xff}
)

const (
	routeWidth  = 4.0
	casingWidth = 10.0
	markerSize  = 8.0
	ringWidth   = 3.0
)

// Render draws the trail. Each segment is drawn as its own line, so gaps
// between GPX segments stay gaps. seed picks the contour pattern; the same
// seed always gives the same background.
func Render(segments [][]Point, seed float64) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, Width, Height))
	drawGradient(img)
	drawContours(img, seed)

	lines := project(segments)
	if len(lines) == 0 {
		return img
	}

	var casing, route shape
	for _, l := range lines {
		casing.strokeRound(l, casingWidth)
		route.strokeRound(l, routeWidth)
	}
	fill(img, casing, casingColor)
	fill(img, route, routeColor)

	first := lines[0][0]
	lastLine := lines[len(lines)-1]
	last := lastLine[len(lastLine)-1]
	if first.dist(last) > 3*markerSize {
		drawMarker(img, last, endColor)
	}
	drawMarker(img, first, startColor)

	return img
}

func drawMarker(img *image.RGBA, p vec, c color.NRGBA) {
	var ring, dot shape
	ring.circle(p, markerSize+ringWidth)
	dot.circle(p, markerSize)
	fill(img, ring, markerRing)
	fill(img, dot, c)
}

func drawGradient(img *image.RGBA) {
	for y := 0; y < Height; y++ {
		c := gradientAt(float64(y) / float64(Height-1))
		row := img.Pix[y*img.Stride : y*img.Stride+Width*4]
		for x := 0; x < len(row); x += 4 {
			row[x], row[x+1], row[x+2], row[x+3] = c.R, c.G, c.B, c.A
		}
	}
}

func gradientAt(t float64) color.NRGBA {
	for i := 1; i < len(background); i++ {
		a, b := background[i-1], background[i]
		if t <= b.at {
			f := (t - a.at) / (b.at - a.at)
			mix := func(x, y uint8) uint8 { return uint8(math.Round(float64(x) + (float64(y)-float64(x))*f)) }
			return color.NRGBA{mix(a.color.R, b.color.R), mix(a.color.G, b.color.G), mix(a.color.B, b.color.B), 0xff}
		}
	}
	return background[len(background)-1].color
}

// fill rasterizes s within its bounding box and composites c through it.
// Overlapping polygons saturate instead of adding up, so a translucent color
// is applied once even where strokes and joins overlap.
func fill(img *image.RGBA, s shape, c color.NRGBA) {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, poly := range s {
		for _, p := range poly {
			minX, maxX = math.Min(minX, p.x), math.Max(maxX, p.x)
			minY, maxY = math.Min(minY, p.y), math.Max(maxY, p.y)
		}
	}
	box := image.Rect(int(math.Floor(minX)), int(math.Floor(minY)), int(math.Ceil(maxX)), int(math.Ceil(maxY))).Intersect(img.Bounds())
	if box.Empty() {
		return
	}

	r := vector.NewRasterizer(box.Dx(), box.Dy())
	r.DrawOp = draw.Over
	ox, oy := float64(box.Min.X), float64(box.Min.Y)
	for _, poly := range s {
		r.MoveTo(float32(poly[0].x-ox), float32(poly[0].y-oy))
		for _, p := range poly[1:] {
			r.LineTo(float32(p.x-ox), float32(p.y-oy))
		}
		r.ClosePath()
	}
	r.Draw(img, box, image.NewUniform(c), image.Point{})
}

// project maps the segments into pixel space with Web Mercator, fits them
// into the padded frame and drops points closer than a pixel to their
// predecessor.
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
	availX, availY := Width*(1-2*padding), Height*(1-2*padding)
	scale := math.Min(availX/spanX, availY/spanY)
	if math.IsInf(scale, 0) || math.IsNaN(scale) {
		if spanX > 0 {
			scale = availX / spanX
		} else if spanY > 0 {
			scale = availY / spanY
		} else {
			scale = 0
		}
	}
	offX := (Width - spanX*scale) / 2
	offY := (Height - spanY*scale) / 2

	lines := make([][]vec, 0, len(merc))
	for _, seg := range merc {
		line := make([]vec, 0, len(seg))
		for _, v := range seg {
			p := vec{offX + (v.x-minX)*scale, offY + (v.y-minY)*scale}
			if len(line) > 0 && line[len(line)-1].dist(p) < 1 {
				continue
			}
			line = append(line, p)
		}
		lines = append(lines, line)
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
