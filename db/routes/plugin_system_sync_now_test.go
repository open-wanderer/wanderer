package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/meilisearch/meilisearch-go"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/pluginsystem"
)

func TestPluginSyncLimiterCooldownPerPlugin(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	limiter := newPluginSyncLimiter(15*time.Minute, 5)
	limiter.now = func() time.Time { return now }

	release, _, err := limiter.startManual("komoot", "komoot")
	if err != nil {
		t.Fatal(err)
	}
	release()
	// Switching on a second plugin right after the first must still give it
	// its first sync.
	if release, _, err := limiter.startManual("strava", "strava"); err != nil {
		t.Fatalf("another instance blocked by the first one's cooldown: %v", err)
	} else {
		release()
	}
	now = now.Add(10 * time.Minute)
	_, retryAfter, err := limiter.startManual("komoot", "komoot")
	if !errors.Is(err, errPluginSyncCooldown) || retryAfter != 5*time.Minute {
		t.Fatalf("second start within cooldown: retryAfter=%s, err=%v", retryAfter, err)
	}
	now = now.Add(5 * time.Minute)
	if release, _, err := limiter.startManual("komoot", "komoot"); err != nil {
		t.Fatalf("start after cooldown: %v", err)
	} else {
		release()
	}
}

func TestPluginSyncLimiterCapsRunningSyncs(t *testing.T) {
	limiter := newPluginSyncLimiter(0, 2)
	first, _, err := limiter.startManual("a1", "a1")
	if err != nil {
		t.Fatal(err)
	}
	if release, _, err := limiter.startManual("b1", "b1"); err != nil {
		t.Fatal(err)
	} else {
		defer release()
	}
	if _, _, err := limiter.startManual("c1", "c1"); !errors.Is(err, errPluginSyncBusy) {
		t.Fatalf("third running sync: %v", err)
	}
	first()
	if release, _, err := limiter.startManual("c1", "c1"); err != nil {
		t.Fatalf("start after a slot freed: %v", err)
	} else {
		release()
	}
}

func TestManualSyncMaxRunningLeavesAWorkerFree(t *testing.T) {
	for _, c := range []struct{ configured, workers, want int }{
		{2, 2, 1}, {2, 8, 2}, {2, 1, 0}, {5, 4, 3}, {0, 8, 0},
	} {
		if got := manualSyncMaxRunning(c.configured, c.workers); got != c.want {
			t.Errorf("manualSyncMaxRunning(%d, %d) = %d, want %d", c.configured, c.workers, got, c.want)
		}
	}
}

func TestPluginSyncEnvFallsBackOnInvalidValues(t *testing.T) {
	for _, raw := range []string{"soon", "-5m", "0s"} {
		t.Setenv("TEST_PLUGIN_SYNC_DURATION", raw)
		if got := envDuration("TEST_PLUGIN_SYNC_DURATION", time.Minute, time.Second); got != time.Minute {
			t.Errorf("duration %q below its minimum: got %s, want the fallback", raw, got)
		}
	}
	t.Setenv("TEST_PLUGIN_SYNC_DURATION", "0s")
	if got := envDuration("TEST_PLUGIN_SYNC_DURATION", time.Minute, 0); got != 0 {
		t.Errorf("zero cooldown: %s", got)
	}
	t.Setenv("TEST_PLUGIN_SYNC_DURATION", "1h")
	if got := envDuration("TEST_PLUGIN_SYNC_DURATION", time.Minute, time.Second); got != time.Hour {
		t.Errorf("valid duration: %s", got)
	}
	t.Setenv("TEST_PLUGIN_SYNC_INT", "-1")
	if got := envInt("TEST_PLUGIN_SYNC_INT", 2); got != 2 {
		t.Errorf("negative int: %d", got)
	}
	t.Setenv("TEST_PLUGIN_SYNC_INT", "0")
	if got := envInt("TEST_PLUGIN_SYNC_INT", 2); got != 0 {
		t.Errorf("valid int: %d", got)
	}
}

type pluginSyncRouteFixture struct {
	*pluginInstanceFixture
	handler http.Handler
	owner   string
	other   string
}

func newPluginSyncRouteFixture(t *testing.T) *pluginSyncRouteFixture {
	t.Helper()
	f := &pluginSyncRouteFixture{pluginInstanceFixture: newPluginInstanceFixture(t)}
	users := core.NewAuthCollection("plugin_sync_users")
	if err := f.app.Save(users); err != nil {
		t.Fatal(err)
	}
	tokens := make([]string, 2)
	for i, email := range []string{"owner@example.com", "other@example.com"} {
		user := core.NewRecord(users)
		user.SetEmail(email)
		user.SetPassword("test-plugin-sync-password")
		if err := f.app.Save(user); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			f.ownerID = user.Id
		}
		token, err := user.NewAuthToken()
		if err != nil {
			t.Fatal(err)
		}
		tokens[i] = token
	}
	f.owner, f.other = tokens[0], tokens[1]

	router, err := apis.NewRouter(f.app)
	if err != nil {
		t.Fatal(err)
	}
	router.POST("/api/plugins/sync", PluginSystemSync(nil))
	if f.handler, err = router.BuildMux(); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *pluginSyncRouteFixture) sync(t *testing.T, token string, instanceID string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(pluginSyncRequest{InstanceID: instanceID})
	req := httptest.NewRequest(http.MethodPost, "/api/plugins/sync", bytes.NewReader(body))
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	f.handler.ServeHTTP(response, req)
	return response
}

func usePluginSyncLimiter(t *testing.T, limiter *pluginSyncLimiter) {
	t.Helper()
	original := pluginSyncs
	pluginSyncs = limiter
	t.Cleanup(func() { pluginSyncs = original })
}

func retryAfterFromBody(t *testing.T, response *httptest.ResponseRecorder) int {
	t.Helper()
	var body struct {
		Data struct {
			RetryAfter int `json:"retryAfter"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("429 body: %v: %s", err, response.Body.String())
	}
	return body.Data.RetryAfter
}

func TestPluginSystemSyncRejectsBeforeStartingASync(t *testing.T) {
	f := newPluginSyncRouteFixture(t)
	enabled := f.instance(t, true, time.Time{})
	disabled := f.instance(t, false, time.Time{})
	backoff := f.instance(t, true, time.Now().Add(30*time.Minute))

	cases := []struct {
		name     string
		token    string
		instance string
		status   int
	}{
		{"anonymous", "", enabled, http.StatusUnauthorized},
		{"missing instance id", f.owner, "", http.StatusBadRequest},
		{"another user's instance", f.other, enabled, http.StatusNotFound},
		{"disabled instance", f.owner, disabled, http.StatusBadRequest},
		{"provider backoff", f.owner, backoff, http.StatusTooManyRequests},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			limiter := newPluginSyncLimiter(time.Hour, 2)
			usePluginSyncLimiter(t, limiter)
			response := f.sync(t, c.token, c.instance)
			if response.Code != c.status {
				t.Fatalf("status %d, want %d: %s", response.Code, c.status, response.Body.String())
			}
			if len(limiter.lastStart) != 0 || limiter.running != 0 {
				t.Fatal("a rejected request counted toward the limits")
			}
		})
	}
}

func TestPluginSystemSyncLimiterAnswers(t *testing.T) {
	f := newPluginSyncRouteFixture(t)
	instanceID := f.instance(t, true, time.Time{})
	f.installPlugin(t)

	t.Run("cooldown", func(t *testing.T) {
		limiter := newPluginSyncLimiter(15*time.Minute, 2)
		usePluginSyncLimiter(t, limiter)
		release, _, err := limiter.startManual(instanceID, f.ownerID+"/komoot")
		if err != nil {
			t.Fatal(err)
		}
		release()
		response := f.sync(t, f.owner, instanceID)
		if response.Code != http.StatusTooManyRequests || response.Header().Get("Retry-After") != "900" || retryAfterFromBody(t, response) != 900 {
			t.Fatalf("status %d, Retry-After %q: %s", response.Code, response.Header().Get("Retry-After"), response.Body.String())
		}
	})
	t.Run("all slots busy", func(t *testing.T) {
		limiter := newPluginSyncLimiter(0, 1)
		usePluginSyncLimiter(t, limiter)
		if release, _, err := limiter.startManual("someone-else", "someone-else"); err != nil {
			t.Fatal(err)
		} else {
			defer release()
		}
		response := f.sync(t, f.owner, instanceID)
		if response.Code != http.StatusTooManyRequests || response.Header().Get("Retry-After") != "60" || retryAfterFromBody(t, response) != 60 {
			t.Fatalf("status %d, Retry-After %q: %s", response.Code, response.Header().Get("Retry-After"), response.Body.String())
		}
	})
	t.Run("turned off", func(t *testing.T) {
		usePluginSyncLimiter(t, newPluginSyncLimiter(0, 0))
		if response := f.sync(t, f.owner, instanceID); response.Code != http.StatusForbidden {
			t.Fatalf("status %d: %s", response.Code, response.Body.String())
		}
	})
	t.Run("already syncing", func(t *testing.T) {
		limiter := newPluginSyncLimiter(0, 2)
		usePluginSyncLimiter(t, limiter)
		release, _ := pluginsystem.TryLockInstance(instanceID)
		defer release()
		if response := f.sync(t, f.owner, instanceID); response.Code != http.StatusConflict {
			t.Fatalf("status %d: %s", response.Code, response.Body.String())
		}
	})
}

func TestPluginSystemSyncRunsInBackgroundAndReleases(t *testing.T) {
	f := newPluginSyncRouteFixture(t)
	instanceID := f.instance(t, true, time.Time{})
	f.installPlugin(t)
	limiter := newPluginSyncLimiter(0, 2)
	usePluginSyncLimiter(t, limiter)
	called := make(chan bool, 1)
	usePluginInstanceSync(t, func(ctx context.Context, _ core.App, _ meilisearch.ServiceManager, _ pluginsystem.LocalPlugin, instance *core.Record) (*pluginSystemSyncResult, error) {
		_, hasDeadline := ctx.Deadline()
		called <- hasDeadline && instance.Id == instanceID
		return &pluginSystemSyncResult{PluginID: "komoot"}, nil
	})

	if response := f.sync(t, f.owner, instanceID); response.Code != http.StatusAccepted {
		t.Fatalf("status %d: %s", response.Code, response.Body.String())
	}
	select {
	case ok := <-called:
		if !ok {
			t.Fatal("background sync ran without a deadline or on the wrong instance")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("background sync never ran")
	}
	for deadline := time.Now().Add(5 * time.Second); pluginsystem.InstanceBusy(instanceID); {
		if time.Now().After(deadline) {
			t.Fatal("instance stayed locked after the background sync")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestCronSkipsInstanceAManualSyncIsRunning(t *testing.T) {
	f := newPluginSyncRouteFixture(t)
	busy := f.instance(t, true, time.Time{})
	free := f.instance(t, true, time.Time{})
	limiter := newPluginSyncLimiter(0, 2)
	usePluginSyncLimiter(t, limiter)
	release, _, err := limiter.startManual(busy, busy)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	var synced []string
	usePluginInstanceSync(t, func(_ context.Context, _ core.App, _ meilisearch.ServiceManager, _ pluginsystem.LocalPlugin, instance *core.Record) (*pluginSystemSyncResult, error) {
		synced = append(synced, instance.Id)
		return &pluginSystemSyncResult{PluginID: "komoot"}, nil
	})

	if err := syncPluginInstancesOneByOne(t.Context(), f.app, nil, pluginsystem.LocalPlugin{}, []*core.Record{f.stored(t, busy), f.stored(t, free)}); err != nil {
		t.Fatal(err)
	}
	if len(synced) != 1 || synced[0] != free {
		t.Fatalf("cron synced %v; want only the free instance", synced)
	}
	if pluginsystem.InstanceBusy(free) {
		t.Fatal("cron left the instance it synced locked")
	}
}

func TestSyncTimeoutIsNotReportedAsProviderOutage(t *testing.T) {
	f := newPluginSyncRouteFixture(t)
	instanceID := f.instance(t, true, time.Time{})
	ctx, cancel := context.WithTimeout(t.Context(), time.Nanosecond)
	defer cancel()
	<-ctx.Done()

	setPluginInstanceStatusAfterSyncError(ctx, f.app, f.stored(t, instanceID), pluginsystem.RuntimeSessionFatalError{Err: ctx.Err()})
	if code := pluginsystem.JSONMapFromRecord(f.stored(t, instanceID), "last_error")["code"]; code != "sync_timeout" {
		t.Fatalf("last_error code = %v, want sync_timeout", code)
	}
}

// installPlugin registers the shipped Komoot manifest, so a request gets past
// the plugin checks to the limiter.
func (f *pluginSyncRouteFixture) installPlugin(t *testing.T) {
	t.Helper()
	manifestData, err := os.ReadFile(filepath.Join("..", "..", "plugins", "komoot", "plugin.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest pluginsystem.Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatal(err)
	}
	installed := core.NewBaseCollection("installed_plugins")
	installed.Fields.Add(&core.TextField{Name: "plugin_id"}, &core.TextField{Name: "path"},
		&core.JSONField{Name: "manifest"}, &core.JSONField{Name: "config"})
	if err := f.app.Save(installed); err != nil {
		t.Fatal(err)
	}
	pluginDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(pluginDir, manifest.Runtime.Entrypoint), []byte("\x00asm\x01\x00\x00\x00"), 0600); err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(installed)
	record.Set("plugin_id", manifest.ID)
	record.Set("path", pluginDir)
	record.Set("manifest", manifest)
	if err := f.app.Save(record); err != nil {
		t.Fatal(err)
	}
}

func TestManualSyncUsesSharedInstanceLock(t *testing.T) {
	limiter := newPluginSyncLimiter(0, 2)
	release, ok := pluginsystem.TryLockInstance(t.Name())
	if !ok {
		t.Fatal("could not acquire shared lock")
	}
	defer release()
	finished, _, err := limiter.startManual(t.Name(), t.Name())
	if err == nil {
		finished()
	}
	if !errors.Is(err, errPluginSyncRunning) {
		t.Fatalf("manual sync ignored shared lock: %v", err)
	}
	if limiter.running != 0 || len(limiter.lastStart) != 0 {
		t.Fatal("busy instance consumed a slot or cooldown")
	}
}

func TestPluginSyncCooldownSurvivesInstanceReplacement(t *testing.T) {
	limiter := newPluginSyncLimiter(time.Hour, 2)
	release, _, err := limiter.startManual("old-instance", "owner/komoot")
	if err != nil {
		t.Fatal(err)
	}
	release()
	finished, _, err := limiter.startManual("replacement-instance", "owner/komoot")
	if err == nil {
		finished()
	}
	if !errors.Is(err, errPluginSyncCooldown) {
		t.Fatalf("replacement bypassed cooldown: %v", err)
	}
}

type stalledSyncCloseSession struct {
	syncLimitTestSession
	closing chan struct{}
	release chan struct{}
}

func (s stalledSyncCloseSession) Close(ctx context.Context) error {
	close(s.closing)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.release:
		return nil
	}
}

func TestManualSyncCancellationCoversWorkerShutdown(t *testing.T) {
	f := newPluginInstanceFixture(t)
	actors := core.NewBaseCollection("activitypub_actors")
	actors.Fields.Add(&core.TextField{Name: "user"})
	if err := f.app.Save(actors); err != nil {
		t.Fatal(err)
	}
	actor := core.NewRecord(actors)
	actor.Set("user", f.ownerID)
	if err := f.app.Save(actor); err != nil {
		t.Fatal(err)
	}
	id := f.instance(t, true, time.Time{})
	closing, forceRelease := make(chan struct{}), make(chan struct{})
	session := stalledSyncCloseSession{
		syncLimitTestSession: syncLimitTestSession{call: func(context.Context, string, []byte) ([]byte, error) { return []byte("{}"), nil }},
		closing:              closing, release: forceRelease,
	}
	original := pluginRuntimeFor
	pluginRuntimeFor = func(pluginsystem.LocalPlugin) (pluginsystem.Runtime, error) {
		return &reopenTestRuntime{session: session}, nil
	}
	t.Cleanup(func() { pluginRuntimeFor = original })
	plugin := pluginsystem.LocalPlugin{}
	plugin.Manifest.ID = "komoot"
	plugin.Manifest.Capabilities = []pluginsystem.CapabilityManifest{
		{Name: "list_routes", Version: "v1", Export: "list_routes_v1"},
		{Name: "get_route_detail", Version: "v1", Export: "get_route_detail_v1"},
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _, _ = syncPluginInstanceLocked(ctx, f.app, nil, plugin, id)
	}()
	defer func() { close(forceRelease); <-done }()
	select {
	case <-closing:
	case <-time.After(5 * time.Second):
		t.Fatal("sync did not reach worker shutdown")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker shutdown kept the instance locked after sync cancellation")
	}
	if pluginsystem.InstanceBusy(id) {
		t.Fatal("cancelled sync left its instance locked")
	}
}

func TestPluginSyncLimiterCapsConcurrentRequests(t *testing.T) {
	limiter := newPluginSyncLimiter(0, 2)
	start := make(chan struct{})
	releases := make(chan func(), 20)
	var requests sync.WaitGroup
	for i := range 20 {
		requests.Go(func() {
			<-start
			id := fmt.Sprintf("%s-%d", t.Name(), i)
			release, _, err := limiter.startManual(id, id)
			if err == nil {
				releases <- release
			} else if !errors.Is(err, errPluginSyncBusy) {
				t.Errorf("unexpected refusal: %v", err)
			}
		})
	}
	close(start)
	requests.Wait()
	close(releases)
	admitted := len(releases)
	for release := range releases {
		release()
	}
	if admitted != 2 {
		t.Fatalf("concurrent requests admitted %d syncs; want 2", admitted)
	}
}
