package routes

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"pocketbase/util"
)

// A list may name trails of another instance. Only that trail's own origin
// describes it; the list's copy is never used. Each sync asks those origins
// for a bounded, rotating set: references not stored here yet first, then
// placeholders that were never opened and are due for a refresh.
const remoteListReferenceLimit = canonicalRemoteRecordLimit

// A first view waits briefly for other origins, for their objects and then
// for unknown authors. Unavailable or budget-cut entries are retried after a
// delay; entries above the request limit wait for a regular refresh.
var remoteListReferenceFirstView = struct{ objects, authors time.Duration }{3 * time.Second, time.Second}

// Unavailable or budget-cut entries are retried after these delays, then
// wait for the list's regular refresh.
var remoteListReferenceRetries = []time.Duration{time.Minute, 4 * time.Minute}

const remoteListReferenceLeanFields = "id,iri,public,name,location,distance,elevation_gain,elevation_loss,duration,difficulty,date,completed,like_count,lat,lon,min_lat,max_lat,min_lon,max_lon,bounding_box_diagonal,expand.author.iri,expand.category.name,expand.subcategory.name,expand.tags.name"

// Description and line are optional: an answer that is too large with them
// is requested once more without.
const remoteListReferenceFields = remoteListReferenceLeanFields + ",description,polyline"

var remoteListReferenceScalars = []string{"name", "description", "location", "distance", "elevation_gain", "elevation_loss", "duration", "difficulty", "date", "completed", "like_count", "lat", "lon", "min_lat", "max_lat", "min_lon", "max_lon", "bounding_box_diagonal", "polyline"}

type remoteListReferencesKey struct{}

type remoteListReference struct {
	data              map[string]any
	iri               string
	authorIRI, author string
}

// The references of one sync, keyed by exact IRI; refreshed placeholders by id.
type remoteListReferenceState struct {
	confirmed map[string]remoteListReference
	refreshed map[string]remoteListReference
}

type remoteListReferenceAnswer int

const (
	remoteListReferenceLoaded remoteListReferenceAnswer = iota
	// The origin answered: not this object, not public, not acceptable.
	remoteListReferenceRefused
	// No answer of the origin: try again later.
	remoteListReferenceUnavailable
)

type remoteListReferenceResult struct {
	reference remoteListReference
	answer    remoteListReferenceAnswer
	// The host did not answer at all or is overloaded.
	silent   bool
	tooLarge bool
}

type remoteListReferenceFlight struct {
	done   chan struct{}
	result remoteListReferenceResult
}

type remoteListContinuation struct {
	due    time.Time
	misses int
	// A view started a sync that has not reached its reference preparation.
	started bool
}

var remoteListReferences = struct {
	sync.Mutex
	failed        map[string]time.Time
	flights       map[string]*remoteListReferenceFlight
	continuations map[string]remoteListContinuation
}{failed: map[string]time.Time{}, flights: map[string]*remoteListReferenceFlight{}, continuations: map[string]remoteListContinuation{}}

type remoteListReferenceJob struct {
	iri          string
	stored       *core.Record
	storedAuthor string
}

func prepareRemoteListReferences(app core.App, ctx context.Context, list *core.Record, data map[string]any, origin string) context.Context {
	state := remoteListReferenceState{confirmed: map[string]remoteListReference{}, refreshed: map[string]remoteListReference{}}
	expand, _ := data["expand"].(map[string]any)
	trails, _ := expand["trails"].([]any)
	objects := []string{}
	forms := []any{}
	seen := map[string]bool{}
	for _, value := range trails {
		raw, _ := value.(map[string]any)
		iri, _ := raw["iri"].(string)
		if iri == "" || remoteSyncIRIMatchesOrigin(iri, origin) || remoteListReferenceIsLocal(iri) || seen[iri] {
			continue
		}
		if _, err := canonicalRecordURL(iri, "trail"); err != nil {
			continue
		}
		seen[iri] = true
		objects = append(objects, iri)
		forms = append(forms, iri)
	}
	stored := storedRemoteListReferences(app, forms)
	unknown, refresh := []remoteListReferenceJob{}, []remoteListReferenceJob{}
	for _, iri := range objects {
		record := stored[iri]
		if record == nil {
			if !remoteListReferenceFailed(app, iri) {
				unknown = append(unknown, remoteListReferenceJob{iri: iri})
			}
			continue
		}
		if remoteListReferenceRefreshDue(record) && !remoteListReferenceFailed(app, iri) {
			refresh = append(refresh, remoteListReferenceJob{iri: iri, stored: record})
		}
	}
	listKey := app.DataDir() + "\x00list-references\x00" + list.GetString("iri")
	jobs := []remoteListReferenceJob{}
	if len(unknown) > 0 {
		take := min(len(unknown), remoteListReferenceLimit)
		start := canonicalRotationStart(listKey, len(unknown), take)
		for offset := range take {
			jobs = append(jobs, unknown[(start+offset)%len(unknown)])
		}
	}
	if capacity := remoteListReferenceLimit - len(jobs); capacity > 0 && len(refresh) > 0 {
		take := min(len(refresh), capacity)
		start := canonicalRotationStart(listKey+"\x00refresh", len(refresh), take)
		for offset := range take {
			job := refresh[(start+offset)%len(refresh)]
			if job.storedAuthor = remoteListReferenceStoredAuthor(app, job.stored); job.storedAuthor != "" {
				jobs = append(jobs, job)
			}
		}
	}
	firstView := !hasCompletedFullSync(list)
	if len(jobs) == 0 {
		rememberRemoteListContinuation(app, list.GetString("iri"), false)
		return context.WithValue(ctx, remoteListReferencesKey{}, state)
	}

	objectBudget, authorBudget := canonicalRemoteFetchBudget, canonicalRemoteFetchBudget
	if firstView {
		objectBudget, authorBudget = remoteListReferenceFirstView.objects, remoteListReferenceFirstView.authors
	}
	fetchCtx, cancel := context.WithTimeout(withCanonicalRemoteBodyBudget(ctx), objectBudget)
	defer cancel()
	client := *newRemoteSyncHTTPClient()
	defer client.CloseIdleConnections()
	client.CheckRedirect = canonicalRemoteRedirect
	work := make(chan int, len(jobs))
	for index := range jobs {
		work <- index
	}
	close(work)
	results := make([]*remoteListReferenceResult, len(jobs))
	// A host that does not answer is not asked again in this sync.
	silent := map[string]bool{}
	var lock sync.Mutex
	var wait sync.WaitGroup
	for worker := 0; worker < canonicalRemoteFetchParallelism && worker < len(jobs); worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for index := range work {
				if fetchCtx.Err() != nil || canonicalRemoteBodyBudgetExhausted(fetchCtx) {
					return
				}
				remoteURL, _ := url.Parse(jobs[index].iri)
				host := strings.ToLower(remoteURL.Host)
				lock.Lock()
				if silent[host] {
					results[index] = &remoteListReferenceResult{answer: remoteListReferenceUnavailable}
					lock.Unlock()
					continue
				}
				lock.Unlock()
				result := fetchRemoteListReference(app, fetchCtx, &client, jobs[index].iri)
				// A request this sync's own budget cut off is left for later.
				if result.answer == remoteListReferenceUnavailable && fetchCtx.Err() != nil {
					return
				}
				lock.Lock()
				results[index] = &result
				if result.silent {
					silent[host] = true
				}
				lock.Unlock()
			}
		}()
	}
	wait.Wait()

	authorIRIs := []string{}
	for index, job := range jobs {
		if result := results[index]; result != nil && result.answer == remoteListReferenceLoaded && job.stored == nil {
			authorIRIs = append(authorIRIs, result.reference.authorIRI)
		}
	}
	authors := map[string]*core.Record{}
	if len(authorIRIs) > 0 {
		authorCtx, cancelAuthors := context.WithTimeout(ctx, authorBudget)
		authors = resolveCanonicalRemoteAuthors(app, authorCtx, authorIRIs)
		cancelAuthors()
	}
	retry := false
	for index, job := range jobs {
		result := results[index]
		switch {
		case result == nil:
			retry = retry || job.stored == nil
		case job.stored != nil:
			if result.answer == remoteListReferenceLoaded && result.reference.authorIRI == job.storedAuthor {
				reference := result.reference
				reference.author = job.stored.GetString("author")
				state.refreshed[job.stored.Id] = reference
			}
		case result.answer != remoteListReferenceLoaded:
			retry = retry || result.answer == remoteListReferenceUnavailable
		case authors[result.reference.authorIRI] == nil:
			// An author that cannot be confirmed now is asked again later.
			retry = true
		default:
			reference := result.reference
			reference.author = authors[reference.authorIRI].Id
			state.confirmed[job.iri] = reference
		}
	}
	rememberRemoteListContinuation(app, list.GetString("iri"), retry)
	return context.WithValue(ctx, remoteListReferencesKey{}, state)
}

// Concurrent syncs naming the same reference share one request. Each caller
// receives its own copy of the answer.
func fetchRemoteListReference(app core.App, ctx context.Context, client *http.Client, iri string) remoteListReferenceResult {
	key := app.DataDir() + "\x00" + iri
	remoteListReferences.Lock()
	if flight := remoteListReferences.flights[key]; flight != nil {
		remoteListReferences.Unlock()
		select {
		case <-flight.done:
			result := flight.result
			result.reference = result.reference.clone()
			result.silent = false
			return result
		case <-ctx.Done():
			return remoteListReferenceResult{answer: remoteListReferenceUnavailable}
		}
	}
	flight := &remoteListReferenceFlight{done: make(chan struct{})}
	remoteListReferences.flights[key] = flight
	remoteListReferences.Unlock()
	defer func() {
		remoteListReferences.Lock()
		delete(remoteListReferences.flights, key)
		remoteListReferences.Unlock()
		close(flight.done)
	}()
	result := loadRemoteListReference(ctx, client, iri, remoteListReferenceFields)
	if result.tooLarge {
		result = loadRemoteListReference(ctx, client, iri, remoteListReferenceLeanFields)
	}
	// Only an answer of the origin is remembered, for the requested IRI.
	// Waiting here, deadlines, budgets, failed connections and overloaded
	// origins say nothing about the trail.
	if result.answer == remoteListReferenceRefused {
		rememberRemoteListReferenceFailure(app, iri)
	}
	flight.result = result
	result.reference = result.reference.clone()
	return result
}

func (reference remoteListReference) clone() remoteListReference {
	if reference.data == nil {
		return reference
	}
	data := make(map[string]any, len(reference.data))
	for key, value := range reference.data {
		data[key] = value
	}
	if expand, ok := reference.data["expand"].(map[string]any); ok {
		copied := make(map[string]any, len(expand))
		for key, value := range expand {
			if tags, ok := value.([]any); ok {
				value = slices.Clone(tags)
			}
			copied[key] = value
		}
		data["expand"] = copied
	}
	reference.data = data
	return reference
}

func loadRemoteListReference(ctx context.Context, client *http.Client, iri, fields string) remoteListReferenceResult {
	remoteURL, err := canonicalRecordURL(iri, "trail")
	if err != nil {
		return remoteListReferenceResult{answer: remoteListReferenceRefused}
	}
	remoteURL.RawQuery = url.Values{"expand": {"author,category,subcategory,tags"}, "fields": {fields}}.Encode()
	probeCtx, stop := context.WithTimeout(ctx, canonicalRemoteFetchTimeout)
	response, err := fetchRemoteDocument(probeCtx, client, remoteURL, "trail", canonicalRemoteBodyLimit)
	stop()
	if err != nil {
		var status *remoteFetchStatusError
		var transport *url.Error
		var syntax *json.SyntaxError
		var kind *json.UnmarshalTypeError
		switch {
		case errors.Is(err, util.ErrFederationBodyTooLarge):
			return remoteListReferenceResult{answer: remoteListReferenceRefused, tooLarge: true}
		case errors.As(err, &status) && status.StatusCode < http.StatusInternalServerError && status.StatusCode != http.StatusRequestTimeout && status.StatusCode != http.StatusTooManyRequests:
			return remoteListReferenceResult{answer: remoteListReferenceRefused}
		case errors.As(err, &status), errors.As(err, &transport) && !errors.Is(err, util.ErrRateLimited):
			return remoteListReferenceResult{answer: remoteListReferenceUnavailable, silent: true}
		case errors.As(err, &syntax), errors.As(err, &kind):
			// A complete answer that is not a trail document.
			return remoteListReferenceResult{answer: remoteListReferenceRefused}
		}
		return remoteListReferenceResult{answer: remoteListReferenceUnavailable}
	}
	canonical, _ := response["iri"].(string)
	if canonical == "" && remoteResponseIDMatchesObjectIRI("trails", iri, response) {
		canonical = iri
	}
	if iri != canonical || response["public"] != true {
		return remoteListReferenceResult{answer: remoteListReferenceRefused}
	}
	expand, _ := response["expand"].(map[string]any)
	author, _ := expand["author"].(map[string]any)
	authorIRI, _ := author["iri"].(string)
	if util.IsLocalIRI(authorIRI) || !remoteSyncIRIMatchesOrigin(authorIRI, canonical) {
		return remoteListReferenceResult{answer: remoteListReferenceRefused}
	}
	data := canonicalScalarFields(response, remoteListReferenceScalars...)
	data["public"] = true
	// Prepared tags are matched by this key.
	data["iri"] = canonical
	names := map[string]any{}
	for _, relation := range []string{"category", "subcategory"} {
		value, _ := expand[relation].(map[string]any)
		if name, ok := value["name"].(string); ok {
			names[relation] = map[string]any{"name": name}
		}
	}
	if values, ok := expand["tags"].([]any); ok {
		tags := []any{}
		for _, value := range values {
			tag, _ := value.(map[string]any)
			if name, ok := tag["name"].(string); ok {
				tags = append(tags, map[string]any{"name": name})
			}
		}
		names["tags"] = tags
	}
	data["expand"] = names
	return remoteListReferenceResult{reference: remoteListReference{data: data, iri: canonical, authorIRI: authorIRI}, answer: remoteListReferenceLoaded}
}

// One query for all entries; the partial iri index cannot serve these lookups.
func storedRemoteListReferences(app core.App, forms []any) map[string]*core.Record {
	stored := map[string]*core.Record{}
	if len(forms) == 0 {
		return stored
	}
	records := []*core.Record{}
	if err := app.RecordQuery("trails").Where(dbx.In("iri", forms...)).All(&records); err != nil {
		return stored
	}
	for _, record := range records {
		stored[record.GetString("iri")] = record
	}
	return stored
}

// Placeholders are refreshed from their origin until they are opened; a full
// copy refreshes itself.
func remoteListReferenceRefreshDue(trail *core.Record) bool {
	return trail.GetBool("public") && trail.GetBool("needs_full_sync") && !hasCompletedFullSync(trail) &&
		time.Since(trail.GetDateTime("updated").Time()) >= remoteSyncThreshold
}

// The author IRI the origin must confirm for a placeholder, or "".
func remoteListReferenceStoredAuthor(app core.App, trail *core.Record) string {
	actor, err := app.FindRecordById("activitypub_actors", trail.GetString("author"))
	if err != nil || actor.GetBool("is_local") || !remoteSyncIRIMatchesOrigin(actor.GetString("iri"), trail.GetString("iri")) {
		return ""
	}
	return actor.GetString("iri")
}

func remoteListReferenceStateOf(ctx context.Context) remoteListReferenceState {
	state, _ := ctx.Value(remoteListReferencesKey{}).(remoteListReferenceState)
	return state
}

// Never request this instance under another scheme or port.
func remoteListReferenceIsLocal(iri string) bool {
	if util.IsLocalIRI(iri) {
		return true
	}
	origin, err := url.Parse(os.Getenv("ORIGIN"))
	if err != nil || origin.Hostname() == "" {
		return false
	}
	parsed, err := url.Parse(iri)
	if err != nil {
		return true
	}
	originHost := strings.TrimSuffix(strings.ToLower(origin.Hostname()), ".")
	requestedHost := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	return requestedHost == originHost || requestedHost == strings.TrimPrefix(originHost, "www.")
}

func remoteListReferenceFailed(app core.App, iri string) bool {
	key := app.DataDir() + "\x00" + iri
	remoteListReferences.Lock()
	defer remoteListReferences.Unlock()
	until, found := remoteListReferences.failed[key]
	if found && time.Now().After(until) {
		delete(remoteListReferences.failed, key)
		return false
	}
	return found
}

func rememberRemoteListReferenceFailure(app core.App, iri string) {
	remoteListReferences.Lock()
	defer remoteListReferences.Unlock()
	now := time.Now()
	if len(remoteListReferences.failed) >= canonicalRemoteLogCacheLimit {
		oldest := ""
		var first time.Time
		for key, until := range remoteListReferences.failed {
			if now.After(until) {
				delete(remoteListReferences.failed, key)
			} else if oldest == "" || until.Before(first) {
				oldest, first = key, until
			}
		}
		if len(remoteListReferences.failed) >= canonicalRemoteLogCacheLimit {
			delete(remoteListReferences.failed, oldest)
		}
	}
	remoteListReferences.failed[app.DataDir()+"\x00"+iri] = now.Add(remoteSyncThreshold)
}

func rememberRemoteListContinuation(app core.App, iri string, retry bool) {
	key := app.DataDir() + "\x00" + iri
	remoteListReferences.Lock()
	defer remoteListReferences.Unlock()
	entry := remoteListReferences.continuations[key]
	if !retry || entry.misses >= len(remoteListReferenceRetries) {
		delete(remoteListReferences.continuations, key)
		return
	}
	entry.due = time.Now().Add(remoteListReferenceRetries[entry.misses])
	entry.misses++
	entry.started = false
	if _, known := remoteListReferences.continuations[key]; !known && len(remoteListReferences.continuations) >= canonicalRemoteLogCacheLimit {
		for candidate := range remoteListReferences.continuations {
			delete(remoteListReferences.continuations, candidate)
			break
		}
	}
	remoteListReferences.continuations[key] = entry
}

// A view starts a due retry once. An unfinished sync consumes its step.
func remoteListReferencesDue(app core.App, iri string) bool {
	key := app.DataDir() + "\x00" + iri
	remoteListReferences.Lock()
	defer remoteListReferences.Unlock()
	entry, found := remoteListReferences.continuations[key]
	if !found || time.Now().Before(entry.due) {
		return false
	}
	if entry.started {
		if entry.misses >= len(remoteListReferenceRetries) {
			delete(remoteListReferences.continuations, key)
			return false
		}
		entry.misses++
	}
	entry.started = true
	entry.due = time.Now().Add(remoteListReferenceRetries[min(entry.misses, len(remoteListReferenceRetries)-1)])
	remoteListReferences.continuations[key] = entry
	return true
}
