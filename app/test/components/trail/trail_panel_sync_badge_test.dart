// Widget test, deliberately not a source-grep test (36-21, closing the round
// 2 UAT gap: an unsynced trail's DETAIL screen showed the generic
// cache-provenance-gated "Offline" badge instead of its upload state). That
// badge no longer exists at all -- 38-04/D-10 deleted it and re-gated the
// surviving green pill onto library membership; Cases D/D2 below now pin
// that membership-derived behaviour.
//
// Why `TrailPanel` IS mountable here, with a `Trail.empty().copyWith(...)`
// fixture that carries `expand: null`:
// - `expand: null` => `trail.expand?.gpx == null` => the `TrailMap` /
//   `ElevationProfile` block (`trail_panel.dart`) is skipped entirely, so no
//   native `maplibre` platform view and no `onlineStatusProvider` read (the
//   only place the panel watches it lives inside that same block).
// - `expand: null` => no `TrailExpand.author` => no `ActorAvatar`;
//   `waypointsViaTrail ?? []` is empty => `TrailTimeline` early-returns
//   `SizedBox.shrink()` before it reads anything.
// - `photos`/`localPhotos` empty => no `PhotoCollage`.
// - `description: ''` => the `no_description_for_now` `Text` branch, not
//   `flutter_html`.
// - `category: null` => no `TrailCategoryLabel`.
// - For the unsynced fixtures `showsServerTabs` is `false` (WR-11's
//   `!isUnsyncedState(...)`), so there is no `TabBar` and `_TabContent` holds
//   only the About tab. For the synced fixtures it is `true`, so
//   `SummitLogList` and `CommentList` are CONSTRUCTED (both are `const`-
//   constructor `ConsumerWidget`s) but `_TabContentState.build` returns only
//   `children[_index]` (0 = About), so neither is ever built and
//   `summitLogListProvider`/`commentProvider` are never read.
//
// Three provider overrides, and what each stands in for:
// - `authProvider` -- `TrailPanel` opens with
//   `ref.watch(authProvider).requireValue!`. `requireValue` THROWS on
//   `AsyncLoading`, so `_StubAuth` overrides `build()` SYNCHRONOUSLY
//   (`UserEntity? build() => ...`, a legal covariant override of the
//   generated `FutureOr<UserEntity?> build()`), publishing `AsyncData`
//   immediately instead of the one-frame-loading async shape
//   `trail_dropdown_menu_test.dart`'s `_StubAuth` uses.
// - `unitProvider` -- a plain `$Provider<String>` with a generated
//   `overrideWithValue`; without it `settingsProvider` -> ObjectBox is
//   reached.
// - `trailSyncProvider` -- `_StubTrailSync`, copied from
//   `sync_status_chip_test.dart`, with a caller-supplied in-flight `Set` and
//   an `onRetry` recorder.

import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/components/trail/sync_status_chip.dart';
import 'package:wanderer/components/trail/trail_panel.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/provider/trail/trail_sync_provider.dart';

/// Copied from `trail_dropdown_menu_test.dart`'s fixture fields, but with a
/// SYNCHRONOUS `build()` -- see this file's header comment for why.
class _StubAuth extends Auth {
  @override
  UserEntity? build() => UserEntity(
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

/// Copied from `sync_status_chip_test.dart` -- caller-controlled in-flight
/// set, and a `retry` recorder instead of touching ObjectBox/the API.
class _StubTrailSync extends TrailSync {
  _StubTrailSync(this._initial, {this.onRetry});

  final Set<String> _initial;
  final void Function(String localId)? onRetry;

  @override
  Set<String> build() => _initial;

  @override
  Future<void> retry(String localId) async {
    onRetry?.call(localId);
  }
}

Widget _harness(
  Trail trail, {
  required ScrollController scrollController,
  bool availableOffline = false,
  Set<String> inFlight = const {},
  void Function(String)? onRetry,
}) {
  return ProviderScope(
    overrides: [
      authProvider.overrideWith(_StubAuth.new),
      unitProvider.overrideWithValue('metric'),
      trailSyncProvider.overrideWith(
        () => _StubTrailSync(inFlight, onRetry: onRetry),
      ),
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
      home: Scaffold(
        body: TrailPanel(
          trail: trail,
          scrollController: scrollController,
          availableOffline: availableOffline,
        ),
      ),
    ),
  );
}

void main() {
  final baseFixture = Trail.empty().copyWith(
    expand: null,
    photos: const [],
    localPhotos: const [],
    description: '',
    category: null,
  );

  testWidgets(
    'Case A -- unsynced, pending: shows Waiting to upload, exactly one '
    'SyncStatusChip, and no Offline text',
    (tester) async {
      final controller = ScrollController();
      addTearDown(controller.dispose);

      final trail = baseFixture.copyWith(
        id: '',
        localId: 'local-1-0',
        isLocal: true,
        syncState: TrailSyncState.pending,
      );

      await tester.pumpWidget(_harness(trail, scrollController: controller));
      await tester.pumpAndSettle();

      expect(find.text('Waiting to upload'), findsOneWidget);
      expect(find.byType(SyncStatusChip), findsOneWidget);
      expect(find.text('Offline'), findsNothing);
    },
  );

  testWidgets(
    'Case B -- unsynced, in-flight: shows Uploading… and a spinner',
    (tester) async {
      final controller = ScrollController();
      addTearDown(controller.dispose);

      final trail = baseFixture.copyWith(
        id: '',
        localId: 'local-1-0',
        isLocal: true,
        syncState: TrailSyncState.pending,
      );

      await tester.pumpWidget(
        _harness(
          trail,
          scrollController: controller,
          inFlight: {'local-1-0'},
        ),
      );
      // The indeterminate spinner animates forever -- pump a single frame
      // rather than pumpAndSettle, which would time out waiting for it to
      // stop (same reasoning as sync_status_chip_test.dart).
      await tester.pump();

      expect(find.text('Uploading…'), findsOneWidget);
      expect(find.byType(CircularProgressIndicator), findsOneWidget);
    },
  );

  testWidgets(
    'Case C -- unsynced, failed: shows Upload failed · Tap to retry, and '
    "tapping the chip's InkWell calls TrailSync.retry with the local id",
    (tester) async {
      final controller = ScrollController();
      addTearDown(controller.dispose);

      final trail = baseFixture.copyWith(
        id: '',
        localId: 'local-1-0',
        isLocal: true,
        syncState: TrailSyncState.failed,
      );

      String? retried;
      await tester.pumpWidget(
        _harness(
          trail,
          scrollController: controller,
          onRetry: (id) => retried = id,
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Upload failed · Tap to retry'), findsOneWidget);

      // Scoped to the chip's own InkWell -- the panel has other InkWells in
      // branches this fixture does not build, and scoping keeps the finder
      // honest if that changes.
      final inkWell = find.descendant(
        of: find.byType(SyncStatusChip),
        matching: find.byType(InkWell),
      );
      expect(inkWell, findsOneWidget);

      await tester.tap(inkWell);
      await tester.pumpAndSettle();

      expect(retried, 'local-1-0');
    },
  );

  testWidgets(
    'Case D control -- downloaded trail (cached AND library member): shows '
    'Available offline, renders NO SyncStatusChip at all, and renders none '
    'of the three sync strings',
    (tester) async {
      final controller = ScrollController();
      addTearDown(controller.dispose);

      final trail = baseFixture.copyWith(
        id: 'server-1',
        isLocal: true,
        syncState: TrailSyncState.synced,
      );

      await tester.pumpWidget(
        _harness(trail, scrollController: controller, availableOffline: true),
      );
      await tester.pumpAndSettle();

      // D-10 (38-04): the badge follows library membership, not the
      // cache-provenance flag -- the grey "Offline" pill no longer exists
      // at all.
      expect(find.text('Available offline'), findsOneWidget);
      expect(find.text('Offline'), findsNothing);
      // Load-bearing: the fix gates the chip's construction on
      // isUnsyncedState, so for a synced trail the widget is never built --
      // not merely built-and-self-suppressed. findsNothing proves the
      // stronger guarantee the round-2 UAT constraint demands.
      expect(find.byType(SyncStatusChip), findsNothing);
      expect(find.text('Waiting to upload'), findsNothing);
      expect(find.text('Uploading…'), findsNothing);
      expect(find.text('Upload failed · Tap to retry'), findsNothing);
    },
  );

  testWidgets(
    'Case D2 -- cached-looking trail NOT in this account\'s library: '
    'renders no stored-on-device badge at all',
    (tester) async {
      final controller = ScrollController();
      addTearDown(controller.dispose);

      // The invariant this pins: the badge follows library membership, not
      // cache provenance, so a timed-out fetch that leaves a downloaded-
      // looking cached model on screen cannot conjure a badge this account
      // has no claim to.
      final trail = baseFixture.copyWith(
        id: 'server-1',
        isLocal: true,
        syncState: TrailSyncState.synced,
      );

      await tester.pumpWidget(
        _harness(
          trail,
          scrollController: controller,
          availableOffline: false,
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Available offline'), findsNothing);
      expect(find.text('Offline'), findsNothing);
    },
  );

  testWidgets(
    'Case E control -- remote trail: shows Available offline, no Offline '
    'text, and no SyncStatusChip',
    (tester) async {
      final controller = ScrollController();
      addTearDown(controller.dispose);

      final trail = baseFixture.copyWith(
        id: 'server-2',
        isLocal: false,
        syncState: TrailSyncState.synced,
      );

      await tester.pumpWidget(
        _harness(trail, scrollController: controller, availableOffline: true),
      );
      await tester.pumpAndSettle();

      expect(find.text('Available offline'), findsOneWidget);
      expect(find.text('Offline'), findsNothing);
      expect(find.byType(SyncStatusChip), findsNothing);
    },
  );
}
