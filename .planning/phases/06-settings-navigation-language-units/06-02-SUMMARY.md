---
phase: 06-settings-navigation-language-units
plan: 02
subsystem: ui
tags: [flutter, riverpod, go_router, i18n, settings, language, units]

requires:
  - phase: 06-01
    provides: localeProvider, unitProvider, live-locale MaterialApp wiring, Wave 0 test scaffolds
provides:
  - SettingsLanguageScreen (14-locale RadioGroup + metric/imperial switch, auto-save)
  - SettingsPrivacyScreen (stub, Phase 7 fills body)
  - SettingsNotificationsScreen (stub, Phase 9 fills body)
  - Three new ListTile rows in SettingsScreen (Privacy, Language, Notifications) in D-06 order
  - privacy/language/notifications GoRoute children under /settings
  - Real SETNAV-01 and LANG-01 widget-test assertions
affects:
  - Phase 07 (Privacy — fills SettingsPrivacyScreen body)
  - Phase 09 (Notifications — fills SettingsNotificationsScreen body)
  - Plan 06-03 (unit-display wiring — units toggle now persists)

tech-stack:
  added: []
  patterns:
    - "Stub screen pattern: themed Scaffold + titled AppBar + SizedBox.shrink body for not-yet-built sub-screens"
    - "Auto-save-on-select with try/catch toast-on-failure wrapping settingsProvider.saveToServer (no Save button)"
    - "scrollUntilVisible in widget tests to reach lazily-laid-out ListView children below the viewport"

key-files:
  created:
    - app/lib/routes/settings_language_screen.dart
    - app/lib/routes/settings_privacy_screen.dart
    - app/lib/routes/settings_notifications_screen.dart
  modified:
    - app/lib/routes/settings_screen.dart
    - app/lib/provider/router_provider.dart
    - app/test/routes/settings_screen_test.dart
    - app/test/routes/settings_language_screen_test.dart
    - app/lib/i18n/app_en.arb
    - app/lib/i18n/app_de.arb
    - app/lib/i18n/app_localizations.dart
    - app/lib/i18n/app_localizations_en.dart
    - app/lib/i18n/app_localizations_de.dart

key-decisions:
  - "Switch polarity: imperial = on-position, metric = off (UI-SPEC D-10); SwitchListTile title is l10n.imperial"
  - "RadioGroup<Language> (not <String>) to use the enum directly and avoid string-matching bugs"
  - "Added error_saving_settings ARB key (en/de) — no suitable generic save-error key existed for the mandated failure toast"

patterns-established:
  - "Stub sub-screen: titled AppBar + empty body, looks intentional, not blank"
  - "Auto-save selection with toast-on-failure, no Save button"

requirements-completed: [SETNAV-01, LANG-01, LANG-02]

duration: 14min
completed: 2026-06-19
---

# Phase 6 Plan 02: Settings Navigation + Language & Units Screen Summary

**Five-row settings list (Account/Privacy/Language/Notifications/Appearance) wired to /settings sub-routes, with a full 14-locale RadioGroup<Language> + metric/imperial switch screen that auto-saves to the server, plus two themed stub screens.**

## Performance

- **Duration:** ~14 min
- **Started:** 2026-06-19T20:55:41Z
- **Completed:** 2026-06-19T21:07:23Z
- **Tasks:** 3
- **Files modified:** 12 (3 created lib screens, 2 modified lib, 2 tests, 5 i18n)

## Accomplishments

- `SettingsLanguageScreen` renders 14 native-name locale radio tiles (`RadioGroup<Language>`) and an imperial/metric `SwitchListTile`, both auto-saving via `settingsProvider.saveToServer` with toast-on-failure.
- Two intentional stub screens (`SettingsPrivacyScreen`, `SettingsNotificationsScreen`) with titled AppBars, ready for Phases 7 and 9.
- Settings list now shows all five rows in D-06 order, each navigating via `context.push`; account/appearance routes unchanged.
- Three new `/settings` GoRoute children (privacy, language, notifications) added without regressing existing routes.
- Wave 0 test scaffolds replaced with real assertions: 5-row SETNAV-01 test and 14-tile + units-switch LANG-01 test, both green.

## Task Commits

Each task was committed atomically:

1. **Task 1: Build the Language & Units screen** - `10c7accc` (feat)
2. **Task 2: Add stub Privacy + Notifications screens and wire routes** - `19eb3c53` (feat)
3. **Task 3: Add settings rows and fill Wave 0 widget tests** - `bd96e289` (feat)

## Files Created/Modified

- `app/lib/routes/settings_language_screen.dart` - Language & Units screen (RadioGroup<Language> + SwitchListTile, auto-save)
- `app/lib/routes/settings_privacy_screen.dart` - Stub Privacy screen (Phase 7 fills body)
- `app/lib/routes/settings_notifications_screen.dart` - Stub Notifications screen (Phase 9 fills body)
- `app/lib/routes/settings_screen.dart` - Added Privacy/Language/Notifications rows in D-06 order
- `app/lib/provider/router_provider.dart` - Added privacy/language/notifications GoRoute children + imports
- `app/test/routes/settings_screen_test.dart` - Real SETNAV-01 5-row assertions
- `app/test/routes/settings_language_screen_test.dart` - Real LANG-01 14-tile + units-switch assertions
- `app/lib/i18n/app_en.arb`, `app_de.arb` - Added error_saving_settings key
- `app/lib/i18n/app_localizations*.dart` - Regenerated from ARB

## Decisions Made

- Switch polarity locked per UI-SPEC D-10: imperial is on, metric is off; switch title is `l10n.imperial` with a `l10n.units` section label above.
- Used `RadioGroup<Language>` with a hardcoded `const Map<Language, String>` of native names (the single approved hardcoded-string exception), avoiding string-match bugs.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added error_saving_settings ARB key**
- **Found during:** Task 1 (Language & Units screen)
- **Issue:** The plan mandates a toast on save failure, but no localized message existed for a generic settings-save error (only entity-specific keys like error_saving_trail). Referencing a non-existent `l10n.error` was an analyzer error.
- **Fix:** Added `error_saving_settings` to `app_en.arb` ("Error saving settings") and `app_de.arb` ("Fehler beim Speichern der Einstellungen"), regenerated localizations via `flutter gen-l10n`, and used `l10n.error_saving_settings` in the toast.
- **Files modified:** app/lib/i18n/app_en.arb, app/lib/i18n/app_de.arb, app/lib/i18n/app_localizations*.dart
- **Verification:** `flutter analyze lib/routes/settings_language_screen.dart` → No issues found
- **Committed in:** 10c7accc (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical)
**Impact on plan:** The added ARB key is required for the plan's mandated save-failure toast. No scope creep.

## Issues Encountered

- The LANG-01 widget test initially failed (`Bad state: No element`) because the units `SwitchListTile` is a `ListView` child laid out below the 14 radio tiles and is not built/laid-out in the test viewport. Resolved by adding `tester.scrollUntilVisible(find.byType(SwitchListTile), 300)` before the assertion. (The 14 `RadioListTile`s count correctly because they live in an eagerly-built `Column`.)

## Known Stubs

- `SettingsPrivacyScreen` and `SettingsNotificationsScreen` render a titled AppBar with an empty body (`SizedBox.shrink`). These are intentional, plan-specified stubs (UI-SPEC line 125) — Phase 7 fills Privacy, Phase 9 fills Notifications. They are reachable and look intentional (not blank), so they do not block this plan's goal (all five settings rows reachable).
- The language-screen tap-to-save assertion is left as a TODO comment in `settings_language_screen_test.dart` because the test harness has no `apiProvider`/HTTP override fixture; the save path is covered by provider-level unit tests and the render/count assertions.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- SETNAV-01 surface complete: all five settings rows reachable. Phases 7 (Privacy) and 9 (Notifications) can fill their stub screen bodies.
- LANG-01/LANG-02 persistence complete: language selection and units toggle save to the server; live locale re-render is wired via Plan 01's `localeProvider`. Plan 03 will wire unit-display call sites to `unitProvider`.

## Self-Check: PASSED

- FOUND: app/lib/routes/settings_language_screen.dart
- FOUND: app/lib/routes/settings_privacy_screen.dart
- FOUND: app/lib/routes/settings_notifications_screen.dart
- FOUND: commit 10c7accc
- FOUND: commit 19eb3c53
- FOUND: commit bd96e289

---
*Phase: 06-settings-navigation-language-units*
*Completed: 2026-06-19*
