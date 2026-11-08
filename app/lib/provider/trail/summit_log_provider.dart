import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/summit_log.dart';
import 'package:wanderer/models/list_result.dart';
import 'package:wanderer/provider/api_provider.dart';

part 'summit_log_provider.g.dart';

@riverpod
class SummitLogListNotifier extends _$SummitLogListNotifier {
  @override
  FutureOr<List<SummitLog>> build(String trailId) async {
    try {
      final api = ref.watch(apiProvider);

      final response = await api.get(
        "/summit-log",
        queryParameters: {"filter": "trail='$trailId'", "expand": "author"},
      );

      if (response.data == null) {
        throw Exception("No summit log data received from server");
      }

      ListResult<SummitLog> summitlogListResult = ListResult.fromJson(
        response.data,
        (json) => SummitLog.fromJson(json as Map<String, dynamic>),
      );

      return summitlogListResult.items;
    } catch (e) {
      throw Exception('Failed to fetch categories: $e');
    }
  }
}
