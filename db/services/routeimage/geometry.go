package routeimage

import "math"

type vec struct{ x, y float64 }

func (a vec) add(b vec) vec      { return vec{a.x + b.x, a.y + b.y} }
func (a vec) sub(b vec) vec      { return vec{a.x - b.x, a.y - b.y} }
func (a vec) dist(b vec) float64 { return math.Hypot(a.x-b.x, a.y-b.y) }
func (a vec) normal(half float64) vec {
	l := math.Hypot(a.x, a.y)
	return vec{-a.y / l * half, a.x / l * half}
}

// shape is a set of closed polygons that are filled together.
type shape [][]vec

// add appends a polygon, always with the same winding. The rasterizer sums
// signed coverage, so a polygon wound the other way would cut a hole into any
// polygon it overlaps.
func (s *shape) add(pts ...vec) {
	area := 0.0
	for i, p := range pts {
		q := pts[(i+1)%len(pts)]
		area += p.x*q.y - q.x*p.y
	}
	if area < 0 {
		for i, j := 0, len(pts)-1; i < j; i, j = i+1, j-1 {
			pts[i], pts[j] = pts[j], pts[i]
		}
	}
	*s = append(*s, pts)
}

func (s *shape) circle(c vec, radius float64) {
	n := max(12, int(math.Ceil(2*math.Pi*radius/1.5)))
	pts := make([]vec, n)
	for i := range pts {
		a := 2 * math.Pi * float64(i) / float64(n)
		pts[i] = vec{c.x + radius*math.Cos(a), c.y + radius*math.Sin(a)}
	}
	s.add(pts...)
}

// strokeRound strokes a polyline with round joins and caps. It is meant for
// lines at least a few pixels wide: joins overlap, and where a pixel is only
// partly covered the overlap shows as a slightly heavier edge.
func (s *shape) strokeRound(line []vec, width float64) {
	half := width / 2
	for i := 1; i < len(line); i++ {
		a, b := line[i-1], line[i]
		n := b.sub(a).normal(half)
		s.add(a.add(n), b.add(n), b.sub(n), a.sub(n))
	}
	for _, p := range line {
		s.circle(p, half)
	}
}

// strokeOutline strokes a gently curving polyline as one outline, which keeps
// hairlines even: there are no overlapping joins to brighten partly covered
// pixels. Sharp turns would fold the outline over itself.
func (s *shape) strokeOutline(line []vec, width float64) {
	if len(line) < 2 {
		return
	}
	half := width / 2
	left := make([]vec, len(line))
	right := make([]vec, len(line))
	for i, p := range line {
		prev, next := line[max(i-1, 0)], line[min(i+1, len(line)-1)]
		n := next.sub(prev).normal(half)
		left[i], right[i] = p.add(n), p.sub(n)
	}
	pts := make([]vec, 0, 2*len(line))
	pts = append(pts, left...)
	for i := len(right) - 1; i >= 0; i-- {
		pts = append(pts, right[i])
	}
	s.add(pts...)
}
