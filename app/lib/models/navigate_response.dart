import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:latlong2/latlong.dart';

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
  const factory NavigateResponse({
    required List<NavigateManeuver> maneuvers,
    required List<List<double>> shape,
  }) = _NavigateResponse;

  factory NavigateResponse.fromJson(Map<String, dynamic> json) =>
      _$NavigateResponseFromJson(json);
}

/// Extension exposing the shape as a list of [LatLng] values.
///
/// Coordinate order: shape entries are [lat, lon] pairs — confirmed against
/// the Phase-1 endpoint (`+server.ts` lines 120-122) which pushes
/// `[lat, lng]` after decoding the Valhalla polyline.
/// [0] = latitude, [1] = longitude.
extension NavigateResponseX on NavigateResponse {
  List<LatLng> get shapeAsLatLng => shape
      .where((p) => p.length >= 2)
      .map((p) => LatLng(p[0], p[1]))
      .toList();
}
