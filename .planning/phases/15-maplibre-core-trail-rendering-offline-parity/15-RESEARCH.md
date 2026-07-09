# Phase 15: MapLibre Core, Trail Rendering & Offline Parity - Research

**Researched:** 2026-07-08
**Domain:** Flutter native MapLibre GL rendering (`maplibre` 0.3.5), offline tile/glyph/sprite caching, MapLibre Style Spec v8
**Confidence:** HIGH on package API surface (verified against installed source at `~/.pub-cache/hosted/pub.dev/maplibre-0.3.5`); MEDIUM-to-LOW on the one thing this phase exists to resolve — whether MapLibre Native resolves `file://` glyph URLs at runtime (the risk gate, intentionally unverifiable without a physical device).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Risk-gate spike (blocks all other Phase 15 work):**
- **D-01:** The `file://` glyph resolution spike must be validated on a **physical device**, not a simulator/emulator — simulators can have different filesystem sandboxing than real devices, which is exactly the false-positive this gate exists to prevent.
- **D-02:** The user (Christian) runs the physical-device spike build himself and reports pass/fail plus any error output. Claude cannot access a physical iOS/Android device.
- **D-03:** No fallback is pre-decided for a spike failure. If maplibre-native rejects `file://` glyph URLs, stop and bring the actual failure mode back to the user for a fresh decision (e.g. local loopback HTTP server, or re-scoping OFFL-04) rather than assuming a fallback now.
- The first plan of this phase MUST be this throwaway spike, proving `file://` resolution against a hand-built minimal style, before any trail-rendering or download-caching work is planned.

**Directional arrows (TRAIL-02):**
- **D-04:** Re-enable the directional-arrow feature for real on a native MapLibre symbol layer (currently 100% dead code behind `showArrows = false`).
- **D-05:** Render **static** arrows at fixed intervals — no crawling/pulsing animation loop. Keep the spacing-by-zoom density logic (denser at higher zoom), just without motion.

**Attribution & scale bar (CORE-04):**
- **D-06:** Use maplibre's default/built-in attribution control as-is — no custom-styled attribution UI. First time the app shows ANY attribution (ODbL obligation not met today).
- **D-07:** Scale bar bottom-left, attribution bottom-right (or maplibre-default corner). Keep clear of compass (top-right on some screens), bottom sheets, elevation-profile panel.

**Glyph/sprite caching scope & timing (GLYPH-04, OFFL-01/02):**
- **D-08:** One shared **app-wide** cache — not a per-trail pruned subset. Cache the full set (all 4 fontstacks, complete 256-range set per fontstack, both light/dark sprite themes) once.
- **D-09:** Fetch is lazy — first triggered on **first map open** (mirrors `mapStyleSourcesProvider`).
- **D-10:** Trail download is a **second, independent trigger** for the same cache-warm fetch. Whichever of {first map open, first trail download} happens first performs the fetch; the other is a no-op against the warm cache.

### Claude's Discretion
- Exact style-JSON extraction mechanism for STYLE-01 (dumping the 7,585/7,677-line `wandererLightTheme`/`DarkTheme` Dart `Map` literals to `.json` assets) — programmatic dump vs. manual port.
- Exact storage location/format for the app-wide glyph/sprite cache (ObjectBox vs. plain files in the app documents directory).
- `CORE-01`'s exact widget-contract preservation details for `WandererMap` (which params/callbacks survive verbatim vs. need signature changes), informed by the Phase-14 `map_coordinate_adapter.dart` boundary at the 4 not-yet-migrated screens.

### Deferred Ideas (OUT OF SCOPE)
- Per-trail pruned glyph caching (rejected in favor of one app-wide cache — D-08).
- Custom-styled attribution control (rejected in favor of maplibre default — D-06).
- Continuous/animated directional-arrow motion (rejected in favor of static — D-05).
- Pre-deciding a `file://` spike-failure fallback (deferred to if/when the spike fails — D-03).
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| STYLE-01 | Wanderer light/dark styles exist as plain `.json` assets, equivalent output to `wandererLightTheme`/`wandererDarkTheme` | The themes are plain `Map<String,dynamic>` Dart literals (7,585 / 7,677 lines) taking one `String tileUrl` arg. A one-off Dart script (`jsonEncode`) dumps them to assets — no hand-transcription. See "Standard Stack" + Pattern 5. |
| STYLE-02 | App injects operator `TILE_SERVER_URL` (via `/map/style-sources`) into the style's `protomaps` source at runtime | Theme puts `tileUrl` at `sources.protomaps.tiles[0]`. Dump with a placeholder token, replace at runtime — or JSON-parse and set the field. Pattern 5. |
| STYLE-03 | Style carries a `glyphs` key; 14 symbol layers render 4 fontstacks incl. data-driven Devanagari | Themes carry NO `glyphs` key today (drawn from bundled fonts by `vector_tile_renderer`). Extraction must ADD `glyphs` from `mapStyleSourcesProvider.glyphUrl`. `text-font` incl. `Noto Sans Devanagari Regular v1` confirmed in theme source. |
| STYLE-04 | Style carries a `sprite` key; `arrow` + route-shield icons render (silently dropped today) | Add `sprite` from `mapStyleSourcesProvider.spriteUrl`. Native GL reads sprite from style; light+dark variants both cached (D-08). |
| GLYPH-04 | App resolves glyph/sprite URLs from server on first use, caches app-wide | Build a `@Riverpod(keepAlive: true)` cache provider on top of existing `mapStyleSourcesProvider`. Pattern 4. |
| CORE-01 | `WandererMap` renders via `MapLibreMap`, same widget contract | Two consumers (`trail_detail_map_screen.dart`, `trail_panel.dart`) + `TrailLayer`. Param-by-param mapping in "Component Responsibilities". |
| CORE-02 | Map honors light/dark theme, switches styles live | `MapController.setStyle(String)` swaps style in place; two full style docs. Pattern 2. |
| CORE-03 | Initial camera fits trail bounds with caller padding | `MapController.fitBounds(bounds, padding, nativeDuration: Duration.zero)` in `onStyleLoaded`; no init-bounds field on `MapOptions`. Pattern 3. |
| CORE-04 | Scale bar + Protomaps/OSM attribution on every map (ODbL) | Built-in `MapScalebar` (default bottom-left) + `SourceAttribution` (default bottom-right) widgets, added to `MapLibreMap.children`. Note: the real class is `SourceAttribution`, NOT `AttributionButton`. |
| TRAIL-01 | GPX track as native line layer with casing (5px route over 2px white) | Two stacked `LineStyleLayer`s over one `GeoJsonSource`. Pattern 1. |
| TRAIL-02 | Directional arrows via native symbol layer (replace dead bearing math) | `SymbolStyleLayer` with `symbol-placement: line` + `symbol-spacing` + `icon-image: arrow` + `icon-rotation-alignment: map`. Native spaces & rotates icons along the line — deletes ALL manual math. Pattern 6. |
| TRAIL-03 | Waypoints as tappable widget markers, preserve `AnimatedScale` + `onWaypointTap` | `maplibre` `WidgetLayer` + `Marker(child:)` with `allowInteraction: true` — keeps the Flutter widget + gesture. Pattern 7. |
| TRAIL-04 | Start/finish pins as widget markers incl. 36px alignment nudge | Same `WidgetLayer`/`Marker` mechanism; screen-distance nudge via `MapController.toScreenLocations`. Pattern 7. |
| TRAIL-05 | Elevation-profile position marker tracks scrub position | Single `Marker` in `WidgetLayer`, point driven by `elevationMarkerPosition` param. Pattern 7. |
| OFFL-01 | Trail download fetches glyph ranges + sprite once; 2nd download reuses cache | App-wide cache (D-08/D-10) — 2nd download is a no-op against warm cache by construction. |
| OFFL-02 | On downloaded trail, rewrite `glyphs` + `sprite` keys to `file://` before handing style to map | Rewrite the parsed style map's `glyphs`/`sprite` string values to `file://<app-docs>/...` before `initStyle`/`setStyle`. **Depends on the spike passing.** |
| OFFL-03 | Downloaded basemap renders from `.pmtiles` via native `pmtiles://`, no network | `pmtiles://file:///<path>.pmtiles` scheme confirmed supported on both platforms (VERIFIED against MapLibre docs). Replaces `MultiPmTilesVectorTileProvider`. |
| OFFL-04 | Downloaded trail renders place-name labels with no network — the offline parity gate | **THE RISK GATE.** Requires `file://` glyph resolution on MapLibre Native. UNVERIFIED — the spike proves it. |
| OFFL-05 | Trail spanning multiple `.pmtiles` cells renders every cell | Native pmtiles source references ONE file. Multi-cell needs either merge-at-download into one archive, or N sources × N layer copies. OPEN QUESTION — see below. |
| OFFL-06 | `lib/vendor/vector_map_tiles/pm_tile_provider.dart` is deleted | Removable once OFFL-03/05 land; `MultiPmTilesVectorTileProvider` no longer referenced. |
</phase_requirements>

## Summary

The `maplibre` 0.3.5 package (already pinned in `app/pubspec.lock`, sha256-verified) is a thin Dart/FFI-JNI wrapper over MapLibre GL Native (Android 13.0, iOS 6.25). Its style-loading contract is the single most important fact for this phase: **`MapOptions.initStyle` and `MapController.setStyle(String)` accept a raw style JSON string**, which the native layer passes straight to `Style$Builder.fromJson` (Android) / `MLNMapView.styleJSON` (iOS). Every URL field inside that JSON — `glyphs`, `sprite`, source `tiles`/`url` — is resolved by MapLibre GL Native itself, not by any Dart code. So the entire offline story reduces to: *can the native SDK resolve the schemes we put in those fields?* For `pmtiles://file://` (basemap, OFFL-03) the answer is a documented **yes**. For `file://` glyph templates (labels, OFFL-04) the answer is **unverified** — which is precisely why the roadmap makes it the risk gate.

All the rendering primitives this phase needs exist in the package as first-class API: `LineStyleLayer`/`SymbolStyleLayer`/`CircleStyleLayer` take raw Style-Spec `layout`/`paint` maps (full passthrough), `GeoJsonSource(data:)` carries the track, `WidgetLayer`+`Marker(allowInteraction: true)` preserves the exact Flutter waypoint widgets and their `AnimatedScale`/tap behavior, and `MapScalebar`+`SourceAttribution` are built-in controls (the additional-context guess "`AttributionButton`" is wrong — the class is `SourceAttribution`, default `bottomRight`; scalebar defaults `bottomLeft`, matching D-07 with zero config). The directional-arrow feature (TRAIL-02) collapses from ~60 lines of dead bearing math to a single `SymbolStyleLayer` with `symbol-placement: line` — MapLibre spaces and rotates icons along the line natively.

The 7,585/7,677-line style themes are plain `Map<String,dynamic>` Dart literals taking one `String tileUrl` argument, so STYLE-01 extraction is a safe programmatic `jsonEncode` dump, not a hand-transcription risk. The one genuine research gap beyond the spike is OFFL-05 multi-cell pmtiles: native pmtiles sources reference a single file, so a trail spanning several cells needs a design decision (merge at download vs. multiple sources) that today's `MultiPmTilesVectorTileProvider` fan-out hides.

**Primary recommendation:** Plan 1 is the throwaway `file://` glyph spike (D-01/D-02) — build it minimal per the "Spike Design" section, hand it to Christian, and **do not plan trail rendering or download caching until it returns PASS**. Everything else in this research is prescriptive and ready once the gate clears.

## Architectural Responsibility Map

This is a mobile (Flutter `app/`) phase. "Tiers" are the app's rendering/storage/service layers, not a web stack.

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Basemap + label + icon rendering | Native GL (MapLibre C++ core via FFI/JNI) | Flutter widget (`MapLibreMap`) | Style JSON is consumed and rendered entirely by native; Dart only hands over the string and declares layers |
| Style JSON assembly (tile/glyph/sprite injection, offline rewrite) | Flutter client (Dart) | Backend `/map/style-sources` (Phase 13, done) | URL sources come from the server endpoint; the app assembles + rewrites the style before handing to native |
| Track / arrow / cluster geometry layers | Native GL style layers | Flutter (declares `layers:` list) | Line/Symbol/Circle layers render on the GL thread for performance |
| Waypoint / pin / elevation markers | Flutter widgets (`WidgetLayer`) | — | Per-marker tap + `AnimatedScale` selection require Flutter widgets, not GL symbol layers (REQUIREMENTS explicitly excludes native symbol waypoints) |
| Glyph/sprite/pmtiles caching | On-device storage (app documents dir) | Riverpod provider (keepAlive) | Mirrors existing `.pmtiles` cache convention; provider shape mirrors `mapStyleSourcesProvider` |
| Camera fit / animation | Native GL (`MapController.fitBounds`/`animateCamera`) | Flutter (invokes in `onStyleLoaded`) | Native camera math; Dart triggers it |
| Attribution / scale bar | Flutter widgets (built-in `SourceAttribution`/`MapScalebar`) | Native (`getAttributions`) | Rendered as Flutter overlays reading native state |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `maplibre` | **0.3.5 (pin exact)** | Native MapLibre GL Flutter binding — `MapLibreMap`, style layers, controls, offline | Already chosen (STATE.md: "Migrate to `maplibre`, not `maplibre_gl` — FFI/JNI bindings, reads our style JSON directly"). Locked in `pubspec.lock` sha256 `581e17a1…`. `[VERIFIED: pubspec.lock + installed source]` |
| `pmtiles` | 1.2.0 | (Interim) Dart pmtiles reader used by the vendor provider being deleted | Already present; **not** needed once native `pmtiles://` lands — but keep until OFFL-06 removes the vendor file. `[VERIFIED: pubspec.lock]` |
| `path_provider` | 2.1.5 | Resolve app documents directory for the glyph/sprite/pmtiles cache | Already present; existing `.pmtiles` cache uses it (STATE.md convention). `[VERIFIED: pubspec.lock]` |
| `riverpod_annotation` / `flutter_riverpod` | (in-tree) | `@Riverpod(keepAlive: true)` cache + style providers | Established pattern (`map_style_sources_provider.dart`, `map_camera_provider.dart`). `[VERIFIED: codebase]` |
| `font_awesome_flutter` | 11.0.0 | Waypoint/pin glyph icons inside `WidgetLayer` markers | Already used by `trail_layer.dart`; ports verbatim. `[VERIFIED: pubspec.lock]` |

### Supporting (already-installed, no new adds)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `geobase` | (re-exported by `maplibre`) | `Geographic`, `LngLatBounds` coordinate types | Already the app's coordinate vocabulary after Phase 14. `maplibre` re-exports it. `[VERIFIED: lib/maplibre.dart export]` |
| `vector_tile_renderer` (flomp fork) | git `ref: main` → `d52dd7d…` | Source of `wandererLightTheme`/`wandererDarkTheme` — read once at extraction time | Only needed by the one-off STYLE-01 dump script; deleted in Phase 18 (CLEAN-02). `[VERIFIED: pubspec.lock resolved-ref]` |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Native `pmtiles://file://` source (OFFL-03) | Keep `MultiPmTilesVectorTileProvider` (Dart pmtiles reader) | Rejected: the whole point of OFFL-06 is to delete the vendor provider; native pmtiles removes the Dart tile-decode cost. But see OFFL-05 multi-cell open question. |
| `SymbolStyleLayer` for waypoints | Native symbol layer for all markers | Rejected by REQUIREMENTS ("Loses per-marker tap handling and selection animation"). Use `WidgetLayer` for waypoints/pins, native symbol only for arrows. |
| Programmatic style dump (STYLE-01) | Hand-transcribe 7,677 lines | Rejected: transcription-error risk on 15k lines. Programmatic `jsonEncode` is exact. |
| `file://` glyphs (OFFL-04) | `asset://` glyphs | `asset://` only addresses **bundled** APK/app-bundle assets, not runtime-downloaded files in the documents dir — cannot serve a lazily-fetched cache. `file://` is the only scheme that fits D-08/D-10. This is why the gate exists. |

**Installation:** No new packages. `maplibre` 0.3.5 already resolved. CLEAN-03 (Phase 18) later replaces the `^0.3.3+2` caret in `pubspec.yaml` with an exact pin — but this phase should pin exact on first real use per D/STATE.md ("Pin the exact version on first add").

**Version verification (2026-07-08):** `pubspec.lock` resolves `maplibre` to `0.3.5` (sha256 `581e17a1eca80b2be555b00acc03afff22464efd0712d68fcc46be56efa61e59`, pub.dev). CHANGELOG 0.3.5 bundles MapLibre Native Android **13.0** and iOS **6.25**. PMTiles landed Android side at Native 11.8.0 (< 13.0, so present). `[VERIFIED: installed CHANGELOG]`

## Package Legitimacy Audit

> This phase installs **no new external packages**. Every dependency it uses is already resolved in `app/pubspec.lock` from prior phases/milestones. slopcheck (npm/PyPI-oriented) does not apply to Dart pub; the pub.dev lockfile with sha256 pins is the equivalent legitimacy control.

| Package | Registry | Age | Source Repo | Verdict | Disposition |
|---------|----------|-----|-------------|---------|-------------|
| `maplibre` 0.3.5 | pub.dev | Mature (MapLibre org) | github.com/maplibre/flutter-maplibre | sha256-pinned in lock | Approved (pin exact, D/CLEAN-03) |
| `pmtiles` 1.2.0 | pub.dev | Existing | github.com/protomaps | sha256-pinned | Approved (removed with OFFL-06) |
| `path_provider` 2.1.5 | pub.dev | Flutter-team maintained | flutter/packages | sha256-pinned | Approved |
| `vector_tile_renderer` (flomp fork) | git | Fork, existing override | github.com/flomp/dart-vector-tile-renderer | resolved-ref pinned `d52dd7d…` | Approved (read-only at extraction; deleted Phase 18) |

**Packages removed due to slop verdict:** none.
**Packages flagged suspicious:** none. The only non-registry dependency (`flomp/dart-vector-tile-renderer`) is a pre-existing, already-vetted `dependency_overrides` git fork, read only by the one-off extraction script and slated for deletion in Phase 18.

## Architecture Patterns

### System Architecture Diagram (Phase-15 map render + offline path)

```
                         ┌─────────────────────────────────────────┐
   app theme change ────▶│  mapStyleProvider (Riverpod, keepAlive)  │
                         │  1. watch themeMode → light|dark          │
                         │  2. await mapStyleSourcesProvider         │◀── GET /map/style-sources
                         │     {tileUrl, glyphUrl, spriteUrl}        │    (Phase 13, done)
                         │  3. load asset wanderer_{theme}.json      │◀── STYLE-01 asset
                         │  4. inject tileUrl (STYLE-02),            │
                         │     glyphs=glyphUrl (STYLE-03),           │
                         │     sprite=spriteUrl (STYLE-04)           │
                         │  5. IF offline: rewrite glyphs/sprite →   │◀── glyphSpriteCacheProvider
                         │     file://<docs> (OFFL-02); protomaps    │    (GLYPH-04, app-docs dir)
                         │     tiles → pmtiles://file://<cell>       │    (OFFL-03/05)
                         └───────────────────┬───────────────────────┘
                                             │ style JSON String
                                             ▼
                    ┌───────────────────────────────────────────────┐
                    │  WandererMap (MapLibreMap widget, CORE-01)      │
                    │  initStyle: <json string>                       │
                    │  onStyleLoaded → fitBounds(trail.bounds) CORE-03│
                    │  setStyle(newJson) on theme swap        CORE-02 │
                    │                                                 │
                    │  layers: [                                      │
                    │    GeoJsonSource(track)                         │
                    │    ├ LineStyleLayer casing  (white, w=9) TRAIL-1│
                    │    ├ LineStyleLayer route    (#3549bb, w=5)     │
                    │    └ SymbolStyleLayer arrows (placement:line)   │──TRAIL-2
                    │  ]                                              │
                    │  children: [                                    │
                    │    WidgetLayer[ waypoint / start / finish /     │──TRAIL-3/4/5
                    │                 elevation markers ]             │
                    │    MapScalebar(bottomLeft)               CORE-4 │
                    │    SourceAttribution(bottomRight)               │
                    │    Column(controls, topRight)                   │
                    │  ]                                              │
                    └───────────────────┬───────────────────────────┘
                                        │ raw JSON string (starts with '{')
                                        ▼
          ┌──────────────────────────────────────────────────────────┐
          │  MapLibre GL Native  (Android 13.0 / iOS 6.25, via FFI/JNI)│
          │  Style$Builder.fromJson / MLNMapView.styleJSON            │
          │  resolves every URL field itself:                         │
          │   • protomaps tiles: pmtiles://file://…  → local pmtiles  │  ✅ documented
          │   • glyphs: file://…/{fontstack}/{range}.pbf → ???        │  ⚠️ RISK GATE (OFFL-04)
          │   • sprite: file://…/sprite → local png/json              │  ⚠️ same scheme, same gate
          └──────────────────────────────────────────────────────────┘
```

### Recommended Project Structure
```
app/lib/
├── components/base/wanderer_map.dart      # CORE-01: MapLibreMap-based, same public API
├── components/map/trail_layer.dart        # rewritten: GeoJsonSource + Line/Symbol layers + WidgetLayer markers
├── provider/
│   ├── map_style_provider.dart            # rewritten: emits style JSON String (not vtr Style)
│   ├── map_style_sources_provider.dart    # unchanged (Phase 13)
│   └── glyph_sprite_cache_provider.dart   # NEW GLYPH-04: keepAlive, warms app-wide cache
├── util/
│   └── offline_style_rewriter.dart        # NEW OFFL-02: glyphs/sprite/tiles → file:// / pmtiles://
├── assets/map/
│   ├── wanderer_light.json                # NEW STYLE-01 (generated, checked in)
│   └── wanderer_dark.json                 # NEW STYLE-01 (generated, checked in)
└── tool/extract_map_styles.dart           # NEW one-off dump script (STYLE-01), run once
```
(`lib/vendor/vector_map_tiles/pm_tile_provider.dart` deleted at OFFL-06.)

### Pattern 1: Track line with casing (TRAIL-01)
**What:** Two `LineStyleLayer`s stacked over one `GeoJsonSource`, casing under route.
**Why:** MapLibre has no single-layer "border" — the web `ClusterLayer`/line convention and today's `Polyline(borderStrokeWidth: 2)` both map to a wider under-line.
**Ported values:** route `line-color #3549bb`, `line-width 5`; casing `line-color #ffffff`, `line-width 9` (= 5 + 2×2 to show 2px each side); both `line-cap round`, `line-join round`.
```dart
// Source: maplibre 0.3.5 style-layer API (installed: lib/src/style/layers/line_style_layer.dart)
GeoJsonSource(id: 'trail', data: /* GeoJSON LineString of trail.expand.gpx.allPoints */);
LineStyleLayer(
  id: 'trail-casing', sourceId: 'trail',
  layout: {'line-cap': 'round', 'line-join': 'round'},
  paint: {'line-color': '#ffffff', 'line-width': 9},
);
LineStyleLayer(
  id: 'trail-route', sourceId: 'trail',
  layout: {'line-cap': 'round', 'line-join': 'round'},
  paint: {'line-color': '#3549bb', 'line-width': 5}, // overridable via routeColor
);
```
Layer ORDER matters: casing must be added before route. `[VERIFIED: installed source] [CITED: maplibre.org/maplibre-style-spec/layers/#line]`

### Pattern 2: Live theme swap (CORE-02)
**What:** Rebuild the style JSON for the new brightness and call `MapController.setStyle(newJsonString)` — no widget remount, no flash.
**Verified mechanism:** Android `setStyle` branches on `trimmed.startsWith('{')` → `Style$Builder.fromJson`; iOS sets `_mapView.styleJSON`. `mapStyleProvider` already re-runs on `themeModeProvider` change; wire its output to `setStyle` instead of today's `ObjectKey(style)` remount.
`[VERIFIED: installed maplibre_android map_state.dart:788, maplibre_ios map_state.dart:431]`

### Pattern 3: Initial camera fit (CORE-03)
**What:** `MapOptions` has **no** initial-bounds-fit field (only `initCenter`/`initZoom`). Fit in the `onStyleLoaded` callback.
```dart
// Source: installed lib/src/map_controller.dart:70 (fitBounds signature)
onStyleLoaded: (style) async {
  if (trail.bounds != null) {
    await controller.fitBounds(
      bounds: trail.bounds!,                     // ml.LngLatBounds (Phase 14)
      padding: initialCameraFitPadding,          // default EdgeInsets.all(40)
      nativeDuration: Duration.zero,             // instant for the INITIAL fit (no animation)
    );
  } else {
    await controller.moveCamera(
      center: ml.Geographic(lat: trail.lat ?? 0, lon: trail.lon ?? 0), zoom: 18);
  }
}
```
`fitBounds` takes `EdgeInsets padding` directly — the current `CameraFit.bounds(padding:)` maps 1:1. `[VERIFIED: installed source]`

### Pattern 4: App-wide glyph/sprite cache provider (GLYPH-04, OFFL-01)
**What:** `@Riverpod(keepAlive: true)` provider that, on first read, downloads all 4 fontstacks × 256 ranges + both sprite themes into the app documents dir, then returns the local base path. Idempotent: if files already present, no-op (satisfies OFFL-01 second-download reuse and D-10 dual-trigger).
```dart
// Source: mirrors app/lib/provider/map_style_sources_provider.dart pattern
@Riverpod(keepAlive: true)
class GlyphSpriteCache extends _$GlyphSpriteCache {
  @override
  Future<GlyphSpriteCachePaths> build() async {
    final sources = await ref.watch(mapStyleSourcesProvider.future);
    final dir = await getApplicationDocumentsDirectory();       // path_provider
    // fetch {fontstack}/{range}.pbf for the 4 fontstacks (0..255 ranges),
    // sprite.json/.png/@2x.png light+dark, into dir/map_cache/...
    // skip any file already on disk (idempotent → OFFL-01)
    return GlyphSpriteCachePaths(root: '${dir.path}/map_cache');
  }
}
```
Trigger points (D-09/D-10): call `ref.read(glyphSpriteCacheProvider.future)` on first map open AND at trail-download start. Whichever runs first warms it. **Fontstack count = 4** (Regular, Medium, Italic, Devanagari Regular v1) — confirmed against theme `text-font` expressions. `[VERIFIED: codebase pattern + theme source]`

### Pattern 5: Programmatic style extraction (STYLE-01/02/03/04)
**What:** A one-off Dart CLI (`tool/extract_map_styles.dart`) that imports the theme functions, dumps to JSON with a placeholder tile token, and the phase then hand-adds the `glyphs`/`sprite` keys the themes lack today.
```dart
// Source: theme fn signature verified — Map<String,dynamic> wandererLightTheme(String tileUrl)
import 'dart:convert';
import 'dart:io';
import 'package:vector_tile_renderer/vector_tile_renderer.dart' as vtr;

void main() {
  const token = '__TILE_URL__';
  for (final (name, fn) in [('light', vtr.wandererLightTheme), ('dark', vtr.wandererDarkTheme)]) {
    final map = fn(token);
    // ADD keys the Dart themes omit (they draw text from bundled fonts today):
    map['glyphs'] = '__GLYPH_URL__';   // {fontstack}/{range}.pbf template
    map['sprite'] = '__SPRITE_URL__';
    File('assets/map/wanderer_$name.json')
      .writeAsStringSync(const JsonEncoder.withIndent('  ').convert(map));
  }
}
```
At runtime, `mapStyleProvider` loads the asset, then string-replaces the three tokens (or `jsonDecode` → set `sources.protomaps.tiles[0]`, `glyphs`, `sprite` → re-encode). The themes are pure static literals (no runtime logic), so the dump is lossless. **Critical:** the themes carry NO `glyphs`/`sprite` key today (STATE.md blocker), so extraction is dump-**and-augment**, not dump alone. `[VERIFIED: theme source at resolved-ref d52dd7d]`

### Pattern 6: Directional arrows as native symbol layer (TRAIL-02, D-04/D-05)
**What:** One `SymbolStyleLayer` over the track `GeoJsonSource` with `symbol-placement: line`. MapLibre spaces icons along the line (`symbol-spacing`) and rotates them to the line tangent automatically — **this deletes the entire `_addArrowsAlongPath` bearing-math loop and the `AnimationController`.**
```dart
// Source: maplibre.org/maplibre-style-spec/layers/#symbol (layout passthrough)
SymbolStyleLayer(
  id: 'trail-arrows', sourceId: 'trail', minZoom: 8,   // TRAIL-02 zoom>8 threshold
  layout: {
    'symbol-placement': 'line',
    'icon-image': 'arrow',                 // from sprite (STYLE-04)
    'icon-rotation-alignment': 'map',
    'icon-allow-overlap': true, 'icon-ignore-placement': true,
    // spacing-by-zoom density (px, not meters — convert or use zoom interpolation):
    'symbol-spacing': ['interpolate', ['linear'], ['zoom'],
        8, 100, 10, 60, 12, 50, 14, 45, 16, 40],  // tune to match the meter targets
  },
);
```
**Note the units mismatch:** today's spacing is in **meters** (240/600/2000/5000/10000 m by zoom); `symbol-spacing` is in **screen pixels**. The density *intent* (denser at higher zoom) ports directly, but the exact meter values do not map to pixels 1:1. Recommend: replicate the "denser at higher zoom" feel via a zoom-interpolated pixel spacing, and treat exact spacing as visual-parity-approximate (D-05 already accepts simplification). `[VERIFIED: style-spec] [ASSUMED: exact pixel values need on-device tuning]`

### Pattern 7: Widget markers preserving Flutter interaction (TRAIL-03/04/05)
**What:** `maplibre`'s own `WidgetLayer` + `Marker` (NOT `flutter_map`'s) hosts the existing `_buildCircularMarker` widgets verbatim, including `AnimatedScale` and `GestureDetector`. Set `allowInteraction: true` for tappable waypoints/pins.
```dart
// Source: installed lib/src/widget_layer.dart (Marker: point,size,child,alignment,rotate,flat)
ml.WidgetLayer(
  allowInteraction: true,                       // enables tap on children
  markers: [
    ml.Marker(
      point: ml.Geographic(lon: wp.lon, lat: wp.lat),
      size: const Size(32, 32),                 // TRAIL-03 32×32
      alignment: startAlignment,                // TRAIL-04 nudge reuses this
      child: GestureDetector(
        onTap: () => onWaypointTap?.call(wp),
        child: AnimatedScale(                    // preserve 0.875→1.0, 200ms, easeOutBack
          scale: isSelected ? 1.0 : 0.875,
          duration: const Duration(milliseconds: 200),
          curve: Curves.easeOutBack,
          child: _buildCircularMarker(...),
        ),
      ),
    ),
  ],
);
```
The 36px proximity nudge (TRAIL-04) uses `MapController.toScreenLocations([start,last])` (verified present) to measure screen distance, exactly like today's `camera.latLngToScreenOffset`. The elevation marker (TRAIL-05) is a single non-interactive `Marker` (12×12, `allowInteraction:false`). `[VERIFIED: installed source]`

### Anti-Patterns to Avoid
- **Remounting the map on theme change** (`key: ObjectKey(style)` as today): use `setStyle` instead — CORE-02 requires "no reload, no flash."
- **`asset://` for the runtime glyph cache:** `asset://` reads bundled build assets only; the cache is downloaded at runtime → must be `file://`. Using `asset://` would silently fail for downloaded trails.
- **Native symbol layer for waypoints:** loses tap + `AnimatedScale` (REQUIREMENTS explicitly excludes this). Native symbol only for arrows.
- **Hand-transcribing the style JSON:** 15k lines — use the dump script.
- **Assuming one `pmtiles://` source covers a multi-cell trail:** it does not (see OFFL-05 open question).
- **Planning trail rendering before the spike returns PASS:** violates D-01/D-03 and risks building on a broken offline-label foundation.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Arrow spacing + rotation along line | The `_addArrowsAlongPath` bearing loop (`SphericalGreatCircle`, `initialBearingTo`) | `SymbolStyleLayer` `symbol-placement: line` + `symbol-spacing` | Native does interpolation, spacing, tangent rotation, and overlap collision on the GL thread |
| Attribution UI | Custom attribution widget | `SourceAttribution` (built-in) | D-06 mandates the default; reads `getAttributions()` from native automatically |
| Scale bar | Custom `CustomPainter` scale | `MapScalebar` (built-in) | Handles meters-per-pixel-at-latitude math internally |
| Style JSON serialization | Manual string building | `dart:convert` `jsonEncode` on the theme `Map` | Lossless; escapes correctly |
| Local pmtiles tile decode | `MultiPmTilesVectorTileProvider` fan-out | Native `pmtiles://file://` source | Deletes vendor code (OFFL-06); no Dart-side tile parsing |
| Camera bounds fit | Manual zoom/center math | `MapController.fitBounds(bounds, padding)` | Matches `CameraFit.bounds` semantics natively |

**Key insight:** MapLibre GL Native is a full Style-Spec renderer. Almost everything this phase "ports" is actually a *deletion* of Flutter-side code (bearing math, tile fan-out, marker layers) in favor of declarative style layers the native engine already implements. The only things that stay in Flutter are the interactive widget markers.

## Runtime State Inventory

> This is a rendering-engine migration with an on-disk cache and downloaded-trail artifacts. Runtime state matters.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | Existing downloaded-trail `.pmtiles` cells in the app documents dir (v1.1). Their internal tile schema (Protomaps) is UNCHANGED (REQUIREMENTS "Out of Scope": not switching tile generation). NEW: glyph `.pbf` ranges + sprite sheets to be written to the same dir (GLYPH-04). | No migration of existing pmtiles. Add glyph/sprite cache alongside; addressed by `file://` after spike passes. |
| Live service config | Operator `TILE_SERVER_URL` + glyph/sprite override — already served by `/map/style-sources` (Phase 13). No new server config. | None — consume existing endpoint. |
| OS-registered state | None — no OS-level registrations (no task scheduler, no background services introduced this phase). | None — verified: phase is in-app rendering + cache only. |
| Secrets/env vars | None new. Tile key handling stays inside the operator's tile URL template (unchanged). | None. |
| Build artifacts / installed packages | NEW checked-in assets `assets/map/wanderer_{light,dark}.json` must be added to `pubspec.yaml` `flutter/assets`. The generated JSON is a build input, regenerate via `tool/extract_map_styles.dart` if the flomp theme fork changes before Phase 18 deletes it. | Register assets in `pubspec.yaml`; document regeneration command. |

**Nothing found in OS-registered / secrets categories:** confirmed by scoping — this phase touches only in-app map rendering and the app-documents cache.

## Common Pitfalls

### Pitfall 1: `file://` glyph resolution silently returning blank labels (THE RISK GATE)
**What goes wrong:** Native accepts the style JSON (no throw), the basemap renders, but place labels are blank or show tofu boxes — because the SDK either didn't resolve the `file://` glyph template per-range, or resolved but found nothing, or requires `asset://`.
**Why it happens:** MapLibre mobile docs emphasize `asset://` for local resources; `file://` for a runtime glyph *template* (`file:///…/{fontstack}/{range}.pbf`) is undocumented for glyphs (it IS documented for pmtiles and for the style doc itself). The failure is silent — no exception.
**How to avoid:** The Plan-1 spike. Build minimal, run on a physical device in airplane mode, watch for labels. If they don't render, capture the native log (Android logcat / iOS console) for the actual resource-load error and return it to the user (D-03).
**Warning signs:** basemap renders but no text; native log lines mentioning glyph/resource 404 or "unsupported scheme."

### Pitfall 2: Multi-cell pmtiles rendering only one cell (OFFL-05)
**What goes wrong:** A trail spanning 3 downloaded `.pmtiles` cells renders only the cell that happens to back the single `pmtiles://` source; the rest of the basemap is blank.
**Why it happens:** A native `pmtiles://file://` source points at ONE archive. Today's `MultiPmTilesVectorTileProvider` fans one logical source across N archives — native has no equivalent fan-out.
**How to avoid:** Decide the strategy up front (see Open Questions): (a) merge cells into one archive at download time, or (b) N sources + N duplicated style layers. (a) is far simpler given 14 symbol layers × N cells for (b).
**Warning signs:** partial basemap on trails whose bounds cross a cell boundary.

### Pitfall 3: Theme swap flashing / remounting the map
**What goes wrong:** Using `ObjectKey(style)` (as today) rebuilds the whole `MapLibreMap`, causing a visible flash and camera reset on every light/dark toggle.
**How to avoid:** `MapController.setStyle(newJson)` in place; keep the widget mounted. CORE-02 explicitly requires "switching styles live … no reload, no flash."

### Pitfall 4: Losing the `glyphs`/`sprite` keys during extraction
**What goes wrong:** Dumping the theme to JSON produces a valid style that still renders NO labels online, because the Dart themes never had a `glyphs` key (they used bundled fonts).
**How to avoid:** Extraction must ADD `glyphs` + `sprite` (Pattern 5). Verify the generated JSON has both top-level keys before wiring it up.

### Pitfall 5: `symbol-spacing` meter/pixel unit confusion (TRAIL-02)
**What goes wrong:** Copying today's 240/600/2000 m spacing values into `symbol-spacing` (which is pixels) yields wildly wrong arrow density.
**How to avoid:** Treat spacing as pixel-based zoom interpolation; tune on-device. D-05 accepts this simplification.

## Code Examples

### Loading a raw style JSON string (the whole offline story hinges on this)
```dart
// Source: installed maplibre_android/lib/src/map_state.dart:788, maplibre_ios:431/443
// Native branches on the first char: '{' => raw JSON => Style$Builder.fromJson / styleJSON.
// So glyphs/sprite/tiles URLs inside the JSON are resolved by MapLibre GL Native itself.
MapLibreMap(
  options: MapOptions(initStyle: styleJsonString /* starts with '{' */),
  onStyleLoaded: (style) => controller.fitBounds(bounds: trail.bounds!, padding: pad),
  layers: [ /* GeoJsonSource + Line/Symbol layers */ ],
  children: [
    const ml.MapScalebar(),                       // default bottomLeft, metric  (CORE-04/D-07)
    const ml.SourceAttribution(),                 // default bottomRight          (CORE-04/D-06/D-07)
    ml.WidgetLayer(allowInteraction: true, markers: [...]), // TRAIL-3/4/5
    Align(alignment: Alignment.topRight, child: Column(children: controls)), // preserved
  ],
);
```

### Local pmtiles basemap source (OFFL-03) — VERIFIED scheme
```dart
// Source: maplibre.org/maplibre-native/{android,ios} PMTiles docs (WebFetch verified)
// In the style JSON, set the protomaps source tiles/url to:
//   "pmtiles://file:///data/user/0/.../map_cache/cell_x.pmtiles"
// Remote form is pmtiles://https://… ; asset form pmtiles://asset://…
// The rest of the URL (after pmtiles://) is a normal file:// path to the local archive.
```

### Built-in controls (CORE-04) — corrected class names
```dart
// Source: installed lib/src/ui/{map_scalebar,source_attribution}.dart
const ml.MapScalebar(alignment: Alignment.bottomLeft, units: ml.ScaleBarUnit.metric);
const ml.SourceAttribution(alignment: Alignment.bottomRight); // NOT "AttributionButton"
```

## Component Responsibilities — `WandererMap` param → MapLibre mapping (CORE-01)

| Current param/callback | MapLibre equivalent | Notes |
|------------------------|---------------------|-------|
| `trail` | source data + `fitBounds` target | GeoJSON from `trail.expand.gpx.allPoints` |
| `mapController` (`MapController` from flutter_map) | `ml.MapController` | **Signature change** — different type. Consumers `trail_detail_map_screen.dart` + `trail_panel.dart` must update. |
| `disabled` | `MapOptions.gestures = MapGestures.none()` | `MapGestures` replaces `InteractionOptions` |
| `offline` | selects style rewrite branch (pmtiles://file:// + file:// glyphs) | Replaces `MultiPmTilesVectorTileProvider` init |
| `controls` | `children:` `Column` topRight | Ports verbatim |
| `elevationMarkerPosition` | single `WidgetLayer`/`Marker` | Pattern 7 |
| `initialCameraFitPadding` | `fitBounds(padding:)` | 1:1, default `EdgeInsets.all(40)` |
| `showTrail` | conditionally add track layers | — |
| `showLocation` | **defer** — location puck is CORE-07/Phase 17 | Confirm during planning if an interim location display is needed (CONTEXT notes `CurrentLocationLayer`/`foregroundPositionStreamProvider` wiring); `enableLocation`/`trackLocation` exist on `MapController` but are Phase-17 scope. |
| `selectedWaypoint` | drives `AnimatedScale` in marker | Pattern 7 |
| `onTap` | `onEvent` → `MapEventClick` | `MapEvent` type changes |
| `onMapEvent` | `onEvent` callback | Note: current code has a bug (`(e) => widget.onMapEvent` returns the field, never calls it) — fix on port |
| `onWaypointTap` | `GestureDetector.onTap` in marker | Pattern 7 |

**Blast radius:** 2 consumer screens (`trail_detail_map_screen.dart`, `trail_panel.dart` — CONTEXT.md's "sole consumer" is slightly off, `trail_panel.dart` also constructs `WandererMap`) + `TrailLayer` rewrite. `[VERIFIED: grep]`

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `flutter_map` + `vector_map_tiles` + `vector_tile_renderer` (Dart CPU tile rasterization) | `maplibre` native GL rendering | This milestone | GPU rendering, real sprite/glyph support, live style swap |
| pmtiles decoded in Dart (`MultiPmTilesVectorTileProvider`) | native `pmtiles://file://` source (MapLibre Native ≥11.8 Android) | maplibre 0.3.x | Deletes vendor code (OFFL-06) |
| Manual arrow bearing math (dead) | `symbol-placement: line` | This phase | ~60 lines deleted |
| No attribution (ODbL non-compliant) | built-in `SourceAttribution` | This phase | First-ever attribution UI |

**Deprecated/outdated:**
- `VectorSource.sourceLayer` and `PmTilesVectorTileProvider.silenceTileNotFound` are `@Deprecated` in installed sources — don't use.
- The additional-context term "`AttributionButton`" does not exist in this package; the class is `SourceAttribution`.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | MapLibre GL Native resolves a `file://` glyph URL **template** (`file:///…/{fontstack}/{range}.pbf`) at runtime for labels | OFFL-04 / Risk Gate | HIGH — this is the milestone gate; the spike exists to convert this from ASSUMED to verified. If false, OFFL-04 needs a different strategy (D-03). |
| A2 | The same `file://` scheme works for the local `sprite` base URL | STYLE-04/OFFL-02 | MEDIUM — if glyphs work but sprite doesn't, icons (arrow/shields) stay blank offline. Spike should test sprite too. |
| A3 | Exact arrow `symbol-spacing` pixel values need on-device tuning to feel like today's meter spacing | TRAIL-02 | LOW — visual polish only; D-05 accepts simplification. |
| A4 | Multi-cell offline needs merge-at-download (vs. N sources) | OFFL-05 | MEDIUM — wrong choice means a rework of the download-time tile handling. Confirm with a 2-cell trail test. |
| A5 | `showLocation` can be deferred (no interim puck needed in Phase 15) | CORE-01 | LOW-MEDIUM — if `trail_detail_map_screen` relies on live location today, an interim display may be needed. Confirm at planning. |
| A6 | The `glyphs`/`sprite` keys are absent in the current themes and must be added at extraction | STYLE-01/03/04 | LOW — verified in theme source; stated for the planner's awareness. |

**These ASSUMED items — especially A1/A2 — must be confirmed by the physical-device spike before any dependent task is planned as "safe."**

## Open Questions (RESOLVED — see plan pointers below)

1. **`file://` glyph (+ sprite) resolution on MapLibre Native mobile — THE RISK GATE (A1/A2).** *(RESOLVED: gated by the throwaway spike in `15-01-PLAN.md`, physical-device verified before any dependent plan runs.)*
   - What we know: native accepts raw style JSON and resolves URL fields itself; `pmtiles://file://` local is documented; Android wraps `/path` style docs as `file://`; MapLibre Native has a LocalFileSource that handles `file://`.
   - What's unclear: whether the glyph *template* per-range fetch resolves from a runtime `file://` path (vs. requiring bundled `asset://`). Mobile docs lean on `asset://` for local resources.
   - Recommendation: **Plan 1 spike (see Spike Design). Do not plan dependent work until PASS.** On fail, capture the native resource-load error and return to user (D-03).

2. **Multi-cell offline basemap (OFFL-05, A4).** *(RESOLVED: investigated and decided in-task in `15-06-PLAN.md` Task 1 — `pmtiles` 1.2.0 is read-only and `generator.go`/`grid.go` produce one `.pmtiles` per 0.5° cell, so the plan decides between N-source/N-layer duplication and merge-at-download with that evidence, documenting the choice in the plan's SUMMARY rather than defaulting silently.)*
   - What we know: native pmtiles source = one file; current provider fans across N.
   - What's unclear: does the download service already/could it produce one merged archive per trail? Or must the style carry N sources + N×14 layers?
   - Recommendation: prefer **merge cells into a single `.pmtiles` at download time** — keeps the style to one protomaps source and 14 layers. Investigate the existing download service's cell-generation during planning; if merge is infeasible, fall back to programmatic N-source layer duplication.

3. **Interim location display in `WandererMap` (A5, CORE-07 is Phase 17).** *(RESOLVED: confirmed `trail_detail_map_screen.dart:88` passes `showLocation: true` — `15-04-PLAN.md` Task 1 adds an interim `WidgetLayer`/`Marker` puck driven by `foregroundPositionStreamProvider`, explicitly excluding Phase-17 native follow/heading behavior.)*
   - What we know: `enableLocation`/`trackLocation` exist on `MapController` but are Phase-17 scope; today `WandererMap` shows `CurrentLocationLayer` when `showLocation`.
   - Recommendation: confirm whether `trail_detail_map_screen`/`trail_panel` pass `showLocation: true`; if not, defer cleanly; if yes, a minimal `WidgetLayer` puck or an early `enableLocation` call may be needed.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `maplibre` package | all rendering | ✓ | 0.3.5 (lock) | — |
| MapLibre Native (Android) | native render + pmtiles | ✓ (bundled) | 13.0 | — |
| MapLibre Native (iOS) | native render + pmtiles | ✓ (bundled) | 6.25 | — |
| `path_provider` | cache dir | ✓ | 2.1.5 | — |
| `/map/style-sources` endpoint | glyph/sprite/tile URLs | ✓ | Phase 13 shipped | — |
| Physical iOS/Android device (airplane mode) | spike (OFFL-04) + final offline gate | ✗ to Claude | — | **User (Christian) runs it — D-02.** No Claude-side fallback; this is by design. |
| Flutter theme extraction toolchain (`dart run tool/...`) | STYLE-01 dump | ✓ | Flutter SDK 3.11.5 | — |

**Missing dependencies with no fallback:** physical-device execution of the spike and the final offline gate — delegated to the user per D-01/D-02. Everything Claude can build/verify (code, style JSON, providers, layers) is available.

## Spike Design (Plan 1 — throwaway, per D-01/D-02/D-03)

**Goal:** Prove (or disprove) that MapLibre Native renders place-name labels from `file://`-scheme glyphs (and ideally sprite) fetched to the app documents directory, on a physical device in airplane mode. Nothing else.

**Minimal build:**
1. A throwaway route/screen with a single `MapLibreMap`.
2. A hand-built **minimal** style JSON (not the 7,677-line one) with:
   - one simple source (a bundled `pmtiles://asset://` or even a tiny GeoJSON point source with a label),
   - `"glyphs": "file:///<app-docs>/glyphs/{fontstack}/{range}.pbf"`,
   - optionally `"sprite": "file:///<app-docs>/sprite"`,
   - one `symbol` layer with a `text-field` (e.g. a place label or a static GeoJSON point label) using `text-font: ["Noto Sans Regular"]`.
3. Pre-seed the cache: at spike start (with network, before airplane mode) fetch the glyph ranges covering the label's codepoints (e.g. `Noto Sans Regular/0-255.pbf`) from the `/map/style-sources` glyph URL into `<app-docs>/glyphs/…`; likewise sprite files. Then instruct the user to enable airplane mode and reopen.
4. Build to the physical device (D-01), hand to Christian (D-02).

**Pass signal:** the text label renders correctly with no network.
**Fail signals to capture (D-03):** (a) label blank / tofu boxes, (b) style-load throws, (c) native log shows glyph resource 404 / "unsupported scheme" / falls back to online host. Capture Android logcat or iOS console output and return the exact error — do NOT pick a fallback direction.

**Do NOT include in the spike:** trail rendering, real style extraction, the full glyph cache warm, download flow. Those are later plans, gated on PASS.

## Security Domain

> `security_enforcement: true`, ASVS level 1. This phase is client-side map rendering, but it fetches remote resources, writes files to disk, and injects operator-controlled URLs into a style — so input-validation and file-path controls apply.

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | No auth logic in this phase (uses existing session for `/map/style-sources`) |
| V3 Session Management | no | — |
| V4 Access Control | no | Rendering only |
| V5 Input Validation | **yes** | Glyph/sprite/tile URLs come from `/map/style-sources` (operator-controlled). Validate scheme (`https://` online, `file://`/`pmtiles://` offline) before injecting into style JSON; reject unexpected schemes. Validate that downloaded glyph/pmtiles paths stay within the app documents dir (no `..` traversal in constructed filenames). |
| V6 Cryptography | no | No crypto introduced; TLS handled by platform HTTP for glyph fetch |
| V12/V13 Files & API | **yes** | Cache file writes: sanitize `{fontstack}`/`{range}` before using them as path segments to prevent path traversal when writing to `<app-docs>/map_cache`. Confirm downloaded content-type/size limits on glyph/sprite fetch. |

### Known Threat Patterns for Flutter native-map + on-device cache
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Path traversal via crafted fontstack/range in cache filename | Tampering | Whitelist fontstack names (the 4 known) + numeric range only; build paths with `path.join`, reject `..` |
| SSRF-ish: operator/endpoint returns a malicious glyph/tile host | Info disclosure / Spoofing | Endpoint is Wanderer-controlled (Phase 13) + defaults to Protomaps; validate scheme+host allowlist is out of scope but note the trust boundary is the operator (documented, acceptable) |
| Rendering untrusted style JSON causing native crash | DoS | Style comes from bundled asset + trusted endpoint values, not user input — low risk; keep the asset checked-in and generated, not runtime-fetched |
| Offline `file://` reads outside sandbox | Info disclosure | Only ever construct `file://` paths rooted at `getApplicationDocumentsDirectory()`; never accept an external absolute path |

**Net:** no high-severity findings block this phase; the load-bearing control is **path sanitization on the glyph/sprite cache writes** (V5/V12). Add a verification step asserting cache filenames derive only from the 4 known fontstacks + numeric ranges.

## Sources

### Primary (HIGH confidence)
- Installed package source `~/.pub-cache/hosted/pub.dev/maplibre-0.3.5/` (exports, docs, CHANGELOG), `maplibre_platform_interface-0.3.5` (`MapOptions`, `MapController`, `StyleController`, `VectorSource`, `GeoJsonSource`, `Line/Symbol/CircleStyleLayer`, `Marker`/`WidgetLayer`, `SourceAttribution`, `MapScalebar`), `maplibre_android-0.3.5`/`maplibre_ios-0.3.5` (`setStyle` scheme branching, file:// wrapping, pmtiles FFI) — API surface, style-loading contract, control class names.
- Codebase: `app/lib/components/base/wanderer_map.dart`, `app/lib/components/map/trail_layer.dart`, `app/lib/provider/map_style_provider.dart`, `map_style_sources_provider.dart`, `app/lib/vendor/vector_map_tiles/pm_tile_provider.dart`, theme source at flomp fork resolved-ref `d52dd7d…`, `pubspec.lock`.
- https://maplibre.org/maplibre-native/android/examples/data/PMTiles/ and iOS PMTiles docs — `pmtiles://file://` local scheme (WebFetch).

### Secondary (MEDIUM confidence)
- MapLibre Style Spec (glyphs, sprite, layers/#line, layers/#symbol) — layout/paint field semantics.
- oliverwipfli.ch "About Text Rendering in MapLibre", MapLibre GL JS PR #4564 (local TinySDF fallback — JS only, not native).

### Tertiary (LOW confidence — flagged for the spike)
- WebSearch consensus that mobile local resources use `asset://` and that "glyph URLs must be absolute" — suggests `file://` glyph templates may or may not resolve; the exact runtime-`file://`-glyph behavior is UNVERIFIED and is what the spike proves.
- GitHub maplibre/flutter-maplibre-gl#338 (old package, `asset://` glyphs) — adjacent prior art, not the same package.

## Metadata

**Confidence breakdown:**
- Standard stack / package API: HIGH — read directly from installed 0.3.5 source, not training data.
- Architecture / rendering patterns: HIGH — layout/paint passthrough + control classes verified in source.
- STYLE-01 extraction: HIGH — theme is a plain `Map` literal, signature verified.
- Offline pmtiles (OFFL-03/05): MEDIUM — `pmtiles://file://` verified; multi-cell strategy is an open design question.
- `file://` glyphs (OFFL-04): LOW by design — the risk gate; intentionally deferred to the physical-device spike.

**Research date:** 2026-07-08
**Valid until:** ~2026-08-07 for the package API (stable, pinned). The spike outcome supersedes the OFFL-04 assumption the moment it runs — treat A1/A2 as provisional until then.
