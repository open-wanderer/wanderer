import 'package:freezed_annotation/freezed_annotation.dart';
import 'package:wanderer/models/region_status.dart';

part 'region_catalog_entry.freezed.dart';
part 'region_catalog_entry.g.dart';

/// A single element of the `GET /api/v1/regions` catalog response
/// (`db/routes/regions_get.go`'s `RegionsList`). The endpoint returns a bare
/// top-level JSON array — there is no list-wrapper class; Plan 02 parses the
/// response element-by-element.
///
/// Field presence mirrors the Go handler exactly: `id`/`name`/`bbox`/`status`
/// are always set; `version` only once a build has succeeded;
/// `vector_url`/`vector_size` only when `status == "ready"` AND the file
/// exists on disk; `dem_status` only if a DEM build was ever attempted;
/// `dem_url`/`dem_size` only when `dem_status == "ready"` AND the file
/// exists; `error` only when `status == "error"`.
@freezed
abstract class RegionCatalogEntry with _$RegionCatalogEntry {
  const factory RegionCatalogEntry({
    required String id,

    /// The region's materialized `path` (e.g. `canada.alberta.south`) — the
    /// key the backend names every on-disk archive dir and download URL after
    /// (`RegionsList` in `db/routes/regions_get.go`). Distinct from [id],
    /// which is the opaque PocketBase record id used only for local storage.
    required String path,
    required String name,

    /// `[minLon, minLat, maxLon, maxLat]` — matches `generator.go`'s pmtiles
    /// extract argument order and the region archive path builders.
    required List<double> bbox,
    required CatalogStatus status,
    String? version,
    @JsonKey(name: 'vector_url') String? vectorUrl,
    @JsonKey(name: 'vector_size') int? vectorSize,
    @JsonKey(name: 'dem_status') CatalogStatus? demStatus,
    @JsonKey(name: 'dem_url') String? demUrl,
    @JsonKey(name: 'dem_size') int? demSize,
    String? error,
  }) = _RegionCatalogEntry;

  factory RegionCatalogEntry.fromJson(Map<String, dynamic> json) =>
      _$RegionCatalogEntryFromJson(json);
}
