import 'package:flutter/material.dart';
import 'package:wanderer/i18n/app_localizations.dart';

/// Presents the "Save recording" options bottom sheet: two independent
/// toggles ("Recalculate heights", "Follow roads"), both off by default,
/// gating `_saveRecordedTrack`'s opt-in Valhalla cleanup pipeline.
///
/// Styled after `travel_profile_sheet.dart`'s flat, bordered "Terrain Log"
/// card aesthetic, but — unlike that tap-to-close picker — holds local
/// toggle state until the user explicitly confirms.
///
/// Returns `(recalcHeights, followRoads)` when confirmed, or `null` when
/// dismissed/backed out (cancel aborts the save with no change to the
/// session).
Future<(bool recalcHeights, bool followRoads)?> showTrackSaveOptionsSheet(
  BuildContext context,
) {
  return showModalBottomSheet<(bool, bool)>(
    context: context,
    isDismissible: true,
    enableDrag: true,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
    ),
    builder: (context) => const _TrackSaveOptionsSheetContent(),
  );
}

class _TrackSaveOptionsSheetContent extends StatefulWidget {
  const _TrackSaveOptionsSheetContent();

  @override
  State<_TrackSaveOptionsSheetContent> createState() =>
      _TrackSaveOptionsSheetContentState();
}

class _TrackSaveOptionsSheetContentState
    extends State<_TrackSaveOptionsSheetContent> {
  bool _recalcHeights = false;
  bool _followRoads = false;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;

    return SingleChildScrollView(
      padding: EdgeInsets.fromLTRB(
        16,
        16,
        16,
        24 + MediaQuery.of(context).viewInsets.bottom,
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 8.0),
            child: Center(
              child: Container(
                width: 30,
                height: 5,
                decoration: BoxDecoration(
                  borderRadius: const BorderRadius.all(Radius.circular(24)),
                  color: theme.colorScheme.secondaryContainer,
                ),
              ),
            ),
          ),
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 8.0),
            child: Text(
              l10n.save_recording_options,
              style: theme.textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.bold,
              ),
            ),
          ),
          const SizedBox(height: 8),
          _ToggleCard(
            title: l10n.recalculate_heights,
            subtitle: l10n.recalculate_heights_description,
            value: _recalcHeights,
            onChanged: (v) => setState(() => _recalcHeights = v),
          ),
          const SizedBox(height: 8),
          _ToggleCard(
            title: l10n.follow_roads,
            subtitle: l10n.follow_roads_description,
            value: _followRoads,
            onChanged: (v) => setState(() => _followRoads = v),
          ),
          const SizedBox(height: 16),
          FilledButton(
            onPressed: () => Navigator.pop(
              context,
              (_recalcHeights, _followRoads),
            ),
            child: Text(l10n.save),
          ),
        ],
      ),
    );
  }
}

/// A bordered, flat (`elevation: 0`) card hosting a single [SwitchListTile],
/// matching this codebase's "Terrain Log" doctrine (borders only, no
/// shadows) already used by `_TravelProfileCard` and `SettingsTab`'s
/// `_BucketCard`.
class _ToggleCard extends StatelessWidget {
  final String title;
  final String subtitle;
  final bool value;
  final ValueChanged<bool> onChanged;

  const _ToggleCard({
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
