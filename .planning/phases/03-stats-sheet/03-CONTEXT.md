# Phase 3: Stats Sheet — Context

**Gathered:** 2026-06-13
**Status:** Ready for planning
**Source:** Conversation design session (replaces discuss-phase)

<domain>
## Phase Boundary

Add a persistent bottom sheet to `NavigationScreen` that shows live navigation stats and provides primary nav controls (Pause, Exit) and a secondary elevation profile view. The sheet is always visible and coexists with the full-screen map. No new screen or route is introduced.

</domain>

<decisions>
## Implementation Decisions

### Bottom Sheet Structure
- Use `DraggableScrollableSheet` anchored at the bottom of `NavigationScreen`'s outer `Stack`
- Sheet has two snap points: collapsed (compact) and expanded (full stats)
- A `PageView` inside the sheet provides two pages: **Stats page** and **Elevation Profile page**
- `PageView` uses `NeverScrollableScrollPhysics` — page switching is button-controlled only (avoids conflict with the `DraggableScrollableSheet`'s vertical drag)
- A `PageController` field on `_NavigationScreenState` drives page transitions via `animateToPage`

### Compact Stats Row (collapsed state, always visible)
Three stats shown when sheet is collapsed:
- **Time elapsed** (stopwatch from navigation start, `HH:MM:SS` format)
- **Distance** (metres/km covered since navigation start)
- **Elevation gain** (cumulative gain in metres since navigation start)

### Expanded Stats (drag sheet up)
When dragged to expanded snap point, three additional stats appear below the compact row:
- **Elevation loss** (cumulative descent in metres)
- **Current speed** (GPS speed, km/h)
- **Average speed** (total distance / elapsed time, km/h)

### Button Row (below stats, always visible in compact state)
Three items in a horizontal row:
- **Left — Elevation profile button**: `IconButton`, taps → `_pageController.animateToPage(1)` to switch to elevation profile page
- **Center — Pause/Resume button**: large `FilledButton`, toggles navigation pause (pauses the stats timer and GPS-advance logic)
- **Right — Exit button**: `IconButton` (replaces the current top-left X button on `NavigationScreen`), calls `context.pop()`

### Current top-left exit button
Remove the existing `SafeArea > Align > Padding > _buildExitButton` overlay from `NavigationScreen`'s outer Stack — exit is consolidated into the sheet's button row.

### Elevation Profile Page (PageView page 1)
- Renders the trail's GPX elevation chart — same chart widget used in `trail_detail_map_screen.dart`
- A back arrow or close button in the top-left of this page calls `_pageController.animateToPage(0)` to return to stats
- The map remains fully interactive behind the sheet while on this page
- Sheet can still be dragged up/down while on the elevation profile page

### Stats Computation (provider-side)
- All stat values computed in a new Riverpod provider `navigationStatsProvider` (family keyed on `NavigateResponse` like `navigationProvider`)
- Provider exposes a `NavigationStats` frozen state class:
  - `elapsed`: `Duration` (wall clock from first GPS fix)
  - `distanceMeters`: `double` (cumulative Haversine distance)
  - `elevationGainMeters`: `double` (cumulative positive altitude delta)
  - `elevationLossMeters`: `double` (cumulative negative altitude delta, stored as positive)
  - `currentSpeedKmh`: `double` (from `Position.speed`, converted)
  - `averageSpeedKmh`: `double` (distanceMeters / elapsed.inSeconds * 3.6)
- Stats update on each GPS position event (same stream as `navigationProvider`)
- Timer for `elapsed` uses a `Ticker` via `TickerProviderStateMixin` already on `_NavigationScreenState`, or a `Stream.periodic` in the provider

### Pause / Resume
- Pausing freezes `elapsed` and stops `distanceMeters`/elevation accumulation
- GPS dot and map follow continue while paused (camera not affected)
- `isPaused` bool lives in `NavigationStats` state

### Format Utilities
- Reuse or extend `app/lib/util/format_util.dart` for distance (m/km) and speed (km/h) formatting
- Time formatted as `HH:MM:SS` or `MM:SS` when < 1 hour

### No page indicators / no swipe
- No dot indicators for the PageView pages (clean minimal UI)
- Horizontal swipe is disabled (`NeverScrollableScrollPhysics`) — only button-driven

### Keep existing map controls
- The compass + recenter `IconButton` column (bottom-right, inside `FlutterMap.children`) stays as-is from Phase 2 fixes
- The maneuver banner at the top stays as-is

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Navigation Screen (Phase 2 output — primary anchor)
- `app/lib/routes/navigation_screen.dart` — current screen layout; sheet and exit button changes go here
- `app/lib/provider/navigation_provider.dart` — existing `NavigationState`/`Navigation` notifier pattern to follow for stats provider

### Elevation Profile Reference
- `app/lib/routes/trail_detail_map_screen.dart` — contains the elevation chart widget to reuse

### Format Utilities
- `app/lib/util/format_util.dart` — existing formatting helpers; extend for stats display

### Riverpod Codegen Pattern
- `app/lib/provider/navigation_provider.dart` — `@riverpod` + `@freezed` pattern to replicate for `navigationStatsProvider`
- `app/lib/models/navigate_response.dart` — freezed model pattern

### GPX / GPS Data
- `app/lib/util/gpx_util.dart` — GPX parsing helpers
- `app/lib/models/navigate_response.dart` — `NavigateResponse` (trail shape, maneuvers)

</canonical_refs>

<specifics>
## Specific Ideas

- Reference UI: AllTrails navigation screenshot — Time / Distance / Elev. gain compact row, large centered Pause button, secondary actions on each side
- The elevation profile page is the same chart already built in `trail_detail_map_screen.dart` — no new chart implementation needed
- Pause center button should be visually dominant (large `FilledButton` or `ElevatedButton` with icon)
- Stats numbers should use a large, readable font weight (like `TextStyle(fontSize: 24, fontWeight: FontWeight.bold)`)
- Labels ("Time", "Distance", "Elev. gain") in a smaller subdued style above each value

</specifics>

<deferred>
## Deferred Ideas

- Animated transitions between stat values (e.g., counting up) — v2
- Audio/haptic cues on pause/resume — v2
- Background GPS tracking (app backgrounded during navigation) — separate phase
- Offline map tiles — separate phase

</deferred>

---

*Phase: 03-stats-sheet*
*Context gathered: 2026-06-13 via conversation design session*
