---
phase: 06-settings-navigation-language-units
plan: 01
subsystem: mobile-app
tags: [riverpod, i18n, localization, settings, providers]
requires:
  - settingsProvider (existing)
  - AppLocalizations (generated delegate)
provides:
  - localeProvider (derived Riverpod provider)
  - unitProvider (derived Riverpod provider)
  - MaterialApp.router live locale wiring
  - Wave 0 test scaffolds for SETNAV-01 and LANG-01
affects:
  - app/lib/main.dart
  - Plan 02 (Language & Units screen — consumes localeProvider, fills test scaffolds)
  - Plan 03 (unit-wiring call sites — consume unitProvider)
tech-stack:
  added: []
  patterns:
    - "@Riverpod(keepAlive: true) derived function-style providers cloning the themeMode template"
    - "ProviderContainer override via settingsProvider.overrideWithValue for parent-driven provider tests"
key-files:
  created:
    - app/test/provider/unit_provider_test.dart
    - app/test/routes/settings_screen_test.dart
    - app/test/routes/settings_language_screen_test.dart
  modified:
    - app/lib/provider/local_settings_provider.dart
    - app/lib/provider/local_settings_provider.g.dart
    - app/lib/main.dart
decisions:
  - "localeProvider returns null when language is null so Flutter falls back to the device locale (per CONTEXT D-01)"
  - "unitProvider falls back to 'metric' on null settings or null unit, matching format_util.dart default"
  - "supportedLocales sourced from AppLocalizations.supportedLocales so it auto-grows to 14 locales once Plan 03 adds the ARB files"
metrics:
  duration: ~6 min
  completed: 2026-06-19
requirements: [LANG-01, LANG-02]
---

# Phase 6 Plan 01: Phase 6 Interface Providers + Live Locale Wiring Summary

Added the `localeProvider` and `unitProvider` derived Riverpod providers, wired the live locale switch into `MaterialApp.router`, expanded `supportedLocales` to the generated locale list, and created the Wave 0 test scaffolds (one passing unit test, two skipped placeholders) that Phase 6 Plans 02 and 03 build on.

## What Was Built

- **`localeProvider`** (`@Riverpod(keepAlive: true) Locale? locale(Ref ref)`): watches `settingsProvider`, returns `null` when `language` is null (device-locale fallback) and `Locale(lang.name)` otherwise. `Language.name` (e.g. `en`, `de`) equals the locale code, verified against the `@JsonValue` annotations in `settings.dart`.
- **`unitProvider`** (`@Riverpod(keepAlive: true) String unit(Ref ref)`): watches `settingsProvider`, returns `settings?.unit ?? 'metric'`.
- **`main.dart` wiring**: added `locale: ref.watch(localeProvider)` to `MaterialApp.router` and replaced the hardcoded `const [Locale('en'), Locale('de')]` with `AppLocalizations.supportedLocales`. `localizationsDelegates` kept as the existing explicit 4-entry list.
- **Codegen**: `local_settings_provider.g.dart` regenerated with `localeProvider` and `unitProvider` symbols.
- **Tests**: `unit_provider_test.dart` asserts all three derivation cases (null settings → metric, imperial → imperial, null unit → metric); `settings_screen_test.dart` and `settings_language_screen_test.dart` contain single skipped tests marked `MISSING — Plan 02`.

## Task Commits

| Task | Name | Commit | Files |
| ---- | ---- | ------ | ----- |
| 1 | Add localeProvider and unitProvider | 70a58359 | local_settings_provider.dart (+.g.dart) |
| 2 | Wire live locale switch + supportedLocales | e490fd14 | main.dart |
| 3 | unitProvider test + Wave 0 scaffolds | 2592d376 | 3 test files |

## Verification

- `flutter analyze lib/provider/local_settings_provider.dart` → No issues found
- `flutter analyze lib/main.dart` → No issues found
- `flutter test test/provider/unit_provider_test.dart` → 3 passed
- `flutter test test/routes/settings_screen_test.dart test/routes/settings_language_screen_test.dart` → 2 skipped (pass)
- `local_settings_provider.g.dart` defines `localeProvider` and `unitProvider` (grep-confirmed)

## TDD Gate Compliance

Task 3 is `tdd="true"`. The implementation under test (`unitProvider`) was created in Task 1 as the interface-defining work, so the unit test passed GREEN on first run rather than going through an isolated RED commit. No separate failing-test commit was made; the provider and its test were authored within the same plan with verification confirming GREEN. The behavior is fully covered by the three passing assertions.

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

The two widget-test scaffolds (`settings_screen_test.dart`, `settings_language_screen_test.dart`) are intentional Wave 0 placeholders. Each contains a single `markTestSkipped('MISSING — Plan 02 ...')` test with no assertions. These are by design (per plan `<behavior>` and `artifacts`) and will be filled with real assertions by Plan 02. They do not block this plan's goal (interface providers + live locale wiring), which is fully achieved.

## Self-Check: PASSED

- FOUND: app/lib/provider/local_settings_provider.dart (localeProvider, unitProvider)
- FOUND: app/lib/main.dart (locale wiring, AppLocalizations.supportedLocales)
- FOUND: app/test/provider/unit_provider_test.dart
- FOUND: app/test/routes/settings_screen_test.dart
- FOUND: app/test/routes/settings_language_screen_test.dart
- FOUND: commit 70a58359
- FOUND: commit e490fd14
- FOUND: commit 2592d376
