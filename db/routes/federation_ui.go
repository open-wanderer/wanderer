package routes

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

//go:embed federation_ext/federation_ui.html
var federationUIHTML []byte

// FederationExtFS is the embedded filesystem for the PocketBase UI extension.
// Registers a header link in the admin panel pointing to /federation/.
//
//go:embed federation_ext
var federationExtEmbedded embed.FS

// FederationExtFS returns the sub-filesystem rooted at federation_ext/.
func FederationExtFS() fs.FS {
	sub, err := fs.Sub(federationExtEmbedded, "federation_ext")
	if err != nil {
		// fall back to disk for development hot-reload
		return os.DirFS("routes/federation_ext")
	}
	return sub
}

// FederationDashboard serves the embedded federation admin SPA.
//
// The page contains no secrets or tokens. All privileged operations are
// performed via fetch() calls to API handlers that enforce superuser auth
// server-side. The superuser JWT is read at runtime from the admin's own
// localStorage — it is never embedded in the document.
//
// X-Frame-Options: DENY prevents clickjacking of the admin page.
func FederationDashboard(e *core.RequestEvent) error {
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	e.Response.Header().Set("X-Frame-Options", "DENY")
	e.Response.Header().Set("X-Content-Type-Options", "nosniff")
	// Every directive is same-origin only. Scripts and styles come from files
	// served by the asset route, the page holds no inline code or style, and the
	// Alpine build it runs needs no runtime code generation. The admin's browser
	// therefore contacts no third-party host while this page holds their token.
	e.Response.Header().Set("Content-Security-Policy",
		"default-src 'none'; "+
			"script-src 'self'; "+
			"style-src 'self'; "+
			"font-src 'self'; "+
			"connect-src 'self'; "+
			"img-src 'self'; "+
			"frame-ancestors 'none'; "+
			"base-uri 'none'; "+
			"form-action 'none'")
	e.Response.WriteHeader(http.StatusOK)
	_, err := e.Response.Write(federationUIHTML)
	return err
}

// federationAssetExtensions lists the file types FederationAssets will serve.
// The HTML page and the markdown notes live in the same embedded tree but are
// not assets: the page is only safe behind the headers FederationDashboard sets.
var federationAssetExtensions = map[string]bool{
	".css":   true,
	".js":    true,
	".woff2": true,
	".txt":   true,
}

// FederationAssets serves the first-party files of the federation page (CSS,
// scripts, fonts, vendored libraries) from the embedded federation_ext tree.
//
// It is its own route so a changed file is revalidated on every load instead of
// being kept for 14 days, as on PocketBase's extension path. Only the file types
// in federationAssetExtensions are served, so federation_ui.html is reachable
// solely through FederationDashboard with its CSP and X-Frame-Options headers.
// apis.Static rejects path traversal and can only reach files embedded in the
// binary.
func FederationAssets() func(*core.RequestEvent) error {
	static := apis.Static(FederationExtFS(), false)
	return func(e *core.RequestEvent) error {
		ext := strings.ToLower(path.Ext(e.Request.PathValue("path")))
		if !federationAssetExtensions[ext] {
			return e.NotFoundError("", nil)
		}
		e.Response.Header().Set("Cache-Control", "no-cache")
		e.Response.Header().Set("X-Content-Type-Options", "nosniff")
		return static(e)
	}
}
