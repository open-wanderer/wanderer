package migrations

import (
	"fmt"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Anonymous requests for a remote follower list stored every listed actor
// (crawlers walked whole collections). Remote person actors nothing points at
// are a cache and are fetched again on demand. The raw DELETE neither
// cascades nor checks relations, so an actor any relation or recorded
// activity still names is kept; deleting it would leave dangling ids.
func init() {
	m.Register(upPurgeUnreferencedRemoteActors1791713197, nil)
}

// A temp table lives on one connection; the transaction pins it.
func upPurgeUnreferencedRemoteActors1791713197(app core.App) error {
	return app.RunInTransaction(purgeUnreferencedRemoteActors)
}

func purgeUnreferencedRemoteActors(app core.App) error {
	actors, err := app.FindCollectionByNameOrId("activitypub_actors")
	if err != nil {
		return err
	}
	collections, err := app.FindAllCollections(core.CollectionTypeBase, core.CollectionTypeAuth)
	if err != nil {
		return err
	}

	if _, err := app.DB().NewQuery("CREATE TEMP TABLE referenced_actors (id TEXT PRIMARY KEY) WITHOUT ROWID").Execute(); err != nil {
		return err
	}
	defer app.DB().NewQuery("DROP TABLE IF EXISTS temp.referenced_actors").Execute()

	for _, collection := range collections {
		for _, field := range collection.Fields {
			relation, ok := field.(*core.RelationField)
			if !ok || relation.CollectionId != actors.Id {
				continue
			}
			table, column := "{{"+collection.Name+"}}", "[["+relation.Name+"]]"
			query := fmt.Sprintf("INSERT OR IGNORE INTO referenced_actors SELECT %s FROM %s WHERE %s != ''", column, table, column)
			if relation.IsMultiple() {
				query = fmt.Sprintf("INSERT OR IGNORE INTO referenced_actors SELECT j.value FROM %s, json_each(CASE WHEN json_valid(%s) THEN %s ELSE '[]' END) j", table, column, column)
			}
			if _, err := app.DB().NewQuery(query).Execute(); err != nil {
				return err
			}
		}
	}

	// Delete fanout reaches mentioned actors and revoked shares only through
	// recorded activities: their cc lists inboxes, an Announce's to lists IRIs.
	// Each address is joined on the iri and inbox indexes; the temp store is
	// in memory, so the addresses themselves are not collected.
	for _, address := range []string{"iri", "inbox"} {
		queries := []string{fmt.Sprintf("INSERT OR IGNORE INTO referenced_actors SELECT aa.id FROM activitypub_activities ac INNER JOIN activitypub_actors aa ON aa.%s = ac.actor WHERE ac.actor != ''", address)}
		for _, column := range []string{"[[to]]", "cc"} {
			queries = append(queries, fmt.Sprintf("INSERT OR IGNORE INTO referenced_actors SELECT aa.id FROM activitypub_activities ac, json_each(CASE WHEN json_valid(ac.%s) THEN ac.%s ELSE '[]' END) r INNER JOIN activitypub_actors aa ON aa.%s = r.value WHERE r.value != ''", column, column, address))
		}
		for _, query := range queries {
			if _, err := app.DB().NewQuery(query).Execute(); err != nil {
				return err
			}
		}
	}

	const batchSize = 5000
	var total int64
	var batches int
	for {
		result, err := app.DB().NewQuery(`
			DELETE FROM activitypub_actors
			WHERE rowid IN (
				SELECT rowid
				FROM activitypub_actors
				WHERE is_local = 0
				  AND actor_type != 'instance'
				  AND id NOT IN (SELECT id FROM referenced_actors)
				ORDER BY rowid
				LIMIT {:limit}
			)
		`).Bind(dbx.Params{"limit": batchSize}).Execute()
		if err != nil {
			return err
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			break
		}
		total += affected
		batches++
		if batches%20 == 0 {
			app.Logger().Info("Purging unreferenced remote actors", "deleted", total, "batches", batches)
		}
	}
	if total > 0 {
		app.Logger().Info("Purged unreferenced remote actors", "deleted", total, "batches", batches)
	}
	return nil
}
