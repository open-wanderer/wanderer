// Widget test, deliberately not a source-grep test.
//
// 36-20 (WR-05, T-36-20-04): the drain's retired-id carry-forward (36-16)
// and the detail screen's redirect that reads it are pinned everywhere else
// in this phase by source-slicing gates (`local_trail_retirement_gate_test.
// dart`), because `_drainOne` and `retireUploadedLocalTrail` both mutate a
// live ObjectBox `Store` that `flutter test` cannot open. This screen branch
// is the one exception: `TrailDetailScreen.build()`'s `widget.localId != null
// && trail == null` branch reads `localTrailProvider` (a plain synchronous
// `@riverpod` family, trivially overridable with a value) and
// `TrailSync.serverIdForRetired` (a method on a `keepAlive` Notifier,
// trivially subclassable) -- and, critically, never builds `TrailPanel`, so
// no native `maplibre` platform view and no network-backed provider is ever
// reached. This file mounts the REAL `TrailDetailScreen` inside a REAL
// `GoRouter` and asserts on the router's actual settled location and the
// actual rendered text, matching `profile_trail_screen_navigation_test.dart`'s
// technique (a real router; assert on `router.state.uri`, not on pushed-call
// mocks).
//
// The sibling branch -- `trail != null`, i.e. the ordinary detail view -- is
// NOT mountable this way: `_buildDetail` unconditionally builds `TrailPanel`,
// which unconditionally embeds `TrailMap` (`trail_panel.dart:107`), which
// needs a live native MapLibre platform channel `flutter test` has no
// binding for. It also reads `trailLibraryProvider`, `downloadingTrailIdsProvider`
// and (through `TrailDropdown`) `authProvider`'s actor-id comparison -- all
// straightforward to stub, but the `TrailMap` platform-view dependency is
// not something a `ProviderScope` override can remove. Test 3 below is
// therefore the substitution the plan's own action text anticipates for
// exactly this situation, adjusted to match this screen's ACTUAL guard
// (`if (trail == null)`, not a `trailHasServerId`-style empty-string check --
// see that test's own comment for why the plan's literally-suggested
// "empty string is treated as no target" assertion does not hold against
// this source and what was substituted instead, recorded here rather than
// silently dropped per the plan's own instruction).

import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/trail/local_trail_provider.dart';
import 'package:wanderer/provider/trail/trail_sync_provider.dart';
import 'package:wanderer/routes/trail_detail_screen.dart';

/// A `TrailSync` subclass with a caller-supplied [serverIdForRetired] result,
/// and a call counter so a test can assert whether the redirect logic even
/// consulted it (Test 3, the mountability-substitution case). Mirrors
/// `test/routes/profile_trail_screen_navigation_test.dart`'s `_StubTrailSync`
/// (same `build() => {}` shape), extended with the one method this screen
/// actually calls.
class _StubTrailSync extends TrailSync {
  _StubTrailSync(this._result);

  final String? _result;
  int serverIdForRetiredCallCount = 0;

  @override
  Set<String> build() => {};

  @override
  String? serverIdForRetired(String localId) {
    serverIdForRetiredCallCount++;
    return _result;
  }
}

void main() {
  const localId = 'local-1-0';

  Future<GoRouter> pumpScreen(
    WidgetTester tester, {
    required Trail? localTrail,
    required String? retiredServerId,
  }) async {
    final trailSync = _StubTrailSync(retiredServerId);

    final router = GoRouter(
      initialLocation: '/trail/local/$localId',
      routes: [
        GoRoute(
          path: '/trail/local/:localId',
          builder: (context, state) =>
              const TrailDetailScreen(id: '', localId: localId),
        ),
        // A bare placeholder carrying a findable Key -- this test's subject
        // is the pushed location, not the destination screen's own
        // dependency graph (same convention as
        // profile_trail_screen_navigation_test.dart's bare-Scaffold routes).
        GoRoute(
          path: '/trail/:id',
          builder: (context, state) =>
              const Scaffold(key: Key('server-trail-placeholder')),
        ),
      ],
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          localTrailProvider(localId).overrideWithValue(localTrail),
          trailSyncProvider.overrideWith(() => trailSync),
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
    // The redirect fires from a post-frame callback
    // (`WidgetsBinding.instance.addPostFrameCallback`), so a single pump is
    // not enough to observe it -- pumpAndSettle drives frames until the
    // post-frame callback and the resulting route transition both land.
    await tester.pumpAndSettle();

    return router;
  }

  // Same accessor `profile_trail_screen_navigation_test.dart` settled on:
  // `GoRouter.state.uri` reflects a push made after the initial build in
  // this `MaterialApp.router` harness; `routerDelegate.currentConfiguration`
  // was verified NOT to, there.
  String currentLocation(GoRouter router) => router.state.uri.toString();

  testWidgets(
    'a retired local trail with a resolved server id redirects to /trail/<serverId>',
    (tester) async {
      final router = await pumpScreen(
        tester,
        localTrail: null,
        retiredServerId: 'server-1',
      );

      expect(currentLocation(router), '/trail/server-1');
      expect(find.byKey(const Key('server-trail-placeholder')), findsOneWidget);
      expect(
        find.text(
          AppLocalizations.of(tester.element(find.byType(Scaffold).first))!
              .trail_not_on_this_device,
        ),
        findsNothing,
      );
    },
  );

  testWidgets(
    'a retired local trail with no resolved server id stays on the dead end',
    (tester) async {
      final router = await pumpScreen(
        tester,
        localTrail: null,
        retiredServerId: null,
      );

      expect(currentLocation(router), '/trail/local/$localId');
      expect(
        find.text(
          AppLocalizations.of(tester.element(find.byType(Scaffold).first))!
              .trail_not_on_this_device,
        ),
        findsOneWidget,
      );
    },
  );

  testWidgets(
    'a present local trail never consults serverIdForRetired -- the redirect '
    'guard is `if (trail == null)`, not an empty-id check',
    (tester) async {
      // Mountability note (see this file's header comment): the plan's own
      // action text asked, as a fallback, for a case that keeps
      // `localTrailProvider` null and sets `serverIdForRetired` to the empty
      // string, asserting the empty string is treated as "no target" and the
      // dead-end text renders. That assertion does not hold against this
      // screen's ACTUAL source: `router_provider.dart:360-365`'s own comment
      // states that `'/trail/${trail.id}'` with an empty id "would emit
      // '/trail/', which go_router canonicalizes to '/trail', a path with no
      // route" -- so `trail_detail_screen.dart`'s
      // `if (retiredServerId != null)` guard, given an EMPTY STRING (which is
      // not null), would still attempt `context.pushReplacement('/trail/')`
      // and hit an unmatched route, not render `trail_not_on_this_device`.
      // The screen has no `trailHasServerId`-style non-empty check on this
      // path at all.
      //
      // The property actually worth pinning here -- and provable without
      // mounting the unmountable `trail != null` branch (see header) -- is
      // the guard's real shape: the redirect logic (and therefore
      // `serverIdForRetired`) is reached ONLY when `trail == null`.
      // `localTrailProvider` returning a non-null value must short-circuit
      // before `serverIdForRetired` is ever called, regardless of what it
      // would have returned. This is proven directly via the call counter on
      // `_StubTrailSync`, without needing `_buildDetail` (and therefore
      // `TrailPanel`/`TrailMap`) to mount at all.
      final trailSync = _StubTrailSync('server-should-never-be-read');

      final router = GoRouter(
        initialLocation: '/trail/local/$localId',
        routes: [
          GoRoute(
            path: '/trail/local/:localId',
            builder: (context, state) =>
                const TrailDetailScreen(id: '', localId: localId),
          ),
          GoRoute(
            path: '/trail/:id',
            builder: (context, state) =>
                const Scaffold(key: Key('server-trail-placeholder')),
          ),
        ],
      );

      final localTrail = Trail.empty().copyWith(
        id: '',
        localId: localId,
        name: 'A trail still on this device',
      );

      // `_buildDetail` (reached because `trail != null`) throws building
      // `trailLibraryProvider` (needs `objectBoxProvider`, deliberately NOT
      // overridden here -- this environment cannot open an ObjectBox
      // `Store` at all, per this file's header and every source-slicing
      // gate's own header in this phase) well before it would even reach
      // `TrailPanel`/`TrailMap`'s native platform-view dependency. That
      // throw is EXPECTED and is itself confirmation this test is exercising
      // the right code path -- `_buildDetail`, not the `trail == null`
      // branch. `tester.takeException()` consumes it explicitly (rather
      // than letting it fail the test) so the call-count assertion below,
      // the actual property under test, can run.
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            localTrailProvider(localId).overrideWithValue(localTrail),
            trailSyncProvider.overrideWith(() => trailSync),
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

      final caught = tester.takeException();
      expect(
        caught,
        isNotNull,
        reason:
            '_buildDetail was expected to throw in this environment (no '
            'ObjectBox Store, no native maplibre platform view) -- if it '
            'stops throwing, this branch may have become mountable and '
            'this test should be upgraded to assert on the rendered '
            'widgets directly instead of the call count below.',
      );

      expect(
        trailSync.serverIdForRetiredCallCount,
        0,
        reason:
            'serverIdForRetired must never be consulted when '
            'localTrailProvider already resolved a present trail -- the '
            'redirect branch is gated on `trail == null`, and calling it '
            'unconditionally would be at best wasted work and at worst a '
            'sign the guard\'s condition regressed. This assertion holds '
            'REGARDLESS of the expected _buildDetail crash above: by the '
            'time that crash happens, `TrailDetailScreen.build()` already '
            'evaluated `if (trail == null)`, found it false, and returned '
            'straight into `_buildDetail` without ever reaching the '
            'redirect branch\'s `serverIdForRetired` call.',
      );
    },
  );
}
