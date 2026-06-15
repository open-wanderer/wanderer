# Research: Flutter Background Geolocation for Navigation

**Researched:** 2026-06-15
**Domain:** Flutter background location tracking for trail navigation
**Confidence:** HIGH (package docs verified, codebase inspected)

---

## Summary

The existing `NavigationScreen` uses `Geolocator.getPositionStream()` without any background configuration. When the phone locks, the stream is suspended on iOS and degraded on Android, silently stopping maneuver advances and breadcrumb recording. Two viable paths exist: (1) upgrade `geolocator` from `^13.0.2` to `^14.0.0` and use its built-in `ForegroundNotificationConfig` on Android + iOS background-modes capability — no new packages, no license cost; or (2) add `flutter_background_geolocation` from Transistor Software, which costs $500 per production app for Android RELEASE builds.

**Primary recommendation:** Stay on `geolocator` and upgrade to 14.x. It already supports Android foreground service via `ForegroundNotificationConfig` and iOS background location via `AppleSettings`. Zero license cost, minimal diff to the existing stream setup.

---

## Package Evaluation

### Option A: Upgrade `geolocator` to 14.x (recommended)

**Current install:** `geolocator 13.0.4` / `geolocator_android 4.6.2`

`ForegroundNotificationConfig` was introduced in `geolocator_android 3.1.0` and is **already present in the installed 4.6.2**. However, `geolocator ^13.0.2` constrains to 13.x on the pub semver resolver. The constraint must be bumped to `^14.0.0` to pick up Android 14 manifest fixes (version 14.0.1 adds `FOREGROUND_SERVICE_LOCATION` docs) and iOS 16+ `UIBackgroundModes` support.

**Breaking change in 14.0.0:** Requires Flutter 3.29.0+. Project is on Flutter 3.11.5+ — **must verify exact SDK version before upgrading.** [ASSUMED — exact Flutter SDK on developer's machine not verified in this session]

**License:** MIT / BSD — free for production. [VERIFIED: pub.dev]

### Option B: `flutter_background_geolocation` (Transistor Software)

**Version:** 5.2.1 (published 2026-06-13) [VERIFIED: pub.dev]

**Licensing:** Apache-2.0 on pub.dev, but the underlying native SDKs require a paid license for Android RELEASE builds. Debug builds are fully functional with no license. **Production cost: ~$500 per app.** [CITED: transistorsoft.com shop]

**What it adds over geolocator:**
- Motion-detection (accelerometer/gyroscope) to shut off GPS when stationary — significant battery saving on long hikes
- Geofencing built-in
- Headless execution after app termination

**Verdict for this project:** Over-engineered for v1 trail navigation. The $500 cost and setup complexity are not justified when geolocator already handles the use case.

### Option C: `flutter_foreground_task` + existing `geolocator`

Runs geolocator inside a separate foreground isolate. More setup than Option A. Adds a second package. No benefit over just using `ForegroundNotificationConfig` directly. **Skip.**

### Option D: `background_fetch`

Fires callbacks every ~15 minutes minimum — system-controlled, cannot be increased. **Unsuitable for real-time navigation.** [VERIFIED: pub.dev]

---

## Platform Requirements

### Android

**Current state:** `ACCESS_FINE_LOCATION` + `ACCESS_COARSE_LOCATION` already in `AndroidManifest.xml`. Missing:

```xml
<!-- Required for any foreground service (Android 9+) -->
<uses-permission android:name="android.permission.FOREGROUND_SERVICE" />

<!-- Required to declare foreground service type = location (Android 14+, API 34) -->
<uses-permission android:name="android.permission.FOREGROUND_SERVICE_LOCATION" />

<!-- Required for "Always" background location (Android 10+) -->
<uses-permission android:name="android.permission.ACCESS_BACKGROUND_LOCATION" />
```

**Note on `ACCESS_BACKGROUND_LOCATION`:** Google Play Store reviews apps that request this permission carefully. If navigation is only active while the user is interacting with the phone (screen on), this permission may not be necessary — the foreground service keeps the stream alive even without it. Trail navigation while the screen is on-but-locked (display timeout) does need the foreground service; full background (screen off, app backgrounded) needs `ACCESS_BACKGROUND_LOCATION` on Android 10+. [ASSUMED — exact behavior depends on Android version and OEM; test required]

**Foreground service type (Android 14+):** The `geolocator_android` plugin registers its own service in its manifest. As of geolocator_android 4.6.2, this should be handled by the plugin's manifest merge. Verify after upgrade with `./gradlew mergeDebugManifest`. [ASSUMED — plugin manifest merge not verified in this session]

### iOS

**Current state:** `Info.plist` only has `NSLocationWhenInUseUsageDescription`. Missing:

```xml
<!-- Required for "Always" permission prompt -->
<key>NSLocationAlwaysAndWhenInUseUsageDescription</key>
<string>Wanderer needs your location in the background to continue turn-by-turn navigation when the screen locks.</string>

<!-- iOS 16+: enables background location updates -->
<key>UIBackgroundModes</key>
<array>
  <string>location</string>
</array>
```

**Xcode Signing & Capabilities:** Must add "Background Modes" capability and check "Location Updates" checkbox. This writes to `Runner.entitlements` / project settings — cannot be done via file edit alone, requires Xcode. [CITED: Apple developer docs, geolocator readme]

**Apple review note:** Apple requires a detailed justification for always-on location when submitting to the App Store. Trail navigation is an accepted use case, but the review note must be filled out in App Store Connect. [CITED: geolocator readme]

**iOS `AppleSettings`:** Pass `allowBackgroundLocationUpdates: true` when constructing the position stream (see Integration Pattern below).

---

## Integration Pattern

The existing stream in `NavigationScreen._NavigationScreenState.initState`:

```dart
// Current (foreground only)
_positionStream = Geolocator.getPositionStream().asBroadcastStream();
```

**Replace with platform-aware settings:**

```dart
// Background-capable stream
LocationSettings _buildLocationSettings() {
  if (defaultTargetPlatform == TargetPlatform.android) {
    return AndroidSettings(
      accuracy: LocationAccuracy.high,
      distanceFilter: 5, // meters — avoid GPS noise on stationary hiker
      foregroundNotificationConfig: const ForegroundNotificationConfig(
        notificationTitle: 'Wanderer Navigation',
        notificationText: 'Tracking your position on trail',
        enableWakeLock: true,
      ),
    );
  } else if (defaultTargetPlatform == TargetPlatform.iOS ||
             defaultTargetPlatform == TargetPlatform.macOS) {
    return AppleSettings(
      accuracy: LocationAccuracy.high,
      activityType: ActivityType.fitness,
      distanceFilter: 5,
      pauseLocationUpdatesAutomatically: false,
      allowBackgroundLocationUpdates: true,
      showBackgroundLocationIndicator: true, // blue pill indicator — required by Apple
    );
  }
  return const LocationSettings(accuracy: LocationAccuracy.high);
}

@override
void initState() {
  super.initState();
  _positionStream = Geolocator.getPositionStream(
    locationSettings: _buildLocationSettings(),
  ).asBroadcastStream();
  _sub = _positionStream.listen(/* existing handler unchanged */);
}
```

**No changes needed in `NavigationProvider` or `NavigationStatsProvider`.** Both consume `LatLng` / `Position` objects via `onPosition()` — the position source is opaque to them.

**`navigation_launch_util.dart`:** The permission check already calls `Geolocator.requestPermission()`. For iOS background access, this must be upgraded to request "Always" permission, which requires calling `Geolocator.requestPermission()` after the user has already granted "When In Use" — Apple's two-step always-permission flow. Alternatively, accept that users keep the screen on during navigation and only use `whenInUse` + foreground service on Android. [ASSUMED — UX decision for the team]

---

## Licensing Summary

| Package | Production License | Cost |
|---------|-------------------|------|
| `geolocator` 14.x | MIT/BSD | Free |
| `flutter_background_geolocation` | Apache-2.0 (wrapper) + proprietary native SDK | ~$500/app (Android RELEASE) |
| `background_fetch` | MIT | Free (unsuitable) |

---

## Common Pitfalls

### Pitfall 1: Stream suspended on iOS without `allowBackgroundLocationUpdates: true`

**What goes wrong:** `Geolocator.getPositionStream()` with no `AppleSettings` delivers zero events when the app is backgrounded. Navigation silently freezes.

**How to avoid:** Always pass `AppleSettings(allowBackgroundLocationUpdates: true)` when constructing the stream for navigation. Gate this behind `defaultTargetPlatform == TargetPlatform.iOS`.

### Pitfall 2: `ForegroundNotificationConfig` available in geolocator_android 4.6.2 but geolocator 13.x API doesn't expose `AndroidSettings` cleanly

**What goes wrong:** The class exists in the underlying plugin but the main `geolocator` package's API surface at 13.x may not fully expose `AndroidSettings`. Upgrading to `geolocator ^14.0.0` is the clean path.

**Warning signs:** `AndroidSettings` not found as a top-level import; `geolocator_android` exports it but requires a direct platform import.

### Pitfall 3: Android Doze mode kills foreground service after extended inactivity

**What goes wrong:** On some Android OEMs (Samsung, Xiaomi), aggressive battery optimization kills foreground services even with `enableWakeLock: true`.

**How to avoid:** `enableWakeLock: true` in `ForegroundNotificationConfig` helps. Instruct users to disable battery optimization for Wanderer in device settings. There is no programmatic API to guarantee this across all OEMs. [ASSUMED — OEM behavior varies; not verified on specific devices]

### Pitfall 4: iOS `showBackgroundLocationIndicator` omission causes App Store rejection

**What goes wrong:** Apple requires the blue location pill to be shown when collecting background location. Omitting `showBackgroundLocationIndicator: true` in `AppleSettings` violates App Store guidelines.

**How to avoid:** Always pass `showBackgroundLocationIndicator: true` in `AppleSettings` for navigation use cases.

### Pitfall 5: `geolocator` 14.0.0 requires Flutter 3.29.0

**What goes wrong:** Bumping `geolocator` to `^14.0.0` in `pubspec.yaml` will fail if the Flutter SDK is below 3.29.0. The project's `pubspec.yaml` specifies `sdk: ^3.11.5` — need to check the actual SDK installed.

**How to avoid:** Run `flutter --version` before bumping. If SDK < 3.29.0, stay on `geolocator ^13.0.2` and use `AndroidSettings` / `AppleSettings` directly from `geolocator_android` / `geolocator_apple` packages (they support background at 13.x on the underlying plugin level).

---

## Existing Code Touchpoints

| File | Change Needed |
|------|---------------|
| `app/pubspec.yaml` | Bump `geolocator: ^13.0.2` → `^14.0.0` (if Flutter SDK >= 3.29.0) |
| `app/android/app/src/main/AndroidManifest.xml` | Add `FOREGROUND_SERVICE`, `FOREGROUND_SERVICE_LOCATION`, optionally `ACCESS_BACKGROUND_LOCATION` |
| `app/ios/Runner/Info.plist` | Add `NSLocationAlwaysAndWhenInUseUsageDescription` + `UIBackgroundModes: [location]` |
| `app/ios/Runner.xcodeproj` | Enable "Location Updates" in Background Modes capability (Xcode only) |
| `app/lib/routes/navigation_screen.dart` | Replace bare `Geolocator.getPositionStream()` with `_buildLocationSettings()` pattern (lines 82-83) |
| `app/lib/util/navigation_launch_util.dart` | Optionally upgrade permission request to "always" for iOS (lines 113-125) — v1 may skip this |

**`NavigationProvider` and `NavigationStatsProvider`: no changes needed.** Both are already stream-agnostic — they receive `LatLng` / `Position` and do not care about the stream source.

---

## Assumptions Log

| # | Claim | Risk if Wrong |
|---|-------|---------------|
| A1 | Flutter SDK on developer machine is >= 3.29.0 (required for geolocator 14.x) | Must stay on 13.x; use `geolocator_android` direct import instead |
| A2 | `ACCESS_BACKGROUND_LOCATION` may not be required if foreground service is active and user has screen-on use | Would need to add this permission for full lock-screen navigation on Android 10+ |
| A3 | geolocator_android plugin manifest merge handles `android:foregroundServiceType="location"` internally at 4.6.2+ | Would need manual `<service>` declaration in app manifest if not handled |
| A4 | iOS background location via `allowBackgroundLocationUpdates: true` is sufficient for screen-locked navigation without "Always" permission | Would need to request "Always" permission for full lock-screen support |
| A5 | OEM battery optimization does not kill the foreground service in typical trail navigation sessions | Users on aggressive-battery OEMs would need manual battery optimization exemption |

---

## Sources

### Primary (HIGH confidence)
- [pub.dev/packages/geolocator](https://pub.dev/packages/geolocator) — version, license, background API
- [pub.dev/packages/geolocator_android changelog](https://pub.dev/packages/geolocator_android/changelog) — ForegroundNotificationConfig version history
- [pub.dev/packages/flutter_background_geolocation](https://pub.dev/packages/flutter_background_geolocation) — version, license model
- [pub.dev/packages/background_fetch](https://pub.dev/packages/background_fetch) — 15-minute minimum, unsuitable for navigation

### Secondary (MEDIUM confidence)
- [docs.transistorsoft.com/flutter/setup](https://docs.transistorsoft.com/flutter/setup/) — iOS Info.plist keys, Android meta-data
- [developer.android.com/about/versions/14/changes/fgs-types-required](https://developer.android.com/about/versions/14/changes/fgs-types-required) — Android 14 foreground service type requirements
- [geolocator ForegroundNotificationConfig API docs](https://pub.dev/documentation/geolocator/latest/geolocator/ForegroundNotificationConfig-class.html)

### Tertiary (LOW confidence / ASSUMED)
- Medium articles on geolocator + Riverpod integration patterns
- transistorsoft.com pricing (~$500/app for Android production)
