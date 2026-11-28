package migrations

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

func ensurePhotoMigrationCleanupQueue(app core.App) error {
	_, err := app.DB().NewQuery(`CREATE TABLE IF NOT EXISTS _photo_migration_cleanup (
 legacy_key TEXT PRIMARY KEY NOT NULL, asset_key TEXT NOT NULL, expected_hash TEXT NOT NULL, thumb_keys TEXT NOT NULL DEFAULT '[]',
 attempts INTEGER NOT NULL DEFAULT 0,last_error TEXT NOT NULL DEFAULT '',
 next_attempt_at INTEGER NOT NULL DEFAULT 0,needs_review INTEGER NOT NULL DEFAULT 0
)`).Execute()
	return err
}

func enqueuePhotoMigrationCleanup(app core.App, legacy, copied, hash string, thumbs []string) (photoMigrationCleanupItem, error) {
	if thumbs == nil {
		thumbs = []string{}
	}
	encoded, err := json.Marshal(thumbs)
	if err != nil {
		return photoMigrationCleanupItem{}, err
	}
	item := photoMigrationCleanupItem{Legacy: legacy, Copied: copied, Hash: hash, Thumbs: string(encoded)}
	_, err = app.DB().NewQuery(`INSERT INTO _photo_migration_cleanup(legacy_key,asset_key,expected_hash,thumb_keys)
 VALUES({:legacy},{:copied},{:hash},{:thumbs}) ON CONFLICT(legacy_key) DO UPDATE SET asset_key=excluded.asset_key,expected_hash=excluded.expected_hash,thumb_keys=excluded.thumb_keys,attempts=0,last_error='',next_attempt_at=0,needs_review=0`).Bind(dbx.Params{"legacy": legacy, "copied": copied, "hash": hash, "thumbs": item.Thumbs}).Execute()
	return item, err
}

// Completion callbacks run after their SQL transaction has closed. Resolve the
// dbx transaction's original database before registering the callback.
func photoMigrationCommittedDB(app core.App) (dbx.Builder, error) {
	if tx, ok := app.NonconcurrentDB().(*dbx.Tx); ok {
		if provider, ok := tx.Builder.(interface{ DB() *dbx.DB }); ok {
			return provider.DB(), nil
		}
		return nil, errors.New("photo cleanup cannot resolve committed database")
	}
	return app.DB(), nil
}

type photoMigrationCleanupItem struct {
	Legacy   string `db:"legacy_key"`
	Copied   string `db:"asset_key"`
	Hash     string `db:"expected_hash"`
	Thumbs   string `db:"thumb_keys"`
	Attempts int    `db:"attempts"`
	Review   bool   `db:"needs_review"`
}

// Keep the filesystem and the hard-link proof on the same storage backend,
// even if live PocketBase settings change during a resumed cleanup.
type photoMigrationStorage struct {
	*filesystem.System
	localRoot string // empty for S3
}

func newPhotoMigrationStorage(app core.App) (photoMigrationStorage, error) {
	settings, err := app.Settings().Clone()
	if err != nil {
		return photoMigrationStorage{}, err
	}
	s3 := settings.S3
	if s3.Enabled {
		fs, err := filesystem.NewS3(s3.Bucket, s3.Region, s3.Endpoint, s3.AccessKey, s3.Secret, s3.ForcePathStyle)
		return photoMigrationStorage{System: fs}, err
	}
	root := filepath.Join(app.DataDir(), core.LocalStorageDirName)
	fs, err := filesystem.NewLocal(root)
	return photoMigrationStorage{System: fs, localRoot: root}, err
}

func (storage photoMigrationStorage) localPath(key string) string {
	return filepath.Join(storage.localRoot, filepath.FromSlash(key))
}

// A symbolic link to the legacy path must not count as an independent link
// to its inode. The two paths must also be distinct directory entries.
func (storage photoMigrationStorage) sameLocalFile(legacy, copied string) (bool, error) {
	if storage.localRoot == "" {
		return false, nil
	}
	legacyPath, copiedPath := storage.localPath(legacy), storage.localPath(copied)
	original, err := os.Lstat(legacyPath)
	if err != nil {
		return false, err
	}
	replacement, err := os.Lstat(copiedPath)
	if err != nil {
		return false, err
	}
	if !original.Mode().IsRegular() || !replacement.Mode().IsRegular() {
		return false, errors.New("legacy original and migrated copy must be regular files")
	}
	if !os.SameFile(original, replacement) {
		return false, nil
	}
	originalDir, err := os.Stat(filepath.Dir(legacyPath))
	if err != nil {
		return false, err
	}
	replacementDir, err := os.Stat(filepath.Dir(copiedPath))
	if err != nil {
		return false, err
	}
	if os.SameFile(originalDir, replacementDir) {
		return false, errors.New("legacy original and migrated copy resolve to the same directory")
	}
	return true, nil
}

// ResumePhotoMigrationCleanup is deliberately separate from the migration
// transaction: a restart must serve requests while remaining files are cleaned.
// Failed items retain durable retry state; ambiguous items remain for review.
// On the first pass after startup, reportDetails lists retained rows and allows
// already-missing originals to finish cleanup even if previously suspended.
func ResumePhotoMigrationCleanup(ctx context.Context, app core.App, reportDetails bool) error {
	if app.TxInfo() != nil {
		return errors.New("photo cleanup must run after migration commit")
	}
	cleaned, attempted := 0, 0
	defer func() {
		if reportDetails || attempted > 0 || cleaned > 0 {
			reportRetainedPhotoMigrationCleanup(app, cleaned, reportDetails)
		}
	}()
	storage, err := newPhotoMigrationStorage(app)
	if err != nil {
		return err
	}
	defer storage.Close()
	storage.SetContext(ctx)
	database := app.DB()
	cursor := ""
	var firstError error
	now := time.Now().UnixMilli()
	var pending int
	const eligible = "((needs_review=0 AND next_attempt_at<={:now}) OR (needs_review=1 AND {:review}))"
	if err := database.NewQuery("SELECT COUNT(*) FROM _photo_migration_cleanup WHERE " + eligible).Bind(dbx.Params{"now": now, "review": reportDetails}).Row(&pending); err != nil {
		if strings.Contains(err.Error(), "no such table: _photo_migration_cleanup") {
			return nil
		}
		return err
	}
	_, markerErr := storage.Attributes(photoMigrationCommitMarker)
	if markerErr != nil && !errors.Is(markerErr, filesystem.ErrNotFound) {
		return markerErr
	}
	marked := markerErr == nil
	if pending == 0 && marked {
		return ensurePhotoMigrationCommitMarker(storage)
	}
	ready, err := preparePhotoMigrationCleanup(database, storage, !marked)
	if err != nil || !ready {
		return err
	}
	if pending == 0 {
		return nil
	}
	fmt.Printf("Resuming cleanup of %d migrated legacy photos...\n", pending)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		items := []photoMigrationCleanupItem{}
		if err := database.NewQuery("SELECT legacy_key,asset_key,expected_hash,thumb_keys,attempts,needs_review FROM _photo_migration_cleanup WHERE legacy_key>{:cursor} AND " + eligible + " ORDER BY legacy_key LIMIT 200").Bind(dbx.Params{"cursor": cursor, "now": now, "review": reportDetails}).All(&items); err != nil {
			if strings.Contains(err.Error(), "no such table: _photo_migration_cleanup") {
				return nil
			}
			return err
		}
		if len(items) == 0 {
			return firstError
		}
		cursor = items[len(items)-1].Legacy
		var resultMu sync.Mutex
		pageErr := runPhotoMigrationPage(items, func(item photoMigrationCleanupItem) error {
			recordError := func(err error) {
				resultMu.Lock()
				if firstError == nil {
					firstError = err
				}
				resultMu.Unlock()
			}
			resultMu.Lock()
			attempted++
			photoMigrationProgress("Resuming migrated legacy photo cleanup", attempted, pending)
			resultMu.Unlock()
			if err := ctx.Err(); err != nil {
				return err
			}
			// Don't retry suspended items while the original is still present.
			// If it is gone, require the same proof of a current replacement.
			if item.Review {
				_, err := storage.Attributes(item.Legacy)
				if err == nil {
					return nil
				}
				if !errors.Is(err, filesystem.ErrNotFound) {
					recordError(err)
					return nil
				}
			}
			next := time.Now().Add(photoMigrationCleanupBackoff(item.Attempts + 1)).UnixMilli()
			claim, err := database.NewQuery("UPDATE _photo_migration_cleanup SET attempts=attempts+1,last_error='interrupted',next_attempt_at={:next},needs_review=0 WHERE legacy_key={:key} AND attempts={:attempts} AND needs_review={:review} AND (needs_review=1 OR next_attempt_at<={:now})").Bind(dbx.Params{"next": next, "key": item.Legacy, "attempts": item.Attempts, "now": now, "review": item.Review}).WithContext(ctx).Execute()
			if err != nil {
				recordError(err)
				return nil
			}
			affected, _ := claim.RowsAffected()
			if affected == 0 {
				return nil
			}
			if code, err := cleanPhotoMigrationCleanupItem(database, storage, item); err != nil {
				recordError(retainPhotoMigrationCleanup(ctx, database, item, next, code, err))
			} else {
				resultMu.Lock()
				cleaned++
				resultMu.Unlock()
			}
			return nil
		})
		if pageErr != nil {
			return pageErr
		}
	}
}

func retainPhotoMigrationCleanup(ctx context.Context, database dbx.Builder, item photoMigrationCleanupItem, next int64, code string, err error) error {
	review := errors.Is(err, errPhotoMigrationCleanupNeedsReview)
	var reviewError *photoMigrationCleanupReviewError
	if errors.As(err, &reviewError) {
		code = reviewError.code
	}
	if ctx.Err() != nil {
		code = "interrupted"
		next = 0
	}
	if _, writeErr := database.NewQuery("UPDATE _photo_migration_cleanup SET last_error={:error},needs_review={:review},next_attempt_at={:next} WHERE legacy_key={:key}").Bind(dbx.Params{"error": code, "review": review, "next": next, "key": item.Legacy}).Execute(); writeErr != nil {
		err = errors.Join(err, fmt.Errorf("retain photo cleanup item: %w", writeErr))
	}
	photoMigrationWarning("legacy photo cleanup failed", "key", item.Legacy, "code", code, "error", err)
	return err
}

// Both the transaction's synchronous callback and resumed cleanup use this
// exact deletion path. A queue item is only checkpointed after verified cleanup.
func cleanPhotoMigrationCleanupItem(database dbx.Builder, storage photoMigrationStorage, item photoMigrationCleanupItem) (string, error) {
	var thumbs []string
	if err := json.Unmarshal([]byte(item.Thumbs), &thumbs); err != nil {
		return "invalid_thumbnail_keys", fmt.Errorf("%w: %v", errPhotoMigrationCleanupNeedsReview, err)
	}
	directory, name := path.Split(item.Legacy)
	prefix := directory + "thumbs_" + name + "/"
	for _, key := range thumbs {
		if !strings.HasPrefix(key, prefix) || !canonicalPhotoMigrationKey(key) {
			return "invalid_thumbnail_keys", fmt.Errorf("%w: unexpected thumbnail key %q", errPhotoMigrationCleanupNeedsReview, key)
		}
	}

	legacyPresent, err := verifyPhotoMigrationCleanup(database, storage, item)
	if err != nil {
		return "verification_failed", err
	}
	var thumbErrors []error
	for _, thumb := range thumbs {
		if err := storage.Delete(thumb); err != nil && !errors.Is(err, filesystem.ErrNotFound) {
			thumbErrors = append(thumbErrors, err)
		}
	}
	if err := removeEmptyLegacyPhotoThumbDirectory(storage, item.Legacy); err != nil {
		thumbErrors = append(thumbErrors, err)
	}
	if len(thumbErrors) > 0 {
		return "thumbnail_delete_failed", errors.Join(thumbErrors...)
	}

	if len(thumbs) > 0 {
		// Thumbnail deletes can take time. Reverify immediately before the
		// original, while preserving the thumbnail-first failure semantics.
		legacyPresent, err = verifyPhotoMigrationCleanup(database, storage, item)
		if err != nil {
			return "verification_failed", err
		}
	}
	if legacyPresent {
		if err := storage.Delete(item.Legacy); err != nil && !errors.Is(err, filesystem.ErrNotFound) {
			return "legacy_delete_failed", err
		}
	}
	if err := removeEmptyLegacyWaypointDirectory(storage, item.Legacy); err != nil {
		return "directory_delete_failed", err
	}
	if _, err := database.NewQuery("DELETE FROM _photo_migration_cleanup WHERE legacy_key={:key}").Bind(dbx.Params{"key": item.Legacy}).Execute(); err != nil {
		return "checkpoint_failed", err
	}
	return "", nil
}

func canonicalPhotoMigrationKey(key string) bool {
	return path.Clean(key) == key && !strings.HasPrefix(key, "/") && key != "." && key != ".." && !strings.HasPrefix(key, "../") &&
		!strings.ContainsFunc(key, func(r rune) bool { return r < 32 || r == '\\' })
}

func verifyPhotoMigrationCleanup(database dbx.Builder, storage photoMigrationStorage, item photoMigrationCleanupItem) (bool, error) {
	legacy := strings.Split(item.Legacy, "/")
	if len(legacy) != 3 || legacy[0] == photoMigrationAssetCollectionID || !canonicalPhotoMigrationKey(item.Legacy) {
		return false, &photoMigrationCleanupReviewError{code: "invalid_legacy_path"}
	}
	copied := strings.Split(item.Copied, "/")
	if len(copied) != 3 || copied[0] != photoMigrationAssetCollectionID || copied[2] != legacy[2] || !canonicalPhotoMigrationKey(item.Copied) {
		return false, &photoMigrationCleanupReviewError{code: "invalid_copy_path"}
	}

	copyAttrs, err := storage.Attributes(item.Copied)
	if err != nil {
		return false, photoMigrationCopyReadError("stat migrated copy", err)
	}
	legacyAttrs, err := storage.Attributes(item.Legacy)
	legacyPresent := err == nil
	if errors.Is(err, filesystem.ErrNotFound) {
		// Check that the body agrees with the attributes. A missing original
		// never needs a DELETE, but still requires a verified replacement.
		if readErr := confirmMissingLegacyPhoto(storage.System, item.Legacy); readErr != nil {
			return false, fmt.Errorf("check missing legacy original: %w", readErr)
		}
	} else if err != nil {
		return false, fmt.Errorf("stat legacy original: %w", err)
	}

	sameFile := false
	if legacyPresent {
		sameFile, err = storage.sameLocalFile(item.Legacy, item.Copied)
		if err != nil {
			return false, err
		}
		if sameFile {
			if copyAttrs.Size != legacyAttrs.Size {
				return false, errors.New("hard-linked original and copy report different sizes")
			}
			reader, err := storage.GetReader(item.Copied)
			if err != nil {
				return false, photoMigrationCopyReadError("open migrated hard link", err)
			}
			size, err := reader.Size(), reader.Close()
			if err != nil {
				return false, fmt.Errorf("close migrated hard link: %w", err)
			}
			if size < 0 || size != copyAttrs.Size {
				return false, fmt.Errorf("migrated hard link reports %d bytes, expected %d from file attributes", size, copyAttrs.Size)
			}
		} else {
			hash, err := checkedPhotoMigrationRead(storage.System, item.Legacy, legacyAttrs.Size)
			if err != nil {
				return false, fmt.Errorf("read legacy original: %w", err)
			}
			if hash != item.Hash {
				return false, &photoMigrationCleanupReviewError{code: "legacy_media_changed", cause: errors.New("legacy original bytes do not match the recorded migration digest")}
			}
		}
	}
	if !sameFile {
		hash, err := checkedPhotoMigrationRead(storage.System, item.Copied, copyAttrs.Size)
		if err != nil {
			return false, photoMigrationCopyReadError("read migrated copy", err)
		}
		if hash != item.Hash {
			return false, &photoMigrationCleanupReviewError{code: "migrated_copy_changed", cause: errors.New("migrated copy bytes do not match the recorded migration digest")}
		}
	}

	// Check the current canonical asset path, not just an object that happens
	// to exist at the recorded key. Do this after the potentially long reads.
	var currentKey, mode, status string
	err = database.NewQuery(`SELECT c.id || '/' || a.id || '/' || COALESCE(a.file, ''), COALESCE(a.storage_mode, ''), COALESCE(a.remote_status, '')
 FROM assets a JOIN _collections c ON c.name='assets' WHERE a.id={:id}`).Bind(dbx.Params{"id": copied[1]}).Row(&currentKey, &mode, &status)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, fmt.Errorf("check current asset reference: %w", err)
	}
	if err != nil || currentKey != item.Copied || mode != "copy" {
		return false, &photoMigrationCleanupReviewError{code: "migrated_copy_unreferenced"}
	}
	if status != "available" {
		return false, &photoMigrationCleanupReviewError{code: "migrated_copy_unavailable"}
	}
	return legacyPresent, nil
}

func photoMigrationCopyReadError(operation string, err error) error {
	if errors.Is(err, filesystem.ErrNotFound) {
		return &photoMigrationCleanupReviewError{code: "migrated_copy_missing", cause: err}
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func confirmMissingLegacyPhoto(fsys *filesystem.System, key string) error {
	reader, err := fsys.GetReader(key)
	if errors.Is(err, filesystem.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return errors.Join(errors.New("legacy original was missing in attributes but readable"), reader.Close())
}

var errPhotoMigrationCleanupNeedsReview = errors.New("photo cleanup needs manual review")

type photoMigrationCleanupReviewError struct {
	code  string
	cause error
}

func (err *photoMigrationCleanupReviewError) Error() string {
	message := "photo cleanup needs manual review: " + err.code
	if err.cause != nil {
		message += ": " + err.cause.Error()
	}
	return message
}
func (err *photoMigrationCleanupReviewError) Unwrap() []error {
	if err.cause == nil {
		return []error{errPhotoMigrationCleanupNeedsReview}
	}
	return []error{errPhotoMigrationCleanupNeedsReview, err.cause}
}

func photoMigrationCleanupBackoff(attempt int) time.Duration {
	return min(24*time.Hour, time.Hour*time.Duration(1<<min(5, max(0, attempt-1))))
}

// Diagnostics stay private, contain stable categories rather than provider
// messages, and are visible in ordinary production container logs.
func reportRetainedPhotoMigrationCleanup(app core.App, cleaned int, details bool) {
	type retainedItem struct {
		Key    string `db:"legacy_key"`
		Error  string `db:"last_error"`
		Next   int64  `db:"next_attempt_at"`
		Review bool   `db:"needs_review"`
	}
	retained := 0
	if !details {
		if err := app.DB().NewQuery("SELECT COUNT(*) FROM _photo_migration_cleanup").Row(&retained); err != nil {
			return
		}
	} else {
		cursor := ""
		for {
			rows := []retainedItem{}
			err := app.DB().NewQuery("SELECT legacy_key,last_error,next_attempt_at,needs_review FROM _photo_migration_cleanup WHERE legacy_key>{:cursor} ORDER BY legacy_key LIMIT 200").Bind(dbx.Params{"cursor": cursor}).All(&rows)
			if err != nil {
				return
			}
			if len(rows) == 0 {
				break
			}
			for _, item := range rows {
				cursor = item.Key
				retained++
				state := "pending"
				if item.Review {
					state = "needs_review"
				} else if item.Error != "" {
					state = "retry_deferred"
				}
				fields := []any{"key", item.Key, "state", state, "last_error", item.Error, "next_attempt_at", item.Next}
				switch item.Error {
				case "legacy_media_changed", "migrated_copy_changed":
					fields = append(fields, "note", "a mismatch with the recorded digest does not identify the correct object")
				}
				photoMigrationWarning("legacy photo cleanup retained", fields...)
			}
		}
	}
	if cleaned > 0 || retained > 0 {
		fmt.Fprintf(os.Stderr, "Photo migration cleanup: %d cleaned, %d retained for retry or review.\n", cleaned, retained)
	}
}

// Direct key deletion preserves unrelated files. Remove the now empty local
// thumbnail directory too so a later record deletion can remove its parent.
func removeEmptyLegacyPhotoThumbDirectory(storage photoMigrationStorage, key string) error {
	if storage.localRoot == "" {
		return nil
	}
	directory, name := path.Split(key)
	err := os.Remove(storage.localPath(directory + "thumbs_" + name))
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENOTEMPTY) {
		return nil
	}
	return err
}

// RunPhotoMigrationCleanup keeps deferred work moving while the server runs.
// Items needing manual review are inspected at startup but do not keep it alive.
func RunPhotoMigrationCleanup(ctx context.Context, app core.App) error {
	first := true
	for {
		resumeErr := ResumePhotoMigrationCleanup(ctx, app, first)
		first = false
		if resumeErr != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			app.Logger().Warn("photo migration cleanup requires retry", "error", resumeErr)
		}
		var next sql.NullInt64
		err := app.DB().NewQuery("SELECT MIN(next_attempt_at) FROM _photo_migration_cleanup WHERE needs_review=0").Row(&next)
		if err != nil {
			if strings.Contains(err.Error(), "no such table: _photo_migration_cleanup") {
				return nil
			}
			return err
		}
		if !next.Valid {
			fs, err := app.NewFilesystem()
			if err == nil {
				_, err = fs.Attributes(photoMigrationCommitMarker)
				fs.Close()
				if err == nil && resumeErr == nil {
					return nil
				}
			}
			// An empty library still needs its committed-storage marker.
			// Retry infrastructure failures without delaying server startup.
			next.Int64 = time.Now().Add(time.Minute).UnixMilli()
		}
		delay := min(time.Hour, time.Until(time.UnixMilli(next.Int64)))
		if delay <= 0 {
			// Infrastructure errors before claiming a row must not spin.
			delay = time.Minute
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

const photoMigrationAssetCollectionID = "assetcollect001"
const photoMigrationCommitMarker = photoMigrationAssetCollectionID + "/.photo-migration-committed"

// The marker is written only after committed rows and copied media are durable.
// A busy checkpoint leaves every queue row and original for the resume worker.
func preparePhotoMigrationCleanup(database dbx.Builder, storage photoMigrationStorage, syncCopies bool) (bool, error) {
	if database == nil {
		return false, errors.New("photo cleanup has no committed database")
	}
	var busy, logPages, checkpointed int
	if err := database.NewQuery("PRAGMA wal_checkpoint(FULL)").Row(&busy, &logPages, &checkpointed); err != nil {
		return false, fmt.Errorf("checkpoint photo migration: %w", err)
	}
	if busy != 0 {
		photoMigrationWarning("legacy photo cleanup deferred until database checkpoint completes")
		return false, nil
	}
	if storage.localRoot != "" {
		root := storage.localPath(photoMigrationAssetCollectionID)
		if err := syncPhotoMigrationCopies(root, syncCopies); err != nil {
			return false, fmt.Errorf("make copied photos durable: %w", err)
		}
	}
	if err := ensurePhotoMigrationCommitMarker(storage); err != nil {
		return false, err
	}
	return true, nil
}

// Sync directory entries children before parents, including thumbnail directories.
// Copied file contents and attribute sidecars also need syncing; hard-linked files
// share already-written inodes but still need their new directory entries synced.
func syncPhotoMigrationCopies(root string, syncFiles bool) error {
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := filepath.Join(root, entry.Name())
		if entry.IsDir() {
			if err := syncPhotoMigrationCopies(name, syncFiles); err != nil {
				return err
			}
		} else if !entry.Type().IsRegular() {
			return fmt.Errorf("unexpected file type in photo copies: %s", name)
		} else if syncFiles {
			if err := syncPhotoMigrationPath(name); err != nil && !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	}
	err = syncPhotoMigrationPath(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func syncPhotoMigrationPath(name string) error {
	file, err := os.Open(name)
	if err != nil {
		return err
	}
	return errors.Join(file.Sync(), file.Close())
}

func ensurePhotoMigrationCommitMarker(storage photoMigrationStorage) error {
	if _, err := storage.Attributes(photoMigrationCommitMarker); err == nil {
		if storage.localRoot == "" {
			return nil
		}
		name := storage.localPath(photoMigrationCommitMarker)
		directory := filepath.Dir(name)
		return errors.Join(syncPhotoMigrationPath(name), syncPhotoMigrationPath(directory), syncPhotoMigrationPath(filepath.Dir(directory)))
	} else if !errors.Is(err, filesystem.ErrNotFound) {
		return err
	}
	if storage.localRoot == "" {
		return storage.Upload([]byte("committed\n"), photoMigrationCommitMarker)
	}
	name := storage.localPath(photoMigrationCommitMarker)
	directory := filepath.Dir(name)
	if err := os.MkdirAll(directory, 0o777); err != nil {
		return err
	}
	file, err := os.CreateTemp(directory, ".photo-migration-marker-")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.WriteString("committed\n"); err != nil {
		return errors.Join(err, file.Close())
	}
	if err := errors.Join(file.Sync(), file.Close()); err != nil {
		return err
	}
	if err := os.Rename(file.Name(), name); err != nil {
		return err
	}
	return errors.Join(syncPhotoMigrationPath(directory), syncPhotoMigrationPath(filepath.Dir(directory)))
}

func discardUncommittedPhotoMigration(app core.App, fs *filesystem.System, collectionID string) (bool, error) {
	if app.Settings().S3.Enabled {
		objects, err := fs.List(collectionID + "/")
		if err != nil {
			return false, err
		}
		if len(objects) == 0 {
			return false, nil
		}
		return true, errors.Join(fs.DeletePrefix(collectionID + "/")...)
	}
	root := localPhotoMigrationPath(app, collectionID)
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(root, entry.Name())); err != nil {
			return false, err
		}
	}
	return len(entries) > 0, nil
}

// Waypoints no longer have any file fields, so their empty directories need a
// final explicit removal. Other record directories remain PocketBase's concern.
func removeEmptyLegacyWaypointDirectory(storage photoMigrationStorage, key string) error {
	if storage.localRoot == "" || !strings.HasPrefix(key, "goeo2ubp103rzp9/") {
		return nil
	}
	err := os.Remove(storage.localPath(path.Dir(key)))
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, syscall.ENOTEMPTY) {
		return nil
	}
	return err
}
