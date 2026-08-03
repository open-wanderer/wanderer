import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/settings/settings_offline_banner.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/settings.dart';
import 'package:wanderer/provider/settings_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/actions/guard_online.dart';

class SettingsPrivacyScreen extends ConsumerWidget {
  const SettingsPrivacyScreen({super.key});

  /// Persists [updated] to the server. `saveToServer` has no internal error
  /// handling, so the DioException is caught here and surfaced as a toast. On
  /// success the watched provider updates the selection optimistically.
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

  /// Builds a section header with the canonical 16/16/16/8 padding and the
  /// single explicit type override on this screen (titleSmall + onSurfaceVariant).
  Widget _sectionHeader(BuildContext context, String label) {
    final colorScheme = Theme.of(context).colorScheme;
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
      child: Text(
        label,
        style: Theme.of(
          context,
        ).textTheme.titleSmall?.copyWith(color: colorScheme.onSurfaceVariant),
      ),
    );
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
        title: Text(l10n.privacy),
      ),
      body: ListView(
        children: [
          const SettingsOfflineBanner(),
          _sectionHeader(context, l10n.account_privacy),
          RadioGroup<String>(
            groupValue: settings?.privacy?.account ?? "public",
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
              final updated =
                  (settings.privacy ??
                          const SettingsPrivacy(
                            account: 'public',
                            trails: 'private',
                            lists: 'private',
                          ))
                      .copyWith(account: value);
              _save(ref, l10n, settings.copyWith(privacy: updated));
            },
            child: Column(
              children: [
                RadioListTile<String>(
                  title: Text(l10n.public),
                  subtitle: Text(
                    l10n.settings_privacy_account_public,
                    style: Theme.of(
                      context,
                    ).textTheme.bodyMedium!.copyWith(fontSize: 12),
                  ),
                  value: "public",
                  activeColor: activeColor,
                ),
                RadioListTile<String>(
                  title: Text(l10n.private),
                  subtitle: Text(
                    l10n.settings_privacy_account_private,
                    style: Theme.of(
                      context,
                    ).textTheme.bodyMedium!.copyWith(fontSize: 12),
                  ),
                  value: "private",
                  activeColor: activeColor,
                ),
              ],
            ),
          ),
          _sectionHeader(context, l10n.trail(2)),
          RadioGroup<String>(
            groupValue: settings?.privacy?.trails ?? "private",
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
              final updated =
                  (settings.privacy ??
                          const SettingsPrivacy(
                            account: 'public',
                            trails: 'private',
                            lists: 'private',
                          ))
                      .copyWith(trails: value);
              _save(ref, l10n, settings.copyWith(privacy: updated));
            },
            child: Column(
              children: [
                RadioListTile<String>(
                  title: Text(l10n.public),
                  subtitle: Text(
                    l10n.settings_privacy_trails_public,
                    style: Theme.of(
                      context,
                    ).textTheme.bodyMedium!.copyWith(fontSize: 12),
                  ),
                  value: "public",
                  activeColor: activeColor,
                ),
                RadioListTile<String>(
                  title: Text(l10n.only_me),
                  subtitle: Text(
                    l10n.settings_privacy_trails_private,
                    style: Theme.of(
                      context,
                    ).textTheme.bodyMedium!.copyWith(fontSize: 12),
                  ),
                  value: "private",
                  activeColor: activeColor,
                ),
              ],
            ),
          ),
          _sectionHeader(context, l10n.list(2)),
          RadioGroup<String>(
            groupValue: settings?.privacy?.lists ?? "private",
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
              final updated =
                  (settings.privacy ??
                          const SettingsPrivacy(
                            account: 'public',
                            trails: 'private',
                            lists: 'private',
                          ))
                      .copyWith(lists: value);
              _save(ref, l10n, settings.copyWith(privacy: updated));
            },
            child: Column(
              children: [
                RadioListTile<String>(
                  title: Text(l10n.public),
                  subtitle: Text(
                    l10n.settings_privacy_lists_public,
                    style: Theme.of(
                      context,
                    ).textTheme.bodyMedium!.copyWith(fontSize: 12),
                  ),
                  value: "public",
                  activeColor: activeColor,
                ),
                RadioListTile<String>(
                  title: Text(l10n.only_me),
                  subtitle: Text(
                    l10n.settings_privacy_lists_private,
                    style: Theme.of(
                      context,
                    ).textTheme.bodyMedium!.copyWith(fontSize: 12),
                  ),
                  value: "private",
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
