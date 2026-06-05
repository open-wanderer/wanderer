import 'package:freezed_annotation/freezed_annotation.dart';

part 'global_search_models.freezed.dart';
part 'global_search_models.g.dart';

enum GlobalSearchCategory { all, trails, lists, locations, actors }

@freezed
abstract class ListSearchResult with _$ListSearchResult {
  const factory ListSearchResult({
    required String id,
    required String author,
    @JsonKey(name: 'author_name') required String authorName,
    @JsonKey(name: 'author_avatar') required String authorAvatar,
    String? avatar,
    required String name,
    required String description,
    @JsonKey(name: 'elevation_gain') required double elevationGain,
    @JsonKey(name: 'elevation_loss') required double elevationLoss,
    required double distance,
    required double duration,
    String? domain,
    required bool public,
    required int trails,
    List<String>? shares,
    String? iri,
  }) = _ListSearchResult;

  factory ListSearchResult.fromJson(Map<String, dynamic> json) =>
      _$ListSearchResultFromJson(json);
}

@freezed
abstract class LocationSearchResult with _$LocationSearchResult {
  const factory LocationSearchResult({
    required String name,
    required String description,
    required double lat,
    required double lon,
    required String category,
    required String type,
  }) = _LocationSearchResult;
}

@freezed
abstract class ActorSearchResult with _$ActorSearchResult {
  const factory ActorSearchResult({
    @Default('') String id,
    @Default('') String username,
    @JsonKey(name: 'preferred_username') @Default('') String preferredUsername,
    @JsonKey(name: 'is_local') @Default(false) bool isLocal,
    @Default('') String domain,
    String? icon,
    @Default('') String iri,
  }) = _ActorSearchResult;

  factory ActorSearchResult.fromJson(Map<String, dynamic> json) =>
      _$ActorSearchResultFromJson(json);
}
