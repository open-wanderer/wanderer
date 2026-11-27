package routes

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"pocketbase/util"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type canonicalRemoteSummitLog struct {
	data                     map[string]any
	remoteID, origin, author string
	authorIRI                string
	removed                  bool
	omitted                  bool
	cacheKey, updatedHint    string
	storedSnapshot           canonicalLogAbsenceSnapshot
	absenceProbe             canonicalLogAbsenceProbe
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
	omitted                bool
	storedSnapshot         canonicalLogAbsenceSnapshot
}

type canonicalLogReference struct {
	data    map[string]any
	omitted bool
}

func canonicalLogReferences(app core.App, trail *core.Record, origin string, logs []any, authoritative bool) []canonicalLogReference {
	references := make([]canonicalLogReference, 0, len(logs))
	listed := map[string]struct{}{}
	for _, entry := range logs {
		raw, _ := entry.(map[string]any)
		id, _ := raw["id"].(string)
		iri, _ := raw["iri"].(string)
		if iri == "" {
			iri = fmt.Sprintf("%s/api/v1/summit-log/%s", origin, id)
		}
		listed[iri] = struct{}{}
		// A fresh listing is counterevidence even if the generic cache skips it.
		canonicalRemoteLogAbsenceCache.reset(canonicalLogCacheKey(app, trail, iri))
		references = append(references, canonicalLogReference{data: raw})
	}
	if !authoritative || trail.Id == "" {
		return references
	}
	// Keep author for the historical-local-identity guard, but avoid loading
	// text, photos and other unused fields. This post-fetch view is separate
	// from the pre-fetch cleanup candidate snapshot.
	var stored []*core.Record
	err := app.RecordQuery("summit_logs").Select("id", "iri", "author").Where(dbx.HashExp{"trail": trail.Id}).All(&stored)
	if err != nil {
		return references
	}
	for _, record := range stored {
		iri := record.GetString("iri")
		if _, exists := listed[iri]; exists || util.RecordHasLocalIdentity(app, record) || remoteSyncIRIMatchesOrigin(iri, origin) {
			canonicalRemoteLogAbsenceCache.reset(canonicalLogCacheKey(app, trail, iri))
			continue
		}
		references = append(references, canonicalLogReference{data: map[string]any{"iri": iri, "updated": "canonical-absence"}, omitted: true})
	}
	return references
}

func fetchCanonicalRemoteSummitLogs(app core.App, ctx context.Context, trailData map[string]any, trail *core.Record, origin string, complete ...bool) map[string]canonicalRemoteSummitLog {
	authoritative := remoteTrailChildIDs(app, trailData, origin, "summit_logs_via_trail", "summit_logs") != nil
	if len(complete) > 0 {
		authoritative = complete[0]
	}
	return fetchCanonicalRemoteSummitLogsWithin(app, ctx, trailData, trail, origin, authoritative, canonicalRemoteFetchBudget)
}

func fetchCanonicalRemoteSummitLogsWithin(app core.App, ctx context.Context, trailData map[string]any, trail *core.Record, origin string, authoritative bool, budget time.Duration) map[string]canonicalRemoteSummitLog {
	result := map[string]canonicalRemoteSummitLog{}
	expand, _ := trailData["expand"].(map[string]any)
	logs, _ := expand["summit_logs_via_trail"].([]any)
	fetchCtx, cancel := context.WithTimeout(withCanonicalRemoteBodyBudget(ctx), budget)
	defer cancel()
	client := *newRemoteSyncHTTPClient()
	client.CheckRedirect = canonicalRemoteRedirect
	seen := map[string]struct{}{}
	priorities := [3][]canonicalLogRequest{}
	parentPublic := trailData["public"] == true
	if !parentPublic || !authoritative {
		canonicalRemoteLogAbsenceCache.resetTrail(app, trail)
	}
	for _, reference := range canonicalLogReferences(app, trail, origin, logs, authoritative) {
		if fetchCtx.Err() != nil {
			break
		}
		raw := reference.data
		iri, _ := raw["iri"].(string)
		key := canonicalLogCacheKey(app, trail, iri)
		if iri == "" || util.IsLocalIRI(iri) || remoteSyncIRIMatchesOrigin(iri, origin) {
			canonicalRemoteLogAbsenceCache.reset(key)
			continue
		}
		parsed, err := canonicalSummitLogURL(iri)
		if err != nil {
			canonicalRemoteLogAbsenceCache.reset(key)
			continue
		}
		if _, duplicate := seen[iri]; duplicate {
			continue
		}
		seen[iri] = struct{}{}
		existing, _ := app.FindFirstRecordByData("summit_logs", "iri", iri)
		storedAuthor, storedUpdated := "", ""
		var storedSnapshot canonicalLogAbsenceSnapshot
		if existing != nil {
			storedSnapshot = canonicalLogAbsenceSnapshotOf(existing, trail)
			canonicalRemoteLogAbsenceCache.resetIfChanged(key, storedSnapshot)
			if existing.GetString("trail") != trail.Id || util.RecordHasLocalIdentity(app, existing) {
				canonicalRemoteLogAbsenceCache.reset(key)
				continue
			}
			actor, err := app.FindRecordById("activitypub_actors", existing.GetString("author"))
			if err != nil || actor.GetBool("is_local") || (actor.Id != trail.GetString("author") && !remoteSyncIRIMatchesOrigin(actor.GetString("iri"), iri)) {
				canonicalRemoteLogAbsenceCache.reset(key)
				continue
			}
			storedAuthor, storedUpdated = actor.Id, existing.GetString("updated")
		} else {
			canonicalRemoteLogAbsenceCache.reset(key)
			if reference.omitted {
				continue
			}
		}
		hint := canonicalLogUpdatedHint(raw)
		previousHint, knownHint := canonicalRemoteLogCache.previousHint(key)
		if canonicalRemoteLogCache.skip(key, hint, storedUpdated, existing != nil) {
			continue
		}
		priority := 2
		if reference.omitted {
			priority = 1
		} else if existing == nil || knownHint && previousHint != hint {
			priority = 0
		}
		priorities[priority] = append(priorities[priority], canonicalLogRequest{iri: iri, updated: hint, cacheKey: key, url: parsed, storedAuthor: storedAuthor, parentPublic: parentPublic, omitted: reference.omitted, storedSnapshot: storedSnapshot})
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
				func() {
					defer recoverBackgroundSyncJob(app, "summit_logs", job.iri, func() {
						canonicalRemoteLogAbsenceCache.reset(job.cacheKey)
						canonicalRemoteLogCache.failure(job.cacheKey, job.updated)
					})
					value, accepted := fetchCanonicalRemoteSummitLog(app, fetchCtx, &client, trail, job)
					if !accepted {
						return
					}
					resultLock.Lock()
					result[job.iri] = value
					resultLock.Unlock()
				}()
			}
		}()
	}
	wait.Wait()
	authorIRIs := make([]string, 0, len(result))
	for _, value := range result {
		if !value.removed && !value.omitted {
			authorIRIs = append(authorIRIs, value.authorIRI)
		}
	}
	authorCtx, refusals := withRemoteAuthorRefusals(ctx)
	authors := resolveCanonicalRemoteAuthors(app, authorCtx, authorIRIs)
	for iri, value := range result {
		if value.removed || value.omitted {
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
	var probe canonicalLogAbsenceProbe
	if job.omitted && job.parentPublic && job.storedAuthor != "" {
		// Reserve only when a selected job actually attempts HTTP, not while
		// enumerating potentially thousands of stored omissions.
		probe = canonicalRemoteLogAbsenceCache.begin(job.cacheKey, job.storedSnapshot, time.Now())
	}
	data, err := fetchRemoteDocument(ctx, client, job.url, "summit log", canonicalRemoteBodyLimit)
	var status *remoteFetchStatusError
	notFound := errors.As(err, &status) && status.StatusCode == http.StatusNotFound
	if !notFound || !job.omitted || !job.parentPublic {
		canonicalRemoteLogAbsenceCache.reset(job.cacheKey)
	}
	if errors.Is(err, util.ErrRateLimited) || errors.Is(err, errRemoteSyncFetchQueue) {
		return canonicalRemoteSummitLog{}, false
	}
	if err != nil {
		canonicalRemoteLogCache.failure(job.cacheKey, job.updated)
		// A denied view can look missing. Omitted public-parent 404s need two
		// separated observations; private-parent 404s never remove. A 410 is final.
		final := status != nil && (status.StatusCode == http.StatusGone || job.parentPublic)
		if errors.Is(err, errRemoteGone) && final && job.storedAuthor != "" {
			confirmed := canonicalRemoteSummitLog{removed: true, omitted: job.omitted, author: job.storedAuthor, cacheKey: job.cacheKey, updatedHint: job.updated, storedSnapshot: job.storedSnapshot}
			if job.omitted && notFound {
				if !canonicalRemoteLogAbsenceCache.observe404(probe, time.Now()) {
					return canonicalRemoteSummitLog{}, false
				}
				confirmed.absenceProbe = probe
				// Application may never be reached (e.g. a newer trail or an earlier
				// batch failure). Do not negative-cache an unapplied candidate.
				canonicalRemoteLogCache.invalidate(job.cacheKey)
			}
			return confirmed, true
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
	if job.omitted {
		// This probe only confirms existence. Keep the stored author snapshot
		// for the transactional guard/cache, without enriching unused data.
		return canonicalRemoteSummitLog{omitted: true, author: job.storedAuthor, cacheKey: job.cacheKey, updatedHint: job.updated}, true
	}

	return canonicalRemoteSummitLog{data: canonicalSummitLogData(data), remoteID: remoteID, origin: job.url.Scheme + "://" + job.url.Host, authorIRI: authorIRI, cacheKey: job.cacheKey, updatedHint: job.updated}, true
}

func canonicalSummitLogURL(iri string) (*url.URL, error) {
	return canonicalRecordURL(iri, "summit-log")
}
