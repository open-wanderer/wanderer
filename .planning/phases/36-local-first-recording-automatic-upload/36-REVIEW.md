---
phase: 36-local-first-recording-automatic-upload
reviewed: 2026-08-03T17:40:00Z
depth: standard
files_reviewed: 55
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
  - app/test/components/trail/sync_status_chip_test.dart
  - app/test/components/trail/trail_dropdown_delete_gate_test.dart
  - app/test/components/trail/trail_dropdown_menu_test.dart
  - app/test/entities/trail_entity_test.dart
  - app/test/provider/trail/local_trail_addressing_gate_test.dart
  - app/test/provider/trail/trail_filter_fallback_test.dart
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
  critical: 3
  warning: 17
  info: 0
  total: 20
status: issues_found
---

# Phase 36: Code Review Report

**Reviewed:** 2026-08-03T17:40:00Z
**Depth:** standard
**Files Reviewed:** 55
**Status:** issues_found

## Summary

Re-review of the four blocker fixes (`6b32de55`, `9af61ffd`, `04075d12`, `b9ee66ca`,
plus the codegen fixpoint `588d46b6`) against current source. `app/lib/routes/profile_trail_screen.dart`
and its test are excluded per the scope note; generated artifacts are out of scope.

**Verdict on the four fixes:**

| Fix | Landed as described? | Sound? |
|---|---|---|
| CR-01 `trailHasServerId` blank-id refusal | Yes (`local_trail_store.dart:138`, `trail_create_screen.dart:625-651`) | **No** — the refusal fires on the phase's primary flow, so "record → save → fix a typo" is now a guaranteed, unexplained failure with no recovery (**CR-01** below) |
| CR-02 `_localId` published after the write | Yes (`trail_create_screen.dart:504`) | **Yes.** The ordering is correct, the invariant `resolveLocalSaveModeForRow` documents now actually holds, and the gate pins it. Nothing new introduced. |
| CR-03 `LocalUpdateOutcome.alreadyUploaded` | Yes (`local_trail_store.dart:286,343-345`, `trail_create_screen.dart:558`) | **No** — the edit now reaches the server, but nothing reconciles the local row, and `mergeOwnTrails` then hides the server's copy behind the stale local one indefinitely (**CR-03** below), plus duplicate photo upload (**WR-13**) |
| CR-04 delete decides on `trailHasServerId` + real server DELETE | Yes (`trail_dropdown.dart:255`, `trail_sync_provider.dart:390-424`) | **Partly** — the decision reads the id through `readLocalTrail`, which returns `null` on any `toModel()` parse failure, so an unparseable row still strands a live server trail (**CR-02** below); and any DELETE failure now makes the trail permanently undeletable (**WR-15**) |

The two structurally sound fixes are CR-02 and the *decision* half of CR-04
(`trailHasServerId` is the right predicate, it is pure, and it is unit-tested at
`local_trail_store_test.dart:249-259`). The two "minimum acceptable" fixes both traded a
loud failure for a quiet one:

- CR-01's refusal is correct as a *guard* — an empty-id POST genuinely cannot be routed —
  but it was installed without the recovery path the previous review asked for ("tell the
  user to re-open the trail"). What ships is `error_saving_trail`, the same generic string
  every other failure uses, on the single most common post-record action.
- CR-03's routing is correct as far as it goes, but `_saveViaNetwork` writes only to the
  network and to the screen's own `trail` field. The ObjectBox row it just bypassed is
  still owner-scoped, still non-synced, still carrying the pre-edit name — and
  `own_trails_merge.dart:40-42` dedupes the network hit *against that row's server id*.
  The list therefore renders the pre-edit values under a green success toast, which is the
  exact failure shape CR-03 was raised to eliminate.

**On the gate tests.** The scope note is right to be suspicious. The three gates added by
the fixes (`trail_create_screen_local_save_gate_test.dart:414-555`) assert only token
*presence* and *ordering*. The CR-01 gate at `:465-503` would pass verbatim if the
`if (!trailHasServerId(...))` block had an empty body and fell through to the POST — it
checks that the substring `trailHasServerId(` appears before the substring
`trailSaveProvider.notifier)`, nothing more. None of the twelve gates in that file
constrains what is *passed* to `_saveViaNetwork`, which is where all three of CR-01's,
CR-03's and WR-13's defects live. The suite is green; the primary flow is broken.

Genuinely sound and genuinely tested, unchanged from the last pass: `sync_backoff.dart`,
`own_trails_merge.dart`, `trail_route_location.dart`, `offline_trail_filter_bounds.dart`,
`local_id.dart` (path-segment validation is thorough and correct), `local_photo_store_util.dart`
(the canonicalize-both-sides fix and the sweep are right), `unsynced_signout_guard.dart`,
and the behavioural `trail_dropdown_menu_test.dart`.

**Prior-warning disposition** — none of the twelve were resolved by the blocker fixes.
WR-01, WR-02, WR-03, WR-04, WR-07, WR-09, WR-11 and WR-12 are carried forward verbatim
(re-verified against current line numbers). WR-05, WR-06, WR-08 and WR-10 are restated:
each changed materially under the fixes without being closed. Five new warnings (WR-13 …
WR-17) come from the fixes themselves.

## Critical Issues

### CR-01: The phase's primary flow — record, save, fix a typo — now fails permanently with a generic error

**File:** `app/lib/routes/trail_create_screen.dart:449-463`, `app/lib/routes/trail_create_screen.dart:625-651`, `app/lib/routes/trail_create_screen.dart:835-842`, `app/lib/util/local_trail_store.dart:114-123`

**Issue:** The `9af61ffd` fix stops the un-routable `POST /trail/form/`. It does not restore
the flow. Trace the exact sequence the phase exists to support, with the fix in place:

1. `_finishLocalSave` sets `trail = readLocalTrail(store, localId)` (`:835-842`). The row's
   `id` is still the local sentinel at that instant, so `TrailEntity.toModel()` blanks it
   (`trail_entity.dart:320`): **`trail.id == ''`**. `_originalTrail = trail` too.
2. `_finishLocalSave` kicks `drainIfOnline()` (`:820`). The upload succeeds and
   `retireUploadedLocalTrail` **deletes the row** (`trail_sync_provider.dart:306`). The
   screen watches nothing (there are no ObjectBox `Query.watch()` streams — the file's own
   comment at `:777-783` says so), so `trail.id` stays `''` forever.
3. The hiker — still on the same screen, which never pops after a save — fixes a typo and
   taps save. `persistedLocalId != null && persisted == null` ⇒
   `LocalSaveMode.networkUpdate` (`local_trail_store.dart:119-121`), and `_onSave` calls
   `_saveViaNetwork(l10n, updatedTrail, …)` (`:456-461`).
4. `trailHasServerId(updatedTrail.id)` is **false**. The method shows `l10n.error_saving_trail`
   and returns without touching anything (`:625-651`).

Every retry reproduces step 4 identically — the only writer of `trail` on this path is
`_finishLocalSave`'s `readLocalTrail`, which now returns `null` forever. `_hasUnsavedChanges`
stays true, so backing out prompts "discard changes?", and the edit is destroyed. The user is
shown `error_saving_trail`, the same string a 500, a timeout and a malformed payload produce,
with no indication that the trail is fine, is on the server, and simply needs re-opening from
the list.

The previous review's stated minimum was: *"refuse the save **and tell the user to re-open the
trail**, rather than issuing a request that cannot possibly be routed."* Only the first half
shipped. A guard that converts a 100 %-reproducible primary-flow failure from "wrong HTTP
verb" to "generic red toast" is not a fix.

The same dead end is reachable from the `LocalUpdateOutcome.missing` branch (`:556-581`) and
from the Edit button on `trail_detail_screen.dart:168-178` for a row the drain retired while
the screen was mounted (see WR-01).

**Fix:** carry the server id across retirement so the network path has a target, and give the
refusal a message the hiker can act on.

```dart
// local_trail_store.dart -- retirement must report the id it kept
String? retireUploadedLocalTrail(Store store, String localId) {
  // ... existing body, but return entity.id before removing the row
}

// trail_sync_provider.dart -- record it where the screen can find it
final serverId = retireUploadedLocalTrail(store, localId);
if (serverId != null) _retiredLocalIdToServerId[localId] = serverId;
ref.invalidate(localTrailProvider(localId));           // WR-01

// trail_create_screen.dart -- _onSave, networkUpdate branch
final serverId = trailHasServerId(updatedTrail.id)
    ? updatedTrail.id
    : ref.read(trailSyncProvider.notifier).serverIdForRetired(persistedLocalId!);
if (serverId == null) {
  _toast(l10n.trail_uploaded_reopen_to_edit);   // NEW string, not error_saving_trail
  return;
}
await _saveViaNetwork(
  l10n,
  updatedTrail.copyWith(id: serverId, localId: null, localPhotos: const []),
  authorId: authorId,
  // the unsynced/<localId>/ copies are gone; never re-read them
  newPhotoFiles: newPhotoFiles.where((f) => f.existsSync()).toList(),
);
```

Minimum acceptable, if the id carry-forward is out of scope: add a dedicated l10n key that
names the state ("This trail finished uploading — re-open it from your trails to keep
editing") and pop back to the list, rather than leaving the hiker on a screen whose save
button can never work again. Keep `trailHasServerId` as the guard either way.

Test it behaviourally: override `apiProvider` with a recording client, drive `_onSave` from
the retired state, and assert (a) no request is issued with a `/trail/form/` path and (b) the
toast is not `error_saving_trail`. The existing source gate at
`trail_create_screen_local_save_gate_test.dart:465-503` cannot see either property.

---

### CR-02: `deleteUnsynced` decides "is there a server copy?" through a call that returns `null` on parse failure — the CR-04 orphan is still reachable

**File:** `app/lib/provider/trail/trail_sync_provider.dart:400-409`, `app/lib/util/local_trail_store.dart:482-500`, `app/lib/entities/trail_entity.dart:375`

**Issue:** The `b9ee66ca` fix reads the server id like this:

```dart
final row = readLocalTrail(store, localId);
final serverId = (row != null && trailHasServerId(row.id)) ? row.id : null;

if (serverId != null) {
  await ref.read(apiProvider).delete('/trail/$serverId');
}
deleteLocalTrailRow(store, localId);
await _deletePhotoDirBestEffort(localId);
```

`readLocalTrail` does **not** return the row's id. It returns `entity.toModel()` — and it
catches every exception from that conversion and returns `null` (`local_trail_store.dart:491-499`):

```dart
try {
  return entity.toModel();
} catch (e, st) {
  debugPrint('local_trail_store: readLocalTrail("$localId") failed to parse: $e\n$st');
  return null;
}
```

`toModel()` runs `parseGpxSafely(gpxData!)` (`trail_entity.dart:375`), which the same file's
comment states *"a `FormatException` from an unsanitized tag escaped the notifier entirely and
made the trail permanently un-openable offline once cached"*. So a row whose cached GPX no
longer parses yields `row == null` ⇒ `serverId == null` ⇒ **the server DELETE is skipped**, and
`deleteLocalTrailRow` + `_deletePhotoDirBestEffort` run unconditionally.

Result, for a row that `writeServerTrailId` already stamped: the local copy and its photos are
destroyed, the possibly-public server trail — indexed by Meilisearch, federated if public,
visible on the web profile — survives, and the device no longer holds the id that reached it.
That is the CR-04 failure verbatim, just behind a narrower trigger. And it triggers precisely
where the hiker is *most* likely to reach for delete: a trail that renders broken.

The `_confirmDelete` copy is also wrong in this state — `trail.id` there comes from a model
that *did* convert successfully, so the dialog and the executor can disagree about whether the
server holds a copy. The doc comment at `:395-399` acknowledges "the confirm dialog and this
call are not atomic", then re-checks through a lossier reader than the dialog used.

**Fix:** read the entity's id column directly; never route a delete decision through a
conversion that can silently fail.

```dart
// local_trail_store.dart -- new, alongside readLocalTrail
/// The raw persisted `TrailEntity.id` for [localId], or null when no such
/// row exists. Deliberately NOT via toModel(): a delete decision must not
/// depend on the GPX still parsing (CR-02).
String? readLocalTrailServerId(Store store, String localId) {
  final query = store.box<TrailEntity>()
      .query(TrailEntity_.localId.equals(localId)).build();
  final entity = query.findFirst();
  query.close();
  if (entity == null) return null;
  return isLocalId(entity.id) ? null : entity.id;
}

// trail_sync_provider.dart -- deleteUnsynced
final serverId = readLocalTrailServerId(store, localId);
```

Add a `deleteUnsyncedBody()` assertion to `local_trail_retirement_gate_test.dart:272-382`
pinning that `readLocalTrail(` does **not** appear in `deleteUnsynced` — the existing gates
there assert `trailHasServerId(` is present and that the DELETE precedes the local delete,
neither of which catches this.

---

### CR-03: An `alreadyUploaded` edit reaches the server and is then hidden by its own stale local row, indefinitely, under a success toast

**File:** `app/lib/routes/trail_create_screen.dart:548-582`, `app/lib/routes/trail_create_screen.dart:653-714`, `app/lib/util/own_trails_merge.dart:40-44`, `app/lib/util/local_trail_store.dart:343-345`

**Issue:** `04075d12` added `LocalUpdateOutcome.alreadyUploaded` and routes it to
`_saveViaNetwork`. The network `POST` lands and the server is correct. Nothing else is.

`_saveViaNetwork` writes to exactly two places: the remote record, and this screen's own
`trail`/`_originalTrail` fields (`:665-669`). It never touches the ObjectBox row. So after the
green `trail_saved_successfully` toast, the row for `localId` still holds the **pre-edit**
name, description, privacy and category, is still `owner`-scoped to this account, and is still
`pending`/`uploading`/`failed`.

`_invalidateOwnTrailsList()` then re-reads that row. `readOwnLocalTrails` returns it, and
`mergeOwnTrails` puts it first — and drops the network hit that carries the edit:

```dart
final localIds = local.map((t) => t.id).where((id) => id.isNotEmpty).toSet();
final dedupedNetwork = network.where((t) => !localIds.contains(t.id));
return [...local, ...dedupedNetwork];
```

The row's `id` **is** the real server id in this window (that is what `alreadyUploaded` means),
so `localIds` contains it and the server's up-to-date result is suppressed by the device's
stale one. The own-trails list shows the old name. Tapping it opens `/trail/local/<localId>`
(`trail_route_location.dart:24-27`), which renders the stale row. Editing again repeats the
whole cycle: server updated, list unchanged.

This persists until the drain's waypoint loop finally succeeds and
`retireUploadedLocalTrail` removes the row. `alreadyUploaded` exists *because* that loop is
failing — and after `kMaxSyncAttempts` the row parks as `failed`, `isDrainDue` returns false
forever (`local_trail_store.dart:149-157`), and only a manual chip tap can revive it. If the
waypoint failure is deterministic (a corrupt photo, a rejected field), the stale row is
permanent.

From the hiker's seat this is indistinguishable from the silent-loss CR-03 was raised to fix.
The loss simply moved from "the server never gets it" to "the device never shows it".

**Fix:** the local row must be reconciled whenever the network path is taken for a row that
still exists. Either write the edit into the row while preserving its sync bookkeeping, or
make the row stop shadowing the server:

```dart
// local_trail_store.dart -- new, called only after a successful network save
/// Applies [trail] to the row for [localId] WITHOUT touching id/owner/
/// localId/syncState/syncAttempts/syncNextAttemptAt/savedByUserIds, so the
/// drain's resume position is preserved while the row stops showing
/// pre-edit values (CR-03).
void applyNetworkEditToLocalRow(Store store, {
  required String localId,
  required Trail trail,
});

// trail_create_screen.dart -- after _saveViaNetwork returns successfully
if (outcome == LocalUpdateOutcome.alreadyUploaded) {
  applyNetworkEditToLocalRow(store, localId: localId, trail: result.trail);
}
```

If reconciliation is too much for this phase, the minimum is to stop the row shadowing the
server: have `mergeOwnTrails` prefer the network entry when a local row is non-synced *and*
carries a real server id, since in that state the server is authoritative for content and the
row is authoritative only for `syncState`.

`own_trails_merge_test.dart` should grow a case for "local row with a real server id and a
non-synced state" — the existing suite only covers empty-id local rows.

## Warnings

### WR-01: `localTrailProvider` is still never invalidated when the drain retires the row

**Status:** still present, unchanged. **File:** `app/lib/provider/trail/trail_sync_provider.dart:306-320`, `app/lib/provider/trail/local_trail_provider.dart:26-35`, `app/lib/routes/trail_detail_screen.dart:71-88`

**Issue:** `_drainOne` invalidates `trailLibraryProvider` and
`profileTrailsProvider('@<username>')` after retirement, but not `localTrailProvider(localId)`.
A hiker sitting on `/trail/local/<localId>` while the upload completes keeps rendering a row
that no longer exists, with a live Edit button (`trail_detail_screen.dart:168-178`) that walks
straight into **CR-01**'s dead end. Navigating away and back gives
`trail_not_on_this_device` with no path to the now-uploaded trail. This warning is now
load-bearing rather than cosmetic: it is one of the two doors into CR-01.

**Fix:**

```dart
retireUploadedLocalTrail(store, localId);
ref.invalidate(localTrailProvider(localId));
```

plus, in `trail_detail_screen.dart`'s null branch, redirect to `/trail/<serverId>` when the
retirement recorded one (CR-01's suggested return value) instead of a dead end.

---

### WR-02: `_invalidateOwnTrailsList` still interpolates a nullable username into the family key

**Status:** still present, unchanged. **File:** `app/lib/routes/trail_create_screen.dart:796-803`

**Issue:**

```dart
ref.invalidate(
  profileTrailsProvider('@${ref.read(authProvider).value?.preferredUsername}'),
);
```

When `authProvider.value` is null (token refresh in flight, mid-logout race) the key becomes
the literal `'@null'`, matching no mounted instance — the invalidation is a silent no-op. The
doc comment immediately above (`:789-791`) states "the family key must match the one
`profile_trail_screen` watches or the invalidation is a silent no-op", which is exactly what
the interpolation permits.

**Fix:**

```dart
final username = ref.read(authProvider).value?.preferredUsername;
if (username != null) ref.invalidate(profileTrailsProvider('@$username'));
```

`trail_sync_provider.dart:320` uses the non-nullable `userEntity.preferredUsername` and is
correct; this is the only site with the hole.

---

### WR-03: `TrailFilterNotifier.defaultFilter` is still unassigned on the non-connection error path

**Status:** still present, unchanged. **File:** `app/lib/provider/trail/trail_filter_provider.dart:68`, `:103-107`, `:150-152`

**Issue:** `defaultFilter` is `late` and assigned only on the success path (`:84`) and the
connection-failure fallback (`:104`). Every other failure (a 500, a malformed payload,
`TrailFilterValues.fromJson` throwing) rethrows at `:107` without assigning it.
`resetFilter()` then reads it unconditionally and throws `LateInitializationError` out of a
button callback. The provider is `keepAlive`, so the un-initialised instance survives.

**Fix:** initialise at declaration and drop `late`:

```dart
TrailFilter defaultFilter = buildDefaultTrailFilter(kOfflineTrailFilterValues);
```

This solves both this and the `late final` rebuild crash the current `late` was retained for.

---

### WR-04: A `null` waypoint `localKey` still burns retry attempts and can park a good trail as `failed`

**Status:** still present, unchanged. **File:** `app/lib/provider/trail/trail_sync_provider.dart:245-251`, compare `:147-159`

**Issue:** The `StateError` thrown for a keyless waypoint sits **inside** `_drainOne`'s `try`,
so it lands in the generic handler at `:321-332` and consumes one of the four
`kMaxSyncAttempts`. This is an invariant break, not a network condition — no retry will mint a
`localKey` — yet four passes (a lifecycle/connectivity flurry produces these within seconds;
`syncBackoffDelay` only starts at 30 s after the *first* failure) park the trail as `failed`,
after which `isDrainDue` returns false forever. The identical reasoning was applied one screen
up for the missing `UserEntity` (`:140-159`) and not here.

**Fix:** resolve keys before joining the in-flight set, and bail without recording a failure:

```dart
final keyless = entity.waypoints.where((w) => w.localKey == null && isLocalId(w.id));
if (keyless.isNotEmpty) {
  debugPrint('trail_sync_provider: "$localId" has keyless waypoints; skipping drain');
  return;   // before `state = {...state, localId}` and before the try
}
```

---

### WR-05: The three gates the fixes added assert token order only — each would pass for a guard that does nothing

**Status:** re-derived from scratch. **File:** `app/test/routes/trail_create_screen_local_save_gate_test.dart:197-243`, `:414-463`, `:465-503`, `:505-555`

The previous WR-05 text no longer describes this file. Current state, gate by gate:

- **`:414-463` (CR-02, `_localId` ordering)** — the strongest of the three. It slices the
  `createLocal` branch and asserts `saveNewLocalTrail(` precedes `_localId = localId;`. It
  would still pass if the assignment were moved into the `catch` block or after
  `_finishLocalSave`, but it does pin the specific regression. Acceptable.
- **`:465-503` (CR-01, blank-id refusal)** — asserts only that the substring
  `trailHasServerId(` appears in `_saveViaNetwork`'s body **before** the substring
  `trailSaveProvider.notifier)`. It says nothing about the guard having a body, a `return`,
  or a message. Replace the `if (!trailHasServerId(...)) { … return; }` block with
  `if (!trailHasServerId(updatedTrail.id)) { /* TODO */ }` and this gate stays green while the
  blank-id POST is fully restored. It is a token-order check named as a behaviour check.
- **`:505-555` (CR-03, `alreadyUploaded`)** — asserts the outcome name appears before
  `_finishLocalSave(`. It does not assert it is part of the `_saveViaNetwork` guard
  expression, so moving it into an unrelated `debugPrint` would satisfy it.

The pre-existing test at `:197-243` is now **factually mis-titled**: it is called *"the
updateLocal branch reaches the network ONLY on the alreadySynced escape hatch"* and its
`reason:` says the reach *"must stay guarded on that specific outcome"*, but the guard is now
`alreadySynced || missing || alreadyUploaded` (`trail_create_screen.dart:556-558`). Its actual
assertions (one `_saveViaNetwork(` in the slice; `alreadySynced` appears before it) still hold,
so it passes for a reason unrelated to the property it names. Its `networkCalls == 1` `reason:`
also claims a property of `_onSave` while the slice deliberately excludes the `networkUpdate`
branch at `:455-463`.

Crucially, **no gate in this file constrains what is passed to `_saveViaNetwork`** — which is
where CR-01, CR-03 and WR-13 all live.

**Fix:** rename `:197-243` to match the three-outcome guard and assert the full expression;
strengthen `:465-503` to require a `return;` inside the guard block; and add the behavioural
`apiProvider`-override test CR-01 needs anyway, which subsumes all three token-order gates.

---

### WR-06: The destructive-action strings are still English-only in 13 locales

**Status:** restated — the correctness half is resolved, the coverage half is not.
**File:** `app/lib/i18n/untranslated_messages.json`, `app/lib/i18n/app_de.arb`, `app/lib/i18n/app_fr.arb` (and 11 more)

**Issue:** The previous review flagged that `delete_unsynced_trail_confirm`'s English text was
*wrong* as well as untranslated. **That half is resolved by `b9ee66ca`**: `_confirmDelete` now
picks on `trailHasServerId(trail.id)` (`trail_dropdown.dart:255-257`), so a row with a server
id no longer claims the deletion is device-only and irreversible.

The coverage half is unchanged. `untranslated_messages.json` still lists 13 locales missing
`delete_unsynced_trail_confirm`, `signout_unsynced_warning`, `delete_blocked_while_uploading`,
`trail_not_on_this_device`, `sync_pending`, `sync_uploading`, `sync_failed`,
`photo_copy_failed_toast`, `own_trails_offline_banner`, `own_trails_empty_title`,
`own_trails_empty_body`, `trails_on_device` and `retry_upload`. Two of these are the copy on
irreversible actions; a non-English hiker sees English text on the one dialog that destroys
data.

**Fix:** ship `signout_unsynced_warning` and `delete_unsynced_trail_confirm` to every locale
before release; treat `lib/i18n/untranslated_messages.json` as the work list.

---

### WR-07: `retry_upload` is still a dead l10n key

**Status:** still present, unchanged. **File:** `app/lib/i18n/app_en.arb:234`

**Issue:** `retry_upload` appears in the template ARB and in all 13 locales' untranslated
report, but has no reference anywhere in `app/lib` or `app/test`. The retry affordance uses
`sync_failed` ("Upload failed · Tap to retry") as both label and action
(`sync_status_chip.dart:56-61`). It inflates the translation backlog by one string in 13
locales for nothing.

**Fix:** delete the key and regenerate, or wire it up if the chip was meant to carry a separate
action label.

---

### WR-08: An unsynced trail with a null `localId` still falls through to a silent un-download no-op — now with a live server copy at stake

**Status:** still present; consequence worsened by the CR-04 fix.
**File:** `app/lib/components/trail/trail_dropdown.dart:295`, `:370-374`

**Issue:** The unsynced branch is still guarded on
`isUnsyncedState(trail.syncState) && trail.localId != null`. A row that is unsynced but has a
null `localId` — which `retireUploadedLocalTrail`'s demote branch creates deliberately
(`local_trail_store.dart:456`) and `TrailDownloadService`'s carry-forward can produce from
`existing?.localId` (`trail_download_service.dart:212`) — skips it and lands in the
`trail.isLocal` branch (`:370`), which pops the route and calls
`trailLibraryProvider.deleteTrail(trail.id)`.

Before CR-04 this "merely" no-oped on an empty id. Now it is worse: `_confirmDelete` chose its
copy on `trailHasServerId(trail.id)` (`:255`), so a row in this shape with a real server id was
shown `delete_trail_confirm` — implying a server delete — and then routed to a local
un-download that never contacts the server at all. The dialog and the executor disagree.

**Fix:** make the fall-through explicit and never let it reach the un-download branch:

```dart
if (isUnsyncedState(trail.syncState)) {
  final localId = trail.localId;
  if (localId == null) {
    // No local handle: this is not a device-only copy. Route to the server
    // delete rather than silently un-downloading.
    return _deleteOnServer(context, trail);
  }
  // ... existing deleteUnsynced flow
}
```

---

### WR-09: `writeServerWaypointId` still clears `localPhotos` before the trail's upload has completed

**Status:** still present, unchanged. **File:** `app/lib/util/local_trail_store.dart:734-737`, `app/lib/provider/trail/trail_sync_provider.dart:281-287`

**Issue:** The bookkeeping write sets `target.photos = serverPhotoFilenames` and
`target.localPhotos = []` the moment one waypoint is created. If a **later** waypoint in the
same loop fails and the trail parks as `failed`, the row survives with waypoints whose
model-level photos are server filenames and whose local copies are unreachable — while the
JPEGs still sit on disk under `unsynced/<localId>/waypoints/<key>/` (nothing deletes them on
the failure path, `trail_sync_provider.dart:322-324`). Viewing that trail offline shows a
waypoint with zero photos, and the files are dead weight until the trail is deleted.

**Fix:** keep the local copies until the whole trail is retired:

```dart
target.id = serverWaypointId;
target.photos = serverPhotoFilenames;
// localPhotos deliberately retained -- the trail's upload is not complete yet,
// and retireUploadedLocalTrail / deleteUnsyncedPhotoDir own the cleanup.
```

---

### WR-10: The local delete path still has no owner clause — and the CR-04 fix added a third unscoped read on the same path

**Status:** restated — the exposure changed shape.
**File:** `app/lib/util/local_trail_store.dart:394-408`, `:440-471`, `:482-500`, `app/lib/provider/trail/trail_sync_provider.dart:400`, `:409`

**Issue:** `readOwnLocalTrail` (`:516-541`) is owner-scoped and its doc comment explains why:
its argument arrives from a route parameter that "survives a logout in the deep-link and
back-stack". The delete path takes the same kind of value from the same kind of source —
`TrailSync.deleteUnsynced(trail.localId!)`, driven by a `Trail` that may have been constructed
before an account switch — and now makes **two** unscoped reads plus one unscoped write:
`readLocalTrail` (`trail_sync_provider.dart:400`), then `deleteLocalTrailRow` (`:409`), neither
with an `owner` clause.

The CR-04 fix partly mitigates the *server* half by accident: with a server id present the
`DELETE` runs under the current session and a cross-account attempt would 403 and throw before
`deleteLocalTrailRow`. But when `serverId` is null — the common case, and the case CR-02 above
widens — nothing stops account B destroying account A's device-only row.

**Fix:** thread the account id through and add the clause, matching `readOwnLocalTrail`:

```dart
void deleteLocalTrailRow(Store store, String localId, {required String accountId}) {
  final query = trailBox.query(
    TrailEntity_.localId.equals(localId) & TrailEntity_.owner.equals(accountId),
  ).build();
  // ...
}
```

Apply the same to `readLocalTrailServerId` (CR-02's suggested reader) and to
`resetDrainBackoff` (`:802`), which `SyncStatusChip`'s retry tap (`sync_status_chip.dart:58-61`)
reaches with the same kind of value.

---

### WR-11: `TrailPanel` still hides the summit-log and comment tabs for every trail read off the device

**Status:** still present, unchanged. **File:** `app/lib/components/trail/trail_panel.dart:242-254`, `:404-408`, `app/lib/entities/trail_entity.dart:355`

**Issue:** The `TabBar` is gated on `!trail.isLocal`, and `TrailEntity.toModel()` hardcodes
`isLocal: true` for **every** row, downloaded trails included. `_TabContent` still renders
`children[_index]` with `_index` pinned at 0, so summit logs and comments are unreachable on any
trail served from ObjectBox — including the case `trail_dropdown.dart:361-365` calls out by
name: `TrailNotifier.build()` falls back to the cache on *any* fetch exception, so one timeout
on your own downloaded trail hides its comments until the provider refetches.

`trail_dropdown.dart:48-53` documents that `isLocal` stopped being a usable "device-only" signal
in this phase and switches to `isUnsynced`; `trail_panel.dart` was edited in the same plan and
was not brought onto the same footing.

**Fix:**

```dart
if (!isUnsyncedState(trail.syncState))
  TabBar(/* ... */),
```

and derive `_TabContent`'s child count from the same predicate so a local trail cannot index
past the About tab.

---

### WR-12: `LocalTrailMetrics`' "three parallel lists" contract is not what the query returns

**Status:** still present, unchanged. **File:** `app/lib/util/offline_trail_filter_bounds.dart:18-26`, `:189-198`

**Issue:** The typedef is documented as "Three parallel lists of raw on-device column values".
`PropertyQuery<double>.find()` **excludes null values** unless `replaceNullWith` is supplied
(objectbox 5.3.1), and `distance`, `elevationGain` and `elevationLoss` are all nullable on
`TrailEntity`. The three lists therefore have different lengths and no positional
correspondence to each other or to the row set. `computeOfflineTrailFilterValues` only takes
per-axis maxima, so this is harmless today — but the comment invites the next reader to zip
them, which would pair one trail's distance with another's elevation.

**Fix:** correct the doc to "three independent, null-dropped value lists, one per axis — NOT
row-aligned", or pass `replaceNullWith: 0` on all three if alignment is ever wanted.

---

### WR-13: The `alreadyUploaded` network save re-uploads every photo the drain already uploaded

**Status:** new, introduced by `04075d12`.
**File:** `app/lib/routes/trail_create_screen.dart:406-407`, `:1301-1307`, `app/lib/provider/trail/trail_save_provider.dart:126-136`, `app/lib/util/form_data_util.dart:47-49`

**Issue:** The photo form field's `initialValue` is `trail.localPhotos`
(`trail_create_screen.dart:1303`). For a row in the `alreadyUploaded` window,
`TrailEntity.toModel()` returns the app-owned copies under `unsynced/<localId>/`
(`trail_entity.dart:360`) — the *same files* the drain's step 2 already uploaded as part of
`PUT /trail/form` (`trail_sync_provider.dart:184`). `_onSave` turns them into
`newPhotoFiles = localPhotoPaths.map((p) => File(p))` (`:406-407`) and hands them to
`_saveViaNetwork` → `updateTrail(newPhotos: …)`, which emits them under the `photos+` key
(`form_data_util.dart:47-49`) — **append**, not replace.

Every save in this window therefore doubles the trail's server-side photo set. The photo picker
also renders them twice (`initialWebPhotos: trail.photos` holds the server filenames written by
`writeServerTrailId`, `initialLocalPhotos: trail.localPhotos` holds the same images as local
paths), so the duplication is visible before the save even runs.

**Fix:** the network path must not re-send photos that already exist server-side. Either drop
`newPhotoFiles` entirely when the trail already carries a server id and no *new* file was
picked, or diff against `trail.photos`:

```dart
final alreadyOnServer = trail.photos.map(p.basename).toSet();
final newPhotoFiles = localPhotoPaths
    .where((path) => !alreadyOnServer.contains(p.basename(path)))
    .map((p) => File(p))
    .toList();
```

---

### WR-14: `_copyPhotosForLocalSave` runs before the routing decision, so a refused write still mutates the photo directory and its failure count is discarded

**Status:** new, surfaced by `04075d12`'s added outcome.
**File:** `app/lib/routes/trail_create_screen.dart:542-582`, `app/lib/util/local_photo_store_util.dart:131-151`

**Issue:** In the `updateLocal` branch, `_copyPhotosForLocalSave` is awaited at `:542-546`,
**before** `updateLocalTrail` at `:548` reveals that the write will be refused. Two
consequences:

1. `reconcileLocalPhotos` deletes any file in `unsynced/<localId>/` that is not in the kept set
   (`local_photo_store_util.dart:143-151`). If the hiker removed a photo in the form and the
   outcome comes back `alreadyUploaded`, the file is deleted from disk while the row still
   lists it in `localPhotos` — the local detail screen then renders a broken image until the
   drain retires the row.
2. On the `missing` outcome (the row was retired), the copy re-creates the directory the
   retirement just deleted. The startup sweep (`main.dart:120-123`) reclaims it, so this
   self-heals, but only at next launch.

Additionally `photoCopy.failedCount` is silently dropped on all three network routes — a photo
that failed to copy is reported by `photo_copy_failed_toast` on the local path (`:822-832`) and
by nothing at all here.

**Fix:** call `updateLocalTrail` first with the *existing* photo lists, branch on the outcome,
and only reconcile photos on the `updated` path. If the ordering must stay, at minimum surface
`failedCount` before routing to `_saveViaNetwork`.

---

### WR-15: A failed server DELETE now makes the trail permanently undeletable, with an unlocalised message

**Status:** new, introduced by `b9ee66ca`.
**File:** `app/lib/provider/trail/trail_sync_provider.dart:403-407`, `app/lib/components/trail/trail_dropdown.dart:316-333`

**Issue:** `deleteUnsynced` lets **every** failure from `api.delete('/trail/$serverId')` escape
uncaught, and `_deleteTrail` catches it into a generic toast and returns without deleting
anything. That is right for a transient network failure. It is wrong for the three states where
the server delete can never succeed:

- **404** — the trail was already deleted from the web UI or another device. The local row can
  now never be removed; every attempt shows "Error deleting trail" forever.
- **403 / 401** — the row belongs to another account (WR-10's stale-widget vector) or the
  session expired. Same permanent block.
- **Offline** — a hiker on a multi-day trip whose upload got as far as `PUT /trail/form` and
  then stalled cannot delete the trail at all until they regain signal. This is a functional
  regression against the phase's own local-first premise; before the fix the delete worked.

The toast text is also a hardcoded English literal (`trail_dropdown.dart:329`), duplicating the
one at `:394`, in a file that otherwise routes every string through `l18n`. And `catch (e)`
binds `e` without using it, so the actual failure is not even logged.

**Fix:** classify the failure rather than treating all of them as retryable.

```dart
try {
  await ref.read(apiProvider).delete('/trail/$serverId');
} on DioException catch (e) {
  final status = e.response?.statusCode;
  // Already gone server-side -- proceed with the local delete.
  if (status != 404) rethrow;
}
```

and in `_deleteTrail`, replace the literal with a localised key, log the exception, and offer a
"delete on this device only" second confirmation for the offline case so the hiker is not
stuck.

---

### WR-16: `resolveLocalSaveMode`'s doc comment now contradicts the behaviour the CR-03 fix installed

**Status:** new, introduced by `04075d12`.
**File:** `app/lib/util/local_trail_store.dart:61-65`, compare `:312-326` and `:343-345`

**Issue:** `resolveLocalSaveMode`'s doc still asserts:

> A trail with a non-empty server id whose `Trail.syncState` is NOT `TrailSyncState.synced`
> (still `uploading` or `pending`, the mid-drain resume case) is **deliberately** routed to
> `LocalSaveMode.updateLocal`, not `LocalSaveMode.networkUpdate` — the server id exists but the
> row is not yet confirmed synced, **so the safe write target is still the local row.**

That is now false, and the file says so 250 lines further down: `updateLocalTrail` refuses
exactly this row with `LocalUpdateOutcome.alreadyUploaded` because writing it locally *"would
sit on the row until a later retry succeeds and `retireUploadedLocalTrail` destroys it"*
(`:312-326`). Two load-bearing doc comments in one file give opposite guidance on the same row
shape. The next reader who trusts the first one will re-open CR-03.

**Fix:** amend `:61-65` to say that `updateLocal` is the routing *decision* but that
`updateLocalTrail` is the authority on whether the local row is a legal write target, and
cross-reference `LocalUpdateOutcome.alreadyUploaded`.

---

### WR-17: The new server DELETE interpolates an unvalidated record id into a request path

**Status:** new, introduced by `b9ee66ca`.
**File:** `app/lib/provider/trail/trail_sync_provider.dart:406`, compare `app/lib/util/local_id.dart:74-84`

**Issue:** `await ref.read(apiProvider).delete('/trail/$serverId');` interpolates a value that
originated in a server response body (`trail_sync_provider.dart:218`, `rawBody['id']`) straight
into a Dio path, guarded only by `trailHasServerId` — which checks `isNotEmpty` and nothing
else. A federated or compromised instance returning an id containing `../` steers the request
at a different endpoint under the same authenticated session.

This phase introduced `recordIdDirSegment` (`local_id.dart:74-84`) for precisely this class of
value, with the reasoning *"a record id arrives over the network from a server that may be
federated or compromised"*, and `trail_download_service.dart:63,89` already applies it. The new
call site does not. (`trail_save_provider.dart:203` has the same shape and predates this phase;
both should be brought onto the same footing.)

**Fix:** validate at the point the id is persisted, so every consumer inherits it:

```dart
// trail_sync_provider.dart, step 2
if (rawId is String && rawId.isNotEmpty) {
  writeServerTrailId(store, localId: localId, serverId: recordIdDirSegment(rawId), ...);
}
```

or, at minimum, `delete('/trail/${recordIdDirSegment(serverId)}')` at the call site.

---

_Reviewed: 2026-08-03T17:40:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
