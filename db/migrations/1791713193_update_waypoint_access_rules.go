package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
	"regexp"
)

func init() {
	m.Register(func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("waypoints")
		if err != nil {
			return err
		}
		leading := regexp.MustCompile(`(^|[ (\n])author\s*=\s*@request\.auth\.id\s*\|\|\s*`)
		trailing := regexp.MustCompile(`\s*\|\|\s*author\s*=\s*@request\.auth\.id([ )\n]|$)`)
		for _, rule := range []**string{&collection.ListRule, &collection.ViewRule, &collection.UpdateRule, &collection.DeleteRule} {
			if *rule == nil {
				continue
			}
			updated := leading.ReplaceAllString(**rule, "${1}")
			updated = trailing.ReplaceAllString(updated, "${1}")
			*rule = types.Pointer(updated)
		}
		return app.Save(collection)
	}, func(core.App) error { return nil })
}
