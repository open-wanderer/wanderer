---
phase: 260615-mxk-implement-background-navigation-so-locat
reviewed: 2026-06-15T00:00:00Z
depth: quick
files_reviewed: 5
files_reviewed_list:
  - app/android/app/src/main/AndroidManifest.xml
  - app/ios/Runner/Info.plist
  - app/lib/routes/navigation_screen.dart
  - app/lib/util/navigation_launch_util.dart
  - app/pubspec.yaml
findings:
  critical: 3
  warning: 2
  info: 1
  total: 6
status: issues_found
---

# Phase 260615-mxk: Code Review Report

**Reviewed:** 2026-06-15
**Depth:** quick (with full file reads per file scope)
**Files Reviewed:** 5
**Status:** issues_found

## Summary

The implementation adds background GPS tracking via `geolocator ^14.0.0` with `AndroidSettings`/`AppleSettings` platform branching, a two-step iOS Always upgrade flow, and a foreground notification for Android. Three blockers were found: the Android foreground service `<service>` element is absent from the manifest (guaranteed crash on Android 10+), the two-step iOS Always upgrade is incorrectly executed on Android (disruptive/incorrect behavior), and `dart:io Platform` is called unconditionally without a `kIsWeb` guard (crash on web builds). Two warnings cover a `requireValue` crash risk and an unprotected GPS stream start at screen init time.

---

## Critical Issues

### CR-01: Missing `<service>` declaration with `foregroundServiceType="location"` in AndroidManifest

**File:** `app/android/app/src/main/AndroidManifest.xml:9` (inside `<application>`)

**Issue:** `AndroidSettings.foregroundNotificationConfig` causes `geolocator` to start a foreground service at runtime. Android 10 (API 29) requires every foreground service to be declared in the manifest, and Android 14 (API 34) made enforcement strict: starting an undeclared foreground service or one whose declared type does not match the runtime type throws `MissingForegroundServiceTypeException` / `SecurityException` and kills the app. The permissions `FOREGROUND_SERVICE` and `FOREGROUND_SERVICE_LOCATION` are present but are not sufficient alone — the `<service>` element must also be declared inside `<application>`.

**Fix:**
```xml
<!-- Inside <application>, before the closing tag -->
<service
    android:name="com.baseflow.geolocator.GeolocatorService"
    android:foregroundServiceType="location"
    android:exported="false" />
```

The exact class name `com.baseflow.geolocator.GeolocatorService` is the internal service class from `geolocator_android ^4.x`. Verify against the version pulled by `geolocator ^14.0.0` (`grep -r "GeolocatorService" app/.dart_tool/` or the plugin's `AndroidManifest.xml`). If the plugin ships its own `<service>` entry via manifest merger, this may be handled automatically — confirm by inspecting the merged manifest in the build output (`app/build/intermediates/merged_manifests/`).

---

### CR-02: Two-step iOS Always upgrade executes on Android, triggering background location dialog unexpectedly

**File:** `app/lib/util/navigation_launch_util.dart:134–137`

**Issue:** On Android 10+, `Geolocator.checkPermission()` can legitimately return `LocationPermission.whileInUse`. The code block at line 134 then calls `Geolocator.requestPermission()` a second time unconditionally. On Android this is not a no-op — it triggers the system background location dialog (`ACCESS_BACKGROUND_LOCATION`), which Android requires to be shown in a separate, explicit user action with rationale. Popping this dialog unexpectedly (without prior rationale display) violates Google Play policy and will confuse users who already granted fine location. The code comment at line 132 incorrectly states "On Android this call is a no-op."

```dart
// Current (buggy):
if (permission == LocationPermission.whileInUse) {
  permission = await Geolocator.requestPermission();
}
```

**Fix:** Gate the second request to iOS only:
```dart
if (permission == LocationPermission.whileInUse && Platform.isIOS) {
  permission = await Geolocator.requestPermission();
  // Do not block navigation if Always is declined — proceed with whileInUse.
}
```

---

### CR-03: `dart:io Platform` called without `kIsWeb` guard — crash on web build target

**File:** `app/lib/routes/navigation_screen.dart:80–107`

**Issue:** `_buildLocationSettings()` calls `Platform.isAndroid` and `Platform.isIOS` (via `dart:io show Platform`, imported at line 2). On Flutter Web, `dart:io` is not available and any call to `Platform.*` throws `UnsupportedError` at runtime, crashing the screen. Even if navigation is a mobile-only feature today, the app's build configuration may include web as a target (no explicit web exclusion exists in `pubspec.yaml`), and the error surface is silent until runtime.

```dart
// Current (line 80):
if (Platform.isAndroid) {
```

**Fix:** Add a `kIsWeb` early-exit guard at the top of the method:
```dart
LocationSettings _buildLocationSettings() {
  if (kIsWeb) {
    // Web does not support foreground services or Apple settings.
    return const LocationSettings(
      accuracy: LocationAccuracy.high,
      distanceFilter: 5,
    );
  }
  if (Platform.isAndroid) {
    // ... existing AndroidSettings ...
  } else if (Platform.isIOS || Platform.isMacOS) {
    // ... existing AppleSettings ...
  }
  return const LocationSettings(
    accuracy: LocationAccuracy.high,
    distanceFilter: 5,
  );
}
```

`kIsWeb` is already imported via `package:flutter/foundation.dart` (line 4).

---

## Warnings

### WR-01: `requireValue` on `authProvider` crashes if provider is in loading or error state

**File:** `app/lib/routes/navigation_screen.dart:206`

**Issue:** `ref.watch(authProvider).requireValue` throws `StateError` ("Tried to call `requireValue` on `AsyncLoading`" or `AsyncError`) if the auth provider has not yet resolved. If the `NavigationScreen` is ever reached via deep link, state restore, or a hot-reload scenario where providers are rebuilt mid-session, this throws and renders the entire screen as an error. The `user` value is only consumed in the `WaypointSheet` child (line 374-377), so the crash happens even though auth may not be strictly necessary to render the map.

**Fix:**
```dart
// Replace line 206:
final user = ref.watch(authProvider).valueOrNull;

// And in WaypointSheet call (line 373-377), guard:
if (_selectedWaypoint != null && user != null)
  WaypointSheet(
    waypoint: _selectedWaypoint!,
    user: user,
    controller: _waypointSheetController,
    onClose: () => setState(() => _selectedWaypoint = null),
  ),
```

---

### WR-02: GPS position stream started in `initState` without permission check — silent failure if screen reached without `launchNavigation`

**File:** `app/lib/routes/navigation_screen.dart:114–132`

**Issue:** `Geolocator.getPositionStream()` is called unconditionally in `initState`. If `NavigationScreen` is pushed without going through `launchNavigation` (e.g. via a future deep-link route, state restoration, or test harness), the stream will emit a `PermissionDeniedException`. The `onError` handler at line 129 only calls `debugPrint` — the user sees a frozen map with no GPS dot and no error message. There is no recovery path.

**Fix:** Either (a) assert that the route is only reachable via `launchNavigation` with a documented invariant and add a dev-mode assertion, or (b) wrap the stream subscription in a try/catch and show an error UI on `PermissionDeniedException`:
```dart
_sub = _positionStream.listen(
  (pos) { /* ... */ },
  onError: (Object error) {
    debugPrint('NavigationScreen: GPS stream error — $error');
    // TODO: surface a user-visible error if error is PermissionDeniedException
  },
);
```

---

## Info

### IN-01: `NSLocationAlwaysUsageDescription` absent from Info.plist — potential App Store review issue

**File:** `app/ios/Runner/Info.plist` (no line — key is absent)

**Issue:** The legacy `NSLocationAlwaysUsageDescription` key (required for iOS 10.x) is absent. iOS 11+ only requires `NSLocationAlwaysAndWhenInUseUsageDescription`, which is present. However, some App Store reviewers flag its absence, and some third-party SDK documentation still recommends including it as a belt-and-suspenders measure. This is low risk but has caused App Store rejections for other apps.

**Fix:** Add the key alongside the existing ones:
```xml
<key>NSLocationAlwaysUsageDescription</key>
<string>Wanderer needs your location at all times to continue navigation when the screen locks.</string>
```

---

_Reviewed: 2026-06-15_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: quick (with full file reads)_
