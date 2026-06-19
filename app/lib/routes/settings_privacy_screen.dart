import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/i18n/app_localizations.dart';

/// Stub Privacy screen. Phase 7 fills the body; for now it renders an
/// intentional themed Scaffold with a titled AppBar (not a blank screen).
class SettingsPrivacyScreen extends StatelessWidget {
  const SettingsPrivacyScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => context.pop(),
        ),
        title: Text(AppLocalizations.of(context)!.privacy),
      ),
      body: const SizedBox.shrink(),
    );
  }
}
