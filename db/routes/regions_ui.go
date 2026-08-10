package routes

import (
	"embed"
	"io/fs"
	"net/http"
	"os"

	"github.com/pocketbase/pocketbase/core"
)

//go:embed regions_ext/regions_ui.html
var regionsUIHTML []byte

// RegionsExtFS is the embedded filesystem for the PocketBase UI extension.
// Registers a header link in the admin panel pointing to /region-catalog/.
//
//go:embed regions_ext
var regionsExtEmbedded embed.FS

// RegionsExtFS returns the sub-filesystem rooted at regions_ext/.
func RegionsExtFS() fs.FS {
	sub, err := fs.Sub(regionsExtEmbedded, "regions_ext")
	if err != nil {
		// fall back to disk for development hot-reload
		return os.DirFS("routes/regions_ext")
	}
	return sub
}

// RegionsDashboard serves the embedded region-catalog admin SPA.
//
// The page contains no secrets or tokens. Most privileged operations go
// directly through PocketBase's own JWT-validated collection REST API
// (/api/collections/regions/records, /api/collections/region_geometry/records).
// The page's hover-preview flow additionally calls the superuser-gated Go
// route /regions/{id}/geometry directly, so there are three privileged
// surfaces in total, not two. The superuser JWT is read at runtime from the
// admin's own localStorage — it is never embedded in the document.
//
// The frame-deny header below prevents clickjacking of the admin page.
func RegionsDashboard(e *core.RequestEvent) error {
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	e.Response.Header().Set("X-Frame-Options", "DENY")
	e.Response.Header().Set("X-Content-Type-Options", "nosniff")
	// Widened vs. federation_ui.go's CSP — MapLibre needs unpkg.com (its
	// own script/style host), a blob: worker allowance (its Web Worker pool), and
	// tiles.openfreemap.org (connect-src for style/tile/glyph/sprite
	// fetches, img-src for raster assets). Copying federation's CSP verbatim
	// silently breaks the map (blank pane, no visible error unless devtools
	// are open).
	e.Response.Header().Set("Content-Security-Policy",
		"default-src 'none'; "+
			"script-src 'unsafe-inline' 'unsafe-eval' https://cdn.jsdelivr.net https://unpkg.com; "+
			"style-src 'unsafe-inline' https://fonts.googleapis.com https://cdn.jsdelivr.net https://unpkg.com; "+
			"font-src https://fonts.gstatic.com https://cdn.jsdelivr.net; "+
			"worker-src blob:; "+
			"connect-src 'self' https://tiles.openfreemap.org; "+
			"img-src 'self' data: blob: https://tiles.openfreemap.org; "+
			"frame-ancestors 'none'")
	e.Response.WriteHeader(http.StatusOK)
	_, err := e.Response.Write(regionsUIHTML)
	return err
}
