---
phase: quick-260619-okw
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - app/pubspec.yaml
  - app/lib/util/tracelet_position_source.dart
  - app/lib/routes/navigation_screen.dart
autonomous: false
requirements:
  - TRACELET-01
must_haves:
  truths:
    - "The tracelet package is verified as legitimate on pub.dev before install (verified publisher, real download/like counts)"
    - "`flutter pub get` resolves tracelet with no version-conflict errors"
    - "During navigation, GPS position events originate from tracelet (not Geolocator.getPositionStream) and continue firing while the phone screen is locked"
    - "navigationProvider.onPosition still receives a LatLng per fix and the maneuver banner advances as before"
    - "navigationStatsProvider.onPosition still receives altitude+speed per fix and distance/elevation/speed stats accumulate as before"
    - "CurrentLocationLayer on the map still tracks the live user position from the same tracelet feed"
    - "tracelet tracking is started in initState and stopped/disposed when the navigation screen is exited (no leaked background service)"
  artifacts:
    - path: app/pubspec.yaml
      provides: "tracelet dependency added under dependencies"
      contains: "tracelet:"
    - path: app/lib/util/tracelet_position_source.dart
      provides: "Adapter that starts tracelet and exposes its updates as a Stream<Position> mapped from Location.coords"
      contains: "Stream<Position>"
    - path: app/lib/routes/navigation_screen.dart
      provides: "Navigation screen wired to the tracelet-backed position stream instead of Geolocator.getPositionStream"
      contains: "TraceletPositionSource"
  key_links:
    - from: app/lib/util/tracelet_position_source.dart
      to: tracelet Tracelet.onLocation / ready / start / stop
      via: "onLocation callback pushes Position onto a StreamController; ready()+start() in start(); stop() on dispose"
      pattern: "Tracelet\\.(ready|start|stop|onLocation)"
    - from: app/lib/routes/navigation_screen.dart
      to: app/lib/util/tracelet_position_source.dart
      via: "_positionStream sourced from TraceletPositionSource.stream; start in initState, dispose in dispose"
      pattern: "TraceletPositionSource"
---

<objective>
Replace the navigation screen's location source with the `tracelet` background-geolocation package so position updates are produced by tracelet's battery-efficient background engine while the existing maneuver/stats/marker consumers keep working unchanged.

Purpose: tracelet is a purpose-built background-location library (motion-aware, headless-capable). Routing the navigation screen's position feed through tracelet hardens lock-screen tracking beyond the current raw `Geolocator.getPositionStream` while leaving the maneuver engine, stats accumulator, and live map marker untouched.

Output: tracelet added to pubspec, a `TraceletPositionSource` adapter that bridges `Tracelet.onLocation` into a `Stream<Position>`, and `navigation_screen.dart` rewired to consume that stream and to start/stop tracelet with the screen lifecycle.
</objective>

<execution_context>
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/workflows/execute-plan.md
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@app/pubspec.yaml
@app/lib/routes/navigation_screen.dart
@app/lib/provider/navigation_provider.dart
@app/lib/provider/navigation_stats_provider.dart
@app/lib/util/navigation_launch_util.dart
@app/android/app/src/main/AndroidManifest.xml
@app/ios/Runner/Info.plist

# Interface contract the adapter must satisfy (consumers in navigation_screen.dart initState):
#   ref.read(navigationProvider(response).notifier).onPosition(LatLng(lat, lng))
#   ref.read(navigationStatsProvider(response).notifier).onPosition(Position)  // uses .latitude .longitude .altitude .speed
#   CurrentLocationLayer(positionStream: fromGeolocatorPositionStream(stream: <Stream<Position>>))  // needs a geolocator Position stream
#
# tracelet API (pub.dev/packages/tracelet 3.5.x, verified publisher ikolvi.com, Apache-2.0):
#   await Tracelet.ready(Config.balanced().copyWith(geo: GeoConfig(desiredAccuracy: DesiredAccuracy.high, distanceFilter: 5.0), app: AppConfig(stopOnTerminate: false)))
#   await Tracelet.start();  await Tracelet.stop();
#   Tracelet.onLocation((Location loc) { loc.coords.latitude/.longitude/.altitude/.speed(m/s)/.heading/.accuracy });
</context>

<tasks>

<task type="checkpoint:human-verify" gate="blocking-human">
  <name>Task 1: Verify tracelet package legitimacy on pub.dev</name>
  <what-built>No code yet — this is the mandatory package-legitimacy gate before adding a new pub dependency. Research summary: `tracelet` 3.5.x, publisher ikolvi.com (verified), Apache-2.0, ~160 pub points, ~6.88k weekly downloads, ~38 likes. It is a real published package but lower-adoption than first-party plugins, so confirm before install.</what-built>
  <how-to-verify>
    1. Open https://pub.dev/packages/tracelet in a browser.
    2. Confirm the publisher badge shows a VERIFIED publisher (ikolvi.com) — not an unverified/uploader-only package.
    3. Confirm the license is Apache-2.0 and the package has real pub points (~150+), downloads (thousands/week), and likes (non-zero).
    4. Skim the changelog/README to confirm it is actively maintained (a recent version published).
    5. Confirm you accept adding this dependency for background location in the navigation screen.
  </how-to-verify>
  <resume-signal>Type "approved" to proceed with the install, or "reject" to stop and choose a different approach.</resume-signal>
</task>

<task type="auto">
  <name>Task 2: Add tracelet and build the TraceletPositionSource adapter</name>
  <files>app/pubspec.yaml, app/lib/util/tracelet_position_source.dart</files>
  <action>Add `tracelet` to the `dependencies:` block in app/pubspec.yaml (alongside geolocator), then run `flutter pub get` in the app/ directory to resolve it. Keep geolocator — it still owns permission checks in navigation_launch_util.dart and the existing platform permissions in AndroidManifest.xml / Info.plist already cover tracelet's needs (ACCESS_FINE/COARSE/BACKGROUND_LOCATION, FOREGROUND_SERVICE, FOREGROUND_SERVICE_LOCATION on Android; NSLocation* + UIBackgroundModes location on iOS). Do NOT remove or re-request permissions here.

  Create app/lib/util/tracelet_position_source.dart defining a `TraceletPositionSource` class that bridges tracelet's callback API into a broadcast `Stream<geolocator.Position>` so the navigation screen's existing consumers stay type-compatible. Requirements: (a) a `Stream<Position> get stream` backed by a broadcast `StreamController<Position>`; (b) a `Future<void> start()` that calls `Tracelet.onLocation(...)` to register the callback, then `await Tracelet.ready(Config.balanced().copyWith(geo: GeoConfig(desiredAccuracy: DesiredAccuracy.high, distanceFilter: 5.0), app: AppConfig(stopOnTerminate: false)))`, then `await Tracelet.start()` — match tracelet's documented ordering (register callback before ready, ready before start); (c) inside the onLocation callback, map `location.coords` to a geolocator `Position` and add it to the controller, populating latitude, longitude, altitude, speed (tracelet speed is already m/s, matching geolocator), accuracy, heading, and `timestamp: DateTime.now()`, with safe defaults (0) for any geolocator Position field tracelet does not provide; (d) a `Future<void> dispose()` that calls `await Tracelet.stop()` and closes the controller so the background service is not leaked on screen exit. Use a prefixed import for geolocator (`import 'package:geolocator/geolocator.dart' as geo;`) if tracelet exposes a conflicting `Location`/`Position` symbol. Add a `///` doc comment explaining the adapter mirrors the prior Geolocator.getPositionStream contract (high accuracy, 5 m distanceFilter, background-on-lock) per the 260615-mxk background-navigation decisions, and that tracelet is the new source of truth for position fixes.</action>
  <verify>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/app && flutter pub get && dart analyze lib/util/tracelet_position_source.dart</automated>
  </verify>
  <done>tracelet appears in pubspec.yaml dependencies, `flutter pub get` resolves with no conflict, and `dart analyze` reports no errors for tracelet_position_source.dart. The adapter exposes a `Stream<Position>` and start()/dispose() lifecycle methods backed by Tracelet.ready/start/stop and Tracelet.onLocation.</done>
</task>

<task type="auto">
  <name>Task 3: Rewire navigation_screen.dart to the tracelet-backed stream</name>
  <files>app/lib/routes/navigation_screen.dart</files>
  <action>Replace the Geolocator-based position feed with the TraceletPositionSource adapter while preserving every existing consumer.

  In _NavigationScreenState: add a `late final TraceletPositionSource _positionSource;` field. Keep the existing `late final Stream<Position> _positionStream;` and `StreamSubscription<Position>? _sub;` fields. In initState, construct `_positionSource = TraceletPositionSource()`, set `_positionStream = _positionSource.stream` (already a broadcast stream — drop the `.asBroadcastStream()` since the source is broadcast), call `unawaited(_positionSource.start())` to begin tracelet tracking, and keep the existing `_sub = _positionStream.listen(...)` block exactly as-is (it forwards each Position to `navigationProvider.onPosition(LatLng(pos.latitude, pos.longitude))`, `navigationStatsProvider.onPosition(pos)`, and triggers `_recenterTrigger` when follow is enabled). Remove the `_buildLocationSettings()` helper and its call, and remove the direct `Geolocator.getPositionStream(...)` call — tracelet now owns the OS location subscription and the per-platform accuracy/distanceFilter/background config lives in the adapter. Keep the `geolocator` import only if still referenced (Position type); otherwise prefix or trim it so analyze stays clean. Leave `CurrentLocationLayer(positionStream: const LocationMarkerDataStreamFactory().fromGeolocatorPositionStream(stream: _positionStream), ...)` unchanged — it consumes the same broadcast Position stream and keeps tracking the live marker. In dispose(), before/instead of any geolocator teardown, call `_sub?.cancel()` (existing) and `unawaited(_positionSource.dispose())` so tracelet is stopped when navigation exits. Add `import 'package:wanderer/util/tracelet_position_source.dart';` and ensure `unawaited` is available (it is imported via dart:async at the top).</action>
  <verify>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/app && dart analyze lib/routes/navigation_screen.dart lib/util/tracelet_position_source.dart && grep -v '^[[:space:]]*//' lib/routes/navigation_screen.dart | grep -c 'Geolocator.getPositionStream'</automated>
  </verify>
  <done>`dart analyze` reports no errors for both files; the grep returns 0 (no remaining Geolocator.getPositionStream call in non-comment code); navigation_screen.dart constructs TraceletPositionSource, starts it in initState, sources `_positionStream` from it, and disposes it in dispose(). The maneuver banner, stats sheet, and live location marker all consume the tracelet-backed stream unchanged.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| third-party pub package -> app runtime | tracelet runs a background service with location access and (per its feature set) optional HTTP sync / SQLite persistence — must not exfiltrate or sync location off-device |
| OS location service -> app | background GPS fixes cross into the app while the screen is locked |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-okw-SC | Tampering | tracelet pub install | mitigate | slopcheck + blocking-human legitimacy checkpoint (Task 1) verifying verified publisher, license, pub points, downloads on pub.dev before install |
| T-okw-01 | Information disclosure | tracelet background location data | mitigate | Use only on-device features (onLocation callback + local Config.balanced); do NOT configure tracelet's HTTP sync / remote endpoint — position fixes stay in-app, fed to existing in-memory providers only |
| T-okw-02 | Denial of service | leaked background service | mitigate | TraceletPositionSource.dispose() calls Tracelet.stop() and is invoked from navigation_screen dispose(); stopOnTerminate:false is bounded by explicit stop on screen exit |
| T-okw-03 | Elevation of privilege | location permission scope | accept | No new permissions requested; existing geolocator-granted fine/background permissions reused; permission gating remains in navigation_launch_util.dart |
</threat_model>

<verification>
- `flutter pub get` in app/ resolves tracelet with no dependency conflict.
- `dart analyze lib/util/tracelet_position_source.dart lib/routes/navigation_screen.dart` reports no errors.
- No remaining `Geolocator.getPositionStream` call in non-comment code of navigation_screen.dart.
- Manual (developer, on device): start navigation, lock the phone, walk; confirm the maneuver banner advances, stats accumulate, and the live marker tracks — proving tracelet feeds all three consumers and continues in the background.
</verification>

<success_criteria>
- tracelet legitimacy verified and approved before install (Task 1 checkpoint).
- tracelet added to app/pubspec.yaml and resolved via `flutter pub get`.
- TraceletPositionSource adapter bridges Tracelet.onLocation into a Stream<Position> and manages the ready/start/stop lifecycle.
- navigation_screen.dart sources its position stream from tracelet, starts tracking in initState, disposes it on exit, and keeps the maneuver provider, stats provider, and CurrentLocationLayer working unchanged.
- Background location continues while the screen is locked, now driven by tracelet.
</success_criteria>

<output>
Create `.planning/quick/260619-okw-use-tracelet-to-facilitate-background-lo/260619-okw-SUMMARY.md` when done.
</output>
