---
phase: quick-260702-gib
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - app/lib/routes/settings_categories_screen.dart
autonomous: true
requirements: []
must_haves:
  truths:
    - "Under each category row, a wrap of read-only chips renders — one chip per subcategory of that category"
    - "Each chip shows the subcategory's badge_icon (when present) and its locale-resolved display name"
    - "A chip is visually dimmed when that subcategory's OWN visibility preference is off (subcategoryVisible == false), independent of the parent category's visibility"
    - "No chip is tappable/interactive in either the visible or dimmed state"
    - "settings_subcategories_screen.dart is not modified — subcategory toggling still happens only there"
  artifacts:
    - path: "app/lib/routes/settings_categories_screen.dart"
      provides: "Per-category subcategory chip row wired into _buildRow via subcategoryProvider + subcategoryPreferenceProvider"
      contains: "Wrap"
  key_links:
    - from: "app/lib/routes/settings_categories_screen.dart"
      to: "subcategoryPreferenceProvider"
      via: "ref.watch folded into combined AsyncValue record (hasValue-based combine)"
      pattern: "subcategoryPreferenceProvider"
    - from: "app/lib/routes/settings_categories_screen.dart"
      to: "subcategoryVisible"
      via: "dim state per chip"
      pattern: "subcategoryVisible\\("
---

<objective>
Add a read-only row of subcategory chips beneath each category row in `SettingsCategoriesScreen`. For every subcategory belonging to a category, render a non-interactive chip containing that subcategory's `badge_icon` (when present) plus its locale-resolved display name. Dim the chip when the subcategory's OWN visibility preference is off — driven solely by `subcategoryVisible(subcategory.id, subcategoryPrefs)`, NOT compounded with the parent category's visibility.

Purpose: Bring the Flutter category-settings screen to parity with the web reference (`web/src/routes/settings/categories/+page.svelte` ~863-885), which surfaces subcategory chips inline so an admin sees each category's subcategories at a glance without opening the leaf screen. Per the user's explicit instruction, this task diverges from the web reference's compound `categoryVisible && isSubcategoryVisible(...)` — the chip dims only on the subcategory's own toggle.

Output: A modified `app/lib/routes/settings_categories_screen.dart` that watches two additional providers, folds `subcategoryPreferenceProvider` into the existing combined `AsyncValue` record (preserving the already-fixed `hasValue`-based combine — no regression to `isLoading`), and renders a `Wrap` of bare `Chip` widgets per category row.
</objective>

<execution_context>
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/workflows/execute-plan.md
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md

# File to modify (read the CURRENT state — the hasValue-combine fix is already applied)
@app/lib/routes/settings_categories_screen.dart

# Existing helpers/providers to reuse verbatim (do NOT reimplement)
@app/lib/util/category_preference_sort.dart
@app/lib/util/category_icon_util.dart
@app/lib/models/subcategory.dart
@app/lib/provider/trail/subcategory_provider.dart
@app/lib/provider/subcategory_preference_provider.dart

# Sibling screen — the canonical patterns for the SAME providers/helpers
# (subcategoryProvider filter by `s.category == id`, subcategoryFilterAvatar
#  usage, displayName(locale), subcategoryVisible). Do NOT modify this file.
@app/lib/routes/settings_subcategories_screen.dart

# Parity reference only (chip shape: icon + truncated label, dimmed when off)
@web/src/routes/settings/categories/+page.svelte
</context>

<tasks>

<task type="auto">
  <name>Task 1: Watch subcategory data and fold subcategory prefs into the combined AsyncValue</name>
  <files>app/lib/routes/settings_categories_screen.dart</files>
  <action>
Extend the screen's data intake so the subcategory chips have their source data, WITHOUT regressing the already-fixed `hasValue`-based combine (STATE.md notes this pattern was fixed twice already — extend it to a third field, never revert to an `isLoading`-based combine).

1. Add imports at the top of the file, matching the sibling `settings_subcategories_screen.dart` import set:
   - `package:wanderer/models/subcategory.dart`
   - `package:wanderer/models/subcategory_preference.dart`
   - `package:wanderer/provider/subcategory_preference_provider.dart`
   - `package:wanderer/provider/trail/subcategory_provider.dart`

2. In `build()`, after the existing `prefsAsync` watch, add two watches — mirroring how the sibling screen watches them:
   - `subcategoryProvider` (synchronous `List<Subcategory>`) — bind to a local `subcategories` variable. Like `categoryProvider`, it is NOT async, so it does NOT participate in the error/loading branches of the combine (confirm against `subcategory_provider.dart`: its `build()` returns a plain `List<Subcategory>`, not a `Future`).
   - `subcategoryPreferenceProvider` (`AsyncValue<List<SubcategoryPreference>>`) — bind to `subcategoryPrefsAsync`. This IS async and MUST join the combine.

3. Extend the `combined` record TYPE from `({List<Category> categories, List<CategoryPreference> prefs})` to `({List<Category> categories, List<CategoryPreference> prefs, List<SubcategoryPreference> subcategoryPrefs})`. Update EVERY occurrence of the record type annotation (the `.error(...)`, `.data(...)`, `.loading()`, the `AsyncLoader<...>` generic, and the `mockData` literal — `mockData` gains `subcategoryPrefs: []`).

4. Extend the combine expression, preserving the exact error-first → hasValue → loading ordering already present:
   - Error branch: add a third `: subcategoryPrefsAsync.hasError ? AsyncValue<...>.error(subcategoryPrefsAsync.error!, subcategoryPrefsAsync.stackTrace!)` clause after the existing `prefsAsync.hasError` clause.
   - Data branch guard: change `categoriesAsync.hasValue && prefsAsync.hasValue` to `categoriesAsync.hasValue && prefsAsync.hasValue && subcategoryPrefsAsync.hasValue`.
   - Data payload: add `subcategoryPrefs: subcategoryPrefsAsync.value ?? const []`.
   - Keep the trailing `const AsyncValue<...>.loading()` fallback (the record type inside it also grows the third field).
   Do NOT introduce any `isLoading ||` combine condition — the guard stays `hasValue`-based.

5. Thread the new data down to the row builder. `subcategories` (sync) is available directly in `build()` — pass it (or the pre-filtered slice) plus `data.subcategoryPrefs` into `_buildList` → `_buildRow`. Add parameters `List<Subcategory> subcategories` and `List<SubcategoryPreference> subcategoryPrefs` to both `_buildList` and `_buildRow` signatures and forward them through. Do NOT touch the reorder round-trip logic, `_orderedIds` seeding, `_onReorder`, `_onToggle`, `_onToggleOff`, or `_viewOwnTrails`.

This task compiles with the chips not yet rendered (Task 2 consumes the threaded params); `dart analyze` may report unused-parameter info-lints between tasks, which Task 2 resolves.
  </action>
  <verify>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/app && dart analyze lib/routes/settings_categories_screen.dart 2>&1 | grep -v '^#' | grep -Ei 'error' ; test ${PIPESTATUS[1]} -eq 1</automated>
  </verify>
  <done>File imports the two new providers + two models; `build()` watches `subcategoryProvider` and `subcategoryPreferenceProvider`; the `combined` record carries a third `subcategoryPrefs` field with an error-first/`hasValue`/loading combine (no `isLoading ||`); `_buildList`/`_buildRow` receive the subcategory list + prefs. `dart analyze` reports no `error`-level diagnostics.</done>
</task>

<task type="auto">
  <name>Task 2: Render the read-only subcategory chip Wrap under each category row</name>
  <files>app/lib/routes/settings_categories_screen.dart</files>
  <action>
In `_buildRow`, render a `Wrap` of non-interactive chips beneath the existing category `Row`. Wrap the current `Row` and the new chip row in a `Column` (the outer `Padding` stays; its `child` becomes the `Column`).

1. Compute this category's subcategories from the threaded `subcategories` param:
   `final subs = subcategories.where((s) => s.category == category.id).toList();`
   (Same filter pattern the sibling screen uses.) Optionally sort with `sortedSubcategoriesByPreference(subs, ...)` — but the user spec does not require a specific chip order, so a plain filter is acceptable; if you sort, reuse the existing helper, do not hand-roll.

2. Only render the chip row when `subs.isNotEmpty` (mirrors the web `{#if childSubcategories.length > 0}` guard). When empty, render nothing extra (the row is unchanged).

3. Build the chip row as a `Wrap` (Flutter's flex-wrap equivalent of the web `flex flex-wrap`) with modest `spacing` and `runSpacing` (e.g. 6) and left-aligned under the category label. For each subcategory `s` in `subs`, build one chip:
   - Dim state: `final dimmed = !subcategoryVisible(s.id, subcategoryPrefs);` — use ONLY the subcategory's own visibility. Do NOT AND with `categoryVisible(...)`; this deliberately diverges from the web reference per the user's explicit instruction.
   - Chip widget: a bare `Chip` from `package:flutter/material.dart` (already imported). Use:
     - `avatar:` the subcategory badge/icon. Reuse `subcategoryFilterAvatar(context, s, category, locale)` (already imported from `category_icon_util.dart`) — it renders the FA icon with the `badge_icon` overlay and null-handles a missing `badge_icon` gracefully. (Passing the parent `category` gives the icon-fallback the sibling screen relies on.)
     - `label:` `Text(s.displayName(locale), overflow: TextOverflow.ellipsis)`.
     - Keep the chip compact: set `visualDensity: VisualDensity.compact` and/or `materialTapTargetSize: MaterialTapTargetSize.shrinkWrap` and `labelStyle: Theme.of(context).textTheme.bodySmall` to approximate the web `text-xs` pill; exact spacing follows Material defaults, not literal Tailwind parity.
   - Wrap each `Chip` in `Opacity(opacity: dimmed ? 0.5 : 1.0, child: Chip(...))` to render the dim state (mirrors web `opacity-50`, matching the sibling screen's `Opacity`-based dim approach).
   - MUST be non-interactive: use the BARE `Chip` only. Do NOT use `ActionChip`, `ChoiceChip`, `InputChip`, or `FilterChip` (all interactive). Do NOT add `onTap`, `onPressed`, `onDeleted`, `InkWell`, or `GestureDetector` anywhere in the chip subtree. A bare `Chip` has no tap affordance, satisfying "no interaction in either state".

4. To keep the chips visually associated with the category (and not overlapping the drag handle), place the `Wrap` inside the `Column` aligned with the label — e.g. add left padding roughly matching the avatar + gap, or nest it under the `Expanded` body area. Presentation detail; use existing Material spacing conventions, keep it readable.

Do NOT modify `settings_subcategories_screen.dart`. Do NOT alter the Switch, drag handle, navigation InkWell (that InkWell is the pre-existing category-body nav — leave it), reorder logic, or any toggle handler.
  </action>
  <verify>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/app && dart analyze lib/routes/settings_categories_screen.dart 2>&1 | grep -v '^#' | grep -Ei 'error' ; test ${PIPESTATUS[1]} -eq 1</automated>
  </verify>
  <done>`_buildRow` renders a `Wrap` of bare `Chip` widgets (guarded by `subs.isNotEmpty`), each with `subcategoryFilterAvatar` + `displayName(locale)`, wrapped in `Opacity` driven solely by `!subcategoryVisible(s.id, subcategoryPrefs)`. No interactive chip variant or gesture handler is attached to the chip subtree. `dart analyze` reports no `error`-level diagnostics.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| provider → widget | Read-only render of already-fetched subcategory + preference data; no new user input crosses into this screen |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-gib-01 | Information disclosure | Chip renders subcategory names/icons | accept | Data is the user's own already-fetched category config; no new network call, no cross-user exposure, no privacy field surfaced |
| T-gib-02 | Tampering | Read-only chips | mitigate | Chips are non-interactive (bare `Chip`, no gesture handlers) — no state mutation path is added; toggling remains confined to the subcategory screen |
| T-gib-SC | Tampering | npm/pip/cargo installs | mitigate | No package installs in this task — all widgets/providers/helpers already exist in the codebase; N/A |
</threat_model>

<verification>
Run from `app/`:

1. **No new analyzer errors** (info-level pre-existing lints acceptable):
   `dart analyze lib/routes/settings_categories_screen.dart` reports 0 `error`-level and 0 new `warning`-level diagnostics.

2. **No interaction handlers on the new chip code** (grep-gate for absence, comment-stripped):
   `grep -v '^\s*//' lib/routes/settings_categories_screen.dart | grep -E 'ActionChip|ChoiceChip|InputChip|FilterChip'` returns nothing (the category body's pre-existing `InkWell`/`onTap` for navigation is expected and unrelated — it is NOT part of the chip subtree). Confirm the new `Chip(...)` construction has no adjacent `onTap`/`onPressed`/`onDeleted`/`InkWell`/`GestureDetector`.

3. **Subcategory screen untouched** (scope is category screen only):
   `git diff --name-only` does NOT list `app/lib/routes/settings_subcategories_screen.dart`.

4. **hasValue-based combine preserved** (no regression):
   `grep -c 'hasValue' lib/routes/settings_categories_screen.dart` is ≥ 3 (three providers now guarded), AND no new `isLoading ||` combine condition is present:
   `grep -v '^\s*//' lib/routes/settings_categories_screen.dart | grep -E 'isLoading\s*\|\|'` returns nothing.

5. **Dim state uses only the subcategory's own visibility:**
   `grep 'subcategoryVisible(' lib/routes/settings_categories_screen.dart` is present, and it is NOT ANDed with `categoryVisible(` on the chip dim line.
</verification>

<success_criteria>
- Each category row in `SettingsCategoriesScreen` shows a `Wrap` of chips, one per subcategory (when the category has ≥1 subcategory).
- Each chip contains the subcategory's `badge_icon` (via `subcategoryFilterAvatar`, gracefully omitting the badge when absent) and its `displayName(locale)`.
- A chip dims (opacity 0.5) exactly when `subcategoryVisible(s.id, subcategoryPrefs) == false` — independent of parent category visibility.
- No chip is tappable/interactive in either state (bare `Chip`, no gesture handlers).
- `settings_subcategories_screen.dart` is unmodified; subcategory toggling still works only via that screen.
- The combined `AsyncValue` uses the `hasValue`-based combine extended to three fields (no `isLoading` regression).
- `dart analyze lib/routes/settings_categories_screen.dart` reports no error-level diagnostics.
</success_criteria>

<output>
Create `.planning/quick/260702-gib-add-subcategory-chips-under-each-categor/260702-gib-SUMMARY.md` when done.
</output>
