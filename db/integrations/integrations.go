package integrations

import (
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/integrations/hammerhead"
	"pocketbase/integrations/komoot"
	"pocketbase/integrations/strava"
)

// ErrSyncInProgress is returned by RunAll when another sync run is already
// active. Callers can detect it with errors.Is to surface a friendly message.
var ErrSyncInProgress = errors.New("a sync is already in progress")

// syncRunning guards against overlapping sync runs. Both the nightly cron and
// the manual trigger share it, so a manual run cannot collide with the cron run
// (or another manual run), which could otherwise create duplicate work or race
// on the same integration records.
var syncRunning atomic.Bool

// RunAll runs every provider sync sequentially. It returns ErrSyncInProgress
// immediately if another run is already active. Per-provider failures are
// logged and do not stop the remaining providers.
func RunAll(app core.App, client meilisearch.ServiceManager) error {
	if !syncRunning.CompareAndSwap(false, true) {
		return ErrSyncInProgress
	}
	defer syncRunning.Store(false)

	runProviders(app, client)
	return nil
}

// StartManualSync starts a sync run in the background and returns true if it was
// started, or false if a run is already in progress. It is meant for the manual
// trigger route, which should respond immediately rather than block for the
// (potentially minutes-long) duration of a full sync.
func StartManualSync(app core.App, client meilisearch.ServiceManager) bool {
	if !syncRunning.CompareAndSwap(false, true) {
		return false
	}

	go func() {
		defer syncRunning.Store(false)
		app.Logger().Info("manual integration sync started")
		runProviders(app, client)
		app.Logger().Info("manual integration sync finished")
	}()

	return true
}

func runProviders(app core.App, client meilisearch.ServiceManager) {
	if err := strava.SyncStrava(app, client); err != nil {
		warning := fmt.Sprintf("Error syncing with strava: %v", err)
		fmt.Println(warning)
		app.Logger().Error(warning)
	}
	if err := komoot.SyncKomoot(app, client); err != nil {
		warning := fmt.Sprintf("Error syncing with komoot: %v", err)
		fmt.Println(warning)
		app.Logger().Error(warning)
	}
	if err := hammerhead.SyncHammerhead(app, client); err != nil {
		warning := fmt.Sprintf("Error syncing with hammerhead: %v", err)
		fmt.Println(warning)
		app.Logger().Error(warning)
	}
}
