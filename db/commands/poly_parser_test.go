package commands

import "testing"

func TestParsePoly(t *testing.T) {
	t.Run("single outer ring yields a Polygon with correct bbox", func(t *testing.T) {
		fixture := `TestRegion
1
	0.000000E+00	0.000000E+00
	1.000000E+00	0.000000E+00
	1.000000E+00	1.000000E+00
	0.000000E+00	1.000000E+00
END
END
`
		geometry, bbox, err := ParsePoly([]byte(fixture))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if geometry["type"] != "Polygon" {
			t.Fatalf("got type %v, want Polygon", geometry["type"])
		}
		coords, ok := geometry["coordinates"].([][][]float64)
		if !ok {
			t.Fatalf("unexpected coordinates type: %T", geometry["coordinates"])
		}
		if len(coords) != 1 {
			t.Fatalf("got %d rings, want 1", len(coords))
		}

		wantBbox := [4]float64{0, 0, 1, 1}
		if bbox != wantBbox {
			t.Fatalf("got bbox %v, want %v", bbox, wantBbox)
		}

		// Ring closure: first coordinate == last coordinate.
		ring := coords[0]
		first, last := ring[0], ring[len(ring)-1]
		if first[0] != last[0] || first[1] != last[1] {
			t.Fatalf("ring not closed: first=%v last=%v", first, last)
		}
	})

	t.Run("two outer rings yield a MultiPolygon, not a single Polygon with two rings", func(t *testing.T) {
		fixture := `TestMultiRegion
1
	0.0	0.0
	1.0	0.0
	1.0	1.0
	0.0	1.0
END
2
	10.0	10.0
	11.0	10.0
	11.0	11.0
	10.0	11.0
END
END
`
		geometry, bbox, err := ParsePoly([]byte(fixture))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if geometry["type"] != "MultiPolygon" {
			t.Fatalf("got type %v, want MultiPolygon", geometry["type"])
		}
		polygons, ok := geometry["coordinates"].([][][][]float64)
		if !ok {
			t.Fatalf("unexpected coordinates type: %T", geometry["coordinates"])
		}
		if len(polygons) != 2 {
			t.Fatalf("got %d polygons, want 2", len(polygons))
		}
		for i, poly := range polygons {
			if len(poly) != 1 {
				t.Fatalf("polygon %d: got %d rings, want 1 (no holes)", i, len(poly))
			}
		}

		wantBbox := [4]float64{0, 0, 11, 11}
		if bbox != wantBbox {
			t.Fatalf("got bbox %v, want %v", bbox, wantBbox)
		}
	})

	t.Run("outer ring plus a hole yields a Polygon with exactly 2 rings", func(t *testing.T) {
		fixture := `TestHoleRegion
1
	0.0	0.0
	4.0	0.0
	4.0	4.0
	0.0	4.0
END
!1
	1.0	1.0
	2.0	1.0
	2.0	2.0
	1.0	2.0
END
END
`
		geometry, _, err := ParsePoly([]byte(fixture))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if geometry["type"] != "Polygon" {
			t.Fatalf("got type %v, want Polygon", geometry["type"])
		}
		coords, ok := geometry["coordinates"].([][][]float64)
		if !ok {
			t.Fatalf("unexpected coordinates type: %T", geometry["coordinates"])
		}
		if len(coords) != 2 {
			t.Fatalf("got %d rings, want 2 (outer + hole)", len(coords))
		}

		// Outer ring should be CCW (positive area), hole should be CW (negative area).
		outerArea := shoelaceForTest(coords[0])
		if outerArea <= 0 {
			t.Fatalf("outer ring area = %v, want > 0 (CCW)", outerArea)
		}
		holeArea := shoelaceForTest(coords[1])
		if holeArea >= 0 {
			t.Fatalf("hole ring area = %v, want < 0 (CW)", holeArea)
		}
	})

	t.Run("a clockwise-wound outer ring is reversed to counter-clockwise", func(t *testing.T) {
		// Points wound clockwise: (0,0) -> (0,1) -> (1,1) -> (1,0).
		fixture := `TestClockwiseRegion
1
	0.0	0.0
	0.0	1.0
	1.0	1.0
	1.0	0.0
END
END
`
		geometry, _, err := ParsePoly([]byte(fixture))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		coords, ok := geometry["coordinates"].([][][]float64)
		if !ok {
			t.Fatalf("unexpected coordinates type: %T", geometry["coordinates"])
		}
		area := shoelaceForTest(coords[0])
		if area <= 0 {
			t.Fatalf("got area %v, want > 0 (reversed to CCW)", area)
		}
	})

	t.Run("coordinate line with wrong field count returns an error", func(t *testing.T) {
		fixture := `Bad
1
	0.0	0.0	0.0
END
END
`
		_, _, err := ParsePoly([]byte(fixture))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("unparseable float returns an error", func(t *testing.T) {
		fixture := `Bad
1
	notanumber	0.0
END
END
`
		_, _, err := ParsePoly([]byte(fixture))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("ring never closed with END returns an error", func(t *testing.T) {
		fixture := `Bad
1
	0.0	0.0
	1.0	1.0
`
		_, _, err := ParsePoly([]byte(fixture))
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

// shoelaceForTest mirrors the package's internal signedArea helper against
// the [][]float64 shape exposed in emitted GeoJSON coordinates, so tests
// can assert on winding without reaching into unexported internals.
func shoelaceForTest(ring [][]float64) float64 {
	area := 0.0
	n := len(ring)
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		area += ring[i][0]*ring[j][1] - ring[j][0]*ring[i][1]
	}
	return area / 2
}
