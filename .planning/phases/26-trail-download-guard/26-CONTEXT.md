# Phase 26: Trail Download Guard - Context

**Gathered:** 2026-07-24
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase inserts a **coverage guard** in front of the existing trail-download flow. Before a trail's tiles download — via the single shared entry point `DownloadingTrailIds.download(trail)` in `app/lib/provider/trail/trail_download_state_provider.dart`, used by both the detail-screen button (`trail_detail_screen.dart:130`) and the dropdown item (`trail_dropdown.dart:107`) — the app checks the trail's bbox (`minLat`/`minLon`/`maxLat`/`maxLon`) against locally-known regions. When every overlapping region is already `downloaded`/`updateAvailable`, the download proceeds unchanged. When coverage is missing, a **bottom modal sheet** lets the user select which missing region packages to download and starts them **together with the trail** in one action.

In scope: the coverage-check algorithm, the missing-coverage bottom sheet (per-region vector/DEM checkboxes, sizes, single Download button), starting selected region downloads + the trail in parallel, and a unified background download notification. Out of scope: any change to the region download engine (`TileRepositoryManager`/`TileRepositoryStatus` consumed as-is), the Settings Offline Maps screen (Phase 24), map rendering (Phase 25/25.1), and legacy trail-tile removal (Phase 27 — the trail's own tile download still runs here).

</domain>

<decisions>
## Implementation Decisions

### Coverage Algorithm
- **D-01:** Coverage is **overlap-based**, not union-containment. Compute the set of catalog regions whose bbox intersects the trail's bbox; the trail is "covered" iff **all** of those overlapping regions have local status `downloaded` OR `updateAvailable`. The sheet names exactly the overlapping regions that are NOT in one of those two states.
- **D-02:** `updateAvailable` satisfies coverage identically to `downloaded` (GUARD-04) — the guard never re-fires for a merely-stale region. Only `notDownloaded`/`downloading`/`error` (and any region not yet locally present) count as "missing." (Note: `RegionStatus` has no `paused` value — it was removed in Phase 23/24; do not test for it.)
- **D-03:** Overlap uses **raw bbox-vs-bbox intersection** — no polyline-aware test, no minimum-overlap-area threshold. A trail whose rectangular bbox clips the far corner of a large neighboring region may occasionally over-name that neighbor; accepted because the worst case is one extra *optional* region offered, which the user can leave unchecked (D-06/D-07 make regions opt-in and GUARD-03 lets the trail proceed regardless). Deterministic and simple beats precise here.
- **D-04:** When part (or all) of the trail's bbox falls inside **no catalog region at all** (the instance offers no region there): **proceed with a non-blocking warning**. If every region that *does* overlap is downloaded, let the trail download start; if there's a genuine no-region gap, surface a non-blocking notice ("part of this trail isn't covered by any offered region") but still allow the download. Never a hard block, never a generic dead-end message (GUARD-02/GUARD-03).

### Missing-Coverage Bottom Sheet (GUARD-02 / GUARD-03)
- **D-05:** The guard surfaces as a **bottom modal sheet** (not an `AlertDialog`), shown only when coverage is missing per D-01. It lists every missing overlapping region as a row, styled like the Settings Offline Maps/Regions screen (sizes in the row subtitle).
- **D-06:** Each region row has **two independent checkboxes: Vector data and DEM data**, each with its own size shown. Checking a box only sets selection state — it does **not** start a download. This makes region-package scope per-package and user-chosen (resolves the "vector-only vs vector+DEM" question: neither is fixed; the user picks).
- **D-07:** **Default checkbox state: Vector on, DEM off.** Vector is the basemap coverage the guard exists to ensure, so it's pre-checked; DEM (elevation/3D) is optional and opt-in, mirroring the Settings DEM toggle convention.
- **D-08:** A single primary **Download** button at the bottom of the sheet starts **all checked region packages plus the trail**, together, in one tap. Selecting zero regions and tapping Download starts **trail-only** — this is the GUARD-03 "proceed without full coverage" escape hatch; there is no separate "download anyway" button, the same button serves both.
- **D-09:** On tapping Download, the sheet **dismisses immediately** and all downloads (trail + each selected region package) run in the **background** via their existing engines (`trailDownloadService.downloadTrail` and the region `TileRepositoryStatus` notifier). The guard does not own long-running progress UI, cancel, or a persistent sheet.
- **D-10:** Progress is surfaced through a **single unified download notification** whose progress bar aggregates the trail download and all selected region packages into one combined bar — not one notification per download. (Researcher/planner to determine the aggregation approach across the trail service's `onProgress(done,total)` and the region engine's per-package progress; today `downloadNotificationServiceProvider.showProgress` already drives the trail notification.)

### Catalog Freshness
- **D-11:** The coverage check runs against the **locally-stored region catalog only** — no network fetch of `/api/v1/regions` on the download tap. The Settings screen already refreshes the catalog on open (Phase 24), so the local catalog is normally current, and a local-only check keeps the guard instant and works offline (e.g. a hiker at a trailhead with no signal).

### Claude's Discretion
- Where exactly the coverage check + sheet is triggered (inside `DownloadingTrailIds.download` vs. at the two call sites before invoking it) — pick whichever cleanly surfaces a `BuildContext`/modal from the shared entry point without duplicating the check across both call sites. The keepAlive notifier `download()` is the single shared entry point today; keep it single.
- The exact bbox-intersection helper (new util vs inline) and where the missing-region set is computed (a provider vs the sheet's own build) — implementation detail.
- Visual specifics of the sheet beyond "Settings-region-screen-style rows with sizes" and the unified notification's exact copy/aggregation math.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements & Roadmap
- `.planning/ROADMAP.md` — Phase 26 section (goal, 4 success criteria, "Depends on: Phase 25")
- `.planning/REQUIREMENTS.md` — GUARD-01 through GUARD-04 (all `Pending`)

### Trail Download Flow (the guard's insertion point)
- `app/lib/provider/trail/trail_download_state_provider.dart` — `DownloadingTrailIds.download(trail)`, the single shared keepAlive entry point both call sites use; the coverage check gates this. Note its existing toast + `downloadNotificationServiceProvider.showProgress` usage (relevant to D-10's unified notification).
- `app/lib/routes/trail_detail_screen.dart` (~line 130) — detail-screen download button call site
- `app/lib/components/trail/trail_dropdown.dart` (~line 107) — dropdown download-action call site
- `app/lib/services/trail_download_service.dart` — `downloadTrail(...)` with `onProgress(done,total)`/`onGeneratingChanged` callbacks; still runs in this phase (legacy tile removal is Phase 27)
- `app/lib/provider/download_notification_provider.dart` — `downloadNotificationServiceProvider` (`showProgress`/`showGenerating`/`showSuccess`/`showError`) — basis for D-10's unified progress notification

### Region Data Model & Coverage Inputs
- `app/lib/entities/region_entity.dart` — bbox stored as four discrete fields `minLat`/`minLon`/`maxLat`/`maxLon` (same order as backend `[minLon,minLat,maxLon,maxLat]`); `status` computed getter (D-12 from Phase 22); `vectorPackage`/`demPackage` `ToOne` relations
- `app/lib/models/region_status.dart` — `RegionStatus` values `notDownloaded(0)`/`downloading(1)`/`downloaded(2)`/`updateAvailable(3)`/`error(2 code)` — **no `paused`** (removed); explicit `.code` ints, never `.index`
- `app/lib/models/trail.dart` / `app/lib/entities/trail_entity.dart` — trail `minLat`/`minLon`/`maxLat`/`maxLon` fields (the coverage-check input)

### Region Download Engine (consumed as-is for the parallel region downloads)
- `app/lib/provider/region/tile_repository_provider.dart` — `TileRepositoryStatus` notifier (`downloadVector`, `downloadDem`, ...) — the sheet's Download button invokes these for each checked package
- `app/lib/provider/region/region_provider.dart` — `RegionRepository` (locally-stored catalog the guard reads for D-11; do NOT call `refreshCatalog()` from the guard)
- `app/lib/models/region_download_state.dart` — `RegionDownloadState` (`vectorProgress`/`demProgress`) — a progress source for D-10's aggregation

### Prior Phase Context (design lineage)
- `.planning/phases/24-settings-offline-maps-regions-ui/24-CONTEXT.md` — Settings region-row styling (sizes in subtitle), DEM-as-opt-in convention (D-07 mirrors this), `updateAvailable` treatment; the sheet's rows should visually echo this screen
- `.planning/phases/22-region-package-data-model/22-CONTEXT.md` — D-12 status-getter design and `updateAvailable` staleness mechanism (background for D-02)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `DownloadingTrailIds.download(trail)` — single shared download entry point (keepAlive notifier); insert the guard here once rather than at each of the two call sites.
- `TileRepositoryStatus` notifier already exposes `downloadVector`/`downloadDem` per region — the sheet's Download button is a consumer, no new engine method needed (unlike Phase 24's DEM-delete addition).
- `downloadNotificationServiceProvider` already drives the trail's progress notification — extend/aggregate it for D-10's unified bar rather than building a new notification system.
- Settings Offline Maps/Regions screen (Phase 24) — source of truth for region-row layout, size formatting, and DEM opt-in styling the sheet should match.

### Established Patterns
- Region bbox = four discrete `minLat/minLon/maxLat/maxLon` doubles; trail bbox is the same shape — coverage is a plain rectangle-intersection test, no geo library needed.
- Explicit `.code` int enum comparisons (never `.index`) when reading `RegionStatus`; `paused` no longer exists.
- Fire-and-forget background downloads surfaced via toasts + notifications (existing trail + region UX) — D-09/D-10 stay within this pattern.

### Integration Points
- Coverage check must read the locally-persisted region catalog via `RegionRepository` (no network — D-11).
- The sheet triggers `downloadTrail` (trail service) AND `downloadVector`/`downloadDem` (region notifier) from one button press; their progress feeds one unified notification (D-10).

</code_context>

<specifics>
## Specific Ideas

- Guard is a **bottom modal sheet**, explicitly not an alert dialog (D-05).
- Per-region rows carry **two checkboxes (Vector, DEM)** with sizes in the subtitle, "similar to the settings region screen" (user's words) — D-06.
- Checkboxes are **selection-only**; nothing downloads until the **one big bottom Download button**, which starts all selected region data **plus the trail** at once (D-08).
- Downloads run in parallel in the background, reported by a **single unified progress-bar notification** (D-10).
- Default selection: Vector checked, DEM unchecked (D-07).

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope. All decisions clarify how the already-scoped guard is implemented; no new capabilities were proposed.

</deferred>

---

*Phase: 26-trail-download-guard*
*Context gathered: 2026-07-24*
