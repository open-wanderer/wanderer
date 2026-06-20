import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/settings.dart';
import 'package:wanderer/provider/settings_provider.dart';
import 'package:wanderer/routes/settings_privacy_screen.dart';

void main() {
  testWidgets(
    'privacy screen renders three visibility sections (PRIV-01, PRIV-02, PRIV-03)',
    (tester) async {
      // The six tiles carry long wrapping subtitles; give the test a tall
      // viewport so the lazily-built ListView mounts all six at once.
      tester.view.physicalSize = const Size(1080, 4000);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            settingsProvider.overrideWithValue(
              const Settings(
                id: '1',
                privacy: SettingsPrivacy(
                  account: 'public',
                  trails: 'private',
                  lists: 'private',
                ),
              ),
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
            home: SettingsPrivacyScreen(),
          ),
        ),
      );
      await tester.pumpAndSettle();

      // Two string radios per section × three sections.
      expect(find.byType(RadioListTile<String>), findsNWidgets(6));

      // Three section headers in fixed order: Account → Trails → Lists.
      expect(find.text('Account privacy'), findsOneWidget);
      expect(find.text('Trails'), findsOneWidget);
      expect(find.text('Lists'), findsOneWidget);

      // Labels: Public appears in all three sections; Only me in trails + lists;
      // Private only in the account section (D-03).
      expect(find.text('Public'), findsWidgets);
      expect(find.text('Only me'), findsNWidgets(2));
      expect(find.text('Private'), findsOneWidget);

      // TODO: tap-to-save assertion requires an apiProvider/HTTP override
      // fixture that the current test harness does not provide. The save path
      // is covered by the provider-level unit tests and acceptance criteria.
    },
  );

  testWidgets('null privacy applies web-parity defaults (D-04)', (tester) async {
    tester.view.physicalSize = const Size(1080, 4000);
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
          home: SettingsPrivacyScreen(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    // With privacy null the screen renders without throwing and all six tiles
    // are present. The groupValue defaults (account=public, trails=private,
    // lists=private) are independently covered by Task 1 source assertions.
    expect(find.byType(RadioListTile<String>), findsNWidgets(6));

    // The three RadioGroups expose the defaulted groupValues directly.
    final groups = tester
        .widgetList<RadioGroup<String>>(find.byType(RadioGroup<String>))
        .toList();
    expect(groups, hasLength(3));
    expect(groups[0].groupValue, 'public'); // account
    expect(groups[1].groupValue, 'private'); // trails
    expect(groups[2].groupValue, 'private'); // lists
  });
}
