// Widget test, deliberately not a source-grep test.
//
// Same rationale as `profile_trail_screen_navigation_test.dart`, which this
// file is modelled on: a source-grep test for `trail_dropdown.dart` passed
// green for an entire phase while the menu it described was unreachable in
// the running app. So this pumps the real `ProfileTrailScreen` inside a real
// `GoRouter` and asserts the location the router actually ends up at after
// the map action button is tapped.
//
// The destination route is a bare placeholder rather than the real
// `ProfileTrailMapScreen`: mounting that screen would drag in MapLibre, the
// style-JSON providers and the two scoped search families, none of which is
// this test's subject. The subject is solely that the AppBar action exists
// and reaches `/profile/@tester/trails/map`.

import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/provider/profile/profile_constants.dart';
import 'package:wanderer/provider/profile/profile_trails_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';
import 'package:wanderer/provider/trail/trail_sync_provider.dart';
import 'package:wanderer/routes/profile_trail_screen.dart';
import 'package:wanderer/util/trail/offline_filter_bounds.dart';

/// Stub Auth notifier returning a fixture [UserEntity] synchronously, so the
/// screen renders without touching ObjectBox or the network. Copied from
/// `profile_trail_screen_navigation_test.dart`, keeping
/// `preferredUsername: 'tester'` so the screen's handle `'@tester'` matches.
class _StubAuth extends Auth {
  @override
  Future<UserEntity?> build() async => UserEntity(
    id: 'user-test-id',
    collectionId: 'users',
    collectionName: 'users',
    actorId: 'actor-id',
    username: 'tester',
    preferredUsername: 'tester',
    email: 'tester@example.com',
    iri: 'https://example.com/u/tester',
    serverUrl: 'https://example.com',
    created: DateTime(2024),
    updated: DateTime(2024),
    avatar: null,
  );
}

/// Stub `TrailSync` notifier returning a fixed empty in-flight set.
class _StubTrailSync extends TrailSync {
  @override
  Set<String> build() => {};
}

/// Fixed [ProfileTrailsNotifier] returning a caller-supplied state, so this
/// test's subject is the pushed location, not the provider's own network or
/// merge behaviour.
class _StubProfileTrails extends ProfileTrailsNotifier {
  _StubProfileTrails(this._state);

  final ProfileTrailsState _state;

  @override
  Future<ProfileTrailsState> build(String handle) async => _state;
}

/// Fixed [TrailFilterNotifier] returning the offline-fallback default filter.
class _StubFilter extends TrailFilterNotifier {
  @override
  Future<TrailFilter> build(String filterId) async =>
      buildDefaultTrailFilter(kOfflineTrailFilterValues);
}

void main() {
  final syncedTrail = Trail.empty().copyWith(
    name: 'Synced Trail',
    id: 'server-1',
    syncState: TrailSyncState.synced,
  );

  Future<GoRouter> pumpScreen(WidgetTester tester) async {
    tester.view.physicalSize = const Size(1080, 4000);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final state = ProfileTrailsState(
      trails: [syncedTrail],
      page: 1,
      perPage: kProfileSearchPerPage,
      totalPages: 1,
      offline: false,
      isOwnHandle: true,
    );

    final router = GoRouter(
      initialLocation: '/profile/@tester/trails',
      routes: [
        GoRoute(
          path: '/profile/:handle/trails',
          builder: (context, routerState) =>
              ProfileTrailScreen(handle: routerState.pathParameters['handle']!),
          routes: [
            // Placeholder standing in for ProfileTrailMapScreen -- see the
            // header comment for why the real screen is not mounted here.
            GoRoute(
              path: 'map',
              builder: (context, routerState) => const Scaffold(),
            ),
          ],
        ),
        GoRoute(
          path: '/trail/:id',
          builder: (context, routerState) => const Scaffold(),
        ),
      ],
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          profileTrailsProvider(
            '@tester',
          ).overrideWith(() => _StubProfileTrails(state)),
          trailFilterProvider(
            'profile_trail_@tester',
          ).overrideWith(() => _StubFilter()),
          authProvider.overrideWith(_StubAuth.new),
          unitProvider.overrideWithValue('metric'),
          trailSyncProvider.overrideWith(() => _StubTrailSync()),
        ],
        child: MaterialApp.router(
          routerConfig: router,
          localizationsDelegates: const [
            AppLocalizations.delegate,
            GlobalMaterialLocalizations.delegate,
            GlobalWidgetsLocalizations.delegate,
            GlobalCupertinoLocalizations.delegate,
          ],
          supportedLocales: AppLocalizations.supportedLocales,
          locale: const Locale('en'),
        ),
      ),
    );
    await tester.pumpAndSettle();

    return router;
  }

  // `GoRouter.state.uri` is the router's live "where am I now" accessor --
  // unlike `routerDelegate.currentConfiguration.uri`, which in this
  // MaterialApp.router harness does not reflect a push made after the
  // initial build.
  String currentLocation(GoRouter router) => router.state.uri.toString();

  testWidgets('the profile trails AppBar exposes a map action', (tester) async {
    await pumpScreen(tester);

    expect(
      find.byTooltip('Map'),
      findsOneWidget,
    );
  });

  testWidgets('tapping the map action pushes /profile/<handle>/trails/map', (
    tester,
  ) async {
    final router = await pumpScreen(tester);

    await tester.tap(
      find.byTooltip('Map'),
    );
    await tester.pumpAndSettle();

    expect(currentLocation(router), '/profile/@tester/trails/map');
  });

  testWidgets('the map action does not disturb trail row navigation', (
    tester,
  ) async {
    final router = await pumpScreen(tester);

    await tester.tap(find.text('Synced Trail'));
    await tester.pumpAndSettle();

    expect(currentLocation(router), '/trail/server-1');
  });
}
