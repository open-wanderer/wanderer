package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"pocketbase/federation"
	"strings"

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

	// 1. Resolve the "Actual" Record or Shell
	if handle != "" {
		// If we have a handle, we are looking for a remote trail.
		// Construct the IRI first to see if we already know this trail.
		record, err = findLocalByRemoteInfo(e, userActor, handle, trailID)
		if err != nil {
			return e.InternalServerError("Failed to resolve trail", err)
		}

		// If the record has no ID, it's a new Shell
		if record.Id == "" {
			// Blocking sync for new records
			record, err = performFullSync(e.App, userActor, e.Request.URL, record)
			if err != nil {
				return e.InternalServerError("Sync failed", err)
			}
		} else {
			// We already have it locally. Show and update background.
			// updatedAt := record.GetDateTime("updated").Time()
			// if time.Since(updatedAt) > 60*time.Minute {
			go performFullSync(e.App, userActor, e.Request.URL, record)
			// }
		}
	} else {
		// Standard local fetch by ID
		record, err = e.App.FindRecordById("trails", trailID)
		if err != nil {
			return e.NotFoundError("Trail not found", nil)
		}
	}

	return expandAndReturn(e, record, expandQuery)
}

func findLocalByRemoteInfo(e *core.RequestEvent, userActor *core.Record, handle, trailID string) (*core.Record, error) {
	// 1. Get Actor to build the IRI
	actor, err := federation.GetActorByHandle(e.App, userActor, handle, false)
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

func performFullSync(app core.App, userActor *core.Record, reqURL *url.URL, localTrail *core.Record) (*core.Record, error) {
	iri := localTrail.GetString("iri")
	remoteUrl, _ := url.Parse(iri)
	remoteUrl.RawQuery = reqURL.RawQuery // Forward params
	origin := fmt.Sprintf("%s://%s", remoteUrl.Scheme, remoteUrl.Host)

	res, err := http.Get(remoteUrl.String())
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
		syncRecordFiles(localTrail, "trails", remoteID, origin, remoteMap)

		// 2. Map Relations & Simple Fields
		syncMetadata(txApp, localTrail, remoteMap)

		if err := txApp.Save(localTrail); err != nil {
			return err
		}

		// 3. Sync Waypoints
		if expand, ok := remoteMap["expand"].(map[string]any); ok {
			if wps, ok := expand["waypoints_via_trail"].([]any); ok {
				err = syncWaypoints(txApp, localTrail, origin, wps)
				if err != nil {
					return err
				}
			}
		}

		// 3. Sync SummitLogs
		if expand, ok := remoteMap["expand"].(map[string]any); ok {
			if sls, ok := expand["summit_logs_via_trail"].([]any); ok {
				err = syncSummitLogs(txApp, userActor, localTrail, origin, sls)
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

func syncMetadata(app core.App, record *core.Record, data map[string]any) {
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

func syncWaypoints(txApp core.App, trail *core.Record, origin string, waypoints []any) error {
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

		syncRecordFiles(wp, "waypoints", wpID, origin, raw)

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

func syncSummitLogs(txApp core.App, userActor *core.Record, trail *core.Record, origin string, summitLogs []any) error {
	col, _ := txApp.FindCollectionByNameOrId("summit_logs")

	for _, slData := range summitLogs {
		raw := slData.(map[string]any)
		slID, _ := raw["id"].(string)
		iri, _ := raw["iri"].(string)
		if iri == "" {
			iri = fmt.Sprintf("%s/api/v1/summit_logs/%s", origin, slID)
		}

		sl, _ := txApp.FindFirstRecordByData("summit_logs", "iri", iri)
		if sl == nil {
			sl = core.NewRecord(col)
		}

		author := trail.GetString("author")
		if expand, ok := raw["expand"].(map[string]any); ok {
			if authorMap, ok := expand["author"].(map[string]any); ok {
				actor, err := federation.GetActorByIRI(txApp, userActor, authorMap["iri"].(string), false)
				if err != nil {
					return err
				}
				author = actor.Id
			}
		}

		syncRecordFiles(sl, "summit_logs", slID, origin, raw)

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

func syncRecordFiles(record *core.Record, collection, remoteID, origin string, data map[string]any) {
	// Handle GPX
	if gpx, ok := data["gpx"].(string); ok && record.GetString("gpx") == "" {
		if f, err := downloadFile(origin, collection, remoteID, gpx); err == nil {
			record.Set("gpx", f)
		}
	}

	// Handle Photos
	if photos, ok := data["photos"].([]any); ok && len(record.GetStringSlice("photos")) == 0 {
		var files []*filesystem.File
		for _, p := range photos {
			if f, err := downloadFile(origin, collection, remoteID, p.(string)); err == nil {
				files = append(files, f)
			}
		}
		if len(files) > 0 {
			record.Set("photos", files)
		}
	}
}

func downloadFile(origin, col, id, name string) (*filesystem.File, error) {
	url := fmt.Sprintf("%s/api/v1/files/%s/%s/%s", origin, col, id, name)
	res, err := http.Get(url)
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
