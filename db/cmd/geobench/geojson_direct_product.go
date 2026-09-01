package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/meilisearch/meilisearch-go"
)

// Direct and hybrid paths materialize the same final trail-list projection.
// In particular, neither response includes _geojson or raw route coordinates.
var geoJSONDirectProductAttributes = productPageAttributes()

type geoJSONDirectSearchResult struct {
	IDs                []string
	TotalHits          int
	TotalPages         int
	Page               int
	HitsPerPage        int
	WallDuration       time.Duration
	EngineProcessingMS float64
	ResponseBytes      int64
	RequestBytes       int64
	WireResponseBytes  int64
	HTTPRequests       int
	ShardQueries       int
	ShardFanout        int
	Truncated          bool
}

type geoJSONDirectProductSearch func(
	context.Context,
	ProductQuery,
	GeoJSONQueryPlan,
	bool,
	bool,
	int64,
	[]string,
	[]string,
) (geoJSONDirectSearchResult, error)

type geoJSONDirectPreparedProductCase struct {
	benchmark        productBenchmarkCase
	expectedPage     []string
	directPage       []string
	directTotalHits  int
	directTotalPages int
	directPageNumber int
	directPerPage    int
	caseReport       GeoJSONDirectProductCaseReport
}

type geoJSONDirectReferenceProductCase struct {
	benchmark     productBenchmarkCase
	normalized    ProductQuery
	sortValues    []string
	sortSupported bool
	expected      []string
	expectedPage  []string
	distances     map[string]float64
}

type geoJSONDirectProductBenchmark struct {
	documentsByID map[string]ProductQueryDocument
	references    []geoJSONDirectReferenceProductCase
	fullLimit     int64
	search        geoJSONDirectProductSearch
}

type geoJSONDirectPlanAudit struct {
	plan                 GeoJSONQueryPlan
	report               *GeoJSONDirectProductWorkloadReport
	prepared             []geoJSONDirectPreparedProductCase
	calibrationLatencies []float64
	search               geoJSONDirectProductSearch
	warmContainer        *meiliContainer
	// beforeConcurrentWorkers is a focused test seam for the QPS clock. The
	// production path leaves it nil.
	beforeConcurrentWorkers func()
	err                     error
}

type geoJSONDirectProductSample struct {
	caseIndex   int
	result      geoJSONDirectSearchResult
	completedAt time.Time
	err         error
}

// benchmarkGeoJSONDirectProductWorkload is the single-plan entry point used by
// focused tests. Production benchmark runs use
// benchmarkGeoJSONDirectProductCandidates so that all spatially qualified
// resolutions share one exact reference and are audited before selection.
func benchmarkGeoJSONDirectProductWorkload(
	ctx context.Context,
	client meilisearch.ServiceManager,
	dataset Dataset,
	radii []float64,
	repetitions, clients int,
	documents []ProductQueryDocument,
	plan GeoJSONQueryPlan,
	config GeoJSONDirectConfig,
) (*GeoJSONDirectProductWorkloadReport, error) {
	benchmark, err := prepareGeoJSONDirectProductBenchmark(ctx, client, dataset, radii, documents)
	return benchmarkGeoJSONDirectProductWorkloadPrepared(ctx, client, benchmark, err, repetitions, clients, plan, config)
}

func benchmarkGeoJSONDirectProductWorkloadPrepared(
	ctx context.Context,
	client meilisearch.ServiceManager,
	benchmark *geoJSONDirectProductBenchmark,
	prepareErr error,
	repetitions, clients int,
	plan GeoJSONQueryPlan,
	config GeoJSONDirectConfig,
) (*GeoJSONDirectProductWorkloadReport, error) {
	workload, _, err := benchmarkGeoJSONDirectProductWorkloadPreparedTimed(
		ctx, client, benchmark, prepareErr, repetitions, clients, plan, config, nil,
	)
	return workload, err
}

func benchmarkGeoJSONDirectProductWorkloadPreparedTimed(
	ctx context.Context,
	client meilisearch.ServiceManager,
	benchmark *geoJSONDirectProductBenchmark,
	prepareErr error,
	repetitions, clients int,
	plan GeoJSONQueryPlan,
	config GeoJSONDirectConfig,
	warmContainer *meiliContainer,
) (*GeoJSONDirectProductWorkloadReport, float64, error) {
	err := prepareErr
	if err != nil {
		return &GeoJSONDirectProductWorkloadReport{Status: "failed"}, 0, err
	}
	auditStarted := time.Now()
	audits, err := auditGeoJSONDirectProductPlans(ctx, client, benchmark, []GeoJSONQueryPlan{plan}, config)
	auditWallMS := milliseconds(time.Since(auditStarted))
	if err != nil {
		return &GeoJSONDirectProductWorkloadReport{Status: "failed"}, auditWallMS, err
	}
	if len(audits) != 1 {
		return &GeoJSONDirectProductWorkloadReport{Status: "failed"}, auditWallMS, fmt.Errorf("GeoJSON direct audit returned %d plans, want one", len(audits))
	}
	if audits[0].err != nil {
		return audits[0].report, auditWallMS, audits[0].err
	}
	// Warm requests are useful performance evidence even when the untimed
	// accuracy audit rejects a plan. The copied accuracy status keeps the
	// workload failed and prevents selection, while latency, throughput and RSS
	// remain measurable for diagnosing the rejected projection.
	audits[0].warmContainer = warmContainer
	workload, err := measureGeoJSONDirectProductWorkload(ctx, client, &audits[0], repetitions, clients)
	return workload, auditWallMS, err
}

// benchmarkGeoJSONDirectProductCandidates evaluates the real product request
// for every plan that passed the spatial screen. The exhaustive ProductQuery
// oracle plus exact point-to-polyline distances is prepared once per query and
// remains outside the timed path. Every completed audit runs the complete warm
// workload so rejected plans retain performance diagnostics. Selection still
// requires an accuracy pass and uses warm p95/p99, with calibration and the
// spatial screen ranking as deterministic tie breakers.
func benchmarkGeoJSONDirectProductCandidates(
	ctx context.Context,
	client meilisearch.ServiceManager,
	dataset Dataset,
	radii []float64,
	repetitions, clients int,
	documents []ProductQueryDocument,
	sweep *GeoJSONSweepReport,
	config GeoJSONDirectConfig,
) (*GeoJSONDirectReport, error) {
	direct := selectGeoJSONDirect(sweep, config)
	if direct == nil || sweep == nil {
		return direct, nil
	}
	direct.SelectionRule = geoJSONDirectWarmSelectionRule(false)
	indices := rankGeoJSONDirectCandidates(sweep, config)
	if len(indices) == 0 {
		return direct, nil
	}
	oracleStarted := time.Now()
	benchmark, err := prepareGeoJSONDirectProductBenchmark(ctx, client, dataset, radii, documents)
	direct.ProductOracleMS = milliseconds(time.Since(oracleStarted))
	if err != nil {
		direct.SelectionStatus = "product_failed"
		direct.Error = err.Error()
		return direct, err
	}
	plans := make([]GeoJSONQueryPlan, len(indices))
	for index, variantIndex := range indices {
		plans[index] = sweep.Variants[variantIndex].Plan
	}
	auditStarted := time.Now()
	audits, err := auditGeoJSONDirectProductPlans(ctx, client, benchmark, plans, config)
	direct.PlanAuditWallMS = milliseconds(time.Since(auditStarted))
	if err != nil {
		direct.SelectionStatus = "product_failed"
		direct.Error = err.Error()
		return direct, err
	}
	direct.ProductAudits = make([]GeoJSONDirectProductAuditReport, len(audits))
	qualified := make([]int, 0, len(audits))
	measurable := make([]int, 0, len(audits))
	for index := range audits {
		audit := &audits[index]
		status := audit.report.Status
		errorText := ""
		if audit.err != nil {
			status = "error"
			errorText = audit.err.Error()
		} else {
			measurable = append(measurable, index)
			if audit.report.Status == "audited" {
				qualified = append(qualified, index)
			}
		}
		plan := audit.plan
		direct.ProductAudits[index] = GeoJSONDirectProductAuditReport{
			Plan:               &plan,
			PlanID:             audit.plan.ID,
			Resolution:         audit.plan.Resolution,
			Status:             status,
			Cases:              audit.report.Cases,
			ProductPageLatency: audit.report.FirstSampleLatency,
			Accuracy:           audit.report.Accuracy,
			CountQualification: audit.report.CountDiagnostics,
			Error:              errorText,
		}
	}
	sort.SliceStable(qualified, func(i, j int) bool {
		left, right := audits[qualified[i]].report.FirstSampleLatency, audits[qualified[j]].report.FirstSampleLatency
		if left.P95 != right.P95 {
			return left.P95 < right.P95
		}
		if left.P99 != right.P99 {
			return left.P99 < right.P99
		}
		// audits preserve rankGeoJSONDirectCandidates order, so stable sorting
		// retains the engine ranking for equal product latency.
		return false
	})
	// Every successfully audited plan gets the complete repeated/concurrent
	// workload, including accuracy or count failures. Only full audit passers remain
	// selectable, but rejected plans still retain latency, throughput and RSS.
	for _, auditIndex := range measurable {
		workload, measureErr := measureGeoJSONDirectProductWorkload(
			ctx, client, &audits[auditIndex], repetitions, clients,
		)
		direct.ProductAudits[auditIndex].WarmWorkload = workload
		if measureErr != nil {
			direct.ProductAudits[auditIndex].WarmError = measureErr.Error()
			if ctx.Err() != nil {
				return direct, ctx.Err()
			}
		}
	}
	if len(qualified) == 0 {
		clearGeoJSONDirectSelection(direct)
		direct.SelectionStatus = "product_not_qualified"
		direct.Error = "no spatially qualified GeoJSON plan passed the product-query audit; warm diagnostics were still measured"
		return direct, nil
	}
	selectGeoJSONDirectWarmPlan(direct, false)
	return direct, nil
}

func selectGeoJSONDirectWarmPlan(direct *GeoJSONDirectReport, requirePerformance bool) bool {
	if direct == nil {
		return false
	}
	direct.SelectionRule = geoJSONDirectWarmSelectionRule(requirePerformance)
	eligible := make([]int, 0, len(direct.ProductAudits))
	warmPassed := false
	for index := range direct.ProductAudits {
		audit := &direct.ProductAudits[index]
		workload := audit.WarmWorkload
		if audit.Status != "audited" || audit.WarmError != "" || workload == nil ||
			workload.Status != "passed" || workload.Samples != workload.ExpectedSamples ||
			workload.UnstableWarmSamples != 0 {
			continue
		}
		warmPassed = true
		if requirePerformance && (audit.Performance == nil || audit.Performance.Status != "passed") {
			continue
		}
		eligible = append(eligible, index)
	}
	if len(eligible) == 0 {
		clearGeoJSONDirectSelection(direct)
		if requirePerformance && warmPassed {
			direct.SelectionStatus = "performance_not_qualified"
			direct.Error = "no warm-stable GeoJSON direct plan passed the active performance target"
		} else {
			direct.SelectionStatus = "warm_not_qualified"
			direct.Error = "no product-audited GeoJSON direct plan completed a stable warm workload"
		}
		return false
	}
	sort.SliceStable(eligible, func(i, j int) bool {
		left, right := direct.ProductAudits[eligible[i]], direct.ProductAudits[eligible[j]]
		if left.WarmWorkload.TotalLatency.P95 != right.WarmWorkload.TotalLatency.P95 {
			return left.WarmWorkload.TotalLatency.P95 < right.WarmWorkload.TotalLatency.P95
		}
		if left.WarmWorkload.TotalLatency.P99 != right.WarmWorkload.TotalLatency.P99 {
			return left.WarmWorkload.TotalLatency.P99 < right.WarmWorkload.TotalLatency.P99
		}
		if left.ProductPageLatency.P95 != right.ProductPageLatency.P95 {
			return left.ProductPageLatency.P95 < right.ProductPageLatency.P95
		}
		if left.ProductPageLatency.P99 != right.ProductPageLatency.P99 {
			return left.ProductPageLatency.P99 < right.ProductPageLatency.P99
		}
		// ProductAudits retain the spatial screen ranking for a final tie.
		return false
	})
	selected := &direct.ProductAudits[eligible[0]]
	plan := GeoJSONQueryPlan{
		ID:              selected.PlanID,
		RadiusMode:      geoJSONRadiusExact,
		Resolution:      selected.Resolution,
		RadiusFormula:   "r",
		ExactRefinement: false,
	}
	if selected.Plan != nil {
		plan = *selected.Plan
	}
	accuracy := selected.WarmWorkload.Accuracy
	direct.SelectedID = plan.ID
	direct.Plan = &plan
	direct.Accuracy = &accuracy
	direct.ProductWorkload = selected.WarmWorkload
	direct.Performance = selected.Performance
	direct.Error = ""
	if selected.WarmWorkload.Accuracy.UnsupportedSortCases > 0 {
		direct.SelectionStatus = "qualified_with_unsupported_proximity_sort"
	} else {
		direct.SelectionStatus = "qualified"
	}
	return true
}

func geoJSONDirectWarmSelectionRule(requirePerformance bool) string {
	if requirePerformance {
		return "spatial and product-audit pass; complete warm workload; active performance target pass; then lowest warm p95/p99, calibration p95/p99 and spatial rank"
	}
	return "spatial and product-audit pass; complete warm workload; then lowest warm p95/p99, calibration p95/p99 and spatial rank"
}

func clearGeoJSONDirectSelection(direct *GeoJSONDirectReport) {
	direct.SelectedID = ""
	direct.Plan = nil
	direct.Accuracy = nil
	direct.ProductWorkload = nil
	direct.Performance = nil
}

func prepareGeoJSONDirectProductBenchmark(
	ctx context.Context,
	client meilisearch.ServiceManager,
	dataset Dataset,
	radii []float64,
	documents []ProductQueryDocument,
) (*geoJSONDirectProductBenchmark, error) {
	if client == nil {
		return nil, fmt.Errorf("GeoJSON direct product workload requires a Meilisearch client")
	}
	return prepareGeoJSONDirectProductBenchmarkWithSearch(
		ctx,
		dataset,
		radii,
		documents,
		func(
			ctx context.Context,
			query ProductQuery,
			plan GeoJSONQueryPlan,
			includeGeo, allResults bool,
			fullLimit int64,
			attributes, sortValues []string,
		) (geoJSONDirectSearchResult, error) {
			return searchGeoJSONDirectProduct(
				ctx, client, query, plan, includeGeo, allResults, fullLimit, attributes, sortValues,
			)
		},
	)
}

func prepareGeoJSONDirectProductBenchmarkWithSearch(
	ctx context.Context,
	dataset Dataset,
	radii []float64,
	documents []ProductQueryDocument,
	search geoJSONDirectProductSearch,
) (*geoJSONDirectProductBenchmark, error) {
	if search == nil {
		return nil, fmt.Errorf("GeoJSON direct product workload requires a search function")
	}
	if len(documents) == 0 {
		return nil, fmt.Errorf("GeoJSON direct product workload requires product documents")
	}
	documentsByID := productDocumentsByID(documents)
	if len(documentsByID) != len(documents) {
		return nil, fmt.Errorf("GeoJSON direct product workload requires unique product document IDs")
	}
	cases := buildProductBenchmarkCases(dataset, radii)
	if len(cases) == 0 {
		return nil, fmt.Errorf("GeoJSON direct product workload has no query cases")
	}
	benchmark := &geoJSONDirectProductBenchmark{
		documentsByID: documentsByID,
		references:    make([]geoJSONDirectReferenceProductCase, len(cases)),
		fullLimit:     int64(len(documents) + 1),
		search:        search,
	}
	for index, benchmarkCase := range cases {
		normalized, _, err := normalizeProductQuery(benchmarkCase.Query)
		if err != nil {
			return nil, fmt.Errorf("normalize product query %q: %w", benchmarkCase.Name, err)
		}
		sortValues, sortSupported := geoJSONDirectMeiliSort(normalized)
		allQuery := normalized
		allQuery.Page = 1
		allQuery.PerPage = len(documents) + 1
		oracle, err := RunProductQueryExhaustive(ctx, documents, allQuery)
		if err != nil {
			return nil, fmt.Errorf("run exhaustive product oracle for %q: %w", benchmarkCase.Name, err)
		}
		pageOracle, err := RunProductQueryExhaustive(ctx, documents, normalized)
		if err != nil {
			return nil, fmt.Errorf("run paged exhaustive product oracle for %q: %w", benchmarkCase.Name, err)
		}
		expected, distances, err := geoJSONDirectProductOracleResult(oracle)
		if err != nil {
			return nil, fmt.Errorf("prepare exhaustive product oracle for %q: %w", benchmarkCase.Name, err)
		}
		expectedPage := geoJSONDirectProductHitIDs(pageOracle.Hits)
		if pageOracle.TotalItems != len(expected) ||
			pageOracle.TotalPages != productTotalPages(len(expected), normalized.PerPage) ||
			pageOracle.Page != normalized.Page || pageOracle.PerPage != normalized.PerPage ||
			!equalStringSlices(expectedPage, pageOfIDs(expected, normalized.Page, normalized.PerPage)) {
			return nil, fmt.Errorf("paged exhaustive product oracle for %q is inconsistent with its full ordering", benchmarkCase.Name)
		}
		benchmark.references[index] = geoJSONDirectReferenceProductCase{
			benchmark:     benchmarkCase,
			normalized:    normalized,
			sortValues:    sortValues,
			sortSupported: sortSupported,
			expected:      expected,
			expectedPage:  expectedPage,
			distances:     distances,
		}
	}
	return benchmark, nil
}

func auditGeoJSONDirectProductPlans(
	ctx context.Context,
	client meilisearch.ServiceManager,
	benchmark *geoJSONDirectProductBenchmark,
	plans []GeoJSONQueryPlan,
	config GeoJSONDirectConfig,
) ([]geoJSONDirectPlanAudit, error) {
	if benchmark == nil {
		return nil, fmt.Errorf("GeoJSON direct product audit requires a prepared benchmark")
	}
	audits := make([]geoJSONDirectPlanAudit, len(plans))
	accumulators := make([]*geoJSONDirectAccuracyAccumulator, len(plans))
	for index, plan := range plans {
		audits[index] = geoJSONDirectPlanAudit{
			plan:     plan,
			report:   &GeoJSONDirectProductWorkloadReport{Status: "failed"},
			prepared: make([]geoJSONDirectPreparedProductCase, len(benchmark.references)),
			search:   benchmark.search,
		}
		accumulators[index] = newGeoJSONDirectAccuracyAccumulator(config)
		if plan.Conservative || plan.ExactRefinement || plan.RadiusMode != geoJSONRadiusExact {
			audits[index].err = fmt.Errorf("GeoJSON direct product workload requires a non-conservative exact-radius plan, got %q", plan.ID)
		}
	}

	// Rotate the first plan per case so cache warmth cannot systematically
	// favour one resolution during product-page calibration.
	for caseIndex, reference := range benchmark.references {
		for offset := range audits {
			planIndex := (caseIndex + offset) % len(audits)
			audit := &audits[planIndex]
			if audit.err != nil {
				continue
			}
			prepared, observation, latency, err := auditGeoJSONDirectProductCase(
				ctx, benchmark.search, reference, benchmark.documentsByID, audit.plan, config, benchmark.fullLimit,
			)
			if err != nil {
				if ctx.Err() != nil {
					return audits, ctx.Err()
				}
				audit.err = fmt.Errorf("audit GeoJSON direct product query %q: %w", reference.benchmark.Name, err)
				continue
			}
			audit.prepared[caseIndex] = prepared
			audit.calibrationLatencies = append(audit.calibrationLatencies, latency)
			accumulators[planIndex].add(observation)
		}
	}
	for index := range audits {
		audit := &audits[index]
		audit.report.Cases = len(audit.calibrationLatencies)
		audit.report.FirstSampleLatency = latencyStats(audit.calibrationLatencies)
		if audit.err != nil {
			continue
		}
		audit.report.CaseReports = make([]GeoJSONDirectProductCaseReport, len(audit.prepared))
		for caseIndex := range audit.prepared {
			audit.report.CaseReports[caseIndex] = audit.prepared[caseIndex].caseReport
		}
		audit.report.CountDiagnostics = summarizeGeoJSONDirectProductCounts(
			audit.report.CaseReports,
			config.MaximumCountP95RelativeError,
		)
		audit.report.Accuracy = accumulators[index].finish(config, true)
		if audit.report.Accuracy.Status == "passed" && audit.report.CountDiagnostics.Status == "passed" {
			audit.report.Status = "audited"
		} else if audit.report.CountDiagnostics.Status == "inconclusive" {
			audit.report.Status = "count_inconclusive"
		} else if audit.report.CountDiagnostics.Status == "failed" {
			audit.report.Status = "count_failed"
		} else {
			audit.report.Status = "accuracy_failed"
		}
	}
	return audits, nil
}

func auditGeoJSONDirectProductCase(
	ctx context.Context,
	search geoJSONDirectProductSearch,
	reference geoJSONDirectReferenceProductCase,
	documentsByID map[string]ProductQueryDocument,
	plan GeoJSONQueryPlan,
	config GeoJSONDirectConfig,
	fullLimit int64,
) (geoJSONDirectPreparedProductCase, geoJSONDirectObservation, float64, error) {
	prepared := geoJSONDirectPreparedProductCase{benchmark: reference.benchmark}
	direct, err := search(
		ctx, reference.normalized, plan, true, true, fullLimit, []string{"id"}, reference.sortValues,
	)
	if err != nil {
		return prepared, geoJSONDirectObservation{}, 0, fmt.Errorf("search complete direct result: %w", err)
	}
	page, err := search(
		ctx, reference.normalized, plan, true, false, 0, geoJSONDirectProductAttributes, reference.sortValues,
	)
	if err != nil {
		return prepared, geoJSONDirectObservation{}, 0, fmt.Errorf("search direct product page: %w", err)
	}
	if direct.Truncated {
		return prepared, geoJSONDirectObservation{}, 0, fmt.Errorf("direct accuracy result was truncated; raise pagination.maxTotalHits")
	}
	if err := ensureGeoJSONDirectDistances(ctx, direct.IDs, reference.distances, documentsByID, reference.normalized); err != nil {
		return prepared, geoJSONDirectObservation{}, 0, err
	}
	canonicalDirectIDs := append([]string(nil), direct.IDs...)
	directOrderMismatch := false
	if reference.sortSupported {
		canonicalDirectIDs, err = geoJSONDirectSubsetProductOrder(ctx, direct.IDs, documentsByID, reference.normalized)
		if err != nil {
			return prepared, geoJSONDirectObservation{}, 0, fmt.Errorf("order direct product subset: %w", err)
		}
		directOrderMismatch = !equalStringSlices(direct.IDs, canonicalDirectIDs)
	}
	expectedDirectPage := pageOfIDs(canonicalDirectIDs, reference.normalized.Page, reference.normalized.PerPage)
	expectedDirectPages := productTotalPages(len(canonicalDirectIDs), reference.normalized.PerPage)
	if err := validateGeoJSONDirectProductPage(
		page,
		expectedDirectPage,
		len(canonicalDirectIDs),
		expectedDirectPages,
		reference.normalized.Page,
		reference.normalized.PerPage,
	); err != nil {
		return prepared, geoJSONDirectObservation{}, 0, err
	}
	observation := observeGeoJSONDirectProductCase(
		reference.benchmark,
		reference.normalized,
		direct.IDs,
		page.IDs,
		reference.expected,
		reference.expectedPage,
		reference.distances,
		documentsByID,
		config,
		reference.sortSupported,
	)
	if directOrderMismatch {
		observation.semanticFalsePositives++
		observation.semanticError = appendGeoJSONDirectError(
			observation.semanticError,
			"complete direct ordering differs from the ProductQuery ordering of the same returned documents",
		)
		observation.violation.Error = observation.semanticError
	}
	observation.truncated = direct.Truncated

	prepared.expectedPage = append([]string(nil), reference.expectedPage...)
	prepared.directPage = append([]string(nil), expectedDirectPage...)
	prepared.directTotalHits = len(canonicalDirectIDs)
	prepared.directTotalPages = expectedDirectPages
	prepared.directPageNumber = reference.normalized.Page
	prepared.directPerPage = reference.normalized.PerPage
	pageRecall := ratioOrOne(observation.comparablePageReturned, observation.comparablePageExpected)
	pagePrecision := ratioOrOne(
		observation.comparablePageActual-observation.comparablePageMaterialFP,
		observation.comparablePageActual,
	)
	errorText := ""
	if !reference.sortSupported {
		errorText = "route-proximity sorting is unsupported for _geojson; Meilisearch executed the remaining direct query without a sort rule"
	}
	if observation.semanticError != "" {
		if errorText != "" {
			errorText += "; "
		}
		errorText += observation.semanticError
	}
	prepared.caseReport = GeoJSONDirectProductCaseReport{
		Name:                   reference.benchmark.Name,
		QueryID:                reference.benchmark.QueryPoint.ID,
		Scenario:               reference.benchmark.QueryPoint.Scenario,
		RadiusMeters:           reference.benchmark.Radius,
		Sort:                   string(reference.normalized.Sort),
		SortSupported:          reference.sortSupported,
		PageComparable:         reference.sortSupported,
		ExpectedTotalItems:     len(reference.expected),
		ActualTotalItems:       page.TotalHits,
		SignedCountDelta:       page.TotalHits - len(reference.expected),
		FalseNegatives:         observation.falseNegatives,
		BoundaryFalseNegatives: observation.boundaryMisses,
		MaterialFalseNegatives: observation.materialFalseNegatives,
		FalsePositives:         observation.falsePositives,
		BoundaryFalsePositives: observation.allowedBoundaryFalsePos,
		MaterialFalsePositives: observation.materialFalsePositives,
		SemanticFalsePositives: observation.semanticFalsePositives,
		ExpectedTotalPages:     productTotalPages(len(reference.expected), reference.normalized.PerPage),
		ActualTotalPages:       page.TotalPages,
		ExpectedHitIDs:         append([]string(nil), reference.expectedPage...),
		ActualHitIDs:           append([]string(nil), page.IDs...),
		PageRecall:             pageRecall,
		PagePrecision:          pagePrecision,
		FalseEmpty:             observation.falseEmpty,
		CorrectSemantics:       observation.semanticFalsePositives == 0,
		ShardFanout:            page.ShardFanout,
		HTTPRequests:           page.HTTPRequests,
		ShardQueries:           page.ShardQueries,
		WireResponseBytes:      page.WireResponseBytes,
		RequestBytes:           page.RequestBytes,
		Error:                  errorText,
	}
	return prepared, observation, milliseconds(page.WallDuration), nil
}

func summarizeGeoJSONDirectProductCounts(
	cases []GeoJSONDirectProductCaseReport,
	maximumP95RelativeError float64,
) GeoJSONDirectProductCountReport {
	if maximumP95RelativeError <= 0 {
		maximumP95RelativeError = defaultGeoJSONDirectMaximumCountP95RelativeError
	}
	report := GeoJSONDirectProductCountReport{
		Status:                  "inconclusive",
		MaximumP95RelativeError: maximumP95RelativeError,
		Cases:                   len(cases),
	}
	absoluteErrors := make([]float64, 0, len(cases))
	relativeErrors := make([]float64, 0, len(cases))
	for _, testCase := range cases {
		delta := testCase.ActualTotalItems - testCase.ExpectedTotalItems
		absolute := math.Abs(float64(delta))
		relative := absolute / float64(max(1, testCase.ExpectedTotalItems))
		absoluteErrors = append(absoluteErrors, absolute)
		relativeErrors = append(relativeErrors, relative)
		if delta == 0 {
			report.ExactCountCases++
		}
		if relative <= 0.01 {
			report.WithinOnePercentCases++
		}
		if testCase.ActualTotalPages == testCase.ExpectedTotalPages {
			report.TotalPagesMatchCases++
		}
	}
	report.ExactCountRate = ratio(report.ExactCountCases, report.Cases)
	report.WithinOnePercentRate = ratio(report.WithinOnePercentCases, report.Cases)
	report.TotalPagesMatchRate = ratio(report.TotalPagesMatchCases, report.Cases)
	report.CountError = directCountError(absoluteErrors, relativeErrors)
	if report.Cases > 0 {
		if report.CountError.P95Relative <= maximumP95RelativeError {
			report.Status = "passed"
		} else {
			report.Status = "failed"
		}
	}
	return report
}

func measureGeoJSONDirectProductWorkload(
	ctx context.Context,
	client meilisearch.ServiceManager,
	audit *geoJSONDirectPlanAudit,
	repetitions, clients int,
) (*GeoJSONDirectProductWorkloadReport, error) {
	if audit == nil || audit.report == nil {
		return &GeoJSONDirectProductWorkloadReport{Status: "failed"}, fmt.Errorf("GeoJSON direct measurement requires a successful product audit")
	}
	if repetitions < 1 {
		repetitions = 1
	}
	if clients < 1 {
		clients = 1
	}
	value := *audit.report
	report := &value
	prepared := audit.prepared
	search := audit.search
	if search == nil && client != nil {
		search = func(
			ctx context.Context,
			query ProductQuery,
			plan GeoJSONQueryPlan,
			includeGeo, allResults bool,
			fullLimit int64,
			attributes, sortValues []string,
		) (geoJSONDirectSearchResult, error) {
			return searchGeoJSONDirectProduct(
				ctx, client, query, plan, includeGeo, allResults, fullLimit, attributes, sortValues,
			)
		}
	}
	if search == nil {
		return report, fmt.Errorf("GeoJSON direct measurement requires a search function")
	}
	report.Clients = min(clients, max(1, len(prepared)*repetitions))
	report.ExpectedSamples = len(prepared) * repetitions

	totalJobs := report.ExpectedSamples
	jobs := make(chan int)
	// Buffer the finite warm workload so benchmark-only stability checks in
	// the collector cannot stall the measured request workers.
	results := make(chan geoJSONDirectProductSample, totalJobs)
	var engineMonitor, clientMonitor *rssMonitor
	if audit.warmContainer != nil {
		// Correctness audits intentionally allocate complete ID sets and oracle
		// bookkeeping. Release their dead pages before taking the warm-search
		// baseline so the reported delta covers only concurrent page requests.
		debug.FreeOSMemory()
		engineMonitor = startContainerRSSMonitor(audit.warmContainer)
		clientMonitor = startRSSMonitor(os.Getpid())
	}
	concurrentStarted := time.Now()
	if audit.beforeConcurrentWorkers != nil {
		audit.beforeConcurrentWorkers()
	}
	var workers sync.WaitGroup
	workers.Add(report.Clients)
	for range report.Clients {
		go func() {
			defer workers.Done()
			for caseIndex := range jobs {
				results <- executeGeoJSONDirectProductSample(ctx, search, caseIndex, prepared[caseIndex], audit.plan)
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

	type radiusAccumulator struct {
		latencies []float64
	}
	radiusGroups := make(map[float64]*radiusAccumulator)
	totalLatencies := make([]float64, 0, totalJobs)
	engineLatencies := make([]float64, 0, totalJobs)
	var responseBytes, requestBytes, wireResponseBytes, httpRequests, shardQueries int64
	shardFanout := 0
	var firstError error
	lastCompletedAt := concurrentStarted
	for sample := range results {
		if sample.err != nil {
			if firstError == nil {
				firstError = fmt.Errorf("GeoJSON direct product query %q: %w", prepared[sample.caseIndex].benchmark.Name, sample.err)
			}
			continue
		}
		if sample.completedAt.After(lastCompletedAt) {
			lastCompletedAt = sample.completedAt
		}
		latency := milliseconds(sample.result.WallDuration)
		totalLatencies = append(totalLatencies, latency)
		engineLatencies = append(engineLatencies, sample.result.EngineProcessingMS)
		responseBytes += sample.result.ResponseBytes
		requestBytes += sample.result.RequestBytes
		wireResponseBytes += sample.result.WireResponseBytes
		httpRequests += int64(sample.result.HTTPRequests)
		shardQueries += int64(sample.result.ShardQueries)
		if sample.result.ShardFanout > shardFanout {
			shardFanout = sample.result.ShardFanout
		}
		report.Samples++
		if !stableGeoJSONDirectProductResult(prepared[sample.caseIndex], sample.result) {
			report.UnstableWarmSamples++
		}
		radius := prepared[sample.caseIndex].benchmark.Radius
		group := radiusGroups[radius]
		if group == nil {
			group = &radiusAccumulator{}
			radiusGroups[radius] = group
		}
		group.latencies = append(group.latencies, latency)
	}
	report.ConcurrentWallMS = milliseconds(lastCompletedAt.Sub(concurrentStarted))
	if engineMonitor != nil {
		report.EngineBaselineBytes, report.EnginePeakBytes, report.EnginePeakDelta = engineMonitor.finish()
		report.ClientBaselineRSSBytes, report.ClientPeakRSSBytes, report.ClientPeakRSSDelta = clientMonitor.finish()
	}
	if firstError != nil {
		return report, firstError
	}
	report.TotalLatency = latencyStats(totalLatencies)
	report.EngineProcessing = latencyStats(engineLatencies)
	if report.Samples > 0 {
		report.AverageResponseBytes = float64(responseBytes) / float64(report.Samples)
		report.AverageRequestBytes = float64(requestBytes) / float64(report.Samples)
		report.AverageWireResponseBytes = float64(wireResponseBytes) / float64(report.Samples)
		report.AverageHTTPRequests = float64(httpRequests) / float64(report.Samples)
		report.AverageShardQueries = float64(shardQueries) / float64(report.Samples)
		report.ShardFanout = shardFanout
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
		report.RadiusReports = append(report.RadiusReports, GeoJSONDirectRadiusReport{
			RadiusMeters: radius,
			Samples:      len(group.latencies),
			TotalLatency: latencyStats(group.latencies),
		})
	}

	if report.Samples == 0 {
		report.Status = "inconclusive"
	} else if report.Samples == report.ExpectedSamples && report.Accuracy.Status == "passed" &&
		report.CountDiagnostics.Status == "passed" && report.UnstableWarmSamples == 0 {
		report.Status = "passed"
	} else {
		report.Status = "failed"
	}
	return report, nil
}

func executeGeoJSONDirectProductSample(
	ctx context.Context,
	search geoJSONDirectProductSearch,
	caseIndex int,
	prepared geoJSONDirectPreparedProductCase,
	plan GeoJSONQueryPlan,
) geoJSONDirectProductSample {
	started := time.Now()
	normalized, _, err := normalizeProductQuery(prepared.benchmark.Query)
	if err != nil {
		return geoJSONDirectProductSample{caseIndex: caseIndex, err: err}
	}
	sortValues, _ := geoJSONDirectMeiliSort(normalized)
	result, err := search(
		ctx,
		normalized,
		plan,
		true,
		false,
		0,
		geoJSONDirectProductAttributes,
		sortValues,
	)
	result.WallDuration = time.Since(started)
	return geoJSONDirectProductSample{caseIndex: caseIndex, result: result, completedAt: time.Now(), err: err}
}

func stableGeoJSONDirectProductResult(prepared geoJSONDirectPreparedProductCase, actual geoJSONDirectSearchResult) bool {
	if actual.TotalHits != prepared.directTotalHits ||
		actual.TotalPages != prepared.directTotalPages ||
		actual.Page != prepared.directPageNumber ||
		actual.HitsPerPage != prepared.directPerPage ||
		len(actual.IDs) != len(prepared.directPage) {
		return false
	}
	for index := range actual.IDs {
		if actual.IDs[index] != prepared.directPage[index] {
			return false
		}
	}
	return true
}

func searchGeoJSONDirectProduct(
	ctx context.Context,
	client meilisearch.ServiceManager,
	query ProductQuery,
	plan GeoJSONQueryPlan,
	includeGeo, allResults bool,
	fullLimit int64,
	attributes, sortValues []string,
) (geoJSONDirectSearchResult, error) {
	started := time.Now()
	spatialFilter := ""
	if includeGeo {
		if query.Geo == nil {
			return geoJSONDirectSearchResult{}, fmt.Errorf("GeoJSON direct product query requires a geo radius")
		}
		candidateRadius, err := geoJSONCandidateRadius(query.Geo.RadiusMeters, plan)
		if err != nil {
			return geoJSONDirectSearchResult{}, err
		}
		spatialFilter = geoRadiusFilter(query.Geo.Center, candidateRadius, plan.Resolution)
	}
	filter, err := productMeiliFilter(query, spatialFilter)
	if err != nil {
		return geoJSONDirectSearchResult{}, err
	}
	request := &meilisearch.SearchRequest{
		Filter:               filter,
		AttributesToRetrieve: append([]string(nil), attributes...),
		Sort:                 append([]string(nil), sortValues...),
		MatchingStrategy:     meilisearch.All,
	}
	if allResults {
		request.Limit = fullLimit
	} else {
		request.HitsPerPage = int64(query.PerPage)
		request.Page = int64(query.Page)
	}

	response, err := client.Index(benchmarkIndexUID).SearchWithContext(ctx, query.Text, request)
	if err != nil {
		return geoJSONDirectSearchResult{}, err
	}
	ids := make([]string, 0, len(response.Hits))
	seen := make(map[string]struct{}, len(response.Hits))
	for _, hit := range response.Hits {
		rawID, ok := hit["id"]
		if !ok {
			return geoJSONDirectSearchResult{}, fmt.Errorf("direct product search hit has no id")
		}
		var id string
		if err := json.Unmarshal(rawID, &id); err != nil {
			return geoJSONDirectSearchResult{}, fmt.Errorf("decode direct product hit id: %w", err)
		}
		if _, duplicate := seen[id]; duplicate {
			return geoJSONDirectSearchResult{}, fmt.Errorf("direct product search returned duplicate id %q", id)
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	totalHits := response.TotalHits
	if allResults {
		totalHits = response.EstimatedTotalHits
	}
	if totalHits == 0 && len(ids) > 0 {
		totalHits = int64(len(ids))
	}
	var encoded []byte
	if allResults {
		encoded, err = json.Marshal(response)
	} else {
		rawHits := make([]map[string]json.RawMessage, len(response.Hits))
		for index := range response.Hits {
			rawHits[index] = map[string]json.RawMessage(response.Hits[index])
		}
		encoded, err = marshalMeiliProductPagePayload(
			rawHits,
			int(totalHits),
			int(response.TotalPages),
			int(response.Page),
			int(response.HitsPerPage),
		)
	}
	if err != nil {
		return geoJSONDirectSearchResult{}, fmt.Errorf("materialize direct product response: %w", err)
	}
	result := geoJSONDirectSearchResult{
		IDs:                ids,
		TotalHits:          int(totalHits),
		TotalPages:         int(response.TotalPages),
		Page:               int(response.Page),
		HitsPerPage:        int(response.HitsPerPage),
		EngineProcessingMS: float64(response.ProcessingTimeMs),
		ResponseBytes:      int64(len(encoded)),
		HTTPRequests:       1,
		ShardQueries:       1,
		ShardFanout:        1,
		Truncated:          allResults && totalHits > int64(len(ids)),
	}
	result.WallDuration = time.Since(started)
	return result, nil
}

// productMeiliFilter contains only predicates that can be shared by the
// direct GeoJSON path and candidate-plus-exact backends. Callers supply their
// own optional spatial predicate and decide whether sorting and pagination are
// safe before an exact refinement stage.
func productMeiliFilter(query ProductQuery, spatialFilter string) (string, error) {
	normalized, _, err := normalizeProductQuery(query)
	if err != nil {
		return "", err
	}
	parts := make([]string, 0, 12)
	if normalized.ActorID == "" {
		parts = append(parts, "public = true")
	} else {
		actor := strconv.Quote(normalized.ActorID)
		parts = append(parts, "(public = true OR author = "+actor+" OR shares = "+actor+")")
	}

	taxonomy := make([]string, 0, 3)
	if len(normalized.Filters.CategoryIDs) > 0 {
		taxonomy = append(taxonomy, "category_id IN "+meiliQuotedList(normalized.Filters.CategoryIDs))
	}
	if len(normalized.Filters.SubcategoryIDs) > 0 {
		taxonomy = append(taxonomy, "subcategory_id IN "+meiliQuotedList(normalized.Filters.SubcategoryIDs))
	}
	if len(normalized.Filters.CategoriesWithoutSubcategoryIDs) > 0 {
		taxonomy = append(taxonomy,
			"(category_id IN "+meiliQuotedList(normalized.Filters.CategoriesWithoutSubcategoryIDs)+" AND subcategory_id IS NULL)",
		)
	}
	if len(taxonomy) > 0 {
		parts = append(parts, "("+strings.Join(taxonomy, " OR ")+")")
	}
	if normalized.Filters.Federated != nil {
		parts = append(parts, "is_federated = "+strconv.FormatBool(*normalized.Filters.Federated))
	}
	appendMeiliNumericRange := func(field string, value ProductNumericRange) {
		if value.Min != nil {
			parts = append(parts, field+" >= "+strconv.FormatFloat(*value.Min, 'g', -1, 64))
		}
		if value.Max != nil {
			parts = append(parts, field+" <= "+strconv.FormatFloat(*value.Max, 'g', -1, 64))
		}
	}
	appendMeiliNumericRange("distance", normalized.Filters.DistanceMeters)
	appendMeiliNumericRange("duration", normalized.Filters.DurationSeconds)
	appendMeiliNumericRange("elevation_gain", normalized.Filters.ElevationGainMeters)
	appendMeiliNumericRange("elevation_loss", normalized.Filters.ElevationLossMeters)
	appendMeiliNumericRange("difficulty", normalized.Filters.Difficulty)

	if strings.TrimSpace(spatialFilter) != "" {
		parts = append(parts, "("+spatialFilter+")")
	}
	return strings.Join(parts, " AND "), nil
}

func geoJSONDirectMeiliSort(query ProductQuery) ([]string, bool) {
	var field, direction string
	switch query.Sort {
	case ProductSortRelevance:
		// `sort` appears after Meilisearch's text-ranking rules in the
		// benchmark settings. The explicit ID rule therefore only resolves
		// equal-ranking documents and prevents federated search from using
		// query/shard position as an implicit global tie-breaker.
		return []string{"id:asc"}, true
	case ProductSortNameAsc:
		field, direction = "name", "asc"
	case ProductSortNameDesc:
		field, direction = "name", "desc"
	case ProductSortCreatedAsc:
		field, direction = "created", "asc"
	case ProductSortCreatedDesc:
		field, direction = "created", "desc"
	case ProductSortDistanceAsc:
		field, direction = "distance", "asc"
	case ProductSortDistanceDesc:
		field, direction = "distance", "desc"
	case ProductSortDurationAsc:
		field, direction = "duration", "asc"
	case ProductSortDurationDesc:
		field, direction = "duration", "desc"
	case ProductSortElevationGainAsc:
		field, direction = "elevation_gain", "asc"
	case ProductSortElevationGainDesc:
		field, direction = "elevation_gain", "desc"
	case ProductSortElevationLossAsc:
		field, direction = "elevation_loss", "asc"
	case ProductSortElevationLossDesc:
		field, direction = "elevation_loss", "desc"
	case ProductSortDifficultyAsc:
		field, direction = "difficulty", "asc"
	case ProductSortDifficultyDesc:
		field, direction = "difficulty", "desc"
	case ProductSortProximityAsc, ProductSortProximityDesc:
		// Meilisearch can apply _geoRadius to _geojson, but _geoPoint only
		// sorts the separate _geo point field. Silently sorting by the route's
		// start point would claim a route-proximity guarantee that does not
		// exist, so this case is deliberately visible as unsupported.
		return nil, false
	default:
		return nil, false
	}
	return []string{field + ":" + direction, "id:asc"}, true
}

func meiliQuotedList(values []string) string {
	quoted := make([]string, len(values))
	for index, value := range values {
		quoted[index] = strconv.Quote(value)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func geoJSONDirectProductOracleResult(result ProductQueryResult) ([]string, map[string]float64, error) {
	ids := make([]string, len(result.Hits))
	distances := make(map[string]float64, len(result.Hits))
	for index, hit := range result.Hits {
		if hit.ProximityMeters == nil || math.IsNaN(*hit.ProximityMeters) || math.IsInf(*hit.ProximityMeters, 0) {
			return nil, nil, fmt.Errorf("exhaustive product oracle has no finite proximity for trail %q", hit.ID)
		}
		ids[index] = hit.ID
		distances[hit.ID] = *hit.ProximityMeters
	}
	return ids, distances, nil
}

func geoJSONDirectProductHitIDs(hits []ProductQueryHit) []string {
	ids := make([]string, len(hits))
	for index, hit := range hits {
		ids[index] = hit.ID
	}
	return ids
}

func geoJSONDirectSubsetProductOrder(
	ctx context.Context,
	ids []string,
	documentsByID map[string]ProductQueryDocument,
	query ProductQuery,
) ([]string, error) {
	documents := make([]ProductQueryDocument, 0, len(ids))
	for _, id := range ids {
		if document, known := documentsByID[id]; known {
			documents = append(documents, document)
		}
	}
	query.Geo = nil
	query.Page = 1
	query.PerPage = len(ids) + 1
	result, err := RunProductQueryExhaustive(ctx, documents, query)
	if err != nil {
		return nil, err
	}
	return geoJSONDirectProductHitIDs(result.Hits), nil
}

// ensureGeoJSONDirectDistances extends the exhaustive oracle's inside-radius
// distance map only for direct hits. It is outside all timed requests and lets
// the audit distinguish tolerated boundary hits from material false positives.
func ensureGeoJSONDirectDistances(
	ctx context.Context,
	ids []string,
	distances map[string]float64,
	documentsByID map[string]ProductQueryDocument,
	query ProductQuery,
) error {
	if query.Geo == nil {
		return fmt.Errorf("GeoJSON direct distance audit requires a geo radius")
	}
	for index, id := range ids {
		if index%128 == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		if _, measured := distances[id]; measured {
			continue
		}
		document, known := documentsByID[id]
		if !known {
			continue
		}
		distance, err := pointToTrailDistanceMetersContext(ctx, query.Geo.Center, document.Trail)
		if err != nil {
			return fmt.Errorf("exact distance for direct trail %q: %w", id, err)
		}
		if math.IsNaN(distance) || math.IsInf(distance, 0) {
			return fmt.Errorf("exact distance for direct trail %q is not finite", id)
		}
		distances[id] = distance
	}
	return nil
}

func validateGeoJSONDirectProductPage(
	actual geoJSONDirectSearchResult,
	expectedIDs []string,
	expectedTotalHits, expectedTotalPages, expectedPage, expectedPerPage int,
) error {
	if !equalStringSlices(actual.IDs, expectedIDs) {
		return fmt.Errorf("direct paginated IDs %v do not equal page slice %v of the complete direct ordering", actual.IDs, expectedIDs)
	}
	if actual.TotalHits != expectedTotalHits {
		return fmt.Errorf("direct paginated totalHits=%d, want %d from the complete direct result", actual.TotalHits, expectedTotalHits)
	}
	if actual.TotalPages != expectedTotalPages {
		return fmt.Errorf("direct paginated totalPages=%d, want %d", actual.TotalPages, expectedTotalPages)
	}
	if actual.Page != expectedPage {
		return fmt.Errorf("direct paginated page=%d, want %d", actual.Page, expectedPage)
	}
	if actual.HitsPerPage != expectedPerPage {
		return fmt.Errorf("direct paginated hitsPerPage=%d, want %d", actual.HitsPerPage, expectedPerPage)
	}
	return nil
}

func productTotalPages(totalItems, perPage int) int {
	if totalItems == 0 {
		return 0
	}
	return 1 + (totalItems-1)/perPage
}

func equalStringSlices(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func observeGeoJSONDirectProductCase(
	benchmarkCase productBenchmarkCase,
	query ProductQuery,
	directIDs, directPage, expectedIDs, expectedPage []string,
	distances map[string]float64,
	documentsByID map[string]ProductQueryDocument,
	config GeoJSONDirectConfig,
	sortSupported bool,
) geoJSONDirectObservation {
	observation := geoJSONDirectObservation{
		returned:        len(directIDs),
		expected:        len(expectedIDs),
		unsupportedSort: !sortSupported,
	}
	if query.Geo == nil {
		return observation
	}
	directSet := stringSet(directIDs)
	expectedSet := stringSet(expectedIDs)
	terms := normalizedProductTextTerms(query.Text)
	semanticallyValid := func(document ProductQueryDocument, known bool) bool {
		if !known || !productDocumentVisibleTo(document, query.ActorID) ||
			!productDocumentMatchesFilters(document, query.Filters) {
			return false
		}
		_, matchesText := productDocumentTextScore(document, terms)
		return matchesText
	}
	numericalTolerance := math.Max(0.5, query.Geo.RadiusMeters*1e-7)
	clearThreshold := query.Geo.RadiusMeters - config.BoundaryToleranceMeters - numericalTolerance
	outerThreshold := query.Geo.RadiusMeters + config.BoundaryToleranceMeters + numericalTolerance
	clearIDs := make([]string, 0, len(expectedIDs))
	expectedClearOrder := make([]string, 0, len(expectedIDs))

	for _, id := range expectedIDs {
		distance := distances[id]
		_, found := directSet[id]
		clearInside := distance < clearThreshold
		if clearInside {
			observation.clearExpected++
			clearIDs = append(clearIDs, id)
			expectedClearOrder = append(expectedClearOrder, id)
		}
		if found {
			observation.truePositives++
			if clearInside {
				observation.clearTruePositives++
			}
			continue
		}
		observation.falseNegatives++
		observation.maxInsideMiss = max(observation.maxInsideMiss, query.Geo.RadiusMeters-distance)
		if clearInside {
			observation.materialFalseNegatives++
			observation.materialFalseNegativeIDs = retainWorstGeoJSONDirectTrailViolation(
				observation.materialFalseNegativeIDs,
				GeoJSONDirectTrailViolationDetail{
					TrailID:           id,
					DistanceMeters:    distance,
					RadiusDeltaMeters: query.Geo.RadiusMeters - distance,
				},
			)
		} else {
			observation.boundaryMisses++
		}
	}

	directClearOrder := make([]string, 0, len(directIDs))
	for _, id := range directIDs {
		document, known := documentsByID[id]
		if !semanticallyValid(document, known) {
			observation.semanticFalsePositives++
			observation.falsePositives++
			continue
		}
		distance, measured := distances[id]
		if !measured {
			observation.semanticFalsePositives++
			observation.falsePositives++
			continue
		}
		if distance < clearThreshold {
			directClearOrder = append(directClearOrder, id)
		}
		if _, expected := expectedSet[id]; expected {
			continue
		}
		observation.falsePositives++
		outside := distance - query.Geo.RadiusMeters
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
	if sortSupported && len(expectedClearOrder) > 0 &&
		equalStringSets(expectedClearOrder, directClearOrder) &&
		!equalStringSlices(expectedClearOrder, directClearOrder) {
		observation.semanticFalsePositives++
		observation.semanticError = appendGeoJSONDirectError(
			observation.semanticError,
			"direct clear-zone ordering differs from the exhaustive ProductQuery order",
		)
	}

	sort.Slice(clearIDs, func(i, j int) bool {
		left, right := clearIDs[i], clearIDs[j]
		if distances[left] != distances[right] {
			return distances[left] < distances[right]
		}
		return left < right
	})
	if len(clearIDs) > 0 {
		observation.nearestExpected = 1
		if _, found := directSet[clearIDs[0]]; found {
			observation.nearestReturned = 1
		}
		observation.top10Expected = min(10, len(clearIDs))
		for _, id := range clearIDs[:observation.top10Expected] {
			if _, found := directSet[id]; found {
				observation.top10Returned++
			}
		}
		observation.falseEmpty = observation.clearTruePositives == 0
	}
	if sortSupported {
		expectedClearPage := make([]string, 0, len(expectedPage))
		for _, id := range expectedPage {
			if distance, measured := distances[id]; measured && distance < clearThreshold {
				expectedClearPage = append(expectedClearPage, id)
			}
		}
		expectedClearPageSet := stringSet(expectedClearPage)
		observation.comparablePageExpected = len(expectedClearPage)
		for _, id := range directPage {
			document, known := documentsByID[id]
			if !semanticallyValid(document, known) {
				observation.comparablePageActual++
				observation.comparablePageMaterialFP++
				continue
			}
			distance, measured := distances[id]
			if !measured {
				observation.comparablePageActual++
				observation.comparablePageMaterialFP++
				continue
			}
			switch {
			case distance < clearThreshold:
				observation.comparablePageActual++
				if _, expected := expectedClearPageSet[id]; expected {
					observation.comparablePageReturned++
				} else {
					observation.comparablePageMaterialFP++
				}
			case distance > outerThreshold:
				observation.comparablePageActual++
				observation.comparablePageMaterialFP++
			default:
				// Results in the configured boundary band are neutral for the
				// tolerant page-recall and page-precision metrics.
			}
		}
		observation.comparablePage = observation.comparablePageExpected > 0 || observation.comparablePageActual > 0
	}
	if observation.semanticFalsePositives > 0 && observation.semanticError == "" {
		observation.semanticError = "direct result violates ProductQuery text, ACL, filter, or ordering semantics"
	}

	observation.violation = GeoJSONDirectCaseViolation{
		QueryID:               benchmarkCase.QueryPoint.ID,
		Scenario:              benchmarkCase.QueryPoint.Scenario,
		RadiusMeters:          benchmarkCase.Radius,
		Expected:              observation.expected,
		Returned:              observation.returned,
		FalseNegatives:        observation.falseNegatives,
		MaterialFalseNegative: observation.materialFalseNegatives,
		FalsePositives:        observation.falsePositives,
		MaterialFalsePositive: observation.materialFalsePositives,
		Recall:                ratioOrOne(observation.truePositives, observation.expected),
		MaxInsideMissMeters:   observation.maxInsideMiss,
		MaxOutsideHitMeters:   observation.maxOutsideHit,
		MaterialFalseNegativeTrails: append([]GeoJSONDirectTrailViolationDetail(nil),
			observation.materialFalseNegativeIDs...),
		MaterialFalsePositiveTrails: append([]GeoJSONDirectTrailViolationDetail(nil),
			observation.materialFalsePositiveIDs...),
		FalseEmpty: observation.falseEmpty,
		Error:      observation.semanticError,
	}
	return observation
}

func equalStringSets(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	leftSet := stringSet(left)
	if len(leftSet) != len(left) {
		return false
	}
	for _, value := range right {
		if _, found := leftSet[value]; !found {
			return false
		}
	}
	return true
}

func appendGeoJSONDirectError(existing, message string) string {
	if existing == "" {
		return message
	}
	return existing + "; " + message
}

func pageOfIDs(ids []string, page, perPage int) []string {
	start := productPageStart(page, perPage, len(ids))
	if start >= len(ids) {
		return nil
	}
	end := min(start+perPage, len(ids))
	return append([]string(nil), ids[start:end]...)
}

func overlapCount(left, right []string) int {
	rightSet := stringSet(right)
	count := 0
	for _, id := range left {
		if _, found := rightSet[id]; found {
			count++
		}
	}
	return count
}

func stringSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}
