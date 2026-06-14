# Phase 2: Navigation Screen - Context

**Gathered:** 2026-06-12
**Status:** Ready for planning

<domain>
## Phase Boundary

Flutter NavigationScreen accessible at `/trail/:id/navigate` (go_router sub-route). The screen shows a full-screen flutter_map centered on the user's GPS position, displays the current Valhalla maneuver instruction at the top, provides a north-up/heading-up orientation toggle, and auto-advances through maneuvers as the user moves. Entry points are `TrailDetailScreen` (fixed bottom button) and `TrailDetailMapScreen` (floating button over elevation profile). The API call to `POST /api/v1/valhalla/navigate` is made on the _entry screen_ (with a loading spinner), not after navigation starts. Phase 3 adds the DraggableScrollableSheet stats layer.

</domain>

<decisions>
## Implementation Decisions

### Navigate Button Entry

- **D-01:** `TrailDetailScreen` — fixed full-width `ElevatedButton.icon` pinned at the bottom of the screen, positioned in front of (above) the scrollable content. Filled with primary color.
- **D-02:** `TrailDetailMapScreen` — large full-width `ElevatedButton.icon` floating over the elevation profile when it is open. Same style as D-01.
- **D-03:** Button label: "Navigate" with a navigation icon. Style: `ElevatedButton.icon`, filled primary color, full-width.
- **D-04:** go_router path: `/trail/:id/navigate` — sub-route of the trail detail path, not a top-level route. This keeps the trail ID in context and avoids nav bar interference.

### API Call Timing

- **D-05:** The `POST /api/v1/valhalla/navigate` call happens **on the trail detail screen** (before the route transition), not inside `NavigationScreen`. The user taps Navigate → button enters loading state → Dio call fires → on success, navigate to `/trail/:id/navigate` → on failure, show a toast and stay on the detail screen.
- **D-06:** While the API call is in flight, the Navigate button displays a `CircularProgressIndicator` in place of the icon (or inline). The rest of the screen content remains visible and interactive.
- **D-07:** On API failure: show a toast (using existing `toast_provider`) with a generic error message and cancel navigation. Do not change screen.
- **D-08:** `NavigationScreen` receives only the trail ID via the route parameter. It watches the existing `trailProvider(id)` via Riverpod to get trail data — consistent with `TrailDetailScreen` and `TrailDetailMapScreen`. The fetched `NavigateResponse` (maneuvers + shape) is passed as a go_router `extra` object to avoid a second API call inside the screen.

### Map Camera Follow

- **D-09:** Camera mode: **auto-follow with free-pan**. The map auto-centers on the user's GPS position by default. If the user pans away, auto-follow pauses and a recenter button appears (consistent with existing `MapScreen` recenter pattern). Tapping recenter resumes auto-follow.
- **D-10:** Zoom level: **inherit / user-adjustable**. Start at a sensible hiking zoom (~15–16) but respect pinch-zoom. Do not lock zoom.
- **D-11:** Orientation toggle: compass icon button in the top-right corner. Reuse the existing `MapCompass` widget pattern. Tapping cycles between north-up and heading-up. In heading-up mode, the map rotates to match the device bearing from the GPS stream.

### Maneuver Advancement

- **D-12:** Advancement detection: **distance threshold of ~30 m** to the `begin_shape_index` point of the _next_ maneuver. When the user's GPS position is within 30 m of `shape[nextManeuver.begin_shape_index]`, advance `currentManeuverIndex` by 1.
- **D-13:** GPS stream source: **`flutter_map_location_marker` stream** — the same `LocationMarkerDataStreamFactory` used by `CurrentLocationLayer`. Avoids a second Geolocator stream, reuses existing permission and accuracy setup.
- **D-14:** Trail end behavior: when `currentManeuverIndex` reaches the last maneuver, show a **completion banner** in the maneuver instruction area (e.g., "You've arrived!"). Navigation stays active (map + GPS) so the user can still see their position. The user exits manually via back/close button.
- **D-15:** Save-as-summit-log is **deferred** to a future iteration. The completion banner does not include this option in Phase 2.

### Breadcrumb Trace

- **D-18:** A red polyline is drawn on the map showing the actual path the user has traveled during the current navigation session (NAV-08). GPS positions are appended to an in-memory `List<LatLng>` as they arrive from the location stream, then rendered as a `PolylineLayer` on top of the trail polyline.
- **D-19:** The trace is session-only — it is not persisted. When navigation exits, the list is discarded. (Saving it as a GPX track is deferred to the summit-log feature in a future iteration.)
- **D-20:** Visual style: red/crimson color with a slightly thinner stroke than the trail polyline so the trail remains visually dominant.

### Screen Architecture

- **D-16:** `NavigationScreen` is a `ConsumerStatefulWidget` (needs `initState`/`dispose` for stream subscription and animation controller). Use `@riverpod` codegen for the navigate API call provider.
- **D-17:** The navigation provider (holding `NavigateResponse` + current maneuver index) lives in a `StateNotifierProvider` or `@riverpod` notifier — not inline widget state — so the maneuver index survives hot-reload and is testable.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Entry Screen Targets (where Navigate button is added)
- `app/lib/routes/trail_detail_screen.dart` — Add fixed bottom Navigate button (D-01)
- `app/lib/routes/trail_detail_map_screen.dart` — Add floating Navigate button over elevation profile (D-02)

### Map Pattern (NavigationScreen base)
- `app/lib/routes/map_screen.dart` — Full-screen flutter_map implementation; `AnimatedMapController`, `CurrentLocationLayer`, recenter pattern, compass toggle
- `app/lib/components/map/map_compass.dart` — Existing compass widget to reuse for orientation toggle (D-11)
- `app/lib/components/map/trail_layer.dart` — Trail polyline layer reused on NavigationScreen
- `app/lib/components/map/wanderer_map.dart` — Shared map widget; check if NavigationScreen should extend this or compose it

### State Management Patterns
- `app/lib/provider/trail_provider.dart` (or equivalent) — `trailProvider(id)` to fetch trail data inside screen (D-08)
- `app/lib/provider/api_provider.dart` — Existing Dio client for the navigate POST call (D-05)
- `app/lib/routes/trail_detail_screen.dart` — Example of how trail ID flows in via go_router and is watched via Riverpod

### Navigation / Routing
- `app/lib/main.dart` or router config file — go_router route definitions; add `/trail/:id/navigate` sub-route here (D-04)

### Toast Pattern
- `app/lib/provider/toast_provider.dart` (or equivalent) — Existing toast for API failure feedback (D-07)

### Phase 1 API Contract
- `web/src/lib/models/api/valhalla_navigate_schema.ts` — `NavigateResponse` type: `{ maneuvers: NavigateManeuver[], shape: [number, number][] }`
- Each `NavigateManeuver`: `{ instruction: string, length: number, begin_shape_index: number, bearing?: number }`
- Endpoint: `POST /api/v1/valhalla/navigate` with body `{ waypoints: [{lat, lon}], costing?: "pedestrian" | "bicycle" }`
- Waypoints come from trail's GPX track points (lat/lon only — no elevation needed)

</canonical_refs>

<code_context>
## Existing Code Insights

### What Already Exists (reuse, don't rebuild)
- Full-screen flutter_map stack in `map_screen.dart` with `AnimatedMapController`, `CurrentLocationLayer`, `MapCompass`, pinch-zoom, recenter button
- `flutter_map_location_marker` stream integration — permissions and accuracy config already handled
- Trail polyline rendering via `TrailLayer` — NavigationScreen can include this unchanged
- `trailProvider(id)` Riverpod provider for trail data
- Dio client and cookie jar wired in `api_provider.dart`

### Patterns to Follow
- `TrailDetailScreen` / `TrailDetailMapScreen` for: how trail ID arrives via go_router, how `trailProvider` is watched, how bottom buttons are laid out
- `MapScreen` for: `AnimatedMapController.animateTo()`, `CurrentLocationLayer` config, compass toggle state
- Other `ConsumerStatefulWidget` screens for: `@riverpod` async provider pattern, `AsyncValue.when()` error/loading/data rendering

### Integration Points
- New file: `app/lib/routes/navigation_screen.dart`
- New provider: `app/lib/provider/navigation_provider.dart` — holds `NavigateResponse` + `currentManeuverIndex`
- Modify: `app/lib/routes/trail_detail_screen.dart` — add Navigate button + API call trigger
- Modify: `app/lib/routes/trail_detail_map_screen.dart` — add Navigate button + API call trigger
- Modify: go_router config — add `/trail/:id/navigate` route

</code_context>

<specifics>
## Specific Implementation Notes

- **Waypoint source for API call:** The trail's GPX track points are already parsed on the detail screens (trail.gpx / trail.gpxData). Extract `[{lat, lon}]` from the parsed GPX track — same data source used by the elevation profile. Do not re-fetch the trail.
- **`NavigateResponse` as go_router extra:** Pass the pre-fetched response as `extra` on the GoRouter `go()` call so `NavigationScreen` doesn't need to call the API again. The screen reads `GoRouterState.extra as NavigateResponse` in `initState` / `build`.
- **30 m threshold:** This is a starting heuristic. The planner should make it a named constant (e.g., `_kManeuverAdvanceThresholdMeters = 30.0`) so it's easy to tune.
- **Costing derivation:** Flutter derives costing from the trail's category name before calling the API. A simple lowercase-contains check: `"bike" | "cycling" | "bicycle"` → `"bicycle"`, else `"pedestrian"`. This logic belongs in the detail screen before the Dio call.

</specifics>

<deferred>
## Deferred Ideas

- **Save completed trail as summit log** — user mentioned this at trail-end behavior; deferred to a future iteration of the navigation screen. Phase 2 completion banner is display-only.
- **Adaptive threshold for maneuver advancement** — speed-based or accuracy-weighted; 30 m static threshold for now.
- **Off-trail detection / re-routing** — out of scope per REQUIREMENTS.md.

</deferred>

---

*Phase: 2-Navigation Screen*
*Context gathered: 2026-06-12*
