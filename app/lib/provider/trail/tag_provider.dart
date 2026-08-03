import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart' show debugPrint;
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/tag.dart';
import 'package:wanderer/models/list_result.dart';
import 'package:wanderer/provider/api_provider.dart';
import 'package:wanderer/provider/online_status_provider.dart';
import 'package:wanderer/util/pb_filter.dart';

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

    // OFFUI-02: autocomplete degrades to "no suggestions" rather than
    // surfacing anything, because there is deliberately no tag cache — offline
    // the user types a free-form tag and `_resolveTags`
    // (`trail_save_provider.dart`) creates it at save/upload time.
    //
    // But "no suggestions" must not be the whole story for a REAL failure. A
    // blanket `AsyncValue.guard` made a 500, a malformed payload and airplane
    // mode indistinguishable — all silently empty, nothing to find in the
    // field. So the two are split: a connection failure is an expected,
    // uninteresting outcome and publishes as empty data; anything else keeps
    // the error in `state` and logs it (the WR-12 reasoning in
    // `import_trail_file.dart` — an error nobody can see is an error nobody
    // can fix). Both still return `[]`, so the widget behaves identically.
    List<Tag> items = const [];
    try {
      // `name` is raw user keystrokes. Escaped and passed as a query
      // parameter, never concatenated into the path — see `pb_filter_util`
      // for why nothing below this line would do it for us.
      final response = await api.get(
        '/tag',
        queryParameters: {'filter': "name~'${escapePbFilterValue(name)}'"},
      );

      if (response.data == null) {
        throw Exception('No tags data received from server');
      }

      items = ListResult.fromJson(
        response.data,
        (json) => Tag.fromJson(json as Map<String, dynamic>),
      ).items;
      state = AsyncData(items);
    } on DioException catch (e, st) {
      if (isConnectionFailure(e)) {
        state = const AsyncData([]);
      } else {
        debugPrint('tag search failed for "$name": $e');
        state = AsyncError(e, st);
      }
    } catch (e, st) {
      debugPrint('tag search failed for "$name": $e');
      state = AsyncError(e, st);
    } finally {
      link.close();
    }

    return items;
  }

  Future<Tag> create(String name) async {
    final api = ref.read(apiProvider);
    final response = await api.put('/tag', data: {'name': name});
    return Tag.fromJson(response.data);
  }
}
