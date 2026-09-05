import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:gpx/gpx.dart';
import 'package:maplibre/maplibre.dart';
import 'package:wanderer/models/global_search_models.dart';
import 'package:wanderer/models/record.dart';
import 'package:wanderer/models/trail_summary.dart';
import 'package:wanderer/models/trail_sync_state.dart';

import 'actor.dart';
import 'category.dart';
import 'comment.dart';
import 'subcategory.dart';
import 'summit_log.dart';
import 'tag.dart';
import 'trail_like.dart';
import 'trail_share.dart';
import 'waypoint.dart';

part 'trail.freezed.dart';
part 'trail.g.dart';

class _NullableDateTimeConverter implements JsonConverter<DateTime?, Object?> {
  const _NullableDateTimeConverter();

  @override
  DateTime? fromJson(Object? value) {
    if (value == null || value == '') return null;
    return DateTime.parse(value as String);
  }

  @override
  Object? toJson(DateTime? date) => date?.toIso8601String();
}

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
    @_NullableDateTimeConverter() DateTime? date,
    @Default(false) bool public,
    @Default(false) bool completed,
    @Default(0) double distance,
    @JsonKey(name: 'elevation_gain') @Default(0) double elevationGain,
    @JsonKey(name: 'elevation_loss') @Default(0) double elevationLoss,
    @Default(0) double duration,
    // Moving time in seconds for trails recorded in the Wanderer app. No
    // @Default: absence is the meaningful "no moving time known" state
    // -- `duration` always means GPX-derived elapsed time.
    @JsonKey(name: 'moving_duration') double? movingDuration,
    @Default(TrailDifficulty.easy) TrailDifficulty difficulty,
    double? lat,
    double? lon,
    @JsonKey(name: 'max_lat') @Default(0) double maxLat,
    @JsonKey(name: 'max_lon') @Default(0) double maxLon,
    @JsonKey(name: 'min_lat') @Default(0) double minLat,
    @JsonKey(name: 'min_lon') @Default(0) double minLon,
    @Default(0) int thumbnail,
    @Default([]) List<String> photos,
    String? gpx,
    required DateTime created,
    required DateTime updated,
    String? category,
    String? subcategory,
    @Default([]) List<String> tags,
    String? polyline,
    String? domain,
    String? iri,
    @JsonKey(name: 'like_count') @Default(0) int likeCount,
    TrailExpand? expand,
    @Default("") String description,
    @Default("000000000000000") String author,

    /// This `Trail` instance was read from local storage — it is set only by
    /// [TrailEntity.toModel], where it is hardcoded `true`.
    ///
    /// **Provenance, not connectivity.** It answers "did this come off the
    /// device?", never "is the device offline right now?". For the latter,
    /// watch `onlineStatusProvider`. It was called `isOffline` until the two
    /// got conflated in a shipped bug: `TrailMap(offline: trail.isOffline)`
    /// selected the online basemap style for any trail that was not
    /// downloaded, so editing or viewing one with no connection rendered a
    /// blank map. Renamed to `isLocal` so the distinction is
    /// legible at every call site; every `TrailMap` mount now derives
    /// `offline:` from connectivity instead.
    ///
    /// The only live consumers are thumbnail-path selection in
    /// `trail_card.dart` and `trail_list_item.dart` (local file vs network
    /// image). **Never gate a destructive action, a badge, or tab visibility
    /// on this flag** — `TrailEntity.toModel()` hardcodes it `true` for
    /// every cached row, so it cannot distinguish one
    /// account's own local capture from another account's download.
    /// Destructive-action *availability* derives from library membership and
    /// authorship; destructive-action *scoping* derives from `owner`/account
    /// for local capture state and from `savedByUserIds` membership for a
    /// download. Unsynced and downloaded are not mutually exclusive.
    @Default(false) bool isLocal,
    @Default([]) List<String> localPhotos,

    /// Permanent local identity minted once at first local save. Device-local
    /// only — never serialized. `Trail.toJson()` feeds `util/trail/form_data.dart`,
    /// so this MUST carry `includeFromJson/includeToJson: false` or a
    /// device-local value would leak into the multipart body sent to the
    /// server.
    @JsonKey(includeFromJson: false, includeToJson: false) String? localId,

    /// This trail's upload/sync lifecycle. Device-local only — never
    /// serialized, same rationale as [localId].
    @JsonKey(includeFromJson: false, includeToJson: false)
    @Default(TrailSyncState.synced)
    TrailSyncState syncState,
  }) = _Trail;

  const Trail._();

  /// Placeholder trail whose only job is to size Skeletonizer's bones while a
  /// real one loads (`AsyncLoader.mockData`). The literal strings are never
  /// visible -- same convention as [FeedItem.mock].
  ///
  /// The absences are deliberate: no `expand` means `TrailPanel` mounts
  /// neither `TrailMap` nor `ElevationProfile`, and empty `photos` means no
  /// `PhotoCollage` -- so a fake trail never triggers a tile or image fetch.
  /// `TrailPanel` draws a skeleton-only placeholder where the map would go,
  /// so the missing route does not leave a hole. The id is a sentinel that
  /// cannot collide with a real record, because `TrailDetailScreen` matches
  /// it against the download library.
  ///
  /// Text length does not drive the layout: `TrailPanel` replaces the
  /// description with a fixed-size block while the skeleton is on.
  factory Trail.mock() => Trail(
    id: 'mock-trail-id',
    name: 'Mock Trail Name',
    description: 'Mock trail description.',
    date: DateTime(2024, 1, 1),
    created: DateTime(2024, 1, 1),
    updated: DateTime(2024, 1, 1),
    distance: 8000,
    elevationGain: 350,
    elevationLoss: 350,
    duration: 120,
  );

  LngLatBounds get bounds => LngLatBounds(
    longitudeEast: maxLon,
    longitudeWest: minLon,
    latitudeNorth: maxLat,
    latitudeSouth: minLat,
  );

  @override
  String get summaryAuthorName => expand?.author?.username ?? "Unknown";

  @override
  int get summaryDifficulty => difficulty.index;

  @override
  String get summaryAuthorAvatar => expand?.author?.icon ?? "";

  @override
  String? get summaryAuthorActorId =>
      expand?.author?.id ??
      (author.isNotEmpty && author != "000000000000000" ? author : null);

  @override
  DateTime? get summaryDate => date;

  @override
  String get summaryThumbnail => photos.isNotEmpty ? photos[0] : "";

  @override
  String? get categoryId => category;

  @override
  String? get subcategoryId => subcategory;

  @override
  List<String>? get summaryTags => expand?.tags?.map((t) => t.name).toList();

  @override
  List<String>? get summaryShares =>
      expand?.trailShareViaTrail?.map((s) => s.actor).toList();

  factory Trail.fromJson(Map<String, dynamic> json) => _$TrailFromJson(json);

  factory Trail.empty() =>
      Trail(id: '', name: '', created: DateTime.now(), updated: DateTime.now());
}

@freezed
abstract class TrailNear with _$TrailNear {
  const factory TrailNear({double? lat, double? lon, required double radius}) =
      _TrailNear;
}

enum SortOrder { asc, desc }

enum TrailFilterSort {
  created,
  date,
  difficulty,
  distance,
  duration,
  // ignore: constant_identifier_names
  elevation_gain,
  // ignore: constant_identifier_names
  elevation_loss,
  // ignore: constant_identifier_names
  like_count,
  name,
}

@freezed
abstract class TrailFilter with _$TrailFilter {
  const factory TrailFilter({
    required String q,
    required List<Category> category,
    @Default(<Subcategory>[]) List<Subcategory> subcategory,
    required List<Tag> tags,
    required List<int> difficulty, // 0, 1, 2
    ActorSearchResult? author,
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
    // Emitted server-side as an `id IN [...]` whitelist, but only when the
    // caller supplies the ids -- see `toFilterText`'s [offlineTrailIds].
    // `applyTrailFilter` carries a second, non-redundant half for unsynced
    // local captures, which have no server id to whitelist.
    @Default(false) bool offlineOnly,
    required TrailFilterSort
    sort, // "name" | "distance" | "elevation_gain" | "created"
    required SortOrder sortOrder,
  }) = _TrailFilter;

  const TrailFilter._();

  /// The character set a trail id must fall within to be interpolated into a
  /// Meilisearch filter string. PocketBase ids are alphanumeric; federated
  /// ids may add `-`/`_`. Anything outside this set (quotes, brackets,
  /// commas, whitespace) could break out of the `id IN [...]` literal, so a
  /// row carrying one is dropped rather than escaped.
  static final RegExp _kTrailIdPattern = RegExp(r'^[A-Za-z0-9_-]+$');

  /// Builds a Meilisearch filter string from this filter.
  ///
  /// Pass [actor] (the current user's actor ID) to include author, visibility,
  /// and liked constraints. Omit it (or pass null) when the caller does not
  /// need user-scoped filtering — e.g. profile trail searches where the server
  /// already enforces the author constraint.
  ///
  /// [offlineTrailIds] supplies the server ids of the trails held on this
  /// device, and is read ONLY when [offlineOnly] is set. It is a parameter
  /// rather than a field because the id set is a property of the device, not
  /// of the filter: the same `TrailFilter` instance is shared across the map,
  /// library and profile surfaces via one `trailFilterProvider` family, while
  /// the relevant id set differs per surface (author-narrowed on a profile,
  /// whole-library on the map).
  ///
  /// Callers should narrow the set to the author whose trails are being
  /// searched where they can. That is a privacy and payload measure, not a
  /// correctness one — `/profile/{handle}/trails` already ANDs
  /// `author = <actor>` server-side — and it matters most for federated
  /// profiles, whose requests are proxied verbatim to the origin instance.
  String toFilterText({
    String? actor,
    bool includeGeo = true,
    Set<String>? offlineTrailIds,
  }) {
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

    if (actor != null) {
      // Author
      if (author != null) {
        parts.add('author = ${author!.id}');
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
          if (showPrivate && (author == null || author!.id == actor)) {
            publicBlock = "($publicBlock OR author = $actor)";
          }
          visibilityOrBlocks.add(publicBlock);
        } else if (author == null || author!.id == actor) {
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

    // Categories and Subcategories (combined OR group, ID-based).
    // Mirrors the web's trail_store.ts filter logic:
    // - Categories whose subcategories are NOT all selected → category_id IN [...]
    // - All selected subcategories → subcategory_id IN [...]
    // The two clauses are OR-ed so a match on either counts.
    if (category.isNotEmpty || subcategory.isNotEmpty) {
      final selectedSubCategoryParentIds = subcategory
          .map((s) => s.category)
          .toSet();
      final categoriesWithoutSub = category
          .where((c) => !selectedSubCategoryParentIds.contains(c.id))
          .toList();

      final List<String> categoryParts = [];

      if (categoriesWithoutSub.isNotEmpty) {
        final catList = categoriesWithoutSub.map((c) => "'${c.id}'").join(", ");
        categoryParts.add('category_id IN [$catList]');
      }

      if (subcategory.isNotEmpty) {
        final subList = subcategory.map((s) => "'${s.id}'").join(", ");
        categoryParts.add('subcategory_id IN [$subList]');
      }

      if (categoryParts.isNotEmpty) {
        parts.add('(${categoryParts.join(" OR ")})');
      }
    }

    if (tags.isNotEmpty) {
      final tagList = tags.map((t) => "tags = '${t.name}'").join(" OR ");
      parts.add('($tagList)');
    }

    // Completed
    if (completed != null) {
      parts.add('completed = $completed');
    }

    // Offline-available: expressed as an explicit id whitelist. Meilisearch
    // has no field for "is this trail's data on THIS device" -- the server
    // does not track downloads at all (`savedByUserIds` is device-local, and
    // appears nowhere in db/ or web/) -- so the device names the ids itself.
    //
    // The empty set emits a clause that matches nothing rather than being
    // skipped. Skipping it would silently widen the search to every trail,
    // which is the opposite of what the chip asks for; callers that want to
    // avoid the round-trip entirely should short-circuit before calling.
    //
    // Ids are whitelisted to the PocketBase id alphabet before interpolation.
    // These strings come from local ObjectBox rows, so a corrupt or hostile
    // row must not be able to inject filter syntax into the query.
    if (offlineOnly) {
      final safeIds = (offlineTrailIds ?? const <String>{})
          .where((id) => _kTrailIdPattern.hasMatch(id))
          .toList();
      // `id IN ['']` rather than `id IN []`: a one-element array is
      // unambiguously valid filter syntax, and no trail id is ever the empty
      // string, so it reliably matches nothing. An empty array literal was
      // avoided because its acceptance is not worth depending on.
      parts.add(
        safeIds.isEmpty
            ? "id IN ['']"
            : 'id IN [${safeIds.map((id) => "'$id'").join(", ")}]',
      );
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
    @JsonKey(name: 'max_lat') required double maxLat,
    @JsonKey(name: 'min_lat') required double minLat,
    @JsonKey(name: 'max_lon') required double maxLon,
    @JsonKey(name: 'min_lon') required double minLon,
    @JsonKey(name: 'has_trails') @Default(false) bool hasTrails,
  }) = _TrailBoundingBox;

  factory TrailBoundingBox.fromJson(Map<String, dynamic> json) =>
      _$TrailBoundingBoxFromJson(json);
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
  "category_id",
  "subcategory_id",
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
