import 'package:objectbox/objectbox.dart';
import 'package:wanderer/entities/downloaded_tile_package_entity.dart';
import 'package:wanderer/models/region_catalog_entry.dart';
import 'package:wanderer/models/region_status.dart';

/// ObjectBox entity for a single catalog region — the app-wide offline tile
/// repository's core row. Persists every catalog-owned field from the last
/// successful `GET /api/v1/regions` fetch plus local-only
/// download bookkeeping, and exposes a computed local
/// download-lifecycle getter.
@Entity()
class RegionEntity {
  @Id()
  int obxId = 0;

  /// The server's `region_archives` record id. Informational ONLY — it is
  /// deliberately NOT indexed, NOT unique, and never a lookup or storage key.
  ///
  /// The backend re-mints this id: `ReconcileArchives`
  /// (`db/services/regions/reconcile.go`) deletes any `region_archives` row
  /// whose vector AND DEM files are both missing from disk, and the next
  /// build pass recreates the row through `findOrCreateRegionRecord` with a
  /// fresh PocketBase id. Keying local rows or on-disk archive directories
  /// off this value silently detached every download whenever the server
  /// regenerated: the re-minted id arrived as a brand-new row with no package
  /// links, while the old row kept the packages and the archives but vanished
  /// from the catalog. Always key on [path] instead.
  String id;

  // --- Catalog-owned fields (only these are overwritten by
  // applyCatalogEntry) ---
  String name;

  /// The region's materialized `path` (e.g. `canada.alberta.south`) — the
  /// stable identity of a region and the local primary key.
  ///
  /// This is what the backend itself treats as canonical: `region_archives`
  /// stores it in its `path` field, `findOrCreateRegionRecord` looks a region
  /// up by it, and every archive dir and download URL is named after it. It
  /// survives the record-id re-minting described on [id], so `@Unique(replace)`
  /// lives here — a `box.put()` for a known path replaces that row in place,
  /// carrying its `ToOne` package links and `lastDownloadedVersion` through a
  /// re-mint instead of duplicating the region.
  @Index()
  @Unique(onConflict: ConflictStrategy.replace)
  String path;

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

  // --- Local-only fields (never touched by applyCatalogEntry) ---

  /// Defaults `true` on creation; flipped `false` when a catalog
  /// fetch completes without this region's id appearing in the response.
  /// Downloaded files/packages are left on disk untouched when this flips.
  bool inCatalog = true;

  /// Set to the catalog's `version` value at the moment a vector download
  /// completes successfully. Compared against a fresh fetch's
  /// `version` to resolve [status] to [RegionStatus.updateAvailable].
  String? lastDownloadedVersion;

  // --- Catalog-owned enum shadows (explicit .code, never .index) ---

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
  /// an explicit-int enum shadow for anti-pattern consistency, no
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

  // --- Package relations (two independent nullable ToOnes, no
  // discriminator, no @Backlink) ---
  final vectorPackage = ToOne<DownloadedTilePackageEntity>();
  final demPackage = ToOne<DownloadedTilePackageEntity>();

  /// Computed local download-lifecycle status — getter only, no
  /// setter and no `@Property`, so ObjectBox excludes it from persistence.
  /// Region and package status can never drift out of sync as a result.
  ///
  /// The staleness branch checks only the VECTOR `version` because the API
  /// exposes no `dem_version`/DEM-build-date field — DEM archives
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
      case PackageStatus.error:
        return RegionStatus.error;
    }
  }

  RegionEntity({
    required this.path,
    this.id = '',
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
  /// element; the caller catches and skips such an entry.
  factory RegionEntity.fromCatalogEntry(RegionCatalogEntry entry) {
    if (entry.bbox.length != 4) {
      throw FormatException(
        'RegionCatalogEntry.bbox must have exactly 4 elements, '
        'got ${entry.bbox.length}',
      );
    }

    return RegionEntity(
      path: entry.path,
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

  /// Overwrites ONLY the catalog-owned fields — never touches
  /// [obxId], [path], [vectorPackage], [demPackage], or
  /// [lastDownloadedVersion]. Throws [FormatException] when
  /// `entry.bbox.length != 4`, matching [fromCatalogEntry].
  ///
  /// [id] IS overwritten: the caller matched this row by [path], so a differing
  /// `entry.id` means the backend re-minted the record id and this row must
  /// adopt it while keeping its download state. [path] is deliberately not
  /// reassigned — it is the key this row was found by, so writing it back
  /// could only ever be a no-op or a silent identity change.
  void applyCatalogEntry(RegionCatalogEntry entry) {
    if (entry.bbox.length != 4) {
      throw FormatException(
        'RegionCatalogEntry.bbox must have exactly 4 elements, '
        'got ${entry.bbox.length}',
      );
    }

    id = entry.id;
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
