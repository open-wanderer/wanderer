---
phase: quick-260717-seb
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - app/lib/util/reverse_geocode_util.dart
  - app/test/util/reverse_geocode_util_test.dart
  - app/lib/components/route_planner/route_anchor_list_tab.dart
autonomous: true
requirements: [QUICK-260717-seb]

must_haves:
  truths:
    - "Each route-anchor row shows a reverse-geocoded street/place title instead of the literal 'Anchor N', matching web client behavior"
    - "The title falls back to 'Anchor N' (the app's existing naming) until a location resolves or if geocoding fails"
    - "Geocoding runs one anchor at a time (sequential await, not parallel) to respect the server's 1 req/sec Nominatim rate limit"
    - "Resolved titles are cached by rounded coordinate key so an add/delete/reorder never re-fetches an already-resolved anchor"
    - "A geocoding failure is silent best-effort — it never surfaces an error to the UI and never blocks the row from rendering"
  artifacts:
    - path: "app/lib/util/reverse_geocode_util.dart"
      provides: "ReverseLocationResult + pure address parsing (label/fullLabel/country) + searchLocationReverseStructured Dio fetch, mirroring web's getReverseLocationResult/getLocationDescription"
      contains: "class ReverseLocationResult"
    - path: "app/test/util/reverse_geocode_util_test.dart"
      provides: "Unit tests for the pure address-parsing functions"
    - path: "app/lib/components/route_planner/route_anchor_list_tab.dart"
      provides: "Per-anchor reverse-geocoding wiring: local cache, sequential batch loader, resolved title with fallback"
  key_links:
    - from: "app/lib/components/route_planner/route_anchor_list_tab.dart"
      to: "app/lib/util/reverse_geocode_util.dart"
      via: "searchLocationReverseStructured(includeRoad: true)"
      pattern: "searchLocationReverseStructured"
    - from: "app/lib/util/reverse_geocode_util.dart"
      to: "/geocoding/reverse"
      via: "Dio GET with lat/lon query params"
      pattern: "/geocoding/reverse"
---

<objective>
Reverse-geocode route-planner anchors so each row in the Route Anchors tab shows a street/place name (e.g. "Bahnhofstrasse, Zürich, Switzerland") instead of the placeholder `Anchor N`, matching the web client's per-anchor behavior in `trail_anchor_list.svelte`.

Purpose: Bring the Flutter route planner to parity with the web client, where anchors are labeled by their resolved location rather than a bare index. Coordinates alone are meaningless to a planner scanning their route.

Output: A new pure-Dart reverse-geocoding utility (with tests) that ports web's `getReverseLocationResult`/`getLocationDescription`, plus the anchor-list wiring that resolves and displays titles best-effort, one anchor at a time, cached by coordinate.
</objective>

<execution_context>
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/workflows/execute-plan.md
@/Users/christianbeutel/Documents/svelte/wanderer/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md

# The web behavior being ported one-to-one (source of truth):
@web/src/lib/stores/search_store.ts
@web/src/lib/components/trail/trail_anchor_list.svelte

# Flutter files this plan touches / mirrors:
@app/lib/components/route_planner/route_anchor_list_tab.dart
@app/lib/provider/search/global_search_provider.dart
@app/lib/provider/api_provider.dart
@app/lib/models/route_anchor.dart
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: Create reverse-geocode utility mirroring web's getReverseLocationResult/getLocationDescription</name>
  <files>app/lib/util/reverse_geocode_util.dart, app/test/util/reverse_geocode_util_test.dart</files>
  <behavior>
    Pure parsing (getLocationDescription + getReverseLocationResult), the Dart analog of web's search_store.ts helpers. Write these tests FIRST, then implement to pass:
    - getLocationDescription with includeRoad:true builds comma-joined "road, city, state, country" in that exact order (road first, then city fallback chain, then state, then country).
    - getLocationDescription with includeCountry:false omits the country segment (used for `label`); default includes it (used for `fullLabel`).
    - City fallback chain: uses address['city'], else 'town', else 'hamlet', else 'village' — exactly one of them, first present wins (mirror web's else-if ladder).
    - includeRoad:false (the default) omits the road segment entirely, even when address['road'] is present.
    - Missing/absent address keys are skipped (no empty commas), e.g. an address with only {country} yields just the country.
    - getReverseLocationResult returns label WITHOUT country and fullLabel WITH country; country = address['country'] ?? ''.
    - getReverseLocationResult's label falls back to fullLabel when the country-less description is empty (e.g. address has only {country}: label == fullLabel == country).
  </behavior>
  <action>
    Create `app/lib/util/reverse_geocode_util.dart`. Port web's `search_store.ts` reverse-geocoding shape into idiomatic Dart. Do NOT extend `global_search_provider.dart`'s private `_buildLocationDescription` — it lacks the road step and the label-vs-fullLabel split this feature needs; build a dedicated set of functions here.

    Define a plain immutable class `ReverseLocationResult` with three `final String` fields `label`, `fullLabel`, `country` and a const constructor (add `==`/`hashCode` so tests can compare instances, or compare field-by-field in tests — your call).

    Implement pure `String getLocationDescription(Map&lt;String, dynamic&gt; address, {bool includeRoad = false, bool includeCountry = true})` exactly mirroring web's `getLocationDescription`: build a `List&lt;String&gt;` of parts — push `address['road']` only when `includeRoad` and road is a non-empty String; then push the first present of city/town/hamlet/village; then `address['state']`; then `address['country']` only when `includeCountry`. Join with ", ". Guard every lookup as a nullable String (values may be absent or non-String) and skip null/empty.

    Implement pure `ReverseLocationResult getReverseLocationResult(Map&lt;String, dynamic&gt; address, {bool includeRoad = false})` mirroring web: `country = address['country'] as String? ?? ''`; `label = getLocationDescription(address, includeRoad: includeRoad, includeCountry: false)`; `fullLabel = getLocationDescription(address, includeRoad: includeRoad, includeCountry: true)`; return `ReverseLocationResult(label: label.isNotEmpty ? label : fullLabel, fullLabel: fullLabel, country: country)`.

    Implement the async fetch `Future&lt;ReverseLocationResult?&gt; searchLocationReverseStructured(Dio api, double lat, double lon, {bool includeRoad = true, CancelToken? cancelToken})`: `api.get('/geocoding/reverse', queryParameters: {'lat': lat, 'lon': lon}, cancelToken: cancelToken)` (mirror the sibling `/geocoding/search` call already in `global_search_provider.dart`; `apiProvider`'s Dio is already based at `/api/v1`). Parse `response.data['features'] as List&lt;dynamic&gt;? ?? []`; if empty return null; read `features[0]['properties']['address'] as Map&lt;String, dynamic&gt;?`; if null return null; else return `getReverseLocationResult(address, includeRoad: includeRoad)`. Let DioException (including cancel) propagate to the caller — the widget layer decides how to swallow it.

    Create `app/test/util/reverse_geocode_util_test.dart` following `app/test/util/route_segment_util_test.dart`'s structure (`flutter_test`, `group`/`test`, import from `package:wanderer/util/reverse_geocode_util.dart`). Cover every case listed in &lt;behavior&gt;. Do NOT unit-test the Dio network path — there is no http mock adapter in this project's test suite; the pure parsing functions are the tested surface.
  </action>
  <verify>
    <automated>cd app && flutter test test/util/reverse_geocode_util_test.dart</automated>
  </verify>
  <done>The util file exposes ReverseLocationResult, getLocationDescription, getReverseLocationResult, and searchLocationReverseStructured; all new unit tests pass; label/fullLabel/country semantics match web's search_store.ts exactly (road-first ordering, city fallback chain, label omits country, label falls back to fullLabel when empty).</done>
</task>

<task type="auto">
  <name>Task 2: Wire resolved anchor titles into route_anchor_list_tab.dart</name>
  <files>app/lib/components/route_planner/route_anchor_list_tab.dart</files>
  <action>
    Port web's `trail_anchor_list.svelte` per-anchor location wiring into `_RouteAnchorListTabState`, keeping the resolved-title cache as LOCAL widget state (mirror web's module `Map` + `$state` record — do NOT add a title field to the immutable `RouteAnchor` model, and do NOT add a new provider).

    Add these instance fields to `_RouteAnchorListTabState`:
    - `final Map&lt;String, ReverseLocationResult&gt; _locations = {};` — resolved results keyed by rounded coordinate.
    - `final Set&lt;String&gt; _pending = {};` — keys with an in-flight request (dedupe).
    - `CancelToken? _batchToken;` — the single shared token for the current batch, aborted before starting a new batch (mirrors web's single `locationAbortController`).
    - `String? _lastAnchorSignature;` — guards against re-kicking an identical batch every build.

    Add `String _locationCacheKey(RouteAnchor a)` returning `'${a.lat.toStringAsFixed(5)},${a.lon.toStringAsFixed(5)}'` (matches web's 5-decimal `locationCacheKey`).

    Add `Future&lt;void&gt; _loadAnchorLocation(RouteAnchor anchor, CancelToken token)`: compute key; return early if `_locations.containsKey(key)` or `_pending.contains(key)`; add key to `_pending`; in a try/finally (finally removes key from `_pending`), call `searchLocationReverseStructured(ref.read(apiProvider), anchor.lat, anchor.lon, includeRoad: true, cancelToken: token)`; on a non-null result, if `mounted`, `setState(() =&gt; _locations[key] = result)`. Catch `DioException`: if `e.type == DioExceptionType.cancel` return silently (superseded); otherwise log via `debugPrint` and swallow (best-effort, never surfaces to UI). Catch any other error the same silent-log way.

    Add `Future&lt;void&gt; _loadAnchorLocations(List&lt;RouteAnchor&gt; anchors)`: cancel `_batchToken` if non-null; create a fresh `CancelToken` and assign to `_batchToken`; capture it in a local `token`; iterate anchors with an explicit `for (final anchor in anchors)` and `await _loadAnchorLocation(anchor, token)` INSIDE the loop (sequential — NOT `Future.wait`, to respect the server's 1 req/sec Nominatim limit); before each iteration, `if (token.isCancelled) return;`.

    Trigger the batch from `build()` without calling setState during build: after reading `anchors`, compute `final signature = anchors.map(_locationCacheKey).join('|');` and if `signature != _lastAnchorSignature`, set `_lastAnchorSignature = signature` and schedule `WidgetsBinding.instance.addPostFrameCallback((_) { if (mounted) _loadAnchorLocations(anchors); });`. This mirrors web's `$effect` re-running whenever the anchors array changes (add/delete/reorder); an unchanged list (same coords, same order) is skipped, and a reorder of already-resolved anchors re-runs harmlessly since every key early-returns from cache.

    Add `String? _commonAnchorCountry(List&lt;RouteAnchor&gt; anchors)` (optional cross-country nuance, mirror web's `commonAnchorCountry`): collect non-empty resolved countries for the anchors; if fewer than 2, return null; if all equal, return that country, else null.

    Add `String _anchorTitle(RouteAnchor anchor, int index, String? commonCountry)`: look up `_locations[_locationCacheKey(anchor)]`; if null, return the app's EXISTING fallback `'Anchor ${index + 1}'` (keep this literal — do NOT introduce a new l10n string); otherwise return `location.label` when `commonCountry != null && location.country == commonCountry` (cleaner, no redundant country), else `location.fullLabel`.

    In `build()`, compute `final commonCountry = _commonAnchorCountry(anchors);` and replace the ListTile's `title: Text('Anchor ${index + 1}')` with `title: Text(_anchorTitle(anchor, index, commonCountry))`. Wrap the title Text with `maxLines: 1` and `overflow: TextOverflow.ellipsis` so a long street+city+country string truncates cleanly in the row.

    Add `@override void dispose() { _batchToken?.cancel(); super.dispose(); }` to abort any in-flight batch on unmount.

    Add the needed imports: `package:dio/dio.dart` (for `CancelToken`, `DioException`, `DioExceptionType`), `package:wanderer/provider/api_provider.dart` (for `apiProvider`), `package:wanderer/util/reverse_geocode_util.dart`, and `package:wanderer/models/route_anchor.dart` if not already transitively available. Keep the existing reorder/delete behavior untouched — this task only adds the title-resolution layer.
  </action>
  <verify>
    <automated>cd app && flutter analyze lib/components/route_planner/route_anchor_list_tab.dart lib/util/reverse_geocode_util.dart && grep -q "searchLocationReverseStructured" lib/components/route_planner/route_anchor_list_tab.dart && grep -q "_anchorTitle(anchor, index" lib/components/route_planner/route_anchor_list_tab.dart</automated>
  </verify>
  <done>`flutter analyze` reports no errors on both files; the ListTile title renders `_anchorTitle(...)` (resolved location, falling back to 'Anchor N'); geocoding runs sequentially via a shared CancelToken, caches by rounded coordinate, and swallows failures silently; the in-flight batch is cancelled on dispose.</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Flutter app → `/api/v1/geocoding/reverse` | App sends anchor lat/lon (already user-derived map coordinates) to the existing SvelteKit server proxy, which rate-limits and forwards to Nominatim. No new trust boundary — a sibling of the already-used `/geocoding/search` endpoint. |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-seb-01 | Information Disclosure | Geocoding response rendered as anchor title | accept | Response is public place-name data from OSM/Nominatim; anchors are the user's own in-memory plan, never persisted or shared by this feature. |
| T-seb-02 | Denial of Service | Repeated reverse-geocode calls hammering the server's 1 req/sec limit | mitigate | Sequential per-anchor await (no `Future.wait`), coordinate-keyed cache prevents re-fetch, single shared CancelToken aborts a superseded batch before starting a new one. |
| T-seb-03 | Tampering | npm/pip/cargo installs | accept | No package installs — feature uses existing `dio` and `maplibre` deps only. |
</threat_model>

<verification>
- `cd app && flutter test test/util/reverse_geocode_util_test.dart` — all pure-parsing unit tests pass.
- `cd app && flutter analyze lib/util/reverse_geocode_util.dart lib/components/route_planner/route_anchor_list_tab.dart` — no errors.
- Manual (device/emulator, optional): open the Route Planner, drop 2-3 anchors, open the Route Anchors tab; each row's title resolves from `Anchor N` to a street/place name within a few seconds; deleting/reordering does not re-fetch already-resolved anchors.
</verification>

<success_criteria>
- Route-anchor rows display reverse-geocoded street/place titles matching the web client's ordering (road → city → state → country), with `Anchor N` as the pre-resolution/failure fallback.
- Geocoding is best-effort, sequential, coordinate-cached, and abortable; failures never surface to the UI.
- No changes to route saving/persistence, the create/edit handoff, or the `RouteAnchor` model's `id`/`lat`/`lon` fields.
- No new l10n strings introduced.
</success_criteria>

<output>
Create `.planning/quick/260717-seb-reverse-geocode-route-planner-anchors-on/260717-seb-SUMMARY.md` when done.
</output>
