# Quick Task 260615-mxk: Implement Background Navigation - Context

**Gathered:** 2026-06-15
**Status:** Ready for planning

<domain>
## Task Boundary

Implement background navigation so location tracking continues when the phone screen locks. The geolocator package (already in use) is the chosen solution — no new packages required.

</domain>

<decisions>
## Implementation Decisions

### Flutter SDK Version
- Flutter SDK >= 3.29.0 confirmed → bump `geolocator` to `^14.0.0` in pubspec.yaml. Use clean top-level `AndroidSettings`/`AppleSettings` imports from the main geolocator package.

### Android Lock-Screen Depth
- Full background location: add `ACCESS_BACKGROUND_LOCATION` permission (Android 10+) in addition to `FOREGROUND_SERVICE` and `FOREGROUND_SERVICE_LOCATION`. User accepts the Google Play Store extra review this triggers.

### iOS Permission Level
- Request "Always" permission (not just "When In Use"). Implement the two-step Apple always-permission flow. User accepts higher App Store review scrutiny.

### Claude's Discretion
- Notification text/icon for the Android foreground service notification
- Exact `distanceFilter` value (suggest 5m to reduce GPS jitter)
- Whether to show `showBackgroundLocationIndicator: true` on iOS (required by Apple guidelines — not discretionary, must be true)

</decisions>

<specifics>
## Specific Ideas

- Prior spike (260615-ktn-FINDINGS.md) provides the exact implementation order and file change table — executor should use it as the primary reference
- `_buildLocationSettings()` helper goes in `navigation_screen.dart` alongside the existing stream setup
- iOS two-step "Always" flow: request WhenInUse first, then prompt upgrade to Always after user grants it

</specifics>
