---
phase: 36-local-first-recording-automatic-upload
verified: 2026-08-02T15:48:59Z
status: human_needed
score: 6/6 roadmap success criteria verified (41/41 plan-level must-have truths verified)
overrides_applied: 0
human_verification:
  - test: "End a recording (or import a GPX) in airplane mode, fill in title and pick two photos, tap Save; re-open the trail from the own-trails list, change the title, save again."
    expected: "First save succeeds with the success toast and no offline-error. The trail appears once in the own-trails list, badged as not-yet-uploaded. The re-save updates the same trail (still one entry, not two)."
    why_human: "Requires driving ImagePicker, ObjectBox and the full widget tree on a real device; PLAN 36-06 deferred this to end-of-phase human-check (no live Store in `flutter test`)."
  - test: "In airplane mode, open your own profile's own-trails list (`/profile/<handle>/trails`)."
    expected: "The list renders (does not error or spin forever), shows the offline banner text plainly stating it is showing only what's on-device, includes every not-yet-uploaded trail plus authored trails you've downloaded. Tapping an unsynced trail opens the offline-capable edit screen with its title/photos populated. With nothing saved, the empty state shows the cloud-up icon and 'Nothing saved yet' copy."
    why_human: "Needs a real connectivity transition and a populated ObjectBox store; PLAN 36-07 deferred this to end-of-phase human-check."
  - test: "Open the trail dropdown menu on an unsynced trail: check Download is absent (not just disabled) and Delete shows an 'cannot be undone' confirmation. Start (or wait for) that trail's upload and reopen the menu mid-upload. Then check the same menu on an ordinary downloaded trail."
    expected: "Unsynced trail: no Download entry; Delete confirmation states the deletion is unrecoverable. Mid-upload: Delete is greyed out / disabled. Downloaded trail: menu unchanged from today, Delete still only removes the local download."
    why_human: "Requires a live drain in progress and real menu interaction; PLAN 36-08 deferred this to end-of-phase human-check."
  - test: "With the app foregrounded and a working connection (or by regaining connectivity while the app stays open), watch an unsynced trail upload without tapping anything. Separately, force-kill the app (or otherwise interrupt) partway through an upload — e.g. after the trail record is created but before all waypoints/photos finish — then relaunch/reconnect and let the drain resume."
    expected: "The trail's badge transitions Pending -> Uploading -> (badge disappears) with no user action beyond having the app open and online. After an interrupted-and-resumed upload, exactly one trail (and one row per waypoint) exists on the server — no duplicates — and the local row shows no badge (synced) with its photos intact."
    why_human: "SYNC-01/SYNC-04/SYNC-05's duplicate-prevention chain (trail-id-then-waypoint-id write-back, CR-01/CR-02/CR-03/WR-01 fixes) is verified by code inspection and unit tests of the pure decision logic, but no automated test in this repo exercises a live PocketBase server or a real ObjectBox `Store` (confirmed untestable in `flutter test` — `libobjectbox.dylib` fails to load). An end-to-end device+server pass is the only way to confirm no duplicate is produced under a genuine mid-drain interruption."
---

# Phase 36: Local-First Recording & Automatic Upload Verification Report

**Phase Goal:** A hiker who records a trail or uploads a GPX with no signal can save it, review it, and fill in its details on the spot — and it uploads itself the next time the phone has a connection, without the hiker doing anything.

**Verified:** 2026-08-02T15:48:59Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (Roadmap Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Capturing with no connection saves immediately into the own-trails list (recording end or GPX import), no offline save-failure ever shown, unsynced visibly distinct from synced AND downloaded | ✓ VERIFIED | `trail_create_screen.dart:451-506` (`createLocal` branch) and `saveNewLocalTrail` (`local_trail_store.dart:153-195`) never touch the network — no `trailSaveProvider`/`apiProvider`/`onlineStatusProvider` reference in either local-first branch (enforced by `trail_create_screen_local_save_gate_test.dart`). `SyncStatusChip` (`sync_status_chip.dart`) renders Pending/Uploading/Failed, nothing when synced; D-10 makes unsynced/downloaded mutually exclusive by construction (`savedByUserIds` is never written for an unsynced row — confirmed in `saveNewLocalTrail`/`updateLocalTrail`, neither sets it). |
| 2 | Survives app restart, stays tied to the capturing account; a different account never sees/uploads it; logout never deletes it | ✓ VERIFIED | `TrailEntity` is a persisted ObjectBox row (survives restart structurally); every read/write in `local_trail_store.dart` is `owner`-scoped (`TrailEntity_.owner.equals(accountId)` in `readOwnLocalTrails`, `selectDrainCandidates`, `countUnsyncedTrails`); `currentAccountId(store)` is read fresh at every call site, never cached (verified in `_drainPass`, `_readOwnLocal`, `saveNewLocalTrail` call site) — closes WR-06's stale-cache defect. `account_data_purge_util.dart` does not purge `TrailEntity`/`WaypointEntity` on logout or account switch. |
| 3 | Open/review/edit an unsynced trail's title, description, category, photos while offline, on the Phase-35 offline-capable screen | ✓ VERIFIED | `resolveLocalSaveMode` routes a re-save of an unsynced trail to `updateLocalTrail`, fully local, no network (`local_trail_store.dart:65-73`, `234-290`). CR-01's fix persists `category`/`subcategory`/`completed`/`tags` on `TrailEntity` and restores them in `toModel()` — previously silently destroyed. Route re-editing correctly stays out of scope (D-16; edit-route button hidden offline per Phase 35, `trail_create_screen.dart:836,858`). |
| 4 | Once foregrounded with connection, uploads on its own with inline per-item progress on the trail itself (not a separate screen), manual retry on failure/stall | ✓ VERIFIED | `main.dart`: `didChangeAppLifecycleState` (foreground), `onlineStatusProvider` false→true `listenManual` (regained connectivity), and a one-shot cold-start kick all call `trailSyncProvider.notifier.drainIfOnline()` — 3 trigger sites confirmed. `SyncStatusChip` renders progress/failure inline on `TrailCard`/`TrailListItem` (both wired, `grep` confirms `SyncStatusChip(trail: trail)` in both). Tapping a Failed chip calls `retry(localId)`; WR-03's fix (`_rerunRequested` flag) makes retry work even while another trail is mid-drain, not a silent no-op. |
| 5 | An interrupted upload never produces a duplicate trail on retry; once uploaded the trail becomes ordinary in place, keeping its identity rather than appearing twice | ✓ VERIFIED (code-level; live-server behavior needs device verification — see human_verification) | All four review BLOCKERs that threatened this were independently re-read in the fixed code and confirmed fixed: CR-01 (category/tags no longer dropped before upload), CR-02 (`WaypointPhotoUploadException` carries the created id so a photo-only failure doesn't re-`PUT /waypoint`), CR-03 (`writeServerTrailId` now commits `serverPhotoFilenames` in the same transaction as the id, so a resumed drain can't wipe photos), CR-04 (`_onSave` now routes on the persisted row, not a stale screen snapshot, so a post-upload edit reaches the network instead of being silently stranded). WR-01 additionally moves the trail-id write-back before `Trail.fromJson` so a parse failure can't cause a duplicate create. `markTrailSynced` keeps the row's `obxId` unchanged (same row promoted, never deleted+recreated) and `mergeOwnTrails` dedupes any network hit sharing the id — confirmed by `own_trails_merge_test.dart`. |
| 6 | With no connection the own-trails list still renders, shows every not-yet-uploaded trail plus authored-and-downloaded trails, and plainly states it's offline-only | ✓ VERIFIED | `profile_trails_provider.dart`'s `_fetchAndMerge` swallows only `DioException` (WR-09 fix — a parse/type bug no longer masquerades as "offline") and sets `offline: true` for the signed-in hiker's own handle; `readOwnLocalTrails` (`local_trail_store.dart:360-393`) returns both `owner`-matched rows and `author`-matched downloaded rows in one flat list per D-11. `profile_trail_screen.dart:94-124` renders a persistent banner (`own_trails_offline_banner`) when `state.offline && state.isOwnHandle`. |

**Score:** 6/6 roadmap success criteria verified. All 41 plan-level `must_haves.truths` across the 8 plans (36-01 through 36-08) were independently checked against the current code — none failed.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `app/lib/models/trail_sync_state.dart` | `TrailSyncState` enum, `synced` at index 0 | ✓ VERIFIED | Confirmed enum order and doc comment explaining the ObjectBox-default load-bearing reason. |
| `app/lib/util/local_id.dart` | Local id minting/validation, `recordIdDirSegment`/`fileNameSegment` | ✓ VERIFIED | Present, behaviorally tested (`local_id_test.dart`), reused by `trail_download_service.dart` (WR-14 fix). |
| `app/lib/entities/trail_entity.dart` | owner/localId/localPhotos/syncState + CR-01's category/subcategory/tags/completed columns | ✓ VERIFIED | All fields present, `fromModel`/`toModel` round-trip verified by reading and by `trail_entity_test.dart`'s CR-01 group. |
| `app/lib/entities/waypoint_entity.dart` | `localKey` + `localPhotos` persistence | ✓ VERIFIED | Present; both GPX-derived and photo-EXIF-derived waypoints mint `id: ''` with a `localKey` (D-06 fix confirmed in `trail_create_screen.dart:177,214` and `gpx_conversion_util.dart:531,584`). |
| `app/lib/util/local_photo_store_util.dart` | App-owned copy/reconcile/delete/orphan-sweep | ✓ VERIFIED | Present; WR-13's canonicalization fix confirmed (`p.canonicalize` on both sides of the delete-pass comparison). |
| `app/lib/util/local_trail_store.dart` | Single sanctioned read/write layer, `resolveLocalSaveMode`, drain bookkeeping | ✓ VERIFIED | 651 lines, all owner-scoped reads confirmed, `LocalUpdateOutcome.alreadySynced` refusal (CR-04) confirmed, `markTrailUploading`/`markTrailSynced`/`writeServerTrailId` transactional writes confirmed. |
| `app/lib/services/trail_download_service.dart` | Local bookkeeping carried forward on re-download | ✓ VERIFIED | Path construction now routed through `recordIdDirSegment`/`fileNameSegment` (WR-14). |
| `app/lib/util/sync_backoff.dart` | D-07 backoff curve, `kMaxSyncAttempts` | ✓ VERIFIED | Pure, clamped, tested (`sync_backoff_test.dart`). |
| `app/lib/provider/trail/trail_sync_provider.dart` | `keepAlive` drain notifier, resume-from-step, retry, delete | ✓ VERIFIED | 383 lines; in-flight `Set<String>` state, `_drainPass`/`_drainOne` resume logic, `retry()`, `deleteUnsynced()` all present and match the fix report's described shape. |
| `app/lib/util/unsynced_signout_guard.dart` | Count-and-confirm sign-out gate | ✓ VERIFIED | Wired into `settings_screen.dart`'s logout button (confirmed before `logout()` call); `settings_account_screen.dart`'s exemption documented and confirmed deliberate. |
| `app/lib/routes/trail_create_screen.dart` | Three-way `_onSave` branch, photo copy, local-first write | ✓ VERIFIED | `resolveLocalSaveMode` routing on the persisted row (CR-04 fix), all three `LocalSaveMode` branches present, local branches confirmed network-free. |
| `app/lib/util/own_trails_merge.dart` | Pure local+network merge, dedupe by non-empty server id | ✓ VERIFIED | `mergeOwnTrails`/`filterOwnTrailsByQuery` present, dedupe-guard logic matches doc comment, tested (`own_trails_merge_test.dart`, 10 tests). |
| `app/lib/provider/profile/profile_trails_provider.dart` | Local-first own-trails state, offline flag | ✓ VERIFIED | `readOwnLocalTrails` call in `build()`, WR-06/WR-09 fixes confirmed present. |
| `app/lib/routes/profile_trail_screen.dart` | Offline banner, offline empty state, unsynced tap routing | ✓ VERIFIED | `own_trails_offline_banner`/`own_trails_empty_title` rendered; `_onTrailSelect` routes an unsynced trail to `/trail/create/edit`. |
| `app/lib/components/trail/sync_status_chip.dart` | Four-state Pending/Uploading/Failed/absent chip | ✓ VERIFIED | Matches D-08 exactly; widget-tested for all four states (`sync_status_chip_test.dart`, 5 `testWidgets`). |
| `app/lib/components/trail/trail_dropdown.dart` | Three-way delete branch, hidden download for unsynced, unrecoverable-delete confirm | ✓ VERIFIED | `isUnsyncedState` branch ordered before the `isLocal` (downloaded) branch — required ordering confirmed; WR-02 fix (refused-delete toast, no premature pop) confirmed present. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `trail_entity.dart` | `local_id.dart` | `toModel()` blanks a local sentinel id | ✓ WIRED | `id: isLocalId(id) ? '' : id` confirmed at `trail_entity.dart:292`. |
| `local_trail_store.dart` | `current_account.dart` | fresh account-id read on every query/write | ✓ WIRED | Confirmed at every call site read (`_drainPass`, `_readOwnLocal`, sign-out guard, `_onSave`). |
| `trail_sync_provider.dart` | `local_trail_store.dart` | owner-scoped candidate selection + per-step write-back | ✓ WIRED | `selectDrainCandidates`, `writeServerTrailId`, `writeServerWaypointId`, `markTrailUploading`, `markTrailSynced`, `recordDrainFailure` all called in sequence inside `_drainOne`. |
| `main.dart` | `trail_sync_provider.dart` | lifecycle + connectivity + cold-start triggers | ✓ WIRED | 3 call sites to `drainIfOnline()` confirmed (foreground, connectivity transition, cold start). |
| `sync_status_chip.dart` | `trail_sync_provider.dart` | Failed state taps through to `retry()` | ✓ WIRED | `onTap: () => ref.read(trailSyncProvider.notifier).retry(localId)` confirmed. |
| `trail_dropdown.dart` | `trail_sync_provider.dart` | delete refused while in-flight | ✓ WIRED | `isDraining` menu-disable plus `deleteUnsynced()`'s own in-flight refusal (defense in depth), confirmed with WR-02's surfaced-refusal fix. |
| `profile_trails_provider.dart` | `own_trails_merge.dart` | merge local + network deduped by server id | ✓ WIRED | `mergeOwnTrails(local: local, network: networkTrails)` confirmed in `_fetchAndMerge`. |
| `waypoint_provider.dart` | `trail_sync_provider.dart` | photo-upload-only failure surfaces the created id | ✓ WIRED | `WaypointPhotoUploadException` thrown by `WaypointSave.create`, caught in `_drainOne`, id persisted via `writeServerWaypointId` before rethrow (CR-02). |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|---------------------|--------|
| `SyncStatusChip` | `trail.syncState` / `trailSyncProvider` in-flight set | `TrailSummary.syncState` (persisted `dbSyncState`) + live Riverpod state | Yes — persisted enum backed by real ObjectBox column, in-flight set populated by the actual drain | ✓ FLOWING |
| `profile_trail_screen.dart`'s trail list | `ProfileTrailsState.trails` | `mergeOwnTrails(readOwnLocalTrails(...), network fetch)` | Yes — real ObjectBox query + real `/profile/<handle>/trails` fetch, not a static stub | ✓ FLOWING |
| `TrailCard`/`TrailListItem` thumbnail | `trail.localPhotos` / `trail.photos` | `TrailEntity.toModel()` | Mostly yes; one acknowledged pre-existing-shape edge case (see Anti-Patterns) where a just-synced row's `localPhotos` getter falls back to server filenames — guarded by `File(...).existsSync()` so it degrades to the network thumbnail rather than breaking | ⚠️ FLOWING (documented edge case, non-blocking) |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full test suite passes | `cd app && flutter test` | 814 passing, 1 pre-existing skip, 0 failures | ✓ PASS |
| Analyzer clean | `cd app && flutter analyze --no-pub` | 0 errors, 0 warnings, 35 pre-existing `info` lints (vendored/deprecated-icon noise) | ✓ PASS |
| No debt markers in phase-touched files | `grep -n -E "TBD\|FIXME\|XXX\|TODO\|HACK\|PLACEHOLDER"` across all 23 phase-touched lib files | No matches | ✓ PASS |
| Live device/server upload cannot be exercised here | — | — | ? SKIP — see human_verification |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes exist for this phase or this project layout (Flutter/Dart, not the shell-probe convention). Step 7c: SKIPPED — no conventional or PLAN-declared probes found.

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|-----------------|--------------|--------|----------|
| REC-01 | 36-01, 36-02, 36-06 | Captures with no connection save, no offline-caused failure shown | ✓ SATISFIED | Local-first `_onSave` branches confirmed network-free. |
| REC-02 | 36-03, 36-07 | Saved unsynced trail appears in own-trails list immediately, not Library | ✓ SATISFIED | `readOwnLocalTrails` + `mergeOwnTrails` place local rows first. |
| REC-03 | 36-01, 36-02, 36-08 | Unsynced visibly distinct from synced and from downloaded | ✓ SATISFIED | `SyncStatusChip` + D-10 mutual exclusivity. |
| REC-04 | 36-01, 36-03, 36-04, 36-05, 36-07 | Survives restart, account-scoped, logout doesn't delete | ✓ SATISFIED | Owner-scoped queries + purge-util exclusion + sign-out warning guard. |
| REC-05 | 36-02, 36-03, 36-06, 36-07 | Open/review/edit unsynced trail's metadata offline | ✓ SATISFIED | `updateLocalTrail` local-first path + CR-01 field-persistence fix. |
| REC-06 | 36-02, 36-07 | Own-trails list renders offline, states offline-only plainly | ✓ SATISFIED | Offline banner + local+downloaded merge. |
| SYNC-01 | 36-04 | Uploads on its own on foreground with connection | ✓ SATISFIED | 3 confirmed trigger sites in `main.dart`. |
| SYNC-02 | 36-02, 36-08 | Progress/failure visible inline, not a separate screen | ✓ SATISFIED | `SyncStatusChip` on `TrailCard`/`TrailListItem`. |
| SYNC-03 | 36-04, 36-08 | Manual retry for failed/stalled upload | ✓ SATISFIED | `retry()` wired to the Failed chip's tap, WR-03 fix confirmed. |
| SYNC-04 | 36-01, 36-03, 36-04, 36-06 | Interrupted upload never duplicates on retry | ✓ SATISFIED (code-level) | CR-01/CR-02/CR-03/WR-01 fix chain confirmed by direct code reading; live-server confirmation is a human_verification item. |
| SYNC-05 | 36-01, 36-03, 36-04, 36-07 | Uploaded trail keeps identity in place, not duplicated in list | ✓ SATISFIED | `markTrailSynced` keeps `obxId`; `mergeOwnTrails` dedupes by server id, tested. |

No orphaned requirements — all 11 IDs assigned to this phase in `.planning/REQUIREMENTS.md` are claimed by at least one plan's `requirements` frontmatter.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `app/lib/i18n/app_de.arb`, `app_fr.arb`, and 10 other locale files | — | Missing translations for `signout_unsynced_warning` and `delete_unsynced_trail_confirm` (WR-15, partially fixed) | ⚠️ Warning | Two destructive-action/data-safety strings fall back to raw English for 12 non-English locales. Reviewer/fixer both flagged this as deliberately not resolved — authoring destructive-action copy needs native-speaker review, which the fixer correctly declined to fabricate. Non-blocking per the project's documented l10n convention (English fallback for missing keys), but worth routing to a translator before a non-English release. `lib/i18n/untranslated_messages.json` (added by WR-15) itemizes the full gap. |
| `app/lib/entities/trail_entity.dart:328` | 328 | `localPhotos: localPhotos.isNotEmpty ? localPhotos : photos` pre-existing fallback shape, noted by the fixer as adjacent-but-unaddressed | ℹ️ Info | A just-synced local row's `Trail.localPhotos` getter can report server-side filenames rather than an empty list. `TrailCard`'s consumption guards with `File(path).existsSync()`, so this degrades to the network thumbnail rather than crashing or showing a broken image — confirmed by reading `trail_card.dart:41-52`. Documented in `36-REVIEW-FIX.md`'s "Notes for the verifier" as deliberately out of scope. |

No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` markers found in any of the 23 files this phase touched. No stub returns, no empty-handler patterns, no hardcoded-empty data flowing to rendered UI.

### Human Verification Required

See frontmatter `human_verification` for the structured list. Four items, all device-only (three explicitly deferred by the planner per `workflow.human_verify_mode = end-of-phase`, one added by this verification because it is the only way to confirm SYNC-04/SYNC-05's duplicate-prevention chain end-to-end against a live server):

1. Offline recording save + re-save round trip (from PLAN 36-06)
2. Offline own-trails list render, banner, tap-to-edit, empty state (from PLAN 36-07)
3. `trail_dropdown` Download-hidden / Delete-confirm-wording / Delete-disabled-mid-upload (from PLAN 36-08)
4. Live foreground/reconnect auto-upload plus an interrupted-and-resumed upload producing no server-side duplicate (added by verification — no automated test in this repo can open a live ObjectBox `Store` or hit a real PocketBase server; `libobjectbox.dylib` fails to load under `flutter test` per `36-REVIEW-FIX.md`'s own confirmation)

### Gaps Summary

No BLOCKER-level gaps. All four review CRITICAL findings (CR-01 through CR-04) and 14 of 15 WARNING findings were independently re-verified in the current code, not merely trusted from `36-REVIEW-FIX.md`'s narrative — each fix's actual diff was read and cross-checked against the review's description of the defect. The one partially-fixed WARNING (WR-15, non-English translations for two destructive-action strings) is a content/localization gap, not a functional one, and is explicitly non-blocking per the project's English-fallback l10n convention.

The phase's own test suite (814 passing) and analyzer (0 errors/warnings) both pass. The remaining risk is exactly what the fixer's own notes flag: `Store`-touching and network-touching behavior (the actual drain against a real server, actual ObjectBox persistence across a real app restart) cannot be exercised by any test in this harness and needs a device+server pass. That is reflected as `human_needed`, not as a gap — the code-level argument for correctness (id-before-photos write-back, in-place row promotion, dedupe-by-server-id) is sound and directly inspected, but it has not been exercised end-to-end against live infrastructure.

---

_Verified: 2026-08-02T15:48:59Z_
_Verifier: Claude (gsd-verifier)_
