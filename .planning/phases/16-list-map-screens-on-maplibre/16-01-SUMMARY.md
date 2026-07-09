---
phase: 16-list-map-screens-on-maplibre
plan: 01
subsystem: ui
tags: [flutter, maplibre, maplibre-gl, riverpod, geobase, geojson]

# Dependency graph
requires:
  - phase: 15-maplibre-core-trail-rendering-offline-parity
    provides: "mapStyleJsonProvider, ml.MapController/onMapCreated hand-off pattern, live theme-swap-via-setStyle pattern (WandererMap)"
provides:
  - "SearchMap: trail-agnostic MapLibreMap host (style load + live theme swap only) — the shared base map_screen (16-03) also builds on"
  - "list_detail_map_screen.dart fully ported off flutter_map (interactive full list map, per-marker tap-to-fit)"
  - "list_detail_screen.dart's inline _ListMap fully ported off flutter_map (non-interactive, tap-through to full map)"
  - "CORE-08 requirement satisfied"
affects: [16-03-map-screen-clustering]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "SearchMap: thin ConsumerStatefulWidget host (no Trail param, no offline branch) for N-trail/0-trail map screens, distinct from WandererMap's single-trail host"
    - "ml.PolylineLayer(polylines: List<Feature<LineString>>) for multi-trail polyline rendering, built via ml.LineString.from(decoded points) — no manual GeoJSON source/layer"
    - "Imperative bounds-fit-on-load in onStyleLoaded (no declarative fit-to-bounds field exists in MapOptions)"
    - "ConsumerStatefulWidget required (not ConsumerWidget) wherever a caller must hold an ml.MapController across onMapCreated/onStyleLoaded"

key-files:
  created:
    - app/lib/components/base/search_map.dart
  modified:
    - app/lib/routes/list_detail_map_screen.dart
    - app/lib/routes/list_detail_screen.dart

key-decisions:
  - "SearchMap.layers is typed List<ml.Layer>?, not List<Widget>? as PLAN.md's artifact spec stated — ml.MapLibreMap.layers is a list of native style-layer builders (PolylineLayer, CircleLayer, etc.), a different type family from ml.MapLibreMap.children (Flutter widgets). Caught immediately by flutter analyze; fixed before commit."
  - "_ListMap converted from ConsumerWidget to ConsumerStatefulWidget so it can hold ml.MapController across onMapCreated/onStyleLoaded, mirroring list_detail_map_screen's approach — required because SearchMap's controller hand-off is inherently stateful."
  - "Added ml.MapScalebar()/ml.SourceAttribution() to list_detail_map_screen (the flutter_map version had neither) for ODbL attribution parity with every other MapLibre host in the app (WandererMap always shows them). Not added to the non-interactive inline _ListMap thumbnail, matching its minimal-chrome intent."

requirements-completed: [CORE-08]

# Metrics
duration: ~9min
completed: 2026-07-09
---

# Phase 16 Plan 01: List Map Screens on MapLibre Summary

**Ported list_detail_map_screen and list_detail_screen's inline map off flutter_map onto a new shared SearchMap MapLibre host, closing CORE-08.**

## Performance

- **Duration:** ~9 min
- **Started:** 2026-07-09T17:22:30Z
- **Completed:** 2026-07-09T17:30:56Z
- **Tasks:** 3
- **Files modified:** 3 (1 created, 2 modified)

## Accomplishments
- `SearchMap` — a lightweight, trail-agnostic MapLibreMap host (style load + live theme swap only) that both CORE-08 screens use and `map_screen` (16-03) will build on
- `list_detail_map_screen.dart` fully off `flutter_map`/`flutter_map_animations`/`vector_map_tiles`: renders every trail in a list as a polyline + tappable category-icon marker, camera fits all trails on load, tapping a marker fits that trail's own bounds
- `list_detail_screen.dart`'s inline `_ListMap` fully off `flutter_map`: non-interactive thumbnail showing the same trails, tap-through navigates to the full list map

## Task Commits

Each task was committed atomically:

1. **Task 1: Create the lightweight SearchMap host** - `7a71544d` (feat)
2. **Task 2: Port list_detail_map_screen to SearchMap** - `bbdb79b0` (feat)
3. **Task 3: Port list_detail_screen's inline _ListMap to SearchMap** - `c551509b` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified
- `app/lib/components/base/search_map.dart` - New `SearchMap` host: style-load + live theme swap only, exposes `layers`/`children`/`onStyleLoaded`/`onMapEvent`/`onMapCreated` seams, no `Trail` or offline surface
- `app/lib/routes/list_detail_map_screen.dart` - Full list map: `ml.PolylineLayer` for all trails, `ml.WidgetLayer` of tappable category-icon markers, imperative `fitBounds` on load and on marker tap/deselect
- `app/lib/routes/list_detail_screen.dart` - `_ListMap` converted to `ConsumerStatefulWidget`, `SearchMap(disabled: true)`, non-tappable markers, tap-through navigation via `ml.MapEventClick`

## Decisions Made
- `SearchMap.layers` typed `List<ml.Layer>?` (native style-layer builders), not `List<Widget>?` as PLAN.md's artifact spec literally stated — `ml.MapLibreMap.layers` and `.children` are different type families in the installed `maplibre` 0.3.5 package. Confirmed by reading `maplibre_platform_interface-0.3.5/lib/src/widget/map.dart` and `flutter analyze`'s immediate error.
- `_ListMap` converted `ConsumerWidget` → `ConsumerStatefulWidget` to hold `ml.MapController` across the map lifecycle callbacks — a structural requirement of the `SearchMap`/`ml.MapController` hand-off pattern established in Phase 15, not a design choice.
- Added `ml.MapScalebar()`/`ml.SourceAttribution()` to the full list map (matching every other MapLibre host in the app for ODbL attribution); deliberately omitted from the non-interactive inline `_ListMap` thumbnail to keep its minimal chrome.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed SearchMap.layers type mismatch (PLAN.md artifact spec was wrong)**
- **Found during:** Task 1 (Create the lightweight SearchMap host)
- **Issue:** PLAN.md's artifact contract specified `layers` (`List<Widget>?`), but `ml.MapLibreMap.layers` is typed `List<Layer>` (native style-layer builders like `PolylineLayer`), a completely different type from `ml.MapLibreMap.children`'s `List<Widget>`. Passing a `List<Widget>?`-typed field through to `layers:` fails to compile.
- **Fix:** Typed `SearchMap.layers` as `List<ml.Layer>?` instead, matching the installed package's actual `MapLibreMap` widget contract (confirmed by reading `maplibre_platform_interface-0.3.5/lib/src/widget/map.dart`).
- **Files modified:** app/lib/components/base/search_map.dart
- **Verification:** `flutter analyze lib/components/base/search_map.dart` — "No issues found!"
- **Committed in:** 7a71544d (Task 1 commit)

**2. [Rule 3 - Blocking] Removed a pre-existing unused import in list_detail_screen.dart**
- **Found during:** Task 3 (Port list_detail_screen's inline _ListMap)
- **Issue:** `flutter analyze` reported an `unused_import` warning for `package:wanderer/models/subcategory.dart` in `list_detail_screen.dart`. Confirmed via `git show HEAD~3` (pre-Task-1 state) that this import was already dead before this plan touched the file — an unrelated pre-existing issue.
- **Fix:** Removed the unused import. This task's acceptance criterion requires `flutter analyze` to print "No issues found!" for this exact file, and the file was already being modified in this task, so the fix was in-scope per the deviation rules' fix-attempt-limit guidance (directly blocks this task's own verification gate).
- **Files modified:** app/lib/routes/list_detail_screen.dart
- **Verification:** `flutter analyze lib/routes/list_detail_screen.dart` — "No issues found!"
- **Committed in:** c551509b (Task 3 commit)

---

**Total deviations:** 2 auto-fixed (1 bug/spec-mismatch, 1 blocking)
**Impact on plan:** Both fixes were required to satisfy each task's own `flutter analyze` verification gate. No scope creep — no other files were touched.

## Issues Encountered
- A full-project `flutter analyze` (unrelated to this plan's 3 files, all individually clean) surfaced 37 pre-existing issues elsewhere in the codebase — most notably an unused `subcategory.dart` import in `lib/routes/map_screen.dart` (in scope for **16-03**, not 16-01) and unrelated deprecated-icon-constant lints in `icon_util.dart`. Logged to `.planning/phases/16-list-map-screens-on-maplibre/deferred-items.md` per the Scope Boundary rule rather than fixed here.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `SearchMap` is ready for `map_screen.dart` (16-03) to build its cluster-layer host on top of — same style-load/theme-swap seam, no rework needed.
- `map_screen.dart:27`'s unused `subcategory.dart` import is 16-03's file to fix during its own port (see deferred-items.md).
- No blockers for 16-02/16-03.

---
*Phase: 16-list-map-screens-on-maplibre*
*Completed: 2026-07-09*

## Self-Check: PASSED

- FOUND: app/lib/components/base/search_map.dart
- FOUND: app/lib/routes/list_detail_map_screen.dart
- FOUND: app/lib/routes/list_detail_screen.dart
- FOUND commit: 7a71544d
- FOUND commit: bbdb79b0
- FOUND commit: c551509b
