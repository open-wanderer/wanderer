import 'package:wanderer/components/map/map_ui_controls.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:go_router/go_router.dart';
import 'package:maplibre/maplibre.dart' as ml;
import 'package:wanderer/components/map/route_anchor_layer.dart';
import 'package:wanderer/components/map/route_segment_layer.dart';
import 'package:wanderer/components/route_planner/route_anchor_sheet.dart';
import 'package:wanderer/i18n/app_localizations.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/models/route_anchor.dart';
import 'package:wanderer/provider/map_style_json_provider.dart';
import 'package:wanderer/provider/route_anchor_provider.dart';
import 'package:wanderer/provider/toast_provider.dart';
import 'package:wanderer/util/route/planner_handoff.dart';
import 'package:wanderer/util/route/segment.dart';

/// Route Planner screen — hosts the native map and wires anchor/segment
/// gestures to the UI: tap-to-add, drag-to-reposition (handled inside
/// [RouteAnchorLayer]), and tap-to-insert/tap-to-retry on a segment.
///
/// The app-bar title slot is a fake search bar (opens [LocationSearchScreen]
/// via `/location-search`), analogous to `map_screen.dart`'s search pill.
class RoutePlannerScreen extends ConsumerStatefulWidget {
  /// `'pedestrian'` or `'bicycle'` — the initial travel profile for this
  /// planning session. Applied via [RouteAnchors.resetForSession] at mount;
  /// may change mid-session via the Settings tab's [RouteAnchors.switchProfile].
  final String travelProfile;

  /// The initial Valhalla `costing_options` payload matching [travelProfile],
  /// threaded through the router's `extra['costingOptions']`. `null` when the
  /// caller didn't supply one.
  final Map<String, dynamic>? initialCostingOptions;

  /// The map's initial camera center. Required with no fallback default —
  /// inventing a hardcoded center here would silently mask a caller bug.
  final ml.Geographic initialCenter;

  /// Segment-boundary anchors seeded from an existing trail's track —
  /// non-null and non-empty means this session is editing an existing route
  /// rather than planning a new one. When present, [RouteAnchors.seedFromTrack]
  /// replaces the usual [RouteAnchors.resetForSession] empty-session reset,
  /// and Finish pops the final [Gpx] back to the caller instead of
  /// forward-pushing a draft trail via [finishPlanning].
  final List<ml.Geographic>? seedAnchors;

  /// Per-segment full-resolution polylines aligned with [seedAnchors] (from
  /// `segmentPolylinesFromTrack`) — the original recorded points between each
  /// anchor pair, so the map shows the exact prior route rather than a
  /// straight line or a Valhalla-recomputed one until the user actually edits
  /// a segment. `null`/short falls back to a straight line for the affected
  /// pair(s) in [RouteAnchors.seedFromTrack].
  final List<List<ml.Geographic>>? seedSegmentPolylines;

  const RoutePlannerScreen({
    super.key,
    required this.travelProfile,
    this.initialCostingOptions,
    required this.initialCenter,
    this.seedAnchors,
    this.seedSegmentPolylines,
  });

  @override
  ConsumerState<RoutePlannerScreen> createState() => _RoutePlannerScreenState();
}

class _RoutePlannerScreenState extends ConsumerState<RoutePlannerScreen> {
  // Instance field, not static: RouteSegmentLayer tracks whether its layers
  // were added to the CURRENT native style via its own `_added` flag. A
  // static field would go stale across remounts — the second time this
  // screen opens there's a fresh style with no layers yet, but a stale
  // `_added == true` would silently skip re-adding them, leaving segments
  // permanently unrendered.
  final _segmentLayer = RouteSegmentLayer();

  ml.MapController? _mapController;

  /// Buffers a style-loaded event that arrives before [_mapController] is
  /// set (mirrors `TrailMap`'s pattern).
  ml.StyleController? _pendingStyle;

  /// The last resolved style, cached so [ref.listen] can push segment
  /// updates without waiting for another `onStyleLoaded` call.
  ml.StyleController? _resolvedStyle;

  /// Segment keys a retry was just dispatched for. Consumed by the
  /// blocked-segment listener to distinguish a first-time failure from a
  /// second consecutive failed retry.
  final Set<String> _retryAttempted = {};

  /// Segment keys currently showing the blocked-segment error toast, so it
  /// fires once per failure rather than once per state emission.
  final Set<String> _blockedNotified = {};

  /// Re-entrancy guard for [_onFinish]. Instance field so it resets
  /// correctly per screen mount rather than leaking across remounts.
  bool _finishing = false;

  /// True once the post-frame session reset has run. Riverpod forbids
  /// modifying a provider synchronously inside `initState`, so `build()`
  /// gates on this flag until the reset lands via `addPostFrameCallback` —
  /// avoids ever rendering the prior session's stale anchors for a frame, at
  /// the cost of one blank frame on mount.
  bool _sessionReady = false;

  /// Edit mode is inferred from a non-empty [RoutePlannerScreen.seedAnchors]
  /// — no separate boolean/mode flag needed.
  bool get _editMode =>
      widget.seedAnchors != null && widget.seedAnchors!.isNotEmpty;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      if (_editMode) {
        ref
            .read(routeAnchorsProvider.notifier)
            .seedFromTrack(
              widget.seedAnchors!,
              widget.travelProfile,
              widget.initialCostingOptions,
              segmentPolylines: widget.seedSegmentPolylines,
            );
      } else {
        ref
            .read(routeAnchorsProvider.notifier)
            .resetForSession(
              widget.travelProfile,
              widget.initialCostingOptions,
            );
      }
      setState(() => _sessionReady = true);
    });
  }

  @override
  Widget build(BuildContext context) {
    if (!_sessionReady) {
      // Pre-reset frame: render nothing so the prior session's state is
      // never painted.
      return const Scaffold(body: SizedBox.shrink());
    }

    final l10n = AppLocalizations.of(context)!;
    final state = ref.watch(routeAnchorsProvider);

    ref.listen(routeAnchorsProvider, (prev, next) {
      final style = _resolvedStyle;
      if (style != null && !identical(prev?.segments, next.segments)) {
        // Keeps the native layer in sync via updateGeoJsonSource — no
        // remove/re-add flicker.
        _segmentLayer.update(style, next.segments).ignore();
      }

      for (final segment in next.segments) {
        final key = segmentKey(segment.beforeAnchorId, segment.afterAnchorId);
        if (segment.state == SegmentState.blocked) {
          if (_blockedNotified.contains(key)) continue;
          _blockedNotified.add(key);
          // A retry in flight for this segment means this is a second
          // consecutive failure — distinct copy.
          final isRetryFailure = _retryAttempted.remove(key);
          ref
              .read(toastProvider.notifier)
              .add(
                ToastMessage(
                  type: ToastType.error,
                  icon: FontAwesomeIcons.triangleExclamation,
                  text: isRetryFailure
                      ? "Still couldn't find a route here. Check your connection and try again."
                      : "Couldn't find a route for this segment. Tap the dashed line to retry.",
                ),
              );
        } else {
          // A later failure on this segment shows the first-time copy
          // again, not a stale "still couldn't find" message.
          _blockedNotified.remove(key);
        }
      }
    });

    final styleJson = ref.watch(mapStyleJsonProvider).value;

    return Scaffold(
      extendBodyBehindAppBar: true,
      appBar: AppBar(
        leading: IconButton(
          icon: const FaIcon(FontAwesomeIcons.arrowLeft, size: 18),
          onPressed: () => context.canPop()
              ? context.pop()
              : context.pushReplacement('/map'),
          style: IconButton.styleFrom(
            backgroundColor: Theme.of(context).colorScheme.surface,
          ),
        ),
        titleSpacing: 8,
        title: _buildSearchBar(context),
        actions: [_buildFinishAction(state, l10n)],
        backgroundColor: Colors.transparent,
        shadowColor: Colors.transparent,
        elevation: 0,
        scrolledUnderElevation: 0,
      ),
      body: Stack(
        children: [
          Positioned.fill(child: _buildMap(context, state, styleJson)),
          // The tabbed sheet mounts only once the route has >=1 anchor, and
          // unmounts entirely when it returns to empty — no empty state.
          if (state.anchors.isNotEmpty) RouteAnchorSheet(),
        ],
      ),
    );
  }

  /// Frames the whole seeded route on open in edit mode.
  ///
  /// [ml.MapOptions.initCenter]/`initZoom` alone can't do this: the caller
  /// only has a single point (the trail's bbox centroid) to hand over, so a
  /// long route would open zoomed into its middle with both ends off-screen.
  /// No-ops for a fresh session, where `initCenter` is already correct.
  ///
  /// Fits the seeded *polylines* rather than the anchors, since a leg can
  /// bulge well outside the box its two anchors describe.
  Future<void> _fitInitialCamera() async {
    final controller = _mapController;
    if (controller == null || !_editMode) return;

    final points = <ml.Geographic>[
      for (final polyline
          in widget.seedSegmentPolylines ?? const <List<ml.Geographic>>[])
        ...polyline,
      // A seeded session with no polylines still has anchors to frame.
      ...?widget.seedAnchors,
    ];
    if (points.isEmpty) return;

    final bounds = ml.LngLatBounds.fromPoints(points);
    final hasExtent =
        bounds.latitudeNorth != bounds.latitudeSouth ||
        bounds.longitudeEast != bounds.longitudeWest;
    if (!hasExtent) return; // initCenter/initZoom already frame a single point.

    // The map fills the whole screen (extendBodyBehindAppBar), so inset for
    // the two things floating over it: the transparent app bar up top and
    // RouteAnchorSheet's 164px docked peek at the bottom. Read before the
    // await, while the frame's MediaQuery is still current.
    final padding = EdgeInsets.fromLTRB(
      32,
      MediaQuery.paddingOf(context).top + kToolbarHeight + 16,
      32,
      164 + 16,
    );

    // Duration.zero is avoided: the Android binding passes it to
    // `animateCamera` as null, which throws. Mirrors TrailMap's fit.
    await controller.fitBounds(
      bounds: bounds,
      padding: padding,
      nativeDuration: const Duration(milliseconds: 1),
    );
  }

  Widget _buildMap(
    BuildContext context,
    RouteAnchorsState state,
    String? styleJson,
  ) {
    if (styleJson == null) {
      return ColoredBox(color: Theme.of(context).colorScheme.surface);
    }

    return ml.MapLibreMap(
      options: ml.MapOptions(
        initStyle: styleJson,
        initCenter: widget.initialCenter,
        initZoom: widget.initialCenter.lat == 0 && widget.initialCenter.lon == 0
            ? 2
            : 14,
        gestures: const ml.MapGestures.all(),
        androidForegroundLoadColor: Theme.of(context).colorScheme.surface,
        // See TrailMap for why texture mode is off and `hc` is pinned.
        androidTextureMode: false,
        androidMode: ml.AndroidPlatformViewMode.hc,
      ),
      onMapCreated: (controller) {
        _mapController = controller;
        _fitInitialCamera().ignore();
        final pending = _pendingStyle;
        if (pending != null) {
          _pendingStyle = null;
          _onStyleLoaded(pending);
        }
      },
      onStyleLoaded: (style) {
        if (_mapController == null) {
          _pendingStyle = style;
          return;
        }
        _onStyleLoaded(style);
      },
      onEvent: (event) {
        if (event is! ml.MapEventClick) return;

        // A marker tap never reaches here — RouteAnchorLayer's own
        // GestureDetector.onTap consumes it first.
        final hits =
            _mapController?.featuresAtPoint(
              event.screenPoint,
              layerIds: const ['route-segments-hit'],
            ) ??
            const [];
        final notifier = ref.read(routeAnchorsProvider.notifier);

        if (hits.isNotEmpty) {
          final props = hits.first.properties;
          final before = props['beforeAnchorId'] as String;
          final after = props['afterAnchorId'] as String;
          final segState = props['state'] as String;

          if (segState == 'blocked') {
            // A blocked segment retries instead of inserting — mutually
            // exclusive by segment state.
            _retryAttempted.add(segmentKey(before, after));
            ref
                .read(toastProvider.notifier)
                .add(
                  ToastMessage(
                    type: ToastType.info,
                    icon: FontAwesomeIcons.route,
                    text: 'Retrying route…',
                  ),
                );
            notifier.retrySegment(before, after);
          } else {
            notifier.insertAnchorOnSegment(before, after, event.point);
          }
          return;
        }

        // Empty-map tap always appends — no add-mode toggle exists.
        notifier.appendAnchor(event.point);
      },
      children: [
        RouteAnchorLayer(),
        // Top-right controls column hosts undo/redo; the search control
        // lives in the app-bar title (see _buildSearchBar) and the
        // auto-routing toggle lives in the Settings tab.
        const WandererMapCompass(
          hideIfRotatedNorth: true,

          padding: EdgeInsets.only(top: 236, right: 4),
        ),
        Positioned(
          top: 112,
          right: 0,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [_buildUndoButton(state), _buildRedoButton(state)],
          ),
        ),
      ],
    );
  }

  /// (Re)runs the style-loaded work: caches the resolved style and adds all
  /// native segment layers for the first time. Buffered/replayed via
  /// [_pendingStyle] if `onStyleLoaded` fires before `onMapCreated`.
  void _onStyleLoaded(ml.StyleController style) {
    _resolvedStyle = style;
    _segmentLayer
        .update(style, ref.read(routeAnchorsProvider).segments)
        .ignore();
  }

  /// Fake search bar hosted in the app-bar title slot — a button styled like
  /// a search field that opens the real [LocationSearchResult]-returning
  /// search screen on tap. Not an inline `TextField`; typing happens on the
  /// dedicated `/location-search` screen.
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
  /// [LocationSearchResult], then pans the map to it at zoom 13 with an
  /// animated camera move.
  Future<void> _openLocationSearch() async {
    final result = await context.push<LocationSearchResult>('/location-search');
    if (!mounted || result == null || _mapController == null) return;

    _mapController!.animateCamera(
      center: ml.Geographic(lat: result.lat, lon: result.lon),
      zoom: 13,
      nativeDuration: const Duration(milliseconds: 750),
    );
  }

  /// Undo pill, positioned in the top-right controls column.
  Widget _buildUndoButton(RouteAnchorsState state) {
    return Padding(
      padding: const EdgeInsets.all(8.0),
      child: Container(
        decoration: BoxDecoration(
          color: Theme.of(context).canvasColor,
          borderRadius: BorderRadius.circular(24),
        ),
        child: IconButton(
          visualDensity: VisualDensity.compact,
          tooltip: AppLocalizations.of(context)!.undo,
          icon: FaIcon(
            FontAwesomeIcons.arrowRotateLeft,
            size: 18,
            color: state.undoStack.isNotEmpty
                ? Theme.of(context).colorScheme.onSurface
                : Theme.of(context).colorScheme.onSurface.withValues(alpha: .4),
          ),
          onPressed: state.undoStack.isNotEmpty
              ? ref.read(routeAnchorsProvider.notifier).undo
              : null,
        ),
      ),
    );
  }

  /// Redo pill, positioned in the top-right controls column.
  Widget _buildRedoButton(RouteAnchorsState state) {
    return Padding(
      padding: const EdgeInsets.all(8.0),
      child: Container(
        decoration: BoxDecoration(
          color: Theme.of(context).canvasColor,
          borderRadius: BorderRadius.circular(24),
        ),
        child: IconButton(
          visualDensity: VisualDensity.compact,
          tooltip: AppLocalizations.of(context)!.redo,
          icon: FaIcon(
            FontAwesomeIcons.arrowRotateRight,
            size: 18,
            color: state.redoStack.isNotEmpty
                ? Theme.of(context).colorScheme.onSurface
                : Theme.of(context).colorScheme.onSurface.withValues(alpha: .4),
          ),
          onPressed: state.redoStack.isNotEmpty
              ? ref.read(routeAnchorsProvider.notifier).redo
              : null,
        ),
      ),
    );
  }

  /// App-bar "Finish planning" action. Gated on >=2 route anchors; disabled
  /// state uses Flutter's standard opacity treatment via `onPressed: null`,
  /// with explanatory copy available on long-press through
  /// [AppLocalizations.finish_disabled_hint] even while disabled — no toast.
  Widget _buildFinishAction(RouteAnchorsState state, AppLocalizations l10n) {
    return IconButton(
      icon: const FaIcon(FontAwesomeIcons.check, size: 18),
      tooltip: state.anchors.length >= 2
          ? l10n.finish
          : l10n.finish_disabled_hint,
      onPressed: (state.anchors.length >= 2 && !_finishing) ? _onFinish : null,
      style: IconButton.styleFrom(
        backgroundColor: Theme.of(context).colorScheme.surface,
        disabledBackgroundColor: Theme.of(context).colorScheme.surface,
      ),
    );
  }

  /// Hands the finished route off: in edit mode pops the ele-merged [Gpx]
  /// back to the awaiting `trail_create_screen`; otherwise invokes
  /// [finishPlanning] to forward-push a draft Trail (built entirely on-device
  /// — see `buildDraftTrail`).
  ///
  /// Neither branch shows a save-options sheet. The planner used to route the
  /// forward-push branch through the shared online gate
  /// (`resolve_track_save_options.dart`), but neither toggle it offered could
  /// change a planned route's output, so it was removed — see
  /// [finishPlanning].
  ///
  /// [_finishing] guards against a double-tap firing two concurrent
  /// conversions/navigations, and the guard is sound only because nothing is
  /// awaited between the `if (_finishing) return;` below and the `setState`
  /// that raises the flag. The removed gate WAS such an await, which is why
  /// this used to need a second post-await re-check. Do not reintroduce an
  /// await ahead of the flag without restoring that re-check — the finish
  /// action stays enabled while the flag is down (`_buildFinishAction` reads
  /// `!_finishing`).
  ///
  /// With the local conversion, an offline finish reaches the
  /// create screen instead of throwing — the `catch` below is not deleted, it
  /// remains a genuine last-resort guard for anything else that could still
  /// fail (e.g. an unexpected local error), matching `import_trail_file.dart`'s
  /// `importTrailFile` precedent for toast-and-stay on failure.
  Future<void> _onFinish() async {
    if (_finishing) return;
    try {
      if (_editMode) {
        setState(() => _finishing = true);
        final finalGpx = await buildFinalPlannedGpx(ref);
        if (!mounted) return;
        context.pop(finalGpx);
      } else {
        setState(() => _finishing = true);
        await finishPlanning(ref: ref, navContext: context);
      }
    } catch (_) {
      if (!mounted) return;
      ref
          .read(toastProvider.notifier)
          .add(
            ToastMessage(
              type: ToastType.error,
              icon: FontAwesomeIcons.circleExclamation,
              text: AppLocalizations.of(context)!.error_saving_trail,
            ),
          );
    } finally {
      if (mounted) setState(() => _finishing = false);
    }
  }
}
