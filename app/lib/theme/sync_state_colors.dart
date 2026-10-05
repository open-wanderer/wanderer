import 'package:flutter/material.dart';
import 'package:wanderer/models/trail_sync_state.dart';

/// The colors one sync-state surface needs: the filled [container], the text
/// and [onContainer] color drawn on it, and the leading [icon] color.
@immutable
class SyncStateColors {
  final Color container;
  final Color onContainer;
  final Color icon;

  const SyncStateColors({
    required this.container,
    required this.onContainer,
    required this.icon,
  });
}

/// The only source of sync-state colors, shared by `SyncStatusChip` and the
/// trail detail `SyncStatusSection` so the two always read the same.
///
/// Amber (draft) and red (failed) are a deliberate exception to DESIGN.md's
/// White Accent Rule: those states need the user's attention. Action buttons
/// never take these hues.
///
/// Draft is amber: it reads as "needs attention" without the alarm of the red
/// failed state. In dark mode the container is a translucent tint over the
/// surface. Pending and uploading stay neutral.
SyncStateColors syncStateColors(TrailSyncState state, ThemeData theme) {
  final scheme = theme.colorScheme;
  final isDark = theme.brightness == Brightness.dark;

  SyncStateColors neutral(Color icon) => SyncStateColors(
    container: scheme.surfaceContainerHighest,
    onContainer: scheme.onSurface,
    icon: icon,
  );

  return switch (state) {
    TrailSyncState.draft => () {
      final onContainer = isDark
          ? const Color(0xFFFFE082)
          : const Color(0xFF7A4F01);
      return SyncStateColors(
        container: isDark
            ? const Color(0xFFFFB300).withValues(alpha: 0.18)
            : const Color(0xFFFFECB3),
        onContainer: onContainer,
        icon: onContainer,
      );
    }(),
    TrailSyncState.failed => () {
      final onContainer = isDark
          ? const Color(0xFFFFB4AB)
          : const Color(0xFF93000A);
      return SyncStateColors(
        container: isDark
            ? const Color(0xFFE53935).withValues(alpha: 0.18)
            : const Color(0xFFFFDAD6),
        onContainer: onContainer,
        icon: onContainer,
      );
    }(),
    TrailSyncState.uploading => neutral(
      theme.progressIndicatorTheme.color ?? scheme.primary,
    ),
    TrailSyncState.pending ||
    TrailSyncState.synced => neutral(scheme.onSurface.withValues(alpha: 0.6)),
  };
}
