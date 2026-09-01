package main

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
)

func TestProductStoreLoadsAndDecodesOnlyCandidates(t *testing.T) {
	dataset := Dataset{Trails: []Trail{
		{ID: "a", Scenario: scenarioAlpineDense, Parts: [][]Coordinate{{{Lat: 47, Lon: 8}, {Lat: 47.01, Lon: 8.01}}}},
		{ID: "b", Scenario: scenarioBackground, Parts: [][]Coordinate{{{Lat: 40, Lon: 3}, {Lat: 40.02, Lon: 3.01}}}},
		{ID: "c", Scenario: scenarioLongOutlier, Parts: [][]Coordinate{{{Lat: 0, Lon: -10}, {Lat: 0, Lon: 10}}}},
	}}
	documents := generateProductDocuments(dataset)
	store, err := newProductStore(filepath.Join(t.TempDir(), "product.sqlite"), 2)
	if err != nil {
		t.Fatalf("new product store: %v", err)
	}
	defer store.close()
	if err := store.replaceAll(context.Background(), documents); err != nil {
		t.Fatalf("replace product store: %v", err)
	}

	loaded, err := store.load(context.Background(), idSet("c", "a", "missing"))
	if err != nil {
		t.Fatalf("load product store: %v", err)
	}
	if loaded.Rows != 2 || loaded.EncodedBytes == 0 {
		t.Fatalf("load metrics = rows %d bytes %d", loaded.Rows, loaded.EncodedBytes)
	}
	if !reflect.DeepEqual(loaded.Documents, []ProductQueryDocument{documents[0], documents[2]}) {
		t.Fatalf("loaded documents differ: %#v", loaded.Documents)
	}
	if size, err := store.diskBytes(); err != nil || size == 0 {
		t.Fatalf("product store disk size = %d, %v", size, err)
	}
}

func TestGenerateProductDocumentsIsDeterministicAndDoesNotBridgeParts(t *testing.T) {
	dataset := Dataset{Trails: []Trail{{
		ID:       "split",
		Scenario: scenarioBackground,
		Parts: [][]Coordinate{
			{{Lat: 0, Lon: 0}, {Lat: 0, Lon: 0.01}},
			{{Lat: 0, Lon: 20}, {Lat: 0, Lon: 20.01}},
		},
	}}}
	first := generateProductDocuments(dataset)
	second := generateProductDocuments(dataset)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("generated product documents are not deterministic")
	}
	if got, want := first[0].Polyline, compactTrailPolyline(dataset.Trails[0]); got == "" || got != want {
		t.Fatalf("generated compact polyline = %q, want %q", got, want)
	}
	if first[0].AuthorAvatar == "" || first[0].CategoryName == "" || first[0].CategoryIcon == "" ||
		first[0].Thumbnail == "" || first[0].GPX == "" || first[0].DateUnix == 0 ||
		first[0].BoundingBoxDiagonal == 0 {
		t.Fatalf("generated product response fields are incomplete: %+v", first[0])
	}
	if got, want := first[0].Geo, (ProductGeoPoint{Lat: 0, Lng: 0}); got != want {
		t.Fatalf("generated _geo = %+v, want start point %+v", got, want)
	}
	if !first[0].Federated || first[0].Domain == "" || first[0].IRI == "" ||
		first[0].FederatedCategoryName == "" || first[0].FederatedSubcategoryName == "" {
		t.Fatalf("generated federated projection is incomplete: %+v", first[0])
	}
	wantApprox := 2 * 0.01 * (3.141592653589793 / 180) * earthRadiusMeters
	if got := first[0].DistanceMeters; got < wantApprox*0.98 || got > wantApprox*1.02 {
		t.Fatalf("distance = %.1f, want about %.1f without the gap", got, wantApprox)
	}
}
