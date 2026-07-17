import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/provider/trail/category_provider.dart';
import 'package:wanderer/util/route_travel_bucket.dart';

/// Presents the D-01/D-02 travel-profile entry-point bottom sheet, expanded
/// (quick-260717-t7q) from a 2-card Hike/Bike chooser to the unified
/// 5-option picker (Hiking, Biking/Hybrid, Biking/Mountain, Biking/Cross,
/// Biking/Road) shared with the in-planner Settings tab.
///
/// Returns the tapped [RouteTravelBucket], or `null` if the sheet is
/// dismissed (back button / tap outside) without a selection. The sheet is
/// always dismissible (D-02) — there is no forced choice and no
/// "Continue"/"Cancel" button; each card both selects the bucket and closes
/// the sheet in one tap.
Future<RouteTravelBucket?> showTravelProfileSheet(BuildContext context) {
  return showModalBottomSheet<RouteTravelBucket>(
    context: context,
    isDismissible: true,
    enableDrag: true,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
    ),
    builder: (context) {
      return Consumer(
        builder: (context, ref, _) {
          final theme = Theme.of(context);
          final categories = ref.watch(categoryProvider).value ?? const [];

          // 5 cards (up from the original 2) can exceed the available sheet
          // height on smaller screens — SingleChildScrollView keeps the
          // content scrollable instead of overflowing (quick-260717-t7q).
          return SingleChildScrollView(
            padding: const EdgeInsets.fromLTRB(
              16,
              16,
              16,
              24 + kBottomNavigationBarHeight + 56,
            ),
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
                        borderRadius: const BorderRadius.all(
                          Radius.circular(24),
                        ),
                        color: theme.colorScheme.secondaryContainer,
                      ),
                    ),
                  ),
                ),
                for (final bucket in RouteTravelBucket.values) ...[
                  _TravelProfileCard(
                    icon: bucketIcon(bucket, categories),
                    title: bucket.label,
                    description: bucket.description,
                    onTap: () => Navigator.pop(context, bucket),
                  ),
                  const SizedBox(height: 8),
                ],
              ],
            ),
          );
        },
      );
    },
  );
}

/// A tappable travel-bucket card, visually matching
/// [TrailSourceSelectScreen]'s `_SourceActionCard` family (16px card radius,
/// 12px icon-badge radius, `secondaryContainer` @ 40% alpha badge
/// background, `theme.colorScheme.primary` icon tint). Kept self-contained
/// here rather than importing the private widget from that screen. Accepts a
/// pre-resolved [icon] widget (rather than a raw `FaIconData`) so it can host
/// either a resolved category icon or a hardcoded fallback [FaIcon].
class _TravelProfileCard extends StatelessWidget {
  final Widget icon;
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
                child: icon,
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
