package routes

import (
	"context"
	"errors"
	"fmt"
	"github.com/pocketbase/pocketbase/core"
	"net/http"
	"net/url"
	"pocketbase/services/routeimage"
	"pocketbase/util"
	"strings"
)

// syncRecordPhotos materializes federated photos for a synced target
// (trail/waypoint/summit_log) into the local asset model. Photos travel in the
// remote record's expand (e.g. trail_assets_via_trail[].expand.asset) and are
// reconciled against the target by their canonical origin identity: new photos
// are downloaded size-bounded via util.FetchFederatedFile, known ones are kept,
// and photos the origin no longer lists are unlinked again.
func syncRecordPhotos(app core.App, ctx context.Context, targetField, targetID, remoteID, author, origin string, data map[string]any, assetExpandsRequested bool) error {
	if targetID == "" || author == "" {
		return nil
	}

	photos, authoritative := syncRecordPhotoList(targetField, remoteID, origin, data, assetExpandsRequested)
	if !authoritative {
		return nil
	}
	for i := range photos {
		photos[i].FileURL = util.FederatedShareURL(ctx, photos[i].FileURL)
	}
	expand, _ := data["expand"].(map[string]any)
	_, modern := expand[remotePhotoAssetLinkKey(targetField)]
	return util.ReconcileFederatedPhotoAssets(app, ctx, targetField, targetID, author, photos, !modern && len(photos) > 0)
}

func syncRecordPhotoList(targetField, remoteID, origin string, data map[string]any, assetExpandsRequested bool) ([]util.FederatedPhoto, bool) {
	if strings.TrimSpace(remoteID) == "" {
		return nil, false
	}
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
			SourceName:  remotePhotoSourceName(asset),
			IsThumbnail: link.IsThumbnail,
		}
		if metadata, ok := asset["metadata"].(map[string]any); ok && util.AssetMetadataString(metadata, "generated", "kind") == "route-preview" {
			continue
		}
		if lat, ok := asset["lat"].(float64); ok && lat != 0 {
			photo.Lat, photo.HasLat = lat, true
		}
		if lon, ok := asset["lon"].(float64); ok && lon != 0 {
			photo.Lon, photo.HasLon = lon, true
		}
		photos = append(photos, photo)
	}
	// A missing nested expand can be a denied or failed relation read. Only
	// an explicit complete link list, including [], proves its photo set.
	if expand, ok := data["expand"].(map[string]any); ok {
		if links, ok := expand[remotePhotoAssetLinkKey(targetField)].([]any); ok {
			for _, value := range links {
				link, _ := value.(map[string]any)
				linkExpand, _ := link["expand"].(map[string]any)
				if _, complete := linkExpand["asset"].(map[string]any); !complete {
					return nil, false
				}
			}
			return photos, true
		}
	}
	legacy, ok := data["photos"].([]any)
	if !ok || len(legacy) == 0 {
		if ok && !assetExpandsRequested && targetField == "trail" {
			// A retry after the origin rejected modern asset expansions can
			// still confirm the old trail API's complete legacy file list.
			return photos, true
		}
		return nil, false
	}
	for _, value := range legacy {
		filename, ok := value.(string)
		if !ok || filename == "" || strings.ContainsAny(filename, "/\\") {
			// Enriched URLs are not the original legacy file list.
			return nil, false
		}
	}
	return legacyRecordPhotos(origin, targetField, remoteID, data), true
}

type remotePhotoAssetLink struct {
	Asset       map[string]any
	IsThumbnail bool
}

// remotePhotoAssetLinks extracts expanded photo asset records that travel with a
// synced target through its asset-link expand (…_assets_via_…[].expand.asset).
func remotePhotoAssetLinks(data map[string]any, targetField string) []remotePhotoAssetLink {
	linkKey := remotePhotoAssetLinkKey(targetField)
	if linkKey == "" {
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
		assetLinks = append(assetLinks, remotePhotoAssetLink{Asset: asset, IsThumbnail: boolValue(linkMap["is_thumbnail"])})
	}
	return assetLinks
}

func remotePhotoAssetLinkKey(targetField string) string {
	switch targetField {
	case "trail":
		return "trail_assets_via_trail"
	case "waypoint":
		return "waypoint_assets_via_waypoint"
	case "summit_log":
		return "summit_log_assets_via_summit_log"
	default:
		return ""
	}
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
		if !ok || photo == "" || routeimage.IsPlaceholder(photo) {
			continue
		}
		fileURL := legacyPhotoURL(origin, collection, remoteID, photo)
		photos = append(photos, util.FederatedPhoto{
			CanonicalID: fileURL,
			FileURL:     fileURL,
			SourceName:  photo,
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
		return util.WithoutFederatedShareToken(externalID)
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

type remoteAssetExpandsRequestedKey struct{}

func remoteAssetsRequested(ctx context.Context) bool {
	value, present := ctx.Value(remoteAssetExpandsRequestedKey{}).(bool)
	return !present || value
}

func remotePhotoSourceName(asset map[string]any) string {
	name, _ := asset["file"].(string)
	return name
}
