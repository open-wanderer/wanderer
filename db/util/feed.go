package util

import (
	"sync"

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
// item appears at most once per actor.
//
// Nontransactional callers serialize a missing item through its insertion.
// A transaction already holds the database writer and must not wait here.
var feedEntryCreation sync.Mutex

func InsertIntoFeed(app core.App, actorId string, authorId string, itemId string, feedType FeedType) (*core.Record, error) {
	lookup := func() *core.Record {
		existing, err := app.FindFirstRecordByFilter(
			"feed",
			"actor = {:actor} && item = {:item}",
			dbx.Params{"actor": actorId, "item": itemId},
		)
		if err == nil {
			return existing
		}
		return nil
	}
	if existing := lookup(); existing != nil {
		return existing, nil
	}
	if !app.IsTransactional() {
		feedEntryCreation.Lock()
		defer feedEntryCreation.Unlock()
		if existing := lookup(); existing != nil {
			return existing, nil
		}
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
		// A concurrent insert of the same item won, e.g. inside a transaction
		// or in another process; the unique (actor, item) index refused ours.
		if winner := lookup(); winner != nil {
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
