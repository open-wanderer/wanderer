/// Pure local+network merge for the own-trails list
/// (`/profile/<handle>/trails` for the signed-in hiker's own handle).
///
/// Deliberately dependency-light: no Riverpod, no ObjectBox, no filesystem
/// I/O.
/// Both functions here are unit-tested in `own_trails_merge_test.dart`
/// against plain `Trail(...)`/`TrailSearchResult(...)` fixtures, no Store.
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

/// The local half of the own-trails search filter.
///
/// The network half is already filtered server-side by Meilisearch; this
/// is its local equivalent, so a search does not make locally-held trails
/// vanish from the list.
///
/// Returns [local] unchanged for an empty or whitespace-only [q]. Otherwise
/// keeps rows whose `name`, or non-null `location`, contains [q]
/// case-insensitively.
List<Trail> filterOwnTrailsByQuery(List<Trail> local, String q) {
  final trimmed = q.trim();
  if (trimmed.isEmpty) return local;

  final lowerQuery = trimmed.toLowerCase();
  return local.where((t) {
    if (t.name.toLowerCase().contains(lowerQuery)) return true;
    final location = t.location;
    if (location != null && location.toLowerCase().contains(lowerQuery)) {
      return true;
    }
    return false;
  }).toList();
}
