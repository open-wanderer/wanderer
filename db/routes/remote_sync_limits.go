package routes

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"pocketbase/util"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/pocketbase/pocketbase/core"
	"golang.org/x/net/idna"
	"golang.org/x/net/publicsuffix"
)

const (
	remoteSyncConcurrency            = 2
	remoteSyncNetworkConcurrency     = 8
	remoteSyncFetchConcurrency       = 16
	remoteSyncFetchDomainConcurrency = 4
	remoteSyncChildLimit             = 1000
	remoteSyncWriteBatch             = 100
)

type remoteSyncFlight struct {
	done   chan struct{}
	record *core.Record
	err    error
}

type remoteSyncHost struct {
	slot  chan struct{}
	users int
}

type remoteSyncHostSet struct {
	sync.Mutex
	hosts map[string]*remoteSyncHost
}

var remoteSyncHosts = remoteSyncHostSet{hosts: map[string]*remoteSyncHost{}}
var remoteSyncFetchHosts = remoteSyncHostSet{hosts: map[string]*remoteSyncHost{}}

// Subdomains, ports and alternate address spellings share one origin queue.
func remoteSyncDomain(host string) string {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if address, err := netip.ParseAddr(host); err == nil {
		return address.Unmap().String()
	}
	if ascii, err := idna.Lookup.ToASCII(host); err == nil {
		host = ascii
	}
	if domain, err := publicsuffix.EffectiveTLDPlusOne(host); err == nil {
		return domain
	}
	return host
}

// Wait on the origin before reserving process-wide network capacity.
func acquireRemoteSyncHost(ctx context.Context, key string) (func(), error) {
	return remoteSyncHosts.acquire(ctx, key, 1)
}

func (hosts *remoteSyncHostSet) acquire(ctx context.Context, key string, capacity int) (func(), error) {
	hosts.Lock()
	host := hosts.hosts[key]
	if host == nil {
		host = &remoteSyncHost{slot: make(chan struct{}, capacity)}
		hosts.hosts[key] = host
	}
	host.users++
	hosts.Unlock()
	drop := func() {
		hosts.Lock()
		host.users--
		if host.users == 0 {
			delete(hosts.hosts, key)
		}
		hosts.Unlock()
	}
	select {
	case <-ctx.Done():
		drop()
		return nil, ctx.Err()
	case host.slot <- struct{}{}:
		return func() { <-host.slot; drop() }, nil
	}
}

var remoteSyncSlots = make(chan struct{}, remoteSyncConcurrency)
var remoteSyncProcessingQueue = make(chan struct{}, 1)

// Reserve a document's complete weight together. In particular, two large
// documents cannot each retain one permit while waiting for the other.
func acquireRemoteSyncProcessing(ctx context.Context, tokens int) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case remoteSyncProcessingQueue <- struct{}{}:
	}
	defer func() { <-remoteSyncProcessingQueue }()
	weight := 1
	if tokens > remoteSyncJSONSharedTokenLimit {
		weight = remoteSyncConcurrency
	}
	acquired := 0
	for acquired < weight {
		select {
		case <-ctx.Done():
			for acquired > 0 {
				<-remoteSyncSlots
				acquired--
			}
			return nil, ctx.Err()
		case remoteSyncSlots <- struct{}{}:
			acquired++
		}
	}
	if err := ctx.Err(); err != nil {
		for range weight {
			<-remoteSyncSlots
		}
		return nil, err
	}
	return func() {
		for range weight {
			<-remoteSyncSlots
		}
	}, nil
}

var remoteSyncNetworkSlots = make(chan struct{}, remoteSyncNetworkConcurrency)
var remoteSyncFetchSlots = make(chan struct{}, remoteSyncFetchConcurrency)

type remoteSyncStagingKey struct{}

// A wait for local fetch capacity ended before any request was made.
var errRemoteSyncFetchQueue = errors.New("remote fetch capacity unavailable")

func acquireRemoteSyncFetch(ctx context.Context, rawURL string) (func(), error) {
	if ctx.Value(remoteSyncStagingKey{}) == nil {
		return func() {}, nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Hostname() == "" {
		return nil, fmt.Errorf("invalid remote fetch origin")
	}
	// Contributions, authors and media may use a different domain from the
	// workflow. Queue on that target before reserving global fetch capacity.
	// A separate registry avoids recursively acquiring the workflow's slot.
	releaseHost, err := remoteSyncFetchHosts.acquire(ctx, remoteSyncDomain(parsed.Hostname()), remoteSyncFetchDomainConcurrency)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errRemoteSyncFetchQueue, err)
	}
	select {
	case <-ctx.Done():
		releaseHost()
		return nil, fmt.Errorf("%w: %w", errRemoteSyncFetchQueue, ctx.Err())
	case remoteSyncFetchSlots <- struct{}{}:
		return func() { <-remoteSyncFetchSlots; releaseHost() }, nil
	}
}

var remoteSyncFlights = struct {
	sync.Mutex
	calls map[string]*remoteSyncFlight
}{calls: map[string]*remoteSyncFlight{}}

// Bound staged documents and network plans independently of processing.
// Same-copy requests share one result; domain waiters retain no document.
func limitedRemoteSync(app core.App, ctx context.Context, collection string, record *core.Record, run func(context.Context) (*core.Record, error)) (*core.Record, error) {
	return limitedRemoteSyncWithin(app, ctx, collection, record, remoteSyncBudget, run)
}

func limitedRemoteSyncWithin(app core.App, ctx context.Context, collection string, record *core.Record, waitBudget time.Duration, run func(context.Context) (*core.Record, error)) (*core.Record, error) {
	iri := record.GetString("iri")
	if iri == "" || util.IsLocalIRI(iri) || remoteRecordHasLocalAuthor(app, record) {
		return run(ctx)
	}
	if err := ctx.Err(); err != nil {
		return record, fmt.Errorf("%w: %w", errRemoteUnavailable, err)
	}
	waitCtx, cancelWait := context.WithTimeout(ctx, waitBudget)
	defer cancelWait()
	key := app.DataDir() + "\x00" + collection + "\x00" + iri
	remoteSyncFlights.Lock()
	if call := remoteSyncFlights.calls[key]; call != nil {
		remoteSyncFlights.Unlock()
		select {
		case <-waitCtx.Done():
			return record, fmt.Errorf("%w: %w", errRemoteUnavailable, waitCtx.Err())
		case <-call.done:
			if call.record == nil {
				return record, call.err
			}
			return call.record.Fresh(), call.err
		}
	}
	call := &remoteSyncFlight{done: make(chan struct{})}
	remoteSyncFlights.calls[key] = call
	remoteSyncFlights.Unlock()
	defer func() {
		remoteSyncFlights.Lock()
		delete(remoteSyncFlights.calls, key)
		close(call.done)
		remoteSyncFlights.Unlock()
	}()
	parsed, err := url.Parse(iri)
	if err != nil || parsed.Host == "" {
		call.err = fmt.Errorf("%w: invalid remote origin", errRemoteUnavailable)
		return record, call.err
	}
	releaseHost, err := acquireRemoteSyncHost(waitCtx, app.DataDir()+"\x00"+remoteSyncDomain(parsed.Hostname()))
	if err != nil {
		call.err = fmt.Errorf("%w: %w", errRemoteUnavailable, err)
		return record, call.err
	}
	defer releaseHost()
	select {
	case <-waitCtx.Done():
		call.err = fmt.Errorf("%w: %w", errRemoteUnavailable, waitCtx.Err())
		return record, call.err
	case remoteSyncNetworkSlots <- struct{}{}:
	}
	defer func() { <-remoteSyncNetworkSlots }()
	cancelWait()
	if err := ctx.Err(); err != nil {
		call.err = fmt.Errorf("%w: %w", errRemoteUnavailable, err)
		return record, call.err
	}
	result, err := run(context.WithValue(ctx, remoteSyncStagingKey{}, true))
	if result != nil {
		call.record = result.Fresh()
	}
	call.err = err
	return result, err
}

// Apply collection limits before author and media preparation as well as writes.
func capRemoteSyncChildren(app core.App, data map[string]any, origin string, relations ...string) bool {
	capped := false
	expand, _ := data["expand"].(map[string]any)
	for _, relation := range relations {
		children, _ := expand[relation].([]any)
		if len(children) > remoteSyncChildLimit {
			capped = true
			app.Logger().Warn("remote sync child limit reached", "collection", relation, "received", len(children), "processed", remoteSyncChildLimit)
			children = append([]any(nil), children[:remoteSyncChildLimit]...)
			expand[relation] = children
		}
		collection := map[string]string{"trails": "trails", "waypoints_via_trail": "waypoints", "summit_logs_via_trail": "summit_logs"}[relation]
		kept := make([]any, 0, len(children))
		skipped := 0
		for _, child := range children {
			raw, _ := child.(map[string]any)
			id, _ := raw["id"].(string)
			iri, _ := raw["iri"].(string)
			if (iri == "" || remoteSyncIRIMatchesOrigin(iri, origin)) && !remoteSyncIDFitsCollection(app, collection, id) {
				skipped++
				continue
			}
			kept = append(kept, child)
		}
		if skipped > 0 {
			capped = true
			expand[relation] = kept
			app.Logger().Warn("remote sync skipped invalid record ids", "collection", collection, "skipped", skipped)
		}
	}
	return capped
}

// Source record ids follow the collection's existing field contract. Check
// before an id is repeated across derived media URLs or child references.
func remoteSyncIDFitsCollection(app core.App, collection, id string) bool {
	schema, err := app.FindCachedCollectionByNameOrId(collection)
	if err != nil {
		return false
	}
	field, ok := schema.Fields.GetByName("id").(*core.TextField)
	return !ok || field.Max <= 0 || utf8.RuneCountInString(id) <= field.Max
}

// Commit bounded child batches so unrelated writers can run between batches.
func remoteSyncChildBatches(app core.App, ctx context.Context, children []any, save func(core.App, []any) error) error {
	for start := 0; start < len(children); start += remoteSyncWriteBatch {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("%w: %w", errRemoteUnavailable, err)
		}
		end := min(start+remoteSyncWriteBatch, len(children))
		if err := app.RunInTransaction(func(tx core.App) error { return save(tx, children[start:end]) }); err != nil {
			return err
		}
	}
	return nil
}

func remoteListChildIRI(value any, origin string) string {
	raw, _ := value.(map[string]any)
	iri, _ := raw["iri"].(string)
	if iri == "" {
		id, _ := raw["id"].(string)
		iri = fmt.Sprintf("%s/api/v1/trail/%s", origin, id)
	}
	return iri
}

// Keep only earlier trail identities while scanning the complete document, so
// this set cannot grow with the number of unprocessed remote children.
func previousListTrailsInDocument(app core.App, previous []string, children []any, origin string) map[string]bool {
	named := make(map[string]bool, len(previous))
	for _, id := range previous {
		if trail, err := app.FindRecordById("trails", id); err == nil {
			named[trail.GetString("iri")] = false
		}
	}
	for _, value := range children {
		iri := remoteListChildIRI(value, origin)
		if _, earlier := named[iri]; earlier {
			named[iri] = true
		}
	}
	return named
}

func retainUnprocessedListTrails(app core.App, author string, ids, previous []string, children []any, namedPrevious map[string]bool, origin string) []string {
	selected := map[string]bool{}
	for _, value := range children {
		selected[remoteListChildIRI(value, origin)] = true
	}
	seen := map[string]bool{}
	for _, id := range ids {
		seen[id] = true
	}
	for _, id := range previous {
		if seen[id] {
			continue
		}
		trail, err := app.FindRecordById("trails", id)
		if err != nil {
			continue
		}
		iri := trail.GetString("iri")
		if !namedPrevious[iri] || selected[iri] {
			continue
		}
		if (util.IsLocalIRI(iri) || !remoteSyncIRIMatchesOrigin(iri, origin) || remoteRecordHasLocalAuthor(app, trail)) && !remoteListAuthorCanViewLocalTrail(app, author, trail) {
			continue
		}
		ids = append(ids, id)
		seen[id] = true
	}
	return ids
}
