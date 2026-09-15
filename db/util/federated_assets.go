package util

import (
	"context"
	"fmt"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// FederationAssetProvider marks photo assets that were materialized from a
// federated record. Their external_id holds the photo's canonical identity at
// the author's origin instance, so repeated syncs and update activities
// resolve to the same local asset instead of creating duplicates.
const FederationAssetProvider = "federation"

// FederatedPhoto describes one remote photo attached to a federated record.
type FederatedPhoto struct {
	// CanonicalID is the photo's stable identity at the author's origin.
	CanonicalID string
	// FileURL is where the photo bytes can be fetched from.
	FileURL     string
	Lat         float64
	Lon         float64
	HasLat      bool
	HasLon      bool
	IsThumbnail bool
}

// CanonicalFederatedAssetID returns the cross-instance identity of a local
// asset record: an asset that was itself materialized from federation keeps
// the identity of its origin (so the id stays stable across hops), any other
// asset is identified by this instance's asset IRI.
func CanonicalFederatedAssetID(record *core.Record, origin string) string {
	if record == nil {
		return ""
	}
	if record.GetString("external_provider") == FederationAssetProvider {
		if externalID := record.GetString("external_id"); externalID != "" {
			return externalID
		}
	}
	return fmt.Sprintf("%s/api/v1/assets/%s", strings.TrimRight(origin, "/"), record.Id)
}

// ReconcileFederatedPhotoAssets brings the photo assets linked to a federated
// target (trail/waypoint/summit_log) in line with the authoritative photo
// list of the record's origin instance. Photos are matched by their canonical
// origin identity, which makes both delivery paths idempotent: new photos are
// downloaded (size-bounded) and linked, known ones are kept as-is, and
// federation-managed photos the origin no longer lists are unlinked and
// deleted once orphaned. A failed download only skips that photo; it never
// causes removals because the removal set is derived from the remote list,
// not from what could be fetched.
func ReconcileFederatedPhotoAssets(app core.App, ctx context.Context, targetField string, targetID string, author string, photos []FederatedPhoto) error {
	if targetID == "" {
		return nil
	}
	authorID, err := ResolveAssetAuthor(app, author)
	if err != nil {
		return err
	}
	if authorID == "" {
		return fmt.Errorf("missing asset author")
	}

	existing, err := PhotoAssetsForTarget(app, targetField, targetID, -1)
	if err != nil {
		return err
	}
	managed := make(map[string]*core.Record, len(existing))
	for _, record := range existing {
		if record.GetString("external_provider") != FederationAssetProvider {
			continue
		}
		if externalID := record.GetString("external_id"); externalID != "" {
			managed[externalID] = record
		}
	}
	// Photos materialized before assets carried a canonical identity cannot be
	// matched against the remote list; reconciling next to them would duplicate
	// every photo, so such targets keep their existing state.
	if len(existing) > 0 && len(managed) == 0 {
		return nil
	}

	remote := make(map[string]struct{}, len(photos))
	thumbnailAssetID := ""
	for _, photo := range photos {
		if photo.CanonicalID == "" {
			continue
		}
		remote[photo.CanonicalID] = struct{}{}
		if existingManaged := managed[photo.CanonicalID]; existingManaged != nil {
			if photo.IsThumbnail {
				thumbnailAssetID = existingManaged.Id
			}
			continue
		}

		input := PhotoAssetInput{
			Author:           authorID,
			ExternalProvider: FederationAssetProvider,
			ExternalID:       photo.CanonicalID,
			Lat:              photo.Lat,
			Lon:              photo.Lon,
			HasLat:           photo.HasLat,
			HasLon:           photo.HasLon,
			Metadata:         map[string]any{"source": "federation"},
		}
		switch targetField {
		case "trail":
			input.Trail = targetID
		case "waypoint":
			input.Waypoint = targetID
		case "summit_log":
			input.SummitLog = targetID
		default:
			return fmt.Errorf("unsupported asset relation %q", targetField)
		}

		// The author may already hold this photo from another target or an
		// earlier delivery; only fetch the bytes when it is genuinely new.
		existingAsset, err := findExistingExternalPhotoAsset(app, authorID, FederationAssetProvider, photo.CanonicalID)
		if err != nil {
			return err
		}
		if existingAsset != nil {
			if err := LinkAssetToPhotoTargets(app, existingAsset.Id, input); err != nil {
				return err
			}
			if photo.IsThumbnail {
				thumbnailAssetID = existingAsset.Id
			}
			continue
		}

		file, err := FetchPublicFile(ctx, photo.FileURL, "federated-photo", DefaultPluginMediaMaxBytes)
		if err != nil {
			continue
		}
		input.File = file
		asset, err := CreatePhotoAsset(app, input)
		if err != nil {
			return err
		}
		if photo.IsThumbnail && asset != nil {
			thumbnailAssetID = asset.Id
		}
	}

	if targetField == "trail" && thumbnailAssetID != "" {
		if err := SetTrailThumbnailAsset(app, targetID, thumbnailAssetID); err != nil {
			return err
		}
	}

	linkCollection, err := AssetLinkCollectionForTarget(targetField)
	if err != nil {
		return err
	}
	for externalID, record := range managed {
		if _, ok := remote[externalID]; ok {
			continue
		}
		links, err := app.FindRecordsByFilter(
			linkCollection,
			"asset={:asset} && "+targetField+"={:target}",
			"",
			-1,
			0,
			dbx.Params{"asset": record.Id, "target": targetID},
		)
		if err != nil {
			return err
		}
		for _, link := range links {
			if err := app.Delete(link); err != nil {
				return err
			}
		}
		if _, err := DeleteAssetIfOrphanedByAuthor(app, record.Id, authorID); err != nil {
			return err
		}
	}

	return nil
}
