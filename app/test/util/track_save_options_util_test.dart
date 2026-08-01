import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/util/track_save_options_util.dart';

/// Records every route push seen by the app's [Navigator] — used to assert
/// the offline branch shows nothing (no modal route pushed) beyond the
/// app's own initial route.
class _RecordingNavigatorObserver extends NavigatorObserver {
  int pushCount = 0;

  @override
  void didPush(Route<dynamic> route, Route<dynamic>? previousRoute) {
    pushCount++;
  }
}

class _FakeOnlineStatus extends OnlineStatus {
  _FakeOnlineStatus(this._initial);

  final bool _initial;

  @override
  bool build() => _initial;
}

/// Pumps a real [MaterialApp] (with the app's localization delegates, so
/// `AppLocalizations.of(context)!` resolves and `showTrackSaveOptionsSheet`'s
/// text renders) with [onlineStatusProvider] overridden to [online], and
/// hands back the captured [WidgetRef] whose `context` is attached to a live
/// [Navigator].
Future<WidgetRef> _pumpRef(
  WidgetTester tester, {
  required bool online,
  NavigatorObserver? observer,
}) async {
  late WidgetRef capturedRef;
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        onlineStatusProvider.overrideWith(() => _FakeOnlineStatus(online)),
      ],
      child: MaterialApp(
        navigatorObservers: [?observer],
        localizationsDelegates: const [
          AppLocalizations.delegate,
          GlobalMaterialLocalizations.delegate,
          GlobalWidgetsLocalizations.delegate,
          GlobalCupertinoLocalizations.delegate,
        ],
        supportedLocales: AppLocalizations.supportedLocales,
        home: Consumer(
          builder: (context, ref, _) {
            capturedRef = ref;
            return const SizedBox.shrink();
          },
        ),
      ),
    ),
  );
  return capturedRef;
}

void main() {
  group('resolveTrackSaveOptions', () {
    testWidgets(
      'offline: returns (false, false) immediately with no modal route '
      'pushed',
      (tester) async {
        final observer = _RecordingNavigatorObserver();
        final ref = await _pumpRef(tester, online: false, observer: observer);
        final baseline = observer.pushCount;

        final result = await resolveTrackSaveOptions(
          ref,
          ref.context,
          TrackSaveOptionsSource.recording,
        );

        expect(result, (false, false));
        // The app's own initial route is the only push observed — the sheet
        // never opened a second (modal) route.
        expect(observer.pushCount, baseline);
      },
    );

    testWidgets('online, confirmed with both toggles left untouched: returns '
        '(false, false)', (tester) async {
      final ref = await _pumpRef(tester, online: true);
      final l10n = AppLocalizations.of(ref.context)!;

      final future = resolveTrackSaveOptions(
        ref,
        ref.context,
        TrackSaveOptionsSource.recording,
      );
      await tester.pumpAndSettle();

      await tester.tap(find.text(l10n.save));
      await tester.pumpAndSettle();

      final result = await future;
      expect(result, (false, false));
    });

    testWidgets('online, confirmed with both toggles switched on: returns '
        '(true, true)', (tester) async {
      final ref = await _pumpRef(tester, online: true);
      final l10n = AppLocalizations.of(ref.context)!;

      final future = resolveTrackSaveOptions(
        ref,
        ref.context,
        TrackSaveOptionsSource.recording,
      );
      await tester.pumpAndSettle();

      // Tapping the SwitchListTile's title area toggles its switch (default
      // ListTile.onTap behaviour).
      await tester.tap(find.text(l10n.recalculate_heights));
      await tester.pumpAndSettle();
      await tester.tap(find.text(l10n.follow_roads));
      await tester.pumpAndSettle();
      await tester.tap(find.text(l10n.save));
      await tester.pumpAndSettle();

      final result = await future;
      expect(result, (true, true));
    });

    testWidgets('online, dismissed without confirming: returns null', (
      tester,
    ) async {
      final ref = await _pumpRef(tester, online: true);

      final future = resolveTrackSaveOptions(
        ref,
        ref.context,
        TrackSaveOptionsSource.recording,
      );
      await tester.pumpAndSettle();

      // The sheet is isDismissible:true — tapping the barrier above it
      // pops it with no result, same as a back-gesture.
      await tester.tapAt(const Offset(20, 20));
      await tester.pumpAndSettle();

      final result = await future;
      expect(result, isNull);
    });
  });
}
