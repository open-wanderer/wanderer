import 'package:objectbox/objectbox.dart';
import 'package:wanderer/entities/downloaded_tile_package_entity.dart';
import 'package:wanderer/models/region_catalog_entry.dart';
import 'package:wanderer/models/region_status.dart';

/// ObjectBox entity for a single catalog region — the app-wide offline tile
/// repository's core row. Persists every catalog-owned field from the last
/// successful `GET /api/v1/regions` fetch (D-01/D-04/D-05) plus local-only
/// download bookkeeping (D-06/D-08/D-11), and exposes a computed local
/// download-lifecycle getter (D-12).
@Entity()
class RegionEntity {
  @Id()
  int obxId = 0;

  /// The catalog business id (e.g. `"de-nrw"`). `@Unique(replace)` is what
  /// makes Plan 02's upsert-by-id cheap: `box.put()` with a matching unique
  /// id replaces the row in place rather than erroring or duplicating.
  @Index()
  @Unique(onConflict: ConflictStrategy.replace)
  String id;

  // --- Catalog-owned fields (D-01: only these are overwritten by
  // applyCatalogEntry) ---
  String name;

  /// bbox stored as four discrete fields (not a List) — ObjectBox has no
  /// native `List<double>` column support for this shape; matches
  /// `db/routes/regions_get.go`'s `[minLon, minLat, maxLon, maxLat]` order.
  double minLon;
  double minLat;
  double maxLon;
  double maxLat;

  String? version;
  String? vectorUrl;
  int? vectorSize;
  String? demUrl;
  int? demSize;
  String? error;

  // --- Local-only fields (D-01: never touched by applyCatalogEntry) ---

  /// Defaults `true` on creation (D-08); flipped `false` when a catalog
  /// fetch completes without this region's id appearing in the response.
  /// Downloaded files/packages are left on disk untouched when this flips.
  bool inCatalog = true;

  /// Set to the catalog's `version` value at the moment a vector download
  /// completes successfully (Phase 23). Compared against a fresh fetch's
  /// `version` to resolve [status] to [RegionStatus.updateAvailable] (D-06).
  String? lastDownloadedVersion;

  // --- Catalog-owned enum shadows (D-05: explicit .code, never .index) ---

  @Transient()
  CatalogStatus catalogStatus = CatalogStatus.building;

  int get dbCatalogStatus => catalogStatus.code;

  set dbCatalogStatus(int value) {
    catalogStatus = CatalogStatus.values.firstWhere(
      (s) => s.code == value,
      orElse: () => CatalogStatus.building,
    );
  }

  /// Uses the [CatalogStatus.absent] sentinel (rather than a nullable
  /// column) when the last fetch carried no `dem_status` at all — keeps this
  /// an explicit-int enum shadow for anti-pattern consistency (D-05), no
  /// nullable ObjectBox shadow needed.
  @Transient()
  CatalogStatus demStatus = CatalogStatus.absent;

  int get dbDemStatus => demStatus.code;

  set dbDemStatus(int value) {
    demStatus = CatalogStatus.values.firstWhere(
      (s) => s.code == value,
      orElse: () => CatalogStatus.absent,
    );
  }

  // --- Package relations (D-11: two independent nullable ToOnes, no
  // discriminator, no @Backlink) ---
  final vectorPackage = ToOne<DownloadedTilePackageEntity>();
  final demPackage = ToOne<DownloadedTilePackageEntity>();

  /// Computed local download-lifecycle status (D-12) — getter only, no
  /// setter and no `@Property`, so ObjectBox excludes it from persistence.
  /// Region and package status can never drift out of sync as a result.
  ///
  /// The staleness branch checks only the VECTOR `version` because the API
  /// exposes no `dem_version`/DEM-build-date field (D-07) — DEM archives
  /// have no `updateAvailable` concept, only the vector package can go
  /// stale. DEM state is deliberately NOT folded into this getter — the DEM
  /// package lifecycle is independent (matches "[quick-260711-lzb] DEM tile
  /// lifecycle kept fully independent from the vector tile lifecycle; a DEM
  /// failure must never regress or block the vector basemap"), so a
  /// downloaded vector with a not-yet-downloaded DEM still reads
  /// `downloaded`; DEM presence is queried separately via
  /// `demPackage.target` / `demStatus`.
  RegionStatus get status {
    final vectorTarget = vectorPackage.target;
    if (vectorTarget == null) return RegionStatus.notDownloaded;

    switch (vectorTarget.status) {
      case PackageStatus.downloading:
        return RegionStatus.downloading;
      case PackageStatus.downloaded:
        final isStale =
            catalogStatus == CatalogStatus.ready &&
            version != null &&
            version != lastDownloadedVersion;
        return isStale ? RegionStatus.updateAvailable : RegionStatus.downloaded;
      case PackageStatus.notDownloaded:
        return RegionStatus.notDownloaded;
    }
  }

  RegionEntity({
    required this.id,
    required this.name,
    this.minLon = 0,
    this.minLat = 0,
    this.maxLon = 0,
    this.maxLat = 0,
    this.version,
    this.vectorUrl,
    this.vectorSize,
    this.demUrl,
    this.demSize,
    this.error,
    this.inCatalog = true,
    this.lastDownloadedVersion,
    this.catalogStatus = CatalogStatus.building,
    this.demStatus = CatalogStatus.absent,
  });

  /// Constructs a fresh entity from a parsed catalog entry (Plan 02's
  /// insert-half of the upsert). Leaves [lastDownloadedVersion] null and
  /// both `ToOne` targets unset. Throws [FormatException] when
  /// `entry.bbox.length != 4` — defends against a malformed/hostile catalog
  /// element (T-22-01); Plan 02 catches and skips such an entry.
  factory RegionEntity.fromCatalogEntry(RegionCatalogEntry entry) {
    if (entry.bbox.length != 4) {
      throw FormatException(
        'RegionCatalogEntry.bbox must have exactly 4 elements, '
        'got ${entry.bbox.length}',
      );
    }

    return RegionEntity(
      id: entry.id,
      name: entry.name,
      minLon: entry.bbox[0],
      minLat: entry.bbox[1],
      maxLon: entry.bbox[2],
      maxLat: entry.bbox[3],
      version: entry.version,
      vectorUrl: entry.vectorUrl,
      vectorSize: entry.vectorSize,
      demUrl: entry.demUrl,
      demSize: entry.demSize,
      error: entry.error,
      inCatalog: true,
      catalogStatus: entry.status,
      demStatus: entry.demStatus ?? CatalogStatus.absent,
    );
  }

  /// Overwrites ONLY the catalog-owned fields (D-01) — never touches
  /// [obxId], [id], [vectorPackage], [demPackage], or
  /// [lastDownloadedVersion]. Throws [FormatException] when
  /// `entry.bbox.length != 4`, matching [fromCatalogEntry] (T-22-01).
  void applyCatalogEntry(RegionCatalogEntry entry) {
    if (entry.bbox.length != 4) {
      throw FormatException(
        'RegionCatalogEntry.bbox must have exactly 4 elements, '
        'got ${entry.bbox.length}',
      );
    }

    name = entry.name;
    minLon = entry.bbox[0];
    minLat = entry.bbox[1];
    maxLon = entry.bbox[2];
    maxLat = entry.bbox[3];
    catalogStatus = entry.status;
    demStatus = entry.demStatus ?? CatalogStatus.absent;
    version = entry.version;
    vectorUrl = entry.vectorUrl;
    vectorSize = entry.vectorSize;
    demUrl = entry.demUrl;
    demSize = entry.demSize;
    error = entry.error;
    inCatalog = true;
  }
}
