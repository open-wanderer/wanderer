package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/meilisearch/meilisearch-go"
)

func TestGeoJSONFailureStatus(t *testing.T) {
	tests := []struct {
		version string
		want    string
	}{
		{version: "1.11.3", want: "unsupported"},
		{version: "v1.22.0", want: "unsupported"},
		{version: "1.22.1", want: "failed"},
		{version: "1.44.0", want: "failed"},
		{version: "unknown", want: "failed"},
	}
	for _, test := range tests {
		t.Run(test.version, func(t *testing.T) {
			if got := geoJSONFailureStatus(test.version); got != test.want {
				t.Fatalf("geoJSONFailureStatus(%q) = %q, want %q", test.version, got, test.want)
			}
		})
	}
}

func TestValidateMeiliQualificationRuntime(t *testing.T) {
	digestImage := "getmeili/meilisearch@sha256:" + strings.Repeat("b", 64)
	tests := []struct {
		name      string
		image     string
		runtime   string
		wantError bool
	}{
		{name: "minimum release", image: "getmeili/meilisearch:v1.53.1", runtime: "1.53.1"},
		{name: "newer release", image: "getmeili/meilisearch:v1.54.0", runtime: "1.54.0"},
		{name: "digest release", image: digestImage, runtime: "1.55.2"},
		{name: "older runtime", image: digestImage, runtime: "1.53.0", wantError: true},
		{name: "tag mismatch", image: "getmeili/meilisearch:v1.54.0", runtime: "1.54.1", wantError: true},
		{name: "latest alias", image: "getmeili/meilisearch:latest", runtime: "1.54.0", wantError: true},
		{name: "unofficial repository", image: "example.invalid/meilisearch:v1.54.0", runtime: "1.54.0", wantError: true},
		{name: "malformed runtime", image: digestImage, runtime: "1.54", wantError: true},
		{name: "prerelease runtime", image: digestImage, runtime: "1.54.0-rc.1", wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateMeiliQualificationRuntime(test.image, test.runtime)
			if (err != nil) != test.wantError {
				t.Fatalf("validateMeiliQualificationRuntime(%q, %q) error = %v, wantError %t", test.image, test.runtime, err, test.wantError)
			}
		})
	}
}

func TestGeoJSONFederatedPageContractRequiredForQualificationReleases(t *testing.T) {
	tests := []struct {
		name      string
		image     string
		runtime   string
		wantRun   bool
		wantError bool
	}{
		{name: "minimum release", image: "getmeili/meilisearch:v1.53.1", runtime: "1.53.1", wantRun: true},
		{name: "newer release", image: "getmeili/meilisearch:v1.54.0", runtime: "1.54.0", wantRun: true},
		{name: "older contract matrix release", image: "getmeili/meilisearch:v1.44.0", runtime: "1.44.0"},
		{name: "candidate malformed runtime", image: "getmeili/meilisearch:v1.54.0", runtime: "unknown", wantError: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			run, err := shouldRunGeoJSONFederatedPageContract(test.image, test.runtime)
			if (err != nil) != test.wantError || run != test.wantRun {
				t.Fatalf("shouldRunGeoJSONFederatedPageContract(%q, %q) = (%t, %v), want (%t, error=%t)", test.image, test.runtime, run, err, test.wantRun, test.wantError)
			}
		})
	}
}

func TestPrepareMeiliDocumentsKeepsProductionShapedCommonPolyline(t *testing.T) {
	trail := Trail{
		ID:       "route",
		Scenario: "dense",
		Parts: [][]Coordinate{{
			{Lat: 46.8, Lon: 8.2},
			{Lat: 46.81, Lon: 8.21},
		}},
	}
	for _, strategy := range []string{"point", "geojson", "h3"} {
		documents, err := prepareMeiliDocuments(strategy, []Trail{trail}, []int{7})
		if err != nil {
			t.Fatalf("prepare %s: %v", strategy, err)
		}
		if len(documents) != 1 || documents[0]["polyline"] != "_ss|G_q`q@o}@o}@" {
			t.Fatalf("%s common polyline = %#v", strategy, documents[0]["polyline"])
		}
		if _, found := documents[0]["geometry"]; found {
			t.Fatalf("%s retained benchmark-only raw geometry", strategy)
		}
	}
}

func TestGeoJSONNormalizationDoesNotChangeProductPolyline(t *testing.T) {
	original := Trail{ID: "dateline", Scenario: "long", Parts: [][]Coordinate{{
		{Lat: 70, Lon: 170},
		{Lat: 70, Lon: -170},
	}}}
	projected, report, err := projectMeiliIndexTrails(
		context.Background(),
		"geojson",
		[]Trail{original},
		GeoJSONIndexConfig{Shards: 1, MaxSegmentLengthMeters: 500_000},
	)
	if err != nil {
		t.Fatal(err)
	}
	if report.AntimeridianCuts != 1 || compactTrailPolyline(projected[0]) == compactTrailPolyline(original) {
		t.Fatalf("normalization did not change the index projection: projected %+v, report %+v", projected, report)
	}

	documents, err := prepareMeiliDocuments("geojson", projected, nil)
	if err != nil {
		t.Fatal(err)
	}
	indexedGeoJSON := documents[0]["_geojson"]
	if err := mergeMeiliProductDocuments(documents, []ProductQueryDocument{{
		Trail:    original,
		Polyline: compactTrailPolyline(original),
	}}); err != nil {
		t.Fatal(err)
	}
	if got, want := documents[0]["polyline"], compactTrailPolyline(original); got != want {
		t.Fatalf("product polyline = %#v, want original %q", got, want)
	}
	if !reflect.DeepEqual(documents[0]["_geojson"], indexedGeoJSON) {
		t.Fatal("product merge changed normalized _geojson")
	}
	if original.Parts[0][0] != (Coordinate{Lat: 70, Lon: 170}) || original.Parts[0][1] != (Coordinate{Lat: 70, Lon: -170}) {
		t.Fatalf("original/oracle geometry was mutated: %+v", original.Parts)
	}
}

func TestMergeMeiliProductDocumentsAddsSearchFilterAndSortFields(t *testing.T) {
	trail := Trail{ID: "route", Scenario: "dense", Parts: [][]Coordinate{{{Lat: 46.8, Lon: 8.2}, {Lat: 46.81, Lon: 8.21}}}}
	documents, err := prepareMeiliDocuments("geojson", []Trail{trail}, nil)
	if err != nil {
		t.Fatal(err)
	}
	product := ProductQueryDocument{
		Trail:                    trail,
		AuthorName:               "Wanderer",
		AuthorAvatar:             "wanderer.webp",
		Name:                     "Alpine route",
		Description:              "forest summit",
		Location:                 "Bernese Alps",
		Tags:                     []string{"hike"},
		AuthorID:                 "actor-1",
		Public:                   false,
		SharedWithActorIDs:       []string{"actor-2"},
		CategoryID:               "hike",
		CategoryName:             "Hiking",
		CategoryIcon:             "hiking.svg",
		SubcategoryID:            "",
		FederatedCategoryName:    "Walking",
		FederatedSubcategoryName: "Alpine",
		DistanceMeters:           1234,
		DurationSeconds:          900,
		ElevationGainMeters:      120,
		ElevationLossMeters:      80,
		Difficulty:               1,
		Completed:                true,
		DateUnix:                 100,
		CreatedUnix:              123,
		Thumbnail:                "route.webp",
		Domain:                   "remote.example",
		GPX:                      "route.gpx",
		LikeCount:                4,
		IRI:                      "https://remote.example/trails/route",
		BoundingBoxDiagonal:      1400,
		Geo:                      ProductGeoPoint{Lat: 46.8, Lng: 8.2},
	}
	if err := mergeMeiliProductDocuments(documents, []ProductQueryDocument{product}); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{
		"author_name", "author_avatar", "name", "description", "location", "tags", "author", "public", "shares",
		"category", "category_id", "category_icon", "federated_category_name", "federated_subcategory_name",
		"distance", "duration", "elevation_gain", "elevation_loss", "difficulty", "completed", "date", "created",
		"thumbnail", "domain", "gpx", "like_count", "iri", "bounding_box_diagonal", "_geo",
	} {
		if _, found := documents[0][field]; !found {
			t.Errorf("merged document has no %q field", field)
		}
	}
	if value, found := documents[0]["subcategory_id"]; !found || value != nil {
		t.Fatalf("empty subcategory = %#v (present %t), want indexed null", value, found)
	}
	if got, want := documents[0]["polyline"], compactTrailPolyline(trail); got != want {
		t.Fatalf("merged compact polyline = %#v, want %q", got, want)
	}
	if got, want := documents[0]["_geo"], (map[string]float64{"lat": 46.8, "lng": 8.2}); !reflect.DeepEqual(got, want) {
		t.Fatalf("merged _geo = %#v, want %#v", got, want)
	}
	if _, found := documents[0]["_geojson"]; !found {
		t.Fatal("merging product fields removed spatial representation")
	}
}

func TestUpdatedProductDocumentRefreshesCompactPolyline(t *testing.T) {
	original := Trail{ID: "route", Parts: [][]Coordinate{{{Lat: 46.8, Lon: 8.2}, {Lat: 46.81, Lon: 8.21}}}}
	updated := Trail{ID: "route", Parts: [][]Coordinate{{{Lat: 46.9, Lon: 8.3}, {Lat: 47.1, Lon: 8.5}}}}
	products := generateProductDocuments(Dataset{Trails: []Trail{original}})
	got := productDocumentsForUpdatedTrails(products, []Trail{updated})
	if len(got) != 1 || got[0].Polyline != compactTrailPolyline(updated) || got[0].Polyline == products[0].Polyline {
		t.Fatalf("updated product polyline = %+v", got)
	}
	if got[0].Geo != (ProductGeoPoint{Lat: 46.9, Lng: 8.3}) ||
		got[0].BoundingBoxDiagonal == products[0].BoundingBoxDiagonal {
		t.Fatalf("updated route-derived projection = geo %+v, diagonal %.1f", got[0].Geo, got[0].BoundingBoxDiagonal)
	}
}

func TestMeiliDocumentBatchesRespectCountAndBytes(t *testing.T) {
	documents := []map[string]any{
		{"id": "a", "value": "12345"},
		{"id": "b", "value": "12345"},
		{"id": "c", "value": "12345"},
	}
	var encoded [][]byte
	var counts []int
	collect := func(batch []byte, documentCount int) error {
		encoded = append(encoded, append([]byte(nil), batch...))
		counts = append(counts, documentCount)
		return nil
	}
	err := forEachMeiliDocumentBatch(context.Background(), documents, 2, 55, collect)
	if err != nil {
		t.Fatalf("batch documents: %v", err)
	}
	if len(encoded) != 2 {
		t.Fatalf("unexpected batches: %#v", encoded)
	}
	if !reflect.DeepEqual(counts, []int{2, 1}) {
		t.Fatalf("batch document counts = %v, want [2 1]", counts)
	}
	for index, wantDocuments := range []int{2, 1} {
		var decoded []map[string]any
		if err := json.Unmarshal(encoded[index], &decoded); err != nil {
			t.Fatalf("decode batch %d: %v", index, err)
		}
		if len(decoded) != wantDocuments {
			t.Fatalf("batch %d has %d documents, want %d", index, len(decoded), wantDocuments)
		}
	}
	if err := forEachMeiliDocumentBatch(context.Background(), documents[:1], 1, 5, collect); err == nil {
		t.Fatal("expected oversized document error")
	}
}

func TestMeiliPreparationHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	trail := Trail{ID: "route", Parts: [][]Coordinate{{{Lat: 0, Lon: 0}, {Lat: 0, Lon: 1}}}}
	if _, err := prepareMeiliDocumentsContext(ctx, "h3", []Trail{trail}, []int{8}); !errors.Is(err, context.Canceled) {
		t.Fatalf("prepare cancellation error = %v", err)
	}
	if err := forEachMeiliDocumentBatch(ctx, []map[string]any{{"id": "route"}}, 1, 100, func([]byte, int) error { return nil }); !errors.Is(err, context.Canceled) {
		t.Fatalf("batch cancellation error = %v", err)
	}
}

func TestEqualJSONValuesToleratesFloatRoundTripOnly(t *testing.T) {
	wanted := map[string]any{
		"id": "trail-1",
		"geometry": []any{[]any{map[string]any{
			"lat": 39.180746064256596,
			"lon": 8.25,
		}}},
	}
	got := map[string]any{
		"id": "trail-1",
		"geometry": []any{[]any{map[string]any{
			"lat": 39.18074606425659,
			"lon": 8.25,
		}}},
	}
	if !equalJSONValues(got, wanted) {
		t.Fatal("expected insignificant JSON float round-trip difference to compare equal")
	}

	got["id"] = "trail-2"
	if equalJSONValues(got, wanted) {
		t.Fatal("different string values must not compare equal")
	}
	got["id"] = "trail-1"
	gotGeometry := got["geometry"].([]any)[0].([]any)[0].(map[string]any)
	gotGeometry["lat"] = 39.18074706425659
	if equalJSONValues(got, wanted) {
		t.Fatal("materially different coordinates must not compare equal")
	}
}

func TestSuccessfulTaskError(t *testing.T) {
	if err := successfulTaskError(&meilisearch.Task{UID: 7, Status: meilisearch.TaskStatusSucceeded}); err != nil {
		t.Fatalf("succeeded task returned error: %v", err)
	}
	if err := successfulTaskError(&meilisearch.Task{UID: 8, Status: meilisearch.TaskStatusFailed}); err == nil || !strings.Contains(err.Error(), "task 8") {
		t.Fatalf("failed task error = %v", err)
	}
	if err := successfulTaskError(nil); err == nil {
		t.Fatal("nil task returned no error")
	}
}

func TestGeoJSONBoundaryRegressionFixturesRemainInsideTarget(t *testing.T) {
	documents, cases, err := geoJSONBoundaryRegressionFixtures()
	if err != nil {
		t.Fatal(err)
	}
	if len(documents) != 5 || len(cases) != 5 {
		t.Fatalf("fixtures documents/cases = %d/%d, want 5/5", len(documents), len(cases))
	}
	for index, testCase := range cases {
		trailID, _ := documents[index]["id"].(string)
		if trailID != testCase.id {
			t.Fatalf("fixture %d document id = %q, case id = %q", index, trailID, testCase.id)
		}
	}
}

func TestGeoJSONCoverageContractFixtureIsPinnedAndDatasetIndependent(t *testing.T) {
	documents := geoJSONHighLatitudeContractDocuments()
	if len(documents) != 6 {
		t.Fatalf("high-latitude documents = %d, want five GeoJSON shapes plus native control", len(documents))
	}
	wantTypes := map[string]string{
		geoJSONHighLatitudePointID:        "Point",
		geoJSONHighLatitudeNearPointID:    "Point",
		geoJSONHighLatitudeLineID:         "LineString",
		geoJSONHighLatitudeReversedLineID: "LineString",
		geoJSONHighLatitudeDenseLineID:    "LineString",
	}
	for _, document := range documents {
		id, _ := document["id"].(string)
		if id == geoJSONHighLatitudeNativePointID {
			geo, _ := document["_geo"].(map[string]any)
			if geo["lat"] != geoJSONHighLatitudeQuery.Lat || geo["lng"] != geoJSONHighLatitudeQuery.Lon {
				t.Fatalf("native _geo control = %v, want exact high-latitude query point", geo)
			}
			continue
		}
		feature, _ := document["_geojson"].(map[string]any)
		geometry, _ := feature["geometry"].(map[string]any)
		if got, _ := geometry["type"].(string); got != wantTypes[id] {
			t.Fatalf("%s geometry type = %q, want %q", id, got, wantTypes[id])
		}
		if id == geoJSONHighLatitudeDenseLineID {
			coordinates, _ := geometry["coordinates"].([][]float64)
			if len(coordinates) != 11 {
				t.Fatalf("dense line vertices = %d, want 11", len(coordinates))
			}
		}
	}

	distance := pointToSegmentDistanceMeters(
		geoJSONHighLatitudeQuery,
		geoJSONHighLatitudeNear,
		geoJSONHighLatitudeEnd,
	)
	if distance <= 40_000 || distance >= 100_000 {
		t.Fatalf("reduced line distance = %.3f m, want strictly between 40 km and 100 km", distance)
	}

	coverageDocuments := geoJSONCoverageContractDocuments()
	if len(coverageDocuments) != 10 {
		t.Fatalf("coverage documents = %d, want ten isolated controls", len(coverageDocuments))
	}
	tests, err := geoJSONCoverageContractCases()
	if err != nil {
		t.Fatal(err)
	}
	if len(tests) != 6 {
		t.Fatalf("cases = %d, want high-, ordinary- and passing-location controls", len(tests))
	}
	if got, want := tests[0].filter, "_geoRadius(79.1011091, -17.6967346, 40000, 100)"; got != want {
		t.Fatalf("40 km filter = %q, want %q", got, want)
	}
	if got, want := tests[1].filter, "_geoRadius(79.1011091, -17.6967346, 100000, 100)"; got != want {
		t.Fatalf("100 km filter = %q, want %q", got, want)
	}
	if !equalSets(tests[0].expected, idSet(geoJSONHighLatitudePointID, geoJSONHighLatitudeNativePointID)) {
		t.Fatalf("40 km expected = %v, want identical GeoJSON and native center Points", sortedIDs(tests[0].expected))
	}
	if len(tests[1].expected) != len(documents) {
		t.Fatalf("100 km expected = %v, want all %d controls", sortedIDs(tests[1].expected), len(documents))
	}
	if got, want := tests[3].filter, "_geoRadius(20.0000000, 15.0000000, 500, 100)"; got != want {
		t.Fatalf("ordinary-latitude filter = %q, want %q", got, want)
	}
	if got, want := tests[5].filter, "_geoRadius(47.3769000, 8.5417000, 500, 100)"; got != want {
		t.Fatalf("Zurich filter = %q, want %q", got, want)
	}
}

func TestGeoJSONSplitContractFixtureIsDeterministicAndDualIndexed(t *testing.T) {
	documents := geoJSONSplitContractDocuments()
	if len(documents) != geoJSONSplitContractDocumentCount {
		t.Fatalf("split documents = %d, want %d", len(documents), geoJSONSplitContractDocumentCount)
	}
	if !reflect.DeepEqual(documents, geoJSONSplitContractDocuments()) {
		t.Fatal("split fixture changed between identical generator runs")
	}
	first := documents[0]
	if first["id"] != "split-00000" {
		t.Fatalf("first split id = %v", first["id"])
	}
	last := documents[len(documents)-1]
	if last["id"] != "split-04999" {
		t.Fatalf("last split id = %v", last["id"])
	}
	for index, document := range documents {
		if document["id"] != fmt.Sprintf("split-%05d", index) {
			t.Fatalf("split document %d id = %v", index, document["id"])
		}
		geo, ok := document["_geo"].(map[string]any)
		if !ok {
			t.Fatalf("split document has no native _geo: %v", document)
		}
		shape, ok := document["_geojson"].(map[string]any)
		if !ok || shape["type"] != "Point" {
			t.Fatalf("split document has invalid _geojson: %v", document)
		}
		coordinates, ok := shape["coordinates"].([]float64)
		if !ok || len(coordinates) != 2 || coordinates[0] != geo["lng"] || coordinates[1] != geo["lat"] {
			t.Fatalf("split coordinate twins differ: _geo=%v _geojson=%v", geo, shape)
		}
		lat, latOK := geo["lat"].(float64)
		lng, lngOK := geo["lng"].(float64)
		if !latOK || !lngOK || lat < 51 || lat >= 51.6 || lng < 7 || lng >= 7.6 {
			t.Fatalf("split document %d coordinate outside fixture: %v", index, geo)
		}
	}
	firstGeo := first["_geo"].(map[string]any)
	if got, want := firstGeo["lat"], 51.0+0.6*(45454805674.0/float64(^uint64(0))); got != want {
		t.Fatalf("first deterministic latitude = %.15f, want %.15f", got, want)
	}
	if got, want := firstGeo["lng"], 7.0+0.6*(11532217803599905471.0/float64(^uint64(0))); got != want {
		t.Fatalf("first deterministic longitude = %.15f, want %.15f", got, want)
	}
	if geoJSONSplitNativeFilter != "_geoBoundingBox([51.7, 7.7], [50.9, 6.9])" ||
		geoJSONSplitShapeFilter != "_geoPolygon([50.9, 6.9], [50.9, 7.7], [51.7, 7.7], [51.7, 6.9])" {
		t.Fatalf("split filters changed: %q / %q", geoJSONSplitNativeFilter, geoJSONSplitShapeFilter)
	}
}

func TestGeoJSONContractDiagnosticCaseReportsExpectedFoundAndDifferences(t *testing.T) {
	test := geoJSONHighLatitudeContractCase{
		name:     "coverage",
		filter:   "_geoRadius(1, 2, 3, 100)",
		radius:   3,
		expected: idSet("a", "b"),
	}
	report, mismatch := geoJSONContractDiagnosticCaseReport(test, idSet("b", "c"))
	if !mismatch {
		t.Fatal("missing and unexpected IDs did not produce a mismatch")
	}
	if !reflect.DeepEqual(report.ExpectedIDs, []string{"a", "b"}) ||
		!reflect.DeepEqual(report.ActualIDs, []string{"b", "c"}) ||
		!reflect.DeepEqual(report.MissingIDs, []string{"a"}) ||
		!reflect.DeepEqual(report.UnexpectedIDs, []string{"c"}) {
		t.Fatalf("unexpected diagnostic report: %+v", report)
	}
	_, mismatch = geoJSONContractDiagnosticCaseReport(test, idSet("a", "b"))
	if mismatch {
		t.Fatal("equal expected/found sets produced a mismatch")
	}
}
