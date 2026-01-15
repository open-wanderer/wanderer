package migrations

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"pocketbase/util"
)

func init() {
	client := meilisearch.New(
		os.Getenv("MEILI_URL"),
		meilisearch.WithAPIKey(os.Getenv("MEILI_MASTER_KEY")),
	)

	m.Register(func(app core.App) error {
		collections := []string{
			"trail_time_bucket_entries",
			"trail_time_buckets",
			"trail_quad_nodes",
			"quad_nodes",
		}
		for _, name := range collections {
			if err := deleteCollectionIfExists(app, name); err != nil {
				return err
			}
		}

		quadNodesJSON := `{
			"id": "pbc_6000010001",
			"listRule": "",
			"viewRule": "",
			"createRule": null,
			"updateRule": null,
			"deleteRule": null,
			"name": "quad_nodes",
			"type": "base",
			"fields": [
				{
					"autogeneratePattern": "[a-z0-9]{15}",
					"hidden": false,
					"id": "text3208210256",
					"max": 15,
					"min": 15,
					"name": "id",
					"pattern": "^[a-z0-9]+$",
					"presentable": false,
					"primaryKey": true,
					"required": false,
					"system": true,
					"type": "text"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "relparent1",
					"max": 0,
					"min": 0,
					"name": "parent",
					"pattern": "",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"hidden": false,
					"id": "numdepth01",
					"max": null,
					"min": 0,
					"name": "depth",
					"onlyInt": true,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "numminlat",
					"max": null,
					"min": null,
					"name": "min_lat",
					"onlyInt": false,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "numminlon",
					"max": null,
					"min": null,
					"name": "min_lon",
					"onlyInt": false,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "nummaxlat",
					"max": null,
					"min": null,
					"name": "max_lat",
					"onlyInt": false,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "nummaxlon",
					"max": null,
					"min": null,
					"name": "max_lon",
					"onlyInt": false,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "numtrailc",
					"max": null,
					"min": 0,
					"name": "trail_count",
					"onlyInt": true,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "boolleaf1",
					"name": "is_leaf",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "bool"
				},
				{
					"hidden": false,
					"id": "selectsm1",
					"maxSelect": 1,
					"name": "split_mode",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "select",
					"values": [
						"spatial",
						"logical"
					]
				},
				{
					"hidden": false,
					"id": "numcfrom1",
					"max": null,
					"min": null,
					"name": "created_from",
					"onlyInt": true,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "numcto001",
					"max": null,
					"min": null,
					"name": "created_to",
					"onlyInt": true,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "textpath1",
					"max": 0,
					"min": 0,
					"name": "path",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": false,
					"system": false,
					"type": "text"
				},
				{
					"hidden": false,
					"id": "autodate2990389176",
					"name": "created",
					"onCreate": true,
					"onUpdate": false,
					"presentable": false,
					"system": false,
					"type": "autodate"
				},
				{
					"hidden": false,
					"id": "autodate3332085495",
					"name": "updated",
					"onCreate": true,
					"onUpdate": true,
					"presentable": false,
					"system": false,
					"type": "autodate"
				}
			],
			"indexes": [
				"CREATE INDEX idx_quad_nodes_leaf ON quad_nodes(is_leaf)",
				"CREATE INDEX idx_quad_nodes_parent ON quad_nodes(parent)",
				"CREATE INDEX idx_quad_nodes_bbox ON quad_nodes(min_lat, max_lat, min_lon, max_lon)"
			],
			"options": {},
			"system": false
		}`

		trailQuadNodesJSON := `{
			"id": "pbc_6000010002",
			"listRule": "",
			"viewRule": "",
			"createRule": null,
			"updateRule": null,
			"deleteRule": null,
			"name": "trail_quad_nodes",
			"type": "base",
			"fields": [
				{
					"autogeneratePattern": "[a-z0-9]{15}",
					"hidden": false,
					"id": "text3208210256",
					"max": 15,
					"min": 15,
					"name": "id",
					"pattern": "^[a-z0-9]+$",
					"presentable": false,
					"primaryKey": true,
					"required": false,
					"system": true,
					"type": "text"
				},
				{
					"cascadeDelete": true,
					"collectionId": "e864strfxo14pm4",
					"hidden": false,
					"id": "reltrail1",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "trail",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "relation"
				},
				{
					"cascadeDelete": true,
					"collectionId": "pbc_6000010001",
					"hidden": false,
					"id": "relnode01",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "quad_node",
					"presentable": false,
					"required": false,
					"system": false,
					"type": "relation"
				},
				{
					"hidden": false,
					"id": "autodate2990389176",
					"name": "created",
					"onCreate": true,
					"onUpdate": false,
					"presentable": false,
					"system": false,
					"type": "autodate"
				},
				{
					"hidden": false,
					"id": "autodate3332085495",
					"name": "updated",
					"onCreate": true,
					"onUpdate": true,
					"presentable": false,
					"system": false,
					"type": "autodate"
				}
			],
			"indexes": [
				"CREATE UNIQUE INDEX idx_trail_quad_nodes_unique ON trail_quad_nodes(trail, quad_node)",
				"CREATE INDEX idx_trail_quad_nodes_trail ON trail_quad_nodes(trail)",
				"CREATE INDEX idx_trail_quad_nodes_node ON trail_quad_nodes(quad_node)"
			],
			"options": {},
			"system": false
		}`

		trailTimeBucketsJSON := `{
			"id": "pbc_6000010003",
			"listRule": "",
			"viewRule": "",
			"createRule": null,
			"updateRule": null,
			"deleteRule": null,
			"name": "trail_time_buckets",
			"type": "base",
			"fields": [
				{
					"autogeneratePattern": "[a-z0-9]{15}",
					"hidden": false,
					"id": "text3208210256",
					"max": 15,
					"min": 15,
					"name": "id",
					"pattern": "^[a-z0-9]+$",
					"presentable": false,
					"primaryKey": true,
					"required": false,
					"system": true,
					"type": "text"
				},
				{
					"hidden": false,
					"id": "numtfrom1",
					"max": null,
					"min": null,
					"name": "created_from",
					"onlyInt": true,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "numtto001",
					"max": null,
					"min": null,
					"name": "created_to",
					"onlyInt": true,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "numtcount",
					"max": null,
					"min": 0,
					"name": "trail_count",
					"onlyInt": true,
					"presentable": false,
					"required": false,
					"system": false,
					"type": "number"
				},
				{
					"hidden": false,
					"id": "autodate2990389176",
					"name": "created",
					"onCreate": true,
					"onUpdate": false,
					"presentable": false,
					"system": false,
					"type": "autodate"
				},
				{
					"hidden": false,
					"id": "autodate3332085495",
					"name": "updated",
					"onCreate": true,
					"onUpdate": true,
					"presentable": false,
					"system": false,
					"type": "autodate"
				}
			],
			"indexes": [
				"CREATE INDEX idx_trail_time_buckets_range ON trail_time_buckets(created_from, created_to)"
			],
			"options": {},
			"system": false
		}`

		trailTimeBucketEntriesJSON := `{
			"id": "pbc_6000010004",
			"listRule": "",
			"viewRule": "",
			"createRule": null,
			"updateRule": null,
			"deleteRule": null,
			"name": "trail_time_bucket_entries",
			"type": "base",
			"fields": [
				{
					"autogeneratePattern": "[a-z0-9]{15}",
					"hidden": false,
					"id": "text3208210256",
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
					"cascadeDelete": true,
					"collectionId": "e864strfxo14pm4",
					"hidden": false,
					"id": "reltrail2",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "trail",
					"presentable": false,
					"required": true,
					"system": false,
					"type": "relation"
				},
				{
					"cascadeDelete": true,
					"collectionId": "pbc_6000010003",
					"hidden": false,
					"id": "reltb001",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "bucket",
					"presentable": false,
					"required": true,
					"system": false,
					"type": "relation"
				},
				{
					"hidden": false,
					"id": "autodate2990389176",
					"name": "created",
					"onCreate": true,
					"onUpdate": false,
					"presentable": false,
					"system": false,
					"type": "autodate"
				},
				{
					"hidden": false,
					"id": "autodate3332085495",
					"name": "updated",
					"onCreate": true,
					"onUpdate": true,
					"presentable": false,
					"system": false,
					"type": "autodate"
				}
			],
			"indexes": [
				"CREATE UNIQUE INDEX idx_trail_time_bucket_entries_unique ON trail_time_bucket_entries(trail, bucket)",
				"CREATE INDEX idx_trail_time_bucket_entries_trail ON trail_time_bucket_entries(trail)",
				"CREATE INDEX idx_trail_time_bucket_entries_bucket ON trail_time_bucket_entries(bucket)"
			],
			"options": {},
			"system": false
		}`

		payloads := []string{quadNodesJSON, trailQuadNodesJSON, trailTimeBucketsJSON, trailTimeBucketEntriesJSON}
		for _, payload := range payloads {
			collection := &core.Collection{}
			if err := json.Unmarshal([]byte(payload), &collection); err != nil {
				return err
			}
			if err := app.Save(collection); err != nil {
				return err
			}
		}

		if !util.BucketsEnabled() {
			app.Logger().Info("Trail buckets disabled; skipping shard backfill")
			return nil
		}

		const pageSize int64 = 100
		var page int64

		for {
			trailsPage := []*core.Record{}
			if err := app.RecordQuery("trails").
				Limit(pageSize).
				Offset(page * pageSize).
				All(&trailsPage); err != nil {
				return fmt.Errorf("failed to query trails: %w", err)
			}
			if len(trailsPage) == 0 {
				break
			}

			for _, trail := range trailsPage {
				if _, err := util.UpdateTrailBoundingBox(app, trail); err != nil {
					app.Logger().Warn(fmt.Sprintf("Unable to update trail bbox for %s: %v", trail.Id, err))
				}
				if err := util.AssignTrailShardsForced(app, trail); err != nil {
					return fmt.Errorf("failed to assign shards for trail %s: %w", trail.Id, err)
				}
			}

			if err := util.IndexTrails(app, trailsPage, client); err != nil {
				app.Logger().Warn(fmt.Sprintf("Unable to index trails page %d: %v", page, err))
			}

			page++
		}

		app.Logger().Info(fmt.Sprintf("Successfully built global trail shards for %d pages", page))
		return nil
	}, func(app core.App) error {
		collections := []string{
			"trail_time_bucket_entries",
			"trail_time_buckets",
			"trail_quad_nodes",
			"quad_nodes",
		}
		for _, name := range collections {
			if err := deleteCollectionIfExists(app, name); err != nil {
				return err
			}
		}
		return nil
	})
}

func deleteCollectionIfExists(app core.App, name string) error {
	collection, err := app.FindCollectionByNameOrId(name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(strings.ToLower(err.Error()), "not found") {
			return nil
		}
		return err
	}
	return app.Delete(collection)
}
