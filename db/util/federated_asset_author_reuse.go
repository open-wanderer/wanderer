package util

import (
	"database/sql"
	"errors"
	"fmt"
	"path"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

// Reuse the corrected author's canonical photo if another delivery already
// stored it. Move only this target's links; independent uses of the older
// asset retain their file and current owner until their own sync corrects it.
func correctFederatedTargetPhotoAsset(app core.App, source *core.Record, author, targetField, targetID string) (*core.Record, error) {
	return reuseFederatedTargetPhotoAsset(app, source, author, source.GetString("external_id"), targetField, targetID)
}

// FindFederatedPhotoIdentityAsset also lets prefetch recognize a photo already
// held for another target before preparing any bytes.
func FindFederatedPhotoIdentityAsset(app core.App, author, identity string) (*core.Record, error) {
	return findExistingExternalPhotoAsset(app, author, FederationAssetProvider, identity)
}

func reuseFederatedTargetPhotoAsset(app core.App, source *core.Record, author, identity, targetField, targetID string) (*core.Record, error) {
	var result *core.Record
	err := app.RunInTransaction(func(tx core.App) error {
		reuse, err := FindFederatedPhotoIdentityAsset(tx, author, identity)
		if err != nil {
			return err
		}
		if reuse == nil {
			source.Set("author", author)
			source.Set("external_provider", FederationAssetProvider)
			source.Set("external_id", identity)
			if err := tx.Save(source); err != nil {
				return err
			}
			result = source
			return nil
		}
		if reuse.Id == source.Id {
			result = reuse
			return nil
		}
		if reuse.GetString("file") == "" && source.GetString("file") != "" {
			fsys, err := tx.NewFilesystem()
			if err != nil {
				return err
			}
			defer fsys.Close()
			reader, err := fsys.GetReader(path.Join(source.BaseFilesPath(), source.GetString("file")))
			if err != nil {
				return err
			}
			body, err := ReadBoundedForPlugin(reader, DefaultPluginMediaMaxBytes)
			reader.Close()
			if err != nil {
				return err
			}
			file, err := filesystem.NewFileFromBytes(body, source.GetString("file"))
			if err != nil {
				return err
			}
			reuse.Set("file", file)
			reuse.Set("storage_mode", "copy")
			reuse.Set("remote_status", "available")
		}
		if reuse.GetString("file") == "" && reuse.GetString("storage_mode") == "link_private" {
			return fmt.Errorf("cannot reuse an unmaterialized private federated photo")
		}
		metadata, sourceMetadata := AssetMetadataWithProvenance(reuse), AssetMetadataWithProvenance(source)
		for key, value := range sourceMetadata {
			if _, exists := metadata[key]; !exists {
				metadata[key] = value
			}
		}
		reuse.Set("metadata", metadata)
		if err := tx.Save(reuse); err != nil {
			return err
		}
		collection, err := AssetLinkCollectionForTarget(targetField)
		if err != nil {
			return err
		}
		links, err := tx.FindRecordsByFilter(collection, "asset={:source} && "+targetField+"={:target}", "", -1, 0, dbx.Params{"source": source.Id, "target": targetID})
		if err != nil {
			return err
		}
		for _, link := range links {
			existing, err := tx.FindFirstRecordByFilter(collection, "asset={:reuse} && "+targetField+"={:target}", dbx.Params{"reuse": reuse.Id, "target": targetID})
			if errors.Is(err, sql.ErrNoRows) {
				link.Set("asset", reuse.Id)
				if err := tx.Save(link); err != nil {
					return err
				}
			} else if err != nil {
				return err
			} else {
				// Keep the matched link's id and order even if the canonical
				// photo was already linked here. Preserve either chosen cover.
				link.Set("is_thumbnail", link.GetBool("is_thumbnail") || existing.GetBool("is_thumbnail"))
				if err := tx.Delete(existing); err != nil {
					return err
				}
				link.Set("asset", reuse.Id)
				if err := tx.Save(link); err != nil {
					return err
				}

			}
		}
		if _, err := DeleteAssetIfOrphanedByAuthor(tx, source.Id, source.GetString("author")); err != nil {
			return err
		}
		result = reuse
		return nil
	})
	return result, err
}
