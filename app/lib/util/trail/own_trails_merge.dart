/// Pure local+network merge for the own-trails list
/// (`/profile/<handle>/trails` for the signed-in hiker's own handle).
///
/// Deliberately dependency-light: no Riverpod, no ObjectBox, no filesystem
/// I/O. Unit-tested in `own_trails_merge_test.dart` against plain
/// `Trail(...)`/`TrailSearchResult(...)` fixtures, no Store.
///
/// The list's other half — narrowing the local rows by the search query —
/// lives with the rest of the trail filtering in `filter.dart`.
library;

import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/models/trail_summary.dart';
import 'package:wanderer/models/trail_sync_state.dart';

/// The local rows that belong in the merged list for the current
/// connectivity, in the order [readOwnLocalTrails] returned them.
///
/// Online, that is the NOT-YET-SYNCED rows only. A row that has reached the
/// server is already represented by the network half — keeping the device
/// copy too pinned every downloaded trail above every online result, which
/// is the local store deciding the list's order rather than the server's
/// sort. Dropping it here rather than deduping it away below also keeps the
/// server's (freshly indexed) copy as the one that renders.
///
/// Offline there is no network half at all, so every local row is kept —
/// the device copy is the only copy there is.
///
/// The unsynced rows stay at the top online because they exist nowhere else
/// yet; they leave the top of their own accord as soon as the drain retires
/// them.
List<Trail> ownTrailsLocalHalf(List<Trail> local, {required bool offline}) {
  if (offline) return local;
  return local.where((t) => isUnsyncedState(t.syncState)).toList();
}

/// Merges [local] (owner-scoped ObjectBox rows, read via
/// `readOwnLocalTrails`) with [network] (the existing own-trails search
/// page for the signed-in hiker's own handle), local rows first.
///
/// [local] is first narrowed by [ownTrailsLocalHalf], so online only the
/// unsynced rows survive to the front of the list.
///
/// This ordering buys two things:
/// - A just-saved trail is visible at the top of the list immediately,
///   because it is a local row with no round trip required.
/// - A trail that has finished uploading appears exactly once, not twice:
/// online it is dropped from the local half outright; offline
///   its local row and the server's search hit share the same `id`, and any
///   network hit whose `id` matches a surviving local row's `id` is dropped
///   below.
///
/// The dedupe id set is built from NON-EMPTY local ids only. A local row
/// that has never synced has an empty `id` (its identity lives in
/// `Trail.localId`, not `Trail.id`, until the drain writes a server id
/// back) — an empty id can never collide with a real network result's id,
/// so it must never be allowed to suppress one. This is why the id set is
/// built with a `where((id) => id.isNotEmpty)` guard rather than a bare
/// `.map((t) => t.id).toSet()`.
///
/// Produces one flat list — deliberately no sectioning and no
/// special sort mixing unsynced and downloaded-authored-by-me rows.
List<TrailSummary> mergeOwnTrails({
  required List<Trail> local,
  required List<TrailSearchResult> network,
  required bool offline,
}) {
  final localHalf = ownTrailsLocalHalf(local, offline: offline);

  final localIds = localHalf
      .map((t) => t.id)
      .where((id) => id.isNotEmpty)
      .toSet();

  final dedupedNetwork = network.where((t) => !localIds.contains(t.id));

  return [...localHalf, ...dedupedNetwork];
}
