package routes

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"pocketbase/util"
)

var cardTrailID = regexp.MustCompile(`^[a-z0-9]{15}$`)

type trailCardPreview struct {
	Photos    []string `json:"photos"`
	Thumbnail int      `json:"thumbnail"`
}

func AssetTrailPreviews(e *core.RequestEvent) error {
	info, err := e.RequestInfo()
	if err != nil {
		return err
	}
	ids := []string{}
	seen := map[string]bool{}
	rawIDs := e.Request.URL.Query().Get("ids")
	if len(rawIDs) > 1600 {
		return e.BadRequestError("At most 100 trails can be previewed", nil)
	}
	for _, id := range strings.Split(rawIDs, ",") {
		if id == "" {
			continue
		}
		if !cardTrailID.MatchString(id) {
			return e.BadRequestError("Invalid trail id", nil)
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
		if len(ids) > 100 {
			return e.BadRequestError("At most 100 trails can be previewed", nil)
		}
	}
	previews := map[string]trailCardPreview{}
	readable := []string{}
	for _, id := range ids {
		trail, err := e.App.FindRecordById("trails", id)
		if err != nil {
			continue
		}
		if allowed, err := e.App.CanAccessRecord(trail, info, trail.Collection().ViewRule); err != nil || !allowed {
			continue
		}
		readable = append(readable, id)
		previews[id] = trailCardPreview{Photos: []string{}}
	}
	links, err := util.CardPhotoLinks(e.App, info, "trail", readable)
	if err != nil {
		return e.InternalServerError("Failed to load trail previews", err)
	}
	share := e.Request.URL.Query().Get("share")
	for id, rows := range links {
		preview := previews[id]
		for _, link := range rows {
			path := fmt.Sprintf("/api/v1/assets/%s/file", link.GetString("asset"))
			if asset := link.ExpandedOne("asset"); asset != nil && asset.GetString("file") != "" {
				path = fmt.Sprintf("/api/v1/files/%s/%s/%s", asset.Collection().Id, asset.Id, asset.GetString("file"))
			}
			if share != "" {
				path += "?share=" + url.QueryEscape(share)
			}
			preview.Photos = append(preview.Photos, path)
		}
		preview.Thumbnail = util.CardPhotoThumbnail(rows)
		previews[id] = preview
	}
	e.Response.Header().Set("Cache-Control", "private, no-cache, must-revalidate")
	e.Response.Header().Add("Vary", "Authorization")
	return e.JSON(http.StatusOK, map[string]any{"previews": previews})
}
