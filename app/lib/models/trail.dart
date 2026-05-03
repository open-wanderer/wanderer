import 'package:freezed_annotation/freezed_annotation.dart';
import 'tag.dart';
import 'category.dart';
import 'waypoint.dart';
import 'summit_log.dart';
import 'actor.dart';
import 'comment.dart';
import 'trail_share.dart';
import 'trail_like.dart';
import 'package:gpx/gpx.dart';

part 'trail.freezed.dart';
part 'trail.g.dart';

enum TrailDifficulty {
  @JsonValue('easy')
  easy,
  @JsonValue('moderate')
  moderate,
  @JsonValue('difficult')
  difficult,
}

@freezed
abstract class TrailExpand with _$TrailExpand {
  const factory TrailExpand({
    List<Tag>? tags,
    Category? category,
    @JsonKey(name: 'waypoints_via_trail') List<Waypoint>? waypointsViaTrail,
    @JsonKey(name: 'summit_logs_via_trail') List<SummitLog>? summitLogsViaTrail,
    Actor? author,
    @JsonKey(name: 'comments_via_trail') List<Comment>? commentsViaTrail,
    @JsonKey(name: 'gpx_data') String? gpxData,
    @JsonKey(includeFromJson: false, includeToJson: false) Gpx? gpx,
    @JsonKey(name: 'trail_share_via_trail')
    List<TrailShare>? trailShareViaTrail,
    @JsonKey(name: 'trail_like_via_trail') List<TrailLike>? trailLikeViaTrail,
  }) = _TrailExpand;

  factory TrailExpand.fromJson(Map<String, dynamic> json) =>
      _$TrailExpandFromJson(json);
}

@freezed
abstract class Trail with _$Trail {
  const factory Trail({
    String? id,
    required String name,
    String? location,
    String? date,
    @Default(false) bool public,
    @Default(0) double distance,
    @JsonKey(name: 'elevation_gain') @Default(0) double elevationGain,
    @JsonKey(name: 'elevation_loss') @Default(0) double elevationLoss,
    @Default(0) double duration,
    @Default(TrailDifficulty.easy) TrailDifficulty difficulty,
    double? lat,
    double? lon,
    @Default(0) int thumbnail,
    @Default([]) List<String> photos,
    String? gpx,
    String? created,
    String? updated,
    String? category,
    @Default([]) List<String> tags,
    String? polyline,
    String? domain,
    String? iri,
    @JsonKey(name: 'like_count') @Default(0) int likeCount,
    TrailExpand? expand,
    @Default("") String description,
    @Default("000000000000000") String author,
  }) = _Trail;

  factory Trail.fromJson(Map<String, dynamic> json) => _$TrailFromJson(json);
}

@freezed
abstract class GeoLocation with _$GeoLocation {
  const factory GeoLocation({required double lat, required double lng}) =
      _GeoLocation;

  factory GeoLocation.fromJson(Map<String, dynamic> json) =>
      _$GeoLocationFromJson(json);
}

@freezed
abstract class TrailSearchResult with _$TrailSearchResult {
  const factory TrailSearchResult({
    required String id,
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

  factory TrailSearchResult.fromJson(Map<String, dynamic> json) =>
      _$TrailSearchResultFromJson(json);

  // Added logic to match your searchResultToTrailList requirements
  const TrailSearchResult._();

  Trail toTrail() {
    final createdIso = DateTime.fromMillisecondsSinceEpoch(
      created * 1000,
    ).toIso8601String();
    final dateIso = DateTime.fromMillisecondsSinceEpoch(
      date * 1000,
    ).toIso8601String();

    return Trail(
      id: id,
      name: name,
      author: author,
      description: description,
      location: location,
      distance: distance,
      elevationGain: elevationGain,
      elevationLoss: elevationLoss,
      duration: duration,
      difficulty: difficulty == 0
          ? TrailDifficulty.easy
          : difficulty == 1
          ? TrailDifficulty.moderate
          : TrailDifficulty.difficult,
      lat: geo.lat,
      lon: geo.lng,
      public: public,
      photos: thumbnail.isNotEmpty ? [thumbnail] : [],
      tags: tags ?? [],
      category: category,
      created: createdIso,
      date: dateIso,
      gpx: gpx,
      polyline: polyline,
      domain: domain,
      iri: iri,
      likeCount: likeCount,
      expand: TrailExpand(
        trailShareViaTrail: shares
            ?.map(
              (s) => TrailShare(
                actor: s,
                trail: id,
                permission: TrailPermission.view,
              ),
            )
            .toList(),
        trailLikeViaTrail: likes
            ?.map((l) => TrailLike(actor: l, trail: id))
            .toList(),
      ),
    );
  }
}

@freezed
abstract class TrailNear with _$TrailNear {
  const factory TrailNear({double? lat, double? lon, required double radius}) =
      _TrailNear;
}

@freezed
abstract class TrailFilter with _$TrailFilter {
  const factory TrailFilter({
    required String q,
    required List<String> category,
    required List<String> tags,
    required List<int> difficulty, // 0, 1, 2
    String? author,
    bool? public,
    bool? shared,
    bool? private,
    required TrailNear near,
    required double distanceMin,
    required double distanceMax,
    required double distanceLimit,
    required double elevationGainMin,
    required double elevationGainMax,
    required double elevationGainLimit,
    required double elevationLossMin,
    required double elevationLossMax,
    required double elevationLossLimit,
    String? startDate,
    String? endDate,
    bool? completed,
    bool? liked,
    required String sort, // "name" | "distance" | "elevation_gain" | "created"
    required String sortOrder, // "+" | "-"
  }) = _TrailFilter;
}

@freezed
abstract class TrailFilterValues with _$TrailFilterValues {
  const factory TrailFilterValues({
    required double minDistance,
    required double maxDistance,
    required double minElevationGain,
    required double maxElevationGain,
    required double minElevationLoss,
    required double maxElevationLoss,
    required double minDuration,
    required double maxDuration,
  }) = _TrailFilterValues;
}

@freezed
abstract class TrailBoundingBox with _$TrailBoundingBox {
  const factory TrailBoundingBox({
    required double maxLat,
    required double minLat,
    required double maxLon,
    required double minLon,
  }) = _TrailBoundingBox;
}
