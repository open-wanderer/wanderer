import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/models/trail_summary.dart';
import 'package:wanderer/provider/trail/trail_sync_provider.dart';
import 'package:wanderer/theme/sync_state_colors.dart';

/// The sync-status indicator rendered below a trail's title on both
/// `TrailCard` and `TrailListItem`.
///
/// Renders nothing at all when the trail is [TrailSyncState.synced]
/// -- an always-present chip would make every ordinary trail noisier, and
/// it deliberately does not join the public/shared badge group:
/// "is this on the server yet" is a separate axis from visibility.
///
/// A [TrailSyncState.draft] (a finished recording saved on this device) shows
/// a non-tappable "Draft" chip: it is not queued for upload until the user
/// uploads it from the detail screen or saves it in the edit form.
///
/// State is resolved by [resolveSyncDisplayState] and colored by
/// [syncStateColors], the same pair the detail screen's `SyncStatusSection`
/// uses, so the two surfaces cannot disagree.
class SyncStatusChip extends ConsumerWidget {
  final TrailSummary trail;

  const SyncStatusChip({super.key, required this.trail});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    if (trail.syncState == TrailSyncState.synced) {
      return const SizedBox.shrink();
    }

    final localId = trail.localId;
    final display = resolveSyncDisplayState(
      trail.syncState,
      localId: localId,
      inFlight: ref.watch(trailSyncProvider),
    );
    final colors = syncStateColors(display, Theme.of(context));
    final l10n = AppLocalizations.of(context)!;

    switch (display) {
      case TrailSyncState.synced:
        return const SizedBox.shrink();
      case TrailSyncState.draft:
        return _Chip(
          leading: FaIcon(
            FontAwesomeIcons.penToSquare,
            size: 11,
            color: colors.icon,
          ),
          label: l10n.sync_draft,
          labelColor: colors.onContainer,
          backgroundColor: colors.container,
        );
      case TrailSyncState.uploading:
        return _Chip(
          leading: SizedBox(
            width: 12,
            height: 12,
            child: CircularProgressIndicator(
              strokeWidth: 2,
              color: colors.icon,
            ),
          ),
          label: l10n.sync_uploading,
          labelColor: colors.onContainer,
          backgroundColor: colors.container,
        );
      case TrailSyncState.failed:
        return _Chip(
          leading: FaIcon(
            FontAwesomeIcons.triangleExclamation,
            size: 11,
            color: colors.icon,
          ),
          label: l10n.sync_failed,
          labelColor: colors.onContainer,
          backgroundColor: colors.container,
          onTap: localId == null
              ? null
              : () => ref.read(trailSyncProvider.notifier).retry(localId),
        );
      case TrailSyncState.pending:
        return _Chip(
          leading: FaIcon(
            FontAwesomeIcons.cloudArrowUp,
            size: 11,
            color: colors.icon,
          ),
          label: l10n.sync_pending,
          labelColor: colors.onContainer,
          backgroundColor: colors.container,
        );
    }
  }
}

/// Shared visual shell for all four visible states (draft, uploading, failed,
/// pending), matching `trail_card.dart`'s `_Chip` exactly: same padding and
/// radius. Colors always come from [syncStateColors].
class _Chip extends StatelessWidget {
  final Widget leading;
  final String label;
  final Color labelColor;
  final Color backgroundColor;
  final VoidCallback? onTap;

  const _Chip({
    required this.leading,
    required this.label,
    required this.labelColor,
    required this.backgroundColor,
    this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final content = Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: backgroundColor,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          leading,
          const SizedBox(width: 4),
          Text(
            label,
            style: TextStyle(
              fontSize: 11,
              fontWeight: FontWeight.normal,
              color: labelColor,
            ),
          ),
        ],
      ),
    );

    if (onTap == null) return content;

    return InkWell(
      borderRadius: BorderRadius.circular(8),
      onTap: onTap,
      child: content,
    );
  }
}
