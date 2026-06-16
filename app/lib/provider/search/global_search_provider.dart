import 'dart:async';

import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/provider/api_provider.dart';

part 'global_search_provider.g.dart';

class GlobalSearchState {
  final String query;
  final GlobalSearchCategory category;
  final List<TrailSearchResult> trails;
  final List<ListSearchResult> lists;
  final List<LocationSearchResult> locations;
  final List<ActorSearchResult> actors;
  final bool isLoading;
  final String? error;

  const GlobalSearchState({
    required this.query,
    required this.category,
    required this.trails,
    required this.lists,
    required this.locations,
    required this.actors,
    required this.isLoading,
    this.error,
  });

  factory GlobalSearchState.initial() => const GlobalSearchState(
    query: '',
    category: GlobalSearchCategory.all,
    trails: [],
    lists: [],
    locations: [],
    actors: [],
    isLoading: false,
  );

  static const _unset = Object();

  GlobalSearchState copyWith({
    String? query,
    GlobalSearchCategory? category,
    List<TrailSearchResult>? trails,
    List<ListSearchResult>? lists,
    List<LocationSearchResult>? locations,
    List<ActorSearchResult>? actors,
    bool? isLoading,
    Object? error = _unset,
  }) {
    return GlobalSearchState(
      query: query ?? this.query,
      category: category ?? this.category,
      trails: trails ?? this.trails,
      lists: lists ?? this.lists,
      locations: locations ?? this.locations,
      actors: actors ?? this.actors,
      isLoading: isLoading ?? this.isLoading,
      error: identical(error, _unset) ? this.error : error as String?,
    );
  }

  List<TrailSearchResult> get visibleTrails =>
      (category == GlobalSearchCategory.all ||
          category == GlobalSearchCategory.trails)
      ? trails
      : [];

  List<ListSearchResult> get visibleLists =>
      (category == GlobalSearchCategory.all ||
          category == GlobalSearchCategory.lists)
      ? lists
      : [];

  List<LocationSearchResult> get visibleLocations =>
      (category == GlobalSearchCategory.all ||
          category == GlobalSearchCategory.locations)
      ? locations
      : [];

  List<ActorSearchResult> get visibleActors =>
      (category == GlobalSearchCategory.all ||
          category == GlobalSearchCategory.actors)
      ? actors
      : [];

  bool get hasResults =>
      visibleTrails.isNotEmpty ||
      visibleLists.isNotEmpty ||
      visibleLocations.isNotEmpty ||
      visibleActors.isNotEmpty;
}

@riverpod
class GlobalSearchNotifier extends _$GlobalSearchNotifier {
  Timer? _debounce;
  bool _cancelled = false;

  @override
  GlobalSearchState build() {
    ref.onDispose(() {
      _cancelled = true;
      _debounce?.cancel();
    });
    return GlobalSearchState.initial();
  }

  void setQuery(String q) {
    _debounce?.cancel();
    if (q.isEmpty) {
      state = GlobalSearchState.initial().copyWith(category: state.category);
      return;
    }
    state = state.copyWith(query: q, isLoading: true);
    _debounce = Timer(const Duration(milliseconds: 500), _search);
  }

  void setCategory(GlobalSearchCategory category) {
    state = state.copyWith(category: category);
    if (state.query.isNotEmpty) {
      _debounce?.cancel();
      _search();
    }
  }

  Future<void> _search() async {
    final q = state.query;
    final category = state.category;
    if (q.isEmpty) return;

    state = state.copyWith(isLoading: true, error: null);

    final api = ref.read(apiProvider);

    List<TrailSearchResult> trails = [];
    List<ListSearchResult> lists = [];
    List<LocationSearchResult> locations = [];
    List<ActorSearchResult> actors = [];

    final bool searchTrails =
        category == GlobalSearchCategory.all ||
        category == GlobalSearchCategory.trails;
    final bool searchLists =
        category == GlobalSearchCategory.all ||
        category == GlobalSearchCategory.lists;
    final bool searchLocations =
        category == GlobalSearchCategory.all ||
        category == GlobalSearchCategory.locations;
    final bool searchActors =
        category == GlobalSearchCategory.all ||
        category == GlobalSearchCategory.actors;

    await Future.wait([
      if (searchTrails)
        api
            .post(
              '/search/trails',
              data: {
                'q': q,
                'options': {'hitsPerPage': 10, 'page': 1},
              },
            )
            .then((response) {
              trails = (response.data['hits'] as List<dynamic>)
                  .map(
                    (e) =>
                        TrailSearchResult.fromJson(e as Map<String, dynamic>),
                  )
                  .toList();
            })
            .catchError((_) {}),
      if (searchLists)
        api
            .post(
              '/search/lists',
              data: {
                'q': q,
                'options': {'hitsPerPage': 10, 'page': 1},
              },
            )
            .then((response) {
              lists = (response.data['hits'] as List<dynamic>)
                  .map(
                    (e) => ListSearchResult.fromJson(e as Map<String, dynamic>),
                  )
                  .toList();
            })
            .catchError((_) {}),
      if (searchLocations)
        api
            .get('/geocoding/search', queryParameters: {'q': q})
            .then((response) {
              final features =
                  response.data['features'] as List<dynamic>? ?? [];
              locations = features.map((f) {
                final props = f['properties'] as Map<String, dynamic>;
                final coords = f['geometry']['coordinates'] as List<dynamic>;
                final address = props['address'] as Map<String, dynamic>? ?? {};
                final rawName = props['name'] as String?;
                return LocationSearchResult(
                  name: (rawName != null && rawName.isNotEmpty)
                      ? rawName
                      : (props['display_name'] as String?) ?? '',
                  description: _buildLocationDescription(address),
                  lat: (coords[1] as num).toDouble(),
                  lon: (coords[0] as num).toDouble(),
                  category: (props['category'] as String?) ?? '',
                  type: props['type'] == 'administrative'
                      ? (props['addresstype'] as String?) ?? ''
                      : (props['type'] as String?) ?? '',
                );
              }).toList();
            })
            .catchError((_) {}),
      if (searchActors)
        api
            .get('/search/actor', queryParameters: {'q': q})
            .then((response) {
              actors = ((response.data['items'] as List<dynamic>?) ?? [])
                  .map(
                    (e) =>
                        ActorSearchResult.fromJson(e as Map<String, dynamic>),
                  )
                  .toList();
            })
            .catchError((_) {}),
    ]);

    if (_cancelled || state.query != q || state.category != category) return;

    state = state.copyWith(
      trails: trails,
      lists: lists,
      locations: locations,
      actors: actors,
      isLoading: false,
      error: null,
    );
  }

  static String _buildLocationDescription(Map<String, dynamic> address) {
    final parts = <String>[];
    final city =
        address['city'] ??
        address['town'] ??
        address['hamlet'] ??
        address['village'];
    if (city != null) parts.add(city as String);
    if (address['state'] != null) parts.add(address['state'] as String);
    if (address['country'] != null) parts.add(address['country'] as String);
    return parts.join(', ');
  }
}
