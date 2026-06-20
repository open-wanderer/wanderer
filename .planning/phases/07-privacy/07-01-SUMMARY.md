---
phase: 07-privacy
plan: 01
subsystem: mobile-settings
tags: [flutter, riverpod, privacy, settings, ui]
requires:
  - settingsProvider
  - SettingsPrivacy
  - /settings/privacy route
  - privacy ARB keys (account_privacy, only_me, settings_privacy_*)
provides:
  - SettingsPrivacyScreen (full ConsumerWidget implementation)
affects:
  - app/lib/routes/settings_privacy_screen.dart
tech-stack:
  added: []
  patterns:
    - "ConsumerWidget + RadioGroup<String> auto-save mirror of SettingsLanguageScreen"
    - "Tall test viewport to mount lazily-built ListView tiles in widget tests"
key-files:
  created:
    - app/test/routes/settings_privacy_screen_test.dart
  modified:
    - app/lib/routes/settings_privacy_screen.dart
decisions:
  - "Plural ARB getters use positional args (l10n.trail(2), l10n.list(2)) not named (n:) — plan/UI-SPEC syntax was wrong; matched codebase convention"
  - "Widget test sets a 1080x4000 viewport so all six long-subtitle tiles mount at once (ListView lazy-build) instead of scroll-stepping"
metrics:
  duration_min: 15
  completed: "2026-06-20T09:00:19Z"
  tasks: 2
  files: 2
---

# Phase 7 Plan 01: Privacy Screen Summary

SettingsPrivacyScreen now renders three RadioGroup<String> visibility sections (Account, Trails, Lists) with six subtitle-bearing tiles that auto-save the changed field via settingsProvider, plus a passing widget test.

## What Was Built

- **Task 1** — Replaced the `StatelessWidget` stub body of `SettingsPrivacyScreen` with a full `ConsumerWidget` implementation: three `RadioGroup<String>` sections (Account/Trails/Lists) each holding two `RadioListTile<String>` options with descriptive subtitles. Selecting a radio builds an updated `SettingsPrivacy` via `copyWith` on only the changed field (preserving the other two) and auto-saves through `settingsProvider.notifier.saveToServer`, with a verbatim `_save` try/catch that surfaces an `error_saving_settings` toast on failure. Null-privacy defaults: account=public, trails=private, lists=private (D-04). Commit `9206f300`.
- **Task 2** — Added `settings_privacy_screen_test.dart` with two `testWidgets`: one asserting the three section headers and six string radio tiles with correct labels (Public ×3, Only me ×2, Private ×1), and one asserting D-04 defaults via the three `RadioGroup<String>` groupValues when `settings.privacy` is null. Commit `8d2aafd6`.

## Verification

- `flutter analyze lib/routes/settings_privacy_screen.dart` — No issues found.
- `flutter analyze test/routes/settings_privacy_screen_test.dart` — No issues found.
- `flutter test test/routes/settings_privacy_screen_test.dart` — All tests passed (2/2).
- Task 1 source-assertion acceptance criteria verified via grep: ConsumerWidget=1, RadioGroup<String>=3, RadioListTile<String>=6, six distinct subtitle keys, `?? "public"`=1, `?? "private"`=2, `l10n.only_me`=2, `l10n.private`=1, `saveToServer(updated)`=1, hex/fontSize=0.
- Task 2 acceptance criteria verified: `overrideWithValue` present, `findsNWidgets(6)` present, all three section-header text literals present.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Plural ARB getters require positional, not named, arguments**
- **Found during:** Task 1 (`flutter analyze`)
- **Issue:** The plan and UI-SPEC specified `l10n.trail(n: 2)` and `l10n.list(n: 2)`. The generated localization getters take a single positional argument, so the named-parameter form failed to compile (`undefined_named_parameter`, `not_enough_positional_arguments`).
- **Fix:** Changed to `l10n.trail(2)` and `l10n.list(2)`, matching the established codebase convention (e.g. `wanderer_layout.dart`, `profile_screen.dart`).
- **Files modified:** app/lib/routes/settings_privacy_screen.dart
- **Commit:** 9206f300

### Acceptance-criterion grep variance (non-deviation, documented)

- Task 1 acceptance check `grep -c "SettingsPrivacy(account: 'public', trails: 'private', lists: 'private')"` expects the fallback constructor on a single line and returns 0, because `dart format` wraps the long `const SettingsPrivacy(...)` constructor across multiple lines. The functional requirement (fallback constructor with exactly those three values feeding `copyWith`, preserving the other two fields) is met identically in all three sections (lines 89-92, 133-136, 177-180). No behavior change; forcing a single line would be reverted by the formatter.

## Self-Check: PASSED

- FOUND: app/lib/routes/settings_privacy_screen.dart
- FOUND: app/test/routes/settings_privacy_screen_test.dart
- FOUND commit: 9206f300 (feat 07-01 privacy screen)
- FOUND commit: 8d2aafd6 (test 07-01 widget test)
