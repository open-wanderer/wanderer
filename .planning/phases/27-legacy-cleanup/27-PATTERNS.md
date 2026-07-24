# Phase 27: Legacy Cleanup - Pattern Map

**Mapped:** 2026-07-24
**Files analyzed:** 6 (5 modified, 1 deleted outright + 2 generated siblings)
**Analogs found:** N/A — this phase is pure subtraction; each target file IS its own analog (verbatim current-state excerpts below are the deletion source of truth, not a separate file to imitate)

## Special note on methodology

This phase is code **deletion**, not new-file creation. There is no meaningful "closest analog" search to run — the planner needs the exact current-state code (with line context) for each deletion/edit target so plans can specify precise removal boundaries. Accordingly, this PATTERNS.md replaces "Pattern Assignments" (copy-from patterns) with "Deletion Boundaries" (copy-from-then-remove patterns) for each file. No new architectural patterns are being introduced anywhere in this phase.

## File Classification

| File | Role | Data Flow | Change Type | Notes |
|------|------|-----------|--------------|-------|
| `app/lib/services/trail_download_service.dart` | service | file-I/O | surgical edit | delete 3 methods + wiring; keep photo/nav-cache methods |
| `app/lib/services/download_notification_service.dart` | service | event-driven (local notifications) | conditional edit | delete `showGenerating()` only, verified dead |
| `app/lib/models/map_cell.dart` (+ `.freezed.dart`/`.g.dart`) | model | request-response (deserialization) | delete whole file | orphaned once tile methods are gone |
| `app/lib/entities/trail_entity.dart` | model (ObjectBox entity) | CRUD | surgical edit | delete `pmTiles`/`demPmTiles` field + mapping refs |
| `app/lib/models/trail.dart` | model (freezed) | CRUD | surgical edit | delete `pmTiles`/`demPmTiles` freezed fields |
| `app/lib/provider/trail/trail_download_state_provider.dart` | provider (Riverpod notifier) | event-driven | surgical edit | remove `onGeneratingChanged` wiring + `showGenerating` call site |
| `app/lib/objectbox-model.json`, `objectbox.g.dart`, `trail.freezed.dart`, `trail.g.dart`, `map_cell.freezed.dart`, `map_cell.g.dart` | generated | — | regenerate | `dart run build_runner build --delete-conflicting-outputs`, single pass at the end |

## Deletion Boundaries

### `app/lib/services/trail_download_service.dart` (service, file-I/O)

**Current full-file state confirmed via direct read, 2026-07-24.**

**Import to delete** (line 12):
```dart
import 'package:wanderer/models/map_cell.dart';
```

**`downloadTrail()` signature — delete `onGeneratingChanged` param** (lines 41-46):
```dart
Future<void> downloadTrail(
  Trail trail, {
  CancelToken? cancelToken,
  void Function(bool isGenerating)? onGeneratingChanged,   // DELETE this line
  void Function(int done, int total)? onProgress,
}) async {
```

**Internal generating-state machinery to delete** (lines 92-102):
```dart
var isGenerating = false;                                  // DELETE
void report() {
  if (totalPoints == null || isGenerating) return;          // isGenerating ref: simplify to `if (totalPoints == null) return;`
  onProgress?.call(currentPoints, totalPoints!);
}

void handleGeneratingChanged(bool generating) {             // DELETE whole function
  isGenerating = generating;
  if (!generating) report();
  onGeneratingChanged?.call(generating);
}
```

**Tile future + wiring to delete** (lines 119-136, 163, 167-168):
```dart
(List<String>, List<String>)? tileResult;                   // DELETE

() async {
  tileResult = await _downloadMapTiles(                      // DELETE this whole future block
    trail,
    trailDir,
    cancelToken: cancelToken,
    onCellTotal: (cellCount) {
      totalPoints = (cellCount + photoTotal) * _pointsPerUnit;
      report();
    },
    onGeneratingChanged: handleGeneratingChanged,
    onCellPointsDelta: onCellPointsDelta,
  );
}(),

// ...

final (cellPaths, demCellPaths) = tileResult!;               // DELETE
entity.pmTiles = cellPaths;                                  // DELETE
entity.demPmTiles = demCellPaths;                             // DELETE
```

Note: `photoTotal` calc at lines 87-89 and `onCellPointsDelta` at 109-112 stay if still referenced by photo/waypoint paths — re-verify at edit time since `onCellPointsDelta` was tile-only (delete it too, along with the `totalPoints` cell-count contribution logic, since no cells exist anymore — `totalPoints` becomes purely photo-count based).

**Methods to delete outright** (lines 212-403):
```dart
Future<(List<String>, List<String>)> _downloadMapTiles(...) async { ... }   // lines 215-362, DELETE
Future<MapCellInfoList> _fetchCellList(...) async { ... }                    // lines 364-374, DELETE
Future<MapCellStatusResponse> _pollUntilReady(...) async { ... }             // lines 376-403, DELETE
```

**Keep unchanged** (D-04, lines 405-444 and 446-515):
```dart
// KEEP: _downloadTracked — shared byte-progress helper, still used by _downloadPhotos
Future<void> _downloadTracked(
  String url,
  String savePath, {
  required CancelToken? cancelToken,
  required void Function(double fraction) onFraction,
}) async { /* unchanged */ }

// KEEP: _downloadPhotos, _downloadWaypointPhotos — unchanged
```

**Static fields to review** (lines 22-23): `_pollInterval`/`_pollTimeout` are only used by `_pollUntilReady` — delete alongside it.

---

### `app/lib/services/download_notification_service.dart` (service, event-driven)

**Method to delete, conditional on D-07 verification** (lines 37-67):
```dart
/// Indeterminate state shown while a trail's map tiles are still being
/// generated on the server — there's no measurable percentage for this
/// phase, so it's a distinct spinner rather than a fake progress value.
Future<void> showGenerating(String trailName) async {
  await _ensureInitialized();
  final androidDetails = AndroidNotificationDetails(
    _channelId,
    _channelName,
    channelDescription: 'Trail tile download progress',
    importance: Importance.low,
    priority: Priority.low,
    showProgress: true,
    indeterminate: true,
    ongoing: true,
    autoCancel: false,
    onlyAlertOnce: true,
  );
  const iosDetails = DarwinNotificationDetails(
    presentAlert: false,
    presentBadge: false,
    presentSound: false,
  );
  await _plugin.show(
    _notificationId,
    trailName,
    'Generating map tiles...',
    NotificationDetails(android: androidDetails, iOS: iosDetails),
  );
}
```

**Verification command (re-run post-edit of `trail_download_state_provider.dart`, per D-07/Pitfall 4):**
```bash
grep -rn "showGenerating" app/lib --include="*.dart"
# Expected AFTER trail_download_state_provider.dart edit: zero hits.
# If any hit remains outside this method's own definition, do NOT delete — keep method, remove only the trail-path call site.
```

**Keep unchanged:** `showProgress`, `showAggregateProgress`, `showSuccess`, `showError`, `dismiss`, `_ensureInitialized` — all shared/live, no changes.

---

### `app/lib/models/map_cell.dart` (model, request-response) — DELETE WHOLE FILE

Sole purpose: deserializes `/map/cells` and `/map/cells/{key}/status` responses (`MapCellInfoList`, `MapCellInfo`, `MapCellStatusResponse`, `MapCellStatus`), consumed exclusively by the three deleted `trail_download_service.dart` methods. Delete this file plus its generated siblings:
```bash
rm app/lib/models/map_cell.dart
rm app/lib/models/map_cell.freezed.dart
rm app/lib/models/map_cell.g.dart
```
`build_runner` will report these `.freezed.dart`/`.g.dart` files as already gone (or clean them if stale) on the final regeneration pass — safe to delete all three manually first since the source annotations driving generation cease to exist.

**Pre-deletion verification:**
```bash
grep -rln "map_cell\|MapCellInfoList\|MapCellInfo\|MapCellStatusResponse\|MapCellStatus" app/lib app/test --include="*.dart"
# Expected: only trail_download_service.dart (import) and map_cell.dart's own generated siblings.
```

---

### `app/lib/entities/trail_entity.dart` (ObjectBox entity, CRUD)

**Fields to delete** (line 41-42):
```dart
List<String> pmTiles = [];
List<String> demPmTiles = [];
```

**Mapping reference to delete** (`TrailEntity.fromModel`, lines 158-159):
```dart
pmTiles: pmTiles,
demPmTiles: demPmTiles,
```

Pitfall 2 applies: edit this file, `trail.dart`, and any `toModel()`/reverse-mapping reference together before running `build_runner` — do not regenerate mid-edit.

---

### `app/lib/models/trail.dart` (freezed model, CRUD)

**Fields to delete** (lines 102-103):
```dart
@Default([]) List<String> pmTiles,
@Default([]) List<String> demPmTiles,
```

---

### `app/lib/provider/trail/trail_download_state_provider.dart` (provider, event-driven)

**`downloadTrail()` call site — delete `onGeneratingChanged` param entirely** (lines 206-231, keep `onProgress` only):
```dart
try {
  await trailDownloadService.downloadTrail(
    trail,
    onGeneratingChanged: (isGenerating) {                    // DELETE this whole param + callback body
      // WR-01: when packages are selected, keep the unified id-42
      // aggregate notification stable through tile generation instead
      // of reverting to showGenerating's plain trail-name copy
      // (GUARD-03 / D-10). The 0-region path below is unchanged.
      if (!isGenerating) return;
      if (hasSelectedPackages) {
        updateAggregate();
      } else {
        notificationService.showGenerating(trail.name).catchError((_) {});
      }
    },
    onProgress: (done, total) {                               // KEEP — unchanged
      if (hasSelectedPackages) {
        lastTrailFraction = total > 0 ? (done / total).clamp(0, 1) : 0;
        updateAggregate();
      } else {
        notificationService
            .showProgress(trail.name, done, total)
            .catchError((_) {});
      }
    },
  );
```

**KEEP, unchanged (do NOT touch — these are the 26-04/26-05 invariants under regression risk per canonical_refs):**
- `hasSelectedPackages` definition (lines 118-119) — still used at lines 200-201 (`if (hasSelectedPackages) { updateAggregate() } else { showProgress(...) }`) on `downloadTrail`'s START — that branch is unrelated to `onGeneratingChanged` and must survive.
- `vectorLatched`/`demLatched` monotonic latch maps and `updateAggregate()` (lines 145-174).
- The single `state = {...state}..remove(trail.id)` in `finally` (line 274).
- The single `ref.invalidate(regionListNotifierProvider)` (line 293).
- `trailSucceeded`-gated deferred `showSuccess` (lines 298-312).

**Post-edit re-verification commands:**
```bash
grep -n "onGeneratingChanged\|showGenerating\|isGenerating" app/lib/provider/trail/trail_download_state_provider.dart
# Expected after edit: zero hits.
flutter analyze app/lib/provider/trail/trail_download_state_provider.dart
flutter test   # re-run 26-04/26-05 regression coverage
```

---

## Shared Patterns

### Verify-before-delete grep (applies to every deletion in this phase)
**Source:** `27-RESEARCH.md` Pattern 2 / Don't Hand-Roll table
**Apply to:** `pmTiles`/`demPmTiles`, `showGenerating`, `map_cell.dart` symbols
```bash
grep -rn "<symbol>" app/lib app/test --include="*.dart"
```
Run fresh immediately before each deletion, not from CONTEXT.md's/RESEARCH.md's cached results (Pitfall 4).

### Generated-code regeneration (single pass, end of phase)
**Source:** `27-RESEARCH.md` Standard Stack / Pitfall 2
**Apply to:** `trail_entity.dart`, `trail.dart`, `map_cell.dart` deletion
```bash
cd app
dart run build_runner build --delete-conflicting-outputs
```
Only run once all source edits (entity + freezed model + mapping extension) are complete together — running mid-edit produces transient generator errors that look like real bugs (Pitfall 2).

### Import-cleanliness check after file deletion
**Source:** `27-RESEARCH.md` Anti-Patterns
**Apply to:** `trail_download_service.dart` after `map_cell.dart` deletion
```bash
flutter analyze app/lib/services/trail_download_service.dart
# Expect zero unused_import warnings after removing the map_cell.dart import
```

## No Analog Found

Not applicable — every file in this phase's scope is itself the deletion target; there is no separate "analog" file being imitated. The excerpts above are extracted directly from each target file's current state (verified 2026-07-24) and serve as the planner's exact deletion/edit boundaries.

## Out of Scope (explicit, do not plan for)

- **CLEAN-02 / one-time cleanup sweep:** Descoped per CONTEXT.md D-05. No "startup sweep" pattern search was performed because none is being built in this phase — do not create a plan task for it. (Note: the phase brief mentions looking for a startup-task analog for a sweep; CONTEXT.md and RESEARCH.md both explicitly confirm this was cut by the user, not merely deferred — see D-05 verbatim: "No sweep necessary. App is not in production.")
- **`app/lib/components/trail/missing_coverage_sheet.dart`:** Confirmed Phase 26 region-download UI, not a legacy remnant — do NOT touch (imports `tile_repository_provider.dart`/`region_entity.dart`, zero references to `pmTiles`/`downloadTrail`/`MapCellInfoList`). Its currently-uncommitted whitespace-only diff (per `git status`) is unrelated to this phase; leave as-is or handle as a separate stray-formatting commit, per RESEARCH.md's Open Question 1.
- **`tile_repository_provider.dart`/`tile_repository_manager.dart`:** Region-based system, structurally independent, unaffected by this phase.

## Metadata

**Analog search scope:** N/A (deletion phase — no analog search performed; direct reads of all 4 primary target files + grep verification of dependents)
**Files scanned/read directly:** `trail_download_service.dart` (full), `download_notification_service.dart` (full), `trail_download_state_provider.dart` (lines 100-324), `trail_entity.dart`/`trail.dart` (grep-located field lines)
**Pattern extraction date:** 2026-07-24
