package hooks

import (
	"database/sql"
	"encoding/json"
	"errors"
	"sync"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/util"
)

func HashAssetPhotoOnWrite() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		if err := util.HashUnsavedPhotoFile(e.Record); err != nil {
			return err
		}
		return e.Next()
	}
}

func InvalidateAssetContentHashOnFileChange() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		if e.Record.Original().Id == "" || e.Record.GetString("file") == e.Record.Original().GetString("file") {
			return e.Next()
		}
		metadata, err := assetMetadataMap(e.Record)
		if err != nil {
			return err
		}
		if _, ok := metadata["content_hash"]; !ok {
			return e.Next()
		}
		delete(metadata, "content_hash")
		e.Record.Set("metadata", metadata)
		return e.Next()
	}
}

func assetMetadataMap(record *core.Record) (map[string]any, error) {
	result := map[string]any{}
	rawBytes, err := json.Marshal(record.Get("metadata"))
	if err != nil {
		return nil, err
	}
	if len(rawBytes) == 0 || string(rawBytes) == "null" {
		return result, nil
	}
	if err := json.Unmarshal(rawBytes, &result); err != nil {
		return nil, err
	}
	return result, nil
}

const deletedRecordAssetsKey = "__delete_assets"

// CollectDeletedRecordAssetsHandler gathers a cascade's linked photos inside
// its delete transaction. Children already covered by that root skip the lookup.
func CollectDeletedRecordAssetsHandler() func(e *core.RecordEvent) error {
	type cascade struct {
		trails map[string]bool
		actors map[string]bool
	}
	var cascades sync.Map
	return func(e *core.RecordEvent) error {
		state := &cascade{trails: map[string]bool{}, actors: map[string]bool{}}
		if tx := e.App.TxInfo(); tx != nil {
			stored, loaded := cascades.LoadOrStore(tx, state)
			state = stored.(*cascade)
			if !loaded {
				tx.OnComplete(func(error) error { cascades.Delete(tx); return nil })
			}
		}
		name := e.Record.Collection().Name
		if name == "trails" && state.trails[e.Record.Id] ||
			(name == "waypoints" || name == "summit_logs") && (state.trails[e.Record.GetString("trail")] || state.actors[e.Record.GetString("author")]) {
			return e.Next()
		}
		assetIDs, err := linkedAssetIDs(e.App, e.Record)
		if err != nil {
			e.App.Logger().Error(
				"could not collect the photos of a deleted record",
				"collection", name, "record", e.Record.Id, "error", err,
			)
		} else if name == "trails" {
			state.trails[e.Record.Id] = true
		} else if name == "activitypub_actors" {
			trails := []string{}
			if err := e.App.DB().Select("id").From("trails").Where(dbx.HashExp{"author": e.Record.Id}).Column(&trails); err != nil {
				return err
			}
			for _, id := range trails {
				state.trails[id] = true
			}
			state.actors[e.Record.Id] = true
		}
		e.Record.Set(deletedRecordAssetsKey, assetIDs)
		return e.Next()
	}
}

func linkedAssetIDs(app core.App, record *core.Record) ([]string, error) {
	query := ""
	switch record.Collection().Name {
	case "trails":
		query = `SELECT asset FROM trail_assets WHERE trail={:id}
			UNION SELECT asset FROM waypoint_assets WHERE waypoint IN (SELECT id FROM waypoints WHERE trail={:id})
			UNION SELECT asset FROM summit_log_assets WHERE summit_log IN (SELECT id FROM summit_logs WHERE trail={:id})`
	case "activitypub_actors":
		query = `SELECT asset FROM trail_assets WHERE trail IN (SELECT id FROM trails WHERE author={:id})
			UNION SELECT asset FROM waypoint_assets WHERE waypoint IN (SELECT id FROM waypoints WHERE author={:id} OR trail IN (SELECT id FROM trails WHERE author={:id}))
			UNION SELECT asset FROM summit_log_assets WHERE summit_log IN (SELECT id FROM summit_logs WHERE author={:id} OR trail IN (SELECT id FROM trails WHERE author={:id}))`
	case "waypoints":
		return util.AssetIDsForLinkTarget(app, "waypoint_assets", "waypoint", record.Id)
	case "summit_logs":
		return util.AssetIDsForLinkTarget(app, "summit_log_assets", "summit_log", record.Id)
	default:
		return nil, nil
	}
	ids := []string{}
	err := app.DB().NewQuery(query).Bind(dbx.Params{"id": record.Id}).Column(&ids)
	return ids, err
}

// DeleteOrphanedAssetsHandler deletes the noted assets that nothing links to any
// more, whoever added them: like the photo files they replaced, they go with
// the record. It has to run before the other after-delete hooks, which stop at
// their first error, and it only logs its own, as the record is gone already.
func DeleteOrphanedAssetsHandler() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		deleteOrphanedRecordAssets(e.App, e.Record)
		return e.Next()
	}
}

// DeleteOrphanedAssetsAfterErrorHandler covers a delete that went through and
// still failed: the hooks its cascade runs after the commit report their errors
// through it, and its own after-delete hooks are skipped then. Only outside a
// transaction does a missing record prove that the delete went through.
func DeleteOrphanedAssetsAfterErrorHandler() func(e *core.RecordErrorEvent) error {
	return func(e *core.RecordErrorEvent) error {
		if e.App.TxInfo() == nil {
			if _, err := e.App.FindRecordById(e.Record.Collection(), e.Record.Id); errors.Is(err, sql.ErrNoRows) {
				deleteOrphanedRecordAssets(e.App, e.Record)
			}
		}
		return e.Next()
	}
}

func deleteOrphanedRecordAssets(app core.App, record *core.Record) {
	assetIDs, _ := record.GetRaw(deletedRecordAssetsKey).([]string)
	if err := util.DeleteAssetsIfOrphaned(app, assetIDs); err != nil {
		app.Logger().Error(
			"could not delete the photos a deleted record left unlinked",
			"collection", record.Collection().Name, "record", record.Id, "error", err,
		)
	}
}
