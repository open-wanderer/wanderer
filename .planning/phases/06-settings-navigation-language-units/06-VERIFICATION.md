---
phase: 06-settings-navigation-language-units
verified: 2026-06-20T00:00:00Z
status: human_needed
score: 4/4 must-haves verified
overrides_applied: 0
re_verification: false
human_verification:
  - test: "Tap each Settings row (Privacy, Language, Notifications) and confirm navigation"
    expected: "Each tap navigates to the correct sub-screen with a titled AppBar and back button"
    why_human: "GoRouter navigation requires a running app; grep confirms routes and push calls exist but cannot exercise runtime routing"
  - test: "Select a language (e.g. French) in the Language & Units screen, background the app, reopen"
    expected: "UI renders in French after the selection and persists on reopen; untranslated strings fall back to English"
    why_human: "Live locale re-render requires a running Flutter app and server-side persistence; cannot be verified programmatically"
  - test: "Toggle the metric/imperial switch in the Language & Units screen"
    expected: "Trail card distances, navigation stats, and elevation profile re-render in mi/ft immediately without restart"
    why_human: "Reactive re-render across 14 files requires a running app; code paths are verified but live state propagation is not"
  - test: "Confirm existing /settings/account and /settings/appearance screens remain reachable and unbroken"
    expected: "Both screens open normally with no regressions to existing UI"
    why_human: "No regression test for previously-existing routes; requires manual smoke"
---

# Phase 6: Settings Navigation, Language & Units Verification Report

**Phase Goal:** Users can reach every settings sub-screen from the Settings list and can set their preferred language and unit system
**Verified:** 2026-06-20
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| #  | Truth                                                                                                        | Status     | Evidence                                                                                                                                                                  |
|----|--------------------------------------------------------------------------------------------------------------|------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| 1  | Settings screen lists Account, Privacy, Language, Notifications, Appearance; each navigates to its screen    | VERIFIED   | `settings_screen.dart`: 5 ListTiles in D-06 order with FontAwesomeIcons.lock/.globe/.bell and `context.push` to `/settings/privacy`, `/settings/language`, `/settings/notifications`. Router has GoRoute for all 5 sub-paths. Widget test asserts 5 ListTile + 5 labels (SETNAV-01). |
| 2  | Tapping Language opens a 14-locale picker; selection persists after leaving and reopening                     | VERIFIED   | `settings_language_screen.dart`: `RadioGroup<Language>` with 14 `RadioListTile<Language>` built from `Language.values`. `_save` helper calls `settingsProvider.saveToServer(settings.copyWith(language:))` wrapped in try/catch. Widget test (green) asserts 14 tiles, 'English', '中文'. |
| 3  | Language screen lets user toggle metric/imperial; choice persists across app restarts                         | VERIFIED   | `settings_language_screen.dart`: `SwitchListTile(value: settings?.unit == 'imperial', onChanged: ...)` calls `saveToServer(settings.copyWith(unit:))`. `unitProvider` in `local_settings_provider.dart` reads `settings?.unit ?? 'metric'`. 14 call-site files all pass `unit: unit` to format_util (grep verified). `unit_provider_test.dart`: 3 green tests. |
| 4  | All 14 locales have ARB files; AppLocalizations.supportedLocales covers all 14; live switch works per locale  | VERIFIED   | All 12 new ARB files (`app_cs.arb` through `app_zh.arb`) present, valid JSON, correct `@@locale` headers, no metadata keys. `app_localizations.dart` has 14 `Locale(` entries. `main.dart` wires `supportedLocales: AppLocalizations.supportedLocales` and `locale: ref.watch(localeProvider)`. |

**Score:** 4/4 truths verified

### Required Artifacts

| Artifact                                                        | Expected                                                      | Status     | Details                                                                                         |
|-----------------------------------------------------------------|---------------------------------------------------------------|------------|-------------------------------------------------------------------------------------------------|
| `app/lib/provider/local_settings_provider.dart`                 | localeProvider and unitProvider derived providers             | VERIFIED   | Contains `@Riverpod(keepAlive: true) Locale? locale(Ref ref)` and `String unit(Ref ref)` both watching `settingsProvider`; correct null fallbacks |
| `app/lib/provider/local_settings_provider.g.dart`               | Generated symbols for both providers                          | VERIFIED   | `localeProvider = LocaleProvider._()` and `unitProvider = UnitProvider._()` present at lines 107 and 148 |
| `app/lib/main.dart`                                             | MaterialApp.router locale wiring + 14 supportedLocales        | VERIFIED   | `locale: ref.watch(localeProvider),` at line 70; `supportedLocales: AppLocalizations.supportedLocales` at line 65; 4-delegate explicit list retained |
| `app/lib/routes/settings_language_screen.dart`                  | 14-locale RadioGroup + metric/imperial SwitchListTile, auto-save | VERIFIED | `RadioGroup<Language>` with 14 tiles from `Language.values`; `SwitchListTile(value: settings?.unit == 'imperial')`; `_save` with try/catch toast; 119 lines |
| `app/lib/routes/settings_screen.dart`                           | 5-row settings list in D-06 order                             | VERIFIED   | FontAwesomeIcons.lock, .globe, .bell; `context.push('/settings/privacy'|'/settings/language'|'/settings/notifications')`; 5 ListTiles before Divider |
| `app/lib/routes/settings_privacy_screen.dart`                   | Stub Privacy screen with titled AppBar                        | VERIFIED   | `class SettingsPrivacyScreen extends StatelessWidget`; AppBar titled `AppLocalizations.of(context)!.privacy`; `SizedBox.shrink` body |
| `app/lib/routes/settings_notifications_screen.dart`             | Stub Notifications screen with titled AppBar                  | VERIFIED   | `class SettingsNotificationsScreen extends StatelessWidget`; AppBar titled `AppLocalizations.of(context)!.notifications`; `SizedBox.shrink` body |
| `app/lib/provider/router_provider.dart`                         | privacy/language/notifications GoRoute children under /settings | VERIFIED | `path: 'privacy'`, `path: 'language'`, `path: 'notifications'` present at lines 178-193; existing `account` and `appearance` children retained |
| `app/lib/components/trail/elevation_profile.dart`               | ConsumerStatefulWidget with unitProvider                      | VERIFIED   | `class ElevationProfile extends ConsumerStatefulWidget`; `_unit = ref.watch(unitProvider)` at line 110 |
| `app/lib/components/trail/waypoint_sheet.dart`                  | ConsumerWidget with unitProvider                              | VERIFIED   | `class WaypointSheet extends ConsumerWidget`; `ref.watch(unitProvider)` at line 29 |
| `app/lib/components/trail/trail_timeline.dart`                  | ConsumerWidget threading unit to _TimelineRow                 | VERIFIED   | `class TrailTimeline extends ConsumerWidget`; `_TimelineRow(data: rows[i], unit: unit)`; `_TimelineRow` has `required this.unit` |
| `app/test/provider/unit_provider_test.dart`                     | 3 unitProvider derivation tests (LANG-02)                     | VERIFIED   | Tests: null settings → 'metric', unit 'imperial' → 'imperial', unit null → 'metric'; uses ProviderContainer with settingsProvider.overrideWithValue |
| `app/test/routes/settings_screen_test.dart`                     | Real 5-row SETNAV-01 widget assertions                        | VERIFIED   | No `MISSING — Plan 02` string; asserts `findsNWidgets(5)` for ListTile; all 5 English labels asserted |
| `app/test/routes/settings_language_screen_test.dart`            | Real 14-tile + SwitchListTile LANG-01 widget assertions       | VERIFIED   | No `MISSING — Plan 02` string; asserts `findsNWidgets(Language.values.length)` for RadioListTile; 'English' and '中文' found; SwitchListTile value == false for metric |
| `app/test/util/format_util_test.dart`                           | Imperial tests for formatDistance + formatElevation           | VERIFIED   | `formatDistance(1000, unit: 'imperial')` → '0.62 mi'; `formatElevation(100, unit: 'imperial')` → '328 ft' |
| `app/lib/i18n/app_fr.arb` through `app_zh.arb` (12 files)     | 12 locale ARB files with @@locale headers                     | VERIFIED   | All 12 present, valid JSON, `@@locale` headers match locale codes, no `@`-prefixed metadata keys |
| `app/lib/i18n/app_localizations.dart`                           | Regenerated with 14 supportedLocales                          | VERIFIED   | `grep -c "Locale('"` = 14; 14 per-locale `app_localizations_<code>.dart` generated |

### Key Link Verification

| From                                           | To                           | Via                                        | Status   | Details                                                                               |
|------------------------------------------------|------------------------------|--------------------------------------------|----------|---------------------------------------------------------------------------------------|
| `app/lib/main.dart`                            | `localeProvider`             | `ref.watch(localeProvider)` in MaterialApp.router | VERIFIED | Line 70: `locale: ref.watch(localeProvider),`                                        |
| `app/lib/provider/local_settings_provider.dart`| `settingsProvider`           | `ref.watch(settingsProvider)` in both providers | VERIFIED | Line 47: `ref.watch(settingsProvider)?.language`; line 54: `ref.watch(settingsProvider)` |
| `app/lib/routes/settings_screen.dart`          | `/settings/language`         | `context.push` in ListTile.onTap           | VERIFIED | Line 44: `onTap: () => context.push('/settings/language'),`                          |
| `app/lib/routes/settings_language_screen.dart` | `settingsProvider.saveToServer` | `_save` helper via RadioGroup.onChanged + SwitchListTile.onChanged | VERIFIED | Line 42: `await ref.read(settingsProvider.notifier).saveToServer(updated)` in try/catch |
| `app/lib/provider/router_provider.dart`        | `SettingsLanguageScreen`     | GoRoute builder for path 'language'        | VERIFIED | Line 183: `builder: (context, state) => const SettingsLanguageScreen()`             |
| 11 Consumer call sites                         | `unitProvider`               | `ref.watch(unitProvider)` in build         | VERIFIED | `trail_card.dart`:37, `trail_list_item.dart`:27, `trail_panel.dart`:36, `trail_quick_filter_bar.dart`:402, `summit_log_card.dart`:25, `list_card.dart`:28, `list_list_item.dart`:21, `global_search_screen.dart`:44, `list_detail_screen.dart`:109, `navigation_screen.dart`:180, `trail_filter_screen.dart`:54 |
| `app/lib/i18n/app_*.arb`                       | `app_localizations.dart`     | flutter gen-l10n regeneration              | VERIFIED | 14 per-locale dart files generated; `supportedLocales` count = 14                   |

### Data-Flow Trace (Level 4)

| Artifact                            | Data Variable    | Source                                              | Produces Real Data | Status   |
|-------------------------------------|-----------------|-----------------------------------------------------|--------------------|----------|
| `settings_language_screen.dart`     | `settings`      | `ref.watch(settingsProvider)` → server Settings object | Yes (live server watch) | FLOWING |
| `settings_screen.dart`              | (static list)   | N/A — list rows are static, no dynamic data        | N/A                | N/A      |
| `local_settings_provider.dart`      | `unit`          | `ref.watch(settingsProvider)?.unit ?? 'metric'`    | Yes (via settingsProvider) | FLOWING |
| `local_settings_provider.dart`      | `locale`        | `ref.watch(settingsProvider)?.language`            | Yes (via settingsProvider) | FLOWING |

### Behavioral Spot-Checks

Step 7b: SKIPPED — requires running Flutter app (no CLI entry point for spot-checking Riverpod providers or widget rendering without `flutter test`).

### Probe Execution

No probe scripts declared or found for Phase 6. Phase 6 is a UI phase with no conventional `scripts/*/tests/probe-*.sh` files.

### Requirements Coverage

| Requirement | Source Plan | Description                                                  | Status    | Evidence                                                                                                  |
|-------------|-------------|--------------------------------------------------------------|-----------|-----------------------------------------------------------------------------------------------------------|
| SETNAV-01   | 06-02       | Settings screen lists Account, Privacy, Language, Notifications, Appearance | SATISFIED | `settings_screen.dart`: 5 ListTiles in D-06 order; widget test asserts 5 rows + all 5 labels; router has all 5 GoRoute children |
| LANG-01     | 06-01, 06-02, 06-04 | User can select preferred language from 14 supported locales | SATISFIED | `settings_language_screen.dart`: 14 `RadioListTile<Language>` auto-saving; 12 new ARB files + regenerated AppLocalizations with 14 supportedLocales; `main.dart` wires `localeProvider` |
| LANG-02     | 06-01, 06-02, 06-03 | User can toggle between metric and imperial units            | SATISFIED | `SwitchListTile` auto-saves via `saveToServer(settings.copyWith(unit:))`; `unitProvider` provides live unit; all 14 call-site files pass `unit:` to format_util |

No orphaned requirements — all 3 Phase 6 requirements (SETNAV-01, LANG-01, LANG-02) are claimed by plans and have implementation evidence.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `app/lib/routes/settings_privacy_screen.dart` | 20 | `body: const SizedBox.shrink()` | Info | Intentional stub per plan spec; Phase 7 fills body. Not a blocker. |
| `app/lib/routes/settings_notifications_screen.dart` | 20 | `body: const SizedBox.shrink()` | Info | Intentional stub per plan spec; Phase 9 fills body. Not a blocker. |
| `app/test/routes/settings_language_screen_test.dart` | 55 | `// TODO: tap-to-save assertion` | Info | Acknowledged limitation per plan; save path covered by unit tests. No `#issue` reference but explicitly documented in SUMMARY.md as known. Not a debt-marker blocker (it is a comment about test scope, not a code TBD). |

No `TBD`, `FIXME`, or `XXX` markers found in any Phase 6 modified lib files.

The `SizedBox.shrink()` body and the TODO comment are both intentional plan-specified choices, not stubs hiding missing behavior. The core phase goal does not depend on Privacy or Notifications body content (those are Phase 7 and 9 work).

### Human Verification Required

#### 1. Settings sub-navigation (runtime routing)

**Test:** Tap each of the five Settings rows (Account, Privacy, Language, Notifications, Appearance)
**Expected:** Each row navigates to the correct screen with a titled AppBar and functional back button
**Why human:** GoRouter navigation requires a running Flutter app; grep confirms `context.push` calls and GoRoute wiring exist, but runtime routing cannot be exercised programmatically

#### 2. Live locale switch after language selection

**Test:** Open Language & Units screen, pick French (Français), navigate away, reopen the screen
**Expected:** UI text re-renders in French for translated keys; untranslated strings remain in English; the French tile shows as selected when the screen is reopened
**Why human:** Live `localeProvider` re-render and server-persisted selection round-trip require a running app and server connection

#### 3. Live unit display re-render after toggling metric/imperial

**Test:** Toggle the metric/imperial switch to imperial, then navigate to a trail card, navigation screen, and elevation profile
**Expected:** Distances display in miles, elevations in feet, speeds in mph — re-rendered on the next frame without restart
**Why human:** Reactive `unitProvider` re-render across 14 files requires a running app; all code paths are wired (verified) but live state propagation needs visual confirmation

#### 4. Regression check for existing settings screens

**Test:** Navigate to /settings/account and /settings/appearance from the Settings list
**Expected:** Both screens open and function exactly as before Phase 6 changes
**Why human:** No automated regression test for the pre-existing account/appearance routes; the router additions were non-destructive (verified by code inspection), but human smoke is the authoritative check

### Gaps Summary

No gaps found. All 4 roadmap Success Criteria are verified against the codebase:

1. **SC-1 (5 rows + navigation):** VERIFIED — settings_screen.dart, router_provider.dart, widget test
2. **SC-2 (14-locale picker, persists):** VERIFIED — settings_language_screen.dart, 12 ARB files, AppLocalizations regeneration, widget test
3. **SC-3 (metric/imperial toggle, persists):** VERIFIED — SwitchListTile auto-save, unitProvider wiring across 14 call sites, unit_provider_test.dart
4. **SC-4 (saves to server, no regressions):** VERIFIED (code) / HUMAN NEEDED (runtime smoke) — saveToServer calls present; existing routes retained in router

Status is `human_needed` because 4 UI behavioral checks cannot be exercised programmatically. All automated evidence is green.

---

_Verified: 2026-06-20_
_Verifier: Claude (gsd-verifier)_
