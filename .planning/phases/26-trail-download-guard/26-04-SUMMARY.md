---
phase: 26-trail-download-guard
plan: 04
subsystem: mobile-offline-maps
tags: [flutter, riverpod, trail-download, region-coverage-guard, notifications, gap-closure]

# Dependency graph
requires:
  - phase: 26-trail-download-guard
    provides: "Plan 03's coverage guard (missing-coverage sheet, parallel region-package starts, unified aggregate notification) in DownloadingTrailIds.download()"
provides:
  - "regionListNotifierProvider invalidation after the guard's own region packages settle (CR-02 / GUARD-04)"
  - "Crash-safe download() via a single outer try/finally guaranteeing trail.id is always cleared from DownloadingTrailIds.state (CR-01 / GUARD-01)"
  - "Aggregate-aware onGeneratingChanged that keeps the unified id-42 notification stable through tile generation (WR-01 / GUARD-03 / D-10)"
  - "Locally-swallowed exceptions on all three fire-and-forget notification calls (WR-02)"
affects: [27-legacy-cleanup]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Single outer try/finally around an entire async orchestration method to guarantee always-clears-state cleanup, with post-settle side effects (region futures wait, subscription close, cache-warm await) kept outside the try so they don't delay the primary unlock signal"
    - "ref.container.listen(...) subscription typed via the bare (non-generic) ProviderSubscription type, matching main.dart's existing precedent, requiring an explicit package:flutter_riverpod/flutter_riverpod.dart import (riverpod_annotation alone doesn't expose the type to the analyzer reliably)"

key-files:
  created: []
  modified:
    - app/lib/provider/trail/trail_download_state_provider.dart

key-decisions:
  - "Hoisted regionFutures/aggregateSub/glyphCacheWarm above the outer try so the post-try region-wait, subscription-close, and cache-warm-await code (which must run after the trail download settles, unchanged) can still reference them; regionFutures stays a final list mutated via addAll(...) instead of reassigned"
  - "Added an explicit package:flutter_riverpod/flutter_riverpod.dart import for the ProviderSubscription type (previously only riverpod_annotation was imported); its absence produced a misleading 'Undefined class' + 'KeepAliveLink can only be used inside keepAlive providers' pair of diagnostics that resolved once the import was added"

requirements-completed: [GUARD-01, GUARD-03, GUARD-04]

# Metrics
duration: ~5min
completed: 2026-07-24
---

# Phase 26 Plan 04: Trail Download Guard Gap Closure Summary

**Closed the one blocking gap (missing `regionListNotifierProvider` invalidation) and three robustness findings from Phase 26 verification/code-review in `DownloadingTrailIds.download()`, without changing any already-verified guard behavior or button-unlock timing.**

## Performance

- **Duration:** ~5 min
- **Started:** 2026-07-24T12:32:00Z (approx)
- **Completed:** 2026-07-24T12:37:00Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments
- CR-02 (blocking) closed: `download()` now invalidates `regionListNotifierProvider` in a `finally` after its guard-triggered region packages settle, so a still-mounted `RegionListNotifier` (e.g. Settings → Offline Regions) re-reads the current ObjectBox status instead of a stale cached `ToOne.target` and no longer re-shows the missing-coverage sheet for a region the user just downloaded
- CR-01 closed: restructured `download()` into a single outer `try/finally` so `trail.id` is always removed from `DownloadingTrailIds.state` — even on an exception before the trail download itself starts — while preserving today's unlock timing (still clears the instant the trail download settles, not delayed until region packages finish)
- WR-01 closed: `onGeneratingChanged` now branches on `hasSelectedPackages`, calling `updateAggregate()` instead of `showGenerating` when packages are selected, so the unified id-42 aggregate notification's title/body/item-count survives the tile-generation phase
- WR-02 closed: the three fire-and-forget notification calls (`showAggregateProgress`, `showGenerating`, `onProgress`'s `showProgress`) now append `.catchError((_) {})` so a notification-plugin exception is swallowed locally instead of surfacing as an unhandled async error indistinguishable from a real download failure

## Task Commits

1. **Task 1: CR-02 invalidate regionListNotifierProvider + CR-01 always-clear-state restructure** - `5d35ad84` (fix)
2. **Task 2: WR-01 aggregate-aware onGeneratingChanged + WR-02 harden fire-and-forget notification calls** - `12a920ee` (fix)

**Plan metadata:** (this commit)

## Files Created/Modified
- `app/lib/provider/trail/trail_download_state_provider.dart` - Restructured `DownloadingTrailIds.download()` into a single outer try/finally (CR-01), added a `regionListNotifierProvider` invalidation in the region-futures `finally` (CR-02), made `onGeneratingChanged` aggregate-aware (WR-01), and hardened three fire-and-forget notification calls with `.catchError((_) {})` (WR-02)

## Decisions Made
- Hoisted `regionFutures`/`aggregateSub`/`glyphCacheWarm` above the outer try/finally so the post-settle region-wait, subscription-close, and cache-warm-await blocks (which must stay outside the try to preserve unlock timing) can still reference them
- Used the bare `ProviderSubscription` type (no generic parameter) for `aggregateSub`, matching the existing precedent in `main.dart`, and added the `package:flutter_riverpod/flutter_riverpod.dart` import needed for the analyzer to resolve it (the file previously only imported `riverpod_annotation`)

## Deviations from Plan

None - plan executed exactly as written. Both tasks matched the plan's described structural changes; the only additive detail was the `flutter_riverpod` import required to make the `ProviderSubscription` type resolvable, which the plan's action text didn't call out but which acceptance criteria (a clean `flutter analyze`) required.

## Issues Encountered
- After the initial Task 1 edit, `flutter analyze` reported an "Undefined class 'ProviderSubscription'" error plus two "KeepAliveLink can only be used inside keepAlive providers" warnings on unrelated `ref.read(...)` lines. Root cause: the file only imported `riverpod_annotation`, which doesn't reliably expose `ProviderSubscription` to the analyzer, and the broken type resolution cascaded into unrelated false-positive keepAlive warnings. Fixed by adding `import 'package:flutter_riverpod/flutter_riverpod.dart';` — all three diagnostics cleared immediately (Rule 3 auto-fix, blocking).
- After the Task 2 edit, `dart format --set-exit-if-changed` flagged the file as needing reformatting (a long single-line `.catchError` call). Ran `dart format` on the file before the final `flutter analyze`/grep verification pass; no functional change, purely whitespace (Rule 3 auto-fix, blocking for the plan's own verification gate).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- All 4 plans of Phase 26 (Trail Download Guard) are now complete: coverage-guard core logic (26-01), missing-coverage sheet + aggregate notification (26-02), parallel region-package starts wired into `download()` (26-03), and this gap-closure plan (26-04) addressing the blocking CR-02 finding plus CR-01/WR-01/WR-02 robustness fixes
- `flutter analyze` and the full `trail_coverage_util_test.dart` suite (11/11) both pass clean; no regression to the fully-covered / no-region-gap / dismiss-abort / 0-region download paths
- On-device UAT (Human Verification #5 in `26-VERIFICATION.md`) for the CR-02 re-fire-suppression and WR-01 stable-notification behaviors remains outstanding per this phase's `human_verify_mode: end-of-phase` setting — recommended before formally closing Phase 26
- Ready to proceed to Phase 27 (Legacy Cleanup) once end-of-phase UAT is confirmed

---
*Phase: 26-trail-download-guard*
*Completed: 2026-07-24*

## Self-Check: PASSED
