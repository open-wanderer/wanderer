---
phase: 36-local-first-recording-automatic-upload
fixed_at: 2026-08-02T00:00:00Z
review_path: .planning/phases/36-local-first-recording-automatic-upload/36-REVIEW.md
iteration: 1
findings_in_scope: 19
fixed: 18
partial: 1
skipped: 0
status: partial
---

# Phase 36: Code Review Fix Report

**Source review:** `.planning/phases/36-local-first-recording-automatic-upload/36-REVIEW.md`
**Iteration:** 1

**Summary:**

- Findings in scope (blocker + warning): 19
- Fixed: 18
- Partially fixed: 1 (WR-15 — engineering half done, translations deliberately not authored)
- Skipped: 0

**Verification:**

| Check | Before | After |
|---|---|---|
| `flutter test` | 771 passing, 1 skip | **814 passing, 1 skip, 0 failures** |
| `flutter analyze --no-pub` | 43 issues, 0 errors/warnings | **43 issues, 0 errors/warnings** |
| Codegen (`build_runner`) | — | regenerated and committed; tree clean |

No pre-existing analyzer error or warning existed before these changes, and none was introduced. All 43 remaining issues are pre-existing `info`-level lints in `icon_util.dart`, vendored `tiptap_bridge.dart`, and two dangling-library-doc-comment notices.

## A note on test strategy

The review observed that six of this phase's test files are source-level greps and that none of the four blockers would have been caught by any existing test. I confirmed *why* rather than assuming: I attempted to open an ObjectBox store under `flutter test` and it fails with `Failed to load dynamic library 'libobjectbox.dylib'`. There is no native library available to the Dart VM test runner, so every `Store`-touching function in `local_trail_store.dart` is genuinely untestable behaviourally in this harness — not merely untested.

Where a fix touched code reachable without a `Store`, I added real behavioural coverage (CR-01, CR-02, CR-04's decision input, WR-13, WR-14). Where it did not, the compensating move was to change the *shape* of the dangerous call so the bug becomes unwritable rather than merely un-rewritten — see CR-03 and WR-11.

Two of the new test groups were verified to actually **fail against the pre-fix implementation** before being kept (WR-13's three canonicalization cases; CR-01's round-trip assertions).

## Fixed — Blockers

### CR-01: Category, subcategory, tags and completed destroyed by every local-first save

**Commit:** `509c2bf9` (+ `efa40e33` formatting)
**Files:** `app/lib/entities/trail_entity.dart`, `app/lib/objectbox.g.dart`, `app/lib/objectbox-model.json`, `app/test/entities/trail_entity_test.dart`

Added four persisted columns and restored them in `toModel()`.

Two things worth flagging beyond the review's suggestion:

- The suggested field name `categoryId` **cannot be used** — ObjectBox already claims that column for the `ToOne<CategoryEntity> category` relation's foreign key (confirmed in `objectbox-model.json`: property `19` is `categoryId → CategoryEntity`). Named `categoryRecordId`/`subcategoryRecordId` instead, with the collision documented at the field.
- Tags are stored as **JSON** (`tagsJson`), not as the suggested list of names. Names alone would discard the server id of an already-known tag, and `resolveTags` only reuses a tag whose id is non-empty — so every upload would have re-`PUT /tag`'d every tag. Parallel id/name columns were rejected as drift-prone; one JSON column cannot drift. Precedent: `navCacheJson`, `CategoryEntity.translationsJson`.

`toModel()` also now falls back to `category.target?.id`, which fixes the same field for previously-downloaded rows.

**Behavioural coverage added:** 9 round-trip tests, including id-preserving tag round trip and a malformed-`tagsJson` degradation case.

**Codegen:** `objectbox.g.dart` and `objectbox-model.json` regenerated and committed (new properties 36–39).

---

### CR-02: A waypoint whose photo upload fails is created twice

**Commit:** `aee9aa64`
**Files:** `app/lib/provider/waypoint/waypoint_provider.dart`, `app/lib/provider/trail/trail_sync_provider.dart`, `app/test/provider/waypoint_provider_create_test.dart` (new)

Implemented as suggested: `WaypointPhotoUploadException` carries the created record out of `WaypointSave.create`, and the drain persists its id before rethrowing.

The other caller (`trail_save_provider.createTrail`) catches generically, so its behaviour is unchanged.

**Behavioural coverage added:** 4 tests using the `apiProvider` override pattern already established in `tag_provider_test.dart`, driving a real partial failure (PUT succeeds, photo POST times out) and asserting the created id is surfaced and `PUT /waypoint` ran exactly once.

---

### CR-03: A resumed drain wipes the trail's photos

**Commit:** `a3c48f2d`
**Files:** `app/lib/provider/trail/trail_sync_provider.dart`, `app/lib/util/local_trail_store.dart`

Took the review's **alternative** (persist `created.photos` inside `writeServerTrailId` at step 2) rather than its primary suggestion (a fresh `GET /trail/{id}` on the resume path). The alternative needs no extra request, and it commits the id and the photo list in one transaction — which matters, because a row holding the id but not the photos looks to a resumed drain exactly like a trail with no photos.

Additionally hardened the call shape: `markTrailSynced`'s `serverPhotoFilenames` is now **nullable and defaults to null** ("keep what the row holds"). The old `const []` default meant the safest-looking call — omitting the argument — was the one that erased the column. The drain now passes nothing at all.

**No behavioural test** — `markTrailSynced`/`writeServerTrailId` both require a live `Store` (see note above). The signature change is the compensating control: the wipe is no longer expressible by accident.

---

### CR-04: An edit made after upload never reaches the server

**Commit:** `3821fae8`
**Files:** `app/lib/routes/trail_create_screen.dart`, `app/lib/util/local_trail_store.dart`, `app/test/util/local_trail_store_test.dart`, `app/test/routes/trail_create_screen_local_save_gate_test.dart`

Implemented part 1 as suggested (route on the persisted row, not the screen snapshot).

Part 2 was implemented **differently from the suggestion**, because the suggested option of "reset the row to `pending`/`syncAttempts = 0` so the drain re-uploads the change" does not work: the drain's step 2 is guarded on `isLocalId(entity.id)`, so a re-queued synced row skips create, skips already-created waypoints, and goes straight to `markTrailSynced` — marking it synced again **without ever sending the edit**. It would have converted silent loss into silent loss plus churn.

Instead `updateLocalTrail` returns a `LocalUpdateOutcome` and refuses to write over a `synced` row; the caller delegates to the network path, extracted as `_saveViaNetwork` so both routes into it share one implementation. Offline that reports `error_saving_trail` — which is the point: the user is told the edit did not land instead of being shown a success toast over an edit that has nowhere to go.

**Also fixes WR-07** in the same statement (see below).

**Behavioural coverage added:** a group proving the stale snapshot and the persisted row genuinely *disagree* on the routing decision — which is the entire bug — plus a new gate asserting the one sanctioned network reach is guarded on `LocalUpdateOutcome.alreadySynced`.

**Incidental defect found and fixed:** the existing local-save gate anchored the end of `_onSave` at `bool get _hasUnsavedChanges`, which is several members later. Its slice silently included `_copyPhotosForLocalSave` and `_finishLocalSave`, so the gate was asserting over code that was never in `_onSave`. Re-anchored on the method's closing brace.

## Fixed — Warnings

| ID | Commit | Summary |
|---|---|---|
| WR-01 | `4b5f6060` | Trail id + photo list now read off the raw response map and committed **before** `Trail.fromJson`, so a parse failure can no longer cause a duplicate trail. `writeServerTrailId`'s photo param made nullable so an id-only body can't erase photos. |
| WR-02 | `a0899c6d` | Refused delete now surfaces a warning toast (`delete_blocked_while_uploading`, new key) and the pop happens only on success. L10n lookup hoisted above the `await` — `context` is a parameter, not `State.context`, so a post-await `mounted` check does not license reading from it (the analyzer flagged exactly this). |
| WR-03 | `2fc17f5f` | `drainIfOnline` remembers a request that arrives mid-pass and loops once more; pass body extracted as `_drainPass` so each iteration re-reads online status and account id fresh. |
| WR-04 | `08cb9088` | `entity.photos = existing.photos` added. Carry-forward gate extended to this second site — **plus a guard against the anchor bug the new assertions hit on first run**: `'\n}'` matches inside the parameter list, so the sliced body came out empty and all nine assertions passed vacuously. |
| WR-05 | `0704a919` | Doc comments unswapped. |
| WR-06 | `eb8d6557` | `_readOwnLocal` derives both the own-handle test and the actor id from a fresh `authProvider` read. |
| WR-07 | `3821fae8` | Fixed inside CR-04 — same statement, so splitting would have meant editing the line twice. The branch now uses `persisted?.localId ?? updatedTrail.localId ?? _localId!`, and `_finishLocalSave` takes the id as a parameter instead of re-reading `_localId!`, so the two can no longer diverge. |
| WR-08 | `58306e39` | Guard hoisted above the create and made loud (`StateError`); `_copyPhotosForLocalSave` counts a keyless waypoint's photos into `failedCount`. Also typed the loop's empty fallback — a bare `const []` made `wp` dynamic and silently widened the counter to `num` (caught by the analyzer). |
| WR-09 | `8bc14131` | Narrowed to `on DioException`, exactly as suggested — transport failures still degrade to the local half, parse errors and `TypeError`s surface. |
| WR-10 | `a0792753` | Both call sites routed through `_deletePhotoDirBestEffort`. **Found a second site the review did not list:** the drain's step 4 also calls it, inside the try, *after* `markTrailSynced` — so an `ArgumentError` there would have reached `recordDrainFailure` and flipped a completed upload back to `pending`. |
| WR-11 | `439118a9` | `markTrailUploading` added to `local_trail_store.dart`, re-querying inside `runInTransaction` and writing one field. |
| WR-12 | `ed5caa37` | User resolved before the in-flight set is joined and before the `try`; a missing row logs and returns without recording a failed attempt. |
| WR-13 | `86bf04b2` | Both sides canonicalized. **4 behavioural tests added; 3 of them fail against the pre-fix code** (verified by temporarily reverting the change). |
| WR-14 | `b3cbebb7` | `recordIdDirSegment` + `fileNameSegment` added to `local_id.dart`; all four sites in `trail_download_service.dart` now use `p.join` + a validated segment. **13 behavioural tests added.** |

Two WR-14 details worth calling out:

- I used a **looser** whitelist than the suggested `^[a-z0-9]{15}$` — `^[A-Za-z0-9_-]{1,64}$`. The security property needed is only "cannot escape the parent directory", and `[A-Za-z0-9_-]` admits no `.`, `/` or `\`, so every traversal spelling is still rejected. Pinning PocketBase's exact current id shape would turn any future id-format change, or a federated peer with a different convention, into silently broken downloads — a real regression risk for no additional safety.
- The photo-filename case is **worse than the review described**. I probed the actual behaviour: `Uri.parse` resolves `..` segments away, so a traversal URL's path collapses to `/`, and `p.basename('/')` is `'/'` — not `''`. `p.join(dir, '/')` returns `'/'`, discarding `dir` entirely, so the download would have targeted the filesystem root rather than escaping one level. Conversely `Uri.path` leaves `%2F` encoded, so `..%2Fescape.jpg` is a harmless literal filename. Both behaviours are now pinned by tests.

## Partially fixed

### WR-15: Nine new user-facing strings exist only in `app_en.arb`

**Commit:** `29d71ba0`
**Files:** `app/l10n.yaml`, `app/lib/i18n/untranslated_messages.json` (new)

**Done:** the review's second ask — "confirm `flutter gen-l10n`'s untranslated-message report is being read rather than suppressed". It was being *emitted* but only as a per-locale count on stdout that scrolls past unread, which is how this shipped unnoticed. `untranslated-messages-file` now writes an itemised report, committed so the gap appears in a diff when it grows. The report confirms and quantifies the finding: 18 untranslated messages in 12 locales, 11 in `de`.

**Not done, deliberately:** the review's first ask — adding `signout_unsynced_warning` and `delete_unsynced_trail_confirm` to every locale file.

**Reasoning:** authoring destructive-action copy for 13 locales is content work requiring native review, not a code fix. Both strings are plural-form ICU messages whose entire purpose is preventing a destructive misunderstanding; a subtly wrong "this can't be undone" in Hungarian or Basque is a worse outcome than visible English, and I have no way to validate the result. Per the fix-pass guidance, I am recording this rather than applying a speculative change.

**Recommended follow-up:** treat `lib/i18n/untranslated_messages.json` as the work list and route the two destructive-action strings through whoever owns translations. The English source strings and their `@`-metadata (placeholder descriptions) are already in place for them.

## Notes for the verifier

- **Migration:** `TrailEntity` gained four columns. ObjectBox adds them with default values on an existing store, so no data migration is needed — but rows captured by the *pre-fix* build still have `categoryRecordId`/`subcategoryRecordId`/`tagsJson` null and `completed` false. That is unrecoverable (the values were never persisted anywhere) and affects only unreleased local data.
- **Behaviour change worth a manual pass:** editing a trail immediately after its upload completes now goes down the network-update path. Online this is the correct, pre-existing path. **Offline it now shows `error_saving_trail` where it previously showed a false success toast** — intended, per CR-04, but it is a user-visible change and D-16 explicitly puts offline editing of a synced trail out of scope.
- **Not addressed (out of scope, no finding):** `toModel()`'s `localPhotos: localPhotos.isNotEmpty ? localPhotos : photos` fallback means a row that synced from a local capture ends up exposing *server filenames* as `localPhotos`. Combined with `isLocal: true`, card rendering for a just-synced trail may still be off. This is the D-10 fallback's pre-existing shape, not something these fixes introduced or that CR-03/WR-04 changed, so I left it alone — but it is adjacent enough to the photo findings that it may be worth its own look.

---

_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
