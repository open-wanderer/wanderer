---
phase: 08-account-profile
plan: "01"
subsystem: flutter-app
tags: [dependencies, localization, riverpod, auth, image_picker, tdd]
dependency_graph:
  requires: []
  provides:
    - image_picker ^1.2.2 dependency in app/pubspec.yaml
    - NSPhotoLibraryUsageDescription in app/ios/Runner/Info.plist
    - AppLocalizations.account getter returning "Account"
    - Auth.refresh() public method on the Auth Riverpod notifier
  affects:
    - app/lib/provider/auth_provider.dart (new public method)
    - app/lib/i18n/app_localizations.dart (new abstract getter)
tech_stack:
  added:
    - image_picker ^1.2.2 (flutter.dev verified publisher)
  patterns:
    - TDD RED/GREEN for Auth.refresh() compile-time contract test
    - AsyncValue.guard() wrapping in Riverpod notifier method
    - flutter gen-l10n codegen for ARB key addition
key_files:
  created:
    - app/test/provider/auth_provider_refresh_test.dart
  modified:
    - app/pubspec.yaml
    - app/pubspec.lock
    - app/ios/Runner/Info.plist
    - app/lib/i18n/app_en.arb
    - app/lib/i18n/app_localizations.dart
    - app/lib/i18n/app_localizations_en.dart
    - app/lib/i18n/app_localizations_cs.dart
    - app/lib/i18n/app_localizations_de.dart
    - app/lib/i18n/app_localizations_es.dart
    - app/lib/i18n/app_localizations_eu.dart
    - app/lib/i18n/app_localizations_fr.dart
    - app/lib/i18n/app_localizations_hu.dart
    - app/lib/i18n/app_localizations_it.dart
    - app/lib/i18n/app_localizations_nl.dart
    - app/lib/i18n/app_localizations_no.dart
    - app/lib/i18n/app_localizations_pl.dart
    - app/lib/i18n/app_localizations_pt.dart
    - app/lib/i18n/app_localizations_ru.dart
    - app/lib/i18n/app_localizations_zh.dart
    - app/lib/provider/auth_provider.dart
    - app/lib/provider/auth_provider.g.dart
decisions:
  - "Auth.refresh() delegates to existing private _updateUserEntity(id) inside AsyncValue.guard to avoid toEntity() throwing without expand.actor (Pitfall 3)"
  - "TDD compile-time tear-off test used for refresh() contract: references Auth.refresh via notifier.refresh tear-off so missing method causes compile error (RED gate)"
  - "NSPhotoLibraryUsageDescription inserted before NSLocationWhenInUseUsageDescription, matching existing tab indentation in Info.plist"
  - "account ARB key inserted alphabetically between account_delete_confirm and account_privacy with no @-metadata block (matches all other entries in the template)"
metrics:
  duration_minutes: 8
  completed_date: "2026-06-20"
  tasks_completed: 3
  files_changed: 21
---

# Phase 08 Plan 01: Foundation Contracts — image_picker, iOS permission, l10n account key, Auth.refresh() Summary

**One-liner:** Added image_picker ^1.2.2 dependency, iOS photo-library plist key, `AppLocalizations.account` l10n getter, and `Auth.refresh()` Riverpod notifier method as prerequisites for Plans 02 and 03.

## What Was Built

This plan establishes four foundation contracts required by the Account screen (Plan 03) and the email/password form sheets (Plan 02):

1. **image_picker dependency** — `flutter pub add image_picker` resolved version 1.2.2 (flutter.dev verified publisher). Added to `app/pubspec.yaml` as `image_picker: ^1.2.2`; `pubspec.lock` updated with 13 new transitive packages.

2. **iOS photo-library permission** — Added `NSPhotoLibraryUsageDescription` to `app/ios/Runner/Info.plist` with user-facing reason string. Without this key the iOS app crashes on first gallery picker invocation.

3. **`account` localization key** — Added `"account": "Account"` to `app/lib/i18n/app_en.arb` in alphabetical position. Ran `flutter gen-l10n` to regenerate `AppLocalizations.account` abstract getter and all 14 per-locale concrete implementations. German locale now shows 1 untranslated message (expected; falls back to English automatically).

4. **`Auth.refresh()` public method** — Added `Future<void> refresh() async` to the `Auth` Riverpod notifier. The method reads `state.value?.id` (using `.value`, not `.valueOrNull`, per user MEMORY), returns early if `id == null` (no-logged-in-user guard), and otherwise delegates to the existing private `_updateUserEntity(id)` inside `AsyncValue.guard`. After editing the codegen-annotated file, `dart run build_runner build --delete-conflicting-outputs` regenerated `auth_provider.g.dart`. The method is additive — `build()` behavior is unchanged.

## TDD Gate Compliance

Task 3 followed TDD:

- **RED:** `app/test/provider/auth_provider_refresh_test.dart` created with a tear-off `notifier.refresh` that fails to compile because `refresh()` did not exist yet. Confirmed with `flutter test` (compilation failure). Committed as `test(08-01): add failing test for Auth.refresh() compile-time contract`.
- **GREEN:** Added `refresh()` to `auth_provider.dart`. Test compiles and passes. Committed as `feat(08-01): add public refresh() to Auth notifier`.
- **REFACTOR:** No refactoring needed — implementation is the exact Pattern 4 from 08-PATTERNS.md.

## Commits

| Task | Commit | Type | Description |
|------|--------|------|-------------|
| 1 | 521fba7f | feat | Add image_picker dependency and iOS photo-library permission |
| 2 | 1b7f80f8 | feat | Add account localization key and regenerate AppLocalizations |
| 3 (RED) | 62723542 | test | Add failing test for Auth.refresh() compile-time contract |
| 3 (GREEN) | d3efe247 | feat | Add public refresh() to Auth notifier |

## Verification Results

All plan verification checks pass:

- `flutter pub get` exits 0 (image_picker resolved)
- `NSPhotoLibraryUsageDescription` present in `app/ios/Runner/Info.plist`
- `"account":` key present in `app/lib/i18n/app_en.arb`
- `String get account;` present in `app/lib/i18n/app_localizations.dart`
- `Future<void> refresh()` declared on `Auth` class
- Body references `_updateUserEntity` and `AsyncValue.guard`, guards on `state.value?.id == null`
- No `.valueOrNull` usage in `auth_provider.dart`
- `dart run build_runner build --delete-conflicting-outputs` exits 0
- `flutter analyze lib/provider/auth_provider.dart` exits 0

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — this plan adds contracts (dependency, permission, l10n key, method) with no stub values or placeholder text.

## Threat Flags

None — no new network endpoints, auth paths, file access patterns, or schema changes were introduced beyond what the plan's threat model covers. The `NSPhotoLibraryUsageDescription` mitigates T-08-01 as specified.

## Self-Check: PASSED

- `app/pubspec.yaml` contains `image_picker: ^1.2.2`: FOUND
- `app/ios/Runner/Info.plist` contains `NSPhotoLibraryUsageDescription`: FOUND
- `app/lib/i18n/app_en.arb` contains `"account":`: FOUND
- `app/lib/i18n/app_localizations.dart` contains `String get account;`: FOUND
- `app/lib/provider/auth_provider.dart` contains `Future<void> refresh()`: FOUND
- `app/test/provider/auth_provider_refresh_test.dart`: FOUND
- Commits 521fba7f, 1b7f80f8, 62723542, d3efe247: all present in git log
