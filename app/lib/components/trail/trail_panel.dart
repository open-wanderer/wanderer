import 'package:duration/duration.dart';
import 'package:flutter/material.dart';
import 'package:flutter_html/flutter_html.dart' as html;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';
import 'package:wanderer/components/base/wanderer_map.dart';
import 'package:wanderer/components/trail/comment_list.dart';
import 'package:wanderer/components/trail/elevation_profile.dart';
import 'package:wanderer/components/trail/photo_collage.dart';
import 'package:wanderer/components/trail/stat_chip.dart';
import 'package:wanderer/components/trail/summit_log_list.dart';
import 'package:wanderer/components/trail/trail_timeline.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/components/trail/trail_category_label.dart';
import 'package:wanderer/util/format_util.dart';
import 'package:wanderer/util/gpx_util.dart';

class TrailPanel extends ConsumerWidget {
  const TrailPanel({
    super.key,
    required this.trail,
    required this.scrollController,
  });

  final Trail trail;
  final ScrollController scrollController;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final user = ref.watch(authProvider).requireValue!;
    final unit = ref.watch(unitProvider);

    final webPhotos = trail.photos
        .map((p) => trail.getFileUrl(user.serverUrl, p, thumb: '1200x0') ?? '')
        .toList();

    final totals = trail.expand?.gpx?.getTotals();

    final l18n = AppLocalizations.of(context)!;

    return DefaultTabController(
      length: 3,
      child: SingleChildScrollView(
        controller: scrollController,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            if (trail.localPhotos.isEmpty && webPhotos.isEmpty)
              const SizedBox(height: kToolbarHeight + 16),
            if (trail.localPhotos.isNotEmpty || webPhotos.isNotEmpty)
              PhotoCollage(
                webPhotos: webPhotos,
                localPhotos: trail.localPhotos,
              ),
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 20, 20, 0),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      if (trail.summaryDate != null)
                        Text(
                          DateFormat.yMMMMd(
                            Localizations.localeOf(context).toString(),
                          ).format(trail.summaryDate!),
                          style: TextStyle(color: Colors.grey[600]),
                        ),
                      if (trail.isOffline) ...[
                        const SizedBox(width: 8),
                        Container(
                          padding: const EdgeInsets.symmetric(
                            horizontal: 7,
                            vertical: 2,
                          ),
                          decoration: BoxDecoration(
                            color: Colors.grey.withValues(alpha: 0.12),
                            borderRadius: BorderRadius.circular(12),
                            border: Border.all(
                              color: Colors.grey.shade400,
                              width: 1,
                            ),
                          ),
                          child: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              Icon(
                                Icons.cloud_off,
                                size: 9,
                                color: Colors.grey.shade600,
                              ),
                              const SizedBox(width: 4),
                              Text(
                                'Offline',
                                style: TextStyle(
                                  fontSize: 11,
                                  fontWeight: FontWeight.w600,
                                  color: Colors.grey.shade600,
                                ),
                              ),
                            ],
                          ),
                        ),
                      ],
                    ],
                  ),
                  Text(
                    trail.name,
                    style: Theme.of(context).textTheme.headlineMedium?.copyWith(
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                  if (trail.expand?.author != null)
                    InkWell(
                      onTap: () => context.push(
                        '/profile/@${trail.expand!.author!.preferredUsername}@${trail.expand!.author!.domain}',
                      ),
                      child: Padding(
                        padding: const EdgeInsets.all(4.0),
                        child: Row(
                          children: [
                            CircleAvatar(
                              radius: 16,
                              backgroundColor: Colors.grey.shade300,
                              backgroundImage: NetworkImage(
                                trail.summaryAuthorAvatar.isNotEmpty
                                    ? trail.summaryAuthorAvatar
                                    : "https://api.dicebear.com/7.x/initials/png?seed=${trail.summaryAuthorName}&backgroundType=gradientLinear",
                              ),
                              onBackgroundImageError: (_, _) =>
                                  FaIcon(FontAwesomeIcons.user),
                            ),
                            const SizedBox(width: 6),
                            Text(
                              "@${trail.expand!.author!.preferredUsername}@${trail.expand!.author!.domain}",
                              style: Theme.of(context).textTheme.labelLarge,
                            ),
                          ],
                        ),
                      ),
                    ),
                  const SizedBox(height: 12),
                  Wrap(
                    spacing: 10,
                    runSpacing: 10,
                    children: [
                      if (trail.categoryId != null)
                        Container(
                          padding: const EdgeInsets.symmetric(
                            horizontal: 10,
                            vertical: 6,
                          ),
                          decoration: BoxDecoration(
                            color: Colors.grey.withValues(alpha: 0.1),
                            borderRadius: BorderRadius.circular(8),
                          ),
                          child: TrailCategoryLabel(
                            categoryId: trail.categoryId!,
                            subcategoryId: trail.subcategoryId,
                            color: Colors.blueGrey,
                          ),
                        ),
                      StatChip(
                        icon: FontAwesomeIcons.ruler,
                        label: formatDistance(trail.distance, unit: unit),
                      ),
                      if (trail.duration > 0)
                        StatChip(
                          icon: FontAwesomeIcons.clock,
                          label: Duration(seconds: trail.duration.toInt())
                              .pretty(
                                abbreviated: true,
                                tersity: DurationTersity.minute,
                              ),
                        ),
                      StatChip(
                        icon: FontAwesomeIcons.arrowTrendUp,
                        label: formatElevation(trail.elevationGain, unit: unit),
                      ),
                      StatChip(
                        icon: FontAwesomeIcons.arrowTrendDown,
                        label: formatElevation(trail.elevationLoss, unit: unit),
                      ),
                    ],
                  ),
                ],
              ),
            ),

            const SizedBox(height: 16),

            const Divider(height: 1, thickness: 1),
            if (!trail.isOffline)
              TabBar(
                labelStyle: Theme.of(
                  context,
                ).textTheme.labelLarge?.copyWith(fontWeight: FontWeight.w600),
                unselectedLabelStyle: Theme.of(context).textTheme.labelLarge,
                dividerHeight: 0,
                tabs: [
                  Tab(text: l18n.about),
                  Tab(text: l18n.summit_book),
                  Tab(text: l18n.comment(2)),
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
                        l18n.description,
                        style: Theme.of(context).textTheme.titleMedium
                            ?.copyWith(fontWeight: FontWeight.bold),
                      ),
                      const SizedBox(height: 8),
                      if (trail.description.isNotEmpty)
                        html.Html(data: trail.description),
                      if (trail.description.isEmpty)
                        Text(
                          l18n.no_description_for_now,
                          style: Theme.of(context).textTheme.bodyMedium,
                        ),
                      SizedBox(height: 16),
                      Text(
                        l18n.route(1),
                        style: Theme.of(context).textTheme.titleMedium
                            ?.copyWith(fontWeight: FontWeight.bold),
                      ),
                      if (trail.expand?.gpx != null) ...{
                        SizedBox(height: 16),
                        Stack(
                          children: [
                            ClipRRect(
                              borderRadius: BorderRadius.all(
                                Radius.circular(16),
                              ),
                              child: SizedBox(
                                height: 200,
                                child: Material(
                                  child: WandererMap(
                                    trail: trail,
                                    disabled: true,
                                    offline: trail.isOffline,
                                    onTap: (_, _) =>
                                        context.push('/trail/${trail.id}/map'),
                                  ),
                                ),
                              ),
                            ),
                            Align(
                              alignment: Alignment.topRight,
                              child: Padding(
                                padding: const EdgeInsets.all(8.0),
                                child: IconButton(
                                  style: IconButton.styleFrom(
                                    backgroundColor: Theme.of(
                                      context,
                                    ).canvasColor,
                                  ),
                                  icon: FaIcon(
                                    FontAwesomeIcons
                                        .upRightAndDownLeftFromCenter,
                                    size: 18,
                                    color: Theme.of(
                                      context,
                                    ).colorScheme.onSurface,
                                  ),
                                  onPressed: () =>
                                      context.push('/trail/${trail.id}/map'),
                                ),
                              ),
                            ),
                          ],
                        ),
                        SizedBox(height: 16),
                        InkWell(
                          onTap: () => context.push('/trail/${trail.id}/map'),
                          child: ElevationProfile(
                            trail: trail,
                            gpx: trail.expand!.gpx!,
                            enableLineTouch: false,
                          ),
                        ),
                        SizedBox(height: 16),
                      },
                      TrailTimeline(
                        waypoints: trail.expand?.waypointsViaTrail ?? [],
                        totalDistance: totals?.totalDistance,
                      ),
                    ],
                  ),
                ),

                SummitLogList(trail: trail),

                CommentList(trail: trail),
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
  TabController? _controller;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final controller = DefaultTabController.of(context);
    if (_controller != controller) {
      _controller?.removeListener(_onTabChanged);
      _controller = controller;
      _controller!.addListener(_onTabChanged);
    }
  }

  @override
  void dispose() {
    _controller?.removeListener(_onTabChanged);
    super.dispose();
  }

  void _onTabChanged() {
    final controller = DefaultTabController.of(context);
    if (!controller.indexIsChanging) {
      setState(() => _index = controller.index);
    }
  }

  @override
  Widget build(BuildContext context) {
    return widget.children[_index];
  }
}
