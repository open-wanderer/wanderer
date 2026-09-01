package main

import (
	"context"
	"fmt"

	"github.com/meilisearch/meilisearch-go"
)

const (
	geoJSONFederationContractIndexA = "wanderer_geo_bench_contract_federation_0"
	geoJSONFederationContractIndexB = "wanderer_geo_bench_contract_federation_1"
)

// runGeoJSONFederatedPageContract is the upgrade gate for the raw HTTP shape
// used by capacity-created shards. It intentionally covers behavior that the
// ordinary LineString contract cannot prove: exhaustive global pagination and
// counts, a deterministic cross-index sort tie, a geo filter on every subquery,
// and removal of Meilisearch's federation metadata from the product payload.
func runGeoJSONFederatedPageContract(
	ctx context.Context,
	container *meiliContainer,
	client meilisearch.ServiceManager,
) error {
	if container == nil {
		return fmt.Errorf("federated page contract requires a Meilisearch container")
	}
	if client == nil {
		return fmt.Errorf("federated page contract requires a Meilisearch client")
	}
	indexUIDs := []string{geoJSONFederationContractIndexA, geoJSONFederationContractIndexB}
	if err := createConfiguredIndexes(ctx, client, indexUIDs, []string{"_geojson"}, 10); err != nil {
		return err
	}

	nearA := geoJSONDocument("federation-a", [][][]float64{{
		{8.5350, 47.3769}, {8.5450, 47.3769},
	}})
	nearA["created"] = int64(30)
	nearD := geoJSONDocument("federation-d", [][][]float64{{
		{8.5350, 47.3780}, {8.5450, 47.3780},
	}})
	nearD["created"] = int64(10)
	nearB := geoJSONDocument("federation-b", [][][]float64{{
		{8.5350, 47.3775}, {8.5450, 47.3775},
	}})
	nearB["created"] = int64(30)
	// This document sorts ahead of every expected hit. Its absence therefore
	// proves that the geo filter was applied to both shard queries.
	farC := geoJSONDocument("federation-c", [][][]float64{{
		{9.5350, 48.3769}, {9.5450, 48.3769},
	}})
	farC["created"] = int64(40)

	for index, documents := range [][]map[string]any{{nearA, nearD}, {nearB, farC}} {
		task, err := client.Index(indexUIDs[index]).AddDocumentsWithContext(ctx, documents, nil)
		if err == nil {
			_, err = waitForSuccessfulTask(ctx, client, task)
		}
		if err != nil {
			return fmt.Errorf("index federation contract shard %d: %w", index, err)
		}
	}

	rawClient, err := newGeoJSONFederatedSearchClient(container.url, indexUIDs)
	if err != nil {
		return err
	}
	for _, pageCase := range []struct {
		page        int64
		expectedIDs []string
	}{
		{page: 1, expectedIDs: []string{"federation-a", "federation-b"}},
		{page: 2, expectedIDs: []string{"federation-d"}},
	} {
		request := geoJSONFederatedSearchRequest{
			Federation: geoJSONFederatedPagination{Page: pageCase.page, HitsPerPage: 2},
			Queries:    make([]geoJSONFederatedQuery, len(indexUIDs)),
		}
		for index, uid := range indexUIDs {
			request.Queries[index] = geoJSONFederatedQuery{
				IndexUID:             uid,
				Filter:               geoRadiusFilter(Coordinate{Lat: 47.3769, Lon: 8.5400}, 2_000, 100),
				AttributesToRetrieve: []string{"id", "created"},
				Sort:                 []string{"created:desc", "id:asc"},
				MatchingStrategy:     "all",
			}
		}
		result, err := rawClient.execute(ctx, request)
		if err != nil {
			return fmt.Errorf("search federation contract page %d: %w", pageCase.page, err)
		}
		response := result.response
		if err := validateGeoJSONFederatedPagination(response, pageCase.page, 2); err != nil {
			return fmt.Errorf("validate federation contract page %d: %w", pageCase.page, err)
		}
		if response.TotalHits != 3 || response.TotalPages != 2 {
			return fmt.Errorf(
				"federation contract page %d has totalHits/totalPages %d/%d, want 3/2",
				pageCase.page, response.TotalHits, response.TotalPages,
			)
		}
		for hitIndex, hit := range response.Hits {
			if _, ok := hit["_federation"]; !ok {
				return fmt.Errorf("federation contract page %d hit %d has no server _federation metadata", pageCase.page, hitIndex)
			}
		}
		ids, cleanHits, err := cleanGeoJSONFederatedHits(response.Hits)
		if err != nil {
			return fmt.Errorf("clean federation contract page %d: %w", pageCase.page, err)
		}
		if !equalStringSlices(ids, pageCase.expectedIDs) {
			return fmt.Errorf("federation contract page %d ids %v, want %v", pageCase.page, ids, pageCase.expectedIDs)
		}
		for hitIndex, hit := range cleanHits {
			if _, ok := hit["_federation"]; ok {
				return fmt.Errorf("clean federation contract page %d hit %d retained _federation metadata", pageCase.page, hitIndex)
			}
			if _, ok := response.Hits[hitIndex]["_federation"]; !ok {
				return fmt.Errorf("clean federation contract page %d mutated the raw server response", pageCase.page)
			}
			if _, ok := hit["created"]; !ok {
				return fmt.Errorf("clean federation contract page %d hit %d lost created", pageCase.page, hitIndex)
			}
		}
	}
	return nil
}
