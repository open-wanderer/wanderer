---
status: complete
phase: 30-admin-region-picker-ui
source: [30-01-SUMMARY.md, 30-02-SUMMARY.md]
started: 2026-07-28T07:37:01Z
updated: 2026-07-28T07:52:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Cold Start Smoke Test
expected: Stop any running PocketBase/wanderer-db instance. Start the server from scratch. It boots without errors, the regions migrations/seed complete, and navigating to /region-catalog/ returns the admin page (not a 404 or 500).
result: pass

### 2. Auth Gate (D-01)
expected: Visit /region-catalog/ while logged out (or with an expired superuser token). You see a "Log in to the admin dashboard" card linking to /_/ — NOT the region tree or the map. After logging in at /_/ and returning, the tree and map pane appear.
result: pass

### 3. Catalog Tree Renders (ADMINUI-01)
expected: The left pane shows the full CoMaps catalog (~1,306 regions) as a hierarchical tree of continents → countries → subregions. It loads in one go without pagination controls, and the page stays responsive while scrolling.
result: pass

### 4. Default Expansion + Expand/Collapse (D-07)
expected: On load, branches that already contain an enabled leaf are expanded; all other branches start collapsed. Clicking a group row's chevron expands it to reveal children; clicking again collapses it. The chevron icon flips direction (right → down) accordingly.
result: pass

### 5. Name Filter (D-08)
expected: Typing into the filter box narrows the tree to case-insensitive substring matches plus their ancestor chain, after a short (~120ms) pause. No network request fires while filtering (check the Network tab). Typing nonsense shows a "no matches" empty state. Clearing the box restores the full tree.
result: pass

### 6. Toggle a Leaf Region On (ADMINUI-02 + ADMINUI-03)
expected: Clicking a leaf region's toggle switch flips it on immediately. The change persists — reloading the page shows it still enabled. Its boundary polygon appears on the map right away, and the map does NOT re-zoom/re-fit. No errors appear in the browser console.
result: pass
note: "Closes the 30-VERIFICATION.md gap 'toggling a leaf adds/removes only that region's polygon layer immediately'. That report was written against a pre-08696b06 revision; flattenVisible now pushes the canonical tree node (out.push(n)), so path/bbox survive. VERIFICATION.md is stale, not the code."

### 7. Map Renders Enabled Regions on Load (ADMINUI-03)
expected: With at least one region enabled, reloading the page shows the map fitted to the union of all enabled regions' bounding boxes with a comfortable margin. Each enabled region draws as a translucent accent-colored fill with a 2px outline. With zero regions enabled, the map shows a Europe-wide view plus a "No regions enabled" overlay.
result: pass

### 8. Toggle a Leaf Region Off (ADMINUI-03, D-06)
expected: Clicking an enabled leaf's toggle flips it off, the change persists across reload, and that region's polygon disappears from the map immediately while all other enabled polygons stay put. The map does not re-fit.
result: pass
note: "Closes the second 30-VERIFICATION.md gap (map-side revert on PATCH failure) — same stale-report root cause as test 6; fixed in 08696b06."

### 9. Toggle Failure Reverts (ADMINUI-02)
expected: Simulate a failing PATCH (e.g. stop the server, or block /api/collections/regions/records/ in devtools) then click a toggle. The switch flips back to its original position, an inline error message appears on that row (no retry button), and the map returns to matching the reverted state.
result: pass

## Summary

total: 9
passed: 9
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

[none — all 9 tests passed]

Both gaps recorded in `30-VERIFICATION.md` (status `gaps_found`, written
2026-07-27T10:35Z) were verified closed during this session. That report was
produced against a pre-`08696b06` revision of `regions_ext/regions_ui.html`;
commit `08696b06` ("stop flattenVisible from stripping path/bbox off tree rows")
changed `flattenVisible` to push the canonical tree node (`out.push(n)`) rather
than a stripped `{id, name, kind, depth, enabled}` copy, so `path`/`bbox` now
reach `addPolygonForRow`/`removePolygonForRow` on both the success and
failure-revert branches of `toggleRegion`. Live testing confirms polygons add
and remove correctly on toggle. `30-VERIFICATION.md` is stale, not the code.
