package migrations

import (
	"encoding/json"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		jsonData := `{
			"createRule": null,
			"deleteRule": null,
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
					"id": "relation420001001",
					"maxSelect": 1,
					"minSelect": 0,
					"name": "trail",
					"presentable": false,
					"required": true,
					"system": false,
					"type": "relation"
				},
				{
					"hidden": false,
					"id": "select420001002",
					"maxSelect": 1,
					"name": "provider",
					"presentable": false,
					"required": true,
					"system": false,
					"type": "select",
					"values": [
						"strava",
						"komoot",
						"hammerhead"
					]
				},
				{
					"autogeneratePattern": "",
					"hidden": false,
					"id": "text420001003",
					"max": 255,
					"min": 1,
					"name": "external_id",
					"pattern": "",
					"presentable": false,
					"primaryKey": false,
					"required": true,
					"system": false,
					"type": "text"
				},
				{
					"hidden": false,
					"id": "autodate420001004",
					"name": "created",
					"onCreate": true,
					"onUpdate": false,
					"presentable": false,
					"system": false,
					"type": "autodate"
				},
				{
					"hidden": false,
					"id": "autodate420001005",
					"name": "updated",
					"onCreate": true,
					"onUpdate": true,
					"presentable": false,
					"system": false,
					"type": "autodate"
				}
			],
			"id": "pbc_420001000",
			"indexes": [
				"CREATE UNIQUE INDEX ` + "`" + `idx_trail_external_reference_provider_external_id` + "`" + ` ON ` + "`" + `trail_external_reference` + "`" + ` (` + "`" + `provider` + "`" + `, ` + "`" + `external_id` + "`" + `)",
				"CREATE UNIQUE INDEX ` + "`" + `idx_trail_external_reference_trail_provider_external_id` + "`" + ` ON ` + "`" + `trail_external_reference` + "`" + ` (` + "`" + `trail` + "`" + `, ` + "`" + `provider` + "`" + `, ` + "`" + `external_id` + "`" + `)"
			],
			"listRule": null,
			"name": "trail_external_reference",
			"system": false,
			"type": "base",
			"updateRule": null,
			"viewRule": null
		}`

		collection := &core.Collection{}
		if err := json.Unmarshal([]byte(jsonData), &collection); err != nil {
			return err
		}
		if err := app.Save(collection); err != nil {
			return err
		}

		referenceCollection, err := app.FindCollectionByNameOrId("trail_external_reference")
		if err != nil {
			return err
		}

		trails, err := app.FindRecordsByFilter(
			"trails",
			"external_provider != '' && external_id != ''",
			"",
			-1,
			0,
			nil,
		)
		if err != nil {
			return err
		}

		for _, trail := range trails {
			provider := trail.GetString("external_provider")
			externalID := trail.GetString("external_id")
			if provider == "" || externalID == "" {
				continue
			}

			existing, err := app.FindRecordsByFilter(
				"trail_external_reference",
				"provider={:provider} && external_id={:external_id}",
				"",
				1,
				0,
				dbx.Params{
					"provider":    provider,
					"external_id": externalID,
				},
			)
			if err != nil {
				return err
			}
			if len(existing) > 0 {
				app.Logger().Warn("Skipping duplicate trail external reference during migration", "provider", provider, "external_id", externalID, "trail", trail.Id)
				continue
			}

			record := core.NewRecord(referenceCollection)
			record.Load(map[string]any{
				"trail":       trail.Id,
				"provider":    provider,
				"external_id": externalID,
			})
			if err := app.Save(record); err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("pbc_420001000")
		if err != nil {
			return err
		}

		return app.Delete(collection)
	})
}
