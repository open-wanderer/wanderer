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

/// Merges [local] (owner-scoped ObjectBox rows, read via
/// `readOwnLocalTrails`) with [network] (the existing own-trails search
/// page for the signed-in hiker's own handle), local rows first.
///
/// This ordering buys two things:
/// - A just-saved trail is visible at the top of the list immediately,
///   because it is a local row with no round trip required (REC-02).
/// - A trail that has finished uploading appears exactly once, not twice
///   (SYNC-05): its local row and the server's search hit share the same
///   `id`, and any network hit whose `id` matches a local row's `id` is
///   dropped below.
///
/// The dedupe id set is built from NON-EMPTY local ids only. A local row
/// that has never synced has an empty `id` (its identity lives in
/// `Trail.localId`, not `Trail.id`, until the drain writes a server id
/// back) — an empty id can never collide with a real network result's id,
/// so it must never be allowed to suppress one. This is why the id set is
/// built with a `where((id) => id.isNotEmpty)` guard rather than a bare
/// `.map((t) => t.id).toSet()`.
///
/// Produces one flat list — D-11 explicitly rules out sectioning or a
/// special sort mixing unsynced and downloaded-authored-by-me rows.
List<TrailSummary> mergeOwnTrails({
  required List<Trail> local,
  required List<TrailSearchResult> network,
}) {
  final localIds = local.map((t) => t.id).where((id) => id.isNotEmpty).toSet();

  final dedupedNetwork = network.where((t) => !localIds.contains(t.id));

  return [...local, ...dedupedNetwork];
}
