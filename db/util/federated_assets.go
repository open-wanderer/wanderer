package util

import (
	"context"
	"errors"
	"fmt"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// FederationAssetProvider marks photo assets that were materialized from a
// federated record. Their external_id holds the photo's photo identity at
// the author's origin instance, so repeated syncs and update activities
// resolve to the same local asset instead of creating duplicates.
const FederationAssetProvider = "federation"

// FederatedPhoto describes one remote photo attached to a federated record.
type FederatedPhoto struct {
	// CanonicalID is the photo's stable identity at the author's origin.
	CanonicalID string
	// FileURL is where the photo bytes can be fetched from.
	FileURL       string
	SourceName    string
	Lat           float64
	Lon           float64
	HasLat        bool
	HasLon        bool
	IsThumbnail   bool
	GeneratedKind string
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
func ReconcileFederatedPhotoAssets(app core.App, ctx context.Context, targetField string, targetID string, author string, photos []FederatedPhoto, preserveUnlisted ...bool) error {
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
	if err := validateFederatedAssetTargets(app, authorID, []AssetLinkTarget{{Field: targetField, ID: targetID}}); err != nil {
		return err
	}

	existing, err := PhotoAssetsForTarget(app, targetField, targetID, -1)
	if err != nil {
		return err
	}
	managed := make(map[string]*core.Record, len(existing))
	hasLegacyPhoto := false
	remote := make(map[string]struct{}, len(photos))
	thumbnailAssetID := ""
	incomplete := false
	// Finish link additions/corrections before the thumbnail pass reads links:
	// a generated link removed at this flush must not be saved from a stale snapshot.
	if err := WithDeferredTargetSync(app, func() error {
		actorCache := map[string]*core.Record{}
		loadActor := func(id string) *core.Record {
			if actor, known := actorCache[id]; known {
				return actor
			}
			actor, _ := app.FindRecordById("activitypub_actors", id)
			actorCache[id] = actor
			return actor
		}
		corrected := loadActor(authorID)
		correctedAssets := map[string]*core.Record{}
		for i, record := range existing {
			// Correcting an earlier duplicate may already have materialized this
			// keeper. Do not save its stale file/metadata snapshot below.
			if fresh := correctedAssets[record.Id]; fresh != nil {
				record = fresh
				existing[i] = record
			}
			if record.GetString("external_provider") != FederationAssetProvider {
				if !IsGeneratedRoutePreviewAsset(record) {
					hasLegacyPhoto = true
				}
				continue
			}
			if record.GetString("author") != authorID {
				previous := loadActor(record.GetString("author"))
				// A local user's independent photo is never managed by a remote
				// origin, even when it was originally copied from federation.
				if previous == nil || previous.GetBool("is_local") {
					continue
				}
				if corrected != nil && !corrected.GetBool("is_local") && ValidateRemoteObjectAuthor(corrected, record.GetString("external_id"), nil) == nil {
					correctedAsset, err := correctFederatedTargetPhotoAsset(app, record, authorID, targetField, targetID)
					if err != nil {
						return err
					}
					if record.Id == correctedAsset.Id {
						existing[i] = correctedAsset
					}
					record = correctedAsset
					correctedAssets[record.Id] = record
				}
			}
			if externalID := record.GetString("external_id"); externalID != "" {
				managed[externalID] = record
			}
		}
		// A later correction can also refresh an earlier keeper's metadata.
		for i, record := range existing {
			if fresh := correctedAssets[record.Id]; fresh != nil {
				existing[i] = fresh
			}
		}
		legacyAssets := NewFederatedPhotoAssetIndex(existing, authorID)

		originIRI := ""
		originResolved := false
		for _, photo := range photos {
			if photo.CanonicalID == "" {
				continue
			}
			if photo.GeneratedKind == "route-preview" {
				continue
			}
			remote[photo.CanonicalID] = struct{}{}
			if i, asset := legacyAssets.FirstMatch(photo); asset != nil {
				oldID := asset.GetString("external_id")
				adopted, err := reuseFederatedTargetPhotoAsset(app, asset.Fresh(), authorID, photo.CanonicalID, targetField, targetID)
				if err != nil {
					incomplete = true
					continue
				}
				delete(managed, oldID)
				managed[photo.CanonicalID] = adopted
				existing[i] = adopted
				legacyAssets.Replace(i, adopted)
			}

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

			if !originResolved {
				collection := map[string]string{"trail": "trails", "waypoint": "waypoints", "summit_log": "summit_logs"}[targetField]
				target, err := app.FindRecordById(collection, targetID)
				if err != nil {
					return err
				}
				originIRI = target.GetString("iri")
				originResolved = true
			}
			file, cleanup, err := FetchFederatedFile(ctx, photo.FileURL, originIRI, "federated-photo", FederatedPhotoFileLimit(app))
			if err != nil {
				cleanup()
				incomplete = true
				continue
			}
			input.File = file
			asset, err := CreatePhotoAsset(app, input)
			cleanup()
			if err != nil {
				var invalid validation.Errors
				if errors.As(err, &invalid) {
					incomplete = true
					continue
				}
				return err
			}
			if photo.IsThumbnail && asset != nil {
				thumbnailAssetID = asset.Id
			}
		}
		return nil
	}); err != nil {
		return err
	}

	// Thumbnail writes and removals form a separate batch, including early
	// incomplete returns, so bulk unlinking also syncs each target only once.
	return WithDeferredTargetSync(app, func() error {
		if targetField == "trail" && thumbnailAssetID != "" {
			if err := SetTrailThumbnailAsset(app, targetID, thumbnailAssetID); err != nil {
				return err
			}
		}

		// A capped activity is a preview of the origin's photo set. Only a full
		// sync can prove that photos outside its slots were removed.
		if hasLegacyPhoto || incomplete || len(preserveUnlisted) > 0 && preserveUnlisted[0] {
			return nil
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
			if _, err := DeleteAssetIfOrphanedByAuthor(app, record.Id, record.GetString("author")); err != nil {
				return err
			}
		}

		return nil
	})
}

// A remote sync must not place its photos into a local actor's library under
// a remote target, even if an earlier sync already created such an asset.
func validateFederatedAssetTargets(app core.App, author string, targets []AssetLinkTarget) error {
	actor, err := app.FindRecordById("activitypub_actors", author)
	if err != nil {
		return err
	}
	if !actor.GetBool("is_local") {
		return nil
	}
	if len(targets) == 0 {
		return fmt.Errorf("federated photos for a local author require a local target")
	}
	for _, target := range targets {
		collection := ""
		switch target.Field {
		case "trail":
			collection = "trails"
		case "waypoint":
			collection = "waypoints"
		case "summit_log":
			collection = "summit_logs"
		default:
			return fmt.Errorf("unsupported asset relation %q", target.Field)
		}
		record, err := app.FindRecordById(collection, target.ID)
		if err != nil {
			return err
		}
		if iri := record.GetString("iri"); iri != "" && !IsLocalIRI(iri) {
			return fmt.Errorf("refusing federated photos for a local author on a remote target")
		}
	}
	return nil
}
