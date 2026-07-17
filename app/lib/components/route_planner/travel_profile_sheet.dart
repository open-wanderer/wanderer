import 'package:flutter/material.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/i18n/app_localizations.dart';

/// Presents the D-01/D-02 hike/bike entry-point bottom sheet.
///
/// Returns `'pedestrian'` if the Hike card is tapped, `'bicycle'` if the Bike
/// card is tapped, or `null` if the sheet is dismissed (back button / tap
/// outside) without a selection. The sheet is always dismissible (D-02) —
/// there is no forced choice and no "Continue"/"Cancel" button; each card
/// both selects the profile and closes the sheet in one tap.
Future<String?> showTravelProfileSheet(BuildContext context) {
  final l10n = AppLocalizations.of(context)!;

  return showModalBottomSheet<String>(
    context: context,
    isDismissible: true,
    enableDrag: true,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
    ),
    builder: (context) {
      final theme = Theme.of(context);

      return Padding(
        padding: const EdgeInsets.fromLTRB(16, 16, 16, 24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
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
            _TravelProfileCard(
              icon: FontAwesomeIcons.personHiking,
              title: l10n.travel_profile_hike,
              description: l10n.travel_profile_hike_description,
              onTap: () => Navigator.pop(context, 'pedestrian'),
            ),
            const SizedBox(height: 8),
            _TravelProfileCard(
              icon: FontAwesomeIcons.bicycle,
              title: l10n.travel_profile_bike,
              description: l10n.travel_profile_bike_description,
              onTap: () => Navigator.pop(context, 'bicycle'),
            ),
          ],
        ),
      );
    },
  );
}

/// A tappable hike/bike card, visually matching
/// [TrailSourceSelectScreen]'s `_SourceActionCard` family (16px card radius,
/// 12px icon-badge radius, `secondaryContainer` @ 40% alpha badge
/// background, `theme.colorScheme.primary` icon tint). Kept self-contained
/// here rather than importing the private widget from that screen.
class _TravelProfileCard extends StatelessWidget {
  final FaIconData icon;
  final String title;
  final String description;
  final VoidCallback onTap;

  const _TravelProfileCard({
    required this.icon,
    required this.title,
    required this.description,
    required this.onTap,
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
      child: InkWell(
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.all(16.0),
          child: Row(
            children: [
              Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: theme.colorScheme.secondaryContainer.withValues(
                    alpha: 0.4,
                  ),
                  borderRadius: BorderRadius.circular(12),
                ),
                child: FaIcon(icon, color: theme.colorScheme.primary),
              ),
              const SizedBox(width: 16),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      title,
                      style: theme.textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      description,
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
