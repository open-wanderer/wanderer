# Background Geolocation: Implementation Spike Findings

**Produced:** 2026-06-15
**Source:** 260615-ktn-RESEARCH.md
**Status:** Ready for executor

---

## Decision

Upgrade `geolocator` from `^13.0.2` to `^14.0.0` and configure platform-aware `LocationSettings`: use `AndroidSettings` with `ForegroundNotificationConfig` on Android and `AppleSettings(allowBackgroundLocationUpdates: true, showBackgroundLocationIndicator: true)` on iOS. Rationale: free MIT/BSD license, minimal diff to the existing `NavigationScreen` stream setup, and no additional packages required.

**Hard prerequisite:** Flutter SDK must be >= 3.29.0. Run `flutter --version` before touching `pubspec.yaml` — this is the first gate.

---

## Rejected Alternatives

| Option | Why Rejected |
|--------|--------------|
| `flutter_background_geolocation` (Transistor Software) | $500/app Android production license; over-engineered for v1 trail navigation |
| `flutter_foreground_task` + existing `geolocator` | Extra package with no benefit over using `ForegroundNotificationConfig` directly via Option A |
| `background_fetch` | 15-minute minimum fire interval — system-controlled; unsuitable for real-time navigation |

---

## Flutter SDK Gate

Run `flutter --version` and check the reported SDK version:

- **If >= 3.29.0:** Bump `geolocator: ^14.0.0` in `app/pubspec.yaml`, then run `flutter pub upgrade geolocator`.
- **If < 3.29.0:** Do not bump the main `geolocator` package. Instead, import `AndroidSettings` / `AppleSettings` directly from `geolocator_android` and `geolocator_apple` sub-packages at their current 13.x versions — the underlying plugins already ship `ForegroundNotificationConfig` and background-capable Apple settings. See the `geolocator_android` and `geolocator_apple` pub.dev pages for direct import syntax.

---

## Required File Changes

| File | Change | Notes |
|------|--------|-------|
| `app/pubspec.yaml` | Bump `geolocator: ^14.0.0` (if SDK >= 3.29.0) | Run `flutter pub upgrade geolocator` after edit |
| `app/android/app/src/main/AndroidManifest.xml` | Add `FOREGROUND_SERVICE`, `FOREGROUND_SERVICE_LOCATION` (Android 14+, API 34), and optionally `ACCESS_BACKGROUND_LOCATION` (Android 10+) | Omit `ACCESS_BACKGROUND_LOCATION` for v1 if screen-on navigation is acceptable; adding it triggers a Google Play Store review |
| `app/ios/Runner/Info.plist` | Add `NSLocationAlwaysAndWhenInUseUsageDescription` string and `UIBackgroundModes` array containing `location` | Required for iOS background location updates |
| `app/ios/Runner.xcodeproj` | Enable "Location Updates" under Background Modes capability | Cannot be done via file edit; requires a human action in Xcode (Signing & Capabilities tab) |
| `app/lib/routes/navigation_screen.dart` | Replace bare `Geolocator.getPositionStream()` with `_buildLocationSettings()` helper; update `initState` call | See the Integration Pattern section of RESEARCH.md for the exact code snippet |
| `app/lib/util/navigation_launch_util.dart` | Optional: upgrade iOS permission request to "Always" via Apple's two-step flow | Defer for v1 unless lock-screen navigation (screen fully off) is required; document the decision |

---

## Pitfalls to Address Before Shipping

1. **iOS stream suspended without `allowBackgroundLocationUpdates: true`** — `Geolocator.getPositionStream()` with no `AppleSettings` delivers zero events when the app is backgrounded. Navigation silently freezes.
2. **`AndroidSettings` not cleanly exposed at geolocator 13.x** — The class exists in `geolocator_android` 4.6.2 but may not be a clean top-level import from the main `geolocator` package at 13.x. Upgrading to `^14.0.0` is the straightforward path.
3. **Android Doze / OEM battery optimization kills foreground service** — On Samsung, Xiaomi, and other aggressive-battery OEMs, `enableWakeLock: true` in `ForegroundNotificationConfig` helps but does not guarantee survival. Instruct users to exempt Wanderer from battery optimization.
4. **`showBackgroundLocationIndicator: true` required in `AppleSettings`** — Apple requires the blue location pill to be visible when collecting background location. Omitting it violates App Store guidelines and risks rejection.
5. **geolocator 14.0.0 requires Flutter >= 3.29.0** — Bumping the version without verifying the SDK will cause a resolution failure. Check `flutter --version` first.

---

## Open Assumptions (Must Verify During Implementation)

| ID | Assumption | Risk if Wrong | How to Verify |
|----|------------|---------------|---------------|
| A1 | Developer's Flutter SDK is >= 3.29.0 (required for geolocator 14.x) | Must stay on 13.x and use `geolocator_android`/`geolocator_apple` direct imports | Run `flutter --version` |
| A2 | `ACCESS_BACKGROUND_LOCATION` is not required if the foreground service is active and the user has the screen on | Would need to add this permission for full lock-screen navigation on Android 10+ | Test on Android 10+ physical device with screen timed out while stream is active |
| A3 | `geolocator_android` plugin manifest merge at 4.6.2+ handles `android:foregroundServiceType="location"` internally | Would need a manual `<service>` declaration in app manifest if not handled | Run `./gradlew mergeDebugManifest` and inspect the merged output for correct `foregroundServiceType` |
| A4 | `AppleSettings(allowBackgroundLocationUpdates: true)` is sufficient for screen-locked navigation without requesting "Always" permission | Would need to upgrade to "Always" permission request in `navigation_launch_util.dart` | Test on a real iOS device: lock screen while navigation is active; confirm position events continue |
| A5 | OEM battery optimization does not kill the foreground service in typical trail navigation sessions (1-6 hours) | Users on aggressive-battery OEMs would need manual exemption; consider in-app prompt | Test on Samsung/Xiaomi physical device for 30+ minutes with screen off |

---

## Suggested Implementation Order

1. Run `flutter --version` — confirm SDK >= 3.29.0 (gates the entire upgrade path).
2. Bump `geolocator` in `app/pubspec.yaml` per SDK gate result; run `flutter pub upgrade geolocator`.
3. Add Android manifest permissions — start without `ACCESS_BACKGROUND_LOCATION`; add only if full lock-screen is required after testing (addresses A2).
4. Add iOS plist keys (`NSLocationAlwaysAndWhenInUseUsageDescription` + `UIBackgroundModes: [location]`).
5. Open Xcode, enable Background Modes > Location Updates capability in Signing & Capabilities.
6. Implement `_buildLocationSettings()` in `navigation_screen.dart` using the Integration Pattern from RESEARCH.md.
7. Test on a real Android device with screen-off while stream is active — confirm position events continue.
8. Test on a real iOS device with screen-off — confirm events continue and blue pill indicator appears.
9. Run `./gradlew mergeDebugManifest` and verify correct `foregroundServiceType` (addresses A3).
10. Decide on iOS "Always" permission (addresses A4) — if needed, update `navigation_launch_util.dart` with the two-step Apple always-permission flow.

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
