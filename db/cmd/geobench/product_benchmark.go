package main

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

type productBenchmarkCase struct {
	Name       string
	QueryPoint QueryPoint
	Radius     float64
	Query      ProductQuery
}

// productCandidateQuery may push predicates that do not require exact route
// geometry into the candidate backend. It must never paginate: the SQLite
// finalizer still needs every possible match for exact counts and stable pages.
type productCandidateQuery func(context.Context, QueryPoint, float64, ProductQuery) (CandidateResult, error)

type preparedProductCase struct {
	benchmark productBenchmarkCase
	expected  ProductQueryResult
	allIDs    map[string]struct{}
	// beforeCorrectnessCheck is a focused test seam proving that benchmark
	// oracle bookkeeping remains outside the measured product response path.
	beforeCorrectnessCheck func()
}

type productSample struct {
	caseIndex       int
	candidate       CandidateResult
	candidateCount  int
	loaded          productDocumentLoad
	actual          ProductQueryResult
	database        time.Duration
	finalize        time.Duration
	total           time.Duration
	completedAt     time.Time
	responseBytes   int64
	missingExpected int
	correct         bool
	errorMessage    string
	err             error
}

func benchmarkProductWorkload(
	ctx context.Context,
	dataset Dataset,
	radii []float64,
	repetitions int,
	clients int,
	store *productStore,
	documents []ProductQueryDocument,
	query candidateQuery,
) (*ProductWorkloadReport, error) {
	if query == nil {
		return nil, fmt.Errorf("product workload requires a candidate query")
	}
	return benchmarkProductWorkloadWithCandidateQuery(
		ctx, dataset, radii, repetitions, clients, store, documents,
		func(ctx context.Context, point QueryPoint, radius float64, _ ProductQuery) (CandidateResult, error) {
			return query(ctx, point, radius)
		},
	)
}

func benchmarkProductWorkloadWithCandidateQuery(
	ctx context.Context,
	dataset Dataset,
	radii []float64,
	repetitions int,
	clients int,
	store *productStore,
	documents []ProductQueryDocument,
	query productCandidateQuery,
) (*ProductWorkloadReport, error) {
	if store == nil {
		return nil, fmt.Errorf("product workload requires an application database")
	}
	if query == nil {
		return nil, fmt.Errorf("product workload requires a candidate query")
	}
	if repetitions < 1 {
		repetitions = 1
	}
	if clients < 1 {
		clients = 1
	}

	cases := buildProductBenchmarkCases(dataset, radii)
	prepared := make([]preparedProductCase, len(cases))
	oracleStarted := time.Now()
	for index, benchmarkCase := range cases {
		expected, err := RunProductQueryExhaustive(ctx, documents, benchmarkCase.Query)
		if err != nil {
			return nil, fmt.Errorf("prepare product oracle %q: %w", benchmarkCase.Name, err)
		}
		allQuery := benchmarkCase.Query
		allQuery.Page = 1
		allQuery.PerPage = max(1, len(documents)+1)
		allExpected, err := RunProductQueryExhaustive(ctx, documents, allQuery)
		if err != nil {
			return nil, fmt.Errorf("prepare complete product oracle %q: %w", benchmarkCase.Name, err)
		}
		allIDs := make(map[string]struct{}, len(allExpected.Hits))
		for _, hit := range allExpected.Hits {
			allIDs[hit.ID] = struct{}{}
		}
		prepared[index] = preparedProductCase{benchmark: benchmarkCase, expected: expected, allIDs: allIDs}
	}

	report := &ProductWorkloadReport{
		Status:      "correct",
		OracleMS:    milliseconds(time.Since(oracleStarted)),
		Clients:     min(clients, max(1, len(cases)*repetitions)),
		Cases:       len(cases),
		CaseReports: make([]ProductCaseReport, 0, len(cases)),
	}
	firstLatencies := make([]float64, 0, len(cases))
	for index := range prepared {
		sample := executeProductSample(ctx, index, prepared[index], store, query)
		if sample.err != nil {
			return report, fmt.Errorf("product query %q: %w", prepared[index].benchmark.Name, sample.err)
		}
		verifyProductSample(prepared[index], &sample, true)
		firstLatencies = append(firstLatencies, milliseconds(sample.total))
		caseReport := productCaseReport(prepared[index], sample)
		report.CaseReports = append(report.CaseReports, caseReport)
		if caseReport.Correct {
			report.CorrectCases++
		} else {
			report.IncorrectCases++
			report.Status = "incorrect"
		}
	}
	report.FirstSampleLatency = latencyStats(firstLatencies)

	concurrentStarted := time.Now()
	totalJobs := len(prepared) * repetitions
	results := executeConcurrentProductSamples(ctx, prepared, repetitions, report.Clients, store, query)

	candidateLatencies := make([]float64, 0, totalJobs)
	databaseLatencies := make([]float64, 0, totalJobs)
	finalizeLatencies := make([]float64, 0, totalJobs)
	totalLatencies := make([]float64, 0, totalJobs)
	var candidateTotal, loadedTotal, exactTotal int64
	var encodedTotal, responseTotal int64
	type radiusAccumulator struct {
		latencies  []float64
		candidates int64
		incorrect  int
	}
	radiusGroups := make(map[float64]*radiusAccumulator)
	var firstSampleError error
	lastCompletedAt := concurrentStarted
	for sample := range results {
		if sample.err != nil {
			if firstSampleError == nil {
				firstSampleError = fmt.Errorf("product query %q: %w", prepared[sample.caseIndex].benchmark.Name, sample.err)
			}
			continue
		}
		if sample.completedAt.After(lastCompletedAt) {
			lastCompletedAt = sample.completedAt
		}
		verifyProductSample(prepared[sample.caseIndex], &sample, false)
		candidateLatencies = append(candidateLatencies, milliseconds(sample.candidate.WallDuration))
		databaseLatencies = append(databaseLatencies, milliseconds(sample.database))
		finalizeLatencies = append(finalizeLatencies, milliseconds(sample.finalize))
		totalLatencies = append(totalLatencies, milliseconds(sample.total))
		candidateTotal += int64(sample.candidateCount)
		loadedTotal += int64(sample.loaded.Rows)
		encodedTotal += sample.loaded.EncodedBytes
		responseTotal += sample.responseBytes
		exactTotal += int64(sample.actual.ExactChecks)
		report.Samples++
		radius := prepared[sample.caseIndex].benchmark.Radius
		group := radiusGroups[radius]
		if group == nil {
			group = &radiusAccumulator{}
			radiusGroups[radius] = group
		}
		group.latencies = append(group.latencies, milliseconds(sample.total))
		group.candidates += int64(sample.candidateCount)
		if !sample.correct {
			report.IncorrectWarmSamples++
			report.Status = "incorrect"
			group.incorrect++
		}
	}
	report.ConcurrentWallMS = milliseconds(lastCompletedAt.Sub(concurrentStarted))
	if firstSampleError != nil {
		return report, firstSampleError
	}
	report.CandidateLatency = latencyStats(candidateLatencies)
	report.DatabaseLatency = latencyStats(databaseLatencies)
	report.FinalizeLatency = latencyStats(finalizeLatencies)
	report.TotalLatency = latencyStats(totalLatencies)
	if report.Samples > 0 {
		report.AverageCandidates = float64(candidateTotal) / float64(report.Samples)
		report.AverageLoadedRows = float64(loadedTotal) / float64(report.Samples)
		report.AverageEncodedBytes = float64(encodedTotal) / float64(report.Samples)
		report.AverageResponseBytes = float64(responseTotal) / float64(report.Samples)
		report.AverageExactChecks = float64(exactTotal) / float64(report.Samples)
		if report.ConcurrentWallMS > 0 {
			report.ThroughputQPS = float64(report.Samples) * 1000 / report.ConcurrentWallMS
		}
	}
	radiiPresent := make([]float64, 0, len(radiusGroups))
	for radius := range radiusGroups {
		radiiPresent = append(radiiPresent, radius)
	}
	sort.Float64s(radiiPresent)
	for _, radius := range radiiPresent {
		group := radiusGroups[radius]
		radiusReport := ProductRadiusReport{
			RadiusMeters:     radius,
			Samples:          len(group.latencies),
			IncorrectSamples: group.incorrect,
			TotalLatency:     latencyStats(group.latencies),
		}
		if len(group.latencies) > 0 {
			radiusReport.AverageCandidates = float64(group.candidates) / float64(len(group.latencies))
		}
		report.RadiusReports = append(report.RadiusReports, radiusReport)
	}
	return report, nil
}

func executeProductSample(
	ctx context.Context,
	caseIndex int,
	prepared preparedProductCase,
	store *productStore,
	query productCandidateQuery,
) productSample {
	started := time.Now()
	candidate, err := query(ctx, prepared.benchmark.QueryPoint, prepared.benchmark.Radius, prepared.benchmark.Query)
	if err != nil {
		return productSample{caseIndex: caseIndex, err: err}
	}
	if candidate.Truncated {
		return productSample{
			caseIndex: caseIndex,
			candidate: candidate,
			err:       fmt.Errorf("candidate result was truncated at %d IDs (estimated total %d)", len(candidate.IDs), candidate.EstimatedTotalHits),
		}
	}
	databaseStarted := time.Now()
	loaded, err := store.load(ctx, candidate.IDs)
	databaseDuration := time.Since(databaseStarted)
	if err != nil {
		return productSample{caseIndex: caseIndex, candidate: candidate, database: databaseDuration, err: err}
	}
	if loaded.Rows != len(candidate.IDs) {
		return productSample{
			caseIndex: caseIndex,
			candidate: candidate,
			loaded:    loaded,
			database:  databaseDuration,
			err:       fmt.Errorf("candidate index returned %d IDs but application database loaded %d records", len(candidate.IDs), loaded.Rows),
		}
	}
	finalizeStarted := time.Now()
	actual, err := RunProductQueryCandidateRefine(ctx, loaded.Documents, prepared.benchmark.Query)
	if err != nil {
		finalizeDuration := time.Since(finalizeStarted)
		return productSample{caseIndex: caseIndex, candidate: candidate, loaded: loaded, database: databaseDuration, finalize: finalizeDuration, err: err}
	}
	response, err := marshalProductPagePayload(actual, loaded.Documents)
	finalizeDuration := time.Since(finalizeStarted)
	if err != nil {
		return productSample{caseIndex: caseIndex, candidate: candidate, loaded: loaded, actual: actual, database: databaseDuration, finalize: finalizeDuration, err: err}
	}
	// The product-visible response is complete here. Candidate/oracle
	// comparison below is benchmark bookkeeping and must not penalize the
	// hybrid path relative to direct GeoJSON, whose stability check is also
	// outside WallDuration.
	totalDuration := time.Since(started)
	return productSample{
		caseIndex:      caseIndex,
		candidate:      candidate,
		candidateCount: len(candidate.IDs),
		loaded:         loaded,
		actual:         actual,
		database:       databaseDuration,
		finalize:       finalizeDuration,
		total:          totalDuration,
		completedAt:    time.Now(),
		responseBytes:  int64(len(response)),
	}
}

// verifyProductSample is benchmark-only correctness work. It deliberately
// runs after the product response completion timestamp and outside the query
// workers so neither latency nor concurrent throughput includes oracle scans.
// Complete candidate diagnostics are needed once per case; warm samples only
// need the exact count/order/page comparison.
func verifyProductSample(prepared preparedProductCase, sample *productSample, includeCandidateDiagnostics bool) {
	if sample == nil || sample.err != nil {
		return
	}
	if prepared.beforeCorrectnessCheck != nil {
		prepared.beforeCorrectnessCheck()
	}
	if includeCandidateDiagnostics {
		sample.missingExpected = differenceSize(prepared.allIDs, sample.candidate.IDs, nil)
	}
	sample.correct, sample.errorMessage = equalProductQueryResults(prepared.expected, sample.actual)
}

// executeConcurrentProductSamples keeps the measured query workers independent
// of oracle verification. Warm samples are compacted before entering a buffer
// sized for the finite workload, so a slower collector cannot stall production
// completion while the buffer remains bounded by the configured sample count.
func executeConcurrentProductSamples(
	ctx context.Context,
	prepared []preparedProductCase,
	repetitions, clients int,
	store *productStore,
	query productCandidateQuery,
) <-chan productSample {
	totalJobs := len(prepared) * repetitions
	jobs := make(chan int)
	results := make(chan productSample, totalJobs)
	var workers sync.WaitGroup
	workers.Add(clients)
	for range clients {
		go func() {
			defer workers.Done()
			for caseIndex := range jobs {
				sample := executeProductSample(ctx, caseIndex, prepared[caseIndex], store, query)
				// The warm collector only needs scalar load/candidate metrics and
				// the final page. Release potentially large candidate/document
				// collections before buffering the result.
				sample.candidate.IDs = nil
				sample.loaded.Documents = nil
				results <- sample
			}
		}()
	}
	go func() {
		defer close(jobs)
		for range repetitions {
			for caseIndex := range prepared {
				select {
				case <-ctx.Done():
					return
				case jobs <- caseIndex:
				}
			}
		}
	}()
	go func() {
		workers.Wait()
		close(results)
	}()
	return results
}

func productCaseReport(prepared preparedProductCase, sample productSample) ProductCaseReport {
	benchmarkCase := prepared.benchmark
	return ProductCaseReport{
		Name:                   benchmarkCase.Name,
		QueryID:                benchmarkCase.QueryPoint.ID,
		Scenario:               benchmarkCase.QueryPoint.Scenario,
		RadiusMeters:           benchmarkCase.Radius,
		ActorID:                benchmarkCase.Query.ActorID,
		Text:                   benchmarkCase.Query.Text,
		Sort:                   string(benchmarkCase.Query.Sort),
		Page:                   sample.actual.Page,
		PerPage:                sample.actual.PerPage,
		ExpectedTotalItems:     prepared.expected.TotalItems,
		ActualTotalItems:       sample.actual.TotalItems,
		ExpectedHitIDs:         productHitIDs(prepared.expected),
		ActualHitIDs:           productHitIDs(sample.actual),
		CandidateCount:         sample.candidateCount,
		LoadedRows:             sample.loaded.Rows,
		EncodedBytes:           sample.loaded.EncodedBytes,
		ResponseBytes:          sample.responseBytes,
		ExactChecks:            sample.actual.ExactChecks,
		MissingExpectedMatches: sample.missingExpected,
		Correct:                sample.correct,
		Error:                  sample.errorMessage,
	}
}

func equalProductQueryResults(expected, actual ProductQueryResult) (bool, string) {
	if expected.TotalItems != actual.TotalItems {
		return false, fmt.Sprintf("exact count: got %d, want %d", actual.TotalItems, expected.TotalItems)
	}
	if expected.TotalPages != actual.TotalPages || expected.Page != actual.Page || expected.PerPage != actual.PerPage {
		return false, fmt.Sprintf(
			"pagination metadata: got pages/page/per-page %d/%d/%d, want %d/%d/%d",
			actual.TotalPages, actual.Page, actual.PerPage,
			expected.TotalPages, expected.Page, expected.PerPage,
		)
	}
	if expectedIDs, actualIDs := productHitIDs(expected), productHitIDs(actual); !reflect.DeepEqual(expectedIDs, actualIDs) {
		return false, fmt.Sprintf("ordered page IDs: got %v, want %v", actualIDs, expectedIDs)
	}
	return true, ""
}

func productHitIDs(result ProductQueryResult) []string {
	ids := make([]string, len(result.Hits))
	for index, hit := range result.Hits {
		ids[index] = hit.ID
	}
	return ids
}

func buildProductBenchmarkCases(dataset Dataset, radii []float64) []productBenchmarkCase {
	if len(radii) == 0 {
		return nil
	}
	cases := make([]productBenchmarkCase, 0, len(dataset.QueryPoints))
	for index, queryPoint := range dataset.QueryPoints {
		variant := index % 6
		radius := productRadiusForVariant(radii, variant, index)
		query := ProductQuery{
			Geo:     &ProductGeoRadius{Center: queryPoint.Point, RadiusMeters: radius},
			Sort:    ProductSortRelevance,
			Page:    1,
			PerPage: 10,
		}
		name := "anonymous text and ACL"
		scenarioText := strings.ReplaceAll(queryPoint.Scenario, "_", " ")
		switch variant {
		case 0:
			query.Text = scenarioText
		case 1:
			name = "owner/share ACL plus category and distance sort"
			query.ActorID = fmt.Sprintf("actor-%02d", index%23)
			query.Filters.CategoryIDs = []string{"hike", "bike"}
			query.Sort = ProductSortDistanceAsc
			query.Page = 2
		case 2:
			name = "federated-only exact count and created sort"
			query.ActorID = "actor-07"
			federated := true
			query.Filters.Federated = &federated
			query.Sort = ProductSortCreatedDesc
		case 3:
			name = "numeric filters and proximity sort"
			minimumDifficulty, maximumDifficulty := 1.0, 2.0
			query.Filters.Difficulty = ProductNumericRange{Min: &minimumDifficulty, Max: &maximumDifficulty}
			query.Sort = ProductSortProximityAsc
		case 4:
			name = "local text filter with deep stable page"
			query.ActorID = "actor-12"
			query.Text = "forest"
			federated := false
			query.Filters.Federated = &federated
			query.Sort = ProductSortCreatedAsc
			query.Page = 5
			query.PerPage = 5
		case 5:
			name = "subcategory ACL and descending duration"
			query.ActorID = "actor-19"
			query.Filters.SubcategoryIDs = []string{"day", "alpine"}
			query.Sort = ProductSortDurationDesc
			query.Page = 2
		}
		cases = append(cases, productBenchmarkCase{
			Name:       name,
			QueryPoint: queryPoint,
			Radius:     radius,
			Query:      query,
		})
	}
	return cases
}

func productRadiusForVariant(radii []float64, variant, index int) float64 {
	sorted := append([]float64(nil), radii...)
	for left := 0; left < len(sorted); left++ {
		for right := left + 1; right < len(sorted); right++ {
			if sorted[right] < sorted[left] {
				sorted[left], sorted[right] = sorted[right], sorted[left]
			}
		}
	}
	switch variant {
	case 2, 4:
		return sorted[len(sorted)-1]
	case 1, 5:
		return sorted[max(0, len(sorted)-2)]
	case 3:
		return sorted[min(1, len(sorted)-1)]
	default:
		return sorted[index%len(sorted)]
	}
}
