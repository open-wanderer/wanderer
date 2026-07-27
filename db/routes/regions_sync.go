package routes

import (
	"log"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"pocketbase/services/regions"
)

// RegionSyncStart triggers a manual "Sync now": the same BuildAll pass the
// nightly cron runs (build every currently-enabled leaf's vector/DEM
// archives), but kicked off on demand from the admin catalog page.
//
// The sync slot (regions.TryStartSync/FinishSync) is claimed synchronously
// here, before the goroutine is spawned, so a second click (or an
// overlapping cron tick) reliably gets 409 Conflict rather than racing to
// start a second full pass — the goroutine itself only ever runs
// BuildAllLocked, which assumes the slot is already held.
//
// The pass can run for a long time (BuildAllLocked builds one region fully
// before starting the next, deliberately unparallelized — see builder.go),
// so this returns 202 immediately rather than blocking the request; the
// admin page polls RegionSyncStatus / the region_archives collection to
// track progress.
func RegionSyncStart(e *core.RequestEvent) error {
	if !regions.TryStartSync() {
		return e.JSON(http.StatusConflict, map[string]any{"running": true})
	}

	go func() {
		defer regions.FinishSync()
		regions.BuildAllLocked(e.App)
	}()

	return e.JSON(http.StatusAccepted, map[string]any{"started": true})
}

// RegionSyncStatus reports whether a BuildAll pass (cron-triggered or
// manually triggered) is currently in progress, so the admin page can
// disable/re-enable "Sync now" correctly even across a page reload — the
// button's disabled state must not depend solely on the click that
// triggered it, since a sync started in one tab (or by the cron) must also
// disable the button in every other tab/session. Also reports when the
// nightly cron will next fire on its own (RFC3339 UTC), so the admin page
// can show that as a hint — omitted entirely if it can't be computed rather
// than failing the whole status check over a decorative field.
func RegionSyncStatus(e *core.RequestEvent) error {
	resp := map[string]any{"running": regions.SyncRunning()}

	next, err := regions.NextScheduledSync(regions.CronSchedule())
	if err != nil {
		log.Printf("[regions] failed to compute next scheduled sync: %v", err)
	} else {
		resp["nextScheduledRun"] = next.Format(time.RFC3339)
	}

	return e.JSON(http.StatusOK, resp)
}
