package hooks

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
)

// UpdateShareTargetHandler keeps a share bound to its authorized target.
// PocketBase checks update rules against the stored record before loading
// request data, so changing the target could grant access to another owner's
// private trail or list.
func UpdateShareTargetHandler(objectField string) func(*core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		if e.Record.GetString(objectField) != e.Record.Original().GetString(objectField) {
			return e.BadRequestError(fmt.Sprintf("The %s of an existing share cannot be changed.", objectField), nil)
		}
		return e.Next()
	}
}
