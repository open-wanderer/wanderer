import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/list.dart';
import 'package:wanderer/models/trail.dart';

part 'list_filter_provider.g.dart';

@riverpod
class ListFilterNotifier extends _$ListFilterNotifier {
  @override
  Future<ListFilter> build() async {
    try {
      return ListFilter(
        q: "",
        author: "",
        public: true,
        shared: true,
        sort: ListFilterSort.created,
        sortOrder: SortOrder.desc,
      );
    } catch (e) {
      throw Exception('Failed to fetch list filters: $e');
    }
  }

  void updateFilter(ListFilter Function(ListFilter current) updater) {
    final currentState = state.value;
    if (currentState == null) return;

    state = AsyncData(updater(currentState));
  }
}
