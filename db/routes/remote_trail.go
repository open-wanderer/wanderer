package routes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"pocketbase/federation"
	"pocketbase/util"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

// remoteSyncThreshold is the minimum age of a remote record before a background sync is triggered.
// Configurable via POCKETBASE_FEDERATION_SYNC_INTERVAL (minutes). Default: 60.
var remoteSyncThreshold = func() time.Duration {
	if v := os.Getenv("POCKETBASE_FEDERATION_SYNC_INTERVAL"); v != "" {
		if minutes, err := strconv.Atoi(v); err == nil && minutes > 0 {
			return time.Duration(minutes) * time.Minute
		}
	}
	return 60 * time.Minute
}()

// trailSyncing and listSyncing track IRIs currently being synced to prevent concurrent duplicate syncs.
var trailSyncing sync.Map
var listSyncing sync.Map

var remoteTrailSyncCoreExpandPaths = []string{
	"category",
	"subcategory",
	"waypoints_via_trail",
	"summit_logs_via_trail",
	"summit_logs_via_trail.author",
}

var remoteTrailSyncAssetExpandPaths = []string{
	"trail_assets_via_trail.asset",
	"waypoints_via_trail.waypoint_assets_via_waypoint.asset",
	"summit_logs_via_trail.summit_log_assets_via_summit_log.asset",
}

// --- Main Handler ---

func RemoteTrailGet(e *core.RequestEvent) error {
	handle := e.Request.URL.Query().Get("handle")
	trailID := e.Request.PathValue("id")
	expandQuery := e.Request.URL.Query().Get("expand")

	var record *core.Record
	var err error

	var userActor *core.Record
	if e.Auth != nil {
		userActor, _ = e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
	}

	ctx, err := util.GetSafeActorContext(e.Request, userActor)
	if err != nil {
		return err
	}

	// 1. Resolve the "Actual" Record or Shell
	if handle != "" {
		// If we have a handle, we are looking for a remote trail.
		// Construct the IRI first to see if we already know this trail.
		record, err = findLocalTrailByRemoteInfo(e, ctx, handle, trailID)
		if err != nil {
			if errors.Is(err, federation.ErrProfilePrivate) {
				return e.NotFoundError("profile is private", err)
			}
			return e.InternalServerError("Failed to resolve trail", err)
		}

		// If the record has no ID, it's a new Shell
		if record.Id == "" || record.GetBool("needs_full_sync") {
			// Blocking sync for new records
			record, err = performFullSync(e.App, ctx, e.Request.URL, record)
			if err != nil {
				if errors.Is(err, util.ErrRateLimited) {
					return e.TooManyRequestsError("Too many requests", err)
				}
				return e.InternalServerError("Sync failed", err)
			}
			if record.Id == "" {
				// Local content that does not exist (e.g. a stale URL to a
				// missing local trail): performFullSync short-circuits local
				// IRIs and returns the unsaved shell — surface a real 404
				// instead of running access/expand on a non-existent record.
				return e.NotFoundError("Trail not found", nil)
			}
		} else {
			// We already have it locally. Show and update background.
			updatedAt := record.GetDateTime("updated").Time()

			iri := record.GetString("iri")
			if time.Now().UTC().Sub(updatedAt) > remoteSyncThreshold {
				if _, alreadySyncing := trailSyncing.LoadOrStore(iri, struct{}{}); !alreadySyncing {
					urlCopy := *e.Request.URL
					bgCtx := context.WithValue(context.Background(), "actor", ctx.Value("actor"))
					go func() {
						defer trailSyncing.Delete(iri)
						performFullSync(e.App, bgCtx, &urlCopy, record)
					}()
				}
			}
		}
	} else {
		// Standard local fetch by ID
		record, err = e.App.FindRecordById("trails", trailID)
		if err != nil {
			return e.NotFoundError("Trail not found", nil)
		}
	}

	reqInfo, err := e.RequestInfo()
	if err != nil {
		return err
	}

	canAccess, err := e.App.CanAccessRecord(record, reqInfo, record.Collection().ViewRule)

	if err != nil || !canAccess {
		return e.ForbiddenError("forbidden", err)
	}

	return expandAndReturn(e, record, expandQuery)
}

func findLocalTrailByRemoteInfo(e *core.RequestEvent, ctx context.Context, handle, trailID string) (*core.Record, error) {
	// 1. Get Actor to build the IRI
	actor, err := federation.GetActorByHandle(e.App, ctx, handle, false)
	if err != nil && !errors.Is(err, federation.ErrProfilePrivate) {
		return nil, err
	}
	if actor == nil {
		return nil, err
	}

	actorURL, _ := url.Parse(actor.GetString("iri"))
	iri := fmt.Sprintf("%s://%s/api/v1/trail/%s", actorURL.Scheme, actorURL.Host, trailID)

	// 2. Check if this IRI already exists in our DB
	existing, _ := e.App.FindFirstRecordByFilter("trails", "iri={:iri}||id={:id}", dbx.Params{"id": trailID, "iri": iri})
	if existing != nil {
		return existing, nil
	}

	// 3. Not found? Return a new Shell
	collection, _ := e.App.FindCollectionByNameOrId("trails")
	shell := core.NewRecord(collection)
	shell.Set("iri", iri)
	shell.Set("author", actor.Id)
	shell.Set("like_count", 0)

	return shell, nil
}

// --- Core Sync Logic ---

func performFullSync(app core.App, ctx context.Context, reqURL *url.URL, localTrail *core.Record) (*core.Record, error) {
	iri := localTrail.GetString("iri")

	// Never federate with ourselves: a trail whose IRI is empty or points back
	// to this instance is local content (we are the source of truth). Syncing it
	// would fetch our own origin (or fail on an empty URL); just clear the stale
	// flag so the record is no longer stuck in a permanent re-sync loop.
	if iri == "" || util.IsLocalIRI(iri) {
		if localTrail.GetBool("needs_full_sync") {
			localTrail.Set("needs_full_sync", false)
			if err := app.Save(localTrail); err != nil {
				return localTrail, err
			}
		}
		return localTrail, nil
	}

	client := util.SafeHTTPClient()
	remoteUrl, _ := url.Parse(iri)
	origin := fmt.Sprintf("%s://%s", remoteUrl.Scheme, remoteUrl.Host)

	assetExpandsRequested := true
	remoteUrl.RawQuery = remoteTrailSyncQuery(reqURL, true).Encode()
	remoteMap, err := fetchRemoteJSONMap(ctx, client, remoteUrl, "trail")
	if shouldRetryRemoteFetchWithoutAssetExpands(err) {
		legacyURL := *remoteUrl
		legacyURL.RawQuery = remoteTrailSyncQuery(reqURL, false).Encode()
		if legacyURL.RawQuery != remoteUrl.RawQuery {
			if fallbackMap, fallbackErr := fetchRemoteJSONMap(ctx, client, &legacyURL, "trail"); fallbackErr == nil {
				remoteMap = fallbackMap
				assetExpandsRequested = false
				err = nil
			}
		}
	}
	if err != nil {
		return localTrail, err
	}

	err = app.RunInTransaction(func(txApp core.App) error {
		remoteID, _ := remoteMap["id"].(string)

		// 1. Sync Files
		syncRecordFiles(ctx, localTrail, "trails", remoteID, origin, remoteMap)

		// 2. Map Relations & Simple Fields
		syncTrailMetadata(txApp, localTrail, remoteMap)

		localTrail.Set("needs_full_sync", false)

		if err := txApp.Save(localTrail); err != nil {
			return err
		}

		// Materialize federated photos into the asset model. Done after the save
		// so the trail has an ID to link the asset against.
		if err := syncRecordPhotos(txApp, ctx, "trail", localTrail.Id, remoteID, localTrail.GetString("author"), origin, remoteMap, assetExpandsRequested); err != nil {
			return err
		}

		// 3. Sync Waypoints
		if expand, ok := remoteMap["expand"].(map[string]any); ok {
			if wps, ok := expand["waypoints_via_trail"].([]any); ok {
				err = syncWaypoints(txApp, ctx, localTrail, origin, wps, assetExpandsRequested)
				if err != nil {
					return err
				}
			}
		}

		// 3. Sync SummitLogs
		if expand, ok := remoteMap["expand"].(map[string]any); ok {
			if sls, ok := expand["summit_logs_via_trail"].([]any); ok {
				err = syncSummitLogs(txApp, ctx, localTrail, origin, sls, assetExpandsRequested)
				if err != nil {
					return err
				}
			}
		}

		return nil
	})

	return localTrail, err
}

func fetchRemoteJSONMap(ctx context.Context, client *http.Client, remoteURL *url.URL, kind string) (map[string]any, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", remoteURL.String(), nil)
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, &remoteFetchStatusError{
			Kind:       kind,
			URL:        remoteURL.String(),
			StatusCode: res.StatusCode,
		}
	}

	var data map[string]any
	if err := json.NewDecoder(res.Body).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

type remoteFetchStatusError struct {
	Kind       string
	URL        string
	StatusCode int
}

func (err *remoteFetchStatusError) Error() string {
	return fmt.Sprintf("remote %s fetch %s returned: %d", err.Kind, err.URL, err.StatusCode)
}

func shouldRetryRemoteFetchWithoutAssetExpands(err error) bool {
	var statusErr *remoteFetchStatusError
	if !errors.As(err, &statusErr) {
		return false
	}

	switch statusErr.StatusCode {
	case http.StatusBadRequest,
		http.StatusRequestURITooLong,
		http.StatusRequestHeaderFieldsTooLarge,
		http.StatusUnprocessableEntity:
		return true
	default:
		return false
	}
}

func remoteTrailSyncQuery(reqURL *url.URL, includeAssetExpands bool) url.Values {
	query := url.Values{}
	if reqURL != nil {
		for key, values := range reqURL.Query() {
			if key == "handle" {
				continue
			}
			for _, value := range values {
				query.Add(key, value)
			}
		}
	}

	required := append([]string{}, remoteTrailSyncCoreExpandPaths...)
	if includeAssetExpands {
		required = append(required, remoteTrailSyncAssetExpandPaths...)
	} else {
		removeExpandQueryPaths(query, remoteTrailSyncAssetExpandPaths)
	}
	setMergedExpandQuery(query, required)
	return query
}

func removeExpandQueryPaths(query url.Values, blocked []string) {
	blockedSet := map[string]struct{}{}
	for _, path := range blocked {
		blockedSet[path] = struct{}{}
	}

	kept := []string{}
	for _, value := range query["expand"] {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if _, blocked := blockedSet[part]; blocked {
				continue
			}
			kept = append(kept, part)
		}
	}
	if len(kept) == 0 {
		query.Del("expand")
		return
	}
	query.Set("expand", strings.Join(kept, ","))
}

func setMergedExpandQuery(query url.Values, required []string) {
	seen := map[string]struct{}{}
	expands := make([]string, 0, len(required))
	add := func(value string) {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if _, ok := seen[part]; ok {
				continue
			}
			seen[part] = struct{}{}
			expands = append(expands, part)
		}
	}

	for _, value := range query["expand"] {
		add(value)
	}
	for _, value := range required {
		add(value)
	}

	if len(expands) == 0 {
		query.Del("expand")
		return
	}
	query.Set("expand", strings.Join(expands, ","))
}

// --- Sub-Sync Helpers ---

func syncTrailMetadata(app core.App, record *core.Record, data map[string]any) {
	var federatedCategoryName, federatedSubcategoryName string

	if expand, ok := data["expand"].(map[string]any); ok {
		if cat, ok := expand["category"].(map[string]any); ok {
			if name, ok := cat["name"].(string); ok {
				federatedCategoryName = name
			}
		}
		if subcat, ok := expand["subcategory"].(map[string]any); ok {
			if name, ok := subcat["name"].(string); ok {
				federatedSubcategoryName = name
			}
		}
	}

	if federatedCategoryName != "" {
		record.Set("federated_category_name", federatedCategoryName)
	}
	if federatedSubcategoryName != "" {
		record.Set("federated_subcategory_name", federatedSubcategoryName)
	}

	category, subcategory, err := util.ResolveCategoryAndSubcategoryByNormalizedNames(app, federatedCategoryName, federatedSubcategoryName)
	if err == nil && category != nil {
		record.Set("category", category.Id)
		if subcategory != nil {
			record.Set("subcategory", subcategory.Id)
		} else {
			record.Set("subcategory", "")
		}
	} else if err == nil && federatedCategoryName != "" {
		record.Set("category", "")
		record.Set("subcategory", "")
	}

	// Resolve Tags
	localTagIds := resolveAndSyncTags(app, data)
	if len(localTagIds) > 0 {
		record.Set("tags", localTagIds)
	}

	// Clean protected/complex fields before bulk load
	delete(data, "id")
	delete(data, "gpx")
	delete(data, "author")
	delete(data, "category")
	delete(data, "subcategory")
	delete(data, "tags")
	delete(data, "iri")
	delete(data, "federated_category_name")
	delete(data, "federated_subcategory_name")

	record.Load(data)
}

func resolveAndSyncTags(app core.App, data map[string]any) []string {
	var localTagIds []string

	expand, ok := data["expand"].(map[string]any)
	if !ok {
		return localTagIds
	}

	remoteTags, ok := expand["tags"].([]any)
	if !ok {
		return localTagIds
	}

	tagCol, _ := app.FindCollectionByNameOrId("tags")

	for _, t := range remoteTags {
		tagMap, ok := t.(map[string]any)
		if !ok {
			continue
		}

		tagName, _ := tagMap["name"].(string)
		if tagName == "" {
			continue
		}

		localTag, _ := app.FindFirstRecordByData("tags", "name", tagName)

		if localTag == nil {
			localTag = core.NewRecord(tagCol)
			localTag.Set("name", tagName)

			if err := app.Save(localTag); err != nil {
				continue
			}
		}

		localTagIds = append(localTagIds, localTag.Id)
	}

	return localTagIds
}

func syncWaypoints(txApp core.App, ctx context.Context, trail *core.Record, origin string, waypoints []any, assetExpandsRequested bool) error {
	col, _ := txApp.FindCollectionByNameOrId("waypoints")

	for _, wData := range waypoints {
		raw := wData.(map[string]any)
		wpID, _ := raw["id"].(string)
		iri, _ := raw["iri"].(string)
		if iri == "" {
			iri = fmt.Sprintf("%s/api/v1/waypoint/%s", origin, wpID)
		}

		wp, _ := txApp.FindFirstRecordByData("waypoints", "iri", iri)
		if wp == nil {
			wp = core.NewRecord(col)
		}

		syncRecordFiles(ctx, wp, "waypoints", wpID, origin, raw)

		delete(raw, "id")
		wp.Load(raw)
		wp.Set("author", trail.GetString("author"))
		wp.Set("trail", trail.Id)
		wp.Set("iri", iri)

		if err := txApp.Save(wp); err != nil {
			return err
		}

		if err := syncRecordPhotos(txApp, ctx, "waypoint", wp.Id, wpID, wp.GetString("author"), origin, raw, assetExpandsRequested); err != nil {
			return err
		}
	}
	return nil
}

func syncSummitLogs(txApp core.App, ctx context.Context, trail *core.Record, origin string, summitLogs []any, assetExpandsRequested bool) error {
	col, _ := txApp.FindCollectionByNameOrId("summit_logs")

	for _, slData := range summitLogs {
		raw := slData.(map[string]any)
		slID, _ := raw["id"].(string)
		iri, _ := raw["iri"].(string)
		if iri == "" {
			iri = fmt.Sprintf("%s/api/v1/summit-log/%s", origin, slID)
		}

		sl, _ := txApp.FindFirstRecordByData("summit_logs", "iri", iri)
		if sl == nil {
			sl = core.NewRecord(col)
		}

		author := trail.GetString("author")
		if expand, ok := raw["expand"].(map[string]any); ok {
			if authorMap, ok := expand["author"].(map[string]any); ok {
				actor, err := federation.GetActorByIRI(txApp, ctx, authorMap["iri"].(string), false)
				if err != nil {
					return err
				}
				author = actor.Id
			}
		}

		syncRecordFiles(ctx, sl, "summit_logs", slID, origin, raw)

		delete(raw, "id")
		delete(raw, "gpx")

		sl.Load(raw)
		sl.Set("author", author)
		sl.Set("trail", trail.Id)
		sl.Set("iri", iri)

		if err := txApp.Save(sl); err != nil {
			return err
		}

		if err := syncRecordPhotos(txApp, ctx, "summit_log", sl.Id, slID, sl.GetString("author"), origin, raw, assetExpandsRequested); err != nil {
			return err
		}
	}
	return nil
}

func syncRecordFiles(ctx context.Context, record *core.Record, collection, remoteID, origin string, data map[string]any) {
	// Handle GPX
	if gpx, ok := data["gpx"].(string); ok && gpx != "" && record.GetString("gpx") == "" {
		if f, err := downloadFile(ctx, origin, collection, remoteID, gpx); err == nil {
			record.Set("gpx", f)
		}
	}
}

// syncRecordPhotos materializes federated photos for a synced target
// (trail/waypoint/summit_log) into the local asset model. Photos travel in the
// remote record's expand (e.g. trail_assets_via_trail[].expand.asset) and are
// reconciled against the target by their canonical origin identity: new photos
// are downloaded size-bounded via util.FetchPublicFile, known ones are kept,
// and photos the origin no longer lists are unlinked again.
func syncRecordPhotos(app core.App, ctx context.Context, targetField, targetID, remoteID, author, origin string, data map[string]any, assetExpandsRequested bool) error {
	if targetID == "" || author == "" {
		return nil
	}

	photos, authoritative := syncRecordPhotoList(targetField, remoteID, origin, data, assetExpandsRequested)
	if !authoritative {
		return nil
	}
	return util.ReconcileFederatedPhotoAssets(app, ctx, targetField, targetID, author, photos)
}

func syncRecordPhotoList(targetField, remoteID, origin string, data map[string]any, assetExpandsRequested bool) ([]util.FederatedPhoto, bool) {
	photos := []util.FederatedPhoto{}
	for _, link := range remotePhotoAssetLinks(data, targetField) {
		asset := link.Asset
		fileURL := remoteAssetFileURL(origin, asset)
		canonicalID := remoteAssetCanonicalID(origin, asset)
		if fileURL == "" || canonicalID == "" {
			continue
		}
		photo := util.FederatedPhoto{
			CanonicalID: canonicalID,
			FileURL:     fileURL,
			IsThumbnail: link.IsThumbnail,
		}
		if lat, ok := asset["lat"].(float64); ok && lat != 0 {
			photo.Lat, photo.HasLat = lat, true
		}
		if lon, ok := asset["lon"].(float64); ok && lon != 0 {
			photo.Lon, photo.HasLon = lon, true
		}
		photos = append(photos, photo)
	}
	if len(photos) > 0 {
		return photos, true
	}
	if !assetExpandsRequested && !hasLegacyPhotosField(data) {
		return nil, false
	}
	return legacyRecordPhotos(origin, targetField, remoteID, data), true
}

func hasLegacyPhotosField(data map[string]any) bool {
	_, ok := data["photos"]
	return ok
}

type remotePhotoAssetLink struct {
	Asset       map[string]any
	IsThumbnail bool
}

// remotePhotoAssetLinks extracts expanded photo asset records that travel with a
// synced target through its asset-link expand (…_assets_via_…[].expand.asset).
func remotePhotoAssetLinks(data map[string]any, targetField string) []remotePhotoAssetLink {
	var linkKey string
	switch targetField {
	case "trail":
		linkKey = "trail_assets_via_trail"
	case "waypoint":
		linkKey = "waypoint_assets_via_waypoint"
	case "summit_log":
		linkKey = "summit_log_assets_via_summit_log"
	default:
		return nil
	}

	expand, ok := data["expand"].(map[string]any)
	if !ok {
		return nil
	}
	links, ok := expand[linkKey].([]any)
	if !ok {
		return nil
	}

	assetLinks := make([]remotePhotoAssetLink, 0, len(links))
	for _, l := range links {
		linkMap, ok := l.(map[string]any)
		if !ok {
			continue
		}
		linkExpand, ok := linkMap["expand"].(map[string]any)
		if !ok {
			continue
		}
		asset, ok := linkExpand["asset"].(map[string]any)
		if !ok {
			continue
		}
		if t, _ := asset["type"].(string); t != "photo" {
			continue
		}
		assetLinks = append(assetLinks, remotePhotoAssetLink{
			Asset:       asset,
			IsThumbnail: boolValue(linkMap["is_thumbnail"]),
		})
	}
	return assetLinks
}

func legacyRecordPhotos(origin string, targetField string, remoteID string, data map[string]any) []util.FederatedPhoto {
	collection := legacyPhotoCollection(targetField)
	if collection == "" || remoteID == "" {
		return nil
	}
	rawPhotos, ok := data["photos"].([]any)
	if !ok {
		return nil
	}

	thumbnailIndex := -1
	if targetField == "trail" {
		if rawIndex, ok := data["thumbnail"].(float64); ok {
			thumbnailIndex = int(rawIndex)
		}
	}

	photos := make([]util.FederatedPhoto, 0, len(rawPhotos))
	for i, rawPhoto := range rawPhotos {
		photo, ok := rawPhoto.(string)
		if !ok || photo == "" {
			continue
		}
		fileURL := legacyPhotoURL(origin, collection, remoteID, photo)
		photos = append(photos, util.FederatedPhoto{
			CanonicalID: fileURL,
			FileURL:     fileURL,
			IsThumbnail: i == thumbnailIndex,
		})
	}
	return photos
}

func legacyPhotoCollection(targetField string) string {
	switch targetField {
	case "trail":
		return "trails"
	case "waypoint":
		return "waypoints"
	case "summit_log":
		return "summit_logs"
	default:
		return ""
	}
}

func legacyPhotoURL(origin string, collection string, remoteID string, photo string) string {
	if strings.HasPrefix(photo, "http://") || strings.HasPrefix(photo, "https://") {
		return photo
	}
	if strings.HasPrefix(photo, "/") {
		return strings.TrimRight(origin, "/") + photo
	}
	return fmt.Sprintf("%s/api/v1/files/%s/%s/%s", strings.TrimRight(origin, "/"), collection, remoteID, url.PathEscape(photo))
}

func boolValue(value any) bool {
	v, _ := value.(bool)
	return v
}

// remoteAssetCanonicalID mirrors util.CanonicalFederatedAssetID for a remote
// asset map: an asset the origin itself materialized from federation keeps its
// original identity (stable across hops), otherwise the origin's asset IRI
// identifies it.
func remoteAssetCanonicalID(origin string, asset map[string]any) string {
	provider, _ := asset["external_provider"].(string)
	externalID, _ := asset["external_id"].(string)
	if provider == util.FederationAssetProvider && externalID != "" {
		return externalID
	}
	id, _ := asset["id"].(string)
	if id == "" {
		return ""
	}
	return fmt.Sprintf("%s/api/v1/assets/%s", origin, id)
}

// remoteAssetFileURL builds the public media URL for a remote asset map,
// mirroring util.AssetPublicMediaURL: a direct file URL when the asset stores a
// copied file, otherwise the asset file endpoint for remotely-stored assets.
func remoteAssetFileURL(origin string, asset map[string]any) string {
	id, _ := asset["id"].(string)
	if id == "" {
		return ""
	}
	file, _ := asset["file"].(string)
	collectionID, _ := asset["collectionId"].(string)
	if file != "" && collectionID != "" {
		return fmt.Sprintf("%s/api/v1/files/%s/%s/%s", origin, collectionID, id, file)
	}
	return fmt.Sprintf("%s/api/v1/assets/%s/file", origin, id)
}

func downloadFile(ctx context.Context, origin, col, id, name string) (*filesystem.File, error) {
	client := util.SafeHTTPClient()

	url := fmt.Sprintf("%s/api/v1/files/%s/%s/%s", origin, col, id, name)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)

	res, err := client.Do(req)
	if err != nil || res.StatusCode != 200 {
		return nil, fmt.Errorf("download failed")
	}
	defer res.Body.Close()

	data, _ := io.ReadAll(res.Body)
	return filesystem.NewFileFromBytes(data, name)
}

func expandAndReturn(e *core.RequestEvent, record *core.Record, query string) error {
	if query != "" {
		e.App.ExpandRecord(record, strings.Split(query, ","), nil)
	}
	return e.JSON(http.StatusOK, record)
}
