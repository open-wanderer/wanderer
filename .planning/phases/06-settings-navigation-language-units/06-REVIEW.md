---
phase: 06-settings-navigation-language-units
reviewed: 2026-06-20T00:00:00Z
depth: standard
files_reviewed: 25
files_reviewed_list:
  - app/lib/components/list/list_card.dart
  - app/lib/components/list/list_list_item.dart
  - app/lib/components/trail/elevation_profile.dart
  - app/lib/components/trail/summit_log_card.dart
  - app/lib/components/trail/trail_card.dart
  - app/lib/components/trail/trail_list_item.dart
  - app/lib/components/trail/trail_panel.dart
  - app/lib/components/trail/trail_quick_filter_bar.dart
  - app/lib/components/trail/trail_timeline.dart
  - app/lib/components/trail/waypoint_sheet.dart
  - app/lib/main.dart
  - app/lib/provider/local_settings_provider.dart
  - app/lib/provider/router_provider.dart
  - app/lib/routes/global_search_screen.dart
  - app/lib/routes/list_detail_screen.dart
  - app/lib/routes/navigation_screen.dart
  - app/lib/routes/settings_language_screen.dart
  - app/lib/routes/settings_notifications_screen.dart
  - app/lib/routes/settings_privacy_screen.dart
  - app/lib/routes/settings_screen.dart
  - app/lib/routes/trail_filter_screen.dart
  - app/test/provider/unit_provider_test.dart
  - app/test/routes/settings_language_screen_test.dart
  - app/test/routes/settings_screen_test.dart
  - app/test/util/format_util_test.dart
findings:
  critical: 5
  warning: 9
  info: 4
  total: 18
status: issues_found
---

# Phase 06: Code Review Report

**Reviewed:** 2026-06-20
**Depth:** standard
**Files Reviewed:** 25
**Status:** issues_found

## Summary

This phase delivers settings navigation (account / privacy / language / notifications / appearance), language/unit selection, and unit-aware display across trail cards, list cards, the elevation profile, and the navigation screen. The Riverpod `unitProvider` / `localeProvider` wiring is sound at the provider level and the tests exercise the critical paths. However several bugs — ranging from a crash on flat GPX tracks to a guaranteed tab-controller listener accumulation — must be fixed before ship.

---

## Critical Issues

### CR-01: `_niceInterval` crashes / produces `NaN` on flat terrain (zero elevation range)

**File:** `app/lib/components/trail/elevation_profile.dart:650-651`

**Issue:** `_niceInterval` computes `raw = range / targetCount`. When `range == 0` (every GPX point has the same elevation — a flat trail, or a track with a single altitude value), `raw = 0` and `log(0)` produces `-Infinity`. `pow(10, (-Infinity).floor())` evaluates to `0.0`. Passing `horizontalInterval: 0` and `interval: 0` to `fl_chart` axes throws an assertion error at runtime, crashing the elevation-profile widget for any flat trail.

This also affects `xInterval` when `maxDist == 0` (a single-point GPX), though that case is rarer.

**Fix:**
```dart
double _niceInterval(double range, int targetCount) {
  if (range <= 0) return 1.0; // guard against flat/zero-range data
  final raw = range / targetCount;
  final magnitude = pow(10, (log(raw) / ln10).floor()).toDouble();
  final residual = raw / magnitude;
  if (residual <= 1) return magnitude;
  if (residual <= 2) return 2 * magnitude;
  if (residual <= 5) return 5 * magnitude;
  return 10 * magnitude;
}
```

---

### CR-02: Tab-controller listener accumulates on every dependency change (`_TabContentState`)

**File:** `app/lib/components/trail/trail_panel.dart:336-339`

**Issue:** `didChangeDependencies` is called by Flutter every time an `InheritedWidget` dependency of the widget changes (e.g., theme, locale, or any provider the parent injects). Each call to `didChangeDependencies` registers a fresh listener on the `TabController` via `controller.addListener(_onTabChanged)` without ever removing the previous one. After N dependency updates, `_onTabChanged` is called N times per tab change, causing N redundant `setState` calls and potential exponential growth. There is no `dispose()` override in `_TabContentState`, so the listeners are never cleaned up.

**Fix:**
```dart
class _TabContentState extends State<_TabContent> {
  int _index = 0;
  TabController? _controller;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final controller = DefaultTabController.of(context);
    if (_controller != controller) {
      _controller?.removeListener(_onTabChanged);
      _controller = controller;
      _controller!.addListener(_onTabChanged);
    }
  }

  @override
  void dispose() {
    _controller?.removeListener(_onTabChanged);
    super.dispose();
  }

  void _onTabChanged() {
    final controller = DefaultTabController.of(context);
    if (!controller.indexIsChanging) {
      setState(() => _index = controller.index);
    }
  }

  @override
  Widget build(BuildContext context) {
    return widget.children[_index];
  }
}
```

---

### CR-03: Force-unwrap of nullable `author` crashes `TrailPanel` when expand data is absent

**File:** `app/lib/components/trail/trail_panel.dart:120, 139`

**Issue:** The Trail model defines `expand.author` as `Actor?` (nullable). `TrailPanel` force-unwraps it in two places:
- Line 120: `trail.expand!.author!.preferredUsername`
- Line 139: `trail.expand!.author!.preferredUsername` / `.domain`

If the API response omits the `expand.author` relation (e.g., federated trail, author deleted, or server omits expansion), both accesses throw `Null check operator used on a null value`, crashing the trail detail panel.

**Fix:**
```dart
// Guard the entire author section
final author = trail.expand?.author;
if (author != null)
  InkWell(
    onTap: () => context.push(
      '/profile/@${author.preferredUsername}@${author.domain}',
    ),
    child: Padding(
      padding: const EdgeInsets.all(4.0),
      child: Row(
        children: [
          // avatar ...
          Text('@${author.preferredUsername}@${author.domain}',
            style: Theme.of(context).textTheme.labelLarge),
        ],
      ),
    ),
  ),
```

---

### CR-04: Distance range slider max label uses `formatElevation` instead of `formatDistance`

**File:** `app/lib/routes/trail_filter_screen.dart:230`

**Issue:** The RangeSlider for "Distance" renders its max label using `formatElevation`:
```dart
"${formatElevation(filter.value?.distanceMax, unit: unit)}..."
```
`formatElevation` converts meters to feet in imperial mode, while `formatDistance` converts meters to miles. For a trail of 10 km (10,000 m), the max label shows "32,808 ft" instead of "6.21 mi" in imperial mode, and "10000 m" instead of "10.00 km" in metric mode. This is also visually inconsistent with the min label, which correctly uses `formatDistance`.

**Fix:**
```dart
labels: RangeLabels(
  formatDistance(filter.value?.distanceMin, unit: unit),
  "${formatDistance(filter.value?.distanceMax, unit: unit)}"
    "${filter.value?.distanceMax == filter.value?.distanceLimit ? "+" : ""}",
),
```

---

### CR-05: Duplicate `name: "start"` on both `WandererDatePicker` fields breaks form state

**File:** `app/lib/routes/trail_filter_screen.dart:306, 322`

**Issue:** `WandererDatePicker` extends `FormBuilderField`, which requires each field within a `FormBuilder` scope to have a unique `name` for its `FormFieldState` key. Both the "Before" (start date) and "After" (end date) pickers are assigned `name: "start"`. Since `TrailFilterScreen` does not wrap them in a `FormBuilder`, they operate as orphan `FormBuilderField`s and the name collision is currently harmless for value retrieval (the `onChanged` callback is used directly). However, the same name on two fields that could be placed inside a `FormBuilder` in a future refactor, or if a wrapping `FormBuilder` is added, will cause the second field to silently overwrite the first in form state, losing the start-date value.

The "Before/After" semantics are also semantically correct (before = startDate = upper bound; after = endDate = lower bound) — that mapping is consistent with the filter logic at lines 36–41 in `trail_quick_filter_bar.dart`.

**Fix:** Change the second picker's name to avoid future collisions regardless:
```dart
WandererDatePicker(
  name: "end",          // was "start" — must be unique
  initialValue: filter.value?.endDate,
  ...
),
```

---

## Warnings

### WR-01: `authProvider.value!` force-unwrap in five widget `build` methods

**File:** `app/lib/components/list/list_card.dart:27`, `app/lib/components/list/list_list_item.dart:20`, `app/lib/components/trail/trail_card.dart:36`, `app/lib/components/trail/trail_list_item.dart:26`, `app/lib/components/trail/summit_log_card.dart:24`

**Issue:** All five widgets call `ref.watch(authProvider).value!` directly in `build`. If the auth provider is in a loading or error state (immediately after app cold-start, token refresh, or logout race), `.value!` throws `Null check operator used on a null value`. The project uses `requireValue!` in `TrailPanel` and `ListDetailScreen`, which at least provides a clearer error message, but both patterns are unsafe here because these components can appear during loading state before auth settles.

**Fix:** Guard with a null-safe pattern or bail early:
```dart
final user = ref.watch(authProvider).valueOrNull;
if (user == null) return const SizedBox.shrink();
```

---

### WR-02: `GlobalKey<FormBuilderState>()` constructed inline inside `build` re-creates the key on each rebuild

**File:** `app/lib/components/trail/trail_quick_filter_bar.dart:548, 568, 639`

**Issue:** Three `FormBuilder` widgets are given `key: GlobalKey<FormBuilderState>()` as inline expressions inside the `builder` callback of `DraggableScrollableSheet` → `Consumer`. Every time the `Consumer` rebuilds (which happens when the filter provider changes, e.g., when the user adjusts the range slider), a new `GlobalKey` is created, forcing Flutter to remount the `FormBuilder` and discard its state. This causes the pickers/radio group to reset to their default values during live interaction.

**Fix:** Hoist the keys to a `StatefulWidget` field, or — since form values are read through `onChanged` callbacks rather than via the form key — simply remove the `key:` parameter from these `FormBuilder` widgets.

---

### WR-03: Hardcoded `Brightness.light` in thumbnail placeholder SVGs

**File:** `app/lib/components/list/list_list_item.dart:143`, `app/lib/components/trail/trail_list_item.dart:248`, `app/lib/components/trail/summit_log_card.dart:278`

**Issue:** The placeholder SVG path is hard-coded to `empty_state_trail_light.svg` regardless of the current theme:
```dart
"assets/svgs/empty_state_trail_${Brightness.light.name}.svg"
```
In dark mode, the dark-variant SVG (`empty_state_trail_dark.svg`) is never used, causing the placeholder to clash visually with the dark background.

Compare with the correct pattern used in `list_card.dart:103` and `trail_card.dart:309`:
```dart
"assets/svgs/empty_state_trail_${Theme.of(context).brightness.name}.svg"
```

**Fix:** Replace `Brightness.light.name` with `Theme.of(context).brightness.name` in all three files.

---

### WR-04: `_ElevationProfileState` mixes instance field `_unit` with `ref.watch` inside `build`

**File:** `app/lib/components/trail/elevation_profile.dart:40, 110`

**Issue:** `_unit` is declared as an instance field initialized to `'metric'` (line 40) and then overwritten in `build` via `_unit = ref.watch(unitProvider)` (line 110). Writing to instance state inside `build` is an anti-pattern: if `build` is skipped or called during a pump cycle after widget suspension, the field retains the stale value. More importantly, the `_buildLineGradient` and `_buildFillGradient` methods are called from `_buildChart`, which is called from `build` — they all safely read `_unit` synchronously during the same frame so the current code works, but the field serves no purpose as state.

**Fix:** Remove the instance field and use a local variable inside `build`:
```dart
@override
Widget build(BuildContext context) {
  final unit = ref.watch(unitProvider);
  // pass `unit` explicitly to _buildChart / _buildStatText
}
```

---

### WR-05: `SummitLogCard.InkWell` has no `onTap` — renders as non-interactive tappable surface

**File:** `app/lib/components/trail/summit_log_card.dart:61-63`

**Issue:** The outer `InkWell` wrapping the card has no `onTap` callback:
```dart
child: InkWell(
  borderRadius: BorderRadius.circular(16),
  child: Padding(...)
```
Without `onTap`, `InkWell` never shows a ripple, but users may still attempt to tap the card expecting a navigation or expansion action. The `GestureDetector` inside for the photo handles photo navigation, but the overall card is a dead interactive region.

**Fix:** Either add `onTap` if the card is intended to be navigable (e.g., open a summit log detail), or replace `InkWell` with a plain `Padding` widget to avoid implying interactivity.

---

### WR-06: `_niceInterval` called with potentially-zero `maxDist` when trail has a single GPX point

**File:** `app/lib/components/trail/elevation_profile.dart:124`

**Issue:** When a GPX has exactly one point, `_parseGpx` returns a list of length 1, `maxDist = _points.last.distanceM = 0.0`, and `_niceInterval(0, 5)` produces the same `log(0)` crash path described in CR-01 (shared function). This is a second trigger of the same underlying bug and should be fixed by the same guard in `_niceInterval`.

**Fix:** Covered by the fix in CR-01 (`if (range <= 0) return 1.0`).

---

### WR-07: Language settings save may fail silently when `settings` is `null` at widget build time

**File:** `app/lib/routes/settings_language_screen.dart:77-79`

**Issue:** The `onChanged` callback of `RadioGroup<Language>` only fires `_save` when both `value != null && settings != null`. If `settingsProvider` returns `null` (first app launch, before the first server sync), tapping a language radio tile silently does nothing — no toast, no error, no feedback. The user sees the radio appear to select but nothing persists.

**Fix:** Show a toast when `settings == null`:
```dart
onChanged: (value) {
  if (value == null) return;
  if (settings == null) {
    ref.read(toastProvider.notifier).add(ToastMessage(
      type: ToastType.error,
      icon: FontAwesomeIcons.circleExclamation,
      text: l10n.error_saving_settings,
    ));
    return;
  }
  _save(ref, l10n, settings.copyWith(language: value));
},
```

---

### WR-08: `_actorTile` navigates to a local actor with an empty domain, producing a malformed route

**File:** `app/lib/routes/global_search_screen.dart:363`

**Issue:** `_ActorTile.onTap` always constructs the route as:
```dart
context.push('/profile/@${actor.preferredUsername}@${actor.domain}')
```
When `actor.isLocal == true`, `actor.domain` may be empty or `""`. The resulting route becomes `/profile/@username@` — a trailing `@` with no domain. This may not match the router's `/profile/:handle` pattern correctly (depending on how the server handle endpoint parses it), and it is inconsistent with how `_ActorTile` constructs `subtitle` (which correctly omits `@domain` for local actors).

**Fix:**
```dart
onTap: () {
  final route = actor.isLocal
      ? '/profile/@${actor.preferredUsername}'
      : '/profile/@${actor.preferredUsername}@${actor.domain}';
  context.push(route);
},
```

---

### WR-09: `_buildElevationPage` in `NavigationScreen` silently returns `SizedBox.shrink()` on error

**File:** `app/lib/routes/navigation_screen.dart:731-735`

**Issue:** When `trailAsync` is in an error state (e.g., network failure after entering navigation), `_buildElevationPage` returns `SizedBox.shrink()` with no user feedback. In a navigation context this is particularly poor UX because the user has tapped the elevation button and sees a blank area with no indication that the GPX failed to load. The same method also returns `SizedBox.shrink()` when `gpx == null` (line 724), indistinguishable from the error case.

**Fix:** Return a minimal error indicator for the error case:
```dart
error: (e, _) => Center(
  child: Text(
    localizations.error_loading_elevation,
    style: TextStyle(color: Theme.of(context).colorScheme.error),
  ),
),
```

---

## Info

### IN-01: `_languageNames` map will cause a `KeyError` crash if a new `Language` enum value is added without a corresponding entry

**File:** `app/lib/routes/settings_language_screen.dart:85-88`

**Issue:** `_languageNames[language]!` force-dereferences the map lookup. The map is hardcoded with 14 entries matching the current 14 `Language` enum values. Adding a 15th language to the enum without updating the map will throw at runtime in the language screen. The test at line 37 (`Language.values.length`) will also begin failing, which provides a partial safety net — but only for tests, not for the map lookup crash.

**Fix:** Use `_languageNames[language] ?? language.name` or add an assertion/analysis-only lint to ensure map coverage.

---

### IN-02: Commented-out code left in `trail_card.dart`

**File:** `app/lib/components/trail/trail_card.dart:387-391`

**Issue:** Dead code block left in the `_StatsGrid` builder:
```dart
// _StatIcon(
//   icon: FontAwesomeIcons.gaugeHigh,
//   value: _getDifficultyLabel(context, trail.difficulty),
// ),
```
This references a `_StatIcon` widget that does not appear to exist in the current codebase.

**Fix:** Remove the commented-out block.

---

### IN-03: `MainApp` constructor missing `const`

**File:** `app/lib/main.dart:51`

**Issue:** `class MainApp extends ConsumerWidget { const MainApp({super.key}); }` — the constructor is already `const`, but `runApp(ProviderScope(..., child: MainApp()))` at line 47 calls it without `const`. A minor missed optimization.

**Fix:**
```dart
child: const MainApp(),
```

---

### IN-04: `_visibilty_status` key is a misspelling carried through all localization files

**File:** `app/lib/routes/trail_filter_screen.dart:155-156`

**Issue:** The filter screen uses `AppLocalizations.of(context)?.visibilty_status` (missing an `i` in "visibility"). The key exists in the ARB files and generated localizations under the same misspelled name, so it works — but the typo is baked into the public translation contract and all 14 locale files. Fixing it later will require renaming the key everywhere.

**Fix:** This is an existing typo in the shared i18n layer. If the ARB files are regenerated, rename `visibilty` → `visibility` and `visibilty_status` → `visibility_status` across all locale files simultaneously to avoid a broken intermediate state.

---

_Reviewed: 2026-06-20_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
