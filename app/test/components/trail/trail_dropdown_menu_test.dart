// UAT found that the existing delete gate test (`trail_dropdown_delete_gate_
// test.dart`) greps the SOURCE TEXT of `trail_dropdown.dart`, so it stayed
// green for the entire phase while the menu it described could not be opened
// at all in the running app: `TrailDropdown` is instantiated in exactly one
// place, `trail_detail_screen.dart:98`, which the own-trails list diverted
// away from for every unsynced trail (36-11/36-12 fixed the divert). All of
// D-14's delete gating and D-17's download hiding shipped with no automated
// signal that it was live.
//
// This file is the missing signal: it mounts a real `TrailDropdown` inside an
// `AppBar` (matching its only real call site), opens the real
// `PopupMenuButton`, and asserts on the rendered `PopupMenuItem`s -- it fails
// if an item is absent when it shouldn't be, disabled/enabled the wrong way,
// or shows the wrong confirmation copy. No assertion here reads the source
// text of any lib file.

import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/components/trail/trail_dropdown.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/trail/trail_download_state_provider.dart';
import 'package:wanderer/provider/trail/trail_sync_provider.dart';

/// Copied from `test/routes/settings_account_screen_test.dart` -- a stub
/// `Auth` notifier returning a fixture user synchronously so `_canEditTrail`
/// resolves without touching ObjectBox or the network. `actorId` matches
/// this file's fixtures' `author` field so Edit renders.
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

/// Copied from `test/components/trail/sync_status_chip_test.dart` -- a stub
/// `TrailSync` notifier returning a caller-supplied in-flight set.
class _StubTrailSync extends TrailSync {
  _StubTrailSync(this._initial);

  final Set<String> _initial;

  @override
  Set<String> build() => _initial;
}

/// Stub for the `keepAlive` download-state notifier -- a plain caller-
/// supplied `Set<String>` of currently-downloading trail ids.
class _StubDownloadingTrailIds extends DownloadingTrailIds {
  _StubDownloadingTrailIds(this._initial);

  final Set<String> _initial;

  @override
  Set<String> build() => _initial;
}

Widget _harness(
  Trail trail, {
  Set<String> inFlight = const {},
  Set<String> downloading = const {},
}) {
  return ProviderScope(
    overrides: [
      authProvider.overrideWith(_StubAuth.new),
      trailSyncProvider.overrideWith(() => _StubTrailSync(inFlight)),
      downloadingTrailIdsProvider.overrideWith(
        () => _StubDownloadingTrailIds(downloading),
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
      // Mounting inside an AppBar matches the only real call site
      // (trail_detail_screen.dart:98) and gives PopupMenuButton the anchor
      // it expects.
      home: Scaffold(
        appBar: AppBar(
          actions: [TrailDropdown(trail: trail, availableOffline: false)],
        ),
        // TrailDropdown's own build() only watches downloadingTrailIdsProvider
        // and trailSyncProvider eagerly; authProvider is read lazily, inside
        // _canEditTrail/_allowDelete, which only run when the popup menu's
        // itemBuilder fires. Since @riverpod providers build lazily on first
        // read, an unprimed authProvider is still AsyncLoading (.value ==
        // null) at that exact moment, so Edit/Delete would render as absent
        // on the very first menu open. In the real app this is a non-issue:
        // `trail_panel.dart` (rendered on the same screen as TrailDropdown,
        // see trail_detail_screen.dart) already watches authProvider via
        // `.requireValue!` -- it only renders once auth has resolved, which
        // primes the provider well before the dropdown is ever tapped. This
        // inert Consumer reproduces that priming instead of the app's real
        // widget (which needs a full trail fixture and other scaffolding).
        body: Consumer(
          builder: (context, ref, _) {
            ref.watch(authProvider);
            return const SizedBox.shrink();
          },
        ),
      ),
    ),
  );
}

Future<void> _openMenu(WidgetTester tester) async {
  // Let the (async) _StubAuth.build() future resolve and propagate before
  // interacting -- otherwise _canEditTrail/_allowDelete read a still-loading
  // AsyncValue whose .value is null.
  await tester.pump();
  await tester.tap(find.byType(PopupMenuButton<TrailAction>));
  await tester.pumpAndSettle();
}

void main() {
  // lat/lon set so the `directions` entry renders enabled and does not skew
  // a disabled-item assertion.
  final unsynced = Trail.empty().copyWith(
    id: '',
    localId: 'local-1-0',
    author: 'actor-id',
    name: 'Unsynced Trail',
    syncState: TrailSyncState.pending,
    lat: 1,
    lon: 1,
  );
  final synced = Trail.empty().copyWith(
    id: 'server-1',
    author: 'actor-id',
    name: 'Synced Trail',
    syncState: TrailSyncState.synced,
    lat: 1,
    lon: 1,
  );
  // WR-08: an unsynced trail with no local handle at all -- the shape
  // `retireUploadedLocalTrail`'s demote branch and `TrailDownloadService`'s
  // carry-forward can both produce for a row that already has a real
  // server id. The confirm dialog and the executor must agree it is a
  // server-routed delete, not the (silently no-op) un-download path.
  final unsyncedNoLocalHandle = Trail.empty().copyWith(
    id: 'server-2',
    localId: null,
    author: 'actor-id',
    name: 'Unsynced, No Local Handle',
    syncState: TrailSyncState.pending,
    lat: 1,
    lon: 1,
  );

  testWidgets(
    'D-17: an unsynced trail hides Download and "Available offline", but '
    'shows Show on map, Edit and Delete',
    (tester) async {
      await tester.pumpWidget(_harness(unsynced));
      await _openMenu(tester);

      expect(find.text('Download'), findsNothing);
      expect(find.text('Available offline'), findsNothing);
      expect(find.text('Show on map'), findsOneWidget);
      expect(find.text('Edit'), findsOneWidget);
      expect(find.text('Delete'), findsOneWidget);
    },
  );

  testWidgets(
    'D-17 control: a synced trail DOES show Download -- proves the previous '
    'assertion is about sync state, not a harness that renders nothing',
    (tester) async {
      await tester.pumpWidget(_harness(synced));
      await _openMenu(tester);

      expect(find.text('Download'), findsOneWidget);
    },
  );

  testWidgets(
    'D-14: deleting an unsynced trail shows the unrecoverable confirm copy',
    (tester) async {
      await tester.pumpWidget(_harness(unsynced));
      await _openMenu(tester);

      await tester.tap(
        find.widgetWithText(PopupMenuItem<TrailAction>, 'Delete'),
      );
      await tester.pumpAndSettle();

      expect(
        find.text(
          "Delete this trail? It hasn't been uploaded yet, so this can't be undone.",
        ),
        findsOneWidget,
      );
      expect(
        find.text(
          'Do you really want to delete this trail? This action cannot be undone.',
        ),
        findsNothing,
      );
    },
  );

  testWidgets(
    'D-14 control: deleting a synced trail shows the reversible confirm copy',
    (tester) async {
      await tester.pumpWidget(_harness(synced));
      await _openMenu(tester);

      await tester.tap(
        find.widgetWithText(PopupMenuItem<TrailAction>, 'Delete'),
      );
      await tester.pumpAndSettle();

      expect(
        find.text(
          'Do you really want to delete this trail? This action cannot be undone.',
        ),
        findsOneWidget,
      );
    },
  );

  testWidgets(
    'D-14: Delete is disabled while the unsynced trail is mid-drain',
    (tester) async {
      await tester.pumpWidget(_harness(unsynced, inFlight: {'local-1-0'}));
      await _openMenu(tester);

      final deleteItem = tester.widget<PopupMenuItem<TrailAction>>(
        find.widgetWithText(PopupMenuItem<TrailAction>, 'Delete'),
      );
      expect(deleteItem.enabled, isFalse);
    },
  );

  testWidgets(
    'Show on map is enabled for an unsynced trail -- trailMapLocation '
    'resolves it to /trail/local/local-1-0/map, a real route as of 36-12',
    (tester) async {
      await tester.pumpWidget(_harness(unsynced));
      await _openMenu(tester);

      final openItem = tester.widget<PopupMenuItem<TrailAction>>(
        find.widgetWithText(PopupMenuItem<TrailAction>, 'Show on map'),
      );
      expect(openItem.enabled, isTrue);
    },
  );

  testWidgets(
    'WR-08: an unsynced trail with localId: null and a real server id shows '
    'the reversible delete_trail_confirm copy -- the confirm dialog and the '
    'now-server-routed executor agree',
    (tester) async {
      await tester.pumpWidget(_harness(unsyncedNoLocalHandle));
      await _openMenu(tester);

      await tester.tap(
        find.widgetWithText(PopupMenuItem<TrailAction>, 'Delete'),
      );
      await tester.pumpAndSettle();

      expect(
        find.text(
          'Do you really want to delete this trail? This action cannot be undone.',
        ),
        findsOneWidget,
      );
      expect(
        find.text(
          "Delete this trail? It hasn't been uploaded yet, so this can't be undone.",
        ),
        findsNothing,
      );
    },
  );

  testWidgets(
    'A synced trail\'s menu is unchanged -- control case for the WR-08 fix',
    (tester) async {
      await tester.pumpWidget(_harness(synced));
      await _openMenu(tester);

      expect(find.text('Download'), findsOneWidget);
      expect(find.text('Show on map'), findsOneWidget);
      expect(find.text('Edit'), findsOneWidget);
      expect(find.text('Delete'), findsOneWidget);
    },
  );
}
