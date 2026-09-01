package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/meilisearch/meilisearch-go"
)

func TestSearchMeiliProductCandidatesPushesPredicatesWithoutPagination(t *testing.T) {
	type requestBody struct {
		Query            string   `json:"q"`
		Filter           string   `json:"filter"`
		Limit            int64    `json:"limit"`
		Page             int64    `json:"page"`
		HitsPerPage      int64    `json:"hitsPerPage"`
		Sort             []string `json:"sort"`
		MatchingStrategy string   `json:"matchingStrategy"`
	}
	var captured requestBody
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if err := json.NewDecoder(request.Body).Decode(&captured); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"hits":[{"id":"trail-1"}],"estimatedTotalHits":1,"processingTimeMs":2,"query":"forest summit"}`))
	}))
	defer server.Close()

	query := ProductQuery{
		ActorID: "actor-1",
		Text:    "forest summit",
		Geo:     &ProductGeoRadius{Center: Coordinate{Lat: 46.8, Lon: 8.2}, RadiusMeters: 5000},
		Filters: ProductQueryFilters{CategoryIDs: []string{"hike"}},
		Sort:    ProductSortCreatedDesc,
		Page:    3,
		PerPage: 5,
	}
	result, err := searchMeiliProductCandidates(
		context.Background(), meilisearch.New(server.URL), query, `_geoRadius(46.8, 8.2, 5000, 125)`, 100,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.IDs) != 1 || result.Truncated {
		t.Fatalf("candidate result = %+v", result)
	}
	if captured.Query != query.Text || captured.Limit != 100 || captured.MatchingStrategy != "all" {
		t.Fatalf("candidate request = %+v", captured)
	}
	if captured.Page != 0 || captured.HitsPerPage != 0 || len(captured.Sort) != 0 {
		t.Fatalf("hybrid candidate request paginated or sorted before exact refinement: %+v", captured)
	}
	for _, wanted := range []string{"public = true", `category_id IN ["hike"]`, "_geoRadius"} {
		if !strings.Contains(captured.Filter, wanted) {
			t.Errorf("filter %q does not contain %q", captured.Filter, wanted)
		}
	}
}
