import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/actions/guard_online.dart';

/// Pumps a throwaway `Consumer` so `guardOnline` (which needs a `WidgetRef`)
/// can be exercised from a real widget tree, with the app's localization
/// delegates in place so `AppLocalizations.of(context)!` resolves.
Future<WidgetRef> _pumpRef(WidgetTester tester, {required bool online}) async {
  late WidgetRef capturedRef;
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        onlineStatusProvider.overrideWith(() => _FakeOnlineStatus(online)),
      ],
      child: MaterialApp(
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

class _FakeOnlineStatus extends OnlineStatus {
  final bool _initial;
  _FakeOnlineStatus(this._initial);

  @override
  bool build() => _initial;
}

void main() {
  testWidgets('returns true and queues no toast when online', (tester) async {
    final ref = await _pumpRef(tester, online: true);
    final l10n = AppLocalizations.of(
      // ignore: use_build_context_synchronously
      ref.context,
    )!;

    final result = guardOnline(ref, l10n);

    expect(result, isTrue);
    expect(ref.read(toastProvider), isEmpty);
  });

  testWidgets('returns false when offline', (tester) async {
    final ref = await _pumpRef(tester, online: false);
    final l10n = AppLocalizations.of(ref.context)!;

    final result = guardOnline(ref, l10n);

    expect(result, isFalse);

    // Flush the toast's 4s auto-dismiss timer so it doesn't leak past this
    // test (flutter_test asserts no pending timers at teardown).
    await tester.pump(const Duration(seconds: 5));
  });

  testWidgets(
    'when offline, adds exactly one ToastMessage carrying the offline copy',
    (tester) async {
      final ref = await _pumpRef(tester, online: false);
      final l10n = AppLocalizations.of(ref.context)!;

      guardOnline(ref, l10n);

      final toasts = ref.read(toastProvider);
      expect(toasts, hasLength(1));
      expect(toasts.first.text, l10n.offline_action_unavailable);
      expect(toasts.first.text, isNot(l10n.error_saving_settings));

      await tester.pump(const Duration(seconds: 5));
    },
  );

  testWidgets(
    'performs no network call — returns instantly (synchronous contract)',
    (tester) async {
      final ref = await _pumpRef(tester, online: true);
      final l10n = AppLocalizations.of(ref.context)!;

      // guardOnline is declared to return bool, not Future<bool> — this is
      // itself the synchronous-contract assertion (a network probe could
      // not satisfy this signature).
      final bool result = guardOnline(ref, l10n);
      expect(result, isTrue);
    },
  );

  testWidgets('two guardOnline calls while offline queue two distinct toasts', (
    tester,
  ) async {
    final ref = await _pumpRef(tester, online: false);
    final l10n = AppLocalizations.of(ref.context)!;

    guardOnline(ref, l10n);
    guardOnline(ref, l10n);

    final toasts = ref.read(toastProvider);
    expect(toasts, hasLength(2));
    expect(toasts[0].id, isNot(toasts[1].id));

    await tester.pump(const Duration(seconds: 5));
  });
}
