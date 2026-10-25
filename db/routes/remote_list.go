package routes

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"pocketbase/federation"
	"pocketbase/util"
	"slices"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// --- Main Handler ---

func RemoteListGet(e *core.RequestEvent) error {
	handle := e.Request.URL.Query().Get("handle")
	listID := e.Request.PathValue("id")

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

	if handle != "" {
		record, err = findLocalListByRemoteInfo(e, ctx, handle, listID)
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
			return e.InternalServerError("Failed to resolve list", err)
		}

		if record.Id == "" || record.GetBool("needs_full_sync") && (!hasCompletedFullSync(record) || util.IsLocalIRI(record.GetString("iri")) || remoteRecordHasLocalAuthor(e.App, record)) {
			cachedID := record.Id
			record, err = performFullListSync(e.App, ctx, e.Request.URL, record)
			if err != nil {
				if errors.Is(err, errRemoteGone) {
					dropGoneRemoteRecord(e.App, "lists", cachedID)
					return e.NotFoundError("List not found", err)
				}
				cached := cachedRecordFallback(e.App, "lists", cachedID, err)
				if cached == nil {
					if isRemoteDocumentLimitError(err) {
						return remoteDocumentLimitError(e.App, record, "lists")
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
					"serving cached list after failed remote sync",
					"iri", cached.GetString("iri"),
					"error", err,
				)
				record = cached
			}
			if record.Id == "" {
				// Local content that does not exist: performFullListSync
				// short-circuits local IRIs and returns the unsaved shell —
				// surface a real 404 instead of access/expand on a missing record.
				return e.NotFoundError("List not found", nil)
			}
		} else {
			updatedAt := record.GetDateTime("updated").Time()

			iri := record.GetString("iri")
			if record.GetBool("needs_full_sync") || time.Now().UTC().Sub(updatedAt) > remoteSyncThreshold || remoteListReferencesDue(e.App, iri) {
				if _, alreadySyncing := listSyncing.LoadOrStore(iri, struct{}{}); !alreadySyncing {
					urlCopy := *e.Request.URL
					bgCtx := context.WithValue(context.Background(), "actor", ctx.Value("actor"))
					// Sync hooks and remote data must not mutate the response record.
					syncRecord := record.Fresh()
					go func() {
						defer listSyncing.Delete(iri)
						_, err := performFullListSync(e.App, bgCtx, &urlCopy, syncRecord)
						switch {
						case errors.Is(err, errRemoteGone):
							dropGoneRemoteRecord(e.App, "lists", syncRecord.Id)
						case err != nil:
							backgroundSyncFailed(e.App, "lists", iri, err)
						}
					}()
				}
			}
		}
	} else {
		record, err = e.App.FindRecordById("lists", listID)
		if err != nil {
			return e.NotFoundError("List not found", nil)
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

func findLocalListByRemoteInfo(e *core.RequestEvent, ctx context.Context, handle, trailID string) (*core.Record, error) {
	// 1. Get Actor to build the IRI
	// A private profile is not a resolution failure: the list's own ViewRule
	// still decides whether it may be seen, exactly as findLocalTrailByRemoteInfo
	// does for trails (#986). Bailing out here 500'd every /lists/@handle/id URL
	// whose owner had set account privacy to "private", including their public
	// lists.
	actor, err := resolveRemoteHandleActor(e, ctx, handle)
	if actor == nil {
		// No actor available at all — no cache, and the live lookup itself
		// failed (e.g. remote instance unreachable, unknown handle). We can't
		// construct the IRI, but a previously-synced list may still exist
		// locally under this exact ID, so fall back to a plain local lookup
		// instead of failing the whole request. Skip this fallback for a
		// private-profile error so the caller's 404 handling still applies.
		if err != nil && !errors.Is(err, federation.ErrProfilePrivate) {
			if local, localErr := e.App.FindRecordById("lists", trailID); localErr == nil {
				return local, nil
			}
		}
		return nil, err
	}

	// Actor is non-nil even though err may be set: federation.GetActorByHandle
	// (via assembleActor) returns a previously-cached actor record alongside a
	// transport error when a background refresh of stale actor data fails.
	// That cached actor's IRI is still usable to look up the local list, so
	// use it instead of discarding it and 500ing the whole request.
	actorURL, _ := url.Parse(actor.GetString("iri"))
	iri := fmt.Sprintf("%s://%s/api/v1/list/%s", actorURL.Scheme, actorURL.Host, trailID)

	// 2. Check if this IRI already exists in our DB
	existing, _ := e.App.FindFirstRecordByFilter("lists", "iri={:iri}", dbx.Params{"iri": iri})
	if existing == nil {
		candidate, _ := e.App.FindRecordById("lists", trailID)
		if candidate != nil && remoteHandleSelectsRecord(e.App, candidate, actor, iri) {
			existing = candidate
		}
	}
	if existing != nil {
		return existing, nil
	}

	// No cached copy of this list exists locally. If the actor fetch failed
	// for a real (non-private) reason, there's nothing usable to fall back to
	// — surface the original error rather than fabricating a shell that a
	// subsequent sync attempt would just fail on again.
	if err != nil && !errors.Is(err, federation.ErrProfilePrivate) {
		return nil, err
	}

	// 3. Not found? Return a new Shell
	collection, _ := e.App.FindCollectionByNameOrId("lists")
	shell := core.NewRecord(collection)
	shell.Set("iri", iri)
	shell.Set("author", actor.Id)

	return shell, nil
}

func performFullListSync(app core.App, ctx context.Context, reqURL *url.URL, localList *core.Record) (*core.Record, error) {
	return limitedRemoteSync(app, ctx, "lists", localList, func(ctx context.Context) (*core.Record, error) {
		return performFullListSyncDocument(app, ctx, reqURL, localList)
	})
}

func performFullListSyncDocument(app core.App, ctx context.Context, reqURL *url.URL, localList *core.Record) (*core.Record, error) {
	ctx, cancel := context.WithTimeout(ctx, remoteSyncBudget)
	defer cancel()
	iri := localList.GetString("iri")

	// Never federate with ourselves (see performFullSync in remote_trail.go).
	if iri == "" || util.IsLocalIRI(iri) || remoteRecordHasLocalAuthor(app, localList) {
		if localList.GetBool("needs_full_sync") {
			localList.Set("needs_full_sync", false)
			if err := app.Save(localList); err != nil {
				return localList, err
			}
		}
		return localList, nil
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

	document, err := fetchRemoteSyncDocument(ctx, client, remoteUrl, "list")
	if err != nil {
		return localList, err
	}
	defer document.close()
	identity, err := document.identity(app, ctx, localList)
	if err != nil {
		return localList, err
	}
	previousTrails := append([]string(nil), localList.GetStringSlice("trails")...)
	author, err := resolveRemoteTopLevelAuthor(app, ctx, localList, identity)
	if err != nil {
		return localList, fmt.Errorf("%w: %w", errRemoteUnavailable, err)
	}
	plan, err := document.networkPlan(app, ctx, localList, origin, "list")
	if err != nil {
		return localList, err
	}
	remoteMap := plan.data

	ctx = prepareRemoteChildAuthors(app, ctx, remoteMap, origin, "trails", "trail")
	ctx = prepareRemoteListVisibility(app, ctx, remoteMap, origin)
	ctx = prepareRemoteListReferences(app, ctx, localList, remoteMap, origin)

	ctx, cleanupMedia, err := prepareRemoteMedia(ctx, plan.media.requests)
	if err != nil {
		return localList, err
	}
	defer cleanupMedia()
	return document.process(ctx, func(remoteMap map[string]any) (*core.Record, error) {
		remoteID, _ := remoteMap["id"].(string)
		expand, _ := remoteMap["expand"].(map[string]any)
		trails, _ := expand["trails"].([]any)
		namedPrevious := previousListTrailsInDocument(app, previousTrails, trails, origin)
		childrenCapped := capRemoteSyncChildren(app, remoteMap, origin, "trails")
		ctx, err = prepareRemoteSyncTags(app, ctx, remoteMap, author, origin, "list")
		if err != nil {
			return localList, err
		}

		localList.Set("author", author)
		syncListRecordFiles(ctx, localList, "lists", remoteID, origin, remoteMap)
		syncListMetadata(localList, remoteMap)
		localList.Set("author", author)
		if expand, ok := remoteMap["expand"].(map[string]any); ok {
			if trails, ok := expand["trails"].([]any); ok {
				ids := []string{}
				processed := 0
				err = remoteSyncChildBatches(app, ctx, trails, func(tx core.App, batch []any) error {
					working := localList.Fresh()
					working.Set("trails", previousTrails)
					if err := syncTrails(tx, ctx, working, origin, batch); err != nil {
						return err
					}
					ids = append(ids, working.GetStringSlice("trails")...)
					processed += len(batch)
					return nil
				})
				if err != nil && !interruptedRemoteChildren(err) {
					return localList, err
				}
				if childrenCapped || err != nil {
					ids = retainUnprocessedListTrails(app, localList.GetString("author"), ids, previousTrails, trails[:processed], namedPrevious, origin)
				}
				localList.Set("trails", ids)
			}
		}
		util.SanitizeHTMLFieldsWithLimits(localList)
		return finishRemoteSyncChildren(app, ctx, localList, err)
	})
}

func syncListMetadata(record *core.Record, data map[string]any) {
	loadRemoteRecordFields(record, data, "trails")
}

func syncListRecordFiles(ctx context.Context, record *core.Record, collection, remoteID, origin string, data map[string]any) {
	if gpx, ok := data["avatar"].(string); ok && record.GetString("avatar") == "" {
		if f, err := downloadFile(ctx, origin, collection, remoteID, gpx, remoteRecordFieldLimit(record, "avatar", util.RemoteAvatarMaxBytes)); err == nil {
			record.Set("avatar", f)
		}
	}
}

func syncTrails(txApp core.App, ctx context.Context, list *core.Record, origin string, trails []any) error {
	col, _ := txApp.FindCollectionByNameOrId("trails")

	localTrails := make([]string, 0, len(trails))

	for _, tData := range trails {
		raw, ok := tData.(map[string]any)
		if !ok {
			continue
		}
		tID, _ := raw["id"].(string)
		iri, _ := raw["iri"].(string)
		if iri == "" {
			iri = fmt.Sprintf("%s/api/v1/trail/%s", origin, tID)
		}
		trail, _ := txApp.FindFirstRecordByData("trails", "iri", iri)
		fromOrigin := remoteSyncIRIMatchesOrigin(iri, origin)
		if util.IsLocalIRI(iri) || !fromOrigin || (trail != nil && remoteRecordHasLocalAuthor(txApp, trail)) {
			if !fromOrigin && !util.IsLocalIRI(iri) {
				// Another origin's trail exists here only as that origin
				// describes it, under the IRI it uses.
				var err error
				if trail, err = remoteListReferenceRecord(txApp, ctx, col, iri, trail); err != nil {
					return err
				}
			}
			if trail != nil {
				if err := applyRemoteListVisibility(txApp, ctx, trail); err != nil {
					return err
				}
				if remoteListAuthorCanViewLocalTrail(txApp, list.GetString("author"), trail) {
					localTrails = append(localTrails, trail.Id)
				}
			}
			continue
		}
		author, allowed, err := resolveRemoteSyncAuthor(txApp, ctx, list.GetString("author"), iri, raw)
		if err != nil {
			return err
		}
		if !allowed {
			// A failed author refresh does not revoke an existing reference.
			// Keep its stored content until that author can be confirmed again.
			if trail != nil && slices.Contains(list.GetStringSlice("trails"), trail.Id) {
				localTrails = append(localTrails, trail.Id)
			}
			continue
		}

		existingID := ""
		if trail != nil {
			existingID = trail.Id
		}
		if trail == nil {
			trail = core.NewRecord(col)
			trail.Set("needs_full_sync", true)
		} else if !remoteSyncRecordCanUpdate(txApp, trail, author, "") {
			localTrails = append(localTrails, trail.Id)
			continue
		}

		if fromOrigin {
			syncTrailMetadata(txApp, ctx, trail, raw)
		}

		trail.Set("author", author)
		trail.Set("iri", iri)

		util.SanitizeHTMLFieldsWithLimits(trail)
		saved, err := saveRemoteSyncChild(txApp, trail)
		if err != nil {
			return err
		}
		if !saved {
			if existingID != "" {
				localTrails = append(localTrails, existingID)
			}
			continue
		}

		localTrails = append(localTrails, trail.Id)

	}

	list.Set("trails", localTrails)
	return nil
}

// Store or refresh a placeholder only from the trail's own origin and
// author. Opening it runs the full sync at that origin.
func remoteListReferenceRecord(txApp core.App, ctx context.Context, col *core.Collection, iri string, trail *core.Record) (*core.Record, error) {
	state := remoteListReferenceStateOf(ctx)
	if trail != nil {
		return trail, refreshRemoteListReference(txApp, ctx, state, trail)
	}
	reference, confirmed := state.confirmed[iri]
	if !confirmed || remoteListReferenceIsLocal(reference.iri) {
		return nil, nil
	}
	if stored, _ := txApp.FindFirstRecordByData("trails", "iri", reference.iri); stored != nil {
		return stored, nil
	}
	actor, err := txApp.FindRecordById("activitypub_actors", reference.author)
	if err != nil || actor.GetBool("is_local") || !remoteSyncIRIMatchesOrigin(actor.GetString("iri"), reference.iri) {
		return nil, nil
	}
	trail = core.NewRecord(col)
	trail.Set("needs_full_sync", true)
	syncTrailMetadata(txApp, ctx, trail, reference.data)
	trail.Set("author", actor.Id)
	trail.Set("iri", reference.iri)
	util.SanitizeHTMLFieldsWithLimits(trail)
	saved, err := saveRemoteSyncChild(txApp, trail)
	if err != nil || !saved {
		if err == nil {
			// An answer that cannot be stored is not requested again soon.
			rememberRemoteListReferenceFailure(txApp, reference.iri)
		}
		return nil, err
	}
	return trail, nil
}

// A placeholder that was never opened follows its origin's answers.
func refreshRemoteListReference(txApp core.App, ctx context.Context, state remoteListReferenceState, trail *core.Record) error {
	reference, found := state.refreshed[trail.Id]
	if !found || reference.author != trail.GetString("author") || trail.GetString("iri") != reference.iri ||
		!trail.GetBool("public") || !trail.GetBool("needs_full_sync") || hasCompletedFullSync(trail) ||
		remoteRecordHasLocalAuthor(txApp, trail) || util.RecordHasLocalIdentity(txApp, trail) {
		return nil
	}
	syncTrailMetadata(txApp, ctx, trail, reference.data)
	util.SanitizeHTMLFieldsWithLimits(trail)
	saved, err := saveRemoteSyncChild(txApp, trail)
	if !saved && err == nil {
		rememberRemoteListReferenceFailure(txApp, reference.iri)
	}
	return err
}

// A public remote list is indexed with statistics from all its linked trails.
// Merely naming a local private IRI must not expose those statistics.
func remoteListAuthorCanViewLocalTrail(app core.App, author string, trail *core.Record) bool {
	if trail.GetBool("public") || (author != "" && trail.GetString("author") == author) {
		return true
	}
	count, err := app.CountRecords("trail_share", dbx.HashExp{"trail": trail.Id, "actor": author})
	return err == nil && count > 0
}
