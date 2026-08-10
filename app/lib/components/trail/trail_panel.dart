import 'package:duration/duration.dart';
import 'package:flutter/material.dart';
import 'package:flutter_html/flutter_html.dart' as html;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:gpx/gpx.dart' show Gpx;
import 'package:intl/intl.dart';
import 'package:skeletonizer/skeletonizer.dart';
import 'package:wanderer/components/base/actor_avatar.dart';
import 'package:wanderer/components/base/trail_map.dart';
import 'package:wanderer/components/trail/comment_list.dart';
import 'package:wanderer/components/trail/elevation_profile.dart';
import 'package:wanderer/components/trail/photo_collage.dart';
import 'package:wanderer/components/trail/stat_chip.dart';
import 'package:wanderer/components/trail/sync_status_chip.dart';
import 'package:wanderer/components/trail/summit_log_list.dart';
import 'package:wanderer/components/trail/trail_timeline.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/components/trail/trail_category_label.dart';
import 'package:wanderer/util/format.dart';
import 'package:wanderer/util/gpx/conversion.dart';
import 'package:wanderer/util/trail/route_location.dart';

/// Identity-keyed memo for [computeTrailMetrics] — a full pass over every
/// trackpoint that used to re-run on each `TrailPanel` rebuild (the parent
/// screen rebuilds on provider changes; before the AppBar-fade fix it
/// rebuilt per scrolled pixel). [Expando] keys weakly on the parsed [Gpx]
/// object, so entries vanish with the trail model — no manual invalidation,
/// no leak.
final Expando<GpxTrailMetrics> _metricsMemo = Expando<GpxTrailMetrics>();

GpxTrailMetrics _memoizedTrailMetrics(Gpx gpx) =>
    _metricsMemo[gpx] ??= computeTrailMetrics(gpx);

class TrailPanel extends ConsumerWidget {
  const TrailPanel({
    super.key,
    required this.trail,
    required this.scrollController,
    this.availableOffline = false,
  });

  final Trail trail;
  final bool availableOffline;
  final ScrollController scrollController;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final user = ref.watch(authProvider).requireValue!;
    final unit = ref.watch(unitProvider);
    final isOnline = ref.watch(onlineStatusProvider);

    // True only while `AsyncLoader` skeletonizes this panel over `Trail.mock()`
    // (`trail_detail_screen.dart`). The `Skeleton.*` annotations below are
    // inert without it, so the local-trail path renders identically.
    final isSkeleton = Skeletonizer.maybeOf(context)?.enabled ?? false;

    final webPhotos = trail.photos
        .map((p) => trail.getFileUrl(user.serverUrl, p, thumb: '1200x0') ?? '')
        .toList();

    final metrics = trail.expand?.gpx == null
        ? null
        : _memoizedTrailMetrics(trail.expand!.gpx!);

    final displayDuration = trailDisplayDuration(trail);

    final l18n = AppLocalizations.of(context)!;

    // A local-sentinel id is blanked, so '/trail/${trail.id}/map' is
    // '/trail//map' for a not-yet-uploaded trail -- go_router canonicalizes
    // that to '/trail/map', which matches no route. `trailMapLocation`
    // returns '/trail/local/<localId>/map' instead, and null when the trail
    // is not addressable at all.
    final String? mapLocation = trailMapLocation(trail);

    // The trail model's cache-provenance flag is hardcoded `true` by
    // `TrailEntity.toModel()` for every cached row, downloaded trails
    // included, and `TrailNotifier.build()` falls back to the cache on any
    // fetch exception. Gating the server-backed tabs on that flag hid summit
    // logs and comments on any trail read off the device. `isUnsyncedState`
    // is the signal that actually means "has never reached the server, so
    // there is nothing server-side to show" -- same reasoning as
    // `trail_dropdown.dart`.
    final showsServerTabs = !isUnsyncedState(trail.syncState);

    final aboutTab = Padding(
      padding: const EdgeInsets.all(20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            l18n.description,
            style: Theme.of(
              context,
            ).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold),
          ),
          const SizedBox(height: 8),
          // Replaced rather than boned in place: Skeletonizer merges Html's
          // multi-line rich text into one notched, pill-shaped blob that
          // dominates the loading screen. A plain block reads as a paragraph.
          //
          // The dimensions are handed over only while skeletonizing:
          // `Skeleton.replace` wraps its child in a `SizedBox(width, height)`
          // whether or not it is replacing anything, so a constant 60 would go
          // on clipping the real description — an expanded one especially —
          // long after the load finished.
          if (trail.description.isNotEmpty)
            Skeleton.replace(
              width: isSkeleton ? double.infinity : null,
              height: isSkeleton ? 60 : null,
              child: _TrailDescription(description: trail.description),
            ),
          if (trail.description.isEmpty)
            Text(
              l18n.no_description_for_now,
              style: Theme.of(context).textTheme.bodyMedium,
            ),
          SizedBox(height: 16),
          Text(
            l18n.route(1),
            style: Theme.of(
              context,
            ).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold),
          ),
          if (trail.expand?.gpx != null) ...{
            SizedBox(height: 16),
            Stack(
              children: [
                ClipRRect(
                  borderRadius: BorderRadius.all(Radius.circular(16)),
                  child: SizedBox(
                    height: 200,
                    child: Material(
                      child: TrailMap(
                        trail: trail,
                        disabled: true,
                        // Connectivity, NOT trail.isOffline — see
                        // that field's doc comment. Online we
                        // always prefer network tiles, even for a
                        // downloaded trail.
                        offline: !isOnline,
                        onTap: mapLocation == null
                            ? null
                            : (_) => context.push(mapLocation),
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
                        backgroundColor: Theme.of(context).canvasColor,
                      ),
                      icon: FaIcon(
                        FontAwesomeIcons.upRightAndDownLeftFromCenter,
                        size: 18,
                        color: Theme.of(context).colorScheme.onSurface,
                      ),
                      onPressed: mapLocation == null
                          ? null
                          : () => context.push(mapLocation),
                    ),
                  ),
                ),
              ],
            ),
            SizedBox(height: 16),
            InkWell(
              onTap: mapLocation == null
                  ? null
                  : () => context.push(mapLocation),
              child: ElevationProfile(
                trail: trail,
                gpx: trail.expand!.gpx!,
                enableLineTouch: false,
              ),
            ),
            SizedBox(height: 16),
          },
          // The mock trail carries no gpx (loading one would mean fetching map
          // tiles for a trail that does not exist), so without this the
          // skeleton stops dead after the "Route" heading and leaves the lower
          // half of the screen empty. Stands in for the map block above at the
          // same geometry, since nearly every real trail has a route.
          if (isSkeleton && trail.expand?.gpx == null)
            // A single padded element rather than SizedBox/child/SizedBox:
            // the enclosing collection is a set literal, so two identical
            // spacer boxes would collapse into one.
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 16),
              child: Skeleton.leaf(
                child: Container(
                  height: 200,
                  decoration: BoxDecoration(
                    color: Theme.of(
                      context,
                    ).colorScheme.surfaceContainerHighest,
                    borderRadius: const BorderRadius.all(Radius.circular(16)),
                  ),
                ),
              ),
            ),
          TrailTimeline(
            waypoints: trail.expand?.waypointsViaTrail ?? [],
            totalDistance: metrics?.distance,
          ),
        ],
      ),
    );

    return DefaultTabController(
      length: showsServerTabs ? 3 : 1,
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
                      // This badge's axis is "is it stored on
                      // this device", which is library membership
                      // (`trailLibraryProvider`) and nothing else -- never
                      // the cache-provenance flag the trail model carries.
                      // That flag is hardcoded `true` by
                      // `TrailEntity.toModel()` for every cached row, and
                      // `TrailNotifier.build()` falls back to the cache on
                      // any fetch exception, so gating on it flips with
                      // network conditions: it used to appear when a fetch
                      // failed and vanish when one succeeded. Unsynced-ness
                      // and library membership are independent axes and a
                      // row can be both -- which is exactly why this badge
                      // is derived from
                      // membership alone: membership is a fact about this
                      // account, so the badge stays correct on an overlap
                      // row instead of depending on a premise that does not
                      // hold.
                      if (availableOffline) ...[
                        const SizedBox(width: 8),
                        Container(
                          padding: const EdgeInsets.symmetric(
                            horizontal: 7,
                            vertical: 2,
                          ),
                          decoration: BoxDecoration(
                            color: Colors.greenAccent.withValues(alpha: 0.25),
                            borderRadius: BorderRadius.circular(12),
                            border: Border.all(
                              color: Colors.green.shade400,
                              width: 1,
                            ),
                          ),
                          child: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              Icon(
                                Icons.cloud_done,
                                size: 9,
                                color: Colors.green.shade500,
                              ),
                              const SizedBox(width: 4),
                              Text(
                                l18n.available_offline,
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
                  // Gated on isUnsyncedState (rather than relying on
                  // SyncStatusChip's own synced self-suppression) so a
                  // downloaded trail's widget tree is provably unchanged --
                  // no extra SizedBox, no zero-height Align, nothing
                  // constructed at all. Placed below the title rather than
                  // inside the badge Row above: that Row has no overflow
                  // protection and already holds the date Text, so adding a
                  // ~180px "Upload failed · Tap to retry" chip to it would
                  // overflow on a narrow screen. Below-the-title also
                  // matches both existing call sites
                  // (trail_list_item.dart, trail_card.dart).
                  if (isUnsyncedState(trail.syncState)) ...[
                    const SizedBox(height: 4),
                    Align(
                      alignment: Alignment.centerLeft,
                      child: SyncStatusChip(trail: trail),
                    ),
                  ],
                  if (trail.expand?.author != null)
                    InkWell(
                      // The profile screen is server-only -- it has no cached
                      // fallback -- so offline the tap can only land on an
                      // error state. Null onTap also drops the ripple, which
                      // is the feedback that the row is inert right now.
                      onTap: !isOnline
                          ? null
                          : () => context.push(
                              '/profile/@${trail.expand!.author!.preferredUsername}@${trail.expand!.author!.domain}',
                            ),
                      child: Padding(
                        padding: const EdgeInsets.all(4.0),
                        child: Row(
                          children: [
                            ActorAvatar.fromActor(
                              actor: trail.expand!.author,
                              radius: 16,
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
                    children:
                        [
                              if (trail.categoryId?.isNotEmpty == true)
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
                                  ),
                                ),
                              StatChip(
                                icon: FontAwesomeIcons.ruler,
                                label: formatDistance(
                                  trail.distance,
                                  unit: unit,
                                ),
                              ),
                              if (displayDuration != null &&
                                  displayDuration > 0)
                                StatChip(
                                  icon: FontAwesomeIcons.clock,
                                  label:
                                      Duration(
                                        seconds: displayDuration.toInt(),
                                      ).pretty(
                                        abbreviated: true,
                                        tersity: DurationTersity.minute,
                                      ),
                                ),
                              StatChip(
                                icon: FontAwesomeIcons.arrowTrendUp,
                                label: formatElevation(
                                  trail.elevationGain,
                                  unit: unit,
                                ),
                              ),
                              StatChip(
                                icon: FontAwesomeIcons.arrowTrendDown,
                                label: formatElevation(
                                  trail.elevationLoss,
                                  unit: unit,
                                ),
                              ),
                            ]
                            // Each chip is a tinted pill wrapping an icon and a label,
                            // which skeletonizes into a pill with two bones rattling
                            // around inside it -- four of those was the busiest band
                            // on the loading screen. United, a chip is one clean pill.
                            // Inert when the skeleton is off.
                            .map(
                              (chip) => Skeleton.unite(
                                borderRadius: BorderRadius.circular(32),
                                child: chip,
                              ),
                            )
                            .toList(),
                  ),
                ],
              ),
            ),

            const SizedBox(height: 16),

            const Divider(height: 1, thickness: 1),
            if (showsServerTabs)
              // Replaced while loading: the selected-tab indicator keeps its
              // real colour under Skeletonizer, so it was the one dark stroke
              // on an otherwise grey screen. One block of the same height
              // holds the space without drawing attention.
              Skeleton.replace(
                width: double.infinity,
                height: 46,
                child: TabBar(
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
              ),

            // dart format off
            _TabContent(
              children: showsServerTabs ? [aboutTab, SummitLogList(trail: trail), CommentList(trail: trail)] : [aboutTab],
            ),
            // dart format on
          ],
        ),
      ),
    );
  }
}

/// Trail description with a show more / show less toggle.
///
/// Stateful purely to own `_expanded`, which keeps [TrailPanel] a
/// `ConsumerWidget`. Mirrors `_BioSection` in `routes/profile_screen.dart`.
class _TrailDescription extends StatefulWidget {
  const _TrailDescription({required this.description});

  final String description;

  @override
  State<_TrailDescription> createState() => _TrailDescriptionState();
}

class _TrailDescriptionState extends State<_TrailDescription> {
  static const _maxLength = 150;

  bool _expanded = false;

  @override
  Widget build(BuildContext context) {
    final description = widget.description;
    // Preview off the plain text, never off the HTML: cutting the markup at a
    // fixed offset tears tags in half and spends the budget on `<strong>`
    // rather than on words (web #1128).
    final preview = formatHtmlAsTextPreview(description, _maxLength);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (preview.truncated && !_expanded)
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 8),
            child: Text(
              '${preview.text}…',
              style: Theme.of(context).textTheme.bodyMedium,
            ),
          )
        else
          html.Html(data: description),
        if (preview.truncated)
          TextButton(
            onPressed: () => setState(() => _expanded = !_expanded),
            child: Text(
              _expanded
                  ? AppLocalizations.of(context)!.show_less
                  : AppLocalizations.of(context)!.show_more,
            ),
          ),
      ],
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
