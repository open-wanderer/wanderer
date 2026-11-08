package hooks

import (
	"github.com/pocketbase/pocketbase/core"
	"net/http"
	"strings"
)

// PrepareAssetFileResponse covers native file failures that PocketBase returns
// before its download hook, including missing records and thumbnail originals.
func PrepareAssetFileResponse(e *core.RequestEvent) error {
	if (e.Request.Method == http.MethodGet || e.Request.Method == http.MethodHead) && strings.HasPrefix(e.Request.URL.Path, "/api/files/") {
		collection := e.Request.PathValue("collection")
		schema, err := e.App.FindCachedCollectionByNameOrId(collection)
		if collection == "assets" || err == nil && schema.Name == "assets" {
			e.Response.Header().Set("Cache-Control", "no-store")
			e.Response.Header().Add("Vary", "Authorization")
			e.Response = &assetFileCacheResponse{ResponseWriter: e.Response}
		}
	}
	return e.Next()
}

// ServeContent can write a failed conditional or range response without
// returning an error. Set its cache policy before headers reach the client.
type assetFileCacheResponse struct{ http.ResponseWriter }

func (response *assetFileCacheResponse) WriteHeader(status int) {
	if status != http.StatusOK && status != http.StatusPartialContent && status != http.StatusNotModified {
		response.Header().Set("Cache-Control", "no-store")
	}
	response.ResponseWriter.WriteHeader(status)
}

func (response *assetFileCacheResponse) Unwrap() http.ResponseWriter {
	return response.ResponseWriter
}

// AuthorizeAssetFileDownload applies the asset's current access rules to both
// original files and thumbnails, which PocketBase otherwise serves publicly.
func AuthorizeAssetFileDownload() func(e *core.FileDownloadRequestEvent) error {
	return func(e *core.FileDownloadRequestEvent) error {
		// Failed reads never enter the viewer's browser cache.
		e.Response.Header().Set("Cache-Control", "no-store")
		if !strings.Contains(strings.Join(e.Response.Header().Values("Vary"), ","), "Authorization") {
			e.Response.Header().Add("Vary", "Authorization")
		}

		requestInfo, err := e.RequestInfo()
		if err != nil {
			return e.InternalServerError("Failed to load request info", err)
		}
		if allowed, err := e.App.CanAccessRecord(e.Record, requestInfo, e.Record.Collection().ViewRule); !allowed {
			return e.NotFoundError("", err)
		}

		e.Response.Header().Set("Cache-Control", "private, max-age=3600")
		if err := e.Next(); err != nil {
			e.Response.Header().Set("Cache-Control", "no-store")
			return err
		}
		return nil
	}
}
