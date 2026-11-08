package routes

import (
	"database/sql"
	"errors"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"net/http"
	"net/url"
	"pocketbase/util"
)

type assetDeleteLinkTarget struct {
	collection string
	field      string
	id         string
}

func AssetDelete(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}
	assetID := e.Request.PathValue("id")
	if assetID == "" {
		return apis.NewBadRequestError("asset id is required", nil)
	}

	actor, err := e.App.FindFirstRecordByData("activitypub_actors", "user", e.Auth.Id)
	if err != nil {
		return apis.NewUnauthorizedError("actor not found", err)
	}

	canWithdraw := false
	assetMissing := false
	query := e.Request.URL.Query()
	if query.Get("trail") != "" || query.Get("waypoint") != "" || query.Get("summit_log") != "" {
		asset, err := util.FindAssetMediaRecord(e.App, assetID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		assetMissing = errors.Is(err, sql.ErrNoRows)
		canWithdraw = asset != nil && asset.GetString("author") == actor.Id
	}
	targets, err := assetDeleteLinkTargets(e, canWithdraw || assetMissing)
	if err != nil {
		return err
	}

	if assetMissing {
		return e.JSON(http.StatusOK, map[string]any{"acknowledged": true, "assetDeleted": false})
	}

	if len(targets) == 0 {
		asset, err := e.App.FindRecordById("assets", assetID)
		if err != nil {
			return e.NotFoundError("asset not found", err)
		}
		if asset.GetString("author") != actor.Id {
			return apis.NewForbiddenError("Insufficient permissions for asset", nil)
		}
		if e.Request.URL.Query().Get("orphan") == "1" {
			deleted := false
			if err := e.App.RunInTransaction(func(txApp core.App) error {
				var err error
				deleted, err = util.DeleteAssetIfOrphanedByAuthor(txApp, assetID, actor.Id)
				return err
			}); err != nil {
				return err
			}
			return e.JSON(http.StatusOK, map[string]any{"acknowledged": true, "assetDeleted": deleted})
		}
		if err := e.App.Delete(asset); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"acknowledged": true})
	}

	assetDeleted := false
	if err := e.App.RunInTransaction(func(txApp core.App) error {
		removed := false
		for _, target := range targets {
			links, err := txApp.FindRecordsByFilter(
				target.collection,
				"asset={:asset} && "+target.field+"={:target}",
				"",
				-1,
				0,
				dbx.Params{"asset": assetID, "target": target.id},
			)
			if err != nil {
				return err
			}
			for _, link := range links {
				if err := txApp.Delete(link); err != nil {
					return err
				}
				removed = true
			}
		}

		if !removed {
			return nil
		}
		deleted, err := util.DeleteAssetIfOrphanedByAuthor(txApp, assetID, actor.Id)
		if err != nil {
			return err
		}
		assetDeleted = deleted
		return nil
	}); err != nil {
		return err
	}

	return e.JSON(http.StatusOK, map[string]any{
		"acknowledged": true,
		"assetDeleted": assetDeleted,
	})
}

func assetDeleteLinkTargets(e *core.RequestEvent, contextOnly bool) ([]assetDeleteLinkTarget, error) {
	query := e.Request.URL.Query()
	trailID := query.Get("trail")
	waypointID := query.Get("waypoint")
	summitLogID := query.Get("summit_log")
	trailIsDirectTarget := trailID != "" && waypointID == "" && summitLogID == ""

	targets := []assetDeleteLinkTarget{}
	if trailIsDirectTarget {
		if contextOnly {
			if err := ensureAssetDeleteTargetContext(e.App, "trails", trailID, ""); err != nil {
				return nil, err
			}
		} else if err := ensureCanEditTrailTarget(e.App, e.Auth.Id, trailID); err != nil {
			return nil, err
		}
		targets = append(targets, assetDeleteLinkTarget{collection: "trail_assets", field: "trail", id: trailID})
	}
	if waypointID != "" {
		if contextOnly {
			if err := ensureAssetDeleteTargetContext(e.App, "waypoints", waypointID, trailID); err != nil {
				return nil, err
			}
		} else if err := ensureCanEditWaypointTarget(e.App, e.Auth.Id, waypointID, trailID); err != nil {
			return nil, err
		}
		targets = append(targets, assetDeleteLinkTarget{collection: "waypoint_assets", field: "waypoint", id: waypointID})
	}
	if summitLogID != "" {
		if contextOnly {
			if err := ensureAssetDeleteTargetContext(e.App, "summit_logs", summitLogID, trailID); err != nil {
				return nil, err
			}
		} else if err := ensureCanEditSummitLogTarget(e.App, e.Auth.Id, summitLogID, trailID); err != nil {
			return nil, err
		}
		targets = append(targets, assetDeleteLinkTarget{collection: "summit_log_assets", field: "summit_log", id: summitLogID})
	}
	return targets, nil
}

// Owned-photo withdrawals and missing-asset no-ops still require an existing
// target and, for a child, a matching supplied trail context.
func ensureAssetDeleteTargetContext(app core.App, collection, id, trailID string) error {
	target, err := app.FindRecordById(collection, id)
	if err != nil {
		return err
	}
	if trailID == "" || target.GetString("trail") == trailID {
		return nil
	}
	label := "waypoint"
	if collection == "summit_logs" {
		label = "summit log"
	}
	return apis.NewForbiddenError("Insufficient permissions for "+label, nil)
}

func AssetFile(e *core.RequestEvent) error {
	setAssetMediaHeaders(e.Response.Header())
	// Only successful photo redirects may remain in the private browser cache.
	e.Response.Header().Set("Cache-Control", "no-store")
	e.Response.Header().Add("Vary", "Authorization")
	assetID := e.Request.PathValue("id")
	if assetID == "" {
		return e.NotFoundError("", nil)
	}

	asset, err := util.FindAssetMediaRecord(e.App, assetID)
	if err != nil {
		return e.NotFoundError("", err)
	}

	requestInfo, err := e.RequestInfo()
	if err != nil {
		return e.InternalServerError("Failed to load request info", err)
	}
	if ok, _ := e.App.CanAccessRecord(asset, requestInfo, asset.Collection().ViewRule); !ok {
		return e.NotFoundError("", errors.New("insufficient permissions to access the asset"))
	}

	if fileURL := util.AssetFileRedirectURL(asset); fileURL != "" {
		e.Response.Header().Set("Cache-Control", "private, max-age=3600")
		if err := e.Redirect(http.StatusFound, assetFileRedirectURLWithQuery(fileURL, e.Request.URL.Query())); err != nil {
			e.Response.Header().Set("Cache-Control", "no-store")
			return err
		}
		return nil
	}

	return e.NotFoundError("", nil)
}

func assetFileRedirectURLWithQuery(fileURL string, requestQuery url.Values) string {
	parsed, err := url.Parse(fileURL)
	if err != nil {
		return fileURL
	}
	query := parsed.Query()
	for _, key := range []string{"thumb", "share"} {
		if value := requestQuery.Get(key); value != "" {
			query.Set(key, value)
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
