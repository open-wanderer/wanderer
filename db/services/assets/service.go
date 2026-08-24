package assets

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"

	"pocketbase/plugins/importer"
	"pocketbase/pluginsystem"
	"pocketbase/services/pluginhost"
	"pocketbase/util"
)

type RemotePluginAssetsSummary struct {
	Count             int `json:"count"`
	PublicCount       int `json:"publicCount"`
	MissingCount      int `json:"missingCount"`
	InaccessibleCount int `json:"inaccessibleCount"`
}

type ProgressFunc func(total, processed, failed int)

func RemotePluginAssetsSummaryForUser(app core.App, userID string, pluginID string) (RemotePluginAssetsSummary, error) {
	assets, err := remotePluginAssetsForUser(app, userID, pluginID)
	if err != nil {
		return RemotePluginAssetsSummary{}, err
	}
	summary := RemotePluginAssetsSummary{Count: len(assets)}
	for _, asset := range assets {
		if util.IsAssetLinkedToPublicTrail(app, asset) {
			summary.PublicCount++
		}
		switch asset.GetString("remote_status") {
		case "missing":
			summary.MissingCount++
		case "inaccessible":
			summary.InaccessibleCount++
		}
	}
	return summary, nil
}

func MaterializeRemotePluginAssetsForUser(ctx context.Context, app core.App, userID string, pluginID string, publicOnly bool, progress ProgressFunc) error {
	assets, err := remotePluginAssetsForUser(app, userID, pluginID)
	if err != nil {
		return err
	}
	if publicOnly {
		filtered := make([]*core.Record, 0, len(assets))
		for _, asset := range assets {
			if util.IsAssetLinkedToPublicTrail(app, asset) {
				filtered = append(filtered, asset)
			}
		}
		assets = filtered
	}
	return processRemotePluginAssets(ctx, app, pluginID, assets, progress, func(asset *core.Record) error {
		return MaterializeRemotePluginAsset(ctx, app, asset)
	})
}

func RepairRemotePluginAssetsForUser(ctx context.Context, app core.App, userID string, pluginID string, progress ProgressFunc) error {
	assets, err := remotePluginAssetsForUser(app, userID, pluginID)
	if err != nil {
		return err
	}
	filtered := make([]*core.Record, 0, len(assets))
	for _, asset := range assets {
		switch asset.GetString("remote_status") {
		case "missing", "inaccessible":
			filtered = append(filtered, asset)
		}
	}
	total := len(filtered)
	processed := 0
	failed := 0
	reportProgress(progress, total, processed, failed)
	for _, asset := range filtered {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := FetchRemotePluginAsset(ctx, app, asset, util.DefaultPluginMediaMaxBytes); err != nil {
			failed++
			if markErr := MarkAssetRemoteStatus(app, asset, RemoteStatusForError(err), err); markErr != nil {
				app.Logger().Warn("failed to update remote asset status", "asset", asset.Id, "error", markErr)
			}
		} else if err := MarkAssetRemoteStatus(app, asset, "available", nil); err != nil {
			failed++
			app.Logger().Warn("failed to update remote asset status", "asset", asset.Id, "error", err)
		}
		processed++
		reportProgress(progress, total, processed, failed)
	}
	return nil
}

func DeleteRemotePluginAssetsForUser(ctx context.Context, app core.App, userID string, pluginID string, progress ProgressFunc) error {
	assets, err := remotePluginAssetsForUser(app, userID, pluginID)
	if err != nil {
		return err
	}
	total := len(assets)
	processed := 0
	failed := 0
	reportProgress(progress, total, processed, failed)
	for _, asset := range assets {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := app.Delete(asset); err != nil {
			failed++
			app.Logger().Warn("failed to delete remote asset", "asset", asset.Id, "plugin", pluginID, "error", err)
		}
		processed++
		reportProgress(progress, total, processed, failed)
	}
	return nil
}

func MaterializePrivateRemotePluginAssetsForTrail(ctx context.Context, app core.App, trailID string) error {
	if trailID == "" {
		return nil
	}
	trail, err := app.FindRecordById("trails", trailID)
	if err != nil {
		return err
	}
	assetIDs, err := util.AssetIDsForTrail(app, trailID)
	if err != nil {
		return err
	}
	assets, err := privateRemotePluginPhotoAssetsByIDs(app, trail.GetString("author"), assetIDs)
	if err != nil {
		return err
	}
	for _, asset := range assets {
		if err := MaterializeRemotePluginAsset(ctx, app, asset); err != nil {
			return fmt.Errorf("materialize remote asset %s: %w", asset.Id, err)
		}
	}
	return nil
}

func EnsurePublicTrailSafeAssetLink(ctx context.Context, app core.App, collectionName string, targetField string, targetID string, assetID string) (*core.Record, error) {
	if err := MaterializePrivateRemotePluginAssetForPublicLink(ctx, app, targetField, targetID, assetID); err != nil {
		return nil, err
	}
	return util.EnsureAssetLink(app, collectionName, targetField, targetID, assetID)
}

func MaterializePrivateRemotePluginAssetForPublicLink(ctx context.Context, app core.App, targetField string, targetID string, assetID string) error {
	trailID, err := util.TrailIDForLinkTarget(app, targetField, targetID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	if trailID == "" {
		return nil
	}
	trail, err := app.FindRecordById("trails", trailID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	if !trail.GetBool("public") {
		return nil
	}
	return MaterializePrivateRemotePluginAsset(ctx, app, assetID)
}

func IsRemotePhotoStorageMode(mode string) bool {
	return mode == "link_private"
}

// MaterializePrivateRemotePluginAsset downloads and stores the given asset's
// remote media if it is still a private remote plugin photo. It is a no-op for
// assets that are already materialized or are not remote photos.
func MaterializePrivateRemotePluginAsset(ctx context.Context, app core.App, assetID string) error {
	if assetID == "" {
		return nil
	}
	asset, err := app.FindRecordById("assets", assetID)
	if err != nil {
		return err
	}
	if asset.GetString("type") != "photo" || asset.GetString("storage_mode") != "link_private" {
		return nil
	}
	return MaterializeRemotePluginAsset(ctx, app, asset)
}

func MaterializeRemotePluginAsset(ctx context.Context, app core.App, asset *core.Record) error {
	fetched, err := FetchRemotePluginAsset(ctx, app, asset, util.DefaultPluginMediaMaxBytes)
	if err != nil {
		return err
	}
	file, err := filesystem.NewFileFromBytes(fetched.Body, remoteAssetFileName(asset, fetched.ContentType))
	if err != nil {
		return err
	}
	asset.Set("file", file)
	asset.Set("storage_mode", "copy")
	return MarkAssetRemoteStatus(app, asset, "available", nil)
}

func FetchRemotePluginAsset(ctx context.Context, app core.App, asset *core.Record, maxBytes int64) (*util.SafeFetchResult, error) {
	remote, err := RemotePhotoAssetFromRecord(asset)
	if err != nil {
		return nil, err
	}
	pluginID := remote.PluginID
	if pluginID == "" {
		pluginID = asset.GetString("external_provider")
	}
	if pluginID == "" {
		return nil, fmt.Errorf("remote asset has no plugin id")
	}
	userID, err := util.AssetAuthorUserID(app, asset)
	if err != nil {
		return nil, fmt.Errorf("remote asset author is not a local plugin user: %w", err)
	}
	instance, err := app.FindFirstRecordByFilter(
		"plugin_instances",
		"user={:user} && plugin_id={:plugin_id} && enabled=true",
		dbx.Params{"user": userID, "plugin_id": pluginID},
	)
	if err != nil {
		return nil, fmt.Errorf("remote asset plugin instance is not enabled: %w", err)
	}
	plugin, err := pluginhost.LocalPlugin(app, pluginID)
	if err != nil {
		return nil, err
	}
	if plugin.Manifest.Type != pluginsystem.PluginTypeAssets {
		return nil, fmt.Errorf("remote asset plugin %q is not an asset plugin", pluginID)
	}
	auth, err := pluginhost.DecryptedInstanceAuth(instance)
	if err != nil {
		return nil, err
	}
	config := pluginhost.EffectiveConfig(app, plugin.Manifest.ID, instance)
	config = pluginhost.AssetConnectorConfig(plugin, config)
	photo := pluginsystem.Photo{
		ExternalID:  asset.GetString("external_id"),
		Filename:    remote.Filename,
		ContentType: remote.ContentType,
		Source:      remote.Source,
	}
	return importer.FetchPhotoMedia(ctx, photo, importer.Options{
		UserID:   userID,
		ActorID:  asset.GetString("author"),
		Manifest: plugin.Manifest,
		Policy:   pluginhost.InstancePolicy(plugin, config).WithHostAuth(auth),
		Auth:     auth,
	}, maxBytes)
}

func MarkAssetRemoteStatus(app core.App, asset *core.Record, status string, cause error) error {
	asset.Set("remote_status", status)
	asset.Set("remote_checked_at", time.Now().UTC())
	if cause != nil {
		asset.Set("remote_error", cause.Error())
	} else {
		asset.Set("remote_error", "")
	}
	if status == "missing" {
		if asset.GetString("remote_missing_since") == "" {
			asset.Set("remote_missing_since", time.Now().UTC())
		}
	} else {
		asset.Set("remote_missing_since", "")
	}
	return app.Save(asset)
}

func RemoteStatusForError(err error) string {
	var statusErr util.HTTPStatusError
	if errors.As(err, &statusErr) && (statusErr.StatusCode == http.StatusNotFound || statusErr.StatusCode == http.StatusGone) {
		return "missing"
	}
	return "inaccessible"
}

func processRemotePluginAssets(ctx context.Context, app core.App, pluginID string, assets []*core.Record, progress ProgressFunc, process func(*core.Record) error) error {
	total := len(assets)
	processed := 0
	failed := 0
	reportProgress(progress, total, processed, failed)
	for _, asset := range assets {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := process(asset); err != nil {
			failed++
			if markErr := MarkAssetRemoteStatus(app, asset, RemoteStatusForError(err), err); markErr != nil {
				app.Logger().Warn("failed to update remote asset status", "asset", asset.Id, "error", markErr)
			}
			app.Logger().Warn("failed to materialize remote asset", "asset", asset.Id, "plugin", pluginID, "error", err)
		}
		processed++
		reportProgress(progress, total, processed, failed)
	}
	return nil
}

func reportProgress(progress ProgressFunc, total int, processed int, failed int) {
	if progress != nil {
		progress(total, processed, failed)
	}
}

func remotePluginAssetsForUser(app core.App, userID string, pluginID string) ([]*core.Record, error) {
	if userID == "" || pluginID == "" {
		return []*core.Record{}, nil
	}
	actorID, err := util.ResolveAssetAuthor(app, userID)
	if err != nil {
		return nil, err
	}
	if actorID == "" {
		return []*core.Record{}, nil
	}
	return app.FindRecordsByFilter(
		"assets",
		"author={:author} && external_provider={:plugin} && type='photo' && storage_mode='link_private'",
		"created",
		-1,
		0,
		dbx.Params{"author": actorID, "plugin": pluginID},
	)
}

func privateRemotePluginPhotoAssetsByIDs(app core.App, actorID string, assetIDs []string) ([]*core.Record, error) {
	if actorID == "" || len(assetIDs) == 0 {
		return []*core.Record{}, nil
	}

	const chunkSize = 50
	records := make([]*core.Record, 0, len(assetIDs))
	for start := 0; start < len(assetIDs); start += chunkSize {
		end := start + chunkSize
		if end > len(assetIDs) {
			end = len(assetIDs)
		}
		params := dbx.Params{"author": actorID}
		filters := make([]string, 0, end-start)
		for i, assetID := range assetIDs[start:end] {
			key := fmt.Sprintf("asset_%d", i)
			params[key] = assetID
			filters = append(filters, "id={:"+key+"}")
		}
		found, err := app.FindRecordsByFilter(
			"assets",
			"author={:author} && type='photo' && storage_mode='link_private' && ("+strings.Join(filters, " || ")+")",
			"created",
			-1,
			0,
			params,
		)
		if err != nil {
			return nil, err
		}
		records = append(records, found...)
	}
	return records, nil
}

func RemotePhotoAssetFromRecord(asset *core.Record) (pluginsystem.RemotePhotoAsset, error) {
	raw := asset.Get("metadata")
	rawBytes, err := json.Marshal(raw)
	if err != nil {
		return pluginsystem.RemotePhotoAsset{}, err
	}
	var metadata map[string]json.RawMessage
	if err := json.Unmarshal(rawBytes, &metadata); err != nil {
		return pluginsystem.RemotePhotoAsset{}, err
	}
	rawRemote := metadata["remote"]
	if len(rawRemote) == 0 {
		return pluginsystem.RemotePhotoAsset{}, fmt.Errorf("remote asset metadata is missing")
	}
	var remote pluginsystem.RemotePhotoAsset
	if err := json.Unmarshal(rawRemote, &remote); err != nil {
		return pluginsystem.RemotePhotoAsset{}, err
	}
	if remote.Source.Type == "" {
		return pluginsystem.RemotePhotoAsset{}, fmt.Errorf("remote asset source is missing")
	}
	return remote, nil
}

func remoteAssetFileName(asset *core.Record, contentType string) string {
	name := strings.TrimSpace(asset.GetString("external_id"))
	if name == "" {
		name = asset.Id
	}
	name = filepath.Base(name)
	if filepath.Ext(name) != "" {
		return name
	}
	if extensions, err := mime.ExtensionsByType(contentType); err == nil && len(extensions) > 0 {
		return name + extensions[0]
	}
	return name + ".bin"
}
