# Phase 30: Admin Region Picker UI - Context

**Gathered:** 2026-07-27
**Status:** Ready for planning

<domain>
## Phase Boundary

A server owner manages the region catalog visually on a custom PocketBase admin page: a collapsible tree over the seeded 1,306-row hierarchy (1,153 leaf + 153 group, depth 0-2) where toggling a leaf's `enabled` flag persists immediately, plus a live MapLibre map showing the boundary polygons of every currently-enabled leaf region. Covers ADMINUI-01, ADMINUI-02, ADMINUI-03 only — no changes to the cron, the extraction pipeline, or the client-facing `GET /api/v1/regions` API (those are Phase 29, already shipped).

</domain>

<decisions>
## Implementation Decisions

### Data access pattern
- **D-01:** No new custom Go endpoints for listing/toggling regions. The admin page authenticates as a PocketBase superuser (same pattern as `federation_ui.html` on `feature/ap-instance-actors`: read `__pb_superusers__/_` from localStorage, redirect to `/_/` if missing/expired) and talks directly to PocketBase's built-in collection REST API (`/api/collections/regions/records`, `/api/collections/region_polygons/records`). Neither collection currently has custom list/view/update rules, so they default to superuser-only — matching this access pattern with zero rule changes required.
- **D-02:** Tree hierarchy is loaded with **one full fetch** — a single `regions` list call with `perPage` overridden past PocketBase's default 30/page cap (e.g. 1500) — and the parent/child tree is assembled client-side from `parent`/`path`/`depth`. Rejected lazy-per-expand loading: 1,306 flat hierarchy rows (no polygon/bbox bulk) is a small enough payload that round-tripping on every expand click isn't worth the added latency.
- **D-03:** `region_polygons` (a separate, deliberately-split-out table at ~165KB/leaf — see `db/migrations/1785000000_create_regions_collection.go`) is fetched **only for currently-enabled leaves**, both on initial load and again for the single region whenever a toggle flips. Rejected: fetching all 1,153 polygons upfront (payload would be hundreds of MB) and lazy-fetch-on-group-expand (would hide enabled regions sitting in collapsed branches).

### Live map
- **D-04:** Map style is MapLibre GL JS (CDN, no build step available on this standalone page) pointed directly at OpenFreeMap's public hosted style URL (`https://tiles.openfreemap.org/styles/liberty`) — same tile family as the web app's default (`web/static/styles/ofm.json`'s `openmaptiles` vector source), no API key, no file to keep in sync. Rejected copying `ofm.json` into `db/routes/` (duplicated-file drift risk) and plain raster OSM tiles (against OSM's tile usage policy).
- **D-05:** On page load, the map auto-fits its bounds to the union of all currently-enabled leaf polygons, so the admin sees existing coverage at a glance without manual panning.
- **D-06:** Toggling a region is **optimistic** end-to-end: the checkbox/switch flips and the map polygon appears/disappears immediately on click, before the PATCH resolves. On PATCH failure: revert the toggle to its prior state, remove/re-add the map polygon to match, and show an inline error next to that specific tree row (not a global toast). No dedicated retry button — the reverted toggle is already clickable again, so re-clicking re-fires the same PATCH.

### Tree UX
- **D-07:** Default expand state is neither fully collapsed nor fully expanded: only branches that contain at least one already-`enabled` leaf are auto-expanded on load; everything else starts collapsed. Gives the admin an at-a-glance view of current state without a 1,306-row wall of nodes.
- **D-08:** A simple client-side name-filter box is in scope for this phase (not deferred) — it filters over the already-fetched 1,306 rows with no extra API call, given the tree's size (153 groups, 3 levels deep) makes unaided scrolling impractical. Typing in the filter narrows the tree to matches plus their ancestor chain, auto-expanded, so results are visible without additional manual expand clicks.

### Claude's Discretion
- Exact visual styling of the inline per-row error message (color, icon, dismiss behavior) — Tailwind CDN, consistent with `federation_ui.html`'s existing look.
- Whether the filter box does substring or fuzzy matching — substring is sufficient given the decision was purely about surfacing ancestors, not match ranking.
- Debounce timing on the filter input, if any.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Design rationale
- `.planning/notes/streamlined-region-definition.md` — the full design decision trail for why a curated seeded catalog + toggle (not freehand draw) was chosen, and why the admin page reuses the `feature/ap-instance-actors` custom-PocketBase-page pattern.

### Data model (Phase 28/29 — already shipped, read-only for this phase)
- `db/migrations/1785000000_create_regions_collection.go` — `regions` schema (comaps_id/path/depth/sort_order/name/kind/bbox/enabled/parent) and the **separate** `region_polygons` collection (keyed by `path`, deliberately split out to keep hierarchy queries free of ~165KB/row boundary geometry). No custom collection rules are set on either — both default to superuser-only access, which the admin page relies on directly.
- `db/routes/regions_get.go` — the existing `GET /api/v1/regions` handler (client-facing, bbox-only, joins `region_archives` for build status). Not used by the admin page — it's a *different* consumer of the same `regions` table, kept unchanged by this phase.

### Reusable pattern (branch `feature/ap-instance-actors`, not yet merged)
- `db/routes/federation_ui.html` (commit `ed0f31eb`) — the template for this phase's admin page: self-contained Alpine.js + Tailwind CDN SPA, synchronous theme IIFE, superuser-token auth via `__pb_superusers__/_` localStorage read + 401/expiry redirect to `/_/`, `apiFetch()` wrapper attaching the raw JWT (no `Bearer` prefix) as `Authorization`. Copy this auth/shell pattern; the region tree + map are new.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `db/routes/federation_ui.html` (`feature/ap-instance-actors`, commit `ed0f31eb`) — full auth shell, theme handling, and `apiFetch()` pattern to copy into the new region-picker page.
- Web app's OpenFreeMap style config (`web/src/lib/vendor/maplibre-layer-manager/layers.ts:46-100+`, `web/static/styles/ofm.json`) — reference for which OpenFreeMap style URL/sprite/glyph endpoints to point at directly (no local copy needed per D-04).

### Established Patterns
- Custom PocketBase admin pages in this codebase are single self-contained HTML files with CDN-loaded Alpine.js + Tailwind, served as a static route — no SvelteKit/Vite build involved. This phase's page follows the same shape.
- `region_polygons` is intentionally kept out of `regions` and out of the client-facing API specifically so bulk hierarchy/listing reads never carry full-precision boundary geometry — this phase must respect that split rather than pulling polygon data into a combined query.

### Integration Points
- Admin page reads/writes `regions` (hierarchy + `enabled`) and `region_polygons` (boundary geometry for enabled leaves) directly via PocketBase's built-in collection REST API — no new Go route needed for this phase's scope.
- The archive-generation cron (Phase 29, already shipped) reads `enabled = true` on `regions` on its own schedule — this phase only needs to persist the toggle; no signaling/webhook back to the cron is required.

</code_context>

<specifics>
## Specific Ideas

No specific visual/interaction references beyond "follow the `federation_ui.html` shell" and "match the web app's OpenFreeMap tile style" — both captured as decisions above.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope. (The search/filter box was considered for deferral but was decided in-scope; see D-08.)

### Reviewed Todos (not folded)
- `2026-07-24-comaps-poly-region-extraction-spike.md` — matched Phase 30 by keyword overlap (region/comaps/polygon/admin) but its `resolves_phase` is explicitly 29, and Phase 29's polygon-extraction work is already complete. Not folded — stale match, not applicable to this phase.
- `2026-07-18-way-types-and-surfaces-breakdown.md` — a mobile-first feature todo unrelated to the admin region picker. Not folded — different domain entirely.

</deferred>

---

*Phase: 30-admin-region-picker-ui*
*Context gathered: 2026-07-27*
