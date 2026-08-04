// The single sanctioned way to build a trail detail/map route path.
//
// `TrailEntity.toModel()` blanks a local-sentinel id (D-06): a not-yet-
// uploaded trail's `Trail.id` is `''`, so `'/trail/${trail.id}'` becomes
// `'/trail/'`. go_router 17.3.0's `canonicalUri` normalizes that to `'/trail'`
// -- a path with no GoRoute, which go_router renders as a no-route error page
// instead of the detail screen. Every push that interpolates a trail id must
// go through [trailDetailLocation] / [trailMapLocation] instead of building
// the path inline. A null return means "not addressable"; callers must
// disable the affordance rather than push it.

import 'package:wanderer/models/trail_sync_state.dart';
import 'package:wanderer/models/trail_summary.dart';

/// The `/trail/...` route location for [trail], or null when [trail] cannot
/// be addressed at all.
///
/// An unsynced trail (per [isUnsyncedState]) is addressed by its
/// [TrailSummary.localId] via `/trail/local/<localId>`; null when
/// [TrailSummary.localId] is null or empty. A synced trail is addressed by
/// its [TrailSummary.id] via `/trail/<id>`; null when [TrailSummary.id] is
/// empty.
///
/// [forceOffline] appends the `?offline=1` query parameter, which tells the
/// detail screen to show the downloaded copy rather than the server one. It is
/// a no-op for an unsynced trail, which has no server copy to prefer over.
String? trailDetailLocation(TrailSummary trail, {bool forceOffline = false}) {
  if (isUnsyncedState(trail.syncState)) {
    final localId = trail.localId;
    if (localId == null || localId.isEmpty) return null;
    return '/trail/local/$localId';
  }
  if (trail.id.isEmpty) return null;
  return forceOffline ? '/trail/${trail.id}?offline=1' : '/trail/${trail.id}';
}

/// The `/trail/.../map` route location for [trail], or null under the same
/// conditions as [trailDetailLocation].
String? trailMapLocation(TrailSummary trail, {bool forceOffline = false}) {
  // Built from the UN-flagged base: the query string belongs at the end of the
  // full path, and '/trail/x?offline=1/map' would match no route. The unsynced
  // exclusion is repeated here for the same reason it exists above — the local
  // route has no server copy to prefer the download over.
  final base = trailDetailLocation(trail);
  if (base == null) return null;
  return forceOffline && !isUnsyncedState(trail.syncState)
      ? '$base/map?offline=1'
      : '$base/map';
}
