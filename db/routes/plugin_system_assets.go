package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"pocketbase/plugins/importer"
	"pocketbase/pluginsystem"
	"pocketbase/services/pluginhost"
	"pocketbase/util"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/tkrajina/gpxgo/gpx"
)

const pluginAssetThumbnailMaxBytes int64 = 8 << 20
const defaultAssetPluginMaxWaypoints = 25
const pluginAssetMaintenanceProvider = "maintenance"

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
}

type pluginAssetLibraryInput struct {
	Instance pluginsystem.InstanceRef      `json:"instance"`
	Auth     map[string]any                `json:"auth,omitempty"`
	Config   map[string]any                `json:"config,omitempty"`
	Limits   importer.PhotoImportLimits    `json:"limits,omitempty"`
	Request  pluginAssetLibraryActionInput `json:"request"`
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

type pluginAssetTrackPoint struct {
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	Distance  float64 `json:"distance,omitempty"`
	Timestamp string  `json:"timestamp,omitempty"`
}

type pluginAssetLibraryOutput struct {
	OK            bool                      `json:"ok"`
	UserID        string                    `json:"userId,omitempty"`
	Candidates    []pluginAssetCandidate    `json:"candidates,omitempty"`
	Photos        []pluginsystem.Photo      `json:"photos,omitempty"`
	HasMore       bool                      `json:"hasMore,omitempty"`
	TakenAfter    string                    `json:"takenAfter,omitempty"`
	HasTimestamps bool                      `json:"hasTimestamps,omitempty"`
	Error         *pluginsystem.PluginError `json:"error,omitempty"`
}

type pluginAssetCandidate struct {
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
}

type pluginAssetImportResult struct {
	AssetID  string       `json:"assetId"`
	Waypoint *core.Record `json:"waypoint,omitempty"`
	Asset    *core.Record `json:"asset,omitempty"`
}

type pluginAssetAutoAttachRequest struct {
	TrailID  string `json:"trailId"`
	Provider string `json:"provider"`
}

type pluginAssetAutoAttachPluginResult struct {
	PluginID   string `json:"pluginId"`
	InstanceID string `json:"instanceId"`
	Imported   int    `json:"imported"`
	Error      string `json:"error,omitempty"`
}

type pluginAssetAutoAttachSummary struct {
	OK       bool                                `json:"ok"`
	TrailID  string                              `json:"trailId"`
	Imported int                                 `json:"imported"`
	Plugins  []pluginAssetAutoAttachPluginResult `json:"plugins"`
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

	for _, trail := range trails {
		if strings.TrimSpace(trail.GetString("gpx")) == "" {
			continue
		}
		hasPhotos, err := trailHasVisiblePhotoAssets(e.App, trail.Id)
		if err != nil {
			return err
		}
		if hasPhotos {
			continue
		}
		response.Trails = append(response.Trails, pluginAssetMaintenanceTrailCandidate(e.App, trail))
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
	plugin, capability, instance, auth, config, err := assetPluginInvocation(e, pluginID)
	if err != nil {
		return err
	}
	output, err := callAssetPlugin(e.Request.Context(), plugin, capability, instance, auth, config, pluginAssetLibraryActionInput{
		Action:   "thumbnail",
		AssetIDs: []string{assetID},
	})
	if err != nil {
		return err
	}
	if output.Error != nil {
		return apis.NewBadRequestError(output.Error.Message, output.Error)
	}
	if len(output.Photos) == 0 {
		return e.NotFoundError("thumbnail not found", nil)
	}
	fetched, err := importer.FetchPhotoMedia(e.Request.Context(), output.Photos[0], importer.Options{
		UserID:   e.Auth.Id,
		Manifest: plugin.Manifest,
		Policy:   pluginhost.InstancePolicy(plugin, config).WithHostAuth(auth),
		Auth:     auth,
	}, pluginAssetThumbnailMaxBytes)
	if err != nil {
		return e.NotFoundError("thumbnail not available", err)
	}
	contentType := fetched.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	e.Response.Header().Set("Cache-Control", "private, max-age=300")
	return e.Blob(http.StatusOK, contentType, fetched.Body)
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
	output, err := callAssetPlugin(e.Request.Context(), plugin, capability, instance, auth, config, request)
	if err != nil {
		return err
	}
	if output.Error != nil {
		return apis.NewBadRequestError(output.Error.Message, output.Error)
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
		}
		imported, err := autoAttachAssetPluginForTrail(ctx, app, userID, trailID, plugin, capability, instance, auth, config)
		if err != nil {
			result.Error = err.Error()
			summary.Plugins = append(summary.Plugins, result)
			app.Logger().Warn("asset plugin auto attach failed for plugin", "plugin", plugin.Manifest.ID, "instance", instance.Id, "provider", provider, "trail", trailID, "error", err)
			continue
		}
		result.Imported = imported
		summary.Imported += imported
		summary.Plugins = append(summary.Plugins, result)
		if imported > 0 {
			app.Logger().Info("asset plugin auto attach imported photos", "plugin", plugin.Manifest.ID, "instance", instance.Id, "provider", provider, "trail", trailID, "photos", imported)
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

func pluginAssetMaintenanceTrailCandidate(app core.App, trail *core.Record) pluginAssetTrailPhotoMaintenanceCandidate {
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
		Thumbnail:     trailMaintenanceThumbnail(app, trail.Id),
	}
}

func trailMaintenanceThumbnail(app core.App, trailID string) string {
	assetIDs, err := util.AssetIDsForTrail(app, trailID)
	if err != nil {
		return ""
	}
	for _, assetID := range assetIDs {
		asset, err := app.FindRecordById("assets", assetID)
		if err != nil {
			continue
		}
		if asset.GetString("type") == "photo" && util.IsGeneratedRoutePreviewAsset(asset) {
			return util.AssetPublicMediaURL(asset, "")
		}
	}
	return ""
}

func trailHasVisiblePhotoAssets(app core.App, trailID string) (bool, error) {
	assetIDs, err := util.AssetIDsForTrail(app, trailID)
	if err != nil {
		return false, err
	}
	for _, assetID := range assetIDs {
		asset, err := app.FindRecordById("assets", assetID)
		if err != nil {
			return false, err
		}
		if asset.GetString("type") == "photo" && !util.IsGeneratedRoutePreviewAsset(asset) {
			return true, nil
		}
	}
	return false, nil
}

func autoAttachAssetPluginForTrail(ctx context.Context, app core.App, userID string, trailID string, plugin pluginsystem.LocalPlugin, capability pluginsystem.CapabilityManifest, instance *core.Record, auth map[string]any, config map[string]any) (int, error) {
	candidatesRequest, err := assetLibraryActionInputForApp(app, pluginAssetLibraryRequest{
		Action:  "candidates",
		TrailID: trailID,
	}, true)
	if err != nil {
		return 0, err
	}
	if !assetPluginAutoAttachHasTimeWindow(candidatesRequest) {
		return 0, nil
	}
	candidatesOutput, err := callAssetPlugin(ctx, plugin, capability, instance, auth, config, candidatesRequest)
	if err != nil {
		return 0, err
	}
	if candidatesOutput.Error != nil {
		return 0, fmt.Errorf("%s: %s", candidatesOutput.Error.Code, candidatesOutput.Error.Message)
	}

	assetIDs := make([]string, 0, len(candidatesOutput.Candidates))
	seen := map[string]bool{}
	for _, candidate := range candidatesOutput.Candidates {
		assetID := strings.TrimSpace(candidate.AssetID)
		if assetID == "" || seen[assetID] {
			continue
		}
		seen[assetID] = true
		assetIDs = append(assetIDs, assetID)
	}
	if len(assetIDs) == 0 {
		return 0, nil
	}
	existing, err := existingTrailAssetExternalIDs(app, userID, plugin.Manifest.ID, trailID, assetIDs)
	if err != nil {
		return 0, err
	}
	filteredAssetIDs := assetIDs[:0]
	for _, assetID := range assetIDs {
		if !existing[assetID] {
			filteredAssetIDs = append(filteredAssetIDs, assetID)
		}
	}
	if len(filteredAssetIDs) == 0 {
		return 0, nil
	}

	importRequest := candidatesRequest
	importRequest.Action = "import"
	importRequest.AssetIDs = filteredAssetIDs
	importOutput, err := callAssetPlugin(ctx, plugin, capability, instance, auth, config, importRequest)
	if err != nil {
		return 0, err
	}
	if importOutput.Error != nil {
		return 0, fmt.Errorf("%s: %s", importOutput.Error.Code, importOutput.Error.Message)
	}
	results, err := importAssetPluginPhotosForTrail(ctx, app, userID, plugin, auth, config, importOutput, pluginAssetLibraryRequest{
		PluginID: plugin.Manifest.ID,
		Action:   "import",
		TrailID:  trailID,
		AssetIDs: filteredAssetIDs,
	}, "", true)
	if err != nil {
		return 0, err
	}
	return len(results), nil
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
	instance, err := e.App.FindFirstRecordByFilter(
		"plugin_instances",
		"user={:user} && plugin_id={:plugin_id} && enabled=true",
		dbx.Params{"user": e.Auth.Id, "plugin_id": pluginID},
	)
	if err != nil {
		return pluginsystem.LocalPlugin{}, pluginsystem.CapabilityManifest{}, nil, nil, nil, apis.NewBadRequestError("no enabled asset plugin instance configured for this plugin", nil)
	}
	plugin, capability, err := localPluginCapability(e.App, pluginID, "asset_library", "v1")
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
	config := pluginhost.EffectiveConfig(e.App, plugin.Manifest.ID, instance)
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
		pluginsystem.DeepMergeConfig(config, submittedConfig)
	}
	config = assetPluginConnectorConfig(plugin, config)

	return plugin, capability, instance, auth, config, nil
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

func callAssetPlugin(ctx context.Context, plugin pluginsystem.LocalPlugin, capability pluginsystem.CapabilityManifest, instance *core.Record, auth map[string]any, config map[string]any, request pluginAssetLibraryActionInput) (pluginAssetLibraryOutput, error) {
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
	input := pluginAssetLibraryInput{
		Instance: pluginsystem.InstanceRef{ID: instance.Id, PluginID: instance.GetString("plugin_id")},
		Auth:     pluginsystem.PluginInputAuth(plugin, auth),
		Config:   pluginhost.RuntimeConfig(config),
		Limits:   pluginPhotoImportLimits(pluginhost.HostConfig(config)),
		Request:  request,
	}
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return pluginAssetLibraryOutput{}, err
	}
	outputBytes, err := session.Call(ctx, capability.Export, inputBytes)
	if err != nil {
		return pluginAssetLibraryOutput{}, err
	}
	var output pluginAssetLibraryOutput
	if err := json.Unmarshal(outputBytes, &output); err != nil {
		return pluginAssetLibraryOutput{}, fmt.Errorf("plugin returned invalid %s output: %w", capability.Export, err)
	}
	return output, nil
}

func assetLibraryActionInput(e *core.RequestEvent, data pluginAssetLibraryRequest) (pluginAssetLibraryActionInput, error) {
	return assetLibraryActionInputForApp(e.App, data, false)
}

func assetLibraryActionInputForApp(app core.App, data pluginAssetLibraryRequest, useTrailTime bool) (pluginAssetLibraryActionInput, error) {
	request := pluginAssetLibraryActionInput{
		Action:       data.Action,
		TrailID:      data.TrailID,
		Lat:          data.Lat,
		Lon:          data.Lon,
		TakenAfter:   strings.TrimSpace(data.TakenAfter),
		TakenBefore:  strings.TrimSpace(data.TakenBefore),
		DoubleRadius: data.DoubleRadius,
		AssetIDs:     data.AssetIDs,
	}
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

func trailTrackPoints(app core.App, trail *core.Record) ([]pluginAssetTrackPoint, time.Time, time.Time, error) {
	gpxName := trail.GetString("gpx")
	if gpxName == "" {
		return nil, time.Time{}, time.Time{}, nil
	}
	fsys, err := app.NewFilesystem()
	if err != nil {
		return nil, time.Time{}, time.Time{}, err
	}
	defer fsys.Close()
	reader, err := fsys.GetReader(trail.BaseFilesPath() + "/" + gpxName)
	if err != nil {
		return nil, time.Time{}, time.Time{}, err
	}
	defer reader.Close()
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, time.Time{}, time.Time{}, err
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

func importAssetPluginPhotos(e *core.RequestEvent, plugin pluginsystem.LocalPlugin, instance *core.Record, auth map[string]any, config map[string]any, output pluginAssetLibraryOutput, data pluginAssetLibraryRequest, waypointID string) error {
	results, err := importAssetPluginPhotosForTrail(e.Request.Context(), e.App, e.Auth.Id, plugin, auth, config, output, data, waypointID, false)
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, results)
}

func importAssetPluginPhotosToTarget(e *core.RequestEvent, plugin pluginsystem.LocalPlugin, auth map[string]any, config map[string]any, output pluginAssetLibraryOutput, data pluginAssetLibraryRequest) error {
	if len(output.Photos) == 0 {
		return e.JSON(http.StatusOK, []pluginAssetImportResult{})
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
	return e.JSON(http.StatusOK, results)
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
		resolvedName, _ := assetPluginWaypointName(ctx, app.Logger(), cluster.Lat, cluster.Lon, mergeSettings.Radius)
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
	actor, err := app.FindRecordById("activitypub_actors", trail.GetString("author"))
	if err != nil {
		return false, err
	}
	return actor.GetString("user") == userID, nil
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
