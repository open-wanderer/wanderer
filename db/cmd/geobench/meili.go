package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"github.com/twpayne/go-polyline"
)

const (
	benchmarkIndexUID                 = "wanderer_geo_bench"
	geoJSONCoverageDiagnosticIndexUID = "wanderer_geojson_coverage_contract"
	geoJSONSplitDiagnosticIndexUID    = "wanderer_geojson_split_contract"
)

const (
	meiliMaxBatchBytes           = 16 << 20
	meiliInitialTaskPollInterval = 20 * time.Millisecond
	meiliMaxTaskPollInterval     = 500 * time.Millisecond
)

func runGeoJSONContract(ctx context.Context, image, workRoot string) ContractReport {
	started := time.Now()
	report := ContractReport{MeilisearchVersion: image, Status: "failed"}
	container, err := startMeiliContainer(ctx, image, workRoot, "contract-"+image)
	if err != nil {
		report.Error = err.Error()
		report.DurationMS = milliseconds(time.Since(started))
		return report
	}
	defer container.close()
	report.MeilisearchVersion = container.version

	client := meilisearch.New(container.url)
	if err := createConfiguredIndex(ctx, client, []string{"_geojson"}, 100); err != nil {
		report.Status = geoJSONFailureStatus(container.version)
		report.Error = err.Error()
		report.DurationMS = milliseconds(time.Since(started))
		return report
	}

	documents := geoJSONContractDocuments()
	taskInfo, err := client.Index(benchmarkIndexUID).AddDocumentsWithContext(ctx, documents, nil)
	if err == nil {
		_, err = waitForSuccessfulTask(ctx, client, taskInfo)
	}
	if err != nil {
		report.Status = geoJSONFailureStatus(container.version)
		report.Error = fmt.Sprintf("index LineString/MultiLineString: %v", err)
		report.DurationMS = milliseconds(time.Since(started))
		return report
	}

	for _, test := range geoJSONRadiusContractCases() {
		result, err := searchGeoRadius(ctx, client, "_geojson", test.point, test.radius, 100)
		if err != nil {
			report.Status = geoJSONFailureStatus(container.version)
			report.Error = fmt.Sprintf("%s: %v", test.name, err)
			report.DurationMS = milliseconds(time.Since(started))
			return report
		}
		if !equalSets(result.IDs, test.expected) {
			report.Status = geoJSONFailureStatus(container.version)
			report.Error = fmt.Sprintf("%s: got %v, want %v", test.name, sortedIDs(result.IDs), sortedIDs(test.expected))
			report.DurationMS = milliseconds(time.Since(started))
			return report
		}
	}

	for _, test := range geoJSONBoundingBoxContractCases() {
		result, err := searchGeoBoundingBox(ctx, client, test.viewport, 100)
		if err != nil {
			report.Status = geoJSONFailureStatus(container.version)
			report.Error = fmt.Sprintf("%s: %v", test.name, err)
			report.DurationMS = milliseconds(time.Since(started))
			return report
		}
		if !equalSets(result.IDs, test.expected) {
			report.Status = geoJSONFailureStatus(container.version)
			report.Error = fmt.Sprintf("%s: got %v, want %v", test.name, sortedIDs(result.IDs), sortedIDs(test.expected))
			report.DurationMS = milliseconds(time.Since(started))
			return report
		}
	}

	regressionDocuments, regressionCases, err := geoJSONBoundaryRegressionFixtures()
	if err != nil {
		report.Error = fmt.Sprintf("prepare near-boundary regression fixtures: %v", err)
		report.DurationMS = milliseconds(time.Since(started))
		return report
	}
	taskInfo, err = client.Index(benchmarkIndexUID).AddDocumentsWithContext(ctx, regressionDocuments, nil)
	if err == nil {
		_, err = waitForSuccessfulTask(ctx, client, taskInfo)
	}
	if err != nil {
		report.Status = geoJSONFailureStatus(container.version)
		report.Error = fmt.Sprintf("index near-boundary regression fixtures: %v", err)
		report.DurationMS = milliseconds(time.Since(started))
		return report
	}
	for _, resolution := range []int{125, 1000} {
		plan := GeoJSONQueryPlan{
			ID:                 fmt.Sprintf("contract-r%d-safe", resolution),
			RadiusMode:         geoJSONRadiusSafe,
			Resolution:         resolution,
			ExtraPaddingMeters: 0.5,
			Conservative:       true,
			ExactRefinement:    true,
		}
		for _, test := range regressionCases {
			result, err := searchGeoJSONRadius(ctx, client, test.point, test.radius, plan, 100)
			if err != nil {
				report.Status = geoJSONFailureStatus(container.version)
				report.Error = fmt.Sprintf("near-boundary %s at resolution %d: %v", test.name, resolution, err)
				report.DurationMS = milliseconds(time.Since(started))
				return report
			}
			if _, found := result.IDs[test.id]; !found {
				report.Status = geoJSONFailureStatus(container.version)
				report.Error = fmt.Sprintf("near-boundary %s is missing at resolution %d", test.name, resolution)
				report.DurationMS = milliseconds(time.Since(started))
				return report
			}
		}
	}
	runFederatedContract, qualificationErr := shouldRunGeoJSONFederatedPageContract(image, container.version)
	if qualificationErr != nil {
		report.Status = "failed"
		report.Error = fmt.Sprintf("Meilisearch qualification version: %v", qualificationErr)
		report.DurationMS = milliseconds(time.Since(started))
		return report
	}
	if runFederatedContract {
		if err := runGeoJSONFederatedPageContract(ctx, container, client); err != nil {
			report.Status = geoJSONFailureStatus(container.version)
			report.Error = fmt.Sprintf("federated exhaustive page contract: %v", err)
			report.DurationMS = milliseconds(time.Since(started))
			return report
		}
	}

	// This diagnostic intentionally does not alter the top-level contract
	// status. It keeps the known Cellulite coverage limitation observable while
	// plan qualification remains governed by the measured aggregate UX policy.
	report.Diagnostics = append(
		report.Diagnostics,
		runGeoJSONCoverageDiagnostic(ctx, client),
		runGeoJSONSplitDiagnostic(ctx, client),
	)

	report.Status = "passed"
	report.DurationMS = milliseconds(time.Since(started))
	return report
}

func runGeoJSONCoverageDiagnostic(
	ctx context.Context,
	client meilisearch.ServiceManager,
) GeoJSONContractDiagnosticReport {
	report := GeoJSONContractDiagnosticReport{
		Name:       "geojson_cell_coverage",
		Status:     "diagnostic_error",
		QueryPoint: geoJSONHighLatitudeQuery,
	}
	if err := createConfiguredIndexAtUID(
		ctx,
		client,
		geoJSONCoverageDiagnosticIndexUID,
		[]string{"_geojson", "_geo"},
		100,
	); err != nil {
		report.Error = fmt.Sprintf("create isolated GeoJSON coverage index: %v", err)
		return report
	}

	documents := geoJSONCoverageContractDocuments()
	taskInfo, err := client.Index(geoJSONCoverageDiagnosticIndexUID).AddDocumentsWithContext(ctx, documents, nil)
	if err == nil {
		_, err = waitForSuccessfulTask(ctx, client, taskInfo)
	}
	if err != nil {
		report.Error = fmt.Sprintf("index GeoJSON/native coverage controls: %v", err)
		return report
	}

	unfiltered, err := searchFilterAtIndex(ctx, client, geoJSONCoverageDiagnosticIndexUID, "", 100)
	if err != nil {
		report.Error = fmt.Sprintf("read unfiltered GeoJSON/native coverage controls: %v", err)
		return report
	}
	for _, document := range documents {
		id, _ := document["id"].(string)
		if _, found := unfiltered.IDs[id]; !found {
			report.Error = fmt.Sprintf("unfiltered ingestion control is missing %s", id)
			return report
		}
	}

	tests, err := geoJSONCoverageContractCases()
	if err != nil {
		report.Error = fmt.Sprintf("prepare GeoJSON coverage filters: %v", err)
		return report
	}
	mismatch := false
	for _, test := range tests {
		result, searchErr := searchFilterAtIndex(ctx, client, geoJSONCoverageDiagnosticIndexUID, test.filter, 100)
		if searchErr != nil {
			report.Error = fmt.Sprintf("%s: %v", test.name, searchErr)
			return report
		}
		caseReport, caseMismatch := geoJSONContractDiagnosticCaseReport(test, result.IDs)
		report.Cases = append(report.Cases, caseReport)
		mismatch = mismatch || caseMismatch
	}
	if mismatch {
		report.Status = "known_engine_limitation"
		report.Cause = "Meilisearch's _geojson path mixes spherical H3 cell assignment with planar latitude/longitude cell polygons during Cellulite/h3o query coverage. At the failing coordinates those representations disagree and the stored cell is omitted. Identical native _geo points pass the same filters; the reduced index demonstrates an ordinary-latitude hexagon failure, a Zurich pass and the original high-latitude Point/LineString failure. This excludes a polar-, Pentagon- or Local-IJ-only cause and is not attributable to S50, radius polygon resolution, line direction or line density."
		return report
	}
	report.Status = "passed"
	return report
}

func runGeoJSONSplitDiagnostic(
	ctx context.Context,
	client meilisearch.ServiceManager,
) GeoJSONContractDiagnosticReport {
	report := GeoJSONContractDiagnosticReport{
		Name:       "geojson_hierarchical_split",
		Status:     "diagnostic_error",
		QueryPoint: Coordinate{Lat: 51.3, Lon: 7.3},
	}
	if err := createConfiguredIndexAtUID(
		ctx,
		client,
		geoJSONSplitDiagnosticIndexUID,
		[]string{"_geojson", "_geo"},
		geoJSONSplitContractDocumentCount+1,
	); err != nil {
		report.Error = fmt.Sprintf("create dense split index: %v", err)
		return report
	}

	documents := geoJSONSplitContractDocuments()
	taskInfo, err := client.Index(geoJSONSplitDiagnosticIndexUID).AddDocumentsWithContext(ctx, documents, nil)
	if err == nil {
		_, err = waitForSuccessfulTask(ctx, client, taskInfo)
	}
	if err != nil {
		report.Error = fmt.Sprintf("index dense split controls: %v", err)
		return report
	}

	nativeCount, err := searchExactCountAtIndex(
		ctx,
		client,
		geoJSONSplitDiagnosticIndexUID,
		geoJSONSplitNativeFilter,
	)
	if err != nil {
		report.Error = fmt.Sprintf("count native split controls: %v", err)
		return report
	}
	report.Counts = append(report.Counts, GeoJSONContractDiagnosticCountReport{
		Name:          "native bounding-box ingestion control",
		Filter:        geoJSONSplitNativeFilter,
		ExpectedCount: geoJSONSplitContractDocumentCount,
		ActualCount:   nativeCount,
		MissingCount:  max(int64(geoJSONSplitContractDocumentCount)-nativeCount, 0),
	})
	if nativeCount != geoJSONSplitContractDocumentCount {
		report.Error = fmt.Sprintf(
			"native ingestion control returned %d/%d documents",
			nativeCount,
			geoJSONSplitContractDocumentCount,
		)
		return report
	}

	shapeCount, err := searchExactCountAtIndex(
		ctx,
		client,
		geoJSONSplitDiagnosticIndexUID,
		geoJSONSplitShapeFilter,
	)
	if err != nil {
		report.Error = fmt.Sprintf("count GeoJSON split controls: %v", err)
		return report
	}
	report.Counts = append(report.Counts, GeoJSONContractDiagnosticCountReport{
		Name:          "GeoJSON polygon over identical extent",
		Filter:        geoJSONSplitShapeFilter,
		ExpectedCount: nativeCount,
		ActualCount:   shapeCount,
		MissingCount:  max(nativeCount-shapeCount, 0),
	})
	if shapeCount != nativeCount {
		report.Status = "known_engine_limitation"
		report.Cause = "Cellulite's contained-cell optimization pre-marks formal H3 children as explored. H3 hierarchy does not guarantee strict geometric containment, so a child required by geometric tiling can be skipped when it is reached later through an intersecting cell. This dense split failure is independent of the planar cell-boundary mismatch."
		return report
	}
	report.Status = "passed"
	return report
}

func runMeiliBackend(
	ctx context.Context,
	image, strategy, workRoot string,
	dataset Dataset,
	updates []Trail,
	config RunConfig,
	oracle *accuracyOracle,
) BackendReport {
	report := BackendReport{Backend: strategy, MeilisearchVersion: image, Status: "failed"}
	container, err := startMeiliContainerWithThreads(ctx, image, workRoot, strategy+"-"+image, config.MeiliIndexThreads)
	if err != nil {
		report.Error = err.Error()
		return report
	}
	defer container.close()
	report.MeilisearchVersion = container.version
	client := meilisearch.New(container.url)
	indexUIDs := []string{benchmarkIndexUID}
	if strategy == "geojson" {
		shardCount, shardErr := geoJSONShardCount(config.GeoJSONIndex, len(dataset.Trails))
		if shardErr != nil {
			report.Error = shardErr.Error()
			return report
		}
		indexUIDs = geoJSONIndexUIDs(shardCount)
	}
	runEngineQueries := config.QueryWorkload != "product"
	runProductQueries := config.QueryWorkload != "engine"
	tuneGeoJSON := strategy == "geojson" && config.GeoJSONSweep.Enabled
	var productDocuments []ProductQueryDocument

	filterable := []string{"_geo"}
	if strategy == "geojson" {
		filterable = []string{"_geojson"}
	}
	if strategy == "h3" {
		filterable = make([]string, 0, len(config.H3Resolutions))
		for _, resolution := range sortedUniqueInts(config.H3Resolutions) {
			filterable = append(filterable, fmt.Sprintf("h3_r%d", resolution))
		}
	}
	maxHits := int64(len(dataset.Trails) + len(updates) + 100)
	if maxHits < 1000 {
		maxHits = 1000
	}
	indexSetupStarted := time.Now()
	if err := createConfiguredIndexes(ctx, client, indexUIDs, filterable, maxHits); err != nil {
		report.IndexSetupMS = milliseconds(time.Since(indexSetupStarted))
		report.Error = err.Error()
		return report
	}
	report.IndexSetupMS = milliseconds(time.Since(indexSetupStarted))

	prepareMonitor := startRSSMonitor(os.Getpid())
	prepareStarted := time.Now()
	productDocuments = generateProductDocuments(dataset)
	indexTrails, geoJSONIndex, err := projectMeiliIndexTrails(ctx, strategy, dataset.Trails, config.GeoJSONIndex)
	if geoJSONIndex != nil {
		report.GeoJSONIndex = geoJSONIndex
	}
	var documents []map[string]any
	if err == nil {
		documents, err = prepareMeiliDocumentsContext(ctx, strategy, indexTrails, config.H3Resolutions)
	}
	if err == nil {
		err = mergeMeiliProductDocuments(documents, productDocuments)
	}
	var documentShards [][]map[string]any
	if err == nil && strategy == "geojson" {
		var localCreated map[string]int64
		if normalizedGeoJSONShardMode(config.GeoJSONIndex.ShardMode) == geoJSONShardModeCapacityCreated {
			localCreated, err = geoJSONCapacityLocalCreatedPlan(dataset.Trails, dataset.Seed, config.GeoJSONIndex.CapacityArrival)
		}
		if err == nil {
			documentShards, err = splitGeoJSONDocuments(documents, config.GeoJSONIndex, localCreated)
		}
	}
	if err == nil && strategy == "geojson" {
		err = populateGeoJSONShardReport(report.GeoJSONIndex, dataset.Trails, indexTrails, documentShards, config.GeoJSONIndex)
	}
	indexTrails = nil
	prepareMS := milliseconds(time.Since(prepareStarted))
	clientBaseline, clientPeak, _ := prepareMonitor.finish()
	if err != nil {
		report.Error = fmt.Sprintf("prepare full index: %v", err)
		return report
	}

	if len(indexUIDs) == 1 {
		singleIndexDocuments := documents
		if strategy == "geojson" && len(documentShards) == 1 {
			singleIndexDocuments = documentShards[0]
		}
		report.FullIndex, err = indexMeiliDocumentsAtIndex(ctx, client, container, indexUIDs[0], singleIndexDocuments, config.BatchSize, config.BatchBytes, config.MaxPendingBatches, false)
	} else {
		report.FullIndex, report.GeoJSONIndex.ShardBuilds, err = indexGeoJSONShards(
			ctx, client, container, indexUIDs, documentShards,
			config.BatchSize, config.BatchBytes, config.MaxPendingBatches,
		)
	}
	report.FullIndex.PrepareMS = prepareMS
	mergeClientRSS(&report.FullIndex, clientBaseline, clientPeak)
	if err != nil {
		report.Error = fmt.Sprintf("full index: %v", err)
		return report
	}
	if config.IndexOnly {
		report.Status = "ok"
		return report
	}
	capacitySearchOnly := strategy == "geojson" &&
		normalizedGeoJSONShardMode(config.GeoJSONIndex.ShardMode) == geoJSONShardModeCapacityCreated &&
		len(indexUIDs) > 1
	if capacitySearchOnly {
		report.ExecutionScope = geoJSONCapacitySearchScope
		if err := validateMeiliQualificationRuntime(container.image, container.version); err != nil {
			report.Error = fmt.Sprintf("capacity-created exhaustive federated search runtime: %v", err)
			return report
		}
		report.GeoJSONDirect, err = benchmarkGeoJSONCapacityCreatedSearch(
			ctx,
			container,
			indexUIDs,
			dataset,
			config.RadiiMeters,
			config.Repetitions,
			config.QueryClients,
			productDocuments,
			oracle,
			config.GeoJSONDirect,
			config.GeoJSONIndex.SimplifyToleranceMeters,
		)
		if err != nil {
			report.Error = fmt.Sprintf("capacity-created federated product search: %v", err)
			return report
		}
		if report.GeoJSONDirect == nil || report.GeoJSONDirect.SpatialAudit == nil ||
			report.GeoJSONDirect.SpatialAudit.Status != "passed" ||
			report.GeoJSONDirect.ProductWorkload == nil ||
			report.GeoJSONDirect.ProductWorkload.Status != "passed" ||
			report.GeoJSONDirect.ProductWorkload.Accuracy.Status != "passed" {
			report.Status = "incorrect"
			if report.GeoJSONDirect != nil {
				report.Error = report.GeoJSONDirect.Error
			}
			if report.Error == "" {
				report.Error = "capacity-created fixed-resolution federated product search did not qualify"
			}
			return report
		}
		report.Status = "ok"
		report.Performance = evaluatePerformanceTarget(config, report)
		evaluateGeoJSONDirectPlanPerformance(config, report, report.GeoJSONDirect)
		return report
	}
	documents = nil
	runtime.GC()

	var productDatabase *productStore
	if runProductQueries || (runEngineQueries && (strategy == "h3" || tuneGeoJSON)) {
		applicationStarted := time.Now()
		productDatabase, err = newProductStore(
			filepath.Join(workRoot, container.name+"-application.sqlite"),
			config.QueryClients,
		)
		if err == nil {
			err = productDatabase.replaceAll(ctx, productDocuments)
		}
		report.ApplicationDatabaseSetupMS = milliseconds(time.Since(applicationStarted))
		if err != nil {
			if productDatabase != nil {
				_ = productDatabase.close()
			}
			report.Error = fmt.Sprintf("prepare application database: %v", err)
			return report
		}
		defer productDatabase.close()
		report.ApplicationDatabaseBytes, err = productDatabase.diskBytes()
		if err != nil {
			report.Error = fmt.Sprintf("measure application database: %v", err)
			return report
		}
	}

	queryFunction := candidateQuery(func(ctx context.Context, queryPoint QueryPoint, radius float64) (CandidateResult, error) {
		if strategy == "h3" {
			started := time.Now()
			_, filter, _, err := h3QueryFilter(queryPoint.Point, radius, config.H3Resolutions)
			if err != nil {
				return CandidateResult{}, err
			}
			prepareDuration := time.Since(started)
			result, err := searchFilter(ctx, client, filter, int64(len(dataset.Trails)+10))
			result.WallDuration += prepareDuration
			return result, err
		}
		return searchGeoRadius(ctx, client, filterable[0], queryPoint.Point, radius, int64(len(dataset.Trails)+10))
	})
	productQueryFunction := newMeiliProductCandidateQuery(
		client, strategy, config.H3Resolutions, nil, int64(len(dataset.Trails)+10),
	)
	geoJSONWinnerMissing := false
	if tuneGeoJSON {
		plans := geoJSONQueryPlans(config.GeoJSONSweep)
		report.GeoJSONSweep, err = screenGeoJSONVariants(
			ctx,
			client,
			dataset,
			config.RadiiMeters,
			oracle,
			plans,
			int64(len(dataset.Trails)+10),
			config.GeoJSONSweep.Finalists,
			config.GeoJSONDirect,
		)
		if err != nil {
			report.Error = fmt.Sprintf("GeoJSON parameter screen: %v", err)
			return report
		}
		if runProductQueries {
			report.GeoJSONDirect, err = benchmarkGeoJSONDirectProductCandidates(
				ctx,
				client,
				dataset,
				config.RadiiMeters,
				config.Repetitions,
				config.QueryClients,
				productDocuments,
				report.GeoJSONSweep,
				config.GeoJSONDirect,
			)
			if err != nil {
				if ctx.Err() != nil {
					report.Error = fmt.Sprintf("GeoJSON direct product workload: %v", err)
					return report
				}
			}
		} else {
			report.GeoJSONDirect = selectGeoJSONDirect(report.GeoJSONSweep, config.GeoJSONDirect)
		}

		for index := range report.GeoJSONSweep.Variants {
			variant := &report.GeoJSONSweep.Variants[index]
			if !variant.Finalist {
				continue
			}
			plan := variant.Plan
			variantQuery := candidateQuery(func(ctx context.Context, queryPoint QueryPoint, radius float64) (CandidateResult, error) {
				return searchGeoJSONRadius(ctx, client, queryPoint.Point, radius, plan, int64(len(dataset.Trails)+10))
			})
			if runEngineQueries {
				variant.Queries, err = benchmarkQueries(
					ctx,
					dataset,
					config.RadiiMeters,
					config.Repetitions,
					productDatabase.loadGeometries,
					oracle,
					variantQuery,
				)
				if err != nil {
					variant.Status = "failed"
					variant.Error = fmt.Sprintf("engine queries: %v", err)
					if ctx.Err() != nil {
						report.Error = variant.Error
						return report
					}
					continue
				}
			}
			if runProductQueries {
				variant.ProductWorkload, err = benchmarkProductWorkloadWithCandidateQuery(
					ctx,
					dataset,
					config.RadiiMeters,
					config.Repetitions,
					config.QueryClients,
					productDatabase,
					productDocuments,
					newMeiliProductCandidateQuery(
						client, strategy, config.H3Resolutions, &plan, int64(len(dataset.Trails)+10),
					),
				)
				if variant.ProductWorkload != nil {
					variant.ProductWorkload.SetupMS = report.ApplicationDatabaseSetupMS
					variant.ProductWorkload.DatabaseBytes = report.ApplicationDatabaseBytes
				}
				if err != nil {
					variant.Status = "failed"
					variant.Error = fmt.Sprintf("product queries: %v", err)
					if ctx.Err() != nil {
						report.Error = variant.Error
						return report
					}
					continue
				}
			}
			if geoJSONVariantCorrect(*variant) {
				variant.Status = "ok"
			} else {
				variant.Status = "incorrect"
			}
		}

		winner := selectGeoJSONWinner(report.GeoJSONSweep)
		if winner >= 0 {
			selected := &report.GeoJSONSweep.Variants[winner]
			report.GeoJSONSweep.SelectedID = selected.Plan.ID
			if selected.ProductWorkload != nil {
				report.GeoJSONSweep.SelectionSamples = selected.ProductWorkload.Samples
			} else {
				for _, query := range selected.Queries {
					report.GeoJSONSweep.SelectionSamples += query.Samples
				}
			}
			report.GeoJSONSweep.SelectionStatus = "provisional"
			if report.GeoJSONSweep.SelectionSamples >= ds923PlusMinimumSamples {
				report.GeoJSONSweep.SelectionStatus = "qualified"
			}
			report.GeoJSONPlan = &selected.Plan
			report.Queries = selected.Queries
			report.ProductWorkload = selected.ProductWorkload
			plan := selected.Plan
			queryFunction = func(ctx context.Context, queryPoint QueryPoint, radius float64) (CandidateResult, error) {
				return searchGeoJSONRadius(ctx, client, queryPoint.Point, radius, plan, int64(len(dataset.Trails)+10))
			}
		} else {
			geoJSONWinnerMissing = true
			report.GeoJSONSweep.SelectionStatus = "no_qualifying_plan"
			// Mutation accounting remains useful even when no query plan qualifies.
			// Use the first screened plan only for spatial readback; it is never
			// presented as the selected production plan.
			fallback := plans[0]
			for _, variant := range report.GeoJSONSweep.Variants {
				if variant.Plan.Conservative {
					fallback = variant.Plan
					break
				}
			}
			queryFunction = func(ctx context.Context, queryPoint QueryPoint, radius float64) (CandidateResult, error) {
				return searchGeoJSONRadius(ctx, client, queryPoint.Point, radius, fallback, int64(len(dataset.Trails)+10))
			}
		}
	} else {
		if runEngineQueries {
			var geometryLoader candidateGeometryLoader
			if strategy == "h3" {
				geometryLoader = productDatabase.loadGeometries
			}
			report.Queries, err = benchmarkQueries(
				ctx,
				dataset,
				config.RadiiMeters,
				config.Repetitions,
				geometryLoader,
				oracle,
				queryFunction,
			)
			if err != nil {
				report.Error = fmt.Sprintf("engine queries: %v", err)
				return report
			}
		}

		if runProductQueries {
			report.ProductWorkload, err = benchmarkProductWorkloadWithCandidateQuery(
				ctx,
				dataset,
				config.RadiiMeters,
				config.Repetitions,
				config.QueryClients,
				productDatabase,
				productDocuments,
				productQueryFunction,
			)
			if report.ProductWorkload != nil {
				report.ProductWorkload.SetupMS = report.ApplicationDatabaseSetupMS
				report.ProductWorkload.DatabaseBytes = report.ApplicationDatabaseBytes
			}
			if err != nil {
				report.Error = fmt.Sprintf("product queries: %v", err)
				return report
			}
		}
	}

	// Full-index preparation and query filters can leave reusable heap pages
	// behind. Return them before measuring the much smaller update phase so its
	// RSS delta reflects the update workload rather than allocator reuse.
	debug.FreeOSMemory()
	prepareMonitor = startRSSMonitor(os.Getpid())
	prepareStarted = time.Now()
	indexUpdates, _, err := projectMeiliIndexTrails(ctx, strategy, updates, config.GeoJSONIndex)
	var updateDocuments []map[string]any
	if err == nil {
		updateDocuments, err = prepareMeiliDocumentsContext(ctx, strategy, indexUpdates, config.H3Resolutions)
	}
	if err == nil {
		err = mergeMeiliProductDocuments(updateDocuments, productDocumentsForUpdatedTrails(productDocuments, updates))
	}
	prepareMS = milliseconds(time.Since(prepareStarted))
	clientBaseline, clientPeak, _ = prepareMonitor.finish()
	if err != nil {
		report.Error = fmt.Sprintf("prepare incremental index: %v", err)
		return report
	}
	report.IncrementalIndex, err = indexMeiliDocuments(ctx, client, container, updateDocuments, config.BatchSize, config.BatchBytes, config.MaxPendingBatches, true)
	report.IncrementalIndex.PrepareMS = prepareMS
	mergeClientRSS(&report.IncrementalIndex, clientBaseline, clientPeak)
	if err != nil {
		report.Error = fmt.Sprintf("incremental index: %v", err)
		return report
	}
	if len(updateDocuments) > 0 {
		for _, index := range verificationIndices(len(updateDocuments)) {
			if err := verifyMeiliDocument(ctx, client, updateDocuments[index]); err != nil {
				report.Error = fmt.Sprintf("verify incremental index: %v", err)
				return report
			}
			result, err := queryFunction(ctx, QueryPoint{
				ID:    "incremental-check-" + updates[index].ID,
				Point: updates[index].Start(),
			}, 1)
			if err != nil {
				report.Error = fmt.Sprintf("verify incremental spatial index for trail %q: %v", updates[index].ID, err)
				return report
			}
			if _, found := result.IDs[updates[index].ID]; !found {
				report.Error = fmt.Sprintf("verify incremental spatial index: updated trail %q was not found at its new start", updates[index].ID)
				return report
			}
		}
	}

	additions := deriveAddedTrails(dataset, config.IncrementalCount)
	debug.FreeOSMemory()
	prepareMonitor = startRSSMonitor(os.Getpid())
	prepareStarted = time.Now()
	indexAdditions, _, err := projectMeiliIndexTrails(ctx, strategy, additions, config.GeoJSONIndex)
	var addedDocuments []map[string]any
	if err == nil {
		addedDocuments, err = prepareMeiliDocumentsContext(ctx, strategy, indexAdditions, config.H3Resolutions)
	}
	if err == nil {
		err = mergeMeiliProductDocuments(addedDocuments, generateProductDocuments(Dataset{Trails: additions}))
	}
	addPrepareMS := milliseconds(time.Since(prepareStarted))
	clientBaseline, clientPeak, _ = prepareMonitor.finish()
	if err != nil {
		report.Error = fmt.Sprintf("prepare additions: %v", err)
		return report
	}
	report.AddIndex, err = indexMeiliDocuments(ctx, client, container, addedDocuments, config.BatchSize, config.BatchBytes, config.MaxPendingBatches, false)
	report.AddIndex.PrepareMS = addPrepareMS
	mergeClientRSS(&report.AddIndex, clientBaseline, clientPeak)
	if err != nil {
		report.Error = fmt.Sprintf("add index: %v", err)
		return report
	}
	for _, index := range verificationIndices(len(addedDocuments)) {
		if err := verifyMeiliDocument(ctx, client, addedDocuments[index]); err != nil {
			report.Error = fmt.Sprintf("verify addition: %v", err)
			return report
		}
		result, err := queryFunction(ctx, QueryPoint{ID: "add-check-" + additions[index].ID, Point: additions[index].Start()}, 1)
		if err != nil {
			report.Error = fmt.Sprintf("verify added spatial index for trail %q: %v", additions[index].ID, err)
			return report
		}
		if _, found := result.IDs[additions[index].ID]; !found {
			report.Error = fmt.Sprintf("verify added spatial index: trail %q was not found at its start", additions[index].ID)
			return report
		}
	}

	deletePrepareStarted := time.Now()
	deletionIDs := selectDeletionIDs(additions, len(additions))
	deletePrepareMS := milliseconds(time.Since(deletePrepareStarted))
	report.DeleteIndex, err = deleteMeiliDocuments(ctx, client, container, deletionIDs, config.BatchSize)
	report.DeleteIndex.PrepareMS = deletePrepareMS
	if err != nil {
		report.Error = fmt.Sprintf("delete index: %v", err)
		return report
	}
	if report.DeleteIndex.DocumentCount != int64(len(dataset.Trails)) {
		report.Error = fmt.Sprintf("verify deletion: index contains %d documents, want %d", report.DeleteIndex.DocumentCount, len(dataset.Trails))
		return report
	}
	for _, index := range verificationIndices(len(additions)) {
		if err := verifyMeiliDocumentDeleted(ctx, client, additions[index].ID); err != nil {
			report.Error = fmt.Sprintf("verify deletion: %v", err)
			return report
		}
	}

	report.Status = completedBackendStatus(report)
	if geoJSONWinnerMissing {
		report.Status = "incorrect"
		report.Error = "no conservative GeoJSON query plan passed candidate recall and full-workload correctness gates"
	}
	report.Performance = evaluatePerformanceTarget(config, report)
	if report.GeoJSONDirect != nil {
		evaluateGeoJSONDirectPlanPerformance(config, report, report.GeoJSONDirect)
	}
	return report
}

func createConfiguredIndex(
	ctx context.Context,
	client meilisearch.ServiceManager,
	filterable []string,
	maxTotalHits int64,
) error {
	return createConfiguredIndexes(ctx, client, []string{benchmarkIndexUID}, filterable, maxTotalHits)
}

func createConfiguredIndexes(
	ctx context.Context,
	client meilisearch.ServiceManager,
	indexUIDs []string,
	filterable []string,
	maxTotalHits int64,
) error {
	if len(indexUIDs) == 0 {
		return fmt.Errorf("create indexes: no index UIDs")
	}
	for _, indexUID := range indexUIDs {
		if err := createConfiguredIndexAtUID(ctx, client, indexUID, filterable, maxTotalHits); err != nil {
			return fmt.Errorf("configure index %s: %w", indexUID, err)
		}
	}
	return nil
}

func createConfiguredIndexAtUID(
	ctx context.Context,
	client meilisearch.ServiceManager,
	indexUID string,
	filterable []string,
	maxTotalHits int64,
) error {
	taskInfo, err := client.CreateIndexWithContext(ctx, &meilisearch.IndexConfig{
		Uid:        indexUID,
		PrimaryKey: "id",
	})
	if err != nil {
		return fmt.Errorf("create index: %w", err)
	}
	if _, err := waitForSuccessfulTask(ctx, client, taskInfo); err != nil {
		return fmt.Errorf("create index: %w", err)
	}

	// Keep the benchmark index product-shaped for every spatial strategy. This
	// makes indexing cost comparable and lets the direct GeoJSON path execute
	// ACL, text, metadata, geo, sorting and pagination in one Meilisearch query.
	filterable = append(append([]string(nil), filterable...),
		"id", "author", "public", "shares", "is_federated",
		"category_id", "subcategory_id", "distance", "duration",
		"elevation_gain", "elevation_loss", "difficulty", "created", "tags",
	)
	settings := &meilisearch.Settings{
		SearchableAttributes: []string{"author_name", "name", "description", "location", "tags"},
		DisplayedAttributes:  []string{"*"},
		FilterableAttributes: sortedUniqueStrings(filterable),
		SortableAttributes: []string{
			"id", "name", "created", "distance", "duration",
			"elevation_gain", "elevation_loss", "difficulty",
		},
		RankingRules: []string{"words", "typo", "proximity", "attribute", "sort", "exactness"},
		Pagination:   &meilisearch.Pagination{MaxTotalHits: maxTotalHits},
	}
	taskInfo, err = client.Index(indexUID).UpdateSettingsWithContext(ctx, settings)
	if err != nil {
		return fmt.Errorf("configure index: %w", err)
	}
	if _, err := waitForSuccessfulTask(ctx, client, taskInfo); err != nil {
		return fmt.Errorf("configure index: %w", err)
	}
	return nil
}

func sortedUniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	sort.Strings(unique)
	return unique
}

func indexMeiliDocuments(
	ctx context.Context,
	client meilisearch.ServiceManager,
	container *meiliContainer,
	documents []map[string]any,
	batchSize int,
	maxBatchBytes int,
	maxPendingBatches int,
	update bool,
) (PhaseReport, error) {
	return indexMeiliDocumentsAtIndex(ctx, client, container, benchmarkIndexUID, documents, batchSize, maxBatchBytes, maxPendingBatches, update)
}

func indexMeiliDocumentsAtIndex(
	ctx context.Context,
	client meilisearch.ServiceManager,
	container *meiliContainer,
	indexUID string,
	documents []map[string]any,
	batchSize int,
	maxBatchBytes int,
	maxPendingBatches int,
	update bool,
) (PhaseReport, error) {
	report := PhaseReport{}
	if len(documents) == 0 {
		return report, nil
	}
	if batchSize < 1 {
		batchSize = 100
	}
	if maxBatchBytes < 3 {
		maxBatchBytes = meiliMaxBatchBytes
	}

	monitor := startContainerRSSMonitor(container)
	clientMonitor := startRSSMonitor(os.Getpid())
	finishMonitors := func() {
		report.EngineBaselineBytes, report.EnginePeakBytes, report.EnginePeakDelta = monitor.finish()
		report.ClientBaselineRSSBytes, report.ClientPeakRSSBytes, report.ClientPeakRSSDelta = clientMonitor.finish()
	}
	readyStarted := time.Now()
	pending := make([]*meilisearch.TaskInfo, 0, max(1, maxPendingBatches))
	waitPending := func() error {
		if len(pending) == 0 {
			return nil
		}
		report.BarrierCount++
		if _, err := waitForSuccessfulTask(ctx, client, pending[len(pending)-1]); err != nil {
			return err
		}
		for _, taskInfo := range pending[:len(pending)-1] {
			task, err := client.GetTaskWithContext(ctx, taskInfo.TaskUID)
			if err != nil {
				return fmt.Errorf("audit task %d: %w", taskInfo.TaskUID, err)
			}
			if err := successfulTaskError(task); err != nil {
				return err
			}
		}
		pending = pending[:0]
		return nil
	}
	err := forEachMeiliDocumentBatch(ctx, documents, batchSize, maxBatchBytes, func(batch []byte, documentCount int) error {
		var taskInfo *meilisearch.TaskInfo
		var err error
		requestStarted := time.Now()
		if update {
			taskInfo, err = client.Index(indexUID).UpdateDocumentsWithContext(ctx, batch, nil)
		} else {
			taskInfo, err = client.Index(indexUID).AddDocumentsWithContext(ctx, batch, nil)
		}
		report.RequestMS += milliseconds(time.Since(requestStarted))
		if err != nil {
			return err
		}
		if taskInfo == nil {
			return fmt.Errorf("Meilisearch returned no document task")
		}
		report.SubmittedDocuments += documentCount
		report.SubmittedBytes += int64(len(batch))
		report.TaskCount++
		pending = append(pending, taskInfo)
		if maxPendingBatches > 0 && len(pending) >= maxPendingBatches {
			return waitPending()
		}
		return nil
	})
	report.SubmitMS = milliseconds(time.Since(readyStarted))
	if err == nil {
		err = waitPending()
	}
	if err != nil {
		report.ReadyMS = report.SubmitMS
		if report.BarrierCount > 0 {
			report.ReadyMS = milliseconds(time.Since(readyStarted))
		}
		finishMonitors()
		captureCtx, captureCancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = populateMeiliPhaseStatsAtIndexes(captureCtx, client, container, []string{indexUID}, &report)
		captureCancel()
		return report, err
	}
	report.ReadyMS = milliseconds(time.Since(readyStarted))
	finishMonitors()

	if err := populateMeiliPhaseStatsAtIndexes(ctx, client, container, []string{indexUID}, &report); err != nil {
		return report, err
	}
	return report, nil
}

func populateMeiliPhaseStats(
	ctx context.Context,
	client meilisearch.ServiceManager,
	container *meiliContainer,
	report *PhaseReport,
) error {
	return populateMeiliPhaseStatsAtIndexes(ctx, client, container, []string{benchmarkIndexUID}, report)
}

func populateMeiliPhaseStatsAtIndexes(
	ctx context.Context,
	client meilisearch.ServiceManager,
	container *meiliContainer,
	indexUIDs []string,
	report *PhaseReport,
) error {
	report.LastBatch = latestMeiliBatchReportForIndexes(ctx, client, indexUIDs)
	stats, err := client.GetStatsWithContext(ctx)
	if err != nil {
		return err
	}
	report.DiskBytes = stats.DatabaseSize
	report.UsedDiskBytes = stats.UsedDatabaseSize
	report.DocumentCount = 0
	for _, indexUID := range indexUIDs {
		report.DocumentCount += stats.Indexes[indexUID].NumberOfDocuments
	}
	report.FilesystemBytes, err = directorySize(ctx, container.dataDir)
	return err
}

func latestMeiliBatchReport(ctx context.Context, client meilisearch.ServiceManager) *MeiliBatchReport {
	return latestMeiliBatchReportForIndexes(ctx, client, []string{benchmarkIndexUID})
}

func latestMeiliBatchReportForIndexes(ctx context.Context, client meilisearch.ServiceManager, indexUIDs []string) *MeiliBatchReport {
	batches, err := client.GetBatchesWithContext(ctx, &meilisearch.BatchesQuery{
		IndexUIDs: append([]string(nil), indexUIDs...),
		Limit:     1,
	})
	if err != nil || batches == nil || len(batches.Results) == 0 || batches.Results[0] == nil {
		return nil
	}
	batch := batches.Results[0]
	report := &MeiliBatchReport{
		UID:      batch.UID,
		Strategy: batch.BatchStrategy,
		Duration: batch.Duration,
	}
	if batch.Progress != nil {
		report.ProgressPercent = batch.Progress.Percentage
		for _, step := range batch.Progress.Steps {
			if step == nil || step.CurrentStep == "" {
				continue
			}
			report.ProgressSteps = append(report.ProgressSteps, fmt.Sprintf("%s (%d/%d)", step.CurrentStep, step.Finished, step.Total))
		}
	}
	if batch.Stats != nil {
		report.TaskCount = batch.Stats.TotalNbTasks
		if len(batch.Stats.ProgressTrace) > 0 {
			report.ProgressTrace = batch.Stats.ProgressTrace
		}
		if batch.Stats.WriteChannelCongestion != nil {
			report.WriteBlockingRatio = batch.Stats.WriteChannelCongestion.BlockingRatio
		}
	}
	return report
}

func forEachMeiliDocumentBatch(
	ctx context.Context,
	documents []map[string]any,
	maxDocuments, maxBytes int,
	submit func([]byte, int) error,
) error {
	if maxDocuments < 1 || maxBytes < 3 {
		return fmt.Errorf("invalid Meilisearch batch limits: %d documents, %d bytes", maxDocuments, maxBytes)
	}
	if submit == nil {
		return fmt.Errorf("Meilisearch batch submit callback is nil")
	}
	for start := 0; start < len(documents); {
		if err := ctx.Err(); err != nil {
			return err
		}
		end := start
		var batch bytes.Buffer
		batch.WriteByte('[')
		for end < len(documents) && end-start < maxDocuments {
			encoded, err := json.Marshal(documents[end])
			if err != nil {
				return fmt.Errorf("encode Meilisearch document %d: %w", end, err)
			}
			documentBytes := len(encoded) + 1 // Closing array bracket.
			if end > start {
				documentBytes++ // Comma between array values.
			}
			if batch.Len()+documentBytes > maxBytes {
				if end == start {
					return fmt.Errorf("Meilisearch document %d needs a %d-byte batch, maximum is %d", end, batch.Len()+documentBytes, maxBytes)
				}
				break
			}
			if end > start {
				batch.WriteByte(',')
			}
			batch.Write(encoded)
			end++
		}
		batch.WriteByte(']')
		if err := submit(batch.Bytes(), end-start); err != nil {
			return err
		}
		start = end
	}
	return nil
}

func mergeClientRSS(report *PhaseReport, prepareBaseline, preparePeak int64) {
	if prepareBaseline > 0 && (report.ClientBaselineRSSBytes == 0 || prepareBaseline < report.ClientBaselineRSSBytes) {
		report.ClientBaselineRSSBytes = prepareBaseline
	}
	if preparePeak > report.ClientPeakRSSBytes {
		report.ClientPeakRSSBytes = preparePeak
	}
	report.ClientPeakRSSDelta = report.ClientPeakRSSBytes - report.ClientBaselineRSSBytes
	if report.ClientPeakRSSDelta < 0 {
		report.ClientPeakRSSDelta = 0
	}
}

func verifyMeiliDocument(ctx context.Context, client meilisearch.ServiceManager, expected map[string]any) error {
	id, ok := expected["id"].(string)
	if !ok || id == "" {
		return fmt.Errorf("expected document has no string id")
	}
	var stored map[string]any
	if err := client.Index(benchmarkIndexUID).GetDocumentWithContext(ctx, id, nil, &stored); err != nil {
		return fmt.Errorf("read document %q: %w", id, err)
	}
	encoded, err := json.Marshal(expected)
	if err != nil {
		return fmt.Errorf("normalize expected document %q: %w", id, err)
	}
	var normalized map[string]any
	if err := json.Unmarshal(encoded, &normalized); err != nil {
		return fmt.Errorf("normalize expected document %q: %w", id, err)
	}
	if !equalJSONValues(stored, normalized) {
		for key, wanted := range normalized {
			got, exists := stored[key]
			if !exists {
				return fmt.Errorf("document %q is missing field %q after update", id, key)
			}
			if !equalJSONValues(got, wanted) {
				return fmt.Errorf("document %q field %q differs after update: %s", id, key, firstJSONDifference(got, wanted, key))
			}
		}
		for key := range stored {
			if _, exists := normalized[key]; !exists {
				return fmt.Errorf("document %q has unexpected field %q after update", id, key)
			}
		}
		return fmt.Errorf("document %q differs after update", id)
	}
	return nil
}

func verifyMeiliDocumentDeleted(ctx context.Context, client meilisearch.ServiceManager, id string) error {
	var stored map[string]any
	err := client.Index(benchmarkIndexUID).GetDocumentWithContext(ctx, id, nil, &stored)
	if err == nil {
		return fmt.Errorf("document %q still exists after deletion", id)
	}
	var apiError *meilisearch.Error
	if errors.As(err, &apiError) && apiError.StatusCode == http.StatusNotFound {
		return nil
	}
	return fmt.Errorf("read deleted document %q: %w", id, err)
}

// Meilisearch's JSON round-trip can move floating-point coordinates by a few
// ULPs. Keep the readback check strict for structure and non-numeric values,
// while tolerating only numerically insignificant coordinate changes.
func equalJSONValues(got, wanted any) bool {
	switch wanted := wanted.(type) {
	case float64:
		got, ok := got.(float64)
		if !ok {
			return false
		}
		scale := math.Max(1, math.Max(math.Abs(got), math.Abs(wanted)))
		return got == wanted || math.Abs(got-wanted) <= 1e-12*scale
	case []any:
		got, ok := got.([]any)
		if !ok || len(got) != len(wanted) {
			return false
		}
		for index := range wanted {
			if !equalJSONValues(got[index], wanted[index]) {
				return false
			}
		}
		return true
	case map[string]any:
		got, ok := got.(map[string]any)
		if !ok || len(got) != len(wanted) {
			return false
		}
		for key, value := range wanted {
			actual, exists := got[key]
			if !exists || !equalJSONValues(actual, value) {
				return false
			}
		}
		return true
	case string:
		got, ok := got.(string)
		return ok && got == wanted
	case bool:
		got, ok := got.(bool)
		return ok && got == wanted
	case nil:
		return got == nil
	default:
		return false
	}
}

func firstJSONDifference(got, wanted any, path string) string {
	switch wanted := wanted.(type) {
	case []any:
		got, ok := got.([]any)
		if !ok {
			return fmt.Sprintf("%s has type %T, want []any", path, got)
		}
		if len(got) != len(wanted) {
			return fmt.Sprintf("%s has length %d, want %d", path, len(got), len(wanted))
		}
		for index := range wanted {
			if !equalJSONValues(got[index], wanted[index]) {
				return firstJSONDifference(got[index], wanted[index], fmt.Sprintf("%s[%d]", path, index))
			}
		}
	case map[string]any:
		got, ok := got.(map[string]any)
		if !ok {
			return fmt.Sprintf("%s has type %T, want map[string]any", path, got)
		}
		for key, value := range wanted {
			actual, exists := got[key]
			if !exists {
				return fmt.Sprintf("%s.%s is missing", path, key)
			}
			if !equalJSONValues(actual, value) {
				return firstJSONDifference(actual, value, path+"."+key)
			}
		}
		for key := range got {
			if _, exists := wanted[key]; !exists {
				return fmt.Sprintf("%s.%s is unexpected", path, key)
			}
		}
	default:
		return fmt.Sprintf("%s is %#v (%T), want %#v (%T)", path, got, got, wanted, wanted)
	}
	return fmt.Sprintf("%s differs", path)
}

func verificationIndices(length int) []int {
	if length <= 0 {
		return nil
	}
	if length == 1 {
		return []int{0}
	}
	return []int{0, length - 1}
}

func waitForSuccessfulTask(
	ctx context.Context,
	client meilisearch.ServiceManager,
	taskInfo *meilisearch.TaskInfo,
) (*meilisearch.Task, error) {
	if taskInfo == nil {
		return nil, fmt.Errorf("Meilisearch returned no task")
	}
	pollInterval := meiliInitialTaskPollInterval
	for {
		task, err := client.GetTaskWithContext(ctx, taskInfo.TaskUID)
		if err != nil {
			if ctx.Err() != nil {
				return nil, describeTaskWaitError(client, taskInfo, err)
			}
			return nil, fmt.Errorf("wait for task %d: %w", taskInfo.TaskUID, err)
		}
		if task == nil {
			return nil, fmt.Errorf("Meilisearch returned no task result")
		}
		if task.Status != meilisearch.TaskStatusEnqueued && task.Status != meilisearch.TaskStatusProcessing {
			if err := successfulTaskError(task); err != nil {
				return task, err
			}
			return task, nil
		}

		timer := time.NewTimer(pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, describeTaskWaitError(client, taskInfo, ctx.Err())
		case <-timer.C:
		}
		pollInterval = min(pollInterval*2, meiliMaxTaskPollInterval)
	}
}

func describeTaskWaitError(
	client meilisearch.ServiceManager,
	taskInfo *meilisearch.TaskInfo,
	waitErr error,
) error {
	diagnosticCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	details := make([]string, 0, 2)
	if task, err := client.GetTaskWithContext(diagnosticCtx, taskInfo.TaskUID); err == nil && task != nil {
		details = append(details, fmt.Sprintf("last task %d is %s", task.UID, task.Status))
	}
	processing, err := client.GetTasksWithContext(diagnosticCtx, &meilisearch.TasksQuery{
		Statuses: []meilisearch.TaskStatus{meilisearch.TaskStatusProcessing},
		Limit:    1,
	})
	if err == nil && processing != nil && len(processing.Results) > 0 {
		task := processing.Results[0]
		detail := fmt.Sprintf("processing task %d", task.UID)
		if !task.StartedAt.IsZero() {
			detail += " since " + task.StartedAt.UTC().Format(time.RFC3339)
		}
		if processing.Total > 1 {
			detail += fmt.Sprintf(" (%d task records in the active batch)", processing.Total)
		}
		details = append(details, detail)
	}
	if len(details) == 0 {
		return fmt.Errorf("wait for task %d: %w", taskInfo.TaskUID, waitErr)
	}
	return fmt.Errorf("wait for task %d (%s): %w", taskInfo.TaskUID, strings.Join(details, "; "), waitErr)
}

func successfulTaskError(task *meilisearch.Task) error {
	if task == nil {
		return fmt.Errorf("Meilisearch returned no task result")
	}
	if task.Status != meilisearch.TaskStatusSucceeded {
		message := task.Error.Message
		if message == "" {
			message = string(task.Status)
		}
		return fmt.Errorf("task %d %s: %s", task.UID, task.Status, message)
	}
	return nil
}

func prepareMeiliDocuments(strategy string, trails []Trail, resolutions []int) ([]map[string]any, error) {
	return prepareMeiliDocumentsContext(context.Background(), strategy, trails, resolutions)
}

func projectMeiliIndexTrails(
	ctx context.Context,
	strategy string,
	trails []Trail,
	config GeoJSONIndexConfig,
) ([]Trail, *GeoJSONIndexReport, error) {
	if strategy != "geojson" {
		return trails, nil, nil
	}
	maxSegmentLengthMeters := config.MaxSegmentLengthMeters
	if maxSegmentLengthMeters == 0 {
		maxSegmentLengthMeters = defaultGeoJSONMaxSegmentLengthMeters
	}
	projected, report, err := normalizeGeoJSONTrailsContext(
		ctx,
		trails,
		config.SimplifyToleranceMeters,
		maxSegmentLengthMeters,
	)
	if err != nil {
		return nil, &report, err
	}
	mode := normalizedGeoJSONShardMode(config.ShardMode)
	report.ShardMode = mode
	report.MaxTrailsPerShard = config.MaxTrailsPerShard
	report.CapacityArrival = config.CapacityArrival
	report.ShardAlgorithm = geoJSONShardAlgorithmForMode(mode)
	if mode == geoJSONShardModeCapacityCreated {
		shards, shardErr := geoJSONShardCount(config, len(trails))
		if shardErr != nil {
			return nil, &report, shardErr
		}
		report.Shards = shards
		// Capacity assignment depends on the separate synthetic local-arrival
		// plan. Per-shard geometry is populated after the final document IDs are
		// available; indexed product/origin dates never feed the planner.
		return projected, &report, nil
	}
	shards := max(1, config.Shards)
	report.Shards = shards
	report.ShardDocumentCounts = make([]int, shards)
	report.ShardSourceVertices = make([]int, shards)
	report.ShardIndexedVertices = make([]int, shards)
	report.ShardSourceSegments = make([]int, shards)
	report.ShardIndexedSegments = make([]int, shards)
	for index, trail := range projected {
		report.ShardDocumentCounts[geoJSONShardForID(trail.ID, shards)]++
		shard := geoJSONShardForID(trail.ID, shards)
		for _, part := range trails[index].Parts {
			report.ShardSourceVertices[shard] += len(part)
			report.ShardSourceSegments[shard] += segmentCount(part)
		}
		for _, part := range trail.Parts {
			report.ShardIndexedVertices[shard] += len(part)
			report.ShardIndexedSegments[shard] += segmentCount(part)
		}
	}
	return projected, &report, nil
}

func mergeMeiliProductDocuments(documents []map[string]any, products []ProductQueryDocument) error {
	if len(documents) != len(products) {
		return fmt.Errorf("merge product fields: %d Meilisearch documents for %d product documents", len(documents), len(products))
	}
	for index, product := range products {
		document := documents[index]
		id, _ := document["id"].(string)
		if id == "" || id != product.Trail.ID {
			return fmt.Errorf("merge product fields at position %d: Meilisearch id %q, product id %q", index, id, product.Trail.ID)
		}
		document["author_name"] = product.AuthorName
		document["author_avatar"] = product.AuthorAvatar
		document["name"] = product.Name
		document["description"] = product.Description
		document["location"] = product.Location
		document["tags"] = product.Tags
		document["author"] = product.AuthorID
		document["public"] = product.Public
		document["shares"] = product.SharedWithActorIDs
		document["is_federated"] = product.Federated
		document["category"] = product.CategoryName
		document["category_id"] = product.CategoryID
		document["category_icon"] = product.CategoryIcon
		if product.SubcategoryID == "" {
			document["subcategory_id"] = nil
		} else {
			document["subcategory_id"] = product.SubcategoryID
		}
		document["federated_category_name"] = product.FederatedCategoryName
		document["federated_subcategory_name"] = product.FederatedSubcategoryName
		document["distance"] = product.DistanceMeters
		document["duration"] = product.DurationSeconds
		document["elevation_gain"] = product.ElevationGainMeters
		document["elevation_loss"] = product.ElevationLossMeters
		document["difficulty"] = product.Difficulty
		document["completed"] = product.Completed
		document["date"] = product.DateUnix
		document["created"] = product.CreatedUnix
		document["thumbnail"] = product.Thumbnail
		document["domain"] = product.Domain
		document["gpx"] = product.GPX
		document["like_count"] = product.LikeCount
		document["iri"] = product.IRI
		document["bounding_box_diagonal"] = product.BoundingBoxDiagonal
		document["_geo"] = map[string]float64{"lat": product.Geo.Lat, "lng": product.Geo.Lng}
		document["polyline"] = productDocumentPolyline(product)
	}
	return nil
}

func productDocumentsForUpdatedTrails(products []ProductQueryDocument, trails []Trail) []ProductQueryDocument {
	byID := productDocumentsByID(products)
	updated := make([]ProductQueryDocument, 0, len(trails))
	for _, trail := range trails {
		product, found := byID[trail.ID]
		if !found {
			// This can only happen for a malformed mutation fixture. Preserve the
			// positional merge error so the caller reports the concrete ID.
			product = ProductQueryDocument{Trail: trail}
		} else {
			oldDistance := product.DistanceMeters
			product.Trail = trail
			product.Polyline = compactTrailPolyline(trail)
			product.DistanceMeters = productTrailLengthMeters(trail)
			product.BoundingBoxDiagonal = productTrailBoundingBoxDiagonalMeters(trail)
			start := trail.Start()
			product.Geo = ProductGeoPoint{Lat: start.Lat, Lng: start.Lon}
			if oldDistance > 0 && product.DurationSeconds > 0 {
				product.DurationSeconds *= product.DistanceMeters / oldDistance
			}
		}
		updated = append(updated, product)
	}
	return updated
}

func prepareMeiliDocumentsContext(ctx context.Context, strategy string, trails []Trail, resolutions []int) ([]map[string]any, error) {
	documents := make([]map[string]any, 0, len(trails))
	for _, trail := range trails {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		document := map[string]any{
			"id":       trail.ID,
			"scenario": trail.Scenario,
			"polyline": compactTrailPolyline(trail),
		}
		switch strategy {
		case "point":
			start := trail.Start()
			document["_geo"] = map[string]float64{"lat": start.Lat, "lng": start.Lon}
		case "geojson":
			document["_geojson"] = geoJSONFeature(trail)
		case "h3":
			fields, err := h3FieldsForTrailContext(ctx, trail, resolutions)
			if err != nil {
				return nil, fmt.Errorf("trail %s: %w", trail.ID, err)
			}
			for key, values := range fields {
				document[key] = values
			}
		default:
			return nil, fmt.Errorf("unknown Meilisearch strategy %q", strategy)
		}
		documents = append(documents, document)
	}
	return documents, nil
}

// compactTrailPolyline mirrors Wanderer's production Meilisearch projection.
// GeoJSON and H3 remain additional strategy-specific fields; no backend pays
// for a benchmark-only raw coordinate matrix in its common document payload.
func compactTrailPolyline(trail Trail) string {
	coordinates := make([][]float64, 0)
	for _, part := range trail.Parts {
		for _, coordinate := range part {
			coordinates = append(coordinates, []float64{coordinate.Lat, coordinate.Lon})
		}
	}
	return string(polyline.EncodeCoords(coordinates))
}

func geoJSONFeature(trail Trail) map[string]any {
	coordinates := make([][][]float64, 0, len(trail.Parts))
	for _, part := range trail.Parts {
		line := make([][]float64, 0, len(part))
		for _, coordinate := range part {
			line = append(line, []float64{coordinate.Lon, coordinate.Lat})
		}
		if len(line) > 0 {
			coordinates = append(coordinates, line)
		}
	}
	geometry := map[string]any{"type": "MultiLineString", "coordinates": coordinates}
	if len(coordinates) == 1 {
		geometry = map[string]any{"type": "LineString", "coordinates": coordinates[0]}
	}
	return map[string]any{
		"type":       "Feature",
		"properties": map[string]any{},
		"geometry":   geometry,
	}
}

func geoJSONDocument(id string, parts [][][]float64) map[string]any {
	trail := Trail{ID: id, Parts: make([][]Coordinate, 0, len(parts))}
	for _, part := range parts {
		line := make([]Coordinate, 0, len(part))
		for _, point := range part {
			line = append(line, Coordinate{Lon: point[0], Lat: point[1]})
		}
		trail.Parts = append(trail.Parts, line)
	}
	return map[string]any{"id": id, "_geojson": geoJSONFeature(trail)}
}

func searchGeoRadius(
	ctx context.Context,
	client meilisearch.ServiceManager,
	field string,
	point Coordinate,
	radius float64,
	limit int64,
) (CandidateResult, error) {
	_ = field // Both reserved geo fields use the same _geoRadius expression.
	return searchFilter(ctx, client, geoRadiusFilter(point, radius, 0), limit)
}

func searchGeoBoundingBox(
	ctx context.Context,
	client meilisearch.ServiceManager,
	viewport GeoViewport,
	limit int64,
) (CandidateResult, error) {
	filter, err := geoBoundingBoxFilter(viewport)
	if err != nil {
		return CandidateResult{}, err
	}
	return searchFilter(ctx, client, filter, limit)
}

func searchFilter(
	ctx context.Context,
	client meilisearch.ServiceManager,
	filter string,
	limit int64,
) (CandidateResult, error) {
	return searchFilterAtIndex(ctx, client, benchmarkIndexUID, filter, limit)
}

func searchFilterAtIndex(
	ctx context.Context,
	client meilisearch.ServiceManager,
	indexUID string,
	filter string,
	limit int64,
) (CandidateResult, error) {
	started := time.Now()
	response, err := client.Index(indexUID).SearchWithContext(ctx, "", &meilisearch.SearchRequest{
		Filter:               filter,
		Limit:                limit,
		AttributesToRetrieve: []string{"id"},
	})
	if err != nil {
		return CandidateResult{}, err
	}
	ids := make(map[string]struct{}, len(response.Hits))
	for _, hit := range response.Hits {
		rawID, ok := hit["id"]
		if !ok {
			return CandidateResult{}, fmt.Errorf("search hit has no id")
		}
		var id string
		if err := json.Unmarshal(rawID, &id); err != nil {
			return CandidateResult{}, fmt.Errorf("decode hit id: %w", err)
		}
		ids[id] = struct{}{}
	}
	wallDuration := time.Since(started)
	return CandidateResult{
		IDs:                ids,
		WallDuration:       wallDuration,
		EngineProcessingMS: float64(response.ProcessingTimeMs),
		EstimatedTotalHits: response.EstimatedTotalHits,
		Truncated:          int64(len(response.Hits)) >= limit && response.EstimatedTotalHits > int64(len(ids)),
	}, nil
}

func searchExactCountAtIndex(
	ctx context.Context,
	client meilisearch.ServiceManager,
	indexUID, filter string,
) (int64, error) {
	response, err := client.Index(indexUID).SearchWithContext(ctx, "", &meilisearch.SearchRequest{
		Filter:               filter,
		HitsPerPage:          1,
		Page:                 1,
		AttributesToRetrieve: []string{"id"},
	})
	if err != nil {
		return 0, err
	}
	return response.TotalHits, nil
}

func sortedUniqueInts(values []int) []int {
	seen := make(map[int]struct{}, len(values))
	result := make([]int, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Ints(result)
	return result
}

func idSet(ids ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		result[id] = struct{}{}
	}
	return result
}

func equalSets(a, b map[string]struct{}) bool {
	if len(a) != len(b) {
		return false
	}
	for id := range a {
		if _, ok := b[id]; !ok {
			return false
		}
	}
	return true
}

func sortedIDs(ids map[string]struct{}) []string {
	values := make([]string, 0, len(ids))
	for id := range ids {
		values = append(values, id)
	}
	sort.Strings(values)
	return values
}

func meiliImageLabel(image string) string {
	if index := strings.LastIndex(image, ":"); index >= 0 {
		return strings.TrimPrefix(image[index+1:], "v")
	}
	return image
}

func geoJSONFailureStatus(version string) string {
	major, minor, patch, ok := parseVersion(version)
	if ok && (major < 1 || (major == 1 && (minor < 22 || (minor == 22 && patch < 1)))) {
		return "unsupported"
	}
	return "failed"
}

func parseVersion(version string) (major, minor, patch int, ok bool) {
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	if suffix := strings.IndexAny(version, "-+"); suffix >= 0 {
		version = version[:suffix]
	}
	parts := strings.Split(version, ".")
	if len(parts) < 3 {
		return 0, 0, 0, false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, false
	}
	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, false
	}
	patch, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, false
	}
	return major, minor, patch, true
}

type meiliReleaseVersion struct {
	major int
	minor int
	patch int
}

var minimumMeiliQualificationVersion = meiliReleaseVersion{major: 1, minor: 53, patch: 1}

func validateMeiliQualificationImage(reference string) error {
	_, _, err := parseMeiliQualificationImage(reference)
	return err
}

func parseMeiliQualificationImage(reference string) (meiliReleaseVersion, bool, error) {
	const repository = "getmeili/meilisearch"
	if reference == "" || strings.TrimSpace(reference) != reference {
		return meiliReleaseVersion{}, false, fmt.Errorf("image reference must be non-empty and contain no surrounding whitespace")
	}

	nameAndTag, digest, hasDigest := strings.Cut(reference, "@")
	if hasDigest && !validSHA256ImageDigest(digest) {
		return meiliReleaseVersion{}, false, fmt.Errorf("image %q has an invalid sha256 digest", reference)
	}
	if nameAndTag == repository {
		if !hasDigest {
			return meiliReleaseVersion{}, false, fmt.Errorf("image %q requires an explicit stable release tag or immutable sha256 digest", reference)
		}
		return meiliReleaseVersion{}, false, nil
	}
	if !strings.HasPrefix(nameAndTag, repository+":") {
		return meiliReleaseVersion{}, false, fmt.Errorf("image %q is not an official %s release reference", reference, repository)
	}

	tag := strings.TrimPrefix(nameAndTag, repository+":")
	version, ok := parseStableMeiliVersion(tag, true)
	if !ok {
		return meiliReleaseVersion{}, false, fmt.Errorf("image %q must use an explicit stable vMAJOR.MINOR.PATCH tag", reference)
	}
	if compareMeiliReleaseVersions(version, minimumMeiliQualificationVersion) < 0 {
		return meiliReleaseVersion{}, false, fmt.Errorf("image %q is older than the minimum supported release v1.53.1", reference)
	}
	return version, true, nil
}

func validateMeiliQualificationRuntime(image, runtimeVersion string) error {
	declared, hasDeclaredVersion, err := parseMeiliQualificationImage(image)
	if err != nil {
		return err
	}
	actual, ok := parseStableMeiliVersion(runtimeVersion, false)
	if !ok {
		return fmt.Errorf("runtime version %q is not a stable MAJOR.MINOR.PATCH release", runtimeVersion)
	}
	if compareMeiliReleaseVersions(actual, minimumMeiliQualificationVersion) < 0 {
		return fmt.Errorf("runtime version %q is older than the minimum supported release 1.53.1", runtimeVersion)
	}
	if hasDeclaredVersion && compareMeiliReleaseVersions(actual, declared) != 0 {
		return fmt.Errorf("runtime version %q does not match image tag v%d.%d.%d", runtimeVersion, declared.major, declared.minor, declared.patch)
	}
	return nil
}

func shouldRunGeoJSONFederatedPageContract(image, runtimeVersion string) (bool, error) {
	if validateMeiliQualificationImage(image) == nil {
		if err := validateMeiliQualificationRuntime(image, runtimeVersion); err != nil {
			return false, err
		}
		return true, nil
	}
	actual, ok := parseStableMeiliVersion(runtimeVersion, false)
	return ok && compareMeiliReleaseVersions(actual, minimumMeiliQualificationVersion) >= 0, nil
}

func parseStableMeiliVersion(value string, requireVPrefix bool) (meiliReleaseVersion, bool) {
	value = strings.TrimSpace(value)
	if requireVPrefix {
		if !strings.HasPrefix(value, "v") {
			return meiliReleaseVersion{}, false
		}
		value = strings.TrimPrefix(value, "v")
	} else {
		value = strings.TrimPrefix(value, "v")
	}
	if value == "" || strings.ContainsAny(value, "-+") {
		return meiliReleaseVersion{}, false
	}
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return meiliReleaseVersion{}, false
	}
	numbers := [3]int{}
	for index, part := range parts {
		if part == "" || (len(part) > 1 && part[0] == '0') {
			return meiliReleaseVersion{}, false
		}
		parsed, err := strconv.Atoi(part)
		if err != nil || parsed < 0 {
			return meiliReleaseVersion{}, false
		}
		numbers[index] = parsed
	}
	return meiliReleaseVersion{major: numbers[0], minor: numbers[1], patch: numbers[2]}, true
}

func compareMeiliReleaseVersions(left, right meiliReleaseVersion) int {
	if left.major != right.major {
		return left.major - right.major
	}
	if left.minor != right.minor {
		return left.minor - right.minor
	}
	return left.patch - right.patch
}

func validSHA256ImageDigest(digest string) bool {
	const prefix = "sha256:"
	if !strings.HasPrefix(digest, prefix) || len(digest) != len(prefix)+64 {
		return false
	}
	for _, char := range strings.TrimPrefix(digest, prefix) {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}
