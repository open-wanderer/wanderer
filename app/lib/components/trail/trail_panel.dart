import 'package:duration/duration.dart';
import 'package:flutter/material.dart';
import 'package:flutter_html/flutter_html.dart' as html;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:wanderer/components/trail/elevation_profile.dart';
import 'package:wanderer/components/trail/photo_collage.dart';
import 'package:wanderer/components/trail/stat_chip.dart';
import 'package:wanderer/components/trail/trail_timeline.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/util/format_util.dart';
import 'package:wanderer/util/gpx_util.dart';

class TrailPanel extends ConsumerWidget {
  const TrailPanel({
    super.key,
    required this.trail,
    required this.scrollController,
    this.actionMenu,
  });

  final Trail trail;
  final ScrollController scrollController;
  final Widget? actionMenu;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final user = ref.watch(authProvider).requireValue!;

    final webPhotos = trail.photos
        .map((p) => trail.getFileUrl(user.serverUrl, p, thumb: '600x0') ?? '')
        .toList();

    final totals = trail.expand?.gpx?.getTotals();

    return DefaultTabController(
      length: 3,
      child: SingleChildScrollView(
        controller: scrollController,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Center(
              child: Container(
                margin: const EdgeInsets.symmetric(vertical: 12),
                width: 40,
                height: 5,
                decoration: BoxDecoration(
                  color: Colors.grey.withValues(alpha: 0.3),
                  borderRadius: BorderRadius.circular(10),
                ),
              ),
            ),
            PhotoCollage(webPhotos: webPhotos, localPhotos: trail.localPhotos),
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 20, 20, 0),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Expanded(
                        child: Text(
                          trail.name,
                          style: Theme.of(context).textTheme.headlineMedium
                              ?.copyWith(fontWeight: FontWeight.bold),
                        ),
                      ),
                      SizedBox(width: 8),
                      ?actionMenu,
                    ],
                  ),
                  const SizedBox(height: 12),

                  Wrap(
                    spacing: 10,
                    runSpacing: 10,
                    children: [
                      StatChip(
                        icon: FontAwesomeIcons.ruler,
                        label: formatDistance(trail.distance),
                      ),
                      StatChip(
                        icon: FontAwesomeIcons.clock,
                        label: Duration(seconds: trail.duration.toInt()).pretty(
                          abbreviated: true,
                          tersity: DurationTersity.minute,
                        ),
                      ),
                      StatChip(
                        icon: FontAwesomeIcons.arrowTrendUp,
                        label: formatElevation(trail.elevationGain),
                      ),
                      StatChip(
                        icon: FontAwesomeIcons.arrowTrendDown,
                        label: formatElevation(trail.elevationLoss),
                      ),
                      if (trail.expand?.category != null)
                        StatChip(
                          icon: FontAwesomeIcons.route,
                          label: trail.expand!.category!.name,
                        ),
                    ],
                  ),
                ],
              ),
            ),

            const SizedBox(height: 16),

            const Divider(height: 1, thickness: 1),
            TabBar(
              labelStyle: Theme.of(
                context,
              ).textTheme.labelLarge?.copyWith(fontWeight: FontWeight.w600),
              unselectedLabelStyle: Theme.of(context).textTheme.labelLarge,
              dividerHeight: 0,
              tabs: [
                Tab(text: AppLocalizations.of(context)!.about),
                Tab(text: AppLocalizations.of(context)!.summit_book),
                Tab(text: AppLocalizations.of(context)!.comment(2)),
              ],
            ),

            _TabContent(
              children: [
                Padding(
                  padding: const EdgeInsets.all(20),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        AppLocalizations.of(context)!.description,
                        style: Theme.of(context).textTheme.titleMedium
                            ?.copyWith(fontWeight: FontWeight.bold),
                      ),
                      const SizedBox(height: 8),
                      if (trail.description.isNotEmpty)
                        html.Html(data: trail.description),
                      if (trail.description.isEmpty)
                        Text(
                          AppLocalizations.of(context)!.no_description_for_now,
                          style: Theme.of(context).textTheme.bodyMedium,
                        ),
                      SizedBox(height: 16),
                      Text(
                        AppLocalizations.of(context)!.route(1),
                        style: Theme.of(context).textTheme.titleMedium
                            ?.copyWith(fontWeight: FontWeight.bold),
                      ),
                      if (trail.expand?.gpx != null) ...{
                        ElevationProfile(gpx: trail.expand!.gpx!),
                        SizedBox(height: 16),
                      },
                      TrailTimeline(
                        waypoints: trail.expand?.waypointsViaTrail ?? [],
                        totalDistance: totals?.totalDistance,
                      ),
                    ],
                  ),
                ),

                const SizedBox.shrink(),

                const SizedBox.shrink(),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _TabContent extends StatefulWidget {
  const _TabContent({required this.children});

  final List<Widget> children;

  @override
  State<_TabContent> createState() => _TabContentState();
}

class _TabContentState extends State<_TabContent> {
  int _index = 0;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final controller = DefaultTabController.of(context);
    controller.addListener(_onTabChanged);
  }

  void _onTabChanged() {
    final controller = DefaultTabController.of(context);
    if (!controller.indexIsChanging) {
      setState(() => _index = controller.index);
    }
  }

  @override
  Widget build(BuildContext context) {
    return IndexedStack(
      index: _index,
      sizing: StackFit.passthrough,
      children: widget.children,
    );
  }
}
