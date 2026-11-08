import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/components/base/actor_avatar.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/actor.dart';
import 'package:wanderer/models/list.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/list/list_filter_provider.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/provider/profile/profile_follows_provider.dart';
import 'package:wanderer/provider/profile/profile_lists_provider.dart';
import 'package:wanderer/provider/profile/profile_trails_provider.dart';
import 'package:wanderer/provider/trail/trail_filter_provider.dart';
import 'package:wanderer/provider/trail/trail_sync_provider.dart';
import 'package:wanderer/routes/profile_follow_screen.dart';
import 'package:wanderer/routes/profile_list_screen.dart';
import 'package:wanderer/routes/profile_trail_screen.dart';
import 'package:wanderer/util/trail/offline_filter_bounds.dart';

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

class _StubTrailFilter extends TrailFilterNotifier {
  @override
  Future<TrailFilter> build(String filterId) async =>
      buildDefaultTrailFilter(kOfflineTrailFilterValues);
}

class _HungListFilter extends ListFilterNotifier {
  @override
  Future<ListFilter> build(String filterId) => Completer<ListFilter>().future;
}

class _HungTrails extends ProfileTrailsNotifier {
  @override
  Future<ProfileTrailsState> build(String handle) =>
      Completer<ProfileTrailsState>().future;
}

class _HungLists extends ProfileListsNotifier {
  @override
  Future<ProfileListsState> build(String handle) =>
      Completer<ProfileListsState>().future;
}

class _StubFollows extends ProfileFollowsNotifier {
  _StubFollows(this._load);

  final Future<ProfileFollowsState> Function() _load;

  @override
  Future<ProfileFollowsState> build(String handle, String type) => _load();
}

ProfileFollowsState _follows() => ProfileFollowsState(
  items: [
    for (var i = 0; i < 40; i++)
      Actor.mock().copyWith(
        id: 'actor-$i',
        username: 'Real $i',
        preferredUsername: 'real$i',
        domain: null,
      ),
  ],
  page: 1,
  totalPages: 1,
);

void main() {
  final pixel = base64Decode(
    'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR4'
    '2mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==',
  );

  setUp(() => debugAvatarImageProviderFactory = (_) => MemoryImage(pixel));
  tearDown(() => debugAvatarImageProviderFactory = null);

  Widget app(Widget screen, List overrides) => ProviderScope(
    overrides: [
      authProvider.overrideWith(_StubAuth.new),
      unitProvider.overrideWithValue('metric'),
      trailSyncProvider.overrideWith(() => _StubTrailSync()),
      ...overrides.cast(),
    ],
    child: MaterialApp(
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      supportedLocales: AppLocalizations.supportedLocales,
      locale: const Locale('en'),
      home: screen,
    ),
  );

  Widget followApp(Future<ProfileFollowsState> Function() load) =>
      app(const ProfileFollowScreen(handle: '@tester', type: 'followers'), [
        profileFollowsProvider(
          '@tester',
          'followers',
        ).overrideWith(() => _StubFollows(load)),
      ]);

  testWidgets('trails show a skeleton while loading, not a spinner', (
    tester,
  ) async {
    await tester.pumpWidget(
      app(const ProfileTrailScreen(handle: '@tester'), [
        profileTrailsProvider('@tester').overrideWith(_HungTrails.new),
        trailFilterProvider('profile_trail_@tester')
            .overrideWith(_StubTrailFilter.new),
      ]),
    );
    await tester.pump(const Duration(milliseconds: 500));

    expect(find.byType(CircularProgressIndicator), findsNothing);
    expect(find.byType(LinearProgressIndicator), findsNothing);
    expect(find.text('Mock Trail Name'), findsWidgets);
    expect(find.byType(ActorAvatar), findsNothing);
  });

  testWidgets('lists show a skeleton while loading, not a spinner', (
    tester,
  ) async {
    await tester.pumpWidget(
      app(const ProfileListScreen(handle: '@tester'), [
        profileListsProvider('@tester').overrideWith(_HungLists.new),
        listFilterProvider('profile_list_@tester')
            .overrideWith(_HungListFilter.new),
      ]),
    );
    await tester.pump(const Duration(milliseconds: 500));

    expect(find.byType(CircularProgressIndicator), findsNothing);
    expect(find.byType(LinearProgressIndicator), findsNothing);
    expect(find.text('Mock Trail List'), findsWidgets);
  });

  testWidgets('follows show a skeleton while loading, not a spinner', (
    tester,
  ) async {
    await tester.pumpWidget(
      followApp(() => Completer<ProfileFollowsState>().future),
    );
    await tester.pump(const Duration(milliseconds: 500));

    expect(find.byType(CircularProgressIndicator), findsNothing);
    expect(find.text('Mock Username'), findsWidgets);
    expect(find.byType(ActorAvatar), findsNothing);
  });

  // REGRESSION GUARD. The skeleton and the real list share the screen during
  // the cross-fade. With the controller on both, this drag throws on
  // `ScrollController.position`.
  testWidgets('scrolling during the cross-fade is safe', (tester) async {
    final load = Completer<ProfileFollowsState>();
    await tester.pumpWidget(followApp(() => load.future));
    await tester.pump(const Duration(milliseconds: 500));

    load.complete(_follows());
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 20));
    expect(find.text('Mock Username'), findsWidgets);
    expect(find.text('Real 0'), findsOneWidget);

    await tester.drag(find.text('Real 0'), const Offset(0, -200));
    await tester.pump(const Duration(seconds: 1));

    expect(tester.takeException(), isNull);
    expect(find.text('Mock Username'), findsNothing);
  });

  testWidgets('a refresh keeps the list and its scroll position', (
    tester,
  ) async {
    var loads = 0;
    await tester.pumpWidget(
      followApp(
        () => loads++ == 0
            ? Future.value(_follows())
            : Completer<ProfileFollowsState>().future,
      ),
    );
    await tester.pump(const Duration(seconds: 1));

    await tester.drag(find.text('Real 0'), const Offset(0, -300));
    await tester.pump(const Duration(seconds: 1));
    final offset = tester
        .state<ScrollableState>(find.byType(Scrollable))
        .position
        .pixels;
    expect(offset, greaterThan(0));

    ProviderScope.containerOf(tester.element(find.byType(ProfileFollowScreen)))
        .invalidate(profileFollowsProvider('@tester', 'followers'));
    await tester.pump(const Duration(seconds: 1));

    expect(find.text('Mock Username'), findsNothing);
    expect(
      tester.state<ScrollableState>(find.byType(Scrollable)).position.pixels,
      offset,
    );
  });
}
