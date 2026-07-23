# Phase 25: Map Rendering — Region-Based Viewport Pipeline - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-23
**Phase:** 25-map-rendering-region-based-viewport-pipeline
**Areas discussed:** Uncovered-viewport behavior, Region-swap visual feel while navigating, Trail spanning two downloaded regions, Viewport scope: both screens or just navigation

---

## Uncovered-Viewport Behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Blank/empty basemap | No vector sources contribute at all — same failure mode as today's offline trail-map when cellPaths is empty | ✓ |
| Inline banner/warning | Blank basemap plus a visible "No offline data for this area" banner | |
| Attempt online tiles if network is available | Falls back to the existing online mapStyleJsonProvider style when connected | |

**User's choice:** Blank/empty basemap
**Notes:** No new UI surface — this stays a pure offline-rendering pipeline swap.

| Option | Description | Selected |
|--------|-------------|----------|
| Same behavior everywhere | Blank basemap is uniform on both TrailMap and navigation_screen | ✓ |
| Navigation screen gets a distinct offline-gap indicator | Small "no map data here" indicator specifically while navigating | |

**User's choice:** Same behavior everywhere
**Notes:** Navigation continues live GPS/maneuver tracking regardless of basemap tile availability.

---

## Region-Swap Visual Feel While Navigating

| Option | Description | Selected |
|--------|-------------|----------|
| Brief flicker acceptable | Region boundary crossings are infrequent; a short reload flash is an acceptable v1.6 tradeoff | ✓ |
| Seamless required — blocks on spike confirming incremental support | Becomes a phase blocker if the spike finds only full-style-reload is possible | |

**User's choice:** Brief flicker acceptable
**Notes:** Ship whichever composition strategy RENDER-03's spike confirms works; seamless swapping is not a gating requirement.

| Option | Description | Selected |
|--------|-------------|----------|
| Debounce on camera-idle | Matches ARCHITECTURE.md's recommendation and map_screen.dart's existing cluster-bbox debounce precedent | ✓ |
| Recompute continuously during movement | Checks on every camera-move event | |

**User's choice:** Debounce on camera-idle
**Notes:** Avoids recomposing/reloading mid-pan or mid-stride; minimizes Pitfall 5 exposure.

---

## Trail Spanning Two Downloaded Regions

| Option | Description | Selected |
|--------|-------------|----------|
| Both regions render together | localTilePathsForBounds already returns every region overlapping the trail's fixed bbox query | ✓ |
| Only the region containing the trail's center renders | Simpler but leaves part of a boundary-straddling trail's basemap blank | |

**User's choice:** Both regions render together
**Notes:** TrailMap's viewport is fixed for the screen's lifetime — no swap logic needed, matches existing multi-cell trail precedent (15-06).

---

## Viewport Scope: Both Screens or Just Navigation

| Option | Description | Selected |
|--------|-------------|----------|
| Swap logic is navigation_screen-only | TrailMap's bounds are static for the screen's lifetime; viewport scoping only has meaning where the camera moves | ✓ |
| Both screens get the same debounced camera-idle recompute | Uniform implementation even though TrailMap's bounds rarely change | |

**User's choice:** Swap logic is navigation_screen-only
**Notes:** navigation_screen is also the only screen where regions could otherwise accumulate unconditionally, per RENDER-02's own wording.

---

## Claude's Discretion

- The exact composition strategy (incremental `addSource`/`removeSource` vs. full style reload) is resolved by RENDER-03's spike against the installed maplibre 0.3.5 API, not a user preference.
- Whether the spike ships as a standalone throwaway harness or is folded into the first implementation plan's Task 1 is a planning-level call.

## Deferred Ideas

None — discussion stayed entirely within phase scope.
