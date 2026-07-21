# Phase 22: Region & Package Data Model - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-21
**Phase:** 22-region-package-data-model
**Areas discussed:** Explicit-int status pattern, regions.json initial content, DownloadedTilePackage shape, Region vs package status relationship

---

## Explicit-int status pattern

| Option | Description | Selected |
|--------|-------------|----------|
| Enum with explicit int getter | Enhanced enum with a `const int code` per value; persist/read via `code`, never `.index`. Order-independent when adding new statuses later. | ✓ |
| Plain int class constants, no enum | `static const int` values on a plain class, raw int field on entity. Simpler persistence, loses exhaustiveness checking. | |
| Enhanced enum + @Transient (mirrors existing shadow pattern) | Same enhanced-enum idea but explicitly keeping the existing shadow-property shape from TrailEntity/ActiveNavigationEntity, swapping `.index` for `.code`. | |

**User's choice:** Enum with explicit int getter (Recommended option).
**Notes:** Confirmed persistence still needs a shadow int property (never `.index`) — effectively merges the recommended option with the structural shape of option 3.

---

## regions.json initial content

| Option | Description | Selected |
|--------|-------------|----------|
| Real regions, real URLs | A handful of real named regions with real bbox/PMTiles/DEM URLs against the existing backend pipeline. | ✓ |
| Placeholder/test entries | Dummy ids/names/URLs just to prove the parse model works; real content swapped in later. | |
| One real region, rest placeholder | One fully real testable region + a few placeholders to prove multi-entry/partial-DEM handling. | |

**User's choice:** Real regions, real URLs.

**Follow-up question:** Which real regions should the initial manifest cover?

| Option | Description | Selected |
|--------|-------------|----------|
| You decide / Claude's discretion | Pick a small sensible starting set based on existing trail data density. | |
| One region covering the whole test/demo dataset | Single region bbox covering seeded/demo trail data. | |
| I'll specify exact regions now | (free text) | ✓ |

**User's choice (free text):** "Research how osmand and comaps split the world into regions. Model it after that. Include 3-4 regions for now."
**Notes:** Captured as a research flag (D-05 in CONTEXT.md) — exact region boundaries/URLs are not fully locked in this discussion; the phase researcher should investigate OsmAnd/CoMaps' region-splitting convention and propose concrete regions + URLs before planning finalizes manifest content.

---

## DownloadedTilePackage shape

| Option | Description | Selected |
|--------|-------------|----------|
| Two ToOne fields on Region | `vectorPackage`/`demPackage` as two nullable `ToOne<DownloadedTilePackage>` fields. Direct field access, no discriminator. | ✓ |
| ToMany + type discriminator | `ToMany<DownloadedTilePackage>` with a `PackageType` enum field per row. More extensible, needs a filter on every read. | |

**User's choice:** Two ToOne fields on Region (Recommended option).

---

## Region vs package status relationship

| Option | Description | Selected |
|--------|-------------|----------|
| Region.status is independently tracked | Own stored field, set directly by TileRepositoryManager alongside package updates. | (initial choice, revised) |
| Region.status is a computed getter, not stored | Derived on the fly from package statuses; guarantees no drift but contradicts REGN-02's literal "persists a live status" wording. | ✓ |

**User's choice:** Region.status is a computed getter, not stored.

**Follow-up question (flagging the REGN-02 wording tension):** What should the getter return when no package rows exist yet (pre-download state)?

| Option | Description | Selected |
|--------|-------------|----------|
| Getter defaults to notDownloaded when no packages exist | `vectorPackage.target?.status ?? RegionStatus.notDownloaded`, folding in demPackage when required/present. | ✓ |
| Actually, store it after all | Revert to a stored field set by TileRepositoryManager in the same transaction as package updates. | |

**User's choice:** Getter defaults to `notDownloaded` when no packages exist.
**Notes:** Flagged for the planner that this is an intentional, discussed deviation from REGN-02's literal "persists... a live status" wording — not an oversight. If Phase 23/24 need to query/filter/sort by Region status directly in ObjectBox, that may require revisiting this (stored+synced field, or in-memory post-fetch filter).

---

## Claude's Discretion

None — all four selected gray areas were explicitly decided by the user.

## Deferred Ideas

None — discussion stayed within phase scope.
