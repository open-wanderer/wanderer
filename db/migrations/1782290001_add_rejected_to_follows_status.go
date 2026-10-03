package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("8obn1ukumze565i")
		if err != nil {
			return err
		}

		field := collection.Fields.GetByName("status")
		if field == nil {
			return fmt.Errorf("status field not found in follows collection")
		}

		selectField, ok := field.(*core.SelectField)
		if !ok {
			return fmt.Errorf("status field is not a SelectField")
		}

		// Idempotent: only append "rejected" if not already present
		for _, v := range selectField.Values {
			if v == "rejected" {
				return nil
			}
		}
		selectField.Values = append(selectField.Values, "rejected")

		return app.Save(collection)
	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("8obn1ukumze565i")
		if err != nil {
			return nil
		}

		field := collection.Fields.GetByName("status")
		if field == nil {
			return nil
		}

		selectField, ok := field.(*core.SelectField)
		if !ok {
			return nil
		}

		filtered := make([]string, 0, len(selectField.Values))
		for _, v := range selectField.Values {
			if v != "rejected" {
				filtered = append(filtered, v)
			}
		}
		selectField.Values = filtered

		return app.Save(collection)
	})
}
