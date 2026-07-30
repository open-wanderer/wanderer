import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/tag.dart';
import 'package:wanderer/models/trail.dart';
import 'package:wanderer/provider/api_provider.dart';

part 'trail_filter_provider.g.dart';

@Riverpod(keepAlive: true)
class TrailFilterNotifier extends _$TrailFilterNotifier {
  /// Deliberately `late` and NOT `late final`: [build] assigns it, and
  /// Riverpod keeps one Notifier instance alive across rebuilds — only
  /// `build()` re-runs. A `late final` therefore threw
  /// `LateInitializationError: Field 'defaultFilter' has already been
  /// initialized` on the second build, which an account switch triggers every
  /// time (this provider is invalidated from `accountScopedProviders`), and
  /// which any other refresh would have hit too.
  late TrailFilter defaultFilter;

  @override
  Future<TrailFilter> build(String filterId) async {
    final api = ref.watch(apiProvider);

    try {
      final response = await api.get('/trail/filter');

      if (response.data == null) {
        throw Exception('No filter metadata received from server');
      }
      TrailFilterValues filterValues = TrailFilterValues.fromJson(
        response.data,
      );

      defaultFilter = TrailFilter(
        q: "",
        category: [],
        subcategory: [],
        tags: [],
        difficulty: [0, 1, 2],
        author: null,
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
        sortOrder: SortOrder.desc,
      );

      return defaultFilter;
    } catch (e) {
      throw Exception('Failed to fetch trail filters: $e');
    }
  }

  void resetFilter() {
    state = AsyncData(defaultFilter);
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
