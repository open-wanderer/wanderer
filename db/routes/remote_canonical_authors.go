package routes

import (
	"context"
	"errors"
	"sync"
	"time"

	"pocketbase/federation"
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/core"
)

// Resolve each confirmed author once, after the object responses have arrived.
// Slow object responses do not consume the author's independent fetch budget.
func resolveCanonicalRemoteAuthors(app core.App, ctx context.Context, references []string) map[string]*core.Record {
	return resolveRemoteAuthorsWithin(app, ctx, references, canonicalRemoteFetchTimeout, canonicalRemoteFetchBudget)
}

func resolveRemoteAuthorsWithin(app core.App, ctx context.Context, references []string, perActor, budget time.Duration) map[string]*core.Record {
	ctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()
	seen := map[string]struct{}{}
	work := make(chan string, len(references))
	for _, iri := range references {
		if _, duplicate := seen[iri]; duplicate || iri == "" {
			continue
		}
		seen[iri] = struct{}{}
		work <- iri
	}
	close(work)
	result := map[string]*core.Record{}
	var lock sync.Mutex
	var wait sync.WaitGroup
	for worker := 0; worker < canonicalRemoteFetchParallelism && worker < len(seen); worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for iri := range work {
				if ctx.Err() != nil {
					return
				}
				actorCtx, cancel := context.WithTimeout(ctx, perActor)
				actor, err := resolveRemoteActorForSync(app, actorCtx, iri)
				cancel()
				if errors.Is(err, util.ErrRateLimited) || errors.Is(err, errRemoteSyncFetchQueue) {
					rememberRemoteAuthorRefusal(ctx, iri)
					continue
				}
				if err != nil || actor == nil || actor.GetBool("is_local") || actor.GetString("iri") != iri || !federation.CachedActorIdentityMatchesOrigin(actor) {
					continue
				}
				lock.Lock()
				result[iri] = actor
				lock.Unlock()
			}
		}()
	}
	wait.Wait()
	return result
}
