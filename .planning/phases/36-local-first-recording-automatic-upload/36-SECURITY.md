---
phase: 36
slug: local-first-recording-automatic-upload
status: verified
threats_open: 0
asvs_level: 1
created: 2026-08-02
---

# Phase 36 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

All 43 threats declared across `36-01-PLAN.md` through `36-08-PLAN.md`'s `<threat_model>` blocks were independently re-verified against the CURRENT implementation (post `36-REVIEW.md` / `36-REVIEW-FIX.md`, which reshaped several mitigations — notably CR-01 through CR-04 and WR-01 through WR-14). Documentation/doc-comment claims were not accepted as evidence; each row below cites the actual code read.

No `## Threat Flags` section exists in any of `36-01-SUMMARY.md` through `36-08-SUMMARY.md` — confirmed by direct grep across all eight files. No new/unregistered attack surface was flagged by any executor.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| account A → account B (same device) | `TrailEntity.owner` is the only separator between accounts' captures in a shared ObjectBox store | Trail/waypoint content |
| local id → filesystem path | `localId` becomes a directory segment for photo storage | Path segments |
| device-local model field → outbound multipart body | `Trail.toJson()`/`toFormData()` feed the server upload | Trail metadata |
| picked photo path → app-owned filesystem | `image_picker`/OS-share paths copied into app storage | Photo bytes/filenames |
| unsynced photo root → rest of app-docs | Sweep/delete walk a tree beside `library/`, `regions/`, `objectbox/` | Filesystem paths |
| device → Wanderer API | The drain replays authenticated `PUT`/`POST` calls on the shared `Api` client | Session credentials, trail/waypoint payloads |
| signed-in account → local drain queue | Drain uploads rows under the current session | Account id, trail ownership |
| crash point → server state | Interruption between "server accepted" and "local row recorded" is an integrity hazard | Server-assigned ids |
| server response → local row identity | Server-returned id written back into an existing local row | Server ids |
| download path → local bookkeeping | `@Unique(onConflict: replace)` lets a download overwrite a locally-owned row | Local bookkeeping fields |
| menu action → durable local data / server | Delete branches between local-only and server `DELETE` | Trail identity, deletion intent |
| rendered chip → drain provider | Tapping a badge invokes an authenticated upload | Retry trigger |

---

## Threat Register

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-36-01-01 | Information Disclosure | `TrailEntity.owner` | mitigate | `String? owner;` no default (`trail_entity.dart:111`) — null never matches an `owner.equals()` filter | closed |
| T-36-01-02 | Tampering | `localIdDirSegment` | mitigate | `^local-\d+-\d+$` whitelist, throws `ArgumentError` (`local_id.dart:46-56`) | closed |
| T-36-01-03 | Information Disclosure | `Trail.localId` / `Trail.syncState` | mitigate | Both `@JsonKey(includeFromJson: false, includeToJson: false)` (`trail.dart:133,137-139`); additionally `toFormData` (`form_data_util.dart`) is hand-built field-by-field and never reads either field at all — double-enforced | closed |
| T-36-01-04 | Tampering | `@Unique(onConflict: replace)` on `TrailEntity.id`/`WaypointEntity.id` | mitigate | Confirmed on both entities (`trail_entity.dart:20`, `waypoint_entity.dart:14`); `mintLocalId()` used, never `''` (`waypoint_entity.dart:59`, `local_trail_store.dart:164` via `saveNewLocalTrail`) | closed |
| T-36-01-05 | Denial of Service | ObjectBox schema change | accept | See Accepted Risks Log — CR-01's four new columns (`categoryRecordId`, `subcategoryRecordId`, `completed`, `tagsJson`) are exactly the additive nullable/defaulted shape the acceptance assumed; `build_runner` regenerated cleanly (36-REVIEW-FIX.md) | closed |
| T-36-02-01 | Tampering | `unsyncedTrailPhotoDir`/`unsyncedWaypointPhotoDir` | mitigate | Both route every id through `localIdDirSegment` before any `dart:io` call (`local_photo_store_util.dart:37-56`) | closed |
| T-36-02-02 | Tampering | destination filename in `reconcileLocalPhotos` | mitigate | `p.basename(source)` joined via `p.join(dir, destName)` (`local_photo_store_util.dart:116-123`) | closed |
| T-36-02-03 | Denial of Service | `sweepOrphanedUnsyncedPhotos` | mitigate | Enumerates only immediate children of `<app-docs>/unsynced`, per-directory failures swallowed (`local_photo_store_util.dart:203-227`) | closed |
| T-36-02-04 | Denial of Service | disk exhaustion from leaked copies | mitigate | Real function, wired at startup: `main.dart:120-123` calls `sweepOrphanedUnsyncedPhotos` unawaited on launch | closed |
| T-36-02-05 | Information Disclosure | copies surviving an account switch | accept | See Accepted Risks Log — `account_data_purge_util.dart` confirms `TrailEntity`/`WaypointEntity` are deliberately never purged; scoping (not deletion) is the documented invariant | closed |
| T-36-03-01 | Information Disclosure | `readOwnLocalTrails`/`countUnsyncedTrails`/`selectDrainCandidates` | mitigate | All three filter `TrailEntity_.owner.equals(accountId)`, `accountId` a required param (`local_trail_store.dart:360-408,440-457`) | closed |
| T-36-03-02 | Elevation of Privilege | drain candidate selection | mitigate | `accountId` required param, sourced fresh in `_drainPass` via `currentAccountId(store)` (`trail_sync_provider.dart:103,106-110`) | closed |
| T-36-03-03 | Tampering | duplicate server trail on retry | mitigate | `writeServerTrailId` standalone transaction (`local_trail_store.dart:480-499`); WR-01 fix moved the call in `trail_sync_provider.dart:215-228` to read the raw response id/photos and persist BEFORE `Trail.fromJson` (line 230), so a parse failure can no longer skip the write | closed |
| T-36-03-04 | Tampering | `TrailDownloadService` unique-replace put | mitigate | Six bookkeeping fields (`owner`, `localId`, `syncState`, `syncAttempts`, `syncNextAttemptAt`, `localPhotos`) re-read inside the same `runInTransaction` and carried forward (`trail_download_service.dart:187-218`); path construction additionally hardened post-review (WR-14) via `recordIdDirSegment`/`fileNameSegment` (`trail_download_service.dart:63,89,288`) | closed |
| T-36-03-05 | Denial of Service | one corrupt cached row blanking the whole list | mitigate | Per-entity `try/catch` around `toModel()` in `readLocalTrail` (`local_trail_store.dart:332-340`) and `readOwnLocalTrails` (`:380-387`) | closed |
| T-36-03-06 | Repudiation | `unsyncedLocalIds` returning cross-account ids | accept | See Accepted Risks Log — grep confirms its ONLY consumer is `main.dart:122`'s startup photo sweep; no other call site in `lib/` or `test/` | closed |
| T-36-04-01 | Elevation of Privilege | `drainIfOnline` candidate selection | mitigate | `currentAccountId(store)` re-read at the top of every `_drainPass` (`trail_sync_provider.dart:100-110`), never a notifier field | closed |
| T-36-04-02 | Tampering | duplicate server trail on retry | mitigate | `writeServerTrailId` commits before any waypoint step; create step guarded by `if (isLocalId(entity.id))` (`trail_sync_provider.dart:178-232`) | closed |
| T-36-04-03 | Denial of Service | unbounded retry burning battery/data | mitigate | `kMaxSyncAttempts = 4` (`sync_backoff.dart:18`), clamped `syncBackoffDelay` curve (30s/2min/10min, `:28-32`); `isDrainDue` excludes `failed` rows (`local_trail_store.dart:84-92`) | closed |
| T-36-04-04 | Denial of Service | overlapping drains desyncing bookkeeping | mitigate | Per-trail `state.contains(localId)` guard (`trail_sync_provider.dart:135`) plus whole-drain `_draining` flag (`:45,73-88`), both released in `finally` (`:85-88,316-318`) | closed |
| T-36-04-05 | Information Disclosure | in-flight bookkeeping surviving an account switch | mitigate | `trailSyncProvider` absent from `accountScopedProviders` list (`account_scope_invalidation.dart:71-84`), exclusion doc-commented with rationale (`:17-29`) | closed |
| T-36-04-06 | Tampering | startup sweep deleting live photos | mitigate | `keepLocalIds` built from `unsyncedLocalIds(store)`, whose query (`dbSyncState.notEquals(synced.index)`) includes `uploading` rows (`local_trail_store.dart:417-433`); sweep confined to `<app-docs>/unsynced` (`local_photo_store_util.dart:203-227`) | closed |
| T-36-04-07 | Spoofing/Session | credentials at deferred-upload time | accept | See Accepted Risks Log — drain calls `ref.read(apiProvider)` (`trail_sync_provider.dart:189`), the same shared authenticated client used elsewhere (`trail_save_provider.dart:62`); no new auth surface; 401 not special-cased, generic `catch` in `_drainOne` (`:304-315`) consumes an attempt | closed |
| T-36-05-01 | Information Disclosure | unsynced count in the dialog | mitigate | `countUnsyncedTrails` filters `owner.equals(currentAccountId(store))` re-read at point of use (`unsynced_signout_guard.dart:48-52`) | closed |
| T-36-05-02 | Repudiation | silent data-loss perception on sign-out | mitigate | `signout_unsynced_warning` ARB string explicitly states "Signing out won't delete it/them" (`app_en.arb:273`) | closed |
| T-36-05-03 | Tampering | a future sign-out call site bypassing the guard | mitigate | `settings_screen.dart:80-83` calls `confirmSignOutWithUnsyncedTrails` before `logout()`; source-gate test exists (`test/routes/settings_screen_signout_gate_test.dart`); the one exemption (`settings_account_screen.dart:114-121`, account deletion) is doc-commented as deliberate | closed |
| T-36-05-04 | Denial of Service | blocking sign-out on a pending upload | accept | See Accepted Risks Log — `confirmSignOutWithUnsyncedTrails` never returns `false` except on explicit cancel (`unsynced_signout_guard.dart:44-76`); `account_data_purge_util.dart` confirms trails are never purged | closed |
| T-36-06-01 | Tampering | photo copy destination | mitigate | `_copyPhotosForLocalSave` sources every dir from `unsyncedTrailPhotoDir`/`unsyncedWaypointPhotoDir` (`trail_create_screen.dart:667-701`), never constructs a path itself | closed |
| T-36-06-02 | Elevation of Privilege | owner assigned at save time | mitigate | `accountId = currentAccountId(store)` read fresh inside `_onSave`'s `createLocal` branch (`trail_create_screen.dart:457`), not cached at screen construction | closed |
| T-36-06-03 | Tampering | duplicate server trail from a re-saved draft | mitigate | `resolveLocalSaveMode` routes non-null `localId` to `updateLocal`, never `createLocal` (`local_trail_store.dart:65-73`) | closed |
| T-36-06-04 | Information Disclosure | device-local fields reaching the server | mitigate | Same double-enforcement as T-36-01-03: `@JsonKey(includeToJson:false)` plus `toFormData`'s hand-built field list never references `localId`/`syncState` | closed |
| T-36-06-05 | Denial of Service | disk pressure during photo copy | mitigate | `reconcileLocalPhotos` drops the failing photo into `failedCount`, never aborts (`local_photo_store_util.dart:110-129`) | closed |
| T-36-06-06 | Repudiation | silent photo loss | mitigate | `_finishLocalSave` shows `photo_copy_failed_toast(failedPhotoCount)` when `failedPhotoCount > 0` (`trail_create_screen.dart:725-735`) | closed |
| T-36-07-01 | Information Disclosure | local half of `build(handle)` | mitigate | `_isOwnHandle` gates the local read; `readOwnLocalTrails` additionally filters fresh `accountId` (`profile_trails_provider.dart:74-96`) | closed |
| T-36-07-02 | Information Disclosure | another hiker's profile page | mitigate | Non-own handle takes `const <Trail>[]` unconditionally, both in `build()` (`:94`) and `_readOwnLocal` (`:170-172`) | closed |
| T-36-07-03 | Tampering | duplicate rendering after upload | mitigate | `mergeOwnTrails` drops network hits whose id matches a local row's non-empty id (`own_trails_merge.dart:36-45`) | closed |
| T-36-07-04 | Denial of Service | one corrupt cached row blanking the list | mitigate | `readOwnLocalTrails` per-entity guard (T-36-03-05) plus `_fetchAndMerge`'s narrowed `on DioException` catch — WR-09 fix — never rethrows for the own handle (`profile_trails_provider.dart:214-227`) | closed |
| T-36-07-05 | Repudiation | silently showing a partial list | mitigate | `state.offline && state.isOwnHandle` drives a persistent banner (`profile_trail_screen.dart:94-124`), not a toast | closed |
| T-36-08-01 | Tampering | `_deleteTrail` branch ordering | mitigate | Unsynced branch (`trail_dropdown.dart:253-287`) precedes and returns before the `isLocal` branch (`:304-308`) | closed |
| T-36-08-02 | Tampering | delete during an in-flight upload | mitigate | Menu `enabled: !isDraining` (`trail_dropdown.dart:59-61,161`) AND `TrailSync.deleteUnsynced` independently refuses via `state.contains(localId)` (`trail_sync_provider.dart:361-362`) — defense in depth against the UI-only race | closed |
| T-36-08-03 | Denial of Service | download action on an unsynced trail | mitigate | Download item hidden under `if (!isUnsynced)` (`trail_dropdown.dart:122`), never merely disabled | closed |
| T-36-08-04 | Repudiation | misleading delete confirmation | mitigate | `delete_unsynced_trail_confirm` is its own l10n key stating "this can't be undone" (`app_en.arb:70`), used only for the unsynced branch (`trail_dropdown.dart:213-215`) | closed |
| T-36-08-05 | Elevation of Privilege | retry tap uploading another account's trail | mitigate | Chip's `retry(localId)` (`sync_status_chip.dart:58-60`) funnels into `drainIfOnline` → `_drainPass`, which re-reads `currentAccountId` (T-36-04-01) | closed |

*Status: open · closed*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-36-01 | T-36-01-05 | ObjectBox schema change is additive-only (nullable/defaulted columns) on a pre-production app; no migration path needed. Re-verified against the shipped code: CR-01's four new `TrailEntity` columns match this exact shape, and `build_runner` regenerated with a clean tree (36-REVIEW-FIX.md). | Phase 36 planner, re-confirmed by this audit | 2026-08-02 |
| AR-36-02 | T-36-02-05 | Unsynced photo copies live under a `localId`-named directory whose owning row carries an `owner`; another signed-in account never sees the row, so bytes are unreachable through the app even though they remain on disk. `account_data_purge_util.dart` documents scoping (not deletion) as this app's chosen invariant — re-confirmed: `TrailEntity`/`WaypointEntity` are explicitly excluded from both `purgeAccountScopedBoxes` and `accountScopedDirNames`. | Phase 36 planner, re-confirmed by this audit | 2026-08-02 |
| AR-36-03 | T-36-03-06 | `unsyncedLocalIds` is deliberately not account-scoped because its only consumer is the startup photo-orphan sweep, which must not delete a signed-out account's still-pending photos. Re-confirmed by grep: the only call site in `lib/` is `main.dart:122`; it returns opaque local ids only, never trail content, and is never used for display or upload. | Phase 36 planner, re-confirmed by this audit | 2026-08-02 |
| AR-36-04 | T-36-04-07 | The drain reuses the existing authenticated `Api` client and `PersistCookieJar`; no new auth or session surface is introduced. A 401 during token refresh is deliberately not special-cased and simply consumes one of `kMaxSyncAttempts`. Re-confirmed: `trail_sync_provider.dart` calls `ref.read(apiProvider)` (the same provider used by every other authenticated write path, e.g. `trail_save_provider.dart`), and `_drainOne`'s single generic `catch` treats every failure identically. | Phase 36 planner, re-confirmed by this audit | 2026-08-02 |
| AR-36-05 | T-36-05-04 | Sign-out is deliberately never blocked by the unsynced-trail warning — D-12 makes it a warning, not a hard stop; trails survive logout by design. Re-confirmed: `confirmSignOutWithUnsyncedTrails` returns `true` (proceed) in every path except an explicit user cancel of the dialog, and `account_data_purge_util.dart` never purges `TrailEntity`/`WaypointEntity`. | Phase 36 planner, re-confirmed by this audit | 2026-08-02 |

*Accepted risks do not resurface in future audit runs.*

---

## Non-Blocking Observations (out of threat-model scope)

These were noted while reading the cited files but are not declared threats in any PLAN's `<threat_model>` block, so they are recorded here for visibility only — not scored against `threats_open`, and not a basis for BLOCKER status per this audit's scope constraint ("verify against the threat register, do not scan for new threats beyond it"):

- **WR-15 (partial, from 36-REVIEW-FIX.md):** `signout_unsynced_warning` and `delete_unsynced_trail_confirm` — the two strings load-bearing for T-36-05-02 and T-36-08-04 — exist only in `app_en.arb`; 12 non-English locales fall back to raw English. The English-fallback behavior does not weaken either mitigation's logic (the correct string is always shown, just not always translated), so both threats remain CLOSED for this ASVS-1 audit. Flagged for translator follow-up per `36-REVIEW-FIX.md`.
- **Live-server/live-device verification gap (from 36-VERIFICATION.md):** SYNC-04/SYNC-05's duplicate-prevention chain (T-36-03-03, T-36-04-02, T-36-04-06) is enforced by call shape and confirmed by direct code reading, but `libobjectbox.dylib` cannot load under `flutter test` in this repo, so no automated test exercises a live ObjectBox `Store` or a real PocketBase server end-to-end. This audit's CLOSED determinations are code-level, consistent with `36-VERIFICATION.md`'s own `human_needed` status; a device+server pass remains outstanding per that report's `human_verification` list.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-02 | 43 | 43 | 0 | Claude (gsd-security-auditor) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-02
