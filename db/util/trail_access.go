package util

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// TrailViewableByUser mirrors the trails view/read rule for custom backend
// routes that load a trail server-side and therefore bypass PocketBase's normal
// collection API permission checks.
func TrailViewableByUser(app core.App, trail *core.Record, userID string, shareToken string) bool {
	if trail == nil || userID == "" {
		return false
	}
	if trail.GetBool("public") {
		return true
	}

	actor, err := app.FindFirstRecordByData("activitypub_actors", "user", userID)
	if err != nil {
		return false
	}
	if trail.GetString("author") == actor.Id {
		return true
	}

	share, err := app.FindFirstRecordByFilter(
		"trail_share",
		"trail={:trail} && actor={:actor}",
		dbx.Params{"trail": trail.Id, "actor": actor.Id},
	)
	if err == nil && share != nil {
		return true
	}

	if shareToken == "" {
		return false
	}
	linkShare, err := app.FindFirstRecordByFilter(
		"trail_link_share",
		"trail={:trail} && token={:token}",
		dbx.Params{"trail": trail.Id, "token": shareToken},
	)
	return err == nil && linkShare != nil
}
