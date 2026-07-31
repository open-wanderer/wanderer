---
phase: 33
slug: conversion-correctness
status: verified
threats_open: 0
asvs_level: 1
created: 2026-07-31
---

# Phase 33 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

Register authored at plan time (all 5 PLANs carry a `<threat_model>` block), so this
audit verified existing mitigations rather than building a retroactive STRIDE register.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| HTTP client → `PUT /api/v1/trail/upload`, `POST /api/v1/trail/convert` | Untrusted, user-supplied GPX XML reaches `GPX.parse()` then `getTotals()`. Coordinates are attacker-controlled numeric strings coerced by `attrValueProcessors`; `<ele>` is element text and is **not** coerced by them. | GPX XML, arbitrary numeric strings |
| Browser file picker / route planner → `GPX` constructor | Same code path, client side. Values originate from an imported file or the Valhalla proxy response. | GPX XML, Valhalla route/height JSON |
| `getTotals()` output → `gpx2trail()` → PocketBase + Meilisearch | Computed `distance`, `elevation_gain`, `elevation_loss`, bounding box persisted with no server-side recomputation. `trail.distance` is also read by `findDuplicate()` as a similarity threshold. | Derived numeric trail metrics |
| `features.cumulativeDistance` → browser DOM / MapLibre | The crop slider converts array entries into `setLngLat()` coordinates on live map markers; MapLibre throws on `NaN`. | Interpolated coordinates |
| Cached derived state (`cropPreview`) → `setRoute()` → saved trail geometry | A stale cache crossing this boundary would destroy user route data. | Route geometry |

---

## Threat Register

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-33-01 | Tampering | `GPX.getTotals()` centroid divisor | mitigate | `summedPointCount++` in the same loop body as `totalLat`/`totalLon`; centroid divides by it. `gpx.ts:104,126-138,151` — zero `allPoints.length` divisor occurrences. | closed |
| T-33-02 | Tampering | Zero-point / empty-segment GPX reaching `centroid` | accept | `gpx_util.ts:82` gates the bbox write behind `Number.isFinite()`; `centroid` is not persisted. Pinned by the zero-point regression guard in `gpx.test.ts`. | closed |
| T-33-03 | DoS | `allPoints.push(...points)` over a very large track | accept | Unchanged by the phase; request body limits remain the controlling bound. | closed |
| T-33-04 | Tampering | XML entity expansion / XXE via `isomorphic-xml2js` | accept | `GPX.parse()`'s `xml2js.parseString` call and options byte-identical to pre-phase. | closed |
| T-33-05 | Tampering | `<ele>` element text reaching elevation accumulation | mitigate | `parseElevation` is the sole coercion point (`gpx-metrics-computation.ts:15-24,105,142`); rejects `NaN`/`Infinity`/non-numeric. Only two `point.ele` reads exist, both routed through it. | closed |
| T-33-06 | Tampering | Elevation anchors initialized from a hostile first point | mitigate | Zero `?? 0` occurrences in the file; `lastZ`/`lastFilteredZ` assigned only from `parseElevation`'s validated output. | closed |
| T-33-07 | Tampering | CONV-04 restructure weakening `thresholdXY_m` | mitigate | Positive `if (smoothedDistance >= this.thresholdXY_m)` block retained (`:231-234`), no early return. Distance-smoothing tests pass (0 m smoothed / ~4.403 raw on the scramble). | closed |
| T-33-08 | DoS | Extremely large `<ele>` value strings | accept | `Number(...)` cost bounded by the existing request-body limit. | closed |
| T-33-09 | Tampering | XXE via `isomorphic-xml2js` | accept | Parser config untouched. | closed |
| T-33-10 | Tampering | `totalDistance` feeding persisted `trail.distance` and `findDuplicate()` | mitigate | `if (Number.isFinite(distance)) { this.totalDistance += distance; }` (`:132-134`). | closed |
| T-33-11 | Tampering | `cumulativeDistance` desynchronizing from `flatten()` | mitigate | Exactly two unconditional push sites (first-call `:114`, main body `:138`) — one per `addAndFilter()` call regardless of input validity. | closed |
| T-33-12 | Tampering (client) | `updateCropMarkers()` deriving markers from a bad cumulative array | mitigate | **Drift-adjusted.** Guard is now `!basisMatchesRoute \|\| !hasCropInterpolationBasis(cumulativeRoute)` (`+page.svelte:1494-1504`) — stronger than the plan text, also rejecting `total === 0` and a length mismatch with `flatten()`. Runs before any target-distance math or `setLngLat`. | closed |
| T-33-13 | DoS | One extra `cumulativeDistance` entry per GPX | accept | O(points), bound unchanged. | closed |
| T-33-14 | Tampering | XXE via `isomorphic-xml2js` | accept | Parser untouched. | closed |
| T-33-SC | Supply chain | npm installs | n/a | No `package.json`/`package-lock.json` diff in the phase commit range or working tree. | closed |
| T-33-04-01 | Tampering | Smoothed elevation totals becoming `NaN`/`Infinity`/negative | mitigate | **Drift-adjusted.** Commit-then-retract was replaced by defer-then-publish, so the described retract path no longer exists. Property verified directly: `parseElevation` guarantees finite input; `publishPending()` (`:87-95`) only ever adds; `finalElevationGain/Loss` add `Math.max(±pendingDelta, 0)`. Backed by 6 monotonicity fixtures + 1 segment-differencing test. | closed |
| T-33-04-02 | Tampering | `trail.elevation_gain` / `elevation_loss` persistence | mitigate | `gpx.ts:146-147` assigns from `finalElevationGain/Loss`; `gpx_util.ts:74-75` persists verbatim — boundary intact. | closed |
| T-33-04-03 | DoS | `addAndFilter()` per-point cost | accept | Still O(1) per point (≤1 extra `haversineDistance`); no new per-point allocation. | closed |
| T-33-04-SC | Supply chain | npm installs | mitigate | No package changes. | closed |
| T-33-05-01 | DoS (client) | `[NaN, NaN]` reaching MapLibre `LngLat.convert` | mitigate | `crop.ts:52-53` `span > 0` guard replaces the `0/0` division; `hasCropInterpolationBasis` rejects non-positive totals. Both reproduced-defect fixtures assert finite coordinates. | closed |
| T-33-05-02 | Tampering (data loss) | Stale crop cache reaching `setRoute()` | mitigate | **Drift-adjusted.** `croppedGPX` is now `cropPreview`. Reset at the degenerate early-return (`:1502`), `toggleCropMarkers(false)` (`:1463`), and `updateTrailWithRouteData()` (`:1596`) — the choke point all 13 route-mutation sites route through. `confirmCrop()` hoists the value before the choke point fires. | closed |
| T-33-05-03 | Info Disclosure / EoP | Trail-edit crop panel | accept | No new data read, transmitted, or authorized differently; browser-local geometry only. | closed |
| T-33-05-04 | Tampering | `crop.ts` numeric returns | mitigate | Module is total: empty/single-point/all-zero/coincident-leading inputs all return finite numbers; tests assert `Number.isFinite` on every fixture. | closed |
| T-33-05-SC | Supply chain | npm installs | mitigate | No package changes. | closed |

*Status: open · closed*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-33-01 | T-33-02 | Zero-point GPX yields `NaN` centroid / `Infinity` bbox sentinels — pre-existing, and `gpx_util.ts:82` gates the bbox write behind `Number.isFinite()`. `centroid` is never persisted. | Christian Beutel | 2026-07-31 |
| AR-33-02 | T-33-03, T-33-13 | Allocation stays O(points) and is already bounded by the request-body limit governing the whole GPX document. | Christian Beutel | 2026-07-31 |
| AR-33-03 | T-33-04, T-33-09, T-33-14 | XXE / entity expansion via `isomorphic-xml2js` is out of scope: `GPX.parse()` and its parser options are byte-identical before and after this phase. Pre-existing posture, unchanged. | Christian Beutel | 2026-07-31 |
| AR-33-04 | T-33-08 | `Number(...)` on a long numeric string is O(n) in token length, already bounded by the request-body limit; the parser materializes the same strings today. | Christian Beutel | 2026-07-31 |
| AR-33-05 | T-33-04-03 | `addAndFilter()` adds ≤1 extra `haversineDistance` call per sample, O(1); a hostile 10^6-point GPX costs the same order as before. | Christian Beutel | 2026-07-31 |
| AR-33-06 | T-33-05-03 | The crop panel operates only on route geometry already loaded in the user's own editor session. No ASVS L1 control applies. | Christian Beutel | 2026-07-31 |

---

## Residual Notes

Not gating (risk-reducing or pre-existing), recorded so the accounting stays complete:

- **Uncommitted crash fixes** — `gpx.ts:207-208`, `valhalla.ts:135`, `valhalla_store.svelte.ts:107`, `+page.svelte:580` close two crash-on-real-input defects found in `33-REVIEW.md` round 3 (a missing Valhalla `height` array crashing `correctElevation()`; a trailing empty `<trkseg>` crashing `initRouteAnchors()` on mount). Both predate phase 33 and map to no threat ID. Verified fixed by inspection; risk-reducing, not new surface.
- **WR-09 residual fragility** — `setRoute()` does not call `getTotals()` explicitly, relying on `json-diff-ts` to transport `features.cumulativeDistance` faithfully. Currently holds empirically, but T-33-05-01 / T-33-12's closure depends on that invariant continuing to hold.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-07-31 | 24 | 24 | 0 | gsd-security-auditor |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-07-31
