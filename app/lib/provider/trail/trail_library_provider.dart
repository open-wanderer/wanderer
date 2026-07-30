import 'package:path_provider/path_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'package:wanderer/util/current_account.dart';
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
    final trails = query.find().map((t) => t.toModel()).toList();
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
