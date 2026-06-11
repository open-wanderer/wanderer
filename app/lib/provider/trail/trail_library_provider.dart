import 'package:path_provider/path_provider.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/objectbox.g.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';
import 'dart:io';

part 'trail_library_provider.g.dart';

@riverpod
class TrailLibraryNotifier extends _$TrailLibraryNotifier {
  @override
  List<Trail> build() {
    final store = ref.watch(objectBoxProvider);
    final box = store.box<TrailEntity>();
    return box.getAll().map((t) => t.toModel()).toList();
  }

  Future<void> deleteTrail(String id) async {
    final store = ref.read(objectBoxProvider);
    final box = store.box<TrailEntity>();
    final query = box.query(TrailEntity_.id.equals(id)).build();
    final entity = query.findFirst();
    query.close();

    if (entity == null) return;

    box.remove(entity.obxId);

    final appDir = await getApplicationDocumentsDirectory();
    final trailDir = Directory('${appDir.path}/library/$id');
    if (await trailDir.exists()) {
      await trailDir.delete(recursive: true);
    }

    state = state.where((t) => t.id != id).toList();
  }
}
