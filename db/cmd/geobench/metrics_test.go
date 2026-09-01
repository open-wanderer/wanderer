package main

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
)

func TestLatencyStats(t *testing.T) {
	stats := latencyStats([]float64{5, 1, 2, 3, 4})
	if stats.Min != 1 || stats.P50 != 3 || stats.Max != 5 || stats.Mean != 3 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if math.Abs(stats.P95-4.8) > 1e-9 {
		t.Fatalf("p95 = %v", stats.P95)
	}
}

func TestRefineMatchesHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := refineMatches(
		ctx,
		map[string]struct{}{"trail": {}},
		map[string]Trail{"trail": {ID: "trail", Parts: [][]Coordinate{{{Lat: 47, Lon: 8}, {Lat: 47.1, Lon: 8.1}}}}},
		Coordinate{Lat: 47, Lon: 8},
		500,
		true,
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("refineMatches error = %v, want context.Canceled", err)
	}
}

func TestRefineCandidateGeometriesRequiresEveryDatabaseRecord(t *testing.T) {
	loader := func(context.Context, map[string]struct{}) (geometryLoad, error) {
		return geometryLoad{Trails: map[string]Trail{"a": {ID: "a"}}, Rows: 1}, nil
	}
	_, err := refineCandidateGeometries(
		context.Background(),
		idSet("a", "missing"),
		Coordinate{},
		500,
		loader,
	)
	if err == nil || !strings.Contains(err.Error(), "loaded 1 records") {
		t.Fatalf("refinement error = %v", err)
	}
}

func TestSetMetricsExcludeBoundary(t *testing.T) {
	a := map[string]struct{}{"a": {}, "b": {}, "boundary": {}}
	b := map[string]struct{}{"a": {}, "c": {}, "boundary": {}}
	excluded := map[string]struct{}{"boundary": {}}
	if got := intersectionSize(a, b, excluded); got != 1 {
		t.Fatalf("intersection = %d", got)
	}
	if got := differenceSize(a, b, excluded); got != 1 {
		t.Fatalf("difference = %d", got)
	}
}

func TestCompletedBackendStatusKeepsCorrectnessSeparateFromExecution(t *testing.T) {
	if got := completedBackendStatus(BackendReport{Queries: []QueryReport{{TruePositives: 2}}}); got != "ok" {
		t.Fatalf("correct status = %q", got)
	}
	if got := completedBackendStatus(BackendReport{Queries: []QueryReport{{FalseNegatives: 1}}}); got != "incorrect" {
		t.Fatalf("engine mismatch status = %q", got)
	}
	if got := completedBackendStatus(BackendReport{ProductWorkload: &ProductWorkloadReport{Status: "incorrect"}}); got != "incorrect" {
		t.Fatalf("product mismatch status = %q", got)
	}
}

func TestAccuracyOracleMatchesExactReference(t *testing.T) {
	dataset := Dataset{
		Trails: []Trail{
			{ID: "crossing", Parts: [][]Coordinate{{{Lat: 0, Lon: -0.01}, {Lat: 0, Lon: 0.01}}}},
			{ID: "near", Parts: [][]Coordinate{{{Lat: 0.01, Lon: -0.01}, {Lat: 0.01, Lon: 0.01}}}},
			{ID: "far", Parts: [][]Coordinate{{{Lat: 1, Lon: -0.01}, {Lat: 1, Lon: 0.01}}}},
		},
		QueryPoints: []QueryPoint{
			{ID: "query", Point: Coordinate{Lat: 0, Lon: 0}},
			{ID: "empty-query", Point: Coordinate{Lat: -10, Lon: 0}},
		},
	}
	radii := []float64{500, 5_000}
	oracle, err := buildAccuracyOracle(context.Background(), dataset, radii)
	if err != nil {
		t.Fatalf("build oracle: %v", err)
	}

	all := map[string]struct{}{"crossing": {}, "near": {}, "far": {}, "unknown": {}}
	for _, radius := range radii {
		expected, boundary := exactMatches(dataset.Trails, dataset.QueryPoints[0].Point, radius)
		expectedCount, boundaryCount, err := oracle.counts("query", radius)
		if err != nil {
			t.Fatalf("oracle counts at %.0f m: %v", radius, err)
		}
		if expectedCount != len(expected) || boundaryCount != len(boundary) {
			t.Fatalf("oracle counts at %.0f m = (%d, %d), want (%d, %d)", radius, expectedCount, boundaryCount, len(expected), len(boundary))
		}
		truePositives, falsePositives, err := oracle.classify("query", radius, all)
		if err != nil {
			t.Fatalf("classify at %.0f m: %v", radius, err)
		}
		wantFalse := len(dataset.Trails) - len(expected) - len(boundary) + 1
		if truePositives != len(expected) || falsePositives != wantFalse {
			t.Fatalf("classification at %.0f m = (%d, %d), want (%d, %d)", radius, truePositives, falsePositives, len(expected), wantFalse)
		}
	}
	expectedCount, boundaryCount, err := oracle.counts("empty-query", 500)
	if err != nil {
		t.Fatalf("zero-match oracle counts: %v", err)
	}
	if expectedCount != 0 || boundaryCount != 0 {
		t.Fatalf("zero-match oracle counts = (%d, %d), want (0, 0)", expectedCount, boundaryCount)
	}
}

func TestBenchmarkQueriesCountsRawBoundaryCandidates(t *testing.T) {
	dataset := Dataset{
		Trails:      []Trail{{ID: "boundary"}},
		QueryPoints: []QueryPoint{{ID: "query", Scenario: "boundary", Point: Coordinate{}}},
	}
	oracle := &accuracyOracle{
		trailIndex: map[string]int{"boundary": 0},
		trailIDs:   []string{"boundary"},
		queries: map[string]oracleQuery{
			"query": {
				distances:      []float64{100},
				expectedCounts: map[float64]int{100: 0},
				boundaryCounts: map[float64]int{100: 1},
			},
		},
	}
	reports, err := benchmarkQueries(
		context.Background(),
		dataset,
		[]float64{100},
		1,
		nil,
		oracle,
		func(context.Context, QueryPoint, float64) (CandidateResult, error) {
			return CandidateResult{IDs: idSet("boundary")}, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 || reports[0].AverageCandidates != 1 || reports[0].BoundaryExcludedCount != 1 {
		t.Fatalf("query reports = %+v", reports)
	}
}
