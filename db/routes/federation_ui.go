package routes

import (
	"embed"
	"io/fs"
	"net/http"
	"os"

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
// The page is intentionally public (D-01): the HTML contains no secrets or
// tokens.  All privileged operations are performed via fetch() calls to Phase 5
// API handlers that enforce superuser auth server-side.  The superuser JWT is
// read at runtime from the admin's own localStorage — it is never embedded in
// the document.
//
// T-06-05: X-Frame-Options: DENY prevents clickjacking of the admin page.
func FederationDashboard(e *core.RequestEvent) error {
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	e.Response.Header().Set("X-Frame-Options", "DENY")
	e.Response.Header().Set("X-Content-Type-Options", "nosniff")
	// CSP locks down script sources to the two CDNs the page loads (Alpine + Tailwind).
	// style-src 'unsafe-inline' is required by Tailwind Play CDN which injects inline styles.
	e.Response.Header().Set("Content-Security-Policy",
		"default-src 'none'; "+
			"script-src 'unsafe-inline' 'unsafe-eval' https://cdn.tailwindcss.com https://cdn.jsdelivr.net; "+
			"style-src 'unsafe-inline' https://cdnjs.cloudflare.com; "+
			"font-src https://cdnjs.cloudflare.com; "+
			"connect-src 'self'; "+
			"img-src 'self'; "+
			"frame-ancestors 'none'")
	e.Response.WriteHeader(http.StatusOK)
	_, err := e.Response.Write(federationUIHTML)
	return err
}
