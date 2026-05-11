package routes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"pocketbase/federation"
	"pocketbase/util"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

// --- Main Handler ---

func RemoteTrailGet(e *core.RequestEvent) error {
	handle := e.Request.URL.Query().Get("handle")
	trailID := e.Request.PathValue("id")
	expandQuery := e.Request.URL.Query().Get("expand")

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
			return e.InternalServerError("Failed to resolve trail", err)
		}

		// If the record has no ID, it's a new Shell
		if record.Id == "" || record.GetBool("needs_full_sync") {
			// Blocking sync for new records
			record, err = performFullSync(e.App, ctx, e.Request.URL, record)
			if err != nil {
				if errors.Is(err, util.ErrRateLimited) {
					return e.TooManyRequestsError("Too many requests", err)
				}
				return e.InternalServerError("Sync failed", err)
			}
		} else {
			// We already have it locally. Show and update background.
			updatedAt := record.GetDateTime("updated").Time()
			if time.Now().UTC().Sub(updatedAt) > 60*time.Minute {
				go performFullSync(e.App, ctx, e.Request.URL, record)
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

	return expandAndReturn(e, record, expandQuery)
}

func findLocalTrailByRemoteInfo(e *core.RequestEvent, ctx context.Context, handle, trailID string) (*core.Record, error) {
	// 1. Get Actor to build the IRI
	actor, err := federation.GetActorByHandle(e.App, ctx, handle, false)
	if err != nil {
		return nil, err
	}

	actorURL, _ := url.Parse(actor.GetString("iri"))
	iri := fmt.Sprintf("%s://%s/api/v1/trail/%s", actorURL.Scheme, actorURL.Host, trailID)

	// 2. Check if this IRI already exists in our DB
	existing, _ := e.App.FindFirstRecordByFilter("trails", "iri={:iri}||id={:id}", dbx.Params{"id": trailID, "iri": iri})
	if existing != nil {
		return existing, nil
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
	client := util.SafeHTTPClient()

	iri := localTrail.GetString("iri")
	remoteUrl, _ := url.Parse(iri)
	remoteUrl.RawQuery = reqURL.RawQuery // Forward params
	origin := fmt.Sprintf("%s://%s", remoteUrl.Scheme, remoteUrl.Host)

	req, _ := http.NewRequestWithContext(ctx, "GET", remoteUrl.String(), nil)
	res, err := client.Do(req)
	if err != nil || res.StatusCode != 200 {
		return localTrail, err
	}
	defer res.Body.Close()

	var remoteMap map[string]any
	if err := json.NewDecoder(res.Body).Decode(&remoteMap); err != nil {
		return localTrail, err
	}

	err = app.RunInTransaction(func(txApp core.App) error {
		remoteID, _ := remoteMap["id"].(string)

		// 1. Sync Files
		syncRecordFiles(ctx, localTrail, "trails", remoteID, origin, remoteMap)

		// 2. Map Relations & Simple Fields
		syncTrailMetadata(txApp, localTrail, remoteMap)

		localTrail.Set("needs_full_sync", false)

		if err := txApp.Save(localTrail); err != nil {
			return err
		}

		// 3. Sync Waypoints
		if expand, ok := remoteMap["expand"].(map[string]any); ok {
			if wps, ok := expand["waypoints_via_trail"].([]any); ok {
				err = syncWaypoints(txApp, ctx, localTrail, origin, wps)
				if err != nil {
					return err
				}
			}
		}

		// 3. Sync SummitLogs
		if expand, ok := remoteMap["expand"].(map[string]any); ok {
			if sls, ok := expand["summit_logs_via_trail"].([]any); ok {
				err = syncSummitLogs(txApp, ctx, localTrail, origin, sls)
				if err != nil {
					return err
				}
			}
		}

		return nil
	})

	return localTrail, err
}

// --- Sub-Sync Helpers ---

func syncTrailMetadata(app core.App, record *core.Record, data map[string]any) {
	// Resolve Category if present in expand
	if expand, ok := data["expand"].(map[string]any); ok {
		if cat, ok := expand["category"].(map[string]any); ok {
			if name, ok := cat["name"].(string); ok {
				if c, _ := app.FindFirstRecordByData("categories", "name", name); c != nil {
					record.Set("category", c.Id)
				}
			}
		}
	}

	// Clean protected/complex fields before bulk load
	delete(data, "id")
	delete(data, "photos")
	delete(data, "gpx")
	delete(data, "author")
	delete(data, "category")
	delete(data, "iri")

	record.Load(data)
}

func syncWaypoints(txApp core.App, ctx context.Context, trail *core.Record, origin string, waypoints []any) error {
	col, _ := txApp.FindCollectionByNameOrId("waypoints")

	for _, wData := range waypoints {
		raw := wData.(map[string]any)
		wpID, _ := raw["id"].(string)
		iri, _ := raw["iri"].(string)
		if iri == "" {
			iri = fmt.Sprintf("%s/api/v1/waypoint/%s", origin, wpID)
		}

		wp, _ := txApp.FindFirstRecordByData("waypoints", "iri", iri)
		if wp == nil {
			wp = core.NewRecord(col)
		}

		syncRecordFiles(ctx, wp, "waypoints", wpID, origin, raw)

		delete(raw, "id")
		delete(raw, "photos")
		wp.Load(raw)
		wp.Set("author", trail.GetString("author"))
		wp.Set("trail", trail.Id)
		wp.Set("iri", iri)

		if err := txApp.Save(wp); err != nil {
			return err
		}
	}
	return nil
}

func syncSummitLogs(txApp core.App, ctx context.Context, trail *core.Record, origin string, summitLogs []any) error {
	col, _ := txApp.FindCollectionByNameOrId("summit_logs")

	for _, slData := range summitLogs {
		raw := slData.(map[string]any)
		slID, _ := raw["id"].(string)
		iri, _ := raw["iri"].(string)
		if iri == "" {
			iri = fmt.Sprintf("%s/api/v1/summit_logs/%s", origin, slID)
		}

		remoteSummitLogUrl, _ := url.Parse(iri)
		possibleLocalId := path.Base(remoteSummitLogUrl.Path)

		sl, _ := txApp.FindFirstRecordByFilter("summit_logs", "iri={:iri} || id={:id}", dbx.Params{"id": possibleLocalId, "iri": iri})
		if sl == nil {
			sl = core.NewRecord(col)
		}

		author := trail.GetString("author")
		if expand, ok := raw["expand"].(map[string]any); ok {
			if authorMap, ok := expand["author"].(map[string]any); ok {
				actor, err := federation.GetActorByIRI(txApp, ctx, authorMap["iri"].(string), false)
				if err != nil {
					return err
				}
				author = actor.Id
			}
		}

		syncRecordFiles(ctx, sl, "summit_logs", slID, origin, raw)

		delete(raw, "id")
		delete(raw, "photos")
		delete(raw, "gpx")

		sl.Load(raw)
		sl.Set("author", author)
		sl.Set("trail", trail.Id)
		sl.Set("iri", iri)

		if err := txApp.Save(sl); err != nil {
			return err
		}
	}
	return nil
}

func syncRecordFiles(ctx context.Context, record *core.Record, collection, remoteID, origin string, data map[string]any) {
	// Handle GPX
	if gpx, ok := data["gpx"].(string); ok && record.GetString("gpx") == "" {
		if f, err := downloadFile(ctx, origin, collection, remoteID, gpx); err == nil {
			record.Set("gpx", f)
		}
	}

	// Handle Photos
	if photos, ok := data["photos"].([]any); ok && len(record.GetStringSlice("photos")) == 0 {
		var files []*filesystem.File
		for _, p := range photos {
			if f, err := downloadFile(ctx, origin, collection, remoteID, p.(string)); err == nil {
				files = append(files, f)
			}
		}
		if len(files) > 0 {
			record.Set("photos", files)
		}
	}
}

func downloadFile(ctx context.Context, origin, col, id, name string) (*filesystem.File, error) {
	client := util.SafeHTTPClient()

	url := fmt.Sprintf("%s/api/v1/files/%s/%s/%s", origin, col, id, name)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)

	res, err := client.Do(req)
	if err != nil || res.StatusCode != 200 {
		return nil, fmt.Errorf("download failed")
	}
	defer res.Body.Close()

	data, _ := io.ReadAll(res.Body)
	return filesystem.NewFileFromBytes(data, name)
}

func expandAndReturn(e *core.RequestEvent, record *core.Record, query string) error {
	if query != "" {
		e.App.ExpandRecord(record, strings.Split(query, ","), nil)
	}
	return e.JSON(http.StatusOK, record)
}
