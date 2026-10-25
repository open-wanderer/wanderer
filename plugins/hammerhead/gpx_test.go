package main

import (
	"encoding/json"
	"encoding/xml"
	"testing"
)

func TestActivityGPXPreservesDashboardMetres(t *testing.T) {
	var ride activity
	// Provider values are metres, including fractional and below-sea-level heights.
	if err := json.Unmarshal([]byte(`{"activityData":{"name":"Synthetic ride"},"recordData":{"timestamp":[1700000000,1700000010,1700000020,1700000030],"lat":[46.1,46.2,46.3,46.4],"lng":[8.1,8.2,8.3,8.4],"elevation":[487.5,502.75,495.25,-12.5]}}`), &ride); err != nil {
		t.Fatal(err)
	}
	data, err := activityGPX(&ride)
	if err != nil {
		t.Fatal(err)
	}
	assertGPXElevations(t, data, []float64{487.5, 502.75, 495.25, -12.5})
}

func TestTourGPXPreservesPlannedRouteMetres(t *testing.T) {
	data, err := tourGPX(&tour{
		Name:          "Synthetic planned route",
		RoutePolyline: "_p~iF~ps|U_ulLnnqC_mqNvxq`@",
		// Independently encoded provider elevation deltas at scale 100000.
		Elevation: elevation{Polyline: "_gjaR_gayB~jbvD"},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertGPXElevations(t, data, []float64{100, 120, 90})
}

func assertGPXElevations(t *testing.T, data []byte, want []float64) {
	t.Helper()
	var gpx struct {
		Points []struct {
			Elevation float64 `xml:"ele"`
		} `xml:"trk>trkseg>trkpt"`
	}
	if err := xml.Unmarshal(data, &gpx); err != nil {
		t.Fatal(err)
	}
	if len(gpx.Points) != len(want) {
		t.Fatalf("GPX contains %d points, want %d", len(gpx.Points), len(want))
	}
	for i, point := range gpx.Points {
		if point.Elevation != want[i] {
			t.Errorf("point %d: GPX elevation=%g metres, want %g", i, point.Elevation, want[i])
		}
	}
}
