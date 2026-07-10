# Phase 17: Navigation on MapLibre - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-09
**Phase:** 17-navigation-on-maplibre
**Areas discussed:** Compass button behavior, Compass visibility, Recenter state coupling, Location puck visuals

---

## Compass button behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Keep today's toggle | Override MapCompass's onPressed to toggle trackLocation(trackBearing: gps/none) — same UX as today, backed by the native API. | ✓ |
| Native reset-only + separate heading-up control | Compass always resets to north; heading-up follow becomes a separate button or ties into recenter instead. | |

**User's choice:** Keep today's toggle (Recommended)
**Notes:** Preserves the validated "north-up/heading-up toggle" requirement from PROJECT.md exactly.

---

## Compass visibility

| Option | Description | Selected |
|--------|-------------|----------|
| Always visible | Matches today's navigation_screen behavior exactly. | ✓ |
| Hide when facing north | Match Phase 16's map-screen convention (hideIfRotatedNorth:true). | |

**User's choice:** Always visible (Recommended)
**Notes:** During turn-by-turn the compass is a primary orientation control, not an occasional map-browsing aid — Phase 16's hide-when-north rationale doesn't apply here.

---

## Recenter state

| Option | Description | Selected |
|--------|-------------|----------|
| Restore prior heading-up state | trackLocation(trackLocation:true, trackBearing: previous mode) — matches today's independent _followEnabled/_headingUp booleans. | ✓ |
| Always recenter to north-up | Recenter always resets bearing to north regardless of prior mode. | |

**User's choice:** Restore prior heading-up state (Recommended)
**Notes:** If heading-up was active before dragging away, recenter brings you back into heading-up — not a plain north-up reset.

---

## Puck visuals

| Option | Description | Selected |
|--------|-------------|----------|
| Sensible defaults | Enable pulse + accuracy animation (maplibre's defaults) — standard GPS-puck behavior. | ✓ |
| Minimal / no pulse | Disable pulse and accuracy animation for a plainer, closer-to-today's-static-dot look. | |

**User's choice:** Sensible defaults, don't worry about it (Recommended)
**Notes:** CORE-07 mandates the native enableLocation()/trackLocation() puck itself (no custom color/size API), so the visual changes from today's custom blue dot regardless — accepted as a deliberate, non-regression change.

---

## Claude's Discretion

- Exact camera animation durations for recenter/heading-up transitions (short, non-`Duration.zero` per the Phase 16 checkpoint lesson).
- Whether the legacy flutter_map `TrailLayer` widget and `pm_tile_provider.dart` are physically deleted this phase or left dead until Phase 18's cleanup pass.
- Exact widget/provider structure for wiring `enableLocation`/`trackLocation` into the screen's lifecycle.
- Offline vector tile source wiring (porting `WandererMap`'s `pmtiles://file://` + `rewriteStyleForOffline` pattern).

## Deferred Ideas

- Compass icon redesign to match native look-and-feel more closely (or vice versa) — not raised as a concern.
