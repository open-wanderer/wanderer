---
id: SEED-261104-ux5
status: dormant
planted: 2026-11-04
planted_during: Phase 39 (unified-tile-model); captured from /gsd-explore for Phase 45
trigger_when: after the maintainer meeting on map styles
scope: medium
---

# SEED-261104-ux5: Base style picker in the app

Bring the web's base-style choice to the app while keeping the map usable offline.

## Why This Matters

The web offers OpenFreeMap, OpenTopoMap, OpenHikingMap, CyclOSM and Carto; the app has one bundled
wanderer style. Phase 45 deliberately left base styles out pending a maintainer meeting.

Constraints found during exploration:
- Downloaded regions are Protomaps-schema vector tiles. Only styles written against that schema can
  render offline (Protomaps light/dark/white/grayscale/black flavours, or custom ones).
- OpenTopoMap, OpenHikingMap and CyclOSM are third-party rasters; OpenFreeMap and Carto use the
  OpenMapTiles schema. None can read region tiles.
- Protomaps tiles have no contours or hillshade, so a true topo look needs terrain data not in the
  regions today.
- Direction discussed: offer the web's list while online and fall back to the bundled wanderer
  style when `onlineStatusProvider` reports offline. That provider measures backend reachability,
  defaults to `true` on cold start, and a style swap reloads the whole style, so the app's runtime
  layers (trail, route segments, location, Phase 45 overlays/POIs) must re-attach. Navigation and
  recording should not swap style mid-session.

## When to Surface

**Trigger:** after the maintainer meeting on map styles.

## Scope Estimate

**Medium.** One phase: style registry, online/offline fallback, layer re-attachment on swap.

## Breadcrumbs

- `app/lib/provider/map_style_json_provider.dart`
- `app/lib/provider/online_status_provider.dart`
- `web/src/lib/vendor/maplibre-layer-manager/layers.ts` (`baseMapStyles`)
- `.planning/notes/app-map-overlays-pois.md`
