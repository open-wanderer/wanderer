import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:url_launcher/url_launcher.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/welcome/server_compatibility_provider.dart';
import 'package:wanderer/provider/welcome/server_selection_provider.dart';

final Uri _updateDocsUrl = Uri.parse(
  'https://wanderer.to/run/backend-configuration/mobile-app',
);

/// Persistent alert below the server selector, shown while the selected
/// server is known to be incompatible with the app. Sits on every screen that
/// lets the server be changed, so it stays in view for as long as the
/// sign-in buttons are blocked; renders nothing otherwise.
class ServerCompatibilityAlert extends ConsumerWidget {
  const ServerCompatibilityAlert({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final url = ref.watch(serverSelectionProvider).value?.selectedServer?.url;
    if (url == null) return const SizedBox.shrink();

    final compatibility = ref.watch(serverCompatibilityProvider(url)).value;
    if (compatibility == null || !compatibility.isIncompatible) {
      return const SizedBox.shrink();
    }

    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final version = compatibility.serverVersion;

    return Semantics(
      liveRegion: true,
      child: Container(
        width: double.infinity,
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: theme.colorScheme.errorContainer,
          border: Border.all(color: theme.colorScheme.error),
          borderRadius: BorderRadius.circular(12),
        ),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Padding(
              padding: const EdgeInsets.only(top: 2),
              child: FaIcon(
                FontAwesomeIcons.triangleExclamation,
                size: 16,
                color: theme.colorScheme.onErrorContainer,
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                spacing: 4,
                children: [
                  Text(
                    l10n.server_incompatible_title,
                    style: theme.textTheme.titleSmall?.copyWith(
                      color: theme.colorScheme.onErrorContainer,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                  Text(
                    l10n.server_incompatible,
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.onErrorContainer,
                    ),
                  ),
                  if (version != null)
                    Text(
                      l10n.server_incompatible_version(version),
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.onErrorContainer,
                      ),
                    ),
                  TextButton(
                    style: TextButton.styleFrom(
                      padding: EdgeInsets.zero,
                      minimumSize: const Size(0, 36),
                      tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                      foregroundColor: theme.colorScheme.onErrorContainer,
                      textStyle: theme.textTheme.bodySmall?.copyWith(
                        fontWeight: FontWeight.bold,
                        decoration: TextDecoration.underline,
                      ),
                    ),
                    onPressed: () => launchUrl(
                      _updateDocsUrl,
                      mode: LaunchMode.externalApplication,
                    ),
                    child: Text(l10n.server_incompatible_docs),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
