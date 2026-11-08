import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:wanderer/components/trail/elevation_profile.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/provider/planned_gpx_provider.dart';
import 'package:wanderer/provider/route_anchor_provider.dart';
import 'package:wanderer/util/gpx/gpx.dart';

/// The Elevation tab of the route planner's tabbed sheet.
///
/// Watches [plannedElevationGpxProvider] for the elevation-merged `Gpx`
/// derived from the in-progress route's segments. Each segment's own
/// `/valhalla/height` fetch is owned by `route_anchor_provider.dart`, so this
/// tab makes no network call of its own — it renders whatever's already on
/// `routeAnchorsProvider`, filling in progressively as fetches resolve.
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
    final l10n = AppLocalizations.of(context)!;
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
            l10n.add_at_least_2_anchors_hint,
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
