---
phase: 260721-eob
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - web/src/lib/server/url.ts
  - web/src/lib/server/valhalla.ts
  - web/src/lib/models/api/valhalla_trace_route_schema.ts
  - web/src/routes/api/v1/valhalla/trace-route/+server.ts
  - app/lib/components/navigation/track_save_options_sheet.dart
  - app/lib/util/route_planner_handoff_util.dart
  - app/lib/routes/navigation_screen.dart
  - app/lib/i18n/app_en.arb
  - app/test/util/route_planner_handoff_util_test.dart
autonomous: true
requirements:
  - quick-260721-eob
must_haves:
  truths:
    - "Saving a recorded track first opens a bottom sheet with 'Recalculate heights' and 'Follow roads' toggles, both off by default."
    - "Enabling 'Follow roads' snaps the recorded path to the road network via Valhalla trace_route before the trail is handed to trail_create_screen."
    - "Enabling 'Recalculate heights' replaces recorded GPS elevations with Valhalla /height values computed on the final (possibly snapped) shape."
    - "Cancelling/dismissing the sheet aborts the save with no change to the session."
    - "A trace_route or /height failure (or a truncated snap) falls back silently to the pre-transformation track — no error toast, no blocked save."
    - "Both the exit-dialog Save and the completion-banner Save route through the sheet."
  artifacts:
    - path: "web/src/routes/api/v1/valhalla/trace-route/+server.ts"
      provides: "Authenticated POST proxy to Valhalla trace_route; decodes trip.legs[].shape and returns { shape } as {lat,lon} points"
      exports: ["POST"]
    - path: "web/src/lib/models/api/valhalla_trace_route_schema.ts"
      provides: "Zod TraceRouteRequestSchema (shape 2-500 bounded, costing enum)"
      contains: "TraceRouteRequestSchema"
    - path: "app/lib/components/navigation/track_save_options_sheet.dart"
      provides: "showTrackSaveOptionsSheet returning (bool recalcHeights, bool followRoads)?"
      contains: "showTrackSaveOptionsSheet"
    - path: "app/lib/util/route_planner_handoff_util.dart"
      provides: "snapShapeToRoads best-effort helper + snapResultAcceptable pure truncation guard"
      contains: "snapShapeToRoads"
  key_links:
    - from: "app/lib/routes/navigation_screen.dart"
      to: "app/lib/components/navigation/track_save_options_sheet.dart"
      via: "showTrackSaveOptionsSheet gates _saveRecordedTrack"
      pattern: "showTrackSaveOptionsSheet"
    - from: "app/lib/routes/navigation_screen.dart"
      to: "/valhalla/trace-route"
      via: "snapShapeToRoads → apiProvider.post"
      pattern: "trace-route"
    - from: "web/src/routes/api/v1/valhalla/trace-route/+server.ts"
      to: "VALHALLA_TRACE_ROUTE_URL"
      via: "getValhallaTraceRouteUrl env upstream"
      pattern: "getValhallaTraceRouteUrl"
    - from: "app/lib/routes/navigation_screen.dart"
      to: "/valhalla/height"
      via: "mergeHeightsIntoGpx on snapped shape"
      pattern: "mergeHeightsIntoGpx"
---

<objective>
Add a two-toggle bottom sheet ("Recalculate heights", "Follow roads", both off by default) that appears when a user saves a recorded track from the navigation/recording screen. "Follow roads" snaps the recorded breadcrumb to the road network via Valhalla `trace_route` through a NEW authenticated SvelteKit proxy route; "Recalculate heights" replaces recorded elevations with Valhalla `/height` values on the final shape. When both are on, snap runs first, then heights on the snapped shape. Both transforms are best-effort with silent fallback. After the transforms conclude, the (possibly transformed) trail hands off to `trail_create_screen` exactly as today.

Purpose: Give the user explicit, opt-in cleanup of a raw GPS recording (road-snapping + accurate elevation) at save time, without changing the default raw-preservation behavior.
Output: One new SvelteKit proxy route + env wiring; one new Flutter bottom sheet; snap pipeline helpers with a unit-tested truncation guard; rewired `_saveRecordedTrack`; new `app_en.arb` keys.
</objective>

<execution_context>
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/workflows/execute-plan.md
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@.planning/quick/260721-eob-add-a-bottom-modal-sheet-on-track-save-f/260721-eob-CONTEXT.md
@.planning/quick/260721-eob-add-a-bottom-modal-sheet-on-track-save-f/260721-eob-RESEARCH.md

# Backend clone source + wiring
@web/src/routes/api/v1/valhalla/navigate/+server.ts
@web/src/lib/server/valhalla.ts
@web/src/lib/server/url.ts
@web/src/lib/models/api/valhalla_navigate_schema.ts

# Flutter sheet pattern + reusable helpers + save flow
@app/lib/components/route_planner/travel_profile_sheet.dart
@app/lib/util/route_planner_handoff_util.dart
# navigation_screen.dart: _saveRecordedTrack lives ~line 689; call sites at ~1190 (exit dialog) and ~1352 (completion banner)
</context>

<tasks>

<task type="auto">
  <name>Task 1: New authenticated Valhalla trace-route SvelteKit proxy + env wiring</name>
  <files>web/src/lib/server/url.ts, web/src/lib/server/valhalla.ts, web/src/lib/models/api/valhalla_trace_route_schema.ts, web/src/routes/api/v1/valhalla/trace-route/+server.ts</files>
  <action>
Clone the `/valhalla/navigate` proxy into a new `trace-route` route that returns ONLY a decoded shape (no maneuvers), following the sibling convention.

1. In `web/src/lib/server/url.ts`, add `"VALHALLA_TRACE_ROUTE_URL"` to the `ExternalServiceUrlKey` union type (leave `resolveBaseUrl`/`normalizeBaseUrl` untouched — the `PUBLIC_`-fallback and https-normalization apply automatically).

2. In `web/src/lib/server/valhalla.ts`, add `getValhallaTraceRouteUrl(): string | null` that calls `getServiceUrl("VALHALLA_TRACE_ROUTE_URL")`, mirroring `getValhallaNavigateUrl`.

3. Create `web/src/lib/models/api/valhalla_trace_route_schema.ts` exporting a Zod `TraceRouteRequestSchema` — a copy of `NavigateRequestSchema`'s shape: `shape` = array of `{ lat: number.min(-90).max(90), lon: number.min(-180).max(180) }` with `.min(2, "at_least_two_shape_points").max(500)`, and `costing: z.enum(["pedestrian","bicycle"]).default("pedestrian")`. Also export the inferred `TraceRouteRequest` type. A dedicated schema (not a re-import of navigate's) keeps the two routes independently evolvable per RESEARCH.

4. Create `web/src/routes/api/v1/valhalla/trace-route/+server.ts` as a `POST(event: RequestEvent)` handler. Keep the authenticated pattern from navigate: `if (!event.locals.user) return error(401, "Unauthorized")` (this route is Flutter-app-only, same trust class as navigate). Inside a try/catch: read `getValhallaTraceRouteUrl()`, return `json({ message: "VALHALLA_TRACE_ROUTE_URL not set" }, { status: 400 })` when unset. Parse the body with `TraceRouteRequestSchema.parse(await event.request.json())`. Build the upstream body as `{ shape: body.shape, costing: body.costing, shape_match: "map_snap" }` — NO `directions_type` (map-matching cleanup, not turn-by-turn). `event.fetch` the upstream with `Content-Type: application/json`; on non-ok return `json({ message: "valhalla_error", detail }, { status: 502 })`. Read `data.trip`; if `!trip?.legs` return the same 502 shape. Build `shape: { lat: number; lon: number }[]` by iterating `trip.legs`, decoding each `leg.shape` with `decodePolyline` (from `$lib/util/polyline_util`, precision-6 default) — note `decodePolyline` yields `[lng, lat]`, so push `{ lat, lon: lng }` to keep coordinate order correct (Phase 14 transposed-coordinate hazard). Read ONLY `trip.legs` (partial-match remainder that Valhalla drops into `data.alternates` is intentionally ignored here; the Flutter truncation guard in Task 2 catches a shortened result). Return `json({ shape })`. On any thrown error `return handleError(e)` (from `$lib/util/api_util`). Add a `@swagger` JSDoc block modeled on navigate's, documenting the request (shape/costing) and the `{ shape }` response.

Do NOT touch any docker-compose file — `VALHALLA_TRACE_ROUTE_URL` being unset in the repo's compose is a pre-existing, out-of-scope discrepancy shared by the other three Valhalla actions (per CONTEXT).
  </action>
  <verify>
    <automated>grep -q 'VALHALLA_TRACE_ROUTE_URL' web/src/lib/server/url.ts && grep -q 'getValhallaTraceRouteUrl' web/src/lib/server/valhalla.ts && grep -q 'TraceRouteRequestSchema' web/src/lib/models/api/valhalla_trace_route_schema.ts && test -f 'web/src/routes/api/v1/valhalla/trace-route/+server.ts' && grep -q 'shape_match' 'web/src/routes/api/v1/valhalla/trace-route/+server.ts' && grep -q 'locals.user' 'web/src/routes/api/v1/valhalla/trace-route/+server.ts'</automated>
    <secondary>cd web && npm run check 2>&1 | tail -5 (svelte-check reports no NEW type errors in the four touched files)</secondary>
  </verify>
  <done>The new POST /api/v1/valhalla/trace-route route exists, is auth-gated, validates its body with a bounded Zod schema, forwards `{shape,costing,shape_match:"map_snap"}` to the env-configured upstream, and returns a `{ shape: {lat,lon}[] }` decoded from `trip.legs[].shape`. `VALHALLA_TRACE_ROUTE_URL` is registered in the url-key union with a matching getter. `npm run check` surfaces no new type errors.</done>
</task>

<task type="auto" tdd="true">
  <name>Task 2: Track-save options sheet + snap-then-heights pipeline wired into _saveRecordedTrack</name>
  <files>app/lib/components/navigation/track_save_options_sheet.dart, app/lib/util/route_planner_handoff_util.dart, app/lib/routes/navigation_screen.dart, app/lib/i18n/app_en.arb, app/test/util/route_planner_handoff_util_test.dart</files>
  <behavior>
    - snapResultAcceptable(original, snapped): returns false when `snapped` is empty; returns false when the snapped bounding-box diagonal is < 0.6x the original's diagonal (partial map-match truncation, valhalla#4802); returns true when diagonals are comparable (e.g. snapped within a few percent of original).
    - snapResultAcceptable(original, snapped): returns true for a snapped shape with a DIFFERENT point count but comparable bbox (trace_route returns Valhalla's own vertex density — count must NOT be the rejection signal).
  </behavior>
  <action>
Insert the opt-in transforms BEFORE the existing breadcrumb→handoff path in `_saveRecordedTrack`, reusing in-repo helpers only.

1. New sheet — `app/lib/components/navigation/track_save_options_sheet.dart`: a top-level `Future<(bool recalcHeights, bool followRoads)?> showTrackSaveOptionsSheet(BuildContext context)` modeled on `showTravelProfileSheet` (drag handle, `RoundedRectangleBorder` top `Radius.circular(20)`, flat bordered `Card`s `elevation:0` + `side: BorderSide(color: theme.colorScheme.outline)`, "Terrain Log" flat aesthetic). Unlike the travel-profile sheet (tap-to-close), this holds state: back the two bools with a `StatefulBuilder` (or a private `StatefulWidget`), each surfaced as a `SwitchListTile` inside its bordered card with a title + subtitle from the new l10n keys. BOTH default to `false`. A trailing confirm `FilledButton` pops `(recalcHeights, followRoads)`; dismiss/back pops `null`. Use `AppLocalizations.of(context)!` for all copy.

2. Pipeline helpers — add to `app/lib/util/route_planner_handoff_util.dart`:
   - `bool snapResultAcceptable(List<Map<String, double>> original, List<Map<String, double>> snapped)`: PURE. Returns false if `snapped.isEmpty`. Compute each shape's bbox diagonal as `sqrt(dLat^2 + dLon^2)` over its `lat`/`lon` extents; return false when `snappedDiag < 0.6 * originalDiag` (guards the trace_route truncation-into-alternates behavior so "Follow roads" can't silently shorten the saved trail). Point count is NOT a rejection criterion — Valhalla re-vertexes the path. Guard against a zero-length original (return true).
   - `Future<List<Map<String, double>>> snapShapeToRoads(WidgetRef ref, List<Map<String, double>> shape, String costing)`: best-effort. `POST '/valhalla/trace-route'` via `ref.read(apiProvider)` with `data: {'shape': shape, 'costing': costing}`; read `response.data['shape']` as a list of `{lat,lon}` maps → `List<Map<String,double>>`. Return the snapped shape ONLY when `snapResultAcceptable(shape, snapped)` is true; otherwise return the original `shape` unchanged. Wrap the whole call in try/catch and return the original `shape` on ANY error/timeout — mirror `buildFinalPlannedGpx`'s silent-fallback precedent (no toast, no rethrow).

3. Rewire `_saveRecordedTrack` in `app/lib/routes/navigation_screen.dart` (~line 689): show the sheet FIRST, before the `_savingTrack` guard, so both call sites (exit-dialog `_NavExitChoice.saveTrack` ~1190 and completion-banner button ~1352) inherit it with no change. `final options = await showTrackSaveOptionsSheet(context); if (options == null) return;` (cancel aborts save). Then set `_savingTrack` and run the pipeline inside the existing try/catch:
   - Read `navState` via `navigationProvider(widget.response, resumeManeuverIndex: _resumeManeuverIndex, resumeBreadcrumb: _resumeBreadcrumb)` using the EXACT same seed args as every other read in this file (split-brain provider hazard). Read `originalTrail = ref.read(trailProvider(widget.id)).value`.
   - Build the working shape from the breadcrumb: map `navState.breadcrumb` (lat/lon-bearing `Wpt`s) to `Geographic` and pass through `buildNavShape` (≤500 downsample) → `List<Map<String,double>>`.
   - If `options.followRoads` and the shape has ≥2 points: `final costing = costingForCategory(originalTrail?.expand?.category?.name)` (same derivation as `launchNavigation`/`trail_create_screen`, NOT hardcoded pedestrian), then `workingShape = await snapShapeToRoads(ref, workingShape, costing)`.
   - Build the `Gpx`: if `options.recalcHeights` and ≥2 points, best-effort `POST '/valhalla/height'` with `{'shape': workingShape}`, read `response.data['height']` as `List<num>`, then `gpx = mergeHeightsIntoGpx(workingShape, heights)` (on height failure, `heights = const []` → still merges the snapped coordinates with null ele). Else if `options.followRoads` (snapped, no heights): `gpx = mergeHeightsIntoGpx(workingShape, const [])`. Else (neither toggle): preserve today's exact behavior — `gpx = buildGpxFromPoints(navState.breadcrumb)` (keeps the raw recorded points' ele + time verbatim).
   - Keep the rest unchanged: `buildDraftTrail(ref, gpx, category: originalTrail?.categoryId)`, `active_nav.clear(_store)`, `pendingImportedTrail = trail`, `context.pushReplacement('/trail/create/edit', extra: trail)`, the existing error toast, and the `finally { _savingTrack = false }`.
   - NOTE the deliberate consequence: any transform path (snap and/or heights) yields a timeless track (the merge/handoff helpers are elevation-only, matching the planner handoff). The no-transform path alone preserves recorded timestamps. This follows the CONTEXT pipeline exactly; do not add ad-hoc time-preservation.

4. i18n — add keys to `app/lib/i18n/app_en.arb` ONLY (other locales sync via Crowdin): `save_recording_options` ("Save recording"), `recalculate_heights` ("Recalculate heights"), `recalculate_heights_description` ("Replace recorded GPS elevation with more accurate values from the map."), `follow_roads` ("Follow roads"), `follow_roads_description` ("Snap the recorded path to the nearest roads and trails."), and a confirm label — reuse an existing generic `save` key if one exists, else add `save` ("Save"). None of these values contain an apostrophe/interpolation, so the ICU single-quote trap does not apply; keep values double-quoted JSON. Run `flutter gen-l10n` so the getters exist before analyze.

5. Test — add a `group` to `app/test/util/route_planner_handoff_util_test.dart` exercising `snapResultAcceptable`: (a) empty snapped → false; (b) snapped bbox diagonal ~0.9x original → true; (c) snapped diagonal ~0.3x original → false; (d) snapped with far fewer points but comparable bbox → true. Pure fixtures, no network/WidgetRef needed.
  </action>
  <verify>
    <automated>cd app && flutter gen-l10n && flutter analyze lib/routes/navigation_screen.dart lib/components/navigation/track_save_options_sheet.dart lib/util/route_planner_handoff_util.dart && flutter test test/util/route_planner_handoff_util_test.dart</automated>
  </verify>
  <done>Saving a recorded track (from either the exit dialog or the completion banner) opens the two-toggle sheet with both toggles off; cancelling aborts. Follow-roads snaps via /valhalla/trace-route with the category-derived costing and silently falls back on failure or a truncated result; recalculate-heights merges /valhalla/height onto the final shape, silently falling back on failure. `flutter analyze` is clean on the three touched Dart files and the `snapResultAcceptable` unit tests pass.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Flutter client → SvelteKit `/valhalla/trace-route` | App-generated shape array crosses into the server proxy |
| SvelteKit proxy → Valhalla upstream | Server forwards a validated body to a fixed env-configured URL |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-eob-01 | Tampering/Information Disclosure | trace-route request body | mitigate | `TraceRouteRequestSchema.parse` bounds the array (2–500), clamps lat/lon ranges, and restricts costing to a two-value enum before any upstream call (ASVS V5) |
| T-eob-02 | Elevation of Privilege | trace-route route access | mitigate | `if (!event.locals.user) return error(401)` gate mirroring `/valhalla/navigate` (ASVS V4) |
| T-eob-03 | Information Disclosure (SSRF) | upstream URL selection | accept | Upstream comes only from server env (`VALHALLA_TRACE_ROUTE_URL` via `resolveBaseUrl`), never from the request body — identical trust model to the three existing Valhalla proxies (ASVS V10) |
| T-eob-04 | Tampering (data integrity) | snapped-shape result | mitigate | `snapResultAcceptable` bbox-diagonal guard rejects partial map-match truncation (valhalla#4802) so "Follow roads" cannot silently shorten the saved trail; silent fallback preserves the raw recording |
| T-eob-SC | Tampering (supply chain) | package installs | accept | No new npm/pub packages installed — all helpers are in-repo; no supply-chain surface added |
</threat_model>

<verification>
- Backend: `grep` gate confirms the env key, getter, schema, and auth-gated route with `shape_match` exist; `npm run check` (svelte-check) surfaces no new type errors in the four touched web files.
- Flutter: `flutter gen-l10n` succeeds (new ARB keys valid); `flutter analyze` clean on the three touched Dart files; `snapResultAcceptable` unit tests pass.
- Behavior (manual, on-device — not automatable here): trigger save from BOTH the exit dialog and the completion banner → sheet appears, both toggles off; confirm with Follow roads on → path snaps to roads; confirm with Recalculate heights on → elevations refreshed; cancel → no save; disable network → save still completes with the raw recording (silent fallback).
</verification>

<success_criteria>
- New authenticated `POST /api/v1/valhalla/trace-route` returns a `{lat,lon}` shape decoded from Valhalla `trace_route`, wired to `VALHALLA_TRACE_ROUTE_URL`.
- `_saveRecordedTrack` opens the two-toggle sheet before any transform; both toggles default off; cancel aborts.
- Follow-roads snaps first, then heights recalc on the snapped shape; both are best-effort with silent fallback (including truncation-guard fallback).
- No-transform save preserves today's raw-breadcrumb behavior byte-for-byte.
- New copy lives in `app_en.arb` only; automated gates (grep + svelte-check + flutter analyze + unit test) pass.
</success_criteria>

<output>
Create `.planning/quick/260721-eob-add-a-bottom-modal-sheet-on-track-save-f/260721-eob-SUMMARY.md` when done.
</output>
