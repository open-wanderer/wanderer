# Phase 17: Navigation on MapLibre - Pattern Map

**Mapped:** 2026-07-10
**Files analyzed:** 6 (1 modified in-place, 3 read-only analogs, 2 deleted, 1 partially-deleted)
**Analogs found:** 6 / 6

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `app/lib/routes/navigation_screen.dart` (rewritten map section, `ml.MapLibreMap` inlined) | route/component (map host + turn-by-turn UI) | event-driven (GPS stream + native camera/location events) | `app/lib/routes/map_screen.dart` (`SearchMap` host usage) + `app/lib/components/base/wanderer_map.dart` (`onMapCreated`/`onStyleLoaded`/style-swap lifecycle) | role-match (no exact prior "nav screen on maplibre" analog exists — this phase is the first) |
| `app/lib/components/map/trail_layer.dart` (remove legacy `TrailLayer` widget + its `flutter_map` imports; navigation switches to `addTrailTrackLayers`/`TrailMarkerLayer`) | utility/component (native style-layer + marker layer) | transform (GeoJSON in, native GL layers out) | itself — `addTrailTrackLayers`/`TrailMarkerLayer` already in this file, used by `WandererMap` | exact (already the intended target API, just needs a new caller) |
| Breadcrumb source/layer (new code inside `navigation_screen.dart`, no new file) | utility (imperative GeoJSON source update) | streaming (grows on every GPS fix) | `app/lib/components/map/cluster_layer.dart` (`addClusterLayers`/`updateClusterSource` — add-once + in-place update pattern) | role-match |
| Compass control (new code inside `navigation_screen.dart`, replaces `map_compass.dart` import) | component (native GL widget wrapper) | request-response (tap → camera command) | `app/lib/routes/map_screen.dart` line 398 (`ml.MapCompass(hideIfRotatedNorth: true)`) — needs `onPressed`/`rotateNorthOnPressed` override added, no exact override precedent in-repo | role-match (partial — repo has no `onPressed`-override precedent; RESEARCH.md Pattern 3 supplies the verified API shape) |
| Location puck + follow (new code inside `navigation_screen.dart`, replaces `CurrentLocationLayer`) | service/event-driven (native `enableLocation`/`trackLocation`) | event-driven | none in-repo — first caller of these APIs; RESEARCH.md Pattern 1/Code Examples is the authoritative source | no analog (new ground, see "No Analog Found") |
| `app/lib/components/map/map_compass.dart` | component | request-response | DELETED (CORE-05) — no replacement file, folded into `navigation_screen.dart`'s inline `ml.MapCompass` usage | n/a (deletion) |
| `app/lib/vendor/vector_map_tiles/pm_tile_provider.dart` | utility (offline tile provider) | file-I/O | DELETED (OFFL-06) — replaced by `WandererMap`'s `rewriteStyleForOffline` + `pmtiles://file://` pattern, see `app/lib/util/offline_style_rewriter.dart` | n/a (deletion, superseded by existing pattern) |
| `app/lib/util/map_coordinate_adapter.dart` | utility | transform | usages removed from `navigation_screen.dart`; file itself left in place (Pitfall 6 — grep before deleting; out of scope this phase per CONTEXT.md) | n/a (no modification to file itself) |

## Pattern Assignments

### `app/lib/routes/navigation_screen.dart` (route/component, event-driven map host)

**Primary analog:** `app/lib/components/base/wanderer_map.dart` (lifecycle/style-swap) + `app/lib/routes/map_screen.dart` (screen-level `MapLibreMap`/`SearchMap` wiring, drag-vs-gesture handling, compass placement)

**Imports pattern** — drop all `flutter_map`/plugin imports, keep app-tier imports (from current `navigation_screen.dart` lines 1–31); target shape (by analogy with `map_screen.dart` lines 1–29):
```dart
import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:geolocator/geolocator.dart' as geo;
import 'package:go_router/go_router.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/components/map/trail_layer.dart'; // addTrailTrackLayers, TrailMarkerLayer, kTrailRouteColor
import 'package:wanderer/components/trail/elevation_profile.dart';
import 'package:wanderer/components/trail/waypoint_sheet.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/navigate_response.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/waypoint.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/provider/map_style_json_provider.dart'; // replaces mapStyleProvider (Style/theme) with the JSON provider WandererMap uses
import 'package:wanderer/provider/navigation_provider.dart';
import 'package:wanderer/provider/navigation_stats_provider.dart';
import 'package:wanderer/provider/trail/trail_provider.dart';
import 'package:wanderer/util/format_util.dart';
import 'package:wanderer/util/offline_style_rewriter.dart'; // replaces pm_tile_provider.dart
import 'package:wanderer/util/tracelet_position_source.dart';
// DROP: flutter_map, flutter_map_animations, flutter_map_location_marker,
// vector_map_tiles, map_compass.dart, map_coordinate_adapter.dart, pm_tile_provider.dart
```

**Style/offline composition pattern** (source: `wanderer_map.dart` lines 85–172, `_composeStyle`/`_swapStyle`/`_lastStyleJson`):
```dart
// Source: app/lib/components/base/wanderer_map.dart lines 140-172
String? _composeStyle(String? baseJson, GlyphSpriteCachePaths? cache) {
  if (baseJson == null) return null;
  if (!widget.isOffline) return baseJson;
  if (cache == null) return null;
  try {
    final decoded = jsonDecode(baseJson) as Map<String, dynamic>;
    final offlineStyle = rewriteStyleForOffline(
      decoded,
      cacheRoot: cache.root,
      cellPaths: trail.pmTiles,
      dark: effectiveBrightness(ref.read(themeModeProvider)) == Brightness.dark,
    );
    return jsonEncode(offlineStyle);
  } catch (e) {
    debugPrint('NavigationScreen: offline style rewrite failed — $e');
    return null;
  }
}
```
Apply this instead of `navigation_screen.dart`'s current `_initOffline`/`MultiPmTilesVectorTileProvider.fromSources` (lines 107–118) and `_buildTileLayer` (lines 155–171) — those two methods are fully replaced, not adapted.

**onMapCreated/onStyleLoaded hand-off pattern** (source: `wanderer_map.dart` lines 190–201, extended per RESEARCH.md Pattern 1):
```dart
onMapCreated: (controller) => _controller = controller,
onStyleLoaded: (style) async {
  _fitInitialCamera().ignore(); // or fitBounds to widget.response.shapeAsGeographic bounds
  if (trail?.expand?.gpx != null) {
    await addTrailTrackLayers(style, trail!); // re-added every style load (CORE-02 survives)
  }
  await style.addSource(ml.GeoJsonSource(id: 'breadcrumb', data: _breadcrumbGeoJson(navState.breadcrumb)));
  await style.addLayer(const ml.LineStyleLayer(
    id: 'breadcrumb-route',
    sourceId: 'breadcrumb',
    paint: {'line-color': '#DC2626', 'line-width': 3.5},
  ));
  final controller = _controller;
  if (controller != null) {
    await controller.enableLocation(bearingRenderMode: ml.BearingRenderMode.gps); // D-04
    await controller.trackLocation(
      trackLocation: _followEnabled,
      trackBearing: _headingUp ? ml.BearingTrackMode.gps : ml.BearingTrackMode.none,
    );
  }
},
```

**Drag-only follow-break pattern** (source: current `navigation_screen.dart` lines 233–240 for the *behavior contract*; RESEARCH.md Pattern 2 for the *native implementation*, since `map_screen.dart`'s `apiGesture` check at lines 308–313 does NOT distinguish drag from pinch and is insufficient alone):
```dart
// Behavior contract (KEEP — from current navigation_screen.dart, lines 233-240):
// "Only drag events disable follow — pinch-zoom events must NOT pause follow"
// Native implementation (NEW — RESEARCH.md Pattern 2, no in-repo precedent):
int _activePointers = 0;
// wrap MapLibreMap in a Listener; onPointerDown/Up/Cancel track _activePointers;
// onEvent: if (event is ml.MapEventStartMoveCamera &&
//     event.reason == ml.CameraChangeReason.apiGesture &&
//     _activePointers <= 1 && _followEnabled) { _onPanStart(); }
```

**Compass toggle pattern** (source: current `navigation_screen.dart` lines 293–301 for the *toggle contract* D-01/D-02; `map_screen.dart` line 398 for in-repo `ml.MapCompass` usage shape; RESEARCH.md Pattern 3 for the verified `onPressed`/`rotateNorthOnPressed` override API):
```dart
ml.MapCompass(
  hideIfRotatedNorth: false, // D-02 — always visible during nav (contrast map_screen.dart's `true`)
  rotateNorthOnPressed: false, // prevent double-handling
  onPressed: () {
    setState(() => _headingUp = !_headingUp);
    final controller = _controller;
    if (controller == null) return;
    controller.trackLocation(
      trackLocation: _followEnabled,
      trackBearing: _headingUp ? ml.BearingTrackMode.gps : ml.BearingTrackMode.none,
    );
    if (!_headingUp) {
      controller.animateCamera(bearing: 0, nativeDuration: const Duration(milliseconds: 400));
    }
  },
)
```

**Recenter pattern** (source: current `navigation_screen.dart` lines 150–153 for the *contract*; RESEARCH.md D-03 code example for the *native call*):
```dart
void _onRecenter() {
  setState(() => _followEnabled = true);
  _controller?.trackLocation(
    trackLocation: true,
    trackBearing: _headingUp ? ml.BearingTrackMode.gps : ml.BearingTrackMode.none, // D-03: restores prior heading-up state
  );
}
```

**Error handling pattern** (source: `map_screen.dart` lines 280–288, T-16-02 fail-soft on malformed data — apply the same guard to breadcrumb GeoJSON updates):
```dart
try {
  await style.updateGeoJsonSource(id: 'breadcrumb', data: _breadcrumbGeoJson(navState.breadcrumb));
} catch (e) {
  debugPrint('navigation_screen: failed to update breadcrumb — $e');
}
```

**GPS stream / provider wiring — UNCHANGED** (source: current `navigation_screen.dart` lines 82–105, `initState`/`_positionSource`/`_sub`): `TraceletPositionSource` keeps feeding `navigationProvider`/`navigationStatsProvider` exactly as today. Do NOT try to feed it into `enableLocation` (Pitfall 2 — no stream parameter exists on that API).

---

### `app/lib/components/map/trail_layer.dart` (utility/component, transform)

**Analog:** itself — `addTrailTrackLayers` (lines 53–129) and `TrailMarkerLayer` (lines 145–237) are already the CORE-01/TRAIL-01..05 pattern used by `WandererMap`; navigation_screen just needs to become a second caller instead of using the legacy `TrailLayer` widget (lines 275–398, doc-commented for Phase-17 deletion at lines 270–274).

**Action:** Remove `TrailLayer`/`_TrailLayerState` (lines 275–398) and the `flutter_map`/`map_coordinate_adapter` imports (lines 5, 11) once `navigation_screen.dart` no longer references them. Keep `_buildCircularMarker` (lines 242–268) — shared by `TrailMarkerLayer`.

---

### Breadcrumb source/layer (new code, no new file)

**Analog:** `app/lib/components/map/cluster_layer.dart` — `addClusterLayers`/`updateClusterSource` (lines 21–95), the add-once-then-patch-in-place pattern.

**Core pattern** (source: `cluster_layer.dart` lines 21–22, 91–95):
```dart
// add once in onStyleLoaded:
await style.addSource(ml.GeoJsonSource(id: 'breadcrumb', data: geojson));
await style.addLayer(const ml.LineStyleLayer(id: 'breadcrumb-route', sourceId: 'breadcrumb', paint: {...}));

// update in place on every navState.breadcrumb change — RESEARCH.md Code Examples:
ref.listen(navigationProvider(widget.response), (prev, next) {
  final style = _controller?.style;
  if (style == null || prev?.breadcrumb == next.breadcrumb) return;
  style.updateGeoJsonSource(id: 'breadcrumb', data: jsonEncode({...}));
});
```
Do NOT use declarative `ml.PolylineLayer` in `MapLibreMap.layers` for this (Anti-Pattern — causes layer churn on every GPS fix, per RESEARCH.md and `map_screen.dart`'s existing single-use of `PolylineLayer` for the one-shot `_selectedPolyline`, lines 373–385, which is fine only because it changes rarely).

---

## Shared Patterns

### Native map lifecycle (onMapCreated/onStyleLoaded/style-swap)
**Source:** `app/lib/components/base/wanderer_map.dart` lines 85–201
**Apply to:** `navigation_screen.dart`'s new `ml.MapLibreMap` — capture `_controller` in `onMapCreated`, do all style-bound work (`addTrailTrackLayers`, breadcrumb source/layer, `enableLocation`/`trackLocation`) in `onStyleLoaded` since `setStyle` (CORE-02 theme swap) drops layers/sources/location-component bindings (Pitfall 4).

### Duration.zero crash avoidance
**Source:** `app/lib/components/base/wanderer_map.dart` lines 249–256 (comment + `nativeDuration: const Duration(milliseconds: 1)`), `app/lib/routes/map_screen.dart` lines 100, 155, 332 (750ms/400ms for animateCamera/fitBounds)
**Apply to:** All `animateCamera`/`fitBounds` calls in navigation_screen.dart (compass reset, recenter, initial camera fit) — never pass `Duration.zero`.

### Drag-vs-gesture distinction requires pointer-count heuristic, not `CameraChangeReason` alone
**Source:** `app/lib/routes/map_screen.dart` lines 308–313 (existing but insufficient — treats all `apiGesture` as a search-area trigger, doesn't need to distinguish drag/pinch); RESEARCH.md Pattern 2 (the correct, source-verified pointer-count wrapper)
**Apply to:** `navigation_screen.dart`'s follow-break detection (NAV-02) — this is genuinely new code, not a mechanical port; flag `checkpoint:human-verify` per RESEARCH.md Open Question 1.

### Fail-soft on malformed/async GeoJSON data
**Source:** `app/lib/routes/map_screen.dart` lines 280–288 (T-16-02)
**Apply to:** breadcrumb `updateGeoJsonSource` calls — wrap in try/catch, log and continue rather than crash live navigation.

### `ConsumerStatefulWidget` state field naming (`_followEnabled`, `_headingUp`)
**Source:** current `app/lib/routes/navigation_screen.dart` lines 74–80 — KEEP these field names and their independent-boolean semantics unchanged; only their native wiring (RHS of assignments) changes.

## No Analog Found

| File/Code | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `enableLocation`/`trackLocation` wiring (native location puck + follow) | service, event-driven | No prior phase has called these APIs — `WandererMap`/`map_screen.dart`'s `_LocationLayer`/`_buildLocationLayer` are hand-rolled static `WidgetLayer` dots fed by `foregroundPositionStreamProvider`, not the native follow-tracking component (CORE-07 explicitly requires moving off that pattern for navigation). Use RESEARCH.md's Pattern 1, Code Examples, and Pitfalls 2–5 (all source-verified against the installed `maplibre_android`/`maplibre_ios` package) as the authoritative reference instead of an in-repo analog. |
| Pointer-count `Listener` wrapper for drag detection | utility, event-driven | No in-repo precedent; `map_screen.dart` reads `CameraChangeReason.apiGesture` alone for a different, coarser purpose (triggering "search this area", where over-triggering on pinch is harmless) — navigation's drag-only requirement needs the finer distinction RESEARCH.md Pattern 2 provides. |

## Metadata

**Analog search scope:** `app/lib/components/base/`, `app/lib/components/map/`, `app/lib/routes/`, `app/lib/util/`, `app/lib/vendor/vector_map_tiles/`
**Files scanned:** `wanderer_map.dart`, `map_screen.dart`, `trail_layer.dart`, `cluster_layer.dart`, `navigation_screen.dart` (current), `map_compass.dart`, `17-CONTEXT.md`, `17-RESEARCH.md`
**Pattern extraction date:** 2026-07-10
