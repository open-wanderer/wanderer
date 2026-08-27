package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type assetCandidateSearchLimits struct {
	MaxItems            int
	MaxScannedItems     int
	MaxProviderRequests int
}

type assetCandidateProviderPage struct {
	Items    []immichAsset
	NextPage *string
}

type assetCandidateSearchResult struct {
	Candidates       []assetCandidate
	State            map[string]any
	HasMore          bool
	ScannedItems     int
	ProviderRequests int
}

type assetCandidatePageFetcher func(page int) (assetCandidateProviderPage, error)

func searchAssetCandidatePages(req assetLibraryRequest, cfg immichConfig, state map[string]any, limits assetCandidateSearchLimits, fetchPage assetCandidatePageFetcher) (assetCandidateSearchResult, error) {
	if limits.MaxItems <= 0 {
		limits.MaxItems = 100
	}
	if limits.MaxScannedItems <= 0 {
		limits.MaxScannedItems = 2500
	}
	if limits.MaxProviderRequests <= 0 {
		limits.MaxProviderRequests = 10
	}

	page := assetCandidateStateInt(state, "page", 1)
	offset := assetCandidateStateInt(state, "offset", 0)
	if page < 1 || offset < 0 {
		return assetCandidateSearchResult{}, fmt.Errorf("invalid asset candidate state")
	}

	result := assetCandidateSearchResult{Candidates: make([]assetCandidate, 0, limits.MaxItems)}
	for {
		if result.ProviderRequests >= limits.MaxProviderRequests {
			result.State = candidateSearchState(page, offset)
			result.HasMore = true
			result.Candidates = sortMatches(result.Candidates)
			return result, nil
		}

		providerPage, err := fetchPage(page)
		if err != nil {
			return assetCandidateSearchResult{}, err
		}
		result.ProviderRequests++
		items := providerPage.Items
		if offset > len(items) {
			offset = len(items)
		}
		for index := offset; index < len(items); index++ {
			if result.ScannedItems >= limits.MaxScannedItems {
				result.State = candidateSearchState(page, index)
				result.HasMore = true
				result.Candidates = sortMatches(result.Candidates)
				return result, nil
			}
			result.ScannedItems++
			result.Candidates = append(result.Candidates, matchAssetCandidates(items[index:index+1], req, cfg)...)
			if len(result.Candidates) >= limits.MaxItems {
				nextOffset := index + 1
				if nextOffset < len(items) {
					result.State = candidateSearchState(page, nextOffset)
					result.HasMore = true
					result.Candidates = sortMatches(result.Candidates)
					return result, nil
				}
				nextPage, hasMore, err := nextCandidatePage(page, providerPage.NextPage)
				if err != nil {
					return assetCandidateSearchResult{}, err
				}
				if hasMore {
					result.State = candidateSearchState(nextPage, 0)
					result.HasMore = true
				}
				result.Candidates = sortMatches(result.Candidates)
				return result, nil
			}
		}

		nextPage, hasMore, err := nextCandidatePage(page, providerPage.NextPage)
		if err != nil {
			return assetCandidateSearchResult{}, err
		}
		if !hasMore {
			result.Candidates = sortMatches(result.Candidates)
			return result, nil
		}
		page = nextPage
		offset = 0
		if result.ScannedItems >= limits.MaxScannedItems {
			result.State = candidateSearchState(page, offset)
			result.HasMore = true
			result.Candidates = sortMatches(result.Candidates)
			return result, nil
		}
	}
}

func assetCandidateStateInt(state map[string]any, key string, fallback int) int {
	switch value := state[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	case json.Number:
		parsed, err := value.Int64()
		if err == nil {
			return int(parsed)
		}
	case string:
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func nextCandidatePage(page int, value *string) (int, bool, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return 0, false, nil
	}
	next, err := strconv.Atoi(*value)
	if err != nil || next <= page {
		return 0, false, fmt.Errorf("invalid non-progressing Immich nextPage %q after page %d", *value, page)
	}
	return next, true, nil
}

func candidateSearchState(page, offset int) map[string]any {
	return map[string]any{"page": page, "offset": offset}
}
