# Phase 5: Cache Write + Fallback + UI - Context

**Gathered:** 2026-06-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Wire the offline navigation capability end-to-end: cache Valhalla instructions to ObjectBox when a trail is downloaded, fall back to the cached instructions when `launchNavigation` hits a `DioException`, silently re-cache after a successful online session, and show a wifi-off icon inside the maneuver banner when operating from cache. All four OFFLINE-xx requirements deliver in this phase.

</domain>

<decisions>
## Implementation Decisions

### isOffline Flag Propagation (OFFLINE-04)

- **D-01:** Add `isOffline: bool` as a new constructor parameter to `NavigationScreen` alongside the existing `response: NavigateResponse`. No new wrapper class needed.
- **D-02:** Pass both values via a Dart record as the router `extra`: `context.push('/trail/${trail.id}/navigate', extra: (response, isOffline))`. The router unpacks it as `(NavigateResponse, bool) = extra`. Typed, no new model type, consistent with Dart 3 records.
- **D-03:** `launchNavigation` passes `isOffline: false` on the online path and `isOffline: true` on the cache fallback path.

### Offline Indicator Design (OFFLINE-04)

- **D-04:** The offline indicator appears inside the existing maneuver banner widget (`_buildBanner`). No new Stack layer or Positioned widget.
- **D-05:** Use `FaIcon(FontAwesomeIcons.wifiSlash)` to match the FontAwesome icons already used throughout `NavigationScreen` (e.g., `FontAwesomeIcons.triangleExclamation`, `FontAwesomeIcons.locationCrosshairs`). The icon is shown only when `isOffline == true`.

### Cache Write at Download Time (OFFLINE-01)

- **D-06:** The cache write happens inside `TrailDownloadService.downloadTrail()`, as a sequential try/catch block after the map-tile download succeeds and before (or during) `box.put(entity)`. A Valhalla outage must never block or error the tile download.
- **D-07:** Shape source: `trail.expand?.gpx.allPoints`. If `trail.expand?.gpx` is null at download time, skip the cache write silently (best-effort). No fallback to `waypointsViaTrail` — mirrors exactly what `launchNavigation` does.
- **D-08:** Extract the downsampling logic (≤500 points, preserving first and last) from `launchNavigation` into a shared private helper — e.g., `_buildNavShape(List<WptType> points)` in `app/lib/util/gpx_util.dart`. Both `downloadTrail` and `launchNavigation` call this helper. Prevents the two paths from diverging.

### DioException Fallback (OFFLINE-02)

- **D-09:** In `launchNavigation`, catch `DioException` specifically (not a generic `catch (e)`). On catch: read `navCacheJson` from `ObjectBox` for the given trail ID. If found, decode via `NavigateResponse.fromJson(jsonDecode(...))`, validate non-empty maneuvers + shape, then push navigation with `isOffline: true`.
- **D-10:** Access `ObjectBox` store from `launchNavigation` via `ref.read(objectBoxProvider)` — the function already has `WidgetRef ref`, so no signature change is needed beyond reading the provider.
- **D-11:** If the cache read also fails (null or decode error), fall through to the existing error toast — same UX as today's network failure.

### Silent Re-cache After Online Session (OFFLINE-03)

- **D-12:** After the online POST succeeds and navigation is pushed, fire an unawaited `Future` that writes the fresh `NavigateResponse` as `navCacheJson` to `ObjectBox`. Do not await it before the push — the push happens immediately, the cache write runs in the background.
- **D-13:** Use `unawaited(...)` explicitly (from `dart:async`) to suppress the implicit discard lint warning.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Core Integration Points
- `app/lib/util/navigation_launch_util.dart` — `launchNavigation()` function; add DioException catch, ObjectBox read, cache write, isOffline flag, and unawaited re-cache here
- `app/lib/services/trail_download_service.dart` — `TrailDownloadService.downloadTrail()`; add best-effort Valhalla cache write before `box.put(entity)`
- `app/lib/routes/navigation_screen.dart` — add `isOffline: bool` constructor param; show `FaIcon(FontAwesomeIcons.wifiSlash)` inside `_buildBanner` when true

### Providers & Store Access
- `app/lib/provider/objectbox_store_provider.dart` — `objectBoxProvider`; provides `Store`; already `keepAlive: true`
- `app/lib/provider/trail/trail_download_provider.dart` — `TrailDownloadServiceNotifier`; injects `objectBoxProvider` and `apiProvider` into `TrailDownloadService`

### Models
- `app/lib/entities/trail_entity.dart` — `TrailEntity`; `navCacheJson: String?` field added in Phase 4; read/write this for the offline cache
- `app/lib/models/navigate_response.dart` — `NavigateResponse` freezed class with `maneuvers`, `shape`, `shapeAsLatLng` extension; serialization fixed in Phase 4

### Shared Helper Target
- `app/lib/util/gpx_util.dart` — extract `_buildNavShape` (downsampling, ≤500 points, first+last preserved) here; both `launchNavigation` and `downloadTrail` call it

### Router
- `app/lib/provider/router_provider.dart` — navigate route handler that unpacks `extra`; must be updated to unpack `(NavigateResponse, bool)` record and pass both to `NavigationScreen`

### Requirements Reference
- `.planning/REQUIREMENTS.md` — OFFLINE-01 through OFFLINE-04 definitions
- `.planning/ROADMAP.md` — Phase 5 success criteria (4 items)

### Phase 4 Context (prerequisite)
- `.planning/phases/04-serialization-fix-entity-schema/04-CONTEXT.md` — D-06 through D-08 (entity schema decisions); serialization fix decisions

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `launchNavigation()` in `navigation_launch_util.dart` — existing try/catch wrapper; extend the catch block for DioException → ObjectBox read; add unawaited re-cache after push
- `TrailDownloadService._downloadMapTiles()` — existing sequential try/catch pattern with `rethrow` for fatal failures; the Valhalla cache write should use the same sequential-try/catch-skip pattern (not rethrow)
- `objectBoxProvider` — already available via `ref.read()` in any Riverpod context; `launchNavigation` already has `WidgetRef ref`
- `_buildBanner()` in `NavigationScreen` — add trailing `FaIcon(FontAwesomeIcons.wifiSlash)` conditionally on `isOffline`

### Established Patterns
- `String? navCacheJson` on `TrailEntity` — entity-only field; read via `box.query(TrailEntity_.id.equals(trail.id)).build().findFirst()?.navCacheJson`
- Best-effort try/catch in download service — `_downloadPhotos` swallows `DioException` (non-cancel) and returns null; Valhalla cache write follows the same pattern
- `unawaited()` from `dart:async` — Dart lint-safe way to fire-and-forget a Future
- `FaIcon(FontAwesomeIcons.*)` — used for `triangleExclamation`, `locationCrosshairs`, `compass` in NavigationScreen; `wifiSlash` is available in font_awesome_flutter 11.0.0

### Integration Points
- Router `extra` for navigate route: currently `NavigateResponse`; change to `(NavigateResponse, bool)` record; update both the push site (`launchNavigation`) and the route handler in `router_provider.dart`
- `TrailEntity.navCacheJson` write path: `entity.navCacheJson = jsonEncode(response.toJson()); box.put(entity);` inside a `_store.runInTransaction(TxMode.write, ...)` call — matches existing pattern on line 73 of `trail_download_service.dart`
- `_buildNavShape` helper: takes `List<WptType>` (from `gpx.allPoints`), returns `List<Map<String, double>>` (shape points); costing is derived separately via `_costingFor()` which stays in `navigation_launch_util.dart`

</code_context>

<specifics>
## Specific Ideas

- The `(NavigateResponse, bool)` Dart record is idiomatic Dart 3 and avoids adding a new model file. The router handler pattern should match what already exists in `router_provider.dart` for the existing navigate route.
- `FaIcon(FontAwesomeIcons.wifiSlash)` at the trailing edge of the maneuver banner is visible without obscuring the instruction text. The banner currently shows an icon + text row — adding a trailing wifi-off icon with `if (widget.isOffline)` is a minimal, contained change.
- The unawaited re-cache in `launchNavigation` should wrap its own try/catch so a cache write failure (ObjectBox error) doesn't surface to the user after a successful online navigation session.

</specifics>

<deferred>
## Deferred Ideas

- None — discussion stayed within phase scope.

</deferred>

---

*Phase: 05-cache-write-fallback-ui*
*Context gathered: 2026-06-14*
