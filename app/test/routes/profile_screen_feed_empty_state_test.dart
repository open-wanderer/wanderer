// Widget test, deliberately not a source-grep test: it pumps the real
// `ProfileScreen` inside a real `GoRouter` and asserts what the feed section
// actually renders (and where the New Trail button actually navigates).
//
// Covers the profile feed empty state: shown only once the feed has loaded
// with zero items, never while loading and never on error.

import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/actor.dart';
import 'package:wanderer/models/follow.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/provider/profile/follow_provider.dart';
import 'package:wanderer/provider/profile/profile_counts_provider.dart';
import 'package:wanderer/provider/profile/profile_feed_provider.dart';
import 'package:wanderer/provider/profile/profile_lists_provider.dart';
import 'package:wanderer/provider/profile/profile_provider.dart';
import 'package:wanderer/provider/trail/trail_sync_provider.dart';
import 'package:wanderer/routes/profile_screen.dart';

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

class _StubTrailSync extends TrailSync {
  @override
  Set<String> build() => {};
}

Actor _actor({
  required String id,
  required String username,
  required String preferredUsername,
}) => Actor(
  id: id,
  collectionId: 'activitypub_actors',
  collectionName: 'activitypub_actors',
  created: DateTime(2024),
  updated: DateTime(2024),
  username: username,
  preferredUsername: preferredUsername,
  // Non-null counts: a null count renders a never-settling Skeletonizer.
  followerCount: 0,
  followingCount: 0,
  iri: 'https://example.com/u/$preferredUsername',
  inbox: 'https://example.com/u/$preferredUsername/inbox',
  publicKey: 'key',
  lastFetched: '2024-01-01 00:00:00.000Z',
  user: 'user-$preferredUsername',
);

class _StubOwnProfile extends OwnProfile {
  @override
  Future<Actor> build() async {
    // Like the real provider: the screen derives its own handle from auth,
    // so auth must have resolved by the time the actor does.
    await ref.watch(authProvider.future);
    return _actor(
      id: 'actor-id',
      username: 'Tester',
      preferredUsername: 'tester',
    );
  }
}

class _StubProfile extends ProfileNotifier {
  @override
  Future<Actor> build(String handle) async =>
      _actor(id: 'actor-alice', username: 'Alice', preferredUsername: 'alice');
}

class _StubFeed extends ProfileFeedNotifier {
  _StubFeed(this._factory);

  final Future<ProfileFeedState> Function() _factory;

  @override
  Future<ProfileFeedState> build(String handle) => _factory();
}

class _StubLists extends ProfileListsNotifier {
  @override
  Future<ProfileListsState> build(String handle) async =>
      const ProfileListsState(lists: [], page: 1, perPage: 5, totalPages: 1);
}

class _StubFollow extends FollowNotifier {
  // Synchronous on purpose: the loading-state follow button overflows the
  // 60px box it is given, which is unrelated to this test's subject.
  @override
  FutureOr<FollowState> build(String profileActorId) =>
      const FollowState(isFollowing: false);
}

const _emptyFeed = ProfileFeedState(
  items: [],
  page: 1,
  perPage: 10,
  totalPages: 1,
  totalItems: 0,
);

void main() {
  Future<GoRouter> pumpScreen(
    WidgetTester tester, {
    required String location,
    required Future<ProfileFeedState> Function() feed,
    bool settle = true,
  }) async {
    // The CustomScrollView builds slivers lazily and the feed sits below the
    // fold, so give the tester a tall viewport.
    tester.view.physicalSize = const Size(1080, 4000);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    // The follow button's fixed 100px slot overflows under the test
    // environment's wide fallback font. That is a layout artefact unrelated
    // to this test's subject, so drop overflow reports and forward the rest.
    final originalOnError = FlutterError.onError;
    FlutterError.onError = (details) {
      if (details.exceptionAsString().contains('overflowed')) return;
      originalOnError?.call(details);
    };
    addTearDown(() => FlutterError.onError = originalOnError);

    final router = GoRouter(
      initialLocation: location,
      routes: [
        GoRoute(
          path: '/profile',
          builder: (context, routerState) => const ProfileScreen(),
        ),
        GoRoute(
          path: '/profile/:handle',
          builder: (context, routerState) =>
              ProfileScreen(handle: routerState.pathParameters['handle']),
        ),
        GoRoute(
          path: '/trail/create',
          builder: (context, routerState) => const Scaffold(),
        ),
        GoRoute(
          path: '/settings/account',
          builder: (context, routerState) => const Scaffold(),
        ),
      ],
    );

    const counts = ProfileCounts(trailCount: 0, listCount: 0);

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          authProvider.overrideWith(_StubAuth.new),
          unitProvider.overrideWithValue('metric'),
          trailSyncProvider.overrideWith(() => _StubTrailSync()),
          ownProfileProvider.overrideWith(() => _StubOwnProfile()),
          profileProvider('@alice').overrideWith(() => _StubProfile()),
          profileFeedProvider('@tester').overrideWith(() => _StubFeed(feed)),
          profileFeedProvider('@alice').overrideWith(() => _StubFeed(feed)),
          profileListsProvider('@tester').overrideWith(() => _StubLists()),
          profileListsProvider('@alice').overrideWith(() => _StubLists()),
          profileCountsProvider('actor-id').overrideWith((ref) async => counts),
          profileCountsProvider(
            'actor-alice',
          ).overrideWith((ref) async => counts),
          followProvider('actor-alice').overrideWith(() => _StubFollow()),
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
    if (settle) {
      await tester.pumpAndSettle();
    } else {
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));
    }
    return router;
  }

  String currentLocation(GoRouter router) => router.state.uri.toString();

  testWidgets('other profile with an empty feed shows the username line', (
    tester,
  ) async {
    await pumpScreen(
      tester,
      location: '/profile/@alice',
      feed: () async => _emptyFeed,
    );

    expect(find.text('alice has no activity yet.'), findsOneWidget);
    expect(find.text('New Trail'), findsNothing);
    expect(find.text('Feed'), findsOneWidget);
  });

  testWidgets('own profile with an empty feed shows the message and CTA', (
    tester,
  ) async {
    final router = await pumpScreen(
      tester,
      location: '/profile',
      feed: () async => _emptyFeed,
    );

    expect(find.text('You have no activity yet.'), findsOneWidget);
    expect(find.text('New Trail'), findsOneWidget);
    expect(find.text('Feed'), findsOneWidget);

    await tester.tap(find.text('New Trail'));
    await tester.pumpAndSettle();

    expect(currentLocation(router), '/trail/create');
  });

  testWidgets('no empty state while the feed is still loading', (tester) async {
    final never = Completer<ProfileFeedState>();
    await pumpScreen(
      tester,
      location: '/profile/@alice',
      feed: () => never.future,
      settle: false,
    );

    expect(find.text('alice has no activity yet.'), findsNothing);
    expect(find.text('You have no activity yet.'), findsNothing);
  });

  testWidgets('no empty state when the feed fails to load', (tester) async {
    await pumpScreen(
      tester,
      location: '/profile/@alice',
      feed: () async => throw Exception('boom'),
      settle: false,
    );

    expect(find.text('alice has no activity yet.'), findsNothing);
    expect(find.text('You have no activity yet.'), findsNothing);
  });

  testWidgets('a non-empty feed renders its items, not the empty state', (
    tester,
  ) async {
    await pumpScreen(
      tester,
      location: '/profile/@alice',
      feed: () async => ProfileFeedState.mock(),
    );

    expect(find.text('Mock Trail Name'), findsWidgets);
    expect(find.text('alice has no activity yet.'), findsNothing);
    expect(find.text('You have no activity yet.'), findsNothing);
  });

  testWidgets('own profile without a bio offers an Add Bio button', (
    tester,
  ) async {
    final router = await pumpScreen(
      tester,
      location: '/profile',
      feed: () async => _emptyFeed,
    );

    expect(find.text('No bio yet.'), findsOneWidget);
    expect(find.text('Add Bio'), findsOneWidget);

    await tester.tap(find.text('Add Bio'));
    await tester.pumpAndSettle();

    expect(currentLocation(router), '/settings/account');
  });

  testWidgets('other profile without a bio has no Add Bio button', (
    tester,
  ) async {
    await pumpScreen(
      tester,
      location: '/profile/@alice',
      feed: () async => _emptyFeed,
    );

    expect(find.text('No bio yet.'), findsOneWidget);
    expect(find.text('Add Bio'), findsNothing);
  });
}
