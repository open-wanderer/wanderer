package routes

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"pocketbase/util"
)

// PocketBase caps reverse-relation expands at this size. A saturated response
// does not establish that an unlisted child has been removed.
const remoteChildExpandLimit = 1000

func remoteTrailChildIDs(app core.App, data map[string]any, origin, relation, collection string) map[string]struct{} {
	expand, _ := data["expand"].(map[string]any)
	children, present := expand[relation].([]any)
	if !present || len(children) >= remoteChildExpandLimit {
		return nil
	}
	listed := make(map[string]struct{}, len(children))
	kind := map[string]string{"waypoints": "waypoint", "summit_logs": "summit-log"}[collection]
	for _, value := range children {
		raw, ok := value.(map[string]any)
		if !ok {
			return nil
		}
		id, _ := raw["id"].(string)
		if strings.TrimSpace(id) == "" {
			return nil
		}
		iri, _ := raw["iri"].(string)
		if iri == "" {
			iri = fmt.Sprintf("%s/api/v1/%s/%s", origin, kind, id)
		} else if !remoteSyncIRIMatchesOrigin(iri, iri) {
			return nil
		}
		if remoteSyncIRIMatchesOrigin(iri, origin) && !remoteSyncIDFitsCollection(app, collection, id) {
			return nil
		}
		listed[iri] = struct{}{}
	}
	return listed
}

func storedRemoteTrailChildIDs(app core.App, trail, origin, collection string) (map[string]struct{}, error) {
	stored := map[string]struct{}{}
	if trail == "" {
		return stored, nil
	}
	col, err := app.FindCollectionByNameOrId(collection)
	if errors.Is(err, sql.ErrNoRows) {
		return stored, nil
	}
	if err != nil {
		return nil, err
	}
	var records []struct {
		ID  string `db:"id"`
		IRI string `db:"iri"`
	}
	err = app.RecordQuery(col).Select("id", "iri").Where(dbx.HashExp{"trail": trail}).All(&records)
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		if remoteSyncIRIMatchesOrigin(record.IRI, origin) {
			stored[record.ID] = struct{}{}
		}
	}
	return stored, nil
}

// A complete child expand describes copies owned by the fetched origin.
// Local entries and contributions hosted elsewhere remain independent.
func pruneRemoteTrailChildren(app core.App, trail, origin, collection string, listed, stored map[string]struct{}, cleanup *util.RemoteSummitLogCleanup) error {
	if listed == nil {
		return nil
	}
	ids := make([]any, 0, len(stored))
	for id := range stored {
		ids = append(ids, id)
	}
	// The frozen snapshot may be much larger than a fresh expand. Bound PK
	// reads, not transactions: every chunk still deletes atomically in the
	// caller's single per-collection cleanup transaction.
	for start := 0; start < len(ids); start += remoteSyncWriteBatch {
		end := min(start+remoteSyncWriteBatch, len(ids))
		records, err := app.FindAllRecords(collection, dbx.In("id", ids[start:end]...))
		if err != nil {
			return err
		}
		for _, record := range records {
			iri := record.GetString("iri")
			if record.GetString("trail") != trail || util.RecordHasLocalIdentity(app, record) || !remoteSyncIRIMatchesOrigin(iri, origin) {
				continue
			}
			if _, exists := listed[iri]; exists {
				continue
			}
			author, err := app.FindRecordById("activitypub_actors", record.GetString("author"))
			if err != nil || author.GetBool("is_local") {
				continue
			}
			if err := cleanup.Delete(app, record); err != nil {
				return err
			}
		}
	}
	return nil
}

func applyOmittedRemoteSummitLogs(app core.App, trail *core.Record, canonical map[string]canonicalRemoteSummitLog, cleanup *util.RemoteSummitLogCleanup) error {
	for iri, confirmed := range canonical {
		if !confirmed.omitted || confirmed.cacheKey != canonicalLogCacheKey(app, trail, iri) {
			continue
		}
		if confirmed.absenceProbe.entry != nil && confirmed.absenceProbe.key != confirmed.cacheKey {
			continue
		}
		record, _ := app.FindFirstRecordByData("summit_logs", "iri", iri)
		if record == nil || record.GetString("trail") != trail.Id || record.GetString("author") != confirmed.author || util.RecordHasLocalIdentity(app, record) || remoteRecordHasLocalAuthor(app, record) {
			canonicalRemoteLogAbsenceCache.reset(confirmed.cacheKey)
			continue
		}
		if confirmed.removed {
			currentTrail, err := app.FindRecordById("trails", trail.Id)
			if err != nil {
				canonicalRemoteLogAbsenceCache.reset(confirmed.cacheKey)
				if errors.Is(err, sql.ErrNoRows) {
					continue
				}
				return err
			}
			snapshot := canonicalLogAbsenceSnapshotOf(record, currentTrail)
			if snapshot != confirmed.storedSnapshot || util.RecordHasLocalIdentity(app, currentTrail) || remoteRecordHasLocalAuthor(app, currentTrail) {
				canonicalRemoteLogAbsenceCache.reset(confirmed.cacheKey)
				continue
			}
			author, err := app.FindRecordById("activitypub_actors", record.GetString("author"))
			if err != nil || author.GetBool("is_local") || (author.Id != currentTrail.GetString("author") && !remoteSyncIRIMatchesOrigin(author.GetString("iri"), iri)) {
				canonicalRemoteLogAbsenceCache.reset(confirmed.cacheKey)
				continue
			}
			if confirmed.absenceProbe.entry != nil {
				if !currentTrail.GetBool("public") {
					canonicalRemoteLogAbsenceCache.reset(confirmed.cacheKey)
					continue
				}
				if !canonicalRemoteLogAbsenceCache.ready(confirmed.absenceProbe, snapshot) {
					continue
				}
			}
			canonicalRemoteLogCache.invalidateAfterRollback(app, confirmed.cacheKey)
			if err := cleanup.Delete(app, record); err != nil {
				canonicalRemoteLogCache.invalidate(confirmed.cacheKey)
				return err
			}
			canonicalRemoteLogAbsenceCache.forgetAfterCommit(app, confirmed.absenceProbe)
		} else {
			canonicalRemoteLogCache.successAfterCommit(app, confirmed.cacheKey, confirmed.updatedHint, record.GetString("updated"))
		}
	}
	return nil
}
