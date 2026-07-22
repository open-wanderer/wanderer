import 'package:dio/dio.dart';
import 'package:objectbox/objectbox.dart';
import 'package:wanderer/entities/region_entity.dart';
import 'package:wanderer/models/region_catalog_entry.dart';
import 'package:wanderer/objectbox.g.dart';

/// Typed error thrown by [fetchRegionCatalog]/[RegionRepository.fetchCatalog]
/// on any network or parse failure (D-03). Callers must not silently swallow
/// a catalog fetch failure the way `subcategory_provider.dart` does — Phase
/// 24's Settings/Regions screen decides how to surface this to the user.
class RegionCatalogException implements Exception {
  const RegionCatalogException(this.message, [this.cause]);

  final String message;
  final Object? cause;

  @override
  String toString() => 'RegionCatalogException: $message'
      '${cause != null ? ' (cause: $cause)' : ''}';
}

/// Parses a bare `GET /api/v1/regions` JSON array (`db/routes/regions_get.go`
/// -- no `{items:[...]}` wrapper) into [RegionCatalogEntry] values.
///
/// A non-`List` payload throws [RegionCatalogException] (unexpected shape).
/// Individual malformed elements are caught and skipped rather than aborting
/// the whole parse (T-22-05) -- one bad/hostile array element cannot corrupt
/// or block the rest of the catalog.
List<RegionCatalogEntry> parseRegionCatalog(dynamic data) {
  if (data is! List) {
    throw const RegionCatalogException('unexpected catalog response shape');
  }

  final entries = <RegionCatalogEntry>[];
  for (final element in data) {
    try {
      entries.add(RegionCatalogEntry.fromJson(element as Map<String, dynamic>));
    } catch (_) {
      // Skip this malformed element; do not abort the whole parse.
    }
  }
  return entries;
}

/// GETs `/regions` (resolves to the SvelteKit-proxied `/api/v1/regions`) via
/// the cookie-authenticated [api] client and parses the response.
///
/// Wraps any [DioException] or parse failure in [RegionCatalogException]
/// (preserving the original error as [RegionCatalogException.cause]) rather
/// than swallowing it -- because this fetch fully completes (or fails)
/// before any store write happens, a failure here always leaves every
/// persisted [RegionEntity] row untouched (D-03).
Future<List<RegionCatalogEntry>> fetchRegionCatalog(Dio api) async {
  try {
    final response = await api.get('/regions');
    return parseRegionCatalog(response.data);
  } on RegionCatalogException {
    rethrow;
  } catch (e) {
    throw RegionCatalogException('failed to fetch region catalog', e);
  }
}

/// The persisted [existing] region ids that are absent from [fetchedIds] --
/// i.e. regions dropped from the latest catalog fetch (D-08).
Set<String> orphanedRegionIds(
  Iterable<String> fetchedIds,
  Iterable<RegionEntity> existing,
) {
  final fetchedIdSet = fetchedIds.toSet();
  return existing
      .map((e) => e.id)
      .where((id) => !fetchedIdSet.contains(id))
      .toSet();
}

/// Fetches and upserts the region catalog into ObjectBox, preserving local
/// download status and `ToOne` package links across refreshes (D-01) --
/// never the destructive `removeAll()`+`putMany()` merge used by
/// `subcategory_provider.dart`/`category_provider.dart`.
class RegionRepository {
  RegionRepository(this._api, this._store);

  final Dio _api;
  final Store _store;

  /// Fetches the catalog only; throws [RegionCatalogException] on failure
  /// without touching any persisted row.
  Future<List<RegionCatalogEntry>> fetchCatalog() => fetchRegionCatalog(_api);

  /// Upserts [entries] into ObjectBox by business id inside a single write
  /// transaction: an existing row is merged in place via
  /// [RegionEntity.applyCatalogEntry] (preserving `obxId`, both `ToOne`
  /// targets, and `lastDownloadedVersion`); a new row is inserted via
  /// [RegionEntity.fromCatalogEntry]. After upserting, any persisted region
  /// whose id is absent from [entries] is flipped `inCatalog = false` --
  /// its row and any on-disk files are left untouched (D-08).
  void upsertCatalog(List<RegionCatalogEntry> entries) {
    _store.runInTransaction(TxMode.write, () {
      final box = _store.box<RegionEntity>();

      for (final entry in entries) {
        try {
          final query = box.query(RegionEntity_.id.equals(entry.id)).build();
          final existing = query.findFirst();
          query.close();

          if (existing != null) {
            existing.applyCatalogEntry(entry);
            box.put(existing);
          } else {
            box.put(RegionEntity.fromCatalogEntry(entry));
          }
        } on FormatException {
          // Malformed entry (e.g. bad bbox) -- skip it, never fatal.
        }
      }

      final fetchedIds = entries.map((e) => e.id);
      final orphans = orphanedRegionIds(fetchedIds, box.getAll());
      if (orphans.isEmpty) return;

      for (final id in orphans) {
        final query = box.query(RegionEntity_.id.equals(id)).build();
        final entity = query.findFirst();
        query.close();

        if (entity != null) {
          entity.inCatalog = false;
          box.put(entity);
        }
      }
    });
  }

  /// Fetches the catalog then upserts it. A fetch failure (thrown before any
  /// write) leaves all persisted rows untouched.
  Future<void> refreshCatalog() async {
    upsertCatalog(await fetchCatalog());
  }
}
