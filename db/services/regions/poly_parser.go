package regions

import (
	"bufio"
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ParsePoly parses an Osmosis-format .poly file (as used by CoMaps'
// data/borders/*.poly boundary files, see
// https://wiki.openstreetmap.org/wiki/Osmosis/Polygon_Filter_File_Format)
// into a GeoJSON Polygon or MultiPolygon geometry, plus a derived bounding
// box.
//
// Relocated here from package commands: the runtime on-demand geometry
// fetch inside package regions now consumes this verbatim, not the
// maintainer seed-regions CLI — a services/regions -> commands import is a
// dependency direction this package deliberately refuses.
// Format:
//   - line 1: a free-text name (ignored — not necessarily the region id)
//   - each ring: a header line ("1", "2", ... for an outer ring; a
//     "!"-prefixed header for a hole), then tab/whitespace-separated
//     "lon<TAB>lat" coordinate lines in scientific notation, terminated by
//     a lone "END"
//   - the whole file is terminated by a final "END"
//
// A single outer ring produces a GeoJSON Polygon; more than one outer ring
// (e.g. an antimeridian-split country) produces a MultiPolygon.
// Every outer ring is re-wound counter-clockwise and every hole clockwise
// per RFC 7946, and every ring is closed (first coordinate ==
// last) even if the source omitted the closing repeat. bbox is
// [minLon, minLat, maxLon, maxLat] computed over all outer-ring vertices,
// matching the element order used by db/services/regions/config.go.
//
// Malformed input (a coordinate line with a field count other than 2, an
// unparseable float, or a ring that reaches EOF before its terminating
// "END") returns a descriptive error tagged with the offending line
// content; ParsePoly always returns cleanly, it does not abort the process
// itself.
func ParsePoly(data []byte) (map[string]any, [4]float64, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)

	lineNo := 0
	nextLine := func() (string, bool) {
		if !scanner.Scan() {
			return "", false
		}
		lineNo++
		return scanner.Text(), true
	}

	// Line 1: free-text name, ignored.
	if _, ok := nextLine(); !ok {
		return nil, [4]float64{}, fmt.Errorf("poly_parser: empty input, expected a name line")
	}

	var outerRings [][][2]float64
	var holeRings [][][2]float64

	for {
		headerLine, ok := nextLine()
		if !ok {
			return nil, [4]float64{}, fmt.Errorf("poly_parser: unexpected EOF at line %d, expected a ring header or the file-terminating END", lineNo)
		}
		header := strings.TrimSpace(headerLine)
		if header == "END" {
			break // file terminator
		}
		isHole := strings.HasPrefix(header, "!")

		var ring [][2]float64
		for {
			coordLine, ok := nextLine()
			if !ok {
				return nil, [4]float64{}, fmt.Errorf("poly_parser: unexpected EOF at line %d, ring starting at header %q was never closed with END", lineNo, header)
			}
			if strings.TrimSpace(coordLine) == "END" {
				break // ring terminator
			}
			fields := strings.Fields(coordLine)
			if len(fields) != 2 {
				return nil, [4]float64{}, fmt.Errorf("poly_parser: line %d: expected 2 fields (lon, lat), got %d: %q", lineNo, len(fields), coordLine)
			}
			lon, err := strconv.ParseFloat(fields[0], 64)
			if err != nil {
				return nil, [4]float64{}, fmt.Errorf("poly_parser: line %d: invalid longitude %q: %w", lineNo, fields[0], err)
			}
			lat, err := strconv.ParseFloat(fields[1], 64)
			if err != nil {
				return nil, [4]float64{}, fmt.Errorf("poly_parser: line %d: invalid latitude %q: %w", lineNo, fields[1], err)
			}
			ring = append(ring, [2]float64{lon, lat})
		}

		if len(ring) == 0 {
			return nil, [4]float64{}, fmt.Errorf("poly_parser: ring at line %d (header %q) has no coordinates", lineNo, header)
		}

		if isHole {
			holeRings = append(holeRings, ring)
		} else {
			outerRings = append(outerRings, ring)
		}
	}

	if len(outerRings) == 0 {
		return nil, [4]float64{}, fmt.Errorf("poly_parser: no outer rings found")
	}

	// Correct winding (outer CCW, hole CW) and close every ring.
	for i, r := range outerRings {
		outerRings[i] = closeRing(windRing(r, true))
	}
	for i, r := range holeRings {
		holeRings[i] = closeRing(windRing(r, false))
	}

	bbox := boundingBox(outerRings)

	var geometry map[string]any
	if len(outerRings) == 1 {
		coords := make([][][]float64, 0, 1+len(holeRings))
		coords = append(coords, toRingSlice(outerRings[0]))
		for _, h := range holeRings {
			coords = append(coords, toRingSlice(h))
		}
		geometry = map[string]any{
			"type":        "Polygon",
			"coordinates": coords,
		}
	} else {
		polygons := make([][][][]float64, len(outerRings))
		for i, outer := range outerRings {
			polygons[i] = [][][]float64{toRingSlice(outer)}
		}
		for _, hole := range holeRings {
			idx := 0
			assigned := false
			if len(hole) > 0 {
				for i, outer := range outerRings {
					if pointInRing(hole[0], outer) {
						idx = i
						assigned = true
						break
					}
				}
			}
			if !assigned {
				idx = 0
			}
			polygons[idx] = append(polygons[idx], toRingSlice(hole))
		}
		geometry = map[string]any{
			"type":        "MultiPolygon",
			"coordinates": polygons,
		}
	}

	return geometry, bbox, nil
}

// signedArea computes a ring's signed area via the shoelace formula,
// wrapping around from the last point back to the first regardless of
// whether the ring is already explicitly closed. Positive = counter-
// clockwise (when x=lon, y=lat); negative = clockwise.
func signedArea(ring [][2]float64) float64 {
	area := 0.0
	n := len(ring)
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		area += ring[i][0]*ring[j][1] - ring[j][0]*ring[i][1]
	}
	return area / 2
}

// windRing reverses ring's point order if needed so its signed area has
// the sign required by ccw (true = require counter-clockwise/positive,
// false = require clockwise/negative).
func windRing(ring [][2]float64, ccw bool) [][2]float64 {
	area := signedArea(ring)
	if (ccw && area < 0) || (!ccw && area > 0) {
		reversed := make([][2]float64, len(ring))
		for i, p := range ring {
			reversed[len(ring)-1-i] = p
		}
		return reversed
	}
	return ring
}

// closeRing appends a copy of the first point to the end of ring if it
// isn't already closed (first coordinate == last coordinate).
func closeRing(ring [][2]float64) [][2]float64 {
	if len(ring) == 0 {
		return ring
	}
	if ring[0] != ring[len(ring)-1] {
		ring = append(ring, ring[0])
	}
	return ring
}

// boundingBox computes [minLon, minLat, maxLon, maxLat] over every vertex
// of every ring passed in.
func boundingBox(rings [][][2]float64) [4]float64 {
	minLon, minLat := math.Inf(1), math.Inf(1)
	maxLon, maxLat := math.Inf(-1), math.Inf(-1)
	for _, ring := range rings {
		for _, p := range ring {
			if p[0] < minLon {
				minLon = p[0]
			}
			if p[0] > maxLon {
				maxLon = p[0]
			}
			if p[1] < minLat {
				minLat = p[1]
			}
			if p[1] > maxLat {
				maxLat = p[1]
			}
		}
	}
	return [4]float64{minLon, minLat, maxLon, maxLat}
}

// pointInRing reports whether p lies inside ring using a standard ray-cast
// point-in-polygon test. Used only to assign a hole to the outer ring that
// contains it in the MultiPolygon case.
func pointInRing(p [2]float64, ring [][2]float64) bool {
	inside := false
	n := len(ring)
	for i, j := 0, n-1; i < n; j, i = i, i+1 {
		xi, yi := ring[i][0], ring[i][1]
		xj, yj := ring[j][0], ring[j][1]
		if ((yi > p[1]) != (yj > p[1])) &&
			(p[0] < (xj-xi)*(p[1]-yi)/(yj-yi)+xi) {
			inside = !inside
		}
	}
	return inside
}

// toRingSlice converts an internal [][2]float64 ring into the
// [][]float64 shape used in the emitted GeoJSON coordinates.
func toRingSlice(ring [][2]float64) [][]float64 {
	out := make([][]float64, len(ring))
	for i, p := range ring {
		out[i] = []float64{p[0], p[1]}
	}
	return out
}
