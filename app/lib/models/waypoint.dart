import 'package:font_awesome_flutter/font_awesome_flutter.dart';
import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:wanderer/models/converter/fa_icon_data_converter.dart';

part 'waypoint.freezed.dart';
part 'waypoint.g.dart';

@freezed
abstract class Waypoint with _$Waypoint {
  const factory Waypoint({
    String? id,
    @Default("") String? name,
    @Default("") String? description,
    required double lat,
    required double lon,
    double? distanceFromStart,
    @FaIconDataConverter() @Default(FontAwesomeIcons.circle) FaIconData icon,
    @Default([]) List<String> photos,
    @Default("000000000000000") String author,
    String? trail,

    // Non-serializable local fields
    @JsonKey(includeFromJson: false, includeToJson: false) dynamic marker,

    @JsonKey(includeFromJson: false, includeToJson: false)
    @Default([])
    List<dynamic> localPhotos,
  }) = _Waypoint;

  factory Waypoint.fromJson(Map<String, dynamic> json) =>
      _$WaypointFromJson(json);
}
