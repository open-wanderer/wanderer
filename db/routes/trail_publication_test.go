package routes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	assetservice "pocketbase/services/assets"
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/core"
	pbtests "github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/pocketbase/pocketbase/tools/types"
)

func TestTrailPublicationStatusIdle(t *testing.T) {
	for _, tt := range []struct {
		name          string
		public        bool
		expiredStatus string
	}{
		{name: "no job"},
		{name: "already public", public: true},
		{name: "expired completed job", public: true, expiredStatus: "completed"},
		{name: "expired failed job", expiredStatus: "failed"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newPublicationRouteFixture(t)
			f.trail.Set("public", tt.public)
			f.save(t, f.trail)
			if tt.expiredStatus != "" {
				f.manager.jobs[f.trail.Id] = &trailPublicationEntry{
					job: TrailPublicationJob{
						ID: "expired", TrailID: f.trail.Id, Status: tt.expiredStatus,
						UpdatedAt: time.Now().Add(-3 * time.Hour),
					},
					userID: f.owner.Id,
				}
			}
			f.requestIdle(t, f.owner, f.trail.Id)
			assertPublicationPrivacy(t, f.app, f.trail.Id, tt.public)
		})
	}
}

func TestTrailPublicationStatusIdleRequiresTrailAndPermission(t *testing.T) {
	f := newPublicationRouteFixture(t)
	f.requestError(t, f.owner, http.MethodGet, "missing", http.StatusNotFound, "")
	f.requestError(t, nil, http.MethodGet, f.trail.Id, http.StatusUnauthorized, "")
	f.requestError(t, f.stranger, http.MethodGet, f.trail.Id, http.StatusForbidden, "")
}

func TestTrailPublicationBackgroundProgressAndReuse(t *testing.T) {
	f := newPublicationRouteFixture(t)
	started, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	f.manager.prepare = func(ctx context.Context, _ core.App, trailID string, progress assetservice.ProgressFunc) error {
		calls.Add(1)
		if trailID != f.trail.Id {
			return fmt.Errorf("unexpected trail %q", trailID)
		}
		progress(3, 1, 0)
		close(started)
		select {
		case <-release:
			progress(3, 3, 0)
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	requestCtx, cancelRequest := context.WithCancel(context.Background())
	defer cancelRequest()
	job := f.requestJob(t, requestCtx, f.owner, http.MethodPost, f.trail.Id)
	cancelRequest()
	awaitPublicationSignal(t, started)
	assertPublicationPrivacy(t, f.app, f.trail.Id, false)
	if job.Status != "running" {
		t.Fatalf("start status = %q, want running", job.Status)
	}
	duplicate := f.requestJob(t, context.Background(), f.owner, http.MethodPost, f.trail.Id)
	if duplicate.ID != job.ID || calls.Load() != 1 {
		t.Fatalf("duplicate request = %#v, prepare calls = %d", duplicate, calls.Load())
	}
	status := f.requestJob(t, context.Background(), f.owner, http.MethodGet, f.trail.Id)
	if status.Total != 3 || status.Processed != 1 || status.Failed != 0 || status.Status != "running" {
		t.Fatalf("progress = %#v", status)
	}
	// Both users can edit; only the requester can see the job or reuse it.
	f.requestIdle(t, f.editor, f.trail.Id)
	f.requestError(t, f.editor, http.MethodPost, f.trail.Id, http.StatusConflict, "asset_publish_in_progress")
	f.requestError(t, nil, http.MethodGet, f.trail.Id, http.StatusUnauthorized, "")
	f.requestError(t, nil, http.MethodPost, f.trail.Id, http.StatusUnauthorized, "")
	f.requestError(t, f.stranger, http.MethodPost, f.trail.Id, http.StatusForbidden, "")
	f.requestError(t, f.stranger, http.MethodGet, f.trail.Id, http.StatusForbidden, "")
	// A consumer cannot mutate manager-owned status through a returned snapshot.
	snapshot := f.manager.snapshot(f.trail.Id, f.owner.Id)
	snapshot.Processed = 100
	if current := f.manager.snapshot(f.trail.Id, f.owner.Id); current.Processed != 1 {
		t.Fatalf("snapshot changed manager state: %#v", current)
	}
	close(release)
	completed := waitPublicationTerminal(t, f.manager, f.trail.Id, f.owner.Id)
	if completed.Status != "completed" || completed.Processed != 3 || completed.Error != "" {
		t.Fatalf("completed job = %#v", completed)
	}
	assertPublicationPrivacy(t, f.app, f.trail.Id, true)
}

func TestTrailPublicationFailureIsSafeAndRetryable(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		code string
	}{
		{"provider", errors.New("fetch https://photos.example/private?token=secret failed"), "asset_publish_failed"},
		{"oversized", util.ErrPluginMediaTooLarge, "asset_publish_photo_too_large"},
		{"timeout", context.DeadlineExceeded, "asset_publish_failed"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newPublicationRouteFixture(t)
			var calls atomic.Int32
			f.manager.prepare = func(_ context.Context, _ core.App, _ string, progress assetservice.ProgressFunc) error {
				if calls.Add(1) == 1 {
					progress(2, 2, 1)
					return tt.err
				}
				progress(1, 1, 0)
				return nil
			}
			first := f.requestJob(t, context.Background(), f.owner, http.MethodPost, f.trail.Id)
			failed := waitPublicationTerminal(t, f.manager, f.trail.Id, f.owner.Id)
			if failed.Status != "failed" || failed.Error != tt.code || failed.Failed != 1 || strings.Contains(failed.Error, "secret") {
				t.Fatalf("failed job = %#v", failed)
			}
			assertPublicationPrivacy(t, f.app, f.trail.Id, false)
			retry := f.requestJob(t, context.Background(), f.owner, http.MethodPost, f.trail.Id)
			if retry.ID == first.ID {
				t.Fatal("retry reused terminal job")
			}
			completed := waitPublicationTerminal(t, f.manager, f.trail.Id, f.owner.Id)
			if completed.Status != "completed" || completed.Failed != 0 || completed.Total != 1 {
				t.Fatalf("retry job = %#v", completed)
			}
			assertPublicationPrivacy(t, f.app, f.trail.Id, true)
		})
	}
}

func TestTrailPublicationRechecksBeforePublishing(t *testing.T) {
	for _, tt := range []struct {
		name    string
		change  func(*testing.T, *publicationRouteFixture)
		code    string
		deleted bool
	}{
		{
			name: "trail changed", code: "asset_publish_changed",
			change: func(t *testing.T, f *publicationRouteFixture) {
				trail := f.reload(t, "trails", f.trail.Id)
				trail.Set("updated", "2026-09-16 12:00:01.000Z")
				f.save(t, trail)
			},
		},
		{
			name: "current user permission revoked", code: "asset_publish_failed",
			change: func(t *testing.T, f *publicationRouteFixture) {
				user := f.reload(t, "publication_users", f.owner.Id)
				user.Set("canPublish", false)
				f.save(t, user)
			},
		},
		{
			name: "trail edit permission revoked", code: "asset_publish_failed",
			change: func(t *testing.T, f *publicationRouteFixture) {
				trail := f.reload(t, "trails", f.trail.Id)
				trail.Set("author", f.stranger.Id)
				f.save(t, trail)
			},
		},
		{
			name: "requesting user deleted", code: "asset_publish_failed",
			change: func(t *testing.T, f *publicationRouteFixture) {
				if err := f.app.Delete(f.owner); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "trail deleted", code: "asset_publish_failed", deleted: true,
			change: func(t *testing.T, f *publicationRouteFixture) {
				if err := f.app.Delete(f.trail); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "remote photo linked during download", code: "asset_publish_required",
			change: func(t *testing.T, f *publicationRouteFixture) {
				asset := f.record(t, "assets", map[string]any{"type": "photo", "storage_mode": "link_private"})
				f.record(t, "trail_assets", map[string]any{"trail": f.trail.Id, "asset": asset.Id})
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newPublicationRouteFixture(t)
			started, release := make(chan struct{}), make(chan struct{})
			f.manager.prepare = func(ctx context.Context, _ core.App, _ string, _ assetservice.ProgressFunc) error {
				close(started)
				select {
				case <-release:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			f.requestJob(t, context.Background(), f.owner, http.MethodPost, f.trail.Id)
			awaitPublicationSignal(t, started)
			tt.change(t, f)
			close(release)
			job := waitPublicationTerminal(t, f.manager, f.trail.Id, f.owner.Id)
			if job.Status != "failed" || job.Error != tt.code {
				t.Fatalf("job = %#v, want failed/%s", job, tt.code)
			}
			if !tt.deleted {
				assertPublicationPrivacy(t, f.app, f.trail.Id, false)
			}
		})
	}
}

func TestTrailPublicationWorkerBoundAndShutdown(t *testing.T) {
	f := newPublicationRouteFixture(t)
	started := make(chan string, 3)
	f.manager.prepare = func(ctx context.Context, _ core.App, trailID string, _ assetservice.ProgressFunc) error {
		started <- trailID
		<-ctx.Done()
		return ctx.Err()
	}
	trailIDs := []string{f.trail.Id}
	for i := 0; i < 2; i++ {
		trail := f.record(t, "trails", map[string]any{"author": f.owner.Id, "public": false, "updated": "2026-09-16 12:00:00.000Z"})
		trailIDs = append(trailIDs, trail.Id)
	}
	for _, trailID := range trailIDs {
		f.requestJob(t, context.Background(), f.owner, http.MethodPost, trailID)
	}
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Fatal("publication worker did not start")
		}
	}
	select {
	case trailID := <-started:
		t.Fatalf("third worker started for %s while both slots were occupied", trailID)
	case <-time.After(30 * time.Millisecond):
	}
	f.manager.stop()
	for _, trailID := range trailIDs {
		job := f.manager.snapshot(trailID, f.owner.Id)
		if job.Status != "failed" || job.Error != "asset_publish_interrupted" {
			t.Fatalf("shutdown job = %#v", job)
		}
		assertPublicationPrivacy(t, f.app, trailID, false)
	}
	f.requestError(t, f.owner, http.MethodPost, f.trail.Id, http.StatusServiceUnavailable, "asset_publish_interrupted")
}

func TestTrailPublicationReportsCommittedSaveAfterFollowupFailure(t *testing.T) {
	f := newPublicationRouteFixture(t)
	f.manager.prepare = func(context.Context, core.App, string, assetservice.ProgressFunc) error {
		return nil
	}
	f.app.OnRecordAfterUpdateSuccess("trails").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetBool("public") {
			return errors.New("search indexing temporarily unavailable")
		}
		return e.Next()
	})
	f.requestJob(t, context.Background(), f.owner, http.MethodPost, f.trail.Id)
	job := waitPublicationTerminal(t, f.manager, f.trail.Id, f.owner.Id)
	if job.Status != "completed" || job.Error != "" {
		t.Fatalf("committed publication reported as failed: %#v", job)
	}
	assertPublicationPrivacy(t, f.app, f.trail.Id, true)
}

func TestTrailPublicationAlreadyPublicSkipsPreparation(t *testing.T) {
	f := newPublicationRouteFixture(t)
	trail := f.reload(t, "trails", f.trail.Id)
	trail.Set("public", true)
	f.save(t, trail)
	// Existing public trails may still contain remote links from older builds.
	asset := f.record(t, "assets", map[string]any{"type": "photo", "storage_mode": "link_private"})
	f.record(t, "trail_assets", map[string]any{"trail": trail.Id, "asset": asset.Id})
	var calls atomic.Int32
	f.manager.prepare = func(context.Context, core.App, string, assetservice.ProgressFunc) error {
		calls.Add(1)
		return errors.New("legacy photo source unavailable")
	}
	f.requestJob(t, context.Background(), f.owner, http.MethodPost, trail.Id)
	job := waitPublicationTerminal(t, f.manager, trail.Id, f.owner.Id)
	if job.Status != "completed" || job.Error != "" || calls.Load() != 0 {
		t.Fatalf("already public job = %#v, prepare calls = %d", job, calls.Load())
	}
	assertPublicationPrivacy(t, f.app, trail.Id, true)
}

func TestTrailPublicationConcurrentPublicationCompletes(t *testing.T) {
	f := newPublicationRouteFixture(t)
	started, release := make(chan struct{}), make(chan struct{})
	f.manager.prepare = func(ctx context.Context, _ core.App, _ string, _ assetservice.ProgressFunc) error {
		close(started)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	f.requestJob(t, context.Background(), f.owner, http.MethodPost, f.trail.Id)
	awaitPublicationSignal(t, started)
	trail := f.reload(t, "trails", f.trail.Id)
	trail.Set("public", true)
	trail.Set("updated", "2026-09-16 12:00:01.000Z")
	f.save(t, trail)
	close(release)
	job := waitPublicationTerminal(t, f.manager, f.trail.Id, f.owner.Id)
	if job.Status != "completed" || job.Error != "" {
		t.Fatalf("concurrent publication reported as failed: %#v", job)
	}
	assertPublicationPrivacy(t, f.app, f.trail.Id, true)
}

func TestTrailPublicationRechecksQueuedWorkBeforeDownload(t *testing.T) {
	f := newPublicationRouteFixture(t)
	var calls atomic.Int32
	f.manager.prepare = func(context.Context, core.App, string, assetservice.ProgressFunc) error {
		calls.Add(1)
		return nil
	}
	// Occupy both slots so the trail changes after authorization but before work.
	f.manager.workers <- struct{}{}
	f.manager.workers <- struct{}{}
	f.requestJob(t, context.Background(), f.owner, http.MethodPost, f.trail.Id)
	trail := f.reload(t, "trails", f.trail.Id)
	trail.Set("updated", "2026-09-16 12:00:01.000Z")
	f.save(t, trail)
	<-f.manager.workers
	job := waitPublicationTerminal(t, f.manager, f.trail.Id, f.owner.Id)
	if job.Status != "failed" || job.Error != "asset_publish_changed" || calls.Load() != 0 {
		t.Fatalf("queued job = %#v, prepare calls = %d", job, calls.Load())
	}
	assertPublicationPrivacy(t, f.app, f.trail.Id, false)
}

func TestTrailPublicationManagerConcurrentReuseAndRetention(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	manager := &trailPublicationManager{jobs: make(map[string]*trailPublicationEntry), ctx: ctx, cancel: cancel}
	var wg sync.WaitGroup
	results := make(chan TrailPublicationJob, 12)
	errors := make(chan error, 12)
	var starts atomic.Int32
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			job, start, err := manager.start("trail", "owner")
			if err != nil {
				errors <- err
				return
			}
			if start {
				starts.Add(1)
				manager.wg.Done() // This manager-only test does not launch a worker.
			}
			results <- job
		}()
	}
	wg.Wait()
	close(results)
	close(errors)
	for err := range errors {
		t.Fatal(err)
	}
	if starts.Load() != 1 {
		t.Fatalf("workers started = %d, want 1", starts.Load())
	}
	var id string
	for job := range results {
		if id != "" && job.ID != id {
			t.Fatalf("concurrent POSTs returned different jobs: %q and %q", id, job.ID)
		}
		id = job.ID
	}
	old := time.Now().Add(-3 * time.Hour)
	manager.jobs["trail"].job.UpdatedAt = old
	manager.jobs["expired"] = &trailPublicationEntry{job: TrailPublicationJob{TrailID: "expired", Status: "failed", UpdatedAt: old}, userID: "owner"}
	if manager.snapshot("expired", "owner") != nil || manager.snapshot("trail", "owner") == nil {
		t.Fatal("retention expired an active job or retained an expired terminal job")
	}
	for i := 0; i < maxRemotePluginAssetJobsPerUser-1; i++ {
		if _, start, err := manager.start(fmt.Sprintf("another-%d", i), "owner"); err != nil || !start {
			t.Fatalf("starting allowed job: start=%t error=%v", start, err)
		}
		manager.wg.Done()
	}
	if manager.jobs["expired"] != nil || manager.jobs["trail"] == nil {
		t.Fatal("pruning removed running job or kept expired terminal job")
	}
	_, _, err := manager.start("over-limit", "owner")
	assertPublicationAPIError(t, err, http.StatusTooManyRequests, "asset_publish_busy")
	if duplicate, start, err := manager.start("trail", "owner"); err != nil || start || duplicate.ID != id {
		t.Fatalf("reuse at user cap = %#v, start=%t error=%v", duplicate, start, err)
	}
}

type publicationRouteFixture struct {
	app                     *pbtests.TestApp
	manager                 *trailPublicationManager
	owner, editor, stranger *core.Record
	trail                   *core.Record
}

func newPublicationRouteFixture(t *testing.T) *publicationRouteFixture {
	t.Helper()
	app, err := pbtests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	f := &publicationRouteFixture{app: app}
	users := core.NewAuthCollection("publication_users")
	users.Fields.Add(&core.BoolField{Name: "canPublish"})
	f.save(t, users)
	newUser := func(email string) *core.Record {
		user := core.NewRecord(users)
		user.Set("email", email)
		user.Set("canPublish", true)
		user.SetPassword("test-password-123")
		f.save(t, user)
		return user
	}
	f.owner = newUser("owner@example.com")
	f.editor = newUser("editor@example.com")
	f.stranger = newUser("stranger@example.com")
	collections := map[string][]core.Field{
		"trails":            {&core.TextField{Name: "author"}, &core.TextField{Name: "editor"}, &core.BoolField{Name: "public"}, &core.DateField{Name: "updated"}},
		"waypoints":         {&core.TextField{Name: "trail"}},
		"summit_logs":       {&core.TextField{Name: "trail"}},
		"assets":            {&core.TextField{Name: "type"}, &core.TextField{Name: "storage_mode"}, &core.AutodateField{Name: "created", OnCreate: true}},
		"trail_assets":      {&core.TextField{Name: "trail"}, &core.TextField{Name: "asset"}},
		"waypoint_assets":   {&core.TextField{Name: "waypoint"}, &core.TextField{Name: "asset"}},
		"summit_log_assets": {&core.TextField{Name: "summit_log"}, &core.TextField{Name: "asset"}},
	}
	for name, fields := range collections {
		collection := core.NewBaseCollection(name)
		collection.Fields.Add(fields...)
		if name == "trails" {
			collection.UpdateRule = types.Pointer(`@request.auth.canPublish = true && (author = @request.auth.id || editor = @request.auth.id) && @request.body.public = true`)
		}
		f.save(t, collection)
	}
	f.trail = f.record(t, "trails", map[string]any{"author": f.owner.Id, "editor": f.editor.Id, "public": false, "updated": "2026-09-16 12:00:00.000Z"})
	f.manager = publicationManager(app)
	t.Cleanup(f.manager.stop)
	return f
}

func (f *publicationRouteFixture) save(t *testing.T, model core.Model) {
	t.Helper()
	if err := f.app.Save(model); err != nil {
		t.Fatal(err)
	}
}

func (f *publicationRouteFixture) record(t *testing.T, collectionName string, data map[string]any) *core.Record {
	t.Helper()
	collection, err := f.app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(collection)
	for field, value := range data {
		record.Set(field, value)
	}
	f.save(t, record)
	return record
}

func (f *publicationRouteFixture) reload(t *testing.T, collection, id string) *core.Record {
	t.Helper()
	record, err := f.app.FindRecordById(collection, id)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func (f *publicationRouteFixture) request(ctx context.Context, auth *core.Record, method, trailID string) (*httptest.ResponseRecorder, error) {
	e := &core.RequestEvent{App: f.app, Auth: auth}
	e.Request = httptest.NewRequestWithContext(ctx, method, "/trails/"+trailID+"/publication", nil)
	e.Request.SetPathValue("id", trailID)
	response := httptest.NewRecorder()
	e.Response = response
	if method == http.MethodPost {
		return response, TrailPublicationStart(e)
	}
	return response, TrailPublicationStatus(e)
}

func (f *publicationRouteFixture) requestJob(t *testing.T, ctx context.Context, auth *core.Record, method, trailID string) TrailPublicationJob {
	t.Helper()
	response, err := f.request(ctx, auth, method, trailID)
	if err != nil {
		t.Fatalf("%s publication: %v", method, err)
	}
	wantStatus := http.StatusOK
	if method == http.MethodPost {
		wantStatus = http.StatusAccepted
	}
	if response.Code != wantStatus {
		t.Fatalf("%s publication status = %d, body=%s", method, response.Code, response.Body.String())
	}
	var job TrailPublicationJob
	if err := json.Unmarshal(response.Body.Bytes(), &job); err != nil {
		t.Fatal(err)
	}
	return job
}

func (f *publicationRouteFixture) requestError(t *testing.T, auth *core.Record, method, trailID string, status int, code string) {
	t.Helper()
	_, err := f.request(context.Background(), auth, method, trailID)
	assertPublicationAPIError(t, err, status, code)
}

func (f *publicationRouteFixture) requestIdle(t *testing.T, auth *core.Record, trailID string) {
	t.Helper()
	response, err := f.request(context.Background(), auth, http.MethodGet, trailID)
	if err != nil {
		t.Fatalf("GET publication: %v", err)
	}
	if response.Code != http.StatusOK {
		t.Fatalf("GET publication status = %d, body=%s", response.Code, response.Body.String())
	}
	var status map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if len(status) != 2 || status["trailId"] != trailID || status["status"] != "idle" {
		t.Fatalf("idle status = %#v, want only trailId=%q and status=idle", status, trailID)
	}
}

func assertPublicationAPIError(t *testing.T, err error, status int, code string) {
	t.Helper()
	var apiErr *router.ApiError
	if !errors.As(err, &apiErr) || apiErr.Status != status || (code != "" && apiErr.Message != code) {
		t.Fatalf("API error = %v, want status=%d code=%q", err, status, code)
	}
}

func assertPublicationPrivacy(t *testing.T, app core.App, trailID string, public bool) {
	t.Helper()
	trail, err := app.FindRecordById("trails", trailID)
	if err != nil || trail.GetBool("public") != public {
		t.Fatalf("trail privacy: record=%v error=%v want public=%t", trail, err, public)
	}
}

func awaitPublicationSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatal("publication worker did not reach checkpoint")
	}
}

func waitPublicationTerminal(t *testing.T, manager *trailPublicationManager, trailID, userID string) *TrailPublicationJob {
	t.Helper()
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		job := manager.snapshot(trailID, userID)
		if job != nil && job.Status != "running" {
			return job
		}
		select {
		case <-timeout.C:
			t.Fatalf("publication did not finish: %#v", job)
		case <-ticker.C:
		}
	}
}
