import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/provider/pending_trail_import_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/routes/trail_source_select_screen.dart';

class _FakeOnlineStatus extends OnlineStatus {
  @override
  bool build() => false;

  @override
  Future<bool> refresh() async => false;
}

/// An unsupported extension makes `importTrailFile` stop at its extension
/// check with an error toast, which is enough to prove the screen ran it
/// without parsing or uploading anything.
const _inboundPath = '/cache/inbound/notes.txt';

Future<ProviderContainer> _pump(
  WidgetTester tester, {
  String? pendingBeforeMount,
}) async {
  final container = ProviderContainer(
    overrides: [onlineStatusProvider.overrideWith(_FakeOnlineStatus.new)],
  );
  addTearDown(container.dispose);
  if (pendingBeforeMount != null) {
    container.read(pendingTrailImportProvider.notifier).set(pendingBeforeMount);
  }

  final router = GoRouter(
    initialLocation: '/trail/create',
    routes: [
      GoRoute(
        path: '/trail/create',
        builder: (context, state) => const TrailSourceSelectScreen(),
      ),
      GoRoute(
        path: '/other',
        builder: (context, state) => const Scaffold(body: Text('other')),
      ),
    ],
  );
  addTearDown(router.dispose);

  await tester.pumpWidget(
    UncontrolledProviderScope(
      container: container,
      child: MaterialApp.router(
        routerConfig: router,
        localizationsDelegates: AppLocalizations.localizationsDelegates,
        supportedLocales: AppLocalizations.supportedLocales,
      ),
    ),
  );
  await tester.pumpAndSettle();
  return container;
}

/// Lets the error toast's 4s auto-dismiss timer fire before teardown.
Future<void> _expireToasts(WidgetTester tester) =>
    tester.pump(const Duration(seconds: 5));

void main() {
  testWidgets('imports a file that was pending before the screen opened', (
    tester,
  ) async {
    final container = await _pump(tester, pendingBeforeMount: _inboundPath);

    expect(container.read(pendingTrailImportProvider), isNull);
    expect(container.read(toastProvider), hasLength(1));
    await _expireToasts(tester);
  });

  testWidgets('imports a file that arrives while the screen is open', (
    tester,
  ) async {
    final container = await _pump(tester);
    expect(container.read(toastProvider), isEmpty);

    container.read(pendingTrailImportProvider.notifier).set(_inboundPath);
    await tester.pumpAndSettle();

    expect(container.read(pendingTrailImportProvider), isNull);
    expect(container.read(toastProvider), hasLength(1));
    await _expireToasts(tester);
  });

  testWidgets('leaves the file for the visible screen when covered', (
    tester,
  ) async {
    final container = await _pump(tester);
    GoRouter.of(
      tester.element(find.byType(TrailSourceSelectScreen)),
    ).push('/other');
    await tester.pumpAndSettle();

    container.read(pendingTrailImportProvider.notifier).set(_inboundPath);
    await tester.pumpAndSettle();

    expect(container.read(pendingTrailImportProvider), _inboundPath);
    expect(container.read(toastProvider), isEmpty);
  });
}
