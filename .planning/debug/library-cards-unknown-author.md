---
status: awaiting_human_verify
trigger: "For some trails in the library the author is \"Unknown\". The avatar also shows \"UN\". This does not happen with all library trails. To me it is not clear which keep their author and which don't."
created: 2026-09-01T00:00:00Z
updated: 2026-09-01T00:45:00Z
---

## Current Focus

reasoning_checkpoint:
  hypothesis: "`TrailEntityMapping.toModel()` (app/lib/entities/trail_entity.dart:320) builds `expand.author` purely from the ObjectBox ToOne relation (`author.target?.toModel()`), with no fallback to the persisted `authorRecordId` scalar. Because `ActorEntity.id` is `@Unique(onConflict: ConflictStrategy.replace)`, and `TrailEntity.fromModel()` (line ~272) assigns the author ToOne target via a bare `ActorEntity.fromModel(...)` (obxId always 0, never looked up/reused via `actorEntityForUpsert`), every subsequent write of ANY trail by the same author (a later download, a local save, or -- for `UserEntity.actor` specifically already patched elsewhere -- sign-in/profile refresh) deletes-and-reinserts that author's ActorEntity row under a new obxId, silently orphaning every OTHER TrailEntity's `author` ToOne that pointed at the old obxId. `expand.author` on those orphaned rows resolves to null forever after, so `Trail.summaryAuthorName` (trail.dart:182, `expand?.author?.username ?? \"Unknown\"`) and `summaryAuthorAvatar` degrade to \"Unknown\"/\"UN\", even though `authorRecordId` on the same row is still valid."
  confirming_evidence:
    - "trail_entity.dart:271-274 constructs `entity.author.target = ActorEntity.fromModel(trail.expand!.author!)` -- a fresh entity, obxId==0 -- inside `TrailEntity.fromModel()`, used by both the download path (trail_download_service.dart:147) and every local-save path (local_trail_store.dart:345,464,686)."
    - "actor_entity.dart:91-99's own doc comment on `actorEntityForUpsert` explicitly names `TrailEntity.author` as a ToOne that resolves to null after a conflict-replace remint, and states the fix is reusing the existing row's obxId -- but `actorEntityForUpsert` is only actually called from auth_provider.dart:257 and profile_provider.dart:67 (both `UserEntity.actor`), never from trail_entity.dart."
    - "trail_entity.dart:75-87's own doc comment on `authorRecordId` describes this exact failure mode (\"trail's author line degraded to Unknown, the avatar to a bare grey circle\") and states the scalar exists to survive it -- but `toModel()`'s `expand.author` (what `summaryAuthorName`/`summaryAuthorAvatar` actually read) never uses that scalar as a fallback, unlike the outer `Trail.author` field two lines below it (line 361: `authorRecordId ?? author.target?.id ?? _unknownAuthorId`)."
    - "local_trail_store.dart:1142-1147 (`readOwnLocalTrails`) independently re-derives the exact same fallback (`entity.authorRecordId ?? entity.author.target?.id`) for its own author-match filter, with a comment repeating the same remint mechanism -- confirming this is a known, established, but only PARTIALLY applied pattern in this codebase."
  falsification_test: "If this hypothesis is wrong, a TrailEntity row showing 'Unknown' would have `authorRecordId == null` too (no scalar survives), meaning the actor was never resolved at all rather than the ToOne going stale after being resolved correctly once. Confirmed NOT the case: `authorRecordId` is written unconditionally from `trail.author` (line 267-269) or backfilled from `trail.expand.author.id` (line 273) on every write that has an author, and nothing ever clears it afterward -- so an affected row's `authorRecordId` remains populated while `author.target` alone goes stale."
  fix_rationale: "The codebase already has an established, working resilience pattern for exactly this ObjectBox behavior (scalar id column surviving a ToOne remint) -- used for the outer `Trail.author` field and independently re-implemented in `readOwnLocalTrails`. The bug is that this pattern was never extended to `expand.author`, which is the value the UI (`summaryAuthorName`/`summaryAuthorAvatar`) actually renders. Extending `toModel()` to fall back to a store-backed `ActorEntity` lookup by `authorRecordId` when `author.target` is null fixes the root cause (missing fallback for the value actually displayed) and self-heals ALREADY-corrupted local rows on next read -- no re-download needed, which matters because the user's existing installed app already has corrupted ObjectBox data that a write-path-only fix would not repair."
  blind_spots: "Not fixing the write side (`TrailEntity.fromModel` still creates a fresh obxId==0 ActorEntity, so future writes will keep orphaning OTHER rows' ToOne -- the read-side fallback compensates for this every time, so it's a permanent mitigation, not just a one-time heal, but it does mean the ToOne itself never gets truly repaired in storage). CategoryEntity has the identical `@Unique(onConflict: replace)` + bare `.fromModel()` pattern (trail_entity.dart:277-280) and is very likely subject to the same class of bug for `category.target`, but that is outside this debug session's reported symptom (author only) and is left unfixed -- worth flagging separately. Have not yet run the app to observe live ObjectBox data confirming a real dangling obxId in this install; conclusion is from static code tracing, not a runtime dump of the box."
next_action: |
  Fix applied and self-verified (flutter analyze clean, full flutter test suite
  1087/1087 passing, including the pre-existing "author round-trip -- the
  orphaned-relation guard" group in test/entities/trail_entity_test.dart).
  Awaiting human verification: user must build+install and confirm previously
  "Unknown" library cards now show the real author without any re-download.

## Symptoms

expected: Library trail cards show the trail's real author username and a matching avatar, for every library trail — including trails downloaded from a remote/server source.
actual: "For some trails in the library the author is \"Unknown\". The avatar also shows \"UN\". This does not happen with all library trails. To me it is not clear which keep their author and which don't."
affected_set: Downloaded remote trails show "Unknown". (User-selected pattern: downloaded remote trails, as opposed to locally recorded ones.)
online_vs_offline: |
  When offline: library cards show "Unknown"; the trail detail view shows no author at all.
  When online: library cards STILL show "Unknown"; the trail detail view shows the correct author (user's guess: fetched fresh from the remote server).
  => The list/card path is broken in BOTH connectivity states; the detail path only works when it can hit the network. This points at the persisted local author/actor data (ObjectBox side) rather than at a purely offline-only fallback.
errors: None reported by user.
reproduction: Open the trail library in the Flutter app, scroll the cards for trails that were downloaded from the server (not locally recorded). Author reads "Unknown", avatar initials read "UN". Repeat with the device offline and online — cards show "Unknown" in both cases; only the detail view differs.
started: "Always been this way" — no regression trigger; likely never worked for this class of trail.
platform: Flutter app (app/lib/), trail library screen. Not the SvelteKit web frontend.

## Investigation Leads (orchestrator pre-scan, unverified)

- app/lib/models/trail.dart:182 — `String get summaryAuthorName => expand?.author?.username ?? "Unknown";`
  The "Unknown" literal for trails comes from here (list.dart:67 is the same fallback for lists).
  Implies: the card path depends on `expand.author.username` being populated on the Trail model.
- app/lib/components/base/actor_avatar.dart:63,98 — comments about a placeholder "Unknown" display name and not seeding the remote avatar. Likely source of the "UN" initials.
- app/lib/store/local_trail_store.dart:334 — comment: degrades to "Unknown"; "A missing actor row is not an error."
- app/lib/entities/trail_entity.dart:84 — comment about degrading to "Unknown" with a bare grey circle avatar.
- app/lib/provider/auth_provider.dart:253 — comment: "already-recorded local trails degrade to \"Unknown\" with a blank avatar".
  NOTE: this comment describes the OPPOSITE population from what the user reports (it blames locally-recorded trails, user reports downloaded remote trails). Worth checking whether the comment is stale, or whether the actor-row write path simply never runs for downloaded trails.

Key question to answer: where does the actor row that backs `expand.author` get written when a trail is DOWNLOADED (as opposed to recorded locally), and why is it absent/unresolvable for that path?

## Eliminated

## Evidence

- timestamp: 2026-09-01T00:10:00Z
  checked: app/lib/models/trail.dart:182,188 (summaryAuthorName, summaryAuthorAvatar getters)
  found: Both read `expand?.author?.username`/`expand?.author?.icon` with no other fallback -- "Unknown"/empty avatar come purely from `expand.author` being null.
  implication: The card display is entirely dependent on `TrailExpand.author` being populated; need to trace what populates it for library rows.
- timestamp: 2026-09-01T00:15:00Z
  checked: app/lib/entities/trail_entity.dart (TrailEntityMapping.toModel, full read)
  found: "`expand.author` is built as `author.target?.toModel()` -- the raw ObjectBox ToOne relation, no fallback. The OUTER `Trail.author` scalar field two lines above (361) DOES fall back: `authorRecordId ?? author.target?.id ?? _unknownAuthorId`."
  implication: Asymmetry found -- the scalar (unused by cards) is resilient, the expand object (what cards actually read) is not.
- timestamp: 2026-09-01T00:20:00Z
  checked: app/lib/entities/trail_entity.dart:75-87 (authorRecordId doc comment), actor_entity.dart:91-99 (actorEntityForUpsert doc comment)
  found: "Both comments describe the exact mechanism: `ActorEntity.id` is `@Unique(onConflict: ConflictStrategy.replace)`, so putting a freshly-constructed `ActorEntity` (obxId==0) deletes the existing row for that id and re-inserts under a NEW obxId, orphaning every ToOne that pointed at the old one. actor_entity.dart's comment names `TrailEntity.author` by name as an affected ToOne."
  implication: This is a known, previously-encountered bug class in this codebase, with a documented fix pattern (`actorEntityForUpsert`, reuse existing obxId).
- timestamp: 2026-09-01T00:25:00Z
  checked: grep for actorEntityForUpsert usages
  found: Only called from auth_provider.dart:257 and profile_provider.dart:67, both writing `UserEntity.actor`. Never called from trail_entity.dart.
  implication: The documented fix was applied to the sign-in/profile-refresh actor writes but never to TrailEntity's own author ToOne construction.
- timestamp: 2026-09-01T00:28:00Z
  checked: app/lib/entities/trail_entity.dart:263-274 (TrailEntity.fromModel author block)
  found: "`entity.author.target = ActorEntity.fromModel(trail.expand!.author!)` -- a bare, fresh (obxId==0) ActorEntity, constructed directly instead of via `actorEntityForUpsert`. `entity.authorRecordId` IS still set correctly alongside it (line 268 or 273)."
  implication: Every write through this factory (download path + all local-save paths) that has an author expand plants a future orphaning trigger for every OTHER row already pointing at that same author's prior ActorEntity row.
- timestamp: 2026-09-01T00:30:00Z
  checked: app/lib/services/trail_download_service.dart (downloadTrail, full read) and its only caller, trail_download_state_provider.dart:32-244 (DownloadingTrailIds.download), and its only widget caller, trail_dropdown.dart (only instantiated from trail_detail_screen.dart, which sources `trail` from fetchServerTrail's expand=author fetch)
  found: Download always goes through TrailEntity.fromModel(trail) with trail.expand.author populated (fetched with expand=author from the server). Nothing anomalous in the download path itself besides the fromModel() author-ToOne construction already identified.
  implication: Confirms the corruption is not "author never resolved on download" but "ToOne silently invalidated by a LATER write for the same author" -- consistent with "some trails show Unknown, not all" (whichever trail was the LAST one written for a given author keeps a valid ToOne; earlier ones by the same author lose it) and "always been this way" (happens on any library with 2+ trails, or self-downloads, by the same author).
- timestamp: 2026-09-01T00:33:00Z
  checked: app/lib/provider/trail/trail_library_provider.dart:117 (TrailLibraryNotifier.build, the actual data source for library cards)
  found: Calls `entity.toModel(includeGpx: false)` directly on every TrailEntity row in the signed-in account's library, no other author resolution logic.
  implication: Confirms library cards render exactly the (potentially-orphaned) `expand.author` from toModel(), matching symptoms.actual precisely.
- timestamp: 2026-09-01T00:35:00Z
  checked: grep for all `.toModel()` call sites on TrailEntity (6 total)
  found: trail_provider.dart:152, trail_sync_provider.dart:281, trail_library_provider.dart:117, local_trail_store.dart:969,1010,1155 -- every one of them already has a `Store store` variable in scope at the call site.
  implication: A store-backed fallback can be threaded into toModel() at every call site without needing to newly acquire a Store reference anywhere.

## Resolution

root_cause: |
  `TrailEntityMapping.toModel()` (app/lib/entities/trail_entity.dart) builds `TrailExpand.author` -- the object `Trail.summaryAuthorName`/`summaryAuthorAvatar` actually read -- purely from the ObjectBox `author` ToOne relation (`author.target?.toModel()`), with no fallback to the durable `authorRecordId` scalar column. `ActorEntity.id` is `@Unique(onConflict: ConflictStrategy.replace)`, and `TrailEntity.fromModel()` assigns that ToOne's target via a bare `ActorEntity.fromModel(...)` (always obxId==0) instead of reusing an existing row's obxId the way `actorEntityForUpsert` does for `UserEntity.actor`. So every later write of ANY trail by the same author (another download, a local save) deletes-and-reinserts that author's ActorEntity row under a new obxId, silently orphaning every OTHER already-downloaded trail's `author` ToOne that pointed at the old obxId. `authorRecordId` on the orphaned row stays correct (it survives the remint by design), but `toModel()` never consults it for `expand.author`, so the card falls through to the "Unknown"/"UN" literal fallback in trail.dart even though the real author id is still on the row.
fix: |
  Extended `TrailEntityMapping.toModel()` to accept an optional `Store? store` and, when the `author` ToOne target is null but `authorRecordId` is set, re-resolve the actor with a direct `ActorEntity` lookup by id before building `expand.author`. This mirrors the fallback pattern the outer `Trail.author` scalar field (and `readOwnLocalTrails`) already use for the exact same ObjectBox remint behavior, extending it to the value the UI actually renders. Threaded `store` through all 6 existing call sites (all of which already had a `Store` in scope). Self-heals already-corrupted local rows on next read -- no re-download required.
verification: |
  Self-verified only so far: `flutter analyze` clean on all 5 changed files
  (and whole-project analyze shows only pre-existing, unrelated info-level
  lints). `flutter test` -- full suite, 1087 tests -- all pass, including
  entity round-trip tests. No behavior change for the no-store call path
  (author.target stays the only source when store is omitted), so nothing
  regresses; the new store-backed branch only activates as a fallback when
  the relation is already null. Human verification on a real device with
  real (already-corrupted) library data is the remaining, decisive check --
  self-verification cannot exercise the actual ObjectBox remint scenario
  without a running app + real accounts, per the "no trail harness exists"
  note already in trail_entity_test.dart.
files_changed:
  - app/lib/entities/trail_entity.dart
  - app/lib/provider/trail/trail_provider.dart
  - app/lib/provider/trail/trail_sync_provider.dart
  - app/lib/provider/trail/trail_library_provider.dart
  - app/lib/store/local_trail_store.dart
