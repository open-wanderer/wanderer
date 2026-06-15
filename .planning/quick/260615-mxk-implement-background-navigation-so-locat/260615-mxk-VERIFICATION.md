---
phase: quick-260615-mxk
verified: 2026-06-15T00:00:00Z
status: gaps_found
score: 4/6 must-haves verified
overrides_applied: 0
gaps:
  - truth: "GPS position events continue to fire while the phone screen is locked on Android"
    status: failed
    reason: "The two-step Always upgrade block in navigation_launch_util.dart (line 134) is not gated to iOS only. On Android 10+, Geolocator.checkPermission() can return LocationPermission.whileInUse, causing a second Geolocator.requestPermission() call that triggers the OS background-location dialog without user rationale — violating Google Play policy and producing unexpected UX. The comment at line 132 incorrectly asserts this is a no-op on Android. CR-02 from code review is confirmed unresolved."
    artifacts:
      - path: app/lib/util/navigation_launch_util.dart
        issue: "Line 134: `if (permission == LocationPermission.whileInUse)` missing `&& Platform.isIOS` guard. On Android, this unexpectedly triggers the ACCESS_BACKGROUND_LOCATION system dialog."
    missing:
      - "Gate the second requestPermission() call to iOS only: `if (permission == LocationPermission.whileInUse && Platform.isIOS)`"
      - "Add `import 'dart:io' show Platform;` to navigation_launch_util.dart if not already present (check line 1-20)"
  - truth: "geolocator 14.x imports compile without errors (AndroidSettings, AppleSettings, ForegroundNotificationConfig from top-level geolocator package)"
    status: failed
    reason: "navigation_screen.dart calls Platform.isAndroid and Platform.isIOS at lines 80 and 94 inside _buildLocationSettings() without a kIsWeb early-return guard. dart:io is not available on Flutter Web, so any call to Platform.* throws UnsupportedError at runtime on web build targets. kIsWeb is already imported on line 4 but is not used. CR-03 from code review is confirmed unresolved."
    artifacts:
      - path: app/lib/routes/navigation_screen.dart
        issue: "Line 79-108: _buildLocationSettings() calls Platform.isAndroid/Platform.isIOS without a `if (kIsWeb) return const LocationSettings(...)` guard at the top of the method."
    missing:
      - "Add `if (kIsWeb) { return const LocationSettings(accuracy: LocationAccuracy.high, distanceFilter: 5); }` as the first statement in _buildLocationSettings()"
human_verification:
  - test: "Enable Background Modes in Xcode: open app/ios/Runner.xcworkspace, select Runner target, Signing & Capabilities tab, + Capability, add Background Modes, check Location Updates."
    expected: "Runner target shows 'Location Updates' checked under Background Modes in Signing & Capabilities."
    why_human: "UIBackgroundModes in Info.plist alone is insufficient; the Xcode capability writes to Runner.xcodeproj/project.pbxproj and cannot be automated. This is a plan-declared blocking manual step (Task 1)."
  - test: "Build and run the app on a physical Android 10+ device. Grant fine location permission, start navigation, lock the screen, walk for 30 seconds, then unlock."
    expected: "The navigation screen advances maneuvers correctly after unlock; the Android notification tray shows a persistent 'Wanderer Navigation' foreground service notification while locked."
    why_human: "Background GPS continuation and foreground service notification require runtime on a real device — cannot be verified statically."
  - test: "Build and run on a physical iOS device running iOS 14+. Tap Navigate, grant When In Use permission when prompted (decline Always if prompted). Lock the screen during navigation."
    expected: "The blue location indicator pill is visible in the iOS status bar while locked; navigation continues to advance. Navigation is NOT blocked when Always is declined."
    why_human: "iOS background location indicator and permission flow behavior require runtime on a real device."
---

# Quick Task 260615-mxk: Background Navigation Verification Report

**Task Goal:** Implement background navigation so location tracking continues when the phone screen locks
**Verified:** 2026-06-15
**Status:** gaps_found
**Score:** 4/6 must-haves verified

## Goal Achievement

### Observable Truths

| #  | Truth                                                                                     | Status    | Evidence                                                                                     |
|----|-------------------------------------------------------------------------------------------|-----------|----------------------------------------------------------------------------------------------|
| 1  | GPS position events continue to fire while the phone screen is locked on Android          | FAILED    | CR-02 unresolved: `whileInUse` guard missing `&& Platform.isIOS` — triggers background dialog on Android unexpectedly |
| 2  | GPS position events continue to fire while the phone screen is locked on iOS              | UNCERTAIN | Requires human/device test; plist and AppleSettings wiring present but Xcode capability not confirmed |
| 3  | Android displays a persistent foreground service notification titled 'Wanderer Navigation' | VERIFIED  | `ForegroundNotificationConfig(notificationTitle: 'Wanderer Navigation', setOngoing: true, enableWakeLock: true)` in navigation_screen.dart line 85-92 |
| 4  | iOS shows the blue location indicator pill when navigation runs in background             | UNCERTAIN | `showBackgroundLocationIndicator: true` set in AppleSettings; Xcode Background Modes capability unconfirmed |
| 5  | Navigation is not blocked if the iOS user declines the Always permission upgrade          | VERIFIED  | Line 136 comment confirms no block; permission variable updated but navigation proceeds regardless |
| 6  | geolocator 14.x imports compile without errors                                            | FAILED    | CR-03 unresolved: `Platform.isAndroid`/`Platform.isIOS` called without `kIsWeb` guard — runtime crash on web builds |

**Score:** 4/6 truths verified (CR-01 from code review resolved by manifest merger — plugin ships its own `<service>` element)

### Required Artifacts

| Artifact                                              | Expected                                           | Status   | Details                                                                 |
|-------------------------------------------------------|----------------------------------------------------|----------|-------------------------------------------------------------------------|
| `app/pubspec.yaml`                                    | geolocator constraint `^14.0.0`                    | VERIFIED | Line 32: `geolocator: ^14.0.0`; pubspec.lock resolves to `14.0.3`      |
| `app/android/app/src/main/AndroidManifest.xml`        | Foreground service and background location perms   | VERIFIED | FOREGROUND_SERVICE, FOREGROUND_SERVICE_LOCATION, ACCESS_BACKGROUND_LOCATION all present (lines 6-8) |
| `app/ios/Runner/Info.plist`                           | Always description and UIBackgroundModes           | VERIFIED | `NSLocationAlwaysAndWhenInUseUsageDescription` (line 7) and `UIBackgroundModes` with `location` (lines 9-12) |
| `app/lib/routes/navigation_screen.dart`               | `_buildLocationSettings()` wired into initState    | PARTIAL  | Method exists and is called; missing `kIsWeb` guard — CR-03 open        |
| `app/lib/util/navigation_launch_util.dart`            | Two-step iOS Always permission flow                | PARTIAL  | `whileInUse` block present (line 134) but lacks `Platform.isIOS` guard — CR-02 open |

### Key Link Verification

| From                                | To                              | Via                                                              | Status  | Details                                                                    |
|-------------------------------------|---------------------------------|------------------------------------------------------------------|---------|----------------------------------------------------------------------------|
| `navigation_screen.dart`            | geolocator AndroidSettings/AppleSettings | `_buildLocationSettings()` called in initState, passed to `getPositionStream(locationSettings: ...)` | WIRED   | Lines 114-116: `Geolocator.getPositionStream(locationSettings: _buildLocationSettings())` confirmed |
| `navigation_launch_util.dart`       | iOS CoreLocation two-step flow  | Second `Geolocator.requestPermission()` when `permission == LocationPermission.whileInUse` | PARTIAL | Pattern present (line 134-136) but missing `Platform.isIOS` guard; triggers on Android too |

### Android Service Declaration (CR-01 — Resolved by manifest merger)

The code review flagged a missing `<service>` element in the app's own AndroidManifest.xml. Verification found the geolocator_android plugin ships its own entry via Android manifest merger:

```
app/build/geolocator_android/intermediates/merged_manifest/debug/.../AndroidManifest.xml
  <service
      android:name="com.baseflow.geolocator.GeolocatorLocationService"
      android:enabled="true"
      android:exported="false"
      android:foregroundServiceType="location" />
```

The app manifest does not need to re-declare this. CR-01 is a non-issue for this codebase.

### Behavioral Spot-Checks

Step 7b: SKIPPED — verifying background GPS behavior requires runtime on a physical device with locked screen. Static analysis is the applicable verification method here.

### Anti-Patterns Found

| File                                      | Line | Pattern                                              | Severity | Impact                                                                 |
|-------------------------------------------|------|------------------------------------------------------|----------|------------------------------------------------------------------------|
| `app/lib/routes/navigation_screen.dart`   | 80   | `Platform.isAndroid` called without `kIsWeb` guard   | BLOCKER  | Runtime `UnsupportedError` crash on Flutter Web build target           |
| `app/lib/routes/navigation_screen.dart`   | 94   | `Platform.isIOS` called without `kIsWeb` guard       | BLOCKER  | Same crash path as above; both checks within same method               |
| `app/lib/util/navigation_launch_util.dart`| 134  | `whileInUse` block missing `Platform.isIOS` guard    | BLOCKER  | Triggers Android background-location system dialog without rationale; violates Google Play policy |
| `app/lib/routes/navigation_screen.dart`   | 206  | `requireValue` on authProvider (WR-01 from review)   | WARNING  | Throws `StateError` on loading/error state; not directly blocking background nav goal |

### Human Verification Required

**1. Xcode Background Modes Capability**

**Test:** Open `app/ios/Runner.xcworkspace` in Xcode. Select the Runner target. Go to Signing & Capabilities. Verify that Background Modes is present with Location Updates checked.
**Expected:** "Location Updates" checkbox is checked in Background Modes.
**Why human:** This capability is recorded in `Runner.xcodeproj/project.pbxproj` by Xcode — it cannot be verified by reading Info.plist alone, and was Task 1 (blocking manual prerequisite) that the executor explicitly deferred.

**2. Android background GPS (physical device)**

**Test:** Build and install on a physical Android 10+ device. Grant fine location, start navigation, lock the screen, walk 30+ seconds, unlock.
**Expected:** Navigation maneuvers advance after unlock; foreground service notification "Wanderer Navigation" is visible in the notification tray while locked.
**Why human:** Background GPS continuation requires runtime behavior that cannot be verified statically.

**3. iOS background GPS and permission flow (physical device)**

**Test:** Build and install on a physical iOS 14+ device. Tap Navigate, grant When In Use when prompted, decline Always if offered. Lock the screen during navigation.
**Expected:** Blue location indicator pill visible in status bar while locked; navigation advances; app does not block or crash on Always decline.
**Why human:** iOS background location indicator and permission prompts require runtime on a real device.

### Gaps Summary

Two code-level gaps block the goal:

**Gap 1 (CR-02) — Android Always-upgrade guard missing (`navigation_launch_util.dart:134`)**
The two-step iOS Always upgrade block executes on all platforms. On Android 10+, `checkPermission()` legitimately returns `whileInUse`, causing the second `requestPermission()` to pop the system background-location dialog with no rationale — a Google Play policy violation and unexpected UX. Fix: add `&& Platform.isIOS` to the condition and import `dart:io show Platform` in that file.

**Gap 2 (CR-03) — Missing `kIsWeb` guard in `_buildLocationSettings()` (`navigation_screen.dart:79-108`)**
`Platform.isAndroid` and `Platform.isIOS` are called without a `kIsWeb` early-return check. `dart:io` is unavailable on Flutter Web; any call to `Platform.*` throws `UnsupportedError`. `kIsWeb` (from `package:flutter/foundation.dart`) is already imported at line 4 but unused. Fix: add `if (kIsWeb) return const LocationSettings(accuracy: LocationAccuracy.high, distanceFilter: 5);` as the first statement in `_buildLocationSettings()`.

Both gaps were identified in the code review (REVIEW.md) as critical blockers CR-02 and CR-03. The executor did not address them.

One previously-flagged blocker (CR-01, missing `<service>` element) is a non-issue: the `geolocator_android` plugin ships `GeolocatorLocationService` with `foregroundServiceType="location"` via Android manifest merger, confirmed in the build output.

---

_Verified: 2026-06-15_
_Verifier: Claude (gsd-verifier)_
