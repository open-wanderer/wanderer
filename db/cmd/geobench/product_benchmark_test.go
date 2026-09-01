package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBenchmarkProductWorkloadVerifiesFullResultAndRunsConcurrently(t *testing.T) {
	dataset := productBenchmarkTestDataset()
	documents := generateProductDocuments(dataset)
	store, err := newProductStore(filepath.Join(t.TempDir(), "product.sqlite"), 2)
	if err != nil {
		t.Fatalf("new product store: %v", err)
	}
	defer store.close()
	if err := store.replaceAll(context.Background(), documents); err != nil {
		t.Fatalf("replace product store: %v", err)
	}
	query := func(context.Context, QueryPoint, float64) (CandidateResult, error) {
		return CandidateResult{IDs: idSet("a", "b", "c"), WallDuration: time.Millisecond}, nil
	}

	report, err := benchmarkProductWorkload(
		context.Background(), dataset, []float64{5_000}, 3, 2, store, documents, query,
	)
	if err != nil {
		t.Fatalf("benchmark product workload: %v", err)
	}
	if report.Status != "correct" || report.CorrectCases != 1 || report.IncorrectCases != 0 || report.IncorrectWarmSamples != 0 {
		t.Fatalf("correctness report = %+v", report)
	}
	if report.Samples != 3 || report.Clients != 2 || report.AverageLoadedRows != 3 {
		t.Fatalf("workload report = %+v", report)
	}
	if report.ThroughputQPS <= 0 || report.DatabaseLatency.Max <= 0 || report.FinalizeLatency.Max <= 0 {
		t.Fatalf("missing timed stages: %+v", report)
	}
	benchmarkCase := buildProductBenchmarkCases(dataset, []float64{5_000})[0]
	expected, err := RunProductQueryExhaustive(context.Background(), documents, benchmarkCase.Query)
	if err != nil {
		t.Fatal(err)
	}
	encodedPage, err := marshalProductPagePayload(expected, documents)
	if err != nil {
		t.Fatal(err)
	}
	if report.AverageEncodedBytes <= 0 || report.AverageResponseBytes != float64(len(encodedPage)) {
		t.Fatalf("database/response bytes = %.1f/%.1f, want response %d", report.AverageEncodedBytes, report.AverageResponseBytes, len(encodedPage))
	}
	if len(report.CaseReports) != 1 || report.CaseReports[0].ResponseBytes != int64(len(encodedPage)) {
		t.Fatalf("case response bytes = %+v, want %d", report.CaseReports, len(encodedPage))
	}
}

func TestBenchmarkProductWorkloadExposesCandidateFalseNegative(t *testing.T) {
	dataset := productBenchmarkTestDataset()
	documents := generateProductDocuments(dataset)
	store, err := newProductStore(filepath.Join(t.TempDir(), "product.sqlite"), 1)
	if err != nil {
		t.Fatalf("new product store: %v", err)
	}
	defer store.close()
	if err := store.replaceAll(context.Background(), documents); err != nil {
		t.Fatalf("replace product store: %v", err)
	}
	query := func(context.Context, QueryPoint, float64) (CandidateResult, error) {
		return CandidateResult{IDs: idSet("a")}, nil
	}

	report, err := benchmarkProductWorkload(
		context.Background(), dataset, []float64{5_000}, 1, 1, store, documents, query,
	)
	if err != nil {
		t.Fatalf("benchmark product workload: %v", err)
	}
	if report.Status != "incorrect" || report.IncorrectCases == 0 || report.IncorrectWarmSamples == 0 {
		t.Fatalf("incorrect candidate set was accepted: %+v", report)
	}
	if len(report.CaseReports) != 1 || report.CaseReports[0].MissingExpectedMatches == 0 {
		t.Fatalf("missing candidate not reported: %+v", report.CaseReports)
	}
}

func TestBenchmarkProductWorkloadRejectsTruncatedCandidates(t *testing.T) {
	dataset := productBenchmarkTestDataset()
	documents := generateProductDocuments(dataset)
	store, err := newProductStore(filepath.Join(t.TempDir(), "product.sqlite"), 1)
	if err != nil {
		t.Fatal(err)
	}
	defer store.close()
	if err := store.replaceAll(context.Background(), documents); err != nil {
		t.Fatal(err)
	}
	query := func(context.Context, QueryPoint, float64) (CandidateResult, error) {
		return CandidateResult{
			IDs:                idSet("a", "b", "c"),
			EstimatedTotalHits: 4,
			Truncated:          true,
		}, nil
	}
	if _, err := benchmarkProductWorkload(
		context.Background(), dataset, []float64{5_000}, 1, 1, store, documents, query,
	); err == nil || !strings.Contains(err.Error(), "truncated") {
		t.Fatalf("truncated candidate error = %v", err)
	}
}

func TestExecuteProductSampleExcludesOracleBookkeepingFromTotal(t *testing.T) {
	dataset := productBenchmarkTestDataset()
	documents := generateProductDocuments(dataset)
	store, err := newProductStore(filepath.Join(t.TempDir(), "product.sqlite"), 1)
	if err != nil {
		t.Fatal(err)
	}
	defer store.close()
	if err := store.replaceAll(context.Background(), documents); err != nil {
		t.Fatal(err)
	}
	benchmarkCase := buildProductBenchmarkCases(dataset, []float64{5_000})[0]
	expected, err := RunProductQueryExhaustive(context.Background(), documents, benchmarkCase.Query)
	if err != nil {
		t.Fatal(err)
	}
	prepared := preparedProductCase{
		benchmark: benchmarkCase,
		expected:  expected,
		allIDs:    idSet("a", "b", "c"),
		beforeCorrectnessCheck: func() {
			time.Sleep(75 * time.Millisecond)
		},
	}
	query := func(context.Context, QueryPoint, float64, ProductQuery) (CandidateResult, error) {
		return CandidateResult{IDs: idSet("a", "b", "c")}, nil
	}
	wallStarted := time.Now()
	sample := executeProductSample(context.Background(), 0, prepared, store, query)
	verifyProductSample(prepared, &sample, true)
	wallDuration := time.Since(wallStarted)
	if sample.err != nil {
		t.Fatal(sample.err)
	}
	if bookkeeping := wallDuration - sample.total; bookkeeping < 70*time.Millisecond {
		t.Fatalf("wall-total = %v, want the 75ms oracle hook outside total", bookkeeping)
	}
}

func TestConcurrentProductSamplesExcludeOracleBookkeepingFromThroughputClock(t *testing.T) {
	dataset := productBenchmarkTestDataset()
	documents := generateProductDocuments(dataset)
	store, err := newProductStore(filepath.Join(t.TempDir(), "product.sqlite"), 2)
	if err != nil {
		t.Fatal(err)
	}
	defer store.close()
	if err := store.replaceAll(context.Background(), documents); err != nil {
		t.Fatal(err)
	}
	benchmarkCase := buildProductBenchmarkCases(dataset, []float64{5_000})[0]
	expected, err := RunProductQueryExhaustive(context.Background(), documents, benchmarkCase.Query)
	if err != nil {
		t.Fatal(err)
	}
	prepared := []preparedProductCase{{
		benchmark: benchmarkCase,
		expected:  expected,
		allIDs:    idSet("a", "b", "c"),
		beforeCorrectnessCheck: func() {
			time.Sleep(50 * time.Millisecond)
		},
	}}
	query := func(context.Context, QueryPoint, float64, ProductQuery) (CandidateResult, error) {
		return CandidateResult{IDs: idSet("a", "b", "c")}, nil
	}
	started := time.Now()
	lastCompletedAt := started
	samples := 0
	for sample := range executeConcurrentProductSamples(context.Background(), prepared, 3, 2, store, query) {
		if sample.err != nil {
			t.Fatal(sample.err)
		}
		if sample.completedAt.After(lastCompletedAt) {
			lastCompletedAt = sample.completedAt
		}
		verifyProductSample(prepared[sample.caseIndex], &sample, false)
		if !sample.correct {
			t.Fatalf("sample correctness = %q", sample.errorMessage)
		}
		samples++
	}
	if samples != 3 {
		t.Fatalf("samples = %d, want 3", samples)
	}
	productionWall := lastCompletedAt.Sub(started)
	collectorWall := time.Since(started)
	if bookkeepingTail := collectorWall - productionWall; bookkeepingTail < 140*time.Millisecond {
		t.Fatalf("collector-production wall = %v, want three 50ms oracle hooks outside the throughput clock", bookkeepingTail)
	}
}

func productBenchmarkTestDataset() Dataset {
	return Dataset{
		Trails: []Trail{
			{ID: "a", Scenario: scenarioAlpineDense, Parts: [][]Coordinate{{{Lat: 47, Lon: 8}, {Lat: 47, Lon: 8.01}}}},
			{ID: "b", Scenario: scenarioAlpineDense, Parts: [][]Coordinate{{{Lat: 47.001, Lon: 8}, {Lat: 47.001, Lon: 8.01}}}},
			{ID: "c", Scenario: scenarioAlpineDense, Parts: [][]Coordinate{{{Lat: 47.002, Lon: 8}, {Lat: 47.002, Lon: 8.01}}}},
		},
		QueryPoints: []QueryPoint{{ID: "q", Scenario: scenarioAlpineDense, Point: Coordinate{Lat: 47, Lon: 8.005}}},
	}
}
