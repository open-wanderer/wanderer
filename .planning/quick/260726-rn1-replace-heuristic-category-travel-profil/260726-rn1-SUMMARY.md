---
phase: quick-260726-rn1
plan: 01
subsystem: categories/routing
tags: [valhalla, categories, subcategories, routing, pocketbase, flutter]
requires:
  - db/util/category.go (ParseCategoryTranslations pattern)
  - app/lib/util/category_preference_sort.dart (subcategoryVisible)
  - app/lib/components/trail/category_picker.dart (CategoryPicker.resolve)
provides:
  - subcategories.settings JSON field (PocketBase)
  - settings.valhalla_profile open vocabulary (categories + subcategories)
  - ValhallaProfile open value type (Dart)
  - resolveValhallaProfile / costingForCategory / bucketForProfile / categorySelectionForBucket
affects:
  - route planner handoff, navigation costing, offline download costing, icon resolution
tech-stack:
  added: []
  patterns:
    - open-vocabulary settings key with format guard instead of allowlist
    - only-if-absent retroactive backfill (never clobber operator values)
    - derive-not-duplicate (RouteTravelBucket.valhallaProfileKey from costingOptions)
key-files:
  created:
    - db/migrations/1785200000_updated_subcategories.go
    - app/lib/models/valhalla_profile.dart
  modified:
    - db/util/category.go
    - db/util/category_defaults.go
    - db/util/subcategory_defaults.go
    - db/util/category_test.go
    - db/main.go
    - app/lib/models/category.dart
    - app/lib/models/subcategory.dart
    - app/lib/entities/category_entity.dart
    - app/lib/entities/subcategory_entity.dart
    - app/lib/util/valhalla_util.dart
    - app/lib/util/route_travel_bucket.dart
    - app/lib/util/route_planner_handoff_util.dart
    - app/lib/util/navigation_launch_util.dart
    - app/lib/services/trail_download_service.dart
    - app/lib/provider/trail/trail_download_state_provider.dart
    - app/lib/routes/navigation_screen.dart
    - app/lib/routes/trail_create_screen.dart
    - app/lib/components/route_planner/settings_tab.dart
    - app/lib/components/route_planner/travel_profile_sheet.dart
    - app/test/util/valhalla_util_test.dart
    - app/test/util/route_travel_bucket_test.dart
    - app/test/components/route_planner/travel_profile_sheet_test.dart
    - docs/src/content/docs/run/backend-configuration/custom-categories.md
  regenerated:
    - app/lib/models/category.freezed.dart
    - app/lib/models/category.g.dart
    - app/lib/models/subcategory.freezed.dart
    - app/lib/models/subcategory.g.dart
    - app/lib/objectbox.g.dart
    - app/lib/objectbox-model.json
decisions:
  - Migration timestamp moved from 1785100000 to 1785200000 (collision with an existing migration file)
  - bucketIcon/bucketForState rewrite pulled forward from Task 4 into Task 3 (compile-order blocker)
metrics:
  tasks: 6
  completed: 2026-07-28
---

# Quick Task 260726-rn1: Replace heuristic category/travel-profile matching with an explicit Valhalla-costing bijection — Summary

Category→Valhalla-costing resolution is now data-driven via an **open** `settings.valhalla_profile`
key on both `categories` and `subcategories`, replacing English-substring name matching — so an
operator can point a brand-new `Car` category at `auto` with no code change, and translated/custom
category names resolve correctly.

## ⚠️ No commits were made

Per explicit user instruction, **zero commits, zero staging** were performed. Every change is left as
an unstaged working-tree modification (plus 2 untracked new source files). `HEAD` is unchanged at
`4b192622`. This deliberately overrides the plan's per-task "commit atomically" instruction and the
GSD executor's default commit protocol. `.planning/STATE.md` and `.planning/ROADMAP.md` were **not**
updated, also per instruction.

## What was built

### Task 1 — Backend (Go/PocketBase)

- **New migration** `db/migrations/1785200000_updated_subcategories.go`: adds a `settings` JSON field
  (`json_settings_9002`) to the `subcategories` collection (`pbc_1781100000`) at the end of the field
  list, backfills every existing row to `{}`, and removes the field on Down.
- **`db/util/category.go`**: new unexported `parseSettingsMap(raw any) map[string]any` — a total,
  never-erroring, never-nil best-effort decoder mirroring `ParseCategoryTranslations`'s type switch
  (`map[string]any` / `types.JSONRaw` / `[]byte` / `string` / default).
- **`db/util/category_defaults.go`**: `defaultCategorySettings()` now takes a `name` and consults a
  new `defaultCategoryValhallaProfiles = {"Hiking": "pedestrian"}`. New
  `PrepopulateDefaultCategoryValhallaProfiles(app)` retroactively backfills existing installs,
  skipping any record where the `valhalla_profile` key already exists (any value).
- **`db/util/subcategory_defaults.go`**: `defaultSubcategorySeed` gained a `settings` field; MTB /
  Gravel / Touring / Road under Biking seed `bicycle_mountain` / `bicycle_cross` / `bicycle_hybrid` /
  `bicycle_road`; E-Bike stays unmapped. `applyDefaultSubcategorySeed` gained a `hasSettingsField`
  parameter and merges seed settings **only for absent keys**, so a second `SeedDefaultSubcategories`
  run is a true no-op (asserted via an `updated` timestamp comparison in the test).
- **`db/main.go`**: `initCategories` now checks `PrepopulateDefaultCategoryIcons`'s error and returns
  `PrepopulateDefaultCategoryValhallaProfiles(app)`.
- **`db/util/category_test.go`**: added `&core.JSONField{Name: "settings"}` to both test collections,
  extended `TestDefaultSubcategories` with a `wantValhallaProfiles` map (which also asserts that *no
  other* default subcategory declares a profile), and added 5 new test functions covering seed,
  backfill, non-overwrite, and the `{"Hiking": "pedestrian"}` exactness guard.

### Task 2 — Flutter value type, models, ObjectBox parity

- **New `app/lib/models/valhalla_profile.dart`**: plain Dart `ValhallaProfile { costing, bicycleType? }`
  with value equality. `ValhallaProfile.parse` applies a `^[a-z][a-z_]*$` format guard (failures →
  `null` = unset), decomposes only the 4 literal `bicycle_(road|hybrid|cross|mountain)` forms into
  `bicycle` + capitalized `bicycle_type`, and passes **everything else through verbatim** — it never
  splits on `_` generally, so `motor_scooter` survives whole.
- **`route_travel_bucket.dart`**: `valhallaProfileKey` derives its string from `costing`/
  `costingOptions['bicycle_type']` (no hand-duplicated table), plus a `valhallaProfile` getter.
- **`Category`/`Subcategory`**: gained `Map<String, dynamic>? settings` plus a `valhallaProfile`
  extension getter returning `ValhallaProfile?`. `valhalla_profile.dart` is a leaf with no Wanderer
  imports, so the anticipated model↔util import cycle never arises.
- **`CategoryEntity`/`SubcategoryEntity`**: `settingsJson` round-trip so the ObjectBox cache (cold
  start for subcategories, offline fallback for categories) carries `valhalla_profile`.
- **Codegen**: `build_runner` regenerated the 4 freezed/json files; `objectbox-model.json` gained two
  new properties **additively** (`7:…` on CategoryEntity, `9:…` on SubcategoryEntity) with no retired
  UIDs.

### Task 3 — New settings-driven mapping (TDD)

Deleted `categoryForTravelProfile` and `categoryForBikeBucket`; replaced `costingForCategory(String?)`
with a differently-shaped API:

- `resolveValhallaProfile({category, subcategoryId, subcategories, subcategoryPrefs})` — subcategory
  tier (present **and** visible per the reused `subcategoryVisible`) → category tier → bicycle-hybrid
  tier (only when a sibling subcategory of the *same* category resolves to a `bicycle` costing) →
  `null`.
- `costingForCategory({...})` — thin wrapper, `pedestrian` default.
- `bucketForProfile(ValhallaProfile?)` — narrows back to a picker bucket, `null` for `auto` etc.
- `TravelBucketCategorySelection` + `categorySelectionForBucket(bucket, categories, subcategories)` —
  reverse lookup, subcategory-preferred.

25 tests written RED-first (verified failing to compile before implementation), then GREEN.

### Task 4 — Icon resolution + planner handoff

- `bucketIcon` gained a `List<Subcategory>` positional parameter and now resolves its matched
  (sub)category via `categorySelectionForBucket`, passing the subcategory into `trailCategoryIcon`.
- `bucketForState` returns `RouteTravelBucket?` and yields `null` for any costing that is neither
  `pedestrian` nor `bicycle` (previously `default:` silently mapped `auto` to `bikingHybrid`).
- Both picker callers pass `ref.watch(subcategoryProvider)`; both already compared with
  `bucket == selectedBucket`, which tolerates `null` = "nothing highlighted".
- `buildDraftTrail` gained a `String? subcategory` parameter (`?? ''` clears the relation, matching
  the existing convention); `finishPlanning` resolves the exact bucket via `bucketForState` and
  pre-fills both ids, pre-filling nothing when the session's costing matches no bucket.

### Task 5 — Remaining call sites + root-cause-2 fix

- `navigation_launch_util.dart`, `trail_download_service.dart` (new `subcategories`/`subcategoryPrefs`
  params, since the service holds no `Ref`), `trail_download_state_provider.dart`, and
  `navigation_screen.dart`'s road-snap branch all use the new `costingForCategory` shape.
- `navigation_screen.dart`'s trail-less-recording path maps the binary `_recordingCosting` onto a
  bucket and then `categorySelectionForBucket`, producing a `(categoryId, subcategoryId?)` pair — this
  also fixes a pre-existing silent gap where a real trail's own `subcategoryId` was never carried
  through the save-track flow. A doc comment records the binary-precision limitation.
- **`_onEditRoute` (root cause 2)**: now reads the live form value
  (`_formKey.currentState?.fields['category']?.value`, no `saveAndValidate()`) through
  `CategoryPicker.resolve`, falling back to the trail's saved ids only when the field is unset. It
  passes the resolved costing through unclamped (so an `auto` trail seeds the planner with `auto`) and
  adds `'costingOptions': {'bicycle_type': …}` when the profile carries one, so an MTB/Gravel/Road
  trail re-opens on its own bike variant instead of Valhalla's Hybrid default.

### Task 6 — Docs

`docs/.../custom-categories.md` gained a `settings` row on the Subcategory fields table and a
`### valhalla_profile` section covering: the open grammar and `bicycle_<type>` special form, a worked
`Car` → `auto` JSON example (linking to Valhalla's own costing-models docs rather than duplicating the
list), the shipped defaults table, the 4-step resolution/fallback order with the typo-degrades note,
and the planner-picker caveat.

## Verification results

| Task | Command | Result |
| ---- | ------- | ------ |
| 1 | `go build ./... && go vet ./... && go test ./util/... -run 'ValhallaProfile\|SeedDefault…'` | PASS (all 8 new/extended tests) |
| 2 | `build_runner build` + `flutter analyze` (6 files) + 4 greps | PASS — "No issues found!" |
| 3 | `flutter test test/util/valhalla_util_test.dart` + analyze + zero-heuristics grep | PASS — 25/25 |
| 4 | `flutter analyze` (4 files) + `categorySelectionForBucket` grep + zero-heuristics grep (sum=0) | PASS |
| 5 | `flutter analyze` (whole project) + tests + old-signature grep (sum=0) | PASS |
| 6 | docs greps + `go build`/`go vet`/`go test ./util/...` + `flutter analyze` + tests | PASS |

Whole-project `flutter analyze` reports **37 issues, all pre-existing `deprecated_member_use` infos in
`lib/util/icon_util.dart`** (deprecated Font Awesome icon names), exit code 0 — zero errors, zero
warnings, none in touched files.

### Full `flutter test` suite

A full-suite run (beyond the plan's verify commands) surfaced 7 widget-test failures:

- **3 in `travel_profile_sheet_test.dart` — caused by this task, fixed.** See Deviation 6.
- **4 in `settings_tab_test.dart` — pre-existing at `HEAD`, NOT fixed** (out of scope). Logged to
  `deferred-items.md` with proof: `AppLocalizations.of(context)!` already existed at
  `git show HEAD:…/settings_tab.dart:31`, the harness has never supplied `localizationsDelegates`,
  and the throw happens at line 32 — before any line this task touched in that file.

Both Go (`go test ./util/...`) and the two util Dart test files are fully green.

## Deviations from Plan

### 1. [Rule 3 — Blocking] Migration timestamp changed 1785100000 → 1785200000

- **Found during:** Task 1
- **Issue:** The plan specified `db/migrations/1785100000_updated_subcategories.go`, but
  `db/migrations/1785100000_rename_region_archives_region_id_to_path.go` already occupies that
  timestamp, and `1785180139_updated_trails_filter.go` is later still. A duplicate timestamp gives
  PocketBase two migrations with an ambiguous ordering key.
- **Fix:** Used `1785200000_updated_subcategories.go` — unique and strictly after every existing
  migration. Field id, collection id, and JSON literal are exactly as the plan specified.
- **Files:** `db/migrations/1785200000_updated_subcategories.go`

### 2. [Rule 3 — Blocking] Task 4's `route_travel_bucket.dart` rewrite pulled forward into Task 3

- **Found during:** Task 3 verification
- **Issue:** Task 3's verify command runs `flutter test test/util/valhalla_util_test.dart`. The new
  test file imports `route_travel_bucket.dart` (for `RouteTravelBucket`), which still called the two
  functions Task 3 deletes — so the test could not compile, regardless of its own correctness. The
  plan's Task 3 `<done>` anticipated call sites failing analyze but the verify gate itself was blocked.
- **Fix:** Performed Task 4 steps 1, 1b, 2 and 3 (`bucketIcon` rewrite, `bucketForState` nullability,
  both picker callers) before re-running Task 3's verify. No behavior differs from the plan — only the
  ordering. Task 4's own verify was then re-run in full and passes.
- **Files:** `app/lib/util/route_travel_bucket.dart`,
  `app/lib/components/route_planner/settings_tab.dart`,
  `app/lib/components/route_planner/travel_profile_sheet.dart`

### 3. [Rule 2 — Missing coverage] Added regression tests to `route_travel_bucket_test.dart`

- **Found during:** Task 5/6
- **Issue:** The plan's file list did not include `app/test/util/route_travel_bucket_test.dart`, but
  Task 4 changed `bucketForState`'s return type to nullable and added `valhallaProfileKey` — the new
  "returns `null` for a non-bucket costing" contract (the whole point of deviation-proofing the `auto`
  case) had zero test coverage, and nothing pinned `valhallaProfileKey` to the bucket's real payload.
- **Fix:** Added one test asserting `bucketForState('auto'|'truck'|'motor_scooter', …) == null`, and a
  group asserting `valhallaProfileKey` for all 5 buckets plus a round-trip that `valhallaProfile`
  matches each bucket's own `costing`/`costingOptions['bicycle_type']`. Every pre-existing test in
  that file was left untouched and still passes.
- **Files:** `app/test/util/route_travel_bucket_test.dart`

### 4. [Minor] Explicit `<String, dynamic>` annotation on the `costingOptions` extra

- **Found during:** Task 5
- **Issue:** `router_provider.dart:262` casts `extra?['costingOptions'] as Map<String, dynamic>?`. An
  inferred `Map<String, String?>` literal would be a covariant subtype and work, but relies on
  inference that a later refactor could silently change.
- **Fix:** Wrote the literal as `<String, dynamic>{'bicycle_type': profile!.bicycleType}`.
- **Files:** `app/lib/routes/trail_create_screen.dart`

### 5. [Minor] Doc wording touch-up outside the plan's stated edit

- **Found during:** Task 6
- **Issue:** The existing "Currently, the following setting is supported:" line would become
  inaccurate once a second settings key is documented on the same page.
- **Fix:** Changed to "Currently, the following waypoint-merge settings are supported:" and added an
  anchor link from the Category fields table's `settings` row.
- **Files:** `docs/src/content/docs/run/backend-configuration/custom-categories.md`

### 6. [Rule 1 — Bug I introduced] `travel_profile_sheet_test.dart` provider override

- **Found during:** post-Task-6 full-suite run
- **Issue:** Task 4 made `travel_profile_sheet.dart` call `ref.watch(subcategoryProvider)`. The real
  `SubcategoryNotifier.build()` reads `objectBoxProvider`, which the widget test does not stub, so all
  3 tests in the file began throwing. This is a genuine regression from this task, not a pre-existing
  failure (unlike `settings_tab_test.dart` — see below).
- **Fix:** Added a `_FakeSubcategoryNotifier extends SubcategoryNotifier` returning `const []`
  (mirroring the file's existing `_FakeCategoryNotifier` pattern) and a
  `subcategoryProvider.overrideWith(...)` line to all 3 `ProviderScope` override lists. No production
  code changed; no existing assertion altered.
- **Files:** `app/test/components/route_planner/travel_profile_sheet_test.dart`
- **Verified:** `flutter test test/components/route_planner/travel_profile_sheet_test.dart` — 3/3 pass.

## Deferred Issues

`settings_tab_test.dart`'s 4 failures are **pre-existing at `HEAD` and were deliberately not fixed**,
per the executor's scope boundary (only issues caused by this task's changes are auto-fixed). Full
diagnosis, proof of pre-existence, and a suggested fix are recorded in
`.planning/quick/260726-rn1-replace-heuristic-category-travel-profil/deferred-items.md`, along with
the 37 pre-existing `icon_util.dart` deprecation infos.

Note for whoever picks that up: fixing the l10n delegate will unblock those 4 tests to the point where
they *also* need a `subcategoryProvider` override, for the same reason Deviation 6 documents — the
l10n throw currently masks it.

## Known Stubs

None. Every resolution path is wired to real data; no placeholder values, empty-literal props, or
TODO markers were introduced.

## Threat Flags

None beyond the plan's existing `<threat_model>`. The two mitigate-dispositioned threats are
implemented as specified:

- **T-rn1-01** — `ValhallaProfile.parse` is total and non-throwing: non-String, empty/blank, and
  format-guard-failing input all return `null`. No `.byName`-style throwing lookup and no force-unwrap
  on operator-supplied data (`RouteTravelBucket.valhallaProfile`'s `!` is on a compile-time-constant
  string derived from the bucket's own payload, not on external input).
- **T-rn1-05** — no allowlist; the `^[a-z][a-z_]*$` guard rejects whitespace, digits, punctuation, and
  structural JSON characters. The value is serialized as a normal JSON string value (no
  concatenation), so it cannot alter request structure.

No new packages were added on either side (`Package Legitimacy Gate` not triggered).

## Manual smoke checks — NOT performed

The plan's 3 manual checks require a running instance and were not executed:

1. Route planner "Biking / Mountain" → Finish → draft trail's subcategory should be Biking/MTB.
2. Edit that draft's route after toggling its category away from Biking and back — planner should
   reopen on bike costing, not pedestrian.
3. In PocketBase admin, create a `Car` category with `{"valhalla_profile": "auto"}`, assign a trail,
   start navigation — `/valhalla/route` should carry `costing: "auto"`, and the planner picker should
   highlight nothing rather than defaulting to a bike bucket.

## Self-Check: PASSED

- Both new source files exist on disk (`db/migrations/1785200000_updated_subcategories.go`,
  `app/lib/models/valhalla_profile.dart`).
- `HEAD` is still `4b192622` — unchanged from the baseline.
- `git diff --cached --name-only` is empty — **nothing staged**.
- `git stash list` is empty — **no stash was created or popped**.
- `git diff --diff-filter=D --name-only` is empty — **no tracked file was deleted**.
- 28 modified + 5 untracked paths in the working tree; `git checkout .` plus removing the 2 new
  source files reverts everything cleanly.
