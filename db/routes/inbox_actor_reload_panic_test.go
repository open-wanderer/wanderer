package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type inboxReloadLogBuffer struct {
	sync.Mutex
	buffer bytes.Buffer
}

func (b *inboxReloadLogBuffer) Write(value []byte) (int, error) {
	b.Lock()
	defer b.Unlock()
	return b.buffer.Write(value)
}

func (b *inboxReloadLogBuffer) String() string {
	b.Lock()
	defer b.Unlock()
	return b.buffer.String()
}

type inboxReloadLoggerApp struct {
	core.App
	logger *slog.Logger
}

func (app inboxReloadLoggerApp) Logger() *slog.Logger { return app.logger }

func TestInboxKnownKeyReloadReturnsErrorsToAllWaitingRequests(t *testing.T) {
	w := newKnownKeyWorld(t)
	actor := w.sender(t, "sender", "remote.example", true)
	var logs inboxReloadLogBuffer
	app := inboxReloadLoggerApp{App: w.app, logger: slog.New(slog.NewJSONHandler(&logs, nil))}
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	router.POST("/inbox", ActivitypubActivityProcess)
	handler, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	started, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	var fetches atomic.Int32
	getInboxActorByIRI = func(core.App, context.Context, string, bool) (*core.Record, error) {
		fetches.Add(1)
		close(started)
		<-release
		panic("actor refresh stopped")
	}
	requests := make([]*http.Request, 5)
	for i := range requests {
		requests[i] = w.inboxRouterRequest(t, actor.GetString("iri"), nil, nil)
	}
	type response struct {
		status int
		body   string
	}
	responses := make(chan response, len(requests))
	deliver := func(request *http.Request) {
		result := httptest.NewRecorder()
		handler.ServeHTTP(result, request)
		responses <- response{status: result.Code, body: result.Body.String()}
	}
	go deliver(requests[0])
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("actor refresh did not start")
	}
	for _, request := range requests[1:] {
		go deliver(request)
	}
	waitForInboxActorReloads(t, len(requests))
	select {
	case result := <-responses:
		t.Fatalf("delivery returned before the refresh completed: %d", result.status)
	default:
	}
	once.Do(func() { close(release) })
	for range requests {
		select {
		case result := <-responses:
			if result.status != http.StatusInternalServerError {
				t.Errorf("waiting request response: %d", result.status)
			}
			if strings.Contains(result.body, "actor refresh stopped") || strings.Contains(result.body, "goroutine ") {
				t.Errorf("response includes refresh details: %s", result.body)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("waiting request did not finish")
		}
	}
	if got := fetches.Load(); got != 1 {
		t.Fatalf("shared actor refreshes: %d", got)
	}
	diagnostics := 0
	for _, line := range strings.Split(strings.TrimSpace(logs.String()), "\n") {
		if line == "" {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}
		if record["msg"] != "actor refresh did not complete" {
			continue
		}
		diagnostics++
		stack, _ := record["stack"].(string)
		if record["level"] != "ERROR" || record["value"] != "actor refresh stopped" || !strings.Contains(stack, "runtime/debug.Stack") || !strings.Contains(stack, "reloadKnownInboxActor") {
			t.Errorf("refresh diagnostic: %v", record)
		}
	}
	if diagnostics != 1 {
		t.Errorf("refresh diagnostics: %d, want 1", diagnostics)
	}
	health := httptest.NewRecorder()
	handler.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("router health after refresh error: %d", health.Code)
	}
}

func TestInboxKnownKeyRefreshFailurePrecedesActorRead(t *testing.T) {
	w := newKnownKeyWorld(t)
	actor := w.sender(t, "sender", "remote.example", true)
	handler := knownKeyInboxRouter(t, w)
	getInboxActorByIRI = func(app core.App, _ context.Context, _ string, _ bool) (*core.Record, error) {
		if err := app.Delete(actor); err != nil {
			return nil, err
		}
		panic("actor refresh stopped")
	}
	if status := knownKeyRouterStatus(handler, w.inboxRouterRequest(t, actor.GetString("iri"), nil, nil)); status != http.StatusInternalServerError {
		t.Errorf("refresh failure after record removal: %d", status)
	}
}
