package routes

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"sync"
	"time"

	assetservice "pocketbase/services/assets"
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

type TrailPublicationJob struct {
	ID        string    `json:"id"`
	TrailID   string    `json:"trailId"`
	Status    string    `json:"status"`
	Total     int       `json:"total"`
	Processed int       `json:"processed"`
	Failed    int       `json:"failed"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type trailPublicationEntry struct {
	job    TrailPublicationJob
	userID string
}

// Status is process-local, while each completed photo is saved independently.
// After a restart a new publication request resumes from the remaining photos.
type trailPublicationManager struct {
	mu      sync.Mutex
	jobs    map[string]*trailPublicationEntry
	workers chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	prepare func(context.Context, core.App, string, assetservice.ProgressFunc) error
}

func publicationManager(app core.App) *trailPublicationManager {
	return app.Store().GetOrSet("trailPublicationManager", func() any {
		ctx, cancel := context.WithCancel(context.Background())
		manager := &trailPublicationManager{
			jobs: make(map[string]*trailPublicationEntry), workers: make(chan struct{}, 2), ctx: ctx, cancel: cancel,
			prepare: assetservice.MaterializePrivateRemotePluginAssetsForTrailWithProgress,
		}
		app.OnTerminate().BindFunc(func(e *core.TerminateEvent) error {
			manager.stop()
			return e.Next()
		})
		return manager
	}).(*trailPublicationManager)
}

func (manager *trailPublicationManager) stop() {
	manager.mu.Lock()
	manager.cancel()
	manager.mu.Unlock()
	manager.wg.Wait()
}

func TrailPublicationStart(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}
	info, err := e.RequestInfo()
	if err != nil {
		return err
	}
	info = info.Clone()
	info.Method = http.MethodPatch
	info.Body = map[string]any{"public": true}
	trailID := e.Request.PathValue("id")
	trail, err := authorizedPublicationTrail(e.App, trailID, info)
	if err != nil {
		return err
	}
	manager := publicationManager(e.App)
	job, start, err := manager.start(trailID, e.Auth.Id)
	if err != nil {
		return err
	}
	if start {
		// Copy all request-owned state before responding. Work deliberately
		// outlives this HTTP request and never uses its canceled context.
		app := e.App
		updated := trail.GetString("updated")
		go manager.run(app, job, info, updated)
	}
	return e.JSON(http.StatusAccepted, job)
}

func TrailPublicationStatus(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}
	info, err := e.RequestInfo()
	if err != nil {
		return err
	}
	info = info.Clone()
	info.Method = http.MethodPatch
	info.Body = map[string]any{"public": true}
	trailID := e.Request.PathValue("id")
	if _, err := authorizedPublicationTrail(e.App, trailID, info); err != nil {
		return err
	}
	job := publicationManager(e.App).snapshot(trailID, e.Auth.Id)
	if job == nil {
		return e.NotFoundError("publication job not found", nil)
	}
	return e.JSON(http.StatusOK, job)
}

func authorizedPublicationTrail(app core.App, trailID string, info *core.RequestInfo) (*core.Record, error) {
	trail, err := app.FindRecordById("trails", trailID)
	if err != nil {
		return nil, apis.NewNotFoundError("trail not found", err)
	}
	allowed, err := app.CanAccessRecord(trail, info, trail.Collection().UpdateRule)
	if err != nil || !allowed {
		return nil, apis.NewForbiddenError("Insufficient permissions for trail", err)
	}
	return trail, nil
}

func (manager *trailPublicationManager) start(trailID, userID string) (TrailPublicationJob, bool, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if manager.ctx.Err() != nil {
		return TrailPublicationJob{}, false, publicationAPIError(http.StatusServiceUnavailable, "asset_publish_interrupted", nil)
	}
	now := time.Now().UTC()
	active := 0
	for id, entry := range manager.jobs {
		if entry.job.Status != "running" && now.Sub(entry.job.UpdatedAt) > 2*time.Hour {
			delete(manager.jobs, id)
			continue
		}
		if entry.job.Status == "running" && entry.userID == userID {
			active++
		}
	}
	if existing := manager.jobs[trailID]; existing != nil && existing.job.Status == "running" {
		if existing.userID != userID {
			return TrailPublicationJob{}, false, publicationAPIError(http.StatusConflict, "asset_publish_in_progress", nil)
		}
		return existing.job, false, nil
	}
	if active >= maxRemotePluginAssetJobsPerUser {
		return TrailPublicationJob{}, false, publicationAPIError(http.StatusTooManyRequests, "asset_publish_busy", nil)
	}
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return TrailPublicationJob{}, false, err
	}
	job := TrailPublicationJob{
		ID: hex.EncodeToString(id[:]), TrailID: trailID, Status: "running", CreatedAt: now, UpdatedAt: now,
	}
	manager.jobs[trailID] = &trailPublicationEntry{job: job, userID: userID}
	manager.wg.Add(1)
	return job, true, nil
}

func (manager *trailPublicationManager) snapshot(trailID, userID string) *TrailPublicationJob {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	entry := manager.jobs[trailID]
	if entry == nil || entry.userID != userID || (entry.job.Status != "running" && time.Since(entry.job.UpdatedAt) > 2*time.Hour) {
		return nil
	}
	job := entry.job
	return &job
}

func (manager *trailPublicationManager) update(job TrailPublicationJob, update func(*TrailPublicationJob)) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	entry := manager.jobs[job.TrailID]
	if entry == nil || entry.job.ID != job.ID {
		return
	}
	update(&entry.job)
	entry.job.UpdatedAt = time.Now().UTC()
}

func (manager *trailPublicationManager) run(app core.App, job TrailPublicationJob, info *core.RequestInfo, updated string) {
	defer manager.wg.Done()
	err := manager.publish(app, job, info, updated)
	manager.update(job, func(status *TrailPublicationJob) {
		if err == nil {
			status.Status = "completed"
			return
		}
		app.Logger().Warn("trail publication failed", "trail", job.TrailID, "error", err)
		status.Status = "failed"
		status.Error = publicationErrorCode(err)
	})
}

func (manager *trailPublicationManager) publish(app core.App, job TrailPublicationJob, info *core.RequestInfo, updated string) error {
	select {
	case manager.workers <- struct{}{}:
		defer func() { <-manager.workers }()
	case <-manager.ctx.Done():
		return manager.ctx.Err()
	}
	// Queued work may start after permissions or the trail have changed.
	trail, err := currentPublicationTrail(app, job.TrailID, info, updated)
	if err != nil {
		return err
	}
	if trail.GetBool("public") {
		return nil
	}
	if err := manager.prepare(manager.ctx, app, job.TrailID, func(total, processed, failed int) {
		manager.update(job, func(status *TrailPublicationJob) {
			status.Total, status.Processed, status.Failed = total, processed, failed
		})
	}); err != nil {
		return err
	}
	if err := manager.ctx.Err(); err != nil {
		return err
	}
	// Keep network transfers out of the transaction. Re-read both permissions
	// and linked assets while serializing the final privacy transition.
	committed := false
	err = app.RunInTransaction(func(txApp core.App) error {
		txApp.TxInfo().OnComplete(func(txErr error) error {
			committed = txErr == nil
			return nil
		})
		trail, err := currentPublicationTrail(txApp, job.TrailID, info, updated)
		if err != nil {
			return err
		}
		if trail.GetBool("public") {
			return nil
		}
		if err := assetservice.EnsureTrailAssetsMaterialized(txApp, job.TrailID); err != nil {
			return err
		}
		trail.Set("public", true)
		return txApp.SaveWithContext(manager.ctx, trail)
	})
	if err != nil && committed {
		// After-success hooks (indexing/federation) can fail after commit. The
		// trail is already public; do not misreport it as a failed private draft.
		app.Logger().Warn("trail published but follow-up processing failed", "trail", job.TrailID, "error", err)
		return nil
	}
	return err
}

func currentPublicationTrail(app core.App, trailID string, info *core.RequestInfo, updated string) (*core.Record, error) {
	auth, err := app.FindRecordById(info.Auth.Collection().Name, info.Auth.Id)
	if err != nil {
		return nil, apis.NewForbiddenError("publication permission revoked", err)
	}
	currentInfo := info.Clone()
	currentInfo.Auth = auth
	trail, err := authorizedPublicationTrail(app, trailID, currentInfo)
	if err != nil {
		return nil, err
	}
	if !trail.GetBool("public") && trail.GetString("updated") != updated {
		return nil, publicationAPIError(http.StatusConflict, "asset_publish_changed", nil)
	}
	return trail, nil
}

func publicationErrorCode(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "asset_publish_interrupted"
	case errors.Is(err, util.ErrPluginMediaTooLarge):
		return "asset_publish_photo_too_large"
	case errors.Is(err, assetservice.ErrTrailMaterializationRequired):
		return "asset_publish_required"
	}
	var apiErr *router.ApiError
	if errors.As(err, &apiErr) && apiErr.Message == "asset_publish_changed" {
		return apiErr.Message
	}
	// Provider errors may contain signed URLs or credentials.
	return "asset_publish_failed"
}

func publicationAPIError(status int, code string, cause error) *router.ApiError {
	err := apis.NewApiError(status, code, cause)
	err.Message = code
	return err
}
