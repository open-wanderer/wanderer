import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/list.dart';
import 'package:wanderer/provider/api_provider.dart';

part 'list_provider.g.dart';

@riverpod
class ListNotifier extends _$ListNotifier {
  @override
  FutureOr<WandererList> build(String id) async {
    final api = ref.watch(apiProvider);

    try {
      final response = await api.get(
        "/list/$id",
        queryParameters: {
          "expand":
              "author,trails,trails.author,trails.category,trails.tags,list_share_via_list.actor",
        },
      );

      if (response.data == null) {
        throw Exception("No list data received from server");
      }

      var list = WandererList.fromJson(response.data);

      return list;
    } catch (_) {
      rethrow;
    }
  }
}
