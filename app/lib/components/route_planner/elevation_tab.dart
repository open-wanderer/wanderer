import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:gpx/gpx.dart';
import 'package:wanderer/components/trail/elevation_profile.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/planned_gpx_provider.dart';
import 'package:wanderer/util/gpx_util.dart';

/// The Elevation tab of the route planner's tabbed sheet (PLANUI-02).
///
/// Watches [plannedGpxProvider] for the live pre-elevation `Gpx` derived from
/// the in-progress route, fetches `POST /valhalla/height` only while this tab
/// is the active `TabBarView` page (D-11, Pitfall 2 — gated on `TabController`
/// visibility, not widget lifecycle, since `TabBarView` keeps both tabs built
/// simultaneously), debounced ~500ms on further route edits (matching
/// `GlobalSearchNotifier`'s debounce idiom). Renders the adapted
/// `ElevationProfile` (D-12: `trail: null`) with a local ele-merged copy of
/// the `Gpx`; never writes `ele` back into [plannedGpxProvider] (D-10) — that
/// provider stays pre-elevation for Phase 21's handoff.
///
/// Assumes this widget is the second page (index 1) of a 2-tab
/// `DefaultTabController` (tab 0 = Route Anchors, tab 1 = Elevation).
class ElevationTab extends ConsumerStatefulWidget {
  final ScrollController? scrollController;
  final bool isVisible;
  const ElevationTab({
    super.key,
    required this.isVisible,
    this.scrollController,
  });

  @override
  ConsumerState<ElevationTab> createState() => _ElevationTabState();
}

class _ElevationTabState extends ConsumerState<ElevationTab> {
  Timer? _debounce;
  Gpx? _eleMergedGpx;

  @override
  void didUpdateWidget(covariant ElevationTab oldWidget) {
    super.didUpdateWidget(oldWidget);

    // Gated on visibility transitions directly via widget properties
    if (widget.isVisible && !oldWidget.isVisible) {
      _scheduleFetch();
    } else if (!widget.isVisible && oldWidget.isVisible) {
      _debounce?.cancel();
    }
  }

  @override
  void dispose() {
    _debounce?.cancel();
    super.dispose();
  }

  void _scheduleFetch() {
    _debounce?.cancel();
    _debounce = Timer(const Duration(milliseconds: 500), _fetchHeights);
  }

  Future<void> _fetchHeights() async {
    if (!mounted || !widget.isVisible) return;

    final gpx = ref.read(plannedGpxProvider);
    final points = gpx.allPoints;
    if (points.length < 2) return;

    // Downsample once and merge the response back against this exact shape
    // (not the full original point list) so index alignment holds.
    final shape = buildNavShape(points);

    try {
      final api = ref.read(apiProvider);
      final response = await api.post(
        '/valhalla/height',
        data: {'shape': shape},
      );
      if (!mounted || !widget.isVisible) return;

      final heights = (response.data['height'] as List).cast<num>();
      final merged = _buildEleMergedGpx(shape, heights);
      setState(() {
        _eleMergedGpx = merged;
      });
    } catch (_) {
      // Best-effort: leave the last successfully-merged Gpx (or none) in
      // place rather than surfacing a fetch error in this tab.
    }
  }

  Gpx _buildEleMergedGpx(List<Map<String, double>> shape, List<num> heights) {
    final gpx = Gpx();
    if (shape.isEmpty) return gpx;
    gpx.trks = [
      Trk(
        trksegs: [
          Trkseg(
            trkpts: [
              for (var i = 0; i < shape.length; i++)
                Wpt(
                  lat: shape[i]['lat'],
                  lon: shape[i]['lon'],
                  ele: i < heights.length ? heights[i].toDouble() : null,
                ),
            ],
          ),
        ],
      ),
    ];
    return gpx;
  }

  @override
  Widget build(BuildContext context) {
    final gpx = ref.watch(plannedGpxProvider);

    // Re-schedule the (debounced) height fetch whenever the planned route
    // changes while this tab is visible. plannedGpxProvider itself re-emits
    // regardless of tab visibility, so the gating happens here, not there.
    ref.listen(plannedGpxProvider, (previous, next) {
      if (widget.isVisible) _scheduleFetch();
    });

    if (gpx.allPoints.length < 2) {
      return const _ElevationEmptyState();
    }

    final displayGpx = _eleMergedGpx ?? gpx;

    return SingleChildScrollView(
      controller: widget.scrollController,
      child: Padding(
        padding: const EdgeInsets.all(20.0),
        child: ElevationProfile(trail: null, gpx: displayGpx),
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
