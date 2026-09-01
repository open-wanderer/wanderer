package main

import (
	"fmt"
	"math"
	"sort"
)

type geoJSONDirectObservation struct {
	violation                GeoJSONDirectCaseViolation
	returned                 int
	expected                 int
	truePositives            int
	falsePositives           int
	falseNegatives           int
	clearExpected            int
	clearTruePositives       int
	materialFalseNegatives   int
	boundaryMisses           int
	allowedBoundaryFalsePos  int
	materialFalsePositives   int
	semanticFalsePositives   int
	nearestExpected          int
	nearestReturned          int
	top10Expected            int
	top10Returned            int
	comparablePageExpected   int
	comparablePageReturned   int
	comparablePageActual     int
	comparablePageMaterialFP int
	comparablePage           bool
	unsupportedSort          bool
	truncated                bool
	falseEmpty               bool
	maxInsideMiss            float64
	maxOutsideHit            float64
	materialFalseNegativeIDs []GeoJSONDirectTrailViolationDetail
	materialFalsePositiveIDs []GeoJSONDirectTrailViolationDetail
	semanticError            string
}

const maxGeoJSONDirectTrailViolationDetails = 5

type geoJSONDirectAccuracyAccumulator struct {
	report         GeoJSONDirectAccuracy
	absoluteErrors []float64
	relativeErrors []float64
}

func newGeoJSONDirectAccuracyAccumulator(config GeoJSONDirectConfig) *geoJSONDirectAccuracyAccumulator {
	return &geoJSONDirectAccuracyAccumulator{report: GeoJSONDirectAccuracy{
		Status:                         "measuring",
		BoundaryToleranceMeters:        config.BoundaryToleranceMeters,
		MinimumRecall:                  config.MinimumRecall,
		MinimumPrecision:               config.MinimumPrecision,
		MinimumTop10Recall:             config.MinimumTop10Recall,
		MinimumComparablePageRecall:    config.MinimumComparablePageRate,
		MinimumComparablePagePrecision: config.MinimumComparablePagePrecision,
		WorstRequestRecall:             1,
		WorstComparablePageRecall:      1,
		WorstComparablePagePrecision:   1,
	}}
}

func (a *geoJSONDirectAccuracyAccumulator) add(observation geoJSONDirectObservation) {
	report := &a.report
	report.Requests++
	if observation.expected > 0 {
		report.RequestsWithExpected++
	}
	if observation.clearExpected > 0 {
		report.RequestsWithClearExpected++
	}
	report.ReturnedHits += observation.returned
	report.ExpectedHits += observation.expected
	report.TruePositives += observation.truePositives
	report.FalsePositives += observation.falsePositives
	report.FalseNegatives += observation.falseNegatives
	report.ClearExpectedHits += observation.clearExpected
	report.ClearTruePositives += observation.clearTruePositives
	report.MaterialFalseNegatives += observation.materialFalseNegatives
	report.BoundaryMisses += observation.boundaryMisses
	report.AllowedBoundaryFalsePositives += observation.allowedBoundaryFalsePos
	report.MaterialFalsePositives += observation.materialFalsePositives
	report.SemanticFalsePositives += observation.semanticFalsePositives
	report.NearestExpected += observation.nearestExpected
	report.NearestReturned += observation.nearestReturned
	report.Top10Expected += observation.top10Expected
	report.Top10Returned += observation.top10Returned
	report.ComparablePageExpected += observation.comparablePageExpected
	report.ComparablePageReturned += observation.comparablePageReturned
	report.ComparablePageActual += observation.comparablePageActual
	report.ComparablePageMaterialFP += observation.comparablePageMaterialFP
	if observation.comparablePage {
		report.ComparablePageCases++
		if observation.comparablePageExpected > 0 {
			report.WorstComparablePageRecall = min(
				report.WorstComparablePageRecall,
				ratioOrOne(observation.comparablePageReturned, observation.comparablePageExpected),
			)
		}
		if observation.comparablePageActual > 0 {
			report.WorstComparablePagePrecision = min(
				report.WorstComparablePagePrecision,
				ratioOrOne(observation.comparablePageActual-observation.comparablePageMaterialFP, observation.comparablePageActual),
			)
		}
	}
	if observation.unsupportedSort {
		report.UnsupportedSortCases++
	}
	if observation.truncated {
		report.TruncatedRequests++
	}
	if observation.falseEmpty {
		report.FalseEmptyRequests++
	}
	report.MaxInsideMissMeters = max(report.MaxInsideMissMeters, observation.maxInsideMiss)
	report.MaxOutsideHitMeters = max(report.MaxOutsideHitMeters, observation.maxOutsideHit)

	if observation.clearExpected > 0 {
		requestRecall := ratioOrOne(observation.clearTruePositives, observation.clearExpected)
		report.WorstRequestRecall = min(report.WorstRequestRecall, requestRecall)
	}
	absoluteError := math.Abs(float64(observation.returned - observation.expected))
	relativeError := absoluteError / float64(max(1, observation.expected))
	a.absoluteErrors = append(a.absoluteErrors, absoluteError)
	a.relativeErrors = append(a.relativeErrors, relativeError)

	if observation.materialFalseNegatives > 0 || observation.materialFalsePositives > 0 ||
		observation.semanticFalsePositives > 0 || observation.falseEmpty || observation.truncated {
		if len(report.Violations) < 20 {
			report.Violations = append(report.Violations, observation.violation)
		}
	}
}

func (a *geoJSONDirectAccuracyAccumulator) finish(config GeoJSONDirectConfig, requireComparablePage bool) GeoJSONDirectAccuracy {
	report := a.report
	report.Recall = ratioOrOne(report.TruePositives, report.ExpectedHits)
	// The UX precision denominator ignores the neutral boundary band. A result
	// inside that band is neither rewarded nor punished.
	report.Precision = ratioOrOne(report.ClearTruePositives, report.ClearTruePositives+report.MaterialFalsePositives+report.SemanticFalsePositives)
	report.ClearRecall = ratioOrOne(report.ClearTruePositives, report.ClearExpectedHits)
	report.NearestRecall = ratioOrOne(report.NearestReturned, report.NearestExpected)
	report.Top10Recall = ratioOrOne(report.Top10Returned, report.Top10Expected)
	report.ComparablePageRecall = ratioOrOne(report.ComparablePageReturned, report.ComparablePageExpected)
	report.ComparablePagePrecision = ratioOrOne(report.ComparablePageActual-report.ComparablePageMaterialFP, report.ComparablePageActual)
	report.NonEmptyRequestSuccessRate = ratioOrOne(
		report.RequestsWithClearExpected-report.FalseEmptyRequests,
		report.RequestsWithClearExpected,
	)
	report.CountError = directCountError(a.absoluteErrors, a.relativeErrors)

	coreEvidence := report.Requests > 0 &&
		report.RequestsWithExpected > 0 &&
		report.ExpectedHits > 0 &&
		report.ClearExpectedHits > 0 &&
		report.NearestExpected > 0 &&
		report.Top10Expected > 0
	qualified := coreEvidence &&
		report.TruncatedRequests == 0 &&
		report.SemanticFalsePositives == 0 &&
		report.Recall >= config.MinimumRecall &&
		report.ClearRecall >= config.MinimumRecall &&
		report.NearestRecall >= config.MinimumRecall &&
		report.NonEmptyRequestSuccessRate >= config.MinimumRecall &&
		report.Precision >= config.MinimumPrecision &&
		report.Top10Recall >= config.MinimumTop10Recall
	if requireComparablePage {
		qualified = qualified && report.ComparablePageCases > 0 &&
			report.ComparablePageExpected > 0 &&
			report.ComparablePageActual > 0 &&
			report.ComparablePageRecall >= config.MinimumComparablePageRate &&
			report.ComparablePagePrecision >= config.MinimumComparablePagePrecision
	}
	if qualified {
		report.Status = "passed"
	} else if !coreEvidence || (requireComparablePage && (report.ComparablePageCases == 0 || report.ComparablePageExpected == 0)) {
		report.Status = "inconclusive"
	} else {
		report.Status = "failed"
	}
	return report
}

func directCountError(absolute, relative []float64) GeoJSONDirectCountError {
	if len(absolute) == 0 {
		return GeoJSONDirectCountError{}
	}
	absStats := latencyStats(absolute)
	relStats := latencyStats(relative)
	return GeoJSONDirectCountError{
		MeanAbsolute: absStats.Mean,
		P95Absolute:  absStats.P95,
		MaxAbsolute:  int(math.Round(absStats.Max)),
		MeanRelative: relStats.Mean,
		P95Relative:  relStats.P95,
		MaxRelative:  relStats.Max,
	}
}

func ratioOrOne(numerator, denominator int) float64 {
	if denominator == 0 {
		return 1
	}
	return float64(numerator) / float64(denominator)
}

func directGeoJSONPlanEligible(plan GeoJSONQueryPlan, config GeoJSONDirectConfig) bool {
	return config.Enabled && plan.RadiusMode == geoJSONRadiusExact &&
		containsInt(config.Resolutions, plan.Resolution)
}

func containsInt(values []int, wanted int) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func selectGeoJSONDirect(sweep *GeoJSONSweepReport, config GeoJSONDirectConfig) *GeoJSONDirectReport {
	if !config.Enabled {
		return nil
	}
	report := &GeoJSONDirectReport{
		SelectionRule:   "UX pass outside the configured boundary band; engine-only runs then use lowest candidate p95, p99, fan-out and resolution",
		SelectionStatus: "no_qualifying_plan",
	}
	if sweep == nil {
		report.SelectionStatus = "inconclusive"
		report.Error = "GeoJSON direct requires the parameter screen"
		return report
	}
	indices := rankGeoJSONDirectCandidates(sweep, config)
	if len(indices) == 0 {
		return report
	}
	selected := sweep.Variants[indices[0]]
	plan := selected.Plan
	accuracy := *selected.DirectUX
	report.SelectedID = plan.ID
	report.SelectionStatus = "spatial_qualified"
	report.Plan = &plan
	report.Accuracy = &accuracy
	return report
}

// rankGeoJSONDirectCandidates returns every plan that passed the spatial UX
// screen. Product-workload runs audit all of these plans before choosing one;
// this engine ranking is only the deterministic tie-breaker.
func rankGeoJSONDirectCandidates(sweep *GeoJSONSweepReport, config GeoJSONDirectConfig) []int {
	if sweep == nil || !config.Enabled {
		return nil
	}
	indices := make([]int, 0, len(sweep.Variants))
	for index := range sweep.Variants {
		variant := &sweep.Variants[index]
		if directGeoJSONPlanEligible(variant.Plan, config) && variant.DirectUX != nil && variant.DirectUX.Status == "passed" {
			indices = append(indices, index)
		}
	}
	sort.SliceStable(indices, func(i, j int) bool {
		left, right := sweep.Variants[indices[i]], sweep.Variants[indices[j]]
		if left.CandidateLatency.P95 != right.CandidateLatency.P95 {
			return left.CandidateLatency.P95 < right.CandidateLatency.P95
		}
		if left.CandidateLatency.P99 != right.CandidateLatency.P99 {
			return left.CandidateLatency.P99 < right.CandidateLatency.P99
		}
		if left.AverageCandidates != right.AverageCandidates {
			return left.AverageCandidates < right.AverageCandidates
		}
		return left.Plan.Resolution < right.Plan.Resolution
	})
	return indices
}

func (o *accuracyOracle) observeGeoJSONDirect(
	queryPoint QueryPoint,
	radius float64,
	ids map[string]struct{},
	config GeoJSONDirectConfig,
	truncated bool,
) (geoJSONDirectObservation, error) {
	query, ok := o.queries[queryPoint.ID]
	if !ok {
		return geoJSONDirectObservation{}, fmt.Errorf("query %q is missing from the accuracy oracle", queryPoint.ID)
	}
	if len(query.distances) != len(o.trailIDs) {
		return geoJSONDirectObservation{}, fmt.Errorf("query %q has an invalid accuracy matrix", queryPoint.ID)
	}

	observation := geoJSONDirectObservation{returned: len(ids), truncated: truncated}
	clear := make([]int, 0)
	knownReturned := 0
	numericalTolerance := math.Max(0.5, radius*1e-7)
	clearThreshold := radius - config.BoundaryToleranceMeters - numericalTolerance
	outerThreshold := radius + config.BoundaryToleranceMeters + numericalTolerance
	for trailIndex, distance := range query.distances {
		id := o.trailIDs[trailIndex]
		_, found := ids[id]
		if found {
			knownReturned++
		}
		inside := distance <= radius
		clearInside := distance < clearThreshold
		switch {
		case inside:
			observation.expected++
			if clearInside {
				observation.clearExpected++
				clear = append(clear, trailIndex)
			}
			if found {
				observation.truePositives++
				if clearInside {
					observation.clearTruePositives++
				}
			} else {
				observation.falseNegatives++
				observation.maxInsideMiss = max(observation.maxInsideMiss, radius-distance)
				if clearInside {
					observation.materialFalseNegatives++
					observation.materialFalseNegativeIDs = retainWorstGeoJSONDirectTrailViolation(
						observation.materialFalseNegativeIDs,
						GeoJSONDirectTrailViolationDetail{
							TrailID:           id,
							DistanceMeters:    distance,
							RadiusDeltaMeters: radius - distance,
						},
					)
				} else {
					observation.boundaryMisses++
				}
			}
		case found:
			observation.falsePositives++
			outside := distance - radius
			observation.maxOutsideHit = max(observation.maxOutsideHit, outside)
			if distance <= outerThreshold {
				observation.allowedBoundaryFalsePos++
			} else {
				observation.materialFalsePositives++
				observation.materialFalsePositiveIDs = retainWorstGeoJSONDirectTrailViolation(
					observation.materialFalsePositiveIDs,
					GeoJSONDirectTrailViolationDetail{
						TrailID:           id,
						DistanceMeters:    distance,
						RadiusDeltaMeters: outside,
					},
				)
			}
		}
	}
	unknown := len(ids) - knownReturned
	observation.falsePositives += unknown
	observation.materialFalsePositives += unknown

	sort.Slice(clear, func(i, j int) bool {
		left, right := clear[i], clear[j]
		if query.distances[left] != query.distances[right] {
			return query.distances[left] < query.distances[right]
		}
		return o.trailIDs[left] < o.trailIDs[right]
	})
	if len(clear) > 0 {
		observation.nearestExpected = 1
		if _, found := ids[o.trailIDs[clear[0]]]; found {
			observation.nearestReturned = 1
		}
		observation.top10Expected = min(10, len(clear))
		for _, trailIndex := range clear[:observation.top10Expected] {
			if _, found := ids[o.trailIDs[trailIndex]]; found {
				observation.top10Returned++
			}
		}
		observation.falseEmpty = observation.clearTruePositives == 0
	}

	recall := ratioOrOne(observation.truePositives, observation.expected)
	observation.violation = GeoJSONDirectCaseViolation{
		QueryID:               queryPoint.ID,
		Scenario:              queryPoint.Scenario,
		RadiusMeters:          radius,
		Expected:              observation.expected,
		Returned:              observation.returned,
		FalseNegatives:        observation.falseNegatives,
		MaterialFalseNegative: observation.materialFalseNegatives,
		FalsePositives:        observation.falsePositives,
		MaterialFalsePositive: observation.materialFalsePositives,
		Recall:                recall,
		MaxInsideMissMeters:   observation.maxInsideMiss,
		MaxOutsideHitMeters:   observation.maxOutsideHit,
		MaterialFalseNegativeTrails: append([]GeoJSONDirectTrailViolationDetail(nil),
			observation.materialFalseNegativeIDs...),
		MaterialFalsePositiveTrails: append([]GeoJSONDirectTrailViolationDetail(nil),
			observation.materialFalsePositiveIDs...),
		FalseEmpty: observation.falseEmpty,
	}
	return observation, nil
}

func retainWorstGeoJSONDirectTrailViolation(
	details []GeoJSONDirectTrailViolationDetail,
	detail GeoJSONDirectTrailViolationDetail,
) []GeoJSONDirectTrailViolationDetail {
	details = append(details, detail)
	sort.Slice(details, func(i, j int) bool {
		if details[i].RadiusDeltaMeters != details[j].RadiusDeltaMeters {
			return details[i].RadiusDeltaMeters > details[j].RadiusDeltaMeters
		}
		return details[i].TrailID < details[j].TrailID
	})
	if len(details) > maxGeoJSONDirectTrailViolationDetails {
		details = details[:maxGeoJSONDirectTrailViolationDetails]
	}
	return details
}
