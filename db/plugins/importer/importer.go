package importer

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"mime"
	"net/http"
	"net/url"
	urlpath "path"
	"path/filepath"
	"slices"
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
	CategoryMapping             map[string]string
	Manifest                    pluginsystem.Manifest
	Policy                      pluginsystem.RequestPolicyContext
	Auth                        map[string]any
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
	trackIndex := trackDistanceIndexFromGPX(parsedGPX)
	applyProviderStart(&metrics, trackIndex, item.Metadata)
	applyProviderMetrics(&metrics, item.Metadata)
	public := publicFromPrivacy(item.Privacy, opts.DefaultPublic)
	categoryID := categoryIDForImport(app, item, opts.CategoryMapping)
	date := dateFromImport(item, metrics)
	mediaBudget := &pluginMediaBudget{}
	photos := photoFiles(ctx, app, item.Photos, opts, mediaBudget)

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

	if err := util.EnsureTrailExternalReference(app, record.Id, item.Source.Provider, item.Source.ExternalID, opts.Manifest.ID, ProviderCategoryFromImport(item)); err != nil {
		return nil, err
	}

	if err := createWaypoints(ctx, app, item.Waypoints, opts, mediaBudget, record.Id, trackIndex); err != nil {
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

type geoPoint struct {
	Lat float64
	Lon float64
}

type trackDistanceIndex struct {
	points   []indexedTrackPoint
	segments []indexedTrackSegment
}

type indexedTrackPoint struct {
	point    geoPoint
	distance float64
}

type indexedTrackSegment struct {
	start         geoPoint
	end           geoPoint
	startDistance float64
	length        float64
}

const maxProviderStartDistanceMeters = 1000

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

// metricsFromGPX derives fallback trail fields from the GPX. Provider metadata
// may override summary metrics and, when plausible, the displayed start point.
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

// applyProviderStart lets providers correct the displayed trail start when the
// provider's intended start is close to the imported GPX track. Implausible
// starts are ignored so broken metadata does not move trails off their geometry.
func applyProviderStart(metrics *trailMetrics, trackIndex trackDistanceIndex, metadata map[string]any) {
	if metrics == nil || len(metadata) == 0 {
		return
	}
	start, ok := providerStartFromMetadata(metadata)
	if !ok || !providerStartNearTrack(trackIndex, start) {
		return
	}
	metrics.StartLat = start.Lat
	metrics.StartLon = start.Lon
}

func providerStartFromMetadata(metadata map[string]any) (geoPoint, bool) {
	raw, ok := metadata["providerStart"]
	if !ok {
		return geoPoint{}, false
	}
	values, ok := raw.(map[string]any)
	if !ok {
		return geoPoint{}, false
	}
	lat, ok := floatMetadata(values, "lat")
	if !ok {
		lat, ok = floatMetadata(values, "latitude")
	}
	if !ok {
		return geoPoint{}, false
	}
	lon, ok := floatMetadata(values, "lon")
	if !ok {
		lon, ok = floatMetadata(values, "longitude")
	}
	if !ok || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return geoPoint{}, false
	}
	return geoPoint{Lat: lat, Lon: lon}, true
}

func providerStartNearTrack(trackIndex trackDistanceIndex, start geoPoint) bool {
	distance, ok := trackIndex.nearest(start)
	return ok && distance.offTrack <= maxProviderStartDistanceMeters
}

type trackDistance struct {
	fromStart float64
	offTrack  float64
}

func trackDistanceIndexFromGPX(gpxData *gpx.GPX) trackDistanceIndex {
	index := trackDistanceIndex{}
	if gpxData == nil {
		return index
	}
	totalDistance := 0.0
	for _, track := range gpxData.Tracks {
		for _, segment := range track.Segments {
			var previous geoPoint
			hasPrevious := false
			for _, point := range segment.Points {
				current := geoPoint{Lat: point.Latitude, Lon: point.Longitude}
				if !hasPrevious {
					index.points = append(index.points, indexedTrackPoint{
						point:    current,
						distance: totalDistance,
					})
					previous = current
					hasPrevious = true
					continue
				}
				length := util.HaversineDistanceMeters(previous.Lat, previous.Lon, current.Lat, current.Lon)
				if length > 0 {
					index.segments = append(index.segments, indexedTrackSegment{
						start:         previous,
						end:           current,
						startDistance: totalDistance,
						length:        length,
					})
					totalDistance += length
				}
				index.points = append(index.points, indexedTrackPoint{
					point:    current,
					distance: totalDistance,
				})
				previous = current
			}
		}
	}
	return index
}

func (index trackDistanceIndex) nearest(point geoPoint) (trackDistance, bool) {
	var nearest trackDistance
	found := false
	for _, candidate := range index.points {
		offTrack := util.HaversineDistanceMeters(point.Lat, point.Lon, candidate.point.Lat, candidate.point.Lon)
		if !found || offTrack < nearest.offTrack {
			nearest = trackDistance{fromStart: candidate.distance, offTrack: offTrack}
			found = true
		}
	}
	for _, segment := range index.segments {
		offTrack, t := pointToSegmentProjectionMeters(point, segment.start, segment.end)
		fromStart := segment.startDistance + segment.length*t
		if !found || offTrack < nearest.offTrack {
			nearest = trackDistance{fromStart: fromStart, offTrack: offTrack}
			found = true
		}
	}
	return nearest, found
}

func pointToSegmentProjectionMeters(point geoPoint, start geoPoint, end geoPoint) (float64, float64) {
	const earthRadius = 6371000.0
	latRad := point.Lat * math.Pi / 180
	toXY := func(p geoPoint) (float64, float64) {
		x := (p.Lon - point.Lon) * math.Pi / 180 * math.Cos(latRad) * earthRadius
		y := (p.Lat - point.Lat) * math.Pi / 180 * earthRadius
		return x, y
	}

	startX, startY := toXY(start)
	endX, endY := toXY(end)
	dx := endX - startX
	dy := endY - startY
	lengthSquared := dx*dx + dy*dy
	if lengthSquared == 0 {
		return math.Hypot(startX, startY), 0
	}
	t := -(startX*dx + startY*dy) / lengthSquared
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	closestX := startX + t*dx
	closestY := startY + t*dy
	return math.Hypot(closestX, closestY), t
}

// applyProviderMetrics lets plugins preserve provider-provided summary metrics
// where those values are more authoritative than values recalculated from a
// simplified/import GPX. GPX parsing remains mandatory and provides fallback
// metrics plus the start coordinate.
func applyProviderMetrics(metrics *trailMetrics, metadata map[string]any) {
	if metrics == nil || len(metadata) == 0 {
		return
	}
	if value, ok := positiveFloatMetadata(metadata, "distance"); ok {
		metrics.Distance = value
	}
	if value, ok := positiveFloatMetadata(metadata, "elevationGain"); ok {
		metrics.ElevationGain = value
	}
	if value, ok := positiveFloatMetadata(metadata, "elevationLoss"); ok {
		metrics.ElevationLoss = value
	}
	if value, ok := positiveFloatMetadata(metadata, "duration"); ok {
		metrics.Duration = value
	}
}

func positiveFloatMetadata(metadata map[string]any, key string) (float64, bool) {
	value, ok := floatMetadata(metadata, key)
	return value, ok && value > 0
}

func floatMetadata(metadata map[string]any, key string) (float64, bool) {
	switch value := metadata[key].(type) {
	case float64:
		return value, true
	case float32:
		floatValue := float64(value)
		return floatValue, true
	case int:
		floatValue := float64(value)
		return floatValue, true
	case int64:
		floatValue := float64(value)
		return floatValue, true
	case int32:
		floatValue := float64(value)
		return floatValue, true
	case json.Number:
		parsed, err := value.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
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
func createWaypoints(ctx context.Context, app core.App, waypoints []pluginsystem.Waypoint, opts Options, mediaBudget *pluginMediaBudget, trailID string, trackIndex trackDistanceIndex) error {
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
		distanceFromStart := 0.0
		if distance, ok := trackIndex.nearest(geoPoint{Lat: waypoint.Lat, Lon: waypoint.Lon}); ok {
			distanceFromStart = distance.fromStart
		}
		photos := photoFiles(ctx, app, waypoint.Photos, opts, mediaBudget)
		record.Load(map[string]any{
			"name":                waypoint.Name,
			"description":         waypoint.Description,
			"lat":                 waypoint.Lat,
			"lon":                 waypoint.Lon,
			"icon":                icon,
			"author":              opts.ActorID,
			"distance_from_start": distanceFromStart,
			"trail":               trailID,
		})
		if len(photos) > 0 {
			record.Set("photos", photos)
		}
		if err := app.Save(record); err != nil {
			return err
		}
	}

	return nil
}

// photoFiles converts plugin photo descriptors into PocketBase file objects.
// Individual photo failures are logged and skipped so one broken media URL does
// not fail the whole trail import.
type pluginMediaBudget struct {
	items int
	bytes int64
}

func (b *pluginMediaBudget) remainingBytes() int64 {
	remaining := util.DefaultPluginMaxImportMediaBytes - b.bytes
	if remaining < util.DefaultPluginMediaMaxBytes {
		return remaining
	}
	return util.DefaultPluginMediaMaxBytes
}

func photoFiles(ctx context.Context, app core.App, photos []pluginsystem.Photo, opts Options, budget *pluginMediaBudget) []*filesystem.File {
	if len(photos) == 0 {
		return nil
	}

	files := make([]*filesystem.File, 0, len(photos))
	now := time.Now()
	for _, photo := range photos {
		if budget.items >= util.DefaultPluginMaxImportMediaItems {
			app.Logger().Warn("skipping plugin photo because media item limit was reached", "limit", util.DefaultPluginMaxImportMediaItems)
			continue
		}
		if err := ctx.Err(); err != nil {
			app.Logger().Warn("skipping plugin photo because import context was cancelled", "error", err)
			return files
		}
		if photo.Source.ExpiresAt != nil && photo.Source.ExpiresAt.Before(now) {
			app.Logger().Warn("skipping expired plugin photo", "external_id", photo.ExternalID)
			continue
		}
		maxBytes := budget.remainingBytes()
		if maxBytes <= 0 {
			app.Logger().Warn("skipping plugin photo because aggregate media byte limit was reached", "external_id", photo.ExternalID, "limit", util.DefaultPluginMaxImportMediaBytes)
			continue
		}

		file, bytesRead, err := photoFile(ctx, photo, opts, maxBytes)
		if err != nil {
			app.Logger().Warn("skipping plugin photo", "external_id", photo.ExternalID, "error", err)
			continue
		}
		if file != nil {
			files = append(files, file)
			budget.items++
			budget.bytes += bytesRead
		}
	}

	return files
}

// photoFile fetches one plugin-provided photo source. URL sources are validated
// before PocketBase performs the server-side download.
func photoFile(ctx context.Context, photo pluginsystem.Photo, opts Options, maxBytes int64) (*filesystem.File, int64, error) {
	switch photo.Source.Type {
	case "url":
		if photo.Source.URL == "" {
			return nil, 0, fmt.Errorf("photo URL is empty")
		}
		if err := validateRemoteMediaURLSyntax(photo.Source.URL); err != nil {
			return nil, 0, err
		}
		fetched, err := util.FetchPublicURL(ctx, photo.Source.URL, maxBytes)
		if err != nil {
			return nil, 0, err
		}
		file, err := filesystem.NewFileFromBytes(fetched.Body, safeMediaFileName(photo.Filename, urlPathBase(fetched.FinalURL), fetched.ContentType, photo.ContentType))
		return file, int64(len(fetched.Body)), err
	case "connector":
		fetched, err := fetchConnectorMedia(ctx, photo, opts, maxBytes)
		if err != nil {
			return nil, 0, err
		}
		file, err := filesystem.NewFileFromBytes(fetched.Body, safeMediaFileName(photo.Filename, urlPathBase(fetched.FinalURL), fetched.ContentType, photo.ContentType))
		return file, int64(len(fetched.Body)), err
	default:
		return nil, 0, fmt.Errorf("unsupported photo source type %q", photo.Source.Type)
	}
}

func fetchConnectorMedia(ctx context.Context, photo pluginsystem.Photo, opts Options, maxBytes int64) (*util.SafeFetchResult, error) {
	if photo.Source.MediaRef == nil {
		return nil, fmt.Errorf("connector mediaRef is required")
	}
	ref := *photo.Source.MediaRef
	if ref.AssetID != "" && ref.Path == "" {
		return nil, fmt.Errorf("mediaRef.assetId is metadata only; path is required")
	}
	target := pluginsystem.RequestTarget{
		Type:      "connector",
		Connector: ref.Connector,
		Path:      ref.Path,
		Query:     ref.Query,
	}
	resolved, err := pluginsystem.ResolveRequestTarget(opts.Manifest, target, opts.Policy)
	if err != nil {
		return nil, err
	}
	if ref.Auth != "" {
		if !resolved.Connector.SupportsMediaAuth {
			return nil, fmt.Errorf("connector %q does not support media auth", ref.Connector)
		}
		if len(resolved.Connector.Auth) > 0 && !slices.Contains(resolved.Connector.Auth, ref.Auth) {
			return nil, fmt.Errorf("auth context %q is not permitted for connector %q", ref.Auth, ref.Connector)
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resolved.URL.String(), nil)
	if err != nil {
		return nil, err
	}
	if err := pluginsystem.InjectRequestAuthForContext(opts.Manifest, opts.Auth, ref.Auth, req); err != nil {
		return nil, err
	}
	var storageRedirect *storageRedirectTarget
	client, err := util.ConnectorHTTPClient(util.ConnectorHTTPPolicy{
		BaseURL:      resolved.Connector.BaseURL,
		AllowPrivate: resolved.Connector.AllowPrivate,
		TLSMode:      resolved.Connector.TLS.Mode,
		TLSCABundle:  resolved.Connector.TLS.CABundle,
	}, func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("too many redirects")
		}
		previous := resolved.URL
		if len(via) > 0 {
			previous = via[len(via)-1].URL
		}
		if err := pluginsystem.ValidateConnectorRedirect(resolved.Connector, previous, req.URL); err == nil {
			return nil
		}
		origin, err := pluginsystem.ConnectorStorageRedirectOrigin(resolved.Connector, previous, req.URL)
		if err != nil {
			return err
		}
		stripConnectorAuth(req, opts.Manifest, ref.Auth)
		storageRedirect = &storageRedirectTarget{
			URL:    req.URL.String(),
			Origin: origin,
		}
		return http.ErrUseLastResponse
	})
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if storageRedirect != nil && resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return fetchStorageRedirectMedia(ctx, *storageRedirect, maxBytes)
	}
	body, err := util.ReadBoundedForPlugin(resp.Body, maxBytes)
	if err != nil {
		return nil, err
	}
	return &util.SafeFetchResult{Body: body, ContentType: resp.Header.Get("Content-Type"), FinalURL: resp.Request.URL.String()}, nil
}

type storageRedirectTarget struct {
	URL    string
	Origin pluginsystem.ResolvedConnectorOrigin
}

func fetchStorageRedirectMedia(ctx context.Context, redirect storageRedirectTarget, maxBytes int64) (*util.SafeFetchResult, error) {
	storageConnector := pluginsystem.ResolvedConnectorTarget{
		Name:                redirect.Origin.Name,
		BaseURL:             redirect.Origin.BaseURL,
		BasePath:            redirect.Origin.BasePath,
		AllowPrivate:        redirect.Origin.AllowPrivate,
		TLS:                 redirect.Origin.TLS,
		AllowedPathPrefixes: []string{"/"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, redirect.URL, nil)
	if err != nil {
		return nil, err
	}
	client, err := util.ConnectorHTTPClient(util.ConnectorHTTPPolicy{
		BaseURL:      redirect.Origin.BaseURL,
		AllowPrivate: redirect.Origin.AllowPrivate,
		TLSMode:      redirect.Origin.TLS.Mode,
		TLSCABundle:  redirect.Origin.TLS.CABundle,
	}, func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("too many redirects")
		}
		previous := req.URL
		if len(via) > 0 {
			previous = via[len(via)-1].URL
		}
		return pluginsystem.ValidateConnectorRedirect(storageConnector, previous, req.URL)
	})
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := util.ReadBoundedForPlugin(resp.Body, maxBytes)
	if err != nil {
		return nil, err
	}
	return &util.SafeFetchResult{Body: body, ContentType: resp.Header.Get("Content-Type"), FinalURL: resp.Request.URL.String()}, nil
}

func stripConnectorAuth(req *http.Request, manifest pluginsystem.Manifest, authName string) {
	req.Header.Del(pluginsystem.AuthHeaderAuthorization)
	if authName == "" {
		return
	}
	authContext, ok := manifest.Auth.Contexts[authName]
	if !ok {
		return
	}
	if authContext.Name != "" {
		req.Header.Del(authContext.Name)
		req.URL.RawQuery = removeRawQueryParamOrdered(req.URL.RawQuery, authContext.Name)
	}
	if authContext.SecretField != "" {
		req.Header.Del(authContext.SecretField)
		req.URL.RawQuery = removeRawQueryParamOrdered(req.URL.RawQuery, authContext.SecretField)
	}
}

func validateRemoteMediaURLSyntax(rawURL string) error {
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
	return nil
}

func urlPathBase(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return urlpath.Base(parsed.Path)
}

func removeRawQueryParamOrdered(rawQuery string, name string) string {
	if rawQuery == "" || name == "" {
		return rawQuery
	}
	parts := strings.Split(rawQuery, "&")
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		rawName := part
		if idx := strings.Index(rawName, "="); idx >= 0 {
			rawName = rawName[:idx]
		}
		decodedName, err := url.QueryUnescape(rawName)
		if err == nil && decodedName == name {
			continue
		}
		kept = append(kept, part)
	}
	return strings.Join(kept, "&")
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

func categoryIDForImport(app core.App, item pluginsystem.TrailImport, mapping map[string]string) string {
	if category, matched := CategoryFromProviderMapping(app, ProviderCategoryFromImport(item), mapping); matched {
		return category
	}
	return categoryIDForActivityType(app, item.ActivityType)
}

func ProviderCategoryFromImport(item pluginsystem.TrailImport) string {
	value, _ := item.Metadata["providerCategory"].(string)
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	value, _ = item.Metadata["sourceSport"].(string)
	return strings.TrimSpace(value)
}

func CategoryFromProviderMapping(app core.App, providerCategory string, mapping map[string]string) (string, bool) {
	providerCategory = strings.TrimSpace(providerCategory)
	if providerCategory == "" || len(mapping) == 0 {
		return "", false
	}
	rawTarget, matched := mapping[providerCategory]
	if !matched {
		return "", false
	}
	target := strings.TrimSpace(rawTarget)
	if target == "" {
		return "", true
	}
	if category, err := app.FindRecordById("categories", target); err == nil && category != nil {
		return category.Id, true
	}
	category, _ := app.FindFirstRecordByData("categories", "name", target)
	if category == nil {
		return "", false
	}
	return category.Id, true
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
		base := filepath.Base(candidate)
		if base == "." || base == ".." {
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
	if ext := filepath.Ext(filename); ext == "" || ext == "." {
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
