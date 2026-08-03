import 'package:flutter/foundation.dart';
import 'package:path_provider/path_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/store/current_account.dart';
import 'package:wanderer/util/library_membership.dart';
import 'dart:io';

part 'trail_library_provider.g.dart';

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
    // Per-entity guard, not a bulk `.map()`: toModel() parses the cached GPX,
    // and a parse failure there used to propagate out of build() and fail the
    // ENTIRE offline library — one unopenable trail hid every other downloaded
    // trail, with no way for the user to fix or even identify it. Losing the
    // one bad row is the correct blast radius.
    final trails = <Trail>[];
    for (final entity in query.find()) {
      try {
        trails.add(entity.toModel());
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
  /// Only drops this account's membership. The row and its `library/<id>/`
  /// files are deleted when the LAST account gives it up -- another account
  /// that downloaded the same trail keeps a working offline copy, since all
  /// accounts share the one row and one set of files.
  Future<void> deleteTrail(String id) async {
    final store = ref.read(objectBoxProvider);
    final userId = currentAccountId(store);
    if (userId == null) return;

    final box = store.box<TrailEntity>();

    final stillHeldByAnother = store.runInTransaction(TxMode.write, () {
      final query = box.query(TrailEntity_.id.equals(id)).build();
      final entity = query.findFirst();
      query.close();
      if (entity == null) return false;

      final remaining = libraryMembersAfterDelete(
        entity.savedByUserIds,
        userId,
      );

      if (remaining.isEmpty) {
        box.remove(entity.obxId);
        return false;
      }

      entity.savedByUserIds = remaining;
      box.put(entity);
      return true;
    });

    if (!stillHeldByAnother) {
      final appDir = await getApplicationDocumentsDirectory();
      final trailDir = Directory('${appDir.path}/library/$id');
      if (await trailDir.exists()) {
        await trailDir.delete(recursive: true);
      }
    }

    state = state.where((t) => t.id != id).toList();
  }
}
