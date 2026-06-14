# Architecture Research

**Project:** Wanderer Offline Navigation (v1.1)
**Researched:** 2026-06-14
**Scope:** Integration of cached Valhalla navigation instructions into the existing ObjectBox + Riverpod architecture

---

## Integration Points

### 1. ObjectBox Entity Model — JSON blob on TrailEntity

**Decision: Add `navigationJson String?` field directly to `TrailEntity`.** Do not create a separate `NavigateResponseEntity`.

**Rationale:**

`NavigateResponse` is a dependent aggregate — it is meaningless without its `TrailEntity`. Its lifecycle is identical: cached when the trail is downloaded, deleted when the trail is deleted from the library. ObjectBox's relational model (via `ToOne`/`ToMany`) exists to express independent entities with their own lifecycle. `NavigateResponse` has no such independence.

The existing codebase already uses this pattern: `gpxData String?` on `TrailEntity` stores the raw GPX string rather than a `GpxEntity`. Navigation JSON is structurally the same case — a serialised payload whose value belongs entirely to the trail it describes.

**Field to add to `TrailEntity`:**

```dart
String? navigationJson;
```

Store the result of `jsonEncode(response.toJson())`. On read, decode with `NavigateResponse.fromJson(jsonDecode(navigationJson!))`. The `NavigateResponse` freezed class already derives `toJson` / `fromJson` from `json_serializable` (confirmed via `navigate_response.g.dart`), so no additional serialisation code is needed.

**ObjectBox schema impact:** Adding a nullable `String?` property to an existing entity is a non-breaking ObjectBox migration. The generator produces a new schema version; existing stored entities gain the field as `null`. No migration hook is required.

**Alternative rejected — `NavigateResponseEntity`:**
A separate entity would require a `ToOne<NavigateResponseEntity>` on `TrailEntity`, a new entity class with flattened fields (lists of doubles and maneuver scalars cannot be stored directly in ObjectBox without custom converters), and coordinated lifecycle management across two boxes. All of this adds complexity with no architectural benefit.

---

### 2. Download Flow — Where to Trigger the Valhalla Call

**Decision: Fetch and persist navigation instructions at the end of `TrailDownloadService.downloadTrail()`, after the entity is written to the ObjectBox store.**

**Current `downloadTrail` sequence (from `trail_download_service.dart`):**
1. Create local directory.
2. Download trail photos.
3. Download waypoint photos.
4. Download map tile cells (PMTiles) — the longest-running step.
5. Write `TrailEntity` to ObjectBox via `_store.runInTransaction`.

**New step 6** — call `/valhalla/navigate` and patch `entity.navigationJson` before (or as part of) the transaction:

```
downloadTrail()
  ...existing steps 1-5...
  (6) build shape list from trail GPX — same downsampling logic already in navigation_launch_util.dart
  (7) POST /valhalla/navigate
  (8) entity.navigationJson = jsonEncode(response.toJson())
  (9) _store.runInTransaction(TxMode.write, () { box.put(entity); })
```

The Valhalla call uses the same `_api` (`Dio`) instance `TrailDownloadService` already holds. The shape-building logic should be extracted from `launchNavigation` into a shared `_buildShape(List<Wpt> points)` helper (see New vs Modified below) so both call sites use identical downsampling.

**Failure handling:** If the Valhalla call throws (network error, API error, timeout), log the error and continue — `entity.navigationJson` remains `null`. The trail is still fully downloaded for offline map use. `launchNavigation` will detect `null` and attempt a live network call, which is the correct fallback behaviour. Do not rethrow; do not cancel the whole download.

---

### 3. `launchNavigation` — Cache-First Strategy

**Decision: Try cache first; only call the network if no cached instructions are available.**

**Revised `launchNavigation` flow:**

```
launchNavigation(context, ref, trail)
  (0) Location guards — unchanged
  (1) GPX guard — unchanged
  (2) Costing derivation — unchanged
  (3) Try cache:
        store = ref.read(objectBoxProvider)
        entity = box.query(TrailEntity_.id.equals(trail.id)).findFirst()
        if entity?.navigationJson != null:
          response = NavigateResponse.fromJson(jsonDecode(entity!.navigationJson!))
          if response.maneuvers.isNotEmpty && response.shape.isNotEmpty:
            context.push('/trail/${trail.id}/navigate', extra: response)
            return
  (4) Network call (existing steps 3-7) — unchanged
  (5) On success: context.push(...)
  (6) On error: error toast
```

**Why cache-first, not network-first with cache fallback:**

The PROJECT.md requirement is "Navigation falls back to cached instructions when offline." A network-first strategy attempts the POST call even when the device is offline, which means the hiker waits for a Dio timeout (default 5-15 seconds) before the cache is consulted. Cache-first is instant for the common offline case, and the network path is still reached whenever the cache is empty (e.g. trail downloaded by an older app version without navigation JSON).

**Staleness:** The PROJECT.md requirement explicitly states "silently use cached version if trail updated since caching." No freshness check is required. The cache is used if present, regardless of the trail's `updated` timestamp.

**No online-vs-offline detection needed:** The cache-first path handles both cases correctly. If cached instructions exist, they are used even when online (instant, no network cost). If no cache exists, the network is called. This is simpler than detecting connectivity and matches the stated goal.

---

### 4. Riverpod Provider Pattern for Cache Read/Write

**Decision: No new provider is needed. Use synchronous ObjectBox reads directly in `launchNavigation` and `TrailDownloadService`.**

**Cache read (in `launchNavigation`):**

Read `objectBoxProvider` synchronously via `ref.read(objectBoxProvider)` — already the established pattern in `trail_provider.dart`'s offline fallback block. `launchNavigation` already accepts a `WidgetRef`, so `ref.read(objectBoxProvider)` is available without any additional scaffolding.

**Cache write (in `TrailDownloadService`):**

`TrailDownloadService` already holds a `Store` reference injected via its constructor (`TrailDownloadService(this._store, this._api)`). Writing `entity.navigationJson` before `_store.runInTransaction` follows the exact same pattern as the existing `entity.photos = localPaths` and `entity.pmTiles = cellPaths` assignments.

**Why not a new `navigationCacheProvider`:**

A dedicated provider would be justified if the navigation cache needed to be observed reactively (e.g. a UI widget that updates when the cache populates). In this milestone the cache is write-once-at-download-time and read-once-at-navigation-launch. Both access sites are already in contexts that have `ref` or `Store` available. Adding a provider layer would introduce indirection without benefit.

**If a provider is later needed** (e.g. to show "navigation cached" badge in the library): a simple `@riverpod Future<bool> hasNavigationCache(Ref ref, String trailId)` that reads from the store would be the right pattern at that point, following `trail_provider.dart`'s offline query.

---

## New vs Modified

### New

| Artifact | Type | Purpose |
|----------|------|---------|
| `TrailEntity.navigationJson` | Field (`String?`) | Persists serialised `NavigateResponse` JSON alongside the trail entity |
| `_buildShape()` helper | Free function in `navigation_launch_util.dart` (or extracted to `gpx_util.dart`) | Shared downsampling logic used by both `launchNavigation` and `TrailDownloadService` |

### Modified

| Artifact | Change | Why |
|----------|--------|-----|
| `app/lib/entities/trail_entity.dart` | Add `String? navigationJson` field; add `navigationJson: entity.navigationJson` in `TrailEntityMapping.toModel()` if surfaced on `Trail`, or keep as entity-only field accessed directly from the box | Stores the navigation cache |
| `app/lib/models/trail.dart` | Optionally add `String? navigationJson` to the `Trail` freezed model if the detail screen needs to know whether navigation is cached; otherwise omit (entity-only is sufficient for `launchNavigation` querying the box directly) | Propagates cache status to UI if needed |
| `app/lib/services/trail_download_service.dart` | Add step 6-8 after existing ObjectBox write: build shape, POST `/valhalla/navigate`, set `entity.navigationJson`, put entity | Caches navigation instructions at download time |
| `app/lib/util/navigation_launch_util.dart` | Add step 3 (cache lookup before network call); extract shape-building into `_buildShape()` | Implements cache-first launch; eliminates duplicated downsampling logic |

**No changes required to:**
- `navigation_provider.dart` — consumes `NavigateResponse` in-memory, unaffected
- `router_provider.dart` — `/trail/:id/navigate` route unchanged
- `trail_library_provider.dart` — library list read unchanged
- `TrailDownloadServiceNotifier` provider — injects same dependencies, no change needed
- Any screen files — `launchNavigation` signature is unchanged; callers are unaffected

---

## Data Flow

### Download-Time (Cache Write)

```
TrailDropdown._downloadTrail()
  → ref.read(trailDownloadServiceProvider).downloadTrail(trail)
      → [existing] download photos, waypoints, PMTiles
      → [new] _buildShape(gpx.allPoints)
      → [new] _api.post('/valhalla/navigate', data: {shape, costing})
      → [new] entity.navigationJson = jsonEncode(response.toJson())
      → _store.runInTransaction { box.put(entity) }   // unchanged, now includes navigationJson
```

### Navigation Launch — Cached Path (Offline / Cache Hit)

```
TrailPanel / TrailDetailScreen → launchNavigation(context, ref, trail)
  → [new] store = ref.read(objectBoxProvider)
  → [new] entity = box.query(TrailEntity_.id.equals(trail.id)).findFirst()
  → [new] response = NavigateResponse.fromJson(jsonDecode(entity.navigationJson!))
  → context.push('/trail/${trail.id}/navigate', extra: response)
      → NavigationScreen(id, response)
          → ref.watch(navigationProvider(response))   // unchanged
```

### Navigation Launch — Network Path (No Cache / Online)

```
launchNavigation(context, ref, trail)
  → [cache miss — navigationJson is null]
  → _api.post('/valhalla/navigate', ...)              // unchanged existing flow
  → context.push('/trail/${trail.id}/navigate', extra: response)
```

### Key Invariant

`NavigationScreen` and `navigationProvider` receive a `NavigateResponse` object regardless of whether it came from cache or network. No changes are needed downstream of the push.

---

## Build Order

The following sequence minimises broken-build windows and respects code-generation dependencies.

### Step 1 — Entity schema (no codegen, but triggers ObjectBox regeneration)

Add `String? navigationJson` to `TrailEntity`. Run `dart run build_runner build` to regenerate `objectbox.g.dart` with the new schema version. Verify the generated `_TrailEntityBinding` includes the new property.

**Dependency:** Must complete before any code reads or writes `navigationJson` at runtime.

### Step 2 — Shared shape helper

Extract `_buildShape(List<Wpt> points)` from `navigation_launch_util.dart` into a private or package-level function. This is a pure refactor — existing behaviour of `launchNavigation` is unchanged. Verify via existing unit tests if present.

**Dependency:** Must complete before Step 3 (download service) consumes it.

### Step 3 — Download service: cache write

Add the Valhalla call + `entity.navigationJson` assignment to `TrailDownloadService.downloadTrail`. Error is swallowed (logged, not rethrown). Manually test: download a trail, inspect the entity in debug, confirm `navigationJson` is populated.

**Dependency:** Requires Step 1 (field exists) and Step 2 (shared shape helper).

### Step 4 — `launchNavigation`: cache-first read

Add the cache lookup block before the existing `api.post` call. Guard for `null` / empty response as the existing code does. Manual test: with device in airplane mode, tap Navigate on a downloaded trail — should launch instantly from cache. With device online and no cache, should fall back to network normally.

**Dependency:** Requires Step 1 (field readable from entity).

### Step 5 — Integration smoke test

End-to-end: download a trail on Wi-Fi, enable airplane mode, launch navigation. Confirm the navigation screen appears with correct maneuvers and shape. Confirm that navigating a non-downloaded trail online still works.

---

## Confidence Assessment

| Area | Confidence | Basis |
|------|------------|-------|
| Entity field placement (blob vs. entity) | HIGH | Direct code read of `TrailEntity`, `WaypointEntity`, `gpxData` precedent, ObjectBox nullable field semantics |
| Download trigger point | HIGH | Full read of `TrailDownloadService.downloadTrail`; clear sequential structure with single ObjectBox write at end |
| Cache-first strategy | HIGH | PROJECT.md requirement explicitly states offline-first intent; Dio timeout behaviour well understood |
| No new provider needed | HIGH | `launchNavigation` already has `WidgetRef`; `TrailDownloadService` already has `Store`; no reactive UI need identified |
| ObjectBox schema migration safety | MEDIUM | Nullable field additions are documented as non-breaking in ObjectBox Flutter docs; not verified against the exact 5.3.1 changelog |
| `NavigateResponse` JSON round-trip | HIGH | `json_serializable` codegen confirmed present in `navigate_response.g.dart`; `toJson`/`fromJson` already exercised in the existing online flow |
