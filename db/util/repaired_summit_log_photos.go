package util

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// HandOverRepairedSummitLogPhotos keeps migrated photos with an entry when
// its contributor replaces the earlier remote trail-author attribution.
// Independent uses and generated previews retain their existing ownership.
func HandOverRepairedSummitLogPhotos(app core.App, entryID, fromAuthor, toAuthor string) error {
	if entryID == "" || fromAuthor == "" || toAuthor == "" || fromAuthor == toAuthor {
		return nil
	}
	return app.RunInTransaction(func(tx core.App) error {
		entry, err := tx.FindRecordById("summit_logs", entryID)
		if err != nil {
			return err
		}
		if entry.GetString("author") != toAuthor {
			return nil
		}
		parent, err := tx.FindRecordById("trails", entry.GetString("trail"))
		if err != nil {
			return err
		}
		if parent.GetString("author") != fromAuthor {
			return nil
		}
		previous, err := tx.FindRecordById("activitypub_actors", fromAuthor)
		if err != nil {
			return err
		}
		corrected, err := tx.FindRecordById("activitypub_actors", toAuthor)
		if err != nil {
			return err
		}
		if previous.GetBool("is_local") || ValidateRemoteObjectAuthor(corrected, entry.GetString("iri"), nil) != nil {
			return nil
		}
		photos, err := PhotoAssetsForTarget(tx, "summit_log", entryID, -1)
		if err != nil {
			return err
		}
		for _, photo := range photos {
			if photo.GetString("author") != fromAuthor || photo.GetString("external_provider") != "" || IsGeneratedRoutePreviewAsset(photo) {
				continue
			}
			independent, err := summitLogPhotoHasOtherLinks(tx, photo.Id, entryID)
			if err != nil {
				return err
			}
			if independent {
				continue
			}
			photo.Set("author", toAuthor)
			if err := tx.Save(photo); err != nil {
				return err
			}
		}
		return nil
	})
}

func summitLogPhotoHasOtherLinks(app core.App, assetID, entryID string) (bool, error) {
	for _, collection := range []string{"trail_assets", "waypoint_assets", "summit_log_assets"} {
		filter := "asset={:asset}"
		if collection == "summit_log_assets" {
			filter += " && summit_log!={:entry}"
		}
		links, err := app.FindRecordsByFilter(collection, filter, "", 1, 0, dbx.Params{"asset": assetID, "entry": entryID})
		if err != nil {
			return false, err
		}
		if len(links) != 0 {
			return true, nil
		}
	}
	return false, nil
}
