package routes

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"pocketbase/pluginsystem"
	assetservice "pocketbase/services/assets"
	"pocketbase/util"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

type RemotePluginAssetsJob struct {
	ID        string    `json:"id"`
	UserID    string    `json:"-"`
	PluginID  string    `json:"pluginId"`
	Kind      string    `json:"kind"`
	Status    string    `json:"status"`
	Total     int       `json:"total"`
	Processed int       `json:"processed"`
	Failed    int       `json:"failed"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Remote plugin asset jobs are intentionally process-local. They provide
// lightweight status reporting for the single-instance deployment model; jobs
// are not durable across restarts and are not shared between app instances.
var remotePluginAssetJobs = struct {
	sync.Mutex
	jobs map[string]*RemotePluginAssetsJob
}{jobs: map[string]*RemotePluginAssetsJob{}}

const maxRemotePluginAssetJobsPerUser = 5

func PluginSystemAssetRemoteAssetsSummary(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}
	pluginID := e.Request.PathValue("plugin")
	if err := ensureAssetPlugin(e.App, pluginID); err != nil {
		return err
	}
	summary, err := assetservice.RemotePluginAssetsSummaryForUser(e.App, e.Auth.Id, pluginID)
	if err != nil {
		return err
	}
	return e.JSON(http.StatusOK, summary)
}

func PluginSystemAssetMaterializeAll(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}
	pluginID := e.Request.PathValue("plugin")
	if err := ensureAssetPlugin(e.App, pluginID); err != nil {
		return err
	}
	var data struct {
		PublicOnly bool `json:"publicOnly"`
	}
	_ = e.BindBody(&data)
	job, err := newRemotePluginAssetsJob(e.Auth.Id, pluginID, "materialize")
	if err != nil {
		return e.InternalServerError("Failed to start asset materialization job", err)
	}
	app := e.App
	userID := e.Auth.Id
	go func() {
		if err := assetservice.MaterializeRemotePluginAssetsForUser(context.Background(), app, userID, pluginID, data.PublicOnly, func(total, processed, failed int) {
			updateRemotePluginAssetsJob(job.ID, func(job *RemotePluginAssetsJob) {
				job.Total = total
				job.Processed = processed
				job.Failed = failed
			})
		}); err != nil {
			app.Logger().Warn("failed to materialize remote plugin assets", "user", userID, "plugin", pluginID, "error", err)
			updateRemotePluginAssetsJob(job.ID, func(job *RemotePluginAssetsJob) {
				job.Status = "failed"
				job.Error = err.Error()
			})
			return
		}
		updateRemotePluginAssetsJob(job.ID, func(job *RemotePluginAssetsJob) {
			job.Status = "completed"
		})
	}()
	return e.JSON(http.StatusAccepted, job)
}

func PluginSystemAssetRepairRemoteAssets(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}
	pluginID := e.Request.PathValue("plugin")
	if err := ensureAssetPlugin(e.App, pluginID); err != nil {
		return err
	}
	job, err := newRemotePluginAssetsJob(e.Auth.Id, pluginID, "repair")
	if err != nil {
		return e.InternalServerError("Failed to start asset repair job", err)
	}
	app := e.App
	userID := e.Auth.Id
	go func() {
		if err := assetservice.RepairRemotePluginAssetsForUser(context.Background(), app, userID, pluginID, func(total, processed, failed int) {
			updateRemotePluginAssetsJob(job.ID, func(job *RemotePluginAssetsJob) {
				job.Total = total
				job.Processed = processed
				job.Failed = failed
			})
		}); err != nil {
			app.Logger().Warn("failed to repair remote plugin assets", "user", userID, "plugin", pluginID, "error", err)
			updateRemotePluginAssetsJob(job.ID, func(job *RemotePluginAssetsJob) {
				job.Status = "failed"
				job.Error = err.Error()
			})
			return
		}
		updateRemotePluginAssetsJob(job.ID, func(job *RemotePluginAssetsJob) {
			job.Status = "completed"
		})
	}()
	return e.JSON(http.StatusAccepted, job)
}

func PluginSystemAssetDeleteRemoteAssets(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}
	pluginID := e.Request.PathValue("plugin")
	if err := ensureAssetPlugin(e.App, pluginID); err != nil {
		return err
	}
	job, err := newRemotePluginAssetsJob(e.Auth.Id, pluginID, "delete")
	if err != nil {
		return e.InternalServerError("Failed to start asset delete job", err)
	}
	app := e.App
	userID := e.Auth.Id
	go func() {
		if err := assetservice.DeleteRemotePluginAssetsForUser(context.Background(), app, userID, pluginID, func(total, processed, failed int) {
			updateRemotePluginAssetsJob(job.ID, func(job *RemotePluginAssetsJob) {
				job.Total = total
				job.Processed = processed
				job.Failed = failed
			})
		}); err != nil {
			app.Logger().Warn("failed to delete remote plugin assets", "user", userID, "plugin", pluginID, "error", err)
			updateRemotePluginAssetsJob(job.ID, func(job *RemotePluginAssetsJob) {
				job.Status = "failed"
				job.Error = err.Error()
			})
			return
		}
		updateRemotePluginAssetsJob(job.ID, func(job *RemotePluginAssetsJob) {
			job.Status = "completed"
		})
	}()
	return e.JSON(http.StatusAccepted, job)
}

func PluginSystemAssetMaterializeStatus(e *core.RequestEvent) error {
	if e.Auth == nil {
		return apis.NewUnauthorizedError("authentication required", nil)
	}
	job := remotePluginAssetsJobSnapshot(e.Request.PathValue("id"))
	if job == nil || job.UserID != e.Auth.Id {
		return e.NotFoundError("asset materialization job not found", nil)
	}
	return e.JSON(http.StatusOK, job)
}

func AssetFile(e *core.RequestEvent) error {
	assetID := e.Request.PathValue("id")
	if assetID == "" {
		return e.NotFoundError("", nil)
	}

	asset, err := e.App.FindRecordById("assets", assetID)
	if err != nil {
		return e.NotFoundError("", err)
	}

	requestInfo, err := e.RequestInfo()
	if err != nil {
		return e.InternalServerError("Failed to load request info", err)
	}
	if ok, _ := e.App.CanAccessRecord(asset, requestInfo, asset.Collection().ViewRule); !ok {
		return e.NotFoundError("", errors.New("insufficient permissions to access the asset"))
	}

	if fileURL := util.AssetFileRedirectURL(asset); fileURL != "" {
		return e.Redirect(http.StatusFound, fileURL)
	}

	storageMode := asset.GetString("storage_mode")
	if !assetservice.IsRemotePhotoStorageMode(storageMode) {
		return e.NotFoundError("", nil)
	}
	linkedToPublicTrail := util.IsAssetLinkedToPublicTrail(e.App, asset)
	if storageMode == "link_private" && linkedToPublicTrail {
		return e.NotFoundError("", nil)
	}

	fetched, err := assetservice.FetchRemotePluginAsset(e.Request.Context(), e.App, asset, util.DefaultPluginMediaMaxBytes)
	if err != nil {
		if markErr := assetservice.MarkAssetRemoteStatus(e.App, asset, assetservice.RemoteStatusForError(err), err); markErr != nil {
			e.App.Logger().Warn("failed to update remote asset status", "asset", asset.Id, "error", markErr)
		}
		return e.NotFoundError("", err)
	}
	if err := assetservice.MarkAssetRemoteStatus(e.App, asset, "available", nil); err != nil {
		e.App.Logger().Warn("failed to update remote asset status", "asset", asset.Id, "error", err)
	}
	contentType := fetched.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	e.Response.Header().Set("Cache-Control", "private, max-age=300")
	return e.Blob(http.StatusOK, contentType, fetched.Body)
}

func ensureAssetPlugin(app core.App, pluginID string) error {
	if pluginID == "" {
		return apis.NewBadRequestError("plugin is required", nil)
	}
	plugin, err := localPlugin(app, pluginID)
	if err != nil {
		return err
	}
	if plugin.Manifest.Type != pluginsystem.PluginTypeAssets {
		return apis.NewBadRequestError("plugin is not an asset plugin", nil)
	}
	return nil
}

func newRemotePluginAssetsJob(userID string, pluginID string, kind string) (*RemotePluginAssetsJob, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	job := &RemotePluginAssetsJob{
		ID:        hex.EncodeToString(b[:]),
		UserID:    userID,
		PluginID:  pluginID,
		Kind:      kind,
		Status:    "running",
		CreatedAt: now,
		UpdatedAt: now,
	}
	remotePluginAssetJobs.Lock()
	defer remotePluginAssetJobs.Unlock()
	userJobCount := 0
	for id, existing := range remotePluginAssetJobs.jobs {
		if now.Sub(existing.UpdatedAt) > 2*time.Hour {
			delete(remotePluginAssetJobs.jobs, id)
			continue
		}
		if existing.UserID == userID && existing.Status == "running" {
			userJobCount++
		}
	}
	if userJobCount >= maxRemotePluginAssetJobsPerUser {
		return nil, fmt.Errorf("too many active asset materialization jobs")
	}
	remotePluginAssetJobs.jobs[job.ID] = job
	return job, nil
}

func updateRemotePluginAssetsJob(id string, update func(*RemotePluginAssetsJob)) {
	remotePluginAssetJobs.Lock()
	defer remotePluginAssetJobs.Unlock()
	job := remotePluginAssetJobs.jobs[id]
	if job == nil {
		return
	}
	update(job)
	job.UpdatedAt = time.Now().UTC()
}

func remotePluginAssetsJobSnapshot(id string) *RemotePluginAssetsJob {
	remotePluginAssetJobs.Lock()
	defer remotePluginAssetJobs.Unlock()
	job := remotePluginAssetJobs.jobs[id]
	if job == nil {
		return nil
	}
	copy := *job
	return &copy
}
