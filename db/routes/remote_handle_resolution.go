package routes

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"pocketbase/federation"
	"pocketbase/util"
)

type remoteHandleFailure struct {
	actorID string
	err     error
	until   time.Time
}

var remoteHandleFailures = struct {
	sync.Mutex
	entries map[string]remoteHandleFailure
}{entries: map[string]remoteHandleFailure{}}
var remoteHandleGetActor = federation.GetActorByHandle

func resolveRemoteHandleActor(e *core.RequestEvent, ctx context.Context, handle string) (*core.Record, error) {
	caller := "anonymous"
	if e.Auth != nil {
		caller = e.Auth.Id
	}
	digest := sha256.Sum256([]byte(e.App.DataDir() + "\x00" + caller + "\x00" + handle))
	key := fmt.Sprintf("%x", digest)
	remoteHandleFailures.Lock()
	failure, found := remoteHandleFailures.entries[key]
	if found && time.Now().After(failure.until) {
		delete(remoteHandleFailures.entries, key)
		found = false
	}
	remoteHandleFailures.Unlock()
	if found {
		var actor *core.Record
		if failure.actorID != "" {
			actor, _ = e.App.FindRecordById("activitypub_actors", failure.actorID)
		} else if username, domain := util.SplitHandle(handle); domain != "" {
			// A different request may have stored this actor and its copies.
			actor, _ = e.App.FindFirstRecordByFilter("activitypub_actors", "preferred_username={:username}&&domain={:domain}&&is_local=false", dbx.Params{"username": username, "domain": domain})
		}
		return actor, failure.err
	}
	actor, err := remoteHandleGetActor(e.App, ctx, handle, false)
	if err == nil || errors.Is(err, federation.ErrProfilePrivate) {
		return actor, err
	}
	if !errors.Is(err, federation.ErrActorNotFound) {
		err = fmt.Errorf("%w: %w", errRemoteUnavailable, err)
	}
	// A refused dial budget belongs to this caller and made no request.
	if ctx.Err() == nil && !errors.Is(err, util.ErrRateLimited) {
		cachedError := errRemoteUnavailable
		if errors.Is(err, federation.ErrActorNotFound) {
			cachedError = federation.ErrActorNotFound
		}
		failure = remoteHandleFailure{err: cachedError, until: time.Now().Add(30 * time.Second)}
		if actor != nil {
			failure.actorID = actor.Id
		}
		remoteHandleFailures.Lock()
		if len(remoteHandleFailures.entries) >= 512 {
			oldest := ""
			var until time.Time
			for candidate, value := range remoteHandleFailures.entries {
				if oldest == "" || value.until.Before(until) {
					oldest, until = candidate, value.until
				}
			}
			delete(remoteHandleFailures.entries, oldest)
		}
		remoteHandleFailures.entries[key] = failure
		remoteHandleFailures.Unlock()
	}
	return actor, err
}
