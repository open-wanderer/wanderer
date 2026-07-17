import 'package:collection/collection.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/provider/route_anchor_provider.dart';
import 'package:wanderer/provider/trail/category_provider.dart';
import 'package:wanderer/util/category_icon_util.dart';
import 'package:wanderer/util/gpx_util.dart';
import 'package:wanderer/util/route_travel_bucket.dart';

/// Resolves the leading icon for [bucket]'s picker card: prefers a matching
/// operator [Category] icon (via [trailCategoryIcon]), falling back to
/// [RouteTravelBucket.fallbackIcon] when no category matches. Mirrors
/// `travel_profile_sheet.dart`'s identical resolution (Task 5) — icon
/// resolution only, never the costing payload (RESEARCH.md Q2).
Widget bucketIcon(
  RouteTravelBucket bucket,
  List<Category> categories, {
  double size = 20,
}) {
  Category? matched;
  if (bucket == RouteTravelBucket.hiking) {
    final id = categoryForTravelProfile('pedestrian', categories);
    matched = id == null
        ? null
        : categories.firstWhereOrNull((c) => c.id == id);
  } else {
    matched = categoryForBikeBucket(bucket.keywords, categories);
  }

  return matched != null
      ? trailCategoryIcon(matched, size: size)
      : FaIcon(bucket.fallbackIcon, size: size);
}

/// The route planner sheet's "Settings" tab (quick-260717-t7q): hosts the
/// relocated auto-routing toggle plus the unified 5-option travel-profile
/// picker that replaces the web's separate activity + bike-type pickers.
///
/// Reads/writes [routeAnchorsProvider] directly — no constructor params, and
/// (CRITICAL, RESEARCH.md Q4 / flutter#55388) never accepts the sheet's
/// shared `scrollController`: `TabBarView` keeps every page mounted
/// simultaneously, so only [RouteAnchorListTab] may hold that controller.
/// This tab manages its own scroll via a plain [SingleChildScrollView].
class SettingsTab extends ConsumerWidget {
  const SettingsTab({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final state = ref.watch(routeAnchorsProvider);
    final categories = ref.watch(categoryProvider).value ?? const [];
    final notifier = ref.read(routeAnchorsProvider.notifier);
    final selectedBucket = bucketForState(
      state.travelProfile,
      state.costingOptions,
    );
    final theme = Theme.of(context);

    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SwitchListTile(
            contentPadding: EdgeInsets.zero,
            title: const Text('Auto-routing'),
            subtitle: const Text(
              'Automatically follow roads and paths between anchors.',
            ),
            value: state.autoRoutingEnabled,
            onChanged: (_) => notifier.toggleAutoRouting(),
          ),
          const SizedBox(height: 8),
          Text(
            'Travel profile',
            style: theme.textTheme.titleSmall?.copyWith(
              fontWeight: FontWeight.bold,
            ),
          ),
          const SizedBox(height: 8),
          for (final bucket in RouteTravelBucket.values) ...[
            _BucketCard(
              bucket: bucket,
              icon: bucketIcon(bucket, categories),
              selected: bucket == selectedBucket,
              // No-op on the already-active bucket — switchProfile always
              // clears undo/redo and re-dispatches every segment (a fresh
              // baseline, by design), so re-tapping the current selection
              // must not pay that cost for nothing.
              onTap: bucket == selectedBucket
                  ? null
                  : () => notifier.switchProfile(
                      bucket.costing,
                      bucket.costingOptions,
                    ),
            ),
            const SizedBox(height: 8),
          ],
        ],
      ),
    );
  }
}

/// A tappable travel-bucket card, visually matching
/// `travel_profile_sheet.dart`'s `_TravelProfileCard` (16px card radius,
/// `secondaryContainer` @ 40% alpha icon badge, `primary` icon tint) with an
/// added accent border/tint when [selected] is the current bucket.
class _BucketCard extends StatelessWidget {
  final RouteTravelBucket bucket;
  final Widget icon;
  final bool selected;
  final VoidCallback? onTap;

  const _BucketCard({
    required this.bucket,
    required this.icon,
    required this.selected,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(
          color: selected ? theme.colorScheme.primary : theme.colorScheme.outline,
          width: selected ? 2 : 1,
        ),
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
                      bucket.label,
                      style: theme.textTheme.titleMedium?.copyWith(
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                    const SizedBox(height: 4),
                    Text(
                      bucket.description,
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
              ),
              if (selected)
                Icon(Icons.check_circle, color: theme.colorScheme.primary),
            ],
          ),
        ),
      ),
    );
  }
}
