package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/util"
)

// searchIndexVersionsFile records, per index, the document version the last
// full rebuild was made from.
const searchIndexVersionsFile = "meilisearch_index_versions.json"

const searchTaskCancelTimeout = 30 * time.Second

var pendingSearchTaskStatuses = []meilisearch.TaskStatus{
	meilisearch.TaskStatusEnqueued,
	meilisearch.TaskStatusProcessing,
}

// documentSearchTaskTypes limits the queued-task check and the cancel to
// document writes. initMeilisearchConfig queues a settings update on every
// start, which says nothing about the documents and must not be canceled.
var documentSearchTaskTypes = []meilisearch.TaskType{
	meilisearch.TaskTypeDocumentAdditionOrUpdate,
	meilisearch.TaskTypeDocumentDeletion,
}

type searchIndexSource struct {
	index      string
	collection string
	pageLabel  string
	pageSize   int64
	add        func(records []*core.Record) error
}

// initMeilisearchDocuments rebuilds each search index that is missing,
// incomplete or built from an older document version.
func initMeilisearchDocuments(app core.App, client meilisearch.ServiceManager) error {
	sources := []searchIndexSource{
		{
			index: "trails", collection: "trails", pageLabel: "trails", pageSize: 100,
			add: func(records []*core.Record) error { return util.IndexTrails(app, records, client) },
		},
		{
			index: "lists", collection: "lists", pageLabel: "list", pageSize: 100,
			add: func(records []*core.Record) error { return util.IndexLists(app, records, client) },
		},
		{
			// Actor documents are a few hundred bytes each, so they go in
			// larger batches to keep the number of queued tasks down.
			index: "actors", collection: "activitypub_actors", pageLabel: "actor", pageSize: 1000,
			add: func(records []*core.Record) error { return util.IndexActors(records, client) },
		},
	}

	built := readSearchIndexVersions(app)
	var errs []error
	for _, source := range sources {
		version := util.SearchDocumentVersions[source.index]
		if built[source.index] == version {
			current, err := searchIndexCurrent(app, client, source)
			if err != nil {
				app.Logger().Warn(fmt.Sprintf("Unable to check search index %s, rebuilding it: %v", source.index, err))
			} else if current {
				app.Logger().Info(fmt.Sprintf("Search index %s is current, skipping rebuild", source.index))
				continue
			}
		}

		app.Logger().Info(fmt.Sprintf("Rebuilding search index %s", source.index))
		// Drop the entry while the rebuild runs, so a start that interrupts
		// it does not take the batches it left queued for a current index.
		delete(built, source.index)
		if err := writeSearchIndexVersions(app, built); err != nil {
			app.Logger().Warn(fmt.Sprintf("Unable to record search index versions: %v", err))
		}
		if err := rebuildSearchIndex(app, client, source); err != nil {
			errs = append(errs, fmt.Errorf("rebuild search index %s: %w", source.index, err))
			continue
		}
		built[source.index] = version
		if err := writeSearchIndexVersions(app, built); err != nil {
			app.Logger().Warn(fmt.Sprintf("Unable to record search index versions: %v", err))
		}
	}

	return errors.Join(errs...)
}

// searchIndexCurrent reports whether an index built from the current document
// version can be kept. Queued tasks mean an earlier rebuild or update is still
// being applied, so its document count is not final yet and the index is left
// to finish; otherwise it has to hold one document per record.
func searchIndexCurrent(app core.App, client meilisearch.ServiceManager, source searchIndexSource) (bool, error) {
	pending, err := pendingSearchTasks(client, source.index)
	if err != nil {
		return false, err
	}
	if pending > 0 {
		return true, nil
	}

	stats, err := client.Index(source.index).GetStats()
	if err != nil {
		return false, err
	}
	records, err := app.CountRecords(source.collection)
	if err != nil {
		return false, err
	}
	if stats.NumberOfDocuments != records {
		// A record that fails to index keeps this from ever matching, which
		// turns the skip off for the index; the numbers make that visible.
		app.Logger().Info(fmt.Sprintf(
			"Search index %s holds %d documents for %d records",
			source.index, stats.NumberOfDocuments, records,
		))
		return false, nil
	}
	return true, nil
}

func rebuildSearchIndex(app core.App, client meilisearch.ServiceManager, source searchIndexSource) error {
	// A rebuild interrupted by a restart leaves its batches queued; cancel
	// them so they do not run ahead of this one.
	if err := cancelPendingSearchTasks(client, source.index); err != nil {
		return err
	}

	if _, err := client.Index(source.index).DeleteAllDocuments(nil); err != nil {
		return err
	}

	var page int64
	for {
		records := []*core.Record{}
		err := app.RecordQuery(source.collection).
			Limit(source.pageSize).
			Offset(page * source.pageSize).
			All(&records)
		if err != nil {
			return err
		}
		if len(records) == 0 {
			return nil
		}

		if err := source.add(records); err != nil {
			// Omit this page from the rebuild and advance to the next one.
			app.Logger().Warn(fmt.Sprintf("Unable to index %s page %d: %v", source.pageLabel, page, err))
		}

		page++
	}
}

func pendingSearchTasks(client meilisearch.ServiceManager, index string) (int64, error) {
	tasks, err := client.GetTasks(&meilisearch.TasksQuery{
		IndexUIDS: []string{index},
		Statuses:  pendingSearchTaskStatuses,
		Types:     documentSearchTaskTypes,
		Limit:     1,
	})
	if err != nil {
		return 0, err
	}
	return tasks.Total, nil
}

func cancelPendingSearchTasks(client meilisearch.ServiceManager, index string) error {
	pending, err := pendingSearchTasks(client, index)
	if err != nil || pending == 0 {
		return err
	}

	task, err := client.CancelTasks(&meilisearch.CancelTasksQuery{
		IndexUIDS: []string{index},
		Statuses:  pendingSearchTaskStatuses,
		Types:     documentSearchTaskTypes,
	})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), searchTaskCancelTimeout)
	defer cancel()
	_, err = client.WaitForTaskWithContext(ctx, task.TaskUID, 500*time.Millisecond)
	return err
}

func readSearchIndexVersions(app core.App) map[string]int {
	versions := map[string]int{}
	data, err := os.ReadFile(filepath.Join(app.DataDir(), searchIndexVersionsFile))
	if errors.Is(err, os.ErrNotExist) {
		return versions
	}
	if err == nil {
		err = json.Unmarshal(data, &versions)
	}
	if err != nil {
		app.Logger().Warn(fmt.Sprintf("Unable to read search index versions, rebuilding every index: %v", err))
		return map[string]int{}
	}
	return versions
}

// writeSearchIndexVersions replaces the file through a rename, so a crash
// mid-write cannot leave a truncated file behind.
func writeSearchIndexVersions(app core.App, versions map[string]int) error {
	data, err := json.Marshal(versions)
	if err != nil {
		return err
	}
	path := filepath.Join(app.DataDir(), searchIndexVersionsFile)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
