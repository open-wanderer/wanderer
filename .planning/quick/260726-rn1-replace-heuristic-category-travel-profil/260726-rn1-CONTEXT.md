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
- **[Correction 1, locked]** The setting's value vocabulary must translate
  *directly* to a Valhalla costing option, NOT an arbitrary Dart enum name.
  This supersedes the earlier "mirror `RouteTravelBucket.name`
  (`bikingMountain` etc.)" plan — that vocabulary was an internal Dart
  identifier, not something an operator reading raw PocketBase JSON would
  recognize as a Valhalla costing.
- **[Correction 2, locked — OPEN vocabulary]** `valhalla_profile` is **not**
  restricted to a fixed set of 5 strings. It accepts **any** Valhalla costing
  model name, so an operator who creates a new category (e.g. "Car") can set
  `valhalla_profile: "auto"` and the Flutter app will route that category
  with Valhalla's `auto` costing. Grammar:

  ```
  valhalla_profile := <costing>                     e.g. "auto", "pedestrian",
                                                         "motor_scooter", "truck"
                    | "bicycle_" <bicycle_type>     e.g. "bicycle_mountain"
  ```

  - The **only** special-cased form is `bicycle_<type>`, where `<type>` is one
    of Valhalla's 4 `bicycle_type` values (`road`, `hybrid`, `cross`,
    `mountain`). It parses to costing `bicycle` + `bicycle_type` `Road`/
    `Hybrid`/`Cross`/`Mountain` (Valhalla's capitalization).
  - **Every other value is passed through verbatim as the costing name**, with
    no `bicycle_type`. Critically, do NOT split on `_` generally — Valhalla has
    legitimately underscored costing names (`motor_scooter`) that must survive
    intact. Match the literal `bicycle_(road|hybrid|cross|mountain)` prefix
    form only; everything else is an opaque costing string.
  - Unknown/garbage values are **not** validated against an allowlist (Valhalla
    gains costing models over time; hardcoding a list would block operators
    from new ones the moment upstream adds them). They pass through to the
    Valhalla proxy, which rejects genuinely invalid ones. Apply only a cheap
    format sanity guard (lowercase letters/underscores) and treat a value
    failing that as "unset" so a typo degrades to the fallback chain rather
    than producing a malformed routing request.

  Model this as a small parsed value type (e.g. `ValhallaProfile` with
  `costing` + `bicycleType?`) rather than a closed enum, since the value space
  is open. `RouteTravelBucket` keeps a `valhallaProfileKey` getter deriving its
  own canonical string from its existing `.costing`/`.costingOptions`
  (`pedestrian`, `bicycle_hybrid`, `bicycle_mountain`, `bicycle_cross`,
  `bicycle_road`) so the 5 shipped buckets can never drift from their actual
  Valhalla request payload — but the bucket enum is now only the **route
  planner's picker vocabulary**, not the limit of what a category may map to.
- Seed the new setting on the built-in default categories/subcategories so
  the mapping works out of the box (values are the Valhalla-native strings):
  - Category `Hiking` (`db/util/category_defaults.go`) → `pedestrian`.
  - Subcategory `Biking / Touring` → `bicycle_hybrid`.
  - Subcategory `Biking / MTB` → `bicycle_mountain`.
  - Subcategory `Biking / Gravel` → `bicycle_cross`.
  - Subcategory `Biking / Road` → `bicycle_road`.
  - Subcategory `Biking / E-Bike` → `bicycle_hybrid` (Valhalla has no
    dedicated e-bike costing model; Hybrid is its closest general-purpose
    bike profile).
  - (Default subcategories are defined in `db/util/subcategory_defaults.go`;
    confirmed these 5 names — MTB, Gravel, Touring, Road, E-Bike — already
    exist as Biking's default subcategories.)
- Update `app/lib/models/category.dart` (`Category.settings`) and the
  `Subcategory` model (find/add `settings` field) to parse this JSON field,
  plus a typed accessor (`Category.valhallaProfile` /
  `Subcategory.valhallaProfile`) returning the parsed `ValhallaProfile?`.
- Update `docs/src/content/docs/run/backend-configuration/custom-categories.md`
  — add `settings` to the Subcategory fields table (currently only listed
  under Category fields) and document the `valhalla_profile` key: the open
  grammar above, the `bicycle_<type>` special form, the shipped defaults, and
  a worked "Car" → `auto` example showing operators they can map new
  categories onto any Valhalla costing.

### Route planner picker vs. open profiles (implication of Correction 2)
The Route Planner's entry sheet / Settings tab keeps offering exactly its 5
`RouteTravelBucket` options — this change does not add UI for arbitrary
costings. Consequences to handle explicitly:
- **Icon resolution + planner category pre-fill** stay bucket-based; a
  category mapped to a non-bucket costing (e.g. `auto`) simply never matches a
  bucket and is not offered/pre-filled by the planner. That's correct, not a
  bug.
- **Editing the route of a trail whose category maps to a non-bucket costing**
  (e.g. a "Car" trail) must still route with that costing — pass the resolved
  costing through to the planner/Valhalla rather than silently downgrading it
  to pedestrian. `bucketForState` currently defaults any non-`pedestrian`
  costing to `bikingHybrid`; it must instead return `null` for a costing that
  matches no bucket, and the picker must tolerate "no bucket highlighted".
- **Navigation, road-snap, and offline download** all consume the resolved
  costing directly, so they get open-costing support for free.

`valhalla_profile` remains an operator-facing **opt-in** field — categories and
subcategories that don't set it (e.g. Climbing, Running, Skiing, Walking)
simply have no mapped profile and fall through the fallback chain below.

### Category with no subcategory chosen
When a trail's category is "biking-shaped" (the category itself, or at least
one of its subcategories, resolves to a profile whose costing is `bicycle`)
but no specific subcategory is selected/available, default to
**`bicycle_hybrid`** — Valhalla's general-purpose bike costing
(`RouteTravelBucket.bikingHybrid`'s own profile). This is a code-level
fallback, not necessarily a seeded setting on the parent "Biking" category
itself.

Note this tier is deliberately **bicycle-specific**: it exists because
Valhalla's 4 bike variants share one `bicycle` costing and need a sane
`bicycle_type` default. It does not generalize to other costings — a "Car"
category with no subcategory just resolves to its own `auto` profile via the
category tier, no special-casing needed.

### Definition of "unavailable" for fallback-to-parent
**[Correction 3, locked — supersedes the earlier "both" answer]** Visibility
preferences apply to the **reverse** direction only, not to resolving an
existing trail.

- **Forward (existing trail → costing):** a subcategory counts as unavailable
  *only* when it does not exist in the operator's currently loaded subcategory
  list (never created, or deleted). This check is load-bearing:
  `trails.subcategory` is declared `cascadeDelete: false`
  (`db/migrations/1781000000_categories_redesign.go`), so deleting a
  subcategory leaves a **dangling id** on every trail that referenced it.
  Visibility preferences are deliberately NOT consulted here. Evidence: the
  subcategory `visible` flag drives only display surfaces everywhere else in
  the app — which entries appear in `CategoryPicker`'s dropdown
  (`category_picker.dart`), and opacity/toggles on the two settings screens.
  Treating it as semantic would mean hiding "MTB" to declutter a picker
  silently changes how existing MTB trails route, which is a surprising side
  effect from a UI preference and inconsistent with its use everywhere else.
- **Reverse (bucket → category/subcategory pre-fill):** visibility DOES apply.
  A hidden subcategory is never auto-assigned to a trail on the user's behalf;
  `categorySelectionForBucket` falls back to a matching category instead. This
  is the correct home for the original requirement ("fallback ... because a
  user has disabled it") — the preference governs what we may newly assign,
  not how an already-tagged trail routes.

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
