package routes

import (
	"encoding/json"
	"fmt"
	"net/url"
	"pocketbase/federation"
	"pocketbase/util"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// --- Main Handler ---

func RemoteListGet(e *core.RequestEvent) error {
	handle := e.Request.URL.Query().Get("handle")
	listID := e.Request.PathValue("id")
	expandQuery := e.Request.URL.Query().Get("expand")

	var record *core.Record
	var err error

	var userActor *core.Record
	if e.Auth != nil {
		userActor, _ = e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
	}

	if handle != "" {
		record, err = findLocalListByRemoteInfo(e, userActor, handle, listID)
		if err != nil {
			return e.InternalServerError("Failed to resolve trail", err)
		}

		if record.Id == "" {
			record, err = performFullListSync(e.App, userActor, e.Request.URL, record)
			if err != nil {
				return e.InternalServerError("Sync failed", err)
			}
		} else {
			updatedAt := record.GetDateTime("updated").Time()
			if time.Now().UTC().Sub(updatedAt) > 60*time.Minute {
				go performFullSync(e.App, userActor, e.Request.URL, record)
			}
		}
	} else {
		record, err = e.App.FindRecordById("lists", listID)
		if err != nil {
			return e.NotFoundError("List not found", nil)
		}
	}

	return expandAndReturn(e, record, expandQuery)
}

func findLocalListByRemoteInfo(e *core.RequestEvent, userActor *core.Record, handle, trailID string) (*core.Record, error) {
	// 1. Get Actor to build the IRI
	actor, err := federation.GetActorByHandle(e.App, userActor, handle, false)
	if err != nil {
		return nil, err
	}

	actorURL, _ := url.Parse(actor.GetString("iri"))
	iri := fmt.Sprintf("%s://%s/api/v1/list/%s", actorURL.Scheme, actorURL.Host, trailID)

	// 2. Check if this IRI already exists in our DB
	existing, _ := e.App.FindFirstRecordByFilter("lists", "iri={:iri}||id={:id}", dbx.Params{"id": trailID, "iri": iri})
	if existing != nil {
		return existing, nil
	}

	// 3. Not found? Return a new Shell
	collection, _ := e.App.FindCollectionByNameOrId("lists")
	shell := core.NewRecord(collection)
	shell.Set("iri", iri)
	shell.Set("author", actor.Id)

	return shell, nil
}

func performFullListSync(app core.App, userActor *core.Record, reqURL *url.URL, localList *core.Record) (*core.Record, error) {
	client := util.SafeHTTPClient()

	iri := localList.GetString("iri")
	remoteUrl, _ := url.Parse(iri)
	remoteUrl.RawQuery = reqURL.RawQuery
	origin := fmt.Sprintf("%s://%s", remoteUrl.Scheme, remoteUrl.Host)

	res, err := client.Get(remoteUrl.String())
	if err != nil || res.StatusCode != 200 {
		return localList, err
	}
	defer res.Body.Close()

	var remoteMap map[string]any
	if err := json.NewDecoder(res.Body).Decode(&remoteMap); err != nil {
		return localList, err
	}

	err = app.RunInTransaction(func(txApp core.App) error {
		remoteID, _ := remoteMap["id"].(string)

		// 1. Sync Files
		syncListRecordFiles(localList, "lists", remoteID, origin, remoteMap)

		// 2. Map Relations & Simple Fields
		syncListMetadata(localList, remoteMap)

		// 3. Sync Trails
		if expand, ok := remoteMap["expand"].(map[string]any); ok {
			if trails, ok := expand["trails"].([]any); ok {
				err = syncTrails(txApp, userActor, localList, origin, trails)
				if err != nil {
					return err
				}
			}
		}

		if err := txApp.Save(localList); err != nil {
			return err
		}

		return nil
	})

	return localList, err
}

func syncListMetadata(record *core.Record, data map[string]any) {
	delete(data, "id")
	delete(data, "avatar")
	delete(data, "author")
	delete(data, "iri")

	record.Load(data)
}

func syncListRecordFiles(record *core.Record, collection, remoteID, origin string, data map[string]any) {
	if gpx, ok := data["avatar"].(string); ok && record.GetString("avatar") == "" {
		if f, err := downloadFile(origin, collection, remoteID, gpx); err == nil {
			record.Set("avatar", f)
		}
	}
}

func syncTrails(txApp core.App, userActor *core.Record, list *core.Record, origin string, trails []any) error {
	col, _ := txApp.FindCollectionByNameOrId("trails")

	localTrails := make([]string, 0, len(trails))

	for _, tData := range trails {
		raw := tData.(map[string]any)
		tID, _ := raw["id"].(string)
		iri, _ := raw["iri"].(string)
		if iri == "" {
			iri = fmt.Sprintf("%s/api/v1/trail/%s", origin, tID)
		}

		trail, _ := txApp.FindFirstRecordByData("trails", "iri", iri)
		if trail == nil {
			trail = core.NewRecord(col)
			trail.Set("needs_full_sync", true)
		}

		syncTrailMetadata(txApp, trail, raw)

		author := list.GetString("author")
		if expand, ok := raw["expand"].(map[string]any); ok {
			if authorMap, ok := expand["author"].(map[string]any); ok {
				actor, err := federation.GetActorByIRI(txApp, userActor, authorMap["iri"].(string), false)
				if err != nil {
					return err
				}
				author = actor.Id
			}
		}

		trail.Set("author", author)
		trail.Set("iri", iri)

		if err := txApp.Save(trail); err != nil {
			return err
		}

		localTrails = append(localTrails, trail.Id)

	}

	list.Set("trails", localTrails)
	return nil
}
