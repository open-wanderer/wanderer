package main

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sort"
	"testing"
)

func TestProductPagePayloadMatchesDirectProjection(t *testing.T) {
	trailA := Trail{ID: "a", Parts: [][]Coordinate{{{Lat: 46.8, Lon: 8.2}, {Lat: 46.81, Lon: 8.21}}}}
	trailB := Trail{ID: "b", Parts: [][]Coordinate{{{Lat: 47.0, Lon: 8.0}, {Lat: 47.01, Lon: 8.01}}}}
	documents := []ProductQueryDocument{
		{
			Trail:                    trailA,
			Polyline:                 compactTrailPolyline(trailA),
			AuthorID:                 "actor-a",
			AuthorName:               "Author A",
			AuthorAvatar:             "author-a.webp",
			Name:                     "Trail A",
			Description:              "Description A",
			Location:                 "Location A",
			DistanceMeters:           1200,
			ElevationGainMeters:      300,
			ElevationLossMeters:      200,
			DurationSeconds:          3600,
			Difficulty:               2,
			CategoryID:               "hike",
			CategoryName:             "Hiking",
			CategoryIcon:             "hiking.svg",
			Federated:                true,
			FederatedCategoryName:    "Walking",
			FederatedSubcategoryName: "Alpine",
			Completed:                true,
			DateUnix:                 100,
			CreatedUnix:              123,
			Public:                   true,
			Thumbnail:                "trail-a.webp",
			Domain:                   "remote.example",
			GPX:                      "trail-a.gpx",
			Tags:                     []string{"summit"},
			LikeCount:                17,
			SharedWithActorIDs:       []string{"actor-c"},
			IRI:                      "https://remote.example/trails/a",
			BoundingBoxDiagonal:      1200,
			Geo:                      ProductGeoPoint{Lat: 46.8, Lng: 8.2},
		},
		{
			Trail:               trailB,
			Polyline:            compactTrailPolyline(trailB),
			AuthorID:            "actor-b",
			AuthorName:          "Author B",
			AuthorAvatar:        "author-b.webp",
			Name:                "Trail B",
			Description:         "Description B",
			Location:            "Location B",
			DistanceMeters:      2400,
			ElevationGainMeters: 600,
			ElevationLossMeters: 400,
			DurationSeconds:     7200,
			Difficulty:          1,
			CategoryID:          "bike",
			CategoryName:        "Cycling",
			CategoryIcon:        "cycling.svg",
			SubcategoryID:       "gravel",
			DateUnix:            400,
			CreatedUnix:         456,
			Thumbnail:           "trail-b.webp",
			GPX:                 "trail-b.gpx",
			Tags:                []string{"forest"},
			LikeCount:           3,
			SharedWithActorIDs:  []string{},
			BoundingBoxDiagonal: 2400,
			Geo:                 ProductGeoPoint{Lat: 47, Lng: 8},
		},
	}
	result := ProductQueryResult{
		Hits:       []ProductQueryHit{{ID: "b"}, {ID: "a"}},
		TotalItems: 7,
		TotalPages: 4,
		Page:       2,
		PerPage:    2,
	}

	encoded, err := marshalProductPagePayload(result, documents)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Hits        []map[string]json.RawMessage `json:"hits"`
		TotalHits   int                          `json:"totalHits"`
		TotalPages  int                          `json:"totalPages"`
		Page        int                          `json:"page"`
		HitsPerPage int                          `json:"hitsPerPage"`
	}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.TotalHits != 7 || decoded.TotalPages != 4 || decoded.Page != 2 || decoded.HitsPerPage != 2 {
		t.Fatalf("page metadata = %+v", decoded)
	}
	if !reflect.DeepEqual(productPageAttributes(), geoJSONDirectProductAttributes) {
		t.Fatalf("hybrid fields %v differ from direct fields %v", productPageAttributes(), geoJSONDirectProductAttributes)
	}
	wantProjection := []string{
		"id", "author", "author_name", "author_avatar", "name", "description", "location",
		"distance", "elevation_gain", "elevation_loss", "duration", "difficulty", "category",
		"category_id", "category_icon", "subcategory_id", "is_federated", "federated_category_name",
		"federated_subcategory_name", "completed", "date", "created", "public", "thumbnail", "domain",
		"gpx", "tags", "like_count", "shares", "iri", "bounding_box_diagonal", "_geo",
	}
	if !reflect.DeepEqual(productPageAttributes(), wantProjection) {
		t.Fatalf("product projection = %v, want current defaultTrailSearchAttributes %v", productPageAttributes(), wantProjection)
	}
	wantFields := productPageAttributes()
	sort.Strings(wantFields)
	for index, hit := range decoded.Hits {
		gotFields := make([]string, 0, len(hit))
		for field := range hit {
			gotFields = append(gotFields, field)
		}
		sort.Strings(gotFields)
		if !reflect.DeepEqual(gotFields, wantFields) {
			t.Fatalf("hit %d fields = %v, want %v", index, gotFields, wantFields)
		}
	}
	var gotID, gotGPX string
	if err := json.Unmarshal(decoded.Hits[0]["id"], &gotID); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(decoded.Hits[0]["gpx"], &gotGPX); err != nil {
		t.Fatal(err)
	}
	if gotID != "b" || gotGPX != "trail-b.gpx" {
		t.Fatalf("first hit id/gpx = %q/%q", gotID, gotGPX)
	}
	if _, found := decoded.Hits[0]["polyline"]; found {
		t.Fatal("list response unexpectedly contains indexed polyline")
	}
	var gotGeo ProductGeoPoint
	if err := json.Unmarshal(decoded.Hits[0]["_geo"], &gotGeo); err != nil {
		t.Fatal(err)
	}
	if gotGeo != documents[1].Geo {
		t.Fatalf("first hit _geo = %+v, want %+v", gotGeo, documents[1].Geo)
	}
	if string(decoded.Hits[1]["subcategory_id"]) != "null" {
		t.Fatalf("empty subcategory JSON = %s, want null", decoded.Hits[1]["subcategory_id"])
	}

	meiliDocuments, err := prepareMeiliDocuments("geojson", []Trail{trailA, trailB}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := mergeMeiliProductDocuments(meiliDocuments, documents); err != nil {
		t.Fatal(err)
	}
	rawHits := []map[string]json.RawMessage{
		rawProductPageHit(t, meiliDocuments[1]),
		rawProductPageHit(t, meiliDocuments[0]),
	}
	meiliEncoded, err := marshalMeiliProductPagePayload(rawHits, 7, 4, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(meiliEncoded, encoded) {
		t.Fatalf("Meilisearch and document payloads differ:\nmeili:   %s\ndocument: %s", meiliEncoded, encoded)
	}

	missingField := rawProductPageHit(t, meiliDocuments[0])
	delete(missingField, "_geo")
	if _, err := marshalMeiliProductPagePayload([]map[string]json.RawMessage{missingField}, 1, 1, 1, 2); err == nil {
		t.Fatal("Meilisearch hit with a missing projection field was accepted")
	}
	extraField := rawProductPageHit(t, meiliDocuments[0])
	extraField["unexpected"] = json.RawMessage("true")
	if _, err := marshalMeiliProductPagePayload([]map[string]json.RawMessage{extraField}, 1, 1, 1, 2); err == nil {
		t.Fatal("Meilisearch hit with an extra projection field was accepted")
	}
}

func TestMaterializeProductPageRejectsMissingResultDocument(t *testing.T) {
	_, err := materializeProductPagePayload(
		ProductQueryResult{Hits: []ProductQueryHit{{ID: "missing"}}, Page: 1, PerPage: 30},
		[]ProductQueryDocument{{Trail: Trail{ID: "present"}}},
	)
	if err == nil {
		t.Fatal("missing page document was accepted")
	}
}

func rawProductPageHit(t *testing.T, document map[string]any) map[string]json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	var hit map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &hit); err != nil {
		t.Fatal(err)
	}
	projected := make(map[string]json.RawMessage, len(productPageAttributeNames))
	for _, attribute := range productPageAttributeNames {
		if value, found := hit[attribute]; found {
			projected[attribute] = value
		}
	}
	return projected
}
