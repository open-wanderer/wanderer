---
status: diagnosed
phase: 36-local-first-recording-automatic-upload
source: [36-VERIFICATION.md]
started: 2026-08-02T15:48:59Z
updated: 2026-08-03T08:12:21Z
---

## Current Test

[testing paused -- 1 item outstanding (Test 3 blocked)]

## Tests

### 1. Offline save and re-save of a captured trail
steps: End a recording (or import a GPX) in airplane mode, fill in title and pick two photos, tap Save; re-open the trail from the own-trails list, change the title, save again.
expected: First save succeeds with the success toast and no offline-error. The trail appears once in the own-trails list, badged as not-yet-uploaded. The re-save updates the same trail (still one entry, not two).
why_human: Requires driving ImagePicker, ObjectBox and the full widget tree on a real device; PLAN 36-06 deferred this to end-of-phase human-check (no live Store in `flutter test`).
result: pass

### 2. Offline own-trails list
steps: In airplane mode, open your own profile's own-trails list (`/profile/<handle>/trails`).
expected: The list renders (does not error or spin forever), shows the offline banner text plainly stating it is showing only what's on-device, includes every not-yet-uploaded trail plus authored trails you've downloaded. Tapping an unsynced trail opens the offline-capable edit screen with its title/photos populated. With nothing saved, the empty state shows the cloud-up icon and "Nothing saved yet" copy.
why_human: Needs a real connectivity transition and a populated ObjectBox store; PLAN 36-07 deferred this to end-of-phase human-check.
result: pass
notes: "Core expectation met, but user reported 3 defects — see Gaps."


### 3. Trail dropdown gating for unsynced trails
steps: Open the trail dropdown menu on an unsynced trail: check Download is absent (not just disabled) and Delete shows a "cannot be undone" confirmation. Start (or wait for) that trail's upload and reopen the menu mid-upload. Then check the same menu on an ordinary downloaded trail.
expected: Unsynced trail — no Download entry; Delete confirmation states the deletion is unrecoverable. Mid-upload — Delete is greyed out / disabled. Downloaded trail — menu unchanged from today, Delete still only removes the local download.
why_human: Requires a live drain in progress and real menu interaction; PLAN 36-08 deferred this to end-of-phase human-check.
result: blocked
blocked_by: other
reason: "Not testable as trail detail screen can never be reached for a not synced trail. Goes directly to edit screen" (blocked by gap: unsynced trail tap routes to edit instead of detail)

### 4. Automatic upload and interrupted-upload resume (no duplicates)
steps: With the app foregrounded and a working connection (or by regaining connectivity while the app stays open), watch an unsynced trail upload without tapping anything. Separately, force-kill the app (or otherwise interrupt) partway through an upload — e.g. after the trail record is created but before all waypoints/photos finish — then relaunch/reconnect and let the drain resume.
expected: The trail's badge transitions Pending → Uploading → (badge disappears) with no user action beyond having the app open and online. After an interrupted-and-resumed upload, exactly one trail (and one row per waypoint) exists on the server — no duplicates — and the local row shows no badge (synced) with its photos intact.
why_human: SYNC-01/SYNC-04/SYNC-05's duplicate-prevention chain is verified by code inspection and unit tests of the pure decision logic, but no automated test in this repo exercises a live PocketBase server or a real ObjectBox `Store` (confirmed untestable in `flutter test` — `libobjectbox.dylib` fails to load). An end-to-end device+server pass is the only way to confirm no duplicate is produced under a genuine mid-drain interruption.
result: pass

### 5. Delete a trail after it has synced (orphan check)
steps: Create a trail while offline; go online and let it upload successfully; delete the trail; observe the own-trails list and tap the remaining entry.
expected: Deleting a synced-from-local trail removes it server side AND removes the local row, so it disappears from the own-trails list.
result: issue
reported: "1. create trail while offline 2. go online 3. trail is uploaded correctly 4. delete trail 5. trail is now deleted server side 6. an orphaned trail now permanently exists in the 'own trail' list. Clicking on it leads to an idefinite loading spinner."
severity: blocker
source: reported by user after the initial 4-test pass; not derived from a SUMMARY

## Summary

total: 5
passed: 3
issues: 5
pending: 0
skipped: 0
blocked: 1

## Gaps

- truth: "Own-trails list renders stably without spurious loading states"
  status: failed
  reason: "User reported: The own trails list shows a spinner every so often"
  severity: minor
  test: 2
  root_cause: "ProfileTrailsNotifier.build watches trailFilterProvider, which throws on every offline API failure and is therefore auto-retried 10x by Riverpod 3 defaultRetry (200ms doubling to 6400ms cap, ~20 emissions over ~45s). ref.watch invalidates asReload:true, producing a genuine AsyncLoading; profile_trail_screen uses .when() with the default skipLoadingOnReload:false, so each retry replaces the whole list with a full-screen spinner. The drain's ref.invalidate calls are NOT the cause (asReload:false -> seamless refresh -> skipLoadingOnRefresh defaults true)."
  artifacts:
    - path: "app/lib/provider/trail/trail_filter_provider.dart:60"
      issue: "Rethrows offline failure as Exception, opting into Riverpod's 10-attempt retry loop; no offline fallback despite local-first goal"
    - path: "app/lib/provider/profile/profile_trails_provider.dart:71"
      issue: "Watches the retry-prone async filter provider, so its failures become list reloads; also :101 search() assigns a bare AsyncLoading() discarding previous value"
    - path: "app/lib/routes/profile_trail_screen.dart:90"
      issue: ".when() with default skipLoadingOnReload:false discards data already held; :173 renders a full-screen CircularProgressIndicator. Repo-wide grep found ZERO uses of skipLoadingOnReload/skipLoadingOnRefresh in app/lib - codebase-wide latent exposure"
  missing:
    - "Give trailFilterProvider an offline fallback (default filter) and/or an explicit retry policy so connectivity failures are not retried 10x"
    - "Decouple the list from filter loading churn - watch a selected slice (.select((a) => a.value)) so transitions that do not change the actual TrailFilter do not reload the list"
    - "Pass skipLoadingOnReload:true at profile_trail_screen.dart:90 (or render an inline indicator when isLoading && hasValue), and preserve the previous value in search()'s AsyncLoading"
    - "Note: the render-path fix alone hides the flicker but leaves ~20 redundant network fetches and spurious drainIfOnline kicks in place"
  debug_session: ".planning/debug/own-trails-spinner-flicker.md"

- truth: "Tapping an unsynced trail opens the trail detail screen (like any other trail), from which the user can choose to edit"
  status: failed
  reason: "User reported: Tapping on a not synced trail should open this trail like any other trail in the trail detail screen. From here the user can decide to edit it."
  severity: major
  test: 2
  root_cause: "_onTrailSelect at profile_trail_screen.dart:53-62 short-circuits every unsynced trail to /trail/create/edit. This was a deliberate PLAN 36-07 decision forced by D-06: TrailEntity.toModel() blanks local-sentinel ids (trail_entity.dart:292), so context.push('/trail/${trail.id}') emits '/trail/' which go_router normalizes to '/trail' - a path with no GoRoute. Deleting the branch alone yields a no-route error page, not the detail screen. trailProvider also cannot load a local trail: its ObjectBox fallback filters on savedByUserIds, which local captures never set (D-10 keeps ownership and library membership separate)."
  artifacts:
    - path: "app/lib/routes/profile_trail_screen.dart:53-62"
      issue: "The routing divert itself"
    - path: "app/lib/entities/trail_entity.dart:292"
      issue: "Blanks the id for local trails, making /trail/:id unaddressable (D-06)"
    - path: "app/lib/provider/trail/trail_provider.dart:17-98"
      issue: "Server-id-keyed; ObjectBox cache fallback gated on savedByUserIds which local captures never have"
    - path: "app/lib/routes/trail_detail_screen.dart:69,72,96,123-159"
      issue: "availableOffline/isDownloading compare empty ids; bottom bar renders a Download button whenever !availableOffline, violating D-17 the moment detail becomes reachable. LikeButton reads trailProvider(trail.id) and PUTs /trail-like"
    - path: "app/lib/components/trail/trail_panel.dart:300,324,332"
      issue: "Push /trail/${trail.id}/map -> '/trail//map', same no-route failure"
    - path: "app/test/components/trail/trail_dropdown_delete_gate_test.dart"
      issue: "Source-inspection test (greps source text of trail_dropdown.dart) so it passes green while the menu is unreachable in the running app - no automated signal caught this"
  missing:
    - "Make the detail screen addressable without a server id: add a sibling route (e.g. /trail/local/:localId before /trail/:id) or accept a nullable localId. Trail.localId (local-<micros>-<seq>) is the only stable handle"
    - "Back the local case with readLocalTrail(store, localId) (local_trail_store.dart:323-341) via a localTrailProvider family. Do NOT write savedByUserIds on local captures - D-10 forbids conflating ownership with library membership"
    - "Gate id-dependent chrome on isUnsyncedState(trail.syncState): hide bottom-bar Download (D-17), hide/disable LikeButton, hide or re-target the three /trail/:id/map pushes and the navigate push"
    - "Add an Edit affordance on the detail screen reusing the existing push (/trail/create/edit with the Trail as extra)"
    - "Add a widget test that pumps the own-trails list with an unsynced Trail and asserts the pushed location is the detail route - existing dropdown tests are source-greps and will not catch a regression"
    - "Only then delete the divert branch. This also unblocks UAT Test 3 (TrailDropdown is instantiated only at trail_detail_screen.dart:98, so all D-14/D-17 gating is currently unreachable)"
  debug_session: ".planning/debug/unsynced-trail-skips-detail-screen.md"

- truth: "Edits to an unsynced trail are reflected in the own-trails list without a manual reload"
  status: failed
  reason: "User reported: After saving edits on a non synced trail the 'own trail' list needs a manual reload before edits are shown"
  severity: major
  test: 2
  root_cause: "The local-first save path commits to ObjectBox but emits no invalidation signal at all. trail_create_screen.dart contains ZERO invalidate/refresh calls - its shared post-save tail _finishLocalSave (714-762) only drains, toasts and setStates its own trail field. ProfileTrailsNotifier.build() reads the local half as a one-shot readOwnLocalTrails query and the app has no ObjectBox Query.watch() streams anywhere, so a row mutation produces no reactive signal. The list screen stays mounted beneath the pushed edit route (maintainState:true), keeping the auto-dispose provider alive with its pre-edit state. The three existing invalidate(profileTrailsProvider(...)) call sites cover only successful drain upload, deleteUnsynced, and server-side delete - none fire offline."
  artifacts:
    - path: "app/lib/routes/trail_create_screen.dart:714-762"
      issue: "_finishLocalSave is the shared tail of createLocal (451-506) and updateLocal (508-569) and notifies no other consumer of the row it just wrote. The alreadySynced -> _saveViaNetwork branch (583-649) is equally silent"
    - path: "app/lib/provider/profile/profile_trails_provider.dart:85-96"
      issue: "build() local read is one-shot; watches only trailFilterProvider, objectBoxProvider and authProvider, none of which change on a row write"
    - path: "app/lib/routes/profile_trail_screen.dart:58"
      issue: "Pushes the edit route without await and without any invalidate, unlike trail_dropdown.dart:107-110 which compensates caller-side - invalidation was left to callers and implemented inconsistently"
  missing:
    - "Invalidate profileTrailsProvider('@<preferredUsername>') (and likely trailLibraryProvider, matching the pair trail_sync_provider already invalidates) inside _finishLocalSave right after the ObjectBox write commits, deriving the handle from a fresh authProvider read"
    - "Apply the same treatment to the _saveViaNetwork branch"
    - "Verify the family-key handle string used by the existing invalidations ('@${userEntity.preferredUsername}') matches the screen's widget.handle exactly - a mismatch would silently no-op every one of those invalidations"
  debug_session: ".planning/debug/unsynced-trail-edit-not-reflected-in-list.md"

- truth: "Deleting a trail that was created offline and has since synced removes both the server record and the local row, leaving no entry in the own-trails list"
  status: failed
  reason: "User reported: after an offline-created trail uploads successfully and is then deleted, it is removed server side but an orphaned trail permanently remains in the own-trails list; tapping it shows an indefinite loading spinner"
  severity: blocker
  test: 5
  root_cause: "markTrailSynced (local_trail_store.dart:579-600) deliberately KEEPS the local row after a successful drain (SYNC-05), leaving owner==accountId, id==<server id>, syncState==synced, savedByUserIds==[]. Deleting such a trail takes _deleteTrail's THIRD branch (trail_dropdown.dart:310-332) because the model came from GET /trail/<id>, so syncState is the `synced` default and isLocal the `false` default - skipping both the unsynced and un-download branches. That branch is a pure network DELETE plus four invalidates and touches no local storage. deleteLocalTrailRow has exactly ONE call site app-wide (trail_sync_provider.dart:365, inside deleteUnsynced) which that branch never reaches. readOwnLocalTrails filters only on owner/savedByUserIds with no syncState and no server-existence predicate, so the invalidate at trail_dropdown.dart:318 faithfully rebuilds the list and re-emits the orphan."
  artifacts:
    - path: "app/lib/components/trail/trail_dropdown.dart:310-332"
      issue: "Server-delete branch never removes the local row"
    - path: "app/lib/provider/trail/trail_save_provider.dart:202-204"
      issue: "Bare network DELETE; receives the full Trail (with localId) but discards it"
    - path: "app/lib/util/local_trail_store.dart:360-393"
      issue: "readOwnLocalTrails has no syncState or server-existence predicate"
    - path: "app/lib/util/local_trail_store.dart:298"
      issue: "deleteLocalTrailRow is localId-keyed and reachable only via deleteUnsynced; a network-loaded Trail has localId == null"
  missing:
    - "Remove the local row as part of the server-delete flow. Keying problem: deleteLocalTrailRow is localId-keyed but a network-loaded Trail has localId == null - needs an id-keyed sibling or a localId lookup by server id"
    - "Consider reusing TrailLibraryNotifier.deleteTrail(id), which already removes a row whose membership drains to empty (the orphan's exact shape) - but note it also deletes library/<id>/"
    - "Decide whether a synced local row should stay owner-scoped forever with no reconciliation sweep at all (none exists today; unsyncedLocalIds feeds only sweepOrphanedUnsyncedPhotos)"
    - "Sequence AFTER 36-10 (which rewrites trail_dropdown.dart:318 inside this very branch) and AFTER 36-13 (which modifies trail_dropdown.dart and whose gate test pins _deleteTrail branch order - appending local cleanup after deleteTrail(trail) is compatible; reordering branches is not)"
    - "Reuse 36-11's readOwnLocalTrail(store, localId:, accountId:) rather than duplicating the primitive"
  debug_session: ".planning/debug/orphaned-local-row-after-post-sync-delete.md"

- truth: "Opening a trail whose server record no longer exists fails fast with a clear error, not a chromeless spinner"
  status: failed
  reason: "Separate defect isolated during diagnosis of test 5: the 'indefinite' spinner is a ~38s Riverpod retry storm. A 404 is permanent, not transient, yet defaultRetry retries the DioException 10x (200/400/800/1600/3200/6400x5) with each pending retry rendering as a bare full-screen spinner with no AppBar and no cancel, before finally showing WandererError."
  severity: major
  test: 5
  root_cause: "TrailNotifier.build 404s and its ObjectBox fallback (trail_provider.dart:70-96) is gated on savedByUserIds.containsElement(userId), which a local capture never gains (D-10). The row IS in the box under that exact id but the membership clause excludes it, so line 96 rethrows into Riverpod 3's defaultRetry. trail_detail_screen.dart:196-199 renders the resulting AsyncLoading(retrying:true) as a chromeless, uncancellable spinner. Same class as gap 1's retry storm but in a different provider - 36-09 fixes trailFilterProvider only, so this survives all five existing plans."
  artifacts:
    - path: "app/lib/provider/trail/trail_provider.dart:70-96"
      issue: "ObjectBox fallback gated on savedByUserIds excludes the very row it needs; rethrows a permanent 404 into a 10-attempt retry loop"
    - path: "app/lib/routes/trail_detail_screen.dart:196-199"
      issue: "Chromeless, uncancellable loading state - no AppBar, no back affordance, no cancel"
  missing:
    - "Do not retry a permanent 404 - classify it as terminal (explicit retry policy or catch-and-surface) rather than letting defaultRetry run 10 attempts"
    - "Give the detail screen's loading state chrome (AppBar with back) so the user is never trapped"
    - "Note this fires for ANY trail deleted elsewhere (e.g. from the web UI), not just orphans"
  debug_session: ".planning/debug/orphaned-local-row-after-post-sync-delete.md"
