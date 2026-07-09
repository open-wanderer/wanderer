# Phase 15: MapLibre Core, Trail Rendering & Offline Parity - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-08
**Phase:** 15-maplibre-core-trail-rendering-offline-parity
**Areas discussed:** Risk-gate spike scope, Directional arrows fate (TRAIL-02), Attribution & scale bar UX (CORE-04), Glyph/sprite caching scope (GLYPH-04, OFFL-01/02)

---

## Risk-gate spike scope

| Option | Description | Selected |
|--------|-------------|----------|
| Physical device required | Matches ROADMAP.md's risk-gate wording literally; avoids simulator false-positives | ✓ |
| Simulator/emulator first, device later | Faster iteration, confirm on device only once promising | |

**User's choice:** Physical device required.

| Option | Description | Selected |
|--------|-------------|----------|
| Local HTTP server in-app | Loopback HTTP server serving cached files if file:// is rejected | |
| Re-scope: skip full offline label parity | Accept basemap-only offline rendering, revisit later | |
| Let me know when we hit it | Don't pre-decide the fallback | ✓ |

**User's choice:** Let me know when we hit it — no fallback pre-decided.

| Option | Description | Selected |
|--------|-------------|----------|
| You test on your device | User runs the spike build, reports pass/fail | ✓ |
| Claude verifies via simulator only | Claude checks what it can without a physical device | |

**User's choice:** You test on your device.
**Notes:** No fallback decided in advance; revisit with the actual failure mode if the spike fails.

---

## Directional arrows fate (TRAIL-02)

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, re-enable on symbol layer | Port arrow-along-path logic to a native symbol layer with icon-rotate | ✓ |
| Drop it — descope TRAIL-02 | Treat as already-abandoned scope, delete instead of reimplement | |

**User's choice:** Yes, re-enable on symbol layer.

| Option | Description | Selected |
|--------|-------------|----------|
| Match old behavior | Same spacing-by-zoom + continuous animation loop | |
| Simplify — static arrows, no animation | Fixed-interval arrows, no crawling/pulsing motion | ✓ |

**User's choice:** Simplify — static arrows, no animation.
**Notes:** Spacing-by-zoom density logic should still be replicated; only the motion is dropped.

---

## Attribution & scale bar UX (CORE-04)

| Option | Description | Selected |
|--------|-------------|----------|
| maplibre's default AttributionButton | Built-in control as-is, zero custom UI work | ✓ |
| Custom-styled to match app UI | Themed attribution control | |

**User's choice:** maplibre's default AttributionButton.

| Option | Description | Selected |
|--------|-------------|----------|
| Bottom corners, out of the way | Scale bar bottom-left, attribution bottom-right, standard convention | ✓ |
| You decide per-screen | Claude places sensibly per existing screen overlays | |

**User's choice:** Bottom corners, out of the way.

---

## Glyph/sprite caching scope (GLYPH-04, OFFL-01/02)

| Option | Description | Selected |
|--------|-------------|----------|
| One shared app-wide cache | Trail download warms the same shared cache used by map rendering | ✓ |
| Per-trail pruned subset | Cache only the ranges a specific trail needs, keyed per-trail | |

**User's choice:** One shared app-wide cache.

| Option | Description | Selected |
|--------|-------------|----------|
| First map open | Lazy-fetch on first map screen need | ✓ (as one of two triggers) |
| Eagerly at app startup | Fetch regardless of whether a map is ever opened | |
| First trail download OR first map open | Two independent triggers for the same cache | (superseded by follow-up question) |

**User's choice:** First map open.

**Follow-up:** OFFL-01 requires trail download to also fetch glyphs/sprite — if a hiker downloads before ever opening a map, does download independently warm the cache?

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — download also warms the cache if cold | Both entry points check/populate the same shared cache | ✓ |
| No — map-open is the only trigger | Leaves an OFFL-01 gap for download-before-map-open ordering | |

**User's choice:** Yes — download also warms the cache if cold.
**Notes:** Final resolution: first map open AND first trail download are both independent triggers for the same shared app-wide cache; whichever happens first performs the fetch, the other is a no-op.

---

## Claude's Discretion

- Exact style-JSON extraction mechanism for STYLE-01 (programmatic dump vs. manual port of the 7,677-line theme literals).
- Exact on-disk storage format/location for the glyph/sprite cache (ObjectBox vs. plain files).
- `CORE-01`'s exact widget-contract preservation details for `WandererMap`.

## Deferred Ideas

- Per-trail pruned glyph caching — rejected in favor of one shared app-wide cache.
- Custom-styled attribution control — rejected in favor of maplibre's default.
- Continuous/animated directional-arrow motion — rejected in favor of static arrows.
- Pre-deciding a `file://` spike-failure fallback — explicitly deferred until/if the spike fails.
