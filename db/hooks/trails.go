package hooks

import (
	"errors"
	"fmt"
	"log"
	"os"
	"pocketbase/federation"
	assetservice "pocketbase/services/assets"
	"pocketbase/util"
	"time"

	"github.com/go-ap/activitypub"
	pub "github.com/go-ap/activitypub"
	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func CreateTrailHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		record := e.Record

		userActor, err := e.App.FindRecordById("activitypub_actors", record.GetString(("author")))
		if err != nil {
			return err
		}
		if err := util.SavePolyline(e.App, record); err != nil {
			log.Printf("failed to save polyline for trail %s: %v", record.Id, err)
		}

		// add local iri
		origin := os.Getenv("ORIGIN")
		if origin == "" {
			return fmt.Errorf("ORIGIN not set")
		}
		if e.Record.GetString("iri") == "" {
			e.Record.Set("iri", fmt.Sprintf("%s/api/v1/trail/%s", origin, e.Record.Id))
			if err = e.App.UnsafeWithoutHooks().Save(e.Record); err != nil {
				return err
			}
		}

		if err := util.IndexTrails(e.App, []*core.Record{record}, client); err != nil {
			return err
		}

		err = e.Next()
		if err != nil {
			return err
		}

		if !userActor.GetBool("is_local") {
			// this happens if someone fetches a remote list
			// we create a stub list record for later reference
			// no need to create an activity for that
			return nil
		}

		ctx, err := util.GetSafeActorContext(nil, userActor)

		if err != nil {
			return err
		}

		err = federation.CreateTrailActivity(e.App, ctx, e.Record, activitypub.CreateType)
		if err != nil {
			return err
		}

		_, err = util.InsertIntoFeed(e.App, userActor.Id, userActor.Id, record.Id, util.TrailFeed)
		if err != nil {
			return err
		}

		return nil
	}
}

// SetTrailCompletedAtHandler keeps completed_at consistent for every trail
// write, including imports and other server-side writes that don't pass through
// the public API request hooks.
func SetTrailCompletedAtHandler() func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		setTrailCompletedAt(e.Record, time.Now())
		return e.Next()
	}
}

func setTrailCompletedAt(record *core.Record, now time.Time) {
	if !record.GetBool("completed") {
		record.Set("completed_at", "")
		return
	}

	if record.GetDateTime("completed_at").IsZero() {
		record.Set("completed_at", now.UTC())
	}
}

func UpdateTrailHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		record := e.Record
		userActor, err := e.App.FindRecordById("activitypub_actors", record.GetString(("author")))
		if err != nil {
			return err
		}

		if record.GetString("gpx") != record.Original().GetString("gpx") {
			if err := util.SavePolyline(e.App, record); err != nil {
				log.Printf("failed to save polyline for trail %s: %v", record.Id, err)
			}
		}

		err = util.UpdateTrail(e.App, record, userActor, client)
		if err != nil {
			return err
		}
		if !userActor.GetBool("is_local") {
			// this happens if someone fetches a remote trail
			// we create a stub trail record for later reference
			// no need to create an activity for that
			return e.Next()
		}

		err = e.Next()
		if err != nil {
			return err
		}

		ctx, err := util.GetSafeActorContext(nil, userActor)

		if err != nil {
			return err
		}

		err = federation.CreateTrailActivity(e.App, ctx, e.Record, pub.UpdateType)
		if err != nil {
			return err
		}

		return nil
	}
}

// MaterializePrivateRemoteAssetLinksBeforePublish requires preparation to finish
// before publication. Downloads run in a background job; this final guard only
// reads local records, including photos linked while the job was running.
func MaterializePrivateRemoteAssetLinksBeforePublish(app core.App) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		if !e.Record.Original().GetBool("public") && e.Record.GetBool("public") {
			originalApp := e.App
			return originalApp.RunInTransaction(func(txApp core.App) error {
				if err := assetservice.EnsureTrailAssetsMaterialized(txApp, e.Record.Id); err != nil {
					return trailPublicationError(err)
				}
				e.App = txApp
				defer func() { e.App = originalApp }()
				return e.Next()
			})
		}
		return e.Next()
	}
}

func trailPublicationError(err error) error {
	code := "asset_publish_failed"
	if errors.Is(err, assetservice.ErrTrailMaterializationRequired) {
		code = "asset_publish_required"
	}
	apiErr := apis.NewBadRequestError(code, err)
	// PocketBase sentence-cases messages; preserve the public error code.
	apiErr.Message = code
	return apiErr
}

func restoreRemoteAssetStatusesAfterRollback(app, txApp core.App, materializeErr error) {
	var failures *assetservice.MaterializationError
	if txApp.TxInfo() == nil || !errors.As(materializeErr, &failures) || len(failures.RemoteFailures) == 0 {
		return
	}
	txApp.TxInfo().OnComplete(func(txErr error) error {
		if txErr == nil {
			return nil
		}
		for assetID, cause := range failures.RemoteFailures {
			asset, err := app.FindRecordById("assets", assetID)
			if err != nil || asset.GetString("storage_mode") != "link_private" {
				continue
			}
			if err := assetservice.MarkAssetRemoteStatus(app, asset, assetservice.RemoteStatusForError(cause), cause); err != nil {
				app.Logger().Warn("failed to restore remote asset status after rollback", "asset", assetID, "error", err)
			}
		}
		return nil
	})
}

// MaterializePrivateRemoteAssetOnPublicLink materializes a private remote plugin
// photo when it is linked to an already-public trail. A link can be created
// without saving the trail, so it would otherwise stay link_private with a dead
// /api/v1/assets/{id}/file URL for public/federated consumers (the file endpoint
// serves 404 for link_private assets on public trails).
func MaterializePrivateRemoteAssetOnPublicLink(app core.App, targetField string) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		assetID := e.Record.GetString("asset")
		targetID := e.Record.GetString(targetField)

		trailID, err := util.TrailIDForLinkTarget(e.App, targetField, targetID)
		if err != nil {
			return err
		}
		if trailID == "" {
			return e.Next()
		}

		trail, err := e.App.FindRecordById("trails", trailID)
		if err != nil {
			return err
		}
		if trail.GetBool("public") {
			if err := assetservice.MaterializePrivateRemotePluginAssetForPublicLink(e.Context, e.App, targetField, targetID, assetID); err != nil {
				restoreRemoteAssetStatusesAfterRollback(app, e.App, err)
				return apis.NewBadRequestError("Could not link remote photo because it could not be downloaded. Please download or remove the photo first.", err)
			}
		}
		// Serialize the final check and insertion with publication. A trail can
		// become public after the check above; in that case the caller must retry
		// so its download happens before opening this short transaction.
		originalApp := e.App
		return originalApp.RunInTransaction(func(txApp core.App) error {
			currentTrailID, err := util.TrailIDForLinkTarget(txApp, targetField, targetID)
			if err != nil {
				return err
			}
			currentTrail, err := txApp.FindRecordById("trails", currentTrailID)
			if err != nil {
				return err
			}
			if currentTrail.GetBool("public") && assetID != "" {
				asset, err := txApp.FindRecordById("assets", assetID)
				if err != nil {
					return err
				}
				if asset.GetString("type") == "photo" && asset.GetString("storage_mode") == "link_private" {
					return apis.NewBadRequestError("The trail became public before this photo was linked. Please try again to download the photo first.", assetservice.ErrTrailMaterializationRequired)
				}
			}
			e.App = txApp
			defer func() { e.App = originalApp }()
			return e.Next()
		})
	}
}

func DeleteTrailAssetCleanupHandler() func(e *core.RecordRequestEvent) error {
	return func(e *core.RecordRequestEvent) error {
		assetIDs, err := util.AssetIDsForTrail(e.App, e.Record.Id)
		if err != nil {
			return err
		}

		if err := e.Next(); err != nil {
			return err
		}

		return util.DeleteAssetsIfOrphanedByAuthor(e.App, assetIDs, e.Record.GetString("author"))
	}
}

func DeleteTrailHandler(client meilisearch.ServiceManager) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		record := e.Record
		task, err := client.Index("trails").DeleteDocument(record.Id, nil)
		if err != nil {
			return err
		}

		interval := 500 * time.Millisecond
		_, err = client.WaitForTask(task.TaskUID, interval)
		if err != nil {
			log.Fatalf("Error waiting for task completion: %v", err)
		}

		err = federation.CreateTrailDeleteActivity(e.App, e.Record)
		if err != nil {
			return err
		}

		err = util.DeleteFromFeed(e.App, record.Id)
		if err != nil {
			return err
		}

		return e.Next()
	}
}
