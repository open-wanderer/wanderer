package util

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

type PhotoAssetInput struct {
	Author           string
	Trail            string
	Waypoint         string
	SummitLog        string
	Lat              float64
	Lon              float64
	HasLat           bool
	HasLon           bool
	TakenAt          *time.Time
	StorageMode      string
	File             *filesystem.File
	Metadata         map[string]any
	ExternalProvider string
	ExternalID       string
}

type AssetLinkTarget struct {
	Collection string
	Field      string
	ID         string
}

func CreatePhotoAsset(app core.App, input PhotoAssetInput) (*core.Record, error) {
	storageMode := input.StorageMode
	if storageMode == "" {
		storageMode = "copy"
	}
	if storageMode == "copy" && input.File == nil {
		return nil, nil
	}

	author, err := ResolveAssetAuthor(app, input.Author)
	if err != nil {
		return nil, err
	}
	if author == "" {
		return nil, fmt.Errorf("missing asset author")
	}

	if input.ExternalProvider != "" && input.ExternalID != "" {
		existing, err := findExistingExternalPhotoAsset(app, author, input.ExternalProvider, input.ExternalID)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			changed := false
			if input.File != nil && (existing.GetString("file") == "" || existing.GetString("storage_mode") != "copy") {
				existing.Set("file", input.File)
				existing.Set("storage_mode", "copy")
				existing.Set("remote_status", "available")
				changed = true
			}
			if input.TakenAt != nil && existing.GetDateTime("taken_at").IsZero() {
				existing.Set("taken_at", input.TakenAt)
				changed = true
			}
			if changed {
				if err := app.Save(existing); err != nil {
					return nil, err
				}
			}
			if err := LinkAssetToPhotoTargets(app, existing.Id, input); err != nil {
				return nil, err
			}
			return existing, nil
		}
	}

	collection, err := app.FindCollectionByNameOrId("assets")
	if err != nil {
		return nil, err
	}

	record := core.NewRecord(collection)
	record.Set("type", "photo")
	record.Set("storage_mode", storageMode)
	record.Set("remote_status", "available")
	record.Set("author", author)
	if input.File != nil {
		record.Set("file", input.File)
	}
	if input.HasLat || input.Lat != 0 {
		record.Set("lat", input.Lat)
	}
	if input.HasLon || input.Lon != 0 {
		record.Set("lon", input.Lon)
	}
	if input.TakenAt != nil {
		record.Set("taken_at", input.TakenAt)
	}
	if input.Metadata != nil {
		record.Set("metadata", input.Metadata)
	}
	if input.ExternalProvider != "" {
		record.Set("external_provider", input.ExternalProvider)
	}
	if input.ExternalID != "" {
		record.Set("external_id", input.ExternalID)
	}

	if err := app.Save(record); err != nil {
		return nil, err
	}
	if err := LinkAssetToPhotoTargets(app, record.Id, input); err != nil {
		return nil, err
	}
	return record, nil
}

func findExistingExternalPhotoAsset(app core.App, author string, provider string, externalID string) (*core.Record, error) {
	records, err := app.FindRecordsByFilter(
		"assets",
		"author={:author} && external_provider={:provider} && external_id={:external_id} && type='photo'",
		"created",
		1,
		0,
		dbx.Params{"author": author, "provider": provider, "external_id": externalID},
	)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	return records[0], nil
}

func LinkAssetToPhotoTargets(app core.App, assetID string, input PhotoAssetInput) error {
	for _, target := range PhotoAssetLinkTargets(input.Trail, input.Waypoint, input.SummitLog) {
		if _, err := EnsureAssetLink(app, target.Collection, target.Field, target.ID, assetID); err != nil {
			return err
		}
	}
	return nil
}

func PhotoAssetLinkTargets(trailID string, waypointID string, summitLogID string) []AssetLinkTarget {
	targets := make([]AssetLinkTarget, 0, 3)
	if trailID != "" && waypointID == "" && summitLogID == "" {
		targets = append(targets, AssetLinkTarget{Collection: "trail_assets", Field: "trail", ID: trailID})
	}
	if waypointID != "" {
		targets = append(targets, AssetLinkTarget{Collection: "waypoint_assets", Field: "waypoint", ID: waypointID})
	}
	if summitLogID != "" {
		targets = append(targets, AssetLinkTarget{Collection: "summit_log_assets", Field: "summit_log", ID: summitLogID})
	}
	return targets
}

func EnsureAssetLink(app core.App, collectionName string, targetField string, targetID string, assetID string) (*core.Record, error) {
	if targetID == "" || assetID == "" {
		return nil, nil
	}
	existing, err := app.FindRecordsByFilter(
		collectionName,
		"asset={:asset} && "+targetField+"={:target}",
		"",
		1,
		0,
		dbx.Params{"asset": assetID, "target": targetID},
	)
	if err != nil {
		return nil, err
	}
	if len(existing) > 0 {
		return existing[0], nil
	}
	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		return nil, err
	}
	record := core.NewRecord(collection)
	record.Set("asset", assetID)
	record.Set(targetField, targetID)
	if err := app.Save(record); err != nil {
		return nil, err
	}
	return record, nil
}

func ResolveAssetAuthor(app core.App, rawAuthor string) (string, error) {
	rawAuthor = strings.TrimSpace(rawAuthor)
	if rawAuthor == "" {
		return "", nil
	}
	if _, err := app.FindRecordById("activitypub_actors", rawAuthor); err == nil {
		return rawAuthor, nil
	}
	actor, err := app.FindFirstRecordByData("activitypub_actors", "user", rawAuthor)
	if err != nil {
		return rawAuthor, nil
	}
	return actor.Id, nil
}

func AssetAuthorUserID(app core.App, asset *core.Record) (string, error) {
	if asset == nil {
		return "", fmt.Errorf("missing asset")
	}
	actorID := asset.GetString("author")
	if actorID == "" {
		return "", fmt.Errorf("missing asset author")
	}
	actor, err := app.FindRecordById("activitypub_actors", actorID)
	if err != nil {
		return "", err
	}
	userID := actor.GetString("user")
	if userID == "" {
		return "", fmt.Errorf("asset author is not a local user actor")
	}
	return userID, nil
}

func IsAssetLinkedToPublicTrail(app core.App, asset *core.Record) bool {
	if asset == nil {
		return false
	}
	trailIDs, err := TrailIDsForAsset(app, asset.Id)
	if err != nil {
		return false
	}
	for _, trailID := range trailIDs {
		trail, err := app.FindRecordById("trails", trailID)
		if err == nil && trail.GetBool("public") {
			return true
		}
	}
	return false
}

func AssetPublicMediaURL(record *core.Record, origin string) string {
	if fileURL := AssetPublicFileURL(record, origin); fileURL != "" {
		return fileURL
	}
	if record == nil {
		return ""
	}
	if record.GetString("storage_mode") == "" || record.GetString("storage_mode") == "copy" {
		return ""
	}
	return withOrigin(origin, fmt.Sprintf("/api/v1/assets/%s/file", record.Id))
}

func AssetPublicFileURL(record *core.Record, origin string) string {
	if record == nil {
		return ""
	}
	file := record.GetString("file")
	if file == "" {
		return ""
	}
	collectionID := assetCollectionID(record)
	if collectionID == "" {
		return ""
	}
	return withOrigin(origin, fmt.Sprintf("/api/v1/files/%s/%s/%s", collectionID, record.Id, file))
}

func AssetFileRedirectURL(record *core.Record) string {
	if record == nil {
		return ""
	}
	file := record.GetString("file")
	if file == "" {
		return ""
	}
	collectionID := assetCollectionID(record)
	if collectionID == "" {
		return ""
	}
	return fmt.Sprintf("/api/files/%s/%s/%s", collectionID, record.Id, file)
}

func assetCollectionID(record *core.Record) string {
	if record == nil || record.Collection() == nil {
		return ""
	}
	return record.Collection().Id
}

func withOrigin(origin string, path string) string {
	if origin == "" {
		return path
	}
	return strings.TrimRight(origin, "/") + path
}

func PhotoAssetURLs(app core.App, targetField string, targetID string, origin string, limit int) ([]string, error) {
	records, err := PhotoAssetsForTarget(app, targetField, targetID, limit)
	if err != nil {
		return nil, err
	}

	urls := make([]string, 0, len(records))
	for _, record := range records {
		if url := AssetPublicMediaURL(record, origin); url != "" {
			urls = append(urls, url)
		}
	}
	return urls, nil
}

func PhotoAssetsForTarget(app core.App, targetField string, targetID string, limit int) ([]*core.Record, error) {
	linkCollection, err := AssetLinkCollectionForTarget(targetField)
	if err != nil {
		return nil, err
	}
	if targetID == "" {
		return []*core.Record{}, nil
	}
	links, err := app.FindRecordsByFilter(linkCollection, targetField+"={:id}", "", -1, 0, dbx.Params{"id": targetID})
	if err != nil {
		return nil, err
	}
	assetIDs := make([]string, 0, len(links))
	for _, link := range links {
		assetIDs = append(assetIDs, link.GetString("asset"))
	}
	return recordsByFieldValues(app, "assets", "id", assetIDs, "type='photo'", nil, "-created", limit)
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
func TrailIDForLinkTarget(app core.App, targetField string, targetID string) (string, error) {
	if targetID == "" {
		return "", nil
	}
	switch targetField {
	case "trail":
		return targetID, nil
	case "waypoint":
		waypoint, err := app.FindRecordById("waypoints", targetID)
		if err != nil {
			return "", err
		}
		return waypoint.GetString("trail"), nil
	case "summit_log":
		summitLog, err := app.FindRecordById("summit_logs", targetID)
		if err != nil {
			return "", err
		}
		return summitLog.GetString("trail"), nil
	default:
		return "", fmt.Errorf("unsupported asset relation %q", targetField)
	}
}

func TrailIDsForAsset(app core.App, assetID string) ([]string, error) {
	seen := map[string]struct{}{}
	add := func(id string) {
		if id != "" {
			seen[id] = struct{}{}
		}
	}

	trailLinks, err := app.FindRecordsByFilter("trail_assets", "asset={:asset}", "", -1, 0, dbx.Params{"asset": assetID})
	if err != nil {
		return nil, err
	}
	for _, link := range trailLinks {
		add(link.GetString("trail"))
	}

	waypointLinks, err := app.FindRecordsByFilter("waypoint_assets", "asset={:asset}", "", -1, 0, dbx.Params{"asset": assetID})
	if err != nil {
		return nil, err
	}
	waypoints, err := recordsByIDs(app, "waypoints", recordFieldValues(waypointLinks, "waypoint"), "", -1)
	if err != nil {
		return nil, err
	}
	for _, waypoint := range waypoints {
		add(waypoint.GetString("trail"))
	}

	summitLogLinks, err := app.FindRecordsByFilter("summit_log_assets", "asset={:asset}", "", -1, 0, dbx.Params{"asset": assetID})
	if err != nil {
		return nil, err
	}
	summitLogs, err := recordsByIDs(app, "summit_logs", recordFieldValues(summitLogLinks, "summit_log"), "", -1)
	if err != nil {
		return nil, err
	}
	for _, summitLog := range summitLogs {
		add(summitLog.GetString("trail"))
	}

	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	return ids, nil
}

func IsAssetLinkedToTrail(app core.App, asset *core.Record, trailID string) bool {
	if asset == nil || trailID == "" {
		return false
	}
	trailIDs, err := TrailIDsForAsset(app, asset.Id)
	if err != nil {
		return false
	}
	for _, id := range trailIDs {
		if id == trailID {
			return true
		}
	}
	return false
}

func AssetIDsLinkedToTrail(app core.App, trailID string, assetIDs []string) (map[string]bool, error) {
	linked := map[string]bool{}
	if trailID == "" || len(assetIDs) == 0 {
		return linked, nil
	}

	directLinks, err := recordsByFieldValues(app, "trail_assets", "asset", assetIDs, "trail={:trail}", dbx.Params{"trail": trailID}, "", -1)
	if err != nil {
		return nil, err
	}
	for _, link := range directLinks {
		linked[link.GetString("asset")] = true
	}

	waypointLinks, err := recordsByFieldValues(app, "waypoint_assets", "asset", assetIDs, "", nil, "", -1)
	if err != nil {
		return nil, err
	}
	waypoints, err := recordsByIDs(app, "waypoints", recordFieldValues(waypointLinks, "waypoint"), "", -1)
	if err != nil {
		return nil, err
	}
	waypointTrails := map[string]string{}
	for _, waypoint := range waypoints {
		waypointTrails[waypoint.Id] = waypoint.GetString("trail")
	}
	for _, link := range waypointLinks {
		if waypointTrails[link.GetString("waypoint")] == trailID {
			linked[link.GetString("asset")] = true
		}
	}

	summitLogLinks, err := recordsByFieldValues(app, "summit_log_assets", "asset", assetIDs, "", nil, "", -1)
	if err != nil {
		return nil, err
	}
	summitLogs, err := recordsByIDs(app, "summit_logs", recordFieldValues(summitLogLinks, "summit_log"), "", -1)
	if err != nil {
		return nil, err
	}
	summitLogTrails := map[string]string{}
	for _, summitLog := range summitLogs {
		summitLogTrails[summitLog.Id] = summitLog.GetString("trail")
	}
	for _, link := range summitLogLinks {
		if summitLogTrails[link.GetString("summit_log")] == trailID {
			linked[link.GetString("asset")] = true
		}
	}

	return linked, nil
}

func AssetIDsForLinkTarget(app core.App, linkCollection string, targetField string, targetID string) ([]string, error) {
	if linkCollection == "" || targetField == "" || targetID == "" {
		return []string{}, nil
	}
	links, err := app.FindRecordsByFilter(linkCollection, targetField+"={:target}", "", -1, 0, dbx.Params{"target": targetID})
	if err != nil {
		return nil, err
	}
	return recordFieldValues(links, "asset"), nil
}

func AssetIDsForTrail(app core.App, trailID string) ([]string, error) {
	if trailID == "" {
		return []string{}, nil
	}

	assetIDs, err := AssetIDsForLinkTarget(app, "trail_assets", "trail", trailID)
	if err != nil {
		return nil, err
	}

	waypoints, err := app.FindRecordsByFilter("waypoints", "trail={:trail}", "", -1, 0, dbx.Params{"trail": trailID})
	if err != nil {
		return nil, err
	}
	waypointLinks, err := recordsByFieldValues(app, "waypoint_assets", "waypoint", recordIDs(waypoints), "", nil, "", -1)
	if err != nil {
		return nil, err
	}
	assetIDs = append(assetIDs, recordFieldValues(waypointLinks, "asset")...)

	summitLogs, err := app.FindRecordsByFilter("summit_logs", "trail={:trail}", "", -1, 0, dbx.Params{"trail": trailID})
	if err != nil {
		return nil, err
	}
	summitLogLinks, err := recordsByFieldValues(app, "summit_log_assets", "summit_log", recordIDs(summitLogs), "", nil, "", -1)
	if err != nil {
		return nil, err
	}
	assetIDs = append(assetIDs, recordFieldValues(summitLogLinks, "asset")...)

	return UniqueNonEmptyStrings(assetIDs), nil
}

func DeleteAssetsIfOrphaned(app core.App, assetIDs []string) error {
	for _, assetID := range UniqueNonEmptyStrings(assetIDs) {
		if _, err := DeleteAssetIfOrphaned(app, assetID); err != nil {
			return err
		}
	}
	return nil
}

func DeleteAssetIfOrphaned(app core.App, assetID string) (bool, error) {
	linked, err := AssetHasLinks(app, assetID)
	if err != nil {
		return false, err
	}
	if linked {
		return false, nil
	}

	assets, err := app.FindRecordsByFilter("assets", "id={:id}", "", 1, 0, dbx.Params{"id": assetID})
	if err != nil {
		return false, err
	}
	if len(assets) == 0 {
		return false, nil
	}
	if err := app.Delete(assets[0]); err != nil {
		return false, err
	}
	return true, nil
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

func recordsByIDs(app core.App, collection string, ids []string, sort string, limit int) ([]*core.Record, error) {
	return recordsByFieldValues(app, collection, "id", ids, "", nil, sort, limit)
}

func recordsByFieldValues(app core.App, collection string, field string, values []string, extraFilter string, extraParams dbx.Params, sort string, limit int) ([]*core.Record, error) {
	values = UniqueNonEmptyStrings(values)
	if len(values) == 0 {
		return []*core.Record{}, nil
	}

	params := dbx.Params{}
	for key, value := range extraParams {
		params[key] = value
	}
	filters := make([]string, 0, len(values))
	for i, value := range values {
		key := fmt.Sprintf("value_%d", i)
		params[key] = value
		filters = append(filters, field+"={:"+key+"}")
	}

	filter := "(" + strings.Join(filters, " || ") + ")"
	if strings.TrimSpace(extraFilter) != "" {
		filter += " && (" + extraFilter + ")"
	}
	return app.FindRecordsByFilter(collection, filter, sort, limit, 0, params)
}

func recordFieldValues(records []*core.Record, field string) []string {
	values := make([]string, 0, len(records))
	for _, record := range records {
		values = append(values, record.GetString(field))
	}
	return values
}

func recordIDs(records []*core.Record) []string {
	values := make([]string, 0, len(records))
	for _, record := range records {
		values = append(values, record.Id)
	}
	return values
}
