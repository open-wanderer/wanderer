package importer

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"net"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"pocketbase/pluginsystem"
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/tkrajina/gpxgo/gpx"
)

type Options struct {
	UserID                      string
	ActorID                     string
	DefaultPublic               bool
	CreateSummitLogForCompleted bool
}

// Result tells the sync loop whether a plugin item created a new trail or was
// skipped because the same provider/external id had already been imported.
type Result struct {
	TrailID string
	Created bool
	Skipped bool
}

// ImportTrail is the boundary between plugin output and wanderer records. It
// validates the provider identity, deduplicates by trail_external_reference,
// stores the GPX/photos, maps GPX metrics onto the trail record, and creates the
// optional related waypoints and summit log.
func ImportTrail(ctx context.Context, app core.App, item pluginsystem.TrailImport, opts Options) (*Result, error) {
	if item.Source.Provider == "" || item.Source.ExternalID == "" {
		return nil, fmt.Errorf("source provider and externalId are required")
	}
	if existing, err := util.FindTrailByExternalReferenceForUser(app, opts.UserID, item.Source.Provider, item.Source.ExternalID); err != nil {
		return nil, err
	} else if existing != nil {
		return &Result{TrailID: existing.Id, Skipped: true}, nil
	}

	gpxBytes, parsedGPX, err := decodeAndParseGPX(item.Track)
	if err != nil {
		return nil, err
	}

	gpxFile, err := filesystem.NewFileFromBytes(gpxBytes, safeGPXFileName(item.Name))
	if err != nil {
		return nil, err
	}

	collection, err := app.FindCollectionByNameOrId("trails")
	if err != nil {
		return nil, err
	}

	record := core.NewRecord(collection)
	metrics := metricsFromGPX(parsedGPX)
	public := publicFromPrivacy(item.Privacy, opts.DefaultPublic)
	categoryID := categoryIDForActivityType(app, item.ActivityType)
	date := dateFromImport(item, metrics)
	photos := photoFiles(ctx, app, item.Photos)

	record.Load(map[string]any{
		"name":           fallbackName(item.Name),
		"description":    item.Description,
		"public":         public,
		"completed":      item.Kind == "completed",
		"distance":       metrics.Distance,
		"elevation_gain": metrics.ElevationGain,
		"elevation_loss": metrics.ElevationLoss,
		"duration":       metrics.Duration,
		"date":           date,
		"lat":            metrics.StartLat,
		"lon":            metrics.StartLon,
		"difficulty":     "easy",
		"category":       categoryID,
		"author":         opts.ActorID,
	})
	record.Set("gpx", gpxFile)
	if len(photos) > 0 {
		record.Set("photos", photos)
	}

	if err := app.Save(record); err != nil {
		return nil, err
	}

	if err := util.EnsureTrailExternalReference(app, record.Id, item.Source.Provider, item.Source.ExternalID); err != nil {
		return nil, err
	}

	if err := createWaypoints(ctx, app, item.Waypoints, opts.ActorID, record.Id); err != nil {
		return nil, err
	}

	if opts.CreateSummitLogForCompleted && item.Kind == "completed" {
		if err := createSummitLog(app, record.Id, opts.ActorID, date, metrics); err != nil {
			return nil, err
		}
	}

	return &Result{TrailID: record.Id, Created: true}, nil
}

type trailMetrics struct {
	Distance      float64
	ElevationGain float64
	ElevationLoss float64
	Duration      float64
	StartLat      float64
	StartLon      float64
	StartTime     time.Time
}

// decodeAndParseGPX keeps the importer strict for now: plugins must return GPX
// as base64 so the host can compute canonical trail metrics itself.
func decodeAndParseGPX(track pluginsystem.Track) ([]byte, *gpx.GPX, error) {
	if track.Format != "gpx" {
		return nil, nil, fmt.Errorf("unsupported track format %q", track.Format)
	}
	if track.ContentBase64 == "" {
		return nil, nil, fmt.Errorf("track contentBase64 is required")
	}

	content, err := base64.StdEncoding.DecodeString(track.ContentBase64)
	if err != nil {
		return nil, nil, fmt.Errorf("decode GPX: %w", err)
	}

	parsed, err := gpx.Parse(bytes.NewReader(content))
	if err != nil {
		return nil, nil, fmt.Errorf("parse GPX: %w", err)
	}

	return content, parsed, nil
}

// metricsFromGPX derives wanderer trail fields from the GPX instead of trusting
// provider/plugin metadata for distance, elevation, duration, and start point.
func metricsFromGPX(gpxData *gpx.GPX) trailMetrics {
	uphillDownhill := gpxData.UphillDownhill()
	movingData := gpxData.MovingData()
	timeBounds := gpxData.TimeBounds()

	metrics := trailMetrics{
		Distance:      gpxData.Length2D(),
		ElevationGain: uphillDownhill.Uphill,
		ElevationLoss: uphillDownhill.Downhill,
		Duration:      movingData.MovingTime + movingData.StoppedTime,
		StartTime:     timeBounds.StartTime,
	}

	for _, track := range gpxData.Tracks {
		for _, segment := range track.Segments {
			if len(segment.Points) == 0 {
				continue
			}
			metrics.StartLat = segment.Points[0].Latitude
			metrics.StartLon = segment.Points[0].Longitude
			return metrics
		}
	}

	return metrics
}

// publicFromPrivacy respects explicit provider privacy when present and falls
// back to the user's wanderer default when the plugin leaves privacy unset.
func publicFromPrivacy(privacy *string, defaultPublic bool) bool {
	if privacy == nil || *privacy == "" {
		return defaultPublic
	}
	return *privacy == "public"
}

// dateFromImport chooses the best available trail date: provider start time,
// GPX start time, then the import time.
func dateFromImport(item pluginsystem.TrailImport, metrics trailMetrics) time.Time {
	if item.StartedAt != nil {
		return *item.StartedAt
	}
	if !metrics.StartTime.IsZero() {
		return metrics.StartTime
	}
	return time.Now()
}

// createWaypoints persists plugin-provided waypoints after the trail exists so
// they can reference the imported trail record.
func createWaypoints(ctx context.Context, app core.App, waypoints []pluginsystem.Waypoint, actorID string, trailID string) error {
	if len(waypoints) == 0 {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	collection, err := app.FindCollectionByNameOrId("waypoints")
	if err != nil {
		return err
	}

	for _, waypoint := range waypoints {
		record := core.NewRecord(collection)
		icon := waypoint.Icon
		if icon == "" {
			icon = "circle"
		}
		record.Load(map[string]any{
			"name":                waypoint.Name,
			"description":         waypoint.Description,
			"lat":                 waypoint.Lat,
			"lon":                 waypoint.Lon,
			"icon":                icon,
			"author":              actorID,
			"distance_from_start": 0,
			"trail":               trailID,
		})
		if err := app.Save(record); err != nil {
			return err
		}
	}

	return nil
}

// photoFiles converts plugin photo descriptors into PocketBase file objects.
// Individual photo failures are logged and skipped so one broken media URL does
// not fail the whole trail import.
func photoFiles(ctx context.Context, app core.App, photos []pluginsystem.Photo) []*filesystem.File {
	if len(photos) == 0 {
		return nil
	}

	files := make([]*filesystem.File, 0, len(photos))
	now := time.Now()
	for _, photo := range photos {
		if err := ctx.Err(); err != nil {
			app.Logger().Warn("skipping plugin photo because import context was cancelled", "error", err)
			return files
		}
		if photo.Source.ExpiresAt != nil && photo.Source.ExpiresAt.Before(now) {
			app.Logger().Warn("skipping expired plugin photo", "external_id", photo.ExternalID)
			continue
		}

		file, err := photoFile(ctx, photo)
		if err != nil {
			app.Logger().Warn("skipping plugin photo", "external_id", photo.ExternalID, "error", err)
			continue
		}
		if file != nil {
			files = append(files, file)
		}
	}

	return files
}

// photoFile fetches one plugin-provided photo source. URL sources are validated
// before PocketBase performs the server-side download.
func photoFile(ctx context.Context, photo pluginsystem.Photo) (*filesystem.File, error) {
	switch photo.Source.Type {
	case "url":
		if photo.Source.URL == "" {
			return nil, fmt.Errorf("photo URL is empty")
		}
		if err := validateRemoteMediaURL(ctx, photo.Source.URL); err != nil {
			return nil, err
		}
		return filesystem.NewFileFromURL(ctx, photo.Source.URL)
	default:
		return nil, fmt.Errorf("unsupported photo source type %q", photo.Source.Type)
	}
}

// validateRemoteMediaURL is a best-effort SSRF guard for plugin-provided media
// URLs that wanderer fetches server-side. It only allows http(s) and rejects
// hosts resolving to loopback, private, link-local or other non-public
// addresses. This is defense-in-depth (plugins are admin-configured/trusted) and
// not fully robust against DNS rebinding, since the actual fetch re-resolves.
func validateRemoteMediaURL(ctx context.Context, rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid media URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported media URL scheme %q", parsed.Scheme)
	}
	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("media URL has no host")
	}

	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("resolve media URL host: %w", err)
	}
	for _, addr := range addrs {
		if !isPublicIP(addr.IP) {
			return fmt.Errorf("media URL host resolves to a non-public address")
		}
	}

	return nil
}

// isPublicIP rejects address ranges that should never be reachable through
// plugin-provided media URLs.
func isPublicIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return false
	}
	return true
}

// createSummitLog mirrors completed imported trails into summit_logs when the
// user has enabled that compatibility option.
func createSummitLog(app core.App, trailID string, actorID string, date time.Time, metrics trailMetrics) error {
	collection, err := app.FindCollectionByNameOrId("summit_logs")
	if err != nil {
		return err
	}

	record := core.NewRecord(collection)
	record.Load(map[string]any{
		"distance":       metrics.Distance,
		"elevation_gain": metrics.ElevationGain,
		"elevation_loss": metrics.ElevationLoss,
		"duration":       metrics.Duration,
		"date":           date,
		"author":         actorID,
		"trail":          trailID,
	})

	return app.Save(record)
}

// categoryIDForActivityType maps common provider activity labels to wanderer's
// built-in categories. Unknown labels intentionally leave the category empty.
func categoryIDForActivityType(app core.App, activityType string) string {
	categoryMap := map[string]string{
		"hiking":   "Hiking",
		"hike":     "Hiking",
		"walking":  "Walking",
		"walk":     "Walking",
		"running":  "Walking",
		"run":      "Walking",
		"biking":   "Biking",
		"cycling":  "Biking",
		"ride":     "Biking",
		"mtb":      "Biking",
		"skiing":   "Skiing",
		"canoeing": "Canoeing",
		"climbing": "Climbing",
	}

	name := categoryMap[strings.ToLower(activityType)]
	if name == "" {
		return ""
	}

	category, _ := app.FindFirstRecordByData("categories", "name", name)
	if category == nil {
		return ""
	}
	return category.Id
}

func fallbackName(name string) string {
	if strings.TrimSpace(name) != "" {
		return name
	}
	return "Imported trail"
}

// safeGPXFileName turns provider trail names into filesystem-safe GPX filenames.
func safeGPXFileName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "imported-trail"
	}
	name = filepath.Base(name)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	name = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '-'
		default:
			return r
		}
	}, name)
	return name + ".gpx"
}

// safeMediaFileName picks the first safe candidate filename and adds a best
// effort extension when providers only expose a content type.
func safeMediaFileName(candidates ...string) string {
	filename := ""
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || strings.Contains(candidate, "/") {
			continue
		}
		filename = candidate
		break
	}
	if filename == "" {
		filename = "photo"
	}
	filename = filepath.Base(filename)
	filename = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '-'
		default:
			return r
		}
	}, filename)
	if filepath.Ext(filename) == "" {
		filename += extensionFromContentTypes(candidates...)
	}
	return filename
}

func extensionFromContentTypes(candidates ...string) string {
	for _, candidate := range candidates {
		if extensions, err := mime.ExtensionsByType(strings.TrimSpace(candidate)); err == nil && len(extensions) > 0 {
			return extensions[0]
		}
	}
	return ".jpg"
}
