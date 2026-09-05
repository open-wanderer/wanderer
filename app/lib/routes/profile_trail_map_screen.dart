import 'package:wanderer/components/map/map_ui_controls.dart';
import 'dart:async';
import 'dart:convert';

import 'package:collection/collection.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_svg/flutter_svg.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/components/async_loader.dart';
import 'package:wanderer/components/base/trail_collection_map.dart';
import 'package:wanderer/components/base/wanderer_attribution.dart';
import 'package:wanderer/components/base/wanderer_error.dart';
import 'package:wanderer/components/base/wanderer_offline_state.dart';
import 'package:wanderer/components/map/cluster_layer.dart';
import 'package:wanderer/components/map/location_marker_layer.dart';
import 'package:wanderer/components/map/trail_layer.dart' show kTrailRouteColor;
import 'package:wanderer/components/map/trail_markers.dart';
import 'package:wanderer/components/trail/trail_card.dart';
import 'package:wanderer/components/trail/trail_list_item.dart';
import 'package:wanderer/components/trail/trail_quick_filter_bar.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/provider/map_style_json_provider.dart';
import 'package:wanderer/provider/map_style_sources_provider.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/provider/profile/profile_provider.dart';
import 'package:wanderer/provider/settings_provider.dart';
import 'package:wanderer/provider/trail/category_provider.dart';
import 'package:wanderer/provider/trail/map_cluster_search_provider.dart';
import 'package:wanderer/provider/trail/map_trail_search_provider.dart';
import 'package:wanderer/provider/trail/profile_trail_bounding_box_provider.dart';
import 'package:wanderer/provider/trail/subcategory_provider.dart';
import 'package:wanderer/provider/trail/trail_deletion_provider.dart';
import 'package:wanderer/provider/trail/trail_polyline_provider.dart';
import 'package:wanderer/util/map/sheet_metrics.dart';

/// Zoom used when centering on a specific point (GPS fix or saved home
/// location). Mirrors `map_screen.dart`'s constant of the same value.
const double _kPointZoom = 13.0;

/// World-view zoom for the (0,0) fallback when no other location is known.
/// Mirrors `map_screen.dart`'s constant of the same value.
const double _kWorldZoom = 3.0;

/// Author-scoped fullscreen map — reachable from an action button on
/// `profile_trail_screen.dart` — showing exactly one profile's trails.
///
/// This is a gate-then-build split, not a single widget: [profileProvider]
/// must resolve before the inner view can exist, because the resolved actor
/// id is used as a *provider family key* for [mapTrailSearchProvider] and
/// [mapClusterSearchProvider]. The gate guarantees that key is a
/// stable, non-null string for the inner widget's entire lifetime, so an
/// unscoped search can never fire while the actor is still resolving, and
/// `dispose()` always has an unambiguous key to invalidate.
///
/// Own-profile note: `profile_trails_provider.dart` merges unsynced local
/// ObjectBox rows into the list screen; those rows are not in Meilisearch,
/// so they cannot appear here. This divergence between the list and this map
/// is explicitly permitted by CONTEXT.md — no merge is built for it.
class ProfileTrailMapScreen extends ConsumerWidget {
  final String handle;

  const ProfileTrailMapScreen({super.key, required this.handle});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final profileAsync = ref.watch(profileProvider(handle));

    return profileAsync.when(
      loading: () =>
          const Scaffold(body: Center(child: CircularProgressIndicator())),
      error: (err, stack) => Scaffold(
        body: WandererError(err: err, stack: stack),
      ),
      data: (actor) => _ProfileTrailMapView(handle: handle, authorId: actor.id),
    );
  }
}

class _ProfileTrailMapView extends ConsumerStatefulWidget {
  final String handle;
  final String authorId;

  const _ProfileTrailMapView({required this.handle, required this.authorId});

  @override
  ConsumerState<_ProfileTrailMapView> createState() =>
      _ProfileTrailMapViewState();
}

class _ProfileTrailMapViewState extends ConsumerState<_ProfileTrailMapView>
    with TickerProviderStateMixin {
  ml.MapController? _controller;
  bool _styleLoaded = false;
  bool _initialFitDone = false;

  TrailSearchResult? _selectedTrail;
  List<ml.Geographic>? _selectedPolyline;

  final DraggableScrollableController _sheetController =
      DraggableScrollableController();
  late final ValueNotifier<double> _sheetSize;
  late final AnimationController _mapButtonController;
  late final Animation<double> _mapButtonScale;
  late final AnimationController _searchAreaController;
  late final Animation<double> _searchAreaScale;
  ScrollController? _sheetScrollController;

  // No bottom nav on this route (unlike `/map`), so this starts much smaller
  // than map_screen.dart's equivalent field. Recomputed every build() from
  // the real MediaQuery once one is available.
  double sheetMinSize = 0.3;
  final sheetMediumsize = 0.5;
  final sheetMaxSize = 1.0;

  /// Deliberately the same filter id the profile's trail LIST screen
  /// uses, so a filter set in either view applies to both.
  String get _filterId => 'profile_trail_${widget.handle}';

  @override
  void initState() {
    super.initState();
    _sheetController.addListener(_onSheetSizeChanged);
    _sheetSize = ValueNotifier<double>(sheetMinSize);
    _mapButtonController = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 500),
      reverseDuration: const Duration(milliseconds: 0),
    );
    _mapButtonScale = CurvedAnimation(
      parent: _mapButtonController,
      curve: Curves.elasticOut,
      reverseCurve: Curves.easeOut,
    );
    _searchAreaController = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 500),
      reverseDuration: const Duration(milliseconds: 0),
    );
    _searchAreaScale = CurvedAnimation(
      parent: _searchAreaController,
      curve: Curves.elasticOut,
      reverseCurve: Curves.easeOut,
    );
    // Deliberately no GPS-chase subscription here — this screen never
    // reads or writes `mapCameraProvider`; the initial camera is either the
    // bbox fit below or the world-view/settings-location fallback.
  }

  void _onSheetSizeChanged() {
    _sheetSize.value = _sheetController.size;
    if (_sheetController.size >= sheetMaxSize) {
      if (!_mapButtonController.isAnimating &&
          !_mapButtonController.isCompleted) {
        _mapButtonController.forward();
      }
    } else {
      if (_mapButtonController.value > 0) {
        _mapButtonController.reverse();
      }
    }
  }

  @override
  void dispose() {
    _sheetController.removeListener(_onSheetSizeChanged);
    _sheetController.dispose();
    _sheetSize.dispose();
    _mapButtonController.dispose();
    _searchAreaController.dispose();
    // So keepAlive family instances do not accumulate one pair per visited
    // profile — this screen's own two family instances are invalidated on
    // exit rather than left to live forever.
    ref.invalidate(
      mapTrailSearchProvider(authorId: widget.authorId, filterId: _filterId),
    );
    ref.invalidate(
      mapClusterSearchProvider(authorId: widget.authorId, filterId: _filterId),
    );
    super.dispose();
  }

  /// Fetch-then-fit trail selection — identical shape to `map_screen.dart`'s
  /// `_selectTrail`, scoped to this screen's family key.
  void _selectTrail(String trailId) {
    final trails =
        ref
            .read(
              mapTrailSearchProvider(
                authorId: widget.authorId,
                filterId: _filterId,
              ),
            )
            .value ??
        [];
    setState(() {
      _selectedTrail = trails.firstWhereOrNull((t) => t.id == trailId);
      _selectedPolyline = null;
    });

    ref.read(trailPolylineProvider(trailId).future).then((polyline) {
      if (!mounted || _selectedTrail?.id != trailId) return;
      if (polyline != null) {
        _controller?.fitBounds(
          bounds: ml.LngLatBounds.fromPoints(polyline),
          padding: const EdgeInsets.fromLTRB(40, 56, 40, 248),
          nativeDuration: const Duration(milliseconds: 750),
        );
      }
      setState(() => _selectedPolyline = polyline);
    });
  }

  /// Fits the camera to the profile's bounding box (or falls back to the
  /// default camera when there is none/it could not be resolved), then
  /// always runs the initial bounds search so the sheet populates.
  ///
  /// Called from both `onStyleLoaded` and a `ref.listen` on
  /// [profileTrailBoundingBoxProvider] — the bbox fetch is async and either
  /// one can resolve first. No-ops until both are ready, and only ever runs
  /// once per screen lifetime.
  void _maybeFitAndSearch() {
    if (_initialFitDone) return;
    final controller = _controller;
    if (controller == null || !_styleLoaded) return;

    final bboxAsync = ref.read(profileTrailBoundingBoxProvider(widget.handle));
    final bbox = bboxAsync.value;
    if (bbox == null) return;

    _initialFitDone = true;

    // Read MediaQuery before any await below.
    final padding = EdgeInsets.fromLTRB(
      32,
      MediaQuery.paddingOf(context).top + kToolbarHeight + 16 + 48,
      32,
      MediaQuery.of(context).size.height * sheetMediumsize + 16,
    );

    Future<void> cameraFuture;
    if (bbox.hasTrails) {
      final bounds = ml.LngLatBounds(
        longitudeEast: bbox.maxLon,
        longitudeWest: bbox.minLon,
        latitudeNorth: bbox.maxLat,
        latitudeSouth: bbox.minLat,
      );
      final hasExtent =
          bounds.latitudeNorth != bounds.latitudeSouth ||
          bounds.longitudeEast != bounds.longitudeWest;
      if (hasExtent) {
        // Never Duration.zero — the Android native binding throws.
        cameraFuture = controller.fitBounds(
          bounds: bounds,
          padding: padding,
          nativeDuration: const Duration(milliseconds: 1),
        );
      } else {
        // A single single-point trail — no real extent to fit.
        cameraFuture = controller.animateCamera(
          center: ml.Geographic(lat: bbox.minLat, lon: bbox.minLon),
          zoom: _kPointZoom,
          nativeDuration: const Duration(milliseconds: 1),
        );
      }
    } else {
      // No trails, or every degraded federated path: skip the fit
      // entirely and leave the default camera. The search below still runs
      // so the sheet shows its real empty state rather than a spinner.
      cameraFuture = Future<void>.value();
    }

    cameraFuture.then((_) {
      if (!mounted) return;
      final bounds = controller.getVisibleRegion();
      final zoom = controller.getCamera().zoom;
      ref
          .read(
            mapClusterSearchProvider(
              authorId: widget.authorId,
              filterId: _filterId,
            ).notifier,
          )
          .searchInBounds(bounds, zoom);
      ref
          .read(
            mapTrailSearchProvider(
              authorId: widget.authorId,
              filterId: _filterId,
            ).notifier,
          )
          .searchInBounds(bounds);
    });
  }

  /// Re-probes connectivity, and if it's back, invalidates the keepAlive
  /// providers `TrailCollectionMap` depends on and re-runs the last bounds
  /// search — copied verbatim from `map_screen.dart`'s `_retryOnline`.
  Future<void> _retryOnline() async {
    final online = await ref.read(onlineStatusProvider.notifier).refresh();
    if (!online) return;

    ref.invalidate(mapStyleSourcesProvider);
    ref.invalidate(mapStyleJsonProvider);

    _controller = null;
    _styleLoaded = false;
    _initialFitDone = false;
  }

  /// Offline stand-in for the trail-results list inside the draggable sheet.
  /// Identical to `map_screen.dart`'s equivalent except for the bottom
  /// padding, which has no bottom nav to clear on this route.
  /// The result sheet's no-matches state — extracted from the sheet list so
  /// the list itself could become a `ListView.builder` (mirrors map_screen).
  Widget _buildSheetEmptyState(BuildContext context) {
    return Padding(
      padding: EdgeInsetsGeometry.only(top: 64),
      child: Column(
        children: [
          SvgPicture.asset(
            "assets/svgs/empty_state_search_${Theme.of(context).brightness.name}.svg",
            semanticsLabel: 'wanderer comment empty state',
            height: 120,
          ),
          const SizedBox(height: 16),
          Text(
            AppLocalizations.of(context)!.no_trails_found,
            style: Theme.of(context).textTheme.labelLarge!.copyWith(
              color: Theme.of(
                context,
              ).colorScheme.onSurface.withValues(alpha: 0.5),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildSheetOfflineState(ScrollController scrollController) {
    final theme = Theme.of(context);
    final l10n = AppLocalizations.of(context)!;
    final muted = theme.colorScheme.onSurface.withValues(alpha: 0.5);

    return ListView(
      controller: scrollController,
      physics: const AlwaysScrollableScrollPhysics(),
      padding: EdgeInsets.fromLTRB(
        16,
        8,
        16,
        MediaQuery.paddingOf(context).bottom + 16 + 32,
      ),
      children: [
        Center(
          child: Container(
            width: 30,
            height: 5,
            margin: const EdgeInsets.only(top: 4),
            decoration: BoxDecoration(
              color: theme.colorScheme.outline,
              borderRadius: BorderRadius.circular(10),
            ),
          ),
        ),
        Padding(
          padding: const EdgeInsets.only(top: 64),
          child: Column(
            children: [
              FaIcon(FontAwesomeIcons.linkSlash, size: 32, color: muted),
              const SizedBox(height: 16),
              Text(
                l10n.offline_title,
                style: theme.textTheme.labelLarge!.copyWith(color: muted),
              ),
              const SizedBox(height: 8),
              Text(
                l10n.offline_trail_search_body,
                textAlign: TextAlign.center,
                style: theme.textTheme.bodySmall?.copyWith(color: muted),
              ),
            ],
          ),
        ),
      ],
    );
  }

  /// Fake search bar hosted in the app-bar title slot, copied from
  /// `route_planner_screen.dart`'s `_buildSearchBar`.
  Widget _buildSearchBar(BuildContext context) {
    return GestureDetector(
      onTap: _openLocationSearch,
      child: Material(
        color: Theme.of(context).colorScheme.surface,
        borderRadius: BorderRadius.circular(24),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
          child: Row(
            children: [
              FaIcon(
                FontAwesomeIcons.magnifyingGlass,
                size: 16,
                color: Theme.of(context).colorScheme.onSurface,
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  AppLocalizations.of(context)!.search_location,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(color: Colors.grey, fontSize: 16),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  /// Pushes the location-search screen, awaits the selected
  /// [LocationSearchResult], then pans the map to it — copied from
  /// `route_planner_screen.dart`'s `_openLocationSearch`.
  Future<void> _openLocationSearch() async {
    final result = await context.push<LocationSearchResult>('/location-search');
    if (!mounted || result == null || _controller == null) return;

    _controller!.animateCamera(
      center: ml.Geographic(lat: result.lat, lon: result.lon),
      zoom: 13,
      nativeDuration: const Duration(milliseconds: 750),
    );
  }

  @override
  Widget build(BuildContext context) {
    final isOnline = ref.watch(onlineStatusProvider);

    sheetMinSize = (56 + 48) / MediaQuery.of(context).size.height;

    ml.Geographic initCenter = const ml.Geographic(lat: 0, lon: 0);
    double initZoom = _kWorldZoom;
    final settingsLocation = ref.watch(settingsProvider)?.location;
    if (settingsLocation != null) {
      initCenter = ml.Geographic(
        lat: settingsLocation.lat,
        lon: settingsLocation.lon,
      );
      initZoom = _kPointZoom;
    }

    // The bbox is async and can resolve after the style already loaded —
    // this listener is the other half of the race `_maybeFitAndSearch` has
    // to win against `onStyleLoaded`.
    ref.listen(profileTrailBoundingBoxProvider(widget.handle), (
      previous,
      next,
    ) {
      if (next.hasValue) _maybeFitAndSearch();
    });

    // Swap the native cluster source's data in place on every new result —
    // never remove and re-add the source.
    ref.listen(
      mapClusterSearchProvider(authorId: widget.authorId, filterId: _filterId),
      (previous, next) {
        final style = _controller?.style;
        final data = next.value;
        if (style == null || data == null || next.isLoading) return;
        updateClusterSource(style, jsonEncode(data)).ignore();
      },
    );

    // A deleted trail must leave the selection panel immediately — see
    // map_screen.dart's identical listener for the full rationale.
    ref.listen(trailDeletionsProvider, (previous, next) {
      if (next == null || _selectedTrail?.id != next.id) return;
      setState(() {
        _selectedTrail = null;
        _selectedPolyline = null;
      });
    });

    final searchResultAsync = ref.watch(
      mapTrailSearchProvider(authorId: widget.authorId, filterId: _filterId),
    );
    final trails = searchResultAsync.value ?? [];
    final allCategories = ref.watch(categoryProvider).value ?? [];
    final allSubcategories = ref.watch(subcategoryProvider);

    final clusterAsync = ref.watch(
      mapClusterSearchProvider(authorId: widget.authorId, filterId: _filterId),
    );
    final featureCollection = clusterAsync.value;
    final features =
        (featureCollection?['features'] as List<dynamic>?) ?? const [];

    final unclusteredMarkers = buildUnclusteredTrailMarkers(
      features: features,
      trails: trails,
      categories: allCategories,
      subcategories: allSubcategories,
      context: context,
      onTrailTap: _selectTrail,
    );

    // Clears the transparent AppBar plus the filter bar docked beneath it —
    // the profile map's equivalent of map_screen's search bar + chips
    // clearance for the compass/scalebar.
    final topOverlayClearance =
        MediaQuery.paddingOf(context).top + kToolbarHeight + 48;

    return Scaffold(
      extendBodyBehindAppBar: true,
      appBar: AppBar(
        backgroundColor: Colors.transparent,
        elevation: 0,
        scrolledUnderElevation: 0,
        leading: IconButton(
          icon: const FaIcon(FontAwesomeIcons.arrowLeft, size: 18),
          onPressed: () => context.pop(),
          style: IconButton.styleFrom(
            backgroundColor: Theme.of(context).colorScheme.surface,
          ),
        ),
        titleSpacing: 8,
        title: _buildSearchBar(context),
      ),
      body: Stack(
        children: [
          if (!isOnline)
            Container(
              color: Theme.of(context).colorScheme.surface,
              child: WandererOfflineState(
                title: AppLocalizations.of(context)!.offline_title,
                body: AppLocalizations.of(context)!.offline_map_body,
                retryLabel: AppLocalizations.of(context)!.offline_try_again,
                onRetry: _retryOnline,
              ),
            )
          else
            TrailCollectionMap(
              initCenter: initCenter,
              initZoom: initZoom,
              onMapCreated: (controller) => _controller = controller,
              onStyleLoaded: (style) async {
                try {
                  final data =
                      ref
                          .read(
                            mapClusterSearchProvider(
                              authorId: widget.authorId,
                              filterId: _filterId,
                            ),
                          )
                          .value ??
                      const {
                        'type': 'FeatureCollection',
                        'features': <Object>[],
                      };
                  await addClusterLayers(style, jsonEncode(data));
                } catch (e) {
                  debugPrint(
                    'profile_trail_map_screen: failed to add cluster layers — $e',
                  );
                }

                _styleLoaded = true;
                _maybeFitAndSearch();
              },
              onMapEvent: (event) {
                if (event is ml.MapEventStartMoveCamera &&
                    event.reason == ml.CameraChangeReason.apiGesture) {
                  _searchAreaController.forward();
                  return;
                }

                if (event is ml.MapEventClick) {
                  final controller = _controller;
                  if (controller == null) return;

                  final clusterHits = controller.featuresAtPoint(
                    event.screenPoint,
                    layerIds: const ['clusters'],
                  );
                  if (clusterHits.isNotEmpty) {
                    final currentZoom = controller.getCamera().zoom;
                    controller
                        .animateCamera(
                          center: event.point,
                          zoom: currentZoom + 2,
                          nativeDuration: const Duration(milliseconds: 400),
                        )
                        .then((_) {
                          if (!mounted) return;
                          final bounds = controller.getVisibleRegion();
                          final zoom = controller.getCamera().zoom;
                          ref
                              .read(
                                mapClusterSearchProvider(
                                  authorId: widget.authorId,
                                  filterId: _filterId,
                                ).notifier,
                              )
                              .searchInBounds(bounds, zoom);
                          ref
                              .read(
                                mapTrailSearchProvider(
                                  authorId: widget.authorId,
                                  filterId: _filterId,
                                ).notifier,
                              )
                              .searchInBounds(bounds);
                        });
                    return;
                  }

                  // Background tap — deselect and collapse the sheet.
                  if (_sheetController.isAttached) {
                    _sheetController.animateTo(
                      sheetMediumsize,
                      curve: Curves.easeOut,
                      duration: const Duration(milliseconds: 300),
                    );
                  }
                  setState(() {
                    _selectedTrail = null;
                    _selectedPolyline = null;
                  });
                  return;
                }

                // Deliberately no MapEventCameraIdle branch here —
                // this screen never reads or writes `mapCameraProvider`.
              },
              layers: _selectedPolyline != null
                  ? [
                      ml.PolylineLayer(
                        polylines: [
                          ml.Feature<ml.LineString>(
                            geometry: ml.LineString.from(_selectedPolyline!),
                          ),
                        ],
                        color: kTrailRouteColor,
                        width: 5,
                      ),
                    ]
                  : null,
              children: [
                if (unclusteredMarkers.isNotEmpty)
                  ml.WidgetLayer(
                    allowInteraction: true,
                    markers: unclusteredMarkers,
                  ),
                const LocationMarkerLayer(),
                Positioned(
                  top: topOverlayClearance + 16,
                  right: 8,
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.center,
                    children: const [
                      WandererMapCompass(hideIfRotatedNorth: true),
                    ],
                  ),
                ),
                WandererMapScalebar(
                  alignment: Alignment.topLeft,
                  padding: EdgeInsets.only(left: 24, top: kToolbarHeight + 24),
                ),
                WandererAttribution(
                  alignment: Alignment.bottomLeft,
                  padding: EdgeInsets.only(
                    left: 10,
                    bottom: MediaQuery.paddingOf(context).bottom + 72,
                  ),
                ),
              ],
            ),

          // Filter bar, docked directly beneath the transparent AppBar.
          if (isOnline)
            Positioned(
              top: MediaQuery.paddingOf(context).top + kToolbarHeight,
              left: 0,
              right: 0,
              // The offline chip works here because the clause is an
              // `id IN [...]` whitelist evaluated server-side, so clusters are
              // aggregated over the already-filtered set. A client-side
              // intersection could not have done this -- a cluster is a count,
              // not a trail -- and would have made filtering depend on zoom.
              child: TrailQuickFilterBar(filterId: _filterId),
            ),

          if (_selectedTrail == null && isOnline)
            ValueListenableBuilder<double>(
              valueListenable: _sheetSize,
              builder: (context, sheetSize, child) {
                final clampedSize = sheetSize < sheetMediumsize
                    ? sheetSize
                    : sheetMediumsize;
                return Positioned(
                  bottom: MediaQuery.of(context).size.height * clampedSize + 12,
                  right: 16,
                  child: child!,
                );
              },
              child: Material(
                color: Theme.of(context).colorScheme.surface,
                elevation: 4,
                shadowColor: Colors.black26,
                shape: const CircleBorder(),
                child: IconButton(
                  icon: const FaIcon(
                    FontAwesomeIcons.locationCrosshairs,
                    size: 18,
                  ),
                  tooltip: AppLocalizations.of(context)!.center_on_my_location,
                  onPressed: () {
                    final location = ref.read(settingsProvider)?.location;
                    if (location == null) return;
                    _controller?.animateCamera(
                      center: ml.Geographic(
                        lat: location.lat,
                        lon: location.lon,
                      ),
                      zoom: _kPointZoom,
                      nativeDuration: const Duration(milliseconds: 750),
                    );
                  },
                ),
              ),
            ),

          if (_selectedTrail == null && isOnline)
            ValueListenableBuilder(
              valueListenable: _sheetSize,
              builder: (context, sheetSize, child) {
                final clampedSize = sheetSize < sheetMediumsize
                    ? sheetSize
                    : sheetMediumsize;
                return Positioned(
                  bottom: MediaQuery.of(context).size.height * clampedSize + 12,
                  left: 0,
                  right: 0,
                  child: Center(
                    child: ScaleTransition(
                      scale: _searchAreaScale,
                      child: FilledButton.icon(
                        onPressed: () {
                          _searchAreaController.reverse();
                          final controller = _controller;
                          if (controller == null) return;
                          final bounds = controller.getVisibleRegion();
                          final zoom = controller.getCamera().zoom;
                          ref
                              .read(
                                mapClusterSearchProvider(
                                  authorId: widget.authorId,
                                  filterId: _filterId,
                                ).notifier,
                              )
                              .searchInBounds(bounds, zoom);
                          ref
                              .read(
                                mapTrailSearchProvider(
                                  authorId: widget.authorId,
                                  filterId: _filterId,
                                ).notifier,
                              )
                              .searchInBounds(bounds);
                        },
                        icon: const FaIcon(
                          FontAwesomeIcons.mapLocationDot,
                          size: 14,
                        ),
                        label: Text(
                          AppLocalizations.of(context)!.search_this_area,
                        ),
                      ),
                    ),
                  ),
                );
              },
            ),

          if (isOnline)
            Opacity(
              key: const ValueKey('profile_trail_sheet'),
              opacity: _selectedTrail == null ? 1 : 0,
              child: DraggableScrollableSheet(
                controller: _sheetController,
                initialChildSize: sheetMediumsize,
                minChildSize: sheetMinSize,
                maxChildSize: sheetMaxSize,
                snap: true,
                snapSizes: [sheetMinSize, sheetMediumsize, sheetMaxSize],
                builder: (context, scrollController) {
                  _sheetScrollController = scrollController;
                  return ValueListenableBuilder<double>(
                    valueListenable: _sheetSize,
                    builder: (context, size, child) => Container(
                      decoration: BoxDecoration(
                        color: Theme.of(context).canvasColor,
                        borderRadius: BorderRadius.vertical(
                          top: Radius.circular(size >= sheetMaxSize ? 0 : 16),
                        ),
                        boxShadow: [
                          BoxShadow(
                            color: Colors.black.withValues(alpha: 0.15),
                            blurRadius: 10,
                            offset: const Offset(0, -2),
                          ),
                        ],
                      ),
                      child: child,
                    ),
                    // Keyed on connectivity alone, deliberately NOT on
                    // `searchResultAsync.hasError` — the keepAlive stale-replay
                    // trap map_screen.dart documents applies per-family-key too.
                    child: !isOnline
                        ? _buildSheetOfflineState(scrollController)
                        : AsyncLoader(
                            asyncValue: searchResultAsync,
                            mockData: List.generate(
                              5,
                              (_) => TrailSearchResult.mock(),
                            ),
                            // .builder, not children — same rationale as
                            // map_screen's search sheet: rows scale with the
                            // accumulated result set. Index 0 is the header.
                            builder: (trails) => ListView.builder(
                              padding: EdgeInsets.fromLTRB(
                                16,
                                8,
                                16,
                                MediaQuery.paddingOf(context).bottom + 16 + 32,
                              ),
                              controller: scrollController,
                              itemCount:
                                  1 + (trails.isEmpty ? 1 : trails.length),
                              itemBuilder: (context, index) {
                                if (index > 0) {
                                  if (trails.isEmpty) {
                                    return _buildSheetEmptyState(context);
                                  }
                                  final t = trails[index - 1];
                                  return TrailCard(
                                    key: ValueKey(t.id),
                                    trail: t,
                                    onTrailSelect: () => context.push(
                                      "/trail/${t.id}",
                                      extra: t,
                                    ),
                                  );
                                }
                                return ValueListenableBuilder<double>(
                                  valueListenable: _sheetSize,
                                  builder: (context, size, child) {
                                    final opacity = sheetHeaderOpacity(
                                      size,
                                      sheetMediumsize: sheetMediumsize,
                                    );

                                    return Column(
                                      children: [
                                        if (opacity > 0.0)
                                          Opacity(
                                            opacity: opacity,
                                            child: Column(
                                              children: [
                                                Center(
                                                  child: Container(
                                                    width: 30,
                                                    height: 5,
                                                    margin:
                                                        const EdgeInsets.only(
                                                          top: 4,
                                                        ),
                                                    decoration: BoxDecoration(
                                                      color: Theme.of(
                                                        context,
                                                      ).colorScheme.outline,
                                                      borderRadius:
                                                          BorderRadius.circular(
                                                            10,
                                                          ),
                                                    ),
                                                  ),
                                                ),
                                                Padding(
                                                  padding: const EdgeInsets.all(
                                                    8.0,
                                                  ),
                                                  child: Center(
                                                    child: Opacity(
                                                      opacity: trails.isNotEmpty
                                                          ? 1
                                                          : (1 - size / sheetMediumsize)
                                                                .clamp(0, 1),
                                                      child: Text(
                                                        "${trails.length}${trails.length == 100 ? '+' : ''} ${AppLocalizations.of(context)!.trail(trails.length)}",
                                                        style: Theme.of(
                                                          context,
                                                        ).textTheme.labelLarge,
                                                      ),
                                                    ),
                                                  ),
                                                ),
                                              ],
                                            ),
                                          ),
                                        SizedBox(
                                          height: dynamicSheetPadding(
                                            size,
                                            sheetMinSize: sheetMinSize,
                                            sheetMediumsize: sheetMediumsize,
                                            sheetMaxSize: sheetMaxSize,
                                          ),
                                        ),
                                      ],
                                    );
                                  },
                                );
                              },
                            ),
                          ),
                  );
                },
              ),
            ),

          if (isOnline)
            Positioned(
              bottom: MediaQuery.paddingOf(context).bottom + 64,
              left: 0,
              right: 0,
              child: Center(
                child: ScaleTransition(
                  scale: _mapButtonScale,
                  child: FilledButton.icon(
                    onPressed: () {
                      if (_sheetScrollController?.hasClients == true) {
                        _sheetScrollController!.jumpTo(0);
                      }
                      _sheetController.animateTo(
                        sheetMinSize,
                        curve: Curves.easeOut,
                        duration: const Duration(milliseconds: 300),
                      );
                    },
                    icon: const FaIcon(FontAwesomeIcons.map),
                    label: Text(AppLocalizations.of(context)!.map),
                  ),
                ),
              ),
            ),

          if (_selectedTrail != null)
            Positioned(
              bottom: MediaQuery.paddingOf(context).bottom + 16 + 45,
              left: 16,
              right: 16,
              child: Dismissible(
                key: ValueKey(_selectedTrail!.id),
                direction: DismissDirection.down,
                onDismissed: (_) {
                  setState(() {
                    _selectedTrail = null;
                    _selectedPolyline = null;
                  });
                },
                child: TrailListItem(
                  trail: _selectedTrail!,
                  onTrailSelect: () {
                    context.push('/trail/${_selectedTrail!.id}');
                  },
                ),
              ),
            ),
        ],
      ),
    );
  }
}
