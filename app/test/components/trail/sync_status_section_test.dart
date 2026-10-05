import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/components/trail/sync_status_chip.dart';
import 'package:wanderer/components/trail/sync_status_section.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/record.dart';
import 'package:wanderer/models/trail_summary.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
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

/// Stub `TrailSync` notifier: caller-supplied in-flight set, recorded calls,
/// no ObjectBox/API.
class _StubTrailSync extends TrailSync {
  _StubTrailSync(this._initial, {this.discardResult = true});

  final Set<String> _initial;
  final bool discardResult;
  final List<String> retried = [];
  final List<String> uploaded = [];
  final List<String> discarded = [];
  int drains = 0;

  @override
  Set<String> build() => _initial;

  @override
  Future<void> retry(String localId) async {
    retried.add(localId);
  }

  @override
  Future<void> drainIfOnline() async {
    drains++;
  }

  @override
  Future<bool> uploadDraft(String localId) async {
    uploaded.add(localId);
    return true;
  }

  @override
  Future<bool> discardDraft(String localId) async {
    discarded.add(localId);
    return discardResult;
  }
}

class _StubOnline extends OnlineStatus {
  _StubOnline(this.value);
  final bool value;

  @override
  bool build() => value;
}

/// Records toasts without starting the 4s auto-dismiss Timer.
class _RecordingToast extends Toast {
  final List<ToastMessage> added = [];

  @override
  List<ToastMessage> build() => [];

  @override
  void add(ToastMessage toast) {
    added.add(toast);
  }
}

class _Rig {
  _Rig({
    Set<String> inFlight = const {},
    bool online = true,
    bool discardResult = true,
  }) : sync = _StubTrailSync(inFlight, discardResult: discardResult),
       toast = _RecordingToast(),
       _online = online;

  final _StubTrailSync sync;
  final _RecordingToast toast;
  final bool _online;

  Widget harness(Widget child, {ThemeData? theme, Widget? home}) {
    return ProviderScope(
      overrides: [
        trailSyncProvider.overrideWith(() => sync),
        onlineStatusProvider.overrideWith(() => _StubOnline(_online)),
        toastProvider.overrideWith(() => toast),
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
        home: home ?? Scaffold(body: child),
      ),
    );
  }
}

/// Hosts [child] on a route pushed from a home screen, so a pop is visible.
Widget _pushedHost(Widget child) {
  return Builder(
    builder: (context) => Scaffold(
      body: Center(
        child: Column(
          children: [
            const Text('home'),
            TextButton(
              onPressed: () => Navigator.of(context).push(
                MaterialPageRoute<void>(builder: (_) => Scaffold(body: child)),
              ),
              child: const Text('open'),
            ),
          ],
        ),
      ),
    ),
  );
}

Finder _buttons(Type type) => find.descendant(
  of: find.byType(SyncStatusSection),
  matching: find.byWidgetPredicate((w) => w.runtimeType == type),
);

Finder _elevated(String label) => find.widgetWithText(ElevatedButton, label);
Finder _text(String label) => find.widgetWithText(TextButton, label);

void main() {
  group('draft', () {
    final trail = _FakeTrail(
      syncState: TrailSyncState.draft,
      localId: 'local-5',
    );

    testWidgets('shows the copy, an Upload ElevatedButton and a Discard '
        'TextButton; Upload calls uploadDraft once', (tester) async {
      final rig = _Rig();
      await tester.pumpWidget(rig.harness(SyncStatusSection(trail: trail)));
      await tester.pumpAndSettle();

      expect(find.text('Not uploaded yet'), findsOneWidget);
      expect(
        find.text('This recording is only on this device.'),
        findsOneWidget,
      );
      expect(_elevated('Upload'), findsOneWidget);
      expect(_text('Discard'), findsOneWidget);

      await tester.tap(_elevated('Upload'));
      await tester.pumpAndSettle();

      expect(rig.sync.uploaded, ['local-5']);
      expect(rig.sync.discarded, isEmpty);
    });

    testWidgets('Discard then Cancel closes the dialog and discards nothing', (
      tester,
    ) async {
      final rig = _Rig();
      await tester.pumpWidget(rig.harness(SyncStatusSection(trail: trail)));
      await tester.pumpAndSettle();

      await tester.tap(_text('Discard'));
      await tester.pumpAndSettle();

      expect(
        find.text("Delete this recording? It hasn't been uploaded anywhere."),
        findsOneWidget,
      );
      expect(find.text('Cancel'), findsOneWidget);
      expect(find.text('Delete'), findsOneWidget);

      await tester.tap(find.text('Cancel'));
      await tester.pumpAndSettle();

      expect(find.byType(AlertDialog), findsNothing);
      expect(rig.sync.discarded, isEmpty);
    });

    testWidgets('Discard then Delete calls discardDraft and pops the screen', (
      tester,
    ) async {
      final rig = _Rig(discardResult: true);
      await tester.pumpWidget(
        rig.harness(
          const SizedBox(),
          home: _pushedHost(SyncStatusSection(trail: trail)),
        ),
      );
      await tester.tap(find.text('open'));
      await tester.pumpAndSettle();
      expect(find.byType(SyncStatusSection), findsOneWidget);

      await tester.tap(_text('Discard'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Delete'));
      await tester.pumpAndSettle();

      expect(rig.sync.discarded, ['local-5']);
      expect(find.byType(SyncStatusSection), findsNothing);
      expect(find.text('home'), findsOneWidget);
      expect(rig.toast.added, isEmpty);
    });

    testWidgets('a refused discard stays on the screen and toasts the error', (
      tester,
    ) async {
      final rig = _Rig(discardResult: false);
      await tester.pumpWidget(
        rig.harness(
          const SizedBox(),
          home: _pushedHost(SyncStatusSection(trail: trail)),
        ),
      );
      await tester.tap(find.text('open'));
      await tester.pumpAndSettle();

      await tester.tap(_text('Discard'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Delete'));
      await tester.pumpAndSettle();

      expect(rig.sync.discarded, ['local-5']);
      expect(find.byType(SyncStatusSection), findsOneWidget);
      expect(rig.toast.added, hasLength(1));
      expect(rig.toast.added.single.type, ToastType.error);
      expect(rig.toast.added.single.text, 'Error deleting trail');
    });
  });

  group('pending', () {
    final trail = _FakeTrail(
      syncState: TrailSyncState.pending,
      localId: 'local-1',
    );

    testWidgets('online: Waiting to upload plus Upload now, which retries', (
      tester,
    ) async {
      final rig = _Rig();
      await tester.pumpWidget(rig.harness(SyncStatusSection(trail: trail)));
      await tester.pumpAndSettle();

      expect(find.text('Waiting to upload'), findsOneWidget);
      expect(
        find.text("Uploads when you're back online"),
        findsNothing,
        reason: 'The body text is the offline variant only.',
      );
      expect(_text('Discard'), findsNothing);
      expect(find.text('Discard'), findsNothing);

      await tester.tap(_elevated('Upload now'));
      await tester.pumpAndSettle();

      expect(rig.sync.retried, ['local-1']);
      expect(
        rig.sync.drains,
        0,
        reason: 'Upload now goes through retry, which clears any backoff.',
      );
    });

    testWidgets('offline: explains itself and offers no button at all', (
      tester,
    ) async {
      final rig = _Rig(online: false);
      await tester.pumpWidget(rig.harness(SyncStatusSection(trail: trail)));
      await tester.pumpAndSettle();

      expect(find.text('Waiting to upload'), findsOneWidget);
      expect(find.text("Uploads when you're back online"), findsOneWidget);
      expect(_buttons(ElevatedButton), findsNothing);
      expect(_buttons(TextButton), findsNothing);
      expect(find.text('Discard'), findsNothing);
    });
  });

  group('uploading', () {
    for (final entry in {
      'persisted uploading': (
        _FakeTrail(syncState: TrailSyncState.uploading, localId: 'local-2'),
        <String>{},
      ),
      'pending but in flight': (
        _FakeTrail(syncState: TrailSyncState.pending, localId: 'local-4'),
        {'local-4'},
      ),
    }.entries) {
      testWidgets('${entry.key}: Uploading… with a spinner and no buttons', (
        tester,
      ) async {
        final rig = _Rig(inFlight: entry.value.$2);
        await tester.pumpWidget(
          rig.harness(SyncStatusSection(trail: entry.value.$1)),
        );
        await tester.pump();

        expect(find.text('Uploading…'), findsOneWidget);
        expect(find.byType(CircularProgressIndicator), findsOneWidget);
        expect(_buttons(ElevatedButton), findsNothing);
        expect(_buttons(TextButton), findsNothing);
        expect(find.text('Discard'), findsNothing);
      });
    }
  });

  group('failed', () {
    testWidgets('Upload failed with a Retry button that retries', (
      tester,
    ) async {
      final trail = _FakeTrail(
        syncState: TrailSyncState.failed,
        localId: 'local-3',
      );
      final rig = _Rig();
      await tester.pumpWidget(rig.harness(SyncStatusSection(trail: trail)));
      await tester.pumpAndSettle();

      expect(find.text('Upload failed'), findsOneWidget);
      expect(_text('Discard'), findsNothing);
      expect(find.text('Discard'), findsNothing);

      await tester.tap(_elevated('Retry'));
      await tester.pumpAndSettle();

      expect(rig.sync.retried, ['local-3']);
    });
  });

  testWidgets('a synced trail renders nothing', (tester) async {
    final rig = _Rig();
    await tester.pumpWidget(
      rig.harness(
        SyncStatusSection(trail: _FakeTrail(syncState: TrailSyncState.synced)),
      ),
    );
    await tester.pumpAndSettle();

    expect(
      find.byKey(const ValueKey('sync-status-section-block')),
      findsNothing,
    );
  });

  group('chip/section color parity', () {
    for (final brightness in Brightness.values) {
      for (final state in [
        TrailSyncState.draft,
        TrailSyncState.pending,
        TrailSyncState.uploading,
        TrailSyncState.failed,
      ]) {
        testWidgets('${state.name} (${brightness.name})', (tester) async {
          final theme = AppTheme.createTheme(brightness);
          final trail = _FakeTrail(syncState: state, localId: 'local-9');
          final rig = _Rig();
          await tester.pumpWidget(
            rig.harness(
              Column(
                children: [
                  SyncStatusChip(trail: trail),
                  SyncStatusSection(trail: trail),
                ],
              ),
              theme: theme,
            ),
          );
          await tester.pump();

          final expected = syncStateColors(state, theme);

          final chipContainer = tester.widget<Container>(
            find
                .descendant(
                  of: find.byType(SyncStatusChip),
                  matching: find.byType(Container),
                )
                .first,
          );
          expect(
            (chipContainer.decoration as BoxDecoration).color,
            expected.container,
          );

          final block = tester.widget<Container>(
            find.byKey(const ValueKey('sync-status-section-block')),
          );
          expect(block.color, expected.container);
          // Full-bleed band: square corners, no decoration at all.
          expect(block.decoration, isNull);

          final chipLabel = tester.widget<Text>(
            find.descendant(
              of: find.byType(SyncStatusChip),
              matching: find.byType(Text),
            ),
          );
          final sectionTitle = tester.widget<Text>(
            find
                .descendant(
                  of: find.byKey(const ValueKey('sync-status-section-block')),
                  matching: find.byType(Text),
                )
                .first,
          );
          expect(chipLabel.style?.color, expected.onContainer);
          expect(sectionTitle.style?.color, expected.onContainer);
        });
      }
    }
  });

  group('layout stability', () {
    for (final textScale in [1.0, 1.5]) {
      testWidgets('every state renders at the same height (scale $textScale)', (
        tester,
      ) async {
        tester.platformDispatcher.textScaleFactorTestValue = textScale;
        addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);
        final heights = <String, double>{};
        for (final (state, online) in [
          (TrailSyncState.draft, true),
          (TrailSyncState.pending, true),
          (TrailSyncState.pending, false),
          (TrailSyncState.uploading, true),
          (TrailSyncState.failed, true),
        ]) {
          final rig = _Rig(online: online);
          await tester.pumpWidget(
            rig.harness(
              SyncStatusSection(
                trail: _FakeTrail(syncState: state, localId: 'local-h'),
              ),
            ),
          );
          await tester.pump();
          heights['${state.name}/${online ? 'online' : 'offline'}'] = tester
              .getSize(find.byKey(const ValueKey('sync-status-section-block')))
              .height;
        }
        expect(
          heights.values.toSet(),
          hasLength(1),
          reason: 'heights per state: $heights',
        );
      });
    }
  });
}
