# Phase 16: List & Map Screens on MapLibre - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-09
**Phase:** 16-list-map-screens-on-maplibre
**Areas discussed:** Search trigger model, Unclustered-point tap behavior, is_large trail handling

---

## Search trigger model

| Option | Description | Selected |
|--------|-------------|----------|
| Keep manual button | Matches today's actual behavior byte-for-byte | ✓ |
| Switch to auto-debounce | New UX matching ROADMAP's literal wording, needs its own timing decision | |

**User's choice:** Keep manual button.
**Notes:** ROADMAP.md's "debounced exactly as today" is interpreted as "same manual-trigger mechanism as today," not a new auto-debounce feature — the current app doesn't actually auto-debounce, it uses a manual "Search this area" button.

---

## Unclustered-point tap behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Fetch trail polyline, fit once loaded | Reuse existing trailPolylineProvider pattern already in map_screen.dart | ✓ |
| Instant flyTo then refine | New two-stage camera pattern, more responsive-feeling but unprecedented in codebase | |

**User's choice:** Fetch trail polyline, fit once loaded.
**Notes:** Contrasts with web's simpler fixed flyTo(zoom: 12) — the app is intentionally richer here per ROADMAP.md's explicit "fits the camera to its polyline" wording.

---

## is_large trail handling

| Option | Description | Selected |
|--------|-------------|----------|
| Invisible for now | Matches web's current behavior and PROJECT.md's FUT-01 deferral | ✓ |
| Add placeholder/indicator | Small scope addition beyond current FUT-01 scope | |

**User's choice:** Invisible for now.

---

## Claude's Discretion

- Exact native layer/source ids for cluster circles/count/unclustered-point layers.
- Whether list_detail_map_screen/list_detail_screen reuse WandererMap directly or use a lighter-weight MapLibre host.

## Deferred Ideas

- Auto-debounced re-query on pan/zoom.
- Two-stage instant-flyTo-then-refine camera animation.
- is_large trail placeholder/indicator on the map screen.
- Rendering is_large trails as full polylines (FUT-01, future milestone).
