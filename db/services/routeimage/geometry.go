package routeimage

import "math"

type vec struct{ x, y float64 }

func (a vec) dist(b vec) float64 { return math.Hypot(a.x-b.x, a.y-b.y) }

// simplify drops the points that stray less than tolerance from the line
// between their neighbours (Ramer–Douglas–Peucker).
func simplify(line []vec, tolerance float64) []vec {
	if len(line) < 3 {
		return line
	}
	keep := make([]bool, len(line))
	keep[0], keep[len(line)-1] = true, true

	type span struct{ from, to int }
	stack := []span{{0, len(line) - 1}}
	for len(stack) > 0 {
		s := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		worst, index := 0.0, -1
		for i := s.from + 1; i < s.to; i++ {
			if d := segmentDistance(line[i], line[s.from], line[s.to]); d > worst {
				worst, index = d, i
			}
		}
		if index >= 0 && worst > tolerance {
			keep[index] = true
			stack = append(stack, span{s.from, index}, span{index, s.to})
		}
	}

	out := make([]vec, 0, len(line))
	for i, p := range line {
		if keep[i] {
			out = append(out, p)
		}
	}
	return out
}

// segmentDistance is the distance from p to the segment a–b.
func segmentDistance(p, a, b vec) float64 {
	dx, dy := b.x-a.x, b.y-a.y
	l2 := dx*dx + dy*dy
	if l2 == 0 {
		return p.dist(a)
	}
	t := math.Max(0, math.Min(1, ((p.x-a.x)*dx+(p.y-a.y)*dy)/l2))
	return p.dist(vec{a.x + t*dx, a.y + t*dy})
}
