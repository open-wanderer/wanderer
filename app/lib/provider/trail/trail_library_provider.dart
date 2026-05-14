import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/entities/trail_entity.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/objectbox_store_provider.dart';

part 'trail_library_provider.g.dart';

@riverpod
class TrailLibraryNotifier extends _$TrailLibraryNotifier {
  @override
  List<Trail> build() {
    final store = ref.watch(objectBoxProvider);

    final box = store.box<TrailEntity>();

    final savedTrails = box.getAll();

    return savedTrails.map((t) => t.toModel()).toList();
  }
}
