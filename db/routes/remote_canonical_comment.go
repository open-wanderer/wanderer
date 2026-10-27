package routes

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sync"

	"pocketbase/util"

	"github.com/pocketbase/pocketbase/core"
)

// A comment list of a remote trail can include comments by users of other
// instances: they reach the origin through the commenter's Create. Their
// content and author may only come from the comment's own host, never from
// the origin's listing. Like canonical summit logs, they are fetched outside
// the transaction, in parallel and bounded in time, and a page refresh never
// fetches more than one origin page worth of them.

type canonicalCommentReference struct {
	iri, updated string
	omitted      bool
}

type canonicalRemoteComment struct {
	data                  map[string]any
	author, authorIRI     string
	cacheKey, updatedHint string
	removed               bool
}

type canonicalCommentRequest struct {
	iri, updated, cacheKey string
	url                    *url.URL
	activityPubNote        bool
	storedAuthor           string
	omitted                bool
}

func fetchCanonicalRemoteComments(app core.App, ctx context.Context, base *http.Client, trail *core.Record, references []canonicalCommentReference) map[string]canonicalRemoteComment {
	result := map[string]canonicalRemoteComment{}
	if len(references) == 0 {
		return result
	}
	fetchCtx, cancel := context.WithTimeout(withCanonicalRemoteBodyBudget(ctx), canonicalRemoteFetchBudget)
	defer cancel()
	client := *base
	client.CheckRedirect = canonicalRemoteRedirect
	seen := map[string]struct{}{}
	priorities := [3][]canonicalLogRequest{}
	requests := map[string]canonicalCommentRequest{}
	for _, reference := range references {
		if fetchCtx.Err() != nil {
			break
		}
		iri := reference.iri
		if _, duplicate := seen[iri]; duplicate || util.IsLocalIRI(iri) {
			continue
		}
		seen[iri] = struct{}{}
		parsed, err := canonicalRecordURL(iri, "comment")
		activityPubNote := err != nil
		if activityPubNote {
			parsed, err = canonicalAPNoteURL(iri)
		}
		if err != nil {
			continue
		}
		existing, _ := app.FindFirstRecordByData("comments", "iri", iri)
		storedUpdated, storedAuthor := "", ""
		if existing != nil {
			// Never move or take over a stored comment: it must already be a
			// copy on this trail, by a remote author of the comment's host.
			if existing.GetString("trail") != trail.Id || util.RecordHasLocalIdentity(app, existing) || util.IRIHasRecordIdentity(iri, "comments", existing.Id) {
				continue
			}
			actor, err := app.FindRecordById("activitypub_actors", existing.GetString("author"))
			if err != nil || actor.GetBool("is_local") || !remoteSyncIRIMatchesOrigin(actor.GetString("iri"), iri) {
				continue
			}
			storedUpdated = existing.GetString("updated")
			storedAuthor = actor.Id
		}
		key := canonicalLogCacheKey(app, trail, iri)
		previousHint, knownHint := canonicalRemoteLogCache.previousHint(key)
		if canonicalRemoteLogCache.skip(key, reference.updated, storedUpdated, existing != nil) {
			continue
		}
		priority := 2
		if reference.omitted {
			priority = 1
		} else if existing == nil || knownHint && previousHint != reference.updated {
			priority = 0
		}
		priorities[priority] = append(priorities[priority], canonicalLogRequest{iri: iri})
		requests[iri] = canonicalCommentRequest{iri: iri, updated: reference.updated, cacheKey: key, url: parsed, activityPubNote: activityPubNote, storedAuthor: storedAuthor, omitted: reference.omitted}
	}
	jobs := []canonicalCommentRequest{}
	for _, selected := range selectCanonicalRemoteRequests(canonicalLogCacheKey(app, trail, "comment-refresh-window"), priorities[:]) {
		jobs = append(jobs, requests[selected.iri])
	}
	work := make(chan canonicalCommentRequest, len(jobs))
	for _, job := range jobs {
		work <- job
	}
	close(work)
	var wait sync.WaitGroup
	var resultLock sync.Mutex
	for worker := 0; worker < canonicalRemoteFetchParallelism && worker < len(jobs); worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for job := range work {
				if fetchCtx.Err() != nil || canonicalRemoteBodyBudgetExhausted(fetchCtx) {
					return
				}
				value, accepted := fetchCanonicalRemoteComment(app, fetchCtx, &client, trail, job)
				if !accepted {
					continue
				}
				resultLock.Lock()
				result[job.iri] = value
				resultLock.Unlock()
			}
		}()
	}
	wait.Wait()
	authorIRIs := make([]string, 0, len(result))
	for _, value := range result {
		if !value.removed {
			authorIRIs = append(authorIRIs, value.authorIRI)
		}
	}
	authorCtx, refusals := withRemoteAuthorRefusals(ctx)
	authors := resolveCanonicalRemoteAuthors(app, authorCtx, authorIRIs)
	for iri, value := range result {
		if value.removed {
			continue
		}
		if author := authors[value.authorIRI]; author != nil {
			value.author = author.Id
			result[iri] = value
		} else {
			if !refusals.refused(value.authorIRI) {
				canonicalRemoteLogCache.failure(value.cacheKey, value.updatedHint)
			}
			delete(result, iri)
		}
	}
	return result
}

// A response is accepted only when the comment's host confirms the exact
// comment, this trail as its parent and one of its own remote actors as the
// author. A missing stored comment is removed only under the captured owner.
func fetchCanonicalRemoteComment(app core.App, ctx context.Context, client *http.Client, trail *core.Record, job canonicalCommentRequest) (canonicalRemoteComment, bool) {
	ctx, cancel := context.WithTimeout(ctx, canonicalRemoteFetchTimeout)
	defer cancel()
	if job.activityPubNote {
		return fetchCanonicalAPNoteComment(app, ctx, client, trail, job)
	}
	target := *job.url
	target.RawQuery = url.Values{"expand": {"author,trail"}}.Encode()
	reject := func() (canonicalRemoteComment, bool) {
		canonicalRemoteLogCache.failure(job.cacheKey, job.updated)
		return canonicalRemoteComment{}, false
	}
	data, err := fetchRemoteDocument(ctx, client, &target, "comment", canonicalRemoteBodyLimit)
	if err != nil {
		if errors.Is(err, errRemoteGone) {
			return canonicalGoneComment(job)
		}
		if errors.Is(err, util.ErrRateLimited) {
			return canonicalRemoteComment{}, false
		}
		return reject()
	}
	remoteID, _ := data["id"].(string)
	objectIRI, _ := data["iri"].(string)
	if remoteID == "" || target.Path != "/api/v1/comment/"+remoteID || !(sameRemoteSyncObjectIRI(job.iri, objectIRI) || objectIRI == "" && remoteResponseIDMatchesObjectIRI("comments", job.iri, data)) {
		return reject()
	}
	expand, _ := data["expand"].(map[string]any)
	parent, _ := expand["trail"].(map[string]any)
	parentIRI, _ := parent["iri"].(string)
	if !sameRemoteSyncObjectIRI(trail.GetString("iri"), parentIRI) {
		return reject()
	}
	authorMap, _ := expand["author"].(map[string]any)
	authorIRI, _ := authorMap["iri"].(string)
	if util.IsLocalIRI(authorIRI) || !remoteSyncIRIMatchesOrigin(authorIRI, job.iri) {
		return reject()
	}
	text, ok := data["text"].(string)
	if !ok {
		return reject()
	}
	return canonicalRemoteComment{data: map[string]any{"text": text}, authorIRI: authorIRI, cacheKey: job.cacheKey, updatedHint: job.updated}, true
}

func canonicalGoneComment(job canonicalCommentRequest) (canonicalRemoteComment, bool) {
	canonicalRemoteLogCache.failure(job.cacheKey, job.updated)
	if job.storedAuthor == "" {
		return canonicalRemoteComment{}, false
	}
	return canonicalRemoteComment{removed: true, author: job.storedAuthor, cacheKey: job.cacheKey, updatedHint: job.updated}, true
}
