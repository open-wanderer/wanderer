# Deferred Items — Phase 09 Notifications

Out-of-scope discoveries logged during execution of plan 09-01. These are NOT
caused by this plan's changes and were left untouched per the executor scope
boundary (only auto-fix issues directly caused by the current task's changes).

## Pre-existing compile error (Phase 8 regression)

- **File:** `app/lib/components/settings/password_change_sheet.dart:56`
- **Error:** `The name 'DioException' isn't a type and can't be used in an on-catch clause` (`non_type_in_catch_clause`)
- **Cause:** `on DioException catch (error)` is used but `package:dio/dio.dart` is not imported. Present at the Phase 9 baseline commit `f652037a` (introduced by Phase 8 commit `63ec9b81`).
- **Impact:** `flutter analyze` reports 1 error; `settings_account_screen_test.dart` fails to load because it transitively imports this file.
- **Fix:** Add `import 'package:dio/dio.dart';` to `password_change_sheet.dart`.

## Pre-existing widget-test failures (full-suite only)

- **Files:** `test/routes/settings_language_screen_test.dart` ("language screen renders 14 locale tiles + units switch (LANG-01)"), `test/routes/settings_privacy_screen_test.dart` (fails on full-suite load, passes in isolation).
- **Cause:** Not related to plan 09-01. `settings_privacy_screen_test.dart` passes when run alone; it only fails when run as part of `flutter test test/routes/` (cross-test viewport contamination). The language test fails on a `dragUntilVisible`/`scrollUntilVisible` viewport assertion independent of notifications work.
- **Impact:** `flutter test test/routes/` is not fully green at the phase level.
- **Note:** The new `settings_notifications_screen_test.dart` passes both in isolation and contributes 2 passing tests to the suite.
