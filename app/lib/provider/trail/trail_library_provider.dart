import 'package:flutter/foundation.dart';
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/entities/waypoint_entity.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/store/current_account.dart';
import 'package:wanderer/store/local_trail_store.dart';
import 'package:wanderer/util/local/id.dart';
import 'package:wanderer/util/local/library_membership.dart';
import 'dart:io';

part 'trail_library_provider.g.dart';

/// The set of trail ids currently in the signed-in account's downloaded
/// library -- the single named id-set view other providers/utils inject into
/// `applyTrailFilter`'s `downloadedIds` parameter for the `offlineOnly` chip.
///
/// Account scoping is INHERITED from [trailLibraryProvider] and must never
/// be re-implemented here: that provider already returns `const []` for a
/// signed-out account, which is what keeps one account's downloads from
/// leaking into another's filter through this view.
///
/// Empty ids are dropped -- an unsynced local capture has a blanked server
/// id, and is covered instead by `applyTrailFilter`'s separate `isLocal`
/// branch.
@riverpod
Set<String> downloadedTrailIds(Ref ref) {
  final library = ref.watch(trailLibraryProvider);
  return library.map((t) => t.id).where((id) => id.isNotEmpty).toSet();
}

/// [downloadedTrailIds] narrowed to the trails authored by [authorActorId] --
/// the set a profile search sends to the server as its `id IN [...]` clause.
///
/// The parameter is non-nullable on purpose. A caller that has not resolved
/// the actor id must NOT fall through to the unnarrowed set on a profile:
/// that would send the whole library, including trails from other instances
/// and other authors, to whichever server is serving that profile — and for
/// a federated actor that server is a third party, since
/// `/profile/{handle}/trails` proxies the request body verbatim to the origin
/// instance. Passing a sentinel like `''` yields an empty set, which fails
/// closed.
///
/// Narrowing is a privacy and payload measure, not a correctness one:
/// `/profile/{handle}/trails` already ANDs `author = <actor>` server-side, so
/// an unnarrowed set would still return the right trails. That is what makes
/// the whole-library fallback safe on the map, where the request goes only to
/// the user's own instance.
/// The id set a MAP search sends as its `id IN [...]` clause.
///
/// Unlike the profile path this may fall back to the whole library when no
/// author is in scope (the global map). That is deliberate and safe: both map
/// endpoints (`/search/trails`, `/search/trails/cluster`) are served by the
/// user's OWN instance against its local Meilisearch index and are never
/// proxied elsewhere, so there is no third party to disclose the library to.
/// The federated-proxy hazard is specific to `/profile/{handle}/trails`.
///
/// Returns an empty set when the chip is off, so the clause is omitted
/// entirely rather than emitted as a never-matching one.
Set<String> offlineTrailIdsForMapSearch(
  Ref ref, {
  required bool offlineOnly,
  String? authorId,
}) {
  if (!offlineOnly) return const {};
  if (authorId != null && authorId.isNotEmpty) {
    return ref.read(downloadedTrailIdsForAuthorProvider(authorId));
  }
  return ref.read(downloadedTrailIdsProvider);
}

@riverpod
Set<String> downloadedTrailIdsForAuthor(Ref ref, String authorActorId) {
  if (authorActorId.isEmpty) return const {};
  return ref
      .watch(trailLibraryProvider)
      .where((t) => t.author == authorActorId)
      .map((t) => t.id)
      .where((id) => id.isNotEmpty)
      .toSet();
}

@riverpod
class TrailLibraryNotifier extends _$TrailLibraryNotifier {
  @override
  List<Trail> build() {
    final store = ref.watch(objectBoxProvider);
    final userId = currentAccountId(store);

    // No signed-in account means an EMPTY library, never the whole box --
    // an unfiltered read here is exactly the leak that let account B see
    // account A's downloaded (including private) trails.
    if (userId == null) return const [];

    final box = store.box<TrailEntity>();
    final query = box
        .query(TrailEntity_.savedByUserIds.containsElement(userId))
        .build();
    // Per-entity guard, not a bulk `.map()`: toModel() can throw on a corrupt
    // row, and a failure there used to propagate out of build() and fail the
    // ENTIRE offline library — one unopenable trail hid every other downloaded
    // trail, with no way for the user to fix or even identify it. Losing the
    // one bad row is the correct blast radius.
    //
    // includeGpx: false — the library list renders scalar columns only, and
    // parsing every downloaded trail's full GPX here was a synchronous
    // UI-thread XML pass over the entire library on every open, then held
    // every trackpoint + raw XML in memory for the list's lifetime. Detail
    // screens re-read their own full model on open.
    final trails = <Trail>[];
    for (final entity in query.find()) {
      try {
        trails.add(entity.toModel(includeGpx: false));
      } catch (e, st) {
        debugPrint(
          'TrailLibrary: skipping cached trail "${entity.id}" — '
          'toModel() failed: $e\n$st',
        );
      }
    }
    query.close();

    trails.sort((a, b) => b.created.compareTo(a.created));
    return trails;
  }

  /// Removes [id] from the signed-in account's library.
  ///
  /// Always drops this account's `savedByUserIds` membership -- "Remove
  /// download" is never a silent no-op. The row and its `library/<id>/`
  /// files are deleted ONLY when this account was the last library member
  /// AND the row is not still some account's live capture ([isLiveCaptureRow]).
  ///
  /// Both "Remove download" confirm dialogs promise "the trail itself is
  /// not deleted". This method used to unconditionally remove the row from
  /// the box whenever this account was the last library member -- but on a
  /// row with `owner` set, `localId` set, `syncState` unsynced and
  /// `savedByUserIds == [thisAccount]`, that row IS the hiker's pending
  /// recording: removing it destroyed the queued upload
  /// (`selectDrainCandidates` can never find it again), leaked its
  /// `WaypointEntity` children, and orphaned its unsynced photo directory
  /// that this method does not (and must not) clean. Download removal is
  /// therefore scoped by `savedByUserIds` membership, never by anything
  /// that could also be capture state.
  ///
  /// Deliberate consequence: when the row is kept as a live capture with no
  /// remaining library members, `library/<id>/` is NOT deleted. That
  /// directory holds the local file paths `TrailDownloadService` wrote into
  /// the row's `photos` column -- deleting those files while keeping the
  /// row would leave the hiker's pending recording pointing at missing
  /// images. A leaked directory is recoverable (and the row stays reachable
  /// from the hiker's own-trails list); a broken image reference is not.
  Future<void> deleteTrail(String id) async {
    final store = ref.read(objectBoxProvider);
    final userId = currentAccountId(store);
    if (userId == null) return;

    final box = store.box<TrailEntity>();

    // `rowRemoved` is true ONLY on the branch that actually removes the
    // entity from the box. Every other branch (no matching row, or the row
    // is kept) returns false.
    final rowRemoved = store.runInTransaction(TxMode.write, () {
      final query = box.query(TrailEntity_.id.equals(id)).build();
      final entity = query.findFirst();
      query.close();
      if (entity == null) return false;

      final remaining = libraryMembersAfterDelete(
        entity.savedByUserIds,
        userId,
      );

      if (remaining.isEmpty && !isLiveCaptureRow(entity)) {
        // The waypoint children go with the row.
        // `local_trail_store.dart` names this exact call site as the leak
        // `retireUploadedLocalTrail` deliberately does not copy: an orphaned
        // WaypointEntity has a dangling `trail` ToOne, is invisible to every
        // read path, and accumulates for the life of the install (it only
        // self-heals if the same trail is re-downloaded, because
        // WaypointEntity.id is @Unique(onConflict: replace)). `author` and
        // `category` are ToOne targets SHARED with other trails and are
        // deliberately left alone, exactly as in retireUploadedLocalTrail.
        final waypointBox = store.box<WaypointEntity>();
        for (final waypoint in entity.waypoints) {
          waypointBox.remove(waypoint.obxId);
        }
        box.remove(entity.obxId);
        return true;
      }

      // Keep-the-row path: membership is dropped unconditionally -- that is
      // what "Remove download" means, and it must still happen even when
      // the row is kept because it is a live capture for some account.
      entity.savedByUserIds = remaining;
      box.put(entity);
      return false;
    });

    if (rowRemoved) {
      // Best-effort. Both call sites are confirm handlers that
      // discard this future, so an I/O error here -- a file held open by the
      // photo viewer, a permission failure, a malformed id rejected by
      // `recordIdDirSegment` -- used to escape as an unhandled async error
      // AND skip the `state` update below, leaving the library list
      // rendering a trail whose row is already gone. An unreclaimed
      // directory must never leave the list stale. This matches the
      // best-effort discipline of `_deletePhotoDirBestEffort`,
      // `sweepOrphanedUnsyncedPhotos` and `deleteUnsyncedPhotoDir`; this
      // path was the odd one out.
      try {
        final appDir = await getApplicationDocumentsDirectory();
        // p.join + a whitelisted segment, never interpolation: `id` comes
        // straight off a `Trail` model built from a server response, and
        // `trail_download_service.dart` builds this same path this same way.
        final trailDir = Directory(
          p.join(appDir.path, 'library', recordIdDirSegment(id)),
        );
        if (await trailDir.exists()) {
          await trailDir.delete(recursive: true);
        }
      } catch (e, st) {
        debugPrint(
          'TrailLibrary: library dir cleanup failed for "$id": $e\n$st',
        );
      }
    }

    state = state.where((t) => t.id != id).toList();
  }
}
