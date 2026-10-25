package hooks

import (
	"pocketbase/util"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"
)

func UpdateTagIndexHandler(client meilisearch.ServiceManager) func(*core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		if e.Record.GetString("name") == e.Record.Original().GetString("name") {
			return nil
		}
		return util.UpdateTagReferences(e.App, e.Record.Id, client)
	}
}
