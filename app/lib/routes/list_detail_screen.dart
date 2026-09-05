import 'package:cached_network_image/cached_network_image.dart';
import 'package:wanderer/components/map/map_ui_controls.dart';
import 'package:duration/duration.dart';
import 'package:flutter/material.dart';
import 'package:flutter_html/flutter_html.dart' as html;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/components/base/trail_collection_map.dart';
import 'package:wanderer/components/base/wanderer_attribution.dart';
import 'package:wanderer/components/base/wanderer_error.dart';
import 'package:wanderer/components/map/trail_layer.dart'
    show TrailLayer, kTrailRouteColor;
import 'package:wanderer/components/trail/stat_chip.dart';
import 'package:wanderer/components/trail/trail_list_item.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/list.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/auth_provider.dart';
import 'package:wanderer/provider/local_settings_provider.dart';
import 'package:wanderer/provider/trail/list_provider.dart';
import 'package:wanderer/util/format.dart';
import 'package:collection/collection.dart';
import 'package:wanderer/components/base/actor_avatar.dart';
import 'package:wanderer/models/category.dart';
import 'package:wanderer/provider/trail/category_provider.dart';
import 'package:wanderer/provider/trail/subcategory_provider.dart';
import 'package:wanderer/components/category/category_icon.dart';
import 'package:wanderer/util/geo/polyline.dart';

class ListDetailScreen extends ConsumerStatefulWidget {
  final String id;
  const ListDetailScreen({super.key, required this.id});

  @override
  ConsumerState<ListDetailScreen> createState() => _ListDetailScreenState();
}

class _ListDetailScreenState extends ConsumerState<ListDetailScreen> {
  late final ScrollController _scrollController = ScrollController();
  double _appBarOpacity = 0.0;

  @override
  void initState() {
    super.initState();
    _scrollController.addListener(_onScroll);
  }

  @override
  void dispose() {
    _scrollController.removeListener(_onScroll);
    _scrollController.dispose();
    super.dispose();
  }

  void _onScroll() {
    if (_scrollController.hasClients) {
      const maxScrollDistance = 128.0;
      final newOpacity = (_scrollController.offset / maxScrollDistance).clamp(
        0.0,
        1.0,
      );
      if (newOpacity != _appBarOpacity) {
        setState(() => _appBarOpacity = newOpacity);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final listAsync = ref.watch(listProvider(widget.id));
    final theme = Theme.of(context);

    return listAsync.when(
      data: (list) => Scaffold(
        extendBodyBehindAppBar: true,
        appBar: AppBar(
          leading: IconButton(
            icon: const FaIcon(FontAwesomeIcons.arrowLeft, size: 18),
            onPressed: () => context.pop(),
            style: IconButton.styleFrom(
              backgroundColor: theme.colorScheme.surface.withValues(
                alpha: 1.0 - _appBarOpacity,
              ),
            ),
          ),
          backgroundColor: theme.colorScheme.surface.withValues(
            alpha: _appBarOpacity,
          ),
          shadowColor: Colors.black.withValues(alpha: _appBarOpacity * 0.15),
          elevation: _appBarOpacity > 0 ? 2 : 0,
          scrolledUnderElevation: 0,
        ),
        body: SingleChildScrollView(
          controller: _scrollController,
          child: _ListHeader(list: list),
        ),
      ),
      loading: () =>
          const Scaffold(body: Center(child: CircularProgressIndicator())),
      error: (err, stack) => Scaffold(
        body: WandererError(err: err, stack: stack),
      ),
    );
  }
}

class _ListHeader extends ConsumerWidget {
  final WandererList list;
  const _ListHeader({required this.list});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final user = ref.watch(authProvider).requireValue!;
    final unit = ref.watch(unitProvider);
    final theme = Theme.of(context);
    final l18n = AppLocalizations.of(context)!;

    final avatarUrl = list.getFileUrl(
      user.serverUrl,
      list.avatar,
      thumb: '1200x0',
    );
    final author = list.expand?.author;
    final trails = list.expand?.trails ?? [];

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (avatarUrl == null) const SizedBox(height: kToolbarHeight + 16),
        if (avatarUrl != null)
          AspectRatio(
            aspectRatio: 16 / 9,
            child: Image(
              image: CachedNetworkImageProvider(avatarUrl),
              fit: BoxFit.cover,
              errorBuilder: (_, _, _) => const SizedBox.shrink(),
            ),
          ),

        Padding(
          padding: const EdgeInsets.fromLTRB(20, 20, 20, 0),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                crossAxisAlignment: CrossAxisAlignment.center,
                children: [
                  Text(
                    list.name,
                    style: theme.textTheme.headlineMedium?.copyWith(
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                  if (list.public) FaIcon(FontAwesomeIcons.globe, size: 18),
                ],
              ),

              if (author != null) ...[
                const SizedBox(height: 4),
                InkWell(
                  onTap: () => context.push(
                    '/profile/@${author.preferredUsername}@${author.domain}',
                  ),
                  borderRadius: BorderRadius.circular(8),
                  child: Padding(
                    padding: const EdgeInsets.all(4.0),
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        ActorAvatar.fromActor(actor: author, radius: 16),
                        const SizedBox(width: 6),
                        Text(
                          "@${author.preferredUsername}@${author.domain}",
                          style: theme.textTheme.labelLarge,
                        ),
                      ],
                    ),
                  ),
                ),
              ],

              const SizedBox(height: 8),

              Text(
                '${list.trailCount} ${l18n.trail(list.trailCount)}',
                style: TextStyle(
                  color: theme.colorScheme.onSurface.withValues(alpha: 0.6),
                  fontSize: 13,
                ),
              ),

              const SizedBox(height: 12),

              Wrap(
                spacing: 10,
                runSpacing: 10,
                children: [
                  StatChip(
                    icon: FontAwesomeIcons.ruler,
                    label: formatDistance(list.distance, unit: unit),
                  ),
                  if (list.duration != null && list.duration! > 0)
                    StatChip(
                      icon: FontAwesomeIcons.clock,
                      label: Duration(seconds: list.duration!.toInt()).pretty(
                        abbreviated: true,
                        tersity: DurationTersity.minute,
                      ),
                    ),
                  StatChip(
                    icon: FontAwesomeIcons.arrowTrendUp,
                    label: formatElevation(list.elevationGain, unit: unit),
                  ),
                  StatChip(
                    icon: FontAwesomeIcons.arrowTrendDown,
                    label: formatElevation(list.elevationLoss, unit: unit),
                  ),
                ],
              ),
            ],
          ),
        ),

        const SizedBox(height: 16),
        const Divider(height: 1, thickness: 1),

        Padding(
          padding: const EdgeInsets.all(20),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              if (list.description?.isNotEmpty == true) ...[
                Text(
                  l18n.description,
                  style: theme.textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                ),
                const SizedBox(height: 16),
                html.Html(data: list.description),
              ],
              Text(
                l18n.map,
                style: theme.textTheme.titleMedium?.copyWith(
                  fontWeight: FontWeight.bold,
                ),
              ),
              const SizedBox(height: 16),
              if (trails.isNotEmpty) ...[
                ClipRRect(
                  borderRadius: BorderRadius.circular(16),
                  child: SizedBox(
                    height: 200,
                    child: Material(
                      child: Stack(
                        children: [
                          _ListMap(list: list, trails: trails),
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
                                  FontAwesomeIcons.upRightAndDownLeftFromCenter,
                                  size: 18,
                                  color: Theme.of(
                                    context,
                                  ).colorScheme.onSurface,
                                ),
                                onPressed: () =>
                                    context.push('/list/${list.id}/map'),
                              ),
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
                const SizedBox(height: 16),
                Text(
                  l18n.trail(2),
                  style: theme.textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                ),
                const SizedBox(height: 8),
                ...(trails.map(
                  (t) => TrailListItem(
                    trail: t,
                    onTrailSelect: () => context.push('/trail/${t.id}'),
                  ),
                )),
              ],
            ],
          ),
        ),
      ],
    );
  }
}

class _ListMap extends ConsumerStatefulWidget {
  final WandererList list;
  final List<Trail> trails;
  const _ListMap({required this.list, required this.trails});

  @override
  ConsumerState<_ListMap> createState() => _ListMapState();
}

class _ListMapState extends ConsumerState<_ListMap> {
  static const _trailLayer = TrailLayer();

  ml.MapController? _controller;

  @override
  Widget build(BuildContext context) {
    final allCategories = ref.watch(categoryProvider).value ?? [];
    final allSubcategories = ref.watch(subcategoryProvider);

    final polylines = widget.trails
        .where((t) => t.polyline != null && t.polyline!.isNotEmpty)
        .map(
          (t) => ml.Feature<ml.LineString>(
            geometry: ml.LineString.from(PolylineUtil.decode(t.polyline!)),
          ),
        )
        .toList();

    final lines = widget.trails
        .where((t) => t.polyline != null && t.polyline!.isNotEmpty)
        .map((t) => PolylineUtil.decode(t.polyline!))
        .toList();

    final markers = widget.trails
        .where((t) => t.lat != null && t.lon != null)
        .map((t) {
          final Category? category = t.categoryId != null
              ? allCategories.firstWhereOrNull((c) => c.id == t.categoryId)
              : null;
          final subcategory = t.subcategoryId != null
              ? allSubcategories.firstWhereOrNull(
                  (s) => s.id == t.subcategoryId,
                )
              : null;
          return ml.Marker(
            point: ml.Geographic(lat: t.lat!, lon: t.lon!),
            size: const Size(36, 36),
            // Non-interactive: no GestureDetector wrapper — matches the
            // plain, non-tappable MarkerLayer.
            child: Container(
              decoration: BoxDecoration(
                color: Theme.of(context).primaryColor,
                shape: BoxShape.circle,
                border: Border.all(color: Colors.white, width: 2),
                boxShadow: [
                  BoxShadow(
                    color: Colors.black.withValues(alpha: 0.2),
                    blurRadius: 4,
                    offset: const Offset(0, 2),
                  ),
                ],
              ),
              child: Center(
                child: trailCategoryIcon(
                  category,
                  subcategory: subcategory,
                  color: Colors.white,
                ),
              ),
            ),
          );
        })
        .toList();

    final combinedBounds = _combinedBounds(widget.trails);

    return TrailCollectionMap(
      disabled: true,
      embedded: true,
      onMapCreated: (controller) => _controller = controller,
      onStyleLoaded: (style) {
        if (combinedBounds != null) {
          _controller?.fitBounds(
            bounds: combinedBounds,
            padding: const EdgeInsets.all(40),
            // Duration.zero crashes the Android native binding
            // (animateCamera receives a null duration).
            nativeDuration: const Duration(milliseconds: 1),
          );
        }
        _trailLayer.addArrows(style, lines).ignore();
      },
      onMapEvent: (event) {
        if (event is ml.MapEventClick) {
          context.push('/list/${widget.list.id}/map');
        }
      },
      layers: polylines.isNotEmpty
          ? [
              ml.PolylineLayer(
                polylines: polylines,
                color: kTrailRouteColor,
                width: 5,
              ),
            ]
          : null,
      children: [
        if (markers.isNotEmpty) ml.WidgetLayer(markers: markers),
        const WandererMapScalebar(alignment: Alignment.topLeft),
        const WandererAttribution(
          alignment: Alignment.topLeft,
          padding: EdgeInsets.symmetric(horizontal: 10, vertical: 44),
        ),
      ],
    );
  }

  ml.LngLatBounds? _combinedBounds(List<Trail> trails) {
    final withBounds = trails.where(
      (t) => t.minLat != 0 || t.maxLat != 0 || t.minLon != 0 || t.maxLon != 0,
    );
    if (withBounds.isEmpty) return null;

    double minLat = double.infinity;
    double maxLat = double.negativeInfinity;
    double minLon = double.infinity;
    double maxLon = double.negativeInfinity;

    for (final t in withBounds) {
      if (t.minLat < minLat) minLat = t.minLat;
      if (t.maxLat > maxLat) maxLat = t.maxLat;
      if (t.minLon < minLon) minLon = t.minLon;
      if (t.maxLon > maxLon) maxLon = t.maxLon;
    }

    return ml.LngLatBounds(
      longitudeEast: maxLon,
      longitudeWest: minLon,
      latitudeNorth: maxLat,
      latitudeSouth: minLat,
    );
  }
}
