package routes

import (
	"context"
	"sync"

	"github.com/pocketbase/pocketbase/core"
	"pocketbase/util"
)

type remoteListVisibilityKey struct{}
type remoteTrailVisibility struct{ iri, author string }

// Refresh only the visibility of known private copies at their own origin.
// The list's copy cannot grant access to another host's trail.
func prepareRemoteListVisibility(app core.App, ctx context.Context, data map[string]any, origin string) context.Context {
	expand, _ := data["expand"].(map[string]any)
	trails, _ := expand["trails"].([]any)
	jobs := []*core.Record{}
	seen := map[string]bool{}
	for _, value := range trails {
		raw, _ := value.(map[string]any)
		iri, _ := raw["iri"].(string)
		if seen[iri] || remoteSyncIRIMatchesOrigin(iri, origin) || util.IsLocalIRI(iri) {
			continue
		}
		seen[iri] = true
		trail, err := app.FindFirstRecordByData("trails", "iri", iri)
		if err != nil || trail.GetBool("public") || remoteRecordHasLocalAuthor(app, trail) || util.RecordHasLocalIdentity(app, trail) || util.IRIHasRecordIdentity(iri, "trails", trail.Id) {
			continue
		}
		if _, err := canonicalRecordURL(iri, "trail"); err != nil {
			continue
		}
		jobs = append(jobs, trail)
		if len(jobs) == canonicalRemoteRecordLimit {
			break
		}
	}
	confirmed := map[string]remoteTrailVisibility{}
	fetchCtx, cancel := context.WithTimeout(withCanonicalRemoteBodyBudget(ctx), canonicalRemoteFetchBudget)
	defer cancel()
	client := *newRemoteSyncHTTPClient()
	client.CheckRedirect = canonicalRemoteRedirect
	work := make(chan *core.Record, len(jobs))
	for _, trail := range jobs {
		work <- trail
	}
	close(work)
	var lock sync.Mutex
	var wait sync.WaitGroup
	for worker := 0; worker < canonicalRemoteFetchParallelism && worker < len(jobs); worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for trail := range work {
				if fetchCtx.Err() != nil || canonicalRemoteBodyBudgetExhausted(fetchCtx) {
					return
				}
				iri := trail.GetString("iri")
				remoteURL, _ := canonicalRecordURL(iri, "trail")
				query := remoteURL.Query()
				query.Set("expand", "author")
				query.Set("fields", "id,iri,public,expand.author.iri")
				remoteURL.RawQuery = query.Encode()
				probeCtx, stop := context.WithTimeout(fetchCtx, canonicalRemoteFetchTimeout)
				response, err := fetchRemoteDocument(probeCtx, &client, remoteURL, "trail", canonicalRemoteBodyLimit)
				stop()
				if err != nil || response["public"] != true {
					continue
				}
				responseIRI, _ := response["iri"].(string)
				if !sameRemoteSyncObjectIRI(iri, responseIRI) && !(responseIRI == "" && remoteResponseIDMatchesObjectIRI("trails", iri, response)) {
					continue
				}
				expand, _ := response["expand"].(map[string]any)
				author, _ := expand["author"].(map[string]any)
				authorIRI, _ := author["iri"].(string)
				stored, err := app.FindRecordById("activitypub_actors", trail.GetString("author"))
				if err != nil || stored.GetBool("is_local") || stored.GetString("iri") != authorIRI || !remoteSyncIRIMatchesOrigin(authorIRI, iri) {
					continue
				}
				lock.Lock()
				confirmed[trail.Id] = remoteTrailVisibility{iri: iri, author: stored.Id}
				lock.Unlock()
			}
		}()
	}
	wait.Wait()
	return context.WithValue(ctx, remoteListVisibilityKey{}, confirmed)
}

func applyRemoteListVisibility(app core.App, ctx context.Context, trail *core.Record) error {
	confirmed, _ := ctx.Value(remoteListVisibilityKey{}).(map[string]remoteTrailVisibility)
	proof, found := confirmed[trail.Id]
	if !found || proof.iri != trail.GetString("iri") || proof.author != trail.GetString("author") || remoteRecordHasLocalAuthor(app, trail) || util.RecordHasLocalIdentity(app, trail) || util.IRIHasRecordIdentity(proof.iri, "trails", trail.Id) {
		return nil
	}
	trail.Set("public", true)
	return app.Save(trail)
}
