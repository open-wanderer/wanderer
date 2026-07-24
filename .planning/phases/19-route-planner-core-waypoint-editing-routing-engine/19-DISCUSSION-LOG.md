# Phase 19: Route Planner Core — Waypoint Editing & Routing Engine - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-16
**Phase:** 19-route-planner-core-waypoint-editing-routing-engine
**Areas discussed:** Waypoint gesture disambiguation, Auto-routing + profile toggle placement, Blocked segment & retry treatment, Undo/redo affordance, Terminology (route anchors)

---

## Waypoint gesture disambiguation

| Option | Description | Selected |
|--------|-------------|----------|
| Always adds a waypoint | Simplest mental model, matches WAYP-01's literal wording; appends to route end | ✓ |
| Adds only in an explicit "add mode" | User toggles add-mode first, avoids accidental adds | |

**User's choice:** Always adds a waypoint (recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| Marker wins on overlap | Markers get a larger invisible hit-radius (matches existing 32px marker + proximity-nudge pattern), checked first | ✓ |
| Nearest-feature-wins | Screen-space distance comparison between nearest marker and nearest segment | |

**User's choice:** Marker wins on overlap (recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| Reuse the same onPan drag pattern | Same GestureDetector.onPanStart/Update/End as existing TrailMarkerLayer drag | ✓ |
| Something different | Live route preview during drag, snap-to-segment, etc. | |

**User's choice:** Reuse the same onPan drag pattern (recommended)
**Notes:** Grounded in `app/lib/components/map/trail_layer.dart`'s existing `TrailMarkerLayer` — proven pattern for marker tap/drag coexisting with native map gestures.

---

## Auto-routing + profile toggle placement

| Option | Description | Selected |
|--------|-------------|----------|
| Top-right map control buttons | Matches existing TrailMap.controls Column pattern, consistent with Phase 20's planned toggle buttons | ✓ |
| Bottom toolbar/app bar | Separate persistent bottom bar/app bar area | |

**User's choice:** Top-right map control buttons (recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| Two separate controls | On/off toggle + separate foot/bike segmented switch | |
| One combined control | Single control cycles off → foot → bike | |
| **(free text)** | "There is no profile switch control. The profile is determined once when the user opens the planner. Afterwards it cannot be changed again." | ✓ |

**User's choice:** Neither preset option — no in-planner profile switch exists at all.
**Notes:** This directly contradicted the locked requirement ROUTE-03 ("User can switch the travel profile (foot/bike) for the whole route; changing it re-routes existing segments") and ROADMAP.md Phase 21's success criterion 2 ("still changeable afterward via the toggle from Phase 19"). Flagged the conflict explicitly:

- Asked whether to keep ROUTE-03 (add a profile switch) or cut it (profile entry-only). **User chose: cut ROUTE-03.**
- Asked whether to formally amend REQUIREMENTS.md/ROADMAP.md now or just note the conflict in CONTEXT.md. **User chose: amend now.**
- Amended `.planning/REQUIREMENTS.md` (removed ROUTE-03 from Phase 19's requirement list and Traceability table, reworded HANDOFF-03, added **PLANNER-07** to the v2/deferred section, added an Out of Scope row, updated coverage counts 16→15), `.planning/ROADMAP.md` (Phase 19 goal/requirements/success-criteria text, Phase 19 one-line roadmap summary, Phase 21 success criterion 2), and `.planning/PROJECT.md` (Active requirements bullet, Context target-features bullet, Out of Scope table).

---

## Blocked segment & retry treatment

| Option | Description | Selected |
|--------|-------------|----------|
| Dashed red/warning-colored line | Distinct dash + warning color, scannable without zooming in | ✓ |
| Same line style, warning icon at midpoint | Normal line style + small warning-icon marker | |

**User's choice:** Dashed red/warning-colored line (recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| Tap the blocked segment itself | Consistent with the existing tap-segment-to-insert gesture; on a blocked segment, tap retries instead | ✓ |
| Snackbar/banner with a Retry button | Persistent banner/snackbar, retries all blocked segments at once | |

**User's choice:** Tap the blocked segment itself (recommended)

---

## Undo/redo affordance

| Option | Description | Selected |
|--------|-------------|----------|
| App bar icon buttons | Two icon buttons in the screen's app bar, always visible, standard placement | ✓ |
| Top-right map controls | Grouped alongside the auto-routing toggle in the top-right control stack | |

**User's choice:** App bar icon buttons (recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| Disabled when empty | Grayed out and non-interactive when the history stack is empty in that direction | ✓ |
| Always enabled, no-op if empty | Buttons stay tappable but do nothing if there's no history | |

**User's choice:** Disabled when empty (recommended)

---

## Terminology (route anchors)

Raised via the "explore more gray areas" prompt after all four selected areas completed, as free text rather than a preset AskUserQuestion:

> "I noticed that you refer to the tapped points in the planner as waypoints. This can lead to confusion, because trails also have waypoints which represent a different data type. Please refer to them as 'route anchors'. Also make sure that route anchors are numbered in ascending order as the user adds more of them in the planner. Note that the numbering must change when a user inserts a new anchor between two existing ones."

**Decision:** In-progress route tap points are called "route anchors" everywhere (code, UI copy, docs) for this phase — distinct from the persisted `Trail`'s `Waypoint` model. Route anchors are numbered in ascending order; inserting an anchor mid-route renumbers everything after the insertion point.

---

## Claude's Discretion

None — all gray areas resolved to explicit user decisions.

## Deferred Ideas

- **PLANNER-07**: mid-session travel-profile switching (formerly ROUTE-03), added to REQUIREMENTS.md's v2/deferred Route Planner section during this discussion. Profile is fixed at entry via HANDOFF-03 instead.
