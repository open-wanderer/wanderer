# Milestone v1.2 — Project Summary

**Generated:** 2026-06-21
**Purpose:** Team onboarding and project review

---

## 1. Project Overview

**Wanderer Trail Navigation** is a Flutter mobile app that lets hikers follow any trail step-by-step with turn-by-turn Valhalla-powered maneuvers, a live GPS map, and real-time stats — without leaving the app.

**Core Value:** A hiker can tap "Navigate" on any online trail and follow it step by step without leaving the app.

### Milestone Context

- **v1.0 (complete):** Three phases delivered turn-by-turn navigation from zero: a SvelteKit API endpoint for Valhalla maneuvers (Phase 1), the full-screen Flutter NavigationScreen with map/GPS/maneuver display (Phase 2), and a draggable stats sheet with live distance/elevation/speed and elevation-profile chart (Phase 3).
- **v1.1 (complete):** Two phases added offline navigation: an ObjectBox serialization fix that unblocked caching (Phase 4), and cache-write-on-download, DioException fallback, silent re-cache, and offline AppBar indicator (Phase 5).
- **v1.2 (this milestone):** Four phases ported the web client's settings screens into Flutter. The v1.2 Settings milestone is now **fully complete** — all 20 requirements delivered across Phases 6–9.

---

## 2. Architecture & Technical Decisions

### State Management
- **Decision:** `settingsProvider` is a synchronous Riverpod `Notifier` returning `Settings?` (not `AsyncValue<Settings?>`).
  - **Why:** Settings are loaded from ObjectBox (local store) synchronously at startup, then updated in the background. Using `AsyncValue` would require `.when()` boilerplate in every screen for a value that is never actually async at the UI layer.
  - **Phase:** 6 (established), 9 (corrected misunderstanding in plan/context — D-11 fix)

- **Decision:** Derived providers (`localeProvider`, `unitProvider`, `themeModeProvider`) follow a shared pattern: `@Riverpod(keepAlive: true)` function-style providers that watch `settingsProvider` and return a typed value with a safe default.
  - **Why:** Keeps MaterialApp.router reactive to settings changes without polling; matches the existing `themeModeProvider` pattern already in the codebase.
  - **Phase:** 6

### Save Pattern (all settings screens)
- **Decision:** Auto-save on selection/toggle — no explicit Save button for any settings sub-screen (except bio).
  - **Why:** Matches the web client UX; avoids unsaved-state bugs; consistent with Appearance screen's immediate `localSettingsProvider` update pattern.
  - **Phase:** 6 (established), 7/8/9 (followed)

- **Decision:** Save via a `_save(WidgetRef, AppLocalizations, Settings)` private helper that calls `settingsProvider.saveToServer(...)` in a try/catch, surfacing `l10n.error_saving_settings` as an error toast on failure.
  - **Why:** Centralizes error handling per screen; copied verbatim from the first screen it was established on (Privacy) to prevent drift.
  - **Phase:** 7 (established), 8/9 (copied verbatim)

### Map Key Convention (Notifications)
- **Decision:** Notification preference map keys are snake_case `@JsonValue` literal strings (e.g., `'trail_comment'`), never `NotificationType.x.name`.
  - **Why:** `.name` returns camelCase (`'trailComment'`), which never matches the server's snake_case keys. Using `.name` would silently default every toggle to `true` and break persistence.
  - **Phase:** 9

### Screen Structure Pattern
- **Decision:** All settings sub-screens are `ConsumerWidget` → `Scaffold` → `AppBar` (back via `context.pop()`) → `ListView` body with `Padding + Text(titleSmall)` section headers.
  - **Why:** Matches the established `SettingsLanguageScreen` pattern for visual and structural consistency across all sub-screens.
  - **Phase:** 6 (established), 7/8/9 (followed)

### Unit Wiring
- **Decision:** Unit preference (metric/imperial) is wired to all `formatDistance`, `formatElevation`, `formatSpeed` call sites via a live `unitProvider` — not hardcoded at the call sites.
  - **Why:** Units toggle must visibly re-render stats across the entire app, including the navigation screen stats sheet and trail detail screens.
  - **Phase:** 6

### Avatar Upload
- **Decision:** Gallery-only (`ImageSource.gallery`), auto-upload on pick (no staging), `image_picker ^1.2.2` as the only new dependency in v1.2.
  - **Why:** Camera permission would increase the permission surface unnecessarily; auto-upload matches the web client's immediate-save UX.
  - **Phase:** 8

### Bio Save
- **Decision:** Bio uses an explicit Save button (enabled only when `controller.text != persisted`), while all other settings auto-save.
  - **Why:** Bio is free-text — auto-saving on every keystroke would flood the server. Save-only-when-changed matches the web client's pattern.
  - **Phase:** 8

### Localizations
- **Decision:** 14 locale ARB files (`en` + 12 ported from web + `de` that already existed), maintained by `flutter gen-l10n` — never hand-edit generated files.
  - **Why:** Ensures all 14 `Language` enum values have a corresponding locale file; gen-l10n synthesizes missing keys from the English template as fallbacks.
  - **Phase:** 6

---

## 3. Phases Delivered

| Phase | Name | Status | What Was Built |
|-------|------|--------|----------------|
| 1 | Backend API | ✓ Complete | SvelteKit POST /api/v1/valhalla/navigate, Zod-validated, returns structured maneuver list |
| 2 | Navigation Screen | ✓ Complete | Full-screen Flutter map, GPS centering, maneuver banner, compass toggle, breadcrumb, entry points |
| 3 | Stats Sheet | ✓ Complete | DraggableScrollableSheet with live distance/elevation/speed stats + elevation-profile PageView |
| 4 | Serialization Fix + Entity Schema | ✓ Complete | Fixed NavigateResponse.toJson() bug, added navCacheJson to TrailEntity for ObjectBox caching |
| 5 | Cache Write + Fallback + UI | ✓ Complete | Cache-write on download, DioException offline fallback, silent re-cache, offline AppBar indicator |
| 6 | Settings Navigation + Language & Units | ✓ Complete | Settings nav wiring (5 entries + routes), Language screen (14-locale RadioGroup + metric/imperial toggle), live locale switch, unit wiring across all call sites, 12 new locale ARB files |
| 7 | Privacy | ✓ Complete | Privacy screen: account/trails/lists visibility controls via RadioGroup auto-save |
| 8 | Account & Profile | ✓ Complete | Account screen: avatar upload (gallery), bio edit + Save, email change sheet, password change sheet, delete account with confirm dialog |
| 9 | Notifications | ✓ Complete | Notifications screen: 9 notification types × 2 toggles (Web + Email) = 18 SwitchListTiles, auto-save, new `"web"` ARB key |

---

## 4. Requirements Coverage

### v1.2 Settings Requirements (20 total — all complete)

**Settings Navigation**
- ✅ **SETNAV-01** — Settings screen lists Account, Privacy, Language, Notifications, Appearance entries (Phase 6)

**Language & Units**
- ✅ **LANG-01** — User can select preferred language from 14 supported locales (Phase 6)
- ✅ **LANG-02** — User can toggle metric/imperial units (Phase 6)

**Privacy**
- ✅ **PRIV-01** — User can set account visibility (Phase 7)
- ✅ **PRIV-02** — User can set trails default visibility (Phase 7)
- ✅ **PRIV-03** — User can set lists default visibility (Phase 7)

**Account & Profile**
- ✅ **ACCT-01** — User can view and update avatar (Phase 8)
- ✅ **ACCT-02** — User can view and edit bio (Phase 8)
- ✅ **ACCT-03** — User can change email address (Phase 8)
- ✅ **ACCT-04** — User can change password (Phase 8)
- ✅ **ACCT-05** — User can delete account with confirmation (Phase 8)

**Notifications**
- ✅ **NOTIF-01** — Web + email toggles for trail comments (Phase 9)
- ✅ **NOTIF-02** — Web + email toggles for new followers (Phase 9)
- ✅ **NOTIF-03** — Web + email toggles for trail shares (Phase 9)
- ✅ **NOTIF-04** — Web + email toggles for trail likes (Phase 9)
- ✅ **NOTIF-05** — Web + email toggles for list shares (Phase 9)
- ✅ **NOTIF-06** — Web + email toggles for summit log creates (Phase 9)
- ✅ **NOTIF-07** — Web + email toggles for trail mentions (Phase 9)
- ✅ **NOTIF-08** — Web + email toggles for comment mentions (Phase 9)
- ✅ **NOTIF-09** — Web + email toggles for summit log mentions (Phase 9)

**Coverage: 20/20 requirements complete. No gaps.**

### Out of Scope (intentional)
| Feature | Reason |
|---------|--------|
| API token management | Mobile clients don't need API tokens; web-only feature |
| Favourite sport picker | Being removed from web; not porting to mobile |
| Export settings | Desktop workflow |
| Integrations (Strava, Komoot) | Complex OAuth; separate milestone |
| Camera permission for avatar | Minimises permission surface; gallery-only is sufficient |

---

## 5. Key Decisions Log

| ID | Decision | Phase | Rationale |
|----|----------|-------|-----------|
| D-01 | Language switch is live (updates app locale immediately) | 6 | Matches web UX; MaterialApp.router gets `locale:` from `localeProvider` |
| D-02 | 12 missing locale ARB files ported from web client JSON | 6 | All 14 Language enum values need a real locale file for the picker to work |
| D-03 | `localeProvider` returns null → Flutter falls back to device locale | 6 | Safe default when no language is saved |
| D-04 | Unit wiring is immediate in Phase 6, not deferred | 6 | Units toggle must visibly update stats across the whole app |
| D-05 | All format_util call sites wired to `unitProvider` | 6 | No hardcoded `'metric'` strings should remain |
| D-06 | Settings row order: Account → Privacy → Language → Notifications → Appearance | 6 | Matches web client order |
| D-07 | Privacy null defaults: account=public, trails=private, lists=private | 7 | Matches web client defaults |
| D-08 | Privacy labels: public/private for account; public/only_me for trails+lists | 7 | Matches web client label choices (not symmetric) |
| D-09 | Avatar: gallery-only, auto-upload on pick, image_picker only new dep | 8 | Minimises permissions; matches web UX |
| D-10 | Bio: explicit Save button, enabled only when text differs from persisted | 8 | Avoids flooding server on every keystroke |
| D-11 | Email/password change via modal bottom sheet, not new route | 8 | No new route needed; matches web's modal pattern |
| D-12 | Notification map keys: snake_case literals, never `.name` | 9 | `.name` returns camelCase, which never matches server keys |
| D-13 | 18 SwitchListTiles (Web + Email per type), not a table | 9 | Matches web client's row-based layout |
| D-14 | All toggles default to `true` when no preference saved | 9 | Web client parity; opt-out model |

---

## 6. Tech Debt & Deferred Items

### Bugs Fixed During v1.2
The following pre-existing bugs were found and fixed during Phase 9 execution (all unrelated to Phase 9's own scope):

- **`password_change_sheet.dart`** — Missing `import 'package:dio/dio.dart'` caused a compile error preventing the account test suite from running (Phase 8 regression, fixed in Phase 9).
- **`settings_language_screen_test.dart`** — Test asserted `SwitchListTile` for the units row but the implementation uses `RadioListTile<String>` via a `RadioGroup` ancestor (Phase 6 test bug, fixed in Phase 9).
- **`settingsProvider.saveToServer()`** — `Settings.id` is `String?` but was interpolated directly into the URL, producing `/settings/null` when id is null. Added a null/empty guard (CR-01 from code review).
- **Dead ARB keys** — `settings_notification_list_create` and `settings_notification_trail_create` were defined in `app_en.arb` but never referenced (no matching `NotificationType` enum value). Removed.

### Known Limitations
- **Notifications toggle-to-save path is untested** (WR-02 from Phase 9 code review) — the auto-save path requires a Dio mock fixture that the current widget test harness doesn't provide. Save is covered by UAT manual testing and the provider-level unit tests.
- **13 locales have untranslated keys** — The 12 newly ported locale ARB files only translate keys that exist in the web client's JSON files. New mobile-only keys (e.g., `"web"`) fall back to English.
- **`RadioGroup.groupValue` deprecated in Flutter 3.32+** — The language screen uses a `RadioGroup` ancestor; the deprecated `RadioListTile.groupValue` property is not accessed in app code, only was briefly hit in a test (now fixed).

### Deferred for v1.3+
- **ACCT-F01** — API token management (generate, view, delete). Mobile use case unclear; web-only feature for now.
- **Offline settings** — Currently requires network to save settings changes. Stale-cache dialogs / "cached N days ago" UI deferred from v1.1 research.
- **Integrations** — Strava, Komoot, Hammerhead OAuth flows are a separate milestone.

---

## 7. Getting Started

### Run the App
```bash
cd app && flutter run          # iOS Simulator or Android Emulator
cd app && flutter run -d <id>  # Specific device (flutter devices to list)
```

### Run Tests
```bash
cd app && flutter test                      # All tests
cd app && flutter test test/routes/         # Settings screen widget tests
cd app && flutter analyze                   # Static analysis (no errors expected)
```

### Key Directories
```
app/lib/
  routes/                      # Screen files (one per route)
    settings_*.dart            # All settings sub-screens added in v1.2
  provider/
    settings_provider.dart     # Central settings state (synchronous Notifier)
    auth_provider.dart         # User auth + UserEntity refresh
  models/settings.dart         # Settings freezed model (NotificationType, NotificationPreference, SettingsPrivacy)
  i18n/app_en.arb              # English ARB template (source of truth for all locales)
  components/settings/         # Bottom-sheet components (email_change_sheet, password_change_sheet)

web/src/routes/api/v1/
  valhalla/navigate/           # SvelteKit navigation endpoint
  settings/                    # Settings API (GET + PATCH)
  user/[id]/                   # User profile endpoints (avatar, email, password, delete)
```

### Where to Look First
- **Settings flow:** `app/lib/routes/settings_screen.dart` → child screens in `settings_*.dart`
- **Settings data model:** `app/lib/models/settings.dart` (freezed — read generated `.freezed.dart` for all fields)
- **Navigation flow:** `app/lib/routes/navigation_screen.dart` + `app/lib/provider/navigation_provider.dart`
- **Route map:** `app/lib/provider/router_provider.dart` (all GoRoutes)
- **API layer:** `web/src/routes/api/v1/` (SvelteKit API endpoints with Zod validation)

---

## Stats

- **Timeline:** 2026-06-19 → 2026-06-21 (3 days)
- **Phases (v1.2):** 4/4 complete (Phases 6–9)
- **Phases (total, all milestones):** 9/9 complete
- **Commits (v1.2):** ~70
- **Files changed (v1.2, app + web source):** 79 files (+27,124 / -461 lines)
- **Requirements delivered:** 20/20 (v1.2) + 15/15 (v1.0 + v1.1) = 35 total
- **Widget tests added (v1.2):** 5 new test files covering all 4 settings sub-screens
- **Locales supported:** 14 (en, cs, de, es, eu, fr, hu, it, nl, no, pl, pt, ru, zh)
