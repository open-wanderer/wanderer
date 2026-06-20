import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/settings.dart';
import 'package:wanderer/provider/settings_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';

/// Native names for each supported language (D-09). These are the single
/// approved hardcoded-string exception — a language's own name must read in
/// its own script regardless of the active UI locale, so they are NOT ARB keys.
const Map<Language, String> _languageNames = {
  Language.cs: 'Čeština',
  Language.en: 'English',
  Language.de: 'Deutsch',
  Language.es: 'Español',
  Language.eu: 'Euskara',
  Language.fr: 'Français',
  Language.hu: 'Magyar',
  Language.it: 'Italiano',
  Language.nl: 'Nederlands',
  Language.no: 'Norsk',
  Language.pl: 'Polski',
  Language.pt: 'Português',
  Language.ru: 'Русский',
  Language.zh: '中文',
};

class SettingsLanguageScreen extends ConsumerWidget {
  const SettingsLanguageScreen({super.key});

  /// Persists [updated] to the server. `saveToServer` has no internal error
  /// handling (Pitfall 2), so the DioException is caught here and surfaced as a
  /// toast. On success the watched provider updates the selection optimistically.
  Future<void> _save(
    WidgetRef ref,
    AppLocalizations l10n,
    Settings updated,
  ) async {
    try {
      await ref.read(settingsProvider.notifier).saveToServer(updated);
    } catch (_) {
      ref
          .read(toastProvider.notifier)
          .add(
            ToastMessage(
              type: ToastType.error,
              icon: FontAwesomeIcons.circleExclamation,
              text: l10n.error_saving_settings,
            ),
          );
    }
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context)!;
    final settings = ref.watch(settingsProvider);
    final colorScheme = Theme.of(context).colorScheme;
    final activeColor = Theme.of(context).brightness == Brightness.dark
        ? colorScheme.onSurface
        : colorScheme.primary;

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => context.pop(),
        ),
        title: Text(l10n.language),
      ),
      body: ListView(
        children: [
          RadioGroup<Language>(
            groupValue: settings?.language,
            onChanged: (value) {
              if (value == null) return;
              if (settings == null) {
                ref
                    .read(toastProvider.notifier)
                    .add(
                      ToastMessage(
                        type: ToastType.error,
                        icon: FontAwesomeIcons.circleExclamation,
                        text: l10n.error_saving_settings,
                      ),
                    );
                return;
              }
              _save(ref, l10n, settings.copyWith(language: value));
            },
            child: Column(
              children: [
                for (final language in Language.values)
                  RadioListTile<Language>(
                    title: Text(_languageNames[language]!),
                    value: language,
                    activeColor: activeColor,
                  ),
              ],
            ),
          ),
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
            child: Text(
              l10n.units,
              style: Theme.of(context).textTheme.titleSmall?.copyWith(
                color: colorScheme.onSurfaceVariant,
              ),
            ),
          ),
          SwitchListTile(
            title: Text(l10n.imperial),
            value: settings?.unit == 'imperial',
            onChanged: (isImperial) {
              if (settings != null) {
                _save(
                  ref,
                  l10n,
                  settings.copyWith(unit: isImperial ? 'imperial' : 'metric'),
                );
              }
            },
          ),
        ],
      ),
    );
  }
}
