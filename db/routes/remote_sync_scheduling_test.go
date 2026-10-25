package routes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
	"pocketbase/util"
)

func TestRemoteSyncHostWaitersLeaveCapacityForOtherHosts(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteSyncFixture(t)
	release := make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	entered := make(chan string, 10)
	done := make(chan error, 10)
	start := func(iri string) {
		record := core.NewRecord(f.trails)
		record.Set("iri", iri)
		record.Set("author", f.remoteActor.Id)
		go func() {
			_, err := limitedRemoteSync(f.app, t.Context(), "trails", record, func(ctx context.Context) (*core.Record, error) {
				entered <- iri
				select {
				case <-release:
				case <-ctx.Done():
					return record, ctx.Err()
				}
				return record, nil
			})
			done <- err
		}()
	}
	start("https://remote.example/api/v1/trail/first")
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("first host never started")
	}
	start("https://REMOTE.example:9443/api/v1/trail/second")
	start("http://remote.example:8080/api/v1/trail/third")
	// Observe registered host waiters before starting the unrelated origin.
	deadline := time.Now().Add(3 * time.Second)
	for {
		remoteSyncHosts.Lock()
		host := remoteSyncHosts.hosts[f.app.DataDir()+"\x00remote.example"]
		n := 0
		if host != nil {
			n = host.users
		}
		remoteSyncHosts.Unlock()
		if n == 3 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("host waiters did not register")
		}
		time.Sleep(time.Millisecond)
	}
	start("https://other.example/api/v1/trail/fast")
	select {
	case iri := <-entered:
		if !strings.Contains(iri, "other.example") {
			t.Fatalf("same hostname acquired another slot: %s", iri)
		}
	case <-time.After(time.Second):
		t.Fatal("slow host blocked a different origin")
	}
	once.Do(func() { close(release) })
	for range 4 {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
}

func TestRemoteSyncQueueDoesNotConsumeDocumentBudget(t *testing.T) {
	f := newRemoteSyncFixture(t)
	release, entered := make(chan struct{}), make(chan struct{})
	holder := core.NewRecord(f.trails)
	holder.Set("iri", "https://remote.example/first")
	done := make(chan error, 1)
	go func() {
		_, err := limitedRemoteSync(f.app, t.Context(), "trails", holder, func(ctx context.Context) (*core.Record, error) { close(entered); <-release; return holder, nil })
		done <- err
	}()
	<-entered
	ctx, cancel := context.WithTimeout(t.Context(), 2*remoteSyncBudget)
	defer cancel()
	queued := core.NewRecord(f.trails)
	queued.Set("iri", "https://remote.example/second")
	go func() { time.Sleep(30 * time.Millisecond); close(release) }()
	_, err := limitedRemoteSync(f.app, ctx, "trails", queued, func(runCtx context.Context) (*core.Record, error) {
		deadline, ok := runCtx.Deadline()
		if !ok || time.Until(deadline) < remoteSyncBudget+10*time.Second {
			return nil, fmt.Errorf("queue consumed document budget")
		}
		return queued, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestRemoteSyncCompletedCopiesDoNotWaitForCapacity(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	for _, kind := range []string{"trail", "list"} {
		t.Run(kind, func(t *testing.T) {
			f := newRemoteSyncFixture(t)
			col := f.trails
			handler := RemoteTrailGet
			if kind == "list" {
				col = f.lists
				handler = RemoteListGet
			}
			col.ViewRule = types.Pointer("public=true")
			if err := f.app.Save(col); err != nil {
				t.Fatal(err)
			}
			record := f.save(col, map[string]any{"iri": "https://remote.example/api/v1/" + kind + "/copy", "name": "Completed copy", "public": true, "author": f.remoteActor.Id, "needs_full_sync": true, "full_sync_completed": true})
			oldActor, oldClient := remoteHandleGetActor, newRemoteSyncHTTPClient
			remoteHandleGetActor = func(core.App, context.Context, string, bool) (*core.Record, error) { return f.remoteActor, nil }
			newRemoteSyncHTTPClient = func() *http.Client {
				return &http.Client{Transport: richTextSyncTransport{map[string]any{"id": "copy", "iri": record.GetString("iri"), "name": "Refreshed", "public": true, "expand": map[string]any{"author": map[string]any{"iri": f.remoteActor.GetString("iri")}}}}}
			}
			defer func() { remoteHandleGetActor, newRemoteSyncHTTPClient = oldActor, oldClient }()
			for range remoteSyncConcurrency {
				remoteSyncSlots <- struct{}{}
			}
			released := false
			release := func() {
				if !released {
					released = true
					for range remoteSyncConcurrency {
						<-remoteSyncSlots
					}
				}
			}
			defer release()
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			e := &core.RequestEvent{App: f.app}
			e.Request = httptest.NewRequest(http.MethodGet, "/remote/"+kind+"/copy?handle=@riley@remote.example", nil).WithContext(ctx)
			e.Request.SetPathValue("id", "copy")
			response := httptest.NewRecorder()
			e.Response = response
			started := time.Now()
			err := handler(e)
			elapsed := time.Since(started)
			release()
			if waitErr := waitForBackgroundSync(record, 3*time.Second); waitErr != nil {
				t.Fatal(waitErr)
			}
			if err != nil || response.Code != 200 || elapsed > 250*time.Millisecond || !strings.Contains(response.Body.String(), "Completed copy") {
				t.Fatalf("completed view waited %s: status%d err%v body%s", elapsed, response.Code, err, response.Body.String())
			}
		})
	}
}

func TestRemoteSyncDocumentComplexityKeepsWireLimit(t *testing.T) {
	prefix := `{"id":"copy","name":"`
	padding := strings.Repeat("a", int(util.FederationSyncBodyLimit)-len(prefix)-3)
	var data map[string]any
	if err := decodeRemoteSyncJSON(t.Context(), strings.NewReader(prefix+padding+`"}`), util.FederationSyncBodyLimit, &data); err != nil {
		t.Fatal(err)
	}
	compact := `{"extra":[` + strings.Repeat(`{"x":0},`, remoteSyncJSONTokenLimit/4) + `{}]}`
	if err := decodeRemoteSyncJSON(t.Context(), strings.NewReader(compact), util.FederationSyncBodyLimit, &data); !errors.Is(err, errRemoteSyncJSONComplexity) {
		t.Fatalf("complex document accepted: %v", err)
	}
}

func TestRemoteSyncLimitsDecodedDocumentsAcrossHosts(t *testing.T) {
	f := newRemoteSyncFixture(t)
	release := make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	entered := make(chan struct{}, remoteSyncConcurrency+2)
	done := make(chan error, remoteSyncConcurrency+2)
	for i := 0; i < remoteSyncConcurrency+2; i++ {
		record := core.NewRecord(f.trails)
		record.Set("iri", fmt.Sprintf("https://origin%d.example/trail/copy", i))
		go func() {
			_, err := limitedRemoteSync(f.app, t.Context(), "trails", record, func(ctx context.Context) (*core.Record, error) {
				document := &remoteSyncDocument{}
				file, err := os.CreateTemp(t.TempDir(), "document-")
				if err != nil {
					return nil, err
				}
				document.file = file
				defer document.close()
				if _, err := file.WriteString(`{"id":"copy"}`); err != nil {
					return nil, err
				}
				return document.process(ctx, func(map[string]any) (*core.Record, error) { entered <- struct{}{}; <-release; return record, nil })
			})
			done <- err
		}()
	}
	for range remoteSyncConcurrency {
		select {
		case <-entered:
		case <-time.After(3 * time.Second):
			t.Fatal("independent origins did not start")
		}
	}
	select {
	case <-entered:
		t.Error("too many documents decoded concurrently")
	case <-time.After(30 * time.Millisecond):
	}
	once.Do(func() { close(release) })
	for range remoteSyncConcurrency + 2 {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
}

func TestRemoteSyncRejectsComplexDocumentBeforeStorage(t *testing.T) {
	t.Setenv("ORIGIN", "https://local.example")
	f := newRemoteSyncFixture(t)
	extra := make([]any, remoteSyncJSONTokenLimit/4+10)
	for i := range extra {
		extra[i] = map[string]any{"x": 0}
	}
	payload := map[string]any{"id": "copy", "name": "Remote copy", "extra": extra, "expand": map[string]any{"author": map[string]any{"iri": f.remoteActor.GetString("iri")}}}
	previous := newRemoteSyncHTTPClient
	defer func() { newRemoteSyncHTTPClient = previous }()
	newRemoteSyncHTTPClient = func() *http.Client { return &http.Client{Transport: richTextSyncTransport{payload}} }
	record := core.NewRecord(f.trails)
	record.Set("iri", "https://remote.example/api/v1/trail/copy")
	record.Set("author", f.remoteActor.Id)
	if _, err := performFullSync(f.app, t.Context(), nil, record); !errors.Is(err, errRemoteUnavailable) || !errors.Is(err, errRemoteSyncJSONComplexity) {
		t.Fatalf("document not rejected before allocation: %v", err)
	}
	if count, err := f.app.CountRecords("trails"); err != nil || count != 0 {
		t.Fatalf("rejected document changed storage: %d %v", count, err)
	}
}

func TestRemoteSyncQueueDeadlineCleansWaitersAndLeavesRunningBudget(t *testing.T) {
	f := newRemoteSyncFixture(t)
	record := func(iri string) *core.Record { r := core.NewRecord(f.trails); r.Set("iri", iri); return r }
	holder := record("https://remote.example/trail/first")
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	var releaseOnce sync.Once
	releaseHeld := func() { releaseOnce.Do(func() { close(release) }) }
	defer releaseHeld()
	go func() {
		_, err := limitedRemoteSync(f.app, t.Context(), "trails", holder, func(context.Context) (*core.Record, error) { close(entered); <-release; return holder, nil })
		done <- err
	}()
	<-entered
	for _, iri := range []string{holder.GetString("iri"), "https://remote.example:9443/trail/other"} {
		called := false
		waitCtx, cancelWait := context.WithTimeout(t.Context(), 200*time.Millisecond)
		started := time.Now()
		_, err := limitedRemoteSyncWithin(f.app, waitCtx, "trails", record(iri), 10*time.Millisecond, func(context.Context) (*core.Record, error) { called = true; return nil, nil })
		cancelWait()
		if !errors.Is(err, context.DeadlineExceeded) || called || time.Since(started) > 100*time.Millisecond {
			t.Fatalf("expired queue started work: %v called%v", err, called)
		}
	}
	remoteSyncFlights.Lock()
	flights := len(remoteSyncFlights.calls)
	remoteSyncFlights.Unlock()
	remoteSyncHosts.Lock()
	hosts := len(remoteSyncHosts.hosts)
	users := remoteSyncHosts.hosts[f.app.DataDir()+"\x00remote.example"].users
	remoteSyncHosts.Unlock()
	releaseHeld()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if flights != 1 || hosts != 1 || users != 1 {
		t.Fatalf("expired queue retained flights%d hosts%d users%d", flights, hosts, users)
	}
	for range remoteSyncNetworkConcurrency {
		remoteSyncNetworkSlots <- struct{}{}
	}
	called := false
	waitCtx, cancelWait := context.WithTimeout(t.Context(), 200*time.Millisecond)
	startedAt := time.Now()
	_, err := limitedRemoteSyncWithin(f.app, waitCtx, "trails", record("https://other.example/trail/queued"), 10*time.Millisecond, func(context.Context) (*core.Record, error) { called = true; return nil, nil })
	for range remoteSyncNetworkConcurrency {
		<-remoteSyncNetworkSlots
	}
	cancelWait()
	if !errors.Is(err, context.DeadlineExceeded) || called || time.Since(startedAt) > 100*time.Millisecond {
		t.Fatalf("global queue ignored expiry: %v called%v", err, called)
	}
	remoteSyncHosts.Lock()
	hosts = len(remoteSyncHosts.hosts)
	remoteSyncHosts.Unlock()
	remoteSyncFlights.Lock()
	flights = len(remoteSyncFlights.calls)
	remoteSyncFlights.Unlock()
	if hosts != 0 || flights != 0 {
		t.Fatalf("expired global waiter retained hosts%d flights%d", hosts, flights)
	}
	started := record("https://other.example/trail/started")
	_, err = limitedRemoteSyncWithin(f.app, t.Context(), "trails", started, time.Millisecond, func(ctx context.Context) (*core.Record, error) {
		time.Sleep(10 * time.Millisecond)
		return started, ctx.Err()
	})
	if err != nil {
		t.Fatalf("queue timer canceled running document phase: %v", err)
	}
}

func TestRemoteSyncDocumentAcceptsExpandedListCollections(t *testing.T) {
	// Include ordinary serialized metadata and expanded actors and tags for a
	// list slightly larger than one collection batch. Truncation happens later.
	children := make([]any, 1105)
	for i := range children {
		child := map[string]any{}
		for _, field := range []string{"id", "iri", "collectionId", "collectionName", "created", "updated", "name", "description", "author", "public", "distance", "elevation_gain", "elevation_loss", "duration", "difficulty", "category", "subcategory", "tags", "photos", "gpx", "lat", "lon", "max_altitude", "min_altitude", "ascent", "descent", "bounds", "polyline", "full_sync_completed", "needs_full_sync", "federated_category_name", "federated_subcategory_name", "likes", "bounding_box_diagonal", "summit_logs", "waypoints", "date", "source", "url", "language"} {
			child[field] = ""
		}
		actor := map[string]any{}
		for _, field := range []string{"id", "iri", "inbox", "outbox", "following", "followers", "domain", "preferred_username", "last_fetched", "public_key", "username", "name", "summary", "avatar", "url", "is_local"} {
			actor[field] = ""
		}
		tags := make([]any, 5)
		for n := range tags {
			tags[n] = map[string]any{"id": fmt.Sprint(n), "name": "An ordinary tag", "collectionId": "tags", "collectionName": "tags", "created": "", "updated": ""}
		}
		child["expand"] = map[string]any{"author": actor, "tags": tags}
		children[i] = child
	}
	encoded, err := json.Marshal(map[string]any{"id": "list", "expand": map[string]any{"trails": children}})
	if err != nil {
		t.Fatal(err)
	}
	var data map[string]any
	if err := decodeRemoteSyncJSON(t.Context(), strings.NewReader(string(encoded)), util.FederationSyncBodyLimit, &data); err != nil {
		t.Fatalf("ordinary expanded list rejected before collection limit: %v", err)
	}
}
