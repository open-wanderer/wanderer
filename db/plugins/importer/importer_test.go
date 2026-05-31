package importer

import (
	"context"
	"encoding/base64"
	"net"
	"strings"
	"testing"
	"time"

	pluginsystem "pocketbase/pluginsystem"
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
}

func TestExtensionFromContentTypes(t *testing.T) {
	if got := extensionFromContentTypes("application/x-unknown-xyz"); got != ".jpg" {
		t.Fatalf("expected .jpg fallback, got %q", got)
	}
	if got := extensionFromContentTypes("image/png"); !strings.HasPrefix(got, ".") {
		t.Fatalf("expected an extension, got %q", got)
	}
}

func TestIsPublicIP(t *testing.T) {
	cases := map[string]bool{
		"8.8.8.8":     true,
		"1.1.1.1":     true,
		"127.0.0.1":   false,
		"10.0.0.1":    false,
		"192.168.1.1": false,
		"169.254.1.1": false,
		"0.0.0.0":     false,
		"224.0.0.1":   false,
		"::1":         false,
	}
	for ip, want := range cases {
		if got := isPublicIP(net.ParseIP(ip)); got != want {
			t.Fatalf("isPublicIP(%s) = %v, want %v", ip, got, want)
		}
	}
}

func TestValidateRemoteMediaURL(t *testing.T) {
	ctx := context.Background()

	t.Run("rejects non-http scheme", func(t *testing.T) {
		if err := validateRemoteMediaURL(ctx, "ftp://example.com/x"); err == nil {
			t.Fatal("expected error for ftp scheme")
		}
	})
	t.Run("rejects missing host", func(t *testing.T) {
		if err := validateRemoteMediaURL(ctx, "http://"); err == nil {
			t.Fatal("expected error for missing host")
		}
	})
	t.Run("rejects loopback", func(t *testing.T) {
		if err := validateRemoteMediaURL(ctx, "http://127.0.0.1/photo.jpg"); err == nil {
			t.Fatal("expected error for loopback host")
		}
	})
	t.Run("rejects private", func(t *testing.T) {
		if err := validateRemoteMediaURL(ctx, "http://10.0.0.5/photo.jpg"); err == nil {
			t.Fatal("expected error for private host")
		}
	})
	t.Run("allows public literal ip", func(t *testing.T) {
		if err := validateRemoteMediaURL(ctx, "https://8.8.8.8/photo.jpg"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestPhotoFile(t *testing.T) {
	ctx := context.Background()

	t.Run("empty url", func(t *testing.T) {
		photo := pluginsystem.Photo{Source: pluginsystem.MediaSource{Type: "url"}}
		if _, err := photoFile(ctx, photo); err == nil {
			t.Fatal("expected error for empty url")
		}
	})

	t.Run("unsupported type", func(t *testing.T) {
		photo := pluginsystem.Photo{Source: pluginsystem.MediaSource{Type: "carrier-pigeon"}}
		if _, err := photoFile(ctx, photo); err == nil {
			t.Fatal("expected error for unsupported source type")
		}
	})
}
