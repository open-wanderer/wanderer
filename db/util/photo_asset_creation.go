package util

import (
	"errors"
	"fmt"
	"strings"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
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
	if err := validatePhotoAssetInputTargets(app, author, input); err != nil {
		return nil, err
	}

	if input.ExternalProvider != "" && input.ExternalID != "" {
		existing, err := findExistingExternalPhotoAsset(app, author, input.ExternalProvider, input.ExternalID)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return reuseExternalPhotoAsset(app, existing, input)
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
	if err := HashUnsavedPhotoFile(record); err != nil {
		return nil, err
	}

	if err := app.Save(record); err != nil {
		if isExternalPhotoAssetUniqueConflict(err) {
			existing, findErr := findExistingExternalPhotoAsset(app, author, input.ExternalProvider, input.ExternalID)
			if findErr != nil {
				return nil, errors.Join(err, findErr)
			}
			if existing != nil {
				return reuseExternalPhotoAsset(app, existing, input)
			}
		}
		return nil, err
	}
	if err := LinkAssetToPhotoTargets(app, record.Id, input); err != nil {
		return nil, err
	}
	return record, nil
}

func reuseExternalPhotoAsset(app core.App, existing *core.Record, input PhotoAssetInput) (*core.Record, error) {
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
		if err := HashUnsavedPhotoFile(existing); err != nil {
			return nil, err
		}
		if err := app.Save(existing); err != nil {
			return nil, err
		}
	}
	if err := LinkAssetToPhotoTargets(app, existing.Id, input); err != nil {
		return nil, err
	}
	return existing, nil
}

func isExternalPhotoAssetUniqueConflict(err error) bool {
	var fieldErrors validation.Errors
	if !errors.As(err, &fieldErrors) {
		return false
	}
	for _, field := range []string{"author", "external_provider", "external_id", "type"} {
		if fieldErrors[field] == nil {
			return false
		}
	}
	return true
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
