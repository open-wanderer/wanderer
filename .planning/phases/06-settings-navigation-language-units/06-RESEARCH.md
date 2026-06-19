# Phase 6: Settings Navigation + Language & Units - Research

**Researched:** 2026-06-19
**Domain:** Flutter (Material 3) settings UI, Riverpod derived providers, go_router sub-routes, Flutter gen-l10n internationalization (ARB), unit-system display wiring
**Confidence:** HIGH

## Summary

This phase is almost entirely **UI + wiring on top of existing infrastructure** — no new packages, no new persistence layer, no new API endpoints. Every model, provider, entity, format utility, screen pattern, ARB key, and go_router parent route this phase needs already exists in the codebase and was verified directly during research. The work is: (1) add 3 list rows + 3 sub-routes, (2) build one combined Language & Units screen by copying `SettingsAppearanceScreen`, (3) add two derived Riverpod providers (`localeProvider`, `unitProvider`) cloned from the existing `themeModeProvider` pattern, (4) wire `unitProvider` into ~50 `format_util` call sites across ~16 files, and (5) port 12 locale files from the web client's JSON into Flutter ARB files.

Two items carry the real risk and effort. First, the **i18n port (D-02)**: the web client stores translations as **kebab-case JSON keys** (`account-delete-confirm`) while Flutter ARB uses **snake_case** (`account_delete_confirm`). The mapping is mostly mechanical (`-` → `_`) and ICU plural syntax is byte-identical between the two systems, but the web JSON has 573 keys vs the ARB's 484, so not every web key maps and not every ARB key has a web source — untranslated strings fall back to English automatically via gen-l10n's default behavior, which matches D-02. Second, the **unit-wiring scope (D-05)**: of the 14 files containing `format_util` calls, **3 are plain `StatelessWidget`/`StatefulWidget` with no `ref`** (`elevation_profile.dart`, `waypoint_sheet.dart`, `trail_timeline.dart`) — these need conversion to Consumer variants or a parameter-passing strategy, which is the only non-trivial structural change in the phase.

One correctness note the planner must build in: `settingsProvider.saveToServer()` has **no internal try/catch** and `settingsProvider` state can be `null` before login completes — the Language screen's auto-save must guard both.

**Primary recommendation:** Build by cloning existing patterns verbatim — `themeModeProvider` → `localeProvider`/`unitProvider`, `SettingsAppearanceScreen` → `SettingsLanguageScreen`, existing `ListTile` rows → 3 new rows. Treat the i18n port and the 3 non-Consumer unit call-sites as the two tasks needing the most care; everything else is mechanical.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Settings list navigation | Mobile Frontend (Flutter screens) | — | Pure client navigation via go_router `context.push` |
| Language selection persistence | Mobile Providers (Riverpod) → SvelteKit API → Backend | Mobile (ObjectBox cache) | `settingsProvider.saveToServer()` POSTs to `/settings/:id`; server is source of truth, ObjectBox is local cache |
| Unit selection persistence | Mobile Providers → SvelteKit API → Backend | Mobile (ObjectBox cache) | Same path as language — `unit` field on `Settings` model |
| Live UI locale switch | Mobile Frontend (`MaterialApp.router`) | Mobile Providers (`localeProvider`) | `locale:` param re-renders the widget tree client-side; no server round-trip needed for the visual switch |
| Unit display formatting | Mobile Frontend (`format_util` + call-site widgets) | Mobile Providers (`unitProvider`) | Formatting is presentation logic; provider supplies the current preference reactively |
| Translation strings | Mobile (ARB → generated `AppLocalizations`) | — | Compile-time generated; no runtime tier involved |

**No backend, API-route, or database schema work in this phase.** The `unit` and `languageCode` fields already exist on both the `Settings` freezed model and the `SettingsEntity` ObjectBox entity. `saveToServer()` POSTs to the existing `/settings/:id` endpoint. [VERIFIED: codebase — `app/lib/provider/settings_provider.dart`, `app/lib/entities/settings_entity.dart`]

## Standard Stack

No new dependencies. Every capability is covered by packages already in `app/pubspec.yaml`. [VERIFIED: codebase — `app/pubspec.yaml`, import statements in referenced files]

### Core (all pre-existing)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| flutter_riverpod | 3.3.1 | State + derived providers (`localeProvider`, `unitProvider`) | Existing project state-management standard; `themeModeProvider` is the exact template |
| riverpod_annotation / riverpod_generator | 3.x / 4.0.3 | `@Riverpod(keepAlive: true)` codegen for derived providers | All providers in the app use codegen; `local_settings_provider.dart` shows the pattern |
| go_router | 17.2.1 | `/settings/{privacy,language,notifications}` sub-routes | Existing routing standard; `/settings` parent + `account`/`appearance` children already present |
| flutter_localizations + intl | SDK / any | gen-l10n ARB → `AppLocalizations` | Already configured via `l10n.yaml` + `generate: true` in pubspec |
| font_awesome_flutter | 11.0.0 | `FaIcon` leading icons for new rows | Existing `SettingsScreen` rows use `FaIcon(size: 18)` |
| freezed / json_serializable | 3.2.5 / 6.13.0 | `Settings.copyWith()` for building updated settings before save | `Settings` is already a freezed model; `copyWith` is generated |
| objectbox | 5.3.1 | Local settings cache (already wired through `settingsProvider`) | No new entity needed; `unit`/`languageCode` fields exist |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Material 3 widgets | Flutter SDK | `RadioGroup`, `RadioListTile`, `SwitchListTile`, `ListTile`, `Scaffold`, `AppBar`, `Divider` | All UI in this phase; `RadioGroup<T>` confirmed available and in use in `settings_appearance_screen.dart` |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Cloning `themeModeProvider` for unit/locale | A single combined `preferencesProvider` | More refactoring, breaks the established one-provider-per-derived-value pattern; NOT recommended — D-03/D-04 explicitly mandate the cloned pattern |
| gen-l10n ARB | Runtime JSON loading (like the web client) | Loses compile-time key safety and the generated typed `AppLocalizations` API the app already uses; NOT recommended |

**Installation:** None required. Run `dart run build_runner build --delete-conflicting-outputs` after adding the two new providers (Riverpod codegen) and `flutter gen-l10n` after adding ARB files.

**Version verification:** Flutter `3.41.9` (stable, framework revision 2026-04-29) is the installed SDK — **newer than the `3.11.5` documented in CLAUDE.md**. [VERIFIED: `flutter --version`] This matters because `RadioGroup<T>` is a relatively recent Material widget; it is confirmed present and already compiled in `settings_appearance_screen.dart` and `trail_filter_screen.dart`, so no API-availability risk. [VERIFIED: codebase grep — `RadioGroup` usage]

## Package Legitimacy Audit

> No external packages are installed in this phase. All libraries listed in Standard Stack are pre-existing project dependencies, already vetted and present in `app/pubspec.yaml`.

| Package | Registry | Disposition |
|---------|----------|-------------|
| (none) | — | No new installs — audit not applicable |

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
User taps "Language" row in SettingsScreen
        │  context.push('/settings/language')
        ▼
go_router → SettingsLanguageScreen (ConsumerWidget)
        │
        ├── watches settingsProvider ──► current Language + unit (from ObjectBox cache / server)
        │
        ├── RadioGroup<String> (14 locales)
        │       onChanged(locale)
        │            │
        │            ▼
        │       settingsProvider.saveToServer(settings.copyWith(language: ...))
        │            │
        │            ├──► POST /settings/:id (SvelteKit API) ──► Backend (source of truth)
        │            └──► updateFromServer() ──► ObjectBox put + invalidateSelf
        │                        │
        │                        ▼
        │              localeProvider re-derives Locale
        │                        │
        │                        ▼
        │              MaterialApp.router locale: ──► whole tree re-renders in new language
        │
        └── SwitchListTile (metric/imperial)
                onChanged(isImperial)
                     │
                     ▼
                settingsProvider.saveToServer(settings.copyWith(unit: ...))
                     │
                     ▼
                unitProvider re-derives 'metric'|'imperial'
                     │
                     ▼
        ~50 format_util call sites read ref.watch(unitProvider) ──► distances/elevations/speeds re-render
```

### Recommended Project Structure (new/modified files)
```
app/lib/
├── routes/
│   ├── settings_screen.dart              # MODIFY: add Privacy/Language/Notifications rows (D-06, D-07)
│   ├── settings_language_screen.dart     # NEW: combined Language & Units screen (D-08..D-12)
│   ├── settings_privacy_screen.dart      # NEW: stub Scaffold + AppBar (Phase 7 fills it)
│   └── settings_notifications_screen.dart# NEW: stub Scaffold + AppBar (Phase 9 fills it)
├── provider/
│   ├── settings_provider.dart            # add localeProvider + unitProvider derived providers (D-03, D-04)
│   │                                     #   OR a new file; follow local_settings_provider.dart layout
│   └── router_provider.dart              # MODIFY: 3 GoRoute children under /settings (D-11)
├── main.dart                             # MODIFY: locale: ref.watch(localeProvider) + 14 supportedLocales (D-01)
├── i18n/
│   ├── app_en.arb                        # unchanged (template, source of truth)
│   ├── app_de.arb                        # unchanged (already exists)
│   └── app_{cs,es,eu,fr,hu,it,nl,no,pl,pt,ru,zh}.arb  # NEW ×12 (ported from web JSON, D-02)
└── util/format_util.dart                 # unchanged (already accepts unit: param)
    # ~14 call-site files MODIFIED to pass unit: ref.watch(unitProvider)
```

### Pattern 1: Derived keepAlive provider (clone of `themeModeProvider`)
**What:** A `@Riverpod(keepAlive: true)` function provider that watches a parent state provider and returns a computed value.
**When to use:** For both `localeProvider` and `unitProvider` (D-03, D-04).
**Example:**
```dart
// Source: app/lib/provider/local_settings_provider.dart (VERIFIED existing pattern)
@Riverpod(keepAlive: true)
ThemeMode themeMode(Ref ref) {
  final settings = ref.watch(localSettingsProvider);
  return switch (settings.themeMode) {
    'light' => ThemeMode.light,
    'dark' => ThemeMode.dark,
    _ => ThemeMode.system,
  };
}

// NEW — localeProvider (returns Locale? so null falls back to device locale per D-spec)
@Riverpod(keepAlive: true)
Locale? locale(Ref ref) {
  final settings = ref.watch(settingsProvider);
  final lang = settings?.language;
  return lang == null ? null : Locale(lang.name);
}

// NEW — unitProvider
@Riverpod(keepAlive: true)
String unit(Ref ref) {
  final settings = ref.watch(settingsProvider);
  return settings?.unit ?? 'metric';
}
```
Note: `Language.values` `.name` returns the lowercase enum name (`cs`, `de`, …) which equals the locale code — verified against the `@JsonValue` annotations in `settings.dart`. [VERIFIED: codebase — `app/lib/models/settings.dart`]

### Pattern 2: RadioGroup + RadioListTile picker (clone of `SettingsAppearanceScreen`)
**What:** `RadioGroup<T>` wrapping a `ListView` of `RadioListTile<T>`; selection calls a notifier.
**When to use:** The 14-locale language picker (D-08).
**Example:**
```dart
// Source: app/lib/routes/settings_appearance_screen.dart (VERIFIED existing pattern)
final activeColor = Theme.of(context).brightness == Brightness.dark
    ? colorScheme.onSurface
    : colorScheme.primary;

RadioGroup<Language>(
  groupValue: settings?.language,
  onChanged: (value) {
    if (value != null && settings != null) {
      ref.read(settingsProvider.notifier)
         .saveToServer(settings.copyWith(language: value));
    }
  },
  child: ListView(children: [ /* 14 RadioListTile<Language> with native-name titles */ ]),
)
```
The UI-SPEC (D-08, line 121) mentions `RadioGroup<String>`; using `RadioGroup<Language>` with the enum is cleaner and matches the model — either works, but the enum avoids string-matching bugs. Flag for planner: pick one and be consistent. [CITED: 06-UI-SPEC.md]

### Pattern 3: Auto-save SwitchListTile for units (D-10, D-12)
```dart
SwitchListTile(
  title: Text(l10n.metric),   // or two labels; D-10 says metric=on
  value: settings?.unit != 'imperial',  // metric = on per D-10
  onChanged: (isMetric) {
    if (settings != null) {
      ref.read(settingsProvider.notifier)
         .saveToServer(settings.copyWith(unit: isMetric ? 'metric' : 'imperial'));
    }
  },
)
```
Note: D-10/specifics describe `value: settings?.unit == 'imperial'` (imperial=on). The exact polarity is a planner decision — the spec text is internally slightly inconsistent (D-10 says "metric = on, imperial = off" in line 37 but line 115 says `value: settings?.unit == 'imperial'`). **Flag: planner must lock the switch polarity and label.** [CITED: 06-CONTEXT.md D-10 + specifics]

### Anti-Patterns to Avoid
- **Calling `saveToServer` with a null `settings`:** `settingsProvider` returns `null` until login populates it via `auth_provider.dart`. Guard every save with `if (settings != null)`. [VERIFIED: codebase — `settings_provider.dart build()` returns `Settings?`]
- **Unguarded `saveToServer` await:** the method has no try/catch and will throw `DioException` on network failure. Wrap in try/catch and surface via the existing toast mechanism (do not let it bubble as an unhandled async error). [VERIFIED: codebase — `settings_provider.dart` has no error handling]
- **Hardcoding `fontSize` or colors:** UI-SPEC mandates `TextTheme` roles and `colorScheme` only. [CITED: 06-UI-SPEC.md]
- **Localizing native language names:** native names are intentionally hardcoded (D-09) — do NOT add ARB keys for them.
- **Overriding ListTile internal padding:** accept Material defaults to match existing screens (UI-SPEC Spacing). [CITED: 06-UI-SPEC.md]

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Reactive locale/unit preference | Custom InheritedWidget or global singleton | Cloned `@riverpod` derived provider | Project standard; reactivity + keepAlive handled by Riverpod |
| Server persistence of settings | New API call / new endpoint | `settingsProvider.saveToServer()` | Already POSTs to `/settings/:id` and syncs ObjectBox |
| Local settings cache | New ObjectBox entity | `SettingsEntity` (`unit`, `languageCode` fields exist) | Fields already present and mapped |
| Distance/elevation/speed conversion | Inline math at call sites | `format_util` `formatDistance/Elevation/Speed(.., unit:)` | Conversion + null/NaN guards already implemented |
| Translation loading | Runtime JSON loader | gen-l10n ARB → `AppLocalizations` | Compile-time typed API already wired |
| Language radio group | Custom selectable list | Material `RadioGroup` + `RadioListTile` | Exact pattern in `SettingsAppearanceScreen` |

**Key insight:** This phase's entire value is *consistency with existing patterns*. Every "new" thing is a copy of something that already works. The planner should resist any task that introduces a novel abstraction.

## Runtime State Inventory

> This phase adds new screens/providers and ports i18n files. It is partly a wiring/refactor phase (D-05 touches ~50 call sites), so the inventory applies.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | `SettingsEntity` in ObjectBox already has `unit` and `languageCode` columns; populated at login. No schema change. Existing cached records may have `unit == null` → `unitProvider` defaults to `'metric'`. | None — `?? 'metric'` fallback handles null. No data migration. |
| Live service config | None — no external service config embeds language/unit strings. Server `/settings/:id` already accepts these fields (web client writes them). | None — verified server contract unchanged (web `de.json` confirms server-side `language`/`unit` already in use). |
| OS-registered state | None — no OS-level registrations involved. | None — verified (pure in-app feature). |
| Secrets/env vars | None. | None. |
| Build artifacts | gen-l10n generates `app_localizations.dart` + per-locale `app_localizations_*.dart`; Riverpod codegen generates `*.g.dart` for the 2 new providers. Both are committed generated artifacts. | Run `flutter gen-l10n` and `dart run build_runner build --delete-conflicting-outputs`; commit regenerated files. ObjectBox model unchanged (no new entity), so **no `objectbox-model.json` regeneration needed**. |

**Canonical question — after every file is updated, what runtime state still holds the old value?** Nothing problematic. The only "old state" is ObjectBox-cached `Settings` with null `unit`/`language`, which the providers' null-coalescing fallback handles gracefully. No migration task required.

## Common Pitfalls

### Pitfall 1: `settingsProvider` is null before login / on fresh state
**What goes wrong:** Building `SettingsLanguageScreen` assuming non-null `Settings`; `saveToServer` called on null → NPE or crash.
**Why it happens:** `SettingsNotifier.build()` returns `box.getAll().firstOrNull?.toModel()` which is `null` until `auth_provider` calls `updateFromServer`.
**How to avoid:** Watch `settingsProvider`, guard every read/save with `if (settings != null)`, and render a sensible default selection (English / metric) when null.
**Warning signs:** Screen reached before authentication, or first run with empty ObjectBox.
[VERIFIED: codebase — `settings_provider.dart`, `auth_provider.dart:136`]

### Pitfall 2: `saveToServer` throws on network failure (no error handling)
**What goes wrong:** Auto-save on selection (D-12) fails offline → unhandled `DioException`, possibly leaving the radio in an inconsistent visual state.
**Why it happens:** `saveToServer` awaits `apiProvider.post(...)` with no try/catch.
**How to avoid:** Wrap the save call in try/catch in the screen; on failure show a toast (existing `ToastOverlay` mechanism) and let the watched provider keep the prior value. Note the optimistic-update expectation in the UI-SPEC interaction contract.
**Warning signs:** Selecting a language while offline.
[VERIFIED: codebase — `settings_provider.dart` saveToServer]

### Pitfall 3: i18n key-format mismatch (kebab vs snake) during the port
**What goes wrong:** Copying web JSON values into ARB without renaming keys → gen-l10n can't match the template keys, untranslated strings silently fall back to English, or build fails on malformed keys.
**Why it happens:** Web `en.json` uses `account-delete-confirm`; ARB template uses `account_delete_confirm`. Web has 573 keys, ARB has 484 — non-overlapping sets exist in both directions.
**How to avoid:** For each ARB key in `app_en.arb`, look up the web JSON value under the kebab-cased equivalent; if found, copy the translated value; if not, omit the key (gen-l10n auto-fills from template = English, satisfying D-02). Do the lookup keyed off the **ARB** key set (484), not the web set. Preserve `@@locale` header and ICU plural syntax verbatim (it is identical between the two systems — verified on the `activity` plural key).
**Warning signs:** `flutter gen-l10n` warnings about untranslated messages; runtime English text in a non-English locale where a translation existed.
[VERIFIED: codebase — key format diff between `web/.../en.json` and `app/lib/i18n/app_en.arb`; ICU plural identical on `activity` key]

### Pitfall 4: Unit wiring at non-Consumer call sites
**What goes wrong:** Trying to call `ref.watch(unitProvider)` inside `formatDistance(...)` arguments in a `StatelessWidget`/`StatefulWidget` that has no `ref`.
**Why it happens:** 3 of the 14 call-site files have no Riverpod `ref`: `elevation_profile.dart` (StatefulWidget), `waypoint_sheet.dart` (StatelessWidget), `trail_timeline.dart` (StatelessWidget + nested `_TimelineRow`).
**How to avoid:** Either (a) convert those widgets to `ConsumerWidget`/`ConsumerStatefulWidget`, or (b) pass `unit` down as a constructor parameter from a Consumer parent. Decide per-widget — conversion is cleaner for leaf widgets, parameter-passing avoids touching deeply nested private widgets. **This is the only structurally non-trivial part of the unit-wiring task and deserves its own plan task.**
**Warning signs:** Compile error "The method 'watch' isn't defined" / "Undefined name 'ref'".
[VERIFIED: codebase grep — ConsumerWidget detection across all 14 call-site files]

### Pitfall 5: Forgetting to expand `supportedLocales` in MaterialApp
**What goes wrong:** Setting `locale: Locale('fr')` while `supportedLocales` only lists `en`/`de` → Flutter ignores the unsupported locale and falls back to the first supported one; the live switch appears broken.
**Why it happens:** `main.dart` currently hardcodes `supportedLocales: const [Locale('en'), Locale('de')]`.
**How to avoid:** Expand to all 14 locales (or use `AppLocalizations.supportedLocales` once the 12 ARB files are generated — the generated list auto-updates). Add `locale: ref.watch(localeProvider)` alongside.
**Warning signs:** Picking a non-en/de language doesn't change UI text.
[VERIFIED: codebase — `main.dart:65`, generated `app_localizations.dart` only lists de/en]

## Code Examples

### Adding the 3 new settings rows (D-06, D-07)
```dart
// Source: app/lib/routes/settings_screen.dart (existing row pattern, VERIFIED)
ListTile(
  leading: const FaIcon(FontAwesomeIcons.lock, size: 18),   // Privacy
  title: Text(l10n.privacy),
  trailing: const Icon(Icons.chevron_right),
  onTap: () => context.push('/settings/privacy'),
),
ListTile(
  leading: const FaIcon(FontAwesomeIcons.globe, size: 18),  // Language
  title: Text(l10n.language),
  trailing: const Icon(Icons.chevron_right),
  onTap: () => context.push('/settings/language'),
),
ListTile(
  leading: const FaIcon(FontAwesomeIcons.bell, size: 18),   // Notifications
  title: Text(l10n.notifications),
  trailing: const Icon(Icons.chevron_right),
  onTap: () => context.push('/settings/notifications'),
),
// Final order (D-06): Account → Privacy → Language → Notifications → Appearance → Divider → Logout
```

### Adding the 3 go_router children (D-11)
```dart
// Source: app/lib/provider/router_provider.dart (existing /settings subtree, VERIFIED)
GoRoute(
  path: '/settings',
  builder: (context, state) => const SettingsScreen(),
  routes: [
    GoRoute(path: 'account',  builder: (c, s) => const SettingsAccountScreen()),
    GoRoute(path: 'privacy',  builder: (c, s) => const SettingsPrivacyScreen()),       // NEW
    GoRoute(path: 'language', builder: (c, s) => const SettingsLanguageScreen()),      // NEW
    GoRoute(path: 'notifications', builder: (c, s) => const SettingsNotificationsScreen()), // NEW
    GoRoute(path: 'appearance', builder: (c, s) => const SettingsAppearanceScreen()),
  ],
),
```

### Wiring a unit call site (Consumer case)
```dart
// In a ConsumerWidget/ConsumerStatefulWidget build with `ref` in scope:
final unit = ref.watch(unitProvider);
// ...
label: formatDistance(trail.distance, unit: unit),
label: formatElevation(trail.elevationGain, unit: unit),
```

### main.dart locale wiring (D-01)
```dart
// Source: app/lib/main.dart (VERIFIED) — additions:
return MaterialApp.router(
  localizationsDelegates: AppLocalizations.localizationsDelegates, // or keep explicit list
  supportedLocales: AppLocalizations.supportedLocales,             // auto-includes all 14 after gen-l10n
  locale: ref.watch(localeProvider),                              // NEW — live switch
  themeMode: ref.watch(themeModeProvider),
  // ...
);
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `Radio`/`RadioListTile` with per-tile `groupValue`+`onChanged` | `RadioGroup<T>` ancestor managing shared `groupValue`/`onChanged` | Recent Flutter (Material 3) — already adopted in this codebase | Use `RadioGroup` as in `SettingsAppearanceScreen`; do not hand-wire individual radios |

**Deprecated/outdated:**
- CLAUDE.md lists Flutter `3.11.5+`; the installed toolchain is `3.41.9`. Treat `3.41.9` as authoritative for API availability. [VERIFIED: `flutter --version`]

## Project Constraints (from CLAUDE.md)

- **Tech stack locked:** Flutter + Riverpod (riverpod_annotation codegen) + go_router + flutter_map + freezed — follow existing patterns. ✓ This phase adds nothing outside that set.
- **No breaking changes:** existing trail detail screens, bottom nav, routes must be unaffected. ✓ New routes are additive; Success Criterion 4 explicitly requires Settings/Account/Appearance remain reachable.
- **API constraint:** new endpoints only via SvelteKit; Flutter calls via dio. ✓ This phase adds **no** new endpoints — reuses `/settings/:id`.
- **Naming:** Dart screens PascalCase (`SettingsLanguageScreen`); provider files `_provider.dart` → `_provider.g.dart`; entity files `_entity.dart`; camelCase functions; private `_`-prefixed. Files snake_case.
- **Dart style:** strong typing, null-safety, full annotations, 2-space indent, `flutter_lints` baseline (`analysis_options.yaml` includes `package:flutter_lints/flutter.yaml`).
- **Error handling:** `AsyncValue.guard()` / try-catch for async; toast for user feedback. Apply to `saveToServer` calls.
- **i18n:** mobile localization lives in `app/lib/i18n/`; message keys referenced in components — no hardcoded English (native language names are the sole approved exception per D-09).
- **GSD workflow:** all edits go through a GSD command (this phase is `/gsd-execute-phase`).

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SETNAV-01 | Settings screen lists entries for Account, Privacy, Language, Notifications, and Appearance | 3 new `ListTile` rows + 3 `GoRoute` children + 2 stub screens. Pattern verified in `settings_screen.dart` / `router_provider.dart`. All label ARB keys (`privacy`, `language`, `notifications`) already exist. |
| LANG-01 | User can select preferred language from 14 supported locales | `SettingsLanguageScreen` with `RadioGroup<Language>` (14 `Language` enum values verified in `settings.dart`); persists via `settingsProvider.saveToServer`; live switch via `localeProvider` + `MaterialApp.locale`. Requires 12 new ARB files (D-02). |
| LANG-02 | User can toggle metric/imperial units | `SwitchListTile` on the same screen; persists via `saveToServer`; `unitProvider` wired into ~50 `format_util` call sites across ~16 files (3 require Consumer conversion). `unit` field exists on model + entity. |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Every web JSON key maps to its ARB key by replacing `-` with `_`, and ICU plural syntax is identical | Pitfall 3, Summary | Some keys may not follow the mechanical mapping (e.g. abbreviations); spot-checks confirm the pattern but the full 484-key port should be verified per-key during execution. Verified on sample keys only, not all 484. |
| A2 | The server `/settings/:id` endpoint accepts `language` and `unit` and the web client already writes them | Runtime State Inventory | If server rejects these fields the save fails. Strongly supported (web `de.json` localizes `language`/`units` UI and the web settings screen writes them) but not directly verified against the SvelteKit/PocketBase schema this session. |
| A3 | gen-l10n default behavior fills missing keys from the template (English fallback) without erroring | Pitfall 3, D-02 | If `l10n.yaml` ever enables strict untranslated handling, the build could warn/fail. Current `l10n.yaml` has no such option set — default fallback applies. |
| A4 | `Locale(language.name)` is sufficient for all 14 locales (no region subtags / `zh` script variants needed) | Pattern 1 | `zh` without script subtag (`zh-Hans`/`zh-Hant`) may pick an unexpected default Chinese variant. Low risk for v1 but worth confirming the desired Chinese variant. |

## Open Questions

1. **Switch polarity & label for units (D-10 internal inconsistency)**
   - What we know: D-10 line 37 says "metric = on, imperial = off"; specifics line 115 says `value: settings?.unit == 'imperial'` (imperial = on).
   - What's unclear: which way the switch reads, and whether it shows one label or two.
   - Recommendation: Planner locks one polarity. Suggest metric=on (matches the app's metric default and reads "metric on/off"), single `l10n.metric` title — but defer to planner/UI consistency.

2. **`RadioGroup<String>` vs `RadioGroup<Language>`**
   - What we know: UI-SPEC says `<String>`; the model exposes a `Language` enum.
   - What's unclear: which type the picker uses.
   - Recommendation: Use `RadioGroup<Language>` to avoid string-matching bugs; map enum → native-name label via a hardcoded const map.

3. **Chinese locale variant (A4)**
   - What we know: enum value is `zh`; native label is 中文.
   - What's unclear: Simplified vs Traditional. Web client uses `zh.json` (single file).
   - Recommendation: Match the web client — single `zh` ARB ported from `web/.../zh.json`; revisit if users report wrong script.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Flutter SDK | Build, gen-l10n, codegen | ✓ | 3.41.9 stable | — |
| `flutter gen-l10n` | ARB → AppLocalizations | ✓ (bundled with SDK) | — | — |
| `dart run build_runner` | Riverpod provider codegen | ✓ (build_runner 2.13.1 in pubspec) | 2.13.1 | — |
| Web locale JSON source files | i18n port (D-02) | ✓ | all 14 present in `web/src/lib/i18n/locales/` | — |
| font_awesome_flutter | New row icons | ✓ | 11.0.0 | — |

**Missing dependencies with no fallback:** none
**Missing dependencies with fallback:** none

## Validation Architecture

> `.planning/config.json` was not found during research. Treating `nyquist_validation` as enabled (default). Note: this is a Flutter mobile UI phase; much of the value is visual/interaction and best validated by widget tests + manual smoke, not unit tests.

### Test Framework
| Property | Value |
|----------|-------|
| Framework | `flutter_test` (SDK, bundled) |
| Config file | none — `flutter test` uses `test/` convention |
| Quick run command | `flutter test test/<file>_test.dart` |
| Full suite command | `flutter test` (from `app/`) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SETNAV-01 | Settings list renders 5 rows + navigates | widget | `flutter test test/routes/settings_screen_test.dart` | ❌ Wave 0 |
| LANG-01 | Language picker shows 14 tiles; selecting persists & switches locale | widget | `flutter test test/routes/settings_language_screen_test.dart` | ❌ Wave 0 |
| LANG-02 | Unit switch toggles & `unitProvider` reflects change | unit/widget | `flutter test test/provider/unit_provider_test.dart` | ❌ Wave 0 |
| LANG-02 | `formatDistance/Elevation/Speed` honor unit param | unit | `flutter test test/util/format_util_test.dart` | ❓ check existing |

### Sampling Rate
- **Per task commit:** `flutter analyze` + the relevant `flutter test test/<file>`
- **Per wave merge:** `flutter test` (full suite) + `flutter analyze`
- **Phase gate:** `flutter analyze` clean + `flutter test` green + `flutter gen-l10n` + `build_runner` succeed before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `test/provider/unit_provider_test.dart` — covers LANG-02 (provider derivation + null fallback)
- [ ] `test/routes/settings_language_screen_test.dart` — covers LANG-01 (14 tiles, selection persists)
- [ ] `test/routes/settings_screen_test.dart` — covers SETNAV-01 (row count, navigation)
- [ ] Verify `format_util` already has a test; if not, add `test/util/format_util_test.dart` for imperial conversions
- [ ] Confirm whether `app/test/` exists and what helper/ProviderScope override fixtures are available (research did not enumerate `test/`)

*Manual smoke remains essential for this phase:* live locale switch and unit re-render across screens are visual behaviors best confirmed by running the app.

## Security Domain

> No `security_enforcement` config found (treating as enabled). This phase has minimal security surface: no auth changes, no new endpoints, no new input parsing beyond enum-bounded selections.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | No auth logic touched; existing PocketBase session unchanged |
| V3 Session Management | no | Unchanged |
| V4 Access Control | no | Settings already scoped to authenticated user via existing `/settings/:id` |
| V5 Input Validation | yes (minor) | Language is a closed `Language` enum; unit is `'metric'`/`'imperial'` literal — no free-text input. Server already validates the existing payload. |
| V6 Cryptography | no | None |

### Known Threat Patterns for Flutter mobile settings

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Tampered settings payload | Tampering | Server-side validation on `/settings/:id` (existing, unchanged); client sends enum-bounded values only |
| Save on unauthenticated state | Info disclosure / error | Guard `settingsProvider != null`; route already behind auth redirect in `router_provider.dart` |

**No new security-sensitive code paths are introduced.**

## Sources

### Primary (HIGH confidence)
- Codebase (VERIFIED via Read/grep): `app/lib/provider/local_settings_provider.dart`, `settings_provider.dart`, `router_provider.dart`, `models/settings.dart`, `entities/settings_entity.dart`, `routes/settings_screen.dart`, `routes/settings_appearance_screen.dart`, `routes/settings_account_screen.dart`, `util/format_util.dart`, `main.dart`, `i18n/app_en.arb`, `i18n/app_de.arb`, `l10n.yaml`, `pubspec.yaml`, `analysis_options.yaml`, `web/src/lib/i18n/locales/*.json`
- `flutter --version` → 3.41.9 stable (VERIFIED)
- `.planning/phases/06-settings-navigation-language-units/06-CONTEXT.md` (locked decisions D-01..D-12)
- `.planning/phases/06-settings-navigation-language-units/06-UI-SPEC.md` (visual/interaction contract)
- `.planning/REQUIREMENTS.md`, `.planning/STATE.md`

### Secondary (MEDIUM confidence)
- Cross-reference of web JSON key set (573) vs ARB key set (484) and kebab→snake mapping (grep-based, sampled)

### Tertiary (LOW confidence)
- None — no WebSearch needed; phase is fully grounded in existing codebase patterns

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — every library pre-exists and was verified in pubspec/imports; no new packages
- Architecture: HIGH — all patterns are direct clones of verified existing code
- Pitfalls: HIGH — each pitfall was confirmed by reading the actual source (null-state, no try/catch, non-Consumer call sites, key-format diff, supportedLocales hardcode)
- i18n port effort (D-02): MEDIUM — mapping pattern verified on samples but not exhaustively across all 484 keys

**Research date:** 2026-06-19
**Valid until:** 2026-07-19 (stable; codebase-grounded, low churn risk)
