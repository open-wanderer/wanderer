# Quick Task 260726-rn1: Replace heuristic category/travel-profile matching - Research

**Researched:** 2026-07-26
**Domain:** Go/PocketBase migrations + Flutter/Dart models & call sites
**Confidence:** HIGH (all findings verified by direct file reads in this session)

## Summary

The task requires: (1) a new PocketBase migration adding a `settings` JSON field to
`subcategories` (mirroring the existing `categories.settings` migration), (2) seeding
`valhalla_profile` into `settings` on 5 built-in (sub)categories, (3) adding `settings`
+ a typed accessor to the Dart `Category`/`Subcategory` models, (4) replacing the 3
heuristic functions in `valhalla_util.dart` with settings-driven lookups (with
subcategory→parent fallback), and (5) fixing `_onEditRoute` to read the *live* form
selection instead of the stale `trail.expand?.category` relation.

**Primary recommendation:** Follow the exact `AddMarshaledJSONAt` + backfill pattern from
`1763300311_updated_categories.go` for the new migration; extend `defaultSubcategorySeed`
with a `settings map[string]any` field for seeding; reuse `visibleSortedCategories`/
`visibleSortedSubcategories` (not a new visibility check) for "is this subcategory
available" in the fallback chain; and read the category form field via
`_formKey.currentState?.fields['category']?.value` + `CategoryPicker.resolve(...)`
(NOT via `trail.expand?.category`).

## Two corrections to CONTEXT.md's assumptions

These were explicitly checked against current file contents and differ from what
CONTEXT.md states — the planner should use the verified version below.

1. **`costingForCategory` signature.** CONTEXT.md says it was "recently changed
   mid-session from `(String? category)` to `(Category? category)`". As read in this
   session, `app/lib/util/valhalla_util.dart:48` is still
   `String costingForCategory(String? category)` (confirmed also by
   `test/util/valhalla_util_test.dart:50`, which calls it with `category.name`, a
   `String`). Either the change was reverted or never landed. **Plan against the
   current `String?` signature**, or explicitly decide to change it as part of this
   task's own diff — don't assume it already happened.

2. **`_onEditRoute` "existing pattern at trail_create_screen.dart:120".** Line 120 is
   `_formKey.currentState?.fields['location']?.didChange(result.fullLabel)` — a
   *write* to the `location` field, unrelated to category. There is **no existing
   precedent** in the codebase for reading the live category/subcategory selection via
   `.fields['category']?.value`. The actual, closest existing pattern is in `_onSave`
   (`trail_create_screen.dart:329-346`):
   ```dart
   final values = formState.value;       // after formState.saveAndValidate()
   final subcategories = ref.read(subcategoryProvider);
   final categorySelection = CategoryPicker.resolve(
     values['category'] as String?,
     subcategories,
   );
   ```
   `_onEditRoute` should NOT call `saveAndValidate()` (it would trigger validation
   errors on other required fields like `name` before the user is done). Instead use
   the per-field live value, which does not require validation:
   ```dart
   final categoryValue =
       _formKey.currentState?.fields['category']?.value as String?;
   final subcategories = ref.read(subcategoryProvider);
   final selection = CategoryPicker.resolve(categoryValue, subcategories);
   ```
   `CategoryPicker.resolve` (`app/lib/components/trail/category_picker.dart:56-77`) is
   the required parsing step — the field's raw value is a prefixed string
   (`'category:<id>'` / `'subcategory:<id>'`), not a bare id.

3. **CONTEXT's "line 466-485 default-empty-category logic" is not a call site of the
   old heuristics.** That block (`trail_create_screen.dart:466-485`, guarded by
   `_categoryDefaulted`) only calls `visibleSortedCategories` to default a *categoryless*
   trail's category — it never calls `costingForCategory`/`categoryForTravelProfile`.
   It's relevant as the **pattern to mirror** for availability-checking (see Don't
   Hand-Roll below), not as a call site requiring the bijection itself.

## Standard Stack

N/A — this is an internal refactor of existing Go/PocketBase + Flutter/Dart code, no
new external dependencies. Package Legitimacy Audit: **skipped, no packages installed.**

## Backend: Migration Mechanics

**Existing pattern to mirror exactly** — `db/migrations/1763300311_updated_categories.go`:
- `app.FindCollectionByNameOrId("kjxvi8asj2igqwf")` (categories collection id)
- `collection.Fields.AddMarshaledJSONAt(6, []byte(...))` with a JSON field literal:
  `{"hidden": false, "id": "<new-id>", "maxSize": 0, "name": "settings", "presentable": false, "required": false, "system": false, "type": "json"}`
- `app.Save(collection)`
- Backfill: `app.FindAllRecords("categories")` → loop → `record.Set("settings", map[string]any{...})` → `app.Save(record)`
- Down migration: `collection.Fields.RemoveById("<new-id>")` + `app.Save(collection)`

**For `subcategories`** (collection id `pbc_1781100000`, confirmed via
`db/migrations/1781000000_categories_redesign.go:549` — no `settings` field currently
exists. Current fields: `id`, `category` (relation), `name`, `short_name`,
`icon`, `badge_icon`, `translations`, `created`, `updated`. No `settings` field. Use
`AddMarshaledJSONAt(len(collection.Fields), ...)` (append at end, matching how
`1781000000` itself appends new fields) with a fresh id, e.g. `"json_settings_9002"` or
any unique PocketBase-style id string (must not collide with existing field ids in that
collection — none currently use that pattern, so any new distinct id string is safe).

**Migration filename/timestamp convention:** PocketBase's `migratecmd` plugin
(registered in `db/main.go:85`) drives migration files. Confirmed via `ls
db/migrations/*.go | sort`: the most recent file is
`1785000000_create_regions_collection.go` (unix `1785000000` = 2026-07-25 17:20 UTC).
The project's own migration CLI (`go run . migrate create <name>`, standard
PocketBase `migratecmd` command run from `db/`) stamps the current unix timestamp as
the filename prefix automatically — this is the idiomatic way to generate the next
migration file/timestamp rather than hand-picking a number. If run manually instead,
any integer greater than `1785000000` works (current real unix time at research time
was `1785089889` — safe to use `1785089889` or the actual `go run . migrate create`
output). Suggested filename: `1785100000_updated_subcategories.go` (or whatever
`migrate create` emits) — package `migrations`, mirroring the `_updated_<collection>.go`
naming convention already used for `1763300311_updated_categories.go`.

## Backend: Default Seed Data

**`db/util/category_defaults.go`** (154 lines read in full):
- `defaultCategoryNames = []string{"Hiking", "Walking", "Running", "Climbing", "Skiing", "Canoeing", "Biking", "Other"}`
- `SeedDefaultCategories(app)`: creates missing-by-name categories, conditionally sets
  `settings` via `defaultCategorySettings()` **only if** `collection.Fields.GetByName("settings") != nil` (defensive — safe if field doesn't exist yet).
- `defaultCategorySettings() map[string]any { return map[string]any{"wp_merge_enabled": true, "wp_merge_radius": 50} }`
  — **add the new key here**: `"valhalla_profile": "hiking"` would apply to ALL default
  categories (wrong — only Hiking should get it). Must special-case: either add a
  per-category settings builder (e.g. `defaultCategorySettingsFor(name string)`), or
  set `valhalla_profile` conditionally after calling `defaultCategorySettings()` only
  for `name == "Hiking"`.
- Separately, `ensureRunningCategory` (in `1781000000_categories_redesign.go:348-377`)
  independently constructs `settings` inline (`wp_merge_enabled`/`wp_merge_radius`) — a
  second call site that would need the same `valhalla_profile` key IF Hiking could ever
  reach this path (it can't — this function only creates "Running"), so **no change
  needed there**, but flag it as a second inline `settings`-builder pattern that exists
  in the codebase (don't miss it if searching for "wp_merge_enabled").

**`db/util/subcategory_defaults.go`** (285 lines read in full):
- `type defaultSubcategorySeed struct { parentCategory, name, shortName, badgeIcon string; translations map[string]CategoryTranslation; aliases []string }`
  — **no `settings`-like field exists yet.** Add `settings map[string]any` to this
  struct.
- Confirmed default Biking subcategories (lines 19-35):
  `MTB`, `Gravel`, `Touring`, `Road`, `E-Bike` — exactly the 4 named in CONTEXT plus
  unmapped `E-Bike`, confirming CONTEXT's claim.
- `SeedDefaultSubcategories(app)`: for each seed, finds-or-creates the subcategory
  record. **Two code paths to update**:
  1. New-record path (lines 175-187): add
     `if len(seed.settings) > 0 { record.Set("settings", seed.settings) }` alongside
     the existing `badge_icon`/`translations` conditional sets.
  2. Existing-record path via `applyDefaultSubcategorySeed` (lines 193-225): this
     function only backfills *empty* fields (`short_name`, `badge_icon`,
     translations-merge) and renames alias matches — it does **not** currently touch
     `settings`. Mirror its "only fill if empty" idiom: add a check like
     `if len(record.Get("settings")...) == 0 && len(seed.settings) > 0 { record.Set(...); changed = true }`
     so re-running the seed on an operator's already-customized subcategory doesn't
     clobber a manually-configured `valhalla_profile`.

**Minimal idiomatic addition**, in seed data:
```go
// category_defaults.go
func defaultCategorySettings() map[string]any {
    return map[string]any{"wp_merge_enabled": true, "wp_merge_radius": 50}
}
// new: only Hiking gets valhalla_profile — set post-hoc in SeedDefaultCategories
// or via a small per-name override map.

// subcategory_defaults.go
{parentCategory: "Biking", name: "Touring", shortName: "TOUR", ..., settings: map[string]any{"valhalla_profile": "bikingHybrid"}},
{parentCategory: "Biking", name: "MTB", ..., settings: map[string]any{"valhalla_profile": "bikingMountain"}},
{parentCategory: "Biking", name: "Gravel", ..., settings: map[string]any{"valhalla_profile": "bikingCross"}},
{parentCategory: "Biking", name: "Road", ..., settings: map[string]any{"valhalla_profile": "bikingRoad"}},
```
Vocabulary matches `RouteTravelBucket` enum values exactly (see below) — use the
lowerCamelCase enum names (`hiking`, `bikingHybrid`, `bikingMountain`, `bikingCross`,
`bikingRoad`) as the string values so Dart-side parsing can do a direct
`RouteTravelBucket.values.byName(...)`-style lookup (wrap in try/catch — `byName`
throws on no match).

## Backend: Existing Test Patterns to Mirror

`db/util/category_test.go` (grepped, not fully read — 1100+ lines):
- `TestSeedDefaultCategoriesAddsMissingDefaults` (line 120): uses
  `setupCategoryValidationTestApp(t)`, `createTestCategory(t, app, "Running")`, calls
  `SeedDefaultCategories(app)`, asserts via `app.FindAllRecords("categories")` +
  `byName` map. **This is the pattern to mirror** for a new
  `TestSeedDefaultCategoriesSetsValhallaProfile`-style test.
- `TestSeedDefaultSubcategoriesSkipsNormalizedExisting` (line 490),
  `TestSeedDefaultSubcategoriesRenamesLegacyDefault` (674),
  `TestSeedDefaultSubcategoriesDoesNotOverwriteBadgeIcon` (714),
  `TestSeedDefaultSubcategoriesDoesNotRenameAliasWhenCanonicalExists` (738) — these are
  the direct precedent set for a new
  `TestSeedDefaultSubcategoriesDoesNotOverwriteValhallaProfile`-style test, mirroring
  the "does not overwrite badge_icon" test's structure exactly (call `SeedDefaultSubcategories`
  twice / with a pre-set value, assert it's preserved).
- **No existing Go test currently asserts on the `settings` JSON key at all** — grep
  for `"settings"` in `category_test.go` returned zero matches. This is new test
  surface, not a rewrite of existing coverage.

## Flutter: Category/Subcategory Models

**`app/lib/models/category.dart`** (44 lines, read in full):
```dart
@freezed
abstract class Category with _$Category {
  @JsonSerializable(explicitToJson: true)
  const factory Category({
    required String id,
    required String name,
    @JsonKey(name: 'short_name') String? shortName,
    String? icon,
    Map<String, CategoryTranslation>? translations,
  }) = _Category;
  factory Category.fromJson(...) => _$CategoryFromJson(json);
}
```
No `settings` field currently. Add `Map<String, dynamic>? settings,` (untyped JSON
passthrough, matching how `translations` is a typed map but `settings` has no fixed
shape server-side) plus a `CategoryDisplay`-style extension (file already has one, line
39-44) adding:
```dart
extension CategoryValhallaProfile on Category {
  RouteTravelBucket? get valhallaProfile =>
      RouteTravelBucket.values.firstWhereOrNull(
        (b) => b.name == settings?['valhalla_profile'],
      );
}
```
(`firstWhereOrNull` from `package:collection`, already a project dependency used
throughout `valhalla_util.dart`.)

**`app/lib/models/subcategory.dart`** (37 lines, read in full) — same shape, same gap:
```dart
const factory Subcategory({
  required String id,
  required String category,
  required String name,
  @JsonKey(name: 'short_name') String? shortName,
  String? icon,
  @JsonKey(name: 'badge_icon') String? badgeIcon,
  Map<String, CategoryTranslation>? translations,
}) = _Subcategory;
```
Add `Map<String, dynamic>? settings,` + the equivalent `valhallaProfile` extension
getter, identical pattern.

**Regeneration requirement — confirmed, not assumed:** `category.g.dart`,
`category.freezed.dart`, `subcategory.g.dart`, `subcategory.freezed.dart` are **checked
into git** (`git check-ignore` on all four returned exit 1 / no match; `git ls-files`
confirmed all four are tracked). `build_runner: ^2.13.1` is a `dev_dependencies` entry
in `app/pubspec.yaml:66`. **No Makefile target, no CI workflow, and no documented
script invokes `build_runner`** (grepped `Makefile` and `.github/workflows/*.yml` — no
matches). This means: after editing `category.dart`/`subcategory.dart`, the plan MUST
include an explicit task to run
`dart run build_runner build --delete-conflicting-outputs` from `app/` and commit the
4 regenerated files — there is no automation that does this.

## Flutter: RouteTravelBucket — Canonical Vocabulary

`app/lib/util/route_travel_bucket.dart` (227 lines, read in full):
```dart
enum RouteTravelBucket { hiking, bikingHybrid, bikingMountain, bikingCross, bikingRoad }
```
Each variant carries `.costing` (`'pedestrian'`/`'bicycle'`), `.costingOptions`
(fixed map incl. `bicycle_type`), `.keywords` (old heuristic, to retire),
`.fallbackIcon`, `.badgeIcon`. **Use `RouteTravelBucket.name` (enum's built-in
`.name` getter, e.g. `'bikingMountain'`) as the exact string stored in
`settings.valhalla_profile`** — do not invent a separate string constant table.

`bucketIcon(bucket, categories, {size})` (lines 166-199) currently calls
`categoryForTravelProfile('pedestrian', categories)` (hiking) and
`categoryForBikeBucket(bucket.keywords, categories)` (bike buckets) internally — this
function's **internal implementation** must be rewritten to look up by
`valhallaProfile` (a **direct call site**, distinct from its own external callers).
Its signature likely needs a `List<Subcategory> subcategories` parameter added, since
bike-bucket matches are now subcategory-level, not category-level. External callers
(`settings_tab.dart:57`, `travel_profile_sheet.dart:40`) currently call
`bucketIcon(bucket, categories)` — both need updating to pass subcategories too, but
neither calls the 3 old heuristic functions *directly* (they're indirect via
`bucketIcon`).

## Flutter: Complete Call-Site Inventory

Grepped the entire `app/lib/` tree for `categoryForTravelProfile`, `costingForCategory`,
`categoryForBikeBucket`, `RouteTravelBucket` (excluding generated `.g.dart`/
`.freezed.dart`). Full list:

| File:Line | Call | Notes |
|---|---|---|
| `app/lib/util/valhalla_util.dart:48,68,96` | function definitions | to be replaced/retired |
| `app/lib/util/route_travel_bucket.dart:173,178` | `categoryForTravelProfile`/`categoryForBikeBucket` inside `bucketIcon` | direct call site — rewrite internals |
| `app/lib/util/navigation_launch_util.dart:160` | `costingForCategory(trail.expand?.category?.name)` | direct call site |
| `app/lib/util/route_planner_handoff_util.dart:263` | `categoryForTravelProfile(travelProfile, categories)` in `finishPlanning` | direct call site; comments at 112, 250 reference it too (doc only) |
| `app/lib/services/trail_download_service.dart:139` | `costingForCategory(trail.expand?.category?.name)` | direct call site |
| `app/lib/routes/navigation_screen.dart:770` | `costingForCategory(originalTrail?.expand?.category?.name)` | direct call site; falls back from `_recordingCosting` |
| `app/lib/routes/navigation_screen.dart:813` | `categoryForTravelProfile(_recordingCosting, ref.read(categoryProvider).value ?? const [])` | direct call site — derives category (not subcategory) for a trail-less recording; note: bijection maps some buckets to *subcategories*, so this call site's result type may need to become a `(category, subcategory)` pair, not a bare category id |
| `app/lib/routes/trail_create_screen.dart:303` | `costingForCategory(trail.expand?.category?.name)` in `_onEditRoute` | direct call site — **the root-cause-2 fix target**; must switch to live form selection (see Corrections section above) |
| `app/lib/components/route_planner/settings_tab.dart:54,57,82` | `RouteTravelBucket.values`, `bucketIcon(bucket, categories)` | indirect (via `bucketIcon`) |
| `app/lib/components/route_planner/travel_profile_sheet.dart:13,14,38,40` | `RouteTravelBucket` picker, `bucketIcon(bucket, categories)` | indirect (via `bucketIcon`) |
| `app/lib/routes/trail_source_select_screen.dart:47-50` | comment only, no call | references `costingForCategory` in a doc comment explaining `NavigationScreen.recordingCosting`; no code change needed here itself |

**Also checked and confirmed NOT a call site**: `trail_create_screen.dart:466-485`
(the `_categoryDefaulted` block) — see Corrections section above.

## Flutter: `visibleSortedCategories` / Availability Logic

`app/lib/util/category_preference_sort.dart` (118 lines, read in full).

Relevant signatures:
```dart
List<Category> visibleSortedCategories(
  List<Category> categories,
  List<CategoryPreference> prefs,
  Locale? locale, {
  String? keepVisibleId,
}) // sorted + filtered to visible

List<Subcategory> visibleSortedSubcategories(
  List<Subcategory> subs,
  List<SubcategoryPreference> prefs,
  Locale? locale, {
  String? keepVisibleId,
}) // sorted + filtered to visible

bool categoryVisible(String categoryId, List<CategoryPreference> prefs)
bool subcategoryVisible(String subcategoryId, List<SubcategoryPreference> prefs)
```
Visibility rule (both): visible unless a preference row exists with `visible == false`;
no-row or `visible == null` both mean visible.

**For the fallback-to-parent check** ("subcategory unavailable" = doesn't exist in the
loaded list OR hidden by preference), the correct reusable primitives are:
1. Existence: `subcategories.any((s) => s.id == targetId)` against the operator's
   currently-loaded `ref.read(subcategoryProvider)` list.
2. Visibility: `subcategoryVisible(targetId, subcategoryPrefs)` directly — no need to
   call the full `visibleSortedSubcategories` (that also sorts, which is unneeded
   overhead for a single-id availability check). Use `subcategoryVisible` alone unless
   the plan also wants the sorted list for another reason.

This matches (and should reuse, not reimplement) the same visibility semantics already
driving `CategoryPicker`'s dropdown (`category_picker.dart:92-114`, which calls
`visibleSortedCategories`/`visibleSortedSubcategories` with `keepVisibleId`).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---|---|---|---|
| "Is this subcategory available to the user" | A new visibility predicate | `subcategoryVisible()` / `categoryVisible()` (`category_preference_sort.dart`) | Already the single source of truth used by `CategoryPicker`; a second implementation would drift |
| Bucket↔string vocabulary | A new string-constant table | `RouteTravelBucket.name` (Dart enum's built-in `.name`) + `RouteTravelBucket.values.byName(...)`/`firstWhereOrNull` | Enum is already the canonical source per CONTEXT; avoids a second mapping table to keep in sync |
| Reading live form category selection | A new form-state helper | `_formKey.currentState?.fields['category']?.value` (raw prefixed string) → `CategoryPicker.resolve(value, subcategories)` | Existing parsing logic (`category_picker.dart:56-77`) already handles the `category:`/`subcategory:` prefix format; reimplementing this parsing is exactly the kind of drift CONTEXT is trying to eliminate |

## Common Pitfalls

### Pitfall 1: Seeding `valhalla_profile` onto ALL default categories via `defaultCategorySettings()`
**What goes wrong:** `defaultCategorySettings()` is a single shared function called for
every default category name in `SeedDefaultCategories`. Naively adding
`"valhalla_profile": "hiking"` there would tag Walking, Running, Climbing, Skiing,
Canoeing, Biking, Other all as `hiking` too.
**How to avoid:** Special-case by category name — only set the key when
`name == "Hiking"` (or introduce a small `map[string]string{"Hiking": "hiking"}`
override table, applied after the shared defaults).

### Pitfall 2: `applyDefaultSubcategorySeed` silently not touching `settings` on already-existing subcategories
**What goes wrong:** The existing-record backfill path only updates fields it
explicitly checks (`short_name`, `badge_icon`, translations). If the new `settings`
write isn't added there, an operator's pre-existing "MTB"/"Gravel"/"Touring"/"Road"
subcategories (seeded before this migration) will silently never receive
`valhalla_profile`, and the feature won't work retroactively.
**Warning signs:** New installs work; existing installs upgrading through this
migration don't get the bijection populated.

### Pitfall 3: `costingForCategory` current signature is `String?`, not `Category?`
**What goes wrong:** Assuming CONTEXT.md's claim that the signature was already
changed leads to writing code against a `Category?` parameter that doesn't compile.
**How to avoid:** Verify current signature before editing (`String? category` as of
this research) — see Corrections section.

### Pitfall 4: Calling `formState.saveAndValidate()` inside `_onEditRoute`
**What goes wrong:** `_onEditRoute` fires when the user taps "edit route" mid-form-fill;
calling `saveAndValidate()` would surface validation errors (e.g. missing required
`name`) the user hasn't gotten to yet, blocking navigation to the route planner.
**How to avoid:** Read the field's live value via `_formKey.currentState?.fields['category']?.value`
(no validation triggered), not via `formState.value` (which requires `save()`/`saveAndValidate()` first).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|---|---|---|
| A1 | New migration filename should be `1785100000_updated_subcategories.go` or the output of `go run . migrate create <name>` | Migration Mechanics | Low — any timestamp > 1785000000 works; exact number is cosmetic |
| A2 | `RouteTravelBucket.name` (enum `.name` getter) should be the stored string vocabulary, not a hand-written constant | RouteTravelBucket vocabulary | Low-Medium — if planner chooses a different string scheme, Dart/Go sides must still agree; `.name` is the simplest single source of truth |
| A3 | New subcategories `settings` field id can be any unused PocketBase field-id string (no fixed naming rule found beyond "must not collide") | Migration Mechanics | Low |

**If this table is empty:** N/A — see above, all three are low-risk implementation
details the planner can decide freely within the constraints already locked in
CONTEXT.md.

## Sources

### Primary (HIGH confidence — direct file reads this session)
- `db/migrations/1763300311_updated_categories.go` — full file
- `db/migrations/1781000000_categories_redesign.go` — full file (804 lines)
- `db/util/category_defaults.go` — full file (290 lines)
- `db/util/subcategory_defaults.go` — full file (285 lines)
- `db/util/category_test.go` — grepped + targeted reads (lines 1-40, 115-180)
- `app/lib/models/category.dart` — full file
- `app/lib/models/subcategory.dart` — full file
- `app/lib/util/valhalla_util.dart` — full file
- `app/lib/util/route_travel_bucket.dart` — full file (227 lines)
- `app/lib/util/category_preference_sort.dart` — full file
- `app/lib/util/route_planner_handoff_util.dart` — full file (400 lines)
- `app/lib/routes/trail_create_screen.dart` — targeted reads (90-130, 280-330, 455-500, 760-830)
- `app/lib/routes/navigation_screen.dart` — targeted read (750-840)
- `app/lib/services/trail_download_service.dart` — targeted read (120-150)
- `app/lib/util/navigation_launch_util.dart` — targeted read (140-170)
- `app/lib/components/trail/category_picker.dart` — full file
- `app/lib/routes/trail_source_select_screen.dart` — targeted read (35-55)
- `app/lib/models/trail.dart` — grep for category/subcategory fields
- `app/test/util/valhalla_util_test.dart` — full file
- `docs/src/content/docs/run/backend-configuration/custom-categories.md` — full file
- `app/pubspec.yaml` — grep for `build_runner`
- `db/main.go` — grep for `migratecmd`
- `Makefile`, `.github/workflows/*.yml` — grepped for `build_runner` (no matches)
- `git ls-files` / `git check-ignore` on the 4 generated Dart files — confirmed tracked

### Secondary/Tertiary
None — no web search was needed; this is a pure internal-codebase research task.

## Metadata

**Confidence breakdown:**
- Backend migration mechanics: HIGH — verified against 2 real migration files in this repo
- Default seed data shape: HIGH — full files read, exact struct/function signatures confirmed
- Dart model gaps: HIGH — full files read, confirmed no `settings` field exists yet
- Call-site inventory: HIGH — exhaustive grep across `app/lib/`, cross-checked against file reads
- CONTEXT.md corrections: HIGH — directly contradicted by current file contents, cited with line numbers

**Research date:** 2026-07-26
**Valid until:** Short-lived — this is a snapshot of an actively-changing branch
(`feature/app`); re-verify signatures immediately before executing if any other
commits land on this branch in the meantime.
