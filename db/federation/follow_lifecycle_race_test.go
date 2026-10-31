package federation

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pub "github.com/go-ap/activitypub"
	"github.com/pocketbase/pocketbase/core"
)

// These tests run Undo(F1) and Follow(F2) concurrently, as a peer retrying
// after a rejection does. Interleavings are forced through PocketBase hooks.

const (
	raceHookWait    = 300 * time.Millisecond
	raceGoroutineTO = 5 * time.Second
	raceUndoIRI     = "https://peer.example.com/api/v1/activitypub/activity/u1"
)

// raceRun is a function running in its own goroutine. done is closed when it
// has returned; err is only valid after that.
type raceRun struct {
	done chan struct{}
	err  error
}

func raceStart(fn func() error) *raceRun {
	r := &raceRun{done: make(chan struct{})}
	go func() {
		defer close(r.done)
		r.err = fn()
	}()
	return r
}

// wait reports whether the run finished within d.
func (r *raceRun) wait(d time.Duration) bool {
	select {
	case <-r.done:
		return true
	case <-time.After(d):
		return false
	}
}

func raceAssertSinglePendingF2(t *testing.T, app core.App, remote, local *core.Record) {
	t.Helper()
	if n := lifecycleCountRows(t, app); n != 1 {
		t.Fatalf("want exactly 1 follows row, got %d", n)
	}
	row := lifecycleRow(t, app, remote.Id, local.Id)
	if row == nil {
		t.Fatalf("no follows row for the pair; the peer's retry is lost")
	}
	if row.GetString("status") != "pending" || row.GetString("activity_iri") != lifecycleF2 {
		t.Fatalf("want pending row with activity_iri F2, got status=%s activity_iri=%s",
			row.GetString("status"), row.GetString("activity_iri"))
	}
}

// Interleaving 1: Follow(F2) has read the rejected F1 row, Undo(F1) runs before
// the Follow writes.
func TestLifecycleConcurrentFollowReadsBeforeUndoDeletes(t *testing.T) {
	app := newLifecycleTestApp(t)
	local, remote := lifecycleInboundFixture(t, app)

	f1 := lifecycleFollow(lifecycleF1, remote.GetString("iri"), local.GetString("iri"))
	if err := ProcessFollowActivity(app, remote, f1); err != nil {
		t.Fatalf("F1: %v", err)
	}
	lifecycleSetStatus(t, app, lifecycleRow(t, app, remote.Id, local.Id), "rejected")

	var fired atomic.Bool
	var undoDone *raceRun
	app.OnRecordUpdate("follows").BindFunc(func(e *core.RecordEvent) error {
		if fired.CompareAndSwap(false, true) {
			undoDone = raceStart(func() error {
				return ProcessUndoActivity(app, remote, lifecycleUndo(raceUndoIRI, f1))
			})
			// With the Follow's transaction open, the Undo blocks and the wait expires.
			undoDone.wait(raceHookWait)
		}
		return e.Next()
	})

	f2 := lifecycleFollow(lifecycleF2, remote.GetString("iri"), local.GetString("iri"))
	if err := ProcessFollowActivity(app, remote, f2); err != nil {
		t.Fatalf("F2: %v", err)
	}
	if !fired.Load() {
		t.Fatalf("update hook never fired; the interleaving was not injected")
	}
	if !undoDone.wait(raceGoroutineTO) {
		t.Fatalf("injected Undo did not finish within %s", raceGoroutineTO)
	}
	if undoDone.err != nil {
		t.Fatalf("injected Undo: %v", undoDone.err)
	}
	raceAssertSinglePendingF2(t, app, remote, local)
}

// Interleaving 2: Undo(F1) has read the F1 row, Follow(F2) runs before the
// Undo deletes.
func TestLifecycleConcurrentUndoReadsBeforeFollowWrites(t *testing.T) {
	app := newLifecycleTestApp(t)
	local, remote := lifecycleInboundFixture(t, app)

	f1 := lifecycleFollow(lifecycleF1, remote.GetString("iri"), local.GetString("iri"))
	if err := ProcessFollowActivity(app, remote, f1); err != nil {
		t.Fatalf("F1: %v", err)
	}
	lifecycleSetStatus(t, app, lifecycleRow(t, app, remote.Id, local.Id), "rejected")

	f2 := lifecycleFollow(lifecycleF2, remote.GetString("iri"), local.GetString("iri"))
	var fired atomic.Bool
	var followDone *raceRun
	app.OnRecordDelete("follows").BindFunc(func(e *core.RecordEvent) error {
		if fired.CompareAndSwap(false, true) {
			followDone = raceStart(func() error {
				return ProcessFollowActivity(app, remote, f2)
			})
			followDone.wait(raceHookWait)
		}
		return e.Next()
	})

	if err := ProcessUndoActivity(app, remote, lifecycleUndo(raceUndoIRI, f1)); err != nil {
		t.Fatalf("Undo F1: %v", err)
	}
	if !fired.Load() {
		t.Fatalf("delete hook never fired; the interleaving was not injected")
	}
	if !followDone.wait(raceGoroutineTO) {
		t.Fatalf("injected Follow did not finish within %s", raceGoroutineTO)
	}
	if followDone.err != nil {
		t.Fatalf("injected Follow: %v", followDone.err)
	}
	raceAssertSinglePendingF2(t, app, remote, local)
}

// Stress: many independent pairs, each Undo(F1)/Follow(F2) released together by
// a start barrier. Every pair must converge on one pending F2 row.
func TestLifecycleConcurrentRetryStress(t *testing.T) {
	const pairs = 20
	app := newLifecycleTestApp(t)
	local := createTestActor(t, app, lifecycleLocalInstance, "instance", true)

	type pair struct {
		remote *core.Record
		f1, f2 pub.Activity
		undo   pub.Activity
	}
	all := make([]pair, 0, pairs)
	for i := 0; i < pairs; i++ {
		host := fmt.Sprintf("https://peer%d.example.com", i)
		remote := createTestActor(t, app, host+"/api/v1/activitypub/instance", "instance", false)
		f1 := lifecycleFollow(host+"/api/v1/activitypub/activity/f1", remote.GetString("iri"), local.GetString("iri"))
		f2 := lifecycleFollow(host+"/api/v1/activitypub/activity/f2", remote.GetString("iri"), local.GetString("iri"))
		if err := ProcessFollowActivity(app, remote, f1); err != nil {
			t.Fatalf("F1 %d: %v", i, err)
		}
		lifecycleSetStatus(t, app, lifecycleRow(t, app, remote.Id, local.Id), "rejected")
		all = append(all, pair{
			remote: remote, f1: f1, f2: f2,
			undo: lifecycleUndo(host+"/api/v1/activitypub/activity/u1", f1),
		})
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	followErrs := make([]error, pairs)
	undoErrs := make([]error, pairs)
	for i := range all {
		p := all[i]
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			<-start
			undoErrs[i] = ProcessUndoActivity(app, p.remote, p.undo)
		}(i)
		go func(i int) {
			defer wg.Done()
			<-start
			followErrs[i] = ProcessFollowActivity(app, p.remote, p.f2)
		}(i)
	}
	close(start)

	finished := make(chan struct{})
	go func() { wg.Wait(); close(finished) }()
	select {
	case <-finished:
	case <-time.After(60 * time.Second):
		t.Fatalf("stress goroutines did not finish in time")
	}

	failed := 0
	for i, p := range all {
		if followErrs[i] != nil {
			t.Errorf("pair %d: Follow F2: %v", i, followErrs[i])
			failed++
			continue
		}
		if undoErrs[i] != nil {
			t.Logf("pair %d: Undo F1 returned %v", i, undoErrs[i])
		}
		row := lifecycleRow(t, app, p.remote.Id, local.Id)
		if row == nil || row.GetString("status") != "pending" || row.GetString("activity_iri") != p.f2.GetID().String() {
			t.Errorf("pair %d did not converge on one pending F2 row, got %v", i, row)
			failed++
		}
	}
	if n := lifecycleCountRows(t, app); n != pairs {
		t.Errorf("want %d follows rows, got %d", pairs, n)
	}
	t.Logf("stress: %d of %d pairs failed to converge", failed, pairs)
}
