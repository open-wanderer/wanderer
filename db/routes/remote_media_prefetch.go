package routes

import (
	"context"
	"errors"
	"fmt"
	"path"
	"pocketbase/util"
	"sort"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

const (
	remoteSyncBudget             = 60 * time.Second
	remoteMediaFetchBudget       = 30 * time.Second
	remoteMediaMaxFiles          = 100
	remoteMediaMaxBytes    int64 = 64 << 20
	remoteGPXMaxBytes      int64 = 5 << 20
)

var fetchRemoteSyncMedia = util.DownloadRemoteSyncFile

type remoteMediaRequest struct {
	url, origin, name string
	maxBytes          int64
}

type remoteMediaCollector struct {
	requests []remoteMediaRequest
	seen     map[string]bool
}

func (c *remoteMediaCollector) add(request remoteMediaRequest) {
	key := request.url + "\x00" + request.origin
	if len(c.requests) >= remoteMediaMaxFiles || c.seen[key] {
		return
	}
	c.seen[key] = true
	c.requests = append(c.requests, request)
}

func remoteFieldLimit(app core.App, collection, name string, fallback int64) int64 {
	schema, err := app.FindCachedCollectionByNameOrId(collection)
	if err == nil && schema != nil {
		if field, ok := schema.Fields.GetByName(name).(*core.FileField); ok && field.MaxSize > 0 {
			return field.MaxSize
		}
	}
	return fallback
}

func (c *remoteMediaCollector) record(app core.App, target, remoteID, origin, iri string, data map[string]any, requested ...bool) {
	collection := map[string]string{"trail": "trails", "waypoint": "waypoints", "summit_log": "summit_logs", "list": "lists"}[target]
	if !remoteSyncIDFitsCollection(app, collection, remoteID) {
		return
	}
	stored, _ := app.FindFirstRecordByData(collection, "iri", iri)
	if stored != nil && remoteRecordHasLocalAuthor(app, stored) {
		return
	}
	fileField, limit := "gpx", util.RemoteGPXMaxBytes
	if target == "list" {
		fileField, limit = "avatar", util.RemoteAvatarMaxBytes
	}
	if filename, _ := data[fileField].(string); filename != "" && !strings.ContainsAny(filename, "/\\") && (stored == nil || stored.GetString(fileField) == "") {
		c.add(remoteMediaRequest{url: fmt.Sprintf("%s/api/v1/files/%s/%s/%s", origin, collection, remoteID, filename), origin: origin, name: filename, maxBytes: remoteFieldLimit(app, collection, fileField, limit)})
	}
	if target == "list" {
		return
	}
	known := map[string]bool{}
	var existingAssets []*core.Record
	assetAuthor := ""
	if stored != nil {
		assets, err := util.PhotoAssetsForTarget(app, target, stored.Id, -1)
		if err != nil {
			return
		}
		existingAssets = assets
		assetAuthor = stored.GetString("author")
		for _, asset := range assets {
			if asset.GetString("external_provider") == util.FederationAssetProvider && asset.GetString("file") != "" {
				known[asset.GetString("external_id")] = true
			}
		}
	}
	assetsRequested := len(requested) == 0 || requested[0]
	photos, complete := syncRecordPhotoList(target, remoteID, origin, data, assetsRequested)
	if !complete {
		return
	}
	legacyAssets := util.NewFederatedPhotoAssetIndex(existingAssets, assetAuthor)
	for _, photo := range photos {
		// Unlike reconciliation this only sets a boolean; an already known
		// identity needs neither an adoption lookup nor an identity query.
		if known[photo.CanonicalID] {
			continue
		}
		if assetAuthor != "" {
			holder, err := util.FindFederatedPhotoIdentityAsset(app, assetAuthor, photo.CanonicalID)
			if err != nil {
				return
			}
			if holder != nil && holder.GetString("file") != "" {
				known[photo.CanonicalID] = true
			}
		}
		if known[photo.CanonicalID] {
			continue
		}
		if _, asset := legacyAssets.FirstMatch(photo); asset != nil {
			known[photo.CanonicalID] = true
		}
		if !known[photo.CanonicalID] {
			c.add(remoteMediaRequest{url: photo.FileURL, origin: origin, name: path.Base(photo.FileURL), maxBytes: util.FederatedPhotoFileLimit(app)})
		}
	}

}

func collectRemoteTrailMedia(app core.App, trail *core.Record, origin, remoteID string, data map[string]any, requested ...bool) remoteMediaCollector {
	c := remoteMediaCollector{seen: map[string]bool{}}
	c.record(app, "trail", remoteID, origin, trail.GetString("iri"), data, requested...)
	expand, _ := data["expand"].(map[string]any)
	for _, target := range []string{"waypoint", "summit_log"} {
		relation := map[string]string{"waypoint": "waypoints_via_trail", "summit_log": "summit_logs_via_trail"}[target]
		children, _ := expand[relation].([]any)
		for _, value := range children {
			raw, _ := value.(map[string]any)
			id, _ := raw["id"].(string)
			if id == "" {
				continue
			}
			iri, _ := raw["iri"].(string)
			kind := strings.ReplaceAll(target, "_", "-")
			if iri == "" {
				iri = fmt.Sprintf("%s/api/v1/%s/%s", origin, kind, id)
			}
			if util.IsLocalIRI(iri) || !remoteSyncIRIMatchesOrigin(iri, origin) {
				continue
			}
			c.record(app, target, id, origin, iri, raw, requested...)
		}
	}
	return c
}

func prepareRemoteTrailMedia(app core.App, ctx context.Context, trail *core.Record, origin, remoteID string, data map[string]any, canonical map[string]canonicalRemoteSummitLog) (context.Context, func(), error) {
	c := collectRemoteTrailMedia(app, trail, origin, remoteID, data)
	return prepareCollectedRemoteTrailMedia(app, ctx, c, canonical)
}

func prepareCollectedRemoteTrailMedia(app core.App, ctx context.Context, c remoteMediaCollector, canonical map[string]canonicalRemoteSummitLog) (context.Context, func(), error) {
	// Stable order also makes a capped set predictable between syncs.
	keys := make([]string, 0, len(canonical))
	for iri := range canonical {
		keys = append(keys, iri)
	}
	sort.Strings(keys)
	for _, iri := range keys {
		entry := canonical[iri]
		if entry.removed || entry.omitted || entry.data == nil {
			continue
		}
		c.record(app, "summit_log", entry.remoteID, entry.origin, iri, entry.data, entry.assetsRequested)
	}
	for i := range c.requests {
		c.requests[i].url = util.FederatedShareURL(ctx, c.requests[i].url)
	}
	return prepareRemoteMedia(ctx, c.requests)
}

func prepareRemoteListMedia(app core.App, ctx context.Context, list *core.Record, origin, remoteID string, data map[string]any) (context.Context, func(), error) {
	c := remoteMediaCollector{seen: map[string]bool{}}
	c.record(app, "list", remoteID, origin, list.GetString("iri"), data)
	return prepareRemoteMedia(ctx, c.requests)
}

func prepareRemoteMedia(ctx context.Context, requests []remoteMediaRequest) (context.Context, func(), error) {
	return prepareRemoteMediaWithin(ctx, requests, remoteMediaFetchBudget)
}

func prepareRemoteMediaWithin(ctx context.Context, requests []remoteMediaRequest, budget time.Duration) (context.Context, func(), error) {
	fetchCtx, cancel := context.WithTimeout(util.WithFederatedMediaByteBudget(ctx, remoteMediaMaxBytes), budget)
	defer cancel()
	cleanups := []func(){}
	cleanup := func() {
		for _, closeFile := range cleanups {
			closeFile()
		}
	}
	remaining := remoteMediaMaxBytes
	prepared := make([]util.PreparedFederatedFile, 0, len(requests))
	for _, request := range requests {
		if remaining <= 0 {
			break
		}
		limit := min(remaining, request.maxBytes)
		releaseFetch, err := acquireRemoteSyncFetch(fetchCtx, request.url)
		if err != nil {
			break
		}
		file, closeFile, err := fetchRemoteSyncMedia(fetchCtx, request.url, request.origin, request.name, limit)
		releaseFetch()
		if closeFile != nil {
			cleanups = append(cleanups, closeFile)
		}
		if fetchCtx.Err() != nil {
			break
		}
		if errors.Is(err, util.ErrFederatedMediaBudget) {
			break
		}
		if err != nil || file == nil {
			continue
		}
		if file.Size > limit || file.Size < 0 {
			continue
		}
		file.Name = remoteSyncFileName(request.name)
		remaining -= file.Size
		prepared = append(prepared, util.PreparedFederatedFile{URL: request.url, OriginIRI: request.origin, File: file})
	}
	if ctx.Err() != nil {
		cleanup()
		return ctx, func() {}, fmt.Errorf("%w: %w", errRemoteUnavailable, ctx.Err())
	}
	return util.WithPreparedFederatedFiles(ctx, prepared), cleanup, nil
}
