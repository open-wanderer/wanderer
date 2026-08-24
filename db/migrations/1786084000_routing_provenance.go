package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const trailRoutingProvenanceFieldJSON = `{
	"hidden": false,
	"id": "jsonrouteprovx",
	"maxSize": 262144,
	"name": "routing_provenance",
	"presentable": false,
	"required": false,
	"system": false,
	"type": "json"
}`

func init() {
	m.Register(func(app core.App) error {
		return ensureRoutingProvenanceField(app)
	}, func(app core.App) error {
		if trails, err := app.FindCollectionByNameOrId("trails"); err == nil {
			trails.Fields.RemoveByName("routing_provenance")
			if err := app.Save(trails); err != nil {
				return err
			}
		}
		return nil
	})
}

func ensureRoutingProvenanceField(app core.App) error {
	trails, err := app.FindCollectionByNameOrId("trails")
	if err != nil {
		return err
	}
	if existing := trails.Fields.GetByName("routing_provenance"); existing != nil {
		field, ok := existing.(*core.JSONField)
		if !ok || field.MaxSize != 256*1024 {
			return fmt.Errorf("trails.routing_provenance has an incompatible schema")
		}
		return nil
	}
	if err := trails.Fields.AddMarshaledJSONAt(len(trails.Fields), []byte(trailRoutingProvenanceFieldJSON)); err != nil {
		return err
	}
	return app.Save(trails)
}
