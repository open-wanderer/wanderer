import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/list.dart';
import 'package:wanderer/models/trail.dart';

part 'list_filter_provider.g.dart';

@riverpod
class ListFilterNotifier extends _$ListFilterNotifier {
  late final ListFilter defaultFilter;

  @override
  Future<ListFilter> build(String filterId) async {
    defaultFilter = const ListFilter(
      q: "",
      author: "",
      public: true,
      shared: true,
      sort: ListFilterSort.created,
      sortOrder: SortOrder.desc,
    );
    return defaultFilter;
  }

  void resetFilter() {
    state = AsyncData(defaultFilter);
  }

  void updateFilter(ListFilter Function(ListFilter current) updater) {
    final currentState = state.value;
    if (currentState == null) return;
    state = AsyncData(updater(currentState));
  }
}
