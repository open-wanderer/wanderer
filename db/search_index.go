package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"pocketbase/util"
)

// searchIndexVersionsFile records, per index, the document version the last
// complete rebuild was made from and the Meilisearch index it went into.
const searchIndexVersionsFile = "meilisearch_index_versions.json"

// searchIndexState is what searchIndexVersionsFile records for one index.
// CreatedAt identifies the Meilisearch index: one recreated after its data was
// wiped gets a new creation time, even when live writes have already put a
// few documents into it.
type searchIndexState struct {
	Version   int    `json:"version"`
	CreatedAt string `json:"created_at"`
}

// searchRepairTaskTimeout bounds the wait for one repair batch. A batch that
// waits this long sits behind a backlog, and the repair gives up on the index
// for the night instead of waiting the same again per batch.
var searchRepairTaskTimeout = 5 * time.Minute

const (
	searchTaskCancelTimeout = 30 * time.Second
	// searchRepairIDPageSize is how many document ids the orphan scan reads
	// from Meilisearch per request.
	searchRepairIDPageSize = 10000
)

// searchQueueDrainTimeout and searchQueuePollInterval bound how long the
// repair waits for an index's queued document tasks before skipping it. A
// live write clears in milliseconds; only a backlog makes the repair skip.
var (
	searchQueueDrainTimeout = 10 * time.Minute
	searchQueuePollInterval = 5 * time.Second
)

var pendingSearchTaskStatuses = []meilisearch.TaskStatus{
	meilisearch.TaskStatusEnqueued,
	meilisearch.TaskStatusProcessing,
}

// documentSearchTaskTypes limits the queued-task check and the cancel to
// document writes. initMeilisearchConfig queues a settings update on every
// start, which must not be canceled.
var documentSearchTaskTypes = []meilisearch.TaskType{
	meilisearch.TaskTypeDocumentAdditionOrUpdate,
	meilisearch.TaskTypeDocumentDeletion,
}

// searchIndexLock keeps the startup rebuild and the nightly repair from
// writing the same indexes at the same time.
var searchIndexLock sync.Mutex

type searchIndexSource struct {
	index      string
	collection string
	pageLabel  string
	pageSize   int64
	document   func(r *core.Record) (map[string]any, error)
	// repairFields limits the nightly comparison to these fields; nil
	// compares the whole document.
	repairFields []string
}

func searchIndexSources(app core.App) []searchIndexSource {
	return []searchIndexSource{
		{
			index: "trails", collection: "trails", pageLabel: "trails", pageSize: 100,
			document: func(r *core.Record) (map[string]any, error) { return util.TrailSearchDocument(app, r) },
		},
		{
			index: "lists", collection: "lists", pageLabel: "list", pageSize: 100,
			document: func(r *core.Record) (map[string]any, error) { return util.ListSearchDocument(app, r) },
		},
		{
			// Actor documents hold only the actor's own fields, so they do
			// not drift when other records change. A failed write is the
			// only way one goes stale, and a stale name is the one change
			// that does not repair itself: profiles are opened by handle,
			// and opening one refetches the actor and rewrites its document.
			// Actor documents are a few hundred bytes each, so they go in
			// larger batches to keep the number of queued tasks down.
			index: "actors", collection: "activitypub_actors", pageLabel: "actor", pageSize: 1000,
			document:     util.ActorSearchDocument,
			repairFields: []string{"preferred_username"},
		},
	}
}

// initMeilisearchDocuments rebuilds each search index that was built from an
// older document version, or never completely, or was recreated or emptied in
// Meilisearch since. Every other index is left as it is: rebuilding the actors index of an instance that has
// cached hundreds of thousands of remote actors keeps Meilisearch's single
// task queue busy for hours, and the nightly repair keeps the contents
// current.
func initMeilisearchDocuments(app core.App, client meilisearch.ServiceManager) error {
	searchIndexLock.Lock()
	defer searchIndexLock.Unlock()

	built := readSearchIndexVersions(app)
	var errs []error
	for _, source := range searchIndexSources(app) {
		version := util.SearchDocumentVersions[source.index]
		reason := "its documents were built from an older version"
		if state, ok := built[source.index]; ok && state.Version == version {
			stale, err := searchIndexStale(app, client, source, state)
			if err != nil {
				// Meilisearch may not be up yet. Rebuilding now would fail
				// too, and the nightly repair covers the index meanwhile.
				app.Logger().Warn(fmt.Sprintf("Unable to check search index %s, leaving it for the nightly repair: %v", source.index, err))
				continue
			}
			if stale == "" {
				app.Logger().Info(fmt.Sprintf("Search index %s is current, skipping rebuild", source.index))
				continue
			}
			reason = stale
		}

		app.Logger().Info(fmt.Sprintf("Rebuilding search index %s: %s", source.index, reason))
		complete, err := rebuildSearchIndex(app, client, source, func() {
			// Drop the entry once the index is cleared, so a start that
			// interrupts the rebuild rebuilds again instead of keeping a
			// half-filled index.
			delete(built, source.index)
			if err := writeSearchIndexVersions(app, built); err != nil {
				app.Logger().Warn(fmt.Sprintf("Unable to record search index versions: %v", err))
			}
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("rebuild search index %s: %w", source.index, err))
			continue
		}
		if !complete {
			continue
		}
		createdAt, err := searchIndexCreatedAt(client, source.index)
		if err != nil {
			app.Logger().Warn(fmt.Sprintf("Unable to read search index %s, it will be rebuilt on the next start: %v", source.index, err))
			continue
		}
		built[source.index] = searchIndexState{Version: version, CreatedAt: createdAt}
		if err := writeSearchIndexVersions(app, built); err != nil {
			app.Logger().Warn(fmt.Sprintf("Unable to record search index versions: %v", err))
		}
	}

	return errors.Join(errs...)
}

// searchIndexStale reports why an index built from the current document
// version has to be rebuilt anyway, or "" when it does not: Meilisearch
// recreated it, as after its data was wiped, or it holds no documents although
// its collection has records.
func searchIndexStale(app core.App, client meilisearch.ServiceManager, source searchIndexSource, state searchIndexState) (string, error) {
	createdAt, err := searchIndexCreatedAt(client, source.index)
	if err != nil {
		return "", err
	}
	if createdAt != state.CreatedAt {
		return "Meilisearch recreated the index", nil
	}

	stats, err := client.Index(source.index).GetStats()
	if err != nil {
		return "", err
	}
	if stats.NumberOfDocuments > 0 {
		return "", nil
	}
	records, err := app.CountRecords(source.collection)
	if err != nil {
		return "", err
	}
	if records > 0 {
		return "the index is empty", nil
	}
	return "", nil
}

func searchIndexCreatedAt(client meilisearch.ServiceManager, index string) (string, error) {
	info, err := client.GetIndex(index)
	if err != nil {
		return "", err
	}
	return info.CreatedAt.UTC().Format(time.RFC3339Nano), nil
}

// rebuildSearchIndex replaces every document of an index, calling cleared
// once the old documents are gone. It reports the rebuild complete only when
// Meilisearch accepted every page into its queue; a page it rejects while
// processing, and a record that cannot be turned into a document, are left
// to the nightly repair. Waiting for each page here would hold up the start
// for as long as the actors rebuild takes.
func rebuildSearchIndex(app core.App, client meilisearch.ServiceManager, source searchIndexSource, cleared func()) (bool, error) {
	// A rebuild interrupted by a restart leaves its batches queued; cancel
	// them so they do not run ahead of this one.
	if err := cancelPendingSearchTasks(client, source.index); err != nil {
		return false, err
	}

	if _, err := client.Index(source.index).DeleteAllDocuments(nil); err != nil {
		return false, err
	}
	cleared()

	complete := true
	var page int64
	err := forEachRecordPage(app, source.collection, source.pageSize, func(records []*core.Record) error {
		documents := searchDocuments(app, source, records)
		if len(documents) > 0 {
			if _, err := client.Index(source.index).AddDocuments(documents, nil); err != nil {
				// Omit this page from the rebuild and advance to the next one.
				app.Logger().Warn(fmt.Sprintf("Unable to index %s page %d: %v", source.pageLabel, page, err))
				complete = false
			}
		}
		page++
		return nil
	})
	return complete, err
}

// searchDocuments builds the documents of a page of records, leaving out and
// logging the records that fail.
func searchDocuments(app core.App, source searchIndexSource, records []*core.Record) []map[string]any {
	documents := make([]map[string]any, 0, len(records))
	for _, r := range records {
		document, err := source.document(r)
		if err != nil {
			app.Logger().Warn(fmt.Sprintf("Unable to index %s %s: %v", source.pageLabel, r.Id, err))
			continue
		}
		documents = append(documents, document)
	}
	return documents
}

// forEachRecordPage pages through a collection by id, so records deleted
// while a long rebuild runs do not shift later pages.
func forEachRecordPage(app core.App, collection string, pageSize int64, fn func([]*core.Record) error) error {
	lastID := ""
	for {
		records := []*core.Record{}
		query := app.RecordQuery(collection).OrderBy("id ASC").Limit(pageSize)
		if lastID != "" {
			query = query.AndWhere(dbx.NewExp("id > {:lastID}", dbx.Params{"lastID": lastID}))
		}
		if err := query.All(&records); err != nil {
			return err
		}
		if len(records) == 0 {
			return nil
		}
		if err := fn(records); err != nil {
			return err
		}
		lastID = records[len(records)-1].Id
	}
}

// forEachRecordIDPage is forEachRecordPage for record ids only.
func forEachRecordIDPage(app core.App, collection string, pageSize int64, fn func([]string) error) error {
	lastID := ""
	for {
		var ids []string
		query := app.DB().Select("id").From(collection).OrderBy("id ASC").Limit(pageSize)
		if lastID != "" {
			query = query.AndWhere(dbx.NewExp("id > {:lastID}", dbx.Params{"lastID": lastID}))
		}
		if err := query.Column(&ids); err != nil {
			return err
		}
		if len(ids) == 0 {
			return nil
		}
		if err := fn(ids); err != nil {
			return err
		}
		lastID = ids[len(ids)-1]
	}
}

// reindexSearchTrails rewrites the documents of the given trails, which were
// changed without hooks.
func reindexSearchTrails(app core.App, client meilisearch.ServiceManager, ids []string) error {
	var source searchIndexSource
	for _, candidate := range searchIndexSources(app) {
		if candidate.index == "trails" {
			source = candidate
		}
	}
	for start := 0; start < len(ids); start += int(source.pageSize) {
		end := min(start+int(source.pageSize), len(ids))
		records, err := app.FindRecordsByIds(source.collection, ids[start:end])
		if err != nil {
			return err
		}
		documents := searchDocuments(app, source, records)
		if len(documents) == 0 {
			continue
		}
		if _, err := client.Index(source.index).AddDocuments(documents, nil); err != nil {
			return err
		}
	}
	return nil
}

// runSearchRepair is the nightly search-repair cron job. A run is skipped
// while the startup rebuild or an earlier run still holds the indexes.
func runSearchRepair(app core.App, client meilisearch.ServiceManager) {
	if !searchIndexLock.TryLock() {
		app.Logger().Warn("Search repair skipped: the search indexes are still being written")
		return
	}
	defer searchIndexLock.Unlock()
	// PocketBase runs cron jobs without recovering panics, so one would take
	// the whole server down.
	defer func() {
		if r := recover(); r != nil {
			app.Logger().Error(fmt.Sprintf("Search repair panicked: %v", r))
		}
	}()

	if err := repairSearchIndexes(app, client); err != nil {
		app.Logger().Error(fmt.Sprintf("Search repair failed: %v", err))
	}
}

func repairSearchIndexes(app core.App, client meilisearch.ServiceManager) error {
	var errs []error
	for _, source := range searchIndexSources(app) {
		if err := repairSearchIndex(app, client, source); err != nil {
			errs = append(errs, fmt.Errorf("repair search index %s: %w", source.index, err))
		}
	}
	return errors.Join(errs...)
}

// repairSearchIndex brings an index in line with its collection without
// rebuilding it. Each page of records is compared with the stored documents,
// and only missing or differing documents are written, one batch at a time,
// so live writes queued meanwhile are not held up behind the whole repair.
// Documents whose record no longer exists are removed afterwards. A page that
// fails is logged and skipped, so one bad document cannot stop the repair of
// the rest of the index.
func repairSearchIndex(app core.App, client meilisearch.ServiceManager, source searchIndexSource) error {
	// Document tasks still queued, such as a startup rebuild Meilisearch is
	// working through, would make the stored documents look stale and keep
	// every repair batch waiting behind them.
	drained, err := searchQueueDrained(client, source.index)
	if err != nil {
		return err
	}
	if !drained {
		app.Logger().Warn(fmt.Sprintf("Search repair %s skipped: document tasks are still queued", source.index))
		return nil
	}

	var fields []string
	if source.repairFields != nil {
		fields = append([]string{"id"}, source.repairFields...)
	}

	var errs []error
	rewritten := 0
	err = forEachRecordIDPage(app, source.collection, source.pageSize, func(ids []string) error {
		// The stored documents are read before the records, so a live write
		// landing in between leaves both sides current instead of making the
		// record read first look stale.
		stored, err := storedSearchDocuments(client, source.index, ids, fields)
		if err != nil {
			errs = append(errs, fmt.Errorf("read documents: %w", err))
			return nil
		}
		records, err := app.FindRecordsByIds(source.collection, ids)
		if err != nil {
			return err
		}

		var stale []map[string]any
		built := make(map[string]types.DateTime, len(records))
		for _, r := range records {
			document, err := source.document(r)
			if err != nil {
				app.Logger().Warn(fmt.Sprintf("Unable to index %s %s: %v", source.pageLabel, r.Id, err))
				continue
			}
			current, ok := stored[r.Id]
			if !ok || !searchDocumentMatches(document, current, source.repairFields) {
				stale = append(stale, document)
				built[r.Id] = r.GetDateTime("updated")
			}
		}
		if len(stale) == 0 {
			return nil
		}

		afterSearchRepairBuild()
		stale, err = unchangedSearchDocuments(app, source.collection, stale, built)
		if err != nil {
			return err
		}
		if len(stale) == 0 {
			return nil
		}

		written, err := writeSearchRepairBatch(app, client, source, stale)
		rewritten += written
		if errors.Is(err, errSearchTaskTimeout) {
			return err
		}
		if err != nil {
			errs = append(errs, err)
		}
		return nil
	})
	if err != nil {
		errs = append(errs, err)
	}

	removed := 0
	if !errors.Is(err, errSearchTaskTimeout) {
		removed, err = removeOrphanSearchDocuments(app, client, source)
		if err != nil {
			errs = append(errs, fmt.Errorf("remove orphaned documents: %w", err))
		}
	}

	app.Logger().Info(fmt.Sprintf("Search repair %s: %d documents rewritten, %d removed, %d errors", source.index, rewritten, removed, len(errs)))
	return errors.Join(errs...)
}

// afterSearchRepairBuild runs between building a page's stale documents and
// checking them against the database; tests change records there.
var afterSearchRepairBuild = func() {}

// unchangedSearchDocuments drops the documents whose record was changed or
// deleted after the document was built, which built maps to the record's
// updated time. A live write already brought those documents up to date, and
// writing the older copy over it could, for one, make a trail just made
// private searchable again.
func unchangedSearchDocuments(app core.App, collection string, documents []map[string]any, built map[string]types.DateTime) ([]map[string]any, error) {
	ids := make([]any, 0, len(documents))
	for _, document := range documents {
		ids = append(ids, document["id"])
	}
	var rows []struct {
		ID      string         `db:"id"`
		Updated types.DateTime `db:"updated"`
	}
	err := app.DB().Select("id", "updated").From(collection).Where(dbx.In("id", ids...)).All(&rows)
	if err != nil {
		return nil, err
	}
	current := make(map[string]types.DateTime, len(rows))
	for _, row := range rows {
		current[row.ID] = row.Updated
	}

	unchanged := documents[:0]
	for _, document := range documents {
		id, _ := document["id"].(string)
		updated, ok := current[id]
		if ok && updated.Equal(built[id]) {
			unchanged = append(unchanged, document)
		}
	}
	return unchanged, nil
}

// writeSearchRepairBatch writes a batch of documents and waits for it. When
// Meilisearch rejects the batch, its documents are written one at a time, so
// only the documents it rejects are left out.
func writeSearchRepairBatch(app core.App, client meilisearch.ServiceManager, source searchIndexSource, documents []map[string]any) (int, error) {
	err := writeSearchDocuments(client, source.index, documents)
	if err == nil {
		return len(documents), nil
	}
	if !errors.Is(err, errSearchTaskFailed) || len(documents) == 1 {
		return 0, fmt.Errorf("write documents: %w", err)
	}

	written := 0
	var errs []error
	for _, document := range documents {
		if err := writeSearchDocuments(client, source.index, []map[string]any{document}); err != nil {
			app.Logger().Warn(fmt.Sprintf("Unable to repair %s %v: %v", source.pageLabel, document["id"], err))
			errs = append(errs, fmt.Errorf("write %s %v: %w", source.pageLabel, document["id"], err))
			continue
		}
		written++
	}
	return written, errors.Join(errs...)
}

func writeSearchDocuments(client meilisearch.ServiceManager, index string, documents []map[string]any) error {
	task, err := client.Index(index).AddDocuments(documents, nil)
	if err != nil {
		return err
	}
	return waitForSearchRepairTask(client, task.TaskUID)
}

// searchQueueDrained waits for an index's queued document tasks to finish,
// and reports whether they did within searchQueueDrainTimeout.
func searchQueueDrained(client meilisearch.ServiceManager, index string) (bool, error) {
	deadline := time.Now().Add(searchQueueDrainTimeout)
	for {
		pending, err := pendingSearchTasks(client, index)
		if err != nil {
			return false, err
		}
		if pending == 0 {
			return true, nil
		}
		if time.Now().Add(searchQueuePollInterval).After(deadline) {
			return false, nil
		}
		time.Sleep(searchQueuePollInterval)
	}
}

// storedSearchDocuments fetches the stored documents with the given ids,
// keyed by id. fields limits the fetched fields; nil fetches them all.
func storedSearchDocuments(client meilisearch.ServiceManager, index string, ids []string, fields []string) (map[string]map[string]any, error) {
	var result meilisearch.DocumentsResult
	err := client.Index(index).GetDocuments(&meilisearch.DocumentsQuery{
		Ids:    ids,
		Fields: fields,
		Limit:  int64(len(ids)),
	}, &result)
	if err != nil {
		return nil, err
	}
	var documents []map[string]any
	if err := result.Results.Decode(&documents); err != nil {
		return nil, err
	}

	stored := make(map[string]map[string]any, len(documents))
	for _, document := range documents {
		id, _ := document["id"].(string)
		stored[id] = document
	}
	return stored, nil
}

// searchDocumentMatches compares a freshly built document with the stored
// one, field by field after a JSON round trip, which is the form Meilisearch
// returns documents in. fields limits the comparison; nil compares every
// field of the built document.
func searchDocumentMatches(expected, stored map[string]any, fields []string) bool {
	normalized, err := normalizeSearchDocument(expected)
	if err != nil {
		return false
	}
	if fields == nil {
		for field := range normalized {
			fields = append(fields, field)
		}
	}
	for _, field := range fields {
		storedValue, ok := stored[field]
		if !ok || !reflect.DeepEqual(normalized[field], storedValue) {
			return false
		}
	}
	return true
}

func normalizeSearchDocument(document map[string]any) (map[string]any, error) {
	data, err := json.Marshal(document)
	if err != nil {
		return nil, err
	}
	var normalized map[string]any
	err = json.Unmarshal(data, &normalized)
	return normalized, err
}

// removeOrphanSearchDocuments deletes the documents whose record no longer
// exists. All ids are read before anything is deleted, so the deletes do not
// shift the pages being read.
func removeOrphanSearchDocuments(app core.App, client meilisearch.ServiceManager, source searchIndexSource) (int, error) {
	var orphans []string
	for offset := int64(0); ; offset += searchRepairIDPageSize {
		var result meilisearch.DocumentsResult
		err := client.Index(source.index).GetDocuments(&meilisearch.DocumentsQuery{
			Fields: []string{"id"},
			Limit:  searchRepairIDPageSize,
			Offset: offset,
		}, &result)
		if err != nil {
			return 0, err
		}
		var documents []struct {
			ID string `json:"id"`
		}
		if err := result.Results.Decode(&documents); err != nil {
			return 0, err
		}
		if len(documents) == 0 {
			break
		}

		ids := make([]any, len(documents))
		for i, document := range documents {
			ids[i] = document.ID
		}
		var existing []string
		err = app.DB().Select("id").From(source.collection).Where(dbx.In("id", ids...)).Column(&existing)
		if err != nil {
			return 0, err
		}
		exists := make(map[string]bool, len(existing))
		for _, id := range existing {
			exists[id] = true
		}
		for _, document := range documents {
			if !exists[document.ID] {
				orphans = append(orphans, document.ID)
			}
		}

		if int64(len(documents)) < searchRepairIDPageSize {
			break
		}
	}

	for start := 0; start < len(orphans); start += searchRepairIDPageSize {
		end := min(start+searchRepairIDPageSize, len(orphans))
		task, err := client.Index(source.index).DeleteDocuments(orphans[start:end], nil)
		if err != nil {
			return 0, err
		}
		if err := waitForSearchRepairTask(client, task.TaskUID); err != nil {
			return 0, err
		}
	}
	return len(orphans), nil
}

// errSearchTaskFailed marks a task Meilisearch processed and rejected, as
// opposed to one that could not be waited for.
var errSearchTaskFailed = errors.New("search task failed")

// errSearchTaskTimeout marks a task that did not finish within
// searchRepairTaskTimeout.
var errSearchTaskTimeout = errors.New("search task did not finish in time")

func waitForSearchRepairTask(client meilisearch.ServiceManager, taskUID int64) error {
	ctx, cancel := context.WithTimeout(context.Background(), searchRepairTaskTimeout)
	defer cancel()
	task, err := client.WaitForTaskWithContext(ctx, taskUID, time.Second)
	if ctx.Err() != nil {
		return fmt.Errorf("%w: task %d", errSearchTaskTimeout, taskUID)
	}
	if err != nil {
		return err
	}
	if task.Status == meilisearch.TaskStatusFailed {
		return fmt.Errorf("%w: task %d: %s", errSearchTaskFailed, taskUID, task.Error.Message)
	}
	return nil
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

func readSearchIndexVersions(app core.App) map[string]searchIndexState {
	versions := map[string]searchIndexState{}
	data, err := os.ReadFile(filepath.Join(app.DataDir(), searchIndexVersionsFile))
	if errors.Is(err, os.ErrNotExist) {
		return versions
	}
	if err == nil {
		err = json.Unmarshal(data, &versions)
	}
	if err != nil {
		app.Logger().Warn(fmt.Sprintf("Unable to read search index versions, rebuilding every index: %v", err))
		return map[string]searchIndexState{}
	}
	return versions
}

// writeSearchIndexVersions replaces the file through a rename, so a crash
// mid-write cannot leave a truncated file behind.
func writeSearchIndexVersions(app core.App, versions map[string]searchIndexState) error {
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
