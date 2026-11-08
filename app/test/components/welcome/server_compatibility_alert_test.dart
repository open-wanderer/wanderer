import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/components/welcome/server_compatibility_alert.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/server_instance.dart';
import 'package:wanderer/provider/welcome/server_compatibility_provider.dart';
import 'package:wanderer/provider/welcome/server_selection_provider.dart';
import 'package:wanderer/util/server_compatibility.dart';

const _url = 'https://old.example';

/// Serves a selection without fetching wanderer.to/server/servers.json.
class _StubServerSelection extends ServerSelectionNotifier {
  @override
  Future<ServerState> build() async =>
      ServerState(const [], const ServerInstance(url: _url));
}

/// Shows the alert next to a probe of [selectedServerIncompatibleProvider],
/// the flag the welcome, login and register screens block their buttons on.
Widget _harness(ServerCompatibility compatibility) {
  return ProviderScope(
    overrides: [
      serverSelectionProvider.overrideWith(_StubServerSelection.new),
      serverCompatibilityProvider(
        _url,
      ).overrideWith((ref) async => compatibility),
    ],
    child: MaterialApp(
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
      ],
      supportedLocales: const [Locale('en')],
      home: Scaffold(
        body: Column(
          children: [
            const ServerCompatibilityAlert(),
            Consumer(
              builder: (context, ref, _) => Text(
                'blocked: ${ref.watch(selectedServerIncompatibleProvider)}',
              ),
            ),
          ],
        ),
      ),
    ),
  );
}

void main() {
  testWidgets('shows the alert and blocks sign-in on an incompatible server', (
    tester,
  ) async {
    await tester.pumpWidget(
      _harness(const ServerCompatibility(false, '0.20.0')),
    );
    await tester.pumpAndSettle();

    expect(find.text('Server not supported'), findsOneWidget);
    expect(find.text('Server version: 0.20.0'), findsOneWidget);
    expect(find.text('How to update the server'), findsOneWidget);
    expect(find.text('blocked: true'), findsOneWidget);
  });

  testWidgets('omits the version line when the server reports none', (
    tester,
  ) async {
    await tester.pumpWidget(_harness(const ServerCompatibility(false)));
    await tester.pumpAndSettle();

    expect(find.text('Server not supported'), findsOneWidget);
    expect(find.textContaining('Server version'), findsNothing);
  });

  testWidgets('stays hidden on a compatible server', (tester) async {
    await tester.pumpWidget(
      _harness(const ServerCompatibility(true, '0.21.0')),
    );
    await tester.pumpAndSettle();

    expect(find.text('Server not supported'), findsNothing);
    expect(find.text('blocked: false'), findsOneWidget);
  });

  testWidgets('stays hidden and does not block when the check is undecided', (
    tester,
  ) async {
    await tester.pumpWidget(_harness(const ServerCompatibility(null)));
    await tester.pumpAndSettle();

    expect(find.text('Server not supported'), findsNothing);
    expect(find.text('blocked: false'), findsOneWidget);
  });
}
