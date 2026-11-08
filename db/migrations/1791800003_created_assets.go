package migrations

import (
	"encoding/json"
	"fmt"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		if _, err := createAssetsCollection(app); err != nil {
			return err
		}
		for _, schema := range []string{trailAssetLinkCollectionJSON(), waypointAssetLinkCollectionJSON(), summitLogAssetLinkCollectionJSON()} {
			if err := createAssetLinkCollection(app, schema); err != nil {
				return err
			}
		}
		waypoints, err := app.FindCollectionByNameOrId("waypoints")
		if err != nil {
			return err
		}
		waypoints.AddIndex("idx_waypoints_author", false, "author", "")
		if err := app.Save(waypoints); err != nil {
			return err
		}
		return nil
	}, func(core.App) error {
		return fmt.Errorf("photo collections cannot be removed safely; restore the matching database and storage backup")
	})
}

const assetFileMaxBytes int64 = 50 << 20

const assetExternalUniqueIndex = "CREATE UNIQUE INDEX idx_assets_external ON assets (author, external_provider, external_id, type) WHERE external_provider != '' AND external_id != ''"
const assetAuthorIndex = "CREATE INDEX idx_assets_author ON assets (author, type, external_provider, external_id)"
const assetAuthorTakenAtIndex = "CREATE INDEX idx_assets_author_taken_at ON assets (author, taken_at, id)"
const assetAuthorLibraryIndex = "CREATE INDEX idx_assets_author_library ON assets (author, type, taken_at DESC, created DESC, id)"

// Match PocketBase's guarded JSON filter expression, including its invalid-
// JSON fallback; a plain json_extract expression only indexes the author.
const assetAuthorCopyLineageIndex = "CREATE INDEX idx_assets_author_copy_lineage ON assets (author, (CASE WHEN json_valid(source_metadata) THEN json_extract(source_metadata, '$.copied_from') ELSE json_extract(json_object('pb', source_metadata), '$.pb.copied_from') END), type)"

func createAssetsCollection(app core.App) (*core.Collection, error) {
	jsonData := fmt.Sprintf(`{
		"createRule": null,
		"deleteRule": null,
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
					"300x300",
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
			%q,
			%q,
			%q,
			%q
		],
		"listRule": null,
		"name": "assets",
		"system": false,
		"type": "base",
		"updateRule": null,
		"viewRule": null
	}`, assetFileMaxBytes, assetExternalUniqueIndex, assetAuthorIndex, assetAuthorTakenAtIndex, assetAuthorLibraryIndex)

	collection := &core.Collection{}
	if err := json.Unmarshal([]byte(jsonData), collection); err != nil {
		return nil, err
	}
	collection.Fields.Add(&core.JSONField{Name: "source_metadata", Hidden: true, MaxSize: 8 << 20})
	collection.Indexes = append(collection.Indexes, assetAuthorCopyLineageIndex)
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

func trailAssetLinkCollectionJSON() string {
	return `{
		"createRule": null,
		"deleteRule": null,
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
			"CREATE INDEX idx_trail_assets_trail ON trail_assets (trail)"
		],
		"listRule": null,
		"name": "trail_assets",
		"system": false,
		"type": "base",
		"updateRule": null,
		"viewRule": null
	}`
}

func waypointAssetLinkCollectionJSON() string {
	return `{
		"createRule": null,
		"deleteRule": null,
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
			"CREATE INDEX idx_waypoint_assets_waypoint ON waypoint_assets (waypoint)"
		],
		"listRule": null,
		"name": "waypoint_assets",
		"system": false,
		"type": "base",
		"updateRule": null,
		"viewRule": null
	}`
}

func summitLogAssetLinkCollectionJSON() string {
	return `{
		"createRule": null,
		"deleteRule": null,
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
			"CREATE INDEX idx_summit_log_assets_summit_log ON summit_log_assets (summit_log)"
		],
		"listRule": null,
		"name": "summit_log_assets",
		"system": false,
		"type": "base",
		"updateRule": null,
		"viewRule": null
	}`
}
