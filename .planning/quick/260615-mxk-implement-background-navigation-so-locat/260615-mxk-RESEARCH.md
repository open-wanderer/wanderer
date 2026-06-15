# Quick Task 260615-mxk: Background Navigation — Research

**Researched:** 2026-06-15
**Domain:** Flutter geolocator 14.x background location — Android foreground service + iOS background mode
**Confidence:** HIGH (API details verified against pub.dev docs and GitHub source)

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Flutter SDK >= 3.29.0 confirmed → bump `geolocator` to `^14.0.0` in pubspec.yaml. Use clean top-level `AndroidSettings`/`AppleSettings` imports from the main geolocator package.
- Full background location: add `ACCESS_BACKGROUND_LOCATION` permission (Android 10+) in addition to `FOREGROUND_SERVICE` and `FOREGROUND_SERVICE_LOCATION`. User accepts the Google Play Store extra review this triggers.
- Request "Always" permission (not just "When In Use"). Implement the two-step Apple always-permission flow. User accepts higher App Store review scrutiny.

### Claude's Discretion
- Notification text/icon for the Android foreground service notification
- Exact `distanceFilter` value (suggest 5m to reduce GPS jitter)
- Whether to show `showBackgroundLocationIndicator: true` on iOS (required by Apple guidelines — not discretionary, must be true)
</user_constraints>

---

## Summary

The `geolocator` package ships `AndroidSettings` and `AppleSettings` as top-level exports from `^14.0.0`. The only code change in `navigation_screen.dart` is replacing the bare `Geolocator.getPositionStream()` call with one that passes a `LocationSettings` built by a small platform-detecting helper. The permission-request upgrade (WhenInUse → Always) belongs in `navigation_launch_util.dart`, which already owns the permission-check flow.

Flutter SDK 3.41.9 is confirmed on this machine — the 14.x gate passes.

**Primary recommendation:** Bump geolocator to `^14.0.0`, add four manifest entries (three Android permissions + one iOS plist key + UIBackgroundModes), enable Xcode Background Modes capability, then wire `_buildLocationSettings()` into `initState`.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Flutter SDK >= 3.29.0 | geolocator ^14.0.0 | Yes | 3.41.9 | N/A — gate already passed |
| Xcode (macOS) | iOS Background Modes capability | Required human action | — | — |

---

## File 1 — pubspec.yaml

**Exact line to change:**

```yaml
# Before (line 32 in current file)
  geolocator: ^13.0.2

# After
  geolocator: ^14.0.0
```

After editing: `flutter pub upgrade geolocator`

**Verified:** geolocator 14.0.3 is the current latest on pub.dev. [CITED: pub.dev/packages/geolocator]
**Resolved current version:** 13.0.4 (from pubspec.lock). Bumping the constraint to `^14.0.0` will resolve to 14.0.3.

---

## File 2 — AndroidManifest.xml

**Current state:** Has `ACCESS_FINE_LOCATION`, `ACCESS_COARSE_LOCATION`, `POST_NOTIFICATIONS`, `VIBRATE`. Missing all foreground-service and background-location entries.

**Exact diff — add these four lines inside `<manifest>` before `<application>`:**

```xml
<!-- Add after existing <uses-permission> lines, before <application> -->
<uses-permission android:name="android.permission.FOREGROUND_SERVICE" />
<uses-permission android:name="android.permission.FOREGROUND_SERVICE_LOCATION" />
<uses-permission android:name="android.permission.ACCESS_BACKGROUND_LOCATION" />
```

**Resulting block (full permissions section):**

```xml
<manifest xmlns:android="http://schemas.android.com/apk/res/android">
    <uses-permission android:name="android.permission.ACCESS_FINE_LOCATION" />
    <uses-permission android:name="android.permission.ACCESS_COARSE_LOCATION" />
    <uses-permission android:name="android.permission.POST_NOTIFICATIONS" />
    <uses-permission android:name="android.permission.VIBRATE" />
    <uses-permission android:name="android.permission.FOREGROUND_SERVICE" />
    <uses-permission android:name="android.permission.FOREGROUND_SERVICE_LOCATION" />
    <uses-permission android:name="android.permission.ACCESS_BACKGROUND_LOCATION" />
    <application
        ...
```

**Does the plugin handle the foreground service declaration?**
Yes — `geolocator_android`'s own `AndroidManifest.xml` declares:

```xml
<service
    android:enabled="true"
    android:exported="false"
    android:foregroundServiceType="location"
    android:name=".GeolocatorLocationService" />
```

This is merged into the app manifest at build time via Android's manifest merger. The app does **not** need to re-declare `GeolocatorLocationService`. The only entries the app manifest must add are the three `<uses-permission>` lines above. [CITED: github.com/Baseflow/flutter-geolocator — geolocator_android/android/src/main/AndroidManifest.xml]

**Why `FOREGROUND_SERVICE` is needed:** Android requires this base permission to start any foreground service. The plugin's manifest declares the service but not this permission — the app must declare it.

**Why `FOREGROUND_SERVICE_LOCATION` is needed:** Android 14 (API 34) requires this specific permission for location-type foreground services. Without it, `startForeground()` throws on API 34+. [CITED: developer.android.com/about/versions/14/changes/fgs-types-required]

**Why `ACCESS_BACKGROUND_LOCATION` is needed:** Android 10+ (API 29) restricts background location unless this permission is declared. Without it, the foreground service loses GPS when the screen locks on Android 10+. This permission triggers a Google Play Store policy review (user has accepted this). [CITED: pub.dev/packages/geolocator — Android section]

---

## File 3 — Info.plist

**Current state:** Has `NSLocationWhenInUseUsageDescription` only. Missing the Always description and background modes.

**Exact diff — add these two entries inside the root `<dict>` (place after the existing NSLocationWhenInUseUsageDescription block):**

```xml
<!-- Add after the existing NSLocationWhenInUseUsageDescription key/string pair -->
<key>NSLocationAlwaysAndWhenInUseUsageDescription</key>
<string>Wanderer needs your location at all times to continue navigation when the screen locks.</string>

<key>UIBackgroundModes</key>
<array>
    <string>location</string>
</array>
```

**Full resulting plist top section (the two new entries shown in context):**

```xml
<key>NSLocationWhenInUseUsageDescription</key>
<string>Your location is shown on the map to help you navigate trails.</string>
<key>NSLocationAlwaysAndWhenInUseUsageDescription</key>
<string>Wanderer needs your location at all times to continue navigation when the screen locks.</string>
<key>UIBackgroundModes</key>
<array>
    <string>location</string>
</array>
```

**Notes:**
- `NSLocationAlwaysAndWhenInUseUsageDescription` is the correct key for iOS 11+ (replaces the deprecated `NSLocationAlwaysUsageDescription`). [CITED: pub.dev/packages/geolocator — iOS section]
- `UIBackgroundModes` with `location` is required for iOS 16+ to receive location in background. [CITED: pub.dev/packages/geolocator — iOS section]
- The description text is discretionary — adjust wording as needed.

---

## File 4 — navigation_screen.dart

**What changes:** Replace the bare `Geolocator.getPositionStream()` call in `initState` with one that passes `_buildLocationSettings()`.

**New imports to add** (at top of file, after existing geolocator import):

```dart
import 'dart:io' show Platform;
```

**New helper method** — add inside `_NavigationScreenState` class, before `initState`:

```dart
// Source: pub.dev/documentation/geolocator/latest — AndroidSettings, AppleSettings
LocationSettings _buildLocationSettings() {
  if (Platform.isAndroid) {
    return AndroidSettings(
      accuracy: LocationAccuracy.high,
      distanceFilter: 5,
      intervalDuration: const Duration(seconds: 1),
      foregroundNotificationConfig: const ForegroundNotificationConfig(
        notificationTitle: 'Wanderer Navigation',
        notificationText: 'Your location is being tracked for turn-by-turn navigation.',
        notificationChannelName: 'Navigation',
        enableWakeLock: true,
        setOngoing: true,
      ),
    );
  } else if (Platform.isIOS || Platform.isMacOS) {
    return AppleSettings(
      accuracy: LocationAccuracy.high,
      distanceFilter: 5,
      allowBackgroundLocationUpdates: true,
      showBackgroundLocationIndicator: true,
      pauseLocationUpdatesAutomatically: false,
    );
  }
  // Web / other platforms — minimal settings
  return const LocationSettings(
    accuracy: LocationAccuracy.high,
    distanceFilter: 5,
  );
}
```

**Modified `initState`** — replace line 82:

```dart
// Before:
_positionStream = Geolocator.getPositionStream().asBroadcastStream();

// After:
_positionStream = Geolocator.getPositionStream(
  locationSettings: _buildLocationSettings(),
).asBroadcastStream();
```

**Full updated `initState`:**

```dart
@override
void initState() {
  super.initState();

  _positionStream = Geolocator.getPositionStream(
    locationSettings: _buildLocationSettings(),
  ).asBroadcastStream();
  _sub = _positionStream.listen(
    (pos) {
      ref
          .read(navigationProvider(widget.response).notifier)
          .onPosition(LatLng(pos.latitude, pos.longitude));
      ref
          .read(navigationStatsProvider(widget.response).notifier)
          .onPosition(pos);
      if (_followEnabled) {
        _recenterTrigger.add(null);
      }
    },
    onError: (Object error) {
      debugPrint('NavigationScreen: GPS stream error — $error');
    },
  );
}
```

**API field reference (verified):** [CITED: pub.dev/documentation/geolocator/latest]

| Class | Field | Type | Default | Notes |
|-------|-------|------|---------|-------|
| `AndroidSettings` | `accuracy` | `LocationAccuracy` | `best` | Use `high` for navigation |
| `AndroidSettings` | `distanceFilter` | `int` | `0` | Meters; 5 reduces jitter |
| `AndroidSettings` | `intervalDuration` | `Duration?` | null | Minimum interval between updates |
| `AndroidSettings` | `foregroundNotificationConfig` | `ForegroundNotificationConfig?` | null | Required for background service |
| `ForegroundNotificationConfig` | `notificationTitle` | `String` | required | Shown in notification |
| `ForegroundNotificationConfig` | `notificationText` | `String` | required | Shown in notification |
| `ForegroundNotificationConfig` | `notificationChannelName` | `String` | `'Background Location'` | Android notification channel |
| `ForegroundNotificationConfig` | `enableWakeLock` | `bool` | `false` | Prevents CPU sleep |
| `ForegroundNotificationConfig` | `setOngoing` | `bool` | `false` | Makes notification non-dismissible |
| `ForegroundNotificationConfig` | `enableWifiLock` | `bool` | `false` | Optional; not needed for navigation |
| `AppleSettings` | `allowBackgroundLocationUpdates` | `bool` | `true` | Must be `true` for background |
| `AppleSettings` | `showBackgroundLocationIndicator` | `bool` | `false` | **Must be `true`** — App Store guideline |
| `AppleSettings` | `pauseLocationUpdatesAutomatically` | `bool` | `false` | Keep `false` for active navigation |
| `AppleSettings` | `distanceFilter` | `int` | `0` | Meters |

---

## File 5 — navigation_launch_util.dart (iOS Always Permission Flow)

**Current state:** Lines 113–125 request `WhenInUse` only via a single `Geolocator.requestPermission()` call. No upgrade to `Always`.

**What the Apple two-step flow means in practice:**

iOS 13+ no longer shows an "Always Allow" dialog on first request. Instead:
1. The app calls `requestPermission()` once when `NSLocationAlwaysAndWhenInUseUsageDescription` is present in Info.plist — iOS shows the standard "While Using / Once / Deny" prompt.
2. After the user grants `whileInUse`, the app calls `requestPermission()` a second time. iOS presents a second system prompt asking whether to upgrade to "Always Allow".

In geolocator, both calls use `Geolocator.requestPermission()`. The second call only shows the system upgrade dialog if `NSLocationAlwaysAndWhenInUseUsageDescription` is present in Info.plist (which we add in File 3). [CITED: pub.dev/packages/geolocator — iOS section; snippetsource.wordpress.com iOS 13 permission changes]

**Implementation note on WhenInUse sufficiency:** Apple's CoreLocation documentation confirms that if location tracking *starts* in the foreground (`WhenInUse`), updates continue in the background after the screen locks — even without `Always` permission — as long as `allowBackgroundLocationUpdates = true`. [CITED: developer.apple.com/forums/thread/748098] However, the user has decided to request `Always` for maximum reliability, so implement the two-step flow.

**Exact code change in `navigation_launch_util.dart`:**

Replace the current permission block (lines 113–125):

```dart
// BEFORE — single requestPermission call, no Always upgrade:
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
```

```dart
// AFTER — two-step Always flow:
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
// requestPermission() again. On iOS this triggers the system prompt
// asking "Change to Always Allow?" only when
// NSLocationAlwaysAndWhenInUseUsageDescription is present in Info.plist.
// On Android this call is a no-op (already granted fine/background).
// The result may still be whileInUse if the user declines the upgrade —
// navigation proceeds either way (background tracking still works via
// allowBackgroundLocationUpdates on iOS and the foreground service on Android).
if (permission == LocationPermission.whileInUse) {
  permission = await Geolocator.requestPermission();
  // Do not block navigation if Always is declined — proceed with whileInUse.
}
```

**Why not block on Always denial:** The navigation use case (user starts in foreground, screen locks) works correctly with `WhenInUse` + `allowBackgroundLocationUpdates: true` on iOS. Blocking navigation when the user declines the Always upgrade would break usability. The upgrade is best-effort.

---

## Xcode Manual Step

**What:** Enable "Location Updates" under Background Modes capability.

**Exact steps:**
1. Open `app/ios/Runner.xcworkspace` in Xcode (not the `.xcodeproj` file).
2. In the Project Navigator (left sidebar), click **Runner** (the blue project icon at the top).
3. In the main editor, select the **Runner** target (under TARGETS, not PROJECTS).
4. Click the **Signing & Capabilities** tab.
5. Click the **+ Capability** button (top left of the capabilities area).
6. Search for **Background Modes** and double-click it to add.
7. In the newly appeared Background Modes section, check **Location Updates**.

**Why this is required:** Without the Xcode capability enabled, the `UIBackgroundModes` key in Info.plist alone is insufficient — iOS silently ignores background location updates if the Xcode capability is not set. [CITED: pub.dev/packages/geolocator — iOS setup section]

**This step cannot be automated** via file edits. The capability is recorded in `Runner.xcodeproj/project.pbxproj` by Xcode. Editing that file manually is error-prone and not recommended.

---

## Common Pitfalls

### Pitfall 1: Missing FOREGROUND_SERVICE permission
**What goes wrong:** `startForeground()` throws `SecurityException` on API 28+ (Android 9+) if the base `FOREGROUND_SERVICE` permission is absent.
**Why it happens:** Developers add `FOREGROUND_SERVICE_LOCATION` (Android 14-specific) but forget the base `FOREGROUND_SERVICE` permission.
**How to avoid:** Declare both `FOREGROUND_SERVICE` and `FOREGROUND_SERVICE_LOCATION`.

### Pitfall 2: Xcode capability not enabled
**What goes wrong:** iOS silently delivers zero location events when backgrounded, even with `allowBackgroundLocationUpdates: true` and the Info.plist `UIBackgroundModes` key.
**Why it happens:** `UIBackgroundModes` in Info.plist and the Xcode capability are two separate things — both are required.
**How to avoid:** Complete the Xcode Signing & Capabilities step.

### Pitfall 3: `showBackgroundLocationIndicator: false` on iOS
**What goes wrong:** App Store submission rejection for collecting background location without the blue location indicator visible to the user.
**How to avoid:** Always set `showBackgroundLocationIndicator: true` in `AppleSettings`.

### Pitfall 4: OEM battery optimization kills the Android foreground service
**What goes wrong:** On Samsung/Xiaomi/OPPO devices with aggressive battery optimization, the foreground service is killed after ~10–30 minutes despite `enableWakeLock: true`.
**How to avoid:** Set `enableWakeLock: true` in `ForegroundNotificationConfig`. For further reliability, prompt users to exempt Wanderer from battery optimization in device settings. This is a device-level issue, not a code bug.

### Pitfall 5: distanceFilter = 0 causes excessive updates
**What goes wrong:** At `distanceFilter: 0` (default), every GPS sample triggers an update. On a trail with good signal this can be 1 Hz. `onPosition` calls are cheap but `setState` via `_recenterTrigger` repaints the map every cycle.
**How to avoid:** Use `distanceFilter: 5` (5 metres). A hiker travels ~1.4 m/s — 5 m gives one update every ~3–4 seconds, which is more than sufficient for navigation.

### Pitfall 6: `dart:io` Platform import missing
**What goes wrong:** `Platform.isAndroid` / `Platform.isIOS` are undefined without `import 'dart:io' show Platform;`.
**How to avoid:** Add the import. Note: `dart:io` is available in Flutter apps but not in pure Dart web targets — safe for this mobile-only screen.

---

## Implementation Order

1. Bump `geolocator: ^14.0.0` in pubspec.yaml; run `flutter pub upgrade geolocator`.
2. Add three `<uses-permission>` lines to `app/android/app/src/main/AndroidManifest.xml`.
3. Add `NSLocationAlwaysAndWhenInUseUsageDescription` and `UIBackgroundModes` to `app/ios/Runner/Info.plist`.
4. Open Xcode → Runner target → Signing & Capabilities → add Background Modes → check Location Updates.
5. Add `import 'dart:io' show Platform;` to `navigation_screen.dart`.
6. Add `_buildLocationSettings()` helper to `_NavigationScreenState`.
7. Update `initState` to pass `locationSettings: _buildLocationSettings()`.
8. Update the permission block in `navigation_launch_util.dart` with the two-step Always flow.
9. Test on Android device: lock screen mid-navigation, verify position events continue and notification is visible.
10. Test on iOS device: lock screen mid-navigation, verify blue location pill appears and events continue.

---

## Sources

### Primary (HIGH confidence)
- [pub.dev/packages/geolocator](https://pub.dev/packages/geolocator) — version 14.0.3, AndroidSettings/AppleSettings/ForegroundNotificationConfig API
- [pub.dev/documentation/geolocator/latest/geolocator/AndroidSettings-class.html](https://pub.dev/documentation/geolocator/latest/geolocator/AndroidSettings-class.html) — constructor parameters
- [pub.dev/documentation/geolocator/latest/geolocator/AppleSettings-class.html](https://pub.dev/documentation/geolocator/latest/geolocator/AppleSettings-class.html) — constructor parameters
- [pub.dev/documentation/geolocator/latest/geolocator/ForegroundNotificationConfig-class.html](https://pub.dev/documentation/geolocator/latest/geolocator/ForegroundNotificationConfig-class.html) — constructor parameters
- [github.com/Baseflow/flutter-geolocator — geolocator_android AndroidManifest.xml](https://github.com/Baseflow/flutter-geolocator/blob/main/geolocator_android/android/src/main/AndroidManifest.xml) — confirms plugin handles foregroundServiceType via manifest merge
- [developer.apple.com/forums/thread/748098](https://developer.apple.com/forums/thread/748098) — WhenInUse vs Always for screen-locked updates

### Secondary (MEDIUM confidence)
- [pub.dev/packages/geolocator/changelog](https://pub.dev/packages/geolocator/changelog) — 14.0.0 breaking change (Flutter >= 3.29.0)
- [developer.android.com/about/versions/14/changes/fgs-types-required](https://developer.android.com/about/versions/14/changes/fgs-types-required) — FOREGROUND_SERVICE_LOCATION requirement
- [snippetsource.wordpress.com — iOS 13 permission changes](https://snippetsource.wordpress.com/2019/08/28/ios-13-location-permission-changes/) — two-step Always flow
- [developer.apple.com/forums/thread/117256](https://developer.apple.com/forums/thread/117256) — iOS requestWhenInUse → requestAlways two-step

### Tertiary (LOW confidence / ASSUMED)
- None. All implementation details are verified from official docs or plugin source.
