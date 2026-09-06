import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/server_instance.dart';
import 'package:wanderer/provider/welcome/server_selection_provider.dart';
import 'package:wanderer/routes/server_selection_screen.dart';

/// Serves a selection without touching the network — the real notifier fetches
/// wanderer.to/server/servers.json in `build()`. An empty instance list keeps
/// the list body free of `CachedNetworkImageProvider` rows.
class _StubServerSelection extends ServerSelectionNotifier {
  _StubServerSelection(this.selected, this.available);

  final ServerInstance? selected;
  final List<ServerInstance> available;

  @override
  Future<ServerState> build() async => ServerState(available, selected);
}

Widget _harness(
  ServerInstance? selected, {
  List<ServerInstance> available = const [],
}) {
  return ProviderScope(
    overrides: [
      serverSelectionProvider.overrideWith(
        () => _StubServerSelection(selected, available),
      ),
    ],
    child: const MaterialApp(
      localizationsDelegates: [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
      ],
      supportedLocales: [Locale('en')],
      home: ServerSelectionScreen(),
    ),
  );
}

void main() {
  testWidgets(
    'prefills the field with the instance already selected, port included',
    (tester) async {
      await tester.pumpWidget(
        _harness(const ServerInstance(url: 'https://example.com:8443')),
      );
      await tester.pumpAndSettle();

      expect(find.text('https://example.com:8443'), findsOneWidget);
    },
  );

  testWidgets('leaves the field empty when no instance is selected', (
    tester,
  ) async {
    await tester.pumpWidget(_harness(null));
    await tester.pumpAndSettle();

    final field = tester.widget<TextField>(find.byType(TextField));
    expect(field.controller!.text, isEmpty);
  });

  testWidgets('the prefill seeds the list filter as well as the field', (
    tester,
  ) async {
    await tester.pumpWidget(
      _harness(
        const ServerInstance(url: 'https://self.hosted.example'),
        available: const [
          ServerInstance(name: 'Wanderer', url: 'https://wanderer.to'),
        ],
      ),
    );
    await tester.pumpAndSettle();

    // The "no servers match" branch is gated on a non-empty search query, so
    // reaching it proves the prefilled URL was applied as the filter — and it
    // is the branch offering the custom-URL button.
    expect(find.textContaining('No servers match'), findsOneWidget);
  });
}
