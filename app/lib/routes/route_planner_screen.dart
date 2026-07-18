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
import 'package:wanderer/util/route_planner_handoff_util.dart';
import 'package:wanderer/util/route_segment_util.dart';

/// Route Planner screen — hosts the native map and wires every Phase 19
/// gesture/state interaction to the UI: tap-to-add (WAYP-01), drag-to-
/// reposition (WAYP-02, handled inside [RouteAnchorLayer]), tap-to-insert /
/// tap-to-retry on a segment (WAYP-03/D-09), and the auto-routing toggle
/// (ROUTE-01/02, D-06). This is Phase 19's final integration point — 19-01/
/// 19-02/19-03 built correct, isolated components; this screen makes them
/// reachable by a user.
///
/// Router registration/entry-point wiring (the real [initialCenter]/
/// [travelProfile] values) is HANDOFF-02/03 — explicitly Phase 21's scope,
/// not touched here. The app-bar title slot is a fake search bar (opens
/// [LocationSearchScreen] via `/location-search`), analogous to
/// `map_screen.dart`'s search pill.
class RoutePlannerScreen extends ConsumerStatefulWidget {
  /// `'pedestrian'` or `'bicycle'` — the initial travel profile for this
  /// planning session, set once by the Phase 21 entry-point dialog. Applied
  /// via [RouteAnchors.resetForSession] at mount; may change mid-session via
  /// the Settings tab's [RouteAnchors.switchProfile] (Rec B).
  final String travelProfile;

  /// The initial Valhalla `costing_options` payload matching
  /// [travelProfile]'s selected bucket, threaded through the router's
  /// `extra['costingOptions']` (quick-260717-t7q). `null` when the caller
  /// didn't supply one.
  final Map<String, dynamic>? initialCostingOptions;

  /// The map's initial camera center. Required with no fallback default —
  /// inventing a hardcoded center here would silently mask a caller bug;
  /// Phase 21's entry point supplies the real value (e.g. the user's current
  /// location or the trail-source-select flow's last map camera).
  final ml.Geographic initialCenter;

  /// Segment-boundary anchors seeded from an existing trail's track
  /// (quick-260718-e9j, PLANNER-02) — non-null and non-empty means this
  /// session is editing an existing route rather than planning a new one.
  /// When present, [RouteAnchors.seedFromTrack] replaces the usual
  /// [RouteAnchors.resetForSession] empty-session reset, and Finish pops the
  /// final [Gpx] back to the caller instead of forward-pushing a draft trail
  /// via [finishPlanning].
  final List<ml.Geographic>? seedAnchors;

  /// Per-segment full-resolution polylines aligned with [seedAnchors]
  /// (quick-260718-e9j follow-up, from `segmentPolylinesFromTrack`) — the
  /// original recorded points between each anchor pair, so the map shows the
  /// exact prior route rather than a straight line or a Valhalla-recomputed
  /// one until the user actually edits a segment. `null`/short falls back to
  /// a straight line for the affected pair(s) in [RouteAnchors.seedFromTrack].
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
  // Instance field, NOT static: RouteSegmentLayer tracks whether its
  // source/layers have been added to the CURRENT native style via its own
  // `_added` flag. A `static` field would share that flag (and go stale)
  // across every mount of this screen for the app's whole lifetime — the
  // second time this screen opens, the native map/style is a fresh
  // instance with no source/layers on it yet, but a stale `_added == true`
  // would make `update()` skip straight to `updateGeoJsonSource` on a
  // source that was never added to this style, silently failing (caught by
  // the call site's `.ignore()`) and leaving segments permanently
  // unrendered. A fresh instance per screen mount keeps `_added` accurate.
  final _segmentLayer = RouteSegmentLayer();

  ml.MapController? _mapController;

  /// Buffers a style-loaded event that arrives before [_mapController] is
  /// set — copied verbatim from `TrailMap`'s pattern (Pitfall 5).
  ml.StyleController? _pendingStyle;

  /// The last resolved style. Cached so [ref.listen] can push segment
  /// updates to the native layer without waiting for another
  /// `onStyleLoaded` call.
  ml.StyleController? _resolvedStyle;

  /// Segment keys a retry was just dispatched for, populated by the
  /// blocked-segment tap branch below. Consumed by the blocked-segment
  /// listener to distinguish a first-time failure from a second consecutive
  /// failed retry (UI-SPEC's Copywriting Contract).
  final Set<String> _retryAttempted = {};

  /// Segment keys currently showing the first-time blocked-segment error
  /// toast — bookkeeping so the toast fires exactly once per failure, not
  /// once per state emission while the segment stays blocked.
  final Set<String> _blockedNotified = {};

  /// Re-entrancy guard for [_onFinish] (T-21-04-01): a `static` field would
  /// share the in-flight flag across every mount of this screen; an
  /// instance field resets correctly per Phase 19's own established
  /// non-static-field discipline (Pitfall 3).
  bool _finishing = false;

  /// True once [_resetSessionAndRebuild]'s post-frame reset has run. Riverpod
  /// forbids modifying a provider synchronously inside `initState` (it fires
  /// "Tried to modify a provider while the widget tree was building" —
  /// confirmed on-device) since Rec B made `routeAnchorsProvider` a single
  /// `keepAlive` instance with no family key, so a re-entry must reset it
  /// via `addPostFrameCallback` instead. Gating `build()` on this flag until
  /// the reset lands avoids ever rendering the PRIOR session's stale
  /// anchors/segments for a frame (T-t7q-03), at the cost of one blank frame
  /// on mount instead — imperceptible against the map's own async style load.
  bool _sessionReady = false;

  /// quick-260718-e9j (PLANNER-02): edit mode is inferred from a non-empty
  /// [RoutePlannerScreen.seedAnchors] — no separate boolean/mode flag needed.
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
      // Pre-reset frame — render nothing observing routeAnchorsProvider so
      // the prior session's state (if any) is never painted (T-t7q-03).
      return const Scaffold(body: SizedBox.shrink());
    }

    final l10n = AppLocalizations.of(context)!;
    final state = ref.watch(routeAnchorsProvider);

    ref.listen(routeAnchorsProvider, (prev, next) {
      final style = _resolvedStyle;
      if (style != null && !identical(prev?.segments, next.segments)) {
        // Keeps the native segment layer's GeoJSON in sync with every
        // mutation via updateGeoJsonSource — no remove/re-add flicker.
        _segmentLayer.update(style, next.segments).ignore();
      }

      for (final segment in next.segments) {
        final key = segmentKey(segment.beforeAnchorId, segment.afterAnchorId);
        if (segment.state == SegmentState.blocked) {
          if (_blockedNotified.contains(key)) continue;
          _blockedNotified.add(key);
          // If a retry was in flight for this exact segment, this is a
          // second consecutive failure — distinct copy (UI-SPEC).
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
          // A future failure on this same segment pair shows the first-time
          // copy again, not a stale "still couldn't find" from a prior,
          // already-resolved failure.
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
          // D-03: the tabbed sheet mounts only once the route has >=1
          // anchor, and un-mounts entirely when it returns to empty — it
          // never shows an empty state of its own.
          if (state.anchors.isNotEmpty) RouteAnchorSheet(),
        ],
      ),
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
      ),
      onMapCreated: (controller) {
        _mapController = controller;
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
        // GestureDetector.onTap consumes it first (D-04: marker wins on
        // overlap).
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
            // D-09: a blocked segment retries instead of inserting — mutually
            // exclusive by segment state, never a competing gesture.
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

        // Empty-map tap always appends (D-03) — no add-mode toggle exists.
        notifier.appendAnchor(event.point);
      },
      children: [
        RouteAnchorLayer(),
        // D-06: shared top-right controls slot. D-02 replaced the planned
        // list/elevation toggle buttons with the tabbed RouteAnchorSheet; the
        // search control (D-04, 20-CONTEXT) has since moved into the app-bar
        // title as a fake search bar (see _buildSearchBar). Undo/redo now
        // live here too (D-04, 21-CONTEXT). The auto-routing toggle has
        // moved into the Settings tab (quick-260717-t7q) — this column now
        // hosts only undo/redo.
        const ml.MapCompass(
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

  /// (Re)runs the style-loaded work: caches the resolved style and does the
  /// first-time add of all 5 native segment layers. Buffered/replayed via
  /// [_pendingStyle] when the native platform channel fires `onStyleLoaded`
  /// before `onMapCreated` (Pitfall 5).
  void _onStyleLoaded(ml.StyleController style) {
    _resolvedStyle = style;
    _segmentLayer
        .update(style, ref.read(routeAnchorsProvider).segments)
        .ignore();
  }

  /// D-04: fake search bar hosted in the app-bar title slot — a button
  /// styled like a search field (analogous to `map_screen.dart`'s search
  /// pill) that opens the real [LocationSearchResult]-returning search
  /// screen on tap. Not an inline `TextField`; typing happens on the
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
                  'Search location',
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

  /// D-14/D-15: pushes the location-search screen, awaits the selected
  /// [LocationSearchResult], then pans the map to it at a fixed zoom of 13
  /// with an animated (not instant) camera move.
  Future<void> _openLocationSearch() async {
    final result = await context.push<LocationSearchResult>('/location-search');
    if (!mounted || result == null || _mapController == null) return;

    _mapController!.animateCamera(
      center: ml.Geographic(lat: result.lat, lon: result.lon),
      zoom: 13,
      nativeDuration: const Duration(milliseconds: 750),
    );
  }

  /// D-04: undo pill, relocated from the app bar into the top-right controls
  /// Column. Same pill container/icon treatment the (now-removed)
  /// auto-routing toggle used — only position changed, not styling.
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
          tooltip: 'Undo',
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

  /// D-04: redo pill, relocated from the app bar into the top-right controls
  /// Column. Same pill container/icon treatment [_buildUndoButton] uses —
  /// only position changed, not styling.
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
          tooltip: 'Redo',
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

  /// D-04/D-05: app-bar "Finish planning" action — takes the app-bar slot
  /// undo/redo previously occupied. Gated on >=2 route anchors (D-05); the
  /// disabled state uses Flutter's standard ~38%-opacity treatment via
  /// `onPressed: null`, with the explanatory copy available on long-press
  /// through [AppLocalizations.finish_disabled_hint] even while disabled —
  /// no toast/snackbar (UI-SPEC).
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

  /// HANDOFF-01 (import path): invokes [finishPlanning] to hand the finished
  /// route off as a draft Trail via forward-push. quick-260718-e9j
  /// (PLANNER-02, edit path): pops the ele-merged [Gpx] straight back to the
  /// awaiting `trail_create_screen` instead. Guarded by [_finishing]
  /// (T-21-04-01) so a double-tap cannot fire two `/valhalla/height` fetches
  /// or two navigations while a handoff is already in flight.
  Future<void> _onFinish() async {
    if (_finishing) return;
    setState(() => _finishing = true);
    try {
      if (_editMode) {
        final finalGpx = await buildFinalPlannedGpx(ref);
        if (!mounted) return;
        context.pop(finalGpx);
      } else {
        await finishPlanning(ref: ref, navContext: context);
      }
    } finally {
      if (mounted) setState(() => _finishing = false);
    }
  }
}
