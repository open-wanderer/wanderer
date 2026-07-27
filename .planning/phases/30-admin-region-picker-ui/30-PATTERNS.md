# Phase 30: Admin Region Picker UI - Pattern Map

**Mapped:** 2026-07-27
**Files analyzed:** 4 new + 1 modified
**Analogs found:** 5 / 5

**Important note on analog location:** The primary analog (`federation_ui.go` / `federation_ext/federation_ui.html` / `federation_ext/main.js`) lives on the **unmerged** branch `feature/ap-instance-actors`, not in this branch's working tree. Every excerpt below was extracted via `git show feature/ap-instance-actors:<path>`. Do not `Read` these paths directly on `feature/app` — they 404. The planner/executor must re-run `git show` themselves to pull full file contents when writing plans/code.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `db/routes/regions_ui.go` | route (Go embed handler) | request-response | `db/routes/federation_ui.go` (branch `feature/ap-instance-actors`) | exact |
| `db/routes/regions_ext/regions_ui.html` | component (standalone Alpine SPA) | CRUD (client-side REST against PocketBase collections) + streaming-ish map render | `db/routes/federation_ext/federation_ui.html` (branch `feature/ap-instance-actors`) | exact (shell/auth/CSS), no analog for tree+map (net new) |
| `db/routes/regions_ext/main.js` | config (UI extension header-link injection) | event-driven (push into PB dashboard's Alpine store) | `db/routes/federation_ext/main.js` (branch `feature/ap-instance-actors`) | exact |
| `db/main.go` (modified) | route registration | request-response | Existing `se.Router.GET("/federation/...")` block + `regionsGroup` block, both already in this file / branch | exact |
| (reference, read-only) `db/migrations/1785000000_create_regions_collection.go` | model/schema | CRUD | n/a — this is the data source being read, not created | — |

## Pattern Assignments

### `db/routes/regions_ui.go` (route, request-response)

**Analog:** `db/routes/federation_ui.go` (`git show feature/ap-instance-actors:db/routes/federation_ui.go`)

**Full pattern to copy (adapt embed path + widen CSP):**
```go
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

//go:embed regions_ext
var regionsExtEmbedded embed.FS

func RegionsExtFS() fs.FS {
	sub, err := fs.Sub(regionsExtEmbedded, "regions_ext")
	if err != nil {
		return os.DirFS("routes/regions_ext")
	}
	return sub
}

func RegionsDashboard(e *core.RequestEvent) error {
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	e.Response.Header().Set("X-Frame-Options", "DENY")
	e.Response.Header().Set("X-Content-Type-Options", "nosniff")
	// WIDENED vs. federation_ui.go's CSP — MapLibre needs unpkg.com (script/style),
	// worker-src blob:, and tiles.openfreemap.org (connect-src + img-src).
	// See 30-RESEARCH.md Pitfall 3 / Pattern 1 for the exact widened header.
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
```

**Do not copy verbatim:** the CSP block — federation's `default-src 'none'` + jsdelivr-only allowlist silently breaks MapLibre (blank map, no console-visible error unless devtools open). Use the widened version above.

---

### `db/routes/regions_ext/regions_ui.html` (component, CRUD + client-side render)

**Analog:** `db/routes/federation_ext/federation_ui.html` (`git show feature/ap-instance-actors:db/routes/federation_ext/federation_ui.html`, 1094 lines)

**Head/theme IIFE pattern** (analog lines 1-28) — copy verbatim, no changes needed:
```html
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Region Catalog · Wanderer</title>
  <link rel="icon" href="/_/images/favicon.png">
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=IBM+Plex+Sans:wght@400;500;600&display=swap">
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/remixicon@4.5.0/fonts/remixicon.css">
  <!-- ADD (net new for this phase): MapLibre CSS -->
  <link rel="stylesheet" href="https://unpkg.com/maplibre-gl@5.24.0/dist/maplibre-gl.css">

  <!-- Sync: theme + accent color before first paint to avoid flash -->
  <script>
    (function () {
      var s = localStorage.getItem('pbColorScheme');
      if (s === 'dark' || (!s && window.matchMedia('(prefers-color-scheme: dark)').matches)) {
        document.documentElement.classList.add('dark');
      }
      try {
        var pb = JSON.parse(localStorage.getItem('pbSettings') || '{}');
        var accent = (pb.meta && pb.meta.accentColor) || '#1055c9';
        document.documentElement.style.setProperty('--accent', accent);
        var r = parseInt(accent.slice(1, 3), 16), g = parseInt(accent.slice(3, 5), 16), b = parseInt(accent.slice(5, 7), 16);
        document.documentElement.style.setProperty('--accent-fade', 'rgba(' + r + ',' + g + ',' + b + ',0.1)');
      } catch (_) { }
    })();
  </script>
```

**CSS custom-property root block** (analog lines 30-51) — copy verbatim, this IS the design system per UI-SPEC Color section (no Tailwind, contra UI-SPEC's stale claim — see RESEARCH.md Pitfall 5):
```css
:root {
  --accent: #1055c9;
  --accent-fade: rgba(16, 85, 201, 0.1);
  --bg: #ffffff;
  --surface: #f8f9fa;
  --surface2: #e4e8ec;
  --border: #e2e2e4;
  --text: #16181a;
  --text2: #9a9a9a;
  --danger: #e1473d;
}
html.dark {
  --bg: #1f1f1f;
  --surface: #232323;
  --surface2: #2f2f2f;
  --border: #353849;
  --text: #e4e5ea;
  --text2: #6e7491;
}
[x-cloak] { display: none !important; }
```
Extend this block with new component classes for phase 30: `.tree-row`, `.tree-row:hover`, `.toggle-switch` (track/thumb per UI-SPEC's `36px×20px`/`16px` spec), `.map-pane`, `.row-error`, `.count-badge`. Reuse `.input`/`.btn`/`.btn-outline`/`.btn-primary`/`.btn-danger`/`.btn-sm`/`.spinner-accent` classes verbatim — do not redefine them.

**Auth shell + apiFetch pattern** (analog lines 627-673) — copy verbatim, rename `federationApp` → `regionsApp`:
```js
function regionsApp() {
  return {
    token: null,
    authenticated: false,
    email: '',
    // ...phase-30 state added below (regions, visibleRows, expanded, map, etc.)

    init() {
      var stored = localStorage.getItem('__pb_superusers__/_');
      if (!stored) { return; }
      try {
        var parsed = JSON.parse(stored);
        this.token = parsed.token;
        try {
          var payload = JSON.parse(atob(this.token.split('.')[1]));
          this.email = payload.email || '';
        } catch (_) { }
      } catch (_) { return; }
      if (!this.token) { return; }
      if (this.isTokenExpired(this.token)) { return; }
      this.authenticated = true;
      this.loadRegions(); // was loadPeers() in federation
    },

    isTokenExpired(token) {
      try {
        var payload = JSON.parse(atob(token.split('.')[1]));
        return payload.exp && Date.now() / 1000 > payload.exp;
      } catch (_) { return true; }
    },

    async apiFetch(path, opts) {
      opts = opts || {};
      var res = await fetch(path, Object.assign({}, opts, {
        headers: Object.assign({ 'Authorization': this.token }, opts.headers || {})
      }));
      if (res.status === 401) { this.authenticated = false; return null; }
      return res;
    },
  };
}
```
Note: `apiFetch` here calls PocketBase's raw collection REST API directly (`/api/collections/regions/records`, `/api/collections/region_polygons/records`) rather than the custom `/federation/*` routes federation_ui.html targets — per D-01, no new Go handlers are needed for this phase's CRUD.

**Optimistic toggle-with-revert pattern** (analog lines 747-773, `approve`/`reject`) — adapt to a single leaf-toggle PATCH, per RESEARCH.md's already-adapted example:
```js
async toggleRegion(row) {
  const prevEnabled = row.enabled;
  row.enabled = !prevEnabled;
  if (row.enabled) this.addPolygonForRow(row); else this.removePolygonForRow(row);
  this.rowErrors = Object.assign({}, this.rowErrors, { [row.id]: undefined });

  const res = await this.apiFetch('/api/collections/regions/records/' + row.id, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ enabled: row.enabled }),
  });
  if (!res) return;
  if (!res.ok) {
    row.enabled = prevEnabled;
    if (prevEnabled) this.addPolygonForRow(row); else this.removePolygonForRow(row);
    const data = await res.json().catch(function () { return {}; });
    this.rowErrors = Object.assign({}, this.rowErrors, {
      [row.id]: "Couldn't update" + (data.message ? ': ' + data.message : '') + ' — try again.'
    });
  }
}
```
This mirrors federation's `approve`/`reject` shape (`this.loading = true` / try / `if (!res) return` / `if (res.ok) {...} else { rowErrors[...] = ... }` / `finally`) exactly, just swapped from a POST-action-then-reload to a PATCH-with-local-optimistic-state, since D-06 requires instant UI feedback rather than a full reload.

**App-header markup** (analog line ~807 onward) — reuse verbatim: logo SVG, "Back to dashboard" link to `/_/`, header height 38px sticky. Extract full block via `git show feature/ap-instance-actors:db/routes/federation_ext/federation_ui.html` lines 807-880 when implementing (not excerpted here — purely static markup, no logic changes).

**No analog exists for (net new, build per RESEARCH.md/UI-SPEC directly):**
- Tree-building/flattening (`buildTree`, `flattenVisible`, `computeDefaultExpanded`, `computeFilterMatches`) — see 30-RESEARCH.md Pattern 3, D-07, D-08 code examples; use those verbatim, they're already fully specified.
- Two-page `loadAllRegions()` fetch loop (RESEARCH.md Pattern 4) and chunked `loadEnabledPolygons()` (RESEARCH.md Pattern 5) — no prior CRUD-loop analog in this codebase talks to raw PocketBase collection REST directly from an admin HTML page; federation's `loadPeers()` hits a custom Go route instead, so only the try/finally/loading-flag shape carries over, not the fetch mechanics.
- MapLibre `addRegionLayer`/`removeRegionLayer`/`fitToEnabled` (RESEARCH.md Pattern 6) — genuinely new capability, no map exists anywhere in `db/routes/` today. Cross-reference `web/src/lib/vendor/maplibre-layer-manager/layers.ts:46-100+` only for which OpenFreeMap style/sprite/glyph URLs are conventionally used in this codebase (informational, not copy-paste — that file is SvelteKit/TS, this page is vanilla JS).

---

### `db/routes/regions_ext/main.js` (config, event-driven)

**Analog:** `db/routes/federation_ext/main.js` (`git show feature/ap-instance-actors:db/routes/federation_ext/main.js`, 5 lines, full file)

**Full pattern — copy and adapt label/icon/href:**
```js
app.store.headerLinks.push({
  href: "/region-catalog/",
  icon: "ri-map-2-line",
  label: "Region Catalog",
});
```
(Original: `href: "/federation/", icon: "ri-fediverse-fill", label: "Federation"`.)

---

### `db/main.go` (modified — route registration)

**Analog A (route + UIExtension registration shape):** `git show feature/ap-instance-actors:db/main.go` lines 211-222:
```go
se.Router.POST("/federation/discover", routes.FederationDiscover)
se.Router.POST("/federation/follow", routes.FederationFollow)
se.Router.POST("/federation/approve/{id}", routes.FederationApprove)
se.Router.POST("/federation/reject/{id}", routes.FederationReject)
se.Router.POST("/federation/disconnect/{id}", routes.FederationDisconnect)
se.Router.GET("/federation/peers", routes.FederationPeers)
se.Router.GET("/federation/", routes.FederationDashboard)

se.UIExtensions = append(se.UIExtensions, core.UIExtension{
  Name: "wanderer-federation",
  FS:   routes.FederationExtFS(),
})
```

**Adapted for this phase (no custom action routes needed — D-01, only the dashboard route + UIExtension):**
```go
se.Router.GET("/region-catalog/", routes.RegionsDashboard)

se.UIExtensions = append(se.UIExtensions, core.UIExtension{
  Name: "wanderer-region-catalog",
  FS:   routes.RegionsExtFS(),
})
```

**Analog B (existing route-naming collision to avoid):** this branch's `db/main.go` lines 216-229 already registers an unrelated `/regions` group:
```go
// db/main.go:224-229 (current branch, feature/app — already present, do not modify)
regionsGroup := se.Router.Group("/regions")
regionsGroup.Bind(apis.RequireAuth())

regionsGroup.GET("", routes.RegionsList)
regionsGroup.GET("/{id}/download", routes.RegionArchiveDownload)
regionsGroup.GET("/{id}/download-dem", routes.RegionArchiveDownloadDem)
```
Confirms RESEARCH.md Pitfall 6: the new admin route MUST use a distinct top-level path (e.g. `/region-catalog/`), not anything nested under or prefixed with `/regions`, to avoid colliding with this pre-existing any-authenticated-user group.

**Insertion point:** add the new `se.Router.GET("/region-catalog/", ...)` + `UIExtensions` append as a new block near line 214 (after the `/remote/profile/{handle}/follows` route, before the `regionsGroup` block), inside the same function where `se.Router.GET(...)` calls already live (verify exact function name/signature by reading `db/main.go` in full — the excerpt above only shows the relevant lines, not the enclosing function).

---

## Shared Patterns

### Superuser auth + JWT expiry check
**Source:** `federation_ui.html` `init()` + `isTokenExpired()` (analog lines 642-664, extracted via `git show feature/ap-instance-actors:db/routes/federation_ext/federation_ui.html`)
**Apply to:** `regions_ui.html`'s `regionsApp()` — copy verbatim, only the post-auth `init()` tail call changes (`this.loadRegions()` instead of `this.loadPeers()`).

### `apiFetch()` wrapper (401 → logged-out state)
**Source:** same file, lines 666-673
**Apply to:** every PocketBase REST call in `regions_ui.html` (`loadAllRegions`, `loadEnabledPolygons`, `toggleRegion`'s PATCH) — copy verbatim, unchanged (it already forwards the raw JWT with no `Bearer` prefix, matching PocketBase's expected header format).

### CSS custom-property theming (no Tailwind)
**Source:** same file, lines 30-51 (`:root` / `html.dark` blocks)
**Apply to:** all new visual elements (tree row, toggle switch, map pane, filter input, count badge, row error) — extend this property set, do not introduce a Tailwind CDN tag despite the UI-SPEC's stale claim (RESEARCH.md Pitfall 5 confirms zero `tailwind` references in the actual canonical file).

### Optimistic action + inline per-row error (loading/try/finally + rowErrors map)
**Source:** same file, `approve()`/`reject()`/`disconnect()` (lines 747-797)
**Apply to:** `toggleRegion()` — same `rowErrors` object-spread-update pattern, same "no dedicated retry button, the reverted control is already actionable" UX contract (D-06 explicitly matches this existing convention).

### CSP header (widened, not verbatim)
**Source:** `federation_ui.go` (`git show feature/ap-instance-actors:db/routes/federation_ui.go`, full 51-line file)
**Apply to:** `regions_ui.go`'s `RegionsDashboard` handler — same X-Frame-Options/X-Content-Type-Options/CSP-header shape, but the CSP directive list itself must be widened per the excerpt above (add `unpkg.com`, `worker-src blob:`, `tiles.openfreemap.org`) — do not copy the CSP string verbatim, only the surrounding handler structure.

## No Analog Found

| File/Concern | Role | Data Flow | Reason |
|---|---|---|---|
| Tree flatten/expand/filter logic (`buildTree`, `flattenVisible`, `computeDefaultExpanded`, `computeFilterMatches`) | component logic | transform | No tree UI exists anywhere in this codebase (web app or admin pages); fully specified instead in 30-RESEARCH.md Pattern 3 + D-07/D-08 code examples — use those directly as the "analog." |
| MapLibre polygon layer add/remove + bbox-union `fitBounds` | component logic | streaming/transform | No MapLibre usage exists in any `db/routes/` admin page; fully specified in 30-RESEARCH.md Pattern 6 — use directly. |
| Chunked OR-filter pagination against `region_polygons` | service/data-fetch | batch | No prior raw-PocketBase-REST consumer in this codebase does OR-filter chunking; fully specified in 30-RESEARCH.md Pattern 5 (60-path chunks, tune per A2 assumption) — use directly. |

## Metadata

**Analog search scope:** `db/routes/` (current branch + `feature/ap-instance-actors` via `git show`), `db/main.go`, `db/migrations/1785000000_create_regions_collection.go`, `web/src/lib/vendor/maplibre-layer-manager/layers.ts` (informational only)
**Files scanned:** `federation_ui.go` (51 lines, full), `federation_ext/federation_ui.html` (1094 lines, targeted reads: 1-60, 627-800; remaining sections — app-header markup ~807-880, tree/table markup ~880-1090 — not yet excerpted, planner should pull via `git show` when writing the plan's action section), `federation_ext/main.js` (5 lines, full), `db/main.go` (current branch, ~230 lines relevant section read), `db/migrations/1785000000_create_regions_collection.go` (235 lines, full)
**Pattern extraction date:** 2026-07-27
