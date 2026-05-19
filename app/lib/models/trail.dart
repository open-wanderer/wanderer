import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:gpx/gpx.dart';
import 'package:wanderer/models/record.dart';
import 'package:wanderer/models/trail_summary.dart';

import 'actor.dart';
import 'category.dart';
import 'comment.dart';
import 'summit_log.dart';
import 'tag.dart';
import 'trail_like.dart';
import 'trail_share.dart';
import 'waypoint.dart';

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
abstract class Trail with _$Trail, RecordFunctions implements TrailSummary {
  const factory Trail({
    required String id,
    @Default('trails') String collectionId,
    required String name,
    String? location,
    DateTime? date,
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
    required DateTime created,
    required DateTime updated,
    String? category,
    @Default([]) List<String> tags,
    String? polyline,
    String? domain,
    String? iri,
    @JsonKey(name: 'like_count') @Default(0) int likeCount,
    TrailExpand? expand,
    @Default("") String description,
    @Default("000000000000000") String author,

    @Default(false) bool isOffline,
    @Default([]) List<String> localPhotos,
    @Default([]) List<String> pmTiles,
  }) = _Trail;

  const Trail._();

  @override
  String get summaryAuthorName => expand?.author?.username ?? "Unknown";

  @override
  int get summaryDifficulty => difficulty.index;

  @override
  String get summaryAuthorAvatar => expand?.author?.icon ?? "";

  @override
  DateTime? get summaryDate => date;

  @override
  String get summaryThumbnail => photos.isNotEmpty ? photos[0] : "";

  @override
  String get summaryCategory => expand?.category?.name ?? "";

  @override
  List<String>? get summaryTags => expand?.tags?.map((t) => t.name).toList();

  @override
  List<String>? get summaryShares =>
      expand?.trailShareViaTrail?.map((s) => s.actor).toList();

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
    required List<Category> category,
    required List<Tag> tags,
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
    DateTime? startDate,
    DateTime? endDate,
    bool? completed,
    bool? liked,
    required String sort, // "name" | "distance" | "elevation_gain" | "created"
    required String sortOrder, // "+" | "-"
  }) = _TrailFilter;

  const TrailFilter._();

  String toFilterText({required String actor, bool includeGeo = true}) {
    List<String> parts = [];

    // Basic Numeric Filters
    parts.add('distance >= ${distanceMin.floor()}');
    parts.add('elevation_gain >= ${elevationGainMin.floor()}');
    parts.add('elevation_loss >= ${elevationLossMin.floor()}');

    if (distanceMax < distanceLimit) {
      parts.add('distance <= ${distanceMax.ceil()}');
    }
    if (elevationGainMax < elevationGainLimit) {
      parts.add('elevation_gain <= ${elevationGainMax.ceil()}');
    }
    if (elevationLossMax < elevationLossLimit) {
      parts.add('elevation_loss <= ${elevationLossMax.ceil()}');
    }

    // Difficulty
    if (difficulty.isNotEmpty) {
      parts.add('difficulty IN [${difficulty.join(",")}]');
    }

    // Author
    if (author != null && author!.isNotEmpty) {
      parts.add('author = $author');
    }

    // Visibility Logic (Public / Private / Shared)
    if (public != null || private != null || shared != null) {
      List<String> visibilityOrBlocks = [];

      final bool showPublic = public ?? true;
      final bool showPrivate = private ?? true;
      final bool showShared = shared ?? false;

      if (showPublic) {
        String publicBlock = "public = TRUE";
        // If showing private trails and user is the author (or no specific author requested)
        if (showPrivate &&
            (author == null || author!.isEmpty || author == actor)) {
          publicBlock = "($publicBlock OR author = $actor)";
        }
        visibilityOrBlocks.add(publicBlock);
      } else if (author == null || author!.isEmpty || author == actor) {
        visibilityOrBlocks.add("(public = FALSE AND author = $actor)");
      }

      if (shared != null) {
        if (showShared) {
          visibilityOrBlocks.add("shares = $actor");
        } else {
          parts.add("NOT shares = $actor");
        }
      }

      if (visibilityOrBlocks.isNotEmpty) {
        parts.add("(${visibilityOrBlocks.join(" OR ")})");
      }
    }

    // Liked
    if (liked == true) {
      parts.add('likes = $actor');
    }

    // Dates
    if (startDate != null) {
      final seconds = startDate!.millisecondsSinceEpoch ~/ 1000;
      parts.add('date >= $seconds');
    }
    if (endDate != null) {
      final seconds = endDate!.millisecondsSinceEpoch ~/ 1000;
      parts.add('date <= $seconds');
    }

    // Categories and Tags
    if (category.isNotEmpty) {
      final catList = category.map((c) => "'${c.name}'").join(", ");
      parts.add('category IN [$catList]');
    }

    if (tags.isNotEmpty) {
      final tagList = tags.map((t) => "tags = '${t.name}'").join(" OR ");
      parts.add('($tagList)');
    }

    // Completed
    if (completed != null) {
      parts.add('completed = $completed');
    }

    // Geo Location
    if (includeGeo && near.lat != null && near.lon != null) {
      parts.add('_geoRadius(${near.lat}, ${near.lon}, ${near.radius})');
    }

    return parts.join(" AND ");
  }
}

@freezed
abstract class TrailFilterValues with _$TrailFilterValues {
  const factory TrailFilterValues({
    @JsonKey(name: 'min_distance') required double minDistance,
    @JsonKey(name: 'max_distance') required double maxDistance,
    @JsonKey(name: 'min_elevation_gain') required double minElevationGain,
    @JsonKey(name: 'max_elevation_gain') required double maxElevationGain,
    @JsonKey(name: 'min_elevation_loss') required double minElevationLoss,
    @JsonKey(name: 'max_elevation_loss') required double maxElevationLoss,
    @JsonKey(name: 'min_duration') required double minDuration,
    @JsonKey(name: 'max_duration') required double maxDuration,
  }) = _TrailFilterValues;

  factory TrailFilterValues.fromJson(Map<String, dynamic> json) =>
      _$TrailFilterValuesFromJson(json);
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

const List<String> defaultTrailSearchAttributes = [
  "id",
  "author",
  "author_name",
  "author_avatar",
  "name",
  "description",
  "location",
  "distance",
  "elevation_gain",
  "elevation_loss",
  "duration",
  "difficulty",
  "category",
  "completed",
  "date",
  "created",
  "public",
  "thumbnail",
  "domain",
  "gpx",
  "tags",
  "like_count",
  "shares",
  "iri",
  "_geo",
];
