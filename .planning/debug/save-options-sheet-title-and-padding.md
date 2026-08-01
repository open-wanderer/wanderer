---
status: diagnosed
trigger: "c) partial. The save options sheet has the wrong title (\"Save recordings\") and needs more bottom padding because of the bottom navigation bar"
created: 2026-08-01T00:00:00Z
updated: 2026-08-01T00:00:00Z
---

## Current Focus

hypothesis: CONFIRMED (both defects). (1) At HEAD `showTrackSaveOptionsSheet` takes no title parameter, so all three capture sources render the recording-specific l10n key `save_recording_options` ("Save recording"). (2) The sheet's only bottom inset is `MediaQuery.viewInsets.bottom` (keyboard); it never consumes `MediaQuery.padding.bottom` / `viewPadding.bottom` and `showModalBottomSheet`'s `useSafeArea` defaults to `false`, so nothing reserves space for the shell's `BottomAppBar` + docked FAB (shell-hosted flows) or the system gesture inset (root-hosted flows).
test: Read sheet + gate + all three callers at HEAD, the router shell structure, `WandererLayout`'s Scaffold, all 9 `showModalBottomSheet` call sites, Flutter's `Scaffold`/`bottom_sheet.dart` sources, and the i18n assets.
expecting: n/a — diagnosis complete, goal is find_root_cause_only.
next_action: Hand to planning. Do NOT fix here.

## Symptoms

expected: The save-options sheet is correctly titled for the flow that opened it, and its content is not obscured by the bottom navigation bar.
actual: In the GPX import flow (online), the sheet is titled "Save recording" and its bottom content sits under/too close to the bottom navigation bar.
errors: none (purely presentational)
reproduction: Test 1(c) in .planning/phases/34-dart-conversion-port/34-UAT.md — online, import a .gpx.
started: Discovered during UAT of Phase 34 (dart-conversion-port).

## Eliminated

- hypothesis: A second, import-specific sheet widget exists and is the one being mistitled.
  evidence: `grep showTrackSaveOptionsSheet` returns exactly one definition (track_save_options_sheet.dart:15) and one call site (track_save_options_util.dart). All three capture flows funnel through `resolveTrackSaveOptions`.
  timestamp: T2

- hypothesis: `SafeArea` / `useSafeArea: true` alone is inert inside the shell because `Scaffold` strips bottom padding from `body` when a `bottomNavigationBar` is present.
  evidence: `Scaffold` does strip it (scaffold.dart:3158 `removeBottomPadding: widget.bottomNavigationBar != null`), BUT `WandererLayout` sets `extendBody: true`, which re-inserts it via `_BodyBuilder` (scaffold.dart:969: `bottom = extendBody ? max(metrics.padding.bottom, bodyConstraints.bottomWidgetsHeight) : ...`). So inside the shell body `MediaQuery.padding.bottom == bottomWidgetsHeight` (the BottomAppBar height). `SafeArea` therefore DOES work there — it just does not cover the docked FAB's overhang.
  timestamp: T7

## Evidence

- timestamp: T0
  checked: git status / git diff HEAD
  found: The working tree carries UNCOMMITTED exploratory edits by the user to `track_save_options_sheet.dart` (adds `String? title` + `EdgeInsets? padding` params), `track_save_options_util.dart` (passes `title: "Adjust track"` behind a `// TODO: make this l10n` and `padding: EdgeInsets.only(bottom: kBottomNavigationBarHeight + 48)`), and `trail_source_select_screen.dart` (unrelated: import no longer network-blocked).
  implication: Diagnosis is against HEAD (c62527b2), the state that failed UAT. The working-tree patch is a hand workaround, not a fix — it hardcodes an English literal and applies a shell-sized constant inside a SHARED gate that also serves two non-shell screens.

- timestamp: T1
  checked: `git show HEAD:app/lib/components/navigation/track_save_options_sheet.dart`
  found: line 15 `showTrackSaveOptionsSheet(BuildContext context)` — no parameters beyond context. Line 74 renders `l10n.save_recording_options` unconditionally. Line 48-52 the only bottom inset is `24 + MediaQuery.of(context).viewInsets.bottom`.
  implication: Both defects live in this one widget. `viewInsets` is the KEYBOARD inset — it is 0 when no text field is focused, so the sheet effectively ends 24px above whatever hosts it.

- timestamp: T2
  checked: callers of `resolveTrackSaveOptions`
  found: One shared gate (track_save_options_util.dart:29) called by navigation_screen.dart:736 (recording), route_planner_screen.dart:529 (planner Finish) and trail_import_util.dart:104 (file import). The gate takes no source parameter and at HEAD passed nothing to the sheet.
  implication: Root cause (title): the title is a property of the widget, not of the flow. Import and planner both inherit the recording title.

- timestamp: T3
  checked: app/lib/i18n/*.arb, l10n.yaml, generated app_localizations_*.dart
  found: `save_recording_options` = "Save recording" and is defined ONLY in `app_en.arb` (the template per l10n.yaml). 14 locales exist (cs, de, en, es, eu, fr, hu, it, nl, no, pl, pt, ru, zh); en has 385 keys, every other locale has 327 and falls back to the English literal in its generated class. Generated `app_localizations*.dart` files are committed to the repo.
  implication: i18n scope for new title keys = add to `app_en.arb` only + `flutter gen-l10n` + commit the 15 regenerated dart files. No translator round trip required (matches how this key was introduced).

- timestamp: T4
  checked: app/lib/provider/router_provider.dart:135-192
  found: A single `ShellRoute` wraps ONLY `/map`, `/list`, `/profile`, `/library`, `/trail/create` in `WandererLayout`. `/route-planner` (line 257), `/record` (line 285) and `/trail/create/edit` (line 337) are top-level routes OUTSIDE the shell.
  implication: The bottom nav bar exists only for the import entry point (`/trail/create` = TrailSourceSelectScreen). Recording and planner never have it.

- timestamp: T5
  checked: app/lib/components/base/wanderer_layout.dart:52-71
  found: shell Scaffold uses `extendBody: true`, `bottomNavigationBar: BottomAppBar(height: kBottomNavigationBarHeight, shape: CircularNotchedRectangle())` and a `FloatingActionButtonLocation.centerDocked` FAB that protrudes above the bar.
  implication: With `extendBody: true` the body (and anything the shell's nested Navigator draws) extends to the physical screen bottom, and Scaffold paints `bottomNavigationBar` + FAB ON TOP of it.

- timestamp: T6
  checked: go_router 17.3.0 builder.dart:301 (`_CustomNavigator` for ShellRouteMatch), builder.dart:459 (`Navigator(...)`)
  found: `ShellRoute` builds a real nested `Navigator` for its sub-routes; that Navigator is the `child` handed to `WandererLayout` and therefore lives inside the Scaffold `body`.
  implication: `showModalBottomSheet` defaults to `useRootNavigator: false`, so a sheet opened with a shell child's `BuildContext` is pushed into the SHELL's nested Navigator — its overlay is clipped to the body, and the BottomAppBar + FAB draw over its bottom ~56-84px. This is the exact mechanism the user saw.

- timestamp: T7
  checked: flutter/lib/src/material/bottom_sheet.dart:892, :1167
  found: `useSafeArea` defaults to `false`; when false the route only does `MediaQuery.removePadding(removeTop: true)`. `padding.bottom` therefore survives into the sheet content and is available to it.
  implication: The correct signal is already in scope at the sheet's build context — it just is not read. Inside the shell it resolves to the BottomAppBar height; on the root navigator it resolves to the real Android system gesture inset.

- timestamp: T8
  checked: trail_import_util.dart:104 vs its two entry points
  found: `importTrailFile(navContext: ...)` receives TrailSourceSelectScreen's own `context` from the in-app picker (trail_source_select_screen.dart:195, a shell child) but receives `navigatorKey.currentContext` — the ROOT Navigator — from the OS share-intent handler (main.dart:181-191).
  implication: The two import entry points host the sheet on DIFFERENT navigators. In-app picker: sheet is under the app's BottomAppBar + FAB. Share intent: sheet is above them, but with only 24px of bottom padding and `useSafeArea: false` it sits under the Android system gesture bar. Same root cause, two different obstructions — which is why a single hardcoded `kBottomNavigationBarHeight` constant in the shared gate is the wrong shape of fix.

- timestamp: T9
  checked: all 9 `showModalBottomSheet` call sites
  found: Two existing conventions. (a) `SafeArea` wrapper — library_screen.dart:119 (a shell child) and map_app_sheet.dart:23. (b) hardcoded `24 + kBottomNavigationBarHeight + 56` — travel_profile_sheet.dart:34, which is opened from TrailSourceSelectScreen (trail_source_select_screen.dart:50 and :135), the SAME shell screen as the import flow, and is the very sheet `track_save_options_sheet.dart`'s doc comment says it was "styled after". Also: wanderer_icon_picker.dart:166 and missing_coverage_sheet.dart:153 repeat the same `viewInsets.bottom`-only defect.
  implication: The bug is a copy of the aesthetic without the layout compensation. `missing_coverage_sheet.dart` (opened from trail_download_state_provider.dart:87, reachable from shell screens) carries the same latent defect.

- timestamp: T10
  checked: user-reported string vs actual
  found: User wrote "Save recordings"; the actual en string is "Save recording". No other title can be rendered by this widget at HEAD.
  implication: Report is a paraphrase; the key is unambiguous.

## Resolution

root_cause: |
  TITLE — `showTrackSaveOptionsSheet` (track_save_options_sheet.dart:15 @HEAD) exposes no way to name the sheet, and hardcodes `l10n.save_recording_options` ("Save recording") at line 74. The shared gate `resolveTrackSaveOptions` (track_save_options_util.dart:29) is called identically by all three capture sources with no source discriminator, so the recording-specific title leaks into the file-import and route-planner flows.

  PADDING — The sheet's only bottom inset is `24 + MediaQuery.of(context).viewInsets.bottom` (line 48-52 @HEAD). `viewInsets` is the keyboard inset and is 0 here; the sheet never reads `MediaQuery.padding.bottom` / `viewPadding.bottom`, and `showModalBottomSheet`'s `useSafeArea` defaults to `false`. Nothing reserves space for the obstruction below it. Because `showModalBottomSheet` also defaults to `useRootNavigator: false`, when the sheet is opened from a shell child (`/trail/create`) it is pushed into go_router's nested shell Navigator, which lives inside `WandererLayout`'s `extendBody: true` Scaffold body — so the `BottomAppBar` (56px) plus the centerDocked FAB's overhang (~28px) paint on top of the sheet's bottom edge, covering the Save button.

fix: (not applied — goal: find_root_cause_only)
verification: (n/a)
files_changed: []
