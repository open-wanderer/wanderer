---
phase: 30-admin-region-picker-ui
verified: 2026-07-27T10:35:35Z
status: gaps_found
score: 6/8 must-haves verified
overrides_applied: 0
gaps:
  - truth: "Toggling a leaf adds/removes only that region's polygon layer immediately (ADMINUI-03, D-06)"
    status: failed
    reason: >
      flattenVisible() (db/routes/regions_ext/regions_ui.html:531-547) builds each
      visibleRows entry as a brand-new object literal containing only
      {id, name, kind, depth, enabled} (line 536). It does NOT copy `path` (or
      `bbox`) from the source tree node. The tree-row template's toggle button
      (line 953: `@click="toggleRegion(row)"`) passes this flattened object
      straight into toggleRegion, which forwards it unchanged to
      addPolygonForRow(row)/removePolygonForRow(row) (lines 686, 697). Those
      functions immediately dereference `row.path` (line 798:
      `row.path.replace(/'/g, "\\'")`), which is `undefined` on every row that
      ever reaches them. Reproduced by extracting the live <script> block and
      executing buildTree/flattenVisible/addPolygonForRow verbatim in Node:
      `addPolygonForRow` throws `TypeError: Cannot read properties of
      undefined (reading 'replace')` on every real toggle click. Because
      addPolygonForRow/removePolygonForRow are called without `await` or
      `.catch()` inside toggleRegion, this surfaces only as a silent unhandled
      promise rejection — the PATCH itself still fires (it uses `row.id`,
      which IS present), but the map's polygon layer is never added or
      removed in response to a toggle.
    artifacts:
      - path: "db/routes/regions_ext/regions_ui.html"
        issue: "flattenVisible() (line 536) omits `path`/`bbox` from the row objects handed to the tree-row template and, transitively, to toggleRegion/addPolygonForRow/removePolygonForRow (lines 683-703, 796-809)."
    missing:
      - "Add `path` (and `bbox`, needed nowhere else on this object but harmless to include) to the object literal pushed in flattenVisible() (regions_ui.html:536), OR resolve the canonical row by id from `this.regions`/`this.roots` inside toggleRegion/addPolygonForRow instead of trusting the flattened copy."
      - "Re-verify with a live browser click after the fix: toggling a leaf ON must show its polygon appear on the map without a console error, and toggling it OFF must remove that polygon."
  - truth: "On PATCH failure, the map polygon is removed/re-added to match the reverted toggle state (ADMINUI-02, D-06)"
    status: failed
    reason: >
      Same root cause as above — the revert branch of toggleRegion (line 697)
      calls addPolygonForRow/removePolygonForRow with the same path-less `row`
      object, so the map-side revert silently no-ops/throws exactly like the
      success path. The toggle control itself does revert correctly (row.enabled
      flips back, rowErrors is set — that part is independently verified), but
      the "map polygon is removed/re-added to match" clause of this must-have
      fails.
    artifacts:
      - path: "db/routes/regions_ext/regions_ui.html"
        issue: "toggleRegion's failure branch (line 697) inherits the same missing-`path` defect as the success branch."
    missing:
      - "Same fix as the gap above — once addPolygonForRow/removePolygonForRow can resolve a real path, both the success and failure branches of toggleRegion are fixed together."
deferred:
  - truth: "Live browser verification of tree render/expand/collapse/filter (30-01 <verify><human-check>) and toggle-persist/map-render/failure-revert/many-enabled-chunking (30-02 <verify><human-check>)"
    addressed_in: "Explicitly deferred to end-of-phase by both plans themselves (30-01-PLAN.md line 172, 30-02-PLAN.md line 121); not a later roadmap phase — flagged here as still outstanding, see Human Verification Required below."
    evidence: "Both plans' own <human-check> blocks and both SUMMARY.md 'Next Phase Readiness' sections explicitly state this check has not yet been run on a live instance."
---

# Phase 30: Admin Region Picker UI Verification Report

**Phase Goal:** A server owner manages the region catalog visually — toggling regions on a tree with a live coverage map — instead of hand-authoring a config file.
**Verified:** 2026-07-27T10:35:35Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | D-01: unauthenticated/expired token shows a log-in CTA, not the tree | ✓ VERIFIED | `regions_ui.html:887-897` — `x-show="!authenticated"` renders a `.login-card` with a "Log in to the admin dashboard" link to `/_/`; `main-split` (the tree+map) is gated behind `x-show="authenticated"` (line 900). |
| 2 | ADMINUI-01: full 1,306-row catalog loads via 2-page `perPage=1000` loop and renders as a collapsible tree from parent/path/depth | ✓ VERIFIED | `loadAllRegions()` loops `page`/`totalPages` with `perPage=1000` (regions_ui.html:656-667, no `perPage=1500` anywhere); `buildTree()` (499-517) attaches children by `parent` id; single non-recursive `x-for="row in visibleRows"` (line 933) renders the flattened tree; `go build ./...` and `go vet ./...` both pass clean. |
| 3 | D-07: branches with an already-enabled leaf auto-expand on load; others start collapsed | ✓ VERIFIED | `computeDefaultExpanded()` (519-529) walks depth-first, adding a group's id to the returned Set only when a leaf descendant has `enabled === true`; seeded into `this.expanded` in `loadRegions()` (line 646). Computed once per fresh load from server-fetched `enabled` values, so this is unaffected by the toggle bug below. |
| 4 | D-08: filter narrows to case-insensitive substring matches + ancestor chain, no extra API call | ✓ VERIFIED | `computeFilterMatches()` (549-568) is a pure client-side function over already-loaded `roots` — no `fetch`/`apiFetch` call inside it; 120ms debounce via `$watch('filterQuery', ...)` + `setTimeout` (608-614); zero-match empty state present (lines 924-930). |
| 5 | ADMINUI-02: clicking a leaf's toggle flips it optimistically and PATCHes `/api/collections/regions/records/{id}` with `{enabled: bool}` | ✓ VERIFIED | `toggleRegion()` (683-703): optimistic flip (line 685), `apiFetch(..., {method:'PATCH', body: JSON.stringify({enabled: row.enabled})})` (689-693) — uses `row.id`, which IS present on the flattened row object, so the PATCH reaches the correct record and persists correctly. Toggle is an accessible `role="switch"` with `aria-checked`/`aria-label`, keyboard-operable via Space/Enter (947-958). |
| 6 | ADMINUI-02: on PATCH failure the toggle reverts, the map polygon is removed/re-added to match, and an inline per-row error shows (no retry button) | ✗ **FAILED** (partial) | Toggle-state revert (line 696) and `rowErrors` inline message (698-701, rendered via `x-text` at line 962, no retry button) DO work. The **map-side revert does not** — see gap below; `addPolygonForRow`/`removePolygonForRow` are called with the same path-less row object and fail identically to the success path. |
| 7 | ADMINUI-03: live MapLibre map renders the boundary polygon of every currently-enabled leaf on load, fit to their bbox union (48px padding) with a Europe fallback | ✓ VERIFIED | `initMap()` (713-736) creates the map with style `tiles.openfreemap.org/styles/liberty`; on `load`, `enabledLeafRows()` (reading `this.regions`, which DOES carry `path`/`bbox` straight from the API) drives `loadEnabledPolygons()` → `addRegionLayer()` (fill 0.18 opacity + 2px line, both accent-colored) for each; `fitToEnabled()` unions bboxes and calls `map.fitBounds(..., {padding:48})`, falling back to `jumpTo({center:[10,45], zoom:3})` + the "No regions enabled" overlay when the enabled set is empty (777-791, 977-982). |
| 8 | ADMINUI-03/D-06: toggling a leaf adds/removes only that region's polygon layer immediately, without re-fitting the map | ✗ **FAILED** | Confirmed by direct reproduction (see Data-Flow Trace below): `addPolygonForRow`/`removePolygonForRow` throw/no-op on every toggle because the `row` object passed from the tree template lacks `path`. `toggleRegion` itself never calls `fitBounds` (verified by reading the function body — the "no re-fit" half of this truth holds), but the "adds/removes the polygon" half does not. |

**Score:** 6/8 truths verified (2 failed, same root cause)

### Deferred Items

Both plans explicitly deferred their live-browser human checks to "end of phase" (not to a later roadmap phase — there is no Phase 31+ item that covers this). Listed here per the instruction to call out deferred items rather than silently pass them; see **Human Verification Required** below for the concrete outstanding checks.

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | Live tree render/expand/collapse/filter + toggle-persist/map-render/failure-revert/many-enabled-chunking check | Not addressed elsewhere — still outstanding | 30-01-PLAN.md `<verify><human-check>` (line 172), 30-02-PLAN.md `<verify><human-check>` (line 121), both SUMMARY.md "Next Phase Readiness" sections |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `db/routes/regions_ui.go` | `//go:embed` handler `RegionsDashboard` + widened CSP + `RegionsExtFS` | ✓ VERIFIED | `func RegionsDashboard` (line 40) and `func RegionsExtFS` (line 22) both present; CSP includes `unpkg.com`, `worker-src blob:`, `tiles.openfreemap.org` (×2, connect-src + img-src); `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff` both set. |
| `db/routes/regions_ext/regions_ui.html` | `regionsApp()` Alpine SPA shell + tree/toggle/map | ✓ VERIFIED (exists, substantive, wired) / ⚠️ partially HOLLOW at Level 4 — see gaps | 989 lines. `function regionsApp()` present once; MapLibre 5.24.0 CDN pinned (css + js); no Tailwind reference; zero `x-html`, zero literal `Bearer`; all five phase-30 CSS classes present (`.tree-row`, `.toggle-switch`, `.map-pane`, `.row-error`, `.count-badge`). |
| `db/routes/regions_ext/main.js` | PocketBase `/_/` header-link injection | ✓ VERIFIED | `app.store.headerLinks.push({href:"/region-catalog/", icon:"ri-map-2-line", label:"Region Catalog"})`. |
| `db/main.go` | `GET /region-catalog/` route + `wanderer-region-catalog` UIExtension | ✓ VERIFIED | Line 224: bare `se.Router.GET("/region-catalog/", routes.RegionsDashboard)` — NOT nested under `regionsGroup` (line 239, still bound to `apis.RequireAuth()` for the unrelated pre-existing `/regions` API), NOT wrapped in `RequireAuth()`. Line 226-229: `UIExtension{Name: "wanderer-region-catalog", FS: routes.RegionsExtFS()}`. |
| `db/migrations/1785000000_create_regions_collection.go` | Source schema this page reads (read-only reference) | ✓ VERIFIED | `regions`/`region_polygons` collections carry no `ListRule`/`ViewRule`/`UpdateRule` — default nil rule is superuser-only, matching the page's own trust assumption (T-30-01); confirmed no later migration touches either collection's rules. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `db/routes/regions_ui.go` | `regions_ext/regions_ui.html` | `//go:embed regions_ext/regions_ui.html` | WIRED | Line 12; `go build ./...` succeeds (proves the embed target resolves). |
| `db/main.go` | `routes.RegionsDashboard` | `se.Router.GET("/region-catalog/", ...)` | WIRED | main.go:224. |
| `regions_ui.html` | `/api/collections/regions/records` | `apiFetch` 2-page `perPage=1000` loop | WIRED | Lines 656-667. |
| `regions_ui.html` | `/api/collections/regions/records/{id}` | `PATCH` in `toggleRegion` | WIRED (persistence only) | Lines 689-693 — record-level PATCH correctly targets `row.id`. |
| `regions_ui.html` | `/api/collections/region_polygons/records` | `loadEnabledPolygons` chunked OR-filter (initial load) | WIRED | Lines 743-758, driven by `this.regions` (has `path`) at initial `initMap()` time — line 725. |
| `regions_ui.html` | `/api/collections/region_polygons/records` | single-path fetch in `addPolygonForRow` (per-toggle) | ⚠️ **NOT WIRED — throws before the request is made** | Line 798 dereferences `row.path` before building the fetch; `row` (from `visibleRows`) never carries `path` — see Data-Flow Trace. |
| `regions_ui.html` | `https://tiles.openfreemap.org/styles/liberty` | `new maplibregl.Map({style: ...})` | WIRED | Line 718. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|---------------------|--------|
| Initial map render (`initMap`/`loadEnabledPolygons`/`fitToEnabled`) | `enabledLeafRows()` → `this.regions` | `loadAllRegions()` (real PocketBase fetch, `path`/`bbox` intact) | Yes | ✓ FLOWING |
| Per-toggle map sync (`addPolygonForRow`/`removePolygonForRow` called from `toggleRegion`) | `row` argument | `visibleRows` entry produced by `flattenVisible()` (regions_ui.html:536) — a **new object literal containing only `{id, name, kind, depth, enabled}`** | No — `row.path` is `undefined` | ✗ **DISCONNECTED** |
| Enabled-count badge (`enabledCount` getter, line 633-635) | `this.regions` | Same array `toggleRegion` never mutates | No — stale after every toggle | ✗ **DISCONNECTED** |
| Tree re-render on expand/collapse or filter after a toggle | `this.roots` tree nodes (never mutated by `toggleRegion`) | `flattenVisible(this.roots, ...)` re-run by `recomputeVisibleRows()` | No — reverts to pre-toggle `enabled` value | ✗ **DISCONNECTED** |

**Reproduction (executed, not just read):** the live `<script>` block was extracted verbatim from `db/routes/regions_ext/regions_ui.html` and run under Node to exercise the actual shipped functions:

```
$ node -e "... eval(extracted script) ... app.addPolygonForRow(visibleRows[1]) ..."
addPolygonForRow THREW: Cannot read properties of undefined (reading 'replace')

$ node -e "... simulate optimistic flip + enabledCount + expand/collapse cycle ..."
enabledCount before toggle: 0
enabledCount immediately after optimistic flip (should reflect toggle but does not): 0
leaf enabled state after an unrelated expand/collapse cycle: false (expected true, shows stale value from this.roots)
```

This confirms, by execution rather than inference, that: (1) every real toggle click throws inside `addPolygonForRow`, so the map never reflects the change; (2) the enabled-count badge never updates after a toggle; (3) expanding/collapsing any branch or typing in the filter box after a toggle silently reverts the toggled row's visual state to its pre-toggle value, even though the PocketBase record itself was correctly updated.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `go build ./...` (db module) | `go build ./...` (run from `db/`) | exit 0 | ✓ PASS |
| `go vet ./...` (db module) | `go vet ./...` | exit 0, no output | ✓ PASS |
| Live toggle → map polygon add (extracted-script reproduction) | see Data-Flow Trace above | `TypeError` thrown | ✗ FAIL |
| Live toggle → count-badge/tree-state consistency (extracted-script reproduction) | see Data-Flow Trace above | stale/reverted state | ✗ FAIL |
| Full server boot + browser interaction | N/A | not run | ? SKIP — requires a running PocketBase instance with the seeded catalog; this is exactly the scope of the deferred human check below. |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes exist for this phase and neither plan nor SUMMARY references any probe script. Step 7c: SKIPPED (no probes declared or found).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|--------------|--------|----------|
| ADMINUI-01 | 30-01-PLAN.md | Custom PocketBase admin page renders the region catalog as a collapsible tree | ✓ SATISFIED | Truths 1-4 above; route, embed, and tree pipeline all verified in code and via `go build`/`go vet`. |
| ADMINUI-02 | 30-02-PLAN.md | Admin toggles a leaf's `enabled` flag directly from the tree; persists with no other admin action | ⚠️ PARTIALLY SATISFIED | The PATCH-persistence mechanism works (truth 5, VERIFIED); the failure-path map-revert sub-requirement (truth 6) is FAILED — see gaps. |
| ADMINUI-03 | 30-02-PLAN.md | Live map renders boundary polygons of all currently-enabled leaves, visible before/after toggling | ⚠️ PARTIALLY SATISFIED | Initial-load rendering + fit-to-bounds (truth 7) VERIFIED; "before and after toggling" (truth 8, the roadmap's own SC #3 wording) FAILED — toggling never updates the map in practice. |

No orphaned requirements found — REQUIREMENTS.md maps only ADMINUI-01/02/03 to Phase 30, and both plans jointly declare exactly these three in their `requirements:` frontmatter.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `db/routes/regions_ext/regions_ui.html` | 536 | Data-flow break: flattened row object omits `path`/`bbox`, silently disconnecting toggle-time map sync from the map's data source | 🛑 Blocker | Root cause of the two FAILED truths above (ADMINUI-02 map-revert, ADMINUI-03 live toggle sync). |

No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` debt markers found anywhere in the phase's modified files (`regions_ui.go`, `regions_ui.html`, `main.js`). No `x-html` usage (XSS mitigation intact). No literal `Bearer` prefix (auth header format intact). No Tailwind reference (design-system constraint honored).

## Human Verification Required

The following checks were explicitly deferred by both plans to "end of phase" and have not been run on a live instance. They remain outstanding regardless of this report's findings — and given the confirmed data-flow bug above, at least items 2-4 below are **expected to currently fail** in a live browser until the `row.path` gap is fixed.

### 1. Tree render, expand/collapse, filter (30-01 deferred check)

**Test:** Log into `/_/` as superuser, open `/region-catalog/`.
**Expected:** The tree renders CoMaps groups that expand to child groups + leaf regions; branches with an already-enabled leaf start expanded; typing in the filter narrows to matches + ancestors.
**Why human:** Requires a running PocketBase instance with the seeded 1,306-row catalog and real browser DOM/Alpine rendering — not verifiable by static grep alone.

### 2. Map tiles + initial coverage render (30-02 deferred check)

**Test:** With the same session, observe the right pane.
**Expected:** OpenFreeMap tiles load (no blank/gray pane, no CSP "Refused to connect" console errors); every already-enabled leaf's boundary renders in accent fill+line and the map fits to their union (or shows the Europe fallback + "No regions enabled" overlay when none are enabled).
**Why human:** Requires live network/CSP behavior and visual confirmation of tile/polygon rendering.

### 3. Toggle → map sync (30-02 deferred check — will currently FAIL per this report's findings)

**Test:** Toggle a leaf ON, then OFF, watching both the map and the browser devtools console.
**Expected (per plan):** Toggling ON makes the polygon appear immediately and persists across reload; toggling OFF removes it.
**Actual per this verification:** The polygon will NOT appear/disappear (confirmed by code-level reproduction — `addPolygonForRow` throws `TypeError: Cannot read properties of undefined (reading 'replace')`); the PocketBase record itself will still be correctly updated (persists across reload), but the live map/UI will not reflect it until the page is reloaded.
**Why human (in addition to the static proof already gathered):** Confirms the exact devtools console error and visual non-update in a real browser, and confirms the record does persist correctly across reload despite the client-side desync.

### 4. Failure-path revert (30-02 deferred check)

**Test:** Simulate a PATCH failure (block the request or expire the token) and toggle a leaf.
**Expected:** Toggle reverts, polygon reverts, inline row error appears, no retry button.
**Actual per this verification:** Toggle-state revert and the inline error message will work (verified in code); the polygon-revert half will silently fail for the same reason as item 3.
**Why human:** Requires simulating a network/auth failure condition in a live browser session.

### 5. Many-enabled chunking (30-02 deferred check)

**Test:** Enable many regions and confirm all polygons still render (chunked OR-filter stays under PocketBase's 3500-char filter cap).
**Expected:** All polygons render without a truncated/malformed filter request.
**Why human:** Requires a live dataset with many enabled leaves and inspection of actual network request sizes; the chunk-sizing logic (line 748) is statically correct but its behavior at initial-load scale is easiest to confirm by observation.

## Gaps Summary

Both plans' individual grep-based acceptance criteria pass, and `go build`/`go vet` are clean — this is genuinely a well-executed phase at the "does the code exist, is it wired, does it compile" level. However, a Level-4 data-flow trace surfaced one concrete, reproducible bug that both plans' acceptance criteria were structurally unable to catch (they check for the presence of function names and API-call patterns via grep, not the shape of the objects flowing between them):

`flattenVisible()` (introduced in 30-01, before the toggle/map wiring existed) intentionally produces a minimal flattened row `{id, name, kind, depth, enabled}` for the tree's own rendering needs. Plan 30-02 then wired the toggle switch and the per-toggle map-sync (`addPolygonForRow`/`removePolygonForRow`) directly onto that same flattened row object — but never extended it to carry `path` (which those two functions require), and never established any mechanism for a toggle to write back into the canonical `this.regions` array or `this.roots` tree nodes that other parts of the app (the enabled-count badge, and any subsequent tree re-flatten triggered by expand/collapse or filtering) read from.

The net effect, confirmed by executing the actual shipped code in Node: clicking a leaf's toggle switch (1) correctly PATCHes the PocketBase record (persistence is real and durable across reload), but (2) throws inside `addPolygonForRow`/`removePolygonForRow` so the live map never adds or removes the polygon, (3) never updates the "N enabled" count badge, and (4) silently reverts the toggle's own visual state back to its pre-toggle value the next time the admin expands/collapses any branch or types in the filter box — creating a confusing, effectively broken live-editing experience even though the underlying data model is sound.

This must be fixed (either by carrying `path`/`bbox` through `flattenVisible()`, or by having `toggleRegion` resolve/mutate the canonical row by id) before the phase goal — "a server owner manages the region catalog visually... with a live coverage map" — can be considered achieved. The deferred end-of-phase human/browser check (outstanding regardless) would very likely have caught this on the very first toggle click.

---

*Verified: 2026-07-27T10:35:35Z*
*Verifier: Claude (gsd-verifier)*
