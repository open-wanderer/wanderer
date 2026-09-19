package hooks

import "github.com/pocketbase/pocketbase/core"

// AuthorizeAssetFileDownload applies the asset's current access rules to both
// original files and thumbnails, which PocketBase otherwise serves publicly.
func AuthorizeAssetFileDownload() func(e *core.FileDownloadRequestEvent) error {
	return func(e *core.FileDownloadRequestEvent) error {
		// Revalidate every use so revoking a share also revokes cached file URLs.
		e.Response.Header().Set("Cache-Control", "private, no-cache, must-revalidate")
		e.Response.Header().Add("Vary", "Authorization")

		requestInfo, err := e.RequestInfo()
		if err != nil {
			return e.InternalServerError("Failed to load request info", err)
		}
		if allowed, err := e.App.CanAccessRecord(e.Record, requestInfo, e.Record.Collection().ViewRule); !allowed {
			return e.NotFoundError("", err)
		}

		return e.Next()
	}
}
