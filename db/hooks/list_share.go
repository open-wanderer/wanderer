package hooks

import (
	"pocketbase/federation"

	"github.com/pocketbase/pocketbase/core"
)

// Search projection runs on successful record mutations, including internal
// writes. The request hook validates sharing and sends the federation announcement.
func CreateListShareHandler() func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		if err := ensureShareAllowed(e, "lists", "list"); err != nil {
			return err
		}

		if err := e.Next(); err != nil {
			return err
		}
		return federation.CreateAnnounceActivity(e.App, e.Record, federation.ListAnnounceType)
	}
}
