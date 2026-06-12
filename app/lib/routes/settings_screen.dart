import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/i18n/app_localizations.dart';

class SettingsScreen extends StatelessWidget {
  const SettingsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final l10n = AppLocalizations.of(context)!;

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => context.pop(),
        ),
        title: Text(l10n.settings),
      ),
      body: ListView(
        children: [
          ListTile(
            leading: const Icon(Icons.palette_outlined),
            title: Text(l10n.appearance),
            trailing: const Icon(Icons.chevron_right),
            onTap: () => context.push('/settings/appearance'),
          ),
        ],
      ),
    );
  }
}
