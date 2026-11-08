package routes

import (
	"database/sql"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func userOwnsTrail(app core.App, userID, trailID string) (bool, error) {
	if userID == "" || trailID == "" {
		return false, nil
	}
	trail, err := app.FindRecordById("trails", trailID)
	if err != nil {
		return false, err
	}
	return actorBelongsToUser(app, userID, trail.GetString("author"))
}

func userCanEditTrailTarget(app core.App, userID, trailID string) (bool, error) {
	if userID == "" || trailID == "" {
		return false, nil
	}
	owns, err := userOwnsTrail(app, userID, trailID)
	if err != nil {
		return false, err
	}
	if owns {
		return true, nil
	}

	actor, err := app.FindFirstRecordByData("activitypub_actors", "user", userID)
	if err != nil {
		return false, err
	}
	_, err = app.FindFirstRecordByFilter(
		"trail_share",
		"trail={:trail} && actor={:actor} && permission='edit'",
		dbx.Params{"trail": trailID, "actor": actor.Id},
	)
	if err == nil {
		return true, nil
	}
	if err == sql.ErrNoRows {
		return false, nil
	}
	return false, err
}

func userCanEditWaypointTarget(app core.App, userID, waypointID, trailID string) (bool, error) {
	if userID == "" || waypointID == "" {
		return false, nil
	}
	waypoint, err := app.FindRecordById("waypoints", waypointID)
	if err != nil {
		return false, err
	}
	if trailID != "" && waypoint.GetString("trail") != trailID {
		return false, nil
	}
	return userCanEditTrailTarget(app, userID, waypoint.GetString("trail"))
}

func userCanEditSummitLogTarget(app core.App, userID, summitLogID, trailID string) (bool, error) {
	if userID == "" || summitLogID == "" {
		return false, nil
	}
	summitLog, err := app.FindRecordById("summit_logs", summitLogID)
	if err != nil {
		return false, err
	}
	if trailID != "" && summitLog.GetString("trail") != trailID {
		return false, nil
	}
	if ok, err := actorBelongsToUser(app, userID, summitLog.GetString("author")); err != nil || ok {
		return ok, err
	}
	return userOwnsTrail(app, userID, summitLog.GetString("trail"))
}

func actorBelongsToUser(app core.App, userID, actorID string) (bool, error) {
	if userID == "" || actorID == "" {
		return false, nil
	}
	actor, err := app.FindRecordById("activitypub_actors", actorID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return actor.GetString("user") == userID, nil
}

func ensureCanEditTrailTarget(app core.App, userID, trailID string) error {
	ok, err := userCanEditTrailTarget(app, userID, trailID)
	if err != nil {
		return err
	}
	if !ok {
		return apis.NewForbiddenError("Insufficient permissions for trail", nil)
	}
	return nil
}

func ensureCanEditSummitLogTarget(app core.App, userID, summitLogID, trailID string) error {
	ok, err := userCanEditSummitLogTarget(app, userID, summitLogID, trailID)
	if err != nil {
		return err
	}
	if !ok {
		return apis.NewForbiddenError("Insufficient permissions for summit log", nil)
	}
	return nil
}

func ensureCanEditWaypointTarget(app core.App, userID, waypointID, trailID string) error {
	ok, err := userCanEditWaypointTarget(app, userID, waypointID, trailID)
	if err != nil {
		return err
	}
	if !ok {
		return apis.NewForbiddenError("Insufficient permissions for waypoint", nil)
	}
	return nil
}
