import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:wanderer/models/list_summary.dart';
import 'package:wanderer/models/record.dart';
import 'package:wanderer/models/trail_summary.dart';

part 'global_search_models.freezed.dart';
part 'global_search_models.g.dart';

enum GlobalSearchCategory { all, trails, lists, locations, actors }

@freezed
abstract class GeoLocation with _$GeoLocation {
  const factory GeoLocation({required double lat, required double lng}) =
      _GeoLocation;

  factory GeoLocation.fromJson(Map<String, dynamic> json) =>
      _$GeoLocationFromJson(json);
}

@freezed
abstract class TrailSearchResult
    with _$TrailSearchResult, RecordFunctions
    implements TrailSummary {
  const factory TrailSearchResult({
    required String id,
    @Default('trails') String collectionId,
    required String author,
    @JsonKey(name: 'author_name') required String authorName,
    @JsonKey(name: 'author_avatar') required String authorAvatar,
    required String name,
    required String description,
    required String location,
    required double distance,
    @JsonKey(name: 'elevation_gain') required double elevationGain,
    @JsonKey(name: 'elevation_loss') required double elevationLoss,
    required double duration,
    required int difficulty, // 0 | 1 | 2
    required String category,
    required bool completed,
    required int date,
    required int created,
    required bool public,
    required String thumbnail,
    String? polyline,
    List<String>? likes,
    @JsonKey(name: 'like_count') required int likeCount,
    List<String>? shares,
    List<String>? tags,
    String? domain,
    String? iri,
    required String gpx,
    @JsonKey(name: '_geo') required GeoLocation geo,
  }) = _TrailSearchResult;

  const TrailSearchResult._();

  @override
  String get summaryAuthorName => authorName;

  @override
  String get summaryAuthorAvatar => authorAvatar;

  @override
  DateTime? get summaryDate => DateTime.fromMillisecondsSinceEpoch(date * 1000);

  @override
  int get summaryDifficulty => difficulty;

  @override
  String get summaryThumbnail => thumbnail;

  @override
  String get summaryCategory => category;

  @override
  List<String>? get summaryTags => tags;

  @override
  List<String>? get summaryShares => shares;

  @override
  bool get isOffline => false;

  @override
  List<String> get localPhotos => [];

  factory TrailSearchResult.fromJson(Map<String, dynamic> json) =>
      _$TrailSearchResultFromJson(json);

  factory TrailSearchResult.mock() => const TrailSearchResult(
    id: 'mock-trail-id',
    author: 'mock-author-id',
    authorName: 'Mock Author',
    authorAvatar: '',
    name: 'Mock Trail Name',
    description: 'A placeholder trail for skeleton loading',
    location: 'Mock Location',
    distance: 9500.0,
    elevationGain: 420.0,
    elevationLoss: 420.0,
    duration: 150.0,
    difficulty: 1,
    category: '',
    completed: false,
    date: 0,
    created: 0,
    public: false,
    thumbnail: '',
    likeCount: 0,
    gpx: '',
    geo: GeoLocation(lat: 0, lng: 0),
  );
}

@freezed
abstract class ListSearchResult
    with _$ListSearchResult, RecordFunctions
    implements ListSummary {
  const factory ListSearchResult({
    required String id,
    @Default('lists') String collectionId,
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

  const ListSearchResult._();

  @override
  int get trailCount => trails;

  @override
  String get summaryAuthorName => authorName;

  @override
  String get summaryAuthorAvatar => authorAvatar;

  factory ListSearchResult.fromJson(Map<String, dynamic> json) =>
      _$ListSearchResultFromJson(json);

  factory ListSearchResult.mock() => const ListSearchResult(
    id: 'mock-list-id',
    author: 'mock-author-id',
    authorName: 'Mock Author',
    authorAvatar: '',
    name: 'Mock Trail List',
    description: 'A placeholder list for skeleton loading',
    elevationGain: 450.0,
    elevationLoss: 450.0,
    distance: 12000.0,
    duration: 180.0,
    public: true,
    trails: 5,
  );
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
