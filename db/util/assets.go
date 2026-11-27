package util

import (
	"errors"
	"fmt"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func AssetIDsForLinkTarget(app core.App, linkCollection string, targetField string, targetID string) ([]string, error) {
	if linkCollection == "" || targetField == "" || targetID == "" {
		return []string{}, nil
	}
	ids := []string{}
	err := app.DB().Select("asset").From(linkCollection).Where(dbx.HashExp{targetField: targetID}).Column(&ids)
	return ids, err
}

// DeleteAssetsIfOrphaned deletes those of the assets that nothing links to any
// more, whoever their author. One failure does not stop the others; the errors
// are reported together.
func DeleteAssetsIfOrphaned(app core.App, assetIDs []string) error {
	var errs []error
	for _, assetID := range UniqueNonEmptyStrings(assetIDs) {
		if _, err := DeleteAssetIfOrphaned(app, assetID); err != nil {
			errs = append(errs, fmt.Errorf("asset %s: %w", assetID, err))
		}
	}
	return errors.Join(errs...)
}

func DeleteAssetIfOrphaned(app core.App, assetID string) (bool, error) {
	return DeleteAssetIfOrphanedByAuthor(app, assetID, "")
}

func DeleteAssetIfOrphanedByAuthor(app core.App, assetID string, authorID string) (bool, error) {
	deleted := false
	err := app.RunInTransaction(func(tx core.App) error {
		linked, err := AssetHasLinks(tx, assetID)
		if err != nil || linked {
			return err
		}
		assets, err := tx.FindRecordsByFilter("assets", "id={:id}", "", 1, 0, dbx.Params{"id": assetID})
		if err != nil || len(assets) == 0 {
			return err
		}
		if authorID != "" && assets[0].GetString("author") != authorID {
			return nil
		}
		if err := tx.Delete(assets[0]); err != nil {
			return err
		}
		deleted = true
		return nil
	})
	return deleted && err == nil, err
}

func AssetHasLinks(app core.App, assetID string) (bool, error) {
	if assetID == "" {
		return false, nil
	}
	for _, collection := range []string{"trail_assets", "waypoint_assets", "summit_log_assets"} {
		links, err := app.FindRecordsByFilter(collection, "asset={:asset}", "", 1, 0, dbx.Params{"asset": assetID})
		if err != nil {
			return false, err
		}
		if len(links) > 0 {
			return true, nil
		}
	}
	return false, nil
}
