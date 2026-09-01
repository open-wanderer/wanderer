package main

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"
)

func TestSimplifyGeoJSONTrailsPreservesEndpointsWithinSphericalTolerance(t *testing.T) {
	trail := Trail{
		ID:       "bent",
		Scenario: "test",
		Parts: [][]Coordinate{{
			{Lat: 0, Lon: 0},
			{Lat: 0.001, Lon: 0.01},
			{Lat: 0, Lon: 0.02},
		}},
	}
	deviation := pointToSegmentDistanceMeters(trail.Parts[0][1], trail.Parts[0][0], trail.Parts[0][2])
	if deviation <= 100 || deviation >= 120 {
		t.Fatalf("fixture deviation = %.3f m, want between 100 m and 120 m", deviation)
	}

	kept, keptReport, err := simplifyGeoJSONTrailsContext(context.Background(), []Trail{trail}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(kept[0].Parts[0], trail.Parts[0]) {
		t.Fatalf("100 m simplification = %+v, want bend retained", kept[0].Parts[0])
	}
	if keptReport.IndexedVertices != 3 || keptReport.VertexReductionPercent != 0 {
		t.Fatalf("retained report = %+v", keptReport)
	}

	removed, removedReport, err := simplifyGeoJSONTrailsContext(context.Background(), []Trail{trail}, 120)
	if err != nil {
		t.Fatal(err)
	}
	want := []Coordinate{trail.Parts[0][0], trail.Parts[0][2]}
	if !reflect.DeepEqual(removed[0].Parts[0], want) {
		t.Fatalf("120 m simplification = %+v, want endpoints %+v", removed[0].Parts[0], want)
	}
	if removedReport.SourceVertices != 3 || removedReport.IndexedVertices != 2 ||
		removedReport.SourceSegments != 2 || removedReport.IndexedSegments != 1 {
		t.Fatalf("simplified report = %+v", removedReport)
	}
	if removedReport.VertexReductionPercent < 33.3 || removedReport.VertexReductionPercent > 33.4 {
		t.Fatalf("vertex reduction = %.6f%%, want about 33.33%%", removedReport.VertexReductionPercent)
	}
}

func TestSimplifyGeoJSONTrailsPreservesMultipartBoundaries(t *testing.T) {
	trail := Trail{ID: "multipart", Parts: [][]Coordinate{
		{{Lat: 0, Lon: 0}, {Lat: 0, Lon: 0.01}, {Lat: 0, Lon: 0.02}},
		{},
		{{Lat: 1, Lon: 1}, {Lat: 1, Lon: 1.01}, {Lat: 1, Lon: 1.02}},
		{{Lat: 2, Lon: 2}},
	}}
	original := cloneTrailParts(trail.Parts)

	got, report, err := simplifyGeoJSONTrailsContext(context.Background(), []Trail{trail}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(got[0].Parts) != len(trail.Parts) {
		t.Fatalf("part count = %d, want %d", len(got[0].Parts), len(trail.Parts))
	}
	want := [][]Coordinate{
		{{Lat: 0, Lon: 0}, {Lat: 0, Lon: 0.02}},
		{},
		{{Lat: 1, Lon: 1}, {Lat: 1, Lon: 1.02}},
		{{Lat: 2, Lon: 2}},
	}
	if !reflect.DeepEqual(got[0].Parts, want) {
		t.Fatalf("simplified multipart = %+v, want %+v", got[0].Parts, want)
	}
	if !reflect.DeepEqual(trail.Parts, original) {
		t.Fatalf("input was mutated: got %+v, want %+v", trail.Parts, original)
	}
	if report.SourceVertices != 7 || report.IndexedVertices != 5 ||
		report.SourceSegments != 4 || report.IndexedSegments != 2 {
		t.Fatalf("multipart report = %+v", report)
	}
}

func TestSimplifyGeoJSONTrailsZeroToleranceReturnsDeepCopy(t *testing.T) {
	trails := []Trail{{
		ID:       "unchanged",
		Scenario: "zero",
		Parts: [][]Coordinate{
			{{Lat: 1, Lon: 2}, {Lat: 3, Lon: 4}, {Lat: 5, Lon: 6}},
			{{Lat: 7, Lon: 8}},
		},
	}}
	want := []Trail{{ID: trails[0].ID, Scenario: trails[0].Scenario, Parts: cloneTrailParts(trails[0].Parts)}}

	got, report, err := simplifyGeoJSONTrailsContext(context.Background(), trails, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("zero-tolerance result = %+v, want %+v", got, want)
	}
	if report.SimplifyToleranceMeters != 0 || report.SourceVertices != 4 ||
		report.IndexedVertices != 4 || report.SourceSegments != 2 ||
		report.IndexedSegments != 2 || report.VertexReductionPercent != 0 {
		t.Fatalf("zero-tolerance report = %+v", report)
	}

	got[0].Parts[0][0].Lat = 99
	if trails[0].Parts[0][0].Lat == 99 {
		t.Fatal("zero-tolerance result aliases input coordinates")
	}
}

func TestSimplifyGeoJSONTrailsHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got, _, err := simplifyGeoJSONTrailsContext(ctx, []Trail{{
		ID:    "cancelled",
		Parts: [][]Coordinate{{{Lat: 0, Lon: 0}, {Lat: 0, Lon: 1}, {Lat: 0, Lon: 2}}},
	}}, 10)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation error = %v, want context.Canceled", err)
	}
	if got != nil {
		t.Fatalf("cancelled result = %+v, want nil", got)
	}
}

func TestNormalizeGeoJSONTrailsDensifiesLongHighLatitudeArc(t *testing.T) {
	const maxSegmentMeters = 250_000.0
	trail := Trail{ID: "high-latitude", Parts: [][]Coordinate{{
		{Lat: 75, Lon: -70},
		{Lat: 75, Lon: 70},
	}}}

	got, report, err := normalizeGeoJSONTrailsContext(context.Background(), []Trail{trail}, 50, maxSegmentMeters)
	if err != nil {
		t.Fatal(err)
	}
	if report.PreNormalizationVertices != 2 || report.PreNormalizationSegments != 1 ||
		report.DensifiedVertices == 0 || report.AntimeridianCuts != 0 ||
		report.PostNormalizationVertices != report.IndexedVertices ||
		report.PostNormalizationSegments != report.IndexedSegments {
		t.Fatalf("normalization report = %+v", report)
	}
	if len(got[0].Parts) != 1 || len(got[0].Parts[0]) <= 2 {
		t.Fatalf("densified parts = %+v", got[0].Parts)
	}
	maximumLatitude := -90.0
	for _, coordinate := range got[0].Parts[0] {
		maximumLatitude = math.Max(maximumLatitude, coordinate.Lat)
	}
	if maximumLatitude <= 75 {
		t.Fatalf("maximum latitude = %.6f, want a great-circle arc north of 75 degrees", maximumLatitude)
	}
	assertGeoJSONPartsNormalized(t, got[0].Parts, maxSegmentMeters)
}

func TestNormalizeGeoJSONTrailsCutsHighLatitudeAntimeridianArc(t *testing.T) {
	const maxSegmentMeters = 500_000.0
	trail := Trail{ID: "dateline", Parts: [][]Coordinate{{
		{Lat: 70, Lon: 170},
		{Lat: 70, Lon: -170},
	}}}

	got, report, err := normalizeGeoJSONTrailsContext(context.Background(), []Trail{trail}, 0, maxSegmentMeters)
	if err != nil {
		t.Fatal(err)
	}
	if report.AntimeridianCuts != 1 || len(got[0].Parts) != 2 {
		t.Fatalf("dateline normalization = parts %+v, report %+v", got[0].Parts, report)
	}
	left := got[0].Parts[0][len(got[0].Parts[0])-1]
	right := got[0].Parts[1][0]
	if left.Lon != 180 || right.Lon != -180 || math.Abs(left.Lat-right.Lat) > 1e-9 || left.Lat <= 70 {
		t.Fatalf("seam endpoints = %+v / %+v, want matching high-latitude +180/-180 points", left, right)
	}
	assertGeoJSONPartsNormalized(t, got[0].Parts, maxSegmentMeters)
}

func TestNormalizeGeoJSONTrailsHandlesExplicitSeamVertex(t *testing.T) {
	trail := Trail{ID: "seam-vertex", Parts: [][]Coordinate{{
		{Lat: 0, Lon: 179},
		{Lat: 0, Lon: 180},
		{Lat: 0, Lon: -179},
	}}}

	got, report, err := normalizeGeoJSONTrailsContext(context.Background(), []Trail{trail}, 0, 1_000_000)
	if err != nil {
		t.Fatal(err)
	}
	want := [][]Coordinate{
		{{Lat: 0, Lon: 179}, {Lat: 0, Lon: 180}},
		{{Lat: 0, Lon: -180}, {Lat: 0, Lon: -179}},
	}
	if !reflect.DeepEqual(got[0].Parts, want) || report.AntimeridianCuts != 1 {
		t.Fatalf("explicit seam normalization = %+v, report %+v; want %+v", got[0].Parts, report, want)
	}
	assertGeoJSONPartsNormalized(t, got[0].Parts, 1_000_000)
}

func TestNormalizeGeoJSONTrailsPreservesOriginalMultipartGaps(t *testing.T) {
	trail := Trail{ID: "multipart-dateline", Parts: [][]Coordinate{
		{{Lat: 0, Lon: 179}, {Lat: 0, Lon: -179}},
		{{Lat: 10, Lon: 20}, {Lat: 10, Lon: 21}},
	}}

	got, report, err := normalizeGeoJSONTrailsContext(context.Background(), []Trail{trail}, 0, 1_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if len(got[0].Parts) != 3 || report.AntimeridianCuts != 1 {
		t.Fatalf("multipart normalization = %+v, report %+v", got[0].Parts, report)
	}
	if got[0].Parts[1][len(got[0].Parts[1])-1].Lon != -179 || got[0].Parts[2][0] != (Coordinate{Lat: 10, Lon: 20}) {
		t.Fatalf("original part gap was not retained: %+v", got[0].Parts)
	}
	assertGeoJSONPartsNormalized(t, got[0].Parts, 1_000_000)
}

func TestNormalizeGeoJSONTrailsRejectsInvalidMaximumSegmentLength(t *testing.T) {
	trail := []Trail{{ID: "invalid", Parts: [][]Coordinate{{{Lat: 0, Lon: 0}, {Lat: 0, Lon: 1}}}}}
	for _, maximum := range []float64{-1, math.NaN(), math.Inf(1)} {
		if _, _, err := normalizeGeoJSONTrailsContext(context.Background(), trail, 0, maximum); err == nil {
			t.Fatalf("maximum segment length %v unexpectedly accepted", maximum)
		}
	}
}

func TestNormalizeGeneratedDatasetEmitsOnlyLocalLineSegments(t *testing.T) {
	dataset, err := generateDataset(100, 20, 42)
	if err != nil {
		t.Fatal(err)
	}
	indexed, report, err := normalizeGeoJSONTrailsContext(context.Background(), dataset.Trails, 50, defaultGeoJSONMaxSegmentLengthMeters)
	if err != nil {
		t.Fatal(err)
	}
	if report.DensifiedVertices == 0 || report.AntimeridianCuts == 0 {
		t.Fatalf("generated normalization did not exercise long/dateline routes: %+v", report)
	}
	for _, trail := range indexed {
		assertGeoJSONPartsNormalized(t, trail.Parts, defaultGeoJSONMaxSegmentLengthMeters)
	}
}

func assertGeoJSONPartsNormalized(t *testing.T, parts [][]Coordinate, maxSegmentMeters float64) {
	t.Helper()
	for partIndex, part := range parts {
		if len(part) == 1 {
			t.Fatalf("part %d is a singleton: %+v", partIndex, part)
		}
		for index := 1; index < len(part); index++ {
			longitudeDelta := math.Abs(part[index].Lon - part[index-1].Lon)
			if longitudeDelta > 180+1e-9 {
				t.Fatalf("part %d segment %d crosses the antimeridian: %+v -> %+v", partIndex, index-1, part[index-1], part[index])
			}
			distance := angularDistance(part[index-1], part[index]) * earthRadiusMeters
			if distance > maxSegmentMeters+1e-6*maxSegmentMeters {
				t.Fatalf("part %d segment %d length = %.3f m, maximum %.3f m", partIndex, index-1, distance, maxSegmentMeters)
			}
		}
	}
}

func cloneTrailParts(parts [][]Coordinate) [][]Coordinate {
	cloned := make([][]Coordinate, len(parts))
	for index, part := range parts {
		cloned[index] = cloneGeoJSONPart(part)
	}
	return cloned
}
