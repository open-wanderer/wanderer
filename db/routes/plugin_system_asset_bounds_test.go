package routes

import (
	"encoding/json"
	"math"
	"reflect"
	"sort"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestPluginAssetLibraryRequestValidatesBounds(t *testing.T) {
	for _, tc := range []struct {
		name      string
		body      string
		wantError bool
	}{
		{"absent", `{}`, false},
		{"null", `{"bounds":null}`, false},
		{"viewport", `{"bounds":{"west":7,"south":45,"east":9,"north":47}}`, false},
		{"antimeridian", `{"bounds":{"west":170,"south":-10,"east":-170,"north":10}}`, false},
		{"world", `{"bounds":{"west":-180,"south":-90,"east":180,"north":90}}`, false},
		{"zero", `{"bounds":{"west":0,"south":0,"east":0,"north":0}}`, false},
		{"missing", `{"bounds":{"west":7,"south":45,"north":47}}`, true},
		{"null coordinate", `{"bounds":{"west":null,"south":45,"east":9,"north":47}}`, true},
		{"latitude order", `{"bounds":{"west":7,"south":47,"east":9,"north":45}}`, true},
		{"latitude range", `{"bounds":{"west":7,"south":-91,"east":9,"north":47}}`, true},
		{"longitude range", `{"bounds":{"west":-181,"south":45,"east":9,"north":47}}`, true},
		{"infinite", `{"bounds":{"west":1e999,"south":45,"east":9,"north":47}}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var request pluginAssetLibraryRequest
			err := json.Unmarshal([]byte(tc.body), &request)
			if (err != nil) != tc.wantError {
				t.Fatalf("error = %v, wantError = %v", err, tc.wantError)
			}
		})
	}
}

func TestAssetLibraryActionInputValidatesAndForwardsBounds(t *testing.T) {
	for _, builder := range []struct {
		name  string
		build func(core.App, pluginAssetLibraryRequest) (pluginAssetLibraryActionInput, error)
	}{
		{"plugin", func(app core.App, data pluginAssetLibraryRequest) (pluginAssetLibraryActionInput, error) {
			return assetLibraryActionInputForApp(app, data, false)
		}},
		{"wanderer", assetLibraryActionInputForWandererLibrary},
	} {
		t.Run(builder.name, func(t *testing.T) {
			bounds := &pluginAssetSearchBounds{West: 7, South: 45, East: 9, North: 47}
			request, err := builder.build(nil, pluginAssetLibraryRequest{Bounds: bounds})
			if err != nil || request.Bounds == nil || *request.Bounds != *bounds {
				t.Fatalf("bounds were not forwarded: request=%#v, error=%v", request, err)
			}
			for _, invalid := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), -181, 181} {
				bounds.West = invalid
				if _, err := builder.build(nil, pluginAssetLibraryRequest{Bounds: bounds}); err == nil {
					t.Fatalf("invalid west %v was accepted", invalid)
				}
			}
		})
	}
}

func TestAssetLibraryBoundsOverrideRadiusAndPaginate(t *testing.T) {
	app, _ := newAssetImportRouteTestApp(t)
	collection, err := app.FindCollectionByNameOrId("assets")
	if err != nil {
		t.Fatal(err)
	}
	collection.Fields.Add(&core.NumberField{Name: "lat"}, &core.NumberField{Name: "lon"}, &core.DateField{Name: "taken_at"})
	if err := app.Save(collection); err != nil {
		t.Fatal(err)
	}
	for _, photo := range []struct {
		name     string
		lat, lon float64
	}{
		{"east-near", 46, 20},
		{"east-far", 46, 21},
		{"north", 60, 20},
		{"dateline-east", 46, 175},
		{"dateline-west", 46, -175},
		{"zero-longitude", 46, 0},
	} {
		record := core.NewRecord(collection)
		record.Set("author", "actor")
		record.Set("type", "photo")
		record.Set("storage_mode", "link")
		record.Set("external_id", photo.name)
		record.Set("lat", photo.lat)
		record.Set("lon", photo.lon)
		record.Set("taken_at", "2026-01-01T12:00:00Z")
		if err := app.Save(record); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name   string
		bounds pluginAssetSearchBounds
		want   []string
	}{
		{"ordinary", pluginAssetSearchBounds{West: 20, South: 45, East: 21, North: 47}, []string{"east-far", "east-near"}},
		{"antimeridian", pluginAssetSearchBounds{West: 170, South: 45, East: -170, North: 47}, []string{"dateline-east", "dateline-west"}},
		{"world", pluginAssetSearchBounds{West: -180, South: -90, East: 180, North: 90}, []string{"dateline-east", "dateline-west", "east-far", "east-near", "north", "zero-longitude"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := pluginAssetLibraryActionInput{
				Bounds: &tc.bounds,
				Lat:    46, Lon: 8,
				Points: []pluginAssetTrackPoint{{Lat: 46, Lon: 8, Distance: 500}},
			}
			maxDistance := assetLibraryMaxDistance(request, true)
			var names []string
			for page := 1; page <= len(tc.want); page++ {
				pagination := assetLibraryPaginationForRequest(pluginAssetLibraryRequest{Page: page, PerPage: 1}, request, true)
				if !pagination.Enabled {
					t.Fatal("viewport queries must paginate even with route and waypoint coordinates")
				}
				records, hasMore, err := assetLibraryRecords(app, "actor", request, true, pagination)
				if err != nil || len(records) != 1 || hasMore != (page < len(tc.want)) {
					t.Fatalf("page %d: records=%d, hasMore=%v, error=%v", page, len(records), hasMore, err)
				}
				candidate, ok := assetLibraryCandidate(records[0], request, true)
				if !ok || candidate.Distance > maxDistance {
					t.Fatalf("photo inside viewport was excluded by old radius: %#v", candidate)
				}
				if candidate.PointLat != 46 || candidate.PointLon != 8 || candidate.DistanceFromStart != 500 {
					t.Fatalf("route metadata was lost: %#v", candidate)
				}
				names = append(names, records[0].GetString("external_id"))
			}
			sort.Strings(names)
			if !reflect.DeepEqual(names, tc.want) {
				t.Fatalf("got photos %v, want %v", names, tc.want)
			}
		})
	}
}

func TestPluginAssetCursorBindingIncludesViewport(t *testing.T) {
	request := pluginAssetLibraryActionInput{
		Action: "candidates",
		Bounds: &pluginAssetSearchBounds{West: 7, South: 45, East: 9, North: 47},
	}
	first, err := newPluginAssetCursorBinding("user", "actor", "immich", "instance", request, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Bounds.West = 8
	second, err := newPluginAssetCursorBinding("user", "actor", "immich", "instance", request, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.QueryFingerprint == second.QueryFingerprint {
		t.Fatal("moving viewport did not change cursor binding")
	}
}
