import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/provider/trail/category_provider.dart';
import 'package:wanderer/provider/trail/subcategory_provider.dart';
import 'package:wanderer/util/route_travel_bucket.dart';

/// Presents the travel-profile entry-point bottom sheet: a 5-option picker
/// (Hiking, Biking/Hybrid, Biking/Mountain, Biking/Cross, Biking/Road)
/// shared with the in-planner Settings tab.
///
/// Returns the tapped [RouteTravelBucket], or `null` if the sheet is
/// dismissed without a selection. Always dismissible — no forced choice and
/// no "Continue"/"Cancel" button; each card selects and closes in one tap.
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
          final categories = ref.watch(categoryProvider).value ?? const [];

          // 5 cards can exceed the sheet height on smaller screens, so make
          // it scrollable instead of overflowing.
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
                for (final bucket in RouteTravelBucket.values) ...[
                  _TravelProfileCard(
                    icon: bucketIcon(
                      bucket,
                      categories,
                      ref.watch(subcategoryProvider),
                    ),
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

/// A tappable travel-bucket card, styled to match
/// [TrailSourceSelectScreen]'s `_SourceActionCard` family. Kept
/// self-contained rather than importing that private widget. Accepts a
/// pre-resolved [icon] widget so it can host either a category icon or a
/// hardcoded fallback.
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
                        color: theme.colorScheme.onSurface.withValues(
                          alpha: 0.6,
                        ),
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
