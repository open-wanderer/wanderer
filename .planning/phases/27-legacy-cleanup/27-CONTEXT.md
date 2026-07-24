# Phase 27: Legacy Cleanup - Context

**Gathered:** 2026-07-24
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase deletes the old trail-scoped tile-download system outright now that Phase 26 has proven the region-based system end-to-end. In scope: removing `trail_download_service.dart`'s tile-download methods and their entity-field wiring (`TrailEntity.pmTiles`/`demPmTiles`), removing the notification-service copy that only existed to report tile generation, and adapting the Phase 26 guard's notification wiring to the simplified `downloadTrail()` signature. Out of scope (descoped during this discussion, see Decisions): the "one-time cleanup sweep" for orphaned legacy tile files (CLEAN-02) and the ROADMAP's success-criterion-2 disk-usage-figure claim tied to it — the app is pre-production, so there is no real-world install base to clean up, and this is being cut rather than built. Also out of scope: the region tile system itself (unchanged, Phases 22-26), glyph/sprite caching for offline label rendering (`glyph_sprite_cache_provider.dart` — a separate, still-active system, not "trail-scoped tile download"), and photo/waypoint-photo/GPX-nav-cache downloading (these stay in `downloadTrail()`, only the tile-specific pieces are removed).

</domain>

<decisions>
## Implementation Decisions

### downloadTrail() Surgery Scope
- **D-01:** `downloadTrail()` remains ONE method (not split). Delete `_downloadMapTiles`, `_fetchCellList`, `_pollUntilReady`, and all `tileResult`/`pmTiles`/`demPmTiles` wiring from `trail_download_service.dart`. Keep photo downloads, waypoint-photo downloads, and the best-effort Valhalla nav-cache write exactly as they are — those aren't "tile-download methods" per CLEAN-01's scope.
- **D-02:** `onGeneratingChanged` is removed entirely from `downloadTrail()`'s signature (and from `_downloadPhotos`/`_downloadWaypointPhotos`, which never used it anyway). There is no more server-side "tile generation" wait once tiles are gone — only real, trackable photo-download progress remains, so `isGenerating`/`handleGeneratingChanged` in `trail_download_service.dart` go away too.
- **D-03:** In `trail_download_state_provider.dart` (Phase 26's guard, already UAT-passed), remove the `onGeneratingChanged` callback wiring and the WR-01 `hasSelectedPackages` branch that reacted to it — there's nothing left to report as "generating." The aggregate notification shows real download progress only (trail photo/nav-cache progress + selected region package progress via the existing `vectorLatched`/`demLatched` monotonic latch from 26-05).
- **D-04:** `_downloadTracked` (the shared byte-progress-with-fake-fallback helper) stays unchanged — it's still needed by `_downloadPhotos` after tile removal. Do not delete it alongside the tile-specific methods.

### Legacy Sweep — Descoped
- **D-05 (scope cut, not a gray-area decision):** CLEAN-02 ("one-time cleanup sweep deletes orphaned legacy tile files... disk-usage figure is accurate") and ROADMAP.md's Phase 27 success criterion #2 are dropped from this phase entirely, per explicit user decision. Rationale given: the app is pre-production, so there's no real install base with orphaned legacy tile files to clean up — any dev/test device carrying old `library/{trailId}/tiles/` directories can just be wiped/reinstalled manually. This phase is CLEAN-01 (code removal) only. **Downstream note:** `.planning/REQUIREMENTS.md` (CLEAN-02, currently `Pending`, mapped to Phase 27) and `.planning/ROADMAP.md`'s Phase 27 success criterion #2 should be updated to reflect this cut in a future editing pass — the planner should NOT create a plan for CLEAN-02, and should flag CLEAN-02 as intentionally unaddressed by this phase rather than treating its absence as a gap.

### TrailEntity Field Removal Mechanics
- **D-06:** Delete `pmTiles`/`demPmTiles` fields outright now from `TrailEntity` (ObjectBox entity), the `Trail` freezed model, and regenerate `.g.dart`/`.freezed.dart`/`objectbox-model.json`. Confirmed via grep: zero remaining readers anywhere in `app/lib` outside the entity/model definitions themselves and `trail_download_service.dart` (Phase 25 already moved map rendering to the region-based pipeline). Matches CLAUDE.md's "no migration path, pre-production app" instruction — no ObjectBox migration step needed for a field removal.

### Trail-Scoped Tile-Download UI Remnants
- **D-07:** Delete `download_notification_service.dart`'s `showGenerating()` method and its `'Generating map tiles...'` string outright, since D-02/D-03 make it unreachable from the trail-download path. **Before deleting, the executor MUST grep for other callers** (e.g. region downloads in `tile_repository_provider.dart`/`tile_repository_manager.dart`) — if region downloads also call `showGenerating()`, keep the method and only remove the trail-path call site; the user's intent is "delete if truly dead," not "delete unconditionally."

### Claude's Discretion
- Exact grep/verification approach for confirming zero remaining references before deleting `pmTiles`/`demPmTiles` and `showGenerating()` — implementation detail, but MUST be done (not assumed) per D-06/D-07's conditions.
- Whether to regenerate ObjectBox codegen (`build_runner`) as a single pass at the end or incrementally per file — implementation detail.
- Any other genuinely-dead tile-generation-only strings/assets beyond `showGenerating()`'s copy that a full-file read of `trail_download_service.dart`/`download_notification_service.dart` surfaces — researcher/planner should do a final sweep of both files for anything tile-generation-specific this discussion didn't explicitly enumerate.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements & Roadmap
- `.planning/ROADMAP.md` — Phase 27 section (goal, 3 success criteria — note success criterion #2 is descoped per D-05, only #1 and #3 apply)
- `.planning/REQUIREMENTS.md` — CLEAN-01 (in scope), CLEAN-02 (explicitly out of scope per D-05, do not plan for it)

### Tile-Download Removal Target
- `app/lib/services/trail_download_service.dart` — `downloadTrail()`, `_downloadMapTiles`/`_fetchCellList`/`_pollUntilReady` (delete), `_downloadTracked`/`_downloadPhotos`/`_downloadWaypointPhotos` (keep unchanged)
- `app/lib/entities/trail_entity.dart` — `pmTiles`/`demPmTiles` fields (delete, D-06)
- `app/lib/models/trail.dart` — `pmTiles`/`demPmTiles` freezed fields (delete, D-06)
- `app/lib/services/download_notification_service.dart` — `showGenerating()` (delete conditionally, D-07)

### Phase 26 Guard (ripple target)
- `app/lib/provider/trail/trail_download_state_provider.dart` — the single caller of `downloadTrail()`; remove `onGeneratingChanged` wiring and the WR-01 `hasSelectedPackages` branch (D-03). This file was just fixed and UAT-passed in Phase 26 (26-04/26-05) — treat it carefully, re-run `flutter test`/`flutter analyze` and confirm no regression to the 26-04/26-05 invariants (single `remove(trail.id)`, single `invalidate(regionListNotifierProvider)`, `vectorLatched`/`demLatched` monotonic latch, `trailSucceeded`-gated `showSuccess`).

### Prior Phase Context (design lineage)
- `.planning/phases/26-trail-download-guard/26-CONTEXT.md` — the guard's design, including its explicit note that "legacy trail-tile removal (Phase 27)" was out of scope for Phase 26 and the trail's own tile download still ran there — this phase is that deferred removal.
- `.planning/phases/26-trail-download-guard/26-05-SUMMARY.md` — the `vectorLatched`/`demLatched`/`trailSucceeded` mechanism this phase's guard changes must not regress.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- None new — this phase removes code, it doesn't add reusable assets.

### Established Patterns
- `_downloadTracked` is a shared byte-progress helper already used by both tile and photo downloads — after tile removal it remains the photo-download progress mechanism, unchanged.
- The region tile system (`tile_repository_provider.dart`, `tile_repository_manager.dart`) is completely independent of `trail_download_service.dart` — confirmed no shared state beyond the guard's read-only coverage check (Phase 26), so this phase's changes cannot affect region downloads.

### Integration Points
- `downloadTrail()`'s only caller in the whole codebase is `trail_download_state_provider.dart:207` (the Phase 26 guard) — confirmed via grep. No other call site needs updating.
- `pmTiles`/`demPmTiles` have zero readers outside their own definitions and `trail_download_service.dart` — confirmed via grep, safe to delete outright.

</code_context>

<specifics>
## Specific Ideas

- "No sweep necessary. App is not in production" — the user's explicit rationale for cutting CLEAN-02, verbatim.

</specifics>

<deferred>
## Deferred Ideas

- CLEAN-02 (one-time legacy-file cleanup sweep) and ROADMAP Phase 27 success criterion #2 — cut from this phase per D-05, not deferred to a later phase; the user's position is that it's unnecessary given the pre-production status, not merely "not now."

### Reviewed Todos (not folded)
- "Way Types & Surfaces breakdown feature (mobile-first)" — matched Phase 27 with weak keyword-only relevance (score 0.6, generic terms "first, plans, app, trail"); reviewed and left out per explicit user decision — unrelated to legacy tile-download cleanup.

</deferred>

---

*Phase: 27-legacy-cleanup*
*Context gathered: 2026-07-24*
