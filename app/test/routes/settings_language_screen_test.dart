import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/components/base/wanderer_select.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/settings.dart';
import 'package:wanderer/provider/settings_provider.dart';
import 'package:wanderer/routes/settings_language_screen.dart';

void main() {
  testWidgets(
    'language screen renders a 14-locale select + units switch',
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

      // One select option per Language enum value (14), preselected from settings.
      final select = tester.widget<WandererSelect<Language>>(
        find.byType(WandererSelect<Language>),
      );
      expect(select.items.length, Language.values.length);
      expect(select.initialValue, Language.en);

      // Native-name labels are used (the single hardcoded-string exception).
      expect(
        select.items.map((item) => item.label),
        containsAll(<String>['English', '中文']),
      );

      // The closed dropdown shows the active language.
      expect(find.text('English'), findsWidgets);

      // The units section uses RadioListTile<String> (metric/imperial), not SwitchListTile.
      // Scroll the metric tile into view and assert both unit tiles render.
      await tester.scrollUntilVisible(find.text('Metric'), 300);
      await tester.pumpAndSettle();
      expect(find.byType(RadioListTile<String>), findsNWidgets(2));
      expect(find.text('Metric'), findsOneWidget);
      expect(find.text('Imperial'), findsOneWidget);

      // TODO: tap-to-save assertion requires an apiProvider/HTTP override
      // fixture that the current test harness does not provide. The save path
      // is covered by the provider-level unit tests and acceptance criteria.
    },
  );
}
