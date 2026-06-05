package hooks

import (
	"fmt"
	"os"

	"github.com/pocketbase/pocketbase/core"
)

func CreateWaypointHandler() func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		err := e.Next()
		if err != nil {
			return err
		}

		// add local iri
		origin := os.Getenv("ORIGIN")
		if origin == "" {
			return fmt.Errorf("ORIGIN not set")
		}
		if e.Record.GetString("iri") == "" {
			e.Record.Set("iri", fmt.Sprintf("%s/api/v1/waypoint/%s", origin, e.Record.Id))
		}

		return e.App.UnsafeWithoutHooks().Save(e.Record)
	}
}
