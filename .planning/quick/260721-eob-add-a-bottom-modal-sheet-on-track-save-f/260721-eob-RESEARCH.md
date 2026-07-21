# Quick Task 260721-eob: Bottom modal sheet on track save (Recalculate heights / Follow roads) — Research

**Researched:** 2026-07-21
**Domain:** Flutter bottom-sheet UX + Valhalla `trace_route` map-matching + new SvelteKit proxy route
**Confidence:** HIGH (all patterns are in-repo; Valhalla `trace_route` shape verified against official docs)

## Summary

Everything this task needs already exists in the codebase as a directly reusable pattern. The Flutter side is a straight extension of the existing `_saveRecordedTrack` flow in `navigation_screen.dart`: insert a two-toggle modal sheet before the current breadcrumb→`buildDraftTrail` handoff, and conditionally run two best-effort Valhalla transforms (snap-then-heights) on the breadcrumb shape first. The new SvelteKit route is a near-clone of `/valhalla/navigate/+server.ts` — same auth gate, same Zod schema shape, same `decodePolyline(leg.shape)` decode — minus the maneuver extraction.

The one external unknown, the Valhalla `trace_route` request/response contract, is now **verified** against the official Map Matching API reference: `trace_route` takes `{shape:[{lat,lon,type?}], costing, shape_match}` and returns a route whose outputs are identical to the `/route` action (`trip.legs[].shape` = precision-6 encoded polyline), so the existing `decodePolyline` path applies unchanged.

**Primary recommendation:** Clone `/valhalla/navigate` into a new `trace-route` route returning only a decoded `shape` array; on the Flutter side, add `showTrackSaveOptionsSheet(...)` (modeled on `travel_profile_sheet.dart`) and a `snap+heights` best-effort pipeline in `_saveRecordedTrack` that reuses `buildNavShape` / `mergeHeightsIntoGpx` / `costingForCategory`, falling back silently to the raw breadcrumb on any failure.

## User Constraints (from CONTEXT.md)

### Locked Decisions
- **Toggle defaults:** Both "Recalculate heights" and "Follow roads" default to **off**. Raw GPS recording is preserved unless the user opts in.
- **Operation order (both on):** Snap roads **first**, then recalculate heights **on the snapped points**. Pipeline: breadcrumb → (if Follow roads) `trace_route` snap → (if Recalculate heights) `/valhalla/height` on the resulting shape → merge → `buildDraftTrail`.
- **Follow-roads costing:** Derive via existing `costingForCategory()` using the origin trail's category — not hardcoded `pedestrian`.
- **Failure handling:** Both the new `trace_route` call and the `/valhalla/height` call are **best-effort with silent fallback** — on error/timeout, proceed with the pre-transformation track. No blocking, no error toast. Matches `buildFinalPlannedGpx`'s existing precedent.

### Claude's Discretion
- Modal sheet visual layout/spacing — follow existing bottom-sheet patterns (`travel_profile_sheet.dart`, `waypoint_sheet.dart`) and the "Terrain Log" design (flat, bordered, IBM Plex Sans).
- New SvelteKit route path — sibling convention `web/src/routes/api/v1/valhalla/trace-route/+server.ts` (kebab `trace-route` recommended; see note below).
- New env var name — `VALHALLA_TRACE_ROUTE_URL`, added to `ExternalServiceUrlKey` (`url.ts`) + getter (`valhalla.ts`). None of the per-action vars are set in the repo's own compose files — pre-existing, out of scope.
- i18n strings — add new keys to `app_en.arb` only; other locales sync via Crowdin.
- Whether the new route requires `event.locals.user` auth — planner's call; leaning toward matching `/valhalla/navigate`'s authenticated pattern (Flutter-app-only, same as navigate).

### Deferred Ideas (OUT OF SCOPE)
None specified.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Toggle sheet UI + state | Flutter (client) | — | Pure presentation; returns a 2-bool result to caller |
| Snap-then-heights orchestration | Flutter (`_saveRecordedTrack`) | — | Sequences two API calls, owns best-effort fallback |
| `trace_route` upstream proxy | SvelteKit API route | Valhalla backend | Same trust boundary as `/valhalla/navigate`; decodes polyline server-side |
| Road snapping / map-matching | Valhalla | — | External routing engine; not hand-rolled |
| Elevation lookup | Valhalla (`/height`, existing) | — | Already proxied via `/valhalla/height` |

## Standard Stack

No new packages. Everything uses in-repo dependencies:

| Library | Version | Purpose | Notes |
|---------|---------|---------|-------|
| `dio` (via `apiProvider`) | 5.9.2 | Flutter → SvelteKit HTTP | `ref.read(apiProvider).post(...)` — same call style as `buildFinalPlannedGpx` |
| `zod` | 3.24.1 | Request validation on new route | Reuse the `NavigateRequestSchema` shape |
| `gpx` | 2.3.0 | Build/merge GPX | via `mergeHeightsIntoGpx` / `buildGpxFromPoints` |
| `maplibre` (`Geographic`) | 0.3.5 | Coordinate type | already used throughout |

**No `npm install` / `pub add` needed.** (Package Legitimacy Audit omitted — no external packages installed.)

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| (quick task) | Two-toggle save sheet | `travel_profile_sheet.dart` = the modal pattern; `_confirmExit` + `_buildCompletionBannerContent` = the two call sites |
| (quick task) | Follow roads via Valhalla `trace_route` + new proxy | Verified `trace_route` contract below; `/valhalla/navigate/+server.ts` = clone source |
| (quick task) | Recalculate heights | Existing `/valhalla/height` + `mergeHeightsIntoGpx` |
| (quick task) | Best-effort fallback | `buildFinalPlannedGpx` precedent |

## Valhalla `trace_route` — Verified Contract

**Request** (POST JSON) — `[CITED: valhalla.github.io/valhalla/api/map-matching/]`:
```jsonc
{
  "shape": [ {"lat": 39.98, "lon": -76.73}, ... ],   // array of {lat,lon}; optional "type":"break"|"via"
  "costing": "pedestrian",                             // any route costing except multimodal; use costingForCategory()
  "shape_match": "map_snap"                            // see table below
}
```

`shape_match` values (verified):
| Value | Meaning |
|-------|---------|
| `edge_walk` | Requires near-exact shape from a prior Valhalla route. **Wrong for raw GPS.** |
| `map_snap` | Full map-matching algorithm; use when input may not closely match edges. **Correct for recorded breadcrumbs.** More expensive. |
| `walk_or_snap` | **Default.** Tries edge-walk, falls back to map-matching. Also acceptable. |

**Recommendation:** Send `shape_match: "map_snap"` explicitly — this is exactly what `/valhalla/navigate/+server.ts` already hardcodes (line 96), and it's the correct mode for noisy on-foot GPS. Do NOT set per-point `type:"break"` — leave the raw `{lat,lon}` array; a single leg is returned by default. `[CITED: map-matching api-reference]`

Optional tuning fields (all under `trace_options`, all optional — only add if snapping proves too aggressive/loose in testing): `search_radius` (meters, **max 100**), `gps_accuracy` (meters), `breakage_distance`, `interpolation_distance`. `[CITED: map-matching api-reference]` Recommend shipping without them first (Valhalla defaults are tuned for GPS traces).

**Response** — outputs are **identical to the `/route` action** `[VERIFIED: official docs — "The outputs of the trace_route action are the same as the outputs of a route action"]`. That means `data.trip.legs[]`, each leg carrying `shape` = a **precision-6 encoded polyline** of the snapped path. This is byte-for-byte the same decode path `/valhalla/navigate` already uses:
```ts
for (const [lng, lat] of decodePolyline(leg.shape)) { shape.push([lat, lng]); }
```
`decodePolyline` (`web/src/lib/util/polyline_util.ts`) already defaults to precision 6.

### Failure modes to guard against (relevant to the silent-fallback requirement)

- **Discontinuities / unmatchable spans:** "If the path contains one or more discontinuities (i.e. no path can be found between two locations), it is split into multiple paths. Any remaining paths from the first discontinuity onwards are stored as route alternates on the response." `[CITED: map-matching api-reference]` → **Implication:** a recorded track with an indoor gap or a jump can yield a `trip` whose `legs` cover only the *first* matched span, with the rest dropped into `data.alternates`. If the proxy only reads `trip.legs`, a partial-match silently returns a **truncated** snapped track. **Guard:** the Flutter side must sanity-check the returned shape isn't drastically shorter than the input (e.g. point-count or bbox check) and fall back to the raw breadcrumb if so — otherwise "Follow roads" could silently shorten the saved trail. Confirmed as a real reported behavior (valhalla/valhalla#4802). `[CITED: github.com/valhalla/valhalla/issues/4802]`
- **Total match failure:** unmatchable input returns a non-2xx (Valhalla emits an error body, surfaced by the proxy as 502 like the siblings). The existing `try/catch → silent fallback` handles this.
- **Sparse/doubling-back on-foot tracks:** `map_snap` tolerates these far better than `edge_walk`, but retraced segments can still map-match onto the wrong parallel edge. This is inherent to map-matching and is exactly why the feature is **opt-in and off by default** per CONTEXT.md — no mitigation needed beyond the fallback.

## Architecture Patterns

### Pipeline flow (Flutter `_saveRecordedTrack`, revised)

```
[Save tapped]  (exit dialog _NavExitChoice.saveTrack  OR  completion banner button)
      │
      ▼
showTrackSaveOptionsSheet(context)  ──►  returns (recalcHeights: bool, followRoads: bool)  or null(cancel)
      │  null → abort save (do nothing)
      ▼
navState.breadcrumb  ──►  buildNavShape(points)   // ≤500 {lat,lon}, existing helper
      │
      ▼  if followRoads:
   POST /valhalla/trace-route {shape, costing: costingForCategory(originalTrail?.category?.name), shape_match:"map_snap"}
      │  success → decoded shape[] replaces working shape   (guard: reject if far shorter → keep raw)
      │  fail/timeout → keep raw shape  (silent)
      ▼  if recalcHeights:
   POST /valhalla/height {shape: workingShape}   // existing route
      │  success → mergeHeightsIntoGpx(workingShape, heights)
      │  fail/timeout → keep current gpx  (silent)
      ▼
   buildGpxFromPoints(...) or mergeHeightsIntoGpx(...) → Gpx
      │
      ▼
   buildDraftTrail(ref, gpx, category: originalTrail?.categoryId)  → Trail
      │
      ▼
   active_nav.clear(_store); pendingImportedTrail = trail; context.pushReplacement('/trail/create/edit', extra: trail)
```

**Key wiring note — shape vs. Gpx types:** `buildNavShape` returns `List<Map<String,double>>` (`{'lat','lon'}`). `trace_route`'s decoded response should be returned to Flutter in that **same** `{lat,lon}` shape (or `[lat,lon]` pairs mapped to it) so it drops straight into the existing `/valhalla/height` request and `mergeHeightsIntoGpx(shape, heights)` — both already consume that exact structure. If only `followRoads` is on (no heights), build the Gpx from the snapped shape via `mergeHeightsIntoGpx(snappedShape, const [])` (yields trkpts with null ele) or map to `Wpt`s and use `buildGpxFromPoints`. Prefer the former for one code path.

### New SvelteKit route (`web/src/routes/api/v1/valhalla/trace-route/+server.ts`)

Clone `/valhalla/navigate/+server.ts` with these deltas:
1. Auth: keep `if (!event.locals.user) return error(401, ...)` (recommended — matches navigate).
2. URL getter: `getValhallaTraceRouteUrl()` → reads `VALHALLA_TRACE_ROUTE_URL`.
3. Zod: reuse the `NavigateRequestSchema` shape (`shape` array min 2 / max 500, `costing` enum default `pedestrian`). A dedicated `TraceRouteRequestSchema` is warranted for clarity but can literally be a copy; Zod validation IS worth it here (vs. the pass-through `/route`/`/height` style) because the input is app-generated and bounded, matching navigate's precedent.
4. Upstream body: `{ shape: body.shape, costing: body.costing, shape_match: "map_snap" }` (no `directions_type`).
5. Response: decode `trip.legs[].shape` via `decodePolyline` exactly as navigate does, but return **only** `{ shape }` — no maneuver loop. Return each point as `[lat, lon]` (matching navigate's `shape` convention) or `{lat,lon}` — pick whichever the Flutter caller consumes; `{lat,lon}` is the more convenient one for the downstream `/height` + merge path.

Register `VALHALLA_TRACE_ROUTE_URL` in `ExternalServiceUrlKey` (`web/src/lib/server/url.ts` line 4 union) and add the getter to `web/src/lib/server/valhalla.ts`.

**Route path casing:** siblings are single-word (`route`, `height`, `navigate`), so there's no kebab-vs-underscore precedent to match. `trace-route` (kebab) is idiomatic for SvelteKit/URL segments; `trace_route` mirrors Valhalla's own action name. Either works — kebab recommended.

### Flutter toggle sheet — model on `travel_profile_sheet.dart`

`showTravelProfileSheet` (`app/lib/components/route_planner/travel_profile_sheet.dart`) is the closest pattern: a top-level `Future<T?> showXxxSheet(BuildContext)` wrapping `showModalBottomSheet`, with a drag handle, rounded top (`Radius.circular(20)`), bordered flat `Card`s (`elevation:0`, `side: BorderSide(color: theme.colorScheme.outline)`), returning a result via `Navigator.pop`.

**Difference:** the travel-profile sheet is tap-to-select-and-close (each card pops immediately). This sheet needs **two persistent `SwitchListTile`/toggle rows + a confirm button**, so it must be `StatefulWidget`-backed (or use a `StatefulBuilder` inside the builder) to hold the two bools, and pop `(recalcHeights, followRoads)` only on the confirm button. `waypoint_sheet.dart`'s `DraggableScrollableSheet` is the other in-repo bottom-sheet reference but is heavier than needed here — `showModalBottomSheet` (travel-profile style) is the right fit. Return type: a small record `(bool recalcHeights, bool followRoads)?` (null = cancelled).

### Both call sites (must both route through the sheet)

- `_confirmExit` → `.then((choice))` → `case _NavExitChoice.saveTrack: _saveRecordedTrack();` (line ~1190)
- `_buildCompletionBannerContent` → `FilledButton.icon(onPressed: _savingTrack ? null : _saveRecordedTrack, ...)` (line ~1352)

Cleanest wiring: show the sheet **at the top of `_saveRecordedTrack`** (before setting `_savingTrack`), so both call sites inherit it with no change. Abort early if the sheet returns null. Keep the `_savingTrack` double-tap guard.

## Don't Hand-Roll

| Problem | Don't build | Use instead |
|---------|-------------|-------------|
| Road snapping | Custom nearest-edge projection | Valhalla `trace_route` (`map_snap`) |
| Polyline decode | New decoder | `decodePolyline` (`polyline_util.ts`, precision 6) |
| Shape downsampling to ≤500 | New sampler | `buildNavShape` (`gpx_util.dart`) |
| Height→GPX merge | New zip logic | `mergeHeightsIntoGpx` (`route_planner_handoff_util.dart`) |
| Costing derivation | Hardcoded `pedestrian` | `costingForCategory` (`valhalla_util.dart`) |
| Best-effort height fetch | New try/catch | Mirror `buildFinalPlannedGpx` |

## Common Pitfalls

1. **Split-brain provider seed:** every `ref.read(navigationProvider(...))` in `navigation_screen.dart` MUST pass the identical `resumeManeuverIndex` / `resumeBreadcrumb` seed, or a different (empty) provider instance resolves. The new snap logic reads `navState.breadcrumb` — copy the exact seed args used at line ~694.
2. **Partial map-match truncation** (see Failure modes): guard the snapped result against dramatic shortening before accepting it; otherwise "Follow roads" can silently save a shorter trail. `[CITED: valhalla/valhalla#4802]`
3. **`buildNavShape` runs before snapping, not after:** snap consumes the ≤500-point downsampled shape; heights then run on the snapped shape (which may have a different point count than the input — `trace_route` returns Valhalla's own vertex density). `mergeHeightsIntoGpx` already zips by index against whatever shape you pass to `/height`, so always call `/height` with the **snapped** shape, never the original.
4. **Coordinate order:** Valhalla encoded polyline decodes to `[lng, lat]`; `decodePolyline` already returns `[lng, lat]` and navigate re-pushes as `[lat, lng]`. Keep the same convention so Flutter isn't handed transposed coordinates (a silent bug per this project's Phase 14 note).
5. **i18n single-quote trap:** new ARB values with interpolation must use double-quote literals (single quote is ICU's escape char) — established in quick-260720-s7m.

## Security Domain (security_enforcement: true)

| ASVS | Applies | Control |
|------|---------|---------|
| V5 Input Validation | yes | Zod `TraceRouteRequestSchema` (bounded array 2–500, lat/lon range, costing enum) — same as navigate |
| V4 Access Control | yes (if auth chosen) | `event.locals.user` gate mirroring `/valhalla/navigate` |
| V10 SSRF | yes | Upstream URL comes only from server env (`VALHALLA_TRACE_ROUTE_URL` via `resolveBaseUrl`), never from request body — same trust model as siblings; no user-controlled URL |

No new threat surface beyond the three existing Valhalla proxies. The route forwards a validated, bounded body to a fixed env-configured upstream.

## Assumptions Log

| # | Claim | Section | Risk if wrong |
|---|-------|---------|---------------|
| A1 | Returning snapped shape as `{lat,lon}` (not `[lat,lon]`) is the cleaner Flutter contract | New route / pipeline | Low — trivially adjustable; both feed the same merge helper |
| A2 | Valhalla defaults (no `trace_options`) give acceptable snapping for on-foot GPS without tuning | trace_route contract | Low — opt-in feature; can add `search_radius`/`gps_accuracy` later if testing shows over/under-snapping |
| A3 | `VALHALLA_TRACE_ROUTE_URL` upstream will be `.../trace_route` and is deployed by operators same as the other unset-in-repo vars | Discretion | Low — pre-existing pattern; not this task's job to wire compose |

## Sources

### Primary (HIGH)
- Valhalla Map Matching API reference (`docs/docs/api/map-matching.md`, master) — `trace_route` request/response, `shape_match` modes, discontinuity/alternates behavior, `trace_options`. Fetched via `gh api`.
- In-repo: `web/src/routes/api/v1/valhalla/{navigate,route,height}/+server.ts`, `web/src/lib/server/{valhalla,url,http}.ts`, `web/src/lib/models/api/valhalla_navigate_schema.ts`, `web/src/lib/util/polyline_util.ts`, `app/lib/util/{route_planner_handoff_util,gpx_util,valhalla_util,navigation_launch_util}.dart`, `app/lib/routes/navigation_screen.dart`, `app/lib/components/route_planner/travel_profile_sheet.dart`, `app/lib/components/trail/waypoint_sheet.dart`.

### Secondary (MEDIUM)
- valhalla/valhalla#4802 — map-matching returning part of route in alternates (partial-match truncation risk).

## Metadata

- Standard stack: HIGH — no new deps, all in-repo helpers verified by direct read.
- Architecture: HIGH — direct clone of an existing sibling route + existing pipeline extension.
- `trace_route` contract: HIGH — verified against official docs; only tuning-parameter behavior is MEDIUM (A2).
- Validation Architecture: omitted (`nyquist_validation: false`).
- Research date: 2026-07-21 · Valid until: ~2026-08-20 (stable; Valhalla map-matching API is long-stable).
