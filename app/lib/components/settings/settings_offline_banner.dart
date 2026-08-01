import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/online_status_provider.dart';

/// Single shared offline banner shown at the top of every network-dependent
/// settings sub-screen. A one-line insert per screen, restructuring no
/// control tree — the "single banner" option CONTEXT's Discretion section
/// explicitly permits for keeping Settings browsable-but-read-only offline.
///
/// Distinct from `WandererOfflineState` (a full-area takeover with a retry
/// CTA for screens that have nothing to show): this is an inline strip above
/// content that stays fully browsable.
class SettingsOfflineBanner extends ConsumerWidget {
  const SettingsOfflineBanner({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final isOnline = ref.watch(onlineStatusProvider);
    if (isOnline) return const SizedBox.shrink();

    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;

    return Container(
      width: double.infinity,
      color: theme.colorScheme.surfaceContainerHighest,
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      child: Row(
        children: [
          FaIcon(
            FontAwesomeIcons.linkSlash,
            size: 14,
            color: theme.colorScheme.onSurfaceVariant,
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Text(
              l10n.offline_settings_banner,
              style: theme.textTheme.bodySmall,
            ),
          ),
        ],
      ),
    );
  }
}
