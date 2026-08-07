import 'package:objectbox/objectbox.dart';
import 'package:wanderer/models/region_status.dart';

/// ObjectBox entity for a single downloaded tile package (vector OR DEM) for
/// a region. Standalone row, addressed only via its owning [RegionEntity]'s
/// `ToOne` (`vectorPackage`/`demPackage`) — no business string id of its own.
///
/// A row only comes into existence once the download engine
/// actually begins downloading that specific package; it is never created
/// speculatively for a `building`/`error`/absent catalog status.
@Entity()
class DownloadedTilePackageEntity {
  @Id()
  int obxId = 0;

  @Transient()
  PackageStatus status = PackageStatus.notDownloaded;

  /// Persists [status] via its explicit `.code` value — never `.index` — so
  /// a future non-append enum edit cannot silently reinterpret an on-device
  /// row. Decoded by value-lookup, not positional indexing;
  /// an out-of-range value falls back to [PackageStatus.notDownloaded].
  int get dbStatus => status.code;

  set dbStatus(int value) {
    status = PackageStatus.values.firstWhere(
      (s) => s.code == value,
      orElse: () => PackageStatus.notDownloaded,
    );
  }

  String? localFilePath;

  @Property(type: PropertyType.dateUtc)
  DateTime? downloadedAtUtc;

  int? sizeBytesOnDisk;

  DownloadedTilePackageEntity({
    this.obxId = 0,
    this.status = PackageStatus.notDownloaded,
    this.localFilePath,
    this.downloadedAtUtc,
    this.sizeBytesOnDisk,
  });
}
