# Phase 36: Local-First Recording & Automatic Upload - Pattern Map

**Mapped:** 2026-08-02
**Files analyzed:** 13 (new/modified, Dart only — this phase is `app/lib/**` scoped)
**Analogs found:** 13 / 13

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `app/lib/entities/trail_entity.dart` (add owner/syncState/localId/localPhotos fields) | model (ObjectBox entity) | CRUD | itself (existing `savedByUserIds`/`photos` fields) | exact — additive edit |
| `app/lib/entities/waypoint_entity.dart` (fix synthetic id, add local key) | model (ObjectBox entity) | CRUD | itself (`localPhotos` already present) | exact — additive edit |
| `app/lib/provider/trail/trail_sync_provider.dart` (new — the drain) | service/provider (keepAlive notifier) | event-driven / batch | `app/lib/provider/trail/trail_download_state_provider.dart` (`DownloadingTrailIds`) | exact — same "in-flight id set + finally cleanup" shape |
| `app/lib/provider/trail/trail_save_provider.dart` (extend, not replace) | service | request-response / CRUD | itself — `createTrail`/`updateTrail` tag→trail→waypoint sequence is literally the drain's step order | exact |
| `app/lib/routes/trail_create_screen.dart` (`_onSave` three-way branch, waypoint stub fixes) | controller (screen event handler) | request-response | itself — current two-way `_onSave` branch (`:401-418`) | exact — extend existing branch |
| `app/lib/provider/profile/profile_trails_provider.dart` (rebuild local-first) | service/provider | CRUD (merge local+network) | `app/lib/provider/trail/trail_library_provider.dart` (`TrailLibraryNotifier`) for the local-read half; itself for the network half | role-match (hybrid — no single exact analog exists, this is the "genuinely new" merge work per RESEARCH.md) |
| `app/lib/util/local_photo_store_util.dart` (new — app-owned photo copy dir) | utility (file I/O) | file-I/O | `app/lib/services/trail_download_service.dart` (`_downloadPhotos`, app-doc-dir `library/<id>/` convention) | role-match — same app-owned-directory precedent, different trigger (copy vs download) |
| `app/lib/components/trail/trail_dropdown.dart` (split `isLocal` branch, hide download, block delete mid-drain) | component | request-response (UI event) | itself — `_allowDelete`/`downloadEnabled`/`_deleteTrail` (`:43-46`, `169-223`) | exact — same file, new branch |
| `app/lib/components/trail/trail_card.dart` / `trail_list_item.dart` (new `_SyncStatusChip`) | component | request-response (render) | itself — existing `_Chip`/badge rendering already branching on `trail.isLocal` | exact — new sibling widget, same slot pattern |
| `app/lib/main.dart` (add `WidgetsBindingObserver`, trigger drain) | provider/entry-point | event-driven | `app/lib/routes/navigation_screen.dart` (`didChangeAppLifecycleState`, `:593-602`) — the only existing lifecycle observer in the app | role-match — same observer mixin, new scope (app-root vs single screen) |
| Sign-out warning dialog (wherever sign-out is triggered — likely `settings`/`profile` screen) | component | request-response | `trail_dropdown.dart` `_confirmDelete` (`:179-201`) — title-less `AlertDialog` + Cancel/confirm `TextButton` pair | exact |
| `app/test/entities/trail_entity_test.dart` (extend for new fields) | test | CRUD | itself — existing pure `TrailEntity` construction tests | exact |
| `app/test/components/trail/trail_dropdown_delete_gate_test.dart` (update for split branch) | test | request-response | itself — existing source-level gate test | exact |

## Pattern Assignments

### `app/lib/entities/trail_entity.dart` (model, CRUD)

**Analog:** itself — extend the existing entity in place, following the `savedByUserIds` field's own documented style (doc-comment explaining the *why*, not just the *what*).

**Existing field pattern to copy** (`trail_entity.dart:44-60`):
```dart
List<String> photos = [];

/// Ids of the accounts that have this trail in their offline library.
/// ...
List<String> savedByUserIds = [];
```
New fields (owner, syncState, localId, localPhotos) should follow this exact style: plain field + a doc comment stating why it exists and what invisible invariant depends on it (per A3 in RESEARCH.md: `owner` is `String?`, singular — not a list like `savedByUserIds`, because authorship is 1:1 vs downloads' 1:N).

**`fromModel`/`toModel` round-trip pattern** (`trail_entity.dart:108-153`, `156-195`):
```dart
factory TrailEntity.fromModel(Trail trail) {
  final entity = TrailEntity(
    id: trail.id,
    name: trail.name,
    // ...
  );
  entity.dbDifficulty = trail.difficulty.index;
  // conditional expand-based population (waypoints, author, category)
  return entity;
}
```
```dart
extension TrailEntityMapping on TrailEntity {
  Trail toModel() {
    return Trail(
      id: id,
      // ...
      isLocal: true,
      localPhotos: photos,
      // ...
    );
  }
}
```
New fields must be threaded through **both** directions of this round trip — `isLocal: true` is hardcoded here (the landmine CONTEXT.md flags: it will now be true for both downloaded and unsynced rows).

---

### `app/lib/entities/waypoint_entity.dart` (model, CRUD)

**Analog:** itself — `localPhotos` field already exists at line 24; copy its shape for the new non-serialized local-key field on the **`Waypoint` model** (not the entity — the entity already round-trips fine).

**Non-serialized local-key precedent** (`app/lib/models/waypoint.dart:29-33`, cited in RESEARCH.md Pitfall 1):
```dart
// Pattern to copy for the new list-identity field:
// @JsonKey(includeFromJson: false, includeToJson: false)
// final String? marker;  /* or localPhotos-equivalent */
```
Read the exact lines before implementing:
```
Read app/lib/models/waypoint.dart around lines 20-40 for the marker/localPhotos JsonKey annotation shape.
```

**The two synthetic-id sites to fix** (both must become `id: ''`):
```dart
// trail_create_screen.dart:160 (manually-created waypoint stub)
final stub = Waypoint(
  id: DateTime.now().microsecondsSinceEpoch.toString(),  // → id: '', localKey: ...
  ...
);

// trail_create_screen.dart:194 (photo-EXIF waypoint)
id: '${now.microsecondsSinceEpoch}-${created.length}',  // → id: '', localKey: ...
```

---

### `app/lib/provider/trail/trail_sync_provider.dart` (NEW — service, event-driven/batch)

**Analog:** `app/lib/provider/trail/trail_download_state_provider.dart` (`DownloadingTrailIds`)

**In-flight-tracking shape to copy** (`trail_download_state_provider.dart:27-30`, `32-36`, `272-280`):
```dart
@Riverpod(keepAlive: true)
class DownloadingTrailIds extends _$DownloadingTrailIds {
  @override
  Set<String> build() => {};

  Future<void> download(Trail trail) async {
    if (state.contains(trail.id)) return;   // re-entry guard
    state = {...state, trail.id};
    try {
      // ... work
    } finally {
      state = {...state}..remove(trail.id);  // CR-01: always runs
    }
  }
}
```
The drain provider (`TrailSyncNotifier` per RESEARCH.md) must copy this exact shape, keyed on the trail's **local id** (not server `id`, which is empty pre-sync). This same `Set<String>` also becomes the signal `trail_dropdown.dart`'s delete-blocked-mid-drain check reads (D-14), mirroring how `isDownloading` gates `downloadEnabled` today (`trail_dropdown.dart:43-46`).

**Account-switch exclusion — must be a deliberate decision, not an omission** (`account_scope_invalidation.dart:14-16`):
```dart
/// - `downloadingTrailIdsProvider` / `tileRepositoryManagerProvider` stay:
///   invalidating them mid-download would desync in-flight download
///   bookkeeping from its `CancelToken`s.
```
The new drain provider needs the identical exclusion, documented with the identical reasoning, in `account_scope_invalidation.dart`'s doc comment and NOT added to `accountScopedProviders` (`account_scope_invalidation.dart:58-71`).

**Drain step sequence to reuse verbatim** — `app/lib/provider/trail/trail_save_provider.dart:34-105` (`_resolveTags` + `createTrail`'s tag→trail→waypoint loop) is literally the sequence D-05 specifies. The drain should call into this provider's existing methods (or a resumable variant of them) rather than re-implementing the `PUT /tag`/`PUT /trail/form`/`PUT /waypoint` calls:
```dart
// trail_save_provider.dart:34-44 — id.isEmpty as the resolve/create discriminator (D-06's rule already lives here)
Future<List<Tag>> _resolveTags(List<Tag> tags) async {
  final resolved = <Tag>[];
  for (final tag in tags) {
    if (tag.id != null && tag.id!.isNotEmpty) {
      resolved.add(tag);
    } else {
      resolved.add(await ref.read(tagProvider.notifier).create(tag.name));
    }
  }
  return resolved;
}
```
```dart
// trail_save_provider.dart:64-70 — the exact point (Pitfall 3) where the server id
// becomes available and must be written back to TrailEntity SYNCHRONOUSLY, before
// waypoint uploads begin, for SYNC-04's idempotency:
final response = await api.put('/trail/form', data: formData, queryParameters: {...});
var model = Trail.fromJson(response.data);
```
```dart
// trail_save_provider.dart:76-88 — per-waypoint try/catch that swallows individual
// failures into a boolean rather than aborting the whole loop; the drain's per-item
// resume-from-step logic is a stricter version of this same shape (id.isEmpty check
// instead of collect-and-report):
for (final waypoint in trail.expand?.waypointsViaTrail ?? const []) {
  try {
    createdWaypoints.add(await waypointNotifier.create(waypoint, authorId: authorId, trailId: model.id));
  } catch (e) {
    hadFailures = true;
  }
}
```

**Error handling / backoff (D-07)** — no existing backoff precedent in this codebase; this is genuinely new. Use the try/catch-per-step shape above but escalate to a `Failed` `syncState` after N attempts rather than swallowing indefinitely.

---

### `app/lib/routes/trail_create_screen.dart` — `_onSave` three-way branch

**Analog:** itself (`:401-469`)

**Current two-way branch to extend into three-way** (`trail_create_screen.dart:401-418`, cited verbatim in RESEARCH.md Code Example 1):
```dart
try {
  final result = trail.id.isEmpty
      ? await ref.read(trailSaveProvider.notifier).createTrail(
          updatedTrail, authorId: authorId, newPhotos: newPhotoFiles,
        )
      : await ref.read(trailSaveProvider.notifier).updateTrail(
          _originalTrail, updatedTrail,
          authorId: authorId, newPhotos: newPhotoFiles,
          removedPhotoFilenames: _removedServerPhotos,
        );
  // ... success handling
} catch (e) {
  ref.read(toastProvider.notifier).add(ToastMessage(
    type: ToastType.error,
    icon: FontAwesomeIcons.circleExclamation,
    text: l10n.error_saving_trail,
  ));
} finally {
  if (mounted) setState(() => _saving = false);
}
```
New third branch (trail.id empty AND a local row already exists → update existing `TrailEntity` in place, no network) is a **local-only** write — model it on `trail_library_provider.dart`'s direct-ObjectBox-write style (`TrailLibraryNotifier.deleteTrail`, `:58-95`, for the `store.runInTransaction(TxMode.write, ...)` pattern), not on the network `trailSaveProvider` calls.

**Photo copy failure — best-effort precedent to copy for D-03** (same `catch`/toast shape as the block above, but non-blocking — trail save still succeeds):
```dart
// pattern: toast on partial failure, do not throw
ref.read(toastProvider.notifier).add(ToastMessage(
  type: ToastType.error,
  icon: FontAwesomeIcons.triangleExclamation,
  text: l10n.photo_copy_failed_toast(count: failedCount),
));
```

---

### `app/lib/util/local_photo_store_util.dart` (NEW — file-I/O utility)

**Analog:** `app/lib/services/trail_download_service.dart` (app-owned directory + path-join discipline, `:53-60`)

```dart
final appDir = await getApplicationDocumentsDirectory();
final trailDir = Directory('${appDir.path}/library/$trailId');
if (!await trailDir.exists()) {
  await trailDir.create(recursive: true);
}
```
The new photo-copy util should follow the same `getApplicationDocumentsDirectory()` + subdirectory convention (D-01's discretion item: name it distinctly from `library/`, e.g. `unsynced/` or similar, so D-02's cleanup/orphan-sweep can target it specifically without touching downloaded-trail files). **Must use `package:path`'s `p.join`**, not string concatenation — `trail_download_service.dart:7` already imports `path` as `p` but this specific snippet still string-interpolates; RESEARCH.md's Security Domain section explicitly calls out `p.join` as the required pattern for the new photo-copy paths (matching `map_cache_path.dart`'s discipline), so do not copy the raw `'${appDir.path}/library/$trailId'` interpolation verbatim — use `p.join(appDir.path, 'unsynced', trailId)` instead.

**Photo download error handling to mirror** (`trail_download_service.dart:104-135`):
```dart
// Default Future.wait (eagerError: false) lets in-flight work settle before
// cleanup runs on a failure, so we never delete the directory while a photo
// is still being written.
try {
  await Future.wait(futures);
} catch (e) {
  if (await trailDir.exists()) {
    await trailDir.delete(recursive: true);
  }
  rethrow;
}
```
D-03 wants the opposite polarity (drop the one failed photo, keep the trail) — copy the `Future.wait`/cleanup *shape*, not this exact abort-everything behavior.

---

### `app/lib/provider/profile/profile_trails_provider.dart` (rebuild local-first)

**Analogs:** `trail_library_provider.dart` (local-read half) + itself (network half, unchanged)

**Local-first read pattern to copy** (`trail_library_provider.dart:14-50`):
```dart
@riverpod
class TrailLibraryNotifier extends _$TrailLibraryNotifier {
  @override
  List<Trail> build() {
    final store = ref.watch(objectBoxProvider);
    final userId = currentAccountId(store);
    if (userId == null) return const [];   // never an unfiltered read

    final box = store.box<TrailEntity>();
    final query = box.query(TrailEntity_.savedByUserIds.containsElement(userId)).build();
    final trails = <Trail>[];
    for (final entity in query.find()) {
      try {
        trails.add(entity.toModel());
      } catch (e, st) {
        debugPrint('...');   // per-entity guard, not bulk .map()
      }
    }
    query.close();
    trails.sort((a, b) => b.created.compareTo(a.created));
    return trails;
  }
}
```
The new own-trails provider needs the identical per-entity try/catch guard (one corrupt cached row must not blank the whole list) and the identical `currentAccountId(store) == null → empty, never unfiltered` guard — but filtered on the new **owner** field, not `savedByUserIds`.

**Network half — unchanged, kept as-is** (`profile_trails_provider.dart:80-124`, existing `_fetchPage`) — reuse verbatim when online; skip entirely when offline (REC-06).

**Merge point — the `TrailSummary` interface both sides already implement** (`app/lib/models/global_search_models.dart:78-83`):
```dart
@override
bool get isLocal => false;

@override
List<String> get localPhotos => [];
```
Both `Trail` and `TrailSearchResult` implement `TrailSummary` (consumed unchanged by `TrailListItem`/`TrailCard`); the merged provider should produce `List<TrailSummary>` by concatenating owner-scoped `Trail`s (ObjectBox) with `TrailSearchResult`s (network) — no widget changes needed for the merge itself.

**Empty-state precedent** — `library_screen.dart`'s `_LibraryEmptyState.icon` (private class today; UI-SPEC.md requires promoting to shared or duplicating exactly, not inventing a third visual language). Read `app/lib/routes/library_screen.dart` lines ~200-230 before implementing.

---

### `app/lib/components/trail/trail_dropdown.dart` (split `isLocal` branch)

**Analog:** itself

**Download-gating pattern to extend** (`trail_dropdown.dart:43-46`, `104-134`):
```dart
final isDownloading = ref.watch(downloadingTrailIdsProvider).contains(trail.id);
final downloadEnabled = !widget.availableOffline && !isDownloading;
...
PopupMenuItem<TrailAction>(
  value: TrailAction.download,
  onTap: downloadEnabled ? () => ref.read(downloadingTrailIdsProvider.notifier).download(trail) : null,
  enabled: downloadEnabled,
  ...
),
```
D-17 requires the item **hidden entirely** for an unsynced trail, not merely disabled — wrap this whole `PopupMenuItem` in an `if (!isUnsynced) ...[...]` the same way `_allowDelete`/`_canEditTrail` already gate other items (`trail_dropdown.dart:88`, `135`: `if (_canEditTrail(ref)) ...[`, `if (_allowDelete(ref)) ...[`).

**Delete-gating pattern to extend** (`trail_dropdown.dart:169-177`, `203-223`):
```dart
bool _allowDelete(WidgetRef ref) {
  if (widget.trail.isLocal) {
    return true;
  }
  final user = ref.watch(authProvider).value;
  if (user == null) return false;
  return widget.trail.author == user.actorId;
}
```
```dart
Future<void> _deleteTrail(BuildContext context, Trail trail) async {
  // For a local trail this means "remove the download", and NOTHING else...
  if (trail.isLocal) {
    Navigator.of(context).pop();
    ref.read(trailLibraryProvider.notifier).deleteTrail(trail.id);
    return;
  }
  // ... server delete path
}
```
This `if (trail.isLocal)` branch is the landmine RESEARCH.md/CONTEXT.md both flag: it must become a three-way branch (downloaded vs unsynced vs server-owned), calling the new unsynced-delete path (D-14: confirm-as-unrecoverable, blocked mid-drain via the new drain provider's in-flight set — same `isDownloading`-style gate as above) instead of `trailLibraryProvider.deleteTrail` for an unsynced trail.

**Confirm-dialog shape to copy for both new dialogs (unsynced-delete confirm, sign-out warning)** (`trail_dropdown.dart:179-201`):
```dart
void _confirmDelete(BuildContext context, Trail trail) {
  final l18n = AppLocalizations.of(context)!;
  showDialog(
    context: context,
    builder: (ctx) => AlertDialog(
      content: Text(l18n.delete_trail_confirm),
      actions: [
        TextButton(onPressed: () => Navigator.of(ctx).pop(), child: Text(l18n.cancel)),
        TextButton(
          onPressed: () { Navigator.of(ctx).pop(); _deleteTrail(context, trail); },
          child: Text(l18n.delete, style: const TextStyle(color: Colors.red)),
        ),
      ],
    ),
  );
}
```

---

### `app/lib/components/trail/trail_card.dart` / `trail_list_item.dart` (new `_SyncStatusChip`)

**Analog:** itself — existing `_Chip`/badge widgets in these two files (referenced but not fully quoted in UI-SPEC.md; read the files directly before implementing).

UI-SPEC.md's exact contract:
```
Chip shape: same visual language as the existing `_Chip` tag chips
(horizontal: 8, vertical: 4 padding, BorderRadius.circular(8),
background colorScheme.surfaceContainerHighest) — icon + label in a Row,
mainAxisSize: MainAxisSize.min, 4px gap between icon and text.
```
Spinner reuse (`trail_dropdown.dart:114-118`):
```dart
const SizedBox(
  width: 18,
  height: 18,
  child: CircularProgressIndicator(strokeWidth: 2),
),
```
UI-SPEC.md specifies 12-14px for the sync chip's spinner — same `CircularProgressIndicator(strokeWidth: 2)` shape, smaller `SizedBox`.

---

### `app/lib/main.dart` — app-wide `WidgetsBindingObserver`

**Analog:** `app/lib/routes/navigation_screen.dart` (`didChangeAppLifecycleState`, `:593-602` per RESEARCH.md) — read this file directly before implementing; it is the only existing lifecycle-observer precedent in the app, currently scoped to one screen for GPS stream lifecycle. `MainApp` (`main.dart:72`) is a plain `ConsumerStatefulWidget` today and needs the `WidgetsBindingObserver` mixin added, `didChangeAppLifecycleState` overridden to call the drain on `AppLifecycleState.resumed`, plus a `ref.listen` on `onlineStatusProvider` for the connectivity-regained half (D-15). Also mirror the existing startup refresh call (`main.dart:90`):
```dart
unawaited(ref.read(onlineStatusProvider.notifier).refresh());
```
— the drain's own foreground/connectivity handler must call this same `.refresh()` before trusting `onlineStatusProvider` (Pitfall 5: the provider is optimistic, `build() => true`).

---

## Shared Patterns

### Account scoping (V4 Access Control — applies to every new read/write path)
**Source:** `app/lib/util/current_account.dart` + `app/lib/provider/trail/trail_library_provider.dart:18-29`, `trail_provider.dart:70-76`, `app/lib/util/navigation_launch_util.dart:36-42`
```dart
final userId = currentAccountId(store);
if (userId == null) return const [];  // never an unfiltered read
final query = box.query(TrailEntity_.<field>.containsElement(userId) /* or .equals(userId) for a singular owner */).build();
```
**Apply to:** the new own-trails provider, the drain provider's query, and any delete path touching unsynced rows. `currentAccountId` must be re-read fresh at each use (never cached), per its own doc comment — including inside the drain, not just at provider construction (RESEARCH.md Open Question 2).

### In-flight-operation tracking (keepAlive Set<String> + finally cleanup)
**Source:** `app/lib/provider/trail/trail_download_state_provider.dart:27-36`, `272-280`
```dart
@Riverpod(keepAlive: true)
class SomeInFlightIds extends _$SomeInFlightIds {
  @override
  Set<String> build() => {};
  Future<void> op(String key) async {
    if (state.contains(key)) return;
    state = {...state, key};
    try { /* work */ } finally { state = {...state}..remove(key); }
  }
}
```
**Apply to:** the new drain provider (`TrailSyncNotifier`); its in-flight set doubles as the "delete blocked mid-drain" signal in `trail_dropdown.dart`.

### Account-switch invalidation exclusion (deliberate, documented)
**Source:** `app/lib/util/account_scope_invalidation.dart:14-16`, `58-71`
**Apply to:** the new drain provider must be added to the exclusion doc comment (not the `accountScopedProviders` list) with the same "mid-operation desync" reasoning already used for `downloadingTrailIdsProvider`.

### Error handling / toast on best-effort failure
**Source:** `trail_create_screen.dart:401-469` (catch → `toastProvider` → `ToastType.error`), `trail_download_state_provider.dart:262-271`
```dart
ref.read(toastProvider.notifier).add(ToastMessage(
  type: ToastType.error,
  icon: FontAwesomeIcons.<icon>,
  text: l10n.<key>,
));
```
**Apply to:** D-03's photo-copy-failure toast, D-07's parked-Failed surfacing (if a toast is used in addition to the inline chip).

### App-owned storage directory convention
**Source:** `trail_download_service.dart:53-60` (`getApplicationDocumentsDirectory()` + subdirectory), contrast note re: `p.join` discipline
**Apply to:** the new local-photo-copy util (D-01) and its D-02 cleanup/orphan sweep.

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| Backoff/retry curve logic for D-07 | utility | event-driven | No retry/backoff precedent exists anywhere in this codebase (RESEARCH.md confirms no retryable-vs-permanent taxonomy exists); planner should design a simple fixed-count exponential or linear backoff, informed only by the in-flight-tracking pattern above for where it plugs in |
| Startup orphan sweep for leaked photo copies (D-02) | utility | file-I/O / batch | No existing "startup sweep" task exists in this codebase; closest conceptual sibling is the account-scoping purge in `account_data_purge_util.dart`, but that fires on logout, not app startup — treat as new, following only the general "best-effort, swallow individual failures" discipline used elsewhere |

## Metadata

**Analog search scope:** `app/lib/entities/`, `app/lib/provider/trail/`, `app/lib/provider/profile/`, `app/lib/routes/trail_create_screen.dart`, `app/lib/components/trail/`, `app/lib/services/trail_download_service.dart`, `app/lib/util/current_account.dart`, `app/lib/util/account_scope_invalidation.dart`, `app/lib/main.dart`
**Files scanned:** ~15 read in full or targeted sections, plus grep sweeps over `trail_dropdown.dart`, `trail_create_screen.dart`
**Pattern extraction date:** 2026-08-02
