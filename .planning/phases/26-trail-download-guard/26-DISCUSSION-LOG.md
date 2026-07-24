# Phase 26: Trail Download Guard - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-24
**Phase:** 26-trail-download-guard
**Areas discussed:** Coverage algorithm, Dialog/CTA flow, Region download scope, Catalog freshness

---

## Coverage algorithm — matching method

| Option | Description | Selected |
|--------|-------------|----------|
| Overlap-based | Every catalog region overlapping the trail bbox must be downloaded; names the missing overlapping regions | ✓ |
| Union-containment | Trail bbox must be fully contained in the geometric union of downloaded region bboxes | |

**User's choice:** Overlap-based
**Notes:** Matches how region archives tile space; easier to name which region to download.

---

## Coverage algorithm — no-region-at-all gap

| Option | Description | Selected |
|--------|-------------|----------|
| Proceed + warn | Let trail proceed; non-blocking notice if a genuine no-region gap exists | ✓ |
| Silent proceed | Treat no-region areas as covered, no hint to the user | |
| Block with generic message | Refuse — rejected by GUARD-02 | |

**User's choice:** Proceed + warn
**Notes:** Honors GUARD-03 (never force full coverage) and GUARD-02 (no silent block / generic message).

---

## Coverage algorithm — overlap strictness

| Option | Description | Selected |
|--------|-------------|----------|
| Raw bbox overlap | Any bbox-vs-bbox intersection counts | ✓ |
| Polyline-aware | Use decoded GPX polyline to test true region traversal | |
| Overlap-area threshold | Require minimum intersection area/percentage | |

**User's choice:** Raw bbox overlap
**Notes:** Deterministic and simple; worst case offers one extra optional region the user can leave unchecked.

---

## Dialog/CTA flow — region CTA vs trail download

| Option | Description | Selected |
|--------|-------------|----------|
| Fire + decouple | Region download backgrounds; trail is a separate later re-tap | |
| Auto-chain trail after | Trail auto-fires once regions complete; sheet owns orchestration | |
| Route to Settings | CTA navigates to Settings screen | |
| **Other (user free-text)** | Both trail and region(s) download in parallel | ✓ |

**User's choice:** Other — "Both downloads fire at the same time. Download the trail and region(s) in parallel."
**Notes:** Trail never waits on region coverage; independent engines run concurrently.

---

## Dialog/CTA flow — the "proceed anyway" path

| Option | Description | Selected |
|--------|-------------|----------|
| Explicit "Download trail anyway" button | Per-region buttons + separate trail-only button | |
| Trail starts only on a region tap | No zero-region start path — rejected | |
| Region tap starts region only; separate trail button | Two distinct buttons, not auto-parallel | |
| **Other (user free-text)** | Bottom modal sheet with per-region Vector/DEM checkboxes (selection only) + one bottom Download button that starts all selected packages + the trail | ✓ |

**User's choice:** Other — bottom modal sheet redesign (see CONTEXT.md D-05 through D-08). Checkboxes set state only; a single Download button starts everything selected plus the trail; sizes in subtitle like the Settings region screen.
**Notes:** This folded in the Region-download-scope area (per-package vector/DEM checkboxes). Zero regions selected + Download = trail-only (GUARD-03 escape).

---

## Dialog/CTA flow — checkbox defaults

| Option | Description | Selected |
|--------|-------------|----------|
| Vector on, DEM off | Vector pre-checked (basemap coverage), DEM opt-in | ✓ |
| Both on | Maximal completeness, pulls large DEM by default | |
| Both off | Fully explicit, extra taps for the common case | |

**User's choice:** Vector on, DEM off
**Notes:** Mirrors Settings DEM opt-in convention.

---

## Dialog/CTA flow — button action / progress

| Option | Description | Selected |
|--------|-------------|----------|
| Dismiss + background | Sheet closes; downloads background via existing engines + existing toasts/notifications | ✓ (refined) |
| Stay open with progress | Sheet remains showing per-download progress; owns orchestration | |
| **Other (user free-text)** | Dismiss + background, with a download notification showing a **unified** progress bar | ✓ |

**User's choice:** Other — "Dismiss + background. Create download notification with unified progress bar."
**Notes:** One notification aggregating trail + selected region packages into a single progress bar (CONTEXT.md D-10).

---

## Catalog freshness

| Option | Description | Selected |
|--------|-------------|----------|
| Local-only | Check locally-stored regions; no network on download tap | ✓ |
| Refresh then check | Fetch /api/v1/regions before checking coverage | |
| Refresh in background, check local now | Instant local check + non-blocking refresh | |

**User's choice:** Local-only
**Notes:** Settings screen already refreshes catalog on open; keeps guard instant and offline-friendly.

---

## Claude's Discretion

- Exact trigger site of the coverage check + sheet (inside the shared `download()` notifier vs. at the two call sites).
- The bbox-intersection helper location/shape and where the missing-region set is computed.
- Visual specifics of the sheet beyond "Settings-region-style rows with sizes" and the unified notification's exact copy/aggregation math.

## Deferred Ideas

None — discussion stayed entirely within phase scope.
