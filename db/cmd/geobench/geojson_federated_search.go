package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const geoJSONCapacitySearchScope = "search-only: one fixed-resolution direct product query uses one exhaustive federated /multi-search request across every immutable capacity shard; incremental updates, additions, deletes, and safe-hybrid queries are intentionally skipped"

// geoJSONFederatedSearchClient deliberately uses the HTTP API instead of the
// Meilisearch Go SDK. The SDK version used by Wanderer does not yet expose the
// exhaustive page/hitsPerPage fields on the top-level federation object, while
// Meilisearch 1.53 does. One logical product query therefore remains one real
// /multi-search request, including the exact global count and page merge.
type geoJSONFederatedSearchClient struct {
	endpoint   string
	httpClient *http.Client
	indexUIDs  []string
}

type geoJSONFederatedSearchRequest struct {
	Federation geoJSONFederatedPagination `json:"federation"`
	Queries    []geoJSONFederatedQuery    `json:"queries"`
}

type geoJSONFederatedPagination struct {
	Page        int64 `json:"page"`
	HitsPerPage int64 `json:"hitsPerPage"`
}

type geoJSONFederatedQuery struct {
	IndexUID             string   `json:"indexUid"`
	Query                string   `json:"q"`
	Filter               string   `json:"filter"`
	AttributesToRetrieve []string `json:"attributesToRetrieve"`
	Sort                 []string `json:"sort,omitempty"`
	MatchingStrategy     string   `json:"matchingStrategy"`
}

type geoJSONFederatedSearchResponse struct {
	Hits             []map[string]json.RawMessage `json:"hits"`
	ProcessingTimeMS int64                        `json:"processingTimeMs"`
	TotalHits        int64                        `json:"totalHits"`
	TotalPages       int64                        `json:"totalPages"`
	Page             int64                        `json:"page"`
	HitsPerPage      int64                        `json:"hitsPerPage"`
}

type geoJSONFederatedHTTPResult struct {
	response      geoJSONFederatedSearchResponse
	requestBytes  int64
	responseBytes int64
	wallDuration  time.Duration
}

func newGeoJSONFederatedSearchClient(endpoint string, indexUIDs []string) (*geoJSONFederatedSearchClient, error) {
	parsed, err := url.Parse(strings.TrimRight(endpoint, "/"))
	if err != nil {
		return nil, fmt.Errorf("parse Meilisearch endpoint: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("Meilisearch endpoint must use http or https: %q", endpoint)
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("Meilisearch endpoint has no host: %q", endpoint)
	}
	if len(indexUIDs) < 2 {
		return nil, fmt.Errorf("federated GeoJSON search requires at least two indexes, got %d", len(indexUIDs))
	}
	uids := append([]string(nil), indexUIDs...)
	seen := make(map[string]struct{}, len(uids))
	for index, uid := range uids {
		if strings.TrimSpace(uid) == "" {
			return nil, fmt.Errorf("federated GeoJSON index %d has an empty UID", index)
		}
		if _, duplicate := seen[uid]; duplicate {
			return nil, fmt.Errorf("federated GeoJSON index UID %q is duplicated", uid)
		}
		seen[uid] = struct{}{}
	}
	return &geoJSONFederatedSearchClient{
		endpoint:   parsed.String(),
		httpClient: http.DefaultClient,
		indexUIDs:  uids,
	}, nil
}

func (client *geoJSONFederatedSearchClient) execute(
	ctx context.Context,
	requestBody geoJSONFederatedSearchRequest,
) (geoJSONFederatedHTTPResult, error) {
	started := time.Now()
	if client == nil || client.httpClient == nil {
		return geoJSONFederatedHTTPResult{}, fmt.Errorf("federated GeoJSON search client is nil")
	}
	encodedRequest, err := json.Marshal(requestBody)
	if err != nil {
		return geoJSONFederatedHTTPResult{}, fmt.Errorf("encode federated GeoJSON search: %w", err)
	}
	httpRequest, err := http.NewRequestWithContext(
		ctx, http.MethodPost, client.endpoint+"/multi-search", bytes.NewReader(encodedRequest),
	)
	if err != nil {
		return geoJSONFederatedHTTPResult{}, fmt.Errorf("create federated GeoJSON search: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpResponse, err := client.httpClient.Do(httpRequest)
	if err != nil {
		return geoJSONFederatedHTTPResult{}, fmt.Errorf("execute federated GeoJSON search: %w", err)
	}
	defer httpResponse.Body.Close()
	const maximumResponseBytes = 128 << 20
	wireResponse, err := io.ReadAll(io.LimitReader(httpResponse.Body, maximumResponseBytes+1))
	if err != nil {
		return geoJSONFederatedHTTPResult{}, fmt.Errorf("read federated GeoJSON search: %w", err)
	}
	if len(wireResponse) > maximumResponseBytes {
		return geoJSONFederatedHTTPResult{}, fmt.Errorf("federated GeoJSON search response exceeds %d bytes", maximumResponseBytes)
	}
	if httpResponse.StatusCode != http.StatusOK {
		return geoJSONFederatedHTTPResult{}, fmt.Errorf(
			"federated GeoJSON search returned HTTP %d: %s",
			httpResponse.StatusCode, strings.TrimSpace(string(wireResponse)),
		)
	}
	var response geoJSONFederatedSearchResponse
	if err := json.Unmarshal(wireResponse, &response); err != nil {
		return geoJSONFederatedHTTPResult{}, fmt.Errorf("decode federated GeoJSON search: %w", err)
	}
	return geoJSONFederatedHTTPResult{
		response:      response,
		requestBytes:  int64(len(encodedRequest)),
		responseBytes: int64(len(wireResponse)),
		wallDuration:  time.Since(started),
	}, nil
}

func validateGeoJSONFederatedPagination(response geoJSONFederatedSearchResponse, page, hitsPerPage int64) error {
	if response.Page != page || response.HitsPerPage != hitsPerPage {
		return fmt.Errorf(
			"federated pagination metadata page/per-page = %d/%d, want %d/%d",
			response.Page, response.HitsPerPage, page, hitsPerPage,
		)
	}
	if response.TotalHits < int64(len(response.Hits)) {
		return fmt.Errorf(
			"federated totalHits %d is smaller than returned hits %d", response.TotalHits, len(response.Hits),
		)
	}
	wantPages := int64(productTotalPages(int(response.TotalHits), int(hitsPerPage)))
	if response.TotalPages != wantPages {
		return fmt.Errorf(
			"federated totalPages %d does not match totalHits/hitsPerPage (%d)", response.TotalPages, wantPages,
		)
	}
	return nil
}

func (client *geoJSONFederatedSearchClient) searchProduct(
	ctx context.Context,
	query ProductQuery,
	plan GeoJSONQueryPlan,
	includeGeo, allResults bool,
	fullLimit int64,
	attributes, sortValues []string,
) (geoJSONDirectSearchResult, error) {
	if client == nil || client.httpClient == nil {
		return geoJSONDirectSearchResult{}, fmt.Errorf("federated GeoJSON search client is nil")
	}
	if allResults && fullLimit < 1 {
		return geoJSONDirectSearchResult{}, fmt.Errorf("federated complete-result limit must be positive: %d", fullLimit)
	}

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

	page := int64(query.Page)
	hitsPerPage := int64(query.PerPage)
	if allResults {
		page = 1
		hitsPerPage = fullLimit
	}
	requestBody := geoJSONFederatedSearchRequest{
		Federation: geoJSONFederatedPagination{Page: page, HitsPerPage: hitsPerPage},
		Queries:    make([]geoJSONFederatedQuery, len(client.indexUIDs)),
	}
	for index, uid := range client.indexUIDs {
		requestBody.Queries[index] = geoJSONFederatedQuery{
			IndexUID:             uid,
			Query:                query.Text,
			Filter:               filter,
			AttributesToRetrieve: append([]string(nil), attributes...),
			Sort:                 append([]string(nil), sortValues...),
			MatchingStrategy:     "all",
		}
	}
	httpResult, err := client.execute(ctx, requestBody)
	if err != nil {
		return geoJSONDirectSearchResult{}, fmt.Errorf("federated GeoJSON product search: %w", err)
	}
	response := httpResult.response
	if err := validateGeoJSONFederatedPagination(response, page, hitsPerPage); err != nil {
		return geoJSONDirectSearchResult{}, err
	}

	ids, productHits, err := cleanGeoJSONFederatedHits(response.Hits)
	if err != nil {
		return geoJSONDirectSearchResult{}, err
	}

	var productResponse []byte
	if allResults {
		productResponse, err = json.Marshal(productHits)
	} else {
		productResponse, err = marshalMeiliProductPagePayload(
			productHits,
			int(response.TotalHits),
			int(response.TotalPages),
			int(response.Page),
			int(response.HitsPerPage),
		)
	}
	if err != nil {
		return geoJSONDirectSearchResult{}, fmt.Errorf("materialize federated direct product response: %w", err)
	}
	result := geoJSONDirectSearchResult{
		IDs:                ids,
		TotalHits:          int(response.TotalHits),
		TotalPages:         int(response.TotalPages),
		Page:               int(response.Page),
		HitsPerPage:        int(response.HitsPerPage),
		EngineProcessingMS: float64(response.ProcessingTimeMS),
		ResponseBytes:      int64(len(productResponse)),
		RequestBytes:       httpResult.requestBytes,
		WireResponseBytes:  httpResult.responseBytes,
		HTTPRequests:       1,
		ShardQueries:       len(client.indexUIDs),
		ShardFanout:        len(client.indexUIDs),
		Truncated:          allResults && response.TotalHits > int64(len(ids)),
	}
	result.WallDuration = httpResult.wallDuration
	return result, nil
}

type geoJSONFederatedSpatialResult struct {
	IDs                map[string]struct{}
	TotalHits          int
	WallDuration       time.Duration
	EngineProcessingMS float64
	RequestBytes       int64
	ResponseBytes      int64
	HTTPRequests       int
	ShardQueries       int
	ShardFanout        int
	Truncated          bool
}

func (client *geoJSONFederatedSearchClient) searchSpatial(
	ctx context.Context,
	point Coordinate,
	radius float64,
	plan GeoJSONQueryPlan,
	fullLimit int64,
) (geoJSONFederatedSpatialResult, error) {
	if fullLimit < 1 {
		return geoJSONFederatedSpatialResult{}, fmt.Errorf("federated spatial-result limit must be positive: %d", fullLimit)
	}
	candidateRadius, err := geoJSONCandidateRadius(radius, plan)
	if err != nil {
		return geoJSONFederatedSpatialResult{}, err
	}
	filter := geoRadiusFilter(point, candidateRadius, plan.Resolution)
	requestBody := geoJSONFederatedSearchRequest{
		Federation: geoJSONFederatedPagination{Page: 1, HitsPerPage: fullLimit},
		Queries:    make([]geoJSONFederatedQuery, len(client.indexUIDs)),
	}
	for index, uid := range client.indexUIDs {
		requestBody.Queries[index] = geoJSONFederatedQuery{
			IndexUID:             uid,
			Filter:               filter,
			AttributesToRetrieve: []string{"id"},
			Sort:                 []string{"id:asc"},
			MatchingStrategy:     "all",
		}
	}
	httpResult, err := client.execute(ctx, requestBody)
	if err != nil {
		return geoJSONFederatedSpatialResult{}, fmt.Errorf("federated GeoJSON spatial search: %w", err)
	}
	response := httpResult.response
	if err := validateGeoJSONFederatedPagination(response, 1, fullLimit); err != nil {
		return geoJSONFederatedSpatialResult{}, err
	}
	ids, _, err := cleanGeoJSONFederatedHits(response.Hits)
	if err != nil {
		return geoJSONFederatedSpatialResult{}, err
	}
	result := geoJSONFederatedSpatialResult{
		IDs:                make(map[string]struct{}, len(ids)),
		TotalHits:          int(response.TotalHits),
		WallDuration:       httpResult.wallDuration,
		EngineProcessingMS: float64(response.ProcessingTimeMS),
		RequestBytes:       httpResult.requestBytes,
		ResponseBytes:      httpResult.responseBytes,
		HTTPRequests:       1,
		ShardQueries:       len(client.indexUIDs),
		ShardFanout:        len(client.indexUIDs),
		Truncated:          response.TotalHits > int64(len(ids)),
	}
	for _, id := range ids {
		result.IDs[id] = struct{}{}
	}
	return result, nil
}

func auditGeoJSONCapacitySpatial(
	ctx context.Context,
	client *geoJSONFederatedSearchClient,
	dataset Dataset,
	radii []float64,
	oracle *accuracyOracle,
	plan GeoJSONQueryPlan,
	config GeoJSONDirectConfig,
) (*GeoJSONDirectSpatialAuditReport, error) {
	report := &GeoJSONDirectSpatialAuditReport{Status: "failed"}
	if oracle == nil {
		err := fmt.Errorf("capacity-created spatial audit requires the exact accuracy oracle")
		report.Error = err.Error()
		return report, err
	}
	if client == nil {
		err := fmt.Errorf("capacity-created spatial audit requires a federated search client")
		report.Error = err.Error()
		return report, err
	}
	accumulator := newGeoJSONDirectAccuracyAccumulator(config)
	latencies := make([]float64, 0, len(dataset.QueryPoints)*len(radii))
	engine := make([]float64, 0, cap(latencies))
	fullLimit := int64(len(dataset.Trails) + 1)
	for _, queryPoint := range dataset.QueryPoints {
		for _, radius := range radii {
			result, err := client.searchSpatial(ctx, queryPoint.Point, radius, plan, fullLimit)
			if err != nil {
				report.Error = fmt.Sprintf("query %s at %.0f m: %v", queryPoint.ID, radius, err)
				return report, fmt.Errorf("capacity-created spatial audit: %s", report.Error)
			}
			observation, err := oracle.observeGeoJSONDirect(
				queryPoint, radius, result.IDs, config, result.Truncated,
			)
			if err != nil {
				report.Error = err.Error()
				return report, err
			}
			accumulator.add(observation)
			report.Cases++
			report.HTTPRequests += result.HTTPRequests
			report.ShardQueries += result.ShardQueries
			report.RequestBytes += result.RequestBytes
			report.ResponseBytes += result.ResponseBytes
			report.ShardFanout = max(report.ShardFanout, result.ShardFanout)
			latencies = append(latencies, milliseconds(result.WallDuration))
			engine = append(engine, result.EngineProcessingMS)
		}
	}
	report.CandidateLatency = latencyStats(latencies)
	report.EngineProcessing = latencyStats(engine)
	report.Accuracy = accumulator.finish(config, false)
	if report.Cases != len(dataset.QueryPoints)*len(radii) {
		report.Error = fmt.Sprintf("spatial audit completed %d cases, want %d", report.Cases, len(dataset.QueryPoints)*len(radii))
		return report, fmt.Errorf("capacity-created spatial audit: %s", report.Error)
	}
	report.Status = report.Accuracy.Status
	if report.Status != "passed" {
		report.Error = fmt.Sprintf("fixed r%d full spatial matrix did not pass the UX accuracy gate", plan.Resolution)
	}
	return report, nil
}

func cleanGeoJSONFederatedHits(hits []map[string]json.RawMessage) ([]string, []map[string]json.RawMessage, error) {
	ids := make([]string, 0, len(hits))
	seen := make(map[string]struct{}, len(hits))
	productHits := make([]map[string]json.RawMessage, len(hits))
	for index, hit := range hits {
		rawID, ok := hit["id"]
		if !ok {
			return nil, nil, fmt.Errorf("federated direct product search hit has no id")
		}
		var id string
		if err := json.Unmarshal(rawID, &id); err != nil {
			return nil, nil, fmt.Errorf("decode federated direct product hit id: %w", err)
		}
		if _, duplicate := seen[id]; duplicate {
			return nil, nil, fmt.Errorf("federated direct product search returned duplicate id %q", id)
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
		clean := make(map[string]json.RawMessage, len(hit))
		for field, value := range hit {
			if field == "_federation" {
				continue
			}
			clean[field] = value
		}
		productHits[index] = clean
	}
	return ids, productHits, nil
}

func benchmarkGeoJSONCapacityCreatedSearch(
	ctx context.Context,
	container *meiliContainer,
	indexUIDs []string,
	dataset Dataset,
	radii []float64,
	repetitions, queryClients int,
	documents []ProductQueryDocument,
	oracle *accuracyOracle,
	config GeoJSONDirectConfig,
	indexSimplificationMeters float64,
) (*GeoJSONDirectReport, error) {
	if container == nil {
		err := fmt.Errorf("capacity-created search requires a Meilisearch container")
		return &GeoJSONDirectReport{SelectionStatus: "failed", Error: err.Error()}, err
	}
	if len(config.Resolutions) != 1 {
		err := fmt.Errorf("capacity-created search requires exactly one direct GeoJSON resolution, got %v", config.Resolutions)
		return &GeoJSONDirectReport{SelectionStatus: "failed", Error: err.Error()}, err
	}
	client, err := newGeoJSONFederatedSearchClient(container.url, indexUIDs)
	if err != nil {
		return &GeoJSONDirectReport{SelectionStatus: "failed", Error: err.Error()}, err
	}
	resolution := config.Resolutions[0]
	plan := geoJSONDirectPlan(resolution, indexSimplificationMeters)
	report := &GeoJSONDirectReport{
		SelectionRule:   fmt.Sprintf("fixed r%d capacity-shard measurement; full spatial matrix and federated product page/semantics must pass the configured UX gates; oracle count relative p95 must be at most 1%% while absolute and per-case deltas remain diagnostic", resolution),
		SelectedID:      plan.ID,
		SelectionStatus: "not_qualified",
		Plan:            &plan,
	}
	report.SpatialAudit, err = auditGeoJSONCapacitySpatial(
		ctx, client, dataset, radii, oracle, plan, config,
	)
	if err != nil {
		report.SelectionStatus = "failed"
		report.Error = err.Error()
		return report, err
	}
	oracleStarted := time.Now()
	prepared, prepareErr := prepareGeoJSONDirectProductBenchmarkWithSearch(
		ctx,
		dataset,
		radii,
		documents,
		client.searchProduct,
	)
	oracleMS := milliseconds(time.Since(oracleStarted))
	workload, auditWallMS, err := benchmarkGeoJSONDirectProductWorkloadPreparedTimed(
		ctx, nil, prepared, prepareErr, repetitions, queryClients, plan, config, container,
	)
	report.ProductOracleMS = oracleMS
	report.PlanAuditWallMS = auditWallMS
	report.ProductWorkload = workload
	if workload != nil {
		accuracy := workload.Accuracy
		report.Accuracy = &accuracy
	}
	if err != nil {
		report.SelectionStatus = "failed"
		report.Error = err.Error()
		return report, err
	}
	if report.SpatialAudit == nil || report.SpatialAudit.Status != "passed" ||
		workload == nil || workload.Status != "passed" || workload.Accuracy.Status != "passed" {
		report.Error = fmt.Sprintf("fixed r%d federated spatial and product workloads did not both pass their UX gates", resolution)
		return report, nil
	}
	if workload.Accuracy.UnsupportedSortCases > 0 {
		report.SelectionStatus = "qualified_with_unsupported_proximity_sort"
	} else {
		report.SelectionStatus = "qualified"
	}
	return report, nil
}
