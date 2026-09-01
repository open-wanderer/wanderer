package main

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/meilisearch/meilisearch-go"
)

func TestProductMeiliFilterCombinesProductAndSpatialPredicates(t *testing.T) {
	minimumDistance, maximumDuration := 1200.5, 7200.0
	minimumDifficulty, maximumDifficulty := 1.0, 2.0
	federated := false
	query := ProductQuery{
		ActorID: `actor-"quoted`,
		Geo: &ProductGeoRadius{
			Center:       Coordinate{Lat: 46.8, Lon: 8.2},
			RadiusMeters: 5000,
		},
		Filters: ProductQueryFilters{
			CategoryIDs:                     []string{"hike", "bike"},
			SubcategoryIDs:                  []string{"alpine"},
			CategoriesWithoutSubcategoryIDs: []string{"run"},
			Federated:                       &federated,
			DistanceMeters:                  ProductNumericRange{Min: &minimumDistance},
			DurationSeconds:                 ProductNumericRange{Max: &maximumDuration},
			Difficulty:                      ProductNumericRange{Min: &minimumDifficulty, Max: &maximumDifficulty},
		},
		Sort:    ProductSortCreatedDesc,
		Page:    1,
		PerPage: 10,
	}
	filter, err := productMeiliFilter(query, `h3_r7 IN ["871f"]`)
	if err != nil {
		t.Fatal(err)
	}
	for _, wanted := range []string{
		`(public = true OR author = "actor-\"quoted" OR shares = "actor-\"quoted")`,
		`(category_id IN ["hike", "bike"] OR subcategory_id IN ["alpine"] OR (category_id IN ["run"] AND subcategory_id IS NULL))`,
		`is_federated = false`,
		`distance >= 1200.5`,
		`duration <= 7200`,
		`difficulty >= 1`,
		`difficulty <= 2`,
		`(h3_r7 IN ["871f"])`,
	} {
		if !strings.Contains(filter, wanted) {
			t.Errorf("filter %q does not contain %q", filter, wanted)
		}
	}
	if strings.Index(filter, "public = true") > strings.Index(filter, "h3_r7") {
		t.Fatalf("ACL should precede the supplied spatial predicate: %q", filter)
	}
}

func TestAuditGeoJSONDirectProductCaseReportsCountClassificationsAndPages(t *testing.T) {
	query := ProductQuery{
		Geo: &ProductGeoRadius{
			Center:       Coordinate{Lat: 46.8, Lon: 8.2},
			RadiusMeters: 1_000,
		},
		Sort:    ProductSortProximityAsc,
		Page:    1,
		PerPage: 10,
	}
	documents := map[string]ProductQueryDocument{
		"inside":         {Trail: Trail{ID: "inside"}, Public: true},
		"boundary-miss":  {Trail: Trail{ID: "boundary-miss"}, Public: true},
		"boundary-extra": {Trail: Trail{ID: "boundary-extra"}, Public: true},
		"far-extra":      {Trail: Trail{ID: "far-extra"}, Public: true},
	}
	reference := geoJSONDirectReferenceProductCase{
		benchmark: productBenchmarkCase{
			Name:       "count diagnostics",
			QueryPoint: QueryPoint{ID: "query-count", Scenario: "alpine_dense"},
			Radius:     1_000,
			Query:      query,
		},
		normalized:    query,
		sortSupported: false,
		expected:      []string{"inside", "boundary-miss"},
		expectedPage:  []string{"inside", "boundary-miss"},
		distances: map[string]float64{
			"inside": 100, "boundary-miss": 980, "boundary-extra": 1_020, "far-extra": 1_100,
		},
	}
	calls := 0
	search := func(
		context.Context,
		ProductQuery,
		GeoJSONQueryPlan,
		bool,
		bool,
		int64,
		[]string,
		[]string,
	) (geoJSONDirectSearchResult, error) {
		calls++
		return geoJSONDirectSearchResult{
			IDs:         []string{"inside", "boundary-extra", "far-extra"},
			TotalHits:   3,
			TotalPages:  1,
			Page:        1,
			HitsPerPage: 10,
		}, nil
	}
	prepared, _, _, err := auditGeoJSONDirectProductCase(
		context.Background(),
		search,
		reference,
		documents,
		GeoJSONQueryPlan{ID: "r100-exact", RadiusMode: geoJSONRadiusExact, Resolution: 100},
		GeoJSONDirectConfig{BoundaryToleranceMeters: 50},
		10,
	)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("search calls = %d, want complete IDs plus product page", calls)
	}
	report := prepared.caseReport
	if report.ExpectedTotalItems != 2 || report.ActualTotalItems != 3 || report.SignedCountDelta != 1 {
		t.Fatalf("count expected/actual/delta = %d/%d/%d, want 2/3/+1", report.ExpectedTotalItems, report.ActualTotalItems, report.SignedCountDelta)
	}
	if report.FalseNegatives != 1 || report.BoundaryFalseNegatives != 1 || report.MaterialFalseNegatives != 0 {
		t.Fatalf("FN total/boundary/material = %d/%d/%d, want 1/1/0", report.FalseNegatives, report.BoundaryFalseNegatives, report.MaterialFalseNegatives)
	}
	if report.FalsePositives != 2 || report.BoundaryFalsePositives != 1 || report.MaterialFalsePositives != 1 || report.SemanticFalsePositives != 0 {
		t.Fatalf(
			"FP total/boundary/material/semantic = %d/%d/%d/%d, want 2/1/1/0",
			report.FalsePositives,
			report.BoundaryFalsePositives,
			report.MaterialFalsePositives,
			report.SemanticFalsePositives,
		)
	}
	if report.ExpectedTotalPages != 1 || report.ActualTotalPages != 1 {
		t.Fatalf("totalPages expected/actual = %d/%d, want 1/1", report.ExpectedTotalPages, report.ActualTotalPages)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		`"signed_count_delta":1`,
		`"boundary_false_negatives":1`,
		`"material_false_positives":1`,
		`"expected_total_pages":1`,
		`"actual_total_pages":1`,
	} {
		if !strings.Contains(string(encoded), key) {
			t.Fatalf("case JSON %s is missing %s", encoded, key)
		}
	}
}

func TestSummarizeGeoJSONDirectProductCounts(t *testing.T) {
	report := summarizeGeoJSONDirectProductCounts([]GeoJSONDirectProductCaseReport{
		{ExpectedTotalItems: 100, ActualTotalItems: 100, ExpectedTotalPages: 10, ActualTotalPages: 10},
		{ExpectedTotalItems: 100, ActualTotalItems: 101, ExpectedTotalPages: 10, ActualTotalPages: 11},
		{ExpectedTotalItems: 0, ActualTotalItems: 1, ExpectedTotalPages: 0, ActualTotalPages: 1},
	}, 0.01)
	if report.Status != "failed" || report.MaximumP95RelativeError != 0.01 ||
		report.Cases != 3 || report.ExactCountCases != 1 || report.WithinOnePercentCases != 2 || report.TotalPagesMatchCases != 1 {
		t.Fatalf("count diagnostics = %+v", report)
	}
	if report.ExactCountRate != 1.0/3.0 || report.WithinOnePercentRate != 2.0/3.0 || report.TotalPagesMatchRate != 1.0/3.0 {
		t.Fatalf("count diagnostic rates = %+v", report)
	}
	if math.Abs(report.CountError.MeanAbsolute-2.0/3.0) > 1e-12 || report.CountError.MaxAbsolute != 1 || report.CountError.MaxRelative != 1 {
		t.Fatalf("count errors = %+v", report.CountError)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{`"exact_count_rate"`, `"within_one_percent_rate"`, `"total_pages_match_rate"`, `"count_error"`} {
		if !strings.Contains(string(encoded), key) {
			t.Fatalf("aggregate JSON %s is missing %s", encoded, key)
		}
	}
}

func TestGeoJSONDirectProductCountQualificationUsesOnlyRelativeP95(t *testing.T) {
	tests := []struct {
		name   string
		cases  []GeoJSONDirectProductCaseReport
		status string
	}{
		{
			name: "one percent passes",
			cases: []GeoJSONDirectProductCaseReport{{
				ExpectedTotalItems: 100, ActualTotalItems: 101,
			}},
			status: "passed",
		},
		{
			name: "over one percent fails",
			cases: []GeoJSONDirectProductCaseReport{{
				ExpectedTotalItems: 100, ActualTotalItems: 102,
			}},
			status: "failed",
		},
		{
			name: "absolute error above one still passes",
			cases: []GeoJSONDirectProductCaseReport{{
				ExpectedTotalItems: 1_000, ActualTotalItems: 1_005,
			}},
			status: "passed",
		},
		{name: "missing evidence is inconclusive", status: "inconclusive"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report := summarizeGeoJSONDirectProductCounts(test.cases, 0.01)
			if report.Status != test.status {
				t.Fatalf("count qualification = %+v, want status %q", report, test.status)
			}
		})
	}

	falseEmptyCases := make([]GeoJSONDirectProductCaseReport, 101)
	for index := range falseEmptyCases {
		falseEmptyCases[index] = GeoJSONDirectProductCaseReport{
			ExpectedTotalItems: 1,
			ActualTotalItems:   1,
		}
	}
	falseEmptyCases[len(falseEmptyCases)-1].ActualTotalItems = 0
	falseEmptyCases[len(falseEmptyCases)-1].FalseEmpty = true
	if report := summarizeGeoJSONDirectProductCounts(falseEmptyCases, 0.01); report.Status != "passed" {
		t.Fatalf("an isolated false-empty diagnostic unexpectedly became a separate count gate: %+v", report)
	}
}

func TestProductMeiliFilterAnonymousReferenceOmitsSpatialPredicate(t *testing.T) {
	query := ProductQuery{Sort: ProductSortRelevance, Page: 1, PerPage: 10}
	filter, err := productMeiliFilter(query, "")
	if err != nil {
		t.Fatal(err)
	}
	if filter != "public = true" {
		t.Fatalf("anonymous non-spatial filter = %q, want public-only ACL", filter)
	}
}

func TestGeoJSONDirectMeiliSortMakesRouteProximityUnsupported(t *testing.T) {
	if sortValues, supported := geoJSONDirectMeiliSort(ProductQuery{Sort: ProductSortRelevance}); !supported || !reflect.DeepEqual(sortValues, []string{"id:asc"}) {
		t.Fatalf("relevance tie-break sort = %v, supported %t", sortValues, supported)
	}
	if sortValues, supported := geoJSONDirectMeiliSort(ProductQuery{Sort: ProductSortProximityAsc}); supported || len(sortValues) != 0 {
		t.Fatalf("route-proximity sort = %v, supported %t; want visibly unsupported", sortValues, supported)
	}
	if sortValues, supported := geoJSONDirectMeiliSort(ProductQuery{Sort: ProductSortElevationGainDesc}); !supported || len(sortValues) != 2 || sortValues[0] != "elevation_gain:desc" || sortValues[1] != "id:asc" {
		t.Fatalf("elevation sort = %v, supported %t", sortValues, supported)
	}
}

func TestGeoJSONDirectSubsetProductOrderUsesProductSemantics(t *testing.T) {
	documents := map[string]ProductQueryDocument{
		"later": {
			Trail: Trail{ID: "later"}, Public: true, Name: "alpine later", CreatedUnix: 20,
		},
		"private": {
			Trail: Trail{ID: "private"}, Public: false, Name: "alpine private", CreatedUnix: 5,
		},
		"wrong-text": {
			Trail: Trail{ID: "wrong-text"}, Public: true, Name: "coastal route", CreatedUnix: 1,
		},
		"earlier": {
			Trail: Trail{ID: "earlier"}, Public: true, Name: "alpine earlier", CreatedUnix: 10,
		},
	}
	ordered, err := geoJSONDirectSubsetProductOrder(
		context.Background(),
		[]string{"later", "private", "wrong-text", "earlier"},
		documents,
		ProductQuery{Text: "alpine", Sort: ProductSortCreatedAsc, Page: 1, PerPage: 10},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !equalStringSlices(ordered, []string{"earlier", "later"}) {
		t.Fatalf("ordered direct subset = %v, want ProductQuery-filtered [earlier later]", ordered)
	}
}

func TestPrepareGeoJSONDirectProductBenchmarkUsesOnlyExhaustiveOracle(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		http.Error(writer, "prepare must not query Meilisearch", http.StatusInternalServerError)
	}))
	defer server.Close()

	nearPart := [][]Coordinate{{
		{Lat: 46.8, Lon: 8.2},
		{Lat: 46.8001, Lon: 8.2001},
	}}
	dataset := Dataset{
		Trails: []Trail{
			{ID: "match", Scenario: "alpine_dense", Parts: nearPart},
			{ID: "wrong-text", Scenario: "alpine_dense", Parts: nearPart},
			{ID: "private", Scenario: "alpine_dense", Parts: nearPart},
		},
		QueryPoints: []QueryPoint{{
			ID: "query", Scenario: "alpine_dense", Point: Coordinate{Lat: 46.8, Lon: 8.2},
		}},
	}
	documents := generateProductDocuments(dataset)
	for index := range documents {
		documents[index].Public = true
	}
	documents[1].AuthorName = ""
	documents[1].Name = "coastal"
	documents[1].Description = "sea"
	documents[1].Location = "coast"
	documents[1].Tags = nil
	documents[2].Public = false

	benchmark, err := prepareGeoJSONDirectProductBenchmark(
		context.Background(), meilisearch.New(server.URL), dataset, []float64{500}, documents,
	)
	if err != nil {
		t.Fatal(err)
	}
	if requests != 0 {
		t.Fatalf("prepare issued %d Meilisearch requests, want none", requests)
	}
	if len(benchmark.references) != 1 || !equalStringSlices(benchmark.references[0].expected, []string{"match"}) {
		t.Fatalf("exhaustive reference = %+v, want only visible text match", benchmark.references)
	}
}

func TestValidateGeoJSONDirectProductPageChecksOrderingAndMetadata(t *testing.T) {
	valid := geoJSONDirectSearchResult{
		IDs: []string{"c", "d"}, TotalHits: 5, TotalPages: 3, Page: 2, HitsPerPage: 2,
	}
	if err := validateGeoJSONDirectProductPage(valid, []string{"c", "d"}, 5, 3, 2, 2); err != nil {
		t.Fatalf("valid page: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*geoJSONDirectSearchResult)
	}{
		{name: "order", mutate: func(result *geoJSONDirectSearchResult) { result.IDs = []string{"d", "c"} }},
		{name: "total hits", mutate: func(result *geoJSONDirectSearchResult) { result.TotalHits = 4 }},
		{name: "total pages", mutate: func(result *geoJSONDirectSearchResult) { result.TotalPages = 2 }},
		{name: "page", mutate: func(result *geoJSONDirectSearchResult) { result.Page = 1 }},
		{name: "per page", mutate: func(result *geoJSONDirectSearchResult) { result.HitsPerPage = 10 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := valid
			test.mutate(&actual)
			if err := validateGeoJSONDirectProductPage(actual, []string{"c", "d"}, 5, 3, 2, 2); err == nil {
				t.Fatal("invalid page metadata/order was accepted")
			}
		})
	}
}

func TestObserveGeoJSONDirectProductCaseUsesFiftyMeterUXBand(t *testing.T) {
	query := ProductQuery{
		Geo:     &ProductGeoRadius{Center: Coordinate{}, RadiusMeters: 1000},
		Sort:    ProductSortCreatedAsc,
		Page:    1,
		PerPage: 10,
	}
	documents := map[string]ProductQueryDocument{
		"deep": {Trail: Trail{ID: "deep"}, Public: true},
		"edge": {Trail: Trail{ID: "edge"}, Public: true},
	}
	benchmarkCase := productBenchmarkCase{
		QueryPoint: QueryPoint{ID: "q", Scenario: "alpine_dense"},
		Radius:     1000,
		Query:      query,
	}
	observation := observeGeoJSONDirectProductCase(
		benchmarkCase,
		query,
		[]string{"edge"},
		[]string{"edge"},
		[]string{"deep", "edge"},
		[]string{"deep", "edge"},
		map[string]float64{"deep": 800, "edge": 980},
		documents,
		GeoJSONDirectConfig{BoundaryToleranceMeters: 50},
		true,
	)
	if observation.materialFalseNegatives != 1 || observation.boundaryMisses != 0 {
		t.Fatalf("material misses=%d boundary misses=%d, want 1/0", observation.materialFalseNegatives, observation.boundaryMisses)
	}
	if observation.maxInsideMiss != 200 || !observation.falseEmpty {
		t.Fatalf("max inside miss=%v false empty=%t, want 200/true", observation.maxInsideMiss, observation.falseEmpty)
	}
}

func TestObserveGeoJSONDirectProductCaseScoresMaterialPagePrecision(t *testing.T) {
	query := ProductQuery{
		Geo:     &ProductGeoRadius{RadiusMeters: 1_000},
		Sort:    ProductSortCreatedAsc,
		Page:    1,
		PerPage: 10,
	}
	documents := map[string]ProductQueryDocument{
		"inside":  {Trail: Trail{ID: "inside"}, Public: true},
		"too-far": {Trail: Trail{ID: "too-far"}, Public: true},
	}
	observation := observeGeoJSONDirectProductCase(
		productBenchmarkCase{QueryPoint: QueryPoint{ID: "q"}, Radius: 1_000, Query: query},
		query,
		[]string{"inside", "too-far"},
		[]string{"inside", "too-far"},
		[]string{"inside"},
		[]string{"inside"},
		map[string]float64{"inside": 100, "too-far": 1_100},
		documents,
		GeoJSONDirectConfig{BoundaryToleranceMeters: 50},
		true,
	)
	if observation.comparablePageActual != 2 || observation.comparablePageMaterialFP != 1 {
		t.Fatalf("page actual/material FP = %d/%d, want 2/1", observation.comparablePageActual, observation.comparablePageMaterialFP)
	}
	accumulator := newGeoJSONDirectAccuracyAccumulator(GeoJSONDirectConfig{MinimumComparablePagePrecision: 0.75})
	accumulator.add(observation)
	report := accumulator.finish(GeoJSONDirectConfig{MinimumComparablePagePrecision: 0.75}, true)
	if report.ComparablePagePrecision != 0.5 || report.Status != "failed" {
		t.Fatalf("page precision report = %+v", report)
	}
}

func TestObserveGeoJSONDirectProductCaseRejectsWrongClearOrdering(t *testing.T) {
	query := ProductQuery{
		Geo: &ProductGeoRadius{RadiusMeters: 1_000}, Sort: ProductSortCreatedAsc, Page: 1, PerPage: 10,
	}
	documents := map[string]ProductQueryDocument{
		"a": {Trail: Trail{ID: "a"}, Public: true},
		"b": {Trail: Trail{ID: "b"}, Public: true},
	}
	observation := observeGeoJSONDirectProductCase(
		productBenchmarkCase{QueryPoint: QueryPoint{ID: "q"}, Radius: 1_000, Query: query},
		query,
		[]string{"b", "a"},
		[]string{"b", "a"},
		[]string{"a", "b"},
		[]string{"a", "b"},
		map[string]float64{"a": 100, "b": 200},
		documents,
		GeoJSONDirectConfig{BoundaryToleranceMeters: 50},
		true,
	)
	if observation.semanticFalsePositives == 0 || !strings.Contains(observation.semanticError, "ordering") {
		t.Fatalf("wrong clear ordering was not rejected: %+v", observation)
	}
}

func TestBenchmarkGeoJSONDirectProductWorkloadUsesSingleProductSearch(t *testing.T) {
	type capturedRequest struct {
		Query                string   `json:"q"`
		Filter               string   `json:"filter"`
		Limit                int64    `json:"limit"`
		HitsPerPage          int64    `json:"hitsPerPage"`
		Page                 int64    `json:"page"`
		MatchingStrategy     string   `json:"matchingStrategy"`
		AttributesToRetrieve []string `json:"attributesToRetrieve"`
		Sort                 []string `json:"sort"`
	}
	var lock sync.Mutex
	requests := make([]capturedRequest, 0)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/indexes/"+benchmarkIndexUID+"/search" {
			http.NotFound(writer, request)
			return
		}
		var captured capturedRequest
		if err := json.NewDecoder(request.Body).Decode(&captured); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		lock.Lock()
		requests = append(requests, captured)
		lock.Unlock()
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"hits":               []map[string]any{geoJSONDirectTestProductHit()},
			"estimatedTotalHits": 1,
			"totalHits":          1,
			"page":               1,
			"hitsPerPage":        10,
			"totalPages":         1,
			"processingTimeMs":   2,
			"query":              captured.Query,
		})
	}))
	defer server.Close()

	dataset := Dataset{
		Trails: []Trail{{
			ID:       "trail-1",
			Scenario: "alpine_dense",
			Parts: [][]Coordinate{{
				{Lat: 46.8, Lon: 8.2},
				{Lat: 46.801, Lon: 8.201},
			}},
		}},
		QueryPoints: []QueryPoint{{
			ID:       "query-1",
			Scenario: "alpine_dense",
			Point:    Coordinate{Lat: 46.8, Lon: 8.2},
		}},
	}
	documents := generateProductDocuments(dataset)
	documents[0].Public = true
	report, err := benchmarkGeoJSONDirectProductWorkload(
		context.Background(),
		meilisearch.New(server.URL),
		dataset,
		[]float64{500},
		1,
		1,
		documents,
		GeoJSONQueryPlan{
			ID:              "r125-exact",
			RadiusMode:      geoJSONRadiusExact,
			Resolution:      125,
			RadiusFormula:   "r",
			ExactRefinement: false,
		},
		GeoJSONDirectConfig{
			Enabled:                        true,
			Resolutions:                    []int{125},
			BoundaryToleranceMeters:        50,
			MinimumRecall:                  0.99,
			MinimumPrecision:               0.99,
			MinimumTop10Recall:             0.99,
			MinimumComparablePageRate:      0.99,
			MinimumComparablePagePrecision: 0.99,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "passed" || report.Samples != 1 || report.Accuracy.Status != "passed" {
		t.Fatalf("report status=%s samples=%d accuracy=%s", report.Status, report.Samples, report.Accuracy.Status)
	}
	if report.AverageResponseBytes <= 0 || report.TotalLatency.P50 <= 0 {
		t.Fatalf("response bytes=%v latency=%+v", report.AverageResponseBytes, report.TotalLatency)
	}

	lock.Lock()
	defer lock.Unlock()
	if len(requests) != 3 {
		t.Fatalf("search requests=%d, want 3 (direct audit, audit page, warm)", len(requests))
	}
	var sawTimedProduct bool
	for _, request := range requests {
		if request.Query != "alpine dense" {
			t.Errorf("query text=%q, want %q", request.Query, "alpine dense")
		}
		if !strings.Contains(request.Filter, "public = true") {
			t.Errorf("request filter has no ACL: %q", request.Filter)
		}
		if request.MatchingStrategy != "all" {
			t.Errorf("matching strategy=%q, want all", request.MatchingStrategy)
		}
		if request.HitsPerPage == 10 && request.Page == 1 && strings.Contains(request.Filter, "_geoRadius") {
			if stringListed("name", request.AttributesToRetrieve) &&
				stringListed("description", request.AttributesToRetrieve) &&
				stringListed("gpx", request.AttributesToRetrieve) &&
				stringListed("_geo", request.AttributesToRetrieve) &&
				!stringListed("polyline", request.AttributesToRetrieve) &&
				!stringListed("_geojson", request.AttributesToRetrieve) {
				sawTimedProduct = true
			}
		}
	}
	if !sawTimedProduct {
		t.Fatalf("saw timed product=%t; requests=%+v", sawTimedProduct, requests)
	}
}

func TestMeasureGeoJSONDirectProductWorkloadStartsConcurrentClockBeforeWorkers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/indexes/"+benchmarkIndexUID+"/search" {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"hits":               []map[string]any{geoJSONDirectTestProductHit()},
			"estimatedTotalHits": 1,
			"totalHits":          1,
			"page":               1,
			"hitsPerPage":        10,
			"totalPages":         1,
			"processingTimeMs":   1,
		})
	}))
	defer server.Close()

	const setupDelay = 75 * time.Millisecond
	query := ProductQuery{
		Geo: &ProductGeoRadius{
			Center:       Coordinate{Lat: 46.8, Lon: 8.2},
			RadiusMeters: 500,
		},
		Sort:    ProductSortRelevance,
		Page:    1,
		PerPage: 10,
	}
	audit := &geoJSONDirectPlanAudit{
		plan: GeoJSONQueryPlan{
			ID: "r125-exact", RadiusMode: geoJSONRadiusExact, Resolution: 125, RadiusFormula: "r",
		},
		report: &GeoJSONDirectProductWorkloadReport{
			Status: "audited", Accuracy: GeoJSONDirectAccuracy{Status: "passed"},
			CountDiagnostics: GeoJSONDirectProductCountReport{Status: "passed"},
		},
		prepared: []geoJSONDirectPreparedProductCase{{
			benchmark:        productBenchmarkCase{Name: "timing", Query: query, Radius: 500},
			directPage:       []string{"trail-1"},
			directTotalHits:  1,
			directTotalPages: 1,
			directPageNumber: 1,
			directPerPage:    10,
		}},
		beforeConcurrentWorkers: func() { time.Sleep(setupDelay) },
	}
	report, err := measureGeoJSONDirectProductWorkload(
		context.Background(), meilisearch.New(server.URL), audit, 1, 1,
	)
	if err != nil {
		t.Fatal(err)
	}
	if report.ConcurrentWallMS < milliseconds(setupDelay) {
		t.Fatalf("concurrent wall %.2f ms excludes %.2f ms pre-worker setup", report.ConcurrentWallMS, milliseconds(setupDelay))
	}
	if report.ThroughputQPS > 1000/milliseconds(setupDelay) {
		t.Fatalf("throughput %.2f QPS ignores pre-worker setup in its denominator", report.ThroughputQPS)
	}
}

func TestGeoJSONDirectCountFailureFailsWarmAndCannotBeSelected(t *testing.T) {
	query := ProductQuery{
		Geo: &ProductGeoRadius{
			Center:       Coordinate{Lat: 46.8, Lon: 8.2},
			RadiusMeters: 500,
		},
		Sort:    ProductSortRelevance,
		Page:    1,
		PerPage: 10,
	}
	audit := &geoJSONDirectPlanAudit{
		plan: GeoJSONQueryPlan{
			ID: "r100-exact", RadiusMode: geoJSONRadiusExact, Resolution: 100, RadiusFormula: "r",
		},
		report: &GeoJSONDirectProductWorkloadReport{
			Status:           "count_failed",
			Accuracy:         GeoJSONDirectAccuracy{Status: "passed"},
			CountDiagnostics: GeoJSONDirectProductCountReport{Status: "failed", MaximumP95RelativeError: 0.01},
		},
		prepared: []geoJSONDirectPreparedProductCase{{
			benchmark:        productBenchmarkCase{Name: "count-failed", Query: query, Radius: 500},
			directPage:       []string{"trail-1"},
			directTotalHits:  1,
			directTotalPages: 1,
			directPageNumber: 1,
			directPerPage:    10,
		}},
		search: func(
			context.Context,
			ProductQuery,
			GeoJSONQueryPlan,
			bool,
			bool,
			int64,
			[]string,
			[]string,
		) (geoJSONDirectSearchResult, error) {
			return geoJSONDirectSearchResult{
				IDs: []string{"trail-1"}, TotalHits: 1, TotalPages: 1, Page: 1, HitsPerPage: 10,
				HTTPRequests: 1, ShardQueries: 1, ShardFanout: 1,
			}, nil
		},
	}

	warm, err := measureGeoJSONDirectProductWorkload(context.Background(), nil, audit, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if warm.Accuracy.Status != "passed" || warm.CountDiagnostics.Status != "failed" || warm.Status != "failed" {
		t.Fatalf("count-failed warm qualification = %+v", warm)
	}
	direct := &GeoJSONDirectReport{ProductAudits: []GeoJSONDirectProductAuditReport{{
		PlanID: "r100-exact", Resolution: 100, Status: "count_failed", WarmWorkload: warm,
	}}}
	if selectGeoJSONDirectWarmPlan(direct, false) || direct.Plan != nil || direct.SelectionStatus != "warm_not_qualified" {
		t.Fatalf("count-failed plan became selectable: %+v", direct)
	}
}

func TestBenchmarkGeoJSONDirectProductWorkloadMeasuresWarmSearchAfterAccuracyFailure(t *testing.T) {
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"hits":             []map[string]any{},
			"totalHits":        0,
			"page":             1,
			"hitsPerPage":      10,
			"totalPages":       0,
			"processingTimeMs": 1,
		})
	}))
	defer server.Close()

	dataset := Dataset{
		Trails: []Trail{{
			ID: "trail-1", Scenario: "alpine_dense",
			Parts: [][]Coordinate{{{Lat: 46.8, Lon: 8.2}, {Lat: 46.801, Lon: 8.201}}},
		}},
		QueryPoints: []QueryPoint{{
			ID: "query-1", Scenario: "alpine_dense", Point: Coordinate{Lat: 46.8, Lon: 8.2},
		}},
	}
	documents := generateProductDocuments(dataset)
	documents[0].Public = true
	report, err := benchmarkGeoJSONDirectProductWorkload(
		context.Background(), meilisearch.New(server.URL), dataset, []float64{500},
		1, 1, documents,
		GeoJSONQueryPlan{ID: "r100-exact", RadiusMode: geoJSONRadiusExact, Resolution: 100, RadiusFormula: "r"},
		GeoJSONDirectConfig{
			Enabled: true, Resolutions: []int{100}, BoundaryToleranceMeters: 50,
			MinimumRecall: 0.99, MinimumPrecision: 0.99, MinimumTop10Recall: 0.99,
			MinimumComparablePageRate: 0.99, MinimumComparablePagePrecision: 0.99,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if report.Accuracy.Status != "failed" || report.Status != "failed" {
		t.Fatalf("accuracy/workload status = %s/%s, want failed/failed", report.Accuracy.Status, report.Status)
	}
	if report.Samples != 1 || report.ExpectedSamples != 1 || report.TotalLatency.P50 <= 0 {
		t.Fatalf("warm diagnostics were not collected after the failed audit: %+v", report)
	}
	if requests.Load() != 3 {
		t.Fatalf("requests = %d, want audit IDs + audit page + warm page", requests.Load())
	}
}

func TestBenchmarkGeoJSONDirectProductCandidatesAuditsAllPlansAndSelectsProductLatency(t *testing.T) {
	type requestBody struct {
		Filter      string `json:"filter"`
		Limit       int64  `json:"limit"`
		HitsPerPage int64  `json:"hitsPerPage"`
		Page        int64  `json:"page"`
	}
	var lock sync.Mutex
	planRequests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var captured requestBody
		if err := json.NewDecoder(request.Body).Decode(&captured); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		planID := "unknown"
		switch {
		case strings.Contains(captured.Filter, ", 100)"):
			planID = "r100-exact"
		case strings.Contains(captured.Filter, ", 125)"):
			planID = "r125-exact"
		}
		lock.Lock()
		planRequests[planID]++
		lock.Unlock()
		if planID == "r100-exact" && captured.HitsPerPage > 0 {
			time.Sleep(20 * time.Millisecond)
		}
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"hits":               []map[string]any{geoJSONDirectTestProductHit()},
			"estimatedTotalHits": 1,
			"totalHits":          1,
			"page":               max(int(captured.Page), 1),
			"hitsPerPage":        max(int(captured.HitsPerPage), 10),
			"totalPages":         1,
			"processingTimeMs":   1,
		})
	}))
	defer server.Close()

	dataset := Dataset{
		Trails: []Trail{{
			ID: "trail-1", Scenario: "alpine_dense",
			Parts: [][]Coordinate{{{Lat: 46.8, Lon: 8.2}, {Lat: 46.801, Lon: 8.201}}},
		}},
		QueryPoints: []QueryPoint{{
			ID: "query-1", Scenario: "alpine_dense", Point: Coordinate{Lat: 46.8, Lon: 8.2},
		}},
	}
	documents := generateProductDocuments(dataset)
	documents[0].Public = true
	passed := GeoJSONDirectAccuracy{Status: "passed"}
	sweep := &GeoJSONSweepReport{Variants: []GeoJSONVariantReport{
		{
			Plan:     geoJSONDirectPlan(100, 50),
			DirectUX: &passed, CandidateLatency: LatencyStats{P95: 1},
		},
		{
			Plan:     geoJSONDirectPlan(125, 50),
			DirectUX: &passed, CandidateLatency: LatencyStats{P95: 50},
		},
	}}
	config := GeoJSONDirectConfig{
		Enabled: true, Resolutions: []int{100, 125}, BoundaryToleranceMeters: 50,
		MinimumRecall: 0.99, MinimumPrecision: 0.99, MinimumTop10Recall: 0.99,
		MinimumComparablePageRate: 0.99, MinimumComparablePagePrecision: 0.99,
	}
	direct, err := benchmarkGeoJSONDirectProductCandidates(
		context.Background(), meilisearch.New(server.URL), dataset, []float64{500},
		1, 1, documents, sweep, config,
	)
	if err != nil {
		t.Fatal(err)
	}
	if direct.SelectedID != "r125-exact" || direct.SelectionStatus != "qualified" {
		t.Fatalf("selection = %s/%s, want product-faster r125 qualified; report=%+v", direct.SelectedID, direct.SelectionStatus, direct)
	}
	wantPlan := geoJSONDirectPlan(125, 50)
	if direct.Plan == nil || !reflect.DeepEqual(*direct.Plan, wantPlan) {
		t.Fatalf("selected plan = %+v, want original %+v", direct.Plan, wantPlan)
	}
	if len(direct.ProductAudits) != 2 || direct.ProductAudits[0].WarmWorkload == nil ||
		direct.ProductAudits[1].WarmWorkload == nil || direct.ProductWorkload == nil || direct.ProductWorkload.Status != "passed" {
		t.Fatalf("direct report = %+v", direct)
	}
	if direct.ProductOracleMS <= 0 || direct.PlanAuditWallMS <= 0 {
		t.Fatalf("untimed phases were not measured: oracle=%v audit=%v", direct.ProductOracleMS, direct.PlanAuditWallMS)
	}
	lock.Lock()
	defer lock.Unlock()
	if planRequests["r100-exact"] != 3 || planRequests["r125-exact"] != 3 || planRequests["unknown"] != 0 {
		t.Fatalf("requests by plan = %v, want both audited and warmed", planRequests)
	}
}

func TestSelectGeoJSONDirectWarmPlanFallsBackAcrossWarmAndPerformanceGates(t *testing.T) {
	accuracy := GeoJSONDirectAccuracy{Status: "passed"}
	direct := &GeoJSONDirectReport{ProductAudits: []GeoJSONDirectProductAuditReport{
		{
			PlanID: "r100-exact", Resolution: 100, Status: "audited",
			ProductPageLatency: LatencyStats{P95: 1, P99: 2},
			WarmWorkload: &GeoJSONDirectProductWorkloadReport{
				Status: "passed", Samples: 10, ExpectedSamples: 10, UnstableWarmSamples: 1,
				TotalLatency: LatencyStats{P95: 10, P99: 20}, Accuracy: accuracy,
			},
		},
		{
			PlanID: "r125-exact", Resolution: 125, Status: "audited",
			ProductPageLatency: LatencyStats{P95: 3, P99: 4},
			WarmWorkload: &GeoJSONDirectProductWorkloadReport{
				Status: "passed", Samples: 10, ExpectedSamples: 10,
				TotalLatency: LatencyStats{P95: 30, P99: 40}, Accuracy: accuracy,
			},
		},
	}}
	if !selectGeoJSONDirectWarmPlan(direct, false) || direct.SelectedID != "r125-exact" {
		t.Fatalf("warm fallback selected %q, want stable r125", direct.SelectedID)
	}
	if strings.Contains(direct.SelectionRule, "active performance target") {
		t.Fatalf("non-target selection rule claims an active target: %q", direct.SelectionRule)
	}

	direct.ProductAudits[0].WarmWorkload.UnstableWarmSamples = 0
	direct.ProductAudits[0].Performance = &PerformanceReport{Status: "failed"}
	direct.ProductAudits[1].Performance = &PerformanceReport{Status: "passed"}
	if !selectGeoJSONDirectWarmPlan(direct, true) || direct.SelectedID != "r125-exact" {
		t.Fatalf("performance fallback selected %q, want passing r125", direct.SelectedID)
	}
	if !strings.Contains(direct.SelectionRule, "active performance target") {
		t.Fatalf("target selection rule omits the active target: %q", direct.SelectionRule)
	}
	direct.ProductAudits[1].Performance.Status = "failed"
	if selectGeoJSONDirectWarmPlan(direct, true) || direct.SelectionStatus != "performance_not_qualified" || direct.Plan != nil {
		t.Fatalf("all-failed performance selection = %+v", direct)
	}
}

func TestSelectGeoJSONDirectWarmPlanPreservesCompletePlanMetadata(t *testing.T) {
	plan := geoJSONDirectPlan(100, 50)
	accuracy := GeoJSONDirectAccuracy{Status: "passed"}
	direct := &GeoJSONDirectReport{ProductAudits: []GeoJSONDirectProductAuditReport{{
		Plan:       &plan,
		PlanID:     plan.ID,
		Resolution: plan.Resolution,
		Status:     "audited",
		WarmWorkload: &GeoJSONDirectProductWorkloadReport{
			Status: "passed", Samples: 10, ExpectedSamples: 10, Accuracy: accuracy,
		},
	}}}

	if !selectGeoJSONDirectWarmPlan(direct, false) {
		t.Fatalf("complete plan was not selected: %+v", direct)
	}
	if direct.Plan == nil || !reflect.DeepEqual(*direct.Plan, plan) {
		t.Fatalf("selected plan = %+v, want original %+v", direct.Plan, plan)
	}
	if direct.Plan.IndexSimplificationMeters != 50 ||
		!strings.Contains(direct.Plan.RadiusFormula, "not exact geometry") {
		t.Fatalf("selected plan lost S50 or radius semantics: %+v", direct.Plan)
	}
}

func stringListed(value string, values []string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func geoJSONDirectTestProductHit() map[string]any {
	return map[string]any{
		"id":                         "trail-1",
		"author":                     "actor-01",
		"author_name":                "wanderer 01",
		"author_avatar":              "avatar-01.webp",
		"name":                       "alpine dense trail",
		"description":                "forest summit",
		"location":                   "Bernese Alps",
		"distance":                   1000,
		"elevation_gain":             120,
		"elevation_loss":             80,
		"duration":                   900,
		"difficulty":                 1,
		"category":                   "Hiking",
		"category_id":                "hike",
		"category_icon":              "hiking.svg",
		"subcategory_id":             "alpine",
		"is_federated":               false,
		"federated_category_name":    "",
		"federated_subcategory_name": "",
		"completed":                  true,
		"date":                       1_609_372_800,
		"created":                    1_609_459_200,
		"public":                     true,
		"thumbnail":                  "trail-1.webp",
		"domain":                     "",
		"gpx":                        "trail-1.gpx",
		"tags":                       []string{"alpine_dense"},
		"like_count":                 2,
		"shares":                     []string{},
		"iri":                        "",
		"bounding_box_diagonal":      1400,
		"_geo":                       map[string]float64{"lat": 46.8, "lng": 8.2},
	}
}
