package main

import (
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestH3FieldsForTrailAreDeterministicSortedAndDeduplicated(t *testing.T) {
	t.Parallel()

	trail := Trail{
		ID: "zurich-loop",
		Parts: [][]Coordinate{{
			{Lat: 47.3769, Lon: 8.5417},
			{Lat: 47.3810, Lon: 8.5520},
			{Lat: 47.3769, Lon: 8.5417},
		}},
	}

	first, err := h3FieldsForTrail(trail, []int{9, 7, 9})
	if err != nil {
		t.Fatalf("first rasterization: %v", err)
	}
	second, err := h3FieldsForTrail(trail, []int{7, 9})
	if err != nil {
		t.Fatalf("second rasterization: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("rasterization is not deterministic:\nfirst:  %#v\nsecond: %#v", first, second)
	}
	if len(first) != 2 {
		t.Fatalf("got %d fields, want 2", len(first))
	}

	for field, values := range first {
		if len(values) == 0 {
			t.Fatalf("field %s contains no cells", field)
		}
		if !sort.StringsAreSorted(values) {
			t.Fatalf("field %s is not sorted: %v", field, values)
		}
		for index := 1; index < len(values); index++ {
			if values[index] == values[index-1] {
				t.Fatalf("field %s contains duplicate cell %s", field, values[index])
			}
		}
	}
}

func TestH3FieldsForTrailDoesNotBridgeMultiLineParts(t *testing.T) {
	t.Parallel()

	firstPart := []Coordinate{
		{Lat: 47.3769, Lon: 8.5417},
		{Lat: 47.3810, Lon: 8.5520},
	}
	secondPart := []Coordinate{
		{Lat: 46.2044, Lon: 6.1432},
		{Lat: 46.2080, Lon: 6.1510},
	}

	multi, err := h3FieldsForTrail(Trail{ID: "multi", Parts: [][]Coordinate{firstPart, secondPart}}, []int{8})
	if err != nil {
		t.Fatalf("rasterize MultiLine: %v", err)
	}
	first, err := h3FieldsForTrail(Trail{ID: "first", Parts: [][]Coordinate{firstPart}}, []int{8})
	if err != nil {
		t.Fatalf("rasterize first LineString: %v", err)
	}
	second, err := h3FieldsForTrail(Trail{ID: "second", Parts: [][]Coordinate{secondPart}}, []int{8})
	if err != nil {
		t.Fatalf("rasterize second LineString: %v", err)
	}

	wantSet := make(map[string]struct{}, len(first["h3_r8"])+len(second["h3_r8"]))
	for _, cell := range first["h3_r8"] {
		wantSet[cell] = struct{}{}
	}
	for _, cell := range second["h3_r8"] {
		wantSet[cell] = struct{}{}
	}
	want := make([]string, 0, len(wantSet))
	for cell := range wantSet {
		want = append(want, cell)
	}
	sort.Strings(want)

	if !reflect.DeepEqual(multi["h3_r8"], want) {
		t.Fatalf("MultiLine rasterization contains cells outside its independent parts\ngot:  %v\nwant: %v", multi["h3_r8"], want)
	}
}

func TestH3QueryFilterSelectsResolutionByRadius(t *testing.T) {
	t.Parallel()

	point := Coordinate{Lat: 47.3769, Lon: 8.5417}
	resolutions := []int{9, 5, 7, 6, 9}
	tests := []struct {
		name   string
		radius float64
		field  string
	}{
		{name: "zero", radius: 0, field: "h3_r9"},
		{name: "500m", radius: 500, field: "h3_r9"},
		{name: "above 500m", radius: 501, field: "h3_r7"},
		{name: "5km", radius: 5_000, field: "h3_r7"},
		{name: "above 5km", radius: 5_001, field: "h3_r6"},
		{name: "25km", radius: 25_000, field: "h3_r6"},
		{name: "above 25km", radius: 25_001, field: "h3_r5"},
		{name: "100km", radius: 100_000, field: "h3_r5"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			field, filter, cells, err := h3QueryFilter(point, test.radius, resolutions)
			if err != nil {
				t.Fatalf("h3QueryFilter: %v", err)
			}
			if field != test.field {
				t.Fatalf("field = %q, want %q", field, test.field)
			}
			if !strings.HasPrefix(filter, field+" IN [\"") || !strings.HasSuffix(filter, "\"]") {
				t.Fatalf("filter is not a quoted Meilisearch IN expression: %q", filter)
			}
			if cells < 1 || cells != len(h3CellsFromFilter(t, filter)) {
				t.Fatalf("reported cell count %d does not match filter %q", cells, filter)
			}
		})
	}
}

func TestH3CandidatesCoverSimpleNearbyTrails(t *testing.T) {
	t.Parallel()

	anchor := Coordinate{Lat: 47.3769, Lon: 8.5417}
	trail := Trail{
		ID: "short-line",
		Parts: [][]Coordinate{{
			{Lat: anchor.Lat, Lon: anchor.Lon - 0.002},
			{Lat: anchor.Lat, Lon: anchor.Lon + 0.002},
		}},
	}
	resolutions := []int{5, 6, 7, 9}
	fields, err := h3FieldsForTrail(trail, resolutions)
	if err != nil {
		t.Fatalf("rasterize trail: %v", err)
	}

	for _, radius := range []float64{500, 5_000, 25_000, 100_000} {
		radius := radius
		t.Run(strconv.Itoa(int(radius)), func(t *testing.T) {
			// About 60% of the tested radius north of the line. The point is
			// therefore an unambiguous oracle match without lying on a boundary.
			queryPoint := Coordinate{
				Lat: anchor.Lat + (radius*0.6)/111_320,
				Lon: anchor.Lon,
			}
			field, filter, _, err := h3QueryFilter(queryPoint, radius, resolutions)
			if err != nil {
				t.Fatalf("query filter: %v", err)
			}

			queryCells := h3CellsFromFilter(t, filter)
			covered := false
			for _, trailCell := range fields[field] {
				if _, ok := queryCells[trailCell]; ok {
					covered = true
					break
				}
			}
			if !covered {
				t.Fatalf("radius %.0f at %s has no candidate-cell overlap", radius, field)
			}
		})
	}
}

func TestH3InputValidation(t *testing.T) {
	t.Parallel()

	if _, err := h3FieldsForTrail(Trail{ID: "invalid", Parts: [][]Coordinate{{{Lat: 91, Lon: 0}}}}, []int{9}); err == nil {
		t.Fatal("expected invalid coordinate error")
	}
	if _, err := h3FieldsForTrail(Trail{ID: "invalid"}, nil); err == nil {
		t.Fatal("expected empty resolution error")
	}
	if _, _, _, err := h3QueryFilter(Coordinate{}, -1, []int{9}); err == nil {
		t.Fatal("expected negative radius error")
	}
	if _, _, _, err := h3QueryFilter(Coordinate{}, 500, []int{16}); err == nil {
		t.Fatal("expected invalid resolution error")
	}
	if _, _, _, err := h3QueryFilter(Coordinate{}, 100_000, []int{15}); err == nil || !strings.Contains(err.Error(), "maximum") {
		t.Fatalf("expected query cell budget error, got %v", err)
	}
}

var h3FilterCellPattern = regexp.MustCompile(`"([0-9a-f]+)"`)

func h3CellsFromFilter(t *testing.T, filter string) map[string]struct{} {
	t.Helper()
	matches := h3FilterCellPattern.FindAllStringSubmatch(filter, -1)
	result := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		result[match[1]] = struct{}{}
	}
	return result
}
