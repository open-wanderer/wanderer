---
phase: quick-260615-mxk
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - app/pubspec.yaml
  - app/android/app/src/main/AndroidManifest.xml
  - app/ios/Runner/Info.plist
  - app/lib/routes/navigation_screen.dart
  - app/lib/util/navigation_launch_util.dart
autonomous: false
requirements:
  - BG-01
must_haves:
  truths:
    - "GPS position events continue to fire while the phone screen is locked on Android"
    - "GPS position events continue to fire while the phone screen is locked on iOS"
    - "Android displays a persistent foreground service notification titled 'Wanderer Navigation' during navigation"
    - "iOS shows the blue location indicator pill when navigation runs in background"
    - "Navigation is not blocked if the iOS user declines the Always permission upgrade"
    - "geolocator 14.x imports compile without errors (AndroidSettings, AppleSettings, ForegroundNotificationConfig from top-level geolocator package)"
  artifacts:
    - path: app/pubspec.yaml
      provides: "geolocator constraint bumped to ^14.0.0"
      contains: "geolocator: ^14.0.0"
    - path: app/android/app/src/main/AndroidManifest.xml
      provides: "Foreground service and background location permissions"
      contains: "FOREGROUND_SERVICE_LOCATION"
    - path: app/ios/Runner/Info.plist
      provides: "Always permission description and UIBackgroundModes"
      contains: "NSLocationAlwaysAndWhenInUseUsageDescription"
    - path: app/lib/routes/navigation_screen.dart
      provides: "_buildLocationSettings() helper wired into initState"
      contains: "_buildLocationSettings"
    - path: app/lib/util/navigation_launch_util.dart
      provides: "Two-step iOS Always permission flow"
      contains: "whileInUse"
  key_links:
    - from: app/lib/routes/navigation_screen.dart
      to: geolocator AndroidSettings/AppleSettings
      via: "_buildLocationSettings() called in initState, result passed to Geolocator.getPositionStream(locationSettings: ...)"
      pattern: "locationSettings: _buildLocationSettings"
    - from: app/lib/util/navigation_launch_util.dart
      to: iOS CoreLocation two-step flow
      via: "second Geolocator.requestPermission() call when permission == LocationPermission.whileInUse"
      pattern: "whileInUse"
---

<objective>
Enable background location tracking so navigation continues when the phone screen locks.

Purpose: Hikers lock their phone while walking; without background location the navigation provider stops receiving position events, causing the maneuver engine to freeze.

Output: geolocator bumped to 14.x with platform-specific background settings wired in; Android foreground-service permissions and notification config; iOS Always permission flow and background mode plist entries; Xcode Background Modes capability enabled by the developer.
</objective>

<execution_context>
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/workflows/execute-plan.md
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/quick/260615-mxk-implement-background-navigation-so-locat/260615-mxk-RESEARCH.md
@app/pubspec.yaml
@app/android/app/src/main/AndroidManifest.xml
@app/ios/Runner/Info.plist
@app/lib/routes/navigation_screen.dart
@app/lib/util/navigation_launch_util.dart
</context>

<tasks>

<task type="checkpoint:human-action" gate="blocking">
  <name>Task 1 (prerequisite): Enable Background Modes in Xcode</name>
  <what-built>Nothing automated — this is a manual Xcode capability toggle that cannot be scripted. UIBackgroundModes in Info.plist alone is insufficient; iOS silently drops background updates if the Xcode capability is absent.</what-built>
  <how-to-verify>
    1. Open `app/ios/Runner.xcworkspace` in Xcode (use the .xcworkspace, NOT the .xcodeproj).
    2. In the Project Navigator (left sidebar), click Runner (the blue project icon at the top).
    3. In the main editor, select the Runner target (under TARGETS, not PROJECTS).
    4. Click the Signing & Capabilities tab.
    5. Click the + Capability button (top-left of the capabilities section).
    6. Search for "Background Modes" and double-click to add it.
    7. In the newly appeared Background Modes section, check Location Updates.
    8. Save (Cmd+S). You should see "Location Updates" checked in Signing & Capabilities.
  </how-to-verify>
  <resume-signal>Type "done" when the Location Updates checkbox is checked in Xcode</resume-signal>
</task>

<task type="auto">
  <name>Task 2: Bump geolocator to 14.x and add platform permissions/plist entries</name>
  <files>app/pubspec.yaml, app/android/app/src/main/AndroidManifest.xml, app/ios/Runner/Info.plist</files>
  <action>
    Make three file edits then run the upgrade command.

    **app/pubspec.yaml — line 32:**
    Change `geolocator: ^13.0.2` to `geolocator: ^14.0.0`.

    **app/android/app/src/main/AndroidManifest.xml:**
    After the existing `<uses-permission android:name="android.permission.POST_NOTIFICATIONS" />` line, add these three lines (before `<application>`):
    ```
    <uses-permission android:name="android.permission.VIBRATE" />
    <uses-permission android:name="android.permission.FOREGROUND_SERVICE" />
    <uses-permission android:name="android.permission.FOREGROUND_SERVICE_LOCATION" />
    <uses-permission android:name="android.permission.ACCESS_BACKGROUND_LOCATION" />
    ```
    Note: FOREGROUND_SERVICE is the base permission required for any foreground service on Android 9+ (API 28+). FOREGROUND_SERVICE_LOCATION is additionally required for location-type foreground services on Android 14 (API 34+). ACCESS_BACKGROUND_LOCATION is required for Android 10+ (API 29+) to keep GPS alive when the screen locks. The plugin's own manifest declares the GeolocatorLocationService service entry via manifest merger — the app does NOT re-declare it.

    **app/ios/Runner/Info.plist:**
    After the existing `NSLocationWhenInUseUsageDescription` key/string pair, add:
    ```xml
    <key>NSLocationAlwaysAndWhenInUseUsageDescription</key>
    <string>Wanderer needs your location at all times to continue navigation when the screen locks.</string>
    <key>UIBackgroundModes</key>
    <array>
        <string>location</string>
    </array>
    ```
    Use `NSLocationAlwaysAndWhenInUseUsageDescription` (the iOS 11+ key) — the deprecated `NSLocationAlwaysUsageDescription` must NOT be used.

    After all edits, run:
    `flutter pub upgrade geolocator`
    from the `app/` directory. This resolves to geolocator 14.0.3.
  </action>
  <verify>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/app && grep "geolocator:" pubspec.yaml | grep -v "^#" && grep "FOREGROUND_SERVICE_LOCATION" android/app/src/main/AndroidManifest.xml && grep "NSLocationAlwaysAndWhenInUseUsageDescription" ios/Runner/Info.plist && grep "UIBackgroundModes" ios/Runner/Info.plist</automated>
  </verify>
  <done>pubspec.yaml shows geolocator ^14.0.0; AndroidManifest.xml contains FOREGROUND_SERVICE, FOREGROUND_SERVICE_LOCATION, and ACCESS_BACKGROUND_LOCATION; Info.plist contains NSLocationAlwaysAndWhenInUseUsageDescription and UIBackgroundModes with location; pubspec.lock resolves geolocator to 14.x.</done>
</task>

<task type="auto">
  <name>Task 3: Wire _buildLocationSettings() into navigation_screen.dart and add two-step Always permission to navigation_launch_util.dart</name>
  <files>app/lib/routes/navigation_screen.dart, app/lib/util/navigation_launch_util.dart</files>
  <action>
    **app/lib/routes/navigation_screen.dart — two edits:**

    Edit 1 — Add import after the existing `import 'dart:async';` line:
    ```dart
    import 'dart:io' show Platform;
    ```

    Edit 2 — Add `_buildLocationSettings()` helper method inside `_NavigationScreenState`, placed immediately before `initState`. The method is platform-detecting:
    - Android: returns `AndroidSettings` with `accuracy: LocationAccuracy.high`, `distanceFilter: 5`, `intervalDuration: const Duration(seconds: 1)`, and a `ForegroundNotificationConfig` with `notificationTitle: 'Wanderer Navigation'`, `notificationText: 'Your location is being tracked for turn-by-turn navigation.'`, `notificationChannelName: 'Navigation'`, `enableWakeLock: true`, `setOngoing: true`.
    - iOS/macOS: returns `AppleSettings` with `accuracy: LocationAccuracy.high`, `distanceFilter: 5`, `allowBackgroundLocationUpdates: true`, `showBackgroundLocationIndicator: true` (required — not optional per Apple guidelines), `pauseLocationUpdatesAutomatically: false`.
    - Other platforms (fallback): returns `const LocationSettings(accuracy: LocationAccuracy.high, distanceFilter: 5)`.

    Edit 3 — In `initState` at line 82, replace:
    ```dart
    _positionStream = Geolocator.getPositionStream().asBroadcastStream();
    ```
    with:
    ```dart
    _positionStream = Geolocator.getPositionStream(
      locationSettings: _buildLocationSettings(),
    ).asBroadcastStream();
    ```

    All imports used (`AndroidSettings`, `AppleSettings`, `ForegroundNotificationConfig`, `LocationSettings`) come from the top-level `package:geolocator/geolocator.dart` import already present in the file — do NOT add sub-package imports.

    **app/lib/util/navigation_launch_util.dart — one edit:**

    After the existing permission block (lines 113–125, which ends after the inner `if` that calls `showError` on denied/deniedForever), add the two-step Always upgrade block. The full replacement for that permission block becomes:

    ```dart
    var permission = await Geolocator.checkPermission();
    if (permission == LocationPermission.deniedForever) {
      showError(l10n.location_permission_permanently_denied);
      return;
    }
    if (permission == LocationPermission.denied) {
      permission = await Geolocator.requestPermission();
      if (permission == LocationPermission.denied ||
          permission == LocationPermission.deniedForever) {
        showError(l10n.location_permission_denied);
        return;
      }
    }
    // Two-step iOS Always upgrade: if the user granted WhenInUse, call
    // requestPermission() again. iOS presents a system prompt asking
    // "Change to Always Allow?" only when NSLocationAlwaysAndWhenInUseUsageDescription
    // is present in Info.plist. On Android this call is a no-op.
    // Navigation proceeds even if the user declines the upgrade (whileInUse
    // + allowBackgroundLocationUpdates is sufficient on iOS).
    if (permission == LocationPermission.whileInUse) {
      permission = await Geolocator.requestPermission();
      // Do not block navigation if Always is declined — proceed with whileInUse.
    }
    ```

    After edits, run `flutter analyze app/lib/routes/navigation_screen.dart app/lib/util/navigation_launch_util.dart` from the repo root (or `flutter analyze` from the app/ directory) to verify no compilation errors.
  </action>
  <verify>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/app && grep -c "_buildLocationSettings" lib/routes/navigation_screen.dart && grep -c "dart:io" lib/routes/navigation_screen.dart && grep -c "whileInUse" lib/util/navigation_launch_util.dart && flutter analyze lib/routes/navigation_screen.dart lib/util/navigation_launch_util.dart 2>&1 | tail -5</automated>
  </verify>
  <done>navigation_screen.dart contains _buildLocationSettings() helper and passes locationSettings: _buildLocationSettings() to getPositionStream(); navigation_launch_util.dart contains the two-step whileInUse → requestPermission() Always upgrade block; flutter analyze reports no errors on both files.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Device OS → App | GPS position events cross the OS/app boundary via the geolocator foreground service; malicious location spoofing is possible at the OS level |
| App → Android notification system | Foreground service notification is displayed in the system tray; content is fixed strings, no user input |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-mxk-01 | Spoofing | Geolocator position stream | accept | GPS spoofing is an OS-level concern; app trusts OS location APIs (same posture as all navigation apps) |
| T-mxk-02 | Information Disclosure | ACCESS_BACKGROUND_LOCATION permission | mitigate | Permission is declared only for navigation use; Google Play Store review enforces this (user has accepted the review requirement) |
| T-mxk-03 | Denial of Service | OEM battery optimization killing foreground service | accept | enableWakeLock: true is set; further mitigation (battery exemption prompt) is a UX enhancement deferred to v2 per pitfall documentation |
| T-mxk-SC | Tampering | npm/pip/cargo installs | accept | No new package manager installs — geolocator version bump only, resolved via pub.dev (verified 14.0.3) |
</threat_model>

<verification>
1. `grep "geolocator: \^14.0.0" app/pubspec.yaml` returns a match
2. `grep "FOREGROUND_SERVICE_LOCATION" app/android/app/src/main/AndroidManifest.xml` returns a match
3. `grep "NSLocationAlwaysAndWhenInUseUsageDescription" app/ios/Runner/Info.plist` returns a match
4. `grep "_buildLocationSettings" app/lib/routes/navigation_screen.dart` returns a match
5. `grep "locationSettings: _buildLocationSettings" app/lib/routes/navigation_screen.dart` returns a match
6. `grep "whileInUse" app/lib/util/navigation_launch_util.dart` returns a match
7. `flutter analyze` (from app/) reports no errors
8. Xcode: Runner target → Signing & Capabilities → Background Modes → Location Updates is checked
</verification>

<success_criteria>
- geolocator 14.x resolves cleanly (pubspec.lock updated)
- AndroidManifest.xml contains all three foreground/background permission entries
- Info.plist contains NSLocationAlwaysAndWhenInUseUsageDescription and UIBackgroundModes location entry
- navigation_screen.dart uses _buildLocationSettings() with platform-specific Android foreground service notification and iOS always-background settings
- navigation_launch_util.dart performs two-step iOS Always permission upgrade without blocking navigation on decline
- Xcode Background Modes capability has Location Updates checked (manual step confirmed by developer)
- flutter analyze passes with no new errors
</success_criteria>

<output>
Create `.planning/quick/260615-mxk-implement-background-navigation-so-locat/260615-mxk-SUMMARY.md` when done
</output>
