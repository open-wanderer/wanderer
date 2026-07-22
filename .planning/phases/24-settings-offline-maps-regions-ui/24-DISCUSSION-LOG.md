# Phase 24: Settings — Offline Maps/Regions UI - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-22
**Phase:** 24-settings-offline-maps-regions-ui
**Areas discussed:** DEM toggle/delete & errors, Status display mapping, Disk usage & progress detail, List content & sorting

---

## DEM Toggle, Delete & Errors

| Option | Description | Selected |
|--------|-------------|----------|
| Delete DEM file immediately (needs new engine method) | Toggle off = instant delete, mirrors toggle on = instant `downloadDem()`. Requires new `deleteDemPackage(regionId)` on `TileRepositoryManager`. | ✓ |
| Toggle off just stops future downloads, doesn't delete existing | Toggle represents a preference only; existing DEM stays until region deleted entirely. | |
| Toggle off shows a confirm dialog before deleting | Same as option 1 but with a confirmation step. | |

**User's choice:** Delete DEM file immediately (needs new engine method)
**Notes:** No confirmation needed for the DEM toggle specifically — asymmetric with full-region delete (see below).

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, confirm dialog | Matches `settings_categories_screen.dart`'s own-trail-disable-confirm pattern. | ✓ |
| No, direct delete with undo-via-redownload | Delete acts immediately; re-download always possible. | |

**User's choice:** Yes, confirm dialog
**Notes:** Applies to full-region delete (`deleteRegion()`), not the DEM toggle.

| Option | Description | Selected |
|--------|-------------|----------|
| Inline error badge + retry action | Distinct error indicator with tappable Retry that re-invokes download from scratch. | ✓ |
| Inline error badge + tap row to see message, then manual re-download | Surfaces error text first; retry is a separate manual step. | |

**User's choice:** Inline error badge + retry action

---

## Status Display Mapping

| Option | Description | Selected |
|--------|-------------|----------|
| 6 distinct visual states | Each of the 6 `RegionStatus` values gets its own icon/label/action; roadmap's "4-state" phrasing describes the baseline lifecycle, not a visual cap. | ✓ |
| Paused/error fold into 'downloading' visually | Only 4 top-level looks; paused/error are sub-states of the "in progress" look. | |

**User's choice:** 6 distinct visual states

| Option | Description | Selected |
|--------|-------------|----------|
| Small badge/chip next to the row, tap opens update action | Row reads as fully "downloaded" with a small non-blocking badge. | |
| Persistent banner text under the row name | More visible text line plus inline update button, always visible. | ✓ |

**User's choice:** Persistent banner text under the row name

---

## Disk Usage & Progress Detail

| Option | Description | Selected |
|--------|-------------|----------|
| Include partial bytes | Sums `sizeBytesOnDisk` across every package regardless of status. | ✓ |
| Only completed downloads | Sums only `status==downloaded` packages. | |

**User's choice:** Include partial bytes

| Option | Description | Selected |
|--------|-------------|----------|
| Two separate progress bars (vector + DEM) | Each package gets its own labeled bar, matching the independent-package model. | |
| One combined progress bar for the region | Averages/combines `vectorProgress` and `demProgress` into one bar. | ✓ |

**User's choice:** One combined progress bar for the region

---

## List Content & Sorting

| Option | Description | Selected |
|--------|-------------|----------|
| Region name only | Filters by `RegionEntity.name` substring match. | ✓ |
| Name + id | Also matches the internal catalog id. | |

**User's choice:** Region name only

| Option | Description | Selected |
|--------|-------------|----------|
| Alphabetical; building/error rows shown disabled with a label | List sorts A-Z always; not-yet-ready regions shown grayed-out with a label. | ✓ |
| Downloaded-first, then alphabetical; building/error rows hidden entirely | Downloaded/in-progress regions surface at top; not-ready regions filtered out entirely. | |

**User's choice:** Alphabetical; building/error rows shown disabled with a label

---

## Claude's Discretion

None — every gray area identified was explicitly decided by the user.

## Deferred Ideas

None — discussion stayed within phase scope; no new capabilities were proposed.
