# Quick Task 260726-rn1: Replace heuristic category/travel-profile matching with explicit Valhalla-costing<->category bijection with subcategory-to-parent fallback - Context

**Gathered:** 2026-07-26
**Status:** Ready for planning

<domain>
## Task Boundary

Replace the current heuristic, English-substring-based matching between
Valhalla costing profiles and Wanderer categories/subcategories
(`app/lib/util/valhalla_util.dart`: `categoryForTravelProfile`,
`costingForCategory`, `categoryForBikeBucket`) with an explicit,
operator-configurable bijection:

| Valhalla travel bucket (`RouteTravelBucket`) | Wanderer (sub)category |
|---|---|
| Hiking | Category: Hiking |
| Biking / Hybrid | Subcategory: Biking / Touring |
| Biking / Mountain | Subcategory: Biking / MTB |
| Biking / Cross | Subcategory: Biking / Gravel |
| Biking / Road | Subcategory: Biking / Road |

This is the root fix for two related bugs already diagnosed this session:
1. Route planner "Finish planning" (biking profile) lands on
   `trail_create_screen` with the wrong category (e.g. "Hiking") because the
   old heuristic (`categoryForTravelProfile`) only substring-matches a
   category's raw, untranslated `name`/`short_name` against English keywords
   (`bike`/`cycling`/`bicycle`) and misses non-matching/translated names.
2. Re-opening the route planner in edit mode after manually fixing the
   category reverts to pedestrian costing (duration balloons) because
   `_onEditRoute` (`trail_create_screen.dart:303`) derives the profile from
   `trail.expand?.category?.name` — a relation that's never populated on an
   unsaved draft trail — instead of the live category selection.

Both the matching-heuristic replacement AND the live-selection read fix are
in scope; the new bijection alone does not fix bug 2 unless `_onEditRoute`
is also changed to resolve the *current* form category/subcategory selection
(not the stale `expand.category` relation) through it.

</domain>

<decisions>
## Implementation Decisions

### Matching key for (sub)categories
Add a new **optional field to `(sub)category.settings`** (the existing JSON
settings blob) rather than name/keyword matching. Concretely:
- `categories` collection already has a `settings` JSON field (added in
  `db/migrations/1763300311_updated_categories.go`, currently holding
  `wp_merge_enabled`/`wp_merge_radius`). Add a new key to it, e.g.
  `valhalla_profile`.
- `subcategories` collection (`db/migrations/1781000000_categories_redesign.go`,
  collection id `pbc_1781100000`) has **no `settings` field at all yet** —
  add one via a new migration, mirroring the categories migration's
  `AddMarshaledJSONAt` JSON-field pattern (categories collection id
  `kjxvi8asj2igqwf`, field id `json3846545605`, type `"json"`).
- The setting's value vocabulary must exactly mirror
  `RouteTravelBucket` (`app/lib/util/route_travel_bucket.dart`) — the 5
  fixed buckets: `hiking`, `bikingHybrid`, `bikingMountain`, `bikingCross`,
  `bikingRoad` (or equivalent stable identifiers derived from that enum) —
  not raw Valhalla costing strings, since a bucket also carries
  `bicycle_type`/`costingOptions`, not just `costing`.
- Seed the new setting on the built-in default categories/subcategories so
  the mapping works out of the box:
  - Category `Hiking` (`db/util/category_defaults.go`) → `hiking` bucket.
  - Subcategory `Biking / Touring` → `bikingHybrid`.
  - Subcategory `Biking / MTB` → `bikingMountain`.
  - Subcategory `Biking / Gravel` → `bikingCross`.
  - Subcategory `Biking / Road` → `bikingRoad`.
  - (Default subcategories are defined in `db/util/subcategory_defaults.go`;
    confirmed these 4 names — MTB, Gravel, Touring, Road — plus an unmapped
    `E-Bike` already exist as Biking's default subcategories.)
- Update `app/lib/models/category.dart` (`Category.settings`) and the
  `Subcategory` model (find/add `settings` field) to parse this JSON field,
  plus a typed accessor (e.g. `Category.valhallaProfile` /
  `Subcategory.valhallaProfile` getter parsing the enum-ish string).
- Update `docs/src/content/docs/run/backend-configuration/custom-categories.md`
  — add `settings` to the Subcategory fields table (currently only listed
  under Category fields) and document the new `valhalla_profile` setting
  key/values alongside the existing `wp_merge_enabled`/`wp_merge_radius`
  documentation, including which of the 5 bucket identifiers are valid.
- This is an operator-facing opt-in field — categories/subcategories that
  don't set it (e.g. Climbing, Running, Skiing, Walking, E-Bike) simply have
  no mapped profile and fall through the fallback chain below.

### Category with no subcategory chosen
When a trail's category resolves to "the biking category" (i.e. the
category itself, or its default subcategory set, maps into the biking
buckets) but no specific subcategory is selected/available, default to the
**Hybrid** bucket (`RouteTravelBucket.bikingHybrid` — Valhalla's
general-purpose bike costing). This is a code-level fallback, not
necessarily a seeded setting on the parent "Biking" category itself.

### Definition of "unavailable" for fallback-to-parent
A subcategory counts as unavailable — triggering fallback to the parent
category's mapping — in **both** of these cases:
- It does not exist in the operator's currently loaded subcategory list at
  all (e.g. never created, or deleted).
- It exists but is hidden by the user's category/subcategory visibility
  preferences (mirrors how `visibleSortedCategories`
  (`app/lib/util/category_preference_sort.dart`) already determines what's
  "available" elsewhere in the app — reuse/consult that existing
  availability logic rather than re-implementing visibility checks).

### Root cause 2 (live selection) — required, not optional
`_onEditRoute` (`app/lib/routes/trail_create_screen.dart:290-321`) must read
the **currently selected** category/subcategory from the live form field
(pattern already used at `trail_create_screen.dart:120`:
`_formKey.currentState?.fields['category']?.value`), not the stale
`trail.expand?.category` relation, then resolve the Valhalla profile through
the new bijection (with the fallback chain above) instead of
`costingForCategory(trail.expand?.category?.name)`.

</decisions>

<specifics>
## Specific Ideas

- Existing settings-field migration pattern to mirror exactly:
  `db/migrations/1763300311_updated_categories.go` (`AddMarshaledJSONAt`,
  type `"json"`, then backfill existing records via
  `app.FindAllRecords("categories")` + `record.Set("settings", ...)`).
- `categories` collection id: `kjxvi8asj2igqwf`. `subcategories` collection
  id: `pbc_1781100000` (from `db/migrations/1781000000_categories_redesign.go`).
- Default category/subcategory seed data lives in
  `db/util/category_defaults.go` and `db/util/subcategory_defaults.go`.
- `RouteTravelBucket` enum + all bucket data (costing, costingOptions,
  keywords) is in `app/lib/util/route_travel_bucket.dart` — this is the
  canonical vocabulary source; do not invent a separate identifier scheme.
- Old heuristic functions to replace/retire:
  `categoryForTravelProfile`, `costingForCategory`, `categoryForBikeBucket`
  in `app/lib/util/valhalla_util.dart` (note: `costingForCategory`'s
  signature was very recently changed mid-session from
  `(String? category)` to `(Category? category)` — that change is part of
  this same in-progress work and can be further revised/superseded freely).
- Call sites currently depending on the old heuristics (all need updating to
  the new bijection): `app/lib/util/route_planner_handoff_util.dart`
  (`finishPlanning`), `app/lib/routes/trail_create_screen.dart`
  (`_onEditRoute` + the "default empty category to user's top preference"
  logic around line 466-485), `app/lib/routes/navigation_screen.dart`
  (route-snap costing), `app/lib/services/trail_download_service.dart`,
  `app/lib/util/navigation_launch_util.dart`.
- Existing test file to rewrite for the new bijection:
  `test/util/valhalla_util_test.dart` (Go-side: check `db/util/category_test.go`
  for patterns to mirror if new Go-side helpers/tests are warranted for the
  subcategory settings migration/seeding).

</specifics>

<canonical_refs>
## Canonical References

- `docs/src/content/docs/run/backend-configuration/custom-categories.md` —
  must be updated as part of this task (not just a reference).

</canonical_refs>
