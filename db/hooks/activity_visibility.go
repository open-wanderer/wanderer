package hooks

import (
	"pocketbase/federation"

	"github.com/pocketbase/pocketbase/core"
)

// syncActivityVisibility hides the recorded activities of a trail or list,
// and of the replies to it, when it goes private, and shows them again when
// it goes public. What was already delivered elsewhere stays there; only the
// public listing here changes.
func syncActivityVisibility(e *core.RecordEvent) error {
	public := e.Record.GetBool("public")
	if e.Record.Original().GetBool("public") == public {
		return nil
	}
	return federation.SetObjectActivitiesHidden(e.App, e.Record.GetString("iri"), !public)
}
