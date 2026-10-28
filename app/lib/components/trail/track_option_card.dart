import 'package:flutter/material.dart';

/// A bordered, flat (`elevation: 0`) card hosting a single [SwitchListTile],
/// used for the trail editor's track transform switches,
/// matching this codebase's "Terrain Log" doctrine (borders only, no
/// shadows) already used by `_TravelProfileCard` and `SettingsTab`'s
/// `_BucketCard`.
class TrackOptionCard extends StatelessWidget {
  final String title;
  final String subtitle;
  final bool value;

  /// Null disables the switch.
  final ValueChanged<bool>? onChanged;

  const TrackOptionCard({
    super.key,
    required this.title,
    required this.subtitle,
    required this.value,
    required this.onChanged,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(color: theme.colorScheme.outline),
      ),
      clipBehavior: Clip.antiAlias,
      child: SwitchListTile(
        title: Text(
          title,
          style: theme.textTheme.titleMedium?.copyWith(
            fontWeight: FontWeight.bold,
          ),
        ),
        subtitle: Text(
          subtitle,
          style: theme.textTheme.bodySmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
        value: value,
        onChanged: onChanged,
      ),
    );
  }
}
