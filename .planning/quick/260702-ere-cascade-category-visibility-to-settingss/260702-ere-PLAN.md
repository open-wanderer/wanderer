---
phase: quick-260702-ere
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - app/lib/routes/settings_subcategories_screen.dart
autonomous: true
requirements: [QUICK-260702-ERE]

must_haves:
  truths:
    - "When a parent category is disabled, every subcategory Switch renders OFF regardless of that subcategory's own stored preference"
    - "When a parent category is disabled, subcategory Switches are non-interactive (onChanged null) and the row is dimmed to 0.6 opacity"
    - "When a parent category is disabled, subcategory rows cannot be drag-reordered (no ReorderableDragStartListener on the drag handle)"
    - "When the parent category is re-enabled, each subcategory's Switch restores its previously-stored visible value with zero persistence writes having occurred"
    - "SettingsCategoriesScreen is unchanged"
  artifacts:
    - path: "app/lib/routes/settings_subcategories_screen.dart"
      provides: "Presentation-only parent-category visibility cascade into subcategory rows"
      contains: "categoryPreferenceProvider"
  key_links:
    - from: "app/lib/routes/settings_subcategories_screen.dart"
      to: "categoryVisible() in app/lib/util/category_preference_sort.dart"
      via: "computed categoryVisible local from watched categoryPreferenceProvider"
      pattern: "categoryVisible\\(widget\\.category\\.id"
    - from: "_buildRow effectiveVisible"
      to: "Switch value + onChanged gating"
      via: "categoryVisible && subcategoryVisible(sub.id, prefs)"
      pattern: "categoryVisible && subcategoryVisible"
---

<objective>
Cascade parent-category visibility into `SettingsSubcategoriesScreen` so that when the
parent category is disabled, every subcategory toggle renders OFF, becomes non-interactive
(disabled + dimmed), and its row cannot be drag-reordered — mirroring the web reference
`web/src/routes/settings/categories/+page.svelte` cascade
(`categoryVisible`, `class:opacity-60={!categoryVisible}`, `disabled={... || !categoryVisible}`).

Purpose: A user navigating into a disabled category's subcategory screen currently sees
fully-interactive toggles with no indication the parent is hidden, producing confusing,
functionally-inert edits (a subcategory can never be shown while its parent is hidden).

Output: Modified `app/lib/routes/settings_subcategories_screen.dart`. This is a
presentation-only read+render change — NO persistence logic is touched, and each
subcategory's own stored `visible` value is never mutated by the parent's state.
</objective>

<execution_context>
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/workflows/execute-plan.md
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md

# File to modify (full current implementation)
@app/lib/routes/settings_subcategories_screen.dart

# Reference: sibling screen already combines category + preference AsyncValues and uses categoryVisible()
@app/lib/routes/settings_categories_screen.dart

# Existing pure helpers (already imported in the target file) — do NOT duplicate
@app/lib/util/category_preference_sort.dart

# Provider to watch (already imported in the sibling screen, NOT yet in the target)
@app/lib/provider/category_preference_provider.dart

# Canonical parity reference — cascade lives around lines 810-1043
@web/src/routes/settings/categories/+page.svelte
</context>

<tasks>

<task type="auto">
  <name>Task 1: Cascade parent-category visibility into subcategory rows (presentation-only)</name>
  <files>app/lib/routes/settings_subcategories_screen.dart</files>
  <action>
Modify `SettingsSubcategoriesScreen` to mirror the web cascade in
`web/src/routes/settings/categories/+page.svelte`. Confirm before editing that
`categoryPreferenceProvider` and `categoryVisible()` are the existing symbols
(grep — do not invent new providers or helpers).

Imports: add `import 'package:wanderer/models/category_preference.dart';` and
`import 'package:wanderer/provider/category_preference_provider.dart';`. Both
`categoryVisible()` and `subcategoryVisible()` already come from the
already-imported `category_preference_sort.dart` — do not re-import or redefine them.

In `build()`: alongside the existing `subcategoryProvider` and
`subcategoryPreferenceProvider` watches, add
`final categoryPrefsAsync = ref.watch(categoryPreferenceProvider);`. Combine BOTH
async values (`prefsAsync` for subcategory prefs and `categoryPrefsAsync`) into a
single `AsyncValue` record for the `AsyncLoader`, following the `combined` record
pattern in `settings_categories_screen.dart`'s `build()` (error-first, then
loading, then data). Use a record shape such as
`({List<SubcategoryPreference> prefs, List<CategoryPreference> categoryPrefs})`
and update the `AsyncLoader` type parameter, `mockData`, and `builder(data)`
signature accordingly. `subcategoryProvider` returns a plain (non-async) list, so
it stays a direct watch and is not part of the combined record.

Compute the parent visibility ONCE per build using the existing helper:
`final catVisible = categoryVisible(widget.category.id, data.categoryPrefs);`.
(Name the local `catVisible` to avoid shadowing the `categoryVisible` function.)
Thread `catVisible` through `_buildList` -> `_buildRow` (add a `bool categoryVisible`
parameter to each, or pass the value; pick a clear param name that does not shadow
the helper).

In `_buildRow`: replace the current `final isVisible = subcategoryVisible(sub.id, prefs);`
usage with `final effectiveVisible = catVisible && subcategoryVisible(sub.id, prefs);`.
Use `effectiveVisible` for the `Switch`'s `value:`. Set the `Switch`'s `onChanged:` to
`catVisible ? (v) => _onToggle(sub, v) : null` — passing `null` renders the Switch as
disabled/non-interactive (Flutter's standard mechanism; mirrors web's
`disabled={... || !categoryVisible}`).

Dim the row when the parent is off: wrap the existing row content (the `Row(...)`
currently inside the `Padding`) in an `Opacity` with
`opacity: catVisible ? 1.0 : 0.6`, mirroring web's `class:opacity-60={!categoryVisible}`.
Apply this dimming ONLY for the parent-off case — check the current row: today there
is no independent per-row dimming for a subcategory's own OFF state, so do NOT add any;
only the parent cascade dims.

Disable drag-reordering per row when the parent is off: today the drag handle is a
`ReorderableDragStartListener(index: index, child: Padding(... Icon(Icons.drag_handle)))`.
When `catVisible` is false, render a plain dimmed `Padding(... Icon(Icons.drag_handle))`
with NO `ReorderableDragStartListener` wrapper instead; when `catVisible` is true, keep
the existing listener-wrapped handle. Mirrors web's drag hit-area
`disabled={reordering || !categoryVisible}`. Because all subcategories under one parent
share the same `catVisible`, all rows drag-disable together when the category is off —
that is correct and matches web.

Do NOT modify `_onToggle`, `_onToggleOff`, `_onReorder`, `_save`, `_viewOwnTrails`,
`_buildEmptyState`, or any persistence/notifier call. This is a read+render change only:
`effectiveVisible` derives from `catVisible && subcategoryVisible(...)` without mutating
stored state, so re-enabling the category later restores each subcategory's saved
preference for free. Do NOT touch `SettingsCategoriesScreen` (per D-05 it deliberately
renders no inline subcategory badges).
  </action>
  <verify>
    <automated>cd app && dart analyze lib/routes/settings_subcategories_screen.dart 2>&1 | grep -vE '^\s*(info|Analyzing|Info)' | grep -iE 'error|warning' || echo "NO_ERRORS_OR_WARNINGS"</automated>
  </verify>
  <done>
- `dart analyze lib/routes/settings_subcategories_screen.dart` reports 0 errors and 0 warnings (pre-existing info-level lints acceptable).
- `categoryPreferenceProvider` is watched in `build()` and combined with the subcategory prefs into the `AsyncLoader`'s asyncValue.
- `_buildRow` uses `categoryVisible && subcategoryVisible(...)` for the Switch `value:`; `onChanged:` is `null` when the parent category is off.
- Row is wrapped in `Opacity(opacity: catVisible ? 1.0 : 0.6, ...)`; drag handle drops its `ReorderableDragStartListener` when the parent is off.
- No new `upsert`/`reorder`/notifier persistence calls were introduced (grep-confirmed below).
- `SettingsCategoriesScreen` is byte-for-byte unchanged.
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| local widget state → backend preference API | Persistence calls (`upsert`/`reorder`) cross to the server; this change must NOT introduce new crossings |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-ERE-01 | Tampering | subcategory `visible` stored value | mitigate | Cascade derives display state (`effectiveVisible`) purely from read data; verification greps to confirm no new `upsert`/`reorder` calls were added, so no stored value is mutated by the parent's state |
| T-ERE-02 | Information disclosure | privacy filtering elsewhere | accept | Screen is a settings UI reading the current user's own preferences only; no cross-user data, no new network reads beyond the already-existing `categoryPreferenceProvider` |
</threat_model>

<verification>
Run from repo root after Task 1:

1. Static analysis — 0 new errors/warnings:
   `cd app && dart analyze lib/routes/settings_subcategories_screen.dart`
   (info-level pre-existing lints acceptable; no error/warning lines.)

2. No new persistence side-effects — the cascade must be read+render only. Confirm the
   only notifier `.upsert(` / `.reorder(` call sites are the pre-existing ones inside
   `_onToggle`/`_onToggleOff`/`_onReorder` (unchanged by this plan) and that none were
   added to `build`, `_buildList`, or `_buildRow`:
   `grep -nE '\.(upsert|reorder)\(' app/lib/routes/settings_subcategories_screen.dart`
   Expect exactly the three pre-existing `subcategoryPreferenceProvider.notifier` calls
   (two `upsert(sub.id, ...)` in `_onToggle`/`_onToggleOff`, one `reorder(...)` in
   `_onReorder`) and zero calls on `categoryPreferenceProvider.notifier`:
   `grep -nE 'categoryPreferenceProvider\.notifier' app/lib/routes/settings_subcategories_screen.dart`
   Expect NO matches (category prefs are read-only here).

3. Sibling screen untouched:
   `git diff --stat app/lib/routes/settings_categories_screen.dart`
   Expect empty output (no changes).

4. Cascade wiring present:
   `grep -nE 'categoryVisible\(widget\.category\.id|categoryVisible && subcategoryVisible' app/lib/routes/settings_subcategories_screen.dart`
   Expect both patterns present.
</verification>

<success_criteria>
- With the parent category disabled, all subcategory toggles show OFF, are non-interactive, the rows are dimmed (0.6), and drag-reordering is disabled.
- With the parent category enabled, behavior is identical to before this change (each subcategory reflects its own stored `visible`, fully interactive and reorderable).
- No persistence calls were added; each subcategory's stored preference is untouched, so re-enabling the parent restores prior subcategory visibility exactly.
- `SettingsCategoriesScreen` is unmodified.
- `dart analyze` on the target file reports 0 errors/warnings.
</success_criteria>

<output>
Create `.planning/quick/260702-ere-cascade-category-visibility-to-settingss/260702-ere-SUMMARY.md` when done.
</output>
