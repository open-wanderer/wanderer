package main

import (
	"math"
	"testing"
)

func TestObserveGeoJSONDirectUsesThreeZonesIncludingNumericalEpsilon(t *testing.T) {
	const radius = 1_000.0
	// At this radius the numerical epsilon is 0.5 m, so the clear-inside
	// threshold is 949.5 m and the allowed outer threshold is 1,050.5 m.
	oracle := directAccuracyTestOracle([]directAccuracyTestTrail{
		{id: "deep-inside", distance: 900},
		{id: "clear-edge", distance: 949.499},
		{id: "neutral-inside-edge", distance: 949.5},
		{id: "radius-edge", distance: 1_000},
		{id: "neutral-outside-edge", distance: 1_050.5},
		{id: "material-outside", distance: 1_050.501},
	})

	observation, err := oracle.observeGeoJSONDirect(
		QueryPoint{ID: "query", Scenario: scenarioAlpineDense},
		radius,
		idSet("deep-inside", "neutral-inside-edge", "radius-edge", "neutral-outside-edge", "material-outside"),
		directAccuracyTestConfig(),
		false,
	)
	if err != nil {
		t.Fatal(err)
	}

	if observation.expected != 4 || observation.truePositives != 3 || observation.falseNegatives != 1 {
		t.Fatalf("raw target-radius classification = %+v", observation)
	}
	if observation.clearExpected != 2 || observation.clearTruePositives != 1 || observation.materialFalseNegatives != 1 {
		t.Fatalf("clear-inside classification = %+v", observation)
	}
	if observation.boundaryMisses != 0 {
		t.Fatalf("boundary misses = %d, want 0", observation.boundaryMisses)
	}
	if observation.falsePositives != 2 || observation.allowedBoundaryFalsePos != 1 || observation.materialFalsePositives != 1 {
		t.Fatalf("outside classification = %+v", observation)
	}
	if observation.falseEmpty {
		t.Fatal("returning one clear result must not be classified as false-empty")
	}
	if math.Abs(observation.maxInsideMiss-50.501) > 1e-9 {
		t.Fatalf("max inside miss = %.9f, want 50.501", observation.maxInsideMiss)
	}
	if math.Abs(observation.maxOutsideHit-50.501) > 1e-9 {
		t.Fatalf("max outside hit = %.9f, want 50.501", observation.maxOutsideHit)
	}
	if len(observation.violation.MaterialFalseNegativeTrails) != 1 ||
		observation.violation.MaterialFalseNegativeTrails[0].TrailID != "clear-edge" ||
		math.Abs(observation.violation.MaterialFalseNegativeTrails[0].RadiusDeltaMeters-50.501) > 1e-9 {
		t.Fatalf("material false-negative details = %+v", observation.violation.MaterialFalseNegativeTrails)
	}
	if len(observation.violation.MaterialFalsePositiveTrails) != 1 ||
		observation.violation.MaterialFalsePositiveTrails[0].TrailID != "material-outside" ||
		math.Abs(observation.violation.MaterialFalsePositiveTrails[0].DistanceMeters-1_050.501) > 1e-9 ||
		observation.violation.MaxOutsideHitMeters != observation.maxOutsideHit {
		t.Fatalf("material false-positive details = %+v; violation=%+v", observation.violation.MaterialFalsePositiveTrails, observation.violation)
	}
}

func TestRetainWorstGeoJSONDirectTrailViolationsIsBoundedAndDeterministic(t *testing.T) {
	var details []GeoJSONDirectTrailViolationDetail
	for _, detail := range []GeoJSONDirectTrailViolationDetail{
		{TrailID: "z", RadiusDeltaMeters: 2},
		{TrailID: "d", RadiusDeltaMeters: 4},
		{TrailID: "c", RadiusDeltaMeters: 4},
		{TrailID: "a", RadiusDeltaMeters: 8},
		{TrailID: "e", RadiusDeltaMeters: 3},
		{TrailID: "b", RadiusDeltaMeters: 7},
		{TrailID: "discarded", RadiusDeltaMeters: 1},
	} {
		details = retainWorstGeoJSONDirectTrailViolation(details, detail)
	}
	want := []string{"a", "b", "c", "d", "e"}
	if len(details) != len(want) {
		t.Fatalf("retained details = %+v, want %d", details, len(want))
	}
	for index, id := range want {
		if details[index].TrailID != id {
			t.Fatalf("retained details = %+v, want IDs %v", details, want)
		}
	}
}

func TestObserveGeoJSONDirectScalesNumericalEpsilonWithRadius(t *testing.T) {
	const radius = 10_000_000.0
	// radius*1e-7 is 1 m here, so the neutral band extends through +/-51 m.
	oracle := directAccuracyTestOracle([]directAccuracyTestTrail{
		{id: "clear-inside", distance: radius - 51.001},
		{id: "neutral-inside-edge", distance: radius - 51},
		{id: "neutral-outside-edge", distance: radius + 51},
		{id: "material-outside", distance: radius + 51.001},
	})

	observation, err := oracle.observeGeoJSONDirect(
		QueryPoint{ID: "query"},
		radius,
		idSet("neutral-inside-edge", "neutral-outside-edge", "material-outside"),
		directAccuracyTestConfig(),
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if observation.materialFalseNegatives != 1 || observation.boundaryMisses != 0 {
		t.Fatalf("scaled inner epsilon classification = %+v", observation)
	}
	if observation.allowedBoundaryFalsePos != 1 || observation.materialFalsePositives != 1 {
		t.Fatalf("scaled outer epsilon classification = %+v", observation)
	}
}

func TestObserveGeoJSONDirectTreatsSmallMissAsNeutralAndDeepMissAsMaterial(t *testing.T) {
	oracle := directAccuracyTestOracle([]directAccuracyTestTrail{
		{id: "miss-2p42m", distance: 997.58},
		{id: "miss-60m", distance: 940},
	})

	observation, err := oracle.observeGeoJSONDirect(
		QueryPoint{ID: "query"},
		1_000,
		idSet(),
		directAccuracyTestConfig(),
		false,
	)
	if err != nil {
		t.Fatal(err)
	}

	if observation.falseNegatives != 2 || observation.boundaryMisses != 1 || observation.materialFalseNegatives != 1 {
		t.Fatalf("miss classification = %+v", observation)
	}
	if observation.clearExpected != 1 || !observation.falseEmpty {
		t.Fatalf("clear expectation/false-empty = %d/%v, want 1/true", observation.clearExpected, observation.falseEmpty)
	}
	if observation.maxInsideMiss != 60 {
		t.Fatalf("max inside miss = %.2f, want 60", observation.maxInsideMiss)
	}
}

func TestObserveGeoJSONDirectFalseEmptyRequiresClearExpectedResult(t *testing.T) {
	tests := []struct {
		name       string
		distance   float64
		falseEmpty bool
	}{
		{name: "boundary-only", distance: 997.58, falseEmpty: false},
		{name: "clear-result", distance: 940, falseEmpty: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			oracle := directAccuracyTestOracle([]directAccuracyTestTrail{{id: "trail", distance: test.distance}})
			observation, err := oracle.observeGeoJSONDirect(
				QueryPoint{ID: "query"}, 1_000, idSet(), directAccuracyTestConfig(), false,
			)
			if err != nil {
				t.Fatal(err)
			}
			if observation.falseEmpty != test.falseEmpty {
				t.Fatalf("falseEmpty = %v, want %v; observation=%+v", observation.falseEmpty, test.falseEmpty, observation)
			}
		})
	}
}

func TestObserveGeoJSONDirectScoresNearestAndTop10FromClearResults(t *testing.T) {
	trails := make([]directAccuracyTestTrail, 0, 13)
	returned := make([]string, 0, 10)
	for index := 0; index < 12; index++ {
		id := string(rune('a' + index))
		trails = append(trails, directAccuracyTestTrail{id: id, distance: float64(100 + index)})
		if index != 4 && index < 10 {
			returned = append(returned, id)
		}
	}
	// A target-radius hit in the neutral band must not displace or enlarge the
	// nearest/top-10 denominator.
	trails = append(trails, directAccuracyTestTrail{id: "neutral", distance: 999})
	returned = append(returned, "neutral")

	oracle := directAccuracyTestOracle(trails)
	observation, err := oracle.observeGeoJSONDirect(
		QueryPoint{ID: "query"}, 1_000, idSet(returned...), directAccuracyTestConfig(), false,
	)
	if err != nil {
		t.Fatal(err)
	}

	if observation.nearestExpected != 1 || observation.nearestReturned != 1 {
		t.Fatalf("nearest expected/returned = %d/%d, want 1/1", observation.nearestExpected, observation.nearestReturned)
	}
	if observation.top10Expected != 10 || observation.top10Returned != 9 {
		t.Fatalf("top10 expected/returned = %d/%d, want 10/9", observation.top10Expected, observation.top10Returned)
	}
}

func TestObserveGeoJSONDirectSeparatesAllowedAndMaterialFalsePositives(t *testing.T) {
	oracle := directAccuracyTestOracle([]directAccuracyTestTrail{
		{id: "inside", distance: 100},
		{id: "neutral-outside", distance: 1_050.5},
		{id: "material-outside", distance: 1_050.501},
	})
	observation, err := oracle.observeGeoJSONDirect(
		QueryPoint{ID: "query"},
		1_000,
		idSet("inside", "neutral-outside", "material-outside"),
		directAccuracyTestConfig(),
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if observation.falsePositives != 2 || observation.allowedBoundaryFalsePos != 1 || observation.materialFalsePositives != 1 {
		t.Fatalf("false-positive classification = %+v", observation)
	}

	neutralOnly, err := oracle.observeGeoJSONDirect(
		QueryPoint{ID: "query"}, 1_000, idSet("inside", "neutral-outside"), directAccuracyTestConfig(), false,
	)
	if err != nil {
		t.Fatal(err)
	}
	accumulator := newGeoJSONDirectAccuracyAccumulator(directAccuracyTestConfig())
	accumulator.add(neutralOnly)
	report := accumulator.finish(directAccuracyTestConfig(), false)
	if report.Precision != 1 || report.Status != "passed" {
		t.Fatalf("neutral outside hit changed UX precision or status: %+v", report)
	}
}

func TestGeoJSONDirectAccuracyTruncationAlwaysFails(t *testing.T) {
	config := directAccuracyTestConfig()
	oracle := directAccuracyTestOracle([]directAccuracyTestTrail{{id: "inside", distance: 100}})
	observation, err := oracle.observeGeoJSONDirect(
		QueryPoint{ID: "query"}, 1_000, idSet("inside"), config, true,
	)
	if err != nil {
		t.Fatal(err)
	}
	accumulator := newGeoJSONDirectAccuracyAccumulator(config)
	accumulator.add(observation)
	report := accumulator.finish(config, false)
	if report.Status != "failed" || report.TruncatedRequests != 1 {
		t.Fatalf("truncated report = %+v", report)
	}
	if len(report.Violations) != 1 {
		t.Fatalf("truncation violations = %d, want 1", len(report.Violations))
	}
}

func TestGeoJSONDirectAccuracyEmptyEvidenceIsInconclusive(t *testing.T) {
	config := directAccuracyTestConfig()
	accumulator := newGeoJSONDirectAccuracyAccumulator(config)
	accumulator.add(geoJSONDirectObservation{})
	report := accumulator.finish(config, true)
	if report.Status != "inconclusive" {
		t.Fatalf("empty evidence status = %q, want inconclusive; report=%+v", report.Status, report)
	}
}

func TestGeoJSONDirectAccuracyKeepsWorstComparablePageDiagnosticOnly(t *testing.T) {
	config := directAccuracyTestConfig()
	accumulator := newGeoJSONDirectAccuracyAccumulator(config)
	accumulator.add(directAccuracyThresholdObservation(func(observation *geoJSONDirectObservation) {
		observation.comparablePageExpected = 10_000
		observation.comparablePageReturned = 10_000
		observation.comparablePageActual = 10_000
	}))
	accumulator.add(directAccuracyThresholdObservation(func(observation *geoJSONDirectObservation) {
		observation.comparablePageExpected = 100
		observation.comparablePageReturned = 99
		observation.comparablePageActual = 100
		observation.comparablePageMaterialFP = 1
	}))
	report := accumulator.finish(config, true)
	if report.ComparablePageRecall < config.MinimumComparablePageRate ||
		report.ComparablePagePrecision < config.MinimumComparablePagePrecision {
		t.Fatalf("aggregate page metrics should pass to exercise worst-page gate: %+v", report)
	}
	if report.WorstComparablePageRecall != 0.99 || report.WorstComparablePagePrecision != 0.99 || report.Status != "passed" {
		t.Fatalf("worst page was not retained as a non-gating diagnostic: %+v", report)
	}
}

func TestGeoJSONDirectAccuracyUsesAggregateRecallAndFalseEmptyRate(t *testing.T) {
	config := directAccuracyTestConfig()
	config.MinimumRecall = 0.99
	config.MinimumTop10Recall = 0.99

	accumulator := newGeoJSONDirectAccuracyAccumulator(config)
	for index := range 100 {
		miss := index == 0
		observation := directAccuracyThresholdObservation(func(observation *geoJSONDirectObservation) {
			if miss {
				observation.returned = 0
				observation.truePositives = 0
				observation.clearTruePositives = 0
				observation.materialFalseNegatives = 1
				observation.nearestReturned = 0
				observation.top10Returned = 0
				observation.falseNegatives = 1
				observation.falseEmpty = true
			}
		})
		accumulator.add(observation)
	}
	report := accumulator.finish(config, true)
	if report.Status != "passed" {
		t.Fatalf("one miss in 100 should pass the aggregate 99%% contract: %+v", report)
	}
	if report.ClearRecall != 0.99 || report.NearestRecall != 0.99 ||
		report.NonEmptyRequestSuccessRate != 0.99 || report.MaterialFalseNegatives != 1 ||
		report.FalseEmptyRequests != 1 {
		t.Fatalf("aggregate diagnostics = %+v", report)
	}

	below := newGeoJSONDirectAccuracyAccumulator(config)
	for index := range 99 {
		miss := index == 0
		below.add(directAccuracyThresholdObservation(func(observation *geoJSONDirectObservation) {
			if miss {
				observation.returned = 0
				observation.truePositives = 0
				observation.clearTruePositives = 0
				observation.materialFalseNegatives = 1
				observation.nearestReturned = 0
				observation.top10Returned = 0
				observation.falseNegatives = 1
				observation.falseEmpty = true
			}
		}))
	}
	belowReport := below.finish(config, true)
	if belowReport.Status != "failed" || belowReport.NonEmptyRequestSuccessRate >= config.MinimumRecall {
		t.Fatalf("one false-empty in 99 should fail the aggregate 99%% contract: %+v", belowReport)
	}
}

func TestGeoJSONDirectAccuracyStillHardFailsSemanticAndTruncationErrors(t *testing.T) {
	config := directAccuracyTestConfig()
	config.MinimumRecall = 0.99
	for name, mutate := range map[string]func(*geoJSONDirectObservation){
		"semantic":  func(observation *geoJSONDirectObservation) { observation.semanticFalsePositives = 1 },
		"truncated": func(observation *geoJSONDirectObservation) { observation.truncated = true },
	} {
		t.Run(name, func(t *testing.T) {
			accumulator := newGeoJSONDirectAccuracyAccumulator(config)
			for range 1_000 {
				accumulator.add(directAccuracyThresholdObservation(mutate))
			}
			if report := accumulator.finish(config, true); report.Status != "failed" {
				t.Fatalf("%s error was diluted by aggregate evidence: %+v", name, report)
			}
		})
	}
}

func TestGeoJSONDirectAccuracyAcceptsMeasuredNASR100S50At99Percent(t *testing.T) {
	config := directAccuracyTestConfig()
	config.MinimumRecall = 0.99
	config.MinimumPrecision = 0.99
	config.MinimumTop10Recall = 0.99
	config.MinimumComparablePageRate = 0.99
	config.MinimumComparablePagePrecision = 0.99
	accumulator := newGeoJSONDirectAccuracyAccumulator(config)
	accumulator.report.Requests = 240
	accumulator.report.RequestsWithExpected = 188
	accumulator.report.RequestsWithClearExpected = 187
	accumulator.report.ReturnedHits = 1_824_668
	accumulator.report.ExpectedHits = 1_824_938
	accumulator.report.TruePositives = 1_823_952
	accumulator.report.ClearExpectedHits = 1_818_191
	accumulator.report.ClearTruePositives = 1_818_188
	accumulator.report.MaterialFalseNegatives = 3
	accumulator.report.MaterialFalsePositives = 40
	accumulator.report.FalseEmptyRequests = 1
	accumulator.report.NearestExpected = 187
	accumulator.report.NearestReturned = 186
	accumulator.report.Top10Expected = 1_469
	accumulator.report.Top10Returned = 1_468

	report := accumulator.finish(config, false)
	if report.Status != "passed" {
		t.Fatalf("measured NAS r100/s50 spatial result should pass the explicit aggregate 99%% policy: %+v", report)
	}
	if report.NonEmptyRequestSuccessRate <= 0.99 || report.NearestRecall <= 0.99 {
		t.Fatalf("NAS request/nearest rates = %.6f/%.6f, want both above 99%%", report.NonEmptyRequestSuccessRate, report.NearestRecall)
	}
}

func TestGeoJSONDirectAccuracyThresholdsIncludeExact99AndRejectJustBelow(t *testing.T) {
	config := directAccuracyTestConfig()

	atThreshold := newGeoJSONDirectAccuracyAccumulator(config)
	atThreshold.add(geoJSONDirectObservation{
		returned:               99,
		expected:               100,
		truePositives:          99,
		falseNegatives:         1,
		clearExpected:          99,
		clearTruePositives:     99,
		boundaryMisses:         1,
		nearestExpected:        1,
		nearestReturned:        1,
		top10Expected:          100,
		top10Returned:          99,
		comparablePageExpected: 100,
		comparablePageReturned: 99,
		comparablePageActual:   99,
		comparablePage:         true,
	})
	atThresholdReport := atThreshold.finish(config, true)
	if atThresholdReport.Status != "passed" {
		t.Fatalf("exact 99%% report did not pass: %+v", atThresholdReport)
	}
	for name, got := range map[string]float64{
		"recall": atThresholdReport.Recall,
		"top10":  atThresholdReport.Top10Recall,
		"page":   atThresholdReport.ComparablePageRecall,
	} {
		if math.Abs(got-0.99) > 1e-12 {
			t.Fatalf("%s = %.12f, want 0.99", name, got)
		}
	}

	justBelow := newGeoJSONDirectAccuracyAccumulator(config)
	justBelow.add(geoJSONDirectObservation{
		returned:               98,
		expected:               99,
		truePositives:          98,
		falseNegatives:         1,
		clearExpected:          98,
		clearTruePositives:     98,
		boundaryMisses:         1,
		nearestExpected:        1,
		nearestReturned:        1,
		top10Expected:          99,
		top10Returned:          98,
		comparablePageExpected: 99,
		comparablePageReturned: 98,
		comparablePageActual:   98,
		comparablePage:         true,
	})
	justBelowReport := justBelow.finish(config, true)
	if justBelowReport.Recall >= 0.99 || justBelowReport.Top10Recall >= 0.99 || justBelowReport.ComparablePageRecall >= 0.99 {
		t.Fatalf("just-below ratios were not below the threshold: %+v", justBelowReport)
	}
	if justBelowReport.Status != "failed" {
		t.Fatalf("just-below report status = %q, want failed", justBelowReport.Status)
	}
}

func TestGeoJSONDirectAccuracyAppliesEach99ThresholdIndependently(t *testing.T) {
	tests := []struct {
		name        string
		observation geoJSONDirectObservation
		wantStatus  string
	}{
		{
			name: "recall exactly at threshold",
			observation: directAccuracyThresholdObservation(func(observation *geoJSONDirectObservation) {
				observation.returned = 99
				observation.expected = 100
				observation.truePositives = 99
				observation.falseNegatives = 1
				observation.clearExpected = 99
				observation.clearTruePositives = 99
				observation.boundaryMisses = 1
			}),
			wantStatus: "passed",
		},
		{
			name: "recall just below threshold",
			observation: directAccuracyThresholdObservation(func(observation *geoJSONDirectObservation) {
				observation.returned = 98
				observation.expected = 99
				observation.truePositives = 98
				observation.falseNegatives = 1
				observation.clearExpected = 98
				observation.clearTruePositives = 98
				observation.boundaryMisses = 1
			}),
			wantStatus: "failed",
		},
		{
			name: "top10 exactly at threshold",
			observation: directAccuracyThresholdObservation(func(observation *geoJSONDirectObservation) {
				observation.top10Expected = 100
				observation.top10Returned = 99
			}),
			wantStatus: "passed",
		},
		{
			name: "top10 just below threshold",
			observation: directAccuracyThresholdObservation(func(observation *geoJSONDirectObservation) {
				observation.top10Expected = 99
				observation.top10Returned = 98
			}),
			wantStatus: "failed",
		},
		{
			name: "page exactly at threshold",
			observation: directAccuracyThresholdObservation(func(observation *geoJSONDirectObservation) {
				observation.comparablePageExpected = 100
				observation.comparablePageReturned = 99
			}),
			wantStatus: "passed",
		},
		{
			name: "page just below threshold",
			observation: directAccuracyThresholdObservation(func(observation *geoJSONDirectObservation) {
				observation.comparablePageExpected = 99
				observation.comparablePageReturned = 98
			}),
			wantStatus: "failed",
		},
		{
			name: "page precision exactly at threshold",
			observation: directAccuracyThresholdObservation(func(observation *geoJSONDirectObservation) {
				observation.comparablePageActual = 100
				observation.comparablePageMaterialFP = 1
			}),
			wantStatus: "passed",
		},
		{
			name: "page precision just below threshold",
			observation: directAccuracyThresholdObservation(func(observation *geoJSONDirectObservation) {
				observation.comparablePageActual = 99
				observation.comparablePageMaterialFP = 1
			}),
			wantStatus: "failed",
		},
		{
			name: "precision exactly at threshold",
			observation: directAccuracyThresholdObservation(func(observation *geoJSONDirectObservation) {
				observation.returned = 100
				observation.expected = 99
				observation.truePositives = 99
				observation.clearExpected = 99
				observation.clearTruePositives = 99
				observation.falsePositives = 1
				observation.materialFalsePositives = 1
			}),
			wantStatus: "passed",
		},
		{
			name: "precision just below threshold",
			observation: directAccuracyThresholdObservation(func(observation *geoJSONDirectObservation) {
				observation.returned = 99
				observation.expected = 98
				observation.truePositives = 98
				observation.clearExpected = 98
				observation.clearTruePositives = 98
				observation.falsePositives = 1
				observation.materialFalsePositives = 1
			}),
			wantStatus: "failed",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := directAccuracyTestConfig()
			accumulator := newGeoJSONDirectAccuracyAccumulator(config)
			accumulator.add(test.observation)
			report := accumulator.finish(config, true)
			if report.Status != test.wantStatus {
				t.Fatalf("status = %q, want %q; report=%+v", report.Status, test.wantStatus, report)
			}
		})
	}
}

func TestGeoJSONDirectSelectionIsIndependentFromExactWinner(t *testing.T) {
	passed := GeoJSONDirectAccuracy{Status: "passed"}
	sweep := &GeoJSONSweepReport{
		SelectedID: "r64-safe-p0p5",
		Variants: []GeoJSONVariantReport{
			{
				Plan:       GeoJSONQueryPlan{ID: "r64-safe-p0p5", RadiusMode: geoJSONRadiusSafe, Resolution: 64, Conservative: true},
				Finalist:   true,
				Selectable: true,
				ProductWorkload: &ProductWorkloadReport{
					Status:       "correct",
					TotalLatency: LatencyStats{P95: 1, P99: 2},
				},
			},
			{
				Plan:             GeoJSONQueryPlan{ID: "r125-exact", RadiusMode: geoJSONRadiusExact, Resolution: 125},
				CandidateLatency: LatencyStats{P95: 20, P99: 21},
				DirectUX:         &passed,
			},
			{
				Plan:             GeoJSONQueryPlan{ID: "r250-exact", RadiusMode: geoJSONRadiusExact, Resolution: 250},
				CandidateLatency: LatencyStats{P95: 10, P99: 11},
				DirectUX:         &passed,
			},
		},
	}

	if exactWinner := selectGeoJSONWinner(sweep); exactWinner != 0 {
		t.Fatalf("exact winner = %d, want safe finalist at index 0", exactWinner)
	}
	direct := selectGeoJSONDirect(sweep, GeoJSONDirectConfig{Enabled: true, Resolutions: []int{125, 250}})
	if direct == nil || direct.SelectedID != "r250-exact" || direct.SelectionStatus != "spatial_qualified" {
		t.Fatalf("direct selection = %+v, want independently selected r250-exact", direct)
	}
	if sweep.SelectedID != "r64-safe-p0p5" {
		t.Fatalf("direct selection mutated exact selected ID to %q", sweep.SelectedID)
	}
}

type directAccuracyTestTrail struct {
	id       string
	distance float64
}

func directAccuracyTestOracle(trails []directAccuracyTestTrail) *accuracyOracle {
	oracle := &accuracyOracle{
		trailIndex: make(map[string]int, len(trails)),
		trailIDs:   make([]string, len(trails)),
		queries: map[string]oracleQuery{
			"query": {distances: make([]float64, len(trails))},
		},
	}
	query := oracle.queries["query"]
	for index, trail := range trails {
		oracle.trailIndex[trail.id] = index
		oracle.trailIDs[index] = trail.id
		query.distances[index] = trail.distance
	}
	oracle.queries["query"] = query
	return oracle
}

func directAccuracyTestConfig() GeoJSONDirectConfig {
	return GeoJSONDirectConfig{
		Enabled:                        true,
		Resolutions:                    []int{100, 125, 250},
		BoundaryToleranceMeters:        50,
		MinimumRecall:                  0.99,
		MinimumPrecision:               0.99,
		MinimumTop10Recall:             0.99,
		MinimumComparablePageRate:      0.99,
		MinimumComparablePagePrecision: 0.99,
	}
}

func directAccuracyThresholdObservation(mutate func(*geoJSONDirectObservation)) geoJSONDirectObservation {
	observation := geoJSONDirectObservation{
		returned:               1,
		expected:               1,
		truePositives:          1,
		clearExpected:          1,
		clearTruePositives:     1,
		nearestExpected:        1,
		nearestReturned:        1,
		top10Expected:          1,
		top10Returned:          1,
		comparablePageExpected: 1,
		comparablePageReturned: 1,
		comparablePageActual:   1,
		comparablePage:         true,
	}
	mutate(&observation)
	return observation
}
