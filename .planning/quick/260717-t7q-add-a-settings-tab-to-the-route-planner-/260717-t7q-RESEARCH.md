# Quick Task 260717-t7q: Add a Settings tab to the Route Planner sheet — Research

**Researched:** 2026-07-17
**Domain:** Flutter route planner (Riverpod) + Valhalla costing_options
**Confidence:** HIGH (all findings read directly from source; no external lookups needed)

## Summary

All four questions are answered from source. The web reference stores costing options as
already-snake_case objects and POSTs them nested as `costing_options: { <profile>: {...} }`.
The Dart side today sends only `costing: state.travelProfile` with no options — adding the
nested block is a small change to `_resolveSegment`'s request body. The genuinely hard part
is the cross-family anchor migration: `routeAnchorsProvider` is an **autoDispose** family keyed
by `travelProfile` (only two possible keys: `'pedestrian'`/`'bicycle'`), and the screen holds
that key as an **immutable `widget.travelProfile` field**. No bulk re-resolve method exists.

**Primary recommendation:** Store the selected bucket's costing_options *inside* `RouteAnchorsState`
(not as a family key — the family key can't distinguish Road from Mountain since both are `'bicycle'`).
Add three notifier methods: `setCostingOptions(...)`, `resolveAllSegments()`, and `seedFrom(...)`.
Make the screen hold `travelProfile` as **mutable State**. For the cross-family switch, seed the
new family instance in a **post-frame callback** (after the rebuild has re-subscribed) to dodge the
autoDispose race. A cleaner alternative (drop the family arg entirely → single keepAlive provider)
is described in Q3 and is the more robust long-term shape.

---

## Q1 — Exact Valhalla costing_options per bucket

**Source of truth:** `web/src/lib/components/trail/route_editor.svelte` (default objects lines 59–87;
per-type speed `adjustSpeeddependingOnBikeType` lines 103–123) and `web/src/lib/models/valhalla.ts`
(field names, lines 8–42). [VERIFIED: codebase]

**Field names to mirror (already snake_case in the web defaults — copy verbatim):**
- Pedestrian: `max_hiking_difficulty`, `walking_speed`, `use_hills`, `shortest`
- Bicycle: `bicycle_type`, `cycling_speed`, `use_roads`, `use_hills`, `avoid_bad_surfaces`, `shortest`

**Per-type `cycling_speed`** comes from `adjustSpeeddependingOnBikeType` (NOT the generic `20`
default): City/Hybrid → 18, Road → 25, Cross → 20, Mountain → 16. `use_roads` (0.5), `use_hills`
(0.5), and `avoid_bad_surfaces` (0.25) are **not** varied by bike type in the web — only the speed
slider changes. `shortest` stays `false` (CONTEXT: no shortest toggle).

**Request body nesting** (from `valhalla_store.svelte.ts` lines 66–72):
`costing_options: { [modeOfTransport]: <optionsObject> }` — keyed by the same `'bicycle'`/`'pedestrian'`
string as `costing`. [VERIFIED: codebase]

The 5 hardcoded Dart payloads (each goes under `costing_options: { <profile>: {...} }`):

| Bucket | profile (`costing`) | costing_options payload |
|--------|--------------------|--------------------------|
| **Hiking** | `pedestrian` | `{'max_hiking_difficulty': 6, 'walking_speed': 5.1, 'use_hills': 1, 'shortest': false}` |
| **Biking/Hybrid** | `bicycle` | `{'bicycle_type': 'Hybrid', 'cycling_speed': 18, 'use_roads': 0.5, 'use_hills': 0.5, 'avoid_bad_surfaces': 0.25, 'shortest': false}` |
| **Biking/Road** | `bicycle` | `{'bicycle_type': 'Road', 'cycling_speed': 25, 'use_roads': 0.5, 'use_hills': 0.5, 'avoid_bad_surfaces': 0.25, 'shortest': false}` |
| **Biking/Cross** | `bicycle` | `{'bicycle_type': 'Cross', 'cycling_speed': 20, 'use_roads': 0.5, 'use_hills': 0.5, 'avoid_bad_surfaces': 0.25, 'shortest': false}` |
| **Biking/Mountain** | `bicycle` | `{'bicycle_type': 'Mountain', 'cycling_speed': 16, 'use_roads': 0.5, 'use_hills': 0.5, 'avoid_bad_surfaces': 0.25, 'shortest': false}` |

Note `bicycle_type` enum values are capitalized (`'Road' | 'Hybrid' | 'City' | 'Cross' | 'Mountain'`),
as Valhalla expects (valhalla.ts line 35). Only Hybrid/Road/Cross/Mountain are used (no City bucket).

**Current Dart request body** (`route_anchor_provider.dart` lines 118–128) sends only
`directions_type`, `locations`, `costing`. Add one line:
```dart
if (state.costingOptions != null)
  'costing_options': {state.travelProfile: state.costingOptions},
```

---

## Q2 — Category-name → bucket heuristic

**Existing pattern** (`app/lib/util/gpx_util.dart` lines 24–63): two small functions using
case-insensitive `.contains(...)` substring checks against a category `name`/`shortName` haystack.
`costingForCategory` maps a name → `'bicycle'`/`'pedestrian'`; `categoryForTravelProfile` does the
inverse, returning the *first matching `Category.id`* or `null`. Order of checks is documented as
mattering (`bike` → `cycling` → `bicycle`). [VERIFIED: codebase]

**Key finding — do NOT synthesize fake Categories.** `Category` is a `@freezed` class (`category.dart`)
requiring a real `id`, `name`, and PocketBase-backed `translations`. A bucket's `bicycle_type` enum
value is **fixed regardless of what categories the operator happens to have** (CONTEXT D-40). Therefore
**the picker option must carry the enum/costing payload itself**, and the operator's `Category` list is
consulted **only to resolve an icon** (via `Category.icon`). When no category matches a bucket, fall
back to a hardcoded FontAwesome icon — always show all 5 options (CONTEXT: lean toward all 5).

**Recommended structure** — a fixed const/enum of 5 options carrying payload + keywords + fallback
icon, plus a new icon-resolution helper added to `gpx_util.dart` alongside the existing two functions
(so the heuristic family stays co-located). Mirror the existing style exactly:

```dart
/// Sibling of [categoryForTravelProfile]: given a bike-bucket keyword set and
/// the operator's loaded categories, returns the first Category whose
/// name/shortName matches (for icon resolution only), else null. Never throws
/// on an empty list; degrades to null (caller uses a hardcoded fallback icon).
Category? categoryForBikeBucket(List<String> keywords, List<Category> categories) {
  bool matches(Category c) {
    final hay = '${c.name} ${c.shortName ?? ''}'.toLowerCase();
    return keywords.any(hay.contains);
  }
  return categories.firstWhereOrNull(matches);
}
```

**Proposed keyword sets** (specific bucket keywords first; generic bike fallback only if no specific
category exists — planner's discretion on exact wording, this is CONTEXT "Claude's Discretion"):

| Bucket | Keywords (extends existing `bike`/`cycling`/`bicycle` base) |
|--------|-------------------------------------------------------------|
| Hiking | reuse existing: `hik`, `walk`, `foot` |
| Biking/Road | `road`, `race`, `racing` |
| Biking/Mountain | `mountain`, `mtb`, `downhill`, `enduro` |
| Biking/Cross | `cross`, `cyclocross`, `gravel`, `cx` |
| Biking/Hybrid | `hybrid`, `city`, `commut`, `touring`, `trekking`, `urban` |

Fallback icons (when no category matches): reuse the entry sheet's existing
`FontAwesomeIcons.personHiking` (hiking) and `FontAwesomeIcons.bicycle` (all bike buckets), matching
`travel_profile_sheet.dart` lines 49/56.

**Icon resolution call to reuse** (`category_icon_util.dart` + `category_picker.dart`): the picker
uses `trailCategoryIcon(cat, size: 16)` (returns a `Widget`), which already falls back to
`FontAwesomeIcons.route` internally when a category's `icon` string is unknown. For the picker's 5
options, resolve `categoryForBikeBucket(...)` → if non-null, `trailCategoryIcon(cat, size: 20)`; if
null, a plain `FaIcon(<fallback>, size: 20)`. Note `trailCategoryIcon`'s own fallback is
`FontAwesomeIcons.route`, so a null-safe pattern is: pick the hardcoded per-bucket FA icon when the
matched category is null, rather than passing null into `trailCategoryIcon`.

---

## Q3 — Cross-family anchor migration mechanics

**Confirmed facts from `route_anchor_provider.dart`:** [VERIFIED: codebase]
- `RouteAnchorsState` holds `anchors`, `segments`, `autoRoutingEnabled`, `travelProfile` (documented
  "fixed for the notifier's lifetime, never switched mid-session"), `undoStack`, `redoStack`. No
  `costingOptions` field exists.
- `@riverpod class RouteAnchors extends _$RouteAnchors` with `build(String travelProfile)` → generates
  the family `routeAnchorsProvider(travelProfile)`. Lowercase `@riverpod` ⇒ **autoDispose** (riverpod
  3.x codegen default; would need `@Riverpod(keepAlive: true)` to persist). `riverpod_annotation ^4.0.2`,
  `riverpod_generator ^4.0.3`, `flutter_riverpod ^3.3.1` (pubspec.yaml). [VERIFIED: pubspec]
- **No bulk re-resolve exists.** Only `_resolveSegment` (per anchor-pair) and `retrySegment` (single
  pair). `toggleAutoRouting()` just flips the flag — it does NOT re-resolve existing straight segments.
- Anchors are addressable: `state.anchors` is the ordered `List<RouteAnchor>` (each with `id`, `lat`,
  `lon`, `point`). Segments link `beforeAnchorId → afterAnchorId`.

**`widget.travelProfile` is immutable per instance.** `RoutePlannerScreen.travelProfile` is a `final`
field set by the GoRouter entry point (Phase 21) and never mutated. Every read in the screen,
`RouteAnchorSheet`, `RouteAnchorListTab`, `ElevationTab`, `RouteAnchorLayer`, and `plannedGpxProvider`
threads this same key. So mid-session switching **cannot** be done by changing `widget.travelProfile`.

**Widgets/providers keyed by `travelProfile` that must all follow a switch** (from grep):
`route_planner_screen.dart`, `route_anchor_sheet.dart`, `route_anchor_list_tab.dart`,
`elevation_tab.dart`, `route_anchor_layer.dart`, `planned_gpx_provider.dart` (also autoDispose
`@riverpod`, `plannedGpx(Ref, String travelProfile)`), and `route_planner_handoff_util.dart`
(reads the current profile in `finishPlanning`).

### Recommendation A — keep the family, make the screen's profile mutable (smaller diff)

New state field + notifier methods on `RouteAnchors`:
1. Add `Map<String, dynamic>? costingOptions` to `RouteAnchorsState` (+ `copyWith`, + `build` default null).
2. `void setCostingOptions(Map<String,dynamic> opts)` → `state = state.copyWith(costingOptions: opts)`,
   then call `resolveAllSegments()`. Used for **within-`bicycle` bucket switches** (Hybrid→Road etc.)
   and to apply the initial bucket.
3. `void resolveAllSegments()` — iterate every consecutive anchor pair, and for each existing segment
   re-dispatch `_resolveSegment(before, after, a, b)` when `autoRoutingEnabled` (else mark straight).
   This is the missing bulk method; model it on `reorderAnchors`'s `toResolve` loop (lines 397–424).
4. `void seedFrom(List<RouteAnchor> anchors, Map<String,dynamic> opts)` — rebuild `anchors` +
   straight segments between consecutive anchors (mirror `appendAnchor`'s segment construction),
   set `costingOptions: opts`, then `resolveAllSegments()`. Used for the **cross-family** switch.

Screen restructure: replace `widget.travelProfile` reads with a mutable `late String _travelProfile`
in `_RoutePlannerScreenState` (init from `widget.travelProfile` in `initState`). Route it into all
child widgets. A profile switch becomes:

```dart
void _switchProfile(String newProfile, Map<String,dynamic> opts) {
  final crossFamily = newProfile != _travelProfile;
  if (!crossFamily) {
    ref.read(routeAnchorsProvider(_travelProfile).notifier).setCostingOptions(opts);
    return; // within bicycle: no migration, just re-resolve under new costing
  }
  final old = ref.read(routeAnchorsProvider(_travelProfile)); // capture BEFORE switching
  setState(() => _travelProfile = newProfile);                // rebuild re-subscribes to new key
  WidgetsBinding.instance.addPostFrameCallback((_) {          // AFTER watch is established
    if (!mounted) return;
    ref.read(routeAnchorsProvider(newProfile).notifier)
       .seedFrom(old.anchors, opts);
  });
}
```

**Old family instance lifecycle:** because both `routeAnchorsProvider` and `plannedGpxProvider` are
autoDispose, once the rebuild drops all watchers on the old key it is **disposed automatically** — no
leak. `_inFlight`/`_generation` maps live on the (disposed) notifier and go with it.

**Riverpod pitfalls to flag:**
- **autoDispose seeding race (the important one):** calling `ref.read(routeAnchorsProvider(newKey).notifier)`
  before any widget watches the new key creates a temporary instance whose dispose is scheduled the
  moment the one-off `ref.read` link is dropped — seeded state can be thrown away before the rebuild
  subscribes. **Fix:** seed in a `addPostFrameCallback` *after* `setState`, so `ref.watch` in `build`
  has already kept it alive. (This is why the snippet above orders capture → setState → post-frame seed.)
- **Undo/redo across the boundary:** the new instance starts with empty undo/redo stacks. Decide
  (planner) whether a profile switch is itself undoable — simplest is that it is a fresh baseline
  (no cross-family undo), consistent with `travelProfile` previously being "fixed for lifetime."
- **`ref.listen` in `route_planner_screen.dart` (line 98)** is keyed by the profile too — it must key
  off `_travelProfile` so segment-layer sync follows the switch.

### Recommendation B — drop the family arg → single keepAlive provider (cleaner, bigger diff)

Convert `RouteAnchors` to `@Riverpod(keepAlive: true)` with **no** family argument; move `travelProfile`
and `costingOptions` fully into state; add one `void switchProfile(String profile, Map opts)` that
sets both and calls `resolveAllSegments()` — handling cross-family and within-family **uniformly**
(anchors never leave the single instance, so there is **no migration and no autoDispose race at all**).
Cost: every `routeAnchorsProvider(travelProfile)` / `plannedGpxProvider(travelProfile)` call site loses
its argument (mechanical edit across the 6 files above), and the entry point sets the initial profile
via `switchProfile` after mount instead of via a family key. This is the more robust shape and is
squarely within CONTEXT's "where the unified selection state lives" discretion. **Recommend B** if the
planner is willing to touch all call sites; otherwise A is a correct, smaller change.

---

## Q4 — Existing tab / UI patterns to reuse

**Current sheet state** (`route_anchor_sheet.dart`): the working-tree file is the **2-tab committed
version** (`DefaultTabController(length: 2)`, tabs = Route Anchors / Elevation). `git diff` for this
file is **empty** — the "uncommitted TabBar edit" noted in the brief is not present on disk now; plan
against the 2-tab version. [VERIFIED: git diff]

To add a 3rd "Settings" tab:
- Bump `DefaultTabController(length: 2)` → `3` (line 60).
- Add a third `Tab(...)` to the `tabs:` list (lines 120–129) — e.g.
  `Tab(icon: FaIcon(FontAwesomeIcons.gear, size: 16), text: 'Settings')`. Match the existing
  `labelColor`/`indicatorColor` treatment already set on the `TabBar`.
- Add the third page to `TabBarView.children` (lines 135–147). **Scroll-controller rule (documented at
  the top of this file):** only `RouteAnchorListTab` may receive the builder's `scrollController` —
  `ElevationTab` gets none, and the new `SettingsTab` must **not** take the shared controller either
  (TabBarView keeps all pages built simultaneously → sharing a `ScrollController` throws
  "attached to multiple scroll views", flutter#55388). If the Settings tab needs to scroll, give it its
  **own** local `ScrollController`/`SingleChildScrollView`.

**Third-tab structural pattern** (`elevation_tab.dart`): a `ConsumerStatefulWidget` taking
`final String travelProfile` (constructor `const ElevationTab({super.key, required this.travelProfile})`).
The Settings tab can be simpler — a `ConsumerWidget` if it only reads state and dispatches notifier
methods. It receives `travelProfile` (or `_travelProfile` under Rec A) the same way the other tabs do.
It hosts: (a) the relocated auto-routing toggle, and (b) the 5-option unified picker.

**Auto-routing toggle to relocate** (`route_planner_screen.dart` `_buildAutoRoutingToggle`, lines
336–362): backed by `routeAnchorsProvider(travelProfile).notifier.toggleAutoRouting()`, current value
`state.autoRoutingEnabled`. Moving it into the Settings tab means removing it from the top-right
`Column` (lines 254–265, leaving Undo/Redo) and rendering an equivalent control (e.g. a `SwitchListTile`)
in the tab. The notifier call is unchanged.

**Icon resolution for the 5 picker options** (see Q2): reuse `trailCategoryIcon(cat, size: N)` from
`category_icon_util.dart` (same call `category_picker.dart` line 105 uses) for matched categories,
with a per-bucket hardcoded `FaIcon` fallback when `categoryForBikeBucket(...)` returns null.

**Entry-point sheet (`travel_profile_sheet.dart`)** currently returns `'pedestrian'`/`'bicycle'` via two
`_TravelProfileCard`s (lines 48–60). To expand to 5 options it must now return **both** the profile and
the bucket's costing_options (e.g. a small result record/class, since `'bicycle'` alone can't encode
Road vs Mountain). The existing `_TravelProfileCard` (self-contained, 16px radius, `secondaryContainer`
@40% badge, `primary` icon tint) is directly reusable for a 5-card list; per-card icon comes from the
same Q2 resolution.

---

## Assumptions Log

| # | Claim | Section | Risk if wrong |
|---|-------|---------|---------------|
| A1 | Bike-bucket keyword sets (road/mtb/cross/gravel/hybrid/city …) match the operator's real category names | Q2 | Icon falls back to hardcoded FA icon — cosmetic only; enum/costing payload is unaffected (that's the point of carrying it on the option, not the Category) |
| A2 | `use_roads`/`use_hills`/`avoid_bad_surfaces` are intentionally constant across bike types (web only varies `cycling_speed`) | Q1 | Routing differs slightly from web per-type; values are read verbatim from web defaults so low risk |
| A3 | Undo/redo need not span a cross-family profile switch (fresh baseline) | Q3 | If users expect to undo a profile switch, add a snapshot in `seedFrom` — planner decision |

## Open Questions

1. **Does a profile switch push an undo snapshot?** Recommend: no (fresh baseline per profile), matching
   the prior "travelProfile fixed for lifetime" invariant. Planner may decide otherwise.
2. **Rec A vs Rec B** (family-with-mutable-screen-key vs single keepAlive provider). Research recommends
   B for correctness (eliminates the autoDispose seeding race) if the planner accepts editing all 6
   call sites; A is a smaller, still-correct diff with the post-frame-seed guard.

## Sources

- `web/src/lib/components/trail/route_editor.svelte` (defaults + per-type speed) — HIGH
- `web/src/lib/models/valhalla.ts` (field names/enums) — HIGH
- `web/src/lib/stores/valhalla_store.svelte.ts` (costing_options nesting) — HIGH
- `app/lib/util/gpx_util.dart`, `app/lib/provider/route_anchor_provider.dart`,
  `app/lib/provider/planned_gpx_provider.dart`, `app/lib/routes/route_planner_screen.dart`,
  `app/lib/components/route_planner/{route_anchor_sheet,elevation_tab,travel_profile_sheet}.dart`,
  `app/lib/components/trail/category_picker.dart`, `app/lib/util/category_icon_util.dart`,
  `app/lib/models/category.dart`, `app/pubspec.yaml` — HIGH (read in full)
</content>
</invoke>
