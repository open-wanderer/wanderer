package main

import "fmt"

type geoJSONRadiusContractCase struct {
	name     string
	point    Coordinate
	radius   float64
	expected map[string]struct{}
}

type geoJSONBoundingBoxContractCase struct {
	name     string
	viewport GeoViewport
	expected map[string]struct{}
}

type geoJSONBoundaryRegressionCase struct {
	name   string
	point  Coordinate
	radius float64
	id     string
}

type geoJSONHighLatitudeContractCase struct {
	name     string
	filter   string
	radius   float64
	expected map[string]struct{}
}

var (
	geoJSONHighLatitudeQuery = Coordinate{Lat: 79.1011091432348, Lon: -17.696734637372288}
	geoJSONHighLatitudeNear  = Coordinate{Lat: 79.42264939251596, Lon: -18.786811473073783}
	geoJSONHighLatitudeEnd   = Coordinate{Lat: 79.58103445894763, Lon: -17.50266287139425}
	geoJSONMidLatitudeQuery  = Coordinate{Lat: 20, Lon: 15}
	geoJSONZurichQuery       = Coordinate{Lat: 47.3769, Lon: 8.5417}
)

const (
	geoJSONHighLatitudePointID        = "high-latitude-query-point"
	geoJSONHighLatitudeNativePointID  = "high-latitude-query-native-geo"
	geoJSONHighLatitudeNearPointID    = "high-latitude-near-point"
	geoJSONHighLatitudeLineID         = "high-latitude-line"
	geoJSONHighLatitudeReversedLineID = "high-latitude-line-reversed"
	geoJSONHighLatitudeDenseLineID    = "high-latitude-line-dense"
	geoJSONMidLatitudePointID         = "mid-latitude-query-point"
	geoJSONMidLatitudeNativePointID   = "mid-latitude-query-native-geo"
	geoJSONZurichPointID              = "zurich-query-point"
	geoJSONZurichNativePointID        = "zurich-query-native-geo"
	geoJSONSplitContractDocumentCount = 5_000
	geoJSONSplitNativeFilter          = "_geoBoundingBox([51.7, 7.7], [50.9, 6.9])"
	geoJSONSplitShapeFilter           = "_geoPolygon([50.9, 6.9], [50.9, 7.7], [51.7, 7.7], [51.7, 6.9])"
)

// geoJSONSplitContractDocuments is a compact, deterministic reduction of the
// dense Cellulite split failure. Five thousand points are enough to cross the
// 200-document threshold through several H3 resolutions. Every coordinate is
// stored in both fields so the native bounding-box count is an ingestion
// control for the GeoJSON polygon count.
func geoJSONSplitContractDocuments() []map[string]any {
	documents := make([]map[string]any, 0, geoJSONSplitContractDocumentCount)
	state := uint64(42)
	nextUnit := func() float64 {
		state ^= state << 13
		state ^= state >> 7
		state ^= state << 17
		return float64(state) / float64(^uint64(0))
	}
	for index := range geoJSONSplitContractDocumentCount {
		point := Coordinate{
			Lat: 51 + 0.6*nextUnit(),
			Lon: 7 + 0.6*nextUnit(),
		}
		documents = append(documents, map[string]any{
			"id": fmt.Sprintf("split-%05d", index),
			"_geo": map[string]any{
				"lat": point.Lat,
				"lng": point.Lon,
			},
			"_geojson": map[string]any{
				"type":        "Point",
				"coordinates": []float64{point.Lon, point.Lat},
			},
		})
	}
	return documents
}

// geoJSONHighLatitudeContractDocuments is a dataset-independent reduction of
// trail-049997/query-000055 from the deterministic seed-42 50k run. The
// Point at the query center separates shape-cell lookup from line geometry,
// while reversed and densified lines rule out direction and segment density.
func geoJSONHighLatitudeContractDocuments() []map[string]any {
	dense := make([][]float64, 0, 11)
	for index := 0; index <= 10; index++ {
		fraction := float64(index) / 10
		point := Coordinate{
			Lat: geoJSONHighLatitudeNear.Lat + fraction*(geoJSONHighLatitudeEnd.Lat-geoJSONHighLatitudeNear.Lat),
			Lon: geoJSONHighLatitudeNear.Lon + fraction*(geoJSONHighLatitudeEnd.Lon-geoJSONHighLatitudeNear.Lon),
		}
		dense = append(dense, []float64{point.Lon, point.Lat})
	}
	return []map[string]any{
		geoJSONPointDocument(geoJSONHighLatitudePointID, geoJSONHighLatitudeQuery),
		geoNativePointDocument(geoJSONHighLatitudeNativePointID, geoJSONHighLatitudeQuery),
		geoJSONPointDocument(geoJSONHighLatitudeNearPointID, geoJSONHighLatitudeNear),
		geoJSONDocument(geoJSONHighLatitudeLineID, [][][]float64{{
			{geoJSONHighLatitudeNear.Lon, geoJSONHighLatitudeNear.Lat},
			{geoJSONHighLatitudeEnd.Lon, geoJSONHighLatitudeEnd.Lat},
		}}),
		geoJSONDocument(geoJSONHighLatitudeReversedLineID, [][][]float64{{
			{geoJSONHighLatitudeEnd.Lon, geoJSONHighLatitudeEnd.Lat},
			{geoJSONHighLatitudeNear.Lon, geoJSONHighLatitudeNear.Lat},
		}}),
		geoJSONDocument(geoJSONHighLatitudeDenseLineID, [][][]float64{dense}),
	}
}

// geoJSONCoverageContractDocuments adds one ordinary-latitude failure and one
// passing control to the original seed-42 high-latitude reduction. Every
// GeoJSON point has a native _geo twin at exactly the same coordinate, so the
// same filter proves whether the query itself is valid.
func geoJSONCoverageContractDocuments() []map[string]any {
	documents := geoJSONHighLatitudeContractDocuments()
	return append(documents,
		geoJSONPointDocument(geoJSONMidLatitudePointID, geoJSONMidLatitudeQuery),
		geoNativePointDocument(geoJSONMidLatitudeNativePointID, geoJSONMidLatitudeQuery),
		geoJSONPointDocument(geoJSONZurichPointID, geoJSONZurichQuery),
		geoNativePointDocument(geoJSONZurichNativePointID, geoJSONZurichQuery),
	)
}

func geoJSONPointDocument(id string, point Coordinate) map[string]any {
	return map[string]any{
		"id": id,
		"_geojson": map[string]any{
			"type":       "Feature",
			"properties": map[string]any{},
			"geometry": map[string]any{
				"type":        "Point",
				"coordinates": []float64{point.Lon, point.Lat},
			},
		},
	}
}

func geoNativePointDocument(id string, point Coordinate) map[string]any {
	return map[string]any{
		"id": id,
		"_geo": map[string]any{
			"lat": point.Lat,
			"lng": point.Lon,
		},
	}
}

func geoJSONCoverageContractCases() ([]geoJSONHighLatitudeContractCase, error) {
	all := idSet(
		geoJSONHighLatitudePointID,
		geoJSONHighLatitudeNativePointID,
		geoJSONHighLatitudeNearPointID,
		geoJSONHighLatitudeLineID,
		geoJSONHighLatitudeReversedLineID,
		geoJSONHighLatitudeDenseLineID,
	)
	viewport := GeoViewport{
		North: geoJSONHighLatitudeQuery.Lat + 0.01,
		West:  geoJSONHighLatitudeQuery.Lon - 0.01,
		South: geoJSONHighLatitudeQuery.Lat - 0.01,
		East:  geoJSONHighLatitudeQuery.Lon + 0.01,
	}
	bboxFilter, err := geoBoundingBoxFilter(viewport)
	if err != nil {
		return nil, err
	}
	cases := []geoJSONHighLatitudeContractCase{
		{
			name:     "40 km high-latitude center controls",
			filter:   geoRadiusFilter(geoJSONHighLatitudeQuery, 40_000, 100),
			radius:   40_000,
			expected: idSet(geoJSONHighLatitudePointID, geoJSONHighLatitudeNativePointID),
		},
		{
			name:     "100 km high-latitude point and line coverage",
			filter:   geoRadiusFilter(geoJSONHighLatitudeQuery, 100_000, 100),
			radius:   100_000,
			expected: all,
		},
		{
			name:     "small high-latitude bounding box controls",
			filter:   bboxFilter,
			expected: idSet(geoJSONHighLatitudePointID, geoJSONHighLatitudeNativePointID),
		},
		{
			name:     "500 m ordinary-latitude failing-cell controls",
			filter:   geoRadiusFilter(geoJSONMidLatitudeQuery, 500, 100),
			radius:   500,
			expected: idSet(geoJSONMidLatitudePointID, geoJSONMidLatitudeNativePointID),
		},
		{
			name:     "25 km ordinary-latitude recovery controls",
			filter:   geoRadiusFilter(geoJSONMidLatitudeQuery, 25_000, 100),
			radius:   25_000,
			expected: idSet(geoJSONMidLatitudePointID, geoJSONMidLatitudeNativePointID),
		},
		{
			name:     "500 m Zurich passing-cell controls",
			filter:   geoRadiusFilter(geoJSONZurichQuery, 500, 100),
			radius:   500,
			expected: idSet(geoJSONZurichPointID, geoJSONZurichNativePointID),
		},
	}
	return cases, nil
}

func geoJSONContractDiagnosticCaseReport(
	test geoJSONHighLatitudeContractCase,
	actual map[string]struct{},
) (GeoJSONContractDiagnosticCaseReport, bool) {
	missing := make(map[string]struct{})
	for id := range test.expected {
		if _, found := actual[id]; !found {
			missing[id] = struct{}{}
		}
	}
	unexpected := make(map[string]struct{})
	for id := range actual {
		if _, found := test.expected[id]; !found {
			unexpected[id] = struct{}{}
		}
	}
	report := GeoJSONContractDiagnosticCaseReport{
		Name:          test.name,
		Filter:        test.filter,
		RadiusMeters:  test.radius,
		ExpectedIDs:   sortedIDs(test.expected),
		ActualIDs:     sortedIDs(actual),
		MissingIDs:    sortedIDs(missing),
		UnexpectedIDs: sortedIDs(unexpected),
	}
	return report, len(missing) > 0 || len(unexpected) > 0
}

func geoJSONContractDocuments() []map[string]any {
	return []map[string]any{
		geoJSONDocument("line", [][][]float64{{
			{8.5200, 47.3769}, {8.5600, 47.3769},
		}}),
		geoJSONDocument("multiline", [][][]float64{
			{{8.2000, 47.4000}, {8.2100, 47.4000}},
			{{8.5200, 47.3860}, {8.5600, 47.3860}},
		}),
		geoJSONDocument("gap", [][][]float64{
			{{8.5000, 47.4100}, {8.5100, 47.4100}},
			{{8.5700, 47.4100}, {8.5800, 47.4100}},
		}),
		geoJSONDocument("bbox-crossing", [][][]float64{{
			{-1, 40}, {1, 40},
		}}),
		geoJSONDocument("bbox-disjoint", [][][]float64{{
			{-1, 30}, {1, 32},
		}}),
		geoJSONDocument("bbox-multiline-gap", [][][]float64{
			{{-1, 20}, {-0.5, 20}},
			{{0.5, 20}, {1, 20}},
		}),
		geoJSONDocument("bbox-boundary-touch", [][][]float64{{
			{-1, 10}, {0, 10},
		}}),
		// RFC 7946 recommends cutting geometries at the antimeridian. Keeping
		// the two line parts separate also prevents an imaginary segment from
		// crossing the rest of the world.
		geoJSONDocument("bbox-dateline", [][][]float64{
			{{179.5, 0}, {180, 0}},
			{{-180, 0}, {-179.5, 0}},
		}),
	}
}

func geoJSONRadiusContractCases() []geoJSONRadiusContractCase {
	return []geoJSONRadiusContractCase{
		{name: "line midpoint", point: Coordinate{Lat: 47.3769, Lon: 8.5400}, radius: 50, expected: idSet("line")},
		{name: "second multiline part", point: Coordinate{Lat: 47.3860, Lon: 8.5400}, radius: 50, expected: idSet("multiline")},
		{name: "multiline gap", point: Coordinate{Lat: 47.4100, Lon: 8.5400}, radius: 100, expected: idSet()},
	}
}

func geoJSONBoundingBoxContractCases() []geoJSONBoundingBoxContractCase {
	return []geoJSONBoundingBoxContractCase{
		{
			name:     "line crosses viewport with endpoints outside",
			viewport: GeoViewport{North: 40.1, West: -0.1, South: 39.9, East: 0.1},
			expected: idSet("bbox-crossing"),
		},
		{
			name:     "geometry bbox overlaps but line is disjoint",
			viewport: GeoViewport{North: 32, West: -1, South: 31.8, East: -0.8},
			expected: idSet(),
		},
		{
			name:     "multiline gap does not become a segment",
			viewport: GeoViewport{North: 20.1, West: -0.1, South: 19.9, East: 0.1},
			expected: idSet(),
		},
		{
			name:     "viewport boundary touch is inclusive",
			viewport: GeoViewport{North: 10.1, West: 0, South: 9.9, East: 0.2},
			expected: idSet("bbox-boundary-touch"),
		},
		{
			name:     "antimeridian viewport is split",
			viewport: GeoViewport{North: 0.1, West: 179.8, South: -0.1, East: -179.8},
			expected: idSet("bbox-dateline"),
		},
	}
}

// These pairs are the five near-boundary false negatives observed with the
// three-argument default-radius query in the deterministic seed-42 standard
// corpus. Keeping the original generated geometries makes the preflight prove
// that the conservative four-argument plans remain candidate-complete without
// treating the historical default miss as required future behavior.
func geoJSONBoundaryRegressionFixtures() ([]map[string]any, []geoJSONBoundaryRegressionCase, error) {
	dataset, err := generateDataset(5_000, 36, 42)
	if err != nil {
		return nil, nil, err
	}
	trails := make(map[string]Trail, len(dataset.Trails))
	for _, trail := range dataset.Trails {
		trails[trail.ID] = trail
	}
	queries := make(map[string]QueryPoint, len(dataset.QueryPoints))
	for _, query := range dataset.QueryPoints {
		queries[query.ID] = query
	}
	specs := []struct {
		query  string
		radius float64
		trail  string
	}{
		{query: "query-000000", radius: 500, trail: "trail-003027"},
		{query: "query-000000", radius: 5_000, trail: "trail-000175"},
		{query: "query-000004", radius: 25_000, trail: "trail-002789"},
		{query: "query-000010", radius: 5_000, trail: "trail-000702"},
		{query: "query-000013", radius: 25_000, trail: "trail-000277"},
	}
	documents := make([]map[string]any, 0, len(specs))
	cases := make([]geoJSONBoundaryRegressionCase, 0, len(specs))
	for _, spec := range specs {
		trail, trailFound := trails[spec.trail]
		query, queryFound := queries[spec.query]
		if !trailFound || !queryFound {
			return nil, nil, fmt.Errorf("generated GeoJSON regression fixture %s/%s is missing", spec.query, spec.trail)
		}
		documents = append(documents, map[string]any{"id": trail.ID, "_geojson": geoJSONFeature(trail)})
		cases = append(cases, geoJSONBoundaryRegressionCase{
			name:   spec.query + "/" + spec.trail,
			point:  query.Point,
			radius: spec.radius,
			id:     trail.ID,
		})
	}
	return documents, cases, nil
}
