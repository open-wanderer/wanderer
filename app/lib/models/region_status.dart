import 'package:json_annotation/json_annotation.dart';

/// Backend build-state for a region's vector archive (and, independently,
/// its DEM archive via [RegionCatalogEntry.demStatus]). Mirrors
/// `db/routes/regions_get.go`'s `status`/`dem_status` string values verbatim.
///
/// Persisted (on [RegionEntity]) via the explicit `.code` shadow-property
/// pattern — never `.index` — so a future non-append enum edit cannot
/// silently reinterpret an on-device row (REGN-02).
///
/// [absent] is an entity-only sentinel (never emitted by the API, no
/// `@JsonValue`) meaning "no DEM status present in the last fetch" — used as
/// `RegionEntity.demStatus`'s default when a catalog entry carries no
/// `dem_status` at all.
enum CatalogStatus {
  @JsonValue('building')
  building(0),
  @JsonValue('ready')
  ready(1),
  @JsonValue('error')
  error(2),
  absent(3);

  const CatalogStatus(this.code);
  final int code;
}

/// Local on-device download lifecycle for a region (D-10/D-12) — computed
/// from [DownloadedTilePackageEntity.status] via [RegionEntity.status],
/// never persisted directly.
enum RegionStatus {
  notDownloaded(0),
  downloading(1),
  downloaded(2),
  updateAvailable(3);

  const RegionStatus(this.code);
  final int code;
}

/// Per-package (vector or DEM) stored download status (D-10) — persisted on
/// [DownloadedTilePackageEntity] via the explicit `.code` shadow property.
enum PackageStatus {
  notDownloaded(0),
  downloading(1),
  downloaded(2);

  const PackageStatus(this.code);
  final int code;
}
