---
phase: quick-260809-vir
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - web/src/routes/api/v1/regions/[id]/geometry/+server.ts
  - web/src/routes/api/v1/regions/[id]/geometry/server.test.ts
  - app/lib/models/region_geometry.dart
  - app/lib/provider/region/region_geometry_provider.dart
  - app/lib/i18n/app_en.arb
  - app/lib/routes/settings_offline_regions_map_screen.dart
  - app/lib/provider/router_provider.dart
  - app/lib/routes/settings_offline_regions_screen.dart
autonomous: true
requirements: [VIR-01, VIR-02, VIR-03, VIR-04, VIR-05, VIR-06, VIR-07]

must_haves:
  truths:
    - "GET /api/v1/regions/{path}/geometry returns {path, polygon, bbox} for a cached region_geometry row"
    - "The same route returns 404 when no cached row exists, and never issues an outbound request to the Go /regions/{id}/geometry endpoint"
    - "Tapping the map icon on a region row opens a full-screen map already fitted to that region's bbox, with no network wait"
    - "The region's boundary renders as a #1055c9 fill at 0.18 opacity plus a 2px #1055c9 line, identical in light and dark themes"
    - "The polygon renders whether the geometry fetch resolves before or after the map style loads"
    - "A failed geometry fetch shows a toast and leaves the map usable with no polygon — no spinner, no error screen"
    - "The transparent app bar's back button returns to the Offline Regions screen"
  artifacts:
    - path: "web/src/routes/api/v1/regions/[id]/geometry/+server.ts"
      provides: "Authenticated cached-row read of the region_geometry collection"
      exports: ["GET"]
      contains: "region_geometry"
    - path: "web/src/routes/api/v1/regions/[id]/geometry/server.test.ts"
      provides: "vitest coverage for the happy path, the id allow-list, and the 404 path"
      min_lines: 60
    - path: "app/lib/models/region_geometry.dart"
      provides: "freezed + json_serializable RegionGeometry model"
      contains: "RegionGeometry"
    - path: "app/lib/provider/region/region_geometry_provider.dart"
      provides: "riverpod_annotation provider family keyed by region path"
      contains: "regionGeometry"
    - path: "app/lib/routes/settings_offline_regions_map_screen.dart"
      provides: "Full-screen region boundary map with transparent app bar"
      min_lines: 120
      contains: "TrailCollectionMap"
  key_links:
    - from: "app/lib/routes/settings_offline_regions_screen.dart"
      to: "/settings/region/map?path="
      via: "context.push with an encoded region path query param"
      pattern: "settings/region/map\\?path="
    - from: "app/lib/provider/router_provider.dart"
      to: "SettingsOfflineRegionsMapScreen"
      via: "state.uri.queryParameters['path']"
      pattern: "queryParameters\\['path'\\]"
    - from: "app/lib/provider/region/region_geometry_provider.dart"
      to: "/regions/{path}/geometry"
      via: "dio api.get on apiProvider"
      pattern: "/regions/\\$\\w+/geometry"
    - from: "app/lib/routes/settings_offline_regions_map_screen.dart"
      to: "regionGeometryProvider"
      via: "ref.listen in build + _maybeDrawPolygon"
      pattern: "regionGeometryProvider"
---

<objective>
Add a full-screen map screen that draws a downloadable region's boundary polygon, backed by a new
SvelteKit API route that reads the already-cached `region_geometry` PocketBase rows directly.

Purpose: the Offline Regions settings screen already renders a map icon per region, but it pushes a
route that does not exist. Showing the actual boundary is what makes "which region am I about to
download" answerable.

Output:
- `web/src/routes/api/v1/regions/[id]/geometry/+server.ts` + its vitest suite
- `app/lib/models/region_geometry.dart` and `app/lib/provider/region/region_geometry_provider.dart`
  (plus their generated `.freezed.dart` / `.g.dart` siblings)
- `app/lib/routes/settings_offline_regions_map_screen.dart`, registered at `/settings/region/map`
- Two new localized strings in `app/lib/i18n/app_en.arb`
</objective>

<execution_context>
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/workflows/execute-plan.md
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@CLAUDE.md

Web reference (route style, id allow-list, error mapping, test style):
@web/src/routes/api/v1/regions/+server.ts
@web/src/routes/api/v1/regions/[id]/download/+server.ts
@web/src/routes/api/v1/regions/[id]/download/server.test.ts
@web/src/lib/util/api_util.ts

Backend reference (do NOT modify — read to understand why the new route must not proxy):
@db/routes/regions_geometry_get.go
@db/migrations/1786307618_updated_region_geometry.go
@db/migrations/1785000000_create_regions_collection.go
@db/routes/regions_ext/regions_ui.html

Flutter reference (model, provider, map host, camera, imperative layers, screen shell):
@app/lib/models/region_catalog_entry.dart
@app/lib/provider/region/region_provider.dart
@app/lib/provider/api_provider.dart
@app/lib/entities/region_entity.dart
@app/lib/util/region/file_path.dart
@app/lib/components/base/trail_collection_map.dart
@app/lib/routes/list_detail_map_screen.dart
@app/lib/routes/profile_trail_map_screen.dart
@app/lib/components/map/route_segment_layer.dart
@app/lib/components/map/cluster_layer.dart
@app/lib/provider/toast_provider.dart
@app/lib/provider/router_provider.dart
@app/lib/routes/settings_offline_regions_screen.dart
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: SvelteKit region geometry route (cached-row read) + vitest</name>
  <files>web/src/routes/api/v1/regions/[id]/geometry/+server.ts, web/src/routes/api/v1/regions/[id]/geometry/server.test.ts</files>
  <behavior>
    - Test 1 (happy path): `params.id = 'canada.alberta.south'` and a mocked collection read returning
      a record with `path` / `polygon` / `bbox`. The response is 200 and its JSON body is exactly
      `{ path, polygon, bbox }` from the record.
    - Test 2 (param binding): the same happy-path call asserts `locals.pb.filter` was called with the
      literal expression `'path = {:path}'` and the params object `{ path: 'canada.alberta.south' }` —
      proving the path is bound, not concatenated.
    - Test 3 (traversal): `params.id = 'canada..south'` returns 400 and `locals.pb.collection` is
      never called.
    - Test 4 (allow-list): `params.id = 'Canada/Alberta'` returns 400 and `locals.pb.collection` is
      never called.
    - Test 5 (no cached row): the mocked `getFirstListItem` rejects with a real
      `ClientResponseError` constructed as `new ClientResponseError({ status: 404, response: {} })`
      (imported from `pocketbase`); the response status is 404.
    - Test 6 (never proxies upstream): across every case above, `event.fetch` is never called.
  </behavior>
  <action>
Create the route directory `web/src/routes/api/v1/regions/[id]/geometry/` and write `+server.ts` in
it (VIR-01).

Copy the `RegionIdSchema` zod object verbatim from the sibling
`web/src/routes/api/v1/regions/[id]/download/+server.ts` — a `z.object` whose `id` is a string with
the regex allowing a leading alphanumeric followed by lowercase alphanumerics plus underscore, dot,
apostrophe and hyphen, refined to reject any value containing a double dot. The `[id]` segment here
is the region's materialized `path` (for example `canada.alberta.south`), exactly as in the two
`download` siblings — it is NOT a PocketBase record id, so do not treat it as one.

Export an async `GET(event: RequestEvent)` that:
1. Destructures `const { id } = RegionIdSchema.parse(event.params)`.
2. Reads the cached row directly from PocketBase with the caller's own client:
   `event.locals.pb.collection('region_geometry').getFirstListItem(...)`, where the filter argument is
   built with `event.locals.pb.filter('path = {:path}', { path: id })`. Use `pb.filter` param binding,
   never template-string concatenation — the established precedent is
   `web/src/routes/api/v1/user-category-preference/+server.ts:36`.
3. Returns `json({ path: record.path, polygon: record.polygon, bbox: record.bbox })`. The
   `polygon` and `bbox` fields are PocketBase `JSONField`s (see
   `db/migrations/1785000000_create_regions_collection.go:144-149`) so the SDK already hands them
   back parsed — do not re-parse or re-stringify them.
4. Wraps everything in `try { ... } catch (e) { return handleError(e) }` with `handleError` imported
   from `$lib/util/api_util`. No extra 404 branch is needed: `getFirstListItem` throws a PocketBase
   `ClientResponseError` with `status: 404` when no row matches, and `handleError`
   (`web/src/lib/util/api_util.ts:164`) forwards that status verbatim. A zod failure likewise maps to
   400 through the same helper.

HARD CONSTRAINT: this route must never reach the Go backend. Do NOT call `event.locals.pb.send(...)`
and do NOT call `event.fetch(...)`. The Go route `GET /regions/{id}/geometry`
(`db/routes/regions_geometry_get.go`) is deliberately bound to `apis.RequireSuperuserAuth()` because
it performs an outbound CoMaps fetch on cache miss (decision D-13 — open proxy / upstream rate-limit
abuse). This new route is a pure cached-row read: no row means 404, never an upstream fetch. Add a
comment in the file stating this so a future reader does not "fix" it into a proxy.

No Go, migration, or collection-rule changes are needed or wanted: migration
`db/migrations/1786307618_updated_region_geometry.go` already set the collection's `listRule` and
`viewRule` to require an authenticated user, so any signed-in user can read it and an anonymous
caller gets nothing.

Add a swagger JSDoc block above `GET`, matching the style of
`web/src/routes/api/v1/regions/+server.ts` and `web/src/routes/api/v1/regions/[id]/download/+server.ts`:
`@swagger`, the `/api/v1/regions/{id}/geometry` path, `get:`, a `summary`, a `description` that states
this reads the cached `region_geometry` row and never triggers an upstream fetch (a missing row is a
404), `tags: - Regions`, the `id` path parameter described as the region's materialized path, and
responses 200 (object with `path` string, `polygon` object, `bbox` array of numbers), 400, 401, 404,
500.

Then write `server.test.ts` alongside it, modelled on
`web/src/routes/api/v1/regions/[id]/download/server.test.ts`: a local `buildEvent(id, record?)`
helper returning an object cast `as unknown as RequestEvent` whose `locals.pb` carries a
`filter: vi.fn((expr, params) => ...)` returning a deterministic string and a
`collection: vi.fn(() => ({ getFirstListItem: vi.fn(...) }))`, plus a top-level `fetch: vi.fn()` that
the assertions prove is never invoked. Cover the six behaviors listed above.
  </action>
  <verify>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/web && npx vitest run src/routes/api/v1/regions</automated>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/web && grep -v "^\s*\*" "src/routes/api/v1/regions/[id]/geometry/+server.ts" | grep -c "event.fetch\|pb.send" | grep -qx 0</automated>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/web && npm run check</automated>
  </verify>
  <done>
`npx vitest run src/routes/api/v1/regions` passes with the new suite included (the existing download
suite still passes). The negative grep confirms the route body contains no `event.fetch` and no
`pb.send`. `npm run check` reports no NEW svelte-check errors attributable to the two new files
(pre-existing project-wide errors are out of scope — record them in the summary rather than fixing
them).
  </done>
</task>

<task type="auto">
  <name>Task 2: Flutter RegionGeometry model, provider family, and l10n strings</name>
  <files>app/lib/models/region_geometry.dart, app/lib/provider/region/region_geometry_provider.dart, app/lib/i18n/app_en.arb</files>
  <action>
Create `app/lib/models/region_geometry.dart` (VIR-02), following the conventions in
`app/lib/models/region_catalog_entry.dart` exactly: import `freezed_annotation`, declare the two
`part` directives for the `.freezed.dart` and `.g.dart` siblings, annotate with `@freezed`, and
declare `abstract class RegionGeometry with _$RegionGeometry` holding a `const factory` with three
required named fields plus a `fromJson` factory:
- `path` — `String`
- `polygon` — `Map<String, dynamic>`, the raw GeoJSON *geometry* object (`Polygon` or `MultiPolygon`)
  as stored in the `region_geometry.polygon` JSON column. It is NOT a Feature and NOT a
  FeatureCollection; the caller wraps it in a Feature before handing it to MapLibre.
- `bbox` — `List<double>` in `[minLon, minLat, maxLon, maxLat]` order, the same order
  `RegionEntity`'s four discrete columns and `generator.go`'s pmtiles extract arguments use.
Document the class as the response shape of `GET /api/v1/regions/{path}/geometry`. No `@JsonKey`
renames are needed — all three server keys are already snake-free.

Create `app/lib/provider/region/region_geometry_provider.dart` (VIR-02), following
`app/lib/provider/region/region_provider.dart`'s structure (imports, `part` directive,
`riverpod_annotation`). Declare a plain `@riverpod` function-style provider FAMILY keyed by the
region path — `Future<RegionGeometry> regionGeometry(Ref ref, String path)`. Its body:
1. `final validated = assertValidRegionPath(path);` importing
   `package:wanderer/util/region/file_path.dart`. This is the same defense-in-depth guard
   `app/lib/services/tile_repository_manager.dart:499` applies before building a region request URL —
   the path is never string-concatenated into a URL unvalidated.
2. `final api = ref.watch(apiProvider);` then `final response = await api.get('/regions/$validated/geometry');`.
   The dio client's `baseUrl` already ends in `/api/v1` (see `app/lib/provider/api_provider.dart`), so
   the request path carries no `/api/v1` prefix — identical to `tile_repository_manager.dart`'s
   `/regions/$validated/download`.
3. `return RegionGeometry.fromJson(response.data as Map<String, dynamic>);`
Leave it auto-dispose (do NOT add `keepAlive: true`) — one region's outline is a screen-scoped read,
and the screen is the only consumer.

Add exactly two new keys to `app/lib/i18n/app_en.arb` (VIR-07), placed next to the existing
`regions_*` block (around the `regions_offline_unavailable_*` entries) and following how those keys
were added — plain string values, no placeholders, so no `@key` metadata object is required:
- `regions_map_geometry_failed`: "Could not load region outline"
- `regions_map_back_label`: "Back to regions"
IMPORTANT: the working tree already carries uncommitted edits to `app/lib/i18n/app_en.arb` and
`app/lib/i18n/app_localizations.dart` from unrelated work. Append the new keys; never revert or
overwrite the existing modifications. Only `app_en.arb` is hand-edited — every other locale file is
left alone and the two keys land in `lib/i18n/untranslated_messages.json`, which is the established
behavior for new `regions_*` strings.

Finally run codegen in `app/`: `dart run build_runner build --delete-conflicting-outputs` (freezed +
json_serializable + riverpod_generator), then `flutter gen-l10n` to regenerate
`app/lib/i18n/app_localizations*.dart` from the updated ARB. Commit the generated files alongside the
sources, as the repo already does.
  </action>
  <verify>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/app && dart run build_runner build --delete-conflicting-outputs && flutter gen-l10n && flutter analyze</automated>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/app && ls lib/models/region_geometry.freezed.dart lib/models/region_geometry.g.dart lib/provider/region/region_geometry_provider.g.dart</automated>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/app && grep -c "regions_map_geometry_failed\|regions_map_back_label" lib/i18n/app_localizations.dart | grep -qv '^0$'</automated>
  </verify>
  <done>
build_runner emits `region_geometry.freezed.dart`, `region_geometry.g.dart` and
`region_geometry_provider.g.dart` with no errors; `flutter gen-l10n` regenerates
`app_localizations*.dart` carrying both new getters; `flutter analyze` is clean for the new files
(no new errors or warnings introduced).
  </done>
</task>

<task type="auto">
  <name>Task 3: Region boundary map screen, route registration, and call-site wiring</name>
  <files>app/lib/routes/settings_offline_regions_map_screen.dart, app/lib/provider/router_provider.dart, app/lib/routes/settings_offline_regions_screen.dart</files>
  <action>
**Screen** (VIR-03, VIR-05, VIR-06, VIR-07) — create
`app/lib/routes/settings_offline_regions_map_screen.dart` with
`class SettingsOfflineRegionsMapScreen extends ConsumerStatefulWidget` holding `final String path;`
(the region's materialized path). Model the shell on `app/lib/routes/profile_trail_map_screen.dart`
lines 517-532, minus its title/search bar:
- `Scaffold(extendBodyBehindAppBar: true, ...)`
- `AppBar(backgroundColor: Colors.transparent, elevation: 0, scrolledUnderElevation: 0, leading: IconButton(...))`
  whose only content is the back button: a `FaIcon(FontAwesomeIcons.arrowLeft, size: 18)`,
  `onPressed: () => context.pop()`, `style: IconButton.styleFrom(backgroundColor: Theme.of(context).colorScheme.surface)`,
  and `tooltip: AppLocalizations.of(context)!.regions_map_back_label` (no hardcoded English).
- `body:` a bare `TrailCollectionMap` — the trail-agnostic MapLibre host in
  `app/lib/components/base/trail_collection_map.dart`. Do NOT use `TrailMap` (single-trail,
  offline-rewrite host). Do not pass `children`, so the host's default scalebar + attribution apply,
  and do not pass `layers` — the polygon is added imperatively.

State fields: `ml.MapController? _controller;`, `ml.StyleController? _style;`, `bool _polygonAdded = false;`,
`bool _toastShown = false;`.

Callbacks, ordered defensively per `TrailCollectionMap`'s `_pendingStyle` doc comment (the native
channel can fire the style-loaded event before `onMapCreated`; the host buffers it and replays it
after the controller is set, so `onStyleLoaded` may run either synchronously-late or normally — never
assume ordering, and always null-guard):
- `onMapCreated: (controller) => _controller = controller`
- `onStyleLoaded: (style) { _style = style; _fitToBbox(); _maybeDrawPolygon(); }`

**Initial camera** (VIR-05) — `_fitToBbox()` reads the LOCALLY CACHED catalog, never the network:
`ref.read(regionListNotifierProvider).firstWhereOrNull((r) => r.path == widget.path)` (import
`package:collection/collection.dart` for `firstWhereOrNull` and
`app/lib/provider/region/region_provider.dart` for the notifier). If no row matches, return without
fitting — never crash, never fall back to a network read. Otherwise build
`ml.LngLatBounds(longitudeEast: region.maxLon, longitudeWest: region.minLon, latitudeNorth: region.maxLat, latitudeSouth: region.minLat)`
and call `_controller?.fitBounds(bounds: bounds, padding: const EdgeInsets.all(40), nativeDuration: const Duration(milliseconds: 1))`
— exactly the shape `app/lib/routes/list_detail_map_screen.dart:167-173` uses. Never
`Duration.zero`: it crashes the Android native binding.

**Polygon rendering** (VIR-06) — `Future<void> _maybeDrawPolygon()` is the single draw path, called
from BOTH `onStyleLoaded` and the geometry listener, so either resolution order works (the same
two-caller race pattern `profile_trail_map_screen.dart`'s `_maybeFitAndSearch` uses). It returns
immediately if `_polygonAdded` is true, if `_style` is null, or if
`ref.read(regionGeometryProvider(widget.path)).value` is null. Otherwise it sets `_polygonAdded = true`
and, inside a `try`/`catch` that only `debugPrint`s (mirroring `profile_trail_map_screen.dart:550-570`),
awaits in order:
1. `_style!.addSource(ml.GeoJsonSource(id: 'region-outline', data: jsonEncode({'type': 'Feature', 'geometry': geometry.polygon, 'properties': <String, Object>{}})))`
   — `GeoJsonSource.data` is a JSON-encoded String, per `route_segment_layer.dart` and
   `cluster_layer.dart`.
2. `_style!.addLayer(const ml.FillStyleLayer(id: 'region-outline-fill', sourceId: 'region-outline', paint: <String, Object>{'fill-color': '#1055c9', 'fill-opacity': 0.18}))`
3. `_style!.addLayer(const ml.LineStyleLayer(id: 'region-outline-line', sourceId: 'region-outline', paint: <String, Object>{'line-color': '#1055c9', 'line-width': 2}))`
Those paint values are ported verbatim from `db/routes/regions_ext/regions_ui.html:1298-1300` and are
identical in light and dark themes — do NOT derive them from `Theme.of(context)` or restyle them. Add
a comment saying so.

**States** (VIR-07) — no loading UI whatsoever: no spinner, no error screen, no full-screen
placeholder. The map renders immediately, already fitted. In `build()`, first compute
`final hasPath = widget.path.isNotEmpty && isValidRegionPath(widget.path);` (importing
`app/lib/util/region/file_path.dart`). Only when `hasPath` is true, register
`ref.listen<AsyncValue<RegionGeometry>>(regionGeometryProvider(widget.path), (AsyncValue<RegionGeometry>? previous, AsyncValue<RegionGeometry> next) { ... })`
— explicitly type both the generic argument and the closure params; an untyped listener param can be
inferred as `dynamic`, which skips AsyncValue extension resolution and throws at runtime (recorded
decision quick-260712-pac). In the callback: on `next.hasValue`, call `_maybeDrawPolygon()`; on
`next.hasError` when `_toastShown` is false, set `_toastShown = true` and
`ref.read(toastProvider.notifier).add(ToastMessage(type: ToastType.error, icon: FontAwesomeIcons.circleExclamation, text: AppLocalizations.of(context)!.regions_map_geometry_failed))`
following `app/lib/routes/settings_account_screen.dart:59-67`, then leave the polygon undrawn and the
map fully usable. Registering the `ref.listen` is what initializes the provider and starts the fetch;
do not add a separate `ref.watch` that would rebuild the screen on every AsyncValue transition. When
`hasPath` is false the screen renders the bare map with no fit and no geometry subscription — never
construct `regionGeometryProvider('')`.

**Routing** (VIR-04) — in `app/lib/provider/router_provider.dart`, import the new screen and add a
TOP-LEVEL `GoRoute(path: '/settings/region/map', ...)` as a sibling of the existing `/settings`
GoRoute (place it immediately after that route's closing paren, around line 280). It must NOT be
nested inside `/settings`'s `routes:` children list, because the call sites push the absolute path.
Its builder reads the region path from a QUERY param:
`final path = state.uri.queryParameters['path'] ?? '';` then
`return SettingsOfflineRegionsMapScreen(path: path);`. An absent or empty param yields an empty
string, which the screen's `hasPath` guard already handles gracefully.

**Call site** (VIR-04) — in `app/lib/routes/settings_offline_regions_screen.dart`, update the
`context.push('/settings/region/map')` at line 674 (inside `_buildActiveRow`, where a real
`RegionEntity region` is in scope) to
`context.push('/settings/region/map?path=${Uri.encodeComponent(region.path)}')`.
A grep of the file confirms this is the ONLY `context.push('/settings/region/map')` occurrence — the
skeleton placeholder block at lines 524-568 renders only text and two `ListTile`s and contains no map
`IconButton`, so there is no second call site and nothing to disable there. Re-run the grep yourself
before editing; if a second occurrence has appeared, give it the same `?path=` treatment when a real
region is in scope, or make it non-functional if it only renders fake skeleton data — never leave a
push with no `path`.

Do NOT run `flutter build`, `adb install`, or any device/emulator install step. The user builds and
installs.
  </action>
  <verify>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/app && flutter analyze</automated>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/app && flutter test</automated>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/app && grep -c "settings/region/map?path=" lib/routes/settings_offline_regions_screen.dart | grep -qx 1</automated>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/app && grep -v "^\s*//" lib/routes/settings_offline_regions_map_screen.dart | grep -c "#1055c9" | grep -qx 2</automated>
    <automated>cd /Users/christianbeutel/Documents/svelte/wanderer/app && grep -c "queryParameters\['path'\]" lib/provider/router_provider.dart | grep -qx 1</automated>
  </verify>
  <done>
`flutter analyze` is clean for the three touched files. `flutter test` shows no NEW failures (the
three pre-existing failures logged in
`.planning/phases/18-retire-flutter-map-and-the-flomp-forks/deferred-items.md` —
`feed_item_test.dart` x2, `settings_screen_test.dart` x1 — remain out of scope; note them in the
summary if they still fail). The greps confirm exactly one `?path=`-carrying push, exactly two
non-comment `#1055c9` paint values (fill + line), and one query-param read in the router.
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Flutter client -> SvelteKit `/api/v1` | Untrusted `[id]` path segment crosses here |
| SvelteKit -> PocketBase | Filter expression and auth context cross here |
| SvelteKit -> Go backend | Deliberately NOT crossed by this feature |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-vir-01 | Information Disclosure | `GET /api/v1/regions/[id]/geometry` | mitigate | Reads through `event.locals.pb` (the caller's own auth context), never an admin client. Migration `1786307618_updated_region_geometry.go` already gates the collection's `listRule`/`viewRule` on `@request.auth.id != ""`, so an anonymous caller gets nothing. No new collection rules added. |
| T-vir-02 | Tampering (filter injection) | `path` filter in `+server.ts` | mitigate | zod allow-list regex plus double-dot rejection (copied verbatim from the `download` sibling), then `event.locals.pb.filter('path = {:path}', { path: id })` param binding. Task 1 Test 2 asserts the binding; string concatenation is never used. |
| T-vir-03 | Denial of Service (open proxy / upstream rate-limit abuse) | outbound CoMaps fetch | mitigate | The route performs a cached-row read only. No `event.fetch` and no `pb.send` — enforced by a negative grep gate in Task 1's verify — so the superuser-gated Go route (`db/routes/regions_geometry_get.go`, D-13) stays unreachable from this path. A missing row is a 404, never an upstream fetch. |
| T-vir-04 | Tampering (URL injection) | `regionGeometryProvider` request path | mitigate | `assertValidRegionPath(path)` runs before the path is interpolated into `/regions/{path}/geometry`, mirroring `tile_repository_manager.dart:499`. The screen additionally refuses to construct the provider for an empty/invalid query param. |
| T-vir-05 | Tampering (deep link) | `/settings/region/map?path=` | mitigate | The route builder tolerates a missing param (empty string) and the screen's `hasPath` guard skips both the bbox fit and the geometry subscription, so a crafted deep link cannot crash the app or issue an unvalidated request. |
| T-vir-SC | Tampering | npm/pub installs | accept | No new package-manager dependencies are added by this plan (freezed, json_serializable, riverpod_generator, maplibre, dio, go_router, zod and vitest are all already in `app/pubspec.yaml` / `web/package.json`). The Package Legitimacy Gate is not triggered. |
</threat_model>

<verification>
Web:
- `cd web && npx vitest run src/routes/api/v1/regions` — new geometry suite plus the existing download suite pass.
- `cd web && npm run check` — no new svelte-check errors from the two new files.
- Negative grep: the route body contains no `event.fetch` and no `pb.send`.

Flutter:
- `cd app && dart run build_runner build --delete-conflicting-outputs` — generated siblings emitted.
- `cd app && flutter gen-l10n` — `app_localizations*.dart` carries both new getters.
- `cd app && flutter analyze` — clean for all touched files.
- `cd app && flutter test` — no new failures beyond the three known pre-existing ones.

Explicitly NOT run by the executor: `flutter build`, `adb install`, any device/emulator install step.

<human-check>
On device, after the user's own build:
1. Settings -> Offline Maps/Regions, tap the map icon on a ready region row.
2. A full-screen map opens immediately, already framed on that region — no spinner, no camera flight from a world view.
3. The region's boundary appears shortly after as a translucent blue fill with a 2px blue outline; the colors look the same in light and dark mode.
4. The back button in the top-left returns to the Offline Regions list.
5. With the device offline, the same flow still opens the fitted map and shows a "Could not load region outline" toast, with no polygon and no error screen.
</human-check>
</verification>

<success_criteria>
- `GET /api/v1/regions/{path}/geometry` returns `{ path, polygon, bbox }` for a cached row, 404 for a missing one, 400 for a malformed path, and never contacts the Go backend.
- `RegionGeometry` and `regionGeometryProvider` exist with generated code committed, and the provider validates the path before building its request URL.
- `/settings/region/map?path=<encoded>` renders a full-screen `TrailCollectionMap` with a transparent app bar and a single back button, fitted to the cached `RegionEntity` bbox with no network wait.
- The boundary polygon renders in both resolution orderings (geometry-before-style and style-before-geometry) with `#1055c9` fill at 0.18 and a 2px `#1055c9` line, theme-independent.
- A geometry failure surfaces as a toast only; the map stays usable.
- Both new user-facing strings live in `app_en.arb` and are read through `AppLocalizations`.
</success_criteria>

<output>
Create `.planning/quick/260809-vir-region-geometry-map-screen/260809-vir-SUMMARY.md` when done.
</output>
