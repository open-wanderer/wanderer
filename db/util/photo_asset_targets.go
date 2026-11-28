package util

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"strings"
)

func PhotoAssetsForTarget(app core.App, targetField string, targetID string, limit int) ([]*core.Record, error) {
	linkCollection, err := AssetLinkCollectionForTarget(targetField)
	if err != nil {
		return nil, err
	}
	if targetID == "" {
		return []*core.Record{}, nil
	}
	links, err := app.FindRecordsByFilter(linkCollection, targetField+"={:id}", "+created", -1, 0, dbx.Params{"id": targetID})
	if err != nil && strings.Contains(err.Error(), "invalid sort field") {
		links, err = app.FindRecordsByFilter(linkCollection, targetField+"={:id}", "", -1, 0, dbx.Params{"id": targetID})
	}
	if err != nil {
		return nil, err
	}
	assetIDs := make([]string, 0, len(links))
	for _, link := range links {
		assetIDs = append(assetIDs, link.GetString("asset"))
	}
	assets, err := recordsByFieldValues(app, "assets", "id", assetIDs, "type='photo'", nil)
	if err != nil {
		return nil, err
	}
	assetsByID := make(map[string]*core.Record, len(assets))
	for _, asset := range assets {
		assetsByID[asset.Id] = asset
	}

	ordered := make([]*core.Record, 0, len(assets))
	for _, assetID := range assetIDs {
		asset := assetsByID[assetID]
		if asset == nil {
			continue
		}
		ordered = append(ordered, asset)
		if limit > 0 && len(ordered) >= limit {
			break
		}
	}

	return ordered, nil
}

func SetTrailThumbnailAsset(app core.App, trailID string, assetID string) error {
	if trailID == "" {
		return nil
	}
	links, err := app.FindRecordsByFilter(
		"trail_assets",
		"trail={:trail}",
		"",
		-1,
		0,
		dbx.Params{"trail": trailID},
	)
	if err != nil {
		return err
	}
	for _, link := range links {
		shouldBeThumbnail := assetID != "" && link.GetString("asset") == assetID
		if link.GetBool("is_thumbnail") == shouldBeThumbnail {
			continue
		}
		link.Set("is_thumbnail", shouldBeThumbnail)
		if err := app.Save(link); err != nil {
			return err
		}
	}
	return nil
}
