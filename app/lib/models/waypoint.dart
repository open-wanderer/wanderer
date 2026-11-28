import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:wanderer/models/converter/fa_icon_data_converter.dart';
import 'package:wanderer/models/record.dart';
import 'package:wanderer/models/photo_asset.dart';

part 'waypoint.freezed.dart';
part 'waypoint.g.dart';

@freezed
abstract class Waypoint with _$Waypoint, RecordFunctions {
  const factory Waypoint({
    required String id,
    @Default('waypoints') String collectionId,
    @Default('waypoints') String collectionName,

    @Default("") String? name,
    @Default("") String? description,
    required double lat,
    required double lon,
    @JsonKey(name: 'distance_from_start') double? distanceFromStart,
    @FaIconDataConverter() @Default(FontAwesomeIcons.circle) FaIconData icon,
    @Default([]) List<String> photos,
    @Default("000000000000000") String author,
    String? trail,
    WaypointExpand? expand,
    required DateTime created,
    required DateTime updated,

    // Non-serializable local fields
    @JsonKey(includeFromJson: false, includeToJson: false) dynamic marker,

    @JsonKey(includeFromJson: false, includeToJson: false)
    @Default([])
    List<String> localPhotos,

    @JsonKey(includeFromJson: false, includeToJson: false)
    @Default([])
    List<String> removedPendingPhotos,

    /// Carries list identity for a waypoint that has no server id yet
    /// yet. Device-local only, never serialized.
    @JsonKey(includeFromJson: false, includeToJson: false) String? localKey,
  }) = _Waypoint;

  const Waypoint._();

  /// The identity every in-memory waypoint list operation must match on,
  /// once ids can legitimately be empty: the server id when present,
  /// otherwise the local key.
  String get listKey => id.isNotEmpty ? id : (localKey ?? '');

  factory Waypoint.fromJson(Map<String, dynamic> json) =>
      _$WaypointFromJson(json);
}

@freezed
abstract class WaypointExpand with _$WaypointExpand {
  const factory WaypointExpand({
    @JsonKey(name: 'waypoint_assets_via_waypoint')
    List<PhotoAssetLink>? photoAssets,
  }) = _WaypointExpand;

  factory WaypointExpand.fromJson(Map<String, dynamic> json) =>
      _$WaypointExpandFromJson(json);
}
