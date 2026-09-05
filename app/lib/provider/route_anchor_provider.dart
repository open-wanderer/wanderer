import 'package:collection/collection.dart';
import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart' show UniqueKey;
import 'package:maplibre/maplibre.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/route_anchor.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/util/gpx/gpx.dart';
import 'package:wanderer/util/geo/polyline.dart';
import 'package:wanderer/util/geo/reverse_geocode.dart';
import 'package:wanderer/util/route/segment.dart';
import 'package:wanderer/util/route/valhalla.dart';

part 'route_anchor_provider.g.dart';

class RouteAnchorsState {
  const RouteAnchorsState({
    required this.anchors,
    required this.segments,
    required this.autoRoutingEnabled,
    required this.travelProfile,
    required this.costingOptions,
    required this.undoStack,
    required this.redoStack,
  });

  final List<RouteAnchor> anchors;
  final List<RouteSegment> segments;
  final bool autoRoutingEnabled;

  /// `'pedestrian'` or `'bicycle'`; switched via [RouteAnchors.switchProfile]
  /// or [RouteAnchors.resetForSession] rather than fixed at construction.
  final String travelProfile;

  /// Valhalla `costing_options` payload for the current [travelProfile].
  /// `null` until a bucket has been applied; set atomically with
  /// [travelProfile].
  final Map<String, dynamic>? costingOptions;
  final List<RouteAnchorsSnapshot> undoStack;
  final List<RouteAnchorsSnapshot> redoStack;

  RouteAnchorsState copyWith({
    List<RouteAnchor>? anchors,
    List<RouteSegment>? segments,
    bool? autoRoutingEnabled,
    String? travelProfile,
    Map<String, dynamic>? costingOptions,
    List<RouteAnchorsSnapshot>? undoStack,
    List<RouteAnchorsSnapshot>? redoStack,
  }) {
    return RouteAnchorsState(
      anchors: anchors ?? this.anchors,
      segments: segments ?? this.segments,
      autoRoutingEnabled: autoRoutingEnabled ?? this.autoRoutingEnabled,
      travelProfile: travelProfile ?? this.travelProfile,
      costingOptions: costingOptions ?? this.costingOptions,
      undoStack: undoStack ?? this.undoStack,
      redoStack: redoStack ?? this.redoStack,
    );
  }

  /// The route's segments in traversal order, walking the anchor chain from
  /// `anchors.first` via each segment's `beforeAnchorId -> afterAnchorId`
  /// link rather than [segments] array order — so a detached/orphaned
  /// segment never contributes, and the order survives
  /// renumber/insert/delete/reorder operations.
  ///
  /// Empty when there are no anchors. The single source of this walk: every
  /// consumer that needs legs start-to-finish (GPX export, elevation GPX,
  /// cumulative stats) goes through here.
  List<RouteSegment> get orderedSegments {
    if (anchors.isEmpty) return const [];

    final segByBefore = {for (final s in segments) s.beforeAnchorId: s};
    final ordered = <RouteSegment>[];
    var currentId = anchors.first.id;

    while (segByBefore.containsKey(currentId)) {
      final seg = segByBefore[currentId]!;
      ordered.add(seg);
      currentId = seg.afterAnchorId;
    }

    return ordered;
  }

  /// Total estimated travel time across every segment, in seconds: uses each
  /// routed segment's Valhalla `durationSeconds`, falling back to a
  /// distance/speed estimate for any segment that never resolved one.
  double get estimatedDurationSeconds {
    var total = 0.0;
    for (final segment in segments) {
      total +=
          segment.durationSeconds ??
          estimateSegmentDurationSeconds(
            polyline: segment.polyline,
            travelProfile: travelProfile,
            costingOptions: costingOptions,
          );
    }
    return total;
  }

  /// Per-anchor cumulative distance/duration/elevation-gain, keyed by anchor
  /// id (stable across reorder, unlike array index). Walks [orderedSegments],
  /// so a detached/orphaned segment never contributes.
  ///
  /// The first anchor always maps to all-zero stats. Elevation gain reads
  /// `0` until that segment's fire-and-forget height fetch resolves.
  Map<String, AnchorRouteStats> get cumulativeStatsByAnchorId {
    if (anchors.isEmpty) return {};

    final result = <String, AnchorRouteStats>{
      anchors.first.id: const AnchorRouteStats(
        distanceMeters: 0,
        durationSeconds: 0,
        elevationGainMeters: 0,
      ),
    };

    var distance = 0.0;
    var duration = 0.0;
    var elevationGain = 0.0;

    for (final seg in orderedSegments) {
      distance += seg.distanceMeters;
      duration +=
          seg.durationSeconds ??
          estimateSegmentDurationSeconds(
            polyline: seg.polyline,
            travelProfile: travelProfile,
            costingOptions: costingOptions,
          );
      elevationGain += seg.elevationGainMeters;

      result[seg.afterAnchorId] = AnchorRouteStats(
        distanceMeters: distance,
        durationSeconds: duration,
        elevationGainMeters: elevationGain,
      );
    }

    return result;
  }
}

/// A single anchor's running totals through the route up to (and including)
/// it — see [RouteAnchorsState.cumulativeStatsByAnchorId].
class AnchorRouteStats {
  const AnchorRouteStats({
    required this.distanceMeters,
    required this.durationSeconds,
    required this.elevationGainMeters,
  });

  final double distanceMeters;
  final double durationSeconds;
  final double elevationGainMeters;
}

/// Route-planner state provider: owns the ordered anchor list, the
/// per-segment Valhalla routing engine (with a race-guard against
/// out-of-order responses), the geometric segment-split used by a plain
/// insert tap, and an immutable-snapshot undo/redo stack.
///
/// A single `@Riverpod(keepAlive: true)` provider with no family argument —
/// `travelProfile`/`costingOptions` live in state and are switched via
/// [switchProfile] (mid-session) or [resetForSession] (planner entry, since
/// `keepAlive` means this instance survives across mounts and must be reset
/// so re-entry never leaks the previous session's route).
@Riverpod(keepAlive: true)
class RouteAnchors extends _$RouteAnchors {
  // Non-reactive bookkeeping, never stored in `state`. Keyed by the stable
  // `segmentKey` (anchor-id pair), never array index.
  final Map<String, CancelToken> _inFlight = {};
  final Map<String, int> _generation = {};

  // Same cancel/generation bookkeeping, keyed by anchor id for per-anchor
  // reverse-geocode requests — each anchor's location search is independent,
  // so a drag on one anchor never blocks/queues behind another's.
  final Map<String, CancelToken> _locationInFlight = {};
  final Map<String, int> _locationGeneration = {};

  // Same cancel/generation bookkeeping, but for each segment's
  // fire-and-forget `/valhalla/height` fetch — a rapid re-edit of the same
  // segment supersedes rather than races its own prior elevation fetch.
  final Map<String, CancelToken> _elevationInFlight = {};
  final Map<String, int> _elevationGeneration = {};

  @override
  RouteAnchorsState build() {
    return const RouteAnchorsState(
      anchors: [],
      segments: [],
      autoRoutingEnabled: true,
      travelProfile: 'pedestrian',
      costingOptions: null,
      undoStack: [],
      redoStack: [],
    );
  }

  bool _isValidCoordinate(RouteAnchor anchor) {
    return anchor.lat >= -90 &&
        anchor.lat <= 90 &&
        anchor.lon >= -180 &&
        anchor.lon <= 180;
  }

  /// Resolves the segment between [a] and [b] via Valhalla, guarded by a
  /// per-segment `CancelToken` + monotonically increasing generation
  /// counter so an out-of-order (superseded) response is discarded rather
  /// than applied over a newer request's result.
  ///
  /// On failure, the segment is marked [SegmentState.blocked] and its prior
  /// polyline is left untouched — never silently falls back to a straight
  /// line while auto-routing is on.
  Future<void> _resolveSegment(
    String beforeAnchorId,
    String afterAnchorId,
    RouteAnchor a,
    RouteAnchor b,
  ) async {
    final key = segmentKey(beforeAnchorId, afterAnchorId);

    if (!_isValidCoordinate(a) || !_isValidCoordinate(b)) {
      // Reject out-of-range coordinates before ever calling Valhalla.
      _markBlocked(key);
      return;
    }

    _inFlight[key]?.cancel();
    final token = CancelToken();
    _inFlight[key] = token;
    final myGeneration = (_generation[key] ?? 0) + 1;
    _generation[key] = myGeneration;

    try {
      final response = await ref
          .read(apiProvider)
          .post(
            '/valhalla/route',
            data: {
              'directions_type': 'none',
              'locations': [
                {'lat': a.lat, 'lon': a.lon},
                {'lat': b.lat, 'lon': b.lon},
              ],
              'costing': state.travelProfile,
              if (state.costingOptions != null)
                'costing_options': {state.travelProfile: state.costingOptions},
            },
            cancelToken: token,
          );

      // Stale-response guard: a newer request may already have started.
      if (_generation[key] != myGeneration) return;

      final trip = response.data is Map ? response.data['trip'] : null;
      final legs = trip is Map ? trip['legs'] : null;
      final leg = (legs is List && legs.isNotEmpty) ? legs[0] : null;
      final shape = leg is Map ? leg['shape'] : null;

      if (shape is! String) {
        _markBlocked(key);
        return;
      }

      final summary = trip is Map ? trip['summary'] : null;
      final durationSeconds = summary is Map
          ? (summary['time'] as num?)?.toDouble()
          : null;

      final points = PolylineUtil.decode(shape, precision: 6);
      _applySegment(
        key,
        points: points,
        segmentState: SegmentState.routed,
        durationSeconds: durationSeconds,
      );
    } on DioException catch (e) {
      if (e.type == DioExceptionType.cancel) {
        return; // superseded, not a real failure
      }
      if (_generation[key] != myGeneration) return;
      _markBlocked(key); // never revert to straight, never clear
    }
  }

  void _markBlocked(String key) {
    final segments = [
      for (final segment in state.segments)
        if (segmentKey(segment.beforeAnchorId, segment.afterAnchorId) == key)
          segment.copyWith(state: SegmentState.blocked)
        else
          segment,
    ];
    state = state.copyWith(segments: segments);
  }

  /// Replaces [key]'s geometry with [points] and re-fetches its heights.
  ///
  /// Clearing `elevationProfile`/`elevations` is mandatory, not tidiness.
  /// They describe the geometry being replaced, and `_resolveElevation`
  /// below is fire-and-forget, so leaving them in place publishes a segment
  /// whose profile belongs to its *previous* shape. Both consumers read
  /// `elevationProfile ?? polyline` — `buildFinalPlannedGpx`
  /// (util/route/planner_handoff.dart) and `plannedElevationGpx`
  /// (planned_gpx_provider.dart) — so a stale profile silently wins over the
  /// polyline it contradicts: tapping Finish inside that window saved the
  /// pre-edit route, and the elevation preview drew it while the map drew
  /// the new one. Worse than a race: `_resolveElevation` degrades silently
  /// on failure, which leaves a re-edited segment stale permanently rather
  /// than merely briefly.
  ///
  /// Nulling them costs nothing — every caller already triggers a refetch —
  /// and degrades correctly: the `?? polyline` fallback yields the new
  /// geometry at full resolution, and `buildFinalPlannedGpx`'s `pending`
  /// list picks up `elevations == null` and backfills heights at save time.
  void _applySegment(
    String key, {
    required List<Geographic> points,
    required SegmentState segmentState,
    double? durationSeconds,
  }) {
    final segments = [
      for (final segment in state.segments)
        if (segmentKey(segment.beforeAnchorId, segment.afterAnchorId) == key)
          segment.copyWith(
            polyline: points,
            state: segmentState,
            durationSeconds: durationSeconds,
            elevationProfile: null,
            elevations: null,
          )
        else
          segment,
    ];
    state = state.copyWith(segments: segments);
    _resolveElevation(key, points).ignore();
  }

  /// Fetches `/valhalla/height` for [polyline] (downsampled via
  /// `buildNavShape`) and writes the result onto the segment identified by
  /// [key] as its [RouteSegment.elevationProfile]/[RouteSegment.elevations]
  /// pair.
  ///
  /// Fire-and-forget (`.ignore()` at every call site): a segment's polyline
  /// renders on the map the instant it's known, never blocked on this fetch.
  /// Same stale-response discipline as [_resolveSegment] (per-key
  /// `CancelToken` + generation counter); the `for` rewrite is naturally a
  /// no-op if [key] no longer names a live segment.
  Future<void> _resolveElevation(String key, List<Geographic> polyline) async {
    if (polyline.length < 2) return;

    _elevationInFlight[key]?.cancel();
    final token = CancelToken();
    _elevationInFlight[key] = token;
    final myGeneration = (_elevationGeneration[key] ?? 0) + 1;
    _elevationGeneration[key] = myGeneration;

    final shape = buildNavShape(polyline);

    try {
      final response = await ref
          .read(apiProvider)
          .post('/valhalla/height', data: {'shape': shape}, cancelToken: token);

      if (_elevationGeneration[key] != myGeneration) return;

      final heights = (response.data['height'] as List).cast<num>();
      final profile = [
        for (final entry in shape)
          Geographic(lat: entry['lat']!, lon: entry['lon']!),
      ];
      final elevations = [for (final h in heights) h.toDouble()];

      final segments = [
        for (final segment in state.segments)
          if (segmentKey(segment.beforeAnchorId, segment.afterAnchorId) == key)
            segment.copyWith(elevationProfile: profile, elevations: elevations)
          else
            segment,
      ];
      state = state.copyWith(segments: segments);
    } on DioException catch (e) {
      if (e.type == DioExceptionType.cancel) return; // superseded
      // Best-effort, mirrors _resolveAnchorLocation: leave elevation unset
      // on a genuine failure rather than surfacing an error.
    } catch (_) {
      // Non-Dio failure (malformed response) — same silent degrade.
    }
  }

  /// Reverse-geocodes [anchorId]'s current coordinates and, on success,
  /// writes the result onto that anchor's [RouteAnchor.location] field.
  /// Fired once from [appendAnchor]/[insertAnchorOnSegment] (new anchor) and
  /// once at drag-end from [dragAnchor] — never continuously during a drag
  /// gesture.
  ///
  /// Best-effort and silent: on failure (or supersession by a newer request
  /// for the same anchor) the anchor's prior `location` is left untouched
  /// rather than cleared — a dragged anchor keeps showing its last-known
  /// label until the fresh search resolves.
  Future<void> _resolveAnchorLocation(
    String anchorId,
    RouteAnchor anchor,
  ) async {
    _locationInFlight[anchorId]?.cancel();
    final token = CancelToken();
    _locationInFlight[anchorId] = token;
    final myGeneration = (_locationGeneration[anchorId] ?? 0) + 1;
    _locationGeneration[anchorId] = myGeneration;

    try {
      final result = await searchLocationReverseStructured(
        ref.read(apiProvider),
        anchor.lat,
        anchor.lon,
        includeRoad: true,
        cancelToken: token,
      );

      // Stale-response guard, same shape as _resolveSegment: a newer search
      // for this anchor may have started, or the anchor may be deleted.
      if (_locationGeneration[anchorId] != myGeneration) return;
      if (result == null) return;

      final anchors = [
        for (final a in state.anchors)
          if (a.id == anchorId) a.copyWith(location: result) else a,
      ];
      state = state.copyWith(anchors: anchors);
    } on DioException catch (e) {
      // Cancelled (superseded) or a genuine network/parse failure — both
      // degrade silently, leaving the anchor's prior location untouched.
      if (e.type == DioExceptionType.cancel) return;
    } catch (_) {
      // Non-Dio failure (e.g. malformed response) — same silent degrade.
    }
  }

  /// Retries the blocked segment for the given anchor pair by re-dispatching
  /// [_resolveSegment].
  ///
  /// The native map's hit-test layer syncs asynchronously, so a tap can
  /// carry anchor ids that no longer exist in `state` (an undo/delete/
  /// reorder raced ahead of it). `firstWhereOrNull` degrades that stale hit
  /// to a silent no-op instead of an uncaught `StateError`.
  Future<void> retrySegment(String beforeAnchorId, String afterAnchorId) {
    final a = state.anchors.firstWhereOrNull((x) => x.id == beforeAnchorId);
    final b = state.anchors.firstWhereOrNull((x) => x.id == afterAnchorId);
    if (a == null || b == null) return Future.value();
    return _resolveSegment(beforeAnchorId, afterAnchorId, a, b);
  }

  /// Flips [RouteAnchorsState.autoRoutingEnabled]. Turning OFF leaves every
  /// existing segment untouched (only new segments become straight).
  Future<void> toggleAutoRouting() async {
    final enabled = !state.autoRoutingEnabled;
    state = state.copyWith(autoRoutingEnabled: enabled);
  }

  /// Switches the travel bucket mid-session: sets [profile] + [opts]
  /// atomically and re-resolves every existing segment under the new
  /// costing (including a within-`bicycle` sub-type switch, e.g. Hybrid ->
  /// Road).
  ///
  /// A profile switch is a fresh undo baseline — both stacks are cleared and
  /// no undo snapshot is pushed, since the switch itself is not undoable.
  void switchProfile(String profile, Map<String, dynamic> opts) {
    state = state.copyWith(
      travelProfile: profile,
      costingOptions: opts,
      undoStack: const [],
      redoStack: const [],
    );
    resolveAllSegments();
  }

  /// Bulk re-resolve: iterates every consecutive anchor pair with an
  /// existing segment and either re-dispatches a Valhalla resolve
  /// (auto-routing on, fire-and-forget) or applies a straight two-point
  /// polyline (auto-routing off). Used by [switchProfile] to re-resolve the
  /// whole route, not just newly-adjacent pairs.
  void resolveAllSegments() {
    final anchorsById = {for (final a in state.anchors) a.id: a};
    final segByKey = {
      for (final s in state.segments)
        segmentKey(s.beforeAnchorId, s.afterAnchorId): s,
    };

    for (var i = 0; i < state.anchors.length - 1; i++) {
      final a = state.anchors[i];
      final b = state.anchors[i + 1];
      final key = segmentKey(a.id, b.id);
      if (!segByKey.containsKey(key)) continue;

      if (state.autoRoutingEnabled) {
        _resolveSegment(a.id, b.id, a, b).ignore();
      } else {
        final aa = anchorsById[a.id]!;
        final bb = anchorsById[b.id]!;
        _applySegment(
          key,
          points: [aa.point, bb.point],
          segmentState: SegmentState.straight,
        );
      }
    }
  }

  /// Resets the single keepAlive instance to a brand-new empty session.
  /// Called once at planner-screen mount so a re-entry never leaks the
  /// previous session's route. Cancels every in-flight request first.
  void resetForSession(String profile, Map<String, dynamic>? opts) {
    for (final token in _inFlight.values) {
      token.cancel();
    }
    _inFlight.clear();
    _generation.clear();
    for (final token in _locationInFlight.values) {
      token.cancel();
    }
    _locationInFlight.clear();
    _locationGeneration.clear();
    for (final token in _elevationInFlight.values) {
      token.cancel();
    }
    _elevationInFlight.clear();
    _elevationGeneration.clear();

    state = RouteAnchorsState(
      anchors: const [],
      segments: const [],
      autoRoutingEnabled: true,
      travelProfile: profile,
      costingOptions: opts,
      undoStack: const [],
      redoStack: const [],
    );
  }

  /// Seeds the single keepAlive instance from an existing track's
  /// prepopulated anchors — the "edit an existing route" entry mode, as
  /// opposed to [resetForSession]'s empty-session entry for a new/imported
  /// route.
  ///
  /// Cancels + clears all in-flight/generation bookkeeping like
  /// [resetForSession]. Builds one [RouteAnchor] per point (fresh
  /// [UniqueKey] ids) and one [RouteSegment] per consecutive pair, with a
  /// fresh (empty) undo/redo baseline — the seed itself is not undoable.
  ///
  /// [segmentPolylines], when given, supplies each segment's full original
  /// recorded points; a missing/short entry falls back to a straight
  /// 2-point line. Every seeded segment is marked [SegmentState.straight]
  /// regardless of point count — the enum means "not Valhalla-routed", not
  /// "exactly 2 points".
  ///
  /// Deliberately does NOT loop [appendAnchor] (would push an undo snapshot
  /// per anchor), does NOT reverse-geocode any seeded anchor, and does NOT
  /// call [resolveAllSegments] — auto-resolving on open would silently snap
  /// an off-road recording onto nearby roads before the user touched
  /// anything; a segment is only Valhalla-resolved once the user edits it.
  /// It DOES still fire each segment's [_resolveElevation] fetch, since
  /// that's a read-only height lookup rather than a routing decision.
  void seedFromTrack(
    List<Geographic> points,
    String profile,
    Map<String, dynamic>? opts, {
    List<List<Geographic>>? segmentPolylines,
  }) {
    for (final token in _inFlight.values) {
      token.cancel();
    }
    _inFlight.clear();
    _generation.clear();
    for (final token in _locationInFlight.values) {
      token.cancel();
    }
    _locationInFlight.clear();
    _locationGeneration.clear();
    for (final token in _elevationInFlight.values) {
      token.cancel();
    }
    _elevationInFlight.clear();
    _elevationGeneration.clear();

    final anchors = [
      for (final p in points)
        RouteAnchor(id: UniqueKey().toString(), lat: p.lat, lon: p.lon),
    ];
    final segments = [
      for (var i = 0; i < anchors.length - 1; i++)
        RouteSegment(
          beforeAnchorId: anchors[i].id,
          afterAnchorId: anchors[i + 1].id,
          polyline: (segmentPolylines != null && i < segmentPolylines.length)
              ? segmentPolylines[i]
              : [anchors[i].point, anchors[i + 1].point],
          state: SegmentState.straight,
        ),
    ];

    state = RouteAnchorsState(
      anchors: anchors,
      segments: segments,
      autoRoutingEnabled: true,
      travelProfile: profile,
      costingOptions: opts,
      undoStack: const [],
      redoStack: const [],
    );

    for (final segment in segments) {
      _resolveElevation(
        segmentKey(segment.beforeAnchorId, segment.afterAnchorId),
        segment.polyline,
      ).ignore();
    }
  }

  /// Depth cap for [_pushUndo]. Every snapshot duplicates ALL segment
  /// polylines + elevation profiles, and this keepAlive provider survives
  /// screen exits — an unbounded stack over a long planning session was a
  /// measurable slice of peak memory. 20 steps is far beyond practical
  /// undo reach.
  static const _kMaxUndoDepth = 20;

  /// Pushes the current (pre-mutation) anchors/segments onto the undo stack
  /// and clears the redo stack, dropping the oldest snapshot beyond
  /// [_kMaxUndoDepth]. Called first, before mutating, by every mutation
  /// method below.
  void _pushUndo() {
    final stack = state.undoStack;
    state = state.copyWith(
      undoStack: [
        if (stack.length >= _kMaxUndoDepth)
          ...stack.sublist(stack.length - _kMaxUndoDepth + 1)
        else
          ...stack,
        RouteAnchorsSnapshot(anchors: state.anchors, segments: state.segments),
      ],
      redoStack: const [],
    );
  }

  /// Appends a new anchor to the end of the route. If a previous last
  /// anchor existed, also creates the connecting segment (straight by
  /// default, auto-resolved via Valhalla if auto-routing is on).
  void appendAnchor(Geographic point) {
    _pushUndo();

    final newAnchor = RouteAnchor(
      id: UniqueKey().toString(),
      lat: point.lat,
      lon: point.lon,
    );
    final previousLast = state.anchors.isNotEmpty ? state.anchors.last : null;

    state = state.copyWith(anchors: [...state.anchors, newAnchor]);
    _resolveAnchorLocation(newAnchor.id, newAnchor).ignore();

    if (previousLast != null) {
      final newSegment = RouteSegment(
        beforeAnchorId: previousLast.id,
        afterAnchorId: newAnchor.id,
        polyline: [previousLast.point, newAnchor.point],
        state: SegmentState.straight,
      );
      state = state.copyWith(segments: [...state.segments, newSegment]);
      _resolveElevation(
        segmentKey(previousLast.id, newAnchor.id),
        newSegment.polyline,
      ).ignore();

      if (state.autoRoutingEnabled) {
        _resolveSegment(
          previousLast.id,
          newAnchor.id,
          previousLast,
          newAnchor,
        ).ignore();
      }
    }
  }

  /// Repositions [anchorId] and re-resolves only its <=2 adjacent segments.
  /// Invoked once per drag gesture at `onPanEnd`, never during
  /// `onPanUpdate`.
  void dragAnchor(String anchorId, Geographic newPoint) {
    _pushUndo();

    final anchors = [
      for (final a in state.anchors)
        if (a.id == anchorId)
          a.copyWith(lat: newPoint.lat, lon: newPoint.lon)
        else
          a,
    ];
    state = state.copyWith(anchors: anchors);

    final anchorsById = {for (final a in anchors) a.id: a};
    // Drag-end only — re-searches this anchor's location against its new
    // coordinates; the stale prior label persists until this resolves.
    _resolveAnchorLocation(anchorId, anchorsById[anchorId]!).ignore();
    final touched = state.segments.where(
      (s) => s.beforeAnchorId == anchorId || s.afterAnchorId == anchorId,
    );

    for (final segment in touched) {
      final a = anchorsById[segment.beforeAnchorId]!;
      final b = anchorsById[segment.afterAnchorId]!;
      if (state.autoRoutingEnabled) {
        _resolveSegment(
          segment.beforeAnchorId,
          segment.afterAnchorId,
          a,
          b,
        ).ignore();
      } else {
        _applySegment(
          segmentKey(segment.beforeAnchorId, segment.afterAnchorId),
          points: [a.point, b.point],
          segmentState: SegmentState.straight,
        );
      }
    }
  }

  /// On a plain (non-blocked) segment tap, geometrically splits it and
  /// inserts the new anchor between its two endpoint anchors — never calls
  /// Valhalla. The new anchor's list position determines its display
  /// number, which is always derived from index, never stored.
  void insertAnchorOnSegment(
    String beforeAnchorId,
    String afterAnchorId,
    Geographic tapPoint,
  ) {
    final key = segmentKey(beforeAnchorId, afterAnchorId);
    // Same stale-hit-test guard as retrySegment. Bail out before _pushUndo()
    // so a miss never pushes a spurious no-op undo snapshot.
    final targetSegment = state.segments.firstWhereOrNull(
      (s) => segmentKey(s.beforeAnchorId, s.afterAnchorId) == key,
    );
    if (targetSegment == null) return;

    _pushUndo();

    final newAnchor = RouteAnchor(
      id: UniqueKey().toString(),
      lat: tapPoint.lat,
      lon: tapPoint.lon,
    );

    final (first, second) = splitSegmentAt(
      targetSegment,
      newAnchor.id,
      tapPoint,
    );

    final beforeIndex = state.anchors.indexWhere((a) => a.id == beforeAnchorId);
    final insertAt = beforeIndex + 1;
    final anchors = [
      ...state.anchors.sublist(0, insertAt),
      newAnchor,
      ...state.anchors.sublist(insertAt),
    ];

    final segments = [
      for (final s in state.segments)
        if (segmentKey(s.beforeAnchorId, s.afterAnchorId) == key) ...[
          first,
          second,
        ] else
          s,
    ];

    state = state.copyWith(anchors: anchors, segments: segments);
    _resolveAnchorLocation(newAnchor.id, newAnchor).ignore();
    _resolveElevation(
      segmentKey(first.beforeAnchorId, first.afterAnchorId),
      first.polyline,
    ).ignore();
    _resolveElevation(
      segmentKey(second.beforeAnchorId, second.afterAnchorId),
      second.polyline,
    ).ignore();
  }

  /// Removes [anchorId] from the route. If the removed anchor sat between
  /// two surviving anchors, its two touching segments collapse into one new
  /// straight segment spanning the surviving predecessor/successor
  /// (auto-resolved via Valhalla when auto-routing is on). Removing the
  /// first, last, or the only anchor removes at most one segment and creates
  /// none.
  void deleteAnchor(String anchorId) {
    _pushUndo();

    // Cancel any in-flight location search for the deleted anchor — its
    // response would already no-op harmlessly, but no reason to let it run.
    _locationInFlight.remove(anchorId)?.cancel();
    _locationGeneration.remove(anchorId);

    final anchors = state.anchors.where((a) => a.id != anchorId).toList();

    final before = state.segments.firstWhereOrNull(
      (s) => s.afterAnchorId == anchorId,
    );
    final after = state.segments.firstWhereOrNull(
      (s) => s.beforeAnchorId == anchorId,
    );

    RouteAnchor? predecessor;
    RouteAnchor? successor;
    if (before != null && after != null) {
      predecessor = anchors.firstWhere((a) => a.id == before.beforeAnchorId);
      successor = anchors.firstWhere((a) => a.id == after.afterAnchorId);
    }

    final segments = [
      for (final s in state.segments)
        if (s.beforeAnchorId != anchorId && s.afterAnchorId != anchorId) s,
      if (predecessor != null && successor != null)
        RouteSegment(
          beforeAnchorId: predecessor.id,
          afterAnchorId: successor.id,
          polyline: [predecessor.point, successor.point],
          state: SegmentState.straight,
        ),
    ];

    state = state.copyWith(anchors: anchors, segments: segments);

    if (predecessor != null && successor != null) {
      _resolveElevation(segmentKey(predecessor.id, successor.id), [
        predecessor.point,
        successor.point,
      ]).ignore();

      if (state.autoRoutingEnabled) {
        _resolveSegment(
          predecessor.id,
          successor.id,
          predecessor,
          successor,
        ).ignore();
      }
    }
  }

  /// Reassigns the anchor order to [newOrder] (a permutation of the
  /// existing anchor ids). Segments for anchor pairs that remain adjacent
  /// after the reorder are reused verbatim (preserving any resolved
  /// [SegmentState.routed] polyline and issuing zero Valhalla calls); only
  /// newly-adjacent pairs get a fresh straight segment, auto-resolved when
  /// auto-routing is on.
  void reorderAnchors(List<String> newOrder) {
    _pushUndo();

    final anchorsById = {for (final a in state.anchors) a.id: a};
    final reordered = newOrder.map((id) => anchorsById[id]!).toList();

    final oldByKey = {
      for (final s in state.segments)
        segmentKey(s.beforeAnchorId, s.afterAnchorId): s,
    };

    final segments = <RouteSegment>[];
    final toResolve = <(String, String, RouteAnchor, RouteAnchor)>[];

    for (var i = 0; i < reordered.length - 1; i++) {
      final a = reordered[i];
      final b = reordered[i + 1];
      final key = segmentKey(a.id, b.id);
      final existing = oldByKey[key];
      if (existing != null) {
        segments.add(existing);
      } else {
        segments.add(
          RouteSegment(
            beforeAnchorId: a.id,
            afterAnchorId: b.id,
            polyline: [a.point, b.point],
            state: SegmentState.straight,
          ),
        );
        _resolveElevation(key, [a.point, b.point]).ignore();
        if (state.autoRoutingEnabled) toResolve.add((a.id, b.id, a, b));
      }
    }

    state = state.copyWith(anchors: reordered, segments: segments);

    for (final (beforeId, afterId, a, b) in toResolve) {
      _resolveSegment(beforeId, afterId, a, b).ignore();
    }
  }

  /// Removes every anchor and segment at once. No-op on an already-empty
  /// route. Immediate, no confirmation dialog — the app-bar Undo is the sole
  /// safety net.
  ///
  /// Cancels every in-flight location search: once an anchor is gone there
  /// is no reason to let its reverse-geocode request complete.
  void deleteAllAnchors() {
    if (state.anchors.isEmpty && state.segments.isEmpty) return;

    _pushUndo();

    for (final token in _locationInFlight.values) {
      token.cancel();
    }
    _locationInFlight.clear();
    _locationGeneration.clear();

    state = state.copyWith(anchors: const [], segments: const []);
  }

  /// Reverses the anchor order — former start becomes the goal — and
  /// recomputes the route: every segment is rebuilt with its `before`/
  /// `after` ids swapped, then re-resolved via [resolveAllSegments]. A blind
  /// polyline-reversal is deliberately not used — a route can be
  /// direction-sensitive (one-way streets), so a fresh Valhalla resolve in
  /// the new direction is required rather than a mirrored copy of the old.
  ///
  /// No-op below 2 anchors. Anchor coordinates/locations carry over
  /// unchanged — only order flips, so no location re-search is needed.
  void reverseRoute() {
    if (state.anchors.length < 2) return;

    _pushUndo();

    final reversed = state.anchors.reversed.toList();
    final segments = [
      for (var i = 0; i < reversed.length - 1; i++)
        RouteSegment(
          beforeAnchorId: reversed[i].id,
          afterAnchorId: reversed[i + 1].id,
          polyline: [reversed[i].point, reversed[i + 1].point],
          state: SegmentState.straight,
        ),
    ];

    state = state.copyWith(anchors: reversed, segments: segments);
    resolveAllSegments();
  }

  /// Restores the immediately prior anchors/segments snapshot. No-ops if
  /// the undo stack is empty (defensive — the app-bar button is also
  /// disabled in this case).
  void undo() {
    final stack = state.undoStack;
    if (stack.isEmpty) return;

    final previous = stack.last;
    final redoSnapshot = RouteAnchorsSnapshot(
      anchors: state.anchors,
      segments: state.segments,
    );
    state = state.copyWith(
      anchors: previous.anchors,
      segments: previous.segments,
      undoStack: stack.sublist(0, stack.length - 1),
      redoStack: [...state.redoStack, redoSnapshot],
    );
  }

  /// Re-applies the most recently undone mutation. No-ops if the redo stack
  /// is empty.
  void redo() {
    final stack = state.redoStack;
    if (stack.isEmpty) return;

    final next = stack.last;
    final undoSnapshot = RouteAnchorsSnapshot(
      anchors: state.anchors,
      segments: state.segments,
    );
    state = state.copyWith(
      anchors: next.anchors,
      segments: next.segments,
      redoStack: stack.sublist(0, stack.length - 1),
      undoStack: [...state.undoStack, undoSnapshot],
    );
  }
}
