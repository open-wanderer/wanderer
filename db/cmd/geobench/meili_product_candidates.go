package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/meilisearch/meilisearch-go"
)

// searchMeiliProductCandidates pushes every non-geometric product predicate
// into Meilisearch, but deliberately requests the complete unpaginated ID set.
// SQLite plus the exact point-to-polyline finalizer remain authoritative for
// counts, sorting and pages in the hybrid strategies.
func searchMeiliProductCandidates(
	ctx context.Context,
	client meilisearch.ServiceManager,
	query ProductQuery,
	spatialFilter string,
	limit int64,
) (CandidateResult, error) {
	started := time.Now()
	filter, err := productMeiliFilter(query, spatialFilter)
	if err != nil {
		return CandidateResult{}, err
	}
	response, err := client.Index(benchmarkIndexUID).SearchWithContext(ctx, query.Text, &meilisearch.SearchRequest{
		Filter:               filter,
		Limit:                limit,
		AttributesToRetrieve: []string{"id"},
		MatchingStrategy:     meilisearch.All,
	})
	if err != nil {
		return CandidateResult{}, err
	}
	ids := make(map[string]struct{}, len(response.Hits))
	for _, hit := range response.Hits {
		rawID, ok := hit["id"]
		if !ok {
			return CandidateResult{}, fmt.Errorf("product candidate hit has no id")
		}
		var id string
		if err := json.Unmarshal(rawID, &id); err != nil {
			return CandidateResult{}, fmt.Errorf("decode product candidate id: %w", err)
		}
		if _, duplicate := ids[id]; duplicate {
			return CandidateResult{}, fmt.Errorf("product candidate search returned duplicate id %q", id)
		}
		ids[id] = struct{}{}
	}
	return CandidateResult{
		IDs:                ids,
		WallDuration:       time.Since(started),
		EngineProcessingMS: float64(response.ProcessingTimeMs),
		EstimatedTotalHits: response.EstimatedTotalHits,
		Truncated:          response.EstimatedTotalHits > int64(len(ids)),
	}, nil
}

func newMeiliProductCandidateQuery(
	client meilisearch.ServiceManager,
	strategy string,
	h3Resolutions []int,
	geoJSONPlan *GeoJSONQueryPlan,
	limit int64,
) productCandidateQuery {
	return func(ctx context.Context, point QueryPoint, radius float64, query ProductQuery) (CandidateResult, error) {
		preparedAt := time.Now()
		var spatialFilter string
		switch strategy {
		case "point":
			spatialFilter = geoRadiusFilter(point.Point, radius, 0)
		case "geojson":
			plan := GeoJSONQueryPlan{RadiusMode: geoJSONRadiusDefault}
			if geoJSONPlan != nil {
				plan = *geoJSONPlan
			}
			candidateRadius, err := geoJSONCandidateRadius(radius, plan)
			if err != nil {
				return CandidateResult{}, err
			}
			spatialFilter = geoRadiusFilter(point.Point, candidateRadius, plan.Resolution)
		case "h3":
			_, filter, _, err := h3QueryFilter(point.Point, radius, h3Resolutions)
			if err != nil {
				return CandidateResult{}, err
			}
			spatialFilter = filter
		default:
			return CandidateResult{}, fmt.Errorf("unknown product candidate strategy %q", strategy)
		}
		prepareDuration := time.Since(preparedAt)
		result, err := searchMeiliProductCandidates(ctx, client, query, spatialFilter, limit)
		result.WallDuration += prepareDuration
		return result, err
	}
}
