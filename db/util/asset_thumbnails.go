package util

import (
	"encoding/json"
	"fmt"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"strings"
)

// TrailThumbnailURLs picks the search thumbnail of each trail: its marked
// photo, else its first one. A generated route preview stands in only for a
// trail without a real photo, again the marked one before the first.
func TrailThumbnailURLs(app core.App, trails []*core.Record, origin string) (map[string]string, error) {
	urls := make(map[string]string, len(trails))
	if len(trails) == 0 {
		return urls, nil
	}

	assetCollection, err := app.FindCollectionByNameOrId("assets")
	if err != nil {
		return nil, err
	}

	trailIDs := make([]string, 0, len(trails))
	for _, trail := range trails {
		trailIDs = append(trailIDs, trail.Id)
	}

	photos := map[string]*trailThumbnailCandidates{}
	previews := map[string]*trailThumbnailCandidates{}
	for _, chunk := range stringChunks(trailIDs, 200) {
		rows, err := trailThumbnailRows(app, chunk)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			if row.AssetType != "photo" {
				continue
			}
			url := assetPublicMediaURLFromFields(assetCollection.Id, row.AssetID, row.File, row.StorageMode, origin)
			if url == "" {
				continue
			}
			candidates := photos
			if isGeneratedRoutePreviewAssetFields(row) {
				candidates = previews
			}
			if candidates[row.TrailID] == nil {
				candidates[row.TrailID] = &trailThumbnailCandidates{}
			}
			candidates[row.TrailID].add(row, url)
		}
	}

	for trailID, candidates := range photos {
		urls[trailID] = candidates.url()
	}
	for trailID, candidates := range previews {
		if urls[trailID] == "" {
			urls[trailID] = candidates.url()
		}
	}
	return urls, nil
}

// trailThumbnailCandidates are the usable photos of one kind on one trail. If
// several links are marked, the one marked last wins.
type trailThumbnailCandidates struct {
	first         string
	marked        string
	markedUpdated string
}

func (c *trailThumbnailCandidates) add(row trailThumbnailRow, url string) {
	if c.first == "" {
		c.first = url
	}
	if row.IsThumbnail && (c.marked == "" || row.LinkUpdated > c.markedUpdated) {
		c.marked = url
		c.markedUpdated = row.LinkUpdated
	}
}

func (c *trailThumbnailCandidates) url() string {
	if c.marked != "" {
		return c.marked
	}
	return c.first
}

type trailThumbnailRow struct {
	TrailID          string `db:"trail_id"`
	AssetID          string `db:"asset_id"`
	IsThumbnail      bool   `db:"is_thumbnail"`
	LinkUpdated      string `db:"link_updated"`
	LinkCreated      string `db:"link_created"`
	AssetType        string `db:"asset_type"`
	File             string `db:"file"`
	StorageMode      string `db:"storage_mode"`
	ExternalProvider string `db:"external_provider"`
	TakenAt          string `db:"taken_at"`
	Metadata         string `db:"metadata"`
}

func trailThumbnailRows(app core.App, trailIDs []string) ([]trailThumbnailRow, error) {
	if len(trailIDs) == 0 {
		return []trailThumbnailRow{}, nil
	}
	values := make([]any, 0, len(trailIDs))
	for _, trailID := range trailIDs {
		values = append(values, trailID)
	}
	rows := []trailThumbnailRow{}
	err := app.DB().
		Select(
			"ta.trail AS trail_id",
			"ta.asset AS asset_id",
			"ta.is_thumbnail AS is_thumbnail",
			"ta.updated AS link_updated",
			"ta.created AS link_created",
			"a.type AS asset_type",
			"COALESCE(a.file, '') AS file",
			"COALESCE(a.storage_mode, '') AS storage_mode",
			"COALESCE(a.external_provider, '') AS external_provider",
			"COALESCE(a.taken_at, '') AS taken_at",
			"COALESCE(a.metadata, '') AS metadata",
		).
		From("trail_assets ta").
		InnerJoin("assets a", dbx.NewExp("a.id = ta.asset")).
		Where(dbx.In("ta.trail", values...)).
		OrderBy("ta.created ASC").
		All(&rows)
	return rows, err
}

func isGeneratedRoutePreviewAssetFields(row trailThumbnailRow) bool {
	metadata := map[string]any{}
	if raw := strings.TrimSpace(row.Metadata); raw != "" && raw != "null" {
		_ = json.Unmarshal([]byte(raw), &metadata)
	}
	// The row leaves out external_id: only assets with a provider have one,
	// and those are never legacy previews.
	filename := assetFilename(metadata, row.File, "", row.AssetID)
	return isGeneratedRoutePreview(metadata, filename, row.ExternalProvider, strings.TrimSpace(row.TakenAt) != "")
}

func assetPublicMediaURLFromFields(collectionID string, assetID string, file string, storageMode string, origin string) string {
	if assetID == "" {
		return ""
	}
	if file != "" && collectionID != "" {
		return withOrigin(origin, fmt.Sprintf("/api/v1/files/%s/%s/%s", collectionID, assetID, file))
	}
	if storageMode == "" || storageMode == "copy" {
		return ""
	}
	return withOrigin(origin, fmt.Sprintf("/api/v1/assets/%s/file", assetID))
}
