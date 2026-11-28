package util

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

// The embedded interface is deliberately nil: calling anything other than
// TxInfo panics. These tests construct no BaseApp, database, fixture or server.
type targetSyncTestApp struct {
	core.App
	info *core.TxAppInfo
}

func (app *targetSyncTestApp) TxInfo() *core.TxAppInfo { return app.info }

func assertTargetSyncInactive(t *testing.T, app core.App) {
	t.Helper()
	if DeferTargetSync(app, "unexpected", func(core.App) error {
		t.Error("inactive callback executed")
		return nil
	}) {
		t.Fatal("batch state leaked")
	}
}

func TestDeferredTargetSyncInactive(t *testing.T) {
	for _, info := range []*core.TxAppInfo{nil, new(core.TxAppInfo)} {
		app := &targetSyncTestApp{info: info}
		assertTargetSyncInactive(t, app)
	}

	app := &targetSyncTestApp{}
	want := errors.New("body error")
	called := false
	if err := WithDeferredTargetSync(app, func() error {
		called = true
		assertTargetSyncInactive(t, app)
		return want
	}); err != want || !called {
		t.Fatalf("without tx: called=%v error=%v", called, err)
	}
	assertTargetSyncInactive(t, app)
}

func TestDeferredTargetSyncNestedOrderAndFirstCallback(t *testing.T) {
	info := new(core.TxAppInfo)
	app := &targetSyncTestApp{info: info}
	sameTx := &targetSyncTestApp{info: info}
	var order []string
	register := func(from core.App, key, name string) {
		t.Helper()
		if !DeferTargetSync(from, key, func(got core.App) error {
			if got != from {
				t.Error("callback did not receive its registering transaction app")
			}
			order = append(order, name)
			return nil
		}) {
			t.Fatal("active scope not recognized")
		}
	}
	innerErr := errors.New("caught inner error")
	err := WithDeferredTargetSync(app, func() error {
		register(app, "trails/a", "first")
		err := WithDeferredTargetSync(sameTx, func() error {
			register(sameTx, "trails/a", "duplicate")
			register(sameTx, "summit_logs/a", "second")
			return WithDeferredTargetSync(app, func() error {
				register(app, "trails/b", "third")
				return innerErr
			})
		})
		if err != innerErr || len(order) != 0 {
			t.Fatalf("inner return: error=%v callbacks=%v", err, order)
		}
		register(app, "summit_logs/a", "another duplicate")
		return nil
	})
	if err != nil || !reflect.DeepEqual(order, []string{"first", "second", "third"}) {
		t.Fatalf("outer return: error=%v callbacks=%v", err, order)
	}
	assertTargetSyncInactive(t, app)
}

func TestDeferredTargetSyncErrorPrecedenceAndFlush(t *testing.T) {
	bodyErr := errors.New("partial write failed")
	lookupErr := errors.New("fresh target lookup failed")
	for _, tc := range []struct {
		name      string
		body, cb  error
		wantError error
	}{
		{"success", nil, nil, nil},
		{"lookup", nil, lookupErr, lookupErr},
		{"body", bodyErr, nil, bodyErr},
		{"body before lookup", bodyErr, lookupErr, bodyErr},
		{"cancel", context.Canceled, lookupErr, context.Canceled},
		{"deadline", context.DeadlineExceeded, nil, context.DeadlineExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app := &targetSyncTestApp{info: new(core.TxAppInfo)}
			var order []int
			err := WithDeferredTargetSync(app, func() error {
				for i := range 3 {
					DeferTargetSync(app, fmt.Sprint(i), func(core.App) error {
						order = append(order, i)
						if i == 0 {
							return tc.cb
						}
						if tc.cb != nil {
							return errors.New("later lookup error")
						}
						return nil
					})
				}
				return tc.body
			})
			if err != tc.wantError || !reflect.DeepEqual(order, []int{0, 1, 2}) {
				t.Fatalf("error=%v want=%v callbacks=%v", err, tc.wantError, order)
			}
			assertTargetSyncInactive(t, app)
		})
	}
}

func TestDeferredTargetSyncPanicCleanup(t *testing.T) {
	for _, callbackPanic := range []bool{false, true} {
		t.Run(fmt.Sprint(callbackPanic), func(t *testing.T) {
			app := &targetSyncTestApp{info: new(core.TxAppInfo)}
			sentinel := new(int)
			called := 0
			func() {
				defer func() {
					if got := recover(); got != sentinel {
						t.Errorf("panic=%v, want original panic %p", got, sentinel)
					}
				}()
				_ = WithDeferredTargetSync(app, func() error {
					DeferTargetSync(app, "a", func(core.App) error {
						called++
						panic(sentinel)
					})
					DeferTargetSync(app, "b", func(core.App) error {
						t.Error("flushed after panic")
						return nil
					})
					if !callbackPanic {
						panic(sentinel)
					}
					return nil
				})
				t.Error("panic was swallowed")
			}()
			wantCalls := 0
			if callbackPanic {
				wantCalls = 1
			}
			if called != wantCalls {
				t.Fatalf("callbacks=%d, want %d", called, wantCalls)
			}
			assertTargetSyncInactive(t, app)
			if err := WithDeferredTargetSync(app, func() error {
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			assertTargetSyncInactive(t, app)
		})
	}
}

func TestDeferredTargetSyncRecoveredInnerPanicDiscardsOldScope(t *testing.T) {
	app := &targetSyncTestApp{info: new(core.TxAppInfo)}
	var order []string
	err := WithDeferredTargetSync(app, func() error {
		DeferTargetSync(app, "old", func(core.App) error {
			t.Error("discarded outer callback executed")
			return nil
		})
		func() {
			defer func() { _ = recover() }()
			_ = WithDeferredTargetSync(app, func() error {
				DeferTargetSync(app, "inner", func(core.App) error {
					t.Error("discarded inner callback executed")
					return nil
				})
				panic("inner")
			})
		}()
		assertTargetSyncInactive(t, app)
		return WithDeferredTargetSync(app, func() error {
			DeferTargetSync(app, "old", func(core.App) error {
				order = append(order, "new")
				return nil
			})
			return nil
		})
	})
	if err != nil || !reflect.DeepEqual(order, []string{"new"}) {
		t.Fatalf("error=%v callbacks=%v", err, order)
	}
	assertTargetSyncInactive(t, app)
}

func TestDeferredTargetSyncReentryDuringFlush(t *testing.T) {
	app := &targetSyncTestApp{info: new(core.TxAppInfo)}
	var order []string
	err := WithDeferredTargetSync(app, func() error {
		DeferTargetSync(app, "a", func(got core.App) error {
			order = append(order, "first start")
			assertTargetSyncInactive(t, got)
			if err := WithDeferredTargetSync(got, func() error {
				DeferTargetSync(got, "a", func(core.App) error {
					order = append(order, "nested")
					return nil
				})
				return nil
			}); err != nil {
				return err
			}
			order = append(order, "first end")
			return nil
		})
		DeferTargetSync(app, "b", func(core.App) error {
			order = append(order, "last")
			return nil
		})
		return nil
	})
	want := []string{"first start", "nested", "first end", "last"}
	if err != nil || !reflect.DeepEqual(order, want) {
		t.Fatalf("error=%v callbacks=%v", err, order)
	}
	assertTargetSyncInactive(t, app)
}

func TestDeferredTargetSyncBulkFinalSnapshot(t *testing.T) {
	app := &targetSyncTestApp{info: new(core.TxAppInfo)}
	state := map[string]int{"trails/a": 0, "summit_logs/b": 0}
	snapshots := map[string][]int{}
	// Model both an add pass and a later removal pass on the same tx. Each
	// scope must flush once per target, never once per link or across scopes.
	for _, delta := range []int{1, -1} {
		before := len(snapshots["trails/a"])
		err := WithDeferredTargetSync(app, func() error {
			for range 1500 {
				for _, key := range []string{"trails/a", "summit_logs/b"} {
					state[key] += delta
					if !DeferTargetSync(app, key, func(core.App) error {
						snapshots[key] = append(snapshots[key], state[key])
						return nil
					}) {
						t.Fatal("bulk event was not deferred")
					}
				}
			}
			if len(snapshots["trails/a"]) != before {
				t.Fatal("sync ran before the operation's final state")
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		assertTargetSyncInactive(t, app)
	}
	for key, got := range snapshots {
		if !reflect.DeepEqual(got, []int{1500, 0}) {
			t.Errorf("%s snapshots=%v, want one final snapshot per scope", key, got)
		}
	}
}

func TestDeferredTargetSyncHandoverFlushesBeforeCascade(t *testing.T) {
	app := &targetSyncTestApp{info: new(core.TxAppInfo)}
	targets := []string{"trails/owner", "trails/foreign", "summit_logs/owner", "summit_logs/foreign"}
	events, calls, snapshots := map[string]int{}, map[string]int{}, map[string]int{}
	var order []string
	actorAlive, nextCalled := true, false
	// Model HandOverActorAssets followed by the delete hook's e.Next, not
	// a scope per asset or around the cascade itself.
	deleteHook := func() error {
		if err := WithDeferredTargetSync(app, func() error {
			for range 1500 {
				for _, key := range targets {
					// A matching keeper can cause both Save(existing) and
					// Delete(link), or Save(link), for the same target.
					for range 2 {
						events[key]++
						if !DeferTargetSync(app, key, func(core.App) error {
							if !actorAlive || nextCalled {
								t.Error("handover sync ran after the delete cascade")
							}
							calls[key]++
							snapshots[key] = events[key]
							order = append(order, key)
							return nil
						}) {
							return errors.New("handover scope missing")
						}
					}
				}
			}
			if len(order) != 0 {
				return errors.New("handover flushed per asset rather than per operation")
			}
			return nil
		}); err != nil {
			return err
		}
		// mock e.Next: all foreign and owned targets are final while the
		// actor/target authors still exist; only now may deletion continue.
		if !reflect.DeepEqual(order, targets) {
			return fmt.Errorf("sync order before e.Next=%v, want %v", order, targets)
		}
		for _, key := range targets {
			if calls[key] != 1 || snapshots[key] != 3000 {
				return fmt.Errorf("%s: callbacks=%d snapshot=%d", key, calls[key], snapshots[key])
			}
		}
		nextCalled, actorAlive = true, false
		assertTargetSyncInactive(t, app)
		return nil
	}
	if err := deleteHook(); err != nil {
		t.Fatal(err)
	}
	if !nextCalled || actorAlive {
		t.Fatal("delete hook did not continue after the flush")
	}
}

func TestDeferredTargetSyncConcurrentTransactions(t *testing.T) {
	const workers = 16
	ready := make(chan struct{}, workers)
	release := make(chan struct{})
	results := make(chan error, workers)
	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			app := &targetSyncTestApp{info: new(core.TxAppInfo)}
			snapshot, events, calls := 0, 0, 0
			err := WithDeferredTargetSync(app, func() error {
				for range 1500 {
					events++
					if !DeferTargetSync(app, "same key in every tx", func(got core.App) error {
						if got != app {
							return errors.New("cross-transaction callback app")
						}
						calls++
						snapshot = events
						return nil
					}) {
						return errors.New("active transaction missing")
					}
				}
				ready <- struct{}{}
				<-release
				return nil
			})
			if err == nil && (calls != 1 || snapshot != 1500) {
				err = fmt.Errorf("callbacks=%d snapshot=%d", calls, snapshot)
			}
			if DeferTargetSync(app, "same key in every tx", nil) {
				err = errors.New("completed transaction scope leaked")
			}
			results <- err
		})
	}
	for range workers {
		<-ready
	}
	close(release)
	wg.Wait()
	for range workers {
		if err := <-results; err != nil {
			t.Error(err)
		}
	}
}

func TestDeferredTargetSyncConcurrentRegistrations(t *testing.T) {
	app := &targetSyncTestApp{info: new(core.TxAppInfo)}
	calls := make([]int, 8)
	err := WithDeferredTargetSync(app, func() error {
		var wg sync.WaitGroup
		for range 16 {
			wg.Go(func() {
				for i := range 1500 {
					key := i % len(calls)
					if !DeferTargetSync(app, fmt.Sprint(key), func(core.App) error {
						calls[key]++
						return nil
					}) {
						t.Error("active batch disappeared")
					}
				}
			})
		}
		wg.Wait()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for key, count := range calls {
		if count != 1 {
			t.Errorf("target %d: callbacks=%d", key, count)
		}
	}
	assertTargetSyncInactive(t, app)
}
