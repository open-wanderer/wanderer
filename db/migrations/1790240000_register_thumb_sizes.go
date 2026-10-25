package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// PocketBase serves a ?thumb= size only when it is the built-in 100x100 default
// or is listed in the file field's thumbs option. Every other size silently
// falls back to the original file, so the sizes the web UI requests have to be
// registered here to take effect.
var thumbSizes = map[string]map[string][]string{
	"users":       {"avatar": {"300x300"}},
	"lists":       {"avatar": {"300x300", "600x0"}},
	"waypoints":   {"photos": {"300x300", "600x0"}},
	"summit_logs": {"photos": {"300x300", "600x0"}},
}

var previousThumbSizes = map[string]map[string][]string{
	"users":       {"avatar": {}},
	"lists":       {"avatar": {}},
	"waypoints":   {"photos": {}},
	"summit_logs": {"photos": {}},
}

func init() {
	m.Register(func(app core.App) error {
		return applyThumbSizes(app, thumbSizes)
	}, func(app core.App) error {
		return applyThumbSizes(app, previousThumbSizes)
	})
}

func applyThumbSizes(app core.App, sizes map[string]map[string][]string) error {
	for collectionName, fields := range sizes {
		collection, err := app.FindCollectionByNameOrId(collectionName)
		if err != nil {
			return err
		}
		for fieldName, thumbs := range fields {
			field, ok := collection.Fields.GetByName(fieldName).(*core.FileField)
			if !ok {
				return fmt.Errorf("%s.%s is not a file field", collectionName, fieldName)
			}
			field.Thumbs = thumbs
		}
		if err := app.Save(collection); err != nil {
			return err
		}
	}

	return nil
}
