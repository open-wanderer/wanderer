import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/route_anchor_provider.dart';
import 'package:wanderer/provider/trail/category_provider.dart';
import 'package:wanderer/provider/trail/subcategory_provider.dart';
import 'package:wanderer/models/route_travel_bucket.dart';

/// The route planner sheet's "Settings" tab: hosts the auto-routing toggle
/// plus the unified 5-option travel-profile picker.
///
/// Reads/writes [routeAnchorsProvider] directly — no constructor params.
/// Never accepts the sheet's shared `scrollController` (flutter#55388):
/// `TabBarView` keeps every page mounted simultaneously, so only
/// [RouteAnchorListTab] may hold that controller. This tab manages its own
/// scroll via a plain [SingleChildScrollView].
class SettingsTab extends ConsumerWidget {
  final ScrollController? scrollController;

  const SettingsTab({super.key, this.scrollController});

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
    final l10n = AppLocalizations.of(context)!;

    return SingleChildScrollView(
      controller: scrollController,
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SwitchListTile(
            contentPadding: EdgeInsets.zero,
            title: Padding(
              padding: const EdgeInsets.only(bottom: 8.0),
              child: Text(
                l10n.auto_routing,
                style: theme.textTheme.titleSmall?.copyWith(
                  fontWeight: FontWeight.bold,
                ),
              ),
            ),
            subtitle: Text(l10n.auto_routing_hint),
            value: state.autoRoutingEnabled,
            onChanged: (_) => notifier.toggleAutoRouting(),
          ),
          const SizedBox(height: 8),
          Text(
            l10n.travel_profile,
            style: theme.textTheme.titleSmall?.copyWith(
              fontWeight: FontWeight.bold,
            ),
          ),
          const SizedBox(height: 8),
          for (final bucket in RouteTravelBucket.values) ...[
            _BucketCard(
              bucket: bucket,
              icon: bucketIcon(
                bucket,
                categories,
                ref.watch(subcategoryProvider),
              ),
              selected: bucket == selectedBucket,
              // No-op on the already-active bucket — switchProfile clears
              // undo/redo and re-dispatches every segment, so re-tapping the
              // current selection must not pay that cost for nothing.
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
          color: selected
              ? theme.colorScheme.primary
              : theme.colorScheme.outline,
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
                Icon(
                  Icons.check_circle,
                  color: theme.brightness == Brightness.light
                      ? theme.colorScheme.primary
                      : theme.colorScheme.onSurface,
                ),
            ],
          ),
        ),
      ),
    );
  }
}
