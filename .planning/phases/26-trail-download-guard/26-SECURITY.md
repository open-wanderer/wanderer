---
phase: 26
slug: trail-download-guard
status: verified
threats_open: 0
asvs_level: 1
created: 2026-07-24
---

# Phase 26 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| Local ObjectBox region/trail row → coverage computation | The bbox doubles are device-local, already validated at ingestion (`RegionEntity.fromCatalogEntry` throws on `bbox.length != 4`); a corrupted/degenerate row is the only untrusted shape reaching `bboxesOverlap`/`missingCoverageRegions` | Local device data only, no network |
| Local region catalog snapshot → missing-coverage sheet render | The sheet only reads already-validated local `RegionEntity` data and renders it; no new external input path, no download side effect from the render itself | Local device data only, no network |
| Guard decision → user's offline-safety expectation | The guard's verdict shapes whether a user believes a trail is fully usable offline; a trail spanning a no-region gap must not create false confidence | Local computed verdict, no network |
| Local catalog snapshot → guard verdict (staleness) | `RegionEntity.status` reads a cached ObjectBox `ToOne.target`; a stale snapshot after a mutation can misreport coverage until invalidated | Local device data only, no network |
| Notification plugin → app async-error surface | Fire-and-forget notification `Future`s that reject leak as unhandled async errors if uncaught | Local OS notification API, no network |
| Ephemeral per-region progress → aggregate notification integrity | `updateAggregate()` consumes `tileRepositoryStatusProvider`'s ephemeral `vectorProgress`/`demProgress`, cleared to null on completion for a different consumer (Settings/Regions) | Local in-memory state, no network |
| Concurrent region-row writers → persisted RegionEntity relation integrity | Two fire-and-forget downloads (`startVectorDownload`/`startDemDownload`) write the same region row from independent stale snapshots | Local ObjectBox persistence, no network |

---

## Threat Register

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-26-01 | Tampering | `bboxesOverlap` over local region/trail doubles | mitigate | A degenerate bbox (min > max from a corrupted local row) fails the comparison and returns `false` rather than throwing — the broken region silently drops out of `overlappingRegions`/`missingCoverageRegions`; asserted by a dedicated unit test. Reused unchanged by 26-03's guard. | closed |
| T-26-02 | Information disclosure (false offline-safety confidence) | `download()` no-region-gap branch / missing-coverage sheet | mitigate | D-04 non-blocking warning toast ("Part of this trail isn't covered by any offered region") fires whenever `overlappingRegions.isEmpty`, and the download still proceeds — never a silent full-offline claim, never a hard block. The sheet itself also makes the "Downloading trail only" escape hatch explicit at 0 selected. | closed |
| T-26-03 | Repudiation / correctness (stale-state re-fire) | `download()` region-futures completion path | mitigate | `ref.invalidate(regionListNotifierProvider)` added in the `finally` after `Future.wait(regionFutures)`, honoring `region_provider.dart`'s documented invalidation contract so a live `RegionListNotifier` cannot report a just-downloaded region as missing. Verified present in source (26-04) and confirmed via on-device UAT re-fire check. | closed |
| T-26-04 | Denial of service (self-inflicted UI lockout) | `download()` pre-`try` statements | mitigate | Single outer `try/finally` guarantees `trail.id` is always removed from the `keepAlive` state set, so an exception can never permanently disable the trail's download button. Verified present in source (26-04) and confirmed via on-device UAT stranding check. | closed |
| T-26-05 | Information disclosure (misleading error surface) | fire-and-forget notification calls | mitigate | Each fire-and-forget notification call wrapped in `.catchError((_) {})` so a plugin exception cannot masquerade as a download failure. Verified present in source (26-04). | closed |
| T-26-06 | Tampering / correctness (aggregate under-count) | `download()`'s `updateAggregate` reading ephemeral per-region progress | mitigate | Raw `?? 0.0` ephemeral read replaced with per-package monotonic latch maps (`vectorLatched`/`demLatched`) raised by live progress and driven to 1.0 by each region future's `whenComplete`, so a completed package's contribution can never snap back to 0. Verified present in source (26-05) and confirmed via on-device UAT re-test. | closed |
| T-26-07 | Tampering (silent data loss) | `TileRepositoryManager` region-row `put()` on a stale snapshot | mitigate | Each write transaction now re-fetches the current row (`freshRegion = _regionById(id)`) before `put()`, so a concurrently-linked `demPackage`/`vectorPackage` FK is preserved instead of overwritten with a stale value. Verified present in source (26-05) and confirmed via on-device UAT re-test. | closed |
| T-26-08 | Information disclosure (misleading completion signal) | id-42 success notification | mitigate | id-42 success gated on `regionFutures` + `trailSucceeded`, so the notification does not falsely finalize to "success at <100%" before the region packages settle. Verified present in source (26-05) and confirmed via on-device UAT re-test. | closed |

*Status: open · closed*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|

No accepted risks — every threat in this phase was fully mitigated (no `accept`/`transfer` dispositions remained after Plan 03 superseded Plan 02's provisional `accept` on T-26-02 with a concrete mitigation).

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-07-24 | 8 | 8 | 0 | gsd-secure-phase (retroactive register build from 26-01..26-05 PLAN.md threat_model blocks; all-closed short-circuit — register_authored_at_plan_time: true, threats_open: 0, no auditor spawn needed) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-07-24
