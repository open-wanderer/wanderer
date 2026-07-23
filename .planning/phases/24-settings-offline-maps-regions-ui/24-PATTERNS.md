# Phase 24: Settings — Offline Maps/Regions UI - Pattern Map

**Mapped:** 2026-07-22
**Files analyzed:** 8 (5 new, 3 modified)
**Analogs found:** 8 / 8

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `app/lib/routes/settings_offline_regions_screen.dart` (NEW) | component/route (screen) | CRUD + streaming (download progress) | `app/lib/routes/settings_categories_screen.dart` | exact (list + toggle + confirm-dialog + toast structure) |
| `app/lib/provider/region/region_provider.dart` — add `regionListProvider` (MODIFIED) | provider (sync snapshot notifier) | CRUD (ObjectBox read) | `app/lib/provider/trail/trail_library_provider.dart` | exact (identical `@riverpod class ... build() => box.getAll()...` shape) |
| `app/lib/provider/region/tile_repository_provider.dart` — add `deleteDemPackage` notifier method (MODIFIED) | provider (async action notifier method) | request-response (async op + ephemeral state) | same file's existing `downloadVector`/`downloadDem` methods | exact |
| `app/lib/services/tile_repository_manager.dart` — add `deleteDemPackage` method (MODIFIED) | service | CRUD + file-I/O | same file's existing `deleteRegion` method (lines 324-366) | exact (same file, adjacent method, narrower scope) |
| `app/lib/util/byte_format_util.dart` (NEW) | utility | transform | no direct analog — pure function, trivial | role-match only |
| `app/lib/util/region_disk_usage_util.dart` (NEW) | utility | file-I/O (aggregation) | `app/lib/util/region_file_path.dart` (path builders it must call through) + `app/lib/util/disk_space_util.dart` (sibling byte-oriented util) | role-match |
| `app/lib/routes/settings_screen.dart` — add new entry ListTile (MODIFIED) | route (list-of-links screen) | request-response | existing sibling ListTiles for "Categories"/"Account" in same file | exact |
| `app/lib/provider/router_provider.dart` — add nested `GoRoute` under `/settings` (MODIFIED) | route/config | request-response | existing `/settings/categories` and `/settings/categories/subcategories` route registrations (lines ~192-234) | exact |

## Pattern Assignments

### `app/lib/routes/settings_offline_regions_screen.dart` (NEW screen)

**Analog:** `app/lib/routes/settings_categories_screen.dart` (secondary: `settings_subcategories_screen.dart` for the searchable/filtered-list variant)

**Imports pattern** (`settings_categories_screen.dart` lines 1-20):
```dart
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/async_loader.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/toast_provider.dart';
// ... + domain-specific provider/model imports
```
For the new screen: swap category/subcategory provider imports for `regionListProviderProvider`, `tileRepositoryStatusProvider`, `RegionEntity`, `RegionStatus`, `CatalogStatus`, plus `app/lib/components/base/wanderer_searchbar.dart` (not used by either analog — first search-box consumer among Settings screens, but already used elsewhere, e.g. `location_search_screen.dart`).

**Screen shape — `ConsumerStatefulWidget` with an AppBar back-arrow (lines 30-36, 153-160):**
```dart
class SettingsCategoriesScreen extends ConsumerStatefulWidget {
  const SettingsCategoriesScreen({super.key});
  @override
  ConsumerState<SettingsCategoriesScreen> createState() => _SettingsCategoriesScreenState();
}
// ...
return Scaffold(
  appBar: AppBar(
    leading: IconButton(icon: const Icon(Icons.arrow_back), onPressed: () => context.pop()),
    title: Text(l10n.categories),
  ),
  body: Column(children: [ /* ... */ ]),
);
```
Copy this shape verbatim for `OfflineRegionsScreen`; title becomes "Offline Maps/Regions" per UI-SPEC.

**Error-toast-only save wrapper (`_save`, lines 60-79) — reuse verbatim pattern for DEM-toggle-off and retry failures:**
```dart
Future<void> _save(Future<void> Function() op) async {
  final l10n = AppLocalizations.of(context)!;
  try {
    await op();
  } catch (_) {
    if (!mounted) return;
    ref.read(toastProvider.notifier).add(
      ToastMessage(
        type: ToastType.error,
        icon: FontAwesomeIcons.circleExclamation,
        text: l10n.error_saving_settings, // swap for a region-specific key if desired
      ),
    );
  }
}
```

**Confirm-before-destructive-action dialog (D-02, lines 472-495) — reuse verbatim shape for full-region delete:**
```dart
final confirmed = await showDialog<bool>(
  context: context,
  builder: (dialogContext) => AlertDialog(
    title: Text(l10n.settings_categories_confirm_disable_title), // -> "Delete {region name}?"
    content: Text(l10n.settings_categories_confirm_disable_body(count)), // -> region-delete body copy
    actions: [
      TextButton(onPressed: () => Navigator.of(dialogContext).pop(false), child: Text(l10n.cancel)),
      TextButton(onPressed: () => Navigator.of(dialogContext).pop(true), child: Text(l10n.settings_categories_confirm_disable_confirm)),
    ],
  ),
);
if (confirmed != true) return;
if (!mounted) return;
await _save(() => ref.read(tileRepositoryStatusProvider.notifier).delete(region.id));
```
Note: unlike the category dialog's 3-action ("View trails"/"Cancel"/confirm), the region-delete dialog per UI-SPEC copy only needs 2 actions (Cancel/Delete) — drop the middle action, keep the rest of the structure.

**Row layout — `InkWell` + `Padding(16)` + `Row` with leading icon, `Expanded` name column, trailing action (lines 348-421):**
```dart
return InkWell(
  key: ValueKey(category.id),
  onTap: /* navigate or null */,
  child: Padding(
    padding: const EdgeInsets.all(16.0),
    child: Row(children: [
      /* leading icon/avatar */,
      const SizedBox(width: 12),
      Expanded(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(category.displayName(locale),
              overflow: TextOverflow.ellipsis,
              style: Theme.of(context).textTheme.bodyLarge!.copyWith(fontWeight: FontWeight.bold)),
            // secondary content (chips / captions / banners) below
          ],
        ),
      ),
      Switch(value: isVisible, activeThumbColor: activeColor, onChanged: (v) => _onToggle(category, v)),
    ]),
  ),
);
```
Region row: name in the same bold `bodyLarge` style, size-breakdown/status/update-banner as secondary lines below the name (per UI-SPEC Typography table), trailing widget swapped per state (download/pause/resume/retry `IconButton`, or the DEM `Switch` living inside the expanded content per SETUI-04, not as the row's sole trailing widget — the row needs multiple trailing actions, so an explicit `Row`/`Wrap` of `IconButton`s replaces the analog's single `Switch`).

**Empty state (secondary analog, `settings_subcategories_screen.dart` lines 217-241):**
```dart
Widget _buildEmptyState(AppLocalizations l10n) {
  return Center(
    child: Padding(
      padding: const EdgeInsets.symmetric(horizontal: 32),
      child: Column(mainAxisSize: MainAxisSize.min, children: [
        Text(l10n.settings_categories_empty_title, style: Theme.of(context).textTheme.titleMedium, textAlign: TextAlign.center),
        const SizedBox(height: 8),
        Text(l10n.settings_categories_empty_body, style: Theme.of(context).textTheme.bodyMedium, textAlign: TextAlign.center),
      ]),
    ),
  );
}
```
Reuse for both empty states (search-no-match and catalog-empty per UI-SPEC copy table) — two variants of the same widget shape with different copy.

**AsyncLoader combine pattern (lines 104-151)** — NOT directly reused for the primary list (Pitfall 4 in RESEARCH.md explicitly says do NOT gate the whole screen behind a blocking `AsyncLoader` on catalog fetch), but still the correct widget if/when a genuine full-screen error state is needed (fresh install + fetch fails, zero cached regions):
```dart
AsyncLoader<T>(asyncValue: combined, mockData: /* ... */, builder: (data) { /* ... */ });
```
Use `app/lib/components/async_loader.dart`'s `AsyncLoader` + built-in `WandererError` only for that one edge case; render `regionListProvider`'s synchronous snapshot unconditionally otherwise.

---

### `app/lib/provider/region/region_provider.dart` — new `regionListProvider`

**Analog:** `app/lib/provider/trail/trail_library_provider.dart` (full file, 42 lines — read in full, no analog needed for `deleteTrail`, only the `build()` shape)

**Core pattern (lines 11-20), copy verbatim structure, change entity/sort key:**
```dart
@riverpod
class TrailLibraryNotifier extends _$TrailLibraryNotifier {
  @override
  List<Trail> build() {
    final store = ref.watch(objectBoxProvider);
    final box = store.box<TrailEntity>();
    final trails = box.getAll().map((t) => t.toModel()).toList();
    trails.sort((a, b) => b.created.compareTo(a.created));
    return trails;
  }
}
```
For `RegionListNotifier`: `box.getAll()` returns `RegionEntity` directly (no `.toModel()` transform needed unless one already exists), sort alphabetically by `name` (D-09) instead of by `created` descending. Per RESEARCH.md Pitfall 2, callers must `ref.invalidate(regionListProviderProvider)` after every `TileRepositoryStatus` mutation — this provider itself does not need mutation methods (unlike `TrailLibraryNotifier.deleteTrail`), since all mutations happen via `TileRepositoryStatus`/`TileRepositoryManager`.

---

### `app/lib/services/tile_repository_manager.dart` — new `deleteDemPackage` method (D-01)

**Analog:** same file's existing `deleteRegion` (lines 324-366) — copy the shape, narrow scope to DEM only:
```dart
Future<void> deleteRegion(String regionId) async {
  final id = assertValidRegionId(regionId);

  for (final entry in _activeCancelTokens.entries.toList()) {
    if (entry.key == '$id:vector' || entry.key == '$id:dem') {
      entry.value.cancel('deleted');
    }
  }

  final region = _regionById(id);
  if (region == null) return;

  final vectorPackage = region.vectorPackage.target;
  final demPackage = region.demPackage.target;

  _store.runInTransaction(TxMode.write, () {
    final packageBox = _store.box<DownloadedTilePackageEntity>();
    if (vectorPackage != null) packageBox.remove(vectorPackage.obxId);
    if (demPackage != null) packageBox.remove(demPackage.obxId);

    region.vectorPackage.target = null;
    region.demPackage.target = null;
    region.lastDownloadedVersion = null;
    _store.box<RegionEntity>().put(region);
  });

  // Best-effort, outside the transaction: a missing file is never fatal.
  final root = (await getApplicationDocumentsDirectory()).path;
  for (final finalPath in [regionVectorPath(root, id), regionDemPath(root, id)]) {
    for (final candidate in [finalPath, '$finalPath.part']) {
      final file = File(candidate);
      if (file.existsSync()) file.deleteSync();
    }
  }

  final dir = Directory(regionStorageDir(root, id));
  if (dir.existsSync() && dir.listSync().isEmpty) dir.deleteSync();
}
```
`deleteDemPackage(regionId)` must: (1) cancel only the `'$id:dem'` token — not `'$id:vector'` (per CONTEXT.md's code-context note, lines 80); (2) remove only `demPackage` from `DownloadedTilePackageEntity` box and clear `region.demPackage.target` — leave `region.vectorPackage`/its file untouched; (3) delete only `regionDemPath(root, id)` + its `.part` variant, never touch `regionVectorPath`; (4) still route all paths through `assertValidRegionId`/`regionDemPath` (never hand-build paths) per the ASVS V5 note in RESEARCH.md.

---

### `app/lib/provider/region/tile_repository_provider.dart` — new `deleteDemPackage` notifier method

**Analog:** same file's existing `downloadVector`/`downloadDem` methods (lines 33-79+) for the ephemeral-state-wrapping shape; structurally closer to a delete than a download, so mirror the shape but without progress tracking:
```dart
Future<void> downloadVector(String regionId) async {
  if (state[regionId]?.status == RegionStatus.downloading) return;
  state = {
    ...state,
    regionId: (state[regionId] ?? const RegionDownloadState()).copyWith(
      status: RegionStatus.downloading,
      vectorProgress: 0,
    ),
  };
  try {
    await ref.read(tileRepositoryManagerProvider).startVectorDownload(
      regionId,
      onProgress: (received, total) { /* update state */ },
    );
  } finally {
    state = {...state}..remove(regionId);
  }
}
```
`deleteDemPackage(regionId)` should call `ref.read(tileRepositoryManagerProvider).deleteDemPackage(regionId)` inside a try/finally that clears/updates the ephemeral map entry, then the screen's caller invalidates `regionListProviderProvider` (Pitfall 2) since this notifier's `state` map is ephemeral progress only, not the authoritative `RegionEntity` data the list reads.

---

### `app/lib/util/byte_format_util.dart` (NEW)

**Analog:** none in-repo (first byte-formatting utility) — RESEARCH.md's Code Examples section already supplies the exact function to use verbatim:
```dart
String formatBytes(int bytes) {
  const kb = 1024;
  const mb = kb * 1024;
  const gb = mb * 1024;
  if (bytes >= gb) return '${(bytes / gb).toStringAsFixed(1)} GB';
  if (bytes >= mb) return '${(bytes / mb).toStringAsFixed(1)} MB';
  if (bytes >= kb) return '${(bytes / kb).toStringAsFixed(1)} KB';
  return '$bytes B';
}
```

---

### `app/lib/util/region_disk_usage_util.dart` (NEW)

**Analog:** `app/lib/util/region_file_path.dart` (path-builder functions this MUST call through, never re-derive), `app/lib/util/disk_space_util.dart` (sibling byte-oriented utility, same directory/naming convention).

**Pattern (per RESEARCH.md Pitfall 1 — must read actual on-disk bytes, not trust `sizeBytesOnDisk`):**
```dart
// Pseudocode shape, following region_file_path.dart's existing builders:
int regionDiskUsageBytes(String root, RegionEntity region) {
  final id = assertValidRegionId(region.id);
  int total = 0;
  for (final (status, finalPath) in [
    (region.vectorPackage.target?.status, regionVectorPath(root, id)),
    (region.demPackage.target?.status, regionDemPath(root, id)),
  ]) {
    final finalFile = File(finalPath);
    final partFile = File('$finalPath.part');
    if (finalFile.existsSync()) {
      total += finalFile.lengthSync();
    } else if (partFile.existsSync()) {
      total += partFile.lengthSync();
    }
  }
  return total;
}
```
Must reuse `assertValidRegionId`/`regionVectorPath`/`regionDemPath` from `region_file_path.dart` — never construct paths independently (security constraint, see Shared Patterns below).

---

### `app/lib/routes/settings_screen.dart` — new entry ListTile

**Analog:** existing sibling `ListTile`s in the same file for "Categories"/"Account" entries — copy the exact `ListTile` shape (icon size 18, `FontAwesomeIcons.map` per UI-SPEC, `onTap: () => context.push('/settings/regions')`).

### `app/lib/provider/router_provider.dart` — new nested `GoRoute`

**Analog:** existing `/settings/categories` and `/settings/categories/subcategories` `GoRoute` registrations (~lines 192-234) — copy the nesting shape under the parent `/settings` route.

---

## Shared Patterns

### Error-toast-only save wrapper
**Source:** `app/lib/routes/settings_categories_screen.dart` lines 60-79 (`_save` helper) and `app/lib/provider/toast_provider.dart`
**Apply to:** the new screen's DEM-toggle-off action (D-01), region-delete confirm action (D-02), retry action failures (D-03), and catalog-refresh failures (RESEARCH.md Pitfall 4 — toast only, never full-screen error, when the list already has cached data).
```dart
ref.read(toastProvider.notifier).add(
  ToastMessage(type: ToastType.error, icon: FontAwesomeIcons.circleExclamation, text: /* message */),
);
```

### Confirm-before-destructive-action dialog
**Source:** `app/lib/routes/settings_categories_screen.dart` lines 472-503 (`_onToggleOff`'s dialog), `app/lib/routes/settings_account_screen.dart`'s delete-account dialog (per UI-SPEC's own citation)
**Apply to:** D-02's full-region-delete confirmation only (NOT the DEM toggle-off, which is explicitly immediate/no-dialog per D-01).

### Synchronous ObjectBox snapshot provider
**Source:** `app/lib/provider/trail/trail_library_provider.dart` (full file)
**Apply to:** `regionListProvider` — read-once-then-invalidate pattern, not a reactive box-stream (RESEARCH.md explicitly rejects the stream-watch alternative as unprecedented in this codebase).

### Region id / file path safety
**Source:** `app/lib/util/region_file_path.dart` (`assertValidRegionId`, `regionVectorPath`, `regionDemPath`, `regionStorageDir`) — used identically by `deleteRegion` in `tile_repository_manager.dart` (lines 324-366)
**Apply to:** the new `deleteDemPackage` method AND the new `region_disk_usage_util.dart` — both MUST route every path through these existing builders, never string-concatenate a region id into a path directly (ASVS V5 mitigation already established).

### Selective CancelToken cancellation
**Source:** `app/lib/services/tile_repository_manager.dart`'s `deleteRegion` (lines 327-331) — cancels only tokens matching `'$id:vector'` or `'$id:dem'`
**Apply to:** `deleteDemPackage` — must cancel only `'$id:dem'`, leaving any in-flight vector download untouched.

## No Analog Found

None — every file in scope has at least a role-match analog in the existing codebase.

## Metadata

**Analog search scope:** `app/lib/routes/`, `app/lib/provider/region/`, `app/lib/provider/trail/`, `app/lib/services/`, `app/lib/util/`
**Files scanned:** `settings_categories_screen.dart`, `settings_subcategories_screen.dart`, `trail_library_provider.dart`, `tile_repository_manager.dart`, `tile_repository_provider.dart`, `region_provider.dart`, `region_file_path.dart`, `disk_space_util.dart`
**Pattern extraction date:** 2026-07-22
