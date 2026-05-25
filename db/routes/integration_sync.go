package routes

import (
	"net/http"

	"pocketbase/integrations"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// IntegrationSync triggers a manual run of all provider syncs (Strava, Komoot,
// Hammerhead). It requires authentication and starts the sync in the background,
// returning 202 Accepted immediately because a full sync can take minutes. If a
// sync is already running - whether the nightly cron or another manual trigger -
// it returns 409 Conflict so the UI can ask the user to wait.
//
// Note: the underlying sync processes every user's integrations, so on a
// multi-user instance any authenticated user triggers a global sync. This is
// acceptable for single-user self-hosted setups; the in-progress lock prevents
// it from being used to start overlapping runs.
func IntegrationSync(client meilisearch.ServiceManager) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if e.Auth == nil {
			return e.UnauthorizedError("authentication required", nil)
		}

		if !integrations.StartManualSync(e.App, client) {
			return apis.NewApiError(http.StatusConflict, "a sync is already in progress", nil)
		}

		return e.JSON(http.StatusAccepted, map[string]string{"status": "started"})
	}
}
