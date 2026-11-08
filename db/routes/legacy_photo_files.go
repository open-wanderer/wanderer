package routes

import (
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"pocketbase/util"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// LegacyPhotoFileRedirect preserves previously published PocketBase file
// URLs after the photo migration, using the asset's current access rules.
func LegacyPhotoFileRedirect(e *core.RequestEvent) error {
	if e.Request.Method != http.MethodGet && e.Request.Method != http.MethodHead {
		return e.Next()
	}
	if !strings.HasPrefix(e.Request.URL.Path, "/api/files/") {
		return e.Next()
	}
	collection, err := e.App.FindCachedCollectionByNameOrId(e.Request.PathValue("collection"))
	if err != nil {
		return e.Next()
	}
	var linkCollection, targetField string
	switch collection.Name {
	case "trails":
		linkCollection, targetField = "trail_assets", "trail"
	case "waypoints":
		linkCollection, targetField = "waypoint_assets", "waypoint"
	case "summit_logs":
		linkCollection, targetField = "summit_log_assets", "summit_log"
	default:
		return e.Next()
	}
	previousCache := e.Response.Header().Get("Cache-Control")
	e.Response.Header().Set("Cache-Control", "no-store")
	record, err := e.App.FindRecordById(collection, e.Request.PathValue("recordId"))
	if errors.Is(err, sql.ErrNoRows) {
		return e.Next()
	}
	if err != nil {
		return e.InternalServerError("Failed to resolve legacy photo target", err)
	}
	// Current files such as GPX tracks use the normal download path. They
	// need no lookup among migrated photo links.
	for _, field := range collection.Fields {
		if fileField, ok := field.(*core.FileField); ok {
			for _, filename := range record.GetStringSlice(fileField.Name) {
				if filename == e.Request.PathValue("filename") {
					if previousCache == "" {
						e.Response.Header().Del("Cache-Control")
					} else {
						e.Response.Header().Set("Cache-Control", previousCache)
					}
					return e.Next()
				}
			}
		}
	}
	e.Response.Header().Add("Vary", "Authorization")
	info, err := e.RequestInfo()
	if err != nil {
		return e.InternalServerError("Failed to authorize legacy photo", err)
	}
	if allowed, err := e.App.CanAccessRecord(record, info, record.Collection().ViewRule); !allowed {
		return e.NotFoundError("", err)
	}
	links, err := e.App.FindCollectionByNameOrId(linkCollection)
	if err != nil {
		return e.InternalServerError("Failed to resolve legacy photo links", err)
	}
	author := record.GetString("author")
	if collection.Name == "waypoints" {
		author = legacyWaypointPhotoAuthor(e.App, record)
	}
	link := core.NewRecord(links)
	err = e.App.RecordQuery(linkCollection).
		InnerJoin("assets", dbx.NewExp("assets.id = "+linkCollection+".asset")).
		InnerJoin(util.LegacyPhotoAliasTable, dbx.NewExp(util.LegacyPhotoAliasTable+".asset = assets.id")).
		AndWhere(dbx.HashExp{
			linkCollection + "." + targetField: record.Id,
			"assets.author":                    author, "assets.type": "photo",
			util.LegacyPhotoAliasTable + ".source_collection": collection.Name,
			util.LegacyPhotoAliasTable + ".source_record":     record.Id,
			util.LegacyPhotoAliasTable + ".source_file":       e.Request.PathValue("filename"),
		}).Limit(1).One(link)
	if errors.Is(err, sql.ErrNoRows) {
		return e.Next()
	}
	if err != nil {
		return e.InternalServerError("Failed to resolve legacy photo", err)
	}
	asset, err := util.FindAssetMediaRecord(e.App, link.GetString("asset"))
	if err != nil {
		return e.InternalServerError("Failed to resolve legacy photo", err)
	}
	if allowed, err := e.App.CanAccessRecord(asset, info, asset.Collection().ViewRule); !allowed {
		return e.NotFoundError("", err)
	}
	path := "/assets/" + url.PathEscape(asset.Id) + "/file"
	if filename := asset.GetString("file"); filename != "" {
		path = "/api/files/" + url.PathEscape(asset.Collection().Id) + "/" + url.PathEscape(asset.Id) + "/" + url.PathEscape(filename)
	}
	if query := e.Request.URL.RawQuery; query != "" {
		path += "?" + query
	}
	e.Response.Header().Set("Cache-Control", "private, max-age=3600")
	if err := e.Redirect(http.StatusTemporaryRedirect, path); err != nil {
		e.Response.Header().Set("Cache-Control", "no-store")
		return err
	}
	return nil
}

func legacyWaypointPhotoAuthor(app core.App, record *core.Record) string {
	resolve := func(raw string) string {
		if raw == "" {
			return ""
		}
		if actor, err := app.FindRecordById("activitypub_actors", raw); err == nil {
			return actor.Id
		}
		if actor, err := app.FindFirstRecordByData("activitypub_actors", "user", raw); err == nil {
			return actor.Id
		}
		return ""
	}
	if trail, err := app.FindRecordById("trails", record.GetString("trail")); err == nil {
		if author := resolve(trail.GetString("author")); author != "" {
			return author
		}
	}
	if author := resolve(record.GetString("author")); author != "" {
		return author
	}
	return record.GetString("author")
}
