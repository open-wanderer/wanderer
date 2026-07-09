// ignore_for_file: invalid_annotation_target

import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:maplibre/maplibre.dart';

part 'navigate_response.freezed.dart';
part 'navigate_response.g.dart';

@freezed
abstract class NavigateManeuver with _$NavigateManeuver {
  const factory NavigateManeuver({
    required String instruction,
    required double length,
    @JsonKey(name: 'begin_shape_index') required int beginShapeIndex,
    @Default(0.0) double bearing,
    @Default(0) int type,
  }) = _NavigateManeuver;

  factory NavigateManeuver.fromJson(Map<String, dynamic> json) =>
      _$NavigateManeuverFromJson(json);
}

@freezed
abstract class NavigateResponse with _$NavigateResponse {
  @JsonSerializable(explicitToJson: true)
  const factory NavigateResponse({
    required List<NavigateManeuver> maneuvers,
    required List<List<double>> shape,
  }) = _NavigateResponse;

  factory NavigateResponse.fromJson(Map<String, dynamic> json) =>
      _$NavigateResponseFromJson(json);
}

/// Extension exposing the shape as a list of [Geographic] values.
///
/// Coordinate order: shape entries are [lat, lon] pairs — confirmed against
/// the Phase-1 endpoint (`+server.ts` lines 120-122) which pushes
/// `[lat, lng]` after decoding the Valhalla polyline.
/// [0] = latitude, [1] = longitude.
extension NavigateResponseX on NavigateResponse {
  List<Geographic> get shapeAsGeographic => shape
      .where((p) => p.length >= 2)
      .map((p) => Geographic(lat: p[0], lon: p[1]))
      .toList();
}
