# Phase 30: Admin Region Picker UI - Research

**Researched:** 2026-07-27
**Domain:** PocketBase custom admin extension (Go embed + AlpineJS SPA) + MapLibre GL JS polygon rendering
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Data access pattern**
- **D-01:** No new custom Go endpoints for listing/toggling regions. The admin page authenticates as a PocketBase superuser (same pattern as `federation_ui.html` on `feature/ap-instance-actors`: read `__pb_superusers__/_` from localStorage, redirect to `/_/` if missing/expired) and talks directly to PocketBase's built-in collection REST API (`/api/collections/regions/records`, `/api/collections/region_polygons/records`). Neither collection currently has custom list/view/update rules, so they default to superuser-only — matching this access pattern with zero rule changes required.
- **D-02:** Tree hierarchy is loaded with **one full fetch** — a single `regions` list call with `perPage` overridden past PocketBase's default 30/page cap (e.g. 1500) — and the parent/child tree is assembled client-side from `parent`/`path`/`depth`. Rejected lazy-per-expand loading: 1,306 flat hierarchy rows (no polygon/bbox bulk) is a small enough payload that round-tripping on every expand click isn't worth the added latency.
- **D-03:** `region_polygons` (a separate, deliberately-split-out table at ~165KB/leaf — see `db/migrations/1785000000_create_regions_collection.go`) is fetched **only for currently-enabled leaves**, both on initial load and again for the single region whenever a toggle flips. Rejected: fetching all 1,153 polygons upfront (payload would be hundreds of MB) and lazy-fetch-on-group-expand (would hide enabled regions sitting in collapsed branches).

**Live map**
- **D-04:** Map style is MapLibre GL JS (CDN, no build step available on this standalone page) pointed directly at OpenFreeMap's public hosted style URL (`https://tiles.openfreemap.org/styles/liberty`) — same tile family as the web app's default (`web/static/styles/ofm.json`'s `openmaptiles` vector source), no API key, no file to keep in sync. Rejected copying `ofm.json` into `db/routes/` (duplicated-file drift risk) and plain raster OSM tiles (against OSM's tile usage policy).
- **D-05:** On page load, the map auto-fits its bounds to the union of all currently-enabled leaf polygons, so the admin sees existing coverage at a glance without manual panning.
- **D-06:** Toggling a region is **optimistic** end-to-end: the checkbox/switch flips and the map polygon appears/disappears immediately on click, before the PATCH resolves. On PATCH failure: revert the toggle to its prior state, remove/re-add the map polygon to match, and show an inline error next to that specific tree row (not a global toast). No dedicated retry button — the reverted toggle is already clickable again, so re-clicking re-fires the same PATCH.

**Tree UX**
- **D-07:** Default expand state is neither fully collapsed nor fully expanded: only branches that contain at least one already-`enabled` leaf are auto-expanded on load; everything else starts collapsed.
- **D-08:** A simple client-side name-filter box is in scope for this phase (not deferred) — it filters over the already-fetched 1,306 rows with no extra API call. Typing in the filter narrows the tree to matches plus their ancestor chain, auto-expanded.

### Claude's Discretion
- Exact visual styling of the inline per-row error message (color, icon, dismiss behavior) — Tailwind CDN, consistent with `federation_ui.html`'s existing look. **Research correction:** the canonical reference does NOT actually load Tailwind CDN (see Common Pitfalls) — style with the existing hand-rolled CSS custom-property classes instead.
- Whether the filter box does substring or fuzzy matching — substring is sufficient.
- Debounce timing on the filter input, if any — UI-SPEC settled this at 120ms.

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope. The search/filter box was considered for deferral but was decided in-scope (D-08).

Reviewed-but-not-folded todos: `2026-07-24-comaps-poly-region-extraction-spike.md` (stale match, belongs to Phase 29 which is complete) and `2026-07-18-way-types-and-surfaces-breakdown.md` (unrelated mobile feature).
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| ADMINUI-01 | A custom PocketBase admin page (AlpineJS bundle, `feature/ap-instance-actors` pattern) renders the region catalog as a collapsible tree | Go embed/route pattern verified against `federation_ui.go`/`federation_ui.html` (unmerged branch, must be extracted via `git show`); flat-list-with-visibility Alpine pattern recommended over recursive templates; `regions` collection fetch verified against real schema |
| ADMINUI-02 | The admin toggles a leaf region's `enabled` flag directly from the tree; takes effect on the cron's next run | `PATCH /api/collections/regions/records/{id}` verified against installed PocketBase v0.38.0 source; optimistic-UI pattern from `federation_ui.html`'s `approve`/`reject`/`disconnect` handlers directly reusable |
| ADMINUI-03 | A live map renders boundary polygons of all currently-enabled leaf regions | `region_polygons.polygon` confirmed GeoJSON `Polygon`/`MultiPolygon`; `regions.bbox` confirmed `[minLon,minLat,maxLon,maxLat]` (directly usable in `map.fitBounds`); MapLibre CSP requirements researched and are the #1 non-obvious risk for this requirement |
</phase_requirements>

## Summary

This phase adds a fourth self-contained PocketBase admin extension to `db/routes/`, following the exact shape of `federation_ui.go`/`federation_ext/federation_ui.html` on the (currently unmerged) `feature/ap-instance-actors` branch: a Go file that `//go:embed`s a single HTML file and serves it behind a hand-rolled CSP header, registered via `se.Router.GET(...)` plus (optionally) a `core.UIExtension` to inject a header link into PocketBase's own `/_/` dashboard. The page itself is a zero-build AlpineJS + hand-rolled-CSS single file that authenticates by reading the PocketBase admin dashboard's own `localStorage` auth token and talks straight to PocketBase's built-in collection REST API — no new Go handlers are needed for CRUD, confirmed by the `regions`/`region_polygons` migration setting no custom collection rules (defaults to superuser-only for every action).

Three verified findings materially change what was assumed in CONTEXT.md/UI-SPEC and must be corrected during planning: (1) PocketBase v0.38.0's hard `perPage` cap is **1000**, not the "e.g. 1500" D-02 assumed — the 1,306-row catalog needs a 2-page loop, not a literal single fetch; (2) the web app's actual installed `maplibre-gl` is `^5.24.0` per `package-lock.json`, not the `4.7.1` the UI-SPEC pinned to "match web/package.json" — the UI-SPEC's own stated intent (match installed version) points to 5.24.0; (3) `feature/ap-instance-actors` — the branch holding every file this phase is told to copy — is **not merged** into the current branch (199 commits ahead, confirmed via `git merge-base`), so the canonical reference must be extracted with `git show feature/ap-instance-actors:<path>`, not read directly off disk.

The tree itself should be built from the flat `regions` rows (parent/path/depth already materialized) into an ordered array of visible rows with a `depth`/`expanded`/`visible` per row, then rendered with a single non-recursive Alpine `x-for` — Alpine has no first-class recursive-component primitive, and a flat-list-with-visibility model is both simpler and matches the fixed 36px-row/16px-indent visual spec exactly.

**Primary recommendation:** Copy `federation_ui.go` + `federation_ui.html`'s auth shell, CSP pattern, and `apiFetch()` wrapper verbatim (extracted via `git show`); replace the peers list with a flattened, filterable, non-recursive Alpine tree bound to PocketBase's raw REST API; render the map with vanilla `maplibre-gl@5.24.0` (CDN) `fill`+`line` layer pairs keyed by region path, and expand the CSP to add `unpkg.com` (script), `worker-src blob:`, and `tiles.openfreemap.org` (connect-src/img-src) — the single biggest silent-failure risk in this phase.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Region tree rendering (ADMINUI-01) | Browser / Client (AlpineJS, in-page) | API/Backend (PocketBase REST, data source) | No SSR tier exists for this page — it's a static embedded HTML file; all rendering is client-side JS against the raw PocketBase REST API |
| Toggle persistence (ADMINUI-02) | API / Backend (PocketBase built-in collection REST API) | Browser / Client (optimistic local state) | PocketBase's built-in record-update endpoint (`PATCH /api/collections/regions/records/{id}`) is the actual persistence boundary; the client only holds optimistic state pending that round-trip |
| Live coverage map (ADMINUI-03) | Browser / Client (MapLibre GL JS, in-page) | CDN / Static (OpenFreeMap tile/style/glyph hosting) | Map rendering and polygon layers are 100% client-side; the only external dependency is OpenFreeMap's publicly hosted vector tile/style CDN |
| Page serving / auth gate | API / Backend (Go `//go:embed` handler) | — | The Go handler's only job is to serve one static HTML blob with security headers (CSP, X-Frame-Options); it performs no per-request business logic |
| Superuser auth | Browser / Client (reads PocketBase dashboard's own localStorage token) | API / Backend (PocketBase validates JWT server-side on every REST call) | No new auth mechanism — piggybacks entirely on PocketBase's existing `_superusers` JWT, validated by PocketBase itself on each collection API call |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|---------------|
| Alpine.js | `3.14.8` | Reactive SPA behavior with zero build step | `[VERIFIED: npm registry]` — resolves at `https://cdn.jsdelivr.net/npm/alpinejs@3.14.8/dist/cdn.min.js` (curl-confirmed 200). This exact version/CDN is already vetted and shipped in `federation_ui.html`; reuse it verbatim per D-01/CONTEXT.md's "copy-and-extend" instruction rather than bumping to npm's current latest (`3.15.12`, confirmed via `npm view alpinejs version`), since the reuse mandate takes precedence over chasing latest. |
| MapLibre GL JS | `5.24.0` (not `4.7.1` as UI-SPEC states — see Common Pitfalls) | Renders the live coverage map + polygon layers (ADMINUI-03) | `[VERIFIED: npm registry]` — `web/package-lock.json` pins `"maplibre-gl": "^5.24.0"` (resolved), confirmed by direct file read, not the stale `4.7.1` figure in `CLAUDE.md`'s Technology Stack section or the UI-SPEC. CDN URL `https://unpkg.com/maplibre-gl@5.24.0/dist/maplibre-gl.js` (and matching `.css`) curl-confirmed 200. No breaking API changes between 4.x→5.x affect `addSource`/`addLayer`/`fitBounds`/GeoJSON fill+line usage (WebSearch cross-checked against the MapLibre GitHub breaking-changes tracker, issue #3834). |
| Remix Icon | `4.5.0` | Icon font (chevrons, search icon, badges) | `[VERIFIED: npm registry]` — resolves at `https://cdn.jsdelivr.net/npm/remixicon@4.5.0/fonts/remixicon.css` (curl-confirmed 200); reused verbatim from `federation_ui.html`, matches UI-SPEC. |
| PocketBase | `0.38.0` (Go module, already installed) | Collection REST API (list/patch), superuser JWT auth | `[VERIFIED: local go.mod + source read]` — `db/go.mod` line 11 pins `github.com/pocketbase/pocketbase v0.38.0`; behavior below verified directly against the vendored source at `~/go/pkg/mod/github.com/pocketbase/pocketbase@v0.38.0`. |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| IBM Plex Sans | weights `400;500;600` (Google Fonts CDN) | Page typography | Reused verbatim from `federation_ui.html`'s `<link>` tag — do not add weight 500 usage per UI-SPEC's typography contract, it's a legacy-only carryover. |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Flat-list Alpine `x-for` tree | True recursive Alpine `<template>` component (Ben Nadel's "template-outlet" pattern) | Recursive templates are an undocumented workaround (not a first-class Alpine feature even in 3.14.8) requiring a custom directive; the flat-list approach is native Alpine (`x-for` + `x-show`) and matches the UI-SPEC's fixed-row-height/depth-indent visual model exactly. No reason to take on the recursive-template complexity here. |
| Vanilla `fetch()` + raw REST calls | PocketBase JS SDK (`pocketbase` npm package) | The JS SDK would need a CDN UMD build and its own auth-store wiring; `federation_ui.html` deliberately avoids it (see `apiFetch()` wrapper) to stay a zero-dependency-beyond-CDN-assets static file. Keep the same approach for consistency — do not introduce the SDK. |
| Two-page `perPage=1000` loop | PocketBase's `getFullList()`-equivalent auto-pagination (only exists in the JS/Dart SDKs, not the raw REST API) | Since this phase intentionally avoids the JS SDK (see above), auto-pagination isn't available; a manual `while (page <= totalPages)` loop over 2 pages (1000 + 306) is the correct substitute. |

**Installation:**

No `npm install` / `go get` / `pip install` is required. Every dependency is a CDN `<script>`/`<link>` tag embedded directly in the new HTML file (matching `federation_ui.html`'s zero-build-step pattern) — there is no `package.json`/`go.mod` change for this phase.

```html
<!-- inside the new admin page's <head>, alongside the existing Alpine/RemixIcon/font tags -->
<link rel="stylesheet" href="https://unpkg.com/maplibre-gl@5.24.0/dist/maplibre-gl.css">
<script src="https://unpkg.com/maplibre-gl@5.24.0/dist/maplibre-gl.js"></script>
```

**Version verification:** Verified directly, not from training data —
```bash
grep '"maplibre-gl"' web/package-lock.json   # -> "^5.24.0" (resolved)
grep pocketbase db/go.mod                    # -> github.com/pocketbase/pocketbase v0.38.0
curl -sI https://unpkg.com/maplibre-gl@5.24.0/dist/maplibre-gl.js   # -> 200
```

## Package Legitimacy Audit

This phase installs **no** npm/pip/cargo packages — every dependency is a version-pinned CDN `<script>`/`<link>` tag inside a single embedded HTML file (no `package.json` or `go.mod` entries change). The Package Legitimacy Gate (slopcheck + registry verification) targets installed packages and does not apply in the same way here, but each CDN asset was still cross-checked against the npm registry for currency/legitimacy:

| Package (CDN source) | Registry | Age/Status | Downloads | Source Repo | Disposition |
|---|---|---|---|---|---|
| `alpinejs@3.14.8` | npm (served via jsdelivr) | Established project, current npm latest `3.15.12` | Very high (millions/wk) | `github.com/alpinejs/alpine` | Approved — already vetted/shipped in `federation_ui.html`; version deliberately not bumped, see Standard Stack rationale |
| `maplibre-gl@5.24.0` | npm (served via unpkg) | Established project, current npm latest `6.0.0` at research time | Very high | `github.com/maplibre/maplibre-gl-js` | Approved — matches `web/package-lock.json`'s resolved version (verified, not the stale 4.7.1 in UI-SPEC) |
| `remixicon@4.5.0` | npm (served via jsdelivr) | Established project, current npm latest `4.9.1` | Very high | `github.com/Remix-Design/RemixIcon` | Approved — already vetted/shipped in `federation_ui.html` |

**Packages removed due to slopcheck [SLOP] verdict:** none (slopcheck not run — no installable packages exist for this phase; graceful-degradation N/A because there is nothing to gate behind `checkpoint:human-verify`, these are the same CDN assets already running in production via `federation_ui.html`).
**Packages flagged as suspicious [SUS]:** none.

## Architecture Patterns

### System Architecture Diagram

```
Admin browser (already logged into PocketBase dashboard at /_/)
        │
        │ 1. GET /region-catalog/  (or similar path — see Pitfall: route naming)
        ▼
┌─────────────────────────────────────────────┐
│ Go handler (db/routes/regions_ui.go)          │
│  //go:embed regions_ext/regions_ui.html       │
│  sets CSP + X-Frame-Options, writes HTML      │
└─────────────────────────────────────────────┘
        │  (HTML+JS delivered, no server logic runs again until next request)
        ▼
┌─────────────────────────────────────────────┐
│ Browser: AlpineJS app (regionsApp())          │
│                                                │
│  init() ─┬─ read __pb_superusers__/_ from      │
│          │  localStorage → token               │
│          │  (401/expired → show "log in" CTA)  │
│          │                                     │
│          ├─ loadRegions() ── GET /api/collections/regions/records
│          │                    ?perPage=1000&page=1  (then page=2)
│          │                    Authorization: <raw JWT>
│          │                    → 1,306 flat rows
│          │                    → build tree (parent/path/depth)
│          │                    → flatten to visible-row array
│          │                    → auto-expand branches w/ enabled leaves (D-07)
│          │
│          ├─ loadEnabledPolygons() ── GET /api/collections/region_polygons/records
│          │                            ?filter=(path='a'||path='b'||...)
│          │                            → render fill+line layer per polygon
│          │
│          ├─ toggleRegion(leaf) ── optimistic flip (D-06)
│          │                         PATCH /api/collections/regions/records/{id}
│          │                         { enabled: true|false }
│          │                         success → keep; failure → revert + inline error
│          │
│          └─ filterInput (debounced 120ms) ── substring match over
│                                                already-fetched rows,
│                                                auto-expand ancestor chain (D-08)
│                                                │
│                                                ▼
│                                     MapLibre GL JS map instance
│                                     style: tiles.openfreemap.org/styles/liberty
│                                     fitBounds(union of enabled bboxes) on load (D-05)
└─────────────────────────────────────────────┘

Elsewhere, unaffected by this phase:
  Archive-generation cron (Phase 29) ── reads regions WHERE kind='leaf' AND enabled=true
                                          on its own schedule, no signal from this page
```

### Recommended Project Structure

```
db/routes/
├── regions_ui.go              # Go embed handler + UIExtension registration (mirrors federation_ui.go)
├── regions_ext/
│   ├── regions_ui.html        # single-file AlpineJS SPA (mirrors federation_ext/federation_ui.html)
│   └── main.js                # header-link injection into PocketBase's own /_/ dashboard (mirrors federation_ext/main.js)
```

### Pattern 1: Go embed + CSP-locked static page handler

**What:** A Go file `//go:embed`s a single HTML file and serves it with a restrictive `Content-Security-Policy`, `X-Frame-Options: DENY`, and `X-Content-Type-Options: nosniff` — no per-request logic beyond writing the static bytes.
**When to use:** Any custom PocketBase admin page in this codebase (established pattern, not new to this phase).
**Example:**
```go
// Source: feature/ap-instance-actors:db/routes/federation_ui.go (verified via `git show`)
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
	// See Common Pitfalls: this CSP MUST be widened beyond federation_ui.go's,
	// not copied verbatim, or MapLibre silently fails to render tiles/workers.
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

### Pattern 2: Route + UIExtension registration in `main.go`

**What:** Register the page's own route (distinct prefix from `/regions`, see Pitfall below) and, optionally, inject a header link into PocketBase's built-in `/_/` dashboard via `core.UIExtension` — an experimental API (introduced v0.37.0) that this codebase's installed v0.38.0 supports.
**Example:**
```go
// Source: pattern verified from feature/ap-instance-actors:db/main.go via `git show`,
// path must be added to THIS branch's main.go (not present here yet)
se.Router.GET("/region-catalog/", routes.RegionsDashboard)

se.UIExtensions = append(se.UIExtensions, core.UIExtension{
	Name: "wanderer-region-catalog",
	FS:   routes.RegionsExtFS(),
})
```

### Pattern 3: Flat-list tree (not recursive templates)

**What:** Build the tree as an in-memory `Map<id, node>` (children arrays) from the flat `regions` rows, then derive a single flat **visible-rows array** (`{id, name, kind, depth, enabled, expanded, hasEnabledDescendant}`) that Alpine's `x-for` renders directly — expand/collapse and filtering only ever mutate this derived array, never require actual DOM recursion.
**When to use:** Any tree UI in a framework (Alpine) without a native recursive-component primitive.
**Example:**
```js
// Building the tree from flat rows (regions.parent/path/depth are already materialized)
function buildTree(rows) {
  const byId = new Map(rows.map(r => [r.id, { ...r, children: [] }]));
  const roots = [];
  for (const node of byId.values()) {
    if (node.parent && byId.has(node.parent)) byId.get(node.parent).children.push(node);
    else roots.push(node);
  }
  for (const node of byId.values()) node.children.sort((a, b) => a.sort_order - b.sort_order);
  return roots;
}

// Flattening to visible rows respects `expanded` state per branch (D-07 default:
// only branches containing an enabled leaf start expanded)
function flattenVisible(roots, expandedSet, filterMatchSet) {
  const out = [];
  (function walk(nodes, depth) {
    for (const n of nodes) {
      if (filterMatchSet && !filterMatchSet.has(n.id)) continue; // D-08: filter hides non-matches/non-ancestors
      out.push({ id: n.id, name: n.name, kind: n.kind, depth, enabled: n.enabled });
      if (n.kind === 'group' && expandedSet.has(n.id)) walk(n.children, depth + 1);
    }
  })(roots, 0);
  return out;
}
```
```html
<!-- Alpine template: single non-recursive x-for over the derived flat array -->
<template x-for="row in visibleRows" :key="row.id">
  <div class="tree-row" :style="`padding-left:${8 + row.depth * 16}px`" role="treeitem"
       :aria-expanded="row.kind === 'group' ? expanded.has(row.id) : null">
    <template x-if="row.kind === 'group'">
      <i @click="toggleExpand(row.id)"
         :class="expanded.has(row.id) ? 'ri-arrow-down-s-line' : 'ri-arrow-right-s-line'"></i>
    </template>
    <span x-text="row.name"></span>
    <template x-if="row.kind === 'leaf'">
      <button role="switch" :aria-checked="row.enabled"
              @click="toggleRegion(row)"></button>
    </template>
  </div>
</template>
```

### Pattern 4: Two-page fetch loop (perPage cap correction)

**What:** PocketBase v0.38.0's raw REST API caps `perPage` at 1000 (`tools/search.MaxPerPage`, verified in vendored source) — a value above that is silently normalized down, not rejected. 1,306 rows requires two requests.
**Example:**
```js
async function loadAllRegions() {
  let page = 1, all = [], totalPages = 1;
  do {
    const res = await apiFetch(`/api/collections/regions/records?perPage=1000&page=${page}&sort=sort_order`);
    const data = await res.json();
    all = all.concat(data.items);
    totalPages = data.totalPages;
    page++;
  } while (page <= totalPages);
  return all; // 1,306 rows in 2 requests (1000 + 306)
}
```

### Pattern 5: Enabled-leaf polygon fetch (OR-filter, chunked)

**What:** `region_polygons` has no relation back to `regions`, so enabled leaves must be resolved first (from the already-fetched `regions` rows), then their `path`s joined with `||` into a single filter — chunked to stay under PocketBase's `MaxFilterLength` (3500 chars, verified in vendored source).
**Example:**
```js
async function loadEnabledPolygons(enabledPaths) {
  const CHUNK = 60; // conservative: keeps filter well under the 3500-char cap for typical path lengths
  const polygons = [];
  for (let i = 0; i < enabledPaths.length; i += CHUNK) {
    const chunk = enabledPaths.slice(i, i + CHUNK);
    const filter = chunk.map(p => `path='${p.replace(/'/g, "\\'")}'`).join('||');
    const res = await apiFetch(`/api/collections/region_polygons/records?perPage=${CHUNK}&filter=${encodeURIComponent(filter)}`);
    const data = await res.json();
    polygons.push(...data.items);
  }
  return polygons;
}
```

### Pattern 6: MapLibre fill+line polygon pair per region

**What:** Each enabled leaf gets one GeoJSON source + a `fill` (18% opacity accent) + `line` (2px solid accent) layer pair, added/removed as toggles fire (D-06).
**Example:**
```js
function addRegionLayer(map, path, geojsonPolygon) {
  map.addSource(`region-${path}`, { type: 'geojson', data: { type: 'Feature', geometry: geojsonPolygon, properties: {} } });
  map.addLayer({ id: `region-fill-${path}`, type: 'fill', source: `region-${path}`,
    paint: { 'fill-color': accentColor, 'fill-opacity': 0.18 } });
  map.addLayer({ id: `region-line-${path}`, type: 'line', source: `region-${path}`,
    paint: { 'line-color': accentColor, 'line-width': 2 } });
}

function removeRegionLayer(map, path) {
  if (map.getLayer(`region-line-${path}`)) map.removeLayer(`region-line-${path}`);
  if (map.getLayer(`region-fill-${path}`)) map.removeLayer(`region-fill-${path}`);
  if (map.getSource(`region-${path}`)) map.removeSource(`region-${path}`);
}

// Union bbox + fitBounds (D-05) — regions.bbox is already [minLon,minLat,maxLon,maxLat],
// the exact shape MapLibre's fitBounds accepts as [[minLon,minLat],[maxLon,maxLat]]
function fitToEnabled(map, enabledLeafRows) {
  if (enabledLeafRows.length === 0) {
    map.jumpTo({ center: [10, 45], zoom: 3 }); // D-05 fallback per UI-SPEC's "Map empty state"
    return;
  }
  const [minLon, minLat, maxLon, maxLat] = enabledLeafRows.reduce((acc, r) => [
    Math.min(acc[0], r.bbox[0]), Math.min(acc[1], r.bbox[1]),
    Math.max(acc[2], r.bbox[2]), Math.max(acc[3], r.bbox[3]),
  ], [Infinity, Infinity, -Infinity, -Infinity]);
  map.fitBounds([[minLon, minLat], [maxLon, maxLat]], { padding: 48 });
}
```

### Anti-Patterns to Avoid
- **Recursive Alpine `<template>` components for the tree:** undocumented/unsupported workaround territory even in Alpine 3.14.8; use the flat-visible-rows model (Pattern 3) instead.
- **Adding custom `listRule`/`updateRule` to `regions`/`region_polygons` "to make the admin page's life easier":** unnecessary — D-01's whole design relies on the default (nil = superuser-only) rule already in place from the Phase 28 migration; adding a rule would also (accidentally) open the collection to any authenticated end user via the client-facing `/regions` API path space.
- **Copying `federation_ui.go`'s CSP header verbatim:** its `default-src 'none'` + jsdelivr-only allowlist will silently block MapLibre's tile/glyph/sprite fetches and Web Worker creation — see Common Pitfalls.
- **Re-fitBounds on every individual toggle:** D-06 explicitly scopes `fitBounds` to initial load only; re-fitting on each toggle would disorient the admin mid-session.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Superuser session validation | Custom JWT decode/expiry logic | Reuse the exact `isTokenExpired()` (base64-decode `.exp` claim, compare to `Date.now()/1000`) already proven in `federation_ui.html` | It's a two-line, already-audited pattern; a subtly different reimplementation risks a session-handling bug in security-sensitive code |
| Union bounding box across many regions | A geometry library (turf.js etc.) for a min/max reduce | Plain `Array.reduce` over `bbox` arrays (Pattern 4 above) | `regions.bbox` is already a flat `[minLon,minLat,maxLon,maxLat]`; union of axis-aligned bboxes is a trivial min/max reduce, not a geometry-library problem |
| GeoJSON Polygon/MultiPolygon rendering | Custom canvas/SVG polygon renderer | MapLibre GL JS `fill`/`line` layers on a `geojson` source | MapLibre natively accepts both `Polygon` and `MultiPolygon` geometry types in one Feature with zero extra handling |
| Pagination beyond PocketBase's 1000-row cap | A recursive/cursor-based custom pagination scheme | A simple `while (page <= totalPages)` loop (Pattern 4) | PocketBase's list response already returns `totalPages`; 1,306 rows is only 2 pages, no need for anything more elaborate |

**Key insight:** Every "hard part" of this phase (auth, pagination, polygon rendering, bbox union) already has an established, narrow, correct solution either in the existing codebase (`federation_ui.html`) or as a one-line reduce/loop — resist the urge to add a geometry or tree library for what is fundamentally a ~1,300-row, 3-level-deep, already-materialized-path dataset.

## Common Pitfalls

### Pitfall 1: `feature/ap-instance-actors` is not merged — the canonical reference doesn't exist on disk here
**What goes wrong:** CONTEXT.md and the UI-SPEC repeatedly instruct "read/copy `db/routes/federation_ext/federation_ui.html`" as if it's a file in the working tree. On the current branch (`feature/app`) it does not exist — `git merge-base --is-ancestor feature/ap-instance-actors HEAD` returns false, and the branch is 199 commits ahead of what's merged.
**Why it happens:** The canonical reference was built and shipped on a sibling feature branch that hasn't landed yet.
**How to avoid:** Extract the file(s) with `git show feature/ap-instance-actors:db/routes/federation_ext/federation_ui.html > /tmp/federation_ui.html` (and same for `federation_ui.go`, `main.js`) before reading/copying — do not `Read` the path directly, it will 404.
**Warning signs:** A `Read` tool call against `db/routes/federation_ext/federation_ui.html` on this branch returns "file does not exist."

### Pitfall 2: PocketBase's `perPage` hard-caps at 1000, not "1500"
**What goes wrong:** D-02 assumed "a single `regions` list call with `perPage` overridden past PocketBase's default 30/page cap (e.g. 1500)" would return all 1,306 rows in one request. PocketBase v0.38.0 silently normalizes any `perPage` above 1000 down to 1000 (verified in `tools/search/provider.go`: `MaxPerPage int = 1000`), so a `perPage=1500` request returns only 1000 rows with no error — the 306 remaining rows (mostly deep-tree leaves, since the seed is parent-before-child) would silently vanish from the tree.
**Why it happens:** The design decision assumed PocketBase enforces no hard cap, or a higher one; training-data-era PocketBase versions had a 500 cap, adding to the confusion.
**How to avoid:** Loop `page = 1, 2, ...` while `page <= response.totalPages`, requesting `perPage=1000` each time (2 requests total for 1,306 rows). See Pattern 4.
**Warning signs:** Tree renders but is missing entire deep sub-branches, or the tree's total leaf count doesn't match 1,153.

### Pitfall 3: MapLibre GL JS needs a much wider CSP than `federation_ui.go`'s
**What goes wrong:** Copying `federation_ui.go`'s CSP header verbatim (`default-src 'none'; script-src ... https://cdn.jsdelivr.net; ... connect-src 'self'; img-src 'self'`) will silently break the map: MapLibre creates Web Workers from `blob:` URLs (blocked without `worker-src blob:`), fetches the style JSON/vector tiles/sprite/glyphs from `tiles.openfreemap.org` (blocked without that host in `connect-src` and `img-src`), and the map library itself is loaded from `unpkg.com` (blocked without that host in `script-src`/`style-src`).
**Why it happens:** CSP failures for `fetch`/`Worker` calls fail silently in the console with a generic "Refused to..." message that's easy to miss during manual testing if the admin doesn't open devtools; the map pane just shows a blank background or spins forever.
**How to avoid:** Use the widened CSP header in Pattern 1 above: add `unpkg.com` to `script-src`/`style-src`, add `worker-src blob:`, and add `tiles.openfreemap.org` to both `connect-src` and `img-src`.
**Warning signs:** Map pane stays blank/gray or the loading spinner never resolves; browser console shows `Refused to connect to 'https://tiles.openfreemap.org/...' because it violates the following Content Security Policy directive`.

### Pitfall 4: UI-SPEC's MapLibre version (4.7.1) doesn't match the actual installed web app version (5.24.0)
**What goes wrong:** The approved UI-SPEC pins `MapLibre GL JS 4.7.1 (CDN, unpkg.com/maplibre-gl@4.7.1)` with the stated rationale "version pinned to match `web/package.json`'s `maplibre-gl` dependency." That rationale is factually wrong at execution time: `web/package.json`/`package-lock.json` actually pin `^5.24.0` (verified by direct file read — `CLAUDE.md`'s Technology Stack table, which the UI-SPEC's author likely consulted, is itself stale on this and several other dependency versions).
**Why it happens:** `CLAUDE.md`'s dependency table is a point-in-time snapshot that has drifted from `package.json`/`package-lock.json`.
**How to avoid:** Use `5.24.0` (the verified, actually-installed version) for the CDN `<script>`/`<link>` tags — this satisfies the UI-SPEC's own stated intent (match the web app) even though it contradicts the literal version number written in the document. Flag this discrepancy to the user/planner rather than silently picking one; no breaking API changes affect this phase's usage (fill/line layers, fitBounds, GeoJSON sources — cross-checked against MapLibre's v5 breaking-changes list).
**Warning signs:** None at runtime (4.7.1 would also technically work) — this is a documentation-accuracy issue, not a functional bug, but worth resolving before implementation so future maintenance doesn't have two different pinned-version stories.

### Pitfall 5: `federation_ui.html` does not actually load Tailwind CDN
**What goes wrong:** The UI-SPEC's Registry Safety section states "every external asset (Tailwind Play CDN, Alpine.js 3.14.8, Remix Icon 4.5.0, MapLibre GL JS 4.7.1, Google Fonts)... matching the exact pattern already vetted and shipped in `federation_ui.html`." A full-text search of the actual `federation_ui.html` (`grep -in tailwind`) returns zero matches — the canonical reference uses only hand-rolled CSS custom properties and classes (`.card`, `.btn`, `.input`, etc.), no Tailwind at all.
**Why it happens:** Likely conflated with an earlier draft or a different admin page; the shipped file is 100% hand-rolled CSS.
**How to avoid:** Do not add a Tailwind CDN `<script>` tag — style the new tree/toggle/map components using the same hand-rolled CSS custom-property system already defined in `federation_ui.html`'s `<style>` block (`--bg`, `--surface`, `--surface2`, `--accent`, etc.), extending it with the new component classes the UI-SPEC calls out (tree row, toggle switch, map pane).
**Warning signs:** An extra ~50KB CDN script tag with no corresponding utility classes actually used anywhere in the page.

### Pitfall 6: New admin route path must not collide with the existing `/regions` API group
**What goes wrong:** The current branch's `main.go` already registers `se.Router.Group("/regions")` bound to `apis.RequireAuth()` (any logged-in user, for the client-facing `GET /regions`, `/regions/{id}/download`, `/regions/{id}/download-dem` routes — Phase 29, unrelated to this phase). Naming the new admin page's route `/regions/admin/` or similar risks either an outright router conflict or, worse, inheriting the wrong auth middleware (any-user instead of superuser-gated-by-page-content) if added inside that same group.
**Why it happens:** The obvious naming choice ("regions" + "admin") collides with an existing, differently-scoped route group.
**How to avoid:** Register the new page under a distinct top-level path, e.g. `/region-catalog/` (mirroring `/federation/`'s naming precedent) — a sibling `se.Router.GET(...)` call, not nested under the existing `regionsGroup`.
**Warning signs:** Router registration panics at startup, or the admin page unexpectedly requires only regular-user auth instead of being gated by the superuser-only collection rules it relies on.

### Pitfall 7: `region_polygons` filter length can exceed PocketBase's 3500-char cap if many regions are enabled
**What goes wrong:** D-03's design assumes a single filter request naming every currently-enabled leaf's `path`. PocketBase v0.38.0 enforces `MaxFilterLength = 3500` chars (verified in vendored source). If an admin enables a large number of leaves (e.g. most of a continent), the `path='...' || path='...' || ...` filter string could exceed that cap and the request would fail validation.
**Why it happens:** No chunking was specified in D-03; it reads as "fetch polygons for currently-enabled leaves" without addressing scale.
**How to avoid:** Chunk the enabled-leaf `path` list into batches (Pattern 5 uses 60 per request as a conservative default — average path length in this seed appears well under 50 chars, so 60 chunks stays comfortably under 3500) and issue multiple requests, merging results client-side.
**Warning signs:** Map stops updating polygons (silently, or with a 400 from PocketBase) once a large number of regions are enabled in one session; works fine in small manual tests with only a handful enabled.

## Runtime State Inventory

Not applicable — this is a greenfield UI addition (new Go route file + new embedded HTML file + one `main.go` registration diff), not a rename/refactor/migration phase. No existing stored data, live service config, OS-registered state, secrets, or build artifacts reference anything being renamed or moved.

## Code Examples

### Optimistic toggle with revert-on-failure (D-06)
```js
// Source: pattern adapted from feature/ap-instance-actors:db/routes/federation_ext/federation_ui.html's
// approve()/reject()/disconnect() handlers (verified via `git show`), applied to a single leaf toggle.
async toggleRegion(row) {
  const prevEnabled = row.enabled;
  row.enabled = !prevEnabled;                     // 1. optimistic flip
  if (row.enabled) this.addPolygonForRow(row);     // 2. optimistic map update
  else this.removePolygonForRow(row);
  this.rowErrors = { ...this.rowErrors, [row.id]: undefined };

  const res = await this.apiFetch(`/api/collections/regions/records/${row.id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ enabled: row.enabled }),
  });
  if (!res) return; // apiFetch already handled 401 -> logged out
  if (!res.ok) {
    row.enabled = prevEnabled;                     // 3. revert on failure
    if (prevEnabled) this.addPolygonForRow(row); else this.removePolygonForRow(row);
    const data = await res.json().catch(() => ({}));
    this.rowErrors = { ...this.rowErrors, [row.id]: `Couldn't update${data.message ? ': ' + data.message : ''} — try again.` };
  }
}
```

### Auto-expand branches containing an enabled leaf (D-07)
```js
function computeDefaultExpanded(roots) {
  const expanded = new Set();
  function hasEnabledDescendant(node) {
    if (node.kind === 'leaf') return !!node.enabled;
    const any = node.children.some(hasEnabledDescendant);
    if (any) expanded.add(node.id);
    return any;
  }
  roots.forEach(hasEnabledDescendant);
  return expanded;
}
```

### Filter with ancestor auto-expand (D-08)
```js
function computeFilterMatches(roots, query) {
  if (!query) return null; // null = no filter active, show per expandedSet as normal
  const q = query.toLowerCase();
  const matchSet = new Set();
  function walk(node, ancestors) {
    const selfMatch = node.name.toLowerCase().includes(q);
    let childMatch = false;
    for (const child of node.children) {
      if (walk(child, [...ancestors, node.id])) childMatch = true;
    }
    if (selfMatch || childMatch) {
      matchSet.add(node.id);
      ancestors.forEach(a => matchSet.add(a));
      return true;
    }
    return false;
  }
  roots.forEach(r => walk(r, []));
  return matchSet;
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Hand-authored `region_config.json` bbox list | Seeded `regions`/`region_polygons` PocketBase collections, toggled via this admin page | Phase 28 (already shipped) | This phase is purely the UI layer on top of an already-complete, already-populated data model — no schema work remains |

No other "old vs. new" migration applies within this phase's own scope — it's additive UI on a stable backend.

**Deprecated/outdated:** `region_config.json` / `REGION_CATALOG_CONFIG_PATH` loader — fully retired by Phase 29 (EXTRACT-02), not touched by this phase.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|----------------|
| A1 | The new admin page's route path should be `/region-catalog/` (not `/regions-admin/`, `/region-picker/`, etc.) | Architecture Patterns, Pitfall 6 | Low — any distinct top-level path avoids the collision; the exact string is a naming preference for the planner/discuss-phase to confirm, not a functional requirement |
| A2 | 60 paths per filter chunk is a safe default for the `region_polygons` OR-filter batching | Pattern 5, Pitfall 7 | Low-medium — if actual `path` strings in the seed run longer than assumed, 60×~50 chars could approach the 3500 cap; the planner should have the executor compute an actual max-observed `path` length from the seed data and size the chunk conservatively (e.g. `Math.floor(3400 / (maxPathLen + 10))`) rather than hardcoding 60 |
| A3 | `core.UIExtension` header-link injection (Pattern 2) is worth reusing for this phase, not just the route itself | Architecture Patterns Pattern 2 | Low — purely cosmetic (a nav link inside `/_/`); omitting it doesn't block ADMINUI-01/02/03, the page is still reachable by direct URL |

**Assumptions carried over from CONTEXT.md/UI-SPEC that this research corrected (not left as open assumptions, but flagged for planner awareness):** the `perPage=1500` figure (D-02, corrected to a 2-page 1000-cap loop, HIGH confidence/verified), the `maplibre-gl@4.7.1` pin (UI-SPEC, corrected to `5.24.0`, HIGH confidence/verified), and the "Tailwind Play CDN" claim (UI-SPEC Registry Safety, corrected to hand-rolled CSS only, HIGH confidence/verified) — these are not marked `[ASSUMED]` above because they were independently verified against the actual codebase/registry, not left as unverified claims.

## Open Questions

1. **Should the admin page's route be nested under `/_/` (PocketBase's own SPA) or standalone like `/federation/`?**
   - What we know: `federation_ui.html` is a fully standalone page at `/federation/`, reached via a `core.UIExtension` header link from `/_/`, not embedded inside PocketBase's own Vue app.
   - What's unclear: Whether the discuss-phase/planner wants the exact same standalone-page pattern (very likely, given D-01 explicitly cites `federation_ui.html` as "the pattern") or something tighter.
   - Recommendation: Follow the standalone-page pattern verbatim (Pattern 1/2 above) — it's the explicitly named precedent and requires no PocketBase core-UI modification.

2. **Exact final route path string (`/region-catalog/` vs. alternatives).**
   - What we know: It must not collide with the existing `/regions` API group (Pitfall 6).
   - What's unclear: No explicit naming decision was made in CONTEXT.md.
   - Recommendation: Planner picks a short, distinct, human-readable path (e.g. `/region-catalog/`); low-stakes, easily changed later since it's admin-only tooling with no external consumers.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|--------------|-----------|---------|----------|
| Go toolchain / `db/` module | Building `regions_ui.go` | ✓ (existing project dependency) | Go module `pocketbase v0.38.0` per `db/go.mod` | — |
| `feature/ap-instance-actors` branch (for canonical reference extraction) | Copying `federation_ui.html`/`federation_ui.go`/`main.js` | ✓ — branch exists locally and on `origin`, but is **not merged** (see Pitfall 1) | 199 commits ahead of current branch's merge-base | Use `git show feature/ap-instance-actors:<path>` to extract file contents without merging the branch |
| `https://tiles.openfreemap.org` (OpenFreeMap style/tiles/glyphs/sprite) | ADMINUI-03 map rendering | ✓ — public, no API key, already used by the web app's own default style | N/A (external hosted service) | None specified in CONTEXT.md; D-04 explicitly rejected vendoring a local copy |
| `https://unpkg.com` / `https://cdn.jsdelivr.net` (CDN asset hosting) | Loading Alpine.js/MapLibre/RemixIcon | ✓ — curl-confirmed 200 for all three pinned asset URls during this research session | — | None — matches the existing `federation_ui.html` zero-build-step approach |

**Missing dependencies with no fallback:** none — every dependency this phase needs is already available (either in-repo or as a live, already-relied-upon external CDN/service).
**Missing dependencies with fallback:** none.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|----------------|---------|-------------------|
| V2 Authentication | Yes | No new auth mechanism — reuses PocketBase's existing `_superusers` JWT (read from the dashboard's own `localStorage` key, validated server-side by PocketBase on every collection API call). Client-side `isTokenExpired()` check is a UX nicety (redirect-to-login), not a security boundary — the real enforcement is PocketBase's own JWT validation on each REST call. |
| V3 Session Management | Yes | Session lifetime/expiry entirely owned by PocketBase's own superuser auth (unchanged by this phase); this page never creates, extends, or stores its own session — it only reads the token PocketBase's dashboard already manages. |
| V4 Access Control | Yes | Collection-level access control is enforced by PocketBase itself via the `regions`/`region_polygons` collections' default (nil = superuser-only) API rules — **do not add custom `listRule`/`updateRule`** (see Anti-Patterns); doing so would weaken access control below what Phase 28's migration intentionally set. |
| V5 Input Validation | Yes | The only mutation this phase performs is `PATCH .../regions/records/{id}` with body `{ enabled: <bool> }` — PocketBase's own `BoolField` validation on the `regions` collection schema rejects non-boolean values server-side; the Alpine toggle should only ever send a literal `true`/`false`, never pass through unsanitized user text. |
| V6 Cryptography | No | No new cryptographic operations introduced by this phase (JWT signing/verification is entirely PocketBase's existing responsibility). |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|-----------------------|
| Clickjacking of the admin page (framing to trick an authenticated admin into an unintended toggle click) | Tampering | `X-Frame-Options: DENY` (already the established pattern in `federation_ui.go`; carry forward verbatim) |
| CSRF against the PATCH toggle endpoint | Spoofing/Tampering | Not applicable in the traditional cookie-session sense — PocketBase's REST API authenticates via an explicit `Authorization: <JWT>` header the browser never sends automatically (unlike cookies), so a cross-site form/script cannot forge this request without already having read access to the token from `localStorage` (same-origin only) |
| XSS via region name rendering | Tampering/Information Disclosure | Region `name` values originate from the static, maintainer-curated CoMaps seed (not end-user input) — still, always bind with Alpine's `x-text` (auto-escaped), never `x-html`, for any region name/path/error-message rendering |
| Reflected/stored injection via the filter input | Tampering | The filter box (D-08) only performs a client-side `String.includes()` substring match over already-fetched data — it issues no server request and constructs no filter string from that input, so it cannot inject into the `region_polygons` OR-filter (Pattern 5's filter is built from already-validated `path` values fetched from the server, not from the filter box's free text) |
| Over-broad collection access if rules are "temporarily" relaxed for debugging | Elevation of Privilege | Never set `listRule`/`viewRule`/`updateRule` on `regions`/`region_polygons` to anything other than the default `nil` (superuser-only) — flagged explicitly in Anti-Patterns and V4 above |

## Sources

### Primary (HIGH confidence)
- Local repo file reads: `db/migrations/1785000000_create_regions_collection.go`, `db/routes/regions_get.go`, `db/main.go`, `db/go.mod`, `web/package-lock.json`, `web/static/styles/ofm.json`, `web/src/lib/vendor/maplibre-layer-manager/layers.ts`, `.planning/notes/streamlined-region-definition.md`
- `git show feature/ap-instance-actors:db/routes/federation_ui.go` / `federation_ext/federation_ui.html` / `federation_ext/main.js` — canonical reference, extracted since the branch is unmerged
- Vendored Go source, `~/go/pkg/mod/github.com/pocketbase/pocketbase@v0.38.0/`: `apis/record_crud.go` (route registration, PATCH verb), `tools/search/provider.go` (`DefaultPerPage=30`, `MaxPerPage=1000`, `MaxFilterLength=3500`), `core/events.go` (`UIExtension` type), `ui/src/pb.js` (actual `LocalAuthStore` key construction: `"__pb_superusers__" + currentPath`)
- `web/node_modules/pocketbase/dist/pocketbase.cjs.js` — confirmed `{token, record}` LocalAuthStore persisted shape
- `curl -sI` against `unpkg.com/maplibre-gl@5.24.0/...`, `cdn.jsdelivr.net/npm/alpinejs@3.14.8/...`, `cdn.jsdelivr.net/npm/remixicon@4.5.0/...` — all 200
- `npm view alpinejs version` / `npm view remixicon version` / `npm view maplibre-gl version` — registry currency check

### Secondary (MEDIUM confidence)
- [pocketbase.io/docs/api-rules-and-filters](https://pocketbase.io/docs/api-rules-and-filters/) — null rule = superuser-only (WebSearch, cross-checked against local migration having no custom rules set)
- [pocketbase.io/docs/authentication](https://pocketbase.io/docs/authentication/) — `Authorization: <raw token>` header format, no `Bearer` prefix required
- GitHub `pocketbase/pocketbase` discussions #1823/#274 — OR-filter as the only multi-value filter mechanism (no native `IN`)
- GitHub `maplibre/maplibre-gl-js` issue #3834 — v5 breaking-changes tracker, none affect this phase's usage
- MDN `Content-Security-Policy/worker-src` + MapLibre GitHub discussion #4424 — `worker-src blob:`/`img-src blob:` requirement for MapLibre under CSP

### Tertiary (LOW confidence)
- None — every material claim above was either verified directly against local source/registry or corroborated by an official docs page.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — every version verified directly (package-lock.json, go.mod, curl against CDN, npm registry), not from training-data memory
- Architecture: HIGH — pattern extracted directly from the canonical reference file via `git show`, not reconstructed from memory
- Pitfalls: HIGH — all seven pitfalls are independently verified facts (perPage cap, filter length cap, CSP requirements, branch-merge status, dependency-version drift), not speculative

**Research date:** 2026-07-27
**Valid until:** 30 days (stable internal APIs; the one fast-moving external factor is `feature/ap-instance-actors` merging, which would change Pitfall 1's guidance — re-check merge status before planning if this research is consumed later than a few days out)
