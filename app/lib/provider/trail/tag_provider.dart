import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/tag.dart';
import 'package:wanderer/models/list_result.dart';
import 'package:wanderer/provider/api_provider.dart';

part 'tag_provider.g.dart';

@riverpod
class TagNotifier extends _$TagNotifier {
  @override
  FutureOr<List<Tag>> build() async {
    return [];
  }

  Future<List<Tag>> searchByName(String name) async {
    if (name.isEmpty) {
      state = const AsyncData([]);
      return [];
    }

    bool isCancelled = false;
    final link = ref.keepAlive();
    ref.onDispose(() {
      isCancelled = true;
    });

    await Future.delayed(const Duration(milliseconds: 300));

    if (isCancelled) {
      link.close();
      return [];
    }

    final api = ref.read(apiProvider);
    state = const AsyncLoading();

    final result = await AsyncValue.guard(() async {
      final response = await api.get("/tag?filter=name~'$name'");

      if (response.data == null) {
        throw Exception('No tags data received from server');
      }

      ListResult<Tag> tagListResult = ListResult.fromJson(
        response.data,
        (json) => Tag.fromJson(json as Map<String, dynamic>),
      );

      return tagListResult.items;
    });

    state = result;
    link.close();

    return result.value ?? [];
  }

  Future<Tag> create(String name) async {
    final api = ref.read(apiProvider);
    final response = await api.put('/tag', data: {'name': name});
    return Tag.fromJson(response.data);
  }
}
