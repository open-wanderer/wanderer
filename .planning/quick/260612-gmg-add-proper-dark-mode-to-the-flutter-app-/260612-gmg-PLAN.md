---
phase: quick-260612-gmg
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - app/pubspec.yaml
  - app/lib/provider/theme_provider.dart
  - app/lib/provider/theme_provider.g.dart
  - app/lib/main.dart
  - app/lib/routes/settings_screen.dart
  - app/lib/provider/router_provider.dart
  - app/lib/i18n/app_en.arb
  - app/lib/i18n/app_de.arb
  - app/lib/components/base/wanderer_layout.dart
  - app/lib/components/trail/trail_card.dart
  - app/lib/components/profile/feed_item_card.dart
  - app/lib/components/list/list_card.dart
autonomous: false
requirements: [DARKMODE-01, DARKMODE-02, DARKMODE-03]
must_haves:
  truths:
    - "User can open Settings from their own profile via the gear icon and see an Appearance section"
    - "User can choose Light, Dark, or Follow System and the whole app re-themes immediately"
    - "The selected theme survives an app restart (persisted to disk)"
    - "Appearance labels are translated in English and German"
    - "High-visibility surfaces (bottom nav, trail cards, feed/list cards) are legible in dark mode (no white-on-white or hardcoded light-only colors)"
  artifacts:
    - path: "app/lib/provider/theme_provider.dart"
      provides: "Riverpod ThemeMode notifier with SharedPreferences persistence"
      contains: "class ThemeModeNotifier"
    - path: "app/lib/routes/settings_screen.dart"
      provides: "Settings screen with Appearance section and theme selector"
      contains: "class SettingsScreen"
  key_links:
    - from: "app/lib/main.dart"
      to: "app/lib/provider/theme_provider.dart"
      via: "ref.watch(themeModeNotifierProvider) drives MaterialApp.themeMode"
      pattern: "themeMode:\\s*ref.watch"
    - from: "app/lib/routes/profile_screen.dart"
      to: "/settings route"
      via: "existing gear IconButton context.push('/settings')"
      pattern: "context.push\\('/settings'\\)"
    - from: "app/lib/routes/settings_screen.dart"
      to: "app/lib/provider/theme_provider.dart"
      via: "ref.read(themeModeNotifierProvider.notifier).setMode(...)"
      pattern: "themeModeNotifierProvider.notifier"
---

<objective>
Add a proper, user-selectable dark mode to the Flutter app. The app already defines light and dark `ThemeData` (`AppTheme.createTheme`) but hardcodes `themeMode: ThemeMode.system` in `main.dart`, and the profile gear button already pushes a `/settings` route that does not exist yet. This plan makes the theme selectable (Light / Dark / Follow System), persists the choice, exposes it in a new Settings screen under an "Appearance" section reachable from the existing profile gear button, and fixes the highest-visibility hardcoded colors so dark mode is actually legible.

Purpose: Make Wanderer feel like a finished app on mobile — respecting the user's theme preference everywhere, not just where Material defaults happen to adapt.
Output: A `themeModeNotifierProvider` (Riverpod + SharedPreferences), a `SettingsScreen` with an Appearance selector wired into the router, new i18n keys in EN/DE, and theme-aware color fixes on the bottom nav, trail cards, and feed/list cards.
</objective>

<execution_context>
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/workflows/execute-plan.md
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@/Users/christianbeutel/Documents/svelte/wanderer/.planning/STATE.md
@/Users/christianbeutel/Documents/svelte/wanderer/CLAUDE.md

# Theme system already in place (both brightnesses defined)
@/Users/christianbeutel/Documents/svelte/wanderer/app/lib/theme/theme.dart
@/Users/christianbeutel/Documents/svelte/wanderer/app/lib/theme/colors.dart

# App root that currently hardcodes ThemeMode.system
@/Users/christianbeutel/Documents/svelte/wanderer/app/lib/main.dart

# Riverpod codegen patterns to follow (keepAlive notifier with state)
@/Users/christianbeutel/Documents/svelte/wanderer/app/lib/provider/toast_provider.dart

# Router (where /settings route must be registered) — note the existing dead push('/settings')
@/Users/christianbeutel/Documents/svelte/wanderer/app/lib/provider/router_provider.dart

# Settings entry point already wired on the profile (gear icon, line ~205)
@/Users/christianbeutel/Documents/svelte/wanderer/app/lib/routes/profile_screen.dart

# Scaffold/AppBar + context.pop pattern for the new screen
@/Users/christianbeutel/Documents/svelte/wanderer/app/lib/routes/profile_share_screen.dart

# i18n: alphabetically-sorted ARB files, gen-l10n config in l10n.yaml (generate: true)
@/Users/christianbeutel/Documents/svelte/wanderer/app/lib/i18n/app_en.arb
@/Users/christianbeutel/Documents/svelte/wanderer/app/lib/i18n/app_de.arb
</context>

<tasks>

<task type="auto" tdd="false">
  <name>Task 1: Theme provider with persistence + wire MaterialApp</name>
  <files>app/pubspec.yaml, app/lib/provider/theme_provider.dart, app/lib/provider/theme_provider.g.dart, app/lib/main.dart</files>
  <action>
Add `shared_preferences: ^2.3.2` to the `dependencies` block in `app/pubspec.yaml` (alphabetical: between `share_plus` and `textfield_tags`). Run `flutter pub get` in `app/`.

Create `app/lib/provider/theme_provider.dart` following the `toast_provider.dart` codegen pattern (`@Riverpod(keepAlive: true)`, `part 'theme_provider.g.dart';`). Implement `class ThemeModeNotifier extends _$ThemeModeNotifier`. The notifier's `build()` returns `ThemeMode` — synchronously return `ThemeMode.system` as the initial value, then load the persisted value asynchronously and update `state`. Persist using `SharedPreferences` under the key `'theme_mode'` storing the enum name (`'light' | 'dark' | 'system'`). Add a public method `Future<void> setMode(ThemeMode mode)` that sets `state = mode` and writes the enum name to SharedPreferences. Map between the stored string and `ThemeMode` with a small private helper (default to `ThemeMode.system` on unknown/missing). Do NOT block app startup on disk I/O — load happens in build and updates state when ready.

Run `dart run build_runner build --delete-conflicting-outputs` in `app/` to generate `theme_provider.g.dart`.

In `app/lib/main.dart`, import the new provider, convert the relevant scope so `MainApp.build` can `ref.watch(themeModeNotifierProvider)`, and replace the hardcoded `themeMode: ThemeMode.system` with `themeMode: ref.watch(themeModeNotifierProvider)`. Leave `theme:` and `darkTheme:` (AppTheme.createTheme light/dark) unchanged. (DARKMODE-01)
  </action>
  <verify>
    <automated>cd app && grep -q "shared_preferences" pubspec.yaml && test -f lib/provider/theme_provider.g.dart && grep -Eq "themeMode:\s*ref.watch\(themeModeNotifierProvider\)" lib/main.dart && flutter analyze lib/provider/theme_provider.dart lib/main.dart</automated>
  </verify>
  <done>shared_preferences is a dependency; theme_provider.dart + generated .g.dart exist; main.dart drives MaterialApp.themeMode from the provider; flutter analyze passes on the changed files.</done>
</task>

<task type="auto" tdd="false">
  <name>Task 2: Settings screen with Appearance selector + route + i18n keys</name>
  <files>app/lib/routes/settings_screen.dart, app/lib/provider/router_provider.dart, app/lib/i18n/app_en.arb, app/lib/i18n/app_de.arb</files>
  <action>
Add four i18n keys to BOTH `app/lib/i18n/app_en.arb` and `app/lib/i18n/app_de.arb`, inserted in alphabetical position (these files are alphabetically sorted JSON — `appearance` goes near the top after `allow_auto_geolocate`/`all`; `theme_dark`, `theme_light`, `theme_system` go in the `t*` block). EN values: `appearance` = "Appearance", `theme_light` = "Light", `theme_dark` = "Dark", `theme_system` = "Follow system". DE values: `appearance` = "Darstellung", `theme_light` = "Hell", `theme_dark` = "Dunkel", `theme_system` = "System folgen". Reuse the existing `settings` key ("Settings"/"Einstellungen") for the screen title. After editing, run `flutter gen-l10n` in `app/` (l10n.yaml + `generate: true` regenerates `app_localizations*.dart`).

Create `app/lib/routes/settings_screen.dart` as a `ConsumerWidget` named `SettingsScreen`. Use the `Scaffold` + `AppBar` (leading back `IconButton` -> `context.pop()`, title `AppLocalizations.of(context)!.settings`) pattern from `profile_share_screen.dart`. Body: a section header reading `AppLocalizations.of(context)!.appearance`, then three `RadioListTile<ThemeMode>` (or a grouped equivalent) for `ThemeMode.light` / `ThemeMode.dark` / `ThemeMode.system` labeled with `theme_light` / `theme_dark` / `theme_system`. `groupValue` = `ref.watch(themeModeNotifierProvider)`; `onChanged` calls `ref.read(themeModeNotifierProvider.notifier).setMode(value!)`. Use `Theme.of(context).colorScheme` for any colors — do NOT hardcode. Keep the screen structured so additional settings sections can be appended later (don't special-case it as appearance-only beyond what's needed).

In `app/lib/provider/router_provider.dart`, register a top-level `GoRoute(path: '/settings', builder: (context, state) => const SettingsScreen())` and add the import for `settings_screen.dart`. This satisfies the already-present `context.push('/settings')` on the profile gear button. (DARKMODE-02, DARKMODE-03)
  </action>
  <verify>
    <automated>cd app && grep -q '"appearance"' lib/i18n/app_en.arb && grep -q '"appearance"' lib/i18n/app_de.arb && grep -q '"theme_system"' lib/i18n/app_de.arb && test -f lib/routes/settings_screen.dart && grep -q "path: '/settings'" lib/provider/router_provider.dart && grep -q "themeModeNotifierProvider.notifier" lib/routes/settings_screen.dart && flutter analyze lib/routes/settings_screen.dart lib/provider/router_provider.dart</automated>
  </verify>
  <done>Appearance + three theme keys exist in EN and DE ARB files and localizations regenerate; SettingsScreen renders an Appearance section with a 3-way ThemeMode selector wired to the provider; /settings route is registered so the profile gear button navigates to it; flutter analyze passes.</done>
</task>

<task type="auto" tdd="false">
  <name>Task 3: Replace hardcoded light-only colors on high-visibility surfaces</name>
  <files>app/lib/components/base/wanderer_layout.dart, app/lib/components/trail/trail_card.dart, app/lib/components/profile/feed_item_card.dart, app/lib/components/list/list_card.dart</files>
  <action>
Replace hardcoded brightness-unaware colors with `Theme.of(context).colorScheme` / theme-derived equivalents so these always-visible surfaces are legible in dark mode. Do NOT introduce new hardcoded hex colors; derive from the active theme.

In `wanderer_layout.dart` (bottom nav): replace `unselectedItemColor: Colors.grey` and the `Colors.grey.shade300` avatar background with theme-aware values — e.g. `Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.5)` for unselected, and `colorScheme.surfaceContainerHighest` (or `secondary`) for the avatar placeholder. Set the `BottomNavigationBar` `backgroundColor` to `colorScheme.surface` so it follows the theme.

In `trail_card.dart`: replace `Colors.white` (line ~310, image/label backgrounds), `Colors.black87` text, `Colors.grey[600]/[100]/.shade300` with theme equivalents — readable secondary text -> `colorScheme.onSurface.withValues(alpha: 0.6)`; surface fills -> `colorScheme.surface` / `surfaceContainerHighest`; on-image overlays that must stay readable over photos may keep an explicit semi-transparent black scrim (that is intentional, not a theme bug) — leave true photo-overlay scrims, only fix colors used as widget surfaces/text.

In `feed_item_card.dart` and `list_card.dart`: replace `Colors.grey.shade300` placeholder backgrounds and `Colors.grey[600]` / `Colors.grey` secondary text/icon colors with `colorScheme.surfaceContainerHighest` and `colorScheme.onSurface.withValues(alpha: 0.6)` respectively.

Scope is intentionally these four high-traffic files (bottom nav + the three card types seen on the home/profile/library screens). Other files with hardcoded colors are out of scope for this quick task and tracked as follow-up; do not expand beyond these four files. (DARKMODE-01)
  </action>
  <verify>
    <automated>cd app && grep -v '^\s*//' lib/components/base/wanderer_layout.dart | grep -cq "Colors.grey" && echo "still has Colors.grey -- check intentional" ; flutter analyze lib/components/base/wanderer_layout.dart lib/components/trail/trail_card.dart lib/components/profile/feed_item_card.dart lib/components/list/list_card.dart</automated>
  </verify>
  <done>Bottom nav, trail cards, feed cards, and list cards derive their surface/text/icon colors from Theme.of(context).colorScheme (no light-only Colors.grey/white/black surfaces remain except intentional photo-overlay scrims); flutter analyze passes on all four files.</done>
</task>

<task type="checkpoint:human-verify" gate="blocking">
  <what-built>Selectable theme (Light/Dark/Follow System) persisted across restarts, an Appearance section in a new Settings screen reachable from the profile gear icon, EN/DE translations, and dark-mode color fixes on the bottom nav and trail/feed/list cards.</what-built>
  <how-to-verify>
1. Run the app (`cd app && flutter run`).
2. Open your own profile, tap the gear icon (top-right). The new Settings screen opens with an "Appearance" section.
3. Select "Dark" — the whole app should switch to dark immediately. Check the bottom nav bar, a trail card on Home, and feed/list cards on a profile: text and backgrounds should be legible (no white-on-white, no invisible grey-on-dark).
4. Select "Light" — app returns to light. Select "Follow system" and toggle your device theme — app should follow.
5. Fully kill and relaunch the app with "Dark" selected — it should start in dark mode (preference persisted).
6. Switch device language to German and reopen Settings — the section reads "Darstellung" with "Hell / Dunkel / System folgen".
  </how-to-verify>
  <resume-signal>Type "approved" or describe any surface that looks wrong in dark mode</resume-signal>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| app ↔ device local storage | Theme preference written to SharedPreferences (on-device key/value). No network, no untrusted input. |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-gmg-01 | Tampering | SharedPreferences 'theme_mode' value | accept | Low-value local UI preference; unknown/corrupt values fall back to ThemeMode.system via the parser default. No security impact. |
| T-gmg-SC | Tampering | pub add `shared_preferences` | mitigate | shared_preferences is a first-party flutter.dev/Flutter team package (pub.dev verified publisher). Legitimate, widely used; no checkpoint required. qr_flutter/share_plus already vetted in pubspec. |
</threat_model>

<verification>
- `flutter analyze` passes for all changed Dart files.
- `flutter gen-l10n` regenerates localizations without error and `appearance`/`theme_*` keys are accessible via `AppLocalizations`.
- App builds and runs; theme selection changes the live theme and persists across a full restart.
- Bottom nav and trail/feed/list cards are legible in dark mode.
</verification>

<success_criteria>
- User can reach Settings from their own profile (existing gear button now lands on a real screen).
- Settings has an Appearance section offering Light / Dark / Follow System, translated in EN and DE.
- Selecting a theme re-themes the app instantly and the choice survives an app restart (SharedPreferences).
- High-visibility surfaces (bottom nav, trail/feed/list cards) adapt to the selected theme.
- No breaking changes to existing routes or the bottom-nav profile tab.
</success_criteria>

<output>
Create `.planning/quick/260612-gmg-add-proper-dark-mode-to-the-flutter-app-/260612-gmg-SUMMARY.md` when done.
</output>
