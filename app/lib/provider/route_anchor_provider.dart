import 'package:collection/collection.dart';
import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart' show UniqueKey;
import 'package:maplibre/maplibre.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/route_anchor.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/util/polyline_util.dart';
import 'package:wanderer/util/reverse_geocode_util.dart';
import 'package:wanderer/util/route_segment_util.dart';

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

  /// `'pedestrian'` or `'bicycle'` (Rec B) — no longer fixed for the
  /// notifier's lifetime; switched via [RouteAnchors.switchProfile] /
  /// [RouteAnchors.resetForSession], never via a family argument.
  final String travelProfile;

  /// Fixed Valhalla `costing_options` payload for the current
  /// [travelProfile] (e.g. `bicycle_type`, `cycling_speed`). `null` until a
  /// bucket has been applied. Set atomically with [travelProfile] by
  /// [RouteAnchors.switchProfile] / [RouteAnchors.resetForSession].
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
}

/// Route-planner state provider: owns the ordered anchor list, the
/// per-segment Valhalla routing engine (with a race-guard against
/// out-of-order responses), the geometric segment-split used by a plain
/// insert tap, and an immutable-snapshot undo/redo stack.
///
/// Rec B: a single `@Riverpod(keepAlive: true)` provider with NO family
/// argument — `travelProfile` + `costingOptions` live fully inside state and
/// are switched via [switchProfile] (mid-session bucket change) or
/// [resetForSession] (called once per planner entry, since `keepAlive` means
/// this single instance survives across sheet/screen mounts and must be
/// reset so a re-entry never leaks the previous session's route).
@Riverpod(keepAlive: true)
class RouteAnchors extends _$RouteAnchors {
  // Non-reactive bookkeeping — never in `state`, mirrors `_currentShapeIndex`
  // in `navigation_provider.dart`. Keyed by the stable `segmentKey`
  // (anchor-id pair), never array index (Pitfall 3).
  final Map<String, CancelToken> _inFlight = {};
  final Map<String, int> _generation = {};

  // Same cancel/generation bookkeeping as above, but keyed by anchor id for
  // per-anchor reverse-geocode requests (quick-260717-t7q follow-up) — each
  // anchor's location search is independent of every other anchor's, so a
  // drag on one anchor never blocks/queues behind another's in-flight
  // search (mirrors _resolveSegment's per-segment isolation, not the old
  // widget-level sequential batch it replaces).
  final Map<String, CancelToken> _locationInFlight = {};
  final Map<String, int> _locationGeneration = {};

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
  /// On failure (non-cancel `DioException`, or a malformed response), the
  /// segment is marked [SegmentState.blocked] and its prior polyline is left
  /// untouched — ROUTE-05 forbids ever silently falling back to a straight
  /// line while auto-routing is on.
  Future<void> _resolveSegment(
    String beforeAnchorId,
    String afterAnchorId,
    RouteAnchor a,
    RouteAnchor b,
  ) async {
    final key = segmentKey(beforeAnchorId, afterAnchorId);

    if (!_isValidCoordinate(a) || !_isValidCoordinate(b)) {
      // V5: reject out-of-range coordinates before ever calling Valhalla.
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

      // Stale-response guard: a newer request may have started (and even
      // completed) while this one was in flight.
      if (_generation[key] != myGeneration) return;

      final trip = response.data is Map ? response.data['trip'] : null;
      final legs = trip is Map ? trip['legs'] : null;
      final leg = (legs is List && legs.isNotEmpty) ? legs[0] : null;
      final shape = leg is Map ? leg['shape'] : null;

      if (shape is! String) {
        _markBlocked(key);
        return;
      }

      final points = PolylineUtil.decode(shape, precision: 6);
      _applySegment(key, points: points, segmentState: SegmentState.routed);
    } on DioException catch (e) {
      if (e.type == DioExceptionType.cancel)
        return; // superseded, not a real failure
      if (_generation[key] != myGeneration) return;
      _markBlocked(key); // ROUTE-05: never revert to straight, never clear
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

  void _applySegment(
    String key, {
    required List<Geographic> points,
    required SegmentState segmentState,
  }) {
    final segments = [
      for (final segment in state.segments)
        if (segmentKey(segment.beforeAnchorId, segment.afterAnchorId) == key)
          segment.copyWith(polyline: points, state: segmentState)
        else
          segment,
    ];
    state = state.copyWith(segments: segments);
  }

  /// Reverse-geocodes [anchorId]'s current coordinates and, on success,
  /// writes the result onto that anchor's [RouteAnchor.location] field.
  /// Fired once from [appendAnchor]/[insertAnchorOnSegment] (new anchor) and
  /// once at drag-end from [dragAnchor] — never continuously during a drag
  /// gesture (quick-260717-t7q follow-up: the search now lives here instead
  /// of `route_anchor_list_tab.dart`'s local widget-state batch, so it never
  /// re-runs just because the tab remounts).
  ///
  /// Best-effort and silent, mirroring the old widget-level behavior: on
  /// failure (or supersession by a newer request for the same anchor) the
  /// anchor's PRIOR `location` is left untouched rather than cleared — a
  /// dragged anchor keeps showing its last-known label until the fresh
  /// search resolves, never flashing back to the "Anchor N" fallback.
  Future<void> _resolveAnchorLocation(String anchorId, RouteAnchor anchor) async {
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

      // Stale-response guard, same shape as _resolveSegment: a newer
      // search for this same anchor may have started (or the anchor may
      // have since been deleted) while this one was in flight.
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

  /// D-09: retry lives on the blocked segment itself. Re-dispatches
  /// [_resolveSegment] for the given anchor pair.
  ///
  /// The native map's segment/marker hit-test layer syncs asynchronously
  /// (`_segmentLayer.update(...).ignore()` in the screen), so a tap can carry
  /// anchor ids that no longer exist in `state` (undo/delete/reorder raced
  /// ahead of the native layer). `firstWhereOrNull` degrades this stale
  /// hit-test to a silent no-op instead of an uncaught `StateError` thrown
  /// synchronously out of the map's `onEvent` callback.
  Future<void> retrySegment(String beforeAnchorId, String afterAnchorId) {
    final a = state.anchors.firstWhereOrNull((x) => x.id == beforeAnchorId);
    final b = state.anchors.firstWhereOrNull((x) => x.id == afterAnchorId);
    if (a == null || b == null) return Future.value();
    return _resolveSegment(beforeAnchorId, afterAnchorId, a, b);
  }

  /// Flips [RouteAnchorsState.autoRoutingEnabled]. Turning OFF leaves every
  /// existing segment untouched (only new segments created afterward become
  /// straight). Turning ON re-resolves every existing segment via Valhalla,
  /// awaited in parallel via `Future.wait` so callers observe the fully
  /// re-resolved state once this method completes.
  Future<void> toggleAutoRouting() async {
    final enabled = !state.autoRoutingEnabled;
    state = state.copyWith(autoRoutingEnabled: enabled);
  }

  /// Switches the travel bucket mid-session (Rec B): sets [profile] +
  /// [opts] atomically and re-resolves every existing segment under the new
  /// costing (this is the general rule — including a within-`bicycle`
  /// sub-type switch like Hybrid -> Road, CONTEXT). Anchors are never
  /// migrated (they never leave this single keepAlive instance).
  ///
  /// A profile switch is a FRESH undo baseline (CONTEXT: "the switch itself
  /// is not undoable") — both stacks are cleared and no undo snapshot is
  /// pushed, matching the prior "travelProfile fixed for lifetime"
  /// invariant.
  void switchProfile(String profile, Map<String, dynamic> opts) {
    state = state.copyWith(
      travelProfile: profile,
      costingOptions: opts,
      undoStack: const [],
      redoStack: const [],
    );
    resolveAllSegments();
  }

  /// Bulk re-resolve: iterates every consecutive anchor pair that has an
  /// existing segment and either re-dispatches a Valhalla resolve (when
  /// auto-routing is on, fire-and-forget, mirroring [reorderAnchors]'s
  /// `toResolve` loop) or applies a straight two-point polyline (when
  /// auto-routing is off). Used by [switchProfile] so any bucket switch
  /// re-resolves the WHOLE route, not just newly-adjacent pairs.
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

  /// Resets the single keepAlive instance to a brand-new empty session
  /// (T-t7q-03): called once at planner-screen mount so a re-entry never
  /// leaks the previous session's route. Cancels every in-flight request
  /// first.
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
  /// prepopulated anchors (quick-260718-e9j, PLANNER-02) — the Route
  /// Planner's "edit an existing route" entry mode, as opposed to
  /// [resetForSession]'s empty-session entry for a brand-new/imported route.
  ///
  /// Cancels + clears all in-flight/generation bookkeeping exactly like
  /// [resetForSession] (both the segment-resolve maps and the per-anchor
  /// location-search maps). Builds one [RouteAnchor] per point (fresh
  /// [UniqueKey] ids — never reused across sessions) and one [RouteSegment]
  /// per consecutive pair, then sets state in a single assignment with a
  /// fresh (empty) undo/redo baseline — this seed is not itself undoable,
  /// matching [resetForSession]'s own contract.
  ///
  /// [segmentPolylines] (from
  /// [route_planner_handoff_util.dart]'s `segmentPolylinesFromTrack`), when
  /// given, supplies each segment's full original recorded points; a
  /// missing/short entry falls back to a straight 2-point line between its
  /// bounding anchors. Every seeded segment is marked [SegmentState.straight]
  /// regardless of point count — that enum name means "not Valhalla-routed",
  /// not "exactly 2 points" (`splitSegmentAt` already works generically on
  /// any polyline).
  ///
  /// Deliberately does NOT loop [appendAnchor] (each call pushes an undo
  /// snapshot, which would let the user undo back through the seed), does
  /// NOT call [_resolveAnchorLocation] for any seeded anchor (no
  /// reverse-geocode at load time — matches web's `initRouteAnchors`), and,
  /// critically, does NOT call [resolveAllSegments]. The web app never
  /// touches the original recorded points at seed time (`setRoute(gpx)`
  /// loads the untouched GPX; anchors are just markers on top of it) — a
  /// segment is only ever Valhalla-resolved once the user actually edits it.
  /// Auto-resolving every segment on open (this function's first version)
  /// silently snapped an off-road recording onto nearby roads before the
  /// user touched anything.
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
  }

  /// Pushes the current (pre-mutation) anchors/segments onto the undo stack
  /// and clears the redo stack (D-11: any new action clears redo). Called
  /// FIRST, before mutating, by every mutation method below.
  void _pushUndo() {
    state = state.copyWith(
      undoStack: [
        ...state.undoStack,
        RouteAnchorsSnapshot(anchors: state.anchors, segments: state.segments),
      ],
      redoStack: const [],
    );
  }

  /// WAYP-01: appends a new anchor to the end of the route. If a previous
  /// last anchor existed, also creates the connecting segment (straight by
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

  /// WAYP-02: repositions [anchorId] and re-resolves only its ≤2 adjacent
  /// segments. Invoked once per drag gesture at `onPanEnd` (D-05), never
  /// during `onPanUpdate`.
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
    // Drag-end only (D-05, never onPanUpdate) — re-searches this anchor's
    // location against its NEW coordinates; the stale prior label persists
    // until this resolves (never cleared eagerly).
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

  /// WAYP-03: on a plain (non-blocked) segment tap, geometrically splits it
  /// and inserts the new anchor between its two endpoint anchors — never
  /// calls Valhalla (Open Question 1, resolved). The new anchor's list
  /// position IS its D-02 display-number renumbering; numbers are always
  /// derived from index, never stored.
  void insertAnchorOnSegment(
    String beforeAnchorId,
    String afterAnchorId,
    Geographic tapPoint,
  ) {
    final key = segmentKey(beforeAnchorId, afterAnchorId);
    // Same stale-hit-test guard as retrySegment: the native layer's GeoJSON
    // sync is async, so a tap can reference a segment already removed by an
    // undo/delete/reorder. Bail out BEFORE _pushUndo() so a miss never
    // pushes a spurious no-op undo snapshot.
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
  }

  /// WAYP-04: removes [anchorId] from the route. If the removed anchor sat
  /// between two surviving anchors, its two touching segments collapse into
  /// one new straight segment spanning the surviving predecessor/successor
  /// (auto-resolved via Valhalla when auto-routing is on). Removing the
  /// first, last, or the only anchor removes at most one segment and creates
  /// none.
  void deleteAnchor(String anchorId) {
    _pushUndo();

    // Cancel (rather than let dangle) any in-flight location search for the
    // deleted anchor — its eventual response would already no-op harmlessly
    // (the id no longer matches any anchor) but there is no reason to let
    // the network call complete.
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

    if (predecessor != null && successor != null && state.autoRoutingEnabled) {
      _resolveSegment(
        predecessor.id,
        successor.id,
        predecessor,
        successor,
      ).ignore();
    }
  }

  /// WAYP-05: reassigns the anchor order to [newOrder] (a permutation of the
  /// existing anchor ids). Segments for anchor pairs that remain adjacent
  /// after the reorder are reused verbatim (preserving any resolved
  /// [SegmentState.routed] polyline and issuing zero Valhalla calls,
  /// Pitfall 4); only newly-adjacent pairs get a fresh straight segment,
  /// auto-resolved when auto-routing is on.
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
        if (state.autoRoutingEnabled) toResolve.add((a.id, b.id, a, b));
      }
    }

    state = state.copyWith(anchors: reordered, segments: segments);

    for (final (beforeId, afterId, a, b) in toResolve) {
      _resolveSegment(beforeId, afterId, a, b).ignore();
    }
  }

  /// Removes every anchor and segment at once. No-op on an already-empty
  /// route. Immediate, no confirmation dialog (mirrors [deleteAnchor]'s own
  /// D-06 discipline) — the app-bar Undo is the sole safety net.
  ///
  /// Cancels every in-flight location search (mirrors [resetForSession]):
  /// once an anchor is gone there is no reason to let its reverse-geocode
  /// request complete.
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

  /// Reverses the anchor order — the former start becomes the goal and vice
  /// versa — and recomputes the route: every segment is rebuilt with its
  /// `before`/`after` anchor ids swapped to match the new direction, then
  /// re-resolved via [resolveAllSegments] (auto-routing on) or flattened to
  /// a straight line (auto-routing off). A blind polyline-reversal is
  /// deliberately NOT used — a real route can be direction-sensitive
  /// (one-way streets), so the safe behavior is a fresh Valhalla resolve in
  /// the new direction, not a mirrored copy of the old one.
  ///
  /// No-op below 2 anchors (nothing to reverse). Each [RouteAnchor] (and its
  /// resolved `location`, if any) carries over unchanged — only order
  /// flips, coordinates don't change, so no location re-search is needed.
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

  /// ROUTE-04: restores the immediately prior anchors/segments snapshot.
  /// No-ops if the undo stack is empty (defensive — the app-bar button is
  /// also disabled per D-11).
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

  /// ROUTE-04: re-applies the most recently undone mutation. No-ops if the
  /// redo stack is empty.
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
