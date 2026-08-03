import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/components/trail/sync_status_chip.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/record.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/models/trail_summary.dart';
import 'package:wanderer/provider/trail/trail_sync_provider.dart';

/// Minimal `TrailSummary` fake -- only `syncState`/`localId` vary per test,
/// every other getter returns an inert default.
class _FakeTrail with RecordFunctions implements TrailSummary {
  _FakeTrail({required this.syncState, this.localId});

  @override
  final TrailSyncState syncState;

  @override
  final String? localId;

  @override
  String get id => localId ?? 'server-id';

  @override
  String get collectionId => 'trails';

  @override
  String get name => 'Test Trail';

  @override
  String? get domain => null;

  @override
  String? get location => null;

  @override
  double get distance => 0;

  @override
  double get duration => 0;

  @override
  double? get movingDuration => null;

  @override
  double get elevationGain => 0;

  @override
  double get elevationLoss => 0;

  @override
  bool get public => false;

  @override
  DateTime? get summaryDate => null;

  @override
  String get summaryThumbnail => '';

  @override
  int get summaryDifficulty => 0;

  @override
  String get summaryAuthorName => '';

  @override
  String get summaryAuthorAvatar => '';

  @override
  String? get summaryAuthorActorId => null;

  @override
  String? get categoryId => null;

  @override
  String? get subcategoryId => null;

  @override
  List<String>? get summaryShares => null;

  @override
  List<String>? get summaryTags => null;

  @override
  bool get isLocal => localId != null;

  @override
  List<String> get localPhotos => const [];
}

/// Stub `TrailSync` notifier -- returns a caller-controlled in-flight set
/// and records `retry` calls instead of touching ObjectBox/the API.
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
  Widget child, {
  Set<String> inFlight = const {},
  void Function(String)? onRetry,
}) {
  return ProviderScope(
    overrides: [
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
      home: Scaffold(body: child),
    ),
  );
}

void main() {
  testWidgets('a synced trail renders nothing', (tester) async {
    final trail = _FakeTrail(syncState: TrailSyncState.synced);
    await tester.pumpWidget(_harness(SyncStatusChip(trail: trail)));
    await tester.pumpAndSettle();

    expect(find.byType(SyncStatusChip), findsOneWidget);
    expect(
      find.descendant(
        of: find.byType(SyncStatusChip),
        matching: find.byType(Container),
      ),
      findsNothing,
    );
  });

  testWidgets(
    'a pending trail renders the sync_pending string and no spinner',
    (tester) async {
      final trail = _FakeTrail(
        syncState: TrailSyncState.pending,
        localId: 'local-1',
      );
      await tester.pumpWidget(_harness(SyncStatusChip(trail: trail)));
      await tester.pumpAndSettle();

      expect(find.text('Waiting to upload'), findsOneWidget);
      expect(find.byType(CircularProgressIndicator), findsNothing);
    },
  );

  testWidgets('an uploading trail renders a CircularProgressIndicator', (
    tester,
  ) async {
    final trail = _FakeTrail(
      syncState: TrailSyncState.uploading,
      localId: 'local-2',
    );
    // The indeterminate spinner animates forever, so pump a single frame
    // rather than pumpAndSettle (which would time out waiting for it to
    // stop).
    await tester.pumpWidget(_harness(SyncStatusChip(trail: trail)));
    await tester.pump();

    expect(find.byType(CircularProgressIndicator), findsOneWidget);
    expect(find.text('Uploading…'), findsOneWidget);
  });

  testWidgets(
    'a failed trail renders the sync_failed string, is tappable, and retries on tap',
    (tester) async {
      final trail = _FakeTrail(
        syncState: TrailSyncState.failed,
        localId: 'local-3',
      );
      String? retried;
      await tester.pumpWidget(
        _harness(
          SyncStatusChip(trail: trail),
          onRetry: (id) => retried = id,
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('Upload failed · Tap to retry'), findsOneWidget);
      final inkWell = find.descendant(
        of: find.byType(SyncStatusChip),
        matching: find.byType(InkWell),
      );
      expect(inkWell, findsOneWidget);

      await tester.tap(inkWell);
      await tester.pumpAndSettle();

      expect(retried, 'local-3');
    },
  );

  testWidgets(
    'a pending trail whose local id is in the in-flight set renders as Uploading',
    (tester) async {
      final trail = _FakeTrail(
        syncState: TrailSyncState.pending,
        localId: 'local-4',
      );
      // Same indeterminate-spinner reasoning as the Uploading-state test
      // above -- pump a single frame instead of pumpAndSettle.
      await tester.pumpWidget(
        _harness(SyncStatusChip(trail: trail), inFlight: {'local-4'}),
      );
      await tester.pump();

      expect(find.byType(CircularProgressIndicator), findsOneWidget);
      expect(find.text('Uploading…'), findsOneWidget);
    },
  );
}
