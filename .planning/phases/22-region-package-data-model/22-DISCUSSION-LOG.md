# Phase 22: Region & Package Data Model - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-21
**Phase:** 22-region-package-data-model
**Areas discussed:** Catalog fetch & merge strategy, Backend status vs local download status, Staleness → updateAvailable mapping, Region removed from catalog / partial DEM handling

**Note:** This is a full re-discussion, replacing an earlier same-date discussion. The prior CONTEXT.md/DISCUSSION-LOG.md assumed a bundled `assets/map/regions.json` asset — obsoleted by Phase 21.5's insertion, which established the region catalog is fetched from a new backend API instead. This discussion re-derives Phase 22's decisions against that new API contract (`GET /api/v1/regions`, see `db/routes/regions_get.go`).

---

## Catalog Fetch & Merge Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Upsert by id, preserve local fields | Find-or-create by region id; update catalog-owned fields in place; never touch local-only fields | ✓ |
| Replace-all like subcategory_provider | removeAll() + putMany() every refresh, matching an established app pattern | |
| You decide | Claude picks | |

**User's choice:** Upsert by id, preserve local fields.
**Notes:** Rejected the replace-all pattern despite it being the codebase's established shape (subcategory/category providers) because it would destroy ToOne package links and local download status via ObjectBox's obxId-keyed removeAll().

| Option | Description | Selected |
|--------|-------------|----------|
| On-demand (Settings page open) | Fetch only when Settings/Regions screen opens; no background refresh on launch | ✓ |
| Eager on app launch + on-demand refresh | Mirrors subcategory_provider's build()-triggers-refresh pattern | |
| You decide (defer the wiring) | This phase only builds the fetch function, timing decided later | |

**User's choice:** On-demand (Settings page open).
**Notes:** Since this phase has zero UI, the actual call site isn't wired here regardless — this sets intent for Phase 24 without hard-coding it into the function's design.

| Option | Description | Selected |
|--------|-------------|----------|
| Keep existing rows untouched, surface error to caller | Don't swallow the error at this layer since there's no UI yet to swallow into | ✓ |
| You decide | Claude picks | |

**User's choice:** Keep existing rows untouched, surface error to caller.

---

## Backend Status vs Local Download Status

| Option | Description | Selected |
|--------|-------------|----------|
| Two separate fields | `catalogStatus` (backend building/ready/error) + existing computed `status` getter (local download state) | ✓ |
| Single Region.status enum merges both | Extend RegionStatus with notAvailable alongside download states | |
| You decide | Claude picks based on locked D-01/D-02/D-07 | |

**User's choice:** Two separate fields.

| Option | Description | Selected |
|--------|-------------|----------|
| Same explicit-int enum pattern | catalogStatus gets its own enhanced enum with `code` int, same shadow-property persistence | ✓ |
| Store as plain string | Simpler since it's fully overwritten on every fetch, never read via .index | |
| You decide | Claude picks | |

**User's choice:** Same explicit-int enum pattern.

---

## Staleness → updateAvailable Mapping

| Option | Description | Selected |
|--------|-------------|----------|
| Store lastDownloadedVersion on Region, compare on fetch | Persist version string at last successful vector download; flip to updateAvailable on mismatch + ready + already-downloaded | ✓ |
| You decide | Claude designs the mechanism | |

**User's choice:** Store lastDownloadedVersion on Region, compare on fetch.
**Notes:** Claude noted (not asked as a question, derived from reading `db/routes/regions_get.go`) that the backend response has no DEM-equivalent version field — DEM has no staleness/updateAvailable concept in this phase, per Phase 21.5's D-11.

---

## Region Removed from Catalog / Partial DEM Handling

| Option | Description | Selected |
|--------|-------------|----------|
| Keep row + files, mark as orphaned | `inCatalog: bool` field flipped false on missing fetch; files untouched | ✓ |
| Delete row and packages immediately on fetch | Full referential sync with backend as ground truth | |
| You decide | Claude picks the safer default | |

**User's choice:** Keep row + files, mark as orphaned.

| Option | Description | Selected |
|--------|-------------|----------|
| No package row until a download actually starts | DownloadedTilePackage only created when Phase 23's engine begins downloading; matches original D-06 exactly | ✓ |
| You decide | Claude confirms against locked D-06 | |

**User's choice:** No package row until a download actually starts.

---

## Claude's Discretion

None — every gray area reached an explicit user decision.

## Deferred Ideas

None raised. The bigger scope question (what UI shows for orphaned/updateAvailable regions) was explicitly routed to Phase 24/26, not deferred as a new capability — it's already in those phases' scope.
