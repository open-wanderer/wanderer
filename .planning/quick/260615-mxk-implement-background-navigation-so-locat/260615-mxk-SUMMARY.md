---
phase: quick-260615-mxk
plan: 01
subsystem: flutter-navigation
tags: [background-location, geolocator, android, ios, permissions]
dependency_graph:
  requires: []
  provides: [background-gps-tracking]
  affects: [app/lib/routes/navigation_screen.dart, app/lib/util/navigation_launch_util.dart]
tech_stack:
  added: [geolocator 14.0.3 (bumped from 13.0.4)]
  patterns: [AndroidSettings foreground service notification, AppleSettings allowBackgroundLocationUpdates, two-step iOS Always permission flow]
key_files:
  created: []
  modified:
    - app/pubspec.yaml
    - app/pubspec.lock
    - app/android/app/src/main/AndroidManifest.xml
    - app/ios/Runner/Info.plist
    - app/lib/routes/navigation_screen.dart
    - app/lib/util/navigation_launch_util.dart
decisions:
  - geolocator bumped to ^14.0.0 (resolves 14.0.3); AndroidSettings/AppleSettings imported from top-level package
  - ACCESS_BACKGROUND_LOCATION added accepting Google Play Store policy review
  - Two-step Always permission flow implemented; navigation never blocked on decline
metrics:
  duration: ~15 min
  completed: "2026-06-15"
  tasks_completed: 2
  tasks_total: 3
  files_modified: 6
---

# Phase quick-260615-mxk Plan 01: Background Navigation Summary

**One-liner:** geolocator bumped to 14.0.3 with AndroidSettings foreground-service notification and AppleSettings always-background updates wired into navigation screen; two-step iOS Always permission flow added.

## What Was Built

Background location tracking so GPS position events continue firing when the phone screen locks during navigation.

### Task 1 (Manual Prerequisite — Xcode Background Modes)

**THIS STEP REQUIRES HUMAN ACTION IN XCODE.** The automated tasks (2 and 3) are complete and committed. Before testing on a real iOS device, the developer must enable the Background Modes capability in Xcode:

1. Open `app/ios/Runner.xcworkspace` in Xcode (use the `.xcworkspace`, NOT the `.xcodeproj`).
2. In the Project Navigator (left sidebar), click **Runner** (the blue project icon at the top).
3. In the main editor, select the **Runner** target (under TARGETS, not PROJECTS).
4. Click the **Signing & Capabilities** tab.
5. Click the **+ Capability** button (top-left of the capabilities section).
6. Search for **Background Modes** and double-click to add it.
7. In the newly appeared Background Modes section, check **Location Updates**.
8. Save (Cmd+S).

**Why this cannot be automated:** `UIBackgroundModes` in Info.plist alone is insufficient — iOS silently drops background updates if the Xcode capability is absent. The capability is recorded in `Runner.xcodeproj/project.pbxproj` by Xcode itself; editing that file manually is error-prone.

### Task 2 — Platform permissions and pubspec (commit 1aeea7f8)

- `app/pubspec.yaml`: geolocator constraint `^13.0.2` → `^14.0.0` (resolved to 14.0.3 in lock)
- `app/android/app/src/main/AndroidManifest.xml`: added `FOREGROUND_SERVICE`, `FOREGROUND_SERVICE_LOCATION`, `ACCESS_BACKGROUND_LOCATION` permissions
- `app/ios/Runner/Info.plist`: added `NSLocationAlwaysAndWhenInUseUsageDescription` and `UIBackgroundModes` with `location` entry

### Task 3 — Dart code changes (commit 9aef9b95)

- `app/lib/routes/navigation_screen.dart`:
  - Added `import 'dart:io' show Platform;`
  - Added `_buildLocationSettings()` helper returning `AndroidSettings` (foreground service notification with `enableWakeLock: true`, `setOngoing: true`, `distanceFilter: 5`) or `AppleSettings` (`allowBackgroundLocationUpdates: true`, `showBackgroundLocationIndicator: true`, `pauseLocationUpdatesAutomatically: false`) based on platform
  - Updated `initState` to pass `locationSettings: _buildLocationSettings()` to `Geolocator.getPositionStream()`
- `app/lib/util/navigation_launch_util.dart`:
  - Added two-step iOS Always permission upgrade block after `whileInUse` grant; navigation proceeds even if user declines Always

## Commits

| Task | Commit | Files |
|------|--------|-------|
| Task 2: pubspec + permissions | 1aeea7f8 | pubspec.yaml, pubspec.lock, AndroidManifest.xml, Info.plist |
| Task 3: Dart code changes | 9aef9b95 | navigation_screen.dart, navigation_launch_util.dart |

## Verification Results

All automated verification checks passed:

- `grep "geolocator: \^14.0.0" pubspec.yaml` — match found
- `grep "FOREGROUND_SERVICE_LOCATION" AndroidManifest.xml` — match found
- `grep "NSLocationAlwaysAndWhenInUseUsageDescription" Info.plist` — match found
- `grep "_buildLocationSettings" navigation_screen.dart` — 2 matches (definition + call site)
- `grep "locationSettings: _buildLocationSettings" navigation_screen.dart` — match found
- `grep "whileInUse" navigation_launch_util.dart` — 3 matches (comment + condition + comment)
- `flutter analyze lib/routes/navigation_screen.dart lib/util/navigation_launch_util.dart` — **No issues found**
- Xcode Background Modes: pending human action (Task 1)

## Deviations from Plan

None — plan executed exactly as written. Tasks 2 and 3 were executed in order per plan instructions. Task 1 (Xcode) documented as manual step in SUMMARY.

## Known Stubs

None. All code changes are fully wired — no placeholders or hardcoded empty values.

## Threat Surface Scan

No new network endpoints, auth paths, or schema changes introduced. Threat model items from plan:

| Threat ID | Category | Disposition | Notes |
|-----------|----------|-------------|-------|
| T-mxk-01 | Spoofing | accept | GPS spoofing is OS-level; app trusts OS location APIs |
| T-mxk-02 | Information Disclosure | mitigate | ACCESS_BACKGROUND_LOCATION declared for navigation use only |
| T-mxk-03 | DoS | accept | enableWakeLock: true set; OEM battery optimization mitigation deferred to v2 |

## Self-Check: PASSED

Files exist:
- app/pubspec.yaml — contains `geolocator: ^14.0.0`
- app/android/app/src/main/AndroidManifest.xml — contains `FOREGROUND_SERVICE_LOCATION`
- app/ios/Runner/Info.plist — contains `NSLocationAlwaysAndWhenInUseUsageDescription`
- app/lib/routes/navigation_screen.dart — contains `_buildLocationSettings`
- app/lib/util/navigation_launch_util.dart — contains `whileInUse`

Commits exist:
- 1aeea7f8 — feat(quick-260615-mxk): bump geolocator to ^14.0.0, add background permissions
- 9aef9b95 — feat(quick-260615-mxk): wire _buildLocationSettings() and add iOS Always permission flow
