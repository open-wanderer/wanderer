package routes

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"pocketbase/federation"
	"pocketbase/util"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

// remoteSyncThreshold is the minimum age of a remote record before a background sync is triggered.
// Configurable via POCKETBASE_FEDERATION_SYNC_INTERVAL (minutes). Default: 60.
var remoteSyncThreshold = func() time.Duration {
	if v := os.Getenv("POCKETBASE_FEDERATION_SYNC_INTERVAL"); v != "" {
		if minutes, err := strconv.Atoi(v); err == nil && minutes > 0 {
			return time.Duration(minutes) * time.Minute
		}
	}
	return 60 * time.Minute
}()

// trailSyncing and listSyncing track IRIs currently being synced to prevent concurrent duplicate syncs.
var trailSyncing sync.Map
var listSyncing sync.Map

var errRemoteUnavailable = errors.New("remote instance unavailable")

// cachedRecordFallback returns the locally cached record to serve when a
// blocking sync failed for a reason that says nothing about the content
// itself. Only a copy that completed a full sync at least once qualifies: a
// placeholder created from an inbound Create/Announce has a stored id but no
// content, and serving it would present an empty trail/list as real.
func cachedRecordFallback(app core.App, collection, id string, err error) *core.Record {
	if id == "" || !isDegradableSyncError(err) {
		return nil
	}

	cached, findErr := app.FindRecordById(collection, id)
	if findErr != nil || !hasCompletedFullSync(cached) {
		return nil
	}

	return cached
}

func hasCompletedFullSync(record *core.Record) bool {
	return record.GetBool("full_sync_completed")
}

// isDegradableSyncError reports whether a failed sync may be answered from the
// local cache
func isDegradableSyncError(err error) bool {
	return errors.Is(err, errRemoteUnavailable) || errors.Is(err, util.ErrRateLimited) || isRemoteDocumentLimitError(err)
}

func RemoteTrailGet(e *core.RequestEvent) error {
	handle := e.Request.URL.Query().Get("handle")
	trailID := e.Request.PathValue("id")

	var record *core.Record
	var err error

	var userActor *core.Record
	if e.Auth != nil {
		userActor, _ = e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
	}

	ctx, err := util.GetSafeActorContext(e.Request, userActor)
	if err != nil {
		return err
	}

	// 1. Resolve the "Actual" Record or Shell
	if handle != "" {
		// If we have a handle, we are looking for a remote trail.
		// Construct the IRI first to see if we already know this trail.
		record, err = findLocalTrailByRemoteInfo(e, ctx, handle, trailID)
		if err != nil {
			if errors.Is(err, federation.ErrProfilePrivate) || errors.Is(err, federation.ErrActorNotFound) {
				return e.NotFoundError("profile is private", err)
			}
			if errors.Is(err, util.ErrRateLimited) {
				return e.TooManyRequestsError("Too many requests", err)
			}
			if errors.Is(err, errRemoteUnavailable) {
				return apis.NewApiError(http.StatusServiceUnavailable, "Remote instance unavailable", nil)
			}
			return e.InternalServerError("Failed to resolve trail", err)
		}

		// If the record has no ID, it's a new Shell
		if record.Id == "" || record.GetBool("needs_full_sync") && (!hasCompletedFullSync(record) || util.IsLocalIRI(record.GetString("iri")) || remoteRecordHasLocalAuthor(e.App, record)) {
			// Blocking sync for new records
			cachedID := record.Id
			record, err = performFullSync(e.App, ctx, e.Request.URL, record)
			if err == nil && record.Id != "" && record.GetBool("needs_full_sync") && !hasCompletedFullSync(record) {
				// A newer save during the network phase left only the stored
				// placeholder; refresh once more from it.
				record, err = performFullSync(e.App, ctx, e.Request.URL, record)
			}
			if err != nil {
				if errors.Is(err, errRemoteGone) {
					dropGoneRemoteRecord(e.App, "trails", cachedID)
					return e.NotFoundError("Trail not found", err)
				}
				cached := cachedRecordFallback(e.App, "trails", cachedID, err)
				if cached == nil {
					if isRemoteDocumentLimitError(err) {
						return remoteDocumentLimitError(e.App, record, "trails")
					}
					if errors.Is(err, util.ErrRateLimited) {
						return e.TooManyRequestsError("Too many requests", err)
					}
					if errors.Is(err, errRemoteUnavailable) {
						return apis.NewApiError(http.StatusServiceUnavailable, "Remote instance unavailable", nil)
					}
					return e.InternalServerError("Sync failed", err)
				}

				e.App.Logger().Warn(
					"serving cached trail after failed remote sync",
					"iri", cached.GetString("iri"),
					"error", err,
				)
				record = cached
			}
			if record.Id == "" {
				// Local content that does not exist (e.g. a stale URL to a
				// missing local trail): performFullSync short-circuits local
				// IRIs and returns the unsaved shell — surface a real 404
				// instead of running access/expand on a non-existent record.
				return e.NotFoundError("Trail not found", nil)
			}
		} else {
			// We already have it locally. Show and update background.
			updatedAt := record.GetDateTime("updated").Time()

			iri := record.GetString("iri")
			if record.GetBool("needs_full_sync") || time.Now().UTC().Sub(updatedAt) > remoteSyncThreshold {
				if _, alreadySyncing := trailSyncing.LoadOrStore(iri, struct{}{}); !alreadySyncing {
					urlCopy := *e.Request.URL
					bgCtx := context.WithValue(context.Background(), "actor", ctx.Value("actor"))
					// Sync hooks and remote data must not mutate the response record.
					syncRecord := record.Fresh()
					go func() {
						defer trailSyncing.Delete(iri)
						_, err := performFullSync(e.App, bgCtx, &urlCopy, syncRecord)
						switch {
						case errors.Is(err, errRemoteGone):
							dropGoneRemoteRecord(e.App, "trails", syncRecord.Id)
						case err != nil:
							backgroundSyncFailed(e.App, "trails", iri, err)
						}
					}()
				}
			}
		}
	} else {
		// Standard local fetch by ID
		record, err = e.App.FindRecordById("trails", trailID)
		if err != nil {
			return e.NotFoundError("Trail not found", nil)
		}
	}

	reqInfo, err := e.RequestInfo()
	if err != nil {
		return err
	}

	canAccess, err := e.App.CanAccessRecord(record, reqInfo, record.Collection().ViewRule)

	if err != nil || !canAccess {
		return e.ForbiddenError("forbidden", err)
	}

	return expandAndReturn(e, record)
}

func findLocalTrailByRemoteInfo(e *core.RequestEvent, ctx context.Context, handle, trailID string) (*core.Record, error) {
	// 1. Get Actor to build the IRI
	actor, err := resolveRemoteHandleActor(e, ctx, handle)
	if actor == nil {
		// No actor available at all — no cache, and the live lookup itself
		// failed (e.g. remote instance unreachable, unknown handle). We can't
		// construct the IRI, but a previously-synced trail may still exist
		// locally under this exact ID, so fall back to a plain local lookup
		// instead of failing the whole request. Skip this fallback for a
		// private-profile error so the caller's 404 handling still applies.
		if err != nil && !errors.Is(err, federation.ErrProfilePrivate) {
			if local, localErr := e.App.FindRecordById("trails", trailID); localErr == nil {
				return local, nil
			}
		}
		return nil, err
	}

	// Actor is non-nil even though err may be set: federation.GetActorByHandle
	// (via assembleActor) returns a previously-cached actor record alongside a
	// transport error when a background refresh of stale actor data fails.
	// That cached actor's IRI is still usable to look up the local trail, so
	// use it instead of discarding it and 500ing the whole request.
	actorURL, _ := url.Parse(actor.GetString("iri"))
	iri := fmt.Sprintf("%s://%s/api/v1/trail/%s", actorURL.Scheme, actorURL.Host, trailID)

	// 2. Check if this IRI already exists in our DB
	existing, _ := e.App.FindFirstRecordByFilter("trails", "iri={:iri}", dbx.Params{"iri": iri})
	if existing == nil {
		candidate, _ := e.App.FindRecordById("trails", trailID)
		if candidate != nil && remoteHandleSelectsRecord(e.App, candidate, actor, iri) {
			existing = candidate
		}
	}
	if existing != nil {
		return existing, nil
	}

	// No cached copy of this trail exists locally. If the actor fetch failed
	// for a real (non-private) reason, there's nothing usable to fall back to
	// — surface the original error rather than fabricating a shell that a
	// subsequent sync attempt would just fail on again.
	if err != nil && !errors.Is(err, federation.ErrProfilePrivate) {
		return nil, err
	}

	// 3. Not found? Return a new Shell
	collection, _ := e.App.FindCollectionByNameOrId("trails")
	shell := core.NewRecord(collection)
	shell.Set("iri", iri)
	shell.Set("author", actor.Id)
	shell.Set("like_count", 0)

	return shell, nil
}

// --- Core Sync Logic ---

func performFullSync(app core.App, ctx context.Context, reqURL *url.URL, localTrail *core.Record) (*core.Record, error) {
	return limitedRemoteSync(app, ctx, "trails", localTrail, func(ctx context.Context) (*core.Record, error) {
		return performFullSyncDocument(app, ctx, reqURL, localTrail)
	})
}

func performFullSyncDocument(app core.App, ctx context.Context, reqURL *url.URL, localTrail *core.Record) (*core.Record, error) {
	ctx, cancel := context.WithTimeout(ctx, remoteSyncBudget)
	defer cancel()
	iri := localTrail.GetString("iri")

	// Never federate with ourselves: a trail whose IRI is empty or points back
	// to this instance is local content (we are the source of truth). Syncing it
	// would fetch our own origin (or fail on an empty URL); just clear the stale
	// flag so the record is no longer stuck in a permanent re-sync loop.
	if iri == "" || util.IsLocalIRI(iri) || remoteRecordHasLocalAuthor(app, localTrail) {
		if localTrail.GetBool("needs_full_sync") {
			localTrail.Set("needs_full_sync", false)
			if err := app.Save(localTrail); err != nil {
				return localTrail, err
			}
		}
		return localTrail, nil
	}

	client := newRemoteSyncHTTPClient()
	remoteUrl, _ := url.Parse(iri)
	query := url.Values{}
	if reqURL != nil {
		query = reqURL.Query()
	}
	query.Del("handle")
	expands := strings.Split(query.Get("expand"), ",")
	hasAuthor := false
	for _, path := range expands {
		if strings.TrimSpace(path) == "author" {
			hasAuthor = true
			break
		}
	}
	if !hasAuthor {
		expands = append(expands, "author")
	}
	query.Set("expand", strings.Trim(strings.Join(expands, ","), ","))
	remoteUrl.RawQuery = query.Encode()
	origin := fmt.Sprintf("%s://%s", remoteUrl.Scheme, remoteUrl.Host)

	// A save after this snapshot may contain changes the fetched document lacks.
	storedUpdated, storedBeforeFetch := "", false
	if localTrail.Id != "" {
		stored, err := app.FindRecordById("trails", localTrail.Id)
		if err == nil {
			storedUpdated, storedBeforeFetch = stored.GetString("updated"), true
		} else if !errors.Is(err, sql.ErrNoRows) {
			return localTrail, err
		}
	}
	document, err := fetchRemoteSyncDocument(ctx, client, remoteUrl, "trail")
	if err != nil {
		return localTrail, err
	}
	defer document.close()
	identity, err := document.identity(app, ctx, localTrail)
	if err != nil {
		return localTrail, err
	}
	author, err := resolveRemoteTopLevelAuthor(app, ctx, localTrail, identity)
	if err != nil {
		return localTrail, fmt.Errorf("%w: %w", errRemoteUnavailable, err)
	}
	plan, err := document.networkPlan(app, ctx, localTrail, origin, "trail")
	if err != nil {
		return localTrail, err
	}
	remoteMap := plan.data

	canonicalLogs := fetchCanonicalRemoteSummitLogs(app, ctx, remoteMap, localTrail, origin)
	ctx = prepareRemoteChildAuthors(app, ctx, remoteMap, origin, "summit_logs_via_trail", "summit-log")
	ctx, cleanupMedia, err := prepareCollectedRemoteTrailMedia(app, ctx, plan.media, canonicalLogs)
	if err != nil {
		return localTrail, err
	}
	defer cleanupMedia()
	return document.process(ctx, func(remoteMap map[string]any) (*core.Record, error) {
		remoteID, _ := remoteMap["id"].(string)
		capRemoteSyncChildren(app, remoteMap, origin, "waypoints_via_trail", "summit_logs_via_trail")
		ctx, err = prepareRemoteSyncTags(app, ctx, remoteMap, author, origin, "trail")
		if err != nil {
			return localTrail, err
		}

		// Store metadata before the child batches; an incomplete refresh remains
		// eligible for another sync if a later batch cannot be saved.
		var newer *core.Record
		err = app.RunInTransaction(func(txApp core.App) error {
			if storedBeforeFetch {
				current, err := txApp.FindRecordById("trails", localTrail.Id)
				if errors.Is(err, sql.ErrNoRows) {
					return fmt.Errorf("%w: %w", errRemoteGone, err)
				}
				if err != nil {
					return err
				}
				if current.GetString("updated") != storedUpdated {
					newer = current
					return nil
				}
			}
			localTrail.Set("author", author)
			syncRecordFiles(ctx, localTrail, "trails", remoteID, origin, remoteMap)
			syncTrailMetadata(txApp, ctx, localTrail, remoteMap)
			localTrail.Set("author", author)
			localTrail.Set("needs_full_sync", true)
			util.SanitizeHTMLFieldsWithLimits(localTrail)
			return txApp.Save(localTrail)
		})
		if err != nil {
			return localTrail, err
		}
		if newer != nil {
			return newer, nil
		}
		expand, _ := remoteMap["expand"].(map[string]any)
		if waypoints, ok := expand["waypoints_via_trail"].([]any); ok {
			err = remoteSyncChildBatches(app, ctx, waypoints, func(tx core.App, batch []any) error { return syncWaypoints(tx, ctx, localTrail, origin, batch) })
			if err != nil {
				return finishRemoteTrailSync(app, ctx, localTrail, err)
			}
		}
		if logs, ok := expand["summit_logs_via_trail"].([]any); ok {
			err = remoteSyncChildBatches(app, ctx, logs, func(tx core.App, batch []any) error {
				return syncSummitLogs(tx, ctx, localTrail, origin, batch, canonicalLogs)
			})
			if err != nil {
				return finishRemoteTrailSync(app, ctx, localTrail, err)
			}
		}
		return finishRemoteTrailSync(app, ctx, localTrail, nil)
	})
}

// --- Sub-Sync Helpers ---

func syncTrailMetadata(app core.App, ctx context.Context, record *core.Record, data map[string]any) {
	var federatedCategoryName, federatedSubcategoryName string

	if expand, ok := data["expand"].(map[string]any); ok {
		if cat, ok := expand["category"].(map[string]any); ok {
			if name, ok := cat["name"].(string); ok {
				federatedCategoryName = name
			}
		}
		if subcat, ok := expand["subcategory"].(map[string]any); ok {
			if name, ok := subcat["name"].(string); ok {
				federatedSubcategoryName = name
			}
		}
	}

	if federatedCategoryName != "" {
		record.Set("federated_category_name", federatedCategoryName)
	}
	if federatedSubcategoryName != "" {
		record.Set("federated_subcategory_name", federatedSubcategoryName)
	}

	category, subcategory, err := util.ResolveCategoryAndSubcategoryByNormalizedNames(app, federatedCategoryName, federatedSubcategoryName)
	if err == nil && category != nil {
		record.Set("category", category.Id)
		if subcategory != nil {
			record.Set("subcategory", subcategory.Id)
		} else {
			record.Set("subcategory", "")
		}
	} else if err == nil && federatedCategoryName != "" {
		record.Set("category", "")
		record.Set("subcategory", "")
	}

	// Resolve Tags
	localTagIds := resolveAndSyncTags(ctx, data)
	localTagIds = preservePreparedRemoteTags(ctx, record, data, localTagIds)
	if len(localTagIds) > 0 {
		record.Set("tags", localTagIds)
	}

	loadRemoteRecordFields(record, data, "category", "subcategory", "tags", "federated_category_name", "federated_subcategory_name")
}

func resolveAndSyncTags(ctx context.Context, data map[string]any) []string {
	localTagIds := []string{}

	expand, ok := data["expand"].(map[string]any)
	if !ok {
		return localTagIds
	}

	remoteTags, ok := expand["tags"].([]any)
	if !ok {
		return localTagIds
	}

	seenTags := make(map[string]struct{})

	for _, t := range remoteTags {
		tagMap, ok := t.(map[string]any)
		if !ok {
			continue
		}

		tagName, ok := tagMap["name"].(string)
		if !ok {
			continue
		}
		tagID := remoteSyncTagID(ctx, tagName)
		if tagID == "" {
			continue
		}
		if _, seen := seenTags[tagID]; seen {
			continue
		}
		seenTags[tagID] = struct{}{}
		localTagIds = append(localTagIds, tagID)
	}

	return localTagIds
}

func syncWaypoints(txApp core.App, ctx context.Context, trail *core.Record, origin string, waypoints []any) error {
	col, _ := txApp.FindCollectionByNameOrId("waypoints")

	for _, wData := range waypoints {
		raw, ok := wData.(map[string]any)
		if !ok {
			continue
		}
		wpID, _ := raw["id"].(string)
		iri, _ := raw["iri"].(string)
		if iri == "" {
			iri = fmt.Sprintf("%s/api/v1/waypoint/%s", origin, wpID)
		}
		if util.IsLocalIRI(iri) || !remoteSyncIRIMatchesOrigin(iri, origin) {
			continue
		}

		wp, _ := txApp.FindFirstRecordByData("waypoints", "iri", iri)
		if wp == nil {
			wp = core.NewRecord(col)
		} else if !remoteSyncRecordCanUpdate(txApp, wp, trail.GetString("author"), trail.Id) {
			continue
		}

		syncRecordFiles(ctx, wp, "waypoints", wpID, origin, raw)

		loadRemoteRecordFields(wp, raw)
		wp.Set("author", trail.GetString("author"))
		wp.Set("trail", trail.Id)
		wp.Set("iri", iri)

		util.SanitizeHTMLFieldsWithLimits(wp)
		if _, err := saveRemoteSyncChild(txApp, wp); err != nil {
			return err
		}
	}
	return nil
}

func syncSummitLogs(txApp core.App, ctx context.Context, trail *core.Record, origin string, summitLogs []any, canonical ...map[string]canonicalRemoteSummitLog) error {
	col, _ := txApp.FindCollectionByNameOrId("summit_logs")

	for _, slData := range summitLogs {
		raw, ok := slData.(map[string]any)
		if !ok {
			continue
		}
		slID, _ := raw["id"].(string)
		iri, _ := raw["iri"].(string)
		if iri == "" {
			iri = fmt.Sprintf("%s/api/v1/summit-log/%s", origin, slID)
		}
		if util.IsLocalIRI(iri) {
			continue
		}
		childOrigin := origin
		var author string
		var canonicalEntry *canonicalRemoteSummitLog
		if !remoteSyncIRIMatchesOrigin(iri, origin) {
			if len(canonical) == 0 {
				continue
			}
			confirmed, ok := canonical[0][iri]
			if !ok {
				continue
			}
			if confirmed.removed {
				canonicalRemoteLogCache.invalidateAfterRollback(txApp, confirmed.cacheKey)
				record, _ := txApp.FindFirstRecordByData("summit_logs", "iri", iri)
				if record != nil && record.GetString("trail") == trail.Id && record.GetString("author") == confirmed.author && !util.RecordHasLocalIdentity(txApp, record) {
					if err := txApp.Delete(record); err != nil {
						return err
					}
				}
				continue
			}
			canonicalEntry = &confirmed
			raw, slID, author = confirmed.data, confirmed.remoteID, confirmed.author
			childOrigin = confirmed.origin
		} else {
			resolved, allowed, err := resolveRemoteSyncAuthor(txApp, ctx, trail.GetString("author"), iri, raw)
			if err != nil {
				return err
			}
			if !allowed {
				continue
			}
			author = resolved
		}

		sl, _ := txApp.FindFirstRecordByData("summit_logs", "iri", iri)
		if sl == nil {
			sl = core.NewRecord(col)
		} else if !remoteSyncRecordCanUpdate(txApp, sl, author, trail.Id) {
			continue
		}

		syncRecordFiles(ctx, sl, "summit_logs", slID, childOrigin, raw)

		loadRemoteRecordFields(sl, raw)
		sl.Set("author", author)
		sl.Set("trail", trail.Id)
		sl.Set("iri", iri)

		util.SanitizeHTMLFieldsWithLimits(sl)
		saved, err := saveRemoteSyncChild(txApp, sl)
		if err != nil {
			return err
		}
		if !saved {
			continue
		}
		if canonicalEntry != nil {
			canonicalRemoteLogCache.successAfterCommit(txApp, canonicalEntry.cacheKey, canonicalEntry.updatedHint, sl.GetString("updated"))
		}
	}
	return nil
}

// Only an object's canonical origin may supply its mutable fields. Copies
// returned by another instance are references, including echoes of our own
// objects. Requiring a full HTTP origin also rejects relative or malformed
// identities from the payload.
func remoteSyncIRIMatchesOrigin(iri, origin string) bool {
	objectURL, err := url.Parse(iri)
	if err != nil {
		return false
	}
	originURL, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return (objectURL.Scheme == "https" || objectURL.Scheme == "http") &&
		(originURL.Scheme == "https" || originURL.Scheme == "http") &&
		objectURL.User == nil && originURL.User == nil && objectURL.Host != "" &&
		strings.EqualFold(objectURL.Host, originURL.Host)
}

// A sync may refresh an existing remote copy, but cannot take ownership of
// another author's record or move an entry from a different trail.
func remoteSyncRecordCanUpdate(app core.App, record *core.Record, author, trail string) bool {
	if util.IsLocalIRI(record.GetString("iri")) {
		return false
	}
	if actor, err := app.FindRecordById("activitypub_actors", record.GetString("author")); err == nil && actor.GetBool("is_local") {
		return false
	}
	if trail != "" && record.GetString("trail") != trail {
		return false
	}
	if record.GetString("author") == author {
		return true
	}
	// Older syncs assigned contributor entries to the trail author. Repair
	// only that specific remote fallback; arbitrary ownership changes stay
	// forbidden, and the corrected author must own the object's origin.
	if record.Collection().Name != "summit_logs" || trail == "" {
		return false
	}
	parent, err := app.FindRecordById("trails", trail)
	if err != nil || parent.GetString("author") != record.GetString("author") {
		return false
	}
	corrected, err := app.FindRecordById("activitypub_actors", author)
	return err == nil && !corrected.GetBool("is_local") && remoteSyncIRIMatchesOrigin(corrected.GetString("iri"), record.GetString("iri"))
}

// Remote payloads cannot assign content from another instance to a local
// actor. A local user's own entry may be echoed by a remote trail, but its
// canonical IRI must still belong to this instance.
func resolveRemoteSyncAuthor(app core.App, ctx context.Context, fallback, iri string, raw map[string]any) (string, bool, error) {
	author := fallback
	if expand, ok := raw["expand"].(map[string]any); ok {
		if authorMap, ok := expand["author"].(map[string]any); ok {
			authorIRI, _ := authorMap["iri"].(string)
			if util.IsLocalIRI(authorIRI) && !util.IsLocalIRI(iri) {
				return "", false, nil
			}
			var actor *core.Record
			if prepared, ok := ctx.Value(preparedRemoteChildAuthorsKey{}).(map[string]*core.Record); ok {
				actor = prepared[authorIRI]
				if actor == nil {
					return "", false, nil
				}
			} else {
				var err error
				actor, err = resolveRemoteActorForSync(app, ctx, authorIRI)
				if err != nil {
					return "", false, fmt.Errorf("%w: %w", errRemoteUnavailable, err)
				}
			}
			if actor.GetBool("is_local") && !util.IsLocalIRI(iri) {
				return "", false, nil
			}
			author = actor.Id
		}
	}
	actor, err := app.FindRecordById("activitypub_actors", author)
	if err != nil {
		return "", false, fmt.Errorf("%w: %w", errRemoteUnavailable, err)
	}
	if actor.GetBool("is_local") && !util.IsLocalIRI(iri) {
		return "", false, nil
	}
	if !actor.GetBool("is_local") && !remoteSyncIRIMatchesOrigin(actor.GetString("iri"), iri) {
		return "", false, nil
	}
	return author, true, nil
}

func syncRecordFiles(ctx context.Context, record *core.Record, collection, remoteID, origin string, data map[string]any) {
	// Handle GPX
	if gpx, ok := data["gpx"].(string); ok && record.GetString("gpx") == "" {
		if f, err := downloadFile(ctx, origin, collection, remoteID, gpx, remoteRecordFieldLimit(record, "gpx", util.RemoteGPXMaxBytes)); err == nil {
			record.Set("gpx", f)
		}
	}

	// Keep saved files and append only remote names that have not been copied.
	if photos, ok := data["photos"].([]any); ok {
		saved := record.GetStringSlice("photos")
		limit := 1
		if field, ok := record.Collection().Fields.GetByName("photos").(*core.FileField); ok {
			limit = max(1, field.MaxSelect)
		}
		for _, p := range photos {
			name, ok := p.(string)
			if !ok || name == "" || strings.ContainsAny(name, "/\\") || remotePhotoAlreadyStored(saved, name) {
				continue
			}
			if len(saved) >= limit {
				break
			}
			if f, err := downloadFile(ctx, origin, collection, remoteID, name, remoteRecordFieldLimit(record, "photos", util.RemotePhotoMaxBytes)); err == nil {
				record.Set("photos+", f)
				saved = append(saved, f.Name)
			}
		}
	}

}

func downloadFile(ctx context.Context, origin, col, id, name string, limits ...int64) (*filesystem.File, error) {
	maxBytes := util.RemoteGPXMaxBytes
	if len(limits) > 0 {
		maxBytes = limits[0]
	}
	rawURL := fmt.Sprintf("%s/api/v1/files/%s/%s/%s", origin, col, id, name)
	return util.PreparedRemoteSyncFile(ctx, rawURL, origin, maxBytes)
}

// Stable file names let a later sync recognize completed downloads.
func remoteSyncFileName(name string) string {
	sum := sha256.Sum256([]byte(name))
	ext := path.Ext(name)
	if len(ext) > 20 || strings.IndexFunc(ext, func(r rune) bool {
		return !(r == '.' || r == '_' || r == '-' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9')
	}) >= 0 {
		ext = ""
	}
	return "remote_" + hex.EncodeToString(sum[:]) + ext
}

var remoteCopyNameSuffix = regexp.MustCompile(`^[a-z0-9]{10}$`)
var remoteCopyExtInvalid = regexp.MustCompile(`[^A-Za-z0-9._-]+`)
var remoteCopyDetectedExt = regexp.MustCompile(`^\.[a-z0-9]{1,5}$`)

func remotePhotoAlreadyStored(saved []string, source string) bool {
	stable := remoteSyncFileName(source)
	// PocketBase keeps the last two extension components when adding its suffix.
	ext := ""
	if dot := strings.LastIndex(source, "."); dot >= 0 {
		if previous := strings.LastIndex(source[:dot], "."); previous >= 0 {
			dot = previous
		}
		ext = source[dot:]
	}
	base := strings.TrimSuffix(source, ext)
	if len(base) > 100 {
		base = base[:100]
	}
	exts := []string{ext}
	// attachmentFileName normalizes punctuation before PocketBase trims dots.
	normalized := remoteCopyExtInvalid.ReplaceAllString(ext, "_")
	for _, candidate := range []string{normalized, "." + strings.Trim(ext, "."), "." + strings.Trim(normalized, ".")} {
		if candidate != "." && !slices.Contains(exts, candidate) {
			exts = append(exts, candidate)
		}
	}
	for _, name := range saved {
		if name == stable || name == source {
			return true
		}
		// Earlier copies used PocketBase's extra ten-character suffix.
		for _, ext := range exts {
			if strings.HasPrefix(name, base+"_") && strings.HasSuffix(name, ext) && len(name) == len(base)+11+len(ext) && remoteCopyNameSuffix.MatchString(name[len(base)+1:len(base)+11]) {
				return true
			}
		}
		// An extension of only dots is trimmed before PocketBase detects one
		// from the contents. Its base and generated suffix still identify it.
		if ext != "" && strings.Trim(ext, ".") == "" && strings.HasPrefix(name, base+"_") && len(name) > len(base)+11 && remoteCopyNameSuffix.MatchString(name[len(base)+1:len(base)+11]) && remoteCopyDetectedExt.MatchString(name[len(base)+11:]) {
			return true
		}
	}
	return false
}

func remoteRecordFieldLimit(record *core.Record, name string, fallback int64) int64 {
	if field, ok := record.Collection().Fields.GetByName(name).(*core.FileField); ok && field.MaxSize > 0 {
		return field.MaxSize
	}
	return fallback
}
