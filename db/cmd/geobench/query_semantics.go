package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strings"
)

const (
	defaultProductQueryPage    = 1
	defaultProductQueryPerPage = 30
)

// ProductQueryDocument is the backend-independent representation used to
// validate production-like search semantics. A database or search backend can
// hydrate only its candidate IDs into these documents before calling
// RunProductQueryCandidateRefine. Trail geometry is expected to have passed the
// benchmark dataset/store validation before query timing begins.
type ProductQueryDocument struct {
	Trail    Trail
	Polyline string

	AuthorName   string
	AuthorAvatar string
	Name         string
	Description  string
	Location     string
	Tags         []string

	AuthorID           string
	Public             bool
	SharedWithActorIDs []string
	Federated          bool

	CategoryID               string
	CategoryName             string
	CategoryIcon             string
	SubcategoryID            string
	FederatedCategoryName    string
	FederatedSubcategoryName string

	DistanceMeters      float64
	DurationSeconds     float64
	ElevationGainMeters float64
	ElevationLossMeters float64
	Difficulty          float64
	Completed           bool
	DateUnix            int64
	CreatedUnix         int64
	Thumbnail           string
	Domain              string
	GPX                 string
	LikeCount           int
	IRI                 string
	BoundingBoxDiagonal float64
	Geo                 ProductGeoPoint
}

// ProductGeoPoint matches the shape of Wanderer's Meilisearch `_geo` field.
// Coordinate deliberately uses `lon`, while the product response uses `lng`,
// so keeping this small projection type avoids silently changing the API.
type ProductGeoPoint struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// ProductNumericRange is inclusive on both sides. A nil bound is open.
type ProductNumericRange struct {
	Min *float64
	Max *float64
}

type ProductQueryFilters struct {
	// Taxonomy selections form one OR group, matching the trails UI: whole
	// categories, concrete subcategories, and the explicit "no subcategory"
	// choice can be selected together. The remaining filter families are ANDed
	// with that group.
	CategoryIDs                     []string
	SubcategoryIDs                  []string
	CategoriesWithoutSubcategoryIDs []string
	// Federated is nil for both origins, true for federated-only, and false
	// for local-only results.
	Federated *bool

	DistanceMeters      ProductNumericRange
	DurationSeconds     ProductNumericRange
	ElevationGainMeters ProductNumericRange
	ElevationLossMeters ProductNumericRange
	Difficulty          ProductNumericRange
}

type ProductGeoRadius struct {
	Center       Coordinate
	RadiusMeters float64
}

// ProductQuerySort mirrors the conventional API spelling: a leading minus
// means descending order. Every ordering uses the trail ID as its final
// ascending tie-breaker, so pages cannot overlap or develop gaps because the
// candidate source returned a different order.
type ProductQuerySort string

const (
	ProductSortRelevance         ProductQuerySort = "relevance"
	ProductSortNameAsc           ProductQuerySort = "name"
	ProductSortNameDesc          ProductQuerySort = "-name"
	ProductSortCreatedAsc        ProductQuerySort = "created"
	ProductSortCreatedDesc       ProductQuerySort = "-created"
	ProductSortDistanceAsc       ProductQuerySort = "distance"
	ProductSortDistanceDesc      ProductQuerySort = "-distance"
	ProductSortDurationAsc       ProductQuerySort = "duration"
	ProductSortDurationDesc      ProductQuerySort = "-duration"
	ProductSortElevationGainAsc  ProductQuerySort = "elevation_gain"
	ProductSortElevationGainDesc ProductQuerySort = "-elevation_gain"
	ProductSortElevationLossAsc  ProductQuerySort = "elevation_loss"
	ProductSortElevationLossDesc ProductQuerySort = "-elevation_loss"
	ProductSortDifficultyAsc     ProductQuerySort = "difficulty"
	ProductSortDifficultyDesc    ProductQuerySort = "-difficulty"
	ProductSortProximityAsc      ProductQuerySort = "proximity"
	ProductSortProximityDesc     ProductQuerySort = "-proximity"
)

type ProductQuery struct {
	ActorID string
	Text    string
	Geo     *ProductGeoRadius
	Filters ProductQueryFilters
	Sort    ProductQuerySort
	Page    int
	PerPage int
}

type ProductQueryHit struct {
	ID              string
	Name            string
	CategoryID      string
	Relevance       int
	ProximityMeters *float64
}

type ProductQueryResult struct {
	Hits []ProductQueryHit

	// CandidateCount is the number of unique documents supplied by the
	// backend. ExactChecks counts point-to-polyline refinements after cheaper
	// ACL, text, category, and numeric predicates have run.
	CandidateCount int
	ExactChecks    int

	// TotalItems is the exact size of the final verified set, not an estimate
	// and not a count of the backend's coarse candidates.
	TotalItems int
	TotalPages int
	Page       int
	PerPage    int
}

// RunProductQueryExhaustive evaluates every document and is the correctness
// oracle for candidate-producing backends.
func RunProductQueryExhaustive(ctx context.Context, documents []ProductQueryDocument, query ProductQuery) (ProductQueryResult, error) {
	return runProductQuery(ctx, documents, query)
}

// RunProductQueryCandidateRefine finalizes a backend-produced candidate
// superset. For the returned total and pages to be globally exact, documents
// must contain every possible match. False positives are safe: all production
// predicates and the exact point-to-polyline radius check are reapplied here.
func RunProductQueryCandidateRefine(ctx context.Context, documents []ProductQueryDocument, query ProductQuery) (ProductQueryResult, error) {
	return runProductQuery(ctx, documents, query)
}

type productQueryMatch struct {
	document  ProductQueryDocument
	relevance int
	proximity *float64
}

func runProductQuery(ctx context.Context, documents []ProductQueryDocument, query ProductQuery) (ProductQueryResult, error) {
	normalized, terms, err := normalizeProductQuery(query)
	if err != nil {
		return ProductQueryResult{}, err
	}

	uniqueDocuments, err := uniqueProductQueryDocuments(ctx, documents)
	if err != nil {
		return ProductQueryResult{}, err
	}

	result := ProductQueryResult{
		Hits:           make([]ProductQueryHit, 0),
		CandidateCount: len(uniqueDocuments),
		Page:           normalized.Page,
		PerPage:        normalized.PerPage,
	}
	matches := make([]productQueryMatch, 0, len(uniqueDocuments))
	for index := range uniqueDocuments {
		if err := ctx.Err(); err != nil {
			return ProductQueryResult{}, err
		}

		document := uniqueDocuments[index]
		if !productDocumentVisibleTo(document, normalized.ActorID) ||
			!productDocumentMatchesFilters(document, normalized.Filters) {
			continue
		}

		relevance, matchesText := productDocumentTextScore(document, terms)
		if !matchesText {
			continue
		}

		var proximity *float64
		if normalized.Geo != nil {
			result.ExactChecks++
			distance, distanceErr := pointToTrailDistanceMetersContext(ctx, normalized.Geo.Center, document.Trail)
			if distanceErr != nil {
				return ProductQueryResult{}, distanceErr
			}
			if math.IsNaN(distance) {
				return ProductQueryResult{}, fmt.Errorf("trail %q produced a NaN point-to-polyline distance", document.Trail.ID)
			}
			if distance > normalized.Geo.RadiusMeters {
				continue
			}
			value := distance
			proximity = &value
		}

		matches = append(matches, productQueryMatch{
			document:  document,
			relevance: relevance,
			proximity: proximity,
		})
	}

	if err := ctx.Err(); err != nil {
		return ProductQueryResult{}, err
	}
	sortProductQueryMatches(matches, normalized.Sort)
	if err := ctx.Err(); err != nil {
		return ProductQueryResult{}, err
	}
	result.TotalItems = len(matches)
	if result.TotalItems > 0 {
		result.TotalPages = 1 + (result.TotalItems-1)/result.PerPage
	}

	start := productPageStart(result.Page, result.PerPage, result.TotalItems)
	if start >= result.TotalItems {
		return result, nil
	}
	end := start + result.PerPage
	if end > result.TotalItems {
		end = result.TotalItems
	}
	result.Hits = make([]ProductQueryHit, 0, end-start)
	for _, match := range matches[start:end] {
		result.Hits = append(result.Hits, ProductQueryHit{
			ID:              match.document.Trail.ID,
			Name:            match.document.Name,
			CategoryID:      match.document.CategoryID,
			Relevance:       match.relevance,
			ProximityMeters: match.proximity,
		})
	}
	return result, nil
}

func normalizeProductQuery(query ProductQuery) (ProductQuery, []string, error) {
	if query.Page == 0 {
		query.Page = defaultProductQueryPage
	}
	if query.PerPage == 0 {
		query.PerPage = defaultProductQueryPerPage
	}
	if query.Page < 1 {
		return ProductQuery{}, nil, errors.New("product query page must be at least 1")
	}
	if query.PerPage < 1 {
		return ProductQuery{}, nil, errors.New("product query per-page size must be at least 1")
	}
	if query.Sort == "" {
		query.Sort = ProductSortRelevance
	}
	if !validProductQuerySort(query.Sort) {
		return ProductQuery{}, nil, fmt.Errorf("unsupported product query sort %q", query.Sort)
	}
	if (query.Sort == ProductSortProximityAsc || query.Sort == ProductSortProximityDesc) && query.Geo == nil {
		return ProductQuery{}, nil, errors.New("proximity sort requires a geo radius")
	}
	if query.Geo != nil {
		if !validCoordinate(query.Geo.Center) {
			return ProductQuery{}, nil, errors.New("product query geo center is invalid")
		}
		if !finiteProductNumber(query.Geo.RadiusMeters) || query.Geo.RadiusMeters < 0 {
			return ProductQuery{}, nil, errors.New("product query radius must be finite and non-negative")
		}
	}

	ranges := []struct {
		name  string
		value ProductNumericRange
	}{
		{name: "distance", value: query.Filters.DistanceMeters},
		{name: "duration", value: query.Filters.DurationSeconds},
		{name: "elevation gain", value: query.Filters.ElevationGainMeters},
		{name: "elevation loss", value: query.Filters.ElevationLossMeters},
		{name: "difficulty", value: query.Filters.Difficulty},
	}
	for _, numericRange := range ranges {
		if err := validateProductNumericRange(numericRange.name, numericRange.value); err != nil {
			return ProductQuery{}, nil, err
		}
	}

	return query, normalizedProductTextTerms(query.Text), nil
}

func validProductQuerySort(value ProductQuerySort) bool {
	switch value {
	case ProductSortRelevance,
		ProductSortNameAsc, ProductSortNameDesc,
		ProductSortCreatedAsc, ProductSortCreatedDesc,
		ProductSortDistanceAsc, ProductSortDistanceDesc,
		ProductSortDurationAsc, ProductSortDurationDesc,
		ProductSortElevationGainAsc, ProductSortElevationGainDesc,
		ProductSortElevationLossAsc, ProductSortElevationLossDesc,
		ProductSortDifficultyAsc, ProductSortDifficultyDesc,
		ProductSortProximityAsc, ProductSortProximityDesc:
		return true
	default:
		return false
	}
}

func validateProductNumericRange(name string, value ProductNumericRange) error {
	if value.Min != nil && !finiteProductNumber(*value.Min) {
		return fmt.Errorf("product query %s minimum must be finite", name)
	}
	if value.Max != nil && !finiteProductNumber(*value.Max) {
		return fmt.Errorf("product query %s maximum must be finite", name)
	}
	if value.Min != nil && value.Max != nil && *value.Min > *value.Max {
		return fmt.Errorf("product query %s minimum exceeds maximum", name)
	}
	return nil
}

func uniqueProductQueryDocuments(ctx context.Context, documents []ProductQueryDocument) ([]ProductQueryDocument, error) {
	byID := make(map[string]ProductQueryDocument, len(documents))
	for index, document := range documents {
		if index%256 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		if document.Trail.ID == "" {
			return nil, errors.New("product query document has an empty trail ID")
		}
		if err := validateProductQueryDocument(document); err != nil {
			return nil, fmt.Errorf("trail %q: %w", document.Trail.ID, err)
		}
		if existing, exists := byID[document.Trail.ID]; exists {
			if !reflect.DeepEqual(existing, document) {
				return nil, fmt.Errorf("candidate source returned conflicting documents for trail %q", document.Trail.ID)
			}
			continue
		}
		byID[document.Trail.ID] = document
	}

	unique := make([]ProductQueryDocument, 0, len(byID))
	for _, document := range byID {
		unique = append(unique, document)
	}
	return unique, nil
}

func validateProductQueryDocument(document ProductQueryDocument) error {
	values := []struct {
		name  string
		value float64
	}{
		{name: "distance", value: document.DistanceMeters},
		{name: "duration", value: document.DurationSeconds},
		{name: "elevation gain", value: document.ElevationGainMeters},
		{name: "elevation loss", value: document.ElevationLossMeters},
		{name: "difficulty", value: document.Difficulty},
	}
	for _, value := range values {
		if !finiteProductNumber(value.value) {
			return fmt.Errorf("%s must be finite", value.name)
		}
	}
	return nil
}

func productDocumentVisibleTo(document ProductQueryDocument, actorID string) bool {
	if document.Public {
		return true
	}
	if actorID == "" {
		return false
	}
	if document.AuthorID == actorID {
		return true
	}
	for _, sharedActorID := range document.SharedWithActorIDs {
		if sharedActorID == actorID {
			return true
		}
	}
	return false
}

func productDocumentMatchesFilters(document ProductQueryDocument, filters ProductQueryFilters) bool {
	return productDocumentMatchesTaxonomy(document, filters) &&
		(filters.Federated == nil || document.Federated == *filters.Federated) &&
		productNumberInRange(document.DistanceMeters, filters.DistanceMeters) &&
		productNumberInRange(document.DurationSeconds, filters.DurationSeconds) &&
		productNumberInRange(document.ElevationGainMeters, filters.ElevationGainMeters) &&
		productNumberInRange(document.ElevationLossMeters, filters.ElevationLossMeters) &&
		productNumberInRange(document.Difficulty, filters.Difficulty)
}

func productDocumentMatchesTaxonomy(document ProductQueryDocument, filters ProductQueryFilters) bool {
	if len(filters.CategoryIDs) == 0 &&
		len(filters.SubcategoryIDs) == 0 &&
		len(filters.CategoriesWithoutSubcategoryIDs) == 0 {
		return true
	}
	if productStringListed(document.CategoryID, filters.CategoryIDs) ||
		productStringListed(document.SubcategoryID, filters.SubcategoryIDs) {
		return true
	}
	return document.SubcategoryID == "" &&
		productStringListed(document.CategoryID, filters.CategoriesWithoutSubcategoryIDs)
}

func productStringListed(value string, values []string) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func productNumberInRange(value float64, allowed ProductNumericRange) bool {
	if allowed.Min != nil && value < *allowed.Min {
		return false
	}
	if allowed.Max != nil && value > *allowed.Max {
		return false
	}
	return true
}

func normalizedProductTextTerms(value string) []string {
	return strings.Fields(strings.ToLower(value))
}

// productDocumentTextScore intentionally implements a small deterministic
// contract, rather than imitating backend-specific typo tolerance. Query terms
// have AND semantics across fields. Matches in author name, trail name,
// location, tags, and description receive decreasing weights for relevance
// sorting.
func productDocumentTextScore(document ProductQueryDocument, terms []string) (int, bool) {
	if len(terms) == 0 {
		return 0, true
	}

	authorName := strings.ToLower(document.AuthorName)
	name := strings.ToLower(document.Name)
	location := strings.ToLower(document.Location)
	description := strings.ToLower(document.Description)
	tags := make([]string, len(document.Tags))
	for index, tag := range document.Tags {
		tags[index] = strings.ToLower(tag)
	}

	score := 0
	for _, term := range terms {
		termScore := 0
		if strings.Contains(authorName, term) {
			termScore = 16
		}
		if strings.Contains(name, term) {
			if termScore < 8 {
				termScore = 8
			}
		}
		if strings.Contains(location, term) && termScore < 4 {
			termScore = 4
		}
		for _, tag := range tags {
			if strings.Contains(tag, term) && termScore < 3 {
				termScore = 3
				break
			}
		}
		if strings.Contains(description, term) && termScore < 1 {
			termScore = 1
		}
		if termScore == 0 {
			return 0, false
		}
		score += termScore
	}
	return score, true
}

func sortProductQueryMatches(matches []productQueryMatch, ordering ProductQuerySort) {
	sort.Slice(matches, func(leftIndex, rightIndex int) bool {
		left := matches[leftIndex]
		right := matches[rightIndex]
		comparison := compareProductQueryMatches(left, right, ordering)
		if comparison == 0 {
			return left.document.Trail.ID < right.document.Trail.ID
		}
		return comparison < 0
	})
}

func compareProductQueryMatches(left, right productQueryMatch, ordering ProductQuerySort) int {
	switch ordering {
	case ProductSortRelevance:
		return compareProductInt(right.relevance, left.relevance)
	case ProductSortNameAsc:
		return strings.Compare(strings.ToLower(left.document.Name), strings.ToLower(right.document.Name))
	case ProductSortNameDesc:
		return strings.Compare(strings.ToLower(right.document.Name), strings.ToLower(left.document.Name))
	case ProductSortCreatedAsc:
		return compareProductInt64(left.document.CreatedUnix, right.document.CreatedUnix)
	case ProductSortCreatedDesc:
		return compareProductInt64(right.document.CreatedUnix, left.document.CreatedUnix)
	case ProductSortDistanceAsc:
		return compareProductFloat(left.document.DistanceMeters, right.document.DistanceMeters)
	case ProductSortDistanceDesc:
		return compareProductFloat(right.document.DistanceMeters, left.document.DistanceMeters)
	case ProductSortDurationAsc:
		return compareProductFloat(left.document.DurationSeconds, right.document.DurationSeconds)
	case ProductSortDurationDesc:
		return compareProductFloat(right.document.DurationSeconds, left.document.DurationSeconds)
	case ProductSortElevationGainAsc:
		return compareProductFloat(left.document.ElevationGainMeters, right.document.ElevationGainMeters)
	case ProductSortElevationGainDesc:
		return compareProductFloat(right.document.ElevationGainMeters, left.document.ElevationGainMeters)
	case ProductSortElevationLossAsc:
		return compareProductFloat(left.document.ElevationLossMeters, right.document.ElevationLossMeters)
	case ProductSortElevationLossDesc:
		return compareProductFloat(right.document.ElevationLossMeters, left.document.ElevationLossMeters)
	case ProductSortDifficultyAsc:
		return compareProductFloat(left.document.Difficulty, right.document.Difficulty)
	case ProductSortDifficultyDesc:
		return compareProductFloat(right.document.Difficulty, left.document.Difficulty)
	case ProductSortProximityAsc:
		return compareProductFloat(*left.proximity, *right.proximity)
	case ProductSortProximityDesc:
		return compareProductFloat(*right.proximity, *left.proximity)
	default:
		panic("validated product query sort became invalid")
	}
}

func compareProductInt(left, right int) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

func compareProductInt64(left, right int64) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

func compareProductFloat(left, right float64) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

func productPageStart(page, perPage, total int) int {
	if total == 0 || page-1 > (total-1)/perPage {
		return total
	}
	return (page - 1) * perPage
}

func finiteProductNumber(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
