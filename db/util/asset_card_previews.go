package util

import (
	"fmt"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const CardPhotoPreviewLimit = 3

// CardPhotoLinks returns a bounded preview instead of a complete photo
// relation. The marked real photo wins; route previews are fallback images.
func CardPhotoLinks(app core.App, info *core.RequestInfo, targetField string, targetIDs []string) (map[string][]*core.Record, error) {
	result := map[string][]*core.Record{}
	if len(targetIDs) == 0 {
		return result, nil
	}
	unique := make([]string, 0, len(targetIDs))
	seen := map[string]bool{}
	for _, id := range targetIDs {
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	targetIDs = unique
	linkName, err := AssetLinkCollectionForTarget(targetField)
	if err != nil {
		return nil, err
	}
	links, err := app.FindCollectionByNameOrId(linkName)
	if err != nil {
		return nil, err
	}
	assets, err := app.FindCollectionByNameOrId("assets")
	if err != nil {
		return nil, err
	}
	assetFields := []string{}
	for _, name := range []string{"id", "author", "type", "file", "storage_mode", "external_provider", "external_id", "taken_at", "lat", "lon", "remote_status", "remote_checked_at", "remote_missing_since", "remote_error", "created", "updated"} {
		if assets.Fields.GetByName(name) != nil {
			assetFields = append(assetFields, "assets."+name)
		}
	}
	if assets.Fields.GetByName("metadata") != nil {
		assetFields = append(assetFields, "json_object('generated', json_extract(assets.metadata, '$.generated')) AS metadata")
	}
	generated := "0"
	if assets.Fields.GetByName("metadata") != nil {
		generated = "CASE WHEN json_extract(a.metadata, '$.generated.kind')='route-preview' THEN 1 ELSE 0 END"
	}
	usable := "1"
	if assets.Fields.GetByName("file") != nil && assets.Fields.GetByName("storage_mode") != nil {
		usable = "(COALESCE(a.file, '')!='' OR a.storage_mode IN ('link_public','link_private'))"
	}
	marked, created, updated := "0", "l.id", "l.id"
	if links.Fields.GetByName("is_thumbnail") != nil {
		marked = "l.is_thumbnail"
	}
	if links.Fields.GetByName("created") != nil {
		created = "l.created"
	}
	if links.Fields.GetByName("updated") != nil {
		updated = "l.updated"
	}
	for _, chunk := range stringChunks(targetIDs, 100) {
		params := dbx.Params{}
		placeholders := []string{}
		for i, id := range chunk {
			key := fmt.Sprintf("card%d", i)
			params[key] = id
			placeholders = append(placeholders, "{:"+key+"}")
		}
		query := fmt.Sprintf(`SELECT id, target_id FROM (
			SELECT l.id, l.%s AS target_id, ROW_NUMBER() OVER (PARTITION BY l.%s ORDER BY %s ASC, %s DESC, CASE WHEN %s THEN %s END DESC, %s ASC, l.id ASC) AS position
			FROM %s l JOIN assets a ON a.id=l.asset
			WHERE l.%s IN (%s) AND a.type='photo' AND %s
		) WHERE position<=%d ORDER BY target_id, position`, targetField, targetField, generated, marked, marked, updated, created, linkName, targetField, strings.Join(placeholders, ","), usable, CardPhotoPreviewLimit)
		rows := []struct {
			ID       string `db:"id"`
			TargetID string `db:"target_id"`
		}{}
		if err := app.DB().NewQuery(query).Bind(params).All(&rows); err != nil {
			return nil, err
		}
		ids := []string{}
		for _, row := range rows {
			ids = append(ids, row.ID)
		}
		if len(ids) == 0 {
			continue
		}
		candidates, err := app.FindRecordsByIds(links, ids)
		if err != nil {
			return nil, err
		}
		byID := map[string]*core.Record{}
		assetIDs := []string{}
		for _, link := range candidates {
			if allowed, err := app.CanAccessRecord(link, info, links.ViewRule); err != nil || !allowed {
				continue
			}
			byID[link.Id] = link
			assetIDs = append(assetIDs, link.GetString("asset"))
		}
		photos, err := app.FindRecordsByIds(assets, assetIDs, func(query *dbx.SelectQuery) error {
			query.Select(assetFields...)
			return nil
		})
		if err != nil {
			return nil, err
		}
		byAsset := map[string]*core.Record{}
		for _, photo := range photos {
			if allowed, err := app.CanAccessRecord(photo, info, assets.ViewRule); err == nil && allowed {
				byAsset[photo.Id] = photo
			}
		}
		for _, row := range rows {
			link := byID[row.ID]
			if link == nil {
				continue
			}
			photo := byAsset[link.GetString("asset")]
			if photo == nil {
				continue
			}
			link.SetExpand(map[string]any{"asset": photo})
			result[row.TargetID] = append(result[row.TargetID], link)
		}
	}
	return result, nil
}

func CardPhotoThumbnail(links []*core.Record) int {
	for _, generated := range []bool{false, true} {
		first := -1
		for i, link := range links {
			asset := link.ExpandedOne("asset")
			if asset == nil || IsGeneratedRoutePreviewAsset(asset) != generated {
				continue
			}
			if first == -1 {
				first = i
			}
			if link.GetBool("is_thumbnail") {
				return i
			}
		}
		if first >= 0 {
			return first
		}
	}
	return 0
}

func AssetLinkCollectionForTarget(targetField string) (string, error) {
	switch targetField {
	case "trail":
		return "trail_assets", nil
	case "waypoint":
		return "waypoint_assets", nil
	case "summit_log":
		return "summit_log_assets", nil
	default:
		return "", fmt.Errorf("unsupported asset relation %q", targetField)
	}
}

// TrailIDForLinkTarget resolves the trail a given asset-link target belongs to.
// Trail links point at the trail directly; waypoint and summit_log links resolve
// through their parent record.
