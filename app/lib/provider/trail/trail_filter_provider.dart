import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/tag.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/api_provider.dart';

part 'trail_filter_provider.g.dart';

@riverpod
class TrailFilterNotifier extends _$TrailFilterNotifier {
  @override
  Future<TrailFilter> build() async {
    final api = ref.watch(apiProvider);

    try {
      final response = await api.get('/trail/filter');

      if (response.data == null) {
        throw Exception('No filter metadata received from server');
      }
      TrailFilterValues filterValues = TrailFilterValues.fromJson(
        response.data,
      );

      return TrailFilter(
        q: "",
        category: [],
        tags: [],
        difficulty: [0, 1, 2],
        author: "",
        public: true,
        shared: true,
        liked: false,
        private: true,
        near: TrailNear(radius: 2000),
        distanceMin: 0,
        distanceMax: filterValues.maxDistance,
        distanceLimit: filterValues.maxDistance,
        elevationGainMin: 0,
        elevationGainMax: filterValues.maxElevationGain,
        elevationGainLimit: filterValues.maxElevationGain,
        elevationLossMin: 0,
        elevationLossMax: filterValues.maxElevationLoss,
        elevationLossLimit: filterValues.maxElevationLoss,
        sort: TrailFilterSort.created,
        sortOrder: TrailFilterSortOrder.desc,
      );
    } catch (e) {
      throw Exception('Failed to fetch trail filters: $e');
    }
  }

  void updateFilter(TrailFilter Function(TrailFilter current) updater) {
    final currentState = state.value;
    if (currentState == null) return;

    state = AsyncData(updater(currentState));
  }

  void addTag(Tag tag) {
    updateFilter((f) => f.copyWith(tags: [...f.tags, tag]));
  }

  void removeTag(Tag tag) {
    updateFilter(
      (f) => f.copyWith(
        tags: f.tags
            .where(
              (t) => t.name != tag.name && (t.id == null || t.id != tag.id),
            )
            .toList(),
      ),
    );
  }
}
