# Phase 11: Trail Filter Subcategory Support - Pattern Map

**Mapped:** 2026-06-29
**Files analyzed:** 7
**Analogs found:** 7 / 7

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `app/lib/models/trail.dart` | model | transform | self (existing `TrailFilter` + `toFilterText()`) | self-modification |
| `app/lib/provider/trail/trail_filter_provider.dart` | provider | request-response | self (existing `defaultFilter` constructor) | self-modification |
| `app/lib/components/base/wanderer_filter_chip.dart` | component | event-driven | self (existing generic chip) | self-modification |
| `app/lib/routes/trail_filter_screen.dart` | route/screen | request-response | self (existing Category section, lines 73–88) | self-modification |
| `app/lib/components/trail/trail_quick_filter_bar.dart` | component | event-driven | self (`_showCategorySheet`, lines 230–298) | self-modification |
| `app/lib/util/icon_util.dart` | utility | transform | `app/lib/util/icon_util.dart` (read-only) | exact |
| `app/lib/i18n/app_en.arb` (+ other locale ARBs) | config | — | existing ARB key `categories` | role-match |

## Pattern Assignments

---

### `app/lib/models/trail.dart` — add `subcategory` field + rewrite category/subcategory clause

**Analog:** self — existing `TrailFilter` freezed model (lines 159–189) and `toFilterText()` (lines 197–289)

**Existing `TrailFilter` factory** (lines 159–187):
```dart
@freezed
abstract class TrailFilter with _$TrailFilter {
  const factory TrailFilter({
    required String q,
    required List<Category> category,
    required List<Tag> tags,
    required List<int> difficulty,
    // ... other fields ...
    required TrailFilterSort sort,
    required SortOrder sortOrder,
  }) = _TrailFilter;
```
Add `@Default(<Subcategory>[]) List<Subcategory> subcategory` (or `required List<Subcategory> subcategory`) alongside `required List<Category> category`. Using `@Default(<Subcategory>[])` avoids touching other constructors.

**Existing stale category clause** (lines 274–278) — REPLACE entirely:
```dart
// CURRENT (stale — category name is no longer a filterable attribute in Meilisearch)
if (category.isNotEmpty) {
  final catList = category.map((c) => "'${c.name}'").join(", ");
  parts.add('category IN [$catList]');
}
```

**Replacement pattern** (from RESEARCH.md Pattern 1, mirrors `web/src/lib/stores/trail_store.ts:862-908`):
```dart
// REPLACEMENT — combined OR group using IDs
if (category.isNotEmpty || subcategory.isNotEmpty) {
  final List<String> categoryParts = [];
  if (category.isNotEmpty) {
    final ids = category.map((c) => "'${c.id}'").join(", ");
    categoryParts.add('category_id IN [$ids]');
  }
  if (subcategory.isNotEmpty) {
    final ids = subcategory.map((s) => "'${s.id}'").join(", ");
    categoryParts.add('subcategory_id IN [$ids]');
  }
  if (categoryParts.isNotEmpty) {
    parts.add('(${categoryParts.join(" OR ")})');
  }
}
```

**Required import addition:**
```dart
import 'package:wanderer/models/subcategory.dart';
```

---

### `app/lib/provider/trail/trail_filter_provider.dart` — add `subcategory: []` to defaultFilter

**Analog:** self — existing `defaultFilter` constructor (lines 26–48)

**Existing defaultFilter constructor** (lines 26–48):
```dart
defaultFilter = TrailFilter(
  q: "",
  category: [],
  tags: [],
  difficulty: [0, 1, 2],
  author: null,
  public: true,
  shared: true,
  liked: false,
  private: true,
  near: TrailNear(radius: 2000),
  distanceMin: 0,
  distanceMax: filterValues.maxDistance,
  // ... etc
  sort: TrailFilterSort.created,
  sortOrder: SortOrder.desc,
);
```
Add `subcategory: [],` alongside `category: []`. Only needed if the field is declared `required` (not `@Default`).

**`updateFilter` pattern** (lines 60–65) — copy for subcategory chip `onChanged`:
```dart
void updateFilter(TrailFilter Function(TrailFilter current) updater) {
  final currentState = state.value;
  if (currentState == null) return;
  state = AsyncData(updater(currentState));
}
```
Call site pattern for subcategory toggle:
```dart
ref.read(trailFilterProvider(widget.filterId).notifier)
    .updateFilter((f) => f.copyWith(subcategory: selected));
```

---

### `app/lib/components/base/wanderer_filter_chip.dart` — add `avatarBuilder` parameter

**Analog:** self — entire file (65 lines)

**Full current implementation** (lines 1–65):
```dart
class WandererFilterChip<T> extends StatelessWidget {
  final List<T> options;
  final List<T> selectedValues;
  final String Function(T) labelBuilder;
  final Function(List<T>) onChanged;
  final bool multiple;

  const WandererFilterChip({
    super.key,
    required this.options,
    required this.selectedValues,
    required this.labelBuilder,
    required this.onChanged,
    this.multiple = false,
  });
```

**Add `avatarBuilder` field and wire to `FilterChip.avatar`:**
```dart
// ADD field:
final Widget? Function(T item)? avatarBuilder;

// ADD to constructor:
this.avatarBuilder,

// ADD to FilterChip inside build():
avatar: avatarBuilder?.call(option),
```
All existing call sites omit `avatarBuilder` → `null` by default, no change needed.

---

### `app/lib/routes/trail_filter_screen.dart` — add Subcategories section

**Analog:** self — existing Categories section (lines 73–88) as the layout template

**Imports to add** (copy from existing imports block, lines 1–19):
```dart
import 'package:wanderer/models/subcategory.dart';
import 'package:wanderer/provider/trail/subcategory_provider.dart';
import 'package:wanderer/provider/category_preference_provider.dart';
import 'package:wanderer/provider/subcategory_preference_provider.dart';
import 'package:wanderer/util/icon_util.dart';
```

**Existing Category section pattern** (lines 73–88) — copy this structure for Subcategories:
```dart
Text(l10n.categories, style: TextTheme.of(context).labelLarge),
const SizedBox(height: 8),
WandererFilterChip<Category>(
  options: categories.value ?? [],
  selectedValues: f.category,
  labelBuilder: (c) => c.name,               // ← change to displayName(locale)
  onChanged: (categories) {
    ref.read(trailFilterProvider(widget.filterId).notifier)
        .updateFilter((filter) => filter.copyWith(category: categories));
  },
),
const SizedBox(height: 16),
```

**FILTER-04 fix — change `labelBuilder` to use locale name (line 79):**
```dart
// FROM:
labelBuilder: (c) => c.name,
// TO:
labelBuilder: (c) => c.displayName(Localizations.localeOf(context)),
```

**New Subcategories section with AnimatedSize** (insert after Category `SizedBox(height: 16)`, before Tags):
```dart
// Visibility filtering (FILTER-06/-07) — mirror categoryVisibleInDesign
final catPrefs = ref.watch(categoryPreferenceProvider).value ?? [];
final visibleCategories = (categories.value ?? [])
    .where((c) => catPrefs.firstWhereOrNull((p) => p.category == c.id)?.visible != false)
    .toList();

// For subcategories section:
final subcategories = ref.watch(subcategoryProvider);  // List<Subcategory> — synchronous
final subPrefs = ref.watch(subcategoryPreferenceProvider).value ?? [];
final selectedCategoryIds = f.category.map((c) => c.id).toSet();
final visibleSubs = subcategories
    .where((s) => selectedCategoryIds.contains(s.category))
    .where((s) => subPrefs.firstWhereOrNull((p) => p.subcategory == s.id)?.visible != false)
    .toList();

AnimatedSize(
  duration: const Duration(milliseconds: 200),
  alignment: Alignment.topCenter,
  child: f.category.isEmpty || visibleSubs.isEmpty
      ? const SizedBox.shrink()
      : Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Text(l10n.subcategories, style: TextTheme.of(context).labelLarge),
          const SizedBox(height: 8),
          WandererFilterChip<Subcategory>(
            options: visibleSubs,
            selectedValues: f.subcategory,
            multiple: true,
            labelBuilder: (s) => s.displayName(Localizations.localeOf(context)),
            avatarBuilder: (s) => _subcategoryAvatar(
              s,
              f.category.firstWhereOrNull((c) => c.id == s.category),
            ),
            onChanged: (sel) => ref
                .read(trailFilterProvider(widget.filterId).notifier)
                .updateFilter((flt) => flt.copyWith(subcategory: sel)),
          ),
          const SizedBox(height: 16),
        ]),
),
```

**Avatar helper methods** (add as private methods on `_TrailFilterScreenState`):
```dart
Widget _categoryAvatar(Category c) {
  final raw = (c.icon ?? '').trim();
  final key = raw.startsWith('fa-') ? raw.substring(3) : raw;
  final faData = fontAwesomeIconsMap[key];
  return faData != null
      ? FaIcon(faData, size: 16)
      : const Icon(Icons.category, size: 16);
}

Widget _subcategoryAvatar(Subcategory s, Category? parent) {
  final primaryRaw = ((s.icon?.trim().isNotEmpty ?? false)
      ? s.icon!
      : (parent?.icon ?? '')).trim();
  final primaryKey = primaryRaw.startsWith('fa-') ? primaryRaw.substring(3) : primaryRaw;
  final primary = fontAwesomeIconsMap[primaryKey];

  final badgeRaw = (s.badgeIcon ?? '').trim();
  final badgeKey = badgeRaw.startsWith('fa-') ? badgeRaw.substring(3) : badgeRaw;
  final badge = badgeKey.isEmpty ? null : fontAwesomeIconsMap[badgeKey];

  return Stack(clipBehavior: Clip.none, children: [
    primary != null ? FaIcon(primary, size: 16) : const Icon(Icons.category, size: 16),
    if (badge != null)
      Positioned(right: -2, bottom: -2, child: FaIcon(badge, size: 10)),
  ]);
}
```

---

### `app/lib/components/trail/trail_quick_filter_bar.dart` — extend category sheet + fix `_isCategoryActive`

**Analog:** self — `_showCategorySheet` (lines 230–298) and `_isCategoryActive` (lines 97–99)

**`_isCategoryActive` update** (line 98) — D-13:
```dart
// FROM:
bool _isCategoryActive(TrailFilter filter) {
  return filter.category.isNotEmpty;
}
// TO:
bool _isCategoryActive(TrailFilter filter) {
  return filter.category.isNotEmpty || filter.subcategory.isNotEmpty;
}
```

**Imports to add** (alongside existing imports, lines 1–16):
```dart
import 'package:wanderer/models/subcategory.dart';
import 'package:wanderer/provider/trail/subcategory_provider.dart';
import 'package:wanderer/provider/category_preference_provider.dart';
import 'package:wanderer/provider/subcategory_preference_provider.dart';
import 'package:wanderer/util/icon_util.dart';
```

**Existing `_showCategorySheet` WandererFilterChip call** (lines 276–287) — extend the sheet Column to add AnimatedSize below it:
```dart
// After existing category WandererFilterChip in the Consumer builder:
// ADD inside the Column's children list, after the category chip:

// Subcategory section (D-11/D-12) — same AnimatedSize pattern as TrailFilterScreen
Builder(builder: (context) {
  final subcategories = ref.watch(subcategoryProvider);
  final subPrefs = ref.watch(subcategoryPreferenceProvider).value ?? [];
  final selectedCategoryIds = currentFilter.category.map((c) => c.id).toSet();
  final visibleSubs = subcategories
      .where((s) => selectedCategoryIds.contains(s.category))
      .where((s) => subPrefs.firstWhereOrNull((p) => p.subcategory == s.id)?.visible != false)
      .toList();

  return AnimatedSize(
    duration: const Duration(milliseconds: 200),
    alignment: Alignment.topCenter,
    child: currentFilter.category.isEmpty || visibleSubs.isEmpty
        ? const SizedBox.shrink()
        : Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            const SizedBox(height: 12),
            Text(l10n.subcategories, style: Theme.of(context).textTheme.labelLarge),
            const SizedBox(height: 8),
            WandererFilterChip<Subcategory>(
              options: visibleSubs,
              selectedValues: currentFilter.subcategory,
              multiple: true,
              labelBuilder: (s) => s.displayName(Localizations.localeOf(context)),
              avatarBuilder: (s) => _subcategoryAvatar(
                s,
                currentFilter.category.firstWhereOrNull((c) => c.id == s.category),
              ),
              onChanged: (sel) => ref
                  .read(trailFilterProvider(filterId).notifier)
                  .updateFilter((f) => f.copyWith(subcategory: sel)),
            ),
          ]),
  );
}),
```

Avatar helpers are needed here too — either extract to a shared utility function or duplicate from `trail_filter_screen.dart`.

**Existing `_showCategorySheet` category `labelBuilder` fix** (line 279) — D-10:
```dart
// FROM:
labelBuilder: (c) => c.name,
// TO:
labelBuilder: (c) => c.displayName(Localizations.localeOf(context)),
```

---

### `app/lib/util/icon_util.dart` — read-only reference

**No modifications.** The `fontAwesomeIconsMap` (line 1018) is:
```dart
const Map<String, FaIconData> fontAwesomeIconsMap = <String, FaIconData>{
  "zero": FontAwesomeIcons.zero,
  // ... 2000+ entries, keys have no 'fa-' prefix
};
```
**Usage pattern:** strip `fa-` from `category.icon` / `subcategory.icon` before lookup:
```dart
final key = raw.startsWith('fa-') ? raw.substring(3) : raw;
final faData = fontAwesomeIconsMap[key]; // null if not found → fall back to Icons.category
```

---

### `app/lib/i18n/app_en.arb` (+ other locale ARBs) — add `subcategories` key

**Analog:** existing `categories` key pattern

**Key to add** (mirror the existing `categories` entry style):
```json
"subcategories": "Subcategories",
"@subcategories": {
  "description": "Label for the subcategories filter section"
}
```
Add the same key to all ~13 locale ARB files (`app_de.arb`, `app_fr.arb`, etc.), using the English string as a fallback for untranslated locales. Run `flutter gen-l10n` after editing ARB files.

---

## Shared Patterns

### Provider consumption — `AsyncValue` vs. synchronous list
**Source:** `app/lib/provider/trail/subcategory_provider.dart` and `app/lib/provider/category_preference_provider.dart`
**Apply to:** `trail_filter_screen.dart` and `trail_quick_filter_bar.dart`

```dart
// subcategoryProvider — synchronous List<Subcategory>, NO .value call
final subcategories = ref.watch(subcategoryProvider);  // List<Subcategory> directly

// categoryPreferenceProvider — AsyncValue<List<...>>, use .value for nullable access
final catPrefs = ref.watch(categoryPreferenceProvider).value ?? [];

// subcategoryPreferenceProvider — same shape as categoryPreferenceProvider
final subPrefs = ref.watch(subcategoryPreferenceProvider).value ?? [];
```

### `updateFilter` call pattern
**Source:** `app/lib/provider/trail/trail_filter_provider.dart` lines 60–65
**Apply to:** every chip `onChanged` callback in both filter surfaces
```dart
ref.read(trailFilterProvider(filterId).notifier)
    .updateFilter((f) => f.copyWith(subcategory: selectedList));
```

### freezed field with default
**Source:** existing `TrailFilter` field pattern in `app/lib/models/trail.dart`
**Apply to:** new `subcategory` field declaration
```dart
// Preferred: use @Default to avoid touching all constructors
@Default(<Subcategory>[]) List<Subcategory> subcategory,
```

### FA icon lookup with `fa-` strip and Material fallback
**Source:** `app/lib/util/icon_util.dart` line 1018 + RESEARCH.md D-08/D-09
**Apply to:** `_categoryAvatar()` and `_subcategoryAvatar()` helper methods
```dart
final key = raw.startsWith('fa-') ? raw.substring(3) : raw;
final faData = fontAwesomeIconsMap[key];
// fall back:
faData != null ? FaIcon(faData, size: 16) : const Icon(Icons.category, size: 16)
```

### `firstWhereOrNull` usage
**Source:** RESEARCH.md Pattern 2 / Pattern 3
**Apply to:** visibility filtering in both screens
```dart
// Requires: import 'package:collection/collection.dart';
// (package:collection is a transitive Flutter dep — already available)
subPrefs.firstWhereOrNull((p) => p.subcategory == s.id)?.visible != false
```

### Locale access in widget context
**Source:** `app/lib/routes/profile_screen.dart:219` (documented in RESEARCH.md)
**Apply to:** every `labelBuilder` that uses `displayName`
```dart
Localizations.localeOf(context)  // pass to displayName(locale)
```

---

## No Analog Found

None — all files are modifications to existing files with existing analogs.

---

## Build Artifacts Required After Changes

| File changed | Command to run |
|---|---|
| `app/lib/models/trail.dart` (freezed field added) | `dart run build_runner build` |
| `app/lib/i18n/app_*.arb` (new key) | `flutter gen-l10n` |

---

## Metadata

**Analog search scope:** `app/lib/models/`, `app/lib/routes/`, `app/lib/components/`, `app/lib/provider/`, `app/lib/util/`
**Files scanned:** 9 source files read directly
**Pattern extraction date:** 2026-06-29
