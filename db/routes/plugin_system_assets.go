package routes

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"pocketbase/plugins/importer"
	"pocketbase/pluginsystem"
	assetservice "pocketbase/services/assets"
	"pocketbase/services/pluginhost"
	"pocketbase/util"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	"github.com/tkrajina/gpxgo/gpx"
)

const pluginAssetThumbnailMaxBytes int64 = 8 << 20
const pluginAssetThumbnailCacheMaxBytes int64 = 64 << 20
const pluginAssetThumbnailCacheMaxEntries = 512
const pluginAssetThumbnailCacheTTL = 24 * time.Hour
const defaultAssetPluginMaxWaypoints = 25
const pluginAssetMaintenanceProvider = "maintenance"
const pluginAssetImportMaxIDs = 200
const defaultAssetAutoAttachMaxBatches = 25

var pluginAssetPickerSearchLimits = pluginAssetSearchLimits{
	MaxItems:            100,
	MaxScannedItems:     2500,
	MaxProviderRequests: 10,
}

var pluginAssetAutoAttachSearchLimits = pluginAssetSearchLimits{
	MaxItems:            2500,
	MaxScannedItems:     2500,
	MaxProviderRequests: 16,
}

type pluginAssetThumbnailCacheEntry struct {
	ContentType string
	Body        []byte
	ETag        string
	ExpiresAt   time.Time
	LastAccess  time.Time
}

type pluginAssetThumbnailFetchCall struct {
	done  chan struct{}
	entry pluginAssetThumbnailCacheEntry
	err   error
}

var pluginAssetThumbnailCache = struct {
	sync.Mutex
	items      map[string]pluginAssetThumbnailCacheEntry
	totalBytes int64
}{items: map[string]pluginAssetThumbnailCacheEntry{}}

var pluginAssetThumbnailFetches = struct {
	sync.Mutex
	calls map[string]*pluginAssetThumbnailFetchCall
}{calls: map[string]*pluginAssetThumbnailFetchCall{}}

type pluginAssetLibraryRequest struct {
	PluginID  string `json:"pluginId,omitempty"`
	Action    string `json:"action"`
	TrailID   string `json:"trailId,omitempty"`
	TrailData string `json:"trailData,omitempty"`

	Lat          float64 `json:"lat,omitempty"`
	Lon          float64 `json:"lon,omitempty"`
	TakenAfter   string  `json:"takenAfter,omitempty"`
	TakenBefore  string  `json:"takenBefore,omitempty"`
	DoubleRadius bool    `json:"doubleRadius,omitempty"`

	WaypointID  string   `json:"waypointId,omitempty"`
	SummitLogID string   `json:"summitLogId,omitempty"`
	AssetIDs    []string `json:"assetIds,omitempty"`
	CursorID    string   `json:"cursorId,omitempty"`
	Page        int      `json:"page,omitempty"`
	PerPage     int      `json:"perPage,omitempty"`

	latSet bool
	lonSet bool
}

func (r *pluginAssetLibraryRequest) UnmarshalJSON(data []byte) error {
	type requestAlias pluginAssetLibraryRequest
	var decoded requestAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	decoded.latSet = jsonValueProvided(raw, "lat")
	decoded.lonSet = jsonValueProvided(raw, "lon")
	*r = pluginAssetLibraryRequest(decoded)
	return nil
}

func jsonValueProvided(raw map[string]json.RawMessage, key string) bool {
	value, ok := raw[key]
	return ok && strings.TrimSpace(string(value)) != "null"
}

type pluginAssetLibraryInput struct {
	Instance pluginsystem.InstanceRef      `json:"instance"`
	Auth     map[string]any                `json:"auth,omitempty"`
	Config   map[string]any                `json:"config,omitempty"`
	Limits   importer.PhotoImportLimits    `json:"limits,omitempty"`
	Search   *pluginAssetSearchLimits      `json:"search,omitempty"`
	State    map[string]any                `json:"state,omitempty"`
	Request  pluginAssetLibraryActionInput `json:"request"`
}

type pluginAssetSearchLimits struct {
	MaxItems            int `json:"maxItems,omitempty"`
	MaxScannedItems     int `json:"maxScannedItems,omitempty"`
	MaxProviderRequests int `json:"maxProviderRequests,omitempty"`
}

type pluginAssetLibraryActionInput struct {
	Action       string                  `json:"action"`
	TrailID      string                  `json:"trailId,omitempty"`
	Lat          float64                 `json:"lat,omitempty"`
	Lon          float64                 `json:"lon,omitempty"`
	Points       []pluginAssetTrackPoint `json:"points,omitempty"`
	StartedAt    string                  `json:"startedAt,omitempty"`
	EndedAt      string                  `json:"endedAt,omitempty"`
	TakenAfter   string                  `json:"takenAfter,omitempty"`
	TakenBefore  string                  `json:"takenBefore,omitempty"`
	DoubleRadius bool                    `json:"doubleRadius,omitempty"`
	AssetIDs     []string                `json:"assetIds,omitempty"`
}

type assetLibraryPagination struct {
	Page    int
	PerPage int
	Enabled bool
}

type pluginAssetTrackPoint struct {
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	Distance  float64 `json:"distance,omitempty"`
	Timestamp string  `json:"timestamp,omitempty"`
}

type pluginAssetLibraryOutput struct {
	UserID        string                    `json:"userId,omitempty"`
	Candidates    []pluginAssetCandidate    `json:"candidates,omitempty"`
	Photos        []pluginsystem.Photo      `json:"photos,omitempty"`
	OmittedAssets []pluginAssetOmission     `json:"omittedAssetIds,omitempty"`
	State         map[string]any            `json:"state,omitempty"`
	HasMore       bool                      `json:"hasMore,omitempty"`
	Stats         *pluginAssetSearchStats   `json:"stats,omitempty"`
	TakenAfter    string                    `json:"takenAfter,omitempty"`
	HasTimestamps bool                      `json:"hasTimestamps,omitempty"`
	Error         *pluginsystem.PluginError `json:"error,omitempty"`
}

type pluginAssetOmission struct {
	AssetID string `json:"assetId"`
	Reason  string `json:"reason"`
}

type pluginAssetSearchStats struct {
	ScannedItems int `json:"scannedItems,omitempty"`
}

type pluginAssetCandidate struct {
	Source            string  `json:"source,omitempty"`
	ProviderID        string  `json:"providerId,omitempty"`
	ExternalProvider  string  `json:"externalProvider,omitempty"`
	ExternalID        string  `json:"externalId,omitempty"`
	AssetID           string  `json:"assetId"`
	OriginalFileName  string  `json:"originalFileName"`
	TakenAt           string  `json:"takenAt"`
	Lat               float64 `json:"lat"`
	Lon               float64 `json:"lon"`
	Distance          float64 `json:"distance"`
	PointLat          float64 `json:"pointLat"`
	PointLon          float64 `json:"pointLon"`
	DistanceFromStart float64 `json:"distanceFromStart"`
	City              string  `json:"city,omitempty"`
	Country           string  `json:"country,omitempty"`
	ThumbnailURL      string  `json:"thumbnailUrl,omitempty"`
}

type assetLibraryExternalRef struct {
	Provider string `db:"provider" json:"provider"`
	ID       string `db:"id" json:"id"`
}

type assetLibraryResponse struct {
	HasTimestamps        bool                      `json:"hasTimestamps"`
	Candidates           []pluginAssetCandidate    `json:"candidates"`
	ExistingExternalRefs []assetLibraryExternalRef `json:"existingExternalRefs"`
	HasMore              bool                      `json:"hasMore"`
	TakenAfter           string                    `json:"takenAfter"`
}

type pluginAssetCandidatesResponse struct {
	Candidates      []pluginAssetCandidate `json:"candidates"`
	HasMore         bool                   `json:"hasMore"`
	TakenAfter      string                 `json:"takenAfter"`
	HasTimestamps   bool                   `json:"hasTimestamps"`
	CursorID        string                 `json:"cursorId,omitempty"`
	RestartRequired bool                   `json:"restartRequired,omitempty"`
}

type pluginAssetImportResult struct {
	AssetID  string       `json:"assetId"`
	Waypoint *core.Record `json:"waypoint,omitempty"`
	Asset    *core.Record `json:"asset,omitempty"`
}

type pluginAssetImportResponse struct {
	Imported []pluginAssetImportResult `json:"imported"`
	Omitted  []pluginAssetOmission     `json:"omitted"`
}

type pluginAssetAutoAttachRequest struct {
	TrailID  string `json:"trailId"`
	Provider string `json:"provider"`
}

type pluginAssetAutoAttachPluginResult struct {
	PluginID   string                `json:"pluginId"`
	InstanceID string                `json:"instanceId"`
	Imported   int                   `json:"imported"`
	Omitted    []pluginAssetOmission `json:"omitted"`
	Error      string                `json:"error,omitempty"`
}

type pluginAssetAutoAttachSummary struct {
	OK       bool                                `json:"ok"`
	TrailID  string                              `json:"trailId"`
	Imported int                                 `json:"imported"`
	Omitted  int                                 `json:"omitted"`
	Plugins  []pluginAssetAutoAttachPluginResult `json:"plugins"`
}

type pluginAssetAutoAttachOutcome struct {
	Imported int
	Omitted  []pluginAssetOmission
}

type pluginAssetTrailPhotoMaintenanceCandidate struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Location      string  `json:"location,omitempty"`
	Date          string  `json:"date,omitempty"`
	Created       string  `json:"created,omitempty"`
	Updated       string  `json:"updated,omitempty"`
	Completed     bool    `json:"completed"`
	Public        bool    `json:"public"`
	Distance      float64 `json:"distance,omitempty"`
	ElevationGain float64 `json:"elevation_gain,omitempty"`
	ElevationLoss float64 `json:"elevation_loss,omitempty"`
	Duration      float64 `json:"duration,omitempty"`
	Difficulty    string  `json:"difficulty,omitempty"`
	Thumbnail     string  `json:"thumbnail,omitempty"`
}

type pluginAssetTrailPhotoMaintenanceListResponse struct {
	AssetPluginActive bool                                        `json:"assetPluginActive"`
	Trails            []pluginAssetTrailPhotoMaintenanceCandidate `json:"trails"`
}

type pluginAssetTrailPhotoMaintenanceAttachRequest struct {
	TrailID string `json:"trailId"`
}

type pluginAssetCheckRequest struct {
	Auth   map[string]any `json:"auth,omitempty"`
	Config map[string]any `json:"config,omitempty"`
}

func PluginSystemAssetCheck(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}

	pluginID := e.Request.PathValue("plugin")
	if pluginID == "" {
		return apis.NewBadRequestError("plugin is required", nil)
	}

	var data pluginAssetCheckRequest
	if err := e.BindBody(&data); err != nil {
		return apis.NewBadRequestError("Failed to read request data", err)
	}

	plugin, capability, instance, auth, config, err := assetPluginDraftInvocation(e, pluginID, data.Auth, data.Config)
	if err != nil {
		return err
	}
	output, err := callAssetPlugin(e.Request.Context(), plugin, capability, instance, auth, config, pluginAssetLibraryActionInput{
		Action: "check",
	})
	if err != nil {
		return err
	}
	if output.Error != nil {
		return apis.NewBadRequestError(output.Error.Message, output.Error)
	}
	return e.JSON(http.StatusOK, output)
}

func PluginSystemAssetCandidates(e *core.RequestEvent) error {
	return pluginSystemAssetCall(e, "candidates")
}

func PluginSystemAssetImport(e *core.RequestEvent) error {
	return pluginSystemAssetCall(e, "import")
}

func PluginSystemAssetImportToWaypoint(e *core.RequestEvent) error {
	return pluginSystemAssetCall(e, "import-to-waypoint")
}

func PluginSystemAssetImportToTarget(e *core.RequestEvent) error {
	return pluginSystemAssetCall(e, "import-to-target")
}

func PluginSystemAssetAutoAttach(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}

	var data pluginAssetAutoAttachRequest
	if err := e.BindBody(&data); err != nil {
		return apis.NewBadRequestError("Failed to read request data", err)
	}
	data.TrailID = strings.TrimSpace(data.TrailID)
	data.Provider = strings.TrimSpace(data.Provider)
	if data.Provider == "" {
		data.Provider = "upload"
	}
	if data.TrailID == "" {
		return apis.NewBadRequestError("trailId is required", nil)
	}
	if err := ensureOwnsTrail(e.App, e.Auth.Id, data.TrailID); err != nil {
		return err
	}

	userID := e.Auth.Id
	app := e.App
	go func() {
		if err := AutoAttachAssetPluginsForTrail(context.Background(), app, userID, data.TrailID, data.Provider); err != nil {
			app.Logger().Warn("asset plugin auto attach failed", "user", userID, "trail", data.TrailID, "provider", data.Provider, "error", err)
		}
	}()

	return e.JSON(http.StatusAccepted, map[string]any{"ok": true})
}

func PluginSystemAssetMaintenanceTrails(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}

	pluginCount, err := enabledAssetPluginCount(e.App, e.Auth.Id)
	if err != nil {
		return err
	}
	response := pluginAssetTrailPhotoMaintenanceListResponse{
		AssetPluginActive: pluginCount > 0,
		Trails:            []pluginAssetTrailPhotoMaintenanceCandidate{},
	}
	if pluginCount == 0 {
		return e.JSON(http.StatusOK, response)
	}

	actor, err := e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
	if err != nil {
		return err
	}

	trails, err := e.App.FindRecordsByFilter(
		"trails",
		"author={:author} && completed=true",
		"-date,-created",
		-1,
		0,
		dbx.Params{"author": actor.Id},
	)
	if err != nil {
		return err
	}

	trailsWithGPX := make([]*core.Record, 0, len(trails))
	for _, trail := range trails {
		if strings.TrimSpace(trail.GetString("gpx")) == "" {
			continue
		}
		trailsWithGPX = append(trailsWithGPX, trail)
	}

	visiblePhotos, thumbnails, err := trailAssetMaintenanceState(e.App, trailsWithGPX)
	if err != nil {
		return err
	}

	for _, trail := range trailsWithGPX {
		if visiblePhotos[trail.Id] {
			continue
		}
		response.Trails = append(response.Trails, pluginAssetMaintenanceTrailCandidate(trail, thumbnails[trail.Id]))
	}

	return e.JSON(http.StatusOK, response)
}

func PluginSystemAssetMaintenanceAttach(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}

	var data pluginAssetTrailPhotoMaintenanceAttachRequest
	if err := e.BindBody(&data); err != nil {
		return apis.NewBadRequestError("Failed to read request data", err)
	}
	data.TrailID = strings.TrimSpace(data.TrailID)
	if data.TrailID == "" {
		return apis.NewBadRequestError("trailId is required", nil)
	}
	if err := ensureOwnsTrail(e.App, e.Auth.Id, data.TrailID); err != nil {
		return err
	}

	summary, err := autoAttachAssetPluginsForTrail(e.Request.Context(), e.App, e.Auth.Id, data.TrailID, pluginAssetMaintenanceProvider)
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, summary)
}

func PluginSystemAssetThumbnail(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}
	pluginID := e.Request.PathValue("plugin")
	assetID := e.Request.PathValue("asset")
	if pluginID == "" || assetID == "" {
		return apis.NewBadRequestError("plugin and asset are required", nil)
	}
	entry, err := fetchPluginAssetThumbnailEntryForUser(e.Request.Context(), e.App, e.Auth.Id, "", pluginID, assetID)
	if err != nil {
		return err
	}
	return writePluginAssetThumbnail(e, entry)
}

func fetchPluginAssetThumbnailEntryForUser(ctx context.Context, app core.App, userID string, actorID string, pluginID string, assetID string) (pluginAssetThumbnailCacheEntry, error) {
	plugin, capability, instance, auth, config, err := assetPluginInvocationForUser(app, userID, pluginID)
	if err != nil {
		return pluginAssetThumbnailCacheEntry{}, err
	}
	cacheKey := pluginAssetThumbnailCacheKey(userID, pluginID, instance.Id, assetID)
	if entry, ok := getPluginAssetThumbnailCache(cacheKey); ok {
		return entry, nil
	}

	call, owner := beginPluginAssetThumbnailFetch(cacheKey)
	if !owner {
		select {
		case <-ctx.Done():
			return pluginAssetThumbnailCacheEntry{}, ctx.Err()
		case <-call.done:
			return call.entry, call.err
		}
	}

	var entry pluginAssetThumbnailCacheEntry
	defer func() {
		if recovered := recover(); recovered != nil {
			finishPluginAssetThumbnailFetch(cacheKey, call, entry, fmt.Errorf("thumbnail fetch panicked: %v", recovered))
			panic(recovered)
		}
		finishPluginAssetThumbnailFetch(cacheKey, call, entry, err)
	}()
	entry, err = fetchPluginAssetThumbnailEntryUncached(ctx, cacheKey, plugin, capability, instance, auth, config, userID, actorID, assetID)
	return entry, err
}

func fetchPluginAssetThumbnailEntryUncached(ctx context.Context, cacheKey string, plugin pluginsystem.LocalPlugin, capability pluginsystem.CapabilityManifest, instance *core.Record, auth map[string]any, config map[string]any, userID string, actorID string, assetID string) (pluginAssetThumbnailCacheEntry, error) {
	output, err := callAssetPlugin(ctx, plugin, capability, instance, auth, config, pluginAssetLibraryActionInput{
		Action:   "thumbnail",
		AssetIDs: []string{assetID},
	})
	if err != nil {
		return pluginAssetThumbnailCacheEntry{}, err
	}
	if output.Error != nil {
		return pluginAssetThumbnailCacheEntry{}, apis.NewBadRequestError(output.Error.Message, output.Error)
	}
	if len(output.Photos) == 0 {
		return pluginAssetThumbnailCacheEntry{}, apis.NewNotFoundError("thumbnail not found", nil)
	}
	fetched, err := importer.FetchPhotoMedia(ctx, output.Photos[0], importer.Options{
		UserID:   userID,
		ActorID:  actorID,
		Manifest: plugin.Manifest,
		Policy:   pluginhost.InstancePolicy(plugin, config).WithHostAuth(auth),
		Auth:     auth,
	}, pluginAssetThumbnailMaxBytes)
	if err != nil {
		return pluginAssetThumbnailCacheEntry{}, apis.NewNotFoundError("thumbnail not available", err)
	}
	contentType := fetched.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	entry := pluginAssetThumbnailCacheEntry{
		ContentType: contentType,
		Body:        fetched.Body,
		ETag:        pluginAssetThumbnailETag(fetched.Body),
		ExpiresAt:   time.Now().Add(pluginAssetThumbnailCacheTTL),
		LastAccess:  time.Now(),
	}
	putPluginAssetThumbnailCache(cacheKey, entry)
	return entry, nil
}

func beginPluginAssetThumbnailFetch(cacheKey string) (*pluginAssetThumbnailFetchCall, bool) {
	pluginAssetThumbnailFetches.Lock()
	defer pluginAssetThumbnailFetches.Unlock()

	if call, ok := pluginAssetThumbnailFetches.calls[cacheKey]; ok {
		return call, false
	}
	call := &pluginAssetThumbnailFetchCall{done: make(chan struct{})}
	pluginAssetThumbnailFetches.calls[cacheKey] = call
	return call, true
}

func finishPluginAssetThumbnailFetch(cacheKey string, call *pluginAssetThumbnailFetchCall, entry pluginAssetThumbnailCacheEntry, err error) {
	pluginAssetThumbnailFetches.Lock()
	call.entry = entry
	call.err = err
	delete(pluginAssetThumbnailFetches.calls, cacheKey)
	close(call.done)
	pluginAssetThumbnailFetches.Unlock()
}

func fetchAssetRecordPluginThumbnailEntry(ctx context.Context, app core.App, asset *core.Record) (pluginAssetThumbnailCacheEntry, error) {
	remote, err := assetservice.RemotePhotoAssetFromRecord(asset)
	if err != nil {
		return pluginAssetThumbnailCacheEntry{}, err
	}
	pluginID := strings.TrimSpace(remote.PluginID)
	if pluginID == "" {
		pluginID = strings.TrimSpace(asset.GetString("external_provider"))
	}
	if pluginID == "" {
		return pluginAssetThumbnailCacheEntry{}, fmt.Errorf("remote asset has no plugin id")
	}
	assetID := strings.TrimSpace(asset.GetString("external_id"))
	if assetID == "" {
		return pluginAssetThumbnailCacheEntry{}, fmt.Errorf("remote asset has no external id")
	}
	userID, err := util.AssetAuthorUserID(app, asset)
	if err != nil {
		return pluginAssetThumbnailCacheEntry{}, err
	}
	return fetchPluginAssetThumbnailEntryForUser(ctx, app, userID, asset.GetString("author"), pluginID, assetID)
}

func AssetLibraryCandidates(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}

	actor, err := e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
	if err != nil {
		return apis.NewUnauthorizedError("actor not found", err)
	}

	var data pluginAssetLibraryRequest
	if err := e.BindBody(&data); err != nil {
		return apis.NewBadRequestError("Failed to read request data", err)
	}

	if data.TrailID != "" {
		if err := ensureCanEditTrailTarget(e.App, e.Auth.Id, data.TrailID); err != nil {
			return err
		}
	}
	if data.WaypointID != "" {
		if err := ensureCanEditWaypointTarget(e.App, e.Auth.Id, data.WaypointID, data.TrailID); err != nil {
			return err
		}
	}
	if data.SummitLogID != "" {
		if err := ensureCanEditSummitLogTarget(e.App, e.Auth.Id, data.SummitLogID, data.TrailID); err != nil {
			return err
		}
	}

	request, err := assetLibraryActionInputForWandererLibrary(e.App, data)
	if err != nil {
		return apis.NewBadRequestError("invalid asset library request", err)
	}

	linkedAssetIDs, err := assetLibraryLinkedAssetIDs(e.App, data)
	if err != nil {
		return err
	}

	hasLocation := data.latSet && data.lonSet
	pagination := assetLibraryPaginationForRequest(data, request, hasLocation)
	records, hasMore, err := assetLibraryRecords(e.App, actor.Id, request, hasLocation, pagination)
	if err != nil {
		return err
	}

	maxDistance := assetLibraryMaxDistance(request, hasLocation)
	candidatePool := make([]pluginAssetCandidate, 0, len(records))
	for _, record := range records {
		candidate, ok := assetLibraryCandidate(record, request, hasLocation)
		if !ok || candidate.Distance > maxDistance {
			continue
		}
		candidatePool = append(candidatePool, candidate)
	}

	existingExternalRefs, err := assetLibraryExistingExternalRefs(e.App, actor.Id)
	if err != nil {
		return err
	}
	candidates := make([]pluginAssetCandidate, 0, len(candidatePool))
	for _, candidate := range candidatePool {
		if !linkedAssetIDs[candidate.AssetID] {
			candidates = append(candidates, candidate)
		}
	}
	sortAssetLibraryCandidates(candidates, len(request.Points) > 0)

	hasTimestamps := false
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.TakenAt) != "" {
			hasTimestamps = true
			break
		}
	}

	return e.JSON(http.StatusOK, assetLibraryResponse{
		HasTimestamps:        hasTimestamps,
		Candidates:           candidates,
		ExistingExternalRefs: existingExternalRefs,
		HasMore:              hasMore,
		TakenAfter:           "",
	})
}

func pluginAssetThumbnailCacheKey(userID string, pluginID string, instanceID string, assetID string) string {
	return strings.Join([]string{userID, pluginID, instanceID, assetID}, "\x00")
}

func getPluginAssetThumbnailCache(key string) (pluginAssetThumbnailCacheEntry, bool) {
	pluginAssetThumbnailCache.Lock()
	defer pluginAssetThumbnailCache.Unlock()

	entry, ok := pluginAssetThumbnailCache.items[key]
	if !ok {
		return pluginAssetThumbnailCacheEntry{}, false
	}
	if time.Now().After(entry.ExpiresAt) {
		delete(pluginAssetThumbnailCache.items, key)
		pluginAssetThumbnailCache.totalBytes -= int64(len(entry.Body))
		return pluginAssetThumbnailCacheEntry{}, false
	}
	entry.LastAccess = time.Now()
	pluginAssetThumbnailCache.items[key] = entry
	return entry, true
}

func putPluginAssetThumbnailCache(key string, entry pluginAssetThumbnailCacheEntry) {
	if int64(len(entry.Body)) > pluginAssetThumbnailCacheMaxBytes {
		return
	}

	pluginAssetThumbnailCache.Lock()
	defer pluginAssetThumbnailCache.Unlock()

	if existing, ok := pluginAssetThumbnailCache.items[key]; ok {
		pluginAssetThumbnailCache.totalBytes -= int64(len(existing.Body))
	}
	pluginAssetThumbnailCache.items[key] = entry
	pluginAssetThumbnailCache.totalBytes += int64(len(entry.Body))

	for len(pluginAssetThumbnailCache.items) > pluginAssetThumbnailCacheMaxEntries ||
		pluginAssetThumbnailCache.totalBytes > pluginAssetThumbnailCacheMaxBytes {
		oldestKey := ""
		var oldestAccess time.Time
		for key, entry := range pluginAssetThumbnailCache.items {
			if oldestKey == "" || entry.LastAccess.Before(oldestAccess) {
				oldestKey = key
				oldestAccess = entry.LastAccess
			}
		}
		if oldestKey == "" {
			break
		}
		oldest := pluginAssetThumbnailCache.items[oldestKey]
		delete(pluginAssetThumbnailCache.items, oldestKey)
		pluginAssetThumbnailCache.totalBytes -= int64(len(oldest.Body))
	}
}

func writePluginAssetThumbnail(e *core.RequestEvent, entry pluginAssetThumbnailCacheEntry) error {
	e.Response.Header().Set("Cache-Control", "private, no-cache")
	e.Response.Header().Set("ETag", entry.ETag)
	e.Response.Header().Set("Vary", "Authorization")
	if e.Request.Header.Get("If-None-Match") == entry.ETag {
		return e.NoContent(http.StatusNotModified)
	}
	return e.Blob(http.StatusOK, entry.ContentType, entry.Body)
}

func pluginAssetThumbnailETag(body []byte) string {
	sum := sha256.Sum256(body)
	return fmt.Sprintf("%q", fmt.Sprintf("plugin-thumb-%x", sum[:]))
}

func pluginSystemAssetCall(e *core.RequestEvent, action string) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}

	var data pluginAssetLibraryRequest
	if err := e.BindBody(&data); err != nil {
		return apis.NewBadRequestError("Failed to read request data", err)
	}
	data.PluginID = e.Request.PathValue("plugin")
	if data.PluginID == "" {
		return apis.NewBadRequestError("plugin is required", nil)
	}
	data.Action = action
	if action == "import-to-waypoint" || action == "import-to-target" {
		data.Action = "import"
	}
	if data.Action == "import" {
		normalizedAssetIDs, err := normalizeAssetPluginImportIDs(data.AssetIDs)
		if err != nil {
			return err
		}
		data.AssetIDs = normalizedAssetIDs
	}

	if data.TrailID != "" {
		if err := ensureOwnsTrail(e.App, e.Auth.Id, data.TrailID); err != nil {
			return err
		}
	}
	if action == "import-to-waypoint" {
		if data.TrailID == "" || data.WaypointID == "" || len(data.AssetIDs) == 0 {
			return apis.NewBadRequestError("trailId, waypointId, and assetIds are required", nil)
		}
		if err := ensureOwnsWaypoint(e.App, e.Auth.Id, data.WaypointID, data.TrailID); err != nil {
			return err
		}
	}
	if action == "import-to-target" {
		if data.TrailID == "" || len(data.AssetIDs) == 0 {
			return apis.NewBadRequestError("trailId and assetIds are required", nil)
		}
		targetCount := 0
		if data.WaypointID != "" {
			targetCount++
			if err := ensureOwnsWaypoint(e.App, e.Auth.Id, data.WaypointID, data.TrailID); err != nil {
				return err
			}
		}
		if data.SummitLogID != "" {
			targetCount++
			if err := ensureOwnsSummitLog(e.App, e.Auth.Id, data.SummitLogID, data.TrailID); err != nil {
				return err
			}
		}
		if targetCount > 1 {
			return apis.NewBadRequestError("Only one waypointId or summitLogId target is allowed", nil)
		}
	}

	plugin, capability, instance, auth, config, err := assetPluginInvocation(e, data.PluginID)
	if err != nil {
		return err
	}
	request, err := assetLibraryActionInput(e, data)
	if err != nil {
		return err
	}
	if action == "candidates" {
		return callAssetPluginCandidates(e, plugin, capability, instance, auth, config, request, data.CursorID)
	}
	output, err := callAssetPlugin(e.Request.Context(), plugin, capability, instance, auth, config, request)
	if err != nil {
		return err
	}
	if output.Error != nil {
		return apis.NewBadRequestError(output.Error.Message, output.Error)
	}
	if data.Action == "import" {
		if err := validateAssetPluginImportPartition(data.AssetIDs, output); err != nil {
			return err
		}
	}

	switch action {
	case "import":
		return importAssetPluginPhotos(e, plugin, instance, auth, config, output, data, "")
	case "import-to-waypoint":
		return importAssetPluginPhotos(e, plugin, instance, auth, config, output, data, data.WaypointID)
	case "import-to-target":
		return importAssetPluginPhotosToTarget(e, plugin, auth, config, output, data)
	default:
		return e.JSON(http.StatusOK, output)
	}
}

func callAssetPluginCandidates(e *core.RequestEvent, plugin pluginsystem.LocalPlugin, capability pluginsystem.CapabilityManifest, instance *core.Record, auth map[string]any, config map[string]any, request pluginAssetLibraryActionInput, cursorID string) error {
	actorID, err := util.ResolveAssetAuthor(e.App, e.Auth.Id)
	if err != nil {
		return err
	}
	binding, err := newPluginAssetCursorBinding(e.Auth.Id, actorID, plugin.Manifest.ID, instance.Id, request, auth, config)
	if err != nil {
		return err
	}

	cursorID = strings.TrimSpace(cursorID)
	state := map[string]any(nil)
	inputStateHash := ""
	seen := map[string]bool{}
	if cursorID != "" {
		entry, restartRequired, err := assetPluginCursors.resume(cursorID, binding)
		if err != nil {
			return err
		}
		if restartRequired {
			return e.JSON(http.StatusOK, pluginAssetCandidatesResponse{Candidates: []pluginAssetCandidate{}, RestartRequired: true})
		}
		state = entry.State
		inputStateHash = entry.StateHash
		seen = entry.Seen
	}

	output, err := callAssetPlugin(e.Request.Context(), plugin, capability, instance, auth, config, request, pluginAssetCallInput{
		Search: pluginAssetPickerSearchLimits,
		State:  state,
	})
	if err != nil {
		return err
	}
	if output.Error != nil {
		return apis.NewBadRequestError(output.Error.Message, output.Error)
	}
	if err := validateAssetPluginCandidateBatch(output, pluginAssetPickerSearchLimits); err != nil {
		return err
	}

	outputStateHash, _, err := validatePluginAssetContinuation(state, output.State, output.HasMore, seen)
	if err != nil {
		return err
	}
	responseCursorID := ""
	if cursorID == "" && output.HasMore {
		responseCursorID, err = assetPluginCursors.create(binding, output.State, outputStateHash)
		if err != nil {
			return err
		}
	} else if cursorID != "" {
		if output.HasMore {
			seen[outputStateHash] = true
		}
		restartRequired, err := assetPluginCursors.advance(cursorID, binding, inputStateHash, output.State, outputStateHash, seen, output.HasMore)
		if err != nil {
			return err
		}
		if restartRequired {
			return e.JSON(http.StatusOK, pluginAssetCandidatesResponse{Candidates: []pluginAssetCandidate{}, RestartRequired: true})
		}
		if output.HasMore {
			responseCursorID = cursorID
		}
	}

	return e.JSON(http.StatusOK, pluginAssetCandidatesResponse{
		Candidates:    append([]pluginAssetCandidate{}, output.Candidates...),
		HasMore:       output.HasMore,
		TakenAfter:    output.TakenAfter,
		HasTimestamps: output.HasTimestamps,
		CursorID:      responseCursorID,
	})
}

func stableUniqueAssetIDs(values []string) []string {
	unique := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		unique = append(unique, value)
	}
	return unique
}

func normalizeAssetPluginImportIDs(values []string) ([]string, error) {
	unique := stableUniqueAssetIDs(values)
	if len(unique) > pluginAssetImportMaxIDs {
		return nil, apis.NewBadRequestError(fmt.Sprintf("assetIds must contain at most %d unique IDs", pluginAssetImportMaxIDs), nil)
	}
	return unique, nil
}

func validateAssetPluginImportPartition(requested []string, output pluginAssetLibraryOutput) error {
	requested = stableUniqueAssetIDs(requested)
	requestedSet := make(map[string]bool, len(requested))
	for _, assetID := range requested {
		requestedSet[assetID] = true
	}
	accounted := make(map[string]string, len(requested))
	for _, photo := range output.Photos {
		assetID := strings.TrimSpace(photo.ExternalID)
		if !requestedSet[assetID] {
			return fmt.Errorf("asset plugin import returned unrequested photo %q", assetID)
		}
		if previous := accounted[assetID]; previous != "" {
			return fmt.Errorf("asset plugin import returned duplicate asset %q in %s and photos", assetID, previous)
		}
		accounted[assetID] = "photos"
	}
	for _, omitted := range output.OmittedAssets {
		assetID := strings.TrimSpace(omitted.AssetID)
		if strings.TrimSpace(omitted.Reason) == "" {
			return fmt.Errorf("asset plugin import omitted asset %q without a reason", assetID)
		}
		if !requestedSet[assetID] {
			return fmt.Errorf("asset plugin import omitted unrequested asset %q", assetID)
		}
		if previous := accounted[assetID]; previous != "" {
			return fmt.Errorf("asset plugin import returned duplicate asset %q in %s and omittedAssetIds", assetID, previous)
		}
		accounted[assetID] = "omittedAssetIds"
	}
	for _, assetID := range requested {
		if accounted[assetID] == "" {
			return fmt.Errorf("asset plugin import did not account for requested asset %q", assetID)
		}
	}
	return nil
}

func AutoAttachAssetPluginsForTrail(ctx context.Context, app core.App, userID string, trailID string, provider string) error {
	_, err := autoAttachAssetPluginsForTrail(ctx, app, userID, trailID, provider)
	return err
}

func autoAttachAssetPluginsForTrail(ctx context.Context, app core.App, userID string, trailID string, provider string) (pluginAssetAutoAttachSummary, error) {
	summary := pluginAssetAutoAttachSummary{
		OK:      true,
		TrailID: strings.TrimSpace(trailID),
		Plugins: []pluginAssetAutoAttachPluginResult{},
	}
	userID = strings.TrimSpace(userID)
	trailID = strings.TrimSpace(trailID)
	provider = strings.TrimSpace(provider)
	if userID == "" || trailID == "" || provider == "" {
		return summary, nil
	}

	trail, err := app.FindRecordById("trails", trailID)
	if err != nil {
		return summary, err
	}
	if !assetPluginAutoAttachAllowedForTrail(provider, trail.GetBool("completed")) {
		return summary, nil
	}

	instances, err := app.FindRecordsByFilter(
		"plugin_instances",
		"user={:user} && enabled=true",
		"",
		-1,
		0,
		dbx.Params{"user": userID},
	)
	if err != nil {
		return summary, err
	}

	for _, instance := range instances {
		if err := ctx.Err(); err != nil {
			return summary, err
		}
		pluginID := instance.GetString("plugin_id")
		plugin, err := localPlugin(app, pluginID)
		if err != nil {
			app.Logger().Warn("skipping asset auto attach for unknown plugin", "plugin", pluginID, "instance", instance.Id, "error", err)
			continue
		}
		if plugin.Manifest.Type != pluginsystem.PluginTypeAssets {
			continue
		}
		capability, err := pluginCapability(plugin, "asset_library", "v1")
		if err != nil {
			app.Logger().Warn("skipping asset auto attach for plugin without asset_library.v1", "plugin", plugin.Manifest.ID, "instance", instance.Id, "error", err)
			continue
		}
		auth, err := decryptedInstanceAuth(instance)
		if err != nil {
			app.Logger().Warn("skipping asset auto attach with invalid auth", "plugin", plugin.Manifest.ID, "instance", instance.Id, "error", err)
			continue
		}
		config := pluginhost.EffectiveConfig(app, plugin.Manifest.ID, instance)
		config = assetPluginConnectorConfig(plugin, config)
		if !assetPluginProviderEnabled(pluginhost.HostConfig(config), provider) {
			continue
		}
		result := pluginAssetAutoAttachPluginResult{
			PluginID:   plugin.Manifest.ID,
			InstanceID: instance.Id,
			Omitted:    []pluginAssetOmission{},
		}
		outcome, err := autoAttachAssetPluginForTrail(ctx, app, userID, trailID, plugin, capability, instance, auth, config)
		if err != nil {
			result.Error = err.Error()
			summary.Plugins = append(summary.Plugins, result)
			app.Logger().Warn("asset plugin auto attach failed for plugin", "plugin", plugin.Manifest.ID, "instance", instance.Id, "provider", provider, "trail", trailID, "error", err)
			continue
		}
		result.Imported = outcome.Imported
		result.Omitted = append([]pluginAssetOmission{}, outcome.Omitted...)
		summary.Imported += outcome.Imported
		summary.Omitted += len(outcome.Omitted)
		summary.Plugins = append(summary.Plugins, result)
		if outcome.Imported > 0 || len(outcome.Omitted) > 0 {
			app.Logger().Info("asset plugin auto attach completed", "plugin", plugin.Manifest.ID, "instance", instance.Id, "provider", provider, "trail", trailID, "imported", outcome.Imported, "omitted", len(outcome.Omitted))
		}
	}
	return summary, nil
}

func enabledAssetPluginCount(app core.App, userID string) (int, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return 0, nil
	}
	instances, err := app.FindRecordsByFilter(
		"plugin_instances",
		"user={:user} && enabled=true",
		"",
		-1,
		0,
		dbx.Params{"user": userID},
	)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, instance := range instances {
		plugin, err := localPlugin(app, instance.GetString("plugin_id"))
		if err != nil {
			app.Logger().Warn("skipping unavailable asset plugin during maintenance scan", "plugin", instance.GetString("plugin_id"), "instance", instance.Id, "error", err)
			continue
		}
		if plugin.Manifest.Type != pluginsystem.PluginTypeAssets {
			continue
		}
		if _, err := pluginCapability(plugin, "asset_library", "v1"); err != nil {
			app.Logger().Warn("skipping asset plugin without asset_library.v1 during maintenance scan", "plugin", plugin.Manifest.ID, "instance", instance.Id, "error", err)
			continue
		}
		count++
	}
	return count, nil
}

func pluginAssetMaintenanceTrailCandidate(trail *core.Record, thumbnail string) pluginAssetTrailPhotoMaintenanceCandidate {
	return pluginAssetTrailPhotoMaintenanceCandidate{
		ID:            trail.Id,
		Name:          trail.GetString("name"),
		Location:      trail.GetString("location"),
		Date:          util.RecordDateTimeRFC3339(trail, "date"),
		Created:       util.RecordDateTimeRFC3339(trail, "created"),
		Updated:       util.RecordDateTimeRFC3339(trail, "updated"),
		Completed:     trail.GetBool("completed"),
		Public:        trail.GetBool("public"),
		Distance:      trail.GetFloat("distance"),
		ElevationGain: trail.GetFloat("elevation_gain"),
		ElevationLoss: trail.GetFloat("elevation_loss"),
		Duration:      trail.GetFloat("duration"),
		Difficulty:    trail.GetString("difficulty"),
		Thumbnail:     thumbnail,
	}
}

func trailAssetMaintenanceState(app core.App, trails []*core.Record) (map[string]bool, map[string]string, error) {
	visiblePhotos := make(map[string]bool, len(trails))
	thumbnails := make(map[string]string, len(trails))
	if len(trails) == 0 {
		return visiblePhotos, thumbnails, nil
	}

	trailIDs := make([]any, 0, len(trails))
	for _, trail := range trails {
		trailIDs = append(trailIDs, trail.Id)
	}

	links := []*core.Record{}
	err := app.RecordQuery("trail_assets").
		AndWhere(dbx.In("trail", trailIDs...)).
		OrderBy("created ASC").
		All(&links)
	if err != nil {
		return nil, nil, err
	}

	if errs := app.ExpandRecords(links, []string{"asset"}, nil); len(errs) > 0 {
		return nil, nil, fmt.Errorf("failed to expand trail asset links: %v", errs)
	}

	for _, link := range links {
		trailID := link.GetString("trail")
		asset := link.ExpandedOne("asset")
		if asset == nil || asset.GetString("type") != "photo" {
			continue
		}
		if util.IsGeneratedRoutePreviewAsset(asset) {
			if thumbnails[trailID] == "" {
				thumbnails[trailID] = util.AssetPublicMediaURL(asset, "")
			}
			continue
		}
		visiblePhotos[trailID] = true
	}

	return visiblePhotos, thumbnails, nil
}

func autoAttachAssetPluginForTrail(ctx context.Context, app core.App, userID string, trailID string, plugin pluginsystem.LocalPlugin, capability pluginsystem.CapabilityManifest, instance *core.Record, auth map[string]any, config map[string]any) (pluginAssetAutoAttachOutcome, error) {
	candidatesRequest, err := assetLibraryActionInputForApp(app, pluginAssetLibraryRequest{
		Action:  "candidates",
		TrailID: trailID,
	}, true)
	if err != nil {
		return pluginAssetAutoAttachOutcome{}, err
	}
	if !assetPluginAutoAttachHasTimeWindow(candidatesRequest) {
		return pluginAssetAutoAttachOutcome{}, nil
	}

	allCandidates := make([]pluginAssetCandidate, 0)
	seenAssetIDs := map[string]bool{}
	seenStates := map[string]bool{}
	var state map[string]any
	for batch := 0; batch < defaultAssetAutoAttachMaxBatches; batch++ {
		candidatesOutput, err := callAssetPlugin(ctx, plugin, capability, instance, auth, config, candidatesRequest, pluginAssetCallInput{
			Search: pluginAssetAutoAttachSearchLimits,
			State:  state,
		})
		if err != nil {
			return pluginAssetAutoAttachOutcome{}, err
		}
		if candidatesOutput.Error != nil {
			return pluginAssetAutoAttachOutcome{}, fmt.Errorf("%s: %s", candidatesOutput.Error.Code, candidatesOutput.Error.Message)
		}
		if err := validateAssetPluginCandidateBatch(candidatesOutput, pluginAssetAutoAttachSearchLimits); err != nil {
			return pluginAssetAutoAttachOutcome{}, err
		}
		stateHash, _, err := validatePluginAssetContinuation(state, candidatesOutput.State, candidatesOutput.HasMore, seenStates)
		if err != nil {
			return pluginAssetAutoAttachOutcome{}, err
		}
		for _, candidate := range candidatesOutput.Candidates {
			assetID := strings.TrimSpace(candidate.AssetID)
			if assetID == "" || seenAssetIDs[assetID] {
				continue
			}
			candidate.AssetID = assetID
			seenAssetIDs[assetID] = true
			allCandidates = append(allCandidates, candidate)
		}
		if !candidatesOutput.HasMore {
			break
		}
		if batch+1 >= defaultAssetAutoAttachMaxBatches {
			return pluginAssetAutoAttachOutcome{}, fmt.Errorf("asset plugin candidates exceeded %d batches", defaultAssetAutoAttachMaxBatches)
		}
		seenStates[stateHash] = true
		state = candidatesOutput.State
	}

	sortAssetPluginAutoAttachCandidates(allCandidates)
	assetIDs := make([]string, 0, len(allCandidates))
	for _, candidate := range allCandidates {
		assetIDs = append(assetIDs, candidate.AssetID)
	}
	if len(assetIDs) == 0 {
		return pluginAssetAutoAttachOutcome{}, nil
	}
	existing, err := existingTrailAssetExternalIDs(app, userID, plugin.Manifest.ID, trailID, assetIDs)
	if err != nil {
		return pluginAssetAutoAttachOutcome{}, err
	}
	filteredAssetIDs := make([]string, 0, len(assetIDs))
	for _, assetID := range assetIDs {
		if !existing[assetID] {
			filteredAssetIDs = append(filteredAssetIDs, assetID)
		}
	}
	if len(filteredAssetIDs) == 0 {
		return pluginAssetAutoAttachOutcome{}, nil
	}

	photos := make([]pluginsystem.Photo, 0, len(filteredAssetIDs))
	omitted := make([]pluginAssetOmission, 0)
	for start := 0; start < len(filteredAssetIDs); start += pluginAssetImportMaxIDs {
		end := start + pluginAssetImportMaxIDs
		if end > len(filteredAssetIDs) {
			end = len(filteredAssetIDs)
		}
		block := filteredAssetIDs[start:end]
		importRequest := candidatesRequest
		importRequest.Action = "import"
		importRequest.AssetIDs = block
		importOutput, err := callAssetPlugin(ctx, plugin, capability, instance, auth, config, importRequest)
		if err != nil {
			return pluginAssetAutoAttachOutcome{}, err
		}
		if importOutput.Error != nil {
			return pluginAssetAutoAttachOutcome{}, fmt.Errorf("%s: %s", importOutput.Error.Code, importOutput.Error.Message)
		}
		if err := validateAssetPluginImportPartition(block, importOutput); err != nil {
			return pluginAssetAutoAttachOutcome{}, err
		}
		photos = append(photos, importOutput.Photos...)
		omitted = append(omitted, importOutput.OmittedAssets...)
	}

	rank := make(map[string]int, len(filteredAssetIDs))
	for index, assetID := range filteredAssetIDs {
		rank[assetID] = index
	}
	sort.SliceStable(photos, func(i, j int) bool {
		return rank[strings.TrimSpace(photos[i].ExternalID)] < rank[strings.TrimSpace(photos[j].ExternalID)]
	})
	results, err := importAssetPluginPhotosForTrail(ctx, app, userID, plugin, auth, config, pluginAssetLibraryOutput{Photos: photos}, pluginAssetLibraryRequest{
		PluginID: plugin.Manifest.ID,
		Action:   "import",
		TrailID:  trailID,
		AssetIDs: filteredAssetIDs,
	}, "", true)
	if err != nil {
		return pluginAssetAutoAttachOutcome{}, err
	}
	return pluginAssetAutoAttachOutcome{Imported: len(results), Omitted: omitted}, nil
}

func validateAssetPluginCandidateBatch(output pluginAssetLibraryOutput, limits pluginAssetSearchLimits) error {
	if limits.MaxItems > 0 && len(output.Candidates) > limits.MaxItems {
		return fmt.Errorf("asset plugin returned %d candidates, exceeding maxItems %d", len(output.Candidates), limits.MaxItems)
	}
	return nil
}

func sortAssetPluginAutoAttachCandidates(candidates []pluginAssetCandidate) {
	sort.Slice(candidates, func(i, j int) bool {
		a := candidates[i]
		b := candidates[j]
		if a.DistanceFromStart != b.DistanceFromStart {
			return a.DistanceFromStart < b.DistanceFromStart
		}
		if a.Distance != b.Distance {
			return a.Distance < b.Distance
		}
		aTime, aValid := parseAssetPluginCandidateTime(a.TakenAt)
		bTime, bValid := parseAssetPluginCandidateTime(b.TakenAt)
		if aValid != bValid {
			return aValid
		}
		if aValid && !aTime.Equal(bTime) {
			return aTime.Before(bTime)
		}
		return a.AssetID < b.AssetID
	})
}

func parseAssetPluginCandidateTime(value string) (time.Time, bool) {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	return parsed, err == nil
}

func assetPluginAutoAttachAllowedForTrail(provider string, completed bool) bool {
	if strings.TrimSpace(provider) == "upload" {
		return true
	}
	return completed
}

func assetPluginAutoAttachHasTimeWindow(request pluginAssetLibraryActionInput) bool {
	return strings.TrimSpace(request.StartedAt) != "" && strings.TrimSpace(request.EndedAt) != ""
}

func assetPluginInvocation(e *core.RequestEvent, pluginID string) (pluginsystem.LocalPlugin, pluginsystem.CapabilityManifest, *core.Record, map[string]any, map[string]any, error) {
	return assetPluginInvocationForUser(e.App, e.Auth.Id, pluginID)
}

func assetPluginInvocationForUser(app core.App, userID string, pluginID string) (pluginsystem.LocalPlugin, pluginsystem.CapabilityManifest, *core.Record, map[string]any, map[string]any, error) {
	instance, err := app.FindFirstRecordByFilter(
		"plugin_instances",
		"user={:user} && plugin_id={:plugin_id} && enabled=true",
		dbx.Params{"user": userID, "plugin_id": pluginID},
	)
	if err != nil {
		return pluginsystem.LocalPlugin{}, pluginsystem.CapabilityManifest{}, nil, nil, nil, apis.NewBadRequestError("no enabled asset plugin instance configured for this plugin", nil)
	}
	plugin, capability, err := localPluginCapability(app, pluginID, "asset_library", "v1")
	if err != nil {
		return pluginsystem.LocalPlugin{}, pluginsystem.CapabilityManifest{}, nil, nil, nil, err
	}
	if plugin.Manifest.Type != pluginsystem.PluginTypeAssets {
		return pluginsystem.LocalPlugin{}, pluginsystem.CapabilityManifest{}, nil, nil, nil, apis.NewBadRequestError("plugin is not an asset plugin", nil)
	}
	auth, err := decryptedInstanceAuth(instance)
	if err != nil {
		return pluginsystem.LocalPlugin{}, pluginsystem.CapabilityManifest{}, nil, nil, nil, err
	}
	config := pluginhost.EffectiveConfig(app, plugin.Manifest.ID, instance)
	config = assetPluginConnectorConfig(plugin, config)
	return plugin, capability, instance, auth, config, nil
}

func assetPluginDraftInvocation(e *core.RequestEvent, pluginID string, submittedAuth map[string]any, submittedConfig map[string]any) (pluginsystem.LocalPlugin, pluginsystem.CapabilityManifest, *core.Record, map[string]any, map[string]any, error) {
	plugin, capability, err := localPluginCapability(e.App, pluginID, "asset_library", "v1")
	if err != nil {
		return pluginsystem.LocalPlugin{}, pluginsystem.CapabilityManifest{}, nil, nil, nil, err
	}
	if plugin.Manifest.Type != pluginsystem.PluginTypeAssets {
		return pluginsystem.LocalPlugin{}, pluginsystem.CapabilityManifest{}, nil, nil, nil, apis.NewBadRequestError("plugin is not an asset plugin", nil)
	}

	instance, err := e.App.FindFirstRecordByFilter(
		"plugin_instances",
		"user={:user} && plugin_id={:plugin_id}",
		dbx.Params{"user": e.Auth.Id, "plugin_id": pluginID},
	)
	if err != nil {
		collection, collectionErr := e.App.FindCollectionByNameOrId("plugin_instances")
		if collectionErr != nil {
			return pluginsystem.LocalPlugin{}, pluginsystem.CapabilityManifest{}, nil, nil, nil, collectionErr
		}
		instance = core.NewRecord(collection)
		instance.Set("user", e.Auth.Id)
		instance.Set("plugin_id", pluginID)
	}

	auth := map[string]any{}
	if instance.Id != "" {
		auth, err = decryptedInstanceAuth(instance)
		if err != nil {
			return pluginsystem.LocalPlugin{}, pluginsystem.CapabilityManifest{}, nil, nil, nil, err
		}
	}
	mergeSubmittedPluginAuth(auth, submittedAuth)

	config := pluginhost.EffectiveConfig(e.App, plugin.Manifest.ID, instance)
	if submittedConfig != nil {
		mergeAssetPluginDraftConfig(config, submittedConfig)
	}
	config = assetPluginConnectorConfig(plugin, config)

	return plugin, capability, instance, auth, config, nil
}

func mergeAssetPluginDraftConfig(config map[string]any, submitted map[string]any) {
	pluginsystem.DeepMergeConfig(config, pluginhost.InstanceConfigOverrides(submitted))
}

func mergeSubmittedPluginAuth(auth map[string]any, submitted map[string]any) {
	for key, value := range submitted {
		if text, ok := value.(string); ok && strings.TrimSpace(text) == "" {
			continue
		}
		if value == nil {
			continue
		}
		auth[key] = value
	}
}

func assetPluginConnectorConfig(plugin pluginsystem.LocalPlugin, config map[string]any) map[string]any {
	return pluginhost.AssetConnectorConfig(plugin, config)
}

type pluginAssetCallInput struct {
	Search pluginAssetSearchLimits
	State  map[string]any
}

func callAssetPlugin(ctx context.Context, plugin pluginsystem.LocalPlugin, capability pluginsystem.CapabilityManifest, instance *core.Record, auth map[string]any, config map[string]any, request pluginAssetLibraryActionInput, callInputs ...pluginAssetCallInput) (pluginAssetLibraryOutput, error) {
	runtime, err := pluginsystem.NewRuntimeRegistry().RuntimeFor(plugin)
	if err != nil {
		return pluginAssetLibraryOutput{}, err
	}
	policy := pluginhost.InstancePolicy(plugin, config).WithHostAuth(auth)
	session, err := runtime.OpenSession(ctx, plugin, policy)
	if err != nil {
		return pluginAssetLibraryOutput{}, err
	}
	defer func() {
		_ = session.Close(context.Background())
	}()
	callInput := pluginAssetCallInput{}
	if len(callInputs) > 0 {
		callInput = callInputs[0]
	}
	var search *pluginAssetSearchLimits
	if callInput.Search != (pluginAssetSearchLimits{}) {
		searchValue := callInput.Search
		search = &searchValue
	}
	input := pluginAssetLibraryInput{
		Instance: pluginsystem.InstanceRef{ID: instance.Id, PluginID: instance.GetString("plugin_id")},
		Auth:     pluginsystem.PluginInputAuth(plugin, auth),
		Config:   pluginhost.RuntimeConfig(config),
		Limits:   pluginPhotoImportLimits(pluginhost.HostConfig(config)),
		Search:   search,
		State:    callInput.State,
		Request:  request,
	}
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return pluginAssetLibraryOutput{}, err
	}
	outputBytes, err := session.Call(ctx, capability.Export, inputBytes, assetPluginRuntimeCallOptions(request))
	if err != nil {
		return pluginAssetLibraryOutput{}, err
	}
	var output pluginAssetLibraryOutput
	if err := json.Unmarshal(outputBytes, &output); err != nil {
		return pluginAssetLibraryOutput{}, fmt.Errorf("plugin returned invalid %s output: %w", capability.Export, err)
	}
	return output, nil
}

func assetPluginRuntimeCallOptions(request pluginAssetLibraryActionInput) pluginsystem.RuntimeCallOptions {
	maxHostRequests := 24
	switch request.Action {
	case "check", "thumbnail":
		maxHostRequests = 4
	case "import":
		maxHostRequests = len(request.AssetIDs) + 8
	}
	return pluginsystem.RuntimeCallOptions{MaxHostRequests: maxHostRequests}
}

func assetLibraryActionInput(e *core.RequestEvent, data pluginAssetLibraryRequest) (pluginAssetLibraryActionInput, error) {
	return assetLibraryActionInputForApp(e.App, data, false)
}

func newAssetLibraryActionInput(data pluginAssetLibraryRequest) pluginAssetLibraryActionInput {
	return pluginAssetLibraryActionInput{
		Action:       data.Action,
		TrailID:      data.TrailID,
		Lat:          data.Lat,
		Lon:          data.Lon,
		TakenAfter:   strings.TrimSpace(data.TakenAfter),
		TakenBefore:  strings.TrimSpace(data.TakenBefore),
		DoubleRadius: data.DoubleRadius,
		AssetIDs:     data.AssetIDs,
	}
}

func assetLibraryActionInputForApp(app core.App, data pluginAssetLibraryRequest, useTrailTime bool) (pluginAssetLibraryActionInput, error) {
	request := newAssetLibraryActionInput(data)
	if err := applyAssetLibraryExplicitTimeWindow(&request); err != nil {
		return request, err
	}

	if strings.TrimSpace(data.TrailData) != "" {
		points, start, end, err := trailTrackPointsFromBytes([]byte(data.TrailData))
		if err != nil {
			if data.TrailID == "" {
				return request, err
			}
		} else {
			applyAssetLibraryTrackPoints(&request, points, start, end)
			if useTrailTime {
				applyAssetLibraryTrailTimeWindow(&request, start, end)
			}
			if assetPluginAutoAttachHasTimeWindow(request) || data.TrailID == "" {
				return request, nil
			}
		}
	}

	if data.TrailID == "" {
		return request, nil
	}
	trail, err := app.FindRecordById("trails", data.TrailID)
	if err != nil {
		return request, err
	}
	points, start, end, err := trailTrackPoints(app, trail)
	if err != nil {
		return request, err
	}
	applyAssetLibraryTrackPoints(&request, points, start, end)
	if useTrailTime {
		applyAssetLibraryTrailTimeWindow(&request, start, end)
	}
	return request, nil
}

func assetLibraryActionInputForWandererLibrary(app core.App, data pluginAssetLibraryRequest) (pluginAssetLibraryActionInput, error) {
	request := newAssetLibraryActionInput(data)
	if err := applyAssetLibraryExplicitTimeWindow(&request); err != nil {
		return request, err
	}

	if strings.TrimSpace(data.TrailData) != "" {
		points, start, end, err := trailTrackPointsFromBytes([]byte(data.TrailData))
		if err != nil {
			app.Logger().Warn("unable to parse trail data for asset library", "trail", data.TrailID, "error", err)
		} else {
			applyAssetLibraryTrackPoints(&request, points, start, end)
			return request, nil
		}
	}

	if data.TrailID == "" {
		return request, nil
	}
	trail, err := app.FindRecordById("trails", data.TrailID)
	if err != nil {
		return request, err
	}
	points, start, end, err := trailTrackPoints(app, trail)
	if err != nil {
		app.Logger().Warn("unable to read trail data for asset library", "trail", data.TrailID, "error", err)
		return request, nil
	}
	applyAssetLibraryTrackPoints(&request, points, start, end)
	return request, nil
}

func trailTrackPoints(app core.App, trail *core.Record) ([]pluginAssetTrackPoint, time.Time, time.Time, error) {
	content, err := util.ReadTrailGPX(app, trail)
	if err != nil {
		return nil, time.Time{}, time.Time{}, err
	}
	if len(content) == 0 {
		return nil, time.Time{}, time.Time{}, nil
	}
	return trailTrackPointsFromBytes(content)
}

func trailTrackPointsFromBytes(content []byte) ([]pluginAssetTrackPoint, time.Time, time.Time, error) {
	parsed, err := gpx.ParseBytes(content)
	if err != nil {
		return nil, time.Time{}, time.Time{}, err
	}
	points := make([]pluginAssetTrackPoint, 0)
	totalDistance := 0.0
	var previous *pluginAssetTrackPoint
	var startedAt time.Time
	var endedAt time.Time
	for _, track := range parsed.Tracks {
		for _, segment := range track.Segments {
			previous = nil
			for _, point := range segment.Points {
				current := pluginAssetTrackPoint{Lat: point.Latitude, Lon: point.Longitude, Distance: totalDistance}
				if !point.Timestamp.IsZero() {
					current.Timestamp = point.Timestamp.UTC().Format(time.RFC3339)
					if startedAt.IsZero() || point.Timestamp.Before(startedAt) {
						startedAt = point.Timestamp
					}
					if endedAt.IsZero() || point.Timestamp.After(endedAt) {
						endedAt = point.Timestamp
					}
				}
				if previous != nil {
					totalDistance += util.HaversineDistanceMeters(previous.Lat, previous.Lon, current.Lat, current.Lon)
					current.Distance = totalDistance
				}
				points = append(points, current)
				previous = &current
			}
		}
	}
	return decimateTrackPoints(points, 2000), startedAt, endedAt, nil
}

func applyAssetLibraryTrackPoints(request *pluginAssetLibraryActionInput, points []pluginAssetTrackPoint, start time.Time, end time.Time) {
	if len(points) > 0 {
		request.Points = points
	}
}

func applyAssetLibraryExplicitTimeWindow(request *pluginAssetLibraryActionInput) error {
	if request.TakenAfter == "" && request.TakenBefore == "" {
		return nil
	}
	if request.TakenAfter != "" {
		takenAfter, err := time.Parse(time.RFC3339, request.TakenAfter)
		if err != nil {
			return fmt.Errorf("invalid takenAfter: %w", err)
		}
		request.TakenAfter = takenAfter.UTC().Format(time.RFC3339)
		request.StartedAt = request.TakenAfter
	}
	if request.TakenBefore != "" {
		takenBefore, err := time.Parse(time.RFC3339, request.TakenBefore)
		if err != nil {
			return fmt.Errorf("invalid takenBefore: %w", err)
		}
		request.TakenBefore = takenBefore.UTC().Format(time.RFC3339)
		request.EndedAt = request.TakenBefore
	}
	return nil
}

func applyAssetLibraryTrailTimeWindow(request *pluginAssetLibraryActionInput, start time.Time, end time.Time) {
	if start.IsZero() || end.IsZero() {
		return
	}
	request.StartedAt = start.UTC().Format(time.RFC3339)
	request.EndedAt = end.UTC().Format(time.RFC3339)
}

func decimateTrackPoints(points []pluginAssetTrackPoint, limit int) []pluginAssetTrackPoint {
	if limit <= 0 || len(points) <= limit {
		return points
	}
	step := (len(points) + limit - 1) / limit
	decimated := make([]pluginAssetTrackPoint, 0, limit)
	for i := 0; i < len(points); i += step {
		decimated = append(decimated, points[i])
	}
	return decimated
}

func assetLibraryLinkedAssetIDs(app core.App, data pluginAssetLibraryRequest) (map[string]bool, error) {
	linked := map[string]bool{}
	targets := []struct {
		collection string
		field      string
		id         string
	}{
		{collection: "trail_assets", field: "trail", id: data.TrailID},
		{collection: "waypoint_assets", field: "waypoint", id: data.WaypointID},
		{collection: "summit_log_assets", field: "summit_log", id: data.SummitLogID},
	}

	for _, target := range targets {
		if target.id == "" {
			continue
		}
		assetIDs, err := util.AssetIDsForLinkTarget(app, target.collection, target.field, target.id)
		if err != nil {
			return nil, err
		}
		for _, assetID := range assetIDs {
			linked[assetID] = true
		}
	}
	return linked, nil
}

func assetLibraryPaginationForRequest(data pluginAssetLibraryRequest, request pluginAssetLibraryActionInput, hasLocation bool) assetLibraryPagination {
	if hasLocation || len(request.Points) > 0 {
		return assetLibraryPagination{}
	}
	perPage := data.PerPage
	if perPage <= 0 {
		perPage = 100
	}
	if perPage > 250 {
		perPage = 250
	}
	page := data.Page
	if page <= 0 {
		page = 1
	}
	return assetLibraryPagination{Page: page, PerPage: perPage, Enabled: true}
}

func assetLibraryRecords(app core.App, actorID string, request pluginAssetLibraryActionInput, hasLocation bool, pagination assetLibraryPagination) ([]*core.Record, bool, error) {
	filters := []string{"author={:author}", "type='photo'"}
	params := dbx.Params{"author": actorID}
	if request.TakenAfter != "" {
		takenAfter, err := normalizeAssetLibraryDateFilter(request.TakenAfter)
		if err != nil {
			return nil, false, apis.NewBadRequestError("invalid takenAfter", err)
		}
		filters = append(filters, "taken_at >= {:takenAfter}")
		params["takenAfter"] = takenAfter
	}
	if request.TakenBefore != "" {
		takenBefore, err := normalizeAssetLibraryDateFilter(request.TakenBefore)
		if err != nil {
			return nil, false, apis.NewBadRequestError("invalid takenBefore", err)
		}
		filters = append(filters, "taken_at <= {:takenBefore}")
		params["takenBefore"] = takenBefore
	}
	if bounds, ok := assetLibraryCoordinateBounds(request, hasLocation); ok {
		filters = append(filters, "lat >= {:minLat}", "lat <= {:maxLat}", "lon >= {:minLon}", "lon <= {:maxLon}")
		params["minLat"] = bounds.minLat
		params["maxLat"] = bounds.maxLat
		params["minLon"] = bounds.minLon
		params["maxLon"] = bounds.maxLon
	}
	limit := -1
	offset := 0
	if pagination.Enabled {
		limit = pagination.PerPage + 1
		offset = (pagination.Page - 1) * pagination.PerPage
	}
	records, err := app.FindRecordsByFilter(
		"assets",
		strings.Join(filters, " && "),
		"-taken_at,-created",
		limit,
		offset,
		params,
	)
	if err != nil {
		return nil, false, err
	}
	hasMore := false
	if pagination.Enabled && len(records) > pagination.PerPage {
		hasMore = true
		records = records[:pagination.PerPage]
	}
	return records, hasMore, nil
}

func assetLibraryExistingExternalRefs(app core.App, actorID string) ([]assetLibraryExternalRef, error) {
	rows := []assetLibraryExternalRef{}
	err := app.DB().
		Select("external_provider AS provider", "external_id AS id").
		From("assets").
		Where(dbx.NewExp(
			"author={:author} AND external_provider != '' AND external_id != ''",
			dbx.Params{"author": actorID},
		)).
		All(&rows)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func normalizeAssetLibraryDateFilter(value string) (string, error) {
	parsed, err := types.ParseDateTime(value)
	if err != nil {
		return "", err
	}
	return parsed.String(), nil
}

type assetLibraryBounds struct {
	minLat float64
	maxLat float64
	minLon float64
	maxLon float64
}

func assetLibraryCoordinateBounds(request pluginAssetLibraryActionInput, hasLocation bool) (assetLibraryBounds, bool) {
	if !hasLocation && len(request.Points) == 0 {
		return assetLibraryBounds{}, false
	}

	minLat := math.Inf(1)
	maxLat := math.Inf(-1)
	minLon := math.Inf(1)
	maxLon := math.Inf(-1)
	addPoint := func(lat float64, lon float64) {
		if !isFiniteCoordinate(lat, lon) {
			return
		}
		minLat = math.Min(minLat, lat)
		maxLat = math.Max(maxLat, lat)
		minLon = math.Min(minLon, lon)
		maxLon = math.Max(maxLon, lon)
	}
	for _, point := range request.Points {
		addPoint(point.Lat, point.Lon)
	}
	if hasLocation {
		addPoint(request.Lat, request.Lon)
	}
	if math.IsInf(minLat, 1) {
		return assetLibraryBounds{}, false
	}

	paddingMeters := assetLibraryMaxDistance(request, hasLocation)
	if math.IsInf(paddingMeters, 1) {
		return assetLibraryBounds{}, false
	}
	latPadding := paddingMeters / 111320
	minLat = math.Max(-90, minLat-latPadding)
	maxLat = math.Min(90, maxLat+latPadding)

	maxAbsLat := math.Max(math.Abs(minLat), math.Abs(maxLat))
	cosLat := math.Cos(maxAbsLat * math.Pi / 180)
	if cosLat <= 0.01 {
		return assetLibraryBounds{}, false
	}
	lonPadding := paddingMeters / (111320 * cosLat)
	minLon -= lonPadding
	maxLon += lonPadding
	if minLon < -180 || maxLon > 180 {
		return assetLibraryBounds{}, false
	}

	return assetLibraryBounds{
		minLat: minLat,
		maxLat: maxLat,
		minLon: minLon,
		maxLon: maxLon,
	}, true
}

func isFiniteCoordinate(lat float64, lon float64) bool {
	return !math.IsNaN(lat) && !math.IsInf(lat, 0) &&
		!math.IsNaN(lon) && !math.IsInf(lon, 0) &&
		lat >= -90 && lat <= 90 &&
		lon >= -180 && lon <= 180
}

func assetLibraryMaxDistance(request pluginAssetLibraryActionInput, hasLocation bool) float64 {
	if !hasLocation && len(request.Points) == 0 {
		return math.Inf(1)
	}
	if request.DoubleRadius {
		return 2000
	}
	return 1000
}

func assetLibraryCandidate(record *core.Record, request pluginAssetLibraryActionInput, hasLocation bool) (pluginAssetCandidate, bool) {
	thumbnailURL := util.AssetPublicMediaURL(record, "")
	if thumbnailURL == "" || util.IsGeneratedRoutePreviewAsset(record) {
		return pluginAssetCandidate{}, false
	}

	lat := record.GetFloat("lat")
	lon := record.GetFloat("lon")
	if lat == 0 && lon == 0 {
		return pluginAssetCandidate{}, false
	}

	metadata := map[string]any{}
	_ = record.UnmarshalJSONField("metadata", &metadata)
	nearest, hasNearest := nearestAssetLibraryTrackPoint(request.Points, lat, lon)
	distance := 0.0
	pointLat := 0.0
	pointLon := 0.0
	distanceFromStart := 0.0
	if hasNearest {
		distance = nearest.distanceToPhoto
		pointLat = nearest.lat
		pointLon = nearest.lon
		distanceFromStart = nearest.distanceFromStart
	} else if hasLocation {
		distance = util.HaversineDistanceMeters(request.Lat, request.Lon, lat, lon)
	}

	providerID := record.GetString("external_provider")
	if providerID == "" {
		providerID = "wanderer"
	}
	takenAt := util.RecordDateTimeRFC3339(record, "taken_at")
	if takenAt == "" {
		takenAt = util.RecordDateTimeRFC3339(record, "created")
	}

	return pluginAssetCandidate{
		Source:            "wanderer",
		ProviderID:        providerID,
		ExternalProvider:  record.GetString("external_provider"),
		ExternalID:        record.GetString("external_id"),
		AssetID:           record.Id,
		OriginalFileName:  assetLibraryFilename(record, metadata),
		TakenAt:           takenAt,
		Lat:               lat,
		Lon:               lon,
		PointLat:          pointLat,
		PointLon:          pointLon,
		Distance:          distance,
		DistanceFromStart: distanceFromStart,
		City:              "",
		Country:           "",
		ThumbnailURL:      thumbnailURL,
	}, true
}

func assetLibraryFilename(record *core.Record, metadata map[string]any) string {
	if file := record.GetString("file"); file != "" {
		return file
	}
	if remoteFilename := util.AssetMetadataString(metadata, "remote", "filename"); remoteFilename != "" {
		return remoteFilename
	}
	if externalID := record.GetString("external_id"); externalID != "" {
		return externalID
	}
	return record.Id
}

type nearestAssetLibraryPoint struct {
	lat               float64
	lon               float64
	distanceFromStart float64
	distanceToPhoto   float64
}

func nearestAssetLibraryTrackPoint(points []pluginAssetTrackPoint, lat float64, lon float64) (nearestAssetLibraryPoint, bool) {
	var best nearestAssetLibraryPoint
	found := false
	for _, point := range points {
		distanceToPhoto := util.HaversineDistanceMeters(point.Lat, point.Lon, lat, lon)
		if !found || distanceToPhoto < best.distanceToPhoto {
			best = nearestAssetLibraryPoint{
				lat:               point.Lat,
				lon:               point.Lon,
				distanceFromStart: point.Distance,
				distanceToPhoto:   distanceToPhoto,
			}
			found = true
		}
	}
	return best, found
}

func sortAssetLibraryCandidates(candidates []pluginAssetCandidate, hasTrackPoints bool) {
	sort.Slice(candidates, func(i int, j int) bool {
		a := candidates[i]
		b := candidates[j]
		if hasTrackPoints && a.DistanceFromStart != b.DistanceFromStart {
			return a.DistanceFromStart < b.DistanceFromStart
		}
		if a.Distance != b.Distance {
			return a.Distance < b.Distance
		}
		aTime := assetLibraryCandidateTime(a.TakenAt)
		bTime := assetLibraryCandidateTime(b.TakenAt)
		if !aTime.Equal(bTime) {
			return aTime.After(bTime)
		}
		return a.AssetID < b.AssetID
	})
}

func assetLibraryCandidateTime(value string) time.Time {
	if strings.TrimSpace(value) == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func importAssetPluginPhotos(e *core.RequestEvent, plugin pluginsystem.LocalPlugin, instance *core.Record, auth map[string]any, config map[string]any, output pluginAssetLibraryOutput, data pluginAssetLibraryRequest, waypointID string) error {
	results, err := importAssetPluginPhotosForTrail(e.Request.Context(), e.App, e.Auth.Id, plugin, auth, config, output, data, waypointID, false)
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, pluginAssetImportResponse{Imported: results, Omitted: append([]pluginAssetOmission{}, output.OmittedAssets...)})
}

func importAssetPluginPhotosToTarget(e *core.RequestEvent, plugin pluginsystem.LocalPlugin, auth map[string]any, config map[string]any, output pluginAssetLibraryOutput, data pluginAssetLibraryRequest) error {
	if len(output.Photos) == 0 {
		return e.JSON(http.StatusOK, pluginAssetImportResponse{Imported: []pluginAssetImportResult{}, Omitted: append([]pluginAssetOmission{}, output.OmittedAssets...)})
	}
	actor, err := e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
	if err != nil {
		return err
	}
	hostConfig := pluginhost.HostConfig(config)
	records, err := importer.ImportPhotoAssets(e.Request.Context(), e.App, output.Photos, importer.Options{
		UserID:      e.Auth.Id,
		ActorID:     actor.Id,
		PhotoMode:   util.ConfigString(hostConfig, "photoMode"),
		PhotoLimits: assetPluginPhotoImportLimits(hostConfig, false),
		Manifest:    plugin.Manifest,
		Policy:      pluginhost.InstancePolicy(plugin, config).WithHostAuth(auth),
		Auth:        auth,
	}, importer.PhotoAssetTarget{
		Trail:       data.TrailID,
		Waypoint:    data.WaypointID,
		SummitLog:   data.SummitLogID,
		PublicTrail: trailIsPublic(e.App, data.TrailID),
	})
	if err != nil {
		return err
	}
	results := make([]pluginAssetImportResult, 0, len(records))
	var waypoint *core.Record
	if data.WaypointID != "" {
		waypoint, _ = e.App.FindRecordById("waypoints", data.WaypointID)
	}
	for _, record := range records {
		result := pluginAssetImportResult{Asset: record, Waypoint: waypoint}
		result.AssetID = record.GetString("external_id")
		results = append(results, result)
	}
	return e.JSON(http.StatusOK, pluginAssetImportResponse{Imported: results, Omitted: append([]pluginAssetOmission{}, output.OmittedAssets...)})
}

func importAssetPluginPhotosForTrail(ctx context.Context, app core.App, userID string, plugin pluginsystem.LocalPlugin, auth map[string]any, config map[string]any, output pluginAssetLibraryOutput, data pluginAssetLibraryRequest, waypointID string, enforceLimits bool) ([]pluginAssetImportResult, error) {
	if len(output.Photos) == 0 {
		return []pluginAssetImportResult{}, nil
	}
	actor, err := app.FindFirstRecordByData("activitypub_actors", "user", userID)
	if err != nil {
		return nil, err
	}
	hostConfig := pluginhost.HostConfig(config)
	opts := importer.Options{
		UserID:      userID,
		ActorID:     actor.Id,
		PhotoMode:   util.ConfigString(hostConfig, "photoMode"),
		PhotoLimits: assetPluginPhotoImportLimits(hostConfig, enforceLimits),
		Manifest:    plugin.Manifest,
		Policy:      pluginhost.InstancePolicy(plugin, config).WithHostAuth(auth),
		Auth:        auth,
	}
	if waypointID != "" {
		records, err := importer.ImportPhotoAssets(ctx, app, output.Photos, opts, importer.PhotoAssetTarget{
			Trail:       data.TrailID,
			Waypoint:    waypointID,
			PublicTrail: trailIsPublic(app, data.TrailID),
		})
		if err != nil {
			return nil, err
		}
		results := make([]pluginAssetImportResult, 0, len(records))
		waypoint, _ := app.FindRecordById("waypoints", waypointID)
		for _, record := range records {
			result := pluginAssetImportResult{AssetID: record.GetString("external_id"), Asset: record, Waypoint: waypoint}
			results = append(results, result)
		}
		return results, nil
	}

	trackPoints := []pluginAssetTrackPoint{}
	if data.TrailID != "" {
		points, err := assetPluginWaypointTrackPoints(app, data.TrailID)
		if err != nil {
			app.Logger().Warn("failed to read trail track points for asset plugin waypoints", "trail", data.TrailID, "error", err)
		} else {
			trackPoints = points
		}
	}
	clusters, photosByClusterID, mergeSettings, err := assetPluginPhotoClusters(app, data.TrailID, output.Photos)
	if err != nil {
		return nil, err
	}
	if enforceLimits {
		clusters = limitAssetPluginWaypointClusters(clusters, assetPluginMaxWaypoints(config))
	}

	results := make([]pluginAssetImportResult, 0, len(output.Photos))
	for _, cluster := range clusters {
		photos := assetPluginPhotosForCluster(cluster.Photos, photosByClusterID)
		if len(photos) == 0 {
			continue
		}
		var waypoint *core.Record
		createdWaypoint := false
		target := importer.PhotoAssetTarget{
			Trail:       data.TrailID,
			PublicTrail: trailIsPublic(app, data.TrailID),
		}
		if assetPluginClusterHasWaypointTarget(cluster) {
			var err error
			waypoint, createdWaypoint, err = assetPluginWaypointForCluster(ctx, app, actor.Id, data.TrailID, cluster, trackPoints, mergeSettings)
			if err != nil {
				return nil, err
			}
			target.Waypoint = waypoint.Id
		}
		records, err := importer.ImportPhotoAssets(ctx, app, photos, opts, target)
		if err != nil {
			if createdWaypoint {
				if deleteErr := app.Delete(waypoint); deleteErr != nil {
					app.Logger().Warn("failed to delete asset plugin waypoint after photo import error", "waypoint", waypoint.Id, "error", deleteErr)
				}
			}
			return nil, err
		}
		if len(records) == 0 {
			if createdWaypoint {
				if deleteErr := app.Delete(waypoint); deleteErr != nil {
					return nil, deleteErr
				}
			}
			continue
		}
		for _, record := range records {
			result := pluginAssetImportResult{AssetID: record.GetString("external_id"), Waypoint: waypoint}
			result.Asset = record
			results = append(results, result)
		}
	}
	return results, nil
}

func assetPluginPhotoImportLimits(hostConfig map[string]any, enforce bool) *importer.PhotoImportLimits {
	if !enforce {
		return nil
	}
	limits := pluginPhotoImportLimits(hostConfig)
	return &limits
}

func assetPluginWaypointForCluster(ctx context.Context, app core.App, actorID string, trailID string, cluster waypointPhotoCluster, trackPoints []pluginAssetTrackPoint, mergeSettings waypointMergeSettings) (*core.Record, bool, error) {
	if cluster.Waypoint != "" {
		waypoint, err := app.FindRecordById("waypoints", cluster.Waypoint)
		return waypoint, false, err
	}
	if !assetPluginClusterHasLocation(cluster) {
		return nil, false, fmt.Errorf("asset plugin waypoint cluster has no coordinates")
	}
	name := ""
	if cluster.Count > 0 {
		resolvedName, _ := resolveWaypointName(ctx, app.Logger(), cluster.Lat, cluster.Lon, mergeSettings.Radius)
		name = resolvedName
	}
	waypoint, err := createAssetPluginWaypoint(app, actorID, trailID, name, cluster.Lat, cluster.Lon, trackPoints)
	return waypoint, true, err
}

func assetPluginPhotoClusters(app core.App, trailID string, photos []pluginsystem.Photo) ([]waypointPhotoCluster, map[string]pluginsystem.Photo, waypointMergeSettings, error) {
	photosByID := map[string]pluginsystem.Photo{}
	clusterPhotos := make([]waypointClusterPhoto, 0, len(photos))
	standaloneClusters := []waypointPhotoCluster{}
	for i, photo := range photos {
		id := assetPluginPhotoClusterID(i, photo)
		photosByID[id] = photo
		if photo.Lat == nil || photo.Lon == nil {
			standaloneClusters = append(standaloneClusters, waypointPhotoCluster{Photos: []string{id}})
			continue
		}
		clusterPhotos = append(clusterPhotos, waypointClusterPhoto{
			ID:  id,
			Lat: *photo.Lat,
			Lon: *photo.Lon,
		})
	}

	existingWaypoints, categoryID, err := assetPluginWaypointClusterContext(app, trailID)
	if err != nil {
		return nil, nil, waypointMergeSettings{}, err
	}
	mergeSettings, err := getWaypointMergeSettings(app, categoryID)
	if err != nil {
		return nil, nil, waypointMergeSettings{}, err
	}
	clusters := clusterWaypointPhotos(clusterPhotos, existingWaypoints, mergeSettings)
	clusters = append(clusters, standaloneClusters...)
	return clusters, photosByID, mergeSettings, nil
}

func assetPluginPhotoClusterID(index int, photo pluginsystem.Photo) string {
	if strings.TrimSpace(photo.ExternalID) != "" {
		return photo.ExternalID
	}
	return fmt.Sprintf("photo_%d", index)
}

func assetPluginMaxWaypoints(config map[string]any) int {
	pluginConfig := pluginhost.RuntimeConfig(config)
	if len(pluginConfig) == 0 {
		pluginConfig = config
	}
	return util.ConfigInt(pluginConfig, "maxWaypoints", defaultAssetPluginMaxWaypoints)
}

func limitAssetPluginWaypointClusters(clusters []waypointPhotoCluster, maxNewWaypoints int) []waypointPhotoCluster {
	if maxNewWaypoints <= 0 || len(clusters) == 0 {
		return clusters
	}

	limited := make([]waypointPhotoCluster, 0, len(clusters))
	newWaypointCount := 0
	for _, cluster := range clusters {
		if !assetPluginClusterCreatesWaypoint(cluster) {
			limited = append(limited, cluster)
			continue
		}
		if newWaypointCount >= maxNewWaypoints {
			continue
		}
		newWaypointCount++
		limited = append(limited, cluster)
	}
	return limited
}

func assetPluginClusterCreatesWaypoint(cluster waypointPhotoCluster) bool {
	return cluster.Waypoint == "" && len(cluster.Photos) > 0 && assetPluginClusterHasLocation(cluster)
}

func assetPluginClusterHasWaypointTarget(cluster waypointPhotoCluster) bool {
	return cluster.Waypoint != "" || assetPluginClusterCreatesWaypoint(cluster)
}

func assetPluginClusterHasLocation(cluster waypointPhotoCluster) bool {
	return cluster.Count > 0
}

func assetPluginPhotosForCluster(ids []string, photosByID map[string]pluginsystem.Photo) []pluginsystem.Photo {
	photos := make([]pluginsystem.Photo, 0, len(ids))
	for _, id := range ids {
		photo, ok := photosByID[id]
		if ok {
			photos = append(photos, photo)
		}
	}
	return photos
}

func assetPluginWaypointClusterContext(app core.App, trailID string) ([]waypointClusterWaypoint, string, error) {
	if trailID == "" {
		return nil, "", nil
	}
	trail, err := app.FindRecordById("trails", trailID)
	if err != nil {
		return nil, "", err
	}
	records, err := app.FindRecordsByFilter(
		"waypoints",
		"trail={:trail}",
		"distance_from_start",
		-1,
		0,
		dbx.Params{"trail": trailID},
	)
	if err != nil {
		return nil, "", err
	}
	waypoints := make([]waypointClusterWaypoint, 0, len(records))
	for _, record := range records {
		waypoints = append(waypoints, waypointClusterWaypoint{
			ID:  record.Id,
			Lat: record.GetFloat("lat"),
			Lon: record.GetFloat("lon"),
		})
	}
	return waypoints, trail.GetString("category"), nil
}

func assetPluginProviderEnabled(hostConfig map[string]any, provider string) bool {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return false
	}
	key := autoAttachProviderKey(provider)
	if raw, ok := hostConfig["autoAttach"].(map[string]any); ok {
		return autoAttachEnabled(raw, key)
	}
	return true
}

func autoAttachProviderKey(provider string) string {
	if provider == pluginAssetMaintenanceProvider {
		return pluginAssetMaintenanceProvider
	}
	if provider == "upload" {
		return "upload"
	}
	return "trailPlugins"
}

func autoAttachEnabled(raw map[string]any, key string) bool {
	value, ok := raw[key]
	if !ok {
		return true
	}
	enabled, ok := value.(bool)
	return !ok || enabled
}

func existingTrailAssetExternalIDs(app core.App, userID string, pluginID string, trailID string, assetIDs []string) (map[string]bool, error) {
	existing := map[string]bool{}
	if userID == "" || pluginID == "" || trailID == "" || len(assetIDs) == 0 {
		return existing, nil
	}
	actorID, err := util.ResolveAssetAuthor(app, userID)
	if err != nil {
		return nil, err
	}
	if actorID == "" {
		return existing, nil
	}

	params := dbx.Params{
		"author": actorID,
		"plugin": pluginID,
	}
	idFilters := make([]string, 0, len(assetIDs))
	seen := map[string]bool{}
	for _, assetID := range assetIDs {
		assetID = strings.TrimSpace(assetID)
		if assetID == "" || seen[assetID] {
			continue
		}
		seen[assetID] = true
		param := fmt.Sprintf("asset_%d", len(idFilters))
		params[param] = assetID
		idFilters = append(idFilters, "external_id={:"+param+"}")
	}
	if len(idFilters) == 0 {
		return existing, nil
	}

	records, err := app.FindRecordsByFilter(
		"assets",
		"author={:author} && external_provider={:plugin} && ("+strings.Join(idFilters, " || ")+")",
		"",
		len(idFilters),
		0,
		params,
	)
	if err != nil {
		return nil, err
	}
	recordIDs := make([]string, 0, len(records))
	for _, record := range records {
		recordIDs = append(recordIDs, record.Id)
	}
	linkedAssetIDs, err := util.AssetIDsLinkedToTrail(app, trailID, recordIDs)
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		if linkedAssetIDs[record.Id] {
			existing[record.GetString("external_id")] = true
		}
	}
	return existing, nil
}

func assetPluginWaypointTrackPoints(app core.App, trailID string) ([]pluginAssetTrackPoint, error) {
	request, err := assetLibraryActionInputForApp(app, pluginAssetLibraryRequest{TrailID: trailID}, false)
	if err != nil {
		return nil, err
	}
	return request.Points, nil
}

func trailIsPublic(app core.App, trailID string) bool {
	if trailID == "" {
		return false
	}
	trail, err := app.FindRecordById("trails", trailID)
	return err == nil && trail.GetBool("public")
}

func createAssetPluginWaypoint(app core.App, actorID string, trailID string, name string, lat float64, lon float64, trackPoints []pluginAssetTrackPoint) (*core.Record, error) {
	collection, err := app.FindCollectionByNameOrId("waypoints")
	if err != nil {
		return nil, err
	}
	waypoint := core.NewRecord(collection)
	distanceFromStart := assetPluginDistanceFromStart(trackPoints, lat, lon)
	waypoint.Load(map[string]any{
		"name":                strings.TrimSpace(name),
		"description":         "",
		"lat":                 lat,
		"lon":                 lon,
		"icon":                "camera",
		"author":              actorID,
		"trail":               trailID,
		"distance_from_start": distanceFromStart,
	})
	if err := app.Save(waypoint); err != nil {
		return nil, err
	}
	return waypoint, nil
}

func assetPluginDistanceFromStart(points []pluginAssetTrackPoint, lat, lon float64) float64 {
	bestDistance := -1.0
	bestFromStart := 0.0
	for _, point := range points {
		distance := util.HaversineDistanceMeters(lat, lon, point.Lat, point.Lon)
		if bestDistance < 0 || distance < bestDistance {
			bestDistance = distance
			bestFromStart = point.Distance
		}
	}
	return bestFromStart
}

func userOwnsTrail(app core.App, userID, trailID string) (bool, error) {
	if userID == "" || trailID == "" {
		return false, nil
	}
	trail, err := app.FindRecordById("trails", trailID)
	if err != nil {
		return false, err
	}
	return actorBelongsToUser(app, userID, trail.GetString("author"))
}

func userOwnsWaypoint(app core.App, userID, waypointID, trailID string) (bool, error) {
	if userID == "" || waypointID == "" {
		return false, nil
	}
	waypoint, err := app.FindRecordById("waypoints", waypointID)
	if err != nil {
		return false, err
	}
	if trailID != "" && waypoint.GetString("trail") != trailID {
		return false, nil
	}
	return userOwnsTrail(app, userID, waypoint.GetString("trail"))
}

func userOwnsSummitLog(app core.App, userID, summitLogID, trailID string) (bool, error) {
	if userID == "" || summitLogID == "" {
		return false, nil
	}
	summitLog, err := app.FindRecordById("summit_logs", summitLogID)
	if err != nil {
		return false, err
	}
	if trailID != "" && summitLog.GetString("trail") != trailID {
		return false, nil
	}
	return userOwnsTrail(app, userID, summitLog.GetString("trail"))
}

func userCanEditTrailTarget(app core.App, userID, trailID string) (bool, error) {
	if userID == "" || trailID == "" {
		return false, nil
	}
	owns, err := userOwnsTrail(app, userID, trailID)
	if err != nil {
		return false, err
	}
	if owns {
		return true, nil
	}

	actor, err := app.FindFirstRecordByData("activitypub_actors", "user", userID)
	if err != nil {
		return false, err
	}
	_, err = app.FindFirstRecordByFilter(
		"trail_share",
		"trail={:trail} && actor={:actor} && permission='edit'",
		dbx.Params{"trail": trailID, "actor": actor.Id},
	)
	if err == nil {
		return true, nil
	}
	if err == sql.ErrNoRows {
		return false, nil
	}
	return false, err
}

func userCanEditWaypointTarget(app core.App, userID, waypointID, trailID string) (bool, error) {
	if userID == "" || waypointID == "" {
		return false, nil
	}
	waypoint, err := app.FindRecordById("waypoints", waypointID)
	if err != nil {
		return false, err
	}
	if trailID != "" && waypoint.GetString("trail") != trailID {
		return false, nil
	}
	if ok, err := actorBelongsToUser(app, userID, waypoint.GetString("author")); err != nil || ok {
		return ok, err
	}
	return userCanEditTrailTarget(app, userID, waypoint.GetString("trail"))
}

func userCanEditSummitLogTarget(app core.App, userID, summitLogID, trailID string) (bool, error) {
	if userID == "" || summitLogID == "" {
		return false, nil
	}
	summitLog, err := app.FindRecordById("summit_logs", summitLogID)
	if err != nil {
		return false, err
	}
	if trailID != "" && summitLog.GetString("trail") != trailID {
		return false, nil
	}
	if ok, err := actorBelongsToUser(app, userID, summitLog.GetString("author")); err != nil || ok {
		return ok, err
	}
	return userCanEditTrailTarget(app, userID, summitLog.GetString("trail"))
}

func actorBelongsToUser(app core.App, userID, actorID string) (bool, error) {
	if userID == "" || actorID == "" {
		return false, nil
	}
	if actorID == userID {
		return true, nil
	}
	actor, err := app.FindRecordById("activitypub_actors", actorID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return actor.GetString("user") == userID, nil
}

func ensureCanEditTrailTarget(app core.App, userID, trailID string) error {
	ok, err := userCanEditTrailTarget(app, userID, trailID)
	if err != nil {
		return err
	}
	if !ok {
		return apis.NewForbiddenError("Insufficient permissions for trail", nil)
	}
	return nil
}

func ensureCanEditSummitLogTarget(app core.App, userID, summitLogID, trailID string) error {
	ok, err := userCanEditSummitLogTarget(app, userID, summitLogID, trailID)
	if err != nil {
		return err
	}
	if !ok {
		return apis.NewForbiddenError("Insufficient permissions for summit log", nil)
	}
	return nil
}

func ensureCanEditWaypointTarget(app core.App, userID, waypointID, trailID string) error {
	ok, err := userCanEditWaypointTarget(app, userID, waypointID, trailID)
	if err != nil {
		return err
	}
	if !ok {
		return apis.NewForbiddenError("Insufficient permissions for waypoint", nil)
	}
	return nil
}

func ensureOwnsTrail(app core.App, userID, trailID string) error {
	ok, err := userOwnsTrail(app, userID, trailID)
	if err != nil {
		return err
	}
	if !ok {
		return apis.NewForbiddenError("Insufficient permissions for trail", nil)
	}
	return nil
}

func ensureOwnsSummitLog(app core.App, userID, summitLogID, trailID string) error {
	ok, err := userOwnsSummitLog(app, userID, summitLogID, trailID)
	if err != nil {
		return err
	}
	if !ok {
		return apis.NewForbiddenError("Insufficient permissions for summit log", nil)
	}
	return nil
}

func ensureOwnsWaypoint(app core.App, userID, waypointID, trailID string) error {
	ok, err := userOwnsWaypoint(app, userID, waypointID, trailID)
	if err != nil {
		return err
	}
	if !ok {
		return apis.NewForbiddenError("Insufficient permissions for waypoint", nil)
	}
	return nil
}
