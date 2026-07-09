# Phase 15: MapLibre Core, Trail Rendering & Offline Parity - Pattern Map

**Mapped:** 2026-07-08
**Files analyzed:** 10 (2 rewrite consumers untouched-signature, 3 core rewrites, 3 new, 2 generated assets, 1 one-off tool, 1 deletion)
**Analogs found:** 9 / 10 (spike screen intentionally low-pattern-value)

> **Scope note:** RESEARCH.md already carries the *verified maplibre 0.3.5 package API* (Patterns 1-7, control class names, `setStyle`/`fitBounds`/`WidgetLayer` signatures). This document is the **codebase-analog side** — what exists in `app/lib` today that the new code replaces or must stay compatible with. Cross-reference RESEARCH.md Patterns 1-7 for the *target* API; use the excerpts here for the *source* values, signatures, and provider shapes to preserve.

---

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `components/base/wanderer_map.dart` (rewrite, CORE-01) | component (map host) | request-response (style→render) | itself (current `flutter_map` impl) | exact (self-port) |
| `components/map/trail_layer.dart` (rewrite, TRAIL-01..05) | component (overlay layers) | transform (geometry→layers) | itself (current `flutter_map` impl) | exact (self-port) |
| `provider/map_style_provider.dart` (NOT MODIFIED — kept as-is for the 4 not-yet-migrated `flutter_map` screens; see `provider/map_style_json_provider.dart` below for the new sibling) | provider | request-response (async config→style) | itself | n/a — superseded by 15-02-PLAN.md's actual approach |
| `provider/map_style_json_provider.dart` (NEW, STYLE/CORE-02, per 15-02-PLAN.md) | provider | request-response (async config→style JSON String) | `map_style_provider.dart` (shape) + `map_style_sources_provider.dart` | role-match |
| `provider/glyph_sprite_cache_provider.dart` (NEW, GLYPH-04) | provider | file-I/O + batch fetch | `map_style_sources_provider.dart` (shape) + `trail_download_service.dart` (fetch/write) | role-match |
| `util/offline_style_rewriter.dart` (NEW, OFFL-02) | utility | transform (style map→file:// URLs) | `util/map_coordinate_adapter.dart` (stateless converter shape) | partial (no existing style rewriter) |
| `assets/map/wanderer_light.json` / `wanderer_dark.json` (NEW, STYLE-01) | config (asset) | n/a (generated) | current `vtr.wandererLightTheme/DarkTheme` output | exact (dump of source) |
| `tool/extract_map_styles.dart` (NEW, STYLE-01) | utility (one-off CLI) | file-I/O (Map→jsonEncode→file) | none in `app/lib` (first `tool/` script) | no analog |
| `routes/*` spike screen (Plan 1, throwaway) | component (screen) | request-response | `routes/trail_detail_map_screen.dart` (screen shell) | role-match, low value |
| `routes/trail_detail_map_screen.dart` (modified consumer) | component (screen) | request-response | itself | exact (signature-break update) |
| `components/trail/trail_panel.dart` (modified consumer) | component | request-response | itself | exact (signature-break update) |
| `vendor/vector_map_tiles/pm_tile_provider.dart` (DELETE, OFFL-06) | — | — | — | deletion |

---

## Pattern Assignments

### `provider/map_style_json_provider.dart` (NEW sibling provider — NOT a rewrite of `map_style_provider.dart`)

**Correction (post plan-check):** 15-02-PLAN.md does NOT rewrite `map_style_provider.dart` in place. It stays untouched, returning `vtr.Style`, because the 4 not-yet-migrated `flutter_map` screens (`list_detail_map_screen`, `list_detail_screen`, `map_screen`, `navigation_screen`) still consume it — mutating its return type would break phase success criterion #5 ("the app still builds and runs with `flutter_map` serving" those screens). Instead, a NEW sibling provider (`map_style_json_provider.dart`) is added, returning a raw style-JSON `String` for the `MapLibreMap`-based `WandererMap`.

**Analog:** `map_style_provider.dart` (below, for the wiring shape to copy) + `map_style_sources_provider.dart` (for the `@Riverpod(keepAlive: true)` + async-config chaining pattern).

**Current implementation (full — the shape to preserve, type to change)** `map_style_provider.dart:10-25`:
```dart
@Riverpod(keepAlive: true)
Future<Style> mapStyle(Ref ref) async {
  final mode = ref.watch(themeModeProvider);
  final sources = await ref.watch(mapStyleSourcesProvider.future);
  final brightness = effectiveBrightness(mode);
  final asset = brightness == Brightness.dark
      ? vtr.wandererDarkTheme(sources.tileUrl)
      : vtr.wandererLightTheme(sources.tileUrl);
  return StyleReader.map(asset).read();       // <-- becomes: load asset json, inject sources, jsonEncode → String
}

Brightness effectiveBrightness(ThemeMode mode) { ... }  // KEEP verbatim — theme resolution unchanged
```
Port notes:
- Keep `ref.watch(themeModeProvider)` + `await ref.watch(mapStyleSourcesProvider.future)` — this is exactly what makes CORE-02 live-swap work (provider re-runs on theme change; new output feeds `setStyle`).
- Replace `vtr.wandererXTheme(...)` + `StyleReader.map(...).read()` with: load `assets/map/wanderer_{light|dark}.json`, string-replace `__TILE_URL__`/`__GLYPH_URL__`/`__SPRITE_URL__` from `sources.{tileUrl,glyphUrl,spriteUrl}`, and (if `offline`) delegate to `offline_style_rewriter.dart`.
- `effectiveBrightness()` helper ports verbatim.

---

### `provider/map_style_sources_provider.dart` — DO NOT MODIFY (reference analog)

This Phase-13 provider is the canonical `@Riverpod(keepAlive: true)` async-config pattern the two new/rewritten providers copy. Full source `map_style_sources_provider.dart:7-15`:
```dart
@Riverpod(keepAlive: true)
class MapStyleSourcesNotifier extends _$MapStyleSourcesNotifier {
  @override
  Future<MapStyleSources> build() async {
    final api = ref.watch(apiProvider);
    final response = await api.get('/map/style-sources');
    return MapStyleSources.fromJson(response.data);
  }
}
```
Model it consumes (`models/map_style_sources.dart`, freezed): `{ tileUrl, glyphUrl, spriteUrl }` — these three fields feed STYLE-02/03/04 injection and GLYPH-04 fetch URLs.

---

### `provider/glyph_sprite_cache_provider.dart` (NEW, GLYPH-04)

**Analog (provider shape):** `map_style_sources_provider.dart` (above) — same `@Riverpod(keepAlive: true)` class + async `build()`, chaining off `mapStyleSourcesProvider.future`.
**Analog (fetch + on-disk write + skip-if-exists):** `services/trail_download_service.dart` — the established app-documents-dir + `path_provider` + idempotent-download convention D-08/D-10 must follow.

**Storage/fetch convention to copy** `trail_download_service.dart:32-33, 144-186`:
```dart
final appDir = await getApplicationDocumentsDirectory();       // path_provider — SAME dir root
final trailDir = Directory('${appDir.path}/library/$trailId'); // glyph cache → e.g. ${appDir.path}/map_cache/...
...
final tilesDir = Directory('${trailDir.path}/tiles');
if (!await tilesDir.exists()) { await tilesDir.create(recursive: true); }
...
final localPath = '${tilesDir.path}/$key.pmtiles';
if (await File(localPath).exists()) {                          // idempotent skip → OFFL-01 second-download reuse
  return localPath;
}
...
await _api.download(readyCell.downloadUrl!, localPath, cancelToken: cancelToken);  // dio download to disk
```
Key carry-overs: same `getApplicationDocumentsDirectory()` root, `Directory(...).create(recursive: true)`, `File(localPath).exists()` idempotency check (this IS the OFFL-01/D-10 "no-op against warm cache"), `_api.download(...)` (dio) for byte fetch. **Path-safety (RESEARCH §Security V5/V12):** whitelist the 4 fontstack names + numeric range before using them as path segments; never accept an external absolute path.

---

### `components/base/wanderer_map.dart` (rewrite, CORE-01..04)

**Analog:** itself. Must preserve the public widget contract (RESEARCH.md "Component Responsibilities" table). The constructor is the compatibility surface — keep param names/defaults; `mapController` type is the one breaking change.

**Widget contract to preserve** `wanderer_map.dart:16-48`:
```dart
class WandererMap extends ConsumerStatefulWidget {
  final Trail trail;
  final MapController? mapController;              // ml.MapController now (SIGNATURE CHANGE — 2 consumers)
  final bool disabled;                            // → MapOptions.gestures = MapGestures.none()
  final bool offline;                             // → style-rewrite branch (pmtiles://file:// + file:// glyphs)
  final List<Widget>? controls;                   // → children: Column topRight (ports verbatim)
  final ml.Geographic? elevationMarkerPosition;   // → single WidgetLayer Marker (TRAIL-05)
  final EdgeInsets initialCameraFitPadding;       // → fitBounds(padding:), default EdgeInsets.all(40)
  final bool showTrail;                           // → conditionally add track layers
  final bool showLocation;                        // → DEFER (CORE-07/Phase 17); confirm interim need
  final Waypoint? selectedWaypoint;               // → drives AnimatedScale in marker
  final TapCallback? onTap;                        // → onEvent MapEventClick
  final Function(MapEvent)? onMapEvent;            // → onEvent (NOTE bug below)
  final Function(Waypoint wp)? onWaypointTap;      // → GestureDetector.onTap in marker
  const WandererMap({ ... this.initialCameraFitPadding = const EdgeInsets.all(40), });
}
```

**Loading / error passthrough to port as-is** `wanderer_map.dart:101-112` (UI-SPEC Copywriting Contract requires verbatim port):
```dart
if (_error != null) { return Center(child: Text(_error.toString())); }
return styleAsync.when(
  skipLoadingOnRefresh: false,
  loading: () => ColoredBox(color: Theme.of(context).colorScheme.surface),
  error: (e, _) => Center(child: Text(e.toString())),
  data: (style) {
    if (widget.offline && _offlineTileProvider == null) {
      return const Center(child: CircularProgressIndicator());   // offline init spinner
    }
```

**Initial camera fit — value source for CORE-03** `wanderer_map.dart:123-135` (maps 1:1 to `controller.fitBounds(bounds, padding, nativeDuration: Duration.zero)` in `onStyleLoaded`, RESEARCH Pattern 3):
```dart
initialCameraFit: _bounds != null
    ? CameraFit.bounds(bounds: toLatLngBounds(_bounds!), padding: widget.initialCameraFitPadding)
    : null,
initialCenter: toLatLng(ml.Geographic(lat: widget.trail.lat ?? 0, lon: widget.trail.lon ?? 0)),
initialZoom: 18,   // fallback when no bounds
```

**Elevation marker — exact visual to reproduce as a `WidgetLayer` Marker (TRAIL-05)** `wanderer_map.dart:152-175`:
```dart
Marker(width: 12, height: 12, point: toLatLng(widget.elevationMarkerPosition!),
  child: Container(decoration: BoxDecoration(
    color: Colors.white, shape: BoxShape.circle,
    boxShadow: [BoxShadow(color: Colors.black.withValues(alpha: .2), blurRadius: 4, offset: const Offset(0, 2))],
    border: Border.all(color: Colors.black, width: 2))));
```

**Controls stack (ports verbatim)** `wanderer_map.dart:177-183`:
```dart
Align(alignment: Alignment.topRight,
  child: Column(crossAxisAlignment: CrossAxisAlignment.center, children: widget.controls!));
```

**Deletions this rewrite performs:** `MultiPmTilesVectorTileProvider _offlineTileProvider` + `_initOffline()` + `_buildTileLayer()` (`wanderer_map.dart:55-95`) → replaced by native `pmtiles://file://` source in style JSON (OFFL-03). `map_coordinate_adapter` imports drop here (native uses `ml.Geographic`/`ml.LngLatBounds` directly). `CurrentLocationLayer` + `foregroundPositionStreamProvider` (`wanderer_map.dart:147-150`) — defer per A5, confirm `showLocation` usage.

**Bug to fix on port** `wanderer_map.dart:119`: `onMapEvent: (e) => widget.onMapEvent` returns the field, never invokes it. Wire the new `onEvent` to actually call `widget.onMapEvent`.

---

### `components/map/trail_layer.dart` (rewrite, TRAIL-01..05)

**Analog:** itself — the single source of the pixel-exact port values. This rewrite is largely a **deletion**: the entire `_addArrowsAlongPath` bearing loop and `AnimationController` (lines 39-107, 218-232) collapse to one `SymbolStyleLayer` (RESEARCH Pattern 6), and the `PolylineLayer` becomes two `LineStyleLayer`s (Pattern 1). Only the interactive widget markers stay as Flutter (`WidgetLayer`, Pattern 7).

**Track line values → two `LineStyleLayer`s (TRAIL-01)** `trail_layer.dart:23,204-215`:
```dart
this.routeColor = const Color(0xff3549bb),   // route line-color #3549bb, overridable
this.strokeWidth = 5.0,                       // route line-width 5
...
Polyline(points: toLatLngList(pathPoints), color: widget.routeColor,
  strokeWidth: widget.strokeWidth, borderColor: Colors.white, borderStrokeWidth: 2);
// → casing LineStyleLayer #ffffff width 9 (=5+2×2) UNDER route LineStyleLayer #3549bb width 5
```

**Waypoint marker — preserve verbatim in `WidgetLayer`/`Marker` (TRAIL-03)** `trail_layer.dart:120-142` + `_buildCircularMarker` `241-267`:
```dart
Marker(point: ..., width: 32, height: 32,
  child: GestureDetector(onTap: () => widget.onWaypointTap?.call(wp),
    child: AnimatedScale(scale: isSelected ? 1.0 : 0.875,
      duration: const Duration(milliseconds: 200), curve: Curves.easeOutBack,
      child: _buildCircularMarker(wp.icon, color: Theme.of(context).primaryColor, selected: isSelected))));
// _buildCircularMarker: bg selected?white:color, border 2px selected?color:white,
//   boxShadow black@.2 blur4 offset(0,2), FaIcon selected?color:white size selected?16:14
```

**Start/finish pins + 36px proximity nudge (TRAIL-04)** `trail_layer.dart:145-184`:
```dart
final startPx = camera.latLngToScreenOffset(toLatLng(pathPoints.first));   // → ml MapController.toScreenLocations
final endPx = camera.latLngToScreenOffset(toLatLng(pathPoints.last));
if (math.sqrt(dx*dx + dy*dy) < 36) { startAlignment = Alignment(1,0); endAlignment = Alignment(-1,0); }
// start: width/height 28, FontAwesomeIcons.bullseye, Colors.greenAccent
// finish: width/height 28, FontAwesomeIcons.flagCheckered, Colors.redAccent
```

**Directional arrows — spacing-by-zoom intent to replicate, motion to DELETE (TRAIL-02, D-04/D-05)** `trail_layer.dart:187-201`:
```dart
double arrowSpacingMeters = 100.0;
if (currentZoom >= 16) arrowSpacingMeters = 240.0;
else if (currentZoom >= 14) arrowSpacingMeters = 600.0;
else if (currentZoom >= 12) arrowSpacingMeters = 2000.0;
else if (currentZoom >= 10) arrowSpacingMeters = 5000.0;
else arrowSpacingMeters = 10000.0;
bool showArrows = currentZoom > 8.0;
showArrows = false;   // <-- DEAD CODE today (D-04: re-enable for real on symbol layer)
```
Port: symbol layer `minZoom: 8`, `symbol-placement: line`, `icon-image: 'arrow'` (from sprite STYLE-04), `icon-rotation-alignment: map`. Meter values do NOT map 1:1 to `symbol-spacing` (pixels) — replicate "denser at higher zoom" via zoom-interpolated pixel spacing, tune on device (RESEARCH Pitfall 5, D-05 accepts approximation). Delete `AnimationController` (`44-47`), `_addArrowsAlongPath` (`56-107`), `AnimatedBuilder` block (`218-232`).

---

### `util/offline_style_rewriter.dart` (NEW, OFFL-02)

**Analog:** `util/map_coordinate_adapter.dart` (Phase 14) — the app's convention for a stateless, named-export conversion utility. No existing style-rewriter exists; follow that shape (pure functions, no state). **Depends on the Plan-1 spike passing.**

**What it does (RESEARCH Pattern, no existing analog for the logic):** given the parsed style `Map<String,dynamic>` + cache root path, rewrite:
- `sources.protomaps.tiles[0]` → `pmtiles://file:///<docs>/library/<trail>/tiles/<cell>.pmtiles` (OFFL-03/05)
- `glyphs` → `file:///<docs>/map_cache/glyphs/{fontstack}/{range}.pbf` (OFFL-02, gated on spike)
- `sprite` → `file:///<docs>/map_cache/sprite` (OFFL-02)

Security: validate scheme allowlist and that all constructed `file://` paths root at `getApplicationDocumentsDirectory()` (RESEARCH §Security V5/V12).

---

### `tool/extract_map_styles.dart` (NEW, one-off, STYLE-01)

**Analog:** none in `app/lib` (first `tool/` script). Follows RESEARCH Pattern 5 directly — imports `vector_tile_renderer` flomp fork, calls `wandererLightTheme('__TILE_URL__')` / `wandererDarkTheme(...)`, **adds the `glyphs`/`sprite` keys the themes lack today** (RESEARCH Pitfall 4), `JsonEncoder.withIndent('  ')` → `assets/map/wanderer_{light,dark}.json`. Register the two assets in `pubspec.yaml` `flutter/assets`.

---

### Consumer updates (signature-break, CORE-01)

Both construct `WandererMap` with a `flutter_map` `MapController` typed field that must become `ml.MapController`, and both currently import `map_coordinate_adapter` for the `fitCamera`/`CameraFit.bounds` call.

**`routes/trail_detail_map_screen.dart`** — the primary consumer:
- `trail_detail_map_screen.dart:39` `final MapController _mapController = MapController();` → `ml.MapController`.
- `trail_detail_map_screen.dart:82-102` constructs `WandererMap(... mapController: _mapController, showLocation: true, offline: trail.isOffline, ...)` — `showLocation: true` here is the one call site that forces the A5 interim-location decision.
- `trail_detail_map_screen.dart:261-266` `_mapController.fitCamera(CameraFit.bounds(bounds: toLatLngBounds(bounds), padding: padding))` → `ml.MapController.fitBounds(bounds: bounds, padding: padding)` (bounds already `ml.LngLatBounds`, drop `toLatLngBounds`).

**`components/trail/trail_panel.dart`** — the secondary consumer (`trail_panel.dart:249-255`): constructs `WandererMap(trail:, disabled: true, offline: trail.isOffline, onTap: ...)` with no `mapController` — only affected if `WandererMap`'s param defaults change. `disabled: true` → `MapGestures.none()`. Minimal change; verify `onTap` still maps to the new `onEvent`.

---

## Shared Patterns

### `@Riverpod(keepAlive: true)` async-config provider
**Source:** `provider/map_style_sources_provider.dart:7-15`
**Apply to:** `map_style_provider.dart` (rewrite), `glyph_sprite_cache_provider.dart` (new). Both chain off `ref.watch(mapStyleSourcesProvider.future)`; keepAlive gives the app-wide single-fetch semantics D-08/D-09 require.

### App-documents-dir file cache with idempotent skip
**Source:** `services/trail_download_service.dart:32-33, 144-186`
**Apply to:** `glyph_sprite_cache_provider.dart`. `getApplicationDocumentsDirectory()` root, `Directory(...).create(recursive: true)`, `File(path).exists()` skip (= OFFL-01 warm-cache no-op / D-10 dual-trigger), `dio _api.download(...)` fetch. **Trigger wiring (D-09/D-10):** call `ref.read(glyphSpriteCacheProvider.future)` from first map open (in `WandererMap`) AND from `provider/trail/trail_download_provider.dart` / `services/trail_download_service.dart` at download start.

### Coordinate types — native, drop the adapter
**Source:** `util/map_coordinate_adapter.dart` (Phase 14 bridge)
**Apply to:** every file this phase migrates fully to `MapLibreMap` (`wanderer_map.dart`, `trail_layer.dart`, the `fitCamera` call in `trail_detail_map_screen.dart`) — delete `toLatLng`/`toLatLngBounds`/`toLatLngList` usages; native API takes `ml.Geographic`/`ml.LngLatBounds` directly. The adapter STAYS for the 4 not-yet-migrated `flutter_map` screens (Phase 16/17).

### Loading / error / offline-spinner UI (port verbatim, no new copy)
**Source:** `wanderer_map.dart:101-112`
**Apply to:** rewritten `wanderer_map.dart`. `ColoredBox(surface)` while style loads, `Center(Text(error))` passthrough, `CircularProgressIndicator` while offline provider initializes. UI-SPEC Copywriting Contract forbids new error/loading copy.

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `tool/extract_map_styles.dart` | utility (CLI) | file-I/O | First `tool/` script in `app/`; no in-repo Dart CLI precedent. Follow RESEARCH Pattern 5 verbatim. |
| `util/offline_style_rewriter.dart` (logic) | utility | transform | No existing style-JSON rewriter. Shape from `map_coordinate_adapter.dart`; logic is net-new (RESEARCH Pattern, gated on spike). |
| Plan-1 spike screen | component (screen) | request-response | Throwaway; deliberately minimal (not the real style). Reuse `trail_detail_map_screen.dart`'s `Scaffold`+`ConsumerStatefulWidget` shell only. Low pattern value by design. |

## Metadata

**Analog search scope:** `app/lib/components/{base,map,trail}`, `app/lib/provider`, `app/lib/provider/trail`, `app/lib/services`, `app/lib/models`, `app/lib/util`, `app/lib/routes`
**Files scanned:** ~12 read in full/part; providers + download service + 2 consumers + both rewrite targets
**Pattern extraction date:** 2026-07-08
</content>
</invoke>
