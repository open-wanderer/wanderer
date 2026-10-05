import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail_summary.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/provider/trail/trail_sync_provider.dart';
import 'package:wanderer/theme/sync_state_colors.dart';

/// The detail screen's flat sync-status block for an unsynced trail, with the
/// right action for each state.
///
/// State is resolved by [resolveSyncDisplayState] and colored by
/// [syncStateColors], the same pair `SyncStatusChip` uses, so the chip on a
/// list card and this section can never disagree. It watches the same
/// in-flight set as the chip, so a promoted draft reads pending -> uploading
/// and then disappears when the row is retired.
///
/// Renders nothing for a synced trail. Discard exists ONLY for a draft; the
/// button is gated here and `TrailSync.discardDraft` re-checks the row.
class SyncStatusSection extends ConsumerWidget {
  final TrailSummary trail;

  const SyncStatusSection({super.key, required this.trail});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final localId = trail.localId;
    final display = resolveSyncDisplayState(
      trail.syncState,
      localId: localId,
      inFlight: ref.watch(trailSyncProvider),
    );
    if (display == TrailSyncState.synced) return const SizedBox.shrink();

    final isOnline = ref.watch(onlineStatusProvider);
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final colors = syncStateColors(display, theme);

    final Widget leading;
    final String title;
    String? body;
    final actions = <Widget>[];

    switch (display) {
      case TrailSyncState.synced:
        return const SizedBox.shrink();
      case TrailSyncState.draft:
        leading = FaIcon(
          FontAwesomeIcons.penToSquare,
          size: 16,
          color: colors.icon,
        );
        title = l10n.sync_section_draft_title;
        body = l10n.sync_section_draft_body;
        if (localId != null) {
          actions.add(
            ElevatedButton(
              onPressed: () =>
                  ref.read(trailSyncProvider.notifier).uploadDraft(localId),
              child: Text(l10n.sync_section_upload),
            ),
          );
          actions.add(
            TextButton(
              onPressed: () => _confirmDiscard(context, ref, localId),
              child: Text(l10n.discard),
            ),
          );
        }
      case TrailSyncState.pending:
        leading = FaIcon(
          FontAwesomeIcons.cloudArrowUp,
          size: 16,
          color: colors.icon,
        );
        title = l10n.sync_pending;
        if (!isOnline) {
          body = l10n.sync_section_pending_offline_body;
        } else if (localId != null) {
          actions.add(
            ElevatedButton(
              // retry, not drainIfOnline: an explicit tap must also clear a
              // backoff window the pending row may still be sitting in.
              onPressed: () =>
                  ref.read(trailSyncProvider.notifier).retry(localId),
              child: Text(l10n.sync_section_upload_now),
            ),
          );
        }
      case TrailSyncState.uploading:
        leading = SizedBox(
          width: 16,
          height: 16,
          child: CircularProgressIndicator(strokeWidth: 2, color: colors.icon),
        );
        title = l10n.sync_uploading;
      case TrailSyncState.failed:
        leading = FaIcon(
          FontAwesomeIcons.triangleExclamation,
          size: 16,
          color: colors.icon,
        );
        title = l10n.sync_section_failed_title;
        if (localId != null) {
          actions.add(
            ElevatedButton(
              onPressed: () =>
                  ref.read(trailSyncProvider.notifier).retry(localId),
              child: Text(l10n.sync_section_retry),
            ),
          );
        }
    }

    // Flat by doctrine: a filled block, no shadow, elevation or border.
    return Container(
      key: const ValueKey('sync-status-section-block'),
      width: double.infinity,
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: colors.container,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              leading,
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      title,
                      style: theme.textTheme.titleSmall?.copyWith(
                        fontWeight: FontWeight.w600,
                        color: colors.onContainer,
                      ),
                    ),
                    if (body != null) ...[
                      const SizedBox(height: 4),
                      Text(
                        body,
                        style: theme.textTheme.bodyMedium?.copyWith(
                          color: colors.onContainer,
                        ),
                      ),
                    ],
                  ],
                ),
              ),
            ],
          ),
          if (actions.isNotEmpty) ...[
            const SizedBox(height: 12),
            Wrap(spacing: 8, runSpacing: 8, children: actions),
          ],
        ],
      ),
    );
  }

  /// Confirms, then discards a draft through the guarded
  /// `TrailSync.discardDraft` and pops the detail screen.
  Future<void> _confirmDiscard(
    BuildContext context,
    WidgetRef ref,
    String localId,
  ) async {
    // Resolved before the first await: `context` is a parameter, so a
    // post-await `mounted` check does not license reading from it.
    final l10n = AppLocalizations.of(context)!;

    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        content: Text(l10n.sync_discard_draft_confirm),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: Text(l10n.cancel),
          ),
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(true),
            child: Text(l10n.delete, style: const TextStyle(color: Colors.red)),
          ),
        ],
      ),
    );

    if (confirmed != true || !context.mounted) return;

    final deleted = await ref
        .read(trailSyncProvider.notifier)
        .discardDraft(localId);
    if (!context.mounted) return;

    if (deleted) {
      if (Navigator.of(context).canPop()) Navigator.of(context).pop();
      return;
    }
    ref
        .read(toastProvider.notifier)
        .add(
          ToastMessage(
            type: ToastType.error,
            icon: FontAwesomeIcons.xmark,
            text: l10n.error_deleting_trail,
          ),
        );
  }
}
