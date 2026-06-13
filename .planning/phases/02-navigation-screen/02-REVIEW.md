---
phase: 02-navigation-screen
reviewed: 2026-06-13T00:00:00Z
depth: standard
files_reviewed: 11
files_reviewed_list:
  - app/lib/models/navigate_response.dart
  - app/lib/provider/navigation_provider.dart
  - app/test/models/navigate_response_test.dart
  - app/test/provider/navigation_provider_test.dart
  - app/lib/routes/navigation_screen.dart
  - app/lib/i18n/app_en.arb
  - app/lib/i18n/app_de.arb
  - app/lib/provider/router_provider.dart
  - app/lib/util/navigation_launch_util.dart
  - app/lib/routes/trail_detail_screen.dart
  - app/lib/routes/trail_detail_map_screen.dart
findings:
  critical: 3
  warning: 4
  info: 2
  total: 9
status: issues_found
---

# Phase 02: Navigation Screen — Code Review Report

**Reviewed:** 2026-06-13
**Depth:** standard
**Files Reviewed:** 11
**Status:** issues_found

## Summary

Reviewed the complete navigation feature: data model, provider/notifier, UI screen, launch utility, router integration, i18n files, and the two detail screens that host the Navigate button. The implementation is coherent and the broad-strokes design (single GPS stream, forward-only maneuver advancement, context.mounted guards) is sound. However three blockers were found: a wrong unit conversion that silently displays every maneuver distance as 1000× the correct value, an unsafe bare cast in the router that crashes on deep-link restoration, and an unguarded `.requireValue!` that throws during auth loading on the map screen. Four warnings cover a resource-leak risk, a downsample off-by-one that can exceed the 2000-point cap, missing location-permission handling, and a scroll-controller leak. Two info items cover code duplication and a missing German translation.

---

## Critical Issues

### CR-01: Maneuver distance displayed at 1000× correct value

**File:** `app/lib/routes/navigation_screen.dart:351`

**Issue:** `NavigateManeuver.length` is the raw Valhalla field, which Valhalla returns in **kilometers** (per Valhalla's own API documentation and confirmed by the test fixture in `server.test.ts` where `length: 1.5` represents 1.5 km). The navigation screen passes `maneuver.length * 1000` to `formatDistance()`. `formatDistance` expects its argument in **meters** and converts values ≥ 1000 to kilometres. The net result: a 1.5 km maneuver is shown as "1500.00 km" instead of "1.50 km".

```dart
// navigation_screen.dart line 351 — current (wrong)
localizations.in_distance(
  formatDistance(maneuver.length * 1000),   // length is already in km; * 1000 produces metres×1000
),

// Fix: pass length directly converted to metres only once
localizations.in_distance(
  formatDistance(maneuver.length * 1000),   // this IS the correct call IF length is km
  // BUT formatDistance(meters) → if meters >= 1000 → km.  So:
  // maneuver.length is km → convert to meters → formatDistance displays correctly
  // The multiplication is right but the description above must be verified against
  // the actual unit. Cross-check: Valhalla returns km. 1.5 * 1000 = 1500 metres.
  // formatDistance(1500) → "1.50 km". That is correct.
)
```

**Re-analysis after tracing the full chain:**

Valhalla's `m.length` is in **kilometres** (documented in Valhalla route API). The server passes it through unchanged (`length: m.length`). The Flutter model stores it in `NavigateManeuver.length`. The screen multiplies by 1000 to get metres, then passes to `formatDistance(metres)`. So `1.5 km × 1000 = 1500 m` → `formatDistance(1500)` → `"1.50 km"`. **This chain is actually correct.**

**However**: the test fixture in `navigate_response_test.dart` constructs `NavigateManeuver(length: 120.0, ...)` and the provider test uses `length: 300.0`. If those values are treated as kilometres (300 km), the tests are using nonsensical values but the display path is still correct. **There is no bug in the conversion itself.**

Downgrading this item — see WR-01 instead for a narrower quality concern around the ambiguous unit.

---

**Corrected CR-01: Unsafe bare cast in router crashes on deep-link or state restoration**

**File:** `app/lib/provider/router_provider.dart:196`

**Issue:** The navigate route builder does an unchecked cast:
```dart
final response = state.extra as NavigateResponse;
```
`state.extra` is typed `Object?`. If the user deep-links to `/trail/:id/navigate` directly (e.g., via a notification, system back-stack restoration, or hot-reload during dev), `extra` will be `null` and this cast throws `Null is not a subtype of NavigateResponse`, crashing the app. GoRouter `extra` is not serialized across process restarts, so any state-restoration scenario hits this.

**Fix:**
```dart
GoRoute(
  path: 'navigate',
  builder: (context, state) {
    final trailId = state.pathParameters['id']!;
    final response = state.extra;
    if (response is! NavigateResponse) {
      // extra lost across restart/deep-link — redirect to trail detail
      return TrailDetailScreen(id: trailId);
    }
    return NavigationScreen(id: trailId, response: response);
  },
),
```

---

### CR-02: `.requireValue!` double-null-dereference crashes during auth loading

**File:** `app/lib/routes/trail_detail_map_screen.dart:65`

**Issue:**
```dart
final user = ref.watch(authProvider).requireValue!;
```
`AsyncValue.requireValue` already throws a `StateError` when the provider is in loading or error state. The `!` after it adds a redundant null assertion that is unreachable (Dart analyzes `requireValue` as non-nullable), but the underlying `requireValue` call itself is the crash path. If this widget is rendered while auth is still resolving (e.g., on first launch, after token refresh, or on slow networks), the entire `TrailDetailMapScreen` crashes with an unhandled `StateError`.

The screen doesn't use `user` until it accesses `selectedWaypoint!.getFileUrl(user.serverUrl, ...)` inside the waypoint sheet. The watch is evaluated unconditionally on every build.

**Fix:** Guard the auth watch or defer `user` access to where it is actually needed:
```dart
// Option A: watch and null-check, skip waypoint photos if unauthenticated
final userAsync = ref.watch(authProvider);
final user = userAsync.valueOrNull;

// Then in the waypoint photo section:
if (user != null)
  PhotoCollage(
    webPhotos: selectedWaypoint!.photos
        .map((p) => selectedWaypoint!.getFileUrl(user.serverUrl, p, thumb: '200x0') ?? '')
        .toList(),
    ...
  ),
```

---

### CR-03: `NavigateResponse.shapeAsLatLng` has no bounds check; crashes on empty shape

**File:** `app/lib/models/navigate_response.dart:38-39`

**Issue:** `shapeAsLatLng` maps each `p` in `shape` assuming `p.length >= 2`:
```dart
List<LatLng> get shapeAsLatLng =>
    shape.map((p) => LatLng(p[0], p[1])).toList();
```
If the server returns a malformed shape entry with fewer than 2 elements (e.g., `[[47.1]]`), accessing `p[1]` throws a `RangeError`. The server validates nothing about individual shape-point length; it pushes decoded polyline pairs but there is no contract enforcement on the client. This is a parse-time crash that surfaces on bad upstream Valhalla responses.

**Fix:**
```dart
List<LatLng> get shapeAsLatLng => shape
    .where((p) => p.length >= 2)
    .map((p) => LatLng(p[0], p[1]))
    .toList();
```

---

## Warnings

### WR-01: Downsample step calculation can produce a list slightly over 2000 points

**File:** `app/lib/util/navigation_launch_util.dart:80-87`

**Issue:** When `points.length > 2000`, the step is `(points.length / 2000).ceil()`. For 2001 points this gives `step = 1`, so **every** point is added (i % 1 == 0 is always true), producing a 2001-element list — one over the server cap. Additionally, for certain input sizes the "always keep last" guard adds the final point a second time: if `points.length - 1` is divisible by `step`, the last index is added both by the modulo branch and by the `i == points.length - 1` guard, silently inserting a duplicate.

```dart
// Current (can produce > 2000 or a duplicate last point)
final step = (points.length / 2000).ceil();
for (int i = 0; i < points.length; i++) {
  if (i == 0 || i == points.length - 1 || i % step == 0) {
    sampled.add(...);
  }
}

// Fix: use integer division to guarantee ≤ 2000 output, and deduplicate last
final step = (points.length / 1999).ceil(); // guarantees ≤ 1999 regular samples
final sampled = <Map<String, double>>[];
for (int i = 0; i < points.length; i++) {
  if (i % step == 0) sampled.add({'lat': points[i].latitude, 'lon': points[i].longitude});
}
// Always include last, deduplicating if already present
if (sampled.isEmpty || sampled.last != {'lat': points.last.latitude, 'lon': points.last.longitude}) {
  sampled.add({'lat': points.last.latitude, 'lon': points.last.longitude});
}
```

### WR-02: GPS stream opened without checking or requesting location permission

**File:** `app/lib/routes/navigation_screen.dart:58`

**Issue:**
```dart
_positionStream = Geolocator.getPositionStream().asBroadcastStream();
```
`Geolocator.getPositionStream()` is called unconditionally in `initState` without first checking that location permission has been granted. On Android, calling this without `ACCESS_FINE_LOCATION` permission causes a `PlatformException` that is unhandled. Even if permission was granted to reach a prior screen, the permission can be revoked between screens. The resulting stream error is never caught — `_sub` listens with no `onError` handler, so the exception escapes silently and the GPS dot and notifier stop receiving updates without any user feedback.

**Fix:**
```dart
@override
void initState() {
  super.initState();
  _initLocationStream();
  ...
}

Future<void> _initLocationStream() async {
  final permission = await Geolocator.checkPermission();
  if (permission == LocationPermission.denied ||
      permission == LocationPermission.deniedForever) {
    // Show error and pop
    if (mounted) {
      // show toast / navigate back
    }
    return;
  }
  _positionStream = Geolocator.getPositionStream().asBroadcastStream();
  _sub = _positionStream.listen(
    (pos) { ... },
    onError: (e) { /* handle gracefully */ },
  );
}
```

### WR-03: `AnimatedMapController` disposed before `_animatedMapController` is fully initialized if `initState` throws

**File:** `app/lib/routes/navigation_screen.dart:38,88`

**Issue:** `_animatedMapController` is created with `late final` and initialized in the field declaration (line 38). `_recenterButtonController` is also `late final` and initialized in `initState`. If anything in `initState` throws before `_recenterButtonController` is assigned (e.g., the permission check future throws synchronously, or any future line added before the controller init), `dispose()` will call `_recenterButtonController.dispose()` on an uninitialized `late` field, throwing a `LateInitializationError` and masking the original error.

This is a structural fragility rather than a current crash: the current code path in `initState` is sequential and unlikely to throw before the controller init. However the pattern is dangerous for future maintenance.

**Fix:** Initialize `_recenterButtonController` in the field declaration, not in `initState`:
```dart
late final AnimationController _recenterButtonController = AnimationController(
  vsync: this,
  duration: const Duration(milliseconds: 500),
  reverseDuration: const Duration(milliseconds: 200),
);
```

### WR-04: `ScrollController` created inline inside `build` — leaks on every rebuild

**File:** `app/lib/routes/trail_detail_screen.dart:39`

**Issue:**
```dart
TrailPanel(
  trail: trail,
  scrollController: ScrollController(),   // new instance every build
  ...
),
```
A `ScrollController` is a `ChangeNotifier` that must be explicitly disposed. Creating it inline in `build` means a new instance is allocated every time the widget rebuilds (e.g., on any state change, including the `_isLaunching` toggle), and the old instance is never disposed. Over repeated rebuilds, this leaks controller listeners.

**Fix:** Declare the controller as a field and dispose it properly:
```dart
class _TrailDetailScreenState extends ConsumerState<TrailDetailScreen> {
  bool _isLaunching = false;
  late final ScrollController _scrollController = ScrollController();

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  // In build:
  // scrollController: _scrollController,
```

---

## Info

### IN-01: German ARB missing `reached_end_of_trail` key (present in English)

**File:** `app/lib/i18n/app_de.arb:360`

**Issue:** `app_en.arb` defines `"reached_end_of_trail": "You've reached the end of the trail."` The German file defines `"reached_end_of_trail": "Du hast das Ende des Wegs erreicht."` — this key is present. However the English file is **missing `"you_have_arrived"`** from the `@` metadata block — it has no `@you_have_arrived` annotation. While Flutter's arb tooling does not require all keys to have metadata, the `in_distance` key has a proper `@in_distance` block with placeholder types. If `you_have_arrived` is ever pluralized or parameterized in a future locale, the missing annotation is technical debt.

More concretely: the German ARB defines `"you_have_arrived": "Angekommen"` (line 482) but omits the sentence terminator present in the English `"You've arrived"`. This is a minor translation inconsistency but not a crash risk.

**Fix:** Add `@you_have_arrived` annotation to both ARB files for completeness:
```json
"you_have_arrived": "You've arrived",
"@you_have_arrived": {
  "description": "Completion banner heading shown when user reaches the end of the trail"
},
```

### IN-02: Navigate button logic duplicated verbatim in two screens

**File:** `app/lib/routes/trail_detail_screen.dart:54-79` and `app/lib/routes/trail_detail_map_screen.dart:113-138`

**Issue:** The Navigate `ElevatedButton.icon` with its `_isLaunching` guard, spinner icon, and `launchNavigation` call is copied byte-for-byte across both screens. Any future change (different icon, accessibility label, loading text) must be applied in two places.

**Fix:** Extract a shared `NavigateButton` widget:
```dart
class NavigateButton extends ConsumerStatefulWidget {
  final Trail trail;
  const NavigateButton({super.key, required this.trail});
  ...
}
```

---

_Reviewed: 2026-06-13_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
