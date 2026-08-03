// Widget test, deliberately not a source-grep test.
//
// UAT recorded that `trail_dropdown_delete_gate_test.dart` greps the source
// of `trail_dropdown.dart` and therefore passed green for the entire phase
// while the menu it described was unreachable in the running app. Nothing
// caught the divert. This test pumps the real `ProfileTrailScreen` inside a
// real `GoRouter` and asserts the location the router actually ends up at --
// the automated signal that was missing.

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

/// Stub Auth notifier that returns a fixture [UserEntity] synchronously so
/// the screen renders without touching ObjectBox or the network.
///
/// Copied from `test/routes/settings_account_screen_test.dart`, keeping
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

/// Stub `TrailSync` notifier -- returns a fixed empty in-flight set, copied
/// from `test/components/trail/sync_status_chip_test.dart`.
class _StubTrailSync extends TrailSync {
  @override
  Set<String> build() => {};
}

/// Fixed [ProfileTrailsNotifier] returning a caller-supplied state, so this
/// test's subject is the pushed location, not the provider's own network/
/// merge behaviour.
class _StubProfileTrails extends ProfileTrailsNotifier {
  _StubProfileTrails(this._state);

  final ProfileTrailsState _state;

  @override
  Future<ProfileTrailsState> build(String handle) async => _state;
}

/// Fixed [TrailFilterNotifier] returning the offline-fallback default
/// filter, per 36-09's `buildDefaultTrailFilter`/`kOfflineTrailFilterValues`.
class _StubFilter extends TrailFilterNotifier {
  @override
  Future<TrailFilter> build(String filterId) async =>
      buildDefaultTrailFilter(kOfflineTrailFilterValues);
}

void main() {
  final unsyncedTrail = Trail.empty().copyWith(
    name: 'Unsynced Trail',
    id: '',
    localId: 'local-1-0',
    syncState: TrailSyncState.pending,
  );
  final syncedTrail = Trail.empty().copyWith(
    name: 'Synced Trail',
    id: 'server-1',
    syncState: TrailSyncState.synced,
  );

  Future<GoRouter> pumpScreen(WidgetTester tester) async {
    // The screen may carry more rows than a default viewport shows -- give
    // the tester a tall viewport so both rows mount, same pattern as
    // settings_account_screen_test.dart.
    tester.view.physicalSize = const Size(1080, 4000);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final state = ProfileTrailsState(
      trails: [unsyncedTrail, syncedTrail],
      page: 1,
      perPage: kProfileSearchPerPage,
      totalPages: 1,
      offline: true,
      isOwnHandle: true,
    );

    final router = GoRouter(
      initialLocation: '/profile/@tester/trails',
      routes: [
        GoRoute(
          path: '/profile/:handle/trails',
          builder: (context, routerState) => ProfileTrailScreen(
            handle: routerState.pathParameters['handle']!,
          ),
        ),
        // Bare placeholders: this test's subject is the pushed location,
        // nothing else, so neither route mounts TrailDetailScreen's own
        // dependency graph.
        GoRoute(
          path: '/trail/local/:localId',
          builder: (context, routerState) => const Scaffold(),
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

  // `GoRouter.state.uri` is the router's own live "where am I now" accessor
  // -- unlike `routerDelegate.currentConfiguration.uri`, which in this
  // MaterialApp.router harness was observed NOT to reflect a push made
  // after the initial build (verified empirically while writing this test).
  String currentLocation(GoRouter router) => router.state.uri.toString();

  testWidgets(
    'tapping an unsynced trail pushes /trail/local/<localId>, not /trail/create/edit',
    (tester) async {
      final router = await pumpScreen(tester);

      await tester.tap(find.text('Unsynced Trail'));
      await tester.pumpAndSettle();

      expect(currentLocation(router), '/trail/local/local-1-0');
      expect(currentLocation(router), isNot('/trail/create/edit'));
    },
  );

  testWidgets('tapping a synced trail still pushes /trail/<id>', (
    tester,
  ) async {
    final router = await pumpScreen(tester);

    await tester.tap(find.text('Synced Trail'));
    await tester.pumpAndSettle();

    expect(currentLocation(router), '/trail/server-1');
  });
}
