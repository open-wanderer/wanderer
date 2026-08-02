import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/models/trail_summary.dart';
import 'package:wanderer/provider/trail/trail_sync_provider.dart';

/// The four-state sync-status indicator rendered below a trail's title on
/// both `TrailCard` and `TrailListItem` (REC-03, SYNC-02, SYNC-03).
///
/// Renders nothing at all when the trail is [TrailSyncState.synced] (D-08)
/// -- an always-present chip would make every ordinary trail noisier, and
/// it deliberately does not join the public/shared badge group (D-09):
/// "is this on the server yet" is a separate axis from visibility.
class SyncStatusChip extends ConsumerWidget {
  final TrailSummary trail;

  const SyncStatusChip({super.key, required this.trail});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    if (trail.syncState == TrailSyncState.synced) {
      return const SizedBox.shrink();
    }

    // The in-flight drain set is the authoritative in-the-moment signal --
    // a row still persisted as `pending` reads as Uploading the instant its
    // drain starts, before the row's own write lands. A restart mid-drain
    // (no active in-flight entry) still reads as Uploading from the
    // persisted state alone.
    final localId = trail.localId;
    final inFlight = ref.watch(trailSyncProvider);
    final isUploading =
        (localId != null && inFlight.contains(localId)) ||
        trail.syncState == TrailSyncState.uploading;

    if (isUploading) {
      return _Chip(
        leading: const SizedBox(
          width: 12,
          height: 12,
          child: CircularProgressIndicator(strokeWidth: 2),
        ),
        label: AppLocalizations.of(context)!.sync_uploading,
      );
    }

    if (trail.syncState == TrailSyncState.failed) {
      return _Chip(
        leading: const FaIcon(
          FontAwesomeIcons.triangleExclamation,
          size: 11,
          color: Colors.red,
        ),
        label: AppLocalizations.of(context)!.sync_failed,
        labelColor: Colors.red,
        onTap: localId == null
            ? null
            : () => ref.read(trailSyncProvider.notifier).retry(localId),
      );
    }

    // TrailSyncState.pending -- the only state left once synced/uploading/
    // failed are handled above.
    return _Chip(
      leading: FaIcon(
        FontAwesomeIcons.cloudArrowUp,
        size: 11,
        color: Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.6),
      ),
      label: AppLocalizations.of(context)!.sync_pending,
    );
  }
}

/// Shared visual shell for all three visible states, matching
/// `trail_card.dart`'s `_Chip` exactly: same padding, radius and background.
class _Chip extends StatelessWidget {
  final Widget leading;
  final String label;
  final Color? labelColor;
  final VoidCallback? onTap;

  const _Chip({
    required this.leading,
    required this.label,
    this.labelColor,
    this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final content = Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surfaceContainerHighest,
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
              color: labelColor ?? Theme.of(context).colorScheme.onSurface,
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
