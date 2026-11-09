package hooks

import (
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

func TestFederationSaveBatchCarriesSlowUploadsInOneCreate(t *testing.T) {
	f := newFederationUpdatesFixture(t)
	synctest.Test(t, func(t *testing.T) {
		f.t = t
		f.updates.createDelay = testFederationUpdateDelay
		f.updates.batchDelay = 5 * testFederationUpdateDelay
		local := f.save(f.actors, map[string]any{"is_local": true})
		trail := f.save(f.trails, map[string]any{"public": true, "author": local.Id})
		log := f.save(f.logs, map[string]any{"trail": trail.Id, "author": local.Id})
		const batch = "save-batch-for-slow-upload"
		if !f.updates.BeginSave(log, batch) {
			t.Fatal("save batch was not accepted")
		}
		f.updates.EnqueueCreate(log)
		// The multipart upload lasts longer than the ordinary Create delay.
		time.Sleep(3 * testFederationUpdateDelay)
		f.check()
		f.link("summit_log_assets", "summit_log", log)
		if !f.updates.BeginSave(log, batch) {
			t.Fatal("save batch was not renewed")
		}
		time.Sleep(3 * testFederationUpdateDelay)
		f.check()
		f.link("summit_log_assets", "summit_log", log)
		if !f.updates.CompleteSave(log, batch) {
			t.Fatal("save batch did not complete")
		}
		f.expect(federatedActivity{typ: pub.CreateType, target: "summit_logs/" + log.Id, photos: 2})
		f.terminate()
		f.check()
	})
}

func TestFederationSaveBatchesDoNotCompleteAnotherEditorsSave(t *testing.T) {
	f := newFederationUpdatesFixture(t)
	synctest.Test(t, func(t *testing.T) {
		f.t = t
		local := f.save(f.actors, map[string]any{"is_local": true})
		trail := f.save(f.trails, map[string]any{"public": true, "author": local.Id})
		first, second := "first-editors-save-token", "second-editors-save-token"
		f.updates.BeginSave(trail, first)
		f.updates.BeginSave(trail, second)
		f.updates.EnqueueCreate(trail)
		f.link("trail_assets", "trail", trail)
		if f.updates.CompleteSave(trail, "unknown-editors-save-token") {
			t.Fatal("unknown token released a save")
		}
		if !f.updates.CompleteSave(trail, first) {
			t.Fatal("first save did not complete")
		}
		f.check()
		f.link("trail_assets", "trail", trail)
		if !f.updates.CompleteSave(trail, second) {
			t.Fatal("second save did not complete")
		}
		f.expect(federatedActivity{typ: pub.CreateType, target: "trails/" + trail.Id, photos: 2})
	})
}

func TestFederationSaveBatchExpiresWhenTheEditorDisappears(t *testing.T) {
	f := newFederationUpdatesFixture(t)
	synctest.Test(t, func(t *testing.T) {
		f.t = t
		f.updates.createDelay = testFederationUpdateDelay
		f.updates.batchDelay = 3 * testFederationUpdateDelay
		local := f.save(f.actors, map[string]any{"is_local": true})
		trail := f.save(f.trails, map[string]any{"public": true, "author": local.Id})
		f.updates.BeginSave(trail, "abandoned-editors-save-token")
		f.updates.EnqueueCreate(trail)
		f.link("trail_assets", "trail", trail)
		f.expect(federatedActivity{typ: pub.CreateType, target: "trails/" + trail.Id, photos: 1})
		if f.updates.CompleteSave(trail, "abandoned-editors-save-token") {
			t.Fatal("expired batch was retained")
		}
		// Even a batch that never changed its record is discarded on expiry.
		f.updates.BeginSave(trail, "unchanged-editors-save-token")
		time.Sleep(5 * testFederationUpdateDelay)
		f.updates.mu.Lock()
		remaining := len(f.updates.batches)
		f.updates.mu.Unlock()
		if remaining != 0 {
			t.Fatalf("abandoned leases remain in memory: %d", remaining)
		}
	})
}

func TestFederationSaveRenewalSupersedesAnAlreadyFiredTimer(t *testing.T) {
	f := newFederationUpdatesFixture(t)
	synctest.Test(t, func(t *testing.T) {
		f.t = t
		local := f.save(f.actors, map[string]any{"is_local": true})
		trail := f.save(f.trails, map[string]any{"public": true, "author": local.Id})
		const batch = "renewed-editors-save-token"
		f.updates.BeginSave(trail, batch)
		f.updates.Enqueue(trail)
		target := federationUpdateTarget{collection: "trails", id: trail.Id}
		f.updates.mu.Lock()
		oldTimer := f.updates.pending[target]
		f.updates.mu.Unlock()
		f.updates.BeginSave(trail, batch)
		// Simulate a timer callback that entered just before Stop, then waited
		// for the lease renewal's lock. It must not flush the renewed save.
		f.updates.flush(target, oldTimer)
		f.check()
		f.updates.CompleteSave(trail, batch)
		f.expect(federatedActivity{typ: pub.UpdateType, target: "trails/" + trail.Id})
	})
}

func TestFederationSaveBatchesAreBoundedAndSkipRemoteIRIs(t *testing.T) {
	f := newFederationUpdatesFixture(t)
	synctest.Test(t, func(t *testing.T) {
		f.t = t
		local := f.save(f.actors, map[string]any{"is_local": true})
		trail := f.save(f.trails, map[string]any{"public": true, "author": local.Id})
		for i := range maxFederationSaveBatchesPerTarget {
			if !f.updates.BeginSave(trail, fmt.Sprintf("editors-save-token-%d", i)) {
				t.Fatalf("save %d was rejected", i)
			}
		}
		if f.updates.BeginSave(trail, "too-many-editors-save-token") {
			t.Fatal("unbounded save tokens accepted")
		}
		if !f.updates.BeginSave(trail, "editors-save-token-0") {
			t.Fatal("renewal counted as another concurrent save")
		}
		remote := f.save(f.trails, map[string]any{"public": true, "author": local.Id, "iri": "https://b.example/api/v1/trail/copy"})
		if f.updates.BeginSave(remote, "remote-editors-save-token") {
			t.Fatal("remote IRI accepted a local save lease")
		}
		f.terminate()
	})
}

func TestRenewedSaveLeasesCannotIndefinitelyHoldAnotherEditorsChanges(t *testing.T) {
	f := newFederationUpdatesFixture(t)
	synctest.Test(t, func(t *testing.T) {
		f.t = t
		f.updates.batchDelay = time.Second
		f.updates.batchMaxDelay = 4 * testFederationUpdateDelay
		local := f.save(f.actors, map[string]any{"is_local": true})
		trail := f.save(f.trails, map[string]any{"public": true, "author": local.Id})
		for i := range maxFederationSaveBatchesPerTarget {
			if !f.updates.BeginSave(trail, fmt.Sprintf("holding-editors-save-token-%d", i)) {
				t.Fatal("lease admission failed")
			}
		}
		f.updates.Enqueue(trail)
		target := federationUpdateTarget{collection: "trails", id: trail.Id}
		f.updates.mu.Lock()
		limit := f.updates.pending[target].leaseLimit
		f.updates.mu.Unlock()
		// An editor can renew all ten admitted tokens, but cannot extend the
		// announcement beyond the first pending activity's absolute bound.
		for range 2 {
			time.Sleep(testFederationUpdateDelay)
			for i := range maxFederationSaveBatchesPerTarget {
				f.updates.BeginSave(trail, fmt.Sprintf("holding-editors-save-token-%d", i))
			}
			f.updates.mu.Lock()
			pending := f.updates.pending[target]
			if pending == nil || !pending.deadline.Equal(limit) || !pending.leaseLimit.Equal(limit) {
				f.updates.mu.Unlock()
				t.Fatal("renewed leases extended the announcement deadline")
			}
			f.updates.mu.Unlock()
		}
		f.expect(federatedActivity{typ: pub.UpdateType, target: "trails/" + trail.Id})
		f.terminate()
	})
}

func TestCompletingSaveReturnsWhileActivityDeliveryIsBlocked(t *testing.T) {
	f := newFederationUpdatesFixture(t)
	synctest.Test(t, func(t *testing.T) {
		f.t = t
		local := f.save(f.actors, map[string]any{"is_local": true})
		trail := f.save(f.trails, map[string]any{"public": true, "author": local.Id})
		started, release := make(chan struct{}), make(chan struct{})
		defer close(release)
		f.updates.send = func(core.App, *core.Record, *core.Record, pub.ActivityVocabularyType) error {
			close(started)
			<-release
			return nil
		}
		const token = "completed-editors-save-token"
		f.updates.BeginSave(trail, token)
		f.updates.Enqueue(trail)
		completed := make(chan bool, 1)
		go func() { completed <- f.updates.CompleteSave(trail, token) }()
		select {
		case accepted := <-completed:
			if !accepted {
				t.Fatal("completion was rejected")
			}
		case <-time.After(2 * testFederationUpdateDelay):
			t.Fatal("completion waited for activity delivery")
		}
		select {
		case <-started:
		case <-time.After(2 * testFederationUpdateDelay):
			t.Fatal("completion did not schedule activity delivery")
		}
	})
}

func TestFederationSaveBatchAnnouncesEntryAfterPhotoRemoval(t *testing.T) {
	f := newFederationUpdatesFixture(t)
	synctest.Test(t, func(t *testing.T) {
		f.t = t
		author := f.save(f.actors, map[string]any{"is_local": true})
		trail := f.save(f.trails, map[string]any{"public": true, "author": author.Id})
		entry := f.save(f.logs, map[string]any{"trail": trail.Id, "author": author.Id})
		removed := f.link("summit_log_assets", "summit_log", entry)
		f.link("summit_log_assets", "summit_log", entry)
		time.Sleep(6 * testFederationUpdateDelay)
		f.check(federatedActivity{typ: pub.UpdateType, target: "summit_logs/" + entry.Id, photos: 2})
		const token = "entry-editor-save-token"
		if !f.updates.BeginSave(entry, token) {
			t.Fatal("entry lease not acquired")
		}
		f.updates.Enqueue(entry)
		time.Sleep(6 * testFederationUpdateDelay)
		f.check()
		if err := f.app.Delete(removed); err != nil {
			t.Fatal(err)
		}
		if !f.updates.CompleteSave(entry, token) {
			t.Fatal("entry lease not completed")
		}
		f.expect(federatedActivity{typ: pub.UpdateType, target: "summit_logs/" + entry.Id, photos: 1})
	})
}
