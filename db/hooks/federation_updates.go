package hooks

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"maps"
	"pocketbase/federation"
	"pocketbase/util"
	"slices"
	"sync"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/hook"
)

// federationUpdateDelay is how long the Update of a trail or summit log waits
// for further changes. The editor saves the record first and its photo links
// one request at a time afterwards.
const federationUpdateDelay = 2 * time.Second

// federationCreateDelay is how long the Create of a trail or summit log waits.
// The editor uploads the photos of new content after saving the record, and
// they have to go out with the Create: an Update following with them would
// notify the mentioned users a second time.
const federationCreateDelay = 10 * time.Second

// Editors renew this lease while uploads are running and complete it once the
// save finishes. Abandoned saves still go out after this bounded fallback.
const federationSaveBatchDelay = time.Minute
const federationSaveBatchMaxDelay = 5 * time.Minute
const maxFederationSaveBatchesPerTarget = 10

// federationShutdownTimeout bounds how long the app's shutdown waits for the
// pending activities, which it sends right away, and for their deliveries.
const federationShutdownTimeout = 5 * time.Second

// FederationUpdates sends the Create or Update of a trail or summit log once
// its changes have settled: one for a save from the editor, built after the
// photo links that came with it, and one for a photo that is linked, unlinked
// or marked as thumbnail on its own. Pending activities are kept in memory
// only: a shutdown sends them right away and waits for their delivery, a crash
// loses them.
type FederationUpdates struct {
	app             core.App
	updateDelay     time.Duration
	createDelay     time.Duration
	batchDelay      time.Duration
	batchMaxDelay   time.Duration
	shutdownTimeout time.Duration
	send            func(app core.App, author *core.Record, record *core.Record, typ pub.ActivityVocabularyType) error
	mu              sync.Mutex
	pending         map[federationUpdateTarget]*pendingFederationUpdate
	batches         map[federationUpdateTarget]*federationSaveBatches
	// federating counts the activities that are being built and handed over
	// for delivery, so that a shutdown can wait for those whose timers have
	// fired before it. Unlike a WaitGroup, it may be added to while it is
	// waited for.
	federating int
	// idle is closed when the last of them has been handed over.
	idle chan struct{}
}

type federationUpdateTarget struct {
	collection string
	id         string
}

type pendingFederationUpdate struct {
	timer      *time.Timer
	typ        pub.ActivityVocabularyType
	deadline   time.Time
	leaseLimit time.Time
}

type federationSaveBatches struct {
	deadlines map[string]time.Time
	timer     *time.Timer
}

// NewFederationUpdates sends through app, which must not be a transaction: the
// activities go out after the request that caused them.
func NewFederationUpdates(app core.App) *FederationUpdates {
	return newFederationUpdates(app, federationUpdateDelay, federationCreateDelay, sendFederatedActivity)
}

func newFederationUpdates(app core.App, updateDelay time.Duration, createDelay time.Duration, send func(core.App, *core.Record, *core.Record, pub.ActivityVocabularyType) error) *FederationUpdates {
	u := &FederationUpdates{
		app:             app,
		updateDelay:     updateDelay,
		createDelay:     createDelay,
		batchDelay:      federationSaveBatchDelay,
		batchMaxDelay:   federationSaveBatchMaxDelay,
		shutdownTimeout: federationShutdownTimeout,
		send:            send,
		pending:         map[federationUpdateTarget]*pendingFederationUpdate{},
		batches:         map[federationUpdateTarget]*federationSaveBatches{},
	}
	// The timers do not outlive the process. This runs after PocketBase has
	// stopped its web server (priority -9999), which gives running requests a
	// second to finish, and before it writes its logs for the last time (-999).
	app.OnTerminate().Bind(&hook.Handler[*core.TerminateEvent]{
		Func: func(e *core.TerminateEvent) error {
			u.flushAll()
			return e.Next()
		},
		Priority: -1000,
	})
	return u
}

// Enqueue schedules the Update of a trail or summit log that followers can see
// at the time of the change: a public trail, or a summit log on one. What
// changes while a trail is private goes out with its publication, if at all.
func (u *FederationUpdates) Enqueue(record *core.Record) {
	u.enqueue(record, pub.UpdateType)
}

// EnqueueCreate schedules the Create of a trail or summit log that followers
// see for the first time. Changes until it goes out, photo links included,
// are sent with it rather than in an Update of their own.
func (u *FederationUpdates) EnqueueCreate(record *core.Record) {
	u.enqueue(record, pub.CreateType)
}

// EnqueueTrailSave schedules the activity for a saved trail: a Create when the
// save made it public, as its followers have not seen it before, and an
// Update otherwise.
func (u *FederationUpdates) EnqueueTrailSave(trail *core.Record) {
	if !trail.Original().GetBool("public") && trail.GetBool("public") {
		u.EnqueueCreate(trail)
		return
	}
	u.Enqueue(trail)
}

func (u *FederationUpdates) enqueue(record *core.Record, typ pub.ActivityVocabularyType) {
	if record == nil || !u.visible(record) {
		return
	}
	target := federationUpdateTarget{collection: record.Collection().Name, id: record.Id}
	delay := u.updateDelay
	if typ == pub.CreateType {
		delay = u.createDelay
	}
	deadline := time.Now().Add(delay)

	u.mu.Lock()
	defer u.mu.Unlock()
	if previous, ok := u.pending[target]; ok {
		previous.timer.Stop()
		// A pending Create stays one, and a later change does not cut short
		// the time it waits for the photos.
		if previous.typ == pub.CreateType {
			typ = pub.CreateType
		}
		if previous.deadline.After(deadline) {
			deadline = previous.deadline
		}
	}
	if batchDeadline := u.batchDeadlineLocked(target, time.Now()); batchDeadline.After(deadline) {
		deadline = batchDeadline
	}
	u.scheduleLocked(target, typ, deadline)
}

func (u *FederationUpdates) scheduleLocked(target federationUpdateTarget, typ pub.ActivityVocabularyType, deadline time.Time) {
	leaseLimit := time.Now().Add(u.batchMaxDelay)
	if previous := u.pending[target]; previous != nil {
		leaseLimit = previous.leaseLimit
	}
	if deadline.After(leaseLimit) {
		deadline = leaseLimit
	}
	update := &pendingFederationUpdate{typ: typ, deadline: deadline, leaseLimit: leaseLimit}
	update.timer = time.AfterFunc(time.Until(deadline), func() {
		u.flush(target, update)
	})
	u.pending[target] = update
}

// BeginSave holds this record's pending activity during a multi-request save.
// Renewing the same token extends its bounded lease. Concurrent editors keep
// separate tokens so one completed save cannot flush another editor's uploads.
func (u *FederationUpdates) BeginSave(record *core.Record, batchID string) bool {
	if record == nil || !util.ValidFederationSaveBatchID(batchID) || (record.Collection().Name != "trails" && record.Collection().Name != "summit_logs") {
		return false
	}
	if iri := record.GetString("iri"); iri != "" && !util.RecordHasLocalIdentity(u.app, record) {
		return false
	}
	target := federationUpdateTarget{collection: record.Collection().Name, id: record.Id}
	u.mu.Lock()
	defer u.mu.Unlock()
	u.batchDeadlineLocked(target, time.Now())
	batches := u.batches[target]
	if batches == nil {
		batches = &federationSaveBatches{deadlines: map[string]time.Time{}}
		u.batches[target] = batches
	}
	if _, exists := batches.deadlines[batchID]; !exists && len(batches.deadlines) >= maxFederationSaveBatchesPerTarget {
		return false
	}
	batches.deadlines[batchID] = time.Now().Add(u.batchDelay)
	deadline := u.batchDeadlineLocked(target, time.Now())
	if batches.timer != nil {
		batches.timer.Stop()
	}
	batches.timer = time.AfterFunc(time.Until(deadline), func() {
		u.mu.Lock()
		defer u.mu.Unlock()
		if u.batches[target] == batches {
			u.batchDeadlineLocked(target, time.Now())
		}
	})
	if pending := u.pending[target]; pending != nil {
		pending.timer.Stop()
		if pending.deadline.After(deadline) {
			deadline = pending.deadline
		}
		u.scheduleLocked(target, pending.typ, deadline)
	}
	return true
}

// CompleteSave releases only this save's lease and sends the pending activity
// immediately once all active saves have finished.
func (u *FederationUpdates) CompleteSave(record *core.Record, batchID string) bool {
	if record == nil || !util.ValidFederationSaveBatchID(batchID) {
		return false
	}
	target := federationUpdateTarget{collection: record.Collection().Name, id: record.Id}
	u.mu.Lock()
	batches := u.batches[target]
	if batches == nil {
		u.mu.Unlock()
		return false
	}
	if _, exists := batches.deadlines[batchID]; !exists {
		u.mu.Unlock()
		return false
	}
	delete(batches.deadlines, batchID)
	if !u.batchDeadlineLocked(target, time.Now()).IsZero() {
		u.mu.Unlock()
		return true
	}
	pending := u.pending[target]
	if pending == nil {
		u.mu.Unlock()
		return true
	}
	pending.timer.Stop()
	// Building the activity can resolve mentions over the network. Release
	// the request immediately and let the ordinary timer/shutdown lifecycle
	// deliver it in the background.
	u.scheduleLocked(target, pending.typ, time.Now())
	u.mu.Unlock()
	return true
}

// The caller holds u.mu. Expired leases never extend a later save's timer.
func (u *FederationUpdates) batchDeadlineLocked(target federationUpdateTarget, now time.Time) time.Time {
	batches := u.batches[target]
	if batches == nil {
		return time.Time{}
	}
	latest := time.Time{}
	for token, deadline := range batches.deadlines {
		if !deadline.After(now) {
			delete(batches.deadlines, token)
		} else if deadline.After(latest) {
			latest = deadline
		}
	}
	if len(batches.deadlines) == 0 {
		if batches.timer != nil {
			batches.timer.Stop()
		}
		delete(u.batches, target)
	}
	return latest
}

func (u *FederationUpdates) visible(record *core.Record) bool {
	if iri := record.GetString("iri"); iri != "" && !util.RecordHasLocalIdentity(u.app, record) {
		return false
	}
	switch record.Collection().Name {
	case "trails":
		return record.GetBool("public")
	case "summit_logs":
		if record.GetBool("federation_suppressed") {
			return false
		}
		trail, err := u.app.FindRecordById("trails", record.GetString("trail"))
		return err == nil && trail.GetBool("public")
	}
	return false
}

func (u *FederationUpdates) flush(target federationUpdateTarget, update *pendingFederationUpdate) {
	u.mu.Lock()
	current := u.pending[target] == update
	if current {
		delete(u.pending, target)
		// Counted under the same lock, so that a shutdown that no longer
		// finds it pending waits for it.
		u.startFederating()
	}
	u.mu.Unlock()
	if !current {
		// A later change has rescheduled it.
		return
	}
	defer u.finishFederating()
	u.federate(target, update.typ)
}

// flushAll sends what is pending right away, as the app shuts down, in the
// order its timers would have, and waits for the activities whose timers have
// fired before and that are still being built. The inboxes receive the
// activities in the background, which ends with the process, so it then waits
// for these deliveries and for any others still running. It waits up to the
// shutdown timeout in all and leaves what takes longer behind.
func (u *FederationUpdates) flushAll() {
	ctx, cancel := context.WithTimeout(context.Background(), u.shutdownTimeout)
	defer cancel()

	u.mu.Lock()
	pending := u.pending
	u.pending = map[federationUpdateTarget]*pendingFederationUpdate{}
	for _, batches := range u.batches {
		if batches.timer != nil {
			batches.timer.Stop()
		}
	}
	u.batches = map[federationUpdateTarget]*federationSaveBatches{}
	for _, update := range pending {
		update.timer.Stop()
	}
	// The sends below are counted with those of the timers that have fired,
	// so idle is closed once all of them have been handed over.
	u.startFederating()
	idle := u.idle
	u.mu.Unlock()

	targets := slices.SortedFunc(maps.Keys(pending), func(a, b federationUpdateTarget) int {
		return pending[a].deadline.Compare(pending[b].deadline)
	})
	go func() {
		defer u.finishFederating()
		for _, target := range targets {
			u.federate(target, pending[target].typ)
		}
	}()
	var err error
	select {
	case <-idle:
		err = federation.WaitForDeliveries(ctx)
	case <-ctx.Done():
		err = ctx.Err()
	}
	if err != nil {
		u.app.Logger().Warn("stopped waiting for the federation activities on shutdown", "timeout", u.shutdownTimeout)
	}
}

// startFederating counts an activity that is about to be built and handed
// over for delivery. The caller holds u.mu.
func (u *FederationUpdates) startFederating() {
	if u.federating == 0 {
		u.idle = make(chan struct{})
	}
	u.federating++
}

func (u *FederationUpdates) finishFederating() {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.federating--
	if u.federating == 0 {
		close(u.idle)
	}
}

// federate sends the Create or Update of a trail or summit log as it is now.
func (u *FederationUpdates) federate(target federationUpdateTarget, typ pub.ActivityVocabularyType) {
	app := u.app
	defer func() {
		if r := recover(); r != nil {
			app.Logger().Error(fmt.Sprintf("Recovered from panic while federating the %s of %s %s: %v", typ, target.collection, target.id, r))
		}
	}()

	record, err := app.FindRecordById(target.collection, target.id)
	if errors.Is(err, sql.ErrNoRows) {
		// Deleted in the meantime, which is announced on its own.
		return
	}
	if err != nil {
		app.Logger().Warn("could not load a changed record to federate", "collection", target.collection, "record", target.id, "error", err)
		return
	}
	if !u.visible(record) {
		return
	}
	author, err := app.FindRecordById("activitypub_actors", record.GetString("author"))
	if err != nil {
		app.Logger().Warn("could not load the author of a changed record to federate", "collection", target.collection, "record", target.id, "error", err)
		return
	}
	// Copies of remote content get their links while they sync; their own
	// instance announces them.
	if !author.GetBool("is_local") {
		return
	}
	if err := u.send(app, author, record, typ); err != nil {
		app.Logger().Error("could not federate a change", "type", typ, "collection", target.collection, "record", target.id, "error", err)
	}
}

func sendFederatedActivity(app core.App, author *core.Record, record *core.Record, typ pub.ActivityVocabularyType) error {
	ctx, err := util.GetSafeActorContext(nil, author)
	if err != nil {
		return err
	}
	if record.Collection().Name == "summit_logs" {
		return federation.CreateSummitLogActivity(app, ctx, record, typ)
	}
	return federation.CreateTrailActivity(app, ctx, record, typ)
}

// FederateOnAssetLinkChange announces a photo that is linked to, unlinked from
// or marked on a trail, waypoint or summit log. Waypoints are not federated on
// their own: the Update of their trail has followers fetch it again.
func FederateOnAssetLinkChange(updates *FederationUpdates) func(e *core.RecordEvent) error {
	return func(e *core.RecordEvent) error {
		updates.Enqueue(federatedRecordForAssetLink(e.App, e.Record))
		return e.Next()
	}
}

// federatedRecordForAssetLink returns the trail or summit log a link belongs
// to, or nil when that has gone and taken the link with it.
func federatedRecordForAssetLink(app core.App, link *core.Record) *core.Record {
	var record *core.Record
	var err error
	switch link.Collection().Name {
	case "trail_assets":
		record, err = app.FindRecordById("trails", link.GetString("trail"))
	case "waypoint_assets":
		var trailID string
		if trailID, err = util.TrailIDForLinkTarget(app, "waypoint", link.GetString("waypoint")); err == nil {
			record, err = app.FindRecordById("trails", trailID)
		}
	case "summit_log_assets":
		record, err = app.FindRecordById("summit_logs", link.GetString("summit_log"))
	}
	if err != nil {
		return nil
	}
	return record
}
