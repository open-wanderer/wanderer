package hooks

import (
	"database/sql"
	"errors"
	"slices"
	"sort"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func UpdateTrailShareIndexHandler(client meilisearch.ServiceManager) func(*core.RecordEvent) error {
	return updateShareIndexHandler("trail_share", "trail", "trails", client)
}

func UpdateListShareIndexHandler(client meilisearch.ServiceManager) func(*core.RecordEvent) error {
	return updateShareIndexHandler("list_share", "list", "lists", client)
}

func updateShareIndexHandler(collection, relation, index string, client meilisearch.ServiceManager) func(*core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		ids := []string{e.Record.GetString(relation)}
		if previous := e.Record.Original().GetString(relation); previous != "" && previous != ids[0] {
			ids = append(ids, previous)
		}
		for _, id := range ids {
			if id == "" {
				continue
			}
			// A cascade from deleting the parent must not recreate an index stub.
			if _, err := e.App.FindRecordById(index, id); errors.Is(err, sql.ErrNoRows) {
				continue
			} else if err != nil {
				return err
			}
			shares, err := e.App.FindAllRecords(collection, dbx.HashExp{relation: id})
			if err != nil {
				return err
			}
			actors := []string{}
			for _, share := range shares {
				if actor := share.GetString("actor"); actor != "" {
					actors = append(actors, actor)
				}
			}
			sort.Strings(actors)
			patch := map[string]any{"id": id, "shares": slices.Compact(actors)}
			if _, err := client.Index(index).UpdateDocuments([]map[string]any{patch}, nil); err != nil {
				return err
			}
		}
		return nil
	}
}
