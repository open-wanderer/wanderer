package main

import "strings"

const ds923PlusMinimumSamples = 5_000

type performanceRadiusMeasurement struct {
	radiusMeters float64
	latency      LatencyStats
}

type performanceWorkloadMeasurement struct {
	samples           int
	clients           int
	throughputQPS     float64
	correct           bool
	radii             []performanceRadiusMeasurement
	correctnessReason string
}

func evaluatePerformanceTarget(config RunConfig, backend BackendReport) *PerformanceReport {
	report := newPerformanceTargetReport(config)
	if report == nil || report.Reason != "" {
		return report
	}
	workload := backend.ProductWorkload
	if workload == nil {
		report.Reason = "the DS923+ gate requires the product workload"
		return report
	}
	radii := make([]performanceRadiusMeasurement, 0, len(workload.RadiusReports))
	for _, radius := range workload.RadiusReports {
		radii = append(radii, performanceRadiusMeasurement{
			radiusMeters: radius.RadiusMeters,
			latency:      radius.TotalLatency,
		})
	}
	return evaluateDS923PlusPerformance(config, backend, report, performanceWorkloadMeasurement{
		samples:           workload.Samples,
		clients:           workload.Clients,
		throughputQPS:     workload.ThroughputQPS,
		correct:           workload.Status == "correct" && workload.IncorrectCases == 0 && workload.IncorrectWarmSamples == 0,
		radii:             radii,
		correctnessReason: "product correctness mismatch",
	})
}

// evaluateGeoJSONDirectPerformanceTarget applies the same DS923+ scale,
// rebuild, concurrency, throughput, and radius-latency SLOs as the exact
// product path. Only the workload source and correctness contract differ: the
// direct path must pass its UX accuracy audit and remain stable across all
// warm samples.
func evaluateGeoJSONDirectPerformanceTarget(
	config RunConfig,
	backend BackendReport,
	direct *GeoJSONDirectReport,
) *PerformanceReport {
	report := newPerformanceTargetReport(config)
	if report == nil || report.Reason != "" {
		return report
	}
	if direct == nil || direct.ProductWorkload == nil {
		report.Reason = "the DS923+ direct gate requires the GeoJSON direct product workload"
		return report
	}
	workload := direct.ProductWorkload
	radii := make([]performanceRadiusMeasurement, 0, len(workload.RadiusReports))
	for _, radius := range workload.RadiusReports {
		radii = append(radii, performanceRadiusMeasurement{
			radiusMeters: radius.RadiusMeters,
			latency:      radius.TotalLatency,
		})
	}
	return evaluateDS923PlusPerformance(config, backend, report, performanceWorkloadMeasurement{
		samples:       workload.Samples,
		clients:       workload.Clients,
		throughputQPS: workload.ThroughputQPS,
		correct: workload.Status == "passed" && workload.ExpectedSamples > 0 &&
			workload.Samples == workload.ExpectedSamples && workload.UnstableWarmSamples == 0 &&
			workload.Accuracy.Status == "passed",
		radii:             radii,
		correctnessReason: "direct product UX correctness mismatch",
	})
}

// evaluateGeoJSONDirectPlanPerformance applies the target independently to
// every warm-qualified resolution, then refreshes the top-level compatibility
// aliases from the best complete plan. One slow/unstable resolution therefore
// cannot hide another plan that satisfies the NAS contract.
func evaluateGeoJSONDirectPlanPerformance(config RunConfig, backend BackendReport, direct *GeoJSONDirectReport) {
	if direct == nil {
		return
	}
	if len(direct.ProductAudits) == 0 {
		direct.Performance = evaluateGeoJSONDirectPerformanceTarget(config, backend, direct)
		return
	}
	for index := range direct.ProductAudits {
		audit := &direct.ProductAudits[index]
		if audit.WarmWorkload == nil {
			continue
		}
		planView := &GeoJSONDirectReport{ProductWorkload: audit.WarmWorkload}
		audit.Performance = evaluateGeoJSONDirectPerformanceTarget(config, backend, planView)
	}
	requirePerformance := config.PerformanceTarget != "" && config.PerformanceTarget != "none"
	selectGeoJSONDirectWarmPlan(direct, requirePerformance)
}

func newPerformanceTargetReport(config RunConfig) *PerformanceReport {
	if config.PerformanceTarget == "" || config.PerformanceTarget == "none" {
		return nil
	}
	report := &PerformanceReport{
		Target:            config.PerformanceTarget,
		Status:            "not_evaluated",
		MinimumSamples:    ds923PlusMinimumSamples,
		RequiredClients:   4,
		MinimumThroughput: 4,
		RequiredTrails:    50_000,
		RebuildBudgetMS:   30 * 60 * 1000,
		NormalThresholds: LatencyStats{
			P50: 250,
			P95: 750,
			P99: 1_500,
		},
		BroadThresholds: LatencyStats{
			P50: 500,
			P95: 2_000,
			P99: 4_000,
		},
	}
	if config.PerformanceTarget != "ds923plus" {
		report.Reason = "unsupported performance target"
	}
	return report
}

func evaluateDS923PlusPerformance(
	config RunConfig,
	backend BackendReport,
	report *PerformanceReport,
	workload performanceWorkloadMeasurement,
) *PerformanceReport {
	report.MeasuredSamples = workload.samples
	report.MeasuredClients = workload.clients
	report.MeasuredThroughput = workload.throughputQPS
	report.MeasuredTrails = config.TrailCount
	report.ScalePass = config.TrailCount >= report.RequiredTrails
	// A deployable shadow rebuild starts with creating and configuring its
	// index, so include setup instead of timing ingestion against a prebuilt
	// empty index only.
	report.FullRebuildMS = backend.IndexSetupMS + phaseTotalMS(backend.FullIndex)
	report.RebuildPass = report.FullRebuildMS > 0 && report.FullRebuildMS <= report.RebuildBudgetMS
	report.CorrectnessPass = workload.correct
	report.ConcurrencyPass = workload.clients >= report.RequiredClients
	report.ThroughputPass = workload.throughputQPS >= report.MinimumThroughput

	normalFound := false
	broadFound := false
	for _, radius := range workload.radii {
		if radius.radiusMeters <= 25_000 {
			normalFound = true
			report.NormalWorst.P50 = max(report.NormalWorst.P50, radius.latency.P50)
			report.NormalWorst.P95 = max(report.NormalWorst.P95, radius.latency.P95)
			report.NormalWorst.P99 = max(report.NormalWorst.P99, radius.latency.P99)
		} else {
			broadFound = true
			report.BroadWorst.P50 = max(report.BroadWorst.P50, radius.latency.P50)
			report.BroadWorst.P95 = max(report.BroadWorst.P95, radius.latency.P95)
			report.BroadWorst.P99 = max(report.BroadWorst.P99, radius.latency.P99)
		}
	}
	report.NormalLatencyPass = normalFound && withinLatencyThreshold(report.NormalWorst, report.NormalThresholds)
	report.BroadLatencyPass = broadFound && withinLatencyThreshold(report.BroadWorst, report.BroadThresholds)

	reasons := make([]string, 0, 5)
	if !normalFound {
		reasons = append(reasons, "no normal-radius samples (<=25 km)")
	}
	if !broadFound {
		reasons = append(reasons, "no broad-radius samples (>25 km)")
	}
	if normalFound && !report.NormalLatencyPass {
		reasons = append(reasons, "normal-radius latency budget exceeded")
	}
	if broadFound && !report.BroadLatencyPass {
		reasons = append(reasons, "broad-radius latency budget exceeded")
	}
	if !report.ConcurrencyPass {
		reasons = append(reasons, "fewer than four product clients")
	}
	if report.MeasuredSamples < report.MinimumSamples {
		reasons = append(reasons, "fewer than 5,000 warm samples")
	}
	if !report.ThroughputPass {
		reasons = append(reasons, "throughput below four queries per second")
	}
	if !report.ScalePass {
		reasons = append(reasons, "fewer than 50,000 trails")
	}
	if !report.RebuildPass {
		reasons = append(reasons, "full rebuild exceeds 30 minutes or was not measured")
	}
	if !report.CorrectnessPass {
		reasons = append(reasons, workload.correctnessReason)
	}
	report.Reason = strings.Join(reasons, "; ")

	functionalPass := report.CorrectnessPass && report.ConcurrencyPass && report.ThroughputPass && report.RebuildPass && report.NormalLatencyPass && report.BroadLatencyPass
	if functionalPass && report.ScalePass && report.MeasuredSamples >= report.MinimumSamples {
		report.Status = "passed"
	} else if !report.CorrectnessPass || !report.ThroughputPass || !report.RebuildPass || !report.NormalLatencyPass || !report.BroadLatencyPass {
		report.Status = "failed"
	} else {
		report.Status = "inconclusive"
	}
	return report
}

func withinLatencyThreshold(measured, threshold LatencyStats) bool {
	return measured.P50 <= threshold.P50 && measured.P95 <= threshold.P95 && measured.P99 <= threshold.P99
}
