package util

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type FeedType string

const (
	ListFeed      FeedType = "list"
	TrailFeed     FeedType = "trail"
	SummitLogFeed FeedType = "summit_log"
)

// InsertIntoFeed adds an item to an actor's feed and returns the feed row. An
// item appears at most once per actor; if a concurrent insert of the same item
// wins, its row is returned.
func InsertIntoFeed(app core.App, actorId string, authorId string, itemId string, feedType FeedType) (*core.Record, error) {
	// an item appears at most once in an actor's feed — repeated Update
	// activities for an already-known item must not create duplicate entries
	const filter = "actor = {:actor} && item = {:item}"
	params := dbx.Params{"actor": actorId, "item": itemId}

	existing, err := app.FindFirstRecordByFilter("feed", filter, params)
	if err == nil && existing != nil {
		return existing, nil
	}

	collection, err := app.FindCollectionByNameOrId("feed")
	if err != nil {
		return nil, err
	}

	record := core.NewRecord(collection)

	record.Set("actor", actorId)
	record.Set("author", authorId)
	record.Set("item", itemId)
	record.Set("type", string(feedType))

	if saveErr := app.Save(record); saveErr != nil {
		// a concurrent insert of the same item won
		if winner, findErr := app.FindFirstRecordByFilter("feed", filter, params); findErr == nil && winner != nil {
			return winner, nil
		}
		return record, saveErr
	}

	return record, nil
}

func DeleteFromFeed(app core.App, itemId string) error {
	records, err := app.FindAllRecords("feed", dbx.HashExp{"item": itemId})
	if err != nil {
		return err
	}

	for _, record := range records {
		if err := app.Delete(record); err != nil {
			return err
		}
	}

	return nil
}
