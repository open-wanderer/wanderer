import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/comment.dart';
import 'package:wanderer/models/list_result.dart';
import 'package:wanderer/provider/api_provider.dart';

part 'comment_provider.g.dart';

@riverpod
class CommentNotifier extends _$CommentNotifier {
  @override
  FutureOr<List<Comment>> build(String trailId) async {
    try {
      final api = ref.watch(apiProvider);

      final response = await api.get(
        "/trail/$trailId/comment",
        queryParameters: {"expand": "author", "sort": "-created"},
      );

      if (response.data == null) {
        throw Exception("No comment data received from server");
      }

      ListResult<Comment> commentListResult = ListResult.fromJson(
        response.data,
        (json) => Comment.fromJson(json as Map<String, dynamic>),
      );

      return commentListResult.items;
    } catch (e) {
      throw Exception('Failed to fetch categories: $e');
    }
  }
}
