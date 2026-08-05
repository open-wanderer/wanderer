# Quick Task 260805-h1e: Profile trail map screen — Research

**Researched:** 2026-08-05
**Domain:** Flutter / Riverpod 3 code-gen families, maplibre-dart camera, SvelteKit+Meilisearch endpoint
**Confidence:** HIGH (everything below is read out of this repo at the cited file:line; no external claims)

---

<user_constraints>
## User Constraints (from CONTEXT.md — LOCKED, do not relitigate)

### Locked Decisions
1. **Bounding box source:** extend `web/src/routes/api/v1/trail/bounding-box/+server.ts` with an optional
   `handle` query param; resolve handle→actor exactly as `profile/[handle]/trails/+server.ts` does and AND
   `author = {actor.id}` into all four `multiSearch` queries. Absent → unchanged behaviour.
   `trails_bounding_box` PocketBase view was considered and rejected (viewRule is own-row-only; no
   public/private predicate; bypasses the tenant token).
2. **Provider scoping:** convert **both** `mapTrailSearchProvider` and `mapClusterSearchProvider` to
   `.family`. `map_screen.dart` passes the global/null key; the profile map passes the profile's actor id.
   Parallel duplicate providers rejected. The family key must reach the two internal
   `ref.listen(trailFilterProvider(...))` calls inside each `build()`.
3. **Camera state:** the profile map keeps **local** camera state; it never reads or writes
   `mapCameraProvider`. Accepted: it opens at the bbox fit every time, remembers nothing.
4. **Filter identity:** the profile map's `TrailQuickFilterBar` shares the list screen's filter id
   `profile_trail_{handle}`. Accepted: the map's filter bar mutates the list's filter state.

### Claude's Discretion
- handle → actorId resolution path on the Dart side.
- Route path and placement of the action button in `profile_trail_screen.dart`'s AppBar.
- Own-profile local/unsynced trails: default assumption is network-only, reusing map_screen's offline sheet state.
- Exact family key shape (record of authorId + filterId vs. a small value class).

### Deferred / OUT OF SCOPE
100-result cap, Meilisearch `maxTotalHits`, line/heatmap tile pipeline, any web or Go change beyond the
one endpoint.
</user_constraints>

---

## Summary

Every mechanism this task needs already exists in the repo; nothing has to be invented. `fitBounds` is a
real method on `ml.MapController` with two working precedents. `trailFilterProvider` is already a
riverpod-generated `keepAlive` family, so the exact conversion shape for the two map providers is
demonstrable from its own `.g.dart`. The bounding-box endpoint change is ~10 lines. The only genuinely
novel decisions are (a) how the author clause reaches the two search providers' `_executeSearch`, and
(b) what stops `keepAlive` family instances accumulating per visited author.

**Primary recommendation:** two **named** family params — `build({String? authorId, required String
filterId})` — carrying the author clause into `_executeSearch` as an *appended filter part*, never by
mutating `TrailFilter.author` on the shared filter state. Copy `map_screen.dart` into a new screen and
extract only the two pure pieces that would actually drift (marker construction, sheet math).

---

## 1. Riverpod family conversion mechanics

**Versions** [VERIFIED: `app/pubspec.yaml:27,52,76`]: `flutter_riverpod ^3.3.1`, `riverpod_annotation ^4.0.2`,
`riverpod_generator ^4.0.3`. This is the Riverpod 3 generator (`$AsyncNotifierProvider`, `ref.$arg`), **not**
the legacy 2.x `AutoDisposeFamilyAsyncNotifier` API. Do not copy Riverpod 2 family snippets from the web.

### Exact syntax

Source change (both files, identical shape):

```dart
@Riverpod(keepAlive: true)
class MapTrailSearch extends _$MapTrailSearch {
  LngLatBounds? _lastBounds;
  Timer? _debounce;

  @override
  FutureOr<List<TrailSearchResult>> build({
    String? authorId,
    required String filterId,
  }) async {
    ref.onDispose(() => _debounce?.cancel());
    ref.listen(trailFilterProvider(filterId), (previous, next) { ... });   // was 'map'
    ...
  }
}
```

Generated shape [VERIFIED: `riverpod_generator-4.0.3/lib/src/templates/notifier.dart:30-52`]:

| Params | Generated `_$args` | Accessor |
|--------|-------------------|----------|
| 1 param | `ref.$arg as String` | `String get filterId => _$args;` |
| 2+ positional | record | `_$args.$1`, `_$args.$2` |
| 2+ named | record with named fields | `_$args.authorId`, `_$args.filterId` |

So with named params the call sites become:

```dart
ref.watch(mapTrailSearchProvider(authorId: null, filterId: 'map'))
ref.read(mapTrailSearchProvider(authorId: null, filterId: 'map').notifier).searchInBounds(bounds)
ref.read(mapClusterSearchProvider(authorId: actorId, filterId: 'profile_trail_$handle').notifier)
    .searchInBounds(bounds, zoom)
```

**Use named, not positional.** `(String?, String)` positional records make `(null, 'map')` vs `('map', null)`
a silent runtime miskey; named record fields make it a compile error. Family identity is `argument ==`
[VERIFIED: `trail_filter_provider.g.dart:43-50`] and Dart records have structural equality, so
`(authorId: null, filterId: 'map')` from two call sites resolves to the same instance.

**Do not give the params default values.** riverpod_generator's family parameter handling has no
default-value support in the templates above; be explicit at both call sites.

### Hazards (project memory + repo evidence)

- **No `late final` fields assigned in `build()`.** Riverpod keeps one Notifier instance across rebuilds and
  only re-runs `build()`; a `late final` assigned there throws `LateInitializationError` on the second pass.
  This exact bug is documented in-repo at `trail_filter_provider.dart:61-83` and
  `account_scope_invalidation.dart:67-71`. The generated `late final _$args` is fine (framework-owned, the
  arg never changes for a given instance) — the rule applies to *your* fields. The existing `_lastBounds`,
  `_lastZoom`, `_debounce` are plain nullable fields; leave them as-is. One instance per family key means
  each key gets its own debounce timer, which is what we want.
- **`.value`, never `.valueOrNull`,** for nullable AsyncValue reads. `map_screen.dart:196` and `:358` already
  do this; keep it.
- **`@Riverpod(keepAlive: true)` + family = permanent instance per key.** `isAutoDispose: false` is baked
  into the generated family [VERIFIED: `trail_filter_provider.g.dart:24,71`], so every distinct
  `(authorId, filterId)` visited creates a Notifier that never disposes. Visiting five profiles' maps leaves
  five cached `List<TrailSearchResult>` (≤100 each) plus five GeoJSON `FeatureCollection`s resident.

  **Cannot be fixed by flipping to autoDispose:** `/map` sits inside a plain `ShellRoute`
  [VERIFIED: `router_provider.dart:142-188`], not a `StatefulShellRoute`, so `MapScreen` **unmounts** on
  every bottom-nav tab switch. `keepAlive` is load-bearing for the global key.

  **Recommended disposal strategy:** the profile map screen invalidates its own two family instances in
  `dispose()`:
  ```dart
  @override
  void dispose() {
    ref.invalidate(mapTrailSearchProvider(authorId: _authorId, filterId: _filterId));
    ref.invalidate(mapClusterSearchProvider(authorId: _authorId, filterId: _filterId));
    super.dispose();
  }
  ```
  `ref.invalidate` on a keepAlive instance with no remaining listeners disposes the Notifier and drops the
  family entry. This also satisfies locked decision 3's "opens at the bbox fit every time". Confidence
  MEDIUM on `ref` usage inside `dispose()` — if it misbehaves, the equivalent safe fallback is to invalidate
  in `initState()` instead, which still guarantees a fresh open and bounds the leak to one stale instance
  per author rather than zero.

- **`accountScopedProviders` needs no functional change but its doc comment does.**
  `account_scope_invalidation.dart:77-78` lists both providers; `ProviderOrFamily` accepts a family and
  family-level invalidate hits every instance (this is exactly why `trailFilterProvider` is already in the
  list). But the comment at `:61-65` — *"Every other entry is a plain provider"* — becomes false. Update it.

### Where the author clause enters the search

`TrailFilter.toFilterText()` emits `author = {id}` only when `author != null` **and** `actor != null`
[VERIFIED: `app/lib/models/trail.dart:306-310`]. Two options:

- ❌ **Set `TrailFilter.author` on the shared `profile_trail_{handle}` filter state.** Invisible to the list
  screen (its `_fetchPage` calls `toFilterText()` with no `actor`, so the author branch is skipped —
  `profile_trails_provider.dart:293`), but it silently changes the visibility logic at
  `trail.dart:323-329` and would show up in any `_countActiveFilters`-style count. Rejected.
- ✅ **Append the clause in `_executeSearch` from the family key.** Both providers already build a
  `filterText` and hand it to the server as a list element / string:
  - `map_trail_search_provider.dart:96-99` — `'filter': ['_geoBoundingBox(...)', if (filterText.isNotEmpty) filterText]`
    → add `if (authorId != null) 'author = $authorId'` as a third element (Meili ANDs array elements).
  - `map_cluster_search_provider.dart:117` — `'filterText': filterText` is a single **string**; the server
    puts it in an array alongside the geo filter [VERIFIED: `web/src/routes/api/v1/search/trails/cluster/+server.ts:37-39`].
    So append with ` AND `: `filterText.isEmpty ? 'author = $authorId' : '$filterText AND author = $authorId'`.

  Security backstop: the Meilisearch tenant token already scopes `trails` to `public = true OR author = me OR
  shares = me` (CONTEXT decision 1), so this clause is narrowing only, never authorizing.

  Visibility interaction is already correct for a foreign profile: with `author == null` on the filter and
  `showPublic/showPrivate` both true, `toFilterText` emits `(public = TRUE OR author = $me)`
  (`trail.dart:320-326`) — ANDed with `author = $profileActor`, a foreign profile's private trails are
  unreachable and your own profile's private trails appear. Matches CONTEXT's stated intent.

---

## 2. Every call site that must change

### `app/lib/routes/map_screen.dart` — 15 sites (all become `(authorId: null, filterId: 'map')`)

| Line | Expression |
|------|-----------|
| 133 | `ref.read(mapClusterSearchProvider.notifier).searchInBounds(bounds, zoom)` (GPS-chase) |
| 135 | `ref.read(mapTrailSearchProvider.notifier).searchInBounds(bounds)` |
| 159 | `mapClusterSearchProvider.notifier` (didUpdateWidget) |
| 161 | `mapTrailSearchProvider.notifier` |
| 196 | `ref.read(mapTrailSearchProvider).value ?? []` (`_selectTrail` metadata lookup) |
| 322 | `ref.listen(mapClusterSearchProvider, ...)` (updateClusterSource) |
| 357 | `ref.watch(mapTrailSearchProvider)` (sheet list) |
| 365 | `ref.watch(mapClusterSearchProvider)` (marker source) |
| 460 | `ref.read(mapClusterSearchProvider).value` (onStyleLoaded seed) |
| 476 / 479 | initial-search pair in `onStyleLoaded` |
| 515 / 518 | cluster tap-to-zoom pair in `onMapEvent` |
| 666 / 669 | "search this area" pair |

`trailFilterProvider('map')` at `map_screen.dart:346` and `:350` is **unchanged** — map_screen keeps `'map'`.

### `app/lib/provider/trail/map_trail_search_provider.dart`
- `:20` `build()` signature → `build({String? authorId, required String filterId})`
- `:23` `ref.listen(trailFilterProvider('map'), ...)` → `trailFilterProvider(filterId)`
- `:70` `ref.read(trailFilterProvider('map').future)` → `trailFilterProvider(filterId).future`
- `:96-99` filter array → append `author = $authorId`

### `app/lib/provider/trail/map_cluster_search_provider.dart`
- `:25` `build()` signature
- `:28` `ref.listen(trailFilterProvider('map'), ...)` → `trailFilterProvider(filterId)`
- `:94` `ref.read(trailFilterProvider('map').future)` → `trailFilterProvider(filterId).future`
- `:117` `'filterText'` → author clause appended

### `app/lib/provider/account_scope_invalidation.dart`
- `:61-65` doc comment is now wrong (see above). Entries at `:77-78` still compile and still work.

### `app/test/provider/trail/map_search_deletion_test.dart` — **will not compile**
`.notifier` / `.future` on a bare family is a compile error. Sites: `:182, :183, :187, :195, :206, :208,
:213, :221, :239, :240, :245`. All need `(authorId: null, filterId: 'map')`. The test's stub also serves
`/trail/filter` for the real `trailFilterProvider('map')` (`:75`) — that stays valid.

**Not affected:** `lib/provider/trail/trail_search_provider.dart:52` watches `trailFilterProvider('map')`
but is a different provider entirely. Every other `trailFilterProvider(...)` site (`trail_quick_filter_bar.dart`,
`trail_filter_screen.dart`, `trail_sort_screen.dart`, `settings_*_screen.dart`, `library_screen.dart`,
`profile_trails_provider.dart`) is unrelated.

---

## 3. Reproducing map_screen's map: extract vs. copy

### Already reusable — no work needed

| Piece | Where | Note |
|-------|-------|------|
| Style loading, live theme swap, offline branch, `onStyleLoaded`-before-`onMapCreated` buffering | `components/base/trail_collection_map.dart:20-80` | `TrailCollectionMap` is already the shared map shell. It even buffers `_pendingStyle` precisely so a caller can `fitBounds` from `onStyleLoaded` (`:54-59`) — exactly what the bbox fit needs. |
| Cluster layers | `components/map/cluster_layer.dart:20,93` | `addClusterLayers(style, geojson)` / `updateClusterSource(style, geojson)`, single source id `cluster-trails`. Reusable verbatim. |
| Selected-trail polyline | `ml.PolylineLayer` + `kTrailRouteColor` from `components/map/trail_layer.dart` | verbatim |
| Selected-trail card | `components/trail/trail_list_item.dart` | verbatim |
| Filter bar | `components/trail/trail_quick_filter_bar.dart` (takes `filterId`) | verbatim |
| Location search bar | `route_planner_screen.dart:378-405` `_buildSearchBar` + `:411-419` `_openLocationSearch` | ~35 lines, pushes `/location-search`, awaits `LocationSearchResult`, `animateCamera(zoom: 13)` |

### Genuinely map_screen-specific — do NOT try to parameterize

- GPS-chase in `initState` (`:109-138`) and the `fallbackCenter`/`fallbackZoom` chain (`:305-318`) — replaced
  entirely by the bbox fit.
- `mapCameraProvider` read at `:294` and write at `:540-547` — **must be dropped** (locked decision 3).
- The `/search` fake search bar (`:851-883`) and the filter/sort `ActionChip` row + `_countActiveFilters`
  (`:885-947`, `:1040-1061`) — replaced by AppBar + `TrailQuickFilterBar`.
- Every `kBottomNavigationBarHeight` offset (`:255, :585, :604+, :736, :955, :981`) and `sheetMinSize`'s
  `(56 + kBottomNavigationBarHeight + 48)` (`:297-299`): `/map` is inside the bottom-nav `ShellRoute`
  (`router_provider.dart:142-188`); `/profile/:handle` is a **top-level** route outside it
  (`router_provider.dart:441-477`). **The profile map has no bottom nav.** Every one of those offsets is
  wrong there and must be re-derived, and the top inset changes too (transparent AppBar + filter bar
  instead of search bar + chips).

### Recommendation: **copy the screen, extract two pure helpers**

At 1063 lines with ~40% of the layout differing (all the offsets above, plus the whole top overlay and the
whole camera-init story), a shared widget would need roughly ten boolean/nullable flags and would make both
screens harder to read than two files. Copy `map_screen.dart` → `profile_trail_map_screen.dart`.

But extract the two pieces where silent drift is a real bug, both pure and unit-testable:

1. **`buildUnclusteredTrailMarkers(...)`** — the 65-line loop at `map_screen.dart:370-435` that walks the
   GeoJSON features, filters `point_count == 1`, `firstWhereOrNull`s the trail by untrusted id, resolves
   category/subcategory, and builds the `ml.Marker`. This is where a copy would rot (the `is_large` TODO at
   `:378-380`, the untrusted-id guard at `:384-386`). Suggested home: `lib/components/map/trail_markers.dart`.
2. **Sheet math** — `_sheetHeaderOpacity` (`:1007-1013`) and `_getDynamicPadding` (`:1015-1038`), both pure
   functions of `(currentSize, sheetMinSize, sheetMediumsize, sheetMaxSize)`. Suggested home:
   `lib/util/map/sheet_metrics.dart`.

Everything else — `featuresAtPoint(layerIds: ['clusters'])` tap-to-zoom (`:498-521`), `_selectTrail`'s
fetch-then-fit (`:195-213`), the `trailDeletionsProvider` listener (`:338-344`), `_retryOnline`'s
controller-nulling (`:219-234`), `_buildSheetOfflineState` (`:243-290`) — copy, adjusting only the offsets.

---

## 4. Bounding-box endpoint change

### Server: `web/src/routes/api/v1/trail/bounding-box/+server.ts`

Handle→actor resolution to copy verbatim from `profile/[handle]/trails/+server.ts:57`:

```ts
import { getActorResponseForHandle } from '$lib/util/activitypub_server_util';
const { actor } = await getActorResponseForHandle(event, handle);
```

The filter build. `withTrailPreferenceMeiliFilter(event, filter)` takes
`MeiliFilter = string | string[] | undefined` and returns the same union
[VERIFIED: `web/src/lib/server/category_preference_filter.ts:6,99-129`]. It flattens whatever you pass via
`meiliFilterParts` (`:19-31`), appends hidden-category/subcategory exclusions, and returns the array —
Meilisearch ANDs array elements. So the minimal diff at line 45 is:

```ts
const handle = event.url.searchParams.get('handle');
let baseFilter: string | string[] | undefined = undefined;
if (handle) {
    const { actor } = await getActorResponseForHandle(event, handle);
    if (!actor.is_local) {
        return json({ min_lat: 0, max_lat: 0, min_lon: 0, max_lon: 0, has_trails: false });
    }
    baseFilter = [`author = ${actor.id}`];
}
const filter = await withTrailPreferenceMeiliFilter(event, baseFilter);
```

`filter` already flows into all four `multiSearch` queries (`:50,59,68,77`) — no other change. The
`isIdOnlyDetailQuery` short-circuit at `:104` does not trigger for an `author =` clause.

**Open decision — remote (federated) actors.** `profile/[handle]/trails` proxies to the origin instance for
`!actor.is_local` (`:71-89`). CONTEXT decision 1 only specifies the local case. Proxying the bbox would
require the *remote* instance to already support the `handle` param, which older instances won't.
Recommendation above: return `has_trails: false` for a non-local actor and let the client skip the fit.
Cheap, honest, and forward-compatible with adding a proxy later. Flag for the planner.

Unauthenticated callers already short-circuit to `has_trails: false` at `:35-43` — unchanged.

### Dart: `TrailBoundingBox` has no call site — CONFIRMED

`grep -rn TrailBoundingBox app/lib app/test` returns **only** `lib/models/trail.dart:427-434` and generated
`trail.freezed.dart` matches. Zero call sites, zero tests. The web TS side is the only consumer
(`web/src/lib/stores/trail_store.ts:632`).

The Dart model is **not usable as-is** — three defects:

```dart
@freezed
abstract class TrailBoundingBox with _$TrailBoundingBox {
  const factory TrailBoundingBox({
    required double maxLat,   // server sends "max_lat" — no @JsonKey
    required double minLat,
    required double maxLon,
    required double minLon,
  }) = _TrailBoundingBox;     // no has_trails field
}                             // no fromJson factory → no _$TrailBoundingBoxFromJson generated
```

Server payload is snake_case with a fifth field [VERIFIED: `+server.ts:92-107`,
`web/src/lib/models/trail.ts:317-323`]: `max_lat, min_lat, max_lon, min_lon, has_trails?`. Fix:

```dart
@freezed
abstract class TrailBoundingBox with _$TrailBoundingBox {
  const factory TrailBoundingBox({
    @JsonKey(name: 'max_lat') required double maxLat,
    @JsonKey(name: 'min_lat') required double minLat,
    @JsonKey(name: 'max_lon') required double maxLon,
    @JsonKey(name: 'min_lon') required double minLon,
    @JsonKey(name: 'has_trails') @Default(false) bool hasTrails,
  }) = _TrailBoundingBox;

  factory TrailBoundingBox.fromJson(Map<String, dynamic> json) =>
      _$TrailBoundingBoxFromJson(json);
}
```

`trail.dart` already has `part 'trail.g.dart'` (proven by `TrailFilterValues.fromJson` at `:422-423`), so no
new part directive. Note the server emits JSON **numbers**; Meili can return an int for a whole-degree
value, so `required double` risks a `TypeError` — `json_serializable` generates `(json['max_lat'] as num).toDouble()`
for a `double` field, so this is safe. [VERIFIED: json_serializable num→double coercion; matches
`TrailFilterValues`' existing `required double` fields fed by the same kind of endpoint.]

### Dart: handle → actorId

`profileProvider(handle)` is an autoDispose family returning a full `Actor` with `id` and `isLocal`
[VERIFIED: `app/lib/provider/profile/profile_provider.dart:24-31`; `app/lib/models/actor.dart:11-31`]. Two
paths for the planner's discretion:

- **`ref.watch(profileProvider(handle))`** on the map screen — one extra `GET /profile/{handle}` on open
  (usually already warm from the profile screen that navigated here), no route-contract change. Simplest.
- **Pass `actorId` through the route** as `extra` from `profile_trail_screen.dart` — zero extra request but
  breaks on deep-link/cold-start into the URL.

Recommend the first, with the second as an `extra`-supplied fast path if the planner wants it.
`ActorSearchResult` (which `TrailFilter.author` wants) is **not** needed under the recommendation in §1 —
only the raw id string, which goes straight into the family key.

---

## 5. Camera fit mechanics

`ml.MapController.fitBounds` exists in `maplibre: 0.3.5` [VERIFIED: `app/pubspec.yaml:42`] with two live
precedents:

```dart
_controller?.fitBounds(
  bounds: ml.LngLatBounds(
    longitudeEast: maxLon, longitudeWest: minLon,
    latitudeNorth: maxLat, latitudeSouth: minLat,
  ),
  padding: const EdgeInsets.fromLTRB(40, 56, 40, 248),
  nativeDuration: const Duration(milliseconds: 750),
);
```
[`list_detail_map_screen.dart:41-46, 59-63, 165-171, 242-247`; also `map_screen.dart:205-209` using
`ml.LngLatBounds.fromPoints(polyline)`.]

There is **no** center+zoom computation to write. Three hard-won rules from the precedents:

1. **Never `Duration.zero`** — the Android native binding passes it to `animateCamera` as null and throws.
   Use `Duration(milliseconds: 1)` for an instant fit. [VERIFIED: `list_detail_map_screen.dart:168-170`,
   `route_planner_screen.dart:256-257`.]
2. **Guard the zero-area case.** `route_planner_screen.dart:240-243` is the canonical check:
   ```dart
   final hasExtent = bounds.latitudeNorth != bounds.latitudeSouth ||
                     bounds.longitudeEast != bounds.longitudeWest;
   if (!hasExtent) return;
   ```
   **A profile with one trail is NOT zero-area** — the bbox is that trail's own min/max, which has real
   extent (migration `1778583800_persist_trail_bounds.go` upgraded the index to true per-trail bounds, per
   CONTEXT). Zero-area only happens for a single trail with a single recorded point. Handle it by
   `animateCamera(center: Geographic(lat: minLat, lon: minLon), zoom: 13)` — do not call `fitBounds`.
3. **Fit before searching.** `map_screen.dart:469-482` runs the initial bounds search inside `onStyleLoaded`
   from `getVisibleRegion()`. On the profile map the bbox is async, so the order must be:
   bbox resolves → `await fitBounds(...)` → `getVisibleRegion()` → `searchInBounds`. Every existing
   camera-move-then-search site already uses `.then((_) { ... searchInBounds ... })`
   (`:120-136`, `:148-162`, `:504-520`) — copy that shape. If `onStyleLoaded` fires before the bbox
   resolves, drive the fit from a `ref.listen` on the bbox provider instead of from `onStyleLoaded`.

**`has_trails == false`:** skip the fit entirely (leave `initCenter`/`initZoom` — reasonable default: the
user's settings location, same fallback map_screen builds at `:305-314`) and **still run the bounds search**
so the sheet renders its real empty state (`no_trails_found`, `map_screen.dart:804-833`) rather than a
permanent spinner.

**Padding:** the profile map is full-bleed under a transparent AppBar with the sheet docked at
`sheetMediumsize`. Fit padding should be roughly
`EdgeInsets.fromLTRB(32, MediaQuery.paddingOf(context).top + kToolbarHeight + 16 + <filterBarHeight>, 32,
MediaQuery.of(context).size.height * sheetMediumsize + 16)` — the top half is exactly
`route_planner_screen.dart:249-254`'s formula. Read `MediaQuery` **before** any `await`.

---

## 6. Pitfalls before planning

1. **build_runner is mandatory and touches three generated files.**
   `dart run build_runner build --delete-conflicting-outputs` from `app/`. Regenerates
   `map_trail_search_provider.g.dart`, `map_cluster_search_provider.g.dart` (family conversion) and
   `trail.g.dart` + `trail.freezed.dart` (the `TrailBoundingBox` fromJson/@JsonKey change), plus a new
   `.g.dart` for the bbox provider. Do this **before** touching `map_screen.dart`, otherwise the 15 call
   sites there show phantom errors.
2. **`test/provider/trail/map_search_deletion_test.dart` is a hard compile break** (11 sites, §2). It is the
   only test touching these providers. Budget a task for it.
3. **No existing test covers `map_screen.dart` itself** (`app/test/routes/` has none) — so a regression in
   the 15 rewritten call sites will not be caught by CI. Manual verification on device is the only gate.
   Repo memory: verify on device over ADB rather than reasoning from symptoms.
4. **The cluster provider's server contract differs from the trail provider's.** `/search/trails` takes
   `options.filter` as an **array**; `/search/trails/cluster` takes `filterText` as a **string**
   (`cluster/+server.ts:16,37-39`). The author clause must be appended differently in each (§1). Getting this
   wrong yields author-scoped pins with unscoped sheet contents — precisely the failure CONTEXT decision 2
   exists to prevent.
5. **`_selectTrail` depends on the two providers agreeing.** `map_screen.dart:196` resolves tapped-trail
   metadata by `firstWhereOrNull` over `mapTrailSearchProvider`'s results using the id stamped on the
   *cluster* feature. If the two family keys ever diverge, taps on visible pins silently select nothing.
   Both keys must be built from one source in the screen (a single `_authorId` / `_filterId` pair of fields).
6. **`profile_trail_screen.dart`'s AppBar currently has `leading` + `title` and no `actions:`**
   (`:83-86`) — the action button is a clean addition. Route placement: `/profile/:handle/trails/map` as a
   nested `routes:` child of the existing `trails` GoRoute (`router_provider.dart:448-454`) mirrors the
   `/list/:id/map` precedent at `:431-439` exactly.
7. **Own-profile unsynced trails.** `profile_trails_provider.dart:202-223` merges local ObjectBox rows into
   the list. The map path has no such merge and unsynced trails are not in Meilisearch, so they are invisible
   on the map. CONTEXT explicitly permits this default; if the planner keeps it, the profile map's trail
   count will differ from the list's on the hiker's own profile while a sync is pending. Worth a note in the
   plan, not necessarily a fix.
8. **Offline.** `mapTrailSearchProvider` is keepAlive, so a return visit while offline replays stale results
   with no error set. `map_screen.dart:716-724` documents this and gates the sheet on connectivity alone,
   not `hasError`. Copy that gating verbatim — the same trap applies per-family-key.

---

## Assumptions Log

| # | Claim | Section | Risk if wrong |
|---|-------|---------|---------------|
| A1 | `ref.invalidate(...)` inside `ConsumerState.dispose()` is safe in flutter_riverpod 3.3.1 | §1 disposal | Fallback stated inline (invalidate in `initState`); low risk |
| A2 | json_serializable coerces a JSON int into a `double` field via `(x as num).toDouble()` | §4 | A whole-degree bbox value would throw `TypeError`; mitigated by testing against a real response |
| A3 | Returning `has_trails: false` for a non-local actor is acceptable v1 behaviour | §4 | Federated profiles open without a fit; flagged as an open decision for the planner |

## Sources

All findings are direct reads of this repository at the cited `file:line`, plus
`~/.pub-cache/hosted/pub.dev/riverpod_generator-4.0.3/lib/src/templates/notifier.dart:30-52` for the family
code-gen shape. No external/web sources were needed or used.

**Research date:** 2026-08-05
**Valid until:** stable — pinned to repo state at commit `66bbab3e`
</content>
</invoke>
