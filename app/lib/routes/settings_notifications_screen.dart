import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:wanderer/components/settings/settings_offline_banner.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/settings.dart';
import 'package:wanderer/provider/settings_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/util/offline_guard_util.dart';

class SettingsNotificationsScreen extends ConsumerWidget {
  const SettingsNotificationsScreen({super.key});

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

  /// Builds the updated notifications map touching only [key]'s changed field
  /// (D-09), then persists. When [settings] is null we cannot build a payload,
  /// so surface the same generic error toast the Privacy screen uses (D-08).
  void _onToggle(
    WidgetRef ref,
    AppLocalizations l10n,
    Settings? settings,
    String key, {
    required bool isWeb,
    required bool value,
  }) {
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
    final existing =
        settings.notifications?[key] ??
        const NotificationPreference(web: true, email: true);
    final pref = isWeb
        ? existing.copyWith(web: value)
        : existing.copyWith(email: value);
    final updated = Map<String, NotificationPreference>.from(
      settings.notifications ?? {},
    )..[key] = pref;
    _save(ref, l10n, settings.copyWith(notifications: updated));
  }

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final l10n = AppLocalizations.of(context)!;
    final settings = ref.watch(settingsProvider);
    final colorScheme = Theme.of(context).colorScheme;
    final activeColor = Theme.of(context).brightness == Brightness.dark
        ? colorScheme.onSurface
        : colorScheme.primary;

    // D-05 order: snake_case @JsonValue keys paired with their human-readable
    // settings_notification_* description. The enum's camelCase identifier is
    // never used as a key — it would never match the server's snake_case keys.
    final types = <({String key, String label})>[
      (key: 'trail_comment', label: l10n.settings_notification_trail_comment),
      (key: 'new_follower', label: l10n.settings_notification_new_follower),
      (key: 'trail_share', label: l10n.settings_notification_trail_share),
      (key: 'trail_like', label: l10n.settings_notification_trail_like),
      (key: 'list_share', label: l10n.settings_notification_list_share),
      (
        key: 'summit_log_create',
        label: l10n.settings_notification_summit_log_create,
      ),
      (key: 'trail_mention', label: l10n.settings_notification_trail_mention),
      (
        key: 'comment_mention',
        label: l10n.settings_notification_comment_mention,
      ),
      (
        key: 'summit_log_mention',
        label: l10n.settings_notification_summit_log_mention,
      ),
    ];

    final children = <Widget>[const SettingsOfflineBanner()];
    for (final type in types) {
      children.add(_sectionHeader(context, type.label));
      children.add(
        SwitchListTile(
          title: Text(l10n.web),
          value: settings?.notifications?[type.key]?.web ?? true,
          activeThumbColor: activeColor,
          onChanged: (value) => _onToggle(
            ref,
            l10n,
            settings,
            type.key,
            isWeb: true,
            value: value,
          ),
        ),
      );
      children.add(
        SwitchListTile(
          title: Text(l10n.email),
          value: settings?.notifications?[type.key]?.email ?? true,
          activeThumbColor: activeColor,
          onChanged: (value) => _onToggle(
            ref,
            l10n,
            settings,
            type.key,
            isWeb: false,
            value: value,
          ),
        ),
      );
    }

    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => context.pop(),
        ),
        title: Text(l10n.notifications),
      ),
      body: ListView(children: children),
    );
  }
}
