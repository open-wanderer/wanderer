# Phase 27: Legacy Cleanup - Research

**Researched:** 2026-07-24
**Domain:** Flutter/Dart code deletion (dead-code removal, ObjectBox schema field removal, notification-service pruning) — no new libraries, no new architecture
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**downloadTrail() Surgery Scope**
- **D-01:** `downloadTrail()` remains ONE method (not split). Delete `_downloadMapTiles`, `_fetchCellList`, `_pollUntilReady`, and all `tileResult`/`pmTiles`/`demPmTiles` wiring from `trail_download_service.dart`. Keep photo downloads, waypoint-photo downloads, and the best-effort Valhalla nav-cache write exactly as they are — those aren't "tile-download methods" per CLEAN-01's scope.
- **D-02:** `onGeneratingChanged` is removed entirely from `downloadTrail()`'s signature (and from `_downloadPhotos`/`_downloadWaypointPhotos`, which never used it anyway). There is no more server-side "tile generation" wait once tiles are gone — only real, trackable photo-download progress remains, so `isGenerating`/`handleGeneratingChanged` in `trail_download_service.dart` go away too.
- **D-03:** In `trail_download_state_provider.dart` (Phase 26's guard, already UAT-passed), remove the `onGeneratingChanged` callback wiring and the WR-01 `hasSelectedPackages` branch that reacted to it — there's nothing left to report as "generating." The aggregate notification shows real download progress only (trail photo/nav-cache progress + selected region package progress via the existing `vectorLatched`/`demLatched` monotonic latch from 26-05).
- **D-04:** `_downloadTracked` (the shared byte-progress-with-fake-fallback helper) stays unchanged — it's still needed by `_downloadPhotos` after tile removal. Do not delete it alongside the tile-specific methods.

**Legacy Sweep — Descoped**
- **D-05 (scope cut, not a gray-area decision):** CLEAN-02 ("one-time cleanup sweep deletes orphaned legacy tile files... disk-usage figure is accurate") and ROADMAP.md's Phase 27 success criterion #2 are dropped from this phase entirely, per explicit user decision. Rationale given: the app is pre-production, so there's no real install base with orphaned legacy tile files to clean up — any dev/test device carrying old `library/{trailId}/tiles/` directories can just be wiped/reinstalled manually. This phase is CLEAN-01 (code removal) only. **Downstream note:** `.planning/REQUIREMENTS.md` (CLEAN-02, currently `Pending`, mapped to Phase 27) and `.planning/ROADMAP.md`'s Phase 27 success criterion #2 should be updated to reflect this cut in a future editing pass — the planner should NOT create a plan for CLEAN-02, and should flag CLEAN-02 as intentionally unaddressed by this phase rather than treating its absence as a gap.

**TrailEntity Field Removal Mechanics**
- **D-06:** Delete `pmTiles`/`demPmTiles` fields outright now from `TrailEntity` (ObjectBox entity), the `Trail` freezed model, and regenerate `.g.dart`/`.freezed.dart`/`objectbox-model.json`. Confirmed via grep: zero remaining readers anywhere in `app/lib` outside the entity/model definitions themselves and `trail_download_service.dart` (Phase 25 already moved map rendering to the region-based pipeline). Matches CLAUDE.md's "no migration path, pre-production app" instruction — no ObjectBox migration step needed for a field removal.

**Trail-Scoped Tile-Download UI Remnants**
- **D-07:** Delete `download_notification_service.dart`'s `showGenerating()` method and its `'Generating map tiles...'` string outright, since D-02/D-03 make it unreachable from the trail-download path. **Before deleting, the executor MUST grep for other callers** (e.g. region downloads in `tile_repository_provider.dart`/`tile_repository_manager.dart`) — if region downloads also call `showGenerating()`, keep the method and only remove the trail-path call site; the user's intent is "delete if truly dead," not "delete unconditionally."

### Claude's Discretion
- Exact grep/verification approach for confirming zero remaining references before deleting `pmTiles`/`demPmTiles` and `showGenerating()` — implementation detail, but MUST be done (not assumed) per D-06/D-07's conditions.
- Whether to regenerate ObjectBox codegen (`build_runner`) as a single pass at the end or incrementally per file — implementation detail.
- Any other genuinely-dead tile-generation-only strings/assets beyond `showGenerating()`'s copy that a full-file read of `trail_download_service.dart`/`download_notification_service.dart` surfaces — researcher/planner should do a final sweep of both files for anything tile-generation-specific this discussion didn't explicitly enumerate. **Research finding: `app/lib/models/map_cell.dart` (+ generated `.freezed.dart`/`.g.dart`) is exactly such an item — see Summary and Pitfall 1 below.**

### Deferred Ideas (OUT OF SCOPE)
- CLEAN-02 (one-time legacy-file cleanup sweep) and ROADMAP Phase 27 success criterion #2 — cut from this phase per D-05, not deferred to a later phase; the user's position is that it's unnecessary given the pre-production status, not merely "not now."
- "Way Types & Surfaces breakdown feature (mobile-first)" — matched Phase 27 with weak keyword-only relevance during context gathering; reviewed and left out per explicit user decision — unrelated to legacy tile-download cleanup.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|--------------------|
| CLEAN-01 | Trail-scoped tile download code is removed outright — `trail_download_service.dart` tile-download methods, `TrailEntity.pmTiles`/`demPmTiles` fields, and related UI — no dual-run, no migration path (app is pre-production) | Fully addressed. See Architecture Patterns (Pattern 1, Pattern 2), Code Examples, and the Recommended File-Level Changes list — every deletion target independently re-verified via direct file reads and fresh grep on 2026-07-24. Includes a newly-discovered dependent orphan (`map_cell.dart`) not named in CONTEXT.md's canonical refs, now folded into scope via the "Claude's Discretion" clause above. |
| CLEAN-02 | A one-time on-device cleanup sweep deletes orphaned legacy tile files left on existing dev/test installs, so the new disk-usage figure is accurate | **Explicitly out of scope per D-05 — do not plan for it.** Research independently confirms (`region_disk_usage_util.dart` full read) that the Settings disk-usage total was never polluted by legacy tile bytes to begin with (it has always summed only `regions/{id}/*.pmtiles`), reinforcing that no sweep is needed for the stated goal even setting aside the pre-production rationale. |

</phase_requirements>

## Summary

This phase is pure subtraction, not construction. Phases 22-26 already built and proved the region-based tile system end-to-end; this phase deletes the now-redundant trail-scoped tile-download machinery that Phase 25 already stopped reading from (map rendering moved to the region pipeline in Phase 25) but that `trail_download_service.dart` was still writing to. Codebase investigation independently confirms every grep claim already made in `27-CONTEXT.md`: `pmTiles`/`demPmTiles` have exactly two write sites (`trail_download_service.dart`, `TrailEntity.fromModel`'s mirror) and zero other readers; `showGenerating()` has exactly one caller (`trail_download_state_provider.dart:218`, itself gated behind `onGeneratingChanged`); `downloadTrail()` has exactly one caller (the Phase 26 guard). No separate "trail-scoped tile-download UI" file exists beyond the entity/model field wiring — `missing_coverage_sheet.dart` (also read in this research) is Phase 26's region-download UI, not a legacy remnant, and must NOT be touched by this phase.

A second, self-contained dead-code family was found during investigation and is now in scope per D-04's "Claude's Discretion" clause: `map_cell.dart` (`MapCellInfoList`, `MapCellInfo`, `MapCellStatusResponse`, `MapCellStatus`) is a freezed model file used exclusively by the three deleted `trail_download_service.dart` methods (`_downloadMapTiles`, `_fetchCellList`, `_pollUntilReady`) and has zero other importers anywhere in `lib` or `test`. It should be deleted alongside those methods, or it becomes a new orphan the moment this phase lands.

Disk-usage accuracy (originally success criterion #2, now descoped per CONTEXT D-05) is a non-issue in practice: `region_disk_usage_util.dart` (Phase 24, SETUI-05) already sums bytes exclusively from `region_file_path.dart`'s `<root>/regions/<id>/{vector,dem}.pmtiles` paths — it has never read the legacy `<root>/library/<trailId>/tiles/` directory, so the Settings disk-usage total is already accurate today and will remain so after this phase with zero additional work. This confirms the user's "no sweep necessary" rationale from a different angle than the one stated in CONTEXT.md: it's not just that there's no install base to clean, it's that the disk-usage figure was never polluted by legacy tile bytes in the first place.

**Primary recommendation:** Delete `_downloadMapTiles`/`_fetchCellList`/`_pollUntilReady` and their `tileResult`/`pmTiles`/`demPmTiles`/`isGenerating`/`onGeneratingChanged` wiring from `trail_download_service.dart`; delete `map_cell.dart` (+ generated `.freezed.dart`/`.g.dart`) as a newly-discovered dependent orphan; delete `pmTiles`/`demPmTiles` from `TrailEntity` and `Trail`, then run `dart run build_runner build --delete-conflicting-outputs` once at the end to regenerate `.g.dart`/`.freezed.dart`/`objectbox-model.json`/`objectbox.g.dart`; remove `onGeneratingChanged` wiring and the `hasSelectedPackages`-vs-`showGenerating` branch from `trail_download_state_provider.dart`; delete `DownloadNotificationService.showGenerating()` only after confirming (already done in this research — see below) that no region-download code path calls it.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Trail entity persistence (ObjectBox) | Mobile App (data layer) | — | `TrailEntity`/`Trail` freezed model; field removal is a local schema change, no server involvement |
| Trail-scoped tile download (deleted) | Mobile App (service layer) | — | Was `TrailDownloadService._downloadMapTiles`; entirely client-side orchestration against the (still-existing, untouched) `/map/cells` backend endpoint |
| Region-based tile download (unaffected) | Mobile App (service layer) | Backend (pre-built archives) | `TileRepositoryManager`/`tile_repository_provider.dart` — structurally independent of `trail_download_service.dart`, confirmed via grep (no shared state, no shared file paths, no shared entity fields) |
| Download progress notification | Mobile App (service layer) | — | `DownloadNotificationService` — local-only, no network; `showGenerating` is trail-path-specific, `showAggregateProgress`/`showProgress`/`showSuccess`/`showError` are shared and unaffected |
| Settings disk-usage total | Mobile App (util layer) | — | `region_disk_usage_util.dart` — already region-only, unaffected by this phase's deletions |

## Standard Stack

No new libraries. This phase touches only existing project dependencies:

| Library | Version (pinned in `pubspec.yaml`) | Role in this phase |
|---------|-------------------------------------|---------------------|
| `objectbox` / `objectbox_generator` | `^5.3.1` | Regenerate `objectbox-model.json`/`objectbox.g.dart` after field removal |
| `freezed`/`freezed_annotation`/`json_serializable` (via `build_runner`) | `build_runner: ^2.13.1` | Regenerate `trail.freezed.dart`/`trail.g.dart` after field removal; also removes `map_cell.freezed.dart`/`map_cell.g.dart` when the source file is deleted |
| `flutter_local_notifications` | (existing, unpinned check not needed — untouched by this phase) | `DownloadNotificationService` — one method (`showGenerating`) deleted, rest untouched |

**No package legitimacy audit needed** — no new packages are installed in this phase. `## Package Legitimacy Audit` section omitted per the protocol's own scope (audit is required only "whenever this phase installs external packages").

**Regeneration command** (verified against installed `pubspec.yaml` versions above):
```bash
cd app
dart run build_runner build --delete-conflicting-outputs
```
Run once as a single pass at the end (per CONTEXT.md's "Claude's Discretion" — this research recommends single-pass over incremental, since ObjectBox schema regeneration and freezed regeneration both want a fully-consistent source tree, and running it mid-edit risks generating stale/broken intermediate `.g.dart` files against half-deleted fields).

## Architecture Patterns

### System Architecture Diagram

```
                     BEFORE (dual-run, this phase's target state to eliminate)
┌─────────────────────────────────────────────────────────────────────┐
│  DownloadingTrailIds.download(trail)   [trail_download_state_provider]│
│         │                                                             │
│         ├──> TrailDownloadService.downloadTrail(trail, ...)          │
│         │         │                                                   │
│         │         ├──> _downloadMapTiles() ──> GET /map/cells        │
│         │         │         │                    (poll until ready)  │
│         │         │         └──> writes library/{id}/tiles/*.pmtiles │
│         │         │                            (DEAD CODE — TARGET)  │
│         │         ├──> _downloadPhotos()  ──> writes library/{id}/photos/*
│         │         ├──> _downloadWaypointPhotos() ─> writes .../waypoints/*
│         │         └──> best-effort Valhalla nav-cache write          │
│         │                                                             │
│         └──> tileRepoNotifier.downloadVector/downloadDem(region.id)  │
│                   │                                                   │
│                   └──> writes regions/{id}/{vector,dem}.pmtiles      │
│                        (INDEPENDENT — region system, untouched)      │
└─────────────────────────────────────────────────────────────────────┘

                     AFTER (this phase's end state)
┌─────────────────────────────────────────────────────────────────────┐
│  DownloadingTrailIds.download(trail)   [trail_download_state_provider]│
│         │                                                             │
│         ├──> TrailDownloadService.downloadTrail(trail, ...)          │
│         │         ├──> _downloadPhotos()  ──> writes library/{id}/photos/*
│         │         ├──> _downloadWaypointPhotos() ─> writes .../waypoints/*
│         │         └──> best-effort Valhalla nav-cache write          │
│         │              (no onGeneratingChanged param; no tile I/O)   │
│         │                                                             │
│         └──> tileRepoNotifier.downloadVector/downloadDem(region.id)  │
│                   └──> writes regions/{id}/{vector,dem}.pmtiles      │
│                        (UNCHANGED)                                    │
│                                                                        │
│  Map rendering (trail_map.dart, navigation_screen.dart, Phase 25.1)  │
│         └──> reads exclusively from regions/{id}/*.pmtiles via the   │
│              local loopback tile proxy — never touched library/{id}/ │
│              tiles/, even before this phase (Phase 25 migration)     │
└─────────────────────────────────────────────────────────────────────┘
```

### Recommended File-Level Changes

```
app/lib/services/trail_download_service.dart      # delete 3 methods + wiring
app/lib/services/download_notification_service.dart # delete showGenerating() (conditional, see D-07)
app/lib/models/map_cell.dart                       # DELETE (newly discovered orphan, see Summary)
app/lib/models/map_cell.freezed.dart               # auto-deleted by build_runner
app/lib/models/map_cell.g.dart                     # auto-deleted by build_runner
app/lib/entities/trail_entity.dart                 # delete pmTiles/demPmTiles fields
app/lib/models/trail.dart                          # delete pmTiles/demPmTiles freezed fields
app/lib/provider/trail/trail_download_state_provider.dart # delete onGeneratingChanged wiring + hasSelectedPackages branch
app/lib/objectbox-model.json                       # regenerated (build_runner)
app/lib/objectbox.g.dart                            # regenerated (build_runner)
app/lib/models/trail.freezed.dart, trail.g.dart     # regenerated (build_runner)
```

### Pattern 1: Surgical method deletion inside a still-live class
**What:** `TrailDownloadService` keeps existing (photos/waypoint-photos/nav-cache still need it) — only 3 of its methods and their call-site wiring are removed, not the whole file.
**When to use:** Whenever a class has a mix of methods, some dead and some live, and the live ones share private helpers (`_downloadTracked`) with the dead ones.
**Example (present in the file today, confirmed by direct read):**
```dart
// Source: app/lib/services/trail_download_service.dart:119-172 (current state)
// DELETE this variable + its only writer (_downloadMapTiles call in the futures list)
(List<String>, List<String>)? tileResult;
// ...
final (cellPaths, demCellPaths) = tileResult!;   // DELETE
entity.pmTiles = cellPaths;                       // DELETE
entity.demPmTiles = demCellPaths;                 // DELETE

// KEEP UNCHANGED — _downloadTracked is still needed by _downloadPhotos below,
// per CONTEXT.md D-04
Future<void> _downloadTracked(
  String url,
  String savePath, {
  required CancelToken? cancelToken,
  required void Function(double fraction) onFraction,
}) async { /* ... unchanged ... */ }
```

### Pattern 2: Conditional deletion gated on a fresh grep, not an assumption
**What:** `showGenerating()` is deleted only if it truly has zero remaining callers after the trail-path call site is removed.
**When to use:** Whenever CONTEXT.md flags a "delete if truly dead" condition (D-07) rather than an unconditional deletion.
**Verification already run in this research session** (result: safe to delete outright):
```bash
$ grep -rn "showGenerating" app/lib --include="*.dart"
app/lib/provider/trail/trail_download_state_provider.dart:212  # doc comment only
app/lib/provider/trail/trail_download_state_provider.dart:218  # the ONLY real call site
app/lib/services/download_notification_service.dart:40         # the method definition itself
```
No hit in `tile_repository_provider.dart`, `tile_repository_manager.dart`, or anywhere else in `lib`. `showGenerating()` is safe to delete outright once its one caller (line 218) is removed as part of the `onGeneratingChanged` wiring deletion. The executor should re-run this exact grep after editing `trail_download_state_provider.dart` (not before) to catch the post-edit zero-remaining-callers state, per D-07's letter.

### Anti-Patterns to Avoid
- **Deleting `_downloadTracked` alongside the tile methods:** It's the shared byte-progress helper `_downloadPhotos` still needs (D-04). Deleting it breaks photo downloads.
- **Deleting `map_cell.dart` without also removing its import** from `trail_download_service.dart` (`import 'package:wanderer/models/map_cell.dart';` at line 12) — leaving a dangling import after deleting the file will fail `flutter analyze`/compilation immediately, which is a fast, safe-to-ignore signal (not a real risk, just noting the expected mechanical step).
- **Touching `missing_coverage_sheet.dart`:** It is Phase 26's region-download UI (confirmed by direct read: imports `tile_repository_provider.dart`, `region_entity.dart`, has zero references to `pmTiles`/`downloadTrail`/`MapCellInfoList`). It is NOT the "trail-scoped tile-download UI" CONTEXT.md's phase description gestures at — that UI never existed as a separate file; it was only the entity-field wiring inside `trail_download_service.dart` itself. Do not delete or modify this file in this phase.
- **Assuming ObjectBox migration is needed:** Per CLAUDE.md's "no migration path, pre-production app" instruction and CONTEXT.md D-06, a plain field removal + `build_runner build --delete-conflicting-outputs` is sufficient — ObjectBox tracks property IDs via UIDs in `objectbox-model.json`, and removing a property from the entity class is a supported, non-breaking regeneration for a pre-production app with no real installed-base data migration concern.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|--------------|-----|
| Confirming zero remaining references before deletion | Manual code reading / memory of what CONTEXT.md claimed | `grep -rn "<symbol>" app/lib app/test --include="*.dart"` re-run fresh, post-edit where order matters (D-07) | CONTEXT.md's claims were verified independently in this research session and found accurate — but the planner/executor should still re-run greps themselves before each deletion, since D-06/D-07 both explicitly require it "not assumed" |
| Regenerating ObjectBox/freezed code | Hand-editing `.g.dart`/`.freezed.dart`/`objectbox-model.json`/`objectbox.g.dart` | `dart run build_runner build --delete-conflicting-outputs` | These are generated files; hand-editing them drifts from the source-of-truth annotations and will be silently overwritten or cause generator conflicts on the next real build |

**Key insight:** This phase has no genuine "don't hand-roll" risk in the traditional sense (no algorithm to reinvent) — the only real risk is under-verifying deletions (deleting something still used) or over-verifying (leaving something dead because a stale grep result was trusted instead of re-run).

## Common Pitfalls

### Pitfall 1: Deleting `map_cell.dart` without noticing it's the phase's actual scope boundary
**What goes wrong:** CONTEXT.md's `<code_context>` and `<canonical_refs>` sections don't mention `map_cell.dart` by name — the discussion focused on `pmTiles`/`demPmTiles`/`showGenerating`. A planner working strictly from the CONTEXT.md file list alone could leave `map_cell.dart` in place, creating a new orphan file immediately after this phase ships.
**Why it happens:** `map_cell.dart` wasn't grepped for during the `/gsd-discuss-phase` session; CONTEXT.md's own discretion clause anticipates this ("Any other genuinely-dead tile-generation-only strings/assets... researcher/planner should do a final sweep").
**How to avoid:** Delete `map_cell.dart` (+ its two generated siblings) as part of the same task/commit that deletes `_downloadMapTiles`/`_fetchCellList`/`_pollUntilReady`, and remove the now-unused `import 'package:wanderer/models/map_cell.dart';` from `trail_download_service.dart`.
**Warning signs:** `flutter analyze` reporting `unused_import` on `map_cell.dart`, or `dart run build_runner build` reporting build errors for `map_cell.freezed.dart`/`map_cell.g.dart` referencing a class no longer imported anywhere.

### Pitfall 2: Regenerating build_runner output before all source edits are complete
**What goes wrong:** Running `build_runner build` mid-edit (e.g., after deleting `TrailEntity.pmTiles` but before deleting `Trail.pmTiles`) can produce a transient state where `TrailEntity.fromModel`/`toModel()` reference a field that no longer exists on one side, causing confusing generator or analyzer errors that look like a real bug.
**Why it happens:** `TrailEntity` and `Trail` are two independent freezed/ObjectBox-annotated classes bridged by hand-written `fromModel`/`toModel()` extension methods (not code-generated) — the bridge code needs both sides edited together before regeneration.
**How to avoid:** Edit `TrailEntity` (entity), `Trail` (freezed model), and the `TrailEntityMapping` extension's `pmTiles`/`demPmTiles` references in `trail_entity.dart` all together, then run `build_runner` once (CONTEXT.md's discretion note explicitly permits either single-pass or incremental — this research recommends single-pass for this exact reason).
**Warning signs:** `dart run build_runner build` reporting "The non-abstract class '_Trail' is missing implementations" or similar freezed constructor-mismatch errors.

### Pitfall 3: The Phase 26 guard's `onGeneratingChanged` removal changing observable notification behavior for the 0-region path
**What goes wrong:** `trail_download_state_provider.dart`'s `onGeneratingChanged` callback currently branches on `hasSelectedPackages`: when packages are selected it calls `updateAggregate()`; when none are selected, it calls `notificationService.showGenerating(trail.name)`. Removing this callback entirely (D-02/D-03) means the 0-region path's UI experience during what *used to be* the tile-generation wait window simply has no "Generating map tiles..." interstitial notification anymore — there's no replacement state, because (per D-02) there IS no more generation phase to report on. This is intentional (not a gap), but is worth flagging because a superficial read might expect a 1:1 replacement notification state.
**Why it happens:** The whole reason `onGeneratingChanged`/`isGenerating` existed was to report the server-side tile-generation wait — once tile generation itself is gone from the client, there is nothing left in that time window to report; the download simply goes straight from "started" to "downloading photos" (a state that already has its own progress via `onProgress`).
**How to avoid:** Confirm during planning/execution that `showProgress`'s existing indeterminate handling (`indeterminate: total == 0` in `download_notification_service.dart`) already covers the brief window before `onProgress`'s first real callback fires — no new notification state needs to be built.
**Warning signs:** None expected; flagging for planner awareness only, not a functional risk.

### Pitfall 4: Trusting `27-CONTEXT.md`'s grep claims without independent re-verification
**What goes wrong:** CONTEXT.md states "Confirmed via grep: zero remaining readers" for `pmTiles`/`demPmTiles` and "confirmed via grep" for `showGenerating`'s caller count. If the executor treats these as immutable fact rather than a snapshot-in-time, a change made between context-gathering (2026-07-24) and execution could silently invalidate the claim.
**Why it happens:** Context and research are gathered before planning/execution; the codebase is mutable in between.
**How to avoid:** This research independently re-ran every grep CONTEXT.md's claims rest on (`pmTiles`, `demPmTiles`, `showGenerating`, `onGeneratingChanged`, `downloadTrail(`) as of 2026-07-24 and found them all still accurate. The planner should still instruct the executor to re-run these greps immediately before each deletion, not rely on either CONTEXT.md's or this RESEARCH.md's cached results, since D-06/D-07 explicitly require verification to be "done, not assumed."
**Warning signs:** Any grep hit outside the files already named in this research (`trail_download_service.dart`, `trail_entity.dart`, `trail.dart`, `trail_download_state_provider.dart`, `download_notification_service.dart`, `map_cell.dart` + its generated siblings).

## Code Examples

### Current `onGeneratingChanged` wiring to be removed (trail_download_state_provider.dart:206-231)
```dart
// Source: app/lib/provider/trail/trail_download_state_provider.dart (current state, read 2026-07-24)
await trailDownloadService.downloadTrail(
  trail,
  onGeneratingChanged: (isGenerating) {
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
  onProgress: (done, total) {
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
After removal, `downloadTrail()`'s call site keeps only the `onProgress` parameter — the `onGeneratingChanged` param and its entire callback body are deleted, and `hasSelectedPackages`'s own definition (line 118-119, `WR-01`) and every reference tied ONLY to the `onGeneratingChanged` branch should be checked for continued use (it's also read at lines 200-201 for the `updateAggregate()`-vs-`showProgress` branch on `downloadTrail`'s START, which is unrelated to `onGeneratingChanged` and must be KEPT — only the `onGeneratingChanged`-specific branch at line 209-220 is deletion scope).

### Legacy tile-file naming convention vs. region-file naming convention (for planner awareness — no sweep is being built, per D-05)
```dart
// Legacy (deleted by this phase, files NOT swept per D-05):
// <appDocs>/library/<trailId>/tiles/<cellKey>.pmtiles
// <appDocs>/library/<trailId>/tiles/<cellKey>_dem.pmtiles
// Source: app/lib/services/trail_download_service.dart:234, :264-265 (current state)

// Region system (unaffected, current naming convention):
// <appDocs>/regions/<regionId>/vector.pmtiles
// <appDocs>/regions/<regionId>/dem.pmtiles  (+ `.part` while downloading)
// Source: app/lib/util/region_file_path.dart:40-56 (current state)
```
These naming conventions never collide (`library/` vs `regions/` are disjoint top-level directories under app documents), so no accidental cross-deletion risk exists even if a future phase does add a sweep.

### Trail deletion already cleans up the legacy directory for actively-managed trails
```dart
// Source: app/lib/provider/trail/trail_library_provider.dart:22-40 (current state)
Future<void> deleteTrail(String id) async {
  // ...
  box.remove(entity.obxId);

  final appDir = await getApplicationDocumentsDirectory();
  final trailDir = Directory('${appDir.path}/library/$id');
  if (await trailDir.exists()) {
    await trailDir.delete(recursive: true);   // deletes tiles/, photos/, waypoints/ together
  }
  // ...
}
```
This confirms the only files left behind after this phase are `library/<trailId>/tiles/` subfolders belonging to trails that were downloaded under the OLD system and never subsequently deleted by the user — exactly the "existing dev/test installs" scenario D-05 explicitly declines to build a sweep for.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|-------------------|---------------|--------|
| Per-trail, per-cell PMTiles download (`GET /map/cells`, poll-until-ready, `library/{id}/tiles/`) | App-wide, pre-built, region-scoped PMTiles archives (`regions/{id}/{vector,dem}.pmtiles`) | Phases 21.5-26 (this v1.6 milestone) | Trail download no longer waits on server-side tile generation; map rendering works anywhere within a downloaded region, not just within a specific trail's cells (already live since Phase 25/25.1) |
| Server-side tile-generation wait reported via `onGeneratingChanged`/`isGenerating` | No generation wait exists client-side after this phase — region archives are pre-built by a cronjob (BACK-02/03) before any user request | This phase (27) | Simpler `downloadTrail()` signature; one less notification state to maintain |

**Deprecated/outdated:**
- `TrailEntity.pmTiles`/`demPmTiles`, `Trail.pmTiles`/`demPmTiles`: superseded by region-based rendering (Phase 25); this phase performs their actual removal.
- `map_cell.dart` (`MapCellInfoList`/`MapCellInfo`/`MapCellStatusResponse`/`MapCellStatus`): superseded the same way, newly identified in this research as needing removal alongside the above.
- `DownloadNotificationService.showGenerating()`: superseded by the fact that there is no more generation phase to report (region archives are pre-built server-side, not generated on-demand per trail).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|----------------|
| A1 | No CI workflow currently runs `flutter analyze`/`flutter test` for the `app/` directory (none found under `.github/workflows/`) | Common Pitfalls / general context | Low — if a CI workflow does exist elsewhere, the planner should still run these commands locally regardless; this doesn't change what needs to be done, only whether CI additionally gates it |

**All other claims in this research were independently verified via direct file reads and fresh `grep`/`python3 json` inspection of the actual codebase during this research session (2026-07-24)** — not from training data or CONTEXT.md's claims alone. No package-name or library-capability claims required verification since no new packages are introduced.

## Open Questions

1. **Should the `_MissingCoverageSheetContentState` widget's two "Downloading…" `Text` widgets (currently reformatted by an uncommitted dart-format-only diff) be left alone by this phase?**
   - What we know: `git diff -- app/lib/components/trail/missing_coverage_sheet.dart` shows only whitespace/line-wrap changes (no logic changes), and this file belongs to Phase 26's region-download UI, entirely out of this phase's scope.
   - What's unclear: Whether the uncommitted local edit should be committed as part of this phase's work or handled separately (e.g., a stray `dart format` run left uncommitted from Phase 26).
   - Recommendation: Leave the uncommitted diff as-is; it's unrelated to CLEAN-01 and outside this phase's `files_modified` scope. Flag to the user that `app/lib/components/trail/missing_coverage_sheet.dart` and `app/lib/services/trail_download_service.dart` both have pre-existing uncommitted whitespace-only changes from before this phase started, visible in `git status` — the planner/executor should decide whether to preserve, discard, or fold them into this phase's first commit (folding into `trail_download_service.dart`'s first edit is natural since that file is being edited anyway; `missing_coverage_sheet.dart`'s stray diff is unrelated and should probably just be committed separately or left as pre-existing untracked formatting noise, not blocked on).

## Environment Availability

Skipped — this phase has no external service/tool dependencies beyond the already-installed Flutter/Dart toolchain and existing project dependencies (`objectbox_generator`, `build_runner`, `freezed`), all already present in `pubspec.yaml` and used successfully throughout Phases 22-26. No new environment probing needed.

## Validation Architecture

Skipped — `.planning/config.json`'s `workflow.nyquist_validation` is explicitly `false`.

## Security Domain

`security_enforcement: true` per `.planning/config.json` (ASVS Level 1, block on `high`). This phase is code deletion only — it removes attack surface (fewer network calls to `/map/cells`, fewer local file-write paths) and adds none. No new input validation, authentication, session, or cryptography concerns apply.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|----------------|---------|--------------------|
| V2 Authentication | No | No auth logic touched — `_api` (Dio client) auth wiring is unaffected, untouched by this phase |
| V3 Session Management | No | Not applicable |
| V4 Access Control | No | Not applicable — no permission checks added or removed |
| V5 Input Validation | No (net reduction) | The deleted `_fetchCellList`/`_pollUntilReady` methods parsed server responses (`MapCellStatusResponse.fromJson`) — their removal is a net reduction in externally-parsed-data surface, not an addition |
| V6 Cryptography | No | Not applicable |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|------------------------|
| Path traversal via unsanitized directory names | Tampering | Not newly relevant here — `library/{trailId}/` paths use trail IDs sourced from the authenticated backend response (unchanged, pre-existing, out of this phase's scope); this phase does not add any new path-building logic, and the surviving `region_file_path.dart` builders already enforce `assertValidRegionId` (pre-existing, Phase 23, unaffected) |

## Sources

### Primary (HIGH confidence — direct codebase reads, 2026-07-24)
- `app/lib/services/trail_download_service.dart` — full file read, confirms exact deletion boundaries
- `app/lib/services/download_notification_service.dart` — full file read, confirms `showGenerating` is a distinct, isolatable method
- `app/lib/provider/trail/trail_download_state_provider.dart` — full file read, confirms `onGeneratingChanged`/`hasSelectedPackages` wiring shape
- `app/lib/entities/trail_entity.dart` — full file read, confirms `pmTiles`/`demPmTiles` field + mapping locations
- `app/lib/models/trail.dart` — partial read (lines 1-140), confirms freezed field locations
- `app/lib/components/trail/missing_coverage_sheet.dart` — full file read, confirms this is Phase 26 region UI, not legacy trail-tile UI
- `app/lib/provider/trail/trail_library_provider.dart` — full file read, confirms trail deletion already cleans up `library/{id}/` recursively
- `app/lib/util/region_disk_usage_util.dart` — full file read, confirms disk-usage total is already region-only
- `app/lib/util/region_file_path.dart` — full file read, confirms region file-naming convention
- `app/lib/models/map_cell.dart` — full file read, confirms this file's sole purpose is supporting the deleted tile-download methods
- `app/lib/objectbox-model.json` — inspected via `python3 -c "import json..."`, confirms `pmTiles`/`demPmTiles` property IDs exist and need regeneration
- `app/pubspec.yaml` — confirms `build_runner`/`objectbox`/`objectbox_generator` versions
- `.planning/config.json` — confirms `nyquist_validation: false`, `security_enforcement: true`

### Secondary (MEDIUM confidence)
- None — no external documentation lookups were needed for this phase (pure internal code deletion, no new library APIs)

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new libraries; only regeneration commands for already-installed, already-used tooling
- Architecture: HIGH — every deletion boundary independently re-verified via direct file reads and fresh grep, not assumed from CONTEXT.md
- Pitfalls: HIGH — all four pitfalls derived from direct code inspection of the actual current file states (including the two files with pre-existing uncommitted edits), not speculative

**Research date:** 2026-07-24
**Valid until:** Codebase-dependent, not time-dependent — valid as long as no other work lands on `trail_download_service.dart`, `trail_entity.dart`, `trail.dart`, `trail_download_state_provider.dart`, or `download_notification_service.dart` before this phase executes. Given `mode: yolo` and sequential phase execution per STATE.md, re-run the key greps (see Pitfall 4) immediately before executing, not before planning.
