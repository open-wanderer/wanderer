import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/components/trail/sync_status_chip.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/record.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/models/trail_summary.dart';
import 'package:wanderer/provider/trail/trail_sync_provider.dart';
import 'package:wanderer/theme/sync_state_colors.dart';
import 'package:wanderer/theme/theme.dart';

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
  ThemeData? theme,
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
      theme: theme,
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
        _harness(SyncStatusChip(trail: trail), onRetry: (id) => retried = id),
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
    'a draft trail renders a pen-to-square icon and the Draft label, no '
    'spinner',
    (tester) async {
      final trail = _FakeTrail(
        syncState: TrailSyncState.draft,
        localId: 'local-5',
      );
      await tester.pumpWidget(_harness(SyncStatusChip(trail: trail)));
      await tester.pumpAndSettle();

      expect(find.text('Draft'), findsOneWidget);
      final icon = tester.widget<FaIcon>(find.byType(FaIcon));
      expect(icon.icon?.codePoint, FontAwesomeIcons.penToSquare.codePoint);
      expect(find.byType(CircularProgressIndicator), findsNothing);

      final colors = syncStateColors(
        TrailSyncState.draft,
        Theme.of(tester.element(find.text('Draft'))),
      );
      final chip = tester.widget<Container>(
        find.ancestor(of: find.text('Draft'), matching: find.byType(Container)),
      );
      expect((chip.decoration as BoxDecoration).color, colors.container);
      expect(icon.color, colors.onContainer);
    },
  );

  testWidgets('a draft chip is not tappable and never retries', (tester) async {
    final trail = _FakeTrail(
      syncState: TrailSyncState.draft,
      localId: 'local-5',
    );
    String? retried;
    await tester.pumpWidget(
      _harness(SyncStatusChip(trail: trail), onRetry: (id) => retried = id),
    );
    await tester.pumpAndSettle();

    expect(
      find.descendant(
        of: find.byType(SyncStatusChip),
        matching: find.byType(InkWell),
      ),
      findsNothing,
    );
    await tester.tap(find.text('Draft'));
    await tester.pumpAndSettle();
    expect(retried, isNull);
  });

  testWidgets('a draft whose local id is (impossibly) in flight still reads '
      'Draft', (tester) async {
    final trail = _FakeTrail(
      syncState: TrailSyncState.draft,
      localId: 'local-6',
    );
    await tester.pumpWidget(
      _harness(SyncStatusChip(trail: trail), inFlight: {'local-6'}),
    );
    await tester.pumpAndSettle();

    expect(find.text('Draft'), findsOneWidget);
    expect(find.text('Uploading…'), findsNothing);
  });

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

  for (final brightness in Brightness.values) {
    testWidgets('a failed chip has the red container and no hardcoded red '
        '(${brightness.name})', (tester) async {
      final theme = AppTheme.createTheme(brightness);
      final trail = _FakeTrail(
        syncState: TrailSyncState.failed,
        localId: 'local-3',
      );
      await tester.pumpWidget(
        _harness(SyncStatusChip(trail: trail), theme: theme),
      );
      await tester.pumpAndSettle();

      final colors = syncStateColors(TrailSyncState.failed, theme);
      final container = tester.widget<Container>(
        find.ancestor(
          of: find.text('Upload failed · Tap to retry'),
          matching: find.byType(Container),
        ),
      );
      expect((container.decoration as BoxDecoration).color, colors.container);
      final label = tester.widget<Text>(
        find.text('Upload failed · Tap to retry'),
      );
      expect(label.style?.color, colors.onContainer);
      final icon = tester.widget<FaIcon>(find.byType(FaIcon));
      expect(icon.color, colors.onContainer);
    });
  }

  testWidgets('a pending chip takes its colors from the helper', (
    tester,
  ) async {
    final theme = AppTheme.createTheme(Brightness.light);
    final trail = _FakeTrail(
      syncState: TrailSyncState.pending,
      localId: 'local-1',
    );
    await tester.pumpWidget(
      _harness(SyncStatusChip(trail: trail), theme: theme),
    );
    await tester.pumpAndSettle();

    final colors = syncStateColors(TrailSyncState.pending, theme);
    final container = tester.widget<Container>(
      find.ancestor(
        of: find.text('Waiting to upload'),
        matching: find.byType(Container),
      ),
    );
    expect((container.decoration as BoxDecoration).color, colors.container);
    expect(
      tester.widget<Text>(find.text('Waiting to upload')).style?.color,
      colors.onContainer,
    );
    expect(tester.widget<FaIcon>(find.byType(FaIcon)).color, colors.icon);
  });

  testWidgets('an uploading chip spinner uses the helper icon color', (
    tester,
  ) async {
    final theme = AppTheme.createTheme(Brightness.dark);
    final trail = _FakeTrail(
      syncState: TrailSyncState.uploading,
      localId: 'local-2',
    );
    await tester.pumpWidget(
      _harness(SyncStatusChip(trail: trail), theme: theme),
    );
    await tester.pump();

    final spinner = tester.widget<CircularProgressIndicator>(
      find.byType(CircularProgressIndicator),
    );
    expect(
      spinner.color,
      syncStateColors(TrailSyncState.uploading, theme).icon,
    );
  });

  test('the chip source resolves state and colors through the shared '
      'helpers', () {
    final libDir = Directory('lib');
    expect(
      libDir.existsSync(),
      isTrue,
      reason: 'Run with the working directory set to "app/".',
    );
    final code = File('lib/components/trail/sync_status_chip.dart')
        .readAsStringSync()
        .split('\n')
        .where((l) => !RegExp(r'^\s*//').hasMatch(l))
        .join('\n');
    expect(code.contains('resolveSyncDisplayState('), isTrue);
    expect(code.contains('syncStateColors('), isTrue);
    expect(
      code.contains('Colors.red'),
      isFalse,
      reason: 'The failed color must come from syncStateColors only.',
    );
    expect(
      code.contains('AppColors.'),
      isFalse,
      reason: 'Sync-state colors live in sync_state_colors.dart only.',
    );
  });
}
