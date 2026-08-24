package importer

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"

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

	if got := categoryIDForImport(nil, item, map[string]CategoryMappingValue{"Ride": {}}); got != "" {
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

func TestCategoryNameForActivityType(t *testing.T) {
	cases := map[string]string{
		"run":        "Running",
		"running":    "Running",
		"VirtualRun": "Running",
		"trailrun":   "Running",
		"jogging":    "Running",
		"walk":       "Walking",
		"hike":       "Hiking",
		"unknown":    "",
	}

	for activityType, want := range cases {
		if got := categoryNameForActivityType(activityType); got != want {
			t.Fatalf("categoryNameForActivityType(%q) = %q, want %q", activityType, got, want)
		}
	}
}

func TestCategoryFromProviderMappingUsesNormalizedCategoryName(t *testing.T) {
	app := setupImporterCategoryTestApp(t)

	category := core.NewRecord(mustFindImporterTestCollection(t, app, "categories"))
	category.Set("name", "Trail Running")
	if err := app.Save(category); err != nil {
		t.Fatal(err)
	}

	got, matched := CategoryFromProviderMapping(app, "Run", map[string]CategoryMappingValue{"Run": {Category: "trail-running"}})
	if !matched {
		t.Fatal("expected provider mapping to match")
	}
	if got != category.Id {
		t.Fatalf("CategoryFromProviderMapping() = %q, want %q", got, category.Id)
	}
}

func TestCategoryTargetFromProviderMappingSupportsSubcategoryPath(t *testing.T) {
	app := setupImporterCategoryTestApp(t)

	category := core.NewRecord(mustFindImporterTestCollection(t, app, "categories"))
	category.Set("name", "Running")
	if err := app.Save(category); err != nil {
		t.Fatal(err)
	}

	subcategory := core.NewRecord(mustFindImporterTestCollection(t, app, "subcategories"))
	subcategory.Set("category", category.Id)
	subcategory.Set("name", "Trail")
	if err := app.Save(subcategory); err != nil {
		t.Fatal(err)
	}

	target, matched := CategoryTargetFromProviderMapping(app, "TrailRun", map[string]CategoryMappingValue{"TrailRun": {Category: "Running", Subcategory: "Trail"}})
	if !matched {
		t.Fatal("expected provider mapping to match")
	}
	if target.CategoryID != category.Id || target.SubcategoryID != subcategory.Id {
		t.Fatalf("CategoryTargetFromProviderMapping() = %#v, want category=%q subcategory=%q", target, category.Id, subcategory.Id)
	}
}

func TestCategoryTargetFromProviderMappingPrefersLiteralCategoryWithSlash(t *testing.T) {
	app := setupImporterCategoryTestApp(t)

	slashCategory := core.NewRecord(mustFindImporterTestCollection(t, app, "categories"))
	slashCategory.Set("name", "Foo/Bar")
	if err := app.Save(slashCategory); err != nil {
		t.Fatal(err)
	}

	parentCategory := core.NewRecord(mustFindImporterTestCollection(t, app, "categories"))
	parentCategory.Set("name", "Foo")
	if err := app.Save(parentCategory); err != nil {
		t.Fatal(err)
	}

	subcategory := core.NewRecord(mustFindImporterTestCollection(t, app, "subcategories"))
	subcategory.Set("category", parentCategory.Id)
	subcategory.Set("name", "Bar")
	if err := app.Save(subcategory); err != nil {
		t.Fatal(err)
	}

	target, matched := CategoryTargetFromProviderMapping(app, "Provider", map[string]CategoryMappingValue{"Provider": {Category: "Foo/Bar"}})
	if !matched {
		t.Fatal("expected provider mapping to match")
	}
	if target.CategoryID != slashCategory.Id || target.SubcategoryID != "" {
		t.Fatalf("CategoryTargetFromProviderMapping() = %#v, want literal category %q", target, slashCategory.Id)
	}
}

func TestCategoryTargetFromProviderMappingDoesNotSplitSlashCategoryName(t *testing.T) {
	app := setupImporterCategoryTestApp(t)

	category := core.NewRecord(mustFindImporterTestCollection(t, app, "categories"))
	category.Set("name", "Foo")
	if err := app.Save(category); err != nil {
		t.Fatal(err)
	}

	subcategory := core.NewRecord(mustFindImporterTestCollection(t, app, "subcategories"))
	subcategory.Set("category", category.Id)
	subcategory.Set("name", "Bar")
	if err := app.Save(subcategory); err != nil {
		t.Fatal(err)
	}

	target, matched := CategoryTargetFromProviderMapping(app, "Provider", map[string]CategoryMappingValue{"Provider": {Category: "Foo/Bar"}})
	if matched {
		t.Fatalf("CategoryTargetFromProviderMapping() = %#v, expected slash category name not to be split", target)
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

func TestPhotoAssetInputPreservesZeroCoordinates(t *testing.T) {
	zero := 0.0
	input := photoAssetInput(Options{
		UserID: "user1",
		Manifest: pluginsystem.Manifest{
			ID: "immich",
		},
	}, PhotoAssetTarget{}, pluginsystem.Photo{
		ExternalID: "asset1",
		Lat:        &zero,
		Lon:        &zero,
	})

	if !input.HasLat || !input.HasLon {
		t.Fatalf("expected explicit zero coordinates to be marked present: %#v", input)
	}
	if input.Lat != 0 || input.Lon != 0 {
		t.Fatalf("unexpected coordinates: %v, %v", input.Lat, input.Lon)
	}
}

func TestPhotoAssetInputPreservesTakenAt(t *testing.T) {
	takenAt := time.Date(2024, 7, 25, 10, 27, 38, 339000000, time.UTC)
	input := photoAssetInput(Options{
		UserID: "user1",
		Manifest: pluginsystem.Manifest{
			ID: "immich",
		},
	}, PhotoAssetTarget{}, pluginsystem.Photo{
		ExternalID: "asset1",
		TakenAt:    &takenAt,
	})

	if input.TakenAt == nil || !input.TakenAt.Equal(takenAt) {
		t.Fatalf("expected taken_at to be preserved, got %#v", input.TakenAt)
	}
}

func TestPhotoFile(t *testing.T) {
	ctx := context.Background()

	t.Run("empty url", func(t *testing.T) {
		photo := pluginsystem.Photo{Source: pluginsystem.MediaSource{Type: "url"}}
		if _, _, err := photoFile(ctx, photo, Options{}, 1024, nil); err == nil {
			t.Fatal("expected error for empty url")
		}
	})

	t.Run("unsupported type", func(t *testing.T) {
		photo := pluginsystem.Photo{Source: pluginsystem.MediaSource{Type: "carrier"}}
		if _, _, err := photoFile(ctx, photo, Options{}, 1024, nil); err == nil {
			t.Fatal("expected error for unsupported source type")
		}
	})
}

func TestEffectivePluginMediaMaxBytes(t *testing.T) {
	manifest := pluginsystem.Manifest{Permissions: pluginsystem.PermissionManifest{
		Downloads: pluginsystem.DownloadPermissions{MaxBytes: 16},
	}}
	for _, tc := range []struct {
		name      string
		manifest  pluginsystem.Manifest
		requested int64
		want      int64
	}{
		{"manifest narrows host limit", manifest, 50, 16},
		{"host limit stays narrower", manifest, 8, 8},
		{"manifest applies to missing host limit", manifest, 0, 16},
		{"host limit without manifest", pluginsystem.Manifest{}, 8, 8},
		{"default without either limit", pluginsystem.Manifest{}, 0, util.DefaultPluginMediaMaxBytes},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := effectivePluginMediaMaxBytes(tc.manifest, tc.requested); got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestFetchPhotoMediaURLUsesManifestPolicy(t *testing.T) {
	originalFetch := fetchPublicPluginMedia
	t.Cleanup(func() { fetchPublicPluginMedia = originalFetch })

	requestedLimit := int64(0)
	fetchPublicPluginMedia = func(_ context.Context, _ string, maxBytes int64) (*util.SafeFetchResult, error) {
		requestedLimit = maxBytes
		return &util.SafeFetchResult{
			Body:        []byte("jpeg"),
			ContentType: "image/jpeg; charset=binary",
			FinalURL:    "https://media.example/photo.jpg",
		}, nil
	}
	manifest := pluginsystem.Manifest{Permissions: pluginsystem.PermissionManifest{
		Downloads: pluginsystem.DownloadPermissions{
			MaxBytes:     4,
			ContentTypes: []string{"image/jpeg"},
		},
	}}
	photo := pluginsystem.Photo{Source: pluginsystem.MediaSource{
		Type: "url",
		URL:  "https://media.example/photo.jpg",
	}}

	if _, err := FetchPhotoMedia(context.Background(), photo, Options{Manifest: manifest}, 10); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if requestedLimit != 4 {
		t.Fatalf("public fetch limit = %d, want manifest limit 4", requestedLimit)
	}

	fetchPublicPluginMedia = func(_ context.Context, _ string, _ int64) (*util.SafeFetchResult, error) {
		return &util.SafeFetchResult{Body: []byte("html"), ContentType: "text/html"}, nil
	}
	if _, err := FetchPhotoMedia(context.Background(), photo, Options{Manifest: manifest}, 10); err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("got error %v, want content type rejection", err)
	}
}

func TestPluginMediaResponseEnforcesManifestPolicy(t *testing.T) {
	manifest := pluginsystem.Manifest{Permissions: pluginsystem.PermissionManifest{
		Downloads: pluginsystem.DownloadPermissions{
			MaxBytes:     4,
			ContentTypes: []string{"image/jpeg"},
		},
	}}

	t.Run("allows declared content type with parameters", func(t *testing.T) {
		resp := pluginMediaTestResponse("1234", "image/jpeg; charset=binary")
		result, err := pluginMediaResponse(resp, manifest, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(result.Body) != "1234" || result.FinalURL != "https://media.example/photo" {
			t.Fatalf("unexpected result: %#v", result)
		}
	})

	t.Run("rejects content type outside manifest", func(t *testing.T) {
		resp := pluginMediaTestResponse("1234", "text/html")
		if _, err := pluginMediaResponse(resp, manifest, 10); err == nil || !strings.Contains(err.Error(), "not allowed") {
			t.Fatalf("got error %v, want content type rejection", err)
		}
	})

	t.Run("rejects missing content type", func(t *testing.T) {
		resp := pluginMediaTestResponse("1234", "")
		if _, err := pluginMediaResponse(resp, manifest, 10); err == nil || !strings.Contains(err.Error(), "invalid content type") {
			t.Fatalf("got error %v, want invalid content type", err)
		}
	})

	t.Run("manifest narrows response size", func(t *testing.T) {
		resp := pluginMediaTestResponse("12345", "image/jpeg")
		maxBytes := effectivePluginMediaMaxBytes(manifest, 10)
		if _, err := pluginMediaResponse(resp, manifest, maxBytes); err == nil || !strings.Contains(err.Error(), "maximum size") {
			t.Fatalf("got error %v, want size rejection", err)
		}
	})

	t.Run("host can narrow response size further", func(t *testing.T) {
		resp := pluginMediaTestResponse("1234", "image/jpeg")
		if _, err := pluginMediaResponse(resp, manifest, 3); err == nil || !strings.Contains(err.Error(), "maximum size") {
			t.Fatalf("got error %v, want size rejection", err)
		}
	})
}

func TestValidatePhotoMimeType(t *testing.T) {
	t.Run("allows detected image type", func(t *testing.T) {
		jpegHeader := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00}
		if err := validatePhotoMimeType(jpegHeader, []string{"image/jpeg", "image/png"}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("rejects response body with unsupported type", func(t *testing.T) {
		err := validatePhotoMimeType([]byte("<html><body>not an image</body></html>"), []string{"image/jpeg", "image/png"})
		if err == nil {
			t.Fatal("expected unsupported MIME type error")
		}
		if !strings.Contains(err.Error(), "text/html") {
			t.Fatalf("error should contain detected MIME type, got %q", err)
		}
	})

	t.Run("skips validation without field restrictions", func(t *testing.T) {
		if err := validatePhotoMimeType([]byte("not an image"), nil); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestFetchPluginMediaWithRetry(t *testing.T) {
	t.Run("succeeds after transient failures", func(t *testing.T) {
		attempts := 0
		want := &util.SafeFetchResult{Body: []byte("ok")}
		got, err := fetchPluginMediaWithRetry(context.Background(), []time.Duration{0, 0}, func() (*util.SafeFetchResult, error) {
			attempts++
			if attempts < 3 {
				return nil, util.ValidatePluginMediaStatus(502)
			}
			return want, nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != want || attempts != 3 {
			t.Fatalf("got result %#v after %d attempts", got, attempts)
		}
	})

	t.Run("does not retry permanent status", func(t *testing.T) {
		attempts := 0
		_, err := fetchPluginMediaWithRetry(context.Background(), []time.Duration{0, 0}, func() (*util.SafeFetchResult, error) {
			attempts++
			return nil, util.ValidatePluginMediaStatus(404)
		})
		if err == nil || attempts != 1 {
			t.Fatalf("got error %v after %d attempts", err, attempts)
		}
	})

	t.Run("stops after configured retries", func(t *testing.T) {
		attempts := 0
		_, err := fetchPluginMediaWithRetry(context.Background(), []time.Duration{0, 0}, func() (*util.SafeFetchResult, error) {
			attempts++
			return nil, util.ValidatePluginMediaStatus(503)
		})
		if err == nil || attempts != 3 || !strings.Contains(err.Error(), "after 3 attempts") {
			t.Fatalf("got error %v after %d attempts", err, attempts)
		}
	})

	t.Run("stops retry wait when context is cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		attempts := 0
		_, err := fetchPluginMediaWithRetry(ctx, []time.Duration{time.Hour}, func() (*util.SafeFetchResult, error) {
			attempts++
			cancel()
			return nil, util.ValidatePluginMediaStatus(502)
		})
		if !errors.Is(err, context.Canceled) || attempts != 1 {
			t.Fatalf("got error %v after %d attempts", err, attempts)
		}
	})
}

func pluginMediaTestResponse(body string, contentType string) *http.Response {
	req, _ := http.NewRequest(http.MethodGet, "https://media.example/photo", nil)
	header := http.Header{}
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}
	return &http.Response{
		Body:    io.NopCloser(strings.NewReader(body)),
		Header:  header,
		Request: req,
	}
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

func TestPhotoImportLimitResolution(t *testing.T) {
	defaults := Options{}
	if got := defaults.maxPhotosPerTrail(); got != 0 {
		t.Fatalf("got trail limit %d, want unlimited", got)
	}
	if got := defaults.maxPhotosPerWaypoint(); got != 0 {
		t.Fatalf("got waypoint limit %d, want unlimited", got)
	}
	if got := defaults.maxPhotosPerSummitLog(); got != 0 {
		t.Fatalf("got summit log limit %d, want unlimited", got)
	}

	empty := Options{PhotoLimits: &PhotoImportLimits{}}
	if got := empty.maxPhotosPerTrail(); got != util.DefaultPluginMaxPhotosPerTrail {
		t.Fatalf("got trail limit %d, want %d", got, util.DefaultPluginMaxPhotosPerTrail)
	}
	if got := empty.maxPhotosPerWaypoint(); got != util.DefaultPluginMaxPhotosPerWaypoint {
		t.Fatalf("got waypoint limit %d, want %d", got, util.DefaultPluginMaxPhotosPerWaypoint)
	}
	if got := empty.maxPhotosPerSummitLog(); got != util.DefaultPluginMaxPhotosPerSummitLog {
		t.Fatalf("got summit log limit %d, want %d", got, util.DefaultPluginMaxPhotosPerSummitLog)
	}

	custom := Options{PhotoLimits: &PhotoImportLimits{
		MaxPhotosPerTrail:     11,
		MaxPhotosPerWaypoint:  3,
		MaxPhotosPerSummitLog: 7,
	}}
	if got := custom.maxPhotosForAssetTarget(PhotoAssetTarget{Trail: "trail1"}); got != 11 {
		t.Fatalf("got trail target limit %d, want 11", got)
	}
	if got := custom.maxPhotosForAssetTarget(PhotoAssetTarget{Waypoint: "waypoint1"}); got != 3 {
		t.Fatalf("got waypoint target limit %d, want 3", got)
	}
	if got := custom.maxPhotosForAssetTarget(PhotoAssetTarget{SummitLog: "summit1"}); got != 7 {
		t.Fatalf("got summit log target limit %d, want 7", got)
	}
}

func TestRemoveRawQueryParamOrdered(t *testing.T) {
	raw := "z=last&api_key=secret&a=first&api_key=second"
	if got := removeRawQueryParamOrdered(raw, "api_key"); got != "z=last&a=first" {
		t.Fatalf("unexpected query: %q", got)
	}
}

func setupImporterCategoryTestApp(t *testing.T) *pbtests.TestApp {
	t.Helper()

	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	categories := core.NewBaseCollection("categories")
	categories.Fields.Add(&core.TextField{Name: "name", Required: true})
	if err := app.Save(categories); err != nil {
		app.Cleanup()
		t.Fatal(err)
	}

	subcategories := core.NewBaseCollection("subcategories")
	subcategories.Fields.Add(
		&core.RelationField{Name: "category", CollectionId: categories.Id, MaxSelect: 1, Required: true},
		&core.TextField{Name: "name", Required: true},
	)
	if err := app.Save(subcategories); err != nil {
		app.Cleanup()
		t.Fatal(err)
	}

	return app
}

func mustFindImporterTestCollection(t *testing.T, app core.App, name string) *core.Collection {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId(name)
	if err != nil {
		t.Fatal(err)
	}

	return collection
}
