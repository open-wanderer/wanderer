package pluginsystem

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// This opt-in test executes the release bundle through the actual backend
// worker. Ordinary Go tests do not require TinyGo or a backend executable.
func TestHammerheadDashboardBuiltWASMElevations(t *testing.T) {
	bundle := os.Getenv("WANDERER_HAMMERHEAD_BUNDLE")
	worker := os.Getenv("WANDERER_PLUGIN_WORKER_BIN")
	if bundle == "" || worker == "" {
		t.Skip("set WANDERER_HAMMERHEAD_BUNDLE and WANDERER_PLUGIN_WORKER_BIN")
	}
	plugin, err := LoadLocalPlugin(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if plugin.Manifest.ManifestVersion != "1.0" || plugin.Manifest.Version != "0.1.2" {
		t.Fatalf("expected Dashboard bundle 0.1.2/manifest 1.0, got %s/%s", plugin.Manifest.Version, plugin.Manifest.ManifestVersion)
	}
	// Production HTTP policy rejects loopback even with allowPrivate. Keep that
	// policy intact and bind the synthetic provider to a private interface.
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		t.Fatal(err)
	}
	privateIP := ""
	for _, address := range addresses {
		ip, _, err := net.ParseCIDR(address.String())
		if err == nil && ip.To4() != nil && ip.IsPrivate() && !ip.IsLoopback() {
			privateIP = ip.String()
			break
		}
	}
	if privateIP == "" {
		t.Fatal("built guest test requires a private non-loopback interface")
	}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/auth/token":
			_, _ = w.Write([]byte(`{"access_token":"header.eyJzdWIiOiJ1c2VyLTEyMyJ9.signature"}`))
		case "/v1/users/user-123/activities/ride-1/details":
			_, _ = w.Write([]byte(`{"activityData":{"id":"ride-1","name":"Synthetic ride","createdAt":"2026-10-04T10:00:00Z"},"recordData":{"timestamp":[1700000000,1700000010,1700000020],"lat":[46.1,46.2,46.3],"lng":[8.1,8.2,8.3],"elevation":[487.5,502.75,495.25]}}`))
		case "/v1/users/user-123/routes/route-1":
			_, _ = w.Write([]byte("{\"id\":\"route-1\",\"name\":\"Synthetic planned route\",\"createdAt\":\"2026-10-04T10:00:00Z\",\"routePolyline\":\"_p~iF~ps|U_ulLnnqC_mqNvxq`@\",\"elevation\":{\"polyline\":\"_gjaR_gayB~jbvD\"}}"))
		default:
			t.Errorf("unexpected provider path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	_ = server.Listener.Close()
	server.Listener, err = net.Listen("tcp", net.JoinHostPort(privateIP, "0"))
	if err != nil {
		t.Fatal(err)
	}
	server.Start()
	defer server.Close()
	// Only the resolved endpoint binding changes; guest, manifest permissions,
	// HTTP policy, and backend runtime remain the production implementations.
	policy := testHostPolicy(t, server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	session, err := (WorkerRuntime{Executable: worker}).OpenSession(ctx, *plugin, policy)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close(context.Background())
	for _, tt := range []struct {
		kind, id, export string
		want             []float64
	}{
		{"completed", "ride-1", "get_activity_detail_v1", []float64{487.5, 502.75, 495.25}},
		{"planned", "route-1", "get_route_detail_v1", []float64{100, 120, 90}},
	} {
		t.Run(tt.kind, func(t *testing.T) {
			input, err := json.Marshal(map[string]any{
				"instance": InstanceRef{ID: "synthetic", PluginID: "hammerhead"},
				"auth":     map[string]any{"email": "test@example.invalid", "password": "synthetic-password"},
				"summary":  TrailSummary{Kind: tt.kind, Source: TrailImportSource{Provider: "hammerhead", ExternalID: tt.id}},
			})
			if err != nil {
				t.Fatal(err)
			}
			output, err := session.Call(ctx, tt.export, input)
			if err != nil {
				t.Fatal(err)
			}
			var detail struct {
				Item TrailImport `json:"item"`
			}
			if err := json.Unmarshal(output, &detail); err != nil {
				t.Fatal(err)
			}
			if detail.Item.Kind != tt.kind || detail.Item.Source.ExternalID != tt.id || detail.Item.Track.Format != "gpx" {
				t.Fatalf("unexpected import identity: kind=%s id=%s format=%s", detail.Item.Kind, detail.Item.Source.ExternalID, detail.Item.Track.Format)
			}
			data, err := base64.StdEncoding.DecodeString(detail.Item.Track.ContentBase64)
			if err != nil {
				t.Fatal(err)
			}
			var gpx struct {
				Points []struct {
					Elevation float64 `xml:"ele"`
				} `xml:"trk>trkseg>trkpt"`
			}
			if err := xml.Unmarshal(data, &gpx); err != nil {
				t.Fatal(err)
			}
			if len(gpx.Points) != len(tt.want) {
				t.Fatalf("got %d GPX points, want %d", len(gpx.Points), len(tt.want))
			}
			for i, point := range gpx.Points {
				if point.Elevation != tt.want[i] {
					t.Errorf("point %d: GPX elevation=%g metres, want %g", i, point.Elevation, tt.want[i])
				}
			}
		})
	}
}
