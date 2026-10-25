package hooks

import (
	"pocketbase/services/routeimage"

	"github.com/pocketbase/pocketbase/core"
)

// RoutePlaceholderHandler gives trails and summit logs without photos an image
// of their route. A failed render never fails the save.
func RoutePlaceholderHandler() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		if err := routeimage.Apply(e.App, e.Record); err != nil {
			e.App.Logger().Error("could not generate route placeholder",
				"collection", e.Record.Collection().Name, "id", e.Record.Id, "error", err)
		}
		return e.Next()
	}
}
