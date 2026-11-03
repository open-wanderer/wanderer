---
created: 2026-11-03T00:00:00.000Z
title: Add "New trail" and "Library" home-screen app shortcuts (Android shortcuts + iOS Quick Actions)
area: app
priority: medium
files:
  - app/pubspec.yaml
  - app/lib/main.dart
  - app/lib/provider/router_provider.dart
  - app/lib/i18n/app_en.arb
  - app/android/app/src/main/res/
  - app/ios/Runner/Assets.xcassets/
---

## Goal

Long-press on the launcher icon offers two shortcuts, each pinnable to the home screen as its own
icon (the komoot pattern). Same set on iOS via Home Screen Quick Actions.

## Decisions (from /gsd-explore, 2026-11-03)

- **Two static shortcuts only.** No per-trail/per-list pinning, no separate plan/record shortcuts.
- **New trail** -> `/trail/create` (`TrailSourceSelectScreen`). Planner, recorder and import stay
  behind that screen; the source choice is not duplicated in the launcher.
- **Library** -> `/library` (reuses the tab's `library` string; renamed from "My trails" after device test).
- **Active session wins** (recording or navigation, both shortcuts). Implemented as: an
  `ActiveNavigationEntity` row exists -> the shortcut does nothing; the session screen cannot be left
  while running, so foregrounding shows it, and `_maybeResume` reopens it on a cold start.
- Originally scoped as **Active recording wins.** If a recording is in progress, New trail returns to that recording
  instead of opening the source selector. A second recorder start is a state we don't want to
  guard against. Resume plumbing already exists: `ActiveNavigationEntity` rows and
  `_maybeResume` / `_pushRecordingResume` in `main.dart`.
- **Plugin:** `quick_actions` (one code path for Android dynamic shortcuts + iOS Quick Actions;
  Android launchers make them pinnable for free).

## Approach

Mirror the inbound-share flow (`_handleShare`, `main.dart`):

- Cold start: router is on the `/` splash; store a pending launch target and let the redirect land
  on it instead of `/map` once auth settles (as it does for `pendingTrailImportProvider`).
- Signed out / no server: same redirect path -> login/welcome first, then the target.
- App already running: push/go to the target; for New trail check for an active recording first.
- Release the splash reveal hold when a session is known, as the share path does.

## Checklist

- [x] Add `quick_actions` to `app/pubspec.yaml`
- [x] Register the two shortcuts at startup; re-register on locale change
- [x] ARB keys for both labels in `app_en.arb` (Crowdin does the rest)
- [x] Icons: Android drawables for each shortcut; iOS template images in the asset catalog
- [x] Pending launch target consumed by the router redirect (cold start + signed out)
- [x] New trail -> active recording resume when one exists
- [x] Unit tests: type -> route mapping, icon resources + shrinker keep rule, pending target taken once
  (`test/services/app_shortcuts_test.dart`). The redirect and `_handleShortcut` are not covered:
  `Router.build()` cannot be mounted in tests (see `router_inbound_uri_test.dart`).
- [x] Manual check: cold start, warm start, signed out, mid-recording, pinned icon on Android
