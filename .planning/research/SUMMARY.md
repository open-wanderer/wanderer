# Research Summary: Offline Navigation (v1.1)

**Project:** Wanderer — v1.1 Offline Navigation
**Domain:** Flutter offline navigation caching (Valhalla instructions + ObjectBox)
**Researched:** 2026-06-14
**Confidence:** HIGH

---

## Executive Summary

Offline navigation in Wanderer v1.1 is a caching problem, not a routing problem. The goal is to persist Valhalla turn-by-turn instructions at trail-download time so that `launchNavigation` can serve them from ObjectBox when no network is available. This is structurally identical to how the existing download flow already caches GPX data and PMTile cells — add one more async step (`POST /valhalla/navigate`), serialize the response to a `String?` field on `TrailEntity`, and fall back to that field in `launchNavigation` when the network call fails. No new routing engine, no new persistence layer, no new provider is needed.

The recommended approach is network-first with Dio-catch fallback in `launchNavigation` (try network, catch DioException → read ObjectBox) with a best-effort silent write during `downloadTrail` (swallow Valhalla errors so a Valhalla outage cannot block a map-tile download). Industry references (Komoot, AllTrails, Gaia GPS) all follow this same invisible-to-the-user pattern: "download = ready to navigate offline," no separate download step, no staleness UI.

The single most critical risk is a serialization bug in the current codebase: `NavigateResponse.toJson()` silently produces corrupt JSON for the `maneuvers` list because `@JsonSerializable(explicitToJson: true)` is missing. This must be fixed and verified with a roundtrip unit test **before** any ObjectBox write logic is wired up. The second risk class is connectivity detection: `connectivity_plus` and `internet_connection_checker_plus` both have known false-positive/false-negative failure modes in outdoor environments (captive portals, VPN). The safer and simpler approach is to skip connectivity checking entirely and use Dio try-catch as the sole gate.

---

## Stack Additions

### New Package: None recommended

STACK.md recommends `internet_connection_checker_plus ^3.0.1` as a pre-check before the network call. PITFALLS.md directly contradicts this: connectivity packages report network-interface state, not actual internet reachability. A hiker on a captive portal gets `hasInternetAccess: true` but the Dio POST times out. A hiker using a VPN gets `hasInternetAccess: false` even with full connectivity.

**Recommendation: Use Dio try-catch, not a connectivity package.**

```dart
NavigateResponse? response;
try {
  response = await _fetchFromNetwork(trail);
} on DioException {
  response = _readFromCache(trail.id);  // ObjectBox lookup
}
if (response == null) {
  showError(l10n.couldnt_start_navigation);
  return;
}
```

This handles no internet, captive portal, VPN false-negative, server down, and timeout — all in one block, with no new dependency.

### Existing Package: ObjectBox 5.3.1 (no version change)

Add `String? navCacheJson` to the existing `TrailEntity`. Store `jsonEncode(response.toJson())`. Retrieve with `NavigateResponse.fromJson(jsonDecode(entity.navCacheJson!) as Map<String, dynamic>)`. This follows the existing `gpxData String?` precedent on `TrailEntity`.

---

## Feature Table Stakes

| Feature | Location | Complexity |
|---------|----------|------------|
| Cache Valhalla instructions in `downloadTrail` | `TrailDownloadService` | Low |
| Silent offline fallback in `launchNavigation` | `navigation_launch_util.dart` | Low |
| Persist `navCacheJson` on `TrailEntity` | `trail_entity.dart` + ObjectBox codegen | Low |
| Graceful no-op if Valhalla unreachable at download time | `downloadTrail` try/catch | Trivial |

**Anti-features (do not build in v1.1):** stale-cache dialogs, "cached N days ago" UI, user-initiated cache refresh, blocking progress step for Valhalla fetch, offline re-routing.

**Differentiators (after table stakes):** offline icon in `NavigationScreen` AppBar when running from cache, re-cache on next online launch (fire-and-forget).

---

## Architecture Decisions

**Entity placement:** `String? navCacheJson` on `TrailEntity` directly. Follows `gpxData` precedent. Non-breaking ObjectBox migration.

**Write point:** After all existing download steps in `downloadTrail`, add a sequential try/catch POST, assign `entity.navCacheJson`, then the existing synchronous `_store.runInTransaction`. Keep it synchronous.

**Read strategy:** Network-first, catch DioException → cache. Query ObjectBox only when the network call fails. `NavigationScreen` and `navigationProvider` are unchanged — they receive a `NavigateResponse` regardless of source.

**No new Riverpod provider needed.** `launchNavigation` has `WidgetRef`; `TrailDownloadService` has `Store`. A provider can be added later for the badge UI if needed.

**Data flow:**
```
Download: downloadTrail() → [existing steps] → _buildShape() → POST /valhalla/navigate (try/catch) → entity.navCacheJson = jsonEncode(...) → box.put(entity)

Navigate (online): launchNavigation() → POST /valhalla/navigate → context.push(...)
                   └─ fire-and-forget: update navCacheJson in ObjectBox

Navigate (offline/error): launchNavigation() → POST throws DioException → box.query(TrailEntity) → NavigateResponse.fromJson(jsonDecode(navCacheJson!)) → context.push(...)
```

---

## Critical Pitfalls (must fix)

### 1. `NavigateResponse.toJson()` missing `explicitToJson: true` — BLOCKING BUG

`_$NavigateResponseToJson` in `navigate_response.g.dart` writes `maneuvers` as raw Dart objects. `jsonEncode` throws `JsonUnsupportedObjectError` at runtime. The entire offline cache feature is broken without this fix.

**Fix:** Add `@JsonSerializable(explicitToJson: true)` to the `NavigateResponse` freezed source and regenerate. Verify with roundtrip unit test before wiring ObjectBox.

```dart
test('NavigateResponse toJson/fromJson roundtrip', () {
  final original = NavigateResponse(
    maneuvers: [NavigateManeuver(instruction: 'Turn left', length: 0.5, beginShapeIndex: 0, bearing: 0.0, type: 1)],
    shape: [[47.0, 8.0], [47.1, 8.1]],
  );
  final blob = jsonEncode(original.toJson());
  final decoded = NavigateResponse.fromJson(jsonDecode(blob) as Map<String, dynamic>);
  expect(decoded.maneuvers.first.instruction, equals('Turn left'));
});
```

### 2. ObjectBox unsupported field types silently skipped

`List<List<double>>` and `List<NavigateManeuver>` are both unsupported as ObjectBox native types. The generator silently applies `@Transient()`, field is never persisted, reads always return null.

**Prevention:** Use `String? navCacheJson`. After codegen, verify the field appears in `objectbox-model.json`'s `properties` array for `TrailEntity`.

### 3. `objectbox-model.json` UID conflicts on merge

Two branches adding fields independently get different UIDs. Wrong merge resolution wipes field data silently.

**Prevention:** Commit `objectbox-model.json` immediately after build_runner. Follow ObjectBox UID resolution docs on merge conflicts.

### 4. Connectivity package false positives/negatives

`connectivity_plus` and `internet_connection_checker_plus` both have documented failure modes (captive portals, VPN). Neither should gate the navigation decision.

**Prevention:** Dio try-catch only.

### 5. Unawaited nav cache POST in `downloadTrail`

Adding the Valhalla POST to a `Future.wait` block without explicit error handling can silently swallow failures or propagate and cancel the entire download.

**Prevention:** Sequential try/catch, best-effort, never in `Future.wait`.

---

## Watch Out For

| Risk | Mitigation |
|------|------------|
| `List<List<double>>` shape roundtrip type coercion | Always deserialize via `NavigateResponse.fromJson()`. Generated code handles `num` → `double`. |
| Concurrent downloads for same trail racing to `box.put` | Confirm `TrailDownloadProvider` guards against concurrent downloads per trail ID. |
| Async ObjectBox transaction | Keep `_store.runInTransaction(TxMode.write, ...)` synchronous; `runAsync` can race with `launchNavigation` reads. |
| Stale `trail.isOffline` model in `launchNavigation` | Never gate cache lookup on `trail.isOffline`. Always query ObjectBox directly. |

---

## Recommended Build Order

1. **Fix `NavigateResponse.toJson()` serialization bug** — `@JsonSerializable(explicitToJson: true)`, regenerate, roundtrip unit test. BLOCKING; all other steps depend on this.
2. **Add `String? navCacheJson` to `TrailEntity`** — run build_runner, verify field in `objectbox-model.json`, commit the JSON file.
3. **Extract `_buildShape()` helper** — pure refactor shared between download service and `launchNavigation`.
4. **Download service: cache write** — sequential try/catch POST after tiles, assign `entity.navCacheJson`, existing synchronous transaction.
5. **`launchNavigation`: cache fallback** — Dio try-catch → ObjectBox read; pass `isOfflineCache` flag to `NavigationScreen`.
6. **Offline indicator in `NavigationScreen`** — AppBar icon when `isOfflineCache` is true (differentiator).

---

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack (ObjectBox field approach) | HIGH | `gpxData` precedent confirmed in codebase |
| Stack (connectivity recommendation) | HIGH | Dio try-catch resolves CONN-1/CONN-2 failure modes |
| Features | HIGH | Direct codebase analysis of download service and `launchNavigation` |
| Architecture | HIGH | Integration points confirmed from source reads |
| Serialization bug | HIGH | Identified from actual generated `navigate_response.g.dart` |
| ObjectBox migration safety | MEDIUM | Non-breaking nullable additions; not verified against exact 5.3.1 changelog |

**Overall confidence:** HIGH

---

*Research completed: 2026-06-14*
*Ready for roadmap: yes*
