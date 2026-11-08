package migrations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"maps"
	"os"
	"path"
	"path/filepath"
	"pocketbase/util"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/types"
)

// The old trail editor uploaded route.webp; duplication adds another suffix.
// Old summit-log previews were PNG and can be recognized when the entry has
// a GPX and the preview is its first photo. Later uploads remain ordinary photos.
var legacyRoutePreviewName = regexp.MustCompile(`^route_[a-z0-9]{10}(?:_[a-z0-9]{10})*\.webp$`)
var legacyRoutePlaceholderName = regexp.MustCompile(`^wanderer_route_placeholder_`)
var interruptedPhotoCopyName = regexp.MustCompile(`^(.+)\.[0-9a-f]+\.tmp$`)
var legacyPNGRoutePreviewName = regexp.MustCompile(`^route_[a-z0-9]{10}(?:_[a-z0-9]{10})*\.png$`)

var importedWandererRoutePreviewName = regexp.MustCompile(`^wanderer_route_preview_[a-z0-9]{10}(?:_[a-z0-9]{10})*\.(?:webp|png)$`)

func init() {
	m.Register(withPhotoMigrationLogLevel(func(app core.App) error {
		assetCollection, err := app.FindCollectionByNameOrId("assets")
		if err != nil {
			return err
		}
		if err := migrateExistingPhotosToAssets(app, assetCollection); err != nil {
			return err
		}
		if err := removeLegacyPhotoFields(app); err != nil {
			return err
		}
		if err := moveMigratedPhotoProvenance(app); err != nil {
			return err
		}
		if err := indexMigratedPhotoAliases(app); err != nil {
			return err
		}
		return updateAssetSharingAccessRules(app)
	}), withPhotoMigrationLogLevel(func(app core.App) error {
		// Restore the database together with its matching media backup.
		return errors.New("photo library migration cannot be reverted; restore the matching database and media backup")
	}))
}

const (
	assetReadRule = `(@request.auth.id != "" && (
				author.user = @request.auth.id ||
				trail_assets_via_asset.trail.author.user ?= @request.auth.id ||
				trail_assets_via_asset.trail.trail_share_via_trail.actor.user ?= @request.auth.id ||
				waypoint_assets_via_asset.waypoint.trail.author.user ?= @request.auth.id ||
				waypoint_assets_via_asset.waypoint.trail.trail_share_via_trail.actor.user ?= @request.auth.id ||
				summit_log_assets_via_asset.summit_log.author.user ?= @request.auth.id ||
				summit_log_assets_via_asset.summit_log.trail.author.user ?= @request.auth.id ||
				summit_log_assets_via_asset.summit_log.trail.trail_share_via_trail.actor.user ?= @request.auth.id
			)) ||
			trail_assets_via_asset.trail.public ?= true ||
			waypoint_assets_via_asset.waypoint.trail.public ?= true ||
			summit_log_assets_via_asset.summit_log.trail.public ?= true ||
			(@request.query.share != "" && (
				trail_assets_via_asset.trail.trail_link_share_via_trail.token ?= @request.query.share ||
				waypoint_assets_via_asset.waypoint.trail.trail_link_share_via_trail.token ?= @request.query.share ||
				summit_log_assets_via_asset.summit_log.trail.trail_link_share_via_trail.token ?= @request.query.share
			))`
	trailAssetReadRule = `(@request.auth.id != "" && (
				asset.author.user = @request.auth.id || trail.author.user = @request.auth.id ||
				trail.trail_share_via_trail.actor.user ?= @request.auth.id
			)) || trail.public = true ||
			(@request.query.share != "" && trail.trail_link_share_via_trail.token ?= @request.query.share)`
	waypointAssetReadRule = `(@request.auth.id != "" && (
				asset.author.user = @request.auth.id ||
				waypoint.trail.author.user = @request.auth.id ||
				waypoint.trail.trail_share_via_trail.actor.user ?= @request.auth.id
			)) || waypoint.trail.public = true ||
			(@request.query.share != "" && waypoint.trail.trail_link_share_via_trail.token ?= @request.query.share)`
	summitLogAssetReadRule = `(@request.auth.id != "" && (
				asset.author.user = @request.auth.id || summit_log.author.user = @request.auth.id ||
				summit_log.trail.author.user = @request.auth.id ||
				summit_log.trail.trail_share_via_trail.actor.user ?= @request.auth.id
			)) || summit_log.trail.public = true ||
			(@request.query.share != "" && summit_log.trail.trail_link_share_via_trail.token ?= @request.query.share)`
	trailAssetTargetEditRule = `trail.author.user = @request.auth.id || (trail.trail_share_via_trail.actor.user ?= @request.auth.id && trail.trail_share_via_trail.permission ?= "edit")`
	trailAssetCreateRule     = `@request.auth.id != "" && asset.author.user = @request.auth.id && (` + trailAssetTargetEditRule + `)`
	trailAssetWriteRule      = `@request.auth.id != "" && (asset.author.user = @request.auth.id || ` + trailAssetTargetEditRule + `)`
	trailAssetUpdateRule     = `@request.auth.id != "" && (` + trailAssetTargetEditRule + `) && @request.body.asset:changed = false && @request.body.trail:changed = false`

	waypointAssetTargetEditRule = `waypoint.trail.author.user = @request.auth.id || (waypoint.trail.trail_share_via_trail.actor.user ?= @request.auth.id && waypoint.trail.trail_share_via_trail.permission ?= "edit")`
	waypointAssetCreateRule     = `@request.auth.id != "" && asset.author.user = @request.auth.id && (` + waypointAssetTargetEditRule + `)`
	waypointAssetWriteRule      = `@request.auth.id != "" && (asset.author.user = @request.auth.id || ` + waypointAssetTargetEditRule + `)`
	waypointAssetUpdateRule     = `@request.auth.id != "" && (` + waypointAssetTargetEditRule + `) && @request.body.asset:changed = false && @request.body.waypoint:changed = false`

	summitLogAssetTargetEditRule = `summit_log.trail.author.user = @request.auth.id || summit_log.author.user = @request.auth.id`
	summitLogAssetCreateRule     = `@request.auth.id != "" && asset.author.user = @request.auth.id && (` + summitLogAssetTargetEditRule + `)`
	summitLogAssetWriteRule      = `@request.auth.id != "" && (asset.author.user = @request.auth.id || ` + summitLogAssetTargetEditRule + `)`
	summitLogAssetUpdateRule     = `@request.auth.id != "" && (` + summitLogAssetTargetEditRule + `) && @request.body.asset:changed = false && @request.body.summit_log:changed = false`
)

type photoMigrationConfig struct {
	Collection          string
	LinkCollection      string
	LinkField           string
	LatField            string
	LonField            string
	TrailCoordinateLink bool
}

func updateAssetSharingAccessRules(app core.App) error {
	assetCollection, err := app.FindCollectionByNameOrId("assets")
	if err != nil {
		return err
	}
	assetCollection.ListRule = types.Pointer(assetReadRule)
	assetCollection.ViewRule = types.Pointer(assetReadRule)
	assetCollection.CreateRule = types.Pointer(`@request.auth.id != "" && @request.body.author.user = @request.auth.id &&
		@request.body.external_provider:isset = false && @request.body.external_id:isset = false &&
		@request.body.remote_error:isset = false && @request.body.remote_checked_at:isset = false && @request.body.remote_missing_since:isset = false &&
		@request.body.storage_mode = "copy" && (@request.body.remote_status:isset = false || @request.body.remote_status = "" || @request.body.remote_status = "available")`)
	assetCollection.DeleteRule = types.Pointer(`@request.auth.id != "" && author.user = @request.auth.id`)
	assetCollection.UpdateRule = types.Pointer(`@request.auth.id != "" && author.user = @request.auth.id && @request.body.author:changed = false &&
		@request.body.external_provider:isset = false && @request.body.external_id:isset = false &&
		@request.body.remote_error:isset = false && @request.body.remote_checked_at:isset = false && @request.body.remote_missing_since:isset = false &&
		@request.body.storage_mode:changed = false && @request.body.remote_status:isset = false`)
	if err := app.Save(assetCollection); err != nil {
		return err
	}

	linkCollections := []struct {
		Name       string
		CreateRule string
		ReadRule   string
		WriteRule  string
		UpdateRule string
	}{
		{"trail_assets", trailAssetCreateRule, trailAssetReadRule, trailAssetWriteRule, trailAssetUpdateRule},
		{"waypoint_assets", waypointAssetCreateRule, waypointAssetReadRule, waypointAssetWriteRule, waypointAssetUpdateRule},
		{"summit_log_assets", summitLogAssetCreateRule, summitLogAssetReadRule, summitLogAssetWriteRule, summitLogAssetUpdateRule},
	}

	for _, update := range linkCollections {
		collection, err := app.FindCollectionByNameOrId(update.Name)
		if err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(fmt.Sprintf(`{
			"createRule": %q,
			"deleteRule": %q,
			"listRule": %q,
			"updateRule": %q,
			"viewRule": %q
			}`, update.CreateRule, update.WriteRule, update.ReadRule, update.UpdateRule, update.ReadRule)), collection); err != nil {
			return err
		}
		if err := app.Save(collection); err != nil {
			return err
		}
	}

	return nil
}

func migrateExistingPhotosToAssets(app core.App, assetCollection *core.Collection) error {
	configs := []photoMigrationConfig{
		{Collection: "trails", LinkCollection: "trail_assets", LinkField: "trail", LatField: "lat", LonField: "lon"},
		{Collection: "waypoints", LinkCollection: "waypoint_assets", LinkField: "waypoint", LatField: "lat", LonField: "lon"},
		{Collection: "summit_logs", LinkCollection: "summit_log_assets", LinkField: "summit_log", TrailCoordinateLink: true},
	}

	// The legacy files may only go once the migration has committed, and the
	// copies made for the assets must go again if it rolls back.
	txInfo := app.TxInfo()
	if txInfo == nil {
		return errors.New("the photo migration must run inside a transaction")
	}
	if err := ensurePhotoMigrationCleanupQueue(app); err != nil {
		return err
	}
	committedDB, err := photoMigrationCommittedDB(app)
	if err != nil {
		return err
	}
	storage, err := newPhotoMigrationStorage(app)
	if err != nil {
		return err
	}
	files := &migratedPhotoFiles{legacyItems: make(map[string]photoMigrationCleanupItem)}
	txInfo.OnComplete(func(txErr error) error {
		defer storage.Close()
		files.cleanup(committedDB, storage, txErr)
		return nil
	})
	if err := cleanInterruptedPhotoMigration(app, storage.System, assetCollection.Id, configs); err != nil {
		return err
	}
	var total int64
	for _, cfg := range configs {
		var count int64
		if err := app.DB().NewQuery("SELECT COALESCE(SUM(CASE WHEN json_valid(photos) THEN json_array_length(photos) ELSE 0 END), 0) FROM " + cfg.Collection).Row(&count); err != nil {
			return err
		}
		total += count
	}
	if total > 0 {
		fmt.Printf("Migrating %d legacy photos to the asset library...\n", total)
	}

	trailProviders, err := loadTrailProviders(app)
	if err != nil {
		return err
	}

	// Like the other backfills, write without record hooks: they would reindex
	// and federate every trail with photos and bump its updated timestamp. The
	// file field uploads and deletes files in those hooks, so the migration
	// handles the files itself.
	withoutHooks := app.UnsafeWithoutHooks()
	for _, cfg := range configs {
		lastID := ""
		for {
			records, err := app.FindRecordsByFilter(cfg.Collection, "id>{:last}", "+id", 200, 0, map[string]any{"last": lastID})
			if err != nil {
				return err
			}
			if len(records) == 0 {
				break
			}
			if err := runPhotoMigrationPage(records, func(record *core.Record) error {
				return migrateRecordPhotosToAssets(withoutHooks, storage, files, assetCollection, cfg, record, trailProviders)
			}); err != nil {
				return err
			}
			lastID = records[len(records)-1].Id
		}
	}
	if total > 0 {
		fmt.Printf("Photo migration complete: %d of %d photos migrated.\n", len(files.legacyItems), total)
	}

	return nil
}

// The durable marker distinguishes an uncommitted attempt, which only wrote
// disposable copies, from media that may belong to a newer database backup.
// Validate marked storage completely before removing any recovery source.
func cleanInterruptedPhotoMigration(app core.App, fsys *filesystem.System, collectionID string, configs []photoMigrationConfig) error {
	prefix := collectionID + "/"
	_, markerErr := fsys.Attributes(photoMigrationCommitMarker)
	if errors.Is(markerErr, filesystem.ErrNotFound) {
		removed, err := discardUncommittedPhotoMigration(app, fsys, collectionID)
		if err == nil && removed {
			fmt.Println("Discarded media copies from an uncommitted photo migration.")
		}
		return err
	}
	if markerErr != nil {
		return markerErr
	}
	keys, err := interruptedPhotoMigrationKeys(app, fsys, prefix)
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		if err := fsys.Delete(photoMigrationCommitMarker); err != nil && !errors.Is(err, filesystem.ErrNotFound) {
			return err
		}
		return nil
	}
	fmt.Println("Checking media from an interrupted photo migration...")
	originals := map[string]string{}
	partialCopies := map[string]string{}
	thumbDirectories := map[string]string{}
	needed := map[string]bool{}
	unsafeStorage := func(key string) error {
		return fmt.Errorf("existing asset media %s has no identical legacy source; this storage carries the marker of a committed photo migration, so the database is older than its media; no asset files were deleted. Start the database that completed the upgrade, or restore the database and media backup taken together. Do not delete the asset files or the marker: they can be the only copy of photos whose originals were already removed", key)
	}
	for _, key := range keys {
		parts := strings.SplitN(strings.TrimPrefix(key, prefix), "/", 3)
		if len(parts) == 2 {
			if temp := interruptedPhotoCopyName.FindStringSubmatch(parts[1]); temp != nil {
				name := strings.TrimSuffix(temp[1], ".attrs")
				partialCopies[key] = name
				needed[name] = true
				continue
			}
			if previous := originals[parts[0]]; previous != "" {
				return unsafeStorage(previous + " (additional file: " + key + ")")
			}
			originals[parts[0]] = key
			needed[parts[1]] = true
		} else if len(parts) == 3 && strings.HasPrefix(parts[1], "thumbs_") {
			thumbDirectories[parts[0]] = strings.TrimPrefix(parts[1], "thumbs_")
		} else {
			return unsafeStorage(key)
		}
	}
	for directory, name := range thumbDirectories {
		if original := originals[directory]; original != "" && path.Base(original) != name {
			return unsafeStorage(prefix + directory + "/thumbs_" + name)
		}
		needed[name] = true
	}
	sources := map[string][]string{}
	for _, cfg := range configs {
		lastID := ""
		for {
			records, err := app.FindRecordsByFilter(cfg.Collection, "id>{:last}", "+id", 200, 0, map[string]any{"last": lastID})
			if err != nil {
				return err
			}
			if len(records) == 0 {
				break
			}
			for _, record := range records {
				for _, name := range record.GetStringSlice("photos") {
					if needed[name] {
						sources[name] = append(sources[name], record.BaseFilesPath()+"/"+name)
					}
				}
			}
			lastID = records[len(records)-1].Id
		}
	}
	// An interrupted fileblob write has no complete hash yet. Its exact
	// target must still have a referenced legacy source before it can go.
	for partial, name := range partialCopies {
		recoverable := false
		for _, source := range sources[name] {
			if _, err := fsys.Attributes(source); err == nil {
				recoverable = true
				break
			} else if !errors.Is(err, filesystem.ErrNotFound) {
				return err
			}
		}
		if !recoverable {
			return unsafeStorage(partial)
		}
	}
	// Cleanup can be interrupted after removing a copied original. Its
	// thumbnails remain disposable only while their legacy source still exists.
	for directory, name := range thumbDirectories {
		if originals[directory] != "" {
			continue
		}
		recoverable := false
		for _, source := range sources[name] {
			if _, err := fsys.Attributes(source); err == nil {
				recoverable = true
				break
			} else if !errors.Is(err, filesystem.ErrNotFound) {
				return err
			}
		}
		if !recoverable {
			return unsafeStorage(prefix + directory + "/thumbs_" + name)
		}
	}
	sourceHashes := map[string]string{}
	var hashMu, progressMu sync.Mutex
	copies := make([]string, 0, len(originals))
	for _, copied := range originals {
		copies = append(copies, copied)
	}
	fmt.Printf("Checking %d copied photos before restarting the migration...\n", len(originals))
	checked := 0
	if err := runPhotoMigrationPage(copies, func(copied string) error {
		matched, err := interruptedPhotoCopyMatches(app, fsys, copied, sources[path.Base(copied)], sourceHashes, &hashMu)
		if err != nil {
			return err
		}
		if !matched {
			return unsafeStorage(copied)
		}
		progressMu.Lock()
		checked++
		photoMigrationProgress("Checked copied photos", checked, len(copies))
		progressMu.Unlock()
		return nil
	}); err != nil {
		return err
	}

	if !app.Settings().S3.Enabled {
		// Remove only validated contents and keep the collection directory.
		entries, err := os.ReadDir(localPhotoMigrationPath(app, collectionID))
		if err != nil {
			return err
		}
		fmt.Printf("Removing %d validated interrupted photo directories...\n", len(entries))
		for i, entry := range entries {
			if err := os.RemoveAll(localPhotoMigrationPath(app, collectionID+"/"+entry.Name())); err != nil {
				return fmt.Errorf("clean up interrupted photo migration: %w", err)
			}
			photoMigrationProgress("Removed interrupted photo directories", i+1, len(entries))
		}
		return nil
	}
	fmt.Printf("Removing %d validated interrupted photo files...\n", len(keys))
	removed := 0
	if err := runPhotoMigrationPage(keys, func(key string) error {
		if err := fsys.Delete(key); err != nil && !errors.Is(err, filesystem.ErrNotFound) {
			return fmt.Errorf("clean up interrupted photo migration: %w", err)
		}
		progressMu.Lock()
		removed++
		photoMigrationProgress("Removed interrupted photo files", removed, len(keys))
		progressMu.Unlock()
		return nil
	}); err != nil {
		return err
	}
	return fsys.Delete(photoMigrationCommitMarker)
}

// Local fileblob pagination rescans the complete tree for every page and reads
// sidecar attributes. Scan once without payload or sidecar reads instead.
func interruptedPhotoMigrationKeys(app core.App, fsys *filesystem.System, prefix string) ([]string, error) {
	// ReadDir follows the administrator-configured collection-root symlink.
	// Nested symlinks remain rejected below; cleanup preserves the root link.
	root := localPhotoMigrationPath(app, strings.TrimSuffix(prefix, "/"))
	_, err := os.Stat(root)
	if !app.Settings().S3.Enabled && errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if app.Settings().S3.Enabled {
		objects, err := fsys.List(prefix)
		if err != nil {
			return nil, err
		}
		keys := make([]string, 0, len(objects))
		for _, object := range objects {
			if !object.IsDir && !photoMigrationMetadataPath(strings.TrimPrefix(object.Key, prefix)) {
				keys = append(keys, object.Key)
			}
		}
		return keys, nil
	}
	if err != nil {
		return nil, err
	}
	keys := []string{}
	var scan func(string, string) error
	scan = func(directory, keyPrefix string) error {
		entries, err := os.ReadDir(directory)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			key := keyPrefix + entry.Name()
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("unexpected symbolic link in interrupted photo migration: %s", key)
			}
			if !entry.IsDir() && !entry.Type().IsRegular() {
				return fmt.Errorf("unexpected nonregular file in interrupted photo migration: %s", key)
			}
			if photoMigrationMetadataPath(entry.Name()) {
				continue
			}
			if entry.IsDir() {
				if err := scan(filepath.Join(directory, entry.Name()), key+"/"); err != nil {
					return err
				}
				continue
			}
			if !entry.Type().IsRegular() {
				return fmt.Errorf("unexpected nonregular file in interrupted photo migration: %s", key)
			}
			if !strings.HasSuffix(entry.Name(), ".attrs") {
				keys = append(keys, key)
			}
		}
		return nil
	}
	if err := scan(root, prefix); err != nil {
		return nil, err
	}
	return keys, nil
}

func localPhotoMigrationPath(app core.App, key string) string {
	return filepath.Join(app.DataDir(), "storage", filepath.FromSlash(key))
}

func migrationPhotoHash(fsys *filesystem.System, key string) (string, error) {
	reader, err := fsys.GetReader(key)
	if err != nil {
		return "", err
	}
	hash, _, hashErr := photoMigrationContentHash(reader)
	return hash, errors.Join(hashErr, reader.Close())
}

// migratedPhotoFiles collects the storage paths the photo migration copies
// from and to.
type migratedPhotoFiles struct {
	sync.Mutex
	assetDirs   []string
	legacyItems map[string]photoMigrationCleanupItem
	copyStats   photoMigrationCopyStats
}

// The migration runner owns an auxiliary transaction. PocketBase's logger
// flushes batches through another auxiliary transaction, which deadlocks once
// 200 warnings accumulate. Write migration warnings directly to stderr.
func photoMigrationWarning(message string, attributes ...any) {
	var line strings.Builder
	fmt.Fprint(&line, "Photo migration warning: ", message)
	for i := 0; i+1 < len(attributes); i += 2 {
		fmt.Fprintf(&line, " %v=%q", attributes[i], fmt.Sprint(attributes[i+1]))
	}
	fmt.Fprintln(os.Stderr, line.String())
}

// cleanup deletes the migrated legacy files after a commit and the asset copies
// after a rollback. The transaction outcome is final; failed legacy cleanup
// retains the committed queue item for retry or review.
func (f *migratedPhotoFiles) cleanup(database dbx.Builder, storage photoMigrationStorage, txErr error) {
	if txErr != nil {
		fmt.Printf("Removing %d photo directories after migration rollback...\n", len(f.assetDirs))
		var progressMu sync.Mutex
		cleaned := 0
		_ = runPhotoMigrationPage(f.assetDirs, func(dir string) error {
			for _, err := range storage.DeletePrefix(dir + "/") {
				photoMigrationWarning("failed to delete asset file after asset migration rollback", "dir", dir, "error", err)
			}
			progressMu.Lock()
			cleaned++
			photoMigrationProgress("Processed rollback photo directories", cleaned, len(f.assetDirs))
			progressMu.Unlock()
			return nil
		})
		return
	}

	defer f.copyStats.report()
	ready, err := preparePhotoMigrationCleanup(database, storage, f.copyStats.files > 0)
	if err != nil {
		photoMigrationWarning("legacy photo cleanup retained before deletion", "error", err)
		return
	}
	total := len(f.legacyItems)
	if !ready || total == 0 {
		return
	}
	fmt.Printf("Cleaning %d migrated legacy photos...\n", total)
	var progressMu sync.Mutex
	cleaned := 0
	ctx := context.Background()
	// The map keeps one item per legacy key; dispatch directly without a
	// second library-sized collection of cleanup metadata.
	_ = runPhotoMigrationItems(maps.Values(f.legacyItems), total, func(item photoMigrationCleanupItem) error {
		defer func() {
			progressMu.Lock()
			cleaned++
			photoMigrationProgress("Cleaning migrated legacy photos", cleaned, total)
			progressMu.Unlock()
		}()
		if code, err := cleanPhotoMigrationCleanupItem(database, storage, item); err != nil {
			// As before, transient failures of synchronous cleanup are eligible
			// on the first resume pass, which claims them with retry backoff.
			_ = retainPhotoMigrationCleanup(ctx, database, item, 0, code, err)
		}
		return nil
	})
}

func loadTrailProviders(app core.App) (map[string]string, error) {
	refs, err := app.FindRecordsByFilter("trail_external_reference", "provider='komoot' || provider='strava'", "", -1, 0, nil)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(refs))
	for _, ref := range refs {
		if trailID := ref.GetString("trail"); trailID != "" {
			result[trailID] = ref.GetString("provider")
		}
	}
	return result, nil
}

// migrateRecordPhotosToAssets expects an app without record hooks.
func migrateRecordPhotosToAssets(app core.App, storage photoMigrationStorage, files *migratedPhotoFiles, assetCollection *core.Collection, cfg photoMigrationConfig, source *core.Record, trailProviders map[string]string) error {
	photos := source.GetStringSlice("photos")
	if len(photos) == 0 {
		return nil
	}

	linkCollection, err := app.FindCollectionByNameOrId(cfg.LinkCollection)
	if err != nil {
		return err
	}

	rawAuthor := source.GetString("author")
	if cfg.Collection == "waypoints" {
		if trail, err := app.FindRecordById("trails", source.GetString("trail")); err == nil {
			if owner, err := resolveAssetAuthor(app, trail.GetString("author")); err == nil && owner != "" {
				rawAuthor = owner
			}
		}
	}
	author, err := resolveAssetAuthor(app, rawAuthor)
	if err != nil {
		photoMigrationWarning("skipping legacy photos with unresolved author during asset migration", "collection", cfg.Collection, "record", source.Id, "author", source.GetString("author"), "error", err)
		return nil
	}
	if author == "" {
		photoMigrationWarning("skipping legacy photos with missing author during asset migration", "collection", cfg.Collection, "record", source.Id)
		return nil
	}

	// Without hooks nothing fills in ids and autodates. Links are listed by
	// created, so the timestamps are stepped to keep the legacy photo order.
	migratedAt := types.NowDateTime()
	thumbnailIndex := source.GetInt("thumbnail")
	for i, photo := range photos {
		if photo == "" || strings.ContainsAny(photo, `/\`) {
			photoMigrationWarning("skipping invalid legacy photo during asset migration", "collection", cfg.Collection, "record", source.Id, "photo", photo)
			continue
		}
		createdAt := migratedAt.Add(time.Duration(i) * time.Millisecond)

		asset := core.NewRecord(assetCollection)
		asset.Id = core.GenerateDefaultRandomId()
		legacyFile := source.BaseFilesPath() + "/" + photo
		assetFile := asset.BaseFilesPath() + "/" + photo
		// Registered before copying so that a rollback also removes a partial copy.
		files.Lock()
		files.assetDirs = append(files.assetDirs, asset.BaseFilesPath())
		files.Unlock()
		if err := copyLegacyPhotoFile(storage, legacyFile, assetFile, &files.copyStats); err != nil {
			// Only a missing legacy file may be skipped. Skipping any other
			// failure, such as a full disk or a failed S3 request, would detach
			// the photo for good once the photos column is dropped.
			if errors.Is(err, filesystem.ErrNotFound) {
				_, sourceErr := storage.Attributes(legacyFile)
				if errors.Is(sourceErr, filesystem.ErrNotFound) {
					sourceErr = confirmMissingLegacyPhoto(storage.System, legacyFile)
					if sourceErr == nil {
						photoMigrationWarning("skipping missing legacy photo during asset migration", "collection", cfg.Collection, "record", source.Id, "photo", photo, "error", err)
						continue
					}
				}
				if sourceErr != nil {
					err = errors.Join(err, fmt.Errorf("check legacy original: %w", sourceErr))
				}
			}
			return fmt.Errorf("failed to copy legacy photo %s: %w", legacyFile, err)
		}
		hash, err := checkedLegacyPhotoHash(storage, legacyFile, assetFile)
		if err != nil {
			return fmt.Errorf("failed to verify legacy photo %s and its copy %s: %w", legacyFile, assetFile, err)
		}
		thumbPrefix := source.BaseFilesPath() + "/thumbs_" + photo + "/"
		thumbs, err := storage.List(thumbPrefix)
		if err != nil {
			return err
		}
		var thumbKeys []string
		for _, thumb := range thumbs {
			if thumb.IsDir {
				continue
			}
			// S3 lists zero-byte directory markers as objects. Leave only
			// canonical markers within this thumbnail prefix untouched.
			if storage.localRoot == "" && thumb.Size == 0 && strings.HasPrefix(thumb.Key, thumbPrefix) &&
				strings.HasSuffix(thumb.Key, "/") && canonicalPhotoMigrationKey(strings.TrimSuffix(thumb.Key, "/")) {
				continue
			}
			destination := asset.BaseFilesPath() + "/thumbs_" + photo + "/" + strings.TrimPrefix(thumb.Key, thumbPrefix)
			if err := copyLegacyPhotoFile(storage, thumb.Key, destination, &files.copyStats); err != nil {
				return fmt.Errorf("migrate legacy photo thumbnail: %w", err)
			}
			thumbKeys = append(thumbKeys, thumb.Key)
		}

		asset.Set("type", "photo")
		asset.Set("storage_mode", "copy")
		asset.Set("remote_status", "available")
		asset.Set("file", photo)
		asset.Set("author", author)
		asset.SetRaw("created", createdAt)
		asset.SetRaw("updated", createdAt)
		metadata := map[string]any{
			"source_collection": cfg.Collection,
			"source_record":     source.Id,
			"source_file":       photo,
		}

		metadata["content_hash"] = hash
		if ((cfg.Collection == "trails" || cfg.Collection == "summit_logs") && (legacyRoutePreviewName.MatchString(photo) || importedWandererRoutePreviewName.MatchString(photo) || legacyRoutePlaceholderName.MatchString(photo))) ||
			(cfg.Collection == "summit_logs" && i == 0 && source.GetString("gpx") != "" && legacyPNGRoutePreviewName.MatchString(photo)) {
			metadata["generated"] = map[string]any{"kind": "route-preview"}
		}
		asset.Set("metadata", metadata)
		if isUnmodifiedSinceCreation(source) && cfg.Collection != "summit_logs" {
			trailID := source.Id
			if cfg.Collection == "waypoints" {
				trailID = source.GetString("trail")
			}
			if provider := trailProviders[trailID]; provider != "" {
				metadata["source_external_provider"] = provider
				asset.Set("metadata", metadata)
			}
		}
		if cfg.LatField != "" {
			asset.Set("lat", source.GetFloat(cfg.LatField))
		}
		if cfg.LonField != "" {
			asset.Set("lon", source.GetFloat(cfg.LonField))
		}
		if cfg.TrailCoordinateLink {
			if trail, err := app.FindRecordById("trails", source.GetString("trail")); err == nil && trail.GetString("author") == author {
				asset.Set("lat", trail.GetFloat("lat"))
				asset.Set("lon", trail.GetFloat("lon"))
			}
		}

		if err := insertMigratedPhotoRecord(app, asset); err != nil {
			return err
		}

		link := core.NewRecord(linkCollection)
		link.Id = core.GenerateDefaultRandomId()
		link.Set("asset", asset.Id)
		link.Set(cfg.LinkField, source.Id)
		link.SetRaw("created", createdAt)
		link.SetRaw("updated", createdAt)
		if cfg.Collection == "trails" && i == thumbnailIndex {
			link.Set("is_thumbnail", true)
		}
		if err := insertMigratedPhotoRecord(app, link); err != nil {
			return err
		}
		item, err := enqueuePhotoMigrationCleanup(app, legacyFile, assetFile, hash, thumbKeys)
		if err != nil {
			return err
		}
		files.Lock()
		// Like the durable queue's UPSERT, keep only the last item for
		// a repeated legacy key.
		files.legacyItems[legacyFile] = item
		if len(files.legacyItems)%100 == 0 {
			fmt.Printf("Photo migration: %d photos migrated.\n", len(files.legacyItems))
		}
		files.Unlock()
	}

	// The photos column itself is dropped by removeLegacyPhotoFields.
	return nil
}

// Record saves retain each model in a transaction-completion closure even
// without hooks. Export PocketBase's field values and insert directly so only
// the current page of source records remains live throughout the migration.
func insertMigratedPhotoRecord(app core.App, record *core.Record) error {
	values, err := record.DBExport(app)
	if err != nil {
		return err
	}
	_, err = app.DB().Insert(record.Collection().Name, dbx.Params(values)).Execute()
	return err
}

// Hard links avoid duplicating a local library. Filesystems or ownership rules
// that reject hard links retain the ordinary fileblob copy path and metadata.
// S3 retains its server-side copy operation.
func copyLegacyPhotoFile(storage photoMigrationStorage, source, destination string, stats *photoMigrationCopyStats) error {
	if storage.localRoot == "" {
		return storage.Copy(source, destination)
	}
	sourcePath := storage.localPath(source)
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return filesystem.ErrNotFound
		}
		return err
	}
	destinationPath := storage.localPath(destination)
	if err := os.MkdirAll(filepath.Dir(destinationPath), 0o777); err != nil {
		return err
	}
	if err := os.Link(sourcePath, destinationPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return filesystem.ErrNotFound
		}
		if hardLinkCopyFallback(err) {
			stats.Lock()
			stats.announce(err, false)
			stats.Unlock()
			if err := storage.Copy(source, destination); err != nil {
				return err
			}
			stats.Lock()
			stats.files++
			stats.bytes += sourceInfo.Size()
			stats.Unlock()
			return nil
		}
		return fmt.Errorf("link legacy photo: %w", err)
	}
	// fileblob stores the content type and metadata beside the bytes. Without
	// that sidecar PocketBase serves originals and thumbnails as downloads.
	if err := os.Link(sourcePath+".attrs", destinationPath+".attrs"); err != nil && !errors.Is(err, os.ErrNotExist) {
		if !hardLinkCopyFallback(err) {
			return fmt.Errorf("link legacy photo attributes: %w", err)
		}
		stats.Lock()
		stats.announce(err, true)
		stats.Unlock()
		attributes, err := os.ReadFile(sourcePath + ".attrs")
		if err != nil {
			return err
		}
		if err := os.WriteFile(destinationPath+".attrs", attributes, 0o666); err != nil {
			return err
		}
		stats.Lock()
		stats.files++
		stats.bytes += int64(len(attributes))
		stats.Unlock()
	}
	return nil
}

type photoMigrationCopyStats struct {
	sync.Mutex
	announced        bool
	payloadAnnounced bool
	files            int
	bytes            int64
}

func (stats *photoMigrationCopyStats) announce(err error, attributesOnly bool) {
	if !stats.announced {
		if attributesOnly {
			photoMigrationWarning("attribute sidecars require local copies; photo payloads retain hard links", "error", err)
		}
		stats.announced = true
	}
	if !attributesOnly && !stats.payloadAnnounced {
		stats.payloadAnnounced = true
		photoMigrationWarning("hard links unavailable; switching to local copy mode (additional disk space required)", "error", err)
	}
}

func (stats *photoMigrationCopyStats) report() {
	if stats.announced {
		if !stats.payloadAnnounced {
			fmt.Printf("Photo migration: %d attribute sidecars copied (%d bytes); photo payloads retained hard links.\n", stats.files, stats.bytes)
			return
		}
		fmt.Printf("Photo migration local copy mode: %d files copied (%d bytes, including thumbnails and separately copied attributes).\n", stats.files, stats.bytes)
	}
}

func hardLinkCopyFallback(err error) bool {
	// Link implementations (notably remote mounts) use different errors for
	// unsupported links. A verified ordinary copy is safe for every failure
	// except a missing source; copy errors still abort before references change.
	return err != nil && !errors.Is(err, os.ErrNotExist)
}

func removeLegacyPhotoFields(app core.App) error {
	fields := []struct {
		Collection string
		FieldID    string
	}{
		{Collection: "trails", FieldID: "aqbpyawe"},
		{Collection: "waypoints", FieldID: "tfhs3juh"},
		{Collection: "summit_logs", FieldID: "ixnksbkt"},
	}

	for _, f := range fields {
		collection, err := app.FindCollectionByNameOrId(f.Collection)
		if err != nil {
			return err
		}
		collection.Fields.RemoveById(f.FieldID)
		if err := app.Save(collection); err != nil {
			return err
		}
	}

	return nil
}

func isUnmodifiedSinceCreation(record *core.Record) bool {
	return record.GetDateTime("created").Time().Equal(record.GetDateTime("updated").Time())
}

func resolveAssetAuthor(app core.App, rawAuthor string) (string, error) {
	if rawAuthor == "" {
		return "", nil
	}
	if _, err := app.FindRecordById("activitypub_actors", rawAuthor); err == nil {
		return rawAuthor, nil
	}
	actor, err := app.FindFirstRecordByData("activitypub_actors", "user", rawAuthor)
	if err != nil {
		return "", nil
	}
	return actor.Id, nil
}

// Periodic output also covers validation and post-commit storage work, which
// can take longer than the database backfill on remote or copied storage.
func photoMigrationProgress(phase string, done, total int) {
	if done > 0 && (done%100 == 0 || done == total) {
		fmt.Printf("%s: %d of %d.\n", phase, done, total)
	}
}

// The source digest is authoritative only after the actual copied bytes agree.
// Metadata is a consistency check, not evidence that a CopyObject succeeded.
func checkedLegacyPhotoHash(storage photoMigrationStorage, legacy, copied string) (string, error) {
	original, err := storage.Attributes(legacy)
	if err != nil {
		return "", fmt.Errorf("stat legacy original: %w", err)
	}
	replacement, err := storage.Attributes(copied)
	if err != nil {
		return "", fmt.Errorf("stat migrated copy: %w", err)
	}
	sameFile, err := storage.sameLocalFile(legacy, copied)
	if err != nil {
		return "", err
	}
	if replacement.Size != original.Size {
		return "", fmt.Errorf("migrated copy reports %d bytes instead of %d", replacement.Size, original.Size)
	}
	if sameFile {
		// Read through the new name to verify it is readable. This is also
		// the source payload because both names resolve to the same inode.
		hash, err := checkedPhotoMigrationRead(storage.System, copied, replacement.Size)
		if err != nil {
			return "", fmt.Errorf("read migrated hard link: %w", err)
		}
		return hash, nil
	}
	hash, err := checkedPhotoMigrationRead(storage.System, legacy, original.Size)
	if err != nil {
		return "", fmt.Errorf("read legacy original: %w", err)
	}
	copyHash, err := checkedPhotoMigrationRead(storage.System, copied, replacement.Size)
	if err != nil {
		return "", fmt.Errorf("read migrated copy: %w", err)
	}
	if copyHash != hash {
		return "", errors.New("migrated copy bytes do not match the legacy original (SHA-256 mismatch)")
	}
	return hash, nil
}

func checkedPhotoMigrationRead(fsys *filesystem.System, key string, expectedSize int64) (string, error) {
	reader, err := fsys.GetReader(key)
	if err != nil {
		return "", err
	}
	return checkedPhotoMigrationDigest(reader, reader.Size(), expectedSize)
}

// Check the actual count, even when the reader reports a successful EOF.
func checkedPhotoMigrationDigest(reader io.ReadCloser, readerSize, expectedSize int64) (string, error) {
	if readerSize < 0 || readerSize != expectedSize {
		return "", errors.Join(fmt.Errorf("reader reports %d bytes, expected %d from file attributes", readerSize, expectedSize), reader.Close())
	}
	hash, size, readErr := photoMigrationContentHash(reader)
	if err := errors.Join(readErr, reader.Close()); err != nil {
		return "", err
	}
	if size != expectedSize {
		return "", fmt.Errorf("read %d bytes, expected %d from file attributes", size, expectedSize)
	}
	return hash, nil
}

func runPhotoMigrationPage[T any](items []T, task func(T) error) error {
	return runPhotoMigrationItems(slices.Values(items), len(items), task)
}

// Keep storage concurrency modest and join every worker before committing or
// rolling back its files. The count comes from the underlying map or slice.
func runPhotoMigrationItems[T any](items iter.Seq[T], count int, task func(T) error) error {
	jobs := make(chan T)
	var wait sync.WaitGroup
	var mu sync.Mutex
	var first error
	for i := 0; i < 4 && i < count; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for item := range jobs {
				mu.Lock()
				failed := first != nil
				mu.Unlock()
				if failed {
					continue
				}
				if err := task(item); err != nil {
					mu.Lock()
					if first == nil {
						first = err
					}
					mu.Unlock()
				}
			}
		}()
	}
	for item := range items {
		jobs <- item
	}
	close(jobs)
	wait.Wait()
	return first
}

func interruptedPhotoCopyMatches(app core.App, fsys *filesystem.System, copied string, sources []string, hashes map[string]string, mu *sync.Mutex) (bool, error) {
	if !app.Settings().S3.Enabled {
		copyInfo, err := os.Stat(localPhotoMigrationPath(app, copied))
		if err != nil {
			return false, err
		}
		for _, source := range sources {
			sourceInfo, err := os.Stat(localPhotoMigrationPath(app, source))
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return false, err
			}
			if os.SameFile(copyInfo, sourceInfo) {
				return true, nil
			}
		}
	} else {
		copyAttrs, err := fsys.Attributes(copied)
		if err != nil {
			return false, err
		}
		if copyAttrs.ETag != "" {
			for _, source := range sources {
				attrs, err := fsys.Attributes(source)
				if errors.Is(err, filesystem.ErrNotFound) {
					continue
				}
				if err != nil {
					return false, err
				}
				if attrs.ETag == copyAttrs.ETag && attrs.Size == copyAttrs.Size {
					return true, nil
				}
			}
		}
	}
	copiedHash, err := migrationPhotoHash(fsys, copied)
	if err != nil {
		return false, err
	}
	for _, source := range sources {
		mu.Lock()
		hash, cached := hashes[source]
		mu.Unlock()
		if !cached {
			hash, err = migrationPhotoHash(fsys, source)
			if errors.Is(err, filesystem.ErrNotFound) {
				hash, err = "", nil
			}
			if err != nil {
				return false, err
			}
			mu.Lock()
			hashes[source] = hash
			mu.Unlock()
		}
		if hash == copiedHash {
			return true, nil
		}
	}
	return false, nil
}

func photoMigrationMetadataPath(relative string) bool {
	for _, name := range strings.Split(relative, "/") {
		if strings.HasPrefix(name, ".") || name == "@eaDir" || strings.EqualFold(name, "Thumbs.db") || strings.EqualFold(name, "Desktop.ini") {
			return true
		}
	}
	return false
}

func moveMigratedPhotoProvenance(app core.App) error {
	collection, err := app.FindCollectionByNameOrId("assets")
	if err != nil {
		return err
	}
	cursor := ""
	for {
		records := []*core.Record{}
		if err := app.RecordQuery(collection).AndWhere(dbx.NewExp("id > {:cursor}", dbx.Params{"cursor": cursor})).OrderBy("id").Limit(200).All(&records); err != nil {
			return err
		}
		if len(records) == 0 {
			return nil
		}
		for _, record := range records {
			cursor = record.Id
			metadata := map[string]any{}
			if record.UnmarshalJSONField("metadata", &metadata) != nil || metadata == nil {
				continue
			}
			provenance := util.AssetProvenanceMetadata(record)
			for key := range metadata {
				if util.IsAssetProvenanceKey(key) {
					delete(metadata, key)
				}
			}
			publicJSON, err := json.Marshal(metadata)
			if err != nil {
				return err
			}
			privateJSON, err := json.Marshal(provenance)
			if err != nil {
				return err
			}
			// Avoid runtime asset hooks and the DB logger in the migration
			// runner's auxiliary transaction. IDs and files do not change.
			if _, err := app.DB().NewQuery("UPDATE assets SET metadata = {:public}, source_metadata = {:private} WHERE id = {:id}").Bind(dbx.Params{"public": string(publicJSON), "private": string(privateJSON), "id": record.Id}).Execute(); err != nil {
				return err
			}
		}
	}
}

func indexMigratedPhotoAliases(app core.App) error {
	if err := util.CreateLegacyPhotoAliasTable(app); err != nil {
		return err
	}
	cursor := ""
	for {
		records, err := app.FindRecordsByFilter("assets", "id > {:cursor}", "id", 200, 0, dbx.Params{"cursor": cursor})
		if err != nil {
			return err
		}
		if len(records) == 0 {
			return nil
		}
		for _, record := range records {
			cursor = record.Id
			if err := util.SyncAssetLegacyPhotoAliases(app, record); err != nil {
				return err
			}
		}
	}
}

// Keep the one-time migration's byte hashing independent of runtime changes.
func photoMigrationContentHash(reader io.Reader) (string, int64, error) {
	hash := sha256.New()
	size, err := io.Copy(hash, reader)
	if err != nil {
		return "", size, err
	}
	return hex.EncodeToString(hash.Sum(nil)), size, nil
}
