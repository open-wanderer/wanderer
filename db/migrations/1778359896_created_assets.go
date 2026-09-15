package migrations

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func init() {
	m.Register(func(app core.App) error {
		assetCollection, err := createAssetsCollection(app)
		if err != nil {
			return err
		}
		if err := createAssetLinkCollection(app, trailAssetLinkCollectionJSON()); err != nil {
			return err
		}
		if err := createAssetLinkCollection(app, waypointAssetLinkCollectionJSON()); err != nil {
			return err
		}
		if err := createAssetLinkCollection(app, summitLogAssetLinkCollectionJSON()); err != nil {
			return err
		}
		if err := updateAssetSharingAccessRules(app); err != nil {
			return err
		}

		if err := migrateExistingPhotosToAssets(app, assetCollection); err != nil {
			return err
		}
		if err := removeLegacyPhotoFields(app); err != nil {
			return err
		}

		return nil
	}, func(app core.App) error {
		// Rollbacks are performed by restoring a full backup. A partial down
		// migration would remove migrated asset data without restoring photos.
		return nil
	})
}

const assetFileMaxBytes int64 = 50 << 20

const assetExternalUniqueIndex = "CREATE UNIQUE INDEX idx_assets_external ON assets (author, external_provider, external_id, type) WHERE external_provider != '' AND external_id != ''"

const (
	trailAssetTargetViewRule = `trail.author.user = @request.auth.id || trail.public = true || (@request.auth.id != "" && trail.trail_share_via_trail.actor.user ?= @request.auth.id) || (@request.query.share != "" && trail.trail_link_share_via_trail.token ?= @request.query.share)`
	trailAssetTargetEditRule = `trail.author.user = @request.auth.id || (trail.trail_share_via_trail.actor.user ?= @request.auth.id && trail.trail_share_via_trail.permission = "edit")`
	trailAssetReadRule       = `asset.author.user = @request.auth.id || ` + trailAssetTargetViewRule
	trailAssetCreateRule     = `@request.auth.id != "" && asset.author.user = @request.auth.id && (` + trailAssetTargetEditRule + `)`
	trailAssetWriteRule      = `@request.auth.id != "" && (asset.author.user = @request.auth.id || ` + trailAssetTargetEditRule + `)`
	trailAssetUpdateRule     = trailAssetWriteRule + ` && @request.body.asset:changed = false && @request.body.trail:changed = false`

	waypointAssetTargetViewRule = `waypoint.trail.author.user = @request.auth.id || waypoint.author.user = @request.auth.id || waypoint.trail.public = true || (@request.auth.id != "" && waypoint.trail.trail_share_via_trail.actor.user ?= @request.auth.id) || (@request.query.share != "" && waypoint.trail.trail_link_share_via_trail.token ?= @request.query.share)`
	waypointAssetTargetEditRule = `waypoint.trail.author.user = @request.auth.id || waypoint.author.user = @request.auth.id || (waypoint.trail.trail_share_via_trail.actor.user ?= @request.auth.id && waypoint.trail.trail_share_via_trail.permission = "edit")`
	waypointAssetReadRule       = `asset.author.user = @request.auth.id || ` + waypointAssetTargetViewRule
	waypointAssetCreateRule     = `@request.auth.id != "" && asset.author.user = @request.auth.id && (` + waypointAssetTargetEditRule + `)`
	waypointAssetWriteRule      = `@request.auth.id != "" && (asset.author.user = @request.auth.id || ` + waypointAssetTargetEditRule + `)`
	waypointAssetUpdateRule     = waypointAssetWriteRule + ` && @request.body.asset:changed = false && @request.body.waypoint:changed = false`

	summitLogAssetTargetViewRule = `summit_log.trail.author.user = @request.auth.id || summit_log.author.user = @request.auth.id || summit_log.trail.public = true || (@request.auth.id != "" && summit_log.trail.trail_share_via_trail.actor.user ?= @request.auth.id) || (@request.query.share != "" && summit_log.trail.trail_link_share_via_trail.token ?= @request.query.share)`
	summitLogAssetTargetEditRule = `summit_log.trail.author.user = @request.auth.id || summit_log.author.user = @request.auth.id || (summit_log.trail.trail_share_via_trail.actor.user ?= @request.auth.id && summit_log.trail.trail_share_via_trail.permission = "edit")`
	summitLogAssetReadRule       = `asset.author.user = @request.auth.id || ` + summitLogAssetTargetViewRule
	summitLogAssetCreateRule     = `@request.auth.id != "" && asset.author.user = @request.auth.id && (` + summitLogAssetTargetEditRule + `)`
	summitLogAssetWriteRule      = `@request.auth.id != "" && (asset.author.user = @request.auth.id || ` + summitLogAssetTargetEditRule + `)`
	summitLogAssetUpdateRule     = summitLogAssetWriteRule + ` && @request.body.asset:changed = false && @request.body.summit_log:changed = false`

	assetReadRule = `author.user = @request.auth.id || trail_assets_via_asset.trail.author.user ?= @request.auth.id || trail_assets_via_asset.trail.public ?= true || (@request.auth.id != "" && trail_assets_via_asset.trail.trail_share_via_trail.actor.user ?= @request.auth.id) || (@request.query.share != "" && trail_assets_via_asset.trail.trail_link_share_via_trail.token ?= @request.query.share) || waypoint_assets_via_asset.waypoint.trail.author.user ?= @request.auth.id || waypoint_assets_via_asset.waypoint.author.user ?= @request.auth.id || waypoint_assets_via_asset.waypoint.trail.public ?= true || (@request.auth.id != "" && waypoint_assets_via_asset.waypoint.trail.trail_share_via_trail.actor.user ?= @request.auth.id) || (@request.query.share != "" && waypoint_assets_via_asset.waypoint.trail.trail_link_share_via_trail.token ?= @request.query.share) || summit_log_assets_via_asset.summit_log.author.user ?= @request.auth.id || summit_log_assets_via_asset.summit_log.trail.author.user ?= @request.auth.id || summit_log_assets_via_asset.summit_log.trail.public ?= true || (@request.auth.id != "" && summit_log_assets_via_asset.summit_log.trail.trail_share_via_trail.actor.user ?= @request.auth.id) || (@request.query.share != "" && summit_log_assets_via_asset.summit_log.trail.trail_link_share_via_trail.token ?= @request.query.share)`
)

type photoMigrationConfig struct {
	Collection          string
	LinkCollection      string
	LinkField           string
	LatField            string
	LonField            string
	TrailCoordinateLink bool
}

func createAssetsCollection(app core.App) (*core.Collection, error) {
	jsonData := fmt.Sprintf(`{
		"createRule": "@request.auth.id != \"\" && @request.body.author.user = @request.auth.id",
		"deleteRule": "@request.auth.id != \"\" && author.user = @request.auth.id",
		"fields": [
			{
				"autogeneratePattern": "[a-z0-9]{15}",
				"hidden": false,
				"id": "textassetid01",
				"max": 15,
				"min": 15,
				"name": "id",
				"pattern": "^[a-z0-9]+$",
				"presentable": false,
				"primaryKey": true,
				"required": true,
				"system": true,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "selectassettyp",
				"maxSelect": 1,
				"name": "type",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "select",
				"values": ["photo"]
			},
			{
				"hidden": false,
				"id": "fileassetfile1",
				"maxSelect": 1,
				"maxSize": %d,
				"mimeTypes": [
					"image/jpeg",
					"image/png",
					"image/vnd.mozilla.apng",
					"image/webp",
					"image/svg+xml",
					"image/heic",
					"video/mp4",
					"video/webm",
					"video/ogg"
				],
				"name": "file",
				"presentable": false,
				"protected": false,
				"required": false,
				"system": false,
				"thumbs": [
					"600x0"
				],
				"type": "file"
			},
			{
				"hidden": false,
				"id": "selectassetsto",
				"maxSelect": 1,
				"name": "storage_mode",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "select",
				"values": ["copy", "link_private"]
			},
			{
				"hidden": false,
				"id": "selectassetrem",
				"maxSelect": 1,
				"name": "remote_status",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "select",
				"values": ["available", "missing", "inaccessible"]
			},
			{
				"cascadeDelete": true,
				"collectionId": "pbc_1295301207",
				"hidden": false,
				"id": "relassetauth1",
				"maxSelect": 1,
				"minSelect": 0,
				"name": "author",
				"presentable": false,
				"required": true,
				"system": false,
				"type": "relation"
			},
			{
				"hidden": false,
				"id": "textassetprov",
				"max": 0,
				"min": 0,
				"name": "external_provider",
				"pattern": "",
				"presentable": false,
				"primaryKey": false,
				"required": false,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "textassetextid",
				"max": 0,
				"min": 0,
				"name": "external_id",
				"pattern": "",
				"presentable": false,
				"primaryKey": false,
				"required": false,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "dateassettaken",
				"max": "",
				"min": "",
				"name": "taken_at",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "date"
			},
			{
				"hidden": false,
				"id": "dateassetcheck",
				"max": "",
				"min": "",
				"name": "remote_checked_at",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "date"
			},
			{
				"hidden": false,
				"id": "dateassetmiss",
				"max": "",
				"min": "",
				"name": "remote_missing_since",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "date"
			},
			{
				"hidden": false,
				"id": "textasseterr1",
				"max": 0,
				"min": 0,
				"name": "remote_error",
				"pattern": "",
				"presentable": false,
				"primaryKey": false,
				"required": false,
				"system": false,
				"type": "text"
			},
			{
				"hidden": false,
				"id": "numassetlat01",
				"max": null,
				"min": null,
				"name": "lat",
				"onlyInt": false,
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "numassetlon01",
				"max": null,
				"min": null,
				"name": "lon",
				"onlyInt": false,
				"presentable": false,
				"required": false,
				"system": false,
				"type": "number"
			},
			{
				"hidden": false,
				"id": "jsonassetmeta",
				"maxSize": 2000000,
				"name": "metadata",
				"presentable": false,
				"required": false,
				"system": false,
				"type": "json"
			},
			{
				"hidden": false,
				"id": "autassetcreat",
				"name": "created",
				"onCreate": true,
				"onUpdate": false,
				"presentable": false,
				"system": false,
				"type": "autodate"
			},
			{
				"hidden": false,
				"id": "autassetupdat",
				"name": "updated",
				"onCreate": true,
				"onUpdate": true,
				"presentable": false,
				"system": false,
				"type": "autodate"
			}
		],
		"id": "assetcollect001",
		"indexes": [
			%q
		],
		"listRule": "author.user = @request.auth.id",
		"name": "assets",
		"system": false,
		"type": "base",
		"updateRule": "@request.auth.id != \"\" && author.user = @request.auth.id",
		"viewRule": "author.user = @request.auth.id"
	}`, assetFileMaxBytes, assetExternalUniqueIndex)

	collection := &core.Collection{}
	if err := json.Unmarshal([]byte(jsonData), collection); err != nil {
		return nil, err
	}
	if err := app.Save(collection); err != nil {
		return nil, err
	}
	return collection, nil
}

func createAssetLinkCollection(app core.App, jsonData string) error {
	collection := &core.Collection{}
	if err := json.Unmarshal([]byte(jsonData), collection); err != nil {
		return err
	}
	return app.Save(collection)
}

func updateAssetsForLinkCollections(app core.App, collection *core.Collection) error {
	if err := json.Unmarshal([]byte(fmt.Sprintf(`{
		"listRule": %q,
		"viewRule": %q
	}`, assetReadRule, assetReadRule)), collection); err != nil {
		return err
	}
	return app.Save(collection)
}

func updateAssetSharingAccessRules(app core.App) error {
	assetCollection, err := app.FindCollectionByNameOrId("assets")
	if err != nil {
		return err
	}
	if err := updateAssetsForLinkCollections(app, assetCollection); err != nil {
		return err
	}

	linkCollections := []struct {
		Name       string
		CreateRule string
		ReadRule   string
		WriteRule  string
		UpdateRule string
	}{
		{"trail_assets", trailAssetCreateRule, trailAssetReadRule, trailAssetWriteRule, trailAssetUpdateRule},
		{"waypoint_assets", waypointAssetCreateRule, waypointAssetReadRule, waypointAssetWriteRule, waypointAssetUpdateRule},
		{"summit_log_assets", summitLogAssetCreateRule, summitLogAssetReadRule, summitLogAssetWriteRule, summitLogAssetUpdateRule},
	}

	for _, update := range linkCollections {
		collection, err := app.FindCollectionByNameOrId(update.Name)
		if err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(fmt.Sprintf(`{
			"createRule": %q,
			"deleteRule": %q,
			"listRule": %q,
			"updateRule": %q,
			"viewRule": %q
			}`, update.CreateRule, update.WriteRule, update.ReadRule, update.UpdateRule, update.ReadRule)), collection); err != nil {
			return err
		}
		if err := app.Save(collection); err != nil {
			return err
		}
	}

	return nil
}

func trailAssetLinkCollectionJSON() string {
	return fmt.Sprintf(`{
		"createRule": %q,
		"deleteRule": %q,
		"fields": [
			{"autogeneratePattern":"[a-z0-9]{15}","hidden":false,"id":"txttrailassetid","max":15,"min":15,"name":"id","pattern":"^[a-z0-9]+$","presentable":false,"primaryKey":true,"required":true,"system":true,"type":"text"},
			{"cascadeDelete":true,"collectionId":"assetcollect001","hidden":false,"id":"reltrassetaset","maxSelect":1,"minSelect":0,"name":"asset","presentable":false,"required":true,"system":false,"type":"relation"},
			{"cascadeDelete":true,"collectionId":"e864strfxo14pm4","hidden":false,"id":"reltrassettrai","maxSelect":1,"minSelect":0,"name":"trail","presentable":false,"required":true,"system":false,"type":"relation"},
			{"hidden":false,"id":"booltrassetthu","name":"is_thumbnail","presentable":false,"required":false,"system":false,"type":"bool"},
			{"hidden":false,"id":"auttrassetcre","name":"created","onCreate":true,"onUpdate":false,"presentable":false,"system":false,"type":"autodate"},
			{"hidden":false,"id":"auttrassetupd","name":"updated","onCreate":true,"onUpdate":true,"presentable":false,"system":false,"type":"autodate"}
		],
		"id": "trailassets0001",
		"indexes": [
			"CREATE UNIQUE INDEX idx_trail_assets_unique ON trail_assets (asset, trail)",
			"CREATE INDEX idx_trail_assets_trail ON trail_assets (trail)",
			"CREATE INDEX idx_trail_assets_asset ON trail_assets (asset)"
		],
		"listRule": %q,
		"name": "trail_assets",
		"system": false,
		"type": "base",
		"updateRule": %q,
		"viewRule": %q
	}`, trailAssetCreateRule, trailAssetWriteRule, trailAssetReadRule, trailAssetUpdateRule, trailAssetReadRule)
}

func waypointAssetLinkCollectionJSON() string {
	return fmt.Sprintf(`{
		"createRule": %q,
		"deleteRule": %q,
		"fields": [
			{"autogeneratePattern":"[a-z0-9]{15}","hidden":false,"id":"txtwpassetid01","max":15,"min":15,"name":"id","pattern":"^[a-z0-9]+$","presentable":false,"primaryKey":true,"required":true,"system":true,"type":"text"},
			{"cascadeDelete":true,"collectionId":"assetcollect001","hidden":false,"id":"relwpassetasse","maxSelect":1,"minSelect":0,"name":"asset","presentable":false,"required":true,"system":false,"type":"relation"},
			{"cascadeDelete":true,"collectionId":"goeo2ubp103rzp9","hidden":false,"id":"relwpassetwpoi","maxSelect":1,"minSelect":0,"name":"waypoint","presentable":false,"required":true,"system":false,"type":"relation"},
			{"hidden":false,"id":"autwpassetcre","name":"created","onCreate":true,"onUpdate":false,"presentable":false,"system":false,"type":"autodate"},
			{"hidden":false,"id":"autwpassetupd","name":"updated","onCreate":true,"onUpdate":true,"presentable":false,"system":false,"type":"autodate"}
		],
		"id": "waypointassets1",
		"indexes": [
			"CREATE UNIQUE INDEX idx_waypoint_assets_unique ON waypoint_assets (asset, waypoint)",
			"CREATE INDEX idx_waypoint_assets_waypoint ON waypoint_assets (waypoint)",
			"CREATE INDEX idx_waypoint_assets_asset ON waypoint_assets (asset)"
		],
		"listRule": %q,
		"name": "waypoint_assets",
		"system": false,
		"type": "base",
		"updateRule": %q,
		"viewRule": %q
	}`, waypointAssetCreateRule, waypointAssetWriteRule, waypointAssetReadRule, waypointAssetUpdateRule, waypointAssetReadRule)
}

func summitLogAssetLinkCollectionJSON() string {
	return fmt.Sprintf(`{
		"createRule": %q,
		"deleteRule": %q,
		"fields": [
			{"autogeneratePattern":"[a-z0-9]{15}","hidden":false,"id":"txtslassetid01","max":15,"min":15,"name":"id","pattern":"^[a-z0-9]+$","presentable":false,"primaryKey":true,"required":true,"system":true,"type":"text"},
			{"cascadeDelete":true,"collectionId":"assetcollect001","hidden":false,"id":"relslassetasse","maxSelect":1,"minSelect":0,"name":"asset","presentable":false,"required":true,"system":false,"type":"relation"},
			{"cascadeDelete":true,"collectionId":"dd2l9a4vxpy2ni8","hidden":false,"id":"relslassetslog","maxSelect":1,"minSelect":0,"name":"summit_log","presentable":false,"required":true,"system":false,"type":"relation"},
			{"hidden":false,"id":"autslassetcre","name":"created","onCreate":true,"onUpdate":false,"presentable":false,"system":false,"type":"autodate"},
			{"hidden":false,"id":"autslassetupd","name":"updated","onCreate":true,"onUpdate":true,"presentable":false,"system":false,"type":"autodate"}
		],
		"id": "summitassets001",
		"indexes": [
			"CREATE UNIQUE INDEX idx_summit_log_assets_unique ON summit_log_assets (asset, summit_log)",
			"CREATE INDEX idx_summit_log_assets_summit_log ON summit_log_assets (summit_log)",
			"CREATE INDEX idx_summit_log_assets_asset ON summit_log_assets (asset)"
		],
		"listRule": %q,
		"name": "summit_log_assets",
		"system": false,
		"type": "base",
		"updateRule": %q,
		"viewRule": %q
	}`, summitLogAssetCreateRule, summitLogAssetWriteRule, summitLogAssetReadRule, summitLogAssetUpdateRule, summitLogAssetReadRule)
}

func migrateExistingPhotosToAssets(app core.App, assetCollection *core.Collection) error {
	configs := []photoMigrationConfig{
		{Collection: "trails", LinkCollection: "trail_assets", LinkField: "trail", LatField: "lat", LonField: "lon"},
		{Collection: "waypoints", LinkCollection: "waypoint_assets", LinkField: "waypoint", LatField: "lat", LonField: "lon"},
		{Collection: "summit_logs", LinkCollection: "summit_log_assets", LinkField: "summit_log", TrailCoordinateLink: true},
	}

	fsys, err := app.NewFilesystem()
	if err != nil {
		return err
	}
	defer fsys.Close()

	trailProviders, err := loadTrailProviders(app)
	if err != nil {
		return err
	}

	for _, cfg := range configs {
		records, err := app.FindAllRecords(cfg.Collection)
		if err != nil {
			return err
		}
		for _, record := range records {
			if err := migrateRecordPhotosToAssets(app, fsys, assetCollection, cfg, record, trailProviders); err != nil {
				return err
			}
		}
	}

	return nil
}

func loadTrailProviders(app core.App) (map[string]string, error) {
	refs, err := app.FindRecordsByFilter("trail_external_reference", "provider='komoot' || provider='strava'", "", -1, 0, nil)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(refs))
	for _, ref := range refs {
		if trailID := ref.GetString("trail"); trailID != "" {
			result[trailID] = ref.GetString("provider")
		}
	}
	return result, nil
}

func migrateRecordPhotosToAssets(app core.App, fsys *filesystem.System, assetCollection *core.Collection, cfg photoMigrationConfig, source *core.Record, trailProviders map[string]string) error {
	photos := source.GetStringSlice("photos")
	if len(photos) == 0 {
		return nil
	}

	linkCollection, err := app.FindCollectionByNameOrId(cfg.LinkCollection)
	if err != nil {
		return err
	}

	author, err := resolveAssetAuthor(app, source.GetString("author"))
	if err != nil {
		app.Logger().Warn("skipping legacy photos with unresolved author during asset migration", "collection", cfg.Collection, "record", source.Id, "author", source.GetString("author"), "error", err)
		return nil
	}
	if author == "" {
		app.Logger().Warn("skipping legacy photos with missing author during asset migration", "collection", cfg.Collection, "record", source.Id)
		return nil
	}

	thumbnailIndex := source.GetInt("thumbnail")
	for i, photo := range photos {
		tempPath, cleanup, err := copyLegacyPhotoToTempFile(fsys, source, photo)
		if err != nil {
			app.Logger().Warn("skipping invalid legacy photo during asset migration", "collection", cfg.Collection, "record", source.Id, "photo", photo, "error", err)
			continue
		}
		file, err := filesystem.NewFileFromPath(tempPath)
		if err != nil {
			cleanup()
			app.Logger().Warn("skipping invalid legacy photo during asset migration", "collection", cfg.Collection, "record", source.Id, "photo", photo, "error", err)
			continue
		}

		asset := core.NewRecord(assetCollection)
		asset.Set("type", "photo")
		asset.Set("storage_mode", "copy")
		asset.Set("remote_status", "available")
		asset.Set("file", file)
		asset.Set("author", author)
		asset.Set("metadata", map[string]any{
			"source_collection": cfg.Collection,
			"source_record":     source.Id,
			"source_file":       photo,
		})
		if isUnmodifiedSinceCreation(source) && cfg.Collection != "summit_logs" {
			trailID := source.Id
			if cfg.Collection == "waypoints" {
				trailID = source.GetString("trail")
			}
			if provider := trailProviders[trailID]; provider != "" {
				asset.Set("external_provider", provider)
			}
		}
		if cfg.LatField != "" {
			asset.Set("lat", source.GetFloat(cfg.LatField))
		}
		if cfg.LonField != "" {
			asset.Set("lon", source.GetFloat(cfg.LonField))
		}
		if cfg.TrailCoordinateLink {
			if trail, err := app.FindRecordById("trails", source.GetString("trail")); err == nil {
				asset.Set("lat", trail.GetFloat("lat"))
				asset.Set("lon", trail.GetFloat("lon"))
			}
		}

		if err := app.Save(asset); err != nil {
			cleanup()
			return err
		}
		cleanup()

		link := core.NewRecord(linkCollection)
		link.Set("asset", asset.Id)
		link.Set(cfg.LinkField, source.Id)
		if cfg.Collection == "trails" && i == thumbnailIndex {
			link.Set("is_thumbnail", true)
		}
		if err := app.Save(link); err != nil {
			return err
		}
	}

	source.Set("photos", []string{})
	return app.Save(source)
}

func copyLegacyPhotoToTempFile(fsys *filesystem.System, source *core.Record, photo string) (string, func(), error) {
	reader, err := fsys.GetReader(source.BaseFilesPath() + "/" + photo)
	if err != nil {
		return "", func() {}, err
	}
	defer reader.Close()

	tempDir, err := os.MkdirTemp("", "wanderer-asset-migration-*")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() {
		_ = os.RemoveAll(tempDir)
	}

	name := filepath.Base(photo)
	if name == "." || name == string(filepath.Separator) {
		name = "photo"
	}
	tempPath := filepath.Join(tempDir, name)
	target, err := os.Create(tempPath)
	if err != nil {
		cleanup()
		return "", func() {}, err
	}
	if _, err := io.Copy(target, reader); err != nil {
		_ = target.Close()
		cleanup()
		return "", func() {}, err
	}
	if err := target.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return tempPath, cleanup, nil
}

func removeLegacyPhotoFields(app core.App) error {
	fields := []struct {
		Collection string
		FieldID    string
	}{
		{Collection: "trails", FieldID: "aqbpyawe"},
		{Collection: "waypoints", FieldID: "tfhs3juh"},
		{Collection: "summit_logs", FieldID: "ixnksbkt"},
	}

	for _, f := range fields {
		collection, err := app.FindCollectionByNameOrId(f.Collection)
		if err != nil {
			return err
		}
		collection.Fields.RemoveById(f.FieldID)
		if err := app.Save(collection); err != nil {
			return err
		}
	}

	return nil
}

func isUnmodifiedSinceCreation(record *core.Record) bool {
	return record.GetDateTime("created").Time().Equal(record.GetDateTime("updated").Time())
}

func resolveAssetAuthor(app core.App, rawAuthor string) (string, error) {
	if rawAuthor == "" {
		return "", nil
	}
	if _, err := app.FindRecordById("activitypub_actors", rawAuthor); err == nil {
		return rawAuthor, nil
	}
	actor, err := app.FindFirstRecordByData("activitypub_actors", "user", rawAuthor)
	if err != nil {
		return "", nil
	}
	return actor.Id, nil
}
