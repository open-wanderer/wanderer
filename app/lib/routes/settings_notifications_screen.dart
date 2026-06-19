import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/i18n/app_localizations.dart';

/// Stub Notifications screen. Phase 9 fills the body; for now it renders an
/// intentional themed Scaffold with a titled AppBar (not a blank screen).
class SettingsNotificationsScreen extends StatelessWidget {
  const SettingsNotificationsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => context.pop(),
        ),
        title: Text(AppLocalizations.of(context)!.notifications),
      ),
      body: const SizedBox.shrink(),
    );
  }
}
