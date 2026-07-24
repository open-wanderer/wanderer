# Phase 27: Legacy Cleanup - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-24
**Phase:** 27-legacy-cleanup
**Areas discussed:** Todo match review, downloadTrail() surgery scope, Legacy sweep trigger & scope, TrailEntity field removal mechanics, Trail-scoped tile-download UI remnants

---

## Todo Match Review

| Option | Description | Selected |
|--------|-------------|----------|
| Leave it out | Not related to legacy tile-download removal or the cleanup sweep — belongs in its own future phase if pursued at all | ✓ |
| Fold it in | Include it as a decision/deferred item in this phase's context anyway | |

**User's choice:** Leave it out
**Notes:** "Way Types & Surfaces breakdown feature (mobile-first)" matched with score 0.6 on generic keyword overlap only ("first, plans, app, trail") — no substantive relevance to legacy tile cleanup.

---

## downloadTrail() Surgery Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Strip tiles, keep one method | Delete tile-download methods and wiring, keep downloadTrail() as one method for photos + waypoint photos + nav-cache; onGeneratingChanged removed entirely | ✓ |
| Split into separate methods | Break into distinct downloadPhotos()/downloadNavCache() calls — more invasive, larger diff | |

**User's choice:** Strip tiles, keep one method
**Notes:** None.

| Option | Description | Selected |
|--------|-------------|----------|
| Remove the callback + branch | Delete onGeneratingChanged parameter, handleGeneratingChanged wiring, and the WR-01 hasSelectedPackages branch in the Phase 26 guard | ✓ |
| Keep the parameter as a no-op | Leave onGeneratingChanged in the signature (always false/unused) to minimize diff | |

**User's choice:** Remove the callback + branch
**Notes:** Nothing left to report as "generating" once tile generation is gone.

| Option | Description | Selected |
|--------|-------------|----------|
| Keep _downloadTracked as-is | Still needed by _downloadPhotos after tile removal — no change needed | ✓ |
| You decide | Let the planner/executor figure out helper retention during implementation | |

**User's choice:** Keep _downloadTracked as-is
**Notes:** None.

---

## Legacy Sweep Trigger & Scope

| Option | Description | Selected |
|--------|-------------|----------|
| App startup, once, via a persisted flag | Check a SharedPreferences flag on launch; run sweep in background if unset, then set flag | |
| Manual trigger from Settings | Add a button/action the user taps | |

**User's choice:** "No sweep necessary. App is not in production" (free-text, overriding both options)

| Option | Description | Selected |
|--------|-------------|----------|
| Delete only library/{trailId}/tiles/ subdirs | Recursively delete tiles/ subdirectory, leave photos/waypoints untouched, clear DB fields | |
| Delete tiles/ + clear DB fields only, skip empty-dir cleanup | Same deletion, skip removing the now-empty parent folder | |

**User's choice:** "No sweep needed. App is not in production." (free-text, overriding both options)

| Option | Description | Selected |
|--------|-------------|----------|
| Fire-and-forget in background | Kick off after startup completes, don't block first screen | |
| Block startup until complete | Await sweep before first screen renders | |

**User's choice:** "No sweep needed. App is not in production." (free-text, overriding both options)

**Follow-up confirmation:**

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, drop CLEAN-02 entirely | Phase becomes code-removal only (CLEAN-01); no sweep code built | ✓ |
| Keep a minimal one-line reference, but no active sweep code | Note orphaned files are expected but acceptable, CLEAN-02 nominally left open | |

**User's choice:** Yes, drop CLEAN-02 entirely
**Notes:** Confirmed explicitly after Claude flagged that this drops both CLEAN-02 (REQUIREMENTS.md) and ROADMAP Phase 27 success criterion #2. User's rationale: pre-production app, no real install base needing cleanup.

---

## TrailEntity Field Removal Mechanics

| Option | Description | Selected |
|--------|-------------|----------|
| Delete outright now | Remove fields from TrailEntity, Trail model, regenerate codegen — no migration path per CLAUDE.md | ✓ |
| Leave fields in schema, just stop writing to them | Keep dead fields for one more phase as a safety net | |

**User's choice:** Delete outright now
**Notes:** Zero remaining readers confirmed via grep before asking.

---

## Trail-Scoped Tile-Download UI Remnants

| Option | Description | Selected |
|--------|-------------|----------|
| Delete showGenerating() entirely | Remove the method and 'Generating map tiles...' string, matching "no dual-run" — conditional on confirming no other callers (e.g. region downloads) at implementation time | ✓ |
| Keep the method, just stop calling it from trail path | Leave showGenerating() in case region downloads or a future phase needs it | |

**User's choice:** Delete showGenerating() entirely
**Notes:** Captured in CONTEXT.md D-07 with an explicit executor instruction to grep for other callers before deleting.

---

## Claude's Discretion

- Exact grep/verification approach for confirming zero remaining references before deleting pmTiles/demPmTiles and showGenerating().
- Whether to regenerate ObjectBox codegen as a single pass or incrementally per file.
- Any other genuinely-dead tile-generation-only strings/assets a full-file read surfaces beyond what this discussion explicitly enumerated.

## Deferred Ideas

- CLEAN-02 (legacy-file cleanup sweep) and ROADMAP Phase 27 success criterion #2 — cut entirely per user decision, not deferred to a later phase.
- "Way Types & Surfaces breakdown feature (mobile-first)" — reviewed, not folded; unrelated open todo.
