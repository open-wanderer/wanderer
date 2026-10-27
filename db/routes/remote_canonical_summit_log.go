package routes

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sync"
	"time"

	"pocketbase/util"

	"github.com/pocketbase/pocketbase/core"
)

type canonicalRemoteSummitLog struct {
	data                     map[string]any
	remoteID, origin, author string
	authorIRI                string
	removed                  bool
	cacheKey, updatedHint    string
}

// A trail may include a contributor's entry hosted elsewhere. Its mutable
// fields must come from that origin, not the trail author's cached copy.
// Fetch outside the transaction and bound this optional enrichment in time.
const canonicalRemoteFetchParallelism = 4

var canonicalRemoteFetchTimeout = 2 * time.Second

const canonicalRemoteFetchBudget = 10 * time.Second

type canonicalLogRequest struct {
	iri, updated, cacheKey string
	url                    *url.URL
	storedAuthor           string
	parentPublic           bool
}

func fetchCanonicalRemoteSummitLogs(app core.App, ctx context.Context, trailData map[string]any, trail *core.Record, origin string) map[string]canonicalRemoteSummitLog {
	result := map[string]canonicalRemoteSummitLog{}
	expand, _ := trailData["expand"].(map[string]any)
	logs, _ := expand["summit_logs_via_trail"].([]any)
	fetchCtx, cancel := context.WithTimeout(withCanonicalRemoteBodyBudget(ctx), canonicalRemoteFetchBudget)
	defer cancel()
	client := *newRemoteSyncHTTPClient()
	client.CheckRedirect = canonicalRemoteRedirect
	seen := map[string]struct{}{}
	priorities := [2][]canonicalLogRequest{}
	parentPublic := trailData["public"] == true
	for _, entry := range logs {
		if fetchCtx.Err() != nil {
			break
		}
		raw, _ := entry.(map[string]any)
		iri, _ := raw["iri"].(string)
		if iri == "" || util.IsLocalIRI(iri) || remoteSyncIRIMatchesOrigin(iri, origin) {
			continue
		}
		parsed, err := canonicalSummitLogURL(iri)
		if err != nil {
			continue
		}
		if _, duplicate := seen[iri]; duplicate {
			continue
		}
		seen[iri] = struct{}{}
		existing, _ := app.FindFirstRecordByData("summit_logs", "iri", iri)
		storedAuthor, storedUpdated := "", ""
		if existing != nil {
			if existing.GetString("trail") != trail.Id || util.RecordHasLocalIdentity(app, existing) {
				continue
			}
			actor, err := app.FindRecordById("activitypub_actors", existing.GetString("author"))
			if err != nil || actor.GetBool("is_local") || (actor.Id != trail.GetString("author") && !remoteSyncIRIMatchesOrigin(actor.GetString("iri"), iri)) {
				continue
			}
			storedAuthor, storedUpdated = actor.Id, existing.GetString("updated")
		}
		hint := canonicalLogUpdatedHint(raw)
		key := canonicalLogCacheKey(app, trail, iri)
		previousHint, knownHint := canonicalRemoteLogCache.previousHint(key)
		if canonicalRemoteLogCache.skip(key, hint, storedUpdated, existing != nil) {
			continue
		}
		priority := 1
		if existing == nil || knownHint && previousHint != hint {
			priority = 0
		}
		priorities[priority] = append(priorities[priority], canonicalLogRequest{iri: iri, updated: hint, cacheKey: key, url: parsed, storedAuthor: storedAuthor, parentPublic: parentPublic})
	}
	jobs := selectCanonicalLogRequests(canonicalLogCacheKey(app, trail, "refresh-window"), priorities)
	work := make(chan canonicalLogRequest, len(jobs))
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
				value, accepted := fetchCanonicalRemoteSummitLog(app, fetchCtx, &client, trail, job)
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

func fetchCanonicalRemoteSummitLog(app core.App, ctx context.Context, client *http.Client, trail *core.Record, job canonicalLogRequest) (canonicalRemoteSummitLog, bool) {
	ctx, cancel := context.WithTimeout(ctx, canonicalRemoteFetchTimeout)
	defer cancel()
	job.url.RawQuery = url.Values{"expand": {"author,trail"}}.Encode()
	data, err := fetchRemoteDocument(ctx, client, job.url, "summit log", canonicalRemoteBodyLimit)
	if errors.Is(err, util.ErrRateLimited) || errors.Is(err, errRemoteSyncFetchQueue) {
		return canonicalRemoteSummitLog{}, false
	}
	if err != nil {
		canonicalRemoteLogCache.failure(job.cacheKey, job.updated)
		// A denied view can look like a missing record. A 404 confirms removal
		// only for a public trail in the fresh document; 410 is always final.
		var status *remoteFetchStatusError
		final := errors.As(err, &status) && (status.StatusCode == http.StatusGone || job.parentPublic)
		if errors.Is(err, errRemoteGone) && final && job.storedAuthor != "" {
			return canonicalRemoteSummitLog{removed: true, author: job.storedAuthor, cacheKey: job.cacheKey, updatedHint: job.updated}, true
		}
		return canonicalRemoteSummitLog{}, false
	}
	remoteID, _ := data["id"].(string)
	objectIRI, _ := data["iri"].(string)
	if remoteID == "" || job.url.Path != "/api/v1/summit-log/"+remoteID || !(sameRemoteSyncObjectIRI(job.iri, objectIRI) || objectIRI == "" && remoteResponseIDMatchesObjectIRI("summit_logs", job.iri, data)) {
		canonicalRemoteLogCache.failure(job.cacheKey, job.updated)
		return canonicalRemoteSummitLog{}, false
	}
	expand, _ := data["expand"].(map[string]any)
	parent, _ := expand["trail"].(map[string]any)
	parentIRI, _ := parent["iri"].(string)
	if !sameRemoteSyncObjectIRI(trail.GetString("iri"), parentIRI) {
		canonicalRemoteLogCache.failure(job.cacheKey, job.updated)
		return canonicalRemoteSummitLog{}, false
	}
	authorMap, _ := expand["author"].(map[string]any)
	authorIRI, _ := authorMap["iri"].(string)
	if !remoteSyncIRIMatchesOrigin(authorIRI, job.iri) {
		canonicalRemoteLogCache.failure(job.cacheKey, job.updated)
		return canonicalRemoteSummitLog{}, false
	}

	return canonicalRemoteSummitLog{data: canonicalSummitLogData(data), remoteID: remoteID, origin: job.url.Scheme + "://" + job.url.Host, authorIRI: authorIRI, cacheKey: job.cacheKey, updatedHint: job.updated}, true
}

func canonicalSummitLogURL(iri string) (*url.URL, error) {
	return canonicalRecordURL(iri, "summit-log")
}
