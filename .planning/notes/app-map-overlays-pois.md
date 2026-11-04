---
title: Map overlays and POIs in the app — decisions
date: 2026-11-04
context: /gsd-explore session on porting the web's map layer picker (style_switcher.svelte) to the app
---

## Scope

- **Base styles are deferred.** Pending a maintainer meeting. The app keeps the bundled
  Protomaps-schema wanderer style. The web's OpenTopoMap/OpenHikingMap/CyclOSM are third-party
  rasters and OpenFreeMap/Carto use the OpenMapTiles schema, so none can render from downloaded
  regions. See the "Offline-capable base styles" seed.
- **Overlays and POIs only, online only.** Waymarked Trails rasters (hiking, cycling, MTB,
  skiing) and the web's ~30 Overpass POI categories. While `onlineStatusProvider` reports
  offline, the layers are hidden and the toggles disabled; the selection is kept and returns on
  reconnect. `onlineStatusProvider` measures backend reachability, not third-party tile hosts —
  acceptable for v1.

## Decisions

- **One global state**, like the web's `MapState`. Picker on the main map, trail detail and
  route planner. Not on navigation/recording in v1 (battery, live Overpass traffic).
- **Overlays** are added at runtime as raster source + layer, inserted below the app's trail and
  route layers (same `style.addSource`/`addLayer` path as `trail_layer.dart`). Runtime layers
  bypass `rewriteStyleForProxy`, so they go straight to the network.
- **POI detail sheet: curated OSM tags.** Address, opening hours, website and phone (tappable),
  wheelchair, elevation for summits/passes, drinking water/fee where relevant; raw tags behind
  "Show all tags". No backend work beyond the POI endpoint. The web popup
  (`createOverpassPopup`) currently shows a raw monospace tag grid; the curated formatting could
  be ported back later.
- **Protomaps POIs rejected as a data source.** Tiles carry only `name` and `kind`: no arbitrary
  tag filtering and no detail data. Overpass stays the only POI backend.
- **POI icons** are SVGs on a coloured background on the web; MapLibre needs bitmaps, so the app
  rasterizes each once before registering it.

## Overpass rate limits

The public instance's policy ([OSM wiki: Overpass API](https://wiki.openstreetmap.org/wiki/Overpass_API)):
~10,000 queries and 1 GB per day, divided by 100 for regular use; no parallel scripts; pause
30 s on HTTP 429/406; identify the client via `User-Agent`.

Today `web/src/lib/server/overpass.ts` is a pass-through with no cache, no `User-Agent`, and
immediate retries only on thrown errors. All users of an instance share one server IP. The web
client's query is a raw QL string with float bboxes, so a server cache keyed on it would almost
never hit.

Plan:
1. Server: structured `/api/v1/pois?tile=x,y&category=...` endpoint. The server builds the
   query, caches per (tile, category) for days in persistent storage, single-flights concurrent
   misses, sends one upstream request at a time, backs off 30 s on 429/406 while serving stale
   cache, and sets a wanderer `User-Agent`. The web Overpass layer moves onto it.
2. App: persistent per-(tile, category) cache in ObjectBox with a TTL; fetch only after the
   camera settles, from zoom 12, with a per-screen tile cap.
3. Operators: document pointing `OVERPASS_API_URL` at an own or keyed instance for larger
   installs.

Server and API changes ship as a **new PR stacked on #1397** (the app-merge 1394–1397 stack),
not inside #1397 itself.

## Komoot comparison

- Komoot's route planner shows/hides POIs and Highlights from a categories menu inside search,
  several at once ([komoot.com/help/routeplanner](https://www.komoot.com/help/routeplanner)).
- Unresolved (non-authoritative sources only — support centre returned 403): app "Map layers →
  Map content" toggles; POIs as fixed OSM categories; offline POIs listed as upcoming on Apple
  Watch; sport overlays as Premium.
- A Komoot POI API sample (pasted by the user) returns a curated record: formatted `details`
  (website, phone, address, wheelchair), a Wikipedia extract, and Commons photos with licence
  and attribution. In that sample, name/details were for Hotel Bonverde but the text and all
  images were for the nearby Berlin-Wannsee station — enrichment apparently matched by proximity.
  **If wanderer ever enriches POIs, follow only explicit OSM `wikidata`/`wikipedia`/
  `wikimedia_commons` tags, never nearest article.**

## Open

- Tapping a POI in the route planner to add it as an anchor (the web trail editor does this via
  `buildPoiAnchorAction`) was raised but not decided. Stretch item.
