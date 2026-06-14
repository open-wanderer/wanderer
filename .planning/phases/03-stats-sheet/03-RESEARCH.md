# Phase 3: Stats Sheet - Research

**Researched:** 2026-06-13
**Domain:** Flutter UI (DraggableScrollableSheet + PageView), Riverpod codegen state, GPS-derived live statistics
**Confidence:** HIGH

## Summary

Phase 3 adds a persistent `DraggableScrollableSheet` to the existing `NavigationScreen` (Phase 2 output). The sheet hosts a button-controlled `PageView` with two pages: a live-stats page (time/distance/elevation/speed) and an elevation-profile page that reuses the already-built `ElevationProfile` widget. All numeric stats are computed in a new `navigationStatsProvider` Riverpod notifier, family-keyed on `NavigateResponse` exactly like the existing `navigationProvider`. The existing top-left exit button is removed and consolidated into the sheet's button row.

The single most important architectural fact: **the navigation screen already runs one broadcast GPS stream** (`_positionStream = Geolocator.getPositionStream().asBroadcastStream()` at `navigation_screen.dart:52`) that feeds both `CurrentLocationLayer` and `navigationProvider.onPosition()`. D-13 (locked) forbids a second GPS stream. Therefore the stats provider MUST be fed by a method call from the existing GPS listener (the `navigationProvider` pattern), NOT a self-subscribing `StreamNotifier`. This is the dominant constraint on the provider design.

The second important fact: the `ElevationProfile` component (`app/lib/components/trail/elevation_profile.dart`) is already a fully reusable `StatefulWidget` taking `Trail` + `Gpx`, and `NavigationScreen` already watches `trailProvider(widget.id)` which loads `trail.expand.gpx`. **No new chart code is needed** — page 1 of the PageView is essentially the existing `ElevationProfile(trail: trail, gpx: trail.expand!.gpx!)`.

**Primary recommendation:** Build `navigationStatsProvider` as an `@riverpod class` notifier (family on `NavigateResponse`) with an `onPosition(Position pos)` method called from the existing GPS listener; drive the elapsed clock with a separate 1-second `Timer.periodic` (or `Stream.periodic`) inside the notifier rather than coupling elapsed time to GPS fixes. Use `DraggableScrollableSheet` with `snap: true` and two `snapSizes`, a `PageView` with `NeverScrollableScrollPhysics`, and reuse `ElevationProfile` verbatim for page 1.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Live stat computation (distance, elevation gain/loss, speed) | Mobile Providers (`navigationStatsProvider`) | — | Pure derived state from GPS; belongs in a Riverpod notifier, testable via `ProviderContainer` like `navigationProvider` |
| Elapsed-time clock | Mobile Providers (timer inside notifier) | Screen state | Wall-clock independent of GPS cadence; lives with the stats it annotates |
| GPS acquisition | Screen state (`_positionStream`) | — | Single broadcast stream already owned by `_NavigationScreenState` (D-13); stats provider must NOT open its own |
| Sheet layout, PageView, buttons | Mobile Frontend (`NavigationScreen`) | — | Pure widget composition; `PageController` is screen-local UI state |
| Elevation chart rendering | Mobile Frontend (`ElevationProfile` reuse) | Models (`Gpx`) | Already-built component; consumes parsed `Gpx` from `trailProvider` |
| Distance/speed/time string formatting | Models/Utils (`format_util.dart`) | — | Stateless formatting helpers, extend existing file |
| Pause/Resume | Mobile Providers (`isPaused` in state) + timer control | Screen (button) | Freezing accumulation is a state concern owned by the notifier |

## Standard Stack

### Core (all already in `app/pubspec.yaml` — no new packages)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `flutter_riverpod` | 3.3.1 | Stats provider state mgmt | Project standard; `navigationProvider` already uses it `[VERIFIED: pubspec.yaml]` |
| `riverpod_annotation` / `riverpod_generator` | 4.0.3 | `@riverpod` codegen | Existing `navigation_provider.dart` pattern `[VERIFIED: codebase]` |
| `geolocator` | 13.0.2 | `Position.speed`, `Position.altitude` | Already the GPS source in Phase 2 `[VERIFIED: pubspec.yaml]` |
| `latlong2` | 0.9.1 | `Distance()` Haversine | Used in `navigation_provider.dart:66` and `gpx_util.dart:46` `[VERIFIED: codebase]` |
| `fl_chart` | 1.2.0 | Elevation chart (transitive via `ElevationProfile`) | Already powers `ElevationProfile` `[VERIFIED: pubspec.yaml]` |
| `gpx` | 2.3.0 | Parsed `Gpx` for chart | Loaded by `trailProvider` `[VERIFIED: codebase]` |
| `duration` | 4.0.3 | `Duration.pretty()` formatting (used by ElevationProfile) | Already a dependency `[VERIFIED: pubspec.yaml]` |
| `font_awesome_flutter` | 11.0.0 | Stat/button icons | Project icon standard `[VERIFIED: codebase]` |

**Installation:** None required. Phase 3 adds **zero** new packages. This is a pure composition + new-provider phase.

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Notifier + `onPosition()` method | `StreamNotifier` self-subscribing to `Geolocator.getPositionStream()` | **REJECTED** — violates D-13 (single GPS stream). Would open a second native location subscription → battery drain + position divergence |
| `Timer.periodic` for elapsed clock | `Ticker` via `TickerProviderStateMixin` | Ticker fires every frame (~60fps) — wasteful for a 1Hz clock and ties timing to a widget vsync. A 1-second `Timer.periodic` or `Stream.periodic` in the notifier is cleaner and provider-owned |
| `Timer.periodic` for elapsed clock | Recompute `DateTime.now() - startTime` on each GPS fix | Elapsed would only update when GPS emits (could stall if user stands still); a dedicated tick keeps the clock smooth |

## Package Legitimacy Audit

> Not applicable — Phase 3 installs **no external packages**. All libraries used are already present in `app/pubspec.yaml` and verified in the codebase. No slopcheck run required.

| Package | Registry | Disposition |
|---------|----------|-------------|
| (none added) | — | N/A |

## Phase Requirements

> **Note:** REQUIREMENTS.md STATS-01..05 describe an older "4 horizontal swipe pages" design. The 03-CONTEXT.md design (locked via conversation) **overrides** this: a single stats page with collapsed/expanded states + a separate elevation-profile page, button-driven. The requirement *intent* (show distance, elevation, speed) is fully satisfied; the page partitioning differs. The planner should map to the CONTEXT design and note the override.

| ID | Original Description | CONTEXT Design Realization | Research Support |
|----|---------------------|----------------------------|------------------|
| STATS-01 | Bottom DraggableScrollableSheet with paginated stats | `DraggableScrollableSheet` + button-controlled `PageView` (2 pages). Vertical drag = collapse/expand; pages = stats vs elevation. Horizontal swipe disabled per CONTEXT. | Sheet pattern verified in `trail_detail_map_screen.dart:209`; gesture-conflict analysis below |
| STATS-02 | Distance remaining to trail end | CONTEXT compact row shows **distance covered** (cumulative Haversine). "Remaining" can be derived as `totalRouteLength - distanceMeters` if planner wants both. | `Distance()` Haversine pattern from `navigation_provider.dart:66` |
| STATS-03 | Distance covered + ETA | Compact row: **Distance** + **Time elapsed**. ETA not in CONTEXT (deferred). | `Position` accumulation; elapsed timer pattern below |
| STATS-04 | Cumulative elevation gain + loss | Compact: **Elevation gain**; Expanded: **Elevation loss**. Computed from `Position.altitude` deltas with smoothing. | Altitude noise + smoothing analysis below; mirrors `gpx_util.dart:57-64` gain/loss logic |
| STATS-05 | Current GPS speed + average speed | Expanded: **Current speed** (`Position.speed` × 3.6) + **Average speed** (`distance/elapsed × 3.6`). | `Position.speed` is m/s `[VERIFIED: pub.dev geolocator]` |

## Architecture Patterns

### System Architecture Diagram

```
                    ┌─────────────────────────────────────────────┐
                    │  Geolocator.getPositionStream()              │
                    │  .asBroadcastStream()   (navigation_screen)  │  ← SINGLE stream (D-13)
                    └───────────────┬─────────────────────────────┘
                                    │ Position (lat,lon,altitude,speed)
                  ┌─────────────────┼──────────────────────────────┐
                  ▼                 ▼                               ▼
        CurrentLocationLayer   navigationProvider              navigationStatsProvider  ◄── NEW
         (GPS dot + follow)     .onPosition(LatLng)             .onPosition(Position)
                                 (maneuver advance,             │  accumulate distance,
                                  breadcrumb)                   │  elevation gain/loss,
                                                                │  current speed
                                                                │
                          ┌──── Timer.periodic(1s) ────────────►│ tick elapsed (if !isPaused)
                          │                                     │
                          │                          NavigationStats (frozen state)
                          │                          { elapsed, distanceMeters,
                          │                            elevationGain/Loss,
                          │                            current/avgSpeedKmh, isPaused }
                          │                                     │
                          │                          ref.watch in NavigationScreen
                          │                                     │
                          ▼                                     ▼
                   ┌──────────────────────────────────────────────────────────┐
                   │  DraggableScrollableSheet (outer Stack, bottom)            │
                   │  ┌────────────────────────────────────────────────────┐   │
                   │  │ PageView (NeverScrollableScrollPhysics)             │   │
                   │  │  page 0: Stats  | page 1: ElevationProfile(reused) │   │
                   │  └────────────────────────────────────────────────────┘   │
                   │  Button row: [Elev profile] [Pause/Resume] [Exit]          │
                   └──────────────────────────────────────────────────────────┘
```

### Recommended Structure (files touched / created)
```
app/lib/
├── provider/
│   └── navigation_stats_provider.dart   # NEW — @riverpod class + NavigationStats freezed state
├── routes/
│   └── navigation_screen.dart           # EDIT — add sheet, PageController, remove top-left exit, wire onPosition→stats
├── util/
│   └── format_util.dart                 # EDIT — add formatSpeed(), formatElapsed()
└── components/trail/
    └── elevation_profile.dart           # REUSE AS-IS — no change
```

### Pattern 1: Stats notifier fed by method call (NOT self-subscribing)
**What:** Mirror `navigation_provider.dart`. The screen's existing GPS listener calls `ref.read(navigationStatsProvider(response).notifier).onPosition(pos)`.
**When to use:** Always here — D-13 forbids a second GPS stream.
**Example:**
```dart
// Source: pattern mirrors app/lib/provider/navigation_provider.dart (codebase) [VERIFIED: codebase]
@riverpod
class NavigationStats extends _$NavigationStats {
  final _distance = const Distance();
  Timer? _ticker;
  DateTime? _start;
  double? _lastAltitude;
  LatLng? _lastPoint;

  @override
  NavigationStatsState build(NavigateResponse response) {
    // ref.onDispose to cancel the ticker when the family entry is disposed
    ref.onDispose(() => _ticker?.cancel());
    return const NavigationStatsState();
  }

  void onPosition(Position pos) {
    if (state.isPaused) return;
    _start ??= DateTime.now();
    _ticker ??= Timer.periodic(const Duration(seconds: 1), (_) => _tick());

    final here = LatLng(pos.latitude, pos.longitude);
    double dist = state.distanceMeters;
    if (_lastPoint != null) {
      dist += _distance.as(LengthUnit.Meter, _lastPoint!, here);
    }
    _lastPoint = here;

    double gain = state.elevationGainMeters;
    double loss = state.elevationLossMeters;
    if (_lastAltitude != null) {
      final d = pos.altitude - _lastAltitude!;
      if (d.abs() >= _kAltitudeNoiseFloorM) {      // smoothing threshold
        if (d > 0) gain += d; else loss += -d;
        _lastAltitude = pos.altitude;
      }
    } else {
      _lastAltitude = pos.altitude;
    }

    state = state.copyWith(
      distanceMeters: dist,
      elevationGainMeters: gain,
      elevationLossMeters: loss,
      currentSpeedKmh: (pos.speed.isNaN || pos.speed < 0 ? 0 : pos.speed) * 3.6,
    );
  }

  void _tick() {
    if (state.isPaused || _start == null) return;
    final elapsed = DateTime.now().difference(_start!) - _pausedAccum;
    final avg = elapsed.inSeconds > 0
        ? state.distanceMeters / elapsed.inSeconds * 3.6 : 0.0;
    state = state.copyWith(elapsed: elapsed, averageSpeedKmh: avg);
  }

  void togglePause() { /* freeze: record pause start, accumulate _pausedAccum on resume */ }
}
```
**Note on pause:** simplest robust approach is to track `_pausedAccum` (total paused Duration) and subtract it from wall-clock elapsed. On pause, `onPosition` early-returns so distance/elevation stop accumulating; `_lastPoint`/`_lastAltitude` should be refreshed on resume to avoid a jump. The `currentSpeedKmh` should be forced to 0 while paused.

### Pattern 2: DraggableScrollableSheet + button-controlled PageView
**What:** Sheet provides vertical drag (collapse/expand); inner `PageView` with `NeverScrollableScrollPhysics` switches stats↔elevation only via `_pageController.animateToPage`.
**When to use:** This phase — CONTEXT locks horizontal swipe OFF.
**Example:**
```dart
// Source: sheet shape mirrors trail_detail_map_screen.dart:209-232 [VERIFIED: codebase]
DraggableScrollableSheet(
  controller: _sheetController,          // DraggableScrollableController field
  initialChildSize: 0.18,                // compact
  minChildSize: 0.18,
  maxChildSize: 0.45,                    // expanded
  snap: true,
  snapSizes: const [0.18, 0.45],
  builder: (context, scrollController) {
    return Container(
      decoration: /* canvasColor + top radius + shadow, like trail_detail_map_screen */,
      // IMPORTANT: the builder's scrollController must be attached to the
      // sheet's primary scrollable so drag-to-expand works. With a PageView
      // (which is horizontal) plus fixed-height content, wrap content in a
      // single ListView/SingleChildScrollView(controller: scrollController)
      // OR give the sheet a non-scrolling body and rely on drag-handle area.
      child: ListView(
        controller: scrollController,      // REQUIRED for drag gesture handoff
        physics: const ClampingScrollPhysics(),
        children: [ /* drag handle, PageView (fixed height), button row */ ],
      ),
    );
  },
)
```

### Anti-Patterns to Avoid
- **Second GPS stream in the stats provider** — violates D-13; opens a duplicate native location subscription. Feed via `onPosition()` from the existing listener instead.
- **Ticking elapsed time off GPS fixes** — clock stalls when the user stands still (no new `Position`). Use an independent 1s timer.
- **Forgetting `scrollController` on the sheet body** — `DraggableScrollableSheet` only drags when its builder's `scrollController` is wired into a scrollable; otherwise drag-to-expand silently fails.
- **`setState`-driven stats in the screen** — stats are derived state and belong in the provider (testability, D-17 pattern), not in `_NavigationScreenState`.
- **Calling `notifier.onPosition` during build** — call it from the existing GPS `listen` callback in `initState`, never inside `build()`.
- **Computing elevation gain on raw altitude deltas** — GPS altitude is noisy; every fix would add fake gain+loss. Apply a noise-floor threshold (see Pitfall 2).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Elevation chart on the profile page | A new fl_chart widget | `ElevationProfile(trail:, gpx:)` from `components/trail/elevation_profile.dart` | Already built, smoothed, gradient-colored, touch-scrubbable; CONTEXT explicitly says reuse it `[VERIFIED: codebase]` |
| Haversine distance | Manual `sin/cos` math | `const Distance().as(LengthUnit.Meter, a, b)` (latlong2) | Already the project pattern (`navigation_provider.dart:66`, `gpx_util.dart:46`) `[VERIFIED: codebase]` |
| Loading parsed GPX for the chart | New API/parse call | `ref.watch(trailProvider(widget.id))` already loads `trail.expand.gpx` | `trail_provider.dart:31-45` parses GPX into `expand.gpx`; screen already watches it `[VERIFIED: codebase]` |
| GPX gain/loss totals (if needed for chart header) | New loop | `gpx.getTotals()` extension (`gpx_util.dart:38`) | Returns `GpxStats` with gain/loss/distance `[VERIFIED: codebase]` |
| Bottom sheet shape/shadow | Custom container styling from scratch | Copy the decoration block from `trail_detail_map_screen.dart:218-231` | Consistent look, already tuned `[VERIFIED: codebase]` |

**Key insight:** This phase is ~90% composition of existing components. The only genuinely new logic is the stats accumulation provider and two formatter functions.

## Common Pitfalls

### Pitfall 1: Gesture conflict — vertical sheet drag vs horizontal PageView
**What goes wrong:** A scrollable inside `DraggableScrollableSheet` can fight the sheet's drag-to-expand. A horizontally-scrolling `PageView` can also race vertical drags.
**Why it happens:** Flutter's gesture arena hands a drag to the nearest competing recognizer.
**How to avoid:** CONTEXT already mitigates this by setting `PageView(physics: NeverScrollableScrollPhysics())` — the PageView claims **no** drag gestures, so all vertical drags reach the sheet and horizontal swipes are ignored. Additionally, wire the sheet builder's `scrollController` to the outer scrollable so vertical drag-to-expand is handled by the sheet, not an inner widget. **Confidence: HIGH** (NeverScrollableScrollPhysics removing the PageView from the gesture arena is documented Flutter behavior `[CITED: api.flutter.dev PageView/ScrollPhysics]`).
**Warning signs:** Sheet won't expand when dragging over the PageView area; or page changes happen on vertical drag.

### Pitfall 2: GPS altitude noise inflates elevation gain/loss
**What goes wrong:** Consumer GPS altitude is accurate only to roughly ±10–20 m and jitters several meters between consecutive fixes even when stationary. Summing raw positive deltas as "gain" produces large fake gain (and loss) on flat ground.
**Why it happens:** `Position.altitude` is GPS-derived MSL altitude with no barometric fusion guarantee; noise is per-fix `[CITED: pub.dev geolocator; Baseflow/flutter-geolocator#1412 reports altitude jitter/0-values]`.
**How to avoid:** Apply a **noise-floor threshold** — only accumulate an altitude delta when `|Δalt| >= ~2–3 m` (and update the reference altitude only when the threshold is crossed, so small drifts don't accumulate). Optionally smooth altitude with a short moving average before differencing (the `ElevationProfile` already uses a windowed smoother of size 30 for its chart — `elevation_profile.dart:577`). For live nav, the threshold approach is simpler and adequate. **The exact threshold is `[ASSUMED]`** — 2–3 m is a common starting value but should be a tunable named constant the planner can adjust after field testing.
**Warning signs:** Elevation gain climbs steadily while standing still or walking on flat ground.

### Pitfall 3: `Position.speed` units and invalid values
**What goes wrong:** `Position.speed` is in **meters/second**, not km/h, and can be `0`, negative, or `NaN`/`-1` on platforms/devices that can't compute it.
**Why it happens:** Speed is platform-derived; some devices return a sentinel when unavailable `[CITED: Baseflow/flutter-geolocator#993 "Problem getting user speed"]`.
**How to avoid:** Convert with `× 3.6` for km/h; guard `if (speed.isNaN || speed < 0) speed = 0`. Prefer computing **average** speed from accumulated distance / elapsed (more stable than instantaneous).
**Warning signs:** Negative or absurd current-speed readings; `NaN` rendered in UI.

### Pitfall 4: Sheet absorbs touches meant for the map when collapsed
**What goes wrong:** A `DraggableScrollableSheet` in the outer `Stack` over `FlutterMap` only occupies its own height; the map area above it stays interactive. But if you wrap the sheet region in something full-height (e.g. a `Positioned.fill` or a backdrop), it will swallow map gestures.
**Why it happens:** The sheet is a bottom-anchored child sized to `childSize × screenHeight`; only that bottom band intercepts touches.
**How to avoid:** Place the sheet as a direct child of the existing `Stack` (like the maneuver banner is now), NOT inside a full-screen wrapper. Keep `minChildSize` small (~0.18) so the collapsed sheet covers only the bottom strip, leaving most of the map tappable. The map's existing controls column (`bottom: 24, right: 8`) lives inside `FlutterMap.children` — ensure the collapsed sheet height doesn't cover it, or the recenter/compass buttons become unreachable. **Confidence: HIGH** (matches the existing `trail_detail_map_screen.dart` pattern where the sheet coexists with `WandererMap`).
**Warning signs:** Map pan/zoom dead in the bottom region even when sheet is collapsed; compass/recenter buttons covered.

### Pitfall 5: Family entry disposal must cancel the timer
**What goes wrong:** A `Timer.periodic` started in the notifier keeps firing after screen exit → `setState`/state-after-dispose and battery drain.
**Why it happens:** The notifier is family-keyed on `NavigateResponse`; on screen exit the family entry is disposed but a raw `Timer` outlives it unless cancelled.
**How to avoid:** Register `ref.onDispose(() => _ticker?.cancel())` in `build()`. Mirror the Phase 2 discipline in `navigation_screen.dart:78-86` (cancel subscription, close controllers).
**Warning signs:** Logs show stats ticking after pop; "setState after dispose" style errors.

### Pitfall 6: Pause must not lose the distance/altitude reference
**What goes wrong:** On resume after pause, the first GPS fix is far from `_lastPoint` (user may have moved while paused, or just GPS drift), adding a large false distance/elevation jump.
**Why it happens:** `_lastPoint`/`_lastAltitude` were captured before the pause window.
**How to avoid:** On **resume**, reset `_lastPoint`/`_lastAltitude` to the first post-resume fix *before* accumulating, so the paused interval contributes nothing. Track total paused duration (`_pausedAccum`) and subtract from wall-clock elapsed. CONTEXT confirms GPS dot + map follow continue during pause (camera unaffected) — only accumulation freezes.
**Warning signs:** Distance/elevation jumps the instant Resume is tapped.

## Code Examples

### Extend format_util.dart
```dart
// Source: extends existing app/lib/util/format_util.dart [VERIFIED: codebase]
// formatDistance(meters) and formatElevation(meters) ALREADY EXIST — reuse them.
// formatDistance already does the m→km threshold at 1000 m (line 7). Reuse for "Distance".

String formatSpeed(double? kmh, {String unit = 'metric'}) {
  if (kmh == null || kmh.isNaN || kmh < 0) return "-";
  if (unit == "metric") return "${kmh.toStringAsFixed(1)} km/h";
  return "${(kmh * 0.621371).toStringAsFixed(1)} mph";
}

String formatElapsed(Duration d) {
  final h = d.inHours;
  final m = d.inMinutes.remainder(60);
  final s = d.inSeconds.remainder(60);
  final mm = m.toString().padLeft(2, '0');
  final ss = s.toString().padLeft(2, '0');
  return h > 0 ? "$h:$mm:$ss" : "$mm:$ss";   // HH:MM:SS or MM:SS (CONTEXT)
}
```

### Reusing ElevationProfile as PageView page 1
```dart
// Source: ElevationProfile signature from elevation_profile.dart:21-29 [VERIFIED: codebase]
// trailAsync already watched in build() at navigation_screen.dart:103
trailAsync.when(
  data: (trail) => (trail.expand?.gpx != null)
      ? ElevationProfile(
          trail: trail,
          gpx: trail.expand!.gpx!,
          enableLineTouch: false,   // optional: disable scrub during nav
        )
      : const SizedBox.shrink(),
  loading: () => const Center(child: CircularProgressIndicator()),
  error: (e, _) => const SizedBox.shrink(),
)
```

### Removing the old exit button (CONTEXT)
```dart
// DELETE this block from navigation_screen.dart:259-267 (the top-left SafeArea>Align>exit)
// and the _buildExitButton helper (372-392) — or relocate the InkWell content into
// the sheet's button row IconButton. Exit action stays: () => context.pop().
```

## Runtime State Inventory

> Phase 3 is additive UI + a new in-memory provider. It is **not** a rename/refactor/migration phase. This section is included only to confirm no hidden runtime state is affected.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — `NavigationStats` is session-only in-memory (mirrors `NavigationState` D-19), discarded on family disposal | None |
| Live service config | None — no external services touched | None |
| OS-registered state | None — no new background tasks; GPS stream unchanged (no new permissions) | None |
| Secrets/env vars | None | None |
| Build artifacts | New provider requires `build_runner` regen to produce `navigation_stats_provider.g.dart` (and `.freezed.dart` if `NavigationStats` is a `@freezed` class) | Run `dart run build_runner build` after creating the provider |

**Nothing found requiring data migration — verified by inspecting `navigation_provider.dart` (session-only state) and `trail_provider.dart` (read-only GPX load).**

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| REQUIREMENTS.md "4 horizontal-swipe stat pages" | CONTEXT "1 stats page (collapse/expand) + 1 elevation page, button-driven" | 2026-06-13 conversation design | Planner follows CONTEXT; horizontal swipe is OFF (`NeverScrollableScrollPhysics`) |
| Top-left overlay exit button (NAV-07, Phase 2) | Exit consolidated into sheet button row | CONTEXT | Remove `navigation_screen.dart:259-267` + `_buildExitButton` |

**Deprecated/outdated:** None at the package level — all deps current per `pubspec.yaml`.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Altitude noise-floor threshold of ~2–3 m for gain/loss accumulation | Pitfall 2 | Too low → fake gain on flat ground; too high → real small climbs missed. Tunable named constant; field-test to confirm. |
| A2 | Collapsed `initialChildSize`/`minChildSize` ≈ 0.18 and expanded `maxChildSize` ≈ 0.45 | Pattern 2 | Wrong values may cover the map's bottom-right control column or clip stat content. Exact fractions are a UI-tuning decision for the planner/implementer. |
| A3 | `NavigationStats` state implemented as a `@freezed` class (vs a plain immutable class like `NavigationState`) | Standard Stack | Either works; `navigation_provider.dart` used a hand-written immutable class, `navigate_response.dart` used `@freezed`. Planner picks; freezed gives `copyWith`/equality for free. |
| A4 | `enableLineTouch: false` on the reused `ElevationProfile` during navigation | Code Examples | If scrubbing is desired during nav, leave default `true`. Pure UX preference. |
| A5 | Average speed uses `distance / elapsed × 3.6` excluding paused time | Phase Requirements | If "average including stopped time" is wanted, omit `_pausedAccum` subtraction. CONTEXT formula `distanceMeters / elapsed.inSeconds * 3.6` implies elapsed already excludes pause. |

## Open Questions

1. **Should "distance remaining" also be shown (STATS-02 original intent)?**
   - What we know: CONTEXT compact row specifies distance *covered*; original STATS-02 said distance *remaining*.
   - What's unclear: Whether to surface both. Remaining = total Valhalla route length − covered.
   - Recommendation: Implement distance covered per CONTEXT; if remaining is wanted, derive it cheaply (sum of `NavigateResponse` shape segment lengths once, minus covered). Flag to planner as optional.

2. **Where does the elapsed clock start — first GPS fix or screen mount?**
   - What we know: CONTEXT says "wall clock from first GPS fix" in one place and "stopwatch from navigation start" in another.
   - What's unclear: Slight ambiguity between mount-time and first-fix.
   - Recommendation: Start on first `onPosition` call (first GPS fix) — matches the dominant CONTEXT phrasing and avoids counting GPS-acquisition latency.

3. **Does `Geolocator.getPositionStream()` use default `LocationSettings`?**
   - What we know: `navigation_screen.dart:52` calls it with no `locationSettings` arg → platform default accuracy and `distanceFilter: 0` (emits on every fix). Speed/altitude are populated by default.
   - What's unclear: Whether the default cadence is frequent enough for smooth stats. It is for Phase 2's needs.
   - Recommendation: Do not change the stream settings in Phase 3 (would risk D-13/Phase 2 behavior). Stats consume whatever cadence the existing stream provides.

## Environment Availability

> Skipped — Phase 3 is code-only (new Dart provider + widget edits). No new external tools, services, or runtimes beyond the existing Flutter/Dart toolchain already required by the project. `build_runner` is already a dev dependency and used by the existing codegen providers.

## Validation Architecture

> `workflow.nyquist_validation` is `false` in `.planning/config.json` — this section is intentionally omitted per the research spec.

## Security Domain

> `security_enforcement: true`, `security_asvs_level: 1`. Phase 3 is a client-side UI/state feature with **no new** network calls, no new inputs from untrusted sources, no auth/session/crypto surface, and no new permissions (reuses the existing GPS stream).

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | No auth surface added |
| V3 Session Management | no | No sessions touched |
| V4 Access Control | no | No new authorization decisions |
| V5 Input Validation | minimal | Guard GPS-derived values (`speed`/`altitude` may be `NaN`/negative) before display — see Pitfall 3. Not security-critical, but prevents UI corruption. |
| V6 Cryptography | no | No crypto |
| V7 Error Handling/Logging | minimal | Stats provider should not throw on bad `Position` data; degrade gracefully (mirror Phase 2 GPS `onError` swallow at `navigation_screen.dart:68`) |

### Known Threat Patterns for Flutter client GPS UI

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Spoofed/garbage GPS values (`NaN`, negative speed) crashing UI | Tampering (low) | Validate/clamp before formatting (Pitfall 3) |
| Timer/stream leak draining battery & leaking state post-exit | Denial of Service (local) | `ref.onDispose` cancel timer (Pitfall 5) |

**No HIGH-severity findings.** Nothing here blocks under `security_block_on: high`.

## Project Constraints (from CLAUDE.md)

- **Tech stack is fixed:** Flutter + Riverpod (`riverpod_annotation` codegen) + go_router + flutter_map + freezed — must follow existing patterns. New provider MUST use `@riverpod` codegen.
- **No breaking changes:** Existing trail detail screens, bottom nav, routes unaffected. Phase 3 edits only `NavigationScreen` + adds one provider + extends `format_util.dart`.
- **Online-only v1:** No offline assumptions.
- **Naming:** Provider file `navigation_stats_provider.dart` → `navigation_stats_provider.g.dart` (project convention). Dart classes PascalCase; private fields `_`-prefixed; camelCase members.
- **Riverpod patterns:** `@riverpod` annotation, family-keyed, access other providers via `ref`; codegen `.g.dart`. Freezed for immutable state models (`@freezed abstract class ... with _$...`).
- **Error handling:** Degrade gracefully; do not throw from the notifier on bad GPS data.

## Sources

### Primary (HIGH confidence)
- Codebase (read this session): `navigation_screen.dart`, `navigation_provider.dart`, `navigate_response.dart`, `trail_detail_map_screen.dart`, `elevation_profile.dart`, `gpx_util.dart`, `trail_provider.dart`, `format_util.dart`, `pubspec.yaml`, `CLAUDE.md`, `03-CONTEXT.md`, `REQUIREMENTS.md`, `STATE.md`, `config.json`
- [pub.dev geolocator](https://pub.dev/packages/geolocator) — `Position` fields: `speed` (m/s), `altitude` (m MSL), `LocationSettings`/`distanceFilter`, `getPositionStream`

### Secondary (MEDIUM confidence)
- [Baseflow/flutter-geolocator#1412](https://github.com/Baseflow/flutter-geolocator/issues/1412) — altitude can be 0/jittery on some devices
- [Baseflow/flutter-geolocator#993](https://github.com/Baseflow/flutter-geolocator/issues/993) — speed availability/sentinel-value problems
- Flutter framework docs — `PageView`, `NeverScrollableScrollPhysics`, `DraggableScrollableSheet` (`snap`, `snapSizes`, builder `scrollController` contract)

### Tertiary (LOW confidence)
- Altitude noise-floor threshold (2–3 m) — common practice, not from a single authoritative source; flagged `[ASSUMED]` (A1)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — zero new packages; all verified present in `pubspec.yaml` and used in codebase
- Architecture: HIGH — directly mirrors existing `navigationProvider` + `trail_detail_map_screen` sheet patterns; D-13 constraint confirmed in code
- Pitfalls: HIGH for gesture/sheet/disposal (codebase-grounded); MEDIUM for GPS altitude/speed handling (issue-tracker + docs, exact threshold ASSUMED)

**Research date:** 2026-06-13
**Valid until:** 2026-07-13 (stable — no fast-moving external deps; revisit only if geolocator or fl_chart major versions change)
