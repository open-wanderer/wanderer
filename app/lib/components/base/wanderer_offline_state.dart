import 'package:flutter/material.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';

/// Shared offline empty state with a retry CTA.
///
/// Modelled on `settings_offline_regions_screen.dart`'s `_buildEmptyState`,
/// with a retry affordance added. Takes only plain strings — it has no
/// parameter capable of carrying an exception, which is what keeps raw
/// `DioException` text (including the full request URL) off screen (see
/// `WandererError`, which renders `err.toString()` directly).
class WandererOfflineState extends StatefulWidget {
  final String title;
  final String body;
  final String retryLabel;
  final Future<void> Function() onRetry;

  /// Defaults to `FontAwesomeIcons.linkSlash` — the disconnected-wifi glyph
  /// is Pro-only and absent from this dependency's free icon set.
  final FaIconData icon;

  const WandererOfflineState({
    super.key,
    required this.title,
    required this.body,
    required this.retryLabel,
    required this.onRetry,
    this.icon = FontAwesomeIcons.linkSlash,
  });

  @override
  State<WandererOfflineState> createState() => _WandererOfflineStateState();
}

class _WandererOfflineStateState extends State<WandererOfflineState> {
  bool _inFlight = false;

  Future<void> _handleRetry() async {
    if (_inFlight) return;
    setState(() => _inFlight = true);
    try {
      await widget.onRetry();
    } finally {
      if (mounted) setState(() => _inFlight = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Center(
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            FaIcon(
              widget.icon,
              size: 32,
              color: theme.colorScheme.onSurface.withValues(alpha: 0.5),
            ),
            const SizedBox(height: 16),
            Text(
              widget.title,
              style: theme.textTheme.bodyLarge,
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 8),
            Text(
              widget.body,
              style: theme.textTheme.bodySmall,
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 16),
            FilledButton(
              onPressed: _inFlight ? null : _handleRetry,
              child: _inFlight
                  ? const SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : Text(widget.retryLabel),
            ),
          ],
        ),
      ),
    );
  }
}
