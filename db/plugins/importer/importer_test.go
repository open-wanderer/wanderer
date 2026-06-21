package importer

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	pluginsystem "pocketbase/pluginsystem"
	"pocketbase/util"
)

const sampleGPX = `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="test">
  <trk><trkseg>
    <trkpt lat="46.000000" lon="8.000000"><ele>100</ele><time>2026-01-01T10:00:00Z</time></trkpt>
    <trkpt lat="46.001000" lon="8.001000"><ele>120</ele><time>2026-01-01T10:10:00Z</time></trkpt>
  </trkseg></trk>
</gpx>`

func gpxTrack() pluginsystem.Track {
	return pluginsystem.Track{
		Format:        "gpx",
		ContentBase64: base64.StdEncoding.EncodeToString([]byte(sampleGPX)),
	}
}

func TestDecodeAndParseGPX(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		raw, parsed, err := decodeAndParseGPX(gpxTrack())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if parsed == nil {
			t.Fatal("expected parsed gpx")
		}
		if string(raw) != sampleGPX {
			t.Fatal("decoded bytes do not match input")
		}
	})

	t.Run("unsupported format", func(t *testing.T) {
		if _, _, err := decodeAndParseGPX(pluginsystem.Track{Format: "tcx", ContentBase64: "x"}); err == nil {
			t.Fatal("expected error for unsupported format")
		}
	})

	t.Run("empty content", func(t *testing.T) {
		if _, _, err := decodeAndParseGPX(pluginsystem.Track{Format: "gpx"}); err == nil {
			t.Fatal("expected error for empty content")
		}
	})

	t.Run("invalid base64", func(t *testing.T) {
		if _, _, err := decodeAndParseGPX(pluginsystem.Track{Format: "gpx", ContentBase64: "!!!not-base64"}); err == nil {
			t.Fatal("expected error for invalid base64")
		}
	})

	t.Run("invalid gpx", func(t *testing.T) {
		track := pluginsystem.Track{Format: "gpx", ContentBase64: base64.StdEncoding.EncodeToString([]byte("not gpx"))}
		if _, _, err := decodeAndParseGPX(track); err == nil {
			t.Fatal("expected error for invalid gpx")
		}
	})
}

func TestMetricsFromGPX(t *testing.T) {
	_, parsed, err := decodeAndParseGPX(gpxTrack())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	metrics := metricsFromGPX(parsed)
	if metrics.StartLat != 46.0 || metrics.StartLon != 8.0 {
		t.Fatalf("unexpected start point: %v, %v", metrics.StartLat, metrics.StartLon)
	}
	if metrics.Distance <= 0 {
		t.Fatalf("expected positive distance, got %v", metrics.Distance)
	}
	if metrics.ElevationGain <= 0 {
		t.Fatalf("expected positive elevation gain, got %v", metrics.ElevationGain)
	}
	if !metrics.StartTime.Equal(time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected start time: %v", metrics.StartTime)
	}
}

func TestApplyProviderMetrics(t *testing.T) {
	metrics := trailMetrics{
		Distance:      1,
		ElevationGain: 2,
		ElevationLoss: 3,
		Duration:      4,
		StartLat:      46,
		StartLon:      8,
	}

	applyProviderMetrics(&metrics, map[string]any{
		"distance":      1234.5,
		"elevationGain": 234.5,
		"elevationLoss": 45.5,
		"duration":      3600,
	})

	if metrics.Distance != 1234.5 {
		t.Fatalf("distance = %v", metrics.Distance)
	}
	if metrics.ElevationGain != 234.5 {
		t.Fatalf("elevation gain = %v", metrics.ElevationGain)
	}
	if metrics.ElevationLoss != 45.5 {
		t.Fatalf("elevation loss = %v", metrics.ElevationLoss)
	}
	if metrics.Duration != 3600 {
		t.Fatalf("duration = %v", metrics.Duration)
	}
	if metrics.StartLat != 46 || metrics.StartLon != 8 {
		t.Fatalf("provider metadata must not override start point")
	}
}

func TestApplyProviderStart(t *testing.T) {
	_, parsed, err := decodeAndParseGPX(gpxTrack())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	trackIndex := trackDistanceIndexFromGPX(parsed)

	t.Run("uses plausible provider start", func(t *testing.T) {
		metrics := metricsFromGPX(parsed)
		applyProviderStart(&metrics, trackIndex, map[string]any{
			"providerStart": map[string]any{
				"lat": 45.9995,
				"lon": 7.9995,
			},
		})

		if metrics.StartLat != 45.9995 || metrics.StartLon != 7.9995 {
			t.Fatalf("unexpected provider start: %v, %v", metrics.StartLat, metrics.StartLon)
		}
	})

	t.Run("ignores distant provider start", func(t *testing.T) {
		metrics := metricsFromGPX(parsed)
		applyProviderStart(&metrics, trackIndex, map[string]any{
			"providerStart": map[string]any{
				"lat": 47.0,
				"lon": 8.0,
			},
		})

		if metrics.StartLat != 46.0 || metrics.StartLon != 8.0 {
			t.Fatalf("distant provider start should be ignored: %v, %v", metrics.StartLat, metrics.StartLon)
		}
	})

	t.Run("ignores invalid provider start", func(t *testing.T) {
		metrics := metricsFromGPX(parsed)
		applyProviderStart(&metrics, trackIndex, map[string]any{
			"providerStart": map[string]any{
				"lat": 91.0,
				"lon": 8.0,
			},
		})

		if metrics.StartLat != 46.0 || metrics.StartLon != 8.0 {
			t.Fatalf("invalid provider start should be ignored: %v, %v", metrics.StartLat, metrics.StartLon)
		}
	})
}

func TestTrackDistanceIndexNearest(t *testing.T) {
	_, parsed, err := decodeAndParseGPX(gpxTrack())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	trackIndex := trackDistanceIndexFromGPX(parsed)
	total := util.HaversineDistanceMeters(46.0, 8.0, 46.001, 8.001)

	t.Run("start point", func(t *testing.T) {
		distance, ok := trackIndex.nearest(geoPoint{Lat: 46.0, Lon: 8.0})
		if !ok {
			t.Fatal("expected nearest distance")
		}
		if distance.fromStart != 0 {
			t.Fatalf("got %v, want 0", distance.fromStart)
		}
	})

	t.Run("mid segment projection", func(t *testing.T) {
		distance, ok := trackIndex.nearest(geoPoint{Lat: 46.0005, Lon: 8.0005})
		if !ok {
			t.Fatal("expected nearest distance")
		}
		if distance.fromStart < total*0.45 || distance.fromStart > total*0.55 {
			t.Fatalf("got %v, want about half of %v", distance.fromStart, total)
		}
	})

	t.Run("end point", func(t *testing.T) {
		distance, ok := trackIndex.nearest(geoPoint{Lat: 46.001, Lon: 8.001})
		if !ok {
			t.Fatal("expected nearest distance")
		}
		if distance.fromStart < total-0.001 || distance.fromStart > total+0.001 {
			t.Fatalf("got %v, want %v", distance.fromStart, total)
		}
	})
}

func TestApplyProviderMetricsIgnoresEmptyValues(t *testing.T) {
	metrics := trailMetrics{
		Distance:      1,
		ElevationGain: 2,
		ElevationLoss: 3,
		Duration:      4,
	}

	applyProviderMetrics(&metrics, map[string]any{
		"distance":      0,
		"elevationGain": -1,
		"elevationLoss": "",
		"duration":      nil,
	})

	if metrics.Distance != 1 || metrics.ElevationGain != 2 || metrics.ElevationLoss != 3 || metrics.Duration != 4 {
		t.Fatalf("unexpected metrics after empty metadata: %#v", metrics)
	}
}

func TestPublicFromPrivacy(t *testing.T) {
	public := "public"
	private := "private"
	empty := ""

	cases := []struct {
		name          string
		privacy       *string
		defaultPublic bool
		want          bool
	}{
		{"nil keeps default true", nil, true, true},
		{"nil keeps default false", nil, false, false},
		{"explicit public", &public, false, true},
		{"explicit private", &private, true, false},
		{"empty keeps default", &empty, true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := publicFromPrivacy(tc.privacy, tc.defaultPublic); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCategoryIDForImportDoesNotFallbackWhenProviderMappingIsBlank(t *testing.T) {
	item := pluginsystem.TrailImport{
		ActivityType: "biking",
		Metadata: map[string]any{
			"providerCategory": " Ride ",
		},
	}

	if got := categoryIDForImport(nil, item, map[string]string{"Ride": ""}); got != "" {
		t.Fatalf("expected blank provider mapping to suppress activity fallback, got %q", got)
	}
}

func TestProviderCategoryFromImport(t *testing.T) {
	if got := ProviderCategoryFromImport(pluginsystem.TrailImport{
		Metadata: map[string]any{"providerCategory": " Ride "},
	}); got != "Ride" {
		t.Fatalf("got %q", got)
	}
	if got := ProviderCategoryFromImport(pluginsystem.TrailImport{
		Metadata: map[string]any{"sourceSport": " hiking "},
	}); got != "hiking" {
		t.Fatalf("got %q", got)
	}
}

func TestDateFromImport(t *testing.T) {
	started := time.Date(2025, 6, 1, 8, 0, 0, 0, time.UTC)

	t.Run("uses StartedAt", func(t *testing.T) {
		item := pluginsystem.TrailImport{StartedAt: &started}
		if got := dateFromImport(item, trailMetrics{}); !got.Equal(started) {
			t.Fatalf("got %v, want %v", got, started)
		}
	})

	t.Run("falls back to metrics start time", func(t *testing.T) {
		metricStart := time.Date(2024, 1, 2, 3, 0, 0, 0, time.UTC)
		if got := dateFromImport(pluginsystem.TrailImport{}, trailMetrics{StartTime: metricStart}); !got.Equal(metricStart) {
			t.Fatalf("got %v, want %v", got, metricStart)
		}
	})

	t.Run("falls back to now", func(t *testing.T) {
		got := dateFromImport(pluginsystem.TrailImport{}, trailMetrics{})
		if time.Since(got) > time.Minute {
			t.Fatalf("expected ~now, got %v", got)
		}
	})
}

func TestFallbackName(t *testing.T) {
	if got := fallbackName("My Trail"); got != "My Trail" {
		t.Fatalf("got %q", got)
	}
	if got := fallbackName(""); got != "Imported trail" {
		t.Fatalf("got %q", got)
	}
	if got := fallbackName("   "); got != "Imported trail" {
		t.Fatalf("got %q", got)
	}
}

func TestSafeGPXFileName(t *testing.T) {
	cases := map[string]string{
		"track.gpx":        "track.gpx",
		"My Trip":          "My Trip.gpx",
		"":                 "imported-trail.gpx",
		"../../etc/passwd": "passwd.gpx",
		"a:b*c?":           "a-b-c-.gpx",
	}
	for in, want := range cases {
		if got := safeGPXFileName(in); got != want {
			t.Fatalf("safeGPXFileName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSafeMediaFileName(t *testing.T) {
	t.Run("keeps valid filename", func(t *testing.T) {
		if got := safeMediaFileName("photo.jpg"); got != "photo.jpg" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("skips empty and slashed candidates", func(t *testing.T) {
		if got := safeMediaFileName("", "a/b.jpg", "c.png"); got != "c.png" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("falls back to photo.jpg when no candidate", func(t *testing.T) {
		if got := safeMediaFileName(""); got != "photo.jpg" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("rejects slashed traversal candidate", func(t *testing.T) {
		// Candidates containing "/" are rejected outright (not stripped), so a
		// path-traversal candidate falls back to the safe default name.
		if got := safeMediaFileName("../../x.png"); got != "photo.jpg" {
			t.Fatalf("got %q", got)
		}
	})
	t.Run("rejects dotdot candidate", func(t *testing.T) {
		if got := safeMediaFileName(".."); got != "photo.jpg" {
			t.Fatalf("got %q", got)
		}
	})
}

func TestExtensionFromContentTypes(t *testing.T) {
	if got := extensionFromContentTypes("application/x-unknown-xyz"); got != ".jpg" {
		t.Fatalf("expected .jpg fallback, got %q", got)
	}
	if got := extensionFromContentTypes("image/png"); !strings.HasPrefix(got, ".") {
		t.Fatalf("expected an extension, got %q", got)
	}
}

func TestValidateRemoteMediaURLSyntax(t *testing.T) {
	t.Run("rejects non-http scheme", func(t *testing.T) {
		if err := validateRemoteMediaURLSyntax("ftp://example.com/x"); err == nil {
			t.Fatal("expected error for ftp scheme")
		}
	})
	t.Run("rejects missing host", func(t *testing.T) {
		if err := validateRemoteMediaURLSyntax("http://"); err == nil {
			t.Fatal("expected error for missing host")
		}
	})
	t.Run("allows http syntax", func(t *testing.T) {
		if err := validateRemoteMediaURLSyntax("https://8.8.8.8/photo.jpg"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestPhotoFile(t *testing.T) {
	ctx := context.Background()

	t.Run("empty url", func(t *testing.T) {
		photo := pluginsystem.Photo{Source: pluginsystem.MediaSource{Type: "url"}}
		if _, _, err := photoFile(ctx, photo, Options{}, 1024); err == nil {
			t.Fatal("expected error for empty url")
		}
	})

	t.Run("unsupported type", func(t *testing.T) {
		photo := pluginsystem.Photo{Source: pluginsystem.MediaSource{Type: "carrier"}}
		if _, _, err := photoFile(ctx, photo, Options{}, 1024); err == nil {
			t.Fatal("expected error for unsupported source type")
		}
	})
}

func TestPluginMediaBudgetRemainingBytes(t *testing.T) {
	budget := &pluginMediaBudget{}
	if got := budget.remainingBytes(); got != util.DefaultPluginMediaMaxBytes {
		t.Fatalf("got %d, want per-file limit %d", got, util.DefaultPluginMediaMaxBytes)
	}
	budget.bytes = util.DefaultPluginMaxImportMediaBytes - 10
	if got := budget.remainingBytes(); got != 10 {
		t.Fatalf("got %d, want remaining aggregate budget", got)
	}
	budget.bytes = util.DefaultPluginMaxImportMediaBytes
	if got := budget.remainingBytes(); got != 0 {
		t.Fatalf("got %d, want exhausted budget", got)
	}
}

func TestRemoveRawQueryParamOrdered(t *testing.T) {
	raw := "z=last&api_key=secret&a=first&api_key=second"
	if got := removeRawQueryParamOrdered(raw, "api_key"); got != "z=last&a=first" {
		t.Fatalf("unexpected query: %q", got)
	}
}
