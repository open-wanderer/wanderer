package hooks

import (
	"pocketbase/federation"

	"github.com/pocketbase/pocketbase/core"
)

func CreateFollowHandler() func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		e.Next()
		federation.CreateFollowActivity(e.App, e.Record)

		return nil
	}
}

func DeleteFollowHandler() func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		federation.CreateUnfollowActivity(e.App, e.Record)

		return e.Next()
	}
}
