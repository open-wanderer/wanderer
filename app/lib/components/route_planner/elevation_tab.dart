import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/components/trail/elevation_profile.dart';
import 'package:wanderer/provider/planned_gpx_provider.dart';
import 'package:wanderer/provider/route_anchor_provider.dart';
import 'package:wanderer/util/gpx_util.dart';

/// The Elevation tab of the route planner's tabbed sheet (PLANUI-02).
///
/// Watches [plannedElevationGpxProvider] for the elevation-merged `Gpx`
/// derived from the in-progress route's segments — every segment's own
/// `/valhalla/height` fetch is owned by `route_anchor_provider.dart`'s
/// `_resolveElevation` (fired fire-and-forget on creation/update), so this
/// tab no longer makes any network call of its own; it just renders
/// whatever's already on `routeAnchorsProvider` at any given moment, filling
/// in progressively as each segment's fetch resolves. Renders the adapted
/// `ElevationProfile` (`trail: null`).
class ElevationTab extends ConsumerWidget {
  final ScrollController? scrollController;
  const ElevationTab({super.key, this.scrollController});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final gpx = ref.watch(plannedElevationGpxProvider);

    if (gpx.allPoints.length < 2) {
      return const _ElevationEmptyState();
    }

    final estimatedDuration = Duration(
      seconds: ref.watch(routeAnchorsProvider).estimatedDurationSeconds.round(),
    );

    return SingleChildScrollView(
      controller: scrollController,
      child: Padding(
        padding: const EdgeInsets.all(20.0),
        child: ElevationProfile(
          trail: null,
          gpx: gpx,
          durationOverride: estimatedDuration,
        ),
      ),
    );
  }
}

class _ElevationEmptyState extends StatelessWidget {
  const _ElevationEmptyState();

  @override
  Widget build(BuildContext context) {
    return Container(
      height: 200,
      decoration: BoxDecoration(color: Theme.of(context).colorScheme.surface),
      alignment: Alignment.center,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const Icon(Icons.terrain, size: 48),
          const SizedBox(height: 8),
          Text(
            'Add at least 2 anchors to see the elevation profile.',
            textAlign: TextAlign.center,
            style: TextStyle(
              color: Theme.of(context).colorScheme.onSurface,
              fontSize: 14,
            ),
          ),
        ],
      ),
    );
  }
}
