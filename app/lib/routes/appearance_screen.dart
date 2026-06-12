import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/theme_provider.dart';

class AppearanceScreen extends ConsumerWidget {
  const AppearanceScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context)!;
    final currentMode = ref.watch(themeModeProvider);

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => context.pop(),
        ),
        title: Text(l10n.appearance),
      ),
      body: RadioGroup<ThemeMode>(
        groupValue: currentMode,
        onChanged: (value) {
          if (value != null) {
            ref.read(themeModeProvider.notifier).setMode(value);
          }
        },
        child: ListView(
          children: [
            RadioListTile<ThemeMode>(
              title: Text(l10n.theme_light),
              value: ThemeMode.light,
            ),
            RadioListTile<ThemeMode>(
              title: Text(l10n.theme_dark),
              value: ThemeMode.dark,
            ),
            RadioListTile<ThemeMode>(
              title: Text(l10n.theme_system),
              value: ThemeMode.system,
            ),
          ],
        ),
      ),
    );
  }
}
