import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/settings.dart';
import 'package:wanderer/provider/settings_provider.dart';
import 'package:wanderer/routes/settings_language_screen.dart';

void main() {
  testWidgets(
    'language screen renders 14 locale tiles + units switch (LANG-01)',
    (tester) async {
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            settingsProvider.overrideWithValue(
              const Settings(id: '1', language: Language.en, unit: 'metric'),
            ),
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
            home: SettingsLanguageScreen(),
          ),
        ),
      );
      await tester.pumpAndSettle();

      // One RadioListTile per Language enum value (14).
      expect(
        find.byType(RadioListTile<Language>),
        findsNWidgets(Language.values.length),
      );

      // Native-name labels are rendered (the single hardcoded-string exception).
      expect(find.text('English'), findsOneWidget);
      expect(find.text('中文'), findsOneWidget);

      // The units switch sits below the 14 lazily-built radio tiles; scroll it
      // into view before asserting. It is OFF for metric (imperial = on).
      await tester.scrollUntilVisible(find.byType(SwitchListTile), 300);
      await tester.pumpAndSettle();
      final switchTile = tester.widget<SwitchListTile>(
        find.byType(SwitchListTile),
      );
      expect(switchTile.value, isFalse);

      // TODO: tap-to-save assertion requires an apiProvider/HTTP override
      // fixture that the current test harness does not provide. The save path
      // is covered by the provider-level unit tests and acceptance criteria.
    },
  );
}
