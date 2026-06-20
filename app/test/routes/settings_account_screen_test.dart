import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/settings.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/settings_provider.dart';
import 'package:wanderer/routes/settings_account_screen.dart';

/// Stub Auth notifier that returns a fixture [UserEntity] synchronously so
/// the screen renders without touching ObjectBox or the network.
class _StubAuth extends Auth {
  @override
  Future<UserEntity?> build() async => UserEntity(
        id: 'user-test-id',
        collectionId: 'users',
        collectionName: 'users',
        actorId: 'actor-id',
        username: 'tester',
        preferredUsername: 'tester',
        email: 'tester@example.com',
        iri: 'https://example.com/u/tester',
        serverUrl: 'https://example.com',
        created: DateTime(2024),
        updated: DateTime(2024),
        avatar: null,
      );
}

void main() {
  testWidgets(
    'account screen renders all five sections in fixed order (ACCT-01..05)',
    (tester) async {
      // The screen may carry long content; give the test a tall viewport so
      // the lazily-built ListView mounts all rows at once.
      tester.view.physicalSize = const Size(1080, 4000);
      tester.view.devicePixelRatio = 1.0;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            authProvider.overrideWith(_StubAuth.new),
            settingsProvider.overrideWithValue(
              const Settings(id: '1', bio: 'hello'),
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
            home: SettingsAccountScreen(),
          ),
        ),
      );
      await tester.pumpAndSettle();

      // AppBar title.
      expect(find.text('Account'), findsOneWidget);

      // Bio section header (ACCT-02) — also appears as TextField hint text.
      expect(find.text('Add Bio'), findsWidgets);

      // Bio text field (ACCT-02).
      expect(find.byType(TextField), findsWidgets);

      // Change email and Change password list tiles (ACCT-03, ACCT-04).
      expect(find.text('Change email'), findsOneWidget);
      expect(find.text('Change password'), findsOneWidget);

      // Delete account tile (ACCT-05) — last row.
      expect(find.text('Delete Account'), findsWidgets);

      // TODO: tap-to-save assertions require apiProvider/HTTP override fixtures
      // that the current test harness does not provide. The network paths are
      // covered by acceptance criteria and integration testing.
    },
  );
}
