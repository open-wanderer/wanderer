package main

import (
	"context"
	"fmt"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/meilisearch/meilisearch-go"
)

const (
	geoJSONRadiusDefault = "default_radius"
	geoJSONRadiusExact   = "exact_radius"
	geoJSONRadiusSafe    = "inscribed_safe"
)

func geoJSONQueryPlans(config GeoJSONSweepConfig) []GeoJSONQueryPlan {
	plans := []GeoJSONQueryPlan{{
		ID:                        "default-radius",
		RadiusMode:                geoJSONRadiusDefault,
		IndexSimplificationMeters: config.IndexSimplificationMeters,
		RadiusFormula:             "r (three-argument _geoRadius default)",
		Conservative:              false,
		ExactRefinement:           false,
	}}
	for _, resolution := range sortedUniqueInts(config.Resolutions) {
		plans = append(plans, geoJSONDirectPlan(resolution, config.IndexSimplificationMeters))
		for _, padding := range sortedUniqueNonNegativeFloats(config.ExtraPaddingMeters) {
			id := fmt.Sprintf("r%d-safe-p%s", resolution, floatID(padding))
			formula := "r / cos(pi / resolution) + padding"
			if config.IndexSimplificationMeters > 0 {
				id += "-s" + floatID(config.IndexSimplificationMeters)
				formula += " + index simplification tolerance"
			}
			plans = append(plans, GeoJSONQueryPlan{
				ID:                        id,
				RadiusMode:                geoJSONRadiusSafe,
				Resolution:                resolution,
				ExtraPaddingMeters:        padding,
				IndexSimplificationMeters: config.IndexSimplificationMeters,
				RadiusFormula:             formula,
				Conservative:              true,
				ExactRefinement:           true,
			})
		}
	}
	return plans
}

func geoJSONDirectPlan(resolution int, indexSimplificationMeters float64) GeoJSONQueryPlan {
	return GeoJSONQueryPlan{
		ID:                        fmt.Sprintf("r%d-exact", resolution),
		RadiusMode:                geoJSONRadiusExact,
		Resolution:                resolution,
		IndexSimplificationMeters: indexSimplificationMeters,
		RadiusFormula:             "r (requested radius; direct result, not exact geometry)",
		Conservative:              false,
		ExactRefinement:           false,
	}
}

func sortedUniqueNonNegativeFloats(values []float64) []float64 {
	result := append([]float64(nil), values...)
	slices.Sort(result)
	result = slices.Compact(result)
	return result
}

func floatID(value float64) string {
	formatted := strconv.FormatFloat(value, 'f', -1, 64)
	formatted = strings.ReplaceAll(formatted, ".", "p")
	formatted = strings.ReplaceAll(formatted, "-", "m")
	return formatted
}

func geoJSONCandidateRadius(radius float64, plan GeoJSONQueryPlan) (float64, error) {
	if math.IsNaN(radius) || math.IsInf(radius, 0) || radius <= 0 {
		return 0, fmt.Errorf("invalid target radius %v", radius)
	}
	switch plan.RadiusMode {
	case geoJSONRadiusDefault, geoJSONRadiusExact:
		return radius, nil
	case geoJSONRadiusSafe:
		if plan.Resolution < 3 || plan.Resolution > 1000 {
			return 0, fmt.Errorf("GeoJSON resolution %d is outside 3..1000", plan.Resolution)
		}
		if math.IsNaN(plan.ExtraPaddingMeters) || math.IsInf(plan.ExtraPaddingMeters, 0) || plan.ExtraPaddingMeters < 0 {
			return 0, fmt.Errorf("invalid GeoJSON padding %v", plan.ExtraPaddingMeters)
		}
		if math.IsNaN(plan.IndexSimplificationMeters) || math.IsInf(plan.IndexSimplificationMeters, 0) || plan.IndexSimplificationMeters < 0 {
			return 0, fmt.Errorf("invalid GeoJSON index simplification tolerance %v", plan.IndexSimplificationMeters)
		}
		return radius/math.Cos(math.Pi/float64(plan.Resolution)) + plan.ExtraPaddingMeters + plan.IndexSimplificationMeters, nil
	default:
		return 0, fmt.Errorf("unknown GeoJSON radius mode %q", plan.RadiusMode)
	}
}

func geoRadiusFilter(point Coordinate, radius float64, resolution int) string {
	arguments := fmt.Sprintf(
		"%s, %s, %s",
		strconv.FormatFloat(point.Lat, 'f', 7, 64),
		strconv.FormatFloat(point.Lon, 'f', 7, 64),
		strconv.FormatFloat(radius, 'f', -1, 64),
	)
	if resolution > 0 {
		arguments += fmt.Sprintf(", %d", resolution)
	}
	return "_geoRadius(" + arguments + ")"
}

func searchGeoJSONRadius(
	ctx context.Context,
	client meilisearch.ServiceManager,
	point Coordinate,
	radius float64,
	plan GeoJSONQueryPlan,
	limit int64,
) (CandidateResult, error) {
	candidateRadius, err := geoJSONCandidateRadius(radius, plan)
	if err != nil {
		return CandidateResult{}, err
	}
	return searchFilter(ctx, client, geoRadiusFilter(point, candidateRadius, plan.Resolution), limit)
}

type geoJSONCoverage struct {
	expectedInterior int
	expectedBoundary int
	truePositives    int
	falsePositives   int
	falseNegatives   int
	boundaryMisses   int
	maxInsideMiss    float64
}

// candidateCoverage treats the complete numerical boundary zone as required
// candidate input. That is intentionally stricter than the final exact
// predicate: a prefilter may return extra rows, but it must not decide which
// side of a floating-point boundary a later exact stage will see.
func (o *accuracyOracle) candidateCoverage(queryID string, radius float64, ids map[string]struct{}) (geoJSONCoverage, error) {
	query, ok := o.queries[queryID]
	if !ok {
		return geoJSONCoverage{}, fmt.Errorf("query %q is missing from the accuracy oracle", queryID)
	}
	if len(query.distances) != len(o.trailIDs) {
		return geoJSONCoverage{}, fmt.Errorf("query %q has an invalid accuracy matrix", queryID)
	}
	tolerance := math.Max(0.5, radius*1e-7)
	coverage := geoJSONCoverage{}
	knownReturned := 0
	for trailIndex, distance := range query.distances {
		_, found := ids[o.trailIDs[trailIndex]]
		if found {
			knownReturned++
		}
		switch {
		case distance < radius-tolerance:
			coverage.expectedInterior++
			if found {
				coverage.truePositives++
			} else {
				coverage.falseNegatives++
				coverage.maxInsideMiss = max(coverage.maxInsideMiss, radius-distance)
			}
		case distance <= radius+tolerance:
			coverage.expectedBoundary++
			if found {
				coverage.truePositives++
			} else {
				coverage.falseNegatives++
				coverage.boundaryMisses++
			}
		default:
			if found {
				coverage.falsePositives++
			}
		}
	}
	coverage.falsePositives += len(ids) - knownReturned
	return coverage, nil
}

type geoJSONScreenAccumulator struct {
	report          GeoJSONVariantReport
	latencies       []float64
	totalCandidates int64
	directUX        *geoJSONDirectAccuracyAccumulator
}

func screenGeoJSONVariants(
	ctx context.Context,
	client meilisearch.ServiceManager,
	dataset Dataset,
	radii []float64,
	oracle *accuracyOracle,
	plans []GeoJSONQueryPlan,
	limit int64,
	finalists int,
	directConfig GeoJSONDirectConfig,
) (*GeoJSONSweepReport, error) {
	if oracle == nil {
		return nil, fmt.Errorf("GeoJSON parameter sweep requires the exact accuracy oracle")
	}
	if len(plans) == 0 {
		return nil, fmt.Errorf("GeoJSON parameter sweep has no variants")
	}
	accumulators := make([]geoJSONScreenAccumulator, len(plans))
	for index, plan := range plans {
		accumulators[index].report = GeoJSONVariantReport{Plan: plan, Status: "screening"}
		if directGeoJSONPlanEligible(plan, directConfig) {
			accumulators[index].directUX = newGeoJSONDirectAccuracyAccumulator(directConfig)
		}
	}

	caseIndex := 0
	for _, queryPoint := range dataset.QueryPoints {
		for _, radius := range radii {
			start := caseIndex % len(plans)
			for offset := range plans {
				planIndex := (start + offset) % len(plans)
				acc := &accumulators[planIndex]
				if acc.report.Error != "" {
					continue
				}
				candidateRadius, err := geoJSONCandidateRadius(radius, plans[planIndex])
				if err != nil {
					acc.report.Status = "failed"
					acc.report.Error = err.Error()
					continue
				}
				result, err := searchGeoJSONRadius(ctx, client, queryPoint.Point, radius, plans[planIndex], limit)
				if err != nil {
					acc.report.Status = "failed"
					acc.report.Error = err.Error()
					continue
				}
				coverage, err := oracle.candidateCoverage(queryPoint.ID, radius, result.IDs)
				if err != nil {
					return nil, err
				}
				acc.report.ScreenSamples++
				if result.Truncated {
					acc.report.TruncatedSamples++
				}
				acc.latencies = append(acc.latencies, milliseconds(result.WallDuration))
				acc.totalCandidates += int64(len(result.IDs))
				acc.report.MaximumCandidates = max(acc.report.MaximumCandidates, len(result.IDs))
				acc.report.ExpectedInterior += coverage.expectedInterior
				acc.report.ExpectedBoundary += coverage.expectedBoundary
				acc.report.CandidateTruePositives += coverage.truePositives
				acc.report.CandidateFalsePositives += coverage.falsePositives
				acc.report.CandidateFalseNegatives += coverage.falseNegatives
				acc.report.BoundaryCandidateMisses += coverage.boundaryMisses
				acc.report.MaxInsideMissMeters = max(acc.report.MaxInsideMissMeters, coverage.maxInsideMiss)
				acc.report.MaxRadiusExpansionMeters = max(acc.report.MaxRadiusExpansionMeters, candidateRadius-radius)
				if acc.directUX != nil {
					observation, observeErr := oracle.observeGeoJSONDirect(queryPoint, radius, result.IDs, directConfig, result.Truncated)
					if observeErr != nil {
						return nil, observeErr
					}
					acc.directUX.add(observation)
				}
			}
			caseIndex++
		}
	}

	report := &GeoJSONSweepReport{
		SelectionRule: "zero candidate false negatives and zero truncation; fully measure the best screen finalists; select the lowest end-to-end p95",
		Variants:      make([]GeoJSONVariantReport, len(accumulators)),
	}
	for index := range accumulators {
		acc := &accumulators[index]
		acc.report.CandidateLatency = latencyStats(acc.latencies)
		if acc.report.ScreenSamples > 0 {
			acc.report.AverageCandidates = float64(acc.totalCandidates) / float64(acc.report.ScreenSamples)
		}
		expected := acc.report.ExpectedInterior + acc.report.ExpectedBoundary
		acc.report.CandidateRecall = ratio(acc.report.CandidateTruePositives, expected)
		if acc.directUX != nil {
			direct := acc.directUX.finish(directConfig, false)
			acc.report.DirectUX = &direct
		}
		if acc.report.Error != "" {
			acc.report.Status = "failed"
		} else if !acc.report.Plan.Conservative {
			acc.report.Status = "characterization"
		} else if acc.report.CandidateFalseNegatives > 0 || acc.report.TruncatedSamples > 0 {
			acc.report.Status = "screen_incorrect"
		} else {
			acc.report.Status = "screen_qualified"
			acc.report.Selectable = true
		}
		report.Variants[index] = acc.report
	}
	markGeoJSONFinalists(report, finalists)
	return report, nil
}

func markGeoJSONFinalists(report *GeoJSONSweepReport, count int) {
	indices := make([]int, 0, len(report.Variants))
	for index := range report.Variants {
		if report.Variants[index].Selectable {
			indices = append(indices, index)
		}
	}
	sort.SliceStable(indices, func(i, j int) bool {
		left, right := report.Variants[indices[i]], report.Variants[indices[j]]
		if left.CandidateLatency.P95 != right.CandidateLatency.P95 {
			return left.CandidateLatency.P95 < right.CandidateLatency.P95
		}
		if left.AverageCandidates != right.AverageCandidates {
			return left.AverageCandidates < right.AverageCandidates
		}
		if left.CandidateLatency.P99 != right.CandidateLatency.P99 {
			return left.CandidateLatency.P99 < right.CandidateLatency.P99
		}
		if left.Plan.Resolution != right.Plan.Resolution {
			return left.Plan.Resolution < right.Plan.Resolution
		}
		return left.Plan.ExtraPaddingMeters < right.Plan.ExtraPaddingMeters
	})
	for _, index := range indices[:min(count, len(indices))] {
		report.Variants[index].Finalist = true
	}
}

func geoJSONVariantCorrect(report GeoJSONVariantReport) bool {
	if !report.Finalist || !report.Selectable || report.Error != "" {
		return false
	}
	for _, query := range report.Queries {
		if query.FalsePositives > 0 || query.FalseNegatives > 0 {
			return false
		}
	}
	return report.ProductWorkload == nil || report.ProductWorkload.Status == "correct"
}

func selectGeoJSONWinner(report *GeoJSONSweepReport) int {
	indices := make([]int, 0, len(report.Variants))
	for index := range report.Variants {
		if geoJSONVariantCorrect(report.Variants[index]) {
			indices = append(indices, index)
		}
	}
	if len(indices) == 0 {
		return -1
	}
	sort.SliceStable(indices, func(i, j int) bool {
		left, right := report.Variants[indices[i]], report.Variants[indices[j]]
		leftP95, leftP99, leftCandidates := geoJSONVariantScore(left)
		rightP95, rightP99, rightCandidates := geoJSONVariantScore(right)
		if leftP95 != rightP95 {
			return leftP95 < rightP95
		}
		if leftP99 != rightP99 {
			return leftP99 < rightP99
		}
		if leftCandidates != rightCandidates {
			return leftCandidates < rightCandidates
		}
		if left.Plan.Resolution != right.Plan.Resolution {
			return left.Plan.Resolution < right.Plan.Resolution
		}
		return left.Plan.ExtraPaddingMeters < right.Plan.ExtraPaddingMeters
	})
	return indices[0]
}

func geoJSONVariantScore(report GeoJSONVariantReport) (p95, p99, candidates float64) {
	if report.ProductWorkload != nil {
		return report.ProductWorkload.TotalLatency.P95,
			report.ProductWorkload.TotalLatency.P99,
			report.ProductWorkload.AverageCandidates
	}
	for _, query := range report.Queries {
		p95 = max(p95, query.TotalLatency.P95)
		p99 = max(p99, query.TotalLatency.P99)
		candidates += query.AverageCandidates
	}
	if len(report.Queries) > 0 {
		candidates /= float64(len(report.Queries))
	}
	return p95, p99, candidates
}
