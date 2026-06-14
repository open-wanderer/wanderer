# Pitfalls Research

**Domain:** Offline navigation caching — Flutter + ObjectBox + freezed + Riverpod
**Project:** Wanderer v1.1 Offline Navigation
**Researched:** 2026-06-14
**Scope:** Adding Valhalla navigation instruction caching to an existing download flow

---

## ObjectBox Pitfalls

### Pitfall OBX-1: Dart types that ObjectBox CANNOT store natively

**What goes wrong:**
You add a field to `TrailEntity` using a type that ObjectBox's code generator does not understand. The generator either throws a build error or silently marks the field `@Transient()`, meaning the data is never persisted.

**Unsupported types that will cause problems:**

| Type | What happens | Fix |
|------|-------------|-----|
| `List<NavigateManeuver>` | Build error — ObjectBox supports `List<String>`, `List<int>`, `List<double>`, `List<DateTime>` (homogeneous primitive lists), but NOT `List<CustomObject>` | Serialize to `String` (JSON blob) and store that |
| `List<List<double>>` | Build error — nested lists are not supported | Serialize to `String` (JSON blob) |
| `Map<String, dynamic>` | Supported only via `@Property(type: PropertyType.flex)` — cannot use plain `Map` field | Add `@Property(type: PropertyType.flex)` annotation |
| `NavigateResponse` | Not supported at all — ObjectBox stores scalar fields and primitive lists only | Serialize the entire object to JSON string |
| `LatLng` / `DateTime` lists with complex types | No — only `List<DateTime>` in isolation is supported | Serialize |
| `enum` directly | Not supported natively | Use the `dbDifficulty` int proxy pattern already in `TrailEntity` |

**What IS supported natively:**
- `int`, `double`, `bool`, `String`, `DateTime` (with `@Property(type: PropertyType.dateUtc)`)
- `List<int>`, `List<double>`, `List<bool>`, `List<String>`, `List<DateTime>` (homogeneous, single-type)
- `@Property(type: PropertyType.flex)` for `Map<String, dynamic>` or `dynamic` fields

**For storing `NavigateResponse`:**
The correct approach is a `String? navCacheJson` field on `TrailEntity`, populated with `jsonEncode(response.toJson())`. On read, `NavigateResponse.fromJson(jsonDecode(entity.navCacheJson!) as Map<String, dynamic>)`.

**Source:** ObjectBox property types docs (https://docs.objectbox.io/property-types), confirmed via Context7 `/objectbox/objectbox-dart`

---

### Pitfall OBX-2: Adding a new field to an existing entity — the objectbox-model.json trap

**What goes wrong:**
You add `String? navCacheJson` to `TrailEntity` and run `dart run build_runner build`. ObjectBox assigns a new UID and updates `objectbox-model.json`. This is normally safe. However, three scenarios cause silent or catastrophic data loss:

**Scenario A — Deleted field re-added with a different UID.**
If you add the field, run build, then delete it and add it back under the same name, ObjectBox will treat the second add as a new property (new UID) rather than the same one. Existing cached data in the old UID column is orphaned. The `retiredPropertyUids` in `objectbox-model.json` protects against reuse, but existing rows now have a column whose UID no longer matches the property — the value reads as null.

**Scenario B — objectbox-model.json is NOT committed / is conflict-resolved wrong.**
`objectbox-model.json` is the ground truth for UIDs. If two branches each add a field and both get different UIDs, then merging creates a conflict. Resolving it incorrectly (e.g., keeping one branch's entire file) causes the other branch's field UID to be "new" to ObjectBox, wiping that field's data on next open.

**Scenario C — Changing the field type (e.g., `String?` to `Map<String, dynamic>`).**
ObjectBox does NOT migrate property data when the type changes. The old bytes for the `String` column are not usable as `flex` bytes. Existing rows for that field become corrupt/null. You must treat it as a rename: add the new field under a new name, migrate data in code, then remove the old name in a follow-up.

**Prevention:**
- Always commit `objectbox-model.json` immediately after running build_runner
- Never resolve merge conflicts in `objectbox-model.json` by just keeping one side — follow ObjectBox's UID resolution docs
- Assign `navCacheJson` as a new nullable `String?` field. Don't rename or retype it later

**Source:** https://docs.objectbox.io/advanced/data-model-updates, https://docs.objectbox.io/advanced/meta-model-ids-and-uids

---

### Pitfall OBX-3: The `@Transient()` silent no-op

**What goes wrong:**
If a field type is not supported and ObjectBox cannot generate code for it, the generator marks the field `@Transient()` without a compile error — the build succeeds but the field is never written to disk. You won't notice until the app is restarted and the cache reads back as null.

**Specific risk here:**
If someone writes `NavigateResponse? navCache` on `TrailEntity` and forgets to serialize it to a String first, the generator adds `@Transient()` silently. Navigation looks like it cached, but reopening the app shows the field as null.

**Detection:**
After adding any new field, check the generated `objectbox.g.dart` and `objectbox-model.json`. The new field must appear in the entity's `properties` array in the JSON. If it's absent, it's being treated as `@Transient()`.

**Prevention:**
Only use ObjectBox-supported types. For `NavigateResponse`, use `String? navCacheJson` (serialize via `jsonEncode`) instead of the typed field.

---

## Connectivity Detection

### Pitfall CONN-1: connectivity_plus reports "connected" when there is no internet access

**What goes wrong:**
`connectivity_plus` checks whether a network *interface* is active (Wi-Fi adapter on, mobile radio enabled), not whether actual internet requests will succeed. A device connected to a hotel captive portal, a VPN with a split-tunnel, or an airplane-mode Wi-Fi (no upstream) all report `ConnectivityResult.wifi` or `ConnectivityResult.mobile`.

The official package documentation states explicitly: "Note that on Android, this does not guarantee connection to Internet. For instance, the app might have wifi access but it might be a VPN or a hotel WiFi with no Internet access."

**For this project:**
If `launchNavigation` checks `connectivity_plus` to decide whether to hit the network or use the cache, it will attempt a Dio POST on a captive portal and hang or time out. The user gets a loading spinner, not a graceful fallback to cached instructions.

**What to do instead:**
Do NOT gate the network call on connectivity status. Instead:
1. Always attempt the Dio POST first (with a short timeout, e.g., 6–10 seconds)
2. If the POST throws `DioException` (timeout, socket error, connection refused), fall back to the ObjectBox cache
3. Only use `connectivity_plus` as a soft hint (e.g., to suppress the "fetching fresh route" indicator on the UI)

**Source:** Official connectivity_plus README (https://pub.dev/packages/connectivity_plus), confirmed via Context7 `/websites/pub_dev_packages_connectivity_plus`

---

### Pitfall CONN-2: VPN causes false `ConnectivityResult.none`

**What goes wrong:**
The inverse problem. GitHub issue #3810 on the plus_plugins repo documents that `checkConnectivity()` returns `[ConnectivityResult.none]` on Android when only a VPN interface is active. A hiker using a VPN would be told they are offline even with full internet access, and the app would immediately use the stale cache instead of fetching fresh instructions.

**Prevention:**
Same as CONN-1: use network attempt + error fallback as the truth signal, not `connectivity_plus` status as the gate.

**Source:** https://github.com/fluttercommunity/plus_plugins/issues/3810

---

### Pitfall CONN-3: Race between `onConnectivityChanged` stream and actual network readiness

**What goes wrong:**
When a stream listener fires `ConnectivityResult.mobile` or `ConnectivityResult.wifi`, the underlying TCP stack may not yet have a routable path. Making a Dio request within 50–200 ms of the event fires often results in a `SocketException` even though `connectivity_plus` says connected.

**For this project:**
This matters if navigation auto-retries after a connectivity-restored event (e.g., "you are back online — refreshing route"). An immediate retry will fail because the interface isn't stable yet.

**Prevention:**
Add a short debounce (500ms–1s) before acting on a connectivity-restored event, AND wrap the retry in a try/catch that falls back to cache on any `DioException`.

---

## Serialization

### Pitfall SER-1: `NavigateResponse.toJson()` does NOT serialize nested `NavigateManeuver` objects without `explicitToJson`

**What goes wrong:**
`_$NavigateResponseToJson` in the generated `navigate_response.g.dart` currently outputs:

```dart
Map<String, dynamic> _$NavigateResponseToJson(_NavigateResponse instance) =>
    <String, dynamic>{'maneuvers': instance.maneuvers, 'shape': instance.shape};
```

The `maneuvers` value is stored as a raw `List<NavigateManeuver>` (Dart objects), not as `List<Map<String, dynamic>>`. When this map is passed to `jsonEncode()`, Dart calls `.toString()` on each `NavigateManeuver` instance, which produces `NavigateManeuver(instruction: ..., ...)` — a plain Dart debug string. `jsonEncode` throws a `JsonUnsupportedObjectError` at runtime, or if some version of the generated code calls `toJson()` on the list elements, the output may be inconsistent depending on codegen configuration.

**Why it happens:**
Freezed delegates JSON generation to `json_serializable`. By default, `json_serializable` does NOT call `.toJson()` on nested objects in lists — it only does so when `explicitToJson: true` is set. The `@JsonSerializable()` annotation on `_NavigateResponse` in the generated `navigate_response.freezed.dart` has no explicit options, so the default applies.

**The fix:**
Add `@JsonSerializable(explicitToJson: true)` to the `NavigateResponse` factory in the source file OR set `explicit_to_json: true` globally in `build.yaml`. After regenerating, `_$NavigateResponseToJson` will call `.toJson()` on each maneuver:

```dart
// After fix:
Map<String, dynamic> _$NavigateResponseToJson(_NavigateResponse instance) =>
    <String, dynamic>{
      'maneuvers': instance.maneuvers.map((e) => e.toJson()).toList(),
      'shape': instance.shape,
    };
```

**Verification test:**
```dart
final response = NavigateResponse(
  maneuvers: [NavigateManeuver(instruction: 'Turn left', length: 0.5, beginShapeIndex: 0)],
  shape: [[47.0, 8.0], [47.1, 8.1]],
);
final encoded = jsonEncode(response.toJson());       // must not throw
final decoded = NavigateResponse.fromJson(jsonDecode(encoded) as Map<String, dynamic>);
assert(decoded.maneuvers.first.instruction == 'Turn left');
```

Run this in a unit test before wiring up ObjectBox storage.

**Phase:** Address in whichever phase adds the cache storage logic. This is a blocking bug — the feature cannot work without fixing it first.

---

### Pitfall SER-2: `List<List<double>>` shape roundtrip type coercion

**What goes wrong:**
`NavigateResponse.shape` is `List<List<double>>`. After a `jsonEncode` → `jsonDecode` roundtrip, `jsonDecode` produces `List<dynamic>` where inner lists are also `List<dynamic>` containing `num` values (not necessarily `double`). If the `fromJson` path casts elements as `(e as num).toDouble()`, the roundtrip is safe. The existing generated code in `navigate_response.g.dart` already handles this correctly:

```dart
shape: (json['shape'] as List<dynamic>)
    .map((e) => (e as List<dynamic>).map((e) => (e as num).toDouble()).toList())
    .toList(),
```

**Residual risk:**
If someone manually writes a `fromJson` or uses `jsonDecode` without going through `NavigateResponse.fromJson`, the cast chain may be skipped and a `TypeError` thrown at runtime when `NavigationProvider` accesses `response.shape`.

**Prevention:**
Always deserialize via `NavigateResponse.fromJson(jsonDecode(blob) as Map<String, dynamic>)`. Never deserialize via direct field access on the decoded map.

---

### Pitfall SER-3: JSON blob stored in ObjectBox becomes stale vs. server-side route changes

**What goes wrong:**
The cache stores the Valhalla response at download time. If Valhalla's routing algorithm changes, or the trail GPX is updated, the cached maneuvers still refer to the old shape's `beginShapeIndex` values. The navigation provider will advance through maneuvers based on stale shape indices, potentially skipping maneuvers or never advancing at all.

**For this project:**
The PROJECT.md already accepts this: "Silently use cached version if trail updated since caching." This is an explicit design decision, not a bug to fix. But it must be documented so the implementation does NOT add any logic that validates cache freshness at navigate time (e.g., comparing `trail.updated` to a cached timestamp), because that complexity is out of scope for v1.1.

**Prevention:**
Store only the JSON blob — no accompanying `cachedAt` or `trailUpdatedAt` field. If a future milestone adds cache invalidation, that can be added then as a new field without breaking the existing schema.

---

## Race Conditions

### Pitfall RACE-1: Navigation launched before download write transaction commits

**What goes wrong:**
The download service ends with:
```dart
_store.runInTransaction(TxMode.write, () {
  box.put(entity);
});
```

If navigation is launched immediately after the download "completes" (e.g., the user taps Navigate before the provider rebuilds), the `box.get()` call to read back the cached `navCacheJson` may execute while the write transaction is still pending on the ObjectBox worker isolate. The field reads as null, and the app falls back to a network fetch (correct behavior) or throws (bad behavior if the null check is missing).

**Why ObjectBox transactions are safe in this specific case:**
`store.runInTransaction(TxMode.write, ...)` with a synchronous callback commits before the call returns. The risk is zero if the write uses the synchronous form. The risk is real if someone changes it to `store.runAsync(...)` — then the future returned by `downloadTrail` could complete before the async transaction commits.

**Prevention:**
Keep the transaction synchronous (`TxMode.write` with a synchronous callback). Do not switch to `store.runAsync()` for the cache write step just because the surrounding method is async.

**Detection warning sign:**
`runInTransaction` is already used correctly in `trail_download_service.dart`. Any PR that replaces it with `runAsync` for the nav cache write step introduces this race.

---

### Pitfall RACE-2: Concurrent downloads writing to the same `TrailEntity`

**What goes wrong:**
`downloadTrail` creates a `TrailEntity.fromModel(trail)` early (before network calls) and then calls `box.put(entity)` at the end. If the user somehow triggers two concurrent downloads for the same trail (e.g., tapping download twice before the first completes), both calls create independent `TrailEntity` objects with `obxId = 0`. The `@Unique(onConflict: ConflictStrategy.replace)` on the `id` field ensures only one row exists — but whichever write completes last wins, and the first write's `navCacheJson` may be overwritten by a second write that hasn't fetched the nav cache yet (because it started first but the cache fetch hasn't completed).

**For the nav cache specifically:**
When adding the `navCacheJson` field, the nav cache Dio POST happens mid-download (after photos, before the `box.put`). If two concurrent downloads race and one completes the nav POST while the other hasn't yet, the final `box.put` from the faster download may be overwritten by the `box.put` from the slower one — which has `navCacheJson = null` because it hasn't completed the POST yet.

**Prevention:**
The `TrailDownloadProvider` already uses a Riverpod notifier pattern; enforce that only one download per trail ID can be in-flight by checking the provider state before starting a second. If the download service has no guard, add a `Set<String> _downloading` set to `TrailDownloadService` and skip/throw if the trail ID is already present.

---

### Pitfall RACE-3: `launchNavigation` reads entity before download marks it offline

**What goes wrong:**
The flow is:
1. Download completes → `box.put(entity)` with `navCacheJson` set
2. Riverpod provider watching the trail entity rebuilds (async — next frame or later)
3. User taps Navigate

If the tap happens in the same frame as the `box.put` (step 1), the provider may still be serving the pre-download version of `Trail` (where `trail.isOffline == false`). The `launchNavigation` function, if it checks `trail.isOffline` to decide whether to use the cache, will incorrectly branch to the network path.

**Prevention:**
`launchNavigation` should NOT gate on `trail.isOffline`. Instead:
1. Always attempt a fresh network fetch first
2. Catch `DioException` and fall back to the ObjectBox cache
3. The `isOffline` flag on the model is only for UI display (e.g., showing an "offline available" badge), not for routing logic

---

### Pitfall RACE-4: Unawaited nav cache Dio POST inside `downloadTrail`

**What goes wrong:**
If the Valhalla POST is added to `downloadTrail` but not properly awaited — for example, added as a parallel `Future.wait` task alongside photo downloads without proper error handling — a failure of the POST will not cancel or retry the download. The download completes successfully, `box.put` runs, but `navCacheJson` is null because the POST result was dropped on the floor (or the exception was swallowed).

**Specific risk pattern:**
```dart
// WRONG — exception from navPost will be swallowed if not handled in Future.wait
final results = await Future.wait([
  _downloadPhotos(...),
  _fetchNavCache(trail),  // if this throws, Future.wait re-throws but other tasks are not cancelled
]);
```

The existing `downloadTrail` uses sequential awaits for photos and then a `Future.wait` for map tiles. Inserting the nav fetch without understanding where in this sequence it belongs risks ordering bugs.

**Prevention:**
Fetch the nav cache in a dedicated sequential step with explicit error handling:
```dart
String? navCacheJson;
try {
  final navResponse = await _fetchNavCache(trail, cancelToken: cancelToken);
  navCacheJson = jsonEncode(navResponse.toJson());
} catch (e) {
  // Nav cache is best-effort — log and continue without it
  // The download still succeeds; launchNavigation will hit the network instead
}
```
This way a Valhalla outage does not block a trail download.

---

## Prevention Strategies

### Strategy P-1: Write and run a roundtrip test before wiring ObjectBox

Before adding the `navCacheJson` field to `TrailEntity`, write a standalone Dart unit test:

```dart
test('NavigateResponse toJson/fromJson roundtrip', () {
  final original = NavigateResponse(
    maneuvers: [
      NavigateManeuver(instruction: 'Head north', length: 0.3, beginShapeIndex: 0, bearing: 0.0, type: 1),
    ],
    shape: [[47.123, 8.456], [47.124, 8.457]],
  );
  final blob = jsonEncode(original.toJson());
  final decoded = NavigateResponse.fromJson(jsonDecode(blob) as Map<String, dynamic>);
  expect(decoded.maneuvers.first.instruction, equals('Head north'));
  expect(decoded.shape.first.first, closeTo(47.123, 0.001));
});
```

If this test fails with `JsonUnsupportedObjectError`, the `explicitToJson` fix (Pitfall SER-1) must be applied first.

---

### Strategy P-2: Verify ObjectBox code generation before committing

After adding any field to an entity:
1. Run `dart run build_runner build --delete-conflicting-outputs`
2. Open `objectbox-model.json` — confirm the new property appears in the entity's `properties` array with a non-zero UID
3. If it does NOT appear, the field type is unsupported and is being silently skipped
4. Commit `objectbox-model.json` in the same commit as the entity change

---

### Strategy P-3: Use try/catch + fallback as the network/cache decision boundary

The decision logic in `launchNavigation` should be:

```dart
NavigateResponse? response;

// 1. Always try the network first
try {
  response = await _fetchFromNetwork(trail);
} on DioException {
  // 2. Fall back to cache on any network failure
  response = await _readFromCache(trail.id);
}

// 3. If both fail, show error toast
if (response == null) {
  showError(l10n.couldnt_start_navigation);
  return;
}
```

This pattern handles: no internet, captive portal, VPN false negative, server down, timeout. It does not require `connectivity_plus` at all for the navigation decision.

---

### Strategy P-4: Keep the nav cache fetch best-effort in `downloadTrail`

Add the nav cache POST as a sequential step that never fails the entire download:

```dart
// After all photos and map tiles are downloaded:
String? navCacheJson;
try {
  final r = await _fetchNavCache(trail, cancelToken: cancelToken);
  navCacheJson = jsonEncode(r.toJson());
} catch (_) {
  // Best-effort; null means no offline navigation for this trail
}

final entity = TrailEntity.fromModel(trail);
entity.navCacheJson = navCacheJson;
// ... set photos, pmTiles ...
_store.runInTransaction(TxMode.write, () { box.put(entity); });
```

---

### Strategy P-5: Guard `connectivity_plus` usage

If `connectivity_plus` is added to `pubspec.yaml`, confine its use to:
- UI hints only (e.g., showing "offline mode" label in the navigation screen)
- Never use it as a gate on whether to make a network request

Always wrap Dio calls in try/catch regardless of what `connectivity_plus` reports.

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Add `navCacheJson` to `TrailEntity` | OBX-2 (model.json UID conflicts), OBX-3 (silent @Transient) | Verify field appears in model JSON after build; commit immediately |
| Serialize `NavigateResponse` to/from String | SER-1 (missing explicitToJson on maneuvers) | Write roundtrip unit test first (Strategy P-1) |
| Add nav POST call to `downloadTrail` | RACE-4 (unawaited/swallowed future), RACE-2 (concurrent downloads) | Make best-effort with try/catch; don't use Future.wait for this step |
| Implement fallback in `launchNavigation` | CONN-1 (false positive), CONN-2 (VPN false negative), RACE-3 (stale trail model) | Use try/catch on Dio POST, not connectivity_plus as gate |
| `List<List<double>>` shape field roundtrip | SER-2 (type coercion) | Always use `NavigateResponse.fromJson()`, never raw map access |
| `objectbox-model.json` merge conflicts | OBX-2 Scenario B | Resolve conflicts following ObjectBox UID docs; never keep "one side wins" |

---

## Sources

- ObjectBox property types: https://docs.objectbox.io/property-types
- ObjectBox data model updates / migration: https://docs.objectbox.io/advanced/data-model-updates
- ObjectBox meta model / UIDs: https://docs.objectbox.io/advanced/meta-model-ids-and-uids
- ObjectBox custom types (converters): https://docs.objectbox.io/advanced/custom-types
- ObjectBox Context7 docs: `/objectbox/objectbox-dart` via Context7 CLI
- connectivity_plus package: https://pub.dev/packages/connectivity_plus
- connectivity_plus VPN bug: https://github.com/fluttercommunity/plus_plugins/issues/3810
- freezed nested toJson issue: https://github.com/rrousselGit/freezed/issues/232
- freezed nested toJson issue #86: https://github.com/rrousselGit/freezed/issues/86
- Codebase: `app/lib/entities/trail_entity.dart`, `app/lib/models/navigate_response.dart`, `app/lib/models/navigate_response.g.dart`, `app/lib/services/trail_download_service.dart`
