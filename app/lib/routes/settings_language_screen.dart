import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/base/wanderer_select.dart';
import 'package:wanderer/components/settings/settings_offline_banner.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/settings.dart';
import 'package:wanderer/provider/settings_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/actions/guard_online.dart';

/// Native names for each supported language. These are the single
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
    if (!guardOnline(ref, l10n)) return;
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
        title: Text(l10n.language_and_units(l10n.language, l10n.units)),
      ),
      body: ListView(
        children: [
          const SettingsOfflineBanner(),
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
            child: Text(
              l10n.language,
              style: Theme.of(context).textTheme.titleSmall?.copyWith(
                color: colorScheme.onSurfaceVariant,
              ),
            ),
          ),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: WandererSelect<Language>(
              name: 'language',
              icon: FontAwesomeIcons.language,
              // Until settings load there is nothing to copyWith, so the field
              // stays disabled rather than dropping the user's pick.
              disabled: settings == null,
              initialValue: settings?.language,
              items: [
                for (final language in Language.values)
                  SelectItem(value: language, label: _languageNames[language]!),
              ],
              onChanged: (value) {
                if (value == null || settings == null) return;
                _save(ref, l10n, settings.copyWith(language: value));
              },
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
          RadioGroup<String>(
            groupValue: settings?.unit ?? 'metric',
            onChanged: (value) {
              if (value == null || settings == null) return;
              _save(ref, l10n, settings.copyWith(unit: value));
            },
            child: Column(
              children: [
                RadioListTile<String>(
                  title: Text(l10n.metric),
                  value: 'metric',
                  activeColor: activeColor,
                ),
                RadioListTile<String>(
                  title: Text(l10n.imperial),
                  value: 'imperial',
                  activeColor: activeColor,
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
