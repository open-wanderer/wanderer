package routeimage

import (
	"image"
	"image/color"
	"math"
)

// A still frame of the contour canvas behind the docs hero
// (docs/src/pages/index.astro, "Hero topo contour background").
const (
	contourLevels    = 24
	contourNoise     = 0.0028
	contourAmplitude = 0.095
	contourStep      = 2.5
)

func drawContours(img *image.RGBA, t float64) {
	w, h := float64(Width), float64(Height)
	steps := int(math.Ceil(w / contourStep))

	for i := 0; i < contourLevels; i++ {
		progress := float64(i) / (contourLevels - 1)
		baseY := h * (0.05 + 0.92*progress)
		opacity := 0.018 + progress*0.062
		lineW := 0.3 + progress*0.8

		line := make([]vec, steps+1)
		for j := range line {
			x := float64(j) / float64(steps) * w
			nx := x*contourNoise + t*0.07
			ny := baseY*contourNoise*0.55 + float64(i)*0.18
			line[j] = vec{x, baseY + fbm(nx, ny)*h*contourAmplitude}
		}

		var s shape
		s.strokeOutline(line, lineW)
		fill(img, s, color.NRGBA{0xff, 0xff, 0xff, uint8(math.Round(opacity * 255))})
	}
}

func hash(ix, iy float64) float64 {
	n := math.Sin(ix*127.1+iy*311.7) * 43758.5453
	return n - math.Floor(n)
}

func smoothstep(x float64) float64 { return x * x * (3 - 2*x) }

func valueNoise(x, y float64) float64 {
	xi, yi := math.Floor(x), math.Floor(y)
	u, v := smoothstep(x-xi), smoothstep(y-yi)
	a, b := hash(xi, yi), hash(xi+1, yi)
	c, d := hash(xi, yi+1), hash(xi+1, yi+1)
	return a + (b-a)*u + (c-a)*v + (a-b-c+d)*u*v
}

// fbm is three octaves of value noise, normalised to [-1, 1].
func fbm(x, y float64) float64 {
	v := valueNoise(x, y)*0.5 +
		valueNoise(x*2.1+1.7, y*2.1+9.2)*0.25 +
		valueNoise(x*4.3+4.3, y*4.3+2.8)*0.125
	return v/0.875*2 - 1
}
