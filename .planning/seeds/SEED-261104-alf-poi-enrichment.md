---
id: SEED-261104-alf
status: dormant
planted: 2026-11-04
planted_during: Phase 39 (unified-tile-model); captured from /gsd-explore for Phase 45
trigger_when: once the Phase 45 curated POI sheet ships
scope: medium
---

# SEED-261104-alf: POI enrichment

Add a Wikipedia extract and a Wikimedia Commons photo, with licence attribution, to the app's POI
detail sheet.

## Why This Matters

Phase 45's sheet shows curated OSM tags only. A short description and a photo make a POI
recognisable when planning, which is the main thing Komoot's POI records add over raw tags.

Hard rule: source enrichment **only** from explicit OSM `wikidata` / `wikipedia` /
`wikimedia_commons` tags, never by matching the nearest article. A Komoot POI API sample showed
the failure mode: name and details for Hotel Bonverde, but the text and all 15 photos were for the
neighbouring Berlin-Wannsee station.

## When to Surface

**Trigger:** once the Phase 45 curated POI sheet ships.

## Scope Estimate

**Medium.** Extra API calls (Wikidata/Wikipedia/Commons), an image proxy or cache, and correct
licence and attribution display per image.

## Breadcrumbs

- `.planning/notes/app-map-overlays-pois.md` (Komoot comparison)
- `.planning/ROADMAP.md` Phase 45
- `web/src/lib/util/maplibre_util.ts` `createOverpassPopup` (web popup that would benefit too)
