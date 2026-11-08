package routes

import (
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// AssetOwnedCopy exposes only the caller's reusable copy, after checking
// that the caller can still read its source. Private provenance is never a
// public filter parameter.
func AssetOwnedCopy(e *core.RequestEvent) error {
	if e.Auth == nil {
		return e.UnauthorizedError("authentication required", nil)
	}
	source, err := e.App.FindRecordById("assets", e.Request.PathValue("id"))
	if err != nil || source.GetString("type") != "photo" {
		return e.NotFoundError("Photo source not found", err)
	}
	info, err := e.RequestInfo()
	if err != nil {
		return err
	}
	allowed, err := e.App.CanAccessRecord(source, info, source.Collection().ViewRule)
	if err != nil || !allowed {
		return e.NotFoundError("Photo source not found", err)
	}
	actor, err := e.App.FindFirstRecordByFilter("activitypub_actors", "user={:user} && is_local=true", dbx.Params{"user": e.Auth.Id})
	if err != nil {
		return e.NotFoundError("Photo copy not found", err)
	}
	copy, err := e.App.FindFirstRecordByFilter("assets", "author={:author} && type='photo' && storage_mode='copy' && file!='' && source_metadata.copied_from={:source}", dbx.Params{"author": actor.Id, "source": source.Id})
	if err != nil {
		return e.NotFoundError("Photo copy not found", err)
	}
	if err := apis.EnrichRecord(e, copy); err != nil {
		return err
	}
	e.Response.Header().Set("Cache-Control", "private, no-store")
	return e.JSON(http.StatusOK, copy)
}
