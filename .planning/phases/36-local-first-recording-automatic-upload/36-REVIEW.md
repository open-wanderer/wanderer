---
phase: 36-local-first-recording-automatic-upload
reviewed: 2026-08-03T15:10:00Z
depth: standard
files_reviewed: 57
files_reviewed_list:
  - app/lib/components/trail/sync_status_chip.dart
  - app/lib/components/trail/trail_card.dart
  - app/lib/components/trail/trail_dropdown.dart
  - app/lib/components/trail/trail_list_item.dart
  - app/lib/components/trail/trail_panel.dart
  - app/lib/entities/trail_entity.dart
  - app/lib/entities/waypoint_entity.dart
  - app/lib/i18n/app_en.arb
  - app/lib/main.dart
  - app/lib/models/global_search_models.dart
  - app/lib/models/trail.dart
  - app/lib/models/trail_summary.dart
  - app/lib/models/trail_sync_state.dart
  - app/lib/models/waypoint.dart
  - app/lib/provider/profile/profile_trails_provider.dart
  - app/lib/provider/router_provider.dart
  - app/lib/provider/trail/local_trail_provider.dart
  - app/lib/provider/trail/trail_filter_provider.dart
  - app/lib/provider/trail/trail_save_provider.dart
  - app/lib/provider/trail/trail_sync_provider.dart
  - app/lib/routes/profile_trail_screen.dart
  - app/lib/routes/settings_account_screen.dart
  - app/lib/routes/settings_screen.dart
  - app/lib/routes/trail_create_screen.dart
  - app/lib/routes/trail_detail_map_screen.dart
  - app/lib/routes/trail_detail_screen.dart
  - app/lib/services/trail_download_service.dart
  - app/lib/util/account_scope_invalidation.dart
  - app/lib/util/local_id.dart
  - app/lib/util/local_photo_store_util.dart
  - app/lib/util/local_trail_store.dart
  - app/lib/util/offline_trail_filter_bounds.dart
  - app/lib/util/own_trails_merge.dart
  - app/lib/util/sync_backoff.dart
  - app/lib/util/trail_route_location.dart
  - app/lib/util/unsynced_signout_guard.dart
  - app/pubspec.yaml
  - app/test/components/trail/sync_status_chip_test.dart
  - app/test/components/trail/trail_dropdown_delete_gate_test.dart
  - app/test/components/trail/trail_dropdown_menu_test.dart
  - app/test/entities/trail_entity_test.dart
  - app/test/provider/trail/local_trail_addressing_gate_test.dart
  - app/test/provider/trail/trail_filter_fallback_test.dart
  - app/test/routes/profile_trail_screen_navigation_test.dart
  - app/test/routes/settings_screen_signout_gate_test.dart
  - app/test/routes/trail_create_screen_local_save_gate_test.dart
  - app/test/services/trail_download_service_carry_forward_test.dart
  - app/test/util/account_scope_invalidation_test.dart
  - app/test/util/local_id_test.dart
  - app/test/util/local_photo_store_util_test.dart
  - app/test/util/local_trail_retirement_gate_test.dart
  - app/test/util/local_trail_store_test.dart
  - app/test/util/offline_trail_filter_bounds_test.dart
  - app/test/util/own_trails_merge_test.dart
  - app/test/util/sync_backoff_test.dart
  - app/test/util/trail_route_location_test.dart
  - app/test/util/unsynced_signout_guard_test.dart
findings:
  critical: 4
  warning: 12
  info: 0
  total: 16
status: issues_found
---

# Phase 36: Code Review Report

**Reviewed:** 2026-08-03T15:10:00Z
**Depth:** standard
**Files Reviewed:** 57
**Status:** issues_found

## Summary

This is a full-phase pass over all 14 plans, superseding the 2026-08-02 review that
covered plans 36-01..36-08. The fixes recorded in `36-REVIEW-FIX.md` were verified
present: `TrailEntity` now persists `categoryRecordId`/`subcategoryRecordId`/
`completed`/`tagsJson`; `WaypointPhotoUploadException` carries the created id;
`writeServerTrailId` commits id+photos off the raw body before `Trail.fromJson`;
`markTrailUploading` re-queries inside a transaction; `reconcileLocalPhotos`
canonicalizes both sides; `local_id.dart` grew `recordIdDirSegment`/`fileNameSegment`
and `trail_download_service.dart` uses them. `flutter analyze` reports 36 issues, all
pre-existing `info`-level. The pure-policy modules (`sync_backoff.dart`,
`own_trails_merge.dart`, `trail_route_location.dart`, `offline_trail_filter_bounds.dart`,
`local_id.dart`) are genuinely sound and genuinely tested.

**Prior finding still present:** CR-04 from the previous review ("an edit after upload
never reaches the server") was declared fixed. It is not. The fix changed the routing
decision so a post-upload edit is sent down `_saveViaNetwork` — but the value it sends
carries an **empty server id**, because the screen's snapshot came out of
`TrailEntity.toModel()`, which blanks a local-sentinel id (D-06). Silent loss became a
guaranteed HTTP failure that no retry can clear. See **CR-01**.

The defects cluster in three places, all downstream of the two decisions the last two
plans introduced — **retire-the-row-on-success** (36-13) and **route-the-save-on-the-
persisted-row** (36-REVIEW-FIX CR-04):

- The `persistedLocalId != null && persisted == null` ⇒ `networkUpdate` rule
  (`resolveLocalSaveModeForRow`) conflates "the drain retired this row" with "the row
  was never written", and in both cases hands the network path a trail with no server
  id and photo files that have already been deleted from disk (**CR-01**, **CR-02**).
- The drain's step 2 is guarded on `isLocalId(entity.id)`, so once a server id is
  stamped no subsequent local edit can ever be uploaded — and step 4 then retires the
  row, destroying the edit. `updateLocalTrail` only refuses `synced` rows; it happily
  accepts the `pending`/`uploading`/`failed` rows where this window lives (**CR-03**).
- `deleteUnsynced` is still built on the assumption "unsynced ⇒ never on the server",
  which stopped being true the moment `writeServerTrailId` was pulled forward. A
  `failed` row can hold a real server id, so "delete this trail, it can't be undone"
  deletes the local copy and leaves the (possibly public) server copy behind
  (**CR-04**).

Account scoping holds up well: every read in `local_trail_store.dart` is owner-scoped,
`currentAccountId` is re-read fresh at each drain pass and at each
`_readOwnLocal`/`_offlineFilterValues` call, and `readLocalTrailMetrics`' scoping
reasoning is correct. The one asymmetry is the *delete* path, which has no owner clause
at all (**WR-10**).

The test-strategy note in `36-REVIEW-FIX.md` is accurate — `libobjectbox.dylib` genuinely
does not load under `flutter test` — and `trail_dropdown_menu_test.dart` /
`profile_trail_screen_navigation_test.dart` are real behavioural tests, not greps. But
the source-grep gates have started to drift from what they claim: none of the four
blockers below would be caught by any test in this phase, and
`trail_create_screen_local_save_gate_test.dart` now asserts an invariant the code no
longer satisfies while still passing (**WR-05**).

## Critical Issues

### CR-01: A post-upload edit is POSTed to `/trail/form/` with an empty id and can never succeed

**File:** `app/lib/routes/trail_create_screen.dart:436-463`, `app/lib/routes/trail_create_screen.dart:547-567`, `app/lib/util/local_trail_store.dart:113-122`, `app/lib/provider/trail/trail_save_provider.dart:132`, `app/lib/entities/trail_entity.dart:320`

**Issue:** Trace the ordinary flow the phase exists to support — record, save, wait for
the upload, fix a typo:

1. `_finishLocalSave` sets `trail = readLocalTrail(store, localId)`
   (`trail_create_screen.dart:792-799`). At that instant the row's `id` is still the
   local sentinel, so `TrailEntity.toModel()` blanks it: **`trail.id == ''`**
   (`trail_entity.dart:320`).
2. The drain completes and `retireUploadedLocalTrail` **deletes the row**
   (`trail_sync_provider.dart:306`), then `_deletePhotoDirBestEffort` **deletes
   `<app-docs>/unsynced/<localId>/`** (`:317`).
3. The user edits and saves. `persisted = readLocalTrail(...)` is now `null` while
   `persistedLocalId` is non-null, so `resolveLocalSaveModeForRow` returns
   `LocalSaveMode.networkUpdate` (`local_trail_store.dart:118-120`) and `_onSave`
   calls `_saveViaNetwork(l10n, updatedTrail, ...)` (`trail_create_screen.dart:455-463`).
4. `updateTrail` issues `api.post('/trail/form/${newTrail.id}')`
   (`trail_save_provider.dart:132`) → **`POST /api/v1/trail/form/`**. That path has no
   handler: `web/src/routes/api/v1/trail/form/+server.ts` exports only `PUT`, and
   SvelteKit's default `trailingSlash: 'never'` normalizes the empty `[id]` away. The
   request 405s / 404s.
5. Even before the request, `newPhotoFiles` are `File(p)` over
   `values['photos']`, seeded from `trail.localPhotos` — i.e. paths under the photo
   directory step 2 just deleted. `toFormData` reads those files and throws.

The user sees `error_saving_trail`, and every retry reproduces it: the screen's `trail`
snapshot can never acquire a server id, because the only writer of that field
(`_finishLocalSave`'s `readLocalTrail`) now returns `null` forever. The edit is
unreachable except by backing out of the screen and re-opening the trail from the
server list.

The same code path is reached from the `LocalUpdateOutcome.missing` branch
(`trail_create_screen.dart:547-566`), whose comment explicitly anticipates this state
("`missing` far more often than `alreadySynced`") without noticing the id is blank.

**Fix:** the network path needs a real server id and real photo sources. Carry the
server id back to the screen when the drain retires the row, and fall back to
re-reading the trail from the server rather than POSTing a blank id:

```dart
// local_trail_store.dart -- retire must be able to tell the caller the id it kept
String? retireUploadedLocalTrail(Store store, String localId) { /* return entity.id */ }

// trail_create_screen.dart -- _onSave
if (saveMode == LocalSaveMode.networkUpdate) {
  final serverId = updatedTrail.id.isNotEmpty
      ? updatedTrail.id
      : _serverIdForRetiredLocalId(persistedLocalId); // recorded at retire time
  if (serverId == null || serverId.isEmpty) {
    // Do NOT fabricate a POST with a blank id.
    _toastError(l10n.error_saving_trail);
    return;
  }
  await _saveViaNetwork(
    l10n,
    updatedTrail.copyWith(id: serverId, localId: null,
        // the unsynced/ copies are gone; re-upload nothing, keep server photos
        localPhotos: const []),
    authorId: authorId,
    newPhotoFiles: newPhotoFiles.where((f) => f.existsSync()).toList(),
  );
  return;
}
```

Minimum acceptable alternative: refuse the save and tell the user to re-open the trail,
rather than issuing a request that cannot possibly be routed.

Add a behavioural test in `trail_create_screen_local_save_gate_test.dart`'s place that
overrides `apiProvider` and asserts the request URI is never `/trail/form/`.

---

### CR-02: A first local save that fails permanently bricks the create screen

**File:** `app/lib/routes/trail_create_screen.dart:478-495`, `app/lib/routes/trail_create_screen.dart:503-518`, `app/lib/util/local_trail_store.dart:113-122`

**Issue:** In the `createLocal` branch, `_localId` is assigned **before** the row is
written:

```dart
final localId = mintLocalId();
_localId = localId;                     // :478-479

final photoCopy = await _copyPhotosForLocalSave(...);   // can throw
saveNewLocalTrail(store, ...);                          // can throw
```

`_copyPhotosForLocalSave` awaits `getApplicationDocumentsDirectory()` and
`Directory.create(recursive: true)`, and `saveNewLocalTrail` runs an ObjectBox write
transaction — both can throw (the `catch` at `:503` exists precisely because they can).
When either does, `_localId` is left pointing at an id with **no row**.

The next save then evaluates `persistedLocalId != null && persisted == null`, which
`resolveLocalSaveModeForRow` maps to `LocalSaveMode.networkUpdate`
(`local_trail_store.dart:118-120`) — and lands in CR-01's dead POST. The screen can
never save again, locally or otherwise, for the rest of its life. The user's recorded
track is still only in memory at that point.

`resolveLocalSaveModeForRow`'s doc comment asserts "A null `persistedLocalId` is the
genuinely-never-saved case", which is exactly the invariant this ordering breaks: after
a failed first save, `persistedLocalId` is non-null and the trail was never saved.

**Fix:** publish `_localId` only after the row exists, and make the decision function
distinguish "never written" from "retired" on evidence rather than on a field the
screen sets speculatively.

```dart
final localId = mintLocalId();
final photoCopy = await _copyPhotosForLocalSave(localId, updatedTrail, localPhotoPaths);
saveNewLocalTrail(store, /* ... */ localId: localId, /* ... */);
_localId = localId;                       // only now is the row real
await _finishLocalSave(store, l10n, photoCopy.failedCount, localId: localId);
```

A dedicated unit test on `resolveLocalSaveModeForRow` covering "id set, no row, never
uploaded" belongs next to the existing group in `test/util/local_trail_store_test.dart`.

---

### CR-03: Any edit made after the drain stamps a server id is silently discarded

**File:** `app/lib/provider/trail/trail_sync_provider.dart:178`, `app/lib/provider/trail/trail_sync_provider.dart:306`, `app/lib/util/local_trail_store.dart:289-345`, `app/lib/routes/trail_create_screen.dart:539-574`

**Issue:** The drain's step 2 only sends the trail when `isLocalId(entity.id)`:

```dart
if (isLocalId(entity.id)) {   // :178
  // PUT /trail/form ... writeServerTrailId(...)
}
```

Once `writeServerTrailId` has run, the row's `id` is a server id **for the rest of its
life**, including across every retry and across a parked `failed` state. The drain has
no update path at all — `updateLocalTrail`'s own doc comment says so
(`local_trail_store.dart:281-285`) — and step 4 then retires the row
(`trail_sync_provider.dart:306`), destroying whatever the row held.

Meanwhile `updateLocalTrail` refuses only `TrailSyncState.synced`
(`local_trail_store.dart:302-304`). Every other state is accepted and written. So:

1. Drain step 2 succeeds; step 3's first waypoint upload fails four times;
   `resolveDrainFailureOutcome` parks the row as `failed`
   (`local_trail_store.dart:168-174`).
2. The user opens the trail, fixes the name/description/privacy, saves.
   `resolveLocalSaveMode(persisted)` sees `syncState == failed` → `updateLocal`;
   `updateLocalTrail` writes the edit and returns `updated`; `_finishLocalSave` shows
   `trail_saved_successfully`.
3. The user taps the "Upload failed · Tap to retry" chip
   (`sync_status_chip.dart:58-61`). The drain skips step 2 (id is not local), retries
   the waypoint, succeeds, and calls `retireUploadedLocalTrail`, **deleting the row and
   the edit with it**.

The server keeps the pre-edit values. The user was shown a success toast and a
successful upload. Nothing anywhere records that the edit was dropped. The same window
exists (narrower) between step 2 and step 4 of a single successful pass, and across the
whole 30s/2m/10m backoff ladder.

This is the un-fixed remainder of the previous review's CR-04: the fix addressed the
`synced` case and left the `pending`/`uploading`/`failed`-with-a-server-id case, which
is the larger of the two windows.

**Fix:** either make step 2 handle the update case, or refuse the local write once an
id exists. The first is preferable because it keeps the local-first promise:

```dart
// trail_sync_provider.dart, step 2
if (isLocalId(entity.id)) {
  // ... PUT /trail/form (create) ...
} else if (entity.dirtySinceUpload) {          // new persisted flag, set by updateLocalTrail
  await ref.read(apiProvider).post('/trail/form/${entity.id}', data: formData, ...);
  clearDirtySinceUpload(store, localId);
}
```

`updateLocalTrail` sets `dirtySinceUpload = true` whenever the row it is writing over
already has a non-local `id`. If that is more than this phase should take on, the
minimum is to make `updateLocalTrail` return a new `LocalUpdateOutcome.alreadyUploaded`
for a row whose `id` is not a local sentinel, so the caller is forced to route to the
network path instead of writing an edit that has nowhere to go.

---

### CR-04: "Delete — this can't be undone" leaves a live, possibly public trail on the server

**File:** `app/lib/components/trail/trail_dropdown.dart:243-245`, `app/lib/components/trail/trail_dropdown.dart:283-317`, `app/lib/provider/trail/trail_sync_provider.dart:378-398`, `app/lib/util/local_trail_store.dart:353-367`

**Issue:** `_confirmDelete` picks its copy purely off the sync state:

```dart
final confirmCopy = isUnsyncedState(trail.syncState)
    ? l18n.delete_unsynced_trail_confirm   // "It hasn't been uploaded yet, so this can't be undone."
    : l18n.delete_trail_confirm;
```

and `_deleteTrail` routes on the same predicate to `TrailSync.deleteUnsynced`, which
does `deleteLocalTrailRow` + photo-dir cleanup and **never issues a server DELETE**
(`trail_sync_provider.dart:381-389`).

D-14's premise — unsynced means the device holds the only copy — stopped being true
when SYNC-04 pulled `writeServerTrailId` forward to the instant the create is accepted
(`trail_sync_provider.dart:218-228`). After that write the row is `pending`/`uploading`/
`failed` **and** carries a real server id. Reachable path:

1. `PUT /trail/form` succeeds; `writeServerTrailId` commits the id. The trail is live on
   the server with whatever `public` value the user chose.
2. Step 3 fails four times → the row parks as `failed`.
3. The chip reads "Upload failed"; the user gives up and deletes, and is told the
   deletion cannot be undone because it was never uploaded.
4. The local row and photos are destroyed. The server trail — indexed by Meilisearch,
   federated if public, visible on the web profile — remains, with waypoints partially
   or wholly missing.

The user believes they destroyed a trail they in fact published. There is no affordance
anywhere in the app that reaches it afterwards, because the local row that held its id
is gone.

**Fix:** decide on the id, not on the sync state, and delete both sides when both exist.

```dart
// trail_sync_provider.dart
Future<bool> deleteUnsynced(String localId) async {
  if (state.contains(localId)) return false;
  final store = ref.read(objectBoxProvider);
  final entity = findLocalRow(store, localId);
  final serverId = entity != null && !isLocalId(entity.id) ? entity.id : null;

  if (serverId != null) {
    // The create already landed -- this is not a device-only copy.
    await ref.read(apiProvider).delete('/trail/$serverId');
  }
  deleteLocalTrailRow(store, localId);
  await _deletePhotoDirBestEffort(localId);
  // ...
}
```

and make `_confirmDelete` choose `delete_unsynced_trail_confirm` only when the row's id
is still a local sentinel — a row with a server id must not claim the deletion is
irreversible/device-only. A network failure on the server DELETE must return `false` so
the local row is not destroyed while the server copy survives.

## Warnings

### WR-01: `localTrailProvider` is never invalidated when the drain retires the row

**File:** `app/lib/provider/trail/trail_sync_provider.dart:306-320`, `app/lib/provider/trail/local_trail_provider.dart:26-35`, `app/lib/routes/trail_detail_screen.dart:71-88`

**Issue:** `_drainOne` invalidates `trailLibraryProvider` and
`profileTrailsProvider('@<username>')` after a successful upload, but not
`localTrailProvider(localId)` — and there are no ObjectBox `Query.watch()` streams
anywhere (`trail_create_screen.dart:733-740` says so explicitly). A user sitting on
`/trail/local/<localId>` while the upload completes keeps rendering a `Trail` for a row
that no longer exists, with a live Edit button that walks straight into CR-01. If they
navigate away and back, or deep-link, they get `trail_not_on_this_device` with no path
to the now-uploaded trail.

**Fix:** invalidate it alongside the other two, and give the not-found branch a way
forward:

```dart
retireUploadedLocalTrail(store, localId);
ref.invalidate(localTrailProvider(localId));
```

plus, in `trail_detail_screen.dart`'s null branch, redirect to `/trail/<serverId>` when
the retirement recorded one (see CR-01's suggested return value) instead of a dead end.

---

### WR-02: `_invalidateOwnTrailsList` interpolates a nullable username into the family key

**File:** `app/lib/routes/trail_create_screen.dart:753-760`

**Issue:**

```dart
ref.invalidate(
  profileTrailsProvider('@${ref.read(authProvider).value?.preferredUsername}'),
);
```

When `authProvider.value` is null (a token refresh in flight, a mid-logout race) the key
becomes the literal `'@null'`, which matches no mounted instance — the invalidation is a
silent no-op and the own-trails list keeps its pre-edit snapshot. The doc comment above
this method states "the family key must match the one `profile_trail_screen` watches or
the invalidation is a silent no-op", which is exactly the failure the interpolation
allows.

**Fix:**

```dart
final username = ref.read(authProvider).value?.preferredUsername;
if (username != null) ref.invalidate(profileTrailsProvider('@$username'));
```

`trail_sync_provider.dart:320` uses `userEntity.preferredUsername`, a non-nullable value
resolved from the store, and is correct — this is the only site with the hole.

---

### WR-03: `TrailFilterNotifier.defaultFilter` is unassigned on the non-connection error path

**File:** `app/lib/provider/trail/trail_filter_provider.dart:68`, `app/lib/provider/trail/trail_filter_provider.dart:103-108`, `app/lib/provider/trail/trail_filter_provider.dart:150-152`

**Issue:** `defaultFilter` is `late` and assigned in exactly two places — the success
path (`:84`) and the connection-failure fallback (`:104`). Every other failure (a 500, a
malformed payload, `TrailFilterValues.fromJson` throwing) rethrows at `:107` without
assigning it. `resetFilter()` (`:150-152`) then reads it unconditionally and throws
`LateInitializationError`, an uncaught error out of a button callback. The provider is
`keepAlive`, so the un-initialised instance survives.

**Fix:** initialise it at declaration so the field always has a meaning:

```dart
TrailFilter defaultFilter = buildDefaultTrailFilter(kOfflineTrailFilterValues);
```

and drop the `late`. The `late` was retained from the fix for the `late final` rebuild
crash; a plain initialised field solves both problems.

---

### WR-04: A `null` waypoint `localKey` burns retry attempts and can park a good trail as `failed`

**File:** `app/lib/provider/trail/trail_sync_provider.dart:245-251`, compare `app/lib/provider/trail/trail_sync_provider.dart:147-159`

**Issue:** The `StateError` thrown for a keyless waypoint sits **inside** `_drainOne`'s
`try`, so it lands in the generic handler at `:321-332` and consumes one of the four
`kMaxSyncAttempts`. This is an invariant break, not a network condition — no amount of
retrying will mint a `localKey` — yet four passes (a lifecycle/connectivity flurry
produces these within seconds, and `syncBackoffDelay` only starts at 30s after the
*first* failure) park the trail as `failed`, after which `isDrainDue` returns false
forever and only a manual chip tap revives it. This is precisely the reasoning applied
one screen up for the missing `UserEntity` (`:140-159`, the WR-12 fix), applied
inconsistently.

Worse, the trail may already have been created server-side by step 2 before this throws,
which puts it straight into CR-04's territory.

**Fix:** resolve the keys before joining the in-flight set, and bail without recording a
failure:

```dart
final keyless = entity.waypoints.where((w) => w.localKey == null && isLocalId(w.id));
if (keyless.isNotEmpty) {
  debugPrint('trail_sync_provider: "$localId" has keyless waypoints; skipping drain');
  return;   // before `state = {...state, localId}` and before the try
}
```

---

### WR-05: The local-save gate test asserts an invariant the code no longer satisfies, and still passes

**File:** `app/test/routes/trail_create_screen_local_save_gate_test.dart:197-242`, against `app/lib/routes/trail_create_screen.dart:455-463` and `app/lib/routes/trail_create_screen.dart:547-548`

**Issue:** The test is titled "the updateLocal branch reaches the network ONLY on the
alreadySynced escape hatch (CR-04)" and its `reason:` strings say the network reach
"must stay guarded on that specific outcome". Two things have since changed:

- The guard is now `outcome == LocalUpdateOutcome.alreadySynced || outcome ==
  LocalUpdateOutcome.missing` (`:547-548`). The test only checks that the string
  `LocalUpdateOutcome.alreadySynced` appears *somewhere before* the `_saveViaNetwork(`
  call, so the `missing` addition — the branch that actually fires in practice, and the
  one that carries CR-01's blank id — passes unnoticed.
- The slice starts at `if (saveMode == LocalSaveMode.createLocal) {`, which excludes the
  `networkUpdate` early return at `:455-463`. So "exactly one `_saveViaNetwork` call" is
  true of the slice and false of `_onSave`, while the `reason:` text claims the latter.

The scope note for this review asked whether each gate's stated justification still
holds. This one's does not: it is now a test that passes for a reason unrelated to the
property it names, over a method it does not fully cover.

**Fix:** re-anchor the slice on the whole `_onSave` body, assert on the exact guard
expression (`alreadySynced || outcome == LocalUpdateOutcome.missing`), and — better —
replace the ordering grep with the behavioural `apiProvider`-override test CR-01 needs
anyway, which would have caught both the blank id and the guard drift.

---

### WR-06: The destructive-action strings are still English-only in 12 locales

**File:** `app/lib/i18n/untranslated_messages.json`, `app/lib/i18n/app_de.arb`, `app/lib/i18n/app_fr.arb` (and 10 more)

**Issue:** Re-raised from the previous review's WR-15, which was closed as "partially
fixed — translations deliberately not authored". The engineering half (the committed
`untranslated-messages-file` report) is done and is a genuine improvement. The user-
facing gap is unchanged and has grown: the report now lists 12 locales missing
`delete_unsynced_trail_confirm`, `signout_unsynced_warning`,
`delete_blocked_while_uploading`, `trail_not_on_this_device`, `sync_pending`,
`sync_uploading`, `sync_failed`, `photo_copy_failed_toast`,
`own_trails_offline_banner`, `own_trails_empty_title`, `own_trails_empty_body` and
`retry_upload`.

Two of these are the copy on irreversible actions. Given CR-04, the English text of
`delete_unsynced_trail_confirm` is currently *wrong* as well as untranslated, so this
needs a pass regardless of who authors the translations.

**Fix:** ship `signout_unsynced_warning` and the (corrected) delete-confirm copy to every
locale before release, and treat `lib/i18n/untranslated_messages.json` as the work list.

---

### WR-07: `retry_upload` is a dead l10n key

**File:** `app/lib/i18n/app_en.arb`

**Issue:** `retry_upload` appears in the template ARB and in every locale's untranslated
report, but has no reference anywhere in `app/lib` or `app/test` — the retry affordance
uses `sync_failed` ("Upload failed · Tap to retry") as both label and action
(`sync_status_chip.dart:56`). It inflates the translation backlog by one string in 12
locales for nothing.

**Fix:** delete the key and regenerate, or wire it up if the chip was meant to carry a
separate action label.

---

### WR-08: An unsynced trail with a null `localId` falls through to a silent un-download no-op

**File:** `app/lib/components/trail/trail_dropdown.dart:283`, `app/lib/components/trail/trail_dropdown.dart:334-338`

**Issue:** The unsynced branch is guarded on `isUnsyncedState(trail.syncState) &&
trail.localId != null`. A row that is unsynced but has a null `localId` — which
`retireUploadedLocalTrail`'s demote branch creates deliberately at
`local_trail_store.dart:415`, and which `TrailDownloadService`'s carry-forward can
produce from `existing?.localId` — skips it and lands in the `trail.isLocal` branch,
calling `trailLibraryProvider.deleteTrail(trail.id)`. The route is popped first
(`:335`), so if that id is empty or not a library member the user is navigated away and
nothing is deleted, with no feedback — the same "looks exactly like a completed delete"
failure the unsynced branch was reworked to avoid.

**Fix:** make the fall-through explicit rather than implicit:

```dart
if (isUnsyncedState(trail.syncState)) {
  final localId = trail.localId;
  if (localId == null) {
    _toastError(l18n.error_deleting_trail);   // or route to the server delete
    return;
  }
  // ... existing deleteUnsynced flow
}
```

---

### WR-09: `writeServerWaypointId` clears `localPhotos` before the trail's upload has completed

**File:** `app/lib/util/local_trail_store.dart:693-695`, `app/lib/provider/trail/trail_sync_provider.dart:281-287`

**Issue:** The bookkeeping write sets `target.photos = serverPhotoFilenames` and
`target.localPhotos = []` the moment one waypoint is created. If a **later** waypoint in
the same loop fails and the trail is eventually parked as `failed`, the row survives with
waypoints whose model-level photos are now server filenames and whose local copies are
unreachable — while the actual JPEGs still sit on disk under
`unsynced/<localId>/waypoints/<key>/` (nothing deletes them on the failure path,
`trail_sync_provider.dart:322-324`). Viewing that trail offline shows a waypoint with
zero photos, and the on-disk files are dead weight until the trail is deleted.

**Fix:** keep the local copies until the whole trail is retired:

```dart
target.id = serverWaypointId;
target.photos = serverPhotoFilenames;
// localPhotos deliberately retained -- the trail's upload is not complete yet, and
// retireUploadedLocalTrail / deleteUnsyncedPhotoDir own the cleanup.
```

---

### WR-10: The local delete path has no owner clause while every read path does

**File:** `app/lib/util/local_trail_store.dart:353-367`, `app/lib/util/local_trail_store.dart:399-430`, `app/lib/provider/trail/trail_sync_provider.dart:378-398`

**Issue:** `readOwnLocalTrail` is owner-scoped, and its doc comment explains why: its
argument arrives from a route parameter that "survives a logout in the deep-link and
back-stack". `deleteLocalTrailRow` takes the same kind of value from the same kind of
path — `TrailSync.deleteUnsynced(trail.localId!)`, driven by a `Trail` that may have been
constructed before an account switch — and has **no** `owner` clause at all.
`retireUploadedLocalTrail` documents its own lack of scoping as safe because its only
caller came through `selectDrainCandidates`; `deleteLocalTrailRow` has no such
justification and no such caller guarantee.

Today the exposure is small (a stale widget holding a previous account's `Trail`), but
the asymmetry means the read side is hardened against exactly the vector the write side
is open to.

**Fix:** thread the account id through and add the clause, matching `readOwnLocalTrail`:

```dart
void deleteLocalTrailRow(Store store, String localId, {required String accountId}) {
  final query = trailBox.query(
    TrailEntity_.localId.equals(localId) & TrailEntity_.owner.equals(accountId),
  ).build();
  // ...
}
```

---

### WR-11: `TrailPanel` hides the summit-log and comment tabs for every trail read off the device

**File:** `app/lib/components/trail/trail_panel.dart:242-254`, `app/lib/models/trail.dart:106-125`, `app/lib/entities/trail_entity.dart:355`

**Issue:** The `TabBar` is gated on `!trail.isLocal`, and `TrailEntity.toModel()`
hardcodes `isLocal: true` for **every** row — downloaded trails included. `_TabContent`
still renders `children[_index]` with `_index` pinned at 0, so summit logs and comments
become unreachable on any trail served from ObjectBox. That includes the case
`trail_dropdown.dart:325-329` calls out by name: `TrailNotifier.build()` falls back to
the cache on *any* fetch exception, so one timeout on your own downloaded trail hides
its comments until the provider refetches.

`trail_dropdown.dart:47-52` documents that `isLocal` is no longer a usable
"device-only" signal after this phase and switches to `isUnsynced`; `trail_panel.dart`
was edited in the same plan and was not brought onto the same footing.

**Fix:** gate on the state that actually means "never reached the server":

```dart
if (!isUnsyncedState(trail.syncState))
  TabBar(/* ... */),
```

and derive the `_TabContent` child count from the same predicate so a local trail cannot
index past the About tab.

---

### WR-12: `LocalTrailMetrics`' "three parallel lists" contract is not what the query returns

**File:** `app/lib/util/offline_trail_filter_bounds.dart:18-26`, `app/lib/util/offline_trail_filter_bounds.dart:189-198`

**Issue:** The typedef is documented as "Three parallel lists of raw on-device column
values". `PropertyQuery<double>.find()` **excludes null values** unless
`replaceNullWith` is supplied (objectbox 5.3.1,
`lib/src/native/query/property.dart:220-224`), and `distance`, `elevationGain` and
`elevationLoss` are all nullable on `TrailEntity`. The three lists therefore have
different lengths and no positional correspondence to each other or to the row set.

`computeOfflineTrailFilterValues` only takes per-axis maxima, so today this is harmless —
but the comment invites the next reader to zip them (e.g. "trails with gain > X"), which
would silently pair one trail's distance with another's elevation.

**Fix:** correct the doc to "three independent, null-dropped value lists, one per axis —
NOT row-aligned", or pass `replaceNullWith: 0` on all three if alignment is ever wanted.

---

_Reviewed: 2026-08-03T15:10:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
