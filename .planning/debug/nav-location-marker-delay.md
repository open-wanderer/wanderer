---
slug: nav-location-marker-delay
status: awaiting_human_verify
trigger: "When launching the navigation it takes a while until my location appears on the map. Probably, we do not use the initial location but wait until tracelet returns the first tick. I think we fixed a similar issue when recording a track"
created: 2026-09-05
updated: 2026-09-05T01:20:00Z
---

# Debug Session: nav-location-marker-delay

## Symptoms

- **Expected behavior:** On launching the navigation screen, the user's location marker should appear on the map immediately, seeded from the last known / initial location rather than waiting for the location stream's first emission.
- **Actual behavior:** The marker is absent for a few seconds (2-10s) after the navigation screen opens. The map camera is already centered on roughly the right area — only the position marker itself is missing until the first tick arrives.
- **Error messages:** None reported.
- **Timeline:** Always been this way on the navigation screen. The recording screen does not exhibit the delay — a similar issue was previously fixed for track recording.
- **Reproduction:** Launch navigation (Flutter app, `app/`). Observe that the location marker takes seconds to appear while the map itself is already positioned correctly.

## User's initial hypothesis

The navigation screen subscribes to the tracelet location stream and only renders the marker once the first tick is emitted, instead of seeding from an initial/last-known position the way the recording screen does.

## Current Focus

- hypothesis: CONFIRMED — `launchNavigation` (turn-by-turn nav) seeds `NavigationScreen.initialPosition` by waiting on a *new* live stream tick (`ForegroundPositionStream.currentFix`, 3s timeout) instead of the OS's instantly-available cached last-known position, so the seed is null far more often than not, leaving the marker blank until tracelet's own independent cold GPS acquisition completes.
- test: read `Geolocator` package docs for `getLastKnownPosition` (confirms it's the recommended instant-cache companion to a live fix); traced both `_openRecorder` (recording) and `launchNavigation` (turn-by-turn) call sites of `currentFix`/`seedPositionFrom`.
- expecting: recording never exhibits the delay because `_openRecorder` blocks (spinner, up to 20s) on a live fix before ever pushing the screen, guaranteeing `initialPosition` is populated; navigation exhibits it because its 3s wait is (a) too short for genuine cold-start GPS acquisition and (b) structurally unable to resolve when the foreground stream was already warmed by a prior screen and the device is stationary, since a NEW broadcast-stream subscriber only sees FUTURE ticks, and the native distance-filter (10m) means no future tick may arrive at all while stationary.
- next_action: awaiting human verification on-device (real GPS timing cannot be observed from this environment)
- reasoning_checkpoint:
  hypothesis: "`launchNavigation`'s seed-fetch (`ForegroundPositionStream.currentFix`, 3s timeout, waiting on a NEW stream event) fails to populate `NavigationScreen.initialPosition` in the common case, so the marker waits for tracelet's own independent cold GPS acquisition (2-10s) instead of appearing immediately."
  confirming_evidence:
    - "`_openRecorder` (trail_source_select_screen.dart:103-105) blocks with a 20s timeout + spinner on the SAME `currentFix()` helper before ever pushing `/record` — recording is documented/observed to never show the delay, consistent with it always having a resolved seed by the time the screen mounts."
    - "`launchNavigation` (launch_navigation.dart:153-156) uses only a 3s timeout on the same helper and does not block the whole flow on success — its own code comment concedes this is a 'best-effort... miss here must never delay or block' tradeoff."
    - "`ForegroundPositionStream.currentFix()` (foreground_position_stream_provider.dart:179-190) subscribes fresh to a broadcast stream and does `state.firstWhere((p) => p != null)` — broadcast streams do not replay past events to new subscribers, so it can only resolve on a NEW native position callback."
    - "The underlying native subscription (`_settings`, same file lines 96-100) is deliberately configured with a 10m Android distance filter specifically so 'a stationary device produces no callbacks at all' — meaning if any earlier screen already warmed the stream and the device hasn't moved, `currentFix()` can never resolve within any timeout while stationary, only ever hitting the 3s timeout."
    - "Geolocator's own package docs (`geolocator-14.0.3/lib/geolocator.dart:88-90`) state: 'The recommended use would be to call getLastKnownPosition to receive a cached position and update it with the result of getCurrentPosition' — i.e. the codebase never adopted the documented pattern; `getLastKnownPosition` is not referenced anywhere in `app/lib`."
  falsification_test: "If `Geolocator.getLastKnownPosition()` also frequently returned null/stale-beyond-use immediately after a fresh install with no prior fix ever resolved, the fix would need a live-stream fallback anyway (already retained) — this would not disprove the root cause, only require the fallback path, which is already in place."
  fix_rationale: "Seed from the OS's already-cached last-known position (returns near-instantly, per Geolocator's own recommended pattern) as the primary source, falling back to the existing 3s live-stream wait only when no cached position exists at all (e.g. location never resolved on this device before). This removes the dependency on a NEW stream tick ever arriving within the timeout, which is what the evidence shows fails structurally while stationary and unreliably even when moving."
  blind_spots: "Cannot run the app to observe real on-device GPS timing (project rule: user builds/installs, hand off after analyze+test). Cannot verify how stale a real device's `getLastKnownPosition()` typically is in practice, only that it returns immediately per the API. Not modifying `_openRecorder`/`_resolveInitialCenter` to use the same fast path since they aren't reported as buggy and recording's real-fix requirement is deliberate (no route shape to fall back on) — leaving them unchanged is a deliberate scope limit, not an oversight, but means this same latent pattern remains elsewhere if it is ever reported."
- tdd_checkpoint: (none — tdd_mode not set for this session)

## Evidence

- timestamp: 2026-09-05T00:00:00Z
  checked: app/lib/components/map/location_marker_layer.dart, app/lib/provider/foreground_position_stream_provider.dart, app/lib/services/tracelet_position_source.dart
  found: `TraceletPositionSource.start(seed:)` already emits a caller-supplied seed `geo.Position` into its stream immediately, overwritten the moment tracelet's own `_onLocation` fires. `ForegroundPositionStream.currentFix()` acquires the receiver, then does `state.firstWhere((p) => p != null).timeout(timeout)` — a fresh subscription to a broadcast stream, so it only sees future ticks, never replaying anything already emitted to other listeners.
  implication: The seed-forwarding mechanism itself works correctly; the bug must be in how/whether a seed value is obtained before `NavigationScreen` is constructed.

- timestamp: 2026-09-05T00:05:00Z
  checked: app/lib/routes/navigation_screen.dart (initState, full file)
  found: `widget.initialPosition` is passed as `seed:` to `_positionSource.start()` inside a `WidgetsBinding.instance.addPostFrameCallback`, and the `_sub = _positionStream.listen(...)` subscription is set up synchronously in `initState` BEFORE that callback fires — so ordering is correct; whenever a non-null seed exists, the marker renders essentially immediately (the position tween starts from the seed itself, so no visible lerp delay).
  implication: The wiring inside NavigationScreen is not the bug. The bug is upstream: whether `widget.initialPosition` is non-null when the screen is constructed.

- timestamp: 2026-09-05T00:10:00Z
  checked: app/lib/provider/router_provider.dart (both `/record` and `/trail/:id/navigate` routes)
  found: Both routes correctly thread the caller-resolved seed (`extra` tuple/map) into `NavigationScreen(initialPosition: ...)`.
  implication: Router wiring is correct; the seed's value depends entirely on what each caller resolves before pushing.

- timestamp: 2026-09-05T00:15:00Z
  checked: app/lib/routes/trail_source_select_screen.dart `_openRecorder` (lines 90-125) vs app/lib/actions/launch_navigation.dart (lines 147-156)
  found: `_openRecorder` calls `currentFix(timeout: Duration(seconds: 20))` and BLOCKS the whole flow (spinner shown, `context.push` only happens after) — `pos == null` shows an error and aborts instead of proceeding. `launchNavigation` calls `currentFix(timeout: Duration(seconds: 3))` and always proceeds regardless of the result (explicit design comment: 'a miss here must never delay or block starting navigation').
  implication: Recording is structurally guaranteed a real seed before the screen ever mounts (hence never exhibits the delay). Turn-by-turn navigation is not — a miss silently drops the seed and the marker falls through to tracelet's own cold acquisition.

- timestamp: 2026-09-05T00:20:00Z
  checked: app/lib/provider/foreground_position_stream_provider.dart `_settings` doc comment (lines 96-100) and `_startPositionStream` (lines 197-254)
  found: Android/iOS location settings use a 10m distance filter specifically so "a stationary device produces no callbacks at all" (documented battery optimization). `currentFix()`'s `firstWhere` requires a NEW callback to resolve.
  implication: Even when the foreground stream was already warmed by a prior screen (e.g. `trail_detail_map_screen.dart`'s `TrailMap(showLocation: true)`), a stationary device (the common case: user reads a trail's details, then taps Navigate) will never produce a new callback for `currentFix()` to observe — it can only ever time out at 3s, never succeed, explaining the "always" in the reported symptom.

- timestamp: 2026-09-05T00:25:00Z
  checked: git log -S "seedPositionFrom" -- app/lib (commit a241af58, "Seed nav screen with initial position")
  found: The seed-forwarding feature (`TraceletPositionSource.start(seed:)`, `seedPositionFrom`, `initialPosition` plumbing) was introduced in ONE commit covering both recording and navigation simultaneously — this is "the similar issue... fixed when recording a track" the user recalled. The 3s timeout and "best-effort, never block" comment were part of that same original commit, unchanged since.
  implication: The infrastructure for the fix already exists and is shared; the defect is specifically in `launchNavigation`'s acquisition strategy (waiting on a live tick) rather than a missing feature.

- timestamp: 2026-09-05T00:30:00Z
  checked: web search / package source — geolocator 14.0.3 (`~/.pub-cache/hosted/pub.dev/geolocator-14.0.3/lib/geolocator.dart` lines 50-90); confirmed `getLastKnownPosition` is never called anywhere in `app/lib` (`grep -rn "getLastKnownPosition" app/lib` → no matches)
  found: The package's own doc for `getCurrentPosition`/live acquisition explicitly recommends: "The recommended use would be to call the getLastKnownPosition method to receive a cached position and update it with the result of the getCurrentPosition method." This exact pattern (instant cached seed + live fix that supersedes it) is what `launchNavigation` is missing.
  implication: This directly matches the user's own hypothesis ("we do not use the initial location but wait until tracelet returns the first tick") — the fix is to seed from `getLastKnownPosition()` (instant, OS-cached) rather than waiting on a fresh stream tick.

## Eliminated

- hypothesis: The seed-forwarding plumbing (TraceletPositionSource/NavigationScreen/router) is broken or has an ordering bug.
  evidence: Full trace of `NavigationScreen.initState` shows `_sub` is attached before `_positionSource.start(seed:)` is ever called (deferred to a post-frame callback), so any non-null seed is guaranteed to be observed and rendered immediately. Router correctly threads the seed through both `/record` and `/trail/:id/navigate`.
  timestamp: 2026-09-05T00:15:00Z

## Resolution

- root_cause: `launchNavigation` (app/lib/actions/launch_navigation.dart) seeds the navigation marker by waiting up to 3 seconds for a NEW tick on the shared foreground position stream (`ForegroundPositionStream.currentFix`), rather than reading the OS's already-cached last-known position. Because a broadcast-stream subscriber only observes future events, and the stream's distance filter (10m) means a stationary device may never produce a new event at all, this wait fails far more often than it succeeds — leaving `NavigationScreen.initialPosition` null, so the live marker has nothing to render until tracelet's own independent, equally-cold GPS session produces its first fix (the observed 2-10s delay). Recording (`_openRecorder`) never has this problem because it blocks on a real fix (20s timeout, spinner) before the screen is ever pushed.
- fix: Seed from `Geolocator.getLastKnownPosition()` (returns near-instantly from the OS cache) as the primary source in `launchNavigation`; fall back to the existing 3s live-stream wait only when no cached position exists at all.
- verification: Self-verified — `flutter analyze` clean on the changed file and the whole app (13 pre-existing info-level lints elsewhere, unrelated to this change, no new issues). `flutter test` — all 1093 tests pass, 0 failures. Cannot verify real on-device GPS timing from this environment (project rule: user builds/installs, hand off after analyze+test) — awaiting on-device confirmation that the marker now appears immediately on navigation launch.
- files_changed:
  - app/lib/actions/launch_navigation.dart: seed acquisition now tries `Geolocator.getLastKnownPosition()` first (instant, OS-cached), falling back to the existing 3s live-stream wait only if no cached position exists.
