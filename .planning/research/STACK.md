# Stack Research: Offline Navigation Caching

**Project:** Wanderer — v1.1 Offline Navigation
**Researched:** 2026-06-14
**Overall confidence:** HIGH (ObjectBox flex type: Context7 verified; connectivity: Context7 + pub.dev API verified)

---

## New Dependencies

### Connectivity Detection

**Add: `internet_connection_checker_plus ^3.0.1`**

- Pub.dev latest: 3.0.1 (verified 2026-06-14)
- SDK constraint: `>=2.15.0 <4.0.0` — compatible with project's `^3.11.5`
- Single dependency: `http ^1.0.0` — no conflict (project uses `dio`, `http` is not present and will only be a transitive dep)
- Why not `connectivity_plus 7.1.1`: `connectivity_plus` reports network interface status (WiFi connected, mobile data active), NOT actual internet reachability. A hiker in a dead zone with WiFi associated will get `ConnectivityResult.wifi` but have no usable connection. `internet_connection_checker_plus` fires HEAD requests to multiple endpoints concurrently and resolves `true` only when at least one responds. This is the right semantic for "can we reach the Valhalla API right now?"
- Why not catching `DioException`: Catches failure after the fact. For `launchNavigation` we want to decide cache-vs-fetch *before* making a network attempt, to give a clean UX rather than a spinner followed by a toast. The Dio-error approach is a valid fallback inside a retry loop but not a pre-check.

```yaml
# pubspec.yaml addition
dependencies:
  internet_connection_checker_plus: ^3.0.1
```

Usage in `navigation_launch_util.dart`:
```dart
import 'package:internet_connection_checker_plus/internet_connection_checker_plus.dart';

final bool isOnline = await InternetConnection().hasInternetAccess;
if (isOnline) {
  // fetch fresh from /valhalla/navigate
} else {
  // load cached NavigationCacheEntity from ObjectBox
}
```

---

### ObjectBox: No New Package, New Entity

**Do not add a new package.** ObjectBox 5.3.1 (already in pubspec) supports `PropertyType.flex`, which stores any `Map<String, dynamic>` or `List<dynamic>` as a FlexBuffer binary blob. This is the correct approach for caching the `NavigateResponse` JSON.

**Add a new entity class: `NavigationCacheEntity`**

`NavigateResponse` contains:
- `List<NavigateManeuver> maneuvers` — list of maps
- `List<List<double>> shape` — nested list of doubles

`PropertyType.flex` handles both types natively (confirmed: ObjectBox Context7 docs show `Map<String, dynamic>?` and the FlexBuffer spec supports nested lists). The practical approach is to serialize `NavigateResponse.toJson()` and store the resulting `Map<String, dynamic>` in a single `flex` field, then deserialize on read with `NavigateResponse.fromJson()`. This avoids introducing any new relations or normalized tables.

```dart
// app/lib/entities/navigation_cache_entity.dart
import 'package:objectbox/objectbox.dart';

@Entity()
class NavigationCacheEntity {
  @Id()
  int obxId = 0;

  /// Matches TrailEntity.id (PocketBase record ID).
  @Index()
  @Unique(onConflict: ConflictStrategy.replace)
  String trailId;

  /// Full NavigateResponse serialized to JSON via NavigateResponse.toJson().
  /// Stored as FlexBuffer (ObjectBox PropertyType.flex).
  @Property(type: PropertyType.flex)
  Map<String, dynamic>? responseJson;

  /// UTC timestamp of when this cache entry was written.
  /// Used for staleness display ("cached N days ago") if needed.
  @Property(type: PropertyType.dateUtc)
  DateTime cachedAt;

  NavigationCacheEntity({
    required this.trailId,
    required this.cachedAt,
    this.responseJson,
  });
}
```

After adding this entity, run `dart run build_runner build` to regenerate `objectbox.g.dart`. The generator will assign a new entity ID/UID automatically.

**Retrieval pattern:**
```dart
final box = ref.read(objectBoxProvider).store.box<NavigationCacheEntity>();
final cached = box.query(NavigationCacheEntity_.trailId.equals(trailId)).build().findFirst();
if (cached?.responseJson != null) {
  return NavigateResponse.fromJson(cached!.responseJson!);
}
```

**Write pattern (at trail download time):**
```dart
final response = await fetchNavigationFromApi(trail);
final box = store.box<NavigationCacheEntity>();
box.put(NavigationCacheEntity(
  trailId: trail.id,
  cachedAt: DateTime.now().toUtc(),
  responseJson: response.toJson(),
));
```

---

## Integration Points

### 1. `launchNavigation` in `navigation_launch_util.dart`

Current flow (online-only):
```
guard GPS → build shape → POST /valhalla/navigate → parse → push screen
```

New flow:
```
guard GPS → check InternetConnection().hasInternetAccess
  ├─ online → POST /valhalla/navigate → parse → (write to cache) → push screen
  └─ offline → load NavigationCacheEntity by trail.id
                 ├─ hit  → NavigateResponse.fromJson → push screen
                 └─ miss → showError("No cached navigation for offline use") → return
```

The cache write on online success is a side-effect: `box.put(...)` after successful parse, before `context.push`. This keeps the cache warm after every online session, requiring no separate "download" step.

### 2. ObjectBox `objectbox.g.dart` regeneration

Adding `NavigationCacheEntity` requires re-running codegen. The existing `objectbox.g.dart` lists entity IDs/UIDs — the generator extends this list without touching existing entities. No migration step needed; ObjectBox handles schema evolution automatically for new entities.

### 3. `objectbox_store_provider.dart`

No change needed. `NavigationCacheEntity` is accessed directly via `store.box<NavigationCacheEntity>()`. The provider already exposes the `Store` instance.

### 4. `TrailEntity` — no change

The `NavigationCacheEntity` is linked to a trail by `trailId` (string, same as `TrailEntity.id`). A `ToOne<TrailEntity>` relation is deliberately avoided — the navigation cache should survive trail entity replacement without cascading deletes, and query-by-string-ID is sufficient for this use case.

---

## What NOT to Add

| Package | Reason to Skip |
|---------|---------------|
| `connectivity_plus` | Only checks network interface presence, not actual reachability. Misleading for offline-detection in low-signal outdoor environments. If already added by another team member, it can co-exist but should NOT be used for the online/offline gate in `launchNavigation`. |
| `hive` / `shared_preferences` | ObjectBox is already in use for the same purpose (caching trail data). Adding a second persistence layer for navigation cache creates split-brain risk and unnecessary dependency. |
| `sqflite` | Same reason as hive — redundant. |
| `objectbox_sync` | Not needed; sync is a paid ObjectBox feature for cross-device replication. Local cache is offline-device-only. |
| `flutter_cache_manager` | Designed for file/binary caching (images, PDFs), not structured JSON objects. Overkill and wrong abstraction. |
| `dio_retry` / `pretty_dio_logger` | Out of scope — no changes to the Dio layer are needed for offline navigation. |

---

## Version Constraints Summary

| Package | Action | Version Constraint | Dart SDK compat | Conflicts |
|---------|--------|--------------------|-----------------|-----------|
| `internet_connection_checker_plus` | ADD | `^3.0.1` | `>=2.15.0 <4.0.0` | None |
| `objectbox` | EXISTING (no change) | `^5.3.1` | — | — |
| `objectbox_generator` | EXISTING (no change) | `^5.3.1` (dev) | — | — |

The project's Dart SDK constraint is `^3.11.5` (i.e., `>=3.11.5 <4.0.0`). Both the existing ObjectBox constraint and the new `internet_connection_checker_plus ^3.0.1` fit within this range.

---

## Sources

- ObjectBox `PropertyType.flex` docs: Context7 `/objectbox/objectbox-dart` — HIGH confidence
- `internet_connection_checker_plus` 3.0.1 API: Context7 `/outdatedguy/internet_connection_checker_plus` + pub.dev API — HIGH confidence
- `connectivity_plus` 7.1.1 behavior notes: Context7 `/websites/pub_dev_packages_connectivity_plus` + pub.dev API — HIGH confidence
- Version constraints and SDK ranges: pub.dev REST API (verified 2026-06-14) — HIGH confidence
