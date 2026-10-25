package routes

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"pocketbase/util"

	"github.com/pocketbase/pocketbase/core"
)

func TestKnownKeyReloadKeepsFetchResultWithItsRequest(t *testing.T) {
	w := newKnownKeyWorld(t)
	actor := w.sender(t, "sender", "remote.example", true)
	fetchErr := errors.New("actor refresh result")
	ctx, err := util.GetSafeActorContext(nil, w.recipient)
	if err != nil {
		t.Fatal(err)
	}
	started, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	getInboxActorByIRI = func(core.App, context.Context, string, bool) (*core.Record, error) {
		close(started)
		<-release
		return nil, fetchErr
	}
	results := make(chan error, 5)
	go func() { _, err := reloadKnownInboxActor(w.app, ctx, actor); results <- err }()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("actor refresh did not start")
	}
	for i := 1; i < cap(results); i++ {
		go func() { _, err := reloadKnownInboxActor(w.app, ctx, actor); results <- err }()
	}
	select {
	case err := <-results:
		t.Fatalf("refresh returned before the origin answered: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	once.Do(func() { close(release) })
	fetches := 0
	for i := 0; i < cap(results); i++ {
		select {
		case err := <-results:
			var reloadErr *knownInboxActorReloadError
			if !errors.Is(err, fetchErr) || !errors.As(err, &reloadErr) || reloadErr.claimRefused || reloadErr.waited == reloadErr.fetched {
				t.Fatalf("shared refresh error: %v", err)
			}
			if reloadErr.fetched {
				fetches++
			}
		case <-time.After(3 * time.Second):
			t.Fatal("waiting refresh did not finish")
		}
	}
	if fetches != 1 {
		t.Fatalf("requests allowed to use the fetched result: %d", fetches)
	}
	_, err = reloadKnownInboxActor(w.app, ctx, actor)
	var reloadErr *knownInboxActorReloadError
	if !errors.Is(err, util.ErrRateLimited) || !errors.As(err, &reloadErr) || !reloadErr.claimRefused || reloadErr.fetched || reloadErr.waited {
		t.Fatalf("later refresh claim: %v", err)
	}
}
