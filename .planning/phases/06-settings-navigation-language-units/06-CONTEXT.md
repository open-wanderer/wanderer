# Phase 6: Settings Navigation + Language & Units - Context

**Gathered:** 2026-06-19
**Status:** Ready for planning

<domain>
## Phase Boundary

Add Privacy, Language, and Notifications list entries (with go_router sub-routes) to the existing SettingsScreen. Build the combined Language & Units screen: a scrollable 14-locale RadioListTile picker that updates the live app locale, plus a SwitchListTile for metric/imperial that wires to all `format_util` call sites. Both preferences persist to the server via `settingsProvider.saveToServer()`.

Requirements covered: SETNAV-01, LANG-01, LANG-02

</domain>

<decisions>
## Implementation Decisions

### Language behavior
- **D-01:** Changing language does a **live locale switch** — saves to server AND updates the Flutter app's displayed UI language. `MaterialApp.router` receives a `locale:` param derived from a new `localeProvider` that watches `settingsProvider`.
- **D-02:** All 14 ARB files must exist. The 12 missing locales (`cs`, `es`, `eu`, `fr`, `hu`, `it`, `nl`, `no`, `pl`, `pt`, `ru`, `zh`) are ported from the web client's existing `web/src/lib/i18n/locales/*.json` files. Map overlapping keys; leave untranslated strings as English fallbacks where no web equivalent exists.
- **D-03:** `localeProvider` is a derived Riverpod provider (annotated with `@riverpod`, keepAlive) that reads `settingsProvider` and returns a `Locale` matching the stored `Language` enum value. Follows the exact same pattern as `themeModeProvider` derives from `localSettingsProvider`.

### Unit display wiring
- **D-04:** Metric/imperial preference is wired **now** — not deferred. A new `unitProvider` (same derived pattern: watches `settingsProvider`, returns `settings?.unit ?? 'metric'`) is used at every `formatDistance`, `formatElevation`, and `formatSpeed` call site across the app.
- **D-05:** Scope of unit wiring: **all `format_util` call sites** — navigation stats sheet, trail detail screens, and any other screen that calls these functions. No hardcoded `'metric'` strings should remain after this phase.

### Settings screen layout
- **D-06:** New entry order (top to bottom): Account → Privacy → Language → Notifications → Appearance → [Divider] → [Logout button]. Flat list, no section headers — matches the current pattern.
- **D-07:** Icons for new entries — all from FontAwesome (matching `FaIcon` usage in the existing list):
  - Privacy → `FontAwesomeIcons.lock`
  - Language → `FontAwesomeIcons.globe`
  - Notifications → `FontAwesomeIcons.bell`

### Language picker UI
- **D-08:** The Language screen uses a `RadioGroup` + scrollable `ListView` of `RadioListTile` widgets — identical pattern to `SettingsAppearanceScreen`. No new components.
- **D-09:** Each locale label is the **native name only**: Čeština, Deutsch, English, Español, Euskara, Français, Magyar, Italiano, Nederlands, Norsk, Polski, Português, Русский, 中文. These are hardcoded in the screen (no ARB key needed for the native name of each language).
- **D-10:** Metric/imperial toggle lives on the **same screen as the language picker** (matching the phase name "Language & Units"). A `Text` section label "Units" separates the two groups. The units section uses a `SwitchListTile` (metric = on, imperial = off) — not RadioListTile.

### Routes to add (go_router)
- **D-11:** Add three new `GoRoute` entries under `/settings` in `router_provider.dart`:
  - `path: 'privacy'` → `SettingsPrivacyScreen` (stub — built in Phase 7)
  - `path: 'language'` → `SettingsLanguageScreen` (built this phase)
  - `path: 'notifications'` → `SettingsNotificationsScreen` (stub — built in Phase 9)

### Save behavior
- **D-12:** On the Language screen, save happens via `settingsProvider.saveToServer()` when the user picks a locale or toggles units. No explicit "Save" button — changes auto-save on selection (matches Appearance screen's immediate ThemeMode update via `localSettingsProvider`).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Existing settings infrastructure
- `app/lib/models/settings.dart` — `Settings` freezed model, `Language` enum (14 values), `SettingsPrivacy`, `NotificationPreference`
- `app/lib/provider/settings_provider.dart` — `SettingsNotifier`: `saveToServer()`, `updateFromServer()`, ObjectBox persistence
- `app/lib/entities/settings_entity.dart` — ObjectBox entity; `languageCode` and `unit` fields already exist

### Existing screen patterns (must follow)
- `app/lib/routes/settings_screen.dart` — current SettingsScreen; new entries slot in here
- `app/lib/routes/settings_appearance_screen.dart` — RadioGroup + RadioListTile pattern; Language screen copies this structure
- `app/lib/provider/local_settings_provider.dart` — `themeModeProvider` derivation pattern; `localeProvider` and `unitProvider` follow this exact shape

### Router
- `app/lib/provider/router_provider.dart` — GoRoute tree; add `privacy`, `language`, `notifications` sub-routes under `/settings`

### Format utilities (unit wiring)
- `app/lib/util/format_util.dart` — `formatDistance(meters, {unit})`, `formatElevation(meters, {unit})`, `formatSpeed(kmh, {unit})`; replace all hardcoded `unit: 'metric'` callers with `unit: ref.watch(unitProvider)`

### Internationalization
- `app/lib/i18n/app_en.arb` — source of truth for Flutter string keys; 490 lines
- `app/lib/i18n/app_localizations.dart` — generated delegate; `supportedLocales` currently only `en`, `de`
- `app/lib/main.dart` — `supportedLocales` and `localizationsDelegates` in `MaterialApp.router`; `locale:` param to be added here
- `web/src/lib/i18n/locales/` — 14 JSON files (cs, de, en, es, eu, fr, hu, it, nl, no, pl, pt, ru, zh) — source for porting translations to the 12 missing Flutter ARB files

### Requirements
- `.planning/REQUIREMENTS.md` — SETNAV-01, LANG-01, LANG-02 (the three requirements this phase covers)
- `.planning/ROADMAP.md` §Phase 6 — success criteria and phase goal

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `SettingsAppearanceScreen` (`RadioGroup` + `RadioListTile` pattern): Language picker is structurally identical — copy and adapt
- `SettingsNotifier.saveToServer()`: handles both language and unit saves; no new API calls needed
- `format_util.dart` `formatDistance/Elevation/Speed`: already accept `unit` param — just need callers updated
- `LocalSettingsEntity` + `localSettingsProvider`: template for the `localeProvider` and `unitProvider` derived providers

### Established Patterns
- Derived Riverpod providers: `@riverpod`, `keepAlive: true`, watches a parent provider → returns computed value. Used for `themeModeProvider`; replicate for `localeProvider` and `unitProvider`
- `FaIcon(FontAwesomeIcons.X, size: 18)` for `ListTile.leading` in SettingsScreen
- `context.push('/settings/X')` in `ListTile.onTap` for navigation
- `ref.read(authProvider.notifier).logout()` for the logout button — unaffected

### Integration Points
- `app/lib/main.dart` `MaterialApp.router`: add `locale: ref.watch(localeProvider)` and expand `supportedLocales` to all 14 locales
- `app/lib/provider/router_provider.dart`: add 3 GoRoute entries under the `/settings` parent
- All `formatDistance/Elevation/Speed` call sites: pass `unit: ref.watch(unitProvider)` (or pass via constructor/parameter depending on widget type)
- `app/lib/i18n/` directory: add 12 new `.arb` files, regenerate via `flutter gen-l10n`

### Stub screens needed
- `app/lib/routes/settings_privacy_screen.dart` — minimal Scaffold (Phase 7 will fill it)
- `app/lib/routes/settings_notifications_screen.dart` — minimal Scaffold (Phase 9 will fill it)

</code_context>

<specifics>
## Specific Details

- Language native name labels (hardcoded in screen, not in ARB): `cs` → Čeština, `en` → English, `de` → Deutsch, `es` → Español, `eu` → Euskara, `fr` → Français, `hu` → Magyar, `it` → Italiano, `nl` → Nederlands, `no` → Norsk, `pl` → Polski, `pt` → Português, `ru` → Русский, `zh` → 中文
- Units section label on Language screen: use ARB key `units` (already exists: "Units")
- `SwitchListTile` for units: `value: settings?.unit == 'imperial'`, `onChanged` saves via `settingsProvider`; title uses `metric`/`imperial` ARB keys
- `localeProvider` return type: `Locale?` — returning `null` lets Flutter fall back to device locale; returning `Locale(settings.language.name)` for a matched language
- Settings entry label keys: `privacy`, `language`, `notifications` — all exist in `app_en.arb`

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 6-Settings Navigation + Language & Units*
*Context gathered: 2026-06-19*
