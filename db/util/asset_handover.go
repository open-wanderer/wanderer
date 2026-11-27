package util

import (
	"database/sql"
	"errors"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const assetHandoverBatchSize = 150

type assetHandoverTarget struct {
	collection string
	field      string
}

var assetHandoverTargets = []assetHandoverTarget{
	{"trail_assets", "trail"},
	{"waypoint_assets", "waypoint"},
	{"summit_log_assets", "summit_log"},
}

type assetHandoverOwner struct {
	actor      string
	created    string
	link       string
	collection string
}

// HandOverActorAssets keeps contributed photos with the targets that survive
// their author's deletion. The caller runs it with the delete in one transaction.
func HandOverActorAssets(app core.App, actorID string) error {
	ids := []string{}
	if err := app.DB().Select("id").From("assets").Where(dbx.HashExp{"author": actorID}).Column(&ids); err != nil {
		return err
	}
	for _, chunk := range stringChunks(ids, assetHandoverBatchSize) {
		assets, err := recordsByIDs(app, "assets", chunk)
		if err != nil {
			return err
		}
		owners, err := oldestForeignAssetOwners(app, actorID, recordIDs(assets))
		if err != nil {
			return err
		}
		for _, asset := range assets {
			owner, found := owners[asset.Id]
			if !found {
				continue
			}
			existing, err := assetForHandoverIdentity(app, asset, owner.actor)
			if err != nil {
				return err
			}
			if existing != nil && sameAssetHandoverPhoto(asset, existing) {
				if err := moveAssetHandoverLinks(app, asset.Id, existing.Id); err != nil {
					return err
				}
			} else {
				if existing != nil {
					metadata := AssetProvenanceMetadata(asset)
					metadata["source_external_provider"] = asset.GetString("external_provider")
					metadata["source_external_id"] = asset.GetString("external_id")
					asset.Set(AssetProvenanceField, metadata)
					asset.Set("external_provider", "")
					asset.Set("external_id", "")
				}
				asset.Set("author", owner.actor)
				if err := app.Save(asset); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func sameAssetHandoverPhoto(source, destination *core.Record) bool {
	sourceMetadata, destinationMetadata := map[string]any{}, map[string]any{}
	if source.UnmarshalJSONField("metadata", &sourceMetadata) != nil || destination.UnmarshalJSONField("metadata", &destinationMetadata) != nil {
		return false
	}
	hash, _ := sourceMetadata["content_hash"].(string)
	return hash != "" && hash == destinationMetadata["content_hash"]
}

func oldestForeignAssetOwners(app core.App, actorID string, assetIDs []string) (map[string]assetHandoverOwner, error) {
	owners := map[string]assetHandoverOwner{}
	for _, target := range assetHandoverTargets {
		err := eachAssetHandoverLinkPage(app, target.collection, assetIDs, func(links []*core.Record) error {
			targetOwners, err := assetHandoverTargetOwners(app, target.field, actorID, recordFieldValues(links, target.field))
			if err != nil {
				return err
			}
			ownerIDs := []string{}
			for _, owner := range targetOwners {
				if owner != "" && owner != actorID {
					ownerIDs = append(ownerIDs, owner)
				}
			}
			actors, err := recordsByIDs(app, "activitypub_actors", ownerIDs)
			if err != nil {
				return err
			}
			present := map[string]bool{}
			for _, actor := range actors {
				present[actor.Id] = true
			}
			for _, link := range links {
				owner := targetOwners[link.GetString(target.field)]
				if owner == actorID || !present[owner] {
					continue
				}
				candidate := assetHandoverOwner{actor: owner, created: link.GetString("created"), link: link.Id, collection: target.collection}
				previous, found := owners[link.GetString("asset")]
				if !found || candidate.olderThan(previous) {
					owners[link.GetString("asset")] = candidate
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return owners, nil
}

func (owner assetHandoverOwner) olderThan(other assetHandoverOwner) bool {
	if owner.created != other.created {
		return owner.created < other.created
	}
	if owner.link != other.link {
		return owner.link < other.link
	}
	return owner.collection < other.collection
}

func assetHandoverTargetOwners(app core.App, target, actorID string, ids []string) (map[string]string, error) {
	collection := map[string]string{"trail": "trails", "waypoint": "waypoints", "summit_log": "summit_logs"}[target]
	records, err := recordsByIDs(app, collection, ids)
	if err != nil {
		return nil, err
	}
	owners := map[string]string{}
	if target == "trail" {
		for _, record := range records {
			if record.GetString("author") != actorID {
				owners[record.Id] = record.GetString("author")
			}
		}
		return owners, nil
	}
	trails, err := recordsByIDs(app, "trails", recordFieldValues(records, "trail"))
	if err != nil {
		return nil, err
	}
	trailOwners := map[string]string{}
	for _, trail := range trails {
		trailOwners[trail.Id] = trail.GetString("author")
	}
	for _, record := range records {
		if record.GetString("author") == actorID {
			continue
		}
		trailOwner := trailOwners[record.GetString("trail")]
		if trailOwner == actorID {
			continue
		}
		if target == "waypoint" {
			owners[record.Id] = trailOwner
		} else {
			owners[record.Id] = record.GetString("author")
		}
	}
	return owners, nil
}

func assetForHandoverIdentity(app core.App, source *core.Record, owner string) (*core.Record, error) {
	provider, id := source.GetString("external_provider"), source.GetString("external_id")
	if provider == "" || id == "" {
		return nil, nil
	}
	existing, err := app.FindFirstRecordByFilter("assets", "author={:author} && external_provider={:provider} && external_id={:id} && type={:type}", dbx.Params{"author": owner, "provider": provider, "id": id, "type": source.GetString("type")})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return existing, err
}

func moveAssetHandoverLinks(app core.App, sourceID, destinationID string) error {
	for _, target := range assetHandoverTargets {
		err := eachAssetHandoverLinkPage(app, target.collection, []string{sourceID}, func(links []*core.Record) error {
			for _, link := range links {
				existing, err := app.FindFirstRecordByFilter(target.collection, "asset={:asset} && "+target.field+"={:target}", dbx.Params{"asset": destinationID, "target": link.GetString(target.field)})
				if errors.Is(err, sql.ErrNoRows) {
					link.Set("asset", destinationID)
					if err := app.Save(link); err != nil {
						return err
					}
					continue
				}
				if err != nil {
					return err
				}
				changed := false
				if target.field == "trail" && link.GetBool("is_thumbnail") && !existing.GetBool("is_thumbnail") {
					existing.Set("is_thumbnail", true)
					changed = true
				}
				if link.GetString("created") < existing.GetString("created") {
					existing.SetRaw("created", link.GetRaw("created"))
					changed = true
				}
				if changed {
					if err := app.Save(existing); err != nil {
						return err
					}
				}
				if err := app.Delete(link); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func eachAssetHandoverLinkPage(app core.App, collection string, assetIDs []string, visit func([]*core.Record) error) error {
	for _, ids := range stringChunks(assetIDs, assetHandoverBatchSize) {
		values := make([]any, len(ids))
		for i, id := range ids {
			values[i] = id
		}
		cursor := ""
		for {
			links := []*core.Record{}
			err := app.RecordQuery(collection).AndWhere(dbx.In("asset", values...)).AndWhere(dbx.NewExp("id>{:cursor}", dbx.Params{"cursor": cursor})).OrderBy("id").Limit(assetHandoverBatchSize).All(&links)
			if err != nil {
				return err
			}
			if len(links) == 0 {
				break
			}
			nextCursor := links[len(links)-1].Id
			if err := visit(links); err != nil {
				return err
			}
			cursor = nextCursor
		}
	}
	return nil
}
