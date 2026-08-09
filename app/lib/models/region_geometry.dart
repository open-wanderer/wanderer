import 'package:freezed_annotation/freezed_annotation.dart';

part 'region_geometry.freezed.dart';
part 'region_geometry.g.dart';

/// Response shape of `GET /api/v1/regions/{path}/geometry` — the SvelteKit
/// cached-row read of the `region_geometry` PocketBase collection.
///
/// No `@JsonKey` renames are needed: all three server keys are already
/// snake-free.
@freezed
abstract class RegionGeometry with _$RegionGeometry {
  const factory RegionGeometry({
    required String path,

    /// Raw GeoJSON *geometry* object (`Polygon` or `MultiPolygon`) as stored
    /// in the `region_geometry.polygon` JSON column. This is NOT a Feature
    /// and NOT a FeatureCollection — the caller wraps it in a Feature before
    /// handing it to MapLibre.
    required Map<String, dynamic> polygon,

    /// `[minLon, minLat, maxLon, maxLat]` — the same order [RegionEntity]'s
    /// four discrete bbox columns and `generator.go`'s pmtiles extract
    /// arguments use.
    required List<double> bbox,
  }) = _RegionGeometry;

  factory RegionGeometry.fromJson(Map<String, dynamic> json) =>
      _$RegionGeometryFromJson(json);
}
