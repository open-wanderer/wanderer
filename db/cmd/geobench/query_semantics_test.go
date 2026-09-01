package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestProductQueryGoldenCandidateMatchesExhaustiveAcrossPages(t *testing.T) {
	minimumDistance := 10_000.0
	maximumDistance := 14_000.0
	minimumDuration := 3_000.0
	maximumDuration := 6_000.0
	minimumGain := 500.0
	maximumGain := 800.0
	minimumLoss := 400.0
	maximumLoss := 700.0
	minimumDifficulty := 2.0
	maximumDifficulty := 4.0
	localOnly := false

	documents := []ProductQueryDocument{
		productQueryTestDocument("route-a", 0, func(document *ProductQueryDocument) {
			document.CreatedUnix = 100
			document.DistanceMeters = minimumDistance
			document.ElevationGainMeters = minimumGain
			document.ElevationLossMeters = minimumLoss
		}),
		productQueryTestDocument("route-b", 0.005, func(document *ProductQueryDocument) {
			document.Public = false
			document.AuthorID = "actor-alice"
			document.CreatedUnix = 100
			document.DistanceMeters = maximumDistance
			document.DurationSeconds = maximumDuration
			document.ElevationGainMeters = maximumGain
			document.ElevationLossMeters = maximumLoss
			document.Difficulty = maximumDifficulty
		}),
		productQueryTestDocument("route-c", 0.01, func(document *ProductQueryDocument) {
			document.Public = false
			document.AuthorID = "actor-bob"
			document.SharedWithActorIDs = []string{"actor-alice"}
			document.CreatedUnix = 200
		}),
		productQueryTestDocument("route-private", 0, func(document *ProductQueryDocument) {
			document.Public = false
			document.AuthorID = "actor-bob"
		}),
		productQueryTestDocument("route-far", 1, nil),
		productQueryTestDocument("route-text", 0, func(document *ProductQueryDocument) {
			document.Name = "Forest Stroll"
			document.Description = "Quiet trees"
			document.Location = "Lowlands"
			document.Tags = []string{"easy"}
		}),
		productQueryTestDocument("route-category", 0, func(document *ProductQueryDocument) {
			document.CategoryID = "bike"
			document.SubcategoryID = "road"
		}),
		productQueryTestDocument("route-distance", 0, func(document *ProductQueryDocument) {
			document.DistanceMeters = minimumDistance - 1
		}),
		productQueryTestDocument("route-subcategory", 0, func(document *ProductQueryDocument) {
			document.SubcategoryID = "valley"
		}),
		productQueryTestDocument("route-difficulty", 0, func(document *ProductQueryDocument) {
			document.Difficulty = maximumDifficulty + 1
		}),
		productQueryTestDocument("route-federated", 0, func(document *ProductQueryDocument) {
			document.Federated = true
		}),
	}

	// This is a realistic coarse candidate superset: cheap metadata predicates
	// already removed some records, the far route remains a geo false positive,
	// and a segment index returned route-a twice.
	candidates := []ProductQueryDocument{
		documents[4], documents[2], documents[0], documents[1], documents[0], documents[10],
	}
	query := ProductQuery{
		ActorID: "actor-alice",
		Text:    "ALPINE loop",
		Geo: &ProductGeoRadius{
			Center:       Coordinate{Lat: 0, Lon: 0},
			RadiusMeters: 2_000,
		},
		Filters: ProductQueryFilters{
			SubcategoryIDs: []string{"mountain"},
			Federated:      &localOnly,
			DistanceMeters: ProductNumericRange{Min: &minimumDistance, Max: &maximumDistance},
			DurationSeconds: ProductNumericRange{
				Min: &minimumDuration,
				Max: &maximumDuration,
			},
			ElevationGainMeters: ProductNumericRange{Min: &minimumGain, Max: &maximumGain},
			ElevationLossMeters: ProductNumericRange{Min: &minimumLoss, Max: &maximumLoss},
			Difficulty:          ProductNumericRange{Min: &minimumDifficulty, Max: &maximumDifficulty},
		},
		Sort:    ProductSortCreatedDesc,
		PerPage: 2,
	}

	wantPages := [][]string{{"route-c", "route-a"}, {"route-b"}}
	seen := make(map[string]struct{})
	for page, wantIDs := range wantPages {
		query.Page = page + 1
		reference, err := RunProductQueryExhaustive(context.Background(), documents, query)
		if err != nil {
			t.Fatalf("reference page %d: %v", query.Page, err)
		}
		candidate, err := RunProductQueryCandidateRefine(context.Background(), candidates, query)
		if err != nil {
			t.Fatalf("candidate page %d: %v", query.Page, err)
		}

		if got := productQueryHitIDs(reference.Hits); !reflect.DeepEqual(got, wantIDs) {
			t.Fatalf("reference page %d IDs = %v, want %v", query.Page, got, wantIDs)
		}
		if got := productQueryHitIDs(candidate.Hits); !reflect.DeepEqual(got, wantIDs) {
			t.Fatalf("candidate page %d IDs = %v, want %v", query.Page, got, wantIDs)
		}
		if reference.TotalItems != 3 || candidate.TotalItems != 3 {
			t.Fatalf("page %d totals = reference %d, candidate %d; want 3", query.Page, reference.TotalItems, candidate.TotalItems)
		}
		if reference.TotalPages != 2 || candidate.TotalPages != 2 {
			t.Fatalf("page %d total pages = reference %d, candidate %d; want 2", query.Page, reference.TotalPages, candidate.TotalPages)
		}
		if reference.CandidateCount != len(documents) {
			t.Fatalf("reference candidates = %d, want %d", reference.CandidateCount, len(documents))
		}
		if candidate.CandidateCount != 5 {
			t.Fatalf("deduplicated candidate count = %d, want 5", candidate.CandidateCount)
		}
		if reference.ExactChecks != 4 || candidate.ExactChecks != 4 {
			t.Fatalf("page %d exact checks = reference %d, candidate %d; want 4", query.Page, reference.ExactChecks, candidate.ExactChecks)
		}

		for _, id := range wantIDs {
			if _, duplicate := seen[id]; duplicate {
				t.Fatalf("trail %q occurred on more than one page", id)
			}
			seen[id] = struct{}{}
		}
	}
	if got, want := len(seen), 3; got != want {
		t.Fatalf("unique paged hits = %d, want %d", got, want)
	}

	query.Page = 3
	pastEnd, err := RunProductQueryCandidateRefine(context.Background(), candidates, query)
	if err != nil {
		t.Fatalf("candidate page past end: %v", err)
	}
	if len(pastEnd.Hits) != 0 || pastEnd.TotalItems != 3 || pastEnd.TotalPages != 2 {
		t.Fatalf("page past end = %+v, want empty hits with exact totals", pastEnd)
	}
}

func TestProductQueryVisibilityACL(t *testing.T) {
	documents := []ProductQueryDocument{
		productQueryTestDocument("public", 0, nil),
		productQueryTestDocument("owned", 0, func(document *ProductQueryDocument) {
			document.Public = false
			document.AuthorID = "actor-alice"
		}),
		productQueryTestDocument("shared", 0, func(document *ProductQueryDocument) {
			document.Public = false
			document.AuthorID = "actor-bob"
			document.SharedWithActorIDs = []string{"actor-alice", "actor-carol"}
		}),
		productQueryTestDocument("private", 0, func(document *ProductQueryDocument) {
			document.Public = false
			document.AuthorID = "actor-david"
		}),
	}

	tests := []struct {
		name    string
		actorID string
		want    []string
	}{
		{name: "anonymous", want: []string{"public"}},
		{name: "owner and recipient", actorID: "actor-alice", want: []string{"owned", "public", "shared"}},
		{name: "recipient", actorID: "actor-carol", want: []string{"public", "shared"}},
		{name: "unrelated", actorID: "actor-mallory", want: []string{"public"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := RunProductQueryExhaustive(context.Background(), documents, ProductQuery{
				ActorID: test.actorID,
				PerPage: 20,
			})
			if err != nil {
				t.Fatal(err)
			}
			if got := productQueryHitIDs(result.Hits); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("visible IDs = %v, want %v", got, test.want)
			}
			if result.TotalItems != len(test.want) {
				t.Fatalf("exact total = %d, want %d", result.TotalItems, len(test.want))
			}
		})
	}
}

func TestProductQueryFederatedFilterIsTriState(t *testing.T) {
	local := productQueryTestDocument("local", 0, nil)
	federated := productQueryTestDocument("federated", 0, func(document *ProductQueryDocument) {
		document.Federated = true
	})
	documents := []ProductQueryDocument{local, federated}

	localOnly := false
	federatedOnly := true
	tests := []struct {
		name   string
		filter *bool
		want   []string
	}{
		{name: "both", want: []string{"federated", "local"}},
		{name: "local only", filter: &localOnly, want: []string{"local"}},
		{name: "federated only", filter: &federatedOnly, want: []string{"federated"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := RunProductQueryExhaustive(context.Background(), documents, ProductQuery{
				Filters: ProductQueryFilters{Federated: test.filter},
			})
			if err != nil {
				t.Fatal(err)
			}
			if got := productQueryHitIDs(result.Hits); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("IDs = %v, want %v", got, test.want)
			}
		})
	}
}

func TestProductQueryTaxonomySelectionsFormOneORGroup(t *testing.T) {
	documents := []ProductQueryDocument{
		productQueryTestDocument("whole-category", 0, func(document *ProductQueryDocument) {
			document.CategoryID = "hike"
			document.SubcategoryID = "alpine"
		}),
		productQueryTestDocument("exact-subcategory", 0, func(document *ProductQueryDocument) {
			document.CategoryID = "bike"
			document.SubcategoryID = "mtb"
		}),
		productQueryTestDocument("without-subcategory", 0, func(document *ProductQueryDocument) {
			document.CategoryID = "run"
			document.SubcategoryID = ""
		}),
		productQueryTestDocument("different-subcategory", 0, func(document *ProductQueryDocument) {
			document.CategoryID = "run"
			document.SubcategoryID = "road"
		}),
		productQueryTestDocument("unselected", 0, func(document *ProductQueryDocument) {
			document.CategoryID = "ski"
			document.SubcategoryID = "tour"
		}),
	}

	result, err := RunProductQueryExhaustive(context.Background(), documents, ProductQuery{
		Filters: ProductQueryFilters{
			CategoryIDs:                     []string{"hike"},
			SubcategoryIDs:                  []string{"mtb"},
			CategoriesWithoutSubcategoryIDs: []string{"run"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := productQueryHitIDs(result.Hits), []string{"exact-subcategory", "whole-category", "without-subcategory"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("taxonomy OR group IDs = %v, want %v", got, want)
	}
}

func TestProductQueryTextHasANDSemanticsAndDeterministicRelevance(t *testing.T) {
	documents := []ProductQueryDocument{
		productQueryTestDocument("author", 0, func(document *ProductQueryDocument) {
			document.AuthorName = "Alpine Loop"
			document.Name = "Route"
			document.Description = ""
			document.Location = ""
			document.Tags = nil
		}),
		productQueryTestDocument("name", 0, func(document *ProductQueryDocument) {
			document.Name = "Alpine Loop"
			document.Description = ""
			document.Location = ""
			document.Tags = nil
		}),
		productQueryTestDocument("tags", 0, func(document *ProductQueryDocument) {
			document.Name = "Route"
			document.Description = ""
			document.Location = ""
			document.Tags = []string{"alpine", "loop"}
		}),
		productQueryTestDocument("mixed", 0, func(document *ProductQueryDocument) {
			document.Name = "Route"
			document.Description = "A loop"
			document.Location = "Alpine village"
			document.Tags = nil
		}),
		productQueryTestDocument("one-term-only", 0, func(document *ProductQueryDocument) {
			document.Name = "Alpine traverse"
			document.Description = ""
			document.Location = ""
			document.Tags = nil
		}),
	}

	result, err := RunProductQueryExhaustive(context.Background(), documents, ProductQuery{
		Text:    "alpine LOOP",
		Sort:    ProductSortRelevance,
		PerPage: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := productQueryHitIDs(result.Hits), []string{"author", "name", "tags", "mixed"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("relevance order = %v, want %v", got, want)
	}
	if got, want := []int{result.Hits[0].Relevance, result.Hits[1].Relevance, result.Hits[2].Relevance, result.Hits[3].Relevance}, []int{32, 16, 6, 5}; !reflect.DeepEqual(got, want) {
		t.Fatalf("relevance scores = %v, want %v", got, want)
	}
}

func TestProductQueryRefinesBeforeCountingSortingAndPagination(t *testing.T) {
	documents := []ProductQueryDocument{
		productQueryTestDocument("00-far", 2, nil),
		productQueryTestDocument("01-far", -2, nil),
		productQueryTestDocument("02-near", 0, nil),
		productQueryTestDocument("03-near", 0.005, nil),
		productQueryTestDocument("04-near", 0.01, nil),
	}
	query := ProductQuery{
		Geo: &ProductGeoRadius{
			Center:       Coordinate{Lat: 0, Lon: 0},
			RadiusMeters: 2_000,
		},
		Sort:    ProductSortNameAsc,
		Page:    1,
		PerPage: 2,
	}

	first, err := RunProductQueryCandidateRefine(context.Background(), documents, query)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := productQueryHitIDs(first.Hits), []string{"02-near", "03-near"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("first verified page = %v, want %v", got, want)
	}
	if first.TotalItems != 3 || first.TotalPages != 2 || first.ExactChecks != 5 {
		t.Fatalf("first page counts = %+v, want total=3 pages=2 exact_checks=5", first)
	}

	query.Page = 2
	second, err := RunProductQueryCandidateRefine(context.Background(), documents, query)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := productQueryHitIDs(second.Hits), []string{"04-near"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("second verified page = %v, want %v", got, want)
	}
}

func TestProductQueryGeoRadiusBoundaryIsInclusive(t *testing.T) {
	document := productQueryTestDocument("crosses-center", 0, nil)
	result, err := RunProductQueryExhaustive(context.Background(), []ProductQueryDocument{document}, ProductQuery{
		Geo: &ProductGeoRadius{
			Center:       Coordinate{Lat: 0, Lon: 0},
			RadiusMeters: 0,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.TotalItems != 1 || len(result.Hits) != 1 {
		t.Fatalf("zero-radius boundary result = %+v, want one exact hit", result)
	}
	if result.Hits[0].ProximityMeters == nil || *result.Hits[0].ProximityMeters != 0 {
		t.Fatalf("boundary proximity = %v, want exactly 0", result.Hits[0].ProximityMeters)
	}
}

func TestProductQueryStableTieBreakerPreventsPageGapsAndDuplicates(t *testing.T) {
	documents := make([]ProductQueryDocument, 0, 11)
	for index := 9; index >= 0; index-- {
		id := "route-" + string(rune('a'+index))
		documents = append(documents, productQueryTestDocument(id, 0, func(document *ProductQueryDocument) {
			document.Name = "Same name"
			document.CreatedUnix = 100
		}))
	}
	documents = append(documents, documents[0])

	query := ProductQuery{Sort: ProductSortCreatedDesc, PerPage: 3}
	want := []string{"route-a", "route-b", "route-c", "route-d", "route-e", "route-f", "route-g", "route-h", "route-i", "route-j"}
	got := make([]string, 0, len(want))
	seen := make(map[string]struct{}, len(want))
	for page := 1; page <= 4; page++ {
		query.Page = page
		result, err := RunProductQueryCandidateRefine(context.Background(), documents, query)
		if err != nil {
			t.Fatalf("page %d: %v", page, err)
		}
		if result.TotalItems != 10 || result.TotalPages != 4 || result.CandidateCount != 10 {
			t.Fatalf("page %d counts = %+v, want total=10 pages=4 candidates=10", page, result)
		}
		for _, id := range productQueryHitIDs(result.Hits) {
			if _, duplicate := seen[id]; duplicate {
				t.Fatalf("trail %q appears on multiple pages", id)
			}
			seen[id] = struct{}{}
			got = append(got, id)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("paged IDs = %v, want %v", got, want)
	}
}

func TestProductQuerySupportedSortOrders(t *testing.T) {
	documents := []ProductQueryDocument{
		productQueryTestDocument("a", 0.01, func(document *ProductQueryDocument) {
			document.Name = "Zulu"
			document.CreatedUnix = 100
			document.DistanceMeters = 300
			document.DurationSeconds = 100
			document.ElevationGainMeters = 200
			document.ElevationLossMeters = 300
			document.Difficulty = 2
		}),
		productQueryTestDocument("b", 0, func(document *ProductQueryDocument) {
			document.Name = "Alpha"
			document.CreatedUnix = 300
			document.DistanceMeters = 100
			document.DurationSeconds = 300
			document.ElevationGainMeters = 100
			document.ElevationLossMeters = 200
			document.Difficulty = 3
		}),
		productQueryTestDocument("c", 0.005, func(document *ProductQueryDocument) {
			document.Name = "Mike"
			document.CreatedUnix = 200
			document.DistanceMeters = 200
			document.DurationSeconds = 200
			document.ElevationGainMeters = 300
			document.ElevationLossMeters = 100
			document.Difficulty = 1
		}),
	}
	tests := []struct {
		sort ProductQuerySort
		want []string
	}{
		{sort: ProductSortRelevance, want: []string{"a", "b", "c"}},
		{sort: ProductSortNameAsc, want: []string{"b", "c", "a"}},
		{sort: ProductSortNameDesc, want: []string{"a", "c", "b"}},
		{sort: ProductSortCreatedAsc, want: []string{"a", "c", "b"}},
		{sort: ProductSortCreatedDesc, want: []string{"b", "c", "a"}},
		{sort: ProductSortDistanceAsc, want: []string{"b", "c", "a"}},
		{sort: ProductSortDistanceDesc, want: []string{"a", "c", "b"}},
		{sort: ProductSortDurationAsc, want: []string{"a", "c", "b"}},
		{sort: ProductSortDurationDesc, want: []string{"b", "c", "a"}},
		{sort: ProductSortElevationGainAsc, want: []string{"b", "a", "c"}},
		{sort: ProductSortElevationGainDesc, want: []string{"c", "a", "b"}},
		{sort: ProductSortElevationLossAsc, want: []string{"c", "b", "a"}},
		{sort: ProductSortElevationLossDesc, want: []string{"a", "b", "c"}},
		{sort: ProductSortDifficultyAsc, want: []string{"c", "a", "b"}},
		{sort: ProductSortDifficultyDesc, want: []string{"b", "a", "c"}},
		{sort: ProductSortProximityAsc, want: []string{"b", "c", "a"}},
		{sort: ProductSortProximityDesc, want: []string{"a", "c", "b"}},
	}
	for _, test := range tests {
		t.Run(string(test.sort), func(t *testing.T) {
			result, err := RunProductQueryExhaustive(context.Background(), documents, ProductQuery{
				Geo: &ProductGeoRadius{
					Center:       Coordinate{Lat: 0, Lon: 0},
					RadiusMeters: 2_000,
				},
				Sort:    test.sort,
				PerPage: 20,
			})
			if err != nil {
				t.Fatal(err)
			}
			if got := productQueryHitIDs(result.Hits); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("IDs = %v, want %v", got, test.want)
			}
		})
	}
}

func TestProductQueryValidationAndCancellation(t *testing.T) {
	document := productQueryTestDocument("route", 0, nil)

	maximum := 1.0
	minimum := 2.0
	_, err := RunProductQueryExhaustive(context.Background(), []ProductQueryDocument{document}, ProductQuery{
		Filters: ProductQueryFilters{DistanceMeters: ProductNumericRange{Min: &minimum, Max: &maximum}},
	})
	if err == nil {
		t.Fatal("inverted numeric range was accepted")
	}

	_, err = RunProductQueryExhaustive(context.Background(), []ProductQueryDocument{document}, ProductQuery{
		Sort: ProductSortProximityAsc,
	})
	if err == nil {
		t.Fatal("proximity sort without geo radius was accepted")
	}

	conflict := document
	conflict.Name = "Different"
	_, err = RunProductQueryCandidateRefine(context.Background(), []ProductQueryDocument{document, conflict}, ProductQuery{})
	if err == nil {
		t.Fatal("conflicting duplicate candidate was accepted")
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = RunProductQueryExhaustive(cancelled, []ProductQueryDocument{document}, ProductQuery{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled query error = %v, want context.Canceled", err)
	}
}

func productQueryTestDocument(id string, latitude float64, mutate func(*ProductQueryDocument)) ProductQueryDocument {
	document := ProductQueryDocument{
		Trail: Trail{
			ID: id,
			Parts: [][]Coordinate{{
				{Lat: latitude, Lon: -0.001},
				{Lat: latitude, Lon: 0.001},
			}},
		},
		AuthorName:          "Guide",
		Name:                id + " Alpine Loop",
		Description:         "A mountain route",
		Location:            "Alps",
		Tags:                []string{"hiking"},
		AuthorID:            "actor-owner",
		Public:              true,
		CategoryID:          "hike",
		SubcategoryID:       "mountain",
		DistanceMeters:      12_000,
		DurationSeconds:     4_000,
		ElevationGainMeters: 600,
		ElevationLossMeters: 500,
		Difficulty:          3,
		CreatedUnix:         50,
	}
	if mutate != nil {
		mutate(&document)
	}
	return document
}

func productQueryHitIDs(hits []ProductQueryHit) []string {
	ids := make([]string, len(hits))
	for index, hit := range hits {
		ids[index] = hit.ID
	}
	return ids
}
