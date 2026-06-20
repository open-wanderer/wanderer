---
phase: 06-settings-navigation-language-units
fixed_at: 2026-06-20T00:00:00Z
review_path: .planning/phases/06-settings-navigation-language-units/06-REVIEW.md
iteration: 1
findings_in_scope: 14
fixed: 14
skipped: 0
status: all_fixed
---

# Phase 06: Code Review Fix Report

**Fixed at:** 2026-06-20
**Source review:** .planning/phases/06-settings-navigation-language-units/06-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 14 (5 critical, 9 warning)
- Fixed: 14
- Skipped: 0

All Critical and Warning findings were fixed. WR-06 shares the root cause with
CR-01 and was resolved by the same guard. Info findings (IN-01 to IN-04) were
out of scope (`critical_warning`) and not addressed.

## Fixed Issues

### CR-01: `_niceInterval` crashes / produces `NaN` on flat terrain

**Files modified:** `app/lib/components/trail/elevation_profile.dart`
**Commit:** 586cb106
**Applied fix:** Added `if (range <= 0) return 1.0;` guard at the top of
`_niceInterval`, preventing `log(0)` from producing `-Infinity` and feeding a
zero interval into fl_chart axes. Also resolves WR-06 (single-point GPX trigger).

### CR-02: Tab-controller listener accumulates on every dependency change

**Files modified:** `app/lib/components/trail/trail_panel.dart`
**Commit:** d2c0a13a
**Applied fix:** Added a `TabController? _controller` field; `didChangeDependencies`
now removes the previous listener and only re-registers when the controller
instance actually changes, and a new `dispose()` removes the listener on teardown.
**Note:** requires human verification — listener-lifecycle logic should be confirmed
in-app (tab switching across theme/locale changes).

### CR-03: Force-unwrap of nullable `author` crashes `TrailPanel`

**Files modified:** `app/lib/components/trail/trail_panel.dart`
**Commit:** 5b67cbe2
**Applied fix:** Wrapped the author `InkWell` in a collection-element guard
`if (trail.expand?.author != null)` so the section is omitted when expand data
is absent instead of throwing a null-check error.

### CR-04: Distance range slider max label uses `formatElevation`

**Files modified:** `app/lib/routes/trail_filter_screen.dart`
**Commit:** d45ef249
**Applied fix:** Changed the max label from `formatElevation(...)` to
`formatDistance(...)` so the distance slider shows correct km/mi values
consistent with the min label.

### CR-05: Duplicate `name: "start"` on both date pickers

**Files modified:** `app/lib/routes/trail_filter_screen.dart`
**Commit:** b39272b6
**Applied fix:** Renamed the end-date picker's form field name from `"start"`
to `"end"` to remove the FormBuilderField key collision.

### WR-01: `authProvider.value!` force-unwrap in five widget build methods

**Files modified:** `app/lib/components/list/list_card.dart`,
`app/lib/components/list/list_list_item.dart`,
`app/lib/components/trail/trail_card.dart`,
`app/lib/components/trail/trail_list_item.dart`,
`app/lib/components/trail/summit_log_card.dart`
**Commit:** f890610c
**Applied fix:** Replaced `ref.watch(authProvider).value!` with
`.valueOrNull` plus an early `if (user == null) return const SizedBox.shrink();`
guard in all five widgets.

### WR-02: `GlobalKey<FormBuilderState>()` constructed inline inside build

**Files modified:** `app/lib/components/trail/trail_quick_filter_bar.dart`
**Commit:** b57968e4
**Applied fix:** Removed the inline `key: GlobalKey<FormBuilderState>()` from all
three `FormBuilder` widgets. Form values are read via `onChanged` and each child
field has a unique name, so removing the keys prevents state reset on rebuild.

### WR-03: Hardcoded `Brightness.light` in thumbnail placeholder SVGs

**Files modified:** `app/lib/components/list/list_list_item.dart`,
`app/lib/components/trail/trail_list_item.dart`,
`app/lib/components/trail/summit_log_card.dart`
**Commit:** 0c52557c
**Applied fix:** Replaced `Brightness.light.name` with
`Theme.of(context).brightness.name` in all three placeholder builders so the
dark-variant SVG is used in dark mode.

### WR-04: `_ElevationProfileState` mixes instance field `_unit` with `ref.watch`

**Files modified:** `app/lib/components/trail/elevation_profile.dart`
**Commit:** 5314935b
**Applied fix:** Removed the `String _unit` instance field. `build` still
`ref.watch(unitProvider)` to establish the rebuild dependency, and the helper
methods now read `ref.read(unitProvider)` synchronously during the same frame.
**Note:** requires human verification — behavior-preserving refactor on the
elevation rendering path; confirm unit labels update correctly on unit switch.

### WR-05: `SummitLogCard.InkWell` has no `onTap`

**Files modified:** `app/lib/components/trail/summit_log_card.dart`
**Commit:** f8361411
**Applied fix:** Replaced the non-interactive outer `InkWell` with a plain
`Padding` (the card has no navigation action), removing the misleading tappable
surface. The inner photo `GestureDetector` is unchanged.

### WR-06: `_niceInterval` called with zero `maxDist` for single-point GPX

**Files modified:** `app/lib/components/trail/elevation_profile.dart`
**Commit:** 586cb106 (shared with CR-01)
**Applied fix:** Covered by the `if (range <= 0) return 1.0;` guard added in
CR-01 — same function, same root cause.

### WR-07: Language settings save fails silently when `settings` is null

**Files modified:** `app/lib/routes/settings_language_screen.dart`
**Commit:** 2ff2d06e
**Applied fix:** The `onChanged` handler now bails early on null value, and when
`settings == null` shows an error toast (`error_saving_settings`) instead of
silently doing nothing.

### WR-08: `_actorTile` navigates to a local actor with an empty domain

**Files modified:** `app/lib/routes/global_search_screen.dart`
**Commit:** ca6e2cf4
**Applied fix:** `onTap` now builds the route conditionally: local actors use
`/profile/@username` (no trailing `@domain`); remote actors keep the full
`@username@domain` handle.

### WR-09: `_buildElevationPage` silently returns `SizedBox.shrink()` on error

**Files modified:** `app/lib/routes/navigation_screen.dart`
**Commit:** ca762c2f
**Applied fix:** The `error` branch now renders a centered error message using
the existing `error_reading_file` localization key styled with the theme's error
color, instead of a blank `SizedBox.shrink()`.
**Deviation from suggestion:** The review suggested an `error_loading_elevation`
key, which does not exist in the ARB/localization files. Inventing a new key
would require editing all 14 locale files (out of scope), so the closest existing
key, `error_reading_file`, was reused to stay within the current translation
contract.

## Skipped Issues

None — all in-scope findings were fixed.

---

_Fixed: 2026-06-20_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
