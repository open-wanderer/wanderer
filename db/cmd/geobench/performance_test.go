package main

import (
	"strings"
	"testing"
)

func TestEvaluateDS923PlusPerformanceTarget(t *testing.T) {
	config := RunConfig{PerformanceTarget: "ds923plus", TrailCount: 50_000}
	backend := BackendReport{IndexSetupMS: 100, FullIndex: PhaseReport{ReadyMS: 1_000}, ProductWorkload: &ProductWorkloadReport{
		Status:        "correct",
		Clients:       4,
		Samples:       5_040,
		ThroughputQPS: 10,
		RadiusReports: []ProductRadiusReport{
			{RadiusMeters: 500, TotalLatency: LatencyStats{P50: 100, P95: 600, P99: 1_200}},
			{RadiusMeters: 25_000, TotalLatency: LatencyStats{P50: 200, P95: 700, P99: 1_400}},
			{RadiusMeters: 100_000, TotalLatency: LatencyStats{P50: 400, P95: 1_800, P99: 3_500}},
		},
	}}
	report := evaluatePerformanceTarget(config, backend)
	if report == nil || report.Status != "passed" {
		t.Fatalf("performance report = %+v, want passed", report)
	}
	if report.FullRebuildMS != 1_100 {
		t.Fatalf("full rebuild = %v ms, want setup plus ingestion = 1100 ms", report.FullRebuildMS)
	}

	backend.ProductWorkload.Samples = 100
	report = evaluatePerformanceTarget(config, backend)
	if report.Status != "inconclusive" {
		t.Fatalf("small-sample status = %q, want inconclusive", report.Status)
	}

	backend.ProductWorkload.Samples = 5_040
	backend.ProductWorkload.RadiusReports[0].TotalLatency.P95 = 751
	report = evaluatePerformanceTarget(config, backend)
	if report.Status != "failed" || report.NormalLatencyPass {
		t.Fatalf("slow status = %+v, want failed normal latency", report)
	}
}

func TestEvaluatePerformanceTargetRequiresProductWorkload(t *testing.T) {
	if report := evaluatePerformanceTarget(RunConfig{}, BackendReport{}); report != nil {
		t.Fatalf("disabled target produced %+v", report)
	}
	report := evaluatePerformanceTarget(RunConfig{PerformanceTarget: "ds923plus"}, BackendReport{})
	if report == nil || report.Status != "not_evaluated" {
		t.Fatalf("missing workload report = %+v", report)
	}
}

func TestEvaluateGeoJSONDirectDS923PlusPerformanceTarget(t *testing.T) {
	config := RunConfig{PerformanceTarget: "ds923plus", TrailCount: 50_000}
	backend := BackendReport{IndexSetupMS: 100, FullIndex: PhaseReport{ReadyMS: 1_000}}
	direct := passingGeoJSONDirectPerformanceFixture()

	report := evaluateGeoJSONDirectPerformanceTarget(config, backend, direct)
	if report == nil || report.Status != "passed" {
		t.Fatalf("direct performance report = %+v, want passed", report)
	}
	if report.MeasuredSamples != 5_040 || report.MeasuredClients != 4 || report.MeasuredThroughput != 10 {
		t.Fatalf("direct measurements = samples %d clients %d qps %v", report.MeasuredSamples, report.MeasuredClients, report.MeasuredThroughput)
	}
	if report.FullRebuildMS != 1_100 || !report.RebuildPass || !report.ScalePass {
		t.Fatalf("direct scale/rebuild = %+v", report)
	}
	if report.NormalWorst != (LatencyStats{P50: 200, P95: 700, P99: 1_400}) {
		t.Fatalf("normal worst = %+v", report.NormalWorst)
	}
	if report.BroadWorst != (LatencyStats{P50: 400, P95: 1_800, P99: 3_500}) {
		t.Fatalf("broad worst = %+v", report.BroadWorst)
	}

	direct.ProductWorkload.Samples = 100
	direct.ProductWorkload.ExpectedSamples = 100
	report = evaluateGeoJSONDirectPerformanceTarget(config, backend, direct)
	if report.Status != "inconclusive" || !strings.Contains(report.Reason, "fewer than 5,000 warm samples") {
		t.Fatalf("small direct sample report = %+v, want inconclusive", report)
	}

	direct = passingGeoJSONDirectPerformanceFixture()
	direct.ProductWorkload.RadiusReports[0].TotalLatency.P95 = 751
	report = evaluateGeoJSONDirectPerformanceTarget(config, backend, direct)
	if report.Status != "failed" || report.NormalLatencyPass {
		t.Fatalf("slow direct status = %+v, want failed normal latency", report)
	}
}

func TestEvaluateGeoJSONDirectPerformanceRequiresUXCorrectness(t *testing.T) {
	config := RunConfig{PerformanceTarget: "ds923plus", TrailCount: 50_000}
	backend := BackendReport{FullIndex: PhaseReport{ReadyMS: 1_000}}

	tests := []struct {
		name   string
		mutate func(*GeoJSONDirectProductWorkloadReport)
	}{
		{
			name: "workload failed",
			mutate: func(workload *GeoJSONDirectProductWorkloadReport) {
				workload.Status = "failed"
			},
		},
		{
			name: "accuracy failed",
			mutate: func(workload *GeoJSONDirectProductWorkloadReport) {
				workload.Accuracy.Status = "failed"
			},
		},
		{
			name: "unstable warm result",
			mutate: func(workload *GeoJSONDirectProductWorkloadReport) {
				workload.UnstableWarmSamples = 1
			},
		},
		{
			name: "incomplete warm workload",
			mutate: func(workload *GeoJSONDirectProductWorkloadReport) {
				workload.ExpectedSamples++
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			direct := passingGeoJSONDirectPerformanceFixture()
			test.mutate(direct.ProductWorkload)
			report := evaluateGeoJSONDirectPerformanceTarget(config, backend, direct)
			if report.Status != "failed" || report.CorrectnessPass {
				t.Fatalf("direct correctness report = %+v, want failed", report)
			}
			if !strings.Contains(report.Reason, "direct product UX correctness mismatch") {
				t.Fatalf("direct correctness reason = %q", report.Reason)
			}
		})
	}
}

func TestEvaluateGeoJSONDirectPerformanceTargetAvailability(t *testing.T) {
	if report := evaluateGeoJSONDirectPerformanceTarget(RunConfig{}, BackendReport{}, nil); report != nil {
		t.Fatalf("disabled direct target produced %+v", report)
	}

	report := evaluateGeoJSONDirectPerformanceTarget(
		RunConfig{PerformanceTarget: "ds923plus"},
		BackendReport{},
		nil,
	)
	if report == nil || report.Status != "not_evaluated" || !strings.Contains(report.Reason, "direct product workload") {
		t.Fatalf("missing direct report = %+v", report)
	}

	report = evaluateGeoJSONDirectPerformanceTarget(
		RunConfig{PerformanceTarget: "another-nas"},
		BackendReport{},
		passingGeoJSONDirectPerformanceFixture(),
	)
	if report == nil || report.Status != "not_evaluated" || report.Reason != "unsupported performance target" {
		t.Fatalf("unsupported direct target = %+v", report)
	}
}

func TestEvaluateGeoJSONDirectPlanPerformanceSelectsPassingAlternative(t *testing.T) {
	slow := *passingGeoJSONDirectPerformanceFixture().ProductWorkload
	slow.RadiusReports = append([]GeoJSONDirectRadiusReport(nil), slow.RadiusReports...)
	slow.RadiusReports[0].TotalLatency.P95 = 900
	fast := *passingGeoJSONDirectPerformanceFixture().ProductWorkload
	direct := &GeoJSONDirectReport{ProductAudits: []GeoJSONDirectProductAuditReport{
		{
			PlanID: "r100-exact", Resolution: 100, Status: "audited",
			ProductPageLatency: LatencyStats{P95: 1}, WarmWorkload: &slow,
		},
		{
			PlanID: "r125-exact", Resolution: 125, Status: "audited",
			ProductPageLatency: LatencyStats{P95: 2}, WarmWorkload: &fast,
		},
	}}
	evaluateGeoJSONDirectPlanPerformance(
		RunConfig{PerformanceTarget: "ds923plus", TrailCount: 50_000},
		BackendReport{FullIndex: PhaseReport{ReadyMS: 1_000}},
		direct,
	)
	if direct.ProductAudits[0].Performance == nil || direct.ProductAudits[0].Performance.Status != "failed" ||
		direct.ProductAudits[1].Performance == nil || direct.ProductAudits[1].Performance.Status != "passed" {
		t.Fatalf("per-plan performance = %+v", direct.ProductAudits)
	}
	if direct.SelectedID != "r125-exact" || direct.Performance == nil || direct.Performance.Status != "passed" {
		t.Fatalf("selected direct plan = %+v, want passing r125", direct)
	}
}

func passingGeoJSONDirectPerformanceFixture() *GeoJSONDirectReport {
	return &GeoJSONDirectReport{ProductWorkload: &GeoJSONDirectProductWorkloadReport{
		Status:          "passed",
		Clients:         4,
		ExpectedSamples: 5_040,
		Samples:         5_040,
		ThroughputQPS:   10,
		Accuracy:        GeoJSONDirectAccuracy{Status: "passed"},
		RadiusReports: []GeoJSONDirectRadiusReport{
			{RadiusMeters: 500, TotalLatency: LatencyStats{P50: 100, P95: 600, P99: 1_200}},
			{RadiusMeters: 25_000, TotalLatency: LatencyStats{P50: 200, P95: 700, P99: 1_400}},
			{RadiusMeters: 100_000, TotalLatency: LatencyStats{P50: 400, P95: 1_800, P99: 3_500}},
		},
	}}
}
