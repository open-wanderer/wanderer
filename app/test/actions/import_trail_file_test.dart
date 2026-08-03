import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:gpx/gpx.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/actions/import_trail_file.dart';

/// Fakes any Dio traffic `transcodeToGpx`/`buildLocalTrail`'s reverse-geocode
/// step might issue. Mirrors the `_FakeApi` pattern in
/// `test/util/route_planner_handoff_util_test.dart` /
/// `test/provider/route_anchor_provider_test.dart`.
class _FakeApi extends Api {
  _FakeApi({this.response, this.shouldFail = false});

  final dynamic response;
  final bool shouldFail;

  int requestCount = 0;

  @override
  Dio build() {
    final dio = Dio(BaseOptions(baseUrl: 'https://test.local/api/v1'));
    dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: (options, handler) {
          requestCount++;
          if (shouldFail) {
            handler.reject(
              DioException(
                requestOptions: options,
                type: DioExceptionType.connectionError,
              ),
            );
            return;
          }
          handler.resolve(
            Response(requestOptions: options, statusCode: 200, data: response),
          );
        },
      ),
    );
    return dio;
  }
}

class _FakeOnlineStatus extends OnlineStatus {
  _FakeOnlineStatus(this._initial);

  final bool _initial;

  int refreshCalls = 0;

  @override
  bool build() => _initial;

  /// Deterministic stand-in for the real `isBackendReachable` probe, plus the
  /// counter that pins OFFUI-04's fix.
  ///
  /// `OnlineStatus` is optimistic (`build() => true`) and only settles from
  /// ordinary traffic, so in airplane mode with no request yet attempted it
  /// still reads online. `importTrailFile` therefore refreshes BEFORE its
  /// non-GPX guard; without that, the guard silently misses and the user gets
  /// the generic import error instead of the offline-specific one — exactly
  /// what OFFUI-04 exists to prevent.
  @override
  Future<bool> refresh() async {
    refreshCalls++;
    state = _initial;
    return _initial;
  }
}

/// Pumps a bare `Consumer` inside a `ProviderScope` overriding [apiProvider]
/// with [_FakeApi] and [onlineStatusProvider] with [_FakeOnlineStatus], and
/// hands back the captured [WidgetRef].
Future<WidgetRef> _pumpRef(
  WidgetTester tester, {
  dynamic response,
  bool shouldFail = false,
  required bool online,
}) async {
  late WidgetRef capturedRef;
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        apiProvider.overrideWith(
          () => _FakeApi(response: response, shouldFail: shouldFail),
        ),
        onlineStatusProvider.overrideWith(() => _FakeOnlineStatus(online)),
      ],
      child: Consumer(
        builder: (context, ref, _) {
          capturedRef = ref;
          return const SizedBox();
        },
      ),
    ),
  );
  return capturedRef;
}

/// Runs [body] inside [WidgetTester.runAsync] — Dio schedules its
/// interceptor pipeline via a real `Timer.run`, which `testWidgets`' fake
/// clock never fires on its own. See the identical rationale in
/// `route_planner_handoff_util_test.dart`.
Future<T> _runAsync<T>(WidgetTester tester, Future<T> Function() body) async {
  final result = await tester.runAsync(body);
  if (result == null) {
    throw StateError('tester.runAsync returned null');
  }
  return result;
}

Gpx _buildSampleGpx() {
  final gpx = Gpx();
  gpx.trks = [
    Trk(
      trksegs: [
        Trkseg(
          trkpts: [
            Wpt(lat: 47.000, lon: 9.000, ele: 400),
            Wpt(lat: 47.001, lon: 9.001, ele: 410),
          ],
        ),
      ],
    ),
  ];
  return gpx;
}

void main() {
  group('buildLocalTrail', () {
    testWidgets('offline: issues no request and leaves location null', (
      tester,
    ) async {
      final gpx = _buildSampleGpx();
      // shouldFail:true doubles as a "no request was attempted" guard —
      // if buildLocalTrail regressed to calling the api while offline,
      // this fake would reject and the test would surface a DioException.
      final ref = await _pumpRef(tester, online: false, shouldFail: true);

      final trail = await _runAsync(tester, () => buildLocalTrail(ref, gpx));

      expect(trail.location, isNull);
    });

    testWidgets('online: fills location from the reverse-geocode response '
        "(D-07's fullLabel/includeRoad:false convention)", (tester) async {
      final gpx = _buildSampleGpx();
      final ref = await _pumpRef(
        tester,
        online: true,
        response: {
          'features': [
            {
              'properties': {
                'address': {'city': 'Testtown', 'country': 'Testland'},
              },
            },
          ],
        },
      );

      final trail = await _runAsync(tester, () => buildLocalTrail(ref, gpx));

      expect(trail.location, 'Testtown, Testland');
    });

    testWidgets(
      'online, geocode throws: the trail is still returned with location '
      'null (best-effort, never blocks the save)',
      (tester) async {
        final gpx = _buildSampleGpx();
        final ref = await _pumpRef(tester, online: true, shouldFail: true);

        final trail = await _runAsync(tester, () => buildLocalTrail(ref, gpx));

        expect(trail.location, isNull);
      },
    );

    testWidgets(
      'passes movingDuration through to trail.movingDuration and leaves '
      'duration GPX-derived',
      (tester) async {
        final gpx = _buildSampleGpx();
        final ref = await _pumpRef(tester, online: false, shouldFail: true);

        final trail = await _runAsync(
          tester,
          () => buildLocalTrail(
            ref,
            gpx,
            movingDuration: const Duration(minutes: 20),
          ),
        );

        expect(trail.movingDuration, 1200.0);
        // buildSampleGpx carries no <time>, so GPX-derived duration is 0.
        expect(trail.duration, 0);
      },
    );
  });

  group('transcodeToGpx', () {
    late Directory tempDir;

    setUp(() {
      tempDir = Directory.systemTemp.createTempSync('transcode_test');
    });

    tearDown(() {
      if (tempDir.existsSync()) tempDir.deleteSync(recursive: true);
    });

    String writeTempFile(String contents) {
      final file = File('${tempDir.path}/track.kml');
      file.writeAsStringSync(contents);
      return file.path;
    }

    testWidgets(
      'returns the body verbatim when the response is a raw GPX String',
      (tester) async {
        final path = writeTempFile('<kml></kml>');
        final ref = await _pumpRef(
          tester,
          online: false,
          response: '<gpx><trk><trkseg></trkseg></trk></gpx>',
        );

        final gpxXml = await _runAsync(
          tester,
          () => transcodeToGpx(ref, path, 'track.kml'),
        );

        expect(gpxXml, '<gpx><trk><trkseg></trkseg></trk></gpx>');
      },
    );

    testWidgets(
      'returns expand.gpx_data when the response is the legacy JSON Map',
      (tester) async {
        final path = writeTempFile('<kml></kml>');
        final ref = await _pumpRef(
          tester,
          online: false,
          response: {
            'expand': {'gpx_data': '<gpx><trk></trk></gpx>'},
          },
        );

        final gpxXml = await _runAsync(
          tester,
          () => transcodeToGpx(ref, path, 'track.kml'),
        );

        expect(gpxXml, '<gpx><trk></trk></gpx>');
      },
    );

    testWidgets(
      'throws when neither response shape yields a non-empty string',
      (tester) async {
        final path = writeTempFile('<kml></kml>');
        final ref = await _pumpRef(
          tester,
          online: false,
          response: {'expand': <String, dynamic>{}},
        );

        Object? caught;
        await tester.runAsync(() async {
          try {
            await transcodeToGpx(ref, path, 'track.kml');
          } catch (e) {
            caught = e;
          }
        });

        expect(caught, isA<StateError>());
      },
    );
  });

  group('importTrailFile', () {
    late Directory tempDir;

    setUp(() {
      tempDir = Directory.systemTemp.createTempSync('offline_import_test');
    });

    tearDown(() {
      if (tempDir.existsSync()) tempDir.deleteSync(recursive: true);
      pendingImportedTrail = null;
    });

    String writeTempGpx() {
      final file = File('${tempDir.path}/track.gpx');
      file.writeAsStringSync('''
<?xml version="1.0"?>
<gpx><trk><trkseg>
<trkpt lat="47.000" lon="9.000"><ele>400</ele></trkpt>
<trkpt lat="47.001" lon="9.001"><ele>410</ele></trkpt>
</trkseg></trk></gpx>
''');
      return file.path;
    }

    /// Pumps a minimal `GoRouter` (rather than `_pumpRef`'s bare `Consumer`)
    /// so `importTrailFile`'s `navContext.push('/trail/create/edit', ...)`
    /// handoff has somewhere real to land.
    Future<({WidgetRef ref, BuildContext context, _FakeOnlineStatus status})>
    pumpRouterRef(
      WidgetTester tester, {
      required bool online,
      bool shouldFailAll = false,
    }) async {
      late WidgetRef capturedRef;
      late BuildContext capturedContext;
      final status = _FakeOnlineStatus(online);
      final router = GoRouter(
        initialLocation: '/',
        routes: [
          GoRoute(
            path: '/',
            builder: (context, state) => Consumer(
              builder: (context, ref, _) {
                capturedRef = ref;
                capturedContext = context;
                return const SizedBox.shrink();
              },
            ),
          ),
          GoRoute(
            path: '/trail/create/edit',
            builder: (context, state) => const SizedBox.shrink(),
          ),
        ],
      );
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            apiProvider.overrideWith(() => _FakeApi(shouldFail: shouldFailAll)),
            onlineStatusProvider.overrideWith(() => status),
          ],
          child: MaterialApp.router(
            routerConfig: router,
            localizationsDelegates: AppLocalizations.localizationsDelegates,
            supportedLocales: AppLocalizations.supportedLocales,
          ),
        ),
      );
      return (ref: capturedRef, context: capturedContext, status: status);
    }

    testWidgets(
      'offline: importing a .gpx performs zero HTTP requests end to end '
      'and still produces a trail (D-15\'s most important branch)',
      (tester) async {
        final path = writeTempGpx();
        final pumped = await pumpRouterRef(
          tester,
          online: false,
          shouldFailAll: true,
        );
        final l10n = AppLocalizations.of(pumped.context)!;

        await tester.runAsync(
          () => importTrailFile(
            ref: pumped.ref,
            path: path,
            name: 'track.gpx',
            navContext: pumped.context,
            l10n: l10n,
          ),
        );
        await tester.pumpAndSettle();

        expect(pumped.ref.read(toastProvider), isEmpty);
        expect(pendingImportedTrail, isNotNull);
        expect(pendingImportedTrail!.expand?.gpxData, contains('<gpx'));
      },
    );

    // WR-12: the try/catch used to span the navigation push as well, so a
    // failure could leave a stale non-null `pendingImportedTrail` behind. The
    // catch now ends at `buildLocalTrail`, and a genuine parse failure must
    // toast AND leave the handoff global untouched.
    testWidgets('a structurally broken GPX toasts an error and leaves '
        'pendingImportedTrail untouched', (tester) async {
      // A <trkpt> with no lat/lon no longer throws — the vendored reader
      // tolerates it per-field so one bad point cannot cost a whole file.
      // A file with NO usable coordinate is still unimportable though, and
      // importTrailFile's own emptiness guard is what rejects it now.
      final file = File('${tempDir.path}/broken.gpx');
      file.writeAsStringSync(
        '<?xml version="1.0"?><gpx><trk><trkseg>'
        '<trkpt><ele>400</ele></trkpt>'
        '</trkseg></trk></gpx>',
      );

      final pumped = await pumpRouterRef(
        tester,
        online: false,
        shouldFailAll: true,
      );
      final l10n = AppLocalizations.of(pumped.context)!;

      await tester.runAsync(
        () => importTrailFile(
          ref: pumped.ref,
          path: file.path,
          name: 'broken.gpx',
          navContext: pumped.context,
          l10n: l10n,
        ),
      );
      await tester.pumpAndSettle();

      expect(pumped.ref.read(toastProvider), isNotEmpty);
      expect(pendingImportedTrail, isNull);
    });

    // OFFUI-04. The message has to name the actual constraint ("only GPX can
    // be imported offline"), not the generic "could not import file" — the
    // whole point of the requirement is that a hiker in airplane mode learns
    // WHY their .kml was refused and what would have worked.
    testWidgets(
      'offline: a non-GPX import is refused with the offline-specific '
      'message, before any network call',
      (tester) async {
        final file = File('${tempDir.path}/track.kml');
        file.writeAsStringSync('<kml></kml>');

        final pumped = await pumpRouterRef(
          tester,
          online: false,
          // Doubles as a "no request was attempted" guard: if the guard
          // regressed and the transcode ran, this fake would reject and the
          // toast below would be the generic error instead.
          shouldFailAll: true,
        );
        final l10n = AppLocalizations.of(pumped.context)!;

        await tester.runAsync(
          () => importTrailFile(
            ref: pumped.ref,
            path: file.path,
            name: 'track.kml',
            navContext: pumped.context,
            l10n: l10n,
          ),
        );
        await tester.pumpAndSettle();

        expect(
          pumped.ref.read(toastProvider).single.text,
          l10n.trail_source_offline_import_error,
        );
        expect(pendingImportedTrail, isNull);
        // The guard is only sound because the optimistic status was refreshed
        // first — see _FakeOnlineStatus.refresh.
        expect(pumped.status.refreshCalls, greaterThan(0));
      },
    );

    testWidgets(
      'online: the offline guard does not fire for a non-GPX file — a later '
      'failure reports the generic import error instead',
      (tester) async {
        final file = File('${tempDir.path}/track.kml');
        file.writeAsStringSync('<kml></kml>');

        final pumped = await pumpRouterRef(
          tester,
          online: true,
          shouldFailAll: true,
        );
        final l10n = AppLocalizations.of(pumped.context)!;

        await tester.runAsync(
          () => importTrailFile(
            ref: pumped.ref,
            path: file.path,
            name: 'track.kml',
            navContext: pumped.context,
            l10n: l10n,
          ),
        );
        await tester.pumpAndSettle();

        expect(
          pumped.ref.read(toastProvider).single.text,
          l10n.trail_source_import_error,
        );
      },
    );

    testWidgets(
      'offline: a .gpx import is NOT caught by the non-GPX guard and still '
      'produces a trail',
      (tester) async {
        final path = writeTempGpx();
        final pumped = await pumpRouterRef(
          tester,
          online: false,
          shouldFailAll: true,
        );
        final l10n = AppLocalizations.of(pumped.context)!;

        await tester.runAsync(
          () => importTrailFile(
            ref: pumped.ref,
            path: path,
            name: 'track.gpx',
            navContext: pumped.context,
            l10n: l10n,
          ),
        );
        await tester.pumpAndSettle();

        expect(pumped.ref.read(toastProvider), isEmpty);
        expect(pendingImportedTrail, isNotNull);
      },
    );
  });

  group('PORT-03 gate', () {
    test('PORT-03: exactly one "trail/convert" occurrence exists in app/lib/, '
        'inside actions/import_trail_file.dart', () {
      final libDir = Directory('lib');
      expect(
        libDir.existsSync(),
        isTrue,
        reason:
            'This test must be run with `flutter test`\'s working '
            'directory set to "app/" (e.g. "cd app && flutter test").',
      );

      final matches = <String>[];
      for (final entity in libDir.listSync(recursive: true)) {
        if (entity is! File || !entity.path.endsWith('.dart')) continue;
        final lines = entity.readAsLinesSync();
        for (final line in lines) {
          if (line.contains('trail/convert')) {
            matches.add(entity.path);
          }
        }
      }

      expect(
        matches,
        hasLength(1),
        reason:
            'PORT-03 requires the app to call the convert endpoint from '
            'exactly one place. Found matches: $matches',
      );
      expect(
        matches.single.replaceAll('\\', '/'),
        endsWith('actions/import_trail_file.dart'),
        reason:
            'PORT-03: the sole call site must live in '
            'actions/import_trail_file.dart',
      );
    });
  });
}
