# Quick Task 260718-e9j: Edit an existing route in the trail planner — Research

**Researched:** 2026-07-18
**Domain:** Flutter / go_router navigation, Riverpod route-planner state, GPX ↔ RouteAnchor reconciliation
**Confidence:** HIGH (all findings verified against the actual codebase)

## Summary

Everything needed already exists in the codebase; this is a wiring task, not new machinery. The pop-with-result navigation pattern (`context.push<T>()` awaited, `context.pop(result)`) is already used in **two** places we can copy verbatim. The parsed track is already sitting in `trail.expand.gpx` by the time `trail_create_screen` has the trail. Anchor seeding mirrors web's `initRouteAnchors` trivially against the Dart `gpx` package's `Gpx.trks.first.trksegs`. The route computed by the planner is turned back into a `Gpx` by the existing `finishPlanning` logic — we split that function so the elevation-merge half is reusable and the edit path returns the `Gpx` via `pop` instead of forward-pushing a draft.

**Primary recommendation:** Add an optional edit-mode to `RoutePlannerScreen` (seed anchors + "return via pop" flag), add a `seedFromTrack()` method to `RouteAnchors`, extract the elevation-merge half of `finishPlanning` into a reusable helper, and have `trail_create_screen` await `context.push<Gpx>('/route-planner', extra: {...})` then `copyWith` the returned track onto its in-memory `trail`.

---

## 1. Return-value navigation with go_router (VERIFIED)

The codebase already uses typed pop-with-result — no new pattern needed. Two live examples:

- `trail_create_screen.dart:102` — `final updated = await context.push<Waypoint>('/waypoint/create', extra: waypoint);` then `if (updated == null) return;`. The waypoint screen returns via `context.pop(waypoint)`.
- `route_planner_screen.dart:359` — `final result = await context.push<LocationSearchResult>('/location-search');`

**Pattern to apply:** In `trail_create_screen`'s new app-bar button handler:
```dart
final newGpx = await context.push<Gpx>('/route-planner', extra: {
  'mode': 'edit',
  'seedAnchors': anchorPoints,          // List<ml.Geographic>
  'travelProfile': 'pedestrian',        // no stored original profile — default
});
if (newGpx == null) return;             // user backed out
setState(() { trail = mergeRouteIntoTrail(trail, newGpx); });
```
On the planner side, `_onFinish` in **edit mode** calls `context.pop(finalGpx)` instead of `finishPlanning()`'s forward `push('/trail/create/edit')`.

**Router change (`router_provider.dart:250-268`):** the `/route-planner` builder already reads `state.extra as Map<String, dynamic>?`. Extend it to also read `mode`, `seedAnchors`, and pass them to `RoutePlannerScreen`. `push<Gpx>` on a top-level `GoRoute` returns and pops cleanly — no separate registration needed. The existing GPX-import forward-push flow (`finishPlanning` → `/trail/create/edit`) is untouched (per CONTEXT decision).

---

## 2. Seeding RouteAnchors from an existing Trail's track (VERIFIED)

**The track is already parsed and in memory.** By the time any trail reaches `trail_create_screen`:
- Server-loaded trails: `trail_provider.dart:34-49` fetches the GPX file and sets `expand.gpx` (parsed `Gpx`) + `expand.gpxData` (raw XML).
- Imported trails: `trail_import_util.dart:103` parses `gpxData` into `expand.gpx`.

So `trail_create_screen` already reads `trail.expand?.gpx` at line 209 and 570. **No re-parsing needed** — pass `trail.expand!.gpx!` derived points to the planner.

**Deriving segment-boundary anchors (mirror of web `initRouteAnchors`, `+page.svelte:553-577`):** web iterates `gpx.trk[0].trkseg`, adds an anchor at `trkpt[0]` of each segment, and for the *last* segment also adds its final `trkpt`. Dart equivalent using the `gpx` package (`import 'package:gpx/gpx.dart'`):
```dart
List<ml.Geographic> anchorsFromTrack(Gpx gpx) {
  final segs = gpx.trks.isNotEmpty ? gpx.trks.first.trksegs : const <Trkseg>[];
  final out = <ml.Geographic>[];
  for (var i = 0; i < segs.length; i++) {
    final pts = segs[i].trkpts;
    if (pts.isEmpty) continue;
    out.add(ml.Geographic(lat: pts.first.lat!, lon: pts.first.lon!));
    if (i == segs.length - 1) {
      out.add(ml.Geographic(lat: pts.last.lat!, lon: pts.last.lon!));
    }
  }
  return out;
}
```
Most recorded trails have a single `trkseg`, so this yields exactly `[start, end]` = 2 anchors → satisfies the Finish gate (`anchors.length >= 2`, `route_planner_screen.dart:436-439`). No reverse-geocoding at load (matches web + CONTEXT). Helpers `gpx.allPoints` / `gpx.allWaypoints` exist in `gpx_util.dart:97-110` if you need the full point list, but segment-boundary seeding should use `trksegs` directly.

**New provider method needed** (`route_anchor_provider.dart`): `resetForSession` (line 340) always resets to empty; there is no seed path. Add a `seedFromTrack(List<ml.Geographic> points, String profile, Map<String,dynamic>? opts)` that: cancels in-flight requests (like `resetForSession`), sets `anchors` from the points, builds straight `RouteSegment`s between consecutive anchors, clears undo/redo (fresh baseline), then calls `resolveAllSegments()` so auto-routing re-routes the boundaries via Valhalla (this is exactly web's behavior — anchors seed, segments re-route). Do **not** loop `appendAnchor()` — it pushes an undo snapshot per anchor and would let the user undo back through the seed.

---

## 3. Reconciling planner anchors back into the Trail on save (VERIFIED)

The planner already synthesizes a `Gpx` from its state: `plannedGpxProvider` (`planned_gpx_provider.dart`) walks the anchor→segment chain and returns a points-only `Gpx`. `finishPlanning` (`route_planner_handoff_util.dart:103-132`) then does a best-effort `POST /valhalla/height` elevation merge (`mergeHeightsIntoGpx`, silent-degrade on failure) producing `finalGpx`.

**Minimal correct approach — split, don't duplicate:**
1. Extract the first half of `finishPlanning` (read `plannedGpxProvider`, `buildNavShape`, `/valhalla/height` fetch, `mergeHeightsIntoGpx` with silent fallback) into `Future<Gpx> buildFinalPlannedGpx(WidgetRef ref)`. Import mode keeps calling `buildDraftTrail` + push; edit mode pops the returned `Gpx`.
2. Add a merge helper that preserves all non-track fields:
```dart
Trail mergeRouteIntoTrail(Trail existing, Gpx finalGpx) {
  final xml = GpxWriter().asString(finalGpx);
  final b = finalGpx.getBounds();
  return existing.copyWith(
    lat: b != null ? (b.latitudeNorth + b.latitudeSouth) / 2 : existing.lat,
    lon: b != null ? (b.longitudeEast + b.longitudeWest) / 2 : existing.lon,
    maxLat: b?.latitudeNorth ?? existing.maxLat,
    minLat: b?.latitudeSouth ?? existing.minLat,
    maxLon: b?.longitudeEast ?? existing.maxLon,
    minLon: b?.longitudeWest ?? existing.minLon,
    expand: (existing.expand ?? const TrailExpand()).copyWith(
      gpx: finalGpx,
      gpxData: xml,   // both set — Pitfall 1 in handoff_util: one without the other saves a trackless trail
    ),
  );
}
```
This mirrors `buildDraftTrail` (`route_planner_handoff_util.dart:63-84`) but as a `copyWith` onto the existing trail — title, description, photos, tags, waypoints, category, id, visibility all untouched. Server recomputes `distance`/`elevation_gain`/`duration` on save (they are derived; the local values are cosmetic until then).

**Critical (Pitfall 1, verified in handoff_util doc comment):** set **both** `expand.gpx` (map preview) **and** `expand.gpxData` (the raw XML that `toFormData()` actually uploads). Setting only one saves a trail whose track silently disappears.

---

## 4. Enabling condition for the new app-bar IconButton (VERIFIED)

The trail "has a recorded track" check already used elsewhere: `trail.expand?.gpx != null` — see `trail_create_screen.dart:570` (gates the elevation profile) and `:209`. There is **no** dedicated `hasTrack`/`isEmpty` getter on `Trail`; `expand?.gpx != null` is the established idiom. Gate the button on:
```dart
final hasTrack = trail.expand?.gpx != null &&
                 (trail.expand!.gpx!.trks.isNotEmpty);
```
Place the `IconButton` in the `AppBar.actions` list **before** the existing Save button (`trail_create_screen.dart:405-420`), matching the existing `IconButton.styleFrom(backgroundColor: surface)` treatment. Icon choice is discretion — `FontAwesomeIcons.route` or `FontAwesomeIcons.drawPolygon` fit the existing Font Awesome usage.

---

## 5. Pitfalls (VERIFIED against code + inline docs)

- **`routeAnchorsProvider` is a `@Riverpod(keepAlive: true)` singleton, not a family** (`route_anchor_provider.dart:74`). Its state survives across screen mounts and **must** be reset on every entry or it leaks the prior session's route. `RoutePlannerScreen.initState` (line 114-123) already does this via `addPostFrameCallback` → `resetForSession` (Riverpod forbids mutating a provider synchronously in `initState`; confirmed on-device per the code comment at line 102-111). **Your seed call must run in that same post-frame callback, after/instead of `resetForSession`**, and `_sessionReady` (line 111, 127-131) must flip to `true` only once seeding completes — this gating already prevents a stale/empty-anchor flash frame (T-t7q-03).
- **Two seed-vs-reset modes:** entering the planner fresh (import/new route) → `resetForSession` (empty). Entering to edit → `seedFromTrack`. Branch on the `mode` extra inside the post-frame callback. Never call both.
- **Travel profile is not stored on the Trail.** Editing an existing recorded track has no original profile to restore; default to `'pedestrian'`. Seeding then `resolveAllSegments()` re-routes the boundary anchors along pedestrian paths, so a re-routed edit may deviate slightly from the original recorded geometry between the two boundary anchors — this is inherent to the web reference behavior (anchors seed, Valhalla re-routes) and acceptable. Note for the planner as an accepted tradeoff, not a bug.
- **Segment `_added` / non-static discipline:** `RouteSegmentLayer` and the re-entrancy guards are instance fields specifically so re-mounting the planner works (`route_planner_screen.dart:61-100`). Don't introduce static state when adding edit-mode fields.
- **`context.mounted` guards:** after any `await` (the `push` and the elevation fetch) re-check `mounted` / `context.mounted` before using `context` — existing code does this consistently (e.g. line 360, handoff_util:130).

---

## Files to touch (planner reference)

| File | Change |
|------|--------|
| `app/lib/routes/trail_create_screen.dart` | New app-bar `IconButton` (gated on `expand?.gpx != null`); handler awaits `push<Gpx>('/route-planner', extra:{mode:edit,...})` and `copyWith`s result via `mergeRouteIntoTrail`. |
| `app/lib/routes/route_planner_screen.dart` | Accept optional edit-mode (seed points + return-via-pop flag); seed in the existing post-frame callback; `_onFinish` pops `finalGpx` in edit mode. |
| `app/lib/provider/route_anchor_provider.dart` | Add `seedFromTrack(points, profile, opts)` — sets anchors + straight segments, clears undo/redo, `resolveAllSegments()`. |
| `app/lib/util/route_planner_handoff_util.dart` | Extract `buildFinalPlannedGpx(ref)` (elevation-merge half); add `mergeRouteIntoTrail(existing, finalGpx)`. Leave `finishPlanning` import path intact. |
| `app/lib/provider/router_provider.dart` (~250-268) | `/route-planner` builder reads `mode`/`seedAnchors` from `extra`, passes to `RoutePlannerScreen`. |

## Assumptions Log

| # | Claim | Risk if Wrong |
|---|-------|---------------|
| A1 | Default travel profile `'pedestrian'` for edit mode (no stored original) | Cyclist trails re-route on foot paths; minor geometry deviation, user can switch profile in-session via Settings tab. |
| A2 | Server recomputes distance/elevation/duration on save so local stale values are cosmetic | If server trusts client-sent stats, edited trail shows stale metrics until re-fetch. Verify against `trail_save_provider`/backend. |

## Open Questions

1. **Seed the full recorded track vs. just boundary anchors?** CONTEXT locks "one anchor per segment boundary, no denser sampling" (mirrors web). Confirmed — do not sample interior points. The re-routed geometry between boundaries is the accepted result.

## Sources

- Codebase (HIGH): `trail_create_screen.dart`, `route_planner_screen.dart`, `route_anchor_provider.dart`, `route_anchor.dart`, `planned_gpx_provider.dart`, `route_planner_handoff_util.dart`, `router_provider.dart`, `trail.dart`, `trail_provider.dart`, `gpx_util.dart`, `web/src/routes/trail/edit/[id]/+page.svelte`
- `package:gpx` (Dart) — `Gpx.trks`/`Trk.trksegs`/`Trkseg.trkpts`/`Wpt`, `GpxWriter`, `getBounds` (used throughout existing code)
