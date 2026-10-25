// Android "Open with" hands the app a VIEW intent whose data URI is a
// content:// document. Flutter forwards that URI to the router as a deep link
// (initial platform route on a cold start, `pushRouteInformation` when the
// singleTask activity is already running), and the router has no route for it:
//
//   GoException: no routes for location: content://media/external/downloads/126256
//
// The file itself is delivered separately by receive_sharing_intent, so the
// router must simply ignore anything that is not one of its own locations.
//
// These are widget tests against a REAL GoRouter configured with the same two
// options as `Router.build()` (`overridePlatformDefaultLocation` + the
// `ignoreInboundUris` guard). The platform side is simulated at the same seam
// the engine uses: `PlatformDispatcher.defaultRouteName` for a cold start, the
// `flutter/navigation` channel for a warm one. The last group pins that the
// production router really passes those options, since `Router.build()` reads
// Riverpod providers and cannot be mounted here.

import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/util/inbound_uri_guard.dart';

/// URIs an OS can hand the app instead of a route.
const _inboundUris = <String, String>{
  'Android downloads content URI (the reported one)':
      'content://media/external/downloads/126256',
  'Android SAF document URI':
      'content://com.android.externalstorage.documents/document/primary%3ADownload%2Ftrack.gpx',
  // A path that happens to equal a real app route must not navigate there.
  'content URI whose path matches an app route': 'content://media/settings',
  'Android file URI': 'file:///storage/emulated/0/Download/track.gpx',
  'iOS file URL': 'file:///private/var/mobile/Containers/track.gpx',
  'iOS share extension hand-off':
      'ShareMedia-com.openwanderer.wanderer://dataUrl=ShareKey',
};

class _Refresh extends ChangeNotifier {
  void ping() => notifyListeners();
}

GoRouter _buildRouter({Listenable? refreshListenable}) {
  return GoRouter(
    initialLocation: '/',
    overridePlatformDefaultLocation: true,
    onEnter: ignoreInboundUris,
    refreshListenable: refreshListenable,
    routes: [
      GoRoute(path: '/', builder: (context, state) => const Text('home')),
      GoRoute(
        path: '/settings',
        builder: (context, state) => const Text('settings'),
      ),
    ],
  );
}

Future<void> _pump(WidgetTester tester, GoRouter router) async {
  await tester.pumpWidget(MaterialApp.router(routerConfig: router));
  await tester.pumpAndSettle();
}

/// Delivers [location] the way the Android embedding does from `onNewIntent`.
Future<void> _pushFromPlatform(WidgetTester tester, String location) async {
  final message = const JSONMethodCodec().encodeMethodCall(
    MethodCall('pushRouteInformation', <String, dynamic>{'location': location}),
  );
  await tester.binding.defaultBinaryMessenger.handlePlatformMessage(
    'flutter/navigation',
    message,
    (_) {},
  );
  await tester.pumpAndSettle();
}

void main() {
  group('cold start with an inbound URI as the platform route', () {
    for (final entry in _inboundUris.entries) {
      testWidgets('stays on the home route: ${entry.key}', (tester) async {
        tester.platformDispatcher.defaultRouteNameTestValue = entry.value;
        addTearDown(tester.platformDispatcher.clearDefaultRouteNameTestValue);
        final router = _buildRouter();
        addTearDown(router.dispose);

        await _pump(tester, router);

        expect(find.text('home'), findsOneWidget);
        expect(find.textContaining('no routes for location'), findsNothing);
        expect(router.state.uri.toString(), '/');
      });
    }

    testWidgets('a plain path as the platform route is not honoured either', (
      tester,
    ) async {
      // The app registers no app links, so nothing legitimate arrives this way;
      // pin that the cold start is always '/', as the router redirect expects.
      tester.platformDispatcher.defaultRouteNameTestValue = '/settings';
      addTearDown(tester.platformDispatcher.clearDefaultRouteNameTestValue);
      final router = _buildRouter();
      addTearDown(router.dispose);

      await _pump(tester, router);

      expect(find.text('home'), findsOneWidget);
    });
  });

  group('warm start: inbound URI pushed while the app is running', () {
    for (final entry in _inboundUris.entries) {
      testWidgets('leaves the current screen alone: ${entry.key}', (
        tester,
      ) async {
        final router = _buildRouter();
        addTearDown(router.dispose);
        await _pump(tester, router);
        router.go('/settings');
        await tester.pumpAndSettle();
        expect(find.text('settings'), findsOneWidget);

        await _pushFromPlatform(tester, entry.value);

        expect(find.text('settings'), findsOneWidget);
        expect(find.textContaining('no routes for location'), findsNothing);
        expect(router.state.uri.toString(), '/settings');
      });
    }

    testWidgets('a later router refresh does not resurrect the blocked URI', (
      tester,
    ) async {
      // GoRouteInformationProvider keeps the last platform-reported value; the
      // app's refreshListenable (auth flips, splash release) re-parses it. A
      // blocked URI must not come back through that door.
      final refresh = _Refresh();
      addTearDown(refresh.dispose);
      final router = _buildRouter(refreshListenable: refresh);
      addTearDown(router.dispose);
      await _pump(tester, router);
      router.go('/settings');
      await tester.pumpAndSettle();

      await _pushFromPlatform(tester, _inboundUris.values.first);
      refresh.ping();
      await tester.pumpAndSettle();

      expect(find.text('settings'), findsOneWidget);
      expect(find.textContaining('no routes for location'), findsNothing);
      expect(router.state.uri.toString(), '/settings');
    });
  });

  group('the guard does not swallow ordinary navigation', () {
    testWidgets('in-app go() still navigates', (tester) async {
      final router = _buildRouter();
      addTearDown(router.dispose);
      await _pump(tester, router);

      router.go('/settings');
      await tester.pumpAndSettle();

      expect(find.text('settings'), findsOneWidget);
    });

    testWidgets('a platform-pushed plain path still navigates', (tester) async {
      final router = _buildRouter();
      addTearDown(router.dispose);
      await _pump(tester, router);

      await _pushFromPlatform(tester, '/settings');

      expect(find.text('settings'), findsOneWidget);
    });

    testWidgets('a genuinely unknown path still reaches the error page', (
      tester,
    ) async {
      final router = _buildRouter();
      addTearDown(router.dispose);
      await _pump(tester, router);

      router.go('/does-not-exist');
      await tester.pumpAndSettle();

      expect(
        find.textContaining('no routes for location: /does-not-exist'),
        findsOneWidget,
      );
    });
  });

  group('isInboundUri', () {
    test('is true for every URI an OS can hand the app', () {
      for (final uri in _inboundUris.values) {
        expect(isInboundUri(Uri.parse(uri)), isTrue, reason: uri);
      }
    });

    test('is false for the locations the app navigates to', () {
      for (final location in [
        '/',
        '/map',
        '/trail/abc123',
        '/trail/local/local-1-0/map',
        '/settings/region/map?path=eu%2Falps',
        '/profile/jane@wanderer.to',
        '/profile/jane@wanderer.to/trails',
        '/list/xyz',
      ]) {
        expect(isInboundUri(Uri.parse(location)), isFalse, reason: location);
      }
    });
  });

  group('production router wiring', () {
    test('Router.build() passes both halves of the guard', () {
      final source = File(
        'lib/provider/router_provider.dart',
      ).readAsStringSync();

      // Cold start: ignore the platform's initial route (the VIEW intent URI).
      expect(source, contains('overridePlatformDefaultLocation: true'));
      // Warm start: block inbound URIs pushed while the app is running.
      expect(source, contains('onEnter: ignoreInboundUris'));
    });
  });
}
