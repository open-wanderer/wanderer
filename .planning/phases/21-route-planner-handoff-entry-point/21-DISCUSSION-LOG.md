# Phase 21: Route Planner Handoff & Entry Point - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-17
**Phase:** 21-route-planner-handoff-entry-point
**Areas discussed:** Entry-point flow, Handoff trigger & validation, Draft Trail contents

---

## Entry-point flow

| Option | Description | Selected |
|--------|-------------|----------|
| AlertDialog, two buttons | showDialog<String> with Hike/Bike buttons, closest existing precedent | |
| Modal bottom sheet, two cards | Two _SourceActionCard-style tappable cards, consistent with trail_source_select_screen | ✓ |
| Segmented control in a compact dialog | AlertDialog with segmented toggle + Continue button | |

**User's choice:** Modal bottom sheet, two cards
**Notes:** Matches the visual language of the screen the user just came from.

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — back/tap-outside dismisses | Barrier-dismissible, no navigation on cancel | ✓ |
| No — must choose one to proceed | Non-dismissible, forces a choice | |

**User's choice:** Yes — dismissible, no forced choice

| Option | Description | Selected |
|--------|-------------|----------|
| Port allowAutoGeolocate to Flutter Settings, gate GPS on it | Mirrors web's Settings.behavior field; closes the 2026-07-16 gap flag | ✓ |
| Use GPS unconditionally | Ignores the setting entirely | |
| Skip GPS entirely — fixed/last-known fallback | Avoids porting work, never auto-centers | |

**User's choice:** Port allowAutoGeolocate, gate GPS on it (recommended option)
**Notes:** Directly resolves the standing memory note about the Flutter Settings model gap.

---

## Handoff trigger & validation

| Option | Description | Selected |
|--------|-------------|----------|
| App bar action | Added to existing AppBar.actions alongside undo/redo | ✓ (modified) |
| Primary button in the route anchor sheet | Docked in Phase 20's tabbed sheet | |
| Floating action button on the map | New FAB overlay | |

**User's choice:** "Put it in the appbar. Move the undo/redo buttons into the map actions instead."
**Notes:** User's own free-text modification — Finish takes the app bar slot, undo/redo relocates to the top-right map controls column.

| Option | Description | Selected |
|--------|-------------|----------|
| Require ≥2 anchors, disable/block otherwise | Mirrors Phase 20 D-13's <2-anchor empty-state precedent | ✓ |
| Allow handoff with any anchor count ≥1 | Maximally permissive | |

**User's choice:** Require ≥2 anchors (recommended option)

| Option | Description | Selected |
|--------|-------------|----------|
| Proceed without elevation, hand off silently | Matches Phase 20 D-11 best-effort precedent | ✓ |
| Block handoff, show error with retry | Guarantees elevation but risks a dead-end | |

**User's choice:** Proceed without elevation, hand off silently (recommended option)

---

## Draft Trail contents

| Option | Description | Selected |
|--------|-------------|----------|
| Sequential "Waypoint N" | Every anchor becomes a named Waypoint in route order | |
| Only start/end named, rest blank | First/last anchor → "Start"/"End", rest empty | |
| Leave all names blank | Every Waypoint gets name: "" | |
| *(user correction)* Route anchors never become Waypoints | Anchors stay strictly in the route planner | ✓ |

**User's choice:** "Route anchors do not become trail waypoints. They stay strictly in the route planner."
**Notes:** Overrides HANDOFF-01's original "synthesized GPX + named waypoints" wording. A follow-up clarifying question confirmed the draft Trail's `waypoints` list is empty — no Waypoint records at all, just the GPX track. REQUIREMENTS.md and ROADMAP.md were amended in this session (SCOPE CHANGE).

| Option | Description | Selected |
|--------|-------------|----------|
| Pre-fill category from hike/bike choice | Reverse of costingForCategory's heuristic | ✓ |
| No pre-fill — defaults only | Same as a plain GPX import | |

**User's choice:** Pre-fill category from hike/bike choice (recommended option)

---

## Claude's Discretion

- Exact category ID/lookup for hike→category / bike→category mapping (D-08) — no existing reverse-lookup helper found.
- Bottom sheet card copy/icons for the hike/bike dialog (D-01).
- Exact `initialCenter` fallback value when `allowAutoGeolocate` is false or GPS fails (D-03) — e.g. last map camera position vs. a fixed default.
- Whether `allowAutoGeolocate` needs a settings-screen toggle in this phase, or ships model-only (D-03 leaves this open).

## Deferred Ideas

None — discussion stayed within phase scope. The `allowAutoGeolocate` port and the waypoint-scope correction were reconciled directly in REQUIREMENTS.md/ROADMAP.md rather than deferred.
