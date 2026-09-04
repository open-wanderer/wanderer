package importer

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"

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

func TestReplaceTrailTrackReplacesTrackAndKeepsUserFields(t *testing.T) {
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	trails := core.NewBaseCollection("trails")
	trails.Fields.Add(
		&core.TextField{Name: "name"},
		&core.TextField{Name: "description"},
		&core.FileField{Name: "gpx", MaxSelect: 1, MaxSize: 5 << 20},
		&core.NumberField{Name: "distance"},
		&core.NumberField{Name: "elevation_gain"},
		&core.NumberField{Name: "elevation_loss"},
		&core.NumberField{Name: "duration"},
		&core.NumberField{Name: "lat"},
		&core.NumberField{Name: "lon"},
		&core.TextField{Name: "polyline", Max: util.PolylineMaxLength},
		&core.NumberField{Name: "min_lat"},
		&core.NumberField{Name: "max_lat"},
		&core.NumberField{Name: "min_lon"},
		&core.NumberField{Name: "max_lon"},
		&core.NumberField{Name: "bounding_box_diagonal"},
	)
	if err := app.Save(trails); err != nil {
		t.Fatal(err)
	}

	oldGPX, err := filesystem.NewFileFromBytes([]byte("<gpx></gpx>"), "old.gpx")
	if err != nil {
		t.Fatal(err)
	}
	trail := core.NewRecord(trails)
	trail.Load(map[string]any{
		"name":           "Edited by user",
		"description":    "Edited description",
		"distance":       1.0,
		"elevation_gain": 1.0,
		"elevation_loss": 1.0,
		"duration":       1.0,
		"lat":            1.0,
		"lon":            1.0,
	})
	trail.Set("gpx", oldGPX)
	if err := app.Save(trail); err != nil {
		t.Fatal(err)
	}
	oldFile := trail.GetString("gpx")

	item := pluginsystem.TrailImport{
		Name:        "Provider name",
		Description: "Provider description",
		Track:       gpxTrack(),
		Metadata:    map[string]any{"distance": 4321.0},
	}
	if err := ReplaceTrailTrack(app, trail, item); err != nil {
		t.Fatalf("ReplaceTrailTrack: %v", err)
	}

	saved, err := app.FindRecordById("trails", trail.Id)
	if err != nil {
		t.Fatal(err)
	}
	if saved.GetString("name") != "Edited by user" || saved.GetString("description") != "Edited description" {
		t.Fatalf("user fields changed: name=%q description=%q", saved.GetString("name"), saved.GetString("description"))
	}
	if gpx := saved.GetString("gpx"); gpx == "" || gpx == oldFile || !strings.HasSuffix(gpx, ".gpx") {
		t.Fatalf("expected replaced gpx file, got %q (old %q)", gpx, oldFile)
	}
	if got := saved.GetFloat("distance"); got != 4321 {
		t.Fatalf("expected provider distance override, got %v", got)
	}
	if got := saved.GetFloat("elevation_gain"); got != 20 {
		t.Fatalf("expected elevation gain from track, got %v", got)
	}
	if got := saved.GetFloat("duration"); got != 600 {
		t.Fatalf("expected duration from track, got %v", got)
	}
	if saved.GetFloat("lat") != 46 || saved.GetFloat("lon") != 8 {
		t.Fatalf("expected start from track, got %v/%v", saved.GetFloat("lat"), saved.GetFloat("lon"))
	}
	if saved.GetString("polyline") == "" {
		t.Fatal("expected geometry to be stored with the track")
	}
	if saved.GetFloat("min_lat") != 46 || saved.GetFloat("max_lat") != 46.001 ||
		saved.GetFloat("min_lon") != 8 || saved.GetFloat("max_lon") != 8.001 ||
		saved.GetFloat("bounding_box_diagonal") <= 0 {
		t.Fatalf("unexpected stored geometry: min=%v/%v max=%v/%v diagonal=%v",
			saved.GetFloat("min_lat"), saved.GetFloat("min_lon"),
			saved.GetFloat("max_lat"), saved.GetFloat("max_lon"),
			saved.GetFloat("bounding_box_diagonal"))
	}
}

func TestValidateTrackRequiresUsableGeometry(t *testing.T) {
	track := func(points string) pluginsystem.Track {
		content := `<?xml version="1.0" encoding="UTF-8"?><gpx version="1.1" creator="test" xmlns="http://www.topografix.com/GPX/1/1"><trk><trkseg>` + points + `</trkseg></trk></gpx>`
		return pluginsystem.Track{Format: "gpx", ContentBase64: base64.StdEncoding.EncodeToString([]byte(content))}
	}
	cases := []struct {
		name   string
		points string
		ok     bool
	}{
		{"two points", `<trkpt lat="47.1" lon="8.1"><ele>400</ele></trkpt><trkpt lat="47.2" lon="8.2"><ele>410</ele></trkpt>`, true},
		{"equivalent GPX namespace prefix", `<p:trkpt xmlns:p="http://www.topografix.com/GPX/1/1" lat="47.1" lon="8.1"></p:trkpt><p:trkpt xmlns:p="http://www.topografix.com/GPX/1/1" lat="47.2" lon="8.2"></p:trkpt>`, true},
		{"single point", `<trkpt lat="47.1" lon="8.1"></trkpt>`, false},
		{"no points", ``, false},
		{"latitude out of range", `<trkpt lat="91" lon="8.1"></trkpt><trkpt lat="47.2" lon="8.2"></trkpt>`, false},
		{"longitude out of range", `<trkpt lat="47.1" lon="-181"></trkpt><trkpt lat="47.2" lon="8.2"></trkpt>`, false},
		{"missing coordinates", `<trkpt></trkpt><trkpt></trkpt>`, false},
		{"only one point with coordinates", `<trkpt></trkpt><trkpt lat="47.2" lon="8.2"></trkpt>`, false},
		{"missing latitude", `<trkpt lon="8"></trkpt><trkpt lat="47.2" lon="8.2"></trkpt>`, false},
		{"real null island points are fine", `<trkpt lat="0" lon="0"></trkpt><trkpt lat="0.1" lon="0.1"></trkpt>`, true},
		{"unparseable latitude", `<trkpt lat="abc" lon="8.1"></trkpt><trkpt lat="47.2" lon="8.2"></trkpt>`, false},
		{"not a number", `<trkpt lat="NaN" lon="8.1"></trkpt><trkpt lat="47.2" lon="8.2"></trkpt>`, false},
		{"infinite", `<trkpt lat="Inf" lon="8.1"></trkpt><trkpt lat="47.2" lon="8.2"></trkpt>`, false},
		{"extension namespace lookalikes", `<x:trkpt xmlns:x="urn:extension" lat="47.1" lon="8.1"></x:trkpt><x:trkpt xmlns:x="urn:extension" lat="47.2" lon="8.2"></x:trkpt>`, false},
		{"nested extension lookalikes", `<extensions><trkpt lat="47.1" lon="8.1"></trkpt><trkpt lat="47.2" lon="8.2"></trkpt></extensions>`, false},
	}
	for _, tc := range cases {
		err := ValidateTrack(track(tc.points))
		if tc.ok && err != nil {
			t.Errorf("%s: expected a usable track, got %v", tc.name, err)
		}
		if !tc.ok && err == nil {
			t.Errorf("%s: expected the track to be refused", tc.name)
		}
	}
}
