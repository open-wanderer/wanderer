package assets_test

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"pocketbase/hooks"
	"pocketbase/plugins/importer"
	"pocketbase/pluginsystem"
	assetservice "pocketbase/services/assets"
	"pocketbase/util"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestTrailPreparationIncludesLinkedContributorsAndReportsProgress(t *testing.T) {
	app, owner, contributor := newAssetPublishTestApp(t, nil)
	trail := savePublishRecord(t, app, "trails", map[string]any{"author": owner.Id})
	waypoint := savePublishRecord(t, app, "waypoints", map[string]any{"trail": trail.Id})
	log := savePublishRecord(t, app, "summit_logs", map[string]any{"trail": trail.Id})
	assets := []*core.Record{
		newPublishPhoto(t, app, owner.Id, "owner"),
		newPublishPhoto(t, app, contributor.Id, "contributor-waypoint"),
		newPublishPhoto(t, app, contributor.Id, "contributor-log"),
	}
	linkPublishPhoto(t, app, "trail", trail.Id, assets[0].Id)
	linkPublishPhoto(t, app, "waypoint", waypoint.Id, assets[1].Id)
	linkPublishPhoto(t, app, "summit_log", log.Id, assets[2].Id)
	// A photo linked twice is downloaded once; other trails are unaffected.
	linkPublishPhoto(t, app, "trail", trail.Id, assets[1].Id)
	otherTrail := savePublishRecord(t, app, "trails", map[string]any{"author": owner.Id})
	unrelated := newPublishPhoto(t, app, owner.Id, "other-trail")
	linkPublishPhoto(t, app, "trail", otherTrail.Id, unrelated.Id)

	progress := [][3]int{}
	if err := assetservice.MaterializePrivateRemotePluginAssetsForTrailWithProgress(context.Background(), app, trail.Id, func(total, processed, failed int) {
		progress = append(progress, [3]int{total, processed, failed})
	}); err != nil {
		t.Fatal(err)
	}
	if want := [][3]int{{3, 0, 0}, {3, 1, 0}, {3, 2, 0}, {3, 3, 0}}; !reflect.DeepEqual(progress, want) {
		t.Fatalf("progress = %v, want %v", progress, want)
	}
	stored, err := app.FindRecordById("trails", trail.Id)
	if err != nil || stored.GetBool("public") {
		t.Fatalf("preparation published the trail: trail=%v error=%v", stored, err)
	}
	publicationObserved := false
	app.OnRecordAfterUpdateSuccess("trails").BindFunc(func(e *core.RecordEvent) error {
		publicationObserved = true
		for _, asset := range assets {
			assertPublishPhotoState(t, app, asset.Id, "copy", "available")
		}
		return e.Next()
	})
	if err := savePublishedTrail(app, trail.Id); err != nil {
		t.Fatal(err)
	}
	if !publicationObserved {
		t.Fatal("trail success hook was not called")
	}
	assertPublishPhotoState(t, app, unrelated.Id, "link_private", "available")
}

func TestPublishGuardRejectsRemotePhotosWithoutDownloading(t *testing.T) {
	for _, transactional := range []bool{false, true} {
		t.Run(fmt.Sprintf("transaction=%t", transactional), func(t *testing.T) {
			app, owner, _ := newAssetPublishTestApp(t, nil)
			trail := savePublishRecord(t, app, "trails", map[string]any{"author": owner.Id})
			asset := newPublishPhoto(t, app, owner.Id, "photo")
			linkPublishPhoto(t, app, "trail", trail.Id, asset.Id)
			t.Cleanup(assetservice.SetRemotePhotoMediaFetcherForTest(func(context.Context, pluginsystem.Photo, importer.Options, int64) (*util.SafeFetchResult, error) {
				t.Fatal("publication guard started a download")
				return nil, nil
			}))
			var err error
			if transactional {
				err = app.RunInTransaction(func(txApp core.App) error { return savePublishedTrail(txApp, trail.Id) })
			} else {
				err = savePublishedTrail(app, trail.Id)
			}
			var apiErr *router.ApiError
			if !errors.As(err, &apiErr) || apiErr.Status != http.StatusBadRequest || apiErr.Message != "asset_publish_required" {
				t.Fatalf("publish error = %v, want asset_publish_required HTTP 400", err)
			}
			if err := assetservice.EnsureTrailAssetsMaterialized(app, trail.Id); !errors.Is(err, assetservice.ErrTrailMaterializationRequired) {
				t.Fatalf("readiness error = %v", err)
			}
			assertUnchangedPublishFailure(t, app, trail.Id, asset.Id)
		})
	}
}

func TestPreparationFailurePreservesSuccessfulPhotosForRetry(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusForbidden} {
		t.Run(fmt.Sprintf("status=%d", status), func(t *testing.T) {
			var unavailable atomic.Bool
			unavailable.Store(true)
			fetches := map[string]int{}
			app, owner, contributor := newAssetPublishTestApp(t, func(r *http.Request) int {
				fetches[r.URL.Path]++
				if r.URL.Path == "/photos/broken" && unavailable.Load() {
					return status
				}
				return http.StatusOK
			})
			trail := savePublishRecord(t, app, "trails", map[string]any{"author": owner.Id})
			broken := newPublishPhoto(t, app, contributor.Id, "broken")
			good := newPublishPhoto(t, app, owner.Id, "good")
			linkPublishPhoto(t, app, "trail", trail.Id, broken.Id)
			linkPublishPhoto(t, app, "trail", trail.Id, good.Id)
			progress := [3]int{}
			prepare := func() error {
				return assetservice.MaterializePrivateRemotePluginAssetsForTrailWithProgress(context.Background(), app, trail.Id, func(total, processed, failed int) {
					progress = [3]int{total, processed, failed}
				})
			}
			err := prepare()
			var failures *assetservice.MaterializationError
			if !errors.As(err, &failures) || len(failures.RemoteFailures) != 1 || failures.RemoteFailures[broken.Id] == nil {
				t.Fatalf("preparation error = %v, want remote failure for %s", err, broken.Id)
			}
			if progress != [3]int{2, 2, 1} {
				t.Fatalf("failed preparation progress = %v", progress)
			}
			wantStatus := "missing"
			if status == http.StatusForbidden {
				wantStatus = "inaccessible"
			}
			failed := assertPublishPhotoState(t, app, broken.Id, "link_private", wantStatus)
			if failed.GetString("remote_error") != (util.HTTPStatusError{StatusCode: status}).Error() || failed.GetDateTime("remote_checked_at").IsZero() {
				t.Fatal("failed asset status was not persisted")
			}
			if got := !failed.GetDateTime("remote_missing_since").IsZero(); got != (wantStatus == "missing") {
				t.Fatalf("remote_missing_since populated = %t for %s", got, wantStatus)
			}
			assertPublishPhotoState(t, app, good.Id, "copy", "available")
			if err := savePublishedTrail(app, trail.Id); err == nil {
				t.Fatal("failed preparation allowed publication")
			}
			unavailable.Store(false)
			if err := prepare(); err != nil {
				t.Fatalf("retry failed: %v", err)
			}
			if progress != [3]int{1, 1, 0} || fetches["/photos/good"] != 1 || fetches["/photos/broken"] != 2 {
				t.Fatalf("retry redownloaded completed photos: progress=%v fetches=%v", progress, fetches)
			}
			if err := app.RunInTransaction(func(txApp core.App) error { return savePublishedTrail(txApp, trail.Id) }); err != nil {
				t.Fatalf("publication after retry failed: %v", err)
			}
			for _, asset := range []*core.Record{broken, good} {
				stored := assertPublishPhotoState(t, app, asset.Id, "copy", "available")
				if stored.GetString("remote_error") != "" || !stored.GetDateTime("remote_missing_since").IsZero() {
					t.Fatal("successful retry retained a stale remote error")
				}
			}
		})
	}
}

func TestPublishGuardChecksPhotosLinkedAfterPreparation(t *testing.T) {
	app, owner, _ := newAssetPublishTestApp(t, nil)
	trail := savePublishRecord(t, app, "trails", map[string]any{"author": owner.Id})
	first := newPublishPhoto(t, app, owner.Id, "first")
	linkPublishPhoto(t, app, "trail", trail.Id, first.Id)
	if err := assetservice.MaterializePrivateRemotePluginAssetsForTrail(context.Background(), app, trail.Id); err != nil {
		t.Fatal(err)
	}
	late := newPublishPhoto(t, app, owner.Id, "late")
	linkPublishPhoto(t, app, "trail", trail.Id, late.Id)
	err := app.RunInTransaction(func(txApp core.App) error { return savePublishedTrail(txApp, trail.Id) })
	var apiErr *router.ApiError
	if !errors.As(err, &apiErr) || apiErr.Message != "asset_publish_required" {
		t.Fatalf("late photo bypassed publication guard: %v", err)
	}
	assertUnchangedPublishFailure(t, app, trail.Id, late.Id)
	assertPublishPhotoState(t, app, first.Id, "copy", "available")
}

func TestPreparationContextsArePerPhotoAndSeparateFromPublication(t *testing.T) {
	app, owner, _ := newAssetPublishTestApp(t, nil)
	trail := savePublishRecord(t, app, "trails", map[string]any{"author": owner.Id})
	for _, name := range []string{"first", "second"} {
		photo := newPublishPhoto(t, app, owner.Id, name)
		linkPublishPhoto(t, app, "trail", trail.Id, photo.Id)
	}
	contexts := []context.Context{}
	t.Cleanup(assetservice.SetRemotePhotoMediaFetcherForTest(func(ctx context.Context, _ pluginsystem.Photo, _ importer.Options, _ int64) (*util.SafeFetchResult, error) {
		if len(contexts) > 0 && !errors.Is(contexts[len(contexts)-1].Err(), context.Canceled) {
			t.Fatal("previous photo context was not released")
		}
		contexts = append(contexts, ctx)
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > assetservice.TrailPhotoMaterializationTimeout || time.Until(deadline) < time.Minute {
			t.Fatalf("invalid per-photo deadline: %v", deadline)
		}
		return &util.SafeFetchResult{Body: []byte("photo")}, nil
	}))
	if err := assetservice.MaterializePrivateRemotePluginAssetsForTrail(context.Background(), app, trail.Id); err != nil {
		t.Fatal(err)
	}
	if len(contexts) != 2 || contexts[0] == contexts[1] {
		t.Fatalf("download contexts = %v, want one per photo", contexts)
	}
	app.OnRecordUpdate("trails").BindFunc(func(e *core.RecordEvent) error {
		if e.Context.Err() != nil {
			t.Fatal("publication inherited canceled download context")
		}
		if _, ok := e.Context.Deadline(); ok {
			t.Fatal("publication inherited a download deadline")
		}
		return e.Next()
	})
	if err := savePublishedTrail(app, trail.Id); err != nil {
		t.Fatal(err)
	}
}

func savePublishedTrail(app core.App, trailID string) error {
	record, err := app.FindRecordById("trails", trailID)
	if err != nil {
		return err
	}
	record.Set("public", true)
	return app.Save(record)
}

func TestInternalPublicAssetLinksMaterializeBeforePersisting(t *testing.T) {
	for _, field := range []string{"trail", "waypoint", "summit_log"} {
		t.Run(field, func(t *testing.T) {
			app, owner, contributor := newAssetPublishTestApp(t, func(r *http.Request) int {
				if r.URL.Path == "/photos/broken" {
					return http.StatusNotFound
				}
				return http.StatusOK
			})
			// Covers creation of a public trail and links made by internal app.Save calls.
			trail := savePublishRecord(t, app, "trails", map[string]any{"author": owner.Id, "public": true})
			targetID := trail.Id
			if field != "trail" {
				targetID = savePublishRecord(t, app, field+"s", map[string]any{"trail": trail.Id}).Id
			}
			good := newPublishPhoto(t, app, contributor.Id, "good")
			linkPublishPhoto(t, app, field, targetID, good.Id)
			assertPublishPhotoState(t, app, good.Id, "copy", "available")
			broken := newPublishPhoto(t, app, contributor.Id, "broken")
			collection, err := app.FindCollectionByNameOrId(field + "_assets")
			if err != nil {
				t.Fatal(err)
			}
			link := core.NewRecord(collection)
			link.Set(field, targetID)
			link.Set("asset", broken.Id)
			if err := app.Save(link); err == nil {
				t.Fatal("link to unavailable remote photo was saved on a public trail")
			}
			if _, err := app.FindRecordById(collection, link.Id); err == nil {
				t.Fatal("failed link still exists")
			}
			assertPublishPhotoState(t, app, broken.Id, "link_private", "missing")
		})
	}
}

func TestConcurrentPublicationCannotOvertakePrivatePhotoLink(t *testing.T) {
	app, owner, _ := newAssetPublishTestApp(t, nil)
	trail := savePublishRecord(t, app, "trails", map[string]any{"author": owner.Id})
	photo := newPublishPhoto(t, app, owner.Id, "photo")
	collection, err := app.FindCollectionByNameOrId("trail_assets")
	if err != nil {
		t.Fatal(err)
	}
	link := core.NewRecord(collection)
	link.Set("trail", trail.Id)
	link.Set("asset", photo.Id)
	linkChecked := make(chan struct{})
	continueInsert := make(chan struct{})
	app.OnRecordCreateExecute("trail_assets").BindFunc(func(e *core.RecordEvent) error {
		close(linkChecked)
		<-continueInsert
		return e.Next()
	})
	linkResult := make(chan error, 1)
	go func() { linkResult <- app.Save(link) }()
	select {
	case <-linkChecked:
	case <-time.After(2 * time.Second):
		t.Fatal("link did not reach its insertion hook")
	}
	publishResult := make(chan error, 1)
	go func() {
		publishResult <- app.RunInTransaction(func(txApp core.App) error { return savePublishedTrail(txApp, trail.Id) })
	}()
	// Publication must wait for the link transaction and then see its photo.
	var earlyPublication error
	publicationFinishedEarly := false
	select {
	case earlyPublication = <-publishResult:
		publicationFinishedEarly = true
	case <-time.After(50 * time.Millisecond):
	}
	close(continueInsert)
	if err := <-linkResult; err != nil {
		t.Fatalf("link failed: %v", err)
	}
	if publicationFinishedEarly {
		t.Fatalf("publication finished before the checked photo link was saved: %v", earlyPublication)
	}
	var apiErr *router.ApiError
	if err := <-publishResult; !errors.As(err, &apiErr) || apiErr.Message != "asset_publish_required" {
		t.Fatalf("publication did not see the concurrently linked photo: %v", err)
	}
	assertUnchangedPublishFailure(t, app, trail.Id, photo.Id)
}

func TestPublicLinkRemoteFailureStatusSurvivesRollback(t *testing.T) {
	app, owner, _ := newAssetPublishTestApp(t, func(*http.Request) int { return http.StatusNotFound })
	trail := savePublishRecord(t, app, "trails", map[string]any{"author": owner.Id, "public": true})
	photo := newPublishPhoto(t, app, owner.Id, "missing")
	collection, err := app.FindCollectionByNameOrId("trail_assets")
	if err != nil {
		t.Fatal(err)
	}
	link := core.NewRecord(collection)
	link.Set("trail", trail.Id)
	link.Set("asset", photo.Id)
	if err := app.RunInTransaction(func(txApp core.App) error { return txApp.Save(link) }); err == nil {
		t.Fatal("failed photo link committed")
	}
	if _, err := app.FindRecordById(collection, link.Id); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("failed link exists: %v", err)
	}
	stored := assertPublishPhotoState(t, app, photo.Id, "link_private", "missing")
	if stored.GetString("remote_error") == "" || stored.GetDateTime("remote_missing_since").IsZero() {
		t.Fatal("remote failure status was lost on rollback")
	}
}

func TestAlreadyPublicUpdatesSkipMediaQueriesAndExplicitRepairStillWorks(t *testing.T) {
	for _, remote := range []bool{false, true} {
		t.Run(fmt.Sprintf("remote=%t", remote), func(t *testing.T) {
			var unavailable atomic.Bool
			unavailable.Store(true)
			app, owner, _ := newAssetPublishTestApp(t, func(*http.Request) int {
				if unavailable.Load() {
					return http.StatusForbidden
				}
				return http.StatusOK
			})
			author := owner.Id
			if remote {
				author = "remote-actor"
			}
			trail := savePublishRecord(t, app, "trails", map[string]any{"author": author})
			asset := newPublishPhoto(t, app, owner.Id, "legacy")
			linkPublishPhoto(t, app, "trail", trail.Id, asset.Id)
			// Reproduce an old build's already-public but unmaterialized photo.
			trail.Set("public", true)
			if err := app.UnsafeWithoutHooks().Save(trail); err != nil {
				t.Fatal(err)
			}
			trail, err := app.FindRecordById("trails", trail.Id)
			if err != nil {
				t.Fatal(err)
			}
			mediaQueries := []string{}
			for _, builder := range []dbx.Builder{app.ConcurrentDB(), app.NonconcurrentDB()} {
				db := builder.(*dbx.DB)
				originalLogger := db.QueryLogFunc
				t.Cleanup(func() { db.QueryLogFunc = originalLogger })
				db.QueryLogFunc = func(_ context.Context, _ time.Duration, query string, _ *sql.Rows, _ error) {
					if strings.Contains(query, "assets") || strings.Contains(query, "waypoints") || strings.Contains(query, "summit_logs") {
						mediaQueries = append(mediaQueries, query)
					}
				}
			}
			trail.Set("description", "Updated while Immich is unavailable")
			if err := app.Save(trail); err != nil {
				t.Fatalf("ordinary public update failed: %v", err)
			}
			if len(mediaQueries) != 0 {
				t.Fatalf("ordinary public update queried media: %v", mediaQueries)
			}
			assertPublishPhotoState(t, app, asset.Id, "link_private", "available")
			unavailable.Store(false)
			if err := assetservice.MaterializeRemotePluginAssetsForUser(context.Background(), app, "owner", "photos", true, nil); err != nil {
				t.Fatal(err)
			}
			assertPublishPhotoState(t, app, asset.Id, "copy", "available")
		})
	}
}

func TestLocalMaterializationFailuresDoNotChangeRemoteStatus(t *testing.T) {
	for _, failure := range []string{"file", "database", "metadata"} {
		t.Run(failure, func(t *testing.T) {
			app, owner, _ := newAssetPublishTestApp(t, nil)
			trail := savePublishRecord(t, app, "trails", map[string]any{"author": owner.Id})
			asset := newPublishPhoto(t, app, owner.Id, "photo")
			linkPublishPhoto(t, app, "trail", trail.Id, asset.Id)
			switch failure {
			case "file":
				t.Cleanup(assetservice.SetRemotePhotoMediaFetcherForTest(func(context.Context, pluginsystem.Photo, importer.Options, int64) (*util.SafeFetchResult, error) {
					return &util.SafeFetchResult{}, nil // NewFileFromBytes rejects empty content.
				}))
			case "database":
				app.OnRecordUpdate("assets").BindFunc(func(e *core.RecordEvent) error {
					if e.Record.GetString("storage_mode") == "copy" {
						return errors.New("local asset storage failed")
					}
					return e.Next()
				})
			case "metadata":
				asset.Set("metadata", map[string]any{})
				if err := app.Save(asset); err != nil {
					t.Fatal(err)
				}
			}
			err := assetservice.MaterializePrivateRemotePluginAssetsForTrail(context.Background(), app, trail.Id)
			var failures *assetservice.MaterializationError
			if !errors.As(err, &failures) || len(failures.RemoteFailures) != 0 {
				t.Fatalf("local preparation failure = %v", err)
			}
			assertUnchangedPublishFailure(t, app, trail.Id, asset.Id)
			// Plugin settings must not mislabel local failures either.
			failed := 0
			if err := assetservice.MaterializeRemotePluginAssetsForUser(context.Background(), app, "owner", "photos", false, func(_, _, n int) { failed = n }); err != nil || failed != 1 {
				t.Fatalf("job result failed=%d error=%v", failed, err)
			}
			assertUnchangedPublishFailure(t, app, trail.Id, asset.Id)
		})
	}
}

func TestPreparationKeepsPerPhotoLimitAndContinuesAfterSizeFailures(t *testing.T) {
	app, owner, _ := newAssetPublishTestApp(t, nil)
	trail := savePublishRecord(t, app, "trails", map[string]any{"author": owner.Id})
	for i := range 12 {
		asset := newPublishPhoto(t, app, owner.Id, fmt.Sprintf("photo%d", i))
		linkPublishPhoto(t, app, "trail", trail.Id, asset.Id)
	}
	calls := 0
	t.Cleanup(assetservice.SetRemotePhotoMediaFetcherForTest(func(_ context.Context, _ pluginsystem.Photo, _ importer.Options, maxBytes int64) (*util.SafeFetchResult, error) {
		calls++
		if maxBytes != util.DefaultPluginMediaMaxBytes {
			t.Fatalf("photo %d allowance = %d, want full per-photo limit", calls, maxBytes)
		}
		if calls <= 11 {
			return nil, util.ErrPluginMediaTooLarge
		}
		return &util.SafeFetchResult{Body: []byte("photo")}, nil
	}))
	progress := [3]int{}
	err := assetservice.MaterializePrivateRemotePluginAssetsForTrailWithProgress(context.Background(), app, trail.Id, func(total, processed, failed int) {
		progress = [3]int{total, processed, failed}
	})
	var failures *assetservice.MaterializationError
	if !errors.Is(err, util.ErrPluginMediaTooLarge) || !errors.As(err, &failures) || len(failures.Failures) != 11 || len(failures.RemoteFailures) != 0 {
		t.Fatalf("unexpected size failures: %v", err)
	}
	if calls != 12 || progress != [3]int{12, 12, 11} {
		t.Fatalf("preparation stopped after failed photos: calls=%d progress=%v", calls, progress)
	}
	for id := range failures.Failures {
		assertUnchangedPublishFailure(t, app, trail.Id, id)
	}
	if err := savePublishedTrail(app, trail.Id); err == nil {
		t.Fatal("oversized photo allowed publication")
	}
}

func TestPhotoTimeoutDoesNotCancelRemainingPreparation(t *testing.T) {
	app, owner, _ := newAssetPublishTestApp(t, nil)
	trail := savePublishRecord(t, app, "trails", map[string]any{"author": owner.Id})
	for _, name := range []string{"slow", "good"} {
		photo := newPublishPhoto(t, app, owner.Id, name)
		linkPublishPhoto(t, app, "trail", trail.Id, photo.Id)
	}
	calls := 0
	t.Cleanup(assetservice.SetRemotePhotoMediaFetcherForTest(func(ctx context.Context, photo pluginsystem.Photo, _ importer.Options, _ int64) (*util.SafeFetchResult, error) {
		calls++
		if photo.ExternalID == "slow" {
			<-ctx.Done()
			return nil, ctx.Err()
		}
		if ctx.Err() != nil {
			t.Fatal("slow photo canceled the next download")
		}
		return &util.SafeFetchResult{Body: []byte("photo")}, nil
	}))
	progress := [3]int{}
	err := assetservice.MaterializeTrailWithPhotoTimeoutForTest(context.Background(), app, trail.Id, func(total, processed, failed int) {
		progress = [3]int{total, processed, failed}
	}, 100*time.Millisecond)
	var failures *assetservice.MaterializationError
	if !errors.Is(err, context.DeadlineExceeded) || !errors.As(err, &failures) || len(failures.RemoteFailures) != 0 {
		t.Fatalf("unexpected timeout error: %v", err)
	}
	if calls != 2 || progress != [3]int{2, 2, 1} {
		t.Fatalf("per-photo timeout ended preparation: calls=%d progress=%v", calls, progress)
	}
}

func TestPreparationCancellationBeforeOrAfterMediaResponseKeepsTrailPrivate(t *testing.T) {
	for _, mode := range []string{"already-canceled", "during-fetch", "after-fetch", "deadline"} {
		t.Run(mode, func(t *testing.T) {
			app, owner, _ := newAssetPublishTestApp(t, nil)
			trail := savePublishRecord(t, app, "trails", map[string]any{"author": owner.Id})
			asset := newPublishPhoto(t, app, owner.Id, "photo")
			linkPublishPhoto(t, app, "trail", trail.Id, asset.Id)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "deadline" {
				var deadlineCancel context.CancelFunc
				ctx, deadlineCancel = context.WithTimeout(ctx, 20*time.Millisecond)
				defer deadlineCancel()
			}
			calls := 0
			t.Cleanup(assetservice.SetRemotePhotoMediaFetcherForTest(func(ctx context.Context, _ pluginsystem.Photo, _ importer.Options, maxBytes int64) (*util.SafeFetchResult, error) {
				calls++
				deadline, ok := ctx.Deadline()
				if !ok || time.Until(deadline) > assetservice.TrailPhotoMaterializationTimeout || maxBytes != util.DefaultPluginMediaMaxBytes {
					t.Fatalf("missing publication timeout/per-photo cap: deadline=%v maxBytes=%d", deadline, maxBytes)
				}
				if mode == "deadline" {
					<-ctx.Done()
					return nil, ctx.Err()
				}
				cancel()
				if mode == "during-fetch" {
					return nil, ctx.Err()
				}
				return &util.SafeFetchResult{Body: []byte("photo")}, nil
			}))
			if mode == "already-canceled" {
				cancel()
			}
			err := assetservice.MaterializePrivateRemotePluginAssetsForTrail(ctx, app, trail.Id)
			if err == nil {
				t.Fatal("canceled publication succeeded")
			}
			if mode == "deadline" && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("deadline error = %v, want context deadline exceeded", err)
			}
			if mode != "deadline" && !errors.Is(err, context.Canceled) {
				t.Fatalf("cancellation error = %v, want context canceled", err)
			}
			if mode == "already-canceled" && calls != 0 {
				t.Fatal("already canceled publication started a media request")
			}
			assertUnchangedPublishFailure(t, app, trail.Id, asset.Id)
		})
	}
}

func assertUnchangedPublishFailure(t *testing.T, app core.App, trailID, assetID string) {
	t.Helper()
	trail, err := app.FindRecordById("trails", trailID)
	if err != nil || trail.GetBool("public") {
		t.Fatalf("failed publication persisted: trail=%v error=%v", trail, err)
	}
	asset := assertPublishPhotoState(t, app, assetID, "link_private", "available")
	if asset.GetString("remote_error") != "" || !asset.GetDateTime("remote_checked_at").IsZero() || !asset.GetDateTime("remote_missing_since").IsZero() {
		t.Fatalf("non-remote error changed remote status: %v", asset)
	}
}

func newAssetPublishTestApp(t *testing.T, mediaStatus func(*http.Request) int) (*pbtests.TestApp, *core.Record, *core.Record) {
	t.Helper()
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	var photo bytes.Buffer
	if err := png.Encode(&photo, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(assetservice.SetRemotePhotoMediaFetcherForTest(func(_ context.Context, asset pluginsystem.Photo, opts importer.Options, _ int64) (*util.SafeFetchResult, error) {
		if opts.UserID != "owner" && opts.UserID != "contributor" {
			return nil, fmt.Errorf("unexpected media credentials owner %q", opts.UserID)
		}
		if mediaStatus != nil {
			request := httptest.NewRequest(http.MethodGet, "https://photos.example"+asset.Source.MediaRef.Path, nil)
			if status := mediaStatus(request); status != http.StatusOK {
				return nil, util.HTTPStatusError{StatusCode: status}
			}
		}
		return &util.SafeFetchResult{ContentType: "image/png", Body: photo.Bytes()}, nil
	}))
	collections := map[string][]core.Field{
		"trails":             {&core.TextField{Name: "author"}, &core.BoolField{Name: "public"}, &core.TextField{Name: "description"}},
		"waypoints":          {&core.TextField{Name: "trail"}},
		"summit_logs":        {&core.TextField{Name: "trail"}},
		"activitypub_actors": {&core.TextField{Name: "user"}},
		"plugin_instances":   {&core.TextField{Name: "user"}, &core.TextField{Name: "plugin_id"}, &core.BoolField{Name: "enabled"}, &core.JSONField{Name: "config"}, &core.JSONField{Name: "auth"}},
		"installed_plugins":  {&core.TextField{Name: "plugin_id"}, &core.TextField{Name: "path"}, &core.JSONField{Name: "manifest"}, &core.JSONField{Name: "config"}},
		"assets":             {&core.TextField{Name: "author"}, &core.TextField{Name: "type"}, &core.TextField{Name: "storage_mode"}, &core.TextField{Name: "external_provider"}, &core.TextField{Name: "external_id"}, &core.JSONField{Name: "metadata"}, &core.FileField{Name: "file", MaxSelect: 1}, &core.TextField{Name: "remote_status"}, &core.TextField{Name: "remote_error"}, &core.DateField{Name: "remote_checked_at"}, &core.DateField{Name: "remote_missing_since"}, &core.AutodateField{Name: "created", OnCreate: true}},
	}
	for _, field := range []string{"trail", "waypoint", "summit_log"} {
		collections[field+"_assets"] = []core.Field{&core.TextField{Name: field}, &core.TextField{Name: "asset"}}
	}
	for name, fields := range collections {
		collection := core.NewBaseCollection(name)
		collection.Fields.Add(fields...)
		if err := app.Save(collection); err != nil {
			t.Fatal(err)
		}
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "plugin.wasm"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := pluginsystem.Manifest{
		ManifestVersion: pluginsystem.ManifestVersion, ID: "photos", Type: pluginsystem.PluginTypeAssets, Name: "Photos", Version: "1.0.0",
		Runtime:      pluginsystem.RuntimeManifest{Type: pluginsystem.RuntimeWASM, Entrypoint: "plugin.wasm"},
		Capabilities: []pluginsystem.CapabilityManifest{{Name: "asset_library", Version: "v1", Export: "asset_library_v1"}},
		Permissions: pluginsystem.PermissionManifest{
			Network:   pluginsystem.NetworkPermissions{Connectors: []pluginsystem.ConnectorTargetPermission{{Name: "api", Type: pluginsystem.ConnectorTypeConfigured, ConfigKey: "photos", AllowedPathPrefixes: []string{"/photos"}}}},
			Downloads: pluginsystem.DownloadPermissions{MaxBytes: 1 << 20, ContentTypes: []string{"image/png"}},
		},
	}
	savePublishRecord(t, app, "installed_plugins", map[string]any{"plugin_id": "photos", "path": dir, "manifest": manifest, "config": map[string]any{"host": map[string]any{"connectors": map[string]any{"photos": map[string]any{"baseURL": "https://photos.example"}}}}})
	owner := savePublishRecord(t, app, "activitypub_actors", map[string]any{"user": "owner"})
	contributor := savePublishRecord(t, app, "activitypub_actors", map[string]any{"user": "contributor"})
	for _, user := range []string{"owner", "contributor"} {
		savePublishRecord(t, app, "plugin_instances", map[string]any{"user": user, "plugin_id": "photos", "enabled": true})
	}
	app.OnRecordUpdate("trails").BindFunc(hooks.MaterializePrivateRemoteAssetLinksBeforePublish(app))
	for _, field := range []string{"trail", "waypoint", "summit_log"} {
		app.OnRecordCreate(field + "_assets").BindFunc(hooks.MaterializePrivateRemoteAssetOnPublicLink(app, field))
	}
	return app, owner, contributor
}

func savePublishRecord(t *testing.T, app core.App, collectionName string, values map[string]any) *core.Record {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(collection)
	for field, value := range values {
		record.Set(field, value)
	}
	if err := app.Save(record); err != nil {
		t.Fatalf("save %s: %v", collectionName, err)
	}
	return record
}

func newPublishPhoto(t *testing.T, app core.App, author, name string) *core.Record {
	t.Helper()
	return savePublishRecord(t, app, "assets", map[string]any{
		"author": author, "type": "photo", "storage_mode": "link_private", "remote_status": "available", "external_provider": "photos", "external_id": name,
		"metadata": map[string]any{"remote": pluginsystem.RemotePhotoAsset{PluginID: "photos", Filename: name + ".png", ContentType: "image/png", Source: pluginsystem.MediaSource{Type: "connector", MediaRef: &pluginsystem.MediaRef{Connector: "api", Path: "/photos/" + name}}}},
	})
}

func linkPublishPhoto(t *testing.T, app core.App, field, targetID, assetID string) {
	t.Helper()
	savePublishRecord(t, app, field+"_assets", map[string]any{field: targetID, "asset": assetID})
}

func assertPublishPhotoState(t *testing.T, app core.App, assetID, mode, status string) *core.Record {
	t.Helper()
	asset, err := app.FindRecordById("assets", assetID)
	if err != nil {
		t.Fatal(err)
	}
	if asset.GetString("storage_mode") != mode || asset.GetString("remote_status") != status {
		t.Fatalf("asset %s mode/status = %s/%s, want %s/%s", assetID, asset.GetString("storage_mode"), asset.GetString("remote_status"), mode, status)
	}
	if (asset.GetString("file") != "") != (mode == "copy") {
		t.Fatalf("asset %s file = %q for storage mode %s", assetID, asset.GetString("file"), mode)
	}
	return asset
}
