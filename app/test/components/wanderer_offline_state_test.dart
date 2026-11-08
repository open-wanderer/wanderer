import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/components/base/wanderer_offline_state.dart';
import 'package:wanderer/i18n/app_localizations.dart';

Widget _wrap(Widget child) {
  return MaterialApp(
    localizationsDelegates: const [
      AppLocalizations.delegate,
      GlobalMaterialLocalizations.delegate,
      GlobalWidgetsLocalizations.delegate,
      GlobalCupertinoLocalizations.delegate,
    ],
    supportedLocales: AppLocalizations.supportedLocales,
    home: Scaffold(body: child),
  );
}

void main() {
  testWidgets('renders title and body text', (tester) async {
    await tester.pumpWidget(
      _wrap(
        WandererOfflineState(
          title: 'You\'re offline',
          body: 'Connect to the internet to load the map.',
          retryLabel: 'Try again',
          onRetry: () async {},
        ),
      ),
    );

    expect(find.text('You\'re offline'), findsOneWidget);
    expect(
      find.text('Connect to the internet to load the map.'),
      findsOneWidget,
    );
  });

  testWidgets('renders a retry button with the passed retryLabel', (
    tester,
  ) async {
    await tester.pumpWidget(
      _wrap(
        WandererOfflineState(
          title: 'title',
          body: 'body',
          retryLabel: 'Retry now',
          onRetry: () async {},
        ),
      ),
    );

    expect(find.widgetWithText(FilledButton, 'Retry now'), findsOneWidget);
  });

  testWidgets('tapping the retry button invokes onRetry exactly once', (
    tester,
  ) async {
    var callCount = 0;
    await tester.pumpWidget(
      _wrap(
        WandererOfflineState(
          title: 'title',
          body: 'body',
          retryLabel: 'Try again',
          onRetry: () async {
            callCount++;
          },
        ),
      ),
    );

    await tester.tap(find.text('Try again'));
    await tester.pumpAndSettle();

    expect(callCount, 1);
  });

  testWidgets(
    'while onRetry is unresolved, the button is disabled and a second tap '
    'does not fire a second callback',
    (tester) async {
      var callCount = 0;
      final completer = Completer<void>();
      await tester.pumpWidget(
        _wrap(
          WandererOfflineState(
            title: 'title',
            body: 'body',
            retryLabel: 'Try again',
            onRetry: () {
              callCount++;
              return completer.future;
            },
          ),
        ),
      );

      await tester.tap(find.text('Try again'));
      await tester.pump();

      final button = tester.widget<FilledButton>(find.byType(FilledButton));
      expect(button.onPressed, isNull);

      // A second tap while disabled must not reach the button's handler.
      await tester.tap(find.byType(FilledButton), warnIfMissed: false);
      await tester.pump();
      expect(callCount, 1);

      completer.complete();
      await tester.pumpAndSettle();
    },
  );

  testWidgets('never renders any exception or error text', (tester) async {
    await tester.pumpWidget(
      _wrap(
        WandererOfflineState(
          title: 'title',
          body: 'body',
          retryLabel: 'Try again',
          onRetry: () async {},
        ),
      ),
    );

    expect(find.textContaining('Exception'), findsNothing);
    expect(find.textContaining('DioException'), findsNothing);
  });
}
