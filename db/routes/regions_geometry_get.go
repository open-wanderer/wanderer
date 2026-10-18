package routes

import (
	"net/http"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"pocketbase/services/regions"
)

// RegionGeometryGet returns a single leaf region's boundary geometry (GeoJSON)
// and bbox, fetched on demand and converted server-side from CoMaps'
// Osmosis-format `.poly` files. The conversion cannot move to
// the client: `ParsePoly` is a hand-rolled multi-ring hole/exclave parser,
// so a browser-side
// reimplementation would mean duplicating that parser in JavaScript and
// widening the admin page's CSP to the two upstream hosts (github/codeberg)
// it fetches from.
//
// This route is bound to apis.RequireSuperuserAuth() rather than the weaker
// apis.RequireAuth() used by the sibling /regions group: it triggers
// outbound third-party requests on every call, so an authenticated-user-only
// gate would leave it as both an open proxy and a way to burn CoMaps' rate
// limit from outside.
//
// Persistence is decided entirely inside regions.ResolveGeometry, derived
// from the looked-up region's own `enabled` field — this handler
// never reads `enabled` itself and accepts no query parameter that could
// influence whether a write happens. Hovering a disabled region is therefore
// a pass-through with no write; toggling a region on and then
// re-drawing it persists as a side effect of that same call once the record
// is enabled.
func RegionGeometryGet(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	if !regions.IsValidRegionID(id) {
		return e.BadRequestError("invalid region id", nil)
	}

	region, err := e.App.FindFirstRecordByFilter("regions", "path = {:path}", dbx.Params{"path": id})
	if err != nil || region == nil {
		return e.NotFoundError("region not found", nil)
	}

	if region.GetString("kind") != "leaf" {
		return e.BadRequestError("region is not a leaf", nil)
	}

	polygon, bbox, err := regions.ResolveGeometry(e.App, region)
	if err != nil {
		return e.InternalServerError("could not resolve region geometry", err)
	}

	return e.JSON(http.StatusOK, map[string]any{
		"path":    id,
		"polygon": polygon,
		"bbox":    bbox,
	})
}
