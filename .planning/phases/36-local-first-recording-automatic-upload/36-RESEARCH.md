# Phase 36: Local-First Recording & Automatic Upload - Research

**Researched:** 2026-08-02
**Domain:** Flutter/Dart mobile — ObjectBox local persistence, Riverpod state/background sync, offline-first UX
**Confidence:** HIGH (every claim below is grounded in a real file:line in this repo; no external library research was needed — this phase adds no new dependencies)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Photo durability**
- **D-01:** Local save **copies** each picked photo into app-owned storage; the picker's cache path is never trusted at drain time.
- **D-02:** Copies are deleted **on successful drain**; deleting an unsynced trail deletes its copies. A **startup orphan sweep** catches crash-between-steps leaks.
- **D-03:** If the copy itself fails: **save the trail, drop that photo, tell the user which and why.** Never fall back to the picker path.
- **D-04:** *(user directive)* **Both `Trail` and `Waypoint` must persist `localPhotos`.** `WaypointEntity` already has the field; `TrailEntity` does not and must gain it.

**Partial-failure semantics**
- **D-05:** **Resume from step.** Drain order is `PUT /tag` ×N → `PUT /trail/form` → `PUT /waypoint` ×N; a broken attempt resumes rather than restarts. Satisfies SYNC-04 by construction.
- **D-06:** Progress is **derived from data**: **empty id means not-yet-uploaded, everywhere.** Three of four producers already follow this. **The one known exception (photo-EXIF waypoints, `trail_create_screen.dart:194`) must be fixed to `id: ''` with list identity carried in a separate local key.**
- **D-07:** A failing drain **backs off over a few attempts, then parks as `Failed`** for manual retry (SYNC-03). No retryable-vs-permanent taxonomy.

**Sync state and how it shows**
- **D-08:** Visible states are **Pending → Uploading → Failed**. Synced shows nothing.
- **D-09:** The badge answers **only** "is this on the server yet" — never how it was captured.
- **D-10:** **Unsynced and downloaded are mutually exclusive by construction.** Unsynced trails must **NOT** be expressed via `savedByUserIds`.
- **D-11:** REC-06's offline list mixes unsynced + downloaded-authored-by-me as a **flat list** — no sectioning, no special sort.

**Account scoping and destructive actions**
- **D-12:** Sign-out with unsynced trails shows a **warning dialog naming the count**, but proceeds.
- **D-13:** Account B signing in over account A's unsynced trails sees **nothing** — filtered out of list and drain, never deleted.
- **D-14:** An unsynced trail **can** be deleted, behind an **unrecoverable** confirm; deletion is **blocked while a drain is in flight**.

**Drain triggers and offline editing**
- **D-15:** Drain fires on **app foreground AND connectivity regained while open**. No OS background execution this phase.
- **D-16:** REC-05's offline editing is **metadata only** (title, description, category, photos) — no route re-editing.
- **D-17:** *(user directive)* **An unsynced trail must not be downloadable** — the download action must be hidden/disabled for it.

### Claude's Discretion
- Concrete shape of the sync-state field and owner field on `TrailEntity` (enum vs int, indexed or not).
- Whether the drain uploads one trail at a time or several concurrently.
- Backoff curve and attempt count for D-07.
- Location/naming of the app-owned photo directory (follow `library/`/`avatars/`/`regions/` convention).
- Wording of the warning/confirm strings (must be l10n keys in `app_en.arb`).

### Deferred Ideas (OUT OF SCOPE)
- True OS-level background upload (registering a background task). D-15 scopes this phase to foreground + regained connectivity only.
- Queueing non-GPX imports for transcoding on reconnect (already tracked as REC-F01).
- A retryable-vs-permanent failure taxonomy (rejected for D-07).
- Route re-editing of an unsynced trail (D-16 keeps REC-05 to metadata; possible later).
- Concurrent drains of several trails as a user-visible choice (left to the planner as an implementation detail; a user-facing concurrency *control* is out of scope).
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| REC-01 | Capturing a trail with no connection saves it (recording or GPX import), no offline save-failure ever shown | §Architecture Patterns "The central gap" — `_onSave` has **no local-write branch today**; §Code Examples 1 |
| REC-02 | A saved unsynced trail appears in `/profile/<handle>/trails` immediately, before reaching the server | §Architecture Patterns "own-trails list is 100% network" finding; §Code Examples 4 |
| REC-03 | Unsynced trail visibly distinguishable from synced and from downloaded-for-offline | D-08/D-09/D-10 (locked) + §Don't Hand-Roll badge-precedent |
| REC-04 | Survives restart, stays tied to the capturing account, hidden (not deleted) from other accounts | §Architecture Patterns "account scoping is already solved for downloads — extend, don't reinvent" |
| REC-05 | Open/review/edit an unsynced trail's title/description/category/photos while offline | §Architecture Patterns "the three-way `_onSave` branch" |
| REC-06 | Offline, own-trails list still renders: unsynced + downloaded-authored-by-me, states it's offline-only | §Architecture Patterns "own-trails list is 100% network" finding |
| SYNC-01 | Unsynced trail uploads on its own on foreground + connectivity, no hiker action | §Architecture Patterns "no app-wide lifecycle observer exists yet"; §Don't Hand-Roll connectivity |
| SYNC-02 | Upload progress/failure visible inline on the trail itself, not a separate screen | §Code Examples 5 (progress-state precedent, `DownloadingTrailIds`) |
| SYNC-03 | Manual retry for a failed/stalled upload | Same drain-provider surface as SYNC-01/02 |
| SYNC-04 | An interrupted upload never produces a duplicate trail on retry | §Common Pitfalls 3 "no backend idempotency exists — client-only resume" |
| SYNC-05 | Once uploaded, the trail keeps its identity in place, not a second entry | §Architecture Patterns "in-place promotion" |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

- **Tech stack**: all work is Dart/Flutter under `app/lib/**`; ObjectBox 5.3.1 for on-device storage, Riverpod 3.3.1 for state, dio 5.9.2 for HTTP. `/app` only exists on `feature/app` (current branch) — confirmed by git status.
- **Naming**: snake_case files, `_store`/`_provider` suffixes, `{entity}_{operation}` function names, `is`-prefixed booleans, PascalCase types/classes.
- **l10n**: new strings go into `app/lib/i18n/app_en.arb` first, then `flutter gen-l10n`; other 13 locales fall back to English.
- **Error handling**: API calls wrapped in try/catch → `APIError`; store functions catch and degrade rather than throw; components null-guard (`trail?.expand?.waypoints ?? []`).
- **No linter config** — TypeScript/Dart strict mode is the only enforced discipline; no ESLint/Prettier equivalent for Dart beyond `flutter_lints` (already in `dev_dependencies`).
- **GSD workflow enforcement**: file-changing tools must go through a GSD entry point — not relevant to research itself, but the plan this research feeds must route through `/gsd-execute-phase`.

## Summary

REC-01/SYNC-01 are **not an extension of an existing offline-write path — there is no such path today.** A repo-wide grep confirms `TrailEntity` is written to ObjectBox from exactly one call site, `TrailDownloadService.downloadTrail` (`trail_download_service.dart:137`, via `TrailEntity.fromModel`). `trail_create_screen.dart:_onSave` (`:364-469`) calls only the network `trailSaveProvider.createTrail`/`updateTrail`; on any failure — including plain offline — it shows the generic `error_saving_trail` toast. Building REC-01 is genuinely new: a local-write branch that persists a `Trail` into `TrailEntity` with no server round trip, using the same entity the download path already uses (per the design note's decision 1 — no separate pending-queue entity).

Three schema-shaped facts, all confirmed in code, drive the rest of the phase. First, `TrailEntity.id` is `@Unique(onConflict: ConflictStrategy.replace)` and doubles as the create/update discriminator at `trail_create_screen.dart:401` (`trail.id.isEmpty`) — an unsynced trail has no server id, so it needs (a) a separate local identity so ObjectBox doesn't collide two different empty-id rows under `replace`, and (b) an owner field, since `savedByUserIds` is semantically "downloaded" and D-10 forbids overloading it. Second, of the four producers of not-yet-uploaded waypoints, **three already mint `id: ''`** (GPX-derived at `gpx_conversion_util.dart:531`, the trail itself, `_resolveTags` for tags) but **two, not one**, mint a non-empty synthetic id that D-06's audit did not fully catch: the photo-EXIF path CONTEXT.md names (`trail_create_screen.dart:194`, `'${now.microsecondsSinceEpoch}-${created.length}'`) **and** the manually-created-waypoint stub at `trail_create_screen.dart:160` (`DateTime.now().microsecondsSinceEpoch.toString()`), which survives unchanged through `waypoint_create_screen.dart`'s `copyWith` and into the saved trail. Both need the same fix: `id: ''` plus a separate local key for list identity (`Waypoint` has no such field yet; the existing non-serialized `marker`/`localPhotos` fields, both `@JsonKey(includeFromJson: false, includeToJson: false)`, are the pattern to copy). Third, `/profile/<handle>/trails` (`profile_trails_provider.dart`) is **100% network** — `POST /profile/$handle/trails` with no ObjectBox fallback — and its `TrailSearchResult` model hardcodes `isLocal => false` (`global_search_models.dart:80`). REC-02/REC-06 require making this screen local-first for the hiker's own handle, which the roadmap already flagged as "a meaningful chunk of the phase, not a detail" — confirmed true by this research.

Account scoping, however, is *not* new work — it is a direct extension of a pattern already proven for downloads. `currentAccountId(store)` (`current_account.dart:23`) plus a `.containsElement(userId)`/owner-equality filter is the exact shape used by all three existing `TrailEntity` readers (`trail_library_provider.dart:28`, `trail_provider.dart:73-75`, `navigation_launch_util.dart:39-41`), and `account_data_purge_util.dart` already documents *why* trail rows are scoped-not-deleted on logout. The new owner field just needs the same filter added everywhere unsynced trails are read. Connectivity/foreground triggers (D-15) are two separate gaps: `onlineStatusProvider` (`online_status_provider.dart`) already tracks connectivity live via the shared `Api` interceptor, so "connectivity regained while open" is a `ref.listen` away — but **no app-wide `WidgetsBindingObserver` exists**; `MainApp` (`main.dart:72`) is a plain `ConsumerStatefulWidget`, and the only existing lifecycle observer (`navigation_screen.dart:111`) is scoped to that one screen. The "foreground" half of D-15 needs a new observer, most naturally added to `MainApp`.

**Primary recommendation:** Build the drain as a new `keepAlive` Riverpod provider (`TrailSyncNotifier` or similar) modeled directly on the existing `DownloadingTrailIds` provider (`trail_download_state_provider.dart:27-30`, `Set<String> build() => {}` + a `download()` method) — same "in-flight id set" shape, same account-scoping discipline, same `finally`-guarded cleanup. Persist sync state as new fields on `TrailEntity` (not a separate entity, per the design note's already-made decision 1), write the local-save path into `trail_create_screen.dart:_onSave` as a genuinely new offline branch, and rebuild `/profile/<handle>/trails` as local-first the same way `library_screen.dart` already is (pure ObjectBox read, no network dependency) but merged with the existing network search result via the shared `TrailSummary` interface both `Trail` and `TrailSearchResult` already implement.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Local trail/waypoint/photo persistence | Client (ObjectBox on-device store) | — | REC-01/02/04/05/06 are explicitly offline; no server involvement until drain |
| Sync-state tracking (Pending/Uploading/Failed) | Client (ObjectBox field + Riverpod provider) | — | D-08; must survive restart, so lives on `TrailEntity`, not just in-memory |
| Upload drain (tag→trail→waypoint sequence) | Client (Dart, calling existing API routes) | API/Backend (unchanged endpoints) | D-05/SYNC-04 explicitly scoped client-only; roadmap forbids server contract changes this phase |
| Idempotency on retry | Client (resume-by-empty-id) | — | No backend idempotency-key mechanism exists or is planned (§Common Pitfalls 3) |
| Own-trails list composition (local + network) | Client (Riverpod provider merging ObjectBox + search API) | API/Backend (existing `/profile/{handle}/trails` search) | REC-02/06; server has no concept of "unsynced," so merge must happen client-side |
| Connectivity/foreground detection | Client (existing `onlineStatusProvider` + new `WidgetsBindingObserver`) | — | D-15; both signals are device-local, no server round trip needed to detect them |
| Photo durability (copy to app-owned dir) | Client (`dart:io` File copy into app-documents) | — | D-01; OS cache purging is a device-local concern |

## Standard Stack

No new external packages are required for this phase. Every capability maps onto libraries already in `app/pubspec.yaml`:

| Library | Version (pubspec) | Purpose in this phase |
|---------|--------------------|------------------------|
| `objectbox` / `objectbox_flutter_libs` | ^5.3.1 | New fields on `TrailEntity`/`WaypointEntity` for owner, sync-state, local id, idempotency bookkeeping |
| `objectbox_generator` (dev) | ^5.3.1 | Regenerates `objectbox.g.dart`/`objectbox-model.json` after entity changes |
| `flutter_riverpod` / `riverpod_annotation` | ^3.3.1 / ^4.0.2 | New `keepAlive` drain provider, per-trail progress state, `WidgetsBindingObserver` wiring |
| `dio` | ^5.9.2 | Existing `PUT /tag`, `PUT /trail/form`, `PUT /waypoint` calls, reused unchanged for the drain |
| `path` / `path_provider` | ^1.9.1 / ^2.1.5 | App-owned photo directory (D-01), following the `library/`/`avatars/`/`regions/` precedent |
| `freezed_annotation` / `json_annotation` | ^3.1.0 / ^4.11.0 | New fields on the `Trail`/`Waypoint` freezed models mirroring the entity changes |

**Explicitly NOT needed:** `connectivity_plus` or any OS-level connectivity-change package. This codebase deliberately uses an *active reachability probe* (`isBackendReachable`, `connectivity_util.dart:19-26`) fed into `onlineStatusProvider` by the shared `Api` client's interceptor (`api_provider.dart:49-72`) — this already gives "connectivity regained" as a provider state transition, no new package required. Introducing `connectivity_plus` would report OS radio state, not backend reachability, and would duplicate an already-solved problem.

**Installation:** none — no `pubspec.yaml` changes required for the drain/persistence architecture itself. (An optional `uuid` package could replace the codebase's existing timestamp-string local-id convention — see Assumptions Log A1 — but is not required.)

## Package Legitimacy Audit

**N/A — this phase installs no new external packages.** See Standard Stack above: every capability is satisfied by libraries already present in `app/pubspec.yaml`. If the planner elects to add `uuid` for local-id generation (Assumptions Log A1), it must be run through the Package Legitimacy Gate at that time; this research does not pre-approve it.

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────── ON-DEVICE (offline-capable) ───────────────────────────┐
│                                                                                     │
│  Recording ends / GPX import ──▶ trailFromGpx() ──▶ trail_create_screen (Phase 35) │
│                                          │                                          │
│                                          ▼                                          │
│                              _onSave() [NEW 3-way branch]                          │
│                              ┌───────────┴────────────┐                            │
│                    trail has real          trail has no server id                  │
│                    server id (edit)         ┌─────────┴─────────┐                  │
│                         │              no local row yet   local row exists         │
│                         ▼              (first local save)  (re-edit, still         │
│                 network updateTrail          │              unsynced)              │
│                 (existing path,               ▼                    ▼               │
│                  unchanged)          WRITE new TrailEntity   UPDATE existing        │
│                                       (D-01..07: copy photos, TrailEntity row       │
│                                        id='', localId=new,   (still id='',          │
│                                        owner=account,        owner unchanged,       │
│                                        syncState=Pending)     syncState untouched)  │
│                                          │                         │                │
│                                          └───────────┬─────────────┘                │
│                                                       ▼                             │
│                                     /profile/<handle>/trails [NEW: local-first]     │
│                                     merges ObjectBox (owner=me, syncState!=Synced    │
│                                     OR savedByUserIds contains me)                   │
│                                     with network TrailSearchResult when online       │
│                                     (REC-02, REC-06, D-11 flat list)                 │
│                                                                                       │
└───────────────────────────────────────────┬─────────────────────────────────────────┘
                                             │
                          onlineStatusProvider transition (online) OR
                          new WidgetsBindingObserver.resumed  (D-15)
                                             │
                                             ▼
┌────────────────────────── DRAIN (background, foreground-triggered) ───────────────┐
│                                                                                     │
│   TrailSyncNotifier.drain()  [keepAlive provider, modeled on DownloadingTrailIds]   │
│   for each owner-scoped, syncState in {Pending, Failed-retry} TrailEntity row:      │
│                                                                                     │
│     1. resolvedTags = for each tag: id.isEmpty ? PUT /tag : reuse         (D-05/06)│
│     2. IF trail.id.isEmpty (not yet created server-side):                         │
│          PUT /trail/form  →  write returned server id into TrailEntity IMMEDIATELY │
│          (this write is what makes a crash-after-create-before-waypoints safe —    │
│           see Common Pitfalls 3: no server idempotency key exists)                 │
│        ELSE: server trail already exists from a prior partial attempt, skip create │
│     3. for each waypoint with id.isEmpty: PUT /waypoint  →  write returned id back  │
│        into the local waypoint list keyed by the NEW local key (not the empty id)  │
│     4. upload local photos (Waypoint.localPhotos / Trail.localPhotos)              │
│     5. ALL steps succeeded → syncState=Synced, delete local photo copies (D-02),    │
│        clear owner-vs-savedByUserIds ambiguity is structural (D-10)                │
│        SOME step failed → backoff (D-07); after N attempts syncState=Failed        │
│                                                                                     │
│   Per-item state surfaces inline via trail_card.dart / trail_list_item.dart        │
│   (SYNC-02) — same badge slot pattern as the existing "downloaded" badge           │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### The central gap: `_onSave` has no local-write branch today

`trail_create_screen.dart:_onSave` (`:364-469`) is a two-way branch only:

```dart
// trail_create_screen.dart:401-418 (current code, before this phase)
final result = trail.id.isEmpty
    ? await ref.read(trailSaveProvider.notifier).createTrail(...)   // network PUT /trail/form
    : await ref.read(trailSaveProvider.notifier).updateTrail(...);  // network POST /trail/form/{id}
```

Both branches call the network unconditionally; the surrounding `try`/`catch` (`:401-469`) shows `l10n.error_saving_trail` on **any** exception, including the `DioException` an offline `PUT /trail/form` throws. This is the literal offline-save failure REC-01 must eliminate. Grounding for "no local-write path exists anywhere else": `TrailEntity.fromModel` (`trail_entity.dart:108`) has exactly one call site in `lib/` — `trail_download_service.dart:137`, inside the *download* flow. There is no code path today that writes a self-authored trail into ObjectBox.

### The three-way `_onSave` branch this phase must build

Because `trail.id.isEmpty` (D-06's rule) is true both for "never saved anywhere" and for "saved locally, still unsynced," `_onSave` needs a third signal — whether a local row already exists for this draft — to route correctly:

1. **`trail.id` non-empty** → existing network `updateTrail` path, unchanged (a real, already-uploaded trail).
2. **`trail.id` empty AND no local row exists yet** (first save of a brand-new recording/import, online or offline) → write a **new** `TrailEntity` locally (D-01–07), regardless of connectivity, per the Komoot/AllTrails "local save never touches the network" model the design note documents. This is the REC-01 entry point.
3. **`trail.id` empty AND a local row already exists** (re-opening and re-saving an unsynced trail's metadata — REC-05) → **update** the existing `TrailEntity` row in place; still no network call.

The signal for case 2 vs 3 is the presence of a **local id** on the in-memory `Trail` model — a new field mirroring `TrailEntity`'s new local-identity field, threaded through the same `Trail.fromJson`/`TrailEntity.toModel()` round trip that already carries `isLocal`/`localPhotos` (`trail_entity.dart:157-194`). Whether this phase adopts "always write locally first, drain immediately if online" (Komoot/AllTrails' actual model, matching the design note) or "network first, fall back to local only when offline" is a planning decision within the D-05/D-15 constraints — the design note's own research (`.planning/notes/offline-recording-deferred-upload-design.md`, "How Komoot and AllTrails solve it") favors the former: it removes the online/offline branch entirely and makes upload a uniform background projection.

### `/profile/<handle>/trails` is 100% network — REC-02/REC-06's real cost

`profile_trails_provider.dart` (`_fetchPage`, `:80-124`) calls `POST /profile/$handle/trails` with no ObjectBox fallback and no local branch at all. Compare `library_screen.dart` (`:49-52`), which is **already** fully local-first — `ref.watch(trailLibraryProvider)` reads ObjectBox directly, no network dependency. `TrailListItem`/`TrailCard` (used by both screens) already operate on the shared `TrailSummary` interface (`trail_summary.dart`), which both `Trail` (`trail.dart:64`, `implements TrailSummary`) and `TrailSearchResult` (`global_search_models.dart:21`) implement — but `TrailSearchResult` hardcodes `bool get isLocal => false` (`global_search_models.dart:80`) and has no server-side "unsynced" concept to report, so an unsynced trail can only ever surface through the `Trail`/ObjectBox side of a merge, never through the search API. Practically: the own-trails screen needs a new provider that (a) reads owner-scoped `TrailEntity` rows from ObjectBox (unsynced + this-account's-authored-and-downloaded, filtered similarly to `trailLibraryProvider` but on an *owner* field, not `savedByUserIds`), (b) when online, also fetches the existing network search page, and (c) merges the two into one `List<TrailSummary>` for the flat list D-11 specifies. Offline (REC-06), step (b) is skipped and the screen states plainly it is showing offline-only content — mirroring `library_screen.dart`'s existing two-flavor-empty-state pattern (`_LibraryEmptyState.icon` vs `.artwork`, `:212-229`) as the UI precedent for a "different reason, different message" empty/banner state.

### Account scoping is already solved for downloads — extend, don't reinvent

Every existing account-scoped `TrailEntity` read follows one shape: read `currentAccountId(store)` (`current_account.dart:23`, a plain function reading `UserEntity` directly — never cached in a provider, "the value must never be served from a Riverpod cache"), return empty/null if no account, otherwise filter the ObjectBox query on a membership/equality predicate. The three existing call sites:

- `trail_library_provider.dart:26-29` — `TrailEntity_.savedByUserIds.containsElement(userId)`
- `trail_provider.dart:70-76` — `TrailEntity_.id.equals(id) & TrailEntity_.savedByUserIds.containsElement(userId)`
- `navigation_launch_util.dart:36-42` — same shape (roadmap's ":40" reference is off by ~1 line but the claim is correct in substance)

A new **owner** field (single account id, not a list — a trail is authored by exactly one account, unlike "downloaded by," which can be several) needs the identical filter shape added to every read path that must surface unsynced trails: the new own-trails-list provider, and the drain provider itself (D-13 requires the drain to skip other accounts' unsynced rows, not just hide them from the UI). `account_data_purge_util.dart` already documents why trail rows survive logout (`:1-17`) — no new purge logic is needed; D-04/D-12's "nothing is deleted on logout" falls out of the existing purge design for free, exactly as the roadmap states.

**One new wrinkle not covered by existing precedent:** `account_scope_invalidation.dart` maintains an explicit list of `keepAlive` providers invalidated on every account switch (`accountScopedProviders`, `:38+`), but **deliberately excludes** `downloadingTrailIdsProvider` because "invalidating them mid-download would desync in-flight download bookkeeping from its `CancelToken`s" (`account_scope_invalidation.dart:16-18`). The new drain provider is the same shape — an in-flight-operation-tracking `keepAlive` provider — and needs the same exclusion reasoning applied deliberately, not by omission: mid-drain invalidation must not desync the drain's own bookkeeping (e.g., "which step of this trail's upload sequence am I on"). This should be a conscious decision in the plan, not an oversight.

### In-place promotion (SYNC-05)

Because `TrailEntity.id` is `@Unique(onConflict: ConflictStrategy.replace)`, writing the server-returned id into the *same* `obxId` row (rather than deleting the local row and creating a new one) is what "keeps its identity in place" means concretely — the row's `obxId` (ObjectBox's internal auto-increment PK) never changes across the sync-state transition, only its `id` field gains a value and its `syncState` flips to `Synced`. Any UI reading by `obxId` (there is none today, but any new per-item progress tracking should key on `obxId` or the new local id, not on `id`, precisely because `id` is empty until the trail syncs).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Detecting connectivity | A new `connectivity_plus`-based OS radio listener | `onlineStatusProvider` (already keepAlive, already fed by the shared `Api` interceptor) | Duplicates an already-solved, already-tested problem; OS radio state ≠ backend reachability, which is the signal that actually matters here |
| In-flight-operation tracking state shape | A bespoke sync-state machine class | The `Set<String> build() => {}` + mutator pattern already used by `DownloadingTrailIds` (`trail_download_state_provider.dart:27-30`) | Proven pattern in this exact codebase for "which trail ids are busy right now," including the `finally`-guarded cleanup this phase's drain needs identically |
| Account-scoped ObjectBox filtering | A new query-building abstraction | The existing `currentAccountId(store)` + `.containsElement`/`.equals` filter shape, copied to the new owner field | Three call sites already establish this exact pattern; a fourth divergent implementation would be an inconsistency, not an improvement |
| Badge/sync-state UI slot | A new standalone "sync status" widget system | The existing badge-rendering call sites in `trail_card.dart`/`trail_list_item.dart` that already branch on `trail.isLocal` for the downloaded badge | REC-03/D-09 is one more mutually-exclusive badge state on the same card, not a new UI surface |
| Pending-uploads inbox | A separate "Pending" screen/list | Inline per-item state on the existing own-trails list (SYNC-02, explicitly locked) | Explicitly rejected by both the user's decision (D-08) and the Komoot/AllTrails precedent research cites |

**Key insight:** almost nothing in this phase's *infrastructure* is new-in-kind — it is the same "account-scoped ObjectBox row, filtered read, keepAlive in-flight-tracking provider, inline badge" shape the download/library subsystem already established in Phase 22-27. The genuinely new work is (a) the write path that currently doesn't exist (`_onSave`'s local branch) and (b) merging two data sources (ObjectBox + network search) on one screen, which nothing in the codebase does today.

## Common Pitfalls

### Pitfall 1: The D-06 audit missed a second synthetic-id producer

**What goes wrong:** CONTEXT.md's D-06 names exactly one exception to "empty id means not-yet-uploaded" — the photo-EXIF waypoint stub at `trail_create_screen.dart:194`. A second, structurally identical producer exists at `trail_create_screen.dart:160` inside `_onCreateWaypoint`:
```dart
// trail_create_screen.dart:159-165 (current code)
final stub = Waypoint(
  id: DateTime.now().microsecondsSinceEpoch.toString(),  // ← non-empty, same problem as :194
  lat: point?.lat ?? trail.lat ?? 0,
  lon: point?.lon ?? trail.lon ?? 0,
  created: DateTime.now(),
  updated: DateTime.now(),
);
```
This stub is handed to `waypoint_create_screen.dart`, which returns it via `waypoint.copyWith(...)` (`:57-66`) — the id is never cleared — and it survives into `trail.expand.waypointsViaTrail` via `_appendWaypoint`/`_replaceWaypoint`.

**Why it happens:** any manually-created waypoint (tap-to-add on the map, not from a photo) goes through this exact stub path, which predates this phase and was written only to give the not-yet-saved waypoint a stable key for list diffing (`_onDeleteWaypoint`/`_replaceWaypoint` match by `.id`).

**How to avoid:** apply the same fix D-06 specifies for the photo-EXIF case — `id: ''`, with list identity carried in a new non-serialized field on `Waypoint` (following the `marker`/`localPhotos` `@JsonKey(includeFromJson: false, includeToJson: false)` precedent, `waypoint.dart:29-33`).

**Warning signs:** if the drain's "resume from step" logic checks `waypoint.id.isEmpty` to decide what still needs uploading, a manually-created waypoint would look "already uploaded" and silently never reach the server.

### Pitfall 2: `trail.id.isEmpty` alone cannot route `_onSave` correctly once local-first saves exist

**What goes wrong:** Before this phase, `trail.id.isEmpty` cleanly meant "never saved." After this phase, it also means "saved locally, still unsynced" (D-06's whole point). A naive two-way `_onSave` branch would try to `createTrail` (network) a second time when a hiker re-opens and re-saves an already-locally-saved draft — either erroring offline (acceptable but wrong UX for what should be silent) or, if online, creating a **second** server trail from what the hiker thinks is one edit session.

**Why it happens:** the discriminator that worked pre-phase (`id.isEmpty` = need to create) stops being sufficient the moment "created locally but not yet on the server" becomes a real, persistent state.

**How to avoid:** the three-way branch described in Architecture Patterns — route on local-row-exists, not just `id.isEmpty`.

**Warning signs:** duplicate trails appearing after a hiker edits an unsynced trail's title before ever regaining connectivity.

### Pitfall 3: No backend idempotency mechanism exists — duplicate-prevention is 100% client state

**What goes wrong:** `PUT /trail/form` (`web/src/routes/api/v1/trail/form/+server.ts:30-36`) is a thin `uploadCreate` passthrough with no client-supplied id, no idempotency key, and no dedup logic — PocketBase mints a fresh record id on every call. If the drain calls `PUT /trail/form` twice for the same unsynced trail (e.g., because a crash happened *after* the server accepted the create but *before* the returned id was persisted locally), the result is two server trails, not one rejected duplicate.

**Why it happens:** this phase's boundary (per CONTEXT.md's `<domain>`) explicitly excludes "any change to what `/trail/form` or `/waypoint` accept server-side" — so SYNC-04 must be satisfied entirely client-side.

**How to avoid:** the returned server id from a successful `PUT /trail/form` must be written into the local `TrailEntity` row **synchronously, before** the next drain step (waypoint uploads) begins — not batched with the rest of the drain's success handling. This is exactly D-05's "resume from step" requirement, but it has a sharp edge: the write-back must happen even if everything *after* it fails, or a retry will re-create the trail. `trail_save_provider.dart:70` (`var model = Trail.fromJson(response.data);`) is the point in the existing `createTrail` flow where this id becomes available — the drain's equivalent must persist it immediately at that point, not wait for the whole method to return.

**Warning signs:** duplicate trails on the server after a drain interrupted mid-run (kill the app between trail-create and waypoint-upload during testing).

### Pitfall 4: No app-wide foreground/lifecycle observer exists yet

**What goes wrong:** assuming `onlineStatusProvider`'s connectivity signal alone satisfies D-15. It only covers half — "connectivity regained while open." The "app foregrounded" trigger has no existing hook: `MainApp` (`main.dart:72`) is a plain `ConsumerStatefulWidget` with no `WidgetsBindingObserver` mixin; the only lifecycle observer in the app is scoped to `navigation_screen.dart` (`:111`, `didChangeAppLifecycleState`, `:593-602`) and would not fire for the app being foregrounded while some other screen is on top.

**Why it happens:** no prior phase needed an app-wide foreground signal; navigation's observer was purpose-built for GPS stream lifecycle, not a general-purpose hook.

**How to avoid:** add `WidgetsBindingObserver` to `MainApp` (or an equivalent app-root widget) and trigger the drain from `AppLifecycleState.resumed`, alongside a `ref.listen` on `onlineStatusProvider` for the connectivity-regained half.

**Warning signs:** the drain only ever firing when connectivity flips while already foregrounded, never on a fresh app open with connectivity already present.

### Pitfall 5: `onlineStatusProvider` is optimistic — a drain decision at a cold moment can be wrong

**What goes wrong:** `OnlineStatus.build() => true` (`online_status_provider.dart:50`) — the provider assumes online until proven otherwise by real traffic. Phase 35's own `OFFUI-04` hit this exact trap (documented in CONTEXT.md's code_context) and had to call `.refresh()` before trusting the value at a cold moment.

**Why it happens:** the interceptor only *settles* the value from ordinary request/response traffic; at app-foreground time, no traffic may have happened yet.

**How to avoid:** the drain's foreground/connectivity-regained handler should call `ref.read(onlineStatusProvider.notifier).refresh()` (already exists, `online_status_provider.dart:70-76`) before deciding whether to attempt a drain — exactly the pattern `main.dart:90` already uses at startup (`unawaited(ref.read(onlineStatusProvider.notifier).refresh())`).

**Warning signs:** a drain attempt that immediately fails because the optimistic `true` default was trusted in airplane mode right after a cold app launch.

### Pitfall 6: ObjectBox model changes are additive-only in this codebase — don't invent a migration step

**What goes wrong:** over-engineering a data-migration/backfill task for existing on-device rows when adding the new owner/sync-state/local-id fields.

**Why it happens:** it's a reasonable instinct for a schema change, but this app is pre-production and has an established precedent: Phase 27 removed two `TrailEntity` fields (`pmTiles`/`demPmTiles`) with **no migration step**, reasoning explicitly that "field removal is a supported regeneration for a pre-production app" (STATE.md, `[Phase 27] [27-02]`). Adding new nullable/defaulted fields is even lower-risk than removal — `build_runner` regenerates `objectbox.g.dart` and `objectbox-model.json` (new property ids appended automatically, confirmed by the existing `objectbox-model.json` structure) and existing rows simply read the new fields at their defaults.

**How to avoid:** just add the fields to `TrailEntity`/`WaypointEntity` and run `dart run build_runner build --delete-conflicting-outputs` (the exact command Phase 34 used, `34-04-SUMMARY.md:246`) — no manual migration task needed. A backfill task IS needed for one specific thing, though: existing downloaded `TrailEntity` rows have no owner value, so any new owner-scoped filter must treat a null/absent owner as "not authored by anyone" (never matching, never leaking) rather than crashing or defaulting to the current account.

**Warning signs:** a plan that includes a dedicated "migrate existing ObjectBox rows" task — for this codebase, that is almost certainly unnecessary scope creep.

## Code Examples

### 1. The exact offline-failure path REC-01 must intercept

```dart
// trail_create_screen.dart:401-469 (current code, before this phase)
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
  // This is what fires today when offline — the exact failure REC-01 removes.
  ref.read(toastProvider.notifier).add(ToastMessage(
    type: ToastType.error,
    icon: FontAwesomeIcons.circleExclamation,
    text: l10n.error_saving_trail,
  ));
} finally {
  if (mounted) setState(() => _saving = false);
}
```

### 2. The in-flight-tracking provider pattern to copy for the drain

```dart
// trail_download_state_provider.dart:27-30 — the shape to mirror
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
The drain provider should follow the identical re-entry-guard + `finally`-cleanup shape, keyed on the trail's local id (since `id` is empty pre-sync).

### 3. The account-scoped ObjectBox read shape to extend to the new owner field

```dart
// trail_library_provider.dart:26-29 — the pattern; apply the same shape
// to a new `owner` field for the own-trails-list and drain queries
final box = store.box<TrailEntity>();
final query = box
    .query(TrailEntity_.savedByUserIds.containsElement(userId))
    .build();
```

### 4. `TrailSearchResult` hardcodes `isLocal => false` — the merge point REC-02/06 need

```dart
// global_search_models.dart:78-83 (current code)
@override
bool get isLocal => false;

@override
List<String> get localPhotos => [];
```
Both `Trail` and `TrailSearchResult` implement `TrailSummary` (`trail_summary.dart`), which is what `TrailListItem`/`TrailCard` consume — the merge for the own-trails screen can produce `List<TrailSummary>` by concatenating owner-scoped `Trail`s (from ObjectBox) with `TrailSearchResult`s (from the network), with no changes needed to the card/list-item widgets themselves.

### 5. The library screen's local-first pattern — precedent for the own-trails list rewrite

```dart
// library_screen.dart:49-52 — fully local-first today, no network call
final trailLibrary = ref.watch(trailLibraryProvider);   // pure ObjectBox read
final visible = _filtered(trailLibrary, filterAsync.value);
```
Contrast with `profile_trails_provider.dart:_fetchPage` (`:80-124`), which is pure network. REC-02/06's own-trails screen needs to become a hybrid of these two patterns — not simply copy one or the other.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | No `uuid` package is needed; the existing codebase convention (`DateTime.now().microsecondsSinceEpoch.toString()`, already used for the waypoint stub at `trail_create_screen.dart:160` and for waypoint list keys elsewhere) is sufficient for a local trail/waypoint identity key. `[ASSUMED]` | Standard Stack, Architecture Patterns | Low — collision risk is negligible for single-device, sequential local creation; if the planner disagrees, adding `uuid` is a one-line pubspec change subject to the Package Legitimacy Gate at plan time |
| A2 | "Always write locally first, then drain if online" (rather than "try network first, fall back to local only if offline") is the better default per the Komoot/AllTrails research the design note already cites — but CONTEXT.md does not explicitly lock this choice; it locks the *drain* semantics (D-05..07) and *state* semantics (D-08..11), not the save-time branch order. `[ASSUMED]` | Architecture Patterns, "The three-way `_onSave` branch" | Medium — if the planner instead builds "network-first, offline-fallback," REC-01's "no offline save-failure ever shown" still holds, but the code path differs from what this research recommends; worth confirming with the user or treating as Claude's Discretion at plan time, since CONTEXT.md's "Claude's Discretion" section does not explicitly list this either |
| A3 | The new `owner` field on `TrailEntity` should be a single non-list `String?` (one author per trail), distinct in shape from the existing list-valued `savedByUserIds` (many downloaders per trail). `[ASSUMED — but strongly implied by D-10's own reasoning and by the domain: authorship is 1:1, downloading is 1:N]` | Architecture Patterns "Account scoping" | Low — the cardinality difference is a direct consequence of what each field means, not a debatable design choice |

**If this table is empty:** N/A — two assumptions above need confirmation; none block planning, both are flagged Low/Medium risk.

## Open Questions

1. **Save-time branch order: local-first-always, or offline-fallback?**
   - What we know: D-05/D-15 lock the *drain's* behavior; the Komoot/AllTrails precedent the design note cites favors "local save never touches the network" as the uniform behavior even when online.
   - What's unclear: CONTEXT.md never explicitly locks whether an *online* save also writes locally first (then drains immediately) or goes straight to network as it does today, only falling back to local when offline.
   - Recommendation: treat as a planning decision informed by A2 above; the uniform local-first model is simpler to implement (one code path, not two) and matches both named reference apps, so recommend it unless the planner or a fast-follow discuss-phase surfaces a reason not to.

2. **Where does the new owner field's value come from for a trail created while signed in, if the account signs out mid-session and back in as the same account?**
   - What we know: `currentAccountId(store)` is read fresh at save time, not cached (`current_account.dart:8-13`'s own doc comment explains why).
   - What's unclear: nothing structurally — this is a non-issue given the existing pattern — but worth the planner explicitly re-reading `currentAccountId` at drain time too (not caching it once at drain-provider construction), matching the same "never cache" discipline the rest of the codebase already enforces.
   - Recommendation: no action needed beyond following the established pattern; flagged only so the plan doesn't accidentally introduce a stale-cached-id bug.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V2 Authentication | No | Unchanged — drain reuses the existing authenticated `Api` client/cookie jar; no new auth surface |
| V3 Session Management | No | Same reasoning — no new session handling introduced |
| V4 Access Control | Yes | Owner-field filtering must be applied at **every** new read/write path that touches unsynced `TrailEntity` rows (own-trails list, drain queue, delete), mirroring the existing `savedByUserIds` discipline exactly — a missed filter is the same class of bug the roadmap already flags ("all three `TrailEntity` readers filter on `savedByUserIds`... an unsynced row is invisible unless those gain an owner clause") |
| V5 Input Validation | Yes | Photo copy destination paths must use `package:path`'s `p.join`, never string concatenation, matching `pb_filter_util.dart`'s escaping discipline and the existing `map_cache_path.dart` "never string-concatenate a token into a path" precedent (STATE.md `[15-03]`) |
| V6 Cryptography | No | No new cryptographic surface |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|----------------------|
| Cross-account data leak (account B sees/uploads account A's unsynced trail) | Information Disclosure / Elevation of Privilege | Owner-field filter on every read path (own-trails list, drain, delete), same shape as `savedByUserIds` filtering already proven in `trail_library_provider.dart`/`trail_provider.dart`/`navigation_launch_util.dart` |
| Path traversal via a malformed local photo filename | Tampering | `p.join` (never raw string concatenation) for the new app-owned photo directory, matching the existing `map_cache_path.dart` whitelist-and-reject discipline |
| Duplicate server-side trail from a retried drain (data integrity, not classic security, but explicitly a named requirement — SYNC-04) | Tampering / Repudiation-adjacent | Client-side resume-by-empty-id (§Common Pitfalls 3) — write the server id back to the local row synchronously before any subsequent step |
| Orphaned local photo copies accumulating disk usage after a crash between "server accepted" and "local cleanup" | Denial of Service (disk exhaustion, low severity) | D-02's startup orphan sweep — locked decision, not new research, but worth the plan implementing it as a real startup task, not a "nice to have" |

## Sources

### Primary (HIGH confidence — direct repo inspection)
- `app/lib/entities/trail_entity.dart`, `app/lib/entities/waypoint_entity.dart` — current schema, `@Unique(onConflict: replace)`, `fromModel`/`toModel`
- `app/lib/routes/trail_create_screen.dart` — `_onSave` (`:364-469`), waypoint stub id minting (`:159-165`, `:193-203`), offline map fix (`:543,628`)
- `app/lib/provider/trail/trail_save_provider.dart` — `createTrail`/`updateTrail`, the exact `PUT /tag`→`PUT /trail/form`→`PUT /waypoint` sequence
- `app/lib/provider/trail/trail_library_provider.dart`, `trail_provider.dart`, `app/lib/util/navigation_launch_util.dart` — the three existing account-scoped `savedByUserIds` read sites
- `app/lib/util/account_data_purge_util.dart`, `app/lib/util/current_account.dart`, `app/lib/util/library_membership.dart` — account-scoping precedent
- `app/lib/provider/online_status_provider.dart`, `app/lib/provider/api_provider.dart`, `app/lib/util/connectivity_util.dart` — connectivity signal architecture
- `app/lib/provider/trail/trail_download_state_provider.dart`, `app/lib/services/trail_download_service.dart` — in-flight-tracking provider shape and photo-copy-to-app-docs precedent
- `app/lib/provider/profile/profile_trails_provider.dart`, `app/lib/routes/profile_trail_screen.dart`, `app/lib/routes/library_screen.dart` — network-only vs local-first screen comparison
- `app/lib/models/trail.dart`, `app/lib/models/trail_summary.dart`, `app/lib/models/global_search_models.dart`, `app/lib/models/waypoint.dart` — shared `TrailSummary` interface, `isLocal`/`localPhotos` provenance
- `app/lib/components/trail/trail_dropdown.dart`, `app/lib/components/trail/trail_card.dart`, `app/lib/components/trail/trail_list_item.dart` — badge/delete/download landmine sites named in ROADMAP.md
- `app/lib/util/account_scope_invalidation.dart`, `app/lib/main.dart` — account-switch invalidation list and absence of an app-wide lifecycle observer
- `app/lib/util/gpx_conversion_util.dart` (`trailFromGpx`, `:489-607`) — confirms GPX-derived waypoints already mint `id: ''`
- `app/lib/util/form_data_util.dart`, `app/lib/provider/waypoint/waypoint_provider.dart` — exact multipart body construction for `/trail/form` and `/waypoint`
- `app/lib/objectbox-model.json` — confirms additive, auto-id-assigning ObjectBox model format
- `app/test/entities/trail_entity_test.dart`, `app/test/components/trail/trail_dropdown_delete_gate_test.dart` — existing test patterns (pure `TrailEntity` construction, source-level regression gates)
- `web/src/routes/api/v1/trail/form/+server.ts` — confirms no server-side idempotency mechanism exists
- `.planning/phases/35-offline-trail-creation/35-SUMMARY.md`, `.planning/phases/34-dart-conversion-port/34-04-SUMMARY.md` — Phase 35 delivery record, `build_runner` command precedent
- `.planning/notes/offline-recording-deferred-upload-design.md`, `.planning/research/questions.md`, `.planning/ROADMAP.md` §Phase 36, `.planning/REQUIREMENTS.md` — phase design record and locked scope boundary

### Secondary (MEDIUM confidence)
- None — no web/external research was needed for this phase; every claim traces to a repo file.

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new packages, every capability maps onto pubspec dependencies already installed and already used for structurally identical problems (download progress tracking, account scoping, connectivity detection)
- Architecture: HIGH — every architectural claim (the missing local-write path, the network-only own-trails screen, the missing lifecycle observer, the double synthetic-id gap) is grounded in a specific file:line read directly in this session, not inferred
- Pitfalls: HIGH — all six pitfalls are either directly observed code defects (Pitfalls 1, 2, 4) or logically necessary consequences of the phase's own locked constraints (Pitfalls 3, 5, 6), not speculative

**Research date:** 2026-08-02
**Valid until:** Until the underlying code changes — this research is a snapshot of `feature/app` at commit `6f269067`. Treat as stale the moment Phase 36 planning/execution begins modifying `trail_create_screen.dart`, `trail_entity.dart`, or `profile_trails_provider.dart`, since line numbers cited above will shift.
