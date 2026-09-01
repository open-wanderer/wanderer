package main

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestGeoJSONQueryPlansAreDeterministicAndDistinct(t *testing.T) {
	plans := geoJSONQueryPlans(GeoJSONSweepConfig{
		Resolutions:        []int{250, 125, 125},
		ExtraPaddingMeters: []float64{0.5, 0, 0.5},
	})
	if len(plans) != 7 {
		t.Fatalf("plans = %d, want default + 2 resolutions * 3 modes", len(plans))
	}
	wantIDs := []string{
		"default-radius",
		"r125-exact", "r125-safe-p0", "r125-safe-p0p5",
		"r250-exact", "r250-safe-p0", "r250-safe-p0p5",
	}
	gotIDs := make([]string, len(plans))
	for index, plan := range plans {
		gotIDs[index] = plan.ID
	}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("plan IDs = %v, want %v", gotIDs, wantIDs)
	}
}

func TestGeoJSONCandidateRadiusIsConservativeAndMonotone(t *testing.T) {
	radius := 25_000.0
	lowResolution, err := geoJSONCandidateRadius(radius, GeoJSONQueryPlan{
		RadiusMode: geoJSONRadiusSafe,
		Resolution: 125,
	})
	if err != nil {
		t.Fatal(err)
	}
	highResolution, err := geoJSONCandidateRadius(radius, GeoJSONQueryPlan{
		RadiusMode: geoJSONRadiusSafe,
		Resolution: 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	padded, err := geoJSONCandidateRadius(radius, GeoJSONQueryPlan{
		RadiusMode:         geoJSONRadiusSafe,
		Resolution:         1000,
		ExtraPaddingMeters: 0.5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !(lowResolution > highResolution && highResolution > radius) {
		t.Fatalf("candidate radii low/high/target = %.9f/%.9f/%.9f", lowResolution, highResolution, radius)
	}
	if math.Abs((padded-highResolution)-0.5) > 1e-9 {
		t.Fatalf("padding delta = %.12f, want 0.5", padded-highResolution)
	}
	if _, err := geoJSONCandidateRadius(radius, GeoJSONQueryPlan{RadiusMode: geoJSONRadiusSafe, Resolution: 2}); err == nil {
		t.Fatal("resolution below the Meilisearch contract was accepted")
	}
}

func TestGeoJSONSafePlanCoversIndexSimplificationTolerance(t *testing.T) {
	plans := geoJSONQueryPlans(GeoJSONSweepConfig{
		Resolutions:               []int{100},
		ExtraPaddingMeters:        []float64{0.5},
		IndexSimplificationMeters: 50,
	})
	if len(plans) != 3 {
		t.Fatalf("plans = %d, want default, direct, and safe", len(plans))
	}
	plan := plans[2]
	if plan.ID != "r100-safe-p0p5-s50" || plan.ExtraPaddingMeters != 0.5 || plan.IndexSimplificationMeters != 50 {
		t.Fatalf("safe plan = %+v", plan)
	}
	radius, err := geoJSONCandidateRadius(500, plan)
	if err != nil {
		t.Fatal(err)
	}
	want := 500/math.Cos(math.Pi/100) + 50.5
	if math.Abs(radius-want) > 1e-9 {
		t.Fatalf("candidate radius = %.12f, want %.12f", radius, want)
	}
}

func TestGeoJSONDirectPlansRetainIndexSimplificationMetadata(t *testing.T) {
	plans := geoJSONQueryPlans(GeoJSONSweepConfig{
		Resolutions:               []int{100},
		ExtraPaddingMeters:        []float64{0},
		IndexSimplificationMeters: 50,
	})
	if len(plans) < 2 {
		t.Fatalf("plans = %v", plans)
	}
	for _, plan := range plans[:2] {
		if plan.IndexSimplificationMeters != 50 {
			t.Fatalf("plan %s simplification = %v, want 50", plan.ID, plan.IndexSimplificationMeters)
		}
	}
	if plans[1].ExactRefinement || !strings.Contains(plans[1].RadiusFormula, "not exact geometry") {
		t.Fatalf("direct requested-radius plan is ambiguous: %+v", plans[1])
	}
	capacityPlan := geoJSONDirectPlan(100, 50)
	if !reflect.DeepEqual(capacityPlan, plans[1]) {
		t.Fatalf("capacity plan = %+v, sweep plan = %+v", capacityPlan, plans[1])
	}
}

func TestGeoRadiusFilterSupportsExplicitResolution(t *testing.T) {
	point := Coordinate{Lat: 46.81234567, Lon: 8.21234567}
	if got, want := geoRadiusFilter(point, 500, 0), "_geoRadius(46.8123457, 8.2123457, 500)"; got != want {
		t.Fatalf("default filter = %q, want %q", got, want)
	}
	if got, want := geoRadiusFilter(point, 500.5, 1000), "_geoRadius(46.8123457, 8.2123457, 500.5, 1000)"; got != want {
		t.Fatalf("explicit filter = %q, want %q", got, want)
	}
}

func TestGeoJSONCandidateCoverageGatesBoundaryZone(t *testing.T) {
	oracle := &accuracyOracle{
		trailIndex: map[string]int{"inside": 0, "on": 1, "near-outside": 2, "outside": 3},
		trailIDs:   []string{"inside", "on", "near-outside", "outside"},
		queries: map[string]oracleQuery{
			"query": {distances: []float64{99, 100, 100.4, 101}},
		},
	}
	coverage, err := oracle.candidateCoverage("query", 100, idSet("inside", "on", "outside", "unknown"))
	if err != nil {
		t.Fatal(err)
	}
	if coverage.expectedInterior != 1 || coverage.expectedBoundary != 2 {
		t.Fatalf("expected interior/boundary = %d/%d", coverage.expectedInterior, coverage.expectedBoundary)
	}
	if coverage.truePositives != 2 || coverage.falseNegatives != 1 || coverage.boundaryMisses != 1 {
		t.Fatalf("coverage = %+v", coverage)
	}
	if coverage.falsePositives != 2 {
		t.Fatalf("false positives = %d, want known outside plus unknown", coverage.falsePositives)
	}
}

func TestGeoJSONFinalistAndWinnerSelection(t *testing.T) {
	report := &GeoJSONSweepReport{Variants: []GeoJSONVariantReport{
		{Plan: GeoJSONQueryPlan{ID: "unsafe", Resolution: 1000}, Status: "characterization", CandidateLatency: LatencyStats{P95: 1}},
		{Plan: GeoJSONQueryPlan{ID: "slow", Resolution: 125, Conservative: true}, Selectable: true, Status: "screen_qualified", CandidateLatency: LatencyStats{P95: 10}, AverageCandidates: 20},
		{Plan: GeoJSONQueryPlan{ID: "fast", Resolution: 250, Conservative: true}, Selectable: true, Status: "screen_qualified", CandidateLatency: LatencyStats{P95: 5}, AverageCandidates: 30},
	}}
	markGeoJSONFinalists(report, 2)
	if report.Variants[0].Finalist || !report.Variants[1].Finalist || !report.Variants[2].Finalist {
		t.Fatalf("unexpected finalists: %+v", report.Variants)
	}
	report.Variants[1].ProductWorkload = &ProductWorkloadReport{Status: "correct", TotalLatency: LatencyStats{P95: 30, P99: 35}}
	report.Variants[2].ProductWorkload = &ProductWorkloadReport{Status: "correct", TotalLatency: LatencyStats{P95: 20, P99: 25}}
	if winner := selectGeoJSONWinner(report); winner != 2 {
		t.Fatalf("winner = %d, want fast full-workload variant", winner)
	}
	report.Variants[2].ProductWorkload.Status = "incorrect"
	if winner := selectGeoJSONWinner(report); winner != 1 {
		t.Fatalf("winner after correctness failure = %d, want slow variant", winner)
	}
}

func TestGeoJSONCandidateRadiusRejectsInvalidInputs(t *testing.T) {
	for _, radius := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if _, err := geoJSONCandidateRadius(radius, GeoJSONQueryPlan{RadiusMode: geoJSONRadiusExact}); err == nil {
			t.Fatalf("radius %v was accepted", radius)
		}
	}
	_, err := geoJSONCandidateRadius(100, GeoJSONQueryPlan{RadiusMode: "mystery"})
	if err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("unknown mode error = %v", err)
	}
}
