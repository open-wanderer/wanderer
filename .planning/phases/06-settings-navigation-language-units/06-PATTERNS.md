# Phase 6: Settings Navigation + Language & Units - Pattern Map

**Mapped:** 2026-06-19
**Files analyzed:** 7 new/modified Dart files + 12 new ARB files + ~14 unit-wiring call sites
**Analogs found:** 7 / 7 (all code-bearing files have exact in-repo analogs)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `app/lib/routes/settings_language_screen.dart` (NEW) | screen | request-response (read settings → save) | `app/lib/routes/settings_appearance_screen.dart` | exact |
| `app/lib/routes/settings_privacy_screen.dart` (NEW, stub) | screen | none | `app/lib/routes/settings_appearance_screen.dart` (Scaffold+AppBar shell) | role-match |
| `app/lib/routes/settings_notifications_screen.dart` (NEW, stub) | screen | none | `app/lib/routes/settings_appearance_screen.dart` (Scaffold+AppBar shell) | role-match |
| `app/lib/routes/settings_screen.dart` (MODIFY) | screen | navigation | itself (existing row pattern) | exact |
| `app/lib/provider/local_settings_provider.dart` OR new file — `localeProvider` + `unitProvider` (NEW providers) | provider | derived/transform | `themeMode` provider in `app/lib/provider/local_settings_provider.dart` | exact |
| `app/lib/provider/router_provider.dart` (MODIFY) | route | config | itself (existing `/settings` subtree) | exact |
| `app/lib/main.dart` (MODIFY) | config | config | itself (existing `MaterialApp.router`) | exact |
| `app/lib/i18n/app_{cs,es,eu,fr,hu,it,nl,no,pl,pt,ru,zh}.arb` (NEW ×12) | config (i18n) | transform (port) | `app/lib/i18n/app_de.arb` (existing translated ARB) + `web/src/lib/i18n/locales/*.json` (value source) | role-match |
| ~14 `format_util` call-site files (MODIFY) | component | transform (display) | `settings_appearance_screen.dart` (`ref.watch` in build) | role-match |

## Pattern Assignments

### `app/lib/routes/settings_language_screen.dart` (screen, request-response)

**Analog:** `app/lib/routes/settings_appearance_screen.dart` (read in full — 56 lines). Clone this verbatim and adapt.

**Imports + class shell** (analog lines 1-17):
```dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
// ADD for language screen: settings_provider, models/settings (Language enum)

class SettingsAppearanceScreen extends ConsumerWidget {
  const SettingsAppearanceScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context)!;
    final currentMode = ref.watch(themeModeProvider);   // → watch settingsProvider for language screen
    final colorScheme = Theme.of(context).colorScheme;
    final activeColor = Theme.of(context).brightness == Brightness.dark
        ? colorScheme.onSurface
        : colorScheme.primary;
```

**Scaffold + AppBar with back button** (analog lines 19-26) — copy exactly; swap title to `l10n.language`.

**RadioGroup + RadioListTile picker** (analog lines 27-53) — the core LANG-01 pattern:
```dart
body: RadioGroup<ThemeMode>(                 // → RadioGroup<Language> (D-08, see Open Q2)
  groupValue: currentMode,                   // → settings?.language
  onChanged: (value) {
    if (value != null) {                     // → guard: if (value != null && settings != null)
      ref.read(localSettingsProvider.notifier).setThemeMode(value);
      // → wrap in try/catch + toast, call:
      //   ref.read(settingsProvider.notifier).saveToServer(settings.copyWith(language: value));
    }
  },
  child: ListView(
    children: [
      RadioListTile<ThemeMode>(              // → 14× RadioListTile<Language>, native-name titles (D-09)
        title: Text(l10n.theme_light),
        value: ThemeMode.light,
        activeColor: activeColor,
      ),
      // ...
    ],
  ),
),
```

**Native-name labels (D-09):** hardcoded const map `Language → String` (NOT ARB). Values: cs=Čeština, en=English, de=Deutsch, es=Español, eu=Euskara, fr=Français, hu=Magyar, it=Italiano, nl=Nederlands, no=Norsk, pl=Polski, pt=Português, ru=Русский, zh=中文.

**Units SwitchListTile (D-10, LANG-02)** — append below the language ListView (planner must lock polarity, see Shared Patterns):
```dart
SwitchListTile(
  title: Text(l10n.metric),               // l10n.units section label above (key `units` exists)
  value: settings?.unit != 'imperial',    // metric=on (D-10 line 37) — LOCK vs specifics line 115
  onChanged: (isMetric) {
    if (settings != null) {
      // try/catch + toast:
      ref.read(settingsProvider.notifier)
         .saveToServer(settings.copyWith(unit: isMetric ? 'metric' : 'imperial'));
    }
  },
)
```

---

### `app/lib/routes/settings_privacy_screen.dart` + `settings_notifications_screen.dart` (screen, stub)

**Analog:** `settings_appearance_screen.dart` lines 1-26 (Scaffold + AppBar shell only).

Minimal `ConsumerWidget` (or `StatelessWidget`) returning `Scaffold(appBar: AppBar(leading: back IconButton, title: Text(l10n.privacy / l10n.notifications)), body: ...placeholder)`. Phases 7/9 fill the body. Title ARB keys `privacy` / `notifications` already exist in `app_en.arb`.

---

### `localeProvider` + `unitProvider` (provider, derived) — D-03, D-04

**Analog:** `app/lib/provider/local_settings_provider.dart` lines 34-42 (`themeMode` derived provider — read in full).

**Exact template** (analog lines 34-42):
```dart
@Riverpod(keepAlive: true)
ThemeMode themeMode(Ref ref) {
  final settings = ref.watch(localSettingsProvider);
  return switch (settings.themeMode) {
    'light' => ThemeMode.light,
    'dark' => ThemeMode.dark,
    _ => ThemeMode.system,
  };
}
```

**New providers** (watch `settingsProvider`, which returns `Settings?` — see Shared Patterns null-guard):
```dart
@Riverpod(keepAlive: true)
Locale? locale(Ref ref) {                       // null → device-locale fallback (CONTEXT specifics)
  final lang = ref.watch(settingsProvider)?.language;
  return lang == null ? null : Locale(lang.name);   // .name == lowercase enum == locale code (VERIFIED settings.dart @JsonValue)
}

@Riverpod(keepAlive: true)
String unit(Ref ref) {
  return ref.watch(settingsProvider)?.unit ?? 'metric';
}
```
`Language` enum: 14 values with `@JsonValue('cs'..'zh')` at `app/lib/models/settings.dart` lines 6-34. `unit` is `String?` field (line 97), `language` is `Language?` (line 98) on the `Settings` freezed model. After adding providers: run `dart run build_runner build --delete-conflicting-outputs`.

---

### `app/lib/routes/settings_screen.dart` (screen, navigation) — MODIFY (D-06, D-07)

**Analog:** itself, lines 28-39 (existing Account/Appearance rows). Each row is a `ListTile` with `FaIcon(size: 18)` leading, `Text(l10n.X)` title, `chevron_right` trailing, `context.push('/settings/X')` onTap.

```dart
ListTile(
  leading: const FaIcon(FontAwesomeIcons.circleUser, size: 18),
  title: Text(l10n.my_account),
  trailing: const Icon(Icons.chevron_right),
  onTap: () => context.push('/settings/account'),
),
```
Insert 3 new rows to reach final order (D-06): Account → **Privacy** (`FontAwesomeIcons.lock` → `/settings/privacy`) → **Language** (`FontAwesomeIcons.globe` → `/settings/language`) → **Notifications** (`FontAwesomeIcons.bell` → `/settings/notifications`) → Appearance → `Divider()` → logout `WandererButton` (lines 40-52, unchanged).

---

### `app/lib/provider/router_provider.dart` (route, config) — MODIFY (D-11)

**Analog:** itself, lines 166-178 (existing `/settings` subtree with `account` + `appearance` children):
```dart
GoRoute(
  path: '/settings',
  builder: (context, state) => const SettingsScreen(),
  routes: [
    GoRoute(path: 'account', builder: (context, state) => const SettingsAccountScreen()),
    GoRoute(path: 'appearance', builder: (context, state) => const SettingsAppearanceScreen()),
    // ADD: privacy, language, notifications children (same shape)
  ],
),
```
Add 3 `GoRoute` children + their imports at top (mirror existing `settings_account_screen.dart` import at line 23).

---

### `app/lib/main.dart` (config) — MODIFY (D-01)

**Analog:** itself, lines 58-69 (`MaterialApp.router`):
```dart
return MaterialApp.router(
  localizationsDelegates: const [ AppLocalizations.delegate, GlobalMaterialLocalizations.delegate, ... ],
  supportedLocales: const [Locale('en'), Locale('de')],   // → AppLocalizations.supportedLocales (auto-14 after gen-l10n)
  themeMode: ref.watch(themeModeProvider),                // template for: locale: ref.watch(localeProvider)
```
Add `locale: ref.watch(localeProvider),` and expand `supportedLocales` (Pitfall 5 — required for live switch to work).

---

### i18n ARB ports ×12 (config, transform) — D-02

**Analog (structure):** `app/lib/i18n/app_de.arb` (existing translated ARB — `@@locale` header + flat snake_case keys).
**Value source:** `web/src/lib/i18n/locales/{cs,es,eu,fr,hu,it,nl,no,pl,pt,ru,zh}.json`.

Iterate keyed off the **`app_en.arb` key set (484)**, not the web set. For each ARB key, look up the kebab-cased equivalent in the matching web JSON (`-` ↔ `_`); copy value if present, omit otherwise (gen-l10n fills English from template — satisfies D-02). Preserve `@@locale` header and ICU plural syntax verbatim. Run `flutter gen-l10n` after. (Pitfall 3.)

---

### Unit-wiring call sites (component, transform) — D-05 / LANG-02

**Analog (Consumer case):** `settings_appearance_screen.dart` line 13 — `final x = ref.watch(provider)` in `build`, then use. Pattern: `final unit = ref.watch(unitProvider);` then `formatDistance(d, unit: unit)`.

**Call-site files (14, all currently default to `unit: 'metric'` via `format_util.dart` defaults):**
`list_card.dart`, `list_list_item.dart`, `summit_log_card.dart`, `trail_card.dart`, `trail_list_item.dart`, `trail_panel.dart`, `trail_quick_filter_bar.dart`, `global_search_screen.dart`, `list_detail_screen.dart`, `navigation_screen.dart`, `trail_filter_screen.dart` (these have/can get `ref`), plus the 3 below.

**3 non-Consumer files needing structural change (Pitfall 4 — own plan task):**
| File | Current type (VERIFIED) | Strategy |
|------|------------------------|----------|
| `app/lib/components/trail/elevation_profile.dart` | `StatefulWidget` (line 13) | Convert to `ConsumerStatefulWidget` OR pass `unit` via constructor |
| `app/lib/components/trail/waypoint_sheet.dart` | `StatelessWidget` (line 9) | Convert to `ConsumerWidget` OR constructor param |
| `app/lib/components/trail/trail_timeline.dart` | `StatelessWidget` (line 11) + nested private `_TimelineRow` (line 77) | Pass `unit` as constructor param down to `_TimelineRow` (avoids touching private widget's ref) |

## Shared Patterns

### Null-safe settings access
**Source:** `app/lib/provider/settings_provider.dart` line 16 — `Settings? build()` returns null until login.
**Apply to:** `settings_language_screen.dart`, both new providers.
```dart
final settings = ref.watch(settingsProvider);   // Settings?
if (settings != null) { /* read/save */ }        // providers use ?. with ?? fallback
```
Render default selection (English / metric) when null.

### Auto-save with error handling
**Source:** `app/lib/provider/settings_provider.dart` lines 33-39 (`saveToServer` — has NO internal try/catch; awaits dio POST `/settings/:id`).
**Apply to:** every save in `settings_language_screen.dart` (D-12, Pitfall 2).
```dart
final response = await ref.read(apiProvider).post('/settings/${settings.id}', data: settings.toJson());
final updated = Settings.fromJson(response.data as Map<String, dynamic>);
await updateFromServer(updated);   // ObjectBox put + invalidateSelf
```
Wrap the screen-level `saveToServer(...)` call in try/catch; surface failures via existing toast/`ToastOverlay`. Per CLAUDE.md error convention (`AsyncValue.guard()` / try-catch + toast).

### FaIcon list rows
**Source:** `settings_screen.dart` lines 28-39.
**Apply to:** the 3 new settings rows. `const FaIcon(FontAwesomeIcons.X, size: 18)` leading + `chevron_right` trailing + `context.push('/settings/X')`.

### copyWith for immutable update
**Source:** `Settings` freezed model (`app/lib/models/settings.dart`).
**Apply to:** all saves — `settings.copyWith(language: ...)` / `.copyWith(unit: ...)` before `saveToServer`.

## No Analog Found

None. Every code-bearing file in this phase clones an existing, verified in-repo pattern. The i18n ARB ports have a structural analog (`app_de.arb`) plus an external value source (web JSON).

## Planner Decisions to Lock (from RESEARCH Open Questions)
- **Switch polarity (D-10):** CONTEXT line 37 says metric=on; specifics line 115 says `value: settings?.unit == 'imperial'` (imperial=on). Pick one. Recommended: metric=on, single `l10n.metric` title.
- **RadioGroup type:** `<Language>` (recommended, avoids string bugs) vs `<String>` (UI-SPEC). Pick one, be consistent.
- **Chinese variant:** single `zh` ARB from `web/.../zh.json` (match web).

## Metadata

**Analog search scope:** `app/lib/routes/`, `app/lib/provider/`, `app/lib/util/`, `app/lib/models/`, `app/lib/i18n/`, `app/lib/components/trail/`, `web/src/lib/i18n/locales/`
**Files read for excerpts:** `settings_appearance_screen.dart`, `local_settings_provider.dart`, `settings_screen.dart`, `settings_provider.dart`, `format_util.dart`, `router_provider.dart` (subtree), `main.dart` (MaterialApp block), `settings.dart` (enum/fields grep)
**Pattern extraction date:** 2026-06-19
