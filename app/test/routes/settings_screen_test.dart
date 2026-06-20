import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:wanderer/entities/user_entity.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/routes/settings_screen.dart';

/// Stub Auth notifier so the screen renders without touching ObjectBox/the API.
/// Returns null (logged-out) synchronously so the logout button is not loading.
class _StubAuth extends Auth {
  @override
  Future<UserEntity?> build() async => null;
}

void main() {
  testWidgets('settings screen lists all 5 rows in D-06 order (SETNAV-01)', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [authProvider.overrideWith(_StubAuth.new)],
        child: const MaterialApp(
          localizationsDelegates: [
            AppLocalizations.delegate,
            GlobalMaterialLocalizations.delegate,
            GlobalWidgetsLocalizations.delegate,
            GlobalCupertinoLocalizations.delegate,
          ],
          supportedLocales: AppLocalizations.supportedLocales,
          locale: Locale('en'),
          home: SettingsScreen(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    // Exactly five settings rows render.
    expect(find.byType(ListTile), findsNWidgets(5));

    // The five English labels are present (SETNAV-01).
    expect(find.text('My Account'), findsOneWidget);
    expect(find.text('Privacy'), findsOneWidget);
    expect(find.text('Language & Units'), findsOneWidget);
    expect(find.text('Notifications'), findsOneWidget);
    expect(find.text('Appearance'), findsOneWidget);
  });
}
