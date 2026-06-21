import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/settings.dart';
import 'package:wanderer/provider/settings_provider.dart';
import 'package:wanderer/routes/settings_notifications_screen.dart';

void main() {
  testWidgets(
    'notifications screen renders nine sections with web+email toggles (NOTIF-01..09)',
    (tester) async {
      // 18 tiles + 9 headers need a tall viewport so the lazily-built ListView
      // mounts all rows at once.
      tester.view.physicalSize = const Size(1080, 8000);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            settingsProvider.overrideWithValue(const Settings(id: '1')),
          ],
          child: const MaterialApp(
            localizationsDelegates: [
              AppLocalizations.delegate,
              GlobalMaterialLocalizations.delegate,
              GlobalWidgetsLocalizations.delegate,
              GlobalCupertinoLocalizations.delegate,
            ],
            supportedLocales: AppLocalizations.supportedLocales,
            locale: Locale('en'),
            home: SettingsNotificationsScreen(),
          ),
        ),
      );
      await tester.pumpAndSettle();

      // One Web + one Email switch per type × nine types.
      expect(find.byType(SwitchListTile), findsNWidgets(18));
      expect(find.text('Web'), findsNWidgets(9));
      expect(find.text('Email'), findsNWidgets(9));

      // At least one section header description renders.
      expect(
        find.text('Someone left a comment on your trail'),
        findsOneWidget,
      );

      // TODO: tap-to-save assertion requires an apiProvider/HTTP override
      // fixture that the current test harness does not provide. The save path
      // is covered by the acceptance criteria and manual verification.
    },
  );

  testWidgets('null notifications defaults every toggle to ON (D-07)', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(1080, 8000);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          settingsProvider.overrideWithValue(const Settings(id: '1')),
        ],
        child: const MaterialApp(
          localizationsDelegates: [
            AppLocalizations.delegate,
            GlobalMaterialLocalizations.delegate,
            GlobalWidgetsLocalizations.delegate,
            GlobalCupertinoLocalizations.delegate,
          ],
          supportedLocales: AppLocalizations.supportedLocales,
          locale: Locale('en'),
          home: SettingsNotificationsScreen(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    final tiles = tester
        .widgetList<SwitchListTile>(find.byType(SwitchListTile))
        .toList();
    expect(tiles, hasLength(18));
    for (final tile in tiles) {
      expect(tile.value, isTrue);
    }
  });
}
