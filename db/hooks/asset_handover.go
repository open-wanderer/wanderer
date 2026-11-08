package hooks

import (
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/core"
)

func HandOverContributedAssetsOnActorDelete() func(*core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		originalApp := e.App
		return originalApp.RunInTransaction(func(tx core.App) error {
			e.App = tx
			defer func() { e.App = originalApp }()
			if err := util.HandOverActorAssets(tx, e.Record.Id); err != nil {
				return err
			}
			return e.Next()
		})
	}
}
