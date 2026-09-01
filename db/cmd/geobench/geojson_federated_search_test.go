package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestGeoJSONFederatedSearchUsesOneExhaustiveGlobalPageRequest(t *testing.T) {
	var captured geoJSONFederatedSearchRequest
	var capturedBytes int
	hit := geoJSONDirectTestProductHit()
	hit["id"] = "trail-3"
	hit["_federation"] = map[string]any{
		"indexUid":             "shard-1",
		"queriesPosition":      1,
		"weightedRankingScore": 1,
	}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/multi-search" {
			http.NotFound(writer, request)
			return
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		capturedBytes = len(body)
		if err := json.Unmarshal(body, &captured); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"hits":             []map[string]any{hit},
			"processingTimeMs": 7,
			"totalHits":        3,
			"totalPages":       2,
			"page":             2,
			"hitsPerPage":      2,
		})
	}))
	defer server.Close()

	client, err := newGeoJSONFederatedSearchClient(server.URL, []string{"shard-0", "shard-1"})
	if err != nil {
		t.Fatal(err)
	}
	federated := true
	query := ProductQuery{
		ActorID: "actor-7",
		Text:    "alpine forest",
		Geo: &ProductGeoRadius{
			Center:       Coordinate{Lat: 46.8, Lon: 8.2},
			RadiusMeters: 100_000,
		},
		Filters: ProductQueryFilters{
			CategoryIDs: []string{"hike"},
			Federated:   &federated,
		},
		Sort:    ProductSortCreatedDesc,
		Page:    2,
		PerPage: 2,
	}
	result, err := client.searchProduct(
		context.Background(),
		query,
		GeoJSONQueryPlan{ID: "r100-exact", RadiusMode: geoJSONRadiusExact, Resolution: 100},
		true,
		false,
		0,
		geoJSONDirectProductAttributes,
		[]string{"created:desc", "id:asc"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result.IDs, []string{"trail-3"}) ||
		result.TotalHits != 3 || result.TotalPages != 2 || result.Page != 2 || result.HitsPerPage != 2 {
		t.Fatalf("federated result = %+v", result)
	}
	if result.HTTPRequests != 1 || result.ShardFanout != 2 || result.ShardQueries != 2 {
		t.Fatalf("request/fanout accounting = %+v", result)
	}
	if result.RequestBytes != int64(capturedBytes) || result.RequestBytes == 0 || result.WireResponseBytes == 0 || result.ResponseBytes == 0 {
		t.Fatalf("byte accounting request/wire/product = %d/%d/%d, captured %d", result.RequestBytes, result.WireResponseBytes, result.ResponseBytes, capturedBytes)
	}
	if captured.Federation.Page != 2 || captured.Federation.HitsPerPage != 2 || len(captured.Queries) != 2 {
		t.Fatalf("federated request = %+v", captured)
	}
	for index, request := range captured.Queries {
		if request.IndexUID != []string{"shard-0", "shard-1"}[index] || request.Query != query.Text ||
			request.MatchingStrategy != "all" || !reflect.DeepEqual(request.Sort, []string{"created:desc", "id:asc"}) {
			t.Fatalf("subquery %d = %+v", index, request)
		}
		for _, predicate := range []string{
			`(public = true OR author = "actor-7" OR shares = "actor-7")`,
			`category_id IN ["hike"]`,
			`is_federated = true`,
			`_geoRadius(46.8000000, 8.2000000, 100000, 100)`,
		} {
			if !strings.Contains(request.Filter, predicate) {
				t.Errorf("subquery %d filter %q lacks %q", index, request.Filter, predicate)
			}
		}
	}
}

func TestCleanGeoJSONFederatedHitsStripsMetadataAndRejectsDuplicates(t *testing.T) {
	hits := []map[string]json.RawMessage{
		{"id": json.RawMessage(`"a"`), "name": json.RawMessage(`"A"`), "_federation": json.RawMessage(`{"indexUid":"s0"}`)},
		{"id": json.RawMessage(`"b"`), "name": json.RawMessage(`"B"`), "_federation": json.RawMessage(`{"indexUid":"s1"}`)},
	}
	ids, clean, err := cleanGeoJSONFederatedHits(hits)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ids, []string{"a", "b"}) {
		t.Fatalf("ids = %v", ids)
	}
	for index := range clean {
		if _, exists := clean[index]["_federation"]; exists {
			t.Fatalf("clean hit %d retains federation metadata: %v", index, clean[index])
		}
		if _, exists := hits[index]["_federation"]; !exists {
			t.Fatalf("cleaning mutated source hit %d", index)
		}
	}
	if _, _, err := cleanGeoJSONFederatedHits([]map[string]json.RawMessage{
		{"id": json.RawMessage(`"a"`)}, {"id": json.RawMessage(`"a"`)},
	}); err == nil || !strings.Contains(err.Error(), "duplicate id") {
		t.Fatalf("duplicate error = %v", err)
	}
}

func TestGeoJSONFederatedSearchRejectsHTTPAndPaginationErrors(t *testing.T) {
	for _, test := range []struct {
		name     string
		status   int
		response string
		want     string
	}{
		{name: "Meilisearch error", status: http.StatusBadRequest, response: `{"message":"incompatible ranking rules"}`, want: "HTTP 400"},
		{name: "pagination mismatch", status: http.StatusOK, response: `{"hits":[],"totalHits":0,"totalPages":0,"page":9,"hitsPerPage":10}`, want: "pagination metadata"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(test.status)
				_, _ = writer.Write([]byte(test.response))
			}))
			defer server.Close()
			client, err := newGeoJSONFederatedSearchClient(server.URL, []string{"a", "b"})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.searchProduct(
				context.Background(),
				ProductQuery{Sort: ProductSortCreatedAsc, Page: 1, PerPage: 10},
				GeoJSONQueryPlan{RadiusMode: geoJSONRadiusExact, Resolution: 100},
				false,
				false,
				0,
				geoJSONDirectProductAttributes,
				[]string{"created:asc", "id:asc"},
			)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestNewGeoJSONFederatedSearchClientValidatesIndexes(t *testing.T) {
	for _, indexes := range [][]string{{"one"}, {"a", "a"}, {"a", ""}} {
		if _, err := newGeoJSONFederatedSearchClient("http://localhost:7700", indexes); err == nil {
			t.Fatalf("indexes %v unexpectedly accepted", indexes)
		}
	}
}

func TestAuditGeoJSONCapacitySpatialCoversCompleteRadiusMatrix(t *testing.T) {
	dataset := Dataset{
		Trails: []Trail{
			{ID: "near", Parts: [][]Coordinate{{{Lat: 47.3769, Lon: 8.535}, {Lat: 47.3769, Lon: 8.545}}}},
			{ID: "far", Parts: [][]Coordinate{{{Lat: 49.0, Lon: 10.0}, {Lat: 49.1, Lon: 10.1}}}},
		},
		QueryPoints: []QueryPoint{{ID: "query", Point: Coordinate{Lat: 47.3769, Lon: 8.54}}},
	}
	radii := []float64{500, 5_000, 25_000, 100_000}
	oracle, err := buildAccuracyOracle(context.Background(), dataset, radii)
	if err != nil {
		t.Fatal(err)
	}
	requests := 0
	requestBytes := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		requestBytes += len(body)
		var decoded geoJSONFederatedSearchRequest
		if err := json.Unmarshal(body, &decoded); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		requests++
		if decoded.Federation.Page != 1 || decoded.Federation.HitsPerPage != 3 || len(decoded.Queries) != 2 {
			http.Error(writer, "unexpected exhaustive pagination", http.StatusBadRequest)
			return
		}
		for _, query := range decoded.Queries {
			if query.Query != "" || !reflect.DeepEqual(query.AttributesToRetrieve, []string{"id"}) ||
				!reflect.DeepEqual(query.Sort, []string{"id:asc"}) || !strings.HasPrefix(query.Filter, "_geoRadius(") ||
				strings.Contains(query.Filter, "public") {
				http.Error(writer, "unexpected spatial subquery", http.StatusBadRequest)
				return
			}
		}
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"hits": []map[string]any{{
				"id": "near",
				"_federation": map[string]any{
					"indexUid": "shard-0", "queriesPosition": 0, "weightedRankingScore": 1,
				},
			}},
			"processingTimeMs": 1,
			"totalHits":        1,
			"totalPages":       1,
			"page":             1,
			"hitsPerPage":      3,
		})
	}))
	defer server.Close()

	client, err := newGeoJSONFederatedSearchClient(server.URL, []string{"shard-0", "shard-1"})
	if err != nil {
		t.Fatal(err)
	}
	report, err := auditGeoJSONCapacitySpatial(
		context.Background(), client, dataset, radii, oracle,
		GeoJSONQueryPlan{ID: "r100-exact", RadiusMode: geoJSONRadiusExact, Resolution: 100},
		directAccuracyTestConfig(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "passed" || report.Cases != 4 || report.Accuracy.Requests != 4 {
		t.Fatalf("spatial audit = %+v", report)
	}
	if requests != 4 || report.HTTPRequests != 4 || report.ShardQueries != 8 || report.ShardFanout != 2 {
		t.Fatalf("spatial request accounting: server=%d report=%+v", requests, report)
	}
	if report.RequestBytes != int64(requestBytes) || report.RequestBytes == 0 || report.ResponseBytes == 0 {
		t.Fatalf("spatial byte accounting = %d/%d (captured %d)", report.RequestBytes, report.ResponseBytes, requestBytes)
	}
}
