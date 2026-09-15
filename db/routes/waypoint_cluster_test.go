package routes

import (
	"context"
	"log/slog"
	"testing"
)

func TestResolveWaypointClusterNamesNamesOnlyNewClusters(t *testing.T) {
	clusters := []waypointPhotoCluster{
		{Photos: []string{"photo"}, Count: 1, Lat: 46.1, Lon: 8.2},
		{Waypoint: "existing", Name: "Existing name", Photos: []string{"photo"}, Count: 2, Lat: 46.2, Lon: 8.3},
		{Name: "Empty cluster", Lat: 46.3, Lon: 8.4},
	}
	calls := 0
	resolver := func(_ context.Context, _ *slog.Logger, lat float64, lon float64, radius float64) (string, error) {
		calls++
		if lat != 46.1 || lon != 8.2 || radius != 75 {
			t.Fatalf("resolver received lat=%v lon=%v radius=%v", lat, lon, radius)
		}
		return "Resolved name", nil
	}

	resolveWaypointClusterNames(context.Background(), nil, clusters, 75, resolver)

	if calls != 1 {
		t.Fatalf("resolver called %d times, want 1", calls)
	}
	if clusters[0].Name != "Resolved name" {
		t.Fatalf("new cluster name = %q, want Resolved name", clusters[0].Name)
	}
	if clusters[1].Name != "Existing name" {
		t.Fatalf("existing waypoint cluster name = %q, want Existing name", clusters[1].Name)
	}
	if clusters[2].Name != "Empty cluster" {
		t.Fatalf("empty cluster name = %q, want Empty cluster", clusters[2].Name)
	}
}
