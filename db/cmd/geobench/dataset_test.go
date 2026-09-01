package main

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"
)

func TestGenerateDatasetHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := generateDatasetContext(ctx, 100, 40, 42); !errors.Is(err, context.Canceled) {
		t.Fatalf("generate cancellation error = %v", err)
	}
}

func TestPointToTrailDistanceOnSegment(t *testing.T) {
	trail := Trail{Parts: [][]Coordinate{{
		{Lat: 0, Lon: -1},
		{Lat: 0, Lon: 1},
	}}}

	if got := pointToTrailDistanceMeters(Coordinate{Lat: 0, Lon: 0}, trail); got > 1e-6 {
		t.Fatalf("point on segment has distance %.9f m, want 0", got)
	}

	want := earthRadiusMeters * math.Pi / 180
	got := pointToTrailDistanceMeters(Coordinate{Lat: 1, Lon: 0}, trail)
	if math.Abs(got-want) > 1e-6 {
		t.Fatalf("distance above segment = %.9f m, want %.9f m", got, want)
	}
}

func TestPointToTrailDistanceDegenerate(t *testing.T) {
	origin := Coordinate{Lat: 46.8, Lon: 8.2}
	point := destinationPoint(origin, 1_000, 0)

	tests := []struct {
		name  string
		trail Trail
	}{
		{name: "one point", trail: Trail{Parts: [][]Coordinate{{origin}}}},
		{name: "repeated point", trail: Trail{Parts: [][]Coordinate{{origin, origin}}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := pointToTrailDistanceMeters(point, test.trail)
			if math.Abs(got-1_000) > 1e-6 {
				t.Fatalf("distance = %.9f m, want 1000 m", got)
			}
		})
	}
}

func TestPointToTrailDistanceDoesNotBridgeMultiLineGap(t *testing.T) {
	trail := Trail{Parts: [][]Coordinate{
		{{Lat: 0, Lon: 0}, {Lat: 0, Lon: 1}},
		{{Lat: 0, Lon: 3}, {Lat: 0, Lon: 4}},
	}}

	got := pointToTrailDistanceMeters(Coordinate{Lat: 0, Lon: 2}, trail)
	want := earthRadiusMeters * math.Pi / 180
	if math.Abs(got-want) > 1e-6 {
		t.Fatalf("distance across MultiLineString gap = %.9f m, want %.9f m", got, want)
	}
}

func TestPointToTrailDistanceAcrossAntimeridian(t *testing.T) {
	trail := Trail{Parts: [][]Coordinate{{
		{Lat: 0, Lon: 179},
		{Lat: 0, Lon: -179},
	}}}

	if got := pointToTrailDistanceMeters(Coordinate{Lat: 0, Lon: 180}, trail); got > 1e-6 {
		t.Fatalf("antimeridian point has distance %.9f m, want 0", got)
	}

	want := earthRadiusMeters * math.Pi / 180
	got := pointToTrailDistanceMeters(Coordinate{Lat: 1, Lon: 180}, trail)
	if math.Abs(got-want) > 1e-6 {
		t.Fatalf("distance above antimeridian segment = %.9f m, want %.9f m", got, want)
	}
}

func TestGenerateDatasetIsDeterministic(t *testing.T) {
	first, err := generateDataset(100, 40, 42)
	if err != nil {
		t.Fatalf("generate first dataset: %v", err)
	}
	second, err := generateDataset(100, 40, 42)
	if err != nil {
		t.Fatalf("generate second dataset: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("datasets generated with the same seed differ")
	}

	different, err := generateDataset(100, 40, 43)
	if err != nil {
		t.Fatalf("generate dataset with different seed: %v", err)
	}
	if reflect.DeepEqual(first, different) {
		t.Fatal("datasets generated with different seeds are identical")
	}

	summary := summarizeDataset(first)
	if summary.Trails != 100 || summary.QueryPoints != 40 {
		t.Fatalf("unexpected summary counts: %+v", summary)
	}
	if summary.MultiLineTrails != 10 {
		t.Fatalf("MultiLineString trails = %d, want 10", summary.MultiLineTrails)
	}
	for _, scenario := range datasetScenarios {
		if summary.Scenarios[scenario] == 0 {
			t.Fatalf("scenario %q is missing from trails", scenario)
		}
		foundQuery := false
		for _, query := range first.QueryPoints {
			if query.Scenario == scenario {
				foundQuery = true
				break
			}
		}
		if !foundQuery {
			t.Fatalf("scenario %q is missing from query points", scenario)
		}
	}
}

func TestMutateTrailsDoesNotModifyInput(t *testing.T) {
	dataset, err := generateDataset(30, 9, 7)
	if err != nil {
		t.Fatalf("generate dataset: %v", err)
	}
	original := cloneTrailsForTest(dataset.Trails)

	mutated := mutateTrails(dataset.Trails, 6)
	if len(mutated) != 6 {
		t.Fatalf("mutated trail count = %d, want 6", len(mutated))
	}
	if !reflect.DeepEqual(dataset.Trails, original) {
		t.Fatal("mutateTrails modified its input")
	}
	for i, trail := range mutated {
		index := i * (len(original) - 1) / (len(mutated) - 1)
		if trail.ID != original[index].ID {
			t.Fatalf("mutated trail %d id = %q, want %q", i, trail.ID, original[index].ID)
		}
		if reflect.DeepEqual(trail.Parts, original[index].Parts) {
			t.Fatalf("mutated trail %d has unchanged geometry", i)
		}
	}
	single := mutateTrails(dataset.Trails, 1)
	if len(single) != 1 || single[0].Scenario != scenarioLongOutlier {
		t.Fatalf("single incremental update = %+v, want a long-outlier trail", single)
	}
}

func cloneTrailsForTest(trails []Trail) []Trail {
	clone := make([]Trail, len(trails))
	for i, trail := range trails {
		clone[i] = Trail{ID: trail.ID, Scenario: trail.Scenario, Parts: make([][]Coordinate, len(trail.Parts))}
		for j, part := range trail.Parts {
			clone[i].Parts[j] = append([]Coordinate(nil), part...)
		}
	}
	return clone
}
