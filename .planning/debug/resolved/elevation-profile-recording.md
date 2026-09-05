---
slug: elevation-profile-recording
status: resolved
resolved: 2026-08-09
trigger: "The elevation profile in the navigation screen is not showing up at all. I would expect it to show the profile up until the last recorded point using the elevation captured by GPS. Addendum: this is in recording mode. It works fine for navigation when using an existing track."
created: 2026-08-09
updated: 2026-08-09
---

# Debug Session: elevation-profile-recording

## Symptoms

- **Expected behavior:** While recording a new track in the navigation screen, the elevation profile should render, showing the profile from the start of the recording up to the most recently recorded point, using elevation values captured from GPS.
- **Actual behavior:** Nothing at all is rendered where the elevation profile should be — no card, no placeholder, no empty axes.
- **Error messages:** None observed. Logs have not been checked (no logcat / flutter console capture yet).
- **Timeline:** Never worked in recording mode. The elevation profile has always been missing during recording; it renders correctly when navigating an existing (pre-existing track) route.
- **Reproduction:** Open the navigation screen in recording mode (recording a new track, not following an existing trail) and look for the elevation profile.

## Scope notes

- Platform: Flutter app (`app/lib/`), navigation screen at `app/lib/routes/navigation_screen.dart` (file currently open in the user's IDE).
- Working branch: `feature/app`.
- Contrast case that WORKS: navigation with an existing track — the difference between these two paths is the primary lead.
- Recent related milestone: v1.8 "Offline Recording & Deferred Upload" (completed 2026-08-07).

## Current Focus

- hypothesis: CONFIRMED — `_buildElevationPage` in navigation_screen.dart unconditionally returns `SizedBox.shrink()` when `widget.isRecording` is true, before ever reaching the `trailAsync.when(...)` branch that builds the `ElevationProfile` chart. This is a deliberate early-return (there's a comment explaining it: "Recording mode has no trail GPX to profile — trailProvider('') resolves to AsyncError...") but it means the chart NEVER renders in recording mode, matching "never worked in recording mode" and "container renders (SizedBox height:216 wrapper), chart child renders nothing" exactly.
- test: read navigation_screen.dart in full around _buildElevationPage (line ~1875) and its caller (line ~1789)
- expecting: an explicit isRecording short-circuit that bypasses chart construction entirely
- next_action: DONE — user verified on device 2026-08-09 that the elevation profile now renders live while recording. Session resolved.
- reasoning_checkpoint:
    hypothesis: "The elevation profile never renders in recording mode because `_buildElevationPage` explicitly and unconditionally returns `SizedBox.shrink()` for `widget.isRecording == true`, before any chart-building logic runs."
    confirming_evidence:
      - "Direct code read: navigation_screen.dart lines 1879-1881 — `if (widget.isRecording) return const SizedBox.shrink();` — no condition, always taken for a recording session."
      - "The outer wrapper at line 1786-1790 is `SizedBox(key: ValueKey('elevation'), height: 216, child: _buildElevationPage(...))` — this explains the exact screenshot symptom: a container (the SizedBox) renders with a fixed blank area, while its child renders literally nothing."
      - "The non-recording path (line 1882+) calls `trailAsync.when(data: ... ElevationProfile(trail: trail, gpx: gpx, ...))` — confirming the working (navigate-existing-track) case goes through a completely different code path that isn't short-circuited."
    falsification_test: "If recording mode ever showed ANY chart/placeholder (even briefly), or if the non-recording path also failed to render, this hypothesis would be wrong. Neither is true per user report ('never worked in recording mode', works fine navigating existing track)."
    fix_rationale: "Root cause is the unconditional early-return itself, not a downstream rendering issue in ElevationProfile or elevation_tab.dart (elevation_tab.dart is unrelated — it's the route-planner's tab, not used by navigation_screen.dart at all). The fix removes the short-circuit and instead builds a live Gpx from the in-progress breadcrumb (same data structure/function, buildGpxFromPoints, already used by _saveRecordedTrack for the exact same breadcrumb) and passes it to the same ElevationProfile widget used elsewhere — reusing proven code paths rather than inventing new elevation logic."
    blind_spots: "Have not run flutter analyze/test yet against the actual fix. Have not verified on-device (user builds/installs per project constraint). Have not checked whether `Gpx`'s `==` deep-comparison cost (noted in elevation_profile.dart's didUpdateWidget comment) becomes a meaningful cost at high GPS-fix rates — mitigated by scoping the Consumer/select to breadcrumbLength and only building this widget while the user has the elevation page open."
- tdd_checkpoint:

## Evidence

- timestamp: 2026-08-09 (user-supplied screenshot, recording mode, Android)
  observation: Recording bottom sheet is expanded. The stats row renders correctly ("Time in Motion 00:00", "Afstand 0 m", "Hoogteverschil 0 m" — note mixed/NL locale strings). Below the stats row there is a large fully blank white area occupying most of the sheet where the elevation profile is expected. Bottom row shows three controls: pause (dark circle), stop (red circle), and a bar-chart icon button (bottom right) which appears to be a chart/profile toggle. No chart axes, no placeholder text, no spinner in the blank region.
  note: In this particular screenshot distance is 0 m and time 00:00, i.e. the recording had just started, so an empty series is expected AT THAT MOMENT. User reports the area stays blank throughout a recording, and that it has never worked in recording mode. The blank region + present chart-icon button suggests the container renders but the chart child renders nothing.
  files_of_interest: app/lib/routes/navigation_screen.dart, app/lib/components/route_planner/elevation_tab.dart (both opened by user in IDE)

- timestamp: 2026-08-09
  checked: app/lib/routes/navigation_screen.dart, `_buildElevationPage` (line 1875) and its caller (line 1786-1790)
  found: "`_buildElevationPage` starts with `if (widget.isRecording) return const SizedBox.shrink();` — an unconditional early return. Everything else in the method (the `trailAsync.when(...)` that builds the actual `ElevationProfile` chart) is unreachable in recording mode. The caller wraps this in `SizedBox(height: 216, child: _buildElevationPage(...))` — i.e. a fixed-height container that always renders, with a child that renders nothing when recording. This exactly matches the screenshot (blank area of fixed size, no chart, no placeholder)."
  implication: "Root cause confirmed — deliberate but incomplete short-circuit. `elevation_tab.dart` (the other file the user had open) is unrelated: it's the route-planner's own Elevation tab widget, never referenced by navigation_screen.dart."

- timestamp: 2026-08-09
  checked: app/lib/components/trail/elevation_profile.dart (`ElevationProfile` widget) and app/lib/util/gpx/gpx.dart (`buildGpxFromPoints`) and app/lib/provider/navigation_provider.dart (`NavigationState.breadcrumb`/`breadcrumbLength`)
  found: "`ElevationProfile` takes a plain `Gpx` (and an optional nullable `Trail`) and works standalone from live/unsaved GPX — this exact pattern (`trail: null`, live-built gpx) is already used by `elevation_tab.dart` for the route planner's in-progress route, so it's a proven code path, not new logic. `buildGpxFromPoints(List<Wpt>)` already exists and is already used on this same breadcrumb by `_saveRecordedTrack`. `NavigationState` docs explicitly warn: 'Consumers must key change detection on breadcrumbLength, never on this list's identity' (breadcrumb is a stable grow-in-place list) — so any live-update watch must select on `breadcrumbLength`, not `breadcrumb` itself."
  implication: "Fix path: build gpx from `navState.breadcrumb` via the existing `buildGpxFromPoints`, gate on `gpx.allPoints.length < 2` (matching elevation_tab.dart's own empty-state gate), and feed into the existing `ElevationProfile` — inside a scoped `Consumer` selecting `breadcrumbLength` (mirrors the existing `_buildPauseFab`/stats-sheet scoped-Consumer pattern used throughout this file to avoid whole-screen rebuilds per GPS fix)."

- timestamp: 2026-08-09 (orchestrator verification of the applied fix — SECOND DEFECT FOUND)
  checked: app/lib/components/trail/elevation_profile.dart `didUpdateWidget` (lines 55-66) against ~/.pub-cache/hosted/pub.dev/gpx-2.3.0/lib/src/model/{gpx,trkseg}.dart `operator ==`
  found: "The debugger's own listed blind spot ('have not verified Gpx == cost / behaviour') hid a second, independent defect that would have left the bug visibly unfixed. `ElevationProfile.didUpdateWidget` re-parses only when `!identical(old, new) && old.gpx != new.gpx`. gpx 2.3.0's `Gpx.==` compares `trks` via `const ListEquality().equals(...)`, and `Trkseg.==` compares `trkpts` the same way; `ListEquality.equals` short-circuits `if (identical(list1, list2)) return true`. Because `NavigationState.breadcrumb` is an identity-stable `UnmodifiableListView` over a grow-in-place backing list, a `Gpx` built directly over that view shares the SAME trkpts list instance with the previous rebuild's `Gpx` — so the two compare EQUAL after every GPS fix."
  implication: "As first applied, the fix would render the chart exactly once (at breadcrumbLength == 2) and then FREEZE at two points for the rest of the recording — `_points` never re-parsed. That fails the user's actual requirement ('show the profile up until the last recorded point'). Corrected by snapshotting: `List<Wpt>.of(breadcrumb)` per rebuild, so each `Gpx` wraps a distinct list whose differing length fails ListEquality on the cheap length check before any element comparison."
  verified_empirically: "Two new regression tests in test/components/trail/elevation_profile_test.dart assert both halves — the aliased-view Gpx pair compares equal (the trap), the snapshot pair compares unequal and yields a longer parsed series. Both pass, so the trap is proven rather than merely reasoned about."

## Eliminated

- hypothesis: "The blank area is caused by ElevationProfile failing to render GPS-recorded elevation, or by elevation_tab.dart"
  why_eliminated: "elevation_tab.dart is the route planner's tab and is never referenced by navigation_screen.dart. ElevationProfile itself renders fine — the non-recording path uses it successfully. The recording path never reached it."

- hypothesis: "Recorded breadcrumb points lack GPS elevation, so there would be nothing to plot"
  why_eliminated: "navigation_provider.dart:168 appends `Wpt(lat: pos.lat, lon: pos.lon, ele: altitude, time: DateTime.now())` — every breadcrumb point carries both elevation and a timestamp."

## Resolution

- root_cause: "`_buildElevationPage` in `app/lib/routes/navigation_screen.dart` unconditionally returned `SizedBox.shrink()` whenever `widget.isRecording` was true, before any chart-building code ran — so the elevation chart could never render during a recording session, regardless of how much breadcrumb data existed. The wrapping `SizedBox(height: 216, ...)` still rendered, producing the observed blank area of fixed size."
- fix: "Replaced the unconditional `SizedBox.shrink()` short-circuit with a scoped `Consumer` that builds a live `Gpx` from `navState.breadcrumb` (via the existing `buildGpxFromPoints` helper, the same one `_saveRecordedTrack` already uses for this breadcrumb) and renders it through the existing `ElevationProfile` widget (`trail: null`, matching the route-planner's `elevation_tab.dart` precedent for a live/unsaved gpx). The Consumer selects on `breadcrumbLength` (not `breadcrumb` itself, per `NavigationState`'s documented stable-list-identity gotcha) so only this page rebuilds per GPS fix, not the whole screen. Gated on `gpx.allPoints.length < 2` to show nothing (matching elevation_tab.dart's own empty-state gate) until there are at least 2 recorded points."
- fix_correction: "The fix above, as first applied, was incomplete: it passed the live identity-stable breadcrumb view straight into `buildGpxFromPoints`, which (see the third Evidence entry) makes successive `Gpx` objects compare equal and freezes the chart at its first two points. Corrected to snapshot per rebuild — `List<Wpt>.of(ref.read(_navProviderInstance).breadcrumb)` — with a comment recording why the copy is load-bearing, plus two regression tests locking the behaviour in."
- verification: "flutter analyze on both changed files: no issues. flutter test on elevation_profile_test.dart + navigation_provider_test.dart + navigation_stats_provider_test.dart: 40/40 passing (38 pre-existing + 2 new regression guards). Logic-traced against the two chart-building paths (trailAsync.when for navigate-existing-track — unchanged; new Consumer path for recording). No distance/elevation recording thresholds touched — only which existing display widget renders the already-recorded breadcrumb. Constraint-clean: no `late final` in a Notifier build(), no `.valueOrNull`, no `Isolate.run`, no flutter build / adb install run. AWAITING HUMAN ON-DEVICE VERIFICATION (user builds and installs per project constraint)."
- known_transient: "While `breadcrumbLength < 2` the elevation page still renders nothing (matching elevation_tab.dart's empty-state convention). That means the very first second or two of a recording still shows a blank area before the first two fixes land. Deliberate, not a regression — flagged to the user."
- files_changed:
  - app/lib/routes/navigation_screen.dart
  - app/test/components/trail/elevation_profile_test.dart
