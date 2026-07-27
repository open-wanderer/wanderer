# Phase 30: Admin Region Picker UI - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-27
**Phase:** 30-admin-region-picker-ui
**Areas discussed:** Map style & polygon fetch, Tree loading & default expand state, Navigating 1,153 leaves, Toggle failure feedback

---

## Map style & polygon fetch

| Option | Description | Selected |
|--------|-------------|----------|
| OpenFreeMap public style URL | Point MapLibre GL (CDN) directly at https://tiles.openfreemap.org/styles/liberty — same tile family as the web app's default, zero extra files, no API key | ✓ |
| Copy ofm.json into db/routes/ | Duplicate web/static/styles/ofm.json as a static asset — full parity but a file that can drift | |
| Plain raster OSM tiles | tile.openstreetmap.org raster XYZ — simplest, but violates OSM tile usage policy | |

**User's choice:** OpenFreeMap public style URL

| Option | Description | Selected |
|--------|-------------|----------|
| Enabled leaves only, refetch per toggle | Fetch polygons only for currently-enabled leaves on load; refetch just that region on toggle | ✓ |
| Fetch all 1,153 leaf polygons upfront | One big fetch, filter client-side — payload likely hundreds of MB | |
| Lazy-fetch on group expand | Only fetch for expanded branches — risks hiding enabled regions in collapsed branches | |

**User's choice:** Enabled leaves only, refetch per toggle

| Option | Description | Selected |
|--------|-------------|----------|
| Optimistic — update map immediately | Polygon appears/disappears instantly on toggle click; revert on PATCH failure | ✓ |
| Wait for PATCH success | Map only updates after server confirms — safer, slower feedback | |

**User's choice:** Optimistic — update map immediately

| Option | Description | Selected |
|--------|-------------|----------|
| Auto-fit to all enabled regions | Fit map bounds to union of all enabled leaf polygons on load | ✓ |
| Static world view | Fixed default zoom/center, admin pans manually | |

**User's choice:** Auto-fit to all enabled regions

**Notes:** All four questions in this area landed on the recommended option.

---

## Tree loading & default expand state

| Option | Description | Selected |
|--------|-------------|----------|
| One full fetch, build tree client-side | Single PocketBase list call with perPage override past the 30/page default, build tree in JS | ✓ |
| Lazy-load children per group expand | Fetch only depth-0 groups initially, query children per expand | |

**User's choice:** One full fetch, build tree client-side

| Option | Description | Selected |
|--------|-------------|----------|
| Auto-expand branches with enabled leaves | Everything collapsed except paths down to already-enabled leaves | ✓ |
| Everything collapsed | Admin starts from a fully collapsed 153-group tree | |
| Everything expanded | Full 1,306-row tree open on load | |

**User's choice:** Auto-expand branches with enabled leaves

---

## Navigating 1,153 leaves

| Option | Description | Selected |
|--------|-------------|----------|
| Include a simple name-filter box | Client-side substring filter over the already-fetched 1,306 rows, no extra API call | ✓ |
| Defer to a later phase | Ship the tree without search for v1 | |

**User's choice:** Include a simple name-filter box

| Option | Description | Selected |
|--------|-------------|----------|
| Auto-expand ancestors of matches | Filtering narrows to matches + ancestor chain, auto-expanded | ✓ |
| Highlight matches, leave expand state manual | Matches highlighted but ancestors stay collapsed until manually expanded | |

**User's choice:** Auto-expand ancestors of matches

**Notes:** Search was raised as a possible scope-creep candidate (not in ADMINUI-01/02/03 as written) but the user opted to keep it in scope given the tree's size, rather than defer it.

---

## Toggle failure feedback

| Option | Description | Selected |
|--------|-------------|----------|
| Revert toggle + inline error near the row | Snap back + remove/re-add polygon + inline error message on that specific row | ✓ |
| Revert toggle + global toast | Same revert, page-level toast instead of inline | |
| Revert toggle, silent | No visible error message | |

**User's choice:** Revert toggle + inline error near the row

| Option | Description | Selected |
|--------|-------------|----------|
| Just re-click the toggle | No dedicated retry affordance — reverted toggle is already re-clickable | ✓ |
| Dedicated retry button in the error message | Explicit "Retry" action in the inline/toast error | |

**User's choice:** Just re-click the toggle

---

## Claude's Discretion

- Exact visual styling of the inline per-row error message (color, icon, dismiss behavior).
- Filter matching strategy (substring vs fuzzy) — substring chosen as sufficient.
- Filter input debounce timing, if any.

## Deferred Ideas

None — discussion stayed within phase scope. Search/filter was considered as a possible deferral candidate but was decided in-scope instead (see "Navigating 1,153 leaves" above).

Two todo matches were reviewed but not folded (stale/unrelated):
- `2026-07-24-comaps-poly-region-extraction-spike.md` — resolves Phase 29 (already complete), not Phase 30.
- `2026-07-18-way-types-and-surfaces-breakdown.md` — unrelated mobile feature.
