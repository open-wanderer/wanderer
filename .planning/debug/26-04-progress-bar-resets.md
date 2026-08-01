---
status: diagnosed
trigger: "Investigate issue: 26-04-progress-bar-resets — During a multi-region trail download (trail + selected Vector/DEM packages downloading in parallel with one unified id-42 notification), the aggregate progress bar resets 3 times to different values during the download and finally finishes before reaching 100%, even though the trail itself downloads successfully."
created: 2026-07-24T13:20:00Z
updated: 2026-07-24T13:45:00Z
---

## Current Focus

reasoning_checkpoint:
  hypothesis: "updateAggregate() in trail_download_state_provider.dart sums each selected region package's vectorProgress/demProgress fraction and falls back to 0.0 via `?? 0.0` when the field is null. tile_repository_provider.dart's downloadVector()/downloadDem() intentionally clear their own progress field back to null in their `finally` block the instant that package finishes (documented design: 'each progress field's presence (non-null) IS that package's live downloading signal', consumed by resolveVectorTileStatus/resolveDemTileStatus for the per-region Settings UI). Because updateAggregate() cannot distinguish 'not yet started' from 'just completed' (both read as null -> 0.0), each of the 3 selected packages' contribution to the aggregate sum snaps from 1.0 back to 0.0 at the exact moment it finishes -- and aggregateSub is subscribed to the same provider, so that clear immediately fires a recompute, producing a visible downward jump in the displayed percentage. With 2 vector regions + 1 DEM region selected this produces exactly 3 downward jumps ('resets'), and once all 3 have finished (and thus all read back as 0), combined can never exceed roughly lastTrailFraction/itemCount (~25% here) before the trail's own showSuccess() overwrites the notification -- matching 'finally finishes before reaching 100%'."
  confirming_evidence:
    - "tile_repository_provider.dart lines 68-80 (downloadVector) and 110-120 (downloadDem): `finally { ... _clearVectorProgress(regionId); }` / `_clearDemProgress(regionId)` run unconditionally on completion, setting vectorProgress/demProgress back to null (region_download_state.dart's own doc comment confirms this null-on-completion contract is intentional for the per-region UI)."
    - "trail_download_state_provider.dart lines 131-149 (updateAggregate): `sum += packageStates[region.id]?.vectorProgress ?? 0.0` / `?? 0.0` for demProgress -- treats null identically to 'contributed nothing', with no way to know the package actually finished."
    - "trail_download_state_provider.dart line 155-160: aggregateSub = ref.container.listen(tileRepositoryStatusProvider, (_, _) => updateAggregate()) -- the completion-triggered null-clear itself is a state change that fires this listener, so the drop is immediately rendered in the id-42 notification, not just a stale read."
    - "Selected packages in the reported repro (2 vector regions + 1 DEM region) = exactly 3 independent completion events, matching 'resets 3 times' precisely; itemCount = 1 (trail) + 2 (vector) + 1 (dem) = 4, so once all 3 have cleared to null, combined caps near lastTrailFraction/4 (~25%) even at true completion, matching 'finishes before reaching 100%'."
  falsification_test: "If the hypothesis is right, a single-package-selection repro (1 vector region, no DEM) should show exactly 1 reset instead of 3, timed to that package's own completion -- proportional to packages selected, not a fixed count. This wasn't run live (device/on-device test not available in this environment) but is a direct, deterministic consequence of the code paths read, not an inferred guess: the null-clear -> ?? 0.0 fallback -> subscribed-listener-refire chain is unconditional and always executes on every package completion."
  fix_rationale: "The fix must stop conflating 'this region package's ephemeral downloading-in-progress signal' (which is correctly and intentionally cleared to null on completion for the per-region Settings/Regions UI) with 'this region package's contribution to a multi-item aggregate percentage' (which must latch at 1.0 permanently once done, for as long as the aggregate notification is alive). This needs a completion-aware accumulator local to DownloadingTrailIds.download() (mirroring how `lastTrailFraction` already latches monotonically) rather than re-reading tileRepositoryStatusProvider's ephemeral per-region state directly inside updateAggregate() after a package's future resolves -- not a change to tile_repository_provider.dart's clear-on-completion behavior, which other UI (Settings/Regions screen) correctly depends on."
  blind_spots: "Have not run this live on-device (find_root_cause_only mode) to observe the actual displayed percentages match this math exactly; relying on direct code-path tracing rather than an instrumented repro. Have not fully traced whether showSuccess() firing independently of regionFutures completing is a second, compounding issue (trail's own success can overwrite the id-42 notification before region packages finish) -- flagged as a related but secondary concern, not the primary root cause of the reset/under-100% symptom."

## Symptoms

## Symptoms

expected: One id-42 notification shows a single combined, monotonically advancing progress bar (not one notification per download, not resetting) across the parallel trail+package download.
actual: "The download bar resets 3 times to different values during download and finally finishes before 100%. The trail is successfully downloaded."
errors: None reported by user
reproduction: On a multi-region trail, check Vector for two regions and DEM for one, tap Download once (this is UAT Test 4 in .planning/phases/26-trail-download-guard/26-UAT.md)
started: Discovered during on-device UAT after Phase 26 (trail-download-guard) + gap-closure plan 26-04 executed. Phase 26 wired the coverage guard into DownloadingTrailIds.download() with parallel region+trail downloads and a unified aggregate notification (updateAggregate()/showAggregateProgress in app/lib/provider/trail/trail_download_state_provider.dart and app/lib/services/download_notification_service.dart). 26-04 fixed CR-02 (missing provider invalidation), CR-01 (stranded download button), WR-01 (onGeneratingChanged not aggregate-aware), and WR-02 (unhandled notification errors) — but did NOT touch the progress-percentage computation itself.

## Eliminated

(none — root cause found on first evidence-driven hypothesis; no false leads pursued)

## Evidence

- timestamp: 2026-07-24T13:30:00Z
  checked: app/lib/provider/trail/trail_download_state_provider.dart (full file, DownloadingTrailIds.download())
  found: updateAggregate() computes `sum = lastTrailFraction + Σ packageStates[region.id]?.vectorProgress/demProgress ?? 0.0`, then `combined = (sum / itemCount).clamp(0,1)`, shown via showAggregateProgress. aggregateSub listens on tileRepositoryStatusProvider and calls updateAggregate() on every change. showSuccess() (trail-only) fires as soon as trailDownloadService.downloadTrail() resolves, not gated on regionFutures (selected package downloads) also completing.
  implication: The aggregate math depends entirely on tileRepositoryStatusProvider's per-region vectorProgress/demProgress fields staying at 1.0 once a package finishes; if that provider ever resets a finished package's field to null, the aggregate permanently loses that package's credit.

- timestamp: 2026-07-24T13:33:00Z
  checked: app/lib/provider/region/tile_repository_provider.dart (full file, TileRepositoryStatus)
  found: downloadVector()'s and downloadDem()'s `finally` blocks unconditionally call `_clearVectorProgress`/`_clearDemProgress` the instant the package's own download future resolves — successfully or not. These clear the field back to `null` (or remove the region's map entry entirely if the sibling package also has no progress).
  implication: A region package's progress field is null both BEFORE it starts and immediately AFTER it finishes — there is no persistent "done" signal in this ephemeral state for a since-finished package.

- timestamp: 2026-07-24T13:35:00Z
  checked: app/lib/models/region_download_state.dart doc comment
  found: "each progress field's presence (non-null) IS that package's live 'downloading' signal; see region_tile_status_util.dart's resolveVectorTileStatus/resolveDemTileStatus" — this null-on-completion behavior is a deliberate, documented contract for the per-region Settings/Regions screen (which reads the AUTHORITATIVE final status from RegionEntity/DownloadedTilePackageEntity separately, not from this ephemeral field).
  implication: tile_repository_provider.dart is not itself buggy for its original (Settings/Regions screen) consumer. The bug is that trail_download_state_provider.dart's updateAggregate() (added in Phase 26) reuses this same ephemeral "currently downloading" signal as if it were a monotonic completion-fraction source, via `?? 0.0`, which cannot distinguish "not started" from "just completed."
  implication_2: Given the repro (2 vector regions + 1 DEM region selected, itemCount=4), each of the 3 package completions independently drops the aggregate sum by up to 1.0/4 (25 percentage points) at the moment it finishes and aggregateSub's listener refires updateAggregate() — matching "resets 3 times to different values" precisely. Once all 3 have completed and cleared to null, combined caps at roughly lastTrailFraction/4 (~25%) — matching "finally finishes before reaching 100%."

## Resolution

root_cause: "trail_download_state_provider.dart's updateAggregate() reads each selected region package's live progress via `packageStates[region.id]?.vectorProgress ?? 0.0` / `?.demProgress ?? 0.0` from tileRepositoryStatusProvider. tile_repository_provider.dart's downloadVector()/downloadDem() intentionally and by design clear that same field back to null in their `finally` block the instant the package finishes (documented in region_download_state.dart as the per-region Settings/Regions UI's live 'downloading' signal, consumed by resolveVectorTileStatus/resolveDemTileStatus — the authoritative completed status lives elsewhere, on RegionEntity/DownloadedTilePackageEntity). updateAggregate()'s `?? 0.0` fallback cannot distinguish 'package not yet started' from 'package just finished,' so each selected package's contribution to the aggregate sum snaps from 1.0 back down to 0.0 the moment it completes. Because aggregateSub (ref.container.listen on tileRepositoryStatusProvider) recomputes on every state change including this clear, the drop renders immediately in the id-42 notification. With 2 vector regions + 1 DEM region selected, this produces exactly 3 downward jumps (matching 'resets 3 times to different values'), and once all 3 packages have completed and cleared to null, the aggregate can never exceed roughly lastTrailFraction / itemCount (~25% for itemCount=4) before the trail's own showSuccess() overwrites the notification — matching 'finally finishes before reaching 100%, even though the trail itself downloads successfully.'"
fix: ""
verification: ""
files_changed: []
