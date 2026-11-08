// Drives the REAL RoutePlannerScreen (real provider, segment layer, anchor
// layer, sheet) through anchor creations over a fake MapLibre platform and a
// fake Valhalla/geocoder, and counts what one creation costs.
//
// Background (.planning/debug/route-planner-anchor-pan-lag.md): on a Pixel the
// lag the user feels is the creation burst, not panning at rest. Every
// creation emits seven provider states; four of them re-uploaded the WHOLE
// route to MapLibre as one GeoJSON string over JNI (2 ms at rest, up to 27 ms
// inside a burst at ~570 points), two of those byte-identical because they
// only attached elevations, and two rebuilt every marker widget.
// ignore_for_file: depend_on_referenced_packages, implementation_imports
import 'dart:math' as math;

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:maplibre_platform_interface/maplibre_platform_interface.dart'
    as mpi;
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/provider/map_style_json_provider.dart';
import 'package:wanderer/provider/route_anchor_provider.dart';
import 'package:wanderer/routes/route_planner_screen.dart';
import 'package:wanderer/util/geo/polyline.dart';
import 'package:wanderer/util/route/segment.dart';
import 'package:wanderer/provider/foreground_position_stream_provider.dart';

import '../components/map/map_test_harness.dart';

class _FakeStyle extends ml.StyleController {
  int sourceAdds = 0;
  int layerAdds = 0;
  int uploads = 0;
  int identicalUploads = 0;
  final uploadBytes = <int>[];
  String? lastData;

  @override
  Future<void> addSource(ml.Source source) async => sourceAdds++;

  @override
  Future<void> addLayer(
    ml.StyleLayer layer, {
    String? belowLayerId,
    String? aboveLayerId,
    int? atIndex,
  }) async => layerAdds++;

  @override
  Future<void> updateGeoJsonSource({
    required String id,
    required String data,
  }) async {
    uploads++;
    uploadBytes.add(data.length);
    if (data == lastData) identicalUploads++;
    lastData = data;
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

class _FakeMapState extends mpi.MapLibreMapState {
  static _FakeMapState? current;

  final fakeStyle = _FakeStyle();
  final controller = FakeMapController();

  @override
  ml.StyleController? get style => fakeStyle;

  @override
  void initState() {
    super.initState();
    current = this;
    camera = cameraFor();
    isInitialized = true;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      widget.onMapCreated?.call(this);
      widget.onStyleLoaded?.call(fakeStyle);
    });
  }

  @override
  Widget buildPlatformWidget(BuildContext context) => const SizedBox.expand();

  /// One native camera event, as a pan produces.
  void emitCamera(ml.MapCamera cam) {
    setState(() => camera = cam);
    widget.onEvent?.call(ml.MapEventMoveCamera(camera: cam));
  }

  void click(ml.Geographic p) => widget.onEvent?.call(
    ml.MapEventClick(point: p, screenPoint: FakeMapController.project(p)),
  );

  @override
  ml.MapCamera getCamera() => camera!;

  @override
  Offset toScreenLocation(ml.Geographic g) => controller.toScreenLocation(g);

  @override
  ml.Geographic toLngLat(Offset o) => controller.toLngLat(o);

  @override
  List<Offset> toScreenLocations(List<ml.Geographic> l) =>
      controller.toScreenLocations(l);

  @override
  List<ml.RenderedFeature> featuresAtPoint(
    Offset point, {
    List<String>? layerIds,
  }) => const [];

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

class _FakePlatform extends mpi.MapLibrePlatform {
  @override
  mpi.MapLibreMapState createWidgetState() => _FakeMapState();
}

/// Points per routed segment: a realistic Valhalla leg is a few hundred.
const _pointsPerRoutedSegment = 200;

class _FakeApi extends Api {
  @override
  Dio build() {
    final dio = Dio(BaseOptions(baseUrl: 'https://test.local/api/v1'));
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          Response ok(Object data) =>
              Response(requestOptions: options, statusCode: 200, data: data);
          if (options.path.contains('/geocoding/reverse')) {
            handler.resolve(
              ok({
                'features': [
                  {
                    'properties': {
                      'address': {
                        'road': 'Road ${options.queryParameters['lat']}',
                        'city': 'Town',
                        'country': 'Land',
                      },
                    },
                  },
                ],
              }),
            );
          } else if (options.path.contains('/valhalla/height')) {
            final shape = (options.data as Map)['shape'] as List;
            handler.resolve(
              ok({'height': List<num>.generate(shape.length, (i) => 100 + i)}),
            );
          } else if (options.path.contains('/valhalla/route')) {
            final locs = (options.data as Map)['locations'] as List;
            final a = locs[0] as Map;
            final b = locs[1] as Map;
            final pts = <ml.Geographic>[
              for (var i = 0; i < _pointsPerRoutedSegment; i++)
                ml.Geographic(
                  lat:
                      (a['lat'] as double) +
                      ((b['lat'] as double) - (a['lat'] as double)) *
                          i /
                          (_pointsPerRoutedSegment - 1) +
                      (i.isEven ? 1e-5 : -1e-5),
                  lon:
                      (a['lon'] as double) +
                      ((b['lon'] as double) - (a['lon'] as double)) *
                          i /
                          (_pointsPerRoutedSegment - 1),
                ),
            ];
            handler.resolve(
              ok({
                'trip': {
                  'legs': [
                    {'shape': PolylineUtil.encode(pts, precision: 6)},
                  ],
                  'summary': {'time': 600},
                },
              }),
            );
          } else {
            handler.reject(
              DioException(
                requestOptions: options,
                type: DioExceptionType.unknown,
              ),
            );
          }
        },
      ),
    );
    return dio;
  }
}

/// Dio schedules its interceptor chain with Timer.run, so real time has to
/// elapse for the fake responses to land; then let the resulting state
/// emissions build.
Future<void> _settle(WidgetTester tester) async {
  for (var i = 0; i < 3; i++) {
    await tester.runAsync(
      () => Future<void>.delayed(const Duration(milliseconds: 40)),
    );
    await tester.pump(const Duration(milliseconds: 50));
  }
  await tester.pump(const Duration(seconds: 2));
}

class _Planner {
  _Planner(this.tester);

  final WidgetTester tester;
  final rebuilds = <String, int>{};
  late final ProviderContainer container;
  late final _FakeMapState map;

  _FakeStyle get style => map.fakeStyle;

  int rebuilt(String widgetType) => rebuilds[widgetType] ?? 0;

  Future<void> open() async {
    mpi.MapLibrePlatform.instance = _FakePlatform();
    tester.view.physicalSize = const Size(1080, 2000);
    tester.view.devicePixelRatio = 2.0;
    addTearDown(tester.view.reset);

    debugOnRebuildDirtyWidget = (element, builtOnce) {
      rebuilds.update(
        element.widget.runtimeType.toString(),
        (v) => v + 1,
        ifAbsent: () => 1,
      );
    };
    addTearDown(() => debugOnRebuildDirtyWidget = null);

    container = ProviderContainer(
      overrides: [
        apiProvider.overrideWith(() => _FakeApi()),
        mapStyleJsonProvider.overrideWith((ref) => '{}'),
        unitProvider.overrideWithValue('metric'),
        // The location button and marker layer would otherwise start the
        // real geolocator plugin, which has no implementation in tests.
        liveLocationProvider.overrideWith((ref) => const Stream.empty()),
      ],
    );
    addTearDown(container.dispose);

    await tester.pumpWidget(
      UncontrolledProviderScope(
        container: container,
        child: MaterialApp(
          localizationsDelegates: AppLocalizations.localizationsDelegates,
          supportedLocales: AppLocalizations.supportedLocales,
          home: RoutePlannerScreen(
            travelProfile: 'pedestrian',
            initialCenter: ml.Geographic(lat: 0, lon: 0),
          ),
        ),
      ),
    );
    await tester.pump();
    await tester.pump();
    await tester.pump();

    map = _FakeMapState.current!;
    expect(
      style.sourceAdds,
      1,
      reason: 'style-loaded must have mounted the segment layers once',
    );
  }

  /// Taps the map for the [k]-th anchor.
  void tap(int k) => map.click(
    ml.Geographic(lon: 0.0030 * k, lat: 0.0012 * math.sin(k.toDouble())),
  );
}

/// What the [k]-th creation cost, measured between the tap and the moment its
/// last network result has been applied.
class _Cost {
  _Cost(this.k, _Planner p, {required this.uploads, required this.identical})
    : screen = p.rebuilt('RoutePlannerScreen'),
      layer = p.rebuilt('RouteAnchorLayer'),
      flow = p.rebuilt('ScreenMarkerFlow'),
      markers = p.rebuilt('MapMarkerGestures');

  final int k;
  final int uploads;
  final int identical;
  final int screen;
  final int layer;
  final int flow;
  final int markers;
}

void main() {
  testWidgets(
    'a creation uploads the route only when its geometry changed, and what it '
    'costs does not grow with the number of earlier creations',
    (tester) async {
      final planner = _Planner(tester);
      await planner.open();

      const creations = 12;
      final costs = <int, _Cost>{};
      for (var k = 1; k <= creations; k++) {
        planner.rebuilds.clear();
        final uploads0 = planner.style.uploads;
        final identical0 = planner.style.identicalUploads;

        planner.tap(k);
        await tester.pump();
        await _settle(tester);

        costs[k] = _Cost(
          k,
          planner,
          uploads: planner.style.uploads - uploads0,
          identical: planner.style.identicalUploads - identical0,
        );
      }

      expect(costs[1]!.uploads, 0, reason: 'one anchor has no segment');
      for (var k = 2; k <= creations; k++) {
        final c = costs[k]!;
        expect(
          c.uploads,
          2,
          reason:
              'k=$k: the straight line, then the routed path. Was 4: the '
              'elevation results re-sent geometry the map already showed',
        );
        expect(c.identical, 0, reason: 'k=$k byte-identical uploads');
        expect(
          c.markers,
          lessThanOrEqualTo(k),
          reason:
              'k=$k: one rebuild of the marker set for the new anchor. Was '
              '2k: the reverse-geocode result rebuilt it again',
        );
      }

      // No accumulation: neither the number of uploads nor of screen, layer
      // and flow rebuilds per creation may creep up with the route's age (a
      // ref.listen that stacks, a leaked subscription).
      final reference = costs[4]!;
      for (var k = 5; k <= creations; k++) {
        final c = costs[k]!;
        expect(c.uploads, reference.uploads, reason: 'k=$k uploads');
        expect(c.screen, reference.screen, reason: 'k=$k screen rebuilds');
        expect(c.layer, reference.layer, reason: 'k=$k layer rebuilds');
        expect(c.flow, reference.flow, reason: 'k=$k flow rebuilds');
      }

      // Skipping must never leave the map behind: what it was last sent is
      // exactly the finished route.
      expect(
        planner.style.lastData,
        buildSegmentsGeoJson(
          planner.container.read(routeAnchorsProvider).segments,
        ),
      );
    },
  );

  testWidgets(
    'tapping again before the earlier results are in still costs two uploads '
    'per segment, not more',
    (tester) async {
      final planner = _Planner(tester);
      await planner.open();

      const taps = 6;
      for (var k = 1; k <= taps; k++) {
        planner.tap(k);
        await tester.pump();
      }
      await _settle(tester);

      expect(planner.style.uploads, 2 * (taps - 1));
      expect(planner.style.identicalUploads, 0);
      expect(
        planner.style.lastData,
        buildSegmentsGeoJson(
          planner.container.read(routeAnchorsProvider).segments,
        ),
      );
    },
  );

  testWidgets(
    'with auto-routing off a creation uploads once: the straight line',
    (tester) async {
      final planner = _Planner(tester);
      await planner.open();
      await planner.container
          .read(routeAnchorsProvider.notifier)
          .toggleAutoRouting();

      const taps = 5;
      for (var k = 1; k <= taps; k++) {
        planner.tap(k);
        await tester.pump();
        await _settle(tester);
      }

      expect(planner.style.uploads, taps - 1);
      expect(planner.style.identicalUploads, 0);
    },
  );
  testWidgets(
    'a pan after the creations rebuilds neither the screen, the markers nor '
    'the list, and projects a constant number of points per frame',
    (tester) async {
      final planner = _Planner(tester);
      await planner.open();
      const anchors = 12;
      for (var k = 1; k <= anchors; k++) {
        planner.tap(k);
        await tester.pump();
        await _settle(tester);
      }

      planner.rebuilds.clear();
      planner.map.controller.projections = 0;
      const frames = 10;
      for (var f = 0; f < frames; f++) {
        planner.map.emitCamera(
          ml.MapCamera(
            center: ml.Geographic(lon: 0.0001 * f, lat: 0),
            zoom: 10,
            bearing: 0,
            pitch: 0,
          ),
        );
        await tester.pump(const Duration(milliseconds: 11));
      }

      expect(planner.rebuilt('RoutePlannerScreen'), 0);
      expect(planner.rebuilt('MapMarkerGestures'), 0);
      expect(planner.rebuilt('ListTile'), 0);
      expect(planner.map.controller.projections / frames, 5);
    },
  );
}
